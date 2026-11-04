# AIA 架構與程式碼映射對照表 (Architecture-to-Code Mapping)

**文件代號：** AIA-ARCH-MAP-01  
**當前實作基準：** Phase 6 Active Baseline + AIA-21 & AIA-22 Implemented (Migrations `001` ～ `018`)  
**對照規格：** [docs/specifications/](file:///home/wade/src/github/aia/docs/specifications) (AIA-00 ～ AIA-22)  
**文件狀態：** Active Production Architecture Mapping  
**最後更新：** 2026-11-04  

---

## 1. 規格映射與實作成熟度總覽 (Current Implementation Overview)

目前 Go 代碼庫已完整實作 Phase 0 至 Phase 6 以及 AIA-21（Universal Brief & Clarification）、AIA-22（Complex Problem Analysis & Planning）的領域模型、資料庫結構、核心服務與 CLI：

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   AIA-00 ~ AIA-22 程式碼對照成熟度總覽                 │
├──────────────────────┬─────────────┬───────────────────────────────────┤
│ 成熟度分類           │ 規格數量    │ 代表規格編號                      │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 生產就緒 (Ready)     │ 18 份 (78%) │ AIA-00, AIA-01, AIA-02, AIA-03,   │
│                      │             │ AIA-04, AIA-05, AIA-07, AIA-08,   │
│                      │             │ AIA-09, AIA-10, AIA-11, AIA-12,   │
│                      │             │ AIA-15, AIA-16, AIA-17, AIA-18,   │
│                      │             │ AIA-20, AIA-21                    │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 核心就緒 (Core Ready │ 4 份 (17%)  │ AIA-06 (缺部分前端Tab),           │
│  / 待串接)           │             │ AIA-14 (缺多級執行與重試迴圈),    │
│                      │             │ AIA-19 (缺備份與日誌清理),        │
│                      │             │ AIA-22 (服務與CLI齊全，待入循環)  │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 待補強 (Partial)     │ 1 份 (5%)   │ AIA-13 (僅有投資與世代實驗度量，  │
│                      │             │         缺通用自主性指標計算器)   │
└──────────────────────┴─────────────┴───────────────────────────────────┘
```

---

## 2. 規格逐項映射與代碼位置 (Detailed Spec-to-Code Mapping)

### 2.1 總體藍圖、需求、架構與 Runtime (AIA-00 ~ AIA-04)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-00 總體架構與演進藍圖** | 6 層架構、R1~R6 能力模型。 | [`docs/development/roadmap.md`](file:///home/wade/src/github/aia/docs/development/roadmap.md), [`app/app.go`](file:///home/wade/src/github/aia/go/internal/app/app.go) | 生產就緒 | 7 個 Master Phase 與 M0~M8 里程碑架構已落實於代碼結構。 |
| **AIA-01 產品需求規格** | Local-First、Single User、Task 狀態機。 | [`domain/task/`](file:///home/wade/src/github/aia/go/internal/domain/task), [`storage/sqlite/`](file:///home/wade/src/github/aia/go/internal/storage/sqlite) | 生產就緒 | 本機單人 SQLite WAL 模式，完整 Task 狀態機與依賴 DAG。 |
| **AIA-02 系統架構規格** | Runtime、Autonomy Core、Services、Agents 分層。 | [`app/app.go`](file:///home/wade/src/github/aia/go/internal/app/app.go), [`service/`](file:///home/wade/src/github/aia/go/internal/service), [`eventbus/`](file:///home/wade/src/github/aia/go/internal/eventbus) | 生產就緒 | 嚴格分層，依賴注入，EventBus 事件驅動。 |
| **AIA-03 Go Runtime 實作** | Manager 生命週期、Workers、Recovery、SQLite。 | [`worker/`](file:///home/wade/src/github/aia/go/internal/worker), [`recovery/`](file:///home/wade/src/github/aia/go/internal/recovery), [`storage/sqlite/`](file:///home/wade/src/github/aia/go/internal/storage/sqlite) | 生產就緒 | ScheduleWorker (15s)、TaskWorker (3 slots)、重啟自動恢復。 |
| **AIA-04 測試與驗收發布** | Unit/Integration 測試、Mock Provider、回歸門禁。 | [`go/tests/unit/`](file:///home/wade/src/github/aia/go/tests/unit), [`go/tests/integration/`](file:///home/wade/src/github/aia/go/tests/integration) | 生產就緒 | 163 個自動化測試全數通過（含 real agy 整合測試）。 |

---

### 2.2 領域 Agent、介面與操作手冊 (AIA-05 ~ AIA-07)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-05 Investment Agent** | 虛擬資金、營業日自主觀察、Brier 校準。 | [`service/investment/`](file:///home/wade/src/github/aia/go/internal/service/investment), [`collector/`](file:///home/wade/src/github/aia/go/internal/collector), [`experiment/`](file:///home/wade/src/github/aia/go/internal/experiment) | 生產就緒 | 11 張投資專用表，台股/總經採集，Brier/LogLoss 雙軌校準。 |
| **AIA-06 Web UI & 介面** | Port 7777, Activity Monitor, 審批與監控。 | [`web/host/`](file:///home/wade/src/github/aia/go/internal/web/host), [`web/console/`](file:///home/wade/src/github/aia/go/internal/web/console), [`go/web/assets/dist/`](file:///home/wade/src/github/aia/go/web/assets/dist) | 核心就緒 (75%) | 後端 REST API 齊全；前端目前有 9 個 Tab，缺 Goal/Cycle/Problem 專屬視圖與 Emergency Stop 按鈕。 |
| **AIA-07 使用者操作手冊** | CLI 指令集、安裝配置、維護與故障排除。 | [`cli/`](file:///home/wade/src/github/aia/go/internal/cli), [`docs/operations/`](file:///home/wade/src/github/aia/docs/operations) | 生產就緒 | Cobra CLI 完整支援 agent, task, cycle, goal, learning, brief, clarify, problem。 |

---

### 2.3 自主核心與學習四大支柱 (AIA-08 ~ AIA-11)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-08 Autonomy Core** | 11 步驟自主閉環、共用決策架構、No-Action 判定。 | [`autonomy/cycle/`](file:///home/wade/src/github/aia/go/internal/autonomy/cycle), [`autonomy/deliberation/`](file:///home/wade/src/github/aia/go/internal/autonomy/deliberation), [`autonomy/observation/`](file:///home/wade/src/github/aia/go/internal/autonomy/observation) | 生產就緒 | 11 步 Coordinator，EV 權衡矩陣，支援 `ACT`、`DEFER`、`MONITOR`、`ASK_HUMAN`、`NO_ACTION`。 |
| **AIA-09 主動性與目標演化** | 候選工作探索、Goal 樹、目標演化提案。 | [`autonomy/initiative/`](file:///home/wade/src/github/aia/go/internal/autonomy/initiative), [`service/goal/`](file:///home/wade/src/github/aia/go/internal/service/goal), [`domain/goal/`](file:///home/wade/src/github/aia/go/internal/domain/goal) | 生產就緒 | 候選發現去重，3-way Goal 演化，Anti-Goalpost-Moving 門禁。 |
| **AIA-10 連續性與專案管理** | Append-only Event History、Context 重建、快照。 | [`continuity/`](file:///home/wade/src/github/aia/go/internal/continuity), [`storage/sqlite/migrations/event_history*`](file:///home/wade/src/github/aia/go/internal/storage/sqlite/migrations/event_history_and_provenance.go) | 生產就緒 | SHA-256 雜湊鏈事件流，ContextReconstructor，專案快照還原。 |
| **AIA-11 自我修正與持續學習** | 預期差距比對、RCA 分析、3-Tier 修正、L0~L3 學習。 | [`autonomy/learning/`](file:///home/wade/src/github/aia/go/internal/autonomy/learning), [`domain/learning/`](file:///home/wade/src/github/aia/go/internal/domain/learning) | 生產就緒 | 自動落差評估，可驗證 RCA 假說，7 大比對欄位，負向約束抑制。 |

---

### 2.4 跨領域協作、實驗與自我修改 (AIA-12 ~ AIA-15)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-12 跨領域協作與委派** | Agent 間任務委派合約、交付物驗收、跨專案協同。 | [`service/delegation/`](file:///home/wade/src/github/aia/go/internal/service/delegation), [`domain/delegation/`](file:///home/wade/src/github/aia/go/internal/domain/delegation) | 生產就緒 | `delegation_requests` 狀態機，自動工單生成與驗收檢查。 |
| **AIA-13 自主性評估基準** | 主動率、介入率、連續性復原準確率等獨立指標。 | [`experiment/controller.go`](file:///home/wade/src/github/aia/go/internal/experiment/controller.go), [`service/investment/`](file:///home/wade/src/github/aia/go/internal/service/investment) | 待補強 (40%) | 目前具備投資 Brier/LogLoss 與世代對比，尚缺通用的全系統自主性指標模組。 |
| **AIA-14 安全代碼自我修改** | Level A/B/C 三級隔離、Git Worktree 沙盒、自修流水線。 | [`service/selfmod/`](file:///home/wade/src/github/aia/go/internal/service/selfmod), [`domain/selfmod/`](file:///home/wade/src/github/aia/go/internal/domain/selfmod), [`autonomy/self_evolution_engine.go`](file:///home/wade/src/github/aia/go/internal/autonomy/self_evolution_engine.go) | 核心就緒 (80%) | 三級分類、沙盒隔離與回滾均已完備；現況僅執行單一回呼測試，缺多級管線重試迴圈。 |
| **AIA-15 自主實驗生命週期** | 世代世代實驗 (`G1` vs `G2`)、5 體制判決、長期存檔。 | [`experiment/`](file:///home/wade/src/github/aia/go/internal/experiment), [`domain/experiment/`](file:///home/wade/src/github/aia/go/internal/domain/experiment) | 生產就緒 | 凍結 G1 規則基準與 SQLite 觸發器防篡改，DiD 對照評估。 |

---

### 2.5 知識、治理、參考架構、維運與問題企劃 (AIA-16 ~ AIA-22)

| 規格編號與名稱 | 規格核心要求 | 實際程式碼位置 (`go/internal/`) | 實作成熟度 | 現況與注意事項 |
|---|---|---|:---:|---|
| **AIA-16 證據與知識完整性** | Provenance 來源評估、Source/Evidence/Claim 鏈條。 | [`service/knowledge/`](file:///home/wade/src/github/aia/go/internal/service/knowledge), [`domain/knowledge/`](file:///home/wade/src/github/aia/go/internal/domain/knowledge) | 生產就緒 | 5D 來源可信度評分，Claim 修訂歷史，決策證據鏈回溯。 |
| **AIA-17 運作治理與監督** | G0~G3 治理、審批閘門、1 秒緊急停機。 | [`service/governance/`](file:///home/wade/src/github/aia/go/internal/service/governance), [`service/approval/`](file:///home/wade/src/github/aia/go/internal/service/approval), [`policy/`](file:///home/wade/src/github/aia/go/internal/policy) | 生產就緒 | 受控 Emergency Stop，防代理人自我審批，預算硬約束。 |
| **AIA-18 端到端參考架構** | 11 步驟自主閉環整合、控制流與資料流規範。 | [`autonomy/`](file:///home/wade/src/github/aia/go/internal/autonomy), [`app/`](file:///home/wade/src/github/aia/go/internal/app) | 生產就緒 | 跨領域共用自主閉環，PiT AsOf 時鐘防未來資訊洩漏。 |
| **AIA-19 部署維運與可觀測性** | systemd --user 常駐、Activity Monitor、日誌管理。 | [`scripts/service.sh`](file:///home/wade/src/github/aia/go/scripts/service.sh), [`worker/`](file:///home/wade/src/github/aia/go/internal/worker) | 核心就緒 (65%) | 常駐服務與監控運行良好；備份還原與 `retain_days` 日誌清理待實裝。 |
| **AIA-20 創作規範與小說專案** | 創作大綱、Master Bible、人物譜系與章節管線。 | [`service/writing/`](file:///home/wade/src/github/aia/go/internal/service/writing), [`domain/writing/`](file:///home/wade/src/github/aia/go/internal/domain/writing) | 生產就緒 | 支援小說專案連續性、世界觀設定與一致性檢查。 |
| **AIA-21 專案綱要與釐清管理** | Universal Project Brief、歷史版本追蹤、人機互動釐清。 | [`service/brief/`](file:///home/wade/src/github/aia/go/internal/service/brief), [`service/clarification/`](file:///home/wade/src/github/aia/go/internal/service/clarification), [`domain/brief/`](file:///home/wade/src/github/aia/go/internal/domain/brief) | 生產就緒 | 跨領域通用 Brief Schema，欄位級不可變修訂歷史，釐清請求工作流。 |
| **AIA-22 複雜問題分析與規劃** | 形式化問題建模、多階段診斷、可執行計畫 DAG 與變更。 | [`service/problem/`](file:///home/wade/src/github/aia/go/internal/service/problem), [`service/plan/`](file:///home/wade/src/github/aia/go/internal/service/plan), [`cli/problem.go`](file:///home/wade/src/github/aia/go/internal/cli/problem.go) | 核心就緒 (85%) | 獨立服務、儲存庫、CLI 與 REST 齊備；待串接進 Autonomy 主循環觸發。 |

---

## 3. 現有程式庫關鍵目錄結構

```text
go/
├── cmd/aia/main.go                      # 頂層 CLI 入口
├── internal/
│   ├── app/app.go                       # 依賴注入與服務裝配
│   ├── autonomy/                        # 11 步驟自主閉環核心
│   │   ├── cycle/                       # 主調度協調器
│   │   ├── deliberation/                # EV 權衡與決策
│   │   ├── initiative/                  # 候選工作探索
│   │   ├── learning/                    # 自我修正與 RCA
│   │   ├── observation/                 # 多領域觀察蒐集
│   │   ├── recall/                      # 記憶檢索與約束
│   │   └── self_evolution_engine.go     # 自主演化推進器
│   ├── cli/                             # 所有子命令 (含 brief, clarify, problem)
│   ├── config/                          # 完整 YAML 配置載入器
│   ├── continuity/                      # 脈絡重建與事件歷史還原
│   ├── domain/                          # 領域模型實體 (含 brief, clarification, problem, plan)
│   ├── experiment/                      # 世代實驗控制器 (G1 vs G2)
│   ├── policy/                          # 白名單與預算門禁
│   ├── provider/                        # Antigravity 沙盒適配器
│   ├── service/                         # 各業務服務 (governance, selfmod, brief, clarification, problem, plan)
│   ├── storage/sqlite/                  # SQLite WAL 驅動與 Migrations 001~018
│   ├── web/                             # Web Console API 與靜態資源
│   └── worker/                          # 背景工作排程與受控停止
└── tests/                               # 163 個自動化測試套件
```
