# AIA-07：User Manual & Operational Usage Specification

**Document ID:** AIA-07  
**Version:** 1.1  
**Status:** Draft for Implementation / Current CLI Aligned  
**Project:** AIA — Autonomous Intelligence Architecture  
**Runtime Language:** Go  
**AI Execution Backend:** Antigravity CLI (agy) 1.4.12

---

# 1. 文件目的

本文件定義 AIA 的使用者操作方式，包括：

- AIA CLI 的使用方式。
- AIA Server 的啟動。
- Embedded Web UI 的啟動與登入。
- Personal Agent 對話與任務派發。
- Task 建立、執行、恢復與取消。
- Agent 管理。
- Execution 查詢。
- Memory 管理。
- Project 管理。
- 系統檢查與狀態監控。
- 尚未實作功能的識別與後續操作方向。

本文件以目前已實作的 CLI 為基準，不以未來規劃的指令或介面代替實際功能。

## 1.1 文件與規格的關係

| 文件 | 責任 |
|---|---|
| AIA-01 | 產品需求與目標 |
| AIA-02 | 系統架構 |
| AIA-03 | Go Runtime 實作規格 |
| AIA-04 | 測試、驗收與版本交付 |
| AIA-05 | Investment Agent 領域設計 |
| AIA-06 | Embedded Web UI 與 Agent Interface |
| AIA-07 | 使用者操作與維運手冊 |
| README.md | 安裝、快速啟動與基本操作 |

AIA-07 不取代 AIA-03 的技術規格，也不宣稱所有架構規劃都已經完成。

---

# 2. 功能狀態標記

本文件使用以下標記：

| 標記 | 意義 |
|---|---|
| IMPLEMENTED | 已由目前 CLI Help 確認存在 |
| IMPLEMENTED / VERIFY | 指令入口已存在，但完整行為仍需依實際執行結果驗證 |
| PLANNED | 已規劃，但尚未確認有可用操作入口 |
| NOT IN V1 | 第一階段明確不提供 |

不能將 PLANNED 功能描述成目前可用功能。

CLI Help 只能確認命令入口與公開說明；不能單憑 Help 推論所有內部行為、錯誤處理或資料持久化細節。

---

# 3. 系統操作模型

## 3.1 AIA 的主要組成

AIA 是單一使用者的 Personal AI Agent Platform。

主要組成：

1. AIA CLI
2. AIA Runtime
3. AIA Server Process
4. Embedded Web UI / REST / SSE
5. Personal Agent
6. Domain Agents
7. Task Manager
8. Execution Manager
9. Memory Manager
10. Project Manager
11. Antigravity CLI

## 3.2 CLI 與 Web UI

使用者可以透過兩種方式操作 AIA：

- CLI：命令列操作，適合建立、執行、管理與檢查工作。
- Web UI：瀏覽器操作，提供圖形化介面與 API。

兩者應共用 AIA Application Service 與 Domain Service，不應各自實作不同的 Task、Policy 或 Memory 業務邏輯。

目前已確認 CLI 存在 `server` 與 `web` 兩個不同入口，但兩者的程序依附關係及生命週期仍應以實際實作為準。

## 3.3 工作執行模式

AIA 的目標操作模式包括：

| 模式 | 說明 | 狀態 |
|---|---|---|
| Interactive | 使用者透過 Chat 與 Personal Agent 對話 | IMPLEMENTED |
| Task-driven | 建立 Task，再由 Scheduler 執行 | IMPLEMENTED |
| Scheduled | 依照時間排程自動建立 Task | PLANNED |
| Event-triggered | 由外部事件觸發 Task | PLANNED |

目前已經存在 Scheduler，且 `aia task run` 明確使用 Scheduler 執行 READY Task。

但這不代表已提供完整的時間排程管理功能。

---

# 4. CLI 操作

## 4.1 全域命令

目前 `aia -h` 顯示的頂層指令如下：

