# AIA Volume 6: Domain Specializations & Delegation

**Document ID:** AIA-VOL-06  
**Title:** 領域專業代理與跨領域委派規範 (Domain Specializations & Delegation Specification)  
**Status:** Normative Consolidated Baseline  
**Consolidates:** AIA-05, AIA-12, AIA-20  

---

## 1. 領域路由架構 (Domain Routing & Specialization)

AIA 採用「共用自治核心 + 領域特化適配（Shared Core, Domain Adapters）」架構。所有領域專案均受 `projects.domain_type`（Migration 013）資料庫欄位驅動路由，動態掛載專屬的 Observer、Pilot 與驗證器：

```text
                 ┌────────────────────────────────┐
                 │ 11-Step Shared Autonomy Core   │
                 └──────────────┬─────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
 ┌──────────────┐        ┌──────────────┐        ┌──────────────┐
 │  Investment  │        │    Coding    │        │   Writing    │
 │  Pilot Spec  │        │  Pilot Spec  │        │  Pilot Spec  │
 └──────────────┘        └──────────────┘        └──────────────┘
```

---

## 2. 量化投資領域規範 (Investment Pilot Specification)

依據原 AIA-05 規範：
- **虛擬資產隔離記帳**：每組投資實驗配置獨立的 NT$100,000 虛擬帳簿，嚴禁串接真實券商下單。
- **無前視偏差歷史回放（Point-in-Time AsOf Clock）**：支援時間旅行回放模式，嚴格封鎖未來數據洩漏（Zero Look-Ahead Bias），確保決策只使用當下已知資訊。
- **雙重機率校準（Dual Calibration）**：將客觀事件發生機率（Brier Score / LogLoss）與主觀決策信心度（Decision Confidence）嚴格分開校準，杜絕模型過度自信。

---

## 3. 代碼工程領域規範 (Coding Pilot Specification)

依據原 AIA-15 §24 規範：
- **Git Worktree 實體隔離**：任何代碼檢查、重構與測試修復必須在隔離的臨時工作樹中執行，絕不碰觸主分支未提交之成果。
- **代碼缺陷健康檢測（`CodeHealthObserver`）**：主動掃描編譯錯誤、單元測試失敗與靜態程式碼違規。
- **客觀 L2 轉綠驗收（L2 Deterministic Verification）**：拒絕單純依賴 Exit Code 0，強制要求針對修改點執行目標單元測試與回歸套件，確認測試由紅轉綠。

---

## 4. 創作小說領域規範 (Writing Pilot Specification)

依據原 AIA-20 與 AIA-15 §23 規範：
- **大篇幅專案理解**：具備自主解析真實 6 卷 540 章規模小說目錄結構、90+ 人物生卒年約束與知識分層（`characters.md`, `bible_v1.md`）之能力。
- **三層世界觀知識分級**：
  - `FACT_HISTORY`：不可違反之客觀歷史事實。
  - `PLAUSIBLE_INFERENCE`：情節合理推演。
  - `FICTION`：創作虛構設定。
- **死後登場與世界觀衝突守衛（Post-Mortem Appearance Guard）**：理智區分文藝虛構與硬性年表衝突，主動攔截人物於死亡年份後在現實場景登場的矛盾情節。

---

## 5. 跨領域代理委派契約 (Cross-Domain Delegation Protocol)

依據原 AIA-12 規範，當特定領域 Agent 需要其他專業支援時（如 Writing Agent 委託 Coding Agent 撰寫年表校驗腳本，或 Personal Agent 委派 Investment Agent 進行宏觀分析），依據標準委派契約執行：

### 5.1 委派生命週期狀態機
`PENDING` ➔ `IN_PROGRESS` ➔ `DELIVERED` ➔ `VERIFIED`

### 5.2 委派契約關鍵欄位（`delegation_requests`）
- `requester_agent_id`：發起代理。
- `target_agent_id`：承接代理。
- `source_project_id` 與 `goal_id`：目標關聯。
- `task_spec`：具體工作規格與約束。
- `acceptance_gate`：驗收檢驗條件，成果經委派方驗證通過後才轉為 `VERIFIED`。
