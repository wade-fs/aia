# AIA-19：Deployment, Operations & Observability Specification

**文件編號：** AIA-19  
**文件名稱：** Deployment, Operations & Observability Specification  
**版本：** 2.0  
**文件狀態：** Operational Specification  
**適用系統：** AIA (Autonomous Intelligence Architecture)  
**依賴文件：** AIA-01 ～ AIA-18  
**核心範圍：** Deployment、Service Lifecycle、Operations、Observability、Monitoring、Logging、Recovery、Resource Management

---

# 1. 文件目的

## 1.1 設計背景

AIA 的目標不是執行一次性 AI 任務，而是建立能夠長期運作的自主 Agent 平台。

這代表 AIA 必須面對一般 AI 應用程式較少處理的問題：

- Runtime 長時間運作。
- Agent 跨 Session 維持目標。
- Worker 持續執行不同工作。
- Experiment 跨越數週、數月甚至更長時間。
- Memory 與 Evidence 持續累積。
- AI Model 與 Harness 可能中斷。
- 外部資料來源可能失效。
- Agent 可能陷入重複思考或無效行動。
- 自我修正可能造成新問題。
- 系統必須在故障後恢復原有工作。

因此，AIA 需要完整的部署、運作與可觀測性規格。

## 1.2 文件目標

本文件定義：

1. AIA 部署架構。
2. Service Lifecycle。
3. Runtime 長期運作方式。
4. Process 與 Worker 管理。
5. Configuration Management。
6. Logging。
7. Metrics。
8. Distributed Tracing 與執行關聯追蹤。
9. Autonomous Behavior Monitoring。
10. Resource Management。
11. Health Check。
12. Alerting。
13. Backup 與 Restore。
14. Recovery。
15. Upgrade 與 Rollback。
16. Operational Security。
17. Go Runtime 實作方向。
18. 驗收標準。

## 1.3 設計原則

AIA 的維運設計必須符合：

- Local First。
- Single User。
- Long-running。
- Persistent State。
- Recoverable。
- Observable。
- Auditable。
- Resource Controlled。
- Least Privilege。
- Human Controllable。

維運系統不能只確認 Process 還活著，還必須確認自主能力是否正常運作。

---

# 2. Deployment Architecture

## 2.1 Initial Deployment Model

AIA v1 採用 Local Deployment。

初期建議：

- 單一主機。
- 單一使用者。
- 單一主要 Runtime Process。
- 本地資料儲存。
- 本地 CLI。
- Local Web Console。
- 外部 AI Harness。
- 按需求連接外部資料來源。

```text
+------------------------------------------------------+
|                  User Workstation                    |
|                                                      |
|  +----------------+    +--------------------------+  |
|  | AIA CLI        |    | AIA Web Console          |  |
|  +-------+--------+    +------------+-------------+  |
|          |                          |                |
|          +-------------+------------+                |
|                        |                             |
|                        v                             |
|              +---------------------+                 |
|              | AIA Runtime         |                 |
|              |                     |                 |
|              | Autonomy Core       |                 |
|              | Agent Registry      |                 |
|              | Task / Worker       |                 |
|              | Project / Experiment|                 |
|              | Memory / Evidence   |                 |
|              | Governance / Audit  |                 |
|              +----------+----------+                 |
|                         |                            |
|             +-----------+-----------+                |
|             |                       |                |
|             v                       v                |
|      +-------------+         +-------------+         |
|      | Local Store |         | AIA Logs    |         |
|      +-------------+         +-------------+         |
|                         |                            |
|                         v                            |
|                 +---------------+                    |
|                 | agy Harness   |                    |
|                 +---------------+                    |
+------------------------------------------------------+
                         |
                         v
                External Data Sources
```

這是邏輯架構，不要求每個元件獨立部署。

## 2.2 Single Process First

初期不建議將 AIA 拆成多個微服務。

理由：

1. 單一使用者的工作負載通常不需要分散式架構。
2. 跨模組交易與狀態一致性較容易管理。
3. 部署、除錯與備份更簡單。
4. 可降低維運成本。
5. 有利於先驗證自主能力本身。

即使所有核心元件位於同一個 Go Process，仍應保持清楚的模組邊界。

## 2.3 Process Separation

未來若有必要，可以將以下工作拆分：

- Runtime。
- Web Server。
- Background Worker。
- Data Collection。
- Evaluation。
- Notification。

但拆分必須有明確理由，例如：

- 資源隔離。
- 故障隔離。
- 不同生命週期。
- 不同安全權限。
- 工作負載差異。

不能單純為了追求微服務而拆分。

## 2.4 Network Boundary

維持 AIA v1 的 Local Only 原則：