| Command | 說明 | 狀態 |
|---|---|---|
| `agent` | 管理及檢視 Domain Agent | IMPLEMENTED |
| `chat` | 與 Personal Agent 對話或派發工作 | IMPLEMENTED |
| `completion` | 產生 Shell 自動補全腳本 | IMPLEMENTED |
| `doctor` | 檢查系統依賴與環境健康狀態 | IMPLEMENTED |
| `execution` | 檢視 Execution 與事件串流 | IMPLEMENTED |
| `help` | 顯示說明 | IMPLEMENTED |
| `memory` | 管理長期記憶與 Memory Candidate | IMPLEMENTED |
| `project` | 管理 Workspace Project | IMPLEMENTED |
| `server` | 管理 AIA Server Process | IMPLEMENTED |
| `status` | 顯示系統摘要及執行中 Task | IMPLEMENTED |
| `task` | 管理及執行 Agent Task | IMPLEMENTED |
| `version` | 顯示版本資訊 | IMPLEMENTED |
| `web` | 啟動 Embedded Web UI 與 REST/SSE API | IMPLEMENTED |

查詢命令：

```bash
aia --help
aia -h
aia <command> --help
```

全域設定：

```bash
aia --config /path/to/config.yaml <command>
```

預設設定檔位置：

```text
~/.config/aia/config.yaml
```

或：

```text
./configs/aia.yaml
```

---

# 5. AIA Server 與 Web UI

## 5.1 Server Process

**狀態：IMPLEMENTED / VERIFY**

目前 Server 命令：

```bash
aia server --help
```

已確認子命令：

```text
start    Start the AIA server
```

啟動：

```bash
aia server start
```

Server 負責 AIA Server Process 的啟動。

目前 CLI Help 尚未顯示 `stop`、`shutdown` 或 `restart` 子命令，因此本文件不宣稱這些命令已經存在。

Server 的背景執行方式、程序鎖定、IPC、停止方式及重啟 Recovery 行為，應依實際實作與後續測試確認。

## 5.2 Embedded Web UI

**狀態：IMPLEMENTED**

啟動命令：

```bash
aia web
```

目前參數：

| 參數 | 說明 | 預設值 |
|---|---|---|
| `--host` | Web Server 綁定位址 | `127.0.0.1` |
| `--port` / `-p` | Web Server Port | `8080` |
| `--config` / `-c` | 設定檔路徑 | 預設設定檔 |

例如：

```bash
aia web --host 127.0.0.1 --port 8080
```

預設瀏覽器位址：

```text
http://127.0.0.1:8080
```

### 5.2.1 一次性登入 Token

Web 啟動時：

1. AIA 產生密碼學安全的隨機一次性登入 Token。
2. Token 印在啟動輸出。
3. 使用者開啟瀏覽器。
4. 使用 Token 進行登入。
5. Token 兌換為 Server-Side Session Cookie。
6. 成功兌換後，原 Token 立即失效。

登入後的請求必須逐一檢查：

- Session 是否有效。
- 請求是否具備必要 Permission。
- CSRF 防護。
- Origin / Host 驗證。
- 其他適用的安全政策。

Session Cookie 應使用適當的 HttpOnly、SameSite 與 Secure 屬性；實際 Secure 設定需配合部署模式。

### 5.2.2 Web 安全限制

AIA v1：

- 單一使用者。
- 預設只允許本機存取。
- 不開放 LAN 或 Internet 遠端管理。
- 不提供多使用者帳號管理。
- 不使用動態 Go Plugin。

若使用者自行修改 Host 綁定位址，不能因此視為已完成遠端安全部署。

### 5.2.3 Web 與 Task 的生命週期

Web UI 是操作介面，不應成為 Task 的執行依附條件。

關閉瀏覽器不應中止 Task。

Web Server 停止與 AIA Runtime 停止是不同事件，實際程序生命週期以 Runtime 實作為準。

---

# 6. Personal Agent Chat

**狀態：IMPLEMENTED**

Chat 是使用者與 Personal Agent 互動的主要入口之一。

命令：

```bash
aia chat
```

直接提供訊息：

```bash
aia chat "請協助我研究某個技術主題"
```

目前已確認參數：

| 參數 | 說明 |
|---|---|
| `--json` | 以結構化 JSON 輸出回應 |
| `--project` | 指定對話使用的 Project ID |
| `--config` | 指定設定檔 |

例如：

```bash
aia chat "整理這個專案目前的技術問題"
```

指定 Project：

```bash
aia chat --project PROJECT_ID "請檢查這個專案的架構"
```

JSON 輸出：

```bash
aia chat --json "請摘要目前工作狀態"
```

Chat 可以處理一般對話，也可以根據使用者需求派發工作。

是否建立 Task、指派哪個 Agent、是否需要人工核准，應由 AIA 的工作流程與 Policy 決定。

使用者不應假設每一次 Chat 都必然建立 Task。

