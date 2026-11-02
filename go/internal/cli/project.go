package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/config"
	domainBrief "github.com/wade/aia/internal/domain/brief"
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

var projectDeleteCmd = &cobra.Command{
	Use:     "delete <project-id>",
	Aliases: []string{"rm", "remove", "deregister"},
	Short:   "Delete / unregister a project from AIA",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projectID := args[0]
		if err := appInst.ProjectSvc.Delete(ctx, projectID); err != nil {
			return err
		}
		if appInst != nil {
			if appInst.WritingObserver != nil {
				appInst.WritingObserver.UnregisterProjectPath(projectID)
			}
			if appInst.CodingObserver != nil {
				appInst.CodingObserver.UnregisterProjectDir(projectID)
			}
		}
		fmt.Printf("✓ Project %s deleted / unregistered successfully (local files preserved).\n", projectID)
		return nil
	},
}

var projectBriefHistoryField string

var projectBriefCmd = &cobra.Command{
	Use:     "brief <project-id>",
	Aliases: []string{"show-brief"},
	Short:   "View the living Project Brief, field states, and readiness (AIA-21 §4, §11.1)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projectID := args[0]

		if appInst.GenericBriefSvc == nil {
			return fmt.Errorf("GenericBriefSvc is not initialized")
		}

		brief, err := appInst.GenericBriefSvc.GetBrief(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to load brief: %w", err)
		}
		if brief == nil {
			return fmt.Errorf("no project brief found for %s", projectID)
		}

		fmt.Printf("=== Project Brief: %s (Domain: %s, Version: %d) ===\n", brief.ProjectID, brief.Domain, brief.BriefVersion)
		fmt.Printf("成熟度狀態:       %s\n", brief.Readiness)
		fmt.Printf("成熟度診斷:       %s\n", brief.ReadinessRationale)
		if brief.RawIntent != "" {
			fmt.Printf("原始意圖描述:     %s\n", brief.RawIntent)
		}

		if brief.Generic.Purpose != "" {
			fmt.Printf("\n--- 通用專案規格 ---\n")
			fmt.Printf("專案目的:         %s\n", brief.Generic.Purpose)
			if brief.Generic.Audience != "" {
				fmt.Printf("目標受眾:         %s\n", brief.Generic.Audience)
			}
			if brief.Generic.Scope != "" {
				fmt.Printf("工作範圍:         %s\n", brief.Generic.Scope)
			}
		}

		if len(brief.Fields) > 0 {
			fmt.Printf("\n--- 領域規格欄位狀態 (Field States) ---\n")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "FIELD\tSTATUS\tCONF\tSOURCE\tVALUE\tRATIONALE")
			for k, f := range brief.Fields {
				valStr := fmt.Sprintf("%v", f.Value)
				if len(valStr) > 30 {
					valStr = valStr[:30] + "..."
				}
				ratStr := f.Rationale
				if len(ratStr) > 40 {
					ratStr = ratStr[:40] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%.2f\t%s\t%s\t%s\n",
					k, f.Status, f.Confidence, f.Source, valStr, ratStr)
			}
			_ = w.Flush()
		}

		return nil
	},
}

var projectBriefHistoryCmd = &cobra.Command{
	Use:   "history <project-id>",
	Short: "View immutable revision history of brief fields (AIA-21 §4.5, §11.1)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projectID := args[0]

		if appInst.GenericBriefRepo == nil {
			return fmt.Errorf("GenericBriefRepo is not initialized")
		}

		if projectBriefHistoryField == "" {
			return fmt.Errorf("--field <key> is required to inspect field revision history")
		}

		history, err := appInst.GenericBriefRepo.ListFieldHistory(ctx, projectID, projectBriefHistoryField)
		if err != nil {
			return fmt.Errorf("failed to query history: %w", err)
		}
		if len(history) == 0 {
			fmt.Printf("No history entries found for field '%s' in project '%s'.\n", projectBriefHistoryField, projectID)
			return nil
		}

		fmt.Printf("=== Revision History: %s.%s (%d revisions) ===\n", projectID, projectBriefHistoryField, len(history))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "VER\tSTATUS\tSOURCE\tVALUE\tTIMESTAMP\tREASON")
		for _, h := range history {
			vStr := fmt.Sprintf("%v", h.NewValue)
			if len(vStr) > 25 {
				vStr = vStr[:25] + "..."
			}
			fmt.Fprintf(w, "v%d\t%s\t%s\t%s\t%s\t%s\n",
				h.Version, h.NewStatus, h.Source, vStr, h.Timestamp.Format("2006-01-02 15:04"), h.Reason)
		}
		_ = w.Flush()
		return nil
	},
}