- Web Server 預設監聽 `127.0.0.1`。
- 不預設監聽 `0.0.0.0`。
- 不預設提供 Internet 公開服務。
- 不允許未授權的遠端控制。
- 外部資料連線由明確的 Provider 或 Adapter 管理。

若未來需要 LAN 或 Internet 存取，必須另外制定安全設計。

---

# 3. Service Lifecycle

## 3.1 Service Definition

AIA Runtime 是主要常駐服務。

它負責：

- 初始化系統。
- 恢復持久化狀態。
- 啟動 Event Processing。
- 管理 Task。
- 管理 Worker。
- 喚醒 Autonomy Core。
- 維護 Agent Lifecycle。
- 處理 Shutdown。
- 處理 Recovery。

## 3.2 Startup Sequence

啟動流程：

```text
Process Start
     |
     v
Load Configuration
     |
     v
Initialize Logging
     |
     v
Initialize Storage
     |
     v
Load Governance Policy
     |
     v
Load Agent Registry
     |
     v
Recover Persistent State
     |
     v
Validate Pending Tasks
     |
     v
Validate Pending Approvals
     |
     v
Initialize Event Processing
     |
     v
Start Worker Management
     |
     v
Start Scheduler
     |
     v
Start Web Interface
     |
     v
Runtime Ready
```

啟動順序必須確保治理與持久化服務先於自主工作。

不能在 Policy 尚未載入時就開始執行 Agent。

## 3.3 Readiness

Process 啟動不代表系統已經可以執行自主工作。

Readiness 至少檢查：

- Storage 可用。
- Configuration 有效。
- Governance Policy 已載入。
- Agent Registry 可用。
- Runtime Event Processing 可用。
- Recovery 完成。
- 核心依賴服務已初始化。

外部 AI Provider 暫時不可用，不一定代表整個 AIA 無法啟動。

應區分系統本身可用與特定能力暫時不可用。

## 3.4 Shutdown

收到正常停止訊號後：

1. 停止接收新的自主執行要求。
2. 停止建立新的非必要 Task。
3. 依政策處理進行中的 Task。
4. 通知 Worker 進行 Graceful Shutdown。
5. 保存 Task 與 Worker 狀態。
6. 保存 Project 與 Experiment 狀態。
7. Flush Logs。
8. 關閉外部連線。
9. 關閉 Storage。
10. 結束 Process。

Shutdown 不得直接刪除未完成 Task。

## 3.5 Crash Recovery

若 Process 非正常結束：

- 重新載入持久化狀態。
- 檢查未完成 Task。
- 檢查 Worker 執行狀態。
- 檢查最後一次 Execution Record。
- 檢查是否存在未完成的外部操作。
- 檢查授權是否仍有效。
- 判斷可安全 Resume、Retry、Pause 或人工處理。

不能假設 Process 中斷前的最後一個動作一定沒有成功。

例如虛擬交易可能已經入帳，但 Task 尚未更新完成狀態。恢復時必須透過交易識別碼與持久化紀錄確認，避免重複入帳。

---

# 4. Runtime & Worker Operations

## 4.1 Runtime Responsibilities

Runtime 負責：

- Event Loop。
- Scheduler。
- Task Dispatcher。
- Worker Manager。
- Session Manager。
- Recovery Manager。
- Resource Manager。
- Health Manager。

Runtime 必須能持續運作，但不代表所有 Worker 都必須長駐。

## 4.2 Worker Lifecycle

建議 Worker 狀態：

| 狀態 | 意義 |
|---|---|
| CREATED | 已建立 |
| READY | 等待工作 |
| RUNNING | 執行中 |
| WAITING | 等待依賴或外部條件 |
| PAUSED | 暫停 |
| COMPLETED | 完成 |
| FAILED | 失敗 |
| TERMINATED | 已終止 |

Worker 狀態必須持久化，或能透過 Execution Record 安全重建。

## 4.3 Worker Isolation

不同 Worker 應有明確的：

- 工作範圍。
- Workspace。
- Tool Permission。
- Resource Limit。
- Timeout。
- Cancellation。
- Output Contract。

Worker 不應因為屬於同一個 Agent，就自動取得其他 Worker 的所有權限。

## 4.4 Worker Concurrency

Concurrency 應由 Runtime 管理。

限制條件包括：

- 全域 Worker 上限。
- Agent Worker 上限。
- Task 類型上限。
- 外部 Provider 限流。
- CPU 與 Memory。
- Model API 配額。
- Workspace 衝突。

多個 Worker 同時修改同一份檔案時，必須採取 Workspace Isolation、Lock 或其他衝突控制方式。

## 4.5 Worker Stuck Detection

系統應能辨識：

- 長時間沒有輸出。
- 重複呼叫相同工具。
- 重複執行相同步驟。
- 無進度變化。
- 超過預期執行時間。
- 持續產生相同錯誤。