---

# 7. Task Management

**狀態：IMPLEMENTED**

## 7.1 Task 命令

目前已確認的子命令：

| 命令 | 用途 |
|---|---|
| `aia task create` | 建立 Task |
| `aia task list` | 列出 Task |
| `aia task show` | 檢視 Task 詳細資料 |
| `aia task run` | 使用 Scheduler 執行 READY Task |
| `aia task resume` | 恢復失敗或受阻的 Task |
| `aia task cancel` | 取消 Task |

查詢：

```bash
aia task --help
```

## 7.2 建立 Task

```bash
aia task create --help
```

建立 Task 的具體參數，以目前 CLI 子命令 Help 為準。

Task 的基本概念：

- Task 表示一項需要完成的工作。
- Task 可以被指派給 Domain Agent。
- Task 可以具有依賴關係。
- Task 執行時產生 Execution。
- Task 完成前應經過適當的結果驗證。
- Task 的狀態與 Execution 的狀態不可混為一談。

## 7.3 Task 狀態

AIA 已定義的 Task 狀態如下：

| 狀態 | 意義 |
|---|---|
| CREATED | Task 已建立，尚未進入可執行狀態 |
| READY | 已具備執行條件 |
| RUNNING | 正在執行 |
| WAITING_APPROVAL | 等待使用者核准 |
| WAITING_FOR_AUTH | 等待使用者完成外部身分驗證 |
| RETRY_WAIT | 等待重試 |
| COMPLETED | 工作完成並通過必要驗證 |
| FAILED | 工作失敗 |
| CANCELLED | 工作已取消 |

注意：

- AIA v1 不定義 `QUEUED` 狀態。
- `READY` 是可供 Scheduler 執行的 Task 狀態。
- Execution SUCCESS 不一定代表 Task COMPLETED。
- 若結果尚未完成 Validation，Task 不應提前標記為 COMPLETED。

## 7.4 執行 Task

```bash
aia task run --help
```

目前已確認：

`aia task run` 會使用 Scheduler 執行 READY Task。

執行前應確認：

- Task 狀態為 READY。
- 指派的 Agent 已啟用。
- 所需依賴已完成。
- Policy 沒有阻止執行。
- 所需外部認證可用。

## 7.5 查詢 Task

```bash
aia task list
aia task show --help
```

Task 詳細資訊應用於確認：

- Task ID。
- Task 狀態。
- Task 類型。
- 指派 Agent。
- 執行結果。
- 依賴關係。
- 失敗或等待原因。

實際輸出欄位以 CLI 為準。

## 7.6 恢復 Task

```bash
aia task resume --help
```

適用於符合恢復條件的失敗或受阻 Task。

例如：

- 使用者完成必要登入。
- 使用者處理等待核准事項。
- 暫時性錯誤已排除。
- Task 可以安全地重新執行。

恢復時必須遵守 Retry Limit、Idempotency 與 Recovery Policy。

不得將所有 FAILED Task 無條件重新執行。

## 7.7 取消 Task

```bash
aia task cancel --help
```

取消 Task 不代表已經執行的外部副作用一定可以撤銷。

AIA 必須保留取消事件及相關 Audit 記錄。

---

# 8. Agent Management

**狀態：IMPLEMENTED**

## 8.1 命令

```bash
aia agent --help
```

目前已確認：

| 命令 | 用途 |
|---|---|
| `aia agent list` | 列出已註冊 Agent |
| `aia agent show` | 檢視 Agent 詳情 |
| `aia agent enable` | 啟用 Agent |
| `aia agent disable` | 停用 Agent |

## 8.2 內建 Agent

目前 Runtime 已確認的內建 Agent：

- personal-agent
- coding-agent
- writing-agent
- investment-agent
- video-agent

實際註冊清單以：

```bash
aia agent list
```

為準。

Agent 是否啟用會影響其能否被派發工作。

## 8.3 Agent 職責

Personal Agent：

- 接收使用者需求。
- 理解工作目的。
- 判斷是否需要派發 Task。
- 選擇適合的 Domain Agent。
- 協調工作與結果。

Domain Agent：

- 執行特定領域工作。
- 遵循 AIA Runtime 的 Policy 與 Task 管理。
- 回傳工作結果。
- 不直接取代 AIA 全域 Task、Memory 或 Audit 管理。

Agent 註冊採編譯期靜態註冊。

AIA v1 不支援執行期動態載入 Go Plugin。

