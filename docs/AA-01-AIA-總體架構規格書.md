# AA-01：AIA 總體架構規格書

**Project:** AIA — Autonomous Intelligence Architecture  
**Subtitle:** Personal AI Agent Platform  
**Document ID:** AA-01  
**Version:** 1.0  
**Status:** Initial Architecture Specification  
**Implementation Language:** Go  
**Initial AI Integration:** Antigravity CLI (`agy`)  
**Initial Interface:** Command Line Interface (CLI)

---

# 1. 專案概述

## 1.1 專案定位

AIA（Autonomous Intelligence Architecture）是一套以個人為中心的自主 AI Agent 平台。

目標不是單純建立一個可以對話的 AI，而是建立一個能夠長期運作、理解目標、管理任務、使用工具、累積記憶、執行工作、檢查結果，並透過經驗持續改善的自主系統。

AIA 將 LLM 視為智慧能力的來源之一，而不是整個 Agent 本身。

Agent 的完整運作還需要：

- 任務管理與生命週期控制。
- 長期與短期記憶。
- 工作規劃與執行。
- 工具使用與權限管理。
- 執行結果驗證。
- 錯誤處理與復原。
- 經驗整理與自我改善。
- 人類監督與授權。

AIA 負責整合及管理這些能力，讓 Agent 能夠持續工作，而不是每次對話結束後就失去工作狀態。

## 1.2 核心目標

AIA 第一階段的核心目標：

1. 建立獨立於特定 LLM 的 Agent Runtime。
2. 建立完整的 Task Management 與 Agent Lifecycle。
3. 建立可持續保存的 Memory System。
4. 建立可控管的 Tool Calling 與 Permission System。
5. 建立多任務、多專案及多 Agent 的管理基礎。
6. 建立工作執行、結果驗證及錯誤復原機制。
7. 建立經驗整理與 Agent Evolution 機制。
8. 建立未來更換 AI Provider 或 Runtime 實作語言的能力。

## 1.3 非目標

第一階段不以以下項目為開發目標：

- Android APK。
- iOS App。
- 手機端 Agent Runtime。
- 多人協作平台。
- 雲端 SaaS 服務。
- 自行訓練大型語言模型。
- 自行實作 OAuth 登入及 Credential Management。
- 建立完整的圖形化管理介面。
- 一開始就實作所有 Domain Agent。

這些項目可以在核心架構成熟後再評估。

---

# 2. 系統設計原則

## 2.1 Agent 不等於 LLM

AIA 不將 LLM 等同於 Agent。

LLM 負責提供語言理解、推理、規劃及生成能力；AIA Runtime 負責管理 Agent 的整體運作。

LLM 可以被替換，但 Agent 的任務、記憶、權限、執行狀態及工作成果必須持續存在。

## 2.2 Runtime 掌握控制權

AIA Runtime 是整個系統的管理核心。

無論使用哪一種模型、CLI、工具或 Worker，都必須受到 Runtime 的管理。

Runtime 負責：

- 決定目前有哪些任務。
- 決定任務何時執行。
- 管理任務執行狀態。
- 管理 Agent 使用哪些能力。
- 管理工具的授權與執行。
- 保存任務與記憶。
- 驗證執行成果。
- 處理失敗與恢復。
- 記錄所有重要操作。

外部 AI 工具不得成為 AIA 唯一的任務管理中心。

## 2.3 架構與程式語言分離

AIA 的核心設計應避免依賴 Go 語言特有的實作方式。

第一階段使用 Go，是基於目前的開發效率及使用者技術背景，而不是將 Go 定義為 AIA 永久不可更換的技術。

設計時應注意：

- 核心資料模型保持明確。
- 模組之間透過定義清楚的 Interface 溝通。
- 外部程序透過標準化 Adapter 整合。
- 持久化資料使用明確的 Schema。
- 任務狀態及事件使用可序列化格式。
- 不將 Go 的 Goroutine、Channel 等機制直接暴露為核心業務規則。
- 不讓核心業務邏輯依賴特定 CLI 或模型的回應格式。

未來若改用 Kotlin，應能重新實作 Runtime 的模組，而不必改變 AIA 的核心概念與資料模型。