發現異常時，不應立即將其視為普通失敗。

可以先要求 Worker 回報狀態、暫停、取消或重新評估。

---

# 5. Configuration Management

## 5.1 Configuration Categories

設定分為：

1. System Configuration。
2. Runtime Configuration。
3. Agent Configuration。
4. Worker Configuration。
5. Domain Configuration。
6. Provider Configuration。
7. Governance Configuration。
8. Resource Configuration。

## 5.2 Configuration Source

建議優先順序：

```text
Built-in Defaults
      |
      v
Configuration Files
      |
      v
Environment Overrides
      |
      v
Explicit Runtime Options
```

敏感憑證不應直接寫入一般設定檔。

## 5.3 Configuration Validation

啟動與設定變更時必須檢查：

- 欄位型別。
- 必要欄位。
- 數值範圍。
- 相依設定。
- 路徑有效性。
- 權限設定。
- Provider 可用性。

無效設定不得靜默套用預設值，尤其是安全與治理設定。

## 5.4 Configuration Versioning

重要設定應有版本紀錄。

保存：

- Configuration ID。
- Version。
- Changed Fields。
- Previous Value。
- New Value。
- Changed By。
- Change Reason。
- Timestamp。

涉及 Policy、Permission、Resource Limit 的修改應留下 Audit。

## 5.5 Dynamic Reload

只允許明確標記為可動態更新的設定進行 Reload。

例如：

- 部分通知設定。
- 非安全性 UI 設定。
- 可安全更新的排程參數。

以下設定通常應要求受控重啟或核准：

- Network Binding。
- Security Policy。
- Authorization。
- Harness Permission。
- 核心儲存設定。

---

# 6. Logging Specification

## 6.1 Logging Objectives

Logging 用於：

- 除錯。
- 事件還原。
- 執行追蹤。
- 故障分析。
- 安全稽核。
- 自主行為分析。

Log 不能取代 Audit，也不能取代 Evidence。

## 6.2 Log Levels

| Level | 用途 |
|---|---|
| DEBUG | 詳細除錯資訊 |
| INFO | 正常系統事件 |
| WARN | 異常但可繼續運作 |
| ERROR | 操作失敗 |
| FATAL | 系統無法安全繼續運作 |

## 6.3 Structured Logging

建議使用結構化 JSON Log。

基本欄位：

```json
{
  "timestamp": "2026-11-01T09:00:00+08:00",
  "level": "INFO",
  "component": "runtime",
  "event": "task_started",
  "task_id": "task-001",
  "agent_id": "investment",
  "worker_id": "worker-001",
  "session_id": "session-001",
  "trace_id": "trace-001",
  "message": "Task execution started"
}
```

以上為資料格式示例。

實際欄位依事件類型擴充。

## 6.4 Correlation IDs

以下識別碼應在相關事件中傳遞：

- trace_id。
- session_id。
- project_id。
- experiment_id。
- goal_id。
- decision_id。
- task_id。
- worker_id。
- agent_id。
- execution_id。

這使一次自主決策的觀察、判斷、執行、驗證與學習能夠串聯。

## 6.5 Sensitive Data

Log 不應保存：

- OAuth Token。
- API Secret。
- Password。
- Private Key。
- 未經必要性評估的個人資料。
- 不必要的完整文件內容。

錯誤訊息也必須避免意外洩漏敏感資訊。

## 6.6 Log Retention

Log 應具有保留與輪替機制。

可依：

- 檔案大小。
- 保存天數。
- 儲存空間。
- Log 類型。

執行 Rotation 與 Archive。

重要 Audit 不得因一般 Log Rotation 而刪除。

---

# 7. Metrics Specification

## 7.1 Metrics Categories

AIA 至少需要五類 Metrics：

1. Runtime Metrics。
2. Execution Metrics。
3. Resource Metrics。
4. Autonomy Metrics。
5. Domain Metrics。

## 7.2 Runtime Metrics

| Metric | 說明 |
|---|---|
| runtime_uptime | Runtime 運作時間 |
| runtime_restart_count | 重啟次數 |
| active_sessions | 活躍 Session |
| active_tasks | 執行中 Task |
| pending_tasks | 等待 Task |
| active_workers | 活躍 Worker |
| event_queue_depth | 事件佇列深度 |
| recovery_count | 恢復次數 |

## 7.3 Execution Metrics

| Metric | 說明 |
|---|---|
| task_success_rate | Task 成功率 |
| task_failure_rate | Task 失敗率 |
| task_duration | Task 執行時間 |
| worker_retry_count | Worker 重試次數 |
| harness_failure_rate | Harness 失敗率 |
| tool_soft_deny_count | 工具 Soft-deny 次數 |
| approval_wait_time | 等待核准時間 |

