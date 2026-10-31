# AIA-01：Personal Agent 產品需求規格書

**專案名稱：** AIA — Autonomous Intelligence Architecture  
**產品定位：** Personal AI Agent Platform  
**文件版本：** 1.0  
**產品版本目標：** AIA Personal Agent v1  
**狀態：** Baseline Requirements  
**主要開發語言：** Go  
**第一階段執行平台：** Linux Desktop / CLI  
**AI 執行引擎：** Antigravity CLI 1.4.12  
**資料庫：** SQLite

---

# 1. 產品目標

## 1.1 產品定義

AIA 是一套運行於個人電腦上的 Personal AI Agent 系統。

AIA 不只是 LLM 的命令列包裝程式，也不只是 Antigravity CLI 的啟動器。

AIA 必須具備獨立的任務管理、Agent 組織、專案管理、記憶、政策控制、執行監督、結果驗收及恢復能力。

AIA 使用外部 AI Agent 執行引擎完成需要模型推理及工具操作的工作。第一階段使用 Antigravity CLI（agy），但 AIA 的核心資料模型及業務邏輯不得綁定 agy。

## 1.2 核心目標

AIA Personal Agent v1 必須能夠：

1. 接收使用者以自然語言提出的工作目標。
2. 理解目標並判斷任務類型。
3. 將複雜目標拆解成可執行的 Task。
4. 判斷由 Personal Agent 自行處理，或交由 Domain Agent。
5. 管理多個 Agent 的任務及執行狀態。
6. 呼叫 Antigravity CLI 執行工作。
7. 監控 Agent 的執行事件及工具操作。
8. 保存 Task、Execution、Conversation、Event 及成果。
9. 管理跨 Task 的長期 Memory。
10. 根據 Memory 提供後續工作所需的上下文。
11. 在需要授權、補充資訊或重新登入時暫停任務。
12. 驗證工作成果是否符合原始目標。
13. 在失敗或程序中斷後恢復任務。
14. 評估 Agent 的工作品質，累積可改善的經驗。

## 1.3 非目標

第一版不實作以下項目：

- Android APK、iOS App 或 Web UI。
- 多使用者 SaaS 平台。
- 多主機分散式 Runtime。
- 自製 LLM 推理服務。
- 自製 OAuth 登入及 Token 管理。
- 自製底層 Tool Calling Engine。
- 自製底層 MCP Client。
- 自製 OS-level Sandbox。
- 自動修改 AIA 自身程式碼並直接部署。
- 未經使用者授權的金融交易。
- 未經使用者授權的外部帳戶操作。
- 不受限制的背景自主執行。

第一版以單一使用者、單一 Linux 主機、CLI 操作為範圍。

---

# 2. 使用者及操作模型

## 2.1 使用者角色

第一版只有一種系統使用者：

`OWNER`

OWNER 擁有：

- 建立及取消任務的權限。
- 建立及修改 Project 的權限。
- 建立及修改 Agent 定義的權限。
- 管理執行政策的權限。
- 核准高風險操作的權限。
- 管理 Memory 的權限。
- 查看所有執行紀錄的權限。
- 決定是否接受任務成果的權限。

Agent 不得自行取得 OWNER 權限。

## 2.2 操作方式

使用者透過 AIA CLI 操作系統。

主要操作：

```bash
aia chat
aia task list
aia task show <task-id>
aia task cancel <task-id>
aia task resume <task-id>

aia agent list
aia agent show <agent-id>

aia project list
aia project show <project-id>

aia memory list
aia memory search <query>

aia execution show <execution-id>
aia doctor
aia status
```

CLI 命令為第一版正式介面，詳細規格另見 AIA-03。

---

# 3. Agent 組織需求

## 3.1 Agent 層級

AIA 必須支援以下三種邏輯角色：

### Personal Agent

系統的主要協調者。

負責：

- 接收使用者目標。
- 理解及澄清需求。
- 任務分類。
- 任務拆解。
- 選擇 Domain Agent。
- 管理任務依賴關係。
- 監督執行進度。
- 綜合各 Agent 成果。
- 決定是否需要使用者介入。
- 進行最終結果驗收。

Personal Agent 是 AIA 的全域協調者，不等同於單一 agy Conversation。

### Domain Agent

負責特定領域的工作。

第一版預設建立：

| Agent ID | 名稱 | 工作範圍 |
|---|---|---|
| `coding` | Coding Agent | 程式開發、除錯、測試、程式碼審查 |
| `writing` | Writing Agent | 文件、文章、研究整理及寫作 |
| `investment` | Investment Agent | 投資資料研究、財務資訊整理、風險分析 |
| `video` | Video Production Agent | 影片企劃、腳本、分鏡、製作流程及素材管理 |

