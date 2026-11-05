# AIA-07：User Manual & Operational Usage Specification

| 項目 | 內容 |
|---|---|
| 文件編號 | AIA-07 |
| 文件名稱 | User Manual & Operational Usage Specification |
| 版本 | 2.1 |
| 文件狀態 | Revised Baseline |
| 系統名稱 | AIA — Autonomous Intelligence Architecture |
| Implementation Language | Go |
| Runtime Harness | Antigravity CLI |
| Database | SQLite |
| Primary Domain | Investment |
| 依據文件 | AIA-01 v2.0 至 AIA-06 v2.1 |

---

# 1. 文件目的

本文件說明 AIA 的安裝、啟動、基本操作、投資實驗、資料管理、安全治理及日常使用方式。

本文件同時區分：

- 已經實作的功能。
- 已有部分實作的功能。
- 尚未實作的功能。
- 已完成規格設計、但尚未完成程式開發的功能。

AIA 是一個個人 AI Agent 自主性研究平台。

它不只是一般聊天介面，也不只是多 Agent 任務管理工具。

AIA 的長期目標是研究 Agent 是否能具備：

1. Initiative：主動性思維。
2. Continuity：時間連續性。
3. Self-Correction：自我修正。
4. Goal Evolution：目標演化。

目前以 Investment Agent 作為第一個完整實驗場域。

# 2. AIA 系統概念

## 2.1 系統組成

```text
AIA
│
├── Go Runtime
│
├── Autonomy Core
│   ├── Initiative
│   ├── Continuity
│   ├── Self-Correction
│   └── Goal Evolution
│
├── Agent System
│   └── Investment Agent
│
├── Persistence
│   ├── SQLite
│   ├── Repository
│   ├── Memory
│   └── Audit
│
├── Harness Integration
│   └── Antigravity CLI
│
├── Web Console
│
└── CLI
```

## 2.2 系統分工

| Component | Responsibility |
|---|---|
| AIA Runtime | 長期運作及工作管理 |
| Autonomy Core | 自主性能力 |
| Domain Agent | 領域任務與專業行為 |
| Antigravity CLI | 底層 Agent Harness |
| SQLite | 持久化儲存 |
| Web Console | 使用者操作與觀察 |
| CLI | 系統管理與命令操作 |

AIA 管理全域 Task、Project、Memory、Policy、Audit 及 Agent 生命週期。

Antigravity CLI 負責底層 AI Agent Harness、工具及權限管理。

使用者自行透過 Antigravity CLI 完成 Google OAuth 登入。

AIA 不讀取、不複製、不解析 OAuth Token。

# 3. 目前功能狀態

## 3.1 功能總覽

| 功能 | 狀態 |
|---|---|
| Go Application Bootstrap | IMPLEMENTED |
| SQLite Persistence | PARTIAL |
| Database Migration | IMPLEMENTED |
| Repository Layer | PARTIAL |
| EventBus | IMPLEMENTED |
| Agent Registry | PARTIAL |
| Harness Integration | PARTIAL |
| Task Management | PARTIAL |
| Schedule Management | PARTIAL |
| Recovery | PARTIAL |
| Memory | PARTIAL |
| Web Host | PARTIAL |
| Investment Module | PARTIAL |
| Data Collector Framework | IMPLEMENTED |
| Real Market Data Integration | NOT READY |
| Virtual Portfolio | PARTIAL |
| Prediction Evaluation | PARTIAL |
| Background Runtime Workers | NOT IMPLEMENTED |
| Initiative Engine | NOT IMPLEMENTED |
| Self-Correction Engine | NOT IMPLEMENTED |
| Goal Evolution Engine | NOT IMPLEMENTED |
| Coding Agent | NOT IMPLEMENTED |
| Writing Agent | NOT IMPLEMENTED |

IMPLEMENTED 表示相關基礎功能已有可確認的實作。

PARTIAL 表示已有部分功能，但尚未達到完整規格。

NOT IMPLEMENTED 表示尚未完成實作。

