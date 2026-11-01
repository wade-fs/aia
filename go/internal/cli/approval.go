package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/domain/governance"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	approvalStatusFilter string
	approvalLimit        int
	approvalComment      string
)

var approvalCmd = &cobra.Command{
	Use:   "approval",
	Short: "Inspect and manage operational governance approvals (AIA-17)",
}

var approvalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List approvals (default PENDING)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.ApprovalSvc == nil {
			return fmt.Errorf("approval service not available")
		}

		var approvals []*governance.Approval
		var err error
		if approvalStatusFilter == "" || strings.EqualFold(approvalStatusFilter, "PENDING") {
			approvals, err = appInst.ApprovalSvc.ListPending(ctx)
		} else {
			approvals, err = appInst.ApprovalSvc.List(ctx, repository.ApprovalFilter{
				Status: governance.ApprovalStatus(strings.ToUpper(approvalStatusFilter)),
				Limit:  approvalLimit,
			})
		}

		if err != nil {
			return fmt.Errorf("failed to query approvals: %w", err)
		}

		if len(approvals) == 0 {
			fmt.Println("No approvals found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tOPERATION\tRISK\tSTATUS\tREQUESTER\tTASK/DECISION\tCREATED")
		for _, a := range approvals {
			ref := "-"
			if a.TaskID != nil && *a.TaskID != "" {
				ref = "task:" + *a.TaskID
			} else if a.DecisionID != nil && *a.DecisionID != "" {
				ref = "decision:" + *a.DecisionID
			}

			idShort := a.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			reqShort := a.RequesterEntity
			if len(reqShort) > 16 {
				reqShort = reqShort[:16]
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				idShort, a.OperationType, a.RiskTier, a.Status, reqShort, ref, a.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		return w.Flush()
	},
}

var approvalGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get approval details by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.ApprovalSvc == nil {
			return fmt.Errorf("approval service not available")
		}

		appr, err := appInst.ApprovalSvc.Get(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Approval ID:     %s\n", appr.ID)
		fmt.Printf("Status:          %s\n", appr.Status)
		fmt.Printf("Operation:       %s\n", appr.OperationType)
		fmt.Printf("Risk Tier:       %s\n", appr.RiskTier)
		fmt.Printf("Requester:       %s\n", appr.RequesterEntity)
		if appr.TaskID != nil {
			fmt.Printf("Task ID:         %s\n", *appr.TaskID)
		}
		if appr.DecisionID != nil {
			fmt.Printf("Decision ID:     %s\n", *appr.DecisionID)
		}
		if appr.TimeoutAt != nil {
			fmt.Printf("Timeout At:      %s (Policy: %s)\n", appr.TimeoutAt.Format("2006-01-02 15:04:05"), appr.TimeoutPolicy)
		}
		if appr.ReviewerID != nil {
			fmt.Printf("Reviewer:        %s\n", *appr.ReviewerID)
		}
		if appr.ReviewerNote != nil {
			fmt.Printf("Review Note:     %s\n", *appr.ReviewerNote)
		}
		fmt.Printf("Created At:      %s\n", appr.CreatedAt.Format("2006-01-02 15:04:05"))
		if appr.OperationDetailJSON != "" {
			fmt.Printf("\nOperation Details:\n%s\n", appr.OperationDetailJSON)
		}
		return nil
	},
}

var approvalApproveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve an operational governance gate (Human Owner only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.ApprovalSvc == nil {
			return fmt.Errorf("approval service not available")
		}

		// Explicit Human Owner actor
		actorID := "cli_operator"
		actorType := "HUMAN"

		if err := appInst.ApprovalSvc.Approve(ctx, args[0], actorID, approvalComment, actorType); err != nil {
			return fmt.Errorf("approval failed: %w", err)
		}

		fmt.Printf("✓ Approval %s APPROVED.\n", args[0])
		return nil
	},
}

var approvalRejectCmd = &cobra.Command{
	Use:   "reject <id>",
	Short: "Reject an operational governance gate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.ApprovalSvc == nil {
			return fmt.Errorf("approval service not available")
		}

		actorID := "cli_operator"
		actorType := "HUMAN"

		if err := appInst.ApprovalSvc.Reject(ctx, args[0], actorID, approvalComment, actorType); err != nil {
			return fmt.Errorf("reject failed: %w", err)
		}

		fmt.Printf("✓ Approval %s REJECTED.\n", args[0])
		return nil
	},
}

func init() {
	approvalListCmd.Flags().StringVarP(&approvalStatusFilter, "status", "s", "PENDING", "Filter by status (PENDING, APPROVED, REJECTED, EXPIRED, CANCELLED)")
	approvalListCmd.Flags().IntVarP(&approvalLimit, "limit", "l", 50, "Limit number of records returned")

	approvalApproveCmd.Flags().StringVarP(&approvalComment, "comment", "m", "", "Reviewer comment or rationale")
	approvalRejectCmd.Flags().StringVarP(&approvalComment, "comment", "m", "", "Reviewer rejection note")

	approvalCmd.AddCommand(approvalListCmd)
	approvalCmd.AddCommand(approvalGetCmd)
	approvalCmd.AddCommand(approvalApproveCmd)
	approvalCmd.AddCommand(approvalRejectCmd)
}
