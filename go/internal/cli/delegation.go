package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/service/delegation"
)

var delegationCmd = &cobra.Command{
	Use:   "delegation",
	Short: "Manage cross-domain agent contracts and work delegations (AIA-12 §21, Phase 5)",
}

var delegationRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request work delegation from one domain agent to another",
	RunE: func(cmd *cobra.Command, args []string) error {
		requester, _ := cmd.Flags().GetString("requester")
		target, _ := cmd.Flags().GetString("target")
		title, _ := cmd.Flags().GetString("title")
		specStr, _ := cmd.Flags().GetString("spec")
		gateStr, _ := cmd.Flags().GetString("gate")
		projectID, _ := cmd.Flags().GetString("project")
		goalID, _ := cmd.Flags().GetString("goal")

		if requester == "" || target == "" {
			return fmt.Errorf("--requester and --target are required")
		}

		taskSpec := map[string]any{"title": title}
		if specStr != "" {
			if err := json.Unmarshal([]byte(specStr), &taskSpec); err != nil {
				return fmt.Errorf("invalid --spec JSON: %w", err)
			}
		}
		if title != "" {
			taskSpec["title"] = title
		}

		gate := map[string]any{"type": "STANDARD_DELIVERABLE_CHECK"}
		if gateStr != "" {
			if err := json.Unmarshal([]byte(gateStr), &gate); err != nil {
				return fmt.Errorf("invalid --gate JSON: %w", err)
			}
		}

		var pID, gID *string
		if projectID != "" {
			pID = &projectID
		}
		if goalID != "" {
			gID = &goalID
		}

		del, err := appInst.DelegationSvc.RequestDelegation(context.Background(), delegation.CreateDelegationInput{
			RequesterAgentID:   requester,
			TargetAgentID:      target,
			SourceProjectID:    pID,
			GoalID:             gID,
			TaskSpec:           taskSpec,
			AcceptanceCriteria: gate,
		})
		if err != nil {
			return err
		}

		out, _ := json.MarshalIndent(del, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var delegationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cross-domain delegation requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		requester, _ := cmd.Flags().GetString("requester")
		target, _ := cmd.Flags().GetString("target")
		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")

		list, err := appInst.DelegationSvc.List(context.Background(), requester, target, status, limit)
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var delegationShowCmd = &cobra.Command{
	Use:   "show <delegation-id>",
	Short: "Show details of a delegation request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		del, err := appInst.DelegationSvc.Get(context.Background(), args[0])
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(del, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var delegationAcceptCmd = &cobra.Command{
	Use:   "accept <delegation-id>",
	Short: "Accept a delegation request and transition to IN_PROGRESS",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := appInst.DelegationSvc.AcceptDelegation(context.Background(), args[0]); err != nil {
			return err
		}
		fmt.Printf("Delegation %s accepted and in progress\n", args[0])
		return nil
	},
}

var delegationDeliverCmd = &cobra.Command{
	Use:   "deliver <delegation-id>",
	Short: "Submit deliverable artifact or tool for a delegation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		delStr, _ := cmd.Flags().GetString("json")
		delMap := map[string]any{"submitted": true}
		if delStr != "" {
			if err := json.Unmarshal([]byte(delStr), &delMap); err != nil {
				return fmt.Errorf("invalid --json deliverable: %w", err)
			}
		}
		if err := appInst.DelegationSvc.DeliverWork(context.Background(), args[0], delMap); err != nil {
			return err
		}
		fmt.Printf("Delegation %s marked as DELIVERED\n", args[0])
		return nil
	},
}

var delegationVerifyCmd = &cobra.Command{
	Use:   "verify <delegation-id>",
	Short: "Verify delivery against acceptance criteria",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		passed, _ := cmd.Flags().GetBool("passed")
		comment, _ := cmd.Flags().GetString("comment")

		if err := appInst.DelegationSvc.VerifyDelivery(context.Background(), args[0], passed, comment); err != nil {
			return err
		}
		verdict := "VERIFIED"
		if !passed {
			verdict = "FAILED"
		}
		fmt.Printf("Delegation %s verification concluded: %s (comment: %q)\n", args[0], verdict, comment)
		return nil
	},
}

func init() {
	delegationRequestCmd.Flags().String("requester", "", "Requester agent ID (e.g. writing)")
	delegationRequestCmd.Flags().String("target", "", "Target agent ID (e.g. coding)")
	delegationRequestCmd.Flags().String("title", "", "Short task title")
	delegationRequestCmd.Flags().String("spec", "", "Task specification JSON")
	delegationRequestCmd.Flags().String("gate", "", "Acceptance criteria JSON")
	delegationRequestCmd.Flags().String("project", "", "Source project ID")
	delegationRequestCmd.Flags().String("goal", "", "Associated goal ID")

	delegationListCmd.Flags().String("requester", "", "Filter by requester agent")
	delegationListCmd.Flags().String("target", "", "Filter by target agent")
	delegationListCmd.Flags().String("status", "", "Filter by status")
	delegationListCmd.Flags().Int("limit", 50, "Max items to return")

	delegationDeliverCmd.Flags().String("json", "{}", "JSON deliverable details")

	delegationVerifyCmd.Flags().Bool("passed", true, "Whether deliverable passes acceptance criteria")
	delegationVerifyCmd.Flags().String("comment", "", "Verification feedback comment")

	delegationCmd.AddCommand(delegationRequestCmd)
	delegationCmd.AddCommand(delegationListCmd)
	delegationCmd.AddCommand(delegationShowCmd)
	delegationCmd.AddCommand(delegationAcceptCmd)
	delegationCmd.AddCommand(delegationDeliverCmd)
	delegationCmd.AddCommand(delegationVerifyCmd)
}
