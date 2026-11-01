# AIA-00 — 規格文件總體架構與演進藍圖

**Autonomous Intelligence Architecture**  
**Version:** 2.0  
**Document Type:** Master Specification Index & Evolution Blueprint  
**Status:** Proposed Baseline

---

# 1. 文件目的

本文件定義 AIA 全體規格文件的組織方式、責任邊界、相依關係、開發順序與驗收方向。

AIA 的核心目標不是單純建立一個可以呼叫多個 Agent 的任務管理平台，而是建立一套能夠長期持續運作，具備主動性思維、時間連續性、自我修正與目標演化能力的個人 AI 系統。

AIA 必須能夠：

1. 持續追蹤人類設定的長期目標。
2. 根據環境變化與既有知識主動發現值得處理的問題。
3. 自行規劃、執行及驗證工作。
4. 保留跨時間的工作狀態、決策歷史與經驗。
5. 發現自身成果或能力的不足，並提出修正。
6. 在不同工作領域之間共享經驗與改善能力。
7. 在受控範圍內修改自己的程式、工具、策略與工作流程。
8. 透過持續的回饋循環，逐步提升整體工作能力。

本文件所稱的「自主性」及「演化」，均以可觀察、可量測、可追溯的系統行為為判斷基礎，不預設或宣稱系統具有主觀意識。

# 2. AIA 核心定位

## 2.1 核心定義

AIA 是一套具有持續運作能力的個人自主 AI 平台。

它以 Autonomy Core 為核心，以 Domain Agents 為工作能力載體，以 Runtime 為持續執行環境，透過 Memory、Project、Goal、Task、Evaluation 與 Audit 等服務維持長期狀態。

AIA 不應被限制為等待人類提出問題後才開始工作的工具。

它應能在既定治理規則內，自行判斷：

- 現在正在處理什麼？
- 目前有哪些尚未完成的目標？
- 有哪些新的資訊值得注意？
- 哪些工作已經停滯？
- 哪些成果需要重新檢查？
- 哪些能力需要改善？
- 下一步最值得投入的工作是什麼？
- 是否應該暫停或等待？

## 2.2 四項自主性核心能力

| 能力 | 英文 | 定義 |
|---|---|---|
| 主動性思維 | Initiative | 主動觀察環境、發現問題、提出工作與行動 |
| 時間連續性 | Continuity | 維持跨時間的目標、狀態、記憶、工作與決策脈絡 |
| 自我修正 | Self-Correction | 評估成果、發現錯誤、分析原因並改善行為 |
| 目標演化 | Goal Evolution | 維持、最佳化或修訂目標，並管理目標之間的關係 |

四項能力共同構成 Autonomy Core，不應分散實作成 Investment、Coding、Writing 三套彼此獨立的自主性機制。

## 2.3 系統能力的三個層次

AIA 必須區分：

**知識成長（Knowledge Growth）**

系統累積更多資料、知識、歷史紀錄與經驗。

**能力成長（Capability Growth）**

系統能執行更多工作，或以更高品質、更低成本完成既有工作。

**自主性成長（Autonomy Growth）**

系統能自行發現知識與能力不足，決定改善方向，執行改善並驗證效果。

三者不能互相替代。增加記憶量不代表能力提升，增加工具數量也不代表自主性提升。

# 3. 整體系統架構

AIA 的整體架構分為六個主要層次。

## 3.1 Runtime Layer

負責持續執行及管理系統生命週期。

包含：

- Runtime Manager
- Worker Supervisor
- Schedule Worker
- Task Worker
- Recovery Worker
- Event Processing
- Graceful Shutdown

Runtime 提供持續執行環境，但不直接決定系統應追求什麼目標。

## 3.2 Autonomy Core Layer

負責自主性判斷與決策。

包含：

- Initiative Engine
- Continuity Engine
- Self-Correction Engine
- Goal Evolution Engine
- Deliberation & Decision
- Cross-Domain Reasoning

