# AIA Operations: Installation Guide

This guide details how to build, install, and verify AIA on Linux or macOS systems.

---

## 1. System Requirements & Prerequisites

- **Operating System:** Linux (systemd recommended for service mode) or macOS (12+).
- **Go:** Go 1.22 or newer.
- **C Compiler:** `gcc` or `clang` (required for SQLite CGO compilation).
- **Make:** GNU Make.
- **Antigravity CLI (`agy`):** Optional for local development; required for live AI execution.

---

## 2. Building from Source

Clone the repository and build the binary:

```bash
# Clone repository
git clone https://github.com/wade/aia.git
cd aia

# Build the binary using the Makefile
make build
```

This compiles the executable to `go/bin/aia`.

To run test suites:
```bash
# Run unit and integration tests
make test

# Run tests with race detection
make test-race
```

---

## 3. Local Installation

To install the `aia` binary into your user's executable path (`~/.local/bin`):

```bash
mkdir -p ~/.local/bin
cp go/bin/aia ~/.local/bin/
```

Ensure `~/.local/bin` is in your shell's `PATH`:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

Verify installation:
```bash
aia --help
aia health
```

---

## 4. Background Service Installation

For continuous background execution (workers, scheduled data sync, embedded web console), install AIA as a systemd user daemon:

```bash
# From repository root
make install_service
```

The installer will:
1. Compile and install `aia` to `~/.local/bin/aia`.
2. Check for required environment keys (`FRED_API_KEY`, `FINMIND_TOKEN`). If missing, you will be prompted interactively.
3. Write secure configuration to `~/.config/aia/service.env` with `chmod 600` permissions.
4. Register and enable `~/.config/systemd/user/aia.service`.
5. Start the service on `http://127.0.0.1:7777/` and display the one-time login token.

---

## 5. Verification

Inspect the running system status:

```bash
# View service operational status
make status_service

# Inspect real-time Activity Monitor
aia monitor
```
