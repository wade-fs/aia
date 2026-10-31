# AA-04：Go Agent Runtime — Aider PoC

**版本：** 0.1  
**開發語言：** Go  
**專案階段：** P0 — Proof of Concept  
**核心整合對象：** Aider  
**專案性質：** Personal Agent Runtime 的最小可執行驗證

---

## 1. 專案目標

建立一個使用 Go 開發的最小 Agent Runtime，透過統一的 Worker Adapter 介面呼叫 Aider，驗證以下能力：

1. 建立並保存 Task。
2. 啟動外部 AI Worker。
3. 傳送工作需求。
4. 取得 Worker 執行狀態及結果。
5. 檢查 Worker 對專案所做的修改。
6. 執行測試及驗收。
7. 保存完整執行紀錄。
8. Runtime 中斷後能重新啟動並恢復工作狀態。
9. 避免因不確定的執行結果而重複執行工作。

本階段不追求完整 Agent 智能，而是驗證 Runtime 是否能可靠地管理一個真正會修改檔案的外部 AI Worker。

## 2. 系統範圍

### 2.1 本階段必須實作

- Runtime CLI
- Task Manager
- State Manager
- Worker Adapter
- Aider Adapter
- Workspace Manager
- Git Integration
- Result Validator
- SQLite Persistence
- Recovery Manager
- Audit Log

### 2.2 本階段暫不實作

- 多 Agent 協作
- 長期記憶及語義檢索
- Reflection Engine
- Evolution Engine
- 自動策略改善
- Web UI
- Android App
- LangGraph
- 多使用者及遠端 Runtime
- 自動修改 Runtime 本身

以上功能留待核心執行架構通過驗證後再加入。

## 3. 系統架構

```text
                 User
                  |
                  v
             Runtime CLI
                  |
                  v
          +---------------+
          |  Task Manager |
          +---------------+
                  |
                  v
          +---------------+
          |  State Manager|
          +---------------+
                  |
                  v
          +---------------+
          | Worker Manager|
          +---------------+
                  |
                  v
          +---------------+
          | Worker Adapter|
          +---------------+
                  |
                  v
             Aider Worker
                  |
                  v
              Workspace
                  |
          +-------+-------+
          |               |
          v               v
       Git Diff        Test Runner
          |               |
          +-------+-------+
                  |
                  v
           Result Validator
                  |
                  v
             SQLite DB
```

Runtime 是工作流程的管理者。

Aider 是外部執行者。

Aider 不具有修改 Runtime 狀態、改變 Policy 或擴張自身權限的能力。

## 4. 最小專案目錄

```text
agent-runtime/
├── cmd/
│   └── agent/
│       └── main.go
│
├── internal/
│   ├── runtime/
│   │   └── runtime.go
│   │
│   ├── task/
│   │   ├── manager.go
│   │   └── model.go
│   │
│   ├── worker/
│   │   ├── interface.go
│   │   ├── manager.go
│   │   └── aider/
│   │       └── adapter.go
│   │
│   ├── workspace/
│   │   └── manager.go
│   │
│   ├── state/
│   │   └── manager.go
│   │
│   ├── recovery/
│   │   └── manager.go
│   │
│   ├── validator/
│   │   └── validator.go
│   │
│   ├── git/
│   │   └── git.go
│   │
│   ├── storage/
│   │   └── sqlite.go
│   │
│   └── audit/
│       └── logger.go
│
├── configs/
│   └── config.yaml
│
├── data/
│   └── agent.db
│
├── workspaces/
│
├── testdata/
│   └── sample-project/
│
├── docs/
│   └── poc-test-plan.md
│
├── go.mod
└── README.md
```

原則：採用 Modular Monolith，不拆微服務。

各模組有清楚的責任邊界，但初期可以在同一個 Go Process 執行。

## 5. Task 資料模型

每個 Task 至少包含：

- ID
- Title
- Description
- Status
- Worker Type
- Workspace ID
- Created At
- Started At
- Completed At
- Last Updated At
- Worker Execution ID
- Result
- Error
- Git Base Commit
- Git Final Commit
- Validation Result

Task 狀態：

```text
CREATED
   |
   v
READY
   |
   v
RUNNING
   |
   +------> SUCCEEDED
   |
   +------> FAILED
   |
   +------> CANCELLED
   |
   +------> UNKNOWN
                 |
                 v
             RECONCILING
                 |
        +--------+--------+
        |        |        |
        v        v        v
    SUCCEEDED  FAILED  NEEDS_REVIEW
```

UNKNOWN 表示 Runtime 無法確定 Worker 是否已完成工作。

UNKNOWN 不得直接重新執行。

必須先進入 RECONCILING，檢查 Worker、Workspace、Git 狀態及執行紀錄。

## 6. Worker Adapter

Runtime 不直接依賴 Aider 的 CLI 細節。

統一介面：

```go
type Worker interface {
    Initialize(ctx context.Context) error

    Execute(
        ctx context.Context,
        req WorkerRequest,
    ) (*WorkerResult, error)

    GetStatus(
        ctx context.Context,
        executionID string,
    ) (*WorkerStatus, error)

    Cancel(
        ctx context.Context,
        executionID string,
    ) error

    HealthCheck(
        ctx context.Context,
    ) error
}
```

WorkerRequest：

- Task ID
- Execution ID
- Workspace Path
- Task Description
- Model Configuration
- Timeout
- Environment Configuration

WorkerResult：

