## **我認為現在正是應該抽象化 writing 的時候。譬如科幻、修仙、科學研究、風俗研究、思想研究、地理研究、太空研究，表面上是不同創作或研究領域，但它們其實有一組共同需求：**

* ## **有一個持續發展的研究／創作專案。**

* ## **有一套由使用者或 Agent 建立、修訂、確認的知識與設定。**

* ## **有人物、物件、概念、事件、地點、時間、規則等不同類型的研究對象。**

* ## **有來源、證據、推論、假設、虛構等不同可信度。**

* ## **有跨文件、跨時間、跨對象的一致性檢查。**

* ## **有尚未解決的問題、矛盾、修訂紀錄與決策依據。**

* ## **Agent 必須能持續理解專案、發現問題、提出修正，並在適當權限下執行。**

## **這不只是「小說系統」的問題，而是 Project Knowledge & Consistency Engine（專案知識與一致性引擎）。**

## **我會建議不要把 `writing` 硬改成一個更大的 `writing`，而是把它拆成通用核心與領域模組。小說只是第一個應用場景，歷史小說則是第一個需要高度嚴謹校驗的測試場景。**

## **1\. 目前 `writing` 的問題不只是名稱，而是領域模型已經混在一起**

## **我檢視了目前對話中最新可取得的 `/mnt/data/aia(1).tgz`。這份壓縮檔的 `go/internal/domain/writing/` 目前只有 `models.go`；你貼出的 `ConsistencyEngine` 程式碼不在這份壓縮檔中，因此以下會分成兩部分：先評估壓縮檔裡的模型，再評估你貼出的引擎設計。**

### **`models.go` 的問題**

| 現有設計 | 問題 | 建議方向 |
| ----- | ----- | ----- |
| **`HistoricalCharacter`** | **綁定歷史人物與出生、死亡年份** | **改成通用 `Entity`，人物只是其中一種** |
| **`TimelineEvent`** | **以西元年、月為主要時間單位** | **改成通用 `TemporalReference`** |
| **`KnowledgeTier`** | **只有史實、推演、虛構三種** | **改成通用知識狀態與領域分類** |
| **`ConsistencyAnomaly`** | **只考慮人物、章節、卒年、時代錯置** | **改成通用 `Finding` 或 `ConsistencyFinding`** |
| **`ManuscriptProject`** | **專門描述小說專案** | **使用既有通用 Project** |
| **`ErratumProposal`** | **修正對象是稿件文字** | **改成通用 `ChangeProposal`** |
| **`BookManifest`** | **包含小說、卷章、人物、地理等結構** | **抽象成 `ProjectManifest`，小說使用專屬 Profile** |
| **`BookWritingPolicy`** | **歷史真實性、反時代錯置等** | **改成可組合的 Domain Policy** |

## **最明顯的問題是 `BookManifest`。**

## **它已經不只是一本書的設定檔，還混入了：**

* ## **專案基本資料**

* ## **專案文件結構**

* ## **世界觀**

* ## **研究方法**

* ## **證據與事實裁定**

* ## **版本控制**

* ## **治理**

* ## **記憶與延續性**

## **這些有些是通用專案能力，有些才是小說專屬能力。**

## **不應該讓未來每一個研究領域都繼承 `BookManifest`。**

## **2\. 我建議的抽象化方向**

## **不是把所有東西都做成一個龐大的通用結構，而是分成三層。**

## **第一層：Project Knowledge Core**

## **所有專案共用的知識、證據、時間、空間、關係、規則與一致性模型**

## **第二層：Domain Profile**

## **Writing**

## **小說、劇本、世界觀**

## **Historical Research**

## **史實、年代、人物、制度**

## **Scientific Research**

## **假說、實驗、數據、論證**

## **Geographic Research**

## **地理、地形、區域、空間**

## **Cultural Research**

## **風俗、制度、思想、文化**

## **Space Research**

## **天體、軌道、太空環境**

## **第三層：Project Workspace**

## **Markdown、YAML、資料集、圖片、程式碼、研究文件、版本控制**

## **這樣一來，AIA 不需要知道你將來究竟要研究修仙、宋代風俗，還是火星地質。它只需要知道專案使用哪些知識模型、規則和工具。**

