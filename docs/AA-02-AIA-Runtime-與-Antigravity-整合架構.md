# AA-02：AIA Runtime 與 Antigravity 整合架構

**文件代號：** AA-02  
**文件版本：** 2.0  
**文件狀態：** Architecture Specification  
**適用專案：** AIA — Autonomous Intelligence Architecture  
**開發語言：** Go  
**AI 執行引擎：** Antigravity CLI（agy）  
**目前驗證基準：** Antigravity CLI 1.4.12  
**文件目的：** 定義 AIA Runtime 與 Antigravity CLI 之間的整合方式、責任邊界、執行模型及安全控制。

---

# 1. 文件目的

本文件定義 AIA Runtime 如何整合 Antigravity CLI（以下簡稱 agy），使 AIA 能夠透過外部 Agent 執行引擎完成任務，同時保有對整體任務生命週期、Agent 組織、記憶、授權及執行結果的管理能力。

本文件取代先前 AA-02 初稿中的部分設計假設。

經過 Antigravity 官方文件研究及實際 CLI 1.4.12 參數核對，確認 agy 已具備相當完整的 Agent Harness 能力，包括：

- Agent 執行及管理。
- 工具呼叫。
- Subagent。
- 多輪對話與 Conversation 恢復。
- MCP 管理。
- Terminal Sandbox。
- 細粒度工具權限管理。
- Headless Mode。
- JSON 與 NDJSON 結構化輸出。
- Session 層級設定。

因此，AIA 不應重新實作 agy 已經提供的底層 Agent 執行及工具權限機制。

AIA 的核心工作是管理更高層的 Agent 系統。

# 2. 系統定位

## 2.1 AIA 與 agy 的關係

AIA 是 Agent 管理與協調平台。

agy 是 AIA 所使用的 Agent 執行引擎之一。

兩者不是上下級的智慧模型關係，而是管理系統與執行引擎的分工。

```text
┌────────────────────────────────────────────┐
│                    AIA                     │
│                                            │
│  Personal Agent                            │
│       │                                    │
│       ├── Task Management                  │
│       ├── Agent Organization               │
│       ├── Project Management               │
│       ├── Memory Management                 │
│       ├── High-level Policy                │
│       ├── Execution Management             │
│       ├── Result Validation                │
│       └── Recovery / Audit                 │
│                    │                       │
│              Provider Adapter              │
└────────────────────┬───────────────────────┘
                     │
                     │ Process / Stream JSON
                     ▼
┌────────────────────────────────────────────┐
│              Antigravity CLI               │
│                                            │
│  Agent Harness                             │
│       │                                    │
│       ├── Agent                             │
│       ├── Subagent                          │
│       ├── Planning                          │
│       ├── Tool Calling                      │
│       ├── MCP                               │
│       ├── Permission Engine                 │
│       └── Terminal Sandbox                  │
│                                            │
│  Model / Authentication / Tool Execution   │
└────────────────────────────────────────────┘
```

## 2.2 設計原則

1. AIA 不直接依賴特定 LLM。
2. AIA 不直接管理 agy 的 OAuth Token。
3. AIA 不重新實作 agy 的 Tool Permission Engine。
4. AIA 不重新實作 agy 的 OS-level Sandbox。
5. AIA 不把 agy 的 Agent、Subagent 或內部工作流程視為 AIA 自己的 Task。
6. AIA 必須保有跨 Agent、跨 Project、跨 Execution 的管理能力。
7. AIA 必須能夠替換 AI Provider 或執行引擎。
8. AIA 必須能夠獨立保存任務狀態及執行紀錄。
9. AIA 不可只依靠 Prompt 控制安全性。
10. AIA 不可只依靠程序退出碼判斷任務是否成功。

# 3. 責任邊界

## 3.1 AIA 負責的能力

| 能力 | 責任 |
|---|---|
| Personal Agent | 系統全域任務管理及決策 |
| Domain Agent | 專業領域 Agent 組織及工作分派 |
| Task Management | 任務建立、狀態、優先順序及生命週期 |
| Project Management | 專案資料及工作空間管理 |
| Execution Management | 建立、啟動、監控及終止 Execution |
| Memory | 跨 Task 的長期記憶 |
| High-level Policy | 任務層級授權及限制 |
| Execution Profile | 選擇及產生 agy Session 設定 |
| Result Validation | 檢查 Agent 工作成果 |
| Recovery | 失敗處理、重試及恢復 |
| Audit | 完整記錄任務與執行歷程 |
| Evolution | 評估及改善 Agent 的工作方式 |

