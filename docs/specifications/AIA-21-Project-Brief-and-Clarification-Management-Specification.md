# AIA-21 — Project Brief & Clarification Management Specification

**文件代號：** AIA-21  
**文件名稱：** Project Brief & Clarification Management Specification  
**系統名稱：** AIA — Autonomous Intelligence Architecture  
**文件狀態：** 正式規格 (Normative Specification) · 程式碼實作完成 (Implemented in Migrations `016` & `017`)  
**版本：** v1.0  
**適用範圍：** AIA Core / Project Management / All Domain Agents  
**依賴規格：** AIA-01、AIA-02、AIA-03、AIA-08、AIA-09、AIA-10、AIA-16、AIA-17、AIA-18、AIA-20  
**相關後續規格：** AIA-22 Complex Problem Analysis & Planning、AIA-23 Research & Evidence Lifecycle、AIA-24 Project Workspace & Execution Integration（待整體規格交叉檢查後確認編號）

---

# 1. 文件目的

## 1.1 核心問題

使用者提出一個新目標時，通常不會提供完整規格。

例如：

> 我想寫一本關於計劃徒步環台灣本島一圈的計畫的書，請幫我規劃這本書。

這個需求至少包含兩個不同層次：

1. **實際計畫：** 如何完成徒步環台灣本島一圈。
2. **出版內容：** 如何把這個計畫整理成一本有價值、可閱讀、可執行的書。

其中仍有大量未知事項：

- 書籍是實用指南、旅行文學、個人計畫紀錄，還是綜合型作品？
- 讀者是一般民眾、徒步愛好者，還是有經驗的長途徒步者？
- 徒步者是否已經確定？
- 預計多少天完成？
- 每日步行距離是否有上限？
- 是否允許部分路段搭乘大眾運輸？
- 預算、住宿、補給、安全如何處理？
- 是否需要實際走過路線，還是先完成規劃型書籍？
- 書籍需要多深入的地理、歷史與地方文化資料？

如果 Agent 直接產生目錄，就等於在尚未理解目標的情況下，自行替使用者決定大量重要事項。

這不是自主智慧，而是過早生成。

## 1.2 AIA-21 的目標

AIA-21 建立一套持續性的需求理解機制，使 Agent 能夠：

1. 理解使用者提出的原始目標。
2. 建立可持續更新的 Project Brief。
3. 分析目標與現有資訊之間的落差。
4. 判斷哪些未知資訊需要向使用者詢問。
5. 判斷哪些資訊可以透過研究取得。
6. 對可合理預設的事項提出方案及理由。
7. 將問題、答案、假設及決策持續保存。
8. 根據新的資訊重新評估 Brief。
9. 在需求足夠明確時，主動提出下一階段工作。
10. 在專案進行中，遇到新的重要問題時再次啟動釐清流程。

**Brief 是持續演進的專案需求模型，不是一次性的需求表單。**

---

# 2. 設計原則

## 2.1 Agent 必須先理解，再提出方案

Agent 不應把第一個 Prompt 直接當成執行指令。

應先進行：

- 目標理解。
- 需求拆解。
- 未知事項識別。
- 衝突檢查。
- 影響分析。
- 研究可行性判斷。
- 釐清問題排序。

完成上述分析後，才決定下一步。

## 2.2 不要求使用者提供所有資訊

需求資訊分成三類：

| 類型 | 處理方式 |
|---|---|
| 使用者必須決定 | 向使用者詢問 |
| 可以透過研究取得 | Agent 主動研究 |
| 可以合理預設 | Agent 提出預設值與理由 |

例如，徒步環島的道路距離不應要求使用者猜測，應由 Agent 研究。

但是，使用者是否願意每天走 30 公里，不能由 Agent 擅自認定。

## 2.3 問題必須具有決策價值

Agent 不應為了補齊欄位而提問。

每個問題都必須能說明：

- 為什麼現在需要知道？
- 答案會影響哪些後續決策？
- 如果不知道，會產生什麼風險？
- 是否能先採用可逆的預設方案？

如果答案不會實質影響目前工作，就不必立即詢問。

## 2.4 假設不得偽裝成使用者決定

Agent 可以提出預設值，但必須保留其來源及信心程度。

例如：

```yaml
daily_mileage:
  value: 20
  unit: km
  status: DEFAULTED
  source: agent_recommendation
  confidence: 0.65
  rationale: >
    以一般休閒徒步者為初步規劃對象，
    先採用較保守的每日步行距離。
  user_confirmed: false
```

這不代表使用者已經同意每日走 20 公里。

## 2.5 Brief 必須可以反覆修正

新資訊可能推翻舊假設。

例如：

原先預估花蓮至台東可以連續步行完成，研究後發現某段道路不適合徒步，或住宿與補給條件不足。

Agent 必須能：

