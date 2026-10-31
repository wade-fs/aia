I Agent 開源專案盤點與技術驗證計畫

**文件代號：** AA-03  
**版本：** 1.0  
**文件狀態：** 初版盤點  
**建立日期：** 2026-10-31  
**所屬專案：** Personal AI Agent  
**前置文件：**
- AA-01：AI Agent 總體架構規格書
- AA-02：AI Agent 開源框架評估與整合設計

---

## 一、文件目的

本文件是 Personal AI Agent 專案的第一階段技術盤點文件。

目的不是立即決定所有技術，而是透過開源專案研究及 Proof of Concept（PoC），確認哪些能力可以直接採用，哪些需要封裝整合，哪些必須由本專案自行開發。

本階段優先研究：

1. LangGraph：工作流程引擎。
2. CrewAI：多 Agent 協作框架。
3. Aider：Coding Worker。
4. OpenHands：Coding Agent 平台。
5. Letta：持續性 Agent 與記憶架構。
6. Google ADK for Kotlin：Kotlin / Android Agent 開發框架。

### 1.1 核心原則

- 不重新開發成熟的專業工具。
- 不因使用某個框架而改變總體架構。
- 不讓外部框架成為 Agent 全域狀態的唯一管理者。
- 優先使用可替換、可獨立測試的元件。
- 技術選擇必須有實際 PoC 結果。
- 先建立一個可靠的 Agent，再逐步擴充自主能力。

### 1.2 本階段的成功目標

完成盤點後，應能回答：

1. 是否需要 LangGraph？
2. 是否需要 CrewAI？
3. Aider 能否直接作為 Coding Worker？
4. OpenHands 是否比 Aider 更適合某些開發工作？
5. Letta 是否值得用來管理長期記憶？
6. Google ADK for Kotlin 是否能成為 Android Agent 的技術基礎？
7. 哪些核心功能應該由我們自己開發？

本階段不要求所有問題都得到採用結論；有充分理由暫緩或拒絕，也屬於有效成果。

---

## 二、候選專案總覽

| 專案 | 主要定位 | 初步採用方式 | 優先級 |
|---|---|---|---|
| Aider | Coding Worker | Direct Use / Wrapper | P0 |
| LangGraph | Workflow Engine | PoC 後決定 | P0 |
| Google ADK Kotlin | Kotlin Agent Framework | PoC 後決定 | P0 |
| Letta | Stateful Agent / Memory | 架構研究 | P1 |
| OpenHands | Coding Agent Platform | 與 Aider 比較 | P1 |
| CrewAI | Multi-agent Framework | 暫緩整合 | P2 |

優先級說明：

- P0：第一階段必須實際研究及測試。
- P1：第一階段完成初步研究，視結果進行 PoC。
- P2：目前不是必要依賴，先保留研究結果。

---

## 三、Aider：第一個 Coding Worker

Repository：

https://github.com/Aider-AI/aider

官方文件：

https://aider.chat/docs/

License：Apache-2.0。

### 3.1 專案定位

Aider 是成熟的 CLI AI Pair Programming 工具。

它已經具備：

- 程式碼理解。
- Repository Map。
- 多檔案程式碼修改。
- Git 整合。
- 程式碼差異檢視。
- Lint 與測試輔助。
- 多種 LLM Provider。
- Gemini API 整合。

Aider 的定位不是完整的 Personal Agent，而是專業 Coding Worker。

### 3.2 與本專案的整合方式

本專案不重新開發 Coding Agent 的核心編輯能力。

由 Coding Domain Agent 決定工作內容，再透過 Worker Adapter 啟動 Aider。

預期流程：

```text
Personal Agent
      |
      v
Coding Domain Agent
      |
      v
Task Manager
      |
      v
Aider Worker Adapter
      |
      v
Aider CLI
      |
      v
Gemini / Other LLM
      |
      v
Code Changes
      |
      v
Test / Review / Validation
      |
      v
Personal Agent
```

### 3.3 PoC 測試項目

**測試 A：基本執行**

- 由程式啟動 Aider。
- 指定 Repository。
- 指定工作目錄。
- 指定 LLM Provider。
- 傳入開發需求。
- 取得執行結果。

**測試 B：程式碼修改**

- 修改單一檔案。
- 修改多個檔案。
- 新增檔案。
- 刪除檔案。
- 檢查 Git Diff。
- 驗證是否產生預期修改。

**測試 C：自動化執行**

