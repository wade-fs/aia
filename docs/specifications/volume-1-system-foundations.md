# AIA Volume 1: System Foundations & Architecture

**Document ID:** AIA-VOL-01  
**Title:** 系統架構與總體願景規範 (System Foundations & Architecture Specification)  
**Status:** Normative Consolidated Baseline  
**Consolidates:** AIA-00, AIA-01, AIA-02, AIA-18  

---

## 1. 核心哲學與定位 (Foundational Philosophy)

AIA（Autonomous Intelligence Architecture）是一個以**專案為中心（Project-Centric）**、**本地常駐（Local-First Resident Daemon）**、**受嚴格治理（Governed & Sandboxed）**的自主智能系統。

AIA 恪守以下核心差異化原則：
1. **拒絕單次問答聊天模式**：AIA 不是被動等待人類輸入 prompt 的 Chatbot，而是持續感知專案狀態、主動發現未完成事項、自主推演並推動長期目標落地的常駐系統。
2. **工作區零殘留污染（Zero Workspace Pollution）**：所有代碼修改、實驗推演與草稿驗證必須在獨立沙盒（Git Worktree / 命名空間）中進行，驗證失敗必須 100% 乾淨回滾，絕不可隨意更動使用者的原始工作環境。
3. **歷史不可篡改，理解可以版本化修正**：過去發生的事件與決策是不可改寫的歷史事實（Hash-Chained Event History）；當新證據出現時，系統修正的是對問題的認知模型與後續計畫（Model Versioning），而非偽造過去的判斷。
4. **反自我審批（Anti-Self-Approval）**：涉及核心架構修改、策略性目標變更或不可逆資源操作時，嚴格禁止 Agent 自行批准自身提案。
5. **雙軌演進模型（Dual-Track Evolution）**：
   - **基礎設施軌 (Infrastructure Track)**：Runtime ➔ Continuity ➔ Governance ➔ Autonomy Core ➔ Learning ➔ Safe Self-Evolution。
   - **自主能力驗證軌 (Autonomy Capability Track)**：Investment Pilot ➔ Closed Loop ➔ Self-Correction ➔ Multi-Domain MVA ➔ Complex Problem Decomposition。

---

## 2. 產品需求與系統邊界 (Product Requirements)

依據原 AIA-01 規範，AIA 定義為本機單使用者系統：
- **運行環境**：Linux (`systemd --user`) 與 macOS (`launchd`)。
- **資料儲存**：本機 SQLite (WAL 模式)，零外部雲端資料庫依賴，資料完全本機私有。
- **人機互動**：本機 Web Console (`127.0.0.1:7777`)，採用一次性登入憑證（One-Time Token, OTT）換發 HttpOnly Session，無遠端曝險。
- **模型調用**：透過 Antigravity CLI 本機適配器，執行環境變數脫敏（Secret Scrubbing）與沙盒隔離。

---

## 3. 六層系統架構 (Six-Layer Architecture)

依據原 AIA-02 規範，系統自底向上由六個標準層級組成：

```text
 ┌──────────────────────────────────────────────────────────────┐
 │ Layer 6: Web Console & Operational Tools (Web, CLI, OTT Auth)│
 ├──────────────────────────────────────────────────────────────┤
 │ Layer 5: Domain Pilots (Investment, Coding, Writing, Brief)  │
 ├──────────────────────────────────────────────────────────────┤
 │ Layer 4: Governance & Sandbox (Policy, Approval, Sandbox)    │
 ├──────────────────────────────────────────────────────────────┤
 │ Layer 3: Cognitive & Learning Engine (Autonomy, RCA, Plan)   │
 ├──────────────────────────────────────────────────────────────┤
 │ Layer 2: Continuity, Memory & Knowledge (Event, L1~L5 Memory)│
 ├──────────────────────────────────────────────────────────────┤
 │ Layer 1: Runtime Infrastructure (Go 1.23 Daemon, Workers, DB)│
 └──────────────────────────────────────────────────────────────┘
```

- **Layer 1: Runtime Infrastructure**：Go 1.23.2 常駐服務、Schedule/Task/AutoWake Workers、SQLite Migrations 001~019。
- **Layer 2: Continuity, Memory & Knowledge**：不可變事件流（`event_history`）、知識溯源（`sources`, `evidence`, `claims`）、L1~L5 一致性記憶與快照還原。
- **Layer 3: Cognitive & Learning Engine**：11 步自主循環協調器、深思評估矩陣、可證偽 RCA 與自我修正、開放式問題建模與 DAG 計畫。
- **Layer 4: Governance & Sandbox**：Level A/B/C 三級修改隔離、每日 Token 硬預算、Bwrap 隔離、緊急停機機制。
- **Layer 5: Domain Pilots**：多領域專案路由器、量化投資對照組、代碼健康巡檢、長篇小說設定稽核。
- **Layer 6: Interfaces**：16 分頁 Web Console、Cobra CLI 指令集、結構化阻斷式 Clarification 提問互動。

---

## 4. 端到端整合架構 (End-to-End Reference Architecture)

依據原 AIA-18 規範，全系統各元件間的整合互動遵循統一事件流驅動：
1. **感知輸入**：Workers 與 Observers 週期性或事件觸發，收集多領域事實與工作區變更。
2. **脈絡重構**：ContextReconstructor 從不可變事件流與 L1~L5 記憶召回當前專案與目標上下文。
3. **推演與決策**：Deliberator 計算預期效益矩陣（EV），產出合理決策（包含理性的 `NO_ACTION`）。
4. **受控執行**：高風險或不可逆操作進入 Approval 隊列等待人類簽核；授權操作於沙盒中執行。
5. **閉環驗證與學習**：執行結果必須通過獨立客觀驗證；若失敗則觸發根因分析（RCA）並記錄經驗，絕不盲目無限重試。