## 7.4 Resource Metrics

| Metric | 說明 |
|---|---|
| process_cpu_usage | CPU 使用量 |
| process_memory_usage | Memory 使用量 |
| storage_usage | 儲存空間 |
| model_token_usage | Token 使用量 |
| provider_request_count | Provider 請求數 |
| provider_error_rate | Provider 錯誤率 |
| external_data_usage | 外部資料取得量 |

## 7.5 Autonomy Metrics

這是 AIA 與一般服務監控最大的差異。

| Metric | 說明 |
|---|---|
| candidate_actions_generated | 產生的候選行動數 |
| autonomous_decisions | 自主決策數 |
| no_action_decisions | 不行動決策數 |
| initiative_precision | 主動發現事項的有效程度 |
| goal_persistence | 長期目標維持程度 |
| goal_revision_count | 目標修訂次數 |
| correction_success_rate | 修正成功率 |
| repeated_failure_rate | 重複失敗率 |
| context_reconstruction_success | 脈絡恢復成功率 |
| learning_verification_rate | 學習紀錄驗證比例 |

這些 Metrics 必須與 AIA-13 的評估定義一致。

不能將自主決策數量直接視為自主能力高低。

## 7.6 Domain Metrics

Investment：

- Portfolio Value。
- Cash Balance。
- Unrealized P/L。
- Realized P/L。
- Drawdown。
- Turnover。
- Benchmark Comparison。
- Prediction Calibration。

Coding：

- Defect Detection。
- Correction Success。
- Regression Rate。
- Test Coverage。
- Technical Debt Trend。
- Repeated Defect Rate。

Writing：

- Timeline Conflict Count。
- Historical Claim Verification。
- Character Consistency Issues。
- Unresolved Research Questions。
- Revision Acceptance。
- Cross-Volume Consistency。

Domain Metrics 應反映領域實際成果，不可互相混用。

---

# 8. Distributed Tracing & Execution Trace

## 8.1 Purpose

AIA 不一定需要真正的分散式部署，但需要跨模組追蹤能力。

一次自主循環可能經過：

```text
Runtime Wakeup
    |
    v
Observation
    |
    v
Context Reconstruction
    |
    v
Assessment
    |
    v
Candidate Discovery
    |
    v
Deliberation
    |
    v
Decision
    |
    v
Governance
    |
    v
Task Dispatch
    |
    v
Worker Execution
    |
    v
Harness
    |
    v
Verification
    |
    v
Learning
```

每個階段都應能透過相同 Trace ID 關聯。

## 8.2 Span

每個重要操作可以建立 Span。

例如：

- autonomy.observe。
- autonomy.assess。
- autonomy.discover。
- autonomy.deliberate。
- autonomy.decide。
- governance.evaluate。
- runtime.dispatch。
- worker.execute。
- harness.invoke。
- execution.verify。
- learning.record。

## 8.3 Trace Requirements

必須能回答：

- 哪個事件喚醒 Agent？
- Agent 當時載入了哪些脈絡？
- 使用哪些 Evidence？
- 產生哪些候選行動？
- 為什麼選擇某個行動？
- 哪個 Policy 允許或拒絕？
- 哪個 Worker 執行？
- 呼叫了哪些工具？
- 最後結果如何？
- 是否產生 Learning？

Trace 不應記錄不必要的完整模型思考內容。

應保存可稽核的結構化 Decision、Evidence 與執行結果。

---

# 9. Autonomous Behavior Monitoring

## 9.1 為什麼需要自主性監控

一般監控系統關心：

- Process 是否存活？
- API 是否可用？
- CPU 是否過高？
- Memory 是否不足？

AIA 還必須關心：

- Agent 是否仍在追求原始目標？
- 是否忘記長期 Project？
- 是否反覆執行相同 Task？
- 是否陷入無限修正？
- 是否一直提出沒有價值的 Candidate Actions？
- 是否將錯誤資料當成事實？
- 是否持續產生無效 Learning？
- 是否過度依賴人類？
- 是否在沒有必要時大量消耗資源？

## 9.2 Autonomy Anomaly

定義以下異常：

### A. Repeated Action Loop

Agent 在相同條件下重複提出或執行相同操作，沒有新證據或策略變化。

### B. Endless Deliberation

Agent 持續分析，但無法形成 Decision。

### C. Task Explosion

Agent 不斷產生新 Task，卻沒有完成或改善原始目標。

### D. Goal Drift

Agent 的實際行動逐漸偏離原始 Goal，卻沒有正式提出 Goal Revision。

### E. Memory Drift

Agent 的長期 Memory 與最新驗證資訊不一致。

### F. False Learning

