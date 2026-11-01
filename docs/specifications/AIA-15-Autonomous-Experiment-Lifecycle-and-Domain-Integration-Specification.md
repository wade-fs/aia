# AIA-15：Autonomous Experiment Lifecycle & Domain Integration Specification

**Version:** 2.0  
**Document Type:** System Specification  
**Project:** AIA (Autonomous Intelligence Architecture)  
**Status:** Proposed Specification  
**Language:** Traditional Chinese

**依賴文件：**
- AIA-01 Product Requirements
- AIA-02 System Architecture
- AIA-03 Go Runtime Implementation
- AIA-04 Test Plan & Acceptance Criteria
- AIA-05 Investment Agent
- AIA-08 Autonomy Core & Operational Autonomy
- AIA-09 Initiative & Goal Evolution
- AIA-10 Continuity & Long-term Project Management
- AIA-11 Self-Correction & Learning
- AIA-12 Cross-Domain Collaboration & Agent Delegation
- AIA-13 Autonomy Evaluation & Benchmarking
- AIA-14 Self-Modification & Safe Code Evolution

---

# 1. 文件目的

## 1.1 設計目的

AIA 的核心研究目標，是觀察 AI Agent 是否能在長期運作中形成穩定的自主行為能力。

這些能力包括：

1. 主動性思維（Initiative）。
2. 時間連續性（Continuity）。
3. 自我修正（Self-Correction）。
4. 目標演化（Goal Evolution）。

前述文件已分別定義這些能力的架構、運作方式與評估方法。

但自主能力不能只靠個別模組的存在來證明。

AIA 必須透過實際的長期工作，讓這些能力在不同環境中持續運作，並觀察它們是否真的能改善工作成果。

因此，本文件定義：

- Autonomous Experiment。
- Experiment Lifecycle。
- Domain Integration。
- Cross-Domain Experiment。
- Long-term Experiment Management。
- Experiment Evaluation。
- Experiment Evolution。

## 1.2 實驗目的

AIA 的實驗不是單純執行一組測試案例。

它是一個長期循環：

1. 建立實驗目標。
2. 觀察實際環境。
3. 主動發現問題與機會。
4. 提出行動與實驗假設。
5. 執行實驗。
6. 驗證實驗結果。
7. 分析差異與失敗原因。
8. 修正策略或工作方法。
9. 更新知識與記憶。
10. 進入下一輪實驗。

每一輪實驗都應建立在過去的經驗之上，而不是每次重新開始。

## 1.3 實驗與一般 Task 的差異

| 比較項目 | 一般 Task | Autonomous Experiment |
|---|---|---|
| 起點 | 明確任務指示 | 長期研究目標 |
| 工作範圍 | 單次工作 | 多輪持續工作 |
| 執行方式 | 依指示完成 | 自主決定下一步 |
| 評估 | 任務完成結果 | 長期能力與成果 |
| 記憶 | 保存任務紀錄 | 保存實驗歷程與因果關係 |
| 修正 | 必要時修改 | 持續檢查與改善 |
| 目標 | 通常固定 | 可提出演化建議 |
| 結束 | Task 完成 | 可持續進入下一輪 |

Task 是執行單位。

Experiment 是長期研究與改善單位。

一個 Experiment 可以包含多個 Project、Task、Decision、Worker Execution 與 Evaluation。

## 1.4 核心原則

AIA 必須能夠：

- 自主決定是否開始新的實驗。
- 自主判斷目前實驗是否需要繼續。
- 自主發現值得驗證的新假設。
- 自主提出策略修改。
- 自主評估修改效果。
- 在必要時選擇暫停或結束實驗。
- 維持原始目標與歷史紀錄。
- 在授權範圍內持續改善。

不應把實驗數量當成自主能力。

---

# 2. Autonomous Experiment Model

## 2.1 定義

Autonomous Experiment 是一個具有長期目標、持續觀察、可驗證假設、行動紀錄與評估機制的自主工作單位。

它可以包含：

- 一個或多個 Goal。
- 一個或多個 Project。
- 一個或多個 Domain Agent。
- 多個 Task。
- 多個 Worker。
- 多輪 Decision。
- 多次 Evaluation。
- 多個 Learning Record。
- 多個 Modification Proposal。

## 2.2 Experiment 與 Project 的差異

Project 是長期工作或成果管理單位。

Experiment 是研究某項能力、策略或假設的工作單位。

例如：

**Project：台股虛擬投資**

目標是讓投資組合總資產增加。

**Experiment：投資策略改善**

目標是在不改變投資組合總資產最大化方向的前提下，驗證某項投資判斷方法是否能改善長期成果。

兩者可以關聯，但不能混為一談。

一個 Project 可以同時包含多個 Experiment。

一個 Experiment 也可以支援多個 Project，但必須清楚記錄責任與成果歸屬。

## 2.3 Experiment Scope

每個 Experiment 必須明確定義：

| 欄位 | 說明 |
|---|---|
| Experiment ID | 實驗識別碼 |
| Name | 實驗名稱 |
| Domain | 所屬領域 |
| Owner Agent | 負責 Agent |
| Parent Project | 關聯 Project |
| Primary Goal | 主要目標 |
| Hypothesis | 實驗假設 |
| Scope | 實驗範圍 |
| Constraints | 限制條件 |
| Evaluation Criteria | 評估標準 |
| Start Condition | 啟動條件 |
| Stop Condition | 停止條件 |
| Status | 實驗狀態 |

## 2.4 Experiment 類型

### Capability Experiment

研究 Agent 是否具備某種自主能力。

例如：

- 能否主動發現投資研究機會。
- 能否恢復長期 Project。
- 能否自主除錯。
- 能否主動發現歷史矛盾。

### Strategy Experiment

研究某種工作策略是否有效。

例如：

- 不同投資分析方法。
- 不同程式碼重構方式。
- 不同歷史校勘流程。

### System Improvement Experiment

研究 AIA 本身的架構或程式碼改善。