- 非互動模式執行。
- 取得標準輸出及錯誤輸出。
- 設定執行期限。
- 處理程序異常終止。
- 取得結束代碼。
- 確認執行結果是否完整。

**測試 D：工作隔離**

- 每個 Task 使用指定 Workspace。
- 使用獨立 Git Branch 或 Worktree。
- 不允許任意修改其他 Project。
- 不允許讀取未授權的檔案。
- 不允許取得 Runtime 的秘密憑證。

**測試 E：錯誤恢復**

- Aider 執行中被終止。
- Runtime 重新啟動。
- 檢查檔案是否已被部分修改。
- 檢查 Git 狀態。
- 確認能否安全繼續或重新執行。

特別注意：

Aider 可能已經修改檔案或產生 Git Commit，但 Runtime 尚未收到完成訊息。

因此，不能單純因為 Task 狀態仍是 RUNNING，就重新執行同一個任務。

必須先檢查 Workspace、Git 狀態及 Worker 執行紀錄。

### 3.4 Gemini 整合

Aider 官方文件提供 Gemini API Key 設定方式。

例如：

```bash
export GEMINI_API_KEY="YOUR_API_KEY"

cd /path/to/project

aider --model gemini
```

實際使用的模型名稱、版本及 API 支援情況，以執行時 Aider 和 Gemini 官方文件為準。

API Key 應由 Worker 的安全設定提供，不寫入 Task、Prompt、Git Repository 或一般日誌。

### 3.5 初步決策

**建議列為第一個實際整合的 Coding Worker。**

採用方式：

Direct Use + Wrapper。

暫時不 Fork Aider。

如果 PoC 證明 Aider 可以可靠地接受任務、執行修改、回報結果，並且能在隔離 Workspace 中工作，就沒有必要自行開發程式碼編輯引擎。

---

## 四、LangGraph：Workflow Engine

Repository：

https://github.com/langchain-ai/langgraph

官方文件：

https://docs.langchain.com/oss/python/langgraph/

License：MIT。

### 4.1 專案定位

LangGraph 是低階、具狀態的 Agent Workflow Orchestration Framework。

主要能力：

- Graph Workflow。
- State Management。
- Node Execution。
- Conditional Branching。
- Checkpoint。
- Interrupt / Resume。
- Human-in-the-loop。
- Fault-tolerant Execution。

LangGraph 的 Checkpointer 可以保存 Graph 執行狀態，讓流程在中斷後恢復。

但它保存的是 Workflow 執行狀態，不等於本專案所有 Project、Task、Memory 及 Agent 生命週期的完整資料庫。

### 4.2 本專案的疑問

我們真正需要的基本能力包括：

- Project 管理。
- Task 管理。
- Task Dependency。
- Event 管理。
- Scheduler。
- Worker Execution。
- State Persistence。
- Recovery。

這些功能可以由一般程式設計實作。

因此，必須確認引入 LangGraph 是否真的比自行實作更有價值。

### 4.3 PoC 測試

設計一個 Coding Workflow：

```text
START
  |
  v
Analyze Requirement
  |
  v
Create Development Plan
  |
  v
Execute Coding Task
  |
  v
Run Tests
  |
  +---- PASS ----> Review ----> COMPLETE
  |
  +---- FAIL ----> Fix Code
                      |
                      +------> Run Tests
```

測試：

1. Graph 能否保存執行狀態？
2. 中斷後能否恢復？
3. 節點失敗時能否避免重複執行已完成工作？
4. 是否支援等待人工確認？
5. 是否能呼叫外部 Aider Worker？
6. 是否能由外部 Runtime 管理 Graph 執行？
7. Graph 狀態是否可以獨立保存及檢查？
8. 是否容易建立自有 Adapter？

### 4.4 與自行實作比較

同一個工作流程，分別使用：

- 自有簡單 Workflow Executor。
- LangGraph。

比較：

| 項目 | 自行實作 | LangGraph |
|---|---|---|
| 基本流程 | 必須實作 | 已提供 |
| 條件分支 | 必須實作 | 已提供 |
| Checkpoint | 必須實作 | 已提供 |
| 中斷恢復 | 必須實作 | 已提供 |
| 人工介入 | 必須實作 | 已提供 |
| 技術自由度 | 高 | 依賴 Graph 模型 |
| 初期成本 | 需估算 | 需整合及學習 |
| 長期維護 | 自行負責 | 同時負責框架整合及升級 |

### 4.5 初步決策

**不列為 Runtime 的必要依賴。**

