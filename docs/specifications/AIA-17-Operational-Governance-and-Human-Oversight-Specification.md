# AIA-17：Operational Governance & Human Oversight Specification

**文件編號：** AIA-17  
**文件名稱：** Operational Governance & Human Oversight Specification  
**版本：** 2.0  
**文件狀態：** Architecture Specification  
**適用系統：** AIA (Autonomous Intelligence Architecture)  
**依賴文件：** AIA-01 ～ AIA-16  
**核心範圍：** Operational Governance、Human Oversight、Authorization、Autonomy Boundary、Safety Control

---

# 1. 文件目的

## 1.1 設計背景

AIA 的核心目標，是讓 Agent 具備：

1. Initiative：主動性思維。
2. Continuity：時間連續性。
3. Self-Correction：自我修正。
4. Goal Evolution：目標演化。

當這些能力逐步成熟，Agent 不再只是等待使用者下達任務，而是能夠：

- 主動發現問題。
- 自行提出工作。
- 自行決定工作優先順序。
- 自行執行任務。
- 自行檢查結果。
- 自行修正策略。
- 提出新的長期目標。
- 根據實驗結果修改自身程式。
- 長期維持多個 Project 與 Experiment。

這代表系統的自主程度會逐步提高。

因此，AIA 必須建立一套與自主能力相容的治理制度。

治理的目的不是限制 Agent 所有行為，而是確保：

- Agent 知道自己的權限邊界。
- 重要決策具有明確授權。
- 高風險操作受到控制。
- 人類可以理解 Agent 的行動。
- 必要時可以暫停或終止執行。
- 系統發生異常時能夠恢復。
- Agent 不得自行解除治理規則。

## 1.2 核心設計目標

AIA-17 定義：

1. Operational Governance：日常自主運作治理。
2. Human Oversight：人類監督機制。
3. Authorization Model：授權模型。
4. Autonomy Boundary：自主權限邊界。
5. Risk Classification：風險分類。
6. Approval Workflow：核准流程。
7. Pause、Resume、Stop：暫停、恢復與停止。
8. Emergency Control：緊急控制。
9. Governance Audit：治理稽核。
10. Governance Evaluation：治理成效評估。

## 1.3 核心原則

**人類不必決定 Agent 每一步做什麼，但必須保有對重要目標、權限、資源與風險的最終控制權。**

AIA 的治理不能退化為每個 Task 都要求人類核准。

如果 Agent 每次想要觀察資料、分析問題、檢查程式碼或整理小說設定，都必須等待使用者批准，系統就無法真正展現 Initiative。

相反地，如果 Agent 可以自行修改治理政策、擴大權限、改變長期目標或動用未授權資源，也不符合安全的自主架構。

因此必須建立分層治理制度。

---

# 2. Governance Model

## 2.1 治理層級

AIA 的治理分成四個層級：

| 層級 | 名稱 | 責任 |
|---|---|---|
| G0 | System Governance | 系統核心規則與安全邊界 |
| G1 | Owner Governance | 使用者授權與重大決策 |
| G2 | Autonomy Governance | Autonomy Core 的自主決策治理 |
| G3 | Execution Governance | Agent、Worker、Task 的執行治理 |

### G0：System Governance

定義系統不可自行改變的基本規則。

例如：

- 單一使用者模式。
- Local Only。
- 不允許真實金融自動交易。
- 不允許 Agent 修改 Audit 歷史。
- 不允許 Agent 自行解除安全政策。
- 不允許 Agent 自行擴張系統權限。

G0 的修改必須透過系統擁有者明確授權與受控的系統更新流程。

### G1：Owner Governance

由系統擁有者管理：

- 長期目標。
- 資源上限。
- 重大策略。
- 重要授權。
- 高風險操作。
- Agent 自我修改核准。
- Experiment 啟動與終止。
- 系統治理政策變更。

### G2：Autonomy Governance

由 Autonomy Core 執行。

職責：

- 判斷是否需要人類核准。
- 評估行動風險。
- 檢查權限。
- 管理自主決策範圍。
- 偵測目標衝突。
- 控制自主執行程度。
- 決定 ACT、DEFER、ASK_HUMAN、NO_ACTION 或 SLEEP。

Autonomy Core 不能自行解除 G0 或 G1 的限制。

### G3：Execution Governance

由 Runtime 與相關執行元件落實。

例如：

- Task 執行權限。
- Worker 執行範圍。
- Harness 工具權限。
- Workspace 存取。
- Command 執行。
- Network 存取。
- 資源限制。
- 執行結果檢查。

Runtime 必須執行授權檢查，而不能只相信 Agent 宣稱自己有權執行。

## 2.2 Governance 與 Autonomy 的關係

Governance 不應取代 Autonomy。

兩者的責任不同：

