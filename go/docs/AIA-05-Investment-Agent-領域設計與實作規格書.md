# AIA-05：Investment Agent 領域設計與實作規格書

- 文件編號：AIA-05
- 文件名稱：Investment Agent Domain Design & Implementation Specification
- 系統名稱：AIA — Autonomous Intelligence Architecture
- 版本：1.1
- 狀態：Implementation Baseline
- 日期：2026-10-31
- 適用市場：Taiwan Equities
- 預設模式：User-Defined Portfolio Mode
- 執行原則：公開資訊研究、持倉分析、投資建議、不自動交易

---

# 1. 文件目的

本文件定義 AIA Investment Agent 的領域模型、系統責任、資料來源、研究流程、投資決策、持倉管理、預測評估及實作規格。

Investment Agent 是 AIA Personal Agent 的第一個實際應用領域。

本 Agent 不以自動交易為目標，而是建立一套能夠：

1. 自主蒐集公開財經資訊。
2. 觀察台灣及國際市場。
3. 研究使用者指定的股票。
4. 分析使用者提供的持倉及現金。
5. 提出具有依據的投資建議。
6. 記錄預測並追蹤實際結果。
7. 分析錯誤並驗證改善方法。

的個人投資研究系統。

核心問題：

**當一般投資人只能使用公開新聞、財經媒體、公司公告、公開財務資料及社群資訊時，Agent 能否透過系統化研究，持續改善自己的投資判斷？**

本系統不假設能取得法人內部資料、未公開資訊或特殊市場情報。

## 1.1 階段性實作範疇與版本完成度 (Implementation Phases)

配合系統架構驗證進度，AIA Investment Agent 採分期實作與驗證：

| 階段與模組 | 範圍與能力 | 當前狀態 | 說明 |
|---|---|---|---|
| **P3 Investment 核心** | 投資組合 Profile/Snapshot 管理、資金與部位上限驗證、Go 建議股數計算、交易手續費與稅費計算（`CostConfig`）、Trade Proposal 產生與駁回審計 | **IMPLEMENTED** | 已由 Go 核心服務與 Web API/UI 完整交付，通過單元與整合測試 |
| **P6 Investment Web UI** | 投資組合快照檢視、持倉標的盈虧狀態、交易決策與提案列表、駁回原因檢視 | **IMPLEMENTED** | 已整合於原生 Embedded Web Console，透過 Agent 專屬隔離路由提供 |
| **P4 實際資料抓取** | 台灣證券交易所 (TWSE)、櫃買中心 (TPEx)、公開資訊觀測站 (MOPS)、各大財經新聞與公告爬取整合 | **PLANNED** | 下一階段實作 |
| **P5 預測與評估** | 歷史判斷追蹤、事後收益與風險回測、預測校準評估、策略自我改善反思實驗 | **PLANNED** | 下一階段實作 |
| **CLI-INV 頂層命令** | 頂層終端機命令 `aia investment` | **PLANNED** | 下一階段實作（目前透過 Go Service 與 Web 介面操作） |

---

# 2. 設計原則

## 2.1 公開資訊優先

所有研究必須以一般投資人合法可取得的公開資訊為基礎。

資料來源包括：

- 政府及監管機關。
- 證券交易所。
- 上市櫃公司公開資訊。
- 財經新聞媒體。
- 財經雜誌及研究評論。
- 國際新聞媒體。
- 公開社群及討論區。
- 免費公開行情及財務資料 API。

不要求即時取得所有資料。

每日定時蒐集與分析即可滿足第一階段需求。

## 2.2 使用者指定投資組合

第一階段採用 User-Defined Portfolio Mode。

使用者決定：

- 投資市場。
- 研究股票清單。
- 投資資金上限。
- 風險限制。
- 目前持倉。
- 目前可用現金。

使用者可以只研究一檔股票，例如台積電 2330。

不要求使用者提供每檔股票的配置比例。

Agent 可以自行提出配置建議，但不得自行改變使用者設定的股票池、資金上限及風險限制。

## 2.3 研究與交易分離

Investment Agent 只產生投資建議。

第一階段：

- 不自動下單。
- 不取得券商交易憑證。
- 不管理銀行帳戶。
- 不持有使用者資金。
- 不自行轉移資產。
- 不因為有剩餘預算就必須買進。

所有交易決策由使用者本人執行。

## 2.4 決策可追溯

每一份投資建議都必須保留：

- 使用的資料來源。
- 資料擷取時間。
- 資料所屬時間。
- 研究標的。
- 分析假設。
- 決策理由。
- 信心水準。
- 不確定性。
- 預期觀察期間。
- 失效條件。
- 當時的持倉快照。
- 使用的策略版本。

歷史決策不得任意覆寫。

## 2.5 不以投資報酬率單獨衡量智慧

Agent 的表現不能只看是否賺錢。

必須同時評估：

- 資訊判讀正確性。
- 預測準確性。
- 信心校準程度。
- 錯誤辨識能力。
- 原因分析能力。
- 策略改善能力。
- 改善結果是否經過獨立資料驗證。

---

# 3. 系統範圍

## 3.1 第一階段市場

第一階段以台灣上市股票為主。

預設資產類型：

- 普通股。

資料模型應保留未來支援：

- 上櫃股票。
- ETF。
- 美股。
- 其他市場金融商品。

的擴充能力。

第一階段不要求同時研究多個市場。

## 3.2 股票池

股票池由使用者設定。

建議第一階段限制：

- 最少：1 檔。
- 預設上限：5 檔。
- 可調整上限，但必須由使用者設定。

例如：

```yaml
watchlist:
  - symbol: "2330"
    exchange: "TWSE"
    asset_type: "STOCK"
    name: "台積電"
```

Agent 不得擅自增加研究標的。

若 Agent 發現其他股票具有研究價值，可以提出建議，但必須經使用者同意才加入股票池。

## 3.3 投資資金

投資資金上限由使用者設定。

例如：

```yaml
capital_limit: 100000
currency: TWD
```

`capital_limit` 是使用者授予研究建議使用的資金限制，不等於目前可用現金。

必須區分：

| 欄位 | 定義 |
|---|---|
| capital_limit | 使用者設定的投資資金上限 |
| available_cash | 目前可用現金 |
| holdings_market_value | 目前持倉市值 |
| total_portfolio_value | 現金與持倉市值合計 |
| committed_capital | 已承諾或預留的資金 |

系統不得把這些數值混為一談。

資金上限的計算口徑由使用者在 Portfolio Profile 中設定，例如是否包含既有持倉。

---

# 4. 系統架構與責任分工

Investment Agent 建立於 AIA Runtime 之上。

## 4.1 責任分工

