package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/autonomy"
	"github.com/wade/aia/internal/domain/continuity"
)

var (
	cycleProjectID string
	cycleTrigger   string
	cycleSummary   string
	cycleAsOf      string
	cycleReplay    bool
	cycleLimit     int
)

var cycleCmd = &cobra.Command{
	Use:   "cycle",
	Short: "Inspect and trigger 11-step Autonomy Closed Loops (AIA-08, AIA-18)",
}

var cycleRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Trigger a full 11-step autonomous closed-loop cycle",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.AutonomyEngine == nil {
			return fmt.Errorf("autonomy engine not initialized")
		}

		projID := cycleProjectID
		if projID == "" {
			projID = "default"
		}

		trigType := continuity.CycleTriggerType(cycleTrigger)
		if trigType == "" {
			trigType = continuity.TriggerManual
		}

		sum := cycleSummary
		if sum == "" {
			sum = "Manual autonomy cycle invocation via CLI"
		}

		var asOfPtr *time.Time
		if cycleAsOf != "" {
			parsed, pErr := time.Parse(time.RFC3339, cycleAsOf)
			if pErr != nil {
				return fmt.Errorf("invalid --as-of timestamp (expected RFC3339): %w", pErr)
			}
			u := parsed.UTC()
			asOfPtr = &u
		}

		fmt.Println("⏳ Initiating 11-step Autonomy Closed Loop...")
		outcome, err := appInst.AutonomyEngine.RunCycle(ctx, autonomy.RunCycleInput{
			ProjectID:   projID,
			TriggerType: trigType,
			TriggerRef:  "cli",
			Summary:     sum,
			AsOf:        asOfPtr,
			ReplayMode:  cycleReplay || asOfPtr != nil,
		})
		if err != nil {
			return fmt.Errorf("autonomy cycle failed: %w", err)
		}

		fmt.Println("✓ Autonomy Cycle Finished!")
		fmt.Printf("  Cycle ID:          %s (#%d)\n", outcome.CycleID, outcome.CycleNumber)
		fmt.Printf("  Decision:          %s\n", outcome.Decision.ActionType)
		fmt.Printf("  Governance Result: %s\n", outcome.GovernanceResult)
		if outcome.TaskID != nil {
			fmt.Printf("  Spawned Task:      %s\n", *outcome.TaskID)
		}
		if outcome.ApprovalID != nil {
			fmt.Printf("  Approval Gate:     %s\n", *outcome.ApprovalID)
		}
		if outcome.PostCycleSnapshotID != nil {
			fmt.Printf("  Post Snapshot:     %s\n", *outcome.PostCycleSnapshotID)
		}
		fmt.Printf("  Justification:     %s\n", outcome.Decision.Justification)
		return nil
	},
}

var cycleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent autonomy cycles",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.ContinuityRepo == nil {
			return fmt.Errorf("continuity repository not initialized")
		}
		cycles, err := appInst.ContinuityRepo.ListCycles(ctx, cycleLimit)
		if err != nil {
			return err
		}
		if len(cycles) == 0 {
			fmt.Println("No autonomy cycles recorded yet.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "CYCLE#\tID\tTRIGGER\tSTATUS\tSUMMARY")
		for _, c := range cycles {
			idShort := c.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			sum := c.Summary
			if len(sum) > 48 {
				sum = sum[:45] + "..."
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", c.CycleNumber, idShort, c.TriggerType, c.Status, sum)
		}
		return w.Flush()
	},
}

var cycleGetCmd = &cobra.Command{
	Use:   "get <cycle-id>",
	Short: "Inspect a specific autonomy cycle, its deliberation matrix, and discovered initiatives",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.ContinuityRepo == nil {
			return fmt.Errorf("continuity repository not initialized")
		}
		cyc, err := appInst.ContinuityRepo.GetCycle(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Cycle #%d (%s)\n", cyc.CycleNumber, cyc.ID)
		fmt.Printf("  Status:       %s\n", cyc.Status)
		fmt.Printf("  Trigger:      %s (%s)\n", cyc.TriggerType, cyc.TriggerRefID)
		fmt.Printf("  Snapshot ID:  %s\n", cyc.ContextSnapshotID)
		fmt.Printf("  Summary:      %s\n", cyc.Summary)

		if appInst.AutonomyRepo != nil {
			if delib, dErr := appInst.AutonomyRepo.GetDeliberationByCycle(ctx, cyc.ID); dErr == nil && delib != nil {
				fmt.Printf("  Deliberation: %s (%s)\n", delib.RecommendedAction, delib.CostBenefitAnalysis)
			}
			if inits, iErr := appInst.AutonomyRepo.ListInitiativesByCycle(ctx, cyc.ID); iErr == nil && len(inits) > 0 {
				fmt.Printf("  Initiatives:  %d\n", len(inits))
				for _, init := range inits {
					fmt.Printf("    - [%s | P%d | %s] %s\n", init.Status, init.PriorityScore, init.DiscoveryType, init.DetectedOpportunity)
				}
			}
		}
		return nil
	},
}

var cycleInitiativesCmd = &cobra.Command{
	Use:   "initiatives <cycle-id>",
	Short: "List candidate initiatives discovered during an autonomy cycle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.AutonomyRepo == nil {
			return fmt.Errorf("autonomy repository not initialized")
		}

		initiatives, err := appInst.AutonomyRepo.ListInitiativesByCycle(ctx, args[0])
		if err != nil {
			return err
		}

		if len(initiatives) == 0 {
			fmt.Printf("No initiatives recorded for cycle %s.\n", args[0])
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tCONFIDENCE\tPRIORITY\tSTATUS\tOPPORTUNITY")
		for _, init := range initiatives {
			idShort := init.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			oppShort := init.DetectedOpportunity
			if len(oppShort) > 40 {
				oppShort = oppShort[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", idShort, init.DiscoveryType, init.ConfidenceLevel, init.PriorityScore, init.Status, oppShort)
		}
		return w.Flush()
	},
}

func init() {
	cycleRunCmd.Flags().StringVarP(&cycleProjectID, "project", "p", "default", "Target project ID")
	cycleRunCmd.Flags().StringVarP(&cycleTrigger, "trigger", "t", "MANUAL", "Cycle trigger type (SCHEDULED, MANUAL, EVENT_TRIGGERED)")
	cycleRunCmd.Flags().StringVarP(&cycleSummary, "summary", "s", "", "Cycle intent or run summary")
	cycleRunCmd.Flags().StringVar(&cycleAsOf, "as-of", "", "Point-in-Time historical timestamp (RFC3339) for zero-look-ahead replay")
	cycleRunCmd.Flags().BoolVar(&cycleReplay, "replay", false, "Run in Point-in-Time replay mode (blocks live HTTP fetches)")
	cycleListCmd.Flags().IntVarP(&cycleLimit, "limit", "n", 20, "Maximum cycles to list")

	cycleCmd.AddCommand(cycleRunCmd)
	cycleCmd.AddCommand(cycleListCmd)
	cycleCmd.AddCommand(cycleGetCmd)
	cycleCmd.AddCommand(cycleInitiativesCmd)
}
