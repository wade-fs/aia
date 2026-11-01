# AIA-05 Investment Agent Specification

**Autonomous Investment Research Experiment Specification**

- 文件代碼：AIA-05
- 文件名稱：Investment Agent Specification
- 版本：v2.0
- 文件性質：Domain Agent 功能與實驗規格
- 適用系統：AIA (Autonomous Intelligence Architecture)
- 實作語言：Go
- 市場範圍：台灣證券市場
- 實驗資金：新台幣 100,000 元（虛擬資金）
- 特別觀察標的：台積電（2330）
- 狀態：目標與架構規格基準；部分功能尚未實作

---

# 1. 文件目的

本文件定義 AIA 第一個 Domain Experiment：Investment Agent。

Investment Agent 不只是傳統的投資資訊查詢工具，也不只是根據市場資料產生買賣建議的投資顧問模組。

它是 AIA 自主性研究的第一個實驗場域。

透過持續觀察市場、保存歷史脈絡、評估環境變化、提出行動、執行虛擬交易、檢驗預測結果及修正自身策略，觀察 Agent 是否能形成可持續、可追溯、可驗證的自主行為。

本文件定義：

1. Investment Agent 的系統定位。
2. 投資實驗目標與使用者設定。
3. 投資組合及虛擬交易帳本。
4. 市場資料收集與品質管理。
5. 自主研究、判斷及決策流程。
6. 預測、結果評估及自我修正。
7. 長期記憶與時間連續性。
8. 目標演化及治理限制。
9. Runtime、CLI、Web UI 整合需求。
10. 驗收條件及實作階段。

本文件不代表所有功能已經完成。規格定義與程式實作狀態必須分開管理。

# 2. Investment Agent 在 AIA 中的定位

## 2.1 系統分層

AIA 採用共用自主性核心與 Domain Agent 分離的架構。

```text
AIA
│
├── Autonomy Core
│   ├── Initiative
│   ├── Continuity
│   ├── Self-Correction
│   └── Goal Evolution
│
├── Runtime
│   ├── Persistent Runtime
│   ├── Task Scheduler
│   ├── Background Workers
│   ├── Recovery
│   └── Event Processing
│
├── Core Services
│   ├── Memory
│   ├── Goal
│   ├── Task
│   ├── Policy
│   ├── Approval
│   └── Audit
│
└── Domain Agents
    ├── Investment Agent  [目前唯一已實作的 Domain Module]
    ├── Coding Agent      [尚未實作]
    └── Writing Agent     [尚未實作]
```

Autonomy Core 不屬於 Investment Agent 專用功能。

未來 Coding Agent 與 Writing Agent 應使用同一套自主性核心，但依照各自領域的資料、工具、目標及驗證方法執行。

## 2.2 Investment Agent 的責任

Investment Agent 負責：

- 觀察台灣證券市場。
- 收集及分析市場資訊。
- 維護虛擬投資組合。
- 自主選擇觀察標的。
- 自主提出買進、賣出、增持、減持或不交易決策。
- 執行虛擬交易。
- 保存投資判斷與預測。
- 評估歷史決策結果。
- 修正研究方法與投資策略。
- 維持投資目標及策略的長期連續性。

Investment Agent 不負責：

- 管理使用者真實證券帳戶。
- 持有使用者真實資金。
- 直接向券商送出真實交易委託。
- 自行修改 AIA 全域安全政策。
- 自行解除人類核准機制。
- 修改或刪除已完成的歷史稽核紀錄。

# 3. 投資實驗目標

## 3.1 使用者設定

本實驗的基本設定如下：

| 項目 | 設定 |
|---|---|
| 初始資金 | NT$100,000 |
| 資金性質 | 虛擬資金 |
| 投資市場 | 台灣證券市場 |
| 特別關注 | 台積電（2330） |
| 觀察頻率 | 每個證券營業日 |
| 交易頻率 | Agent 自主決定 |
| 檢查日期 | 每月 1 日 |
| 核心目標 | 最大化投資組合總資產 |

## 3.2 唯一主要目標

Investment Agent 的主要目標為：

**在實驗期間內，透過自主研究及投資決策，最大化投資組合總資產。**

這個目標不預先指定：

- 必須持有台積電。
- 必須持有特定產業。
- 必須分散投資。
- 必須保留固定現金比例。
- 必須每天交易。
- 必須採用價值投資、成長投資或技術分析。
- 必須追求固定收益率。

