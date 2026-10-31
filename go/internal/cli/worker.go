package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage and inspect AIA Runtime Background Workers",
	Long: `Runtime background workers manage low-frequency automated data collection,
non-LLM prediction verification, and materiality-gated investment research.`,
}

var workerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status and metrics of runtime background workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.WorkerMgr == nil {
			return fmt.Errorf("worker manager not initialized")
		}
		stats := appInst.WorkerMgr.Status()

		fmt.Println("=== AIA Runtime Background Worker Status ===")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "Running State:\t%v\n", stats.Running)
		fmt.Fprintf(w, "Schedules Checked:\t%d\n", stats.SchedulesChecked)
		fmt.Fprintf(w, "Tasks Dispatched:\t%d\n", stats.TasksDispatched)
		fmt.Fprintf(w, "Tasks Completed:\t%d\n", stats.TasksCompleted)
		fmt.Fprintf(w, "Tasks Failed:\t%d\n", stats.TasksFailed)
		fmt.Fprintf(w, "Decisions Preserved (0 AI Tokens):\t%d\n", stats.DecisionsPreserved)
		fmt.Fprintf(w, "Decisions Updated:\t%d\n", stats.DecisionsUpdated)

		lastSched := "never"
		if stats.LastScheduleCheckAt != nil {
			lastSched = stats.LastScheduleCheckAt.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "Last Schedule Check:\t%s\n", lastSched)

		lastTask := "never"
		if stats.LastTaskExecutionAt != nil {
			lastTask = stats.LastTaskExecutionAt.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "Last Task Execution:\t%s\n", lastTask)

		lastCol := "never"
		if stats.LastCollectionAt != nil {
			lastCol = stats.LastCollectionAt.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "Last Data Collection:\t%s\n", lastCol)

		if stats.LastError != "" {
			fmt.Fprintf(w, "Last Error:\t%s\n", stats.LastError)
		}
		w.Flush()
		return nil
	},
}

var workerRunOnceCmd = &cobra.Command{
	Use:   "run-once",
	Short: "Execute one complete cycle of schedule dispatch and task execution",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.WorkerMgr == nil {
			return fmt.Errorf("worker manager not initialized")
		}
		ctx := context.Background()
		fmt.Println("⏳ Executing single worker cycle...")
		summary, err := appInst.WorkerMgr.RunOnce(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Dispatched Due Schedules: %d\n", summary.DispatchedSchedules)
		fmt.Printf("✓ Executed Ready Tasks: %d\n", summary.ExecutedTasks)
		return nil
	},
}

var workerDispatchCmd = &cobra.Command{
	Use:   "dispatch",
	Short: "Immediately scan and dispatch due recurring schedules",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.WorkerMgr == nil {
			return fmt.Errorf("worker manager not initialized")
		}
		ctx := context.Background()
		count, err := appInst.WorkerMgr.DispatchDue(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Dispatched %d due scheduled task(s)\n", count)
		return nil
	},
}

var workerProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "Immediately execute all currently READY tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appInst.WorkerMgr == nil {
			return fmt.Errorf("worker manager not initialized")
		}
		ctx := context.Background()
		count, err := appInst.WorkerMgr.ProcessReady(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Processed %d READY task(s)\n", count)
		return nil
	},
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Show real-time AIA Activity Monitor dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runActivityMonitor(cmd)
	},
}

func runActivityMonitor(cmd *cobra.Command) error {
	if appInst.WorkerMgr == nil {
		return fmt.Errorf("worker manager not initialized")
	}
	ctx := context.Background()
	snap, err := appInst.WorkerMgr.GetActivityMonitor(ctx)
	if err != nil {
		return err
	}

	fmt.Println("=== AIA Activity Monitor ===")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "項目\t狀態 / 時間")
	fmt.Fprintln(w, "----\t-----------")
	fmt.Fprintf(w, "Runtime\t%s\n", snap.Runtime)
	fmt.Fprintf(w, "Schedule Worker\t%s\n", snap.ScheduleWorker)
	fmt.Fprintf(w, "Task Worker\t%s\n", snap.TaskWorker)
	fmt.Fprintf(w, "Last Data Sync\t%s\n", snap.LastDataSync)
	fmt.Fprintf(w, "Last Investment Research\t%s\n", snap.LastInvestmentResearch)
	fmt.Fprintf(w, "Pending Tasks\t%d\n", snap.PendingTasks)
	fmt.Fprintf(w, "Next Scheduled Run\t%s\n", snap.NextScheduledRun)
	w.Flush()
	return nil
}

func init() {
	workerCmd.AddCommand(workerStatusCmd)
	workerCmd.AddCommand(workerRunOnceCmd)
	workerCmd.AddCommand(workerDispatchCmd)
	workerCmd.AddCommand(workerProcessCmd)
	workerCmd.AddCommand(&cobra.Command{
		Use:   "monitor",
		Short: "Show real-time AIA Activity Monitor dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runActivityMonitor(cmd)
		},
	})
}
