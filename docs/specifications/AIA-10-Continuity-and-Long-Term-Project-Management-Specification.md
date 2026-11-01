# AIA-10 — Continuity & Long-term Project Management Specification

**Autonomous Intelligence Architecture**  
**Version:** 2.0  
**Document Type:** Continuity & Persistent Project Specification  
**Status:** Proposed Baseline  
**依賴文件：** AIA-00、AIA-01、AIA-02、AIA-03、AIA-08  
**後續相依文件：** AIA-09、AIA-11、AIA-12、AIA-13、AIA-15

---

# 1. 文件目的

本文件定義 AIA 如何在長時間運作、不同 Session、不同 Task 及不同 Domain Agent 之間，維持完整的工作脈絡與專案狀態。

AIA 的目標不是讓 Agent 記得所有過去的對話，而是讓系統能夠持續理解：

1. 正在追求哪些長期目標。
2. 每個目標目前處於什麼狀態。
3. 每個 Project 已經完成哪些工作。
4. 哪些工作尚未完成。
5. 過去做過哪些重要決策。
6. 當時決策的原因與依據。
7. 過去發現過哪些問題。
8. 哪些問題已經修正，哪些仍未解決。
9. 最近有哪些新資訊改變了原本的判斷。
10. 下一次恢復工作時，應該從哪裡繼續。

本文件將這種能力定義為：

**Continuity：系統在時間持續推進的過程中，維持目標、專案、工作、決策及經驗之間的連續關係。**

Continuity 不等於單純保存記憶，也不等於保持一個 Agent Session 永遠不關閉。

真正的連續性應由持久化資料、狀態轉換、歷史紀錄及脈絡重建共同完成。

# 2. 核心設計原則

## 2.1 Continuity 不依賴單一 Session

AI Agent 的 Session 可能因為以下原因結束：

- 使用者關閉對話。
- Runtime 重新啟動。
- Agent 執行時間超過限制。
- 模型服務中斷。
- Task 執行失敗。
- 系統進行更新。
- 工作轉交給其他 Agent。

因此，不應把長期工作狀態只保存在 Agent 的上下文視窗中。

Session 是一次執行過程，Project 才是長期工作單位。

## 2.2 Memory 不等於 Continuity

Memory 負責保存知識、經驗、資訊及歷史摘要。

Continuity 則負責利用這些資料，維持長期工作的狀態與關係。

例如：

Memory 可以記錄：

- 某個人物的出生年份。
- 某項程式設計決策。
- 某次投資判斷的原因。

但 Continuity 還必須知道：

- 這個人物屬於哪個 Project。
- 這項程式決策目前是否仍然有效。
- 這次投資判斷是否已經完成結果評估。
- 這些資訊會影響哪些尚未完成的工作。

因此，Memory 是 Continuity 的資料來源之一，不是完整的 Continuity 系統。

## 2.3 Project 不等於 Task

Task 是可以執行、完成、失敗或取消的工作單位。

Project 是一個具有長期目標、成果、歷史、依賴關係及未完成工作的持續性工作環境。

例如：

一個 Coding Project 可以包含數百個 Task。

一個 Writing Project 可以持續數年，包含多卷作品、人物設定、歷史校勘及大量章節。

一個 Investment Project 可以持續數年，包含市場觀察、投資決策、交易紀錄、績效評估及策略修正。

Project 不應因為某個 Task 完成就自動結束。

## 2.4 狀態與歷史必須分離

系統必須區分：

- Current State：目前狀態。
- Historical State：過去狀態。
- Decision History：過去決策。
- Event History：發生過的事件。
- Evaluation History：過去評估結果。

目前狀態可以更新，但歷史紀錄不能因為狀態改變而消失。

例如：

一項目標從 ACTIVE 變成 COMPLETED，不代表過去的目標內容、決策依據及完成過程可以被覆寫。

## 2.5 長期狀態必須能夠重建

系統不應假設某個記憶檔案或某次摘要永遠完整。

當必要資料遺失、過期或出現矛盾時，應能從原始事件、決策及成果重建必要狀態。

# 3. Continuity 的資料層次

AIA 的時間連續性分為六個層次。

