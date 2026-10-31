package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/wade/aia/agent-runtime/internal/runtime"
	"github.com/wade/aia/agent-runtime/internal/task"
)

var (
	cfgPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent Runtime - AI Worker Management Runtime PoC",
		Long:  "Agent Runtime is a lightweight, reliable runtime engine for executing and managing external AI coding workers like Aider.",
	}

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "./configs/config.yaml", "Path to configuration file")

	// Task subcommands
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Manage and run agent tasks",
	}

	var title, description, workerType string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			rt, err := runtime.New(cfgPath)
			if err != nil {
				return err
			}
			defer rt.Close()

			t, err := rt.TaskManager().Create(title, description, workerType)
			if err != nil {
				return err
			}

			fmt.Printf("Task created successfully!\n\n")
			fmt.Printf("ID:          %s\n", t.ID)
			fmt.Printf("Title:       %s\n", t.Title)
			fmt.Printf("Status:      %s\n", t.Status)
			fmt.Printf("Worker:      %s\n", t.WorkerType)
			fmt.Printf("Created At:  %s\n", t.CreatedAt.Format(time.RFC3339))
			return nil
		},
	}
	createCmd.Flags().StringVarP(&title, "title", "t", "", "Task title (required)")
	createCmd.Flags().StringVarP(&description, "description", "d", "", "Task description")
	createCmd.Flags().StringVarP(&workerType, "worker", "w", "aider", "Worker adapter type")
	_ = createCmd.MarkFlagRequired("title")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := runtime.New(cfgPath)
			if err != nil {
				return err
			}
			defer rt.Close()

			tasks, err := rt.TaskManager().List()
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tWORKER\tCREATED AT")
			fmt.Fprintln(w, "--\t-----\t------\t------\t----------")
			for _, t := range tasks {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					t.ID,
					truncateString(t.Title, 35),
					t.Status,
					t.WorkerType,
					t.CreatedAt.Format("2006-01-02 15:04:05"),
				)
			}
			return w.Flush()
		},
	}

	showCmd := &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show detailed information about a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			rt, err := runtime.New(cfgPath)
			if err != nil {
				return err
			}
			defer rt.Close()

			t, err := rt.TaskManager().Get(taskID)
			if err != nil {
				return err
			}

			fmt.Println("================================================================================")
			fmt.Printf("Task Details: %s\n", t.ID)
			fmt.Println("================================================================================")
			fmt.Printf("Title:               %s\n", t.Title)
			fmt.Printf("Status:              %s\n", t.Status)
			fmt.Printf("Worker Type:         %s\n", t.WorkerType)
			fmt.Printf("Workspace ID:        %s\n", t.WorkspaceID)
			fmt.Printf("Worker Execution ID: %s\n", t.WorkerExecutionID)
			fmt.Printf("Git Base Commit:     %s\n", t.GitBaseCommit)
			fmt.Printf("Git Final Commit:    %s\n", t.GitFinalCommit)
			fmt.Printf("Created At:          %s\n", t.CreatedAt.Format(time.RFC3339))
			if t.StartedAt != nil {
				fmt.Printf("Started At:          %s\n", t.StartedAt.Format(time.RFC3339))
			}
			if t.CompletedAt != nil {
				fmt.Printf("Completed At:        %s\n", t.CompletedAt.Format(time.RFC3339))
			}
			fmt.Println("--------------------------------------------------------------------------------")
			fmt.Printf("Description:\n%s\n", t.Description)
			if t.Result != "" {
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Printf("Result:\n%s\n", t.Result)
			}
			if t.ValidationResult != "" {
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Printf("Validation Result:\n%s\n", t.ValidationResult)
			}
			if t.Error != "" {
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Printf("Error:\n%s\n", t.Error)
			}

			// Events history
			events, err := rt.Storage().ListTaskEvents(context.Background(), taskID)
			if err == nil && len(events) > 0 {
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Println("Event History:")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "TIMESTAMP\tEVENT\tDETAIL")
				for _, ev := range events {
					fmt.Fprintf(w, "%s\t%s\t%s\n",
						ev.CreatedAt.Format("2006-01-02 15:04:05"),
						ev.EventType,
						ev.Detail,
					)
				}
				_ = w.Flush()
			}
			fmt.Println("================================================================================")
			return nil
		},
	}

	runCmd := &cobra.Command{
		Use:   "run <task-id>",
		Short: "Execute a task through the runtime pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			rt, err := runtime.New(cfgPath)
			if err != nil {
				return err
			}
			defer rt.Close()

			ctx := context.Background()
			fmt.Printf("Starting execution of task %s...\n", taskID)
			if err := rt.RunTask(ctx, taskID); err != nil {
				return fmt.Errorf("task run error: %w", err)
			}
			fmt.Printf("Task %s execution finished successfully!\n", taskID)
			return nil
		},
	}

	cancelCmd := &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a running or pending task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			rt, err := runtime.New(cfgPath)
			if err != nil {
				return err
			}
			defer rt.Close()

			t, err := rt.TaskManager().Get(taskID)
			if err != nil {
				return err
			}

			// If worker is running, attempt cancellation
			if t.WorkerExecutionID != "" {
				_ = rt.WorkerManager().Cancel(context.Background(), t.WorkerExecutionID)
			}

			if err := rt.StateManager().Transition(taskID, t.Status, task.TaskStatusCancelled); err != nil {
				return fmt.Errorf("failed to cancel task: %w", err)
			}
			fmt.Printf("Task %s cancelled successfully.\n", taskID)
			return nil
		},
	}

	taskCmd.AddCommand(createCmd, listCmd, showCmd, runCmd, cancelCmd)

	// Recovery subcommand
	recoveryCmd := &cobra.Command{
		Use:   "recovery",
		Short: "Run crash recovery checks for unfinished tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := runtime.New(cfgPath)
			if err != nil {
				return err
			}
			defer rt.Close()

			fmt.Println("Running Agent Runtime Recovery cycle...")
			ctx := context.Background()
			if err := rt.RecoveryManager().Run(ctx); err != nil {
				return fmt.Errorf("recovery failed: %w", err)
			}
			fmt.Println("Recovery cycle completed.")
			return nil
		},
	}

	// Worker subcommands
	workerCmd := &cobra.Command{
		Use:   "worker",
		Short: "Inspect and manage workers",
	}

	workerHealthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check worker availability and health",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := runtime.New(cfgPath)
			if err != nil {
				return err
			}
			defer rt.Close()

			ctx := context.Background()
			err = rt.WorkerManager().HealthCheck(ctx)
			if err != nil {
				fmt.Printf("[HEALTH CHECK FAILED] %v\n", err)
				return err
			}
			fmt.Println("[HEALTH CHECK PASSED] Worker adapter is configured and ready.")
			return nil
		},
	}
	workerCmd.AddCommand(workerHealthCmd)

	rootCmd.AddCommand(taskCmd, recoveryCmd, workerCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