### **第一層：通用知識模型**

## **我認為最重要的是以下幾個核心概念。**

| 模型 | 用途 |
| ----- | ----- |
| **`Entity`** | **研究對象：人物、地點、物件、概念、天體、組織等** |
| **`Relation`** | **對象之間的關係** |
| **`Assertion`** | **對某件事情提出的主張或敘述** |
| **`Evidence`** | **支持或反駁主張的證據** |
| **`TemporalReference`** | **時間、時段、先後、持續期間** |
| **`SpatialReference`** | **地點、區域、座標、空間關係** |
| **`Rule`** | **專案規則、自然法則、世界觀規則、研究限制** |
| **`Finding`** | **發現的矛盾、異常、缺漏或不確定性** |
| **`Decision`** | **使用者或 Agent 對問題所作的裁定** |
| **`ChangeProposal`** | **對文件、知識或設定的修訂提案** |

## **這裡有一個很重要的觀念：**

## **Entity 不應該只有人物。**

## **例如：**

* ## **元末歷史小說：朱元璋是 Entity，1368 年也是時間參照，應天是空間參照。**

* ## **修仙小說：境界、功法、靈氣、人物都是 Entity；境界突破條件是 Rule。**

* ## **科幻小說：星球、飛船、文明、人工智慧都是 Entity；光速限制是 Rule。**

* ## **科學研究：研究對象、假說、實驗設備、測量值都是 Entity 或 Assertion。**

* ## **風俗研究：儀式、服飾、飲食、婚姻制度都是 Entity。**

* ## **思想研究：哲學概念、論點、思想家、學派都是 Entity。**

* ## **太空研究：行星、衛星、軌道、觀測事件都是 Entity。**

## **這些資料可以使用同一套知識關係模型，但不必強迫它們使用相同的細節欄位。**

## **3\. `KnowledgeTier` 應該重新設計**

## **你現在的：**

```
type KnowledgeTier string

const (
    TierFactHistory
    TierPlausibleInference
    TierFiction
)
```

## **適合歷史小說，但無法涵蓋科學研究。**

## **例如科學研究中的：**

* ## **已觀測到的現象**

* ## **文獻支持的事實**

* ## **尚未驗證的假說**

* ## **模型推導結果**

* ## **實驗反駁的主張**

## **這些不能簡單歸類成史實、推演、虛構。**

## **我建議拆成兩個維度：**

## **A. Epistemic Status（知識狀態）**

```
type EpistemicStatus string

const (
    StatusObserved   EpistemicStatus = "OBSERVED"
    StatusSourced    EpistemicStatus = "SOURCED"
    StatusInferred   EpistemicStatus = "INFERRED"
    StatusHypothesis EpistemicStatus = "HYPOTHESIS"
    StatusStipulated EpistemicStatus = "STIPULATED"
    StatusFictional  EpistemicStatus = "FICTIONAL"
    StatusDisputed   EpistemicStatus = "DISPUTED"
    StatusRefuted    EpistemicStatus = "REFUTED"
    StatusUnknown    EpistemicStatus = "UNKNOWN"
)
```

## **B. Domain Classification（領域分類）**

## **例如歷史小說可以保留：**

* ## **`FACT_HISTORY`**

* ## **`PLAUSIBLE_INFERENCE`**

* ## **`FICTION`**

## **修仙小說則可以有：**

* ## **`WORLD_CANON`**

* ## **`CHARACTER_SETTING`**

* ## **`WORLD_HYPOTHESIS`**

## **科學研究則可以有：**

* ## **`EXPERIMENTAL_RESULT`**

* ## **`THEORETICAL_MODEL`**

* ## **`RESEARCH_HYPOTHESIS`**

## **這兩個維度不能混為一談。**

## **例如一個小說設定可以是：**

* ## **知識狀態：`STIPULATED`**

* ## **領域分類：`WORLD_CANON`**

## **一個歷史主張可以是：**

* ## **知識狀態：`SOURCED`**

* ## **領域分類：`FACT_HISTORY`**

## **一個科學假說可以是：**

* ## **知識狀態：`HYPOTHESIS`**

