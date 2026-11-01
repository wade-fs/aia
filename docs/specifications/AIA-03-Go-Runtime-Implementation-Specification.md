# AIA-03：Go Runtime Implementation Specification

| 項目 | 內容 |
|---|---|
| 文件編號 | AIA-03 |
| 文件名稱 | Go Runtime Implementation Specification |
| 版本 | 2.0 |
| 文件狀態 | Revised Baseline |
| 系統名稱 | AIA — Autonomous Intelligence Architecture |
| 實作語言 | Go |
| Database | SQLite |
| Agent Harness | Antigravity CLI |
| 主要依據 | AIA-01 v2.0、AIA-02 v2.0 |
| 相關規格 | AIA-04 v2.0、AIA-05 v2.0、AIA-06 v2.0、AIA-07 v2.0 |

---

# 1. 文件目的

本文件定義 AIA Go Runtime 的程式架構、Package 責任、核心 Interface、Service、Worker、Persistence、Agent Harness 整合及實作順序。

AIA 採用 Go 作為第一階段實作語言。

選擇 Go 的主要原因：

- Goroutine 適合實作長期運作的 Worker。
- Context 適合管理任務生命週期及取消。
- Interface 適合隔離 Domain Agent 與外部 Harness。
- SQLite 適合第一階段單機持久化。
- 單一執行檔便於部署與維護。
- Go 的型別系統有助於建立明確的模組責任邊界。

本文件不要求將所有功能重新實作。

現有 Go 程式碼應保留並逐步調整，讓既有功能能夠成為新的 AIA 架構基礎。

# 2. Implementation Objectives

## 2.1 核心實作目標

Go Runtime 必須支援：

1. AIA 程序持續運作。
2. Runtime Lifecycle 管理。
3. Background Worker 管理。
4. Task 建立、派發與執行。
5. Schedule 到期處理。
6. Runtime Restart Recovery。
7. Persistent Memory。
8. Goal Management。
9. Autonomy Core。
10. Domain Agent Integration。
11. Agent Harness Integration。
12. Policy、Approval 與 Audit。
13. Web Console 與 CLI。
14. 長期自主性行為觀測。

## 2.2 實作優先順序

優先順序不是先建立更多 Agent，而是先建立能夠支撐 Agent 長期運作的 Runtime。

建議順序：

```text
Persistent Runtime
        │
        ▼
Task / Schedule Workers
        │
        ▼
Continuity & Memory
        │
        ▼
Initiative Engine
        │
        ▼
Reasoning & Deliberation
        │
        ▼
Self-Correction
        │
        ▼
Goal Evolution
        │
        ▼
Autonomous Domain Operation
```

## 2.3 不重寫原則

以下既有功能應優先保留：

- Go Application。
- CLI。
- Web Host。
- SQLite。
- Repository。
- Agent Registry。
- Task Management。
- Schedule Management。
- Recovery。
- Event Bus。
- Investment Service。
- Collector Framework。
- Harness Integration。

只有在責任不清、資料模型不足或實作不符合新架構時，才進行重構。

# 3. Go Architecture

## 3.1 Target Package Structure

以下是目標架構。

```text
go/
├── cmd/
│
├── internal/
│   ├── app/
│   │
│   ├── runtime/
│   │   ├── runtime.go
│   │   ├── lifecycle.go
│   │   ├── supervisor.go
│   │   ├── worker.go
│   │   ├── recovery.go
│   │   └── health.go
│   │
│   ├── autonomy/
│   │   ├── core.go
│   │   ├── initiative/
│   │   ├── continuity/
│   │   ├── reasoning/
│   │   ├── deliberation/
│   │   ├── correction/
│   │   └── goal/
│   │
│   ├── domain/
│   │   ├── agent/
│   │   ├── investment/
│   │   ├── coding/
│   │   └── writing/
│   │
│   ├── orchestrator/
│   │   ├── scheduler.go
│   │   └── dispatcher.go
│   │
│   ├── harness/
│   │   ├── interface.go
│   │   └── antigravity/
│   │
│   ├── service/
│   │   ├── task/
│   │   ├── goal/
│   │   ├── memory/
│   │   ├── policy/
│   │   ├── approval/
│   │   ├── audit/
│   │   ├── schedule/
│   │   └── investment/
│   │
│   ├── collector/
│   │
│   ├── repository/
│   │
│   ├── event/
│   │
│   ├── web/
│   │
│   ├── cli/
│   │
│   └── config/
│
├── migrations/
├── docs/
├── Makefile
└── README.md
```

此目錄是目標責任劃分，不要求立即建立所有目錄。

目前已存在的 `internal/service`、`internal/orchestrator`、`internal/domain`、`internal/collector` 等 Package 應優先沿用。

## 3.2 Package Responsibility

| Package | Responsibility |
|---|---|
| app | 建立及組裝系統元件 |
| runtime | 程序生命週期與背景運作 |
| autonomy | 共通自主性能力 |
| domain | Domain Agent 與領域邏輯 |
| orchestrator | Task 執行協調 |
| harness | 外部 Agent Harness 整合 |
| service | 應用服務 |
| collector | 外部資料收集 |
| repository | 資料存取 |
| event | 事件傳遞 |
| web | HTTP Host 與 Web Module |
| cli | 命令列介面 |
| config | 系統設定 |

## 3.3 Dependency Direction

依賴方向應保持清楚。