## 3.2 agy 負責的能力

| 能力 | 責任 |
|---|---|
| Agent Execution | Agent 執行 |
| Planning | Agent 內部規劃 |
| Tool Calling | 工具呼叫及執行 |
| Subagent | Agent 內部子任務及協作 |
| MCP | MCP Server 與工具整合 |
| Permission Engine | 檔案、命令、網路及 MCP 權限 |
| Terminal Sandbox | OS 層級執行隔離 |
| Model Interaction | 模型呼叫 |
| OAuth | 使用者登入與授權資料管理 |
| Conversation | Agent 對話及 Session 狀態 |

## 3.3 共同責任

| 能力 | AIA | agy |
|---|---|---|
| Agent 定義 | 管理全域角色與任務責任 | 提供可執行 Agent |
| 工具授權 | 決定高層政策 | 執行底層工具權限 |
| Workspace | 管理專案及工作目錄 | 在指定範圍內操作 |
| Execution | 建立與監控 | 執行 |
| Result | 驗收 | 提供執行結果 |
| Recovery | 決定是否重試或恢復 | 支援 Conversation 恢復 |
| Security | 管理系統整體授權 | 執行工具及 Sandbox 限制 |

AIA 不得因為 agy 已經具有底層權限管理，就放棄高層授權責任。

反之，AIA 也不得在 agy 之上再建立一套功能重複的底層工具權限引擎。

# 4. Antigravity CLI 整合方式

## 4.1 外部程序整合

第一階段採用 Go `os/exec` 啟動 agy 外部程序。

AIA 不直接嵌入 agy，也不依賴 agy 的內部程式碼。

```text
AIA Runtime
    │
    ▼
Antigravity Provider Adapter
    │
    ▼
Process Manager
    │
    ├── stdin
    ├── stdout
    ├── stderr
    └── process lifecycle
    │
    ▼
agy
```

採用外部程序的理由：

- 降低 AIA 與 agy 的耦合。
- 不需要自行處理模型供應商 API。
- 不需要自行實作 OAuth。
- 可以獨立更新 agy。
- 未來可以替換成其他 Agent 執行引擎。
- 可以利用 Go 標準函式庫管理程序及 I/O。

## 4.2 CLI 啟動參數

目前以 agy 1.4.12 為整合基準。

已確認的主要參數：

| 參數 | 用途 |
|---|---|
| `--print` / `-p` | 非互動式執行 |
| `--prompt` | `--print` 別名 |
| `--agent` | 指定 Agent |
| `--model` | 指定模型 |
| `--effort` | 指定推理強度 |
| `--mode` | 指定執行模式 |
| `--sandbox` | 啟用 Terminal Sandbox |
| `--settings` | 指定本次 Session 的設定 |
| `--project` | 指定 Project |
| `--add-dir` | 增加工作目錄 |
| `--conversation` | 恢復指定 Conversation |
| `--continue` | 繼續最近的 Conversation |
| `--input-format` | 指定輸入格式 |
| `--output-format` | 指定輸出格式 |
| `--json-schema` | 要求結構化輸出 |
| `--print-timeout` | 設定執行時間限制 |

`--mode` 已確認支援：

- `accept-edits`
- `auto`
- `plan`

`--effort` 已確認支援：

- `low`
- `medium`
- `high`
- `xhigh`
- `max`

AIA 應將上述參數封裝於 Provider Adapter，不應讓其他 Runtime 模組直接組裝 agy 命令。

## 4.3 CLI 啟動策略

預計支援兩種執行模式。

### 模式 A：單次 Prompt

使用：

```bash
agy --print "任務內容" --output-format json
```

適用：

- 簡單任務。
- 一次性查詢。
- 不需要持續監控中間事件的工作。
- PoC 初期驗證。

### 模式 B：Stream JSON

使用：

```bash
agy --print \
    --input-format stream-json \
    --output-format stream-json
```

由 AIA 透過 stdin 傳送 NDJSON，每一行代表一個輸入訊息。

agy 透過 stdout 回傳 NDJSON 事件。

適用：

- 多輪互動。
- 長時間任務。
- 需要監控 Agent 執行過程。
- 需要取得工具呼叫資訊。
- 需要取得 Subagent 資訊。
- 需要建立 Execution Event Log。

**第一階段建議以 Stream JSON 作為主要整合模式。**

但實際上是否需要每個 Task 都使用長駐程序，應由 PoC 比較效能、穩定性及程序管理成本後決定。