NOT READY 表示已有部分架構，但目前不適合當作正式功能使用。

## 3.2 重要限制

目前 AIA 尚未形成完整的持續自主 Runtime。

因此：

- Web Server 常駐不代表 Agent 持續運作。
- Schedule 存在不代表排程會自動執行。
- Task 建立不代表 Task 已經執行。
- Collector 存在不代表資料一定是真實資料。
- Prediction Evaluation 存在不代表已經自動週期評估。
- Memory 存在不代表已經具備完整時間連續性。
- Agent 能產生建議不代表具備完整自主決策能力。

# 4. 安裝與環境

## 4.1 系統需求

建議環境：

- Linux。
- Go 開發環境。
- SQLite。
- Antigravity CLI。
- 可連接網際網路的環境。
- 足夠的本機儲存空間。

實際 Go 版本及其他依賴，以 Repository 的 `go.mod`、Makefile 及安裝文件為準。

## 4.2 專案目錄

目前專案主要以 Go 程式碼為核心。

實際目錄應以 Repository 為準。

概念上：

```text
aia/
│
├── go/
│   ├── cmd/
│   ├── internal/
│   ├── configs/
│   ├── migrations/
│   ├── Makefile
│   └── go.mod
│
├── docs/
│
└── README.md
```

以上是概念性示意，不代表所有目錄名稱及層級均已確認。

# 5. Antigravity CLI 設定

## 5.1 登入

使用者自行透過 Antigravity CLI 完成 Google OAuth 登入。

AIA 不管理 OAuth 登入流程。

## 5.2 權限原則

不得使用：

```bash
--dangerously-skip-permissions
```

應由 Antigravity CLI 自身的權限機制管理工具執行。

## 5.3 Authentication Failure

如果 Harness 認證失效：

1. AIA 應停止需要認證的工作。
2. Task 狀態應進入 `WAITING_FOR_AUTH`。
3. 使用者重新透過 Antigravity CLI 登入。
4. 使用者或 Runtime 恢復 Task。
5. AIA 應確認工作是否成功恢復。

不可只依據 Process Exit Code 判定任務成功。

尤其 Headless Mode 可能遇到需要人工核准的工具操作。

即使 Exit Code 為 0，也必須檢查：

- Tool Events。
- stderr。
- Execution Result。
- Workspace Diff。
- Verification Result。

# 6. AIA CLI

## 6.1 基本命令

```bash
aia -h
```

查看所有命令。

```bash
aia version
```

查看版本。

實際支援的命令及參數以目前執行檔的 Help Output 為準。

## 6.2 Investment Commands

目前已存在的主要命令：

```bash
aia investment portfolio
```

查看投資組合。

```bash
aia investment sync <symbol>
```

手動同步指定標的的市場資料及公告。

```bash
aia investment proposals
```

查看投資提案。

```bash
aia investment approve <id>
```

核准投資提案。

```bash
aia investment reject <id>
```

拒絕投資提案。

```bash
aia investment predictions
```

查看預測紀錄。

```bash
aia investment evaluate
```

手動執行預測評估。

以上命令是目前已存在的主要介面，不代表所有投資功能均已自動化。

## 6.3 Schedule Commands

目前已存在的主要命令：

```bash
aia schedule list
```

查看排程。

```bash
aia schedule create
```

建立排程。

```bash
aia schedule trigger
```

手動觸發排程。

```bash
aia schedule tick
```

處理到期排程。

實際參數以：

```bash
aia schedule -h
```

及各子命令 Help Output 為準。

## 6.4 Schedule Tick 的重要限制

目前 `aia schedule tick` 會：

1. 查詢到期 Schedule。
2. 處理 Misfire Policy。
3. 根據 Schedule 建立 Task。
4. 更新 Schedule 執行資訊。

但它不會自動呼叫：

```text
Scheduler.RunTask()
```

也沒有持續輪詢迴圈。

因此：

**Schedule Tick 不等於 Task Execution。**

目前仍需透過既有手動執行方式處理 Task，或等待 Background Task Worker 完成。