* ## **領域分類：`RESEARCH_HYPOTHESIS`**

## **如此就不會讓歷史小說的知識分類限制整個 AIA。**

## **4\. `ConsistencyEngine` 也必須抽象化**

## **你貼出的 `ConsistencyEngine` 已經比原本的硬編碼檢查進步不少，但它仍然是歷史小說專用引擎。**

## **例如：**

```
if c.DeathYear > 0 && chap.StatedYear > c.DeathYear {
    // Post-Mortem Appearance
}
```

## **這條規則對歷史小說很有價值，但不能放在通用引擎裡。**

## **其他領域也有一致性檢查：**

| 領域 | 檢查範例 |
| ----- | ----- |
| **歷史小說** | **人物死亡後仍然登場** |
| **科幻小說** | **飛船違反已確立的航行限制** |
| **修仙小說** | **人物尚未達到設定境界卻使用高階能力** |
| **科學研究** | **結論與實驗數據不一致** |
| **風俗研究** | **某項習俗出現在尚未有該習俗的時代** |
| **思想研究** | **某思想主張與同一作者其他已確認論述矛盾** |
| **地理研究** | **地形、距離、地貌與資料來源不符** |
| **太空研究** | **軌道參數、時間、位置之間不一致** |

## **因此我會設計成：**

```
ConsistencyEngine
    ├── Generic Rules
    │     ├── Missing Reference
    │     ├── Contradictory Assertion
    │     ├── Invalid Relation
    │     └── Temporal Conflict
    │
    ├── Historical Rules
    │     ├── Post Mortem Appearance
    │     ├── Anachronism
    │     └── Historical Location Conflict
    │
    ├── Fiction Rules
    │     ├── World Canon Violation
    │     ├── Character Setting Conflict
    │     └── Ability Constraint Violation
    │
    └── Scientific Rules
          ├── Data-Conclusion Conflict
          ├── Hypothesis Falsification
          └── Methodology Violation
```

## **這裡的關鍵是：引擎負責執行規則，領域模組負責提供規則。**

## **不要在引擎內寫死所有領域的判斷。**

## **而且 `Finding` 不能只有 `Severity`。至少還要能區分：**

* ## **嚴重程度**

* ## **判斷信心**

* ## **證據完整程度**

* ## **是否需要人工裁定**

* ## **是否屬於刻意設定**

* ## **是否已被接受或駁回**

## **例如，發現一個疑似歷史矛盾，不代表它一定是錯誤；也可能是史料互相衝突，或作者刻意採用另一種合理推演。**

## **這個區分對 AIA 的自主修正非常重要。**

## **5\. `BookManifest` 應該演進成 `ProjectManifest`**

## **我不建議直接刪除 `BookManifest`，而是讓它成為通用專案 Manifest 的一種 Profile。**

## **例如：**

```
project:
  id: yuan-end
  name: 元末明初
  type: historical_fiction
  domain: writing
  language: zh-TW

profile:
  name: historical_fiction
  version: "1.0"

workspace:
  root: .
  version_control:
    type: git
    enabled: true

knowledge:
  entities: entities/
  evidence: evidence/
  timeline: timeline.md
  unresolved: unresolved.md

documents:
  outline: outline.md
  characters: characters.md
  volumes: volumes/

rules:
  - historical_accuracy
  - temporal_consistency
  - spatial_consistency
  - character_continuity

governance:
  require_diff_review: true
  require_human_approval_for_major_changes: true
```

## **科幻小說可以改成：**

```
project:
  id: space-opera
  name: 星際文明
  type: science_fiction
  domain: writing

profile:
  name: science_fiction
  version: "1.0"

knowledge:
  entities: entities/
  evidence: evidence/
  timeline: chronology.md

documents:
  outline: outline.md
  worldbuilding: worldbuilding.md
  characters: characters.md

rules:
  - world_canon_consistency
  - technology_constraints
  - temporal_consistency
  - spatial_consistency
```

## **科學研究則可以是：**