var projectBriefAnalysisFile string

var projectBriefPromptCmd = &cobra.Command{
	Use:   "prompt <project-id>",
	Short: "Generate the agy Brief Analysis prompt for a project (AIA-21 §6, §9)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projectID := args[0]

		if appInst.GenericBriefSvc == nil {
			return fmt.Errorf("GenericBriefSvc is not initialized")
		}

		b, err := appInst.GenericBriefSvc.GetBrief(ctx, projectID)
		if err != nil || b == nil {
			return fmt.Errorf("project brief for %s not found: %v", projectID, err)
		}

		catalog, err := domainBrief.GetCatalog(b.Domain)
		if err != nil {
			return fmt.Errorf("failed to get domain catalog: %w", err)
		}

		// Retrieve agent profile instructions if available
		var agentInstructions string
		if appInst.AgentSvc != nil {
			if ags, _ := appInst.AgentSvc.List(ctx); len(ags) > 0 {
				for _, ag := range ags {
					if ag.Domain == b.Domain {
						agentInstructions = ag.Instructions
						break
					}
				}
			}
		}

		prompt := domainBrief.BuildBriefAnalysisPrompt(b, catalog, agentInstructions, nil)
		fmt.Println(prompt)
		return nil
	},
}

var projectBriefApplyCmd = &cobra.Command{
	Use:   "apply-analysis <project-id>",
	Short: "Validate and apply an agy AgentBriefAnalysis JSON file to the project (AIA-21 §6.3)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projectID := args[0]

		if projectBriefAnalysisFile == "" {
			return fmt.Errorf("--file <path.json> is required")
		}

		data, err := os.ReadFile(projectBriefAnalysisFile)
		if err != nil {
			return fmt.Errorf("failed to read analysis file: %w", err)
		}

		var analysis domainBrief.AgentBriefAnalysis
		if err := json.Unmarshal(data, &analysis); err != nil {
			return fmt.Errorf("invalid analysis JSON: %w", err)
		}

		if appInst.GenericBriefSvc == nil {
			return fmt.Errorf("GenericBriefSvc is not initialized")
		}

		updatedBrief, err := appInst.GenericBriefSvc.ApplyAnalysis(ctx, projectID, &analysis, appInst.ClarificationSvc, "", "")
		if err != nil {
			return fmt.Errorf("failed to apply analysis: %w", err)
		}

		fmt.Printf("✓ Brief analysis applied successfully to project %s.\n", projectID)
		fmt.Printf("  新成熟度狀態: %s (%s)\n", updatedBrief.Readiness, updatedBrief.ReadinessRationale)
		fmt.Printf("  已套用預設值: %d 個\n", len(analysis.ProposedDefaults))
		fmt.Printf("  已建立釐清提問: %d 個\n", len(analysis.ClarificationQuestions))
		return nil
	},
}

func init() {
	projectCreateCmd.Flags().StringVar(&projectName, "name", "", "project name")
	projectCreateCmd.Flags().StringVar(&projectDesc, "description", "", "project description")
	projectCreateCmd.Flags().StringVar(&projectWS, "workspace", "", "project workspace directory path")
	projectCreateCmd.Flags().StringVar(&projectPath, "path", "", "project path (alias for --workspace)")
	projectCreateCmd.Flags().StringVar(&projectType, "type", "general", "project domain type (writing, coding, investment, general)")

	projectBriefHistoryCmd.Flags().StringVarP(&projectBriefHistoryField, "field", "f", "", "brief field key to inspect history")
	projectBriefApplyCmd.Flags().StringVarP(&projectBriefAnalysisFile, "file", "f", "", "path to AgentBriefAnalysis JSON file")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectArchiveCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectBriefCmd)
	projectBriefCmd.AddCommand(projectBriefHistoryCmd)
	projectBriefCmd.AddCommand(projectBriefPromptCmd)
	projectBriefCmd.AddCommand(projectBriefApplyCmd)
}
