# Agent Runtime PoC 測試計畫 (AA-04 規格)

本測試計畫依據 **AA-04：Go Agent Runtime — Aider PoC** 第 13 節之規格與驗收標準訂定，旨在全面驗證 Agent Runtime 核心功能與韌性。

---

## 1. 測試矩陣 (Test Matrix)

| 編號 | 測試情境 | 前置條件 | 測試步驟 | 預期結果 | 驗證方式 | 實作與驗收狀態 |
|---|---|---|---|---|---|---|
| **T01** | 正常建立 Task | Runtime DB 初始化完成 | 執行 `agent task create --title "..." --description "..." --worker aider` | 成功產生 UUID，狀態為 `CREATED`，寫入 SQLite `tasks` 表與 `task_events`，印出 Audit Log | `agent task list` 與 `agent task show <id>` 顯示正確資訊 | 已實作 / 通過 |
| **T02** | 正常啟動 Aider | Task 處於 `CREATED` 或 `READY` | 執行 `agent task run <id>` | 建立 Workspace，保存 execution intent（`STARTING`），啟動 Aider 取得 PID 並更新為 `RUNNING` | Audit Log 顯示 `worker_running` 與 PID，DB 記錄 PID | 已實作 / 通過 |
| **T03** | Aider 完成修改 | Task 正在 `RUNNING` 執行 | Aider 收到 message-file 指令並完成檔案修改 | Aider 行程正常結束，Runtime 收集 stdout/stderr 與 exit code 0 | `worker_executions` 表記錄 `COMPLETED` 及 exit_code 0 | 已實作 / 通過 |
| **T04** | 測試與驗收通過 | Aider 產生正確程式碼與測試 | Runtime Result Validator 執行 `git diff` 與 `go test ./...` | 驗收條件全部符合：WorkerExitOK=true, GitDiffExists=true, FilesInWorkspace=true, TestPassed=true；狀態轉為 `SUCCEEDED` | `agent task show <id>` 顯示 `Status: SUCCEEDED`，記錄 GitFinalCommit | 已實作 / 通過 |
| **T05** | Aider 執行失敗 | Aider 執行過程發生錯誤或模型呼叫失敗 | 觸發 Aider 返回非 0 exit code | Runtime 捕捉非 0 exit code，Validator 標記失敗，Task 狀態轉為 `FAILED` | `agent task show <id>` 顯示 `Status: FAILED` 及詳細錯誤原因 | 已實作 / 通過 |
| **T06** | Runtime 執行中關閉 | Worker 已啟動，Runtime process 被 SIGKILL 終止 | 終止 Runtime，重新啟動並執行 `agent recovery` | Recovery Manager 掃描未完成 Task，確認 Workspace 與狀態，不發生資料損壞或狀態遺失 | `agent recovery` 輸出修復與掃描日誌，狀態更新為正確恢復狀態 | 已實作 / 通過 |
| **T07** | Worker 尚在執行時 Runtime 重啟 | Aider process 仍在作業系統中運行 | 重啟 Runtime 執行 `agent recovery` | 檢查發現 PID 仍活躍（`Signal(0)` 成功），輸出 `worker still running (PID ...), monitoring`，不重複啟動新 process | Task 維持 `RUNNING`，不建立重複的 execution | 已實作 / 通過 |
| **T08** | Worker 已修改檔案但未回報完成 | Worker 意外中斷但 Workspace 已有修改 | 執行 `agent recovery` | Task 狀態轉移至 `RECONCILING`，獨立執行 Validator 驗收修改內容 | 若驗收通過則晉升為 `SUCCEEDED`，否則轉為 `UNKNOWN` / `FAILED` | 已實作 / 通過 |
| **T09** | Aider 不存在 | 系統未安裝 Aider 或 binary_path 錯誤 | 執行 `agent worker health` 或 `agent task run <id>` | HealthCheck 回傳 binary 不存在錯誤；RunTask 拒絕執行並記錄明確錯誤，避免崩潰 | 命令返回非 0 並列印具體缺失路徑錯誤訊息 | 已實作 / 通過 |
| **T10** | Workspace 不存在 | Task 記錄之 Workspace 目錄遭手動刪除 | 執行 `agent recovery` 或 `agent task run <id>` | Recovery 偵測 Workspace 目錄不存在，標記 Task 為 `NEEDS_REVIEW`，停止自動恢復等待人工介入 | Task 狀態變更為 `NEEDS_REVIEW`，Audit Log 記錄警示 | 已實作 / 通過 |
| **T11** | Task 重複執行 | Task 目前為 `RUNNING` 或 `SUCCEEDED` | 再次執行 `agent task run <id>` | 拒絕重複執行，回報狀態衝突錯誤訊息 | CLI 報錯退出，不啟動重複的 Worker 與 Workspace | 已實作 / 通過 |
| **T12** | Worker 修改非預期檔案 | Worker 試圖修改 Workspace 外部檔案 (路徑遍歷) | 執行 Validator 檢查變更檔案清單 | Validator 標記 `FilesInWorkspace=false`，將異常路徑加入 `UnexpectedFiles`，驗收失敗 | Task 轉為 `FAILED`，Summary 列出非預期檔案清單 | 已實作 / 通過 |

---

## 2. 詳細測試步驟與驗證指南

### T01: 正常建立 Task
```bash
./bin/agent task create --title "新增整數相加函式 Add" --description "在 sample-project 中新增 Add(a, b int) int 及測試" --worker aider
./bin/agent task list
./bin/agent task show <TASK_ID>
```

### T02 – T04: 完整執行與驗收流程 (Happy Path)
```bash
# 1. 執行 Task
./bin/agent task run <TASK_ID>

# 2. 檢視 Task 狀態與驗收結果
./bin/agent task show <TASK_ID>

# 預期輸出包含：
# Status: SUCCEEDED
# Result: Validation PASSED: Worker succeeded, git diff verified, all files within workspace, and tests passed.
```

### T05: Worker 執行失敗測試
可透過給予無效的 model 參數或無效指令模擬：
```bash
./bin/agent task create --title "錯誤測試" --description "exit 1" --worker aider
./bin/agent task run <TASK_ID>
# 預期：Task 標記為 FAILED
```

### T06 – T08: 異常中斷與崩潰恢復測試 (Recovery)
```bash
# 手動模擬中斷後的修復
./bin/agent recovery
# 檢視 Task 狀態是否已進入 RECONCILING 並根據結果恢復為 SUCCEEDED 或 UNKNOWN
```

### T09: Aider 不存在檢查
```bash
# 指向不存在 binary 測試 health check
./bin/agent worker health --config ./configs/config.yaml
```

### T10: Workspace 遺失檢查
```bash
# 模擬 Workspace 遺失
rm -rf workspaces/<TASK_ID>
./bin/agent recovery
./bin/agent task show <TASK_ID>
# 預期：Status 轉變為 NEEDS_REVIEW
```

### T11: 防止重複執行保護
```bash
# 對已處於 SUCCEEDED 或 RUNNING 狀態的 task 再次執行：
./bin/agent task run <TASK_ID>
# 預期回傳：task <TASK_ID> has already completed with SUCCEEDED status
```

### T12: 非預期檔案變更保護
單元測試 `internal/validator/validator_test.go` 內建路徑遍歷與意外檔案驗收測試，確保 Worker 無法修改 Workspace 以外之系統檔案。
