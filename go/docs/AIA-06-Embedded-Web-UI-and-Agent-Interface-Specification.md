# AIA-06：Embedded Web UI & Agent Interface Specification

**文件編號：** AIA-06  
**版本：** v1.0  
**狀態：** Implementation Baseline  
**適用系統：** AIA Personal Agent  
**技術基準：** Go Runtime、Embedded Static Assets、HTTP API、SSE  
**依賴文件：** AIA-01、AIA-02、AIA-03、AIA-04、AIA-05

---

# 1. 文件目的

本文件定義 AIA 的 Embedded Web UI 架構，以及 Domain Agent 如何提供自己的設定介面、檢視介面與操作介面。

AIA 採用單一 Runtime、單一 Web Host、多個 Agent UI Module 的設計。

每個 Domain Agent 可以提供自己的 Web UI，但不自行建立獨立 Web Server，也不直接管理全域 Runtime 資源。

Web UI 是 AIA 的操作入口之一，不是 Agent 的執行環境。

AIA 必須同時支援：

- CLI 操作。
- Embedded Web UI 操作。
- Runtime 背景 Task 執行。
- 未來其他 Client（例如手機 App）透過 API 操作。

所有入口共用相同的 Domain Service、Policy、Task、Memory、Audit 與資料儲存機制。

# 2. 設計原則

## 2.1 Single Runtime

整個 AIA 系統只存在一個主要 Runtime。

Runtime 負責：

- Task Lifecycle。
- Agent Registry。
- Agent Lifecycle。
- Execution Management。
- Policy Enforcement。
- Memory Management。
- Recovery。
- Audit。
- Web Host。
- API Authentication and Authorization。

Domain Agent 不得自行繞過 Runtime 管理上述全域功能。

## 2.2 Single Web Host

所有 Agent UI 均由 AIA Embedded Web Host 提供。

禁止每個 Agent 自行啟動獨立 HTTP Server，除非未來經過架構變更正式核准。

建議路由：

| 路徑 | 用途 |
|---|---|
| `/` | Global Console |
| `/console/` | 全域管理介面 |
| `/agents/` | Agent 清單 |
| `/agents/{agent_id}/` | Agent UI |
| `/api/v1/` | 全域 API |
| `/api/v1/agents/{agent_id}/` | Agent Domain API |
| `/events/` | SSE Event Stream |
| `/healthz` | Health Check |
| `/readyz` | Readiness Check |

實際路由不得與其他模組衝突。

## 2.3 Domain Isolation

Agent 可以管理自己的領域資料與設定，但不得自行管理其他 Agent 的資料。

例如：

Investment Agent 可以：

- 設定投資股票池。
- 檢視投資組合。
- 檢視研究報告。
- 檢視歷史建議。
- 提交投資分析 Task。
- 檢視預測評估結果。

Investment Agent 不可以：

- 修改全域 Policy。
- 修改其他 Agent 的設定。
- 任意取消其他 Agent 的 Task。
- 讀取其他 Agent 的私有資料。
- 修改 Runtime 的認證與授權設定。
- 直接存取其他 Agent 的資料表。

跨 Agent 操作必須經過 Runtime 授權與正式 Service Interface。

## 2.4 Shared Business Logic

CLI 與 Web UI 不得各自實作一套業務邏輯。

正確架構：

```text
CLI ─────────────┐
                 │
Web UI ──────────┼──> Application Services
                 │           │
Future Client ───┘           ▼
                         Domain Services
                              │
                              ▼
                         Repository
                              │
                              ▼
                           SQLite
```

UI 只負責呈現、輸入、操作及結果回報。

所有商業規則、資料驗證、權限判斷及交易邊界由 Service 層處理。

# 3. 系統組成

## 3.1 Global Console

Global Console 是 AIA 的全域管理介面。

至少提供以下功能：

