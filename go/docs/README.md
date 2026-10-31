# AIA — Autonomous Intelligence Architecture

AIA is a Personal AI Agent Platform implemented in Go.

It coordinates domain agents, manages tasks and executions, persists long-term memories, organizes workspace projects, and integrates with Antigravity CLI.

AIA is designed as a single-user, local-first AI Agent Runtime.

---

## 1. Features

Current CLI capabilities:

- Personal Agent conversation and task dispatch.
- Domain Agent management.
- Task creation, execution, cancellation, and recovery.
- Execution inspection and event stream.
- Long-term memory and memory candidates.
- Workspace project management.
- System health checks.
- Embedded Web UI and REST/SSE API.
- Antigravity CLI integration.

Built-in Agents currently registered:

- `personal-agent`
- `coding-agent`
- `writing-agent`
- `investment-agent`
- `video-agent`

Actual registration and enablement status can be checked with:

```bash
aia agent list
```

---

## 2. Requirements

- Linux environment.
- Go runtime/build environment when building from source.
- Antigravity CLI (`agy`) 1.4.12.
- SQLite support.
- Valid Antigravity CLI authentication when AI execution is required.

AIA does not manage or copy Antigravity OAuth credentials.

Users perform authentication through the Antigravity CLI.

---

## 3. Build

From the Go project directory:

```bash
go build -o bin/aia ./cmd/aia
```

Verify the binary:

```bash
./bin/aia version
```

Check system dependencies:

```bash
./bin/aia doctor
```

---

## 4. CLI Help

Show all available commands:

```bash
./bin/aia --help
```

Show help for a command:

```bash
./bin/aia task --help
./bin/aia agent --help
./bin/aia memory --help
```

The current CLI provides these top-level commands:

| Command | Description |
|---|---|
| `agent` | Manage and inspect domain agents |
| `chat` | Chat with the Personal Agent or dispatch tasks |
| `completion` | Generate shell completion scripts |
| `doctor` | Check dependencies and environment |
| `execution` | Inspect executions and events |
| `memory` | Manage long-term memories and candidates |
| `project` | Manage workspace projects |
| `server` | Manage the AIA server process |
| `status` | Show system summary and running tasks |
| `task` | Manage and execute tasks |
| `version` | Print version information |
| `web` | Start the embedded Web UI and API server |

---

## 5. Configuration

Default configuration locations:

```text
~/.config/aia/config.yaml
./configs/aia.yaml
```

Specify a configuration file:

```bash
./bin/aia --config /path/to/config.yaml doctor
```

The global configuration flag is:

```text
-c, --config
```

---

## 6. System Health

Run the environment check:

```bash
./bin/aia doctor
```

Show system status:

```bash
./bin/aia status
```

Show version:

```bash
./bin/aia version
```

`doctor` checks system dependencies, agy, SQLite, and environment health.

`status` provides a system summary and running Task information.

---

## 7. Server

Start the AIA Server Process:

```bash
./bin/aia server start
```

Show available Server commands:

```bash
./bin/aia server --help
```

The currently exposed Server subcommand is:

```text
start
```

Do not assume `server stop`, `server shutdown`, or `server restart` are available unless they are added to the CLI.

---

## 8. Embedded Web UI

Start the Embedded Web UI and REST/SSE API server:

```bash
./bin/aia web
```

Default settings:

| Parameter | Default |
|---|---|
| Host | `127.0.0.1` |
| Port | `8080` |

Open:

```text
http://127.0.0.1:8080
```

Custom host and port:

```bash
./bin/aia web --host 127.0.0.1 --port 8080
```

The Web server generates a cryptographically random one-time login token at startup and prints it to the terminal.

Use the token to log in through the browser.

After successful login, the token is exchanged for a server-side session and becomes invalid.

**Security:** AIA v1 is designed for local single-user operation. Do not expose the Web server to LAN or the public Internet.

---

## 9. Chat with the Personal Agent

Start an interactive conversation:

```bash
./bin/aia chat
```

Send a message directly:

```bash
./bin/aia chat "Help me review this software architecture."
```

Use structured JSON output:

```bash
./bin/aia chat --json "Summarize the current project status."
```

Associate a conversation with a Project:

```bash
./bin/aia chat --project PROJECT_ID "Review this project's architecture."
```

Available flags:

| Flag | Description |
|---|---|
| `--json` | Output structured JSON |
| `--project` | Specify Project ID |
| `-c`, `--config` | Specify configuration file |

The Personal Agent can respond directly or dispatch work to a suitable Domain Agent when appropriate.

Not every conversation necessarily creates a Task.

---

## 10. Tasks

Show Task commands:

```bash
./bin/aia task --help
```

Available commands:

| Command | Description |
|---|---|
| `task create` | Create a Task |
| `task list` | List Tasks |
| `task show` | Show Task details |
| `task run` | Execute a READY Task using the Scheduler |
| `task resume` | Resume a failed or blocked Task |
| `task cancel` | Cancel a Task |

Inspect command arguments before use:

```bash
./bin/aia task create --help
./bin/aia task run --help
./bin/aia task show --help
./bin/aia task resume --help
./bin/aia task cancel --help
```

Typical operation sequence:

1. Create a Task.
2. Inspect the Task.
3. Execute a READY Task.
4. Inspect the resulting Execution.
5. Resume the Task if recovery is appropriate.