| 層次 | 名稱 | 維護內容 |
|---|---|---|
| C1 | Session Continuity | 單次執行過程與中斷恢復 |
| C2 | Task Continuity | 工作進度、執行結果與重試 |
| C3 | Project Continuity | 長期專案狀態及未完成工作 |
| C4 | Goal Continuity | 長期目標及目標演化歷史 |
| C5 | Decision Continuity | 歷史決策及其影響 |
| C6 | Cross-Domain Continuity | 不同領域之間的工作與知識關聯 |

這六個層次互相依賴。

例如，一次 Coding Task 的失敗可能影響：

- Task 狀態。
- Coding Project 進度。
- 相關 Goal 的完成預期。
- 過去的技術決策。
- Writing Agent 等待的工具交付工作。

Continuity 必須能夠追蹤這些關聯，而不是只記錄 Task 失敗。

# 4. Project Management Model

## 4.1 Project 定義

Project 是 AIA 長期管理工作的主要單位。

每個 Project 至少具有：

| 欄位 | 說明 |
|---|---|
| project_id | 唯一識別碼 |
| name | 專案名稱 |
| description | 專案描述 |
| domain | 所屬領域 |
| status | 專案狀態 |
| primary_goal_id | 主要目標 |
| owner_agent_id | 主要負責 Agent |
| participating_agents | 參與 Agent |
| priority | 專案優先順序 |
| created_at | 建立時間 |
| updated_at | 最後更新時間 |
| last_activity_at | 最後活動時間 |
| next_review_at | 下次檢查時間 |
| state_version | 專案狀態版本 |

實際資料庫欄位以 AIA-03 的 Schema 設計為準。

## 4.2 Project Status

建議狀態：

| 狀態 | 意義 |
|---|---|
| DRAFT | 尚在規劃 |
| ACTIVE | 正在進行 |
| PAUSED | 暫停 |
| BLOCKED | 遇到阻礙 |
| REVIEW | 等待檢查 |
| COMPLETED | 已完成 |
| ARCHIVED | 已封存 |
| CANCELLED | 已取消 |

狀態轉換必須保留原因及時間。

例如：

ACTIVE → BLOCKED：

表示專案遇到無法自行排除的阻礙。

BLOCKED → ACTIVE：

表示阻礙已解除，專案恢復。

ACTIVE → PAUSED：

表示目前沒有必要繼續執行，或資源需要投入其他工作。

COMPLETED → ACTIVE：

若發現新問題或需要追加工作，應透過正式重新開啟流程，而不是直接覆寫歷史完成紀錄。

## 4.3 Project Scope

每個 Project 必須定義：

- 主要目標。
- 工作範圍。
- 預期成果。
- 不包含的範圍。
- 參與 Agent。
- 依賴條件。
- 資源限制。
- 驗收條件。
- 目前已知風險。

Project Scope 可以隨時間演化，但重要變更必須留下版本及決策紀錄。

# 5. Project State Management

## 5.1 Current Project State

每個 Project 必須有可供 Runtime 及 Agent 取得的目前狀態。

建議內容：

| 欄位 | 說明 |
|---|---|
| project_id | 專案識別碼 |
| state_version | 狀態版本 |
| current_phase | 目前階段 |
| progress_summary | 目前進度摘要 |
| completed_work | 已完成工作 |
| active_work | 正在執行的工作 |
| pending_work | 尚未開始的工作 |
| blocked_work | 受阻工作 |
| unresolved_issues | 尚未解決問題 |
| recent_decisions | 最近重要決策 |
| current_risks | 目前風險 |
| next_actions | 候選下一步 |
| updated_at | 更新時間 |

這些欄位是邏輯模型，不要求全部塞在同一筆資料庫紀錄中。

## 5.2 State Version

每次重要 Project State 更新都應增加版本。

例如：

```text
Project: coding-runtime

Version 12
  Runtime Worker 已建立
  Task Worker 尚未完成整合

Version 13
  Task Worker 已完成整合
  Recovery 流程需要重新檢查

Version 14
  Recovery 已驗證
  Graceful Shutdown 尚待處理
```

Agent 恢復工作時，應取得最新有效版本。

若不同工作同時更新同一個 Project，必須避免後寫入的資料直接覆蓋先前更新。

## 5.3 State 與 Event

Project State 是目前狀態的呈現。

Event History 是狀態變化的歷史。

例如：

```text
ProjectCreated
GoalLinked
TaskStarted
TaskCompleted
IssueDiscovered
DecisionRecorded
ProjectBlocked
ProjectResumed
ProjectCompleted
```

應採取事件紀錄與目前狀態相互配合的設計。

