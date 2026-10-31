# AI Agent 開源框架評估與整合設計

**文件代號：** AA-02  
**版本：** 1.0  
**文件狀態：** 初版設計  
**所屬專案：** Personal AI Agent  
**前置文件：** AI Agent 總體架構規格書

---

## 一、文件目的

本文件定義 Personal AI Agent 的開源技術採用策略、框架評估標準、整合方式及自主開發範圍。

本專案不以從零開發所有 Agent 能力為目標。

應優先利用成熟的 Open Source 專案，避免重複開發已經存在且經過驗證的功能。

但也不能因為某個開源框架功能完整，就讓整個 Agent 的核心架構依賴該框架。

基本原則：

1. 成熟功能優先採用。
2. 專業工具優先整合，不重複開發。
3. 核心架構由本專案掌握。
4. 外部框架必須能夠替換。
5. 不因技術選型而改變第一份總體架構規格書的設計目標。
6. 所有採用決策都必須經過實際測試。

## 二、開源技術採用策略

### 2.1 四種採用方式

| 方式 | 定義 | 適用情況 |
|---|---|---|
| Direct Use | 直接使用原始專案 | 功能成熟且符合需求 |
| Wrapper | 透過 Adapter 封裝 | 需要統一介面或隔離相依性 |
| Fork | Fork 後修改原始碼 | 必須修改內部行為且無法透過擴充完成 |
| Independent Implementation | 自行實作 | 屬於核心能力或現有方案不符合需求 |

優先順序：

Direct Use → Wrapper → Fork → Independent Implementation

這是預設評估順序，不代表任何情況都必須遵循。

### 2.2 不重複造輪子

如果成熟工具已經能夠完成某項工作，不應只因為我們正在開發自己的 Agent，就重新實作一次。

例如：

- 已有成熟 Coding Agent，不必重新寫程式碼編輯引擎。
- 已有成熟 Git 整合工具，不必重新寫 Git 操作系統。
- 已有成熟 LLM SDK，不必重新實作 HTTP API Client。
- 已有成熟工作流程引擎，不必重新實作所有 Graph 執行功能。

但外部工具的使用方式、執行權限、工作狀態及結果驗證，仍由本專案管理。

## 三、候選開源專案

以下為第一批研究對象，不代表已經決定採用。

### 3.1 LangGraph

專案：https://github.com/langchain-ai/langgraph

定位：Agent Workflow Engine。

研究重點：

- Graph-based Workflow。
- State Management。
- Checkpoint。
- Interrupt / Resume。
- Human-in-the-loop。
- Multi-agent Workflow。
- 長時間執行及錯誤恢復。
- 與外部 Worker 整合。

評估問題：

1. 是否適合管理跨多步驟的複雜工作？
2. Checkpoint 能否滿足工作中斷及恢復需求？
3. 是否能與本專案自有 Task Manager 分離？
4. 是否必須讓整個 Runtime 依賴 LangGraph？
5. 是否能以獨立服務或 Worker 方式整合？

初步定位：

列為 Workflow Engine 候選，不預設成為 Agent Runtime。

### 3.2 CrewAI

專案：https://github.com/crewAIInc/crewAI

定位：Multi-agent Collaboration Framework。

研究重點：

- Agent Role。
- Task Assignment。
- Crew Collaboration。
- Flow Control。
- 狀態保存及恢復。
- 工具整合。
- 多 Agent 的成本與執行效率。

評估問題：

1. 多 Agent 協作是否真的能改善工作品質？
2. 是否適合長期存在的個人 Agent？
3. 能否獨立管理各 Agent 的記憶與權限？
4. 是否會造成不必要的多 Agent 呼叫成本？
5. 是否能被本專案的 Runtime 統一管理？

初步定位：

列為多 Agent 協作候選，不作為第一階段必要依賴。

### 3.3 Aider

專案：https://github.com/Aider-AI/aider

定位：Coding Worker。

研究重點：

