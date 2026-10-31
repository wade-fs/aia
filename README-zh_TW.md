# AIA — 自主智慧架構

**個人 AI Agent 平台（Personal AI Agent Platform）**

AIA（Autonomous Intelligence Architecture）是一個個人 AI Agent 平台，旨在協調不同領域的專業 Agent、管理長期任務、維護持久化記憶，並將 AI 能力整合至實際工作流程中。

AIA 不將 AI Agent 視為彼此孤立的聊天機器人，而是提供一個共同的 Runtime 環境，讓多個專業 Agent 能夠在統一架構下運作、共享受管理的上下文、執行任務，並保留可追溯的活動紀錄。

本專案目前以 Go 實作，並採用 [Antigravity CLI](https://antigravity.google/) 作為底層 AI Agent 執行框架（Agent Harness）。

---

## 1. 專案願景

AIA 的目標是建立一個個人 AI 作業環境，透過不同的專業領域 Agent，支援使用者在各種領域的工作。

平台設計建立在以下四項原則之上：

- **個人化（Personal）：** 由單一擁有者控制的個人系統。
- **持續性（Persistent）：** 任務、記憶、專案及執行紀錄能跨越個別對話持續保存。
- **模組化（Modular）：** 由不同專業 Agent 處理各自領域，同時共用統一的 Runtime。
- **治理機制（Governed）：** 透過政策、權限、核准流程及稽核紀錄，確保 Agent 的活動受到控制。

長期目標是建立一套 AI 系統，持續支援使用者的工作、研究、學習及決策，不必每次都由使用者手動啟動及管理所有操作。

AIA 並非設計成不受限制的自主系統。所有能力與權限都必須受到平台及其擁有者明確管理。

## 2. 整體架構

AIA 將平台層級的協調管理，與底層 AI 執行環境分離。

```text
┌──────────────────────────────────────────────────────┐
│                         AIA                          │
│                                                      │
│  Personal Agent                                      │
│       │                                              │
│       ├── Domain Agents                              │
│       │     ├── Investment Agent                     │
│       │     ├── Coding Agent                         │
│       │     └── Writing Agent                        │
│       │                                              │
│       ├── Task Management                            │
│       ├── Project Management                         │
│       ├── Memory Management                           │
│       ├── Policy & Approval                           │
│       ├── Scheduling                                 │
│       ├── Data Collection                            │
│       ├── Recovery & Audit                           │
│       └── Web Interface                              │
│                                                      │
├──────────────────────────────────────────────────────┤
│                 AI Agent Harness                     │
│                                                      │
│                    Antigravity CLI                   │
│                                                      │
│          Model Execution / Tools / Sandbox           │
└──────────────────────────────────────────────────────┘
```

### 2.1 AIA Runtime（執行環境）

AIA 負責平台層級的協調與管理：

- Agent 註冊與生命週期管理。
- 任務建立、執行協調及復原。
- 專案與持久化記憶管理。
- 排程與自動化。
- 政策執行與核准流程。
- 執行紀錄與稽核軌跡。
- Web 介面與使用者互動。

### 2.2 AI Agent Harness（Agent 執行框架）

AIA 使用 Antigravity CLI 作為底層 Agent 執行環境。

Harness 負責 AI 模型互動，以及 Agent 可使用的工具與執行權限。

AIA 不打算重新實作底層 AI Agent 執行框架，而是管理更高層級的 Agent 生命週期與協調作業。

使用者透過 Harness 自行完成 Google OAuth 認證。AIA 不讀取、複製或解析 OAuth Token。

### 2.3 Domain Agents（領域專業 Agent）

領域專業 Agent 提供特定領域的能力，同時運作於 AIA 共用的 Runtime 之中。

初期規劃的 Agent 包括：

| Agent | 職責 |
|---|---|
| Personal Agent | 主要使用者介面與任務協調 |
| Investment Agent | 財經資料研究、分析及投資提案 |
| Coding Agent | 軟體開發與程式碼相關工作 |
| Writing Agent | 寫作、研究及文件相關工作 |

Domain Agent 採用靜態註冊方式，不屬於目前架構的部分則包括 Go Plugin 的執行期間動態載入。

## 3. 核心功能

以下說明目前實作所涵蓋的功能。各項功能的完成程度可能不同，不應僅因為存在某個介面或命令，就認定該功能已完整實現。

### 3.1 Agent 管理

AIA 提供專業 Agent 的註冊與管理架構。

平台可依據任務需求選擇適當的 Agent，並透過協調機制呼叫執行。

### 3.2 任務管理

AIA 提供任務生命週期管理，包括：

- 任務建立與派發。
- 任務狀態追蹤。
- 待執行任務處理。
- 執行協調。
- 中斷任務復原。

平台明確區分「建立任務」與「實際執行任務」。背景執行仍須整合並啟動相應的 Runtime Worker。

### 3.3 持久化記憶

AIA 提供持久化儲存機制，保存跨任務與跨工作階段所需的資訊。

記憶由平台管理，而非僅存在於單次 AI 對話的上下文中。

### 3.4 排程管理

AIA 包含排程服務，可建立及觸發預定時間執行的任務。

目前實作已提供排程管理，以及透過 CLI 手動處理排程的能力。

持續性的排程輪詢及自動任務執行，仍須進一步整合至 Runtime。

### 3.5 資料蒐集

AIA 已建立資料蒐集框架，包含 Provider 介面及資料來源管理機制。

目前 Collector 架構包含以下 Provider：

- 臺灣證券交易所（TWSE）
- 證券櫃檯買賣中心（TPEx）
- 美國聯邦準備經濟資料庫（FRED）
- 美國證券交易委員會（SEC）

然而，Provider 的存在不代表它已經能夠取得經過驗證的即時資料。部分現有 Provider 實作仍包含預設或硬編碼資料。

在這些實作完成修正與驗證之前，其輸出資料不能視為可靠的研究資料。

### 3.6 投資研究

Investment Agent 的設計目標是提供研究與決策支援，而非自動交易。

目前應用程式已具備以下能力：

- 投資組合查詢。
- 市場資料同步。
- 公告資料同步。
- 投資提案查詢。
- 投資提案核准與駁回。
- 預測資料查詢。
- 預測結果評估。

研究流程仍在開發中。在系統能被視為自主投資研究服務之前，仍須完成資料品質管理、來源驗證、自動化蒐集及排程研究執行等工作。

### 3.7 Web 介面

AIA 提供本機 Web Host，並採用模組化介面設計，支援平台及領域專業 Agent 的功能。

Web Host 包含應用程式控制台，以及已註冊的 Agent 模組。

預設部署僅限本機使用，不應直接暴露於區域網路或公開網際網路。

## 4. 技術架構

| 元件 | 使用技術 |
|---|---|
| 主要程式語言 | Go |
| AI Agent 執行框架 | Antigravity CLI |
| 持久化儲存 | SQLite |
| Web Server | Go HTTP Server |
| Agent 架構 | 靜態註冊的領域專業 Agent |
| 設定與命令管理 | CLI |
| 服務部署 | systemd |

整體架構強調平台 Runtime 與 AI 執行環境之間的明確分工。

## 5. 專案結構

目前的程式實作位於 `go/` 目錄。

```text
aia/
├── README.md
├── docs/
│   └── ...
│
└── go/
    ├── cmd/
    ├── internal/
    │   ├── app/
    │   ├── cli/
    │   ├── collector/
    │   ├── domain/
    │   ├── orchestrator/
    │   ├── repository/
    │   ├── service/
    │   └── web/
    │
    ├── scripts/
    ├── Makefile
    └── ...
```

以上為概念性目錄結構，完整套件組織請以實際原始碼目錄為準。

## 6. 安裝與啟動

AIA 目前以 Linux 環境為主要開發及部署平台。

Go 實作及安裝腳本位於 `go/` 目錄。

專案包含 Makefile 與服務管理腳本，可用於安裝及整合 systemd。

安裝完成後，主要伺服器啟動命令為：

```bash
aia server start
```

此命令會啟動 AIA Web Server，並持續執行，直到收到終止訊號。

伺服器支援正常關閉程序（Graceful Shutdown）。

如需查詢 CLI 使用方式：

```bash
aia --help
```

查詢各命令的可用功能：

```bash
aia agent --help
aia chat --help
aia investment --help
aia schedule --help
```

實際部署程序及設定選項，請參閱安裝腳本與操作文件。

## 7. Investment Agent CLI

Investment Agent 目前提供以下命令：

```bash
aia investment portfolio
aia investment sync <symbol>
aia investment proposals
aia investment approve <proposal-id>
aia investment reject <proposal-id>
aia investment predictions
aia investment evaluate
```

這些命令提供投資組合查詢、資料同步、投資提案管理及預測結果評估等手動操作能力。

這些命令的存在，不代表相應操作已經能夠在背景自動執行。

## 8. 排程 CLI

AIA 提供以下排程命令：

```bash
aia schedule list
aia schedule create
aia schedule trigger
aia schedule tick
```

排程服務能夠處理已到期的排程，並據此建立任務。

若要形成完整的自動執行流程，仍須讓 Runtime 持續處理到期排程，並進一步執行已準備就緒的任務。

## 9. 安全與治理

AIA 的設計目標是建立一個受到控制的個人 AI 平台。

初期版本採用以下限制。

### 單一使用者（Single User）

AIA 專為單一擁有者設計。

多使用者認證、角色權限管理（RBAC）及多租戶部署，不在初期版本的範圍內。

### 僅限本機（Local Only）

Web Host 預設綁定 `127.0.0.1`。

預設部署模式不支援直接暴露於區域網路或公開網際網路。

### Agent 權限控管

AIA 不會繞過底層 Harness 的權限管理機制。

系統不得使用 `--dangerously-skip-permissions`。

需要使用者授權的操作，必須遵循適用的核准及權限管理機制。

### 認證管理

使用者透過 Antigravity CLI 完成認證。

AIA 不直接管理或操作 Google OAuth 憑證。

當認證失效或需要使用者介入時，受影響的任務必須進入等待認證狀態，直到使用者完成重新認證。

### 財務安全

AIA 是一個研究與決策支援平台。

初期版本不會自動執行金融交易，也不會持有使用者資金。

所有投資提案仍須經過使用者明確核准。

### 稽核能力（Auditability）

任務執行、核准流程及相關系統活動，都應保留可追溯的持久化紀錄。

## 10. 目前開發狀態

AIA 已經超越初期架構原型階段。目前的 Go 實作已包含多項核心服務、CLI 命令、資料蒐集抽象架構、排程機制及領域專業 Agent 介面。

然而，仍有多項重要能力需要進一步整合與驗證。

| 項目 | 目前狀態 |
|---|---|
| Go 應用程式 Runtime | 已實作 |
| Agent 註冊機制 | 已實作 |
| Personal Agent | 已實作 |
| 任務管理 | 已實作 |
| 持久化儲存 | 已實作 |
| 復原機制 | 已實作 |
| Web Host 與 Agent 模組 | 已實作 |
| Investment CLI | 已實作 |
| Investment Service 框架 | 已實作 |
| Data Collector 框架 | 已實作 |
| 外部資料 Provider | 部分實作；仍須驗證真實資料取得能力 |
| Schedule Service | 已實作 |
| 持續性排程 Worker | 尚未整合至 Server Runtime |
| 持續性任務 Worker | 尚未整合至 Server Runtime |
| 自動化資料蒐集 | 尚未整合至 Server Runtime |
| 自動化投資研究 | 尚未整合至 Server Runtime |
| 自動化預測評估 | 尚未整合至 Server Runtime |

本表是目前實作狀態的基準，而非最終驗收報告。每項能力都必須依據實際執行流程及資料品質進行驗證。

## 11. Runtime 與自動化開發路線

AIA 的設計目標是成為一個長時間運作的個人 AI 服務。

預期的 Runtime 模型，將持續運作的平台服務與按需求啟動的 AI 執行作業分開。

AIA 主程序持續運作，而個別 Agent 只在需要執行工作時才啟動。

規劃中的 Runtime 職責包括：

1. **Runtime Supervisor（執行環境監督器）** — 管理 Worker 生命週期、取消作業及正常關閉程序。
2. **Schedule Worker（排程工作程序）** — 持續檢查是否有已到期的排程。
3. **Task Worker（任務工作程序）** — 領取並執行已準備就緒的任務。
4. **Data Collection Worker（資料蒐集工作程序）** — 取得並驗證外部資訊。
5. **Research Worker（研究工作程序）** — 建立研究快照，並在必要時呼叫領域專業 Agent。
6. **Prediction Evaluation Worker（預測評估工作程序）** — 當有足夠的新資料時，評估先前的預測結果。

預期的投資研究流程如下：

```text
外部資料來源
      │
      ▼
資料蒐集
      │
      ▼
資料驗證與正規化
      │
      ▼
研究快照
      │
      ▼
Investment Agent
      │
      ▼
研究與分析
      │
      ▼
投資提案
      │
      ▼
人工審查與核准
```

AIA 的目標不是讓 AI 模型持續不間斷地執行。

例行性的資料蒐集、排程、資料驗證及預測評估，應由 Go Service 負責。只有在需要深入分析或決策時，才呼叫 AI Agent。

## 12. 開發原則

本專案遵循以下開發原則：

- 將平台協調管理與 AI 模型執行分離。
- 優先採用明確的介面與模組化服務。
- 將領域專屬邏輯放在領域專業 Agent 或領域服務中。
- 持久化保存重要的任務狀態與執行狀態。
- 設計可從中斷執行中復原的機制。
- 外部資料必須經過驗證，才能用於研究。
- 確保自動化流程具備可觀測性及稽核能力。
- 重要的金融操作必須經過人工核准。
- 避免不必要地耦合至特定程式語言或 AI 模型供應商。

雖然目前以 Go 實作，但整體架構應保持足夠的語言中立性，以便未來有需要時，能夠使用其他程式語言重新實作。

## 13. 專案範圍與非目標

AIA 初期版本不以提供以下能力為目標：

- 多使用者 SaaS 平台。
- 對外公開的 Agent 託管服務。
- 不受限制的自主執行能力。
- 自動化金融交易。
- 直接保管或管理使用者資金。
- 執行期間動態載入任意 Go Plugin。
- 取代底層 AI Agent 執行框架。

本專案的核心目標，是建立一個可靠、可擴充，並由擁有者掌控的單一使用者 AI 平台。

## 14. 專案現況

AIA 目前仍處於積極開發階段。

現階段的優先工作，是建立能夠長時間可靠運作的基礎：

- 驗證外部資料 Provider 的正確性。
- 防止模擬資料或預設資料進入正式研究流程。
- 將 Background Worker 整合至應用程式 Runtime。
- 串接排程管理與實際任務執行。
- 建立可靠的自動化研究與評估流程。
- 改善 Runtime 可觀測性及操作文件。

專案將採取漸進式開發，以正確性、可復原性及使用者控制權為優先，而非過早追求自主化。

---

**AIA — 自主智慧架構（Autonomous Intelligence Architecture）**

*打造一個能夠協調、記憶、推理與行動，並始終受到人類治理的個人 AI 平台。*