## 2.4 模組化單體架構

初期採用 Modular Monolith。

不急於採用 Microservices，也不為了未來可能擴充而提前建立分散式系統。

所有核心模組先在同一個 Go Runtime 中運作，但保持清楚的責任邊界。

未來若有必要，可以將特定模組獨立成服務或程序。

## 2.5 人類保有最終授權權

AIA 可以自主規劃及執行工作，但不代表所有操作都可以自動取得授權。

系統必須區分：

- 自主判斷。
- 自主規劃。
- 自主執行。
- 需要使用者授權的操作。
- 禁止執行的操作。

Agent 的自主程度必須受到 Policy、Permission 及 Risk Control 約束。

---

# 3. 系統總體架構

AIA 採用六層核心架構。

1. Runtime Layer
2. Intelligence Layer
3. Memory Layer
4. Capability Layer
5. Security Layer
6. Evolution Layer

六層共同組成完整的 Agent 運作環境。

## 3.1 架構示意

```text
+-------------------------------------------------------+
|                         AIA                           |
|                                                       |
|  +-------------------------------------------------+  |
|  |              6. Evolution Layer                 |  |
|  |  Reflection / Evaluation / Improvement / Version|  |
|  +-------------------------------------------------+  |
|                                                       |
|  +-------------------------------------------------+  |
|  |              5. Security Layer                  |  |
|  |  Policy / Permission / Validation / Audit       |  |
|  +-------------------------------------------------+  |
|                                                       |
|  +-------------------------------------------------+  |
|  |              4. Capability Layer                |  |
|  |  Tools / Workspace / Files / External Services  |  |
|  +-------------------------------------------------+  |
|                                                       |
|  +-------------------------------------------------+  |
|  |               3. Memory Layer                   |  |
|  |  Working / Episodic / Semantic / Procedural      |  |
|  +-------------------------------------------------+  |
|                                                       |
|  +-------------------------------------------------+  |
|  |             2. Intelligence Layer               |  |
|  |  Context / Planning / Decision / Reflection     |  |
|  +-------------------------------------------------+  |
|                                                       |
|  +-------------------------------------------------+  |
|  |                1. Runtime Layer                  |  |
|  |  Task / Event / State / Scheduler / Recovery    |  |
|  +-------------------------------------------------+  |
|                                                       |
+-------------------------------------------------------+
                         |
             +-----------+-----------+
             |                       |
       AI Provider              External Tools
       agy Adapter              Tool Adapters
             |                       |
       Antigravity CLI          File / Git / Web
                               Other Services
```

以上為邏輯架構，不代表六層必須各自成為獨立程序。

---

# 4. Runtime Layer

## 4.1 定位

Runtime Layer 是 AIA 的執行核心。

負責管理 Agent 的生命週期、任務、事件、狀態及執行流程。

Runtime 不負責提供智慧本身，而是確保智慧能力能夠被正確地組織、執行、保存及恢復。

## 4.2 核心模組

### Event Manager

負責系統事件管理。

事件來源包括：

- 使用者輸入。
- Task 建立或更新。
- Agent 執行結果。
- Tool 執行結果。
- Scheduler 觸發。
- 外部服務回報。
- 系統錯誤。
- 使用者授權或拒絕。
- Recovery 事件。

事件應具備：

- Event ID。
- Event Type。
- Timestamp。
- Source。
- Related Task ID。
- Payload。
- Correlation ID。

### Task Manager

負責任務管理。

每個 Task 必須具有：

- Task ID。
- Task Type。
- Task Description。
- Project ID。
- Assigned Agent。
- Priority。
- Current Status。
- Created At。
- Updated At。
- Parent Task ID（選用）。
- Dependencies（選用）。
- Deadline（選用）。
- Result。
- Error Information。

Task Manager 負責建立、更新、取消、分派及追蹤任務。

### State Manager

負責管理 Agent 與 Task 的執行狀態。

狀態不得只存在於記憶體中，重要狀態必須持久化。

系統應能在 Runtime 中斷後，重新載入任務狀態。

### Scheduler

負責安排工作執行時間。

支援：

- 立即執行。
- 延遲執行。
- 定期執行。
- 條件觸發。
- 任務相依性。
- 重試排程。