| 功能 | 說明 |
|---|---|
| Dashboard | 系統狀態、執行摘要 |
| Tasks | Task 清單、狀態、詳細內容 |
| Executions | 執行紀錄與結果 |
| Agents | Agent 清單、狀態、能力 |
| Projects | Project 管理 |
| Memory | Memory 檢視與候選核准 |
| Policies | Policy 檢視及授權管理 |
| Approvals | 待人工核准項目 |
| Audit | 操作與執行稽核 |
| System | Runtime、Database、agy 健康狀態 |

全域管理操作必須經過 Runtime 的權限檢查。

## 3.2 Domain Agent UI

每個 Agent 可以註冊自己的 UI Module。

例如：

| Agent | UI 功能 |
|---|---|
| Investment Agent | 投資組合、研究報告、投資建議、資料來源 |
| Writing Agent | 寫作專案、文件、章節、審閱 |
| Coding Agent | 程式專案、執行紀錄、程式碼變更 |
| Video Agent | 影片專案、素材、處理工作、輸出結果 |

Agent UI 的功能由該 Agent 自行定義，但必須遵守 AIA 的共用安全規範。

## 3.3 Agent UI Module

Agent UI Module 是編譯期註冊的 Go 模組。

介面定義：

```go
type AgentWebModule interface {
    AgentID() string
    RegisterRoutes(mux *http.ServeMux) error
    NavItems() []NavItem
}
```

NavItem：

```go
type NavItem struct {
    ID       string `json:"id"`
    Label    string `json:"label"`
    Path     string `json:"path"`
    Icon     string `json:"icon,omitempty"`
    Order    int    `json:"order"`
    Required string `json:"required_permission,omitempty"`
}
```

模組必須遵守：

1. AgentID 必須與 Agent Registry 中的 ID 一致。
2. 不得註冊全域保留路徑。
3. 不得覆蓋其他 Agent 的路由。
4. 不得自行啟動 HTTP Server。
5. 不得直接修改 Runtime 全域設定。
6. 不得繞過 Service 層直接操作資料庫。
7. 不得將秘密、Token 或敏感憑證送至前端。
8. 必須提供錯誤處理與輸入驗證。
9. 不得假設 UI 一定存在或一定開啟。
10. Agent 停用時，其 UI 應顯示停用狀態，而非造成整個 Web Host 失效。

## 3.4 Agent Web Context

為避免模組自行建立不受控的全域依賴，Runtime 應提供受限的 Web Context。

```go
type AgentWebContext struct {
    AgentID   string
    Services  AgentServices
    Authorizer Authorizer
    Audit      AuditWriter
    EventBus   EventSubscriber
    Logger     Logger
}
```

AgentServices 僅包含該 Agent 被授權使用的 Service。

不得直接將 SQLite DB Handle、全域 Policy Engine、Credential Store 或其他 Agent 的 Repository 傳入 UI Module。

實際 Go Interface 以 AIA-03 為準。

# 4. HTTP API 設計

## 4.1 API 基本規則

- Prefix：`/api/v1`
- 格式：JSON
- 編碼：UTF-8
- 時間：RFC 3339，統一使用 UTC 儲存。
- ID：使用不可預測的唯一識別碼。
- API 必須具備輸入驗證。
- API 必須具備權限檢查。
- API 必須留下必要 Audit Record。
- API 不得回傳內部 Token、密碼或秘密資訊。

## 4.2 標準 Response

成功：

```json
{
  "success": true,
  "data": {},
  "request_id": "req_123"
}
```

