# AIA-02：Personal Agent 系統架構與模組設計規格

**文件版本：** 1.0  
**適用版本：** AIA Personal Agent v1  
**前置文件：** AIA-01 產品需求規格書

---

# 1. 架構原則

AIA 採用 Go Modular Monolith。

第一階段不使用微服務，不使用分散式訊息佇列，不將每個 Agent 建立成獨立服務。

所有核心管理能力由 AIA Runtime 負責。

agy 以外部程序形式提供 Agent 執行能力。

## 1.1 架構分層

```text
┌─────────────────────────────────────────────┐
│                  AIA CLI                    │
│          chat / task / agent / memory        │
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│                Application                  │
│                                             │
│  Chat Service       Task Service             │
│  Agent Service      Project Service          │
│  Memory Service     Execution Service        │
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│                  Domain                     │
│                                             │
│  Agent / Task / Project / Memory             │
│  Plan / Policy / Execution / Artifact       │
│  State Machine / Validation                 │
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│                Infrastructure               │
│                                             │
│  SQLite        Antigravity Adapter           │
│  Process       Workspace                     │
│  Event Store   Filesystem                    │
│  Logger        Configuration                 │
└─────────────────────────────────────────────┘
```

Domain 層不得直接依賴 SQLite、agy 或 CLI。

Infrastructure 層負責實作 Domain 所需的 Repository 及 Provider Interface。

---

# 2. 系統模組

## 2.1 App

負責：

- 初始化 Runtime。
- 載入設定。
- 初始化 Database。
- 初始化 Provider。
- 初始化各 Service。
- 啟動 CLI。
- 處理 Shutdown。

App 是組裝層，不負責業務規則。

## 2.2 Agent Manager

負責：

- Agent Definition。
- Agent Version。
- Agent Enabled / Disabled。
- Domain Mapping。
- Agent Capability。
- Agent Prompt。
- Provider Mapping。
- Agent Selection。

Agent Manager 不負責啟動 agy Process。

## 2.3 Personal Agent Orchestrator

負責：

- 使用者輸入理解。
- Context Retrieval。
- Task Classification。
- Task Planning。
- Domain Agent Routing。
- Subtask Dependency。
- Result Aggregation。
- Final Response。

Personal Agent 是 AIA 的主要決策流程。

它使用 Provider 執行推理，但 Task 建立及狀態更新必須透過 AIA Service。

## 2.4 Task Manager

負責：

- Task CRUD。
- Subtask。
- Dependency。
- Task State Machine。
- Priority。
- Assignment。
- Acceptance Criteria。
- Task Query。

Task Manager 不直接呼叫 agy。

## 2.5 Project Manager

負責：

- Project CRUD。
- Project Workspace。
- Project Policy。
- Project Agent。
- Project Memory Scope。
- Project Lifecycle。

## 2.6 Execution Manager

負責：

- Execution 建立。
- Profile 選擇。
- Workspace 配置。
- Provider 呼叫。
- Process Lifecycle。
- Event 收集。
- Execution State。
- Result Collection。

## 2.7 Provider Adapter

負責：

- Antigravity CLI 呼叫。
- CLI Version Check。
- CLI Argument Construction。
- stdin/stdout/stderr。
- NDJSON Parser。
- Conversation ID。
- Process Termination。
- Result Normalization。

不得將 agy 原生事件結構直接傳遞給 Task Manager。

## 2.8 Policy Manager

負責：

- Task Authorization。
- Agent Authorization。
- Project Access。
- Workspace Access。
- Risk Classification。
- Approval Requirement。
- Execution Profile Mapping。

Policy Manager 不重新實作 agy 底層工具權限。

## 2.9 Profile Manager

負責：

- Profile Definition。
- Profile Version。
- Session Settings Generation。
- CLI Parameter Mapping。
- Profile Validation。
- Profile Audit。

Profile Manager 不修改使用者的永久 agy 設定。

## 2.10 Memory Manager

負責：

- Memory CRUD。
- Memory Candidate。
- Memory Classification。
- Memory Retrieval。
- Memory Scope。
- Memory Version。
- Memory Conflict。
- Memory Expiration。