### Recovery Manager

負責處理異常及中斷。

包括：

- Runtime Crash。
- Worker Crash。
- AI Provider 失效。
- 外部服務逾時。
- 工具執行失敗。
- 任務執行狀態不一致。
- 系統重新啟動。

Recovery 必須先判斷任務是否可以安全恢復，不得直接重複執行可能造成重複副作用的操作。

---

# 5. Intelligence Layer

## 5.1 定位

Intelligence Layer 負責 Agent 的思考、理解、規劃及判斷。

透過 LLM 或其他 AI Provider 提供智慧能力。

這一層不直接掌握系統資源，也不應繞過 Runtime 執行未經授權的操作。

## 5.2 核心模組

### LLM Interface

統一不同 AI Provider 的呼叫方式。

初期整合 Antigravity CLI（agy）。

未來可以增加其他 Provider 或替換現有實作。

### Context Builder

負責建立模型執行所需的上下文。

資訊來源包括：

- 使用者需求。
- Task 資訊。
- Project 資訊。
- Working Memory。
- Relevant Episodic Memory。
- Semantic Memory。
- 可用工具。
- Policy 限制。
- 先前執行結果。

Context Builder 必須控制上下文大小及資訊相關性。

### Decision Engine

負責協助 Agent 判斷：

- 是否需要執行任務。
- 是否需要更多資訊。
- 是否需要呼叫工具。
- 是否需要詢問使用者。
- 是否應該停止工作。
- 是否應該將工作交給其他 Agent。

### Planning Engine

負責將目標拆解成可以執行的工作。

支援：

- Task Decomposition。
- Subtask Creation。
- Dependency Planning。
- Execution Planning。
- Plan Revision。

### Curiosity Engine

負責產生主動探索及延伸研究需求。

例如：

- 發現資訊不足。
- 發現潛在問題。
- 發現可以改善的地方。
- 發現值得進一步研究的議題。

Curiosity 不代表可以不受限制地執行操作。

所有探索工作仍須受到 Runtime、Policy 及資源限制管理。

### Reflection Engine

負責分析任務執行結果。

包括：

- 是否達成目標。
- 執行方式是否合理。
- 是否發生錯誤。
- 是否存在更好的方法。
- 是否需要更新記憶。
- 是否產生可重複使用的經驗。

Reflection 的結果可以提出改善建議，但不能自行修改核心系統或授權規則。

## 5.3 AI Provider 抽象

AIA 不應直接將 Antigravity 定義為 Intelligence Layer 本身。

Antigravity 是第一階段採用的 AI 能力整合對象。

架構上應保留：

```text
Intelligence Layer
        |
   LLM Interface
        |
   AI Provider Adapter
        |
   +----+----------------+
   |                     |
agy Adapter        Future Adapter
   |                     |
Antigravity CLI      Other Provider
```

實際上 agy 能承擔多少規劃、工具呼叫及 Agent 工作流程，必須由技術驗證確認。

AIA 不應預先假設 agy 只是純文字模型介面，也不應預先假設它可以完全取代 AIA Runtime。

---

# 6. Memory Layer

## 6.1 定位

Memory Layer 負責保存 Agent 的知識、經驗及工作狀態。

記憶不是單純的聊天紀錄，而是 Agent 長期運作的重要基礎。

## 6.2 四種記憶

### Working Memory

工作記憶。

保存目前正在進行的工作資訊。

例如：

- 目前 Task。
- 當前目標。
- 執行進度。
- 暫存資料。
- 目前的推理上下文。
- 尚未完成的決策。

Working Memory 可以隨任務結束而清理，但重要資訊應轉存到其他記憶類型。

### Episodic Memory

情節記憶。

保存 Agent 過去經歷的事件。

例如：

- 曾經執行的任務。
- 執行過程。
- 成功或失敗原因。
- 使用者的決策。
- 重要事件。
- 特定專案的發展歷程。

### Semantic Memory

語意記憶。

保存整理後的知識。

例如：

- 使用者的長期偏好。
- 專業領域知識。
- 專案背景。
- 技術規格。
- 歷史知識。
- 經過驗證的事實。