1. 記錄新資訊。
2. 找出受到影響的 Brief 欄位。
3. 重新評估原本的方案。
4. 判斷是否需要詢問使用者。
5. 更新工作計畫。

不得直接覆蓋舊值而失去歷史。

---

# 3. 系統架構與責任分工

## 3.1 架構位置

Brief 與 Clarification 屬於 AIA Core 的共用能力。

它們不能只存在於 Writing Agent，因為 Coding、Investment、Scientific Research 等 Domain 同樣需要理解需求及釐清未知事項。

建議架構：

```text
                    User
                     |
                     v
              Project Creation
                     |
                     v
              Project Registry
                     |
                     v
              Brief Management
                     |
             +-------+-------+
             |               |
             v               v
       Domain Profile   Project Memory
             |
             v
        Agent / agy
             |
             v
       Brief Analysis
             |
             v
       AIA Validation
             |
       +-----+------+
       |            |
       v            v
  Clarification   Research
       |            |
       v            v
  Human Answer   Evidence
       |            |
       +-----+------+
             |
             v
       Brief Revision
             |
             v
       Readiness Assessment
             |
             v
       Project Work Plan
```

## 3.2 agy 的責任

agy 是需求理解與推理的主要執行者。

負責：

- 理解使用者原始描述。
- 分析目前 Brief。
- 找出缺漏及相互衝突的需求。
- 評估資訊的重要性。
- 提出值得詢問的問題。
- 設計問題選項。
- 說明提問原因。
- 提出可行的預設方案。
- 評估不同答案對專案的影響。
- 提出 Brief 修訂建議。

agy 不應直接決定資料庫中的最終狀態。

## 3.3 AIA Runtime 的責任

AIA 程式負責確保整個流程可控、可追蹤及可恢復。

負責：

- 建立及保存 Brief。
- 管理 Brief 版本。
- 驗證 agy 輸出。
- 驗證欄位是否屬於合法 Field Catalog。
- 驗證 Clarification Request。
- 管理 Task 狀態。
- 保存使用者答案。
- 更新 Brief。
- 管理逾時與恢復。
- 記錄 Audit。
- 判斷是否符合階段進入條件。

**agy 負責思考，AIA 負責驗證、保存與治理。**

## 3.4 Domain Profile 的責任

Domain Profile 定義不同專案需要哪些資訊，以及何種程度才算足夠。

例如：

- Historical Fiction：時代、地域、人物、史實基準、虛構程度。
- Nonfiction Plan：目的、讀者、實際執行條件、風險、預算。
- Coding：Repository、技術環境、問題範圍、相容性、驗收條件。
- Investment：投資目標、資金、限制、風險政策、績效衡量方式。

Domain Profile 不應自行實作另一套 Brief Engine。

---

# 4. Project Brief 資料模型

## 4.1 Brief 的定位

每個 Project 必須擁有自己的 Brief。

建議檔案：

```text
projects/
  <project_id>/
    project.yaml
    brief.yaml
    manifest.yaml
    memory/
    decisions/
    research/
    artifacts/
```

其中：

- `project.yaml`：Project 基本資訊。
- `brief.yaml`：目前有效的需求狀態。
- `manifest.yaml`：Domain 與專案結構定義。
- `memory/`：長期專案知識。
- `decisions/`：重要決策紀錄。
- `research/`：研究與證據。
- `artifacts/`：實際產出。

檔案內容應納入 Git 版本管理。

AIA Database 則保存版本索引、狀態、關聯、執行歷史及 Audit。

## 4.2 Generic Brief

所有 Project 共用以下基本欄位：

| 欄位 | 說明 |
|---|---|
| purpose | 專案目的 |
| project_type | 專案類型 |
| audience | 目標對象 |
| scope | 工作範圍 |
| depth | 深度與詳細程度 |
| style | 風格或表達方式 |
| deliverables | 預期交付成果 |
| constraints | 限制條件 |
| schedule | 時間規劃 |
| quality_criteria | 品質與驗收標準 |
| success_criteria | 成功條件 |

欄位可以是單值、陣列或結構化物件。

並非所有欄位都必須由使用者回答。

## 4.3 Domain Brief

各 Domain 可以擴充自己的欄位。

例如：

### Nonfiction Plan

```yaml
domain: nonfiction_plan

fields:
  activity:
    description: 計畫的實際活動

  route:
    description: 路線範圍與方向

  start_location:
    description: 起點

  end_location:
    description: 終點

  season:
    description: 預計執行季節

  duration:
    description: 預計總天數

  daily_mileage:
    description: 每日步行距離

  rest_days:
    description: 休息日安排

  transport_policy:
    description: 是否允許使用大眾運輸

  accommodation:
    description: 住宿方式

  budget:
    description: 預算

  safety:
    description: 安全條件

  supplies:
    description: 補給需求

  audience:
    description: 書籍讀者

  book_type:
    description: 書籍類型

  narrative_style:
    description: 敘事方式

  word_count:
    description: 預計字數
```