先完成 PoC，再決定是否用於複雜 Workflow。

即使採用 LangGraph，Project、Task、Event 及 Agent 生命週期仍由本專案掌握。

---

## 五、Google ADK for Kotlin：Kotlin / Android 候選

Repository：

https://github.com/google/adk-kotlin

官方文件：

https://google.github.io/adk-docs/

### 5.1 專案定位

Google ADK for Kotlin 是以 Kotlin 為基礎的 Agent Development Kit。

值得研究的能力：

- Agent Definition。
- Tool Integration。
- Agent Orchestration。
- Session Management。
- Kotlin / Java 整合。
- Android 擴充。
- Room 持久化。
- Cloud 與 On-device Agent。

截至本文件建立日期，Google 已於 2026 年 9 月宣布 ADK for Kotlin 1.0 GA。

這使它成為值得認真評估的 Kotlin 技術候選，而不只是早期實驗性框架。

### 5.2 為什麼重要？

本專案先前考慮 Kotlin，原因包括：

- Android 優先。
- 需要觸控及裝置整合。
- 需要語音輸入與輸出。
- 可能使用 Android 背景工作。
- 希望核心架構保留未來 Server 部署能力。

ADK Kotlin 值得研究是否能減少 Agent、Tool、Session 等基礎能力的開發成本。

### 5.3 PoC 測試

建立一個最小 Kotlin Agent：

1. 呼叫 LLM。
2. 註冊 Tool。
3. 呼叫 Tool。
4. 保存 Session。
5. 關閉 App。
6. 重新啟動 App。
7. 恢復 Session。
8. 繼續執行工作。

再測試：

- Room Session Persistence。
- Android Background Execution。
- Coroutine / Flow。
- 不同 LLM Provider。
- Tool Calling。
- 獨立於 Activity 的核心邏輯。
- 未來 Server 部署的可能性。

### 5.4 評估限制

必須確認：

- 是否與本專案的 Memory Manager 相容。
- 是否能替換 LLM Provider。
- 是否會形成 Google 生態依賴。
- 是否適合長時間背景執行。
- 是否能與自有 Runtime 分離。
- Android 與 Server 是否能共用主要業務邏輯。

### 5.5 初步決策

列為 P0 技術驗證對象。

不預設整個 Personal Agent 必須建立在 ADK Kotlin 上。

應先確認它能否提供可重用的 Agent / Tool / Session 能力。

---

## 六、Letta：持續性 Agent 與記憶架構

Repository：

https://github.com/letta-ai/letta

官方文件：

https://docs.letta.com/

License：Apache-2.0。

### 6.1 專案定位

Letta 專注於 Stateful Agent、Persistent Memory 及長期記憶管理。

值得研究：

- Agent State。
- Long-term Memory。
- Memory Organization。
- Memory Update。
- Memory Consolidation。
- Background Memory Processing。
- Agent Identity。

### 6.2 與本專案的關係

我們的 Agent 需要同時管理多個長期專案。

例如：

**Coding Project**

- 專案架構。
- 程式語言。
- 程式規範。
- 已完成工作。
- 已知問題。
- 技術決策。

**Writing Project**

- 世界觀。
- 人物設定。
- 年表。
- 章節進度。
- 史實考證。
- 已確認的創作原則。

這些資訊不能全部塞進單一 LLM Context。

必須有獨立的記憶管理系統。

### 6.3 PoC 測試

建立兩個獨立 Project：

- Coding Project。
- Writing Project。

測試：

1. 能否保存不同 Project 的記憶？
2. 能否在不同 Project 間切換？
3. 能否找回先前決策？
4. 能否區分事實與推論？
5. 能否更新或修正錯誤記憶？
6. 能否追溯記憶來源？
7. 能否在不同 LLM 間延續 Agent 狀態？
8. 是否支援記憶整理及背景處理？

### 6.4 初步決策

優先研究記憶架構。

暫不決定直接採用 Letta 作為唯一 Memory Manager。

本專案仍需掌握：

- 記憶資料模型。
- 記憶來源。
- 記憶版本。
- 記憶可信度。
- 記憶修正。
- Project 隔離。
- 記憶匯出與遷移。

---

## 七、OpenHands：Coding Agent 替代方案

主要 Repository：

https://github.com/OpenHands/OpenHands

SDK Repository：

https://github.com/OpenHands/software-agent-sdk

License：MIT（以各實際採用 Repository 的 License 為準）。

### 7.1 專案定位

OpenHands 不只是程式碼編輯工具，也提供更完整的 Software Development Agent 平台。

