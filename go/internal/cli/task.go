package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/task"
	taskService "github.com/wade/aia/internal/service/task"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var (
	taskTitle       string
	taskDesc        string
	taskAgentID     string
	taskProjectID   string
	taskStatusFilter string
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage and run agent tasks",
}

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	RunE: func(cmd *cobra.Command, args []string) error {
		if taskTitle == "" {
			return fmt.Errorf("--title is required")
		}
		ctx := context.Background()
		t, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
			Title:       taskTitle,
			Description: taskDesc,
			AgentID:     taskAgentID,
			ProjectID:   taskProjectID,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Task created successfully.\nID:     %s\nStatus: %s\n", t.ID, t.Status)
		return nil
	},
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		filter := repository.TaskFilter{
			Status: task.TaskStatus(taskStatusFilter),
		}
		tasks, err := appInst.TaskSvc.List(ctx, filter)
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tPRIORITY\tAGENT\tTITLE\tCREATED AT")
		for _, t := range tasks {
			idShort := t.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n", idShort, t.Status, t.Priority, t.AgentID, t.Title, t.CreatedAt.Format("2006-01-02 15:04"))
		}
		_ = w.Flush()
		return nil
	},
}

var taskShowCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		t, err := appInst.TaskSvc.Get(ctx, args[0])
		if err != nil {
			return err
		}

		deps, _ := appInst.TaskSvc.GetDependencies(ctx, t.ID)
		execs, _ := appInst.ExecutionSvc.ListByTask(ctx, t.ID)

		fmt.Println("=== Task Details ===")
		fmt.Printf("ID:          %s\n", t.ID)
		fmt.Printf("Title:       %s\n", t.Title)
		fmt.Printf("Status:      %s\n", t.Status)
		fmt.Printf("Agent ID:    %s\n", t.AgentID)
		fmt.Printf("Project ID:  %s\n", t.ProjectID)
		fmt.Printf("Priority:    %d\n", t.Priority)
		fmt.Printf("Risk:        %s\n", t.Risk)
		fmt.Printf("Created At:  %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
		if len(deps) > 0 {
			fmt.Printf("Depends On:  %s\n", strings.Join(deps, ", "))
		}
		fmt.Printf("Description:\n%s\n\n", t.Description)

		if len(execs) > 0 {
			fmt.Println("Executions:")
			for _, e := range execs {
				conv := ""
				if e.ConversationID != "" {
					conv = fmt.Sprintf(", agy Conv: %s", e.ConversationID)
				}
				fmt.Printf("  - [%s] Status: %s%s, Started: %v, Error: %s\n", e.ID[:8], e.Status, conv, e.StartedAt, e.ErrorMessage)
			}
		}
		return nil
	},
}

var taskRunCmd = &cobra.Command{
	Use:   "run <task-id>",
	Short: "Execute a READY task using the Scheduler",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		fmt.Printf("Executing task %s...\n", args[0])
		if err := appInst.Scheduler.RunTask(ctx, domain.ID(args[0])); err != nil {
			return fmt.Errorf("task execution failed: %w", err)
		}
		fmt.Println("Task execution finished successfully.")
		return nil
	},
}

var taskCancelCmd = &cobra.Command{
	Use:   "cancel <task-id>",
	Short: "Cancel a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := appInst.TaskSvc.Cancel(ctx, args[0]); err != nil {
			return err
		}
		fmt.Printf("Task %s cancelled.\n", args[0])
		return nil
	},
}

var taskResumeCmd = &cobra.Command{
	Use:   "resume <task-id>",
	Short: "Resume a failed or blocked task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := appInst.TaskSvc.Resume(ctx, args[0]); err != nil {
			return err
		}
		fmt.Printf("Task %s resumed to READY.\n", args[0])
		return nil
	},
}

func init() {
	taskCreateCmd.Flags().StringVar(&taskTitle, "title", "", "task title")
	taskCreateCmd.Flags().StringVar(&taskDesc, "description", "", "task description")
	taskCreateCmd.Flags().StringVar(&taskAgentID, "agent", "coding-agent", "assigned agent ID")
	taskCreateCmd.Flags().StringVar(&taskProjectID, "project", "", "project ID")

	taskListCmd.Flags().StringVar(&taskStatusFilter, "status", "", "filter by status (READY, RUNNING, etc.)")

	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskRunCmd)
	taskCmd.AddCommand(taskCancelCmd)
	taskCmd.AddCommand(taskResumeCmd)
}
