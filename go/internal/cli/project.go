package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/config"
	domainProject "github.com/wade/aia/internal/domain/project"
)

var (
	projectName string
	projectDesc string
	projectWS   string
	projectPath string
	projectType string
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage workspace projects",
}

func resolveWorkspacePath(raw string) (string, error) {
	if raw == "" {
		return os.Getwd()
	}
	expanded := config.ExpandPath(raw)
	return filepath.Abs(expanded)
}

var projectCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"register"},
	Short:   "Create or register a new project (AIA-15, thinking.md §一/§三)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if projectName == "" {
			return fmt.Errorf("--name is required")
		}

		ws := projectWS
		if ws == "" && projectPath != "" {
			ws = projectPath
		}
		absWS, err := resolveWorkspacePath(ws)
		if err != nil {
			return fmt.Errorf("failed to resolve workspace path %q: %w", ws, err)
		}

		dt := domainProject.DomainType(projectType)
		if dt == "" {
			dt = domainProject.DomainGeneral
		}

		ctx := context.Background()
		p, err := appInst.ProjectSvc.Register(ctx, projectName, projectDesc, absWS, dt)
		if err != nil {
			return err
		}

		// Dynamically register into WritingObserver if writing domain
		if dt == domainProject.DomainWriting && appInst != nil && appInst.WritingObserver != nil {
			appInst.WritingObserver.RegisterProjectPath(string(p.ID), p.Workspace)
		}

		fmt.Printf("Project registered successfully.\nID:          %s\nName:        %s\nType:        %s\nWorkspace:   %s\n", p.ID, p.Name, p.DomainType, p.Workspace)
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
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tSTATUS\tWORKSPACE")
		for _, p := range projects {
			idShort := p.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			dt := string(p.DomainType)
			if dt == "" {
				dt = "general"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", idShort, p.Name, dt, p.Status, p.Workspace)
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
		fmt.Printf("Type:        %s\n", p.DomainType)
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
	projectCreateCmd.Flags().StringVar(&projectPath, "path", "", "project path (alias for --workspace)")
	projectCreateCmd.Flags().StringVar(&projectType, "type", "general", "project domain type (writing, coding, investment, general)")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectArchiveCmd)
}