不要求一開始實作完整 Event Sourcing，但必須保留足以追溯重要狀態變更的資料。

# 6. Goal Continuity

## 6.1 Goal 與 Project 的關係

Goal 定義系統希望達成的結果。

Project 是實現 Goal 的長期工作環境。

一個 Goal 可以由多個 Project 支援。

一個 Project 也可能支援多個 Goal。

例如：

主要 Goal：

改善 AIA 的自主性與長期運作能力。

相關 Project：

- Runtime Project。
- Autonomy Core Project。
- Coding Agent Project。
- Evaluation Project。

Goal 與 Project 之間應建立明確關聯，而不是只在文字描述中互相提及。

## 6.2 Goal History

必須保存：

- Goal Created。
- Goal Activated。
- Goal Optimized。
- Goal Revised。
- Goal Paused。
- Goal Completed。
- Goal Abandoned。

每次變更都應記錄：

- 變更前內容。
- 變更後內容。
- 變更原因。
- 相關 Decision。
- 變更時間。
- 是否經過人類核准。

## 6.3 Goal Reconstruction

當 Agent 恢復工作時，不能只取得目前目標文字。

還應理解：

- 目標最初如何形成。
- 曾經做過哪些調整。
- 為什麼修改。
- 哪些限制仍然有效。
- 哪些目標已經完成。
- 哪些目標目前暫停。
- 是否存在目標衝突。

Goal Evolution 的具體判斷機制由 AIA-09 定義。

# 7. Decision Continuity

## 7.1 Decision Record

重要決策必須具有獨立識別碼及歷史紀錄。

建議資料：

| 欄位 | 說明 |
|---|---|
| decision_id | 決策識別碼 |
| project_id | 相關專案 |
| goal_id | 相關目標 |
| task_id | 相關工作 |
| domain | 所屬領域 |
| decision_type | 決策類型 |
| context_snapshot_id | 當時脈絡 |
| alternatives | 曾考慮的方案 |
| selected_option | 選擇方案 |
| rationale | 決策理由 |
| expected_outcome | 預期結果 |
| confidence | 當時信心程度 |
| created_at | 決策時間 |
| outcome_status | 後續結果狀態 |

## 7.2 Decision 不可被後見之明覆寫

例如 Investment Agent 在某個交易日根據當時資料決定持有股票。

數日後市場發生變化，Agent 重新評估後認為當時判斷不理想。

它可以新增一筆修正紀錄，但不能直接改寫當時的決策理由，假裝自己早已知道後來發生的事情。

這對長期學習及預測校準非常重要。

## 7.3 Decision Dependency

一項決策可能影響其他工作。

例如：

Coding Agent 決定改變某個資料結構。

這可能影響：

- 依賴該結構的其他模組。
- Investment Agent 使用的資料介面。
- Writing Agent 的小說管理工具。

系統必須保留重要決策的相依關係，以便後續變更時找出受影響的 Project。

# 8. Work Continuity

## 8.1 Work History

每個 Project 必須保留工作歷史。

至少包括：

- Task 建立。
- Task 派遣。
- Agent 開始執行。
- 執行過程的重要事件。
- Task 成功或失敗。
- 交付成果。
- 驗證結果。
- 後續修正。
- 相關決策。

Work History 不是單純的 Task 清單，而是專案演進的工作軌跡。

## 8.2 Pending Issues

系統必須能區分：

- 尚未處理的問題。
- 正在處理的問題。
- 已修正但尚未驗證的問題。
- 已驗證完成的問題。
- 因外部條件受阻的問題。
- 暫時不處理的問題。

每個重要問題至少要記錄：

- issue_id。
- project_id。
- 描述。
- 發現來源。
- 重要程度。
- 影響範圍。
- 目前狀態。
- 負責 Agent。
- 相關 Task。
- 預期處理方式。

## 8.3 Next Action Candidates

Project 可以保存候選下一步，但不能把它們直接視為正式 Task。

例如：

Coding Project：

- 修正 Worker 重複啟動問題。
- 改善 Shutdown 流程。
- 補充 Runtime 狀態監控。

Writing Project：

- 校勘某段歷史時間軸。
- 檢查人物年齡。
- 修正前後章節的矛盾。

Investment Project：

- 重新評估持股。
- 檢查市場重大資訊。
- 評估預測偏差。

候選工作必須經過 AIA-08 的 Deliberation 與 Decision，才能轉為正式 Task。