```text
CLI / Web
    │
    ▼
Application Services
    │
    ▼
Autonomy Core
    │
    ▼
Domain Interfaces
    │
    ▼
Domain Implementations

Runtime
    │
    ├── Task Service
    ├── Schedule Service
    ├── Autonomy Core
    └── Recovery

Services
    │
    ├── Repository Interfaces
    ├── Event Interfaces
    ├── Policy Interfaces
    └── Harness Interfaces
```

基本原則：

- Domain 不應直接依賴 CLI。
- Domain 不應直接依賴 Web Handler。
- Autonomy Core 不應直接依賴 Investment Entity。
- Service 不應直接依賴特定 HTTP Handler。
- Repository 實作不應洩漏 SQL 細節給上層。
- Harness 的實作細節應被 Adapter 隔離。

# 4. Application Bootstrap

## 4.1 App Responsibilities

`internal/app` 負責建立整個系統所需的元件。

目前 `NewApp()` 已建立部分核心服務，包括：

- SQLite。
- Repository。
- EventBus。
- Services。
- Collector Service。
- Schedule Service。
- Recovery Manager。
- Scheduler。
- Personal Agent。

這些既有初始化責任應保留。

但必須區分：

**App Initialization 不等於 Runtime Startup。**

建立 Service 只代表元件已存在，不代表它已開始持續工作。

## 4.2 App Structure

建議 App 結構：

```go
type App struct {
    Config Config

    DB *sql.DB

    Repositories *Repositories
    Services     *Services

    Registry  *AgentRegistry
    Harness   HarnessAdapter
    Autonomy  AutonomyCore
    Runtime   *Runtime

    WebHost *WebHost
}
```

以上為目標結構示意。

應以目前 `internal/app/app.go` 的既有型別為基礎逐步調整，不要求直接替換現有 App。

## 4.3 Initialization Sequence

```text
NewApp()
   │
   ├── Load Configuration
   │
   ├── Initialize Database
   │
   ├── Run Migrations
   │
   ├── Initialize Repositories
   │
   ├── Initialize Event Bus
   │
   ├── Initialize Services
   │
   ├── Initialize Agent Registry
   │
   ├── Initialize Harness Adapter
   │
   ├── Initialize Autonomy Core
   │
   ├── Initialize Runtime
   │
   ├── Register Domain Modules
   │
   └── Return App
```

初始化期間可以執行必要的靜態設定與資料檢查。

不得在 `NewApp()` 中直接執行無限迴圈或長期 Worker。

## 4.4 Runtime Startup

Runtime 應由明確的啟動流程控制。

```text
App.Initialize()
       │
       ▼
Runtime.Start()
       │
       ▼
Recovery
       │
       ▼
Worker Startup
       │
       ▼
Runtime Running
```

Web Server 可以與 Runtime 使用同一程序，但兩者必須有獨立的生命週期管理。

# 5. Persistent Runtime

## 5.1 Runtime Interface

建議定義：

```go
type Runtime interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Status() RuntimeStatus
}
```

Runtime 必須提供：

- 啟動。
- 停止。
- 狀態查詢。
- Worker 管理。
- Recovery。
- Health Monitoring。

## 5.2 Runtime State

```go
type RuntimeStatus string

const (
    RuntimeStopped      RuntimeStatus = "STOPPED"
    RuntimeStarting     RuntimeStatus = "STARTING"
    RuntimeRecovering   RuntimeStatus = "RECOVERY"
    RuntimeInitializing RuntimeStatus = "INITIALIZING"
    RuntimeRunning      RuntimeStatus = "RUNNING"
    RuntimeDegraded     RuntimeStatus = "DEGRADED"
    RuntimeStopping     RuntimeStatus = "STOPPING"
)
```

狀態轉移應由 Runtime 統一管理。

## 5.3 Worker Interface

```go
type Worker interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Status() WorkerStatus
}
```

Worker 應由 Runtime Supervisor 管理。

## 5.4 Worker Supervisor

Supervisor 負責：

- 啟動 Worker。
- 停止 Worker。
- 監控 Worker。
- 記錄 Worker 狀態。
- 處理 Worker 非預期終止。
- 控制重啟策略。
- 避免重複啟動。

Worker 發生錯誤時，不應讓整個 AIA 程序無條件退出。

但也不能無限重啟而不留下錯誤紀錄。

應依 Worker 重要程度決定：

- Restart。
- Degraded。
- Disable。
- Runtime Shutdown。

## 5.5 Graceful Shutdown

收到 SIGINT 或 SIGTERM 時：

1. 停止接受新的工作。
2. 停止 Schedule Worker 派發新 Task。
3. 通知 Task Worker 停止接收新工作。
4. 等待正在執行的工作完成或進入可恢復狀態。
5. 保存必要狀態。
6. 停止其他 Worker。
7. 關閉 Event Bus。
8. 關閉 Database。

關閉程序應具有總體 Timeout。

超時後，未完成 Task 必須能在下一次啟動時恢復。

# 6. Background Workers

## 6.1 Worker Architecture

第一階段需要建立下列 Worker：

| Worker | Responsibility | Priority |
|---|---|---|
| Task Worker | 執行 Ready Tasks | P0 |
| Schedule Worker | 處理到期排程 | P0 |
| Initiative Worker | 執行自主性評估 | P1 |
| Recovery Worker | 恢復異常工作 | P1 |
| Evaluation Worker | 執行預測與結果評估 | P1 |
| Observation Worker | Domain 環境觀察 | P1 |

