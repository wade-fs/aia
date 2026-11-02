package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	domainProblem "github.com/wade/aia/internal/domain/problem"
	problemService "github.com/wade/aia/internal/service/problem"
)

var (
	problemProjectID string
	problemTitle     string
	problemObjective string
	problemScope     string
	problemPriority  string
	problemFile      string
)

var problemCmd = &cobra.Command{
	Use:     "problem",
	Aliases: []string{"prob"},
	Short:   "Complex problem modeling, epistemic analysis, and planning (AIA-22 §4, §15)",
}

var problemCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an open-ended problem for analysis",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		if problemProjectID == "" {
			return fmt.Errorf("--project is required")
		}
		if problemTitle == "" {
			return fmt.Errorf("--title is required")
		}
		if problemObjective == "" {
			problemObjective = problemTitle
		}

		p, err := appInst.ProblemSvc.CreateProblem(ctx, problemService.CreateProblemInput{
			ProjectID: problemProjectID,
			Title:     problemTitle,
			Objective: problemObjective,
			Scope:     problemScope,
			Priority:  domainProblem.ProblemPriority(problemPriority),
		})
		if err != nil {
			return fmt.Errorf("failed to create problem: %w", err)
		}

		fmt.Printf("✓ Created problem %s: %s (Priority: %s, Status: %s)\n", p.ID, p.Title, p.Priority, p.Status)
		return nil
	},
}

var problemListCmd = &cobra.Command{
	Use:   "list [project-id]",
	Short: "List problems for a project",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		var targetProj string
		if len(args) > 0 {
			targetProj = args[0]
		} else if problemProjectID != "" {
			targetProj = problemProjectID
		} else {
			projects, err := appInst.ProjectSvc.List(ctx)
			if err == nil && len(projects) == 1 {
				targetProj = string(projects[0].ID)
				fmt.Printf("ℹ️  Using project '%s'\n", targetProj)
			} else {
				return fmt.Errorf("project_id is required; specify via 'aia problem list <project_id>' or --project")
			}
		}

		problems, err := appInst.ProblemSvc.ListProblems(ctx, targetProj)
		if err != nil {
			return fmt.Errorf("failed to list problems: %w", err)
		}
		if len(problems) == 0 {
			fmt.Printf("No problems found for project '%s'.\n", targetProj)
			return nil
		}

		fmt.Printf("=== Problems for '%s' (%d total) ===\n", targetProj, len(problems))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tPRIORITY\tMODEL_VER\tPLAN_VER\tTITLE")
		for _, p := range problems {
			fmt.Fprintf(w, "%s\t%s\t%s\tv%d\tv%d\t%s\n",
				p.ID, p.Status, p.Priority, p.CurrentModelVersion, p.CurrentPlanVersion, p.Title)
		}
		return w.Flush()
	},
}

var problemShowCmd = &cobra.Command{
	Use:   "show <problem-id>",
	Short: "Show details and latest model for a problem",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		probID := args[0]
		p, err := appInst.ProblemSvc.GetProblem(ctx, probID)
		if err != nil {
			return fmt.Errorf("failed to get problem: %w", err)
		}

		fmt.Printf("=== Problem: %s ===\n", p.Title)
		fmt.Printf("ID:                   %s\n", p.ID)
		fmt.Printf("Project ID:           %s\n", p.ProjectID)
		if p.ParentID != "" {
			fmt.Printf("Parent Problem:       %s\n", p.ParentID)
		}
		fmt.Printf("Status:               %s\n", p.Status)
		fmt.Printf("Priority:             %s\n", p.Priority)
		fmt.Printf("Current Model Ver:    v%d\n", p.CurrentModelVersion)
		fmt.Printf("Current Plan Ver:     v%d\n", p.CurrentPlanVersion)
		fmt.Printf("Objective:            %s\n", p.Objective)
		if p.Scope != "" {
			fmt.Printf("Scope:                %s\n", p.Scope)
		}

		// Show latest model details if available
		if p.CurrentModelVersion > 0 {
			model, err := appInst.ProblemSvc.GetLatestModel(ctx, probID)
			if err == nil && model != nil {
				fmt.Printf("\n--- Latest Problem Model (v%d) ---\n", model.Version)
				if len(model.KnownFacts) > 0 {
					fmt.Printf("• Known Facts (%d):\n", len(model.KnownFacts))
					for _, f := range model.KnownFacts {
						fmt.Printf("  - [%.2f][%s] %s\n", f.Confidence, f.Source, f.Statement)
					}
				}
				if len(model.Assumptions) > 0 {
					fmt.Printf("• Assumptions (%d):\n", len(model.Assumptions))
					for _, a := range model.Assumptions {
						fmt.Printf("  - [%.2f][%s] %s\n", a.Confidence, a.Status, a.Statement)
					}
				}
				if len(model.Unknowns) > 0 {
					fmt.Printf("• Unknowns (%d):\n", len(model.Unknowns))
					for _, u := range model.Unknowns {
						fmt.Printf("  - %s\n", u.Statement)
					}
				}
				if len(model.Risks) > 0 {
					fmt.Printf("• Risks (%d):\n", len(model.Risks))
					for _, r := range model.Risks {
						fmt.Printf("  - %s\n", r.Statement)
					}
				}
				if len(model.SuccessCriteria) > 0 {
					fmt.Printf("• Success Criteria (%d):\n", len(model.SuccessCriteria))
					for _, sc := range model.SuccessCriteria {
						fmt.Printf("  - %s\n", sc.Statement)
					}
				}
			}
		}

		// List analysis history count
		records, _ := appInst.ProblemSvc.ListAnalysisRecords(ctx, probID)
		fmt.Printf("\nAnalysis Records:     %d records in audit history\n", len(records))

		return nil
	},
}

