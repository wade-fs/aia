# AIA-06：Embedded Web UI & Agent Interface Specification

| 項目 | 內容 |
|---|---|
| 文件編號 | AIA-06 |
| 文件名稱 | Embedded Web UI & Agent Interface Specification |
| 版本 | 2.1 |
| 文件狀態 | Revised Baseline |
| 系統名稱 | AIA — Autonomous Intelligence Architecture |
| Implementation Language | Go |
| Web Framework | 依目前 Go 專案實作 |
| UI Architecture | Embedded Web UI |
| Primary Domain | Investment |
| 依據文件 | AIA-01 v2.0、AIA-02 v2.0、AIA-03 v2.0、AIA-04 v2.0、AIA-05 v2.0 |

---

# 1. 文件目的

本文件定義 AIA Embedded Web UI、Web Host、Core Interface 及 Domain Agent Interface 的設計規格。

AIA Web UI 不只是管理介面，也不只是投資資訊儀表板。

它是使用者觀察、理解及管理 AIA 自主性行為的主要入口。

介面必須能回答：

1. AIA 目前是否正在運作？
2. Agent 最近觀察了什麼？
3. Agent 是否主動發現值得處理的事件？
4. Agent 如何延續過去的工作與判斷？
5. Agent 做了什麼決策？為什麼？
6. Agent 為什麼選擇不行動？
7. Agent 過去的預測是否正確？
8. Agent 是否發現並修正自己的錯誤？
9. Agent 是否調整過策略或提出目標修正？
10. 目前的自主性能力是否真的有進展？

介面設計必須與 AIA 的核心研究目標一致：

- Initiative：主動性思維。
- Continuity：時間連續性。
- Self-Correction：自我修正。
- Goal Evolution：目標演化。

## 1.1 設計原則

- Core 與 Domain Module 分離。
- Web UI 不直接承擔 Agent 推理責任。
- Web UI 不直接操作資料庫。
- 所有狀態變更必須經過 Service Layer。
- 所有受治理的操作必須經過 Policy。
- 所有重要操作必須留下 Audit。
- UI 不得將尚未實作的功能顯示為已完成。
- 不得將 MOCK 資料顯示成真實市場資料。
- 不得將 Web Server 常駐誤認為 Agent Runtime 常駐。

# 2. 系統定位

## 2.1 Web UI 在 AIA 中的角色

```text
                   User
                    │
                    ▼
               Embedded Web UI
                    │
                    ▼
                 Web Host
                    │
          ┌─────────┴─────────┐
          │                   │
          ▼                   ▼
      Core API            Domain API
          │                   │
          ▼                   ▼
      Core Services      Investment Services
          │                   │
          └─────────┬─────────┘
                    │
                    ▼
                 Repositories
                    │
                    ▼
                  SQLite
```

Web UI 是呈現及操作層。

它不應直接呼叫：

- SQLite。
- Repository。
- Agent Harness。
- 外部資料 Provider。
- Runtime Worker。

這些操作應由相應 Service 負責。

## 2.2 Core 與 Domain Module

AIA 採用 Core + Domain Module 架構。

### Core Interface

Core 提供所有 Domain 共用的管理與觀察能力：

- System Overview。
- Runtime Status。
- Agent Registry。
- Task Management。
- Schedule Management。
- Memory Inspection。
- Goal Management。
- Autonomy Monitor。
- Policy。
- Approval。
- Audit。
- Event History。

### Domain Interface

Domain Module 提供領域專屬的功能。

目前只有 Investment Module。

未來可能包含：

- Investment。
- Coding。
- Writing。

不同 Domain 可以有不同的資料模型、操作流程及呈現方式，但應使用同一套 Core API 與 Autonomy Core。

# 3. Current Implementation Status

目前程式碼已建立：

- Web Host。
- Console Routes。
- Agent Web Module Registration。
- Investment Web Module。
- Coding Web Module Registration。
- Writing Web Module Registration。
- 部分 Investment API。
- 部分 Web UI 與登入流程。

但必須區分「模組已註冊」與「模組功能已完成」。