## 2.11 Workspace Manager

負責：

- Project Workspace。
- Execution Workspace。
- Workspace Isolation。
- Workspace Cleanup。
- Artifact Collection。
- Path Validation。

## 2.12 Result Validator

負責：

- Acceptance Criteria。
- Artifact Verification。
- File Existence。
- File Format。
- Test Result。
- Execution Evidence。
- Result Classification。

## 2.13 Recovery Manager

負責：

- Interrupted Execution。
- Process Crash。
- Task Resume。
- Conversation Resume。
- Retry Decision。
- Idempotency Check。
- Recovery Audit。

## 2.14 Audit Manager

負責保存：

- Task State Change。
- Execution Event。
- Policy Decision。
- Approval Decision。
- Memory Change。
- Workspace Change。
- Recovery Action。

---

# 3. Agent 執行與協作模型

## 3.1 AIA Agent 與 agy Worker

AIA Agent 是邏輯角色。

agy Process 是實際執行程序。

一個 AIA Agent 可以建立多個 Task。

一個 Task 可以產生多個 Subtask。

一個 Subtask 可以有多次 Execution。

```text
Personal Agent
    │
    ├── Task A
    │     ├── Coding Agent
    │     │      └── Execution 1
    │     └── Writing Agent
    │            └── Execution 2
    │
    └── Task B
          └── Research Agent
                 ├── Execution 3
                 └── Execution 4
```

## 3.2 Agent Delegation

Personal Agent 在規劃階段產生 Plan。

Plan 必須包含：

- Task Objective。
- Subtask List。
- Agent Assignment。
- Dependency。
- Input Context。
- Expected Output。
- Acceptance Criteria。
- Risk Level。

AIA 驗證 Plan 後才建立 Task。

Plan 不得直接讓模型執行任意系統命令。

## 3.3 AIA Subtask 與 agy Subagent

AIA Subtask 是 AIA 可追蹤、可持久化、可驗收的任務。

agy Subagent 是 agy 內部的執行協作機制。

兩者必須分開。

第一版：

- 跨 Domain Agent 的工作分派由 AIA 管理。
- Domain Agent 內部的輔助工作可由 agy Subagent 處理。
- agy Subagent 不得取代 AIA 的 Task Persistence。
- AIA 必須保存可取得的 Subagent 事件及關聯資訊。

---

# 4. Personal Agent Orchestration

## 4.1 工作流程

```text
User Input
   │
   ▼
Conversation Context
   │
   ▼
Memory Retrieval
   │
   ▼
Intent Classification
   │
   ▼
Task Planning
   │
   ▼
Policy Evaluation
   │
   ▼
Task Creation
   │
   ▼
Agent Dispatch
   │
   ▼
Execution Monitoring
   │
   ▼
Result Validation
   │
   ▼
Memory Candidate
   │
   ▼
User Report
```

## 4.2 Intent Classification

Personal Agent 必須輸出結構化結果：

```json
{
  "intent": "CREATE_TASK",
  "domain": "coding",
  "complexity": "MEDIUM",
  "risk": "LOW",
  "requires_clarification": false,
  "requires_approval": false,
  "summary": "Implement a configuration parser"
}
```

上述 JSON 是 AIA 的內部資料契約，不是 agy 原生輸出格式。

## 4.3 Planning

Plan 必須符合 AIA 定義的 JSON Schema。

每個 Subtask 至少包含：

```json
{
  "title": "Implement configuration loader",
  "description": "Load and validate application configuration",
  "agent_id": "coding",
  "dependencies": [],
  "expected_outputs": [
    "configuration loader implementation",
    "unit tests"
  ],
  "acceptance_criteria": [
    "Valid configuration loads successfully",
    "Invalid configuration returns a typed error"
  ],
  "risk": "LOW"
}
```

AIA 必須在寫入資料庫前驗證：

- Agent ID 是否存在。
- Agent 是否啟用。
- Dependency 是否存在。
- Dependency 是否形成循環。
- Acceptance Criteria 是否存在。
- Project 是否允許此類工作。
- 所需權限是否可提供。