var problemPromptCmd = &cobra.Command{
	Use:   "prompt <problem-id>",
	Short: "Generate epistemic problem analysis prompt for an Agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		prompt, err := appInst.ProblemSvc.BuildAnalysisPrompt(ctx, args[0], "")
		if err != nil {
			return err
		}
		fmt.Println(prompt)
		return nil
	},
}

var problemApplyAnalysisCmd = &cobra.Command{
	Use:   "apply-analysis <problem-id>",
	Short: "Apply agent's structured problem analysis JSON to a problem",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}
		if problemFile == "" {
			return fmt.Errorf("--file <analysis.json> is required")
		}

		bytes, err := os.ReadFile(problemFile)
		if err != nil {
			return fmt.Errorf("failed to read analysis file: %w", err)
		}

		var analysis domainProblem.AgentProblemAnalysis
		if err := json.Unmarshal(bytes, &analysis); err != nil {
			return fmt.Errorf("invalid analysis JSON: %w", err)
		}

		model, err := appInst.ProblemSvc.ApplyAnalysis(ctx, args[0], &analysis, "cli-agent", "")
		if err != nil {
			return fmt.Errorf("failed to apply analysis: %w", err)
		}

		fmt.Printf("✓ Applied analysis to Problem %s. Advanced to Model Version v%d.\n", args[0], model.Version)
		fmt.Printf("  - Known Facts:    %d\n", len(model.KnownFacts))
		fmt.Printf("  - Assumptions:    %d\n", len(model.Assumptions))
		fmt.Printf("  - Unknowns:       %d\n", len(model.Unknowns))
		if len(analysis.ClarificationQuestions) > 0 {
			fmt.Printf("  - Questions:      %d clarification request(s) dispatched to human\n", len(analysis.ClarificationQuestions))
		}
		return nil
	},
}

func init() {
	problemCreateCmd.Flags().StringVarP(&problemProjectID, "project", "p", "", "Target project ID")
	problemCreateCmd.Flags().StringVarP(&problemTitle, "title", "t", "", "Problem title")
	problemCreateCmd.Flags().StringVarP(&problemObjective, "objective", "o", "", "Problem objective")
	problemCreateCmd.Flags().StringVarP(&problemScope, "scope", "s", "", "Problem scope")
	problemCreateCmd.Flags().StringVarP(&problemPriority, "priority", "r", "MEDIUM", "Priority (LOW, MEDIUM, HIGH, CRITICAL)")

	problemListCmd.Flags().StringVarP(&problemProjectID, "project", "p", "", "Target project ID")
	problemApplyAnalysisCmd.Flags().StringVarP(&problemFile, "file", "f", "", "Path to AgentProblemAnalysis JSON file")

	problemCmd.AddCommand(problemCreateCmd)
	problemCmd.AddCommand(problemListCmd)
	problemCmd.AddCommand(problemShowCmd)
	problemCmd.AddCommand(problemPromptCmd)
	problemCmd.AddCommand(problemApplyAnalysisCmd)

	RootCmd.AddCommand(problemCmd)
}