例如：

- Memory Retrieval 改善。
- Continuity Engine 改善。
- Initiative Engine 改善。
- Worker 排程策略改善。

### Cross-Domain Experiment

研究不同 Domain Agent 如何協作。

例如：

- Investment × Coding。
- Writing × Coding。
- 多 Agent 共同分析同一個問題。

---

# 3. Experiment Lifecycle

## 3.1 完整生命週期

```text
Experiment Discovery
        |
        v
Experiment Proposal
        |
        v
Goal & Hypothesis Definition
        |
        v
Baseline Establishment
        |
        v
Experiment Planning
        |
        v
Authorization Check
        |
        v
Execution
        |
        v
Observation & Evidence Collection
        |
        v
Evaluation
        |
        v
Root Cause Analysis
        |
        v
Correction / Strategy Adjustment
        |
        v
Learning & Continuity Update
        |
        v
Next Experiment Decision
        |
        +------> Continue
        |
        +------> Modify
        |
        +------> Defer
        |
        +------> Stop
```

## 3.2 Experiment Discovery

實驗可以來自：

- 使用者提出的研究目標。
- Autonomy Core 主動發現的問題。
- Domain Agent 的專業分析。
- AIA-13 評估結果。
- AIA-11 Learning Record。
- AIA-14 Modification Proposal。
- 過去實驗中尚未解決的問題。

Agent 不必每次都建立新實驗。

若現有 Experiment 已能處理問題，應優先延續原有實驗。

## 3.3 Experiment Proposal

每個新實驗必須建立 Proposal。

包含：

- 問題描述。
- 研究目的。
- 實驗假設。
- 預期結果。
- 可能的替代解釋。
- 執行方法。
- 資源需求。
- 風險。
- 評估方式。
- 停止條件。

## 3.4 Hypothesis Definition

實驗假設必須能被觀察或驗證。

例如：

不合格：

「改善投資策略。」

合格：

「在相同市場資訊與投資條件下，增加產業供應鏈資訊的分析，是否能改善投資決策的預測校準程度？」

不合格：

「讓程式碼更好。」

合格：

「透過改善資料存取層的快取策略，是否能降低指定工作負載下的平均處理時間，同時維持資料正確性？」

不合格：

「讓小說更完整。」

合格：

「在每卷章節規劃階段加入人物時間線交叉檢查，是否能降低後續章節發生人物年齡與歷史事件衝突的次數？」

## 3.5 Baseline Establishment

每個實驗都應先建立基準。

基準包含：

- 原始系統版本。
- 原始策略。
- 原始工作狀態。
- 原始 Metric。
- 初始環境。
- 已知限制。
- 評估期間。

沒有基準，就無法可靠判斷改善程度。

## 3.6 Execution

實驗執行由 Domain Agent 與 Worker 負責。

Autonomy Core 負責：

- 工作發現。
- 決策。
- 資源分配。
- 任務協調。
- 執行監督。
- 結果評估。

Worker 是實際執行單位，不代表它具有獨立的實驗決策權。

## 3.7 Observation & Evidence Collection

執行期間應持續收集：

- 環境變化。
- 決策。
- 執行結果。
- 失敗事件。
- 預測與實際差異。
- 外部資料。
- 資源消耗。
- 人類回饋。

實驗中途發現新的重要問題時，可以建立新的 Experiment Proposal，但不能直接竄改原實驗假設。

## 3.8 Evaluation

實驗結果應依照 AIA-13 評估。

至少分成：

1. Goal Progress。
2. Task Performance。
3. Autonomy Capability。
4. Strategy Effectiveness。
5. Resource Efficiency。
6. Governance Compliance。

不同面向不能混成單一成功或失敗判斷。

## 3.9 Next Experiment Decision

每輪實驗完成後，Agent 必須決定下一步：

- CONTINUE：依原假設繼續。
- MODIFY：修改實驗方法。
- NEW_EXPERIMENT：提出新實驗。
- DEFER：延後。
- MONITOR：持續觀察。
- STOP：停止。
- ASK_HUMAN：需要人類決策。

STOP 不代表實驗失敗。

當研究問題已經回答、資源效益不足、假設失效或環境不再適用時，停止可能是正確決策。

---

# 4. Experiment State Management

## 4.1 狀態定義

建議狀態：

| 狀態 | 說明 |
|---|---|
| PROPOSED | 已提出 |
| PENDING_APPROVAL | 等待核准 |
| READY | 條件已備妥 |
| RUNNING | 執行中 |
| PAUSED | 暫停 |
| EVALUATING | 評估中 |
| NEEDS_REVISION | 需要修正 |
| COMPLETED | 已完成 |
| STOPPED | 已停止 |
| FAILED | 實驗無法完成 |
| ARCHIVED | 已封存 |

## 4.2 狀態轉換

所有狀態轉換必須透過明確的 Lifecycle Event。

例如：

```text
PROPOSED
   |
   v
PENDING_APPROVAL
   |
   v
READY
   |
   v
RUNNING
   |
   v
EVALUATING
   |
   +----> COMPLETED
   |
   +----> NEEDS_REVISION
   |
   +----> PAUSED
   |
   +----> STOPPED
```

不允許 Agent 直接修改資料庫狀態來跳過必要流程。

## 4.3 Pause & Resume

實驗暫停時，必須保留：

- 當前狀態。
- 已完成工作。
- 未完成工作。
- 已取得證據。
- 尚未驗證的假設。
- 目前資源使用狀況。
- 下一步候選行動。
- 暫停原因。

Resume 時應重新檢查：

- 原始目標是否仍有效。
- 環境是否已變化。
- 依賴是否仍存在。
- 原有假設是否需要更新。
- 是否需要重新取得授權。

## 4.4 Experiment History

實驗歷史必須採用事件紀錄方式保存。

例如：