- Autonomy 決定值得做什麼。
- Governance 決定可以做什麼。
- Authorization 決定是否獲得執行許可。
- Runtime 確保實際執行符合授權。

因此：

```text
Autonomy Core
      |
      v
Candidate Action
      |
      v
Risk Assessment
      |
      v
Governance Check
      |
      +----> Authorized
      |          |
      |          v
      |       Execute
      |
      +----> Approval Required
      |          |
      |          v
      |       Human Review
      |
      +----> Denied
                 |
                 v
              No Execute
```

---

# 3. Autonomy Boundary

## 3.1 定義

Autonomy Boundary 是 Agent 可以自主決定與執行的範圍。

AIA 必須明確區分：

1. 可以自主觀察的事情。
2. 可以自主分析的事情。
3. 可以自主提出的事情。
4. 可以自主執行的事情。
5. 必須經過人類核准的事情。
6. 永遠禁止 Agent 自行執行的事情。

## 3.2 Autonomous Operation

在既有授權範圍內，Agent 可以自主：

- 讀取允許存取的資料。
- 分析資訊。
- 建立 Claim。
- 提出 Hypothesis。
- 建立 Candidate Action。
- 建立 Task。
- 安排工作優先順序。
- 委派給其他 Agent。
- 執行允許的工具。
- 檢查執行結果。
- 建立 Learning Record。
- 提出策略改善。
- 提出 Goal Revision。
- 提出程式碼修正方案。

以上行為仍須受到既有 Policy、Permission 與 Resource Limit 約束。

## 3.3 Approval Required

下列行為原則上必須取得人類核准：

- 改變系統長期核心目標。
- 大幅改變 Experiment 的風險範圍。
- 擴大資源預算。
- 取得新的系統權限。
- 修改 AIA 核心 Runtime。
- 修改 Autonomy Core。
- 修改 Governance Policy。
- 修改安全限制。
- 進行不可逆資料操作。
- 對外發布具有重大影響的內容。
- 啟用真實金融交易能力。
- 變更使用者資料存取範圍。

具體操作是否需要核准，仍應依風險與授權政策判斷。

## 3.4 Forbidden Actions

Agent 不得自行：

1. 關閉或繞過 Governance。
2. 修改自己的權限以取得更高權限。
3. 刪除或竄改 Audit。
4. 偽造人類核准。
5. 將未授權操作包裝成一般 Task。
6. 透過其他 Agent 間接取得被禁止的權限。
7. 隱藏高風險行動。
8. 修改政策以規避現有限制。
9. 將虛擬投資轉換成真實交易。
10. 將系統的安全限制視為可自行演化的目標。

---

# 4. Risk Classification

## 4.1 設計原則

AIA 不應只依照操作名稱判斷風險。

同一種操作可能因為範圍、影響、可逆性與資源使用不同而具有不同風險。

例如：

- 修改一份臨時分析文件，風險低。
- 修改整個 Project 的正式規格，風險較高。
- 修改 AIA Runtime 的權限檢查程式，風險更高。

## 4.2 Risk Levels

建議採用 R0～R5 六級風險模型。

| 等級 | 名稱 | 說明 | 核准要求 |
|---|---|---|---|
| R0 | Read-only | 讀取與觀察 | 不需額外核准 |
| R1 | Reversible | 可逆、低影響操作 | 可依預先授權自主執行 |
| R2 | Controlled | 有明確範圍的修改 | 依授權政策執行 |
| R3 | Significant | 影響重要 Project 或策略 | 原則上需要核准 |
| R4 | Critical | 影響核心系統、重大資源或治理 | 必須核准 |
| R5 | Forbidden | 違反不可變安全限制 | 禁止執行 |

## 4.3 Risk Assessment Dimensions

風險評估至少應考慮：

- Impact：影響程度。
- Probability：發生機率。
- Reversibility：可逆程度。
- Scope：影響範圍。
- Resource Exposure：資源暴露。
- Data Sensitivity：資料敏感程度。
- External Effect：外部影響。
- Governance Impact：治理影響。

風險評估不應只依賴 Agent 自己給出的分數。

Runtime 必須能獨立執行必要的政策檢查。

## 4.4 Risk Reassessment

執行期間若發生下列情況，必須重新評估：

- 操作範圍擴大。
- 使用新的工具。
- 使用新的資料來源。
- 需要更高權限。
- 產生不可預期副作用。
- 目標發生重大變化。
- 風險超出原授權條件。

原本核准某個 Task，不代表所有後續操作都自動獲得授權。

---

# 5. Authorization Model

## 5.1 Authorization Definition

Authorization 是授予 Agent 在特定範圍內執行特定行動的許可。

授權至少包含：