### Procedural Memory

程序記憶。

保存可以重複使用的工作方法。

例如：

- 程式開發流程。
- 文件產生流程。
- 測試流程。
- 特定工具的使用方法。
- 已驗證的問題處理程序。

## 6.3 Memory Management

Memory Layer 必須具備：

- Memory Creation。
- Memory Retrieval。
- Memory Update。
- Memory Consolidation。
- Memory Validation。
- Memory Expiration。
- Memory Relationship。
- Memory Provenance。

重要記憶應保留來源與可信度，避免把推測或模型生成內容直接當成事實。

---

# 7. Capability Layer

## 7.1 定位

Capability Layer 負責提供 Agent 可以使用的工具及外部能力。

Agent 可以提出工具使用需求，但工具的註冊、授權及執行應受到 AIA 控制。

## 7.2 核心模組

### Tool Registry

管理系統可使用的工具。

每個工具應定義：

- Tool ID。
- Tool Name。
- Description。
- Input Schema。
- Output Schema。
- Permission Requirement。
- Execution Timeout。
- Retry Policy。
- Side Effect Level。

### Tool Executor

負責實際執行工具。

工具執行必須經過：

1. Tool Registry 檢查。
2. Permission 檢查。
3. Policy 檢查。
4. Input Validation。
5. Tool Execution。
6. Output Validation。
7. Audit Logging。

### Workspace Manager

負責管理 Agent 的工作空間。

不同 Project 或 Task 可以使用不同 Workspace。

Workspace 應具備：

- 路徑管理。
- 存取權限。
- 檔案操作限制。
- 工作目錄隔離。
- 變更紀錄。
- 必要時的 Git 整合。

### File System

提供受控的檔案操作能力。

### Internet Access

提供受控的網路存取能力。

### Communication

提供訊息與通知能力。

### Productivity

提供文件、資料及其他生產力工具。

### Domain Capability

提供特定領域的專業工具。

例如：

- Coding。
- Writing。
- Investment。
- Video Production。

這些能力未必都需要在第一階段實作。

---

# 8. Security Layer

## 8.1 定位

Security Layer 負責確保 Agent 的行為符合使用者授權及系統規則。

安全機制必須存在於 Runtime 層級，而不是只依靠 Prompt 告訴模型哪些事情不能做。

## 8.2 核心模組

### Permission Manager

管理 Agent、Project、Task 及 Tool 的權限。

權限至少應區分：

- Read。
- Write。
- Execute。
- Delete。
- Network Access。
- External Communication。
- Financial Operation。
- Administrative Operation。

### Policy Engine

負責判斷操作是否符合政策。

Policy 可以依據：

- Agent 身分。
- Project。
- Task。
- Tool。
- 操作類型。
- 操作對象。
- 風險等級。
- 使用者設定。

### Action Validation

負責執行前及執行後的檢查。

例如：

- 路徑是否允許存取。
- 操作是否超出授權範圍。
- 輸入參數是否合法。
- 是否涉及不可逆操作。
- 執行結果是否符合預期。

### Audit Log

記錄重要系統操作。

至少包含：

- 操作時間。
- 操作來源。
- Agent ID。
- Task ID。
- Tool ID。
- 操作內容。
- 授權判斷。
- 執行結果。
- 錯誤資訊。

Audit Log 不應只依賴模型自行產生的敘述，應由 Runtime 在實際操作時記錄。

## 8.3 人類授權機制

以下操作可以依據使用者設定要求額外確認：

- 刪除重要資料。
- 對外發送訊息。
- 修改重要系統設定。
- 存取敏感資訊。
- 執行金融交易。
- 其他具有重大副作用的操作。

系統必須支援：

- Allow。
- Deny。
- Require Approval。

---

# 9. Evolution Layer

## 9.1 定位

Evolution Layer 負責讓 AIA 能夠從過去的工作中整理經驗，改善工作方法及 Agent 配置。

這一層的目標不是讓 Agent 任意修改自己的程式碼，而是建立可驗證、可追蹤及可回復的改善機制。

## 9.2 核心模組

### Reflection

分析過去任務的成功與失敗。

### Evaluation

評估 Agent 的工作品質。

評估項目可以包括：