P0 是建立持續運作能力的必要條件。

P1 是完整自主性實驗所需的下一階段功能。

## 6.2 Task Worker

Task Worker 負責：

- 取得 Ready Task。
- 檢查 Task 狀態。
- 取得必要 Context。
- 呼叫 Scheduler.RunTask。
- 更新執行狀態。
- 保存執行結果。
- 處理失敗與重試。
- 回報 Event。

目前 `internal/orchestrator/scheduler.go` 已有 `RunTask()` 與 `ProcessReadyTasks()` 相關實作。

新的 Task Worker 應優先呼叫這些既有方法，不應再建立另一套重複執行引擎。

## 6.3 Schedule Worker

Schedule Worker 負責：

- 定期檢查到期 Schedule。
- 呼叫 `ProcessDueSchedules()`。
- 建立對應 Task。
- 記錄執行結果。
- 處理 Misfire。
- 避免重複派發。

Schedule Worker 不應直接執行 Agent。

它只負責將 Schedule 轉換為 Task。

## 6.4 Initiative Worker

Initiative Worker 負責觸發 Autonomy Core 的主動性評估。

基本流程：

```text
Runtime Tick
     │
     ▼
Load Current Goals
     │
     ▼
Collect Environment State
     │
     ▼
Load Relevant Memory
     │
     ▼
Initiative Assessment
     │
     ├── CREATE_TASK
     ├── CONTINUE_TASK
     ├── REASSESS_GOAL
     ├── WAIT
     └── NO_ACTION
```

不可將 Initiative Worker 實作成固定時間呼叫 LLM 並要求產生 Task。

它必須根據實際環境、目標、記憶及資源評估是否值得行動。

## 6.5 Observation Worker

Observation Worker 負責觸發 Domain Agent 的環境觀察。

例如 Investment Agent：

- 每個台灣證券營業日觀察市場。
- 檢查台積電。
- 檢查持股。
- 檢查重要事件。
- 建立 Research Snapshot。

觀察工作可以由 Domain Schedule 提供觸發條件，但觀察結果應回到 Autonomy Core，讓 Core 判斷是否需要後續行動。

## 6.6 Evaluation Worker

負責執行到期的評估工作：

- Prediction Evaluation。
- Strategy Evaluation。
- Task Result Verification。
- Self-Correction Validation。
- Portfolio Valuation。
- Monthly Review。

評估週期應由 Domain 設定，不應寫死在 Runtime。

# 7. Task Management

## 7.1 Task Execution Pipeline

```text
Task Repository
      │
      ▼
Ready Task Selection
      │
      ▼
Task Claim
      │
      ▼
Policy Check
      │
      ▼
Context Building
      │
      ▼
Harness Execution
      │
      ▼
Result Collection
      │
      ▼
Verification
      │
      ▼
Task State Update
      │
      ▼
Event & Memory
```

## 7.2 Task Claim

多個 Worker 或程序可能同時嘗試取得同一 Task。

必須有原子化的 Task Claim 機制。

Task Claim 至少應考慮：

- Task ID。
- Worker ID。
- Claimed At。
- Lease Expiration。
- Attempt Count。
- Current Status。

不能只依賴先查詢再更新的非原子操作。

## 7.3 Idempotency

Task 執行必須區分：

- 可重複執行。
- 不可重複執行。
- 可安全恢復。
- 必須人工介入。

涉及資產帳務或外部副作用的操作，必須具備 Idempotency Key 或等效防重機制。

## 7.4 Retry Policy

重試策略至少包含：

- Maximum Attempts。
- Backoff。
- Retryable Errors。
- Non-retryable Errors。
- Dead Letter。
- Manual Intervention。

認證失敗應進入 `WAITING_FOR_AUTH`，不可持續重試直到資源耗盡。

## 7.5 Task Verification

Task 執行完成不等於任務成功。

應區分：

- Process Completed。
- Tool Execution Completed。
- Result Collected。
- Result Verified。
- Task Completed。

Headless Harness 即使回傳 Exit Code 0，也必須檢查工具事件、錯誤輸出、結果內容及必要的 Workspace Diff。

# 8. Schedule Management

## 8.1 Schedule and Task Separation

Schedule 負責觸發時間。

Task 負責工作執行。

兩者不能合併成同一套執行機制。

## 8.2 Schedule Lifecycle

```text
ACTIVE
   │
   ▼
DUE
   │
   ▼
TRIGGERED
   │
   ▼
TASK CREATED
   │
   ▼
NEXT RUN CALCULATED
```

## 8.3 Misfire Handling

Schedule 至少應明確定義：

- SKIP。
- RUN_ONCE。
- CATCH_UP。

各策略應有可驗證的語意：

| Policy | Expected Behavior |
|---|---|
| SKIP | 錯過的執行時間不補執行 |
| RUN_ONCE | 對錯過期間只補執行一次 |
| CATCH_UP | 依規則補處理符合條件的錯過執行 |

目前程式碼對 SKIP 有特殊處理，但其他策略是否完整實作尚不能確認。

應在修正時明確定義，不能只根據 Enum 名稱宣稱功能完成。

## 8.4 Duplicate Prevention

Schedule 必須避免：