| 工作 | 負責者 |
|---|---|
| 全域 Task 管理 | AIA Runtime |
| Project 管理 | AIA Runtime |
| Agent 指派 | AIA Runtime |
| Memory 管理 | AIA Runtime |
| 任務權限及政策 | AIA Runtime |
| 資料來源探索 | agy |
| 資料來源正式擷取 | Go Data Provider |
| 資料格式標準化 | Go Runtime |
| 資料品質驗證 | Go Runtime |
| 資料快照 | Go Runtime |
| 公開資訊研究 | agy |
| 財務及市場分析 | agy + Go |
| 投資建議 | agy + Go |
| 持倉計算 | Go Runtime |
| 交易成本估算 | Go Runtime |
| 預測追蹤 | Go Runtime |
| 信心校準 | Go Runtime |
| 策略版本管理 | Go Runtime |
| 自我改善實驗 | agy 提案、Go 管理與驗證 |

### 4.1.1 Go 計算與驗證責任邊界 (Core Boundary Rule)

AIA 在投資決策與提案產生過程中，嚴格落實「agy 負責研究與判斷，Go 負責數字、限制與最終決策資料的有效性」的責任分工：

1. **agy 職責**：
   - 外部市場研究與公開資料研判。
   - 產業趨勢與財報分析。
   - 形成投資方向判斷（BUY、SELL、HOLD）與信心度評估。
   - 提供結構化理由（Reasoning）、潛在風險（Risks）及失效條件（Invalidation Rules）。

2. **Go Runtime 職責**：
   - **驗證股票代號**：檢查 Instrument 是否存在於已註冊股票池或 Watchlist，且為 Active 狀態。
   - **驗證方向與信心度**：檢查 Action 是否為合法 BUY/SELL/HOLD，且 Confidence 達到決策門檻（預設 >= 0.50）。
   - **計算建議股數**：依市場規範（台股整股 1,000 股或零股規範），根據建議金額與估計市價進行整數股數計算。
   - **計算手續費與稅費**：依券商手續費率（標準 0.1425%，最低 NT$20，含折扣率）及證交稅率（賣出 0.3%，買入 0）精確計算交易成本。
   - **檢查可用現金與資金限制**：檢查扣除最低現金保留（`min_cash_pct`）後之可用現金、檢查投資組合資金上限（`capital_limit`）及單一標的持倉上限（`max_position_pct`）。
   - **產生最終 Trade Proposal**：僅在 Go 驗證與計算全部合規後，才建立狀態為 `PENDING` 的正式 Trade Proposal。
   - **超限自動調整或駁回**：超出限制時由 Go 負責向下調整建議股數或直接駁回，並完整記錄駁回原因至 `investment_decision_rejections` 資料表。

**核心原則**：嚴禁讓 LLM 自己算完金額就直接交付結果，LLM 的輸出絕不允許未經 Go 驗證就直接成為有效交易提案。

### 4.1.2 交易成本計算規則與可配置性 (Transaction Cost Configurability)

為了確保投資提案之財務數字精確且具法律與合規一致性，交易費用、稅額與淨收支一律由 Go Runtime 程式碼實作（位於 `internal/domain/investment/calculator.go`），LLM 不得介入計算。

1. **現行實作之預設計算參數（Default Parameters）**：
   目前實作採用台灣證券市場一般現股交易標準費率作為系統預設值：
   - **標準券商手續費率 (`FeeRate`)**：`0.001425`（即 `0.1425%`，買入與賣出雙向收取）。
   - **電子下單折扣率 (`FeeDiscountRate`)**：預設 `1.0`（無折扣）。
   - **最低手續費門檻 (`MinFee`)**：預設 `20 TWD`（未滿 20 元以 20 元計收）。
   - **證券交易稅率 (`TaxRate`)**：預設 `0.003`（即 `0.3%`，僅於賣出時課徵，買入為 0）。

2. **參數可配置性設計（Configurability）**：
   上述費率僅為當前預設基準，絕非不可修改之硬編碼商業限制。為因應日後市場制度變革、券商個別費率折讓或跨商品交易，Go Runtime 提供 `CostConfig` 與 Profile 擴充配置機制：
   - **券商專屬折讓配置**：支援於 Portfolio Profile 的 `configuration_json` 中定義專屬折扣（例如各大券商常見之 6 折 `0.6`、2.8 折 `0.28` 等），計算公式為 `floor(gross * FeeRate * FeeDiscountRate)`。
   - **活動最低手續費**：支援個別券商活動之低門檻手續費設定（例如 NT$1 元優惠）。
   - **不同商品與法規稅率調整**：支援特定證券商品稅率客製化（例如 ETF / 指數股票型基金法定證券交易稅為 `0.1%`、現股當日沖銷法定降稅優惠為 `0.15%`、或未來主管機關修訂稅率）。
   - **多市場/多資產擴充介面**：架構上保留 `CostConfig` 注入介面，日後擴展美股（SEC Fee、TAF、零手續費）或其他海外資產時，得動態替換計算策略，避免文件或程式碼產生過時死規則。

3. **計算公式與整數截斷規範**：
   - **成交名目金額 (Gross Amount)**：`round(quantity * price)`
   - **券商手續費 (Broker Fee)**：`max(MinFee, floor(GrossAmount * FeeRate * FeeDiscountRate))`
   - **證券交易稅 (Transaction Tax)**：買入為 `0`；賣出為 `floor(GrossAmount * TaxRate)`
   - **買入總支出 (Total Buy Cost)**：`GrossAmount + BrokerFee`
   - **賣出淨現金收益 (Net Sell Proceeds)**：`GrossAmount - BrokerFee - TransactionTax`

## 4.2 重要架構區分

必須分離以下四個階段：

1. Data Collection：取得資料。
2. Research：理解市場及公司。
3. Investment Decision：判斷標的本身的投資價值。
4. Portfolio Action：結合使用者資產與限制，提出操作建議。

例如：

同樣是看好台積電：

- 沒有持股且現金充足：可能建議分批買進。
- 已持有大量台積電：可能建議續抱。
- 現金不足：可能建議不操作。
- 持倉比例過高：可能建議減碼。

因此，股票研究結論不等於投資操作建議。

---

# 5. Domain Agent 定義

## 5.1 Investment Agent

負責整體投資研究與建議。

工作包括：

- 理解使用者投資設定。
- 檢查持倉快照。
- 取得研究資料。
- 指派研究工作。
- 整合分析結果。
- 產生投資建議。
- 記錄預測。
- 檢視過去決策。
- 提出策略改善建議。

## 5.2 Data Research Worker

協助搜尋與分析資料來源。

工作：

- 尋找免費公開 API。
- 尋找新聞 RSS。
- 尋找公開資料網站。
- 比較資料完整性。
- 分析使用限制。
- 檢查資料來源可信度。
- 提出來源替代方案。

agy 可以自主探索來源，但不能自行將未驗證來源加入正式資料管線。

## 5.3 Taiwan Market Research Worker

負責：