## 4.4 Planning Failure

若模型輸出不符合 Schema：

1. 不建立 Task。
2. 保存 Planning Error。
3. 可依限制重試一次。
4. 再次失敗則要求使用者介入。

不可將無效 JSON 自動猜測成有效 Task。

---

# 5. Execution Architecture

## 5.1 Execution 建立流程

```text
Task READY
   │
   ▼
Policy Evaluation
   │
   ▼
Select Agent
   │
   ▼
Select Execution Profile
   │
   ▼
Create Workspace
   │
   ▼
Generate Session Settings
   │
   ▼
Build Provider Request
   │
   ▼
Start agy Process
   │
   ▼
Read Stream Events
   │
   ▼
Persist Events
   │
   ▼
Collect Result
   │
   ▼
Validate
```

## 5.2 Execution Isolation

每個 Execution 必須具有獨立識別碼及 Workspace。

至少隔離：

- Execution Settings。
- Workspace Path。
- Process。
- Event Stream。
- Execution Log。
- Result。
- Artifact。

OAuth Credential 及 agy Cache 是否可以共用，必須透過 PoC 確認。

AIA 不得假設不同 Process 已經完成所有安全隔離。

## 5.3 Process Model

第一版每個 Execution 使用獨立 agy Process。

一個 Process 可使用 Stream JSON 進行多輪對話。

同一 Process 的多輪對話屬於同一個 Execution Session，不應讓不同 Task 任意共用。

若 Task 需要多次獨立工作，建立新的 Execution。

## 5.4 Provider Request

Provider Request 必須包含：

- Execution ID。
- Task ID。
- Agent Definition。
- Prompt。
- Working Directory。
- Settings。
- CLI Arguments。
- Timeout。
- Environment Allowlist。
- Conversation ID（如恢復）。

Provider 不得自行取得 Task 或 Memory 的完整資料庫內容。

所有必要 Context 由 Application 層組裝後傳入。

---

# 6. Provider Adapter Contract

## 6.1 Provider Interface

```go
type Provider interface {
    Name() string
    Version(ctx context.Context) (string, error)
    HealthCheck(ctx context.Context) error
    Start(ctx context.Context, req StartRequest) (Session, error)
}

type Session interface {
    Send(ctx context.Context, msg InputMessage) error
    Events() <-chan ProviderEvent
    CloseInput() error
    Stop(ctx context.Context) error
    Wait(ctx context.Context) (ProcessExit, error)
}
```

## 6.2 Antigravity Adapter

Antigravity Adapter 必須實作：

- `agy --version`
- `agy --help`（診斷用）
- Headless Mode。
- Stream JSON。
- Session Settings。
- Agent Selection。
- Model Selection。
- Effort Selection。
- Workspace。
- Conversation Resume。

所有命令及參數都必須由 Adapter 組裝。

不得透過字串串接使用者輸入建立 Shell Command。

必須使用 `exec.CommandContext` 並分開傳遞參數。

## 6.3 Stream Protocol

輸入格式：

```json
{
  "event": "user",
  "message": {
    "content": "Execute the assigned task."
  }
}
```

輸出事件：

```json
{"event":"init","conversation_id":"...","init":{}}
{"event":"step_update","step_update":{}}
{"event":"result","result":{}}
```

Adapter 必須支援：

- `init`
- `step_update`
- `result`
- 未知事件保留原始資料。
- JSON Parse Error。
- Stream 中斷。
- Process Exit。
- stderr Diagnostics。

官方 Headless Mode 文件定義 `init`、`step_update`、`result` 的輸出流程，以及 `tool_info`、`subagent_info` 等欄位。AIA 應將其轉成自己的 Event Model，而不是依賴原始欄位直接運作。

## 6.4 Process I/O

stdout 與 stderr 必須同時讀取。

不得先讀取 stdout 再讀取 stderr，以免 Pipe Buffer 滿載造成死鎖。

NDJSON Parser 必須處理：

