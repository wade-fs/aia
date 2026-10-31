package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/domain/schedule"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage automated recurring task schedules (Cron & Misfire policies)",
	Long: `Schedule management allows recurring execution of agent tasks on standard Cron intervals.
Supports misfire policies (SKIP, RUN_ONCE, CATCH_UP).`,
}

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scheduled tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		schedules, err := appInst.ScheduleSvc.ListSchedules(ctx, false)
		if err != nil {
			return err
		}

		if len(schedules) == 0 {
			fmt.Println("No automated schedules found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCRON\tAGENT\tACTIVE\tNEXT RUN\tLAST RUN")
		for _, s := range schedules {
			idShort := s.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			lastRun := "never"
			if s.LastRunAt != nil {
				lastRun = s.LastRunAt.Format("2006-01-02 15:04")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%s\t%s\n",
				idShort, s.Name, s.CronExpr, s.AgentID, s.IsActive, s.NextRunAt.Format("2006-01-02 15:04"), lastRun)
		}
		w.Flush()
		return nil
	},
}

var (
	schedName    string
	schedCron    string
	schedAgent   string
	schedMisfire string
	schedJSON    string
)

var scheduleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new recurring schedule",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if schedName == "" || schedCron == "" {
			return fmt.Errorf("both --name and --cron flags are required")
		}
		if schedAgent == "" {
			schedAgent = "investment-agent"
		}
		pol := schedule.MisfirePolicy(schedMisfire)
		if pol == "" {
			pol = schedule.MisfireSkip
		}

		sch, err := appInst.ScheduleSvc.CreateSchedule(ctx, schedName, schedCron, schedAgent, schedJSON, pol)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Created schedule %s (%s)\n", sch.Name, sch.ID[:8])
		fmt.Printf("  Next Run: %s\n", sch.NextRunAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

var scheduleTriggerCmd = &cobra.Command{
	Use:   "trigger <schedule-id>",
	Short: "Immediately dispatch a task for a schedule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		id := args[0]
		t, err := appInst.ScheduleSvc.TriggerSchedule(ctx, id)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Dispatched scheduled task %s (Task ID: %s)\n", t.Title, t.ID[:8])
		return nil
	},
}

var scheduleTickCmd = &cobra.Command{
	Use:   "tick",
	Short: "Check and dispatch any currently due schedules",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		count, err := appInst.ScheduleSvc.ProcessDueSchedules(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Processed due schedules: %d task(s) dispatched\n", count)
		return nil
	},
}

func init() {
	scheduleCreateCmd.Flags().StringVar(&schedName, "name", "", "Schedule name")
	scheduleCreateCmd.Flags().StringVar(&schedCron, "cron", "", "Cron expression (e.g. '@daily', '0 9 * * 1-5')")
	scheduleCreateCmd.Flags().StringVar(&schedAgent, "agent", "investment-agent", "Target Agent ID")
	scheduleCreateCmd.Flags().StringVar(&schedMisfire, "misfire", "SKIP", "Misfire policy: SKIP, RUN_ONCE, CATCH_UP")
	scheduleCreateCmd.Flags().StringVar(&schedJSON, "template", "{}", "Task template JSON payload")

	scheduleCmd.AddCommand(scheduleListCmd)
	scheduleCmd.AddCommand(scheduleCreateCmd)
	scheduleCmd.AddCommand(scheduleTriggerCmd)
	scheduleCmd.AddCommand(scheduleTickCmd)
}