- Repository Understanding。
- Code Editing。
- 多檔案修改。
- Git Integration。
- LLM Provider Integration。
- Lint / Test。
- CLI 自動化。
- 非互動式執行。
- 工作目錄及執行隔離。

評估問題：

1. 能否由 Runtime 自動啟動及管理？
2. 能否指定工作目錄及 Git Branch？
3. 是否能可靠取得執行結果？
4. 是否能控制檔案修改範圍？
5. 能否在執行中斷後安全恢復？
6. 能否將程式碼修改與驗收流程分離？
7. 是否支援本專案所需的 Gemini 等 LLM Provider？

初步定位：

優先研究的 Coding Worker。

原則上不重新開發 Coding Agent 的程式碼編輯能力。

### 3.4 OpenHands

專案：https://github.com/All-Hands-AI/OpenHands

定位：Software Development Agent Platform。

研究重點：

- Coding Agent 執行環境。
- Workspace。
- Tool Execution。
- Agent Server。
- 沙盒與執行隔離。
- 任務管理及結果回報。

評估問題：

1. 與 Aider 相比，哪一種更適合本專案？
2. 是否能提供 Aider 不具備的執行環境能力？
3. 是否適合成為另一種 Coding Worker？
4. 執行環境的資源需求及部署成本如何？

初步定位：

作為 Aider 的替代方案及 Coding Agent 架構參考。

### 3.5 Letta

專案：https://github.com/letta-ai/letta

定位：Stateful Agent / Memory Architecture。

研究重點：

- Persistent Memory。
- Agent State。
- Working Memory。
- Long-term Memory。
- Memory Management。
- Agent State Persistence。
- 背景記憶整理。

評估問題：

1. 是否能管理跨專案的長期記憶？
2. 記憶是否能追溯來源？
3. 是否能區分事實、推論及使用者偏好？
4. 能否修正錯誤記憶？
5. 是否能獨立於 LLM Provider？
6. 能否與本專案 Memory Manager 整合？

初步定位：

優先研究其記憶架構，不預設整套採用。

### 3.6 Google ADK

專案：https://github.com/google/adk-python

另行研究 Kotlin 版本及 Android 整合能力。

定位：Agent Development Framework。

研究重點：

- Agent Definition。
- Tool Integration。
- Workflow。
- Session。
- State。
- Multi-agent。
- Kotlin 生態支援。

初步定位：

作為 Kotlin 技術路線的候選方案及架構參考。

## 四、技術分類與責任歸屬

本專案將技術分成三個層次。

### 4.1 自有核心

以下能力由本專案負責設計及管理：

| 模組 | 責任 |
|---|---|
| Agent Runtime | Agent 生命週期 |
| Project Manager | 跨專案管理 |
| Task Manager | 任務建立、狀態及依賴 |
| Event Manager | 系統事件管理 |
| Scheduler | 工作排程 |
| State Manager | 系統狀態 |
| Memory Manager | 記憶整合及管理 |
| Permission Manager | 權限及授權 |
| Policy Engine | 行為規則及安全限制 |
| Recovery Manager | 系統恢復 |
| Evolution Manager | Agent 版本及演進 |
| Audit Manager | 操作及決策追蹤 |

這些模組不得直接依賴某個特定 Agent 框架的內部資料結構。

### 4.2 外部框架與工具

| 功能 | 候選專案 |
|---|---|
| Workflow | LangGraph |
| Multi-agent | CrewAI |
| Coding | Aider、OpenHands |
| Memory | Letta |
| LLM | Gemini、OpenAI、Claude 等 |
| Git | Git |
| Video Generation | 依實際需求評估 |
| Research Tools | 依實際需求評估 |

這些工具透過統一介面整合。

### 4.3 Domain Agent

Domain Agent 負責理解特定領域的目標、知識及工作方式。

第一階段規劃：