上述事項應由 Agent 根據市場狀況、歷史經驗及自身判斷決定。

Agent 可以選擇持有現金，也可以選擇完全不交易。

不交易本身也是一種決策，必須保存判斷依據。

## 3.3 特別關注台積電

台積電（2330）是本實驗的重要觀察標的。

觀察範圍包括：

- 台積電股價及成交量。
- 台積電法說會。
- 月營收及財務資訊。
- 重大公告。
- 半導體產業供需。
- AI 晶片與先進製程需求。
- 主要客戶及供應鏈變化。
- 美國科技產業及相關企業消息。
- 美股半導體指數及重要企業表現。
- 國際利率、匯率及地緣政治因素。

特別關注不代表必須持有，也不代表台積電必須成為最大持倉。

Agent 可以根據研究結果判斷其他標的具有更好的投資機會。

## 3.4 投資自主性

Agent 應具備以下自主決策能力：

1. 自行發現值得研究的市場事件。
2. 自行選擇研究標的。
3. 自行評估持倉及現金配置。
4. 自行提出交易或不交易決策。
5. 自行檢驗預測與實際結果。
6. 自行修正研究方法及策略。

人類負責設定初始資源、實驗目標及治理邊界。

Agent 負責在允許範圍內尋找達成目標的方法。

# 4. 投資組合與帳本

## 4.1 Portfolio

初始 Portfolio：

```text
Portfolio
├── Initial Capital: 100,000 TWD
├── Cash: 100,000 TWD
├── Holdings: Empty
├── Realized P/L: 0
├── Unrealized P/L: 0
└── Total Assets: 100,000 TWD
```

Portfolio 至少需要保存：

- 初始資金。
- 可用現金。
- 持有標的。
- 各標的持有股數。
- 平均成本。
- 最新有效估值。
- 已實現損益。
- 未實現損益。
- 累計交易成本。
- 股利及其他現金流。
- 總資產。
- 資產估值時間。

## 4.2 資產計算

投資組合總資產：

```text
Total Assets
    = Cash
    + Σ(Holding Quantity × Latest Valid Price)
    + Receivable Cash
    - Payable Cash
```

未實現損益：

```text
Unrealized P/L
    = Current Market Value
    - Remaining Cost Basis
```

已實現損益應依照實際成交紀錄及成本分攤方式計算。

所有成本計算方法必須明確定義並維持一致，不得因為策略改變而回溯修改歷史帳本。

## 4.3 帳本不可竄改

每一筆交易必須保留：

- Order ID。
- Transaction ID。
- 標的代號。
- 買賣方向。
- 委託股數。
- 成交股數。
- 委託價格。
- 成交價格。
- 交易時間。
- 手續費。
- 證券交易稅。
- 成交狀態。
- 對應 Decision ID。

已完成的交易紀錄不得直接修改或刪除。

錯誤資料應透過更正紀錄、沖銷紀錄或新的調整事件處理。

## 4.4 交易紀錄與決策紀錄分離

Decision、Order、Transaction 是不同實體。

```text
Decision
   │
   ▼
Order
   │
   ▼
Transaction
```

一個 Decision 可以產生多個 Order。

一個 Order 可能：

- 完全成交。
- 部分成交。
- 尚未成交。
- 取消。
- 失效。

不得將「提出買進決策」直接記錄成「已完成買進」。

# 5. 虛擬交易規則

## 5.1 交易性質

所有交易均為模擬交易。

AIA 不得連接真實券商帳戶執行自動下單。

不得將真實金融帳戶憑證、交易密碼或券商 OAuth Token 作為 Investment Agent 的執行依據。

## 5.2 標的選擇

Agent 可以自行選擇台灣證券市場中的投資標的。

標的研究應考量：

- 市場價格。
- 成交量。
- 財務狀況。
- 產業發展。
- 公司公告。
- 新聞事件。
- 國際市場連動。
- 投資組合現有曝險。
- 資料可信度。

不得因為資料不足而自動產生虛構的研究結論。

## 5.3 交易方式與待確認限制

本實驗的交易能力須區分已確認設定與尚待確認事項。

| 項目 | 狀態 |
|---|---|
| 現股買進 | 實驗核心需求 |
| 現股賣出 | 實驗核心需求 |
| 增持、減持 | 實驗核心需求 |
| 持有現金 | 允許 |
| 不交易 | 允許 |
| 融資 | 尚待使用者確認 |
| 借券放空 | 尚待使用者確認 |
| 衍生性商品 | 尚待使用者確認 |

