package writing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/memory"
	domainProject "github.com/wade/aia/internal/domain/project"
	domainTask "github.com/wade/aia/internal/domain/task"
	domainWriting "github.com/wade/aia/internal/domain/writing"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// InitiateProjectRequest holds input parameters to begin a new writing project.
type InitiateProjectRequest struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id,omitempty"`
	Title     string `json:"title"`
	RawIntent string `json:"raw_intent"`
	Genre     string `json:"genre"`
	Era       string `json:"era,omitempty"`
	Workspace string `json:"workspace"`
}

// InitiateProjectResponse returns the initialized brief and gap analysis findings.
type InitiateProjectResponse struct {
	Brief                 *domainWriting.CreativeBrief
	ValidationResult      domainWriting.BriefValidationResult
	ClarificationRequests []*domainWriting.ClarificationRequest
	Status                domainWriting.BriefStatus
}

// AnswerClarificationResponse encapsulates the outcome of answering a clarification question.
type AnswerClarificationResponse struct {
	Brief                 *domainWriting.CreativeBrief
	AnsweredClarification *domainWriting.ClarificationRequest
	RemainingPending      []*domainWriting.ClarificationRequest
	AllClarified          bool
	OutlineEstablished    bool
}

// WorkflowService coordinates the writing inception, clarification, outline, and consistency cycle.
type WorkflowService struct {
	workflowRepo *repository.WritingWorkflowRepository
	projectRepo  *repository.ProjectRepository
	memoryRepo   *repository.MemoryRepository
	taskRepo     *repository.TaskRepository
	gapAnalyzer  *GapAnalyzer
}

func NewWorkflowService(
	workflowRepo *repository.WritingWorkflowRepository,
	projectRepo *repository.ProjectRepository,
	memoryRepo *repository.MemoryRepository,
	gapAnalyzer *GapAnalyzer,
) *WorkflowService {
	return &WorkflowService{
		workflowRepo: workflowRepo,
		projectRepo:  projectRepo,
		memoryRepo:   memoryRepo,
		gapAnalyzer:  gapAnalyzer,
	}
}

// SetTaskRepository sets the task repository for managing WAITING_FOR_INPUT and task resumption.
func (s *WorkflowService) SetTaskRepository(tr *repository.TaskRepository) {
	s.taskRepo = tr
}