- 台股行情。
- 公司財報。
- 月營收。
- 重大訊息。
- 法人買賣超。
- 融資融券。
- 估值。
- 台灣市場制度及政策。

## 5.4 Global Intelligence Worker

負責：

- 美國科技產業。
- AI 產業發展。
- 半導體供應鏈。
- 國際企業財報。
- 美國經濟與貨幣政策。
- 全球貿易。
- 出口管制。
- 地緣政治。

## 5.5 Social Sentiment Worker

負責：

- 公開社群討論。
- 投資人情緒。
- 熱門話題。
- 市場共識。
- 市場傳言。
- 不同觀點的整理。

社群資訊不得直接當成已證實事實。

---

# 6. 資料來源設計

資料來源分為五大類。

## 6.1 Taiwan Market Data

優先使用官方資料。

### A. 臺灣證券交易所 TWSE

網址：

https://www.twse.com.tw/

OpenAPI：

https://openapi.twse.com.tw/

資料：

- 上市股票行情。
- 成交資訊。
- 本益比。
- 殖利率。
- 融資融券。
- 三大法人。
- 市場統計。

API Key：依各 API 服務規則確認；不得假設所有端點都不需要驗證。

### B. 公開資訊觀測站 MOPS

網址：

https://mops.twse.com.tw/

資料：

- 財務報表。
- 月營收。
- 年報。
- 重大訊息。
- 公司治理資訊。
- 股東會及法說會公開資料。

資料擷取應遵守網站存取規則。

### C. 櫃買中心 TPEx

網址：

https://www.tpex.org.tw/

OpenAPI：

https://www.tpex.org.tw/openapi/

作為未來上櫃股票擴充來源。

### D. FinMind

網址：

https://finmind.github.io/

資料：

- 台股歷史行情。
- 財務資料。
- 月營收。
- 法人資料。
- 融資融券。
- 其他金融資料集。

使用免費方案前，必須確認：

- API 呼叫限制。
- 資料集可用範圍。
- 非商業使用限制。
- 資料授權。
- 資料更新頻率。

FinMind 是整合資料來源，不是唯一權威來源。

官方資料應優先用於重要數據交叉驗證。

## 6.2 International News

台灣個股研究必須觀察國際資訊。

以台積電為例，國際資訊可能直接影響：

- 客戶需求。
- AI 晶片需求。
- 半導體資本支出。
- 先進製程競爭。
- 設備與材料供應。
- 美國出口管制。
- 美中科技競爭。
- 美元與利率。
- 全球景氣。

### 建議來源

| 來源 | 觀察主題 | 取得方式 |
|---|---|---|
| Reuters | 國際金融、企業、政策 | 公開網頁、可用 RSS 或新聞索引 |
| CNBC | 美國金融、科技、半導體 | RSS、公開網頁 |
| Bloomberg | 全球市場、企業投資 | 公開網頁、可用 RSS |
| BBC | 國際政治與經濟 | RSS |
| AP News | 國際重大事件 | RSS、公開網頁 |
| Financial Times | 國際經濟與產業 | 公開內容、訂閱內容依授權 |
| Deutsche Welle | 歐洲與國際政經 | RSS |
| Nikkei Asia | 亞洲產業與半導體 | 公開內容、訂閱內容依授權 |

新聞來源不得假設全文免費。

若只能取得標題、摘要或新聞索引，必須標記內容完整度。

不得繞過付費牆、登入驗證或存取限制。

## 6.3 Global Economic Data

### A. FRED

網址：

https://fred.stlouisfed.org/

API：

https://fred.stlouisfed.org/docs/api/fred/

免費註冊 API Key。

觀察：

- 美國利率。
- CPI。
- PCE。
- 就業。
- GDP。
- 美元及金融條件。
- 美國公債殖利率。

必須記錄原始資料發布時間與資料所屬期間。

### B. Federal Reserve

網址：

https://www.federalreserve.gov/

觀察：

- FOMC 決議。
- 利率政策。
- 會議紀錄。
- 經濟預測。
- 聯準會官員公開發言。

### C. U.S. Bureau of Labor Statistics

網址：

https://www.bls.gov/

觀察：

- CPI。
- PPI。
- 就業。
- 工資。
- 生產者價格。

### D. U.S. Bureau of Economic Analysis

網址：

https://www.bea.gov/

觀察：

- GDP。
- PCE。
- 國際貿易。
- 國民所得。

### E. SEC EDGAR

網址：

https://www.sec.gov/edgar/search/

公開申報資料 API：

https://www.sec.gov/edgar/sec-api-documentation

觀察：

- NVIDIA。
- AMD。
- Intel。
- 其他半導體公司。
- 雲端服務業者。
- 主要科技客戶。

使用 SEC API 時遵守官方存取頻率及 User-Agent 規範。

## 6.4 Social Sentiment

可觀察：

- PTT Stock。
- Dcard 公開討論。
- Reddit 公開討論。
- YouTube 公開財經節目。
- 公開財經部落格。
- 公開投資討論區。

限制：

1. 只使用合法可取得的公開資訊。
2. 不繞過登入、CAPTCHA 或反爬蟲機制。
3. 不假設所有平台提供免費 API。
4. 不把討論熱度當成投資價值。
5. 不把未經證實的傳言當成事實。
6. 必須保存來源連結與發表時間。

社群資訊的主要用途是理解市場情緒與觀點差異，而非驗證公司基本面。

## 6.5 RSS 與公開網頁

RSS 是第一版新聞蒐集的重要方式。

RSS Collector 必須支援：

- RSS 2.0。
- Atom。
- HTTP ETag。
- Last-Modified。
- Conditional GET。
- Retry。
- Timeout。
- URL 正規化。
- 重複文章識別。
- 發布時間解析。
- 時區轉換。

RSS 不一定包含全文。

系統應區分：

- TITLE_ONLY。
- TITLE_AND_SUMMARY。
- FULL_TEXT.
- LINK_ONLY。

不得因為取得 RSS 就假設可以永久保存或公開重製全文。

---

# 7. Data Source Discovery 與正式啟用

## 7.1 agy 自主探索

agy 可以執行資料來源探索任務。

例如：

- 尋找免費台股行情 API。
- 尋找半導體新聞 RSS。
- 尋找 NVIDIA 財報資料來源。
- 尋找 Fed 經濟資料。
- 比較不同資料來源的更新時間。
- 分析資料授權條款。

探索結果必須形成 Source Proposal。

## 7.2 Source Proposal

每個新來源至少記錄：

- source_name
- source_type
- base_url
- documentation_url
- authentication_type
- api_key_required
- free_tier
- rate_limit
- update_frequency
- data_coverage
- historical_coverage
- licensing
- terms_of_use
- reliability_assessment
- limitations
- discovered_by
- discovered_at

## 7.3 Source Approval

來源分為：

- DISCOVERED
- UNDER_REVIEW
- APPROVED
- ACTIVE
- SUSPENDED
- REJECTED

