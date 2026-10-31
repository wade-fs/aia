# AIA-03：AIA Personal Agent Go Runtime 完整實作規格書

**文件代號：** AIA-03  
**文件版本：** 2.0  
**文件狀態：** Implementation Baseline  
**適用專案：** AIA — Autonomous Intelligence Architecture  
**主要實作語言：** Go  
**初期執行平台：** Linux Desktop / Server  
**主要 AI Provider：** Antigravity CLI（agy）  
**資料庫：** SQLite  
**文件性質：** Go Runtime 完整實作規格

---

# 1. 文件目的

本文件定義 AIA Personal Agent Runtime 的完整實作規格。

AIA 是一套可長期運作、具備自主任務管理、Agent 協作、記憶管理、工具使用、排程、權限控制及失敗恢復能力的個人 AI Agent 平台。

本文件的目的，是讓開發者可以依照規格直接建立可編譯、可執行、可測試、可持續擴充的 Go 專案。

AIA 不等同於單一 LLM，也不等同於 Antigravity CLI。

AIA Runtime 負責整體系統管理；AI Provider 負責提供 AI 推理及 Agent Harness 能力；Domain Agent 負責特定領域的工作邏輯。

## 1.1 核心設計原則

1. AIA Runtime 擁有全域任務及執行生命週期。
2. Domain Agent 是可註冊、可替換、可擴充的工作單元。
3. AI Provider 不應成為 AIA 的核心依賴。
4. 所有長時間工作都必須具備狀態保存及恢復能力。
5. 所有外部操作都必須經過權限及政策檢查。
6. 任務完成必須經過結果驗證，不可只依賴 Provider 的成功回應。
7. 記憶資料必須具備來源、時間、可信度及生命週期。
8. 外部 API 必須透過 Adapter 隔離，不得直接散布於 Domain Agent 業務邏輯。
9. Runtime 必須支援排程任務及人工觸發任務。
10. 系統設計應保持語言中立，未來可使用 Kotlin 或其他語言重新實作。

## 1.2 實作範圍

本版本必須完成：

- Go Runtime
- Personal Agent
- Agent Registry
- Domain Agent Extension Contract
- Project Management
- Task Management
- Execution Management
- Scheduler
- Provider Adapter
- Antigravity CLI Adapter
- External Data Adapter Framework
- Policy / Permission Management
- Workspace Management
- Memory Management
- Result Validation
- Recovery
- Audit
- CLI
- SQLite Persistence

## 1.3 非目標

第一階段不要求：

- Android APK
- iOS App
- Web 管理介面
- 多使用者服務
- Kubernetes 部署
- 分散式 Runtime
- 自行開發 LLM
- 自行開發 Agent Harness
- 必須使用 LangGraph、CrewAI 或其他 Agent Framework
- 自動管理 agy OAuth Token
- 自動執行真實金融交易

---

# 2. 系統總體架構

## 2.1 架構圖

```text
                         User
                          │
                          ▼
                    AIA CLI / API
                          │
                          ▼
                ┌──────────────────┐
                │  Personal Agent  │
                │                  │
                │ Intent Analysis  │
                │ Memory Retrieval │
                │ Planning         │
                │ Clarification    │
                └────────┬─────────┘
                         │
                         ▼
                ┌──────────────────┐
                │   Orchestrator   │
                │                  │
                │ Task Management  │
                │ Agent Dispatch  │
                │ Scheduler        │
                │ Recovery         │
                └────────┬─────────┘
                         │
                         ▼
                ┌──────────────────┐
                │  Agent Registry  │
                └────────┬─────────┘
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
      Coding Agent   Writing Agent  Investment Agent
          │              │              │
          └──────────────┼──────────────┘
                         │
                         ▼
                Domain Agent Contract
                         │
             ┌───────────┴───────────┐
             ▼                       ▼
       Provider Adapter       Capability Adapter
             │                       │
             ▼                       ▼
       Antigravity CLI          External APIs
       Other Providers          Files / Tools
             │                       │
             └───────────┬───────────┘
                         ▼
                Result Validation
                         │
                         ▼
                Task / Execution State
                         │
             ┌───────────┼───────────┐
             ▼           ▼           ▼
          SQLite      Memory       Audit
```

## 2.2 架構分層

### Runtime Layer

負責：

- Application Lifecycle
- Task Management
- Execution Management
- Scheduler
- Orchestration
- Recovery
- Agent Registry

### Intelligence Layer

負責：

- Personal Agent
- Domain Agent
- Intent Classification
- Planning
- Decision Making
- Result Interpretation

### Provider Layer

負責：

- Antigravity CLI
- 未來其他 AI Provider
- Provider Session
- Provider Event
- Provider Result

### Capability Layer

負責：

- External Data API
- Filesystem
- HTTP Client
- MCP
- 其他受控工具

### Security Layer

負責：

- Policy
- Permission
- Credential Reference
- Workspace Isolation
- Execution Authorization

### Memory Layer

負責：

- Memory Storage
- Memory Retrieval
- Memory Candidate
- Memory Approval
- Memory Lifecycle

### Persistence Layer

負責：

- SQLite
- Repository
- Database Migration
- Transaction

### Audit Layer

負責：

- Execution Event
- Audit Log
- Decision Record
- Error Record
- System Event

---

# 3. 技術選擇

## 3.1 開發環境

- Go 1.24 或更新穩定版。
- Linux 作為第一階段主要執行環境。
- 支援標準 Go Toolchain。
- 專案必須可在一般 Linux Desktop 執行。

實際 Go 版本應固定於開發環境及 CI 設定，不可使用不受控的自動升級。

## 3.2 建議套件

| 功能 | 技術 |
|---|---|
| CLI | Cobra |
| Configuration | YAML Parser / Viper |
| Database | SQLite |
| SQLite Driver | modernc.org/sqlite |
| UUID | google/uuid |
| Logging | log/slog |
| Time | Go time |
| HTTP | net/http |
| Testing | testing |
| Mock | 自行實作或輕量 Mock |
| Migration | 自行管理 SQL Migration |

依賴套件應保持精簡。

不應為了使用某個框架而引入不必要的 Runtime 依賴。

所有依賴版本固定在 go.mod / go.sum。

## 3.3 設計原則

- 優先使用 Go 標準函式庫。
- Domain Model 不依賴外部套件。
- Repository Interface 與資料庫實作分離。
- Provider Interface 與特定 CLI 分離。
- Domain Agent 不直接操作 SQLite。
- CLI 不直接操作 Repository。
- 外部 API 不直接散布在 Domain Agent 中。
- 所有長時間工作都必須接受 context.Context。
- 不使用 shell 字串拼接執行外部命令。

---

# 4. 專案目錄結構