| Component | Status | 說明 |
|---|---|---|
| Web Host | PARTIAL | 已有 HTTP Host |
| Embedded Web Assets | VERIFY | 依目前實際嵌入方式確認 |
| Login | PARTIAL | 已有一次性登入 Token 流程 |
| Core Dashboard | PARTIAL | 尚非完整自主性儀表板 |
| Agent Registry UI | PARTIAL | 有模組註冊基礎 |
| Task Interface | PARTIAL | 尚未形成完整 Runtime 操作介面 |
| Schedule Interface | PARTIAL | 有部分操作能力 |
| Memory Interface | NOT READY | 尚未完成長期記憶檢視 |
| Goal Interface | NOT READY | 尚未完成 Goal Evolution 操作 |
| Autonomy Monitor | NOT READY | 尚未完成四項自主能力監控 |
| Investment Module | PARTIAL | 有基本投資操作介面 |
| Coding Module | NOT IMPLEMENTED | 只有預留或註冊基礎，不代表功能完成 |
| Writing Module | NOT IMPLEMENTED | 只有預留或註冊基礎，不代表功能完成 |
| Background Runtime Control | NOT IMPLEMENTED | 尚無完整 Runtime Worker 管理 |
| WebSocket / Live Events | NOT READY / VERIFY | 尚須確認實際實作 |

以上狀態以目前程式碼檢視為基礎。PARTIAL 代表已有局部實作，不代表符合本文件完整驗收標準。

# 4. Web Host Architecture

## 4.1 Web Host Responsibilities

Web Host 負責：

- HTTP Server。
- Route Registration。
- Static Asset Delivery。
- Authentication。
- Session Management。
- CSRF Protection。
- API Request Handling。
- Domain Module Registration。
- Error Handling。
- Graceful Shutdown。

Web Host 不負責：

- 長期自主推理。
- 市場資料收集。
- Task 背景執行。
- Schedule Polling。
- Prediction Evaluation。
- Initiative Discovery。
- Self-Correction。
- Goal Evolution。

這些工作屬於 Runtime 及 Service Layer。

## 4.2 Web Server 與 Runtime 分離

必須明確區分：

```text
aia server start
    │
    └── HTTP Server

aia runtime start
    │
    ├── Runtime Supervisor
    ├── Task Worker
    ├── Schedule Worker
    ├── Observation Worker
    ├── Initiative Worker
    └── Evaluation Worker
```

`aia server start` 目前只是啟動 Web Server。

`aia runtime start` 是目標中的 Runtime 啟動命令，尚未完成實作。

Web Server 可以在 Runtime 不存在時提供歷史資料及系統狀態檢視，但不得宣稱 Agent 正在持續自主運作。

## 4.3 Server Lifecycle

Web Server 必須支援：

- 正常啟動。
- 重複啟動防護。
- HTTP Shutdown。
- Session 清理。
- 資源釋放。
- 錯誤日誌。

目前 `runWebServer()` 使用 HTTP Server 並等待 SIGINT / SIGTERM，收到訊號後執行 Shutdown。

這個流程不代表 Background Worker 已啟動。

## 4.4 Runtime Status Presentation

Web UI 必須分別呈現：

| Status | Meaning |
|---|---|
| WEB_ONLINE | Web Host 正常 |
| RUNTIME_RUNNING | Runtime 正常運作 |
| RUNTIME_STOPPED | Runtime 未運作 |
| RUNTIME_DEGRADED | Runtime 部分功能異常 |
| RUNTIME_RECOVERING | Runtime 正在恢復 |
| UNKNOWN | 無法確認狀態 |

如果只啟動 Web Server，UI 不得顯示 RUNTIME_RUNNING。

# 5. Navigation Architecture

Web Console 建議採用以下主導覽：

```text
AIA
│
├── Overview
│
├── Autonomy
│   ├── Initiative
│   ├── Continuity
│   ├── Self-Correction
│   └── Goal Evolution
│
├── Agents
│   ├── Agent Registry
│   └── Agent Details
│
├── Tasks
│
├── Schedules
│
├── Memory
│
├── Investment
│   ├── Portfolio
│   ├── Market Observation
│   ├── Research
│   ├── Decisions
│   ├── Orders & Transactions
│   ├── Predictions
│   ├── Strategies
│   └── Monthly Review
│
├── Policy & Approval
│
├── Audit
│
└── Settings
```

目前尚未實作的頁面可以先顯示功能狀態，但不得提供沒有作用的操作按鈕。

# 6. Core Dashboard

## 6.1 Overview

首頁應讓使用者快速了解 AIA 的整體狀態。

必要資訊：

