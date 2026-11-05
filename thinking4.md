Day 1：建立目標，產生 Goal、Plan、Tasks 並開始執行：部分

各段零件都在，但沒有串成一條自動流程。goal decompose 只會把一個 Goal 轉成一個 Task，標題和描述要由呼叫者提供，不是系統自己拆解。
Plan 目前屬於 Problem 服務（Problem → 分析 → Plan），Goal 只是可選的連結欄位。所以「建立專案和目標後，系統自己產生 Plan 和一批 Task」這件事，現在要你一步步手動下指令。
開始執行這段沒問題，Task Worker 會接手 pending 的任務。

Day 2：自主喚醒，恢復脈絡、檢查進度、發現未完成工作：部分，而且有個具體漏洞

新版有 runAutonomyWorker，每 15 分鐘醒一次、先檢查 Emergency Stop、再跑一輪 cycle。這是很大的進展。
但它呼叫 RunCycle 時，ProjectID 寫死是 "default"，沒有去列舉你建立的 Project。所以 Day 1 你建的專案，不會被這個 worker 喚醒，它只照顧預設專案。
事件驅動的觸發也只做了一半：TriggerAutonomousCycle 函式存在，但整個程式碼沒有任何地方呼叫它，Task 完成、新證據進來都不會觸發 cycle。
「發現未完成的工作」這段邏輯有測試（phase3 的 GoalGap 發現與去重），所以修這個漏洞很可能只是讓 worker 逐一處理活躍專案，不用重寫判斷邏輯。
一個我還沒在新版確認的疑慮：舊版預設每日 cycle 上限是 48，15 分鐘一次等於一天 96 次，如果預設值沒變，大約半天就會耗盡額度。

Day 3	失敗留證據、分析原因、決定重試或改 Plan	部分	零件都在：學習引擎有 RecordDiscrepancy、AnalyzeRootCause，Plan 有 plan_changes 和 max_plan_retries。但 Scenario D 只測了 coding pilot 的「失敗後第二次修好」，「RCA」在測試裡只是證據紀錄上的一個標籤，沒有被斷言。沒有測試驗證「Task 失敗 → 證據 → RCA → 重試或改 Plan」整條鏈

Day 4	重啟後恢復 Goal、Decision、Evidence、Task	已測	phase1_continuity_and_provenance_test.go 的 M1 模擬崩潰與重啟，重建專案狀態、待辦與事件歷史，之後還做決策溯源回查（M2）

Day 5	新證據與假設矛盾，重新評估原決策	缺口	知識層能記錄 REFUTES 連結和 Claim 修訂，但證據變動後不會回頭找依賴它的決策，AIA-18 §8.4 的傳播機制沒有實作，也沒有對應測試

Day 6	修改 Plan、執行修正、驗證、記錄學習	部分	修 Plan、驗證（V1～V6）、學習紀錄各自存在，但沒有被 Day 5 的矛盾證據自動觸發，驗證也沒有延遲複驗

Day 7	發現次要目標沒價值並提議調整，依治理政策決定	部分	治理那一半做得好：目標演化有「必須證明前提 Claim 被推翻」的防護，STRATEGIC 目標需人工核准。但這個流程要由呼叫者帶入 InvalidatedClaimIDs 才會啟動，沒有東西自己發現「這個目標已經沒價值」

---

Day 1 — Goal Creation
待補接線








breeze.pm


驗證：
1. 建立 Project。
2. 建立 Goal。
3. AIA 自動分析 Goal。
4. 產生 Plan。
5. Plan 自動拆解成多個 Tasks。
6. Tasks 具有 Dependency 與 Acceptance Criteria。
成功條件： 不需要使用者逐一建立 Plan 和 Task。

Day 2 — Autonomous Wake-up
待補接線

驗證：
1. 建立兩個 Active Projects。
2. 建立一個 Paused Project。
3. 觸發 AutoWakeWorker。
4. 兩個 Active Projects 都必須產生 Cycle。
5. Paused Project 不得被喚醒。
6. 注入事件，確認 Event Trigger 能啟動相應處理。
成功條件： 不依賴 default Project，也不需要人工呼叫 Cycle。

Day 3 — Failure & Recovery
待補驗證

驗證：
1. 注入 Task 執行失敗。
2. 保存 Failure Evidence。
3. 執行 Root Cause Analysis。
4. 判斷 Retry 或 Replan。
5. 產生修正後的 Task。
6. 執行並驗證結果。
成功條件： 不是單純重試，而是能依失敗原因採取適當行動。

Day 4 — Runtime Recovery
已有 M1 測試

驗證：
1. 停止 Runtime。
2. 保留 SQLite。
3. 重建 Worker、Service、Orchestrator。
4. 恢復 Goal、Plan、Task、Decision、Evidence。
5. 確認沒有重複執行已完成的 Task。
成功條件： 恢復的是原有工作狀態，而不是建立一個新的工作。

Day 5 — Evidence-Driven Reassessment
核心功能缺口

驗證：
1. 注入與原始假設矛盾的新 Evidence。
2. Evidence 被保存並關聯至原決策。
3. Evidence 變動觸發 Reassessment。
4. 找出受影響的 Decision。
5. 判斷原決策是否仍然有效。
6. 必要時提出新的 Plan。
成功條件： 新證據不只是被記錄，而是實際影響舊決策。

Day 6 — Self-Correction & Learning
待補接線

驗證：
1. 使用 Day 5 的 Reassessment 結果。
2. 修正原有 Plan。
3. 執行修正後的 Task。
4. 比較修正前後的結果。
5. 建立 Learning Record。
6. 確認未來決策能取得這筆 Learning。
成功條件： 學習不是產生一筆文字紀錄，而是能改變後續行為。

Day 7 — Goal Evolution
待補接線

驗證：
1. 注入環境變化。
2. 原本的次要 Goal 失去價值。
3. AIA 主動發現 Goal 與現況不一致。
4. 評估是否需要調整。
5. 提出 Goal Revision。
6. 經 Governance 核准後更新 Goal。
7. 保存舊版 Goal 與變更原因。
成功條件： AIA 不只是執行目標，也能重新思考目標是否仍值得追求。

測試結果必須產生量化報告
除了 PASS / FAIL，我建議每個 Day 都產生一筆驗收紀錄。
指標	計算方式
Goal Creation	自動產生 Goal / 預期 Goal
Task Decomposition	自動產生 Task / 預期 Task
Project Wake-up	成功喚醒 / Active Projects
Failure Recovery	成功恢復 / 注入失敗
State Recovery	恢復成功項目 / 持久化項目
Evidence Response	成功重新評估 / 注入的新證據
Correction Effectiveness	修正後改善次數 / 修正總次數
Goal Adaptation	合理調整次數 / 應調整情境次數

注意：Day 5 的 Evidence Response 不能只檢查有沒有產生事件，還必須檢查 Decision 是否真的重新評估。

Day 6 也不能只檢查 Learning Record 是否存在，而必須檢查後續決策是否能使用它。

還有一個重要問題：不要讓七天測試掩蓋缺口
我建議測試採取兩種模式：
模式	用途
Strict Acceptance	七天全部通過才算成功
Diagnostic	每一天獨立報告成功、失敗及缺失能力

目前已知 Day 5 還有功能缺口，所以第一版 Strict Acceptance 預期不會全部通過。

這不是壞事。

我們要的是一份能夠隨著 AIA 修正而逐步轉綠的驗收測試，而不是一開始就設計成全部 PASS。