```text
AIA/
├── README.md
├── LICENSE
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
│
├── configs/
│   ├── aia.example.yaml
│   ├── agents/
│   │   ├── coding.yaml
│   │   ├── writing.yaml
│   │   ├── investment.yaml
│   │   └── video.yaml
│   │
│   ├── profiles/
│   │   ├── read-only.yaml
│   │   ├── research.yaml
│   │   ├── coding.yaml
│   │   └── restricted.yaml
│   │
│   └── schedules/
│       └── investment.yaml
│
├── cmd/
│   └── aia/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── bootstrap.go
│   │   └── shutdown.go
│   │
│   ├── config/
│   │   ├── config.go
│   │   ├── loader.go
│   │   └── validation.go
│   │
│   ├── domain/
│   │   ├── id.go
│   │   ├── agent.go
│   │   ├── project.go
│   │   ├── task.go
│   │   ├── execution.go
│   │   ├── event.go
│   │   ├── schedule.go
│   │   ├── memory.go
│   │   ├── artifact.go
│   │   ├── decision.go
│   │   └── errors.go
│   │
│   ├── orchestrator/
│   │   ├── orchestrator.go
│   │   ├── planner.go
│   │   ├── dispatcher.go
│   │   ├── scheduler.go
│   │   ├── dependency.go
│   │   └── concurrency.go
│   │
│   ├── service/
│   │   ├── personal_agent.go
│   │   ├── task_service.go
│   │   ├── project_service.go
│   │   ├── execution_service.go
│   │   ├── agent_service.go
│   │   ├── schedule_service.go
│   │   └── decision_service.go
│   │
│   ├── agent/
│   │   ├── contract.go
│   │   ├── registry.go
│   │   ├── manifest.go
│   │   ├── context.go
│   │   ├── result.go
│   │   ├── loader.go
│   │   └── builtin/
│   │       ├── coding/
│   │       ├── writing/
│   │       ├── investment/
│   │       └── video/
│   │
│   ├── provider/
│   │   ├── provider.go
│   │   ├── registry.go
│   │   └── antigravity/
│   │       ├── adapter.go
│   │       ├── command.go
│   │       ├── stream.go
│   │       ├── parser.go
│   │       └── errors.go
│   │
│   ├── capability/
│   │   ├── capability.go
│   │   ├── registry.go
│   │   ├── http/
│   │   ├── filesystem/
│   │   └── externaldata/
│   │       ├── provider.go
│   │       ├── registry.go
│   │       ├── request.go
│   │       ├── response.go
│   │       ├── marketdata/
│   │       ├── financialdata/
│   │       └── news/
│   │
│   ├── policy/
│   │   ├── policy.go
│   │   ├── evaluator.go
│   │   └── rules.go
│   │
│   ├── profile/
│   │   ├── profile.go
│   │   ├── loader.go
│   │   └── validator.go
│   │
│   ├── workspace/
│   │   ├── manager.go
│   │   ├── isolation.go
│   │   └── cleanup.go
│   │
│   ├── validation/
│   │   ├── validator.go
│   │   ├── coding.go
│   │   ├── writing.go
│   │   ├── research.go
│   │   └── investment.go
│   │
│   ├── recovery/
│   │   ├── recovery.go
│   │   ├── retry.go
│   │   └── reconciliation.go
│   │
│   ├── audit/
│   │   ├── audit.go
│   │   └── recorder.go
│   │
│   ├── memory/
│   │   ├── manager.go
│   │   ├── retrieval.go
│   │   ├── candidate.go
│   │   └── lifecycle.go
│   │
│   ├── storage/
│   │   ├── sqlite/
│   │   │   ├── db.go
│   │   │   ├── transaction.go
│   │   │   ├── migrations.go
│   │   │   └── repository/
│   │   └── repository.go
│   │
│   └── cli/
│       ├── root.go
│       ├── chat.go
│       ├── task.go
│       ├── project.go
│       ├── agent.go
│       ├── schedule.go
│       ├── execution.go
│       ├── memory.go
│       └── doctor.go
│
├── migrations/
│   ├── 001_initial.sql
│   ├── 002_agent_registry.sql
│   ├── 003_schedules.sql
│   ├── 004_external_data.sql
│   └── 005_decisions.sql
│
├── prompts/
│   ├── personal_agent/
│   ├── planning/
│   ├── validation/
│   └── agents/
│
├── tests/
│   ├── integration/
│   ├── acceptance/
│   ├── provider/
│   └── fixtures/
│
├── data/
├── workspaces/
└── logs/
```

此目錄是邏輯分層，不要求所有空目錄在第一個 Commit 就建立。

---

# 5. Runtime Application Lifecycle

## 5.1 啟動流程

Runtime 啟動時必須依照以下順序：

1. 載入設定。
2. 驗證設定。
3. 建立必要目錄。
4. 初始化 Logger。
5. 開啟 SQLite。
6. 執行 Database Migration。
7. 初始化 Repository。
8. 初始化 Policy Engine。
9. 初始化 Profile Manager。
10. 初始化 Provider Registry。
11. 初始化 Capability Registry。
12. 初始化 Agent Registry。
13. 載入內建 Domain Agent。
14. 初始化 Memory Manager。
15. 初始化 Workspace Manager。
16. 初始化 Recovery Manager。
17. 初始化 Scheduler。
18. 執行未完成 Execution 的 Recovery。
19. 啟動 Scheduler。
20. 啟動 CLI。

不得在 Database 尚未完成 Migration 時啟動 Scheduler。

## 5.2 關閉流程

收到 SIGINT 或 SIGTERM：

1. 停止接收新的任務。
2. 停止觸發新的 Scheduled Task。
3. 通知執行中的 Execution 進入安全停止流程。
4. 等待可正常完成的工作。
5. 保存未完成 Execution 的狀態及事件。
6. 關閉 Provider Process。
7. 關閉 SQLite。
8. Flush Logger。
9. 結束 Runtime。

Graceful Shutdown 必須有最大等待時間。

超過等待時間時，應取消相關 Context，並確保狀態可在下次啟動時恢復。

---

# 6. Configuration

## 6.1 設定檔位置

預設：

```text
~/.config/aia/config.yaml
~/.local/share/aia/
~/.local/state/aia/
~/.local/state/aia/logs/
```

允許透過 CLI 指定設定檔位置。

## 6.2 範例設定

`configs/aia.example.yaml`

```yaml
runtime:
  name: aia
  environment: development
  timezone: Asia/Taipei
  max_concurrent_executions: 2
  graceful_shutdown_timeout: 30s

database:
  driver: sqlite
  path: ~/.local/share/aia/aia.db
  busy_timeout: 5000
  journal_mode: WAL

provider:
  default: antigravity

  antigravity:
    executable: agy
    required_version: "1.4.12"
    input_format: stream-json
    output_format: stream-json
    default_mode: plan
    sandbox: true
    timeout: 30m

agent:
  registry_path: ~/.config/aia/agents
  builtin_enabled: true

workspace:
  root: ~/.local/share/aia/workspaces
  isolate_per_execution: true
  cleanup_after_success: false

memory:
  enabled: true
  retrieval_limit: 10
  use_fts5: true
  require_approval_for_promotion: true

scheduler:
  enabled: true
  poll_interval: 10s
  max_missed_runs: 1

policy:
  default_profile: restricted

logging:
  level: info
  format: json
  path: ~/.local/state/aia/logs/aia.log
```

此設定檔不包含任何 Credential、API Key 或 OAuth Token。

## 6.3 Config Validation

必須檢查：

- 目錄是否可建立。
- SQLite 路徑是否可寫入。
- agy 是否存在。
- agy 版本是否符合需求。
- Provider 設定是否完整。
- Timezone 是否有效。
- 最大並行數是否合理。
- Scheduler 設定是否有效。
- Profile 是否存在。
- Workspace 是否可建立。

設定錯誤時不得以預設值默默覆蓋重要安全設定。

---

# 7. Domain Model

所有 Domain Model 必須與 SQLite、Provider JSON、CLI 輸出格式分離。

## 7.1 ID

所有主要 Entity 使用 UUID。

```go
type ID string

func NewID() ID
```

## 7.2 RiskLevel

```go
type RiskLevel string

const (
    RiskLow      RiskLevel = "LOW"
    RiskMedium   RiskLevel = "MEDIUM"
    RiskHigh     RiskLevel = "HIGH"
    RiskCritical RiskLevel = "CRITICAL"
)
```

## 7.3 Agent