- Web Host Status。
- Runtime Status。
- Agent Count。
- Active Tasks。
- Pending Tasks。
- Failed Tasks。
- Pending Approvals。
- Recent Events。
- Recent Decisions。
- Last Observation。
- Last Evaluation。

## 6.2 System Health

系統健康狀態至少包含：

- Database。
- Repository。
- EventBus。
- Runtime Supervisor。
- Worker Status。
- Harness Connection。
- Collector Status。
- Memory Service。
- Agent Registry。

尚未實作的元件應顯示 NOT IMPLEMENTED，而不是顯示 HEALTHY。

## 6.3 Current Work

呈現目前正在處理的工作：

- Task。
- Agent。
- Project。
- Started At。
- Last Activity。
- Current Stage。
- Execution Status。

必須區分：

- Task 已建立。
- Task 已排入佇列。
- Task 正在執行。
- Task 已完成。
- Task 失敗。
- Task 等待人工核准。
- Task 等待登入或外部授權。

# 7. Autonomy Monitor

這是 AIA Web UI 的核心功能之一。

Autonomy Monitor 不是一般 Agent 狀態面板，而是呈現四項自主性能力的可觀察證據。

## 7.1 Initiative Monitor

顯示：

- Agent 主動發現的事件。
- 主動建立的研究工作。
- 主動提出的決策。
- 主動要求的資料。
- No-Action Decision。
- 主動性事件的驗證結果。

每個 Initiative Event 至少顯示：

- Event ID。
- Detection Time。
- Trigger。
- Context。
- Agent Assessment。
- Proposed Action。
- Result。
- Verification Status。

必須區分：

- Schedule Triggered。
- User Triggered。
- Event Triggered。
- Agent Initiated。

定時排程觸發不應直接算成 Agent 主動性成果。

## 7.2 Continuity Monitor

顯示：

- Current Goals。
- Active Projects。
- Historical Context References。
- Recent Decisions。
- Pending Tasks。
- Unresolved Issues。
- Last Recovery Status。

應能查看 Agent 是否真的使用過去的記憶，而不是只顯示記憶筆數。

重要指標：

- Historical Context Retrieval。
- Decision Continuity。
- Task Recovery。
- Goal Persistence。
- Restart Recovery Success。

## 7.3 Self-Correction Monitor

顯示：

- Detected Errors。
- Error Classification。
- Correction Proposals。
- Strategy Versions。
- Validation Status。
- Before / After Comparison。

每次修正至少應能追溯：

```text
Original Decision
       │
       ▼
Observed Result
       │
       ▼
Error Analysis
       │
       ▼
Correction Proposal
       │
       ▼
New Strategy Version
       │
       ▼
Validation
       │
       ▼
Result
```

不得只顯示「Agent 已學習」或「Agent 已改善」等無法驗證的文字。

## 7.4 Goal Evolution Monitor

顯示：

- Active Goals。
- Goal History。
- Goal Optimization Proposals。
- Goal Revision Proposals。
- Approval Status。
- Goal Progress。

必須區分：

- Goal Persistence。
- Goal Optimization。
- Goal Revision。

重大 Goal Revision 必須顯示核准狀態。

## 7.5 Autonomy Evidence

所有自主性評估應連結實際事件。

例如：

| Capability | Evidence |
|---|---|
| Initiative | Agent 主動發現市場風險並建立研究 Task |
| Continuity | Agent 使用過去的投資論點重新評估持倉 |
| Self-Correction | Agent 發現預測偏差並建立新策略版本 |
| Goal Evolution | Agent 提出方法優化並完成驗證 |

不能只靠模型自己輸出的自我評分作為能力證據。

# 8. Agent Interface

## 8.1 Agent Registry

顯示：

- Agent ID。
- Name。
- Domain。
- Version。
- Status。
- Capabilities。
- Registered At。
- Last Activity。

## 8.2 Agent Details

每個 Agent 應有獨立詳細頁。

內容包含：

- Agent Description。
- Domain。
- Capabilities。
- Current State。
- Current Goals。
- Active Tasks。
- Recent Decisions。
- Memory References。
- Policy Scope。
- Recent Audit Events。

## 8.3 Agent Lifecycle

顯示：

- REGISTERED。
- INITIALIZED。
- READY。
- RUNNING。
- PAUSED。
- DEGRADED。
- STOPPED。
- FAILED。

實際狀態名稱應與 Domain Model 一致。

