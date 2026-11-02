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

var problemPlanCmd = &cobra.Command{
	Use:   "plan <problem-id>",
	Short: "Show latest plan and work items for a problem (AIA-22 §4.7)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		probID := args[0]
		p, err := appInst.ProblemSvc.GetLatestPlan(ctx, probID)
		if err != nil {
			return fmt.Errorf("no plan found for problem %s: %w", probID, err)
		}

		fmt.Printf("=== Plan: %s (v%d) ===\n", p.ID, p.Version)
		fmt.Printf("Status:               %s\n", p.Status)
		fmt.Printf("Objective:            %s\n", p.Objective)
		if p.Strategy != "" {
			fmt.Printf("Strategy:             %s\n", p.Strategy)
		}
		if p.SupersedesPlanID != "" {
			fmt.Printf("Supersedes:           %s\n", p.SupersedesPlanID)
		}

		if len(p.WorkItems) > 0 {
			fmt.Printf("\n--- Work Items (%d total) ---\n", len(p.WorkItems))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tSTAGE\tSTATUS\tTASK_ID\tTITLE")
			for _, wi := range p.WorkItems {
				taskIDStr := "-"
				if wi.LinkedTaskID != "" {
					taskIDStr = wi.LinkedTaskID
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", wi.ID, wi.Stage, wi.Status, taskIDStr, wi.Title)
			}
			_ = w.Flush()
		}

		changes, _ := appInst.ProblemSvc.ListPlanChanges(ctx, p.ID)
		if len(changes) > 0 {
			fmt.Printf("\n--- Plan Change History (%d changes) ---\n", len(changes))
			for _, c := range changes {
				fmt.Printf("• v%d -> v%d: %s\n", c.FromVersion, c.ToVersion, c.Reason)
			}
		}
		return nil
	},
}

var problemPlanApproveCmd = &cobra.Command{
	Use:   "plan-approve <plan-id>",
	Short: "Formally approve a proposed plan (AIA-22 §9.2)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		planID := args[0]
		p, err := appInst.ProblemSvc.ApprovePlan(ctx, planID, "cli-operator")
		if err != nil {
			return fmt.Errorf("failed to approve plan: %w", err)
		}
		fmt.Printf("✓ Plan %s (v%d) approved by cli-operator. Ready for execution.\n", p.ID, p.Version)
		return nil
	},
}

var problemMaterializeCmd = &cobra.Command{
	Use:   "materialize <plan-id>",
	Short: "Materialize plan work items into executable AIA Tasks (AIA-22 §4.7)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		planID := args[0]
		tasks, err := appInst.ProblemSvc.MaterializeAllWorkItems(ctx, planID)
		if err != nil {
			return fmt.Errorf("failed to materialize work items: %w", err)
		}

		fmt.Printf("✓ Materialized %d work items from Plan %s into Tasks:\n", len(tasks), planID)
		for _, t := range tasks {
			fmt.Printf("  - Task %s: %s (Status: %s)\n", t.ID, t.Title, t.Status)
		}
		return nil
	},
}

var (
	reassessTrigger    string
	reassessCause      string
	reassessEvidence   string
	reassessRationale  string
	reassessAction     string
	reassessFindings   []string
	reassessContradict []string
)

var problemTreeCmd = &cobra.Command{
	Use:   "tree <problem-id>",
	Short: "Display recursive problem decomposition hierarchy (AIA-22 §5.1)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		tree, err := appInst.ProblemSvc.GetProblemTree(ctx, args[0])
		if err != nil {
			return fmt.Errorf("failed to get problem tree: %w", err)
		}

		fmt.Printf("=== Problem Decomposition Tree: %s ===\n", tree.Problem.Title)
		printProblemTreeNode(tree, "", true)
		return nil
	},
}

func printProblemTreeNode(node *domainProblem.ProblemTreeNode, indent string, isLast bool) {
	marker := "├── "
	if isLast {
		marker = "└── "
	}
	if indent == "" {
		marker = "• "
	}
	fmt.Printf("%s%s[%s] %s (%s, Depth: %d, Model: v%d, Plan: v%d)\n",
		indent, marker, node.Problem.ID, node.Problem.Title, node.Problem.Status,
		node.Depth, node.Problem.CurrentModelVersion, node.Problem.CurrentPlanVersion)

	childIndent := indent
	if indent != "" {
		if isLast {
			childIndent += "    "
		} else {
			childIndent += "│   "
		}
	} else {
		childIndent = "  "
	}

	for i, ch := range node.Children {
		printProblemTreeNode(ch, childIndent, i == len(node.Children)-1)
	}
}

