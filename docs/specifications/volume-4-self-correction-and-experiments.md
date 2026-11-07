# AIA Volume 4: Self-Correction, Experiments & Benchmarks

**Document ID:** AIA-VOL-04  
**Title:** 自我修正、實驗與自主評估規範 (Self-Correction, Experiments & Benchmarks Specification)  
**Status:** Normative Consolidated Baseline  
**Consolidates:** AIA-04, AIA-11, AIA-13, AIA-15  

---

## 1. 可驗證自我修正機制 (Verifiable Self-Correction Engine)

依據原 AIA-11 規範，AIA 建立在**科學對照與可證偽性**之上，絕非將所有執行失敗歸咎為單一原因或進行盲目無休止的重試。

### 1.1 結果落差量測 (Outcome Evaluation)
- 每次決策或行動結束後，系統比對「預期成果 (`expected_outcome`)」與「實際成果 (`actual_outcome`)」。
- 量化計算偏離分數（Discrepancy Score）；當落差顯著時，觸發結構化異常標記。

### 1.2 可證偽根因分析 (Testable RCA)
系統嚴格區分失敗源頭（AIA-22 §8.3 / AIA-11 §3）：
- `MODEL_ERROR`：問題模型本身歸因錯誤。
- `ASSUMPTION_INVALID`：先驗假設被客觀事實推翻。
- `DATA_SHORTAGE`：必要輸入資料不足。
- `PLAN_UNREASONABLE`：工作步驟或驗收條件設定不合理。
- `EXECUTION_FAILURE`：代碼或環境臨時故障。

### 1.3 七大強制對照欄位 (Seven Scientific Epistemic Fields)
任何持久化儲存的學習紀錄（`learning_records`），必須完整包含：
1. `baseline`：介入前的歷史基準數據。
2. `hypothesis`：可被證偽的改進假設。
3. `intervention`：具體施加的變更項目。
4. `verification`：客觀驗證結果與數據。
5. `regression`：回歸測試與無副作用檢驗。
6. `followup`：長週期延遲成效追蹤。
7. `confidence`：信心度評估（若僅為單次前後比對，信心度強制封頂 `<= 0.45`）。

### 1.4 四大成熟度分級 (Learning Maturity Levels)
- `L0_RECORDED`：初次觀測記錄，未經重複驗證。
- `L1_APPLIED`：已在具體任務中應用並產出初步改善。
- `L2_VERIFIED`：經獨立測試套件與多輪驗證，證實改善有效。
- `L3_GENERALIZED`：跨專案、跨領域經長週期驗證有效，升級為通用策略。

### 1.5 負向約束抑制 (Negative Constraint Suppression)
若一項修正經評估為無效或導致回歸退步（`REGRESSED` / `INEFFECTIVE`），系統自動將其特徵寫入負向約束表，後續自治循環進行推演時自動壓制該失敗模式，防止重複犯錯。

---

## 2. 長週期跨世代對照實驗 (Longitudinal Autonomous Experiments)

依據原 AIA-15 規範：

### 2.1 11 狀態完整實驗生命週期
所有長週期實驗（`experiments`）嚴格依據 11 狀態機推進：
`PROPOSED` ➔ `PENDING_APPROVAL` ➔ `READY` ➔ `RUNNING` ➔ `PAUSED` ➔ `EVALUATING` ➔ `NEEDS_REVISION` ➔ `COMPLETED` ➔ `STOPPED` ➔ `FAILED` ➔ `ARCHIVED`。

### 2.2 凍結 G1 基準組保護（Frozen G1 Baseline Protection）
- 在投資與決策實驗中，設定 `G1` 凍結規則基準組與 `G2 -> G2.1 -> G2.2` LLM 實驗組。
- 資料庫層面附加 SQLite 防竄改 Trigger（`trg_exp_gen_freeze_update`, `trg_exp_gen_freeze_delete`），硬性阻斷對 `G1` 基準組之任何異動，確保真實長期 Difference-in-Differences 雙重差分對照有效性。

---

## 3. 可計算自主能力指標 (Autonomy Benchmarks M0 ～ M10)

依據原 AIA-13 與 AIA-04 規範，AIA 以可計算之行為指標度量成熟度，而非程式行數：

| 里程碑 | 指標名稱 | 驗收核心能力標準 |
|---|---|---|
| **M0** | **Persistent Runtime** | 常駐 Daemon 零崩潰運行、併發 Worker、SQLite WAL 持久化 |
| **M1** | **Temporal Continuity** | 服務重啟後專案、目標與未決事項 100% 精確還原 |
| **M2** | **Trustworthy Decision** | 每項推論與決策均具備來源證據鏈（5D Provenance Score） |
| **M3** | **Initiative & Rationality** | 主動發現有價值工作；理性 `NO_ACTION` 決策準確率 `>= 70%` (N>=10) |
| **M4** | **Autonomous Closed Loop** | 11 步自治循環在預算與 Point-in-Time 零前視偏差時鐘下流暢運作 |
| **M5** | **Verifiable Self-Correction**| 落差偵測、可證偽 RCA、3-Tier 配置與負向模式抑制 |
| **M6** | **Governed Goal Evolution** | 目標演化三態分流，落實 Anti-Goalpost-Moving 治理 |
| **M7** | **Domain Experiments** | 跨領域長週期實驗、NT$100,000 虛擬帳簿與跨 Agent 契約委派 |
| **M8** | **Safe Self-Modification** | 沙盒代碼自修、V1~V6 多層驗證管線與零殘留回滾 |
| **M9** | **Universal Intent & Brief** | 通用 Brief 欄位歷史追蹤與阻斷式 Clarification 提問互動 |
| **M10**| **Complex Problem Planning**| 開放問題建模、DAG 工作項目拆解與動態重評演進 |