## 6.5 未來命令

以下是規劃中的命令，不代表目前已經實作：

```bash
aia runtime start
aia runtime status
aia runtime stop

aia investment observe
aia investment decisions
aia investment history
aia investment strategies
aia investment review
aia investment report
```

未來應以統一 CLI 架構實作。

# 7. Web Console

## 7.1 啟動方式

目前可透過：

```bash
aia server start
```

啟動 Web Server。

部分版本也可能提供：

```bash
aia web
```

實際命令以 `aia -h` 為準。

## 7.2 Web Server 功能

目前 Web Host 主要負責：

- HTTP Server。
- Login。
- Session。
- Core Routes。
- Agent Module Routes。
- Investment Interface。

預設應使用：

```text
127.0.0.1
```

不應直接開放 LAN 或 Internet。

## 7.3 Web Server 不等於 Runtime

目前：

```text
aia server start
```

只會啟動 HTTP Server。

它不會自動啟動：

- Schedule Polling Worker。
- Task Background Worker。
- Investment Observation Worker。
- Prediction Evaluation Worker。
- Initiative Worker。
- Self-Correction Worker。
- Goal Evolution Worker。

因此，Web Console 即使正常顯示，也不能據此判斷 Agent 正在持續自主運作。

## 7.4 Web Console 建議操作順序

1. 確認 Web Host 狀態。
2. 檢查 Agent Registry。
3. 查看 Task 與 Schedule。
4. 查看 Investment Portfolio。
5. 查看資料來源狀態。
6. 查看最近的投資研究及決策。
7. 檢查 Audit。

目前部分頁面及操作仍在開發中。

# 8. Investment Experiment

## 8.1 實驗目的

Investment Agent 是 AIA 第一個自主性實驗場域。

實驗不只評估股票分析能力，也觀察 Agent 是否能：

- 自主觀察市場。
- 自主研究標的。
- 自主判斷投資機會。
- 自主調整持倉。
- 自主選擇不交易。
- 檢查過去預測。
- 修正錯誤。
- 優化投資策略。
- 長期延續投資目標。

## 8.2 實驗設定

| 項目 | 設定 |
|---|---|
| 初始虛擬資金 | NT$100,000 |
| 市場 | 台灣證券市場 |
| 特別關注 | 台積電 2330 |
| 觀察頻率 | 每個證券營業日 |
| 交易方式 | 虛擬交易 |
| 決策方式 | Agent 自主判斷 |
| 檢查日期 | 每月 1 日 |
| 主要目標 | 投資組合總資產最大化 |

## 8.3 投資標的

Agent 可以自行選擇研究及交易標的。

不預先規定：

- 必須持有台積電。
- 必須持有幾檔股票。
- 必須每天交易。
- 必須維持固定現金比例。
- 必須採用特定投資策略。

Agent 應根據市場資訊及既有投資目標，自行決定。

## 8.4 每日觀察

每個證券營業日應進行：

1. 市場狀態檢查。
2. 台積電資料觀察。
3. 持倉檢查。
4. 重要公告及新聞檢查。
5. 國際市場資訊觀察。
6. Research Snapshot 建立。
7. 歷史記憶檢索。
8. 投資判斷。
9. 決定是否採取行動。

目前尚未完成每日自動觀察 Worker。

因此現階段仍須使用手動同步及其他已存在操作，不可將此流程視為已經自動執行。

## 8.5 不交易也是決策

Agent 不必每天買賣。

例如：

- 市場沒有明確機會。
- 持倉仍符合原有投資論點。
- 目前資料不足。
- 交易成本不值得。
- 等待重要事件確認。

這些都可以是合理的 No-Action Decision。

未來應完整保存不交易理由及重新評估條件。

## 8.6 每月 1 日檢查

使用者每月 1 日檢查：

- 總資產。
- 現金。
- 持股。
- 已實現損益。
- 未實現損益。
- 股利。
- 交易成本。
- 累積報酬。
- 與基準指數比較。
- 重要投資決策。
- 策略修正。