失敗：

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request",
    "details": {}
  },
  "request_id": "req_123"
}
```

## 4.3 HTTP Status

| Status | 用途 |
|---|---|
| 200 | 成功 |
| 201 | 建立成功 |
| 202 | 已接受非同步工作 |
| 204 | 成功，無回傳內容 |
| 400 | Request 格式錯誤 |
| 401 | 未認證 |
| 403 | 權限不足 |
| 404 | 資源不存在 |
| 409 | 狀態衝突 |
| 422 | 業務規則不允許 |
| 429 | 請求頻率超限 |
| 500 | 內部錯誤 |
| 503 | Runtime 尚未 Ready |

錯誤內容不得洩露 Stack Trace、檔案路徑、Credential 或內部環境變數。

## 4.4 全域 API

最低必要 API：

| Method | Path | 功能 |
|---|---|---|
| GET | `/api/v1/system/status` | Runtime 狀態 |
| GET | `/api/v1/tasks` | Task 清單 |
| POST | `/api/v1/tasks` | 建立 Task |
| GET | `/api/v1/tasks/{id}` | Task 詳情 |
| POST | `/api/v1/tasks/{id}/cancel` | 取消 Task |
| POST | `/api/v1/tasks/{id}/resume` | 恢復 Task |
| GET | `/api/v1/agents` | Agent 清單 |
| GET | `/api/v1/agents/{id}` | Agent 詳情 |
| GET | `/api/v1/memory` | Memory 查詢 |
| GET | `/api/v1/approvals` | 待核准項目 |
| POST | `/api/v1/approvals/{id}/approve` | 核准 |
| POST | `/api/v1/approvals/{id}/reject` | 拒絕 |
| GET | `/api/v1/audit` | Audit 查詢 |

所有 API 都必須使用 Runtime Service。

## 4.5 Domain API

Domain Agent 可在自己的 Namespace 下註冊 API。

例如：

```text
/api/v1/agents/investment/profile
/api/v1/agents/investment/watchlist
/api/v1/agents/investment/portfolio
/api/v1/agents/investment/research
/api/v1/agents/investment/decisions
/api/v1/agents/investment/predictions
```

Domain API 的業務規則仍由 Agent Domain Service 負責。

Agent 不得以 API 路徑名稱作為授權依據；授權必須使用明確的 Permission。

# 5. 非同步工作與事件通知

## 5.1 非同步 Task

長時間工作不得讓 HTTP Request 一直等待。

例如：

- 投資研究。
- 文件分析。
- 程式碼修改。
- 影片處理。
- 多 Agent 協作。

API 應建立 Task 後立即回傳：

```json
{
  "success": true,
  "data": {
    "task_id": "task_123",
    "status": "QUEUED"
  }
}
```

UI 透過 Task API 或 SSE 查詢進度。

## 5.2 Server-Sent Events

第一階段使用 SSE，不預設使用 WebSocket。

Endpoint：

```text
GET /events/
```

事件類型：

- `task.created`
- `task.updated`
- `task.completed`
- `task.failed`
- `task.waiting_approval`
- `task.waiting_auth`
- `agent.status_changed`
- `system.health_changed`

事件格式：

```text
id: evt_123
event: task.updated
data: {"task_id":"task_123","status":"RUNNING"}

