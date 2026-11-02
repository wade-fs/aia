package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wade/aia/internal/config"
	domainAutonomy "github.com/wade/aia/internal/domain/autonomy"
	domainProject "github.com/wade/aia/internal/domain/project"
	taskService "github.com/wade/aia/internal/service/task"
)

var (
	codingProjectID string
	codingRepoPath  string
	codingRepoName  string
)

var codingCmd = &cobra.Command{
	Use:   "coding",
	Short: "Manage Coding Agent repositories, run code health scans, and dispatch coding tasks",
	Long: `The coding command suite interacts with the AIA Coding Agent.
It manages Git repositories, inspects code health and build diagnostics,
and dispatches development, debugging, and testing tasks.`,
}

var codingReposCmd = &cobra.Command{
	Use:     "repos",
	Aliases: []string{"list"},
	Short:   "List registered coding repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projects, err := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainCoding)
		if err != nil {
			return fmt.Errorf("failed to list coding projects: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSTATUS\tWORKSPACE")
		for _, p := range projects {
			idShort := string(p.ID)
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", idShort, p.Name, p.Status, p.Workspace)
		}
		_ = w.Flush()
		return nil
	},
}

var codingRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new coding repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		if codingRepoName == "" {
			return fmt.Errorf("--name is required")
		}
		ws := codingRepoPath
		if ws == "" {
			var err error
			ws, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		absWS, err := filepath.Abs(config.ExpandPath(ws))
		if err != nil {
			return err
		}

		ctx := context.Background()
		p, err := appInst.ProjectSvc.Register(ctx, codingRepoName, "Coding Repository", absWS, domainProject.DomainCoding)
		if err != nil {
			return err
		}
		if appInst.CodingObserver != nil {
			appInst.CodingObserver.RegisterProjectDir(string(p.ID), p.Workspace)
		}
		fmt.Printf("Coding repository registered successfully.\nID:        %s\nName:      %s\nWorkspace: %s\n", p.ID, p.Name, p.Workspace)
		return nil
	},
}

var codingScanCmd = &cobra.Command{
	Use:   "scan [project-id]",
	Short: "Run code health and build/test defect scan on repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		targetID := codingProjectID
		if len(args) > 0 {
			targetID = args[0]
		}
		if targetID == "" {
			// Find first coding project or default
			projects, err := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainCoding)
			if err == nil && len(projects) > 0 {
				targetID = string(projects[0].ID)
			} else {
				targetID = "default"
			}
		}

		fmt.Printf("🔍 Scanning repository for project '%s'...\n", targetID)
		if appInst.CodingObserver == nil {
			return fmt.Errorf("CodingObserver is not initialized")
		}

		observations, err := appInst.CodingObserver.Observe(ctx, targetID)
		if err != nil {
			return fmt.Errorf("code health scan failed: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "CATEGORY\tSEVERITY\tSUMMARY")
		anomalyCount := 0
		for _, obs := range observations {
			if obs.Category == domainAutonomy.ObsCategoryAnomaly {
				anomalyCount++
			}
			fmt.Fprintf(w, "%s\t%d\t%s\n", obs.Category, obs.Severity, obs.Summary)
		}
		_ = w.Flush()

		if anomalyCount == 0 {
			fmt.Printf("\n✅ All clean! No defects or build failures detected in '%s'.\n", targetID)
		} else {
			fmt.Printf("\n⚠️ Detected %d actionable defect(s) requiring remediation.\n", anomalyCount)
		}
		return nil
	},
}