這只是初始 Field Catalog。

Agent 可以提出新增欄位，但新增欄位必須經過 AIA Schema 驗證及版本管理，不能自行寫入任意欄位。

## 4.4 Field State

每個欄位必須保存目前值及其來源。

建議狀態：

| 狀態 | 意義 |
|---|---|
| UNKNOWN | 尚未取得資訊 |
| ASKED | 已向使用者提出問題 |
| ANSWERED | 使用者已提供答案 |
| DEFAULTED | Agent 提出暫定預設值 |
| RESEARCHED | 已透過研究取得資訊 |
| CONFIRMED | 已確認為目前採用的正式設定 |
| CONFLICTED | 存在互相矛盾的資訊 |
| SUPERSEDED | 已被新版本取代 |

狀態不應只是一個簡單的流程旗標。

每個欄位還必須保存：

- value
- status
- source
- confidence
- rationale
- updated_at
- updated_by
- confirmation_required
- related_clarification_id
- evidence_refs（適用時）

例如：

```yaml
fields:
  route_direction:
    value: clockwise
    status: ANSWERED
    source: user
    confidence: 1.0
    rationale: 使用者明確指定順時針
    confirmation_required: false

  daily_mileage:
    value: 20
    unit: km
    status: DEFAULTED
    source: agent_recommendation
    confidence: 0.65
    rationale: 初步採用較保守的徒步規劃
    confirmation_required: true
```

`ANSWERED` 不代表所有情況下都已經是不可變的正式決策。

例如，使用者提供一個每日步行距離，但後續研究顯示路線不適合，仍然需要重新討論。

## 4.5 Field History

不得只保存最後一個值。

必須保留歷次變更：

```text
daily_mileage
  v1: UNKNOWN
  v2: DEFAULTED = 20 km
  v3: ANSWERED = 25 km
  v4: CONFLICTED
  v5: CONFIRMED = 22 km
```

每次變更都應記錄：

- 原值。
- 新值。
- 變更原因。
- 變更來源。
- 相關 Task。
- 相關 Clarification。
- 相關 Decision。
- 時間。

---

# 5. Clarification Request

## 5.1 設計目的

Clarification Request 是 Agent 與使用者共同完成需求定義的正式機制。

它不是一般聊天訊息。

每個問題都必須與：

- Project。
- Brief 欄位。
- 問題原因。
- 可能選項。
- 重要性。
- 回答狀態。

建立明確關聯。

## 5.2 資料模型

建議資料表：

`clarification_requests`

| 欄位 | 型態 | 說明 |
|---|---|---|
| id | TEXT | 唯一識別碼 |
| project_id | TEXT | 所屬 Project |
| task_id | TEXT | 發起的 Task |
| field_key | TEXT | 對應 Brief 欄位 |
| question | TEXT | 問題內容 |
| options | JSON | 建議選項 |
| reason | TEXT | 為何需要詢問 |
| impact | TEXT | 對後續工作的影響 |
| priority | INTEGER | 優先級 |
| blocking | BOOLEAN | 是否阻擋目前階段 |
| status | TEXT | 問題狀態 |
| answer | JSON | 使用者答案 |
| answered_at | DATETIME | 回答時間 |
| expires_at | DATETIME | 可選逾時時間 |
| created_at | DATETIME | 建立時間 |
| updated_at | DATETIME | 更新時間 |

建議狀態：

- OPEN
- ANSWERED
- DEFERRED
- CANCELLED
- EXPIRED

問題取消或過期時，不應刪除歷史紀錄。

## 5.3 問題優先級

Agent 必須評估問題對後續工作的影響。

建議分級：

| 等級 | 說明 | 處理方式 |
|---|---|---|
| P0 | 安全、法律、重大資源或不可逆決策 | 必須確認 |
| P1 | 影響整體架構或主要方向 | 優先詢問 |
| P2 | 影響局部規劃 | 可稍後詢問 |
| P3 | 不影響目前階段 | 暫不詢問 |

優先級不是單純由問題數量決定，而是由 Agent 評估其下游影響。

## 5.4 問題數量

一次最多提出 3–5 個問題。

但這是上限，不是最低要求。

如果目前只有一個真正重要的問題，就只問一個。

不應將數十個問題一次丟給使用者。

## 5.5 問題必須附帶原因

不良問題：

> 你想走順時針還是逆時針？

較佳問題：

> 你希望順時針還是逆時針環島？這會影響各地區的行程順序、每日里程安排，以及後續書籍的章節組織。目前沒有足夠資訊判斷哪個方向更適合，因此需要先確認你的偏好。

