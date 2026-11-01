# AIA Development: Master Roadmap

**Project:** Autonomous Intelligence Architecture (AIA)  
**Version:** v2.0 (Official Baseline)  
**Master Plan:** [docs/Plan.md](file:///home/wade/src/github/aia/docs/Plan.md)  
**Last Updated:** 2026-11-01  

---

## 1. Architectural Scope & Strategic Alignment

### Platform Scope & Evolution
AIA operates as a resident, local-first autonomous intelligence platform (`systemd --user`) on Linux / macOS. It coordinates specialized domain agents (Investment, Coding, Writing), manages long-term project continuity, maintains verifiable evidence provenance, and continuously improves its capabilities through safe closed-loop self-correction.

AIA executes a **Dual-Track Evolution Model**:
- **Infrastructure Track:** Runtime ➔ Continuity ➔ Evidence ➔ Governance ➔ Autonomy Core ➔ Learning ➔ Safe Evolution.
- **Autonomy Capability Validation Track:** Investment Pilot ➔ Closed Loop ➔ Self-Correction ➔ Multi-Domain Integration ➔ Safe Self-Modification.

---

## 2. Seven-Phase Implementation Roadmap

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
│ Phase 6: Safe Self-Modification & Autonomous System Evolution [PLANNED]│
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Autonomy Milestones (M0 ～ M8)

Progress is evaluated by verifiable, computable autonomous behavior rather than package or table count (see [AIA-13](file:///home/wade/src/github/aia/docs/specifications/AIA-13-Autonomy-Evaluation-and-Benchmarking-Specification.md) for M3–M7 quantitative acceptance thresholds):

- **M0: Persistent Runtime** ── Crash-free runtime daemon, concurrent workers, SQLite persistence. *(Completed at Baseline `7597902`)*
- **M1: Temporal Continuity** ── Restoration of Project state, goals, and pending issues from Event history across process termination. *(Completed — Phase 1)*
- **M2: Trustworthy Decision** ── Provenance verification linking every assertion and decision to original evidence and sources. *(Completed — Phase 1)*
- **M3: Initiative & Rationality** ── Proactive discovery of valuable work; explicit, auditable `NO_ACTION` decisions with counterfactual calibration (`>= 70%` post-hoc accuracy over `N >= 10`). *(Completed — Phase 3 / 3.5)*
- **M4: Autonomous Closed Loop** ── Execution of the 11-step loop (`Observe` to `Reassess`) under hard daily token/cycle budget gates and Point-in-Time (`AsOf` Clock) zero-look-ahead replay. *(Completed — Phase 3 / 3.5)*
- **M5: Verifiable Self-Correction & Learning Mechanism** ── Discrepancy detection, testable RCA hypotheses, table-driven 3-tier runtime profile (`TIER1_AUTO`, `TIER2_APPROVAL`, `TIER3_BLOCKED`), 4 learning maturity levels (`L0_RECORDED` ～ `L3_GENERALIZED`), 7 epistemic comparison fields, regression rollback, and `INCONCLUSIVE` / `<= 0.45` confidence caps. *(Completed — Phase 4)*
- **M6: Governed Goal Evolution** ── 3-way goal evolution taxonomy (`PERSISTENCE`, `OPTIMIZATION`, `REVISION`) with Anti-Goalpost-Moving guard preventing unapproved metric weakening on `STRATEGIC` goals. *(Completed — Phase 4)*
- **M7: Domain Experiments & Longitudinal Validation** ── Longitudinal validation of `L3_GENERALIZED` learning across Investment (NT$100,000 virtual capital + historical PiT replay), Coding, and Writing experiments, plus cross-domain `DelegationRequest`. *(Next Target — Phase 5)*
- **M8: Safe Self-Modification** ── OS-isolated sandbox code modification, automated multi-level test verification, human approval gate, and automatic rollback. *(Planned — Phase 6)*

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

### Phase 6: Safe Self-Modification & Autonomous System Evolution *(Planned)*
- Scope isolation: Level A (Domain), Level B (AIA Core - Human Approval Required), Level C (Governance - Locked via OS namespace & immutable storage protections).
- Automated self-modification pipeline: Proposal ➔ Sandbox ➔ Multi-level Tests ➔ Diff Review ➔ Approval ➔ Deployment ➔ Verification ➔ Rollback.