| Domain Agent | 工作範圍 | 執行工具 |
|---|---|---|
| Coding Agent | 程式開發、測試、除錯 | Aider、OpenHands |
| Writing Agent | 研究、規劃、寫作、校訂 | LLM、Research Tools |
| Investment Agent | 市場研究、分析、投資決策 | Market Data、Broker Adapter |
| Video Agent | 劇本、分鏡、素材、剪輯 | Image / Video / Audio Tools |

Domain Agent 不等於單一外部工具。

例如 Coding Agent 可以使用 Aider，也可以在未來改用 OpenHands，而不需要改變 Personal Agent 的任務管理方式。

## 五、Domain Agent 與 Worker 的關係

### 5.1 三層結構

**第一層：Personal Agent**

負責：

- 接收使用者目標。
- 管理所有 Project。
- 決定工作優先順序。
- 管理任務及記憶。
- 控制權限。
- 管理 Agent 整體生命週期。

**第二層：Domain Agent**

負責：

- 理解領域需求。
- 建立領域工作計畫。
- 拆解領域任務。
- 選擇適合的 Worker。
- 驗收 Worker 結果。
- 維護領域知識及工作狀態。

**第三層：Worker**

負責：

- 執行具體工作。
- 操作工具。
- 回傳執行結果。
- 回報錯誤及執行狀態。

### 5.2 Coding Agent 範例

使用者提出：

「開發一個 Go 語言的 MQTT Gateway。」

工作流程：

1. Personal Agent 接收需求。
2. 建立 Coding Project。
3. Coding Agent 分析需求。
4. 建立開發 Task。
5. 選擇 Aider 作為 Coding Worker。
6. Aider 在指定 Workspace 執行工作。
7. Coding Agent 檢查修改內容。
8. 執行測試及驗收。
9. 回報 Personal Agent。
10. Personal Agent 更新 Project、Task 及 Memory。

Aider 不負責整個 Project 的生命週期。

### 5.3 Writing Agent 範例

使用者提出：

「繼續撰寫歷史小說下一卷。」

工作流程：

1. Personal Agent 找出對應 Writing Project。
2. 載入作品設定及目前進度。
3. Writing Agent 分析未完成工作。
4. 檢查章節規劃及史實待查事項。
5. 委派研究、撰寫或校訂工作。
6. 驗證產出是否符合作品規範。
7. 更新章節狀態及相關記憶。

Writing Agent 不應只依賴對話摘要維持作品連續性。

### 5.4 Investment Agent 範例

Investment Agent 必須遵守獨立的金融操作及風險政策。

LLM 可以提出投資分析及交易建議，但不得直接決定是否可以突破風險限制。

交易必須經過獨立 Risk Engine、Policy Engine 及授權檢查。

金融操作不能因為採用了某個 Agent Framework，就將安全責任交給該框架。

### 5.5 Video Production Agent 範例

Video Agent 將劇本轉換為可執行的製作流程：

1. 劇本解析。
2. 場景及角色整理。
3. 分鏡規劃。
4. 素材生成。
5. 配音及音效。
6. 剪輯及組裝。
7. 品質檢查。
8. 輸出成品。

不同步驟可以使用不同的外部工具。

## 六、統一 Worker Interface

本專案必須建立自己的 Worker 介面。

外部工具不能直接成為 Runtime 的內部模組。

### 6.1 Worker 基本介面

每個 Worker 至少需要支援：

- Initialize
- Execute
- GetStatus
- GetResult
- Cancel
- HealthCheck

概念介面：

```text
Worker
 ├── Initialize(config)
 ├── Execute(task)
 ├── GetStatus(execution_id)
 ├── GetResult(execution_id)
 ├── Cancel(execution_id)
 └── HealthCheck()
```

實際介面依實作語言及通訊方式決定。

### 6.2 Worker Request

每次委派工作至少包含：

| 欄位 | 說明 |
|---|---|
| request_id | 唯一請求識別碼 |
| project_id | 所屬專案 |
| task_id | 所屬任務 |
| worker_id | Worker 識別碼 |
| workspace | 工作環境 |
| input | 工作內容 |
| constraints | 執行限制 |
| permissions | 授權範圍 |
| deadline | 工作期限 |
| callback | 結果回報方式 |