agy 不得自行將來源設為 ACTIVE。

正式啟用前必須確認：

1. 來源可正常取得。
2. 資料格式可解析。
3. 更新頻率符合需求。
4. 使用方式符合服務條款。
5. API Key 及費用條件明確。
6. 資料品質符合標準。

涉及付費服務、帳戶登入、API Key 或條款變更時，必須由使用者確認。

## 7.4 Source Health Monitoring

Go Runtime 負責監控：

- HTTP 狀態。
- API 回應。
- 資料更新時間。
- 欄位缺漏。
- 資料異常。
- Rate Limit。
- 授權失效。
- 來源中斷。

來源失效時：

1. 停止使用不可靠資料。
2. 記錄 Source Health Event。
3. 搜尋是否有已核准備援來源。
4. 必要時建立 agy 資料來源調查 Task。
5. 不得默默以過期資料代替最新資料。

---

# 8. 資料可信度與新聞判讀

## 8.1 資料類型

所有蒐集內容必須分類：

| 類型 | 定義 |
|---|---|
| OFFICIAL_FACT | 官方公告、財報、監管資料 |
| VERIFIED_REPORT | 有可靠來源支持的新聞報導 |
| MEDIA_ANALYSIS | 媒體分析及評論 |
| EXPERT_OPINION | 專家或分析師觀點 |
| SOCIAL_OPINION | 公開社群個人看法 |
| UNVERIFIED_RUMOR | 未經證實的傳言 |
| AGENT_INFERENCE | Agent 根據資料形成的推論 |

不得混淆事實與推論。

## 8.2 Source Reliability

來源可信度不是固定的單一分數。

應考慮：

- 是否為原始資料來源。
- 是否有編輯制度。
- 是否具名。
- 是否有可追溯證據。
- 是否有其他來源交叉確認。
- 過去更正紀錄。
- 是否存在利益衝突。
- 是否為轉載或二手消息。

同一媒體在不同議題上的可靠程度可能不同。

## 8.3 Cross-Source Verification

重要事件應盡可能交叉驗證。

例如：

某媒體報導台積電將擴大海外投資。

Agent 應嘗試確認：

1. 台積電官方公告。
2. 公司法說會資料。
3. 其他可靠媒體報導。
4. 當地政府公開資訊。
5. 事件是否只是市場推測。

若沒有足夠證據，必須標記為未確認。

## 8.4 News Deduplication

同一事件可能被多家媒體轉載。

系統應識別：

- 原始事件。
- 原始報導。
- 後續轉載。
- 不同媒體的獨立報導。
- 新增的事實或觀點。

不可將十篇轉載文章視為十個獨立證據。

---

# 9. 台積電全球資訊觀測模型

第一版即使只研究台積電，也必須建立國際資訊觀測能力。

模組名稱：

`Global Intelligence Monitor`

## 9.1 觀測主題

### A. AI 產業需求

觀察：

- NVIDIA。
- AMD。
- Microsoft。
- Google。
- Amazon。
- Meta。
- 其他大型雲端服務業者。

重點：

- AI 資本支出。
- 資料中心投資。
- GPU 需求。
- AI 晶片供應。
- 雲端業者營運狀況。
- AI 投資回收能力。

### B. 半導體產業競爭

觀察：

- Samsung。
- Intel。
- TSMC。
- ASML。
- Applied Materials。
- Lam Research。
- Tokyo Electron。

重點：

- 製程技術。
- 產能規劃。
- 設備訂單。
- 先進封裝。
- 良率。
- 產業供需。

### C. 全球經濟

觀察：

- Fed 利率政策。
- 美國 CPI / PCE。
- 美元匯率。
- 美國公債殖利率。
- 全球景氣。
- 中國經濟。
- 歐洲經濟。

### D. 國際政經

觀察：

- 美國半導體政策。
- 出口管制。
- 美中科技競爭。
- 國際貿易政策。
- 台海局勢。
- 海外設廠政策。
- 主要市場法規變化。

## 9.2 影響路徑分析

國際新聞不得直接轉換成台積電買賣訊號。

每一項重要事件都必須分析：

1. 事件內容。
2. 影響對象。
3. 對台積電的直接或間接影響。
4. 影響方向。
5. 預期影響期間。
6. 市場是否已經反映。
7. 仍有哪些不確定因素。

例如：

NVIDIA 宣布增加資本支出，不代表台積電股價必然上漲。

必須進一步分析：

- 支出是否用於 AI 晶片。
- 相關晶片是否由台積電生產。
- 訂單是否已經反映在既有預測。
- 供應能力是否足夠。
- 市場是否早已預期。

## 9.3 Event Impact Model

事件影響分類：

- DIRECT_POSITIVE
- INDIRECT_POSITIVE
- DIRECT_NEGATIVE
- INDIRECT_NEGATIVE
- MIXED
- NEUTRAL
- UNKNOWN

另記錄：

- impact_horizon
- impact_confidence
- evidence_count
- evidence_quality
- market_expectation
- invalidation_conditions

---

# 10. Data Collection 與 Snapshot

## 10.1 Collection Pipeline

```text
Source Registry
      |
      v
Data Collector
      |
      v
Raw Data Storage
      |
      v
Normalization
      |
      v
Validation
      |
      v
Deduplication
      |
      v
Data Snapshot
      |
      v
Research Agent
```

## 10.2 Raw Data

原始資料應保留必要的追溯資訊：

- source_id
- source_record_id
- source_url
- fetched_at
- published_at
- effective_at
- content_hash
- content_type
- raw_metadata
- retention_policy

原始新聞全文是否保存，依來源授權與使用條款決定。

## 10.3 Data Snapshot

每次研究使用固定 Snapshot。

Snapshot 必須記錄：

- snapshot_id
- created_at
- data_cutoff_at
- instrument_scope
- source_versions
- source_health
- market_data_version
- news_collection_window
- validation_summary

後續評估必須能還原當時可取得的資訊。

禁止使用事後才出現的資料回頭評估當時決策，避免 Look-ahead Bias。

---

# 11. Portfolio Profile 與持倉管理

## 11.1 Portfolio Profile

使用者固定設定。

```yaml
portfolio:
  name: "Personal Investment Portfolio"
  market: "TW"
  currency: "TWD"

  capital_limit: 100000
  capital_limit_basis: "TOTAL_PORTFOLIO"

  watchlist:
    - symbol: "2330"
      exchange: "TWSE"
      asset_type: "STOCK"

  odd_lot_enabled: true
  auto_trading_enabled: false
```

上述金額僅為設定範例。

## 11.2 Portfolio Constraints

使用者可以設定：

- 最大投資資金。
- 最大單一標的配置比例。
- 最低現金保留比例。
- 最大單次買進金額。
- 是否允許分批買進。
- 是否允許建議賣出既有持倉。
- 是否允許研究股票池以外的標的。

