package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	decisionProjectID string
)

var decisionCmd = &cobra.Command{
	Use:   "decision",
	Short: "Inspect decisions, reasoning, and provenance evidence",
}

var decisionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recorded decisions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.DecisionRepo == nil {
			return fmt.Errorf("decision repository not available")
		}

		projID := decisionProjectID
		if projID == "" {
			// If not specified, list default project or recent decisions
			projID = "project-default"
		}

		decisions, err := appInst.DecisionRepo.ListByProject(ctx, projID, 20)
		if err != nil {
			return err
		}

		if len(decisions) == 0 {
			fmt.Printf("No decisions recorded for project %s.\n", projID)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tACTION\tGOVERNANCE\tINTENT\tCREATED")
		for _, d := range decisions {
			idShort := d.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			intent := d.IntentSummary
			if len(intent) > 40 {
				intent = intent[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				idShort, d.ActionType, d.GovernanceResult, intent, d.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		_ = w.Flush()
		return nil
	},
}

var decisionTraceCmd = &cobra.Command{
	Use:   "trace <decision-id>",
	Short: "Trace full decision provenance, context snapshot, and evidence chain (Milestone M2)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.KnowledgeSvc == nil {
			return fmt.Errorf("knowledge service not available")
		}

		decisionID := args[0]
		trace, err := appInst.KnowledgeSvc.TraceDecisionProvenance(ctx, decisionID)
		if err != nil {
			return fmt.Errorf("failed to trace decision provenance: %w", err)
		}

		d := trace.Decision
		fmt.Println("=== Decision Provenance Trace ===")
		fmt.Printf("Decision ID:       %s\n", d.ID)
		fmt.Printf("Action Type:       %s\n", d.ActionType)
		fmt.Printf("Governance:        %s\n", d.GovernanceResult)
		fmt.Printf("Intent Summary:    %s\n", d.IntentSummary)
		fmt.Printf("Justification:     %s\n", d.Justification)
		if d.CycleID != nil {
			fmt.Printf("Autonomy Cycle:    %s\n", *d.CycleID)
		}
		if d.ContextSnapshotID != nil {
			fmt.Printf("Context Snapshot:  %s\n", *d.ContextSnapshotID)
		}
		fmt.Printf("Timestamp:         %s\n", d.CreatedAt.Format("2006-01-02 15:04:05 UTC"))

		if trace.ContextSnapshot != nil {
			fmt.Println("\n--- Context Snapshot ---")
			fmt.Printf("Scope:             %s\n", trace.ContextSnapshot.Scope)
			fmt.Printf("Snapshot Type:     %s\n", trace.ContextSnapshot.SnapshotType)
			fmt.Printf("Pending Issues:    %s\n", trace.ContextSnapshot.PendingIssuesJSON)
		}

		fmt.Printf("\n--- Grounded Evidence Chain (%d linked) ---\n", len(trace.EvidenceChain))
		if len(trace.EvidenceChain) == 0 {
			fmt.Println("No external evidence linked (autonomous deliberative rationale or NO_ACTION).")
		} else {
			for i, e := range trace.EvidenceChain {
				fmt.Printf("[%d] Evidence ID: %s (Method: %s, Hash: %s)\n", i+1, e.Evidence.ID, e.Evidence.CaptureMethod, e.Evidence.ContentHash)
				fmt.Printf("    Snippet:     %s\n", e.Evidence.ContentSnippet)
				if e.Source != nil {
					fmt.Printf("    Source:      %s [%s] (Publisher: %s)\n", e.Source.URI, e.Source.SourceType, e.Source.Publisher)
					fmt.Printf("    5D Scores:   Authority=%.2f, Directness=%.2f, Timeliness=%.2f, Independence=%.2f, Repro=%.2f\n",
						e.Source.AuthorityScore, e.Source.DirectnessScore, e.Source.TimelinessScore, e.Source.IndependenceScore, e.Source.ReproducibilityScore)
				}
			}
		}

		return nil
	},
}

func init() {
	decisionListCmd.Flags().StringVarP(&decisionProjectID, "project", "p", "", "project ID filter")

	decisionCmd.AddCommand(decisionListCmd)
	decisionCmd.AddCommand(decisionTraceCmd)
}