# 5. Execution Model

## 5.1 Task 與 Execution 分離

AIA 必須區分 Task 與 Execution。

Task 代表使用者要求完成的工作。

Execution 代表 Agent 實際執行工作的某一次執行過程。

```text
Task
 │
 ├── Execution 001
 │      ├── agy Process
 │      ├── Conversation
 │      ├── Events
 │      └── Result
 │
 ├── Execution 002
 │      ├── agy Process
 │      ├── Conversation
 │      ├── Events
 │      └── Result
 │
 └── Final Result
```

一個 Task 可以有多次 Execution。

例如：

1. 第一次執行失敗。
2. AIA 分析失敗原因。
3. 第二次執行修正問題。
4. AIA 驗證結果。
5. Task 完成。

## 5.2 Task Lifecycle

建議 Task 狀態：

| 狀態 | 說明 |
|---|---|
| CREATED | 任務建立 |
| READY | 等待執行 |
| RUNNING | 正在執行 |
| WAITING_FOR_AUTH | 等待使用者重新登入或授權 |
| WAITING_APPROVAL | 等待高層政策核准 |
| WAITING_FOR_INPUT | 等待使用者提供資訊 |
| VALIDATING | 驗證成果 |
| RETRYING | 準備重新執行 |
| COMPLETED | 成功完成 |
| FAILED | 執行失敗 |
| CANCELLED | 使用者取消 |
| SUSPENDED | 暫停 |

狀態轉移必須由 AIA Runtime 管理，不得單純依賴 agy 的文字回覆決定。

## 5.3 Execution Lifecycle

```text
CREATED
   │
   ▼
PREPARING
   │
   ▼
STARTING
   │
   ▼
RUNNING
   │
   ├── AUTH_REQUIRED
   │
   ├── PROCESS_FAILED
   │
   ├── CANCEL_REQUESTED
   │
   ▼
COLLECTING_RESULT
   │
   ▼
VALIDATING
   │
   ├── ACCEPTED
   │
   ├── REJECTED
   │
   └── RETRY_REQUIRED
```

Execution 的狀態必須與 Task 狀態分開保存。

# 6. Execution Profile

## 6.1 設計目的

agy 已提供 Session 層級設定：

```bash
--settings
```

其用途是指定本次 Session 使用的設定，而不是修改使用者永久設定。

AIA 應利用此能力，為不同類型的任務產生適合的執行設定。

AIA 不應依賴使用者目前的 `settings.json` 作為系統預設設定，也不應任意修改使用者的全域設定。

## 6.2 Profile 管理

建議目錄：

```text
configs/
└── antigravity/
    ├── profiles/
    │   ├── coding.json
    │   ├── writing.json
    │   ├── research.json
    │   └── restricted.json
    │
    └── policies/
        └── policy-mapping.json
```

Profile 是 AIA 的執行政策描述，不代表上述 JSON 檔案已經符合 agy 的正式設定 Schema。

實際寫入 agy `settings.json` 的欄位，必須依照官方文件確認的設定名稱、型別及預設值產生。

## 6.3 Profile 選擇

| 任務類型 | 建議 Profile | 執行方向 |
|---|---|---|
| Coding | coding | 限定 Workspace，必要時允許指定命令 |
| Writing | writing | 文件操作為主，限制不必要的系統命令 |
| Research | research | 依需求允許網路及 MCP |
| 高風險任務 | restricted | 嚴格限制工具與操作 |

以上是初步分類，不代表所有同類 Task 都應使用相同權限。

Profile 應可依 Task 的實際授權需求調整。

## 6.4 每次 Execution 的設定產生

預期流程：

```text
Task
 │
 ▼
AIA Policy Evaluation
 │
 ▼
Select Execution Profile
 │
 ▼
Generate Session Settings
 │
 ▼
Create Execution Workspace
 │
 ▼
Start agy
     --settings <session-settings-file>
     --sandbox
     --mode <execution-mode>
```

暫存設定檔應由 AIA 建立、管理及清理。

必須遵守：

- 不覆寫使用者的全域設定。
- 不讀取或複製 OAuth Token。
- 不將 Credential 寫入 Execution Settings。
- 不允許 Agent 任意修改 AIA 的政策設定。
- 每個 Execution 記錄實際使用的 Profile 版本。
- 設定檔及執行政策必須可追溯。

## 6.5 待驗證事項

以下尚未完成 PoC：

