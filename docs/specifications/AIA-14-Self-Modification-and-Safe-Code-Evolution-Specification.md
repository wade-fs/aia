# AIA-14：Self-Modification & Safe Code Evolution Specification

**Version:** 2.0  
**Document Type:** System Specification  
**Project:** AIA (Autonomous Intelligence Architecture)  
**Status:** Proposed Specification  
**Language:** Traditional Chinese

**依賴文件：**
- AIA-01 Product Requirements
- AIA-02 System Architecture
- AIA-03 Go Runtime Implementation
- AIA-04 Test Plan & Acceptance Criteria
- AIA-08 Autonomy Core & Operational Autonomy
- AIA-09 Initiative & Goal Evolution
- AIA-10 Continuity & Long-term Project Management
- AIA-11 Self-Correction & Learning
- AIA-12 Cross-Domain Collaboration & Agent Delegation
- AIA-13 Autonomy Evaluation & Benchmarking

---

# 1. 文件目的

## 1.1 設計目的

AIA 的自主性不應只停留在分析、建議與任務執行。

如果 Agent 在長期工作中發現：

- 程式設計不合理。
- 某個模組存在缺陷。
- 原有演算法效率不足。
- 資料處理方式不正確。
- Runtime 無法支援新的自主行為。
- Autonomy Core 缺少必要能力。

它應該能夠提出修改方案，並在適當授權下完成修改、驗證與持續改善。

這種能力稱為：

**Self-Modification：自主系統對自身或工作環境中的程式碼進行修改。**

然而，能夠修改程式碼並不代表可以不受限制地修改整個系統。

本文件定義修改範圍、權限層級、執行流程、驗證方式、版本管理、回滾與治理規則。

## 1.2 核心目標

AIA 必須具備以下能力：

1. 發現程式碼或系統架構的改善機會。
2. 分析問題與根本原因。
3. 提出可追溯的修改方案。
4. 評估修改風險及影響範圍。
5. 在授權範圍內修改程式碼。
6. 在隔離環境驗證修改結果。
7. 比較修改前後的品質與行為。
8. 發現 Regression。
9. 必要時自動回滾。
10. 將修改經驗記錄為 Learning。
11. 根據長期評估結果持續改善。

## 1.3 設計原則

Self-Modification 必須遵循：

- Modification Must Be Evidence-driven。
- Modification Must Be Versioned。
- Modification Must Be Reproducible。
- Modification Must Be Verifiable。
- Modification Must Be Reversible。
- Modification Must Respect Authorization。
- Modification Must Preserve Audit Integrity。

Agent 可以提出修改自身程式碼的需求，但不能因此取得修改自身權限規則的能力。

---

# 2. Self-Modification 的三種範圍

這是本文件最重要的架構區分。

## 2.1 Level A：Domain Code Modification

<badge color="success">一般自主修改範圍</badge>

修改 Agent 所負責的領域專案程式碼。

例如：

### Investment

- 修正市場資料解析程式。
- 改善財務資料整理。
- 修改投資分析演算法。
- 改善投資組合計算。
- 修正報告產生程式。
- 優化投資研究工具。

### Coding

- 修改使用者指定的程式專案。
- 修正 Bug。
- 改善演算法。
- 重構模組。
- 增加測試。
- 改善效能。

### Writing

Writing Agent 不一定需要修改程式碼，但可以透過 Coding Agent 建立或修改：

- 歷史年表檢查工具。
- 人物關係檢查工具。
- 章節進度管理工具。
- 史料整理工具。
- 文件轉換與校勘工具。

### 授權原則

在使用者預先授權的專案範圍內，Agent 可以：

- 建立修改分支。
- 修改程式碼。
- 執行測試。
- 修正測試失敗。
- 提出 Pull Request 或等效變更紀錄。
- 在限定範圍內完成自動化驗證。

是否允許自動合併，應由專案 Policy 決定。

## 2.2 Level B：AIA Core Modification

<badge color="warning">受控自主修改範圍</badge>

修改 AIA 自身的核心程式碼。

例如：

- Go Runtime。
- Autonomy Core。
- Initiative Engine。
- Continuity Engine。
- Memory Engine。
- Self-Correction Engine。
- Goal Management。
- Evaluation Engine。
- Collaboration Engine。
- Worker 管理與執行機制。