Agent 將偶然成功或失敗錯誤歸因為策略有效或無效。

### G. Resource Spiral

Agent 因反覆分析、重試或修正，持續消耗大量資源。

### H. Silent Failure

Runtime 表面正常，但自主循環長期沒有產生有效評估或決策。

## 9.3 Detection

可以採用：

- 重複 Task 指紋。
- 相似 Decision 比較。
- State Progress 檢查。
- Goal Alignment 檢查。
- Retry 次數。
- Token Budget。
- 長時間無進展偵測。
- Evidence 更新檢查。
- Evaluation Trend。

## 9.4 Response

異常發生時，系統可以：

- 記錄 Warning。
- 暫停相關 Worker。
- 限制重試。
- 暫停 Experiment。
- 要求重新評估。
- 要求人工檢查。
- 停止特定 Agent。

不得因為監控發現異常，就直接自動解除治理限制或修改核心 Policy。

---

# 10. Resource Management

## 10.1 Resource Categories

AIA 應管理：

- CPU。
- Memory。
- Disk。
- Network。
- Model API。
- Token。
- Worker Slots。
- External Provider Quota。
- Execution Time。

## 10.2 Resource Budget

可以依層級配置：

- Global Budget。
- Agent Budget。
- Project Budget。
- Experiment Budget。
- Task Budget。

例如 Investment Agent 與 Coding Agent 不應無限制競爭同一份模型使用額度。

## 10.3 Priority Management

資源優先順序應考慮：

- Criticality。
- Goal Importance。
- Deadline。
- Risk。
- Resource Cost。
- Dependencies。

不能單純依照 Task 建立時間決定所有工作順序。

## 10.4 Budget Exhaustion

資源不足時：

1. 停止建立非必要 Task。
2. 保存目前狀態。
3. 延後低優先級工作。
4. 保留必要監控。
5. 記錄 Budget Exhaustion。
6. 必要時通知使用者。

Agent 不得自行提高資源上限或繞過限制。

## 10.5 Token Management

應保存：

- Model。
- Provider。
- Input Tokens。
- Output Tokens。
- Request Count。
- Execution Context。
- Estimated Cost（若可取得）。

避免單一 Agent 因無效循環耗盡整體資源。

---

# 11. Health Check & Alerting

## 11.1 Health Categories

區分：

- Liveness。
- Readiness。
- Dependency Health。
- Execution Health。
- Autonomy Health。

## 11.2 Liveness

確認 Runtime Process 是否仍然運作。

Liveness 不代表自主功能正常。

## 11.3 Readiness

確認核心依賴是否已完成初始化，系統是否能安全接受工作。

## 11.4 Dependency Health

檢查：

- Storage。
- Harness。
- Model Provider。
- External Data Provider。
- File System。
- Network。

外部依賴失效應盡可能採取局部降級，而不是讓整個 Runtime 無條件退出。

## 11.5 Autonomy Health

檢查：

- Autonomy Cycle 是否正常。
- 是否能完成 Assessment。
- 是否有持續的狀態更新。
- 是否出現無限循環。
- 是否能正確保存 Decision。
- 是否能跨 Session 恢復。

## 11.6 Alert Levels

| Level | 意義 | 處理 |
|---|---|---|
| INFO | 一般事件 | 記錄 |
| WARNING | 異常但可繼續 | 觀察 |
| ERROR | 部分功能失效 | 通知或暫停 |
| CRITICAL | 系統安全或資料完整性受到威脅 | 停止相關操作並通知 |

## 11.7 Alert Fatigue

相同問題不應反覆發送大量通知。

應支援：

- Alert Deduplication。
- Grouping。
- Cooldown。
- Escalation。
- Resolution Notification。

---

# 12. Backup & Restore

## 12.1 Backup Scope

至少備份：

- Configuration。
- Project。
- Goal。
- Experiment。
- Task。
- Decision。
- Event History。
- Memory。
- Evidence。
- Knowledge。
- Audit。
- Approval History。
- Agent Configuration。

## 12.2 Backup Strategy

建議：

- 定期完整備份。
- 重要資料變更後備份。
- 升級前備份。
- 大型 Self-Correction 前備份。
- 重大 Experiment 里程碑備份。

## 12.3 Backup Integrity

備份應包含：

- Backup ID。
- Creation Time。
- AIA Version。
- Schema Version。
- Included Data。
- Checksum。
- Backup Status。

## 12.4 Restore

恢復時：

1. 停止 Runtime。
2. 確認備份完整。
3. 恢復資料。
4. 驗證 Schema。
5. 驗證 Event History。
6. 驗證 Audit。
7. 驗證重要資料關聯。
8. 恢復 Configuration。
9. 檢查 Policy。
10. 啟動 Runtime。
11. 執行 Recovery Validation。

