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
			projects, lErr := appInst.ProjectSvc.ListByDomain(ctx, domainProject.DomainWriting)
			if lErr == nil {
				for i := range projects {
					if strings.HasPrefix(string(projects[i].ID), targetID) || strings.EqualFold(projects[i].Name, targetID) {
						p = &projects[i]
						err = nil
						break
					}
				}
			}
			if err != nil {
				return fmt.Errorf("project not found: %w", err)
			}
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
		if manifest.Project.Subtitle != "" {
			fmt.Printf("Subtitle:         %s\n", manifest.Project.Subtitle)
		}
		if manifest.Project.Series != "" {
			fmt.Printf("Series:           %s\n", manifest.Project.Series)
		}
		fmt.Printf("Type:             %s\n", manifest.Project.Type)
		if manifest.Project.Era != "" {
			fmt.Printf("Era:              %s\n", manifest.Project.Era)
		}
		fmt.Printf("Language:         %s\n", manifest.Project.Language)
		if manifest.Project.Status != "" {
			fmt.Printf("Status:           %s\n", manifest.Project.Status)
		}

		if manifest.Scale != nil {
			fmt.Println("\n--- Scale & Layout ---")
			if manifest.Scale.TotalVolumes > 0 {
				fmt.Printf("Total Volumes:    %d\n", manifest.Scale.TotalVolumes)
			}
			if manifest.Scale.TotalChapters > 0 {
				fmt.Printf("Total Chapters:   %d\n", manifest.Scale.TotalChapters)
			}
			if manifest.Scale.ChaptersPerVolume > 0 {
				fmt.Printf("Chaps / Volume:   %d\n", manifest.Scale.ChaptersPerVolume)
			}
			if manifest.Scale.VolumeStructure != "" {
				fmt.Printf("Vol. Structure:   %s\n", manifest.Scale.VolumeStructure)
			}
		}

		if manifest.Laboratory != nil && manifest.Laboratory.Enabled {
			fmt.Println("\n--- Dual World / Laboratory ---")
			fmt.Printf("Perspective:      %s\n", manifest.Laboratory.Perspective)
			if manifest.Laboratory.Specification != "" {
				fmt.Printf("Specification:    %s\n", manifest.Laboratory.Specification)
			}
			if manifest.Laboratory.Guidelines != "" {
				fmt.Printf("Guidelines:       %s\n", manifest.Laboratory.Guidelines)
			}
			if manifest.Laboratory.VolumeMap != "" {
				fmt.Printf("Volume Map:       %s\n", manifest.Laboratory.VolumeMap)
			}
		}

		if manifest.Worldview != nil && manifest.Worldview.SystemType != "" {
			fmt.Println("\n--- Worldview System ---")
			fmt.Printf("System Type:      %s\n", manifest.Worldview.SystemType)
			if len(manifest.Worldview.EntryStages) > 0 {
				fmt.Printf("Entry Stages:     %s\n", strings.Join(manifest.Worldview.EntryStages, " -> "))
			}
			if len(manifest.Worldview.FormalRealms) > 0 {
				fmt.Printf("Formal Realms:    %s\n", strings.Join(manifest.Worldview.FormalRealms, " -> "))
			}
		}

		fmt.Println("\n--- Structure ---")
		fmt.Printf("Outline:          %s\n", manifest.Structure.Outline)
		if manifest.Structure.VolumesPlan != "" {
			fmt.Printf("Volumes Plan:     %s\n", manifest.Structure.VolumesPlan)
		}
		if manifest.Structure.Volumes != "" {
			fmt.Printf("Volumes:          %s\n", manifest.Structure.Volumes)
		}
		if manifest.Structure.Chapters != "" {
			fmt.Printf("Chapters:         %s\n", manifest.Structure.Chapters)
		}
		if manifest.Structure.Timeline != "" {
			fmt.Printf("Timeline:         %s\n", manifest.Structure.Timeline)
		}
		if manifest.Structure.TimelineAudit != "" {
			fmt.Printf("Timeline Audit:   %s\n", manifest.Structure.TimelineAudit)
		}
		if manifest.Structure.ChapterLedger != "" {
			fmt.Printf("Chapter Ledger:   %s\n", manifest.Structure.ChapterLedger)
		}
		if manifest.Structure.NarrativeValidation != "" {
			fmt.Printf("Narrative Valid.: %s\n", manifest.Structure.NarrativeValidation)
		}
		if manifest.Structure.ResearchQA != "" {
			fmt.Printf("Research QA:      %s\n", manifest.Structure.ResearchQA)
		}
		if manifest.Structure.Characters != "" {
			fmt.Printf("Characters:       %s\n", manifest.Structure.Characters)
		}
		if manifest.Structure.CharacterNetwork != "" {
			fmt.Printf("Char. Network:    %s\n", manifest.Structure.CharacterNetwork)
		}
		if manifest.Structure.CharacterLifecycle != "" {
			fmt.Printf("Char. Lifecycle:  %s\n", manifest.Structure.CharacterLifecycle)
		}
		if manifest.Structure.Geography != "" {
			fmt.Printf("Geography:        %s\n", manifest.Structure.Geography)
		}
		if manifest.Structure.SocialSystems != "" {
			fmt.Printf("Social Systems:   %s\n", manifest.Structure.SocialSystems)
		}
		if manifest.Structure.MilitaryReference != "" {
			fmt.Printf("Military Ref.:    %s\n", manifest.Structure.MilitaryReference)
		}
		if manifest.Structure.Bibliography != "" {
			fmt.Printf("Bibliography:     %s\n", manifest.Structure.Bibliography)
		}
		if manifest.Structure.EvidenceReview != "" {
			fmt.Printf("Evidence Review:  %s\n", manifest.Structure.EvidenceReview)
		}
		if manifest.Structure.MasterBible != "" {
			fmt.Printf("Master Bible:     %s\n", manifest.Structure.MasterBible)
		}
		if manifest.Structure.Rules != "" {
			fmt.Printf("Rules:            %s\n", manifest.Structure.Rules)
		}
		if manifest.Structure.Guidelines != "" {
			fmt.Printf("Guidelines:       %s\n", manifest.Structure.Guidelines)
		}

		fmt.Println("\n--- Writing Policy & Authority ---")
		fmt.Printf("Historical Acc.:  %s\n", manifest.WritingPolicy.HistoricalAccuracy)
		fmt.Printf("Battle Tactics:   %v\n", manifest.WritingPolicy.BattleTacticsFocus)
		fmt.Printf("Preserve Figures: %v\n", manifest.WritingPolicy.PreserveHistoricalFigures)
		fmt.Printf("Req Consistency:  %v\n", manifest.WritingPolicy.RequireConsistencyCheck)
		if manifest.WritingPolicy.AntiAnachronism {
			fmt.Printf("Anti-Anachronism: %v\n", manifest.WritingPolicy.AntiAnachronism)
		}
		if manifest.WritingPolicy.AntiTeleportation {
			fmt.Printf("Anti-Teleport:    %v\n", manifest.WritingPolicy.AntiTeleportation)
		}
		if len(manifest.WritingPolicy.NarrativeDimensions) > 0 {
			fmt.Printf("Narrative Dim.:   %s\n", strings.Join(manifest.WritingPolicy.NarrativeDimensions, ", "))
		}
		fmt.Printf("Confirmed Set.:   %s\n", manifest.Authority.ConfirmedSettings)
		fmt.Printf("Historical Facts: %s\n", manifest.Authority.HistoricalFacts)
		if len(manifest.Authority.Adjudications) > 0 {
			fmt.Printf("Adjudications:    %s\n", strings.Join(manifest.Authority.Adjudications, "; "))
		}
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

var writingRemoveCmd = &cobra.Command{
	Use:     "remove <project-id>",
	Aliases: []string{"rm", "delete", "deregister"},
	Short:   "Remove / unregister a novel project from AIA",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		projectID := args[0]
		if err := appInst.ProjectSvc.Delete(ctx, projectID); err != nil {
			return fmt.Errorf("failed to remove writing project %s: %w", projectID, err)
		}
		if appInst.WritingObserver != nil {
			appInst.WritingObserver.UnregisterProjectPath(projectID)
		}
		fmt.Printf("✓ Successfully unregistered Writing project %s from AIA (local files preserved).\n", projectID)
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
	writingCmd.AddCommand(writingRemoveCmd)
	writingCmd.AddCommand(writingScanCmd)
	writingCmd.AddCommand(writingManifestCmd)
	writingCmd.AddCommand(writingPromptCmd)
	writingCmd.AddCommand(writingInfoCmd)
}
