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
	writingService "github.com/wade/aia/internal/service/writing"
	"gopkg.in/yaml.v3"
)

var (
	writingProjectID string
	writingBookPath  string
	writingBookName  string
	writingIntent    string
	writingGenre     string
	writingEra       string
	writingAnswer    string
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

var writingInitCmd = &cobra.Command{
	Use:   "init [workspace-path]",
	Short: "Initialize a writing project from creative intent with Brief gap analysis (Node A ➔ F)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		ws := "."
		if len(args) > 0 {
			ws = args[0]
		} else if writingBookPath != "" {
			ws = writingBookPath
		}
		absWS, err := filepath.Abs(config.ExpandPath(ws))
		if err != nil {
			return err
		}

		if writingIntent == "" {
			return fmt.Errorf("--intent is required (使用者創作意圖)")
		}

		title := writingBookName
		if title == "" {
			title = filepath.Base(absWS)
		}

		genre := writingGenre
		if genre == "" {
			genre = "historical_fiction"
		}

		projID := writingProjectID
		if projID == "" {
			projID = filepath.Base(absWS)
		}

		req := writingService.InitiateProjectRequest{
			ProjectID: projID,
			Title:     title,
			RawIntent: writingIntent,
			Genre:     genre,
			Era:       writingEra,
			Workspace: absWS,
		}

		resp, err := appInst.WritingWorkflowSvc.InitiateProject(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to initiate writing project: %w", err)
		}

		fmt.Printf("=== 創作專案初始化完成: %s (%s) ===\n", resp.Brief.Title, resp.Brief.ProjectID)
		fmt.Printf("狀態:       %s\n", resp.Status)
		fmt.Printf("類型:       %s\n", resp.Brief.Genre)
		if resp.Brief.Era != "" {
			fmt.Printf("年代:       %s\n", resp.Brief.Era)
		}
		fmt.Printf("Workspace:  %s\n", absWS)

		if len(resp.ValidationResult.MissingFields) > 0 {
			fmt.Println("\n🔍 程式檢驗缺漏欄位 (Node E):")
			for _, mf := range resp.ValidationResult.MissingFields {
				fmt.Printf("  - %s\n", mf)
			}
		}

		if len(resp.ClarificationRequests) > 0 {
			fmt.Printf("\n❓ 待釐清關鍵事項 (Node G / ASK_HUMAN) [%d 項]:\n", len(resp.ClarificationRequests))
			for i, cr := range resp.ClarificationRequests {
				fmt.Printf("  [%d] %s (%s)\n      問題: %s\n      成因: %s\n", i+1, cr.ID, cr.Category, cr.Question, cr.Rationale)
				if len(cr.Options) > 0 {
					fmt.Println("      建議選項:")
					for _, opt := range cr.Options {
						fmt.Printf("        • %s\n", opt)
					}
				}
			}
			fmt.Println("\n👉 請使用 `aia writing clarify answer <id> --answer \"...\"` 進行回覆。")
		} else {
			fmt.Println("\n✅ 創作綱要完整無缺口，已自動建立創作定位與全書大綱 (Node K)！")
		}
		return nil
	},
}

var writingBriefCmd = &cobra.Command{
	Use:   "brief [project-id]",
	Short: "View the creative brief, scale, and validation status",
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

		brief, err := appInst.WritingWorkflowRepo.GetBriefByProjectID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("failed to get brief: %w", err)
		}
		if brief == nil {
			return fmt.Errorf("no creative brief found for project %s (run `aia writing init` first)", targetID)
		}

		fmt.Printf("=== Creative Brief: %s (%s) ===\n", brief.Title, brief.ProjectID)
		fmt.Printf("狀態:             %s\n", brief.Status)
		fmt.Printf("類型:             %s\n", brief.Genre)
		if brief.Era != "" {
			fmt.Printf("年代背景:         %s\n", brief.Era)
		}
		fmt.Printf("篇幅規模:         %d 卷共 %d 章 (每卷 %d 章)\n", brief.Scale.TotalVolumes, brief.Scale.TotalChapters, brief.Scale.ChaptersPerVolume)
		fmt.Printf("待釐清事項數:     %d\n", brief.PendingQuestions)
		fmt.Printf("\n--- 原始創作意圖 ---\n%s\n", brief.RawIntent)

		if len(brief.ThemeAndPhilosophy) > 0 {
			fmt.Println("\n--- 核心哲學原則與主旨 ---")
			for _, th := range brief.ThemeAndPhilosophy {
				fmt.Printf("  • %s\n", th)
			}
		}
		if len(brief.CoreProtagonists) > 0 {
			fmt.Println("\n--- 主要人物群像 ---")
			for _, cp := range brief.CoreProtagonists {
				fmt.Printf("  • %s\n", cp)
			}
		}
		return nil
	},
}

var writingClarifyCmd = &cobra.Command{
	Use:   "clarify",
	Short: "Manage clarification requests and interactive Q&A (Node G/H/I/J)",
}