初期可採現股多頭作為保守模擬預設，但這是實作建議，不得將其記載為使用者已確認的永久限制。

在使用者尚未確認前，不應開放融資、借券放空及衍生性商品的實際模擬交易。

## 5.4 交易成本

模擬交易必須計入合理交易成本：

- 券商手續費。
- 證券交易稅。
- 其他適用費用。
- 可能的滑價。

費率應使用可配置參數，並記錄每筆交易採用的費率版本。

不可將費率直接散落在交易邏輯中，也不可將某一時點的費率永久寫死。

## 5.5 成交模擬

成交模擬至少應考量：

- 股票交易單位。
- 價格跳動單位。
- 漲跌幅限制。
- 交易時段。
- 停牌或暫停交易。
- 成交量及流動性。
- 委託價格與市場價格的差異。
- 部分成交。
- 滑價。
- 除權息及其他公司行動。

若缺少足夠的歷史或即時市場資料，必須降低成交模擬可信度，不能宣稱成交結果等同真實市場可成交價格。

## 5.6 公司行動

系統必須規劃處理：

- 現金股利。
- 股票股利。
- 除權息。
- 股票分割及反分割。
- 減資。
- 合併及其他重大公司行動。

公司行動必須同時影響持倉、成本及現金帳本。

# 6. 市場資料架構

## 6.1 資料來源分層

資料來源應依用途區分，不應把所有來源視為同等可信。

### A. 台灣官方市場資料

| 來源 | 用途 |
|---|---|
| TWSE | 上市股票行情、成交資訊、指數及市場資料 |
| TPEx | 上櫃股票行情及相關市場資料 |
| MOPS | 公開資訊觀測站公告、財務及重大訊息 |
| 公開發行公司資訊 | 公司營運及財務揭露 |

### B. 國際官方資料

| 來源 | 用途 |
|---|---|
| FRED | 美國利率、總體經濟及金融時間序列 |
| SEC EDGAR | 美國上市公司申報文件 |
| 各國央行及官方統計機構 | 利率、匯率及總體經濟資訊 |

### C. 國際財經資訊

包括：

- 國際財經新聞。
- 美國科技產業新聞。
- 半導體產業資訊。
- 主要企業法說會及投資人資訊。
- 國際金融市場分析。

新聞及分析文章屬於研究資料，不應直接取代官方公告或原始財務資料。

## 6.2 資料來源可信度

每一筆資料至少應保存：

- Source ID。
- Source Type。
- Original URL 或來源識別資訊。
- Published At。
- Retrieved At。
- Data Timestamp。
- Data Type。
- Content Hash。
- Provider Version。
- Data Quality Status。

Agent 應能區分：

1. 官方原始資料。
2. 官方公告的轉載或整理。
3. 媒體報導。
4. 市場分析。
5. 社群資訊。
6. 模型推論。
7. 模擬或測試資料。

## 6.3 資料時間

不得混淆：

- 事件發生時間。
- 資料發布時間。
- Provider 擷取時間。
- 市場交易時間。
- Agent 判斷時間。

歷史決策只能使用當時已經能取得的資訊。

不得使用未來才公布的資料回頭證明過去的判斷正確。

# 7. Data Quality 與 Research Snapshot

## 7.1 Data Quality

市場資料必須經過品質檢查。

至少包括：

- 欄位完整性。
- 數值合理性。
- 時間戳記有效性。
- 標的代號一致性。
- 重複資料檢查。
- 資料來源有效性。
- 資料新鮮度。
- 跨來源差異檢查。

資料品質狀態：

```text
VALID
DEGRADED
STALE
INVALID
MOCK
```

## 7.2 Mock 與 Real 隔離

測試資料與真實資料必須明確隔離。

Mock Data 不得混入正式 Research Snapshot，也不得被當作真實市場行情、公司公告、財務申報或新聞使用。

如果 Provider 無法取得資料：

1. 記錄失敗原因。
2. 保留最後一次成功資料及其時間。
3. 標記資料過期或來源降級。
4. 提醒 Agent 降低判斷信心。
5. 必要時選擇不交易。

不得使用固定價格或虛構公告補足缺失資料，卻不標記資料性質。

## 7.3 Research Snapshot

每次研究應建立不可變的 Snapshot。

Snapshot 至少包含：