- EXPERIMENT_CREATED。
- HYPOTHESIS_DEFINED。
- BASELINE_CAPTURED。
- EXECUTION_STARTED。
- EVIDENCE_ADDED。
- EVALUATION_COMPLETED。
- CORRECTION_PROPOSED。
- STRATEGY_MODIFIED。
- EXPERIMENT_PAUSED。
- EXPERIMENT_RESUMED。
- EXPERIMENT_STOPPED。

歷史事件不可由 Agent 任意覆寫。

---

# 5. Domain Integration Architecture

## 5.1 架構原則

AIA 的自主能力應由共用 Autonomy Core 提供。

Investment、Coding、Writing 是不同的專業工作領域。

它們不應各自重新實作一套完整自主循環。

```text
                    AIA Runtime
                         |
                    Autonomy Core
                         |
       +-----------------+-----------------+
       |                 |                 |
   Investment          Coding            Writing
     Agent              Agent             Agent
       |                 |                 |
    Workers            Workers           Workers
       |                 |                 |
  Virtual Market     Code Projects     Writing Projects
       |                 |                 |
       +-----------------+-----------------+
                         |
                Experiment Management
                         |
                Evaluation & Learning
```

Experiment Management 負責管理跨 Domain 實驗與其生命週期。

Autonomy Core 負責自主決策。

Domain Agent 負責專業工作。

Worker 負責實際執行。

Evaluation Engine 負責評估。

## 5.2 Domain Agent Responsibility

每個 Domain Agent 必須提供：

- Capability Description。
- Project Context。
- Domain Knowledge。
- Execution Interface。
- Evidence Output。
- Evaluation Data。
- Learning Record。

Domain Agent 可以自行提出專業實驗，但必須遵守共用 Experiment Lifecycle。

## 5.3 Experiment Management Responsibility

Experiment Management 負責：

- 建立實驗。
- 管理實驗狀態。
- 關聯 Project 與 Goal。
- 管理假設與基準。
- 追蹤執行歷程。
- 管理實驗資源。
- 接收評估結果。
- 管理下一輪實驗。

它不負責取代 Domain Agent 的專業判斷。

---

# 6. Investment Autonomous Experiment

## 6.1 實驗定位

Investment 是 AIA 第一個完整的自主性實驗場景。

它具有：

- 真實外部市場環境。
- 可觀察的市場資訊。
- 明確的投資組合。
- 可量化的資產變化。
- 持續發生的市場事件。
- 可定期檢驗的結果。

因此適合測試長期 Initiative、Continuity、Self-Correction 與 Goal Evolution。

## 6.2 基本實驗設定

| 項目 | 設定 |
|---|---|
| 初始資金 | NT$100,000 |
| 資金性質 | 虛擬資金 |
| 市場 | 台灣證券市場 |
| 特別關注 | 台積電 2330 |
| 觀察頻率 | 每個證券營業日 |
| 決策 | Agent 自主判斷 |
| 交易 | 虛擬買進、賣出、增持、減持 |
| 主要目標 | 最大化投資組合總資產 |
| 人類檢驗 | 每月 1 日 |
| 真實交易 | 不執行 |

## 6.3 核心目標不可被實驗取代

投資組合總資產最大化是主要目標。

實驗可以研究：

- 新的選股方法。
- 不同市場資訊的影響。
- 產業供應鏈分析。
- 台積電與國際市場連動。
- 持股與現金配置。
- 不同投資策略的有效性。

但不能將實驗目標改成：

- 交易次數最大化。
- 風險最低化。
- 現金比例最大化。
- 單一股票報酬最大化。

這些可以是研究指標，但不能取代主要目標。

## 6.4 每個證券營業日的實驗循環

```text
Market Observation
        |
        v
Portfolio State Reconstruction
        |
        v
News / Event Analysis
        |
        v
Historical Decision Recall
        |
        v
Investment Assessment
        |
        v
Candidate Action Generation
        |
        v
Risk & Opportunity Analysis
        |
        v
Decision
        |
        +----> BUY
        |
        +----> SELL
        |
        +----> HOLD
        |
        +----> CASH
        |
        +----> MONITOR
        |
        v
Virtual Execution (if applicable)
        |
        v
Portfolio Update
        |
        v
Decision & Evidence Record
        |
        v
Future Evaluation
```

每天進行觀察與判斷，不代表每天必須成交。

## 6.5 每月檢驗

每月 1 日進行投資組合檢驗。

至少呈現：

- 初始資金。
- 目前總資產。
- 現金。
- 持股市值。
- 累積報酬。
- 本月交易。
- 本月重大決策。
- 預測與結果差異。
- 策略修正。
- 主要風險。
- 下月觀察方向。

月度檢驗不只檢查獲利，也要檢查 Agent 的自主判斷品質。

## 6.6 Investment Learning

投資 Agent 應從以下資料持續學習：

- 過去投資決策。
- 當時可取得的市場資訊。
- 當時的持股與現金。
- 預測及信心程度。
- 實際市場變化。
- 虛擬交易結果。
- 策略表現。
- 人類月度檢驗回饋。

不得單純以短期獲利或虧損判斷策略正確性。

## 6.7 Investment Self-Modification

Investment Agent 可以提出修改：

- 市場資料處理程式。
- 研究工具。
- 投資組合分析程式。
- 策略模擬程式。

但修改仍須遵守 AIA-14。

任何程式碼修改都不得使虛擬投資轉變成未經核准的真實金融交易。

---

# 7. Coding Autonomous Experiment

## 7.1 實驗定位

Coding 是 AIA 第二個重要的自主性實驗場景。

它不只是讓 Agent 幫使用者寫程式，而是觀察 Agent 是否能長期維護、除錯、優化並改善既有程式專案。

## 7.2 實驗目標

Coding Agent 必須能：

1. 理解既有專案。
2. 維持架構知識。
3. 主動發現技術問題。
4. 主動提出改善方向。
5. 自行分析錯誤。
6. 自行修正程式碼。
7. 驗證修改結果。
8. 維持 Regression Test。
9. 管理技術債。
10. 追蹤長期程式品質。

