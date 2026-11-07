# AIA-04：Test Plan, Acceptance Criteria & Release Specification

> [!NOTE]
> **架構重整與收攏公告 (Consolidated Architecture Specification)**
> 本規範為 AIA 歷史基準規範（規格編號已於 AIA-22 永久封頂）。為杜絕規格膨脹與重複定義，本規範之現行核心標準已正式收攏重整至 **[Volume 4: Self-Correction, Experiments & Benchmarks](file:///home/wade/src/github/aia/docs/specifications/volume-4-self-correction-and-experiments.md)**。最新架構與維護基準請以對應之 Volume 為主，完整六大核心卷軸總覽請參閱 [Specification Catalog (README.md)](file:///home/wade/src/github/aia/docs/specifications/README.md)。

| 項目 | 內容 |
|---|---|
| 文件編號 | AIA-04 |
| 文件名稱 | Test Plan, Acceptance Criteria & Release Specification |
| 版本 | 2.0 |
| 文件狀態 | Revised Baseline |
| 適用產品 | AIA — Autonomous Intelligence Architecture |
| 實作語言 | Go |
| Database | SQLite |
| Agent Harness | Antigravity CLI |
| 依據文件 | AIA-01 v2.0、AIA-02、AIA-03 v2.0、AIA-05 v1.1、AIA-06 v2.0、AIA-07 v2.0 |

---

# 1. 文件目的

本文件定義 AIA 的測試策略、功能驗收、自主性能力驗證、投資實驗評估及版本發布條件。

AIA 的主要研究目標，是建立具備以下四項能力的 Agent：

1. **Initiative：主動性思維**
2. **Continuity：時間連續性**
3. **Self-Correction：自我修正**
4. **Goal Evolution：目標維持、優化與修正**

因此，AIA 的測試不能只確認 API、CLI 或資料庫操作是否正常。

還必須驗證 Agent 是否能在長期執行過程中：

- 維持原始目標。
- 記住並正確使用過去的經驗。
- 觀察環境變化。
- 主動發現問題與機會。
- 自行判斷是否需要採取行動。
- 比較預測與實際結果。
- 發現錯誤並修正策略。
- 在適當情況下提出目標優化或修正。

本文件區分：

- 功能測試。
- 整合測試。
- Runtime 測試。
- 資料品質測試。
- Agent 行為測試。
- 自主性能力測試。
- Domain Experiment 評估。
- Release Acceptance。

# 2. 測試基本原則

## 2.1 不以程式存在判定功能完成

存在某個 Interface、Service、CLI Command 或 Database Table，不代表完整功能已完成。

例如：

| 已存在的元件 | 不代表已完成 |
|---|---|
| Schedule Service | 常駐排程執行 |
| Task Scheduler | Task Background Worker |
| Collector Provider | 真實資料來源整合 |
| Memory Repository | 時間連續性 |
| Prediction Evaluation | 自我修正 |
| Agent Registry | 自主 Agent |
| Web Server | 完整 Runtime Daemon |

測試必須驗證完整執行路徑，而不是只確認元件存在。

## 2.2 不以單次成功判定自主性

Agent 單次完成任務，不代表具備自主性。

自主性必須透過跨時間、跨任務、跨環境變化的連續測試驗證。

## 2.3 不以結果反推決策品質

例如投資：

- 獲利不必然代表判斷正確。
- 虧損不必然代表判斷錯誤。

應檢查決策當時的資訊、假設、風險、信心程度及預測結果。

## 2.4 保留歷史證據

所有重要測試都應保存：

- 測試輸入。
- 當時環境。
- Agent 狀態。
- 記憶內容。
- 決策紀錄。
- 執行結果。
- 評估結果。
- 修正紀錄。

不可只保存最後一次測試結果。

## 2.5 區分測試環境與正式資料

測試資料、模擬行情、虛構公告及 Mock Provider 必須明確標記。

不可將 Mock 測試結果當成真實市場研究成果。

# 3. Test Classification

| 測試類型 | 目的 |
|---|---|
| Unit Test | 驗證單一函式及 Domain Rule |
| Repository Test | 驗證資料存取及交易 |
| Service Test | 驗證業務流程 |
| Integration Test | 驗證跨元件整合 |
| Runtime Test | 驗證常駐 Worker 及生命週期 |
| Harness Test | 驗證 Agent Harness 執行結果 |
| Data Quality Test | 驗證資料來源與可信度 |
| Security Test | 驗證 Policy、Approval、Audit |
| Continuity Test | 驗證跨時間狀態延續 |
| Initiative Test | 驗證主動觀察與工作發現 |
| Self-Correction Test | 驗證錯誤辨識與策略修正 |
| Goal Evolution Test | 驗證目標維持與合理演化 |
| Domain Experiment | 驗證特定領域的長期表現 |

# 4. Test Environment

## 4.1 Local Development

主要開發環境：

- Go。
- SQLite。
- Antigravity CLI。
- Local Filesystem。
- Embedded Web Server。

## 4.2 Test Database

測試資料庫應與正式資料庫分離。

要求：

- 每個測試可建立獨立資料庫。
- 測試完成後可清除。
- 不得污染正式資料。
- Migration 必須可重複驗證。

## 4.3 Mock Provider

Mock Provider 可用於：

- Unit Test。
- Integration Test。
- 錯誤處理測試。
- 歷史資料重播。
- Agent 行為測試。

Mock 資料必須帶有明確識別。

## 4.4 External Provider

真實外部資料來源測試應分開執行。

應保存：

- Provider。
- API Endpoint。
- Request Time。
- Response Time。
- HTTP Status。
- Raw Response Reference。
- Parsing Result。
- Validation Result。

API Key 等秘密資訊不得寫入測試紀錄。

# 5. Core Functional Test

## 5.1 Application Lifecycle

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| APP-001 | App Initialization | 正確建立 DB、Repository、Service |
| APP-002 | Migration | 可正確建立及升級資料庫 |
| APP-003 | Startup Recovery | 可辨識需要恢復的工作 |
| APP-004 | Graceful Shutdown | 可正常釋放資源 |
| APP-005 | Reopen | 關閉後重新啟動，資料仍存在 |

## 5.2 Repository

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| REP-001 | Create | 資料正確建立 |
| REP-002 | Read | 資料正確讀取 |
| REP-003 | Update | 更新符合 Domain Rule |
| REP-004 | Delete | 刪除符合資料生命週期規則 |
| REP-005 | Transaction | 交易失敗時可正確回滾 |
| REP-006 | Concurrency | 並行操作不產生不可接受的資料錯誤 |
| REP-007 | Historical Integrity | 歷史決策不會被後續更新覆蓋 |

## 5.3 Agent Registry

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| AGT-001 | Static Registration | 預期 Agent 可被註冊 |
| AGT-002 | Duplicate ID | 重複識別可被拒絕 |
| AGT-003 | Capability | Capability 可正確識別 |
| AGT-004 | Agent Lookup | 可依識別取得 Agent |
| AGT-005 | Permission | Capability 不會自動取得權限 |

# 6. Runtime Test

本章是 AIA 由單次 CLI 工具轉變為持續運作平台的重要驗收依據。

## 6.1 Runtime Lifecycle

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| RT-001 | Runtime Start | Runtime Manager 可啟動 |
| RT-002 | Worker Start | 指定 Worker 可啟動 |
| RT-003 | Worker Stop | Worker 可正常停止 |
| RT-004 | Context Cancellation | Cancellation 可傳遞至執行中的工作 |
| RT-005 | Graceful Shutdown | 正在執行的工作依規則處理 |
| RT-006 | Worker Error | Worker 錯誤可被記錄 |
| RT-007 | Worker Health | 可取得 Worker 健康狀態 |
| RT-008 | Restart Recovery | Runtime 重啟後可恢復必要狀態 |

## 6.2 Task Worker

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| TW-001 | Poll Ready Tasks | 可取得 READY Task |
| TW-002 | Dispatch | Task 正確派發至 Agent |
| TW-003 | Execute | Agent 能執行指定任務 |
| TW-004 | Result Verification | 可檢查執行結果 |
| TW-005 | Soft Deny | 工具遭 soft-deny 時不可誤判成功 |
| TW-006 | Auth Failure | 授權失效進入 WAITING_FOR_AUTH |
| TW-007 | Resume | 重新授權後可恢復任務 |
| TW-008 | Approval | 需要核准時正確等待 |
| TW-009 | Retry | 失敗後依規則重試 |
| TW-010 | Duplicate Execution | 同一 Task 不會因 Worker 重複取得而無意間重複執行 |

## 6.3 Schedule Worker

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| SC-001 | Create Schedule | 可建立排程 |
| SC-002 | Due Detection | 可正確判斷到期時間 |
| SC-003 | Create Task | 到期 Schedule 能建立 Task |
| SC-004 | Execute Task | 建立的 Task 能進入 Task Worker |
| SC-005 | SKIP | 錯過的排程依規則略過 |
| SC-006 | RUN_ONCE | 錯過的排程只補執行一次 |
| SC-007 | CATCH_UP | 依定義補處理錯過的執行項目 |
| SC-008 | Idempotency | 相同 occurrence 不會重複建立 Task |
| SC-009 | Concurrent Worker | 多 Worker 不會重複派發 |
| SC-010 | Restart | Runtime 重啟後排程狀態正確 |

**特別驗收條件：**

`aia schedule tick` 成功，只能證明手動排程處理指令可執行。

必須確認 Schedule Worker、Task Worker 與 Task Execution 串接後，才可宣稱常駐排程功能完成。

# 7. Data Collection Test

## 7.1 Provider Connectivity

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| DP-001 | Provider Registration | Provider 正確註冊 |
| DP-002 | Real Connectivity | 真實 Provider 可連線 |
| DP-003 | HealthCheck | 能反映真實連線狀態 |
| DP-004 | Timeout | 超時可被處理 |
| DP-005 | Retry | 重試依規則執行 |
| DP-006 | HTTP Error | HTTP 錯誤可被辨識 |
| DP-007 | Parsing | 欄位解析錯誤不可靜默忽略 |

## 7.2 Data Integrity

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| DI-001 | Source Identity | 每筆資料可追溯來源 |
| DI-002 | Timestamp | 取得時間與資料日期分開保存 |
| DI-003 | Duplicate | 重複資料可辨識 |
| DI-004 | Validation | 異常數值可被識別 |
| DI-005 | Missing Data | 缺失資料不會被假值取代 |
| DI-006 | Mock Identification | Mock 資料有明確標記 |
| DI-007 | Fallback Identification | Fallback 資料有明確標記 |
| DI-008 | Snapshot Integrity | Snapshot 不混入未標記的模擬資料 |
| DI-009 | Save Failure | 儲存失敗會回報錯誤 |
| DI-010 | Stale Data | 過期資料可被識別 |

## 7.3 Provider Acceptance

Provider 必須逐一驗收，不得以 Manager 註冊成功代替資料來源驗收。

| Provider | 驗收條件 |
|---|---|
| TWSE | 真實 API、交易日期、欄位解析、資料驗證 |
| TPEx | 真實 API、交易日期、欄位解析、資料驗證 |
| FRED | 真實 API、必要憑證、資料日期與數值 |
| SEC | 真實申報資料、申報識別、來源及日期 |

目前 Provider 的固定值、虛構公告及硬編碼申報資料不得通過 Real Data Acceptance。

# 8. Security & Governance Test

## 8.1 Policy

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| SEC-001 | Permission Check | 未授權操作被拒絕 |
| SEC-002 | Policy Enforcement | Agent 不可繞過 Policy |
| SEC-003 | Capability Isolation | 不同 Agent 權限分離 |
| SEC-004 | Local Binding | Web Host 預設只監聽 127.0.0.1 |
| SEC-005 | Secret Protection | 不記錄 OAuth Token 或秘密資訊 |

## 8.2 Approval

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| AP-001 | Approval Required | 需核准操作正確進入等待 |
| AP-002 | Approval Granted | 核准後可繼續 |
| AP-003 | Approval Rejected | 拒絕後不可繼續 |
| AP-004 | Approval Audit | 核准過程可追溯 |
| AP-005 | Unauthorized Approval | 未授權者不可核准 |

## 8.3 Audit

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| AU-001 | Decision Record | 重要決策有紀錄 |
| AU-002 | Execution Record | 執行結果有紀錄 |
| AU-003 | Historical Integrity | 歷史紀錄不可被覆蓋 |
| AU-004 | Correction Trace | 修正可追溯至原始決策 |
| AU-005 | Policy Trace | 治理決策可追溯 |

# 9. Autonomy Core Test

本章是 AIA 的核心研究驗收項目。

Autonomy Core 不可因為能夠呼叫 LLM、建立 Task 或讀取 Memory，就被判定完成。

每項能力都需要獨立測試。

## 9.1 Initiative Test

目標：驗證 Agent 能否主動觀察、發現問題與機會，並決定是否採取行動。

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| INI-001 | Goal Awareness | 能辨識目前有效目標 |
| INI-002 | Environment Observation | 能取得目標所需環境資訊 |
| INI-003 | Change Detection | 能辨識與過去不同的環境狀態 |
| INI-004 | Opportunity Discovery | 能提出合理機會 |
| INI-005 | Risk Discovery | 能提出合理風險 |
| INI-006 | Autonomous Task Proposal | 能主動提出 Task |
| INI-007 | No-Action Decision | 能合理選擇不行動 |
| INI-008 | Priority Assessment | 能比較不同工作的優先順序 |
| INI-009 | Action Justification | 能說明行動原因及預期結果 |

**測試重點：**

給定相同目標，但提供不同環境狀態。

觀察 Agent 是否能提出不同的行動判斷。

如果 Agent 每次都只等待使用者下達指令，不能判定 Initiative 通過。

## 9.2 Continuity Test

目標：驗證 Agent 是否能跨越不同 Session 與 Runtime Restart 延續工作。

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| CON-001 | Goal Persistence | 重啟後仍能辨識有效目標 |
| CON-002 | Historical Recall | 能取得相關歷史資訊 |
| CON-003 | Task Continuity | 能延續未完成工作 |
| CON-004 | Decision Continuity | 能正確引用過去決策 |
| CON-005 | Context Reconciliation | 能辨識過去資訊與目前狀態的差異 |
| CON-006 | Long-term Consistency | 長期工作不會無故偏離目標 |
| CON-007 | Memory Provenance | 能追溯記憶來源 |
| CON-008 | State Recovery | Runtime 重啟後能恢復必要狀態 |

**測試方式：**

1. 建立一個長期目標。
2. 讓 Agent 執行多個 Task。
3. 保存中間決策。
4. 關閉 Runtime。
5. 經過一段時間後重新啟動。
6. 提供新的環境資料。
7. 觀察 Agent 是否能正確延續先前工作。

僅能檢索舊對話，不足以通過 Continuity Test。

## 9.3 Self-Correction Test

目標：驗證 Agent 是否能從自身判斷及執行結果中發現問題，提出修正並驗證改善。

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| SCF-001 | Prediction Record | 原始預測被保存 |
| SCF-002 | Outcome Comparison | 能比較預測與實際結果 |
| SCF-003 | Error Detection | 能辨識預測或執行偏差 |
| SCF-004 | Cause Analysis | 能提出有證據的原因分析 |
| SCF-005 | Strategy Proposal | 能提出修正方案 |
| SCF-006 | Strategy Versioning | 能保存新舊策略版本 |
| SCF-007 | Improvement Verification | 能檢驗修正後是否改善 |
| SCF-008 | Confidence Calibration | 能比較信心程度與實際準確度 |
| SCF-009 | No False Learning | 不會將偶然結果直接視為因果證據 |

**測試重點：**

Agent 必須能說明：

- 原先判斷是什麼。
- 當時依據哪些資訊。
- 實際結果如何。
- 偏差發生在哪裡。
- 為什麼需要修正。
- 新策略預期改善什麼。
- 如何驗證修正是否有效。

只修改 Prompt、重寫結論或降低信心程度，不足以通過 Self-Correction Test。

## 9.4 Goal Evolution Test

目標：驗證 Agent 是否能在持續維持原始方向的同時，合理提出目標優化或修正。

| ID | 測試項目 | 驗收條件 |
|---|---|---|
| GE-001 | Goal Persistence | 能持續維持有效目標 |
| GE-002 | Goal Optimization | 能提出更有效的達成方式 |
| GE-003 | Goal Revision Proposal | 能提出有根據的目標修正 |
| GE-004 | Goal Conflict Detection | 能發現目標之間的衝突 |
| GE-005 | Goal Versioning | 目標變更有版本紀錄 |
| GE-006 | Goal Rationale | 能說明變更理由 |
| GE-007 | Human Approval | 重大目標變更遵循核准流程 |
| GE-008 | Goal Outcome Evaluation | 能檢查目標調整的實際效果 |

**重要區分：**

Agent 不能因為短期困難就放棄目標。

也不能因為原始目標不再合理，仍然無條件堅持。

應能區分：

- 目標不變，但方法需要優化。
- 目標仍有效，但環境已改變。
- 原始目標與更高層級目標產生衝突。
- 原始目標已失去合理性，需要提出修正。

# 10. Long-term Autonomy Experiment

單元測試及整合測試只能確認元件行為。

AIA 必須另外進行長期自主性實驗。

## 10.1 Experiment Design

每個實驗應定義：

- Experiment ID。
- Domain。
- Initial Goal。
- Initial State。
- Initial Resources。
- Observation Frequency。
- Evaluation Interval。
- Expected Behaviors。
- Constraints。
- Success Criteria。
- Failure Criteria。

## 10.2 Experiment Log

每次自主性循環至少保存：

- Observation。
- Retrieved Memory。
- Current Goal。
- Environment Assessment。
- Discovered Opportunities。
- Discovered Risks。
- Proposed Tasks。
- Decision。
- Action。
- Verification。
- Reflection。
- Strategy Changes。
- Goal Changes。

## 10.3 Evaluation Dimensions

| 維度 | 評估問題 |
|---|---|
| Initiative | 是否能主動發現值得處理的事情？ |
| Continuity | 是否能跨時間維持同一工作脈絡？ |
| Self-Correction | 是否能從錯誤中修正自身？ |
| Goal Evolution | 是否能兼顧目標維持與合理優化？ |
| Reliability | 是否能穩定完成所選行動？ |
| Governance | 是否遵守 Policy 與 Approval？ |
| Traceability | 是否能回溯決策及修正過程？ |

# 11. Investment Agent Experiment Acceptance

Investment 是 AIA 第一個完整自主性實驗場景。

## 11.1 Initial Conditions

| 項目 | 設定 |
|---|---|
| Initial Capital | NT$100,000 |
| Market | Taiwan Stock Market |
| Special Focus | TSMC 2330 |
| Observation Frequency | 每個證券營業日 |
| Evaluation Frequency | 每月 1 日 |
| Primary Goal | 最大化投資組合總資產 |
| Execution Mode | Virtual Portfolio |

Agent 自行判斷：

- 是否買進。
- 是否賣出。
- 是否增持。
- 是否減持。
- 是否持有。
- 是否保留現金。
- 是否研究其他標的。

不要求每天交易。

不預先指定台積電必須持有，也不預先指定現金比例。

## 11.2 Portfolio Accounting

總資產應採一致估值規則：

\[
V_t = C_t + \sum_{i=1}^{n} Q_{i,t}P_{i,t}
\]

其中：

- \(V_t\)：時間 \(t\) 的投資組合總資產。
- \(C_t\)：現金。
- \(Q_{i,t}\)：第 \(i\) 檔持股股數。
- \(P_{i,t}\)：第 \(i\) 檔股票的估值價格。

若採用除權息調整、股利入帳及交易成本，必須明確記錄其計算方式。

月度檢查至少保存：

- 現金。
- 各標的股數。
- 各標的估值價格。
- 持股市值。
- 交易成本。
- 股利及其他現金流。
- 總資產。
- 累計損益。
- 報酬率。

## 11.3 Monthly Evaluation

每月 1 日檢查：

1. 當月期初總資產。
2. 當月期末總資產。
3. 累計總資產變化。
4. 現金比例。
5. 持股變化。
6. 交易次數。
7. 預測準確度。
8. 重大決策。
9. 失誤與修正。
10. 下一階段策略。

使用者的主要評估目標是總資產增加及長期最大化。

不得將交易次數或持股數量當成投資成功指標。

## 11.4 Performance Benchmark

建議額外記錄：

- 加權股價指數同期報酬。
- 0050 同期總報酬。

這些是比較基準，不取代使用者設定的唯一主要目標。

基準比較應納入股利及必要的成本調整，避免不同計算口徑造成誤判。

## 11.5 Decision Quality

每筆決策應能追溯：

- 當時持股。
- 當時現金。
- 當時市場資訊。
- 資料來源。
- Agent 判斷。
- 預期結果。
- 信心程度。
- 風險。
- 實際成交。
- 後續結果。
- 事後反思。
- 策略修正。

不得只用獲利或虧損判斷決策品質。

## 11.6 Experiment Success

實驗應同時評估：

**Financial Outcome**

投資組合總資產是否增加，以及相對基準的表現。

**Autonomy Performance**

Agent 是否能持續觀察、主動決策、記錄結果並修正策略。

**Operational Reliability**

Runtime 是否能依排程正常運作，並在中斷後恢復。

三項結果分開呈現，不得以其中一項掩蓋其他項目的失敗。

# 12. Coding & Writing Domain Acceptance

目前 Coding 與 Writing 尚未實作。

以下是未來驗收標準，不代表現有功能。

## 12.1 Coding Agent

應能：

- 維持長期 Project Goal。
- 理解專案結構。
- 追蹤修改歷史。
- 分析程式錯誤。
- 提出修正。
- 驗證修正效果。
- 避免重複犯錯。
- 根據專案演進優化架構。

## 12.2 Writing Agent

應能：

- 維持長期創作目標。
- 管理世界觀。
- 管理人物與關係。
- 管理卷次與章節進度。
- 維護時間線。
- 根據新史料修正內容。
- 發現人物、事件及制度矛盾。
- 保存每次修改的依據。

兩個 Domain 都必須使用共通 Autonomy Core，而非各自建立一套自主性引擎。

# 13. Web UI Acceptance

Web UI 的驗收應與 AIA-06 一致。

## 13.1 Core UI

- [ ] 顯示 Runtime 狀態。
- [ ] 顯示 Worker 狀態。
- [ ] 顯示 Task 狀態。
- [ ] 顯示 Schedule 狀態。
- [ ] 顯示 Agent 狀態。
- [ ] 顯示 Memory / Goal 基礎資訊。
- [ ] 顯示 Audit。
- [ ] 顯示尚未實作的能力，不得偽裝成已完成。

## 13.2 Autonomy Monitor

未來應顯示：

- Initiative Events。
- Observation History。
- Autonomous Task Proposals。
- Goal Changes。
- Self-Correction Records。
- Continuity Events。

## 13.3 Investment Workbench

- [ ] Portfolio。
- [ ] Market Data。
- [ ] Proposals。
- [ ] Predictions。
- [ ] Evaluation。
- [ ] Data Source Status。
- [ ] Decision History。
- [ ] Monthly Review。

Web UI 不得將手動操作顯示為自動背景作業。

# 14. Current Implementation Acceptance

依目前程式碼檢視結果，應採以下基準。

| 項目 | 目前狀態 |
|---|---|
| Go App Initialization | IMPLEMENTED |
| SQLite / Migration | IMPLEMENTED |
| Repository Layer | IMPLEMENTED |
| Agent Static Registration | IMPLEMENTED |
| Web Host | IMPLEMENTED |
| Web Module Isolation | IMPLEMENTED |
| Task Execution Logic | PARTIAL |
| Schedule Service | PARTIAL |
| Persistent Runtime Worker | NOT IMPLEMENTED |
| Schedule Polling Worker | NOT IMPLEMENTED |
| Task Polling Worker | NOT IMPLEMENTED |
| Schedule-to-Task Execution | PARTIAL |
| Investment Service | PARTIAL |
| Collector Framework | PARTIAL |
| Real Provider Integration | PARTIAL / NOT VERIFIED |
| Prediction Evaluation | PARTIAL |
| Continuity Engine | NOT VERIFIED |
| Initiative Engine | NOT IMPLEMENTED |
| Self-Correction Engine | NOT IMPLEMENTED |
| Goal Evolution Engine | NOT IMPLEMENTED |
| Coding Module | NOT IMPLEMENTED |
| Writing Module | NOT IMPLEMENTED |

這是目前的實作基準，不是未來 Release 的通過清單。

# 15. Release Strategy

AIA 採分階段 Release。

每個階段都必須有清楚的交付條件。

## R0：Current Baseline

目標：建立可辨識的現況基準。

交付：

- Go Runtime 基礎。
- SQLite。
- CLI。
- Web Host。
- Agent Registry。
- Investment 基礎模組。
- 文件一致性。

不得宣稱完整自主 Runtime 已完成。

## R1：Persistent Runtime

交付：

- Runtime Manager。
- Worker Lifecycle。
- Graceful Shutdown。
- Worker Health。
- Task Polling。
- Schedule Polling。
- Recovery。

## R2：Reliable Investment Data

交付：

- 真實 TWSE。
- 真實 TPEx。
- 真實 FRED。
- 真實 SEC（依 Investment 實際需求）。
- Data Validation。
- Snapshot Integrity。
- Mock / Real 分離。

## R3：Autonomous Investment

交付：

- 每個證券營業日觀察。
- 主動建立研究工作。
- 投資決策與紀錄。
- Prediction。
- Evaluation。
- 策略反思與修正。
- 每月資產檢查。

## R4：Autonomy Core

交付：

- Continuity。
- Initiative。
- Self-Correction。
- Goal Evolution。
- Autonomy Monitor。

## R5：Domain Expansion

交付：

- Coding Agent。
- Writing Agent。
- 共用 Autonomy Core。
- 長期 Project Continuity。

# 16. Release Gate

每次 Release 必須通過以下檢查。

## 16.1 Functional Gate

- [ ] 主要功能符合規格。
- [ ] 核心流程完整。
- [ ] 錯誤處理正常。
- [ ] 資料持久化正常。

## 16.2 Runtime Gate

- [ ] Worker 可啟動及停止。
- [ ] Background Execution 正常。
- [ ] Restart Recovery 正常。
- [ ] Schedule 與 Task 執行完整串接。

## 16.3 Security Gate

- [ ] Policy 正常。
- [ ] Approval 正常。
- [ ] Audit 完整。
- [ ] Local Only。
- [ ] 無未授權操作。

## 16.4 Data Gate

- [ ] 真實資料來源已驗證。
- [ ] Mock 資料明確隔離。
- [ ] 資料品質檢查正常。
- [ ] Snapshot 可追溯。

## 16.5 Autonomy Gate

依 Release 階段要求：

- [ ] Initiative 測試通過。
- [ ] Continuity 測試通過。
- [ ] Self-Correction 測試通過。
- [ ] Goal Evolution 測試通過。

未達成的能力不得宣稱已完成。

## 16.6 Documentation Gate

- [ ] AIA-01 至 AIA-07 版本一致。
- [ ] README 與程式現況一致。
- [ ] CLI 說明與實際指令一致。
- [ ] 未實作功能明確標記。
- [ ] Release Notes 完整。

# 17. Metrics & Evaluation

自主性能力不能只用主觀印象評估。

應逐步建立量測指標。

| 能力 | 觀察指標 |
|---|---|
| Initiative | 主動發現事件、有效 Task 提案、合理不行動比例 |
| Continuity | 目標延續率、歷史資訊正確使用率、恢復成功率 |
| Self-Correction | 錯誤辨識率、修正完成率、修正後改善程度 |
| Goal Evolution | 合理優化提案、目標版本品質、修正後成效 |
| Reliability | 任務完成率、錯誤率、恢復成功率 |
| Governance | Policy 違規次數、未授權執行次數 |
| Traceability | 決策紀錄完整度、來源可追溯率 |

所有比率必須明確定義分子、分母、觀察期間及資料來源。

例如：

\[
\text{Task Completion Rate}
=
\frac{\text{成功完成的有效 Task}}
{\text{已進入執行的有效 Task}}
\]

\[
\text{Correction Improvement}
=
\text{修正後表現}
-
\text{修正前表現}
\]

不同 Domain 的改善指標不可直接混為一談。

# 18. 最終驗收原則

AIA 的成功不能只用「程式能正常執行」判斷。

應分成四個層次：

1. **Infrastructure**：Runtime、Database、Worker、Harness 是否可靠。
2. **Capability**：Agent 是否具備持續記憶、主動性及自我修正能力。
3. **Outcome**：Agent 是否在特定領域達成設定目標。
4. **Governance**：整個過程是否安全、可追溯、可檢驗。

其中，Outcome 不應取代 Capability。

例如投資 Agent 即使短期獲利，也不能因此認定它已具備自主性；即使短期虧損，也不能單憑結果否定其自主性。

真正重要的是：

**Agent 是否能理解自己正在追求什麼、為什麼採取某個行動、行動後發生了什麼，以及下一次是否應該改變自己的判斷與方法。**

這才是 AIA 應該持續驗證的核心。
