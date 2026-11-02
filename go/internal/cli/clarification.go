package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	clarificationAnswerStr string
)

var clarificationCmd = &cobra.Command{
	Use:     "clarification",
	Aliases: []string{"clarify"},
	Short:   "Manage requirements clarification requests and human answers (AIA-21 §5, §11.1)",
}

var clarificationListCmd = &cobra.Command{
	Use:   "list [project-id]",
	Short: "List clarification requests for a project",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if appInst.ClarificationSvc == nil {
			return fmt.Errorf("ClarificationSvc is not initialized")
		}

		var targetProj string
		if len(args) > 0 {
			targetProj = args[0]
		}
		if targetProj == "" {
			projects, err := appInst.ProjectSvc.List(ctx)
			if err == nil && len(projects) == 1 {
				targetProj = string(projects[0].ID)
				fmt.Printf("ℹ️  Using project '%s'\n", targetProj)
			} else {
				return fmt.Errorf("project_id is required; specify via 'aia clarification list <project_id>'")
			}
		}

		requests, err := appInst.ClarificationSvc.ListByProject(ctx, targetProj)
		if err != nil {
			return fmt.Errorf("failed to list clarifications: %w", err)
		}
		if len(requests) == 0 {
			fmt.Printf("No clarification requests found for project '%s'.\n", targetProj)
			return nil
		}

		fmt.Printf("=== Clarification Requests for '%s' (%d total) ===\n", targetProj, len(requests))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tPRIORITY\tBLOCKING\tSTATUS\tFIELD\tQUESTION")
		for _, cr := range requests {
			blockingStr := "No"
			if cr.Blocking {
				blockingStr = "YES"
			}
			qShort := cr.Question
			if len(qShort) > 40 {
				qShort = qShort[:40] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				cr.ID, cr.PriorityString(), blockingStr, cr.Status, cr.FieldKey, qShort)
		}
		_ = w.Flush()
		return nil
	},
}

var clarificationShowCmd = &cobra.Command{
	Use:   "show <request-id>",
	Short: "Display detailed clarification question, decision reason, and options",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		reqID := args[0]

		if appInst.ClarificationSvc == nil {
			return fmt.Errorf("ClarificationSvc is not initialized")
		}

		cr, err := appInst.ClarificationSvc.Get(ctx, reqID)
		if err != nil {
			return fmt.Errorf("failed to fetch clarification: %w", err)
		}
		if cr == nil {
			return fmt.Errorf("clarification request %s not found", reqID)
		}

		fmt.Printf("=== Clarification Request: %s ===\n", cr.ID)
		fmt.Printf("所屬專案:       %s\n", cr.ProjectID)
		fmt.Printf("對應欄位:       %s\n", cr.FieldKey)
		fmt.Printf("優先級:         %s\n", cr.PriorityString())
		fmt.Printf("是否阻擋:       %v\n", cr.Blocking)
		fmt.Printf("狀態:           %s\n", cr.Status)
		fmt.Printf("\n--- 問題內容 ---\n%s\n", cr.Question)
		fmt.Printf("\n--- 提問原因 (決策價值) ---\n%s\n", cr.Reason)
		fmt.Printf("\n--- 後續影響 (Downstream Impact) ---\n%s\n", cr.Impact)

		if len(cr.Options) > 0 {
			fmt.Printf("\n--- 建議選項與方案 ---\n")
			for i, opt := range cr.Options {
				fmt.Printf("  [%d] %s\n", i+1, opt)
			}
		}

		if cr.Answer != "" {
			fmt.Printf("\n--- 使用者已確認答案 ---\n%s\n", cr.Answer)
			if cr.AnsweredAt != nil {
				fmt.Printf("回答時間:       %s\n", cr.AnsweredAt.Format("2006-01-02 15:04:05"))
			}
		} else {
			fmt.Println("\n👉 請使用 `aia clarification answer <id> --answer \"...\"` 回覆此問題。")
		}
		return nil
	},
}

var clarificationAnswerCmd = &cobra.Command{
	Use:   "answer <request-id>",
	Short: "Provide a formal decision/answer to a clarification inquiry (AIA-21 §7.3)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		reqID := args[0]

		if clarificationAnswerStr == "" {
			return fmt.Errorf("--answer is required")
		}

		if appInst.ClarificationSvc == nil {
			return fmt.Errorf("ClarificationSvc is not initialized")
		}

		cr, err := appInst.ClarificationSvc.Answer(ctx, reqID, clarificationAnswerStr)
		if err != nil {
			return fmt.Errorf("failed to submit answer: %w", err)
		}

		fmt.Printf("✓ Answer recorded for clarification %s.\n", cr.ID)
		fmt.Printf("對應欄位:   %s\n", cr.FieldKey)
		fmt.Printf("採納內容:   %s\n", cr.Answer)
		fmt.Printf("專案狀態:   Project Brief 已自動更新版本與歷程 (brief.yaml)\n")
		return nil
	},
}

var clarificationDeferCmd = &cobra.Command{
	Use:   "defer <request-id>",
	Short: "Defer answering a non-blocking clarification question (AIA-21 §5.2)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		reqID := args[0]

		if appInst.ClarificationSvc == nil {
			return fmt.Errorf("ClarificationSvc is not initialized")
		}

		if err := appInst.ClarificationSvc.Defer(ctx, reqID); err != nil {
			return fmt.Errorf("failed to defer clarification: %w", err)
		}

		fmt.Printf("✓ Clarification %s marked as DEFERRED.\n", reqID)
		return nil
	},
}

func init() {
	clarificationAnswerCmd.Flags().StringVarP(&clarificationAnswerStr, "answer", "a", "", "user's answer to the clarification question")

	clarificationCmd.AddCommand(clarificationListCmd)
	clarificationCmd.AddCommand(clarificationShowCmd)
	clarificationCmd.AddCommand(clarificationAnswerCmd)
	clarificationCmd.AddCommand(clarificationDeferCmd)

	RootCmd.AddCommand(clarificationCmd)
}