這些修改會直接影響 AIA 自身的運作方式。

例如：

Agent 發現目前的 Continuity Engine 無法正確恢復跨 Session 的 Project 狀態，可能提出修改：

1. 找出狀態恢復失敗原因。
2. 分析資料模型與 Context Reconstruction。
3. 建立修正分支。
4. 修改程式碼。
5. 執行相關測試。
6. 執行 Regression Test。
7. 進行隔離環境驗證。
8. 提交修改結果供人類審核。

### 授權原則

AIA Core Modification 必須採取較高級別的控制。

Agent 可以自主：

- 發現問題。
- 分析根因。
- 提出修改方案。
- 建立獨立分支。
- 在隔離環境修改。
- 執行測試。
- 產生 Diff。
- 撰寫修改報告。
- 提出合併建議。

但不得因為自行修改了核心程式碼，就直接取得部署新版本或修改治理規則的權限。

核心程式碼修改的合併與正式部署，必須經過人類核准。

## 2.3 Level C：Governance Modification

<badge color="danger">禁止 Agent 自行修改</badge>

治理機制包含：

- Policy。
- Permission。
- Approval。
- Audit。
- Security Boundary。
- Credential Boundary。
- Human Oversight。
- Self-Modification Authorization。

Agent 不得自行：

1. 解除原有權限限制。
2. 修改自己的 Approval Requirement。
3. 提高自身權限。
4. 允許自己存取原本禁止的資源。
5. 關閉 Audit。
6. 修改或刪除歷史 Audit。
7. 繞過人類核准。
8. 修改 Self-Modification 的安全限制。
9. 將未經核准的修改直接部署到正式環境。
10. 透過其他 Agent 或 Worker 間接完成上述禁止行為。

Agent 可以分析治理規則，提出改善建議。

但治理規則的修改必須由人類明確核准，並由受信任的管理流程執行。

---

# 3. Modification Authorization Model

## 3.1 授權不是單純的 Agent 權限

是否能修改程式碼，不能只依照 Agent 的角色決定。

必須同時考慮：

- Modification Scope。
- Project Ownership。
- Resource Boundary。
- Risk Level。
- Policy。
- Approval Requirement。
- Execution Environment。

例如：

Coding Agent 具有程式修改能力，不代表它可以修改 AIA Runtime。

Investment Agent 可以修改自己負責的投資分析程式，不代表它可以修改交易或資金治理規則。

## 3.2 修改風險分類

| 等級 | 修改類型 | 建議處理方式 |
|---|---|---|
| M0 | 文件、註解、非執行性資料 | 可依預先授權自動修改 |
| M1 | 一般 Domain Code | 可在授權範圍內修改及驗證 |
| M2 | 影響多個模組的 Domain Code | 需要額外審查 |
| M3 | AIA Core Code | 必須人類核准合併與部署 |
| M4 | Governance / Security / Permission | Agent 不得自行執行 |

M0 至 M4 是修改風險分類，不代表 AIA 的既有 Policy 已經提供這些權限。

實際授權仍以系統 Policy 為準。

## 3.3 Risk Assessment

每次修改前，至少評估：

- 影響模組數量。
- 影響 Project 數量。
- 是否改變公開介面。
- 是否影響資料格式。
- 是否影響歷史資料。
- 是否影響其他 Agent。
- 是否影響 Runtime。
- 是否影響安全邊界。
- 是否容易回滾。
- 是否有完整測試。

修改範圍越大、依賴越多、回滾越困難，所需審核級別越高。

## 3.4 預先授權與單次授權

AIA 應支援兩種授權方式。

### Standing Authorization

使用者預先授權某個 Project 的特定修改範圍。

例如：

允許 Coding Agent 在指定 Repository 中，自行建立分支、修改程式碼、執行測試與修正一般 Bug。

### Explicit Approval

每次修改都必須取得使用者核准。

適用於：

- AIA Core Modification。
- 高風險架構修改。
- 重大資料格式變更。
- Security Boundary 變更。
- Governance Modification。

預先授權不得被解釋為無限制授權。

---

# 4. Self-Modification Lifecycle

## 4.1 完整生命週期