若 1 日不是交易日，應使用最近一個有效交易日的估值資料，並標示估值日期。

## 8.7 Benchmark

可加入：

- 臺灣加權股價指數。
- 元大台灣50（0050）。

Benchmark 是比較參考，不是額外的投資目標。

# 9. Portfolio Accounting

## 9.1 資產計算

```text
Total Assets =
    Cash
    + Market Value of Holdings
    + Recognized Receivables
    - Liabilities
```

## 9.2 帳務原則

所有交易必須留下 Ledger Record。

不得只修改目前持股數量及現金。

交易帳本應能重建歷史資產狀態。

## 9.3 資產估值

估值必須使用可追溯的市場資料。

每筆估值應保留：

- Symbol。
- Quantity。
- Price。
- Price Date。
- Data Source。
- Data Origin。
- Market Value。
- Valuation Time。

如果行情資料過期，必須標示 STALE。

如果行情資料為 MOCK，必須明確隔離。

## 9.4 公司行動

未來應支援：

- 現金股利。
- 股票股利。
- 股票分割。
- 減資。
- 除權息。
- 其他影響股數及資產價值的事件。

公司行動必須形成獨立帳務紀錄。

# 10. Market Data

## 10.1 主要資料來源

| Source | Purpose |
|---|---|
| TWSE | 上市股票及市場資訊 |
| TPEx | 上櫃股票及市場資訊 |
| MOPS | 公司公告及財務資訊 |
| FRED | 美國總體經濟資料 |
| SEC | 美國上市公司申報資料 |
| International News | 國際市場事件與產業資訊 |

## 10.2 資料品質

所有資料必須區分：

- REAL。
- MOCK。
- FALLBACK。
- DERIVED。
- STALE。
- INVALID。

## 10.3 目前資料來源限制

目前 Collector 架構已存在，但部分 Provider 尚未取得真實資料：

- TWSE：有真實 API 呼叫，但仍有固定價格 fallback 及虛構公告。
- TPEx：仍有固定行情及虛構公告。
- FRED：目前使用固定利率資料，尚未完成正式 API 整合。
- SEC：目前有硬編碼申報資料，尚未完成正式資料取得。
- MOPS：正式資料取得與驗證仍須確認。
- International News：尚未完成完整來源整合。

因此：

**目前不可把所有 Collector 輸出視為真實市場資料。**

正式投資實驗啟動前，必須先完成真實資料與模擬資料隔離。

## 10.4 手動同步

例如：

```bash
aia investment sync 2330
```

這是手動資料同步操作。

它不代表每日排程已經啟動，也不代表所有外部資料來源均已完成。

# 11. Prediction & Evaluation

## 11.1 Prediction

Agent 應保存重要投資預測：

- 預測內容。
- 預測方向或數值。
- 信心程度。
- 預測時間。
- 預測期限。
- 預期條件。
- 評估時間。

## 11.2 Evaluation

目前可透過：

```bash
aia investment evaluate
```

手動執行預測評估。

但自動週期評估尚未完成。

## 11.3 Prediction Integrity

不可事後修改原始預測來配合結果。

預測與實際結果必須分開保存。

## 11.4 Self-Correction

未來 Agent 應能：

1. 檢查預測誤差。
2. 分析錯誤原因。
3. 提出策略修正。
4. 建立新策略版本。
5. 進行前向驗證。
6. 評估修正效果。

目前 Self-Correction Engine 尚未實作。

# 12. Memory & Continuity

## 12.1 長期記憶

Investment Agent 應保存：

- 投資目標。
- 投資論點。
- 歷史持倉。
- 歷史交易。
- 歷史決策。
- 預測紀錄。
- 市場事件。
- 策略版本。
- 錯誤分析。
- 修正紀錄。

## 12.2 使用者應能檢視

未來 Web Console 應能回答：

- 當初為什麼買進？
- 當時的資料來源是什麼？
- 當時做了什麼預測？
- 後來發生什麼事？
- Agent 是否重新檢查過原本假設？
- 是否因為過去錯誤而改變策略？