- 任務完成率。
- 結果正確性。
- 執行時間。
- 資源使用量。
- 錯誤率。
- 使用者滿意度。
- 權限違規次數。

### Improvement Engine

根據評估結果提出改善建議。

改善範圍包括：

- Prompt。
- Agent Configuration。
- Tool Selection。
- Planning Strategy。
- Memory Retrieval。
- Workflow。

### Version Management

管理 Agent 及其配置版本。

### Snapshot

保存重要配置與系統狀態快照。

### Rollback

當改善結果不符合預期時，可以回復到先前版本。

## 9.3 改善流程

```text
Task Execution
      |
      v
Result Collection
      |
      v
Reflection
      |
      v
Evaluation
      |
      v
Improvement Proposal
      |
      v
Testing
      |
      v
Human Approval (when required)
      |
      v
Deployment
      |
      v
Monitoring
      |
      v
Rollback (if necessary)
```

所有涉及核心權限、安全政策及系統執行規則的變更，必須受到更嚴格的審核。

---

# 10. Agent 組織架構

AIA 採用分層 Agent 組織。

## 10.1 Personal Agent

Personal Agent 是使用者的主要 AI 代理人。

負責：

- 理解使用者需求。
- 管理整體目標。
- 管理 Project。
- 建立及分派 Task。
- 協調 Domain Agent。
- 維護使用者長期記憶。
- 整合各項工作成果。
- 向使用者回報。

Personal Agent 是使用者主要的工作入口。

## 10.2 Domain Agent

Domain Agent 負責特定專業領域。

初期規劃四個主要領域：

### Coding Agent

負責：

- 軟體開發。
- 程式碼分析。
- 程式碼修改。
- 測試。
- Debugging。
- 技術文件。

### Writing Agent

負責：

- 書籍寫作。
- 歷史研究。
- 資料整理。
- 章節規劃。
- 文件撰寫。
- 校勘與事實驗證。

### Investment Agent

負責：

- 市場資訊整理。
- 投資研究。
- 投資策略分析。
- 投資組合管理。
- 交易建議及授權流程。

涉及實際金融操作時，必須採用獨立且嚴格的權限與風險控制。

### Video Production Agent

負責：

- 影片企劃。
- 腳本撰寫。
- 素材整理。
- 影片製作流程。
- 影音內容管理。

## 10.3 Worker

Worker 是實際執行工作的一個單位。

Worker 可以是：

- AI CLI。
- LLM Agent。
- 外部程式。
- Go 程式模組。
- 自動化工作程序。
- 未來其他執行服務。

Worker 不應自行取得超出 Task 授權範圍的權限。

Worker 的執行結果必須回報 Runtime，由 Runtime 決定是否接受、重試或要求修正。

## 10.4 Agent 組織示意

```text
                    User
                     |
                     v
               Personal Agent
                     |
          +----------+----------+
          |          |          |
          v          v          v
       Coding     Writing   Investment
       Agent       Agent      Agent
          |          |          |
          v          v          v
       Workers    Workers    Workers
                     |
                     v
              Video Production
                   Agent
                     |
                     v
                  Workers
```

這是邏輯上的組織關係，不代表每個 Agent 都必須是一個獨立程序。

初期可以透過 Agent Configuration 與 Runtime 內部模組實現。

---

# 11. Task 與 Agent Lifecycle

## 11.1 Task Lifecycle

所有重要工作都應透過 Task 管理。

基本流程：

```text
CREATED
   |
   v
PENDING
   |
   v
PLANNING
   |
   v
READY
   |
   v
RUNNING
   |
   +----------------------+
   |          |           |
   v          v           v
COMPLETED  WAITING     FAILED
              |
              v
           RESUMING
              |
              v
           RUNNING
```

其他可能狀態：

- CANCELLED。
- BLOCKED。
- WAITING_FOR_AUTH。
- WAITING_APPROVAL。
- RETRYING。
- NEEDS_REVIEW。

Task 狀態轉換必須由 Runtime 控制，不能只依靠 LLM 自行宣稱完成。

## 11.2 Agent Lifecycle

Agent 的基本生命週期：

