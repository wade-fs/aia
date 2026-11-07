# AIA Development: Implementation Status

**Project:** Autonomous Intelligence Architecture (AIA)  
**Active Implementation Status:** **Phase 0 ～ Phase 6 Completed + Phase 7 Hardened Baseline (AIA-21, AIA-22, AIA-19, AIA-13, AIA-14, AIA-06 Implemented & Integrated)** (Phase 6 Safe Self-Modification, Multi-Tier Validation V1~V6 & Post-Deployment Monitoring; Universal Project Brief & Field History; Generic Clarification Management; Complex Problem Analysis, Autonomous Escalation & Versioned Planning Engine; Backup & Restore; Distributed Tracing; Operational Metrics; 16-Tab Web Console & Emergency Stop; Migrations `001` ～ `019` active)  
**Historical Audit Baseline:** Commit [`7597902`](file:///home/wade/src/github/aia) ➔ Phase 3 Baseline [`b66b1c6`](file:///home/wade/src/github/aia) ➔ Phase 4 Baseline [`0e189d9`](file:///home/wade/src/github/aia) ➔ Phase 5A Baseline [`29d1cb3`](file:///home/wade/src/github/aia) ➔ Phase 5B Baseline ➔ Phase 6 Active Baseline ➔ AIA-21/22/19/13 Active  
**Specification Baseline:** [docs/specifications/](file:///home/wade/src/github/aia/docs/specifications) (AIA-00 ～ AIA-22) v2.0  
**Audit Report:** [docs/specifications/cross-spec-consistency-audit.md](file:///home/wade/src/github/aia/docs/specifications/cross-spec-consistency-audit.md)  
**Last Updated:** 2026-11-05  

---

## 1. Current Architecture Overview (Phase 7 Active Implementation Baseline)

The physical Go codebase implements Phases 0 through 6 and the Phase 7 integration baseline (Runtime, Continuity & Provenance, Governance, 11-Step Shared Autonomy Core across 3 Domains, Phase 3.5 Hardening, Phase 4 Verifiable Self-Correction & Goal Evolution, Phase 5A Generational Experimentation, Phase 5B Multi-Domain MVA & Cross-Domain Delegation, Phase 6 Governed Safe Self-Modification & System Evolution, AIA-21 Universal Brief & Clarification, AIA-22 Complex Problem Analysis & Planning, AIA-19 Operations/Backup/Tracing, AIA-13 Autonomy Evaluation Metrics, and AIA-06 16-Tab Web Console):

```text
go/
├── cmd/aia/                  # Cobra CLI Entrypoints
│   └── main.go               # Delegates to internal/cli root & subcommands
├── internal/
│   ├── app/                  # Application bootstrap & dependency wiring
│   ├── audit/                # Audit context helper
│   ├── autonomy/             # 11-step Autonomy Core & Evolution Engine (Phase 3 ~ Phase 7)
│   │   ├── clock/            # Point-in-Time (AsOf) Clock & zero-look-ahead replay guard
│   │   ├── cycle/            # 11-step Coordinator, budget pre-flight, learning & problem escalation hooks
│   │   ├── deliberation/     # EV trade-off matrix, L2/L3 bonus & negative constraint suppression
│   │   ├── evaluation/       # AIA-13 general autonomy evaluation metrics module
│   │   ├── initiative/       # Multi-source CandidateInitiative generator & deduplicator
│   │   ├── learning/         # Outcome eval, RCA, 3-Tier Correction, L0~L3 maturity & rollback
│   │   ├── observation/      # Multi-domain observation collector (incl. brief, coding, writing observers)
│   │   ├── recall/           # L1~L5 working memory + Point-in-Time L2/L3 & negative learnings
│   │   └── self_evolution_engine.go # Autonomous system evolution coordinator (Phase 6)
│   ├── cli/                  # CLI commands (agent, approval, backup, clarification, coding, cycle,
│   │                         #               decision, delegation, doctor, execution, experiment, goal,
│   │                         #               governance, investment, learning, memory, problem, project,
│   │                         #               schedule, status, task, trace, web, worker, writing)
│   ├── collector/            # TWSE, TPEx, SEC, FRED, RSS collectors
│   ├── config/               # YAML configuration loader (incl. Budget, Domains, Logging & Sandbox configs)
│   ├── continuity/           # ContextReconstructor & sanitized snapshot manager
│   ├── domain/               # Domain entities (agent, artifact, autonomy, brief, clarification, coding,
│   │                         #                  consistency, continuity, decision, execution, experiment,
│   │                         #                  goal, governance, investment, knowledge, learning, memory,
│   │                         #                  plan, policy, problem, profile, project, schedule, selfmod, task, writing)
│   ├── eventbus/             # In-memory pub/sub event bus
│   ├── experiment/           # Generational Experiment Controller (G1 frozen rules vs G2/G2.x LLM, full 11 states)
│   ├── orchestrator/         # Scheduler, ContextAssembler & PersonalAgent Strangler-Fig coordinator
│   ├── policy/               # Path/agent policies + hard daily token & cycle budget gates
│   ├── provider/             # Antigravity CLI adapter (with env secret scrubbing & bwrap/systemd sandbox) & mock provider
│   ├── recovery/             # Process restart recovery manager
│   ├── service/              # Application services (agent, approval, backup, brief, clarification, delegation,
│   │                         #                       execution, goal, governance, investment, knowledge, memory,
│   │                         #                       problem, project, schedule, selfmod pipeline, task, writing)
│   ├── storage/sqlite/       # SQLite WAL driver, migrations 001~019, repositories
│   ├── telemetry/            # Distributed Tracing & Operational Metrics (AIA-19)
│   ├── validation/           # Hardened execution validator (output/artifact/fatal-marker checks)
│   ├── web/                  # Port 7777 host, OTT/session auth, Console REST API (/api/v1/*)
│   ├── worker/               # Background ScheduleWorker (15s), TaskWorker (5s, 3 slots), LogRetentionPurger, AutoWakeWorker
│   └── workspace/            # Workspace directory manager & Git worktree isolation
├── prompts/                  # Domain & coordinator prompt templates
├── scripts/
│   └── service.sh            # systemd --user service management utility
├── web/assets/dist/          # Web Console frontend (HTML, CSS, JS with 16 tabs & Emergency Stop button)
├── Makefile                  # Root and go/ build automation
└── tests/
    ├── integration/          # End-to-end integration suites (incl. real agy CLI, Phase 4, Phase 6, problem_stage1~5)
    └── unit/                 # Unit suites (incl. Migrations 001~019, Phase 6 selfmod, brief, clarification, problem, plan, tracing)
```

---

## 2. Module Implementation & Gap Matrix

| Module / Package | Status | Target v2.0 Capability (AIA-00~19) | Implementation Details & Status |
|:---|:---:|:---|:---|
| **`internal/continuity`** | **Production Ready (Phase 1)** | Context Reconstruction Engine | Reconstructs project state, goals, pending issues, snapshot, and decisions after restart. |
| **`internal/service/knowledge`** | **Production Ready (Phase 1)** | Provenance & Knowledge Integrity | 5D source evaluation, evidence management, claim revisions, AIA-16 Rule 5 promotion guard, decision backtrack. |
| **`internal/domain/goal`** | **Production Ready (Phase 4)** | Goal Hierarchy & Evolution | Goal domain model, typed revisions (`PERSISTENCE`, `OPTIMIZATION`, `REVISION`), status machine (`DRAFT` to `ABANDONED`), M:N project links. |
| **`internal/domain/continuity`** | **Production Ready (Phase 1)** | Event Store & Snapshots | Append-only hash chain event history, sanitized context snapshots (`GLOBAL` / `PROJECT`). |
| **`internal/domain/decision`** | **Production Ready (Phase 1)** | Decision & Evidence Linking | Decision model supporting `NO_ACTION`, rationale justification, and evidence links. |
| **`internal/domain/knowledge`** | **Production Ready (Phase 1)** | Provenance, Sources & Claims | Source 5D scoring, Evidence capture methods, Claim revision tracking. |
| **`internal/domain/governance`** | **Production Ready (Phase 2)** | Governance Models & Controls | Approvals model (`TASK`, `DECISION`, `CORRECTION`, `GOAL`), risk tiers, runtime controls, strict human actor checks. |
| **`internal/service/approval`** | **Production Ready (Phase 2)** | Operational Approval Gates | Scope-bound approvals, anti-self-approval enforcement, timeout policies. |
| **`internal/service/governance`** | **Production Ready (Phase 2)** | Runtime Governance & Emergency Stop | Controlled Emergency Stop, dispatch freeze, human-only resume verification. |
| **`internal/storage/sqlite`** | **Production Ready (Phase 0~7)** | Multi-schema persistence | Migrations `001`～`019` active with append-only hash chain, budget token columns, PiT `<= asOf` filtering, learning invariants, frozen `G1` triggers, selfmod proposals & monitoring, generic briefs & field history, clarification requests, and complex problems & versioned plans. |
| **`internal/cli`** | **Production Ready (Phase 0~7)** | CLI Administration & Tracing | Comprehensive Cobra CLI covering `agent`, `approval`, `backup` (create/list/verify/restore), `clarification`, `coding`, `cycle`, `decision`, `delegation`, `doctor`, `execution`, `experiment`, `goal`, `governance`, `investment`, `learning`, `memory`, `problem` (list/analyze/validate), `project` (brief), `schedule`, `status`, `task`, `trace` (list/get), `web`, `worker` (status/run-once/stop/resume/purge-logs/trigger-cycle), and `writing`. |
| **`internal/worker`** | **Production Ready (Phase 0~7)** | Runtime Concurrency Manager | Controlled Emergency Stop (<1s dispatch freeze, context cancellation, `CANCELLED` task persistence), resident `ScheduleWorker` (15s), `TaskWorker` (5s, 3 slots), `LogRetentionPurger` (periodic expired log cleaner), and `AutoWakeWorker` (periodic autonomous cycle trigger). |
| **`internal/web/console`** | **Production Ready (AIA-06)** | Console REST API & 16-Tab Dashboard | One-click Controlled Emergency Stop & human resume verification in navigation bar, 16 full-featured tabs (`tab-overview`, `tab-tasks`, `tab-pending`, `tab-goals`, `tab-cycles`, `tab-learnings`, `tab-experiments`, `tab-delegations`, `tab-clarifications`, `tab-problems`, `tab-investment`, `tab-coding`, `tab-writing`, `tab-memory`, `tab-agents`, `tab-audit`), and complete RESTful backend. |
| **`internal/service/investment`** | **Production Ready (Phase 5A)** | Multi-Portfolio & Dual Calibration | Isolated NT$100,000 cohort profiles, virtual settlement, Brier + LogLoss event calibration separated from Decision Confidence, and PiT replay guard. |
| **`internal/experiment`** | **Production Ready (Phase 5A)** | Generational Experiment Controller | Frozen `G1` rule baseline vs `G2 -> G2.1 -> G2.2` LLM cohorts, 5-regime classification matrix, full 11-state lifecycle engine, and live `G1` Difference-in-Differences control verification. |
| **`internal/service/goal`** | **Production Ready (Phase 4)** | Goal Lifecycle & Governed Evolution | Ancestor cycle guard, `EvaluateGoalGap`, `goal_task_links`, 3-way `EvolveGoal`, and Anti-Goalpost-Moving guard. |
| **`internal/recovery`** | **Production Ready (Phase 1)** | Complete Context Reconstruction | Task recovery integrated with project & goal context continuity. |
| **`internal/policy`** | **Production Ready (Phase 3.5)** | Operational & Budget Policy | Whitelist boundaries, financial transaction lock, and pre-flight `EvaluateResourceBudget` (daily tokens & cycles). |
| **`internal/provider/antigravity`** | **Production Ready (Phase 3.5)** | Subprocess Harness & Sandbox | Whitelist env secret scrubbing, `bwrap`/`systemd-run` read-only mount isolation, fail-closed `RequireOSNamespace`, stream-json token telemetry. |
| **`internal/validation`** | **Production Ready (Phase 3.5)** | Execution Output Validator | Rejects empty/auth-error/quota-error outputs even on exit code `0`; enforces `file_non_empty:` & `contains:`. |
| **`internal/autonomy/clock`** | **Production Ready (Phase 3.5)** | Point-in-Time (`AsOf`) Clock | Context-injected `WithAsOf` / `WithReplayMode` ensuring zero look-ahead bias in historical replay. |
| **`internal/autonomy`** | **Production Ready (Phase 5B~7)** | 11-Step Shared Autonomy Core | Modular `observation` (Domain & Brief Observers), `recall`, `initiative`, `deliberation`, `learning`, and `cycle` coordinator unified across Investment, Coding, and Writing domain agents; dynamic problem escalation and plan execution. |
| **`internal/autonomy/learning`** | **Production Ready (Phase 4)** | Verifiable Self-Correction Engine | Outcome discrepancy eval, testable RCA, 3-tier profile (`TIER1_AUTO`, `TIER2_APPROVAL`, `TIER3_BLOCKED`), `L0`～`L3` maturity, 7 epistemic comparison fields, regression rollback. |
| **`internal/autonomy/evaluation`** | **Production Ready (AIA-13)** | General Autonomy Evaluation Metrics | Computable autonomy scoring module (`evaluator.go`, `metrics.go`): Proactive Rate (M3), Override Frequency / Human Intervention Rate (M4), Decision Calibration & EV (M5), Continuity Recovery Accuracy (M1), and Self-Correction Effectiveness Score (M5). |
| **`internal/service/delegation`** | **Production Ready (Phase 5B)** | Cross-Domain Agent Delegation (AIA-12 §21) | Lifecycle contract state machine (`PENDING` -> `IN_PROGRESS` -> `DELIVERED` -> `VERIFIED`), automatic task spawning, and acceptance verification. |
| **`internal/domain/coding`** | **Production Ready (Phase 5B)** | Coding Agent MVA (AIA-15 §24) | Code defect detection (`CodeHealthObserver`), sandboxed patch application & test re-run (`CodingPilotExecutor`), and deterministic test verification reaching `L2_VERIFIED`. |
| **`internal/domain/writing`** | **Production Ready (Phase 5B)** | Writing Agent MVA (AIA-15 §23) | 3-tier knowledge (`FACT_HISTORY`, `PLAUSIBLE_INFERENCE`, `FICTION`), manuscript consistency observer referencing `~/src/github/novel/yuan/`, intentional fiction license (`NO_ACTION`) vs chronological contradiction (`L2_VERIFIED`). |
| **`internal/service/writing` & `domain/writing`** | **Production Ready (Phase 6+)** | Writing Dual-Loop Workflow & Brief Alignment | `CreativeBrief`, deterministic `BriefValidator`, `GapAnalyzer`, `ClarificationRequest` (`ASK_HUMAN` Q&A loop), automatic `BookManifest` / `outline.md` / `master_bible.md` / `characters.md` generation, and `ConsistencyEngine` integration. |
| **`internal/domain/knowledge` & `consistency`** | **Production Ready (Phase 6)** | Universal Knowledge & Consistency Core | Universal `Entity`, `Relation`, `TemporalReference`, `SpatialReference`, `EpistemicStatus`, `Assertion`, and cross-domain `ConsistencyEngine`. |
| **`internal/domain/selfmod` & `service/selfmod`** | **Production Ready (Phase 6)** | Safe Self-Modification Pipeline (AIA-14) | Tri-level `ScopeClassifier` (Level A domain, Level B core with human approval gate, Level C fail-closed immutable governance), sandbox worktree execution, multi-tier verification plans (V1 compile, V2 unit, V3 integration, V4 benchmark, V5 lint, V6 security), diagnostic RCA failure retry loop, post-deployment metrics monitoring (`error_rate`, `crash_rate`, `autonomy_metric`), zero-residual rollback, and lifecycle state machine (`DRAFT` ➔ `IN_SANDBOX` ➔ `PENDING_APPROVAL` ➔ `APPROVED` ➔ `VERIFIED` ➔ `MERGED` / `ROLLED_BACK` / `REJECTED` / `REQUEST_CHANGES` / `DEFERRED`) (Migrations `014` & `019`). |
| **`internal/autonomy (SelfEvolutionEngine)`** | **Production Ready (Phase 6)** | Autonomous Evolution Coordinator (AIA-14) | Periodic autonomous inspection, hypothesis generation, modification proposal creation, sandbox verification, and end-to-end evolution cycles. |
| **`internal/domain/brief` & `service/brief`** | **Production Ready (AIA-21)** | Universal Project Brief & Field History | Domain-agnostic project brief requirements schema, readiness stage evaluation, Brief Observer integration, and immutable field-level revision history tracking (`brief_field_history`, Migration `016`). |
| **`internal/domain/clarification` & `service/clarification`** | **Production Ready (AIA-21)** | Generic Clarification Management | Interactive human-in-the-loop Q&A (`clarification_requests`), priority/blocking evaluation, option selection, and brief field binding (Migration `017`). |
| **`internal/domain/problem` & `service/problem`** | **Production Ready (AIA-22)** | Complex Problem Analysis & Diagnostic Engine | Formal problem definition, multi-version problem models (`problem_models`), structured analysis records (`analysis_records`), causal graph analysis, autonomous repeated task failure escalation (AIA-22 §10), 11-step Autonomy Loop integration, and diagnostic CLI/REST interface (Migration `018`). |
| **`internal/domain/plan` & `service/problem`** | **Production Ready (AIA-22)** | Complex Problem Planning & Change Management | Executable plan generation (`plans`), versioned plan modifications, formal change proposals (`plan_changes`), evidence references, impact analysis, work item task materialization, and Autonomy Cycle execution. Plan models live in `domain/plan`; coordination via `service/problem`. |
| **`internal/service/backup` & `internal/telemetry`** | **Production Ready (AIA-19)** | Backup, Tracing & Operational Metrics | Online SQLite hot backup & restore via SQLite Backup API (`BackupService`), SHA-256 integrity verification, distributed tracing with span hierarchies (`telemetry/tracing`), process & system operational metrics (`telemetry/metrics`), and background expired log retention purger. |

---

## 3. Database Schema Status

### Active Tables (Migrations 001 ~ 019)
- `agents`: Registered agents, capabilities, configurations.
- `tasks`: Work units, status machine (`READY`, `RUNNING`, `COMPLETED`, `WAITING_APPROVAL`).
- `task_dependencies`: Inter-task DAG dependencies.
- `executions`: Task execution instances, status codes, and token telemetry (`input_tokens`, `output_tokens`, `total_tokens`, `estimated_cost_usd` added in Migration 011).
- `execution_events`: Execution log stream and events.
- `memories`: Stored long-term memories with multi-dimensional provenance columns (`provenance_type`, `provenance_ref_type`, `provenance_ref_id`, `provenance_status`).
- `memory_candidates`: Pending insights awaiting promotion.
- `projects`: Workspace directory registrations with continuity fields (`primary_goal_id`, `latest_snapshot_id`, `pending_issues_json`) and domain routing (`domain_type` added in Migration 013).
- `artifacts`: Generated artifacts and file hashes.
- `audit_logs`: System audit trail, expanded in 009 with `policy_id`, `decision_id`, `approval_id`, `risk_tier`, `governance_action`.
- `schedule_rules`: Periodic cron and interval rules.
- `investment_*` (11 tables): Portfolios, positions, open orders, decisions, evidence, predictions.
- `event_history`: Append-only immutable event store with SHA-256 hash chaining and SQLite `ABORT` triggers (Migration 007).
- `sources`: Data origins with 5-dimensional reliability scores (Authority, Directness, Timeliness, Independence, Reproducibility) (Migration 007).
- `evidence`: Grounded excerpts with capture method, data version, content hash, and temporal validity (Migration 007).
- `claims`, `claim_revisions`, `claim_evidence_links`: Testable assertions and immutable revision rationales (Migration 007).
- `autonomy_cycles`: Temporal autonomy hub tracking cycle numbers, triggers, and execution lifecycle (Migration 008).
- `goals`, `goal_revisions`, `goal_project_links`: Multi-level goal tree with status machine, typed evolution (`revision_type`, `approval_id`, `evidence_ids_json` added in Migration 011), and M:N project mapping (Migration 008 / 011).
- `context_snapshots`: Sanitized workspace environment, pending issues, and active task states (Migration 008).
- `decisions`, `decision_evidence_links`: Deliberative choices supporting `NO_ACTION`, governance results, and evidence links (Migration 008).
- `approvals`: Scope-bound human approval gates (Migration 009).
- `runtime_controls`: Single-row global dispatch control with anti-deletion trigger (Migration 009).
- `candidate_initiatives`: Multi-source autonomous discovery records with confidence, priority, cost/risk, and spawned task linkage (Migration 010).
- `deliberation_records`: Structured Expected Utility (`EV`) option trade-off matrices linked to cycles and decisions (Migration 010).
- `goal_task_links`: Traceable Goal-to-Task decomposition bindings (Migration 010).
- `outcome_evaluations`: Expected vs. observed outcome discrepancies, magnitude, calibration error, and evaluation method (`CONTROL_GROUP`, `HISTORICAL_REPLAY`, `SIMILAR_CONTEXT`, `PRE_POST_ONLY`) (Migration 011).
- `root_cause_analyses`: Testable root-cause hypotheses, category taxonomy, alternative explanations, and supporting/refuting evidence IDs (Migration 011).
- `correction_actions`: Governed corrections bound to the 3-Tier Runtime Profile (`TIER1_AUTO`, `TIER2_APPROVAL`, `TIER3_BLOCKED`) with DB-level `CHECK` blocking `SCOPE_C` and `CODE_PATCH` in Phase 4, plus prior/proposed state snapshots for deterministic rollback (Migration 011).
- `learning_records`: Epistemic learning records enforcing `L0_RECORDED` ～ `L3_GENERALIZED`, the 7 mandatory comparison fields (`baseline_json`, `hypothesis`, `intervention_json`, `verification_json`, `regression_json`, `follow_up_json`, `confidence`), `PRE_POST_ONLY <= 0.45` confidence cap, and negative constraint suppression (`suppressed_pattern_key`) (Migration 011).

- `experiments`, `experiment_goal_links`: Longitudinal autonomous experiments enforcing all 11 AIA-15 lifecycle states (Migration 012).
- `experiment_generations`: Generational cohorts (`G1` frozen rule baseline vs `G2` / `G2.1` / `G2.2` LLM cohorts) with isolated portfolio profile bindings and SQLite triggers (`trg_exp_gen_freeze_update`, `trg_exp_gen_freeze_delete`) blocking mutation/deletion of frozen `G1` (Migration 012).
- `experiment_comparisons`: Point-in-time Track A (performance + Brier/LogLoss & decision confidence calibration) and Track B (autonomy + 5-regime verdict) comparison snapshots (Migration 012).
- `delegation_requests`: Cross-agent delegation contract records with acceptance gates (Migration 012).
- `self_modification_proposals`: Self-modification proposal tracking with scope levels (`LEVEL_A_DOMAIN`, `LEVEL_B_CORE`, `LEVEL_C_GOVERNANCE`), patch diff, status machine (`DRAFT`, `IN_SANDBOX`, `PENDING_APPROVAL`, `APPROVED`, `VERIFIED`, `MERGED`, `ROLLED_BACK`, `REJECTED`, `REQUEST_CHANGES`, `DEFERRED`), verification test logs, multi-tier validation plans (`L1_SYNTAX` ~ `L6_AUTONOMY`), post-deployment monitoring metrics (`error_rate`, `crash_rate`, `autonomy_metric`), and approval binding (Migration 014 / 019).
- `writing_briefs`: Creative brief tracking with genre, era, scale, protagonists, worldview, status, and pending questions (Migration 015).
- `writing_clarifications`: Domain-specific human clarification requests (`ASK_HUMAN` / `WAITING_FOR_INPUT`) with category, options, user answer, and status (Migration 015).
- `project_briefs`: Universal project requirements model across all domains with brief version, readiness stage, and YAML payload (Migration 016).
- `brief_field_history`: Immutable field revision tracking preserving version, old/new value, old/new status, source, rationale, task ID, and clarification ID (Migration 016).
- `clarification_requests`: Generic human-in-the-loop clarification requests supporting domain type, scope, options, answers, and brief field binding (Migration 017).
- `problems`, `problem_models`, `analysis_records`: Universal complex problem tracking, multi-version formal problem models, and multi-stage analysis records (Migration 018).
- `plans`, `plan_changes`: Multi-version executable planning records, work items, acceptance criteria, and formal plan change proposals with impact analysis (Migration 018).

---

## 4. Phase Completion Status & Specification Coverage

AIA 採用三維度度量體系評估各項能力，避免以單一狀態標籤混淆規格、程式與驗證：

1. **Specification Maturity（規格成熟度）**：AIA-00 至 AIA-22 全部 23 份規格書皆已正式核定（Normative / Baseline）。
2. **Implementation Readiness（程式碼實作狀態）**：Phase 0 ～ Phase 6 核心以及 AIA-21（通用綱要與釐清）、AIA-22（複雜問題建模與規劃）、AIA-19（備份、追蹤與指標）、AIA-13（自主性度量模組）、AIA-14（多層驗證與監控）與 AIA-06（16 分頁 Web Console 與緊急停機）皆已全量落實於 Go 程式庫與 SQLite Migrations `001`～`019`。
3. **Verification Scope（行為驗證狀態）**：
   - **單元與整合回歸門禁**：全量自動化測試（含 `go/tests/unit/` 與 `go/tests/integration/`）100% 通過（`ok`）。
   - **真實專案端到端自主行為驗收（Phase 7 Active Verification）**：正針對 Writing（小說 6 卷 540 章規範稽核）、Coding（代碼缺陷因果圖與沙盒自修）與 Investment（虛擬資金雙世代演化與零前視偏差回放）三大場景展開端到端自主性實證。

> [!NOTE]
> **Evidence Directory Note (`.aia/`)**: The Phase 6 readiness verification evidence artifacts are located at `.aia/evidence/phase6_readiness/` (`scenario_a.yaml` ~ `scenario_e.yaml`) and are tracked in Git. When creating release tarballs or archives, ensure hidden dot-directories are not excluded by standard tar/zip filters.

---

## 5. Architectural Trade-offs & Known Technical Debts

### 5.1 Architectural Trade-offs

#### SQLite Single-Writer Lock vs. Embedded Simplicity
- **Decision:** SQLite with WAL mode (`PRAGMA journal_mode=WAL`) was chosen for zero-dependency, local-first portability.
- **Trade-off:** While WAL allows concurrent readers alongside a writer, SQLite permits only one writer at any given moment. Under high worker concurrency, database write contention could result in `busy` timeouts if transactions are held open.
- **Mitigation:** TaskWorker concurrency is throttled to 3 parallel workers, and write transactions in `storage/` are kept short and granular.

#### Embedded Polling vs. Full Server-Sent Events (SSE)
- **Decision:** The Web Console polls `/api/v1/system/activity` periodically, while execution log streaming uses SSE (`/api/v1/events`).
- **Trade-off:** Polling introduces a slight telemetry lag (3–5 seconds) and unnecessary local requests when the dashboard is open.
- **Future Direction:** Unify all dashboard metrics into a single real-time SSE telemetry channel.

#### Fixed Semiconductor Universe vs. Dynamic Watchlists
- **Decision:** Initial data collectors focus on core semiconductor and technology leaders (TSMC 2330, Foxconn 2317, MediaTek 2454) to establish a deterministic testing baseline.
- **Trade-off:** Users cannot yet define dynamic multi-sector watchlists via the Web UI without updating the collector configurations.
- **Future Direction:** Introduce a dynamic `watchlist` database table with user-configurable tickers and custom collection schedules.

### 5.2 Technical Debt Items

#### Debt Item 1: Test Suite Mock Executor vs. Headless agy E2E Harness *(Resolved in Phase 3.5)*
- **Status:** **Resolved** (with graceful skip on unauthenticated CI runners)
- **Description:** Unit and deterministic integration tests use `MockExecutor`, while [`go/tests/integration/antigravity_real_test.go`](file:///home/wade/src/github/aia/go/tests/integration/antigravity_real_test.go) exercises the real `agy` CLI over headless `stream-json` inside [`sandbox.go`](file:///home/wade/src/github/aia/go/internal/provider/antigravity/sandbox.go) environment scrubbing, paired with [`validator.go`](file:///home/wade/src/github/aia/go/internal/validation/validator.go) enforcing output/artifact/fatal-marker checks rather than trusting exit code `0` alone.
- **Residual Constraint:** Automated CI runners without a pre-authenticated `agy` Google OAuth session skip live LLM invocation tests automatically; local release sign-off must run on an authenticated workstation.

#### Debt Item 2: Dynamic Prediction Horizon Resolution
- **Severity:** Low
- **Description:** Predictions currently use fixed evaluation windows (e.g., 30, 60, or 90 days). Event-driven horizons (e.g., "until next earnings call" or "until central bank rate decision") are not dynamically parsed from the thesis text.
- **Remediation:** Enhance `investment.PredictionTarget` to support event-driven trigger dates extracted by the LLM during decision formulation.

#### Debt Item 3: Frontend Build Pipeline
- **Severity:** Low
- **Description:** The embedded web frontend is written in vanilla HTML/CSS/JavaScript with zero build step, bundled directly into the Go binary. While eliminating `npm`/`webpack` dependencies, complex state management is handled manually in DOM operations.
- **Remediation:** Retain the zero-node build ethos, but modularize JavaScript components into clean ES modules if UI complexity expands.