不得在 UI 自行創造另一套與 Backend 不相容的狀態定義。

## 8.4 Domain Module Isolation

每個 Domain Module 應具備獨立 Route Group。

例如：

```text
/core/...
/agents/...
/tasks/...
/schedules/...
/autonomy/...
/investment/...
```

Investment Module 不應直接操作 Core Repository。

Core 也不應直接依賴 Investment 專屬資料表。

# 9. Investment Workbench

Investment 是目前唯一正式 Domain Module。

其介面應優先完整呈現 AIA-05 所定義的投資自主性流程。

## 9.1 Portfolio Overview

顯示：

- Initial Capital。
- Total Assets。
- Cash。
- Holdings Market Value。
- Realized P&L。
- Unrealized P&L。
- Dividends。
- Transaction Costs。
- Total Return。
- Monthly Return。
- Maximum Drawdown。

必須標示資產估值日期及資料品質。

## 9.2 Holdings

每筆持倉顯示：

- Symbol。
- Name。
- Quantity。
- Average Cost。
- Current Price。
- Market Value。
- Unrealized P&L。
- Weight。
- Last Updated。

如果行情資料過期，必須標示 STALE。

如果行情來源是 MOCK，必須明確顯示 MOCK，不得混入正式 Portfolio。

## 9.3 Market Observation

主要觀察項目：

- 台積電 2330。
- 台灣加權指數。
- 主要持倉標的。
- 重要市場指標。
- 國際市場資訊。
- 公司公告。
- 重要新聞。

台積電應有獨立觀察區域，但不代表 Agent 必須持有台積電。

## 9.4 Data Quality View

顯示每個 Provider：

- Provider Name。
- Source Type。
- Connection Status。
- Last Successful Collection。
- Last Failed Collection。
- Data Freshness。
- Data Origin。
- Validation Status。
- Error Message。

資料來源狀態應區分：

- REAL。
- MOCK。
- FALLBACK。
- DERIVED。
- STALE。
- INVALID。

如果 Provider 使用固定資料或假公告，應明確顯示 NOT READY 或 MOCK。

## 9.5 Research

每個研究項目顯示：

- Research ID。
- Symbol。
- Research Time。
- Snapshot。
- Investment Thesis。
- Key Assumptions。
- Supporting Evidence。
- Risk Factors。
- Invalidation Conditions。
- Conclusion。

應能查看當時使用的資料來源及 Snapshot。

## 9.6 Decisions

每筆重要決策顯示：

- Decision ID。
- Decision Time。
- Symbol。
- Action。
- Quantity。
- Expected Outcome。
- Confidence。
- Risk Assessment。
- Reasoning Summary。
- Alternatives Considered。
- Related Snapshot。
- Related Prediction。
- Execution Status。

必須支援歷史比較。

使用者應能回答：

「Agent 當時為什麼買進？當時知道什麼？現在又有哪些事情改變了？」

## 9.7 No-Action Decisions

不交易決策應與一般決策同等重要。

顯示：

- Decision Time。
- Current Portfolio State。
- Reason for No Action。
- Conditions Not Met。
- Reassessment Conditions。

不得將沒有交易紀錄誤認為 Agent 沒有進行決策。

## 9.8 Orders & Transactions

Orders 頁面顯示：

- Order ID。
- Symbol。
- Side。
- Order Type。
- Quantity。
- Limit Price。
- Status。
- Filled Quantity。
- Average Fill Price。
- Related Decision。

Transactions 頁面顯示：

- Transaction ID。
- Symbol。
- Side。
- Quantity。
- Execution Price。
- Commission。
- Transaction Tax。
- Net Amount。
- Execution Time。

Orders 與 Transactions 必須分開。

委託不等於成交。

## 9.9 Predictions

顯示：

- Prediction ID。
- Symbol。
- Prediction Type。
- Predicted Result。
- Confidence。
- Probability。
- Time Horizon。
- Evaluation Status。
- Actual Result。
- Prediction Error。

必須支援依標的、預測類型及時間範圍查詢。

## 9.10 Strategies

顯示：

- Strategy ID。
- Version。
- Previous Version。
- Change Description。
- Reason。
- Evidence。
- Validation Period。
- Validation Result。
- Current Status。

使用者應能比較不同版本的策略及結果。

## 9.11 Monthly Review

每月 1 日的檢查報告應包含：

