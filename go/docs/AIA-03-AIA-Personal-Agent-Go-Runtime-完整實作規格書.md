# AIA-03：Go Runtime Implementation Specification

**文件編號：** AIA-03  
**版本：** v1.1  
**狀態：** Implementation Baseline  
**語言：** Go  
**資料庫：** SQLite  
**Agent Harness：** Antigravity CLI（agy）  
**前置文件：** AIA-01、AIA-02  
**相關文件：** AIA-04、AIA-05、AIA-06

---

# 1. 文件目的

本文件定義 AIA Runtime 的 Go 實作規格。

AIA Runtime 是整個系統的核心，負責：

- Task Management。
- Agent Registry。
- Agent Lifecycle。
- Execution Engine。
- Dependency Management。
- Policy Enforcement。
- Memory Management。
- Recovery。
- Audit。
- Scheduler。
- Approval。
- Embedded Web Host。
- CLI。
- Domain Service Integration。

本版本新增 Embedded Web Host、Agent Web Module、HTTP API、SSE、Web Security 與前端靜態資源嵌入規格。

# 2. 技術基準

| 項目 | 技術 |
|---|---|
| Language | Go |
| Database | SQLite |
| Database Mode | WAL |
| HTTP Server | net/http |
| Static Assets | embed.FS |
| API | REST / JSON |
| Event Stream | SSE |
| CLI | Go CLI |
| Harness | agy CLI |
| Logging | Structured Logging |
| Configuration | YAML 或 TOML，需統一 |
| Migration | Versioned SQL Migration |
| Testing | go test |
| Race Detection | go test -race |

Go 版本以專案建置設定及 CI 實際驗證版本為準，不得將開發機版本視為所有部署環境的最低版本。

## 2.1 實作範疇與版本完成度矩陣 (Version Maturity Baseline)

AIA 目前處於 **v1.0 (Core & Web Baseline)** 實作交付階段，嚴格區分本輪已交付範圍與下一階段規劃：

| 模組代號 | 範圍與能力 | 實作狀態 | 說明 |
|---|---|---|---|
| **P1** | Web 基礎設施 | **IMPLEMENTED** | Web Host、一次性 Token 登入兌換、Server-Side Session、HTTP API、SSE 事件串流、安全機制（CSRF、Origin、Host Header） |
| **P2** | Global Console | **IMPLEMENTED** | Dashboard、Task 建立/執行/取消/恢復、Agent 狀態檢視與啟用/停用、Memory 檢視與 Candidate、待處理事項（Pending Review）、Audit Trail |
| **P3** | Investment 核心 | **IMPLEMENTED** | 投資組合 Profile/Snapshot、資金限制檢查、Go 建議股數計算、交易手續費與稅費計算（`CostConfig`）、Trade Proposal 產生與超限駁回 |
| **P6** | Agent UI 模組 | **IMPLEMENTED** | 原生前端（HTML/CSS/JS 無外部打包）、Agent 專屬 mux 隔離路由、Portfolio 檢視、Market Data 檢視、Calibration Scorecard |
| **P4** | 實際資料抓取 | **IMPLEMENTED** | TWSE OpenAPI 日 K 數據抓取、公開資訊觀測站 (MOPS) 重大訊息與新聞擷取、限流保護與快取機制 |
| **P5** | 預測與評估 | **IMPLEMENTED** | 歷史預測追蹤、真實行情到期結算、Brier Score 信心度校準計算、方向性準確率 (Directional Accuracy) 評分卡 |
| **SCHED** | 定時排程 Cron | **IMPLEMENTED** | 標準五欄式 Cron 與別名解析、Misfire Policy（SKIP, RUN_ONCE, CATCH_UP）、排程自動派發 Task |
| **CLI-INV** | 頂層投資 CLI | **IMPLEMENTED** | `aia investment` 命令列套件（`portfolio`, `sync`, `proposals`, `approve`, `reject`, `predictions`, `evaluate`） |
| **CLI-SCHED**| 頂層排程 CLI | **IMPLEMENTED** | `aia schedule` 命令列套件（`list`, `create`, `trigger`, `tick`） |

# 3. Repository Structure

建議目錄：