# 9. Context Reconstruction

## 9.1 核心目的

Context Reconstruction 是 Continuity 最重要的能力之一。

當 Agent 開始新 Session 或重新接手 Project 時，系統必須提供足以正確繼續工作的脈絡。

不應只提供最近幾段對話，也不應直接將所有歷史紀錄塞入 Context Window。

## 9.2 Context 組成

每次恢復 Project 時，應建立必要的工作脈絡：

**A. Project Identity**

- Project 名稱。
- 主要目標。
- 工作範圍。
- 負責 Agent。

**B. Current State**

- 目前階段。
- 已完成工作。
- 正在執行工作。
- 尚未完成工作。
- 阻礙及風險。

**C. Goal Context**

- 目前有效目標。
- 目標限制。
- 重要目標變更。
- 相關次要目標。

**D. Decision Context**

- 最近重要決策。
- 仍然有效的設計選擇。
- 尚未驗證的決策。
- 可能需要重新評估的決策。

**E. Memory Context**

- 相關知識。
- 過去經驗。
- 已知問題。
- 重要歷史資料。

**F. Work Context**

- 最近完成的 Task。
- 目前進行中的 Task。
- 失敗及重試紀錄。
- 下一步候選工作。

**G. Governance Context**

- 權限限制。
- 核准要求。
- 資源限制。
- 不可執行事項。

## 9.3 Context Priority

Context Resolver 必須依照工作相關性及重要性選取資料。

建議優先順序：

1. 安全與治理限制。
2. 目前有效目標。
3. Project Current State。
4. 目前工作及其依賴。
5. 重要歷史決策。
6. 尚未解決問題。
7. 相關 Memory。
8. 較舊的背景資訊。

不能只以資料最近發生的時間決定重要程度。

較早形成但仍有效的架構決策，可能比昨天的普通工作紀錄更重要。

## 9.4 Context Completeness

Context Reconstruction 必須評估資料是否完整。

如果缺少重要資料：

- 標記缺失。
- 嘗試從歷史紀錄重建。
- 必要時要求重新讀取來源文件。
- 必要時暫停高風險決策。
- 不得自行補造不存在的歷史紀錄。

## 9.5 Context Snapshot

重要工作開始前，應保存當時使用的 Context Snapshot。

用途：

- 重現決策環境。
- 比較不同時間的判斷。
- 追蹤資料變化。
- 分析錯誤原因。
- 支援後續 Evaluation。

Context Snapshot 應記錄來源及版本，不必每次完整複製所有原始資料。

# 10. Cross-Session Continuity

## 10.1 Session 結束

Session 結束時，系統應盡可能保存：

- 當前 Task 狀態。
- 已完成的工作。
- 未完成的工作。
- 執行中的重要決策。
- 新發現的問題。
- 已產生的成果。
- 下一步候選工作。
- 必須保留的 Context。

Session 結束不代表 Project 結束。

## 10.2 Session 恢復

新 Session 啟動時：

1. 識別所屬 Project。
2. 取得最新 Project State。
3. 取得有效 Goal。
4. 取得相關 Decision History。
5. 取得 Pending Issues。
6. 取得最近 Work History。
7. 檢查是否有進行中的 Task。
8. 檢查是否有新事件或資料。
9. 重建 Context。
10. 將恢復後的脈絡交給 Agent。

## 10.3 Session Continuity 與 Agent Identity

不要求每次都使用同一個模型、同一個 Agent Session 或同一個 Process。

只要持久化狀態與必要脈絡能正確恢復，系統就能維持工作連續性。

因此，Continuity 應建立在 AIA 的資料及狀態管理之上，而不是依賴某個模型供應商的 Session Memory。

# 11. Long-term Project Lifecycle

## 11.1 Project 建立

Project 建立時必須定義：

- 目標。
- 範圍。
- 領域。
- 負責 Agent。
- 預期成果。
- 主要限制。
- 初始狀態。

## 11.2 Project 執行

執行期間，系統持續更新：

- Project State。
- Task History。
- Decision History。
- Evaluation History。
- Pending Issues。
- Memory Relations。

## 11.3 Project 暫停

暫停時必須保存完整狀態。

系統應能回答：

- 為什麼暫停？
- 目前完成多少？
- 哪些事情尚未完成？
- 哪些條件恢復後可以繼續？
- 是否有其他 Project 正在等待本專案的成果？

