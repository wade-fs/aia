# AIA 架構與程式碼映射對照表 (Architecture-to-Code Mapping)

**文件代號：** AIA-ARCH-MAP-01  
**歷史盤點基準點：** Commit [`7597902`](file:///home/wade/src/github/aia) (`feat: add AIA Activity Monitor for CLI and Web Console`)  
**對照規格：** [docs/specifications/](file:///home/wade/src/github/aia/docs/specifications) (AIA-00 ～ AIA-19)  
**文件狀態：** Phase 0 Historical Baseline Audit Reference  
**最後更新：** 2026-11-01  

> [!NOTE]
> **歷史快照說明**：本文件完整保留 **Phase 0（Commit `7597902`）啟動前的歷史差距盤點基準**，用以對照既有資產保護邊界。目前系統已完成 **Phase 0 ～ Phase 6（含 Phase 3.5 加固、Phase 5 多領域 MVA、Phase 6 安全自我修改，Migrations `001`～`014`）**，最新模組與資料表實作狀態請參閱 [implementation-status.md](file:///home/wade/src/github/aia/docs/development/implementation-status.md) 與 [roadmap.md](file:///home/wade/src/github/aia/docs/development/roadmap.md)。  

---

## 1. 映射總覽與成熟度評估 (Maturity Overview)

本文件以 commit `7597902` 為絕對基準，逐一盤點目前倉庫中的 Go 原始碼、資料表、CLI、Web API，並精確對照 20 份規格文件的具體要求。

```text
┌────────────────────────────────────────────────────────────────────────┐
│                   AIA-00 ~ AIA-19 程式碼對照成熟度總覽                 │
├──────────────────────┬─────────────┬───────────────────────────────────┤
│ 成熟度分類           │ 規格數量    │ 代表規格編號                      │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 生產就緒 (Ready)     │ 2 份 (10%)  │ AIA-01 (基礎部分), AIA-19 (運維)  │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 雛形就緒 (Prototype) │ 5 份 (25%)  │ AIA-03, AIA-04, AIA-05, AIA-06,   │
│                      │             │ AIA-07                            │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 部分具備 (Partial)   │ 3 份 (15%)  │ AIA-02, AIA-10, AIA-17            │
├──────────────────────┼─────────────┼───────────────────────────────────┤
│ 待實作 (Missing)     │ 10 份 (50%) │ AIA-00, AIA-08, AIA-09, AIA-11,   │
│                      │             │ AIA-12, AIA-13, AIA-14, AIA-15,   │
│                      │             │ AIA-16, AIA-18                    │
└──────────────────────┴─────────────┴───────────────────────────────────┘
```

---

## 2. 規格逐項映射與差距清單 (Detailed Spec Mapping Matrix)

### 2.1 AIA-00 至 AIA-03：總體藍圖、需求、系統與 Runtime 實作

| 規格編號與名稱 | 規格核心要求 | 現有實作位置 (Commit 7597902) | 實作成熟度 | 具體缺口與待實作重點 |
|---|---|---|:---:|---|
| **AIA-00 總體架構與演進藍圖** | 系統定位、6層架構、R1~R6能力演進模型。 | [docs/development/roadmap.md](file:///home/wade/src/github/aia/docs/development/roadmap.md) | 待實作 | 代碼層缺少共用 Autonomy Core 頂層容器；目前 PersonalAgent 僅為薄轉發。 |
| **AIA-01 產品需求規格** | Local-First、Single User、Task 狀態機、四項自主能力。 | [`go/internal/domain/task`](file:///home/wade/src/github/aia/go/internal/domain/task), [`storage/sqlite`](file:///home/wade/src/github/aia/go/internal/storage/sqlite) | 雛形就緒 | 基礎單人本機與 Task 狀態機完備；缺少自主性評估與長期目標演化。 |
| **AIA-02 系統架構規格** | Runtime、Autonomy Core、Core Services、Domain Agents 分層。 | [`go/internal/app/app.go`](file:///home/wade/src/github/aia/go/internal/app/app.go), [`internal/service/`](file:///home/wade/src/github/aia/go/internal/service) | 部分具備 | 服務皆已接上 EventBus 與 SQLite；但 Autonomy Core 尚未抽離為獨立層。 |
| **AIA-03 Go Runtime 實作** | Manager 生命週期、Schedule/Task Worker、Recovery、SQLite WAL。 | [`go/internal/worker/`](file:///home/wade/src/github/aia/go/internal/worker), [`internal/recovery/`](file:///home/wade/src/github/aia/go/internal/recovery) | 雛形就緒 (65%) | Worker 並行控制與重啟恢復良好；缺 GoalService、EvidenceRepo 與受控停止。 |

---

### 2.2 AIA-04 至 AIA-07：測試、投資 Agent、Web Console 與手冊

| 規格編號與名稱 | 規格核心要求 | 現有實作位置 (Commit 7597902) | 實作成熟度 | 具體缺口與待實作重點 |
|---|---|---|:---:|---|
| **AIA-04 測試與驗收發布** | Unit/Integration 測試、Mock Provider、回歸測試門禁。 | [`go/tests/unit/`](file:///home/wade/src/github/aia/go/tests/unit), [`tests/integration/`](file:///home/wade/src/github/aia/go/tests/integration) | 雛形就緒 (60%) | 現有測試覆蓋率高（全綠）；缺自主行為驗證、連續性故障注入測試。 |
| **AIA-05 Investment Agent** | 台股 10 萬虛擬資金、營業日自主觀察、Brier 校準。 | [`go/internal/service/investment/`](file:///home/wade/src/github/aia/go/internal/service/investment), [`collector/`](file:///home/wade/src/github/aia/go/internal/collector) | 雛形就緒 (60%) | TWSE/MOPS/FRED 採集、決策記錄完備；缺部位持股動態記帳與每月自主覆盤。 |
| **AIA-06 Web UI & 介面** | Port 7777, OTT Token, Cookie 認證, Activity Monitor. | [`go/internal/web/host/`](file:///home/wade/src/github/aia/go/internal/web/host), [`web/console/`](file:///home/wade/src/github/aia/go/internal/web/console) | 雛形就緒 (60%) | 儀表板、Token 安全登入與即時監控完備；缺 Goal 檢視與 Approval 審批卡片。 |
| **AIA-07 使用者操作手冊** | CLI 指令集、安裝配置、維護與故障排除指南。 | [`go/internal/cli/`](file:///home/wade/src/github/aia/go/internal/cli), [`docs/operations/`](file:///home/wade/src/github/aia/docs/operations) | 雛形就緒 | 現有 CLI 指令豐富；文檔範例需隨 v2.0 指令演進持續更新。 |

---

### 2.3 AIA-08 至 AIA-11：自主核心四大支柱

| 規格編號與名稱 | 規格核心要求 | 現有實作位置 (Commit 7597902) | 實作成熟度 | 具體缺口與待實作重點 |
|---|---|---|:---:|---|
| **AIA-08 Autonomy Core** | 11 步驟自主閉環、共用決策架構、無行動 (No-Action) 判斷。 | 無獨立模組（散落在 PersonalAgent） | 待實作 (10%) | 僅有個人助理 prompt 分流；尚未建立 `internal/autonomy/` 統一閉環。 |
| **AIA-09 主動性與目標演化** | 候選工作主動發現、Goal 實體樹、目標修訂提案。 | 無獨立模組 | 待實作 (5%) | 缺少 `goals` 資料表、目標演化狀態機與候選問題探索器。 |
| **AIA-10 連續性與專案管理** | Append-only Event History、Context 重建、未決事項 (Pending)。 | [`go/internal/domain/project/`](file:///home/wade/src/github/aia/go/internal/domain/project), [`service/project/`](file:///home/wade/src/github/aia/go/internal/service/project) | 部分具備 (30%) | 現有 project 僅記錄路徑；缺 Context Reconstruction 引擎與 Event 溯源。 |
| **AIA-11 自我修正與持續學習** | 預期 vs 實際比對、RCA 分析、六層修正、學習寫入記憶。 | [`go/tests/unit/calibration_test.go`](file:///home/wade/src/github/aia/go/tests/unit/calibration_test.go) | 待實作 (15%) | 僅有投資 Brier 評分計算；缺通用的成效差距比對與根本原因分析器。 |

---

### 2.4 AIA-12 至 AIA-15：跨領域、評估、自修與實驗生命週期

| 規格編號與名稱 | 規格核心要求 | 現有實作位置 (Commit 7597902) | 實作成熟度 | 具體缺口與待實作重點 |
|---|---|---|:---:|---|
| **AIA-12 跨領域協作與委派** | Agent 任務委派協議、跨專案交付物驗收、共用能力發現。 | 無獨立模組 | 待實作 (0%) | 尚未實作 Agent 間委派請求 (`DelegationRequest`) 與成果交接機制。 |
| **AIA-13 自主性評估基準** | 主動率、介入率、修正成功率等自主性基準量測。 | 無獨立模組 | 待實作 (5%) | 尚未將自主行為度量從業務成果（如投資損益）中拆分量測。 |
| **AIA-14 安全代碼自我修改** | Level A/B/C 修改邊界、沙盒工作區、自修安全流水線。 | [`go/internal/workspace/`](file:///home/wade/src/github/aia/go/internal/workspace) | 待實作 (10%) | 僅有工作區目錄管理；缺隔離沙盒分支、自動自測與防污染回滾機制。 |
| **AIA-15 自主實驗生命週期** | Experiment 實體、Hypothesis 假設、Baseline、長期存檔。 | 無獨立模組 | 待實作 (5%) | 缺少 `experiments` 資料表與長週期科研實驗狀態機。 |

---

### 2.5 AIA-16 至 AIA-19：知識完整性、運作治理、參考架構與部署運維

| 規格編號與名稱 | 規格核心要求 | 現有實作位置 (Commit 7597902) | 實作成熟度 | 具體缺口與待實作重點 |
|---|---|---|:---:|---|
| **AIA-16 證據與知識完整性** | Provenance 來源追溯、Source/Evidence/Claim 鏈條。 | [`go/internal/service/memory/`](file:///home/wade/src/github/aia/go/internal/service/memory), [`repository/memory.go`](file:///home/wade/src/github/aia/go/internal/storage/sqlite/repository) | 部分具備 (20%) | 現有記憶僅有分類文字；缺客觀證據關聯、來源標註與推論/假設區分。 |
| **AIA-17 運作治理與監督** | G0~G3 分層、Policy/Approval/Permission 分立、受控停機。 | [`go/internal/policy/evaluator.go`](file:///home/wade/src/github/aia/go/internal/policy/evaluator.go) | 部分具備 (30%) | 現有 policy 僅有基礎白名單判斷；缺審批工作流與 1 秒停止派發開關。 |
| **AIA-18 端到端參考架構** | 11 步驟自主閉環整合、全系統控制流與資料流權威指引。 | [docs/specifications/cross-spec-consistency-audit.md](file:///home/wade/src/github/aia/docs/specifications/cross-spec-consistency-audit.md) | 待實作 (設計就緒) | 本規格作為後續架構組裝藍圖。 |
| **AIA-19 部署維運與可觀測性** | systemd --user 常駐、Activity Monitor、日誌完整性。 | [`go/scripts/service.sh`](file:///home/wade/src/github/aia/go/scripts/service.sh), [`internal/worker/worker.go`](file:///home/wade/src/github/aia/go/internal/worker/worker.go) | 生產就緒 (80%) | systemd 常駐、日誌輪轉、即時 Activity Monitor 快照皆運作良好。 |

---

## 3. 現有程式庫關鍵目錄與對照目標

```text
現有目錄 (Commit 7597902)               對應目標架構 (Target Architecture)
├── go/cmd/aia/                          ──► 擴充 aia goal, aia approval, aia stop --emergency
├── go/internal/
│   ├── app/app.go                       ──► 維持 Bootstrap 裝配，注入 Goal/Evidence/Policy 服務
│   ├── storage/sqlite/                  ──► 保留 WAL 驅動，增量引入 004/005 Migration
│   ├── worker/                          ──► 保留 ScheduleWorker/TaskWorker，注入受控 Emergency Stop
│   ├── orchestrator/                    ──► PersonalAgent 重構為呼叫 internal/autonomy/ 的外觀
│   ├── autonomy/ (待建)                 ──► 【新增】實作 11 步驟閉環、主動性探索與 No-Action 判定
│   ├── domain/                          ──► 保留現有實體，新增 domain/goal, domain/evidence
│   ├── continuity/ (待建)               ──► 【新增】實作 ContextReconstructor 與 Event 歷史還原
│   ├── evidence/ (待建)                 ──► 【新增】實作 EvidenceService 與 Claim 檢驗
│   ├── policy/                          ──► 升級為 PolicyEngine，分離 Approval 與 Permission
│   ├── collector/                       ──► 保留 TWSE/MOPS/FRED，供 Investment Pilot 調用
│   └── web/host/                        ──► 保留 Port 7777 / Token，增加 Approval REST API
```

本對照表確立了既有程式碼與目標規範的精確接口，後續開發嚴禁擅自更動標記為「保留」的基礎資產。