```text
aia/
├── cmd/
│   └── aia/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── bootstrap.go
│   │   ├── services.go
│   │   └── shutdown.go
│   │
│   ├── runtime/
│   │   ├── runtime.go
│   │   ├── lifecycle.go
│   │   ├── registry.go
│   │   ├── scheduler.go
│   │   └── eventbus.go
│   │
│   ├── task/
│   ├── execution/
│   ├── dependency/
│   ├── policy/
│   ├── approval/
│   ├── memory/
│   ├── recovery/
│   ├── audit/
│   │
│   ├── harness/
│   │   ├── interface.go
│   │   └── agy/
│   │       ├── adapter.go
│   │       ├── process.go
│   │       ├── stream.go
│   │       └── parser.go
│   │
│   ├── web/
│   │   ├── host.go
│   │   ├── router.go
│   │   ├── middleware.go
│   │   ├── session.go
│   │   ├── csrf.go
│   │   ├── api/
│   │   ├── events/
│   │   └── assets/
│   │
│   ├── cli/
│   ├── project/
│   ├── workspace/
│   ├── config/
│   ├── database/
│   ├── logging/
│   │
│   ├── agents/
│   │   ├── personal/
│   │   ├── coding/
│   │   ├── writing/
│   │   ├── investment/
│   │   └── video/
│   │
│   └── shared/
│
├── web/
│   ├── console/
│   └── agents/
│       ├── investment/
│       ├── writing/
│       ├── coding/
│       └── video/
│
├── migrations/
├── configs/
├── docs/
├── scripts/
├── go.mod
└── README.md
```

此目錄為邏輯架構建議。實際目錄可依現有程式碼調整，但不得破壞模組責任邊界。

# 4. Core Runtime Interfaces

## 4.1 Agent

```go
type Agent interface {
    ID() string
    Name() string
    Capabilities() []Capability
    Execute(
        ctx context.Context,
        task AgentTask,
    ) (AgentResult, error)
}
```

Agent 不應直接操作全域資料表。

## 4.2 Agent Registry

```go
type AgentRegistry interface {
    Register(agent Agent) error
    Get(id string) (Agent, error)
    List() []AgentDescriptor
    Enable(id string) error
    Disable(id string) error
}
```

註冊時必須檢查：

- ID 不可重複。
- Capability 合法。
- Agent Configuration Schema 合法。
- UI Module 若存在，Agent ID 必須一致。

## 4.3 Harness Interface

```go
type Harness interface {
    Name() string
    Version(ctx context.Context) (string, error)
    HealthCheck(ctx context.Context) error
    Execute(
        ctx context.Context,
        req ExecuteRequest,
    ) (ExecuteResult, error)
    Cancel(
        ctx context.Context,
        executionID string,
    ) error
}
```

Harness Adapter 必須處理：

- Process Start。
- Process Exit。
- stdout / stderr。
- stream-json。
- Timeout。
- Cancellation。
- Authentication Failure。
- Soft-deny。
- Workspace Diff。
- Result Artifact。

## 4.4 Execution Result

```go
type ExecuteResult struct {
    ExecutionID string
    ExitCode    int
    Status      ExecutionStatus
    Events      []HarnessEvent
    Stdout      string
    Stderr      string
    Artifacts   []Artifact
    WorkspaceDiff []WorkspaceChange
    StartedAt   time.Time
    FinishedAt  time.Time
}
```

敏感資訊必須在 Log 與 API 輸出前過濾。

# 5. Task Engine

## 5.1 Task Model

