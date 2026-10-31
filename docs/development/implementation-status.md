# AIA Development: Implementation Status

This document provides a comprehensive audit of the implemented components, modules, database schemas, and test suites across the AIA codebase.

---

## 1. Go Package Architecture

```text
go/
├── cmd/
│   └── aia/                  # Cobra CLI Entrypoints & Commands
│       ├── main.go           # Root command & flag bindings
│       ├── agent.go          # Agent commands (list, get, register)
│       ├── task.go           # Task lifecycle commands
│       ├── exec.go           # Execution inspection & log tails
│       ├── memory.go         # Memory entry & candidate commands
│       ├── project.go        # Project workspace management
│       ├── invest.go         # Investment decisions & predictions
│       ├── worker.go         # Worker controls & monitor
│       ├── monitor.go        # Top-level activity monitor command
│       └── server.go         # Embedded Web server command
├── internal/
│   ├── agent/                # Agent metadata & lifecycle definitions
│   ├── app/                  # Application bootstrap & dependency wiring
│   ├── host/                 # Embedded Web server, REST API, Auth
│   ├── investment/           # TWSE, MOPS, FRED, FinMind, Brier Scoring
│   ├── memory/               # Short-term and long-term memory store
│   ├── project/              # Workspace project tracker
│   ├── storage/              # SQLite database driver & schema migrations
│   ├── system/               # System health checks & Activity Monitor
│   ├── task/                 # Task state machine & execution records
│   └── worker/               # Background ScheduleWorker & TaskWorker
├── scripts/
│   └── service.sh            # systemd --user service management utility
└── Makefile                  # Build, test, and service automation
```

---

## 2. Module Implementation Matrix

| Module | Implementation Status | Test Coverage | Key Capabilities |
| :--- | :--- | :--- | :--- |
| **`internal/app`** | Production Ready | High | Factory initialization, default agent seeding, DB initialization. |
| **`internal/storage`** | Production Ready | High | SQLite WAL mode, foreign keys, schema migrations, connection pooling. |
| **`internal/worker`** | Production Ready | High | `Manager`, `ScheduleWorker` (15s), `TaskWorker` (5s, 3 parallel slots). |
| **`internal/investment`**| Production Ready | High | TWSE OpenAPI, MOPS announcements, FRED client, Materiality Gate, Brier calibration. |
| **`internal/host`** | Production Ready | High | Port 7777, one-time token auth, HTTP-only session cookie, REST API, embedded assets. |
| **`internal/system`** | Production Ready | High | Health status diagnostics, real-time Activity Monitor snapshot computation. |
| **`internal/task`** | Production Ready | High | Task state machine (`DRAFT`, `READY`, `RUNNING`, `COMPLETED`, `WAITING_APPROVAL`). |
| **`internal/memory`** | Production Ready | High | Long-term memory records and candidate review workflow. |
| **`internal/project`** | Production Ready | High | Workspace directory tracking and active project metadata. |

---

## 3. Database Schema

All tables reside in SQLite at `~/.local/share/aia/aia.db`:

- `agents`: Registered AI agents, capabilities, and configurations.
- `tasks`: Work units, status, priority, and assigned agent.
- `executions`: Task execution runs, start/end times, and exit status.
- `execution_events`: Detailed execution event stream and logs.
- `memories`: Categorized long-term memory entries.
- `memory_candidates`: Pending insights awaiting promotion.
- `projects`: Registered workspace project directories.
- `schedule_rules`: Periodic cron and interval rules for background tasks.
- `investment_decisions`: Formulated BUY/SELL/HOLD theses and target ranges.
- `investment_predictions`: Quantified forecasts tracked for Brier scoring.

---

## 4. CLI Command Reference

```bash
# System & Monitoring
aia health                      # Inspect database, agent, and service health
aia monitor                     # Real-time AIA Activity Monitor table
aia worker monitor              # Worker operational status

# Background Daemon & Server
aia server start --port 7777    # Run embedded console (uses AIA_HOST / AIA_PORT)
aia server token                # Inspect or generate active login token

# Core Entities
aia agent list                  # List registered domain agents
aia task list                   # Query task queue and states
aia exec list                   # View execution history
aia memory list                 # View stored long-term memories
aia project list                # List registered workspace projects

# Investment Domain
aia invest decision             # Latest investment recommendation
aia invest research             # Run investment research cycle
aia invest predict              # View directional accuracy & Brier Score
```

---

## 5. Web Console & REST Endpoints

The embedded console runs on `http://127.0.0.1:7777/`:

- `GET /` — Single Page Application UI.
- `GET /api/v1/system/health` — Public health probe.
- `GET /api/v1/system/activity` — Public activity monitor snapshot.
- `GET /api/v1/agents` — Authenticated agent list.
- `GET /api/v1/tasks` — Authenticated task list.
- `GET /api/v1/executions` — Authenticated execution log.
- `GET /api/v1/investment/decisions` — Authenticated decision list.
- `GET /api/v1/investment/predictions` — Authenticated prediction records.
