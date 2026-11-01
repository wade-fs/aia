package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	learningLevelFilter   string
	learningVerdictFilter string
	learningLimit         int
)

var learningCmd = &cobra.Command{
	Use:   "learning",
	Short: "Inspect verifiable self-correction & learning records (L0–L3, AIA-11 §21)",
}

var learningListCmd = &cobra.Command{
	Use:   "list",
	Short: "List learning records with maturity level (L0–L3) and effectiveness verdict",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.LearningRepo == nil {
			return fmt.Errorf("learning repository not initialized")
		}

		records, err := appInst.LearningRepo.QueryLearningRecords(ctx, repository.LearningFilter{
			Level:   domainLearning.LearningLevel(strings.ToUpper(strings.TrimSpace(learningLevelFilter))),
			Verdict: domainLearning.EffectivenessVerdict(strings.ToUpper(strings.TrimSpace(learningVerdictFilter))),
			Limit:   learningLimit,
		})
		if err != nil {
			return err
		}
		if len(records) == 0 {
			fmt.Println("No learning records found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tLEVEL\tVERDICT\tCONF\tTITLE")
		for _, r := range records {
			idShort := r.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			title := r.Title
			if len(title) > 48 {
				title = title[:45] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%s\n",
				idShort, r.LearningLevel, r.EffectivenessVerdict, r.EpistemicConfidence, title)
		}
		return w.Flush()
	},
}

var learningGetCmd = &cobra.Command{
	Use:   "get <learning-id>",
	Short: "Inspect a learning record, its 7 epistemic comparison fields, RCA, and correction action",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.LearningRepo == nil {
			return fmt.Errorf("learning repository not initialized")
		}
		rec, err := appInst.LearningRepo.GetLearningRecord(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Learning Record: %s (%s)\n", rec.Title, rec.ID)
		fmt.Printf("  Maturity Level:        %s\n", rec.LearningLevel)
		fmt.Printf("  Effectiveness Verdict: %s\n", rec.EffectivenessVerdict)
		fmt.Printf("  Epistemic Confidence:  %.2f\n", rec.EpistemicConfidence)
		if rec.SourceCycleID != nil {
			fmt.Printf("  Source Cycle ID:       %s\n", *rec.SourceCycleID)
		}
		fmt.Printf("  Generalized Cycles:    %s\n", rec.GeneralizedCycleIDsJSON)
		fmt.Printf("  1. Baseline:           %s\n", rec.BaselineJSON)
		fmt.Printf("  2. Hypothesis:         %s\n", rec.HypothesisJSON)
		fmt.Printf("  3. Intervention:       %s\n", rec.InterventionJSON)
		fmt.Printf("  4. Verification:       %s\n", rec.VerificationJSON)
		fmt.Printf("  5. Regression:         %s\n", rec.RegressionJSON)
		fmt.Printf("  6. Follow-up:          %s\n", rec.FollowupJSON)

		if rec.RCAID != nil && *rec.RCAID != "" {
			if rca, rErr := appInst.LearningRepo.GetRCA(ctx, *rec.RCAID); rErr == nil && rca != nil {
				fmt.Printf("  Linked RCA (%s): [%s | %s] %s (Falsifiable: %s)\n",
					rca.ID, rca.RootCauseCategory, rca.RCAValidityStatus, rca.AnalysisNarrative, rca.FalsifiableHypothesis)
			}
		}
		if rec.CorrectionActionID != nil && *rec.CorrectionActionID != "" {
			if ca, cErr := appInst.LearningRepo.GetCorrectionAction(ctx, *rec.CorrectionActionID); cErr == nil && ca != nil {
				fmt.Printf("  Linked Correction (%s): [%s | %s | %s | Regression=%s | Verification=%s] target=%s\n",
					ca.ID, ca.TierProfile, ca.ScopeLevel, ca.CorrectionType, ca.RegressionStatus, ca.VerificationStatus, ca.TargetEntityRef)
			}
		}
		return nil
	},
}

func init() {
	learningListCmd.Flags().StringVarP(&learningLevelFilter, "level", "l", "", "Filter by level (L0_RECORDED, L1_APPLIED, L2_VERIFIED, L3_GENERALIZED)")
	learningListCmd.Flags().StringVarP(&learningVerdictFilter, "verdict", "v", "", "Filter by verdict (EFFECTIVE, INEFFECTIVE, REGRESSED, INCONCLUSIVE)")
	learningListCmd.Flags().IntVarP(&learningLimit, "limit", "n", 25, "Maximum records to list")

	learningCmd.AddCommand(learningListCmd)
	learningCmd.AddCommand(learningGetCmd)
}