```go
type Task struct {
    ID          string
    ProjectID   string
    ParentID    string
    AgentID     string
    Title       string
    Description string
    Status      TaskStatus
    Priority    int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 5.2 Task Lifecycle

Task 狀態由 Runtime 控制。正式 Task State Machine 以此為唯一依據（嚴禁使用未定義之 QUEUED）：

```text
CREATED
READY
RUNNING
WAITING_FOR_DEPENDENCY
WAITING_APPROVAL
WAITING_FOR_AUTH
RETRYING
VALIDATING
COMPLETED
FAILED
CANCELLED
```

狀態轉移必須透過 Task Service。

禁止 Domain Agent 直接更新 Task Status。

## 5.3 Execution Model

Task 與 Execution 分離。

```go
type Execution struct {
    ID          string
    TaskID      string
    AgentID     string
    Harness     string
    Status      ExecutionStatus
    Attempt     int
    StartedAt   time.Time
    FinishedAt  *time.Time
}
```

一個 Task 可有多次 Execution。

每次 Execution 都必須保存獨立紀錄。

## 5.4 Result Validation

```go
type ResultValidator interface {
    Validate(
        ctx context.Context,
        task Task,
        result ExecuteResult,
    ) (ValidationResult, error)
}
```

驗證項目依 Task Type 決定。

不得因 Exit Code = 0 直接完成 Task。

Validation 未完成時，Task 不得進入 COMPLETED。

# 6. Recovery

## 6.1 Recovery Manager

```go
type RecoveryManager interface {
    Recover(ctx context.Context) error
    Resume(ctx context.Context, taskID string) error
    Retry(ctx context.Context, taskID string) error
}
```

## 6.2 Recovery Rules

| 情況 | 處理 |
|---|---|
| Runtime Crash | 依持久化狀態恢復 |
| Execution Running | 判斷是否仍存在或已中斷 |
| SUCCESS 未 Validation | 回到 VALIDATING |
| Auth Failure | WAITING_FOR_AUTH |
| Retry Limit | FAILED |
| Approval Required | WAITING_APPROVAL |
| Dependency Pending | WAITING_FOR_DEPENDENCY |
| Cancel Requested | 執行取消流程 |

## 6.3 Idempotency

對每個會產生副作用的操作保存：

- Operation ID。
- Idempotency Key。
- Task ID。
- Execution ID。
- Operation Type。
- External Reference。
- Status。
- Created At。
- Reconciled At。

外部副作用發生後，不得單憑本地狀態未知就盲目重做。

## 6.4 External Operation Journal

未來外部操作需支援：

```text
PREPARED
SUBMITTED
CONFIRMED
FAILED
UNKNOWN
RECONCILED
```

UNKNOWN 必須進入 Reconciliation，不得直接重送。

# 7. Policy Engine

```go
type PolicyEngine interface {
    Evaluate(
        ctx context.Context,
        action Action,
    ) (PolicyDecision, error)
}
```

Policy Decision：

- ALLOW。
- REQUIRE_APPROVAL。
- DENY。

Risk：

- LOW。
- MEDIUM。
- HIGH。
- CRITICAL。

Policy 必須在實際執行前檢查。

CLI、Web UI、Domain Agent 均不得繞過。

agy 底層權限仍然必須保留。

# 8. Memory Manager

```go
type MemoryManager interface {
    Propose(ctx context.Context, candidate MemoryCandidate) error
    Approve(ctx context.Context, candidateID string) error
    Reject(ctx context.Context, candidateID string) error
    Search(ctx context.Context, query MemoryQuery) ([]Memory, error)
}
```

Memory Candidate 未核准前不得進入正式 Memory。

Memory 必須保存：

- Source。
- Created At。
- Approved At。
- Confidence。
- Scope。
- Version。
- Status。

# 9. Database

## 9.1 SQLite

設定：

- WAL。
- Foreign Keys。
- Busy Timeout。
- Migration Version。
- Transaction。
- Backup。
- Integrity Check。

DB Access 必須經 Repository。

## 9.2 Transaction

以下操作應使用一致性交易：

- 建立 Task 與初始狀態。
- Execution 狀態更新。
- Approval 狀態更新。
- Memory 核准。
- Audit Record 寫入。
- Outbox Event 寫入。

重要狀態變更與通知發布應採 Transactional Outbox 或等效的一致性設計。

## 9.3 Migration

每次 Schema 變更必須新增 Migration。

不得直接修改已發布 Migration。

Migration 必須支援：

- Version。
- Up。
- Down（若可安全回退）。
- Integrity Check。
- Backup Requirement。

# 10. Event Bus

```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(
        ctx context.Context,
        eventType string,
    ) (<-chan Event, error)
}
```

事件至少包括：

- task.created
- task.updated
- task.completed
- task.failed
- task.waiting_approval
- task.waiting_auth
- execution.started
- execution.completed
- agent.status_changed
- system.health_changed

Event Bus 是內部通知機制，不是唯一資料來源。

# 11. Embedded Web Host

## 11.1 目標

Runtime 提供單一 HTTP Server。

Domain Agent 不得各自啟動 HTTP Server。

Web Host 不得承擔 Domain Business Logic。

## 11.2 Web Host Interface

```go
type WebHost interface {
    Start(ctx context.Context) error
    Shutdown(ctx context.Context) error
    Handler() http.Handler
    Address() string
}
```

Host 必須具備：

- HTTP Server。
- Router。
- Middleware。
- Static Assets。
- REST API。
- SSE。
- Authentication。
- Authorization。
- CSRF。
- Request Logging。
- Health Check。

## 11.3 Agent Web Module

```go
type AgentWebModule interface {
    AgentID() string
    RegisterRoutes(mux *http.ServeMux) error
    NavItems() []NavItem
}
```

其中 `mux` 必須是 Web Host 為該 Agent 專屬建立的獨立 ServeMux，Agent 絕不接觸全域 mux。

Host 負責：
- 建立專屬 mux。
- 驗證 Agent ID。
- 將專屬 mux 掛載到命名空間 `/api/v1/agents/{agent_id}/`。
- 防止路由衝突。
- 套用統一的 Authentication、Authorization、Audit Middleware。
Agent 不得註冊或覆蓋全域保留路徑。

## 11.4 Web Context

```go
type AgentWebContext struct {
    AgentID    string
    Services   AgentServices
    Authorizer Authorizer
    Audit      AuditWriter
    Events     EventSubscriber
    Logger     Logger
}
```

實際注入時必須遵守最小權限原則。

不得傳入：

- Raw Database Handle。
- 全域 Credential Store。
- agy OAuth Token。
- 其他 Agent 的 Repository。
- 未受限制的 Global Runtime Pointer。

## 11.5 Module Registry

```go
type AgentWebRegistry interface {
    Register(module AgentWebModule) error
    Get(agentID string) (AgentWebModule, error)
    List() []AgentWebDescriptor
}
```

註冊檢查：

1. Agent 必須存在於 Agent Registry。
2. Agent ID 必須一致。
3. Route Namespace 必須合法。
4. 不得覆蓋保留路由。
5. Nav Item 不得重複。
6. Module 註冊失敗不得使整個 Runtime Crash。

第一階段採編譯期靜態註冊，不實作任意 Go Plugin 動態載入。

## 11.6 Route Namespace

全域路由：

```text
/
/console/
/api/v1/
/events/
/healthz
/readyz
```

Agent 路由：

```text
/agents/{agent_id}/
/api/v1/agents/{agent_id}/
```

所有 Agent Route 必須由 Host 統一掛載。

## 11.7 Static Assets & Frontend Technology

Web UI 靜態資源由 Go embed 編入 binary。明確規定 embed package 位置為 `go/web/assets`，靜態資源位於 `go/web/assets/dist/`：

```go
package assets