---

# 9. Execution Management

**狀態：IMPLEMENTED**

Execution 是 Task 的一次執行紀錄。

同一個 Task 在符合重試條件時，可能具有多次 Execution。

## 9.1 命令

```bash
aia execution --help
```

目前已確認：

| 命令 | 用途 |
|---|---|
| `aia execution show` | 顯示 Execution 詳細資訊 |
| `aia execution events` | 列出 Execution 的事件串流 |

## 9.2 檢視 Execution

```bash
aia execution show --help
```

用於檢查：

- Execution ID。
- 所屬 Task。
- 執行狀態。
- 執行時間。
- Agent 執行資訊。
- 結果與錯誤資訊。

實際欄位依 CLI 輸出為準。

## 9.3 檢視事件

```bash
aia execution events --help
```

事件串流可用於追蹤執行過程。

事件紀錄應保留足夠資訊，以支援：

- 問題診斷。
- Crash Recovery。
- 重試判斷。
- 執行結果追溯。
- Audit。

---

# 10. Memory Management

**狀態：IMPLEMENTED**

## 10.1 命令

```bash
aia memory --help
```

目前已確認：

| 命令 | 用途 |
|---|---|
| `aia memory list` | 列出有效 Memory |
| `aia memory search` | 依關鍵字搜尋 Memory |
| `aia memory candidate` | 管理 Memory Candidate |

## 10.2 搜尋 Memory

```bash
aia memory search --help
```

Memory Search 用於尋找先前保存的長期資訊。

## 10.3 列出 Memory

```bash
aia memory list
```

顯示目前有效的長期記憶。

## 10.4 Memory Candidate

```bash
aia memory candidate --help
```

Memory Candidate 是尚未正式成為有效 Memory 的候選資料。

使用者應檢視候選內容，再依實際提供的操作方式進行核准或管理。

AIA 不應將尚未核准的候選內容視為已經確定的長期事實。

Memory Candidate 與等待核准的 Task 是不同領域物件，不應將兩者混成同一種實體。

---

# 11. Project Management

**狀態：IMPLEMENTED**

## 11.1 命令

```bash
aia project --help
```

目前已確認：

| 命令 | 用途 |
|---|---|
| `aia project create` | 建立 Project |
| `aia project list` | 列出 Project |
| `aia project show` | 檢視 Project |
| `aia project archive` | 封存 Project |

## 11.2 建立 Project

```bash
aia project create --help
```

建立時所需參數以實際 Help 為準。

## 11.3 列出與檢視

```bash
aia project list
aia project show --help
```

Project 用於組織相關工作與對話上下文。

Chat 可使用 `--project` 指定 Project ID：

```bash
aia chat --project PROJECT_ID "請檢查這個專案的工作內容"
```

## 11.4 封存 Project

```bash
aia project archive --help
```

封存是 Project 生命週期管理的一部分，不應假設等同於刪除。

---

# 12. System Health & Status

**狀態：IMPLEMENTED**

## 12.1 Doctor

```bash
aia doctor
```

用途：

- 檢查系統依賴。
- 檢查 agy Binary。
- 檢查 SQLite。
- 檢查環境健康狀態。

當 AIA 無法正常工作時，應優先執行 Doctor。

## 12.2 Status

```bash
aia status
```

用途：

- 顯示系統摘要。
- 顯示執行中的 Task。

Status 是快速檢視入口，不取代 Task、Execution 的詳細查詢。

## 12.3 Version

```bash
aia version
```

顯示目前 AIA 版本資訊。

## 12.4 Completion

```bash
aia completion --help
```

用於產生指定 Shell 的自動補全腳本。

---

# 13. Investment Agent

**狀態：IMPLEMENTED**

目前 Runtime 已註冊內建 `investment-agent`，且已提供完整頂層 CLI 工具：

```bash
aia investment --help
```

可用指令包含：
- `aia investment portfolio`：檢視投資組合、可用現金、資金上限與持倉。
- `aia investment sync [symbol]`：自 TWSE OpenAPI 與 MOPS 抓取最新行情與重大訊息。
- `aia investment proposals`：列出 Go Runtime 產生之交易提案。
- `aia investment approve <id>`：核准交易提案。
- `aia investment reject <id>`：駁回交易提案。
- `aia investment predictions`：列出預測紀錄與狀態。
- `aia investment evaluate`：執行到期預測結算並計算 Brier Score 校準評分卡。