```text
Problem / Improvement Discovery
              |
              v
       Root Cause Analysis
              |
              v
      Modification Proposal
              |
              v
       Risk Assessment
              |
              v
     Authorization Check
              |
              v
       Create Branch
              |
              v
      Isolated Modification
              |
              v
       Static Analysis
              |
              v
       Automated Testing
              |
              v
      Regression Testing
              |
              v
       Quality Evaluation
              |
              v
       Diff / Change Review
              |
              v
       Approval Decision
              |
              v
       Merge / Deployment
              |
              v
       Post-Deployment Monitoring
              |
              v
       Rollback if Required
              |
              v
       Learning Record
```

任何階段失敗，都必須保留失敗紀錄。

## 4.2 Problem Discovery

修改的起點可能來自：

- 使用者要求。
- Runtime Error。
- Worker Failure。
- Benchmark Regression。
- Performance Degradation。
- Repeated Failure。
- Domain Agent 發現的改善機會。
- Autonomy Core 主動發現的問題。

不能因為某段程式碼看起來不夠漂亮，就任意啟動大規模重構。

修改應有具體問題、改善目標或可驗證的效益。

## 4.3 Modification Proposal

每次修改必須建立 Proposal。

至少包含：

| 欄位 | 說明 |
|---|---|
| Proposal ID | 修改提案識別碼 |
| Scope | 修改範圍 |
| Target Repository | 目標 Repository |
| Target Version | 修改基準版本 |
| Problem | 問題描述 |
| Root Cause | 根本原因 |
| Proposed Changes | 修改內容 |
| Expected Benefits | 預期效益 |
| Risks | 風險 |
| Dependencies | 相依項目 |
| Validation Plan | 驗證方式 |
| Rollback Plan | 回滾計畫 |
| Authorization | 授權需求 |

## 4.4 Authorization Check

執行修改前，系統必須確認：

1. 目標 Repository 是否允許存取。
2. 修改範圍是否符合 Standing Authorization。
3. 是否涉及 M3 或 M4。
4. 是否需要人類核准。
5. 執行環境是否符合要求。
6. 是否存在尚未完成的衝突修改。

沒有授權時，Agent 可以產生 Proposal，但不得執行未授權修改。

## 4.5 Isolated Modification

所有程式碼修改應在獨立工作環境進行。

建議使用：

- Git Branch。
- Git Worktree。
- Dedicated Workspace。
- Container。
- Sandbox。

原始正式版本不得被直接覆寫。

同一個 Repository 可以同時存在多個修改分支，但必須避免相互覆蓋或將不同 Proposal 的變更混在一起。

## 4.6 Verification

修改後至少需要執行適用的驗證：

- Formatting。
- Static Analysis。
- Unit Tests。
- Integration Tests。
- Regression Tests。
- Interface Compatibility。
- Data Compatibility。
- Security Checks。
- Performance Evaluation。

並非每種修改都需要執行全部項目。

但 Proposal 必須事先說明驗證範圍及未執行項目的理由。

## 4.7 Diff Review

Agent 必須產生可讀取的修改差異。

至少包含：

- 修改檔案。
- 新增內容。
- 刪除內容。
- 修改原因。
- 影響模組。
- 測試結果。
- 未解決問題。
- 已知風險。

Diff 是修改證據，不應只提供 Agent 自己撰寫的摘要。

## 4.8 Approval

審核結果包含：

- APPROVED。
- REJECTED。
- REQUEST_CHANGES。
- DEFERRED。

審核必須與 Proposal、Diff、測試結果及目標版本建立關聯。

## 4.9 Deployment

正式部署前必須確認：

- 核准狀態有效。
- 目標版本符合預期。
- 部署產物與已審核內容一致。
- 備份或回滾方案存在。
- 部署權限有效。
- Audit 已準備記錄。

Agent 不得自行改變部署目標或將未審核內容混入正式版本。

## 4.10 Post-Deployment Monitoring

部署後仍需觀察：

- Error Rate。
- Crash Rate。
- Task Failure Rate。
- Worker Failure Rate。
- Resource Usage。
- Performance。
- Regression。
- Autonomy Metrics。

修改通過測試，不代表部署後一定成功。

---

# 5. Version Control & Change Isolation

## 5.1 Git 作為修改基礎

Self-Modification 應以 Git 版本控制為主要基礎。

每個 Proposal 應對應：