1. `--settings` 的 JSON 物件與檔案路徑兩種用法。
2. Session Settings 與 Global Settings 的合併方式。
3. 未指定欄位的預設行為。
4. CLI 旗標與 Settings 之間的優先順序。
5. 不同 Session 同時執行時是否完全獨立。
6. Profile 是否可以完整控制各項 Permission。
7. Session 結束後設定檔的清理方式。

在上述項目確認前，不得假設 `--settings` 等同於完整、獨立的安全邊界。

# 7. Permission 與 Security

## 7.1 雙層授權模型

AIA 採用高層授權與底層工具權限分離的模型。

```text
          AIA High-level Policy
                   │
                   ▼
          Task Authorization
                   │
                   ▼
        Execution Profile Selection
                   │
                   ▼
          agy Permission Engine
                   │
                   ▼
          OS-level Sandbox
                   │
                   ▼
             Tool Execution
```

### AIA 高層授權

負責判斷：

- 這個 Agent 是否可以承接此類任務？
- 這個 Task 是否可以讀取指定 Project？
- 是否允許修改指定資料？
- 是否允許呼叫特定外部服務？
- 是否涉及需要使用者確認的操作？
- 是否允許使用特定 MCP 或其他外部能力？

### agy 底層權限

負責判斷：

- 是否可以讀取指定檔案？
- 是否可以寫入指定目錄？
- 是否可以執行指定命令？
- 是否可以連接指定網域？
- 是否可以呼叫指定 MCP Tool？

AIA 的授權結果應轉換成 agy 可執行的權限設定。

## 7.2 agy Permission Engine

已確認 agy 提供三種權限規則：

- Allow
- Ask
- Deny

規則優先順序：

```text
Deny > Ask > Allow
```

支援的權限類型包括：

- `read_file`
- `write_file`
- `read_url`
- `execute_url`
- `command`
- `mcp`

例如：

```text
command(git)
write_file(src/)
mcp(server/tool)
```

AIA 應利用這些原生權限機制，不應另外開發一套功能重複的底層 Permission Engine。

## 7.3 Sandbox

agy 提供 OS-level Terminal Sandbox。

已知平台機制：

- Linux：Namespaces。
- macOS：Seatbelt。

Sandbox 可協助隔離檔案系統、程序及網路存取。

AIA 應優先使用 agy 原生 Sandbox。

但是：

- 工作目錄不等於 Sandbox。
- `--add-dir` 不等於授權所有操作。
- Sandbox 與 Permission 必須共同考慮。
- Sandbox 不能取代 AIA 的高層授權。
- 不得把 Sandbox 狀態視為所有資料都已完全隔離的證明。

對於金融、系統管理或其他高風險任務，未來可以再考慮額外的 OS 使用者、Container 或 VM 隔離。

## 7.4 禁止使用的權限捷徑

AIA 不得將下列方式作為一般自動化策略：

```bash
--dangerously-skip-permissions
```

這個參數會自動核准所有工具權限要求，破壞原有的權限控制。

除非未來經過明確的風險評估及使用者授權，否則 AIA 不應使用此參數。

## 7.5 Headless Mode 的權限核准陷阱

Headless Mode 不會提供一般互動式的人工核准提示。

需要人工核准的工具可能會被 soft-deny。

重要的是：

**soft-deny 不一定代表整個程序失敗，也不保證以非零 Exit Code 結束。**

因此，AIA 不得只依賴 Exit Code 判斷執行結果。

必須綜合檢查：

1. Process Exit Code。
2. Stream JSON Result Status。
3. 工具呼叫事件。
4. Tool Execution Result。
5. Permission Notice。
6. stderr Diagnostics。
7. Workspace 實際變更。
8. 最終成果驗證。

若任務因權限不足而未完成，AIA 應依政策轉入等待核准、調整授權或失敗狀態。

不得自動提高權限來規避 soft-deny。

# 8. Process Manager

## 8.1 責任

Process Manager 負責 agy 外部程序的生命週期。

包括：

- 啟動程序。
- 傳送輸入。
- 接收輸出。
- 監控程序狀態。
- 監控執行時間。
- 處理程序終止。
- 處理 stdout / stderr。
- 記錄 Exit Code。
- 處理程序異常。
- 回報 Execution Event。

Process Manager 不負責判斷 Task 是否完成。

## 8.2 Go 實作方向

使用 Go 標準函式庫：

- `os/exec`
- `context`
- `io`
- `bufio`
- `encoding/json`
- `sync`

預期介面：