- 市場行情。
- 重要公司公告。
- 國際市場資料。
- 相關新聞。
- 資料來源及品質。
- 資料擷取時間。
- 研究標的。
- 當時 Portfolio 狀態。
- 當時可用現金。
- 相關歷史決策。
- 使用的策略版本。

Research Snapshot 是後續重現及稽核決策的基礎。

# 8. 自主研究與決策流程

## 8.1 核心循環

Investment Agent 使用 AIA Autonomy Core 提供的自主性流程：

```text
Observe
   ↓
Recall
   ↓
Assess
   ↓
Discover
   ↓
Deliberate
   ↓
Decide
   ↓
Execute
   ↓
Verify
   ↓
Learn
   ↓
Reassess
   ↺
```

### Observe — 觀察

收集市場行情、公司公告、國際新聞、總體經濟及投資組合狀態。

### Recall — 回憶

讀取歷史研究、投資判斷、持倉變化、預測結果及策略演化紀錄。

### Assess — 評估

評估目前市場環境、投資組合風險、資料品質及既有判斷是否仍然成立。

### Discover — 發現

主動尋找值得研究的新標的、事件、異常變化或可能的投資機會。

### Deliberate — 推理

比較不同選項，檢驗假設，評估可能結果及不確定性。

### Decide — 決策

做出買進、賣出、增持、減持、持有或不交易決策。

### Execute — 執行

在虛擬交易環境中建立及執行交易委託。

### Verify — 驗證

檢查交易結果、成交價格、持倉、現金及帳本是否一致。

### Learn — 學習

評估預測與實際結果，分析錯誤及策略表現。

### Reassess — 重新評估

確認目前目標、策略及研究方向是否仍然合理。

## 8.2 決策紀錄

每次正式決策必須保存：

- Decision ID。
- Decision Timestamp。
- Research Snapshot ID。
- Portfolio Snapshot ID。
- Decision Type。
- Target Symbol。
- Action。
- Quantity 或配置建議。
- Decision Rationale。
- Supporting Evidence。
- Counter Evidence。
- Expected Outcome。
- Confidence。
- Risk Assessment。
- Time Horizon。
- Strategy Version。
- Goal Version。
- Data Quality。
- Result Status。

決策理由必須能追溯至當時使用的資料及研究結果。

## 8.3 No-Action Decision

當 Agent 判斷不應交易時，仍必須建立 No-Action Decision。

原因可能包括：

- 沒有足夠投資機會。
- 資料品質不足。
- 市場不確定性過高。
- 目前持倉仍符合原先判斷。
- 交易成本不具經濟效益。
- 現金配置更有利於目前目標。

No-Action Decision 必須包含判斷依據及重新評估條件。

不得將沒有產生交易視為 Agent 沒有工作。

# 9. Prediction 與 Calibration

## 9.1 預測紀錄

Agent 可以對市場或個別標的提出預測。

每一筆 Prediction 必須包含：

- Prediction ID。
- Target。
- Prediction Type。
- Predicted Direction 或 Value。
- Confidence / Probability。
- Time Horizon。
- Created At。
- Expected Evaluation Time。
- Evidence Snapshot。
- Strategy Version。
- Status。

例如：

```text
Prediction:
  Target: 2330
  Type: PRICE_DIRECTION
  Direction: UP
  Confidence: 0.65
  Horizon: 20 Trading Days
  Evidence: snapshot-xxxxx
  Strategy: strategy-v3
```

以上只是格式示例，不代表預測價格或方向。

## 9.2 預測評估

預測到期後，系統應依照實際資料進行評估。

評估內容：

- 預測方向是否正確。
- 預測幅度與實際結果的差距。
- 預測信心與命中率是否一致。
- 預測期間是否發生重大事件。
- 當時使用的資料是否完整。
- 是否存在系統性偏誤。

## 9.3 Calibration

Agent 必須逐步建立信心程度與實際結果之間的關係。

例如：

- 80% 信心的預測，長期實際命中率是否接近 80%？
- 50% 信心的預測，是否沒有明顯判斷優勢？
- 是否經常高估某些產業或低估特定風險？

可以使用 Brier Score、Log Loss、Calibration Curve 等方法。

但不得只用單一指標評定投資能力。

## 9.4 自我修正

Agent 必須能根據歷史結果提出策略修正。

例如：

