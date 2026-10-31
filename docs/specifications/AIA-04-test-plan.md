# AIA-04：Test Plan, Acceptance Criteria & Release Specification

**文件編號：** AIA-04  
**版本：** v1.1  
**狀態：** Implementation Baseline  
**適用系統：** AIA Personal Agent  
**前置文件：** AIA-01、AIA-02、AIA-03  
**相關文件：** AIA-05、AIA-06

---

# 1. 文件目的

本文件定義 AIA 的測試策略、功能驗收條件、安全性驗收、整合測試、版本交付及 Release Gate。

本版本新增 Embedded Web UI、HTTP API、Agent Web Module、SSE、Web Security 與 CLI/Web 共用服務的測試要求。

本文件不取代 AIA-03 的實作規格，而是定義如何驗證 AIA-03 所要求的行為。

# 2. 測試原則

## 2.1 Test Pyramid

測試分為：

1. Unit Test。
2. Integration Test。
3. System Test。
4. End-to-End Test。
5. Security Test。
6. Recovery Test。
7. Acceptance Test。

Unit Test 應涵蓋大部分業務規則。

End-to-End Test 驗證完整操作流程，不應取代 Unit Test。

## 2.2 可重複測試

測試必須：

- 可重複執行。
- 不依賴真實使用者資料。
- 不依賴未受控外部服務。
- 可使用 Mock Harness。
- 可使用 Temporary SQLite。
- 可建立獨立 Workspace。
- 不因執行順序而改變結果。

## 2.3 測試分層

| 類別 | 目的 |
|---|---|
| Unit | 單一模組行為 |
| Integration | 多模組協作 |
| Harness | agy Adapter |
| Database | Migration、Transaction |
| Recovery | Crash、Retry、Resume |
| Web | Host、API、UI Module |
| Security | 權限、CSRF、Session |
| E2E | 完整使用者流程 |
| Release | Binary 與部署驗證 |

# 3. 測試環境

## 3.1 基本環境

- Go。
- SQLite。
- agy CLI。
- Temporary Workspace。
- Local Embedded Web Host。
- 測試用 Configuration。
- 測試用 Agent。
- 測試用使用者 Session。

## 3.2 測試資料隔離

每次測試應使用獨立：

- Database。
- Workspace。
- Project。
- Task。
- Session。
- Event Stream。

測試不得直接使用正式資料目錄。

## 3.3 Harness 測試

Harness 測試分為：

**Mock Harness**

用於：

- 正常執行。
- Timeout。
- Process Crash。
- Auth Failure。
- Soft-deny。
- 非預期輸出。
- Workspace 無變更。
- Result Artifact 不存在。

**Real Harness**

用於驗證 agy 實際整合。

Real Harness 測試必須明確記錄 agy 版本、測試環境及通過範圍。

不得因少數整合測試通過，就宣稱所有 Harness 功能均已驗證。

# 4. Core Runtime Test

## 4.1 Bootstrap

| ID | 測試 | 預期結果 |
|---|---|---|
| RT-T01 | Runtime 啟動 | 正常 |
| RT-T02 | 缺少必要設定 | 明確錯誤 |
| RT-T03 | Database 不可用 | 啟動失敗或 Not Ready |
| RT-T04 | Workspace 不可用 | 明確錯誤 |
| RT-T05 | Agent Registry 初始化 | 正常 |
| RT-T06 | 重複註冊 Agent | 拒絕 |
| RT-T07 | 關閉 Runtime | 正常 Shutdown |

## 4.2 Task Management

| ID | 測試 | 預期結果 |
|---|---|---|
| TK-T01 | 建立 Task | CREATED |
| TK-T02 | Task 排入 Queue | QUEUED |
| TK-T03 | Task 開始執行 | RUNNING |
| TK-T04 | Task 完成驗證 | COMPLETED |
| TK-T05 | Task 執行失敗 | FAILED |
| TK-T06 | Task 取消 | CANCELLED |
| TK-T07 | Task Resume | 依合法狀態恢復 |
| TK-T08 | 非法狀態轉移 | 拒絕 |
| TK-T09 | 不存在 Task | NOT FOUND |
| TK-T10 | 重複建立同一請求 | 依 Idempotency Policy 處理 |