Autonomy Core 使用 Core Services 提供的狀態、歷史與治理資訊，決定是否採取行動。

## 3.3 Core Services Layer

負責保存系統長期狀態與提供共用能力。

包含：

- Agent Registry
- Memory Service
- Goal Service
- Project Service
- Task Service
- Event Bus
- Policy Service
- Approval Service
- Audit Service
- Evaluation Service
- Capability Registry

## 3.4 Domain Agent Layer

負責特定領域的專業工作。

第一階段規劃三個 Domain Agent：

| Agent | 工作範圍 | 主要實驗目的 |
|---|---|---|
| Investment Agent | 投資研究與虛擬投資 | 觀察、決策、預測、績效評估與策略修正 |
| Coding Agent | 程式開發與維護 | 除錯、重構、測試、工具開發與自我改善 |
| Writing Agent | 小說及長期寫作專案 | 長期創作、史實校勘、結構維護與成果修正 |

第一階段只有 Investment Agent 已有既存實作基礎；Coding 與 Writing 的自主性實驗能力仍須逐步建置。

## 3.5 Harness Layer

負責實際 AI Agent 執行環境。

目前以 Antigravity CLI（agy）作為底層 Agent Harness。

AIA 負責：

- 長期目標與工作狀態。
- Task 生命週期。
- Agent 組織與派遣。
- 治理與稽核。
- 長期記憶及自主性決策。

agy 負責：

- AI 模型互動。
- 工具執行。
- Session 管理。
- 工具權限。
- Sandbox 與底層執行控制。

## 3.6 Interface Layer

提供人類觀察與介入系統的介面。

包含：

- CLI
- Embedded Web UI
- Agent Interface
- Project Dashboard
- Autonomy Dashboard
- Evaluation Reports
- Approval Interface

人類可以觀察、核准、拒絕、暫停或調整系統行為，但不必成為每一項日常工作的派遣者。

# 4. 規格文件總表

AIA 規格文件分成三個群組。

## 4.1 第一群：平台與執行基礎

| 文件 | 名稱 | 主要責任 |
|---|---|---|
| AIA-00 | Specification Master Blueprint | 文件總體架構、責任與相依關係 |
| AIA-01 | Product Requirements Specification | 產品定位、需求、範圍與產品限制 |
| AIA-02 | System Architecture Specification | 系統整體架構、元件與資料流 |
| AIA-03 | Go Runtime Implementation Specification | Go Runtime、Worker、Service 與程式套件 |
| AIA-04 | Test Plan, Acceptance & Release Specification | 測試、驗收、發布與品質管理 |
| AIA-05 | Investment Agent Specification | Investment Domain Agent |
| AIA-06 | Embedded Web UI & Agent Interface Specification | Web UI 與 Agent 介面 |
| AIA-07 | User Manual & Operational Usage Specification | 使用、部署、操作與維護 |

這一群文件建立 AIA 可以運作的基礎。

## 4.2 第二群：自主性核心

| 文件 | 名稱 | 主要責任 |
|---|---|---|
| AIA-08 | Autonomy Core Specification | 四項自主性能力及其協作架構 |
| AIA-09 | Initiative & Goal Evolution Specification | 主動性、候選工作與目標演化 |
| AIA-10 | Continuity & Long-term Project Management Specification | 時間連續性、長期記憶與專案持續性 |
| AIA-11 | Self-Correction & Evaluation Specification | 評估、錯誤分析、修正與經驗回饋 |

這一群文件回答 AIA 如何從被動執行任務，轉變為能持續管理目標與工作的系統。

## 4.3 第三群：跨領域演化與自主治理

| 文件 | 名稱 | 主要責任 |
|---|---|---|
| AIA-12 | Cross-Domain Collaboration & Evolution Specification | 跨領域協作、能力共享與工作轉派 |
| AIA-13 | Self-Modification & Capability Evolution Specification | 程式修改、自我改善與能力演化 |
| AIA-14 | Autonomous Operation & Governance Specification | 自主運作、權限、風險與人類監督 |
| AIA-15 | Autonomy Evaluation & Experiment Protocol | 自主性實驗、指標、基準與長期驗證 |

