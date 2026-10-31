I Agent 總體架構規格書

**文件名稱：** AI Agent Architecture Specification  
**文件版本：** 1.0  
**專案代號：** Personal Autonomous Agent  
**主要開發語言：** Kotlin  
**初期運行平台：** Android 專用裝置（核心架構保持平台獨立）  
**文件狀態：** 初始設計

---

# 第一章：專案理念

## 1.1 專案目標

本專案旨在建立一個具有持續運作能力、自主判斷能力、學習能力及外部操作能力的個人 AI Agent。

它不是單純的聊天機器人，也不是只能依照固定指令執行工作的自動化程式。

我們希望建立的 Agent 能夠：

1. 接收並理解外部資訊。
2. 保存及運用過去的經驗。
3. 主動發現值得研究的問題。
4. 評估情況並作出判斷。
5. 規劃並執行任務。
6. 觀察行動結果。
7. 檢討自己的判斷及行為。
8. 透過持續改善，逐步提升能力。

Agent 不必時時刻刻採取行動。

等待、觀察、拒絕執行及尋求協助，都應該是合法的決策結果。

## 1.2 核心設計觀念

### Agent 不等於 LLM

LLM 是 Agent 的思考能力來源之一，但不是 Agent 本身。

Agent 必須具有獨立於 LLM 的執行環境，負責管理：

- 生命週期。
- 任務及事件。
- 狀態。
- 記憶。
- 排程。
- 權限。
- 外部工具。
- 行動結果。
- 錯誤恢復。

即使 LLM 暫時無法使用，Agent 的 Runtime 仍應能維持基本運作、保存狀態及管理待處理工作。

### Agent 必須具有持續性

一般聊天系統的對話結束，不代表系統仍然持續工作。

本專案的 Agent 必須能夠跨越多次對話、App 關閉及裝置重新啟動，延續尚未完成的任務與長期目標。

### 自主權逐步建立

自主能力不是一次授予，而是隨著功能驗證、可靠性及使用者授權逐步增加。

## 1.3 設計原則

| 原則 | 說明 |
|---|---|
| Runtime First | 先建立可靠的執行環境 |
| Event Driven | 以事件驅動工作流程 |
| Explicit State | 所有重要狀態明確保存 |
| Persistent | 任務與記憶可以跨重啟延續 |
| Incremental Autonomy | 逐步增加自主權 |
| Human Control | 使用者保有最終控制權 |
| Auditable | 重要決策及行動均可追溯 |
| Evolvable | 架構與能力可以持續演進 |
| Fail Safe | 異常時優先停止不安全行動 |

---

# 第二章：系統總體架構

## 2.1 架構分層

系統分為六個主要層次。

1. Runtime Layer
2. Intelligence Layer
3. Memory Layer
4. Capability Layer
5. Security Layer
6. Evolution Layer

各層具有明確責任，不允許 LLM 直接繞過 Runtime 或 Security Layer 操作外部環境。

## 2.2 系統架構圖

```text
                User / External Environment
                           |
                           v
                 +-------------------+
                 |   Interaction     |
                 |   Event Sources   |
                 +-------------------+
                           |
                           v
+------------------------------------------------+
|                 Runtime Layer                  |
|                                                |
|  Event Manager                                 |
|  Task Manager                                  |
|  State Manager                                 |
|  Scheduler                                     |
|  Worker Manager                                |
|  Recovery Manager                              |
+------------------------------------------------+
                           |
                           v
+------------------------------------------------+
|               Intelligence Layer              |
|                                                |
|  LLM Interface                                 |
|  Context Builder                               |
|  Decision Engine                               |
|  Planning Engine                               |
|  Reflection Engine                             |
|  Curiosity Engine                              |
+------------------------------------------------+
                           |
                           v
+------------------------------------------------+
|                  Memory Layer                  |
|                                                |
|  Working Memory                                |
|  Episodic Memory                               |
|  Semantic Memory                               |
|  Procedural Memory                             |
+------------------------------------------------+
                           |
                           v
+------------------------------------------------+
|                 Capability Layer               |
|                                                |
|  Tool Registry                                 |
|  Device Control                                |
|  File System                                   |
|  Internet                                      |
|  Communication                                 |
|  Domain Capabilities                           |
+------------------------------------------------+
                           |
                           v
+------------------------------------------------+
|                  Security Layer                |
|                                                |
|  Permission Manager                            |
|  Policy Engine                                 |
|  Risk Control                                  |
|  Action Validation                             |
|  Audit Log                                     |
+------------------------------------------------+

                 Evolution Layer
          Versioning / Evaluation / Improvement
```

以上是邏輯架構，不代表所有模組都必須在第一階段實作。