var problemDecomposeCmd = &cobra.Command{
	Use:   "decompose <parent-id>",
	Short: "Decompose a problem into child sub-problems with depth/budget bounds (AIA-22 §5.1, §5.2)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}
		if problemFile == "" {
			return fmt.Errorf("--file <subproblems.json> is required")
		}

		bytes, err := os.ReadFile(problemFile)
		if err != nil {
			return fmt.Errorf("failed to read subproblems file: %w", err)
		}

		var subProblems []problemService.CreateProblemInput
		if err := json.Unmarshal(bytes, &subProblems); err != nil {
			return fmt.Errorf("invalid subproblems JSON: %w", err)
		}

		created, err := appInst.ProblemSvc.DecomposeProblem(ctx, args[0], subProblems)
		if err != nil {
			return fmt.Errorf("failed to decompose problem: %w", err)
		}

		fmt.Printf("✓ Successfully decomposed problem %s into %d sub-problem(s):\n", args[0], len(created))
		for _, sp := range created {
			fmt.Printf("  - [%s] %s (Priority: %s)\n", sp.ID, sp.Title, sp.Priority)
		}
		return nil
	},
}

var problemReassessCmd = &cobra.Command{
	Use:   "reassess <problem-id>",
	Short: "Trigger dynamic plan re-evaluation under new evidence or failure (AIA-22 §8)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst.ProblemSvc == nil {
			return fmt.Errorf("ProblemSvc is not initialized")
		}

		if reassessTrigger == "" {
			return fmt.Errorf("--trigger is required (e.g. HYPOTHESIS_CONTRADICTED, RULE_CONFLICT, TASK_FAILURE)")
		}
		if reassessCause == "" {
			return fmt.Errorf("--cause is required (e.g. MODEL_ERROR, ASSUMPTION_INVALID, PLAN_UNREASONABLE)")
		}
		if reassessEvidence == "" {
			return fmt.Errorf("--evidence is required")
		}
		if reassessRationale == "" {
			return fmt.Errorf("--reason is required")
		}

		req := domainProblem.ReassessmentRequest{
			ProblemID:              args[0],
			Trigger:                domainProblem.ReassessmentTriggerType(reassessTrigger),
			TriggerEvidence:        reassessEvidence,
			RootCause:              domainProblem.RootCauseCategory(reassessCause),
			NewFindings:            reassessFindings,
			ContradictedHypotheses: reassessContradict,
			SuggestedAction:        domainProblem.ReassessmentAction(reassessAction),
			Rationale:              reassessRationale,
			AgentID:                "cli-operator",
		}

		res, err := appInst.ProblemSvc.TriggerReassessment(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to trigger reassessment: %w", err)
		}

		fmt.Printf("✓ Triggered Reassessment for Problem %s:\n", res.ProblemID)
		fmt.Printf("  - Action Taken:        %s\n", res.ActionTaken)
		fmt.Printf("  - Model Advanced:      v%d -> v%d\n", res.PreviousModelVer, res.NewModelVer)
		fmt.Printf("  - Contradicted Hypo:   %d marked CONTRADICTED\n", res.ContradictedCount)
		fmt.Printf("  - Requires New Plan:   %t\n", res.RequiresNewPlan)
		fmt.Printf("  - Analysis Record:     %s\n", res.AnalysisRecordID)
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
	problemDecomposeCmd.Flags().StringVarP(&problemFile, "file", "f", "", "Path to SubProblems JSON file")

	problemReassessCmd.Flags().StringVar(&reassessTrigger, "trigger", "", "Trigger type (HYPOTHESIS_CONTRADICTED, RULE_CONFLICT, TASK_FAILURE, etc.)")
	problemReassessCmd.Flags().StringVar(&reassessCause, "cause", "", "Root cause (MODEL_ERROR, ASSUMPTION_INVALID, PLAN_UNREASONABLE, etc.)")
	problemReassessCmd.Flags().StringVar(&reassessEvidence, "evidence", "", "Factual evidence leading to reassessment")
	problemReassessCmd.Flags().StringVar(&reassessRationale, "reason", "", "Explanation and reasoning for plan adaptation")
	problemReassessCmd.Flags().StringVar(&reassessAction, "action", "MODIFY_PLAN", "Action (MODIFY_PLAN, REBUILD_PLAN, REQUEST_CLARIFICATION, PAUSE)")
	problemReassessCmd.Flags().StringSliceVar(&reassessFindings, "findings", nil, "New findings discovered")
	problemReassessCmd.Flags().StringSliceVar(&reassessContradict, "contradict", nil, "Assumption IDs contradicted by new evidence")

	problemCmd.AddCommand(problemCreateCmd)
	problemCmd.AddCommand(problemListCmd)
	problemCmd.AddCommand(problemShowCmd)
	problemCmd.AddCommand(problemPromptCmd)
	problemCmd.AddCommand(problemApplyAnalysisCmd)
	problemCmd.AddCommand(problemPlanCmd)
	problemCmd.AddCommand(problemPlanApproveCmd)
	problemCmd.AddCommand(problemMaterializeCmd)
	problemCmd.AddCommand(problemTreeCmd)
	problemCmd.AddCommand(problemDecomposeCmd)
	problemCmd.AddCommand(problemReassessCmd)

	RootCmd.AddCommand(problemCmd)
}