import "embed"

//go:embed dist
var FS embed.FS
```

這樣 embed 指令只處理套件目錄底下的檔案，不會遇到跨目錄嵌入問題。

### 前端技術決策：選原生 HTML / CSS / JavaScript
1. **不使用 Vite，不引入 React、Vue 等大型框架**。
2. AIA 是單機、單使用者的 Personal Agent，不需要大型前端工程體系。
3. 不需要 Node.js、npm、Vite 等外部建置工具與複雜依賴。
4. 部署只需要單一 Go 執行檔。
5. 必須注意 SPA fallback 不可攔截 API、SSE、Health Check 或其他 Agent Namespace。

## 11.8 HTTP Server

建議設定：

```go
http.Server{
    Addr:              "127.0.0.1:8080",
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      0,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    1 << 20,
}
```

SSE 不適合直接使用會中斷長連線的短 WriteTimeout。

實作時必須對 SSE 路徑採取合適的連線管理及寫入 Deadline 策略。

Request Body 必須另行限制大小。

Port 必須可配置，預設值可調整。

## 11.9 Shutdown

Shutdown 順序：

1. 停止接受新的 HTTP Request。
2. 關閉新 Task 建立入口。
3. 讓既有 HTTP Request 有限時間完成。
4. 關閉 SSE 連線。
5. Runtime 依自己的 Shutdown Policy 處理 Task。
6. 持久化狀態。
7. 關閉 Database。

Web Host Shutdown 不得直接取消所有 Task。

# 12. REST API

Prefix：

```text
/api/v1/
```

全域 API 至少提供：

```text
GET    /system/status
GET    /system/health
GET    /tasks
POST   /tasks
GET    /tasks/{id}
POST   /tasks/{id}/cancel
POST   /tasks/{id}/resume
GET    /agents
GET    /agents/{id}
GET    /memory
GET    /pending
POST   /pending/{type}/{id}/approve
POST   /pending/{type}/{id}/reject
GET    /audit
GET    /events
```

待處理事項整合：`/pending` 端點統一聚合等待核准的 Task（`WAITING_APPROVAL`）、Memory Candidate（`candidate`）以及投資交易提案（`PENDING`），不建立虛構的 Approval Entity。

Domain API：

```text
/api/v1/agents/{agent_id}/...
```

所有 API：

- 使用 JSON。
- 使用統一錯誤格式。
- 必須驗證輸入。
- 必須檢查 Permission。
- 必須有 Request ID。
- 寫入操作必須留下 Audit。
- 不得回傳內部敏感資訊。

# 13. Authentication & Authorization

## 13.1 第一階段

Web Host 預設監聽 127.0.0.1。

不得預設開放 LAN 或 Internet。

即使只監聽 localhost，也必須實作安全防護。

## 13.2 一次性登入 Token 與 Session 流程

AIA 採用高熵一次性登入 Token 與 Server-Side Session：

1. **啟動生成**：AIA 啟動時於記憶體中產生高熵（32 位元組，64 字元 Hex）的一次性登入 Token。
2. **終端呈現**：Token 僅於本機終端機命令列輸出一次，供使用者在本機登入頁輸入。
3. **單次兌換**：Server 接收登入請求驗證 Token 後，立即使該 Token 失效（One-Time Redemption），不可重複使用。
4. **建立 Session**：Server 建立具備閒置逾時（預設 30 分鐘）與絕對逾時（預設 12 小時）的 Server-Side Session。
5. **安全 Cookie**：使用 `HttpOnly`、`SameSite=Strict` Cookie（名稱 `aia_session`）保存 Session ID；不將 Session ID 放入 URL。
6. **每請求驗證**：每個 API Request 均重新驗證 Session 有效性、來源及權限。
7. **過期與登出**：Session 逾期或使用者點擊登出後，Session 立即作廢並清除 Cookie，再次登入需重新取得登入 Token。
8. **憑證保護**：登入 Token 具備 15 分鐘有效期限；驗證資料妥善保存於記憶體，嚴禁將明文 Token 寫入資料庫或 Log。不得將 agy OAuth Token 放入 Web Session 或前端。

## 13.3 CSRF

修改狀態的 API 必須驗證 CSRF。

必須檢查 Origin / Referer，並搭配自訂標頭檢查（如 `X-Requested-With: XMLHttpRequest` 或 CSRF Token）。

不得讓 GET Request 造成業務狀態變更。

## 13.4 Permission

採 Permission-based Authorization。

## 13.5 投資金額與決策責任邊界 (Investment Responsibility Boundary)

Go Runtime 與 AI Provider (agy) 之間具有不可妥協的責任邊界：

- **agy 職責**：專注於金融分析、市場研究、新聞解讀、趨勢判斷與理由說明（Reasoning）。
- **Go Runtime 職責**：
  1. 接收並驗證股票代號（Symbol/Exchange）是否存在且活躍。
  2. 驗證操作方向（BUY / SELL / HOLD）及信心度閾值。
  3. 依市場規則（如台股 1,000 股整股或零股）精確計算建議股數。
  4. 精確計算交易成本與稅費：Go 依交易方向、股數與估計市價計算手續費與證交稅。
     - **預設計算參數（Default Parameters）**：以現行台灣證券市場常態為基準（券商手續費率 `0.1425%`、最低手續費 `NT$20`、賣出證券交易稅率 `0.3%`）。
     - **參數可配置性（Configurability）**：Runtime 定義 `CostConfig` 結構（包含 `FeeRate`、`FeeDiscountRate`、`MinFee`、`TaxRate`），支援從投資組合設定檔（Profile `configuration_json`）或執行環境動態載入。允許彈性設定券商電子下單折讓率（如 6 折、2.8 折）、不同商品稅率（如 ETF 證交稅 0.1%、當沖降稅 0.15%）或未來法定費率調整，避免將常數硬編碼為不可變過時規則。
  5. 嚴格檢查可用現金（Available Cash）、投資組合資金上限（Capital Limit）、單一持倉比例上限（`max_position_pct`）及最低現金儲備比率（`min_cash_pct`）。
  6. 產生最終正式 Trade Proposal；超出限制時由 Go 負責拒絕或調整建議股數，並記錄駁回原因至 `investment_decision_rejections`。
  7. **嚴禁將 LLM 的直接輸出未經 Go 計算與限制驗證就作為有效交易提案交付。**

例如：

```text
system.read
system.manage
task.read
task.create
task.cancel
agent.read
agent.manage
memory.read
memory.approve
policy.read
policy.manage
audit.read
investment.read
investment.configure
investment.research
```

所有 API Request 必須執行授權檢查。

## 13.6 Security Headers

至少考慮：

- Content-Security-Policy。
- X-Content-Type-Options。
- Referrer-Policy。
- Frame-Ancestors。
- Cache-Control（敏感 API）。

CORS 採預設不開放跨來源，必要時採明確 Allowlist。

# 14. SSE

Endpoint：

```text
GET /api/v1/events
```

事件：

- task.created
- task.updated
- task.completed
- task.failed
- task.waiting_approval
- task.waiting_auth
- agent.status_changed
- system.health_changed
- investment.updated

必須支援：

- Event ID。
- Last-Event-ID。
- Reconnection。
- Heartbeat。
- Client Disconnect。
- Permission。
- Connection Limit。

SSE 只負責通知。

前端重新連線後必須以 REST API 重新取得正式狀態。

# 15. CLI

CLI 與 Web 必須共用 Application Service。

CLI 至少包含：

```text
aia doctor
aia status

