# AIA Volume 2: Cognitive Loop, Memory & Planning

**Document ID:** AIA-VOL-02  
**Title:** 認知循環、記憶與規劃規範 (Cognitive Loop, Memory & Planning Specification)  
**Status:** Normative Consolidated Baseline  
**Consolidates:** AIA-08, AIA-09, AIA-10, AIA-16, AIA-22, memory.md  

---

## 1. 11 步共享自治循環 (11-Step Shared Autonomy Loop)

依據原 AIA-08 規範，所有 Domain Agent 均共享同一套嚴謹的 11 步自治認知循環（`internal/autonomy/cycle/`）：

```text
 1. Observe   ➔ 收集環境變更、工作區差異、外部數據與待辦工單
 2. Recall    ➔ 依據 L1~L5 記憶架構與 AsOf 時鐘檢索關聯知識與歷史脈絡
 3. Assess    ➔ 評估目前專案狀態與目標達成落差 (Goal Gap)
 4. Deliberate➔ 多選項推演，計算預期效益值 (Expected Value, EV)
 5. Decide    ➔ 產出結構化裁定 (ACT, DEFER, MONITOR, ASK_HUMAN, NO_ACTION)
 6. Act       ➔ 派遣工作項目或調用沙盒執行器
 7. Verify    ➔ 獨立驗證執行成果 (絕不單憑 Exit Code 0)
 8. Learn     ➔ 比對預期與實際成果，若有落差觸發根因分析 (RCA)
 9. Reconcile ➔ 解決知識衝突，更新認知主張版本
10. Snapshot  ➔ 保存當前乾淨狀態快照，更新 Hash-Chained 事件流
11. Evolve    ➔ 檢討長期目標是否達成或需進入下一階段 (Goal Evolution)
```

**理性的 NO_ACTION 準則**：當環境處於穩定狀態或缺乏行動效益時，系統必須做出審計可追溯的 `NO_ACTION` 決策，杜絕無意義的 Token 浪費。

---

## 2. 主動性發現與目標治理演化 (Initiative & Goal Evolution)

依據原 AIA-09 規範：
1. **主動性發現器（Discoverer）**：無須人類指示，系統主動掃描工作區異常、過期未決事項、知識矛盾與代碼缺陷，生成 `CandidateInitiative`。
2. **目標三態演化分流（Goal Evolution Taxonomy）**：
   - `PERSISTENCE`：目標未達成，持續推進既有路線。
   - `OPTIMIZATION`：在不更動核心指標的前提下，優化執行步驟與效率。
   - `REVISION`：外部環境或核心假設打破時，正式提出目標修訂提案。
3. **防移動球門鐵律（Anti-Goalpost-Moving）**：嚴禁 Agent 在面臨挫折時自行調降 `STRATEGIC` 目標的驗收標準；任何降低標準的修訂必須經過人類審批。

---

## 3. 跨會話時間連續性 (Temporal Continuity)

依據原 AIA-10 規範：
- **第一事實來源**：`event_history` 資料表採用 SHA-256 Hash Chain 與 SQLite `ABORT` 觸發器，任何歷史事件寫入後不可刪除與修改。
- **脈絡重構（Context Reconstruction）**：服務重啟或崩潰復原後，`ContextReconstructor` 從事件流精確還原 Project 當前狀態、關聯 Goals、待決問題與環境快照，保證工作接續進行。

---

## 4. 知識溯源與一致性記憶 (Data Provenance & Consistent Memory)

依據原 AIA-16 與 `memory.md` 規範：

### 4.1 四種記憶劃分
- **情節記憶 (Episodic)**：過去發生的客觀事件、使用者決策與結果。
- **語意記憶 (Semantic)**：已驗證的事實主張與知識版本。
- **程序記憶 (Procedural)**：方法論、驗收流程與技能。
- **工作記憶 (Working)**：當前 Session 正在處理的短暫狀態。

### 4.2 五層權威體系 (Authority Tiers)
- `L1 Event History`：最高權威事實，不可改寫。
- `L2 Evidence & Provenance`：來源可靠度與客觀證據。
  - **5D 來源評估**：Authority、Directness、Timeliness、Independence、Reproducibility。
- `L3 Knowledge & Memory`：整理後的認知主張，保留衝突（`ConflictedState`），版本化修訂。
- `L4 Context & Project State`：目前專案工作狀態與未決問題。
- `L5 Active Context`：當前任務關聯召回的精簡上下文。

---

## 5. 複雜問題形式化建模與版本化規劃 (Complex Problem Analysis & Planning)

依據原 AIA-22 規範，處理開放式未知問題時，禁止直接盲目拆解 Task，必須遵循標準規劃閉環：

```text
 1. Intent Understanding ➔ 釐清使用者真實目標，分離目標、方法與約束
 2. Gap & Conflict Disc. ➔ 發現資料缺口 (Data Gap) 與規範衝突 (Rule Conflict)
 3. Formal Problem Model ➔ 建立包含 Fact (信心度1.0)、Assumption (<1.0) 與 Unknown (0.0) 的結構化模型
 4. Solution Exploration ➔ 提出多個具體可行替代方案並進行利弊權衡 (Trade-off Matrix)
 5. DAG Plan Generation  ➔ 生成包含明確驗收條件與有向無環依賴圖 (DAG) 的版本化 Plan
 6. Governed Materialize ➔ 經審批後具體化為 Task 工單並分派給專屬 Domain Agent
 7. Dynamic Reassessment ➔ 執行失敗或新證據打破假設時，歸因根因並演進 Model v(N+1)，標註舊計畫為 SUPERSEDED
```
