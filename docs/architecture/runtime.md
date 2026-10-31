# AIA Architecture: Runtime & Background Workers

The **AIA Runtime** is a concurrent, local-first execution environment implemented in Go. It operates continuously as a resident system daemon (`systemd --user`), executing scheduled workflows, background polling, task execution queues, and telemetry reporting.

---

## 1. Concurrency Model & Lifecycle

The runtime is managed by [`internal/worker/Manager`](file:///home/wade/src/github/aia/go/internal/worker/manager.go) within the primary daemon process:

```text
                  ┌──────────────────────────────┐
                  │      AIA Runtime Daemon      │
                  │   (PID via systemd user)     │
                  └──────────────┬───────────────┘
                                 │
         ┌───────────────────────┼────────────────────────┐
         │ Start()               │                        │
         ▼                       ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌────────────────────┐
│ ScheduleWorker  │    │    TaskWorker    │    │ Embedded Web Server│
│ Ticker: 15s     │    │ Ticker: 5s       │    │ Port: 7777         │
│ Dispatch READY  │    │ Sem: 3 Parallel  │    │ HTTP/REST & SSE    │
└─────────────────┘    └──────────────────┘    └────────────────────┘
         │                       │                        │
         └───────────────────────┼────────────────────────┘
                                 │ Stop() via context.Context
                                 ▼
                     Graceful Shutdown (10s)
```

- **Graceful Lifecycle:** Both workers consume `context.Context`. When `SIGINT` or `SIGTERM` is captured, running executions are given a graceful drain window before the process terminates cleanly.
- **Worker Separation:** Dispatch logic (`ScheduleWorker`) is strictly decoupled from task execution logic (`TaskWorker`), preventing backlog queuing delays from blocking schedule evaluations.

---

## 2. Three-Frequency Automation Architecture

Rather than invoking resource-heavy AI reasoning in an endless loop, AIA partitions background operations into three distinct operating frequencies:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   Three Operating Frequencies                          │
├───────────────────┬──────────────────────────────┬─────────────────────┤
│ 1. Low Frequency  │ 2. Medium Frequency          │ 3. Low Frequency    │
│ Data Collection   │ Research & Recommendation    │ Prediction Audit    │
├───────────────────┼──────────────────────────────┼─────────────────────┤
│ • EOD stock quotes│ • Evaluate Materiality Gate  │ • Check predictions │
│ • MOPS disclosures│ • Run Investment Agent (LLM) │ • Directional score │
│ • FRED macro data │ • Update recommendations     │ • Brier calibration │
│ • Schedule: Daily │ • Trigger: On material shift │ • Schedule: Weekly  │
└───────────────────┴──────────────────────────────┴─────────────────────┘
```

### Frequency 1: Low-Frequency Data Collection (Pure Automation)
Runs periodically according to source data schedules without invoking LLMs:
- **TWSE OpenAPI:** Synchronizes daily end-of-day closing prices, volumes, and P/E ratios.
- **MOPS (Market Observation Post System):** Fetches official regulatory announcements and corporate material disclosures.
- **FRED (Federal Reserve Economic Data):** Updates macroeconomic indicators (interest rates, CPI, liquidity).
- Records all data collection timestamps and raw telemetry in SQLite.

### Frequency 2: Medium-Frequency Research & Recommendation (Governed AI)
Triggered only when sufficient new information has arrived, guarded by the **Materiality Gate**:
- **Core Principle: "Data update does not equal recommendation update."**
- If market prices fluctuated within ordinary bands (e.g., $< 3\%$) and no major corporate filings appeared, AIA maintains the previous investment thesis (`PreserveExistingDecision`), saving 100% of LLM reasoning tokens.
- If a material event occurs (e.g., severe price swing, major policy change, quarterly report), an active Investment Agent reasoning task is scheduled.

### Frequency 3: Low-Frequency Prediction Audit & Calibration
Operates periodically (e.g., weekly or post-horizon) to inspect the fidelity of historical advice:
- Audits matured price targets and forecasts.
- Calculates directional accuracy and probabilistic **Brier Scores** ($Brier = \frac{1}{N}\sum(p_i - o_i)^2$).
- Compares distinct research models to foster systematic improvement over time.

---

## 3. Worker Implementations

### ScheduleWorker ([`schedule_worker.go`](file:///home/wade/src/github/aia/go/internal/worker/schedule_worker.go))
- **Interval:** 15 seconds.
- Scans `schedule_rules` in SQLite for rules whose `next_run_at <= CURRENT_TIMESTAMP`.
- Instantiates new Tasks in `READY` state.
- Automatically calculates the next trigger time using cron syntax or duration offsets.

### TaskWorker ([`task_worker.go`](file:///home/wade/src/github/aia/go/internal/worker/task_worker.go))
- **Interval:** 5 seconds.
- **Concurrency Limit:** Regulated by a buffered channel semaphore (default: 3 concurrent tasks).
- Claims `READY` tasks atomically, transitions them to `RUNNING`, and dispatches them to their designated Agent executor.
- Dispatches execution events to `execution_events` and manages failure retries (`max_retries`).

---

## 4. Task State Machine

```text
               ┌──────────┐
               │  DRAFT   │
               └────┬─────┘
                    │ Submit
                    ▼
               ┌──────────┐
               │  READY   │ ◄─── Retry on Transient Failure
               └────┬─────┘
                    │ TaskWorker Picks Up
                    ▼
               ┌──────────┐
      ┌────────┤ RUNNING  ├────────┐
      │        └────┬─────┘        │
      │ Requires    │ Success      │ Unrecoverable
      │ Approval    ▼              │ Error
      ▼        ┌───────────┐       ▼
┌───────────┐  │ COMPLETED │  ┌────────┐
│  WAITING  │  └───────────┘  │ FAILED │
│ _APPROVAL │                 └────────┘
└─────┬─────┘
      │ User Approves
      └─────────► READY
```

1. **`DRAFT`:** Initial task creation, arguments validation.
2. **`READY`:** Enqueued for background execution.
3. **`RUNNING`:** Actively claimed by a TaskWorker worker thread.
4. **`WAITING_APPROVAL`:** Paused by policy rule awaiting user authorization via Web Console or CLI.
5. **`COMPLETED`:** Output validated, artifacts saved, and status recorded.
6. **`FAILED`:** Retries exhausted or terminal error encountered.

---

## 5. Activity Monitoring

The runtime continuously exposes its real-time operating metrics via [`internal/system.GetActivityMonitor`](file:///home/wade/src/github/aia/go/internal/system/activity.go):

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

This telemetry is accessible through:
- **CLI:** `aia monitor` / `aia worker monitor`
- **REST API:** `GET /api/v1/system/activity`
- **Web Console:** Live telemetry card on the Overview dashboard.
