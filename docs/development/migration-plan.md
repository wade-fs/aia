# AIA 資料庫遷移演進計畫書 (Database Migration Plan)

**文件代號：** AIA-DEV-MIG-01  
**適用資料庫：** SQLite (WAL 模式)  
**文件狀態：** Active Baseline (Migrations `001`～`019` Implemented)  
**依賴基準：** [docs/architecture/data-model.md](file:///home/wade/src/github/aia/docs/architecture/data-model.md)  
**現有代碼基準：** Git baseline `main` (已落地全量遷移 `001`～`019`)  
**最後更新：** 2026-11-04  

---

## 1. 遷移基本原則 (Migration Principles)

依據 SQLite 特性與 [Volume 5 (Runtime Engine)](file:///home/wade/src/github/aia/docs/specifications/volume-5-runtime-and-interaction.md) 規範，AIA 資料庫遷移恪守以下原則：

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
   │      • 建立 learning_records (7 大科學對照欄位、L0~L3 成熟度、負向約束)
   ▼
 Phase 5: 012_experiments_and_collaboration.sql (長週期實驗與跨領域協作層)
   │      • 建立 experiments (AIA-15 11 狀態機)、experiment_goal_links、experiment_generations (G1 凍結保護)
   │      • 建立 experiment_comparisons、delegation_requests (跨 Agent 委派契約)
   ▼
 Phase 5B: 013_project_domain_type.sql (多領域專案路由層)
   │      • 擴充 projects 表新增 domain_type (GENERAL/INVESTMENT/CODING/WRITING)
   ▼
 Phase 6: 014_self_modification_and_evolution.sql (受控代碼自修與系統演化層 - AIA-14)
   │      • 建立 self_modification_proposals (三級隔離 Scope A/B/C、Git Worktree 沙盒、審批與回滾)
   ▼
 Phase 6+: 015_writing_brief_and_clarification.sql (創作領域 Brief 與釐清層 - AIA-20)
   │      • 建立 writing_briefs 與 writing_clarifications
   ▼
 Phase 7: 016_generic_brief_and_history.sql (通用專案綱要與不可變欄位歷史層 - AIA-21)
   │      • 建立 project_briefs (跨領域需求綱要、YAML Payload、Readiness Stage)
   │      • 建立 brief_field_history (不可變欄位修訂歷史、版本、來源、任務/釐清 ID 溯源)
   ▼
 Phase 7: 017_generic_clarification_management.sql (通用人機互動釐清管理層 - AIA-21)
   │      • 建立 clarification_requests (阻斷式人機 Q&A、選項序列化、答案回填與 Brief 欄位綁定)
   ▼
 Phase 7: 018_complex_problem_and_planning.sql (複雜問題建模、分析紀錄與版本化計畫層 - AIA-22)
   │      • 建立 problems, problem_models (多版本形式化問題模型、目標、事實、假說、未知、風險)
   │      • 建立 analysis_records (多階段診斷推理紀錄、因果圖節點與證據關聯)
   │      • 建立 plans, plan_changes (版本化工作項 DAG、變更提案、影響分析與工單具體化)
   ▼
 Phase 6/7 加固: 019_self_modification_validation_and_monitoring.sql (自修多層驗證與監控擴充 - AIA-14)
          • 擴充 self_modification_proposals 新增 validation_plan_json, verification_results_json, post_deploy_metrics_json
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

### Phase 5 交付：長週期跨世代實驗與跨領域協作 (Migration 012) [IMPLEMENTED]

#### **`012_experiments_and_collaboration.sql`**
*依賴：依賴 `008` 的 `goals`、`projects`、`011` 的 `correction_actions`/`learning_records` 與現有 `tasks`、`investment_*`。*
- 建立 `experiments`：
  - 嚴格遵循 [Volume 4 (Experiments)](file:///home/wade/src/github/aia/docs/specifications/volume-4-self-correction-and-experiments.md) 完整的 11 個生命週期狀態：
    `CHECK (status IN ('PROPOSED', 'PENDING_APPROVAL', 'READY', 'RUNNING', 'PAUSED', 'EVALUATING', 'NEEDS_REVISION', 'COMPLETED', 'STOPPED', 'FAILED', 'ARCHIVED'))`
  - 欄位：`project_id`, `primary_goal_id REFERENCES goals(id)`, `domain`, `owner_agent_id`, `name`, `hypothesis`, `experiment_type`, `baseline_metrics_json`, `current_metrics_json`, `iteration_count`。
- 建立 `experiment_goal_links`：支援 Experiment ↔ Goal 多對多驗證關聯。
- 建立 `experiment_generations`（支撐 [Volume 6 §2 (Investment Pilot)](file:///home/wade/src/github/aia/docs/specifications/volume-6-domain-specializations.md) 雙世代投資決策對照實驗）：
  - 隔離綁定 `portfolio_profile_id`（`UNIQUE(experiment_id, portfolio_profile_id)`）與世代代號（`G1`, `G2`, `G2.1`, `G2.2`）。
  - 強制不變量 `CHECK (role != 'CONTROL_BASELINE' OR is_frozen = 1)`，並附加 SQLite 防竄改觸發器 `trg_exp_gen_freeze_update` 與 `trg_exp_gen_freeze_delete`，硬性阻擋對凍結 `G1` 基準組之任何修改或刪除。
  - 關聯 `parent_generation_id`、`source_correction_id` 與 `source_learning_id`，構成不可覆寫之世代演化譜系。
- 建立 `experiment_comparisons`：保存 Track A（績效與 Brier / LogLoss / DecisionConfidence 雙校準）及 Track B（自主能力與五大判讀情境 `RegimeVerdict`）之時間點對照紀錄。
- **擴充 `investment_predictions` 與 `investment_prediction_outcomes`**：新增 `expected_return_pct`、`expected_range_low_pct`、`expected_range_high_pct` 與 `log_loss`。
- **擴充現有 `tasks` 表**：
  - `ALTER TABLE tasks ADD COLUMN experiment_id TEXT REFERENCES experiments(id);`
- 建立 `delegation_requests`：跨 Agent 委派契約（`requester_agent_id`, `target_agent_id`, `source_project_id`, `goal_id`, `task_spec`, `acceptance_gate`, `status`）。

---

### Phase 5B 交付：多領域專案路由 (Migration 013) [IMPLEMENTED]

#### **`013_project_domain_type.sql`**
*依賴：擴充 `008` 的 `projects` 表。*
- **擴充 `projects` 表**：
  - `ALTER TABLE projects ADD COLUMN domain_type TEXT NOT NULL DEFAULT 'general';`
  - `CREATE INDEX IF NOT EXISTS idx_projects_domain_type ON projects(domain_type);`
  - 支援資料庫驅動之 Observer 與 Agent 路由（`writing`、`coding`、`investment`、`general`），取代啟動時硬編碼路徑註冊。

---

### Phase 6 交付：受控安全代碼自修與系統演化 (Migration 014 & 019) [IMPLEMENTED]

#### **`014_self_modification_and_evolution.sql`**
*依賴：依賴 `009` 的 `approvals`。*
- 建立 `self_modification_proposals`：
  - 欄位：`id`, `title`, `description`, `rationale`, `target_files_json`, `patches_json`, `scope_level` (`LEVEL_A_DOMAIN`, `LEVEL_B_CORE`, `LEVEL_C_GOVERNANCE`), `requires_approval`, `approval_id REFERENCES approvals(id)`, `status` (`DRAFT`, `IN_SANDBOX`, `PENDING_APPROVAL`, `APPROVED`, `VERIFIED`, `MERGED`, `ROLLED_BACK`, `REJECTED`, `REQUEST_CHANGES`, `DEFERRED`), `branch_name`, `pre_commit_sha`, `post_commit_sha`, `verification_output`, `rca_log`, `created_at`, `completed_at`。
  - 索引：`idx_selfmod_status`, `idx_selfmod_scope`。

#### **`019_self_modification_validation_and_monitoring.sql`**
*依賴：擴充 `014` 的 `self_modification_proposals` 表。*
- **擴充 `self_modification_proposals` 表**：
  - `ALTER TABLE self_modification_proposals ADD COLUMN validation_plan_json TEXT NOT NULL DEFAULT '[]';`
  - `ALTER TABLE self_modification_proposals ADD COLUMN verification_results_json TEXT NOT NULL DEFAULT '[]';`
  - `ALTER TABLE self_modification_proposals ADD COLUMN post_deploy_metrics_json TEXT NOT NULL DEFAULT '{}';`
  - 支援 AIA-14 §4.6、§4.10 與 §7.1 所要求之 V1～V6 多層級驗證管線、多輪測試執行結果，以及部署後自主監控度量指標（`error_rate`, `crash_rate`, `autonomy_metric`）。

---

### Phase 6+ 交付：寫作領域創意綱要與人機釐清 (Migration 015) [IMPLEMENTED]

#### **`015_writing_brief_and_clarification.sql`**
*依賴：依賴 `projects`。*
- 建立 `writing_briefs`：專屬創作大綱（`project_id REFERENCES projects(id)`, `title`, `raw_intent`, `genre`, `era`, `theme_and_philosophy_json`, `scale_json`, `core_protagonists_json`, `worldview_settings_json`, `writing_rules_json`, `status`, `pending_questions`）。
- 建立 `writing_clarifications`：創作領域人機互動釐清（`project_id`, `brief_id`, `category`, `question`, `rationale`, `options_json`, `user_answer`, `status`）。

---

### Phase 7 交付：通用專案綱要、動態釐清與複雜問題規劃 (Migrations 016, 017, 018) [IMPLEMENTED]

#### **`016_generic_brief_and_history.sql`**
*依賴：依賴 `projects`。*
- 建立 `project_briefs` (AIA-21 §4)：跨領域通用需求綱要（`project_id UNIQUE`, `domain`, `brief_version`, `readiness_status`, `raw_intent`, `brief_yaml`）。
- 建立 `brief_field_history` (AIA-21 §12)：不可變欄位級修訂歷史（`brief_id`, `project_id`, `field_key`, `version`, `old_value_json`, `new_value_json`, `old_status`, `new_status`, `source`, `rationale`, `related_task_id`, `related_clarification_id`）。

#### **`017_generic_clarification_management.sql`**
*依賴：依賴 `projects` 與 `tasks`。*
- 建立 `clarification_requests` (AIA-21 §5)：跨領域通用阻斷式人機互動提問管理（`project_id`, `task_id`, `field_key`, `question`, `options_json`, `reason`, `impact`, `priority`, `blocking`, `status`, `answer`, `answered_at`, `expires_at`）。

#### **`018_complex_problem_and_planning.sql`**
*依賴：依賴 `projects` 與 `goals`。*
- 建立 `problems` (AIA-22 §6)：形式化開放問題本體（`project_id`, `parent_id`, `goal_id`, `title`, `description`, `objective`, `scope`, `status`, `priority`, `current_model_version`, `current_plan_version`）。
- 建立 `problem_models` (AIA-22 §6)：多版本問題結構化模型（`problem_id`, `version`, `objectives_json`, `constraints_json`, `known_facts_json`, `assumptions_json`, `unknowns_json`, `dependencies_json`, `risks_json`, `success_criteria_json`, `UNIQUE(problem_id, version)`）。
- 建立 `analysis_records` (AIA-22 §6)：多階段診斷與因果推演紀錄（`problem_id`, `model_version`, `analysis_type`, `input_refs_json`, `findings_json`, `conclusions_json`, `open_questions_json`, `agent_id`, `execution_id`）。
- 建立 `plans` (AIA-22 §12)：版本化執行計畫（`problem_id`, `version`, `objective`, `strategy`, `status`, `work_items_json`, `dependencies_json`, `acceptance_criteria_json`, `supersedes_plan_id`, `UNIQUE(problem_id, version)`）。
- 建立 `plan_changes` (AIA-22 §12)：計畫變更提案與衝擊分析（`plan_id`, `from_version`, `to_version`, `reason`, `evidence_refs_json`, `affected_work_items_json`, `cost_risk_impact`, `requires_human_approval`, `approved_by`）。