```go
type Agent struct {
    ID          ID
    Name        string
    Description string
    Domain      string
    Version     string
    Enabled     bool
    Manifest    AgentManifest
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 7.4 Project

```go
type Project struct {
    ID          ID
    Name        string
    Description string
    Status      string
    Workspace   string
    Metadata    map[string]any
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 7.5 Task

```go
type Task struct {
    ID          ID
    ProjectID   *ID
    AgentID     *ID
    ParentTaskID *ID

    Title       string
    Description string

    Status      TaskStatus
    Priority    int
    Risk        RiskLevel

    Input       map[string]any
    Result      map[string]any

    ScheduledAt *time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time

    RetryCount  int
    MaxRetries  int

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Task 不代表一次執行。Task 是工作本身的長期紀錄。

## 7.6 TaskStatus

```go
type TaskStatus string

const (
    TaskPending            TaskStatus = "PENDING"
    TaskReady              TaskStatus = "READY"
    TaskRunning            TaskStatus = "RUNNING"
    TaskWaitingApproval    TaskStatus = "WAITING_FOR_APPROVAL"
    TaskWaitingAuth        TaskStatus = "WAITING_FOR_AUTH"
    TaskWaitingDependency  TaskStatus = "WAITING_FOR_DEPENDENCY"
    TaskCompleted          TaskStatus = "COMPLETED"
    TaskFailed             TaskStatus = "FAILED"
    TaskCancelled          TaskStatus = "CANCELLED"
)
```

狀態轉換必須由 Task Service 管理。

不可讓 Provider Adapter 直接修改 Task 狀態。

## 7.7 Execution

```go
type Execution struct {
    ID          ID
    TaskID      ID
    AgentID     ID

    Status      ExecutionStatus

    Provider    string
    ProviderSessionID string

    Workspace   string
    ProfileID   string
    ProfileHash string

    StartedAt   *time.Time
    CompletedAt *time.Time

    ExitCode    *int
    ErrorCode   string
    ErrorMessage string

    RetryOf     *ID

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

每次 Task 執行都建立獨立 Execution。

重新執行時建立新的 Execution，不得覆蓋舊 Execution。

## 7.8 ExecutionStatus

```go
type ExecutionStatus string

const (
    ExecutionCreated     ExecutionStatus = "CREATED"
    ExecutionQueued      ExecutionStatus = "QUEUED"
    ExecutionRunning     ExecutionStatus = "RUNNING"
    ExecutionBlocked     ExecutionStatus = "BLOCKED"
    ExecutionSucceeded   ExecutionStatus = "SUCCEEDED"
    ExecutionFailed      ExecutionStatus = "FAILED"
    ExecutionCancelled   ExecutionStatus = "CANCELLED"
    ExecutionInterrupted ExecutionStatus = "INTERRUPTED"
)
```

## 7.9 ExecutionEvent

```go
type ExecutionEvent struct {
    ID          ID
    ExecutionID ID
    Sequence    int64

    EventType   string
    Source      string

    Payload     map[string]any

    CreatedAt   time.Time
}
```

Execution Event 必須保留完整且可追溯的事件序列。

## 7.10 Artifact

```go
type Artifact struct {
    ID          ID
    TaskID      ID
    ExecutionID ID

    Name        string
    Type        string
    Path        string
    MIMEType    string
    Checksum    string
    Size        int64

    CreatedAt   time.Time
}
```

Artifact 用於保存任務產出的文件、程式碼、報告及其他結果。

## 7.11 Memory

```go
type Memory struct {
    ID          ID

    Type        string
    Scope       string

    Content     string
    Summary     string

    SourceType  string
    SourceID    *ID

    Confidence  float64

    Status      string

    CreatedAt   time.Time
    UpdatedAt   time.Time
    ExpiresAt   *time.Time
}
```

Memory 必須具備來源及生命週期。

## 7.12 Schedule

```go
type Schedule struct {
    ID          ID
    Name        string
    Description string

    AgentID     ID
    ProjectID   *ID

    Enabled     bool

    ScheduleType string
    CronExpr     string
    Timezone     string

    Input        map[string]any

    NextRunAt    *time.Time
    LastRunAt    *time.Time

    MisfirePolicy string

    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

Schedule 是排程定義，不是 Task 或 Execution。

每次排程觸發都應建立新的 Task，並依一般 Task 流程執行。

## 7.13 Investment Decision

```go
type InvestmentDecision struct {
    ID          ID
    TaskID      ID
    AgentID     ID

    Symbol      string
    Decision    string

    ReferencePrice float64
    Currency       string

    Rationale   string
    Risks       []string

    ExpectedPeriod string
    InvalidationConditions []string

    DataSnapshotID *ID

    CreatedAt   time.Time
    ReviewAt    *time.Time
}
```

Investment Decision 用於記錄 Agent 在特定資料時間點所做的判斷。

此 Entity 不代表交易委託，也不具備下單能力。

---

# 8. Agent Architecture

## 8.1 Agent 類型

AIA 至少有兩種 Agent：

### Personal Agent

負責：

- 接收使用者輸入。
- 理解需求。
- 檢索 Memory。
- 判斷任務類型。
- 建立 Plan。
- 分派 Domain Agent。
- 管理跨 Agent 任務。
- 整合結果。
- 向使用者回報。

### Domain Agent

負責特定領域工作。

第一階段預設：

| Domain Agent | 職責 |
|---|---|
| Coding | 程式開發 |
| Writing | 文件與寫作 |
| Investment | 投資研究與建議 |
| Video Production | 影片製作流程 |

Domain Agent 可以使用相同 Provider，也可以在未來使用不同 Provider。

## 8.2 Domain Agent 設計原則

Domain Agent：

- 不直接存取 SQLite。
- 不直接管理全域 Task 狀態。
- 不自行啟動 Scheduler。
- 不直接管理其他 Agent。
- 不自行決定是否突破全域 Policy。
- 不直接存取其他 Agent Workspace。
- 不保存全域 OAuth Credential。
- 不直接操作其他 Agent 的 Execution。

Domain Agent 可以：

- 提供任務分析。
- 提供領域專業指示。
- 產生工作計畫。
- 呼叫允許的 Capability。
- 使用 Provider。
- 產生 Artifact。
- 回報結果及驗證需求。
- 提出 Memory Candidate。
- 提出後續任務建議。

## 8.3 Agent Manifest

每個 Agent 必須提供 Manifest。

範例：

```yaml
id: investment
name: Investment Agent
version: 1.0.0
domain: investment

description: >
  Research US equities and produce investment recommendations.

capabilities:
  - market_data.read
  - financial_data.read
  - news.read
  - portfolio.read

providers:
  - antigravity

risk_level: medium

permissions:
  filesystem: workspace_only
  network: registered_capabilities_only
  external_write: denied

execution:
  max_concurrent: 1
  timeout: 30m

schedule:
  enabled: true
```

Manifest 宣告能力不代表取得實際授權。

所有 Capability 仍必須通過 Policy Engine。

## 8.4 Agent Registry

Agent Registry 負責：

- 註冊 Agent。
- 載入 Agent。
- 查詢 Agent。
- 啟用與停用 Agent。
- 檢查版本。
- 檢查 Manifest。
- 檢查 Capability。
- 管理 Agent Metadata。

Interface：

```go
type AgentRegistry interface {
    Register(ctx context.Context, agent Agent) error
    Get(ctx context.Context, id ID) (Agent, error)
    List(ctx context.Context) ([]Agent, error)
    Enable(ctx context.Context, id ID) error
    Disable(ctx context.Context, id ID) error
}
```

Agent Registry 的持久化資料由 Agent Service 管理。

## 8.5 Domain Agent Contract

Domain Agent 必須透過統一 Interface 與 Runtime 溝通。

```go
type DomainAgent interface {
    Manifest() AgentManifest

    Analyze(
        ctx context.Context,
        input AgentInput,
    ) (*AgentPlan, error)

    Execute(
        ctx context.Context,
        input AgentInput,
    ) (*AgentResult, error)

    Validate(
        ctx context.Context,
        result AgentResult,
    ) (*ValidationReport, error)
}
```

### AgentInput

```go
type AgentInput struct {
    TaskID      ID
    ExecutionID ID
    ProjectID   *ID

    Title       string
    Description string

    Parameters  map[string]any

    Memory      []Memory

    Workspace   string

    Provider    ProviderSession

    Capabilities CapabilityAccess

    Policy      EffectivePolicy
}
```

### AgentPlan

```go
type AgentPlan struct {
    Summary     string
    Steps       []PlanStep

    RequiresApproval bool

    EstimatedRisk RiskLevel
}

type PlanStep struct {
    ID          string
    Description string
    AgentID     *ID

    CapabilityRequirements []string

    DependsOn   []string
}
```

### AgentResult

```go
type AgentResult struct {
    Status      string

    Summary     string

    Output      map[string]any

    Artifacts   []ArtifactInput

    MemoryCandidates []MemoryCandidate

    FollowUpTasks []TaskProposal

    Warnings    []string
    Errors      []string
}
```

### ValidationReport

```go
type ValidationReport struct {
    Passed      bool

    Checks      []ValidationCheck

    Errors      []string
    Warnings    []string
}

type ValidationCheck struct {
    Name        string
    Passed      bool
    Details     string
}
```

## 8.6 Agent 執行流程

```text
Task Created
     │
     ▼
Agent Registry Lookup
     │
     ▼
Agent Analyze
     │
     ▼
Plan Validation
     │
     ▼
Policy Evaluation
     │
     ├── Denied ──> BLOCKED
     │
     ├── Approval Required ──> WAITING_FOR_APPROVAL
     │
     ▼
Execution Created
     │
     ▼
Agent Execute
     │
     ▼
Agent Validate
     │
     ├── Failed ──> Recovery
     │
     ▼
Artifact Persistence
     │
     ▼
Memory Candidate Persistence
     │
     ▼
Task Completion
```

Plan 與 Execute 必須分離。

不可因為 Agent 已產生 Plan，就直接視為工作已完成。

---

# 9. Provider Architecture

## 9.1 Provider 定義

Provider 是 AI 推理及 Agent Harness 的外部執行介面。

第一個 Provider 是 Antigravity CLI。

未來可加入：

- 其他 CLI Agent
- LLM API
- Local Model
- Remote Agent Service

AIA Domain Model 不得依賴 Antigravity JSON 格式。

## 9.2 Provider Interface

```go
type Provider interface {
    Name() string

    CheckAvailability(
        ctx context.Context,
    ) error

    Start(
        ctx context.Context,
        req StartRequest,
    ) (ProviderSession, error)
}
```

```go
type ProviderSession interface {
    ID() string

    Send(
        ctx context.Context,
        input ProviderInput,
    ) error

    Events() <-chan ProviderEvent

    Result(
        ctx context.Context,
    ) (*ProviderResult, error)

    Cancel() error

    Close() error
}
```

## 9.3 StartRequest

```go
type StartRequest struct {
    ExecutionID ID

    Prompt      string

    Workspace   string

    Agent       string
    Model       string
    Effort      string
    Mode        string

    SettingsPath string
    Sandbox      bool

    Timeout      time.Duration
}
```

## 9.4 ProviderEvent

```go
type ProviderEvent struct {
    Type      string
    Timestamp time.Time

    ToolName  string
    ToolInfo  map[string]any

    SubagentInfo map[string]any

    Content   string

    Raw       map[string]any

    Error     *ProviderError
}
```

## 9.5 ProviderResult

```go
type ProviderResult struct {
    Status       string

    Content      string

    ExitCode     int

    SessionID    string

    Usage        TokenUsage

    Error        *ProviderError
}
```

```go
type TokenUsage struct {
    InputTokens  int64
    OutputTokens int64
    TotalTokens  int64
}
```

Provider Result 不可直接決定 Task 最終狀態。

必須由 Execution Service、Validator 及 Orchestrator 綜合判斷。

---

# 10. Antigravity CLI Adapter

## 10.1 基本設定

使用者目前提供的實際安裝版本：

`agy 1.4.12`

此版本的 CLI help 是第一階段整合的基準。

官方文件：

- https://antigravity.google/docs/cli/reference
- https://antigravity.google/docs/cli/headless
- https://antigravity.google/docs/cli/features
- https://antigravity.google/docs/cli-permissions
- https://antigravity.google/docs/sandbox?tab=cli

若官方文件更新，必須透過 PoC 驗證實際安裝版本，不得直接假設新版功能在 1.4.12 完全相同。

## 10.2 CLI 執行方式

使用：

```bash
agy -p "prompt"
```

正式 Adapter 優先使用：

```bash
agy \
  --input-format stream-json \
  --output-format stream-json \
  --settings /path/to/settings.json \
  --sandbox \
  --print \
  --prompt "..."
```

實際參數組合應以 AG-01 至 AG-18 PoC 驗證結果為準。

不得將使用者輸入透過 shell 字串拼接。

Go 使用：

```go
exec.CommandContext(ctx, executable, args...)
```

Prompt 應依照已驗證的 agy 輸入格式透過 stdin 傳遞。

## 10.3 Settings 管理

每次 Execution 使用獨立設定檔：

```text
data/
└── executions/
    └── <execution-id>/
        └── settings.json
```

要求：

- 不修改 Global Settings。
- 不讀取或複製 OAuth Token。
- 不包含 API Key。
- 每次 Execution 保存 Profile ID。
- 每次 Execution 保存 Profile Hash。
- 設定檔建立前必須完成 Policy Evaluation。
- 設定檔不得允許任務自行覆寫高層安全政策。

## 10.4 Permission 管理

agy 提供底層工具權限管理。

AIA 管理高層任務政策及 Profile 選擇。

責任分工：

| 能力 | 負責者 |
|---|---|
| Task-level Permission | AIA |
| Agent Capability | AIA |
| Permission Profile | AIA |
| CLI Tool Permission | agy |
| Filesystem Permission | agy / OS Sandbox |
| Command Permission | agy / OS Sandbox |
| Network Permission | agy / OS Sandbox |
| MCP Permission | agy |
| OS Process Isolation | agy / OS |

不可使用：

`--dangerously-skip-permissions`

作為自動化捷徑。

## 10.5 Stream JSON Parser

Adapter 必須：

1. 分別讀取 stdout 與 stderr。
2. 逐行解析 stdout NDJSON。
3. 支援大型 JSON Event。
4. 設定 Scanner Buffer 上限。
5. 保存原始 Event。
6. 將事件轉換為 ProviderEvent。
7. 支援未知事件類型。
8. 不因未知事件導致整個 Runtime Crash。
9. 監控 stderr 中的錯誤及權限通知。
10. 處理 Process Exit。

必須辨識：

- init
- step_update
- result
- tool events
- subagent events

未辨識的 Event 仍應保存 Raw Payload。

## 10.6 Headless Soft-deny

Headless Mode 沒有互動式 Permission Prompt。

某些需要人工核准的操作可能 Soft-deny，但整個 Process 仍可能繼續，Exit Code 甚至可能是 0。

因此：

**不得只依 Exit Code 判斷成功。**

Adapter 必須盡可能識別：

- Tool Permission Denied
- Tool Soft-deny
- Tool Execution Error
- Authentication Failure
- Process Failure
- Result Failure

若需要人工核准：

1. 保存相關 Event。
2. Execution 標記 BLOCKED。
3. Task 標記 WAITING_FOR_APPROVAL。
4. 通知使用者。
5. 等待人工核准。
6. 重新執行必要工作。
7. 建立新的 Execution。
8. 保留原 Execution 的 Audit。

不可假設同一個 Headless Process 可以暫停等待使用者核准。

## 10.7 OAuth 邊界

| 工作 | 負責者 |
|---|---|
| Google Login | 使用者 |
| OAuth Authorization | 使用者 |
| Token Storage | agy |
| Token Refresh | agy（若支援） |
| Provider Availability Check | AIA |
| Provider Invocation | AIA |
| Credential Lifecycle | agy |

AIA 不得：

- 讀取 agy OAuth Token。
- 複製 OAuth Token。
- 保存 OAuth Token。
- 解析 OAuth Token。
- 將 Token 傳給 Domain Agent。
- 將 Token 寫入 Log。

若發現 Authentication Failure：

1. 停止需要 Provider 的工作。
2. Execution 標記 BLOCKED。
3. Task 標記 WAITING_FOR_AUTH。
4. 通知使用者。
5. 使用者自行透過 agy 登入或重新授權。
6. AIA 重新檢查 Provider Availability。
7. 恢復原 Task。

---

# 11. Domain Agent Extension Contract

本章是 AIA 的重要擴充介面。

未來新增 Domain Agent，不應修改 Runtime 核心流程。

## 11.1 Agent Registration

每個 Domain Agent 必須：

- 提供唯一 Agent ID。
- 提供 Manifest。
- 宣告 Domain。
- 宣告 Capability Requirements。
- 宣告 Provider Requirements。
- 宣告 Risk Level。
- 宣告執行限制。
- 提供 Analyze / Execute / Validate。

Agent Registry 負責載入與註冊。

## 11.2 內建 Agent 與外部 Agent

第一階段先採用 Go 內建 Agent。

未來才考慮：

- Plugin
- RPC Agent
- 外部 Process Agent
- Remote Agent

第一階段不要求 Go Plugin。

Go Plugin 的版本與部署限制較多，不適合作為初期必要擴充機制。

## 11.3 Agent 不應管理 Runtime

Agent 不應自行：

- 建立 SQLite Connection。
- 修改 Task 狀態。
- 建立 Execution。
- 啟動 Scheduler。
- 執行全域 Recovery。
- 任意修改 Policy。
- 存取其他 Agent 的 Private Workspace。

上述能力均由 Runtime 提供。

## 11.4 Agent Capability Access

Domain Agent 透過 CapabilityAccess 使用外部能力。

```go
type CapabilityAccess interface {
    Call(
        ctx context.Context,
        capability string,
        input map[string]any,
    ) (map[string]any, error)
}
```

Capability Access 必須：

1. 檢查 Capability 是否存在。
2. 檢查 Agent 是否宣告該 Capability。
3. 檢查 Task Policy。
4. 檢查 Execution Profile。
5. 檢查輸入參數。
6. 執行 Capability。
7. 保存 Audit。
8. 回傳標準化結果。

Domain Agent 不得自行繞過 Capability Access 呼叫受控外部服務。

---

# 12. External Data Adapter

## 12.1 設計目的

AIA 必須支援不同外部資料供應商。

外部資料包括：

- 金融市場資料。
- 財務資料。
- 新聞資料。
- 天氣資料。
- 其他公開資料 API。

外部資料供應商不得直接成為 Domain Agent 的程式碼依賴。

## 12.2 External Data Provider Interface

```go
type ExternalDataProvider interface {
    Name() string

    Type() string

    CheckAvailability(
        ctx context.Context,
    ) error

    Fetch(
        ctx context.Context,
        request DataRequest,
    ) (*DataResponse, error)
}
```

## 12.3 DataRequest

```go
type DataRequest struct {
    Resource string

    Parameters map[string]any

    StartTime *time.Time
    EndTime   *time.Time

    Timezone  string

    Limit     int
}
```

## 12.4 DataResponse

```go
type DataResponse struct {
    Provider  string

    Resource  string

    Data      map[string]any

    RetrievedAt time.Time

    DataTimestamp *time.Time

    IsDelayed bool

    DelayDescription string

    Currency string

    Metadata map[string]any
}
```

## 12.5 Data Quality

所有外部資料必須保留：

- Provider。
- RetrievedAt。
- DataTimestamp。
- Timezone。
- Currency（適用時）。
- Delayed / Real-time 狀態。
- 原始資料或可追溯引用。
- API 版本或資料來源識別。

不得將取得時間當成資料發生時間。

不得將延遲行情標示為即時行情。

不得將缺漏資料自行補成看似真實的數值。

## 12.6 API Credential Management

外部 API Key 不得放在：

- Prompt。
- Agent Manifest。
- 普通 YAML 設定。
- SQLite 一般資料表。
- Audit Log。
- Execution Event。
- Git Repository。

Credential 應由獨立的 Credential Provider 管理。

第一階段可支援：

- OS Environment Variable。
- OS-level Secret Store。
- 使用者自行設定的本機 Credential File，並限制檔案權限。

AIA 設定只保存 Credential Reference。

例如：

```yaml
external_data:
  providers:
    massive:
      enabled: true
      credential_ref: env:MASSIVE_API_KEY
```

實際 API Key 由 Runtime 的 Credential Resolver 取得。

Credential Resolver 不得將 Secret 傳入 AgentInput 或 Prompt。

## 12.7 HTTP Client

外部 HTTP Client 必須具備：

- Request Timeout。
- Context Cancellation。
- Retry Policy。
- Rate Limit Handling。
- HTTP Status Validation。
- JSON Validation。
- TLS Certificate Verification。
- Response Size Limit。
- Audit Metadata。

不得無限制重試。

對 401、403、429、5xx 必須採取不同處理方式。

---

# 13. Investment Agent v1 Runtime Support

Investment 是第一個實際落地的 Domain Agent。

本版本只要求 Runtime 提供必要支援，不在 AIA-03 中定義完整投資分析策略。

完整投資領域設計由獨立的 Investment Agent 設計規格書定義。

## 13.1 v1 功能定位

Investment Agent v1：

**Advisory Only — Research and Recommendations**

允許：

- 讀取美股行情。
- 讀取歷史價格。
- 讀取財報及公司資料。
- 讀取新聞及重大事件。
- 管理觀察清單。
- 分析投資標的。
- 產生投資建議。
- 產生研究報告。
- 保存投資決策紀錄。
- 追蹤歷史建議表現。

不允許：

- 連接真實券商下單。
- 建立交易委託。
- 修改券商帳戶。
- 提款。
- 資金轉移。
- 自動執行任何真實交易。

## 13.2 初期資料來源

可評估：

- Massive（原 Polygon.io）：美股行情及歷史資料。
- Alpaca Market Data：市場資料。
- 其他公開財務資料來源。

資料來源選擇由 Investment Agent 規格決定。

AIA-03 只定義統一的 External Data Adapter。

## 13.3 Scheduled Investment Analysis

Investment Agent 必須能被 Scheduler 定時觸發。

例如：

```yaml
id: investment-daily
name: Daily US Market Review

agent_id: investment

enabled: true

schedule_type: cron
cron_expr: "30 7 * * 1-6"
timezone: Asia/Taipei

input:
  task_type: daily_market_review
  market: US
```

上述排程僅為範例。

實際排程應考慮：

- 美國交易日曆。
- 夏令時間。
- 冬令時間。
- 假日。
- 資料更新完成時間。

不得假設每天都有美股交易。

## 13.4 Investment Decision Journal

每次投資建議應保存：

- 標的。
- 決策類型。
- 參考價格。
- 資料時間。
- 分析理由。
- 風險。
- 預期觀察期間。
- 失效條件。
- 預定檢討時間。

Decision 必須可追溯至當時的資料快照。

## 13.5 Runtime Boundary

Investment Agent 只能透過 Read-only External Data Capability 取得資料。

第一階段不得註冊任何 Trading Capability。

即使未來加入 Trading Capability，也必須另行設計獨立權限、資金安全、人工核准及驗收機制。

---

# 14. Orchestrator

## 14.1 職責

Orchestrator 負責：

- Task Planning。
- Agent Dispatch。
- Dependency Management。
- Execution Scheduling。
- Concurrency Control。
- Retry。
- Recovery Coordination。
- Result Integration。

## 14.2 Personal Agent Workflow

```text
User Input
    │
    ▼
Intent Classification
    │
    ▼
Memory Retrieval
    │
    ▼
Clarification (if required)
    │
    ▼
Plan Generation
    │
    ▼
Plan Validation
    │
    ▼
Policy Evaluation
    │
    ▼
Task Creation
    │
    ▼
Scheduler / Dispatcher
    │
    ▼
Domain Agent Execution
    │
    ▼
Result Validation
    │
    ▼
Memory Candidate
    │
    ▼
Result Integration
    │
    ▼
User Response
```

## 14.3 Planning

Planning 可使用 Provider 的結構化輸出功能，例如 `--json-schema`。

Plan 必須包含：

- 任務目標。
- Task 列表。
- Agent 分派。
- Dependency。
- 預期 Artifact。
- Capability Requirements。
- Risk Level。
- Acceptance Criteria。

Planner 產生的 Plan 必須經 Runtime 驗證。

不可將未驗證的模型輸出直接轉換成可執行命令。

## 14.4 Dispatcher

Dispatcher 只派發：

- READY Task。
- Dependency 已完成。
- Agent 已啟用。
- Policy 通過。
- Execution 未超過並行限制。

Dispatcher 不直接執行外部命令。

## 14.5 Dependency Management

支援 Task Dependency。

例如：

```text
Research Task
     │
     ▼
Analysis Task
     │
     ▼
Report Task
```

若前置 Task 未完成，後續 Task 不得執行。

Task Dependency 必須防止循環依賴。

## 14.6 Concurrency

初期預設：

```yaml
max_concurrent_executions: 2
```

可依設定調整。

每個 Agent 也可以有自己的最大並行數。

並行限制必須由 Runtime 統一管理。

## 14.7 Retry

預設最大 Retry：

2 次。

下列錯誤不得直接當成一般錯誤重試：

- Authentication Failure。
- Permission Denied。
- Policy Violation。
- Invalid Credential。
- User Cancellation。

Retry 必須建立新的 Execution。

不可重複執行不具 Idempotency 保證的外部操作。

---

# 15. Scheduler

## 15.1 設計目的

Scheduler 負責週期性任務及指定時間任務。

Scheduler 不直接執行 Agent。

Scheduler 只負責建立 Task 並交給 Orchestrator。

## 15.2 排程類型

第一階段支援：

- Once：指定時間執行一次。
- Cron：週期性排程。
- Interval：固定間隔。

## 15.3 Timezone

所有 Schedule 必須指定 Timezone。

資料庫統一保存 UTC 時間。

排程計算使用指定 Timezone。

例如：

`Asia/Taipei`

## 15.4 Misfire Policy

若 Runtime 關閉期間錯過排程：

- SKIP：略過。
- RUN_ONCE：重新啟動後執行一次。
- CATCH_UP：補執行所有錯過的任務。

第一階段預設 RUN_ONCE。

但必須限制補執行數量，避免長時間關機後產生大量任務。

## 15.5 Duplicate Prevention

同一 Schedule 在同一個排程週期不得重複建立 Task。

必須使用：

- Schedule ID。
- Scheduled Time。
- Unique Constraint 或等效 Idempotency Key。

## 15.6 Scheduler Lifecycle

Scheduler 必須：

- 支援啟動及停止。
- 支援 Enable / Disable。
- 支援 Next Run Time。
- 支援 Last Run Time。
- 支援 Misfire。
- 支援錯誤紀錄。
- 支援 Runtime Restart Recovery。

Scheduler 不得在記憶體中保存唯一的排程狀態。

SQLite 必須保存必要資訊。

---

# 16. Policy / Permission

## 16.1 責任

AIA Policy Engine 負責高層授權。

agy Permission Engine 負責底層工具權限。

兩者不可混為一談。

## 16.2 Policy Decision

```go
type PolicyDecision string

const (
    PolicyAllow PolicyDecision = "ALLOW"
    PolicyAsk   PolicyDecision = "ASK"
    PolicyDeny  PolicyDecision = "DENY"
)
```

## 16.3 Policy Context

```go
type PolicyContext struct {
    AgentID     ID
    TaskID      ID
    ExecutionID ID

    Capability string

    Risk        RiskLevel

    ProfileID   string

    Parameters  map[string]any
}
```

## 16.4 Policy Evaluator

```go
type PolicyEvaluator interface {
    Evaluate(
        ctx context.Context,
        input PolicyContext,
    ) (PolicyDecision, string, error)
}
```

## 16.5 Policy 原則

- 預設拒絕高風險操作。
- Agent Manifest 不能自行提升權限。
- Task 不可自行修改 Profile。
- Execution Profile 必須留下 Hash。
- 人工核准必須留下 Audit。
- Policy Deny 不可透過 Retry 繞過。
- 高層政策不得被 Provider 設定覆蓋。

## 16.6 Permission Profile

Profile 用於定義不同任務的底層工具權限。

範例：

```yaml
id: investment-readonly
name: Investment Read Only

permissions:
  filesystem:
    workspace_only: true

  network:
    mode: registered_capabilities_only

  command:
    mode: deny

  external_write:
    mode: deny

  trading:
    mode: deny
```

實際 agy settings 欄位必須依版本 PoC 驗證後產生。

不得將上述示意 YAML 直接當成 agy 的原生設定格式。

---

# 17. Workspace Management

## 17.1 Workspace 隔離

每個 Execution 使用獨立 Workspace。

```text
workspaces/
└── <project-id>/
    └── executions/
        └── <execution-id>/
```

不同 Agent 不得任意存取其他 Agent 的 Workspace。

## 17.2 Workspace Metadata

每個 Workspace 應保存：

- Execution ID。
- Task ID。
- Agent ID。
- Project ID。
- Created At。
- Profile ID。
- Profile Hash。

## 17.3 Workspace Cleanup

不可在 Execution 結束後立即刪除 Workspace。

必須依 Artifact Persistence、Recovery 及 Audit Policy 決定保留期限。

預設成功的 Execution 也不自動刪除。

## 17.4 Path Security

必須防止：

- Path Traversal。
- Symlink Escape。
- 跨 Workspace 寫入。
- 未授權檔案讀取。
- Workspace Root 外的檔案操作。

---

# 18. Result Validation

## 18.1 驗證原則

Provider 回傳成功，不代表 Task 成功。

Task 必須經過 Validator。

## 18.2 Coding Validator

檢查：

- 預期檔案是否存在。
- Diff 是否符合要求。
- Test 是否通過。
- Build 是否成功。
- Acceptance Criteria 是否完成。

## 18.3 Writing Validator

檢查：

- 文件是否存在。
- 檔案格式是否正確。
- 必要章節是否完整。
- 字數及內容要求。
- 引用及來源要求。

## 18.4 Research Validator

檢查：

- 資料來源。
- 資料日期。
- 來源可信度。
- 不確定性。
- 事實與推論是否區分。

## 18.5 Investment Validator

檢查：

- Symbol 是否有效。
- 資料來源是否存在。
- 行情時間是否明確。
- 是否標示延遲資料。
- 幣別是否明確。
- 分析理由是否存在。
- 風險是否存在。
- 是否保存 Decision Record。
- 是否錯誤產生交易操作。

Investment v1 必須拒絕任何交易委託類操作。

## 18.6 Completion Criteria

Task 完成前必須確認：

1. 所有必要 Subtasks 已完成。
2. 所有必要 Artifacts 已保存。
3. Validator 通過。
4. 不存在未解決的 Permission Block。
5. 不存在未解決的 Authentication Block。
6. Audit 已完成。
7. Memory Candidate 已依政策保存。

---

# 19. Memory Management

## 19.1 設計目的

Memory 用於跨 Task、跨 Execution 保留有價值的資訊。

Memory 不等同於完整 Conversation Log。

## 19.2 Memory 類型

第一階段支援：

- User Preference
- Project Context
- Historical Decision
- Task Knowledge
- Agent Experience
- System Knowledge

## 19.3 Memory Scope

- Global
- Project
- Agent
- Task

Scope 必須明確。

## 19.4 Memory Candidate

Agent 不得直接寫入高可信度的永久 Memory。

必須先產生 Memory Candidate。

```go
type MemoryCandidate struct {
    ID          ID

    AgentID     ID
    TaskID      ID

    Type        string
    Scope       string

    Content     string
    Summary     string

    Confidence  float64

    SourceType  string
    SourceID    *ID

    CreatedAt   time.Time
}
```

## 19.5 Memory Approval

第一階段：

- 一般工作經驗可依 Policy 自動採用。
- 使用者長期偏好必須保留來源及可修訂性。
- 重大使用者資訊變更應取得確認。
- 高風險決策不可直接升級成永久 Memory。

## 19.6 Retrieval

第一階段：

- SQLite。
- FTS5（若支援）。
- Keyword Search。
- Scope Filter。
- Confidence Filter。
- Time Filter。

第一階段不要求 Vector Database。

## 19.7 Memory Lifecycle

Memory 必須支援：

- Create
- Search
- Update
- Approve
- Reject
- Expire
- Archive

Memory 修改必須保留必要 Audit。

---

# 20. Audit

## 20.1 Audit 目的

記錄重要系統行為，以支援：

- 問題調查。
- 安全檢查。
- Execution Replay。
- Decision Review。
- Recovery。
- Agent Evaluation。

## 20.2 必須記錄的事件

- Task Created。
- Task Updated。
- Execution Started。
- Execution Completed。
- Execution Failed。
- Provider Started。
- Provider Stopped。
- Tool Called。
- Tool Denied。
- Policy Decision。
- User Approval。
- Authentication Failure。
- Memory Approved。
- Memory Rejected。
- Schedule Triggered。
- Recovery Started。
- Recovery Completed。
- Agent Enabled / Disabled。

## 20.3 Sensitive Data

不得記錄：

- OAuth Token。
- Password。
- API Key。
- Credential。
- Secret Key。
- Private Key。
- Session Secret。

Log 必須避免將敏感資料寫入 Error Message 或 Provider Raw Payload。

對需要保存的 Provider Event，必須先執行 Secret Redaction。

---

# 21. Recovery

## 21.1 Recovery 原則

AIA 必須能在 Runtime 中斷後恢復未完成工作。

## 21.2 Startup Recovery

Runtime 啟動時：

1. 查詢未完成 Execution。
2. 檢查對應 Process 是否仍存在。
3. 判斷是否可以恢復 Provider Session。
4. 若無法恢復，Execution 標記 INTERRUPTED。
5. 保留既有 Event。
6. 判斷 Task 是否可以重新執行。
7. 若可以，建立新的 Execution。
8. 若需要人工處理，標記 WAITING_FOR_APPROVAL 或 WAITING_FOR_AUTH。

## 21.3 Idempotency

每個可能造成外部副作用的操作必須評估 Idempotency。

例如：

- 建立檔案。
- 發送 Email。
- 建立外部任務。
- 呼叫外部 API。
- 更新外部資料。

第一階段不得假設所有外部操作都可安全重試。

## 21.4 Recovery Policy

| 錯誤 | 處理 |
|---|---|
| Network Timeout | 有條件 Retry |
| Provider Crash | 建立新 Execution |
| Authentication Failure | 等待使用者登入 |
| Permission Denied | 等待人工核准 |
| Policy Denied | 停止 |
| Invalid Input | 回報錯誤 |
| Runtime Crash | Startup Recovery |
| Database Failure | 停止新工作並保護資料 |

---

# 22. SQLite Persistence

## 22.1 Database

第一階段使用 SQLite。

設定：

- WAL Mode。
- Busy Timeout。
- Foreign Key Enforcement。
- Transaction。
- Database Migration。

## 22.2 Tables

第一階段至少建立：

```text
projects
agents
tasks
task_dependencies
executions
execution_events
artifacts
memories
memory_candidates
schedules
schedule_runs
investment_decisions
audit_logs
```

## 22.3 Schema Requirements

### projects

保存 Project 資訊。

### agents

保存 Agent Manifest、Version、Status。

### tasks

保存 Task 狀態及輸入輸出。

### task_dependencies

保存 Task 依賴關係。

### executions

保存每次執行的生命週期。

### execution_events

保存 Execution Event。

### artifacts

保存產出檔案資訊。

### memories

保存正式 Memory。

### memory_candidates

保存待審核 Memory。

### schedules

保存排程定義。

### schedule_runs

保存排程觸發紀錄及 Idempotency Key。

### investment_decisions

保存投資決策紀錄。

### audit_logs

保存重要系統操作。

## 22.4 Repository Interface

至少提供：

```go
type TaskRepository interface {
    Create(ctx context.Context, task Task) error
    Get(ctx context.Context, id ID) (Task, error)
    Update(ctx context.Context, task Task) error
    List(ctx context.Context, filter TaskFilter) ([]Task, error)
}

type ExecutionRepository interface {
    Create(ctx context.Context, execution Execution) error
    Get(ctx context.Context, id ID) (Execution, error)
    Update(ctx context.Context, execution Execution) error
    List(ctx context.Context, filter ExecutionFilter) ([]Execution, error)
    AppendEvent(ctx context.Context, event ExecutionEvent) error
    ListEvents(ctx context.Context, id ID) ([]ExecutionEvent, error)
}

type MemoryRepository interface {
    Create(ctx context.Context, memory Memory) error
    Get(ctx context.Context, id ID) (Memory, error)
    Search(ctx context.Context, query MemoryQuery) ([]Memory, error)
    Update(ctx context.Context, memory Memory) error
}

type AgentRepository interface {
    Create(ctx context.Context, agent Agent) error
    Get(ctx context.Context, id ID) (Agent, error)
    List(ctx context.Context) ([]Agent, error)
    Update(ctx context.Context, agent Agent) error
}

type ScheduleRepository interface {
    Create(ctx context.Context, schedule Schedule) error
    Get(ctx context.Context, id ID) (Schedule, error)
    List(ctx context.Context) ([]Schedule, error)
    Update(ctx context.Context, schedule Schedule) error
}
```

其他 Repository 依需求建立。

## 22.5 Transaction

下列操作必須考慮 Transaction：

- 建立 Task 與 Dependencies。
- 建立 Execution 與 Initial Event。
- 更新 Task 與 Execution 狀態。
- 保存 Artifact Metadata。
- 保存 Memory Candidate。
- 建立 Schedule Run 與對應 Task。
- 保存 Investment Decision。

不得在 Database Transaction 中等待長時間 Provider 執行。

---

# 23. CLI Interface

## 23.1 Root Commands

```bash
aia version
aia status
aia doctor
```

## 23.2 Chat

```bash
aia chat
aia chat "整理最近的工作進度"
```

## 23.3 Task

```bash
aia task create
aia task list
aia task show <task-id>
aia task cancel <task-id>
aia task resume <task-id>
```

## 23.4 Agent

```bash
aia agent list
aia agent show <agent-id>
aia agent enable <agent-id>
aia agent disable <agent-id>
```

## 23.5 Project

```bash
aia project create
aia project list
aia project show <project-id>
aia project archive <project-id>
```

## 23.6 Execution

```bash
aia execution list
aia execution show <execution-id>
aia execution events <execution-id>
aia execution cancel <execution-id>
```

## 23.7 Schedule

```bash
aia schedule list
aia schedule show <schedule-id>
aia schedule create
aia schedule enable <schedule-id>
aia schedule disable <schedule-id>
aia schedule run <schedule-id>
```

## 23.8 Memory

```bash
aia memory list
aia memory search <query>
aia memory candidate
aia memory approve <memory-id>
aia memory reject <memory-id>
```

## 23.9 Investment

Investment 專屬 CLI 不應直接操作 Broker。

第一階段可以提供：

```bash
aia investment report
aia investment decisions
aia investment decision show <decision-id>
aia investment watchlist
```

實際命令可由 Investment Agent 規格決定。

---

# 24. Logging

使用 Go slog。

預設 JSON Log。

Log Level：

- DEBUG
- INFO
- WARN
- ERROR

Log 必須包含：

- Timestamp。
- Level。
- Component。
- Task ID（適用時）。
- Execution ID（適用時）。
- Agent ID（適用時）。
- Event Type。
- Error Code。

不得包含 Secret。

Provider stdout、stderr 不應不加過濾就直接寫入一般 Log。

需要保存原始輸出時，應經過 Redaction 並受 Workspace 權限保護。

---

# 25. Error Handling

統一錯誤分類：

```go
type ErrorCode string

const (
    ErrInvalidInput       ErrorCode = "INVALID_INPUT"
    ErrNotFound           ErrorCode = "NOT_FOUND"
    ErrConflict           ErrorCode = "CONFLICT"
    ErrProviderUnavailable ErrorCode = "PROVIDER_UNAVAILABLE"
    ErrProviderFailure    ErrorCode = "PROVIDER_FAILURE"
    ErrAuthentication     ErrorCode = "AUTHENTICATION_FAILURE"
    ErrPermissionDenied   ErrorCode = "PERMISSION_DENIED"
    ErrPolicyDenied       ErrorCode = "POLICY_DENIED"
    ErrTimeout            ErrorCode = "TIMEOUT"
    ErrCancelled          ErrorCode = "CANCELLED"
    ErrDatabase           ErrorCode = "DATABASE_ERROR"
    ErrValidation         ErrorCode = "VALIDATION_FAILURE"
    ErrExternalData       ErrorCode = "EXTERNAL_DATA_FAILURE"
)
```

錯誤必須能被：

- CLI 顯示。
- Audit 保存。
- Recovery 判斷。
- 使用者理解。

不得只回傳無法分類的字串錯誤。

---

# 26. Testing

## 26.1 Unit Tests

必須測試：

- Domain Model。
- Task State Transition。
- Execution State Transition。
- Agent Registry。
- Policy Engine。
- Scheduler。
- Dependency Manager。
- Recovery。
- Memory。
- Validator。
- External Data Adapter。

## 26.2 Integration Tests

必須測試：

- SQLite Migration。
- Repository CRUD。
- Transaction。
- Task / Execution Lifecycle。
- Agent Registration。
- Scheduled Task。
- Memory Retrieval。
- Workspace Isolation。

## 26.3 Provider Tests

使用 Mock Provider 測試 Runtime。

Antigravity CLI 使用獨立 Integration Test。

不可讓一般 Unit Test 依賴使用者的 Google Login。

## 26.4 Investment Tests

必須測試：

- 延遲行情標記。
- 資料時間與取得時間。
- API Rate Limit。
- API Timeout。
- 缺漏資料。
- 不同幣別。
- Decision Record。
- Scheduled Analysis。
- 重複排程防止。
- 交易 Capability 拒絕。

---

# 27. Antigravity Proof of Concept

實際整合前，必須完成以下 PoC。

| 編號 | 測試項目 |
|---|---|
| AG-01 | agy Version |
| AG-02 | Headless Prompt |
| AG-03 | JSON Output |
| AG-04 | Stream JSON |
| AG-05 | Multi-turn |
| AG-06 | Tool Event |
| AG-07 | Process Exit |
| AG-08 | Process Cancel |
| AG-09 | Timeout |
| AG-10 | Session Settings |
| AG-11 | Settings Precedence |
| AG-12 | Sandbox |
| AG-13 | Permission |
| AG-14 | Soft-deny |
| AG-15 | Conversation Resume |
| AG-16 | Concurrent Process |
| AG-17 | OAuth Availability |
| AG-18 | Workspace Isolation |

未通過的項目不得宣稱已整合成功。

PoC 必須保存：

- 測試環境。
- agy 版本。
- 測試命令。
- 測試輸出。
- 實際觀察結果。
- 已知限制。
- 通過或失敗。

---

# 28. Build / Development

## 28.1 Makefile

至少提供：

```makefile
.PHONY: build test test-race run clean

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

run:
	go run ./cmd/aia

clean:
	go clean
```

另應提供：

```bash
make build
make test
make test-race
```

## 28.2 Build Requirements

必須通過：

```bash
go build ./...
go test ./...
go test -race ./...
```

## 28.3 Doctor

`aia doctor` 必須檢查：

- Go Runtime。
- Config。
- SQLite。
- Database Migration。
- agy 安裝狀態。
- agy 版本。
- Provider Availability。
- Workspace。
- Profile。
- Agent Registry。
- Scheduler。

Doctor 不得讀取或輸出 OAuth Token。

---

# 29. Development Milestones

## 0.1 — Runtime Foundation

完成：

- Go 專案結構。
- Config。
- SQLite。
- Migration。
- Domain Model。
- Repository。
- CLI。
- Logging。

驗收：

- 專案可編譯。
- Database 可初始化。
- 基本 CLI 可執行。

## 0.2 — Agent Registry

完成：

- Agent Manifest。
- Agent Registry。
- Domain Agent Contract。
- Built-in Agent Registration。
- Mock Agent。

驗收：

- 可註冊 Agent。
- 可查詢 Agent。
- 可啟用與停用 Agent。
- 可透過 Contract 執行 Mock Agent。

## 0.3 — Antigravity Provider

完成：

- Provider Interface。
- Antigravity Adapter。
- Stream JSON Parser。
- Process Management。
- Event Persistence。
- Error Classification。

驗收：

- 可執行 Headless Prompt。
- 可解析事件。
- 可取消 Process。
- 可處理 Timeout。
- 可辨識基本錯誤。

AG PoC 未通過項目必須明確標示。

## 0.4 — Task / Execution Orchestration

完成：

- Task Service。
- Execution Service。
- Dispatcher。
- Dependency。
- Concurrency。
- Result Validation。

驗收：

- 可建立 Task。
- 可派發 Agent。
- 可執行。
- 可保存結果。
- 可驗證結果。

## 0.5 — Scheduler

完成：

- Schedule Model。
- Schedule Repository。
- Scheduler。
- Cron / Once。
- Timezone。
- Misfire。
- Duplicate Prevention。

驗收：

- 可建立排程。
- 可定時建立 Task。
- Runtime Restart 後能恢復排程。
- 不重複觸發相同排程週期。

## 0.6 — Memory

完成：

- Memory。
- Memory Candidate。
- Retrieval。
- Approval。
- Lifecycle。

驗收：

- 可保存 Memory。
- 可搜尋 Memory。
- 可依 Scope 篩選。
- 可核准及拒絕 Candidate。

## 0.7 — External Data Capability

完成：

- External Data Provider Interface。
- Capability Registry。
- Credential Resolver。
- HTTP Client。
- Rate Limit。
- Data Quality Metadata。

驗收：

- 可使用 Mock External Data Provider。
- 可處理 API Error。
- 不洩漏 API Key。
- 可保存資料來源及時間。

## 0.8 — Personal Agent

完成：

- Intent Classification。
- Memory Retrieval。
- Planning。
- Clarification。
- Task Creation。
- Agent Dispatch。
- Result Integration。

驗收：

- 使用者可以透過 CLI 對話。
- Personal Agent 能建立 Task。
- 能分派 Domain Agent。
- 能整合結果並回覆。

## 0.9 — Recovery / Reliability

完成：

- Startup Recovery。
- Retry。
- Idempotency。
- Graceful Shutdown。
- Audit。
- Error Recovery。

驗收：

- Runtime 中斷後能辨識未完成工作。
- 能恢復可恢復任務。
- 不重複執行已完成的不可重複操作。
- 狀態及事件完整保存。

## 1.0 — AIA Personal Agent Complete

完成：

- Runtime。
- Personal Agent。
- Agent Registry。
- Domain Agent Contract。
- Provider Adapter。
- Scheduler。
- Memory。
- Policy。
- Recovery。
- Audit。
- External Data Framework。

驗收：

- 可以長期啟動。
- 可以接收任務。
- 可以分派 Agent。
- 可以定時工作。
- 可以保存 Memory。
- 可以恢復未完成任務。
- 可以追蹤所有重要執行事件。

---

# 30. Implementation Rules

以下規則為強制要求。

1. Domain Model 不得依賴 Antigravity JSON。
2. Domain Agent 不得直接操作 SQLite。
3. Domain Agent 不得直接修改 Task 狀態。
4. Provider Adapter 不得直接更新 Task。
5. CLI 不得直接操作 Repository。
6. 所有外部 API 必須經過 Capability。
7. 所有外部操作必須經過 Policy。
8. 所有 Execution 必須保存事件。
9. 所有 Retry 必須建立新的 Execution。
10. 不得只依 Exit Code 判斷 Provider 成功。
11. 不得將 OAuth Token 保存至 AIA。
12. 不得使用 skip-permissions 繞過安全控制。
13. 不得將 Credential 放入 Prompt。
14. 不得將 API Key 寫入 Log。
15. 不得讓 Domain Agent 任意存取其他 Agent Workspace。
16. 所有長時間工作必須接受 Context Cancellation。
17. Scheduler 必須具備 Duplicate Prevention。
18. Scheduled Task 必須保存原始排程時間。
19. 外部資料必須保存來源及資料時間。
20. 延遲資料不得標示為即時資料。
21. Investment v1 不得具備交易 Capability。
22. Provider Result 必須經過 Validation。
23. Runtime 啟動時必須執行 Recovery。
24. SQLite Migration 必須可重複安全執行。
25. 不得在長時間 Provider 執行期間持有 Database Transaction。

---

# 31. 最終交付標準

AIA-03 的實作完成，不以程式碼數量或檔案數量判斷。

必須達成：

- 可編譯。
- 可啟動。
- 可建立及管理 Task。
- 可註冊及執行 Domain Agent。
- 可使用 Antigravity Provider。
- 可保存 Execution Event。
- 可執行 Scheduled Task。
- 可透過 Capability 取得外部資料。
- 可保存及搜尋 Memory。
- 可進行 Policy Evaluation。
- 可驗證 Agent Result。
- 可處理錯誤及 Recovery。
- 可透過 CLI 操作。
- 可追蹤重要系統活動。

未完成的功能必須明確標示：

- Implemented
- Mocked
- Pending PoC
- Not Implemented

不得以 Mock 結果宣稱外部整合已完成。

---

# 32. 架構演進原則

AIA 第一階段採用 Go Modular Monolith。

不急著拆分微服務。

所有核心模組透過明確 Interface 分隔。

未來可以：

- 替換 AI Provider。
- 新增 Domain Agent。
- 新增 External Data Provider。
- 新增 Mobile Client。
- 新增 Web Client。
- 新增其他 Runtime 實作。
- 將 Go Runtime 改用 Kotlin 重寫。

但不應因為更換語言或 Provider，就必須重新設計 Task、Execution、Agent Contract、Memory、Policy 及 Audit 的概念。

**AIA 的核心價值是 Runtime 與生命週期管理，而不是綁定某個模型、CLI 或程式語言。**

---

**文件結束**

**AIA-03 v2.0 — Go Runtime 完整實作規格書**