未設定的限制不得由 Agent 自行假設為已授權。

## 11.3 Portfolio Snapshot

每次研究前，使用者提供目前帳戶狀態。

```yaml
as_of: "2026-10-31T09:00:00+08:00"
currency: "TWD"
source: "MANUAL"

available_cash: 50000

holdings:
  - symbol: "2330"
    exchange: "TWSE"
    quantity: 100
    average_cost: 1450
    market_value: 180000

open_orders: []
```

欄位定義：

| 欄位 | 說明 |
|---|---|
| as_of | 快照時間 |
| currency | 幣別 |
| source | MANUAL / BROKER_IMPORT |
| available_cash | 可用現金 |
| quantity | 持有股數 |
| average_cost | 平均成本 |
| market_value | 持倉市值 |
| open_orders | 未成交委託 |

以上數值僅為格式示意。

`market_value` 可以由使用者提供，也可以由系統使用最新有效行情估算；必須標記資料來源及時間。

## 11.4 Snapshot Validation

系統必須檢查：

- 快照時間是否過舊。
- 現金是否為負值。
- 股數是否有效。
- 平均成本是否有效。
- 持倉市值是否合理。
- 是否有未成交委託。
- 是否存在重複持倉。
- 幣別是否一致。

若快照資料不完整，必須明確標示，不得假裝掌握完整帳戶狀況。

## 11.5 Snapshot History

每次更新建立新快照。

不得覆寫歷史快照。

這是檢驗 Agent 過去建議是否合理的重要依據。

---

# 12. 台股零股與交易成本

第一階段必須支援零股建議。

Agent 可以提出：

- 買進 1 股。
- 買進數股。
- 分批買進。
- 不操作。
- 減碼部分持倉。
- 全數賣出建議。

但不得自動送出委託。

## 12.1 Odd Lot Trading Rules

建立可配置的交易規則：

`OddLotTradingRules`

應包含：

- market
- exchange
- session_type
- trading_hours
- minimum_quantity
- quantity_increment
- order_constraints
- effective_from
- effective_until
- source_url
- verified_at

盤中零股與盤後零股規則不得寫死在業務邏輯中。

規則變更時應更新版本。

## 12.2 Trading Cost Model

建立：

`TradingCostModel`

應包含：

- broker
- commission_rate
- minimum_commission
- commission_discount
- sell_tax_rate
- other_fees
- effective_from
- effective_until
- source_url

券商實際手續費折扣及最低手續費可能不同。

若使用者沒有提供券商費率，系統必須使用明確標示的估算模型，並提醒實際成本可能不同。

## 12.3 Cost Estimation

買進成本：

```text
gross_buy_amount = price * quantity

estimated_buy_cost =
    gross_buy_amount
    + estimated_commission
```

賣出淨收入：

```text
gross_sell_amount = price * quantity

estimated_sell_proceeds =
    gross_sell_amount
    - estimated_commission
    - estimated_sell_tax
    - other_fees
```

不得只使用股價乘股數作為實際交易成本。

## 12.4 Cost Model Validation

交易規則及費率應以官方規則及使用者券商實際費率為準。

第一版不必自行推測所有券商費率。

---

# 13. Research Model

## 13.1 Research Categories

每次研究至少包含：

### A. Company Fundamentals

- 營收。
- 毛利率。
- 營業利益率。
- EPS。
- 現金流。
- 負債。
- 資本支出。
- 股利。
- 財務趨勢。

### B. Industry Analysis

- 產業需求。
- 供需。
- 技術發展。
- 競爭者。
- 產業週期。
- 客戶結構。

### C. Market Analysis

- 股價趨勢。
- 成交量。
- 波動度。
- 估值。
- 本益比。
- 歷史區間。
- 市場預期。

### D. Global Intelligence

- 美國科技業。
- AI 投資。
- 半導體供應鏈。
- 國際政策。
- 利率。
- 匯率。
- 全球經濟。

### E. Sentiment Analysis

- 新聞情緒。
- 社群情緒。
- 市場共識。
- 多空分歧。
- 傳言及不確定消息。

## 13.2 Research Report

每次研究必須產生結構化報告。

```yaml
report:
  instrument: "2330"
  snapshot_id: "..."
  generated_at: "..."

  company_fundamentals:
    summary: ""
    evidence: []

  industry_analysis:
    summary: ""
    evidence: []

  market_analysis:
    summary: ""
    evidence: []

  global_intelligence:
    summary: ""
    events: []

  sentiment_analysis:
    summary: ""
    evidence: []

  risks: []
  opportunities: []
  uncertainties: []
  data_gaps: []
```

研究報告不得只輸出一段沒有證據連結的文字。

---

# 14. Investment Decision

## 14.1 Decision Types

統一使用：

- BUY
- HOLD
- REDUCE
- SELL
- WATCH
- NO_ACTION

定義：

| Decision | 意義 |
|---|---|
| BUY | 研究判斷具備買進條件 |
| HOLD | 維持既有持倉 |
| REDUCE | 建議減少持倉 |
| SELL | 建議出售 |
| WATCH | 持續觀察，等待條件成立 |
| NO_ACTION | 目前不建議操作 |

## 14.2 Decision Record

每次決策至少包含：

```yaml
decision:
  instrument: "2330"
  action: "HOLD"

  confidence: 0.72

  reasoning: []
  evidence: []

  expected_horizon: "1M"

  invalidation_conditions: []
  major_risks: []

  created_at: "..."
  strategy_version: "..."
```

`confidence` 是 Agent 對自身判斷的信心，不是預期報酬率，也不是獲利機率。

必須與後續可驗證預測分開管理。

---

# 15. Portfolio Action 與 Trade Proposal

## 15.1 Portfolio Action

將 Investment Decision 與 Portfolio Snapshot 結合。

產生：

- 建議操作。
- 建議股數。
- 預估交易金額。
- 預估手續費。
- 預估賣出稅費。
- 操作後持倉。
- 操作後現金。
- 操作後資金配置。
- 風險變化。
- 不操作的理由。

## 15.2 Trade Proposal

Trade Proposal 只是建議，不是交易指令。

```yaml
trade_proposal:
  instrument: "2330"
  action: "BUY"

  quantity: 5
  estimated_price: 1800
  estimated_gross_amount: 9000
  estimated_total_cost: 9020

  cash_before: 50000
  cash_after: 40980

  quantity_before: 100
  quantity_after: 105

  reasoning: []
  assumptions: []
  invalidation_conditions: []

  status: "PROPOSED"
```

數字僅為格式示意。

## 15.3 Proposal Status

- PROPOSED
- ACCEPTED
- REJECTED
- EXPIRED
- SUPERSEDED

使用者接受建議，不代表系統已經下單。

只有使用者後續提供實際交易結果，才能更新實際持倉。

