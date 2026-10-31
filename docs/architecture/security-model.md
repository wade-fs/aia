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

## 4. Execution Workspace Isolation

When executing tasks via domain agents or Antigravity (`agy`):
- **Project Sandbox:** Agents are confined to the designated `project_path`.
- **Restricted Tooling:** Antigravity harnesses use fine-grained permission flags, disallowing arbitrary system modification unless explicitly approved.
- **Task Approval Gates:** Tasks marked with elevated risk transition to `WAITING_APPROVAL`, requiring explicit user confirmation before any state-mutating execution proceeds.

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
