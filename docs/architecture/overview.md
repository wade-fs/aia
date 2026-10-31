# AIA Architecture: Overview

**Autonomous Intelligence Architecture (AIA)** is a personal, local-first AI agent platform designed to coordinate specialized domain agents, manage persistent tasks, maintain long-term memory, and integrate AI into reliable, auditable daily workflows.

Rather than treating AI as an ephemeral chat session, AIA provides a structured runtime environment where agents possess managed lifecycles, structured state machines, auditable execution logs, and strict governance boundaries.

---

## 1. Design Philosophy

AIA is designed around four foundational pillars:

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Foundational Pillars                         │
├─────────────────┬─────────────────┬──────────────┬──────────────┤
│    Personal     │   Persistent    │   Modular    │   Governed   │
│  Single-user,   │  Tasks, state,  │ Domain-based │  Permission  │
│  local-first,   │  and memory     │ specialization│ policies &  │
│  user-governed  │  survive chats  │  in Go runtime│ human review │
└─────────────────┴─────────────────┴──────────────┴──────────────┘
```

1. **Personal:** Owned and controlled by a single individual. All data, SQLite databases, and execution workspaces reside locally on the user's machine.
2. **Persistent:** Tasks, execution histories, memory items, and evaluations persist across machine reboots and individual conversation sessions.
3. **Modular:** Specialization through domain agents (e.g., Investment Agent, Coding Agent, Writing Agent) operating within a shared Go runtime.
4. **Governed:** Autonomous background actions are bounded by policies, rate limits, materiality checks, and optional approval gates.

---

## 2. Platform Architecture

AIA explicitly separates **platform orchestration & state management** from the **underlying AI execution engine**.

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   AIA Platform (Go Runtime Daemon)                     │
│                                                                        │
│   ┌──────────────────────────────────────────────────────────────┐     │
│   │                     Personal Agent                           │     │
│   │           (Intent Analysis, Coordination & Routing)           │     │
│   └───────────────┬──────────────────────────────┬───────────────┘     │
│                   │                              │                     │
│   ┌───────────────▼──────────────┐ ┌─────────────▼─────────────────┐   │
│   │        Domain Agents         │ │       System Workers          │   │
│   │  • Investment Agent          │ │  • Schedule Worker (15s)      │   │
│   │  • Coding Agent              │ │  • Task Worker (Pool: 3, 5s)  │   │
│   │  • Writing Agent             │ │  • Materiality Gate Evaluator │   │
│   └───────────────┬──────────────┘ └─────────────┬─────────────────┘   │
│                   │                              │                     │
│   ┌───────────────▼──────────────────────────────▼─────────────────┐   │
│   │                    Core Management Layer                       │   │
│   │  • Task Lifecycle State Machine   • Long-Term Memory System    │   │
│   │  • Policy & Approval Engine       • Workspace Projects         │   │
│   │  • Prediction & Brier Evaluation  • Activity Monitor Snapshot  │   │
│   └──────────────────────────────┬─────────────────────────────────┘   │
│                                  │                                     │
│   ┌──────────────────────────────▼─────────────────────────────────┐   │
│   │                    Persistence & Interfaces                    │   │
│   │  • SQLite Storage (WAL Mode)     • Embedded Web Console (7777) │   │
│   │  • REST & SSE API                • Cobra CLI Tooling           │   │
│   └──────────────────────────────┬─────────────────────────────────┘   │
└──────────────────────────────────┼─────────────────────────────────────┘
                                   │
                                   ▼
             ┌──────────────────────────────────────────┐
             │       AI Execution Harness (agy)         │
             │   Google Antigravity CLI Subprocesses    │
             │   Sandboxed Tools, LLMs, Subagents       │
             └──────────────────────────────────────────┘
```

---

## 3. AIA vs. Conversational Chatbots

| Dimension | Typical Chatbot | Autonomous Intelligence Architecture (AIA) |
| :--- | :--- | :--- |
| **Lifecycle** | Ephemeral (dies when session ends) | Persistent daemon (`systemd --user`) running 24/7 |
| **Execution** | Reactive single-turn answers | Proactive 3-frequency automation (Data, Research, Audit) |
| **State** | Lost or kept in proprietary cloud | Local SQLite database (`~/.local/share/aia/aia.db`) |
| **Token Cost** | Re-analyzes everything unconditionally | **Materiality Gate**: Skips LLM calls if data change is non-material |
| **Verification** | None; produces persuasive text | Tracks predictions, directional accuracy, and **Brier Scores** |
| **Governance** | Unconstrained or model-level | Platform policies, approval gates (`WAITING_APPROVAL`), file isolation |

---

## 4. Key Subsystems

- **[Runtime & Workers](file:///home/wade/src/github/aia/docs/architecture/runtime.md):** Goroutine-based task executor, schedule dispatcher, and 3-frequency automation engine.
- **[Agent Architecture](file:///home/wade/src/github/aia/docs/architecture/agent-architecture.md):** Agent hierarchy, Materiality Gate, Antigravity (`agy`) integration, and persistent memory.
- **[Security Model](file:///home/wade/src/github/aia/docs/architecture/security-model.md):** Local single-user isolation, one-time token authentication, and credential sandboxing.