## 13.1 預定功能

Investment Agent 的目標能力：

- 管理使用者指定的股票池。
- 讀取使用者提供的持倉快照。
- 收集台灣市場與國際公開資訊。
- 分析公司、產業與總體經濟。
- 產生 BUY / HOLD / REDUCE / SELL / WATCH / NO_ACTION 建議。
- 追蹤歷史判斷與預測結果。
- 評估策略與預測品質。

## 13.2 Go 與 AI 的責任邊界

agy / Agent：

- 產生投資判斷。
- 說明風險。
- 分析資訊與影響路徑。
- 提出建議方向。

Go Runtime：

- 驗證股票代號與交易方向。
- 計算建議股數。
- 計算交易成本。
- 計算操作後現金與持倉。
- 檢查資金上限。
- 驗證交易提案是否符合使用者設定的限制。

AI 不得直接決定或繞過 Go 的資金限制。

## 13.3 交易成本計算與可配置性 (Transaction Cost Configurability)

為了確保投資提案之財務數字精確且具法規與券商一致性，交易費用、稅額與淨收支一律由 Go Runtime 程式碼實作，LLM 不得介入計算。

1. **現行實作之預設計算參數（Default Parameters）**：
   目前實作採用台灣證券市場一般現股交易標準費率作為系統預設值：
   - **標準券商手續費率 (`FeeRate`)**：`0.1425%`（買賣雙向）。
   - **電子下單折扣率 (`FeeDiscountRate`)**：預設 `1.0`（無折扣）。
   - **最低手續費門檻 (`MinFee`)**：預設 `NT$20`（未滿 20 元以 20 元計收）。
   - **證券交易稅率 (`TaxRate`)**：預設 `0.3%`（僅於賣出時課徵，買入為 0）。

2. **參數可配置性（Configurability）**：
   上述費率僅為當前預設基準，絕非不可修改之硬編碼限制。為因應日後市場制度變更、券商個別費率折讓或跨商品交易，Go Runtime 定義 `CostConfig` 與 Profile 擴充配置機制：
   - **券商專屬折讓**：支援於 Portfolio Profile 中定義專屬折扣（如 6 折 `0.6`、2.8 折 `0.28` 等）。
   - **活動最低手續費**：支援個別券商活動之低門檻手續費設定（如 NT$1 元優惠）。
   - **不同商品與法規稅率調整**：支援特定證券商品稅率客製化（如 ETF 證交稅為 `0.1%`、現股當沖降稅為 `0.15%`、或未來主管機關修訂稅率）。
   - **多市場/多資產擴充介面**：架構上保留 `CostConfig` 注入介面，日後擴展美股或其他海外市場時動態套用，避免文件或程式碼產生過時死規則。

## 13.4 第一階段限制

- 不自動下單。
- 不持有使用者資金。
- 不保存券商交易憑證。
- 不得擅自擴大使用者指定的股票池。
- 可以建議不交易。

---

# 14. Scheduler 與自動化

**狀態：IMPLEMENTED**

目前已提供定時排程引擎與頂層命令：

```bash
aia schedule --help
```

可用指令包含：
- `aia schedule list`：列出所有自動排程、Cron 表達式與下次執行時間。
- `aia schedule create --name <name> --cron <cron> [--agent <agent>]`：新增定時排程（支援五欄式 Cron 與 `@daily`、`@hourly` 等別名）。
- `aia schedule trigger <id>`：立即手動派發該排程工作為 READY Task。
- `aia schedule tick`：背景執行或手動檢查到期排程並觸發執行。

Misfire Policy 支援 `SKIP`、`RUN_ONCE` 與 `CATCH_UP`，確保電腦重啟或離線時不造成大量任務積壓。

## 14.1 排程設計原則與責任分離

Schedule 與 Task 嚴格分離：

- Schedule：定義何時產生工作（觸發規則與樣板）。
- Task：代表一次具體工作（擁有狀態機與審查）。
- Execution：代表一次實際執行嘗試（追蹤日誌與事件）。

電腦關機期間無法執行工作。透過 Misfire Policy（`SKIP`、`RUN_ONCE`、`CATCH_UP`），系統在重新啟動後依政策決定如何補救錯過的時間點，避免工作爆炸積壓。
- RUN_IF_WITHIN_WINDOW
- REQUIRE_APPROVAL

投資分析等工作不得在系統離線數日後無條件一次補跑所有歷史排程。

