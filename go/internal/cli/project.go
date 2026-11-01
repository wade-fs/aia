package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	projectName string
	projectDesc string
	projectWS   string
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage workspace projects",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if projectName == "" {
			return fmt.Errorf("--name is required")
		}
		ctx := context.Background()
		p, err := appInst.ProjectSvc.Create(ctx, projectName, projectDesc, projectWS)
		if err != nil {
			return err
		}
		fmt.Printf("Project created successfully.\nID:        %s\nName:      %s\nWorkspace: %s\n", p.ID, p.Name, p.Workspace)
		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projects, err := appInst.ProjectSvc.List(ctx)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSTATUS\tWORKSPACE")
		for _, p := range projects {
			idShort := p.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", idShort, p.Name, p.Status, p.Workspace)
		}
		_ = w.Flush()
		return nil
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show <project-id>",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		p, err := appInst.ProjectSvc.Get(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Println("=== Project Details ===")
		fmt.Printf("ID:          %s\n", p.ID)
		fmt.Printf("Name:        %s\n", p.Name)
		fmt.Printf("Status:      %s\n", p.Status)
		fmt.Printf("Workspace:   %s\n", p.Workspace)
		fmt.Printf("Description: %s\n", p.Description)
		fmt.Printf("Created At:  %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))

		if appInst != nil && appInst.Reconstructor != nil {
			if recon, err := appInst.Reconstructor.ReconstructProjectState(ctx, string(p.ID)); err == nil && recon != nil {
				if len(recon.ActiveGoals) > 0 {
					fmt.Println("\n--- Active Goals ---")
					for _, g := range recon.ActiveGoals {
						fmt.Printf("• [%s] %s (%s, priority %d)\n", g.Level, g.Name, g.Status, g.Priority)
					}
				}
				if len(recon.PendingIssues) > 0 {
					fmt.Println("\n--- Pending Issues ---")
					for _, issue := range recon.PendingIssues {
						fmt.Printf("• %s\n", issue)
					}
				}
				if recon.LatestSnapshot != nil {
					fmt.Printf("\nLatest Snapshot: %s (%s, captured %s)\n",
						recon.LatestSnapshot.ID, recon.LatestSnapshot.SnapshotType, recon.LatestSnapshot.CreatedAt.Format("2006-01-02 15:04:05"))
				}
			}
		}
		return nil
	},
}

var projectArchiveCmd = &cobra.Command{
	Use:   "archive <project-id>",
	Short: "Archive a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		if err := appInst.ProjectSvc.Archive(ctx, args[0]); err != nil {
			return err
		}
		fmt.Printf("Project %s archived.\n", args[0])
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().StringVar(&projectName, "name", "", "project name")
	projectCreateCmd.Flags().StringVar(&projectDesc, "description", "", "project description")
	projectCreateCmd.Flags().StringVar(&projectWS, "workspace", "", "project workspace directory path")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectArchiveCmd)
}