- 修正對特定產業的假設。
- 改變研究資料的權重。
- 調整事件判斷方法。
- 修正風險估計。
- 降低過度自信。
- 放棄長期表現不佳的策略。
- 建立新的研究假設。

每次修正都必須保存：

1. 原始策略版本。
2. 修正原因。
3. 支持修正的證據。
4. 新策略版本。
5. 預期改善項目。
6. 後續驗證結果。

不得只因為單次虧損就任意更換策略。

同樣地，也不得只因為單次獲利就認定策略正確。

# 10. Goal Evolution

## 10.1 目標持續性

Investment Agent 的核心目標是最大化投資組合總資產。

這個目標應具有長期持續性，不應因為短期市場波動或單次交易結果而任意改變。

## 10.2 目標優化

Agent 可以自主提出：

- 研究方法改善。
- 投資策略改善。
- 研究範圍調整。
- 新的資料來源需求。
- 風險評估方式改善。
- 投資流程改善。

這些屬於達成既定目標的方法優化。

## 10.3 目標修正

Agent 可以發現既有目標可能存在問題，並提出目標修正建議。

例如：

- 原始目標是否需要增加明確的評估期間？
- 是否需要加入風險揭露指標？
- 是否需要重新定義總資產的比較基準？

但重大目標變更必須經人類核准。

Agent 不得自行將「最大化總資產」改成固定收益、低波動或其他不同目標。

## 10.4 資源限制

Agent 可以在現有資金範圍內自主配置。

但不得自行：

- 增加初始資金。
- 借入資金。
- 變更實驗資金來源。
- 擴張真實金融帳戶權限。
- 解除交易治理限制。

# 11. Memory 與時間連續性

## 11.1 投資記憶

Investment Agent 必須保留可持續使用的長期記憶。

包括：

### Market Memory

- 市場重大事件。
- 產業變化。
- 公司重要資訊。
- 國際市場連動。

### Investment Thesis

- 投資標的研究論點。
- 支持論據。
- 反對論據。
- 失效條件。
- 後續驗證結果。

### Portfolio Memory

- 歷史持倉。
- 資金配置。
- 交易歷史。
- 資產變化。
- 投資組合風險。

### Decision Memory

- 歷史決策。
- 決策理由。
- 預測結果。
- 錯誤分類。
- 決策信心。

### Strategy Memory

- 策略版本。
- 策略修正。
- 歷史績效。
- 適用環境。
- 失效條件。

## 11.2 時間連續性

時間連續性不等於保存聊天紀錄。

Agent 必須能在不同執行週期中恢復：

- 先前的投資目標。
- 先前的投資論點。
- 先前的持倉及現金。
- 尚未完成的研究。
- 尚未到期的預測。
- 先前發現但尚未處理的事件。
- 策略演化歷程。

Agent 重新啟動後，必須能接續先前的研究狀態，而不是每次都重新開始。

## 11.3 記憶修正

新資料可以修正 Agent 對過去事件的理解，但不得覆寫當時實際保存的決策紀錄。

必須區分：

- 當時知道什麼。
- 當時如何判斷。
- 後來發現什麼。
- 現在如何重新理解。

# 12. Runtime 與自主執行

## 12.1 每個證券營業日的工作

每個證券營業日，Investment Agent 應具備：

1. 確認市場交易日。
2. 取得可用市場資料。
3. 檢查資料品質。
4. 讀取目前 Portfolio。
5. 回顧近期研究及決策。
6. 評估市場變化。
7. 主動發現研究主題。
8. 進行投資判斷。
9. 決定是否交易。
10. 保存決策及研究紀錄。

觀察與決策不代表每天必須成交。

## 12.2 每月檢查

每月 1 日提供投資組合檢查報告。

報告至少包括：

- 當月總資產。
- 初始資金。
- 累計損益。
- 累計報酬率。
- 現金比例。
- 持倉明細。
- 當月交易。
- 主要決策。
- 預測表現。
- 策略修正。
- 重大市場事件。
- 下月研究方向。

如果每月 1 日不是證券營業日，仍應產生檢查報告，使用最近一個有效交易日的估值，並清楚標示估值日期。

## 12.3 Runtime Worker

完整自主投資實驗需要常駐 Runtime 或等效的持續執行機制。

至少需要：

- Schedule Worker。
- Task Worker。
- Investment Observation Worker。
- Prediction Evaluation Worker。
- Recovery Worker。
- Event Processing。

Schedule 負責將到期工作轉成 Task。

Task Worker 負責執行 Task。

