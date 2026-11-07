# AIA Volume 5: Runtime Engine, Operations & Interaction

**Document ID:** AIA-VOL-05  
**Title:** 運行時引擎、維運與人機互動規範 (Runtime Engine, Operations & Interaction Specification)  
**Status:** Normative Consolidated Baseline  
**Consolidates:** AIA-03, AIA-06, AIA-07, AIA-19, AIA-21, AIA-23 (含多維度工作運作政策、分階段執行、階層式配額與人機協作)  

---

## 1. Go Runtime 與背景 Worker 架構 (Go Runtime & Concurrency)

依據原 AIA-03 規範，AIA Runtime 採用 Go 1.23+ 打造，設計為高可靠、本機常駐服務（`systemd --user` 或 `launchd`）：

```text
 ┌─────────────────────────────────────────────────────────────┐
 │                      AIA Application Root                   │
 ├────────────────┬────────────────┬───────────────────────────┤
 │ TaskWorker     │ ScheduleWorker │ AutoWakeWorker            │
 │ (並行限制調度)  │ (15s 週期輪詢) │ (排程喚醒循環 / 門禁評估) │
 ├────────────────┴────────────────┴───────────────────────────┤
 │ LogRetentionPurger (逾期執行日誌自動清理，嚴禁清理不可篡改事件) │
 ├─────────────────────────────────────────────────────────────┤
 │ SQLite WAL 儲存引擎 (支援 Backup API 在線冷備份與交易隔離)  │
 └─────────────────────────────────────────────────────────────┘
```

- **TaskWorker**：由設定檔控制並行槽位（預設 3 Slots），避免 SQLite 寫入鎖競爭（`busy timeout`）與 CPU/Token 峰值。
- **ScheduleWorker (15s)**：定時掃描到期任務、週期性工作與喚醒評估。注意：每 15 秒 Polling 僅為資料庫掃描，不等於消耗 LLM 額度。
- **AutoWakeWorker**：依據台灣時區（`Asia/Taipei`）或事件驅動觸發自主評估循環（Autonomy Cycle），結合 Materiality Gate 決定是否進一步發起 LLM 呼叫。
- **LogRetentionPurger**：背景定期清理逾期的一般執行事件與非關鍵日誌（`execution_events`），**絕對禁止**清理不可竄改事實層（`event_history` 與合規治理稽核軌跡 `audit_logs`）。

---

## 2. Agent 多維度運作政策 (Agent Operating Policy & Multidimensional Framework)

依據原 AIA-23 與實務經驗，Agent 工作模式不應與底層 Runtime 執行形式或排程機制混為一談。AIA 嚴格劃分四大正交維度：

```text
Agent Operating Policy
│
├── 1. Work Mode (工作性質模式)
│   ├── Autonomous (獨立自主型，如 Investment、巡檢監控)
│   ├── Collaborative (人機協作型，如 Writing、Coding)
│   └── Dependency-driven (相依驅動型，如 Illustration、Video 生成)
│
├── 2. Trigger Policy (觸發政策)
│   ├── Time-based (定時/Cron 表達式)
│   ├── Event-based (事件觸發，如市場劇烈波動、PR 提交)
│   ├── Condition-based (條件達成，如前置任務完成)
│   └── Manual (使用者手動於 Web Console / CLI 觸發)
│
├── 3. Execution Form (執行形式)
│   ├── Interactive (即時阻塞式，支援 Clarification 對話)
│   ├── Background (Worktree / 背景非同步執行)
│   └── Batch (批次大規模掃描工單)
│
└── 4. Resource Policy (分層資源配額)
    ├── Token Budget (輸入/輸出/總 Token 上限)
    ├── API Call Budget (外部呼叫次數限制)
    ├── Time Budget (執行逾時限制)
    └── Concurrency Limit (並發槽位限制)
```

> **重要原則：工作性質優先且不直接綁死 Agent。**  
> Agent 是能力的提供者，Project 是成果的管理單位，Task 是原子工單，Work Mode 是具體執行的作業政策。同一 Agent 承接不同工單時可採用不同 Work Mode。

### 2.1 三大核心工作模式詳解

1. **Autonomous Mode（獨立自主型）**：
   - 具有明確目標、可自行獲取客觀資料與自律評估（如 Investment Agent 每日依市場交易時段自主喚醒 2～4 次）。
   - 受 Materiality Gate 保護：無重大資訊變化時維持原決策，零 LLM 消耗。
   - 自主決策不等於繞過治理，交易下單或重要行動仍須通過業務邏輯校驗或人工審批。

2. **Collaborative Mode（人機協作型）**：
   - 用於需要人類提供需求、釐清偏好、分階段驗收的工作（如 Writing、Coding Agent）。
   - 禁止將大型專案視為單次連續黑盒生成，必須實行 **工作階段化（Session-based Execution）**。
   - 階段範例（如 Coding）：`Requirements` → `Specification` → `Test Design` → `Implementation` → `Verification`。每個階段皆具備明確目標、產出物與驗收門禁。

3. **Dependency-driven Mode（相依驅動型）**：
   - 任務執行嚴格依賴上游工單產出（如小說章節定稿後驅動封面插畫或廣播劇音訊合成）。
   - 透過有向無環圖（DAG）管理依賴關係，上游未驗收前下游維持 `WAITING_DEPENDENCY`。

---

## 3. 工作階段化 (Session Management) 與 Checkpoint 恢復

