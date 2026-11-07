# AIA-08 — Autonomy Core Specification

> [!NOTE]
> **架構重整與收攏公告 (Consolidated Architecture Specification)**
> 本規範為 AIA 歷史基準規範（規格編號已於 AIA-22 永久封頂）。為杜絕規格膨脹與重複定義，本規範之現行核心標準已正式收攏重整至 **[Volume 2: Cognitive Loop, Memory & Planning](file:///home/wade/src/github/aia/docs/specifications/volume-2-cognitive-loop-and-planning.md)**。最新架構與維護基準請以對應之 Volume 為主，完整六大核心卷軸總覽請參閱 [Specification Catalog (README.md)](file:///home/wade/src/github/aia/docs/specifications/README.md)。

**Autonomous Intelligence Architecture**  
**Version:** 2.0  
**Document Type:** Core Capability Specification  
**Status:** Proposed Baseline  
**依賴文件：** AIA-00、AIA-01、AIA-02、AIA-03、AIA-04  
**後續相依文件：** AIA-09、AIA-10、AIA-11、AIA-12、AIA-13、AIA-14、AIA-15

---

# 1. 文件目的

本文件定義 AIA 的自主性核心（Autonomy Core），說明系統如何從單純接受指令、建立 Task、呼叫 Agent 的工作平台，逐步發展成能夠維持長期目標、主動發現工作、自我檢查及修正的自主系統。

Autonomy Core 是 AIA 的共同自主性能力，不屬於任何單一 Domain Agent。

Investment、Coding、Writing 是不同的工作領域，但它們應共用同一套自主性核心能力。

例如：

- Investment Agent 可以自行發現投資研究不足，提出改善工作。
- Coding Agent 可以自行發現程式錯誤，提出修正工作。
- Writing Agent 可以自行發現人物關係或歷史年表矛盾，提出校勘工作。
- Coding Agent 完成新工具後，Writing Agent 可以重新檢查過去的創作成果。
- Investment Agent 發現分析能力不足時，可以要求 Coding Agent 改善分析工具。

上述行為不應各自實作成三套互不相通的自主決策系統。

Autonomy Core 必須提供共同的觀察、記憶、評估、發現、推理、決策、執行協調及學習機制。

# 2. 核心設計原則

## 2.1 自主性不是排程

定時喚醒、Schedule、Worker、Task Queue 是系統持續運作的基礎設施，但不等於自主性。

例如：

- 每天早上 9 點啟動 Investment Agent。
- 每小時執行一次市場資料收集。
- 每天晚上檢查未完成 Task。

這些行為只能證明系統具有排程及自動執行能力。

真正的自主性還需要判斷：

- 現在有哪些事情值得做？
- 哪些問題比原定工作更重要？
- 是否需要重新檢查過去的決策？
- 是否應該提出新目標？
- 是否應該調整既有策略？
- 是否應該暫停或不採取行動？

Schedule 是 Runtime 的觸發機制，Autonomy Core 才是自主判斷機制。

## 2.2 自主性不是任務數量

系統不能以產生多少 Task、執行多少次 Agent、呼叫多少工具，作為自主性成長的證明。

一個成熟的 Autonomy Core 應該能夠：

- 發現真正有價值的工作。
- 拒絕沒有必要的工作。
- 避免重複執行已經完成的工作。
- 發現目前缺少足夠資訊。
- 在必要時等待。
- 對過去的決策進行重新評估。

合理的不行動（No-Action Decision）也是自主決策的一種。

## 2.3 自主性不是單純記憶

保存對話紀錄、工作紀錄、人物設定、投資資料，不代表系統具有時間連續性。

Autonomy Core 必須能夠利用這些資料重建：

- 目前目標。
- 目標形成的原因。
- 過去做過的決策。
- 已完成及未完成的工作。
- 曾經遇到的問題。
- 過去修正的結果。
- 目前仍然有效的限制。
- 下一步值得考慮的行動。

記憶是資料基礎，連續性是系統行為。

## 2.4 自我修正不等於修改 Prompt

修改 Prompt、調整參數或產生新的 Task，都不能單獨證明系統具備自我修正能力。

完整的自我修正必須具備：

1. 有明確的預期結果。
2. 有可觀察的實際結果。
3. 有評估標準。
4. 能識別差異。
5. 能分析原因。
6. 能提出修正。
7. 能驗證修正效果。
8. 能保存修正經驗。

## 2.5 自主性必須受到治理

Autonomy Core 可以自行提出目標、工作及修改方案，但不能因此取得無限制的執行權。

所有行動仍須遵守：

- Policy。
- Permission。
- Approval。
- Resource Limit。
- Audit。
- Domain-specific Constraints。

Agent 不得透過自主決策或自我修改解除上述限制。

# 3. Autonomy Core 的四項核心能力

## 3.1 Initiative — 主動性思維

### 3.1.1 定義

Initiative 是系統根據環境、目標、記憶及工作狀態，主動發現值得處理的問題與機會，並提出適當行動的能力。

Initiative 不應只依賴人類提出指令。

### 3.1.2 主要能力

- Environment Observation。
- Problem Detection。
- Opportunity Discovery。
- Goal Gap Detection。
- Pending Work Detection。
- Anomaly Detection。
- Action Candidate Generation。
- Priority Assessment。
- Initiative Proposal。
- No-Action Decision。

### 3.1.3 具體情境

**Investment**

系統發現台積電相關重大資訊出現，且可能影響現有持股判斷。

它可以主動提出重新研究及評估的工作，而不必等使用者詢問。

**Coding**

系統發現某個程式模組的錯誤率增加，或既有介面已無法滿足新需求。

它可以提出除錯或重構工作。

**Writing**

系統發現小說某卷的歷史時間軸與人物年齡不一致。

它可以主動建立校勘工作，並評估是否需要重新檢查相關章節。

### 3.1.4 不屬於 Initiative 的行為

以下行為本身不代表主動性：

- Schedule 到期後建立 Task。
- 使用者輸入指令後派遣 Agent。
- Agent 依照 Prompt 執行固定步驟。
- 每次喚醒都強制產生一個新 Task。

Initiative 必須包含對現況的評估及行動價值的判斷。

## 3.2 Continuity — 時間連續性

### 3.2.1 定義

Continuity 是系統在不同時間、Session、Task、Project 及 Domain Agent 之間，持續維護目標、工作狀態、決策脈絡及經驗的能力。

### 3.2.2 主要能力

- Persistent Context。
- Long-term Memory。
- Goal Continuity。
- Project Continuity。
- Decision History。
- Work History。
- Evaluation History。
- Context Reconstruction。
- Project Resumption。
- Cross-Session Continuity。
- Cross-Domain Context Transfer。

### 3.2.3 具體情境

使用者正在創作一部八卷歷史小說。

Writing Agent 已經掌握：

- 八卷的整體規劃。
- 每卷的歷史時間範圍。
- 人物年齡及關係。
- 各章節的完成進度。
- 已確認的史實。
- 尚待查證的歷史問題。
- 已發現的劇情矛盾。

之後系統轉去處理 Investment 工作。

當它再次回到 Writing Project 時，不應重新詢問使用者小說寫到哪裡，也不應只讀取最近一次對話。

它必須從 Project State、Goal History、Work History 及相關 Memory 重建完整工作脈絡。

### 3.2.4 Continuity 的判斷標準

真正的 Continuity 不只是能夠回憶過去，而是能夠根據過去的狀態，正確決定現在應該如何繼續。

詳細資料結構及專案管理機制由 AIA-10 定義。

## 3.3 Self-Correction — 自我修正

### 3.3.1 定義

Self-Correction 是系統根據預期與實際結果的差異，識別問題、分析原因、調整行為，並驗證改善效果的能力。

### 3.3.2 主要能力

- Outcome Evaluation。
- Prediction Evaluation。
- Error Detection。
- Error Classification。
- Root Cause Analysis。
- Strategy Adjustment。
- Workflow Improvement。
- Capability Improvement Proposal。
- Regression Verification。
- Learning Record。

### 3.3.3 具體情境

**Investment**

Agent 曾預測某項市場變化，但實際結果與預期不同。

它應該分析：

- 當時使用了哪些資訊？
- 哪些資訊後來證明不可靠？
- 預測模型是否有系統性偏差？
- 當時風險評估是否合理？
- 是否需要調整研究方法？

不能單純因為投資獲利就認為策略正確，也不能因為虧損就認為策略錯誤。

**Coding**

Agent 修改程式後，測試發現原本的問題雖然解決，卻造成其他模組異常。

它必須重新分析修改範圍、相依關係及回歸測試結果。

**Writing**

Agent 完成一章後，發現新內容與前面章節的人物年齡、歷史事件或人物關係不一致。

它應重新檢查相關章節，而不是只修正最後寫的一段文字。

### 3.3.4 修正必須可驗證

每一次重要修正至少要保留：

- 原始問題。
- 原始狀態。
- 修正提案。
- 修正依據。
- 執行結果。
- 驗證結果。
- 是否改善。
- 是否需要回復。

詳細評估及回饋機制由 AIA-11 定義。

## 3.4 Goal Evolution — 目標演化

### 3.4.1 定義

Goal Evolution 是系統在維持原始方向的同時，根據新資訊、工作成果、環境變化及長期評估，判斷目標是否需要維持、最佳化或修訂的能力。

### 3.4.2 三種目標行為

**Goal Persistence**

維持仍然有效的長期目標。

例如：

- 投資實驗持續追求投資組合總資產最大化。
- 寫作專案持續完成既定小說。
- Coding 專案持續改善 AIA 的程式品質與功能。

**Goal Optimization**

不改變主要目標，但改善達成目標的方法。

例如：

- Investment 改善研究流程。
- Writing 改善歷史校勘方式。
- Coding 改善程式架構及自動化工具。

**Goal Revision**

當目標本身不再合理、互相衝突或環境已經改變時，提出目標修訂。

例如：

- 原始專案的技術依賴已經停止維護。
- 原定寫作架構無法容納後續新增的歷史材料。
- 投資實驗的資料來源已不足以支援原先的研究方法。

### 3.4.3 目標演化的治理要求

Goal Evolution 可以自主提出修訂，但必須區分：

- Agent 自行調整的工作方法。
- 不改變最終目標的策略最佳化。
- 會改變主要目標或資源配置的修訂。
- 涉及系統權限、安全或重大資源的修訂。

重大目標變更須依 AIA-14 的治理規則處理。

詳細目標模型由 AIA-09 定義。

# 4. Autonomy Core 邏輯架構

## 4.1 主要元件

Autonomy Core 應由以下邏輯元件構成：

| 元件 | 職責 |
|---|---|
| Autonomy Coordinator | 協調整個自主性循環 |
| Observation Engine | 收集及整理環境資訊 |
| Context Resolver | 重建相關記憶、目標及專案脈絡 |
| Assessment Engine | 評估目前狀態及問題 |
| Initiative Engine | 發現候選工作與機會 |
| Deliberation Engine | 分析候選方案及可能結果 |
| Decision Engine | 選擇行動或不行動 |
| Execution Dispatcher | 將決策轉換為受治理的工作 |
| Verification Engine | 檢查執行結果 |
| Learning Engine | 保存經驗及改善資訊 |
| Goal Evolution Engine | 評估目標維持、最佳化或修訂 |

上述為邏輯責任，不要求每一個元件都獨立成為 Go Package 或獨立 AI Agent。

實作時應避免為了架構完整而產生過度切分的 Service。

## 4.2 與既有 AIA 元件的關係

| 既有元件 | 與 Autonomy Core 的關係 |
|---|---|
| Runtime | 提供持續執行環境 |
| Worker Supervisor | 啟動與管理背景工作 |
| Event Bus | 提供狀態變化與事件通知 |
| Memory Service | 保存及檢索長期記憶 |
| Goal Service | 管理目標及其狀態 |
| Project Service | 管理長期專案 |
| Task Service | 管理具體工作 |
| Agent Registry | 提供 Agent 能力資訊 |
| Policy Service | 判斷行動是否符合治理規則 |
| Approval Service | 管理需要人類核准的行動 |
| Audit Service | 保存不可任意修改的稽核紀錄 |
| Scheduler | 執行已核准的 Task |
| Harness Adapter | 呼叫底層 Agent Harness |

Autonomy Core 不應重新建立一套與既有 Service 重複的資料管理系統。

# 5. Autonomy Cycle

## 5.1 核心循環

AIA 的自主性運作採用以下循環：

```text
Observe
   |
   v
Recall
   |
   v
Assess
   |
   v
Discover
   |
   v
Deliberate
   |
   v
Decide
   |
   v
Execute
   |
   v
Verify
   |
   v
Learn
   |
   v
Reassess
   |
   +----------> Observe
```

各階段定義如下。

## 5.2 Observe — 觀察

取得目前環境及系統狀態。

觀察來源包括：

- External Data。
- Market Data。
- News。
- Project State。
- Goal State。
- Task State。
- Agent Status。
- Event。
- Previous Evaluation。
- System Health。

觀察必須保留來源、時間、資料品質及可信度。

## 5.3 Recall — 回憶

取得與當前情境相關的歷史資料。

包括：

- 長期目標。
- 相關 Project。
- 先前決策。
- 過去工作。
- 過去錯誤。
- 修正紀錄。
- 已知限制。
- 尚未解決問題。

Recall 不應無限制載入全部記憶。

系統應依目標、任務、時間及關聯性選擇必要脈絡。

## 5.4 Assess — 評估

分析目前狀態與目標之間的差距。

例如：

- 目前工作是否符合目標？
- 有沒有重要工作逾期？
- 是否出現新的風險？
- 過去的決策是否需要重新檢查？
- 是否有可利用的新資訊？
- 現有能力是否足以完成工作？

Assessment 必須考慮資訊可信度及不確定性。

## 5.5 Discover — 發現

產生可能值得處理的工作及機會。

來源包括：

- 新事件。
- 目標差距。
- 未完成工作。
- 過去的錯誤。
- 預測與實際結果差異。
- 新的工具或技術。
- 其他 Domain Agent 的需求。
- 長期專案停滯。
- 能力不足。

Discover 可以產生多個候選行動，但不能直接將所有候選項目都建立成 Task。

## 5.6 Deliberate — 推理

對候選行動進行比較與評估。

至少考慮：

- 與主要目標的關聯。
- 預期效益。
- 可能風險。
- 執行成本。
- 所需資源。
- 執行能力。
- 時效性。
- 資料可信度。
- 是否有更好的替代方案。
- 是否應先取得更多資訊。

Deliberation 必須允許結論為資訊不足。

## 5.7 Decide — 決策

選擇一項行動、延後行動或不行動。

可能結果：

- EXECUTE：立即執行。
- DEFER：延後執行。
- REQUEST_APPROVAL：等待人類核准。
- GATHER_INFORMATION：先收集資料。
- DELEGATE：交由其他 Agent。
- NO_ACTION：目前不採取行動。
- REVISE_GOAL：提出目標修訂。
- ESCALATE：交由人類處理。

每個重要決策都應保留決策依據。

## 5.8 Execute — 執行

將決策轉換成可執行工作。

執行工作可能包括：

- 建立 Task。
- 派遣 Domain Agent。
- 呼叫工具。
- 收集資料。
- 更新 Project。
- 修改策略。
- 提出程式變更。
- 執行已核准的自我修改。

所有執行都必須經過 Policy 檢查。

## 5.9 Verify — 驗證

檢查執行結果是否符合預期。

驗證方式依領域不同：

- Investment：預測、交易紀錄、持股及投資組合結果。
- Coding：測試、程式差異、程式品質及回歸結果。
- Writing：史實、時間軸、人物關係、章節一致性及敘事要求。

驗證不能只依賴執行 Agent 自己宣稱成功。

應盡可能使用獨立資料、測試或評估流程。

## 5.10 Learn — 學習

將執行與驗證結果轉化為可供後續使用的經驗。

可能形成：

- 新增 Memory。
- 更新 Project State。
- 更新 Decision History。
- 更新 Evaluation History。
- 調整策略。
- 建立改善提案。
- 更新 Capability Assessment。
- 提出 Goal Revision。

Learning 不得直接覆寫歷史決策。

## 5.11 Reassess — 重新評估

重新檢查目前目標及工作狀態。

決定：

- 是否繼續原工作。
- 是否需要新增工作。
- 是否需要修正原有成果。
- 是否需要改變工作優先順序。
- 是否需要重新評估目標。
- 是否應該等待。

Reassess 完成後，系統可以進入下一輪循環。

# 6. Autonomy State

## 6.1 設計目的

Autonomy Core 必須保存自身的運作狀態，避免每次喚醒都從零開始。

Autonomy State 是系統目前自主性判斷的狀態快照，不應取代 Goal、Project、Task、Memory 的正式資料。

## 6.2 建議狀態內容

| 欄位 | 說明 |
|---|---|
| autonomy_session_id | 自主性循環識別碼 |
| started_at | 本輪開始時間 |
| last_cycle_at | 上一次循環時間 |
| active_goals | 目前有效目標 |
| active_projects | 目前關注專案 |
| pending_issues | 尚未解決問題 |
| candidate_actions | 候選行動 |
| active_decisions | 尚在執行的決策 |
| resource_state | 可用資源與預算 |
| autonomy_status | 自主性運作狀態 |
| next_evaluation_at | 預定重新評估時間 |

## 6.3 Autonomy Status

建議狀態：

- IDLE：目前沒有正在執行的自主性循環。
- OBSERVING：正在取得資訊。
- ASSESSING：正在評估狀態。
- DELIBERATING：正在分析候選方案。
- DECIDING：正在形成決策。
- EXECUTING：已有決策正在執行。
- VERIFYING：正在驗證結果。
- LEARNING：正在整理經驗。
- WAITING_FOR_APPROVAL：等待人類核准。
- WAITING：目前沒有需要立即執行的工作。
- PAUSED：自主性循環暫停。
- ERROR：發生需要處理的錯誤。

這些狀態是 Autonomy Core 的狀態，不應直接與 Task Status 混用。

# 7. Decision Model

## 7.1 決策基本資料

每個重要決策至少應保存：

| 欄位 | 說明 |
|---|---|
| decision_id | 唯一識別碼 |
| goal_id | 相關目標 |
| project_id | 相關專案 |
| domain | 所屬領域 |
| decision_type | 決策類型 |
| context_snapshot_id | 決策時的脈絡快照 |
| observations | 使用的觀察資訊 |
| alternatives | 評估過的替代方案 |
| selected_action | 選擇的行動 |
| rationale | 決策理由 |
| expected_outcome | 預期結果 |
| confidence | 信心程度 |
| risk_assessment | 風險評估 |
| policy_result | 治理檢查結果 |
| approval_status | 核准狀態 |
| created_at | 決策時間 |

實際欄位與資料庫 Schema 由後續資料設計及 AIA-03 實作規格決定。

## 7.2 信心程度

Agent 可以提出信心程度，但必須避免將語言模型自述的信心直接視為統計上的可靠度。

應逐步透過歷史預測與實際結果校準信心。

信心程度必須與資料品質、資訊完整性及風險分開記錄。

## 7.3 No-Action Decision

不行動必須是一種正式決策。

例如：

- 投資市場沒有值得承擔的交易機會。
- Writing Project 目前沒有需要修改的問題。
- Coding Project 的現有程式已符合需求。
- 新資訊尚不足以支持任何結論。
- 修改成本高於預期效益。
- 目前有更重要的工作。

No-Action Decision 應保存理由及下次重新評估的條件。

不能讓系統為了顯示自主性而不斷產生沒有價值的工作。

# 8. Action Candidate 與 Task 的責任區分

這是 Autonomy Core 與既有 Task Service 之間的重要邊界。

## 8.1 Action Candidate

Action Candidate 是系統發現的可能行動，尚未代表已經決定執行。

例如：

- 重新檢查台積電的財務資料。
- 修正小說第三卷的人物年齡。
- 重構 Coding Agent 的資料存取模組。
- 比較不同投資策略的歷史表現。

## 8.2 Decision

Decision 是 Autonomy Core 評估候選行動後形成的正式判斷。

它可以選擇：

- 執行。
- 延後。
- 收集更多資訊。
- 等待核准。
- 不執行。

## 8.3 Task

Task 是經過決策及治理檢查後，正式交給 Runtime 執行的工作單位。

因此：

```text
Observation
    |
    v
Action Candidate
    |
    v
Deliberation
    |
    v
Decision
    |
    +---- NO_ACTION
    |
    +---- DEFER
    |
    +---- REQUEST_APPROVAL
    |
    +---- EXECUTE
               |
               v
             Policy
               |
               v
             Task
               |
               v
          Domain Agent
```

Autonomy Core 不應把所有 Action Candidate 直接轉換成 Task。

# 9. Domain Agent Integration

## 9.1 共用自主性能力

所有 Domain Agent 都應使用同一套 Autonomy Core。

Domain Agent 只提供領域知識、工具、觀察資料及專業執行能力。

它們可以：

- 回報觀察結果。
- 提出問題。
- 提供候選行動。
- 提供專業評估。
- 執行已派遣工作。
- 回報結果。
- 提出能力改善需求。

## 9.2 Investment Agent

Investment Agent 的領域責任：

- 市場資料解讀。
- 投資研究。
- 持股分析。
- 投資組合決策。
- 預測及績效評估。
- 策略修正。

Autonomy Core 負責：

- 判斷是否值得重新研究。
- 維持長期投資目標。
- 決定是否需要新工作。
- 檢查投資決策的長期品質。
- 判斷是否提出策略或能力改善。

## 9.3 Coding Agent

Coding Agent 的領域責任：

- 程式碼分析。
- Bug 修正。
- 程式開發。
- 重構。
- 測試。
- 技術文件。
- 工具及能力開發。

Autonomy Core 負責：

- 維持 Coding Project 的長期目標。
- 發現未解決問題。
- 評估改善工作的優先順序。
- 決定是否需要跨領域支援。
- 驗證改善是否符合整體目標。

## 9.4 Writing Agent

Writing Agent 的領域責任：

- 小說架構。
- 人物設定。
- 時間軸。
- 歷史資料整理。
- 章節創作。
- 史實校勘。
- 劇情一致性檢查。

Autonomy Core 負責：

- 維持長期寫作目標。
- 追蹤各卷與章節進度。
- 發現待校勘問題。
- 評估是否需要重寫或調整架構。
- 決定是否向 Coding Agent 提出工具需求。

# 10. Cross-Domain Coordination

## 10.1 基本原則

跨領域工作不是由 Agent 任意互相呼叫。

它必須先形成明確的工作需求，再由 Autonomy Core 判斷是否需要委派。

## 10.2 範例：Writing → Coding

Writing Agent 發現小說管理工具缺少歷史年表一致性檢查。

流程：

1. Writing Agent 提出 Capability Gap。
2. Autonomy Core 評估需求。
3. 檢查 Coding Agent 是否具備相關能力。
4. 建立 Cross-Domain Action Candidate。
5. 評估優先順序與成本。
6. 決定是否委派。
7. 建立 Coding Task。
8. Coding Agent 開發工具。
9. 驗證工具功能。
10. Writing Agent 使用工具重新檢查小說。
11. Autonomy Core 評估整體改善效果。

## 10.3 範例：Investment → Coding

Investment Agent 發現資料分析工具無法正確處理某些市場資料。

流程：

1. Investment Agent 提出問題。
2. Autonomy Core 檢查是否屬於資料問題、策略問題或工具問題。
3. 如果屬於工具問題，委派 Coding Agent。
4. Coding Agent 修正或開發工具。
5. 執行測試及資料驗證。
6. Investment Agent 使用新工具重新分析。
7. 比較改善前後的研究品質。

## 10.4 跨領域工作的責任

跨領域委派不代表原始 Agent 可以卸除責任。

原始 Agent 必須：

- 說明需求。
- 提供必要背景。
- 定義交付條件。
- 驗證交付成果。
- 回報使用結果。

Autonomy Core 負責維護整體工作關係及決策紀錄。

詳細規格由 AIA-12 定義。

# 11. Autonomy 與 Self-Modification 的邊界

## 11.1 Autonomy Core 不直接修改自身執行環境

Autonomy Core 可以發現自身能力不足，提出改善方案。

但實際修改程式碼、工具或核心設定，應交由受治理的 Coding 工作流程處理。

例如：

1. Autonomy Core 發現 Context Resolver 無法正確重建長期專案狀態。
2. 提出改善需求。
3. Coding Agent 分析相關程式碼。
4. 建立修改方案。
5. 在隔離環境中實作。
6. 執行測試及驗證。
7. 依變更等級進行核准。
8. 部署新版本。
9. 監控實際效果。
10. 必要時回復。

## 11.2 自我修改不可改變治理邊界

下列功能不得由 Agent 自行解除或繞過：

- Policy Enforcement。
- Approval Requirements。
- Audit Integrity。
- Permission Boundaries。
- Financial Safety。
- Emergency Stop。
- Self-Modification Approval。

詳細規格由 AIA-13 與 AIA-14 定義。

# 12. 與 Runtime 的整合

## 12.1 Runtime 的責任

Runtime 提供：

- 常駐執行。
- Worker Lifecycle。
- Schedule Trigger。
- Task Dispatch。
- Event Processing。
- Recovery。
- Shutdown。

Runtime 不負責替 Autonomy Core 決定所有工作內容。

## 12.2 Autonomy Worker

Autonomy Core 可以透過 Autonomy Worker 定期或事件觸發運作。

觸發來源：

- Schedule。
- Project Updated。
- Task Completed。
- Evaluation Completed。
- New Data Available。
- Agent Capability Changed。
- Human Input。
- System Health Event。

## 12.3 喚醒不等於行動

每次 Autonomy Worker 被喚醒時，不代表必須產生新 Task。

可能結果：

- 完成一次狀態觀察。
- 更新 Context。
- 發現問題。
- 提出候選行動。
- 執行已核准的決策。
- 等待資料。
- 維持現有目標。
- 決定不行動。

## 12.4 執行時間與資源管理

Autonomy Worker 必須遵守：

- 單輪執行時間限制。
- LLM 呼叫預算。
- Token 預算。
- 工具呼叫預算。
- 同時執行數量限制。
- Retry 限制。
- Resource Budget。

不得因為自主循環持續運作而無限制消耗資源。

# 13. Memory、Goal、Project 與 Decision 的關係

Autonomy Core 必須使用既有 Core Services，不得把全部長期狀態塞進單一 Memory 文件。

| 資料類型 | 儲存責任 |
|---|---|
| Memory | 知識、經驗、歷史摘要及可檢索資訊 |
| Goal | 目標、期限、優先順序及目標狀態 |
| Project | 長期專案狀態、成果與工作脈絡 |
| Task | 具體可執行工作 |
| Decision | 決策依據、選擇及預期結果 |
| Evaluation | 執行結果與品質評估 |
| Event | 狀態變化與事件發生紀錄 |
| Audit | 重要操作及治理紀錄 |
| Capability | Agent、工具與系統能力資訊 |

這些資料之間應透過識別碼及關聯關係互相連結。

詳細的 Continuity 與 Project Persistence 由 AIA-10 定義。

# 14. Error Handling 與 Uncertainty

## 14.1 資訊不足

當觀察資料不足時，Autonomy Core 不應自行補造事實。

可以：

- 要求補充資料。
- 派遣資料收集工作。
- 降低決策信心。
- 延後決策。
- 選擇不行動。

## 14.2 互相矛盾的資訊

當不同資料來源或 Memory 出現矛盾時：

1. 保留原始來源。
2. 比較資料時間。
3. 比較來源可信度。
4. 標記衝突。
5. 必要時啟動驗證工作。
6. 不得靜默覆寫舊資料。

## 14.3 Agent 執行失敗

當 Domain Agent 執行失敗時：

- 保存錯誤資訊。
- 更新 Task 狀態。
- 判斷是否重試。
- 評估是否需要其他 Agent。
- 判斷是否需要人類介入。
- 將失敗原因帶入下一輪評估。

## 14.4 自主循環中斷

Runtime 中斷時，Autonomy Core 必須能夠在重新啟動後恢復必要狀態。

不得假設所有自主性循環都能在單一 Process 或 Session 中完成。

# 15. Observability 與自主性紀錄

Autonomy Core 必須能夠回答：

- 為什麼開始這一輪自主性循環？
- 使用了哪些資訊？
- 找到了哪些問題？
- 為什麼選擇某項工作？
- 為什麼沒有採取行動？
- 委派給哪個 Agent？
- 執行結果如何？
- 是否達成預期？
- 是否產生修正？
- 修正是否有效？
- 是否影響其他 Project？
- 是否改變了原有目標？

## 15.1 核心紀錄

至少保存：

- Autonomy Cycle Record。
- Observation Record。
- Context Snapshot。
- Action Candidate Record。
- Deliberation Record。
- Decision Record。
- Execution Link。
- Verification Record。
- Learning Record。
- Goal Change Record。

## 15.2 不可覆寫歷史

重要決策及評估紀錄應採追加式保存。

新結論可以修正目前狀態，但不得抹除過去當時的決策依據。

這對投資實驗、程式修改及歷史小說校勘都非常重要。

# 16. 自主性評估指標

本文件定義初步指標，正式測量方法由 AIA-15 制定。

| 能力 | 指標 | 評估方向 |
|---|---|---|
| Initiative | Independent Discovery Rate | 主動發現有價值問題的能力 |
| Initiative | Action Relevance | 候選行動與目標的關聯性 |
| Continuity | Context Reconstruction Accuracy | 重建歷史脈絡的正確性 |
| Continuity | Project Resumption Accuracy | 恢復長期專案的正確性 |
| Self-Correction | Error Detection Rate | 發現實際問題的能力 |
| Self-Correction | Correction Effectiveness | 修正是否真正改善結果 |
| Goal Evolution | Goal Persistence | 是否維持有效長期目標 |
| Goal Evolution | Goal Revision Quality | 目標修訂是否合理 |
| Cross-Domain | Handoff Success Rate | 跨領域交付是否有效 |
| Overall | Autonomous Outcome Quality | 自主行動對最終成果的貢獻 |

指標不得只看數量。

例如 Independent Discovery Rate 不應鼓勵系統大量提出低價值工作。

# 17. 實作架構建議

## 17.1 Go Package 建議

Autonomy Core 應先以模組化方式實作，不必一開始拆成多個獨立 Process。

建議邏輯結構：

```text
internal/
  autonomy/
    coordinator/
    observation/
    context/
    assessment/
    initiative/
    deliberation/
    decision/
    execution/
    verification/
    learning/
    goal/
    model/
    repository/
```

以上是建議的邏輯責任劃分，不代表目前 Go 專案已經存在這些 Package。

實際 Package 命名與依賴方向應在 AIA-03 修訂時確認。

## 17.2 核心介面概念

建議定義下列抽象介面：

```go
type AutonomyCoordinator interface {
    RunCycle(
        ctx context.Context,
        input CycleInput,
    ) (*CycleResult, error)
}

type Observer interface {
    Observe(
        ctx context.Context,
        scope ObservationScope,
    ) (*ObservationResult, error)
}

type ContextResolver interface {
    Resolve(
        ctx context.Context,
        input ContextRequest,
    ) (*ResolvedContext, error)
}

type Assessor interface {
    Assess(
        ctx context.Context,
        input AssessmentInput,
    ) (*AssessmentResult, error)
}

type InitiativeEngine interface {
    Discover(
        ctx context.Context,
        input DiscoveryInput,
    ) ([]ActionCandidate, error)
}

type Deliberator interface {
    Deliberate(
        ctx context.Context,
        input DeliberationInput,
    ) (*DeliberationResult, error)
}

type DecisionEngine interface {
    Decide(
        ctx context.Context,
        input DecisionInput,
    ) (*DecisionResult, error)
}

type ExecutionDispatcher interface {
    Dispatch(
        ctx context.Context,
        decision Decision,
    ) (*ExecutionResult, error)
}

type Verifier interface {
    Verify(
        ctx context.Context,
        input VerificationInput,
    ) (*VerificationResult, error)
}

type Learner interface {
    Learn(
        ctx context.Context,
        input LearningInput,
    ) (*LearningResult, error)
}
```

上述介面是設計示意，正式型別、錯誤分類及 Repository 契約由 AIA-03 定義。

## 17.3 依賴方向

Autonomy Core 可以依賴 Core Services 的抽象介面。

Core Services 不應反向依賴某一個具體 Domain Agent 的實作。

Domain Agent 不應直接繞過 Autonomy Core 建立未經治理的跨領域工作。

## 17.4 模型與推理能力

Autonomy Core 不應假設所有判斷都由同一個 LLM 完成。

不同工作可以採用：

- 規則引擎。
- 傳統程式邏輯。
- 統計模型。
- LLM 推理。
- 外部工具。
- Domain-specific Evaluation。

LLM 是推理能力的提供者之一，不等於整個 Autonomy Core。

# 18. 目前實作狀態與差距

本節區分既有平台能力與尚待建置的自主性能力。

**注意：目前實作狀態以已提供的 Go 程式碼版本及既有規格分析為依據，不代表尚未提供的更新程式碼不存在相關功能。**

| 功能 | 已知狀態 | 說明 |
|---|---|---|
| Go Application Bootstrap | Implemented | 已有 App 初始化與依賴組裝 |
| Agent Registry | Partially Implemented | 已有 Agent 註冊及介面基礎 |
| Task Scheduler | Partially Implemented | 已有 Task 執行機制 |
| Schedule Service | Implemented | 已有排程管理及 Task 建立能力 |
| Background Worker | Version Verification Required | 已有後續更新版本，須以最新程式碼確認 |
| Memory Service | Partially Implemented | 有記憶管理基礎，但不等於完整 Continuity |
| Goal Service | Version Verification Required | 依目前架構文件確認實際功能 |
| Project Continuity | Planned | 需完整定義長期專案狀態與恢復能力 |
| Initiative Engine | Planned | 尚不能因為有 Schedule 就宣稱完成 |
| Deliberation Engine | Planned | 需建立候選方案比較及決策流程 |
| Decision History | Partially Implemented / Verification Required | 需確認完整決策脈絡與快照 |
| Self-Correction Engine | Planned | 有部分領域評估服務，不等於共用自我修正 |
| Goal Evolution Engine | Planned | 需建立目標維持、最佳化及修訂能力 |
| Cross-Domain Evolution | Planned | 需建立跨領域工作協調 |
| Self-Modification | Planned | 需建立隔離開發、驗證及發布流程 |
| Autonomy Evaluation | Planned | 需建立長期實驗與評估制度 |

已存在的投資預測評估、程式執行或小說文件管理功能，不能直接視為共用 Autonomy Core 已完成。

# 19. 實作階段

## A0 — Autonomy Foundation

目標：建立自主性循環的基本資料模型及協調入口。

工作：

- Autonomy State。
- Cycle Record。
- Observation Model。
- Context Model。
- Decision Model。
- Action Candidate Model。
- Autonomy Coordinator。
- Cycle Persistence。

驗收：

- 可以啟動一輪自主性循環。
- 可以保存每個階段的狀態。
- 中斷後能夠恢復必要脈絡。
- 尚未具備完整主動性時，不得宣稱自主性已完成。

## A1 — Continuity Integration

目標：將 Memory、Goal、Project、Task 及 Decision 整合。

工作：

- Context Resolver。
- Goal Context。
- Project Context。
- Decision History。
- Work History。
- Pending Issues。
- Context Reconstruction。

驗收：

- 可以恢復跨 Session 的專案狀態。
- 可以正確取得歷史決策與未完成工作。
- 可以切換不同 Project 而不混淆脈絡。

詳細設計由 AIA-10 定義。

## A2 — Initiative Integration

目標：從被動接受指令轉為主動發現工作。

工作：

- Observation Engine。
- Assessment Engine。
- Initiative Engine。
- Candidate Ranking。
- No-Action Decision。
- Initiative Proposal。

驗收：

- 可以根據專案狀態發現問題。
- 可以提出與目標相關的候選工作。
- 不會為了產生 Task 而強制製造工作。

詳細設計由 AIA-09 定義。

## A3 — Self-Correction Integration

目標：建立可驗證的回饋循環。

工作：

- Expected Outcome。
- Actual Outcome。
- Evaluation。
- Error Classification。
- Root Cause Analysis。
- Correction Proposal。
- Regression Verification。
- Learning Record。

驗收：

- 能識別執行結果與預期的差異。
- 能提出具體修正。
- 能比較修正前後的結果。
- 能記錄無效修正及失敗經驗。

詳細設計由 AIA-11 定義。

## A4 — Domain Autonomy

目標：讓各 Domain Agent 使用共用 Autonomy Core。

工作：

- Investment Integration。
- Coding Integration。
- Writing Integration。
- Domain Observation。
- Domain Evaluation。
- Domain-specific Action。

驗收：

- 三個領域都能維持長期工作狀態。
- 能自行發現領域內問題。
- 能提出及執行適當工作。
- 能回報結果並進入下一輪評估。

## A5 — Cross-Domain Evolution

目標：建立跨領域工作及能力改善。

工作：

- Capability Registry。
- Cross-Domain Request。
- Agent Handoff。
- Shared Project Context。
- Result Integration。
- Capability Evaluation。

驗收：

- Writing 能向 Coding 提出工具需求。
- Investment 能向 Coding 提出分析能力需求。
- Coding 能交付可供其他領域使用的成果。
- 原始領域能驗證改善是否有效。

詳細設計由 AIA-12 定義。

## A6 — Autonomous Improvement

目標：建立受控的自我修改能力。

工作：

- Capability Gap Detection。
- Modification Proposal。
- Isolated Workspace。
- Automated Verification。
- Approval Workflow。
- Deployment。
- Monitoring。
- Rollback。

驗收：

- 能發現自身能力不足。
- 能產生可檢查的修改方案。
- 能在隔離環境執行修改。
- 能驗證修改結果。
- 能依治理規則發布或等待核准。
- 能在失敗時回復。

詳細設計由 AIA-13、AIA-14 定義。

# 20. Acceptance Criteria

Autonomy Core 至少必須通過以下驗收。

## 20.1 基礎能力

- [ ] 能建立並保存自主性循環。
- [ ] 能整合 Observation、Memory、Goal、Project 及 Task。
- [ ] 能重建相關工作脈絡。
- [ ] 能保存重要決策及其依據。
- [ ] 能恢復中斷的自主性循環。

## 20.2 主動性

- [ ] 能從非使用者直接指令的事件發現問題。
- [ ] 能根據長期目標提出候選工作。
- [ ] 能評估候選工作的重要性。
- [ ] 能決定延後或不行動。
- [ ] 不會將 Schedule 觸發直接等同 Initiative。

## 20.3 時間連續性

- [ ] 能跨 Session 恢復工作脈絡。
- [ ] 能維持多個 Project 的獨立狀態。
- [ ] 能保留歷史決策。
- [ ] 能追蹤未完成工作。
- [ ] 能在不同 Domain 之間切換而不遺失狀態。

## 20.4 自我修正

- [ ] 能比較預期與實際結果。
- [ ] 能發現問題。
- [ ] 能分析問題原因。
- [ ] 能提出修正。
- [ ] 能驗證修正效果。
- [ ] 能保存修正經驗。

## 20.5 目標演化

- [ ] 能維持有效目標。
- [ ] 能提出目標最佳化建議。
- [ ] 能識別目標衝突。
- [ ] 能提出目標修訂。
- [ ] 重大目標變更受到治理。

## 20.6 治理

- [ ] 所有執行工作均經過 Policy 檢查。
- [ ] 需要核准的工作不會自行繞過 Approval。
- [ ] Audit 紀錄不可由 Agent 任意修改。
- [ ] 自我修改受到隔離及發布治理。
- [ ] 系統可以暫停、停止及恢復。

# 21. 與其他規格文件的責任邊界

| 文件 | 與 AIA-08 的關係 |
|---|---|
| AIA-01 | 定義產品層級自主性需求 |
| AIA-02 | 定義 Autonomy Core 在整體架構的位置 |
| AIA-03 | 定義 Go Runtime 的實作方式 |
| AIA-04 | 定義自主性測試及驗收方法 |
| AIA-09 | 深入定義 Initiative 與 Goal Evolution |
| AIA-10 | 深入定義 Continuity 與 Project Persistence |
| AIA-11 | 深入定義 Self-Correction 與 Evaluation |
| AIA-12 | 深入定義 Cross-Domain Collaboration |
| AIA-13 | 深入定義 Self-Modification |
| AIA-14 | 深入定義 Autonomous Governance |
| AIA-15 | 深入定義自主性實驗及量測方法 |

AIA-08 是共用核心規格，不應重複定義上述文件的全部細節。

# 22. 最終設計原則

Autonomy Core 的存在，是為了讓 AIA 不再只是依照人類指令執行工作。

它必須能夠：

- 維持長期目標。
- 觀察外部環境。
- 理解自身目前狀態。
- 從歷史經驗中取得資訊。
- 主動發現值得處理的問題。
- 比較不同可能行動。
- 決定是否執行。
- 驗證工作成果。
- 修正錯誤。
- 重新評估目標。
- 將改善經驗帶入下一輪工作。

Investment、Coding、Writing 是三個不同的工作場域，但 Autonomy Core 應該是同一個。

AIA 可以上午研究投資，下午修改程式，晚上重新檢查小說。

重要的不是它能不能在一天內執行這三種工作，而是它在切換工作之後，仍然知道每個專案的目標、歷史、進度及下一步。

**Autonomy Core 的最終目的，是使 AIA 成為一個能夠持續理解自己正在追求什麼、主動改善工作成果，並在不同領域之間累積能力的自主系統。**
