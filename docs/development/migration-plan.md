# AIA 資料庫遷移演進計畫書 (Database Migration Plan)

**文件代號：** AIA-DEV-MIG-01  
**適用資料庫：** SQLite (WAL 模式)  
**文件狀態：** Active Baseline (Migrations `001`～`011` Implemented; `012` Planned for Phase 5)  
**依賴基準：** [docs/architecture/data-model.md](file:///home/wade/src/github/aia/docs/architecture/data-model.md)  
**現有代碼基準：** Git commit `7597902` 凍結資產 + Phase 1～4 已落地遷移 (`007`～`011`)  
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
   - DDL（`CREATE TABLE`、`ALTER TABLE`、`CREATE TRIGGER`）及初始化 DML 必須包裝在單一 `BEGIN IMMEDIATE TRANSACTION` 內，任一失敗即全量回滾。
   - `PRAGMA foreign_keys`、`PRAGMA journal_mode` 等為**連線層級設定**，不可包含於同一 DDL 交易；WAL 模式為**資料庫層級設定**，在啟動時於交易外設定。
4. **預遷移快照備份 (SQLite Backup API)**：
   - 在執行任何新 Migration 前，Go 驅動必須使用 **SQLite Backup API**（`sqlite3_backup_*`）建立帶有時間戳的完整備份副本。直接複製 `.db` 檔案在 WAL 模式下可能產生不一致；Backup API 能確保備份時資料庫完整性。備份完成後應執行完整性檢查（`PRAGMA integrity_check`），失敗備份需保留供事後鑑識。
5. **資料結構職責三劃分 (三事件軌分離原則)**：
   - **`execution_events`** (已存在於 001)：專責記錄單次 Task/Execution 的微觀執行過程（呼叫 tool, step progress, heartbeat）。生命週期綁定 `execution_id`。
   - **`event_history`** (新增於 007)：專責記錄 AIA 長期存在的業務與自主性宏觀事件（市場異常、目標修訂、專案狀態變更）。具備 Hash Chain 與 Trigger 防竄改，為時間連續性 (Continuity) 的第一事實來源。
   - **`audit_logs`** (已存在於 001/002，擴充於 009)：專責記錄受治理的高風險操作與審批軌跡（誰在何時執行/申請何項受控動作）。不開闢重複的 `policy_audit_logs` 表，統一匯流至此。
6. **通用知識層 vs 領域 Evidence 分層**：
   - 通用知識層（`sources`, `evidence`, `claims`, `claim_revisions`, `claim_evidence_links`）作為系統跨領域的客觀事實底座。
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
   │      • 建立 event_history (完整 Hash Chain、occurred_at vs recorded_at、防竄改 Triggers)
   │      • 建立 sources (五維度評估), evidence (發布/擷取時間戳), claims, claim_revisions
   │      • 擴充 memories (多型 provenance: ref_type + ref_id + provenance_status)
   ▼
 Phase 1: 008_goals_decisions_and_continuity.sql (目標、脈絡與決策骨幹層)
   │      • 建立 autonomy_cycles (自主循環核心樞紐)
   │      • 建立 goals (含 DRAFT), goal_revisions (UNIQUE 遞增), goal_project_links (M:N)
   │      • 擴充 projects (primary_goal_id, latest_snapshot_id 外鍵, status, pending_issues)
   │      • 建立 context_snapshots (GLOBAL/PROJECT 兩層級，工作區敏感資訊安全過濾)
   │      • 建立 decisions (支援 NO_ACTION，關聯 cycle/task/execution) 與 decision_evidence_links
   ▼
 Phase 2: 009_governance_and_approvals.sql (運作治理與審批層)
   │      • 建立 approvals (關聯 tasks 或 decisions 至少一項)
   │      • 擴充 audit_logs (新增 policy_rule, risk_tier, decision_id, approval_id，不重複建表)
   ▼
 Phase 3: 010_initiatives_and_deliberation.sql (主動性候選與推演層)
   │      • 建立 candidate_initiatives (主動探索發現之候選工作)
   │      • 建立 deliberation_records (關聯 cycle_id 與 decision_id，推演替代方案與矩陣)
   ▼
 Phase 4: 011_learning_and_correction.sql (自我修正、RCA 與經驗層)
   │      • 建立 outcome_evaluations (預期 vs 實際結果落差量測)
   │      • 建立 root_cause_analyses (外鍵關聯 outcome_evaluations)
   │      • 建立 correction_actions (徹底分離 correction_type 與 C0～C5 authorization_level)
   ▼
 Phase 5: 012_experiments_and_collaboration.sql (長週期實驗與跨領域協作層)
          • 建立 experiments (嚴格遵循 AIA-15 完整 11 個生命週期狀態)、experiment_goal_links
          • 擴充 tasks 表新增 experiment_id 外鍵欄位
          • 建立 delegation_requests (跨 Agent 委派契約與驗收狀態)
```

---

## 3. 各階段遷移檔案詳細規範 (Detailed Migration Files)

### Phase 1 交付：資料基礎與連續性 (Migration 007 & 008)

#### **`007_event_history_and_provenance.sql`**
*依賴：無新增外鍵依賴，直接擴充現有庫。*
- **`event_history` (不可竄改業務事實與自主性事件流)**：
  - 欄位：`id`, `event_id` (UUID), `event_type`, `event_version`, `aggregate_type`, `aggregate_id`, `aggregate_version`, `correlation_id`, `causation_id`, `actor_id`, `payload_json`, `payload_hash`, `previous_hash`, `occurred_at`, `recorded_at`。
  - 建立唯一約束：`UNIQUE(aggregate_type, aggregate_id, aggregate_version)`。
  - 建立應用層防竄改 Triggers：`trg_event_history_no_update` 與 `trg_event_history_no_delete`。
- **通用知識溯源表**：
  - `sources`：五維度來源信賴評估（`authority_score`, `directness_score`, `timeliness_score`, `independence_score`, `reproducibility_score`）。
  - `evidence`：客觀證據（`source_id`, `content_snippet`, `content_hash`, `capture_method`, `data_version`, `published_at`, `captured_at`, `valid_until`）。
  - `claims`：可檢驗命題陳述（`statement`, `assertion_status`, `is_knowledge`, `current_version`）。
  - `claim_revisions`：主張版本修訂歷史（`claim_id`, `revision_number`, `previous_statement`, `new_statement`, `reason`, `revised_decision_id`）。
  - `claim_evidence_links`：支持/反駁多對多關聯（`claim_id`, `evidence_id`, `support_type`, `confidence_weight`）。
- **擴充 `memories` 表**：
  - 新增 `provenance_type`（`FACT_BASED | INFERENCE | HYPOTHESIS | USER_PREFERENCE | AGENT_DECISION`）。
  - 新增 `provenance_ref_type`（`EVIDENCE | SOURCE | MEMORY | CONTEXT_SNAPSHOT | DECISION | NONE`）。
  - 新增 `provenance_ref_id`（多型引用，不設單一強制外鍵）。
  - 新增 `provenance_status`（`VERIFIED | PARTIAL | UNVERIFIED`），消除預設值與來源錨點的邏輯矛盾。

#### **`008_goals_decisions_and_continuity.sql`**
*依賴：依賴 `007` 的 `evidence`、`event_history` 以及現有 `projects`。*
- **`autonomy_cycles` (自主循環核心樞紐)**：
  - 欄位：`id`, `cycle_number`, `trigger_type`, `trigger_ref_id`, `status`, `context_snapshot_id`, `summary`, `started_at`, `completed_at`, `created_at`。
- **目標實體**：
  - `goals`（status CHECK 含 `DRAFT`, `ACTIVE`, `PAUSED`, `ACHIEVED`, `SUPERSEDED`, `ABANDONED`）。
  - `goal_revisions`（`UNIQUE(goal_id, revision_number)` 嚴格遞增）。
  - `goal_project_links`（Goal ↔ Project 多對多關聯表，附帶 `link_type`）。
- **擴充 `projects`**：新增 `primary_goal_id REFERENCES goals(id)`、`latest_snapshot_id REFERENCES context_snapshots(id)`、`status`、`pending_issues_json`。
- **`context_snapshots`**：
  - 欄位：`id`, `scope` (`GLOBAL | PROJECT`), `project_id`, `snapshot_type`, `workspace_state_json`（安全過濾，禁止密鑰）, `active_tasks_json`, `recent_decisions_json`, `pending_issues_json`。
- **決策骨幹實體 (在 Phase 1 即行落地)**：
  - `decisions`（關聯 `cycle_id REFERENCES autonomy_cycles(id)`、`project_id`、`experiment_id`、`action_task_id REFERENCES tasks(id)`、`action_execution_id REFERENCES executions(id)`，支援 `NO_ACTION`）。
  - `decision_evidence_links`（決策與共用 `evidence` 之間的多對多關聯）。

---

### Phase 2 交付：運作治理與操作邊界 (Migration 009)

#### **`009_governance_and_approvals.sql`**
*依賴：依賴現有 `tasks`、`008` 的 `decisions`、現有 `audit_logs`。*
- 建立 `approvals`：
  - 欄位：`id`, `task_id`, `decision_id`, `operation_type`, `risk_tier`, `requested_by_agent_id`, `status`, `human_comment`, `requested_at`, `responded_at`, `expires_at`。
  - 加入一致性約束：`CHECK (task_id IS NOT NULL OR decision_id IS NOT NULL)`。
- 擴充既有 `audit_logs`（**不新建 `policy_audit_logs`，避免稽核歷史分散**）：
  - `ALTER TABLE audit_logs ADD COLUMN policy_id TEXT DEFAULT '';`
  - `ALTER TABLE audit_logs ADD COLUMN decision_id TEXT DEFAULT '';`
  - `ALTER TABLE audit_logs ADD COLUMN approval_id TEXT DEFAULT '';`
  - `ALTER TABLE audit_logs ADD COLUMN risk_tier TEXT DEFAULT 'LOW';`
  - `ALTER TABLE audit_logs ADD COLUMN governance_action TEXT DEFAULT '';`
  - 建立 `decision_id` 與 `approval_id` 之索引。

---

### Phase 3 交付：自主核心與主動性 (Migration 010)

#### **`010_initiatives_and_deliberation.sql`**
*依賴：依賴 `008` 的 `autonomy_cycles`、`goals`、`projects`、`decisions`。*
- 建立 `candidate_initiatives`：
  - 記錄 Agent 主動觀察發現的候選問題/機會（`cycle_id`, `trigger_source`, `detected_opportunity`, `priority_score`, `status: CANDIDATE | ADOPTED | DISCARDED`）。
- 建立 `deliberation_records`：
  - 關聯 `cycle_id REFERENCES autonomy_cycles(id)` 與 `decision_id REFERENCES decisions(id)`。
  - 記錄深思推演細節（選項評估矩陣、成本效益比推演、風險評估與捨棄方案理由）。

---

### Phase 4 交付：可驗證自我修正、學習機制與目標演化 (Migration 011) [IMPLEMENTED]

#### **`011_learning_and_correction.sql`**
*依賴：依賴 `008` 的 `autonomy_cycles`、`decisions`、`goals`、`009` 的 `approvals` 與現有 `tasks`、`executions`。*
- **擴充 `executions` 表**：新增 `input_tokens`、`output_tokens`、`total_tokens` 欄位（支撐每日 Token 預算硬閘門聚合）。
- 建立 `outcome_evaluations`：預期成果 vs 實際成果落差量測（`cycle_id`, `decision_id`, `task_id`, `project_id`, `domain`, `expected_outcome_json`, `actual_outcome_json`, `discrepancy_score`, `is_anomaly`, `evaluation_status`）。
- 建立 `root_cause_analyses`：具證據錨點與可證偽假設之根因分析（外鍵關聯 `outcome_evaluation_id REFERENCES outcome_evaluations(id)`，含 `falsifiable_hypothesis`, `causal_evidence_refs_json`, `competing_factors_json`, `rca_validity_status`, `confidence`）。
- 建立 `correction_actions`：徹底分離 `scope_level` (`A/B/C`)、`correction_type` (`FACT/JUDGMENT/STRATEGY/PROMPT/WORKFLOW/CODE`)、`authorization_level` (`C0~C5`)、`validation_level` (`V1~V6`) 與 `tier_profile` (`TIER1_AUTO/TIER2_APPROVAL/TIER3_BLOCKED`)，含 `regression_status` 與四態結論（支援 `INCONCLUSIVE` 與 `REGRESSED`）。
- 建立 `learning_records`：強制保存 7 大科學對照欄位（`baseline_json`, `hypothesis_json`, `intervention_json`, `verification_json`, `regression_json`, `followup_json`, `epistemic_confidence`）、對照方法（`CONTROL_BENCHMARK / PIT_REPLAY / SIMILAR_COHORT / PRE_POST_ONLY`）與 `L0_RECORDED ~ L3_GENERALIZED` 成熟度分級。
- 建立 `goal_evolution_proposals`：支援 `PERSISTENCE | OPTIMIZATION | REVISION` 三態分流與「防移動球門 (`invalidated_claim_ids_json`, `REJECTED_GOALPOST_MOVING`)」審批追蹤。

---

### Phase 5 交付：長週期實驗與跨領域協作 (Migration 012)

#### **`012_experiments_and_collaboration.sql`**
*依賴：依賴 `008` 的 `goals`、`projects`、現有 `tasks`。*
- 建立 `experiments`：
  - 嚴格遵循 [AIA-15](file:///home/wade/src/github/aia/docs/specifications/AIA-15-Autonomous-Experiment-Lifecycle-and-Domain-Integration-Specification.md) 完整的 11 個生命週期狀態：
    `CHECK (status IN ('PROPOSED', 'PENDING_APPROVAL', 'READY', 'RUNNING', 'PAUSED', 'EVALUATING', 'NEEDS_REVISION', 'COMPLETED', 'STOPPED', 'FAILED', 'ARCHIVED'))`
  - 欄位：`project_id`, `primary_goal_id REFERENCES goals(id)`, `name`, `hypothesis`, `experiment_type`, `baseline_metrics_json`, `current_metrics_json`, `iteration_count`。
- 建立 `experiment_goal_links`：支援 Experiment ↔ Goal 多對多驗證關聯。
- **擴充現有 `tasks` 表**：
  - `ALTER TABLE tasks ADD COLUMN experiment_id TEXT REFERENCES experiments(id);`
- 建立 `delegation_requests`：跨 Agent 委派契約（`requester_agent_id`, `target_agent_id`, `source_project_id`, `goal_id`, `task_spec`, `acceptance_gate`, `status`）。