## 4.3 Dependency

| ID | 測試 | 預期結果 |
|---|---|---|
| DP-T01 | 單一依賴 | 正常 |
| DP-T02 | 多個依賴 | 全部符合條件才啟動 |
| DP-T03 | Parallel Execution | 可平行執行 |
| DP-T04 | Dependency Failure | 下游依 Policy 處理 |
| DP-T05 | Dependency Cycle | 拒絕 |
| DP-T06 | Fan-out / Fan-in | 正常彙整 |

## 4.4 Policy

| ID | 測試 | 預期結果 |
|---|---|---|
| PL-T01 | Low Risk | ALLOW |
| PL-T02 | Medium Risk | 依設定處理 |
| PL-T03 | High Risk | REQUIRE_APPROVAL |
| PL-T04 | Critical Risk | DENY |
| PL-T05 | 未授權操作 | DENY |
| PL-T06 | Agent 嘗試繞過 Policy | 拒絕 |
| PL-T07 | Web API 繞過 Policy | 拒絕 |
| PL-T08 | CLI 繞過 Policy | 拒絕 |

## 4.5 Memory

| ID | 測試 | 預期結果 |
|---|---|---|
| MM-T01 | 建立 Candidate | PENDING |
| MM-T02 | 核准 Candidate | APPROVED |
| MM-T03 | 拒絕 Candidate | REJECTED |
| MM-T04 | 未核准 Candidate 搜尋 | 不得作為正式 Memory |
| MM-T05 | Memory Search | 回傳符合條件結果 |
| MM-T06 | Memory Source | 可追溯 |
| MM-T07 | 重複 Candidate | 依去重策略處理 |

# 5. Harness Integration Test

## 5.1 Version & Health

| ID | 測試 | 預期結果 |
|---|---|---|
| AG-T01 | agy Binary 存在 | 正常 |
| AG-T02 | agy Version | 可取得 |
| AG-T03 | agy 不存在 | 明確錯誤 |
| AG-T04 | agy HealthCheck | 正常或明確失敗 |
| AG-T05 | 不支援版本 | 依版本政策拒絕或警告 |

## 5.2 Execution

| ID | 測試 | 預期結果 |
|---|---|---|
| AG-T06 | 啟動 agy | 正常 |
| AG-T07 | stream-json init | 可解析 |
| AG-T08 | stream-json step_update | 可解析 |
| AG-T09 | stream-json result | 可解析 |
| AG-T10 | Process Exit | 正確保存 |
| AG-T11 | stderr | 可記錄及分類 |
| AG-T12 | Workspace Diff | 可取得 |
| AG-T13 | Result Artifact | 可取得 |
| AG-T14 | Timeout | 正確處理 |
| AG-T15 | Cancel | 正確處理 |

## 5.3 Permission & Soft-deny

| ID | 測試 | 預期結果 |
|---|---|---|
| AG-T16 | 正常允許工具 | 正常執行 |
| AG-T17 | 需要人工核准工具 | 不得誤判成功 |
| AG-T18 | Soft-deny | Execution 結果明確 |
| AG-T19 | Exit Code 0 但工具未執行 | 不得自動完成 Task |
| AG-T20 | Critical Risk | 由 Policy 阻止 |
| AG-T21 | Sandbox 限制 | 不得越界 |

## 5.4 Authentication

| ID | 測試 | 預期結果 |
|---|---|---|
| AG-T22 | 正常認證 | 可執行 |
| AG-T23 | 認證失效 | WAITING_FOR_AUTH |
| AG-T24 | 使用者重新登入 | 可 Resume |
| AG-T25 | Resume 後 Session | 依 Harness 能力恢復 |
| AG-T26 | Token 不可讀 | AIA 不得讀取 Token |

# 6. Recovery Test

