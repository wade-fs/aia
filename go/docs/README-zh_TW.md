# AIA — 自主智慧架構（Autonomous Intelligence Architecture）

AIA 是一個以 Go 語言實作的個人 AI Agent 平台。

它負責協調不同領域的 Agent、管理任務與執行程序、持久化長期記憶、組織工作空間專案，並整合 Antigravity CLI。

AIA 的設計定位是單一使用者、以本機優先（Local-First）為原則的 AI Agent Runtime。

---

## 1. 功能特色

目前 CLI 提供以下功能：

- Personal Agent 對話與任務派發。
- Domain Agent 管理。
- Task 建立、執行、取消與恢復。
- Execution 執行紀錄檢視與事件串流。
- 長期記憶與記憶候選項管理。
- 工作空間專案管理。
- 系統健康狀態檢查。
- 內嵌式 Web UI 與 REST/SSE API。
- Antigravity CLI 整合。

目前已註冊的內建 Agent：

- `personal-agent`
- `coding-agent`
- `writing-agent`
- `investment-agent`
- `video-agent`

實際註冊與啟用狀態可透過以下指令查詢：

```bash
aia agent list
```

---

## 2. 系統需求

- Linux 環境。
- 從原始碼建置時需要 Go Runtime / Build Environment。
- Antigravity CLI（`agy`）1.4.12。
- SQLite 支援。
- 需要執行 AI 工作時，必須具備有效的 Antigravity CLI 認證。

AIA 不負責管理或複製 Antigravity OAuth 認證憑證。

使用者必須透過 Antigravity CLI 自行完成登入認證。

---

## 3. 建置

在 Go 專案目錄下執行：

```bash
go build -o bin/aia ./cmd/aia
```

驗證執行檔：

```bash
./bin/aia version
```

檢查系統相依套件：

```bash
./bin/aia doctor
```

---

## 4. CLI 說明

顯示所有可用指令：

```bash
./bin/aia --help
```

顯示特定指令的說明：

```bash
./bin/aia task --help
./bin/aia agent --help
./bin/aia memory --help
```

目前 CLI 提供以下頂層指令：

| 指令 | 說明 |
|---|---|
| `agent` | 管理及檢視領域 Agent |
| `chat` | 與 Personal Agent 對話或派發任務 |
| `completion` | 產生 Shell 自動補全腳本 |
| `doctor` | 檢查系統相依套件與執行環境 |
| `execution` | 檢視執行紀錄與事件 |
| `investment` | 管理投資組合、同步資料、檢視預測與交易提案 |
| `memory` | 管理長期記憶與記憶候選項 |
| `project` | 管理工作空間專案 |
| `schedule` | 管理自動化週期任務排程（Cron 與 Misfire Policies） |
| `server` | 管理 AIA Server Process |
| `status` | 顯示系統摘要與執行中的任務 |
| `task` | 管理與執行任務 |
| `version` | 顯示版本資訊 |
| `web` | 啟動內嵌式 Web UI 與 API Server |

---

## 5. 設定

預設設定檔位置：

```text
~/.config/aia/config.yaml
./configs/aia.yaml
```

指定設定檔：

```bash
./bin/aia --config /path/to/config.yaml doctor
```

全域設定檔參數：

```text
-c, --config
```

---

## 6. 系統健康狀態

執行環境檢查：

```bash
./bin/aia doctor
```

顯示系統狀態：

```bash
./bin/aia status
```

顯示版本：

```bash
./bin/aia version
```

`doctor` 會檢查系統相依套件、agy、SQLite 與執行環境健康狀態。

`status` 會提供系統摘要及執行中 Task 的相關資訊。

---

## 7. Server

啟動 AIA Server Process：

```bash
./bin/aia server start
```

## 7. Server 與背景常駐服務管理

手動啟動 AIA Server Process：

```bash
./bin/aia server start
```

### Systemd User 背景常駐服務（推薦）

AIA 提供透過 systemd user service 進行常駐後台管理，具備自動重啟、開機常駐（linger）、災難恢復與排程工作自動觸發：

```bash
# 1. 冪等安裝與啟動（自動偵測避免重複安裝，平滑重載）
make install_service

# 2. 檢視背景服務即時狀態
make status_service

# 3. 取得最新 Web UI 一次性登入 Token
make token

# 4. 檢視服務即時日誌
make logs_service

# 5. 重啟背景服務
make restart_service

# 6. 完全停止並解除安裝背景服務（保留數據目錄）
make uninstall_service
```