```
project:
  id: research-001
  name: 研究計畫
  type: scientific_research
  domain: research

profile:
  name: scientific_research
  version: "1.0"

knowledge:
  entities: entities/
  evidence: evidence/
  hypotheses: hypotheses/
  datasets: datasets/

documents:
  research_plan: research_plan.md
  methodology: methodology.md
  results: results.md
  discussion: discussion.md

rules:
  - evidence_traceability
  - methodology_consistency
  - data_conclusion_consistency
  - reproducibility
```

## **這樣一來，AIA 只要讀取 Manifest，就能知道專案需要哪些能力。**

## **我還會保留一個原則：Manifest 描述專案的結構與能力需求，不應該把所有知識內容塞進 YAML。實際資料仍然放在 Markdown、資料集或其他適合的檔案中。**

## **6\. 建議的 Go Package 結構**

## **我會避免把所有領域的模型塞進 `internal/domain/writing/`。**

## **建議方向如下：**

```
internal/domain/
├── project/
│   ├── project.go
│   ├── manifest.go
│   ├── profile.go
│   └── workspace.go
│
├── knowledge/
│   ├── entity.go
│   ├── relation.go
│   ├── assertion.go
│   ├── evidence.go
│   ├── temporal.go
│   ├── spatial.go
│   └── epistemic.go
│
├── consistency/
│   ├── finding.go
│   ├── rule.go
│   ├── result.go
│   └── engine.go
│
├── research/
│   ├── hypothesis.go
│   ├── methodology.go
│   └── experiment.go
│
└── writing/
    ├── character.go
    ├── chapter.go
    ├── narrative.go
    ├── worldbuilding.go
    ├── historical.go
    └── profile.go
```

## **這是目標結構，不代表現在就要全部建立。**

## **其中有兩個特別重要的界線：**

1. ## **`domain` 定義資料與業務語意，不負責掃描 Markdown、解析 YAML、執行檔案操作。**

2. ## **`ConsistencyEngine` 的具體解析器、規則執行器和檔案讀取應放在 `service` 或相應的 application/infrastructure 層，避免 domain package 直接依賴檔案系統與 YAML。**

## **你貼出的 `ConsistencyEngine` 同時做了文件解析、資料建模、規則判斷、檔案存取，這幾件事最好逐步拆開。**

## **7\. 我會如何安排實際演進**

## **我不建議現在就一次改完所有 Package。尤其你正在準備 Phase 6，這項調整應該先作為架構決策與重構範圍，不應直接擴大 Phase 6 的自我修改權限。**

| 階段 | 工作 | 驗收重點 |
| ----- | ----- | ----- |
| **A** | **定義 Project Knowledge Core** | **不含小說專用概念** |
| **B** | **將 BookManifest 抽象為 ProjectManifest** | **舊 `book.yaml` 可以相容載入** |
| **C** | **抽離 Knowledge 與 Consistency 模型** | **通用引擎不依賴歷史人物或卷章** |
| **D** | **建立 Historical Fiction Profile** | **原本元末專案的檢查能力不退化** |
| **E** | **建立 Science Fiction / Xianxia Profile** | **能檢查世界觀、規則與設定連續性** |
| **F** | **建立 Scientific Research Profile** | **能追蹤假說、證據、實驗與結論** |
| **G** | **再擴充風俗、思想、地理、太空研究** | **證明核心模型具有跨領域適用性** |

## **其中，最重要的驗收不是「能不能建立這些 Profile」，而是：**

## **換了一個領域，AIA 是否仍然能理解專案、維持長期連續性、發現矛盾、提出修正，並根據證據修正自己的判斷？**

## **這才是你最初想驗證的自主性。**

## **8\. 我的最終建議**

## **我會把這項架構演進定位為：**

## **從 Writing Agent 升級為 Project Intelligence Framework。**

## **Writing Agent 不會消失，而是成為第一個成熟的 Domain Agent。**

## **未來可以有：**

* ## **Historical Fiction Agent**

* ## **Science Fiction Agent**

* ## **Xianxia Worldbuilding Agent**

* ## **Scientific Research Agent**

* ## **Cultural Research Agent**

* ## **Geographic Research Agent**

* ## **Space Research Agent**

## **但它們共用同一套 Project Knowledge Core、Continuity、Memory、Consistency、Evidence、Learning 與 Autonomy Core。**

