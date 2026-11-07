# AIA-18：Autonomy Core Integration & End-to-End Reference Architecture

> [!NOTE]
> **架構重整與收攏公告 (Consolidated Architecture Specification)**
> 本規範為 AIA 歷史基準規範（規格編號已於 AIA-22 永久封頂）。為杜絕規格膨脹與重複定義，本規範之現行核心標準已正式收攏重整至 **[Volume 1: System Foundations & Architecture](file:///home/wade/src/github/aia/docs/specifications/volume-1-system-foundations.md)**。最新架構與維護基準請以對應之 Volume 為主，完整六大核心卷軸總覽請參閱 [Specification Catalog (README.md)](file:///home/wade/src/github/aia/docs/specifications/README.md)。

**文件編號：** AIA-18  
**文件名稱：** Autonomy Core Integration & End-to-End Reference Architecture  
**版本：** 2.0  
**文件狀態：** Architecture Reference Specification  
**適用系統：** AIA (Autonomous Intelligence Architecture)  
**依賴文件：** AIA-01 ～ AIA-17  
**核心範圍：** System Integration、Autonomy Core、Runtime、Domain Agent、Worker、Experiment、Continuity、Governance

---

# 1. 文件目的

## 1.1 設計背景

AIA 的規格已經逐步建立以下能力：

- 個人 AI Agent 平台。
- 長期 Project 管理。
- 多 Agent 組織與協作。
- Task 與 Worker 執行。
- 長期 Memory。
- 自主性思維。
- 時間連續性。
- 自我修正。
- 目標演化。
- 自主實驗。
- 證據與知識完整性。
- 人類監督與治理。

這些能力各自具有明確的設計責任。

然而，單獨定義各個元件，並不代表它們自然就能組成完整的自主系統。

實際實作時仍必須回答：

1. 哪個元件負責接收使用者目標？
2. 哪個元件負責持續觀察環境？
3. 哪個元件決定是否產生新工作？
4. 哪個元件決定是否委派給其他 Agent？
5. 哪個元件建立 Task？
6. 哪個元件派發 Worker？
7. 哪個元件呼叫 AI Harness？
8. 哪個元件檢查執行結果？
9. 哪個元件保存 Decision 與 Learning？
10. 哪個元件負責跨 Session 恢復？
11. 哪個元件決定是否需要人類核准？
12. 當 Agent 修改自身程式碼時，如何避免破壞整個系統？

AIA-18 將這些責任整合成完整的參考架構。

## 1.2 文件目標

本文件定義：

1. AIA 整體系統分層。
2. Autonomy Core 的內部模組。
3. Runtime 與 Autonomy Core 的責任分工。
4. Domain Agent 與 Worker 的關係。
5. Task、Project、Experiment 的整合。
6. Memory、Evidence、Knowledge 的資料流。
7. Decision、Execution、Evaluation 的完整循環。
8. Governance 與 Authorization 的整合。
9. Session 與跨時間恢復。
10. 自主實驗的端到端流程。
11. Go Runtime 的整合方向。
12. 演進式實作與驗收基準。

## 1.3 本文件的定位

AIA-18 是 Reference Architecture，不是要求立即重寫所有既有程式碼的 Implementation Specification。

其目的在於：

- 確立整體元件關係。
- 避免各規格重複定義相同責任。
- 找出既有元件應承擔的功能。
- 建立清楚的資料流與控制流。
- 提供未來實作與程式碼審查的共同依據。

---

# 2. AIA 的核心系統模型

## 2.1 AIA 的本質

AIA 是一個具備長期自主運作能力的個人 AI Agent 平台。

它不是單純的：

- Chatbot。
- Task Scheduler。
- Multi-Agent Framework。
- Tool Calling Framework。
- Workflow Engine。
- Long-term Memory System。

這些技術都可能是 AIA 的組成部分，但都不能單獨代表 AIA。

AIA 的核心是：

**利用持續運作的 Runtime、共用 Autonomy Core、Domain Agent、長期狀態與治理機制，使 AI 能夠在不同領域中持續觀察、主動發現工作、執行、驗證、學習與修正。**

## 2.2 四大自主能力

AIA 的 Autonomy Core 必須支援四項核心能力。

| 能力 | 英文 | 系統責任 |
|---|---|---|
| 主動性思維 | Initiative | 主動發現值得處理的問題與機會 |
| 時間連續性 | Continuity | 跨 Session 維持目標、狀態與工作脈絡 |
| 自我修正 | Self-Correction | 根據證據與結果修正錯誤與方法 |
| 目標演化 | Goal Evolution | 提出、優化、調整與管理目標 |

四項能力不是四個獨立 Agent。

它們應由共用 Autonomy Core 提供，再由不同 Domain Agent 在各自領域中使用。

## 2.3 三個自主實驗領域

目前 AIA 的第一批 Domain Agent 包括：

| Domain | 主要實驗內容 |
|---|---|
| Investment | 自主投資研究、虛擬投資、績效評估與策略修正 |
| Coding | 自主理解程式碼、除錯、修正、測試與程式改善 |
| Writing | 自主管理長期寫作、歷史校勘、人物與劇情一致性 |

這三個領域是自主能力的實驗環境，不是三套彼此獨立的自主架構。

未來可以增加其他 Domain，但不應因此重寫 Autonomy Core。

---

# 3. 整體系統分層

## 3.1 Architecture Overview

AIA 建議採取以下邏輯分層：

```text
+------------------------------------------------------+
|                  User / Owner                        |
|                                                      |
| Goals / Instructions / Approval / Oversight         |
+---------------------------+--------------------------+
                            |
                            v
+------------------------------------------------------+
|                User Interaction Layer                |
|                                                      |
| CLI / Embedded Web UI / Notification / Approval     |
+---------------------------+--------------------------+
                            |
                            v
+------------------------------------------------------+
|                    AIA Runtime                       |
|                                                      |
| Lifecycle / Scheduler / Event Loop / Task Dispatch   |
| Session / Worker Management / Recovery              |
+---------------------------+--------------------------+
                            |
                            v
+------------------------------------------------------+
|                    Autonomy Core                     |
|                                                      |
| Observe / Recall / Assess / Discover                 |
| Deliberate / Decide / Execute Coordination           |
| Verify / Learn / Reassess                            |
|                                                      |
| Initiative / Continuity / Self-Correction            |
| Goal Evolution / Experiment Management              |
+---------------------------+--------------------------+
                            |
                            v
+------------------------------------------------------+
|                  Domain Agent Layer                  |
|                                                      |
| Investment Agent / Coding Agent / Writing Agent     |
|                                                      |
| Domain Knowledge / Domain Strategy / Domain Tools   |
+---------------------------+--------------------------+
                            |
                            v
+------------------------------------------------------+
|                    Execution Layer                   |
|                                                      |
| Worker / Task Executor / Tool Adapter                |
| Antigravity CLI Harness / External Tools             |
+---------------------------+--------------------------+
                            |
                            v
+------------------------------------------------------+
|                 External Environment                 |
|                                                      |
| Market Data / Repository / Documents / Web Sources  |
+------------------------------------------------------+
```

上述分層是邏輯責任模型，不代表每一層都必須獨立成為一個 Process 或 Go Package。

## 3.2 Shared Platform Services

以下服務橫跨多個分層：

```text
+------------------------------------------------------+
|                 Shared Platform Services             |
|                                                      |
| Memory / Project / Knowledge / Evidence              |
| Source Registry / Provenance / Claim Store           |
| Policy / Authorization / Approval / Audit            |
| Agent Registry / Capability Registry                 |
| Evaluation / Observability / Resource Management     |
| Persistence / Event Store / Configuration            |
+------------------------------------------------------+
```

這些服務不屬於某一個 Domain Agent。

它們是 AIA 的共用基礎設施。

## 3.3 Logical Separation

邏輯分層不等於部署分層。

初期可以：

- 使用同一個 Go Process。
- 使用同一個資料庫。
- 使用同一個 Runtime。
- 使用同一個 CLI。
- 透過內部 Interface 分離責任。

未來若有必要，再將部分服務拆成獨立 Process。

不應為了追求微服務架構，過早增加部署與維運複雜度。

---

# 4. Core Components & Responsibilities

## 4.1 User Interaction Layer

負責：

- 接收使用者輸入。
- 顯示 Agent 狀態。
- 顯示 Project 與 Experiment。
- 提供人工核准。
- 提供暫停、恢復與停止操作。
- 顯示重要自主決策。
- 顯示長期成果。

CLI 與 Embedded Web UI 是不同的使用介面，不應各自實作一套自主決策邏輯。

## 4.2 AIA Runtime

Runtime 是 AIA 的持續運作與執行管理層。

負責：

- Process Lifecycle。
- Event Loop。
- Scheduling。
- Task Management。
- Worker Management。
- Session Management。
- Agent Dispatch。
- Persistence Coordination。
- Recovery。
- Runtime Health。
- Governance Enforcement。

Runtime 不應自行承擔所有領域判斷。

例如：

Runtime 可以喚醒 Investment Agent，但不應自行決定今天應該買進哪一檔股票。

## 4.3 Autonomy Core

Autonomy Core 是 AIA 的自主決策核心。

負責：

- 觀察環境與系統狀態。
- 取得相關 Memory。
- 評估目前目標。
- 發現值得處理的事項。
- 建立 Candidate Actions。
- 評估成本與風險。
- 決定是否行動。
- 決定是否委派。
- 決定是否需要人類核准。
- 追蹤結果。
- 啟動 Learning 與 Self-Correction。
- 管理 Goal Evolution。

Autonomy Core 不應直接取代 Runtime 的 Task、Worker 與 Process 管理。

## 4.4 Domain Agent

Domain Agent 負責特定領域的專業工作。

例如：

Investment Agent：

- 市場觀察。
- 投資研究。
- 持股分析。
- 虛擬交易決策。
- 投資績效分析。

Coding Agent：

- 程式碼理解。
- 架構分析。
- 除錯。
- 程式修正。
- 測試與回歸。

Writing Agent：

- 長期大綱。
- 人物管理。
- 歷史資料研究。
- 年表檢查。
- 跨卷一致性。
- 劇情修訂。

Domain Agent 可以提供專業判斷，但不能自行突破 AIA 共通治理規則。

## 4.5 Worker

Worker 是具體工作執行單位。

一個 Agent 可以有多個 Worker。

例如：

```text
Investment Agent
    |
    +-- Market Research Worker
    +-- Financial Data Worker
    +-- Portfolio Analysis Worker
    +-- Performance Evaluation Worker

Coding Agent
    |
    +-- Code Analysis Worker
    +-- Debugging Worker
    +-- Test Worker
    +-- Documentation Worker

Writing Agent
    |
    +-- Historical Research Worker
    +-- Timeline Audit Worker
    +-- Character Consistency Worker
    +-- Chapter Review Worker
```

以上是工作角色示意，不代表必須為每個角色建立永久獨立 Agent。

Worker 的生命週期由既有 Runtime 管理。

## 4.6 Harness

Harness 是底層 AI Agent 執行環境。

目前使用 Antigravity CLI（agy）。

其責任包括：

- 模型執行。
- 工具呼叫。
- 工具權限。
- Filesystem 存取。
- Command 執行。
- Network 存取。
- Sandbox。
- MCP 整合。

AIA 不應重複實作 Harness 已提供的底層工具執行能力。

AIA 必須管理更高層的目標、任務、專案、實驗與自主性。

---

# 5. Autonomy Core Internal Architecture

## 5.1 核心模組

Autonomy Core 建議由以下邏輯模組組成：

```text
+------------------------------------------------------+
|                    Autonomy Core                     |
|                                                      |
|  +----------------+   +---------------------------+  |
|  | Observation    |-->| Context Reconstruction    |  |
|  +----------------+   +---------------------------+  |
|                                |                     |
|                                v                     |
|  +----------------+   +---------------------------+  |
|  | Assessment     |-->| Initiative Discovery      |  |
|  +----------------+   +---------------------------+  |
|                                |                     |
|                                v                     |
|  +----------------+   +---------------------------+  |
|  | Deliberation   |-->| Decision & Goal Management|  |
|  +----------------+   +---------------------------+  |
|                                |                     |
|                                v                     |
|  +----------------+   +---------------------------+  |
|  | Governance     |-->| Action Coordination       |  |
|  +----------------+   +---------------------------+  |
|                                |                     |
|                                v                     |
|  +----------------+   +---------------------------+  |
|  | Verification   |-->| Learning & Correction     |  |
|  +----------------+   +---------------------------+  |
|                                |                     |
|                                v                     |
|                       Reassessment                  |
+------------------------------------------------------+
```

## 5.2 Observation

取得目前系統與外部環境資訊。

來源包括：

- Runtime Events。
- Task State。
- Worker State。
- Project State。
- Experiment State。
- Market Data。
- Repository Changes。
- Writing Progress。
- User Input。
- Notifications。
- Scheduled Triggers。

Observation 不代表所有資料都要立即進行完整分析。

應先依照重要性與觸發條件決定是否深入處理。

## 5.3 Context Reconstruction

根據目前工作重建必要脈絡。

包括：

- 相關 Goal。
- Project 狀態。
- Experiment 狀態。
- 過去 Decision。
- Memory。
- Evidence。
- 未完成 Task。
- Next Action Candidates。
- Resource Constraints。
- Governance Constraints。

Context Reconstruction 不應無條件載入所有歷史資料。

應根據目前任務與目標進行相關性選取。

## 5.4 Assessment

評估目前狀態：

- 目標是否仍然有效？
- 是否存在未完成工作？
- 是否出現新資訊？
- 是否發生異常？
- 是否有重要差距？
- 是否有新的機會？
- 是否有既有假設需要重新檢查？

Assessment 必須保留依據與不確定性。

## 5.5 Initiative Discovery

主動產生 Candidate Actions。

可能來源：

- Goal Gap。
- New Opportunity。
- System Anomaly。
- Unresolved Issue。
- Pending Decision。
- Experiment Requirement。
- Knowledge Conflict。
- Scheduled Review。
- User Request。

Candidate Action 不一定會變成 Task。

系統必須允許：

- ACT。
- DEFER。
- MONITOR。
- ASK_HUMAN。
- NO_ACTION。
- SLEEP。

## 5.6 Deliberation

對 Candidate Action 進行評估。

考慮：

- Goal Alignment。
- Expected Benefit。
- Cost。
- Risk。
- Urgency。
- Reversibility。
- Dependencies。
- Resource Availability。
- Opportunity Cost。
- Evidence Quality。

Deliberation 可以呼叫 Domain Agent 提供專業意見。

但最終決策仍須符合共通治理規則。

## 5.7 Decision

建立正式 Decision Record。

Decision 必須包含：

- 決策內容。
- 決策理由。
- 相關 Goal。
- 使用的 Evidence。
- 相關 Claim。
- 風險評估。
- 資源影響。
- 授權結果。
- 預期結果。
- 驗證條件。

NO_ACTION 也是正式 Decision，不能因為沒有執行 Task 就省略紀錄。

## 5.8 Action Coordination

根據 Decision：

- 建立 Task。
- 指派 Domain Agent。
- 選擇 Worker。
- 委派給其他 Agent。
- 建立 Approval Request。
- 設定執行條件。
- 設定結果驗證要求。

Autonomy Core 決定做什麼與為什麼做。

Runtime 負責實際管理執行。

## 5.9 Verification

執行完成後，檢查：

- 是否達成預期？
- 是否符合授權？
- 是否產生副作用？
- 是否有新風險？
- 是否需要重新執行？
- 是否需要修正目標或策略？

Verification 必須依據實際結果，不得只依賴 Agent 自我宣稱成功。

## 5.10 Learning & Reassessment

將結果連結至：

- 原始 Decision。
- Evidence。
- Prediction。
- Expected Outcome。
- Actual Outcome。
- Evaluation。
- Root Cause。
- Correction Proposal。

最後更新 Project、Experiment 與 Goal 狀態，進入下一輪自主循環。

---

# 6. Task、Project、Experiment 的整合

## 6.1 三種物件的定位

三者不可混為一談。

| 物件 | 核心問題 | 主要責任 |
|---|---|---|
| Task | 現在要執行什麼？ | 工作執行 |
| Project | 長期要完成什麼？ | 工作成果與進度 |
| Experiment | 要驗證什麼？ | 假設、實驗與改善 |

## 6.2 Project

Project 是長期工作單位。

例如：

- AIA 程式開發。
- 歷史小說創作。
- 長期投資研究。

Project 保存：

- Project Goal。
- Current State。
- Progress。
- Related Tasks。
- Related Experiments。
- Decisions。
- Knowledge。
- Next Action Candidates。

## 6.3 Experiment

Experiment 是長期研究與改善單位。

例如：

Investment Experiment：

驗證 Agent 是否能透過持續研究、交易判斷與策略修正，增加虛擬投資組合總資產。

Coding Experiment：

驗證 Agent 是否能自主發現程式問題、修正、測試並持續改善程式品質。

Writing Experiment：

驗證 Agent 是否能長期維護複雜作品，主動發現史實與敘事矛盾並提出修正。

## 6.4 Task

Task 是具體執行單位。

例如：

- 擷取今日台股資料。
- 分析台積電最新財報。
- 修正 Go Runtime 的錯誤。
- 執行回歸測試。
- 檢查某卷人物年表。
- 核對歷史文獻。

Task 必須能回到其所屬 Project、Experiment 或 Goal。

## 6.5 Relationship

```text
Goal
 |
 v
Project
 |
 +-------------------+
 |                   |
 v                   v
Experiment         Task
 |                   ^
 |                   |
 +----> Candidate ---+
        Actions
```

一個 Project 可以有多個 Experiment。

一個 Experiment 可以建立多個 Task。

一個 Task 也可以服務多個相關目標，但必須明確記錄其關聯。

---

# 7. Continuity & Persistent State

## 7.1 Continuity Model

AIA 不要求同一個 AI Model、Session 或 Process 永遠存在。

時間連續性由持久化狀態與 Context Reconstruction 維持。

```text
Session 001
    |
    v
Decision / Event / Memory
    |
    v
Persistent State
    |
    v
Process Restart
    |
    v
Session 002
    |
    v
Context Reconstruction
    |
    v
Continue Previous Work
```

## 7.2 State Categories

至少需要保存：

- System State。
- Agent State。
- Project State。
- Experiment State。
- Goal State。
- Task State。
- Worker State。
- Session State。
- Decision History。
- Memory。
- Knowledge。
- Pending Approval。
- Resource State。

## 7.3 Event History

重要狀態變化應以 Event 保存。

例如：

- Goal Created。
- Goal Revised。
- Candidate Discovered。
- Decision Made。
- Task Created。
- Task Started。
- Task Completed。
- Evidence Added。
- Learning Recorded。
- Correction Applied。
- Experiment Paused。
- Experiment Resumed。

目前狀態可以由最新狀態快照與 Event History 組合取得。

## 7.4 Context Reconstruction

Agent 恢復工作時：

1. 載入目前 Goal。
2. 載入相關 Project。
3. 載入 Experiment 狀態。
4. 載入未完成 Task。
5. 載入必要 Memory。
6. 載入相關 Evidence。
7. 載入最近 Decision。
8. 檢查 Pending Approval。
9. 檢查 Authorization。
10. 重新評估是否應繼續執行。

不能只把上一個 Session 的對話文字當成完整狀態。

## 7.5 Context Freshness

恢復工作時必須檢查資料是否仍有效。

例如：

- 市場資訊可能已過期。
- 程式碼可能已經變更。
- 寫作大綱可能已經修訂。
- 原授權可能已經撤銷。

Continuity 不等於盲目延續。

真正的連續性是知道過去發生什麼、目前狀態如何，以及哪些事情需要重新確認。

---

# 8. Memory、Evidence 與 Knowledge Integration

## 8.1 三者分工

| 元件 | 保存內容 |
|---|---|
| Memory | 有用的長期資訊與工作脈絡 |
| Evidence | 支持或反駁 Claim 的資料 |
| Knowledge | 經過一定程度驗證的可引用資訊 |

三者可以互相連結，但不應混成單一資料型態。

## 8.2 Data Flow

```text
External Source
      |
      v
Source Registry
      |
      v
Evidence Registry
      |
      v
Claim Store
      |
      v
Validation
      |
      v
Knowledge
      |
      v
Memory Integration
      |
      v
Autonomy Core
```

Memory 可以保存未驗證資訊，但必須標示其性質。

例如：

- 暫時觀察。
- 待驗證假設。
- 使用者確認事項。
- 已驗證知識。
- 已過時資訊。
- 存在爭議的資訊。

## 8.3 Decision Snapshot

重要 Decision 必須保存當時使用的 Evidence 與 Knowledge 版本。

這能避免未來更新資料後，無法還原當時的決策依據。

## 8.4 Correction Propagation

當原始 Evidence 發生變化：

1. 找出相關 Claim。
2. 找出相關 Knowledge。
3. 找出相關 Memory。
4. 找出使用這些資料的 Decision。
5. 找出相關 Learning Record。
6. 評估是否需要重新驗證。
7. 更新目前有效狀態。
8. 保留所有歷史版本。

---

# 9. Governance Integration

## 9.1 Governance Is a Shared Control Plane

Governance 不是某個 Domain Agent 的附屬功能。

所有 Agent、Worker、Task 與 Experiment 都必須遵守共通治理。

## 9.2 Decision Authorization Flow

```text
Candidate Action
      |
      v
Deliberation
      |
      v
Risk Assessment
      |
      v
Policy Evaluation
      |
      +----> ALLOW
      |
      +----> REQUIRE APPROVAL
      |
      +----> DENY
      |
      v
Authorization Check
      |
      v
Runtime Enforcement
      |
      v
Harness Permission Check
      |
      v
Execution
```

## 9.3 Responsibility Separation

| 元件 | 責任 |
|---|---|
| Autonomy Core | 決定是否值得行動 |
| Governance Engine | 判斷是否允許 |
| Approval Manager | 管理人類核准 |
| Runtime | 確保執行符合授權 |
| Harness | 控制底層工具與環境權限 |
| Audit | 保存治理與執行歷史 |

任何單一元件都不能獨自代表整個治理流程。

## 9.4 Human Oversight

人類可以：

- 查看自主決策。
- 核准高風險行動。
- 拒絕操作。
- 暫停 Agent。
- 暫停 Experiment。
- 撤銷授權。
- 停止系統。
- 修改核心目標。
- 調整資源限制。

人類不需要參與每個低風險 Task。

## 9.5 Emergency Control

Emergency Stop 必須由 Runtime 或更高層級控制機制執行。

不能依賴 Autonomy Core 自己判斷是否應停止。

---

# 10. Agent Collaboration Integration

## 10.1 Agent Registry

Agent Registry 保存：

- Agent ID。
- Domain。
- Capabilities。
- Version。
- Status。
- Available Workers。
- Permission Profile。
- Resource Constraints。

## 10.2 Capability Discovery

Autonomy Core 根據工作需求尋找適合 Agent。

例如：

Investment Agent 需要分析程式碼：

1. 向 Agent Registry 查詢 Coding Agent。
2. 確認其能力。
3. 評估委派效益。
4. 檢查授權。
5. 建立 Delegation。
6. 接收結果。
7. 驗證結果。
8. 記錄責任與來源。

## 10.3 Delegation

Delegation 必須包含：

- Delegation ID。
- Requesting Agent。
- Receiving Agent。
- Task Scope。
- Expected Output。
- Constraints。
- Deadline。
- Evidence Requirements。
- Authorization Reference。

## 10.4 Collaboration Is Optional

不是每個 Task 都需要多 Agent。

Autonomy Core 應比較：

- 自己完成的成本。
- 委派成本。
- 協作效益。
- 風險。
- 時間。
- 資源。

只有在協作具有實際價值時才委派。

## 10.5 Responsibility

委派不代表原始 Agent 可以完全放棄責任。

委派者仍需：

- 檢查輸出。
- 驗證 Evidence。
- 判斷是否採用。
- 保存決策紀錄。

---

# 11. End-to-End Autonomous Cycle

## 11.1 完整循環

AIA 的核心運作循環：

```text
OBSERVE
   |
   v
RECALL
   |
   v
ASSESS
   |
   v
DISCOVER
   |
   v
DELIBERATE
   |
   v
DECIDE
   |
   v
GOVERNANCE CHECK
   |
   v
EXECUTE
   |
   v
VERIFY
   |
   v
LEARN
   |
   v
REASSESS
   |
   +--------> OBSERVE
```

## 11.2 Step 1：Observe

Runtime 根據 Event、Schedule、State Change 或外部訊號喚醒 Autonomy Core。

Autonomy Core 取得目前狀態。

## 11.3 Step 2：Recall

Context Reconstruction 載入相關：

- Goal。
- Project。
- Experiment。
- Memory。
- Knowledge。
- Evidence。
- Decision History。
- Task。
- Worker。

## 11.4 Step 3：Assess

檢查目前狀態與目標之間的差距。

例如：

- 是否有未完成工作？
- 是否有重要資料更新？
- 是否有錯誤尚未修正？
- 是否有新風險？
- 是否有實驗需要評估？

## 11.5 Step 4：Discover

產生 Candidate Actions。

例如：

- 取得最新市場資料。
- 檢查投資策略。
- 修正程式碼。
- 檢查歷史年表。
- 更新 Project 狀態。
- 重新評估某項假設。

## 11.6 Step 5：Deliberate

評估是否值得行動。

可以選擇：

- ACT。
- DEFER。
- MONITOR。
- ASK_HUMAN。
- NO_ACTION。
- SLEEP。

## 11.7 Step 6：Decide

建立 Decision Record。

若決定執行，建立明確的 Action Plan。

## 11.8 Step 7：Governance Check

檢查：

- Policy。
- Authorization。
- Risk。
- Resource。
- Approval。

未獲授權的操作不得進入執行階段。

## 11.9 Step 8：Execute

Runtime 建立 Task，指派 Agent 與 Worker。

Worker 透過 Harness 或其他允許的工具完成工作。

## 11.10 Step 9：Verify

檢查實際結果。

例如：

- 測試是否真的通過？
- 程式碼是否真的修正？
- 虛擬交易是否依模擬規則完成？
- 史實資料是否有足夠來源？
- 章節進度是否符合預期？

## 11.11 Step 10：Learn

建立 Learning Record。

分析預測與結果的差異。

必要時提出 Self-Correction。

## 11.12 Step 11：Reassess

更新：

- Goal。
- Project。
- Experiment。
- Memory。
- Knowledge。
- Next Action Candidates。

然後等待下一次事件或主動觀察。

**自主循環可以持續，但不代表必須不間斷地消耗模型資源。**

系統可以在沒有值得採取的行動時進入 SLEEP。

---

# 12. End-to-End Domain Scenarios

## 12.1 Investment Experiment

### 實驗目標

使用 NT$100,000 虛擬資金進行台股自主投資研究。

主要目標：

最大化投資組合總資產。

特別關注台積電 2330。

### 運作流程

```text
Market Observation
       |
       v
Evidence Collection
       |
       v
Portfolio Reconstruction
       |
       v
Investment Assessment
       |
       v
Candidate Actions
       |
       v
Investment Deliberation
       |
       v
Virtual Trade Decision
       |
       v
Portfolio Simulation
       |
       v
Result Verification
       |
       v
Performance Evaluation
       |
       v
Strategy Learning
       |
       v
Next Trading Day
```

### 每日行為

每個證券營業日：

1. 觀察市場。
2. 取得必要資料。
3. 檢查台積電與相關市場資訊。
4. 檢查持股與現金。
5. 評估投資機會。
6. 自主決定買進、賣出、增持、減持、持有或觀望。
7. 保存決策與證據。
8. 追蹤結果。

每天必須觀察與判斷，但不強迫成交。

### 每月檢驗

每月 1 日：

- 檢查總資產。
- 檢查持股。
- 檢查現金。
- 檢查交易紀錄。
- 檢查風險。
- 檢查策略變化。
- 檢查預測與結果。
- 評估自主性表現。

使用者可以檢查結果，但不必每天決定交易內容。

### 治理邊界

- 僅使用虛擬資金。
- 不自動執行真實金融交易。
- 不自行擴大資金。
- 不改變核心目標。
- 所有交易必須可追溯。

## 12.2 Coding Experiment

### 實驗目標

讓 Coding Agent 能夠持續理解與改善 AIA 專案。

### 運作流程

```text
Repository Observation
       |
       v
Codebase Understanding
       |
       v
Issue Discovery
       |
       v
Root Cause Analysis
       |
       v
Correction Proposal
       |
       v
Governance Check
       |
       v
Isolated Code Modification
       |
       v
Test / Regression
       |
       v
Diff Review
       |
       v
Correction Verification
       |
       v
Learning Record
       |
       v
Next Improvement
```

### 自主能力

Coding Agent 可以：

- 主動檢查技術債。
- 發現錯誤。
- 分析架構問題。
- 提出改善。
- 在授權範圍內修改程式碼。
- 執行允許的測試。
- 檢查回歸問題。
- 追蹤修正成效。

### 治理邊界

- Domain Project 修改依授權執行。
- AIA 核心修改需要嚴格核准。
- Governance 與 Audit 不得由 Agent 自行修改。
- 不得繞過 Harness 權限。
- 修改必須有 Diff 與回滾方式。

## 12.3 Writing Experiment

### 實驗目標

讓 Writing Agent 能夠長期維護大型作品，持續檢查歷史、人物、社會背景與跨卷一致性。

### 運作流程

```text
Project State Observation
       |
       v
Outline / Timeline Recall
       |
       v
Historical Evidence Check
       |
       v
Character / Event Analysis
       |
       v
Inconsistency Discovery
       |
       v
Revision Proposal
       |
       v
Historical Validation
       |
       v
User Setting Check
       |
       v
Revision
       |
       v
Continuity Update
       |
       v
Next Review
```

### 自主能力

Writing Agent 可以：

- 主動檢查大綱。
- 發現人物年齡錯誤。
- 發現事件時間衝突。
- 檢查人物關係。
- 比較史料。
- 提出劇情修正。
- 追蹤各卷進度。
- 維護跨卷設定。

### 治理邊界

- 使用者確認的設定不得無痕覆蓋。
- 史實、合理推演與小說虛構必須區分。
- 重大敘事方向變更應取得使用者確認。
- 重要歷史修訂必須保留依據。

---

# 13. Decision & Execution Data Flow

## 13.1 Decision Record

每個重要 Decision 都必須具有一致資料模型。

建議包含：

```text
Decision
├── decision_id
├── agent_id
├── goal_id
├── project_id
├── experiment_id
├── candidate_action_id
├── decision_type
├── decision_reason
├── evidence_refs[]
├── knowledge_refs[]
├── risk_assessment
├── authorization_ref
├── approval_ref
├── expected_outcome
├── verification_criteria
└── created_at
```

## 13.2 Task Record

Task 應包含：

```text
Task
├── task_id
├── parent_task_id
├── project_id
├── experiment_id
├── goal_id
├── assigned_agent
├── assigned_worker
├── task_type
├── input_refs[]
├── output_refs[]
├── authorization_ref
├── status
├── created_at
└── completed_at
```

## 13.3 Execution Record

Execution Record 保存：

- Worker。
- Harness Session。
- 工具呼叫。
- 輸入與輸出。
- 執行時間。
- 執行狀態。
- Error。
- Retry。
- Resource Usage。

執行結果必須能對應原始 Task。

## 13.4 Verification Record

保存：

- 驗證目標。
- 驗證方法。
- Evidence。
- 驗證結果。
- 未通過項目。
- 後續建議。
- 驗證時間。

## 13.5 Learning Record

連結：

- Decision。
- Execution。
- Verification。
- Expected Outcome。
- Actual Outcome。
- Root Cause。
- Correction Proposal。

這使 AIA 能夠形成真正可追溯的自主學習循環。

---

# 14. Event-Driven Runtime Integration

## 14.1 Runtime Trigger

Autonomy Core 不應只靠固定排程啟動。

觸發來源包括：

- User Input。
- Timer。
- External Data Update。
- Task Completion。
- Worker Failure。
- Project State Change。
- Experiment Milestone。
- New Evidence。
- Knowledge Conflict。
- Approval Result。
- Resource Threshold。
- Recovery Event。

## 14.2 Event Flow

```text
Event Source
    |
    v
Event Bus / Event Queue
    |
    v
Runtime Event Handler
    |
    v
Trigger Evaluation
    |
    v
Autonomy Core Wakeup
    |
    v
Context Reconstruction
    |
    v
Autonomous Assessment
```

## 14.3 Trigger Coalescing

大量相似事件不應造成重複喚醒。

例如：

短時間內收到多筆相同市場資料更新，可以合併成一次 Observation Trigger。

## 14.4 Trigger Priority

建議分成：

- CRITICAL：緊急安全或系統事件。
- HIGH：重大錯誤與重要狀態變更。
- NORMAL：一般工作與資料更新。
- LOW：週期性檢查與非緊急觀察。

Priority 只決定處理順序，不代表可以跳過治理檢查。

## 14.5 Idle State

當沒有值得執行的工作時：

- 不建立空轉 Task。
- 不重複呼叫模型。
- 不產生無意義的工具活動。
- 保存目前狀態。
- 等待下一個觸發條件。

SLEEP 是合法的自主決策。

---

# 15. Failure Handling & Recovery

## 15.1 Failure Categories

至少區分：

- Model Failure。
- Harness Failure。
- Tool Failure。
- Worker Failure。
- Task Failure。
- Data Failure。
- Persistence Failure。
- Governance Failure。
- Resource Exhaustion。
- External Service Failure。

## 15.2 Failure Handling

不同錯誤應有不同處理方式。

例如：

| 錯誤 | 處理方式 |
|---|---|
| Model Timeout | 依政策有限次重試 |
| Tool Failure | 檢查工具狀態與錯誤 |
| Auth Expired | WAITING_FOR_AUTH |
| Worker Failure | 隔離 Worker 並保存狀態 |
| Invalid Evidence | 標記資料錯誤並停止依賴 |
| Policy Denied | 不執行，保留拒絕紀錄 |
| Persistence Failure | 停止可能造成資料不一致的操作 |
| Resource Exhaustion | 暫停或降級工作 |

## 15.3 Authentication Recovery

若 Harness 需要重新登入：

1. 停止需要認證的操作。
2. Task 進入 WAITING_FOR_AUTH。
3. 保留執行狀態。
4. 由使用者完成登入。
5. Runtime 確認 Harness 可用。
6. 重新檢查 Task 與授權。
7. 執行 Resume。

AIA 不應自行讀取或複製使用者的 OAuth Token。

## 15.4 Idempotency

可能重複執行的工作應考慮 Idempotency。

例如：

- 同一筆虛擬交易不能因重試而重複入帳。
- 同一份 Evidence 不應因重試而無限制重複建立。
- 同一個 Task 的完成事件不能造成重複 Learning Record。
- 同一個 Approval 不應被重複使用於不同操作。

## 15.5 Recovery Integrity

恢復後必須重新確認：

- 當前狀態。
- 未完成工作。
- 資料完整性。
- 授權有效性。
- 目標有效性。
- 外部環境變化。

不能只依照舊 Session 的文字描述繼續工作。

---

# 16. Security Architecture

## 16.1 Security Boundaries

AIA 至少需要以下安全邊界：

1. User Boundary。
2. Runtime Boundary。
3. Autonomy Boundary。
4. Agent Boundary。
5. Worker Boundary。
6. Harness Boundary。
7. Workspace Boundary。
8. External Data Boundary。

## 16.2 Least Privilege

所有 Agent 與 Worker 都只能取得完成工作所需的最低權限。

## 16.3 Untrusted Data

外部資料、網頁內容、文件與其他 Agent 的輸出都必須視為不可信輸入。

不得讓外部內容直接覆蓋：

- System Instruction。
- Governance Policy。
- Authorization。
- User Goal。
- Security Constraint。

## 16.4 No Permission Inheritance

委派時不能自動繼承超出必要範圍的權限。

## 16.5 No Governance Self-Modification

Agent 不得自行：

- 關閉 Policy。
- 修改 Authorization。
- 解除工具權限限制。
- 修改 Audit。
- 停用 Emergency Control。

## 16.6 Local Only

AIA v1 的 Local Only 約束仍然適用：

- Web Host 預設綁定 `127.0.0.1`。
- 不預設開放 LAN。
- 不預設提供 Internet 公開存取。

若未來改變部署邊界，必須進行新的安全設計與核准。

---

# 17. Go Runtime Integration

## 17.1 Implementation Philosophy

AIA 目前以 Go 作為核心 Runtime 的實作語言。

架構應保持：

- 明確 Interface。
- 模組化責任。
- 可測試。
- 可替換。
- 語言中立的資料模型。
- 不與單一 AI Model 綁定。
- 不與單一 Harness 綁定。

未來若改用 Kotlin 或其他語言，不應改變整體自主性模型。

## 17.2 Suggested Package Boundaries

以下是邏輯分組建議，不代表要求立即調整既有目錄：

```text
aia/
├── cmd/
│
├── runtime/
│   ├── lifecycle/
│   ├── scheduler/
│   ├── events/
│   ├── recovery/
│   ├── sessions/
│   └── dispatch/
│
├── autonomy/
│   ├── cycle/
│   ├── observation/
│   ├── context/
│   ├── assessment/
│   ├── initiative/
│   ├── deliberation/
│   ├── decision/
│   ├── goals/
│   ├── correction/
│   └── experiments/
│
├── agent/
│   ├── registry/
│   ├── capability/
│   ├── delegation/
│   └── domain/
│
├── worker/
│
├── task/
│
├── project/
│
├── memory/
│
├── knowledge/
│
├── evidence/
│
├── provenance/
│
├── learning/
│
├── governance/
│   ├── policy/
│   ├── authorization/
│   ├── approval/
│   ├── risk/
│   └── oversight/
│
├── audit/
│
├── evaluation/
│
├── harness/
│
├── web/
│
└── storage/
```

實際目錄應以現有程式碼為準。

不應為了符合示意圖而破壞既有模組。

## 17.3 Interface Boundaries

核心模組應透過 Interface 互動。

例如：

```go
type AutonomyEngine interface {
    Observe(ctx context.Context) error
    Assess(ctx context.Context) error
    Discover(ctx context.Context) ([]CandidateAction, error)
    Deliberate(ctx context.Context, action CandidateAction) (DeliberationResult, error)
    Decide(ctx context.Context, result DeliberationResult) (Decision, error)
}

type TaskDispatcher interface {
    Dispatch(ctx context.Context, task Task) error
}

type GovernanceService interface {
    Evaluate(ctx context.Context, request GovernanceRequest) (GovernanceDecision, error)
}

type EvidenceService interface {
    Register(ctx context.Context, evidence Evidence) error
    Validate(ctx context.Context, evidenceID string) error
}

type ContinuityService interface {
    SaveState(ctx context.Context, state ProjectState) error
    RestoreState(ctx context.Context, projectID string) (ProjectState, error)
}

type LearningService interface {
    Record(ctx context.Context, record LearningRecord) error
    Reassess(ctx context.Context, referenceID string) error
}
```

以上為參考介面，不是要求所有模組立即以相同名稱實作。

實際上應優先沿用現有 Runtime 與 Worker 的 Interface。

## 17.4 Harness Adapter

AIA 應透過 Harness Adapter 與 agy 溝通。

Adapter 負責：

- 啟動 Harness。
- 傳遞工作 Context。
- 管理 Session。
- 收集工具事件。
- 收集標準輸出與錯誤輸出。
- 收集執行結果。
- 檢查執行狀態。
- 保存相關紀錄。

不能只依賴 Process Exit Code 判定成功。

特別是 Headless Mode 可能發生工具 Soft-deny，即使 Exit Code 為 0，也必須檢查工具事件、錯誤訊息、實際結果與 Workspace Diff。

## 17.5 Runtime Persistence

持久化至少應涵蓋：

- Task。
- Worker。
- Agent。
- Project。
- Experiment。
- Goal。
- Decision。
- Event。
- Memory。
- Evidence。
- Approval。
- Authorization。
- Learning Record。

實際儲存技術可以依既有實作決定。

重點是資料關係與恢復能力，而不是特定資料庫品牌。

## 17.6 API & UI Boundary

CLI 與 Web UI 應透過共用 Application Service 存取功能。

例如：

```text
CLI --------+
            |
            v
      Application Services
            |
            v
       Core Runtime
            |
            v
       Domain Modules
            |
            v
         Storage

Web UI -----+
```

UI 不應直接修改底層資料庫，也不應自行實作治理規則。

---

# 18. System Lifecycle

## 18.1 Startup

AIA 啟動時：

1. 載入 Configuration。
2. 初始化 Storage。
3. 載入 Policy。
4. 載入 Agent Registry。
5. 恢復 Project 與 Experiment 狀態。
6. 恢復 Task 與 Worker 狀態。
7. 檢查 Pending Approval。
8. 檢查 Authorization。
9. 初始化 Event Processing。
10. 啟動 Runtime。

## 18.2 Normal Operation

正常運作時：

- Runtime 處理事件。
- Autonomy Core 執行自主循環。
- Domain Agent 提供專業能力。
- Worker 執行具體工作。
- Harness 執行 AI 與工具操作。
- Shared Services 保存狀態與證據。
- Governance 管理權限。
- Audit 保存重要事件。

## 18.3 Shutdown

正常關閉時：

1. 停止接受新的高風險工作。
2. 依政策處理進行中的 Task。
3. 保存 Worker 狀態。
4. 保存 Project 與 Experiment 狀態。
5. 保存 Event。
6. 關閉外部連線。
7. 關閉 Storage。
8. 結束 Runtime。

## 18.4 Restart

重新啟動後，AIA 必須透過 Context Reconstruction 恢復工作。

不要求原本的模型 Session 或 Process 永遠存在。

---

# 19. Implementation Roadmap

## Phase A：Runtime Foundation

基礎能力：

- Persistent Runtime。
- Event Loop。
- Task Lifecycle。
- Worker Lifecycle。
- Session Persistence。
- Recovery。

目標：系統能長期運作並在重啟後恢復。

## Phase B：Continuity & Knowledge

能力：

- Project State。
- Goal State。
- Memory Integration。
- Evidence。
- Provenance。
- Context Reconstruction。

目標：系統能維持跨 Session 的工作脈絡與知識來源。

## Phase C：Initiative Engine

能力：

- Observation。
- Assessment。
- Candidate Discovery。
- Deliberation。
- No-Action Decision。
- Autonomous Scheduling。

目標：Agent 能主動發現值得處理的工作。

## Phase D：Decision & Governance

能力：

- Decision Record。
- Risk Assessment。
- Policy Evaluation。
- Authorization。
- Approval。
- Human Oversight。

目標：自主決策能在明確治理範圍內執行。

## Phase E：Learning & Self-Correction

能力：

- Prediction。
- Verification。
- Evaluation。
- Root Cause Analysis。
- Correction Proposal。
- Controlled Correction。

目標：Agent 能根據實際結果修正自己的方法。

## Phase F：Goal Evolution & Experiments

能力：

- Goal Persistence。
- Goal Optimization。
- Goal Revision。
- Experiment Lifecycle。
- Experiment Evaluation。

目標：Agent 能長期維持與改善自主實驗。

## Phase G：Domain Integration

能力：

- Investment Agent。
- Coding Agent。
- Writing Agent。
- Cross-Agent Collaboration。

目標：透過不同領域驗證共用 Autonomy Core。

以上是能力演進順序，不代表必須等前一階段所有功能完全成熟，才開始下一階段的有限實驗。

## Phase H：Longitudinal Autonomy Evaluation

能力：

- 長期自主性量測。
- 跨 Session 評估。
- 自主行為紀錄。
- 人類介入率分析。
- 策略改善追蹤。
- Domain Benchmark。

目標：確認系統是否真的逐步具備長期自主能力，而不是只增加 Task 與工具呼叫數量。

---

# 20. End-to-End Acceptance Criteria

AIA-18 的整體架構應符合以下條件。

## 20.1 Architecture

- [ ] Runtime 與 Autonomy Core 職責分離。
- [ ] Autonomy Core 為共用核心。
- [ ] Domain Agent 負責專業領域工作。
- [ ] Worker 負責具體執行。
- [ ] Harness 負責底層 AI 與工具執行。
- [ ] Shared Services 提供共用資料與治理能力。

## 20.2 Autonomy

- [ ] 能觀察環境。
- [ ] 能重建 Context。
- [ ] 能主動發現工作。
- [ ] 能評估是否行動。
- [ ] 能選擇不行動。
- [ ] 能建立 Decision。
- [ ] 能追蹤執行結果。
- [ ] 能形成 Learning。
- [ ] 能提出 Self-Correction。
- [ ] 能提出 Goal Evolution。

## 20.3 Continuity

- [ ] 跨 Session 保留狀態。
- [ ] Process Restart 後可以恢復。
- [ ] Project 狀態持續保存。
- [ ] Experiment 狀態持續保存。
- [ ] Decision History 可以追溯。
- [ ] Context Reconstruction 能載入必要脈絡。

## 20.4 Governance

- [ ] 所有重要行動經過 Policy Check。
- [ ] Runtime 能阻止未授權操作。
- [ ] 高風險行動需要核准。
- [ ] Agent 無法自行解除治理。
- [ ] 人類可以暫停與停止。
- [ ] Emergency Stop 不依賴 Agent 同意。

## 20.5 Evidence & Learning

- [ ] 重要 Claim 有來源。
- [ ] Evidence 可以追溯。
- [ ] Knowledge 保留不確定性。
- [ ] Decision 保存當時資料。
- [ ] Learning 可以回溯原始經驗。
- [ ] 修正可以找到受影響的歷史紀錄。

## 20.6 Domain Integration

- [ ] Investment 可以自主觀察與模擬交易。
- [ ] Coding 可以自主分析與受控修正。
- [ ] Writing 可以自主維護長期作品。
- [ ] 三個 Domain 共用 Autonomy Core。
- [ ] Domain 不會自行突破共通治理。
- [ ] Agent Collaboration 不會造成權限擴張。

---

# 21. Final Architecture Principles

AIA-18 最終確立以下架構原則：

1. **Autonomy Core 是共同自主核心。** 不應將主動性、連續性、自我修正與目標演化分散到各 Domain Agent。
2. **Runtime 是持續運作基礎。** 負責生命週期、排程、事件、Task、Worker 與恢復。
3. **Agent 是領域專業單位。** 負責各領域的分析、判斷與執行能力。
4. **Worker 是執行單位。** 不等於 Agent，也不取代 Autonomy Core。
5. **Harness 是底層執行環境。** AIA 不應重造 Harness 已提供的工具與權限能力。
6. **Project 是長期工作成果。** 保存工作目標與進度。
7. **Experiment 是長期驗證單位。** 保存假設、基準、結果與改善歷程。
8. **Task 是具體執行單位。** 由 Runtime 管理生命週期。
9. **Memory 不等於 Knowledge。** 所有重要資訊都必須保留來源與可信狀態。
10. **Decision 必須可追溯。** 系統必須知道自己為什麼做出某項選擇。
11. **Learning 必須根據證據。** 結果好壞不能直接代表判斷正確與否。
12. **Governance 必須獨立存在。** Agent 不能自行解除限制。
13. **人類保有最終控制權。** 不必介入每個細節，但可以監督、核准、撤銷與停止。
14. **自主循環必須跨越 Session。** 不依賴單一模型對話或永久執行中的 Process。
15. **架構必須能逐步演進。** 不因為新增 Domain 或更換語言，就必須重建整套系統。

---

# 22. Final Statement

AIA 的核心不是讓 AI 一次完成更多任務，而是讓 AI 能夠長期維持目標、理解環境、主動發現問題、決定行動、執行工作、驗證結果、學習經驗並修正自己。

這種能力不能只靠更大的模型，也不能只靠更多 Agent 或 Worker。

它需要一套完整的系統：

- Runtime 提供持續運作。
- Autonomy Core 提供自主決策。
- Domain Agent 提供專業能力。
- Worker 執行具體工作。
- Harness 提供底層 AI 與工具。
- Memory 維持工作脈絡。
- Evidence 保證知識可追溯。
- Experiment 驗證長期假設。
- Learning 推動自我修正。
- Governance 維持安全與控制權。
- Human Oversight 保留人類最終決定權。

這些元件共同形成 AIA 的自主循環。

**AIA 最終要驗證的，不是 AI 能不能完成一項任務，而是 AI 能不能在時間中持續維持目標、主動改善方法，並且從自己的行動與結果中逐步變得更好。**