這一群文件回答 AIA 如何透過不同領域的工作互相改善，以及如何安全地改變自身。

# 5. 各文件詳細責任與範圍

## 5.1 AIA-01：Product Requirements

定義 AIA 的產品需求與不可違反的基本限制。

必須涵蓋：

- 個人自主 AI 平台定位。
- 四項自主性能力。
- 長期目標與持續運作需求。
- Domain Agent 的角色。
- 使用者介入方式。
- Single User。
- Local First。
- 安全及權限治理。
- 不自動操作真實金融帳戶。
- 目標架構與目前實作狀態的區別。

AIA-01 是全體規格文件的產品需求依據。

## 5.2 AIA-02：System Architecture

定義整體架構與各元件之間的責任邊界。

必須涵蓋：

- Runtime。
- Autonomy Core。
- Core Services。
- Domain Agents。
- Harness Integration。
- Event Architecture。
- Data Architecture。
- Execution Architecture。
- Security Architecture。
- Deployment Architecture。
- Observability。
- Extensibility。

AIA-02 不負責定義每個 Go 函式的實作方式。

## 5.3 AIA-03：Go Runtime Implementation

將 AIA-02 轉化為可實作的 Go 系統規格。

必須涵蓋：

- Package Responsibility。
- Application Bootstrap。
- Dependency Wiring。
- Runtime Lifecycle。
- Worker Supervisor。
- Schedule Worker。
- Task Worker。
- Recovery。
- Event Processing。
- Repository。
- SQLite。
- Transaction。
- Concurrency。
- Harness Adapter。
- Graceful Shutdown。

必須清楚區分已實作、尚未同步到分析版本及尚未實作的功能。

## 5.4 AIA-04：Test Plan, Acceptance & Release

定義系統如何證明自己符合規格。

必須涵蓋：

- Unit Test。
- Integration Test。
- Runtime Test。
- Worker Test。
- Recovery Test。
- Security Test。
- Agent Harness Test。
- Domain Agent Test。
- Autonomy Behavior Test。
- Regression Test。
- Release Gate。
- Rollback。

除了功能測試，也必須測試自主行為是否符合預期。

## 5.5 AIA-05：Investment Agent

定義第一個完整自主性實驗場景。

初始實驗條件：

- NT$100,000 虛擬資金。
- 台灣證券市場。
- 特別關注台積電 2330。
- 每個證券營業日觀察與決策。
- 允許持有現金及不交易。
- 自主決定標的、持股及資金配置。
- 主要目標為最大化投資組合總資產。
- 每月 1 日檢查總資產與自主性表現。

必須涵蓋：

- Market Data。
- News & Announcements。
- Data Quality。
- Portfolio。
- Ledger。
- Virtual Trading。
- Decision。
- Prediction。
- Calibration。
- Strategy Evolution。
- Investment Memory。
- Risk Evaluation。

投資績效不能作為唯一自主性評估指標。

## 5.6 AIA-06：Web UI & Agent Interface

定義使用者與 Agent 的互動介面。

除既有功能外，後續應逐步支援：

- 長期目標檢視。
- Project 狀態。
- Pending Issues。
- Agent 主動提出的工作。
- 跨領域工作關係。
- 自我修正紀錄。
- 能力變更紀錄。
- Approval。
- Autonomy Dashboard。

## 5.7 AIA-07：User Manual

定義人類如何使用與管理 AIA。

包含：

- 安裝與啟動。
- Agent 管理。
- Project 管理。
- Goal 管理。
- Task 管理。
- Approval。
- Runtime 狀態。
- 自主運作監督。
- Recovery。
- 停止與恢復。
- 日常維護。

## 5.8 AIA-08：Autonomy Core

這是自主性核心的總體規格。

定義四項能力如何形成共同的自主決策系統。

必須涵蓋：

