# AIA 資料庫遷移演進計畫書 (Database Migration Plan)

**文件代號：** AIA-DEV-MIG-01  
**適用資料庫：** SQLite (WAL 模式)  
**文件狀態：** Planned Architecture  
**依賴基準：** [docs/architecture/data-model.md](file:///home/wade/src/github/aia/docs/architecture/data-model.md)  
**現有代碼基準：** Git commit `7597902` (已內建 Migration Version 1 ～ 6)  
**最後更新：** 2026-11-01  

---

## 1. 遷移基本原則 (Migration Principles)

依據 SQLite 特性與 [AIA-03](file:///home/wade/src/github/aia/docs/specifications/AIA-03-Go-Runtime-Implementation-Specification.md) 第 2.3 節規範，AIA 資料庫遷移恪守以下原則：

1. **現有資產 100% 保持相容 (Commit 7597902 凍結保護)**：
   - 既有 Migration Version 1 ～ 6（`initial_schema`、`audit_request_source_metadata`、`investment_core`、`investment_data_pipeline`、`investment_predictions_and_evaluations`、`schedules_and_automation`）已在代碼庫中運作正常。
   - **所有後續遷移必須從 Version 7 開始依序編號**，嚴禁修改或覆寫 001～006 既有結構。
2. **向前演進為主 (Forward-Only Migrations)**：
   - SQLite 缺乏完整的 ALTER TABLE 支援，部分修改涉及暫存表重建。因此遷移以「向前相容擴充」與「自動冷備份」為主要安全防線，不強求撰寫脆弱的 Down Migration。
3. **交易完整性與 PRAGMA 邊界分明**：
   - DDL（`CREATE TABLE`、`ALTER TABLE`）及初始化 DML 必須包裝在單一 `BEGIN IMMEDIATE TRANSACTION` 內，任一失敗即全量回滾。
   - `PRAGMA foreign_keys`、`PRAGMA journal_mode` 等為**連線層級設定**，不可包含於同一 DDL 交易；WAL 模式為**資料庫層級設定**，在啟動時於交易外設定。
4. **預遷移快照備份 (SQLite Backup API)**：
   - 在執行任何新 Migration 前，Go 驅動必須使用 **SQLite Backup API**（`sqlite3_backup_*`）建立帶有時間戳的完整備份副本。直接複製 `.db` 檔案在 WAL 模式下可能產生不一致；Backup API 能確保備份時資料庫完整性。備份完成後應執行完整性檢查（`PRAGMA integrity_check`），失敗備份需保留供事後鑑識。
5. **資料結構職責三劃分 (三事件軌分離原則)**：
   - **`execution_events`** (已存在於 001)：專責記錄單次 Task/Execution 的微觀執行過程（呼叫 tool, step progress, heartbeat）。生命週期綁定 `execution_id`。
   - **`event_history`** (新增於 007)：專責記錄 AIA 長期存在的業務與自主性宏觀事件（市場異常、目標修訂、專案狀態變更）。不可竄改，為時間連續性 (Continuity) 的第一事實來源。
   - **`audit_logs`** (已存在於 001/002，擴充於 009)：專責記錄受治理的高風險操作與審批軌跡（誰在何時執行/申請何項受控動作）。不開闢重複的 `policy_audit_logs` 表，統一匯流至此。
6. **通用知識層 vs 領域 Evidence 分層**：
   - 通用知識層（`sources`, `evidence`, `claims`, `claim_evidence_links`）作為系統跨領域的客觀事實底座。
   - 投資專業層（現有 `investment_decision_evidence`, `investment_raw_documents`）保留金融專屬欄位，透過外鍵或關聯引用共用 Evidence，兩者互補而非競爭。

---

## 2. 循序遷移路線圖 (Sequential Migration Roadmap)

遷移順序由**資料實體間的外鍵與因果依賴關係**嚴格決定，而非受限於階段劃分：

```text
 ┌─────────────────────────────────────────────────────────────────────────────────┐
 │ 既有程式碼基準 (Commit 7597902): Migration 001 ～ 006 凍結保護                    │
 │ 001 (Initial) ──► 002 (Audit Meta) ──► 003 (Inv Core) ──► 004 (Inv Pipeline)   │
 │                                    ──► 005 (Inv Predict) ──► 006 (Schedules)    │
 └────────────────────────────────────────┬────────────────────────────────────────┘
                                          │
                                          ▼
 Phase 1: 007_event_history_and_provenance.sql (基礎事實與知識溯源層)
   │      • 建立 event_history (不可竄改業務事件流)
   │      • 建立 sources, evidence, claims, claim_evidence_links (通用知識層)
   │      • 擴充 memories (多型 provenance: ref_type + ref_id)
   ▼
 Phase 1: 008_goals_decisions_and_continuity.sql (目標、脈絡與決策骨幹層)
   │      • 建立 goals, goal_revisions, goal_project_links (M:N)
   │      • 擴充 projects (primary_goal_id, status, pending_issues)
   │      • 建立 context_snapshots (跨 Session 重建基礎)
   │      • 建立 decisions (支援 NO_ACTION) 與 decision_evidence_links
   ▼
 Phase 2: 009_governance_and_approvals.sql (運作治理與審批層)
   │      • 建立 approvals (關聯 tasks, decisions)
   │      • 擴充 audit_logs (新增 policy_rule, risk_tier, decision_id 治理欄位，不重複建表)
   ▼
 Phase 3: 010_initiatives_and_deliberation.sql (主動性候選與推演層)
   │      • 建立 candidate_initiatives (主動探索發現之候選工作)
   │      • 建立 deliberation_records (深思過程、替代方案評估)
   ▼
 Phase 4: 011_learning_and_correction.sql (自我修正、RCA 與經驗層)
   │      • 建立 outcome_evaluations (預期 vs 實際結果落差)
   │      • 建立 root_cause_analyses (根因分析歸類)
   │      • 建立 correction_actions (C1～C5 修正動作紀錄)
   ▼
 Phase 5: 012_experiments_and_collaboration.sql (長週期實驗與跨領域協作層)
          • 建立 experiments (科研假設、基準線、對照組)
          • 建立 delegation_requests (跨 Agent 委派契約與驗收狀態)
```

---

## 3. 各階段遷移檔案詳細規範 (Detailed Migration Files)

### Phase 1 交付：資料基礎與連續性 (Migration 007 & 008)

#### **`007_event_history_and_provenance.sql`**
*依賴：無新增外鍵依賴，直接擴充現有庫。*
- 建立 `event_history`：全域不可竄改事件流（`aggregate_type`, `aggregate_id`, `actor_id`, `payload_json`, `occurred_at`），提供跨 Session 脈絡重建之第一事實來源。
- 建立通用知識溯源表：
  - `sources`：資料來源（URI、來源類型、先驗信賴分數）。
  - `evidence`：客觀證據（`source_id`, `content_snippet`, `content_hash`, `valid_until`）。
  - `claims`：可檢驗命題陳述（`statement`, `assertion_status`, `is_knowledge`）。
  - `claim_evidence_links`：支持/反駁多對多關聯（`claim_id`, `evidence_id`, `support_type`）。
- 擴充 `memories`：
  - 新增 `provenance_type`（`FACT_BASED | INFERENCE | HYPOTHESIS | USER_PREFERENCE | AGENT_DECISION`）。
  - 新增 `provenance_ref_type`（`EVIDENCE | SOURCE | MEMORY | CONTEXT_SNAPSHOT | DECISION | NONE`）。
  - 新增 `provenance_ref_id`（多型引用，不設單一強制外鍵，強調溯源完整性）。
  - 新增 `confidence`。

#### **`008_goals_decisions_and_continuity.sql`**
*依賴：依賴 `007` 的 `evidence`、`event_history` 以及現有 `projects`。*
- 建立目標實體：
  - `goals`（父子目標樹、目標層級、指標 JSON、優先級、狀態機）。
  - `goal_revisions`（目標變更歷史、修訂理由、修訂前/後狀態）。
  - `goal_project_links`（Goal ↔ Project 多對多關聯表，附帶 `link_type`）。
- 擴充 `projects`：新增 `primary_goal_id`（便利查詢外鍵）、`status`、`latest_snapshot_id`、`pending_issues_json`。
- 建立 `context_snapshots`：捕捉工作區狀態、未完成 Task ID 列表、近期決策摘要、Pending Issues。
- 建立決策骨幹實體（**在 Phase 1 即行落地，作為跨 Session 決策脈絡的核心載體**）：
  - `decisions`（記錄 `action_type: ACT | DEFER | MONITOR | ASK_HUMAN | NO_ACTION`、意圖摘要、深思理由、替代方案、關聯之 `context_snapshot_id` 與治理狀態）。
  - `decision_evidence_links`（決策與共用 `evidence` 之間的多對多關聯）。

---

### Phase 2 交付：運作治理與操作邊界 (Migration 009)

#### **`009_governance_and_approvals.sql`**
*依賴：依賴現有 `tasks`、`008` 的 `decisions`、現有 `audit_logs`。*
- 建立 `approvals`：
  - 欄位：`id`, `task_id`, `decision_id`, `operation_type`, `risk_tier`, `requested_by_agent_id`, `status` (`PENDING`, `APPROVED`, `REJECTED`, `EXPIRED`, `CANCELLED`), `human_comment`, `requested_at`, `responded_at`, `expires_at`。
- 擴充既有 `audit_logs`（**不新建 `policy_audit_logs`，避免稽核歷史分散**）：
  - `ALTER TABLE audit_logs ADD COLUMN policy_id TEXT DEFAULT '';`
  - `ALTER TABLE audit_logs ADD COLUMN decision_id TEXT DEFAULT '';`
  - `ALTER TABLE audit_logs ADD COLUMN approval_id TEXT DEFAULT '';`
  - `ALTER TABLE audit_logs ADD COLUMN risk_tier TEXT DEFAULT 'LOW';`
  - `ALTER TABLE audit_logs ADD COLUMN governance_action TEXT DEFAULT '';`
  - 新增索引提升治理審計查詢效能。

---

### Phase 3 交付：自主核心與主動性 (Migration 010)

#### **`010_initiatives_and_deliberation.sql`**
*依賴：依賴 `008` 的 `goals`、`projects`、`decisions`。*
- 建立 `candidate_initiatives`：
  - 記錄 Agent 主動觀察發現的候選問題/機會（`trigger_source`, `detected_opportunity`, `priority_score`, `status: CANDIDATE | ADOPTED | DISCARDED`）。
- 建立 `deliberation_records`：
  - 記錄深思推演細節（成本效益比估算、風險權衡矩陣、捨棄方案理由），可一對一或一對多鏈結至已存在的 `decisions` 實體。

---

### Phase 4 交付：自我修正與持續學習 (Migration 011)

#### **`011_learning_and_correction.sql`**
*依賴：依賴 `008` 的 `decisions`、現有 `tasks`。*
- 建立 `outcome_evaluations`：預期成果 vs 實際成果落差量測（`expected_json`, `actual_json`, `discrepancy_score`, `evaluation_status`）。
- 建立 `root_cause_analyses`：根因歸類與分析結論（`discrepancy_id`, `root_cause_category`, `analysis_summary`）。
- 建立 `correction_actions`：C1～C5 自我修正動作記錄（`rca_id`, `correction_type: C1_FACT | C2_JUDGMENT | C3_STRATEGY | C4_PROMPT | C5_WORKFLOW`, `target_entity_ref`, `before_state`, `after_state`, `verification_status`）。

---

### Phase 5 交付：長週期實驗與跨領域協作 (Migration 012)

#### **`012_experiments_and_collaboration.sql`**
*依賴：依賴 `008` 的 `goals`、`projects`、現有 `tasks`。*
- 建立 `experiments`：科研假設驗證單元（`project_id`, `goal_id`, `hypothesis`, `experiment_type`, `status`, `baseline_metrics_json`, `current_metrics_json`）。
- 建立 `delegation_requests`：跨 Agent 委派契約（`requester_agent_id`, `target_agent_id`, `source_project_id`, `goal_id`, `task_spec`, `acceptance_gate`, `status`）。