| ID | 測試 | 預期結果 |
|---|---|---|
| RC-T01 | Runtime Crash | 可恢復 |
| RC-T02 | Execution 中斷 | 狀態可判斷 |
| RC-T03 | Retry | 正常 |
| RC-T04 | Retry Limit | FAILED |
| RC-T05 | Auth Failure | WAITING_FOR_AUTH |
| RC-T06 | Approval Pending | WAITING_APPROVAL |
| RC-T07 | Dependency Pending | WAITING_FOR_DEPENDENCY |
| RC-T08 | SUCCESS 未更新 Task | 進入 Validation |
| RC-T09 | 重啟後重複恢復 | 不重複執行成功結果 |
| RC-T10 | 外部副作用狀態未知 | 進入 Reconciliation |
| RC-T11 | 已取消 Task | 不得自行恢復執行 |
| RC-T12 | Crash 發生於 DB Transaction | 資料保持一致 |

# 7. Database Test

| ID | 測試 | 預期結果 |
|---|---|---|
| DB-T01 | SQLite WAL | 正常 |
| DB-T02 | Foreign Key | 正常 |
| DB-T03 | Migration Up | 成功 |
| DB-T04 | Migration 重複執行 | 不重複破壞 |
| DB-T05 | Transaction Rollback | 正常 |
| DB-T06 | Concurrent Read | 正常 |
| DB-T07 | Concurrent Write | 不發生資料損毀 |
| DB-T08 | Backup | 可還原 |
| DB-T09 | Integrity Check | 通過 |
| DB-T10 | Schema Version | 正確 |

# 8. Embedded Web Host Test

## 8.1 Host Startup

| ID | 測試 | 預期結果 |
|---|---|---|
| WEB-T01 | 啟動 Web Host | 正常 |
| WEB-T02 | 重複啟動 | 明確處理 |
| WEB-T03 | Port 已被占用 | 明確錯誤 |
| WEB-T04 | 停止 Web Host | 正常 |
| WEB-T05 | Runtime 關閉 | Web Host 正常 Shutdown |
| WEB-T06 | Web Host 關閉時 Task 執行 | Task 不受影響 |
| WEB-T07 | 只監聽 localhost | 預設符合 |
| WEB-T08 | Embedded Assets | 可正常讀取 |

## 8.2 Static Assets

| ID | 測試 | 預期結果 |
|---|---|---|
| WEB-T09 | Global Console | 正常載入 |
| WEB-T10 | Agent UI | 正常載入 |
| WEB-T11 | SPA Fallback | 正確 |
| WEB-T12 | API 不被 SPA 攔截 | 正確 |
| WEB-T13 | SSE 不被 SPA 攔截 | 正確 |
| WEB-T14 | 缺少靜態資源 | 明確回應 |
| WEB-T15 | Binary 不依賴外部 UI 檔案 | 通過 |

## 8.3 Agent Web Module

| ID | 測試 | 預期結果 |
|---|---|---|
| WEB-T16 | 註冊 Agent Module | 正常 |
| WEB-T17 | Agent ID 不存在 | 拒絕 |
| WEB-T18 | Agent ID 不一致 | 拒絕 |
| WEB-T19 | Route Collision | 拒絕 |
| WEB-T20 | Nav Item 重複 | 拒絕 |
| WEB-T21 | Module 初始化失敗 | 其他 Module 正常 |
| WEB-T22 | Agent 停用 | UI 顯示不可用 |
| WEB-T23 | Agent UI 直接存取 DB | 架構測試禁止 |
| WEB-T24 | Agent UI 嘗試跨 Domain 操作 | 權限拒絕 |

## 8.4 REST API

| ID | 測試 | 預期結果 |
|---|---|---|
| WEB-T25 | System Status API | 正常 |
| WEB-T26 | Task List API | 正常 |
| WEB-T27 | Task Create API | 正常 |
| WEB-T28 | Task Detail API | 正常 |
| WEB-T29 | Task Cancel API | 正常 |
| WEB-T30 | Task Resume API | 正常 |
| WEB-T31 | Agent API | 正常 |
| WEB-T32 | Domain API | 正常 |
| WEB-T33 | Invalid JSON | 400 |
| WEB-T34 | Invalid Business Input | 422 |
| WEB-T35 | Unknown Resource | 404 |
| WEB-T36 | Unauthorized | 401 |
| WEB-T37 | Forbidden | 403 |
| WEB-T38 | Request ID | 存在 |
| WEB-T39 | API Error Format | 統一 |
| WEB-T40 | Request Body 過大 | 拒絕 |

# 9. Web Security Test

## 9.1 Authentication