Task states:

- `CREATED`
- `READY`
- `RUNNING`
- `WAITING_APPROVAL`
- `WAITING_FOR_AUTH`
- `RETRY_WAIT`
- `COMPLETED`
- `FAILED`
- `CANCELLED`

There is no `QUEUED` Task state.

A successful Execution does not automatically mean the Task is COMPLETED. Required result validation must be completed first.

---

## 11. Agents

List registered Agents:

```bash
./bin/aia agent list
```

Show Agent details:

```bash
./bin/aia agent show --help
```

Enable or disable an Agent:

```bash
./bin/aia agent enable --help
./bin/aia agent disable --help
```

Available Agent operations:

- List.
- Show.
- Enable.
- Disable.

Actual Agent availability depends on its registration and enablement status.

---

## 12. Executions

List Execution commands:

```bash
./bin/aia execution --help
```

Show Execution details:

```bash
./bin/aia execution show --help
```

Show Execution events:

```bash
./bin/aia execution events --help
```

Execution records represent individual attempts to perform a Task.

They are useful for:

- Inspecting execution results.
- Diagnosing failures.
- Reviewing event history.
- Understanding retries and recovery.

---

## 13. Memory

Show Memory commands:

```bash
./bin/aia memory --help
```

List active memories:

```bash
./bin/aia memory list
```

Search memories:

```bash
./bin/aia memory search --help
```

Manage Memory Candidates:

```bash
./bin/aia memory candidate --help
```

Memory Candidates are proposed long-term memories that have not yet become active memories.

Review candidates before treating their contents as established long-term information.

---

## 14. Investment Core & Cost Calculation

AIA enforces a strict architectural boundary between AI analysis and Go Runtime financial execution:
- **agy / LLM**: Responsible for market research, news synthesis, reasoning, and proposing trade ideas (BUY / SELL / HOLD).
- **Go Runtime**: Validates symbols, calculates exact share sizing (1,000 shares lot or odd lot), checks available cash and portfolio capital limits, computes transaction costs and taxes, and produces formal Trade Proposals. LLM output is never allowed to directly become an executable trade.

### Transaction Cost Parameters & Configurability

Transaction costs and taxes are calculated natively in Go (`internal/domain/investment/calculator.go`):

1. **Default Parameters (Taiwan Equities)**:
   - **Broker Fee Rate (`FeeRate`)**: `0.1425%` (0.001425) for both BUY and SELL.
   - **Broker Discount (`FeeDiscountRate`)**: Default `1.0` (no discount).
   - **Minimum Broker Fee (`MinFee`)**: Default `NT$20`.
   - **Transaction Tax (`TaxRate`)**: `0.3%` (0.003) on SELL; 0 on BUY.

2. **Configurability**:
   These rates are not static or hardcoded business limits. The Go Runtime utilizes a `CostConfig` structure that can be customized per portfolio profile (via `configuration_json`) or runtime configuration:
   - **Broker Discount Rates**: Supports customized broker electronic discounts (e.g. 60% of base rate `0.6`, 28% `0.28`).
   - **Special Minimum Fees**: Supports promotional minimum fees (e.g. NT$1).
   - **Asset/Regulation Specific Tax Rates**: Supports ETF statutory tax rate (`0.1%`), day-trading tax concessions (`0.15%`), or future statutory adjustments.
   - **Multi-market Extensibility**: Prepares the engine for US equities or foreign assets without breaking documentation or calculation rules.

---

## 15. Implementation & Version Maturity Baseline

AIA is currently at **v1.0 (Core & Web Baseline)**. The system implementation baseline is categorized as follows:

| Component | Scope | Implementation Status | Notes |
|---|---|---|---|
| **P1 Web Infrastructure** | Web Host, Session, API, SSE, Security | **IMPLEMENTED** | Localhost-only (127.0.0.1), one-time bootstrap token, HttpOnly session cookie, CSRF/Origin protection |
| **P2 Global Web Console** | Dashboard, Tasks, Agents, Memory, Pending Review, Audit | **IMPLEMENTED** | Native HTML/CSS/JS frontend embedded via `go:embed`, unified approval overview for Tasks and Memory Candidates |
| **P3 Investment Core** | Portfolio Profile/Snapshot, Sizing, Limits, Cost Calculation, Proposals | **IMPLEMENTED** | Go calculation boundary, dynamic `CostConfig`, proposal rejection logging |
| **P6 Agent Web UI** | Dedicated Agent UI & Namespaced Mux | **IMPLEMENTED** | Agent routing isolation (`/api/v1/agents/{id}/`), Investment portfolio and proposal UI |
| **P4 Data Ingestion** | TWSE, TPEx, MOPS, Financial News Scraping | **PLANNED** | Scheduled for next implementation phase |
| **P5 Prediction & Eval** | Forecast Calibration, Backtesting, Self-Improvement | **PLANNED** | Scheduled for next implementation phase |
| **SCHED Automation** | Cron Scheduling & Misfire Policies | **PLANNED** | Scheduled for next implementation phase (Task dependency scheduling currently active) |
| **CLI-INV Top-Level CLI** | `aia investment` CLI suite | **PLANNED** | Scheduled for next implementation phase (Currently fully operable via Go service and Web UI) |
