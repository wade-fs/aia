# AIA Go Workspace (`go/`)

This directory contains the Go runtime, 11-step Autonomy Core, SQLite persistence layer (Migrations `001`–`011`), embedded Web Console, Cobra CLI, and automated unit/integration test suites for **AIA (Autonomous Intelligence Architecture)**.

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
│   │   ├── cycle/            # 11-step Coordinator, budget pre-flight & learning hooks
│   │   ├── deliberation/     # Expected-Value deliberation & negative learning constraints
│   │   ├── initiative/       # CandidateInitiative discovery & deduplication
│   │   ├── learning/         # Discrepancy eval, RCA, 3-Tier Correction & L0~L3 verification
│   │   ├── observation/      # Domain & workspace observers
│   │   └── recall/           # L1~L5 associative recall with <= asOf temporal filtering
│   ├── cli/                  # Cobra CLI commands (cycle, goal, learning, approval, etc.)
│   ├── collector/            # TWSE, TPEx, FRED, SEC, and RSS data collectors
│   ├── config/               # YAML & environment configuration (incl. Budget & Sandbox)
│   ├── continuity/           # ContextReconstructor & sanitized snapshots
│   ├── domain/               # Pure domain models & invariants
│   ├── eventbus/             # In-process pub/sub event bus
│   ├── orchestrator/         # Scheduler & PersonalAgent Strangler-Fig facade
│   ├── policy/               # Path/financial policies & pre-flight token/cycle budget gates
│   ├── provider/             # Antigravity CLI (agy) sandboxed adapter & mock executor
│   ├── recovery/             # Startup crash recovery
│   ├── service/              # Application services (task, exec, inv, knowledge, approval, goal)
│   ├── storage/sqlite/       # SQLite WAL connection pool, migrations 001~011 & repositories
│   ├── validation/           # Output/artifact/fatal-marker execution validator
│   ├── web/                  # Loopback HTTP host (7777), OTT auth & Console REST API
│   ├── worker/               # Resident ScheduleWorker (15s) & TaskWorker (5s, 3 slots)
│   └── workspace/            # Project workspace isolation helpers
├── prompts/                  # Domain & coordinator prompt templates
├── scripts/service.sh        # systemd --user installation & lifecycle helper
└── tests/
    ├── unit/                 # Unit test suites
    └── integration/          # End-to-end SQLite, Autonomy, Learning & live agy suites
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

- Master Plan: [docs/Plan.md](file:///home/wade/src/github/aia/docs/Plan.md)
- Implementation Status: [docs/development/implementation-status.md](file:///home/wade/src/github/aia/docs/development/implementation-status.md)
- Unified Data Model: [docs/architecture/data-model.md](file:///home/wade/src/github/aia/docs/architecture/data-model.md)
- Security & Subprocess Sandbox: [docs/architecture/security-model.md](file:///home/wade/src/github/aia/docs/architecture/security-model.md)