長時間運行與人機協作工作必須切分成獨立的工作階段（Session）：

- **Session Context**：每個 Session 記錄 `Goal`, `Inputs`, `Target Artifacts`, `Token Quota` 與 `Status`。
- **Checkpoint / Resume**：
  - 在階段轉換、等待人類回覆（`WAITING_FOR_INPUT`）或等待安全審核（`WAITING_APPROVAL`）時，系統自動保存執行快照與 Checkpoint。
  - 當遇到系統崩潰、進程重啟或配額耗盡時，Worker 能由最近之有效 Checkpoint 恢復執行，而不需重新消耗全部 Context 重頭生成。

---

## 4. 分層 Quota 管理 (Hierarchical Resource Budgeting)

為防止單一 Agent 或異常迴圈耗盡平台額度，AIA 建立五層資源預算階層：

1. **System Budget**：全系統每日最大 Token、費用與週期總額度。
2. **Agent Budget**：單一 Agent 類型每日被分配之上限。
3. **Project Budget**：特定專案（如某本小說、某個代碼模組）的生命週期或階段預算。
4. **Task Budget**：單一工單的最大 Token 耗費用量。
5. **Session Budget**：單次人機互動階段的預算限制。

---

## 5. 統一 Task 狀態機與解除條件 (Unified Task State Machine)

系統全面統一 Task 狀態定義，杜絕模糊的 `QUEUED` 狀態，各狀態轉移遵循嚴密生命週期：

```text
               ┌──────────┐
               │  DRAFT   │
               └────┬─────┘
                    │ Submit
                    ▼
               ┌──────────┐
               │  READY   │ ◄─── Retry (FAILED -> READY) / 解除等待
               └────┬─────┘
                    │ Worker Claim (占用槽位)
                    ▼
               ┌──────────┐
      ┌────────┤ RUNNING  ├────────┐
      │        └────┬─────┘        │
      ▼             ▼              ▼
┌───────────┐ ┌───────────┐  ┌───────────┐
│  WAITING  │ │ COMPLETED │  │  FAILED   │
│  STATES   │ └───────────┘  └───────────┘
└─────┬─────┘
      │
      ├── WAITING_FOR_INPUT   (等待人類填寫 Clarification，釋放 Worker 槽位)
      ├── WAITING_APPROVAL    (等待安全或風控審批，釋放 Worker 槽位)
      ├── WAITING_DEPENDENCY  (等待前置 Task 完成，釋放 Worker 槽位)
      ├── WAITING_FOR_AUTH    (等待外部授權，釋放 Worker 槽位)
      └── PAUSED              (使用者或系統主動暫停，釋放 Worker 槽位)
```

- **槽位釋放原則**：任何進入 `WAITING_*` 或 `PAUSED` 的任務，必須立即釋放 TaskWorker 的並發執行槽位，不得阻礙其他 `READY` 任務的派發。
- **恢復排程原則**：當阻斷條件解除（如問答完成、審批通過、依賴滿足），Task 轉回 `READY`，等待 Worker 下一次排程調度。

---

## 6. 資料庫持久化、線上備份與可觀測性 (Database, Backup & Observability)

依據原 AIA-19 與 AIA-03 規範：
- **SQLite 交易與 WAL 模式**：嚴格實行讀寫分離與短交易原則。
- **線上熱備份 API**：利用 `sqlite3_backup_*` 實作零中斷備份（`aia backup create`），備份前進行 WAL Checkpoint，備份後執行 `PRAGMA integrity_check`。
- **分散式追蹤與指標（Telemetry）**：每個 Autonomy Cycle 產生唯一的 `TraceID` 與關聯 `SpanID`，記錄各步驟延遲、Token 消耗與錯誤事件。

---

## 7. 16 分頁 Web Console 與 CLI 操作 (Web Console & CLI)

依據原 AIA-06 與 AIA-07 規範：
- **Web Console (`127.0.0.1:7777`)**：提供 16 個專業監控分頁（總覽、任務、目標、週期、學習、實驗、委派、問題、投資、代碼、寫作、記憶、審計等）。
- **一鍵式緊急停機按鈕**：置於導航列顯眼位置，點擊即時發送停機廣播。
- **安全認證**：本地一次性 Token（OTT）登入，自動寫入 HttpOnly Cookie，無外部密碼洩漏風險。
- **Cobra CLI**：提供完整的終端運維子命令集（`agent`, `cycle`, `problem`, `backup`, `worker`, `doctor` 等）。

---

## 8. 通用需求綱要與人機釐清 (Universal Brief & Generic Clarification)

依據原 AIA-21 規範：

### 8.1 通用專案綱要 (Project Brief)
- 適用於所有領域（Writing, Coding, Investment, General）的標準 YAML 需求定義。
- **不可變欄位歷史（`brief_field_history`）**：每次修訂精確記錄修訂人、時間、原因及關聯 Task ID，禁止無聲覆寫。

### 8.2 阻斷式人機互動釐清 (Clarification Management)
- 當 Agent 在規劃或執行過程中面臨「關鍵資訊缺失」或「規範衝突」時，主動發起結構化提問（`ClarificationRequest`）。
- 嚴格遵守提問紀律（單次 ≤ 5 題），任務進入 `WAITING_FOR_INPUT` 狀態，釋放 Worker 槽位；待人類於 Web Console / CLI 答覆後自動回填並解除阻斷回到 `READY`。

