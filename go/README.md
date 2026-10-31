# Go Agent Runtime PoC (AA-04)

> 最小可執行 Agent Runtime，透過 Worker Adapter 介面管理外部 AI Coding Worker（Aider），驗證 Runtime 具備建立任務、環境隔離、執行監控、獨立驗收、崩潰恢復與審計追蹤之能力。

---

## 專案架構

本專案採用 **Modular Monolith** 架構，模組職責清晰，具備嚴格的單向相依原則（Storage 與 Audit 獨立於其他模組，無 Import Cycle）。

```
go/
├── cmd/
│   └── agent/
│       └── main.go              # CLI 入口 (Cobra)
├── internal/
│   ├── runtime/                 # Runtime 核心協調器
│   │   └── runtime.go
│   ├── task/                    # 任務管理模組
│   │   ├── manager.go
│   │   └── model.go
│   ├── worker/                  # Worker Adapter 抽象層與管理
│   │   ├── interface.go
│   │   ├── manager.go
│   │   └── aider/
│   │       └── adapter.go       # Aider CLI 介面實作
│   ├── workspace/               # 工作空間隔離與 Git 初始化
│   │   └── manager.go
│   ├── state/                   # 狀態機與狀態轉移驗證
│   │   └── manager.go
│   ├── recovery/                # 崩潰修復與狀態 Reconciliation
│   │   └── manager.go
│   ├── validator/               # 獨立結果驗收器 (Git Diff + Tests)
│   │   └── validator.go
│   ├── git/                     # Git 操作客戶端
│   │   └── git.go
│   ├── storage/                 # SQLite 持久化儲存 (CGO / go-sqlite3)
│   │   └── sqlite.go
│   └── audit/                   # 審計日誌 (Stdout + SQLite)
│       └── logger.go
├── configs/
│   └── config.yaml              # 預設設定檔
├── testdata/
│   └── sample-project/          # 供 Worker 執行的範例 Go 專案
│       ├── go.mod
│       ├── main.go
│       └── main_test.go
├── docs/
│   └── poc-test-plan.md         # T01–T12 測試計畫書
├── go.mod
└── README.md
```

---

## 狀態機設計

```text
CREATED
   |
   v
READY
   |
   v
RUNNING
   |
   +------> SUCCEEDED
   |
   +------> FAILED
   |
   +------> CANCELLED
   |
   +------> UNKNOWN
                 |
                 v
             RECONCILING
                 |
        +--------+--------+
        |        |        |
        v        v        v
    SUCCEEDED  FAILED  NEEDS_REVIEW
```

- **先保存意圖，再啟動 Worker**：WorkerManager 在啟動 Aider 之前，必先將 execution 狀態以 `STARTING` 寫入資料庫。
- **UNKNOWN 不盲目重跑**：未確定之執行結果轉入 `RECONCILING` 進行環境檢查與結果獨立驗收，必要時轉入 `NEEDS_REVIEW` 等待人工介入。

---

## 編譯與安裝

### 環境需求
- Go 1.21 或更高版本
- GCC (供 `mattn/go-sqlite3` CGO 編譯使用)
- Git 2.0+
- Aider (選配，若未安裝可在 HealthCheck 時偵測)

### 編譯步驟

```bash
cd /home/wade/src/github/aia/go

# 下載相依套件並整理
go mod tidy

# 執行所有單元測試
go test -v ./...

# 編譯 CLI 工具
go build -o bin/agent ./cmd/agent/
```

---

## CLI 命令使用手冊

### 1. 檢視說明
```bash
./bin/agent --help
./bin/agent task --help
```

### 2. 檢查 Worker 健康狀態
```bash
./bin/agent worker health
```

### 3. 建立任務 (Task Create)
```bash
./bin/agent task create \
  --title "新增 Add 函式" \
  --description "在 sample-project 中新增 Add(a, b int) int 函式並新增單元測試" \
  --worker aider
```

### 4. 列出所有任務 (Task List)
```bash
./bin/agent task list
```
輸出格式：
```
ID                                     TITLE                    STATUS      WORKER   CREATED AT
--                                     -----                    ------      ------   ----------
c4963cbf-3548-4ced-bb5f-76dc95779c9b   新增 Add 函式            CREATED     aider    2026-10-31 09:44:49
```

### 5. 檢視任務細節與事件歷程 (Task Show)
```bash
./bin/agent task show <TASK_ID>
```

### 6. 執行任務 (Task Run)
透過 Workspace Manager 建立隔離目錄、記錄 Git Base Commit、啟動 Worker、收集 stdout/stderr，最後由 Result Validator 執行 `git diff` 與 `go test ./...` 驗收：
```bash
./bin/agent task run <TASK_ID>
```

### 7. 取消任務 (Task Cancel)
```bash
./bin/agent task cancel <TASK_ID>
```

### 8. 執行中斷恢復 (Recovery)
當 Runtime 意外中斷重啟時，自動掃描未完成工作並調解狀態：
```bash
./bin/agent recovery
```

---

## 測試計畫

完整 T01–T12 測試情境與驗證請參閱 [docs/poc-test-plan.md](docs/poc-test-plan.md)。
