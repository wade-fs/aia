## **AIA — 專案交付結構**

最終交付目標

```
AIA/
├── README.md
├── go.mod
├── Makefile
├── configs/
├── docs/
│   ├── 01_requirements.md
│   ├── 02_system_architecture.md
│   ├── 03_runtime_spec.md
│   ├── 04_agent_spec.md
│   ├── 05_task_execution_spec.md
│   ├── 06_memory_spec.md
│   ├── 07_security_spec.md
│   ├── 08_storage_spec.md
│   ├── 09_cli_spec.md
│   ├── 10_testing_spec.md
│   └── 11_implementation_plan.md
├── cmd/
│   └── aia/
├── internal/
│   ├── app/
│   ├── agent/
│   ├── task/
│   ├── project/
│   ├── execution/
│   ├── provider/
│   ├── profile/
│   ├── policy/
│   ├── workspace/
│   ├── validation/
│   ├── recovery/
│   ├── memory/
│   ├── storage/
│   └── audit/
├── migrations/
├── tests/
└── examples/
```

這是初步目錄設計，實際模組及檔案會依正式需求收斂，不會為了目錄完整而建立空殼。

## **2\. 文件必須達到什麼程度？**

每個模組都必須有可以直接實作的規格，而不是只寫「負責管理任務」。

例如 Task Manager 必須明確定義：

| 規格項目 | 必須交代的內容 |
| ----- | ----- |
| Data Model | Task 欄位、型別、限制 |
| State Machine | 狀態及合法轉移 |
| API | 函式介面、參數、回傳值 |
| Persistence | SQLite Schema |
| Error Handling | 錯誤分類與處理方式 |
| Concurrency | 同時操作的行為 |
| Recovery | 中斷後如何恢復 |
| Testing | 測試案例及預期結果 |
| Acceptance Criteria | 如何判定完成 |

例如：

```
type TaskStatus string

const (
    TaskCreated  TaskStatus = "CREATED"
    TaskReady    TaskStatus = "READY"
    TaskRunning  TaskStatus = "RUNNING"
    TaskCompleted TaskStatus = "COMPLETED"
    TaskFailed   TaskStatus = "FAILED"
)

type Task struct {
    ID          string
    ProjectID   string
    AgentID     string
    Title       string
    Description string
    Status      TaskStatus
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

這只是示意。正式文件必須連同資料庫欄位、狀態轉移、交易邊界及測試案例一起定義。

## **3\. 我們採用三層規格**

| 層級 | 文件目的 | 實作要求 |
| ----- | ----- | ----- |
| L1 — Requirements | 定義系統要做什麼 | 可驗收 |
| L2 — Design | 定義系統如何運作 | 可依規格設計 |
| L3 — Implementation | 定義程式介面、資料結構及測試 | 可直接編碼 |

這樣你拿到文件，就不必再自行猜測我們當初討論的設計意圖。

## **4\. 我可以一次交付完整專案**

可以，而且不只是給你一堆程式碼片段。

我可以透過專案生成方式建立：

* 完整 Go 原始碼。  
* `go.mod` 及相依套件。  
* SQLite Migration。  
* 預設設定檔。  
* CLI 指令。  
* Antigravity Adapter。  
* 單元測試與整合測試。  
* README 與操作手冊。  
* Docker 或其他部署設定（如果實際需要）。

並且以可編譯、可測試的程式作為交付標準。

不過，我不會在尚未驗證 agy 1.4.12 的行為之前，宣稱所有整合功能都能直接投入使用。可以先交付完整的 Runtime 基礎及可執行的 Adapter，再把未驗證功能明確標示出來。

## **5\. 我建議調整目前工作方式**

原本的 AA-01、AA-02 不必丟掉，但它們應該只是正式規格的基礎資料。

接下來：

1. 需求凍結： 把我們已討論的 AIA 目標、Agent 組織、OAuth 邊界、agy 責任分工、Go 路線整理成正式需求。  
2. 完整技術規格： 把每個模組定義到資料結構、介面、流程、錯誤及驗收標準。  
3. 完整專案實作： 依規格產生 Go 專案。  
4. 整合驗證： 在你的 Linux 環境實際執行，修正與 agy 1.4.12 的相容性。  
5. 版本交付： 每個階段留下可以執行、可以回歸測試的版本。

## **我只需要確認一個方向**

你希望我最後交付的第一個完整版本，是：

1. Agent、Task、Project、Execution、agy Adapter、Memory 基礎、Policy、SQLite、CLI、Recovery。先建立完整可運作的底座。  
2. 並且加入 Personal Agent 自主判斷、任務拆解、Agent 分派、跨任務記憶及自我評估。