- Execution ID
- Exit Code
- Standard Output
- Standard Error
- Start Time
- End Time
- Result Summary

Aider Adapter 負責將統一請求轉換成 Aider 所需的執行參數。

未來可以加入其他 Worker，而不必修改 Task Manager。

## 7. Workspace 管理

每個 Task 使用獨立 Workspace。

基本要求：

1. 不直接修改 Runtime 的原始程式碼目錄。
2. 使用 Git Worktree 或獨立 Git Clone 隔離工作。
3. 執行前記錄 Base Commit。
4. Worker 只能修改指定 Workspace。
5. Runtime 負責檢查 Git Diff。
6. 通過驗收後才決定是否接受成果。
7. 不自動將 Worker 的修改合併到主分支。

本階段先使用單一測試 Repository。

## 8. Persistence

使用 SQLite 保存 Runtime 狀態。

至少建立以下資料表：

- tasks
- worker_executions
- task_events
- validation_results
- audit_logs

重要原則：

**先保存執行意圖，再啟動 Worker。**

例如：

1. 建立 Worker Execution。
2. 將 Execution 狀態保存為 STARTING。
3. 啟動 Aider。
4. 取得 Process ID。
5. 保存 PID 及 RUNNING 狀態。

如果 Runtime 在任何一步驟中斷，重新啟動時都必須檢查資料庫與實際執行環境。

不能只依賴記憶體中的狀態。

## 9. Recovery Manager

Runtime 啟動時執行 Recovery。

檢查：

1. 是否存在未完成 Task。
2. 是否存在 RUNNING 或 STARTING Execution。
3. Worker Process 是否仍存在。
4. Workspace 是否存在。
5. Git 是否有未提交修改。
6. 是否已產生結果。
7. 是否需要重新執行測試。

恢復原則：

- Worker 仍在執行：重新建立監控，不重複啟動。
- Worker 已結束且有可驗收成果：進行結果驗收。
- Worker 已結束但結果不明：標記 UNKNOWN，進行調查。
- Workspace 狀態不完整：標記 NEEDS_REVIEW。
- 無法確認安全性：停止自動恢復，等待人工處理。

本階段不允許自動刪除不明狀態的 Workspace。

## 10. Result Validation

Runtime 不以 Aider 回報成功作為最終成功條件。

必須獨立驗收：

1. Worker 是否正常結束。
2. Git Diff 是否存在。
3. 修改是否位於允許的 Workspace。
4. 是否有非預期檔案變動。
5. 專案是否能通過指定測試。
6. 是否產生可追溯的結果紀錄。

Task 的 SUCCEEDED 必須由 Runtime 根據驗收結果決定。

## 11. CLI

初期只需要以下命令：

```bash
agent task create
agent task list
agent task show <task-id>
agent task run <task-id>
agent task cancel <task-id>
agent recovery
agent worker health
```

不需要 Web UI。

## 12. 第一個測試任務

建立一個極小的 Go 測試 Repository。

內容：

```text
sample-project/
├── go.mod
├── main.go
└── main_test.go
```

交給 Aider 的任務：

「請閱讀現有 Go 專案，新增一個計算兩個整數相加的函式 Add，並新增對應單元測試。不要修改其他不相關檔案。」

Runtime 驗證：

- Aider 是否成功啟動。
- 是否新增正確函式。
- 是否新增測試。
- Git Diff 是否符合預期。
- `go test ./...` 是否通過。
- 是否完整保存執行紀錄。

## 13. 測試情境

| 編號 | 測試情境 | 預期結果 |
|---|---|---|
| T01 | 正常建立 Task | 成功保存 |
| T02 | 正常啟動 Aider | Worker RUNNING |
| T03 | Aider 完成修改 | 取得結果 |
| T04 | 測試通過 | Task SUCCEEDED |
| T05 | Aider 執行失敗 | Task FAILED |
| T06 | Runtime 執行中關閉 | 狀態可恢復 |
| T07 | Worker 尚在執行時 Runtime 重啟 | 不重複啟動 |
| T08 | Worker 已修改檔案但未回報 | 進入 RECONCILING |
| T09 | Aider 不存在 | 明確回報啟動失敗 |
| T10 | Workspace 不存在 | 停止並回報 |
| T11 | Task 重複執行 | 防止重複執行 |
| T12 | Worker 修改非預期檔案 | 驗收失敗或人工審查 |

## 14. PoC 成功標準

以下條件全部通過，才算第一階段完成：

- 能建立並保存 Task。
- 能透過 Adapter 啟動 Aider。
- 能完成實際程式碼修改。
- 能取得 Git Diff。
- 能獨立執行測試。
- 能保存 Worker 執行紀錄。
- Runtime 重啟後能恢復狀態。
- 不會因 UNKNOWN 而盲目重複執行。
- Runtime 不依賴 Aider 的內部實作。

## 15. 後續擴充

PoC 成功後，再依序考慮：

1. 增加第二種 Worker，驗證 Adapter 的可替換性。
2. 建立真正的 Coding Domain Agent。
3. 加入 Workflow Engine，評估 LangGraph。
4. 加入 Memory Manager。
5. 加入多 Project 管理。
6. 加入 Reflection 與 Evolution。

---

**本專案最重要的驗證目標：**

我們不是要證明 Aider 能寫程式，而是要證明自己建立的 Runtime 能夠管理一個具有實際執行能力、可能中斷、可能失敗、可能產生副作用的外部 AI Worker。