## 12.3 Runtime Restart

完整 Runtime 應能在重新啟動後恢復：

- Current Goal。
- Portfolio。
- Pending Tasks。
- Pending Orders。
- Historical Decisions。
- Investment Thesis。
- Prediction。
- Strategy Version。

目前尚未完成完整 Continuity 能力。

# 13. Runtime Operation

## 13.1 現況

目前 AIA 已有：

- App Bootstrap。
- SQLite。
- Repository。
- EventBus。
- 部分 Task Service。
- Schedule Service。
- Recovery Manager。
- Investment Service。
- Collector Framework。

但尚未完成完整常駐 Runtime。

## 13.2 未來 Runtime

目標架構：

```text
aia runtime start
        │
        ▼
Runtime Supervisor
        │
        ├── Task Worker
        │
        ├── Schedule Worker
        │
        ├── Observation Worker
        │
        ├── Initiative Worker
        │
        └── Evaluation Worker
```

`aia runtime start` 是規劃中的命令，尚未實作。

## 13.3 Runtime Restart

未來 Runtime 必須：

1. 恢復未完成 Task。
2. 檢查執行中的 Task。
3. 處理過期 Schedule。
4. 恢復未完成工作。
5. 避免重複執行。
6. 保存 Recovery Audit。

## 13.4 Shutdown

正常停止時應：

- 停止接受新工作。
- 通知 Worker 停止。
- 等待可安全完成的工作。
- 保存執行狀態。
- 關閉資料庫及其他資源。

不得因 Runtime 停止而遺失歷史交易及決策。

# 14. Security & Governance

## 14.1 Single User

目前定位為單一使用者系統。

不提供多使用者帳號、租戶或 RBAC 架構。

## 14.2 Local Only

Web Host 預設：

```text
127.0.0.1
```

不預設開放 LAN 或 Internet。

## 14.3 No Automatic Financial Trading

本實驗只進行虛擬交易。

不得連接真實券商帳戶自動下單。

不得持有或操作使用者真實資金。

## 14.4 Harness Permission

不得使用：

```bash
--dangerously-skip-permissions
```

所有工具操作應遵守 Harness 權限管理。

## 14.5 Policy & Approval

重大操作應依 Policy 決定是否需要人工核准。

Agent 不得自行：

- 擴張權限。
- 解除治理限制。
- 修改使用者核准條件。
- 刪除 Audit。
- 改寫歷史決策。

## 14.6 Audit

以下操作應留存紀錄：

- 使用者操作。
- Agent 決策。
- Task 執行。
- 工具操作。
- Approval。
- 投資交易。
- 策略變更。
- Goal 變更。

# 15. Daily Operational Checklist

目前仍以人工操作為主。

建議檢查：

1. AIA CLI 是否正常。
2. Antigravity CLI 認證是否有效。
3. SQLite 是否正常。
4. Web Host 是否正常。
5. Collector Provider 是否健康。
6. 市場資料是否為 REAL。
7. Portfolio 是否正確。
8. 是否有待處理 Task。
9. 是否有失敗 Task。
10. 是否有重要投資決策。

當 Background Runtime 完成後，這些檢查應逐步由系統自動執行。

# 16. Monthly Investment Review

每月 1 日進行正式檢查。

## 16.1 Asset Review

- [ ] 初始資金。
- [ ] 目前總資產。
- [ ] 現金。
- [ ] 持股市值。
- [ ] 已實現損益。
- [ ] 未實現損益。
- [ ] 股利。
- [ ] 交易成本。
- [ ] 累積報酬。

## 16.2 Decision Review

- [ ] 本月重要買進決策。
- [ ] 本月重要賣出決策。
- [ ] 增持及減持。
- [ ] 不交易決策。
- [ ] 重大研究結果。
- [ ] 重要預測。

## 16.3 Autonomy Review