- Base Commit。
- Working Branch。
- Change Commit。
- Review Record。
- Merge Commit。
- Deployment Version。

## 5.2 Branch Naming

建議格式：

```text
aia/self-mod/<proposal-id>
domain/<agent-name>/<proposal-id>
```

例如：

```text
aia/self-mod/SM-2026-0012
domain/investment/SM-2026-0013
```

實際命名可依既有 Repository 慣例調整。

## 5.3 Change Isolation

不同修改提案應保持獨立。

例如：

- Investment 資料解析修正。
- Coding 專案重構。
- AIA Continuity Engine 修改。

不應在同一個未分離的修改分支中混合執行。

## 5.4 Baseline Preservation

每次修改前必須保存原始基準：

- Git Commit。
- Project State。
- 測試結果。
- 相關 Metric。
- 依賴版本。
- 必要設定。

如此才能在修改後比較差異。

## 5.5 Reproducible Change

每次修改應記錄：

- 使用的模型版本。
- Agent 版本。
- Harness 版本。
- Prompt 或相關設定版本。
- 工具版本。
- 執行環境。
- 相關輸入資料。

目的不是要求完全重現模型的每一個 Token，而是讓修改過程與驗證條件可以被追蹤。

---

# 6. Safe Execution Environment

## 6.1 執行環境隔離

Agent 修改程式碼時，不應直接在正式運作環境中操作。

至少區分：

1. Production Environment。
2. Development Workspace。
3. Validation Environment。
4. Staging Environment。

## 6.2 Harness Boundary

AIA 使用的 Agent Harness 負責提供工具執行與權限控制。

AIA 不應自行繞過 Harness 的安全機制。

例如：

- 不使用跳過權限檢查的執行模式。
- 不複製或解析使用者 OAuth Token。
- 不因為修改程式碼需要而自動取得額外權限。
- 不將正式環境憑證放入一般測試 Workspace。

## 6.3 Resource Limits

修改與驗證工作應受到資源限制：

- CPU。
- Memory。
- Disk。
- Execution Time。
- Network Access。
- Tool Permission。
- Maximum Retry。

超過資源限制時，應停止或暫停修改流程，並保留狀態。

## 6.4 Network Boundary

網路存取必須遵守既有 Policy。

例如：

- 只允許必要的套件來源。
- 不允許任意上傳專案內容。
- 不允許將憑證傳送至外部服務。
- 不允許修改網路權限以繞過限制。

---

# 7. Testing & Verification Strategy

## 7.1 測試層級

### Level 1：Static Validation

- Formatting。
- Lint。
- Static Analysis。
- Dependency Check。

### Level 2：Unit Validation

驗證被修改的函式、模組及資料處理邏輯。

### Level 3：Integration Validation

驗證與其他模組的互動。

### Level 4：Regression Validation

確認原有功能未受到破壞。

### Level 5：Behavioral Validation

確認修改後 Agent 的實際行為符合預期。

### Level 6：Autonomy Benchmark

對 AIA Core 或 Autonomy Core 的重要修改，必須進一步確認自主能力指標沒有退化。

## 7.2 Baseline Comparison

修改前後應比較：

| 類別 | 比較項目 |
|---|---|
| Correctness | 正確性 |
| Reliability | 穩定性 |
| Performance | 效能 |
| Maintainability | 可維護性 |
| Compatibility | 相容性 |
| Security | 安全性 |
| Autonomy | 自主能力 |
| Resource | 資源消耗 |

## 7.3 Regression Policy

若修改造成重大 Regression：

1. 停止後續部署。
2. 保留錯誤紀錄。
3. 判斷是否需要回滾。
4. 建立 Root Cause Analysis。
5. 修正問題。
6. 重新執行驗證。

不得為了完成修改而忽略既有功能退化。

## 7.4 Test Failure Handling

測試失敗時，Agent 可以在原授權範圍內嘗試修正。

但必須設定：

- Maximum Retry。
- Maximum Modification Scope。
- Execution Budget。
- Time Limit。
- Failure Threshold。

超過限制時應停止並交由人類處理。

避免 Agent 因為無法通過測試而無限制地修改程式碼。

---

# 8. Rollback & Recovery

## 8.1 Rollback 原則

每次正式修改都必須具有回滾能力。