- Autonomy State。
- Observation。
- Context Reconstruction。
- Recall。
- Assessment。
- Discovery。
- Deliberation。
- Decision。
- Execution。
- Verification。
- Learning。
- Reassessment。

核心循環：

Observe → Recall → Assess → Discover → Deliberate → Decide → Execute → Verify → Learn → Reassess。

AIA-08 定義各能力的協作介面，不取代 AIA-09～11 的細部規格。

## 5.9 AIA-09：Initiative & Goal Evolution

定義系統如何主動發現值得做的事，以及如何管理目標演化。

必須涵蓋：

- Environment Observation。
- Opportunity Detection。
- Problem Discovery。
- Initiative Proposal。
- Goal Creation。
- Goal Persistence。
- Goal Optimization。
- Goal Revision。
- Goal Conflict。
- Priority Evaluation。
- Resource Allocation。
- No-Action Decision。

必須避免把排程觸發、定時喚醒或自動產生 Task 誤認為真正的主動性。

## 5.10 AIA-10：Continuity & Long-term Project Management

定義跨時間持續性的資料與行為機制。

必須涵蓋：

- Persistent Context。
- Long-term Memory。
- Project State。
- Goal History。
- Decision History。
- Work History。
- Evaluation History。
- Pending Issues。
- Next Action Candidates。
- Context Reconstruction。
- Project Resumption。
- Cross-Session Continuity。

其中 Project Persistence 是重要能力。

例如 Writing Agent 必須能夠持續掌握一部小說的卷數、章節進度、人物關係、歷史時間軸、待校勘事項及未完成問題。

即使暫時轉去處理程式或投資工作，也不能遺失原本的專案脈絡。

## 5.11 AIA-11：Self-Correction & Evaluation

定義系統如何判斷成果是否有效，以及如何修正自身行為。

必須涵蓋：

- Expected Outcome。
- Actual Outcome。
- Evaluation Criteria。
- Prediction Evaluation。
- Error Classification。
- Root Cause Analysis。
- Feedback Generation。
- Strategy Adjustment。
- Workflow Improvement。
- Regression Verification。
- Learning Record。

必須區分：

- 結果好壞。
- 當時決策品質。
- 預測準確度。
- 長期策略有效性。

不能因為一次投資獲利就判斷策略正確，也不能因為一次虧損就判斷策略錯誤。

## 5.12 AIA-12：Cross-Domain Collaboration & Evolution

定義不同 Domain Agent 如何互相提出工作與改善能力。

必須涵蓋：

- Cross-Domain Request。
- Capability Discovery。
- Agent-to-Agent Handoff。
- Shared Project。
- Shared Knowledge。
- Dependency Management。
- Cross-Domain Task。
- Result Integration。
- Cross-Domain Evaluation。

主要情境：

**Writing → Coding**

Writing Agent 發現小說管理工具不足，要求 Coding Agent 開發人物關係與史實校勘工具。

**Coding → Writing**

Coding Agent 完成新工具後，Writing Agent 使用工具重新檢查既有小說成果。

**Investment → Coding**

Investment Agent 發現資料分析或預測能力不足，要求 Coding Agent 改善相關演算法與工具。

**Coding → Investment**

Investment Agent 使用改善後的能力重新分析市場與投資組合。

這些工作必須具有明確的來源、需求、交付成果與驗證紀錄。

## 5.13 AIA-13：Self-Modification & Capability Evolution

定義 AIA 如何改善自己的程式與能力。

必須區分三個層級：

**Level 1：Behavior Modification**

- 工作策略。
- 分析參數。
- Prompt。
- Workflow。
- Agent Configuration。

**Level 2：Capability Modification**

- 工具。
- Domain Module。
- 新增演算法。
- 新增 Provider。
- Agent 功能擴充。

**Level 3：Core System Modification**

- Runtime。
- Autonomy Core。
- 核心 Service。
- Policy。
- Security。
- Approval。
- Audit。

Level 1 與 Level 2 可依授權範圍自主提出、修改及驗證。