Agent 可以提出推薦，但不得把推薦當成使用者已經選擇。

---

# 6. Clarification Workflow

## 6.1 初始流程

```text
User Prompt
    |
    v
Create Project
    |
    v
Initialize Brief
    |
    v
Agent Analyzes Brief
    |
    v
Generate Brief Analysis
    |
    v
AIA Validates Output
    |
    v
Evaluate Missing Fields
    |
    +-------------------+
    |                   |
    v                   v
Need Clarification?   Can Research?
    |                   |
    v                   v
Create Questions      Create Research Task
    |                   |
    v                   v
WAITING_FOR_INPUT     Update Brief
    |
    v
User Answers
    |
    v
Validate Answer
    |
    v
Update Brief Version
    |
    v
Reassess Brief
    |
    +-------------------+
              |
              v
      Ready for Next Stage?
              |
         +----+----+
         |         |
         v         v
        No        Yes
         |         |
         v         v
   Clarify Again  Propose Work Plan
```

## 6.2 Agent 分析輸出

agy 必須依照指定 Schema 回傳結構化結果。

例如：

```json
{
  "summary": "規劃一本徒步環台灣本島的實用計畫書",
  "understanding": [
    "需要規劃實際徒步路線",
    "需要考慮住宿、補給及安全",
    "需要決定書籍的讀者與敘事定位"
  ],
  "missing_fields": [
    {
      "field": "route_direction",
      "importance": "high",
      "blocking": true,
      "reason": "會影響路線與行程順序"
    },
    {
      "field": "daily_mileage",
      "importance": "high",
      "blocking": true,
      "reason": "會影響總天數與住宿安排"
    }
  ],
  "questions": [
    {
      "field": "route_direction",
      "question": "你希望順時針還是逆時針環島？",
      "options": [
        "順時針",
        "逆時針",
        "請 Agent 研究後推薦"
      ],
      "reason": "影響行程規劃及章節安排",
      "priority": 1,
      "blocking": true
    }
  ],
  "defaults": [],
  "research_candidates": [
    "台灣本島主要環島路線",
    "主要道路的徒步可行性",
    "各路段距離與補給條件"
  ],
  "recommendations": [],
  "readiness": {
    "status": "NEEDS_CLARIFICATION",
    "reason": "仍有影響整體規劃的重要未知事項"
  }
}
```

此格式為概念範例，正式實作必須建立版本化 JSON Schema。

## 6.3 輸出驗證

AIA 必須驗證：

1. JSON 格式正確。
2. 符合指定 Schema。
3. Field Key 存在於 Field Catalog。
4. 不得建立未授權欄位。
5. 問題數量符合限制。
6. 不得重複建立相同問題。
7. Blocking 設定合理。
8. 每個問題都有原因。
9. 每個預設值都有理由及來源。
10. 不得把 Agent 推測標記為使用者答案。

如果輸出不合格：

- 記錄 Validation Error。
- 要求 agy 修正。
- 修正仍失敗時，Task 進入可恢復的失敗狀態。
- 不得直接將未驗證內容寫入正式 Brief。

---

# 7. Task 與 Execution 狀態整合

## 7.1 WAITING_FOR_INPUT

AIA 必須新增或確認 Task 狀態：

`WAITING_FOR_INPUT`

完整狀態建議：

- PENDING
- RUNNING
- WAITING_FOR_INPUT
- WAITING_FOR_AUTH
- WAITING_FOR_APPROVAL
- COMPLETED
- FAILED
- CANCELLED

若現有程式已有狀態定義，應以遷移方式整合，不可直接破壞既有狀態機。

## 7.2 等待使用者回答

當 Task 進入 `WAITING_FOR_INPUT`：

1. 保存目前 Execution 狀態。
2. 保存 Brief Version。
3. 保存未回答問題。
4. 保存 Agent 已完成的分析。
5. 釋放 Worker。
6. 保留 Task 與 Project 的關聯。
7. 通知 Web Console 或 CLI 使用者有待回答事項。

不得讓 Worker 持續佔用資源等待使用者。

## 7.3 使用者回答

使用者回答後：

1. 找出對應 Clarification Request。
2. 驗證答案格式及值域。
3. 保存原始答案。
4. 更新 Clarification 狀態。
5. 更新 Brief。
6. 建立新的 Brief Version。
7. 寫入 Project Memory 或 Decision（視內容性質）。
8. 重新評估尚未解決的需求。
9. 決定是否恢復原 Task 或建立 Continuation Execution。

## 7.4 Continuation Execution

不得假設原本的 agy Session 一定持續存在。

建議採用：

- 原 Task 保留。
- 原 Execution 保留。
- 新增 Continuation Execution。
- 載入最新 Brief。
- 載入相關 Clarification。
- 載入 Project Memory。
- 載入必要的 Decision 與 Evidence。