| ID | 測試 | 預期結果 |
|---|---|---|
| SEC-T01 | 未登入存取受保護 API | 401 |
| SEC-T02 | 有效 Session | 正常 |
| SEC-T03 | 過期 Session | 重新認證 |
| SEC-T04 | Logout | Session 失效 |
| SEC-T05 | Session Rotation | 正常 |
| SEC-T06 | Session 不出現在 URL | 通過 |
| SEC-T07 | agy OAuth Token 不暴露 | 通過 |

## 9.2 Authorization

| ID | 測試 | 預期結果 |
|---|---|---|
| SEC-T08 | 無 task.read | 拒絕 |
| SEC-T09 | 無 task.create | 拒絕 |
| SEC-T10 | 無 task.cancel | 拒絕 |
| SEC-T11 | 無 memory.approve | 拒絕 |
| SEC-T12 | 無 policy.manage | 拒絕 |
| SEC-T13 | 跨 Agent 存取 | 拒絕 |
| SEC-T14 | 前端隱藏按鈕但直接呼叫 API | 仍拒絕 |

## 9.3 CSRF / CORS / XSS

| ID | 測試 | 預期結果 |
|---|---|---|
| SEC-T15 | 缺少 CSRF Token | 拒絕 |
| SEC-T16 | 不可信 Origin | 拒絕 |
| SEC-T17 | GET 狀態變更 | 不允許 |
| SEC-T18 | 未授權 CORS | 不允許 |
| SEC-T19 | XSS 輸入 | 不得執行 |
| SEC-T20 | CSP | 正確設定 |
| SEC-T21 | Content-Type Sniffing | 防護 |
| SEC-T22 | Clickjacking | 防護 |

## 9.4 Abuse Protection

| ID | 測試 | 預期結果 |
|---|---|---|
| SEC-T23 | 大量登入嘗試 | Rate Limit |
| SEC-T24 | 大量 API Request | Rate Limit |
| SEC-T25 | 異常 Request Body | 拒絕 |
| SEC-T26 | Path Traversal | 拒絕 |
| SEC-T27 | 不安全檔案路徑 | 拒絕 |
| SEC-T28 | Log Sensitive Data | 不允許 |

# 10. SSE Test

| ID | 測試 | 預期結果 |
|---|---|---|
| SSE-T01 | SSE Connect | 正常 |
| SSE-T02 | Task Event | 收到事件 |
| SSE-T03 | Agent Event | 收到事件 |
| SSE-T04 | Heartbeat | 正常 |
| SSE-T05 | Client Disconnect | 正常清理 |
| SSE-T06 | Reconnect | 正常 |
| SSE-T07 | Last-Event-ID | 可恢復或重新查詢 |
| SSE-T08 | Unauthorized Subscription | 拒絕 |
| SSE-T09 | SSE 斷線 | Task 繼續執行 |
| SSE-T10 | 重複事件 | 前端可依 Event ID 去重 |

# 11. CLI / Web Consistency Test

本測試非常重要。

同一個業務操作，CLI 與 Web 必須產生相同的 Domain 結果。

| ID | 測試 | 預期結果 |
|---|---|---|
| UI-T01 | CLI 建立 Task，Web 檢視 | 相同 Task |
| UI-T02 | Web 建立 Task，CLI 檢視 | 相同 Task |
| UI-T03 | CLI 修改 Domain 設定，Web 檢視 | 相同設定 |
| UI-T04 | Web 修改 Domain 設定，CLI 檢視 | 相同設定 |
| UI-T05 | CLI 觸發 Task，Web 顯示進度 | 一致 |
| UI-T06 | Web 觸發 Task，CLI 顯示進度 | 一致 |
| UI-T07 | CLI 與 Web 同時修改 | 依 Concurrency Policy 處理 |
| UI-T08 | Audit Source | 正確區分 CLI / WEB |

不得在 CLI 與 Web 各自建立重複的業務邏輯。

# 12. End-to-End Test

## E2E-01：Basic Coding Task

1. 使用者建立 Coding Task。
2. Personal Agent 分類。
3. 指派 Coding Agent。
4. Harness 執行。
5. 收集結果。
6. Result Validation。
7. Task Completed。
8. CLI 與 Web 均可查詢結果。