### 6.3 Worker Response

每次執行至少回報：

| 欄位 | 說明 |
|---|---|
| request_id | 原始請求識別碼 |
| execution_id | 執行識別碼 |
| status | 執行狀態 |
| result | 執行結果 |
| artifacts | 產出檔案 |
| errors | 錯誤資訊 |
| started_at | 開始時間 |
| completed_at | 完成時間 |
| usage | 模型及工具資源使用量 |

### 6.4 Worker 狀態

```text
CREATED
   ↓
INITIALIZING
   ↓
READY
   ↓
RUNNING
   ├── COMPLETED
   ├── FAILED
   ├── WAITING
   └── CANCELLED
```

外部 Worker 的執行狀態必須映射至本專案的統一狀態。

不得直接將外部工具的狀態名稱當成本專案的內部狀態。

## 七、Runtime 與 Workflow Engine 的分工

本專案不預設使用 LangGraph 管理所有任務。

必須區分：

### 7.1 Runtime 管理

- Agent 生命週期。
- Project。
- Task。
- Event。
- Scheduler。
- 全域狀態。
- 權限。
- 記憶。
- Recovery。
- Evolution。

### 7.2 Workflow Engine 管理

- 多步驟工作流程。
- 節點執行。
- 條件分支。
- 流程狀態。
- 工作流程 Checkpoint。
- 人工介入。
- 流程恢復。

### 7.3 採用原則

簡單工作由 Runtime 直接管理。

複雜工作可以委派 Workflow Engine。

Workflow Engine 必須透過 Adapter 與 Runtime 整合。

LangGraph 不得成為 Project Manager、Task Manager 或 Memory Manager 的唯一資料來源。

## 八、技術整合方式

### 8.1 通訊方式

依工具特性選擇：

| 方式 | 適用情況 |
|---|---|
| CLI | Aider 等命令列工具 |
| HTTP API | 獨立服務 |
| SDK | 可直接整合的程式庫 |
| IPC | 同一裝置上的獨立程序 |
| Message Queue | 非同步及長時間工作 |

第一階段以 CLI、HTTP API 為主。

### 8.2 語言及技術隔離

本專案核心語言尚待最終確認。

若核心採 Kotlin，而外部工具使用 Python，允許以獨立 Worker 整合。

不得為了使用某個 Python 框架，就強迫整個 Runtime 改成 Python。

各元件應以穩定的介面及資料格式溝通。

### 8.3 LLM Provider Abstraction

建立統一 LLM Provider 介面。

至少支援：

- Model Selection。
- Generate。
- Streaming。
- Tool Calling。
- Token Usage。
- Timeout。
- Retry。
- Error Handling。

各 Domain Agent 不得直接綁定單一 LLM SDK。

API Key、Token 等憑證由安全設定管理，不得交給不需要該憑證的 Worker。

## 九、開源授權與供應鏈審查

任何專案正式採用前，必須檢查：

1. 主專案 License。
2. 重要相依套件 License。
3. 商業使用限制。
4. 修改及散布要求。
5. 原始碼公開義務。
6. 商標及品牌使用規定。
7. 第三方模型及 API 的使用條款。
8. 安全漏洞及維護狀況。
9. 社群活躍程度。
10. 版本相容性。

不得僅因 GitHub 標示 Public Repository，就認定可以自由商用。

每個採用專案都應保存：

- Repository URL。
- 採用版本或 Commit。
- License。
- 相依套件清單。
- 修改紀錄。
- 安全審查結果。
- 升級策略。

## 十、評估與驗收標準

每個候選專案以 1～5 分評估。

| 評估項目 | 權重 |
|---|---:|
| 功能符合度 | 20% |
| 穩定性及恢復能力 | 20% |
| 擴充及整合能力 | 15% |
| 架構相容性 | 15% |
| 維護及社群狀況 | 10% |
| 技術及部署成本 | 10% |
| 授權及商用適用性 | 10% |