## 7.3 Coding Experiment 範例

**Experiment：改善某個資料處理模組**

假設：

現有資料處理流程在特定工作負載下效率不足。

Agent 可以：

1. 主動觀察效能資料。
2. 發現效能瓶頸。
3. 分析根本原因。
4. 提出改善假設。
5. 建立實驗分支。
6. 修改程式碼。
7. 執行測試。
8. 比較效能。
9. 檢查 Regression。
10. 決定是否採用修改。

## 7.4 長期連續性

Coding Agent 必須記得：

- 專案架構。
- 技術選擇。
- 設計決策。
- 已知問題。
- 技術債。
- 歷史修改。
- 已知 Regression。
- 尚未完成的改善。

不能每次開啟 Session 都重新分析整個專案。

## 7.5 Coding 與 AIA Core

Coding Agent 可以修改使用者授權的 Domain Project。

若目標是 AIA 自身程式碼，則進入 AIA Core Modification 流程。

不得將使用者專案與 AIA 核心 Repository 混為一談。

---

# 8. Writing Autonomous Experiment

## 8.1 實驗定位

Writing 是 AIA 第三個自主性實驗場景。

它要測試 Agent 是否能在長期創作中維持整體架構，並根據新史料、人物設定與情節進度持續修正。

## 8.2 實驗目標

Writing Agent 必須能：

1. 維護整體大綱。
2. 維護各卷進度。
3. 維護人物設定。
4. 維護人物關係。
5. 維護歷史年表。
6. 檢查史實。
7. 發現跨章節矛盾。
8. 發現跨卷衝突。
9. 主動提出修正建議。
10. 保留創作決策與修訂歷史。

## 8.3 Writing Experiment 範例

**Experiment：建立跨卷歷史一致性檢查**

假設：

如果在寫作階段持續檢查人物年齡、官職、歷史事件與地理位置，可以減少後續修稿成本。

Agent 可以：

1. 建立人物時間線。
2. 建立歷史事件時間線。
3. 建立人物與地點關聯。
4. 分析章節內容。
5. 發現衝突。
6. 提出修正方案。
7. 更新相關文件。
8. 檢查其他章節是否受到影響。
9. 記錄修訂原因。
10. 追蹤後續矛盾發生率。

## 8.4 史實與小說設定分離

Writing Agent 必須區分：

- 史實。
- 合理推演。
- 小說虛構。

不能因為小說情節需要，就直接改寫歷史證據。

新史料出現時，必須檢查：

- 是否可信。
- 是否有其他來源支持。
- 是否影響既有判斷。
- 是否需要修改人物或事件設定。
- 是否需要修改已完成章節。

## 8.5 Writing Self-Correction

Writing Agent 的修正不應只限於新章節。

它也應能發現：

- 已完成章節的矛盾。
- 人物關係不一致。
- 歷史時間錯置。
- 地理描述不合理。
- 社會制度不符合時代。
- 各卷敘事節奏失衡。

修正時必須保留原始版本與修改原因。

---

# 9. Cross-Domain Experiment

## 9.1 設計目的

Cross-Domain Experiment 用來測試不同專業 Agent 是否能共同解決單一 Agent 難以完成的問題。

它不是單純把多個 Agent 同時叫起來工作。

必須有明確的跨領域需求與整合成果。

## 9.2 Investment × Coding

範例：

Investment Agent 發現市場資料分析工具存在資料錯誤。

流程：

1. Investment Agent 提出問題。
2. Autonomy Core 判斷需要技術協助。
3. Coding Agent 接受委派。
4. Coding Agent 分析資料處理程式。
5. Coding Agent 提出修正。
6. 驗證工具輸出。
7. Investment Agent 重新分析資料。
8. Evaluation Engine 比較修正前後的結果。

評估重點：

- 是否正確發現跨領域問題。
- 是否選擇合適 Agent。
- 委派成本是否合理。
- 修正是否真正改善投資分析品質。
- 是否保留責任與證據。

## 9.3 Writing × Coding

範例：

Writing Agent 發現歷史小說專案需要更可靠的人物年表檢查工具。

流程：

1. Writing Agent 定義檢查需求。
2. Coding Agent 分析需求。
3. Coding Agent 建立檢查工具。
4. Writing Agent 提供人物與事件資料。
5. Coding Agent 執行工具。
6. Writing Agent 檢查結果。
7. 雙方共同處理例外情況。

評估重點：

- 需求理解。
- 工具設計。
- 資料結構。
- 歷史語境。
- 檢查結果品質。
- 長期可維護性。

## 9.4 Investment × Writing

兩者可以交換部分一般研究方法，但不能直接共享專業判斷。

例如：

- 資料來源管理。
- 時間線管理。
- 證據追溯。
- 假設檢查。
- 預測與結果比較。

投資市場判斷不能直接套用於歷史研究。

歷史史料校勘方法也不能直接當成投資策略。

Cross-Domain Knowledge 必須先經過適用性檢查。

## 9.5 三 Agent 協作

可建立更複雜的實驗：

- Investment 提出資料分析需求。
- Coding 建立分析工具。
- Writing 整理相關產業歷史或制度資料。
- Autonomy Core 整合成果。
- Evaluation Engine 評估整體工作品質。

但三個 Agent 不需要為了協作而強制參與每個實驗。

如果單一 Agent 就能完成工作，應避免不必要的委派。

---

# 10. Experiment Resource Management

## 10.1 資源類型

每個 Experiment 必須記錄：

- Model Usage。
- Token Consumption。
- Worker Execution Time。
- CPU。
- Memory。
- Storage。
- Network。
- External API Usage。
- Human Review Time。

## 10.2 Resource Budget

實驗可以設定：

- Maximum Budget。
- Time Limit。
- Maximum Workers。
- Maximum Concurrent Tasks。
- Retry Limit。
- External API Limit。

超出預算時，Agent 必須重新評估：