---

# 第三章：Runtime Layer

Runtime 是整個 Agent 的執行核心。

它負責讓 Agent 持續存在，並管理所有工作流程。

## 3.1 Event Manager

Event 描述系統中發生的事情。

事件來源包括：

- 使用者輸入。
- 定時事件。
- 外部通知。
- 網路資訊更新。
- 工具執行結果。
- 任務完成。
- 系統異常。
- 裝置狀態改變。

Event 本身不代表必須執行某個動作。

它只是提供 Runtime 一項需要處理的事實。

## 3.2 Task Manager

Task 描述 Agent 接下來要完成的工作。

每個 Task 至少具有：

- Task ID。
- 任務名稱。
- 任務內容。
- 任務來源。
- 優先級。
- 建立時間。
- 截止時間（可選）。
- 目前狀態。
- 執行紀錄。
- 完成條件。

Task 狀態：

```text
CREATED
  |
PENDING
  |
RUNNING
  |
+------------------+
|                  |
COMPLETED        FAILED
|                  |
ARCHIVED         RETRY / ABORTED
```

任務必須能在 App 關閉、裝置重新啟動後恢復。

## 3.3 State Manager

負責管理 Agent 的明確狀態。

初始狀態：

- INITIALIZING
- IDLE
- OBSERVING
- THINKING
- PLANNING
- EXECUTING
- WAITING
- REFLECTING
- ERROR
- STOPPED

狀態轉換必須由 Runtime 管理，不允許 LLM 任意修改系統狀態。

## 3.4 Scheduler

負責管理：

- 單次任務。
- 定時任務。
- 週期任務。
- 截止時間。
- 事件觸發任務。
- 延遲重試。

Scheduler 不依賴 LLM 才能運作。

## 3.5 Recovery Manager

負責：

- 系統重新啟動後恢復狀態。
- 找出未完成任務。
- 判斷中斷時的執行狀態。
- 防止重複執行。
- 恢復可安全重試的工作。
- 將無法確認的工作標記為 UNKNOWN，等待進一步查證。

重要外部操作不得因為重啟而自動重複執行。

---

# 第四章：Intelligence Layer

這一層負責 Agent 的思考與判斷。

## 4.1 LLM Interface

LLM 是可替換的思考引擎。

必須提供統一介面，避免其他模組直接依賴特定模型或 SDK。

基本能力：

- Text Generation。
- Structured Output。
- Reasoning Request。
- Tool Proposal。
- Context Processing。

模型輸出必須經過驗證，不得直接視為可信指令。

## 4.2 Context Builder

負責整理思考所需資訊：

- 當前事件。
- 目前任務。
- Agent 狀態。
- 相關記憶。
- 歷史決策。
- 可使用工具。
- 權限及政策。
- 任務限制。

避免無限制地將所有記憶送入 LLM。

## 4.3 Decision Engine

負責判斷是否需要採取行動。

初始決策類型：

- IGNORE
- WAIT
- INVESTIGATE
- PLAN
- EXECUTE
- NOTIFY
- ASK
- ABORT

決策至少包含：

- Decision ID。
- 判斷理由。
- 使用資訊。
- 信心程度。
- 優先級。
- 建議行動。
- 預期結果。
- 下次評估時間。

## 4.4 Planning Engine

將較大的目標拆解成可以執行的小型任務。

必須支援：

- 任務分解。
- 前置條件。
- 任務依賴。
- 完成條件。
- 失敗處理。
- 重新規劃。

## 4.5 Curiosity Engine

讓 Agent 能夠主動提出值得探索的問題。

問題必須經過評估，而不是不斷產生無意義的研究任務。

評估項目：

- Novelty。
- Relevance。
- Importance。
- Feasibility。
- Expected Value。
- Cost。

## 4.6 Reflection Engine

負責檢討：

- 當時掌握的資訊。
- 當時作出的判斷。
- 預期結果。
- 實際結果。
- 判斷錯誤。
- 執行錯誤。
- 資訊遺漏。
- 可改善項目。

Reflection 可以提出改善建議，但不能自行修改 Runtime、權限及安全政策。

---

# 第五章：Memory Layer

Memory 是 Agent 長期持續運作的重要基礎。

## 5.1 Working Memory

保存目前思考及任務所需的暫時資訊。

## 5.2 Episodic Memory

保存實際發生的經驗：

- 事件。
- 任務。
- 決策。
- 行動。
- 結果。
- 反思。

## 5.3 Semantic Memory

保存經過整理的知識、概念、事實及其關係。

必須保留來源、時間及可信程度。

## 5.4 Procedural Memory

保存經驗驗證後的工作方法、流程及操作規則。

