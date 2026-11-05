# AIA Operations: Configuration Reference

AIA uses a hierarchical configuration system prioritizing user overrides while ensuring safe, non-root defaults.

---

## 1. Configuration Precedence

Configurations are resolved according to the following precedence hierarchy (highest to lowest):

```text
1. CLI Flags (e.g. --port 7777, --host 127.0.0.1)
           ▼
2. Active Process Environment Variables ($AIA_PORT, $AIA_HOST)
           ▼
3. User Service Environment File: ~/.config/aia/service.env
           ▼
4. System-wide Environment File: /etc/aia/service.env
           ▼
5. Compiled-in Built-in Defaults
```

---

## 2. Environment Variables Reference

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| **`AIA_PORT`** | `7777` | Web console and REST API listening port. |
| **`AIA_HOST`** | `127.0.0.1` | Network binding host (loopback recommended). |
| **`AIA_DB_PATH`** | `~/.local/share/aia/aia.db` | Absolute or relative path to SQLite database. |
| **`AIA_AGENT_EXECUTOR`** | `mock` / `agy` | Agent execution harness (`agy` for Antigravity CLI, `mock` for testing). |
| **`FRED_API_KEY`** | *None* | Federal Reserve Economic Data API key for macroeconomic indicator sync. |
| **`FINMIND_TOKEN`** | *None* | FinMind token for supplemental Taiwan financial data feeds. |

---

## 3. Configuration File (`service.env`)

When running under systemd, variables are stored in `~/.config/aia/service.env`:

```bash
# ~/.config/aia/service.env
AIA_HOST=127.0.0.1
AIA_PORT=7777

# Data Feeds API Keys
FRED_API_KEY="your_fred_api_key_here"
FINMIND_TOKEN="your_finmind_token_here"
```

### Permission Hardening
Because `service.env` contains private API keys, file access is restricted to the owning user:
```bash
chmod 700 ~/.config/aia
chmod 600 ~/.config/aia/service.env
```

---

## 4. Port Configuration

Port `7777` is AIA's default port (avoiding common conflicts with port 8080).

### Overriding via CLI:
```bash
aia server start --port 8888 --host 0.0.0.0
```

### Overriding via Environment:
```bash
export AIA_PORT=8888
export AIA_HOST=0.0.0.0
aia server start
```

---

## 5. Full YAML Configuration Reference (`config.yaml`)

In addition to `service.env`, AIA supports a comprehensive configuration file (`config.yaml`) loaded from `~/.config/aia/config.yaml` (or custom path via `--config`). The schema is defined by [`internal/config/config.go`](file:///home/wade/src/github/aia/go/internal/config/config.go).

```yaml
# ==============================================================================
# AIA Master Configuration (config.yaml)
# ==============================================================================

system:
  name: "AIA"
  version: "1.0"
  timezone: "Asia/Taipei"

runtime:
  data_dir: "~/.local/share/aia"
  workspace_dir: "~/.local/share/aia/workspaces"
  max_concurrent_executions: 2      # Global worker slot limit
  max_concurrent_per_agent: 1       # Per-agent concurrency ceiling
  shutdown_timeout: 30s             # Graceful teardown timeout for active executions

database:
  driver: "sqlite"                  # WAL-mode SQLite
  path: "~/.local/share/aia/aia.db"

provider:
  default: "antigravity"            # "antigravity" (production agy CLI) or "mock" (testing)
  antigravity:
    executable: "agy"               # Executable binary in PATH or absolute path
    required_version: ""            # Optional version constraint (leave empty to accept any installed agy)
    version_policy: "minimum"
    output_format: "stream-json"
    input_format: "stream-json"
    default_timeout: 30m            # Hard timeout per agent execution
    sandbox_required: true          # Refuse to run without sandbox protection
    sandbox_mode: "auto"            # "auto" | "bwrap" | "systemd" | "strict" | "env_only"
    require_os_namespace: false     # Set true in high-security environments to fail-closed
    hidden_paths:                   # Masked with empty tmpfs / InaccessiblePaths
      - "~/.config/aia"
      - "~/.local/share/aia"
    read_only_paths: []
    allowed_env_vars: []            # Non-secret environment variables permitted past scrub
    skip_permissions: false

budget:
  max_daily_tokens: 500000          # Hard daily token ceiling across all executions (0 = unlimited)
  max_daily_cycles: 48              # Hard daily autonomy cycle ceiling (excess cycles DEFER safely)
  max_single_execution_tokens: 120000 # Hard limit on tokens consumed by a single execution
  max_daily_spend: 100000           # Budget ceiling in local micro-units / tracking

orchestrator:
  max_plan_retries: 1               # Retries on plan decomposition failure
  max_task_retries: 2               # Retries on individual task execution failure
  max_subtasks_per_plan: 20         # Maximum task DAG size allowed per plan
  max_dependency_depth: 8           # Maximum depth of task dependency chain

memory:
  max_context_items: 20             # Maximum working memory items injected into prompt
  max_context_bytes: 24000          # Maximum working memory byte budget in prompt
  auto_create_candidates: true      # Automatically store high-value execution insights as candidates
  auto_approve_low_risk: false      # Require governance approval even for low-risk candidates

logging:
  level: "info"                     # "debug" | "info" | "warn" | "error"
  format: "json"                    # "json" | "text"
  retain_days: 30                   # Log retention period in days (log maintenance policy)

data_sources:
  fred_api_key: ""                  # Alternatively set via FRED_API_KEY environment variable
  finmind_token: ""                 # Alternatively set via FINMIND_TOKEN environment variable
  alpha_vantage_key: ""             # AlphaVantage API key for global equity data
  telegram_bot_token: ""            # Telegram bot token for alert notifications
  telegram_chat_id: ""              # Telegram chat ID for operator alerts

domains:
  writing:
    projects:
      - id: "yuan"
        name: "元朝系列小說"
        path: "~/src/github/novel/yuan"
  coding:
    projects:
      - id: "aia"
        name: "AIA Core Project"
        path: "~/src/github/aia"

policy:
  allowed_paths:                    # File-write whitelist (supports ~/ expansion)
    - "/tmp"
    - "/var/tmp"
```

> **Secret Scrubbing Guarantee**: Regardless of OS namespace availability, [`ScrubEnvironment`](file:///home/wade/src/github/aia/go/internal/provider/antigravity/sandbox.go) strips `AIA_*`, `FRED_API_KEY`, `FINMIND_TOKEN`, and all `*_KEY` / `*_SECRET` / `*_TOKEN` / `*_PASSWORD` variables from child `agy` processes so `service.env` credentials never leak into agent shell sessions.