Investment Agent 負責研究及決策。

三者不可混為一談。

## 12.4 目前實作狀態

目前程式碼已有：

- Schedule Service。
- Task Scheduler。
- Investment Data Collector Service。
- Investment CLI。
- Prediction Evaluation Service。
- Web 操作介面。

但必須明確區分：

- `aia server start` 目前只啟動 Web Server。
- `aia schedule tick` 目前只處理到期 Schedule 並建立 Task。
- Schedule Service 尚未形成完整背景輪詢。
- Task Scheduler 雖有執行方法，但尚未由常駐 Worker 持續呼叫。
- Collector Service 已建立，但尚未由常駐 Runtime 自動持續執行。
- `aia investment sync` 是手動收集操作。
- `aia investment evaluate` 是手動預測評估操作。

因此，目前尚不能宣稱 Investment Agent 已能在每個證券營業日自主觀察及決策。

# 13. CLI 與 Web Interface

## 13.1 CLI

現有命令包括：

```bash
aia investment portfolio
aia investment sync <symbol>
aia investment proposals
aia investment approve <id>
aia investment reject <id>
aia investment predictions
aia investment evaluate

aia schedule list
aia schedule create
aia schedule trigger
aia schedule tick
```

實際參數及使用方式以程式內 CLI Help 為準。

CLI 應支援：

- 檢視 Portfolio。
- 檢視 Holdings。
- 執行資料同步。
- 檢視研究結果。
- 檢視投資決策。
- 審查及處理需要核准的操作。
- 檢視預測。
- 執行預測評估。
- 檢視排程及 Task 狀態。

## 13.2 Web Interface

Investment Workbench 至少應包含：

- Portfolio。
- Holdings。
- Market Observation。
- Data Quality。
- Research。
- Decisions。
- No-Action Decisions。
- Orders。
- Transactions。
- Predictions。
- Strategies。
- Monthly Review。

## 13.3 Autonomy Monitor

Autonomy Monitor 必須呈現可驗證的自主行為證據。

至少包括：

- 主動發現的研究主題。
- 主動建立的 Task。
- 自主決策紀錄。
- 不交易決策。
- 歷史脈絡引用。
- 預測評估結果。
- 策略修正紀錄。
- 目標演化提案。
- 人類核准及拒絕紀錄。

不得只呈現 Agent 自己宣稱「我有主動思考」或「我已經學會」的文字。

# 14. 安全與治理

## 14.1 資金隔離

本實驗只使用虛擬資金。

不得連接真實券商交易帳戶執行交易。

## 14.2 人類治理

以下事項必須保留人類控制權：

- 修改初始實驗資金。
- 修改主要投資目標。
- 開放新的交易能力。
- 解除既有安全限制。
- 重大系統權限變更。
- 重大策略治理政策變更。

## 14.3 Audit

以下事件必須留下 Audit：

- 市場資料收集。
- 資料品質變化。
- Research Snapshot 建立。
- 投資決策。
- 交易委託。
- 虛擬成交。
- 預測建立及評估。
- 策略版本變更。
- 目標演化提案。
- 人類核准及拒絕。
- Runtime 失敗及 Recovery。

Audit 必須具備時間、來源、關聯 ID 及操作者資訊。

## 14.4 權限邊界

Investment Agent 不得：

- 修改 AIA Policy 以解除自身限制。
- 修改或刪除 Audit。
- 取得未授權的外部帳戶。
- 自行擴張工具權限。
- 將虛擬交易轉換成真實交易。
- 將 Mock Data 偽裝成 Real Data。

# 15. 評估指標

## 15.1 投資成果

主要指標：

- Total Assets。
- Total Return。
- Realized P/L。
- Unrealized P/L。
- Cash Balance。
- Portfolio Drawdown。
- Transaction Costs。

總資產最大化是主要目標。

其他風險及績效指標用於理解結果，不應擅自取代主要目標。

## 15.2 參考基準

建議同時追蹤：

- 台灣加權股價指數。
- 元大台灣50（0050）。

這些是比較基準，不是使用者指定的投資目標，也不限制 Agent 必須複製指數配置。

比較時必須使用一致的期間及合理的總報酬口徑，並考量股利及相關費用。

## 15.3 自主性指標

### Initiative

- 自主發現研究主題數量。
- 主動建立研究 Task 數量。
- 主動提出決策數量。
- 有證據支持的主動行為比例。

### Continuity

