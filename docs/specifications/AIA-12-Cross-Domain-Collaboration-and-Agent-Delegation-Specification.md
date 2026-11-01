# AIA-12
# Cross-Domain Collaboration & Agent Delegation Specification
## 跨領域協作與 Agent 任務委派規格書

**文件版本：** v2.0  
**文件代號：** AIA-12  
**文件性質：** Autonomy Core 子系統規格  
**適用範圍：** AIA Runtime、Autonomy Core、Continuity、Domain Agents、Workers  
**主要依據：** AIA-01、AIA-02、AIA-08、AIA-09、AIA-10、AIA-11

---

# 1. 文件目的

本文件定義 AIA 如何管理多個 Domain Agent 之間的協作、任務委派、知識交換、結果整合與責任追蹤。

AIA 的不同 Agent 具有不同的專業能力，但它們並非彼此獨立的 AI 系統。

它們共同服務於同一個 AIA 平台，必須遵循共同的：

- Mission。
- Strategic Goal。
- Project Continuity。
- Policy。
- Audit。
- Autonomy Core。

AIA 應允許 Agent 在必要時向其他 Agent 尋求協助，或將部分工作委派給其他 Agent。

但跨領域協作必須避免：

1. 目標不一致。
2. 工作重複。
3. 責任不明。
4. 錯誤資訊傳播。
5. 權限不當擴張。
6. 不同 Agent 之間產生循環委派。
7. Agent 將其他 Agent 的結論直接當成事實。
8. 協作結果無法追溯。

## 1.1 核心定位

AIA 的 Agent 協作不是單純的多 Agent 對話。

真正的協作必須具有：

- 明確工作目的。
- 明確任務負責者。
- 明確交付成果。
- 明確輸入與輸出。
- 明確驗證方式。
- 明確責任歸屬。
- 可追溯的協作歷史。

Agent 可以互相協助，但不能因此將最終責任推給其他 Agent。

# 2. 核心設計原則

## 2.1 Autonomy Core 是共同協調層

所有 Domain Agent 都使用共同的 Autonomy Core。

```text
                    AIA Runtime
                         │
                    Autonomy Core
                         │
          ┌──────────────┼──────────────┐
          │              │              │
      Investment       Coding         Writing
        Agent           Agent          Agent
          │              │              │
       Workers         Workers        Workers
```

Autonomy Core 負責：

- 維持共同目標。
- 管理跨 Project 工作關係。
- 評估是否需要委派。
- 管理協作優先順序。
- 檢查授權。
- 保存協作歷史。
- 驗證協作結果是否符合原始目標。

Domain Agent 負責各自領域的專業判斷與執行。

## 2.2 Agent 不等於 Worker

Agent 是具有領域知識、目標與工作能力的執行單位。

Worker 是承載具體工作的執行單位。

一個 Agent 可以有多個 Worker。

一個 Worker 也可能承載不同類型的工作，但必須遵守明確的任務上下文與權限。

不能將 Worker 數量視為 Agent 智慧程度的指標。

## 2.3 協作不等於權限共享

Agent 之間可以共享必要資訊，但不代表共享所有權限。

例如：

Coding Agent 可以請 Writing Agent 協助整理文件，但不因此取得 Writing Project 的所有修改權限。

Investment Agent 可以請其他 Agent 協助整理資料，但不能因此取得額外的金融操作授權。

## 2.4 委派不等於責任轉移

委派工作後：

- 委派者仍須追蹤任務。
- 受委派者負責交付指定成果。
- 協調層負責管理工作關係。
- 最終驗證者負責確認成果符合原始目標。

委派者不能只因為工作已交出去，就視為自己已完成責任。

# 3. Collaboration Model

AIA 的跨 Agent 協作分成五種主要形式。

## 3.1 Consultation

向其他 Agent 諮詢專業意見。

例如：

Investment Agent 請 Coding Agent 協助分析某個資料處理程式的技術限制。

Consultation 不會自動建立正式執行任務。

## 3.2 Delegation

將明確工作委派給其他 Agent。

例如：

Writing Agent 請具備歷史資料處理能力的 Agent 協助整理史料。