- Subject：被授權的 Agent 或 Worker。
- Action：允許的操作。
- Resource：允許存取的資源。
- Scope：允許範圍。
- Constraints：限制條件。
- Validity：有效時間。
- Risk Level：風險等級。
- Approval Reference：核准紀錄。
- Revocation：撤銷條件。

## 5.2 Authorization Scope

授權可以分成：

| 類型 | 說明 |
|---|---|
| SYSTEM | 系統級授權 |
| PROJECT | 專案級授權 |
| EXPERIMENT | 實驗級授權 |
| TASK | 任務級授權 |
| RESOURCE | 資源級授權 |
| TOOL | 工具級授權 |
| TEMPORARY | 限時授權 |

## 5.3 Pre-authorized Actions

使用者可以預先授權特定類型的行動。

例如：

Coding Agent 可以在指定 Workspace 中：

- 讀取程式碼。
- 修改允許範圍內的檔案。
- 執行允許的測試。
- 建立修正分支。
- 產生 Diff。
- 提交修正建議。

這不代表 Coding Agent 可以：

- 修改 AIA 核心權限。
- 修改治理政策。
- 對外發布程式碼。
- 存取未授權的系統資料。

## 5.4 Least Privilege

授權必須遵守 Least Privilege 原則。

Agent 只能取得完成目前工作所需的最低權限。

權限不能因為委派而自動擴張。

Worker 不能因為由某個 Agent 建立，就自動取得超出原始授權範圍的權限。

## 5.5 Authorization Expiration

授權應具備有效條件。

例如：

- Task 完成後失效。
- Experiment 終止後失效。
- 指定時間到期。
- 資源條件改變後失效。
- 使用者撤銷後立即失效。

長期授權必須定期重新檢視。

---

# 6. Human Oversight

## 6.1 監督模式

人類監督分為四種模式。

| 模式 | 名稱 | 說明 |
|---|---|---|
| H0 | Observe | 只觀察，不介入 |
| H1 | Notify | Agent 自主執行，重要事件通知 |
| H2 | Approve | 指定行動必須核准 |
| H3 | Direct Control | 人類直接控制執行或停止 |

不同 Project 與 Experiment 可以採用不同監督模式。

## 6.2 H0：Observe

適用於低風險、已建立穩定基準的工作。

Agent 自主：

- 觀察。
- 分析。
- 執行。
- 評估。
- 記錄。

人類可以事後檢查完整紀錄。

## 6.3 H1：Notify

Agent 可以自主執行，但遇到特定事件時通知使用者。

例如：

- 發現重大錯誤。
- 發現新的投資風險。
- 完成重要修正。
- 發現史實衝突。
- Experiment 進度異常。
- 長期目標可能需要調整。

通知不代表需要人類立即回覆。

## 6.4 H2：Approve

特定操作必須取得人類核准。

例如：

- 修改 AIA 核心程式碼。
- 改變 Experiment 資源上限。
- 重大策略變更。
- 修改正式專案架構。
- 進行高風險或不可逆操作。

Agent 可以先完成分析、準備 Diff、整理證據與提出建議，但在核准前不得執行受限制的操作。

## 6.5 H3：Direct Control

人類直接接管。

例如：

- 暫停 Agent。
- 取消 Task。
- 停止 Experiment。
- 撤銷授權。
- 恢復系統。
- 指定新的執行方向。

H3 不代表必須人工操作所有細節，而是人類可以隨時取得控制權。

## 6.6 Oversight Without Micromanagement

AIA 的監督介面應優先呈現：

- Agent 目前目標。
- 正在進行的 Experiment。
- 近期自主決策。
- 重大風險。
- 等待核准的操作。
- 重要結果。
- 未解決問題。
- 需要人類判斷的事項。

不應要求使用者閱讀每一筆內部工具呼叫，才能知道系統是否正常。

詳細執行紀錄仍應保留供追查。

---

# 7. Approval Workflow

## 7.1 Approval Lifecycle

```text
ACTION PROPOSED
      |
      v
RISK ASSESSMENT
      |
      v
POLICY CHECK
      |
      +----> AUTO AUTHORIZED
      |          |
      |          v
      |        EXECUTE
      |
      +----> APPROVAL REQUIRED
      |          |
      |          v
      |      PENDING HUMAN
      |          |
      |          +----> APPROVED
      |          |         |
      |          |         v
      |          |       EXECUTE
      |          |
      |          +----> REJECTED
      |          |
      |          +----> EXPIRED
      |          |
      |          +----> REVISED
      |
      +----> DENIED
```

## 7.2 Approval Request

每筆核准請求至少包含：

- Approval ID。
- Requesting Agent。
- Related Task。
- Related Experiment。
- Proposed Action。
- Reason。
- Expected Benefit。
- Risk Assessment。
- Evidence References。
- Resource Impact。
- Reversibility。
- Alternatives。
- Expiration Time。