回滾目標是恢復到最後一個已知穩定版本。

## 8.2 Rollback Trigger

以下情況可觸發回滾：

- 重大功能失效。
- Runtime 無法正常啟動。
- 資料一致性遭破壞。
- Security Regression。
- Worker 大量失敗。
- Autonomy Metrics 明顯退化。
- 部署後出現不可接受的錯誤。

## 8.3 Rollback Authority

對於已預先授權的 Domain Code，Agent 可以在符合明確回滾條件時自動執行回滾。

對於 AIA Core，回滾必須依事先核准的部署與恢復政策執行。

回滾不是修改 Audit 的行為。

原始修改與失敗紀錄必須保留。

## 8.4 Data Migration

如果修改涉及資料格式或資料庫結構，必須另外考慮：

- Migration。
- Backward Compatibility。
- Forward Compatibility。
- Backup。
- Restore。
- Data Integrity。

不能只回滾程式碼，卻忽略已經被修改的資料格式。

---

# 9. Self-Modification & Learning Integration

## 9.1 與 AIA-11 整合

Self-Modification 是 Self-Correction 的一種執行方式。

兩者不能混為一談。

AIA-11 定義：

- 問題辨識。
- 根因分析。
- 修正提案。
- 修正評估。
- 學習紀錄。

AIA-14 定義：

- 程式碼修改。
- 版本控制。
- 隔離執行。
- 測試驗證。
- 審核部署。
- 回滾。

## 9.2 Learning Record

每次修改完成後，應建立 Learning Record。

包含：

- Original Problem。
- Root Cause。
- Modification。
- Verification Result。
- Regression Result。
- Actual Outcome。
- Lessons Learned。
- Future Applicability。

## 9.3 防止錯誤學習

不能因為修改後某個測試通過，就認定整體品質一定改善。

例如：

- 測試案例可能不完整。
- 修正可能只針對單一情境。
- 其他模組可能受到影響。
- 短期效能改善可能造成長期維護困難。

因此，Self-Modification 的成功判定應結合 AIA-13 的長期評估。

## 9.4 Modification Feedback Loop

```text
Operational Experience
        |
        v
   Evaluation
        |
        v
  Problem Discovery
        |
        v
 Root Cause Analysis
        |
        v
Modification Proposal
        |
        v
 Safe Modification
        |
        v
    Verification
        |
        v
 Deployment / Rollback
        |
        v
 Long-term Evaluation
        |
        v
 Learning Record
        |
        +------> Future Improvement
```

---

# 10. Autonomous Code Evolution

## 10.1 設計目標

AIA 的長期目標之一，是讓 Agent 能夠根據自身工作經驗改善程式架構。

這不應被理解為 Agent 可以任意重寫自身。

正確的設計是：

Agent 能夠持續發現改善機會，提出可驗證的修改，並在授權範圍內完成安全演進。

## 10.2 演進範圍

可分為：

### Domain Evolution

改善各 Domain Agent 所使用的程式碼、工具與演算法。

### Runtime Evolution

改善 Task、Worker、Memory、Session、Event、Recovery 等 Runtime 能力。

### Autonomy Evolution

改善 Initiative、Continuity、Self-Correction、Goal Evolution 等自主能力。

### Governance Evolution

由人類管理的治理制度演進。

前三者可以提出自主修改提案，但 Runtime 與 Autonomy Core 的正式變更必須經人類核准。

Governance Evolution 不允許 Agent 自行完成。

## 10.3 Evolution Proposal

Agent 應能根據：

- 長期錯誤模式。
- 重複工作成本。
- Benchmark Regression。
- 使用者回饋。
- Resource Consumption。
- Domain Experience。

提出架構改善建議。

每項建議都必須有：

- 問題證據。
- 改善假設。
- 預期收益。
- 可能副作用。
- 驗證方式。
- 回滾計畫。

## 10.4 避免自我修改循環失控

應設置：

- Modification Depth Limit。
- Retry Limit。
- Change Scope Limit。
- Maximum Concurrent Proposals。
- Cooldown Period。
- Human Approval Gate。
- Rollback Requirement。

Agent 不應在尚未驗證上一項修改前，無限制啟動下一輪核心修改。

---

# 11. Agent Collaboration & Self-Modification

## 11.1 跨 Agent 修改