1. Initialization。
2. Context Loading。
3. Goal Understanding。
4. Planning。
5. Execution。
6. Result Validation。
7. Reflection。
8. Memory Update。
9. Completion。

Agent 可以在任務之間保持長期存在，但每次工作都必須有明確的 Task Context。

---

# 12. AI Provider 與 Antigravity 整合

## 12.1 第一階段技術選擇

第一階段使用 Antigravity CLI（agy）提供 AI 工作能力。

AIA Runtime 透過 Adapter 呼叫 agy，不直接依賴其內部實作。

## 12.2 OAuth 責任邊界

AIA 不自行實作 Google OAuth 登入流程，也不管理 agy 的 OAuth Token。

責任分配：

| 項目 | 負責者 |
|---|---|
| Google 帳戶登入 | 使用者 |
| 初次 OAuth 授權 | 使用者透過 agy |
| 必要時重新登入 | 使用者透過 agy |
| Credential 保存 | agy |
| Token 更新 | agy（以其實際支援能力為準） |
| agy 可用性檢查 | AIA Runtime |
| agy 程序呼叫 | AIA Runtime |
| Task 管理 | AIA Runtime |
| Memory 管理 | AIA Runtime |
| Tool Permission | AIA Runtime |
| Recovery | AIA Runtime |

AIA 不讀取、不複製、不解析 agy 的 OAuth Token。

## 12.3 授權失效處理

當 agy 回報登入失效或授權問題時：

1. Runtime 停止需要 AI Provider 的工作。
2. Task 狀態更新為 `WAITING_FOR_AUTH`。
3. 通知使用者需要重新登入。
4. 使用者自行透過 agy 完成登入或重新授權。
5. Runtime 再次檢查 agy 是否可用。
6. 恢復原 Task。

Runtime 不得嘗試繞過 agy 的授權機制。

## 12.4 尚待驗證的能力

在技術驗證完成前，不假設 agy 已具備以下能力：

- 穩定的非互動式 CLI 介面。
- 可程式化輸入及輸出。
- 可控的工作目錄。
- Session 管理。
- 執行逾時與取消。
- 工具呼叫限制。
- 可識別的授權錯誤。
- 可預測的退出碼及錯誤格式。

這些能力將列入 AA-03 技術驗證計畫。

---

# 13. 第一階段技術架構

## 13.1 開發語言

**Go**

原因：

- 使用者已具備 Go 開發經驗。
- 適合 CLI Runtime。
- 適合程序管理及系統整合。
- 具備 Goroutine 與 Channel 等並行處理能力。
- 部署簡單。
- 容易建立獨立執行檔。
- 適合建立長時間運作的背景程序。

## 13.2 技術選型

| 項目 | 第一階段選型 |
|---|---|
| Programming Language | Go |
| Architecture | Modular Monolith |
| AI Integration | Antigravity CLI |
| Database | SQLite |
| Data Format | JSON |
| Configuration | YAML / TOML / JSON |
| Interface | CLI |
| Version Control | Git |
| Logging | Structured Logging |
| Testing | Go Test |
| Deployment | Local Executable |

實際套件選擇另於技術設計文件決定。

## 13.3 初期執行環境

第一階段以桌面或伺服器環境為主。

主要工作包括：

- 程式開發。
- 文件撰寫。
- 歷史研究。
- 資料整理。
- Agent 架構實驗。
- 任務自動化。

不需要手機端執行環境。

## 13.4 Runtime 與 CLI

初期提供 CLI 作為主要操作介面。

預計支援：

```text
aia
  task create
  task list
  task show
  task cancel

  project create
  project list
  project show

  agent list
  agent show

  memory search
  memory list

  provider status

  system status
  system recover
```

以上為預計的命令結構，不代表第一階段全部實作。

---

# 14. 資料持久化架構

## 14.1 初期資料庫

第一階段採用 SQLite。

原因：

- 不需要獨立資料庫服務。
- 適合單機及個人使用。
- 支援交易。
- 支援持久化。
- 方便備份。
- 足以支援初期 Task、Project、Memory 及 Event 管理。

## 14.2 核心資料實體

初期至少需要以下資料實體：

