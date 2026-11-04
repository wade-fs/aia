### **第二層：有服務，但尚未形成完整自主行為**

優先整合

我認為這是目前最大的落差。

例如 AIA-22：

現在已經有：

* Problem Model。  
* Analysis Record。  
* Problem Analysis。  
* Dependency Analysis。  
* Plan。  
* Plan Change。  
* Impact Analysis。  
* CLI / REST。

但這些能力存在，不代表 AIA 遇到複雜問題時就會自動啟動它們。

真正需要驗證的是：

```
使用者提出問題
       |
       v
Autonomy Core
       |
       v
判斷是否為複雜問題
       |
       v
啟動 Problem Analysis
       |
       v
調查現有文件與專案
       |
       v
建立 Problem Model
       |
       v
發現未知與缺口
       |
       v
研究 / 釐清
       |
       v
比較解決方案
       |
       v
產生 Plan
       |
       v
Governance
       |
       v
建立 Tasks
       |
       v
Worker 執行
       |
       v
驗證成果
       |
       v
Reassess
```

如果這個流程仍然需要使用者逐一呼叫 CLI 指令，或者必須由某個固定 Prompt 指定執行步驟，那麼它還沒有真正完成 AIA-22 的目標。

### **3.3 第三層：跨 Session 的工作延續**

需要端到端驗證

這是我們最初建構 AIA 的核心原因之一。

假設我告訴 AIA：

「請分析我的小說，確認整本書是否符合既定規範。」

AIA 應該能夠：

1. 自行找到小說 Project。  
2. 載入 Manifest、目錄、Master Bible。  
3. 找出已完成的分析。  
4. 找出尚未完成的工作。  
5. 判斷目前最值得執行的工作。  
6. 建立或恢復 Plan。  
7. 執行下一批工作。  
8. 保存成果及證據。  
9. 下次啟動時繼續。

這裡有一個關鍵：

Continuity 不只是記得以前做過什麼，而是能夠根據以前的工作狀態，正確決定現在應該做什麼。

現有 `continuity` 與 `recovery` 已經提供基礎，但還需要確認 Problem、Plan、Brief、Clarification、Task、Artifact 是否真的能一起恢復並繼續執行。

### **3.4 第四層：自主性是否有客觀證據**

需要補強

AIA-13 不應只是一份統計報表。

它應該回答：

* AIA 能否自行發現值得處理的問題？  
* 能否判斷什麼時候不應該行動？  
* 能否提出正確的下一步？  
* 能否在新證據出現時推翻舊結論？  
* 能否避免重複做已經完成的工作？  
* 能否在中斷後恢復正確的執行位置？  
* 自我修正是否真的改善結果？

這些指標必須能夠跨 Investment、Coding、Writing 等 Domain 使用。

不能只用投資的 Brier Score 代表整個 AIA 的自主能力。

## **四、我在實際程式碼中發現的關鍵問題**

這部分比文件上的完成率更值得注意。

我進一步檢查了 `cycle/coordinator.go`、`problem/service.go`、`brief/service.go` 和 `app/app.go`。

### **4.1 AIA-22 已經接入 Autonomy Core，但還不是完整的智慧分析**

這是一個重要的發現。

`internal/autonomy/cycle/coordinator.go` 已經：

* 接收 Problem Service。  
* 從 Initiative 觸發 Problem 建立。  
* 建立 Problem Model。  
* 產生 Plan。  
* 將 Work Items 轉成 Tasks。  
* 交給既有執行架構。

因此，不能再說 AIA-22 完全沒有串接 Autonomy Core。

但是，我發現一個更深層的問題。

在 `executeAutonomousProblemOrPlan()` 裡面，目前有一段固定產生問題模型的程式：

```
analysis := &domainProblem.AgentProblemAnalysis{
    Intent: domainProblem.IntentSummary{
        UserGoal: prob.Objective,
        SuccessCriteria: []string{
            "Restore operational stability",
            "Remediate failure root causes",
        },
    },
    // ...
}
```