不得將單次成功直接視為可靠程序。

## 5.5 記憶管理原則

每筆重要記憶至少保留：

- Memory ID。
- 類型。
- 內容。
- 來源。
- 建立時間。
- 最後驗證時間。
- 信心程度。
- 重要性。
- 關聯事件及任務。

記憶發生衝突時，必須保留不同來源及時間，不得直接覆蓋舊資料。

---

# 第六章：Capability Layer

Capability 是 Agent 與外部世界互動的能力。

## 6.1 Tool Registry

所有可用工具必須先註冊。

每個工具定義：

- Tool ID。
- 名稱。
- 功能。
- 輸入格式。
- 輸出格式。
- 權限需求。
- 風險等級。
- 執行限制。
- 逾時設定。

LLM 只能提出工具呼叫，由 Runtime 驗證後執行。

## 6.2 初期能力分類

| 類別 | 能力 |
|---|---|
| Information | 搜尋、閱讀、資料擷取 |
| File System | 讀取、建立及整理檔案 |
| Device Control | 螢幕觀察、觸控、鍵盤輸入 |
| Communication | 訊息及通知 |
| Productivity | 行事曆、待辦及文件 |
| Domain | 特定應用領域的專業能力 |

不同能力可以逐步加入，不必在初期全部實作。

## 6.3 Device Control

專用裝置是 Agent 操作外部軟體的重要入口。

初期以 Android 為主，研究：

- Accessibility Service。
- UI Automation。
- Screen Capture。
- OCR。
- Touch Input。
- Keyboard Input。
- App State Detection。

使用者負責安裝 App、初始登入、身分驗證及服務開通。

Agent 在使用者授權的操作環境中執行工作。

裝置操作仍須遵守 Android 權限、App 安全機制及各服務使用條款。

---

# 第七章：Security Layer

安全機制必須獨立於 LLM。

## 7.1 Permission Manager

管理：

- 工具使用權限。
- 資料讀取權限。
- 資料修改權限。
- 外部操作權限。
- 金融操作權限。
- 使用者授權範圍。

## 7.2 Policy Engine

所有重要行動必須先經過政策檢查。

政策應採用明確、可測試及可追溯的規則。

## 7.3 Action Validation

執行前檢查：

1. 工具是否存在。
2. 權限是否足夠。
3. 參數是否正確。
4. 行動是否符合政策。
5. 是否可能重複執行。
6. 是否需要使用者確認。

## 7.4 Audit Log

重要事件、決策、工具呼叫及執行結果必須留下不可任意覆寫的歷史紀錄。

---

# 第八章：Evolution Layer

Evolution Layer 是讓 Agent 持續改善的機制。

它不只是程式碼版本管理，而是整個 Agent 行為與能力的演進管理。

## 8.1 Agent Evolution Version

每一個可識別的 Agent 版本，必須記錄其組成：

- Runtime Version。
- Model Version。
- Prompt Version。
- Memory Schema Version。
- Decision Version。
- Planning Version。
- Tool Configuration Version。
- Policy Version。
- Domain Strategy Version（適用時）。

這些版本共同構成某次實驗或運行時的 Agent Configuration Snapshot。

## 8.2 Improvement Cycle

```text
Observe
   |
Collect Experience
   |
Reflect
   |
Identify Problem
   |
Propose Improvement
   |
Test
   |
Evaluate
   |
Approve
   |
Deploy New Configuration
   |
Observe Again
```

## 8.3 改善原則

1. Agent 可以自主提出改善建議。
2. 改善必須有問題紀錄及依據。
3. 新版本必須先測試。
4. 重要變更必須保留舊版本。
5. 改善結果必須能比較。
6. 不允許未經授權修改安全政策。
7. 必須能夠回復到先前可靠的版本。

Git 可以用來管理程式碼，但 Agent Evolution Version 必須另外保存完整的配置及行為評估紀錄。

---

# 第九章：技術架構

## 9.1 開發語言

主要使用 Kotlin。

選擇原因：

- Android 原生整合能力。
- Coroutines 與 Flow。
- 結構化並行。
- Kotlin Serialization。
- Room / SQLite。
- 未來可延伸至 Kotlin Server。

## 9.2 初期技術選擇

| 項目 | 技術方向 |
|---|---|
| Language | Kotlin |
| Android UI | Jetpack Compose |
| Async | Coroutines / Flow |
| Local Database | Room / SQLite |
| Serialization | Kotlin Serialization |
| Background Work | WorkManager |
| Long-running Visible Work | Foreground Service |
| Logging | Structured Logging |
| LLM | Provider Abstraction |

## 9.3 架構限制

