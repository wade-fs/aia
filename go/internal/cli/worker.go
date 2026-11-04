package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	telemetryMetrics "github.com/wade/aia/internal/telemetry/metrics"
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

var monitorJSON bool

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Show real-time AIA Activity Monitor and operational telemetry dashboard (AIA-19 §7)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runActivityMonitor(cmd)
	},
}

func runActivityMonitor(cmd *cobra.Command) error {
	ctx := context.Background()

	var snap *telemetryMetrics.SystemMetricsSnapshot
	if appInst.MetricsRegistry != nil {
		var err error
		snap, err = appInst.MetricsRegistry.Collect(ctx, telemetryMetrics.MetricCollectorDeps{
			DBPath:       appInst.Config.Database.Path,
			TaskRepo:     repository.NewTaskRepository(appInst.DB),
			ExecRepo:     repository.NewExecutionRepository(appInst.DB),
			AutonomyRepo: appInst.AutonomyRepo,
			DecisionRepo: appInst.DecisionRepo,
			GoalRepo:     appInst.GoalRepo,
			ProblemRepo:  appInst.ProblemRepo,
			PlanRepo:     appInst.PlanRepo,
		})
		if err != nil {
			return err
		}
	}

	if monitorJSON && snap != nil {
		data, _ := json.MarshalIndent(snap, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("=== AIA Operational Monitor & Telemetry (AIA-19 §7) ===")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "指標類別\t指標名稱\t數值")
	fmt.Fprintln(w, "--------\t--------\t----")

	if appInst.WorkerMgr != nil {
		if actSnap, err := appInst.WorkerMgr.GetActivityMonitor(ctx); err == nil {
			fmt.Fprintf(w, "Worker\tSchedule Worker\t%s\n", actSnap.ScheduleWorker)
			fmt.Fprintf(w, "Worker\tTask Worker\t%s\n", actSnap.TaskWorker)
			fmt.Fprintf(w, "Worker\tNext Scheduled Run\t%s\n", actSnap.NextScheduledRun)
		}
	}

	if snap != nil {
		fmt.Fprintf(w, "Runtime\tUptime\t%.1fs\n", snap.Runtime.UptimeSeconds)
		fmt.Fprintf(w, "Runtime\tActive Goroutines\t%d\n", snap.Runtime.ActiveGoroutines)
		fmt.Fprintf(w, "Runtime\tAllocated Memory\t%.2f MB\n", float64(snap.Runtime.AllocMemoryBytes)/(1024*1024))
		fmt.Fprintf(w, "Runtime\tStorage Database Size\t%.2f KB\n", float64(snap.Runtime.StorageDBSizeByte)/1024)
		fmt.Fprintf(w, "Runtime\tGC Runs\t%d\n", snap.Runtime.NumGC)

		fmt.Fprintf(w, "Execution\tTotal Tasks\t%d (Pending: %d, Running: %d)\n",
			snap.Execution.TotalTasks, snap.Execution.PendingTasks, snap.Execution.ActiveTasks)
		fmt.Fprintf(w, "Execution\tTask Success Rate\t%.1f%% (Completed: %d, Failed: %d)\n",
			snap.Execution.TaskSuccessRate*100, snap.Execution.CompletedTasks, snap.Execution.FailedTasks)
		fmt.Fprintf(w, "Execution\tTotal Tokens Consumed\t%d tokens\n", snap.Execution.TotalTokensConsumed)
		fmt.Fprintf(w, "Execution\tEstimated Spend USD\t$%.4f\n", snap.Execution.TotalSpendUSD)

		fmt.Fprintf(w, "Autonomy\tTotal Cycles\t%d\n", snap.Autonomy.TotalCycles)
		fmt.Fprintf(w, "Autonomy\tTotal Decisions\t%d (No-Action: %d)\n",
			snap.Autonomy.TotalDecisions, snap.Autonomy.NoActionDecisions)
		fmt.Fprintf(w, "Autonomy\tCandidate Initiatives\t%d\n", snap.Autonomy.CandidateActionsGenerated)
		fmt.Fprintf(w, "Autonomy\tComposite Score\t%.2f%%\n", snap.Autonomy.CompositeAutonomyScore*100)

		fmt.Fprintf(w, "Domain\tActive Strategic Goals\t%d\n", snap.Domain.ActiveGoals)
		fmt.Fprintf(w, "Domain\tActive Complex Problems\t%d\n", snap.Domain.ActiveProblems)
	}

	w.Flush()
	return nil
}

func init() {
	monitorCmd.Flags().BoolVar(&monitorJSON, "json", false, "Output metrics in JSON format")

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
