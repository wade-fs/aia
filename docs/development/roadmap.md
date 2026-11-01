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
│    [COMPLETED]    │    [COMPLETED]    │           [PLANNED]            │
├───────────────────┼───────────────────┼────────────────────────────────┤
│ Phase 3: Autonomy │ Phase 4: Self-    │ Phase 5: Domain Experiments &  │
│ Core & Initiative │ Correction & Goal │ Cross-Domain Collaboration     │
│     [PLANNED]     │     [PLANNED]     │           [PLANNED]            │
├───────────────────┴───────────────────┴────────────────────────────────┤
│ Phase 6: Safe Self-Modification & Autonomous System Evolution [PLANNED]│
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Autonomy Milestones (M0 ～ M8)

Progress is evaluated by verifiable autonomous behavior rather than package or table count:

- **M0: Persistent Runtime** ── Crash-free runtime daemon, concurrent workers, SQLite persistence. *(Baseline Ready at commit `7597902`)*
- **M1: Temporal Continuity** ── Restoration of Project state, goals, and pending issues from Event history across process termination. *(Phase 1)*
- **M2: Trustworthy Decision** ── Provenance verification linking every assertion and decision to original evidence and sources. *(Phase 1)*
- **M3: Initiative & Rationality** ── Proactive discovery of valuable work; explicit and auditable `NO_ACTION` decisions when inaction is optimal. *(Phase 3)*
- **M4: Autonomous Closed Loop** ── Execution of the 11-step loop (`Observe` to `Reassess`) with preliminary outcome verification. *(Phase 3)*
- **M5: Self-Correction & RCA** ── Identification of outcome discrepancies, automated Root Cause Analysis, and application of lessons to future choices. *(Phase 4)*
- **M6: Goal Evolution** ── Dynamic goal optimization, decomposition, and revision proposals based on longitudinal empirical outcomes. *(Phase 4)*
- **M7: Domain Experiments** ── Autonomous management of live virtual capital (NT$100,000) by Investment Agent, monthly self-review, and cross-domain tool delegation. *(Phase 5)*
- **M8: Safe Self-Modification** ── Isolated sandbox code modification, automated multi-level test verification, human approval gate, and automatic rollback. *(Phase 6)*

---

## 4. Phase Breakdown & Deliverables

### Phase 0: Baseline & Alignment *(Current)*
- Audit codebase at commit [`7597902`](file:///home/wade/src/github/aia).
- Cross-specification consistency audit ([AIA-SPEC-AUDIT-01](file:///home/wade/src/github/aia/docs/specifications/cross-spec-consistency-audit.md)).
- Architecture-to-code mapping ([AIA-ARCH-MAP-01](file:///home/wade/src/github/aia/docs/architecture/architecture-to-code-mapping.md)).
- Unified Data Model & lifecycle specification ([AIA-ARCH-DATA-01](file:///home/wade/src/github/aia/docs/architecture/data-model.md)).
- Sequential migration roadmap ([AIA-DEV-MIG-01](file:///home/wade/src/github/aia/docs/development/migration-plan.md)).

### Phase 1: Persistent State & Knowledge Integrity
- Append-only `event_history` table as primary source of historical truth.
- `sources`, `evidence`, `claims`, and `claim_evidence_links` implementation.
- Flexible `memories` provenance tagging (`provenance_type`, confidence, claim reference).
- `ContextReconstructor` restoring project state, goals, and pending issues after process restart.

### Phase 2: Governance & Operational Control
- Separation of Policy (rule evaluation), Approval (human review), and Permission (environment boundary).
- Controlled Emergency Stop: 1-second freeze on new task dispatch, graceful cancellation with state protection.
- Web Console & CLI pending approval interface with one-click decision.

### Phase 3: Autonomy Core & Initiative
- Shared `internal/autonomy/` package coordinating the complete 11-step autonomous cycle.
- Diverse decision outcomes: `ACT`, `DEFER`, `MONITOR`, `ASK_HUMAN`, `NO_ACTION`.
- Goal management entity and task-to-goal alignment.
- Investment Agent autonomous pilot loop.

### Phase 4: Self-Correction & Goal Evolution
- Automated outcome discrepancy detection and Root Cause Analysis (RCA).
- Six levels of self-correction (Fact, Judgment, Strategy, Prompt, Workflow, Code).
- Goal progress evaluation, priority adjustment, and revision proposals.
- [AIA-13](file:///home/wade/src/github/aia/docs/specifications/AIA-13-Autonomy-Evaluation-and-Benchmarking-Specification.md) baseline metrics tracking.

### Phase 5: Domain Experiments & Cross-Domain Collaboration
- Investment Agent complete autonomous experiment lifecycle ([AIA-05](file:///home/wade/src/github/aia/docs/specifications/AIA-05-Investment-Agent-Specification.md)).
- Coding Agent and Writing Agent specialized integrations.
- Cross-domain delegation protocol (`DelegationRequest`).
- Longitudinal experiment tracking ([AIA-15](file:///home/wade/src/github/aia/docs/specifications/AIA-15-Autonomous-Experiment-Lifecycle-and-Domain-Integration-Specification.md)).

### Phase 6: Safe Self-Modification & Autonomous System Evolution
- Scope isolation: Level A (Domain), Level B (AIA Core - Human Approval Required), Level C (Governance - Locked).
- Automated self-modification pipeline: Proposal ➔ Sandbox ➔ Multi-level Tests ➔ Diff Review ➔ Approval ➔ Deployment ➔ Verification ➔ Rollback.
