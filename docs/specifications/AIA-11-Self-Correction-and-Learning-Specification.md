# AIA-11

> [!NOTE]
> **架構重整與收攏公告 (Consolidated Architecture Specification)**
> 本規範為 AIA 歷史基準規範（規格編號已於 AIA-22 永久封頂）。為杜絕規格膨脹與重複定義，本規範之現行核心標準已正式收攏重整至 **[Volume 4: Self-Correction, Experiments & Benchmarks](file:///home/wade/src/github/aia/docs/specifications/volume-4-self-correction-and-experiments.md)**。最新架構與維護基準請以對應之 Volume 為主，完整六大核心卷軸總覽請參閱 [Specification Catalog (README.md)](file:///home/wade/src/github/aia/docs/specifications/README.md)。

# Self-Correction & Learning Specification

## 自我修正與持續學習規格書

**文件版本：** v2.0
**文件代號：** AIA-11
**文件性質：** Autonomy Core 子系統規格
**適用範圍：** AIA Runtime、Autonomy Core、Continuity、所有 Domain Agent
**主要依據：** AIA-01、AIA-02、AIA-08、AIA-09、AIA-10

---

# 1. 文件目的

本文件定義 AIA 如何透過執行經驗、結果驗證、錯誤分析及知識更新，持續改善自身的判斷與工作能力。

AIA 必須具備從過去經驗中學習的能力，而且學習成果必須能夠影響後續行為。

例如：

- Investment Agent 必須檢查過去投資判斷是否合理，辨識預測誤差與策略缺陷。
- Coding Agent 必須能根據錯誤訊息、測試結果及程式執行狀況，找出問題並修正程式。
- Writing Agent 必須能根據史實校勘、人物關係及跨卷連續性檢查，修正既有內容。

上述三種情境的工作內容不同，但自我修正的基本機制應該相同。

## 1.1 核心目標

AIA 必須建立以下能力：

1. 記錄原始判斷。
2. 保存執行前的預期結果。
3. 取得實際結果。
4. 比較預期與實際結果。
5. 判斷是否存在錯誤或改善機會。
6. 分析可能原因。
7. 提出修正方案。
8. 評估修正風險。
9. 執行或提交核准。
10. 驗證修正是否有效。
11. 將驗證後的成果納入長期知識。
12. 在後續工作中實際使用學習成果。

## 1.2 核心定義

**Self-Correction 是根據可追溯證據，修正既有判斷、策略、知識或執行方法，並透過後續驗證確認修正效果的能力。**

Self-Correction 不等於：

- 產生一段反思文字。
- 單純增加 Memory。
- 修改 System Prompt。
- 修改程式碼。
- 因為一次失敗就更換策略。
- 將所有成功都歸因於自己的能力。
- 將所有失敗都歸因於外部環境。

修正必須有證據、有原因、有變更紀錄，也必須能夠驗證。

# 2. Learning 與 Self-Correction 的關係

## 2.1 Learning

Learning 是從經驗中取得新的知識、規則、模式或判斷依據。

例如：

Investment Agent 發現某類市場資訊對特定投資判斷具有參考價值。

這項發現可以成為新的知識，但不代表原有策略已經需要修改。

## 2.2 Self-Correction

Self-Correction 是發現既有判斷或方法存在問題，並採取修正措施。

例如：

Investment Agent 發現自己過度依賴單一市場指標，導致多次忽略其他重要資訊，因此提出調整評估流程的方案。

## 2.3 Self-Improvement

Self-Improvement 是經過驗證後，確認修正帶來實際改善。

例如：

修改投資評估流程後，經過一段時間的觀察，發現預測校準程度改善，且風險評估更加完整。

必須區分：

| 概念               | 核心問題        |
| ---------------- | ----------- |
| Learning         | 我學到了什麼？     |
| Self-Correction  | 我應該修正什麼？    |
| Self-Improvement | 修正後是否真的變好了？ |

三者形成連續流程，但不能視為同一件事。

# 3. Self-Correction Lifecycle

AIA 應建立完整的自我修正循環。

```text
Experience
    ↓
Observation
    ↓
Expectation vs Actual
    ↓
Discrepancy Detection
    ↓
Root Cause Analysis
    ↓
Correction Proposal
    ↓
Risk Assessment
    ↓
Approval / Authorization
    ↓
Correction Execution
    ↓
Verification
    ↓
Learning Record
    ↓
Future Application
```

## 3.1 Experience

來源可以是：

- Task Execution。
- Agent Decision。
- External Observation。
- Project Milestone。
- User Feedback。
- Automated Test。
- Investment Result。
- Historical Verification。
- Long-term Performance。

## 3.2 Observation

AIA 必須取得足以分析結果的資訊。

觀察資料應包含：

- 當時環境。
- 當時使用的資訊。
- 當時的 Goal。
- 當時的 Strategy。
- 當時的 Decision。
- 實際執行內容。
- 實際結果。

如果無法還原當時狀態，必須標示資料不足。

## 3.3 Expectation vs Actual

比較原始預期與實際結果。

例如：

```text
預期：
程式修改後，指定測試應全部通過。

實際：
部分測試通過，但原有功能出現回歸錯誤。

差異：
新功能完成，但未達成整體品質目標。
```

不能只檢查主要任務是否完成，還必須檢查是否產生其他負面影響。

## 3.4 Discrepancy Detection

差異可能來自：

- 判斷錯誤。
- 資料錯誤。
- 資訊不足。
- 執行錯誤。
- 環境變化。
- 原始假設失效。
- 評估標準不完整。
- 偶然因素。

發現差異不代表一定需要修改。

必須進一步分析原因。

# 4. Error Classification

AIA 必須將問題分類，避免使用單一方式處理所有錯誤。

## 4.1 Knowledge Error

知識本身錯誤或過時。

例如：

- 使用錯誤的歷史年代。
- 使用過期的 API 規格。
- 使用錯誤的市場資料。

處理方式：

- 查證來源。
- 修正知識紀錄。
- 保留舊版本。
- 標示新舊資訊的適用範圍。

## 4.2 Reasoning Error

推理過程存在問題。

例如：

- 忽略重要變數。
- 錯誤解讀資料。
- 過度推論。
- 混淆相關性與因果關係。

處理方式：

- 重建原始推理。
- 找出錯誤假設。
- 建立替代解釋。
- 比較不同推論結果。

## 4.3 Strategy Error

工作策略不適合目前目標或環境。

例如：

- 投資策略長期無法達成預期。
- Coding Agent 使用不適當的架構。
- Writing Agent 的章節安排造成時間線斷裂。

處理方式：

- 評估策略適用條件。
- 提出替代策略。
- 比較預期成果與風險。
- 必要時進行受控實驗。

## 4.4 Execution Error

執行過程發生錯誤。

例如：

- 程式碼修改不完整。
- Task 執行失敗。
- 外部資料抓取錯誤。
- 文件更新遺漏。

處理方式：

- 保存錯誤訊息。
- 分析執行環境。
- 修正執行步驟。
- 重新執行與驗證。

## 4.5 Evaluation Error

評估方法本身存在缺陷。

例如：

- 只看投資報酬，忽略風險。
- 只看程式是否編譯，忽略回歸錯誤。
- 只看章節是否完成，忽略歷史一致性。

處理方式：

- 重新檢查評估指標。
- 增加必要的驗證項目。
- 調整評估方法。
- 保留舊評估結果供比較。

## 4.6 Goal Assumption Error

原始目標的前提可能不再成立。

這種情況應交由 AIA-09 的 Goal Evolution 機制處理。

Self-Correction 可以提出證據與修正建議，但不能直接越權修改 Strategic Goal。

# 5. Root Cause Analysis

發現問題後，AIA 不應立即修改。

必須先分析原因。

## 5.1 Root Cause Model

建議採用以下分析層次：

1. Symptom：觀察到什麼現象？
2. Evidence：有哪些證據？
3. Immediate Cause：直接原因是什麼？
4. Contributing Factors：有哪些促成因素？
5. Root Cause：根本原因可能是什麼？
6. Confidence：原因判斷有多可靠？
7. Alternatives：是否存在其他解釋？

## 5.2 避免錯誤歸因

一次失敗不能直接證明策略錯誤。

一次成功也不能直接證明策略正確。

AIA 必須區分：

- 可控制因素。
- 不可控制因素。
- 隨機因素。
- 資訊缺失。
- 模型判斷偏差。
- 執行品質問題。

## 5.3 Multiple Hypotheses

重要問題應至少考慮合理的替代解釋。

例如：

投資組合下跌，可能原因包括：

- 原始投資判斷錯誤。
- 市場出現無法預見的事件。
- 投資標的受到整體市場影響。
- 資訊來源不完整。
- 投資策略與當時市場環境不匹配。

不能只選擇最符合既有觀點的解釋。

# 6. Correction Proposal

根本原因分析完成後，AIA 可以提出修正方案。

## 6.1 Correction Types

| 類型                | 修正對象             |
| ----------------- | ---------------- |
| KNOWLEDGE_UPDATE  | 知識               |
| REASONING_UPDATE  | 推理規則             |
| STRATEGY_UPDATE   | 工作策略             |
| PROCEDURE_UPDATE  | 執行流程             |
| EVALUATION_UPDATE | 評估方法             |
| MEMORY_CORRECTION | 記憶內容             |
| PROMPT_UPDATE     | Agent 指令或 Prompt |
| CODE_CHANGE       | 程式碼              |
| GOAL_PROPOSAL     | 目標修訂提案           |

## 6.2 Correction Proposal 欄位

每個修正提案至少包含：

- Correction ID。
- Source Experience。
- Related Error。
- Root Cause。
- Proposed Change。
- Expected Improvement。
- Possible Regression。
- Required Resources。
- Risk Level。
- Reversibility。
- Validation Method。
- Approval Requirement。

## 6.3 多方案比較

AIA 應在合理範圍內比較不同修正方式。

例如：

問題：Investment Agent 過度依賴單一新聞來源。

可能方案：

A. 增加其他新聞來源。

B. 增加不同類型的市場資訊。

C. 調整資訊可信度評分。

D. 降低單一來源對決策的影響。

Agent 應比較成本、效果與風險，而不是直接採取第一個想到的方案。

# 7. Correction Authorization

不是所有修正都需要相同的核准程度。

## 7.1 Correction Levels

| 等級 | 修正內容                              | 授權方式               |
| -- | --------------------------------- | ------------------ |
| C0 | 分析與記錄                             | 自主                 |
| C1 | 新增一般知識與觀察紀錄                       | 依 Policy 自主        |
| C2 | 調整一般工作流程或可逆策略                     | 依既有授權自主            |
| C3 | 修改 Domain Agent Prompt、策略或程式碼     | 依專案 Policy 與變更風險核准 |
| C4 | 修改 AIA 核心 Runtime 或 Autonomy Core | 人類明確核准             |
| C5 | 修改治理規則、權限或 Audit 完整性機制            | Agent 不得自行執行       |

實際核准層級應由 AIA Policy 定義，不可只由 Agent 自行判定。

## 7.2 Correction 與 Goal Evolution 的界線

Self-Correction 可以修正：

- 如何分析。
- 如何判斷。
- 如何執行。
- 如何驗證。
- 如何保存知識。

但不能自行改變使用者設定的主要目標。

例如：

Investment Agent 可以修正投資策略，但不能把「最大化總資產」自行改成「降低波動」。

若它認為原始目標需要重新考慮，應提交 Goal Evolution Proposal。

# 8. Controlled Correction

修正必須受到控制，避免因為一次錯誤而造成更大的錯誤。

## 8.1 Baseline

修改前必須保存：

- 原始版本。
- 原始行為。
- 原始評估結果。
- 修正原因。
- 相關證據。

## 8.2 Isolated Validation

可行時，先在隔離環境驗證修正。

例如：

- Coding：獨立 Branch 或 Workspace。
- Investment：歷史資料回測或虛擬投資環境。
- Writing：文件副本與版本比較。

## 8.3 Limited Rollout

重大修正應先採取有限範圍部署。

觀察：

- 是否達成預期改善。
- 是否產生副作用。
- 是否破壞既有能力。
- 是否造成資源異常消耗。

## 8.4 Rollback

如果修正造成明顯退化，應能回復到先前穩定版本。

Rollback 必須保留：

- 回滾原因。
- 回滾時間。
- 被回滾版本。
- 回復版本。
- 相關驗證結果。

回滾不能刪除修正曾經發生的歷史。

# 9. Verification

修正完成不代表修正成功。

必須驗證。

## 9.1 Verification Model

驗證應包含：

1. 是否解決原始問題？
2. 是否達成預期改善？
3. 是否產生新的錯誤？
4. 是否破壞既有功能？
5. 是否仍符合原始 Goal？
6. 是否符合 Policy？
7. 是否有足夠證據支持改善結論？

## 9.2 Verification Status

| 狀態           | 意義   |
| ------------ | ---- |
| PENDING      | 尚未驗證 |
| PASSED       | 驗證通過 |
| FAILED       | 驗證失敗 |
| PARTIAL      | 部分達成 |
| INCONCLUSIVE | 證據不足 |
| ROLLED_BACK  | 已回滾  |

## 9.3 長期驗證

部分修正無法立即判定效果。

例如：

投資策略可能需要經過多個市場週期才能評估。

Writing Agent 的人物設定修正，可能要等後續章節完成後才能確認是否改善整體敘事。

因此，AIA 必須支援延遲驗證。

## 9.4 不可只驗證局部結果

修正某個部分時，必須考慮對整體 Project 的影響。

例如：

修正某一章人物年齡，還必須檢查其他章節是否引用相同資料。

修改程式某個模組，還必須檢查相關依賴與既有功能。

# 10. Learning Record

每次有效學習都應建立 Learning Record。

## 10.1 必要欄位

- Learning ID。
- Source Experience IDs。
- Related Project。
- Related Goal。
- Knowledge Type。
- Observation。
- Conclusion。
- Confidence。
- Supporting Evidence。
- Applicable Conditions。
- Known Limitations。
- Created At。
- Review Status。

## 10.2 Learning Type

- FACT：經查證的事實。
- PATTERN：觀察到的模式。
- HEURISTIC：經驗法則。
- PROCEDURE：工作程序。
- STRATEGY：策略經驗。
- ERROR_PATTERN：錯誤模式。
- EVALUATION_RULE：評估規則。

## 10.3 Learning Confidence

學習成果必須保留信心程度與適用條件。

例如：

「某種投資訊號在過去資料中具有參考價值。」

不能直接推論：

「未來出現這種訊號就一定應該買進。」

Learning 必須保留其資料範圍與限制。

# 11. Memory Integration

Learning Record 應與 AIA-10 的 Memory、Continuity 整合。

但 Memory 不應成為唯一的學習資料庫。

## 11.1 Memory 分類

建議區分：

- Episodic Memory：某次事件與經驗。
- Semantic Memory：已整理的知識。
- Procedural Memory：工作程序與方法。
- Strategic Memory：策略與長期經驗。
- Evaluation Memory：評估結果與校準資料。

## 11.2 Memory 更新原則

AIA 不應直接覆蓋舊知識。

當新證據與舊知識衝突時：

1. 保留舊紀錄。
2. 新增新證據。
3. 建立衝突關係。
4. 評估適用範圍。
5. 必要時建立新版本。
6. 記錄舊知識是否仍有部分適用性。

## 11.3 Memory 與 Learning 的差異

Memory 是保存資訊的機制。

Learning 是從資訊與經驗中形成可用知識的過程。

只有當 Learning 成果能在後續工作中被取用，才真正形成持續學習能力。

# 12. Prediction & Calibration

AIA 應保存重要判斷的預測及其信心程度，以便檢查長期判斷品質。

## 12.1 Prediction Record

建議欄位：

- Prediction ID。
- Decision ID。
- Prediction Statement。
- Expected Outcome。
- Confidence。
- Time Horizon。
- Evaluation Criteria。
- Created At。
- Expiration / Review Date。
- Actual Outcome。
- Evaluation Result。

## 12.2 Calibration

Calibration 是檢查信心程度是否與實際準確程度相符。

例如：

Agent 多次對某類預測給予 90% 信心，但實際命中率明顯低於 90%。

這代表信心校準可能存在問題。

AIA 應調整信心評估方式，而不是只改變預測內容。

## 12.3 Prediction Error

預測錯誤必須分類：

- 資訊不足。
- 推理錯誤。
- 模型能力限制。
- 環境變化。
- 隨機事件。
- 評估時間不適當。
- 原始問題定義不清楚。

預測未命中不必然代表整體判斷品質低落。

# 13. Domain Learning

各 Domain Agent 應有自己的專業學習方法。

但學習資料應遵循共通格式與治理規則。

## 13.1 Investment Learning

Investment Agent 必須保存：

- 每次投資決策。
- 當時市場資訊。
- 當時持股與現金。
- 預期結果。
- 信心程度。
- 風險評估。
- 實際成交紀錄。
- 後續資產變化。
- 決策檢討。
- 策略修正。

### 報酬與判斷品質分離

獲利不代表判斷一定正確。

虧損也不代表判斷一定錯誤。

例如：

某次投資決策在當時資訊下具有合理依據，但後來發生不可預見的重大事件，導致虧損。

這不應直接被歸類為推理錯誤。

反過來，缺乏合理依據的投資決策即使獲利，也不能因此被視為良好決策。

### 評估維度

- 投資組合總資產。
- 累積報酬。
- 風險暴露。
- 預測校準。
- 決策品質。
- 策略穩定性。
- 交易成本與滑價假設。
- 與比較基準的相對表現。

上述指標是不同的評估維度，不能用單一數值取代全部判斷。

投資實驗的主要目標仍是最大化總資產。

### 虛擬投資驗證

Investment Agent 應以虛擬投資環境驗證策略修正。

不得因為策略修正就取得真實金融交易授權。

## 13.2 Coding Learning

Coding Agent 必須保存：

- 原始需求。
- 原始程式狀態。
- 修改前後 Diff。
- 錯誤訊息。
- 測試結果。
- 修正原因。
- 回歸檢查結果。
- 最終版本。
- 後續維護結果。

### Coding Self-Correction

Agent 可以：

1. 發現程式錯誤。
2. 分析錯誤原因。
3. 提出修改方案。
4. 在隔離環境修改。
5. 執行適當驗證。
6. 比較修改前後結果。
7. 必要時回滾。
8. 將有效修正方式保存為知識。

但修改 AIA 自身核心程式碼，必須遵守更嚴格的核准與驗證流程。

## 13.3 Writing Learning

Writing Agent 必須保存：

- 原始大綱。
- 人物設定。
- 歷史時間軸。
- 歷史資料來源。
- 章節內容。
- 校勘結果。
- 修改紀錄。
- 跨卷一致性檢查。
- 使用者修正意見。

### Writing Self-Correction

Agent 可以：

1. 發現史實錯誤。
2. 發現人物年齡矛盾。
3. 發現時間線衝突。
4. 發現人物關係不一致。
5. 發現章節進度與大綱不符。
6. 提出修正方案。
7. 檢查修正對其他章節的影響。
8. 保存修改前後版本。

史實、合理推演與小說虛構必須明確區分。

使用者明確確認的設定也應保留其來源與版本，不可因 Agent 自行推理而無痕覆寫。

# 14. Cross-Domain Learning

不同 Domain 可以分享方法，但不能無條件共享結論。

例如：

Coding Agent 學會的除錯流程，可能有助於 Writing Agent 檢查資料矛盾。

Investment Agent 學會的預測校準方法，可能有助於其他 Agent 評估判斷信心。

但：

- 投資市場的統計關係不能直接套用到歷史研究。
- 程式測試的通過條件不能直接套用到小說品質。
- 某個 Domain 的成功策略不能直接被視為另一個 Domain 的有效策略。

## 14.1 Shared Learning

可以共享：

- 證據管理方法。
- 推理檢查程序。
- 錯誤分類方法。
- 信心校準方法。
- 版本控制原則。
- 驗證與回滾機制。

## 14.2 Domain-specific Learning

必須保留領域差異：

- Investment：市場與資產評估。
- Coding：程式正確性與系統品質。
- Writing：史實、敘事與跨章節一致性。

Cross-Domain Learning 必須經過適用性評估。

# 15. Learning Quality & Anti-Patterns

AIA 必須避免錯誤學習。

## 15.1 Overfitting

只根據少量案例修改策略，導致只適用於特定情況。

控制方式：

- 保留樣本數。
- 檢查適用範圍。
- 使用不同資料驗證。
- 避免單次結果決定長期策略。

## 15.2 Confirmation Bias

只收集支持既有觀點的證據。

控制方式：

- 主動尋找反例。
- 比較替代假設。
- 記錄反對證據。
- 檢查結論是否受到既有策略影響。

## 15.3 Outcome Bias

只根據最後結果評價當時決策。

控制方式：

- 保存決策當時的資訊。
- 保存預測與信心。
- 區分判斷品質與偶然結果。

## 15.4 Repeated Failure

相同錯誤不斷發生，卻沒有改善。

控制方式：

- 錯誤模式分類。
- 建立重複錯誤關聯。
- 提高相關問題優先順序。
- 檢查過去修正是否真正生效。

## 15.5 False Learning

把偶然相關性當成有效規則。

控制方式：

- 保存證據。
- 標示信心。
- 檢查替代解釋。
- 延遲確認。
- 限制未驗證規則的使用範圍。

## 15.6 Uncontrolled Self-Modification

Agent 因為一次錯誤就直接修改自身核心。

控制方式：

- 修正提案。
- 版本控制。
- 隔離驗證。
- Policy。
- 人類核准。
- Rollback。

# 16. Self-Correction Trace

每次重要修正都必須能夠完整追溯。

```text
Experience
    ↓
Observation
    ↓
Discrepancy
    ↓
Error Classification
    ↓
Root Cause Analysis
    ↓
Correction Proposal
    ↓
Authorization
    ↓
Correction
    ↓
Verification
    ↓
Learning Record
    ↓
Future Application
```

必須能夠回答：

- 哪次經驗促成修正？
- 原本的判斷是什麼？
- 發現了什麼問題？
- 為什麼認為這是問題？
- 採取了什麼修正？
- 修正是否經過核准？
- 修正後是否改善？
- 如果失敗，是否已經回滾？
- 這次學習是否影響後續行為？

# 17. Go Implementation 建議

本章描述邏輯元件與介面，不代表目前 AIA 程式碼已完成這些能力。

實際實作應整合既有 Runtime、Worker、Agent、Memory、Task、Policy、Audit 與 Repository。

## 17.1 Learning Engine

負責從 Experience 中形成 Learning Record。

```go
type LearningEngine interface {
    Analyze(
        ctx context.Context,
        experience Experience,
    ) (LearningAnalysis, error)

    CreateRecord(
        ctx context.Context,
        analysis LearningAnalysis,
    ) (LearningRecord, error)

    Retrieve(
        ctx context.Context,
        scope LearningScope,
    ) ([]LearningRecord, error)
}
```

## 17.2 Self-Correction Engine

負責判斷是否需要修正，以及產生修正提案。

```go
type SelfCorrectionEngine interface {
    DetectDiscrepancy(
        ctx context.Context,
        experience Experience,
    ) ([]Discrepancy, error)

    AnalyzeRootCause(
        ctx context.Context,
        discrepancy Discrepancy,
    ) (RootCauseAnalysis, error)

    ProposeCorrection(
        ctx context.Context,
        analysis RootCauseAnalysis,
    ) (CorrectionProposal, error)
}
```

## 17.3 Correction Executor

負責在授權範圍內執行修正。

```go
type CorrectionExecutor interface {
    Validate(
        ctx context.Context,
        proposal CorrectionProposal,
    ) error

    Execute(
        ctx context.Context,
        proposal CorrectionProposal,
    ) (CorrectionResult, error)

    Rollback(
        ctx context.Context,
        correctionID string,
    ) error
}
```

## 17.4 Verification Service

```go
type VerificationService interface {
    Verify(
        ctx context.Context,
        correction CorrectionResult,
    ) (VerificationResult, error)

    ScheduleReview(
        ctx context.Context,
        correctionID string,
        reviewAt time.Time,
    ) error
}
```

## 17.5 核心資料模型

```go
type Experience struct {
    ID         string
    ProjectID  string
    GoalID     string
    DecisionID string
    TaskID     string
    OccurredAt time.Time
    EvidenceIDs []string
}

type Discrepancy struct {
    ID          string
    ExperienceID string
    Type        string
    Description string
    Confidence  float64
}

type RootCauseAnalysis struct {
    ID            string
    DiscrepancyID string
    Causes        []string
    EvidenceIDs   []string
    Confidence    float64
}

type CorrectionProposal struct {
    ID          string
    ProjectID   string
    Type        string
    Description string
    RiskLevel   string
    Reversible  bool
    ApprovalRequired bool
}

type LearningRecord struct {
    ID          string
    ProjectID   string
    Type        string
    Observation string
    Conclusion  string
    Confidence  float64
    EvidenceIDs []string
}

type VerificationResult struct {
    CorrectionID string
    Status       string
    EvidenceIDs  []string
    Summary      string
    VerifiedAt   time.Time
}
```

上述型別是概念示例，實際欄位應配合既有資料模型。

## 17.6 與 Worker 整合

現有 Worker 可以承載 Learning 與 Self-Correction 工作。

建議職責：

- Runtime：提供執行生命週期。
- Worker：執行分析、修正與驗證工作。
- Learning Engine：整理經驗與知識。
- Self-Correction Engine：分析差異並提出修正。
- Correction Executor：執行獲准的變更。
- Verification Service：驗證修正效果。
- Continuity：保存長期工作關係。
- Policy：管理授權。
- Audit：保存不可任意竄改的歷史。

Worker 的存在本身不代表已具備學習能力。

必須確認 Worker 能否持續取得經驗、形成 Learning Record，並在後續決策中實際使用這些成果。

# 18. Implementation Phases

## L0 — Experience Collection

- 建立 Experience Record。
- 串接 Task、Decision、Execution Result。
- 保存原始預期與實際結果。
- 建立證據關聯。

## L1 — Discrepancy Detection

- 建立差異檢查。
- 建立 Error Classification。
- 支援預測與結果比較。
- 建立 Root Cause Analysis。

## L2 — Learning Foundation

- 建立 Learning Record。
- 整合 Memory。
- 支援知識版本。
- 支援信心與適用範圍。

## L3 — Correction Proposal

- 產生 Correction Proposal。
- 比較不同修正方案。
- 評估風險。
- 串接 Policy 與 Approval。

## L4 — Controlled Correction

- 執行獲准修正。
- 支援隔離驗證。
- 支援 Rollback。
- 保存 Correction Trace。

## L5 — Long-term Learning

- 支援延遲驗證。
- 支援 Prediction Calibration。
- 支援重複錯誤分析。
- 支援 Cross-Domain Learning。
- 將已驗證成果納入後續自主決策。

# 19. Acceptance Criteria

## 19.1 Experience

- [ ] 能保存重要 Decision 與 Execution Result。
- [ ] 能還原決策當時的資訊。
- [ ] 能保存原始預期與實際結果。
- [ ] 能連結相關 Project、Goal 與 Task。

## 19.2 Discrepancy

- [ ] 能發現預期與結果的差異。
- [ ] 能分類錯誤。
- [ ] 能分析可能原因。
- [ ] 能保留替代解釋。
- [ ] 不會將單次失敗直接判定為策略錯誤。

## 19.3 Learning

- [ ] 能建立 Learning Record。
- [ ] 能保存證據與信心。
- [ ] 能保留舊知識版本。
- [ ] 能處理新舊知識衝突。
- [ ] 能在後續工作中檢索學習成果。

## 19.4 Self-Correction

- [ ] 能提出修正方案。
- [ ] 能評估修正風險。
- [ ] 能檢查授權範圍。
- [ ] 能執行獲准修正。
- [ ] 能驗證修正效果。
- [ ] 能在必要時回滾。
- [ ] 能保存完整修正歷史。

## 19.5 Long-term Improvement

- [ ] 能追蹤重複錯誤。
- [ ] 能評估預測校準。
- [ ] 能檢查策略長期效果。
- [ ] 能區分短期成果與長期改善。
- [ ] 能讓已驗證學習影響後續行為。

# 20. 最終設計原則

AIA 的學習能力不能只存在於對話文字中。

它必須能夠留下實際影響系統行為的成果。

真正的自我修正應包含：

1. 知道自己當時做了什麼。
2. 知道當時為什麼這樣判斷。
3. 知道實際結果如何。
4. 知道預期與結果有何差異。
5. 能分析差異的原因。
6. 能提出合理修正。
7. 能在授權範圍內執行修正。
8. 能驗證修正是否有效。
9. 能保留失敗與成功的經驗。
10. 能在下一次遇到類似問題時，實際運用學習成果。

AIA 不應只是不斷重複工作。

它應能根據工作經驗改變自己的判斷方式。

AIA 不應只知道自己曾經犯錯。

它應能辨識錯誤、修正錯誤，並驗證自己是否真的改善。

**Learning 使 AIA 累積經驗；Self-Correction 使 AIA 改變行為；Self-Improvement 則證明這些改變確實帶來進步。**

---

# 21. 學習成熟度分級 (`L0~L3`) 與科學對照規範 (v2.1 Normative Addendum)

為防止系統將「產生一份看似合理的檢討報告」誤認為有效學習，AIA 將學習效果嚴格劃分為四個由系統驗收器（Verifier）判定的層級：

1. **`L0_RECORDED` (Experience Recorded)**：已記錄預期與實際結果落差及當時環境快照，尚未套用修正。
2. **`L1_APPLIED` (Correction Applied)**：已根據具證據錨點的可證偽根因假設套用修正，尚未經對照驗證。
3. **`L2_VERIFIED` (Correction Verified)**：修正後通過目標驗證、具備 Baseline/對照比較，且通過退化檢查（`Regression == PASSED`）。
4. **`L3_GENERALIZED` (Learning Generalized)**：後續在**不同循環（`cycle_id != source_cycle_id`）且不同情境/子模組/標的**中自動召回運用該經驗，並再次驗證正向無退化。**唯有達到 `L3` 方可視為泛化有效學習。**

## 21.1 七大強制對照欄位與因果歸因原則

每一筆 `learning_records` 必須結構化保存：
- **`Baseline`**：修正前的量化基準表現。
- **`Hypothesis`**：可證偽假設（預期修正會改善何項指標、在何種條件下若未發生則假設不成立）。
- **`Intervention`**：實際採取的修正內容與修正前狀態快照（供退化回滾）。
- **`Verification`**：修正後的驗證結果與對照方式（`CONTROL_BENCHMARK` / `PIT_REPLAY` / `SIMILAR_COHORT` / `PRE_POST_ONLY`）。
- **`Regression`**：是否造成既有能力或回歸測試退化。
- **`Follow-up`**：後續跨循環實際表現追蹤。
- **`Confidence`**：認識論信心值（`0.0 ~ 1.0`）。

## 21.2 三大因果防呆與四態結論 (`EFFECTIVE | INEFFECTIVE | REGRESSED | INCONCLUSIVE`)

1. **修正後變好，不一定是修正造成的（外部順風防呆）**：若有對照基準（如大盤指數或未修改基準組），必須檢驗「超額改善幅度 (`excess_delta`)」；若帳面變好完全來自外部環境順風，嚴禁標記為 `EFFECTIVE`。
2. **修正後沒變好，不代表根因分析完全錯誤**：系統分開記錄 `rca_validity_status` 與 `effectiveness_verdict`，區分「根因假設被推翻 (`FALSIFIED`)」與「根因可能成立但介入手段不當 (`INTERVENTION_MISMATCH`)」。
3. **無對照組信心上限（Confidence Cap）**：若無法建立對照組或歷史回放比較（`comparison_method == 'PRE_POST_ONLY'`）或樣本數不足，系統在程式碼層強制將 `epistemic_confidence` 上限鎖死於 `<= 0.45`，且結論最高僅能標為 `INCONCLUSIVE`（證據不足）。
4. **Recall 防污染規則**：`L0/L1` 與 `INCONCLUSIVE` 不得提升決策權重；`REGRESSED` 與 `INEFFECTIVE` 必須作為反面約束（Anti-Pattern）召回，防止重複提交已證實失敗的修法。