---

# 15. Policy、Approval 與 Audit

**狀態：IMPLEMENTED / VERIFY**

AIA 必須在 Task 執行前後遵守 Policy。

Policy 的目的：

- 限制高風險操作。
- 要求必要核准。
- 控制外部工具使用。
- 保護 Workspace 與系統資料。
- 保留可追溯紀錄。

## 15.1 Approval

目前 Task 狀態包含：

`WAITING_APPROVAL`

這代表 Task 可以進入等待人工核准的狀態。

但目前不應假設存在獨立的全域 Approval 管理 CLI。

Web Console 未來可將等待核准的 Task 與 Memory Candidate 整合呈現，但兩者仍是不同實體。

## 15.2 Audit

重要操作應保留 Audit：

- Task 建立、執行、恢復、取消。
- Agent 啟用與停用。
- Memory Candidate 核准。
- Policy 判斷。
- 外部工具操作。
- Recovery 與重試。

Audit 是系統追溯資料，不等同於一般使用者對話紀錄。

---

# 16. Web Console 功能規劃

**狀態：FRAMEWORK IMPLEMENTATION / DOMAIN UI PLANNED**

Web Console 的目標功能：

| 功能 | 目標 |
|---|---|
| Dashboard | 系統概況 |
| Tasks | Task 查詢與管理 |
| Agents | Agent 清單與狀態 |
| Executions | 執行紀錄 |
| Pending Items | 等待核准的 Task 與 Memory Candidate |
| Memory | 長期記憶 |
| Projects | Project 管理 |
| Audit | 操作紀錄 |
| Settings | 系統設定 |
| Investment | 投資研究介面 |

以上是產品功能方向，不代表每一個頁面都已完成。

Web UI 應與 CLI 共用 Application Service。

Agent Domain UI 透過 Agent Web Module 提供，不允許 Agent 直接操作全域 DB 或繞過 Runtime Policy。

---

# 17. 備份與維護

**狀態：OPERATIONAL GUIDANCE**

AIA 使用 SQLite 保存系統資料。

使用者應定期備份：

- SQLite Database。
- 設定檔。
- Workspace。
- 必要的專案資料。

備份應採一致性方式，避免在資料庫寫入期間直接複製造成不完整備份。

agy OAuth Token 由 agy 自行管理；AIA 不讀取、不複製、不解析其 Token。

使用者應自行確保 agy 登入狀態有效。

若認證失效，相關 Task 應進入 `WAITING_FOR_AUTH`，待使用者完成登入後再恢復。

---

# 18. AIA v1 架構限制

以下為正式架構限制，不是待完成的功能：

1. Single User：只支援單一系統擁有者。
2. Local Only：Web 預設只綁定 `127.0.0.1`。
3. Static Registration：Agent 與 Agent Web Module 採編譯期靜態註冊。
4. No Dynamic Go Plugin：不支援執行期動態載入 Go Plugin。
5. No Automatic Financial Trading：不提供自動金融交易。

即使只有單一使用者，仍必須保留：

- Policy。
- Approval。
- Audit。
- Session。
- CSRF。
- Permission。

這些機制是系統安全與 Agent 治理要求，不是多使用者功能。

---

# 19. 目前系統功能實作與階段規劃對照 (Version Maturity Baseline)

目前系統已完成 **P1–P3、P6** 核心與 Web 基準，並明確標記當前完成度與下一階段（P4/P5、排程與專屬 CLI）規劃：