```go
type ProcessRequest struct {
    ExecutionID string
    Command     string
    Args        []string
    WorkDir     string
    Environment []string
}

type ProcessResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}

type ProcessManager interface {
    Start(ctx context.Context, req ProcessRequest) error
    SendInput(executionID string, data []byte) error
    Stop(executionID string) error
    Wait(executionID string) (ProcessResult, error)
}
```

以上為概念性介面，正式實作時應另外設計事件串流、輸出大小限制、資源管理及程序狀態。

## 8.3 程序隔離

每個 Execution 應有自己的：

- Execution ID。
- Workspace。
- Session Settings。
- Process。
- Conversation ID（取得後）。
- Event Log。
- Result。

不同 Execution 不應共用可互相修改的工作目錄。

是否需要為每個 Execution 建立獨立 HOME 或設定目錄，須經 OAuth、快取及並行測試後決定。

不能因為設定檔不同，就認定所有使用者層級狀態已經隔離。

# 9. Stream JSON Protocol

## 9.1 設計目的

Stream JSON 是 AIA 與 agy 之間主要的即時通訊格式。

AIA 透過 stdin 傳送訊息。

agy 透過 stdout 傳送事件。

stderr 用於接收診斷資訊及其他非主要結果訊息。

## 9.2 事件分類

官方文件已列出下列重要事件：

| Event | 用途 |
|---|---|
| `init` | 初始化 Conversation 及 Session |
| `step_update` | 回報 Agent 執行步驟 |
| `result` | 回報完整結果 |

事件中可能包含：

- `conversation_id`
- `cwd`
- `tools`
- `permission_mode`
- `agent_response`
- `tool_name`
- `tool_info`
- `subagent_info`
- `status`

實際欄位結構及可選欄位應以 1.4.12 的實際輸出為準。

## 9.3 AIA Event Normalization

AIA 不應讓 Runtime 其他模組直接依賴 agy 的原始 NDJSON 格式。

應由 Adapter 轉換成 AIA 統一的 Event Model。

例如：

```go
type ExecutionEvent struct {
    ExecutionID string
    Sequence    uint64
    Timestamp   time.Time
    Type        string
    Source      string
    Payload     json.RawMessage
}
```

建議 Event 類型：

- `execution.started`
- `execution.initialized`
- `agent.step.updated`
- `agent.response.updated`
- `tool.started`
- `tool.completed`
- `tool.denied`
- `subagent.started`
- `subagent.completed`
- `execution.result`
- `execution.failed`
- `execution.cancelled`

以上是 AIA 的標準化事件名稱，不代表 agy 原生事件名稱。

## 9.4 Event Persistence

所有重要事件都應寫入 AIA 的持久化儲存。

目的：

- 執行歷程追蹤。
- 異常診斷。
- Task Recovery。
- Agent 行為分析。
- 成本與資源分析。
- 日後 Evolution。

原始 NDJSON 可以選擇性保存，但不得只保留最終回答而丟失重要執行紀錄。

# 10. Conversation 與 Recovery

## 10.1 Conversation 管理

agy 支援：

```bash
--conversation <id>
```

及：

```bash
--continue
```

AIA 應保存：

- Task ID。
- Execution ID。
- agy Conversation ID。
- Provider 名稱。
- Agent 名稱。
- 建立時間。
- 最後執行時間。
- Conversation 狀態。

Conversation ID 必須與 AIA 的 Task ID 分開管理。

## 10.2 Recovery 原則

AIA 不應將「重新啟動程序」視為「恢復任務」。

恢復前應先確認：

1. 原程序是否仍在執行？
2. 原 Execution 是否已經完成？
3. 最後一次事件是否已經持久化？
4. Conversation 是否可恢復？
5. Workspace 是否仍存在？
6. 是否已經發生部分修改？
7. 重試是否會重複執行外部副作用？

若無法確認任務是否已經執行成功，不應直接重送可能造成重複副作用的操作。

## 10.3 Idempotency

每次 Execution 必須有唯一識別碼。

AIA 必須避免因為程序中斷或網路問題，重複執行同一個不可重複的外部操作。

對於具備外部副作用的任務，應由 AIA 記錄執行意圖、操作結果及確認狀態。

不能假設 agy Conversation 恢復就能保證外部操作不重複。

# 11. Authentication

## 11.1 OAuth 責任邊界

OAuth 由使用者及 agy 負責。

| 工作 | 負責者 |
|---|---|
| Google 帳戶登入 | 使用者 |
| OAuth 授權 | 使用者透過 agy |
| Credential 保存 | agy |
| Token 更新 | agy（依其支援能力） |
| OAuth Token 管理 | agy |
| agy 可用性檢查 | AIA |
| agy 呼叫 | AIA |
| 登入失效處理 | AIA 與使用者 |

