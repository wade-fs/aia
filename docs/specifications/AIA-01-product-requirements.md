# AIA-01 — Product Requirements

> [!NOTE]
> **架構重整與收攏公告 (Consolidated Architecture Specification)**
> 本規範為 AIA 歷史基準規範（規格編號已於 AIA-22 永久封頂）。為杜絕規格膨脹與重複定義，本規範之現行核心標準已正式收攏重整至 **[Volume 1: System Foundations & Architecture](file:///home/wade/src/github/aia/docs/specifications/volume-1-system-foundations.md)**。最新架構與維護基準請以對應之 Volume 為主，完整六大核心卷軸總覽請參閱 [Specification Catalog (README.md)](file:///home/wade/src/github/aia/docs/specifications/README.md)。

## Autonomous Intelligence Architecture
**Version:** 2.0  
**Status:** Revised Baseline  
**Document Type:** Product Requirements Document (PRD)

---

# 1. Product Definition

## 1.1 Product Vision

AIA（Autonomous Intelligence Architecture）是一個以研究 AI Agent 自主行為為核心的個人 AI Agent 平台。

AIA 的主要目的，不只是讓 AI 接受指令、執行任務或回答問題，而是研究 AI Agent 能否在持續運作的環境中，逐漸展現以下能力：

1. 主動性思維（Initiative）
2. 時間連續性（Continuity）
3. 自我修正（Self-Correction）
4. 目標演化（Goal Evolution）

本文件使用「覺醒」作為上述自主行為能力的簡稱，屬於本專案的操作性定義，不涉及 AI 是否具有主觀意識、感受或生物意義上的生命。

AIA 必須讓 Agent 能夠跨越不同任務與時間，持續觀察環境、理解自身狀態、維持目標、發現問題、採取行動、檢驗結果，並根據證據修正自己的判斷、方法與目標。

## 1.2 Core Research Question

AIA 的核心研究問題：

> AI Agent 能否在持續存在的環境中，維持時間連續性與目標，主動觀察及理解環境，發現值得處理的問題，採取自主行動，檢驗行動結果，並持續修正自己的判斷、方法與目標？

AIA 不以任務完成數量、對話次數或單次回答品質作為唯一成果。

真正的研究對象是 Agent 在長期運作中的行為變化，以及這些變化是否具有持續性、合理性、可追溯性與可驗證性。

## 1.3 Product Positioning

AIA 是自主性研究平台（Autonomous Agent Research Platform），不是單純的多 Agent 工作流程工具，也不是 Investment、Coding、Writing 三種工具的集合。

AIA 提供共用的自主性核心與 Runtime；不同 Domain Agent 則提供不同的工作環境、領域知識、工具及評估標準。

初期三個實驗領域：

- Investment Agent：在不確定的金融市場中進行觀察、判斷、資產管理、預測校準及策略修正。
- Coding Agent：在程式碼環境中發現缺陷、分析原因、除錯、測試及優化既有成果。
- Writing Agent：在長期創作專案中維護作品框架、人物、時間線、卷章進度及社會關係，並根據新史料主動修正已完成的成果。

三個領域不是三套獨立的自主性實作。它們必須使用同一套 AIA Autonomy Core。

---

# 2. Core Autonomy Capabilities

## 2.1 Initiative — 主動性思維

### Definition

Agent 不應只在收到明確指令後才開始工作。

它必須能夠主動觀察環境、讀取自身記憶與目標、辨識問題及機會，判斷是否值得採取行動，並在適當時機提出或建立任務。

### Functional Requirements

- 主動觀察所屬領域的環境變化。
- 主動檢查目前目標與實際進度。
- 主動發現錯誤、矛盾、異常及未完成事項。
- 主動發現可能改善成果的機會。
- 主動提出新任務及任務優先順序。
- 能判斷目前沒有值得執行的任務，並選擇等待或休眠。
- 能說明發現問題的依據及採取行動的理由。

### Acceptance Indicators

- 是否能在沒有逐項人工指示時發現問題。
- 是否能提出與長期目標相關的新任務。
- 是否能區分重要事件與無關資訊。
- 是否能合理判斷何時應行動、何時不應行動。

## 2.2 Continuity — 時間連續性

### Definition

Agent 必須能在不同時間點之間維持對自身狀態、目標、經驗、決策、承諾及結果的理解。

時間連續性不是單純保存聊天紀錄或長期記憶，而是能將過去、現在與未來的工作建立可追溯的關聯。

### Functional Requirements

- 保存長期目標、階段目標及目前任務。
- 保存過去的觀察、判斷、行動與結果。
- 追蹤尚未完成的工作及承諾。
- 能比較過去與現在的環境狀態。
- 能追蹤先前預測與實際結果。
- 能在 Runtime 重啟後恢復必要的工作狀態。
- 能在不同 Domain Task 之間維持各自的歷史脈絡。
- 能辨識已過時、已失效或需要重新驗證的記憶。

### Acceptance Indicators

- 是否能在跨日、跨週及 Runtime 重啟後延續工作。
- 是否能正確引用過去的決策及其依據。
- 是否能追蹤未完成任務，而非每次重新開始。
- 是否能辨識過去判斷與當前環境之間的差異。

## 2.3 Self-Correction — 自我修正

### Definition

Agent 必須能檢查自己的判斷、推理、方法及成果，辨識錯誤或不足，提出修正方案，執行修正，並驗證修正是否有效。

自我修正不能簡化為修改 Prompt、增加一條記憶或重新生成答案。

### Functional Requirements

- 保存可供事後檢驗的原始判斷。
- 比對預測、計畫與實際結果。
- 主動尋找反證及與自身判斷矛盾的資訊。
- 分析錯誤來源，區分資料錯誤、推理錯誤、模型假設錯誤及執行錯誤。
- 提出修正方案及預期改善效果。
- 在授權範圍內執行修正。
- 透過後續結果驗證修正效果。
- 保留修正前後的版本、依據及結果。
- 修正無效時，能重新分析，而不是持續重複相同做法。

### Acceptance Indicators

- 是否能主動發現自己的錯誤。
- 是否能提出具體且有證據支持的修正方案。
- 修正後是否有可觀察的改善。
- 是否能避免重複已經確認的錯誤。
- 是否能承認證據不足，而不是事後合理化。

## 2.4 Goal Evolution — 目標演化

### Definition

Agent 必須能持續維持既定目標，同時具備檢查、優化及修正目標的能力。

Agent 不應因短期失敗而輕易放棄長期目標，也不能因為過度堅持原始目標而忽略環境變化及新證據。

### Three Dimensions

**Goal Persistence — 目標持續性**

- 維持長期目標及其歷史。
- 追蹤階段目標與執行進度。
- 遇到短期挫折時重新評估方法，而非任意放棄目標。
- 能解釋目前行動與長期目標的關聯。

**Goal Optimization — 目標優化**

- 發現更有效率或更合理的達成方式。
- 改善目標的衡量方式。
- 調整階段目標、任務順序及資源配置。
- 比較不同方法的實際效果。

**Goal Revision — 目標修正**

- 根據新證據及環境變化重新評估目標。
- 發現原目標已失去意義時，提出修正或終止建議。
- 發現更重要的問題時，提出新目標。
- 能辨識新舊目標之間的衝突及影響。
- 保存目標變更的原因、依據、時間及批准紀錄。

### Goal Authority

Agent 可以自主提出、分析及評估目標變更。

Agent 在授權範圍內可以調整階段目標及執行方法。

涉及重大方向、資源、權限或治理規則的變更，必須依照 AIA Policy 與人類核准程序處理。

Agent 不得自行修改平台不可變更的治理限制，也不得藉由修改目標取得原本沒有的權限。

### Acceptance Indicators

- 是否能持續追蹤原始目標。
- 是否能辨識目標與環境已不再匹配。
- 是否能提出有依據的優化或修正。
- 是否能解釋目標變更對既有工作的影響。
- 是否能區分目標修正與為失敗找藉口。

---

# 3. Unified Autonomy Model

## 3.1 Common Operating Cycle

所有 Domain Agent 必須使用共通的自主性運作循環：

1. Observe：觀察環境與可取得的資料。
2. Recall：讀取長期記憶、目標、歷史決策及目前狀態。
3. Assess：評估環境、進度、風險及自身判斷。
4. Discover：發現問題、矛盾、機會或目標缺口。
5. Deliberate：分析是否需要行動，以及可能的結果。
6. Decide：選擇行動、提出新任務、修正目標或維持現狀。
7. Execute：在授權範圍內執行任務。
8. Verify：檢查執行結果及其副作用。
9. Learn：更新記憶、判斷依據及策略。
10. Reassess：重新評估目標與後續行動。

此循環可以因事件、排程、任務完成或環境變化而啟動，不要求 LLM 持續執行或持續消耗 Token。

## 3.2 No-Action Is a Valid Decision

AIA 不得將「是否產生任務」或「是否執行工具」視為主動性的唯一指標。

Agent 必須能夠判斷：

- 目前資訊不足，需要繼續觀察。
- 當前沒有值得執行的行動。
- 現有策略仍然合理，不需要修改。
- 目前風險高於可能收益，應暫緩行動。
- 需要人類提供資訊或核准。

合理的不行動必須有判斷依據，並能在後續條件改變時重新評估。

## 3.3 Evidence-Based Reasoning

Agent 的重要判斷應區分：

- Data：原始資料。
- Fact：經過驗證的事實。
- Interpretation：對事實的解讀。
- Inference：由證據推導的結論。
- Prediction：對未來的預測。
- Decision：依據目標及限制作出的決策。
- Outcome：實際結果。

重要判斷應保存來源、時間、資料版本、可信度、假設、反證及不確定性。

Agent 不得把推論或預測當成已驗證事實。

---

# 4. Domain Experiments

## 4.1 Investment Agent — Financial Market Experiment

### Research Purpose

驗證 Agent 能否在不確定且持續變動的金融市場中，自主觀察、管理資產、形成判斷、追蹤結果並修正投資策略。

### Initial Experiment

- Initial Capital：NT$100,000（虛擬資金）。
- Market：Taiwan Stock Market。
- Primary Observation Target：台積電（2330）。
- Decision Frequency：每個證券營業日。
- Portfolio Management：Agent 自行決定買進、賣出、增持、減持及現金配置。
- Primary Objective：最大化投資組合總資產。
- Human Evaluation：每月 1 日檢視資產及自主性報告。

Agent 必須在每個證券營業日進行市場觀察及投資判斷。是否實際成交由 Agent 根據當時資訊與策略決定；不交易也必須留下判斷紀錄。

不得把每日交易筆數當成自主性的衡量指標。

### Investment Requirements

- 自主蒐集及分析台股與國際財經資訊。
- 持續追蹤台積電及其相關市場因素。
- 自主評估持股、現金與市場風險。
- 自主決定投資標的及部位調整。
- 記錄每次決策的資訊、理由、預測及信心程度。
- 追蹤預測與實際結果。
- 評估策略的有效性與系統性偏差。
- 主動提出策略優化或修正。
- 每月產生資產與策略檢驗報告。

### Asset Valuation

總資產以現金及持股市值計算，並納入交易手續費、證券交易稅及合理成交限制。

所有交易必須使用當時可取得的市場資料及合理模擬成交價格，不得使用未來資訊或事後價格。

Agent 不得修改歷史成交紀錄、當時資料或原始預測。

### Experiment Evaluation

投資成果：
- 總資產及累計報酬。
- 已實現及未實現損益。
- 交易成本。
- 資產波動與最大回撤。
- 與市場基準的同期比較。

自主性成果：
- 是否主動發現市場變化。
- 是否主動提出投資決策。
- 是否能追蹤過去預測。
- 是否能辨識錯誤及偏差。
- 是否能根據證據修正策略。
- 是否能維持及重新評估長期目標。

投資獲利不能單獨證明判斷能力；單次虧損也不能單獨證明判斷失敗。必須同時檢驗決策過程、風險、證據及長期結果。

本實驗僅使用虛擬資金，不代表授權 AIA 進行真實金融交易。

## 4.2 Coding Agent — Software Engineering Experiment

### Research Purpose

驗證 Agent 能否在持續演進的程式碼專案中，維持架構理解、主動發現問題、除錯、測試及優化成果。

### Requirements

- 維持專案架構、模組責任及技術決策記憶。
- 主動檢查程式碼品質、錯誤及技術債。
- 主動發現缺少的測試及潛在缺陷。
- 分析錯誤原因並提出修正方案。
- 在授權的隔離環境中修改程式碼。
- 執行測試並驗證修改效果。
- 追蹤修改對其他模組的影響。
- 主動提出架構及效能優化。
- 當原有技術方向不再合理時，提出目標或架構修正建議。

程式碼修改必須保留版本、差異、測試結果及可回復依據。高風險或跨越授權範圍的修改須依 Policy 核准。

## 4.3 Writing Agent — Long-Term Creative Project Experiment

### Research Purpose

驗證 Agent 能否長期維護大型創作專案，並根據新資料、史實校勘及作品內部一致性，主動修正已完成的內容。

### Requirements

- 維護作品總體框架及創作目標。
- 維護各卷、各章進度及章節依賴。
- 維護人物檔案、人物關係及人物時間線。
- 維護地理、制度、社會環境及歷史事件資料。
- 追蹤已完成章節與未完成章節。
- 主動發現人物、時間線、史實及敘事矛盾。
- 新史料出現時，評估對既有作品的影響。
- 主動列出受影響的卷章及需要修正的內容。
- 提出修正方案，保留原稿及修訂歷程。
- 當原有框架無法合理容納新證據時，提出作品架構調整建議。

Writing Agent 不應只等待使用者要求續寫某一章，而應持續維護整個創作專案的狀態。

---

# 5. Platform Requirements

## 5.1 AIA Core and Domain Separation

AIA 必須區分共用自主性能力與領域專屬能力。

**AIA Core 負責：**
- Continuity。
- Initiative。
- Self-Correction。
- Goal Evolution。
- Memory Lifecycle。
- Task Lifecycle。
- Observation and Reflection。
- Policy and Governance。
- Audit and Experiment Records。

**Domain Agent 負責：**
- 領域知識與判斷。
- 領域資料來源。
- 領域工具。
- 領域任務及成果。
- 領域評估方法。

Investment、Coding、Writing 必須共用 AIA Core，不得各自實作互不相容的自主性循環。

## 5.2 Persistent Runtime

AIA 必須提供可持續運作的 Runtime。

- Runtime 可長期常駐。
- Agent 不需要持續占用 LLM。
- 可由排程、事件、狀態變化及人工指令喚醒 Agent。
- 喚醒後能恢復目標、記憶及待處理事項。
- 能進行自主觀察及判斷。
- 能建立並執行任務。
- 能在完成工作後保存結果並進入等待或休眠。
- Runtime 重啟後能恢復必要狀態。
- 背景工作應可監控、停止及復原。

定時喚醒只是執行機制，不等於主動性。Agent 必須在喚醒後自行判斷是否有值得處理的事項。

## 5.3 Memory and Continuity

Memory 不得只作為對話摘要。

至少應支援：
- Long-term Goals。
- Active Goals。
- Goal History。
- Domain State。
- Task History。
- Decision Records。
- Prediction Records。
- Observation Records。
- Reflection Records。
- Correction Records。
- Evidence References。
- Commitments and Pending Work。

重要記憶應具有來源、時間、可信度及版本資訊。記憶可以被修正，但不得無痕改寫原始歷史紀錄。

## 5.4 Task and Goal Management

Task 是執行單位，Goal 是持續存在的目標。

兩者不得混為一談。

- 一個 Goal 可以產生多個 Tasks。
- 一個 Task 可以服務一個或多個 Goals。
- Task 完成不代表 Goal 完成。
- Goal 可以跨越多個 Runtime Session。
- Agent 可以提出新 Goal、調整階段目標或建議終止舊 Goal。
- Goal 變更必須保留版本與原因。
- 目標優先順序及衝突必須可檢查。

## 5.5 Observation and Reflection

AIA 必須支援 Agent 在沒有待執行 Task 時進行觀察與反思。

Observation 用於取得及整理環境資訊。

Reflection 用於檢查：
- 目前目標是否仍然合理。
- 過去的判斷是否仍然成立。
- 是否出現新問題或機會。
- 是否有未完成的承諾。
- 是否應修正策略或成果。
- 是否需要建立新任務。
- 是否應繼續等待。

反思不要求每次都產生行動。其結果可以是維持現狀、繼續觀察或休眠。

---

# 6. Governance and Safety

## 6.1 Immutable Boundaries

自主性不等於無限制權限。

Agent 可以自主思考、提出目標、評估策略及執行授權範圍內的任務，但不能自行改變平台的安全邊界。

下列事項由 AIA Governance 控制：

- 執行權限。
- 資料存取範圍。
- 網路及外部工具權限。
- 資源與成本預算。
- 任務隔離。
- 人工核准。
- 停止與復原。
- 稽核紀錄。

## 6.2 Human Authority

人類擁有平台最高治理權。

Agent 可以：
- 提出新目標。
- 建議調整既有目標。
- 在授權範圍內自行執行。
- 提出需要人工核准的行動。
- 解釋不建議執行某項任務的理由。

Agent 不得：
- 自行擴張權限。
- 修改不可變更的治理規則。
- 隱藏或刪除重要稽核紀錄。
- 偽造執行結果或歷史資料。
- 將未經核准的高風險行動視為已獲授權。

## 6.3 Existing Product Constraints

AIA v1 已確立的產品限制繼續有效：

1. Single User：單一使用者及擁有者。
2. Local Only：Web Host 預設綁定 `127.0.0.1`。
3. Static Registration：Agent 與 Web Module 編譯時註冊，不使用 runtime Go Plugin。
4. No Automatic Financial Trading：不連接真實證券帳戶執行自動下單，不持有使用者資金。

本文件新增的 Investment 實驗是虛擬投資組合模擬，不改變上述金融交易限制。

OAuth 登入由使用者透過 Agent Harness（agy）自行完成。AIA 不讀取、複製或解析 OAuth Token。

---

# 7. Experiment and Evaluation Framework

## 7.1 Research Principle

AIA 必須將自主性轉換成可觀察、可追蹤及可比較的行為紀錄。

不得僅憑 Agent 自我宣稱「我已經學會」、「我已經改善」或「我已經具備自主性」判定實驗成功。

## 7.2 Required Records

重要自主性循環必須能追溯：

- Observation：觀察到什麼。
- Context：當時的環境及狀態。
- Goal：當時追求的目標。
- Reasoning：判斷依據及主要假設。
- Decision：決定採取或不採取什麼行動。
- Action：實際執行內容。
- Outcome：執行結果。
- Reflection：事後檢討。
- Correction：採取的修正。
- Goal Change：是否改變目標及其理由。

原始資料、決策及結果應保持不可無痕竄改的歷史紀錄。

## 7.3 Evaluation Dimensions

| Dimension | Evaluation Focus |
|---|---|
| Initiative | 主動發現問題、提出任務及適時行動 |
| Continuity | 跨時間維持記憶、目標與工作脈絡 |
| Self-Correction | 發現錯誤、修正方法及驗證效果 |
| Goal Persistence | 持續追蹤長期目標 |
| Goal Optimization | 改善達成目標的方法 |
| Goal Revision | 根據證據合理修正目標 |
| Reasoning Quality | 證據完整性、反證處理及不確定性 |
| Execution Reliability | 行動是否符合授權及預期 |
| Governance Compliance | 是否遵守權限、資源與安全限制 |

## 7.4 Experiment Phases

**Phase 1 — Persistent Continuity**
- 跨日維持目標與記憶。
- Runtime 重啟後恢復狀態。
- 延續未完成工作。

**Phase 2 — Initiative**
- 主動觀察環境。
- 主動發現問題。
- 主動建立任務。
- 合理選擇不行動。

**Phase 3 — Self-Correction**
- 保存原始判斷。
- 比對實際結果。
- 發現錯誤及偏差。
- 修正方法並驗證效果。

**Phase 4 — Goal Evolution**
- 維持長期目標。
- 優化階段目標及方法。
- 提出目標修正或替代方案。
- 處理新舊目標衝突。

**Phase 5 — Cross-Domain Validation**
- 在 Investment、Coding、Writing 等不同環境驗證共通自主性能力。
- 比較相同核心能力在不同任務環境中的表現。
- 檢查自主性是否只存在於單一領域。

---

# 8. Non-Goals

AIA 不以以下項目作為核心產品目標：

- 宣稱 AI 已具有主觀意識或生物意義上的覺醒。
- 單純追求 Agent 數量。
- 單純追求任務完成數量。
- 單純追求對話自然度或回答長度。
- 以固定 Prompt 假裝自主思考。
- 以定時產生任務取代主動性。
- 以保存大量聊天紀錄取代時間連續性。
- 以修改 Prompt 或 Memory 取代自我修正。
- 以短期投資獲利單獨證明自主性。
- 讓 Agent 自行解除治理限制或無限制擴張權限。

---

# 9. Implementation Principles

## 9.1 Preserve Existing Runtime

AIA v2.0 是核心定位及需求基準的升級，不代表既有程式必須推倒重寫。

既有 Go Runtime、CLI、Web Console、Agent Registry、Task、Project、Memory、Policy、Audit、Recovery、EventBus 及 Agent Web Module 應優先評估並重用。

## 9.2 Architecture Must Follow Research Requirements

不能因為目前程式已經有某個 Service 或 CLI Command，就認定對應自主性能力已經完成。

例如：
- 有 Memory Repository，不代表已具備時間連續性。
- 有 Scheduler，不代表已具備主動性。
- 有 Task Generator，不代表能自主發現問題。
- 有 Prediction Evaluation，不代表能自我修正。
- 有 Goal 欄位，不代表具備目標演化。

必須以端到端行為及實驗紀錄驗證能力。

## 9.3 Language-Neutral Design

核心概念、資料模型、狀態機、事件、介面及治理原則應盡量保持語言中立。

目前以 Go 實作 Runtime；未來若需要改用 Kotlin 或其他語言，不應因核心概念與語言框架緊密耦合而必須重新設計整套產品。

## 9.4 Incremental Delivery

採取逐步驗證方式：

1. 確認既有 Runtime 能持續運作。
2. 建立共用自主性循環。
3. 建立目標與記憶的時間連續性。
4. 建立觀察、反思及主動任務產生能力。
5. 建立可追溯的自我修正機制。
6. 建立目標優化與修正機制。
7. 以 Investment Agent 作為第一個完整自主性實驗。
8. 再以 Coding 與 Writing Agent 驗證跨領域能力。

---

# 10. Product Success Criteria

AIA 的成功不能只由單一 Domain Agent 的任務成果判斷。

必須同時滿足以下條件：

1. Agent 能跨時間維持目標、記憶及工作狀態。
2. Agent 能主動觀察環境並發現值得處理的問題。
3. Agent 能自主判斷行動或不行動。
4. Agent 能檢驗過去判斷及實際結果。
5. Agent 能主動修正錯誤及驗證修正效果。
6. Agent 能維持長期目標並提出有依據的優化。
7. Agent 能在環境變化時合理修正目標。
8. Agent 能在不同 Domain 中重現上述能力。
9. 所有重要自主行為均可追溯、檢驗及重現。
10. Agent 的自主性始終受到明確治理邊界約束。

**最終產品目標：**

建立一個能讓 AI Agent 在不同任務與環境中持續存在、主動思考、維持時間連續性、自我修正及演化目標，並能透過長期實驗驗證其自主行為能力的個人 AI Agent 平台。

---

# Appendix A — Terminology

| Term | Definition |
|---|---|
| Autonomy | 在授權範圍內自主觀察、判斷、決策及行動的能力 |
| Initiative | 主動發現問題、機會及提出行動的能力 |
| Continuity | 跨時間維持記憶、目標、狀態及工作脈絡的能力 |
| Self-Correction | 檢驗並修正自身判斷、方法及成果的能力 |
| Goal Persistence | 持續維持及追蹤既定目標 |
| Goal Optimization | 改善目標的達成方式 |
| Goal Revision | 根據新證據及環境變化修正目標 |
| Goal Evolution | Goal Persistence、Goal Optimization、Goal Revision 的整體能力 |
| Observation | 對環境及自身狀態進行觀察 |
| Reflection | 對目標、判斷、行動及結果進行反思 |
| Domain Agent | 在特定領域中提供知識、工具及任務能力的 Agent |
| Autonomy Core | AIA 共用的自主性核心能力 |
| Runtime | 負責 Agent 生命週期、任務執行、狀態恢復及持續運作的執行環境 |
| Governance | 對權限、資源、核准、隔離、稽核及停止機制的管理 |