| 模組代號 | 功能項目 | 目前狀態 | 實作現況說明 |
|---|---|---|---|
| CLI | CLI 基本入口與工具 | IMPLEMENTED | `aia --help`, `doctor`, `status`, `version` |
| CLI | Personal Agent Chat | IMPLEMENTED | `aia chat`（對話、意圖識別與 Task 派發） |
| CLI | Task 管理 | IMPLEMENTED | `aia task create/list/show/run/resume/cancel` |
| CLI | Agent 管理 | IMPLEMENTED | `aia agent list/show/enable/disable` |
| CLI | Execution 追蹤 | IMPLEMENTED | `aia execution show/events` |
| CLI | Memory 管理 | IMPLEMENTED | `aia memory list/search/candidate` |
| CLI | Project 管理 | IMPLEMENTED | `aia project create/list/show/archive` |
| **P1** | Embedded Web 基礎設施 | IMPLEMENTED | Web Host、一次性 Token 啟動產生與單次兌換、Server-Side Session、REST/SSE API、CSRF/Origin 安全中介軟體 |
| **P2** | Global Web Console | IMPLEMENTED | Dashboard 系統摘要、Task 列表與生命週期操作、Agent 狀態檢視、Memory 檢視與 Candidate 審批、待處理事項（Pending Review）、Audit Trail 稽核軌跡 |
| **P3** | Investment 核心能力 | IMPLEMENTED | 投資組合 Profile/Snapshot 管理、資金與持倉比例限制檢查、Go 建議股數計算、交易手續費與稅費計算（`CostConfig` 可配置）、Trade Proposal 產生與駁回審計 |
| **P6** | Agent UI 模組 | IMPLEMENTED | 原生前端（HTML/CSS/JS）、Agent 專屬 mux 隔離路由、Portfolio 持倉檢視、行情與重大訊息、校準評分卡 |
| **P4** | 實際資料抓取 | IMPLEMENTED | TWSE OpenAPI 日 K 數據抓取、公開資訊觀測站 (MOPS) 重大訊息與財經新聞整合、限流保護與快取 |
| **P5** | 預測與評估 | IMPLEMENTED | 歷史預測事後追蹤、市場數據到期結算、Brier Score 信心度校準計算、方向性準確率 (Directional Accuracy) |
| SCHED | Schedule 管理 CLI 與 Cron | IMPLEMENTED | `aia schedule` CLI、標準五欄式 Cron 與別名解析、Misfire Policy（SKIP, RUN_ONCE, CATCH_UP）、排程自動派發 Task |
| CLI-INV | Investment 管理 CLI | IMPLEMENTED | `aia investment` CLI（`portfolio`, `sync`, `proposals`, `approve`, `reject`, `predictions`, `evaluate`） |
| CORE | 多使用者隔離 | NOT IN V1 | 單機單使用者 Personal Agent 架構 |
| CORE | 遠端 Web 暴露管理 | NOT IN V1 | 預設僅監聽 127.0.0.1 本機安全防護 |
| CORE | 動態 Go Plugin | NOT IN V1 | 採用標準 Go 編譯整合 |
| CORE | 自動券商下單交易 | NOT IN V1 | 第一階段堅守不自動交易、不持有憑證原則 |

---

# 20. 驗收標準

| ID | 驗收項目 | 驗收條件 |
|---|---|---|
| U-01 | CLI Help | `aia -h` 正常顯示 |
| U-02 | Doctor | 能檢查依賴與環境 |
| U-03 | Status | 能顯示系統摘要及執行中 Task |
| U-04 | Server | `aia server start` 可啟動 Server |
| U-05 | Web | `aia web` 可啟動 Web UI/API |
| U-06 | Web Login | 一次性 Token 可兌換 Session |
| U-07 | Chat | 可與 Personal Agent 對話 |
| U-08 | Task Create | 可建立 Task |
| U-09 | Task Run | 可使用 Scheduler 執行 READY Task |
| U-10 | Task Resume | 可恢復符合條件的 Task |
| U-11 | Task Cancel | 可取消符合條件的 Task |
| U-12 | Agent Management | 可列出、檢視、啟用及停用 Agent |
| U-13 | Execution | 可查詢 Execution 與事件 |
| U-14 | Memory | 可列出、搜尋及管理 Candidate |
| U-15 | Project | 可建立、列出、檢視及封存 Project |
| U-16 | Version / Completion | 可顯示版本與產生 Shell Completion |
| U-17 | No Phantom Commands | 文件不得宣稱不存在的 CLI 命令已可用 |
| U-18 | Runtime / Web Boundary | Web 關閉不得造成已執行 Task 遺失 |

---

# 21. 後續版本規劃

後續版本可以逐步增加：

- Schedule 管理與 Misfire Policy。
- 完整 Web Console。
- Agent Domain UI。
- Investment CLI 與研究工作流程。
- Data Source Registry。
- 投資預測與績效評估。
- 更完整的 Audit 查詢。
- 備份與還原工具。
- Runtime Server 的完整生命週期管理。

每項功能在正式交付前，必須同步更新：

1. AIA-01 需求。
2. AIA-02 架構。
3. AIA-03 Runtime 實作規格。
4. AIA-04 測試與驗收。
5. AIA-07 使用手冊。
6. README.md。

---

**End of AIA-07 v1.1**
