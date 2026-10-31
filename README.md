# AIA — Autonomous Intelligence Architecture

**Personal AI Agent Platform**

AIA (Autonomous Intelligence Architecture) is a personal, local-first AI agent platform designed to coordinate specialized domain agents, manage persistent tasks, maintain long-term memory, and integrate AI capabilities into everyday productivity workflows.

Rather than treating an AI agent as an ephemeral chatbot, AIA provides a robust runtime environment in which multiple specialized agents operate under a shared architecture, maintain auditable state machines, execute scheduled background workflows, and accumulate validated knowledge over time.

The system is implemented in **Go**, with [Antigravity CLI (`agy`)](https://antigravity.google/) serving as the underlying AI execution engine.

---

## 1. Core Principles

- **Personal:** Owned, configured, and governed by a single user. All data remains in local SQLite storage.
- **Persistent:** Tasks, execution traces, long-term memory, and decisions survive restarts and individual conversations.
- **Modular:** Clear domain division across specialized agents (Investment, Coding, Writing, System Workers).
- **Governed:** Bounded by explicit policies, token-saving **Materiality Gates**, and human approval checkpoints.

> [!NOTE] Mobile / Android APK Status: `PAUSED` ⏸️
> Development of a native Android APK was evaluated during early planning and is currently **paused**.
> AIA operates as a resident background daemon on Linux/macOS with a responsive, touch-friendly **Embedded Web Console** (accessible over local Wi-Fi or Tailscale), eliminating mobile app packaging and duplicate client maintenance. See [docs/development/roadmap.md](file:///home/wade/src/github/aia/docs/development/roadmap.md) for details.

---

## 2. Directory Structure

The repository is organized according to the following layout:

```text
aia/
├── README.md                           # Master Project Overview & Quick Start
│
├── docs/                               # Comprehensive Documentation
│   ├── architecture/                   # Architectural Designs & System Models
│   │   ├── overview.md                 # System positioning, vision, and core pillars
│   │   ├── runtime.md                  # Goroutine model, workers, and 3-frequency automation
│   │   ├── agent-architecture.md       # Agent hierarchy, Materiality Gate, and agy harness
│   │   └── security-model.md           # Single-user boundary, token auth, and secrets
│   │
│   ├── specifications/                 # Formal AIA Specifications (AIA-01 to AIA-07)
│   │   ├── AIA-01-product-requirements.md  # Product Requirements Document
│   │   ├── AIA-02-system-architecture.md   # System Architecture & Module Design
│   │   ├── AIA-03-go-runtime.md            # Go Runtime Implementation Specification
│   │   ├── AIA-04-test-plan.md             # Testing Plan & Acceptance Criteria
│   │   ├── AIA-05-investment-agent.md      # Investment Agent Domain Specification
│   │   ├── AIA-06-web-interface.md         # Embedded Web UI & API Specification
│   │   └── AIA-07-user-manual.md           # User Manual & CLI/Web Operations Guide
│   │
│   ├── development/                    # Project Evolution & Audit
│   │   ├── roadmap.md                  # Milestones, future phases, and paused mobile scope
│   │   ├── implementation-status.md    # Package audit, schemas, CLI, and endpoints
│   │   └── technical-debt.md           # Trade-offs, known debt, and future improvements
│   │
│   └── operations/                     # Deployment & Maintenance
│       ├── installation.md             # Compilation, binaries, and environment setup
│       ├── service-management.md       # systemd user daemon, scripts, and logs
│       └── configuration.md            # Configuration hierarchy and environment variables
│
└── go/                                 # Implementation Source Code
    ├── cmd/                            # CLI Commands (aia)
    ├── configs/                        # Configuration defaults & agent profiles
    ├── internal/                       # Core packages (storage, worker, investment, host...)
    ├── migrations/                     # SQLite schema migrations
    ├── prompts/                        # System prompts for agents
    ├── scripts/                        # Operational scripts (service.sh)
    ├── tests/                          # Unit and integration test suites
    ├── web/                            # Embedded Web UI single-page application
    └── Makefile                        # Go build and test automation
```

---

## 3. Key Capabilities & Innovations

### Three-Frequency Background Automation
AIA does not run LLMs in an endless loop. Background operations are partitioned into three distinct frequencies:
1. **Low Frequency (Data Collection):** Automated synchronization of TWSE OpenAPI quotes, MOPS regulatory filings, and FRED macroeconomic indicators.
2. **Medium Frequency (Research & Recommendation):** Governed by the **Materiality Gate ("重大性檢驗閘道")**. If price volatility $< 3\%$ and no material disclosures were filed, AIA preserves existing recommendations (`PreserveExistingDecision`), saving 100% of LLM reasoning tokens.
3. **Low Frequency (Prediction Audit & Calibration):** Audits matured price targets, calculates directional accuracy, and computes probabilistic **Brier Scores** to track judgment quality over time.

### Real-Time AIA Activity Monitor
Track system telemetry at any time via CLI or Web Console:

```bash
$ aia monitor
ITEM                       STATUS / VALUE
Runtime                    RUNNING
Schedule Worker            RUNNING
Task Worker                IDLE
Last Data Sync             2026-10-31 16:00:00
Last Investment Research   2026-10-31 17:00:00
Pending Tasks              0
Next Scheduled Run         2026-11-01 09:00:00
```

---

## 4. Quick Start

### Build & Run Locally
```bash
# Clone the repository
git clone https://github.com/wade/aia.git
cd aia

# Build the binary
make build

# Run health check
./go/bin/aia health
```

### Install as Background Daemon (`systemd --user`)
```bash
# Install and start the background service
make install_service

# Inspect service status
make status_service

# Retrieve your one-time login token
make token
```

Visit the Embedded Web Console at:
```text
http://127.0.0.1:7777/?token=<YOUR_TOKEN>
```

---

## 5. Documentation Navigation

- **Architecture:** [Overview](file:///home/wade/src/github/aia/docs/architecture/overview.md) | [Runtime](file:///home/wade/src/github/aia/docs/architecture/runtime.md) | [Agent Hierarchy](file:///home/wade/src/github/aia/docs/architecture/agent-architecture.md) | [Security Model](file:///home/wade/src/github/aia/docs/architecture/security-model.md)
- **Specifications:** [AIA-01 Requirements](file:///home/wade/src/github/aia/docs/specifications/AIA-01-product-requirements.md) | [AIA-02 Architecture](file:///home/wade/src/github/aia/docs/specifications/AIA-02-system-architecture.md) | [AIA-03 Go Runtime](file:///home/wade/src/github/aia/docs/specifications/AIA-03-go-runtime.md) | [AIA-04 Test Plan](file:///home/wade/src/github/aia/docs/specifications/AIA-04-test-plan.md) | [AIA-05 Investment Agent](file:///home/wade/src/github/aia/docs/specifications/AIA-05-investment-agent.md) | [AIA-06 Web UI](file:///home/wade/src/github/aia/docs/specifications/AIA-06-web-interface.md) | [AIA-07 User Manual](file:///home/wade/src/github/aia/docs/specifications/AIA-07-user-manual.md)
- **Development:** [Roadmap & Android Scope](file:///home/wade/src/github/aia/docs/development/roadmap.md) | [Implementation Status](file:///home/wade/src/github/aia/docs/development/implementation-status.md) | [Technical Debt](file:///home/wade/src/github/aia/docs/development/technical-debt.md)
- **Operations:** [Installation](file:///home/wade/src/github/aia/docs/operations/installation.md) | [Service Management](file:///home/wade/src/github/aia/docs/operations/service-management.md) | [Configuration](file:///home/wade/src/github/aia/docs/operations/configuration.md)