敏感金鑰憑證（如 `FRED_API_KEY`、`FINMIND_TOKEN`）皆安全存放在 `~/.config/aia/service.env` 中，權限嚴格設為 `0600`，由 systemd 於服務啟動時自動載入。

---

## 8. 內嵌式 Web UI

啟動內嵌式 Web UI 與 REST/SSE API Server：

```bash
./bin/aia web
```

預設設定：

| 參數 | 預設值 |
|---|---|
| Host | `127.0.0.1` |
| Port | `8080` |

瀏覽器開啟：

```text
http://127.0.0.1:8080
```

自訂 Host 與 Port：

```bash
./bin/aia web --host 127.0.0.1 --port 8080
```

Web Server 啟動時，會產生一組經過密碼學安全隨機數產生器產生的一次性登入 Token，並將 Token 印出至終端機。

使用該 Token 透過瀏覽器登入。

登入成功後，Token 會被交換為 Server-Side Session，原本的 Token 隨即失效，不可再次使用。

**安全性：**

AIA v1 的設計目標是本機單一使用者環境。請勿將 Web Server 暴露於區域網路（LAN）或公開網際網路。

---

## 9. 與 Personal Agent 對話

啟動互動式對話：

```bash
./bin/aia chat
```

直接傳送訊息：

```bash
./bin/aia chat "Help me review this software architecture."
```

使用結構化 JSON 輸出：

```bash
./bin/aia chat --json "Summarize the current project status."
```

將對話關聯至指定 Project：

```bash
./bin/aia chat --project PROJECT_ID "Review this project's architecture."
```

可用參數：

| 參數 | 說明 |
|---|---|
| `--json` | 輸出結構化 JSON |
| `--project` | 指定 Project ID |
| `-c`, `--config` | 指定設定檔 |

Personal Agent 可以直接回應使用者，也可以在適當情況下將工作派發給合適的 Domain Agent。

並非每一次對話都必然建立 Task。

---

## 10. Tasks（任務管理）

檢視 Task 指令：

```bash
./bin/aia task --help
```

可用指令：

| 指令 | 說明 |
|---|---|
| `task create` | 建立 Task |
| `task list` | 列出 Tasks |
| `task show` | 顯示 Task 詳細資訊 |
| `task run` | 透過 Scheduler 執行 READY 狀態的 Task |
| `task resume` | 恢復失敗或受阻的 Task |
| `task cancel` | 取消 Task |

使用前先檢視各指令的參數：

```bash
./bin/aia task create --help
./bin/aia task run --help
./bin/aia task show --help
./bin/aia task resume --help
./bin/aia task cancel --help
```

典型操作流程：

1. 建立 Task。
2. 檢視 Task。
3. 執行處於 READY 狀態的 Task。
4. 檢視產生的 Execution。
5. 如果符合恢復條件，則恢復該 Task。

Task 狀態：

- `CREATED`
- `READY`
- `RUNNING`
- `WAITING_APPROVAL`
- `WAITING_FOR_AUTH`
- `RETRY_WAIT`
- `COMPLETED`
- `FAILED`
- `CANCELLED`

系統不存在 `QUEUED` 這個 Task 狀態。

Execution 成功不代表 Task 必然進入 `COMPLETED` 狀態。系統必須先完成必要的結果驗證，才能將 Task 標記為完成。

---

## 11. Agents（Agent 管理）

列出已註冊的 Agents：

```bash
./bin/aia agent list
```

檢視 Agent 詳細資訊：

```bash
./bin/aia agent show --help
```

啟用或停用 Agent：

```bash
./bin/aia agent enable --help
./bin/aia agent disable --help
```

目前提供的 Agent 操作：

- List：列出 Agents。
- Show：檢視 Agent。
- Enable：啟用 Agent。
- Disable：停用 Agent。

實際可用的 Agent，取決於其註冊狀態與啟用狀態。

---

## 12. Executions（執行紀錄）

列出 Execution 相關指令：

```bash
./bin/aia execution --help
```

檢視 Execution 詳細資訊：

```bash
./bin/aia execution show --help
```

檢視 Execution 事件：

```bash
./bin/aia execution events --help
```

Execution 紀錄代表執行某個 Task 時所產生的個別執行嘗試。

可用於：

- 檢視執行結果。
- 診斷失敗原因。
- 檢視事件歷史紀錄。
- 了解重試與恢復過程。

---

## 13. Memory（記憶管理）

檢視 Memory 指令：

```bash
./bin/aia memory --help
```

列出目前有效的記憶：

```bash
./bin/aia memory list
```

搜尋記憶：

```bash
./bin/aia memory search --help
```