## 15.4 不操作也是有效決策

Agent 不應因為每日排程而強迫產生交易建議。

以下情況可以直接 NO_ACTION：

- 沒有足夠資訊。
- 資料來源失效。
- 市場價格偏離合理區間。
- 風險過高。
- 目前持倉已符合策略。
- 交易成本不合理。
- 預期報酬不足以補償風險。
- 無法確認關鍵消息。

---

# 16. Prediction 與 Evaluation

## 16.1 Prediction

Agent 必須把重要判斷轉換成可驗證的預測。

例如：

- 未來一個月營收是否維持成長。
- 未來一季毛利率是否下降。
- 未來一個月股價是否優於指定基準。
- 某項政策是否對公司營運造成實際影響。

每項預測必須有：

- prediction_type
- target
- expected_direction
- probability
- evaluation_horizon
- created_at
- data_snapshot_id
- strategy_version
- evaluation_method

## 16.2 Prediction Outcome

預測到期後，由 Go Runtime 取得實際資料並評估。

不得讓 agy 單方面宣稱自己預測正確。

Outcome 必須有可追溯的資料來源。

## 16.3 Confidence Calibration

使用：

- Brier Score。
- Calibration Curve。
- Reliability Diagram。
- 分類準確率。
- 方向預測準確率。
- 預測區間覆蓋率。

必須注意樣本數不足問題。

不得因為少量成功預測就宣稱 Agent 已具備可靠預測能力。

## 16.4 Benchmark

台股預測應選擇適當基準。

例如：

- 加權股價指數。
- 同產業指數。
- 指定期間的買進持有策略。
- 現金持有基準。

基準選擇必須事前固定，不可在結果出來後任意更換。

---

# 17. Self-Improvement

## 17.1 改善流程

```text
Prediction Outcome
       |
       v
Error Detection
       |
       v
Error Diagnosis
       |
       v
Improvement Hypothesis
       |
       v
Strategy Experiment
       |
       v
Out-of-Sample Evaluation
       |
       v
Accept / Reject
```

## 17.2 Error Categories

- DATA_ERROR
- SOURCE_BIAS
- INTERPRETATION_ERROR
- CAUSAL_ERROR
- TIMING_ERROR
- VALUATION_ERROR
- RISK_ESTIMATION_ERROR
- CONFIDENCE_ERROR
- STRATEGY_ERROR
- EXTERNAL_SHOCK

## 17.3 Improvement Proposal

agy 可以提出：

- 新的分析指標。
- 新的資料來源。
- 不同的估值方法。
- 新的風險判斷條件。
- 新的預測模型。
- 新的策略假設。

但不得自行覆寫正式策略。

## 17.4 Strategy Experiment

每次改善實驗必須記錄：

- 原策略版本。
- 新策略版本。
- 改善假設。
- 預期效果。
- 評估指標。
- 評估期間。
- 測試資料範圍。
- Out-of-Sample 結果。
- 是否接受。

正式策略變更必須經過驗證。

禁止只根據過去失敗案例修改規則，再用同一批資料宣稱改善成功。

---

# 18. Database Schema

Investment Agent 使用 AIA 共用 SQLite Database。

所有表格使用一致的 ID、時間欄位與交易管理原則。

## 18.1 基礎資料

### investment_instruments

- id
- symbol
- exchange
- asset_type
- name
- currency
- is_active
- created_at
- updated_at

### investment_watchlists

- id
- portfolio_profile_id
- instrument_id
- priority
- created_at

### investment_data_sources

- id
- source_name
- source_type
- base_url
- documentation_url
- authentication_type
- configuration_json
- status
- health_status
- last_success_at
- last_failure_at
- created_at
- updated_at

### investment_source_proposals

- id
- proposal_json
- discovered_by
- status
- reviewed_by
- reviewed_at
- created_at

## 18.2 資料蒐集

### investment_raw_documents

- id
- source_id
- source_record_id
- source_url
- title
- summary
- content_reference
- content_hash
- content_type
- published_at
- fetched_at
- metadata_json

### investment_market_data

- id
- instrument_id
- data_type
- period
- value_json
- source_id
- fetched_at
- effective_at

### investment_data_snapshots

- id
- snapshot_type
- data_cutoff_at
- instrument_scope_json
- source_versions_json
- validation_summary_json
- created_at

### investment_snapshot_items

- id
- snapshot_id
- item_type
- reference_id
- payload_json

## 18.3 Portfolio

### investment_portfolio_profiles

- id
- name
- market
- currency
- capital_limit
- capital_limit_basis
- auto_trading_enabled
- configuration_json
- created_at
- updated_at

### investment_portfolio_constraints

- id
- portfolio_profile_id
- constraint_type
- constraint_value
- enabled
- created_at
- updated_at

### investment_portfolio_snapshots

- id
- portfolio_profile_id
- as_of
- source
- currency
- available_cash
- total_portfolio_value
- validation_status
- metadata_json
- created_at

### investment_portfolio_positions

- id
- portfolio_snapshot_id
- instrument_id
- quantity
- average_cost
- last_known_price
- market_value
- currency
- metadata_json

### investment_portfolio_open_orders

- id
- portfolio_snapshot_id
- instrument_id
- side
- quantity
- limit_price
- status
- metadata_json

## 18.4 Research 與決策

### investment_research_reports

- id
- instrument_id
- snapshot_id
- report_type
- report_json
- strategy_version
- created_at

### investment_decisions

- id
- instrument_id
- research_report_id
- decision_type
- confidence
- reasoning_json
- risks_json
- invalidation_conditions_json
- expected_horizon
- strategy_version
- created_at

### investment_decision_evidence

- id
- decision_id
- raw_document_id
- snapshot_item_id
- evidence_type
- relevance
- explanation

### investment_allocation_proposals

- id
- portfolio_snapshot_id
- proposal_json
- status
- created_at
- expires_at

### investment_trade_proposals

- id
- allocation_proposal_id
- instrument_id
- action
- quantity
- estimated_price
- estimated_gross_amount
- estimated_total_cost
- proposal_json
- status
- created_at

## 18.5 Prediction 與 Evaluation

### investment_predictions

- id
- decision_id
- prediction_type
- target
- expected_direction
- probability
- evaluation_horizon
- due_at
- strategy_version
- created_at

### investment_prediction_outcomes

- id
- prediction_id
- actual_value
- actual_direction
- outcome_status
- evaluation_source_id
- evaluated_at

### investment_evaluation_results

- id
- evaluation_type
- period_start
- period_end
- sample_count
- metrics_json
- strategy_version
- created_at

## 18.6 Self-Improvement

### investment_strategy_versions

- id
- version
- strategy_definition
- status
- created_at
- activated_at

### investment_strategy_experiments

- id
- baseline_version
- candidate_version
- hypothesis
- metrics_json
- evaluation_period
- result_json
- status
- created_at