接著直接產生固定的兩階段計畫：

```
1. Diagnosis and Root-Cause Isolation
2. Verified Implementation
```

甚至預設：

```
wiAgent = "coding-agent"
```

這代表什麼？

目前這條 Autonomous Problem 路徑，雖然已經串接 Problem、Plan、Task，但它仍然使用固定的問題模型與固定的工作分解方式。

它並沒有真正根據不同問題，讓 agy 自行完成 AIA-22 所要求的：

* Intent Understanding  
* Context Investigation  
* Problem Modeling  
* Gap Analysis  
* Solution Exploration  
* Alternative Evaluation  
* Recursive Decomposition  
* Plan Generation

這是一個非常重要的落差。

目前的架構已經具備讓 Agent 思考的容器，但這條整合路徑還沒有真正把思考能力交給 Agent。

### **4.2 AIA-21 的 Brief 與 Clarification 也有類似問題**

我確認了：

`internal/autonomy/brief_observer.go`

已經能夠觀察：

* Brief 欄位缺漏。  
* 決策衝突。  
* 過期假設。  
* 專案是否達到下一階段的準備程度。

而且它遵守一個正確的設計原則：

Observer 只提出 Observation，不直接修改 Brief 或詢問使用者。

但還需要驗證後續流程：

```
Brief Observer
      |
      v
Observation
      |
      v
Initiative Discovery
      |
      v
Deliberation
      |
      v
Decision
      |
      v
Clarification Service
      |
      v
等待使用者回答
      |
      v
更新 Brief
      |
      v
重新評估
```

目前 Brief、Clarification Service 已經存在，但必須確認 Autonomy Core 能否真正根據 Observation 自主產生適當的釐清問題，而不是只建立一張待處理工作單。

### **4.3 Writing Agent 仍需區分「發現問題」與「解決問題」**

我看到 `writing_observer.go` 已經能夠檢查小說的一致性，並且區分：

* 真正的歷史或時間線矛盾。  
* 合理的虛構。  
* 不需要處理的正常狀態。

這是正確的。

但是 `WritingPilotExecutor.Execute()` 目前主要是產生 revision proposal 的摘要，並沒有在這個 Executor 中完成真正的修訂。

這表示：

Observer 已經有一定的專業判斷能力，但 Domain Agent 的自主處理閉環還需要進一步驗證。

Coding、Writing、Investment 三個 Domain 不應該只是共用同一個固定 Problem Plan。

它們應該共用 Problem Analysis Framework，但各自提供專業知識、工具、驗收條件及執行方式。

### **4.4 AIA-14 的安全自修仍然需要完整驗證循環**

目前 Self-Modification 已有：

* Git Worktree 隔離。  
* Sandbox。  
* Scope Classification。  
* 多級治理。  
* 測試驗證。  
* Rollback。

但規格還要求：

```
提出修改假設
      |
      v
建立修改方案
      |
      v
Sandbox 執行
      |
      v
測試
      |
      +---- 失敗 ----> 分析原因
      |                   |
      |                   v
      |                修正方案
      |                   |
      +-------------------+
      |
      v
獨立驗證
      |
      v
人工批准（適用時）
      |
      v
合併或回滾
```

這個流程不能只用一次測試回呼來代表完成。

建議把 AIA-14 的多級驗證、失敗重試、重試上限及回滾行為納入正式驗收。

## **五、文件本身也有版本不同步問題**

這次我特別注意到，`docs/` 裡面的文件並非全部同步更新。

以下是我確認到的具體矛盾。

