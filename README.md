# AIA — Autonomous Intelligence Architecture

**A Personal AI Agent Platform for Verifiable Autonomous Behavior Research**

AIA 是一個以研究與實證 AI Agent 長期自主行為為目標的本機優先（Local-First）個人自主實驗平台。

本專案聚焦於可觀察、可量測、可證偽且受治理約束的四項自主行為能力（不涉及對主觀意識的宣稱）：

1. **Initiative（主動性）** — 觀察環境與專案脈絡、發現有價值問題，並在無需行動時做出有據可查的 `NO_ACTION`。
2. **Continuity（時間連續性）** — 跨越 Process 重啟與多日週期，透過不可變事件鏈與證據溯源重建工作狀態與未決事項。
3. **Verifiable Self-Correction（可驗證自我修正與學習機制）** — 比對預期與實際落差、提出可檢驗的根因假設（RCA）、藉由對照基準與退化檢查驗證修正成效（`L0`～`L3`），並保留失敗反面約束。
4. **Governed Goal Evolution（受控目標演化）** — 嚴格區分 `PERSISTENCE`、`OPTIMIZATION` 與 `REVISION`，透過防移動球門守衛（Anti-Goalpost-Moving）防止因短期挫敗擅自弱化目標。

---

## 1. Current Status

目前平台已完成 **Phase 0 ～ Phase 5（含 Phase 3.5 安全加固、Phase 4 可驗證學習、Phase 5A 雙世代投資實驗與 Phase 5B 多領域 MVA & 跨領域委派）**，資料庫推進至 **Migration `012`**；下一階段為 **Phase 6（受控沙盒代碼自修與系統演化）**。

| Domain Module | Status | Description |
|---|---|---|
| **Investment** | **GENERATIONAL EXPERIMENT READY (Phase 5A)** | 雙世代獨立 NT$100,000 帳本（`G1` 凍結規則基準組 vs `G2 -> G2.x` LLM 實驗演化組）、Brier + LogLoss 機率校準與行動信心分離、五大科學判讀矩陣與零前視偏差回放 |
| **Coding** | **MVA COMPLETED (Phase 5B)** | 程式碼健康掃描（`CodeHealthObserver`）、沙盒補丁執行（`CodingPilotExecutor`）、測試驅動自我修正達到 `L2_VERIFIED` |
| **Writing** | **MVA COMPLETED (Phase 5B)** | 實證對接 `~/src/github/novel/yuan/`（6 卷 540 章）、三層知識體系（【史實骨架】、【合理推演】、【小說虛構】）、區分藝術虛構（理性 `NO_ACTION`）與年表矛盾（`L2_VERIFIED`） |

### 1.1 Core Implementation Status

| Component | Status | Description |
|---|---|---|
| **Go Runtime & Workers** | **IMPLEMENTED** | 常駐 `ScheduleWorker`（15s）與 `TaskWorker`（5s，3 並行槽位）及啟動恢復（`Recovery`） |
| **SQLite Persistence** | **IMPLEMENTED** | WAL 模式、Migrations `001`～`012`、Repository 層與防竄改 Triggers |
| **Continuity & Provenance** | **IMPLEMENTED (Phase 1)** | SHA-256 Hash-Chained `event_history`、5 維度 `sources`/`evidence`/`claims`、`ContextReconstructor` |
| **Governance & Emergency Stop** | **IMPLEMENTED (Phase 2)** | Policy / Approval / Permission 三層分離、`< 1s` 受控緊急停止（`EMERGENCY_STOP`）與人類強制復歸 |
| **Autonomy Core (11-Step Loop)** | **IMPLEMENTED (Phase 3)** | `Observe` ➔ `Recall` ➔ `Assess` ➔ `Discover` ➔ `Deliberate` ➔ `Decide` (`ACT`/`DEFER`/`MONITOR`/`ASK_HUMAN`/`NO_ACTION`) ➔ `Governance Check` ➔ `Execute` ➔ `Verify` ➔ `Learn` ➔ `Reassess` |
| **Budget Gate & OS Sandbox** | **IMPLEMENTED (Phase 3.5)** | 每日 Token / Cycle 硬閘門、`service.env` 機密剝除、`bwrap`/`systemd-run` 唯讀掛載隔離、拒絕盲信 Exit Code 0 |
| **Point-in-Time Replay Clock** | **IMPLEMENTED (Phase 3.5)** | `AsOf` 時鐘與 `<= asOf` 查詢過濾、回放模式即時網路阻斷，防範前視偏差（Look-Ahead Bias） |
| **Self-Correction & Learning** | **IMPLEMENTED (Phase 4)** | 3-Tier Runtime Profile、`L0_RECORDED`～`L3_GENERALIZED` 四階成熟度、7 大科學對照欄位、退化自動回滾與負向約束 |
| **Goal Evolution Engine** | **IMPLEMENTED (Phase 4)** | `PERSISTENCE` / `OPTIMIZATION` / `REVISION` 三態分流與 `STRATEGIC` 目標防移動球門守衛 |
| **Generational Experiment Engine** | **IMPLEMENTED (Phase 5A)** | `experiments` 11 狀態機、凍結 `G1` 活體對照組、`G2 -> G2.1 -> G2.2` 譜系演化與雙重差分（DiD）驗證橋接 |
| **Cross-Domain Delegation** | **IMPLEMENTED (Phase 5B)** | `DelegationService` 跨領域合約狀態機（`PENDING` ➔ `IN_PROGRESS` ➔ `DELIVERED` ➔ `VERIFIED`）、自動工單派發、Web Console API (`/api/v1/delegations`) 與 CLI (`aia delegation`) |
| **Safe Code Self-Modification** | **PLANNED (Phase 6)** | 隔離工作區代碼自修、多層測試流水線、Diff Review 與異常自動回滾 |