授權不適用、存在無法接受的安全風險，或無法滿足必要功能者，不論總分多少都不得採用。

### 10.1 實際測試項目

每個候選專案必須進行：

- 正常執行測試。
- 長時間執行測試。
- 中途停止測試。
- 程序重新啟動測試。
- 網路中斷測試。
- LLM Provider 異常測試。
- 重複請求測試。
- 執行結果追蹤測試。
- 權限隔離測試。
- 資源使用量測試。

### 10.2 最終決策

每個候選專案只能有以下其中一種決策：

- ADOPT：直接採用。
- WRAP：封裝後採用。
- FORK：Fork 修改。
- REFERENCE：只作為架構參考。
- REJECT：不採用。
- DEFER：暫緩決定。

所有決策都必須留下原因及版本紀錄。

## 十一、第一階段實作策略

第一階段不追求一次整合所有開源專案。

建議依序進行：

### P0：開源專案盤點

- 確認 LangGraph。
- 確認 CrewAI。
- 確認 Aider。
- 確認 OpenHands。
- 確認 Letta。
- 確認 LLM Provider。
- 完成 License 初步審查。

### P1：Coding Worker PoC

以 Aider 作為第一個候選 Worker。

測試：

1. Runtime 能否建立 Coding Task。
2. 能否指定 Aider 工作目錄。
3. 能否啟動 Aider。
4. 能否取得執行狀態。
5. 能否取得修改結果。
6. 能否執行測試。
7. 能否中斷及恢復。
8. 能否保留完整稽核紀錄。

### P2：Runtime 整合

建立：

- Worker Registry。
- Worker Adapter。
- Execution Manager。
- Result Validator。
- Workspace Manager。

### P3：Workflow Engine 評估

選擇一個具有多步驟、分支、等待及恢復的實際工作流程。

比較：

- 自有簡單流程引擎。
- LangGraph。

以實際測試結果決定是否採用。

### P4：第二個 Domain Agent

建立 Writing Agent。

驗證同一個 Runtime 能否管理 Coding Project 和 Writing Project，並在兩者之間切換。

### P5：其他 Domain Agent

依序評估 Video Agent、Investment Agent。

投資 Agent 必須另外完成金融操作安全及風險控制設計，不得直接沿用一般 Worker 的授權模式。

## 十二、架構不可違反的原則

1. Personal Agent Runtime 是本專案的核心。
2. Domain Agent 不得直接繞過 Runtime 操作系統資源。
3. Worker 不得自行擴大授權。
4. 外部框架不得成為唯一的全域狀態來源。
5. LLM 不得自行修改安全政策。
6. 重要任務必須持久化。
7. 外部操作必須可追蹤。
8. Worker 必須能替換。
9. LLM Provider 必須能替換。
10. 系統必須能在外部框架故障時安全停止或降級。

## 十三、尚待決策事項

以下事項不在本文件中預先定案：

- Runtime 最終採 Kotlin、Go 或其他語言。
- 是否採用 LangGraph。
- 是否需要 CrewAI。
- Aider 與 OpenHands 的最終選擇。
- 是否採用 Letta。
- Worker 使用 CLI 或服務化部署。
- Android 與 Server 的部署分工。
- 多 Agent 是否在第一階段實作。
- 記憶資料庫及向量檢索技術。
- 各 LLM Provider 的優先順序。

這些決策應在 PoC 及技術審查後確定。

---

## 文件結論

本專案不以重新開發一套完整的 Coding Agent、Workflow Engine 或 LLM Framework 為目標。

我們要建立的是一個能夠管理不同領域 Agent、整合成熟開源工具、持續運作並且逐步演進的 Personal AI Agent。

**成熟的功能，盡可能採用。**

**必須掌握的核心，自行設計。**

**可以替換的元件，不要形成不可替換的依賴。**

我們不需要重新發明輪子，但必須知道整台車要往哪裡走。