AIA 不讀取、不複製、不解析、不保存 agy OAuth Token。

## 11.2 Authentication Failure

若 agy 回報需要登入或授權：

```text
agy Execution
      │
      ▼
Authentication Error
      │
      ▼
AIA detects authentication failure
      │
      ▼
Task = WAITING_FOR_AUTH
      │
      ▼
Notify User
      │
      ▼
User authenticates through agy
      │
      ▼
AIA checks agy availability
      │
      ▼
Resume Task
```

AIA 不得自行取得使用者帳戶密碼或操作 OAuth Token。

## 11.3 登入檢查

AIA 應提供 Provider Health Check。

檢查內容包括：

- agy 執行檔是否存在。
- agy 版本是否符合要求。
- CLI 是否可以正常啟動。
- 登入狀態是否可用。
- 是否可以完成最小化的模型請求。

實際使用何種指令判斷登入狀態，須以官方支援的 CLI 能力及 PoC 確認。

不可把任意錯誤都判定為 OAuth 失效。

# 12. Workspace Management

## 12.1 Workspace 責任

AIA 管理 Workspace 的建立、分配、回收及關聯。

agy 在指定工作範圍內執行 Agent 工作。

建議目錄：

```text
workspaces/
├── projects/
│   ├── project-a/
│   └── project-b/
│
└── executions/
    ├── execution-001/
    ├── execution-002/
    └── execution-003/
```

## 12.2 Workspace 原則

1. Project Workspace 由 AIA 管理。
2. Execution Workspace 依任務需要建立。
3. 不同 Execution 預設使用不同工作目錄。
4. 只有經授權的資料才可以加入工作範圍。
5. 不應任意把整個使用者 Home Directory 加入 Workspace。
6. Workspace 的實際存取限制應由 agy Permission 與 Sandbox 執行。
7. Workspace 清理前應確認沒有未提交成果或需要保留的紀錄。

## 12.3 Git 整合

Coding 類任務可使用 Git 作為成果管理工具。

AIA 可以管理：

- Git Repository。
- Branch。
- Commit。
- Diff。
- 工作成果檢查。
- 版本回復。

但 Git 的實際操作由 agy 在授權範圍內執行。

AIA 不應要求所有類型的任務都必須使用 Git。

# 13. Result Validation

## 13.1 驗收責任

agy 回報 `result`，不代表 AIA 的 Task 必然完成。

AIA 必須獨立判斷結果是否符合任務要求。

```text
agy Result
    │
    ▼
AIA Result Validator
    │
    ├── Execution Status
    ├── Tool Execution Records
    ├── Permission Notices
    ├── Workspace Changes
    ├── Expected Artifacts
    └── Task Acceptance Criteria
    │
    ▼
Accept / Reject / Retry
```

## 13.2 驗收項目

依任務類型決定：

- 是否產生預期檔案。
- 檔案是否完整。
- 是否符合指定格式。
- 是否符合 Task Acceptance Criteria。
- 是否存在未完成工具呼叫。
- 是否發生 Permission Denied。
- 是否有錯誤或部分成果。
- 是否有未經授權的工作目錄變更。

## 13.3 結果狀態

建議分為：

- `SUCCESS`
- `PARTIAL_SUCCESS`
- `BLOCKED`
- `FAILED`
- `INVALID_RESULT`

Task 是否完成由 AIA 決定。

# 14. SQLite Persistence

第一階段採用 SQLite。

AIA 必須保存自己的任務與執行狀態，不依賴 agy Conversation 作為唯一資料來源。

建議核心資料表：

| Table | 用途 |
|---|---|
| projects | 專案 |
| agents | AIA Agent 定義 |
| tasks | 任務 |
| executions | 執行紀錄 |
| execution_events | 執行事件 |
| conversations | agy Conversation 對應 |
| execution_profiles | Profile 版本 |
| policies | 高層授權政策 |
| workspaces | 工作目錄 |
| artifacts | 執行成果 |
| audit_logs | 稽核紀錄 |

資料表的正式欄位、索引、外鍵及 Migration 規則，另於資料模型文件定義。

# 15. Go Runtime 模組架構

建議採用 Modular Monolith。

