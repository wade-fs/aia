# AIA 資料庫遷移演進計畫書 (Database Migration Plan)

**文件代號：** AIA-DEV-MIG-01  
**適用資料庫：** SQLite (WAL 模式)  
**文件狀態：** Planned Architecture  
**依賴基準：** [docs/architecture/data-model.md](file:///home/wade/src/github/aia/docs/architecture/data-model.md)  
**最後更新：** 2026-11-01  

---

## 1. 遷移基本原則 (Migration Principles)

依據 SQLite 特性與 [AIA-03](file:///home/wade/src/github/aia/docs/specifications/AIA-03-Go-Runtime-Implementation-Specification.md) 第 2.3 節規範，AIA 資料庫遷移恪守以下原則：

1. **向前演進為主 (Forward-Only Migrations)**：SQLite 缺乏完整的 ALTER TABLE 支援，部分修改涉及暫存表重建。因此遷移以「向前相容擴充」與「自動冷備份」為主要安全防線，不強求撰寫脆弱的 Down Migration。
2. **交易完整性 (Transactional Schema Updates)**：DDL（`CREATE TABLE`、`ALTER TABLE`）及初始化 DML 必須包裝在單一 `BEGIN IMMEDIATE TRANSACTION` 內，任一失敗即全量回滾。注意：`PRAGMA foreign_keys`、`PRAGMA journal_mode` 等為**連線層級設定**，不可包含於同一 DDL 交易；WAL 模式為**資料庫層級設定**，應在啟動時於交易外設定。
3. **預遷移快照備份 (Pre-Migration Snapshot)**：在執行任何新 Migration 前，Go 驅動必須使用 **SQLite Backup API**（`sqlite3_backup_*`）建立帶有時間戳的完整備份副本。直接複製 `.db` 檔案在 WAL 模式下可能產生不一致；Backup API 能確保備份時資料庫完整性。備份完成後應執行完整性檢查（`PRAGMA integrity_check`），失敗備份需保留供事後鑑識。
4. **現有資產 100% 保持相容**：絕對不破壞 `001`～`003` 既存欄位。

---

## 2. 循序遷移路線圖 (Sequential Migration Roadmap)

```text
 001_initial_schema.sql (已完成) ──► 002_audit... (已完成) ──► 003_investment_core.sql (已完成)
                                                                     │
                                                                     ▼
 Phase 1: 004_event_history_and_provenance.sql (事件流與來源溯源)
   │
   ▼
 Phase 1: 005_goals_and_continuity.sql (目標樹、專案擴充與脈絡快照)
   │
   ▼
 Phase 2: 006_governance_and_approvals.sql (審批中心與治理稽核)
   │
   ▼
 Phase 3: 007_initiative_and_deliberation.sql (深思決策與候選工作)
   │
   ▼
 Phase 4: 008_learning_and_correction.sql (成效落差、RCA 與修正紀錄)
   │
   ▼
 Phase 5: 009_experiments_and_collaboration.sql (長週期實驗與跨領域委派)
```

---

## 3. 各階段遷移檔案規劃 (Detailed Migration Files)

### Phase 1 交付
- **`004_event_history_and_provenance.sql`**：
  - 建立 `event_history` (Append-only 歷史事件流)。
  - 建立 `sources`、`evidence`、`claims`、`claim_evidence_links`。
  - 擴充 `memories`（新增 `provenance_type`、`provenance_ref_type`、`provenance_ref_id`、`confidence`）。
- **`005_goals_and_continuity.sql`**：
  - 建立 `goals` (支援父子目標樹)、`goal_revisions`。
  - 建立 `goal_project_links` (Goal ↔ Project M:N 關聯表)。
  - 建立 `context_snapshots` (工作區狀態、未完成 Task、Pending Issues 快照)。
  - 擴充 `projects`（新增 `primary_goal_id`、`status`、`pending_issues_json`）。

### Phase 2 交付
- **`006_governance_and_approvals.sql`**：
  - 建立 `approvals` (申請者、操作類型、風險層級、過期時間、人類回覆)。
  - 建立 `policy_audit_logs` (規則評估歷程)。

### Phase 3 交付
- **`007_initiative_and_deliberation.sql`**：
  - 建立 `candidate_initiatives` (主動探索發現之候選工作)。
  - 建立 `decisions` (記錄 ACT / DEFER / MONITOR / ASK_HUMAN / NO_ACTION 決策動機)。
  - 建立 `decision_evidence_links`。
  - **注意**：Phase 1 已透過 `event_history` 記錄決策事件（`event_type = 'DECISION_MADE'`），以保持早期脈絡可回溯。`decisions` 完整實體從本 Migration 起正式建立，避免 Phase 1 過早引入尚未成熟的完整決策模型。

### Phase 4 交付
- **`008_learning_and_correction.sql`**：
  - 建立 `outcome_evaluations` (預期 vs 實際結果落差)。
  - 建立 `root_cause_analyses` (錯誤歸類與分析)。
  - 建立 `correction_actions` (Level 1~5 策略與工作流調優記錄)。

### Phase 5 交付
- **`009_experiments_and_collaboration.sql`**：
  - 建立 `experiments` (長期實驗實體、假設、基準線)。
  - 建立 `delegation_requests` (跨 Agent 委派契約與驗收狀態)。