var codingPromptCmd = &cobra.Command{
	Use:   "prompt <instruction>",
	Short: "Submit a coding instruction or development prompt to Coding Agent",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promptText := strings.Join(args, " ")

		targetProj := codingProjectID
		if targetProj == "" {
			projects, err := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainCoding)
			if err == nil && len(projects) > 0 {
				targetProj = string(projects[0].ID)
			} else {
				targetProj = "default"
			}
		}

		fmt.Printf("🚀 Dispatching task to Coding Agent (Project: %s)...\n", targetProj)
		t, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
			Title:       fmt.Sprintf("Coding: %s", truncate(promptText, 50)),
			Description: promptText,
			AgentID:     "coding-agent",
			ProjectID:   targetProj,
		})
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		fmt.Printf("Task created: %s (Status: %s). Executing via Scheduler...\n", t.ID, t.Status)
		if err := appInst.Scheduler.RunTask(ctx, t.ID); err != nil {
			return fmt.Errorf("task execution encountered error: %w", err)
		}

		execs, _ := appInst.ExecutionSvc.ListByTask(ctx, t.ID)
		if len(execs) > 0 {
			latest := execs[0]
			exitCode := 0
			if latest.ExitCode != nil {
				exitCode = *latest.ExitCode
			}
			fmt.Printf("\n=== Execution Summary (Status: %s, ExitCode: %d, Result: %s) ===\n", latest.Status, exitCode, latest.ResultStatus)
			if latest.ConversationID != "" {
				fmt.Printf("agy Conversation ID: %s\n", latest.ConversationID)
			}
			if latest.ErrorMessage != "" {
				fmt.Printf("Error: %s\n", latest.ErrorMessage)
			}
			events, _ := appInst.ExecutionSvc.ListEvents(ctx, latest.ID)
			if len(events) > 0 {
				fmt.Println("--- Output Stream ---")
				for _, ev := range events {
					var payloadMap map[string]any
					if err := json.Unmarshal(ev.Payload, &payloadMap); err == nil {
						if text, ok := payloadMap["text"].(string); ok && text != "" {
							fmt.Print(text)
						} else if content, ok := payloadMap["content"].(string); ok && content != "" {
							fmt.Print(content)
						} else if chunk, ok := payloadMap["chunk"].(string); ok && chunk != "" {
							fmt.Print(chunk)
						}
					}
				}
				fmt.Println()
			}
		} else {
			fmt.Println("Task completed.")
		}
		return nil
	},
}

var codingInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show Coding Agent system instructions, model, and capabilities",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ag, err := appInst.AgentSvc.Get(ctx, "coding-agent")
		if err != nil {
			return fmt.Errorf("failed to get coding agent info: %w", err)
		}

		fmt.Println("=== Coding Agent Information ===")
		fmt.Printf("ID:           %s\n", ag.ID)
		fmt.Printf("Name:         %s\n", ag.Name)
		fmt.Printf("Domain:       %s\n", ag.Domain)
		fmt.Printf("Version:      %d\n", ag.Version)
		fmt.Printf("Enabled:      %v\n", ag.Enabled)
		fmt.Printf("Capabilities: %s\n", strings.Join(ag.Capabilities, ", "))
		fmt.Printf("\nInstructions (Prompt):\n%s\n", ag.Instructions)
		return nil
	},
}

var codingRemoveCmd = &cobra.Command{
	Use:     "remove <project-id>",
	Aliases: []string{"rm", "delete", "deregister"},
	Short:   "Remove / unregister a coding repository from AIA",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projectID := args[0]
		if err := appInst.ProjectSvc.Delete(ctx, projectID); err != nil {
			return fmt.Errorf("failed to remove coding project %s: %w", projectID, err)
		}
		if appInst.CodingObserver != nil {
			appInst.CodingObserver.UnregisterProjectDir(projectID)
		}
		fmt.Printf("✓ Successfully unregistered Coding repository %s from AIA (local files preserved).\n", projectID)
		return nil
	},
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func init() {
	codingRegisterCmd.Flags().StringVar(&codingRepoName, "name", "", "repository project name")
	codingRegisterCmd.Flags().StringVar(&codingRepoPath, "path", "", "repository workspace path (default: current directory)")

	codingScanCmd.Flags().StringVarP(&codingProjectID, "project", "p", "", "target project ID")
	codingPromptCmd.Flags().StringVarP(&codingProjectID, "project", "p", "", "target project ID")

	codingCmd.AddCommand(codingReposCmd)
	codingCmd.AddCommand(codingRegisterCmd)
	codingCmd.AddCommand(codingRemoveCmd)
	codingCmd.AddCommand(codingScanCmd)
	codingCmd.AddCommand(codingPromptCmd)
	codingCmd.AddCommand(codingInfoCmd)
}
