# AIA Volume 5: Runtime Engine, Operations & Interaction

**Document ID:** AIA-VOL-05  
**Title:** 運行時引擎、維運與人機互動規範 (Runtime Engine, Operations & Interaction Specification)  
**Status:** Normative Consolidated Baseline  
**Consolidates:** AIA-03, AIA-06, AIA-07, AIA-19, AIA-21 (含排程與工作模式)  

---

## 1. Go Runtime 與背景 Worker 架構 (Go Runtime & Concurrency)

依據原 AIA-03 規範，AIA Runtime 採用 Go 1.23+ 打造，設計為高可靠、本機常駐服務（`systemd --user` 或 `launchd`）：

```text
 ┌─────────────────────────────────────────────────────────────┐
 │                      AIA Application Root                   │
 ├────────────────┬────────────────┬───────────────────────────┤
 │ TaskWorker     │ ScheduleWorker │ AutoWakeWorker            │
 │ (3 slots 并發) │ (15s 週期輪詢) │ (台灣時區排程喚醒循環)    │
 ├────────────────┴────────────────┴───────────────────────────┤
 │ LogRetentionPurger (逾期日誌自動清理)                       │
 ├─────────────────────────────────────────────────────────────┤
 │ SQLite WAL 儲存引擎 (支援 Backup API 在線冷備份與交易隔離)  │
 └─────────────────────────────────────────────────────────────┘
```

- **TaskWorker (3 Slots)**：限流併發調度，避免 SQLite 寫入鎖競爭（`busy timeout`）與 CPU 峰值。
- **ScheduleWorker (15s)**：定時掃描到期任務與週期性工作。
- **AutoWakeWorker**：依據台灣時區（`Asia/Taipei`）設定每日自主喚醒時段，主動執行日常巡檢與實驗比對。
- **LogRetentionPurger**：背景定期清理逾期執行事件日誌，保持資料庫輕量敏捷。

---

## 2. Agent 四種運作模式 (Agent Operating Modes)

整合原草案中具備實質意義之工作模式分類，AIA 支援依任務性質動態切換：

1. **Interactive 互動模式**：由使用者在 Web Console 或 CLI 直接觸發，即時輸出結果，支援中途 Clarification 阻斷式對話。
2. **Autonomous Batch 批次模式**：處理大型分析工單（如 540 章小說審計、全庫代碼掃描），於背景 Worktree 靜默運行，完成後發出通知。
3. **Resident Daemon 常駐模式**：常駐背景持續監聽事件流、觀察工作區變更並維持時間連續性。
4. **Scheduled 週期排程模式**：基於 Cron 表達式定時喚醒（如台股開盤前 08:30 數據抓取、每日午夜日誌清理）。

---

## 3. 資料庫持久化、線上備份與可觀測性 (Database, Backup & Observability)

依據原 AIA-19 與 AIA-03 規範：
- **SQLite 交易與 WAL 模式**：嚴格實行讀寫分離與短交易原則。
- **線上熱備份 API**：利用 `sqlite3_backup_*` 實作零中斷備份（`aia backup create`），備份前進行 WAL Checkpoint，備份後執行 `PRAGMA integrity_check`。
- **分散式追蹤與指標（Telemetry）**：每個 Autonomy Cycle 產生唯一的 `TraceID` 與關聯 `SpanID`，記錄各步驟延遲、Token 消耗與錯誤事件。

---

## 4. 16 分頁 Web Console 與 CLI 操作 (Web Console & CLI)

依據原 AIA-06 與 AIA-07 規範：
- **Web Console (`127.0.0.1:7777`)**：提供 16 個專業監控分頁（總覽、任務、目標、週期、學習、實驗、委派、問題、投資、代碼、寫作、記憶、審計等）。
- **一鍵式緊急停機按鈕**：置於導航列顯眼位置，點擊即時發送停機廣播。
- **安全認證**：本地一次性 Token（OTT）登入，自動寫入 HttpOnly Cookie，無外部密碼洩漏風險。
- **Cobra CLI**：提供完整的終端運維子命令集（`agent`, `cycle`, `problem`, `backup`, `worker`, `doctor` 等）。

---

## 5. 通用需求綱要與人機釐清 (Universal Brief & Generic Clarification)

依據原 AIA-21 規範：

### 5.1 通用專案綱要 (Project Brief)
- 適用於所有領域（Writing, Coding, Investment, General）的標準 YAML 需求定義。
- **不可變欄位歷史（`brief_field_history`）**：每次修訂精確記錄修訂人、時間、原因及關聯 Task ID，禁止無聲覆寫。

### 5.2 阻斷式人機互動釐清 (Clarification Management)
- 當 Agent 在規劃或執行過程中面臨「關鍵資訊缺失」或「規範衝突」時，主動發起結構化提問（`ClarificationRequest`）。
- 嚴格遵守提問紀律（單次 ≤ 5 題），任務進入 `WAITING_FOR_INPUT` 狀態，待人類於 Web Console / CLI 答覆後自動回填並解除阻斷。