## 7.3 Approval Decision

核准結果：

- APPROVED。
- REJECTED。
- NEEDS_REVISION。
- EXPIRED。
- REVOKED。

核准紀錄必須保留。

## 7.4 Approval Scope Binding

核准必須綁定具體操作範圍。

例如：

使用者核准「修改 Investment Agent 的策略分析模組」，不代表核准修改 AIA 核心 Runtime。

若執行過程中操作內容超出原核准範圍，必須重新取得授權。

## 7.5 No Approval by Silence

使用者沒有回覆，不代表核准。

Approval Request 逾期時：

- 不執行受限制操作。
- Task 進入等待狀態。
- Agent 可以繼續進行不需要該操作的其他工作。
- 必要時重新提出核准請求。

---

# 8. Goal Governance

## 8.1 Goal Ownership

AIA 必須區分目標提出者與目標核准者。

| 目標類型 | Agent 可以做什麼 |
|---|---|
| User Goal | 執行、拆解、評估 |
| Agent Proposed Goal | 提出、分析、等待核准 |
| Operational Goal | 在授權範圍內自行建立 |
| Experiment Goal | 依 Experiment 授權管理 |
| System Goal | 不得自行修改 |

## 8.2 Goal Persistence

Agent 必須能維持長期目標。

短期結果不應讓 Agent 任意放棄使用者設定的核心目標。

例如：

投資 Experiment 的目標是最大化虛擬投資組合總資產。

Agent 可以改變：

- 投資策略。
- 持股配置。
- 現金比例。
- 研究標的。
- 觀察頻率。
- 評估方法。

但不能自行將目標改成「降低波動」或「追求穩定配息」，並取代原始總資產最大化目標。

## 8.3 Goal Optimization

Agent 可以自行優化達成目標的方法。

例如：

- 改善資料來源。
- 改善分析模型。
- 改善執行效率。
- 改善風險估計。
- 改善預測校準。

只要不改變目標本身或突破授權邊界，通常可以自主進行。

## 8.4 Goal Revision

Agent 可以提出：

- 目標調整。
- 目標拆解。
- 目標優先順序調整。
- 目標終止。
- 新增長期目標。

重大 Goal Revision 必須取得人類核准。

## 8.5 Goal Conflict

當不同目標衝突時：

1. 找出衝突目標。
2. 評估優先順序。
3. 檢查使用者明確設定的目標。
4. 檢查資源與時間限制。
5. 提出解決方案。
6. 必要時詢問使用者。

Agent 不得為了方便執行而私自改寫使用者的核心目標。

---

# 9. Resource Governance

## 9.1 Resource Categories

AIA 必須管理：

- CPU。
- Memory。
- Storage。
- Network。
- Model Usage。
- Token Budget。
- Worker Capacity。
- Task Concurrency。
- External Service Usage。
- Financial Simulation Capital。

## 9.2 Resource Limits

每個 Project 或 Experiment 可以具有：

- Maximum Budget。
- Maximum Runtime。
- Maximum Concurrent Workers。
- Maximum Storage。
- Maximum API Usage。
- Maximum Retry Count。

具體限制由系統設定與使用者授權決定。

## 9.3 Resource Optimization

Agent 可以在授權範圍內自行調整資源使用方式。

例如：

- 減少不必要的模型呼叫。
- 合併重複工作。
- 調整 Worker 數量。
- 延後低優先度工作。
- 改變資料擷取頻率。

但是不能自行突破資源上限。

## 9.4 Resource Expansion

擴大預算、取得付費服務或增加重要系統資源，必須經過授權。

Agent 可以提出理由與預期效益，但不能自行取得新的付費承諾。

## 9.5 Financial Simulation Boundary

Investment Experiment 的 NT$100,000 是虛擬資金。

Agent 可以依照實驗規格調整虛擬持股與現金。

不得將虛擬資金視為真實資產，也不得自行連接真實證券帳戶執行交易。

任何未來真實金融交易能力都屬於新的系統風險範圍，不能由現有投資 Experiment 自動取得。

---

# 10. Self-Modification Governance

## 10.1 修改對象分類

AIA 必須區分三種修改：

| 類型 | 範圍 | 治理要求 |
|---|---|---|
| D1 | Domain Project 程式碼與策略 | 依專案授權 |
| D2 | AIA Runtime、Autonomy Core | 嚴格審查與核准 |
| D3 | Governance、Policy、Permission、Audit | 不得由 Agent 自行修改 |

## 10.2 Domain Self-Modification

Agent 可以在已授權的專案範圍內：

- 分析程式碼。
- 提出修正。
- 建立隔離分支。
- 修改程式碼。
- 執行測試。
- 比較結果。
- 提出合併建議。