這樣才能真正做到跨 Session 的專案連續性。

**時間連續性應建立在 AIA 的持久化狀態，而不是依賴模型 Session。**

---

# 8. Brief Completeness 與 Readiness Assessment

## 8.1 不使用單純完成率

Brief 不能只用：

```text
已填欄位 / 總欄位
```

計算完成率。

因為不同欄位的重要性不同。

例如：

- 書籍封面風格尚未決定，不影響路線研究。
- 每日步行距離未知，可能直接影響行程可行性。
- 徒步者身體狀況未知，可能影響安全評估。

因此必須依照 Domain Profile 定義不同階段的必要條件。

## 8.2 Readiness 狀態

建議：

| 狀態 | 說明 |
|---|---|
| NOT_READY | 仍有重大未知事項 |
| READY_WITH_ASSUMPTIONS | 可以先行，但存在明確假設 |
| READY_FOR_RESEARCH | 可以開始資料研究 |
| READY_FOR_PLANNING | 可以進入正式規劃 |
| READY_FOR_EXECUTION | 已符合實際執行條件 |
| NEEDS_REASSESSMENT | 新資訊使既有設定需要重新評估 |

同一個 Project 可以有不同階段的 Readiness。

例如：

徒步環島書籍可以已經 `READY_FOR_RESEARCH`，但尚未 `READY_FOR_EXECUTION`。

## 8.3 假設與風險

每個重要假設應包含：

- 假設內容。
- 來源。
- 信心程度。
- 可能影響。
- 可逆性。
- 是否需要使用者確認。
- 重新檢查條件。

重大安全、法律及不可逆資源決策，不得僅因等待時間過長就自動視為已確認。

---

# 9. Timeout 與避免流程停滯

## 9.1 不應無限等待

使用者可能暫時沒有回答。

AIA 必須允許專案在沒有立即取得答案的情況下繼續進行其他工作。

例如：

- 路線方向尚未決定，但可以先研究順時針與逆時針的差異。
- 書籍封面尚未決定，但可以先建立內容架構候選方案。
- 預算尚未確認，但可以先整理不同預算級距的方案。

## 9.2 Timeout 策略

依照問題重要性處理：

| 問題類型 | Timeout 後處理 |
|---|---|
| P0 安全或重大法律 | 保持阻擋 |
| P1 主要方向 | 保持待確認，必要時提供可逆方案 |
| P2 局部規劃 | 可採暫定假設 |
| P3 非必要資訊 | 延後處理 |

Timeout 不等於使用者同意。

所有自動採用的預設值都必須標記 `DEFAULTED`，不能標記 `ANSWERED` 或 `CONFIRMED`。

## 9.3 非阻擋工作

當主要 Task 等待使用者回答時，AIA 可以建立其他不依賴該答案的 Task。

但必須檢查 Dependency。

例如：

```text
Task A: 確認環島方向
    |
    +---- Task B: 研究順時針路線
    |
    +---- Task C: 研究逆時針路線
    |
    +---- Task D: 整理全台徒步安全資料
```

Task B、C、D 可以先行，最終路線規劃則等待方向確認。

---

# 10. Brief 與 Autonomy Core 整合

## 10.1 BriefCompletenessObserver

AIA 應提供 `BriefCompletenessObserver`。

它負責觀察：

1. Completeness Gap：重要欄位缺漏。
2. Decision Conflict：不同設定互相矛盾。
3. New Information：新研究結果使既有設定失效。
4. Stale Assumption：長時間未重新確認的假設。
5. Stage Readiness：專案已符合下一階段條件。

Observer 只負責產生 Observation。

不得直接：

- 修改 Brief。
- 建立 Clarification Request。
- 直接向使用者發問。
- 自行批准重要決策。

## 10.2 Deliberator

Deliberator 依照 Observation 評估：

- 是否需要詢問使用者。
- 是否能自行研究。
- 是否可以採用預設值。
- 是否應提出多個方案。
- 是否需要暫停目前工作。
- 是否已經可以進入下一階段。

可能的決策：

- ASK_HUMAN
- RESEARCH
- PROPOSE_DEFAULT
- DEFER
- PROCEED
- REASSESS

## 10.3 Governance

所有重大需求變更及資源使用仍須符合 AIA-17。

Agent 可以提出：

- 修改專案目標。
- 擴大或縮小範圍。
- 改變預算。
- 改變交付成果。
- 改變主要創作方向。

但必須依照既有 Policy 判斷是否需要使用者批准。

Brief 本身不能成為繞過 Governance 的工具。

---

# 11. CLI 與 Web Console

## 11.1 CLI

建議新增或整合：

```bash
aia project brief <project_id>
aia project brief show <project_id>
aia project brief history <project_id>

aia clarification list <project_id>
aia clarification show <request_id>
aia clarification answer <request_id>
aia clarification defer <request_id>
```