受委派者必須有明確交付內容與期限或完成條件。

## 3.3 Collaboration

兩個以上 Agent 共同完成同一項工作。

例如：

Coding Agent 負責程式實作，另一個 Agent 負責檢查文件與需求一致性。

## 3.4 Review

由其他 Agent 檢查既有成果。

例如：

- Coding Agent 檢查程式修改。
- Writing Agent 檢查歷史年表。
- Investment Agent 檢查投資資料與計算結果。

Review 必須保留審查依據與結論。

## 3.5 Knowledge Exchange

Agent 分享已整理的知識與經驗。

例如：

Coding Agent 分享經驗證的資料驗證方法，其他 Agent 可以評估是否適用。

Knowledge Exchange 不代表知識可以無條件直接套用。

# 4. Agent Registry & Capability Discovery

Autonomy Core 必須知道目前有哪些 Agent，以及它們能處理哪些工作。

## 4.1 Agent Capability

每個 Agent 應提供能力描述：

- Agent ID。
- Domain。
- Description。
- Supported Tasks。
- Input Requirements。
- Output Types。
- Required Tools。
- Required Permissions。
- Resource Requirements。
- Limitations。
- Version。

## 4.2 Capability Matching

當 Agent 需要協助時，Autonomy Core 應根據工作需求尋找適合的 Agent。

評估條件：

1. 專業能力是否符合。
2. 是否具有必要工具。
3. 是否具有必要權限。
4. 是否有可用資源。
5. 是否存在利益衝突。
6. 是否能在期限內完成。
7. 是否有適當的驗證能力。

不能只根據 Agent 名稱判斷適用性。

## 4.3 Capability 不等於保證能力

Agent 宣告具備某項能力，不代表它一定能完成工作。

應透過歷史交付品質、驗證結果及錯誤紀錄評估可靠程度。

# 5. Delegation Model

## 5.1 Delegation Lifecycle

```text
Need Discovery
      ↓
Delegation Candidate
      ↓
Capability Matching
      ↓
Cost / Risk Evaluation
      ↓
Authorization
      ↓
Task Assignment
      ↓
Execution
      ↓
Result Submission
      ↓
Review / Verification
      ↓
Integration
      ↓
Closure
```

## 5.2 Delegation Request

每次正式委派都必須建立 Delegation Request。

必要欄位：

| 欄位 | 說明 |
|---|---|
| Delegation ID | 委派識別碼 |
| Parent Task ID | 上層任務 |
| Requesting Agent | 委派者 |
| Assigned Agent | 受委派者 |
| Project ID | 所屬 Project |
| Goal ID | 相關目標 |
| Description | 工作內容 |
| Input References | 輸入資料 |
| Expected Output | 預期交付成果 |
| Acceptance Criteria | 驗收條件 |
| Deadline | 預定完成時間 |
| Resource Budget | 資源預算 |
| Permission Scope | 授權範圍 |
| Status | 委派狀態 |

## 5.3 Delegation Scope

委派必須限制在明確範圍。

例如：

Writing Agent 可以委派：

「整理 1840 至 1860 年廣州、上海、天津的重要外交事件，提供來源與時間表。」

但不能只委派：

「幫我處理清末歷史。」

任務越模糊，越容易造成成果不符合原始需求。

## 5.4 Output Contract

受委派 Agent 必須依照約定格式交付成果。

交付內容應包含：

- Result。
- Evidence。
- Assumptions。
- Limitations。
- Confidence。
- Unresolved Issues。
- References。
- Verification Status。

不得只回傳一段沒有依據的結論。

# 6. Delegation Decision

Autonomy Core 不應將所有工作都委派出去。

委派本身也有成本。

## 6.1 委派適用情境

- 需要其他領域專業能力。
- 工作可以獨立拆分。
- 需要獨立審查。
- 可以平行處理。
- 委派後能明確驗收。

## 6.2 不適合委派的情境

- 工作範圍無法清楚界定。
- 受委派者缺少必要資訊。
- 委派成本高於自行處理。
- 涉及不允許共享的敏感資料。
- 工作必須依賴委派者尚未完成的判斷。
- 委派會造成重複工作或循環依賴。

