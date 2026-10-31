# AIA Development: Roadmap

This roadmap tracks the development milestones of the **Autonomous Intelligence Architecture (AIA)** personal agent platform.

---

## 1. Architectural Scope & Platform Decisions

### Status of Mobile Client / Android APK: `PAUSED` ⏸️
> [!NOTE] Architecture Decision
> Development of a native **Android APK / Mobile App** was evaluated during earlier design phases, but has been **officially paused**.
>
> **Rationale:**
> 1. **Zero-Overhead Access:** AIA runs as a persistent local background daemon (`systemd --user`) on Linux / macOS. Its **Embedded Web Console** is fully responsive, touch-friendly, and mobile-optimized.
> 2. **Network Reachability:** The web console can be securely accessed from any mobile browser (Android, iOS) via local Wi-Fi, WireGuard, or Tailscale without app store packaging or Java/Kotlin runtime overhead.
> 3. **Focus on Core Autonomy:** Engineering effort is prioritized on agent governance, 3-frequency background automation, and quantitative investment calibration rather than maintaining fragmented mobile client codebases.

---

## 2. Milestone Overview

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        Development Milestones                          │
├─────────────────┬─────────────────┬──────────────────┬─────────────────┤
│ Phase 1: Core   │ Phase 2: Domain │ Phase 3: Web UI  │ Phase 4: Auto   │
│ CLI & Runtime   │ Agents & Invest │ & Background Dmn │ & Materiality   │
│   [COMPLETED]   │   [COMPLETED]   │   [COMPLETED]    │   [COMPLETED]   │
├─────────────────┼─────────────────┼──────────────────┼─────────────────┤
│ Phase 5: Multi- │ Phase 6: Memory │ Phase 7: Mobile  │ Phase 8: Multi- │
│ Asset Expansion │ Graph & Evolve  │ Native Client    │ Machine Sync    │
│    [PLANNED]    │    [PLANNED]    │     [PAUSED]     │    [FUTURE]     │
└─────────────────┴─────────────────┴──────────────────┴─────────────────┘
```

---

## 3. Milestone Details

### Phase 1: Core CLI & Storage Runtime ✅ (Completed)
- SQLite schema with WAL mode (`agents`, `tasks`, `executions`, `memories`, `projects`).
- Personal Agent coordinator and prompt decomposition.
- Cobra CLI interface (`aia agent`, `aia task`, `aia exec`, `aia memory`, `aia project`, `aia health`).
- Basic task lifecycle state transitions and execution event streams.

### Phase 2: Domain Agents & Investment Intelligence ✅ (Completed)
- Investment Agent domain modeling (`internal/investment`).
- TWSE OpenAPI connector for real-time EOD prices (TSMC 2330, Foxconn 2317, MediaTek 2454).
- MOPS material announcements integration.
- FRED macroeconomic indicators client (FEDFUNDS, CPI, Liquidity).
- Brier Score and Directional Accuracy calibration model.

### Phase 3: Embedded Web UI & Background Service ✅ (Completed)
- Single-page application console embedded in Go binary via `embed.FS`.
- Server port configured to `7777` with CLI and `AIA_PORT` binding.
- Cryptographic One-Time Token authentication redeeming to secure HTTP-only session cookie.
- `systemd --user` service script (`service.sh`) with rate-limiting and crash-loop prevention.
- Dynamic key resolution fallback (`~/.config/aia/service.env`).

### Phase 4: Background Workers & Materiality Gate ✅ (Completed)
- Concurrent `ScheduleWorker` (15s interval) and `TaskWorker` (5s polling, 3 parallel slots).
- 3-Frequency background operational separation.
- **Materiality Gate ("重大性檢驗閘道"):** Automatic volatility and disclosure filtering with `PreserveExistingDecision` to eliminate redundant LLM token expenditures.
- **AIA Activity Monitor:** Telemetry snapshot reporting across CLI (`aia monitor`), REST endpoint (`/api/v1/system/activity`), and Web Console.

### Phase 5: Multi-Asset Expansion & Portfolio Optimization 📋 (Planned)
- Expand watchlists beyond semiconductor focus to full TWSE/TPEx universe and US tech giants (NVDA, ASML).
- Dynamic asset correlation matrix and automated portfolio weight rebalancing.
- Historical backtesting harness against 5-year tick/daily datasets.

### Phase 6: Memory Graph & Agent Evolution 📋 (Planned)
- Vector embedding integration for semantic memory retrieval.
- Automated memory candidate consolidation and deduplication.
- Experience synthesis and self-improving prompt calibration.