```text
runtime/
├── cmd/
│   └── aia/
│
├── internal/
│   ├── app/
│   │
│   ├── agent/
│   │
│   ├── task/
│   │
│   ├── project/
│   │
│   ├── execution/
│   │
│   ├── provider/
│   │   ├── interface.go
│   │   └── antigravity/
│   │       ├── adapter.go
│   │       ├── process.go
│   │       ├── stream.go
│   │       ├── events.go
│   │       └── health.go
│   │
│   ├── profile/
│   │
│   ├── policy/
│   │
│   ├── workspace/
│   │
│   ├── validation/
│   │
│   ├── recovery/
│   │
│   ├── memory/
│   │
│   ├── storage/
│   │
│   └── audit/
│
└── migrations/
```

上述為邏輯模組劃分。

初期不需要每個模組都獨立成為 Go Package 或獨立程序，應避免過度工程化。

## 15.1 Provider Interface

AIA 不應讓 Task Manager 直接呼叫 agy。

建議透過 Provider Interface：

```go
type Provider interface {
    Name() string

    HealthCheck(ctx context.Context) error

    Start(ctx context.Context, req ExecutionRequest) (
        ExecutionSession,
        error,
    )
}

type ExecutionSession interface {
    Send(ctx context.Context, input Message) error

    Events() <-chan ExecutionEvent

    Stop(ctx context.Context) error

    Wait(ctx context.Context) (ExecutionResult, error)
}
```

此介面為概念性設計，正式實作時可依 Stream JSON 的實際行為調整。

未來可以增加其他 Provider，而不必修改 Task Manager。

# 16. Concurrency

AIA 未來可能同時執行多個 Agent。

例如：

```text
Personal Agent
    │
    ├── Coding Agent
    │      ├── Execution A
    │      └── Execution B
    │
    ├── Writing Agent
    │      └── Execution C
    │
    └── Research Agent
           ├── Execution D
           └── Execution E
```

每個 Execution 必須有獨立的：

- Process。
- Workspace。
- Session Settings。
- Event Stream。
- Execution State。

但下列資源可能是共享的，必須進一步確認：

- OAuth Credential。
- 使用者 Home Directory。
- agy Cache。
- Global Settings。
- MCP Server。
- Project 設定。
- Conversation Storage。

AIA 不得假設不同 Process 就自然具備完整的資料與權限隔離。

第一階段應限制最大並行數，並透過 PoC 確認並行執行的穩定性。

# 17. Logging 與 Audit

AIA 必須保存足以還原執行過程的紀錄。

至少包含：

- Task ID。
- Execution ID。
- Agent ID。
- Project ID。
- Provider。
- agy Version。
- 啟動參數。
- Profile ID 與版本。
- Workspace。
- Conversation ID。
- 開始與結束時間。
- Process Exit Code。
- Execution Result。
- Permission Denied。
- Tool Execution Events。
- 錯誤訊息。
- Result Validation。
- 最終 Task 狀態。

不得把 OAuth Token、密碼、Credential 或其他敏感資料寫入一般 Log。

原始輸出可能包含敏感內容，保存前應評估遮蔽與存取控制。

# 18. Error Handling

建議分類：

| 類型 | 處理方式 |
|---|---|
| PROVIDER_NOT_FOUND | 通知使用者安裝或設定 agy |
| VERSION_UNSUPPORTED | 停止執行並要求相容版本 |
| AUTH_REQUIRED | WAITING_FOR_AUTH |
| PERMISSION_DENIED | 依政策等待核准或失敗 |
| SANDBOX_ERROR | 停止 Execution 並記錄 |
| PROCESS_START_FAILED | FAILED 或 RETRYING |
| PROCESS_CRASHED | 保存紀錄後評估 Recovery |
| STREAM_PARSE_ERROR | 保存原始資料並標記異常 |
| EXECUTION_TIMEOUT | 依政策終止或等待 |
| INVALID_RESULT | 交由 Validator 判斷 |
| WORKSPACE_ERROR | 停止執行並保護既有成果 |

錯誤分類應保留原始錯誤內容，避免只依靠 stderr 的文字比對作為唯一判斷方式。

# 19. 不在第一階段實作的項目

以下功能先不自行開發：

1. 自製 LLM API Client。
2. 自製 OAuth。
3. 自製 Agent Planning Engine。
4. 自製 Subagent 執行引擎。
5. 自製 Tool Calling Engine。
6. 自製 MCP Client。
7. 自製底層 Tool Permission Engine。
8. 自製 OS-level Terminal Sandbox。
9. 自製模型推理服務。

這些能力優先使用 agy 提供的功能。

AIA 應將資源集中在：

