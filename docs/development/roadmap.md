# AIA Development: Master Roadmap

**Project:** Autonomous Intelligence Architecture (AIA)  
**Version:** v2.0 (Official Baseline)  
**Baseline Roadmap:** Master Implementation Plan fulfilled across Phases 0–7  
**Specification Baseline:** [docs/specifications/README.md](file:///home/wade/src/github/aia/docs/specifications/README.md) (Six Consolidated Volumes: `AIA-VOL-01` ～ `AIA-VOL-06`; historical `AIA-00` ～ `AIA-22` frozen baseline)  
**Last Updated:** 2026-11-05  

---

## 1. Architectural Scope & Strategic Alignment

### Platform Scope & Evolution
AIA operates as a resident, local-first autonomous intelligence platform (`systemd --user`) on Linux / macOS. It coordinates specialized domain agents (Investment, Coding, Writing), manages long-term project continuity, maintains verifiable evidence provenance, and continuously improves its capabilities through safe closed-loop self-correction.

AIA executes a **Dual-Track Evolution Model**:
- **Infrastructure Track:** Runtime ➔ Continuity ➔ Evidence ➔ Governance ➔ Autonomy Core ➔ Learning ➔ Safe Evolution ➔ Universal Planning.
- **Autonomy Capability Validation Track:** Investment Pilot ➔ Closed Loop ➔ Self-Correction ➔ Multi-Domain Integration ➔ Safe Self-Modification ➔ Complex Problem Decomposition.

---

## 2. Master Implementation Roadmap

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        Master Development Phases                       │
├───────────────────┬───────────────────┬────────────────────────────────┤
│ Phase 0: Baseline │ Phase 1: State &  │ Phase 2: Governance &          │
│ & Alignment       │ Knowledge Integ.  │ Operational Control            │
│    [COMPLETED]    │    [COMPLETED]    │          [COMPLETED]           │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ Phase 3 & 3.5:    │ Phase 4: Verif.   │ Phase 5: Domain Experiments,   │
│ Autonomy & Harden │ Learning & Goal   │ MVA & Cross-Domain Delegation  │
│    [COMPLETED]    │    [COMPLETED]    │          [COMPLETED]           │
├───────────────────┴───────────────────┴────────────────────────────────┤
│ Phase 6: Safe Self-Modification & Autonomous System Evolution          │
│                              [COMPLETED]                               │
├────────────────────────────────────────────────────────────────────────┤
│ Phase 7: Universal Brief, Clarification, Complex Problem Planning &   │
│ Systemic Autonomy Hardening (AIA-06, 13, 14, 19, 21, 22)               │
│ [DATA, SERVICES & CYCLE INTEGRATION COMPLETED; VERIFICATION ACTIVE]   │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Autonomy Milestones (M0 ～ M10)

Progress is evaluated by verifiable, computable autonomous behavior rather than package or table count (see [AIA-13](file:///home/wade/src/github/aia/docs/specifications/AIA-13-Autonomy-Evaluation-and-Benchmarking-Specification.md) for M3–M7 quantitative acceptance thresholds):

- **M0: Persistent Runtime** ── Crash-free runtime daemon, concurrent workers, SQLite persistence. *(Completed at Baseline `7597902`)*
- **M1: Temporal Continuity** ── Restoration of Project state, goals, and pending issues from Event history across process termination. *(Completed — Phase 1)*
- **M2: Trustworthy Decision** ── Provenance verification linking every assertion and decision to original evidence and sources. *(Completed — Phase 1)*
- **M3: Initiative & Rationality** ── Proactive discovery of valuable work; explicit, auditable `NO_ACTION` decisions with counterfactual calibration (`>= 70%` post-hoc accuracy over `N >= 10`). *(Completed — Phase 3 / 3.5)*
- **M4: Autonomous Closed Loop** ── Execution of the 11-step loop (`Observe` to `Reassess`) under hard daily token/cycle budget gates and Point-in-Time (`AsOf` Clock) zero-look-ahead replay. *(Completed — Phase 3 / 3.5)*
- **M5: Verifiable Self-Correction & Learning Mechanism** ── Discrepancy detection, testable RCA hypotheses, table-driven 3-tier runtime profile (`TIER1_AUTO`, `TIER2_APPROVAL`, `TIER3_BLOCKED`), 4 learning maturity levels (`L0_RECORDED` ～ `L3_GENERALIZED`), 7 epistemic comparison fields, regression rollback, and `INCONCLUSIVE` / `<= 0.45` confidence caps. *(Completed — Phase 4)*
- **M6: Governed Goal Evolution** ── 3-way goal evolution taxonomy (`PERSISTENCE`, `OPTIMIZATION`, `REVISION`) with Anti-Goalpost-Moving guard preventing unapproved metric weakening on `STRATEGIC` goals. *(Completed — Phase 4)*
- **M7: Domain Experiments & Longitudinal Validation** ── Longitudinal validation of `L3_GENERALIZED` learning across Investment (NT$100,000 virtual capital + historical PiT replay), Coding, and Writing experiments, plus cross-domain `DelegationRequest`. *(Completed — Phase 5)*
- **M8: Safe Self-Modification** ── OS-isolated sandbox code modification, multi-tier validation plans (`V1_SYNTAX` ~ `V6_AUTONOMY`), RCA-based failure retries, post-deployment metrics monitoring, human approval gate, and zero-residual rollback (Phase 6, Migrations `014` & `019`). *(Completed)*
- **M9: Universal Intent & Proactive Clarification** ── Dynamic Project Brief YAML generation, schema validation, immutable field history tracking (`brief_field_history`), blocking clarification management, and Brief Observer autonomous perception (AIA-21, Migrations `016` & `017`). *(Completed in Data, Services & Observers)*
- **M10: Complex Problem Modeling & Versioned Planning** ── Multi-perspective problem modeling (objectives, facts, assumptions, risks), analysis deduction records, versioned execution plans, autonomous escalation from repeated task failures (AIA-22 §10), and complete 11-step Autonomy Loop integration (AIA-22, Migration `018`). *(Completed in Data, Services & Autonomy Loop)*

---

## 4. Phase Breakdown & Deliverables

### Phase 0: Baseline & Alignment *(Completed)*
- Audit codebase at commit [`7597902`](file:///home/wade/src/github/aia).
- Cross-specification consistency audit ([AIA-SPEC-AUDIT-01](file:///home/wade/src/github/aia/docs/specifications/cross-spec-consistency-audit.md)).
- Architecture-to-code mapping ([AIA-ARCH-MAP-01](file:///home/wade/src/github/aia/docs/architecture/architecture-to-code-mapping.md)).
- Unified Data Model & lifecycle specification ([AIA-ARCH-DATA-01](file:///home/wade/src/github/aia/docs/architecture/data-model.md)).
- Sequential migration roadmap ([AIA-DEV-MIG-01](file:///home/wade/src/github/aia/docs/development/migration-plan.md)).

### Phase 1: Persistent State & Knowledge Integrity *(Completed)*
- Append-only `event_history` table as primary source of historical truth.
- `sources`, `evidence`, `claims`, and `claim_evidence_links` implementation.
- Flexible `memories` provenance tagging (`provenance_type`, confidence, claim reference).
- `ContextReconstructor` restoring project state, goals, and pending issues after process restart.

### Phase 2: Governance & Operational Control *(Completed)*
- Separation of Policy (rule evaluation), Approval (human review), and Permission (environment boundary).
- Controlled Emergency Stop: 1-second freeze on new task dispatch, graceful cancellation with state protection.
- Web Console & CLI pending approval interface with one-click decision.

### Phase 3 & 3.5: Autonomy Core, Initiative & Platform Hardening *(Completed)*
- Shared `internal/autonomy/` package coordinating the complete 11-step autonomous cycle.
- Diverse decision outcomes: `ACT`, `DEFER`, `MONITOR`, `ASK_HUMAN`, `NO_ACTION`.
- Goal management entity, task-to-goal alignment, and Investment Agent autonomous pilot loop.
- **Phase 3.5 Hardening**: Pre-flight daily token & cycle budget gates, subprocess secret scrubbing & OS mount-namespace isolation (`bwrap` / `systemd-run --user`), harness output/artifact validator hardening (never trusting exit code 0 alone), and Point-in-Time (`AsOf` Clock) zero-look-ahead historical replay.

### Phase 4: Verifiable Self-Correction, Learning Mechanism & Governed Goal Evolution *(Completed)*
- Repositioned as **establishing a verifiable self-correction and learning mechanism** (rather than claiming open-ended self-learning completion in isolation).
- Automated outcome discrepancy detection, testable Root Cause Analysis (`root_cause_analyses`), and table-driven 3-Tier Runtime Profile (`TIER1_AUTO`, `TIER2_APPROVAL`, `TIER3_BLOCKED`).
- Four Learning Maturity Levels (`L0_RECORDED`, `L1_APPLIED`, `L2_VERIFIED`, `L3_GENERALIZED`), 7 mandatory comparison fields (`Baseline`, `Hypothesis`, `Intervention`, `Verification`, `Regression`, `Follow-up`, `Confidence`), `PRE_POST_ONLY` confidence cap (`<= 0.45`), automatic regression rollback, and negative constraint suppression.
- Governed Goal Evolution (`PERSISTENCE`, `OPTIMIZATION`, `REVISION`) with deterministic Anti-Goalpost-Moving enforcement.
- Computable [AIA-13](file:///home/wade/src/github/aia/docs/specifications/AIA-13-Autonomy-Evaluation-and-Benchmarking-Specification.md) M3–M7 thresholds and Learning Effectiveness Acceptance Suite (`Scenarios A–E`).

### Phase 5: Domain Experiments, Multi-Domain MVA & Cross-Domain Delegation *(Completed)*
- **Phase 5A (Migration `012`)**: Generational Investment Decision Experiment (`G1` Frozen Rule-Based Control Baseline vs `G2 -> G2.1 -> G2.2` LLM Experimental Cohorts, isolated NT$100,000 ledgers, Brier + LogLoss event calibration separated from decision confidence, 5-regime classification matrix, and live `G1` Difference-in-Differences control verification bridged into Phase 4 `LearningEngine`).
- **Phase 5B (Multi-Domain MVA & Entry Gate)**:
  - **Shared Autonomy Core Alignment**: Unified 11-step Autonomy Loop (`internal/autonomy/`) running simultaneously across Investment, Coding, and Writing domains with rational `NO_ACTION` baseline.
  - **Coding Agent MVA (AIA-15 §24)**: `CodeHealthObserver`, `CodingPilotExecutor`, sandbox test harness, deterministic test self-correction reaching `L2_VERIFIED`.
  - **Writing Agent MVA (AIA-15 §23)**: `ManuscriptConsistencyObserver` targeting `/home/wade/src/github/novel/yuan/` with 3 knowledge tiers (`FACT_HISTORY`, `PLAUSIBLE_INFERENCE`, `FICTION`), distinguishing artistic license (rational `NO_ACTION`) from chronological contradictions (`POST_MORTEM_APPEARANCE` -> `L2_VERIFIED`).
  - **Cross-Domain Delegation (AIA-12 §21)**: `DelegationService` managing `delegation_requests` lifecycle state machine (`PENDING` -> `IN_PROGRESS` -> `DELIVERED` -> `VERIFIED`) with CLI commands (`aia delegation ...`) and Web Console API routes (`/api/v1/delegations`).
  - **Phase 5 Entry Gate (AIA-15 §22)**: All 7 criteria verified green in integration acceptance suite (`TestPhase5B_EntryGate_*`).

### Phase 6: Safe Self-Modification & Autonomous System Evolution *(Completed)*
- **Scope Isolation & Tri-Level Governance (AIA-14 §2)**:
  - **Level A (Domain Scope)**: Domain tools, collectors, scripts, and prompts; autonomous branch creation, sandbox test verification, L2 regression check, and autonomous merge.
  - **Level B (AIA Core Non-Governance Scope)**: Core services, workers, domain models, and autonomy runtime; verified in isolated sandbox, followed by a mandatory **Human Approval Gate** (`WAITING_APPROVAL`); agent self-approval is strictly forbidden and rejected (`ErrAgentSelfApprovalForbidden`).
  - **Level C (Governance, Policy, Security & Migration Scope)**: Core policy (`internal/policy/`), approval rules (`internal/service/approval/`, `internal/domain/governance/`), database schema migrations (`internal/storage/sqlite/migrations/`), environment config, and system databases. Strictly immutable and locked; any agent self-modification attempt fails closed immediately with high-severity audit logging (`ErrGovernanceImmutable`).
- **Migrations `014` & `019` (`self_modification_proposals`)**: Persists proposals, diff content, scope level (`LEVEL_A`, `LEVEL_B`, `LEVEL_C`), lifecycle status (`PROPOSED` -> `VERIFYING` -> `VERIFIED` / `PENDING_APPROVAL` -> `MERGED` / `REJECTED`), multi-tier validation plans (`validation_plan_json`), multi-level test logs (`verification_results_json`), post-deployment monitoring metrics (`post_deploy_metrics_json`), target branch, and worktree path.
- **Self-Modification Pipeline (`internal/service/selfmod/pipeline.go`)**:
  - `Propose`: Validates, sanitizes, and classifies modification level via `ScopeClassifier`.
  - `ExecuteSandboxVerification`: Clones isolated branch/worktree, applies patch, executes multi-level validation suite (V1 compile, V2 unit, V3 integration, V4 benchmark, V5 lint, V6 security), and captures verification artifacts.
  - `SubmitForApproval`: Creates decision & approval record for Level B changes, transitioning to `PENDING_APPROVAL`.
  - `ApplyMerge`: Merges verified changes into target branch (strictly enforcing human ownership for Level B).
  - **Multi-Level Verification & Retry Loop**: Diagnostic RCA parsing on failure, automatic repair retry loop bounded by `MaxRetries`, and review outcome handling (`REQUEST_CHANGES`, `DEFERRED`, `REJECTED`, `APPROVED`).
  - **Zero-Residual Rollback**: Automatic clean teardown of worktrees, branch pruning, and zero workspace pollution on verification failure.
- **Autonomous System Evolution Engine (`internal/autonomy/self_evolution_engine.go`)**: Coordinates periodic autonomous inspection, hypothesis generation, modification proposal creation, sandbox verification, and end-to-end evolution cycles.
- **Phase 6 Readiness & Entry Gate Verification (G1–G5 All Passed)**:
  - Validated across 5 behavioral acceptance scenarios in [`phase6_readiness_scenarios_test.go`](file:///home/wade/src/github/aia/go/tests/integration/phase6_readiness_scenarios_test.go) with persisted YAML evidence in `.aia/evidence/phase6_readiness/`:
    - **Scenario A (G1 - Writing)**: Real project comprehension (`novel/yuan` 6 volumes, 540 chapters, 90+ character constraints).
    - **Scenario B (G2 - Writing)**: Adaptive conflict detection (distinguishing fiction vs historical hard contradictions).
    - **Scenario C (G3 - Coding)**: Real execution in isolated Git Worktree with L2 test verification and 0 workspace pollution.
    - **Scenario D (G4 - Multi-Domain)**: Autonomous failure detection & self-healing code patch retry loop.
    - **Scenario E (G5 - Coding)**: Zero-residual clean rollback upon unrecoverable error (SHA & workspace 100% preserved).

### Phase 7: Universal Brief, Clarification, Complex Problem Planning & Systemic Hardening *(Data, Services & Cycle Integration Completed; Behavioral Verification Active)*
- **Universal Project Brief & Dynamic Clarification (AIA-21)**:
  - **Migrations `016` (`project_briefs`, `brief_field_history`) & `017` (`clarification_requests`)**: Unified schema persistence for project briefs, immutable field version history, and structured clarification questions with blocking flags.
  - **Domain Brief Service (`internal/service/brief/`)**: Domain template validation, YAML parsing, readiness status checks, and field history tracking.
  - **Clarification Service (`internal/service/clarification/`)**: Management of interactive questions, options serialization, user answer recording, and task unblocking.
  - **Brief Observer (`internal/autonomy/brief_observer.go`)**: Continuous observation of brief omissions, outdated assumptions, and readiness state.
  - **CLI & REST API**: CLI commands (`aia brief`, `aia clarify`) and HTTP endpoints (`/api/v1/briefs`, `/api/v1/clarifications`).
- **Complex Problem Analysis & Versioned Planning (AIA-22)**:
  - **Migration `018` (`problems`, `problem_models`, `analysis_records`, `plans`, `plan_changes`)**: Relational problem modeling (objectives, constraints, facts, assumptions, unknowns, risks, success criteria), deduction analysis records, and versioned plan trees.
  - **Problem Analysis Service (`internal/service/problem/`)**: Multi-stage problem decomposition and validation (verified by `tests/integration/problem_stage1_to_5_test.go`).
  - **Versioned Plan Support (`internal/domain/plan/` + `service/problem/`)**: Versioned work item graphs, supersede chains, and plan alteration governance (models in domain/plan; orchestration via problem service).
  - **Autonomous Escalation from Repeated Failures (AIA-22 §10)**: Coordinator automatically detects repeated task failures and escalates them into formal problem models and versioned plans.
  - **Autonomy Core Integration**: Complete wiring of problem modeling, recall, and plan work item materialization into the 11-step Autonomy Loop (`internal/autonomy/cycle/coordinator.go`), Initiative proactive discovery, and context assembly.
  - **CLI & REST API**: CLI commands (`aia problem` 含 plan 子功能) and HTTP endpoints (`/api/v1/problems`, `/api/v1/plans`).
- **Systemic Operations, Evaluation & Web Console Hardening**:
  - **AIA-13 Autonomy Evaluation & Benchmarking**: `internal/autonomy/evaluation/` (evaluator.go, metrics.go) providing general autonomy indicators (Proactive Rate, Override Frequency, Calibration, Recovery Accuracy, Self-Correction Score).
  - **AIA-19 Operations & Observability**: SQLite Backup & Restore (`BackupService`, CLI `aia backup`), Distributed Tracing (`internal/telemetry/tracing`, CLI `aia trace`), Operational Metrics (`internal/telemetry/metrics`), and background periodic log retention purger (`LogRetentionPurger`).
  - **AIA-06 Web Console**: 16 dedicated tabs (Overview, Tasks, Pending, Goals, Cycles, Learnings, Experiments, Delegations, Clarifications, Problems, Domains, etc.) and Emergency Stop button with immediate freeze & human resume.
- **Next Frontier (Multi-Domain End-to-End Behavioral Verification)**:
  - Validating cross-session continuity and shared problem-solving intelligence across Writing (6 volumes / 540 chapters audit & repair), Coding (repository defect isolation & sandbox test verification), and Investment (closed-loop research, PiT replay, and dual-cohort calibration).