- 是否值得繼續。
- 是否降低實驗範圍。
- 是否等待更多資料。
- 是否需要人類核准增加資源。

## 10.3 Resource Allocation

Autonomy Core 可以根據實驗價值、緊急程度、成本與依賴關係分配資源。

但不能為了某個實驗而擅自挪用其他 Project 已經核准的資源。

## 10.4 Experiment Priority

建議評估：

- Goal Contribution。
- Expected Value。
- Information Gain。
- Time Sensitivity。
- Risk。
- Cost。
- Opportunity Cost。
- Dependency。

不應只按照預期產出數量排序。

---

# 11. Experiment Continuity & Memory

## 11.1 Continuity Integration

每個 Experiment 必須與 AIA-10 的 Continuity Model 整合。

至少保留：

- Experiment State。
- Goal。
- Hypothesis。
- Baseline。
- Decisions。
- Evidence。
- Results。
- Learning。
- Next Action Candidates。

## 11.2 Context Reconstruction

重新啟動 Experiment 時，應能重建：

1. 原始研究問題。
2. 目前假設。
3. 已經完成的實驗。
4. 歷史結果。
5. 已經排除的假設。
6. 尚未解決的問題。
7. 目前執行狀態。
8. 下一步候選行動。

## 11.3 Experiment Memory

Memory 必須區分：

- 原始假設。
- 暫時假設。
- 已驗證結果。
- 尚未驗證結果。
- 已被否定的假設。
- 已失效的知識。
- 歷史決策。

不能將所有實驗結果都直接升級成永久知識。

## 11.4 Experiment Knowledge Promotion

建議採用：

- PROPOSED。
- OBSERVED。
- REVIEWED。
- VERIFIED。
- DOMAIN_APPROVED。
- DEPRECATED。

不同狀態應保留來源、驗證方式與適用範圍。

---

# 12. Experiment Evaluation

## 12.1 評估原則

每個實驗至少應分別評估：

### A. Experiment Performance

實驗是否回答原始研究問題？

### B. Domain Performance

實驗是否改善所屬 Domain 的實際工作成果？

### C. Autonomy Performance

Agent 是否展現更好的自主能力？

### D. Resource Efficiency

成果是否值得投入的資源？

### E. Governance Compliance

實驗是否遵守權限與安全限制？

## 12.2 Evaluation Metrics

可以使用：

| 類別 | 指標 |
|---|---|
| Goal | Goal Progress |
| Initiative | Useful Initiative Rate |
| Continuity | Continuity Recovery Accuracy |
| Correction | Correction Success Rate |
| Learning | Learning Application Rate |
| Experiment | Hypothesis Validation Rate |
| Domain | Domain Outcome Improvement |
| Resource | Cost per Verified Improvement |
| Governance | Policy Compliance |

## 12.3 Negative Result

實驗沒有達到預期結果，不代表實驗沒有價值。

如果實驗能夠：

- 排除錯誤假設。
- 發現未知限制。
- 降低未來風險。
- 改善問題理解。
- 提供新的研究方向。

仍可能產生有價值的知識。

因此，不能只保留成功實驗。

## 12.4 Experiment Comparison

不同實驗比較時，必須考慮：

- 環境差異。
- 資料差異。
- 模型差異。
- 執行時間。
- 資源投入。
- 評估標準版本。

不能直接用不同環境下的原始數字比較。

---

# 13. Human Oversight

## 13.1 人類角色

使用者是 AIA 的最終目標設定者與重要治理者。

Agent 可以：

- 自主發現問題。
- 提出實驗。
- 提出目標拆解。
- 提出策略變更。
- 提出資源需求。
- 提出系統改善。

但以下事項必須依 Policy 取得人類核准：

- 重大目標方向變更。
- 資源預算增加。
- AIA Core 正式部署。
- Governance 修改。
- 超出原授權範圍的行動。
- 其他重大不可逆操作。

## 13.2 Human Review

人類應定期檢查：

- 實驗目標是否仍符合原始方向。
- 實驗是否有實際價值。
- Agent 是否過度消耗資源。
- 實驗結果是否可信。
- 是否需要調整實驗範圍。
- 是否需要停止實驗。

## 13.3 Human Feedback Integration

人類回饋必須保留：

- Feedback。
- Related Experiment。
- Related Decision。
- Feedback Time。
- Agent Interpretation。
- Follow-up Action。
- Verification Result。

Agent 不應只將人類回饋當成一次性指令，而應判斷它是否代表長期設計原則或單次修正要求。

---

# 14. Experiment Security & Governance

## 14.1 Experiment Isolation

不同實驗應有清楚的資源與資料邊界。

避免：

- 一個實驗修改另一個實驗的程式碼。
- 一個實驗覆蓋另一個實驗的基準。
- 不同實驗混用未驗證結果。
- 跨 Project 未授權存取。

## 14.2 Experiment Authority

Experiment 不會自動取得額外權限。

建立 Experiment 只是建立研究工作單位。

實際執行仍須遵守：

- Agent Capability。
- Worker Permission。
- Project Policy。
- Harness Permission。
- Approval Requirement。

## 14.3 Experiment Data Integrity

實驗歷史、原始資料與評估結果必須保留版本與來源。

Agent 不得：

- 刪除失敗實驗。
- 修改原始假設以製造成功。
- 修改歷史基準。
- 隱藏不利證據。
- 將推論結果偽裝成直接觀察。

## 14.4 Virtual Investment Boundary

Investment Experiment 永遠遵守虛擬投資範圍。

不得因為實驗需要或績效壓力：

- 連接真實證券帳戶自動下單。
- 自動轉移使用者資金。
- 自動執行真實買賣。
- 擅自取得金融帳戶權限。

---

# 15. Go Implementation Specification

## 15.1 建議 Package

```text
internal/
  experiment/
    manager/
    lifecycle/
    proposal/
    hypothesis/
    baseline/
    execution/
    evidence/
    evaluation/
    resource/
    continuity/
    model/
    repository/
```