值得研究：

- Agent SDK。
- Agent Server。
- REST API。
- WebSocket。
- Workspace。
- Conversation。
- Event。
- Tool。
- Execution Environment。
- Automation。

### 7.2 與 Aider 的差異

| 項目 | Aider | OpenHands |
|---|---|---|
| 定位 | CLI Coding Agent | Coding Agent Platform |
| 程式碼修改 | 支援 | 支援 |
| Git 整合 | 直接整合 | 依工作環境及工具 |
| 執行環境 | 本機 Repository | Workspace / Agent Environment |
| API | CLI 為主要入口 | 提供 Agent Server API |
| 工作事件 | CLI 輸出及執行狀態 | Event / REST / WebSocket |
| 系統複雜度 | 相對簡單 | 相對完整 |

### 7.3 PoC 測試

1. 建立 Coding Task。
2. 建立獨立 Workspace。
3. 啟動 Agent Server。
4. 透過 API 建立 Conversation。
5. 傳送開發需求。
6. 接收執行事件。
7. 取得產出檔案。
8. 檢查 Git Diff。
9. 終止及恢復工作。
10. 評估資源需求及隔離能力。

### 7.4 特別注意

OpenHands Agent Server 的 API、Workspace、Event 等能力值得研究。

但必須確認：

- 執行環境是否真的隔離。
- 是否使用 Docker 或其他沙盒。
- 權限是否能限制在指定 Workspace。
- API 是否適合由 Kotlin Runtime 呼叫。
- Python 執行環境是否增加部署成本。

### 7.5 初步決策

先作為 Aider 的替代方案。

不在第一階段同時整合 Aider 與 OpenHands，以免增加不必要的維護工作。

---

## 八、CrewAI：多 Agent 協作候選

Repository：

https://github.com/crewAIInc/crewAI

官方文件：

https://docs.crewai.com/

License：MIT。

### 8.1 專案定位

CrewAI 提供：

- Agent。
- Task。
- Crew。
- Flow。
- Role-based Collaboration。
- Event-driven Workflow。

其中 Crew 著重 Agent 協作，Flow 著重流程控制。

### 8.2 為什麼暫緩？

我們目前還沒有證明需要多個 Agent 同時協作。

例如，寫一本書可能需要：

- 歷史研究。
- 人物分析。
- 年表校對。
- 章節撰寫。
- 內容審查。

但這不代表必須建立五個長期存在的 Agent。

也可能由同一個 Writing Domain Agent 依任務呼叫不同 Worker 或 LLM。

多 Agent 會增加：

- Token 成本。
- 任務協調成本。
- 記憶同步問題。
- 錯誤傳播風險。
- 執行時間。
- 系統複雜度。

### 8.3 初步決策

P2，暫不列入第一階段必要依賴。

等實際工作證明單一 Domain Agent 無法有效完成，再評估 CrewAI。

---

## 九、第一階段技術架構假設

本階段先採用以下架構作為 PoC 方向，不視為最終實作決策。

```text
                Personal Agent
                       |
                       v
              Own Agent Runtime
                       |
        +--------------+--------------+
        |              |              |
        v              v              v
   Project Mgr     Task Mgr       Memory Mgr
        |              |              |
        +--------------+--------------+
                       |
                       v
                Worker Manager
                       |
          +------------+------------+
          |                         |
          v                         v
     Coding Agent              Other Workers
          |
          v
    Aider Adapter
          |
          v
       Aider CLI
          |
          v
     Gemini / LLM
```

Workflow Engine 可以在 Worker Manager 與實際 Worker 之間提供複雜流程管理，但不是必要的固定層。

### 9.1 必須自行掌握的核心

| 核心模組 | 是否自行掌握 |
|---|---|
| Agent Runtime | 是 |
| Project Manager | 是 |
| Task Manager | 是 |
| Event Manager | 是 |
| Scheduler | 是 |
| State Manager | 是 |
| Memory Manager | 是 |
| Permission Manager | 是 |
| Policy Engine | 是 |
| Recovery Manager | 是 |
| Worker Adapter | 是 |
| Coding Engine | 優先採用 Aider / OpenHands |
| Workflow Engine | 視 PoC 結果 |
| LLM SDK | 優先採用成熟 SDK |

---

## 十、PoC 執行計畫

### P0-A：Aider 整合

目標：證明我們的 Runtime 能夠管理外部 Coding Worker。

驗收條件：