- 同一時間點重複 Trigger。
- Runtime Restart 後重複建立 Task。
- 多個 Worker 同時建立相同 Task。
- Schedule 更新造成不必要的重複派發。

應透過資料庫交易、唯一識別及執行紀錄確保一致性。

## 8.5 Current Implementation Gap

目前：

- `CreateSchedule()` 已存在。
- `TriggerSchedule()` 已存在。
- `ProcessDueSchedules()` 已存在。
- `aia schedule tick` 可手動處理到期排程。
- Schedule Service 目前主要負責建立 Task。
- 尚未由常駐 Schedule Worker 持續輪詢。
- Schedule 建立 Task 後，尚未形成由 Background Task Worker 自動執行的完整鏈路。

這是 R1 必須完成的工作。

# 9. Autonomy Core Implementation

## 9.1 Core Interface

Autonomy Core 應提供通用的自主性循環入口。

示意：

```go
type AutonomyCore interface {
    Assess(ctx context.Context, input AssessmentInput) (*AssessmentResult, error)

    Discover(ctx context.Context, input DiscoveryInput) (*DiscoveryResult, error)

    Deliberate(ctx context.Context, input DeliberationInput) (*DeliberationResult, error)

    Decide(ctx context.Context, input DecisionInput) (*DecisionResult, error)

    Learn(ctx context.Context, input LearningInput) (*LearningResult, error)

    Reassess(ctx context.Context, input ReassessmentInput) (*ReassessmentResult, error)
}
```

此為目標介面，不代表目前已有完整實作。

實際設計應避免將所有推理工作集中在單一巨大方法中。

## 9.2 Initiative Interface

```go
type InitiativeEngine interface {
    Evaluate(ctx context.Context, input InitiativeInput) (*InitiativeResult, error)
}
```

Initiative Engine 必須能回傳不行動的結果。

## 9.3 Continuity Interface

```go
type ContinuityEngine interface {
    BuildContext(ctx context.Context, input ContinuityInput) (*ContinuityContext, error)

    Restore(ctx context.Context, input RestoreInput) (*ContinuityContext, error)
}
```

Continuity Engine 應整合 Goal、Memory、Task 及歷史 Decision。

## 9.4 Self-Correction Interface

```go
type SelfCorrectionEngine interface {
    Analyze(ctx context.Context, input EvaluationInput) (*CorrectionAnalysis, error)

    Propose(ctx context.Context, input CorrectionInput) (*CorrectionProposal, error)

    Validate(ctx context.Context, input ValidationInput) (*ValidationResult, error)
}
```

修正必須有版本與驗證紀錄。

## 9.5 Goal Evolution Interface

```go
type GoalEvolutionEngine interface {
    Evaluate(ctx context.Context, input GoalEvaluationInput) (*GoalEvaluation, error)

    Optimize(ctx context.Context, input GoalOptimizationInput) (*GoalProposal, error)

    Revise(ctx context.Context, input GoalRevisionInput) (*GoalProposal, error)
}
```

Goal Optimization 與 Goal Revision 必須區分。

一般方法優化可以在既有治理範圍內進行。

重大目標修正則必須進入 Approval 流程。

# 10. Goal Management

## 10.1 Goal Model

Goal 至少應保存：