實際指令名稱應配合現有 CLI 命名規則。

CLI 必須支援：

- 顯示 Brief。
- 顯示未知欄位。
- 顯示 Agent 預設值。
- 顯示待回答問題。
- 回答問題。
- 延後回答。
- 檢視欄位歷史。
- 檢視 Readiness。

## 11.2 Web Console

Project 頁面應顯示：

1. Project Goal。
2. Brief Summary。
3. 已確認需求。
4. Agent 建議與預設值。
5. 待回答問題。
6. 未解決衝突。
7. 研究中事項。
8. Readiness。
9. 下一步建議。

待回答問題必須清楚區分：

- Agent 的問題。
- Agent 的推薦。
- 使用者已確認的答案。
- 尚未確認的假設。

不能把這些資訊混在一般 Chat 訊息中。

## 11.3 主動通知

當 Agent 發現需要使用者決定的問題時，可以建立通知。

但是否立即通知，應依照重要性及既有 Notification Policy 判斷。

不應因為每個欄位缺漏就打斷使用者。

---

# 12. 資料持久化與一致性

## 12.1 資料儲存分工

| 資料 | 主要保存位置 |
|---|---|
| Project Brief | Project Filesystem |
| Brief Version Index | SQLite |
| Field History | SQLite |
| Clarification Requests | SQLite |
| Answers | SQLite + Brief History |
| Creative Decisions | Project Decision Store |
| Project Memory | Project Memory Store |
| Evidence | AIA-23 Evidence Store |
| Execution State | SQLite |
| Audit | AIA Audit Store |

## 12.2 一致性要求

Brief 更新與資料庫索引更新必須具備可恢復機制。

不可假設 Filesystem 與 SQLite 可以在同一個交易中完成原子提交。

建議：

1. 先產生新的 Brief Version。
2. 寫入暫存檔。
3. 驗證 Schema。
4. 原子替換正式 Brief。
5. 更新 SQLite Version Index。
6. 寫入 Audit。
7. 如果中途失敗，透過 Recovery 檢查並完成或回復。

實際順序需依現有 Project Storage 與 Recovery 架構整合。

## 12.3 版本控制

Brief 必須具備版本號：

```yaml
brief_version: 7
project_id: walk-taiwan
domain: nonfiction_plan
updated_at: "2026-11-02T12:00:00+08:00"
```

每次正式變更都產生新版本。

不得讓不同 Execution 在不知情的情況下覆蓋對方的 Brief。

更新時必須檢查預期版本，必要時進行衝突合併或重新評估。

---

# 13. 安全與治理要求

1. 使用者輸入必須視為資料，不得直接視為系統指令。
2. agy 產生的 JSON 必須經過 Schema 驗證。
3. 不允許任意欄位寫入正式 Brief。
4. 不允許未經授權的目標或資源變更。
5. 不允許 Agent 把推測轉成已確認事實。
6. 不允許刪除重要需求變更歷史。
7. 不允許因 Timeout 自動取得使用者同意。
8. Clarification Request 不得修改既有 Policy。
9. Brief 更新必須留下 Audit。
10. 所有外部資料及研究結果依 AIA-16、AIA-23 的 Provenance 與 Evidence 規範管理。

---

# 14. 與其他規格的責任界線

| 規格 | 責任 | 與 AIA-21 的關係 |
|---|---|---|
| AIA-01 | 產品需求 | 定義整體需求 |
| AIA-02 | 系統架構 | 定義元件與依賴 |
| AIA-03 | Go Runtime | 實作執行機制 |
| AIA-08 | Autonomy Core | 使用 Brief 作為持續觀察對象 |
| AIA-09 | Initiative & Goal Evolution | 評估目標與需求是否需要演進 |
| AIA-10 | Continuity | 保存 Brief、答案及決策的時間連續性 |
| AIA-11 | Self-Correction | 根據錯誤與結果修正假設 |
| AIA-12 | Collaboration | 多 Agent 協作需求 |
| AIA-16 | Provenance | 管理資訊來源與可信度 |
| AIA-17 | Governance | 管理重要決策與權限 |
| AIA-18 | Integration Architecture | 定義整體整合方式 |
| AIA-20 | Book Manifest | 定義書籍 Project 的結構 |
| AIA-21 | Brief & Clarification | 定義需求模型及釐清流程 |
| AIA-22 | Complex Problem Analysis & Planning | 定義複雜未知問題分析、問題建模與自主規劃機制 |
| AIA-23 | Research & Evidence | 管理研究及事實驗證 |
| AIA-24 | Workspace Integration | 管理 Project 與 Execution Workspace |

AIA-21 不負責：

- 直接撰寫書籍。
- 決定最終創作風格。
- 執行完整研究。
- 驗證所有外部事實。
- 管理 Git Worktree。
- 實作 Autonomy Core。
- 自行批准重大決策。