Level 3 必須採取更嚴格的變更管理與人類核准。

自我修改流程至少包括：

Proposal → Isolated Workspace → Implementation → Test → Review → Approval → Deployment → Monitoring → Rollback。

AIA 不得自行修改或解除保護自身的治理機制。

## 5.14 AIA-14：Autonomous Operation & Governance

定義自主運作的治理與安全機制。

必須涵蓋：

- Autonomy Boundary。
- Human Oversight。
- Permission Boundary。
- Approval Policy。
- Resource Budget。
- Execution Limit。
- Data Access Policy。
- Financial Safety。
- Self-Modification Governance。
- Emergency Stop。
- Recovery。
- Audit Integrity。

特別要求：

- Agent 不可自行擴張權限。
- Agent 不可自行解除 Approval。
- Agent 不可修改或刪除歷史 Audit。
- Agent 不可自行取得真實金融交易權限。
- Agent 不可透過自我修改繞過安全限制。

## 5.15 AIA-15：Autonomy Evaluation & Experiment Protocol

定義如何驗證 AIA 是否真的具備自主性。

必須涵蓋：

- Experimental Design。
- Baseline。
- Observation Period。
- Behavior Metrics。
- Autonomy Metrics。
- Continuity Metrics。
- Self-Correction Metrics。
- Goal Evolution Metrics。
- Cross-Domain Metrics。
- Capability Growth Metrics。
- Regression。
- Longitudinal Evaluation。

應建立可重複實驗，不以單次成功案例作為能力證明。

# 6. 文件相依關係

規格文件的邏輯相依關係如下：

```text
AIA-00
  |
  +-- AIA-01 Product Requirements
  |      |
  |      +-- AIA-02 System Architecture
  |              |
  |              +-- AIA-03 Go Runtime
  |              +-- AIA-05 Investment Agent
  |              +-- AIA-06 Web UI
  |              +-- AIA-07 User Manual
  |
  +-- AIA-04 Test & Acceptance
  |
  +-- AIA-08 Autonomy Core
         |
         +-- AIA-09 Initiative & Goal Evolution
         |
         +-- AIA-10 Continuity & Project Persistence
         |
         +-- AIA-11 Self-Correction & Evaluation
                |
                +-- AIA-12 Cross-Domain Evolution
                |
                +-- AIA-13 Self-Modification
                |
                +-- AIA-14 Autonomous Governance
                |
                +-- AIA-15 Autonomy Experiments
```

以上是概念依賴，不代表開發必須完全依照樹狀順序進行。

例如 AIA-14 的安全治理原則必須在 AIA-13 自我修改功能實作前就完成設計；AIA-04 也必須隨各項功能規格同步更新。

# 7. Domain Agent 演進規劃

## 7.1 第一階段：Investment

Investment 是第一個自主性實驗場景。

目標是讓 AIA 能夠：

- 自主觀察市場。
- 自主產生投資判斷。
- 自主決定交易或不交易。
- 保存預測與決策依據。
- 評估投資結果。
- 修正投資策略。

## 7.2 第二階段：Coding

Coding Agent 必須能夠：

- 管理長期軟體專案。
- 維護程式架構與技術文件。
- 自行發現 Bug。
- 自行規劃改善工作。
- 修改及驗證程式碼。
- 評估修改前後的品質。
- 將成功的改善經驗累積到長期記憶。

## 7.3 第三階段：Writing

Writing Agent 必須能夠：

- 維護長期創作專案。
- 管理世界觀、人物、時間軸及章節。
- 檢查歷史資料與劇情一致性。
- 發現敘事結構問題。
- 自行提出修改方向。
- 根據校勘結果修正既有內容。
- 追蹤每卷及整部作品的完成度。

## 7.4 第四階段：跨領域演化

三個 Agent 不再只處理各自的工作。

AIA 可以建立跨領域工作：

- Writing 提出工具需求給 Coding。
- Investment 提出分析工具需求給 Coding。
- Coding 提供新工具給 Writing 與 Investment。
- 各領域重新檢查既有成果。
- Autonomy Core 比較改善前後的結果。

