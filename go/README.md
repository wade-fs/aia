# AIA Go Workspace (`go/`)

This directory contains the Go runtime, 11-step Autonomy Core, Generational Experiment Controller, SQLite persistence layer (Migrations `001`–`019`), embedded Web Console (16 tabs & Emergency Stop), Cobra CLI, and automated unit/integration test suites for **AIA (Autonomous Intelligence Architecture)**.

---

## 1. Directory Layout

```text
go/
├── cmd/aia/                  # Main binary entrypoint
├── internal/
│   ├── app/                  # Application dependency wiring & lifecycle bootstrap
│   ├── audit/                # Governance & audit context helpers
│   ├── autonomy/             # Shared 11-step Autonomy Core
│   │   ├── clock/            # Point-in-Time (AsOf) Clock & replay mode guard
│   │   ├── cycle/            # 11-step Coordinator, budget pre-flight, learning & problem hooks
│   │   ├── deliberation/     # Expected-Value deliberation & negative learning constraints
│   │   ├── evaluation/       # AIA-13 general autonomy evaluation metrics module
│   │   ├── initiative/       # CandidateInitiative discovery & deduplication
│   │   ├── learning/         # Discrepancy eval, RCA, 3-Tier Correction & L0~L3 verification
│   │   ├── observation/      # Domain & workspace observers
│   │   ├── recall/           # L1~L5 associative recall with <= asOf temporal filtering
│   │   └── self_evolution_engine.go # Autonomous code modification & evolution engine
│   ├── cli/                  # Cobra CLI commands (agent, approval, backup, clarification, coding,
│   │                         #               cycle, decision, delegation, experiment, goal, governance,
│   │                         #               investment, learning, memory, problem, project, schedule,
│   │                         #               task, trace, worker, writing)
│   ├── collector/            # TWSE, TPEx, FRED, SEC, and RSS data collectors
│   ├── config/               # YAML & environment configuration (incl. Budget, Sandbox, Logging)
│   ├── continuity/           # ContextReconstructor & sanitized snapshots
│   ├── domain/               # Domain models (agent, artifact, autonomy, brief, clarification,
│   │                         #               coding, consistency, continuity, decision, execution,
│   │                         #               experiment, goal, governance, investment, knowledge,
│   │                         #               learning, memory, plan, policy, problem, profile,
│   │                         #               project, schedule, selfmod, task, writing)
│   ├── eventbus/             # In-process pub/sub event bus
│   ├── experiment/           # Generational Experiment Controller (G1 frozen baseline vs G2/G2.x LLM)
│   ├── orchestrator/         # Scheduler, ContextAssembler & PersonalAgent Strangler-Fig facade
│   ├── policy/               # Path/financial policies & pre-flight token/cycle budget gates
│   ├── provider/             # Antigravity CLI (agy) sandboxed adapter & mock executor
│   ├── recovery/             # Startup crash recovery
│   ├── service/              # Services (agent, approval, backup, brief, clarification, delegation,
│   │                         #           execution, goal, governance, investment, knowledge, memory,
│   │                         #           problem, project, schedule, selfmod, task, writing)
│   ├── storage/sqlite/       # SQLite WAL pool, migrations 001~019 & repositories
│   ├── telemetry/            # Distributed Tracing & Operational Metrics (AIA-19)
│   ├── validation/           # Output/artifact/fatal-marker execution validator
│   ├── web/                  # Loopback HTTP host (7777), OTT auth & Console REST API
│   ├── worker/               # Resident ScheduleWorker (15s), TaskWorker (5s, 3 slots),
│   │                         # LogRetentionPurger & Autonomy AutoWakeWorker
│   └── workspace/            # Project workspace isolation & Git worktree helpers
├── prompts/                  # Domain & coordinator prompt templates
├── scripts/service.sh        # systemd --user installation & lifecycle helper
├── web/assets/dist/          # Web Console single-page assets (HTML, CSS, JS with 16 tabs)
└── tests/
    ├── unit/                 # Unit test suites (Autonomy, Selfmod, Problems, Tracing, Migrations 001~019)
    └── integration/          # End-to-end SQLite, Autonomy, Learning, Safe-Selfmod & live agy suites
```

---

## 2. Build & Verification

```bash
# Static analysis
go vet ./...

# Build CLI binary
go build ./cmd/aia

# Run all unit and integration tests (uncached)
go test -count=1 ./...
```

---

## 3. Architectural References

- Master Roadmap: [docs/development/roadmap.md](file:///home/wade/src/github/aia/docs/development/roadmap.md)
- Implementation Status: [docs/development/implementation-status.md](file:///home/wade/src/github/aia/docs/development/implementation-status.md)
- Unified Data Model: [docs/architecture/data-model.md](file:///home/wade/src/github/aia/docs/architecture/data-model.md)
- Security & Subprocess Sandbox: [docs/architecture/security-model.md](file:///home/wade/src/github/aia/docs/architecture/security-model.md)