var writingClarifyListCmd = &cobra.Command{
	Use:   "list [project-id]",
	Short: "List clarification requests for a project",
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

		reqs, err := appInst.WritingWorkflowRepo.ListClarificationsByProjectID(ctx, targetID)
		if err != nil {
			return fmt.Errorf("failed to list clarifications: %w", err)
		}
		if len(reqs) == 0 {
			fmt.Printf("No clarification requests found for project '%s'.\n", targetID)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tCATEGORY\tSTATUS\tQUESTION")
		for _, cr := range reqs {
			qShort := cr.Question
			if len(qShort) > 50 {
				qShort = qShort[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cr.ID, cr.Category, cr.Status, qShort)
		}
		_ = w.Flush()
		return nil
	},
}

var writingClarifyAskCmd = &cobra.Command{
	Use:   "ask <clarification-id>",
	Short: "Inspect a single clarification question with rationale and options",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		id := args[0]

		cr, err := appInst.WritingWorkflowRepo.GetClarificationByID(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get clarification: %w", err)
		}
		if cr == nil {
			return fmt.Errorf("clarification request '%s' not found", id)
		}

		fmt.Printf("=== Clarification Request: %s ===\n", cr.ID)
		fmt.Printf("Project:    %s\n", cr.ProjectID)
		fmt.Printf("Category:   %s\n", cr.Category)
		fmt.Printf("Status:     %s\n", cr.Status)
		fmt.Printf("\n問題:       %s\n", cr.Question)
		fmt.Printf("成因說明:   %s\n", cr.Rationale)

		if len(cr.Options) > 0 {
			fmt.Println("\n建議選項:")
			for _, opt := range cr.Options {
				fmt.Printf("  • %s\n", opt)
			}
		}
		if cr.UserAnswer != "" {
			fmt.Printf("\n使用者回答: %s\n", cr.UserAnswer)
		}
		return nil
	},
}

var writingClarifyAnswerCmd = &cobra.Command{
	Use:   "answer <clarification-id>",
	Short: "Answer a clarification request and update Brief/Memory (Node I ➔ J ➔ D)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		id := args[0]

		if writingAnswer == "" {
			return fmt.Errorf("--answer is required")
		}

		resp, err := appInst.WritingWorkflowSvc.AnswerClarification(ctx, id, writingAnswer, "")
		if err != nil {
			return fmt.Errorf("failed to record clarification answer: %w", err)
		}

		fmt.Printf("✓ Answer recorded for %s.\n", id)
		fmt.Printf("Current Brief Status: %s\n", resp.Brief.Status)
		fmt.Printf("Pending Questions:    %d\n", len(resp.RemainingPending))

		if resp.OutlineEstablished {
			fmt.Println("\n🎉 全部關鍵問題已釐清！系統已自動生成全書大綱與定位 (Node K):")
			fmt.Println("  - book.yaml (專案配置)")
			fmt.Println("  - outline.md (全書總綱與分卷)")
			fmt.Println("  - master_bible.md (寫作指南與防雷規範)")
			fmt.Println("  - characters.md (人物群像與年表約束)")
		} else if resp.AllClarified {
			fmt.Println("\n✅ 當前問題皆已回覆，Brief 驗證完畢！")
		} else {
			fmt.Printf("\n👉 尚有 %d 個待釐清事項，請繼續使用 `aia writing clarify list` 查看。\n", len(resp.RemainingPending))
		}
		return nil
	},
}

var writingOutlineCmd = &cobra.Command{
	Use:   "outline [project-id]",
	Short: "Establish or refresh full-book outline and master bible from approved Brief (Node K)",
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

		if err := appInst.WritingWorkflowSvc.EstablishCreativeOutline(ctx, targetID, ""); err != nil {
			return fmt.Errorf("failed to establish outline: %w", err)
		}

		fmt.Printf("✓ Successfully established outline and master bible for project '%s'.\n", targetID)
		return nil
	},
}

func init() {
	writingRegisterCmd.Flags().StringVar(&writingBookName, "name", "", "novel project name")
	writingRegisterCmd.Flags().StringVar(&writingBookPath, "path", "", "novel directory path (default: current directory)")

	writingInitCmd.Flags().StringVarP(&writingIntent, "intent", "i", "", "creative intent statement (必填)")
	writingInitCmd.Flags().StringVarP(&writingBookName, "title", "t", "", "book/project title")
	writingInitCmd.Flags().StringVarP(&writingGenre, "genre", "g", "historical_fiction", "genre / profile")
	writingInitCmd.Flags().StringVarP(&writingEra, "era", "e", "", "historical era")
	writingInitCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "project ID")

	writingBriefCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "project ID")
	writingClarifyListCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "project ID")
	writingClarifyAnswerCmd.Flags().StringVarP(&writingAnswer, "answer", "a", "", "user's clarification response (必填)")
	writingOutlineCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "project ID")

	writingScanCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "target project ID")
	writingManifestCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "target project ID")
	writingPromptCmd.Flags().StringVarP(&writingProjectID, "project", "p", "", "target project ID")

	writingClarifyCmd.AddCommand(writingClarifyListCmd)
	writingClarifyCmd.AddCommand(writingClarifyAskCmd)
	writingClarifyCmd.AddCommand(writingClarifyAnswerCmd)

	writingCmd.AddCommand(writingInitCmd)
	writingCmd.AddCommand(writingBriefCmd)
	writingCmd.AddCommand(writingClarifyCmd)
	writingCmd.AddCommand(writingOutlineCmd)
	writingCmd.AddCommand(writingBooksCmd)
	writingCmd.AddCommand(writingRegisterCmd)
	writingCmd.AddCommand(writingRemoveCmd)
	writingCmd.AddCommand(writingScanCmd)
	writingCmd.AddCommand(writingManifestCmd)
	writingCmd.AddCommand(writingPromptCmd)
	writingCmd.AddCommand(writingInfoCmd)
}