是否允許自動合併或部署，必須由該 Project 的授權政策決定。

## 10.3 Core Self-Modification

修改 AIA 核心 Runtime 或 Autonomy Core 時：

1. 建立 Modification Proposal。
2. 保存原始版本。
3. 建立隔離分支。
4. 產生 Diff。
5. 執行測試。
6. 執行安全檢查。
7. 評估回歸風險。
8. 取得人類核准。
9. 執行受控更新。
10. 驗證更新結果。
11. 保留 Rollback 能力。

Agent 不得直接修改正在運作的核心程式碼並立即取代現有版本。

## 10.4 Governance Self-Modification

Agent 不得自行修改：

- Governance Policy。
- Authorization Rules。
- Permission Boundary。
- Audit Rules。
- Human Approval Requirements。
- Safety Constraints。

這些設定只能由系統擁有者透過受控管理流程變更。

## 10.5 Self-Modification Evidence

每次修改必須保存：

- 修改原因。
- 相關 Experiment。
- 原始版本。
- 新版本。
- Diff。
- 測試結果。
- 評估結果。
- 核准紀錄。
- 更新時間。
- 回滾方式。

詳細修改流程由 AIA-14 規範。

---

# 11. Pause, Resume & Stop

## 11.1 Pause

Pause 是暫時停止某個執行範圍。

可以作用於：

- Task。
- Worker。
- Agent。
- Project。
- Experiment。
- 整個 AIA Runtime。

Pause 不等於刪除狀態。

暫停時必須保存：

- 當前狀態。
- 已完成工作。
- 未完成工作。
- 正在等待的操作。
- 資源使用狀態。
- 恢復所需 Context。

## 11.2 Resume

Resume 是從暫停狀態恢復執行。

恢復前應檢查：

- 原始授權是否仍有效。
- 目標是否仍有效。
- 相關資料是否過時。
- 執行環境是否改變。
- 是否存在新的風險。
- 是否需要重新評估。

Resume 不代表無條件繼續執行原本所有操作。

## 11.3 Stop

Stop 是終止指定執行單位。

停止時：

- 不再建立新的子任務。
- 取消尚未開始的工作。
- 依安全方式停止進行中的工作。
- 保存必要狀態。
- 記錄停止原因。
- 保留 Audit。

## 11.4 Termination Scope

停止某個 Worker 不應自動停止整個 Agent。

停止某個 Task 不應自動停止整個 Project。

停止某個 Experiment 不應自動刪除相關 Memory。

系統必須明確區分停止範圍。

## 11.5 Emergency Stop

Emergency Stop 是緊急停止機制。

啟動後：

1. 阻止新的高風險操作。
2. 撤銷相關執行授權。
3. 停止可安全停止的 Worker。
4. 暫停相關 Task。
5. 保留必要狀態。
6. 寫入 Audit。
7. 通知使用者。

Emergency Stop 不應依賴 Agent 自己同意停止。

應由 Runtime 或更高層級控制機制執行。

## 11.6 Recovery

恢復時必須檢查：

- 系統完整性。
- Audit 完整性。
- Task 狀態。
- Worker 狀態。
- Authorization 狀態。
- Pending Approval。
- Project 與 Experiment 狀態。

不能只因為 Process 重新啟動，就認定所有工作可以直接恢復。

---

# 12. Operational Monitoring

## 12.1 Monitoring Scope

AIA 應持續監控：

- Runtime Health。
- Agent Health。
- Worker Health。
- Task Progress。
- Experiment Progress。
- Resource Usage。
- Policy Violations。
- Approval Queue。
- Error Rate。
- Recovery Events。
- Integrity Alerts。

## 12.2 Operational Events

重要事件至少包括：

- AGENT_STARTED。
- AGENT_STOPPED。
- TASK_CREATED。
- TASK_COMPLETED。
- TASK_FAILED。
- WORKER_STARTED。
- WORKER_STOPPED。
- EXPERIMENT_STARTED。
- EXPERIMENT_PAUSED。
- EXPERIMENT_COMPLETED。
- APPROVAL_REQUESTED。
- APPROVAL_GRANTED。
- APPROVAL_REJECTED。
- AUTHORIZATION_REVOKED。
- POLICY_VIOLATION。
- EMERGENCY_STOP。
- RECOVERY_STARTED。
- RECOVERY_COMPLETED。

## 12.3 Alert Classification

| 等級 | 說明 |
|---|---|
| INFO | 一般事件 |
| NOTICE | 值得注意的變化 |
| WARNING | 潛在問題 |
| ERROR | 工作或元件發生錯誤 |
| CRITICAL | 重大系統風險 |
| EMERGENCY | 必須立即停止或介入 |

## 12.4 Monitoring Is Not Micromanagement

Monitoring 的目的，是讓人類了解系統狀態，而不是要求人類持續操作。