- 超長行。
- 不完整行。
- 非法 JSON。
- 程序突然結束。
- stdout 與 stderr 的不同步。
- 多個 result event。
- 多輪輸入。

每個輸入 Turn 必須等待對應的 result event 後，才能送出下一個 Turn。

## 6.5 Result Mapping

agy 原始狀態轉換為 AIA 狀態：

| agy Status | AIA Execution Status |
|---|---|
| SUCCESS | RESULT_RECEIVED |
| ERROR | FAILED |
| CANCELED | CANCELLED |
| INTERRUPTED | INTERRUPTED |
| INVALID | FAILED |
| WAITING | WAITING |
| RUNNING | RUNNING |

`SUCCESS` 只代表 agy 本次執行產生成功結果，不代表 AIA Task 已驗收完成。

---

# 7. Permission 與 Profile Architecture

## 7.1 Policy 分層

```text
AIA Task Policy
      │
      ▼
AIA Agent Policy
      │
      ▼
Project Policy
      │
      ▼
Execution Profile
      │
      ▼
agy Permission Configuration
      │
      ▼
agy Sandbox
```

AIA 只管理高層授權及 Profile Mapping。

agy 管理實際工具權限及 Sandbox。

## 7.2 Profile Model

AIA Profile 不等於 agy settings.json。

AIA Profile 是自己的穩定資料模型，必須經由 Adapter 轉換成 agy 設定。

建議資料：

```json
{
  "id": "coding-restricted",
  "version": 1,
  "description": "Coding task with restricted workspace access",
  "sandbox_required": true,
  "mode": "plan",
  "model": null,
  "effort": "high",
  "workspace_scope": "execution",
  "permission_rules": {
    "allow": [],
    "ask": [],
    "deny": []
  }
}
```

這是 AIA Profile Schema，不是 agy settings.json Schema。

## 7.3 Profile Mapping

Adapter 必須依 agy 版本支援狀況轉換設定。

不得假設所有 AIA Profile 欄位都有對應的 agy 設定。

遇到無法映射的必要政策時，必須拒絕啟動 Execution。

不得靜默忽略安全限制。

## 7.4 Session Settings

AIA 應使用 agy 1.4.12 的 `--settings`，為每次 Session 提供專用設定。

必須明確指定必要設定，避免依賴使用者全域設定。

實際 JSON Schema 及設定優先順序必須由版本測試確認。

## 7.5 Approval

Headless Mode 的 Ask 規則可能被 soft-deny。

AIA 不得等待一個不存在的互動式核准視窗。

遇到 soft-deny：

1. 記錄 Tool Denied Event。
2. 停止後續不必要的工作。
3. 標記 Execution 為 BLOCKED。
4. Task 設為 WAITING_APPROVAL。
5. 顯示工具、目標、參數及風險。
6. OWNER 核准後建立新的 Execution。

不得將 `--dangerously-skip-permissions` 作為核准後的通用解法。

---

# 8. Task State Machine

## 8.1 Task 狀態

```text
CREATED
   │
   ▼
READY
   │
   ▼
RUNNING
   │
   ├── WAITING_FOR_AUTH
   ├── WAITING_APPROVAL
   ├── WAITING_FOR_INPUT
   ├── SUSPENDED
   ├── FAILED
   ├── CANCELLED
   ▼
VALIDATING
   │
   ├── RETRYING
   ├── FAILED
   └── COMPLETED
```

## 8.2 狀態轉移規則

- `CREATED → READY`：Plan 及必要欄位驗證成功。
- `READY → RUNNING`：至少一個 Execution 已啟動。
- `RUNNING → WAITING_FOR_AUTH`：確認需要重新登入。
- `RUNNING → WAITING_APPROVAL`：需要 OWNER 核准。
- `RUNNING → WAITING_FOR_INPUT`：缺少必要資訊。
- `RUNNING → VALIDATING`：收到最終結果。
- `VALIDATING → COMPLETED`：驗收通過。
- `VALIDATING → RETRYING`：允許重試。
- `VALIDATING → FAILED`：不可恢復或超過重試上限。
- 任一未完成狀態 → `CANCELLED`：使用者取消。
- `SUSPENDED → READY`：恢復後重新排程。