## 6.3 Delegation Value

可使用下列概念模型：

\[
DV = B - C - R
\]

其中：

- \(B\)：委派帶來的預期效益。
- \(C\)：委派與協調成本。
- \(R\)：錯誤傳播及整合風險。

只有當委派具有合理價值時，才應建立正式任務。

# 7. Collaboration Orchestration

## 7.1 協作模式

AIA 應支援以下模式。

### Sequential

依序完成工作。

```text
Agent A
   ↓
Agent B
   ↓
Agent C
   ↓
Final Review
```

適合具有前後依賴的工作。

### Parallel

多個 Agent 同時處理不同工作。

```text
             ┌── Agent A
Parent Task ─┼── Agent B
             └── Agent C
                    ↓
                Integration
```

適合可以獨立完成的工作。

### Review Loop

由不同 Agent 反覆檢查成果。

```text
Agent A
   ↓
Agent B Review
   ↓
Correction
   ↓
Agent B Re-review
```

必須設定重試次數與停止條件，避免無限循環。

### Hybrid

混合 Sequential、Parallel 與 Review。

大型 Project 通常需要這種模式。

## 7.2 協作狀態

建議狀態：

- CREATED。
- PENDING。
- ASSIGNED。
- RUNNING。
- WAITING_FOR_INPUT。
- WAITING_FOR_REVIEW。
- COMPLETED。
- FAILED。
- CANCELLED。
- BLOCKED。

狀態轉換必須有紀錄。

## 7.3 Partial Result

受委派 Agent 可以提交部分成果。

但部分成果不能直接被視為任務完成。

Autonomy Core 應能判斷：

- 是否有足夠成果供其他工作使用。
- 是否仍缺少關鍵內容。
- 是否可以提前整合。
- 是否需要等待完整交付。

# 8. Multi-Agent Conflict Resolution

多個 Agent 可能對同一問題提出不同結論。

AIA 不應強迫所有 Agent 得出相同答案。

## 8.1 Conflict Types

| 類型 | 說明 |
|---|---|
| Evidence Conflict | 證據不一致 |
| Reasoning Conflict | 推理方式不同 |
| Goal Conflict | 目標優先順序不同 |
| Strategy Conflict | 執行方法不同 |
| Resource Conflict | 資源競爭 |
| Ownership Conflict | 工作責任不清 |

## 8.2 Evidence Conflict

當證據不一致時：

1. 保留所有來源。
2. 比較來源可信度。
3. 檢查時間與適用範圍。
4. 辨識是否存在不同定義。
5. 必要時尋找第三方證據。
6. 無法解決時標記為未確定。

不能只採用多數 Agent 的意見。

## 8.3 Reasoning Conflict

不同 Agent 可能根據相同證據得出不同推論。

應比較：

- 使用的假設。
- 推理步驟。
- 適用條件。
- 信心程度。
- 反例。
- 可驗證的預測。

## 8.4 Goal Conflict

當 Agent 的工作目標衝突時，由 Autonomy Core 依 AIA-09 的 Goal Hierarchy 與 Policy 處理。

Domain Agent 不得自行提高自身目標優先級。

## 8.5 Unresolved Conflict

無法合理解決時：

- 保存雙方結論。
- 記錄爭議原因。
- 標示未解決問題。
- 視重要程度提交人類判斷。

不應為了產生單一答案而隱藏不確定性。

# 9. Shared Knowledge & Evidence

## 9.1 Knowledge Sharing

Agent 可以分享：

- 已驗證知識。
- 工作程序。
- 錯誤模式。
- 評估方法。
- 外部資料來源。
- 經驗教訓。

但必須保留來源、版本與適用範圍。

## 9.2 Evidence Provenance

所有跨 Agent 使用的重要資訊都應保留來源。

至少記錄：

- 原始來源。
- 提供 Agent。
- 取得時間。
- 原始資料版本。
- 驗證狀態。
- 信心程度。
- 適用範圍。

Agent 的回答不是天然可信的證據。

接收者必須依工作重要程度自行驗證。

## 9.3 Knowledge Promotion