預期：完整成功，Audit 可追溯。

## E2E-02：Multi-Agent Dependency

1. 建立 Parent Task。
2. 建立 Writing Task。
3. 建立 Coding Task。
4. 設定 Dependency。
5. Writing 完成後啟動 Coding。
6. 彙整結果。
7. Validation。

預期：Dependency 正確，沒有提前執行。

## E2E-03：Memory Approval

1. Agent 提出 Memory Candidate。
2. Web 顯示待核准。
3. 使用者核准。
4. Memory 進入正式索引。
5. CLI 查詢可找到該 Memory。

預期：核准前不可作為正式 Memory。

## E2E-04：Policy Safety

1. Agent 提出 Critical Risk 操作。
2. Policy Engine 評估。
3. 拒絕執行。
4. Audit 保存結果。
5. Web 顯示拒絕原因。

預期：不得發生危險操作。

## E2E-05：Crash Recovery

1. Task 執行。
2. 模擬 Runtime Crash。
3. 重啟 Runtime。
4. Recovery Manager 掃描。
5. 恢復合法 Task。
6. 不重複執行已成功操作。

預期：資料一致，結果可追溯。

## E2E-06：Authentication Failure

1. Harness 認證失效。
2. Task 進入 WAITING_FOR_AUTH。
3. Web 顯示狀態。
4. 使用者自行重新登入 agy。
5. CLI 或 Web 執行 Resume。
6. Task 繼續。

預期：AIA 不讀取 Token。

## E2E-07：Embedded Web Integration

1. 啟動 AIA。
2. Web Host 啟動。
3. 開啟 Global Console。
4. 檢視 Agent。
5. 開啟 Investment Agent UI。
6. 修改測試設定。
7. 透過 API 建立 Task。
8. SSE 顯示進度。
9. Task 完成。
10. Audit 可查詢。

預期：全流程不需要獨立 Web Server。

## E2E-08：Web Host Failure Isolation

1. 啟動 Runtime。
2. 執行 Task。
3. 關閉 Web Host。
4. 確認 Task 繼續執行。
5. 重新啟動 Web Host。
6. 查詢 Task 最新狀態。

預期：Web Host 不控制 Task Lifecycle。

# 13. Performance Test

第一階段以單一個人使用者為基準。

至少測試：

- Task List 查詢。
- Task Detail 查詢。
- Agent List 查詢。
- Domain API 查詢。
- SSE 長連線。
- 多個並行 Task。
- SQLite Concurrent Read / Write。
- Web Host Shutdown。

性能指標應先以實測建立 Baseline，不應在沒有測量資料時宣稱固定吞吐量。

最低要求：

- 一般 API 不得因 SSE 長連線而被阻塞。
- 慢速 Client 不得無限制占用資源。
- Request Body 必須有大小限制。
- SSE Connection 必須有合理上限。
- Runtime Shutdown 必須有 Timeout 與強制結束策略。

# 14. Code Quality & Build

必要檢查：

```bash
gofmt -w .
go vet ./...
go test ./...
go test -v -count=1 ./...
go test -race ./...
```

若使用前端建置工具，另需執行：

- Frontend Build。
- Frontend Lint。
- Dependency Audit。
- Embedded Asset Verification。

CI 必須能在乾淨環境重新建置完整 Binary。

不得依賴開發者本機未記錄的檔案或環境設定。

# 15. Release Acceptance

## 15.1 Release Gate

| Gate | 要求 |
|---|---|
| G1 | 所有必要 Unit Tests 通過 |
| G2 | Integration Tests 通過 |
| G3 | Recovery Tests 通過 |
| G4 | Harness Integration Tests 通過 |
| G5 | Web API Tests 通過 |
| G6 | Security Tests 通過 |
| G7 | CLI / Web Consistency 通過 |
| G8 | E2E Tests 通過 |
| G9 | Race Test 通過 |
| G10 | Release Binary 可執行 |
| G11 | Database Migration 正常 |
| G12 | 文件與版本一致 |

任何 Critical Security Test 失敗，不得 Release。

## 15.2 Release Package

至少包含：

- AIA Binary。
- Default Configuration。
- Database Migration。
- README。
- CHANGELOG。
- Release Notes。
- Version Information。
- Build Information。
- Checksum。