1. 總資產。
2. 現金。
3. 持倉。
4. 月報酬。
5. 累積報酬。
6. 已實現損益。
7. 未實現損益。
8. 交易成本。
9. 與 Benchmark 比較。
10. 重要決策。
11. 預測評估。
12. 策略修正。
13. 自主性事件摘要。

報告必須保留歷史版本。

# 10. Task & Schedule Interface

## 10.1 Task List

顯示：

- Task ID。
- Task Name。
- Agent。
- Project。
- Priority。
- Status。
- Created At。
- Started At。
- Completed At。
- Retry Count。

支援篩選：

- Pending。
- Running。
- Completed。
- Failed。
- Waiting for Approval。
- Waiting for Auth。

## 10.2 Task Details

詳細頁顯示：

- Task Description。
- Task Context。
- Parent Task。
- Dependencies。
- Execution History。
- Tool Events。
- Result。
- Verification。
- Error。
- Related Memory。
- Audit References。

## 10.3 Schedule Management

顯示：

- Schedule ID。
- Name。
- Agent。
- Cron / Interval。
- Next Run。
- Last Run。
- Status。
- Misfire Policy。

支援：

- Create。
- Update。
- Enable。
- Disable。
- Trigger。
- View History。

## 10.4 Schedule vs Task

Schedule 與 Task 必須分開呈現。

Schedule 負責產生 Task。

Task 負責實際執行。

目前 `aia schedule tick` 只會處理到期 Schedule 並建立 Task，不會自動呼叫 `RunTask()`。

在 Task Background Worker 完成之前，UI 必須清楚呈現這項限制。

## 10.5 Misfire Status

顯示：

- SKIP。
- RUN_ONCE。
- CATCH_UP。

但不得將尚未完成的 Misfire 行為顯示為完整支援。

目前程式碼只有 SKIP 有明確特殊處理，其餘行為仍須進一步驗證。

# 11. Memory Interface

## 11.1 Memory Overview

顯示：

- Memory Count。
- Recent Memories。
- Memory Types。
- Related Agents。
- Last Updated。
- Retrieval History。

## 11.2 Memory Details

每筆 Memory 顯示：

- Memory ID。
- Type。
- Content Summary。
- Source。
- Created At。
- Updated At。
- Related Task。
- Related Decision。
- Confidence。
- Status。

## 11.3 Memory Provenance

Memory 必須能追溯來源。

例如：

- User Input。
- Agent Observation。
- Research Result。
- Decision。
- Task Result。
- External Document。
- Evaluation。

不得將 Agent 推測內容與已確認事實混為一談。

## 11.4 Memory Retrieval

應能查看某次決策使用了哪些 Memory。

這對 Continuity 驗收尤其重要。

只顯示記憶存在，不能證明 Agent 真正使用過記憶。

# 12. Goal Interface

## 12.1 Goal List

顯示：

- Goal ID。
- Goal Name。
- Description。
- Owner。
- Priority。
- Status。
- Created At。
- Last Updated。

## 12.2 Goal Details

顯示：

- Goal Definition。
- Parent Goal。
- Subgoals。
- Constraints。
- Progress。
- Related Tasks。
- Related Decisions。
- Goal History。

## 12.3 Goal Evolution

顯示：

- Goal Optimization Proposal。
- Goal Revision Proposal。
- Reason。
- Evidence。
- Expected Impact。
- Approval Status。
- Final Decision。

重大目標修改必須經過使用者核准。

UI 不得讓 Agent 直接修改受治理的 Goal 而不留下 Approval 及 Audit。

# 13. Policy, Approval & Audit

## 13.1 Policy Interface

顯示：

- Policy ID。
- Name。
- Scope。
- Rules。
- Status。
- Last Updated。

Policy 由 Core 管理。

Domain Module 只能讀取及遵守適用規則。

## 13.2 Approval Interface

顯示待核准事項：

- Approval ID。
- Request Type。
- Requesting Agent。
- Reason。
- Requested Action。
- Risk。
- Created At。
- Expiration。

使用者可以依權限：

- Approve。
- Reject。

核准及拒絕都必須留下紀錄。

## 13.3 Audit Interface

Audit 頁面應能查詢：

- User Actions。
- Agent Actions。
- Task Events。
- Tool Events。
- Policy Decisions。
- Approval Events。
- Investment Decisions。
- Trading Events。
- Strategy Changes。
- Goal Changes。