其他 Agent 提供的資訊，不應直接成為全域知識。

建議分級：

| 等級 | 說明 |
|---|---|
| PROPOSED | 尚未驗證 |
| REVIEWED | 已初步檢查 |
| VERIFIED | 已有充分證據 |
| DOMAIN_APPROVED | 已由領域責任者確認 |
| DEPRECATED | 已不再適用 |

不同領域的專業知識，應由相應 Domain Agent 或人類確認。

# 10. Responsibility & Ownership

多 Agent 協作最重要的是責任清楚。

## 10.1 Task Owner

每個正式 Task 必須有一個主要負責者。

可以有多個協作者，但不得沒有 Owner。

## 10.2 Delegator Responsibility

委派者負責：

- 工作定義。
- 選擇受委派者。
- 提供必要上下文。
- 檢查交付成果。
- 整合結果。
- 確認是否達成原始目標。

## 10.3 Delegate Responsibility

受委派者負責：

- 在指定範圍內執行。
- 遵守授權。
- 提供可驗證成果。
- 說明限制。
- 回報阻塞或失敗。
- 保存執行紀錄。

## 10.4 Final Accountability

協作成果的最終責任依原始 Project 與 Task 的 Owner 決定。

委派不會自動轉移原始目標的責任。

# 11. Delegation Security

## 11.1 Least Privilege

受委派 Agent 只能取得完成任務所需的最小權限。

## 11.2 Context Isolation

委派時只提供必要上下文。

不應因為一項任務，就將整個 Project 的所有記憶與資料交給其他 Agent。

## 11.3 Untrusted Output

其他 Agent 的輸出應視為外部輸入。

接收者必須檢查：

- 是否符合任務範圍。
- 是否包含未經授權的指令。
- 是否嘗試修改治理規則。
- 是否使用不可信來源。
- 是否存在資料洩漏。

## 11.4 No Permission Inheritance

委派者的權限不應自動傳遞給受委派者。

受委派者必須通過獨立 Policy Check。

## 11.5 No Audit Modification

任何 Agent 不得因為協作需要而修改或刪除既有 Audit 紀錄。

# 12. Collaboration with Continuity

AIA-10 定義長期 Project 與工作連續性。

跨 Agent 協作也必須納入 Continuity。

## 12.1 Collaboration Context

每個協作任務必須保留：

- Parent Project。
- Parent Goal。
- Parent Task。
- Delegation History。
- Participating Agents。
- Shared Evidence。
- Decisions。
- Results。
- Unresolved Issues。

## 12.2 Session Recovery

當 Worker 或 Agent 中斷後，AIA 必須能恢復：

- 原始委派目的。
- 已完成工作。
- 尚未完成工作。
- 已提交成果。
- 目前阻塞事項。
- 下一步行動。

不能因為 Agent Session 結束，就把整個協作任務視為失敗或重新開始。

## 12.3 Long-running Collaboration

大型協作可以跨越多個 Session。

AIA 應將其視為長期工作，而不是單次對話。

# 13. Cross-Domain Collaboration Scenarios

## 13.1 Investment × Coding

情境：

Investment Agent 發現目前市場資料處理流程不夠可靠。

協作：

1. Investment Agent 提出問題。
2. Autonomy Core 評估是否需要 Coding Agent。
3. Coding Agent 檢查資料擷取與處理程式。
4. 提交問題分析與修正建議。
5. Investment Agent 確認是否符合投資資料需求。
6. Coding Agent 在授權範圍內修改程式。
7. 執行驗證。
8. Investment Agent 確認資料品質改善。

責任：

- 投資資料需求由 Investment Agent 負責。
- 程式品質由 Coding Agent 負責。
- 協作流程由 Autonomy Core 管理。

## 13.2 Writing × Coding

情境：

Writing Agent 需要建立歷史人物與年表一致性檢查工具。

協作：

1. Writing Agent 定義歷史資料需求。
2. Coding Agent 設計資料結構與檢查工具。
3. Writing Agent 提供人物、事件與時間欄位。
4. Coding Agent 實作檢查流程。
5. Writing Agent 使用工具檢查小說設定。
6. 發現的史實疑點仍須依史料驗證。

