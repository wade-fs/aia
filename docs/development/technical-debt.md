# AIA Development: Technical Debt & Architectural Trade-offs

This document catalogues current architectural trade-offs, known technical debts, and recommended future refactorings for AIA.

---

## 1. Architectural Trade-offs

### SQLite Single-Writer Lock vs. Embedded Simplicity
- **Decision:** SQLite with WAL mode (`PRAGMA journal_mode=WAL`) was chosen for zero-dependency, local-first portability.
- **Trade-off:** While WAL allows concurrent readers alongside a writer, SQLite permits only one writer at any given moment. Under high worker concurrency, database write contention could result in `busy` timeouts if transactions are held open.
- **Mitigation:** TaskWorker concurrency is throttled to 3 parallel workers, and write transactions in `storage/` are kept short and granular.

### Embedded Polling vs. Full Server-Sent Events (SSE)
- **Decision:** The Web Console polls `/api/v1/system/activity` periodically, while execution log streaming uses SSE (`/api/v1/events`).
- **Trade-off:** Polling introduces a slight telemetry lag (3–5 seconds) and unnecessary local requests when the dashboard is open.
- **Future Direction:** Unify all dashboard metrics into a single real-time SSE telemetry channel.

### Fixed Semiconductor Universe vs. Dynamic Watchlists
- **Decision:** Initial data collectors focus on core semiconductor and technology leaders (TSMC 2330, Foxconn 2317, MediaTek 2454) to establish a deterministic testing baseline.
- **Trade-off:** Users cannot yet define dynamic multi-sector watchlists via the Web UI without updating the collector configurations.
- **Future Direction:** Introduce a dynamic `watchlist` database table with user-configurable tickers and custom collection schedules.

---

## 2. Technical Debt Items

### Debt Item 1: Test Suite Mock Executor vs. Headless agy E2E Harness
- **Severity:** Medium
- **Description:** Unit and integration tests rely on `MockExecutor` to simulate Antigravity CLI execution responses. While fast and deterministic, end-to-end integration with live `agy` requires an external Google account and CLI installation.
- **Remediation:** Build an automated end-to-end smoke test suite that executes `agy` in headless mode when `AIA_INTEGRATION_TESTS=1` is provided.

### Debt Item 2: Dynamic Prediction Horizon Resolution
- **Severity:** Low
- **Description:** Predictions currently use fixed evaluation windows (e.g., 30, 60, or 90 days). Event-driven horizons (e.g., "until next earnings call" or "until central bank rate decision") are not dynamically parsed from the thesis text.
- **Remediation:** Enhance `investment.PredictionTarget` to support event-driven trigger dates extracted by the LLM during decision formulation.

### Debt Item 3: Frontend Build Pipeline
- **Severity:** Low
- **Description:** The embedded web frontend is written in vanilla HTML/CSS/JavaScript with zero build step, bundled directly into the Go binary. While eliminating `npm`/`webpack` dependencies, complex state management is handled manually in DOM operations.
- **Remediation:** Retain the zero-node build ethos, but modularize JavaScript components into clean ES modules if UI complexity expands.