它提供的是這些能力共同依賴的需求基礎。

---

# 15. Go Package 建議

應沿用現有架構，不為了新規格重建整個系統。

建議新增或整理：

```text
go/internal/
  domain/
    brief/
      models.go
      field.go
      schema.go
      version.go
      readiness.go

    clarification/
      models.go
      status.go
      validation.go

    project/
      ...

  service/
    brief/
      service.go
      analyzer.go
      validator.go
      version_manager.go

    clarification/
      service.go
      answer_handler.go
      timeout_handler.go

  autonomy/
    brief_observer.go
    brief_readiness.go
```

實際目錄應依現有程式碼組織調整。

原則：

- Domain Model 不依賴 agy。
- Service 負責流程與持久化。
- Agent Adapter 負責與 agy 溝通。
- Observer 只產生 Observation。
- Deliberator 決定下一步。
- Governance 管理權限。

---

# 16. 實作階段

## Stage 0 — 現有架構盤點

檢查：

- Project Registry。
- Project Manifest。
- Project Storage。
- Task State Machine。
- Execution Lifecycle。
- Agent Profile。
- Prompt Injection。
- Memory。
- Audit。
- Recovery。
- CLI。
- Web Console。

確認哪些能力已存在，避免重複實作。

特別確認：

1. Scheduler 是否正確載入 Agent Profile。
2. Domain Prompt 是否真正傳入 agy。
3. Project 是否正確綁定 Workspace。
4. Task 是否支援 WAITING_FOR_INPUT。
5. Execution 是否可以跨 Session Continuation。
6. 新 Project 是否可能誤用既有 Project。

## Stage 1 — Generic Brief Model

完成：

- Generic Brief。
- Domain Field Catalog。
- Field State。
- Field History。
- Brief Version。
- Readiness Model。
- YAML Schema Validation。

驗收：

可以建立不同 Domain 的 Brief，並保存欄位來源與版本。

## Stage 2 — Clarification Management

完成：

- Clarification Request Model。
- SQLite Persistence。
- 問題驗證。
- 使用者回答。
- Brief 更新。
- CLI 顯示與回答。
- Web Console 顯示與回答。

驗收：

使用者可以在 Task 等待期間回答問題，AIA 能夠更新 Brief 並建立 Continuation Execution。

## Stage 3 — Agent Brief Analysis

完成：

- Brief Analysis Prompt。
- JSON Schema。
- Missing Field Detection。
- Conflict Detection。
- Question Generation。
- Default Proposal。
- Research Candidate。
- Readiness Assessment。

驗收：

Agent 能根據不同 Project 的 Brief，提出有理由、有優先級的問題，而不是固定問卷。

## Stage 4 — Timeout、Recovery 與治理

完成：

- Timeout Policy。
- 非阻擋 Task。
- Continuation Execution。
- Brief Version Conflict。
- Audit。
- Recovery。
- Human Approval Integration。

驗收：

中斷、重新啟動及使用者延遲回答，都不會使 Project 失去狀態或錯誤地確認假設。

## Stage 5 — Autonomy Core Integration

完成：

- BriefCompletenessObserver。
- Deliberator Integration。
- ASK_HUMAN。
- RESEARCH。
- PROPOSE_DEFAULT。
- PROCEED。
- REASSESS。

驗收：

Agent 不需要使用者明確要求「請問我問題」，也能在適當時機發現需求缺漏，並自主決定下一步。

---

# 17. 驗收案例：徒步環台灣本島計畫書

本案例是 AIA-21 的主要端到端驗收項目。

## 17.1 初始輸入

```text
我想寫一本關於計劃徒步環台灣本島一圈的計畫的書，
請幫我規劃這本書。
```

## 17.2 預期 Agent 行為

Agent 不應直接輸出完整目錄。

應先：

1. 建立新的 Project。
2. 判斷 Project Type。
3. 初始化 Generic Brief。
4. 載入 Nonfiction Plan Profile。
5. 分析實際徒步計畫與書籍規劃的雙重目標。
6. 找出影響後續決策的重要未知事項。
7. 區分需要詢問、可以研究、可以暫定的資訊。
8. 提出自己的初步看法。
9. 向使用者提出少量高價值問題。
10. 等待回答或進行不受阻擋的研究。

## 17.3 不合格行為

以下均視為驗收失敗：

- 直接產生完整目錄，沒有先理解需求。
- 一次要求使用者填寫所有欄位。
- 將未知事項全部交給使用者自行決定。
- 把 Agent 預設值當成使用者答案。
- 不建立獨立 Project。
- 把新計畫錯誤綁定到既有歷史小說 Project。
- 只保存對話，不保存 Brief。
- 使用者回答後無法恢復工作。
- 無法說明為何提出某個問題。
- 沒有證據卻把道路距離、開放狀態或交通資訊當成事實。