### investment_error_analyses

- id
- prediction_id
- error_category
- analysis_json
- created_at

### investment_improvement_proposals

- id
- source_error_id
- proposal_json
- status
- reviewed_at
- created_at

---

# 19. Go Package Structure

```text
internal/
  investment/
    domain/
      instrument.go
      portfolio.go
      portfolio_snapshot.go
      portfolio_constraint.go
      research.go
      decision.go
      trade_proposal.go
      prediction.go
      strategy.go

    datasource/
      registry.go
      discovery.go
      health.go
      approval.go

    collector/
      twse/
      tpex/
      finmind/
      mops/
      rss/
      fred/
      sec/
      web/

    normalization/
      market_data.go
      news.go
      financial.go

    validation/
      source.go
      market_data.go
      news.go
      portfolio.go

    snapshot/
      builder.go
      repository.go

    research/
      orchestrator.go
      prompt.go
      report.go

    analyzer/
      fundamentals.go
      valuation.go
      market.go
      global_intelligence.go
      sentiment.go

    decision/
      engine.go
      evidence.go
      confidence.go

    portfolio/
      allocation.go
      constraint.go
      action.go

    tradingcost/
      model.go
      estimator.go
      oddlot_rules.go

    prediction/
      model.go
      scheduler.go

    evaluator/
      outcome.go
      calibration.go
      benchmark.go

    improvement/
      diagnosis.go
      experiment.go
      strategy.go

    repository/
      sqlite/

    service/
      investment_service.go
      portfolio_service.go
      research_service.go
      decision_service.go
```

---

# 20. Provider Interfaces

所有外部資料來源必須透過 Provider Interface 接入。

例如：

```go
type MarketDataProvider interface {
    Name() string

    FetchMarketData(
        ctx context.Context,
        instrument Instrument,
        period DateRange,
    ) ([]MarketData, error)
}
```

```go
type NewsProvider interface {
    Name() string

    SearchNews(
        ctx context.Context,
        query NewsQuery,
    ) ([]NewsDocument, error)
}
```

```go
type EconomicDataProvider interface {
    Name() string

    FetchSeries(
        ctx context.Context,
        query EconomicQuery,
    ) ([]EconomicObservation, error)
}
```

```go
type SourceHealthChecker interface {
    HealthCheck(ctx context.Context) SourceHealth
}
```

Provider 必須回傳：

- Source ID。
- Data Timestamp。
- Fetch Timestamp。
- 原始 URL。
- 資料完整度。
- 錯誤資訊。

agy 不得直接取代正式 Data Provider。

---

# 21. agy 整合規格

## 21.1 agy 負責

- 公開資訊研究。
- 新聞摘要。
- 多來源比較。
- 公司分析。
- 產業分析。
- 全球事件分析。
- 投資假設。
- 投資建議。
- 預測解釋。
- 錯誤診斷。
- 改善策略提案。

## 21.2 Go Runtime 負責

- API 呼叫。
- RSS 擷取。
- 網頁資料擷取。
- 資料格式標準化。
- 去重。
- 資料驗證。
- Snapshot。
- 持倉計算。
- 交易成本計算。
- 預測結果評估。
- 信心校準。
- 策略版本。
- Audit。

## 21.3 Prompt Context

每次 agy 研究任務應提供：

1. 研究標的。
2. 研究目標。
3. Data Snapshot。
4. 重要新聞與來源。
5. 公司財務資料。
6. 國際市場資料。
7. 使用者 Portfolio Snapshot。
8. Portfolio Constraints。
9. 前次研究結論。
10. 相關 Memory。
11. 要求的輸出 Schema。

不得要求 agy 自行猜測缺失的帳戶資料。

## 21.4 Structured Output

研究及決策應採用結構化輸出。

無法解析或不符合 Schema 的結果，不得直接寫入正式決策表。

必須保留原始輸出供 Audit。

---

# 22. 執行模式

## 22.1 Daily Research

每日固定時段執行。

建議流程：

```text
Daily Scheduler
      |
      v
Check Source Health
      |
      v
Collect Taiwan Market Data
      |
      v
Collect Company Announcements
      |
      v
Collect Global News
      |
      v
Collect Economic Data
      |
      v
Collect Public Sentiment
      |
      v
Validate and Build Snapshot
      |
      v
Read Latest Portfolio Snapshot
      |
      v
Investment Research
      |
      v
Investment Decision
      |
      v
Portfolio Action
      |
      v
Save Report and Predictions
      |
      v
Notify User
```

若市場休市，仍可執行國際新聞與公司資訊研究，但必須明確標示最近交易日。

## 22.2 Manual Research

CLI：

```bash
aia investment research
```

指定股票：

```bash
aia investment research --symbol 2330
```

指定研究期間：

```bash
aia investment research --symbol 2330 --period 1m
```

## 22.3 Portfolio Management

```bash
aia investment portfolio show
aia investment portfolio configure
aia investment portfolio snapshot
aia investment portfolio history
```

## 22.4 Data Source Management

```bash
aia investment sources list
aia investment sources health
aia investment sources discover
aia investment sources review
aia investment sources enable
aia investment sources disable
```

啟用來源必須符合來源核准政策。

## 22.5 Decision History

```bash
aia investment decisions list
aia investment decisions show <id>
aia investment predictions list
aia investment evaluation run
aia investment strategy list
```

## 22.6 Trade Proposal

```bash
aia investment proposals list
aia investment proposals show <id>
aia investment proposals accept <id>
aia investment proposals reject <id>
```

`accept` 只代表使用者接受建議，不會執行交易。

---

# 23. Security 與 Data Governance

## 23.1 API Key

API Key 由使用者自行申請及提供。

不得：

- 寫入 Git。
- 寫入 Prompt。
- 寫入 Agent Memory。
- 放入研究報告。
- 寫入一般 Audit Log。
- 輸出到 CLI 錯誤訊息。

應使用 AIA Secret Management 或作業系統環境設定管理。

## 23.2 OAuth

第一階段不需要券商 OAuth。

若未來需要券商帳戶匯入，必須獨立設計授權與資料讀取範圍。

不得因為需要讀取持倉而取得交易授權。

## 23.3 Data Retention

資料保存依來源授權及實際需要決定。

應優先保存：

- 資料來源。
- 原始 URL。
- 標題。
- 摘要。
- Hash。
- 發布時間。
- 擷取時間。
- 研究引用。
- 必要的結構化數據。

新聞全文保存必須遵守來源授權。

## 23.4 Prompt Injection

外部新聞、網頁及社群內容均屬不可信輸入。

不得將其中的指令當作系統指令。

外部資料只能作為研究證據。

agy 不得因為網頁內容要求而：

- 讀取 API Key。
- 修改 AIA 設定。
- 執行不相關命令。
- 變更投資限制。
- 增加股票池。
- 執行交易。

---

# 24. Failure Handling