UI 靜態資源必須包含在 Binary 內。

## 15.3 Versioning

採 Semantic Versioning：

```text
MAJOR.MINOR.PATCH
```

- MAJOR：不相容架構變更。
- MINOR：向下相容功能新增。
- PATCH：錯誤修正。

Database Schema 必須獨立保存 Migration Version。

# 16. Test Evidence

每次測試應保留：

- Test ID。
- Test Environment。
- Software Version。
- agy Version。
- Test Start / End Time。
- Result。
- Error Log。
- Relevant Artifact。
- Tester。
- Known Limitations。

測試結果分為：

- PASS。
- FAIL。
- BLOCKED。
- NOT TESTED。

不得把 NOT TESTED 寫成 PASS。

## 16.1 Harness Verification Scope

真實 agy 整合測試必須逐項記錄。

例如：

| 項目 | 狀態 |
|---|---|
| Version Check | PASS / FAIL / NOT TESTED |
| HealthCheck | PASS / FAIL / NOT TESTED |
| stream-json Init | PASS / FAIL / NOT TESTED |
| stream-json Step Update | PASS / FAIL / NOT TESTED |
| stream-json Result | PASS / FAIL / NOT TESTED |
| Permission Handling | PASS / FAIL / NOT TESTED |
| Soft-deny Detection | PASS / FAIL / NOT TESTED |
| Sandbox Isolation | PASS / FAIL / NOT TESTED |
| Cancellation | PASS / FAIL / NOT TESTED |
| Timeout | PASS / FAIL / NOT TESTED |
| Session Resume | PASS / FAIL / NOT TESTED |
| Workspace Isolation | PASS / FAIL / NOT TESTED |

不得因部分項目通過，就將整個 Harness Integration 標記為完整通過。

# 17. Known Risk Register

| ID | 風險 | 處理方式 |
|---|---|---|
| R-01 | Harness Exit Code 誤判 | 多訊號判斷 |
| R-02 | Soft-deny | 檢查工具事件與結果 |
| R-03 | Crash 後重複執行 | Idempotency |
| R-04 | 外部副作用狀態未知 | Operation Journal |
| R-05 | SQLite Concurrent Write | Transaction / Retry |
| R-06 | Memory 錯誤寫入 | Approval |
| R-07 | Policy 被繞過 | Central Enforcement |
| R-08 | Web CSRF | CSRF Token / Origin |
| R-09 | Web Session 洩漏 | HttpOnly / SameSite / Rotation |
| R-10 | Agent UI 越權 | Domain Permission |
| R-11 | SSE 事件遺失 | Last-Event-ID / REST Reconciliation |
| R-12 | Web Host 影響 Task | Lifecycle Isolation |
| R-13 | 前端直接存取 DB | Service Boundary |
| R-14 | 靜態資源與 API 路由衝突 | Namespace Isolation |
| R-15 | localhost 被惡意網站利用 | Origin / CSRF / Host Validation |

# 18. Definition of Done

功能只有在以下條件全部符合時，才算完成：

1. 實作完成。
2. Unit Test 完成。
3. Integration Test 完成。
4. Error Handling 完成。
5. Policy 與權限檢查完成。
6. Audit 完成。
7. Recovery 行為明確。
8. 文件更新。
9. 測試結果保存。
10. Release Gate 通過。

Web UI 功能另須符合：

- API 完整。
- Session 與權限完成。
- CSRF 完成。
- Agent Namespace 隔離。
- CLI / Web 共用 Service。
- SSE 斷線不影響 Task。
- Web Host Shutdown 不影響 Runtime 工作。

# 19. 最終驗收原則

AIA 的成功不能只以「程式可以執行」判斷。

必須同時確認：

- 任務能正確完成。
- 結果能驗證。
- 狀態能持久化。
- Crash 後能恢復。
- Policy 能阻止不允許的操作。
- Agent 之間有清楚邊界。
- CLI 與 Web 操作結果一致。
- UI 不會控制或破壞 Runtime Lifecycle。
- 每個重要決策與操作均可追溯。

**AIA 的驗收標準是系統行為可靠、可驗證、可恢復、可稽核，而不是單純功能展示成功。**