// InitiateProject starts the creative workflow (Nodes A, B, C, D, E, F).
func (s *WorkflowService) InitiateProject(ctx context.Context, req InitiateProjectRequest) (*InitiateProjectResponse, error) {
	if strings.TrimSpace(req.ProjectID) == "" {
		req.ProjectID = "proj-writing-" + uuid.New().String()[:8]
	}
	if strings.TrimSpace(req.Title) == "" {
		req.Title = filepath.Base(req.Workspace)
		if req.Title == "" || req.Title == "." {
			req.Title = "新創作專案"
		}
	}
	if strings.TrimSpace(req.Genre) == "" {
		req.Genre = "historical_fiction"
	}

	// 確保 Workspace 目錄存在
	if req.Workspace != "" {
		if err := os.MkdirAll(req.Workspace, 0755); err != nil {
			return nil, fmt.Errorf("failed to create project workspace directory: %w", err)
		}
	}

	// 確保 Project 記錄存在 (Node B)
	if s.projectRepo != nil {
		p, _ := s.projectRepo.Get(ctx, domain.ID(req.ProjectID))
		if p == nil {
			newProj := &domainProject.Project{
				ID:          domain.ID(req.ProjectID),
				Name:        req.Title,
				Description: req.RawIntent,
				Workspace:   req.Workspace,
				DomainType:  "writing",
				Status:      domainProject.StatusActive,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			}
			_ = s.projectRepo.Create(ctx, newProj)
		}
	}

	// 檢查是否已有既有 Brief
	brief, err := s.workflowRepo.GetBriefByProjectID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing brief: %w", err)
	}

	if brief == nil {
		brief = &domainWriting.CreativeBrief{
			ID:                 "brief-" + uuid.New().String(),
			ProjectID:          req.ProjectID,
			Title:              req.Title,
			RawIntent:          req.RawIntent,
			Genre:              req.Genre,
			Era:                req.Era,
			ThemeAndPhilosophy: make([]string, 0),
			Scale: domainWriting.BookScale{
				TotalVolumes:      0,
				TotalChapters:     0,
				ChaptersPerVolume: 0,
				VolumeStructure:   "files",
			},
			CoreProtagonists:  make([]string, 0),
			WorldviewSettings: make(map[string]any),
			WritingRules: []string{
				"【史實骨架】：無可爭辯的正史記載，不可竄改歷史人物生卒與大勢抉擇",
				"【合理推演】：符合時代民俗、官制與地理行程之推論",
				"【小說虛構】：藝術創作、基層原創人物與方外穿線",
				"防時代錯置（Anti-Anachronism）：嚴禁提早使用後代年號或名詞",
				"防空間瞬移（Anti-Teleportation）：行程時間必須符合古代交通限制",
			},
			Status:           domainWriting.BriefStatusDraft,
			PendingQuestions: 0,
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}
		// 智慧初探：若意圖中已提及知名歷史人物，自動預先收錄
		for _, fig := range []string{"脫脫", "朱元璋", "黃巢", "李克用", "朱溫", "岳飛", "文天祥", "于謙", "賈魯"} {
			if strings.Contains(req.RawIntent, fig) {
				brief.CoreProtagonists = append(brief.CoreProtagonists, fig)
			}
		}
		if err := s.workflowRepo.CreateBrief(ctx, brief); err != nil {
			return nil, fmt.Errorf("failed to create creative brief: %w", err)
		}
	}

	// 執行 Node D & Node E 分析
	outcome, err := s.gapAnalyzer.AnalyzeIntentAndBrief(ctx, brief)
	if err != nil {
		return nil, fmt.Errorf("gap analysis failed: %w", err)
	}

	// Node F: 是否有重要待釐清事項？
	if outcome.RequiresHumanClarify {
		// Node G & H: 建立 Clarification Requests 並設定 UNDER_CLARIFICATION
		brief.Status = domainWriting.BriefStatusUnderClarification
		brief.PendingQuestions = len(outcome.ClarificationRequests)

		for _, cr := range outcome.ClarificationRequests {
			_ = s.workflowRepo.CreateClarification(ctx, cr)
		}
		_ = s.workflowRepo.UpdateBrief(ctx, brief)

		// Transition associated task to WAITING_FOR_INPUT (AIA-21 §7.1, §7.2)
		if s.taskRepo != nil {
			if req.TaskID != "" {
				_ = s.taskRepo.UpdateStatus(ctx, domain.ID(req.TaskID), domainTask.StatusWaitingInput)
			} else {
				tasks, _ := s.taskRepo.List(ctx, repository.TaskFilter{
					ProjectID: req.ProjectID,
					Status:    domainTask.StatusRunning,
				})
				for _, t := range tasks {
					_ = s.taskRepo.UpdateStatus(ctx, t.ID, domainTask.StatusWaitingInput)
				}
			}
		}
	} else {
		// Node F: 沒有 ➔ 推進至 APPROVED
		brief.Status = domainWriting.BriefStatusApproved
		brief.PendingQuestions = 0
		_ = s.workflowRepo.UpdateBrief(ctx, brief)

		// Node K: 建立創作定位與全書大綱
		_ = s.EstablishCreativeOutline(ctx, req.ProjectID, req.Workspace)
	}

	// 同步至本地 brief.md
	if req.Workspace != "" {
		_ = os.WriteFile(filepath.Join(req.Workspace, "brief.md"), []byte(brief.ToMarkdown()), 0644)
	}

	return &InitiateProjectResponse{
		Brief:                 brief,
		ValidationResult:      outcome.ValidationResult,
		ClarificationRequests: outcome.ClarificationRequests,
		Status:                brief.Status,
	}, nil
}

