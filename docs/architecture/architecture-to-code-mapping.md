# AIA 架構與程式碼映射對照表 (Architecture-to-Code Mapping)

**文件代號：** AIA-ARCH-MAP-01  
**當前實作基準：** Phase 0 ～ Phase 6 完整落地 + Phase 7 硬化整合 (Migrations `001` ～ `019`)  
**對照規格：** [docs/specifications/](file:///home/wade/src/github/aia/docs/specifications) (AIA-00 ～ AIA-22)  
**文件狀態：** Active Production Architecture Mapping  
**最後更新：** 2026-11-05  

---

## 1. 規格映射與實作成熟度總覽 (Current Implementation Overview)

目前 Go 代碼庫已完整實作 Phase 0 至 Phase 6 以及 AIA-21（Universal Brief & Clarification）、AIA-22（Complex Problem Analysis & Planning）、AIA-19（Operations, Backup & Tracing）、AIA-13（Autonomy Evaluation Metrics）、AIA-14（Multi-tier Verification & Monitoring）與 AIA-06（16-Tab Web Console & Emergency Stop）的領域模型、資料庫結構、核心服務、CLI 與 Autonomy Loop 整合：

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   AIA-00 ~ AIA-22 程式碼對照成熟度總覽                 │
├──────────────────────┬─────────────┬───────────────────────────────────┤
│ 成熟度分類           │ 規格數量    │ 代表規格編號                      │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 生產就緒 (Ready)     │ 23 份 (100%)│ AIA-00, AIA-01, AIA-02, AIA-03,   │
│                      │             │ AIA-04, AIA-05, AIA-06, AIA-07,   │
│                      │             │ AIA-08, AIA-09, AIA-10, AIA-11,   │
│                      │             │ AIA-12, AIA-13, AIA-14, AIA-15,   │
│                      │             │ AIA-16, AIA-17, AIA-18, AIA-19,   │
│                      │             │ AIA-20, AIA-21, AIA-22            │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 核心就緒 (Core Ready │ 0 份 (0%)   │ (全量規格核心服務與資料庫皆已就緒) │
│  / 待串接)           │             │                                   │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 待補強 (Partial)     │ 0 份 (0%)   │ (度量模組與維運工具皆已實裝)       │
└──────────────────────┴─────────────┴───────────────────────────────────┘
```

> **維度區分說明**：
> - **規格（Specification）**：AIA-00～22 規範皆已正式核定。
> - **實作（Implementation）**：Go 服務、資料庫 Migrations 001～019、CLI 與 Web Console 皆已 100% 落地。
> - **驗證（Verification）**：單元與整合測試套件 100% 通過；多領域真實場景（Writing 小說規範稽核、Coding 代碼自修閉環、Investment 世代演化回放）之長週期自主行為端到端驗證為進行中目標。

---

## 2. 規格逐項映射與代碼位置 (Detailed Spec-to-Code Mapping)

### 2.1 總體藍圖、需求、架構與 Runtime (AIA-00 ~ AIA-04)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-00 總體架構與演進藍圖** | 6 層架構、R1~R6 能力模型。 | [`docs/development/roadmap.md`](file:///home/wade/src/github/aia/docs/development/roadmap.md), [`app/app.go`](file:///home/wade/src/github/aia/go/internal/app/app.go) | 生產就緒 | 7 個 Master Phase 與 M0~M10 里程碑架構已落實於代碼結構。 |
| **AIA-01 產品需求規格** | Local-First、Single User、Task 狀態機。 | [`domain/task/`](file:///home/wade/src/github/aia/go/internal/domain/task), [`storage/sqlite/`](file:///home/wade/src/github/aia/go/internal/storage/sqlite) | 生產就緒 | 本機單人 SQLite WAL 模式，完整 Task 狀態機與依賴 DAG。 |
| **AIA-02 系統架構規格** | Runtime、Autonomy Core、Services、Agents 分層。 | [`app/app.go`](file:///home/wade/src/github/aia/go/internal/app/app.go), [`service/`](file:///home/wade/src/github/aia/go/internal/service), [`eventbus/`](file:///home/wade/src/github/aia/go/internal/eventbus) | 生產就緒 | 嚴格分層，依賴注入，EventBus 事件驅動。 |
| **AIA-03 Go Runtime 實作** | Manager 生命週期、Workers、Recovery、SQLite。 | [`worker/`](file:///home/wade/src/github/aia/go/internal/worker), [`recovery/`](file:///home/wade/src/github/aia/go/internal/recovery), [`storage/sqlite/`](file:///home/wade/src/github/aia/go/internal/storage/sqlite) | 生產就緒 | ScheduleWorker (15s)、TaskWorker (3 slots)、LogRetentionPurger、AutoWakeWorker、重啟自動恢復。 |
| **AIA-04 測試與驗收發布** | Unit/Integration 測試、Mock Provider、回歸門禁。 | [`go/tests/unit/`](file:///home/wade/src/github/aia/go/tests/unit), [`go/tests/integration/`](file:///home/wade/src/github/aia/go/tests/integration) | 生產就緒 | 全量自動化單元與整合測試皆通過（含 real agy 整合測試）。 |

---

### 2.2 領域 Agent、介面與操作手冊 (AIA-05 ~ AIA-07)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-05 Investment Agent** | 虛擬資金、營業日自主觀察、Brier 校準。 | [`service/investment/`](file:///home/wade/src/github/aia/go/internal/service/investment), [`collector/`](file:///home/wade/src/github/aia/go/internal/collector), [`experiment/`](file:///home/wade/src/github/aia/go/internal/experiment) | 生產就緒 | 11 張投資專用表，台股/總經採集，Brier/LogLoss 雙軌校準。 |
| **AIA-06 Web UI & 介面** | Port 7777, Activity Monitor, 審批與監控。 | [`web/host/`](file:///home/wade/src/github/aia/go/internal/web/host), [`web/console/`](file:///home/wade/src/github/aia/go/internal/web/console), [`go/web/assets/dist/`](file:///home/wade/src/github/aia/go/web/assets/dist) | 生產就緒 | 後端 REST API 齊全；前端 16 個導覽分頁（Overview, Tasks, Pending, Goals, Cycles, Learnings, Experiments, Delegations, Clarifications, Problems, Investment, Coding, Writing, Memory, Agents, Audit）與導覽列 Emergency Stop 按鈕。 |
| **AIA-07 使用者操作手冊** | CLI 指令集、安裝配置、維護與故障排除。 | [`cli/`](file:///home/wade/src/github/aia/go/internal/cli), [`docs/operations/`](file:///home/wade/src/github/aia/docs/operations) | 生產就緒 | Cobra CLI 完整支援 agent, task, cycle, goal, learning, brief, clarify, problem (含 plan 功能), backup, trace, worker。 |

---

### 2.3 自主核心與學習四大支柱 (AIA-08 ~ AIA-11)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-08 Autonomy Core** | 11 步驟自主閉環、共用決策架構、No-Action 判定。 | [`autonomy/cycle/`](file:///home/wade/src/github/aia/go/internal/autonomy/cycle), [`autonomy/deliberation/`](file:///home/wade/src/github/aia/go/internal/autonomy/deliberation), [`autonomy/observation/`](file:///home/wade/src/github/aia/go/internal/autonomy/observation) | 生產就緒 | 11 步 Coordinator，EV 權衡矩陣，支援 `ACT`、`DEFER`、`MONITOR`、`ASK_HUMAN`、`NO_ACTION`，整合 AutoWakeWorker。 |
| **AIA-09 主動性與目標演化** | 候選工作探索、Goal 樹、目標演化提案。 | [`autonomy/initiative/`](file:///home/wade/src/github/aia/go/internal/autonomy/initiative), [`service/goal/`](file:///home/wade/src/github/aia/go/internal/service/goal), [`domain/goal/`](file:///home/wade/src/github/aia/go/internal/domain/goal) | 生產就緒 | 候選發現去重，3-way Goal 演化，Anti-Goalpost-Moving 門禁。 |
| **AIA-10 連續性與專案管理** | Append-only Event History、Context 重建、快照。 | [`continuity/`](file:///home/wade/src/github/aia/go/internal/continuity), [`storage/sqlite/migrations/event_history*`](file:///home/wade/src/github/aia/go/internal/storage/sqlite/migrations/event_history_and_provenance.go) | 生產就緒 | SHA-256 雜湊鏈事件流，ContextReconstructor，專案快照還原，跨重啟脈絡恢復。 |
| **AIA-11 自我修正與持續學習** | 預期差距比對、RCA 分析、3-Tier 修正、L0~L3 學習。 | [`autonomy/learning/`](file:///home/wade/src/github/aia/go/internal/autonomy/learning), [`domain/learning/`](file:///home/wade/src/github/aia/go/internal/domain/learning) | 生產就緒 | 自動落差評估，可驗證 RCA 假說，7 大比對欄位，負向約束抑制，橋接自修驗證管線。 |

---

### 2.4 跨領域協作、實驗與自我修改 (AIA-12 ~ AIA-15)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-12 跨領域協作與委派** | Agent 間任務委派合約、交付物驗收、跨專案協同。 | [`service/delegation/`](file:///home/wade/src/github/aia/go/internal/service/delegation), [`domain/experiment/`](file:///home/wade/src/github/aia/go/internal/domain/experiment) (DelegationRequest 等) | 生產就緒 | `delegation_requests` 狀態機，自動工單生成與驗收檢查；委派模型位於 experiment 領域。 |
| **AIA-13 自主性評估基準** | 主動率、介入率、連續性復原準確率等獨立指標。 | [`autonomy/evaluation/`](file:///home/wade/src/github/aia/go/internal/autonomy/evaluation), [`experiment/`](file:///home/wade/src/github/aia/go/internal/experiment) | 生產就緒 | 通用全系統自主性評估指標模組 (`evaluator.go`, `metrics.go`)，涵蓋主動率、人工介入率、校準度與回歸率。 |
| **AIA-14 安全代碼自我修改** | Level A/B/C 三級隔離、Git Worktree 沙盒、自修流水線。 | [`service/selfmod/`](file:///home/wade/src/github/aia/go/internal/service/selfmod), [`domain/selfmod/`](file:///home/wade/src/github/aia/go/internal/domain/selfmod), [`autonomy/self_evolution_engine.go`](file:///home/wade/src/github/aia/go/internal/autonomy/self_evolution_engine.go) | 生產就緒 | 三級分類、沙盒 Worktree 隔離、V1~V6 驗證管線、診斷重試、審核分流與部署後監控 (Migration 014/019)。 |
| **AIA-15 自主實驗生命週期** | 世代世代實驗 (`G1` vs `G2`)、5 體制判決、長期存檔。 | [`experiment/`](file:///home/wade/src/github/aia/go/internal/experiment), [`domain/experiment/`](file:///home/wade/src/github/aia/go/internal/domain/experiment) | 生產就緒 | 凍結 G1 規則基準與 SQLite 觸發器防篡改，完整 11-狀態轉換控制，DiD 對照評估。 |

---

### 2.5 知識、治理、參考架構、維運與問題企劃 (AIA-16 ~ AIA-22)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-16 證據與知識完整性** | Provenance 來源評估、Source/Evidence/Claim 鏈條。 | [`service/knowledge/`](file:///home/wade/src/github/aia/go/internal/service/knowledge), [`domain/knowledge/`](file:///home/wade/src/github/aia/go/internal/domain/knowledge) | 生產就緒 | 5D 來源可信度評分，Claim 修訂歷史，決策證據鏈回溯與傳遞。 |
| **AIA-17 運作治理與監督** | G0~G3 治理、審批閘門、1 秒緊急停機。 | [`service/governance/`](file:///home/wade/src/github/aia/go/internal/service/governance), [`service/approval/`](file:///home/wade/src/github/aia/go/internal/service/approval), [`policy/`](file:///home/wade/src/github/aia/go/internal/policy) | 生產就緒 | 受控 Emergency Stop，防代理人自我審批，預算硬約束，超時處理。 |
| **AIA-18 端到端參考架構** | 11 步驟自主閉環整合、控制流與資料流規範。 | [`autonomy/`](file:///home/wade/src/github/aia/go/internal/autonomy), [`app/`](file:///home/wade/src/github/aia/go/internal/app) | 生產就緒 | 跨領域共用自主閉環，PiT AsOf 時鐘防未來資訊洩漏。 |
| **AIA-19 部署維運與可觀測性** | systemd --user 常駐、Activity Monitor、日誌管理。 | [`service/backup/`](file:///home/wade/src/github/aia/go/internal/service/backup), [`telemetry/`](file:///home/wade/src/github/aia/go/internal/telemetry), [`worker/`](file:///home/wade/src/github/aia/go/internal/worker) | 生產就緒 | 線上 SQLite 備份/還原/校驗 (`BackupService`)、分散式追蹤 (`Tracing`)、維運度量 (`Metrics`) 與背景日誌過期清理器 (`LogRetentionPurger`)。 |
| **AIA-20 創作規範與小說專案** | 創作大綱、Master Bible、人物譜系與章節管線。 | [`service/writing/`](file:///home/wade/src/github/aia/go/internal/service/writing), [`domain/writing/`](file:///home/wade/src/github/aia/go/internal/domain/writing) | 生產就緒 | 支援小說專案連續性、世界觀設定與一致性檢查。 |
| **AIA-21 專案綱要與釐清管理** | Universal Project Brief、歷史版本追蹤、人機互動釐清。 | [`service/brief/`](file:///home/wade/src/github/aia/go/internal/service/brief), [`service/clarification/`](file:///home/wade/src/github/aia/go/internal/service/clarification), [`domain/brief/`](file:///home/wade/src/github/aia/go/internal/domain/brief) | 生產就緒 | 跨領域通用 Brief Schema，欄位級不可變修訂歷史，釐清請求工作流，Brief Observer 自主感知。 |
| **AIA-22 複雜問題分析與規劃** | 形式化問題建模、多階段診斷、可執行計畫 DAG 與變更。 | [`service/problem/`](file:///home/wade/src/github/aia/go/internal/service/problem), [`domain/plan/`](file:///home/wade/src/github/aia/go/internal/domain/plan), [`autonomy/cycle/`](file:///home/wade/src/github/aia/go/internal/autonomy/cycle) | 生產就緒 | 獨立服務、儲存庫、CLI、REST、反覆任務失敗自主升級 (§10) 與 11 步 Autonomy Loop 完整整合；Plan 模型位於 domain/plan，由 problem 服務協調。 |

---

## 3. 現有程式庫關鍵目錄結構

```text
go/
├── cmd/aia/main.go                      # 頂層 CLI 入口
├── internal/
│   ├── app/app.go                       # 依賴注入與服務裝配
│   ├── autonomy/                        # 11 步驟自主閉環核心
│   │   ├── clock/                       # Point-in-Time 回放時鐘
│   │   ├── cycle/                       # 主調度協調器與問題升級鉤子
│   │   ├── deliberation/                # EV 權衡與決策
│   │   ├── evaluation/                  # AIA-13 自主性度量模組
│   │   ├── initiative/                  # 候選工作探索與去重
│   │   ├── learning/                    # 自我修正與 RCA
│   │   ├── observation/                 # 多領域觀察蒐集 (含 brief observer)
│   │   ├── recall/                      # 記憶檢索與約束
│   │   └── self_evolution_engine.go     # 自主演化推進器
│   ├── cli/                             # 所有子命令 (含 backup, trace, worker, brief, clarify, problem)
│   ├── config/                          # 完整 YAML 配置載入器
│   ├── continuity/                      # 脈絡重建與事件歷史還原
│   ├── domain/                          # 領域模型實體 (brief, clarification, problem, plan, selfmod, experiment 含 DelegationRequest 等)
│   ├── experiment/                      # 世代實驗控制器 (G1 vs G2)
│   ├── policy/                          # 白名單與預算門禁
│   ├── provider/                        # Antigravity 沙盒適配器
│   ├── service/                         # 各業務服務 (backup, brief, clarification, problem, delegation, selfmod)
│   ├── storage/sqlite/                  # SQLite WAL 驅動與 Migrations 001~019
│   ├── telemetry/                       # 分散式追蹤與維運度量指標 (AIA-19)
│   ├── web/                             # Web Console API 與靜態資源 (16 tabs & Emergency Stop)
│   └── worker/                          # 背景工作排程、LogRetentionPurger、AutoWake 與受控停止
└── tests/                               # 全量單元與整合測試套件
```