使用者應能快速掌握：

- AIA 是否正常運作。
- Agent 正在做什麼。
- 哪些工作是自主發現的。
- 哪些決策正在等待核准。
- 是否有重大風險。
- 最近的學習與修正成果。

---

# 13. Governance Audit

## 13.1 Audit Requirements

所有重要治理事件必須保存：

- Event ID。
- Timestamp。
- Actor。
- Action。
- Resource。
- Scope。
- Authorization Reference。
- Approval Reference。
- Policy Version。
- Result。
- Reason。
- Related Task。
- Related Experiment。

## 13.2 Audit Integrity

Audit 必須具備：

- Append-only。
- 可追溯。
- 可查詢。
- 可關聯。
- 可驗證完整性。

Agent 不得修改或刪除歷史 Audit。

## 13.3 Decision Audit

自主決策至少應保留：

1. 決策前狀態。
2. Candidate Actions。
3. Risk Assessment。
4. Policy Check。
5. Authorization Result。
6. 最終 Decision。
7. 執行結果。
8. 後續 Evaluation。

## 13.4 Governance Violation

違反治理政策時：

- 阻止未授權操作。
- 建立 Audit。
- 標記相關 Task。
- 必要時暫停 Agent。
- 評估是否影響其他 Project。
- 通知使用者。
- 產生修正建議。

不能只記錄錯誤而繼續執行高風險操作。

---

# 14. Governance Across Domains

## 14.1 Investment Agent

投資實驗目前採用 NT$100,000 虛擬資金。

治理要求：

- Agent 可以自主選擇投資標的。
- Agent 可以自主增減持股。
- Agent 可以自主決定保留現金。
- 每個證券營業日應觀察市場並作出判斷。
- 每月 1 日由使用者檢查總資產。
- 核心目標是最大化投資組合總資產。
- 不得將虛擬交易轉換成真實交易。
- 不得自行擴大實驗資金。
- 不得修改使用者設定的核心投資目標。
- 所有交易與決策必須可追溯。

每天觀察不等於每天交易。

投資 Agent 可以在風險、價格與資訊條件不符合預期時，選擇不交易。

## 14.2 Coding Agent

Coding Agent 可以在授權 Project 中自主：

- 閱讀程式碼。
- 找出問題。
- 建立修正方案。
- 修改允許範圍內的程式碼。
- 執行測試。
- 比較版本。
- 建立技術債清單。
- 追蹤長期品質。

但必須遵守：

- Workspace 限制。
- Harness 工具權限。
- Policy。
- 自我修改等級。
- 核心程式碼核准流程。
- Diff 與測試紀錄。

## 14.3 Writing Agent

Writing Agent 可以自主：

- 維護長期大綱。
- 檢查人物關係。
- 檢查年表。
- 比對歷史資料。
- 發現跨卷矛盾。
- 提出修訂。
- 更新工作進度。
- 建立新的研究任務。

但必須保留：

- 使用者確認的設定。
- 史實與推演的區別。
- 原始版本。
- 修訂理由。
- 歷史來源。
- 章節與人物關聯。

不能因為 Agent 認為某種敘事更順暢，就擅自覆蓋使用者已確認的重要設定。

## 14.4 Cross-Domain Governance

跨 Agent 協作時：

- 委派範圍必須明確。
- 權限不能自動繼承。
- 各 Agent 保留自身責任。
- 共享資料保留來源。
- 高風險操作依原始授權判斷。
- 重大目標衝突由 Autonomy Core 協調。
- 必要時交由使用者裁決。

---

# 15. Go Implementation Architecture

## 15.1 設計原則

AIA-17 應整合既有 Policy、Approval、Audit、Task、Worker、Agent Registry 與 Runtime。

不應另建一套彼此獨立的治理系統。

既有治理元件應逐步補足本文件定義的能力。

## 15.2 Governance Components

```text
AIA Runtime
    |
    +-- Governance Engine
    |      |
    |      +-- Risk Assessment
    |      +-- Policy Evaluator
    |      +-- Authorization Manager
    |      +-- Approval Manager
    |      +-- Goal Governance
    |      +-- Resource Governance
    |
    +-- Human Oversight
    |      |
    |      +-- Notification
    |      +-- Approval Queue
    |      +-- Pause / Resume
    |      +-- Emergency Control
    |
    +-- Audit Service
    |
    +-- Operational Monitor
```

## 15.3 Component Responsibilities

### Governance Engine

統一處理：

- 行動風險。
- 政策檢查。
- 授權判斷。
- 核准要求。
- 執行限制。

### Risk Assessment

評估操作的影響、可逆性、資源與風險。

### Policy Evaluator

檢查行動是否符合既有 Policy。

Policy Evaluator 必須能在執行層被呼叫，不能只依賴 Agent 自行判斷。