管理 Memory Candidates：

```bash
./bin/aia memory candidate --help
```

Memory Candidates 是尚未正式成為有效記憶的長期記憶候選項。

在將候選項內容視為已確立的長期資訊之前，應先進行審查。

---

## 14. Investment Core 與交易成本計算

AIA 在 AI 分析與 Go Runtime 財務執行之間，採取嚴格的架構責任分界：

- **agy / LLM：** 負責市場研究、新聞綜合分析、推理，以及提出交易構想（BUY / SELL / HOLD）。
- **Go Runtime：** 負責驗證股票代號、計算精確股數（整股 1,000 股或零股）、檢查可用現金與投資組合資金上限、計算交易成本與稅費，以及產生正式 Trade Proposal。

LLM 的輸出絕不允許直接成為可執行的交易指令。

### 交易成本參數與可配置性

交易成本與稅費由 Go 原生程式碼計算（`internal/domain/investment/calculator.go`）。

**1. 預設參數（台灣股票）**

- **券商手續費率（`FeeRate`）：** 買進與賣出均為 `0.1425%`（0.001425）。
- **券商手續費折扣率（`FeeDiscountRate`）：** 預設 `1.0`（不打折）。
- **最低手續費（`MinFee`）：** 預設 `NT$20`。
- **證券交易稅率（`TaxRate`）：** 賣出時為 `0.3%`（0.003），買進時為 0。

**2. 可配置性**

這些費率並非固定不變或寫死在程式中的商業限制。Go Runtime 使用 `CostConfig` 結構，可依不同投資組合設定檔（透過 `configuration_json`）或 Runtime 設定進行自訂。

支援項目包括：

- **券商手續費折扣：** 支援自訂電子交易折扣，例如原始費率的 60%（`0.6`）或 28%（`0.28`）。
- **特殊最低手續費：** 支援優惠最低手續費，例如 NT$1。
- **資產類型與法規相關稅率：** 支援 ETF 法定稅率（`0.1%`）、當沖稅率優惠（`0.15%`），以及未來法定稅率調整。
- **多市場擴充能力：** 為美股或其他海外資產預留擴充能力，不必因此破壞既有文件或計算規則。

---

## 15. 實作狀態與版本成熟度基準

AIA 目前版本為 **v1.0（核心與 Web 基礎版本）**。

系統實作成熟度分類如下：

| 元件 | 範圍 | 實作狀態 | 說明 |
|---|---|---|---|
| **P1 Web Infrastructure** | Web Host、Session、API、SSE、安全機制 | **IMPLEMENTED** | 僅限 Localhost（127.0.0.1）、一次性啟動 Token、HttpOnly Session Cookie、CSRF / Origin 防護 |
| **P2 Global Web Console** | Dashboard、Tasks、Agents、Memory、Pending Review、Audit | **IMPLEMENTED** | 使用原生 HTML/CSS/JS 前端，透過 `go:embed` 嵌入；統一呈現 Tasks 與 Memory Candidates 的待審核項目 |
| **P3 Investment Core** | Portfolio Profile / Snapshot、股數計算、資金限制、成本計算、交易提案 | **IMPLEMENTED** | Go 計算責任邊界、動態 `CostConfig`、交易提案駁回紀錄 |
| **P6 Agent Web UI** | Agent 專屬 UI 與 Namespace Mux | **IMPLEMENTED** | Agent 路由隔離（`/api/v1/agents/{id}/`）、投資組合、行情資料與校準評分卡 UI |
| **P4 Data Ingestion** | TWSE、TPEx、MOPS、財經新聞擷取 | **IMPLEMENTED** | TWSE OpenAPI 每日行情資料擷取、MOPS 官方公告解析器、速率限制與快取 |
| **P5 Prediction & Eval** | 預測校準、回測、自我改進 | **IMPLEMENTED** | Brier Score 計算、依市場實際結果進行經驗性結算、方向準確率追蹤 |
| **SCHED Automation** | Cron 排程與 Misfire Policies | **IMPLEMENTED** | 標準五欄位 Cron 解析器與別名、Misfire Policies（SKIP、RUN_ONCE、CATCH_UP）、自動 Task 派發 |
| **CLI-INV Top-Level CLI** | `aia investment` CLI 指令組 | **IMPLEMENTED** | `portfolio`、`sync`、`proposals`、`approve`、`reject`、`predictions`、`evaluate` 子指令 |
| **CLI-SCHED Top-Level CLI** | `aia schedule` CLI 指令組 | **IMPLEMENTED** | `list`、`create`、`trigger`、`tick` 子指令 |