// AnswerClarification handles Node I & Node J (使用者回答並更新 Brief / Project Memory).
func (s *WorkflowService) AnswerClarification(ctx context.Context, clarificationID string, answer string, workspace string) (*AnswerClarificationResponse, error) {
	c, err := s.workflowRepo.GetClarificationByID(ctx, clarificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch clarification: %w", err)
	}
	if c == nil {
		return nil, fmt.Errorf("clarification request %s not found", clarificationID)
	}

	now := time.Now().UTC()
	c.UserAnswer = strings.TrimSpace(answer)
	c.Status = domainWriting.ClarificationStatusAnswered
	c.AnsweredAt = &now

	if err := s.workflowRepo.UpdateClarification(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to update clarification: %w", err)
	}

	// 載入 Brief
	brief, err := s.workflowRepo.GetBriefByID(ctx, c.BriefID)
	if err != nil || brief == nil {
		brief, err = s.workflowRepo.GetBriefByProjectID(ctx, c.ProjectID)
		if err != nil || brief == nil {
			return nil, fmt.Errorf("brief associated with project %s not found", c.ProjectID)
		}
	}

	// Node J: 將回答回填進 Brief 模型
	s.applyAnswerToBrief(brief, c)

	// 寫入 Project Memory (L1~L5)
	if s.memoryRepo != nil {
		mem := &memory.Memory{
			ID:          domain.ID("mem-clarify-" + uuid.New().String()[:8]),
			Type:        memory.TypeDecision,
			Scope:       memory.ScopeProject,
			ProjectID:   domain.ID(c.ProjectID),
			Title:       fmt.Sprintf("創作釐清決策: %s", c.Question),
			Content:     fmt.Sprintf("問題: %s\n成因: %s\n回覆採納: %s", c.Question, c.Rationale, c.UserAnswer),
			Confidence:  1.0,
			Status:      "ACTIVE",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		_ = s.memoryRepo.Create(ctx, mem)
	}

	// 查詢是否尚有未決問題
	pending, err := s.workflowRepo.ListPendingClarifications(ctx, c.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending clarifications: %w", err)
	}

	allClarified := len(pending) == 0
	outlineEstablished := false

	if allClarified {
		// 重新走 Node D & Node E 驗證是否全部滿足
		outcome, err := s.gapAnalyzer.AnalyzeIntentAndBrief(ctx, brief)
		if err == nil && !outcome.RequiresHumanClarify {
			brief.Status = domainWriting.BriefStatusApproved
			brief.PendingQuestions = 0
			_ = s.workflowRepo.UpdateBrief(ctx, brief)

			// Node K: 建立創作定位與全書大綱
			if workspace == "" && s.projectRepo != nil {
				if p, _ := s.projectRepo.Get(ctx, domain.ID(brief.ProjectID)); p != nil {
					workspace = p.Workspace
				}
			}
			if err := s.EstablishCreativeOutline(ctx, brief.ProjectID, workspace); err == nil {
				outlineEstablished = true
			}

			// Resume paused tasks: transition WAITING_FOR_INPUT to READY for continuation (AIA-21 §7.4)
			if s.taskRepo != nil {
				waitingTasks, _ := s.taskRepo.List(ctx, repository.TaskFilter{
					ProjectID: brief.ProjectID,
					Status:    domainTask.StatusWaitingInput,
				})
				for _, wt := range waitingTasks {
					_ = s.taskRepo.UpdateStatus(ctx, wt.ID, domainTask.StatusReady)
				}
			}
		} else if outcome != nil && len(outcome.ClarificationRequests) > 0 {
			allClarified = false
			brief.PendingQuestions = len(outcome.ClarificationRequests)
			for _, newCr := range outcome.ClarificationRequests {
				_ = s.workflowRepo.CreateClarification(ctx, newCr)
			}
			_ = s.workflowRepo.UpdateBrief(ctx, brief)
			pending = outcome.ClarificationRequests
		}
	} else {
		brief.PendingQuestions = len(pending)
		_ = s.workflowRepo.UpdateBrief(ctx, brief)
	}

	// 持久化更新本地 brief.md
	if workspace != "" {
		_ = os.WriteFile(filepath.Join(workspace, "brief.md"), []byte(brief.ToMarkdown()), 0644)
	}

	return &AnswerClarificationResponse{
		Brief:                 brief,
		AnsweredClarification: c,
		RemainingPending:      pending,
		AllClarified:          allClarified,
		OutlineEstablished:    outlineEstablished,
	}, nil
}

// EstablishCreativeOutline performs Node K (建立創作定位與全書大綱).
func (s *WorkflowService) EstablishCreativeOutline(ctx context.Context, projectID string, workspace string) error {
	brief, err := s.workflowRepo.GetBriefByProjectID(ctx, projectID)
	if err != nil || brief == nil {
		return fmt.Errorf("failed to load brief for outline generation: %w", err)
	}

	if workspace == "" && s.projectRepo != nil {
		if p, _ := s.projectRepo.Get(ctx, domain.ID(projectID)); p != nil {
			workspace = p.Workspace
		}
	}
	if workspace == "" {
		return fmt.Errorf("workspace path required for project %s", projectID)
	}

	// 1. 生成 book.yaml (BookManifest 規格)
	manifest := s.generateBookManifest(brief)
	yamlBytes, err := yaml.Marshal(manifest)
	if err == nil {
		_ = os.WriteFile(filepath.Join(workspace, "book.yaml"), yamlBytes, 0644)
	}

	// 2. 生成 outline.md (全書總綱)
	outlineMD := s.generateOutlineMarkdown(brief)
	_ = os.WriteFile(filepath.Join(workspace, "outline.md"), []byte(outlineMD), 0644)

	// 3. 生成 master_bible.md (寫作指南與防雷規範)
	bibleMD := s.generateMasterBibleMarkdown(brief)
	_ = os.WriteFile(filepath.Join(workspace, "master_bible.md"), []byte(bibleMD), 0644)

	// 4. 生成 characters.md (人物清單骨架)
	charsMD := s.generateCharactersMarkdown(brief)
	_ = os.WriteFile(filepath.Join(workspace, "characters.md"), []byte(charsMD), 0644)

	// 5. 確保 chapters 資料夾存在
	_ = os.MkdirAll(filepath.Join(workspace, "chapters"), 0755)

	return nil
}

func (s *WorkflowService) applyAnswerToBrief(b *domainWriting.CreativeBrief, c *domainWriting.ClarificationRequest) {
	ans := c.UserAnswer

	switch c.Category {
	case domainWriting.CategoryScale:
		// 解析如 "6 卷共 540 章"
		reVol := regexp.MustCompile(`(\d+)\s*卷`)
		reChap := regexp.MustCompile(`(\d+)\s*章`)
		matchVol := reVol.FindStringSubmatch(ans)
		matchChap := reChap.FindStringSubmatch(ans)

		if len(matchVol) >= 2 {
			if v, err := strconv.Atoi(matchVol[1]); err == nil && v > 0 {
				b.Scale.TotalVolumes = v
			}
		}
		if len(matchChap) >= 2 {
			if ch, err := strconv.Atoi(matchChap[1]); err == nil && ch > 0 {
				b.Scale.TotalChapters = ch
			}
		}
		if b.Scale.TotalVolumes > 0 && b.Scale.TotalChapters > 0 {
			b.Scale.ChaptersPerVolume = b.Scale.TotalChapters / b.Scale.TotalVolumes
		} else if b.Scale.TotalVolumes > 0 && b.Scale.ChaptersPerVolume == 0 {
			b.Scale.ChaptersPerVolume = 90
			b.Scale.TotalChapters = b.Scale.TotalVolumes * 90
		}

	case domainWriting.CategoryPlotPolicy:
		if strings.Contains(ans, "元末") || strings.Contains(ans, "唐末") || strings.Contains(ans, "宋") || strings.Contains(ans, "明") {
			b.Era = ans
		}
		b.WritingRules = append(b.WritingRules, "作者裁定："+ans)

	case domainWriting.CategoryWorldview:
		if b.WorldviewSettings == nil {
			b.WorldviewSettings = make(map[string]any)
		}
		b.WorldviewSettings["policy"] = ans
		if strings.Contains(ans, "不干預") {
			b.WorldviewSettings["non_interference"] = true
		}
		b.ThemeAndPhilosophy = append(b.ThemeAndPhilosophy, ans)

	case domainWriting.CategoryCharacter:
		b.CoreProtagonists = append(b.CoreProtagonists, ans)

	default:
		b.ThemeAndPhilosophy = append(b.ThemeAndPhilosophy, ans)
	}

	b.UpdatedAt = time.Now().UTC()
	_ = s.workflowRepo.UpdateBrief(context.Background(), b)
}

func (s *WorkflowService) generateBookManifest(b *domainWriting.CreativeBrief) *domainWriting.BookManifest {
	totalVols := b.Scale.TotalVolumes
	if totalVols <= 0 {
		totalVols = 6
	}
	totalChaps := b.Scale.TotalChapters
	if totalChaps <= 0 {
		totalChaps = totalVols * 90
	}
	chapsPerVol := b.Scale.ChaptersPerVolume
	if chapsPerVol <= 0 {
		chapsPerVol = 90
	}

	manifest := &domainWriting.BookManifest{
		Project: domainWriting.ProjectMeta{
			ID:          b.ProjectID,
			Name:        b.Title,
			Type:        b.Genre,
			DomainAgent: "writing",
			Language:    "zh-TW",
			Era:         b.Era,
			Status:      "planning",
		},
		Scale: &domainWriting.BookScale{
			TotalVolumes:      totalVols,
			TotalChapters:     totalChaps,
			ChaptersPerVolume: chapsPerVol,
			VolumeStructure:   "files",
		},
		Structure: domainWriting.BookStructure{
			Outline:             "outline.md",
			VolumesPlan:         "volumes.md",
			Volumes:             ".",
			Chapters:            "chapters/",
			Timeline:            "timeline.md",
			ChapterLedger:       "chapter_ledger.md",
			Characters:          "characters.md",
			MasterBible:         "master_bible.md",
			Rules:               "chapter_writer_rules.md",
		},
		WritingPolicy: domainWriting.BookWritingPolicy{
			HistoricalAccuracy: "strict",
			BattleTacticsFocus: false,
			Classifications: []string{
				"historical_fact",
				"reasonable_inference",
				"fictional_content",
			},
			PreserveHistoricalFigures: true,
			RequireConsistencyCheck:   true,
			AntiAnachronism:           true,
			AntiTeleportation:         true,
			InformationBoundary:       true,
		},
	}

	if strings.Contains(strings.ToLower(b.Genre), "dual_world") || strings.Contains(strings.ToLower(b.Genre), "laboratory") {
		manifest.Laboratory = &domainWriting.LaboratoryConfig{
			Enabled:         true,
			Perspective:     "外星文明人類觀測實驗室",
			Specification:   "laboratory.md",
			NonInterference: true,
		}
	}

	return manifest
}

func (s *WorkflowService) generateOutlineMarkdown(b *domainWriting.CreativeBrief) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 《%s》全書總綱與分卷大綱 (Outline)\n\n", b.Title))
	sb.WriteString(fmt.Sprintf("- 時代背景：%s\n", b.Era))
	sb.WriteString(fmt.Sprintf("- 總體規模：%d 卷共 %d 章\n\n", b.Scale.TotalVolumes, b.Scale.TotalChapters))

	sb.WriteString("## 核心衝突與主旨\n\n")
	for _, theme := range b.ThemeAndPhilosophy {
		sb.WriteString(fmt.Sprintf("- %s\n", theme))
	}
	sb.WriteString("\n## 分卷規劃 (Volume Layout)\n\n")

	vols := b.Scale.TotalVolumes
	if vols <= 0 {
		vols = 6
	}
	for v := 1; v <= vols; v++ {
		startCh := (v-1)*90 + 1
		endCh := v * 90
		sb.WriteString(fmt.Sprintf("### 第 %d 卷：第 %d 章 至 第 %d 章\n", v, startCh, endCh))
		sb.WriteString("- **核心年表與歷史錨點**：*(待分段研究填入)*\n")
		sb.WriteString("- **主要登場角色**：*(參閱 characters.md)*\n\n")
	}

	return sb.String()
}