狀態轉移必須透過單一 Domain Service 執行。

## 8.3 Execution State

Execution 狀態：

- `CREATED`
- `PREPARING`
- `STARTING`
- `RUNNING`
- `BLOCKED`
- `COLLECTING_RESULT`
- `VALIDATING`
- `SUCCEEDED`
- `FAILED`
- `INTERRUPTED`
- `CANCELLED`
- `TIMED_OUT`

Task 與 Execution 狀態不得混用。

---

# 9. Memory Architecture

## 9.1 Memory Pipeline

```text
Execution Result
      │
      ▼
Memory Candidate Extraction
      │
      ▼
Classification
      │
      ▼
Duplicate Check
      │
      ▼
Conflict Check
      │
      ▼
Policy Check
      │
      ▼
Memory Approval / Auto Approval
      │
      ▼
Persistent Memory
```

## 9.2 Memory Retrieval

Retrieval 輸入：

```go
type MemoryQuery struct {
    Query     string
    Scope     string
    ProjectID string
    AgentID   string
    Types     []MemoryType
    Limit     int
}
```

Retrieval 結果必須包含來源及可信度。

## 9.3 Memory Context Budget

每次 Execution 必須設定 Memory Context 上限。

不得將整個 Memory Database 傳入 agy。

Memory Manager 只回傳經過排序及限制的資料。

## 9.4 Memory Conflict

當新 Memory 與既有 Memory 衝突：

- 不直接覆寫。
- 建立 Conflict Record。
- 保留來源及時間。
- 依政策決定是否詢問 OWNER。
- 可用於未來的 Memory Resolution。

---

# 10. Workspace Architecture

## 10.1 目錄結構

```text
AIA/
├── configs/
├── data/
│   ├── aia.db
│   ├── logs/
│   └── cache/
├── workspaces/
│   ├── projects/
│   └── executions/
└── backups/
```

## 10.2 Workspace 規則

每個 Execution 建立獨立目錄：

```text
workspaces/executions/<execution-id>/
```

Execution Workspace 不得直接共用另一個 Execution 的可寫入目錄。

需要 Project 原始資料時，依政策使用唯讀掛載、複製或其他已驗證方式提供。

## 10.3 Cleanup

Workspace 不得在 Execution 結束後立即刪除。

清理必須考慮：

- 是否有未驗收成果。
- 是否需要 Recovery。
- 是否需要保留 Audit。
- 是否有使用者指定保留期限。

---

# 11. Persistence Architecture

第一階段使用 SQLite。

Repository Interface 與 SQL 實作分離。

資料庫採 Migration 管理。

重要狀態轉移與 Event 寫入應使用交易。

對於 Task 狀態及 Event 必須避免一方成功、另一方失敗造成資料不一致。

---

# 12. Error Handling Architecture

錯誤分為：

- `ERR_PROVIDER_UNAVAILABLE`
- `ERR_PROVIDER_VERSION`
- `ERR_AUTH_REQUIRED`
- `ERR_PERMISSION_DENIED`
- `ERR_PROFILE_INVALID`
- `ERR_SANDBOX_UNAVAILABLE`
- `ERR_PROCESS_START`
- `ERR_PROCESS_EXIT`
- `ERR_PROCESS_TIMEOUT`
- `ERR_STREAM_PARSE`
- `ERR_STREAM_CLOSED`
- `ERR_TASK_INVALID`
- `ERR_PLAN_INVALID`
- `ERR_DEPENDENCY_CYCLE`
- `ERR_WORKSPACE_ACCESS`
- `ERR_RESULT_INVALID`
- `ERR_MEMORY_CONFLICT`
- `ERR_STORAGE`

每個錯誤必須包含：

- Code。
- Message。
- Component。
- Task ID（如有）。
- Execution ID（如有）。
- Retryable。
- Cause。

錯誤不得只以字串判斷。

---

# 13. Concurrency Architecture

第一版使用 Go Goroutine 管理 Execution。