Domain Agent 可以建立 Subtask，但不得自行突破 AIA 的授權政策。

Investment Agent 第一版只負責研究、分析及資訊整理，不具備金融交易權限。

### Worker

Worker 是實際執行某一次工作的執行單位。

Worker 可以由 agy Process / Conversation 實現。

Worker 不等於 AIA Domain Agent。

同一個 Domain Agent 可以同時擁有多個 Worker，但並行數必須受 AIA 控制。

## 3.2 Agent 與 agy Agent 的區別

AIA Agent 是 AIA 的邏輯角色、責任及政策定義。

agy Agent 是 Antigravity CLI 可選用的執行 Agent。

兩者不可視為同一個資料實體。

AIA Agent 定義必須包含：

- ID
- Name
- Description
- Domain
- System Instructions
- Capabilities
- Default Execution Profile
- Provider Mapping
- Enabled
- Version

`Provider Mapping` 可以指定 agy 的 `--agent` 值，也可以留空，使用 agy 預設 Agent。

---

# 4. Task Management 需求

## 4.1 Task 定義

Task 是使用者或 Agent 要求系統完成的工作單位。

每個 Task 必須具備：

- 唯一 ID。
- 所屬 Project。
- 標題。
- 目標描述。
- 建立來源。
- 負責 Agent。
- 優先順序。
- 狀態。
- 驗收條件。
- 建立時間。
- 更新時間。

## 4.2 任務拆解

Personal Agent 必須能夠將複雜目標拆成多個 Subtask。

每個 Subtask 必須定義：

- 工作目標。
- 負責 Agent。
- 輸入資料。
- 預期成果。
- 驗收條件。
- 依賴的其他 Task。
- 執行順序。

AIA 必須檢查依賴關係，禁止循環依賴。

簡單任務可以不拆解，直接建立單一 Task。

## 4.3 任務執行原則

- Task 可以有多次 Execution。
- Task 的完成狀態由 AIA 決定。
- Execution 的成功不代表 Task 必然完成。
- Agent 不得自行修改 Task 的最終狀態。
- Task 的驗收條件在執行前必須存在。
- 若驗收條件不足，Personal Agent 應先補充或詢問使用者。

## 4.4 任務狀態

正式狀態：

- `CREATED`
- `READY`
- `RUNNING`
- `WAITING_FOR_AUTH`
- `WAITING_FOR_APPROVAL`
- `WAITING_FOR_INPUT`
- `VALIDATING`
- `RETRYING`
- `COMPLETED`
- `FAILED`
- `CANCELLED`
- `SUSPENDED`

合法狀態轉移由 Runtime 狀態機定義，不允許任意修改。

---

# 5. Personal Agent 自主工作需求

## 5.1 工作循環

Personal Agent 必須實作以下工作循環：

```text
Receive
   ↓
Understand
   ↓
Context Retrieval
   ↓
Classify
   ↓
Plan
   ↓
Authorize
   ↓
Dispatch
   ↓
Monitor
   ↓
Validate
   ↓
Learn
   ↓
Report
```

### Receive

接收使用者輸入。

### Understand

理解使用者真正要完成的目標。

若存在會影響正確性的重要歧義，必須詢問使用者，不可自行補造事實。

### Context Retrieval

從 Project、Memory、既有 Task 及歷史成果取得相關上下文。

不得無限制將全部 Memory 塞入 Prompt。

### Classify

判斷任務類型、風險、複雜度及適合的 Agent。

### Plan

產生任務計畫及 Subtask。

### Authorize

確認 Agent、Project、工具、Workspace 及操作風險是否符合政策。

### Dispatch

將 Task 分派給 Domain Agent，並建立 Execution。

### Monitor

接收 Execution Event，更新任務進度。

### Validate

驗證實際成果。

### Learn

整理可能值得保留的 Memory 或工作經驗。

### Report

向使用者回報成果、未完成事項及需要決策的問題。

## 5.2 自主程度

第一版採用受政策約束的自主執行。

AIA 可以自行：

- 分類任務。
- 建立低風險 Subtask。
- 選擇 Domain Agent。
- 建立 Execution。
- 監控執行。
- 執行允許範圍內的重試。
- 驗證成果。
- 提出 Memory 候選項目。

AIA 不得自行：

- 提升自身權限。
- 修改 OWNER 的授權政策。
- 自行解除 Deny。
- 自行取得使用者 Credential。
- 自行執行未授權的金融交易。
- 未經授權操作外部帳戶。
- 未經驗收直接覆寫重要原始資料。
- 修改自己的核心程式並直接部署。

---