Coding Agent 負責工具正確性，不負責替歷史資料背書。

## 13.3 Investment × Writing

這兩個 Domain 通常沒有直接工作依賴。

但可能共享：

- 資料來源管理。
- 長期研究紀錄。
- 證據可信度評估。
- 預測與結果比較方法。

不得將投資判斷直接套用到歷史研究。

## 13.4 Three-Agent Collaboration

情境：

AIA 要建立一項長期市場研究工具，並保存研究成果。

- Investment Agent 定義市場分析需求。
- Coding Agent 實作資料與分析工具。
- Writing Agent 協助整理可讀的研究報告。
- Autonomy Core 管理目標、委派與結果整合。

每個 Agent 仍對自己的專業交付負責。

# 14. Delegation Loop Prevention

Agent 之間可能產生循環委派。

例如：

```text
Agent A → Agent B
Agent B → Agent C
Agent C → Agent A
```

這可能造成：

- 無限委派。
- 重複工作。
- 資源浪費。
- 責任不清。

## 14.1 Control Mechanisms

AIA 應支援：

- Delegation Depth Limit。
- Cycle Detection。
- Maximum Retry。
- Deadline。
- Resource Budget。
- Duplicate Task Detection。
- Parent Task Tracking。

## 14.2 Re-delegation

受委派者不得無限制再次委派。

需要二次委派時，應檢查：

- 原始委派是否允許。
- 是否具有必要權限。
- 是否增加合理價值。
- 是否仍符合原始驗收條件。

必要時交由原委派者或 Autonomy Core 核准。

# 15. Collaboration Evaluation

AIA 必須評估協作是否真正改善成果。

## 15.1 Evaluation Dimensions

- Delivery Quality。
- Accuracy。
- Completion Time。
- Resource Consumption。
- Coordination Cost。
- Rework Rate。
- Error Propagation。
- Verification Success。
- Goal Contribution。

## 15.2 Delegation Effectiveness

可以比較：

- 自行完成的預期成本。
- 委派完成的實際成本。
- 委派前預估成果。
- 委派後實際成果。

不能只用完成速度衡量協作品質。

## 15.3 Agent Reliability

Agent 的可靠程度應根據歷史交付紀錄評估。

例如：

- 準時交付率。
- 驗收通過率。
- 錯誤率。
- 重工率。
- 證據完整性。
- 長期穩定性。

評估結果可以影響後續 Capability Matching，但不能直接取代每次任務的實際驗證。

# 16. Go Implementation 建議

本章定義邏輯元件，不代表目前程式碼已經完成這些能力。

實際實作應整合既有 Agent Registry、Task、Worker、Runtime、Policy、Audit 與 Repository。

## 16.1 Collaboration Engine

```go
type CollaborationEngine interface {
    Consult(
        ctx context.Context,
        request ConsultationRequest,
    ) (ConsultationResult, error)

    Delegate(
        ctx context.Context,
        request DelegationRequest,
    ) (Delegation, error)

    Coordinate(
        ctx context.Context,
        collaboration Collaboration,
    ) (CollaborationResult, error)

    Review(
        ctx context.Context,
        request ReviewRequest,
    ) (ReviewResult, error)
}
```

## 16.2 Capability Registry

```go
type CapabilityRegistry interface {
    Register(
        ctx context.Context,
        capability AgentCapability,
    ) error

    Discover(
        ctx context.Context,
        requirement CapabilityRequirement,
    ) ([]AgentCapability, error)

    Get(
        ctx context.Context,
        agentID string,
    ) (AgentCapability, error)
}
```

## 16.3 Delegation Service

```go
type DelegationService interface {
    Create(
        ctx context.Context,
        request DelegationRequest,
    ) (Delegation, error)

    Assign(
        ctx context.Context,
        delegationID string,
        agentID string,
    ) error

    SubmitResult(
        ctx context.Context,
        delegationID string,
        result DelegationResult,
    ) error

    Close(
        ctx context.Context,
        delegationID string,
    ) error
}
```

## 16.4 Collaboration Orchestrator