## 11.4 Project 恢復

恢復時不得直接從上一個 Task 開始，而應重新檢查：

- 目標是否仍然有效。
- 過去決策是否仍然適用。
- 外部環境是否改變。
- 是否出現新問題。
- 是否需要調整原定計畫。

## 11.5 Project 完成

Project 完成時：

- 驗證最終成果。
- 更新 Goal 狀態。
- 保存完整工作歷史。
- 保存重要決策。
- 建立 Final Evaluation。
- 整理可重用經驗。
- 更新相關 Project 的依賴狀態。

Project 完成後仍應保留其歷史，以供日後參考。

# 12. Multiple Projects Continuity

## 12.1 多專案並存

AIA 必須能同時維持多個 Project。

例如：

| Project | Domain | 狀態 |
|---|---|---|
| 台股自主投資實驗 | Investment | ACTIVE |
| AIA Runtime 開發 | Coding | ACTIVE |
| 歷史小說創作 | Writing | PAUSED |
| AIA 自主性研究 | Core | ACTIVE |

上述狀態僅為示例。

## 12.2 Project Switching

當系統從一個 Project 切換至另一個 Project 時：

- 保存原 Project 狀態。
- 釋放不再需要的工作 Context。
- 取得新 Project 的必要 Context。
- 檢查新 Project 的目標與限制。
- 恢復相關 Task 或建立新的工作計畫。

不得因為切換 Project 而把前一個 Project 的工作狀態遺失。

## 12.3 Project Priority

多個 Project 同時需要資源時，必須評估：

- Goal Priority。
- Urgency。
- Expected Value。
- Risk。
- Resource Requirement。
- Deadline。
- Dependency。
- Opportunity Cost。

不得簡單地依照建立時間或最後活動時間決定優先順序。

## 12.4 Project Starvation

系統必須偵測長期沒有進展的 Project。

但沒有進展不代表一定需要立即執行。

可能原因：

- 使用者刻意暫停。
- 外部條件尚未成熟。
- 資源正在投入更重要的工作。
- 目前沒有值得執行的下一步。

系統可以主動提出檢查，但不能擅自將暫停中的 Project 恢復成 ACTIVE。

# 13. Cross-Domain Continuity

## 13.1 跨領域關係

不同 Domain Agent 可以參與同一個 Project，也可以透過不同 Project 互相協作。

例如：

Writing Agent 提出小說管理工具需求。

Coding Agent 接受工作並完成工具。

Writing Agent 使用新工具重新檢查既有小說。

這段工作至少涉及：

- Writing Project。
- Coding Project。
- Cross-Domain Task。
- Capability Dependency。
- Shared Deliverable。
- Verification Record。

## 13.2 跨領域資料共享

資料共享必須考慮：

- 資料來源。
- 所屬 Project。
- 資料版本。
- 使用權限。
- 適用範圍。
- 資料可信度。
- 更新時間。

不能因為 Agent 可以存取某個 Memory，就代表它可以將該資料用於所有領域。

## 13.3 Cross-Domain Context

跨領域委派時，必須提供足夠但有限的 Context。

例如 Writing Agent 不需要把整部小說的所有章節交給 Coding Agent。

它只需要提供：

- 工具需求。
- 相關資料結構。
- 預期功能。
- 驗收條件。
- 必要的測試資料。

原始 Project 的完整脈絡仍由原始 Project 管理。

詳細跨領域協作規格由 AIA-12 定義。

# 14. Memory Integration

## 14.1 Memory 的分類

建議將長期記憶依用途分類：

| 類型 | 內容 |
|---|---|
| Semantic Memory | 穩定知識及事實 |
| Episodic Memory | 過去事件與經驗 |
| Procedural Memory | 工作方法與操作經驗 |
| Project Memory | 特定 Project 的背景與進度 |
| Decision Memory | 過去決策及理由 |
| Error Memory | 錯誤與失敗經驗 |
| Capability Memory | 工具與 Agent 能力資訊 |

這是邏輯分類，不要求每一種類型都使用獨立資料庫。

## 14.2 Memory 更新

Memory 更新應考慮：

- 新資訊是否可靠。
- 是否與既有資訊衝突。
- 是否需要保留原始來源。
- 是否只是暫時性資訊。
- 是否影響目前 Project。
- 是否需要觸發重新評估。

## 14.3 Memory Consolidation

