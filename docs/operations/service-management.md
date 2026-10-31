# AIA Operations: Service Management

AIA is designed to run as a resident background daemon using **systemd user services** (`systemd --user`). This allows the daemon to start on user login, run without root permissions, and survive terminal closures.

---

## 1. Quick Reference: Makefile Targets

Service operations can be executed from either the repository root or the `go/` subdirectory:

| Command | Action |
| :--- | :--- |
| `make install_service` | Builds, configures, and starts `aia.service`. |
| `make uninstall_service`| Stops, disables, and removes the systemd unit. |
| `make status_service` | Displays systemd service state, PID, memory, and URLs. |
| `make restart_service` | Reloads configurations and restarts the daemon. |
| `make logs_service` | Tails live systemd journal logs (`journalctl --user -u aia -f`). |
| `make token` | Fetches the active one-time login token from logs. |

---

## 2. Service Management Script (`go/scripts/service.sh`)

The underlying management script is located at [`go/scripts/service.sh`](file:///home/wade/src/github/aia/go/scripts/service.sh). It automatically resolves directory paths whether invoked from the repository root or inside `go/`.

### Manual CLI Invocations:
```bash
./go/scripts/service.sh install
./go/scripts/service.sh status
./go/scripts/service.sh restart
./go/scripts/service.sh logs
./go/scripts/service.sh token
./go/scripts/service.sh uninstall
```

---

## 3. Systemd Unit Configuration

The generated service file is stored at `~/.config/systemd/user/aia.service`:

```ini
[Unit]
Description=AIA - Personal Autonomous Intelligence Architecture Daemon
Documentation=https://github.com/wade/aia
After=network.target
StartLimitIntervalSec=30s
StartLimitBurst=5

[Service]
Type=simple
EnvironmentFile=-%h/.config/aia/service.env
Environment=AIA_HOST=127.0.0.1
Environment=AIA_PORT=7777
ExecStart=%h/.local/bin/aia server start
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
```

---

## 4. Crash-Loop Protection & Rate Limiting

To safeguard your workstation against CPU exhaustion or log filling in the event of configuration errors:

1. **`Restart=on-failure`:** If the daemon is explicitly stopped or exits with code `0`, it is not restarted.
2. **`RestartSec=5s`:** Enforces a 5-second backoff before any restart attempt.
3. **`StartLimitIntervalSec=30s` & `StartLimitBurst=5`:** If the daemon crashes 5 times within 30 seconds (e.g. port already bound by another service), systemd halts further restarts and marks the service failed.

---

## 5. Web Console Access & Token Authentication

1. Fetch your login token:
   ```bash
   make token
   ```
2. Open your web browser to:
   ```text
   http://127.0.0.1:7777/?token=<YOUR_TOKEN>
   ```
3. The token is consumed once to issue a secure 24-hour HTTP-only session cookie. Future visits to `http://127.0.0.1:7777/` remain logged in until the session expires or is cleared.