# 6. Memory 需求

## 6.1 Memory 目標

Memory 是 AIA 的長期知識與經驗管理系統。

Memory 不應只是把所有對話永久保存。

AIA 必須能夠判斷：

- 哪些資訊具有長期價值。
- 哪些資訊只在目前 Task 有效。
- 哪些資訊屬於 Project。
- 哪些資訊是可重複使用的工作經驗。
- 哪些資訊可能過期。
- 哪些資訊彼此矛盾。
- 哪些資訊不應自動保存。

## 6.2 Memory 分類

第一版採用四種 Memory：

| 類型 | 用途 | 範例 |
|---|---|---|
| Working | 當前工作上下文 | 正在處理的 Task 狀態 |
| Episodic | 過去事件與執行經驗 | 某次程式修改及驗收結果 |
| Semantic | 穩定知識及使用者偏好 | 專案技術選擇 |
| Procedural | 可重複使用的方法 | 某類 Task 的成功工作流程 |

## 6.3 Memory 儲存欄位

每筆 Memory 至少包含：

- ID
- Type
- Scope
- Content
- Source
- Confidence
- CreatedAt
- UpdatedAt
- LastUsedAt
- Status
- Version
- ExpirationAt（可為空）

Scope 包含：

- Global
- Project
- Agent
- Task

## 6.4 Memory 寫入原則

AIA 可以自動建立 Memory Candidate。

正式寫入長期 Memory 前，必須依 Memory 類型及敏感程度套用政策。

下列內容不可因為出現在對話中就自動保存：

- 密碼及 Token。
- Credential。
- 個人敏感資料。
- 未經確認的推測。
- 一次性、短期且無長期價值的內容。

對於會影響未來任務的重要使用者偏好，必須保留來源及可信度。

## 6.5 Memory Retrieval

Memory Retrieval 第一階段採用：

1. Scope 過濾。
2. 類型過濾。
3. 關鍵字搜尋。
4. 時間及相關性排序。
5. 限制回傳數量。
6. 將選取結果加入 Prompt Context。

向量資料庫不是第一版必要條件。

---

# 7. Project Management 需求

Project 是工作及資料的管理範圍。

每個 Project 必須具備：

- ID
- Name
- Description
- Workspace Path
- Default Agent
- Policy Profile
- Status
- CreatedAt
- UpdatedAt

Project 必須可以：

- 建立。
- 修改。
- 列出。
- 暫停。
- 封存。
- 關聯 Task。
- 關聯 Memory。
- 指定允許使用的 Workspace。

不同 Project 預設不得任意互相讀取資料。

---

# 8. Execution 與 Provider 需求

## 8.1 Execution

Execution 是 Task 的一次實際執行。

每次 Execution 必須記錄：

- Execution ID
- Task ID
- Agent ID
- Provider ID
- Workspace
- Profile Version
- Conversation ID
- Process ID
- 開始時間
- 結束時間
- Exit Code
- Result Status
- Error
- Usage
- Artifact References

## 8.2 Provider

第一版實作 Antigravity Provider。

Provider 必須支援：

- Health Check。
- Version Check。
- Process Start。
- Input Stream。
- Output Stream。
- Event Parsing。
- Process Stop。
- Process Wait。
- Result Collection。
- Conversation Tracking。

AIA 核心邏輯不得直接依賴 agy 原生 JSON 欄位。

## 8.3 Provider 替換

未來替換 Provider 時，不得要求重寫：

- Task Model。
- Agent Model。
- Project Model。
- Memory Model。
- Policy Model。
- Task Lifecycle。
- Result Validation。
- Audit。

---

# 9. Security 需求

## 9.1 高層授權

AIA 負責：

- Agent 是否有權承接任務。
- Task 是否可以存取指定 Project。
- Workspace 是否允許使用。
- 是否允許特定類型的操作。
- 是否需要 OWNER 核准。
- 是否允許外部副作用。

## 9.2 底層權限

agy 負責：

- 檔案讀寫。
- 命令執行。
- 網路存取。
- MCP Tool。
- Terminal Sandbox。

AIA 必須將高層政策映射到 agy 可用的權限設定。

## 9.3 預設安全原則

- 預設拒絕未授權操作。
- 不使用 `--dangerously-skip-permissions`。
- 不讀取或保存 OAuth Token。
- 不把使用者 Home Directory 整體加入 Workspace。
- 不將 Credential 寫入 Prompt 或 Log。
- 高風險操作必須等待 OWNER 核准。
- 未能確認權限是否生效時，停止相關 Execution。

## 9.4 Headless Approval

Headless Mode 無法直接提供一般互動式核准流程。

需要人工核准的操作可能被 soft-deny。