## 12.5 Recovery Point & Recovery Time

可以定義：

- RPO：Recovery Point Objective，容許資料遺失時間。
- RTO：Recovery Time Objective，容許恢復時間。

初期由使用者設定可接受值，不應在缺乏實際硬體與資料量測試時任意宣稱已達成特定 RPO/RTO。

## 12.6 Audit Preservation

一般備份與還原不得任意刪除或改寫 Audit 歷史。

若因資料損毀需要修復，必須保留修復事件與來源紀錄。

---

# 13. Upgrade & Rollback

## 13.1 Upgrade Scope

升級可能涉及：

- AIA Binary。
- Configuration。
- Database Schema。
- Agent Definition。
- Worker Definition。
- Prompt。
- Policy。
- Harness Adapter。
- Domain Module。

## 13.2 Upgrade Procedure

1. 檢查目前版本。
2. 檢查升級相容性。
3. 停止新工作。
4. 保存進行中的 Task。
5. 備份資料。
6. 執行 Schema Migration。
7. 安裝新版本。
8. 驗證 Configuration。
9. 啟動 Runtime。
10. 執行 Health Check。
11. 驗證既有 Project 與 Experiment。
12. 確認升級成功。

## 13.3 Rollback

若升級失敗：

- 停止新版本。
- 保存故障紀錄。
- 恢復相容的程式版本。
- 必要時恢復資料備份。
- 檢查 Schema 相容性。
- 驗證 Audit 與狀態完整性。

不能假設程式 Binary 回退就能自動回復資料庫 Schema。

## 13.4 Self-Modification Integration

Coding Agent 若修改 AIA：

- 必須使用受控分支或隔離 Workspace。
- 保存 Diff。
- 執行必要驗證。
- 取得規定的核准。
- 保留可回滾版本。
- 不得直接覆蓋正在運作的核心 Runtime。

實際修改流程依 AIA-14 規格執行。

---

# 14. Operational Security

## 14.1 Local Service Security

- Web Server 預設只監聽 Loopback。
- 不公開未授權管理介面。
- 使用適當的 Session 與 CSRF 防護。
- 敏感設定不進入一般 Log。
- 保護本地資料目錄。

## 14.2 Credential Handling

AIA 不得讀取或複製使用者透過 agy 完成的 Google OAuth Token。

Harness 的登入狀態由使用者與 Harness 自行管理。

若認證失效：

- Task 進入 WAITING_FOR_AUTH。
- 通知使用者。
- 使用者完成登入。
- Runtime 重新檢查 Harness 狀態。
- 由使用者或授權流程 Resume。

## 14.3 Permission Monitoring

系統應記錄：

- Permission Denied。
- Approval Required。
- Approval Granted。
- Approval Rejected。
- Tool Soft-deny。
- Authorization Expired。
- Permission Configuration Changed。

## 14.4 Audit

以下事件應保留：

- 使用者登入與重要管理操作。
- Policy 變更。
- Authorization 變更。
- Approval。
- 高風險 Decision。
- Self-Correction。
- 核心程式修改。
- Experiment 停止。
- Emergency Stop。
- Backup 與 Restore。

Audit 不得由一般 Agent 或 Worker 修改。

---

# 15. Operations Dashboard

## 15.1 Dashboard Objectives

Web Console 應讓使用者快速了解：

1. AIA 是否正常運作？
2. 哪些 Agent 正在工作？
3. 哪些 Project 有進展？
4. 哪些 Experiment 正在進行？
5. 有沒有異常？
6. 有沒有需要核准的事項？
7. 自主能力是否持續改善？
8. 資源是否即將耗盡？

## 15.2 Dashboard Sections

### System Overview

- Runtime Status。
- Uptime。
- Version。
- Storage Status。
- Harness Status。
- Resource Usage。

### Agent Overview

- Agent Status。
- Active Worker。
- Current Goal。
- Current Project。
- Last Activity。
- Autonomy Health。

### Project & Experiment

- Active Projects。
- Active Experiments。
- Progress。
- Milestones。
- Pending Evaluation。

### Governance

- Pending Approval。
- Recent Denials。
- Policy Changes。
- High-risk Actions。

### Observability

- Recent Errors。
- Retry。
- Stuck Worker。
- Repeated Action Loop。
- Resource Warnings。

### Autonomy Evaluation

- Initiative Trend。
- Continuity。
- Self-Correction。
- Goal Evolution。
- Long-term Experiment Results。

## 15.3 Dashboard Is Not the Decision Engine

Dashboard 只負責呈現與人工控制。

不能將自主決策邏輯寫在 UI。

---

# 16. Long-running Operations

## 16.1 Long-term Experiment

長期 Experiment 必須具備：