```go
type CollaborationOrchestrator interface {
    Start(
        ctx context.Context,
        plan CollaborationPlan,
    ) (Collaboration, error)

    Execute(
        ctx context.Context,
        collaborationID string,
    ) error

    Recover(
        ctx context.Context,
        collaborationID string,
    ) error

    Cancel(
        ctx context.Context,
        collaborationID string,
    ) error
}
```

## 16.5 核心資料模型

```go
type AgentCapability struct {
    AgentID       string
    Domain        string
    Description   string
    SupportedTasks []string
    RequiredTools  []string
    Limitations    []string
    Version        string
}

type DelegationRequest struct {
    ID                string
    ParentTaskID      string
    ProjectID         string
    GoalID            string
    RequestingAgentID string
    AssignedAgentID   string
    Description       string
    ExpectedOutput    string
    AcceptanceCriteria []string
    PermissionScope   []string
}

type DelegationResult struct {
    DelegationID string
    Status       string
    OutputRefs   []string
    EvidenceIDs  []string
    Limitations  []string
    Confidence   float64
}

type Collaboration struct {
    ID             string
    ParentTaskID   string
    ProjectID      string
    GoalID         string
    Participants   []string
    DelegationIDs  []string
    Status         string
}
```

以上為概念模型，實際欄位應與現有資料結構整合。

## 16.6 Worker Integration

現有 Worker 可以承載受委派的工作。

建議分工：

- Agent Registry：提供 Agent 能力資訊。
- Autonomy Core：決定是否需要協作。
- Collaboration Engine：管理協作流程。
- Delegation Service：建立與追蹤委派。
- Worker：執行具體工作。
- Continuity：保存長期協作狀態。
- Policy：管理授權。
- Audit：保存執行與決策紀錄。

不應為了跨 Agent 協作而重複建立已存在的 Worker 任務執行機制。

# 17. Implementation Phases

## X0 — Capability Discovery

- 整理 Agent Capability。
- 建立能力查詢。
- 建立能力限制描述。
- 支援基本 Agent Matching。

## X1 — Consultation & Delegation

- 支援 Agent Consultation。
- 建立 Delegation Request。
- 支援明確輸入與輸出。
- 建立委派狀態追蹤。

## X2 — Result Integration

- 接收委派成果。
- 支援 Partial Result。
- 支援 Review。
- 保存 Evidence Provenance。
- 建立責任追蹤。

## X3 — Multi-Agent Orchestration

- 支援 Sequential。
- 支援 Parallel。
- 支援 Review Loop。
- 支援 Hybrid。
- 支援 Session Recovery。

## X4 — Conflict & Safety

- 支援 Evidence Conflict。
- 支援 Reasoning Conflict。
- 支援循環委派檢查。
- 支援權限隔離。
- 支援資源限制。

## X5 — Cross-Domain Learning

- 支援 Knowledge Exchange。
- 支援 Learning Promotion。
- 支援 Domain Applicability Review。
- 支援協作品質評估。

# 18. Acceptance Criteria

## 18.1 Capability Discovery

- [ ] 能查詢 Agent 能力。
- [ ] 能辨識能力限制。
- [ ] 能根據任務需求選擇 Agent。
- [ ] 不會只根據 Agent 名稱決定委派對象。

## 18.2 Delegation

- [ ] 每項正式委派都有明確 Owner。
- [ ] 每項委派都有輸入與輸出定義。
- [ ] 每項委派都有驗收條件。
- [ ] 能追蹤委派狀態。
- [ ] 能處理失敗與重試。
- [ ] 能恢復中斷的委派工作。

## 18.3 Collaboration

- [ ] 支援不同協作模式。
- [ ] 能整合多個 Agent 的成果。
- [ ] 能保留不同 Agent 的意見。
- [ ] 能處理未解決的衝突。
- [ ] 能避免循環委派。
- [ ] 能限制資源消耗。

## 18.4 Security

- [ ] 委派不會自動傳遞權限。
- [ ] 只共享必要上下文。
- [ ] 會檢查其他 Agent 的輸出。
- [ ] 不允許 Agent 修改 Audit。
- [ ] 不允許透過二次委派規避 Policy。

