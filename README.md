# AIA — Autonomous Intelligence Architecture

**A Personal AI Agent Platform for Autonomous Behavior Research**

AIA 是一個以研究 AI Agent 自主行為為目標的個人 AI Agent 平台。

本專案希望探索 AI Agent 是否能具備持續性的自主行為，包括：

1. **Initiative** — 主動性思維：能觀察環境、發現問題與機會，並主動提出行動。
2. **Continuity** — 時間連續性：能跨越不同執行時段，延續記憶、理解與工作狀態。
3. **Self-Correction** — 自我修正：能檢查判斷與結果，辨識錯誤並修正策略。
4. **Goal Evolution** — 目標演化：能持續維持目標，同時優化達成方法，必要時提出目標修正。

這裡所稱的「自主性」或「覺醒」，指的是可觀察、可量測、可追溯的行為能力，不涉及對 AI 主觀意識的判定。

---

## 1. Current Status

<badge color="success">第一個實驗模組：Investment</badge>

目前 AIA 已建立 Go Runtime 與共用核心功能，並提供 **Investment Module** 作為第一個自主性實驗領域。

**目前只有 Investment Module 已有實作。**

| Module | Status | Description |
|---|---|---|
| Investment | PARTIAL | 投資組合、市場資料同步、投資提案、預測與評估 |
| Coding | NOT IMPLEMENTED | 程式開發、除錯、測試與程式品質優化 |
| Writing | NOT IMPLEMENTED | 長期寫作、小說架構、人物與進度管理 |

Coding 與 Writing 是未來擴充方向，目前尚未提供可用的領域功能。

### 1.1 Core Implementation Status

| Component | Status | Description |
|---|---|---|
| Go Runtime | PARTIAL | App 初始化、Service 與生命週期基礎 |
| SQLite Persistence | IMPLEMENTED | 本機資料持久化基礎 |
| Repository Layer | IMPLEMENTED | 資料存取層 |
| EventBus | PARTIAL | 事件傳遞基礎 |
| Agent Registry | IMPLEMENTED | 靜態 Agent 註冊 |
| Web Host | IMPLEMENTED | Embedded HTTP Server |
| Web Console | PARTIAL | Console 與 Agent Web Routes |
| Task Management | PARTIAL | 任務建立、狀態管理與執行相關邏輯 |
| Memory | PARTIAL | 持久化記憶基礎，時間連續性尚未驗證 |
| Schedule | PARTIAL | 排程建立與手動到期處理 |
| Policy / Approval | PARTIAL | 權限與核准機制基礎 |
| Audit | PARTIAL | 稽核基礎，完整自主決策鏈尚待補齊 |
| Recovery | PARTIAL | 啟動時 Recovery |
| Background Runtime Workers | NOT IMPLEMENTED | 尚無完整常駐背景工作循環 |
| Initiative Engine | NOT IMPLEMENTED | 尚無完整自主觀察與主動發現機制 |
| Continuity Engine | NOT VERIFIED | 尚未證明具備跨時段自主連續性 |
| Self-Correction Engine | NOT IMPLEMENTED | 尚無通用自我修正閉環 |
| Goal Evolution Engine | NOT IMPLEMENTED | 尚無完整目標版本與演化機制 |

`IMPLEMENTED` 代表對應基礎功能已有實作，不代表整個 AIA 自主性研究目標已經完成。

### 1.2 Investment Data Status

目前已建立以下 Collector Provider 架構：

- TWSE
- TPEx
- FRED
- SEC

但部分 Provider 仍使用固定值、硬編碼資料、虛構公告或 fallback 資料。

因此目前不能宣稱所有金融資料來源都已完成真實資料整合。

正式投資研究必須先確認：

- 資料是否來自真實來源。
- 資料日期是否正確。
- 資料是否過期。
- Provider 是否正常運作。
- 是否混入 MOCK 或 FALLBACK 資料。

