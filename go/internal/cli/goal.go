package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	domainGoal "github.com/wade/aia/internal/domain/goal"
	domainLearning "github.com/wade/aia/internal/domain/learning"
	goalService "github.com/wade/aia/internal/service/goal"
)

var (
	goalProjectID         string
	goalParentID          string
	goalLevel             string
	goalStatus            string
	goalPriority          int
	goalDesc              string
	goalTargetJSON        string
	goalCurrentJSON       string
	goalReason            string
	goalAgentID           string
	goalTaskTitle         string
	goalProposedName      string
	goalInvalidatedClaims []string
	goalActorType         string
)

var goalCmd = &cobra.Command{
	Use:   "goal",
	Short: "Manage long-term Strategic, Tactical, and Operational Goals (AIA-09, Plan Phase 3 §4.3)",
}

var goalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List goals globally or filtered by project/status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.GoalSvc == nil {
			return fmt.Errorf("goal service not initialized")
		}

		var goals []*domainGoal.Goal
		var err error
		if goalProjectID != "" {
			goals, err = appInst.GoalSvc.ListByProject(ctx, goalProjectID)
		} else {
			goals, err = appInst.GoalSvc.ListAll(ctx, domainGoal.GoalStatus(strings.ToUpper(goalStatus)), 50)
		}
		if err != nil {
			return err
		}
		if len(goals) == 0 {
			fmt.Println("No goals found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tLEVEL\tSTATUS\tPRIORITY\tNAME")
		for _, g := range goals {
			idShort := g.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", idShort, g.Level, g.Status, g.Priority, g.Name)
		}
		return w.Flush()
	},
}

var goalCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a long-term goal and optionally link it to a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.GoalSvc == nil {
			return fmt.Errorf("goal service not initialized")
		}
		var parentPtr *string
		if goalParentID != "" {
			parentPtr = &goalParentID
		}
		g, err := appInst.GoalSvc.Create(ctx, goalService.CreateGoalInput{
			ParentGoalID:      parentPtr,
			ProjectID:         goalProjectID,
			Name:              args[0],
			Description:       goalDesc,
			Level:             domainGoal.GoalLevel(strings.ToUpper(goalLevel)),
			TargetMetricJSON:  goalTargetJSON,
			CurrentMetricJSON: goalCurrentJSON,
			Priority:          goalPriority,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Created Goal %s (%s | %s | P%d)\n", g.ID, g.Level, g.Status, g.Priority)
		return nil
	},
}

var goalGetCmd = &cobra.Command{
	Use:   "get <goal-id>",
	Short: "Inspect a goal, its metric gap assessment, revisions, and decomposed tasks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.GoalSvc == nil {
			return fmt.Errorf("goal service not initialized")
		}
		g, err := appInst.GoalSvc.Get(ctx, args[0])
		if err != nil {
			return err
		}
		gap := goalService.EvaluateGoalGap(g)
		fmt.Printf("Goal: %s (%s)\n", g.Name, g.ID)
		fmt.Printf("  Level:          %s\n", g.Level)
		fmt.Printf("  Status:         %s\n", g.Status)
		fmt.Printf("  Priority:       %d\n", g.Priority)
		fmt.Printf("  Target Metric:  %s\n", g.TargetMetricJSON)
		fmt.Printf("  Current Metric: %s\n", g.CurrentMetricJSON)
		if gap.HasGap {
			fmt.Printf("  Gap Assessment: %.1f%% unmet (%s)\n", gap.GapRatio*100, gap.GapSummary)
		} else {
			fmt.Printf("  Gap Assessment: On target / no unmet quantitative gap\n")
		}
		if links, lErr := appInst.GoalSvc.ListDecomposedTasks(ctx, g.ID); lErr == nil && len(links) > 0 {
			fmt.Printf("  Linked Tasks:   %d\n", len(links))
			for _, lk := range links {
				fmt.Printf("    - Task %s: %s\n", lk.TaskID, lk.DecompositionRationale)
			}
		}
		return nil
	},
}

var goalStatusCmd = &cobra.Command{
	Use:   "status <goal-id> <ACTIVE|PAUSED|ACHIEVED|SUPERSEDED|ABANDONED>",
	Short: "Transition a goal status with an auditable revision record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.GoalSvc == nil {
			return fmt.Errorf("goal service not initialized")
		}
		st := domainGoal.GoalStatus(strings.ToUpper(args[1]))
		if err := appInst.GoalSvc.TransitionStatus(ctx, args[0], st, goalReason, ""); err != nil {
			return err
		}
		fmt.Printf("✓ Goal %s transitioned to %s\n", args[0], st)
		return nil
	},
}

var goalLinkProjectCmd = &cobra.Command{
	Use:   "link-project <goal-id> <project-id>",
	Short: "Link a goal to a project (M:N relationship)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.GoalSvc == nil {
			return fmt.Errorf("goal service not initialized")
		}
		if err := appInst.GoalSvc.LinkProject(ctx, args[0], args[1], domainGoal.LinkContributesTo); err != nil {
			return err
		}
		fmt.Printf("✓ Linked Goal %s -> Project %s\n", args[0], args[1])
		return nil
	},
}