核心業務邏輯不得直接依賴：

- Activity。
- Compose UI。
- Android Service。
- 特定 LLM SDK。
- 特定金融機構 App。

未來必須保留將 Runtime 移至 Server 的可能性。

## 9.4 資料與時間

- 時間統一以 UTC 儲存。
- 使用者介面依使用者時區顯示。
- 重要資料必須保存來源及取得時間。
- 任務、事件、記憶及決策使用唯一識別碼。
- 外部操作採用冪等設計。

---

# 第十章：開發階段

本專案採取循序漸進方式。

## P0：建立最小 Agent

目標：讓 Agent 能夠啟動、接收輸入、思考、回應及保存基本狀態。

範圍：

- Kotlin 專案。
- Runtime。
- LLM Interface。
- Event Manager。
- 基本 Task Manager。
- 基本 State Manager。
- SQLite。
- Structured Logging。

驗收：

- 可以正常啟動及停止。
- 可以與 LLM 溝通。
- 可以接收事件。
- 可以建立及完成任務。
- 可以保存基本狀態。
- 重啟後可以恢復必要資料。

## P1：持續運作能力

目標：讓 Agent 不再只是一次性對話程式。

範圍：

- Persistent Task。
- Scheduler。
- Recovery Manager。
- Working Memory。
- Episodic Memory。
- Event Queue。
- Error Handling。

驗收：

- App 關閉後任務不遺失。
- 裝置重新啟動後能恢復。
- 不會任意重複執行外部操作。
- LLM 暫時不可用時 Runtime 仍可運作。

## P2：自主判斷能力

目標：讓 Agent 能夠決定是否需要採取行動。

範圍：

- Context Builder。
- Decision Engine。
- Planning Engine。
- Semantic Memory。
- 基本 Reflection。

驗收：

- 能判斷 WAIT、INVESTIGATE、EXECUTE 等行為。
- 能將目標拆解為任務。
- 能記錄判斷依據。
- 能檢討執行結果。

## P3：外部操作能力

目標：讓 Agent 能夠透過工具與外部環境互動。

範圍：

- Tool Registry。
- Permission Manager。
- Action Validation。
- Internet。
- File System。
- Device Control。
- Audit Log。

驗收：

- 能使用已註冊工具。
- 未授權操作會被阻止。
- 操作結果能被確認。
- 重要操作可以追溯。

## P4：主動探索與持續改善

目標：讓 Agent 能主動發現問題並改善能力。

範圍：

- Curiosity Engine。
- Reflection Enhancement。
- Evolution Version。
- Evaluation Framework。
- Configuration Snapshot。
- Rollback。

驗收：

- 能提出有價值的研究問題。
- 能辨識重複錯誤。
- 能提出改善方案。
- 能比較不同版本的表現。
- 能回復至先前可靠版本。

## P5：Domain Agent

目標：讓通用 Agent 具備特定領域的專業能力。

第一個應用預定為自主投資 Agent。

投資模組必須作為 Domain Capability 加入，不得直接改寫核心 Runtime。

---

# 第十一章：驗收哲學

本專案必須區分三種成果：

1. **程式架構成功**：系統能可靠運作並完成預定功能。
2. **Agent 能力成功**：Agent 能有效研究、判斷、執行及反思。
3. **應用目標成功**：特定領域的實際成果達到預定標準。

三者不能混為一談。

某次任務失敗，不代表整個 Agent 沒有能力；某次任務成功，也不代表系統已經可靠。

每次驗收必須保存：

- 當時的 Agent Configuration Snapshot。
- 輸入資料。
- 決策紀錄。
- 工具執行紀錄。
- 預期結果。
- 實際結果。
- 失敗原因。
- 改善建議。

---

# 第十二章：尚待決定事項

以下事項不在初始版本中武斷定案：

1. Android-first、Server-first 或 Hybrid 的最終部署方式。
2. LLM Provider 及模型選擇。
3. 本地模型與雲端模型的分工。
4. 長期記憶的索引與檢索技術。
5. 多 Agent 協作是否必要。
6. 是否支援多裝置。
7. 未來是否支援其他作業系統。
8. Domain Agent 的擴充與隔離方式。

這些決策應在相關階段有實際需求時再確認。

---

# 結語

我們不打算一開始就建造一個無所不能的 Agent。

我們先讓它能夠啟動、接收資訊、保存狀態、完成任務。

接著讓它能夠持續運作、記住經驗、判斷情況、使用工具。

最後才讓它主動探索、反思、改善，並逐步承擔更複雜的工作。

**先學會走路，再學會跑。**

而我們要建立的，是一個能夠持續成長、但始終受到明確規則約束的個人 AI Agent。