這是 AIA 從多 Agent 平台邁向跨領域自主系統的重要階段。

# 8. 全域開發階段

開發階段不等於文件編號，必須依能力依賴關係安排。

| 階段 | 名稱 | 核心成果 |
|---|---|---|
| R1 | Persistent Runtime | Runtime、Worker、Recovery 穩定運作 |
| R2 | Continuity Foundation | 長期記憶、Goal、Project、Decision History |
| R3 | Initiative Engine | 主動發現問題與產生工作 |
| R4 | Reasoning & Self-Correction | 評估、反省、修正與目標演化 |
| R5 | Autonomous Domain Experiments | Investment、Coding、Writing 自主工作 |
| R6 | Cross-Domain Evolution | 跨領域協作、自我修改與能力演化 |

R1～R6 是全域能力里程碑，不應將它們誤認為每個功能都必須等到前一階段完全結束才開始研究。

安全治理與實驗評估必須貫穿所有階段。

# 9. 文件撰寫與維護原則

## 9.1 單一責任原則

每份文件應有明確責任，不應重複定義其他文件的細部規格。

其他文件需要引用時，應引用原始規格，不應自行建立另一套定義。

## 9.2 目標架構與實際實作分離

每項功能必須標示：

- Implemented：已實作。
- Partially Implemented：部分實作。
- Planned：已規劃但尚未實作。
- Proposed：尚待設計確認。
- Deprecated：已淘汰。

不得因為存在 Interface、CLI 或 Service，就宣稱完整功能已經完成。

## 9.3 需求追溯

每項核心需求應能追溯至：

- 來源文件。
- 架構元件。
- 程式實作。
- 測試案例。
- 驗收結果。

## 9.4 變更管理

當核心定位或需求改變時：

1. 更新 AIA-01。
2. 分析 AIA-02 是否受影響。
3. 找出相依規格。
4. 更新實作規格。
5. 更新測試與驗收條件。
6. 記錄變更理由及版本。

## 9.5 文件與程式碼一致性

文件不能只描述理想架構。

必須同時保留：

- 目前實作現況。
- 已知缺口。
- 預期改善方式。
- 驗收條件。

# 10. 全系統驗收目標

AIA 最終不能只以「Agent 可以執行 Task」作為驗收標準。

應至少驗證以下能力：

1. 能在沒有新的人類指令時，持續追蹤既有目標。
2. 能根據環境及專案狀態主動發現值得處理的工作。
3. 能在不同 Session 之間維持完整工作脈絡。
4. 能在多個長期 Project 之間切換而不遺失進度。
5. 能檢查自己過去的決策與成果。
6. 能根據檢查結果修正自己的策略。
7. 能判斷目標是否需要調整。
8. 能將工作交給其他 Domain Agent。
9. 能透過其他領域改善自身工作能力。
10. 能在受控環境中修改自己的程式與工具。
11. 能驗證修改是否有效。
12. 能在改善失敗時回復到穩定版本。
13. 能遵守人類設定的權限與安全邊界。
14. 能在沒有值得執行的工作時選擇等待。
15. 能提供可追溯的長期自主性成長紀錄。

# 11. 最終設計原則

AIA 的核心不是讓 AI 永遠忙碌，也不是讓 AI 不斷產生新的 Task。

AIA 必須能夠：

- 知道自己正在做什麼。
- 知道為什麼要做。
- 知道過去做過什麼。
- 知道目前有哪些問題。
- 知道什麼值得繼續改善。
- 知道什麼時候應該改變方向。
- 知道什麼時候應該停止。

它可以在投資、程式開發與小說創作之間切換，維持各自的長期目標，並透過跨領域協作改善自身能力。

**最終目標不是建立一個永遠執行任務的 Agent，而是建立一個能夠持續理解目標、維持自身工作脈絡、主動改善成果，並在治理範圍內演化自身能力的自主系統。**