- 可以自動啟動。
- 可以指定工作目錄。
- 可以傳送任務。
- 可以取得執行結果。
- 可以檢查修改內容。
- 可以執行測試。
- 可以處理程序異常。
- 可以保留完整執行紀錄。

### P0-B：Kotlin Agent 基礎測試

目標：確認 ADK Kotlin 是否適合 Android 技術路線。

驗收條件：

- Kotlin Agent 可正常執行。
- Tool Calling 可正常運作。
- Session 可以持久化。
- App 重啟後可以恢復。
- 核心邏輯不依賴 Activity。
- LLM Provider 有替換可能。

### P0-C：LangGraph 比較測試

目標：確認使用 LangGraph 是否比自行實作簡單可靠。

驗收條件：

- 完成同一個多步驟工作流程。
- 比較中斷恢復。
- 比較狀態管理。
- 比較實作成本。
- 比較執行成本。
- 確認是否值得引入。

### P0-D：記憶架構研究

目標：確認 Letta 的記憶模型是否值得採用或借鑒。

驗收條件：

- Coding / Writing 專案隔離。
- 長期記憶可持續保存。
- 記憶可查詢。
- 記憶可修正。
- 記憶來源可追溯。

### P0-E：技術選型報告

彙整各 PoC 結果，決定：

- 採用。
- 封裝。
- Fork。
- 僅參考。
- 暫緩。
- 不採用。

每個決策都必須附上實際測試結果。

---

## 十一、通用 PoC 安全要求

所有外部 Worker 均須遵守：

1. 不得直接取得 Runtime 的資料庫管理權限。
2. 不得直接讀取其他 Project 的工作資料。
3. 不得取得未授權的 API Key。
4. 不得自行擴大檔案操作範圍。
5. 必須有執行期限。
6. 必須有取消機制。
7. 必須回報執行狀態。
8. 必須保留執行紀錄。
9. 必須能檢查輸出結果。
10. 不得因失敗而無限制重試。

Coding Worker 尤其必須在指定 Workspace 中工作。

外部 Worker 回報成功，不等於 Runtime 已經驗收成功。

---

## 十二、開源授權及版本紀錄

每個正式採用的專案，必須建立第三方元件紀錄。

至少保存：

| 欄位 | 說明 |
|---|---|
| Project Name | 專案名稱 |
| Repository | 官方 Repository |
| Version | 採用版本 |
| Commit | 採用 Commit |
| License | 授權 |
| Dependencies | 主要相依套件 |
| Modified | 是否修改 |
| Adapter | 使用的整合介面 |
| Security Review | 安全審查 |
| Decision | 採用決策 |

本文件列出的 License 是初步盤點資訊。

正式採用時，仍須檢查實際版本的 LICENSE、相依套件及第三方服務條款。

---

## 十三、第一階段預期成果

完成本文件規劃後，應產生：

```text
03_open_source/
|
+-- opensource_inventory.md
|
+-- evaluation/
|   +-- aider_evaluation.md
|   +-- langgraph_evaluation.md
|   +-- adk_kotlin_evaluation.md
|   +-- letta_evaluation.md
|   +-- openhands_evaluation.md
|   +-- crewai_evaluation.md
|
+-- poc/
|   +-- aider_poc/
|   +-- adk_kotlin_poc/
|   +-- langgraph_poc/
|
+-- decisions/
    +-- technology_decisions.md
    +-- third_party_components.md
```

PoC 原始碼與正式 Runtime 必須分開。

PoC 的目的在於驗證技術，不應在尚未通過審查前直接成為正式核心程式碼。

---

## 十四、文件結論

本專案的第一個實際整合目標是 Aider。

原因是 Coding Agent 的需求明確，而且 Aider 已經提供成熟的程式碼編輯、Git 整合及 LLM Provider 能力。

第二個重要方向是 Kotlin / Android Agent 基礎能力，Google ADK for Kotlin 值得優先驗證。

LangGraph 則必須透過實際比較，證明它比自行實作簡單流程更有價值，才決定是否採用。

Letta 作為長期記憶架構的研究對象。

OpenHands 作為更完整 Coding Agent 平台的比較對象。

CrewAI 暫時不引入，避免過早增加多 Agent 複雜度。

**我們要自己擁有 Agent 的管理權，但不必自己完成 Agent 所有專業工作。**

第一階段的真正成果，不是寫出多少程式碼，而是建立一個可以管理外部 Worker、保存工作狀態、處理中斷並驗收結果的通用 Agent 雛形。