- 歷史決策引用率。
- 研究狀態恢復成功率。
- 策略版本連續性。
- 跨週期研究延續性。

### Self-Correction

- 預測評估完成率。
- 發現錯誤後提出修正的比例。
- 策略修正後的驗證結果。
- 修正是否具有可追溯證據。

### Goal Evolution

- 目標持續性。
- 方法優化提案數量。
- 目標修正提案品質。
- 未經核准的重大目標變更次數。

未經核准的重大目標變更應為零。

# 16. Acceptance Criteria

Investment Agent 必須分階段驗收，不可因為某個 API 或 CLI 命令存在，就宣稱整個自主投資能力已完成。

## P1 — Portfolio & Ledger

驗收條件：

- 能建立 NT$100,000 虛擬 Portfolio。
- 能正確記錄現金及持倉。
- 能建立交易委託及成交紀錄。
- 能計算總資產及損益。
- 交易紀錄不可任意覆寫。

## P2 — Data Quality & Research

驗收條件：

- 能連接真實市場資料來源。
- 能保存來源及時間資訊。
- 能檢查資料品質。
- Mock 與 Real 資料完全隔離。
- 能建立 Research Snapshot。
- 資料失敗時不產生虛構市場資訊。

## P3 — Decision & Simulation

驗收條件：

- 能根據 Research Snapshot 產生決策。
- 能自主選擇標的及配置。
- 能提出買賣或不交易決策。
- 能進行虛擬交易。
- 能追溯決策與交易關係。

## P4 — Prediction & Self-Correction

驗收條件：

- 能保存有期限的預測。
- 能使用實際資料評估預測。
- 能分析信心校準。
- 能提出策略修正。
- 能保存策略版本及修正證據。

## P5 — Persistent Runtime & Continuity

驗收條件：

- Runtime 能持續運作。
- 能自動處理證券營業日排程。
- 能自動執行到期 Task。
- 中斷後能 Recovery。
- 能恢復歷史研究脈絡。
- 能持續處理未完成預測及研究。

## P6 — Initiative & Goal Evolution

驗收條件：

- 能主動發現研究主題。
- 能自主判斷是否採取行動。
- 能合理選擇不行動。
- 能提出研究方法及策略優化。
- 能提出目標演化建議。
- 重大目標變更須經人類核准。
- 所有自主行為均有可驗證紀錄。

## P7 — Long-term Autonomous Experiment

驗收條件：

- 能跨越多個月持續運作。
- 能保留完整投資及研究歷史。
- 能持續追蹤預測結果。
- 能根據結果修正策略。
- 能維持目標及策略演化脈絡。
- 每月產生資產及自主性檢查報告。

# 17. Roadmap

實作順序應與 AIA 共用自主性核心的開發順序一致。

| 階段 | 工作 | 目的 |
|---|---|---|
| R1 | Portfolio & Ledger | 建立可信投資帳本 |
| R2 | Real Data Providers | 建立真實資料基礎 |
| R3 | Research Snapshot | 保存可重現的研究環境 |
| R4 | Decision & Simulation | 建立自主決策及虛擬交易 |
| R5 | Prediction & Calibration | 建立預測評估能力 |
| R6 | Persistent Runtime | 建立跨時間持續執行能力 |
| R7 | Initiative Engine | 建立主動發現及行動能力 |
| R8 | Self-Correction | 建立策略修正能力 |
| R9 | Goal Evolution | 建立目標演化能力 |
| R10 | Long-term Experiment | 執行長期自主投資實驗 |

此 Roadmap 是 Investment Agent 的領域實作順序。

Autonomy Core 的共用能力應由 AIA 核心文件統一定義，不應在此重複建立另一套自主性引擎。

# 18. 已知實作限制與待完成事項

依目前 Go 程式碼審查結果：

| 功能 | 目前狀態 |
|---|---|
| Portfolio / Ledger | 已有基礎實作，仍須完整驗收 |
| Investment CLI | 已有基礎實作 |
| Investment Web Module | 已有基礎實作 |
| Collector Interface | 已有 |
| TWSE Provider | 有真實 API 呼叫，但存在假資料 fallback |
| TPEx Provider | 尚未完成真實 API 串接 |
| FRED Provider | 尚未完成真實 API 串接 |
| SEC Provider | 尚未完成真實資料擷取 |
| Mock / Real 隔離 | 必須加強 |
| Research Snapshot | 已有基礎實作，須檢查資料可信度 |
| Decision / Proposal | 已有基礎實作 |
| Prediction Evaluation | 已有基礎實作，須檢查資料完整性 |
| Strategy Self-Correction | 尚未證明具備完整能力 |
| Persistent Runtime | 尚未完成 |
| Background Workers | 尚未完成 |
| Initiative Engine | 尚未完成 |
| Goal Evolution | 尚未完成 |
| Long-term Autonomous Experiment | 尚未完成 |