- 明確目標。
- 目前狀態。
- 執行歷史。
- 評估週期。
- 資源預算。
- 失敗處理。
- 暫停與恢復。
- 結束條件。

## 16.2 Periodic Review

AIA 應依 Experiment 設定不同檢查週期。

例如：

| Domain | 觀察週期 | 評估週期 |
|---|---|---|
| Investment | 每個證券營業日 | 每月 1 日 |
| Coding | 依 Repository Event 與工作狀態 | 依里程碑或重大修正 |
| Writing | 依 Project 變更與研究進度 | 依章節、卷或里程碑 |

這些週期是目前實驗的操作要求，不代表 Autonomy Core 只能在排程時運作。

## 16.3 Long-term Drift Detection

應定期檢查：

- Goal 是否偏移。
- Project 是否長期停滯。
- Experiment 是否失去驗證價值。
- Knowledge 是否過時。
- Resource 是否失控。
- Agent 是否反覆產生無效工作。
- 評估指標是否遭到錯誤最佳化。

## 16.4 Human Review

使用者可以選擇：

- 每日查看摘要。
- 每月檢驗成果。
- 重大事件通知。
- 高風險操作核准。
- 長期策略檢討。

人類介入頻率應符合實驗需求，不應把每個低風險決策都變成人工審核。

---

# 17. Go Runtime Implementation Guidance

## 17.1 Existing Runtime First

本文件不要求重新建立 Runtime。

應優先檢查現有 Go 程式碼中的：

- Service 啟動與停止。
- Makefile Service 安裝。
- Runtime Lifecycle。
- Scheduler。
- Event Processing。
- Task Management。
- Worker Management。
- Persistence。
- Logging。
- Recovery。

已有功能應保留並補足缺失。

## 17.2 Suggested Operational Interfaces

```go
type HealthService interface {
    Liveness(ctx context.Context) HealthResult
    Readiness(ctx context.Context) HealthResult
    Dependencies(ctx context.Context) []DependencyHealth
}

type MetricsService interface {
    Record(ctx context.Context, metric Metric) error
    Snapshot(ctx context.Context) (MetricsSnapshot, error)
}

type RecoveryService interface {
    Inspect(ctx context.Context) (RecoveryReport, error)
    Resume(ctx context.Context, taskID string) error
    Pause(ctx context.Context, taskID string) error
}

type ResourceManager interface {
    CheckBudget(ctx context.Context, request ResourceRequest) (BudgetDecision, error)
    RecordUsage(ctx context.Context, usage ResourceUsage) error
}

type BackupService interface {
    Create(ctx context.Context, request BackupRequest) (BackupResult, error)
    Verify(ctx context.Context, backupID string) (VerificationResult, error)
    Restore(ctx context.Context, backupID string) error
}
```

以上為邏輯 Interface 參考，應依現有程式碼整合，不要求照名稱重建。

## 17.3 Graceful Shutdown

Go Runtime 應使用 Context Cancellation 與受控 Shutdown。

Worker 必須能收到取消訊號。

但取消不代表工作成果可以直接丟棄。

在安全可行的情況下，應先保存：

- Current Task State。
- Execution Checkpoint。
- Workspace 狀態。
- 已完成的步驟。
- Pending External Operations。

## 17.4 Background Service

若透過 Makefile 安裝 Linux Service，應確保：

- 正確設定 Working Directory。
- 使用專用使用者與適當權限。
- 設定 Restart Policy。
- 限制檔案與目錄存取。
- 保存標準輸出與錯誤輸出。
- 提供受控停止。
- 不因短暫外部依賴失效而無限快速重啟。

## 17.5 Restart Policy

Restart Policy 應區分：

- Process Crash。
- Configuration Error。
- Storage Failure。
- External Provider Failure。
- Manual Stop。
- Emergency Stop。

Emergency Stop 與明確的人工停止，不應被一般自動重啟機制直接覆蓋。

## 17.6 Operational Independence

Runtime 必須能在沒有外部 AI Provider 的情況下維持基本管理能力。

例如：

- 顯示狀態。
- 保存紀錄。
- 管理 Task。
- 暫停 Worker。
- 處理 Recovery。
- 提供人工控制。

但沒有模型時，不應假裝自主推理仍然正常。

---

# 18. Operational Acceptance Criteria

## 18.1 Deployment

- [ ] 可在 Local Environment 安裝。
- [ ] 可透過 Service 管理。
- [ ] Web Server 預設只監聽 `127.0.0.1`。
- [ ] CLI 與 Web UI 使用共用服務。
- [ ] 不依賴不必要的分散式服務。

## 18.2 Lifecycle