Audit 必須保留歷史完整性。

不得提供一般 UI 操作直接修改或刪除歷史 Audit Record。

# 14. Agent Interaction Interface

## 14.1 Interaction Model

使用者應能透過 Web UI 與 Personal Agent 互動。

支援：

- 一般對話。
- 查詢 Agent 狀態。
- 查詢歷史決策。
- 建立 Task。
- 指派 Agent。
- 查詢投資組合。
- 要求解釋決策。
- 要求檢查策略。
- 提出目標調整要求。

## 14.2 User Input vs Agent Initiative

介面必須區分：

- 使用者要求。
- 系統事件。
- Schedule 觸發。
- Agent 主動提出。

不能把所有 Task 都顯示成 Agent 自主建立。

## 14.3 Agent Response

回應應能連結相關資料：

- Task。
- Decision。
- Memory。
- Research Snapshot。
- Prediction。
- Audit Event。

若 Agent 提到歷史決策，使用者應能查看對應紀錄。

# 15. API Design

## 15.1 General Principles

API 應遵守：

- RESTful Resource Design。
- JSON。
- 明確的 HTTP Status Code。
- 統一 Error Response。
- Pagination。
- Filtering。
- Sorting。
- Authentication。
- Authorization。
- CSRF Protection。

API Version 建議使用：

```text
/api/v1/
```

## 15.2 Core API Groups

建議介面：

```text
/api/v1/system
/api/v1/runtime
/api/v1/agents
/api/v1/tasks
/api/v1/schedules
/api/v1/memory
/api/v1/goals
/api/v1/autonomy
/api/v1/policy
/api/v1/approvals
/api/v1/audit
```

## 15.3 Investment API Groups

```text
/api/v1/investment/portfolio
/api/v1/investment/holdings
/api/v1/investment/market
/api/v1/investment/sources
/api/v1/investment/snapshots
/api/v1/investment/research
/api/v1/investment/decisions
/api/v1/investment/orders
/api/v1/investment/transactions
/api/v1/investment/predictions
/api/v1/investment/strategies
/api/v1/investment/reviews
```

## 15.4 API Contract

每個 API 必須定義：

- Request Schema。
- Response Schema。
- Validation。
- Error Codes。
- Permission Requirement。
- Audit Requirement。
- Idempotency Requirement。

不得由前端自行推測 Backend 資料格式。

## 15.5 Error Response