實際 Package 應整合現有 Runtime、Task、Project、Worker、Memory 與 Evaluation，不應重複建立另一套執行管理系統。

## 15.2 核心資料模型

### Experiment

```go
type Experiment struct {
    ID string
    Name string

    Domain string
    OwnerAgentID string

    ProjectIDs []string
    GoalIDs []string

    PrimaryGoal string
    Hypothesis string

    Scope string
    Constraints []string

    Status ExperimentStatus

    StartCondition string
    StopCondition string

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Experiment Run

```go
type ExperimentRun struct {
    ID string
    ExperimentID string

    BaselineID string
    HypothesisVersion string

    StartedAt time.Time
    CompletedAt *time.Time

    Status string

    TaskIDs []string
    DecisionIDs []string
    EvidenceRefs []string

    EvaluationID string
}
```

### Experiment Hypothesis

```go
type ExperimentHypothesis struct {
    ID string
    ExperimentID string

    Version string
    Statement string

    Assumptions []string
    ExpectedOutcome string

    EvaluationCriteria []string

    Status string
    CreatedAt time.Time
}
```

### Experiment Event

```go
type ExperimentEvent struct {
    ID string
    ExperimentID string

    EventType string
    PayloadReference string

    ActorID string
    CreatedAt time.Time
}
```

以上為概念性資料模型，實際實作應遵循既有 Model 與 Repository 規範。

## 15.3 核心介面

```go
type ExperimentManager interface {
    Create(
        ctx context.Context,
        request CreateExperimentRequest,
    ) (*Experiment, error)

    Get(
        ctx context.Context,
        experimentID string,
    ) (*Experiment, error)

    List(
        ctx context.Context,
        filter ExperimentFilter,
    ) ([]Experiment, error)

    Pause(
        ctx context.Context,
        experimentID string,
    ) error

    Resume(
        ctx context.Context,
        experimentID string,
    ) error

    Stop(
        ctx context.Context,
        experimentID string,
    ) error
}
```

```go
type ExperimentExecutor interface {
    Execute(
        ctx context.Context,
        experimentID string,
    ) (*ExperimentRun, error)
}
```

```go
type ExperimentEvaluator interface {
    Evaluate(
        ctx context.Context,
        runID string,
    ) (*ExperimentEvaluation, error)
}
```

```go
type ExperimentContinuity interface {
    Reconstruct(
        ctx context.Context,
        experimentID string,
    ) (*ExperimentContext, error)
}
```

這些介面定義功能邊界，不要求全部由單一 Service 實作。

## 15.4 與現有 Worker 整合

AIA 已有 Worker 執行機制。

Experiment 不應重新實作 Worker。

應由 Experiment Manager：

1. 建立實驗執行需求。
2. 透過既有 Task 系統建立工作。
3. 交由合適的 Agent 與 Worker 執行。
4. 收集執行結果。
5. 交由 Evaluation Engine 評估。
6. 更新 Experiment State。

Worker 的存在只代表具備工作執行能力，不代表 Experiment Lifecycle 已經完成。

## 15.5 CLI 建議

```bash
aia experiment list

aia experiment show <experiment-id>

aia experiment create

aia experiment pause <experiment-id>

aia experiment resume <experiment-id>

aia experiment stop <experiment-id>

aia experiment runs <experiment-id>

aia experiment report <experiment-id>