| 狀況 | 處理方式 |
|---|---|
| API Key 無效 | 暫停來源，通知使用者 |
| API Rate Limit | 延後重試 |
| RSS 無法取得 | 使用已核准備援來源 |
| 新聞來源過期 | 標記資料過期 |
| 財務資料缺漏 | 降低研究完整度 |
| 持倉快照過舊 | 要求更新或限制建議 |
| 行情資料不一致 | 暫停相關價格估算 |
| agy 無法使用 | Task WAITING_FOR_AUTH 或依 Runtime 狀態處理 |
| 研究輸出格式錯誤 | 驗證失敗，不寫入正式決策 |
| 交易成本未知 | 標記估算值，不宣稱精確 |
| 重要消息未確認 | 標記 UNKNOWN，不轉成確定訊號 |

資料不足時，Agent 可以輸出：

`INSUFFICIENT_DATA`

不得為了完成每日報告而虛構結論。

---

# 25. Acceptance Criteria

## I-01 Portfolio Profile

- 可建立投資組合。
- 可設定市場及幣別。
- 可設定資金上限。
- 可指定一檔股票。
- 不要求配置比例。

## I-02 Portfolio Snapshot

- 可輸入現金。
- 可輸入持倉股數。
- 可輸入平均成本。
- 可輸入持倉市值。
- 可輸入未成交委託。
- 可保留歷史快照。

## I-03 Taiwan Data Collection

- 可取得台股行情。
- 可取得公司財務資料。
- 可取得月營收。
- 可取得重大訊息。
- 可記錄來源及資料時間。

## I-04 Global Intelligence

- 可取得至少兩種國際新聞來源。
- 可取得至少一種官方總體經濟資料。
- 可識別與研究標的相關的國際事件。
- 可保留原始新聞連結。
- 可標示資料發布時間。

## I-05 Source Discovery

- agy 可以提出新來源。
- 系統可以記錄來源提案。
- 未核准來源不能進入正式資料管線。
- 可以停用失效來源。

## I-06 News Validation

- 可以區分事實、分析、意見及傳言。
- 可以識別重複新聞。
- 可以記錄交叉驗證結果。
- 不得將未確認消息視為事實。

## I-07 Research Report

- 可產生結構化研究報告。
- 報告包含資料來源。
- 報告包含國際資訊。
- 報告包含風險與不確定性。
- 報告可追溯 Snapshot。

## I-08 Investment Decision

- 支援 BUY、HOLD、REDUCE、SELL、WATCH、NO_ACTION。
- 每個決策有理由。
- 每個決策有證據。
- 每個決策有失效條件。
- 可以選擇不操作。

## I-09 Portfolio Action

- 可結合持倉與現金產生建議。
- 可計算建議買進股數。
- 可計算建議賣出股數。
- 可估算交易成本。
- 可顯示操作後現金與持倉。
- 不得超過使用者設定的資金限制。

## I-10 Odd Lot

- 支援台股零股股數。
- 支援交易成本估算。
- 交易規則可版本化。
- 未確認券商費率時明確標示估算。

## I-11 Prediction

- 可建立可驗證預測。
- 可指定觀察期間。
- 可取得實際結果。
- 可保留預測與策略版本。

## I-12 Evaluation

- 可計算 Brier Score。
- 可產生 Calibration 結果。
- 可依策略版本比較。
- 可處理樣本數不足。

## I-13 Self-Improvement

- 可以建立錯誤分析。
- 可以提出改善假設。
- 可以建立實驗。
- 可以比較原策略與新策略。
- 不得未經驗證直接修改正式策略。

## I-14 Security

- API Key 不進入 Prompt。
- API Key 不進入一般 Log。
- 外部網頁內容不具備系統指令權限。
- 不具備自動交易能力。
- 不得繞過資料來源的存取限制。

---

# 26. Development Milestones

## I-0：Domain Foundation

- Instrument。
- Portfolio Profile。
- Portfolio Snapshot。
- Portfolio Constraints。
- Data Source Registry。
- Repository。

## I-1：Taiwan Market Data

- TWSE Provider。
- MOPS Provider。
- FinMind Provider。
- Market Data Normalization。
- Data Snapshot。

## I-2：Global Intelligence

- RSS Collector。
- International News Provider。
- FRED Provider。
- SEC Provider。
- News Deduplication。
- Global Event Classification。

## I-3：Research Agent

- Research Orchestrator。
- agy Prompt Context。
- Research Report。
- Evidence Linking。
- Research History。

## I-4：Decision & Portfolio Advisor

- Investment Decision。
- Portfolio Action。
- Trade Proposal。
- Odd Lot。
- Trading Cost Model。
- CLI。

## I-5：Prediction & Evaluation

- Prediction。
- Outcome Collection。
- Calibration。
- Benchmark。
- Evaluation Report。

## I-6：Self-Improvement

- Error Diagnosis。
- Improvement Proposal。
- Strategy Version。
- Strategy Experiment。
- Out-of-Sample Validation。

---

# 27. 第一階段交付定義

第一階段不追求複雜的量化交易系統。

必須完成以下閉環：

```text
使用者設定投資組合
        |
        v
使用者提供持倉快照
        |
        v
AIA 蒐集台灣及國際公開資訊
        |
        v
建立 Data Snapshot
        |
        v
agy 研究指定股票
        |
        v
AIA 結合持倉與資金限制
        |
        v
產生投資建議
        |
        v
使用者自行決定是否交易
        |
        v
使用者更新持倉快照
        |
        v
AIA 追蹤預測及評估結果
```

第一階段完成的標準：

1. 可以只研究一檔台股。
2. 可以由使用者設定投資資金上限。
3. 可以由使用者輸入現金及持倉股數。
4. 可以蒐集台灣官方市場資料。
5. 可以蒐集國際新聞與總體經濟資料。
6. 可以產生有證據來源的研究報告。
7. 可以根據使用者持倉提出零股買賣建議。
8. 可以選擇不操作。
9. 可以記錄歷史決策及預測。
10. 完全不具備自動下單能力。

---

# 28. 最終設計原則

Investment Agent 的核心不是預測明天股價漲跌，而是建立一個能持續研究、提出判斷、承認不確定性並接受結果檢驗的系統。

它只能使用一般投資人合法可取得的公開資訊。

它不能假設自己比市場更早知道消息。

它不能因為有資金就必須投資。

它不能因為過去判斷錯誤就隨意修改歷史。

它不能將預測結果包裝成必然。

它必須清楚說明：

- 我知道什麼。
- 我不知道什麼。
- 我根據什麼做出判斷。
- 我有多大的信心。
- 什麼情況下我的判斷會失效。
- 事後如何確認自己是否判斷正確。

**AIA Investment Agent 的目標，是讓普通投資人擁有一套有紀律、有記憶、可追溯、能接受檢驗的個人投資研究系統。**