| 文件 | 現有內容 | 問題 |
| ----- | ----- | ----- |
| `README.md` | Phase 0～5 完成，Phase 6 Planned | 落後目前實作 |
| `go/README.md` | Migration 001～012 | 實際已到 019 |
| `docs/development/migration-plan.md` | Active Baseline 001～012 | 落後實際 Migration |
| `docs/development/roadmap.md` | Phase 7 Autonomy Integration Next | 沒有完整反映目前已存在的 Coordinator 整合 |
| `docs/development/implementation-status.md` | Phase 7 Data & Services Complete | 對端到端行為的完成程度描述不足 |
| `docs/architecture/architecture-to-code-mapping.md` | AIA-22 核心就緒 85% | 與 Roadmap 的 Phase 7 描述不完全一致 |
| `docs/phase6-readiness-assessment.md` | Phase 6 准入評估 | 屬於歷史性文件，不應當作目前整體狀態 |

另外還有幾個值得修正的地方：

AIA-19

Mapping 文件仍將備份還原列為待實作，但目前已經有：

* `internal/service/backup/service.go`  
* `CreateBackup()`  
* `VerifyBackup()`  
* `RestoreBackup()`  
* CLI backup 指令

而且 `app.go` 已經有 Backup Service 的初始化。

日誌保留清理也已經有部分實作。因此，AIA-19 的 65% 評估需要重新檢查，不能沿用舊結論。

AIA-20～22

規格文件的狀態仍然標示 `Planned Architecture`，但 Development 文件已經宣稱部分功能完成。

建議將規格狀態區分為：

* Specification：規格是否正式核定。  
* Implementation：程式碼實作狀態。  
* Verification：行為驗收狀態。

不要再用一個 `Production Ready` 同時代表這三件事。

我的判斷：目前最大的文件問題不是規格不足，而是不同文件對同一份程式碼有不同的認知。

這種情況如果不先修正，後面容易出現重複實作、錯誤的 Phase 規劃，甚至破壞已經成熟的架構。

## **六、下一步建議：不要急著開 Phase 8**

我建議把目前工作定位為：

# **Phase 7 — Autonomous Problem Solving Integration**

整合與行為驗收階段

目標：讓已經實作的 AIA-08～22 真正形成跨領域、可持續、可恢復的自主工作系統。

不新增一套 Autonomy Core，不重寫 Runtime，也不急著擴充更多 Domain。

我建議分成五個工作包。

### **P7-1：文件與實作基準統一**

優先級 P0

先處理文件。

| 工作 | 交付成果 |
| ----- | ----- |
| 更新 Root README | 反映 Phase 0～7 現況 |
| 更新 Go README | Migration 001～019、完整模組 |
| 更新 Migration Plan | 反映實際 Migration 版本 |
| 更新 Implementation Status | 區分實作與驗收 |
| 更新 Architecture Mapping | 重新盤點 AIA-00～22 |
| 更新 Roadmap | 明確列出 Phase 7 尚未完成的整合項目 |

另外，建立一份統一的 Implementation Evidence Matrix：

| Spec | Package | Service | API/CLI | Integration Test | E2E Verified |
| ----- | ----- | ----- | ----- | ----- | ----- |
| AIA-21 | 有 | 有 | 有 | 有待核對 | 待驗證 |
| AIA-22 | 有 | 有 | 有 | 有 | 待驗證 |
| AIA-13 | 部分 | 部分 | 待核對 | 部分 | 未完成 |
| AIA-14 | 有 | 有 | 有 | 有 | 部分 |

這份矩陣應該成為後續判斷完成度的唯一依據之一。

### **P7-2：真正的 Problem Analysis Engine**

優先級 P0

這是我認為目前最值得投入的工程工作。

需要將目前固定的 Problem Model 與 Plan 產生方式，改成真正的 LLM 驅動分析流程。

建議新增一個共用服務：

```
internal/problem/
    analyzer.go
    investigator.go
    model_builder.go
    gap_analyzer.go
    solution_explorer.go
    plan_generator.go
    acceptance_designer.go
```

實際目錄可以依照現有分層原則調整，不必為了這個建議強制新增 Package。

核心流程：