aia experiment evaluate <run-id>
```

命令應依現有 CLI 慣例調整。

## 15.6 Web Console 建議

提供：

- Experiment Overview。
- Active Experiments。
- Experiment Timeline。
- Hypothesis History。
- Baseline Comparison。
- Evidence Viewer。
- Evaluation Results。
- Resource Usage。
- Cross-Domain Collaboration。
- Human Approval。
- Experiment History。

---

# 16. Implementation Phases

## X0：Experiment Foundation

建立實驗基本資料結構與生命週期。

工作：

- Experiment Model。
- Experiment State。
- Experiment Event。
- Goal / Project Association。
- Experiment Repository。

驗收：

- 能建立實驗。
- 能查詢實驗。
- 能管理實驗狀態。
- 能保留歷史事件。

## X1：Experiment Lifecycle

建立完整實驗流程。

工作：

- Proposal。
- Hypothesis。
- Baseline。
- Execution。
- Evidence。
- Evaluation。
- Next Action Decision。

驗收：

- 能從提案進入執行。
- 能保留假設與基準。
- 能收集執行證據。
- 能完成評估。
- 能決定下一輪行動。

## X2：Domain Integration

整合三個 Domain Agent。

工作：

- Investment Experiment。
- Coding Experiment。
- Writing Experiment。
- Domain-specific Evidence。
- Domain Evaluation。

驗收：

- 三個 Agent 能建立自己的實驗。
- 能使用共同生命週期。
- 能保留各 Domain 專業資料。
- 不會互相覆蓋專業判斷。

## X3：Cross-Domain Experiment

建立跨領域實驗。

工作：

- Capability Matching。
- Delegation。
- Shared Evidence。
- Result Integration。
- Responsibility Tracking。

驗收：

- 能建立跨 Agent 實驗。
- 能正確委派。
- 能整合結果。
- 能追蹤責任。
- 能避免不必要委派。

## X4：Long-term Experiment Continuity

建立長期實驗管理。

工作：

- Cross-Session Recovery。
- Experiment Memory。
- Long-term Evaluation。
- Hypothesis Evolution。
- Resource Management。

驗收：

- 能跨 Session 恢復實驗。
- 能追蹤歷史假設。
- 能比較不同實驗階段。
- 能保留長期成果。
- 能合理暫停、恢復與停止。

## X5：Autonomous Experiment Evolution

建立自主實驗演化。

工作：

- Experiment Discovery。
- Initiative Integration。
- Evaluation-driven Discovery。
- Correction Integration。
- Controlled Experiment Evolution。

驗收：

- Agent 能主動發現新實驗機會。
- 能提出可驗證假設。
- 能自主決定延續、修改或停止。
- 能將結果應用到下一輪。
- 能在治理限制內持續改善。

---

# 17. Acceptance Criteria

## 17.1 Experiment Management

- [ ] 能建立長期 Experiment。
- [ ] 能關聯 Project 與 Goal。
- [ ] 能管理 Hypothesis。
- [ ] 能保存 Baseline。
- [ ] 能追蹤 Experiment State。
- [ ] 能保留完整歷史。

## 17.2 Autonomous Lifecycle

- [ ] 能主動發現實驗機會。
- [ ] 能提出 Experiment Proposal。
- [ ] 能建立可驗證假設。
- [ ] 能執行實驗。
- [ ] 能收集證據。
- [ ] 能完成評估。
- [ ] 能決定下一輪行動。

## 17.3 Domain Integration

- [ ] Investment 能持續執行虛擬投資實驗。
- [ ] Coding 能持續執行程式改善實驗。
- [ ] Writing 能持續執行長期創作與校勘實驗。
- [ ] 三個 Domain 使用共同 Autonomy Core。
- [ ] Domain 專業能力維持獨立。

## 17.4 Cross-Domain Integration

- [ ] 能發現跨領域需求。
- [ ] 能選擇合適 Agent。
- [ ] 能執行 Delegation。
- [ ] 能整合成果。
- [ ] 能追蹤責任。
- [ ] 能避免知識錯誤套用。

## 17.5 Continuity & Learning

- [ ] 能跨 Session 恢復實驗。
- [ ] 能記住歷史假設。
- [ ] 能追蹤實驗結果。
- [ ] 能保留失敗經驗。
- [ ] 能將 Learning 應用於後續實驗。

## 17.6 Governance

- [ ] 實驗不能自行擴張權限。
- [ ] 重大目標變更需要核准。
- [ ] 資源預算受到限制。
- [ ] 實驗歷史不可任意修改。
- [ ] 投資實驗維持虛擬交易。
- [ ] 核心程式碼修改遵守 AIA-14。

---

# 18. Risk & Mitigation

| 風險 | 說明 | 控制方式 |
|---|---|---|
| Experiment Proliferation | Agent 不斷建立實驗 | Duplicate Detection |
| Goal Fragmentation | 實驗偏離原始目標 | Goal Association |
| Endless Experiment | 實驗無限延續 | Stop Condition |
| False Hypothesis | 假設無法驗證 | Hypothesis Review |
| Resource Exhaustion | 實驗消耗過多資源 | Budget Control |
| Cross-Domain Contamination | 專業知識錯誤套用 | Applicability Review |
| False Learning | 錯誤結論被永久保留 | Evidence Promotion |
| Experiment Conflict | 實驗互相干擾 | Scope Isolation |
| Evaluation Bias | 只呈現成功實驗 | Immutable History |
| Unauthorized Action | 實驗擴張權限 | Policy Enforcement |

---

# 19. 與其他 AIA 文件的關係

| 文件 | 關係 |
|---|---|
| AIA-05 | 定義第一個 Domain：Investment |
| AIA-08 | 提供共用自主運作循環 |
| AIA-09 | 提供 Initiative 與 Goal Evolution |
| AIA-10 | 提供長期連續性 |
| AIA-11 | 提供 Learning 與 Self-Correction |
| AIA-12 | 提供跨 Agent 協作 |
| AIA-13 | 提供自主性評估 |
| AIA-14 | 提供安全程式碼修改 |
| AIA-15 | 整合所有能力，形成長期自主實驗生命週期 |

AIA-15 不取代 Task、Project 或 Worker。

它將這些既有元件組合成更高層次的長期實驗管理機制。

---

# 20. 最終設計原則

AIA 的核心價值，不是能夠同時執行多少個 Agent，也不是每天能夠產生多少個 Task。

真正值得研究的是：

Agent 是否能夠持續理解自己的工作狀態，發現值得處理的問題，選擇適當的行動，驗證結果，修正錯誤，並在長時間後仍然朝著原始目標前進。

Investment、Coding、Writing 三個 Domain 提供不同的觀察環境：

- Investment 提供持續變化的外部市場與量化結果。
- Coding 提供可以修改、測試、驗證與回滾的程式環境。
- Writing 提供長期知識累積、複雜關係與跨時間一致性問題。

三者不是三套互不相關的自主系統。

它們是共用 Autonomy Core 的不同實驗場景。

最終，AIA 應形成：

**Observe → Discover → Experiment → Execute → Evaluate → Correct → Learn → Evolve。**

每一輪工作都應該讓下一輪更有根據，而不是單純重複執行。

**AIA-15 的最終目標，是讓 AIA 從一個能夠執行任務的 Agent Platform，逐步成為一個能夠長期進行自主實驗、驗證自身能力並持續改善的系統。**

---

# 21. 雙世代演進實驗控制器規範 (Generational Experiment Controller — v2.1 Normative Addendum)

為串聯 `AIA-05` 投資領域、`AIA-11` 可驗證自我修正與 `AIA-13` 自主性基準，Experiment Management 正式定義 **世代演進控制器 (`ExperimentController`)**：

1. **三角色職責隔離**：
   - **`G1` (`CONTROL_BASELINE`)**：規則版對照組，資料庫層強制 `is_frozen = 1` 並以 SQLite Trigger 禁止任何規則版本或組態覆寫。
   - **`G2` (`EXPERIMENTAL`)**：由 `agy` Harness 承載之初始 LLM 決策世代（`agy` 僅為 LLM 執行環境，並非整個實驗控制器）。
   - **`AIA ExperimentController`**：負責固定當下資訊切面（`AsOf` 快照）、獨立維護各世代 `NT$100,000` 虛擬帳本、結算預測、執行世代對照與管理版本譜系。
2. **非破壞式世代演化 (`G2` ➔ `G2.1` ➔ `G2.2`)**：
   - 當 `G2` 經由 `AIA-11` 完成偏差評估、RCA 與修正提案後，嚴禁原地覆蓋 `G2`；必須衍生子世代 `G2.x`（記錄 `parent_generation_id`、`harness_version`、`model_id`、`prompt_version`、`strategy_version`、`source_correction_id` 與 `source_learning_id`）。
   - `G2.x` 在獨立虛擬帳本中與永久基準 `G1` 同步接受相同市場資料檢驗，並將相對 `G1` 的對照表現回傳至 `AIA-11` 的 `VerifyCorrection`（採用 `CONTROL_GROUP` 對照法），決定正式晉升（`PROMOTED` / `L2_VERIFIED`）或淘汰回滾（`ROLLED_BACK` / `REGRESSED`）。

---

# 22. Phase 5 Entry Gate（三大領域自主進場驗收門檻）

依據架構檢討與 `thinking.md` 決議，Phase 5 不得作為首次開發 Coding Agent 與 Writing Agent 之階段。在全面啟動長週期跨領域實驗之前，三大 Domain Agent 必須全數通過 **Phase 5 Entry Gate**，具備「最低可用版本（Minimum Viable Agent, MVA）」。

### 22.1 七項硬性進場驗收指標
1. **Autonomy Core 統一性**：Investment、Coding、Writing 必須 100% 透過相同的 11 步自主循環（`internal/autonomy/`）驅動，嚴禁為個別領域獨立開發分立的調度引擎。
2. **時間連續性 (Continuity)**：三個 Agent 遭中斷重啟後，皆能透過 `event_history`、`projects` 與 Context Snapshot 完整重建未完成事項。
3. **主動性 (Initiative)**：三個 Agent 的 Domain Observer 皆能自主掃描環境，主動發現至少一項有價值之工作候選（`candidate_initiatives`）。
4. **有據可查的理性觀望 (Rational NO_ACTION)**：三個 Agent 在無需介入時，皆能產出具備明確證據與推演理由的 `NO_ACTION` 決策（投資：未達顯著性；Coding：良性警告或非關鍵債；Writing：刻意保留之小說虛構/藝術誇張）。
5. **可驗證之自我修正 (Verifiable Self-Correction)**：三個 Agent 皆能走完 `OutcomeEvaluation -> RCA -> CorrectionAction -> Verification` 閉環，達到 `L2_VERIFIED`。
6. **領域客觀基準 (Objective Baseline)**：
   - Investment：以凍結之 `G1` Rule-Based Baseline 作為活體對照。
   - Coding：以單元測試回歸全綠（`go test ./...` 100% PASS）與編譯診斷作為客觀判定基準。
   - Writing：以不可違背之【史實骨架】（生卒年、戰役年表、官制）作為客觀判定基準。
7. **跨領域委派契約 (Cross-Domain Delegation)**：至少完成一次端到端委派閉環（如 Writing 委派 Coding 開發年表檢核工具）。

---

# 23. Writing Agent 最低可用版本 (MVA) 規格

### 23.1 素材定位與三層知識分類
Writing Agent 以長篇歷史群像小說（以 `~/src/github/novel/yuan/` 6 卷 540 章為實體素材）為驗證場景，管理跨卷、跨年、跨人物之長期脈絡一致性。

系統嚴格劃分三種敘事層級：
1. **【史實骨架】(`FACT_HISTORY`)**：有正史明確記載之事件、年代、官職、生卒年（如朱元璋 1328 出生、1344 入皇覺寺；脫脫 1354 遭罷黜）。**嚴禁任何未授權修改；若小說出現衝突必須視為錯誤並提出修訂。**
2. **【合理推演】(`PLAUSIBLE_INFERENCE`)**：史料未載但符合時代制度、民俗、地理交通之情節（如各割據勢力之糧價、私鹽運銷、書信傳遞時效）。
3. **【小說虛構】(`FICTION`)**：新創配角（如四方幼徒、民俗神秘情節、藝術改編）。**Deliberation 必須辨別小說虛構並做出有據之 `NO_ACTION`，不可將創作情節誤判為史實錯誤。**

### 23.2 觀察器與異常檢測 (`ManuscriptConsistencyObserver`)
- 讀取小說專案結構（`characters.md`、`outline.md`、各卷逐章詳綱）。
- 建構人物生命歷程圖（登場年代、行蹤、生死狀態）。
- 檢測四類主要矛盾（`ConsistencyAnomaly`）：
  - **`POST_MORTEM_APPEARANCE`（卒後登場）**：人物於卒年之後再次以生者身份參與事件。
  - **`ANACHRONISM`（時代錯置）**：使用尚未發生之年號、官職或跨時空發明。
  - **`LOCATION_CONFLICT`（瞬間移動）**：人物在未經合理旅程時間下同時出現於千里之外。
  - **`CHARACTER_STATE_MISMATCH`（設定矛盾）**：人物能力、身分或親緣關係與先前設定相悖。

---

# 24. Coding Agent 最低可用版本 (MVA) 規格

### 24.1 定位與職責
Coding Agent 聚焦於**「問題發現、客觀測試與零退化自我修正」**，是檢驗 AIA 自我修正機制是否具備真成效之最精確試驗場。

### 24.2 觀察器與驗證流水線 (`CodeHealthObserver`)
- 掃描指定專案目錄，執行靜態檢查、編譯診斷與測試執行。
- 發現三類程式碼問題（`CodeDefect`）：
  - **`TEST_FAILURE`**：既有或新增之單元/整合測試失敗。
  - **`BUILD_ERROR`**：編譯錯誤或型別不相容。
  - **`REGRESSION_RISK`**：未處理之錯誤碼或並行交易競爭。
- **自我修正安全驗證**：
  - 於隔離沙盒工作區（`internal/workspace`）套用修正提案。
  - 執行測試套件：原測試通過 且 其他所有既有測試 100% 綠燈，方可晉升 `L2_VERIFIED`；任一回歸測試失敗立即標記 `REGRESSED` 並自動回滾。