> **認識論聲明**：Phase 4 完成代表「可驗證的自我修正與學習基礎機制」已落地並通過合成對抗情境（Scenarios A～E）驗證，**不代表已宣稱完成通用長期自主學習**；跨週期泛化能力（`L3`）須在 Phase 5 長期實驗中持續檢驗。

### 1.2 Investment Data Status

目前已建立 TWSE、TPEx/MOPS、FRED、SEC 等 Collector Provider 架構，但部分 Provider 在無外部金鑰或離線環境下仍具 Fallback 行為。正式投資實驗必須確認資料來源真實性、基準日期、時效性與零前視偏差（**資料來源存在，不等於資料可信**）。

---

## 2. Architecture

AIA 採用五層職責分離架構（詳見 [docs/Plan.md](file:///home/wade/src/github/aia/docs/Plan.md)）：

```text
AIA
├── Autonomy Core (go/internal/autonomy/)
│   ├── Observation & L1~L5 Associative Recall
│   ├── Candidate Initiative & Expected-Value Deliberation
│   ├── 11-Step Cycle Coordinator & Point-in-Time (AsOf) Clock
│   └── Verifiable Self-Correction & Learning Engine (L0~L3)
├── Runtime & Governance (go/internal/app, worker, policy, service)
│   ├── ScheduleWorker (15s) / TaskWorker (5s) / Recovery
│   ├── Policy Engine, Budget Gate, Scope-Bound Approvals & Emergency Stop
│   ├── Continuity Reconstructor & Provenance Knowledge Graph
│   └── Goal Hierarchy & Anti-Goalpost-Moving Evolution
├── Domain Modules
│   ├── Investment (Pilot Ready)
│   ├── Coding (Phase 5)
│   └── Writing (Phase 5)
└── Execution Harness (go/internal/provider/antigravity/)
    └── Antigravity CLI (agy) + Env Secret Scrubbing + OS Mount Namespace Sandbox
```

使用者自行透過 Antigravity CLI（`agy`）完成 OAuth 登入；AIA 不讀取、不複製、不解析 OAuth Token，且子行程執行時自動剝除 `service.env` 機密並遮蔽 `~/.config/aia` 與 `~/.local/share/aia`。

---

## 3. Autonomous Behavior Loop (11 Steps)

```text
1. Observe ──► 2. Recall ──► 3. Assess ──► 4. Discover ──► 5. Deliberate ──► 6. Decide
                                                                                 │
  11. Reassess ◄── 10. Learn ◄── 9. Verify ◄── 8. Execute ◄── 7. Governance Check ┘
```

- 定時喚醒不等於主動性；保存字串不等於時間連續性；修改 Prompt 不等於有效學習。
- 每一次修正皆須保存 **Baseline、Hypothesis、Intervention、Verification、Regression、Follow-up、Confidence** 七項對照紀錄；缺乏對照組（`PRE_POST_ONLY`）時信心值強制鎖上限 `<= 0.45` 且最高僅能判定為 `INCONCLUSIVE`。

---

## 4. Repository Structure

```text
aia/
├── README.md
├── Makefile
├── docs/
│   ├── Plan.md                         # 七階段總體實作規劃書 (Master Plan)
│   ├── memory.md                       # L1~L5 一致性記憶與認知架構
│   ├── architecture/                   # 系統架構、統一資料模型 (data-model.md)、安全模型 (security-model.md)
│   ├── development/                    # 路線圖 (roadmap.md)、實作現況 (implementation-status.md)、遷移計畫
│   ├── operations/                     # 安裝、設定 (configuration.md) 與 systemd 服務管理
│   └── specifications/                 # 正式規格書 AIA-00 ～ AIA-19 與跨規格一致性稽核
└── go/
    ├── cmd/aia/                        # CLI 入口點
    ├── internal/                       # 核心套件 (autonomy, continuity, domain, policy, provider, service, storage, web, worker)
    ├── prompts/                        # 領域與個人協調 Prompt 模板
    ├── scripts/                        # systemd --user 管理腳本
    └── tests/                          # unit/ 與 integration/ 測試套件
```

---

## 5. Quick Start & Key CLI Commands

### Build & Test

```bash
cd go
go build ./cmd/aia
go test -count=1 ./...
```

### Daemon & Emergency Controls

```bash
aia server start                     # 啟動常駐 Web Console (Port 7777) 與背景 Workers
aia status                           # 查看系統與 Runtime 狀態
aia stop --emergency -r "reason"     # 1 秒內凍結派發並受控取消執行中工單
aia resume --force -a human-admin    # 人類顯式解除緊急停止
```

### Autonomy Cycle, Goal, Learning & Governance CLI

```bash
# 11 步驟自主循環（支援歷史時間點零前視偏差回放）
aia cycle run --project default
aia cycle run --project default --as-of 2026-03-01T00:00:00Z --replay
aia cycle list
aia cycle initiatives

# 目標管理與受控演化
aia goal list
aia goal create --name "Maximize Total Portfolio Assets" --tier STRATEGIC
aia goal evolve <goal-id> --type OPTIMIZATION --reason "..."

# 自我修正與學習檢視 (Phase 4)
aia learning evals
aia learning rcas
aia learning corrections
aia learning records

# 審批與決策溯源
aia approval list
aia approval approve <approval-id> --actor human-admin --note "verified"
aia decision list

# 投資試驗與雙世代對照實驗 (Phase 5A)
aia experiment create-dual --project default --capital 100000
aia experiment step <experiment-id> --symbol 2330 --as-of 2026-05-02T06:00:00Z
aia experiment compare <experiment-id> --baseline G1 --candidate G2
aia experiment generations <experiment-id>
aia investment portfolio
aia investment sync 2330
aia investment proposals
aia investment predictions
aia investment evaluate
```

---

## 6. Seven-Phase Development Roadmap

詳見 [docs/Plan.md](file:///home/wade/src/github/aia/docs/Plan.md) 與 [docs/development/roadmap.md](file:///home/wade/src/github/aia/docs/development/roadmap.md)：

| Phase | Focus | Status |
|---|---|---|
| **Phase 0** | Baseline Audit & Cross-Specification Alignment | **COMPLETED** |
| **Phase 1** | Persistent State, Hash-Chained Event History & Knowledge Provenance (`007`–`008`) | **COMPLETED** |
| **Phase 2** | Operational Governance, Approvals & Controlled Emergency Stop (`009`) | **COMPLETED** |
| **Phase 3 & 3.5** | 11-Step Autonomy Core, Initiative, Budget Gate, OS Sandbox & PiT Replay (`010`) | **COMPLETED** |
| **Phase 4** | Verifiable Self-Correction, Learning Mechanism (`L0`–`L3`) & Governed Goal Evolution (`011`) | **COMPLETED** |
| **Phase 5** | Longitudinal Domain Experiments (`5A` Investment `G1` vs `G2`/`G2.x` **READY**; `5B` Coding, Writing & Delegation) (`012`) | **IN PROGRESS** |
| **Phase 6** | Isolated Sandbox Safe Code Self-Modification & Autonomous System Evolution | **PLANNED** |

---

## 7. Security & Financial Safety

- **No Automatic Real-Money Trading**：所有投資實驗均為虛擬資金記帳（NT$100,000），嚴禁串接真實券商自動下單。
- **Local Single-User & Loopback Default**：Web Console 預設綁定 `127.0.0.1:7777` 並採用一次性登入 Token（OTT）換發 HttpOnly Session。
- **No Permission Bypass**：嚴禁使用 `--dangerously-skip-permissions`；Agent 無權自行通過 Approval、修改 `STRATEGIC` 目標指標或竄改 Hash-Chained 歷史紀錄。

---

## 8. Documentation Index

- **總體規劃與記憶架構**：[docs/Plan.md](file:///home/wade/src/github/aia/docs/Plan.md) · [docs/memory.md](file:///home/wade/src/github/aia/docs/memory.md)
- **實作現況與資料模型**：[implementation-status.md](file:///home/wade/src/github/aia/docs/development/implementation-status.md) · [data-model.md](file:///home/wade/src/github/aia/docs/architecture/data-model.md) · [security-model.md](file:///home/wade/src/github/aia/docs/architecture/security-model.md)
- **正式規格書（v2.0）**：[docs/specifications/](file:///home/wade/src/github/aia/docs/specifications)（`AIA-00` 至 `AIA-19`）