## **這也有一個很重要的好處：你不需要為每一個新興趣重新開發一套 Agent。只需要建立新的 Project Profile、規則、資料結構與工具，就能讓 AIA 開始一個新領域。**

## **我認為目前 `go/internal/domain/writing/` 不應該繼續直接擴充。應先完成抽象化設計，再把既有模型逐步遷移到通用核心。**

## **一、我會重新定義 Phase 6 的進入檢核**

Phase 6 的重點是安全自我修改（Safe Self-Modification）。

在進入之前，我不認為需要先把 Writing Agent、Coding Agent 做到成熟產品的程度。否則可能永遠進不了 Phase 6。

但必須確認：AIA 已經有能力在受治理的環境中，理解問題、提出變更、執行變更、驗證結果，並且保留失敗後的復原能力。

我建議將檢核分成六個面向。

| 檢核面向 | 必須確認的能力 | 目前判斷 |
| ----- | ----- | ----- |
| A. Autonomy Core | Observe、Recall、Assess、Deliberate、Decide 等循環 | 已有架構與實作基礎 |
| B. Continuity | 跨 Session 保留專案狀態、目標、決策與進度 | 有基礎，需實際長期驗證 |
| C. Initiative | 能發現未明確指定的問題與機會 | 已有初步實際案例 |
| D. Domain Execution | 能操作真實小說與程式碼專案 | 已有初步實際案例，成熟度待確認 |
| E. Verification | 能確認執行結果，而不是只回報成功 | 尚需強化驗證 |
| F. Governance | 變更權限、隔離、審核、回復及稽核 | 必須完成 Phase 6 入口檢核 |

這裡我特別強調 E 與 F。

AIA 能產生 `book.yaml` 是一個好結果；但我們還要確認它是否能檢查 YAML 格式、路徑是否存在、欄位是否符合 Manifest 規格，以及後續 Writing Agent 是否真的使用了這份設定。

同樣地，AIA 能產生程式碼，也不等於它已經能安全修改自己的核心。

## **二、我建議 Phase 6 前做五項實際檢核**

這五項不是要求你再開發五個新模組，而是利用現有系統做行為驗收。

### **1\. 真實專案理解檢核**

已有初步成功案例

以你的小說專案為例：

* 給定一個既有小說目錄。  
* 不預先提供完整 `book.yaml`。  
* 讓 AIA 理解目錄、檔案與專案目標。  
* 判斷缺少哪些專案資訊。  
* 自行提出 Manifest。  
* 經過必要確認後建立檔案。  
* 後續工作能讀取並遵循 Manifest。

重點不是能不能產生 YAML，而是能不能理解「這個專案需要什麼」。

### **2\. 未知需求適應檢核**

已有初步成功案例

刻意給 AIA 一個沒有預先設計好的情境。

例如：

> 我希望這部小說能夠持續追蹤人物年齡、官職變動與歷史事件，但目前沒有相關資料結構。

觀察 AIA 是否能：

1. 理解需求。  
2. 檢查現有架構。  
3. 發現能力缺口。  
4. 提出合理的資料模型。  
5. 判斷應該修改既有設計還是新增功能。  
6. 說明修改影響。  
7. 在授權範圍內執行。

這一項非常重要，因為它測試的不是預先寫好的工作流程，而是面對未知問題的能力。

### **3\. 執行與驗證檢核**

需要補強

讓 AIA 執行一項有明確驗收條件的工作。

例如建立 `book.yaml` 後：

* 確認檔案確實存在。  
* 確認格式有效。  
* 確認路徑與檔案關係正確。  
* 重新載入專案。  
* 確認系統確實讀取了新 Manifest。  
* 記錄執行結果。

這一項要確認 AIA 不會把「提出方案」當成「完成工作」。

### **4\. 自我修正檢核**

需要補強

故意讓 AIA 遇到一次失敗。

例如：

* 提供一個不符合規格的 Manifest。  
* 讓它嘗試載入。  
* 觀察錯誤訊息。  
* 要求它判斷失敗原因。  
* 提出修正。  
* 再次執行與驗證。