不同 Agent 可以共同參與程式碼演進。

例如：

- Investment Agent 發現投資資料解析問題。
- Coding Agent 分析與修改程式。
- Evaluation Engine 驗證修正結果。
- Autonomy Core 決定是否採用修改建議。

## 11.2 Responsibility

每次修改必須指定：

- Proposal Owner。
- Modification Agent。
- Reviewer。
- Approver。
- Deployment Owner。

同一個 Agent 可以負責多個角色，但不能因此跳過必要的審核程序。

## 11.3 Independent Review

高風險修改應盡量使用不同 Agent 或獨立驗證流程進行審查。

修改者不能單憑自己的判斷，宣告修改已經安全。

## 11.4 Untrusted Output

其他 Agent 提供的程式碼、Diff、分析與測試結果，都必須視為待驗證輸入。

委派不代表權限繼承。

Reviewer 也不能因為收到其他 Agent 的結果，就取得原本沒有的 Repository 或部署權限。

---

# 12. Security & Governance

## 12.1 Least Privilege

每次修改只取得完成該項工作的必要權限。

## 12.2 No Self-Authorization

Agent 不得：

- 核准自己的高風險修改。
- 提高自己的修改權限。
- 修改自己的核准條件。
- 將 M3 或 M4 降級。
- 建立繞過治理的替代執行流程。

## 12.3 Audit Integrity

Audit 必須記錄：

- Proposal Created。
- Authorization Checked。
- Approval Requested。
- Approval Granted / Rejected。
- Branch Created。
- File Modified。
- Test Executed。
- Diff Generated。
- Merge Completed。
- Deployment Completed。
- Rollback Executed。

Audit 必須由受信任的系統元件管理。

Agent 可以讀取與引用 Audit，但不得修改或刪除歷史紀錄。

## 12.4 Credential Boundary

AIA 不應讀取、複製或解析使用者 OAuth Token。

程式碼修改所需的 Repository、Package Registry 或其他外部服務權限，必須透過既有 Harness 或使用者明確管理的認證機制取得。

## 12.5 Supply Chain Security

外部套件與工具的引入必須檢查：

- 套件來源。
- 版本。
- License。
- 已知漏洞。
- 相依關係。
- 維護狀態。

Agent 不得為了讓測試通過而任意引入不可信套件。

---

# 13. Observability & Reporting

## 13.1 Modification Dashboard

Web Console 建議提供：

- Modification Proposals。
- Active Branches。
- Pending Approvals。
- Verification Results。
- Regression Findings。
- Deployment History。
- Rollback History。
- Modification Trends。

## 13.2 Modification Report

每次修改完成後，至少產生：

1. 修改原因。
2. 問題根因。
3. 修改範圍。
4. 修改檔案。
5. Diff 摘要。
6. 測試結果。
7. Regression 結果。
8. 風險評估。
9. 核准紀錄。
10. 部署狀態。
11. 後續觀察計畫。
12. Learning Record。

## 13.3 Long-term Metrics

可以透過 AIA-13 評估：

- Modification Success Rate。
- Regression Rate。
- Rollback Rate。
- Mean Time to Recovery。
- Repeated Defect Rate。
- Verified Improvement Rate。
- Unauthorized Modification Attempts。
- Core Modification Approval Rate。

這些指標用於觀察修改品質，不應以修改數量越多越好作為評估原則。

---

# 14. Go Implementation Specification

## 14.1 建議 Package

```text
internal/
  modification/
    proposal/
    authorization/
    workspace/
    executor/
    validation/
    review/
    deployment/
    rollback/
    audit/
    model/
    repository/
```

實際實作應整合既有 Runtime、Worker、Policy 與 Audit，不應建立另一套互相競爭的權限系統。

## 14.2 核心資料模型

### ModificationProposal