## 18.5 Continuity

- [ ] 協作跨 Session 後仍可恢復。
- [ ] 能查詢委派歷史。
- [ ] 能還原參與 Agent。
- [ ] 能追蹤成果來源。
- [ ] 能追蹤最終驗收責任。

# 19. 風險與控制

| 風險 | 說明 | 控制方式 |
|---|---|---|
| Agent Proliferation | Agent 數量過多 | Capability Registry、成本評估 |
| Duplicate Delegation | 重複委派 | Task Deduplication |
| Delegation Loop | 循環委派 | Cycle Detection |
| Responsibility Diffusion | 責任不清 | Task Owner |
| Evidence Contamination | 錯誤資訊傳播 | Provenance、Verification |
| Permission Escalation | 權限擴張 | Least Privilege、Policy |
| Context Leakage | 不必要資料共享 | Context Isolation |
| Coordination Overhead | 協作成本過高 | Delegation Value |
| Endless Review | 無限審查 | Retry Limit、Stop Condition |
| Cross-Domain Misuse | 錯誤套用其他領域知識 | Domain Applicability Review |

# 20. 最終設計原則

AIA 的多 Agent 架構不應只是把工作拆給更多 Agent。

真正的跨領域協作必須建立在共同目標、明確責任、證據來源與可驗證成果之上。

AIA 應能：

1. 知道目前有哪些 Agent。
2. 知道每個 Agent 能做什麼。
3. 知道什麼工作值得委派。
4. 知道應該委派給誰。
5. 知道如何提供必要上下文。
6. 知道如何檢查交付成果。
7. 知道如何處理不同 Agent 的衝突。
8. 知道哪些知識可以共享。
9. 知道哪些資訊必須保留領域界線。
10. 知道誰對最後成果負責。

**Agent 可以分工，但目標不能分裂；Agent 可以交換知識，但證據不能失去來源；Agent 可以互相協助，但責任不能消失。**

Autonomy Core 的角色不是替所有 Agent 思考，而是讓不同 Agent 能在共同的長期目標與治理規則下，自主協作並持續改善整體成果。

---

# 21. 跨領域委派契約資料模型與狀態機 (Delegation Request Model — Migration 012 Addendum)

為了落實跨 Agent 協作之契約化與可追溯性，AIA Migration 012 建立了 `delegation_requests` 資料表，支援以下完整生命週期：

```text
  [ PENDING ] ──(Target Agent 評估接收)──► [ ACCEPTED ] ──(派發工單)──► [ IN_PROGRESS ]
       │                                                                      │
       ▼                                                                      ▼
  [ REJECTED ]                                                           [ DELIVERED ]
                                                                              │
                                                                 (驗收檢查 Acceptance Gate)
                                                                       ├── 通過 ──► [ VERIFIED ]
                                                                       └── 失敗 ──► [ FAILED ]
```

### 21.1 契約核心欄位
1. `requester_agent_id`：提出委派之主委託者（如 `writing-agent`）。
2. `target_agent_id`：承接委派之執行者（如 `coding-agent`）。
3. `goal_id` / `source_project_id`：繼承之頂層目標與來源脈絡，避免目標分裂。
4. `task_spec_json`：結構化工作規格（輸入參數、產出格式、資源限制）。
5. `acceptance_gate_json`：可量測之驗收標準（如單元測試全過、輸出符合 schema）。
6. `spawned_task_id`：於執行者端衍生之實際執行工單。

### 21.2 經典實證：Writing × Coding 史料工具委派流
- **觸發**：Writing Agent 在校核《元末明初》小說時，發現需比對 540 章中百餘位歷史人物的卒年與出場年份，人工正則易漏。
- **委派**：Writing Agent 建立 `DelegationRequest`，指名 `coding-agent`，要求開發 `verify_timeline_anomaly` 分析程式。
- **執行**：Coding Agent 在沙盒內實作並通過單元測試，狀態轉為 `DELIVERED`。
- **驗收**：Writing Agent 執行交付腳本，確認能精確標出卒後登場異常，狀態更新為 `VERIFIED`，完成跨領域協作閉環。
