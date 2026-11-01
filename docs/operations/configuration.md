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

## 5. Budget Gate & Subprocess Sandbox Configuration (YAML)

In `config.yaml` (loaded by [`internal/config/config.go`](file:///home/wade/src/github/aia/go/internal/config/config.go)), AIA enforces pre-flight resource budgets and subprocess OS isolation:

```yaml
budget:
  max_daily_tokens: 500000      # Hard daily token ceiling across executions (0 = unlimited)
  max_tokens_per_task: 100000   # Per-task token ceiling
  max_daily_cycles: 24          # Hard daily autonomy cycle ceiling (excess cycles DEFER safely)

sandbox:
  enable_os_namespace: true     # Wrap agy in bwrap / systemd-run read-only mount namespace when available
  require_os_namespace: false   # Set true in strict production to fail-closed if mount namespaces are unavailable
  hidden_paths:                 # Masked with empty tmpfs / InaccessiblePaths inside subprocesses
    - ~/.config/aia
    - ~/.local/share/aia
  allowed_env_vars: []          # Extra non-secret env vars permitted past ScrubEnvironment
```

> **Secret Scrubbing Guarantee**: Regardless of OS namespace availability, [`ScrubEnvironment`](file:///home/wade/src/github/aia/go/internal/provider/antigravity/sandbox.go) strips `AIA_*`, `FRED_API_KEY`, `FINMIND_TOKEN`, and all `*_KEY` / `*_SECRET` / `*_TOKEN` / `*_PASSWORD` variables from child `agy` processes so `service.env` credentials never leak into agent shell sessions.