```

SSE 必須支援：

- Reconnection。
- Last-Event-ID。
- 心跳。
- 權限檢查。
- Client Disconnect。
- Event Replay 或安全的重新查詢機制。

SSE 不得被視為唯一資料來源。前端重新連線後，仍須透過 REST API 取得最新狀態。

# 6. 前端與靜態資源

## 6.1 Embedded Assets

Web UI 靜態資源必須在建置時嵌入 Go Binary。

使用：

```go
//go:embed web/dist/*
var webAssets embed.FS
```

不得要求使用者另外安裝 Node.js、Nginx 或其他 Web Server 才能啟動 AIA。

前端可以使用獨立的建置工具，但正式執行環境只需 AIA Binary。

## 6.2 UI 技術選擇

第一階段不強制指定前端框架。

應符合：

- 靜態 HTML/CSS/JavaScript。
- 可編譯成靜態資源。
- 支援 REST API。
- 支援 SSE。
- 不要求瀏覽器外掛。
- 不在前端保存敏感 Credential。
- 不將前端視為安全邊界。

前端框架由實作階段決定，必須記錄版本及建置方式。

## 6.3 UI Namespace

每個 Agent 的 UI 應位於獨立 Namespace：

```text
/agents/investment/
/agents/writing/
/agents/coding/
/agents/video/
```

靜態資源路徑不得互相衝突。

# 7. Authentication & Authorization

## 7.1 第一階段部署

預設只監聽：

```text
127.0.0.1
```

不得預設監聽 `0.0.0.0`。

本機 Web UI 仍須防範：

- CSRF。
- 跨站腳本。
- Session 劫持。
- 不安全的 Cookie。
- 未授權 API 操作。
- 惡意第三方網站透過瀏覽器發送本機請求。

## 7.2 Session

建議採用 Server-Side Session。

Cookie 應設定：

- HttpOnly。
- SameSite=Strict 或經安全評估的設定。
- Secure：HTTPS 部署時必須啟用。
- 合理的 Max-Age。
- Session Rotation。

不得將 Session ID 放入 URL。

不得將 agy OAuth Token 放入 Web Session 或前端。

## 7.3 CSRF

所有修改狀態的 API 必須有 CSRF 防護。

GET、HEAD、OPTIONS 不得執行狀態變更。

必須驗證 Origin，並搭配 CSRF Token 或等效的安全機制。

## 7.4 Permission Model

權限採明確 Permission。

例如：

```text
system.read
system.manage
task.read
task.create
task.cancel
agent.read
agent.manage
memory.read
memory.approve
policy.read
policy.manage
audit.read
investment.read
investment.configure
investment.research
investment.decision.read
```

不得因為使用者可以開啟某個頁面，就推定其有權呼叫該頁面的所有 API。

每次 API Request 均須重新檢查授權。

## 7.5 Remote Access

第一階段不實作遠端存取。

未來若要開放 LAN 或 Internet：

- 必須使用 HTTPS。
- 必須具備正式 Authentication。
- 必須具備 RBAC 或等效授權。
- 必須有 Session 管理。
- 必須有 CSRF、CORS、Rate Limit。
- 必須有遠端存取 Audit。
- 必須明確定義信任邊界。
- 不得直接將本機模式改成 `0.0.0.0` 就視為完成遠端安全設計。

# 8. Audit 與操作追蹤

所有具備業務影響的 UI 操作都必須留下 Audit Record。

最低欄位：

- actor_id
- actor_type
- action
- resource_type
- resource_id
- request_id
- timestamp
- result
- source（CLI / WEB / API）
- metadata

不可將密碼、Token、Cookie、API Key 或完整敏感內容寫入 Audit。

重要操作包含：

- 建立、取消、恢復 Task。
- 核准、拒絕。
- 修改 Agent 設定。
- 修改投資限制。
- 修改資料來源。
- 修改策略版本。
- 修改全域 Policy。

Audit 不得由 Agent UI 自行刪除或覆寫。

# 9. 錯誤處理與可用性

- 單一 Agent UI 初始化失敗，不得導致整個 Web Host 無法啟動。
- Agent UI Module 註冊失敗時，Runtime 必須留下明確錯誤紀錄。
- Agent 停用時，該 Agent UI 顯示不可用狀態。
- API 必須使用統一錯誤格式。
- SSE 連線中斷不應影響 Task 執行。
- Web Host 關閉不應取消正在執行的 Task。
- Browser 關閉不應影響 Agent。
- UI 不得自行推斷 Task 成功；必須讀取 Runtime 的正式 Task 狀態。

# 10. Security Requirements

| ID | 要求 |
|---|---|
| WEB-SEC-01 | 預設只監聽 127.0.0.1 |
| WEB-SEC-02 | 禁止前端取得 agy OAuth Token |
| WEB-SEC-03 | 所有 API 必須驗證權限 |
| WEB-SEC-04 | 所有寫入 API 必須防止 CSRF |
| WEB-SEC-05 | 不得信任前端輸入 |
| WEB-SEC-06 | 不得由 UI 直接操作資料庫 |
| WEB-SEC-07 | 不得將秘密寫入 Log |
| WEB-SEC-08 | 不得使用不安全的動態路由註冊 |
| WEB-SEC-09 | 必須限制 Request Body 大小 |
| WEB-SEC-10 | 必須設定 Request Timeout |
| WEB-SEC-11 | 必須處理 XSS、CSP 與安全 Header |
| WEB-SEC-12 | 必須限制登入及敏感 API 的請求頻率 |
| WEB-SEC-13 | 遠端存取必須另行核准與設計 |
| WEB-SEC-14 | 不得因本機部署而省略授權檢查 |

# 11. Logging & Monitoring

Web Host 必須提供：

- HTTP Request Log。
- API Error Log。
- Authentication Event。
- Authorization Denial。
- Agent Module Registration。
- SSE Connection。
- Web Host Startup / Shutdown。

Log 至少包含：

- timestamp
- request_id
- method
- path
- status
- duration
- actor_id（可取得時）
- source_ip（依部署模式）
- error_code

不得記錄完整敏感 Request Body。

# 12. Implementation Milestones

## WEB-0：Architecture

- 定義 Web Host。
- 定義 AgentWebModule。
- 定義 AgentWebContext。
- 定義 Global Console 路由。
- 定義 Domain Namespace。

## WEB-1：Embedded Host

- 建立 Go HTTP Server。
- 使用 embed.FS。
- 建立靜態資源 Handler。
- 建立健康檢查 API。
- 預設監聽 127.0.0.1。

## WEB-2：Global Console

- System Status。
- Task List / Detail。
- Agent List。
- Approval List。
- Audit Viewer。

## WEB-3：Agent UI

- Agent Module Registry。
- Agent Navigation。
- Domain API。
- 靜態資源 Namespace。
- 模組隔離。

## WEB-4：Authentication & Security

- Session。
- Permission。
- CSRF。
- Origin Validation。
- Security Headers。
- Rate Limit。

## WEB-5：Event Stream

- SSE。
- Reconnection。
- Last-Event-ID。
- Event Authorization。

## WEB-6：Integration

- CLI / Web 共用 Service。
- Task 執行與 UI 狀態同步。
- Agent UI 隔離測試。
- Security Tests。
- Build / Release。

# 13. Acceptance Criteria

| ID | 驗收條件 |
|---|---|
| WEB-AC-01 | AIA 可啟動單一 Embedded Web Host |
| WEB-AC-02 | Web UI 靜態資源已嵌入 Go Binary |
| WEB-AC-03 | 預設只監聽 127.0.0.1 |
| WEB-AC-04 | Global Console 可檢視 Runtime 狀態 |
| WEB-AC-05 | Global Console 可檢視 Task |
| WEB-AC-06 | 可註冊至少兩個不同 Agent UI Module |
| WEB-AC-07 | Agent UI Namespace 不互相衝突 |
| WEB-AC-08 | Agent UI 不直接存取 DB |
| WEB-AC-09 | CLI 與 Web 使用相同 Service |
| WEB-AC-10 | Web UI 可建立非同步 Task |
| WEB-AC-11 | Task 執行不依賴瀏覽器存活 |
| WEB-AC-12 | SSE 可接收 Task 狀態更新 |
| WEB-AC-13 | SSE 斷線不影響 Task |
| WEB-AC-14 | 未授權 API Request 被拒絕 |
| WEB-AC-15 | CSRF 測試通過 |
| WEB-AC-16 | Agent Module 失敗不影響其他模組 |
| WEB-AC-17 | Web 操作可追溯至 Audit |
| WEB-AC-18 | agy OAuth Token 不暴露於前端 |
| WEB-AC-19 | Web Host Shutdown 不會誤取消 Task |
| WEB-AC-20 | Go Binary 可在沒有獨立 Web Server 的環境執行 |

# 14. 非目標

第一階段不包含：

- 多使用者協作。
- 公網部署。
- 多租戶。
- 動態載入任意 Go Plugin。
- Agent 自行啟動 Web Server。
- 前端直接存取 SQLite。
- 瀏覽器保存 OAuth Credential。
- WebSocket 即時雙向通訊。
- 手機原生 App。
- 遠端 Agent UI 熱插拔。

未來需要時，應透過新版本規格擴充，不得以臨時方式破壞既有安全邊界。

# 15. 最終架構原則

AIA 的 Web UI 是 Runtime 的一部分，不是另一套系統。

Agent 可以擁有自己的操作介面，但不能擁有自己的全域治理規則。

所有操作入口都必須回到同一套 Service、Policy、Task、Memory、Audit 與 Recovery。

**一個 Runtime、一個 Web Host、多個 Agent Module、多種 Client。**