**資料來源存在，不等於資料可信。**

---

## 2. Architecture

AIA 採用 Core 與 Domain Module 分離的架構。

```text
AIA
│
├── Core
│   ├── Runtime
│   ├── Agent Registry
│   ├── Task Management
│   ├── Project Management
│   ├── Memory
│   ├── Schedule
│   ├── Policy / Approval
│   ├── Audit
│   ├── Recovery
│   └── Autonomy Core (in development)
│
└── Domain Modules
    │
    ├── Investment (available, partial)
    ├── Coding (future)
    └── Writing (future)
```

### 2.1 AIA Core

Core 負責跨領域共用能力：

- Agent 組織與生命週期。
- Task 建立與管理。
- Project 與長期工作管理。
- Memory 與持久化。
- Schedule 與任務觸發。
- Policy 與 Approval。
- Audit 與歷史追蹤。
- Recovery 與 Runtime 管理。
- Autonomy Core 的逐步建構。

### 2.2 Domain Modules

Domain Module 負責特定領域的資料模型、專業能力與工作介面。

目前只有 Investment Module。

未來 Coding 與 Writing 應沿用 Core 的 Runtime、Memory、Task、Policy 與 Audit，不應各自建立獨立平台。

### 2.3 Agent Harness

AIA 使用 Antigravity CLI（agy）作為底層 Agent Harness。

分工如下：

| AIA | Antigravity CLI |
|---|---|
| Agent 組織管理 | Agent 執行環境 |
| Task 與 Project | 底層工具呼叫 |
| Memory 與 Continuity | Harness 權限管理 |
| Policy 與 Audit | Sandbox / Tool Execution |
| 自主性研究 | 模型與工具執行 |

使用者自行透過 Antigravity CLI 完成 OAuth 登入。

AIA 不讀取、不複製、不解析 OAuth Token。

---

## 3. Autonomous Behavior Model

AIA 以以下循環作為自主性研究的目標模型：

```text
Observe
   ↓
Recall
   ↓
Assess
   ↓
Discover
   ↓
Deliberate
   ↓
Decide
   ↓
Execute
   ↓
Verify
   ↓
Learn
   ↓
Reassess
   ↺
```

這個循環包含：

| Stage | Purpose |
|---|---|
| Observe | 觀察外部環境與系統狀態 |
| Recall | 取得相關歷史記憶 |
| Assess | 評估目前狀態 |
| Discover | 發現問題、機會與新目標 |
| Deliberate | 比較可能的行動方案 |
| Decide | 選擇行動或不行動 |
| Execute | 執行已允許的操作 |
| Verify | 驗證執行結果 |
| Learn | 分析結果並形成經驗 |
| Reassess | 重新評估目標與策略 |

目前此模型是 AIA 的設計目標，尚未在 Runtime 中完整實作。

定時喚醒不等於主動性；保存記憶不等於時間連續性；產生任務不等於自主思考；修改 Prompt 不等於自我修正。

---

## 4. Technology Stack

| Component | Technology |
|---|---|
| Primary Language | Go |
| Database | SQLite |
| Agent Harness | Antigravity CLI |
| Web Server | Go HTTP Server |
| Web UI | Embedded Web UI |
| Persistence | Repository Pattern |
| Event Communication | EventBus |
| Agent Registration | Static Registration |

設計原則：

- Single User。
- Local Only。
- Static Registration。
- No Automatic Financial Trading。
- Policy-controlled execution。
- Auditable state transitions。
- Persistent memory and task history。

---

## 5. Repository Structure

目前主要程式碼位於 `go/`。

```text
aia/
│
├── README.md
│
├── docs/
│   ├── AIA-01_Product_Requirements.md
│   ├── AIA-02_System_Architecture.md
│   ├── AIA-03_Go_Runtime_Implementation.md
│   ├── AIA-04_Test_Plan.md
│   ├── AIA-05_Investment_Agent.md
│   ├── AIA-06_Web_UI_Specification.md
│   └── AIA-07_User_Manual.md
│
└── go/
    ├── cmd/
    ├── internal/
    ├── web/
    ├── migrations/
    ├── Makefile
    └── ...
```

