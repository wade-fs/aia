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
	"github.com/wade/aia/internal/domain/writing"
	taskService "github.com/wade/aia/internal/service/task"
	"gopkg.in/yaml.v3"
)

var (
	writingProjectID string
	writingBookPath  string
	writingBookName  string
)

var writingCmd = &cobra.Command{
	Use:   "writing",
	Short: "Manage Writing Agent novel projects, book.yaml manifests, consistency scans, and prompts",
	Long: `The writing command suite interacts with the AIA Writing Agent.
It manages novel manuscripts, audits chronological consistency and historical fact tiers,
inspects book.yaml manifests, and dispatches chapter generation and editorial tasks.`,
}

var writingBooksCmd = &cobra.Command{
	Use:     "books",
	Aliases: []string{"list"},
	Short:   "List registered writing and novel projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projects, err := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainWriting)
		if err != nil {
			return fmt.Errorf("failed to list writing projects: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tMANIFEST\tSTATUS\tWORKSPACE")
		for _, p := range projects {
			idShort := string(p.ID)
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			hasManifest := "no"
			if _, err := os.Stat(filepath.Join(p.Workspace, "book.yaml")); err == nil {
				hasManifest = "book.yaml"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", idShort, p.Name, hasManifest, p.Status, p.Workspace)
		}
		_ = w.Flush()
		return nil
	},
}

var writingRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new writing / novel directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		if writingBookName == "" {
			return fmt.Errorf("--name is required")
		}
		ws := writingBookPath
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
		p, err := appInst.ProjectSvc.Register(ctx, writingBookName, "Novel Project", absWS, domainProject.DomainWriting)
		if err != nil {
			return err
		}
		if appInst.WritingObserver != nil {
			appInst.WritingObserver.RegisterProjectPath(string(p.ID), p.Workspace)
		}
		fmt.Printf("Novel project registered successfully.\nID:        %s\nName:      %s\nWorkspace: %s\n", p.ID, p.Name, p.Workspace)
		return nil
	},
}

var writingScanCmd = &cobra.Command{
	Use:   "scan [project-id]",
	Short: "Run manuscript consistency and historical timeline audit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		targetID := writingProjectID
		if len(args) > 0 {
			targetID = args[0]
		}
		if targetID == "" {
			projects, err := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainWriting)
			if err == nil && len(projects) > 0 {
				targetID = string(projects[0].ID)
			} else {
				targetID = "yuan"
			}
		}

		fmt.Printf("📖 Auditing manuscript consistency for project '%s'...\n", targetID)
		if appInst.WritingObserver == nil {
			return fmt.Errorf("WritingObserver is not initialized")
		}

		observations, err := appInst.WritingObserver.Observe(ctx, targetID)
		if err != nil {
			return fmt.Errorf("consistency scan failed: %w", err)
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
			fmt.Printf("\n✅ Manuscript is chronologically consistent! No historical contradictions in '%s'.\n", targetID)
		} else {
			fmt.Printf("\n⚠️ Detected %d contradiction(s) requiring editorial review/remediation.\n", anomalyCount)
		}
		return nil
	},
}

var writingManifestCmd = &cobra.Command{
	Use:   "manifest [project-id]",
	Short: "Inspect book.yaml manifest and structure for a novel project",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		targetID := writingProjectID
		if len(args) > 0 {
			targetID = args[0]
		}
		if targetID == "" {
			projects, err := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainWriting)
			if err == nil && len(projects) > 0 {
				targetID = string(projects[0].ID)
			} else {
				targetID = "yuan"
			}
		}

		p, err := appInst.ProjectSvc.Get(ctx, targetID)
		if err != nil {
			return fmt.Errorf("project not found: %w", err)
		}

		manifestFile := filepath.Join(p.Workspace, "book.yaml")
		data, err := os.ReadFile(manifestFile)
		if err != nil {
			return fmt.Errorf("no book.yaml found at %s: %w", manifestFile, err)
		}

		var manifest writing.BookManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("failed to parse book.yaml: %w", err)
		}

		fmt.Printf("=== Book Manifest: %s (%s) ===\n", manifest.Project.Name, manifest.Project.ID)
		fmt.Printf("Path:             %s\n", manifestFile)
		fmt.Printf("Type:             %s\n", manifest.Project.Type)
		fmt.Printf("Language:         %s\n", manifest.Project.Language)
		fmt.Println("\n--- Structure ---")
		fmt.Printf("Outline:          %s\n", manifest.Structure.Outline)
		fmt.Printf("Characters:       %s\n", manifest.Structure.Characters)
		fmt.Printf("Volumes:          %s\n", manifest.Structure.Volumes)
		fmt.Printf("Chapters:         %s\n", manifest.Structure.Chapters)
		fmt.Println("\n--- Writing Policy & Authority ---")
		fmt.Printf("Historical Acc.:  %s\n", manifest.WritingPolicy.HistoricalAccuracy)
		fmt.Printf("Preserve Figures: %v\n", manifest.WritingPolicy.PreserveHistoricalFigures)
		fmt.Printf("Req Consistency:  %v\n", manifest.WritingPolicy.RequireConsistencyCheck)
		fmt.Printf("Confirmed Set.:   %s\n", manifest.Authority.ConfirmedSettings)
		fmt.Printf("Historical Facts: %s\n", manifest.Authority.HistoricalFacts)
		return nil
	},
}

var writingPromptCmd = &cobra.Command{
	Use:   "prompt <instruction>",
	Short: "Submit a creative writing, editorial, or chapter draft prompt to Writing Agent",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		promptText := strings.Join(args, " ")

		targetProj := writingProjectID
		if targetProj == "" {
			projects, err := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainWriting)
			if err == nil && len(projects) > 0 {
				targetProj = string(projects[0].ID)
			} else {
				targetProj = "yuan"
			}
		}

		fmt.Printf("✍️ Dispatching task to Writing Agent (Project: %s)...\n", targetProj)
		t, err := appInst.TaskSvc.Create(ctx, taskService.CreateTaskRequest{
			Title:       fmt.Sprintf("Writing: %s", truncate(promptText, 50)),
			Description: promptText,
			AgentID:     "writing-agent",
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

var writingInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show Writing Agent system instructions, model, and capabilities",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ag, err := appInst.AgentSvc.Get(ctx, "writing-agent")
		if err != nil {
			return fmt.Errorf("failed to get writing agent info: %w", err)
		}

		fmt.Println("=== Writing Agent Information ===")
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

func init() {
	writingRegisterCmd.Flags().StringVar(&writingBookName, "name", "", "novel project name")
	writingRegisterCmd.Flags().StringVar(&writingBookPath, "path", "", "novel directory path (default: current directory)")

	writingScanCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "target project ID")
	writingManifestCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "target project ID")
	writingPromptCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "target project ID")

	writingCmd.AddCommand(writingBooksCmd)
	writingCmd.AddCommand(writingRegisterCmd)
	writingCmd.AddCommand(writingScanCmd)
	writingCmd.AddCommand(writingManifestCmd)
	writingCmd.AddCommand(writingPromptCmd)
	writingCmd.AddCommand(writingInfoCmd)
}