func (s *WorkflowService) generateMasterBibleMarkdown(b *domainWriting.CreativeBrief) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 《%s》寫作總指南 (Master Bible)\n\n", b.Title))
	sb.WriteString("## 1. 寫作總則\n\n")
	sb.WriteString("- **非戰場中心原則**：歷史的轉折發生在制度與人心，不在單一戰役的得失。\n")
	sb.WriteString("- **史學三層標記**：\n")
	sb.WriteString("  - 【史實骨架】：正史記載，不可竄改重大生死與歷史走向。\n")
	sb.WriteString("  - 【合理推演】：符合時代民俗與行程天數之補白。\n")
	sb.WriteString("  - 【小說虛構】：原創基層人物與觀測穿線。\n\n")
	sb.WriteString("## 2. 嚴禁事項 (Hard Constraints)\n\n")
	sb.WriteString("- 嚴禁時代錯置（Anti-Anachronism）。\n")
	sb.WriteString("- 嚴禁空間瞬移（Anti-Teleportation）。\n")
	sb.WriteString("- 歷史人物決策不得超越當時當地的情報邊界。\n\n")
	return sb.String()
}

func (s *WorkflowService) generateCharactersMarkdown(b *domainWriting.CreativeBrief) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 《%s》人物總表 (Characters)\n\n", b.Title))

	// 依據時代與主角群生成符合 ConsistencyEngine 規格的 ## 區段
	if strings.Contains(b.Era, "元") {
		sb.WriteString("## 脫脫\n- 延祐元年（1314）至至正十五年（1355），元末右丞相 【史實骨架】\n\n")
		sb.WriteString("## 朱元璋\n- 天曆元年（1328）至洪武三十一年（1398），明太祖 【史實骨架】\n\n")
		sb.WriteString("## 賈魯\n- 元貞三年（1297）至至正十三年（1353），工部尚書、治河名臣 【史實骨架】\n\n")
	}

	for _, p := range b.CoreProtagonists {
		if p != "脫脫" && p != "朱元璋" && p != "賈魯" {
			sb.WriteString(fmt.Sprintf("## %s\n- 主要視角角色 【小說虛構】\n\n", p))
		}
	}
	return sb.String()
}