| Entity | 用途 |
|---|---|
| Project | 專案管理 |
| Agent | Agent 定義與設定 |
| Task | 任務管理 |
| Task Event | 任務事件紀錄 |
| Task Dependency | 任務相依關係 |
| Memory | 記憶資料 |
| Tool | 工具定義 |
| Permission | 權限資料 |
| Policy | 系統政策 |
| Execution | 執行紀錄 |
| Audit Log | 稽核紀錄 |
| Agent Version | Agent 版本管理 |

## 14.3 資料設計原則

- 每個重要實體都應具有唯一 ID。
- 重要資料必須有建立及更新時間。
- Task 狀態必須可以持久化。
- Event 應保留歷史紀錄。
- 重要操作應可追蹤。
- Memory 應保留來源資訊。
- Schema 必須具備版本管理能力。
- 未來資料庫更換不應要求改寫核心業務邏輯。

---

# 15. Runtime 與未來 Kotlin 的相容性

## 15.1 設計目標

Go 是第一階段的實作語言。

未來若有需要，可以使用 Kotlin 重新實作 AIA Runtime，或建立 Kotlin 版本的 Client。

但不應為了這個可能性，在第一階段就同時開發兩套 Runtime。

## 15.2 必須保持穩定的部分

以下項目應盡量與程式語言無關：

- Agent 定義。
- Task Model。
- Task Lifecycle。
- Event Model。
- Memory Model。
- Tool Definition。
- Permission Model。
- Policy Model。
- Configuration Format。
- Persistent Data Schema。
- Provider Adapter Contract。

## 15.3 避免語言綁定

不應讓核心架構依賴：

- Go 專屬資料型態。
- Go 專屬序列化格式。
- Go 專屬的業務邏輯。
- Go Channel 作為跨模組的唯一協定。
- Go Runtime 的內部記憶體狀態。
- 特定 CLI 的原始輸出格式。

模組間的資料交換應採用明確的資料結構及契約。

## 15.4 未來 Kotlin 的可能實作

```text
                 AIA Architecture
                        |
          +-------------+-------------+
          |                           |
     Go Runtime                 Kotlin Runtime
     (Phase 1)                  (Future Phase)
          |                           |
          +-------------+-------------+
                        |
                Shared Data Model
                Shared Contracts
                        |
                 SQLite / Storage
```

未來 Kotlin 版本不必與 Go 版本同時存在，也不代表必須將 Go 程式碼逐行翻譯成 Kotlin。

真正需要保留的是架構、資料模型、行為契約及測試規格。

---

# 16. 系統安全與自主程度

AIA 的自主性不是無限制的。

系統應依照不同風險等級決定 Agent 可以採取的行動。

## 16.1 自主程度

| 等級 | 行為 | 控制方式 |
|---|---|---|
| L0 | 僅提供資訊 | 自主 |
| L1 | 提出建議與計畫 | 自主 |
| L2 | 執行可逆操作 | 依預先授權 |
| L3 | 執行具有外部副作用的操作 | 依 Policy 或要求確認 |
| L4 | 高風險或不可逆操作 | 明確授權或禁止 |

實際操作權限由 Permission Manager 與 Policy Engine 決定。

## 16.2 資源限制

Runtime 應能設定：

- Task 執行時間。
- Worker 數量。
- AI Provider 呼叫次數。
- Token 或費用預算（若 Provider 提供可用資訊）。
- Workspace 存取範圍。
- Tool 執行次數。
- 重試次數。
- 磁碟使用量。

## 16.3 安全優先原則

任何模型輸出都不能直接視為授權。

任何工具操作都必須經過 Runtime 控制。

任何任務完成宣告都必須經過結果驗證。

---

# 17. 開發階段規劃

## Phase 1：Core Runtime

目標：建立可以管理任務的 Go Runtime。

工作項目：

- Go Project Structure。
- CLI。
- SQLite。
- Project Manager。
- Task Manager。
- Event Manager。
- State Manager。
- 基本 Audit Log。

驗收目標：

可以透過 CLI 建立、查詢、更新及保存 Task，並在 Runtime 重新啟動後恢復任務資料。

## Phase 2：AI Integration

目標：整合 Antigravity CLI。

工作項目：