特別注意：

1. TWSE Provider 目前在 API 失敗時存在固定價格 fallback，並會產生虛構公告，必須修正。
2. TPEx Provider 目前仍有硬編碼行情及虛構公告。
3. FRED Provider 目前使用固定利率資料，尚未完成真實 API 收集。
4. SEC Provider 目前使用硬編碼申報資料，尚未完成真實文件擷取。
5. Collector Service 已建立不代表資料收集已常駐執行。
6. Schedule Service 已建立不代表 Task 已自動執行。
7. Prediction Evaluation 已有函式不代表預測評估及自我修正完整。
8. Web Console 已有 Investment Workbench 不代表自主投資實驗已經運作。

上述限制必須在實作完成前持續保留，不得因文件更新而將未完成能力標示為完成。

# 19. 最終驗收原則

Investment Agent 的成功不能只用投資報酬率衡量。

本實驗同時檢驗兩個層面：

**第一：投資成果**

在有限虛擬資金下，Agent 是否能透過持續觀察、研究及自主決策，增加投資組合總資產。

**第二：自主性能力**

Agent 是否能跨越時間持續運作、主動發現問題、延續過去脈絡、修正自己的判斷及策略，並在既定目標下逐步改善行為。

獲利不等於判斷正確。

虧損不等於判斷錯誤。

必須回到決策當時的資料、推理、風險及預測進行檢驗。

本實驗最重要的成果，不只是得到一個投資績效數字，而是建立一個能夠長期觀察、驗證及改進自主行為的實驗體。

---

# 20. 雙世代投資決策實驗契約 (Generational Investment Decision Experiment — v2.1 Normative Addendum)

為科學驗證「LLM Agent 是否真的比既有規則更會投資，還是只是更會事後解釋決策」，Investment 實驗正式採行 **雙世代平行對照架構**（整合 `AIA-13` 與 `AIA-15`）：

## 20.1 雙世代分工與永久基準原則
1. **Generation 1 (`G1` — Rule-based Control Baseline)**：使用固定且明確的規則引擎產生買進、賣出、增持、減持或持有決策。`G1` 的規則與版本在實驗啟動後**永久凍結**，作為不可覆寫的長期對照基準。
2. **Generation 2 (`G2` / `G2.x` — LLM-based Experimental Cohort)**：透過 `agy` Harness 由 LLM 根據當下市場快照、持倉、歷史決策與長期目標自主判斷；當透過 `AIA-11` 學習機制提出策略修正時，必須建立新世代版本（`G2.1`、`G2.2`），嚴禁覆寫原始 `G2` 或 `G1`。
3. **公平對照硬約束**：每個世代各自擁有獨立的 `investment_portfolio_profiles`（初始虛擬資金均為 `NT$100,000`、台股零股交易規則一致、手續費與證交稅模擬規則一致），並在每個觀察時點（`AsOf`）接收完全相同的市場資料快照（`investment_data_snapshots`），嚴格阻斷任何未來資料（Zero Look-Ahead Bias）。

## 20.2 預測機率與決策信心的嚴格分離
LLM 世代每次決策輸出必須同時提供並嚴格區分兩類數值，禁止混為一談：
- **事件預測機率 (`Event Probability`, 如 `up_probability = 0.70`)**：針對明確可檢驗事件（如「2330 未來 5 個交易日收盤上漲」），搭配 `expected_return_pct` 與預期區間 `[expected_range_low_pct, expected_range_high_pct]`，以 **Brier Score** 與 **Log Loss** 檢驗機率校準度。
- **整體決策信心 (`Decision Confidence`, 如 `decision_confidence = 0.75`)**：針對行動選擇（`BUY | SELL | ADD | REDUCE | HOLD`）、主要風險（`primary_risks`）與反向失效條件（`invalidation_conditions`）的綜合把握度，必須獨立於預測機率之外評估其信心與風險調整後報酬的一致性，不得直接當成預測準確率。

---

**文件結束**