aia task create
aia task list
aia task show
aia task cancel
aia task resume

aia agent list
aia agent show

aia memory list
aia memory approve
aia memory reject

aia server start
aia server status
aia server stop
```

Server Command 的實際生命週期及 Process Management 需與 Runtime Bootstrap 設計一致。

# 16. Configuration

設定項目至少包括：

```yaml
runtime:
  data_dir: "./data"
  workspace_dir: "./workspace"
  log_level: "info"

database:
  driver: "sqlite"
  path: "./data/aia.db"
  wal: true

harness:
  type: "agy"
  command: "agy"

web:
  enabled: true
  host: "127.0.0.1"
  port: 8080
  session_timeout_minutes: 60
  sse_enabled: true
```

實際設定 Schema 必須有版本及驗證。

不可將密碼、Token、API Key 寫入版本控制的設定檔。

# 17. Logging

使用 Structured Logging。

至少記錄：

- timestamp
- level
- component
- task_id
- execution_id
- agent_id
- request_id
- event
- error_code

敏感資料必須過濾。

# 18. Health Check

`aia doctor` 必須檢查：

- Go Runtime。
- Database。
- SQLite WAL。
- Workspace。
- Agent Registry。
- Harness Binary。
- Harness Version。
- Harness Health。
- Web Host Configuration。
- Embedded Assets。
- Agent Web Module Registration。

Web API：

```text
GET /healthz
GET /readyz
```

`healthz` 表示 Process 存活。

`readyz` 表示必要依賴已可提供服務。

# 19. Testing Requirements

必須支援：

```bash
go test ./...
go test -v -count=1 ./...
go test -race ./...
```

測試類型：

- Unit Test。
- Integration Test。
- Database Test。
- Harness Adapter Test。
- Recovery Test。
- Policy Test。
- API Test。
- Web Security Test。
- SSE Test。
- Agent Module Test。
- End-to-End Test。

測試不得依賴使用者真實投資帳戶或實際金融交易。

# 20. Implementation Milestones

## M0：Core Runtime

- Bootstrap。
- Config。
- Database。
- Migration。
- Logging。
- Agent Registry。

## M1：Task Engine

- Task。
- Execution。
- Dependency。
- Result Validation。

## M2：Policy / Approval / Memory

- Policy Engine。
- Approval。
- Memory Candidate。
- Audit。

## M3：Harness Integration

- agy Version。
- HealthCheck。
- stream-json。
- Process Lifecycle。
- Soft-deny。
- Authentication Failure。
- Workspace Diff。
- Result Validation。

## M4：Recovery

- Crash Recovery。
- Retry。
- Idempotency。
- Waiting for Auth。
- Validation Recovery。

## M5：Embedded Web Host

- HTTP Server。
- Static Assets。
- Global API。
- Health Check。
- Web Authentication。
- Authorization。
- CSRF。

## M6：Agent Web Module

- Module Interface。
- Registry。
- Namespace。
- Agent Navigation。
- Domain API。
- Agent UI。

## M7：SSE & Integration

- Event Stream。
- Reconnection。
- Task Updates。
- Audit Integration。
- CLI / Web Service Sharing。

## M8：Stabilization

- Race Test。
- Security Test。
- Crash Test。
- End-to-End。
- Release Packaging。

# 21. Acceptance Criteria

| ID | 驗收條件 |
|---|---|
| RT-01 | Runtime 可獨立啟動 |
| RT-02 | Agent Registry 可註冊及查詢 |
| RT-03 | Task Lifecycle 正常 |
| RT-04 | Execution 與 Task 分離 |
| RT-05 | Dependency 正常 |
| RT-06 | Policy 可阻止 Critical Risk |
| RT-07 | High Risk 可等待核准 |
| RT-08 | Memory Candidate 可核准 |
| RT-09 | Crash Recovery 正常 |
| RT-10 | Retry Limit 正常 |
| RT-11 | Auth Failure 可 WAITING_FOR_AUTH |
| RT-12 | User Re-auth 後可 Resume |
| RT-13 | SUCCESS 未 Validation 不會完成 Task |
| RT-14 | Idempotency 可防止重複執行 |
| RT-15 | agy Adapter 可取得版本與健康狀態 |
| RT-16 | stream-json 可解析 |
| RT-17 | Soft-deny 不會誤判成功 |
| RT-18 | Web Host 可啟動 |
| RT-19 | UI Assets 嵌入 Binary |
| RT-20 | 預設監聽 127.0.0.1 |
| RT-21 | AgentWebModule 可註冊 |
| RT-22 | Agent UI Namespace 隔離 |
| RT-23 | Global API 可操作 Task |
| RT-24 | Domain API 經 Service 執行 |
| RT-25 | SSE 可推送 Task Event |
| RT-26 | Web Host Shutdown 不影響既有 Task |
| RT-27 | Web 操作有 Audit |
| RT-28 | 未授權 API 被拒絕 |
| RT-29 | CSRF 測試通過 |
| RT-30 | go test ./... PASS |
| RT-31 | go test -race ./... PASS |

# 22. 實作約束

1. 不得讓 Web Handler 直接呼叫 Repository。
2. 不得讓 Domain Agent 直接更新 Global Task 狀態。
3. 不得讓 Agent UI 直接操作 SQLite。
4. 不得讓前端持有 agy Credential。
5. 不得只依 Exit Code 判斷 Execution 成功。
6. 不得因 Web Host 關閉而停止 Runtime Task。
7. 不得讓單一 Agent Module 失敗導致 Runtime 整體失效。
8. 不得將 SSE 當作正式狀態資料來源。
9. 不得以 localhost 作為免除安全檢查的理由。
10. 不得使用不安全的 agy 權限跳過參數。

# 23. 最終實作原則

AIA Runtime 必須保持為唯一的全域協調與治理核心。

CLI、Web UI 及未來 Client 都是 Runtime 的操作入口。

Domain Agent 提供領域服務與選擇性 UI Module。

所有操作最終必須經過 Application Service、Domain Service、Policy、Audit 與 Repository。

**核心執行邏輯不依賴任何特定 UI；UI 也不得成為 Runtime 的生命週期控制者。**
