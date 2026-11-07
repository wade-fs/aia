# AIA-02 System Architecture

> [!NOTE]
> **架構重整與收攏公告 (Consolidated Architecture Specification)**
> 本規範為 AIA 歷史基準規範（規格編號已於 AIA-22 永久封頂）。為杜絕規格膨脹與重複定義，本規範之現行核心標準已正式收攏重整至 **[Volume 1: System Foundations & Architecture](file:///home/wade/src/github/aia/docs/specifications/volume-1-system-foundations.md)**。最新架構與維護基準請以對應之 Volume 為主，完整六大核心卷軸總覽請參閱 [Specification Catalog (README.md)](file:///home/wade/src/github/aia/docs/specifications/README.md)。

**Autonomous Intelligence Architecture — System Architecture Specification**

- 文件代碼：AIA-02
- 文件名稱：System Architecture
- 版本：v2.0
- 文件性質：系統總體架構規格
- 適用系統：AIA (Autonomous Intelligence Architecture)
- 主要實作語言：Go
- Agent Harness：Antigravity CLI（agy）
- 架構模式：Modular Monolith + Domain Agent Architecture
- 部署模式：Single User / Local First
- 狀態：目標架構規格；Phase 0 ～ Phase 6 核心元件與 Phase 7 整合已於 Go 代碼庫落地（註：部分外部 API 數據源仍處於隔離模擬中）

---

# 1. 文件目的

本文件定義 AIA 的整體系統架構、核心元件、責任邊界、資料流、執行模型、安全治理及擴充方式。

AIA 的目的不是單純建立一套多 Agent 任務管理工具，而是建立一個能夠長期運作、持續觀察環境、主動發現問題、自我修正並演化目標的個人 AI Agent 平台。

AIA 的架構必須支援以下四項自主性能力：

1. 主動性思維（Initiative）。
2. 時間連續性（Continuity）。
3. 自我修正（Self-Correction）。
4. 目標演化（Goal Evolution）。

這四項能力共同構成 AIA 的 Autonomy Core。

Runtime 提供持續執行的環境，Domain Agent 提供領域能力，Core Services 提供記憶、目標、任務、政策及稽核等共用服務。

本文件是 AIA-03 Go Runtime Implementation、AIA-05 Investment Agent、AIA-06 Web Interface 等文件的共同架構依據。

# 2. 架構設計原則

## 2.1 Autonomous Intelligence First

AIA 的設計重點是自主性，而不是 Agent 數量。

系統必須能回答：

- Agent 如何知道現在值得做什麼？
- Agent 如何決定是否應該採取行動？
- Agent 如何延續過去的研究與判斷？
- Agent 如何發現自己先前的判斷存在問題？
- Agent 如何修正自己的方法？
- Agent 如何在長期目標下提出新的研究方向？
- Agent 如何判斷目前沒有值得採取的行動？

新增工具或 Agent 並不直接代表自主性提升。

## 2.2 Separation of Concerns

系統應將自主性推理、持續執行、領域能力及安全治理分離。

| 層次 | 主要責任 |
|---|---|
| Autonomy Core | 自主性判斷與目標演化 |
| Runtime | 持續執行與生命週期管理 |
| Core Services | 記憶、任務、目標、政策、稽核 |
| Domain Agents | 領域研究與行動 |
| Agent Harness | AI 模型、工具執行及底層 Agent 能力 |
| Infrastructure | 資料庫、事件傳遞、外部資料來源 |

## 2.3 Persistent State

AIA 不應依賴單次對話或單次 Agent Session 保存長期狀態。

重要資訊必須持久化：

- Goal。
- Memory。
- Task。
- Decision。
- Strategy。
- Agent State。
- Execution History。
- Audit Event。

Agent Session 結束或 Runtime 重新啟動後，應能恢復先前的工作脈絡。

## 2.4 Human Governance

自主性不代表無限制的權限。

Agent 可以自主提出研究方向、執行允許範圍內的工作及修正自身策略，但不能自行解除治理限制。

以下事項必須保留人類控制：

- 重大目標變更。
- 權限擴張。
- 安全政策變更。
- 資源授權變更。
- 影響外部世界的高風險操作。

## 2.5 Incremental Evolution

架構應能逐步演進。

既有 Go Runtime、CLI、Web Console、Repository、Task、Memory、Policy、Audit 等成果應盡量保留。

架構升級不代表所有元件必須重新實作，而是重新明確定義它們在自主性系統中的責任。

# 3. 系統總體架構

## 3.1 Logical Architecture

```text
┌──────────────────────────────────────────────────────────┐
│                          AIA                             │
│                                                          │
│  ┌──────────────────────┐  ┌───────────────────────────┐ │
│  │     Autonomy Core    │  │       AIA Runtime         │ │
│  │                      │  │                           │ │
│  │  Initiative          │  │  Persistent Runtime       │ │
│  │  Continuity          │  │  Lifecycle Manager        │ │
│  │  Self-Correction     │  │  Schedule Worker          │ │
│  │  Goal Evolution      │  │  Task Worker              │ │
│  │                      │  │  Recovery Worker          │ │
│  │  Observe             │  │  Event Processing         │ │
│  │  Recall              │  │                           │ │
│  │  Assess              │  └───────────────────────────┘ │
│  │  Discover            │                                │
│  │  Deliberate          │  ┌───────────────────────────┐ │
│  │  Decide              │  │      Core Services        │ │
│  │  Learn               │  │                           │ │
│  │  Reassess            │  │  Agent Registry            │ │
│  └──────────────────────┘  │  Memory Service            │ │
│                            │  Goal Service              │ │
│  ┌──────────────────────┐  │  Task Service              │ │
│  │     Domain Agents    │  │  Project Service           │ │
│  │                      │  │  Policy Service            │ │
│  │  Investment Agent    │  │  Approval Service          │ │
│  │  Coding Agent        │  │  Audit Service             │ │
│  │  Writing Agent       │  │  Event Bus                 │ │
│  │                      │  └───────────────────────────┘ │
│  └──────────────────────┘                                │
│                                                          │
│  ┌──────────────────────────────────────────────────────┐│
│  │                 Agent Harness Layer                  ││
│  │                                                      ││
│  │  Antigravity CLI / Model / Tools / MCP / Sandbox     ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ┌──────────────────────────────────────────────────────┐│
│  │                 Infrastructure Layer                 ││
│  │                                                      ││
│  │  SQLite / Repositories / External Providers          ││
│  │  Filesystem / HTTP / Event Transport                 ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
└──────────────────────────────────────────────────────────┘
          ▲                              ▲
          │                              │
     CLI Interface                 Web Interface
          │                              │
          └──────────────┬───────────────┘
                         │
                    Human User
```

上述為邏輯架構，不代表目前所有元件都已經實作或完成整合。

## 3.2 Architecture Layers

AIA 分成六個主要層次：

1. Autonomy Core。
2. Runtime。
3. Core Services。
4. Domain Agents。
5. Agent Harness。
6. Infrastructure。

CLI 與 Web Interface 是使用者操作入口，不屬於自主性核心。

# 4. Autonomy Core

## 4.1 定位

Autonomy Core 是 AIA 的核心研究對象。

它負責讓 Agent 不只執行人類交付的任務，也能根據目標、環境、記憶及歷史經驗，自行判斷接下來應該做什麼。

Autonomy Core 是共用能力，不應分別在 Investment、Coding、Writing Agent 中各自實作一套。

## 4.2 四項核心能力

### A. Initiative — 主動性思維

負責判斷是否存在值得主動處理的事情。

能力包括：

- 主動觀察環境。
- 發現異常或變化。
- 發現未完成工作。
- 發現目標與現況之間的差距。
- 主動提出研究問題。
- 主動建立工作計畫。
- 判斷是否應該採取行動。
- 合理選擇不行動或休眠。

Initiative 不等於定時建立 Task。

Schedule 只能提供喚醒機制，真正的主動性必須來自 Agent 對環境、目標及記憶的評估。

### B. Continuity — 時間連續性

負責讓 Agent 能夠跨越不同執行週期延續自身狀態。

能力包括：

- 恢復過去目標。
- 恢復研究脈絡。
- 延續未完成工作。
- 追蹤歷史決策。
- 記得過去的預測及結果。
- 保留策略演化歷程。
- 維持長期任務狀態。

Continuity 不等於保存聊天紀錄。

系統必須透過結構化、可查詢、可追溯的持久化資料建立時間連續性。

### C. Self-Correction — 自我修正

負責檢查過去判斷、行動及結果，並修正錯誤或不適用的方法。

能力包括：

- 建立預測。
- 驗證預測。
- 分析結果與預期的差異。
- 發現推理或資料品質問題。
- 分析錯誤原因。
- 提出方法修正。
- 建立新版本。
- 驗證修正是否有效。

Self-Correction 不等於修改 Prompt。

必須有原始版本、修正理由、修正內容及後續驗證結果。

### D. Goal Evolution — 目標演化

負責維持長期目標，並根據經驗提出目標優化或修正。

包含：

- Goal Persistence：維持既定目標。
- Goal Optimization：優化達成目標的方法。
- Goal Revision：提出目標修正。

目標演化必須區分方法與目的。

Agent 可以自主改善方法，但重大目標變更必須經過人類核准。

Agent 不得自行修改治理政策，以便讓新的目標取得更大的執行權限。

## 4.3 Autonomy Cycle

Autonomy Core 的主要運作循環：

```text
Observe
   ↓
Recall
   ↓
Assess
   ↓
Discover
   ↓
Deliberate
   ↓
Decide
   ↓
Execute
   ↓
Verify
   ↓
Learn
   ↓
Reassess
   ↺
```

各階段責任：

| 階段 | 責任 |
|---|---|
| Observe | 觀察外部環境及內部狀態 |
| Recall | 取得歷史記憶及工作脈絡 |
| Assess | 評估目標、現況及問題 |
| Discover | 發現新問題、機會或研究方向 |
| Deliberate | 分析證據、假設及可能結果 |
| Decide | 決定行動、延後或不行動 |
| Execute | 交由 Runtime 與 Domain Agent 執行 |
| Verify | 驗證執行結果 |
| Learn | 更新記憶及經驗 |
| Reassess | 重新檢查目標及策略 |

這是一個邏輯流程，不要求所有步驟必須由同一個模型呼叫完成。

Autonomy Core 可以透過多個 Service、Agent Session 及 Task 協作完成。

## 4.4 Autonomy State

Autonomy Core 應具備可持久化的狀態。

至少包括：

- Current Goals。
- Active Context。
- Pending Questions。
- Discovered Opportunities。
- Known Risks。
- Ongoing Research。
- Previous Decisions。
- Expected Outcomes。
- Pending Evaluations。
- Current Strategies。
- Proposed Revisions。

狀態必須可在不同 Runtime Session 之間恢復。

## 4.5 Autonomy Evidence

自主性能力必須有可驗證的行為紀錄。

例如：

| 能力 | 可驗證證據 |
|---|---|
| Initiative | Agent 自主發現並建立的研究事項 |
| Continuity | 跨 Session 恢復的歷史脈絡及延續工作 |
| Self-Correction | 有證據支持的策略修正及後續驗證 |
| Goal Evolution | 目標持續、方法優化及目標修正提案 |

不得只使用模型自行生成的文字作為自主性已完成的證據。

# 5. AIA Runtime

## 5.1 定位

Runtime 是 AIA 的持續執行環境。

Autonomy Core 負責自主性判斷，Runtime 負責讓這些判斷能夠持續發生並轉化為實際工作。

Runtime 必須處理：

- 啟動與停止。
- Agent 生命週期。
- Schedule。
- Task 執行。
- Background Workers。
- Event Processing。
- Recovery。
- Session 管理。
- 執行狀態持久化。

## 5.2 Runtime Components

### Persistent Runtime

負責：

- Runtime 啟動。
- Runtime 狀態維護。
- Worker 管理。
- Graceful Shutdown。
- Runtime Health。
- 系統生命週期。

### Lifecycle Manager

負責管理 Agent 及 Worker 的生命週期：

- 建立。
- 啟動。
- 暫停。
- 恢復。
- 結束。
- 異常處理。

### Schedule Worker

負責：

- 讀取到期 Schedule。
- 處理排程時間。
- 處理 Misfire Policy。
- 建立對應 Task。
- 避免重複派發。

Schedule Worker 不直接負責完成領域工作。

### Task Worker

負責：

- 取得 READY Task。
- 檢查執行條件。
- 呼叫 Scheduler 執行。
- 管理 Task 狀態。
- 處理失敗與重試。
- 保存執行結果。

### Recovery Worker

負責：

- 偵測未完成 Task。
- 偵測異常中斷的 Session。
- 恢復允許恢復的工作。
- 處理逾時及失敗狀態。
- 避免重複執行不可重複的操作。

### Event Processing

負責處理系統事件：

- Task Created。
- Task Started。
- Task Completed。
- Task Failed。
- Goal Updated。
- Memory Updated。
- Decision Created。
- Approval Required。
- Runtime State Changed。

事件處理不得取代資料庫中的持久化狀態。

## 5.3 Schedule 與 Task 的區別

Schedule 與 Task 是兩個不同層次。

```text
Schedule
   │
   ▼
Schedule Worker
   │
   ▼
Create Task
   │
   ▼
Task Repository
   │
   ▼
Task Worker
   │
   ▼
Scheduler.RunTask()
   │
   ▼
Agent Execution
```

Schedule 表示何時應該建立工作。

Task 表示具體要執行的工作。

Scheduler 負責執行 Task。

三者不可混為一談。

## 5.4 Runtime 與 Autonomy Core 的關係

Runtime 喚醒 Autonomy Core，不代表 Runtime 自己具備主動性。

例如：

1. Schedule Worker 在指定時間喚醒 Agent。
2. Agent 恢復歷史狀態。
3. Autonomy Core 觀察環境。
4. Agent 評估是否有值得處理的事項。
5. Agent 決定研究、執行、延後或不行動。
6. Runtime 將決策轉成允許執行的 Task。
7. Task Worker 執行工作。
8. 結果保存至 Memory、Goal 及 Audit。

即使沒有新的 Schedule，系統也可以因為事件、工作完成、重大資料變化等原因重新評估是否需要行動。

## 5.5 Runtime Failure Handling

Runtime 必須考量：

- Process Crash。
- Worker Crash。
- Database Error。
- Network Failure。
- Agent Session Interrupted。
- External Tool Failure。
- Authentication Expired。
- Task Timeout。
- Duplicate Execution。

任務狀態必須能夠恢復。

不可因為 Runtime 重啟，就把尚未完成的工作當作成功完成。

# 6. Core Services

Core Services 提供所有 Domain Agent 共用的系統能力。

## 6.1 Agent Registry

負責：

- Agent 註冊。
- Agent Metadata。
- Agent Capability。
- Agent Status。
- Agent Configuration。
- Agent Lifecycle Metadata。

目前採 Static Registration。

Agent 與 Web Module 在編譯時註冊，不使用 Go Plugin 動態載入。

## 6.2 Memory Service

負責：

- 記憶儲存。
- 記憶檢索。
- 記憶分類。
- 記憶關聯。
- 記憶摘要。
- 記憶更新。
- 記憶來源追蹤。

記憶資料應保留來源、時間及關聯識別資訊。

Memory Service 是 Continuity 的基礎元件，但單純存在 Memory Service 不代表已完成時間連續性。

## 6.3 Goal Service

負責：

- Goal 建立。
- Goal 更新。
- Goal 狀態。
- Goal Hierarchy。
- Goal Relationship。
- Goal Version。
- Goal Evaluation。
- Goal Revision Proposal。

Goal Service 保存目標狀態。

Autonomy Core 負責評估目標是否需要持續、優化或提出修正。

## 6.4 Task Service

負責：

- Task 建立。
- Task 查詢。
- Task 狀態轉換。
- Task 關聯。
- Task 優先順序。
- Task 執行紀錄。
- Task Recovery Metadata。

Task 必須保留執行歷史及關聯 Goal、Agent、Project 等識別資訊。

## 6.5 Project Service

負責管理長期工作專案。

包括：

- Project Metadata。
- Project Goals。
- Project Context。
- Project Files。
- Project Task。
- Project Memory。
- Project Progress。

Project 是長期工作環境，不等於單一 Task。

## 6.6 Policy Service

負責：

- 工具權限。
- Agent 權限。
- 執行限制。
- 資源限制。
- 操作政策。
- 風險分類。
- 安全檢查。

Policy 是系統治理邊界，不應由 Agent 自行修改。

## 6.7 Approval Service

負責：

- Approval Request。
- Approval Decision。
- Approval Status。
- Approval History。
- Approval Expiration。

需要人類核准的操作必須先建立 Approval Request。

未取得核准不得執行受限制的操作。

## 6.8 Audit Service

負責保存：

- 使用者操作。
- Agent 決策。
- Task 執行。
- 工具呼叫。
- Policy 判斷。
- Approval。
- Runtime 事件。
- Recovery。
- 策略修正。
- 目標變更。

Audit 必須具備時間、關聯識別碼及來源。

已完成的 Audit 紀錄不得由 Agent 修改或刪除。

## 6.9 Event Bus

負責在 Core Services、Runtime 及 Domain Agents 之間傳遞事件。

Event Bus 用於即時通知與解耦。

需要可靠恢復的事件應有持久化機制或可由持久化狀態重新推導，不能只依賴記憶體中的事件佇列。

# 7. Domain Agent Architecture

## 7.1 定位

Domain Agent 是 AIA 面對不同工作領域的專業執行單元。

Domain Agent 不應自行重複實作 Autonomy Core。

它使用共用自主性能力，並提供領域專屬的：

- Domain Knowledge。
- Data Sources。
- Tools。
- Context。
- Research Process。
- Execution Methods。
- Verification Methods。

## 7.2 目前與規劃中的 Domain Agents

| Domain Agent | 領域 | 狀態 |
|---|---|---|
| Investment Agent | 投資研究與虛擬投資 | 已實作（Phase 1~5A，虛擬帳戶與世代實驗） |
| Coding Agent | 程式開發、除錯及優化 | 已實作 MVA（Phase 5B/6，代碼自修與受控沙盒） |
| Writing Agent | 寫作、內容及長期作品管理 | 已實作 MVA（Phase 5B/6，長篇小說規範稽核與一致性驗證） |

註：Coding Agent 與 Writing Agent 基礎領域引擎與 Observer 已就緒，目前正進行長週期端到端自主解決問題之整合驗證。

## 7.3 Domain Agent Contract

所有 Domain Agent 應遵循一致的介面契約。

概念上至少需要：

```text
DomainAgent
├── Identity()
├── Capabilities()
├── Observe(context)
├── Assess(context)
├── Research(context)
├── Propose(context)
├── Execute(action)
├── Verify(result)
└── Learn(result)
```

這是邏輯介面規格，不是對目前 Go Interface 已經具備上述所有方法的宣稱。

實際 Go Interface 應依 AIA-03 定義。

## 7.4 Domain Isolation

不同 Domain Agent 應維持領域責任隔離。

例如：

Investment Agent 不應直接修改 Coding Agent 的工作資料。

跨領域資訊交換應透過共用 Memory、Project、Goal、Task 或明確的訊息介面完成。

# 8. Agent Harness Integration

## 8.1 Antigravity CLI

AIA 使用 Antigravity CLI（agy）作為底層 Agent Harness。

agy 負責：

- Agent Session。
- 模型互動。
- 工具呼叫。
- 工具權限。
- MCP 整合。
- Filesystem 操作。
- Command 執行。
- Network 權限。
- Sandbox。

AIA 不應重新實作一套與 agy 重複的底層工具執行環境。

## 8.2 AIA 與 agy 的責任邊界

| 項目 | AIA | agy |
|---|---|---|
| 全域目標 | 管理 | 不負責 |
| Domain Agent 組織 | 管理 | 不負責 |
| Task | 管理 | 執行時配合 |
| 長期 Memory | 管理 | 不負責 |
| Autonomy Core | 管理 | 不負責 |
| Agent Session | 協調 | 管理 |
| Model Interaction | 透過 Harness 使用 | 執行 |
| Tool Execution | 提出授權需求及治理 | 執行 |
| Filesystem Permission | 定義 AIA 層級政策 | 執行 Harness 權限 |
| Command Permission | 定義 AIA 層級政策 | 執行 Harness 權限 |
| MCP | 協調使用 | 提供整合能力 |
| Sandbox | 不自行取代 | 管理 |

AIA 的 Policy 與 agy 的權限控制必須同時成立。

AIA 不得因為要提高自主性而繞過 agy 的安全限制。

## 8.3 Authentication

使用者自行透過 agy 完成 Google OAuth 或其他必要登入。

AIA 不讀取、不複製、不解析 OAuth Token。

當 agy 回報認證失效或執行需要人工登入時：

1. AIA 將相關 Task 標記為 `WAITING_FOR_AUTH`。
2. 提示使用者完成登入。
3. 登入完成後由使用者或授權流程恢復 Task。
4. 恢復前重新檢查 Task 狀態及執行條件。

不得把認證失敗直接當成 Task 成功。

## 8.4 Permission Handling

不得使用 `--dangerously-skip-permissions` 繞過工具權限。

Headless Mode 可能出現需要人工核准的工具操作，即使程序 Exit Code 為 0，也不能直接視為成功。

AIA 必須檢查：

- Tool Events。
- stderr。
- Harness Result。
- Task Result。
- Workspace Diff。
- Approval State。

# 9. Data Architecture

## 9.1 資料分層

AIA 的持久化資料應分為以下邏輯類別：

| 類別 | 內容 |
|---|---|
| Identity Data | User Profile、Agent Metadata |
| Control Data | Goal、Task、Schedule、Policy、Approval |
| Memory Data | 長期記憶、研究脈絡、摘要 |
| Execution Data | Session、Tool Event、Execution Result |
| Domain Data | Investment 等領域資料 |
| Audit Data | 不可竄改的稽核事件 |
| Configuration Data | Runtime、Provider、Agent 設定 |

## 9.2 Repository Pattern

Domain Service 不應直接依賴特定 SQL 實作。

資料存取應透過 Repository Interface。

```text
Domain Service
      │
      ▼
Repository Interface
      │
      ▼
SQLite Repository
      │
      ▼
SQLite Database
```

目前以 SQLite 作為主要持久化儲存。

未來如需更換資料庫，應盡量限制在 Infrastructure 層。

## 9.3 Event 與 State

Event 表示發生過的事情。

State 表示目前系統狀態。

兩者不能混為一談。

例如：

```text
TaskCreated
TaskStarted
TaskCompleted
```

是事件。

```text
Task.Status = COMPLETED
```

是狀態。

重要事件應可追溯，狀態應能在重新啟動後恢復。

## 9.4 Snapshot

對於需要事後重現的研究及決策，應建立不可變 Snapshot。

例如 Investment Agent：

- Market Snapshot。
- Research Snapshot。
- Portfolio Snapshot。
- Decision Snapshot。

Snapshot 必須保留來源及建立時間。

不得用事後取得的資料覆寫當時 Snapshot。

## 9.5 Data Provenance

所有重要外部資料應保留來源資訊：

- Source。
- Retrieved At。
- Published At。
- Data Timestamp。
- Content Hash。
- Provider Version。
- Quality Status。

Mock Data 與 Real Data 必須隔離。

# 10. Execution Architecture

## 10.1 Task Execution Flow

```text
User / Autonomy Core / Schedule
              │
              ▼
          Task Service
              │
              ▼
        Task Repository
              │
              ▼
          Task Worker
              │
              ▼
       Scheduler.RunTask()
              │
              ▼
          Agent Registry
              │
              ▼
         Domain Agent
              │
              ▼
         Agent Harness
              │
              ▼
        Tool Execution
              │
              ▼
        Result Collection
              │
       ┌──────┴──────┐
       ▼             ▼
   Verification    Audit
       │
       ▼
   Memory / Goal / Task Update
```

Task 執行必須具備明確狀態及錯誤處理。

## 10.2 Task State

Task 狀態至少應區分：

- READY。
- RUNNING。
- COMPLETED。
- FAILED。
- CANCELLED。
- WAITING_FOR_APPROVAL。
- WAITING_FOR_AUTH。

實際狀態名稱以程式實作及 AIA-03 定義為準。

狀態轉換必須有合法性檢查。

## 10.3 Idempotency

可能被重試或重複派發的工作，必須考量 Idempotency。

例如：

- 重複建立同一個排程 Task。
- 重複收集同一交易日行情。
- 重複評估同一筆 Prediction。
- 重複處理同一個外部事件。

對不可重複的操作，必須有唯一識別碼或其他防重機制。

## 10.4 Retry

Retry 必須區分：

- 暫時性錯誤。
- 永久性錯誤。
- 權限錯誤。
- 認證錯誤。
- 資料錯誤。
- 外部服務錯誤。

不能對所有錯誤採用無限制重試。

## 10.5 Recovery

Recovery 應根據持久化狀態判斷：

- 哪些 Task 可以恢復。
- 哪些 Task 必須重新執行。
- 哪些 Task 必須等待人工處理。
- 哪些 Task 必須標記失敗。
- 哪些操作需要先進行結果核對。

Recovery 不得自行假設外部操作已成功或失敗。

# 11. User Interface Architecture

## 11.1 CLI

CLI 是主要的命令列管理介面。

負責：

- 系統操作。
- Agent 管理。
- Task 管理。
- Project 管理。
- Memory 查詢。
- Investment 操作。
- Schedule 管理。
- Runtime 管理。

CLI 不應直接取代 Core Service 的業務邏輯。

## 11.2 Web Console

Web Console 提供視覺化管理與操作。

主要區域：

- Overview。
- Autonomy Monitor。
- Agent Registry。
- Task Management。
- Schedule。
- Memory。
- Goal。
- Policy。
- Approval。
- Audit。
- Domain Workbench。

目前只有 Investment Module 有實際領域功能。

Coding 與 Writing 的 Web Module 不代表其 Domain Agent 已完成。

## 11.3 Web Host 與 Runtime 分離

Web Host 與 Runtime 是不同的系統職責。

```text
AIA Web Host
├── HTTP Server
├── Authentication
├── Session
├── API Routes
└── Web Console

AIA Runtime
├── Lifecycle Manager
├── Schedule Worker
├── Task Worker
├── Recovery Worker
├── Event Processing
└── Autonomy Core
```

Web Server 啟動不代表 Runtime 已啟動。

未來可以採用：

1. 同一程序內啟動 Web Host 與 Runtime。
2. 分離為 Web Process 與 Runtime Process。

不論採取哪一種部署方式，兩者都必須有明確的生命週期與狀態管理。

目前程式實作不可被描述為已經具備上述完整 Runtime 分離及常駐執行能力。

# 12. Security Architecture

## 12.1 Single User

AIA v1/v2 初期維持 Single User。

- 單一擁有者。
- 不實作多使用者帳號系統。
- 不實作租戶。
- 不實作一般企業 RBAC。

## 12.2 Local First

預設只允許本機存取。

Web Host 預設：

```text
127.0.0.1
```

不得預設開放 LAN 或 Internet。

如果未來要開放遠端存取，必須另行設計認證、授權、TLS、網路邊界及攻擊面控制。

## 12.3 Authentication

Web Console 必須使用安全登入流程。

登入機制應包含：

- One-time Login Token。
- Token Validation。
- Session Creation。
- HttpOnly Cookie。
- Session Expiration。
- Logout。
- CSRF Protection。

## 12.4 Authorization

每個受限制操作都必須經過 Policy 檢查。

高風險操作需要 Approval。

Agent 不得自行提高權限或解除政策限制。

## 12.5 Audit Integrity

Audit 必須保留：

- 操作者。
- 事件來源。
- 時間。
- 關聯 ID。
- 操作內容。
- 結果。

Agent 不得修改或刪除歷史稽核紀錄。

## 12.6 Financial Safety

Investment Agent 初期只進行虛擬投資。

不得連接真實證券帳戶自動下單。

不得持有使用者資金。

不得將虛擬交易介面誤認為真實券商交易介面。

# 13. Deployment Architecture

## 13.1 Initial Deployment

初期部署以單機為主。

```text
Local Machine
│
├── AIA CLI
│
├── AIA Web Host
│
├── AIA Runtime
│   ├── Workers
│   ├── Scheduler
│   └── Autonomy Core
│
├── SQLite
│
├── Workspace
│
└── Antigravity CLI
```

這是目標部署模型，不代表目前所有程序都已經能依此方式啟動。

## 13.2 Process Model

未來應支援明確的程序管理：

- `aia server start`：Web Server。
- `aia runtime start`：Runtime Daemon。
- `aia runtime status`：Runtime 狀態。
- `aia runtime stop`：停止 Runtime。

以上 Runtime 命令是目標介面，不能在尚未實作前當成現有命令。

目前 `aia server start` 只啟動 Web Server。

## 13.3 Resource Management

Runtime 必須管理：

- Worker 數量。
- 同時執行 Task 數量。
- Agent Session 數量。
- 外部 API 呼叫頻率。
- 記憶體使用量。
- Task Timeout。
- Retry 次數。

自主性不代表可以無限制消耗資源。

Agent 應能在資源限制內選擇行動、延後或休眠。

# 14. Observability

AIA 必須具備可觀測性，才能研究自主性是否真的發生。

## 14.1 Runtime Metrics

包括：

- Runtime Uptime。
- Worker Status。
- Task Queue Size。
- Task Execution Duration。
- Task Failure Rate。
- Recovery Count。
- Schedule Delay。
- External Provider Availability。

## 14.2 Autonomy Metrics

包括：

- Initiative Events。
- Autonomous Task Creation。
- Continuity Recovery。
- Historical Context Retrieval。
- Prediction Evaluation。
- Self-Correction Events。
- Strategy Version Changes。
- Goal Evolution Proposals。
- Human Approval Events。

## 14.3 Decision Trace

每個重要決策都應能追溯：

```text
Decision
├── Goal
├── Context
├── Memory References
├── Observation Snapshot
├── Evidence
├── Reasoning Summary
├── Confidence
├── Policy Evaluation
├── Action
├── Execution Result
└── Verification
```

決策追蹤應保存可稽核的依據，不要求公開模型內部隱藏推理過程。

# 15. Extensibility

## 15.1 Static Registration

目前採 Static Registration。

Agent 與 Web Module 在編譯時註冊。

不採用 Go Plugin 動態載入。

## 15.2 Domain Extension

未來新增 Domain Agent 時，應遵循：

1. 實作 Domain Agent Contract。
2. 定義領域資料模型。
3. 定義領域專用 Service。
4. 定義資料來源及工具。
5. 定義驗證方法。
6. 註冊 Agent。
7. 建立 Web Module（如有需要）。
8. 整合共用 Autonomy Core。
9. 建立領域驗收測試。

新增 Domain Agent 不應複製一套 Memory、Goal、Task、Policy、Audit 或 Autonomy Core。

## 15.3 Language Independence

AIA 初期使用 Go 實作。

架構設計應盡量維持語言中立：

- Domain Contract 明確。
- Service 責任清楚。
- 資料模型穩定。
- API 可定義。
- 持久化格式可移植。
- 執行協定不依賴 Go Plugin。

未來若有需要，可考慮使用 Kotlin 或其他語言重寫部分元件，而不改變整體系統概念。

# 16. Current Implementation Mapping

本章區分目標架構與目前 Go 程式實作。

| 架構元件 | 目前狀態 | 說明 |
|---|---|---|
| Go Application Bootstrap | 已有 | 建立主要服務與資源 |
| SQLite Persistence | 已有 | 透過 Repository 存取 |
| Agent Registry | 已有基礎 | Static Registration |
| Task Service | 已有 | Task 管理 |
| Scheduler | 已有基礎 | Task 執行方法存在 |
| Schedule Service | 已有基礎 | 可建立排程及 Task |
| Memory Service | 已有基礎 | 持久化與檢索能力仍須依自主性需求擴充 |
| Goal Service | 需依實際功能驗收 | 不等於已完成 Goal Evolution |
| Policy Service | 已有基礎 | 仍須驗證完整治理流程 |
| Approval Service | 已有基礎 | 依實際操作驗收 |
| Audit Service | 已有基礎 | 需確認完整性及不可竄改要求 |
| Event Bus | 已有 | 需確認持久化事件需求 |
| Investment Module | 已實作 | 支援多投資組合與虛擬結算 (Phase 5A) |
| Coding Agent | 已實作 MVA | 沙盒代碼修復與驗證管線 (Phase 5B/6) |
| Writing Agent | 已實作 MVA | 長篇大綱與多層知識一致性驗證 (Phase 5B/6) |
| Autonomy Core | 已實作 | 11 步共用決策自主閉環 (Phase 3/3.5/7) |
| Persistent Runtime | 已實作 | 常駐 Schedule/Task/AutoWake/Purger Workers |
| Schedule Worker | 已實作 | 15s 週期常駐輪詢 |
| Task Worker | 已實作 | 5s 週期常駐並行槽排程執行 |
| Recovery Worker | 已實作 | 支援 Context Reconstruction 與重啟復原 |
| Initiative Engine | 已實作 | 多來源主動機會與缺口發現 (Phase 3) |
| Self-Correction Engine | 已實作 | 三層式自我修正與反饋驗證 (Phase 4) |
| Goal Evolution | 已實作 | 目標生命週期與三向演化 (Phase 4) |
| Web Host | 已實作 | HTTP Server 與 16-Tab Web Console |
| Autonomy Monitor | 已實作 | AIA-13 通用自主性度量引擎 (evaluator/metrics) |

## 16.1 Current Runtime Reality

目前程式碼的實際執行狀況：

1. `aia server start` 啟動 Web Server。
2. `aia schedule tick` 手動處理到期 Schedule。
3. Schedule Service 可以建立 Task。
4. Scheduler 有執行 Task 的方法。
5. `aia investment sync` 手動執行資料同步。
6. `aia investment evaluate` 手動執行預測評估。
7. Application 啟動時會執行一次 Recovery。

但目前沒有完整的常駐 Background Worker 串接上述工作。

因此：

- Web Server 常駐不代表 Runtime 常駐。
- Schedule 存在不代表 Task 自動執行。
- Task Scheduler 存在不代表有 Worker。
- Memory 存在不代表 Continuity 已完成。
- Prediction Evaluation 存在不代表 Self-Correction 已完成。

這些限制必須反映在 README、AIA-03、AIA-04、AIA-06 及 AIA-07。

# 17. Global Roadmap

AIA 全域開發順序：

| 階段 | 名稱 | 主要目標 |
|---|---|---|
| R1 | Persistent Runtime | 建立持續執行及工作恢復 |
| R2 | Continuity & Memory | 建立跨週期狀態與記憶延續 |
| R3 | Initiative Engine | 建立主動觀察、發現及決策 |
| R4 | Reasoning & Self-Correction | 建立推理、驗證及自我修正 |
| R5 | Goal Evolution | 建立目標持續、優化及修正 |
| R6 | Autonomous Domain Experiments | 在領域中驗證自主性 |

## 17.1 R1 — Persistent Runtime

重點：

- Runtime Lifecycle。
- Schedule Worker。
- Task Worker。
- Recovery Worker。
- Background Execution。
- Graceful Shutdown。
- Runtime Status。

## 17.2 R2 — Continuity & Memory

重點：

- Long-term Memory。
- Persistent Agent State。
- Historical Context Retrieval。
- Goal Continuity。
- Strategy History。
- Session Recovery。

## 17.3 R3 — Initiative Engine

重點：

- Environment Observation。
- Goal Gap Detection。
- Opportunity Discovery。
- Proactive Research。
- Action Prioritization。
- No-Action Decision。

## 17.4 R4 — Reasoning & Self-Correction

重點：

- Evidence Evaluation。
- Hypothesis Testing。
- Prediction Evaluation。
- Error Classification。
- Strategy Revision。
- Revision Verification。

## 17.5 R5 — Goal Evolution

重點：

- Goal Persistence。
- Goal Optimization。
- Goal Revision Proposal。
- Goal Versioning。
- Human Approval。
- Goal Evaluation。

## 17.6 R6 — Autonomous Domain Experiments

以 Investment Agent 作為第一個完整實驗場域。

後續再逐步擴展 Coding Agent、Writing Agent 等其他領域。

Domain Agent 的實驗結果應回饋 Autonomy Core 的設計與驗證。

# 18. Architecture Acceptance Criteria

AIA 架構必須依照下列原則驗收：

## 18.1 Separation of Concerns

- Autonomy Core 不依賴特定 Domain Agent。
- Domain Agent 使用共用自主性能力。
- Runtime 不取代 Autonomy Core。
- Schedule 不等於 Task。
- Task 不等於 Agent Decision。
- Agent Harness 不取代 AIA 的長期狀態管理。

## 18.2 Persistence

- Runtime 中斷後能恢復必要狀態。
- Goal、Task、Memory、Decision 均有持久化紀錄。
- 歷史事件可追溯。
- 重要工作不依賴單次 Session。

## 18.3 Autonomy

- Agent 能根據環境及目標主動發現工作。
- Agent 能合理決定不行動。
- Agent 能延續歷史脈絡。
- Agent 能驗證自己的判斷。
- Agent 能提出並驗證修正。
- Agent 能維持目標並提出目標演化建議。

## 18.4 Governance

- Policy 不可由 Agent 自行解除。
- Approval 必須具備可追溯紀錄。
- Audit 不可由 Agent 修改或刪除。
- 高風險操作須經治理程序。
- 外部操作不得繞過 Harness 權限。

## 18.5 Domain Isolation

- 新增 Domain Agent 不需要重建 Autonomy Core。
- Domain 資料及工具保持領域隔離。
- 共用服務透過明確介面使用。
- 各 Domain 有獨立的驗證方式。

# 19. 最終架構原則

AIA 的架構不是為了讓 AI 能夠執行更多命令，而是為了讓 AI 能夠在長時間跨度中，持續理解自己的目標、環境及過去經驗。

因此，整體架構必須維持以下分工：

**Autonomy Core 決定如何自主思考與演化。**

**Runtime 讓自主行為能夠跨越時間持續發生。**

**Core Services 保存目標、記憶、任務及治理狀態。**

**Domain Agents 在各自領域中研究、判斷及行動。**

**Agent Harness 提供底層模型與工具執行能力。**

**Human Governance 保留必要的控制權及安全邊界。**

AIA 最終要驗證的不是 Agent 能否完成一次任務，而是它能否在長期運作中，持續觀察、主動發現、延續脈絡、修正自己，並在既定治理邊界內演化目標。

---

**文件結束**