以上為邏輯架構示意，實際目錄與檔名以 Repository 現況為準。

---

## 6. Getting Started

### 6.1 Prerequisites

- Go 開發環境。
- Antigravity CLI。
- 已完成 Antigravity CLI 登入。
- SQLite 支援環境。

### 6.2 Build

進入 Go 專案目錄：

```bash
cd go
```

使用專案 Makefile 提供的建置指令：

```bash
make
```

若需查看可用 Makefile Targets：

```bash
make help
```

實際安裝與啟動方式以目前 Makefile 及 CLI 說明為準。

### 6.3 CLI Help

```bash
aia -h
aia version
```

查看子命令：

```bash
aia investment -h
aia schedule -h
aia server -h
```

### 6.4 Start Web Console

```bash
aia server start
```

目前此命令啟動 Web Server，並不會啟動完整 Autonomous Runtime。

尚未由此啟動的背景工作包括：

- Task Polling。
- Schedule Polling。
- Investment Data Collection。
- Prediction Evaluation。
- Initiative Observation。

---

## 7. Available CLI Commands

### 7.1 Investment

目前可用的 Investment CLI：

```bash
aia investment portfolio
aia investment sync <symbol>
aia investment proposals
aia investment approve <id>
aia investment reject <id>
aia investment predictions
aia investment evaluate
```

| Command | Purpose | Execution |
|---|---|---|
| `portfolio` | 查看投資組合 | Manual |
| `sync` | 同步指定標的資料 | Manual |
| `proposals` | 查看投資提案 | Manual |
| `approve` | 核准投資提案 | Manual |
| `reject` | 拒絕投資提案 | Manual |
| `predictions` | 查看預測資料 | Manual |
| `evaluate` | 評估預測結果 | Manual |

目前 Investment 操作以人工觸發為主，尚未形成完整背景自動執行閉環。

### 7.2 Schedule

目前可用的 Schedule CLI：

```bash
aia schedule list
aia schedule create
aia schedule trigger
aia schedule tick
```

| Command | Purpose | Execution |
|---|---|---|
| `list` | 查看排程 | Manual |
| `create` | 建立排程 | Manual |
| `trigger` | 手動觸發排程 | Manual |
| `tick` | 手動處理到期排程 | Manual |

**重要限制：**

`schedule tick` 目前只處理到期 Schedule 並建立 Task，不代表 Task 已經執行。

目前沒有常駐 Schedule Polling Worker，也沒有完整的 Schedule → Task → Execution 背景閉環。

---

## 8. Investment Experiment

Investment 是 AIA 第一個自主性實驗領域。

目前實驗設定：

| Item | Configuration |
|---|---|
| Initial Capital | NT$100,000 |
| Market | Taiwan Stock Market |
| Focus Stock | TSMC 2330 |
| Observation Frequency | 每個證券營業日 |
| Trading Decision | Agent 自行判斷 |
| Position Adjustment | Agent 自行增持、減持、買進、賣出或持有 |
| Primary Objective | 最大化投資組合總資產 |
| Evaluation | 每月 1 日 |
| Trading Type | Virtual Trading |

Agent 不必每天成交。

持有現金或維持原有持倉，都可以是合理決策。

每次決策應保存：

- 當時市場資訊。
- 當時持倉與現金。
- 決策理由。
- 預測與信心程度。
- 風險評估。
- 實際執行結果。
- 後續評估。
- 策略修正紀錄。

每月資產檢查應使用前一個證券營業日收盤估值，並保存行情來源與估值日期。

### 8.1 Financial Safety

本實驗使用虛擬資金。