建議統一格式：

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Requested resource was not found",
    "request_id": "request-id"
  }
}
```

實際欄位及命名應與既有 API 慣例統一。

# 16. Event & Live Update

## 16.1 Event-driven UI

未來 UI 可透過 Server-Sent Events（SSE）或 WebSocket 接收即時事件。

適用事件：

- Runtime Status Changed。
- Task Created。
- Task Started。
- Task Completed。
- Agent State Changed。
- Initiative Detected。
- Decision Created。
- Order Filled。
- Prediction Evaluated。
- Strategy Updated。
- Approval Required。

## 16.2 Polling Fallback

在 Live Event 尚未完成時，可使用定時 API Polling。

但應注意：

- 不得造成過度請求。
- 應支援頁面停止輪詢。
- 應處理連線失敗。
- 應避免重複事件。
- 應能重新同步最新狀態。

## 16.3 Event Integrity

前端收到事件後，不應直接將事件視為最終帳務結果。

重要狀態仍應透過正式 API 重新確認。

例如 Order Filled Event 到達後，應重新取得 Order 與 Portfolio 狀態。

# 17. Security

## 17.1 Network Binding

預設只監聽：

```text
127.0.0.1
```

不預設開放 LAN 或 Internet。

## 17.2 Authentication

沿用目前一次性登入 Token 流程及 Session Management。

要求：

- Token 不可長期重複使用。
- Session 必須具備到期機制。
- 登入失敗應受到限制。
- Cookie 必須設定適當安全屬性。
- 登出後應失效。

## 17.3 CSRF

所有會改變系統狀態的請求必須進行 CSRF 防護。

## 17.4 Permission

前端隱藏按鈕不等於權限控制。

Backend 必須重新驗證所有操作權限。

## 17.5 XSS & Input Validation

必須：

- 避免不安全 HTML 注入。
- 對使用者輸入進行驗證。
- 對外部資料進行安全呈現。
- 防止惡意內容透過新聞、公告或文件進入 UI。

## 17.6 Sensitive Information

不得在 UI 或 API 回傳：

- OAuth Token。
- Harness Credential。
- API Secret。
- Database Credential。
- Session Secret。
- 其他敏感憑證。

使用者自行透過 Antigravity CLI 完成 OAuth 登入。

AIA 不讀取、不複製、不解析 OAuth Token。

# 18. Performance & Reliability

## 18.1 Performance

UI 應避免：

- 一次載入全部歷史資料。
- 大量同步 API 呼叫。
- 不必要的全頁刷新。
- 重複查詢大型 Audit History。

應支援：

- Pagination。
- Lazy Loading。
- Server-side Filtering。
- 合理的 Cache。
- Request Timeout。

## 18.2 Reliability

必須處理：

- Web Host 中斷。
- Runtime 未啟動。
- Database Error。
- Provider Error。
- API Timeout。
- Session Expiration。
- Event Connection Lost。

錯誤狀態應清楚呈現，不得顯示過期資料卻不標示。

## 18.3 Graceful Degradation

某個 Domain Module 失敗，不應使整個 Core Console 無法使用。

Investment 資料來源異常時，仍應允許使用者檢視歷史 Portfolio、決策及 Audit。

但必須標示資料時間與品質狀態。

# 19. Embedded UI Implementation

## 19.1 Asset Management

UI 靜態資源應依目前 Go 專案採用的嵌入方式管理。

建議將：

- HTML。
- CSS。
- JavaScript。
- Icons。
- Static Images。

與 Backend Service 分開管理。

## 19.2 Frontend Boundary

Frontend 不應：

- 直接連接 SQLite。
- 自行計算正式帳務結果。
- 自行產生交易成交紀錄。
- 自行修改策略版本。
- 自行修改 Goal。
- 繞過 Backend Policy。

所有重要狀態應由 Backend 提供。

## 19.3 UI Technology

技術選型應遵循：

- 與現有 Go Runtime 整合。
- 部署簡單。
- 可嵌入執行檔。
- 不依賴外部 CDN 才能正常啟動。
- 易於長期維護。
- 不將前端框架與 Domain Model 綁死。

具體前端技術以 Repository 實際採用方式為準，不在本規格中強制引入新的框架。

# 20. Acceptance Criteria

## 20.1 Web Host

- [ ] Web Host 可正常啟動及停止。
- [ ] 預設只監聽 127.0.0.1。
- [ ] Session 正常運作。
- [ ] CSRF 防護有效。
- [ ] Graceful Shutdown 正常。
- [ ] Web Server 與 Runtime 狀態分離。

## 20.2 Core Interface

- [ ] Overview。
- [ ] System Health。
- [ ] Agent Registry。
- [ ] Task Management。
- [ ] Schedule Management。
- [ ] Memory Inspection。
- [ ] Goal Management。
- [ ] Policy。
- [ ] Approval。
- [ ] Audit。

## 20.3 Autonomy Monitor

- [ ] Initiative Events。
- [ ] Continuity Evidence。
- [ ] Self-Correction History。
- [ ] Goal Evolution History。
- [ ] No-Action Decisions。
- [ ] Evidence-based Metrics。

## 20.4 Investment Workbench

- [ ] Portfolio Overview。
- [ ] Holdings。
- [ ] Market Observation。
- [ ] Data Quality。
- [ ] Research。
- [ ] Decision History。
- [ ] No-Action Decisions。
- [ ] Orders。
- [ ] Transactions。
- [ ] Predictions。
- [ ] Strategies。
- [ ] Monthly Review。

## 20.5 Agent Interface

- [ ] Agent Details。
- [ ] Agent Lifecycle。
- [ ] Agent Capabilities。
- [ ] Agent Tasks。
- [ ] Agent Decisions。
- [ ] Agent Memory References。
- [ ] User Interaction。

## 20.6 Data Integrity

- [ ] REAL / MOCK 隔離。
- [ ] FALLBACK 明確標記。
- [ ] STALE 資料警示。
- [ ] Snapshot 可追溯。
- [ ] 歷史決策可追溯。
- [ ] API 不回傳敏感憑證。

## 20.7 Runtime Integration

- [ ] Runtime Status。
- [ ] Worker Status。
- [ ] Task Background Execution。
- [ ] Schedule Polling。
- [ ] Daily Investment Observation。
- [ ] Monthly Investment Review。
- [ ] Restart Recovery。

# 21. Implementation Roadmap

## R0：Web Interface Baseline

目標：整理現有 Web Host 及 UI 基礎。

- 確認 Route Registration。
- 統一 API Response。
- 確認 Session。
- 確認 CSRF。
- 確認 Embedded Asset。
- 清楚呈現 Web / Runtime 狀態。

## R1：Core Console

目標：完成 Core 管理介面。

- Overview。
- System Health。
- Agent Registry。
- Task。
- Schedule。
- Approval。
- Audit。

## R2：Investment Workbench

目標：完成第一個 Domain Workbench。

- Portfolio。
- Holdings。
- Market Observation。
- Data Quality。
- Research。
- Decisions。
- Orders。
- Transactions。
- Predictions。
- Strategies。
- Monthly Review。

## R3：Continuity & Memory

目標：讓使用者可以檢視 Agent 的歷史脈絡。

- Memory Browser。
- Decision Timeline。
- Investment Thesis History。
- Prediction History。
- Strategy History。
- Restart Recovery Status。

## R4：Autonomy Monitor

目標：呈現四項自主能力的實際證據。

- Initiative Monitor。
- Continuity Monitor。
- Self-Correction Monitor。
- Goal Evolution Monitor。
- Autonomy Metrics。

## R5：Runtime Integration

目標：讓 UI 能管理及觀察完整 Runtime。

- Runtime Control。
- Worker Status。
- Runtime Events。
- Task Execution。
- Schedule Execution。
- Recovery Status。
- Live Updates。

## R6：Future Domain Modules

目標：支援其他 Domain。

- Coding Module。
- Writing Module。
- Shared Domain Interface。
- Cross-domain Project View。
- Cross-domain Memory References。

# 22. Current Implementation Limitations

依目前程式碼檢視，以下限制必須在實作及 UI 中明確反映：

1. `aia server start` 啟動 Web Server 與後台常駐排程與工作槽位（Schedule & Task Workers）。
2. Background Runtime Worker 常駐運行。
3. Schedule Polling 已形成常駐輪詢工作流程。
4. Schedule Tick 自動分派 Task 並由背景槽位自動執行。
5. Investment 模組支援自動與手動同步。
6. Prediction Evaluation 已形成週期評估機制。
7. Collector Providers（TWSE / TPEx / FRED）目前在未接通正式金鑰環境下強制標示為模擬/隔離數據源。
8. 16-Tab Web Console 支援 Autonomy Cycles, Learnings, Experiments, Clarifications, Problems 視覺化。
9. 完整 Context Snapshot 與 Memory Continuity 機制已就緒。
10. Self-Correction Engine 已實作（三層架構）。
11. Goal Evolution Engine 已實作（三向演化與防目標漂移）。
12. Coding / Writing MVA 已實作（支援自修沙盒與知識一致性檢驗）。

UI 必須根據實際狀態呈現，不得以靜態畫面或預設數值假裝功能已經完成。

# 23. Final Design Principles

AIA Web UI 的核心任務是讓自主性行為可觀察、可理解、可追溯、可驗證。

設計必須遵守：

1. Core 與 Domain 分離。
2. Web Host 與 Runtime 分離。
3. UI 與 Business Logic 分離。
4. Agent 行為與使用者操作分離。
5. Schedule Trigger 與 Agent Initiative 分離。
6. Task 建立與 Task 執行分離。
7. Decision 與 Execution 分離。
8. Order 與 Transaction 分離。
9. REAL 與 MOCK 分離。
10. Memory 存在與 Memory 使用分離。
11. 策略修改與策略改善分離。
12. Goal Optimization 與 Goal Revision 分離。
13. 自我評價與可驗證證據分離。

最終的 Web Console 不應只讓使用者知道：

「AIA 今天做了什麼？」

更應讓使用者能夠理解：

- 它為什麼注意到這件事？
- 它過去是否遇過類似情況？
- 它為什麼選擇這個行動？
- 它是否考慮過其他方案？
- 它的預測後來是否正確？
- 它是否因為錯誤而改變方法？
- 它是否仍然朝著原定目標前進？

**Web UI 是 AIA 自主性研究的觀察窗口，不是自主性本身。**

真正的自主能力仍然必須由 Runtime、Autonomy Core 及 Domain Agent 的實際行為證明。