- Agent 組織。
- 任務管理。
- 專案管理。
- 記憶。
- 高層政策。
- Execution Management。
- Result Validation。
- Recovery。
- Audit。
- Evolution。

# 20. 待驗證事項

以下項目尚未視為已完成設計，必須透過官方文件及 PoC 確認。

| 編號 | 項目 | 驗證目標 |
|---|---|---|
| V-01 | `--settings` | 確認 JSON 及檔案路徑用法 |
| V-02 | Settings Precedence | 確認 Session、Global、CLI 參數優先順序 |
| V-03 | Permission Mapping | 確認所有必要權限設定的正式 Schema |
| V-04 | Sandbox | 確認 Linux 實際隔離效果 |
| V-05 | Stream JSON | 確認 1.4.12 完整事件格式 |
| V-06 | Soft-deny | 確認人工核准需求的實際輸出與退出行為 |
| V-07 | Process Control | 確認取消、終止及逾時處理 |
| V-08 | Conversation Recovery | 確認恢復後的狀態與事件行為 |
| V-09 | Concurrent Execution | 確認多個 agy Process 可否穩定並行 |
| V-10 | OAuth Sharing | 確認獨立 Session 是否可使用既有登入狀態 |
| V-11 | Workspace Isolation | 確認不同工作目錄及額外目錄的實際限制 |
| V-12 | Agent Selection | 確認 `--agent` 與 Agent Manager 的使用方式 |
| V-13 | MCP | 確認 MCP Server 設定及 Session 作用範圍 |
| V-14 | Settings Isolation | 確認暫時設定不會污染使用者永久設定 |

在上述項目完成前，不應把未驗證能力寫成正式的系統保證。

# 21. 第一階段實作順序

建議依下列順序進行：

**Phase 1：Provider Integration**

- Go 啟動 agy。
- 取得版本及 Health Check。
- 完成 stdin / stdout / stderr 管理。
- 解析 JSON 及 NDJSON。
- 完成基本 Process Lifecycle。

**Phase 2：Execution Management**

- Task 與 Execution 分離。
- 保存 Execution State。
- 建立 Workspace。
- 保存 Conversation ID。
- 保存 Event Log。

**Phase 3：Profile & Policy**

- 建立 Execution Profile。
- 產生 Session Settings。
- 對應 AIA High-level Policy。
- 驗證 agy Permission。
- 驗證 Sandbox。

**Phase 4：Recovery & Validation**

- 處理 Soft-deny。
- 處理 Authentication Failure。
- 處理 Process Crash。
- 處理 Timeout。
- 驗證執行結果。
- 建立 Recovery 機制。

**Phase 5：Concurrency**

- 多 Execution 並行。
- Workspace 隔離。
- Session Settings 隔離。
- OAuth 共享狀態驗證。
- 資源限制與排程。

# 22. 架構決策摘要

| 項目 | 決策 |
|---|---|
| Runtime 語言 | Go |
| 系統型態 | Modular Monolith |
| AI 執行引擎 | Antigravity CLI |
| 整合方式 | External Process |
| 主要通訊 | Stream JSON |
| Session 設定 | `--settings` |
| 底層權限 | agy Permission Engine |
| OS Sandbox | agy Terminal Sandbox |
| 高層授權 | AIA |
| OAuth | 使用者與 agy |
| Workspace | AIA 管理，agy 執行 |
| Task Lifecycle | AIA |
| Execution Lifecycle | AIA |
| Agent Planning | 優先使用 agy |
| Subagent | 優先使用 agy |
| Memory | AIA |
| Recovery | AIA |
| Result Validation | AIA |
| Persistence | SQLite |
| 未來替換 Provider | 透過 Provider Interface |

# 23. 最終設計原則

AIA 的目的不是重新打造一個 Antigravity。

Antigravity 已經提供 Agent 執行所需的許多底層能力。AIA 應該善用這些能力，而不是重複開發。

AIA 真正需要建立的是一套能夠長期運作的個人 Agent 管理系統：

- 知道自己有哪些 Agent。
- 知道每個 Agent 負責什麼。
- 知道目前有哪些任務。
- 知道任務正在做什麼。
- 知道 Agent 可以使用哪些能力。
- 知道哪些操作需要使用者授權。
- 知道任務是否真正完成。
- 任務失敗時知道如何恢復。
- 能夠累積記憶並改善未來的工作方式。

agy 負責執行工作。

AIA 負責管理整個 Agent 世界。

這是兩者最重要的責任分界。