長期運作可能產生大量記憶。

系統可以進行：

- 重複資料整理。
- 歷史摘要。
- 關聯建立。
- 過期資訊標記。
- 知識版本更新。
- 重要經驗提取。

但不得因為摘要而刪除必要的原始證據。

重要決策及稽核資料必須保留可追溯來源。

# 15. Continuity 與 Self-Correction

Continuity 是 Self-Correction 的必要條件之一。

如果系統不知道過去的決策、預期結果及實際執行過程，就無法正確分析自己的錯誤。

例如：

Investment Agent 不能只知道目前投資組合價值。

它還必須知道：

- 之前持有什麼股票。
- 何時買進及賣出。
- 當時為什麼這樣決策。
- 當時預期什麼結果。
- 後來實際發生什麼。
- 哪些因素與預測不同。
- 過去是否有類似錯誤。

Coding Agent 不能只知道目前程式碼。

它還必須知道：

- 原始設計。
- 過去修改。
- Bug 歷史。
- 回歸測試結果。
- 技術決策。
- 修改前後的差異。

Writing Agent 不能只知道目前章節內容。

它還必須知道：

- 原始大綱。
- 人物設定版本。
- 歷史資料來源。
- 過去校勘結果。
- 已修正的矛盾。
- 尚未確認的推論。

Self-Correction 的評估與修正流程由 AIA-11 定義。

# 16. Data Consistency 與 Recovery

## 16.1 一致性要求

Project State、Goal、Task、Decision 及 Event 可能由不同 Worker 更新。

必須確保：

- 同一筆資料不會被無意義覆寫。
- 重要狀態變更具有一致性。
- 重複事件不會造成重複工作。
- 狀態與歷史紀錄可以互相核對。
- 更新失敗時可以安全重試。

## 16.2 Concurrency Control

建議採用：

- Transaction。
- Optimistic Locking。
- State Version。
- Idempotency Key。
- Unique Constraint。
- Event Deduplication。

對同一 Project 的重要狀態更新，應避免多個 Worker 同時寫入而造成資料遺失。

## 16.3 Recovery

系統重新啟動後應檢查：

- 未完成 Session。
- 執行中的 Task。
- 未完成 Decision。
- 未確認的交付成果。
- 未處理的 Event。
- Project State 與 Task 狀態不一致。
- 尚未完成的跨領域委派。

Recovery 必須能判斷：

- 可以直接恢復。
- 必須重新驗證。
- 需要重試。
- 需要人工介入。

不能只因為某個 Task 的狀態是 RUNNING，就直接認定原本的 Agent 仍然在執行。

# 17. Context Window 與長期記憶限制

## 17.1 不依賴無限 Context

LLM Context Window 有容量限制。

AIA 必須將長期資料存放在外部持久化系統，並依工作需求載入。

## 17.2 Context Selection

載入 Context 時應考慮：

- 工作相關性。
- 目標重要性。
- 資料可信度。
- 資料新鮮度。
- 決策影響範圍。
- Token Budget。

## 17.3 Context Compression

可以對大量歷史資料進行摘要，但必須保留：

- 原始資料位置。
- 來源識別碼。
- 版本。
- 摘要產生時間。
- 摘要所涵蓋的範圍。
- 尚未確認的資訊。

摘要不得被當作原始事實。

## 17.4 Context Reconstruction Failure

若無法重建足夠脈絡：

- 不應假裝記得。
- 不應直接繼續高風險工作。
- 應重新讀取原始文件。
- 必要時向人類確認。
- 保存脈絡缺失事件。

# 18. Project Persistence 的實際應用

## 18.1 Investment Project

長期維持：

- 初始資金。
- 歷史持股。
- 現金。
- 交易紀錄。
- 投資決策。
- 市場觀察。
- 預測。
- 預測評估。
- 策略變更。
- 月度資產檢查。

即使 Investment Agent 每次只在特定時間執行，也必須保留完整投資脈絡。

## 18.2 Coding Project

長期維持：

- 系統需求。
- 架構文件。
- Package 結構。
- 程式碼版本。
- 技術決策。
- Bug。
- Issue。
- 測試結果。
- 修改紀錄。
- 未完成工作。
- Release 狀態。

Coding Agent 重新接手時，必須理解目前程式碼與設計文件之間的關係。

## 18.3 Writing Project

長期維持：