```
User Intent
     |
     v
Intent Analysis
     |
     v
Context Investigation
     |
     v
Problem Model
     |
     v
Gap / Conflict Analysis
     |
     +---- Need Clarification
     |          |
     |          v
     |    AIA-21 Clarification
     |          |
     |          v
     |    Update Problem Model
     |          |
     +----------+
     |
     v
Solution Exploration
     |
     v
Alternative Evaluation
     |
     v
Plan Generation
     |
     v
Acceptance Criteria
     |
     v
AIA Governance
     |
     v
Plan Execution
```

注意責任分工：

* agy：分析、推理、提出方案。  
* Go：驗證結構、來源、狀態、依賴與權限。  
* SQLite：保存模型、分析、計畫與歷史。  
* Worker：執行 Task。  
* Autonomy Core：決定何時啟動、暫停及重新分析。

這樣才能避免 LLM 自己產生一份文字計畫，Go 卻無法驗證及執行的問題。

### **P7-3：Brief、Problem、Plan、Task 的完整串接**

優先級 P0

目前這幾個實體各自已經有模型及服務。

現在應該把它們連成一條真正可運作的資料鏈：

Goal

為什麼要做？最終要達成什麼？

Project Brief

需求、範圍、限制、成功條件

Problem / Problem Model

問題結構、事實、假設、未知、風險

Plan / Work Items

可執行的工作 DAG 與驗收條件

Task / Execution

Worker 派發、執行與結果保存

Verify / Learn / Reassess

驗收、證據、修正與重新規劃  
以上各層均應保留版本、來源及相互關聯。

這裡還有一個重要要求：

當 Problem Model 改變時，AIA 必須知道哪些 Plan、Work Items 和 Tasks 受到影響。

不能直接把舊計畫覆蓋掉。

應該使用 Plan Change 機制，保留舊版本、影響分析與重新批准紀錄。

### **P7-4：跨 Session Continuation**

優先級 P1

設計三個正式驗收場景。

| 場景 | 驗收條件 |
| ----- | ----- |
| Session 中斷 | 重啟後恢復原本的 Problem、Plan 與執行位置 |
| 工作中發現新問題 | 更新 Problem Model，重新評估受影響的 Plan |
| 使用者修改需求 | 更新 Brief，重新評估 Goal、Problem 與工作計畫 |

特別注意：

AIA 不應只是恢復 Task 狀態，而應該恢復整個 Project 的工作脈絡。

### **P7-5：通用自主性評估與介面補強**

優先級 P1

分成兩條：

AIA-13：

建立跨 Domain 的自主性評估指標：

* Initiative Detection Accuracy  
* Decision Quality  
* No-Action Precision  
* Continuity Recovery Accuracy  
* Plan Completion Rate  
* Replanning Effectiveness  
* Self-Correction Effectiveness  
* Human Intervention Rate

AIA-06：

補齊 Web Console：

* Goal View  
* Autonomy Cycle View  
* Problem Analysis View  
* Plan / Work Item DAG View  
* Clarification View  
* Emergency Stop 操作介面

後端已有不少 API，不需要重寫後端才能開始。

## **七、最重要的是：用真實專案驗證，而不是繼續用假設場景**

我建議選三個實際工作場景。