var goalDecomposeCmd = &cobra.Command{
	Use:   "decompose <goal-id>",
	Short: "Decompose a goal into a concrete executable Task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.GoalSvc == nil {
			return fmt.Errorf("goal service not initialized")
		}
		title := goalTaskTitle
		if title == "" {
			title = "Goal Milestone Task"
		}
		tk, err := appInst.GoalSvc.DecomposeToTask(ctx, goalService.DecomposeTaskInput{
			GoalID:      args[0],
			ProjectID:   goalProjectID,
			AgentID:     goalAgentID,
			Title:       title,
			Description: goalDesc,
			Rationale:   goalReason,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Decomposed Goal %s into Task %s (%s)\n", args[0], tk.ID, tk.Title)
		return nil
	},
}

var goalEvolveCmd = &cobra.Command{
	Use:   "evolve <goal-id> <PERSISTENCE|OPTIMIZATION|REVISION>",
	Short: "Propose a governed goal evolution (enforces Anti-Goalpost-Moving on STRATEGIC goals)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if appInst == nil || appInst.GoalSvc == nil {
			return fmt.Errorf("goal service not initialized")
		}
		prop, err := appInst.GoalSvc.ProposeOrApplyEvolution(ctx, goalService.EvolutionInput{
			GoalID:                   args[0],
			Mode:                     domainLearning.GoalEvolutionMode(strings.ToUpper(args[1])),
			ActorType:                strings.ToUpper(goalActorType),
			ActorID:                  "cli_operator",
			ProposedName:             goalProposedName,
			ProposedDescription:      goalDesc,
			ProposedTargetMetricJSON: goalTargetJSON,
			InvalidatedClaimIDs:      goalInvalidatedClaims,
			Rationale:                goalReason,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Goal Evolution Proposal %s (%s) -> Status: %s\n", prop.ID, prop.EvolutionMode, prop.Status)
		if prop.ApprovalID != nil {
			fmt.Printf("  Approval Gate: %s\n", *prop.ApprovalID)
		}
		return nil
	},
}

func init() {
	goalListCmd.Flags().StringVarP(&goalProjectID, "project", "p", "", "Filter by project ID")
	goalListCmd.Flags().StringVarP(&goalStatus, "status", "s", "", "Filter by status (ACTIVE, PAUSED, ACHIEVED, ...)")

	goalCreateCmd.Flags().StringVarP(&goalProjectID, "project", "p", "default", "Project ID to link")
	goalCreateCmd.Flags().StringVar(&goalParentID, "parent", "", "Optional parent goal ID")
	goalCreateCmd.Flags().StringVarP(&goalLevel, "level", "l", "OPERATIONAL", "Goal level (STRATEGIC, TACTICAL, OPERATIONAL)")
	goalCreateCmd.Flags().IntVar(&goalPriority, "priority", 6, "Priority 1..10")
	goalCreateCmd.Flags().StringVarP(&goalDesc, "desc", "d", "", "Goal description")
	goalCreateCmd.Flags().StringVar(&goalTargetJSON, "target", "", "Target metrics JSON (e.g. '{\"coverage\":85}')")
	goalCreateCmd.Flags().StringVar(&goalCurrentJSON, "current", "", "Current metrics JSON (e.g. '{\"coverage\":60}')")

	goalStatusCmd.Flags().StringVarP(&goalReason, "reason", "r", "", "Reason for status revision")

	goalDecomposeCmd.Flags().StringVarP(&goalProjectID, "project", "p", "default", "Target project ID")
	goalDecomposeCmd.Flags().StringVarP(&goalAgentID, "agent", "a", "coding-agent", "Assigned agent ID")
	goalDecomposeCmd.Flags().StringVarP(&goalTaskTitle, "title", "t", "", "Task title")
	goalDecomposeCmd.Flags().StringVarP(&goalDesc, "desc", "d", "", "Task description")
	goalDecomposeCmd.Flags().StringVarP(&goalReason, "rationale", "r", "", "Decomposition rationale")

	goalEvolveCmd.Flags().StringVarP(&goalReason, "rationale", "r", "", "Evidence-backed evolution rationale")
	goalEvolveCmd.Flags().StringVar(&goalTargetJSON, "target", "", "Proposed target metric JSON")
	goalEvolveCmd.Flags().StringVar(&goalProposedName, "name", "", "Proposed revised goal name")
	goalEvolveCmd.Flags().StringVarP(&goalDesc, "desc", "d", "", "Proposed description")
	goalEvolveCmd.Flags().StringSliceVar(&goalInvalidatedClaims, "invalidated-claims", nil, "Claim IDs proving invalidated premise")
	goalEvolveCmd.Flags().StringVar(&goalActorType, "actor-type", "HUMAN", "Actor type (HUMAN or AGENT)")

	goalCmd.AddCommand(goalListCmd)
	goalCmd.AddCommand(goalCreateCmd)
	goalCmd.AddCommand(goalGetCmd)
	goalCmd.AddCommand(goalStatusCmd)
	goalCmd.AddCommand(goalLinkProjectCmd)
	goalCmd.AddCommand(goalDecomposeCmd)
	goalCmd.AddCommand(goalEvolveCmd)
}