- 整體大綱。
- 各卷規劃。
- 章節進度。
- 人物檔案。
- 人物關係。
- 歷史年表。
- 地理及制度資料。
- 史實校勘。
- 虛構設定。
- 待確認問題。
- 修改歷史。

Writing Agent 不應只依賴最近一次對話判斷小說進度。

# 19. Observability

Continuity 必須能夠被人類觀察及檢查。

使用者應能檢視：

- 目前有哪些 Project。
- 各 Project 的狀態。
- 主要目標。
- 最近完成的工作。
- 目前進行中的 Task。
- 尚未完成的工作。
- 長期停滯的問題。
- 最近重要決策。
- 最近的目標變更。
- 最近的 Project State 更新。
- 是否存在資料不一致。

建議提供 Project Timeline。

Project Timeline 應將重要事件依時間排列，讓人類能夠理解專案如何從建立、執行、修正到完成。

# 20. Security 與資料隔離

Continuity 不能成為跨越權限邊界的理由。

必須遵守：

1. Agent 只能讀取獲得授權的 Project。
2. 跨 Domain 資料共享必須經過權限檢查。
3. Memory 不得繞過 Policy。
4. 歷史決策不能被 Agent 任意修改。
5. Project State 更新必須留下紀錄。
6. 外部資料必須保留來源及可信度。
7. 敏感資料必須遵守既有存取限制。
8. 使用者要求刪除或停止使用某些資料時，必須依資料治理流程處理。

重要歷史與 Audit 不得因為一般 Memory 清理而遭到刪除。

# 21. Go Implementation 建議

## 21.1 Package 建議

建議在既有 Go 專案中逐步建立：

```text
internal/
  continuity/
    project/
    goal/
    context/
    decision/
    history/
    snapshot/
    recovery/
    model/
    repository/
```

這是建議的邏輯分工，不代表目前程式碼已存在這些 Package。

## 21.2 核心介面概念

```go
type ProjectStateService interface {
    GetCurrentState(
        ctx context.Context,
        projectID string,
    ) (*ProjectState, error)

    UpdateState(
        ctx context.Context,
        update ProjectStateUpdate,
    ) error
}

type ContextResolver interface {
    ResolveProjectContext(
        ctx context.Context,
        request ContextRequest,
    ) (*ProjectContext, error)
}

type DecisionHistoryService interface {
    RecordDecision(
        ctx context.Context,
        decision DecisionRecord,
    ) error

    GetDecisionHistory(
        ctx context.Context,
        projectID string,
        limit int,
    ) ([]DecisionRecord, error)
}

type ProjectRecoveryService interface {
    RecoverProject(
        ctx context.Context,
        projectID string,
    ) (*RecoveryResult, error)
}
```

以上僅為介面示意，實際設計必須配合既有 Service、Repository 與 Domain Model。

## 21.3 儲存設計

初期可以繼續使用 SQLite。

建議採取：

- 結構化資料表保存 Project、Goal、Decision、Task 關係。
- Event Table 保存重要狀態變更。
- Snapshot 保存重要歷史狀態。
- Memory Repository 保存可檢索記憶。
- File Storage 保存文件、程式碼及大型交付成果的引用資訊。

不應為了長期記憶而將所有資料塞進單一 JSON 欄位。

## 21.4 Repository 責任

Repository 應負責：

- 資料持久化。
- 查詢。
- 版本控制。
- 交易一致性。
- 唯一性與重複資料防護。

Service 應負責：

- 狀態轉換。
- 業務規則。
- 關聯維護。
- Project Lifecycle。
- Context Reconstruction。

Autonomy Core 負責：

- 判斷需要取得哪些脈絡。
- 評估目前狀態。
- 決定是否恢復、繼續或重新規劃。

# 22. 實作階段

## C0 — Project State Foundation

目標：建立可持久化的 Project State。

工作：

- Project State Model。
- State Version。
- Project Lifecycle。
- Project Repository。
- Project Event。
- State Update。

驗收：

- Project 狀態可以持久化。
- 狀態變更可以追溯。
- 不會因 Session 結束而遺失。

## C1 — Goal & Decision Continuity

目標：將 Goal 與 Decision 納入長期專案脈絡。

工作：

- Goal History。
- Decision Record。
- Decision Snapshot。
- Goal-Project Relationship。
- Decision Dependency。

驗收：

- 可以重建目標形成及變更歷史。
- 可以查詢過去重要決策。
- 可以辨識決策對其他工作的影響。