[Comment écrire son premier roman : guide complet pour débutants](https://images.openai.com/static-rsc-4/Qf_KHl__whcG965Sz24Irp4IUhqcWwWaPmkqjqB_K0u33K4Yo_ypb8t6EMv6nKoasa3gz5DFXJaRdxSMF8lKXhUbavSMA6N8H89zoBY2zcptgFJqjudYJQZmoOwZYFHLoMIIP3BQpeS7kYXGTy5mMcRpOqMy5sCnob_AK8XYe8c?purpose=inline)

Scenario A：Writing Agent

使用者提供一本書的目錄及相關規範，要求檢查整本書是否符合規範。

驗證：

* 自行理解目錄與文件。  
* 建立問題模型。  
* 遞迴拆解稽核工作。  
* 保存每卷、每章的進度。  
* 中斷後繼續。

[Recruiting beim DigitalService – Was Bewerbende erwartet \| DigitalService](https://images.openai.com/static-rsc-4/nndPEm4swyRDYCtCPtXr7yYb5lZYcp-VSvujp1gS01tihj8BKWJz181I4LgbMCTQHbRf7kn9aNBry7vJziTwTIMyFP_OdEjRa1gFT-DNn-1iUOwK2bi2kooYuKWCA_R7LoC5tb4lF5lxk_IvNAJsTSTxWJeAtRSx4iAYfNqx39k?purpose=inline)

Scenario B：Coding Agent

要求分析一個 Go Repository 的架構缺陷，提出改善方案。

驗證：

* 自行探索 Repository。  
* 找出架構問題及依賴。  
* 比較替代方案。  
* 產生 Plan。  
* 執行測試及驗收。

[Upstox Review 2026 \| Charges, Margins & Account Opening \| Top 20 Stock Brokers](https://images.openai.com/static-rsc-4/h5zOhqVecy1Pl9WUoH8vokKg1GzoQskGNk4_rlvFzidlhaVJDHWtxFRak9mZ4S3ebBbIlQJIwkMYWLa4ZteTeG1378xUCPehlbWC4Nf57zTT7Q2WcFZFy6znLtPZtg4IrElTakKePYrwZOq_xf0TiZk4v6YBl3gFpUDUXZ-uL68?purpose=inline)

Scenario C：Investment Agent

使用 NT\$100,000 虛擬資金，根據市場變化調整持股。

驗證：

* 自主觀察。  
* 研究與證據溯源。  
* BUY / SELL / HOLD。  
* Go 驗證交易規則。  
* 長期績效評估與策略修正。

這三個場景很重要，因為它們可以證明：

AIA 是否真的有一套共用的智慧核心，而不是三套各自獨立、依賴固定 Prompt 的 Agent。

## **八、我的最終判斷**

我把這次分析濃縮成一張表：

| 項目 | 現況 | 下一步 |
| ----- | ----- | ----- |
| 基礎 Runtime | 成熟 | 保持穩定 |
| Continuity | 已有機制 | 驗證完整工作恢復 |
| Autonomy Core | 已有 11 步驟 | 強化複雜問題推理 |
| Problem Analysis | 有服務與初步整合 | 改為真正的 LLM 分析 |
| Brief / Clarification | 已有服務 | 串接自主釐清閉環 |
| Plan Management | 已有版本化機制 | 加強動態重新規劃 |
| Domain Agents | 三個主要 Domain 已有基礎 | 驗證共用核心 |
| Self-Correction | 已有機制 | 驗證長期效果 |
| Self-Modification | 已有安全管線 | 完成多級修正循環 |
| Evaluation | 有部分指標 | 建立通用自主性基準 |
| Web Console | 後端較成熟 | 補齊自主工作視圖 |

我不建議現在增加 AIA-23、AIA-24 等新規格，也不建議開始大規模重構。

先把 AIA-00～22 已經定義、但尚未形成完整行為的能力落實。

另外，這次我嘗試執行 `go test ./...`，但執行環境要求下載 Go 1.26.2 Toolchain，因為目前環境無法連接 Go Proxy，所以沒有完成實際測試。以上程式判斷是靜態檢查，不應誤認為這次已重新驗證所有測試通過。

## **九、我建議現在就做的下一件事**

不是先寫新程式，而是先建立一份正式文件：

`AIA-23 — Autonomous Problem Solving Integration & End-to-End Verification Specification`

但我會把它定位成整合驗收規格，而不是再創造一套新的智慧架構。

它應該明確定義：

1. AIA-08～22 的實際整合責任。  
2. agy 在 Problem Analysis 中的推理責任。  
3. Brief → Problem → Plan → Task 的資料流。  
4. Clarification 與 Governance 的互動。  
5. Plan 動態修正及版本管理。  
6. 跨 Session Continuation。  
7. Writing / Coding / Investment 三個端到端驗收場景。  
8. 自主性評估及驗收門檻。