必須有全域並行限制。

初始設定：

```yaml
runtime:
  max_concurrent_executions: 2
  max_concurrent_per_agent: 1
```

以上為初始預設值，可透過設定調整。

同一 Task 的互相依賴 Subtask 不得同時執行。

每個 Execution 使用獨立 Context、Process、Workspace 及 Event Stream。

SQLite 寫入必須妥善處理並行交易。

---

# 14. Recovery Architecture

Runtime 啟動時執行 Recovery Scan。

檢查：

- RUNNING Task。
- RUNNING Execution。
- 未正常結束的 Process。
- 未完成的 Event。
- 尚未驗收的 Result。
- 未完成的 Approval。
- 未完成的 Memory Candidate。

Recovery 不得自動重送所有 RUNNING Task。

必須先確認：

1. 原 Process 是否存在。
2. 原 Conversation 是否可恢復。
3. 原 Workspace 是否完整。
4. 最後一個結果事件是否已持久化。
5. 是否有不可重複的外部副作用。

無法確認時，標記 `INTERRUPTED` 並等待處理。

---

# 15. Audit Architecture

重要操作必須產生 Audit Event。Audit 欄位經資料庫版本遷移擴充，包含 Request 追蹤、來源分類與遮蔽機密之結構化 Metadata：

```go
type AuditEvent struct {
    ID           string
    Sequence     uint64
    ActorType    string
    ActorID      string
    Action       string
    ResourceType string
    ResourceID   string
    Outcome      string
    Details      json.RawMessage
    RequestID    string
    Source       string // CLI, WEB, RUNTIME, AGENT, RECOVERY
    Metadata     json.RawMessage
    CreatedAt    time.Time
}
```

Audit 不得保存明文 Token、密碼、Cookie 或 Credential。

## 15.1 Web Host 元件隔離與登入流程

1. **元件與路由隔離**：
   - AIA 採用單一 Web Host，Domain Agent 透過 `AgentWebModule` 註冊自訂 UI。
   - Web Host 為各 Agent 建立獨立專屬的 `*http.ServeMux`，並掛載於 `/api/v1/agents/{agent_id}/`。
   - Agent 不接觸全域 mux，無法攔截或覆寫全域系統保留路徑，統一由 Host 套用認證、授權與 Audit 中介層。
2. **一次性 Token 與 Session**：
   - 啟動時產生高熵一次性登入 Token，於終端機僅輸出一次。
   - 經 `/api/v1/auth/login` 驗證後立即使 Token 永久失效（單次兌換）。
   - 建立具備閒置逾時與絕對逾時的 Server-Side Session，透過 `HttpOnly`、`SameSite=Strict` Cookie 維護。

## 15.2 投資決策責任邊界 (Investment Responsibility Boundary)

- **agy**：負責市場研究、財報與公開資訊研判、決策方向與信心度評估。
- **Go Runtime**：負責股票標的驗證、計算建議股數、計算手續費與證交稅、檢查可用現金與資金上限及風控限制、產生最終正式 Trade Proposal。超出限制時自動駁回或向下調整，嚴禁 LLM 未經 Go 驗證直接產生有效交易提案。

---

# 16. Architecture Acceptance

系統架構必須符合：

1. Task Manager 不依賴 agy。
2. Memory Manager 不依賴 agy。
3. Agent Manager 不依賴 agy 原生資料模型。
4. Provider Adapter 是唯一直接呼叫 agy 的模組。
5. Task State 由 AIA 管理。
6. Execution State 由 AIA 管理。
7. Task 與 Execution 分離。
8. AIA Agent 與 agy Agent 分離。
9. AIA Subtask 與 agy Subagent 分離。
10. AIA Profile 與 agy settings.json 分離。
11. Memory 不依賴 Conversation 作為唯一儲存。
12. 所有重要狀態轉移可稽核。
13. 不可驗證的權限設定不得靜默忽略。
14. Runtime 重啟後可以恢復或明確標示中斷任務。
15. Provider 可以在不修改核心 Domain Model 的情況下替換。
