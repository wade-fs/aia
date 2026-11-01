# AIA 跨規格一致性決議登記冊 (Cross-Specification Resolution Register)

**文件代號：** AIA-SPEC-AUDIT-01  
**稽核範圍：** [docs/specifications/](file:///home/wade/src/github/aia/docs/specifications) (AIA-00 ～ AIA-19)  
**稽核基準：** v2.0 系列規範  
**核心目標：** 消除跨規格名詞衝突、責任模糊、狀態機脫節與資料模型歧異，記錄正式衝突裁決；不取代 AIA-00～19 原始規格。  
**最後更新：** 2026-11-01  

---

## 0. 文件權威層級與使用準則 (Authority Tiers)

本文件在 AIA 規格體系中的定位如下：

| 層級 | 文件 | 職責 |
|:---:|---|---|
| **Tier 1（最高）** | AIA-00 ～ AIA-19（共 20 份原始規格） | 系統性質、能力邊界與設計意圖的最終定義來源 |
| **Tier 2** | **本文件（AIA-SPEC-AUDIT-01）** | 記錄跨規格衝突的正式裁決結果；不新增原始規格未涵蓋的能力 |
| **Tier 3** | [docs/Plan.md](file:///home/wade/src/github/aia/docs/Plan.md)（Master Implementation Plan） | 以 Tier 1+2 為依據，規劃具體實作路線與驗收標準 |

> [!IMPORTANT]
> 本文件的每一項裁決**僅解決跨規格之間的矛盾**，所有裁決均以 Tier 1 原始規格的設計意圖為根本依據。若未來 Tier 1 規格修訂，本文件須同步更新。

---

## 1. 稽核背景與核心發現摘要 (Executive Summary)

AIA 規格體系（AIA-00 至 AIA-19，共計 20 份文件、27,500+ 行）完整構建了一套先進的長期自主實驗架構。在進行具體程式開發前，本稽核針對跨規格的實體邊界、自主循環、治理模型與知識溯源進行了全量交叉比對，共辨識出 **6 項關鍵名詞與模型歧異**，並在此提出權威統一判定（Normative Resolutions）：

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   規格交叉稽核關鍵歧異與統一判定                      │
├───────────────────┬───────────────────┬────────────────────────────────┤
│ 稽核項目          │ 涉及衝突規格      │ 權威統一判定 (Authoritative)   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 1. 自修分級名詞   │ AIA-00 vs AIA-14  │ 修改範圍採 Level A/B/C；       │
│                   │                   │ 驗證深度採 Level 1~6           │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 2. 治理分層體系   │ AIA-17 vs AIA-18  │ G0~G3 專指治理層級，           │
│                   │                   │ 不用於實作階段命名             │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 3. 自主核心閉環   │ AIA-00/13 vs      │ 採 AIA-18 之 11 步驟閉環，     │
│                   │ AIA-18            │ 將 Governance Check 顯式列出   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 4. 實體責任邊界   │ AIA-09/10/15/18   │ Goal(目標) ➔ Project(成果) ➔   │
│                   │                   │ Experiment(驗證) ➔ Task(執行)  │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 5. 知識溯源關聯   │ AIA-10 vs AIA-16  │ Memory 不強制外鍵 Evidence，   │
│                   │                   │ 採 Provenance 來源圖譜關聯     │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 6. 緊急停止規範   │ AIA-17 vs AIA-19  │ 1秒內凍結派發，運行中工作      │
│                   │                   │ 發出 Context 取消並受控收尾    │
└───────────────────┴───────────────────┴────────────────────────────────┘
```

---

## 2. 核心名詞與分級對齊 (Terminology Harmonization)

### 2.1 自我修改分級名詞衝突 (Self-Modification Levels)

#### 【衝突現象】
- [AIA-00](file:///home/wade/src/github/aia/docs/specifications/AIA-00-Overall-Architecture-and-Evolution-Roadmap-of-Specification.md) 第 5.13 節將自修分為：`Level 1 (行為修正)`、`Level 2 (能力修正)`、`Level 3 (核心系統修正)`。
- [AIA-14](file:///home/wade/src/github/aia/docs/specifications/AIA-14-Self-Modification-and-Safe-Code-Evolution-Specification.md) 第 2 節將自修範圍定義為：`Level A (領域代碼)`、`Level B (AIA 核心)`、`Level C (治理核心)`；而在第 5 節又將驗證深度定義為：`Level 1~6 (Static ➔ Unit ➔ Integration ➔ Regression ➔ Behavioral ➔ Benchmark)`。

#### 【權威統一判定】
為消除程式碼命名歧義，**以 AIA-14 的定義為唯一權威依據**，並擴展為三個獨立命名空間：

1. **修改對象範圍 (Modification Scope)**：以字母 `A/B/C` 表示，對應 AIA-14 的 Level A/B/C：
   - **`Scope A (Domain Scope)`**：領域工具、採集器、外掛腳本、Prompt。
   - **`Scope B (Core Scope)`**：AIA Runtime、Autonomy Core、共用 Service、Worker。
   - **`Scope C (Governance Scope)`**：Policy Engine、Approval Engine、Audit Log。**Agent 絕對禁止自行修改**。

2. **修正類型 (Correction Type)**：以 `C1～C6` 表示，說明「修正的是什麼」：
   - **`C1 (事實修正)`**：更新知識網中陳舊的 Claim 或事實。
   - **`C2 (判斷修正)`**：修正權重假設與信心指標。
   - **`C3 (策略修正)`**：調校決策啟發式規則。
   - **`C4 (提示詞調優)`**：受控微調 Agent 領域 Prompt。
   - **`C5 (工作流程改善)`**：調整任務拆解步驟或排程頻率。
   - **`C6 (程式碼修正)`**：委派代碼自修流水線（僅 Phase 6 開放）。

3. **驗證流水線深度 (Validation Level)**：以 `V1～V6` 表示，說明「須通過哪些驗證關卡」：
   - **`V1 (Static)`**：靜態分析與編譯檢查。
   - **`V2 (Unit)`**：單元測試。
   - **`V3 (Integration)`**：整合測試。
   - **`V4 (Regression)`**：回歸測試（確保無破壞性變更）。
   - **`V5 (Behavioral)`**：行為模擬與端到端評估。
   - **`V6 (Benchmark)`**：效能基準與自主性指標評估。

> [!NOTE]
> AIA-00 的 `Level 1/2/3`（行為/能力/核心修正）不再使用。AIA-14 第 2 節的 `Level A/B/C` 僅指 Scope，第 5 節的數字 1~6 僅指 Validation，兩者不可混用。Correction Type C1~C6 為本文件新增的第三維度，供 Plan.md Phase 4 自修分類使用。

---

### 2.2 治理層級 (Governance Tiers) 語意收斂

#### 【衝突現象】
- [AIA-17](file:///home/wade/src/github/aia/docs/specifications/AIA-17-Operational-Governance-and-Human-Oversight-Specification.md) 在第 2 節定義了 `G0~G3` 治理層級；但在第 12 節又使用 `Phase G0~G3` 來標示 AIA-17 自身的階段規劃，容易與系統實作 Roadmap 混淆。

#### 【權威統一判定】
1. **`G0` 至 `G3` 專指「治理架構層級 (Governance Tiers)」**：
   - **`G0 (System Governance)`**：系統底層不可逾越的安全邊界（白名單目錄、禁止真實資金交易、禁止自改審批規則）。
   - **`G1 (Owner Governance)`**：系統擁有者（人類使用者）的顯式授權、配置與審批。
   - **`G2 (Autonomy Governance)`**：Autonomy Core 在產生候選工作與深思決策時的策略邊界。
   - **`G3 (Execution Governance)`**：Worker 與 Task 執行期間的並行槽位、資源消耗配額 (Token/CPU Budget) 與逾時控制。
2. 系統實作階段一律統一遵循 [docs/Plan.md](file:///home/wade/src/github/aia/docs/Plan.md) 的 **Phase 0 至 Phase 6**，禁止在代碼或配置中使用「Phase G0~G3」以防混淆。

---

### 2.3 自主閉環步驟數統一 (Autonomy Cycle Steps)

#### 【衝突現象】
- [AIA-00](file:///home/wade/src/github/aia/docs/specifications/AIA-00-Overall-Architecture-and-Evolution-Roadmap-of-Specification.md) 第 5.8 節與 [AIA-13](file:///home/wade/src/github/aia/docs/specifications/AIA-13-Autonomy-Evaluation-and-Benchmarking-Specification.md) 描述為 10 步驟循環：`Observe → Recall → Assess → Discover → Deliberate → Decide → Execute → Verify → Learn → Reassess`。
- [AIA-18](file:///home/wade/src/github/aia/docs/specifications/AIA-18-Autonomy-Core-Integration-and-End-to-End-Reference-Architecture.md) 第 11 節將其細化為 11 個步驟，在 `Decide` 與 `Execute` 之間顯式插入了 `Governance Check`。

#### 【權威統一判定】
**以 AIA-18 的 11 步驟為全系統權威標準**。  
自主決策 (Decide) 產出的意圖，必須在進入排程執行 (Execute) 前經過獨立的治理審查 (Governance Check)。這充分落實了「治理獨立於 Agent 自主決策之外」的根本原則。

```text
 1. Observe (環境觀察)
    ↓
 2. Recall (檢索記憶與先驗經驗)
    ↓
 3. Assess (評估專案狀態與目標進度)
    ↓
 4. Discover (發現問題與候選機會)
    ↓
 5. Deliberate (深思推演、權衡成本風險)
    ↓
 6. Decide (選定決策: ACT | DEFER | MONITOR | ASK_HUMAN | NO_ACTION)
    ↓
 7. Governance Check (Policy 規則驗證、風險分級、必要審批申請)
    ↓
 8. Execute (排入隊列由 Runtime Worker 執行)
    ↓
 9. Verify (取得執行結果並比對預期)
    ↓
10. Learn (成效評估、RCA、經驗寫入記憶)
    ↓
11. Reassess (更新目標與專案狀態，銜接下一循環)
```

---

## 3. 四大核心實體責任邊界 (Entity Responsibility Matrix)

### 3.1 Goal vs Project vs Experiment vs Task 邊界定義

根據 [AIA-08](file:///home/wade/src/github/aia/docs/specifications/AIA-08-Autonomy-Core-Specification.md)、[AIA-09](file:///home/wade/src/github/aia/docs/specifications/AIA-09-Initiative-and-Goal-Evolution-Specification.md)、[AIA-10](file:///home/wade/src/github/aia/docs/specifications/AIA-10-Continuity-and-Long-Term-Project-Management-Specification.md)、[AIA-15](file:///home/wade/src/github/aia/docs/specifications/AIA-15-Autonomous-Experiment-Lifecycle-and-Domain-Integration-Specification.md) 與 [AIA-18](file:///home/wade/src/github/aia/docs/specifications/AIA-18-Autonomy-Core-Integration-and-End-to-End-Reference-Architecture.md)，四者之責任與生命週期絕不可重疊：

| 實體名稱 | 核心回答的問題 | 主要生命週期責任 | 產出物與關聯 |
|---|---|---|---|
| **Goal (目標)** | **「我們長期期望達成什麼狀態？」** | 描述期望達成的業務/系統狀態。支援層級樹 (Parent/Child)、指標 (Metrics) 與優先級。具備修訂 (Revision) 歷程。 | 一個 Goal 可由多個 Project 共同追求（M:N 關係）。不直接執行命令。 |
| **Project (專案)** | **「我們正在哪個長期環境中工作？」** | 長期工作領域與成果的容器。維護工作區目錄、專案進度、人物/代碼/投資組合設定、待決事項 (Pending Issues)。 | 可關聯多個 Goal（M:N，透過 `goal_project_links`）；以 `primary_goal_id` 標示主要目標；包含多個 Experiment 與 Task。 |
| **Experiment (實驗)** | **「我們要驗證什麼假設或策略？」** | 具備科研性質的驗證單元。維護假設 (Hypothesis)、基準線 (Baseline)、對照變因、評估週期與結論。 | 隸屬於特定 Project，服務特定 Goal；產出多個 Task 與 Learning Record。 |
| **Task (任務)** | **「現在具體要執行什麼工單？」** | 具備原子性、可派發給 Worker 執行的工單。維護狀態機 (`READY`, `RUNNING`, `COMPLETED`, `WAITING_APPROVAL`)。 | 隸屬於 Project 或 Experiment；執行完畢即行結束。 |

### 3.2 實體關聯導航圖 (Entity Relationship Topology)

> [!IMPORTANT]
> **Goal ↔ Project 為多對多關係（M:N）**，透過 `goal_project_links` 關聯表實現。Project 的 `primary_goal_id` 是便利欄位，指向最主要的 Goal，但不限制 Project 同時服務多個 Goal。

```text
       ┌──────────────┐
       │     Goal     │ (長期目標樹)
       └──────┬───────┘
              │ M:N (透過 goal_project_links)
              ▼
       ┌──────────────┐
       │   Project    │ (長期工作專案與工作區)
       │ primary_goal_id → Goal (便利 FK)│
       └──────┬───────┘
              ├─────────────────────────────┐
              │ 1:N                         │ 1:N
              ▼                             ▼
       ┌──────────────┐              ┌──────────────┐
       │  Experiment  │ (假設驗證)   │     Task     │ (具體執行工單)
       └──────┬───────┘              └──────▲───────┘
              │                             │
              └─────── 產出候選行動 ──────────┘
```

---

## 4. 知識、記憶與證據溯源體系 (Knowledge & Provenance Harmonization)

### 4.1 核心資料類型分工 (AIA-16 權威定義)

[AIA-16](file:///home/wade/src/github/aia/docs/specifications/AIA-16-Data-Provenance-Evidence-and-Knowledge-Integrity-Specification.md) 確立了嚴謹的資料演進光譜，嚴禁將中間推論與未驗證資料混淆為客觀事實：

```text
外部輸入 ──► [Data] ──(提取)──► [Evidence] ──(支持/反駁)──► [Claim]
                                                               │
                                                               ▼
[Decision] ◄──(依據 Context)── [Knowledge] ◄──(驗證晉升)── [Assertion]
                                   ▲
                                   │ (標註不確定性)
                [Inference] / [Hypothesis]
```

1. **`Data` (原始資料)**：無結構或有結構的原始抓取文字、HTML、API 回應。
2. **`Source` (來源)**：資料產生的源頭，標記信賴等級（官方公告、新聞媒體、模型生成、工具輸出）。
3. **`Evidence` (證據)**：具備來源 URI、摘錄片段與時間戳的客觀憑據。
4. **`Claim` (主張)**：可被檢驗的命題陳述（如「台積電 10 月營收月增 5%」）。
5. **`Assertion` (系統斷言)**：系統在特定時間對 Claim 的判斷（`ACCEPTED`, `REJECTED`, `UNCERTAIN`, `CONFLICTED`）。
6. **`Inference` (推論結論)**：根據前提 Claim 與模型推理得出的結論，**必須保留推論過程與不確定性**。
7. **`Hypothesis` (未驗證假設)**：待實驗驗證的假設，**嚴禁自動升級為事實**。
8. **`Knowledge` (已驗證知識)**：經過充分驗證、可被跨 Agent 引用的穩定事實。
9. **`Memory` (工作記憶)**：Agent 運作脈絡與先驗經驗的綜合載體，**依 Provenance 關聯回溯至上述節點**。

### 4.2 修正 Memory 與 Evidence 的實作介面

- **舊計畫誤區**：強制在 `memories` 資料表建立 `evidence_id NOT NULL` 外鍵。
- **第二個誤區**：「當且僅當 `provenance_type = FACT_BASED` 時，才強制關聯 `evidence_id`」——此說法仍然僵化，因為 Fact 的來源不限於外部 Evidence，也可能來自先前 Decision、Context Snapshot 或 Memory 自身的推論累積。
- **權威修正**：Memory 記錄的是工作脈絡與經驗，其來源形式多樣。因此：
  - `memories` 表採用多型 Provenance 欄位，**而非單一外鍵**：
    - `provenance_type`：`FACT_BASED | INFERENCE | HYPOTHESIS | USER_PREFERENCE | AGENT_DECISION`
    - `provenance_ref_type`：`EVIDENCE | SOURCE | MEMORY | CONTEXT_SNAPSHOT | DECISION | NONE`（指向何種實體）
    - `provenance_ref_id`：對應實體的 ID（polymorphic reference）
  - **強制要求的是 Provenance 鏈的完整性**，而非特定外鍵欄位。任何 Memory 記錄都必須能回答「這條記憶從哪裡來的？」——其可接受的錨點包含：Evidence、Source、Memory（先驗推導）、Context Snapshot、或先前 Decision。
  - 所有 Agent 生成內容 (Model Generated) 預設為 `HYPOTHESIS` 或 `INFERENCE`，非經驗證不得賦予 `FACT_BASED` 標籤。

---

## 5. 治理與安全控制邊界 (Governance & Operational Boundaries)

### 5.1 治理三大維度分離 (AIA-17 規範)

[AIA-17](file:///home/wade/src/github/aia/docs/specifications/AIA-17-Operational-Governance-and-Human-Oversight-Specification.md) 要求在架構上徹底分離 Policy、Approval 與 Permission：

```text
候選行動 (Candidate Action)
     │
     ▼
┌──────────────────┐
│  Policy Engine   │ ──► 規則評估：判斷是否合規 (ALLOW / DENY / REQUIRE_APPROVAL)
└────────┬─────────┘
         │ 若需要審批
         ▼
┌──────────────────┐
│ Approval Service │ ──► 人類決策：掛起並等待使用者批准 (PENDING_APPROVAL)
└────────┬─────────┘
         │ 人類批准通過
         ▼
┌──────────────────┐
│ Permission Check │ ──► 環境權限：檢查沙盒/作業系統是否有實體執行權限
└────────┬─────────┘
         │ 權限齊備
         ▼
Runtime Worker 派發執行
```

- **Policy 責任**：純粹的邏輯規則判斷（如「單日 API 消耗超過 $5 美元需審核」、「修改代碼需審核」）。
- **Approval 責任**：人類互動生命週期（發出通知、超時自動過期、拒絕理由反饋）。
- **Permission 責任**：底層執行環境實體約束（Linux 檔案讀寫權限、沙盒隔離邊界、API Token 有效性）。獲得使用者 Approval **不等於** Agent 自動獲得系統 Root 或無限制檔案覆寫權限。

### 5.2 受控緊急停止機制 (Controlled Emergency Stop)

[AIA-17](file:///home/wade/src/github/aia/docs/specifications/AIA-17-Operational-Governance-and-Human-Oversight-Specification.md) 與 [AIA-19](file:///home/wade/src/github/aia/docs/specifications/AIA-19-Deployment-Operations-and-Observability-Specification.md) 對緊急停止的具體技術要求：
1. **停止派發**：Runtime 的 `Manager` 必須在 **1 秒內** 將全域派發器切換至 `EMERGENCY_STOP` 狀態，拒絕從隊列取出任何新 Task。此為 Runtime 層級的獨立閘道，不依賴 Agent 自身邏輯或 UI 操作。
2. **受控取消 (Graceful Cancellation)**：向所有執行中的 Worker Goroutine 發送 `ctx.Done()` 取消訊號。Worker 收到訊號後，依序執行：①將 Task 狀態持久化為 `CANCELLED`（防止重啟後重複派發）；②完成或回滾進行中的 SQLite 事務（防止半寫入損壞）；③向子處理程序發送 `SIGTERM` 並等待最多 10 秒——逾時強制 `SIGKILL`，確保無殭屍子程序。
3. **不可逆性與審批恢復**：Emergency Stop 觸發後，Agent 絕對無權自行重啟系統，必須由人類使用者在 CLI 執行 `aia resume --force` 或在 Web Console 顯式點擊「解除緊急停止」方可恢復。

---

## 6. 領域 Agent 演進與跨領域協作 (Domain Agents & Handoff)

### 6.1 共用 Autonomy Core 與領域隔離

根據 [AIA-00](file:///home/wade/src/github/aia/docs/specifications/AIA-00-Overall-Architecture-and-Evolution-Roadmap-of-Specification.md)、[AIA-05](file:///home/wade/src/github/aia/docs/specifications/AIA-05-Investment-Agent-Specification.md)、[AIA-12](file:///home/wade/src/github/aia/docs/specifications/AIA-12-Cross-Domain-Collaboration-and-Agent-Delegation-Specification.md) 與 [AIA-18](file:///home/wade/src/github/aia/docs/specifications/AIA-18-Autonomy-Core-Integration-and-End-to-End-Reference-Architecture.md)：
1. **共用自主核心**：主動性思維 (Initiative)、時間連續性 (Continuity)、自我修正 (Self-Correction) 與目標演化 (Goal Evolution) 統由 `internal/autonomy/` 提供，**嚴禁**在 Investment、Coding、Writing 各自實作一套自主引擎。
2. **領域專業化**：
   - **Investment Agent**：專業於市場觀察、虛擬部位記帳、預測校準與投資策略演化。
   - **Coding Agent**：專業於代碼庫分析、語法樹解析、測試用例修復與沙盒自修。
   - **Writing Agent**：專業於長篇小說創作、世界觀設定庫、人物關係與歷史年表校勘。

### 6.2 跨領域任務委派契約 (Cross-Domain Delegation Protocol)

[AIA-12](file:///home/wade/src/github/aia/docs/specifications/AIA-12-Cross-Domain-Collaboration-and-Agent-Delegation-Specification.md) 定義的跨領域協作標準資料結構：
```go
type DelegationRequest struct {
    ID               domain.ID
    RequesterAgentID domain.ID       // 發起委派的 Agent (如 Writing Agent)
    TargetAgentID    domain.ID       // 接收委派的 Agent (如 Coding Agent)
    SourceProjectID  domain.ID       // 原始專案 (如 小說創作專案)
    GoalID           domain.ID       // 關聯目標 (如 史料校勘自動化)
    TaskSpec         string          // 具體工具/能力需求描述
    AcceptanceGate   string          // 驗收標準 (如 包含單元測試與 CLI 呼叫範例)
    Status           DelegationStatus // REQUESTED, ACCEPTED, IN_PROGRESS, DELIVERED, REJECTED
}
```
委派成果交付後，發起端 Agent 必須進行成果驗收，確認新工具滿足原專案需求後，將其納入長期能力庫。

---

## 7. 稽核結論與實作指導準則 (Resolution Register Guidance)

本文件作為 **Tier 2 跨規格衝突裁決登記冊**，為後續程式碼實作、資料庫設計與測試驗收提供正式裁決依據，但**不取代 AIA-00～19 原始規格（Tier 1）的最終定義權**：

1. **資料模型設計**：以第 3 節（Goal/Project M:N 關係、四實體責任邊界）與第 4 節（Provenance 多型來源鏈、不強制外鍵）為依據產出 `data-model.md`。
2. **架構分層開發**：以第 2.3 節（11 步驟自主閉環，含獨立 Governance Check）與第 5 節（Policy/Approval/Permission 治理三分離）為依據實作 `internal/autonomy/` 與 `internal/policy/`。
3. **程式碼自修實現**：以第 2.1 節（Scope A/B/C 修改範圍、Correction Type C1~C6 修正分類、Validation Level V1~V6 驗證深度）為依據實作 Phase 6 代碼修改流水線。
4. **若本文件裁決與 Tier 1 規格存在新的矛盾**：應優先更新 Tier 1 規格，並同步修訂本文件，保持三層一致性。