- [ ] Runtime 可正常啟動。
- [ ] Runtime 可正常停止。
- [ ] Worker 可受控取消。
- [ ] Process Crash 後可恢復。
- [ ] 未完成 Task 不會無故消失。
- [ ] 不會因重試造成重複副作用。

## 18.3 Observability

- [ ] Log 為結構化格式。
- [ ] 重要事件具有 Correlation ID。
- [ ] Runtime Metrics 可取得。
- [ ] Execution Metrics 可取得。
- [ ] Autonomy Metrics 可取得。
- [ ] 可追蹤 Decision 到 Execution 的完整鏈路。
- [ ] 敏感資料不會洩漏到 Log。

## 18.4 Autonomous Health

- [ ] 可以偵測 Repeated Action Loop。
- [ ] 可以偵測 Endless Deliberation。
- [ ] 可以偵測 Task Explosion。
- [ ] 可以偵測 Goal Drift。
- [ ] 可以偵測 Resource Spiral。
- [ ] 可以偵測長期無進展。
- [ ] 異常時可以暫停或要求人工介入。

## 18.5 Recovery

- [ ] 可以檢查未完成 Task。
- [ ] 可以恢復 Project 狀態。
- [ ] 可以恢復 Experiment 狀態。
- [ ] 可以驗證 Backup。
- [ ] 可以執行 Restore。
- [ ] 可以檢查資料與 Audit 完整性。

## 18.6 Upgrade

- [ ] 升級前可備份。
- [ ] Schema Migration 有版本紀錄。
- [ ] 升級失敗可以回復。
- [ ] 核心修改有 Diff。
- [ ] Self-Modification 不會直接覆蓋正在運作的核心 Runtime。

---

# 19. Implementation Priority

建議依下列順序補足維運能力：

| Priority | 項目 | 目的 |
|---|---|---|
| P0 | Runtime Lifecycle | 確保系統可持續運作 |
| P0 | Persistent Recovery | 避免長期工作中斷後遺失 |
| P0 | Structured Logging | 能追查系統行為 |
| P0 | Health Check | 知道系統是否正常 |
| P1 | Correlation ID | 串聯完整自主循環 |
| P1 | Resource Management | 防止資源失控 |
| P1 | Backup / Restore | 保護長期資料 |
| P1 | Worker Stuck Detection | 避免無效執行 |
| P2 | Autonomy Anomaly Detection | 偵測自主行為異常 |
| P2 | Advanced Dashboard | 觀察長期自主能力 |
| P2 | Automated Recovery | 在授權範圍內自動恢復 |
| P3 | Multi-Process Deployment | 有實際需求再擴展 |

P0、P1、P2、P3 是建議實作優先順序，不代表目前程式碼已完成這些能力。

---

# 20. Final Principles

AIA 的長期運作必須遵守：

1. **Process Alive ≠ System Healthy。** Process 存活不代表自主循環正常。
2. **Task Completed ≠ Goal Achieved。** 任務完成不代表長期目標已達成。
3. **More Activity ≠ More Autonomy。** 更多工具呼叫與 Task 不代表自主能力更高。
4. **Persistent State ≠ Blind Continuation。** 恢復工作前必須重新確認狀態與條件。
5. **Automatic Restart ≠ Recovery。** 重啟成功不代表資料與工作已正確恢復。
6. **Logs ≠ Audit。** 一般 Log 不可取代不可任意修改的稽核紀錄。
7. **Metrics ≠ Understanding。** 數值必須有明確定義與解讀邊界。
8. **Autonomous Operation ≠ Unlimited Resource Usage。** 自主性必須受到資源預算約束。
9. **Self-Improvement ≠ Uncontrolled Self-Modification。** 自我改善必須有隔離、驗證與回滾。
10. **Human Oversight Remains Available。** 人類必須始終能夠查看、暫停、撤銷與停止。

---

# 21. Final Statement

AIA 的自主能力必須能夠跨越時間持續存在。

這意味著系統不能只在 Agent 正在執行時才有意義，也不能因為某次 Session 結束、Process 重啟、模型失效或外部資料來源中斷，就失去長期工作脈絡。

真正的長期自主系統必須具備：

- 持續運作能力。
- 狀態保存能力。
- 故障恢復能力。
- 行為追蹤能力。
- 自主性異常偵測能力。
- 資源管理能力。
- 安全控制能力。
- 長期成果評估能力。

Runtime 必須知道系統正在做什麼。

Autonomy Core 必須知道自己為什麼做這些事。

Observability 必須讓系統與人類知道這些行動是否有效。

Recovery 必須讓系統在中斷後能夠正確延續。

**AIA 不只是能夠長期執行的 AI 系統，而是必須能夠在長期運作中持續檢查自身狀態、判斷行動價值、修正錯誤，並維持目標連續性的自主系統。**