### Authorization Manager

管理授權範圍、有效期間與撤銷。

### Approval Manager

管理人類核准請求與結果。

### Human Oversight Service

提供：

- 狀態通知。
- 核准介面。
- 暫停與恢復。
- 緊急停止。
- 人類接管。

### Emergency Control

提供獨立於一般 Agent 決策流程的緊急控制能力。

### Audit Service

記錄所有重要治理事件。

### Operational Monitor

觀察 Runtime、Agent、Worker、Task 與 Experiment 狀態。

## 15.4 Runtime Enforcement

治理不能只存在於 Autonomy Core 的推理結果中。

實際執行時必須再次檢查：

```text
Agent Decision
      |
      v
Task Authorization
      |
      v
Runtime Policy Check
      |
      v
Harness Permission Check
      |
      v
Tool Execution
      |
      v
Result Verification
      |
      v
Audit
```

即使 Agent 認為某項操作合理，只要 Runtime 判定未授權，就不得執行。

這也代表 AIA 的治理需要與 agy Harness 的工具權限管理相互配合，而不是取代 Harness。

AIA 負責自身任務、專案與實驗層級的授權治理；agy 負責底層工具與執行環境的權限控制。

兩者應採取縱深防禦，而不是互相假設對方已經完成所有檢查。

## 15.5 Suggested Interfaces

```go
type GovernanceRequest struct {
    AgentID      string
    TaskID       string
    ExperimentID string

    Action       string
    Resource     string
    Scope        string

    RiskLevel    string
    Context      map[string]any
}

type GovernanceDecision struct {
    Allowed          bool
    ApprovalRequired bool

    RiskLevel        string
    Reason           string

    PolicyReferences []string
    AuthorizationID  string
    ApprovalID       string
}

type ApprovalRequest struct {
    ID          string
    RequestedBy string

    Action      string
    Scope       string
    Reason      string

    RiskLevel   string
    EvidenceIDs []string

    ExpiresAt   time.Time
    Status      string
}

type Authorization struct {
    ID        string
    Subject   string
    Action    string
    Resource  string
    Scope     string

    GrantedAt time.Time
    ExpiresAt *time.Time

    Constraints map[string]any
    RevokedAt   *time.Time
}

type OversightAction struct {
    TargetType string
    TargetID   string

    Action     string
    Reason     string
    RequestedBy string
}
```

以上為概念模型，實際介面應配合現有 Go Runtime 的資料結構與命名。

## 15.6 Existing Worker Integration

現有 Worker 是執行單位。

治理系統應能：

- 檢查 Worker 所屬 Agent。
- 檢查 Worker 執行 Task。
- 確認授權範圍。
- 限制可用工具。
- 停止或暫停 Worker。
- 保存執行結果。
- 記錄治理事件。

Worker 不應自行成為授權來源。

---

# 16. Metrics & Evaluation

治理能力也必須可以量測。

## 16.1 Unauthorized Action Rate

未授權操作發生率。

目標是阻止未授權操作真正執行，而不是只在事後發現。

## 16.2 Approval Precision

核准要求是否合理。

若低風險操作大量要求人工核准，代表治理可能過度保守。

## 16.3 Approval Coverage

需要核准的操作是否都經過核准。

## 16.4 Policy Violation Detection

政策違規是否能被正確辨識。

## 16.5 Emergency Stop Latency

緊急停止從觸發到實際阻止新操作所需時間。

## 16.6 Recovery Success Rate

暫停、停止或異常後能否正確恢復。

## 16.7 Human Intervention Rate

人類介入頻率。

介入頻率過高可能代表：

- Agent 自主能力不足。
- 授權範圍過窄。
- 治理規則過度保守。
- 任務拆解不合理。

但介入頻率低也不代表治理一定良好。

## 16.8 Governance Transparency

重要自主決策是否具有：

- 清楚理由。
- 風險說明。
- 證據關聯。
- 授權紀錄。
- 執行結果。

## 16.9 Autonomy Without Governance Bypass

系統必須能證明：

Agent 可以在授權範圍內自主完成工作，同時不能透過自我修改、委派或其他間接方式繞過治理限制。

---

# 17. Implementation Phases

## Phase G0：基本治理整合

目標：

- 統一 Policy Check。
- 整合既有 Authorization。
- 整合既有 Approval。
- 整合 Audit。
- 建立統一 Governance Decision。

驗收：

- Task 執行前能確認授權。
- 未授權操作無法執行。
- 核准與拒絕都留下紀錄。

## Phase G1：Risk-based Governance

目標：

- R0～R5 分級。
- Risk Assessment。
- Approval Policy。
- Pre-authorized Actions。

驗收：

