# AIA 跨規格一致性決議登記冊 (Cross-Specification Resolution Register)

**文件代號：** AIA-SPEC-AUDIT-01  
**稽核範圍：** [docs/specifications/](file:///home/wade/src/github/aia/docs/specifications) (AIA-00 ～ AIA-21)  
**稽核基準：** v2.0 系列規範  
**核心目標：** 消除跨規格名詞衝突、責任模糊、狀態機脫節與資料模型歧異，記錄正式衝突裁決；不取代 AIA-00～21 原始規格。  
**現有代碼基準：** Git commit `7597902` (已內建 Migration Version 1 ～ 6)  
**最後更新：** 2026-11-01  

---

## 0. 文件權威層級與使用準則 (Authority Tiers)

本文件在 AIA 規格體系中的定位如下：

| 層級 | 文件 | 職責 |
|:---:|---|---|
| **Tier 1（最高）** | AIA-00 ～ AIA-19（共 20 份原始規格） | 系統性質、能力邊界與設計意圖的最終定義來源 |
| **Tier 2** | **本文件（AIA-SPEC-AUDIT-01）** | 記錄跨規格衝突的正式裁決結果；不新增原始規格未涵蓋的能力 |
| **Tier 3** | [docs/development/roadmap.md](file:///home/wade/src/github/aia/docs/development/roadmap.md)（Master Implementation Roadmap） | 以 Tier 1+2 為依據，規劃具體實作路線與驗收標準 |

> [!IMPORTANT]
> 本文件的每一項裁決**僅解決跨規格之間的矛盾與工程落地歧異**，所有裁決均以 Tier 1 原始規格的設計意圖為根本依據。若未來 Tier 1 規格修訂，本文件須同步更新。

---

## 1. 稽核背景與核心發現摘要 (Executive Summary)

AIA 規格體系（AIA-00 至 AIA-19，共計 20 份文件、27,500+ 行）完整構建了一套先進的長期自主實驗架構。在進行具體程式開發前，本稽核針對跨規格的實體邊界、自主循環、治理模型與知識溯源進行了全量交叉比對，共辨識出 **10 項關鍵名詞與模型歧異**，並在此提出權威統一判定（Normative Resolutions）：

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   規格交叉稽核關鍵歧異與統一判定                      │
├───────────────────┬───────────────────┬────────────────────────────────┤
│ 稽核項目          │ 涉及衝突規格      │ 權威統一判定 (Authoritative)   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 1. 自修分級名詞   │ AIA-00 vs AIA-14  │ 修改範圍採 Scope A/B/C；       │
│                   │                   │ 修正類型採 FACT/STRATEGY等；   │
│                   │                   │ 授權等級採 C0～C5；            │
│                   │                   │ 驗證深度採 V1～V6              │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 2. 治理分層體系   │ AIA-17 vs AIA-18  │ G0~G3 專指治理層級，           │
│                   │                   │ 不用於實作階段命名             │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 3. 自主核心閉環   │ AIA-00/13 vs      │ 採 AIA-18 之 11 步驟閉環，     │
│                   │ AIA-18            │ 將 Governance Check 顯式列出   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 4. 實體責任邊界   │ AIA-09/10/15/18   │ Goal ↔ Project 採 M:N 關聯；   │
│                   │                   │ 引入 Autonomy Cycle 核心樞紐   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 5. 知識溯源關聯   │ AIA-10 vs AIA-16  │ Memory 不強制外鍵，解構為性質、│
│                   │                   │ 錨點與驗證狀態 (VERIFIED 等)   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 6. 緊急停止規範   │ AIA-17 vs AIA-19  │ 1秒凍結派發，Task 標記 CANCEL，│
│                   │                   │ 事務安全，子程序 10 秒受控終止 │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 7. 三事件軌職責   │ AIA-03 vs AIA-10  │ execution_events (工單微觀) vs │
│                   │ vs AIA-17         │ event_history (系統/業務宏觀)  │
│                   │                   │ vs audit_logs (受治理操作)     │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 8. 稽核日誌架構   │ AIA-17 vs 既有庫  │ 擴充既有 audit_logs 表，       │
│                   │                   │ 不開闢重複 policy_audit_logs   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 9. 決策模型落地   │ AIA-08 vs AIA-10  │ decisions 於 Phase 1 落地模型，│
│                   │                   │ 支援 NO_ACTION；Phase 3 推演   │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 10. Evidence 分層 │ AIA-05 vs AIA-16  │ 通用知識層與領域特化層垂直引用 │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ 11. 記憶權威分層  │ AIA-10 vs AIA-16  │ 四種記憶劃分(情節/語意/程序/工作)│
│     與治理體系    │ vs AIA-18         │ 五層權威單向約束(L1~L5)；      │
│                   │                   │ Autonomy Core 提供共用治理機制 │
└───────────────────┴───────────────────┴────────────────────────────────┘
```

---

## 2. 核心名詞與分級對齊 (Terminology Harmonization)

### 2.1 自我修改分級名詞衝突 (Self-Modification Multi-Namespace Alignment)

#### 【衝突現象】
- [AIA-00](file:///home/wade/src/github/aia/docs/specifications/AIA-00-Overall-Architecture-and-Evolution-Roadmap-of-Specification.md) 第 5.13 節將自修分為：`Level 1 (行為修正)`、`Level 2 (能力修正)`、`Level 3 (核心系統修正)`。
- [AIA-11](file:///home/wade/src/github/aia/docs/specifications/AIA-11-Self-Correction-and-Learning-Specification.md) 第 3 節定義了 `C0～C5` 授權等級。
- [AIA-14](file:///home/wade/src/github/aia/docs/specifications/AIA-14-Self-Modification-and-Safe-Code-Evolution-Specification.md) 第 2 節定義了 `Level A/B/C`（修改範圍），第 5 節又定義了 `Level 1~6`（驗證深度）。
- 既有計畫容易將「修正內容 (如策略修正)」、「授權等級 (如 C3)」與「驗證深度 (如 V3)」混用，造成實作混亂。

#### 【權威統一判定】
**解構為四個獨立且正交的命名空間**：

1. **修改對象範圍 (Modification Scope)**：一律採用 `Scope A / B / C`（對應 AIA-14 第 2 節）：
   - **`Scope A (Domain Scope)`**：領域工具、採集器、外掛腳本、Prompt。
   - **`Scope B (Core Scope)`**：AIA Runtime、Autonomy Core、共用 Service、Worker。
   - **`Scope C (Governance Scope)`**：Policy Engine、Approval Engine、Audit Log。**Agent 絕對禁止自行修改**。

2. **修正類型 (Correction Type)**：說明「修正的是什麼對象」：
   - **`FACT` (事實修正)**：更新知識網中陳舊的 Claim 或事實。
   - **`JUDGMENT` (判斷修正)**：修正權重假設與信心指標。
   - **`STRATEGY` (策略修正)**：調校決策啟發式規則（如停損比例）。
   - **`PROMPT` (提示詞調優)**：微調 Agent 領域 Prompt。
   - **`WORKFLOW` (工作流改善)**：調整任務拆解步驟或排程頻率。
   - **`CODE` (程式碼自修)**：修正具體演算法或工具程式碼。

3. **修正授權等級 (Correction Authorization Level)**：一律採用 `C0 ～ C5`（對應 AIA-11 第 3 節）：
   - **`C0`**：自主安全修正（唯讀或低風險事實校正，無須通知）。
   - **`C1`**：低風險修正（後通知人類）。
   - **`C2`**：專案範疇核准（需專案擁有者確認）。
   - **`C3`**：策略層級核准（重大交易/分析策略變更需顯式批准）。
   - **`C4`**：系統行為核准（工作流或跨專案行為變更）。
   - **`C5`**：核心架構/代碼最高核准（Scope B 代碼變更強制最高審批）。

4. **驗證流水線深度 (Validation Level)**：一律採用 `V1 ～ V6`（對應 AIA-14 第 5 節）：
   - **`V1 (Static)`**：靜態分析與編譯檢查。
   - **`V2 (Unit)`**：單元測試。
   - **`V3 (Integration)`**：整合測試。
   - **`V4 (Regression)`**：回歸測試。
   - **`V5 (Behavioral)`**：行為模擬與端到端評估。
   - **`V6 (Benchmark)`**：效能基準與自主性指標評估。

---

### 2.2 治理層級 (Governance Tiers) 語意收斂

#### 【衝突現象】
- [AIA-17](file:///home/wade/src/github/aia/docs/specifications/AIA-17-Operational-Governance-and-Human-Oversight-Specification.md) 定義了 `G0~G3` 治理層級，但部分段落混稱「Phase G0~G3」，容易與系統實作 Roadmap 混淆。

#### 【權威統一判定】
1. **`G0` 至 `G3` 專指「治理架構層級 (Governance Tiers)」**：
   - **`G0 (System Governance)`**：系統底層不可逾越的安全邊界（白名單目錄、禁止真實資金交易、禁止自改審批規則）。
   - **`G1 (Owner Governance)`**：系統擁有者（人類使用者）的顯式授權、配置與審批。
   - **`G2 (Autonomy Governance)`**：Autonomy Core 在產生候選工作與深思決策時的策略邊界。
   - **`G3 (Execution Governance)`**：Worker 與 Task 執行期間的並行槽位、資源消耗配額 (Token/CPU Budget) 與逾時控制。
2. 系統實作階段一律統一遵循 [docs/development/roadmap.md](file:///home/wade/src/github/aia/docs/development/roadmap.md) 的 **Phase 0 至 Phase 6**。

---

### 2.3 自主閉環步驟數統一 (Autonomy Cycle Steps)

#### 【衝突現象】
- [AIA-00](file:///home/wade/src/github/aia/docs/specifications/AIA-00-Overall-Architecture-and-Evolution-Roadmap-of-Specification.md) 與 [AIA-13](file:///home/wade/src/github/aia/docs/specifications/AIA-13-Autonomy-Evaluation-and-Benchmarking-Specification.md) 描述為 10 步驟循環。
- [AIA-18](file:///home/wade/src/github/aia/docs/specifications/AIA-18-Autonomy-Core-Integration-and-End-to-End-Reference-Architecture.md) 第 11 節將其細化為 11 個步驟，在 `Decide` 與 `Execute` 之間顯式插入了 `Governance Check`。

#### 【權威統一判定】
**以 AIA-18 的 11 步驟為全系統權威標準**：
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
 7. Governance Check (Policy 規則驗證、風險分級、必要審批申請 ── 獨立於 Agent 決策之外)
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

## 3. 實體責任邊界與自主循環樞紐 (Entity Responsibility Matrix)

### 3.1 Goal vs Project vs Experiment vs Task 邊界定義

| 實體名稱 | 核心回答的問題 | 主要生命週期責任 | 產出物與關聯 |
|---|---|---|---|
| **Goal (目標)** | **「我們長期期望達成什麼狀態？」** | 描述期望達成的業務/系統狀態。支援層級樹 (Parent/Child)、指標 (Metrics) 與優先級。具備修訂 (Revision) 歷程。 | 一個 Goal 可由多個 Project 共同追求（M:N 關係，透過 `goal_project_links`）。不直接執行命令。 |
| **Project (專案)** | **「我們正在哪個長期環境中工作？」** | 長期工作領域與成果的容器。維護工作區目錄、專案進度、人物/代碼/投資組合設定、待決事項 (Pending Issues)。 | 可關聯多個 Goal；以 `primary_goal_id` 標示主要目標；包含多個 Experiment 與 Task。 |
| **Experiment (實驗)** | **「我們要驗證什麼假設或策略？」** | 具備科研性質的驗證單元。維護假設 (Hypothesis)、基準線 (Baseline)、對照變因、評估週期與結論。 | 隸屬於特定 Project，服務一至多個 Goal（M:N）；完整對齊 AIA-15 的 11 狀態生命週期。 |
| **Task (任務)** | **「現在具體要執行什麼工單？」** | 具備原子性、可派發給 Worker 執行的工單。維護狀態機 (`READY`, `RUNNING`, `COMPLETED`, `WAITING_APPROVAL`)。 | 隸屬於 Project 或 Experiment；執行完畢即行結束。 |

---

### 3.2 引入核心樞紐：Autonomy Cycle (自主循環實體)

自主性不是單次 Task 執行，也不是靜態資料。引入 `autonomy_cycles` 表作為將 Goal、Observation、Decision、Execution、Evaluation 串聯的時間軸樞紐：

```text
       ┌────────────────────────┐
       │     Autonomy Cycle     │ (自主循環核心樞紐)
       └───────────┬────────────┘
                   │
    ┌──────────────┼──────────────┬──────────────┐
    ▼              ▼              ▼              ▼
Context Snapshot  Decisions     Tasks      Outcome Evaluations
(啟動時環境快照) (深思決策紀錄) (執行工單)   (成效比對與 RCA)
```

Cycle 解答 9 大核心問題：①為什麼開始？②當時看到了什麼？③回憶了哪些先驗資訊？④發現什麼問題？⑤評估過哪些選項？⑥最後為什麼採取或不採取行動？⑦執行結果如何？⑧學到了什麼？⑨下次循環應檢查什麼？

---

### 3.3 決策實體 (Decision) 的生命週期與落地時機

- **權威統一判定**：
  - `decisions` 是跨 Session 脈絡與治理的核心樞紐，可關聯 Goal, Project, Experiment, Evidence, Deliberation 與 Outcome，且**必須原生支援 `NO_ACTION`**。
  - **資料模型於 Phase 1 (Migration 008) 即行落地**，確保跨 Session 脈絡重建與 Phase 2 審批有結構化實體可關聯。
  - **Phase 3 實作自主決策推演引擎 (Deliberation Engine)**，負責在無人介入時自動生成決策。

---

## 4. 知識溯源、記憶與證據分層 (Knowledge, Provenance & Evidence)

### 4.1 核心資料類型分工 (AIA-16 權威定義)

```text
外部輸入 ──► [Data] ──(提取)──► [Evidence] ──(支持/反駁)──► [Claim]
                                                               │
                                                               ▼
[Decision] ◄──(依據 Context)── [Knowledge] ◄──(驗證晉升)── [Assertion]
                                   ▲
                                   │ (標註不確定性)
                [Inference] / [Hypothesis]
```

### 4.2 記憶 Provenance 模型邏輯解構

- **衝突/矛盾現象**：既有文件曾出現預設 `provenance_type = INFERENCE` 且 `provenance_ref_type = NONE`，卻又規定「若 provenance_type ≠ NONE 則 ref_id 不得為空」的規則衝突。
- **權威統一判定**：徹底分離三個概念：
  1. `provenance_type`：知識性質（`FACT_BASED | INFERENCE | HYPOTHESIS | USER_PREFERENCE | AGENT_DECISION`）。
  2. `provenance_ref_type`：錨點實體類型（`EVIDENCE | SOURCE | MEMORY | CONTEXT_SNAPSHOT | DECISION | NONE`）。
  3. `provenance_status`：驗證狀態（`VERIFIED | PARTIAL | UNVERIFIED`）。
  - 允許暫時缺少錨點，但未經檢驗的記憶只能是 `UNVERIFIED` 或 `PARTIAL`，**嚴禁標記為 `VERIFIED`**。

### 4.3 多維度 Source 評估與 Claim 修訂歷史

1. **Source 五維評估**：來源信賴度不採單一浮點數，拆分為：
   - `authority_score`（權威性）
   - `directness_score`（第一手直達性）
   - `timeliness_score`（時效性）
   - `independence_score`（利益獨立性）
   - `reproducibility_score`（可重現性）
2. **Claim 修訂歷史 (`claim_revisions`)**：系統禁止直接覆寫已採納之主張，每次修改必須保存舊版本陳述、修訂理由及反對/支持 Evidence。

### 4.4 通用知識層與領域 Evidence 分層

- **通用知識層**：`sources`、`evidence`、`claims`、`claim_revisions` 提供跨領域事實底座。
- **領域特化層**：既有 `investment_decision_evidence` 與 `investment_raw_documents` 保留金融專屬欄位，透過外鍵或關聯引用共用 Evidence，互補而不競爭。

### 4.5 一致性記憶架構與五層權威體系 (Consistent Memory Architecture & Authority Tiers)
依據 [docs/architecture/memory.md](file:///home/wade/src/github/aia/docs/architecture/memory.md)，解決 AIA-10、AIA-16 與 AIA-18 對記憶責任定義分散之問題：

1. **四種記憶明確劃分 (4 Memory Types)**：
   - **情節記憶 (Episodic)**：曾經發生的事件、決策與客觀結果，歷史不可竄改。
   - **語意記憶 (Semantic)**：已知知識與主張，依新證據版本化修訂。
   - **程序記憶 (Procedural)**：技能、規則與評估方法。
   - **工作記憶 (Working)**：當前專案與任務進行狀態。
2. **五層權威體系 (L1～L5)**：
   - `L1 Event History` (最高權威事實) ➔ `L2 Evidence & Provenance` (來源與證據) ➔ `L3 Knowledge & Memory` (認知與主張版本) ➔ `L4 Context & Project State` (專案狀態與未決問題) ➔ `L5 Active Context` (當前動態召回)。
   - **核心鐵律**：歷史不可改寫，理解可以版本化修正（保留完整因果鏈：`原始決策 ➔ 實際結果 ➔ 重新分析 ➔ 修正認知 ➔ 後續驗證`）。
3. **共用記憶治理核心 (Memory Integrity & Governance)**：
   - 由 Autonomy Core 統一負責 9 大記憶治理職責（寫入控制、分類、來源追溯、時效管理、衝突偵測、修復對齊、脈絡重建、一致性審計、關聯召回），供三大領域 Agent 共用。
   - 終極檢驗標準：**不失憶、不亂記、不亂改、不混淆、不停止學習**。

---

## 5. 治理、安全與三軌事件體系 (Governance & Triple-Event Streams)

### 5.1 治理三大維度分離 (AIA-17 規範)

- **Policy 責任**：純粹的邏輯規則判斷（如「單日 API 消耗超過 $5 美元需審核」）。
- **Approval 責任**：人類互動生命週期（掛起、通知、超時過期、批准/駁回）。
- **Permission 責任**：底層執行環境實體約束（Linux 檔案權限、沙盒隔離邊界、API Token 有效性）。

### 5.2 受控緊急停止機制 (Controlled Emergency Stop)

1. **停止派發**：Runtime 的 `Manager` 必須在 **1 秒內** 將全域派發器切換至 `EMERGENCY_STOP` 狀態，凍結新 Task 取出。
2. **受控取消**：向執行中的 Worker Goroutine 發送 `ctx.Done()` 取消訊號。Worker 依序執行：①將 Task 狀態持久化為 `CANCELLED`；②完成或回滾進行中的 SQLite 事務；③發送 `SIGTERM` 至子程序並等待最多 10 秒（逾時強制 `SIGKILL`）。
3. **不可逆性**：Agent 無權自行重啟，必須由人類執行 `aia resume --force` 或在 Web Console 顯式解除。

### 5.3 三軌事件分離與統一稽核日誌

1. **三事件軌職責獨立**：
   - **`execution_events`**：單次工單 micro-level 執行歷程（Worker 內部）。
   - **`event_history`**：全系統 macro-level 業務與自主性事件流（具備 Hash Chain 與 Trigger 防竄改，Continuity 第一事實來源）。
   - **`audit_logs`**：全系統受治理的高風險操作與審批日誌。
2. **稽核日誌不建重複表**：擴充既有 `audit_logs` 表（增加 `policy_id`、`decision_id`、`approval_id`、`risk_tier`），既有 AuditRepository 與 Web/CLI 介面 100% 沿用。

### 5.4 Event History 不可竄改性與防竄改邊界

1. 欄位補齊：`event_id`, `event_version`, `aggregate_version`, `correlation_id`, `causation_id`, `payload_hash`, `previous_hash`, `occurred_at` (業務發生時間) vs `recorded_at` (系統寫入時間)。
2. SQLite 觸發器：`trg_event_history_no_update` 與 `trg_event_history_no_delete`。
3. 邊界宣告：Trigger 與 Hash Chain 提供**應用層保護與事後竄改偵測**，不保證防禦直接具備本機資料庫檔案 root 修改權限者。
4. 交易一致性：狀態更新與 Event 寫入必須在同一個 SQLite 交易內完成。

---

## 6. 領域 Agent 演進與跨領域協作 (Domain Agents & Handoff)

### 6.1 共用 Autonomy Core 與領域隔離
- **共用自主核心**：主動性思維 (Initiative)、時間連續性 (Continuity)、自我修正 (Self-Correction) 與目標演化 (Goal Evolution) 統由 `internal/autonomy/` 提供，**嚴禁**各自造輪子。
- **領域專業化**：Investment（金融市場與資產管理）、Coding（代碼庫與測試）、Writing（小說與歷史考勘）。

### 6.2 跨領域任務委派契約 (Cross-Domain Delegation Protocol)
定義標準 `DelegationRequest`，由發起端 Agent 進行交付成果之驗收閉環。

---

## 7. 稽核結論與實作指導準則 (Resolution Register Guidance)

本文件作為 **Tier 2 跨規格衝突裁決登記冊**，為後續程式碼實作、資料庫設計與測試驗收提供正式裁決依據，但**不取代 AIA-00～21 原始規格（Tier 1）的最終定義權**：

1. **資料模型設計**：以第 3 節（Autonomy Cycle 樞紐、Goal/Project M:N 關係）與第 4 節（多維來源、解構 Provenance 模型）為依據落地 `data-model.md`。
2. **架構分層開發**：以第 2.3 節（11 步驟自主閉環）與第 5 節（Policy/Approval/Permission 治理三分離、三軌事件）為依據實作 `internal/autonomy/` 與 `internal/policy/`。
3. **程式碼自修實現**：以第 2.1 節（Scope A/B/C、Correction Type、C0～C5 授權、V1～V6 驗證深度）為依據實作自我修正流水線。
4. **需求與專案管理**：以第 8 節（AIA-21 Brief & Clarification 跨規格裁決）為依據實作獨立需求模型、agy 上下文組裝管線與跨 Session 延續。

---

## 8. AIA-21 與 AIA-00～20 跨規格一致性稽核與權威裁決 (Project Brief, Clarification, Isolation & Continuation)

### 8.1 專案需求模型 (Project Brief) 與目標 (Goal) 的責任劃分 (AIA-09 vs AIA-21)

#### 【衝突與混淆現象】
- [AIA-09](file:///home/wade/src/github/aia/docs/specifications/AIA-09-Initiative-and-Goal-Evolution-Specification.md) 定義了 Goal 的生命週期、修訂與度量指標。
- [AIA-21](file:///home/wade/src/github/aia/docs/specifications/AIA-21-Project-Brief-and-Clarification-Management-Specification.md) 定義了 Project Brief、Field Catalog、Field State（`UNKNOWN`、`DEFAULTED`、`ANSWERED` 等）與版本演進。
- 若兩者邊界不清，容易出現將 Brief 當作第二個 Goal 系統，或將 Goal 的演進與 Brief 的欄位填補混淆。

#### 【權威統一判定】
1. **Goal 描述長期願景與成效標準（What & Why Long-term）**：
   - 解決「期望系統/專案長期達到什麼宏觀狀態」，具備層級樹、KPI 指標與演化歷程。
2. **Brief 描述專案具體運作需求與參數規格（How & Constraints Specification）**：
   - 解決「當前專案的具體邊界、未決參數、風格、假設與驗收條件」。
   - 一個 Project 擁有一個當前生效的 Brief（`brief.yaml` + SQLite 索引）。
   - 當 Brief 的重大限制發生顛覆性改變時，觸發 Goal 的評估（`REASSESS`），但 Brief 不取代 Goal。

---

### 8.2 澄清請求 (Clarification) 與治理審批 (Human Approval) 的邊界 (AIA-17 vs AIA-21)

#### 【衝突現象】
- [AIA-17](file:///home/wade/src/github/aia/docs/specifications/AIA-17-Operational-Governance-and-Human-Oversight-Specification.md) 定義了安全與風控審批（`WAITING_APPROVAL`），用於阻止越權操作（如高額花費、代碼自修合併）。
- [AIA-21](file:///home/wade/src/github/aia/docs/specifications/AIA-21-Project-Brief-and-Clarification-Management-Specification.md) 引入澄清問答（`WAITING_FOR_INPUT`），用於向使用者獲取創作偏好或業務參數。
- 兩者若混同，會使需求提問繞過治理，或讓普通提問引發繁重的風控審批。

#### 【權威統一判定】
1. **徹底分離互動性質與 Task 狀態**：
   - **`WAITING_FOR_INPUT` (需求釐清)**：Agent 不確定使用者偏好或缺少必要輸入時發起。使用者回覆後更新 Brief，不涉及系統安全政策核准。
   - **`WAITING_APPROVAL` (安全治理)**：Agent 提議之動作觸發了 Policy 規則（如金融交易、Scope B 自修、Token 預算超支）。必須由授權者審批，不屬於 Brief 欄位填充。
2. **Clarification 絕對不可用於繞過 Policy 檢驗**：
   - 使用者即使在澄清問答中指示「允許任意修改代碼」，系統仍必須強制通過 AIA-17 的 G0/G1 治理審查。

---

### 8.3 專案絕對隔離與零寫死預設原則 (Project Isolation & Zero Hardcoded Fallbacks)

#### 【衝突與反模式現象】
- 歷史過渡代碼曾在找不到專案時預設退回 `"yuan"`（元朝小說），導致新專案（如徒步環島、投資分析、唐代小說）在 ID 缺漏時污染或讀寫 `"yuan"` 專案。
- [AIA-02](file:///home/wade/src/github/aia/docs/specifications/AIA-02-System-Architecture.md)、[AIA-10](file:///home/wade/src/github/aia/docs/specifications/AIA-10-Continuity-and-Long-Term-Project-Management-Specification.md) 與 [AIA-21](file:///home/wade/src/github/aia/docs/specifications/AIA-21-Project-Brief-and-Clarification-Management-Specification.md) 明確要求 Project 必須完全獨立隔離。

#### 【權威統一判定】
1. **全面禁止程式碼寫死任何實體專案 ID**：
   - 核心程式、CLI、Web Console、Observer 中**嚴格禁止**出現 `targetID = "yuan"` 等寫死回退邏輯。
   - 若操作命令未指定 `--project`，且系統無法明確推斷唯一現行專案，必須回傳顯式錯誤（`project ID is required`），絕不允許任意猜測或降級至預設專案。
2. **工作區實體目錄完全隔離**：
   - 每個 Project 擁有專屬 Workspace，其 `brief.yaml`、`manifest.yaml`、`memory/`、`decisions/` 完全隔離，不可跨專案共用未命名空間的目錄。

---

### 8.4 agy 執行上下文組裝管線 (Context Assembly Pipeline) (AIA-03 vs AIA-18 vs AIA-21)

#### 【缺漏現象】
- [AIA-03](file:///home/wade/src/github/aia/docs/specifications/AIA-03-Go-Runtime-Implementation-Specification.md) 與 [AIA-18](file:///home/wade/src/github/aia/docs/specifications/AIA-18-Autonomy-Core-Integration-and-End-to-End-Reference-Architecture.md) 定義了 Runtime Worker 與 agy Process 的啟動，但原先只將 `Task.Description` 作為 Prompt 傳入，遺漏了 Agent Profile、Domain Rules、Brief 與 Project Memory。

#### 【權威統一判定】
**agy 啟動時必須由 Context Assembler 統一組裝完整執行上下文**：
```text
┌─────────────────────────────────────────────────────────────┐
│                     Context Assembler                       │
├─────────────────────────────────────────────────────────────┤
│ 1. Agent Profile:  系統人設、責任邊界、溝通風格 (SystemPrompt)│
│ 2. Domain Rules:   領域專屬規則、驗證邊界、規範指引         │
│ 3. Project Brief:  當前有效 Brief（含已確認需求、假設與缺口）│
│ 4. Project Memory: 最近情節決策、語意主張與專案歷史記憶    │
│ 5. Task Goal:      當前 Task 的具體目標、背景與驗收標準     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                 傳入 agy Provider Session
```
禁止發送缺乏專案脈絡的孤立 Prompt 給 agy。

---

### 8.5 跨 Session 延續性 (Continuation Execution) (AIA-10 vs AIA-21)

#### 【衝突現象】
- 傳統聊天模式假設 LLM 的 Session 持續存活；當行程中斷或使用者隔日回答問題時，因 Session 遺失導致專案無法繼續。

#### 【權威統一判定】
1. **專案連續性錨定於 SQLite 與檔案系統，絕不依賴 LLM Session**：
   - Task 進入 `WAITING_FOR_INPUT` 時，釋放 Worker 執行槽位。
   - 使用者透過 CLI 或 Web 提交回答後，AIA 驗證答案、更新 Brief 版本，並將 Task 恢復為 `READY`。
   - 新發起的 Execution 載入更新後的 Brief 與 Project Memory，無縫延續工作。