```go
type ModificationProposal struct {
    ID string

    OwnerAgentID string
    TargetRepository string
    TargetScope string

    BaseCommit string
    BranchName string

    RiskLevel string

    Problem string
    RootCause string
    ProposedChanges string

    ExpectedBenefits []string
    Risks []string

    ValidationPlan []string
    RollbackPlan string

    AuthorizationStatus string
    ApprovalStatus string
    ExecutionStatus string

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### ModificationRecord

```go
type ModificationRecord struct {
    ID string
    ProposalID string

    BaseCommit string
    ResultCommit string

    ChangedFiles []string
    DiffReference string

    TestResults []string
    RegressionResults []string

    ApprovedBy string
    MergedAt *time.Time
    DeployedAt *time.Time
    RolledBackAt *time.Time
}
```

### AuthorizationDecision

```go
type AuthorizationDecision struct {
    ProposalID string

    Allowed bool
    Scope string
    RiskLevel string

    RequiresHumanApproval bool

    PolicyReferences []string
    Reason string

    EvaluatedAt time.Time
}
```

以上為概念資料模型。實際欄位、狀態型別與 Repository 應遵循 AIA-03 的既有設計。

## 14.3 核心介面

```go
type ModificationService interface {
    Propose(
        ctx context.Context,
        request ModificationRequest,
    ) (*ModificationProposal, error)

    Authorize(
        ctx context.Context,
        proposalID string,
    ) (*AuthorizationDecision, error)

    Execute(
        ctx context.Context,
        proposalID string,
    ) (*ModificationRecord, error)

    Review(
        ctx context.Context,
        proposalID string,
    ) (*ModificationReview, error)

    Rollback(
        ctx context.Context,
        modificationID string,
    ) error
}
```

介面僅表示功能責任。

實際實作應將 Proposal、Authorization、Execution、Review、Deployment 分離，避免單一 Service 同時掌握所有高權限操作。

## 14.4 與 Worker 整合

現有 Worker 可以承載：

- 程式碼分析。
- 分支建立。
- 程式碼修改。
- 測試執行。
- Diff 產生。
- 文件更新。
- 修改報告。

Worker 是執行單位，不是授權來源。

Worker 接到修改任務時，仍必須由 AIA 的 Authorization 與 Policy 機制確認權限。

Worker 不得因為被某個 Agent 委派，就自動繼承該 Agent 的全部權限。

---

# 15. Implementation Phases

## M0：Modification Governance Foundation

建立修改提案與授權機制。

工作：

- Modification Proposal。
- Scope Classification。
- Risk Classification。
- Authorization Check。
- Audit Integration。

驗收：

- 能建立修改提案。
- 能判斷修改範圍。
- 能區分 M0 至 M4。
- 未授權修改無法執行。
- 所有授權判斷均可追蹤。

## M1：Domain Code Modification

建立一般 Domain 程式碼修改能力。

工作：

- Git Branch。
- Isolated Workspace。
- Code Modification。
- Test Execution。
- Diff Generation。
- Change Report。

驗收：

- Agent 能在授權 Repository 中修改程式碼。
- 原始版本受到保護。
- 修改可以重現與比較。
- 測試結果可追蹤。

## M2：Safe Verification & Rollback

建立安全驗證與回滾能力。

工作：

- Static Analysis。
- Unit Test。
- Integration Test。
- Regression Test。
- Rollback。
- Recovery。

驗收：

- 能辨識重大 Regression。
- 能依授權條件停止部署。
- 能恢復已知穩定版本。
- 回滾不會刪除原始 Audit。

## M3：AIA Core Modification

建立核心程式碼修改流程。

工作：

- Core Repository Integration。
- Isolated Core Workspace。
- Core Regression Benchmark。
- Human Approval。
- Staging Deployment。
- Post-Deployment Monitoring。

驗收：

- Agent 能提出核心修改方案。
- Agent 能在隔離環境進行修改。
- 核心修改必須經人類核准。
- 正式部署必須可追溯。
- 修改後能執行自主能力 Benchmark。
- 沒有核准時不得正式部署。

## M4：Autonomous Code Evolution

建立長期程式碼演進能力。

工作：

- Improvement Discovery。
- Modification Prioritization。
- Long-term Defect Analysis。
- Evaluation Feedback。
- Learning Integration。
- Controlled Evolution。

驗收：

- 能從長期運作發現改善機會。
- 能提出有證據的修改提案。
- 能驗證修改效果。
- 能避免重複錯誤。
- 能辨識修改造成的退化。
- 能在必要時回滾。
- 不會自行解除治理限制。

---

# 16. Acceptance Criteria

## 16.1 Modification Scope

- [ ] 能區分 Domain Code、AIA Core、Governance 三種修改範圍。
- [ ] 能依修改範圍判斷授權。
- [ ] 能區分修改風險。
- [ ] 能辨識跨模組影響。

## 16.2 Safe Modification

- [ ] 能建立獨立修改分支。
- [ ] 能在隔離環境工作。
- [ ] 能保存原始版本。
- [ ] 能產生完整 Diff。
- [ ] 能執行適用的驗證。
- [ ] 能追蹤測試結果。

## 16.3 Approval

- [ ] 支援預先授權。
- [ ] 支援單次核准。
- [ ] AIA Core 正式合併須人類核准。
- [ ] Governance 修改不得由 Agent 自行執行。
- [ ] Agent 無法自行提高權限。

## 16.4 Rollback

- [ ] 每次正式修改都有回滾方案。
- [ ] 能辨識重大 Regression。
- [ ] 能依政策執行回滾。
- [ ] 回滾不會破壞 Audit。
- [ ] 能處理資料格式變更的恢復問題。

## 16.5 Self-Improvement

- [ ] 能記錄修改原因。
- [ ] 能保留根因分析。
- [ ] 能比較修改前後結果。
- [ ] 能將修改經驗整合至 Learning。
- [ ] 能透過 AIA-13 追蹤長期改善。
- [ ] 不以修改次數作為改善程度。

---

# 17. Risk & Mitigation

| 風險 | 說明 | 控制方式 |
|---|---|---|
| Unsafe Modification | 修改破壞原有功能 | 隔離環境與 Regression Test |
| Permission Escalation | Agent 自行提高權限 | 獨立 Authorization |
| Governance Bypass | 修改治理程式碼繞過限制 | M4 禁止 Agent 自行執行 |
| Infinite Correction | 不斷修改卻無法收斂 | Retry 與 Scope Limit |
| Regression | 修改造成既有功能失效 | Benchmark 與 Rollback |
| Dependency Risk | 引入不可信套件 | Supply Chain Validation |
| Data Corruption | 修改造成資料損壞 | Backup 與 Migration Plan |
| Self-Validation Bias | 修改者自行宣告成功 | Independent Review |
| Uncontrolled Evolution | 長期修改造成架構失控 | Versioning 與 Human Approval |
| Audit Tampering | 修改歷史紀錄 | Audit Integrity Protection |

---

# 18. 與其他 AIA 文件的關係

| 文件 | 關係 |
|---|---|
| AIA-03 | 提供 Go Runtime 與既有執行環境 |
| AIA-08 | 提供自主運作循環 |
| AIA-09 | 提供主動性與目標演化 |
| AIA-10 | 提供長期專案連續性 |
| AIA-11 | 定義自我修正與學習 |
| AIA-12 | 定義跨 Agent 委派與協作 |
| AIA-13 | 評估修改後的自主能力 |
| AIA-14 | 定義程式碼自主修改與安全演進 |

AIA-14 不取代 AIA-11。

AIA-11 回答：

「Agent 為什麼需要修正，以及如何從經驗中學習？」

AIA-14 回答：

「Agent 如何安全地修改程式碼，並證明修改確實有效？」

---

# 19. 最終設計原則

AIA 的長期自主性必須包含改善自身工作能力的可能性。

如果 Agent 能發現問題，卻永遠只能提出建議，無法在授權範圍內修改程式碼，那麼它的自我修正能力仍然受到限制。

但如果 Agent 可以任意修改自身核心、權限與治理機制，也會使整個系統失去可信任的控制邊界。

因此，AIA 必須建立清楚的區分：

- Agent 可以自主改善自己負責的 Domain Code。
- Agent 可以提出 AIA Core 的修改方案，並在隔離環境進行驗證。
- AIA Core 的正式修改與部署必須經人類核准。
- Governance 不得由 Agent 自行修改。
- 每次修改都必須可追溯、可驗證、可回滾。
- 每次修改都必須接受長期評估，而不是只看當下測試結果。

**Self-Modification 的真正目標，不是讓 AI 無限制地改寫自己，而是讓 AI 能夠在可信任的邊界內，持續改善自己的能力。**

AIA 最終應能形成：

Experience → Evaluation → Discovery → Correction → Modification → Verification → Learning → Evolution。

這才是自主系統從「使用工具完成工作」，逐步走向「能夠改善自身工作能力」的重要一步。
