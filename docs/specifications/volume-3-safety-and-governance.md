# AIA Volume 3: Safety, Governance & Self-Evolution

**Document ID:** AIA-VOL-03  
**Title:** 安全治理、審批與受控自修規範 (Safety, Governance & Self-Evolution Specification)  
**Status:** Normative Consolidated Baseline  
**Consolidates:** AIA-14, AIA-17  

---

## 1. 運作治理核心哲學 (Governance Philosophy)

AIA 的自主運作必須在嚴格的安全治理邊界內進行。自主性（Autonomy）與受控性（Control）不是對立關係，治理機制是確保 Agent 在無人值守時仍可被信任的基石。

核心鐵律：
1. **分析不等於執行授權**：Agent 可以自主推演「建議重構整個模組」，但不代表它自動獲得刪除或重寫現有代碼的權限。
2. **反自我審批（Anti-Self-Approval）**：當一項變更被標記為需要人類批准（`RequiresHumanApproval = true`）時，系統底層強制檢查批准者身份，嚴禁由發起 Agent 自行批准（`ErrAgentSelfApprovalForbidden`）。
3. **不可逆性分級**：凡涉及金錢、外部網絡副作用、系統核心配置或無法 100% 回滾的操作，一律標註為高風險等級（`RiskTierCritical` / `RiskTierHigh`）。

---

## 2. 三級修改範圍隔離 (Tri-Level Scope Isolation)

依據原 AIA-14 規範，系統代碼與環境劃分為嚴格的三層修改邊界：

| 層級 | 名稱 | 覆蓋範圍 | 自主權限與審批門檻 |
|---|---|---|---|
| **Level A** | **Domain Scope** | 領域專用工具、數據採集器、提示詞模板與評估腳本 | **完全自主修復**：Agent 可在獨立分支與沙盒中自修代碼，通過 L2 回歸測試後自動合入主線。 |
| **Level B** | **AIA Core Scope** | 核心服務、背景 Worker、領域資料模型、自治循環執行器 | **強制人類審核**：Agent 可在沙盒分支中完成自修與測試驗證，但**必須等待人類在 Web Console / CLI 批准**方可合入。 |
| **Level C** | **Governance & Security Scope** | 治理策略 (`internal/policy/`)、審批機制、資料庫 Migration DDL、環境配置與作業系統沙盒設定 | **絕對不可自改（Immutable）**：代碼庫與資料庫完全鎖定；任何 Agent 企圖修改 Level C 將觸發 Fail-Closed 即刻拒絕並寫入高嚴重度審計日誌。 |

---

## 3. 多層級沙盒驗證管線 (Multi-Tier Validation Pipeline)

Agent 提出的任何代碼變更，必須在獨立的 Git Worktree 沙盒中按順序通過六級驗證管線（V1 ～ V6）：

```text
 [V1 語法與編譯] ➔ Go 編譯與靜態類型檢查 (go build)
       │
 [V2 單元測試]   ➔ 執行目標模組單元測試 (go test -race ./tests/unit/...)
       │
 [V3 整合測試]   ➔ 執行跨元件端到端整合測試 (tests/integration/...)
       │
 [V4 性能基準]   ➔ 檢查關鍵路徑延遲與記憶體分配無顯著回歸 (Benchmark)
       │
 [V5 規範與代碼] ➔ 靜態分析 (golangci-lint) 與文檔完整性檢查
       │
 [V6 自治驗證]   ➔ 驗證自治循環閉環行為與 L2 測試轉綠
```

**自修重試與乾淨回滾（Zero-Residual Rollback）**：
- 若管線驗證失敗，系統解析錯誤日誌執行根因分析（RCA），在預設配額內（`MaxRetries = 3`）進行修復嘗試。
- 若達到重試上限仍無法修復，觸發乾淨回滾：自動刪除暫存工作樹、修剪暫存分支，確保主倉庫 Commit SHA 100% 保持變更前狀態，零殘留垃圾檔案。

---

## 4. 資源硬閘門與緊急停機 (Resource Gates & Emergency Stop)

依據原 AIA-17 規範：

### 4.1 Token 與循環硬預算（Budget Gates）
- 每次自治循環執行前進行 Pre-flight 預算檢查（`EvaluateResourceBudget`）。
- 當日累計 Token 使用量超過上限（如 `daily_token_limit = 200,000`）或循環次數超限時，系統自動阻斷高消耗任務，降級為僅執行只讀監控或休眠。

### 4.2 受控緊急停機（Controlled Emergency Stop）
- Web Console 與 CLI 提供一鍵緊急停機功能。
- 觸發後，全域狀態置為 `EMERGENCY_STOP`，系統於 **1 秒內**凍結所有新 Task 派發，正在執行的 Context 收到 Cancel 訊號安全終止，狀態持久化標註為 `CANCELLED`。
- **重啟驗證**：解除停機必須由具備 `HUMAN` 身份的使用者透過 CLI 或 WebUI 顯式執行認證恢復，Agent 無權自行解除停機狀態。