- Provider Interface。
- agy Adapter。
- Process Management。
- Request / Response。
- Timeout。
- Cancellation。
- Error Handling。
- Authentication Failure Handling。

驗收目標：

Runtime 可以透過 agy 執行一項受限工作，並取得可供後續處理的結果。

## Phase 3：Workspace & Capability

目標：建立受控的工具執行環境。

工作項目：

- Workspace Manager。
- Tool Registry。
- Tool Executor。
- File System Capability。
- Git Integration。
- Result Validator。
- Permission Check。

驗收目標：

Agent 可以在指定 Workspace 執行工作，Runtime 可以檢查實際檔案變更及測試結果。

## Phase 4：Memory System

目標：建立長期記憶。

工作項目：

- Working Memory。
- Episodic Memory。
- Semantic Memory。
- Procedural Memory。
- Memory Retrieval。
- Memory Consolidation。

驗收目標：

Agent 能夠在不同 Task 之間取得相關歷史資訊，並正確保存經驗。

## Phase 5：Agent Management

目標：建立 Personal Agent 與 Domain Agent。

工作項目：

- Agent Definition。
- Agent Configuration。
- Agent Lifecycle。
- Task Assignment。
- Task Decomposition。
- Worker Management。
- Multi-Agent Coordination。

驗收目標：

Personal Agent 可以將一項工作分派給適當的 Domain Agent，並整合執行結果。

## Phase 6：Evolution

目標：建立經驗改善循環。

工作項目：

- Reflection。
- Evaluation。
- Improvement Proposal。
- Agent Versioning。
- Snapshot。
- Testing。
- Approval。
- Rollback。

驗收目標：

Agent 可以根據過去任務提出改善建議，並透過測試及版本管理控制改善結果。

---

# 18. 第一階段成功標準

AIA 第一階段不是以功能數量作為成功標準，而是驗證核心架構是否成立。

必須具備：

1. Go Runtime 可以獨立啟動及運作。
2. Task 可以建立、保存及恢復。
3. Runtime 可以透過 Adapter 呼叫 AI Provider。
4. AI 工作可以在指定 Workspace 執行。
5. 工具操作受到 Runtime 權限控制。
6. 任務結果可以獨立驗證。
7. Runtime 中斷後可以安全處理未完成任務。
8. Memory 可以跨 Task 保存及檢索。
9. Agent 與 AI Provider 不存在不可替換的硬性依賴。
10. 核心資料模型不依賴 Go 專屬實作。

第一階段不要求實現所有六層的完整功能。

應先完成一個可以端到端運作的最小系統，再逐步擴充。

---

# 19. 尚待確認的架構議題

以下議題列為後續技術設計與驗證項目：

1. agy 的正式程式化呼叫方式及支援範圍。
2. agy 的 Agent Planning 與 Tool Calling 能力邊界。
3. AIA 是否能限制 agy 的內建工具。
4. AI Session 與 Task 的對應關係。
5. Worker 的程序隔離方式。
6. Workspace 的安全隔離程度。
7. Tool Calling 的標準化協定。
8. Memory Retrieval 的實作方式。
9. Task Recovery 的交易與冪等性設計。
10. Agent Evolution 的版本及核准機制。

這些議題不應在缺乏實測證據時直接定案。

---

# 20. 最終架構原則

AIA 的核心不是某個模型、某個 CLI、某種程式語言，也不是某個 Agent Framework。

AIA 是一套負責管理自主智慧系統的架構。

它必須能夠：

- 替換 AI 能力來源。
- 管理不同類型的 Agent。
- 管理長期任務。
- 保存持續性記憶。
- 使用外部工具。
- 控制操作權限。
- 驗證工作成果。
- 從錯誤中恢復。
- 從經驗中改善。
- 在必要時更換底層實作技術。

第一階段使用 Go 建立核心 Runtime，透過 Antigravity CLI 提供 AI 能力，以 CLI 作為主要操作介面。

先讓 AIA 真正具備自主工作的能力，再考慮圖形化介面、手機端及其他語言版本。

**AIA 的目標不是打造一個更聰明的聊天機器人，而是建立一個能夠長期自主運作、持續累積經驗並受到有效管理的個人智慧系統。**
