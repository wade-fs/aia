package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/domain/profile"
	"github.com/wade/aia/internal/domain/task"
	"github.com/wade/aia/internal/memory/retrieval"
	"github.com/wade/aia/internal/policy"
	"github.com/wade/aia/internal/provider"
	taskService "github.com/wade/aia/internal/service/task"
)

type UserInput struct {
	Content        string
	ProjectID      domain.ID
	ConversationID string
}

type AgentResponse struct {
	Content        string
	TasksCreated   []domain.ID
	ConversationID string
	RequiresApproval bool
}

type PersonalAgent struct {
	prov       provider.Provider
	retriever  *retrieval.Retriever
	taskSvc    *taskService.Service
	scheduler  *Scheduler
	evaluator  *policy.Evaluator
}

func NewPersonalAgent(
	prov provider.Provider,
	retriever *retrieval.Retriever,
	taskSvc *taskService.Service,
	scheduler *Scheduler,
	evaluator *policy.Evaluator,
) *PersonalAgent {
	return &PersonalAgent{
		prov:      prov,
		retriever: retriever,
		taskSvc:   taskSvc,
		scheduler: scheduler,
		evaluator: evaluator,
	}
}

func (pa *PersonalAgent) HandleInput(ctx context.Context, input UserInput) (*AgentResponse, error) {
	// 1. Retrieve relevant memories
	memories, _ := pa.retriever.Retrieve(ctx, input.Content, input.ProjectID, "personal-agent")
	contextPrompt := retrieval.FormatContext(memories)

	// 2. Classify intent: Actionable task vs Q&A
	isActionable := isActionableTask(input.Content)

	if !isActionable {
		// Conversational / Question response
		fullPrompt := input.Content
		if contextPrompt != "" {
			fullPrompt = contextPrompt + "\nUser Question: " + input.Content
		}

		session, err := pa.prov.Start(ctx, provider.StartRequest{
			ExecutionID:    "chat",
			Agent:          agent.Agent{Name: "personal-agent"},
			Prompt:         fullPrompt,
			ConversationID: input.ConversationID,
			Profile: profile.ExecutionProfile{
				Sandbox: true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start conversation: %w", err)
		}

		res, err := session.Wait(ctx)
		if err != nil {
			return nil, err
		}

		return &AgentResponse{
			Content:        res.Response,
			ConversationID: res.ConversationID,
		}, nil
	}

	// 3. Actionable Task Planning
	risk := assessRisk(input.Content)
	eval := pa.evaluator.EvaluateRisk(ctx, risk)
	if eval.Decision == policy.DecisionDeny {
		return &AgentResponse{
			Content: fmt.Sprintf("⚠️ Action rejected by safety policy: %s", eval.Reason),
		}, nil
	}

	assignedAgent := selectAgentForTask(input.Content)

	// Create Parent Root Task
	rootTask, err := pa.taskSvc.Create(ctx, taskService.CreateTaskRequest{
		ProjectID:   input.ProjectID,
		AgentID:     assignedAgent,
		Title:       deriveTitle(input.Content),
		Description: input.Content,
		Priority:    50,
		Risk:        risk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create planned task: %w", err)
	}

	if eval.Decision == policy.DecisionAsk {
		_ = pa.taskSvc.UpdateStatus(ctx, rootTask.ID, task.StatusWaitingApproval)
		return &AgentResponse{
			Content:          fmt.Sprintf("Task created (%s) but paused: %s. Please approve to execute.", rootTask.ID, eval.Reason),
			TasksCreated:     []domain.ID{rootTask.ID},
			RequiresApproval: true,
		}, nil
	}

	// Execute via scheduler
	go func() {
		_ = pa.scheduler.RunTask(context.Background(), rootTask.ID)
	}()

	return &AgentResponse{
		Content:      fmt.Sprintf("Plan created and dispatched.\nTask ID: %s\nAssigned Agent: %s\nStatus: RUNNING", rootTask.ID, assignedAgent),
		TasksCreated: []domain.ID{rootTask.ID},
	}, nil
}

func isActionableTask(input string) bool {
	lower := strings.ToLower(input)
	triggers := []string{
		"實作", "建立", "新增", "修改", "開發", "產生", "撰寫", "測試", "修復",
		"create", "build", "implement", "add", "fix", "write", "generate", "run", "refactor",
	}
	for _, tr := range triggers {
		if strings.Contains(lower, tr) {
			return true
		}
	}
	return false
}

func selectAgentForTask(input string) string {
	lower := strings.ToLower(input)
	if strings.Contains(lower, "程式") || strings.Contains(lower, "code") || strings.Contains(lower, "go") || strings.Contains(lower, "bug") || strings.Contains(lower, "test") {
		return "coding-agent"
	}
	if strings.Contains(lower, "文章") || strings.Contains(lower, "寫作") || strings.Contains(lower, "文案") || strings.Contains(lower, "write") {
		return "writing-agent"
	}
	if strings.Contains(lower, "投資") || strings.Contains(lower, "股票") || strings.Contains(lower, "財報") || strings.Contains(lower, "invest") {
		return "investment-agent"
	}
	if strings.Contains(lower, "影片") || strings.Contains(lower, "剪輯") || strings.Contains(lower, "分鏡") || strings.Contains(lower, "video") {
		return "video-agent"
	}
	return "coding-agent"
}

func assessRisk(input string) domain.RiskLevel {
	lower := strings.ToLower(input)
	if strings.Contains(lower, "rm -rf /") || strings.Contains(lower, "format disk") {
		return domain.RiskCritical
	}
	if strings.Contains(lower, "刪除資料庫") || strings.Contains(lower, "drop database") || strings.Contains(lower, "deploy prod") {
		return domain.RiskHigh
	}
	return domain.RiskLow
}

func deriveTitle(input string) string {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	firstLine := strings.TrimSpace(lines[0])
	runes := []rune(firstLine)
	if len(runes) > 40 {
		return string(runes[:40]) + "..."
	}
	return firstLine
}