重要的是要保留第一次失敗的證據，不能只留下最後成功的結果。

這會驗證 Self-Correction 是否真正形成閉環。

### **5\. 安全修改與復原檢核**

Phase 6 入口必要條件

這項應該是 Phase 6 前最後一項檢核。

讓 AIA 在隔離環境中修改一個非核心模組，並確認：

* 修改前有基準版本。  
* 修改範圍受到限制。  
* 變更內容可以檢視。  
* 驗收條件明確。  
* 不符合條件時可以回復。  
* Audit 保留完整紀錄。  
* AIA 無法自行放寬自己的權限。

注意：這裡不需要讓 AIA 修改 Autonomy Core，也不需要讓它修改自己的 Governance。

## **三、Phase 6 的進入標準**

我建議不要使用「所有 Agent 已經完成」作為進入條件，而是使用以下 Gate。

Phase 6 Entry Gate

G1 — 專案理解與持續性

有初步證據

能理解真實專案，並在不同 Session 延續工作。

G2 — 主動性與未知需求適應

有初步證據

能發現未明確指定的需求與能力缺口。

G3 — 真實執行與結果驗證

待確認

能確認實際變更，而不是只產生文字回覆。

G4 — 失敗修正與學習

待確認

能保留失敗證據、修正方案並驗證改善。

G5 — 隔離、治理與復原

待確認

能在受限環境修改，並保有審核與回復能力。

這是目前依據程式碼檢視和你補充的實際操作經驗所做的判斷，不是已完成的正式驗收。尤其 G1、G2 仍需要把你觀察到的行為留下可追溯證據。

## **四、我認為還要增加一個很重要的觀念**

不要把 Phase 6 當成 AIA 開始學習修改程式碼的階段。

我認為 Phase 6 真正應該處理的是：

> AIA 能否在理解自身限制的前提下，主動提出對自身能力的改善，並且安全地完成改善、驗證改善，以及在改善失敗時恢復。

例如：

AIA 發現小說專案的歷史事件檢查能力不足。

它可以提出：

* 目前的 Observer 只能檢查人物檔案。  
* 無法交叉檢查章節與時間線。  
* 建議新增 Historical Timeline Index。  
* 建議修改 Manuscript Consistency Engine。  
* 預期改善哪些行為。  
* 需要修改哪些程式與資料結構。  
* 如何驗證修改有效。

接著才進入隔離、審核、執行、測試、評估與合併。

這樣 Phase 6 才是在建立真正的自我改善能力，而不是單純讓 Agent 可以寫自己的程式碼。

## **五、我的建議：先做一份正式檢核文件**

我建議不要立即修改 Master Implementation Plan，也不要急著開發新功能。

先新增：

`docs/phase6-readiness-assessment.md`

文件應包含：

1. Phase 6 進入目的與範圍。  
2. 既有架構及程式碼檢核。  
3. 六個檢核面向。  
4. 五項實際行為驗收。  
5. 每項驗收的操作方式與預期結果。  
6. 證據保存格式。  
7. Entry Gate 判定標準。  
8. 未通過項目的補強順序。  
9. Phase 6 正式啟動條件。

另外，應把你剛才提到的 WebUI 小說操作記錄成一筆真實的行為案例，包含：

* 原始使用者需求。  
* AIA 的理解與判斷。  
* 發現原本缺少的 agy 互動流程。  
* 發現缺少 `book.yaml`。  
* AIA 提出的方案。  
* 實際產生的檔案。  
* 後續是否成功載入與使用。

這筆資料很有價值，因為它是 AIA 從預先設計好的測試情境，走向真實使用者需求的早期證據。

我的結論是：你已經有理由開始準備 Phase 6，但還沒有足夠證據直接宣告 Phase 6 Ready。

我不希望因為 Coding Agent 和 Writing Agent 尚未成熟，就拖住整個 AIA；但也不希望因為 WebUI 已經能完成幾個漂亮的案例，就跳過安全自我修改的必要檢核。

先把這份 Readiness Assessment 做好，接下來就能非常明確地決定：哪些能力可以直接進 Phase 6，哪些需要先補強，以及哪些可以留到 Phase 6 之後逐步成熟。