因此 AIA 的核准流程為：

1. 偵測被拒絕的工具操作。
2. 將 Task 設為 `WAITING_FOR_APPROVAL`。
3. 顯示操作內容及風險。
4. 等待 OWNER 決定。
5. 若核准，重新建立符合授權範圍的 Execution。
6. 若拒絕，維持封鎖或取消任務。

不得假設原本被 soft-deny 的工具會在使用者核准後自動繼續。

---

# 10. CLI 功能需求

第一版必須提供：

| 命令 | 功能 |
|---|---|
| `aia chat` | 與 Personal Agent 對話 |
| `aia task create` | 建立 Task |
| `aia task list` | 列出 Task |
| `aia task show` | 查看 Task |
| `aia task resume` | 恢復 Task |
| `aia task cancel` | 取消 Task |
| `aia agent list` | 列出 Agent |
| `aia agent show` | 查看 Agent |
| `aia project create` | 建立 Project |
| `aia project list` | 列出 Project |
| `aia project show` | 查看 Project |
| `aia memory list` | 列出 Memory |
| `aia memory search` | 搜尋 Memory |
| `aia memory approve` | 核准 Memory Candidate |
| `aia execution show` | 查看 Execution |
| `aia status` | 查看系統狀態 |
| `aia doctor` | 檢查 Runtime 及 Provider |

CLI 必須提供人類可讀輸出及可選的 JSON 輸出。

---

# 11. 非功能需求

## 11.1 Reliability

- Task 及 Execution 狀態必須持久化。
- Runtime 非正常結束後，重新啟動必須能辨識未完成任務。
- Event 必須有序號。
- 重要狀態轉移必須可稽核。
- 不得因為重試而無條件重複外部副作用。

## 11.2 Security

- 本機資料庫預設只供本機使用。
- 不將敏感資訊寫入一般 Log。
- Agent 不得修改 AIA 的系統政策。
- 不可信任的 Agent 輸出不得直接當成系統指令執行。

## 11.3 Maintainability

- Go Modular Monolith。
- 清楚的模組責任邊界。
- Provider Adapter 隔離。
- Database Migration。
- 單元測試及整合測試。
- 不以過度抽象增加開發成本。

## 11.4 Portability

第一版以 Linux 為主要支援平台。

資料模型、Provider Interface、業務邏輯不得依賴 Linux 專屬 API。

未來可以重新使用 Kotlin 實作 Client 或 Runtime，但必須遵守相同的資料模型及行為契約。

---

# 12. 第一版驗收條件

AIA Personal Agent v1 必須至少通過以下驗收：

1. 可以啟動 AIA CLI。
2. 可以檢查 agy 是否存在及可用。
3. 可以建立 Project。
4. 可以建立 Task。
5. Personal Agent 可以將一個複雜目標拆成多個 Subtask。
6. 可以將 Subtask 分派給不同 Domain Agent。
7. 可以透過 agy 執行工作。
8. 可以即時接收 Execution Event。
9. 可以保存 Conversation ID。
10. 可以保存執行結果。
11. 可以在程序中斷後恢復或正確標示未完成狀態。
12. 可以建立及搜尋 Memory。
13. 可以將 Memory 加入後續 Task Context。
14. 可以處理 Authentication Failure。
15. 可以處理 Permission Denied。
16. 可以執行 Result Validation。
17. 可以保留完整 Audit Log。
18. 可以在使用者取消時停止 Execution。
19. 不會因為 agy 回傳 SUCCESS 就直接把未驗收的 Task 標成 COMPLETED。
20. 可以連續執行多個 Task，且資料不互相污染。

第一版的完成定義是：AIA 已經能獨立管理自己的 Agent、Task、Memory 及 Execution，而不是只能把 Prompt 傳給 agy。

---

# 13. 需求優先級

| 優先級 | 功能 |
|---|---|
| P0 | Runtime、Provider、Task、Execution、Project、SQLite |
| P0 | Personal Agent 基本任務規劃與分派 |
| P0 | Stream JSON、Event、Process Control |
| P0 | Policy、Profile、Workspace |
| P0 | Result Validation、Recovery、Audit |
| P1 | Memory 四類及 Retrieval |
| P1 | Domain Agent 管理 |
| P1 | 多 Execution 並行 |
| P1 | Memory Candidate 與評估 |
| P2 | 進階 Memory Retrieval |
| P2 | Agent Performance Evaluation |
| P2 | Evolution / Reflection |
| P2 | 多 Provider 支援 |

P0 是第一個可運作版本的必要條件。P1 必須納入 Personal Agent v1 完整交付。P2 可在 v1 基礎上逐步擴充。
