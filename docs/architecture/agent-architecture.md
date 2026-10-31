# AIA Architecture: Agent Architecture & Domain Design

AIA adopts an **agent hierarchy** designed for personal productivity and autonomous domain expertise. It separates conversational coordination from specialized domain intelligence, while delegating low-level tool invocation and execution sandboxing to the external **Antigravity CLI (`agy`)**.

---

## 1. Agent Hierarchy

```text
                        ┌────────────────────────┐
                        │     Personal Agent     │
                        │ (Coordinator / Router) │
                        └───────────┬────────────┘
                                    │
         ┌──────────────────────────┼──────────────────────────┐
         │                          │                          │
┌────────▼────────┐        ┌────────▼────────┐        ┌────────▼────────┐
│Investment Agent │        │  Coding Agent   │        │  Writing Agent  │
│TWSE, MOPS, FRED │        │ Codebase, Git   │        │ Docs, Synthesis │
└────────┬────────┘        └────────┬────────┘        └────────┬────────┘
         │                          │                          │
┌────────▼────────┐                 │                 ┌────────▼────────┐
│System Collector │                 │                 │System Evaluator │
│Automated Data   │                 │                 │Brier Calibrator │
└─────────────────┘                 │                 └─────────────────┘
                                    │
                                    ▼
                     ┌─────────────────────────────┐
                     │ Antigravity CLI Harness     │
                     │  • Headless execution       │
                     │  • Subagent spawning        │
                     │  • Sandbox & permissions    │
                     └─────────────────────────────┘
```

### Agent Roles & Responsibilities

| Agent Identifier | Display Name | Core Responsibility |
| :--- | :--- | :--- |
| `personal-agent` | Personal Agent | User intent comprehension, task decomposition, and routing to domain agents. |
| `investment-agent` | Investment Agent | Financial analysis (TSMC 2330, Foxconn 2317, MediaTek 2454), macro research, recommendation generation. |
| `coding-agent` | Coding Agent | Code modification, refactoring, test execution, and architecture implementation. |
| `writing-agent` | Writing Agent | Technical documentation, specification drafting, and release notes compilation. |
| `system-collector` | Data Collector | Scheduled, automated sync of market data, MOPS announcements, and macro indicators. |
| `system-evaluator` | Evaluator | Background verification of forecasts, directional scoring, and Brier calibration. |

---

## 2. The Materiality Gate ("重大性檢驗閘道")

A core architectural principle of AIA is: **Data update does not equal recommendation update.**

In typical market conditions, price movements may be normal noise (e.g. fluctuating $\pm 0.5\%$), and corporate disclosures may be routine. Calling LLMs to re-evaluate investment thesis on every data sync burns unnecessary tokens and introduces decision volatility.

```text
[ Data Sync Event Completed ]
              │
              ▼
    ┌───────────────────┐
    │  Materiality Gate │
    └─────────┬─────────┘
              │
    ┌─────────┴────────────────────────────────┐
    │                                          │
    ▼                                          ▼
[ Price Delta < 3% AND                     [ Price Delta >= 3% OR
  No Material Announcements ]                Material Announcement Present ]
    │                                          │
    ▼                                          ▼
┌──────────────────────────────┐           ┌─────────────────────────────┐
│ PreserveExistingDecision()   │           │ Invoke Investment Agent     │
│ • Re-use previous thesis     │           │ • Build research snapshot   │
│ • Carry forward confidence   │           │ • Run agy prompt evaluation │
│ • 100% Token Savings         │           │ • Formulate new thesis      │
└──────────────────────────────┘           └─────────────────────────────┘
```

### Implementation Details ([`investment/service.go`](file:///home/wade/src/github/aia/go/internal/investment/service.go))

1. **Threshold Check:** Default price threshold is $3\%$. If $|\frac{P_{today} - P_{prev}}{P_{prev}}| < 0.03$ and zero material disclosures were filed, the gate is closed.
2. **`PreserveExistingDecision`:** Automatically carries forward the active thesis, updates the verification baseline, notes the reason (`"Price change within normal bounds; no material disclosures"`), and skips LLM dispatch.
3. **Override:** Users can force a full re-evaluation at any time via CLI (`aia invest research --force`) or Web Console.

---

## 3. Integration with Antigravity CLI (`agy`)

AIA integrates with Google's Antigravity CLI (`agy`) as an external AI execution engine:

- **Headless Mode:** Executed with `--headless` and structured JSON/NDJSON streams (`--output-format=json`).
- **Tool Sandbox:** agy's built-in sandbox confines file writes and shell commands to explicit project directories.
- **Subagents:** agy handles subagent invocation (e.g., `research`, `self`, `image-generator`) natively under its session control.
- **Session Preservation:** AIA tracks agy conversation IDs, allowing multi-turn continuation and workspace checkpointing.

---

## 4. Memory Architecture

AIA implements a dual-layer memory system to maintain continuity across sessions:

```text
┌────────────────────────────────────────────────────────┐
│                   AIA Memory System                    │
│                                                        │
│  ┌─────────────────────────┐ ┌───────────────────────┐ │
│  │    Short-Term Memory    │ │   Long-Term Memory    │ │
│  │  • Active task context  │ │ • Verified principles │ │
│  │  • Execution trace logs │ │ • User preferences    │ │
│  │  • Temporary data store │ │ • Domain knowledge    │ │
│  └───────────┬─────────────┘ └───────────▲───────────┘ │
│              │                           │             │
│              ▼ Promote via Review        │             │
│        ┌─────────────────────────────────┴──┐          │
│        │        Memory Candidates           │          │
│        │  (Extract, Deduplicate, Validate)  │          │
│        └────────────────────────────────────┘          │
└────────────────────────────────────────────────────────┘
```

1. **Short-Term Memory:** Transient session logs, conversation histories, and intermediate tool results stored per execution.
2. **Memory Candidates:** Extracted insights awaiting promotion to prevent memory contamination.
3. **Long-Term Memory:** Explicit, versioned knowledge entries categorized by domain (`personal`, `investment`, `coding`, `general`), directly queried by agents during prompt composition.
