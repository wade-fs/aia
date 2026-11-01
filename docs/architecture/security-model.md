# AIA Architecture: Security Model

AIA is designed as a **single-user, local-first platform**. Because it executes shell commands, analyzes sensitive financial portfolios, and interacts with local codebases, strict defense-in-depth boundaries are enforced across network, filesystem, and runtime layers.

---

## 1. Single-User Isolation Principle

- **No Multi-Tenant Shared State:** AIA assumes a single operator per installation. All database files and runtime sockets are bound to the operating system user account (`$UID`).
- **Loopback Default:** By default, all network listeners (Web Console and REST API) bind strictly to loopback (`127.0.0.1:7777`). External access must be explicitly authorized or routed through secure overlay networks (e.g., WireGuard or Tailscale).

---

## 2. Authentication & Session Architecture

AIA eliminates default static passwords in favor of an **ephemeral one-time token** model:

```text
[ Server Starts via systemd ]
              │
              ▼
  Generates 32-byte Cryptographic Token
  (Logged once to system journal, never exposed via unauthenticated API)
              │
              ▼
   User fetches token: `make token` or `aia server token`
              │
              ▼
┌────────────────────────────────────────────────────────┐
│ User Visits: http://127.0.0.1:7777/?token=<TOKEN>      │
│                                                        │
│ 1. Host verifies token against active in-memory secret │
│ 2. Token is INSTANTLY INVALIDATED (Single-use)         │
│ 3. Issues cryptographically signed HTTP-only Cookie:   │
│    • SameSite=Strict                                   │
│    • Path=/                                            │
│    • HttpOnly=true                                     │
│    • MaxAge=24 hours                                   │
└────────────────────────────────────────────────────────┘
```

### Whitelisted Endpoints
To allow local CLI monitors, status checks, and system health probes to function without cookie credentials, the following endpoints are explicitly exempt from session authentication:
- `GET /api/v1/system/health`
- `GET /api/v1/system/activity`

All other endpoints (Tasks, Executions, Decisions, Agents, Projects, Settings) reject unauthenticated requests with `401 Unauthorized`.

---

## 3. Secret Management & Credential Resolution

No API keys, credentials, or tokens are ever hardcoded in the codebase, version control, or scripts. AIA resolves secrets dynamically at runtime through a strict fallback hierarchy:

```text
   Priority 1: Active Process Environment Variables ($FRED_API_KEY, $FINMIND_TOKEN)
                                 │ (if missing)
                                 ▼
   Priority 2: User Config Directory: ~/.config/aia/service.env (chmod 600)
                                 │ (if missing)
                                 ▼
   Priority 3: System-wide Directory: /etc/aia/service.env (chmod 600)
                                 │ (if missing)
                                 ▼
   Interactive Prompt during `make install_service` / graceful degraded execution
```

### File Permission Hardening
The installation script enforces Unix permissions:
```bash
chmod 700 ~/.config/aia
chmod 600 ~/.config/aia/service.env
```
This guarantees that other local non-root users cannot read stored API tokens.

---

## 4. Execution Workspace & Subprocess OS Isolation

### 4.1 Threat Model: Single-UID Subprocess Escalation
Because `aia.service` runs under `systemd --user` and spawns external CLI harnesses (`agy`) that can execute shell commands, relying solely on Go-level `PolicyEvaluator` checks or SQLite triggers/hash-chains is **insufficient**:
- Under an unisolated single `$UID`, a child shell process could inherit `service.env` variables from parent process memory, directly open `~/.local/share/aia/aia.db` via `sqlite3`, `DROP TRIGGER trg_event_history_no_update`, or modify AIA governance code on disk.
- Hash chains (`event_history`) provide **tamper detection**, NOT **tamper prevention**.

### 4.2 Three-Layer OS Isolation Architecture
To make Level B (Core Runtime) and Level C (Governance/Policy/Audit) write protection genuinely enforceable against shell-capable subprocesses, AIA enforces defense-in-depth at the OS boundary (`internal/provider/antigravity/sandbox.go`):

1. **Layer 1 — Mandatory Environment Whitelist Scrubbing (`ScrubEnvironment`)**:
   - Every child `exec.Cmd` explicitly constructs a clean `cmd.Env` whitelist (`PATH`, `HOME`, `USER`, `LOGNAME`, `LANG`, `LC_ALL`, `TERM`, `TMPDIR`).
   - All secrets loaded from `~/.config/aia/service.env` (`*_KEY`, `*_TOKEN`, `*_SECRET`, `*_PASSWORD`, `DATABASE_*`, `AIA_*`) are stripped unconditionally before `cmd.Start()`.

2. **Layer 2 — Linux Mount Namespace Isolation (`bwrap` / `systemd-run --user`)**:
   - Subprocesses execute inside an unprivileged mount namespace (`SandboxMode: auto | bwrap | systemd | strict`) probed against real kernel execution capability (`ProbeBwrap` / `ProbeSystemdUser`, never trusting `LookPath` alone):
     - **Read-Only Root & AIA Core (`Level B/C`)**: `/` and the AIA source/binary tree (especially `internal/policy`, `internal/service/governance`, `internal/service/approval`) are mounted **read-only (`--ro-bind` / `ReadOnlyPaths=/`)**.
     - **Inaccessible / Masked State & Secrets**: `~/.config/aia` (`service.env`) and `~/.local/share/aia` (SQLite `.db`, WAL/SHM files, OTT session secrets) are masked via `--tmpfs` / `InaccessiblePaths=` so child processes cannot read or mutate the SQLite database or drop triggers.
     - **Confined Read-Write Workspace**: Only the task's dedicated `WorkDir` (`workspaces/<execution_id>`) and ephemeral `/tmp` are mounted read-write.
   - **Fail-Closed Enforcement (`RequireOSNamespace: true`)**: High-risk tasks and Phase 6 code self-modification profiles fail closed (`ErrOSSandboxUnavailable`) if kernel namespace isolation is unavailable on the host.

3. **Layer 3 — Dual-UID Production Hardening (Required for Phase 6 Autonomous Code Evolution)**:
   - For unattended Phase 6 deployments, run the control plane (`aia` daemon owning SQLite & `service.env` with `0600` permissions) as user `aia-daemon`, and invoke external `agy` workers via a constrained `sudo -u aia-worker` wrapper so the OS DAC kernel permission model enforces UID separation independent of namespaces.
   - Task Approval Gates (`WAITING_APPROVAL`) remain enforced at Step 7 (`Governance Check`) prior to spawning any subprocess.

---

## 5. Daemon Resilience & Crash-Loop Prevention

When managed under systemd user services, AIA prevents denial-of-service crash loops through strict rate limiting:

```ini
[Service]
Restart=on-failure
RestartSec=5s

[Unit]
StartLimitIntervalSec=30s
StartLimitBurst=5
```

- **`Restart=on-failure`:** If the daemon exits cleanly (`exit 0`), it will not restart unnecessarily.
- **Start Limit:** If the daemon fails more than 5 times within a 30-second window (e.g., due to port conflict or configuration error), systemd halts restarts, preventing high CPU loops and disk log exhaustion.