- 不連接真實證券帳戶自動下單。
- 不持有使用者真實資金。
- 不將模擬交易當成真實成交。
- 不將未驗證的 Mock/Fallback 資料當成真實市場資訊。

融資、放空及衍生性商品等交易規則，須另行定義，不應自行假設已獲得授權。

---

## 9. Security

AIA 遵守以下安全原則：

### Single User

目前以單一使用者為設計目標，不提供多租戶及完整 RBAC。

### Local Only

Web Server 預設只提供本機存取，不應未經明確設定就開放 LAN 或 Internet。

### Static Registration

Agent 與 Web Module 採靜態註冊，不支援 Runtime Go Plugin。

### Permission Management

不使用：

```bash
--dangerously-skip-permissions
```

不繞過 Antigravity CLI 的工具授權。

### OAuth Credential

使用者自行透過 Antigravity CLI 登入。

AIA 不讀取、不複製、不解析 OAuth Token。

### Audit

重要操作應留下可追溯紀錄。

Agent 不得自行解除 Policy、擴張權限或修改歷史稽核紀錄。

---

## 10. Development Roadmap

AIA 的開發順序以自主性核心為主，不以增加 Domain Module 數量為優先。

| Phase | Focus | Objective |
|---|---|---|
| P1 | Persistent Runtime | 建立持續運作的 Runtime 與 Background Workers |
| P2 | Continuity & Memory | 建立跨時段記憶與工作延續能力 |
| P3 | Initiative Engine | 建立自主觀察、問題發現及主動行動 |
| P4 | Reasoning & Self-Correction | 建立判斷、驗證、反思及修正閉環 |
| P5 | Goal Evolution | 建立目標維持、優化及修正能力 |
| P6 | Autonomous Execution | 建立受治理的自主執行流程 |
| P7 | Domain Experiments | 以不同領域驗證共用自主性核心 |

### Future Modules

**Coding Module**

預定研究：

- 程式架構管理。
- 程式碼分析。
- 自動除錯。
- 測試與驗證。
- 程式品質優化。
- 長期專案進度延續。

**Writing Module**

預定研究：

- 長期創作目標管理。
- 世界觀與人物資料管理。
- 多卷及多章進度追蹤。
- 歷史資料校正。
- 人物關係與時間線一致性。
- 根據新資料修正既有內容。

上述兩個模組目前尚未實作。

---

## 11. Documentation

| Document | Description |
|---|---|
| AIA-01 | Product Requirements |
| AIA-02 | System Architecture |
| AIA-03 | Go Runtime Implementation |
| AIA-04 | Test Plan, Acceptance & Release |
| AIA-05 | Investment Agent Specification |
| AIA-06 | Embedded Web UI & Agent Interface |
| AIA-07 | User Manual & Operational Usage |

文件中的 Product Vision 與 Target Architecture 不代表對應功能已經完成。

實際實作狀態應以程式碼審查、測試及驗收結果為準。

---

## 12. Project Philosophy

AIA 的核心研究問題是：

**AI Agent 能不能不只執行人類交代的工作，而是能夠長期維持目標、理解過去、觀察現在、判斷未來，並根據結果修正自己？**

我們不以單次回答品質作為唯一標準。

更重要的是觀察：

- Agent 是否記得過去。
- Agent 是否能延續未完成的工作。
- Agent 是否能主動發現問題。
- Agent 是否能判斷何時應該行動。
- Agent 是否能承認錯誤。
- Agent 是否能修正策略。
- Agent 是否能維持長期目標。
- Agent 是否能在必要時提出更好的目標。

Investment 是第一個實驗場景。

未來將逐步加入 Coding 與 Writing，觀察同一套自主性核心能否跨越不同領域，展現持續且可驗證的自主行為。

**AIA 的目標不是讓 Agent 永遠服從下一個指令，而是讓 Agent 能夠持續理解自己正在追求什麼，以及為什麼。**