```go
type Goal struct {
    ID          string
    ParentID    *string
    Owner       string
    Domain      string

    Title       string
    Description string

    Priority    int
    Status      string

    Version     int
    Constraints []string

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

以上為概念模型。

實際型別應與現有 Domain Entity 整合。

## 10.2 Goal Requirements

Goal Manager 應支援：

- 建立 Goal。
- 讀取 Goal。
- 更新 Goal。
- 建立子目標。
- 保存 Goal Version。
- 提出 Goal Proposal。
- 審核 Goal Proposal。
- 查詢 Goal History。
- 查詢關聯 Task。

## 10.3 Goal Persistence

Goal 不得只存在 Prompt 或 Session Context 中。

Runtime Restart 後必須能恢復目前有效目標。

# 11. Memory Management

## 11.1 Memory Service

Memory Service 應負責：

- 建立記憶。
- 查詢記憶。
- 更新記憶狀態。
- 建立關聯。
- 檢索相關記憶。
- 記憶整合。
- 來源追蹤。
- 版本管理。

## 11.2 Memory Model

```go
type Memory struct {
    ID        string
    AgentID   string
    Domain    string

    Type      string
    Content   string

    Source    string
    SourceID  string

    Confidence float64
    Relevance  float64

    Version   int

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

實際模型應依現有 Memory Entity 調整。

## 11.3 Retrieval Strategy

第一階段不必直接導入向量資料庫。

建議：

1. Metadata Filtering。
2. SQLite Full-Text Search。
3. Keyword Retrieval。
4. Semantic Retrieval。
5. Hybrid Retrieval。

Memory Retrieval 應能依 Goal、Domain、時間及相關性過濾。

## 11.4 Memory Integrity

記憶的來源、建立時間及版本必須可追溯。

Agent 不得將推測性內容直接保存為已確認事實。

# 12. Domain Agent Implementation

## 12.1 Registration

Agent 使用 Static Registration。

不使用 Runtime Go Plugin。

註冊流程：

```text
Application Bootstrap
       │
       ▼
Agent Registry
       │
       ▼
Register Domain Agents
       │
       ▼
Validate Agent Metadata
       │
       ▼
Expose Agent Capabilities
```

## 12.2 Domain Interface

Domain Agent 應提供通用能力，但不要求每個 Agent 都必須使用完全相同的內部服務。

建議基本介面：

```go
type DomainAgent interface {
    ID() string
    Name() string
    Domain() string

    Capabilities() []Capability

    Observe(ctx context.Context, input ObservationInput) (*Observation, error)

    Assess(ctx context.Context, input AssessmentInput) (*Assessment, error)

    Execute(ctx context.Context, input ExecutionInput) (*ExecutionResult, error)

    Verify(ctx context.Context, input VerificationInput) (*VerificationResult, error)
}
```

Domain Agent 的具體分析方法應留在自己的 Domain Package。

## 12.3 Investment Agent

目前唯一已建立的 Domain Module。

主要服務包括：

- Market Data Collection。
- Research Snapshot。
- Portfolio。
- Investment Decision。
- Prediction。
- Evaluation。
- Virtual Trading（依實際完成狀態逐步補足）。

Investment Agent 應使用共通 Autonomy Core。

不可將 Investment 專屬的研究及交易邏輯寫進 Autonomy Core。

## 12.4 Coding Agent

尚未實作。

未來提供：

- Repository Observation。
- Code Analysis。
- Debugging。
- Test Execution。
- Refactoring。
- Code Verification。

## 12.5 Writing Agent

尚未實作。

未來提供：

- Project Outline。
- Character Management。
- Chapter Progress。
- Historical Research。
- Timeline Validation。
- Manuscript Revision。
- Continuity Verification。

# 13. Agent Harness Adapter

## 13.1 Purpose

Harness Adapter 隔離 AIA 與 Antigravity CLI 的互動方式。

AIA 不應讓 Domain Agent 直接依賴 CLI 命令列細節。

## 13.2 Interface

示意：

```go
type Harness interface {
    Execute(ctx context.Context, request ExecutionRequest) (*ExecutionResponse, error)

    Cancel(ctx context.Context, executionID string) error

    Status(ctx context.Context, executionID string) (*ExecutionStatus, error)
}
```

實際介面應依目前 Antigravity CLI 1.4.12 的可用能力設計。

## 13.3 Execution Request

應包含：

- Task ID。
- Agent ID。
- Session ID。
- Prompt。
- Workspace。
- Tool Permissions。
- Execution Mode。
- Timeout。
- Context References。

## 13.4 Execution Response

應包含：

- Execution ID。
- Process Status。
- Exit Code。
- Tool Events。
- Standard Output。
- Standard Error。
- Result。
- Workspace Changes。
- Authentication Status。

## 13.5 Authentication Failure

遇到認證失效：

1. 停止相關 Task。
2. 保存目前執行狀態。
3. Task 進入 `WAITING_FOR_AUTH`。
4. 通知使用者。
5. 使用者透過 Harness 完成重新登入。
6. AIA 檢查認證狀態。
7. 恢復 Task。

AIA 不得自行讀取或管理 OAuth Token。

## 13.6 Permission

不得使用 `--dangerously-skip-permissions`。

AIA 必須同時遵守：

- AIA Policy。
- Harness Permission。
- Workspace Boundary。
- 執行環境限制。

# 14. Collector & External Data

## 14.1 Collector Architecture

Collector 負責外部資料收集。

它不是 Autonomy Core。

```text
Investment Agent
       │
       ▼
Data Collection Service
       │
       ▼
Collector Manager
       │
       ├── TWSE Provider
       ├── TPEx Provider
       ├── MOPS Provider
       ├── FRED Provider
       ├── SEC Provider
       └── News Provider
```

## 14.2 Provider Interface

```go
type Provider interface {
    ID() string
    Name() string

    HealthCheck(ctx context.Context) error

    Collect(ctx context.Context, request CollectRequest) (*CollectResult, error)
}
```

## 14.3 Data Quality

Provider 必須能區分：

- REAL。
- MOCK。
- FALLBACK。
- STALE。
- INVALID。

不可將固定行情或虛構公告混入正式研究資料。

## 14.4 Current Provider Issues

目前程式碼已有 Collector Interface、Manager 及 TWSE、TPEx、FRED、SEC Provider 結構。

但目前已發現：

- TWSE Provider 在資料取得失敗時會使用固定價格 fallback。
- TWSE Provider 會生成虛構公告。
- TPEx Provider 使用固定行情及虛構公告，尚未真正取得 TPEx 行情。
- FRED Provider 使用固定利率資料，尚未真正取得 FRED API 資料。
- SEC Provider 使用硬編碼申報資料，尚未真正取得 SEC 資料。
- 多個 Provider 的 HealthCheck 無法驗證真實連線。
- 部分資料解析錯誤未被完整處理。

這些項目必須修正後，才能宣稱真實資料整合完成。

# 15. Persistence & Repository

## 15.1 Database

第一階段使用 SQLite。

資料庫初始化應由 App 負責。

Repository 負責封裝 SQL 存取。

## 15.2 Repository Interface

Repository 應依 Domain 或 Entity 分離。

例如：

```go
type TaskRepository interface {
    Create(ctx context.Context, task *Task) error
    Get(ctx context.Context, id string) (*Task, error)
    Update(ctx context.Context, task *Task) error

    ClaimReady(ctx context.Context, workerID string) (*Task, error)

    ListReady(ctx context.Context, limit int) ([]*Task, error)
}
```

這是目標介面示意，應與現有 Repository API 整合。

## 15.3 Transaction

涉及多個 Entity 的狀態變更時，應使用 Database Transaction。

尤其包括：

- Task Claim。
- Schedule Trigger。
- Virtual Trade。
- Portfolio Ledger。
- Goal Version。
- Approval。
- Event Persistence。

## 15.4 Migration

Migration 必須：

- 具備版本號。
- 可追蹤執行狀態。
- 可重複啟動而不重複執行。
- 保留既有資料。
- 避免破壞既有資料模型。

既有 Migration Registry 應持續沿用。

新增 Migration 不得假設使用者資料庫是空的。

# 16. Event Architecture

## 16.1 Event Bus

Event Bus 負責程序內服務間的事件傳遞。

重要事件應另行持久化。

## 16.2 Event Types

建議事件：

```text
RuntimeStarted
RuntimeStopped
WorkerStarted
WorkerStopped

GoalCreated
GoalUpdated
GoalProposalCreated
GoalProposalApproved

InitiativeDetected
DecisionCreated

TaskCreated
TaskClaimed
TaskStarted
TaskCompleted
TaskFailed

MemoryCreated
MemoryUpdated

ExecutionStarted
ExecutionCompleted
ExecutionFailed

VerificationCompleted
CorrectionProposed
StrategyUpdated

ApprovalRequired
ApprovalGranted
ApprovalRejected

DomainObservationCompleted
```

## 16.3 Event Reliability

必須考慮：

- Duplicate Event。
- Lost Event。
- Retry。
- Event Ordering。
- Idempotency。
- Event Persistence。

不能只依賴記憶體內 EventBus 保存重要歷史事件。

# 17. Policy, Approval & Audit

## 17.1 Policy Service

Policy Service 應在行動執行前檢查權限。

檢查項目包括：

- Agent Identity。
- Task Type。
- Action Type。
- Resource Scope。
- Workspace。
- External Access。
- Approval Requirement。

## 17.2 Approval Service

Approval 必須具備：

- Approval ID。
- Requester。
- Requested Action。
- Scope。
- Reason。
- Expiration。
- Status。
- Approver。
- Approved At。

## 17.3 Audit Service

Audit 至少記錄：

- Decision。
- Action。
- Actor。
- Timestamp。
- Goal。
- Task。
- Policy Result。
- Approval。
- Execution Result。
- Verification。
- Correction。

Audit 不得由 Agent 自行修改。

# 18. Runtime Configuration

## 18.1 Configuration Categories

設定應區分：

- Runtime。
- Database。
- Web。
- Harness。
- Worker。
- Schedule。
- Domain。
- Policy。
- Logging。

## 18.2 Worker Configuration

Worker 應可設定：

- Enabled。
- Poll Interval。
- Concurrency。
- Timeout。
- Retry Policy。
- Shutdown Timeout。

不應將所有 Worker 的執行頻率寫死在程式碼中。

## 18.3 Resource Limits

AIA 應能限制：

- 同時執行 Task 數量。
- 同時執行 Harness Session 數量。
- 單次 Task Timeout。
- LLM 呼叫預算。
- 外部資料收集頻率。
- Worker 重試次數。

具體數值應由設定檔管理。

# 19. CLI Implementation

## 19.1 CLI Responsibilities

CLI 提供：

- Runtime 管理。
- Agent 管理。
- Task 管理。
- Goal 管理。
- Memory 管理。
- Schedule 管理。
- Investment 管理。
- Web Server 管理。
- System Status。

## 19.2 Runtime Commands

建議提供：

```text
aia runtime start
aia runtime stop
aia runtime status
```

這些是目標命令介面，須依目前 CLI 架構實作。

## 19.3 Server Commands

必須清楚區分：

```text
aia server start
```

與：

```text
aia runtime start
```

Server 是 HTTP Host。

Runtime 是背景自主運作環境。

未來可以提供整合模式：

```text
aia start
```

同時啟動 Runtime 與 Web Server。

但仍應保留兩者獨立管理的能力。

## 19.4 Current CLI

目前已存在的主要命令包含：

- `aia agent`
- `aia chat`
- `aia investment`
- `aia schedule`
- `aia server`
- `aia version`

Task、Memory、Goal、Runtime 等命令應依實際程式碼逐步補齊。

# 20. Web Integration

## 20.1 Web Host

Web Host 負責：

- HTTP Server。
- Authentication。
- Session。
- CSRF。
- Console Routes。
- Domain Web Modules。

## 20.2 Runtime Independence

Web Host 不應負責直接執行所有背景工作。

即使 Web Server 停止，Runtime 的生命週期設計也應允許獨立管理。

第一階段可使用同一個 Go Process，但必須有獨立的 Start、Stop 與 Status。

## 20.3 Web Module

Domain Web Module 使用隔離的 Mux。

Investment Module 可以註冊自己的 API，但不得直接操作其他 Domain 的內部資料。

# 21. Logging & Observability

## 21.1 Logging

Log 應具備：

- Timestamp。
- Level。
- Component。
- Worker ID。
- Task ID。
- Agent ID。
- Execution ID。
- Error。
- Duration。

敏感資訊不得寫入 Log。

## 21.2 Runtime Metrics

至少監控：

- Runtime Uptime。
- Worker Status。
- Worker Restart Count。
- Task Queue Length。
- Task Execution Duration。
- Task Failure Rate。
- Schedule Delay。
- Recovery Count。
- Harness Execution Failure。
- Memory Retrieval Duration。

## 21.3 Autonomy Metrics

逐步加入：

- Initiative Count。
- Initiative Precision。
- Autonomous Task Count。
- No-Action Count。
- Continuity Recovery Success。
- Memory Utilization。
- Correction Proposal Count。
- Correction Validation Success。
- Goal Optimization Count。
- Goal Revision Proposal Count。

這些指標必須由實際事件及結果計算，不可只依賴模型自我評分。

# 22. Error Handling

## 22.1 Error Categories

| Category | Handling |
|---|---|
| Configuration | Startup Failure |
| Database | Retry or Runtime Degraded |
| Harness | Task Retry / WAITING_FOR_AUTH |
| Network | Retry with Backoff |
| Provider | Mark Degraded |
| Policy | Reject Action |
| Approval | WAITING_FOR_APPROVAL |
| Task | Retry / Failed |
| Worker | Supervisor Recovery |
| Data Integrity | Reject Invalid Data |

## 22.2 Error Propagation

錯誤應保留：

- Error Type。
- Original Error。
- Context。
- Retryability。
- Related Entity。
- Timestamp。

不可為了讓流程繼續而默默忽略重要錯誤。

## 22.3 Partial Failure

部分功能失效時，Runtime 可以進入 DEGRADED。

例如：

- Web 正常，但 Investment Provider 失效。
- Runtime 正常，但 Harness 認證失效。
- Investment 正常，但外部新聞來源失效。

不得讓單一非核心 Provider 的錯誤無條件終止整個 AIA。

# 23. Testing Requirements

本章定義 Go Runtime 的主要測試範圍。

完整測試程序及驗收條件以 AIA-04 v2.0 為準。

## 23.1 Runtime Tests

- [ ] Runtime Start。
- [ ] Runtime Stop。
- [ ] Graceful Shutdown。
- [ ] Worker Startup。
- [ ] Worker Shutdown。
- [ ] Worker Failure Recovery。
- [ ] Runtime Restart。
- [ ] Recovery。

## 23.2 Task Tests

- [ ] Ready Task Selection。
- [ ] Atomic Claim。
- [ ] Duplicate Prevention。
- [ ] Retry。
- [ ] WAITING_FOR_AUTH。
- [ ] WAITING_FOR_APPROVAL。
- [ ] Execution Result。
- [ ] Verification。

## 23.3 Schedule Tests

- [ ] Due Schedule。
- [ ] Schedule Trigger。
- [ ] Task Creation。
- [ ] RUN_ONCE。
- [ ] SKIP。
- [ ] CATCH_UP。
- [ ] Duplicate Trigger Prevention。
- [ ] Restart Recovery。

## 23.4 Autonomy Tests

- [ ] Initiative Assessment。
- [ ] No-Action Decision。
- [ ] Memory Retrieval。
- [ ] Continuity Recovery。
- [ ] Decision Traceability。
- [ ] Self-Correction。
- [ ] Strategy Versioning。
- [ ] Goal Persistence。
- [ ] Goal Optimization。
- [ ] Goal Revision Approval。

## 23.5 Domain Tests

- [ ] Investment Agent Integration。
- [ ] Real Data Validation。
- [ ] Mock / Real Isolation。
- [ ] Portfolio Ledger。
- [ ] Virtual Trading。
- [ ] Prediction Evaluation。

Coding 與 Writing 尚未實作，不列入目前版本的已完成驗收項目。

# 24. Current Implementation Assessment

依目前 Go Source Archive 的程式碼檢視，現況如下。

| Component | Status | Notes |
|---|---|---|
| App Bootstrap | IMPLEMENTED | 已有 NewApp |
| SQLite | IMPLEMENTED | 已有資料庫初始化 |
| Migration Registry | IMPLEMENTED | 持續擴充 |
| Repository | IMPLEMENTED / PARTIAL | 依 Domain 檢查 |
| EventBus | IMPLEMENTED / PARTIAL | 程序內事件 |
| Agent Registry | IMPLEMENTED / PARTIAL | 靜態註冊基礎 |
| Harness Integration | PARTIAL | 需驗證完整執行結果 |
| Task Service | PARTIAL | 有管理及執行基礎 |
| Task Scheduler | PARTIAL | 有 RunTask / ProcessReadyTasks |
| Schedule Service | PARTIAL | 可建立 Task |
| Schedule Polling Worker | NOT IMPLEMENTED | 尚無常駐輪詢 |
| Task Background Worker | NOT IMPLEMENTED | 尚無常駐執行 |
| Runtime Supervisor | NOT IMPLEMENTED | 尚未形成完整管理 |
| Runtime Lifecycle | PARTIAL | Web Server 有獨立生命週期 |
| Recovery | PARTIAL | NewApp 啟動時執行一次 |
| Memory | PARTIAL | 尚未形成完整 Continuity |
| Goal Management | PARTIAL / VERIFY | 需依現有資料模型確認 |
| Initiative Engine | NOT IMPLEMENTED |
| Self-Correction Engine | NOT IMPLEMENTED |
| Goal Evolution Engine | NOT IMPLEMENTED |
| Investment Module | PARTIAL |
| Coding Module | NOT IMPLEMENTED |
| Writing Module | NOT IMPLEMENTED |

## 24.1 Important Runtime Findings

目前 `internal/cli/web.go` 的 `runWebServer()`：

- 建立 Web Host。
- 建立 HTTP Server。
- 啟動 `ListenAndServe()`。
- 等待 SIGINT / SIGTERM。
- 呼叫 `Shutdown()`。

但沒有啟動：

- Schedule Polling Worker。
- Task Background Worker。
- Initiative Worker。
- Investment Observation Worker。
- Prediction Evaluation Worker。

因此：

**目前 `aia server start` 不等於完整 AIA Runtime。**

另外，`ProcessDueSchedules()` 目前只負責處理到期 Schedule 並建立 Task，並不會自動呼叫 `Scheduler.RunTask()`。

這兩個差距必須在 R1 修正。

# 25. Implementation Roadmap

## R0：Code & Architecture Alignment

目標：讓現有 Go 程式碼與新架構建立一致責任邊界。

工作：

- 確認現有 Package。
- 確認 Service 依賴。
- 確認 Repository 邊界。
- 確認 Agent Registry。
- 確認 Harness Adapter。
- 確認 Task State。
- 確認 Schedule Misfire 行為。
- 確認 Migration。
- 統一文件與程式碼狀態標記。

原則：不做沒有必要的目錄搬移。

## R1：Persistent Runtime

目標：建立真正的長期運作環境。

工作：

1. 建立 Runtime Lifecycle。
2. 建立 Runtime Supervisor。
3. 建立 Task Worker。
4. 建立 Schedule Worker。
5. 串接 Schedule → Task → RunTask。
6. 建立 Graceful Shutdown。
7. 建立 Worker Health。
8. 建立 Worker Recovery。
9. 建立 Task Claim。
10. 建立 Idempotency。

驗收：

啟動 Runtime 後，Schedule 可以自動建立 Task，Task Worker 可以自動執行 Task，程序重新啟動後能恢復必要工作。

## R2：Continuity & Memory

目標：建立跨時間延續能力。

工作：

- Goal Persistence。
- Memory Retrieval。
- Memory Consolidation。
- Context Builder。
- Historical Decision Retrieval。
- Runtime Restart Recovery。
- Outstanding Task Recovery。

驗收：

Agent 能在 Runtime Restart 後，根據過去目標、記憶與未完成工作延續原有工作。

## R3：Initiative Engine

目標：建立主動性。

工作：

- Environment Observation。
- Goal Assessment。
- Initiative Evaluation。
- Discovery。
- Autonomous Task Proposal。
- No-Action Decision。
- Initiative Audit。

驗收：

Agent 能主動發現值得處理的問題，而不是只執行使用者指令或定時固定產生任務。

## R4：Reasoning & Self-Correction

目標：建立可驗證的自我修正。

工作：

- Deliberation。
- Decision Record。
- Prediction Evaluation。
- Error Analysis。
- Correction Proposal。
- Strategy Versioning。
- Forward Validation。

驗收：

Agent 能根據歷史結果發現判斷問題、提出修正並驗證修正效果。

## R5：Goal Evolution

目標：建立長期目標管理及演化。

工作：

- Goal Persistence。
- Goal Optimization。
- Goal Revision Proposal。
- Goal Conflict Resolution。
- Approval Integration。
- Goal History。

驗收：

Agent 能維持既定目標，主動優化方法，並在必要時提出目標修正。

## R6：Autonomous Domain Operation

目標：完成第一個完整自主性實驗。

工作：

- Investment Data Reliability。
- Portfolio Ledger。
- Virtual Trading。
- Daily Observation。
- Autonomous Investment Decision。
- Strategy Correction。
- Monthly Review。

之後再將共通能力擴展到 Coding 與 Writing。

# 26. Implementation Constraints

## 26.1 Single User

第一階段不實作多使用者帳號、租戶及 RBAC。

## 26.2 Local Only

Web Host 預設使用 `127.0.0.1`。

## 26.3 Static Registration

Agent 與 Web Module 採靜態註冊。

不使用 Runtime Go Plugin。

## 26.4 No Real Financial Trading

Investment Agent 僅使用虛擬投資組合。

## 26.5 No Permission Bypass

不得使用 `--dangerously-skip-permissions`。

## 26.6 OAuth Credential Boundary

OAuth 由使用者透過 Harness 管理。

AIA 不保存或解析 OAuth Token。

## 26.7 Preserve Audit

所有重要自主性決策及修正必須可追溯。

# 27. Final Implementation Principle

AIA 的 Go Runtime 不應只是將 CLI、Web、Task、Schedule 及 Agent 整合在一起。

它必須提供一個能夠長期運作的環境，讓 Autonomy Core 有機會持續觀察、思考、行動、驗證及修正。

實作時必須清楚區分：

- Runtime 負責持續運作。
- Autonomy Core 負責自主性。
- Memory 負責跨時間的經驗與脈絡。
- Goal Manager 負責目標。
- Task Manager 負責工作生命週期。
- Domain Agent 負責領域知識及行動。
- Harness 負責模型及工具執行。
- Policy 與 Audit 負責治理。

最重要的是：

**不要把「程式能持續執行」誤認為「Agent 已經具備主動性」；不要把「能保存記憶」誤認為「具有時間連續性」；不要把「能修改 Prompt」誤認為「能自我修正」。**

這些能力必須透過實際行為、歷史紀錄及可重複驗證的實驗來證明。