- [ ] 是否主動發現事件。
- [ ] 是否延續過去的投資論點。
- [ ] 是否檢查歷史預測。
- [ ] 是否發現錯誤。
- [ ] 是否提出策略修正。
- [ ] 是否維持原始目標。
- [ ] 是否提出合理的目標優化建議。

## 16.4 Monthly Result

投資結果與自主性表現分開記錄。

不能只看總資產增加或減少，就判斷 Agent 是否具備自主性。

# 17. Troubleshooting

## 17.1 Authentication Failure

處理方式：

1. 確認 Antigravity CLI 登入狀態。
2. 重新完成 OAuth 登入。
3. 確認 Harness 可以正常使用。
4. 恢復 `WAITING_FOR_AUTH` Task。

## 17.2 Provider Failure

處理方式：

1. 檢查 Provider 狀態。
2. 檢查網路連線。
3. 檢查外部 API 狀態。
4. 檢查資料格式是否變更。
5. 檢查最後成功取得時間。
6. 確認是否誤用 FALLBACK 或 MOCK。

不可因 Provider 失敗而直接使用假資料冒充真實行情。

## 17.3 Schedule Does Not Execute Task

先確認：

- Schedule 是否到期。
- `aia schedule tick` 是否執行。
- Task 是否建立。
- Task 是否進入 Ready 狀態。

目前沒有常駐 Task Worker，因此 Schedule 建立 Task 後，不代表它會自動執行。

## 17.4 Web Online but Agent Inactive

這是目前架構下可能發生的正常狀況。

原因：

- Web Server 已啟動。
- Background Runtime 尚未啟動或尚未實作。

應分別檢查 Web Host 與 Runtime 狀態。

# 18. Future Operational Model

完整 Runtime 完成後，AIA 的理想操作方式應簡化為：

```text
Start AIA
    │
    ▼
Runtime Recovery
    │
    ▼
Continuous Observation
    │
    ▼
Autonomous Assessment
    │
    ▼
Research / Decision
    │
    ▼
Virtual Execution
    │
    ▼
Verification
    │
    ▼
Learning & Correction
    │
    ▼
Continuous Operation
```

使用者不必每天手動啟動同步、建立研究任務或要求 Agent 檢查持倉。

使用者主要負責：

- 觀察結果。
- 檢查資產。
- 檢視決策。
- 核准重大治理變更。
- 提供必要的人類判斷。

# 19. User Acceptance Principles

使用者評估 AIA 時，應區分三個層次：

## 19.1 Functional Correctness

系統功能是否正確？

例如：

- 資產計算正確。
- 交易帳務正確。
- Task 狀態正確。
- 資料來源正確。

## 19.2 Autonomy Performance

Agent 是否具備自主能力？

例如：

- 是否主動發現問題。
- 是否延續歷史判斷。
- 是否修正過去錯誤。
- 是否能持續優化方法。

## 19.3 Domain Outcome

投資實驗結果如何？

例如：

- 總資產是否增加。
- 是否優於比較基準。
- 是否控制交易成本。
- 是否出現重大回撤。

三個層次必須分開檢查。

# 20. Final Operational Principles

AIA 的操作原則：

1. 使用者設定目標。
2. Agent 自主選擇方法。
3. Agent 持續觀察環境。
4. Agent 自行判斷是否行動。
5. 所有重要行為都要留下紀錄。
6. 所有重要預測都應接受驗證。
7. Agent 必須能從歷史經驗修正方法。
8. 重大治理及目標變更由使用者核准。
9. 歷史紀錄不可任意覆寫。
10. 系統能力必須以實際程式行為驗收。

AIA 的目標不是讓使用者每天提醒 AI 做事。

而是逐步建立一個能夠：

- 記得自己過去做過什麼。
- 理解現在正在發生什麼。
- 主動發現值得處理的問題。
- 自行選擇下一步行動。
- 檢查自己的判斷。
- 修正過去的方法。
- 持續朝著目標前進。

的長期自主 Agent 平台。

**投資只是第一個實驗場域，不是 AIA 的最終用途。**
