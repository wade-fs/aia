package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system summary and running tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		tasks, err := appInst.TaskSvc.List(ctx, repository.TaskFilter{})
		if err != nil {
			return err
		}

		agents, _ := appInst.AgentSvc.List(ctx)
		projects, _ := appInst.ProjectSvc.List(ctx)

		fmt.Println("=== AIA Personal Agent Status ===")
		fmt.Printf("Data Directory:      %s\n", appInst.Config.Runtime.DataDir)
		fmt.Printf("Database Path:       %s\n", appInst.Config.Database.Path)
		fmt.Printf("Provider:            %s\n", appInst.Provider.Name())
		fmt.Printf("Active Projects:     %d\n", len(projects))
		fmt.Printf("Registered Agents:   %d\n", len(agents))
		fmt.Printf("Total Tasks:         %d\n\n", len(tasks))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "TASK ID\tSTATUS\tPRIORITY\tAGENT\tTITLE")
		for _, t := range tasks {
			if t.Status != "COMPLETED" && t.Status != "CANCELLED" {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", t.ID[:8], t.Status, t.Priority, t.AgentID, t.Title)
			}
		}
		_ = w.Flush()
		return nil
	},
}