## 17.4 合格行為

Agent 應能展現：

- 自己理解目標。
- 自己判斷問題。
- 自己提出推薦。
- 自己說明理由。
- 主動研究可以取得的資訊。
- 知道哪些事情不能擅自決定。
- 能隨使用者回答調整原先想法。
- 能在不同 Session 延續同一個 Project。
- 能在新資訊出現時重新檢查舊假設。

## 17.5 行為驗收指標

| 指標 | 驗收要求 |
|---|---|
| Project Isolation | 新目標建立獨立 Project |
| Brief Persistence | Brief 可保存及恢復 |
| Field Provenance | 每個重要欄位可追溯來源 |
| Clarification Quality | 問題具有明確決策價值 |
| Question Discipline | 單次不超過 5 題 |
| Assumption Integrity | 預設值不會偽裝成使用者確認 |
| Continuation | 回答後能繼續專案 |
| Conflict Detection | 能發現需求互相矛盾 |
| Reassessment | 新資訊能觸發重新評估 |
| Governance | 重大決策遵循既有 Policy |

不得以「資料表存在」或「API 回傳成功」當作完整驗收。

必須實際觀察 Agent 在不同資訊條件下的行為。

---

# 18. 測試計畫

至少包含以下測試：

### Unit Tests

- Brief Schema Validation。
- Field State Transition。
- Field History。
- Clarification Validation。
- Duplicate Question Detection。
- Readiness Calculation。
- Timeout Policy。
- Version Conflict。

### Integration Tests

- 建立 Project 與 Brief。
- agy 分析 Brief。
- AIA 驗證 JSON。
- 建立 Clarification。
- 使用者回答。
- 更新 Brief。
- 建立 Continuation Execution。
- 跨 Session 恢復。

### Behavioral Acceptance Tests

1. 使用者只提供一句模糊需求。
2. 使用者提供完整需求。
3. 使用者拒絕回答。
4. 使用者只回答部分問題。
5. 使用者改變原先答案。
6. Agent 發現原先假設錯誤。
7. 研究結果與使用者答案衝突。
8. 使用者長時間沒有回覆。
9. 多個 Task 同時依賴不同 Brief 欄位。
10. Agent 提出不必要問題，應能被評估為低品質行為。

### Longitudinal Tests

同一 Project 跨多個 Execution 持續運作。

觀察：

- Brief 是否保持一致。
- 舊決策是否可追溯。
- 新資訊是否能更新舊假設。
- Agent 是否重複詢問已回答問題。
- Agent 是否能主動發現新問題。
- Agent 是否能判斷何時不需要詢問。

---

# 19. Definition of Done

AIA-21 完成必須同時滿足：

1. 每個 Project 都可以擁有獨立 Brief。
2. Brief 支援 Generic 與 Domain-specific Fields。
3. 欄位具備狀態、來源、信心及歷史。
4. Agent 能分析需求缺漏與衝突。
5. Agent 能提出 Clarification Request。
6. AIA 能驗證並保存問題。
7. Task 支援 WAITING_FOR_INPUT。
8. 使用者能透過 CLI 或 Web Console 回答。
9. 回答能更新 Brief 並建立版本。
10. Execution 能跨 Session Continuation。
11. Timeout 不會被視為使用者同意。
12. Observer 與 Deliberator 能整合 Brief 狀態。
13. Audit 與 Recovery 能處理 Brief 變更。
14. 徒步環島計畫書案例通過 Behavioral Acceptance。
15. 不破壞既有 Investment、Coding、Writing Project。

---

# 20. 最終設計結論

AIA-21 的價值不是讓 AIA 更會問問題，而是讓 AIA 能夠**持續理解一件尚未被完整定義的事情**。

傳統 Task 系統的基本假設是：

```text
User gives complete instruction
        |
        v
Agent executes
        |
        v
Task completed
```

AIA 必須採取不同模式：

```text
User expresses an intention
        |
        v
Agent understands the goal
        |
        v
Agent discovers unknowns
        |
        v
Agent researches and asks
        |
        v
Human and Agent refine requirements
        |
        v
Agent proposes its own direction
        |
        v
Human and Agent make decisions
        |
        v
Agent plans and executes
        |
        v
Agent verifies and learns
        |
        v
New information changes the Brief
        |
        v
Agent reassesses the project
```

這個循環沒有一次性的「需求完成」。

專案的目標、條件與理解會隨著研究、實際執行及新發現而演進。

因此，Brief 不只是專案開始時的文件，而是 AIA 長期認知與決策過程中的重要持久化狀態。

**AIA 不應只是等待使用者把事情說清楚才開始工作；它應該有能力協助使用者，把原本還不清楚的事情逐步弄清楚。**