## C2 — Work History & Pending Issues

目標：建立長期工作歷史與未完成事項管理。

工作：

- Task History。
- Work Record。
- Issue Management。
- Pending Work。
- Next Action Candidates。
- Dependency Tracking。

驗收：

- 可以知道專案已完成及未完成工作。
- 可以辨識長期阻塞問題。
- 不會將候選工作直接當成正式 Task。

## C3 — Context Reconstruction

目標：讓 Agent 能在不同 Session 恢復 Project。

工作：

- Context Resolver。
- Context Selection。
- Context Snapshot。
- Context Completeness。
- Memory Integration。
- Context Versioning。

驗收：

- Agent 可以在新 Session 恢復工作。
- 能取得正確目標與歷史決策。
- 不會混淆不同 Project。
- 資料不足時能明確指出缺失。

## C4 — Multiple Project Continuity

目標：支援多專案長期並存。

工作：

- Project Switching。
- Project Priority。
- Project Starvation Detection。
- Project Recovery。
- Resource Awareness。

驗收：

- 可以同時維持多個 Project。
- 切換 Project 不會遺失狀態。
- 能辨識長期停滯專案。
- 不會擅自恢復人類暫停的 Project。

## C5 — Cross-Domain Continuity

目標：建立跨領域的長期工作關係。

工作：

- Cross-Domain Project Reference。
- Shared Deliverable。
- Capability Dependency。
- Context Handoff。
- Cross-Domain History。

驗收：

- 可以追蹤跨領域工作來源。
- 可以追蹤交付成果及驗證結果。
- 原始 Project 能取得改善後的能力。
- 不會因跨領域共享而混淆資料權限。

# 23. Acceptance Criteria

## 23.1 Project Continuity

- [ ] Project 可以長期存在。
- [ ] Project State 可以跨 Session 保存。
- [ ] Project 可以暫停及恢復。
- [ ] Project 完成後保留歷史。
- [ ] Project 狀態變更具有版本及原因。

## 23.2 Goal Continuity

- [ ] Goal 與 Project 具有明確關係。
- [ ] Goal 變更可以追溯。
- [ ] 可以恢復目前有效目標。
- [ ] 可以辨識已完成、暫停及失效目標。

## 23.3 Decision Continuity

- [ ] 重要決策具有獨立紀錄。
- [ ] 可以取得決策當時的脈絡。
- [ ] 不會因新結論覆寫舊決策。
- [ ] 可以追蹤決策影響範圍。

## 23.4 Context Reconstruction

- [ ] 可以在新 Session 重建 Project 脈絡。
- [ ] 可以取得相關 Memory。
- [ ] 可以取得歷史決策。
- [ ] 可以取得未完成工作。
- [ ] 可以辨識資料缺失及衝突。
- [ ] 不會將摘要誤認為原始事實。

## 23.5 Multiple Projects

- [ ] 可以同時維護多個 Project。
- [ ] 可以切換 Project。
- [ ] 可以維持各 Project 獨立狀態。
- [ ] 可以追蹤跨 Project 依賴。
- [ ] 不會擅自改變人類設定的暫停狀態。

## 23.6 Recovery

- [ ] Runtime 重啟後可以恢復 Project。
- [ ] 未完成 Task 不會被誤判為已完成。
- [ ] 重複事件不會造成重複工作。
- [ ] 狀態不一致可以被發現。
- [ ] 重要歷史紀錄不會遺失。

# 24. 最終設計原則

Continuity 的目標不是讓 AIA 永遠記得所有事情，而是讓 AIA 永遠能夠找回完成目前工作所需要的脈絡。

AIA 必須能夠同時維持多個長期 Project：

- 投資可以持續觀察、決策與檢討。
- 程式可以持續開發、除錯與改善。
- 小說可以持續創作、校勘與修正。

當系統切換領域時，不應遺失任何一個 Project 的歷史。

當系統重新啟動時，不應因為 Session 消失就忘記自己曾經做過什麼。

當新資訊出現時，應能找到受到影響的目標、決策及工作。

當某個 Project 暫時沒有工作時，可以等待，但必須保留完整狀態。

**真正的時間連續性，不是 AI 一直保持清醒，而是即使經過中斷、切換與時間流逝，它仍然能夠正確理解自己過去做過什麼、現在處於什麼狀態，以及接下來應該如何繼續。**