- 低風險工作可以自主執行。
- 高風險工作能要求核准。
- 禁止行動無法透過一般核准流程繞過。

## Phase G2：Human Oversight

目標：

- H0～H3 監督模式。
- Notification。
- Approval Queue。
- Pause / Resume / Stop。

驗收：

- 使用者可以查看重要自主決策。
- 使用者可以核准或拒絕。
- 使用者可以暫停指定執行範圍。
- 恢復前會重新檢查授權。

## Phase G3：Resource & Goal Governance

目標：

- Goal Ownership。
- Goal Revision Approval。
- Resource Limits。
- Resource Expansion Approval。

驗收：

- Agent 可以自主優化方法。
- Agent 不能自行改變核心目標。
- Agent 不能自行突破資源上限。

## Phase G4：Self-Modification Governance

目標：

- Domain Self-Modification。
- Core Self-Modification Approval。
- Governance Modification Protection。
- Rollback。

驗收：

- Domain Project 修改符合授權。
- 核心修改必須核准。
- Governance 修改受到獨立保護。

## Phase G5：Emergency & Recovery

目標：

- Emergency Stop。
- Authorization Revocation。
- Runtime Recovery。
- Governance Integrity Monitoring。

驗收：

- 緊急停止不依賴 Agent 同意。
- 停止後能保留必要狀態。
- 恢復前重新檢查權限與政策。
- Audit 不會因為停止或恢復而遺失。

---

# 18. Acceptance Criteria

AIA-17 完成實作後，至少應符合：

## 18.1 Governance

- [ ] Governance 層級明確。
- [ ] Autonomy 與 Governance 職責分離。
- [ ] Runtime 執行前能檢查授權。
- [ ] Harness 權限與 AIA Policy 相互配合。
- [ ] 禁止操作不能被 Agent 自行解除。

## 18.2 Authorization

- [ ] 授權具有 Subject、Action、Resource、Scope。
- [ ] 授權具有有效條件。
- [ ] 支援授權撤銷。
- [ ] 委派不會擴大權限。
- [ ] 高風險操作具有明確核准流程。

## 18.3 Human Oversight

- [ ] 支援 Observe、Notify、Approve、Direct Control。
- [ ] 使用者可以查看重要自主決策。
- [ ] 使用者可以核准或拒絕。
- [ ] 使用者可以暫停或停止工作。
- [ ] 使用者可以緊急停止系統。

## 18.4 Goal Governance

- [ ] Agent 可以提出新目標。
- [ ] Agent 可以優化執行方法。
- [ ] Agent 不能自行覆蓋使用者核心目標。
- [ ] 重大 Goal Revision 需要核准。

## 18.5 Resource Governance

- [ ] 每個 Experiment 可以設定資源上限。
- [ ] Agent 不能自行突破上限。
- [ ] 資源擴張需要授權。
- [ ] 虛擬投資資金不能轉換成真實交易能力。

## 18.6 Self-Modification

- [ ] Domain 修改有明確範圍。
- [ ] 核心修改需要核准。
- [ ] Governance 修改受到保護。
- [ ] 修改具有 Diff、測試與回滾紀錄。

## 18.7 Recovery & Audit

- [ ] Pause、Resume、Stop 狀態可追溯。
- [ ] Emergency Stop 不依賴 Agent 自己執行。
- [ ] 恢復前重新檢查授權。
- [ ] 治理事件不可無痕修改。

---

# 19. Final Design Principles

AIA-17 確立以下最終原則：

1. **自主不等於無限制。** Agent 可以自主決策，但只能在授權範圍內行動。
2. **治理不等於人工操作。** 人類不必決定每一個 Task 的細節。
3. **目標與方法分離。** Agent 可以優化方法，但不能任意覆蓋使用者的核心目標。
4. **授權必須具體。** 核准必須綁定行動、資源與範圍。
5. **權限不可繼承擴張。** Agent 委派給 Worker 或其他 Agent 時，不能取得更高權限。
6. **執行必須再次驗證。** 不能只相信 Agent 自己認為操作合理。
7. **自我修改必須受控。** Agent 可以提出並驗證改善，但不能自行解除治理。
8. **人類必須保有控制權。** 暫停、撤銷與緊急停止不能依賴 Agent 同意。
9. **歷史必須可追溯。** 核准、拒絕、修正與停止都必須留下紀錄。
10. **治理應隨自主能力成熟而調整。** 在不突破安全邊界的前提下，逐步減少不必要的人工介入。

AIA 的目的不是建立一個每一步都等待人類指令的自動化工具，也不是建立一個不受限制的黑箱 Agent。

真正的目標是：

**讓 Agent 能夠在長期、持續的運作中，自主發現問題、提出目標、執行工作、檢驗結果並修正自己；同時讓人類始終保有對核心目標、重大資源與系統安全的控制權。**
