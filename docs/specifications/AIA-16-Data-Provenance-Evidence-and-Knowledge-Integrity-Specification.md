# AIA-16：Data Provenance, Evidence & Knowledge Integrity Specification

> [!NOTE]
> **架構重整與收攏公告 (Consolidated Architecture Specification)**
> 本規範為 AIA 歷史基準規範（規格編號已於 AIA-22 永久封頂）。為杜絕規格膨脹與重複定義，本規範之現行核心標準已正式收攏重整至 **[Volume 2: Cognitive Loop, Memory & Planning](file:///home/wade/src/github/aia/docs/specifications/volume-2-cognitive-loop-and-planning.md)**。最新架構與維護基準請以對應之 Volume 為主，完整六大核心卷軸總覽請參閱 [Specification Catalog (README.md)](file:///home/wade/src/github/aia/docs/specifications/README.md)。

**文件編號：** AIA-16  
**文件名稱：** Data Provenance, Evidence & Knowledge Integrity Specification  
**版本：** 2.0  
**文件狀態：** Architecture Specification  
**適用系統：** AIA (Autonomous Intelligence Architecture)  
**依賴文件：** AIA-01 ～ AIA-15  
**核心範圍：** Data Provenance、Evidence Management、Claim Validation、Knowledge Integrity

---

# 1. 文件目的

## 1.1 設計背景

AIA 的目標不是單純執行任務，而是建立一個能長期觀察、思考、行動、檢驗結果、修正自身並持續演進的自主系統。

這種系統必須長期累積資料、經驗、記憶、判斷與知識。

然而，長期自主運作會產生一項重要風險：

**錯誤資訊可能在反覆使用、摘要、記憶、學習與跨 Agent 傳遞之後，逐漸失去原始的不確定性，最後被當成已確認的事實。**

例如：

1. Agent 從新聞網站取得一項資訊。
2. Agent 將新聞摘要寫入 Memory。
3. 另一個 Agent 讀取 Memory，將摘要當成既有知識。
4. 第三個 Agent 根據這項知識提出策略。
5. 系統將策略結果寫入 Learning Record。
6. 未來 Agent 再引用這筆 Learning Record，認為該資訊已經經過驗證。

整個過程可能沒有任何一步真正驗證最初的資訊。

這會造成 Knowledge Drift（知識漂移）、False Learning（錯誤學習）與 Evidence Contamination（證據污染）。

因此，AIA 必須建立獨立的資料來源與證據管理機制。

## 1.2 核心目標

本文件定義：

1. Data Provenance：資料來源與來源追溯。
2. Evidence Management：證據管理與生命週期。
3. Claim Validation：主張與證據的支持關係。
4. Knowledge Integrity：知識完整性與可信狀態。
5. Temporal Validity：資料的時間有效性。
6. Memory Integrity：記憶的來源、可信度與更新管理。
7. Learning Integrity：學習紀錄與證據的關聯。
8. Cross-Agent Knowledge Integrity：跨 Agent 知識交換的完整性。

AIA 必須能回答：

- 這項資訊從哪裡來？
- 什麼時候取得？
- 原始資料是否仍然存在？
- 這項主張是否有證據支持？
- 證據是否獨立？
- 資料是否已經過時？
- 是否存在互相矛盾的來源？
- 這是事實、推論，還是假設？
- 哪個 Agent 曾經使用這項資料？
- 這項資訊是否影響過某個 Decision？
- 後續發現錯誤時，哪些 Memory、Learning Record 與 Decision 需要重新評估？

## 1.3 設計原則

AIA 必須遵守以下原則：

1. 所有重要知識都應保留來源。
2. 資料來源可信，不代表資料內容一定正確。
3. 有證據不代表證據足以支持某項主張。
4. 多次引用不等於多份獨立證據。
5. 模型生成內容不能自動升級為事實。
6. 推論與假設必須保留其性質。
7. 資料過時不代表歷史紀錄應被刪除。
8. 發生來源衝突時，必須保留衝突。
9. Knowledge Promotion 必須有明確驗證條件。
10. 所有重要知識變更都必須可追溯。
11. 貫徹一致性記憶五大檢驗標準（[docs/architecture/memory.md](file:///home/wade/src/github/aia/docs/architecture/memory.md)）：不失憶（歷史永久可溯）、不亂記（性質不混淆）、不亂改（版本化修訂因果鏈）、不混淆（跨領域邊界隔離）、不停止學習（新經驗修正舊認知）。

---

# 2. 核心概念與資料模型

## 2.1 Data

Data 是系統取得的原始或衍生資料。

例如：

- 網路新聞。
- 公司財報。
- 股票價格。
- 政府公告。
- 歷史文獻。
- Git Commit。
- 原始碼。
- 測試結果。
- Agent 對話。
- 使用者提供的資訊。
- 模型生成的分析內容。

Data 不一定具有證據效力。

例如，一段模型生成的歷史事件摘要是 Data，但不能因為它存在於系統內，就認定它是史實。

## 2.2 Source

Source 是 Data 的來源。

來源類型：

| 類型 | 說明 |
|---|---|
| PRIMARY_SOURCE | 原始資料來源 |
| OFFICIAL_SOURCE | 官方機構或正式公告 |
| SECONDARY_SOURCE | 二手資料或分析 |
| NEWS_SOURCE | 新聞媒體 |
| ACADEMIC_SOURCE | 學術研究 |
| DATABASE_SOURCE | 結構化資料庫 |
| USER_PROVIDED | 使用者提供 |
| SYSTEM_GENERATED | 系統產生 |
| MODEL_GENERATED | 模型生成 |
| TOOL_OUTPUT | 工具執行結果 |
| UNKNOWN_SOURCE | 無法確認來源 |

Source Registry 必須保存來源身分與基本資訊。

## 2.3 Evidence

Evidence 是可用於支持、反駁或限定某項主張的資料。

Evidence 必須具備：

- 可辨識的來源。
- 可追溯的內容。
- 擷取時間。
- 內容或引用範圍。
- 驗證狀態。
- 與相關 Claim 的關係。

Evidence 不等於 Truth。

一份官方公告可以證明官方曾發布某項公告，但不一定能證明公告中的所有敘述都符合客觀事實。

## 2.4 Claim

Claim 是系統提出或引用的一項可檢驗主張。

例如：

- 台積電某年度營收成長。
- 某歷史人物於某年擔任某官職。
- 某程式函式存在特定錯誤。
- 某項策略在特定條件下可能改善投資績效。

Claim 必須能連結到 Evidence，或明確標示目前沒有證據支持。

## 2.5 Assertion

Assertion 是 Agent 對某項 Claim 所做的判斷。

例如：

- ACCEPTED：目前接受。
- REJECTED：目前不接受。
- UNCERTAIN：尚無法確認。
- CONFLICTED：存在來源衝突。
- SUPERSEDED：已被更新版本取代。
- DEPRECATED：不再適合目前用途。

Assertion 代表系統在特定時間、根據特定證據所作的判斷，不代表永久真理。

## 2.6 Inference

Inference 是根據既有資料與推理過程產生的結論。

Inference 必須保留：

- 前提 Claim。
- 使用的 Evidence。
- 推理方法或模型。
- 推論時間。
- 不確定性。
- 適用條件。

推論不得直接偽裝成原始事實。

## 2.7 Hypothesis

Hypothesis 是尚待驗證的假設。

例如：

- 某項市場變化可能代表資金正在轉向特定產業。
- 某個人物的政治選擇可能受到地方利益影響。
- 某種程式架構可能改善 Agent 的長期記憶能力。

Hypothesis 可以引導 Initiative 與 Experiment，但不能直接升級為 Verified Knowledge。

## 2.8 Knowledge

Knowledge 是經過一定程度驗證、具有明確適用範圍，並且保留來源與可信狀態的資訊。

Knowledge 不應被理解為絕對真理。

任何 Knowledge 都可能因為新證據、資料更新或適用條件改變而被重新評估。

## 2.9 Decision

Decision 是 Agent 根據當時可取得的資料、Evidence、Knowledge、目標與限制所做的選擇。

Decision 必須保存其當時使用的資訊版本。

即使後來發現資料錯誤，也不能回頭改寫原始 Decision。

## 2.10 Learning Record

Learning Record 是系統從經驗、預測、執行結果與評估中形成的學習紀錄。

Learning Record 必須能回溯到：

- 原始 Decision。
- 當時使用的 Evidence。
- 當時的 Knowledge。
- 預期結果。
- 實際結果。
- 評估過程。
- 後續修正。

---

# 3. Data Provenance Model

## 3.1 定義

Data Provenance 是資料來源與流轉歷程的完整紀錄。

AIA 不應只保存最後得到的文字摘要，而應盡可能保存摘要與原始資料之間的關係。

例如：

新聞原文 → 擷取內容 → 正規化資料 → Agent 摘要 → Claim → Decision → Learning Record。

每個階段都應保留來源關聯。

## 3.2 Provenance Metadata

重要資料至少應保存以下欄位：

| 欄位 | 說明 |
|---|---|
| provenance_id | 來源追溯識別碼 |
| source_id | Source Registry 識別碼 |
| source_type | 來源類型 |
| source_uri | 原始來源位置 |
| publisher | 發布者 |
| published_at | 原始發布時間 |
| retrieved_at | 系統取得時間 |
| effective_at | 資料適用或生效時間 |
| content_hash | 原始內容雜湊 |
| content_version | 內容版本 |
| extraction_method | 取得方式 |
| transformation_chain | 資料轉換歷程 |
| responsible_agent | 負責取得或處理的 Agent |
| parent_evidence_ids | 上游 Evidence |
| validation_status | 驗證狀態 |

部分來源可能沒有發布時間、版本或固定 URI。

這些欄位可以為空，但不能自行推測後填入。

## 3.3 Content Hash

AIA 應使用內容雜湊確認資料是否發生變化。

建議：

- SHA-256 或同等安全雜湊演算法。
- 原始內容與正規化內容分別計算 Hash。
- 保存資料版本與 Hash 的對應。
- 內容更新時建立新版本。

Hash 只能證明內容是否與某個已記錄版本一致，不能證明內容本身正確。

## 3.4 Transformation Chain

任何重要資料轉換都應記錄：

- Input Evidence。
- Output Data。
- Transformation Type。
- Transformation Agent。
- Transformation Time。
- Tool 或 Model Version。
- Transformation Parameters。
- Output Hash。

例如：

```text
Original News
    |
    v
Extracted Article
    |
    v
Normalized Text
    |
    v
AI Summary
    |
    v
Claim Extraction
    |
    v
Evidence Validation
```

AI Summary 必須保留與原文的關聯，不得取代原始來源。

## 3.5 Provenance Graph

AIA 應支援 Provenance Graph。

節點可以包括：

- Source。
- Data。
- Evidence。
- Claim。
- Inference。
- Knowledge。
- Decision。
- Learning Record。

關係可以包括：

- DERIVED_FROM。
- SUPPORTS。
- CONTRADICTS。
- QUALIFIES。
- REFERENCES。
- USED_BY。
- PRODUCED_BY。
- SUPERSEDES。
- VALIDATED_BY。

這個圖形結構能讓系統從某項錯誤 Claim 反向追查所有受影響的知識與決策。

---

# 4. Evidence Lifecycle

## 4.1 生命週期

Evidence 必須有明確的生命週期。

```text
COLLECTED
    |
    v
PARSED
    |
    v
NORMALIZED
    |
    v
VALIDATED
    |
    v
REVIEWED
    |
    +----> VERIFIED
    |
    +----> REJECTED
    |
    +----> UNCERTAIN
    |
    +----> CONFLICTED
    |
    +----> DEPRECATED
```

## 4.2 狀態定義

| 狀態 | 說明 |
|---|---|
| COLLECTED | 已取得資料 |
| PARSED | 已完成內容解析 |
| NORMALIZED | 已完成格式正規化 |
| VALIDATED | 已完成基本格式、來源與完整性檢查 |
| REVIEWED | 已經過相關 Agent 或驗證流程檢視 |
| VERIFIED | 在特定範圍內已獲得足夠支持 |
| REJECTED | 已確認不適合用作證據 |
| UNCERTAIN | 尚不足以判斷 |
| CONFLICTED | 與其他 Evidence 存在未解決衝突 |
| DEPRECATED | 已不適合目前使用情境 |

VERIFIED 不表示永遠正確。

例如，一份已驗證的財報資料，可能因為公司後續發布更正公告而需要重新評估。

## 4.3 Evidence Validation

Evidence Validation 至少應檢查：

1. 來源是否可辨識。
2. 原始內容是否完整。
3. 是否有明確擷取時間。
4. 是否存在內容變更。
5. 是否符合預期資料類型。
6. 是否有明顯格式錯誤。
7. 是否能支持所引用的 Claim。
8. 是否存在其他相反證據。
9. 是否符合特定 Domain 的時效要求。

## 4.4 Review Responsibility

不同 Domain 可以使用不同 Agent 負責審查。

例如：

- Investment Agent：金融資料與市場資訊。
- Coding Agent：程式碼、測試與執行結果。
- Writing Agent：歷史資料、文獻與人物年表。

Autonomy Core 負責共通治理與資料流轉，不取代 Domain Agent 的專業判斷。

---

# 5. Source Reliability

## 5.1 來源可信度的多維度評估

AIA 不應使用單一可信度分數直接決定資料真假。

來源品質至少應考慮：

| 維度 | 說明 |
|---|---|
| Authority | 來源是否具備相關權威性 |
| Directness | 是否為直接來源 |
| Accuracy History | 過去資料準確紀錄 |
| Timeliness | 是否符合資料時效要求 |
| Independence | 是否獨立於其他來源 |
| Reproducibility | 是否可以重複取得或驗證 |
| Transparency | 是否公開資料方法與依據 |
| Scope | 是否適合支持目前 Claim |

不同 Claim 對來源品質的要求不同。

例如：

- 公司營收應優先使用公司公告或正式財報。
- 歷史事件應優先尋找原始史料、檔案與可信研究。
- 程式是否能執行，應以實際執行結果為依據。
- 市場價格應使用有明確時間戳的市場資料。

## 5.2 Source Authority 與 Claim Support 必須分開

來源權威性與主張支持程度是兩件事。

例如：

某政府網站具有高度權威性，但若引用的段落只說明政策發布日期，就不能用來證明政策實際執行效果。

因此必須分開評估：

- Source Reliability：來源本身是否可靠。
- Evidence Relevance：證據與 Claim 是否相關。
- Evidence Sufficiency：證據是否足以支持 Claim。

## 5.3 Source Independence

多個來源不一定代表多份獨立證據。

例如：

新聞網站 A 引用新聞社 B。
新聞網站 C 也引用新聞社 B。
新聞網站 D 再引用網站 A。

這四個網址可能只有一個原始資訊來源。

AIA 應盡可能辨識：

- 原始發布者。
- 轉載來源。
- 引用關係。
- 共同資料來源。
- 是否存在真正獨立的交叉驗證。

不得將同一來源的多次轉載計算成多份獨立證據。

## 5.4 Reliability Is Contextual

來源可信度必須依照 Claim 類型判斷。

同一個來源可能：

- 適合提供官方政策內容。
- 不適合獨立證明政策實際成效。
- 適合提供公司公開資訊。
- 不適合預測公司未來獲利。

因此 Source Registry 不應只保存一個永久可信度分數。

應保存來源特性、歷史表現與適用範圍。

---

# 6. Claim & Evidence Relationship

## 6.1 Claim Structure

每項重要 Claim 應具有：

| 欄位 | 說明 |
|---|---|
| claim_id | 主張識別碼 |
| claim_text | 主張內容 |
| claim_type | 主張類型 |
| domain | 所屬領域 |
| created_at | 建立時間 |
| created_by | 建立者 |
| evidence_ids | 相關 Evidence |
| assertion_status | 目前判斷狀態 |
| confidence | 判斷信心 |
| scope | 適用範圍 |
| valid_from | 有效起始時間 |
| valid_until | 有效結束時間 |
| supersedes | 被取代的 Claim |

Confidence 是 Agent 對目前判斷的信心表達，不是客觀真實機率，也不能代替證據品質評估。

## 6.2 Evidence Relationship

Evidence 與 Claim 的關係至少包括：

| 關係 | 說明 |
|---|---|
| SUPPORTS | 支持主張 |
| CONTRADICTS | 反駁主張 |
| QUALIFIES | 限定主張適用範圍 |
| CONTEXTUALIZES | 提供背景 |
| DERIVED_FROM | 主張由此資料推導 |
| DOES_NOT_ADDRESS | 未涉及該主張 |

一項 Claim 可以有多份 Evidence。

一份 Evidence 也可以支持多項 Claim。

## 6.3 Partial Support

Evidence 可能只支持 Claim 的部分內容。

例如：

Claim：

「某公司今年營收成長，而且未來獲利將持續增加。」

Evidence 可能只證明今年營收成長，不能直接支持未來獲利持續增加。

AIA 應將複合 Claim 拆分成更小、可檢驗的主張。

不能因為一部分成立，就認定整項 Claim 成立。

## 6.4 Claim Confidence

信心程度應考慮：

- Evidence Quality。
- Evidence Independence。
- Evidence Coverage。
- Contradictory Evidence。
- Temporal Validity。
- Inference Complexity。
- Known Limitations。

不得單純以 Evidence 數量增加 Confidence。

## 6.5 Claim Revision

當新證據出現時：

1. 建立新的 Assertion。
2. 保留舊 Assertion。
3. 記錄變更原因。
4. 更新 Claim 狀態。
5. 找出受影響的 Knowledge。
6. 找出受影響的 Decision 與 Learning Record。
7. 必要時觸發重新評估。

不得直接覆寫過去的判斷紀錄。

---

# 7. Fact、Inference、Hypothesis 分離

## 7.1 Claim Type

AIA 必須明確區分：

| 類型 | 說明 |
|---|---|
| FACT | 有直接證據支持的事實性敘述 |
| INFERENCE | 根據已知資料推導的結論 |
| HYPOTHESIS | 尚待驗證的假設 |
| OPINION | 評價、觀點或主觀判斷 |
| MODEL_OUTPUT | 模型生成內容 |
| USER_ASSERTION | 使用者提供或提出的主張 |
| UNKNOWN | 性質尚未確認 |

## 7.2 不得自動升級

以下轉換不得自動發生：

```text
MODEL_OUTPUT -> FACT
INFERENCE -> FACT
HYPOTHESIS -> FACT
OPINION -> KNOWLEDGE
USER_ASSERTION -> VERIFIED
```

每次升級都必須具備適當驗證條件。

## 7.3 使用者提供資訊

使用者提供的資訊具有明確來源，但不代表所有內容都已經獲得外部驗證。

例如：

使用者告知目前程式碼已有 Worker，但尚未提供更新版本。

AIA 可以記錄：

- Source：USER_PROVIDED。
- Claim：目前系統已有 Worker。
- Verification：尚未透過最新程式碼確認。
- Applicability：後續分析架構時應納入此資訊。

這樣既尊重使用者提供的資訊，也不會錯誤宣稱已經完成程式碼驗證。

## 7.4 Model Output

模型輸出可以是：

- 分析結果。
- 摘要。
- 推論。
- 假設。
- 建議。

模型輸出必須保留模型與生成時間等資訊。

若模型使用外部 Evidence，必須保留 Evidence 關聯。

模型重複生成相同內容，不構成獨立證據。

---

# 8. Memory Integrity

## 8.1 Memory 不等於 Truth

Memory 是 AIA 保存資訊與工作狀態的機制。

Memory 的存在不能證明內容正確。

因此 Memory 必須保存：

- 原始來源。
- 內容類型。
- 驗證狀態。
- 建立時間。
- 最後驗證時間。
- 適用範圍。
- 不確定性。
- 相關 Claim。
- 相關 Evidence。

## 8.2 Memory Categories

建議區分：

| 類型 | 說明 |
|---|---|
| FACTUAL_MEMORY | 已驗證的事實性資訊 |
| WORKING_MEMORY | 工作中暫存資訊 |
| INFERENTIAL_MEMORY | 推論結果 |
| HYPOTHESIS_MEMORY | 待驗證假設 |
| EXPERIENCE_MEMORY | 執行經驗 |
| DECISION_MEMORY | 決策與理由 |
| USER_CONTEXT_MEMORY | 使用者提供的工作背景 |
| PROCEDURAL_MEMORY | 已驗證的工作方法 |
| CONFLICTED_MEMORY | 存在未解決衝突的資訊 |

## 8.3 Memory Promotion

Memory Promotion 是將暫存或待驗證資訊提升為長期可引用知識的程序。

建議流程：

```text
Working Memory
    |
    v
Source Check
    |
    v
Evidence Validation
    |
    v
Claim Assessment
    |
    v
Knowledge Promotion
    |
    v
Long-term Memory
```

不是所有資訊都需要升級。

例如：

- 一次性的任務參數不必成為長期知識。
- 尚未驗證的市場推論應保留為假設。
- 使用者明確確認的專案決策可以保存，但應標示為使用者決策。
- 程式測試失敗的紀錄可以保存為 Experience，而不是永久認定程式碼有相同錯誤。

## 8.4 Stale Memory

資料可能因時間經過而失效。

例如：

- 公司管理階層變更。
- 股票持股數量變更。
- 程式架構已經重構。
- 歷史研究出現新史料。
- 專案規格已經修訂。

AIA 應支援：

- VALID：目前仍有效。
- STALE：可能已過時。
- EXPIRED：已超過有效期限。
- SUPERSEDED：已被新資訊取代。
- DISPUTED：存在爭議。
- UNKNOWN：有效性尚未確認。

## 8.5 Memory Correction

修正 Memory 時：

1. 保留原始紀錄。
2. 建立修正事件。
3. 記錄新 Evidence。
4. 更新目前有效狀態。
5. 保留舊版本與修正關係。
6. 重新評估受影響的 Knowledge 與 Decision。

Memory 不應採取無痕覆寫方式。

---

# 9. Learning Integrity

## 9.1 Learning 必須有證據鏈

AIA-11 定義 Self-Correction & Learning。

本文件進一步要求：

Learning Record 必須保留其資料來源與證據鏈。

至少包含：

| 欄位 | 說明 |
|---|---|
| learning_id | 學習紀錄識別碼 |
| experiment_id | 所屬 Experiment |
| decision_id | 原始 Decision |
| evidence_ids | 使用的 Evidence |
| prediction | 原始預測 |
| expected_outcome | 預期結果 |
| actual_outcome | 實際結果 |
| evaluation | 評估結果 |
| root_cause | 原因分析 |
| correction_id | 相關修正 |
| confidence | 學習結論信心 |
| created_at | 建立時間 |

## 9.2 Outcome Bias

結果好壞不能直接決定原本判斷是否正確。

例如：

某次投資決策獲利，不代表當時風險評估正確。

某次投資決策虧損，也不代表當時的分析一定錯誤。

Learning 必須同時檢查：

- 當時可取得的資訊。
- 當時的推論。
- 當時的風險。
- 預測是否合理。
- 執行是否符合決策。
- 實際結果。
- 結果與預測差異。

## 9.3 False Learning Prevention

AIA 必須防止：

1. 單次結果形成永久策略。
2. 錯誤資料形成錯誤規則。
3. 多次引用被誤認為多次驗證。
4. 模型生成內容被當成經驗證知識。
5. 只記錄成功案例而忽略失敗案例。
6. 將相關性誤認為因果關係。
7. 過度依賴近期結果。
8. 忽略樣本數與適用條件。

## 9.4 Learning Reassessment

當 Evidence 被撤銷、修正或標記為過時時：

AIA 應能找出相關 Learning Record，並判斷是否需要：

- 保留。
- 降低信心。
- 標記為待重新評估。
- 重新計算。
- 停止應用。
- 提出 Self-Correction。

不得因為新資料出現，就無條件刪除舊 Learning Record。

---

# 10. Cross-Agent Knowledge Integrity

## 10.1 基本原則

不同 Agent 可以共享知識，但不能只傳遞一段失去來源的文字。

跨 Agent 傳遞重要 Knowledge 時，至少必須包含：

- Claim。
- Evidence References。
- Provenance。
- Validation Status。
- Confidence。
- Domain。
- Applicable Scope。
- Temporal Validity。
- Known Limitations。

## 10.2 Knowledge Exchange

建議資料結構：

```text
Knowledge Package
├── knowledge_id
├── claim
├── claim_type
├── domain
├── evidence_refs[]
├── provenance_refs[]
├── validation_status
├── confidence
├── valid_scope
├── valid_from
├── valid_until
├── limitations[]
└── originating_agent
```

## 10.3 Domain Applicability

一個 Domain 的知識不能無條件套用到另一個 Domain。

例如：

Investment Agent 發現某種策略在特定市場環境有效，不代表 Coding Agent 可以直接套用相同決策邏輯。

Writing Agent 對歷史人物動機的推論，也不能直接當成已驗證的歷史事實。

跨 Domain 使用時必須重新評估適用性。

## 10.4 Independent Evidence Preservation

若 Agent A 與 Agent B 都引用同一份來源：

系統必須保留共同來源關係。

不能因為兩個 Agent 都提出相同結論，就認定有兩份獨立證據。

## 10.5 Untrusted Agent Output

其他 Agent 的輸出預設為待評估資料。

接收方必須：

1. 檢查來源。
2. 檢查 Claim Type。
3. 檢查 Evidence。
4. 檢查適用範圍。
5. 判斷是否可以採用。

Agent 身分本身不構成證據。

---

# 11. Temporal Validity

## 11.1 時間維度

AIA 必須區分：

| 時間欄位 | 說明 |
|---|---|
| Published Time | 資料發布時間 |
| Retrieved Time | 系統取得時間 |
| Effective Time | 資料實際生效時間 |
| Observed Time | 系統觀察到的時間 |
| Recorded Time | 系統保存紀錄的時間 |
| Valid Time | 資料適用的時間範圍 |

這些時間不能混為一談。

例如：

某公司於 5 月 10 日發布 4 月營收。

- Published Time：5 月 10 日。
- Effective Period：4 月。
- Retrieved Time：AIA 實際取得資料的時間。

Agent 在 5 月 5 日不能使用 5 月 10 日才發布的資料，回頭假裝當時已經知道。

## 11.2 Bitemporal Data

對重要資料，建議採用 Bitemporal Data Model：

- Valid Time：資料在現實世界中適用的時間。
- Transaction Time：資料進入 AIA 的時間。

這能區分：

「某項事情何時發生」

與

「AIA 何時知道這件事情」。

## 11.3 Temporal Evidence

資料時效要求依 Domain 不同而異。

例如：

- 股價：需要精確交易時間或市場時間。
- 公司財報：需要發布時間與財務期間。
- 歷史文獻：需要版本、年代與出版資訊。
- 程式碼：需要 Commit、Branch 與執行時間。

## 11.4 Look-ahead Bias Prevention

Investment Agent 必須防止 Look-ahead Bias（前視偏誤）。

任何歷史模擬或策略評估，只能使用當時已經公開且可取得的資料。

例如：

若某公司財報於 5 月 10 日公開，4 月 30 日的投資決策不能使用該財報內容。

回測必須保留：

- 決策時間。
- 當時可取得資料。
- 資料發布時間。
- 資料擷取時間。
- 使用資料版本。
- 策略版本。

若無法確認歷史資料的可取得時間，必須標記資料限制，不得宣稱回測完全無前視偏誤。

## 11.5 Historical Knowledge Revision

歷史研究可能出現新史料或新解讀。

AIA 必須保留：

- 舊有說法。
- 新增史料。
- 研究者或來源。
- 修正時間。
- 修正原因。
- 目前採用的判斷。

不能把新研究出現後的判斷，回填成過去已經確定的知識。

---

# 12. Domain-Specific Requirements

## 12.1 Investment Domain

Investment Agent 的資料完整性要求較高，因為資料時間錯誤可能直接造成錯誤投資判斷。

### 12.1.1 Data Sources

應區分：

- 股票行情。
- 指數行情。
- 公司財報。
- 公司重大訊息。
- 法說會資料。
- 公開資訊觀測站公告。
- 交易所公告。
- 新聞。
- 國際市場資料。
- 產業資料。
- 利率、匯率與總體經濟資料。

### 12.1.2 Market Data Metadata

市場資料至少應保留：

- 股票代碼。
- 市場。
- 交易日期。
- 交易時間。
- 價格類型。
- 成交量。
- 幣別。
- 資料來源。
- 是否為即時或延遲資料。
- 是否為調整後價格。
- 除權息處理方式。
- 資料擷取時間。

### 12.1.3 Corporate Actions

必須處理：

- 除權。
- 除息。
- 股票分割。
- 減資。
- 現金增資。
- 停牌。
- 下市。
- 代號變更。

否則歷史價格與持股數量可能產生錯誤。

### 12.1.4 Investment Decision Evidence

每次投資決策應保存：

1. 決策時間。
2. 當時現金。
3. 當時持股。
4. 市場資料快照。
5. 台積電 2330 相關資訊。
6. 其他相關標的資訊。
7. 使用的新聞與財報。
8. Claim 與 Inference。
9. 投資假設。
10. 預期報酬。
11. 風險評估。
12. Decision。
13. 執行結果。

### 12.1.5 Virtual Portfolio

目前投資實驗為 NT$100,000 虛擬資金。

Agent 可以自主決定：

- 買進。
- 賣出。
- 增持。
- 減持。
- 持有。
- 保留現金。
- 觀察其他標的。

每天應觀察與判斷，但不強迫每天成交。

所有模擬交易必須保留當時資料與模擬成交規則。

### 12.1.6 Performance Evaluation

總資產是主要評估目標。

但評估時必須同時保留：

- 當時可用資訊。
- 決策品質。
- 執行品質。
- 風險暴露。
- 現金比例。
- 交易成本假設。
- 預測校準。
- 實際績效。

不能只依照最後資產增加或減少，決定 Agent 是否學習正確。

## 12.2 Coding Domain

Coding Agent 必須保留程式碼與執行結果的來源。

### 12.2.1 Code Evidence

至少包括：

- Repository。
- Branch。
- Commit Hash。
- File Path。
- File Hash。
- Dependency Version。
- Runtime Version。
- Tool Version。
- 修改前後 Diff。
- Test Output。
- Build Output。
- Runtime Log。
- Error Trace。

### 12.2.2 Execution Evidence

必須區分：

| 類型 | 說明 |
|---|---|
| CODE_INSPECTION | 靜態閱讀程式碼 |
| TEST_EXECUTION | 實際執行測試 |
| BUILD_EXECUTION | 實際建置結果 |
| RUNTIME_EXECUTION | 實際執行結果 |
| MODEL_ANALYSIS | 模型分析 |
| USER_REPORTED | 使用者回報 |

模型認為程式碼應該能執行，不代表已經執行成功。

靜態閱讀認為沒有錯誤，也不代表不存在 Runtime Error。

### 12.2.3 Version Binding

所有 Coding Experiment 必須綁定程式碼版本。

否則未來無法判斷：

- 哪個版本產生問題。
- 哪次修正解決問題。
- 哪個測試結果適用於哪個 Commit。
- 是否因為 Dependency 變動而造成結果不同。

## 12.3 Writing Domain

Writing Agent 必須區分歷史史料、合理推演與小說虛構。

### 12.3.1 Historical Evidence

史料至少應保存：

- 作者或編者。
- 書名。
- 版本。
- 出版資訊。
- 卷、篇、章或頁碼。
- 原文。
- 翻譯。
- 引用範圍。
- 取得來源。
- 研究者或解讀者。

### 12.3.2 Writing Claim Types

應支援以下標籤：

| 標籤 | 性質 |
|---|---|
| 【史實骨架】 | 有史料支持的歷史事件或基本事實 |
| 【合理推演】 | 根據史實與時代背景所作的合理推演 |
| 【小說虛構】 | 作者為敘事創作的虛構內容 |

三種標籤不得混淆。

例如：

某人物確實於某年擔任某官職，可以標示為【史實骨架】。

某人物可能因為地方利益而做出某種選擇，如果沒有直接史料支持，應標示為【合理推演】。

完全虛構的人物、對話與事件，應標示為【小說虛構】。

### 12.3.3 Historical Revision

發現史實錯誤時，Writing Agent 必須：

1. 找出原始敘述。
2. 找出原始來源。
3. 提供新 Evidence。
4. 比較不同史料。
5. 判斷是否存在史學爭議。
6. 修正 Claim。
7. 找出受影響的人物、章節與卷次。
8. 提出修訂建議。
9. 保留原版本與修訂紀錄。

不得因為一份新資料，就任意推翻所有既有設定。

### 12.3.4 Fictional Consistency

小說內部設定也應保留來源與版本。

例如：

- 人物年齡。
- 人物關係。
- 人物活動時間。
- 地理位置。
- 政治派系。
- 各卷人物進度。
- 已發生的劇情。
- 尚未揭露的設定。

這些內容屬於小說內部 Knowledge，不應誤認為歷史史實。

---

# 13. Conflict Resolution

## 13.1 衝突不可直接消除

不同來源可能對同一事件提出不同說法。

AIA 不應為了產生單一答案，就任意選擇其中一方並刪除其他說法。

必須保留：

- 各來源的原始主張。
- 各來源的發布時間。
- 來源性質。
- 證據支持範圍。
- 相互矛盾之處。
- 尚未解決的問題。

## 13.2 Conflict Classification

建議分類：

| 類型 | 說明 |
|---|---|
| FACTUAL_CONFLICT | 事實內容衝突 |
| TEMPORAL_CONFLICT | 時間資訊衝突 |
| SCOPE_CONFLICT | 適用範圍不同 |
| DEFINITION_CONFLICT | 名詞定義不同 |
| SOURCE_CONFLICT | 來源本身存在差異 |
| INTERPRETATION_CONFLICT | 對資料的解讀不同 |
| VERSION_CONFLICT | 資料版本不同 |

## 13.3 Resolution Procedure

```text
Conflict Detected
       |
       v
Identify Claims
       |
       v
Compare Evidence
       |
       v
Check Source Independence
       |
       v
Evaluate Scope & Time
       |
       v
Assess Evidence Quality
       |
       +----> Resolved
       |
       +----> Partially Resolved
       |
       +----> Unresolved
```

## 13.4 Unresolved Conflict

如果無法合理解決：

- 保留多種說法。
- 標記 CONFLICTED。
- 記錄各說支持程度。
- 限制相關 Claim 的使用方式。
- 必要時要求 Human Review。

系統可以提出暫時採用的工作假設，但必須明確標示為暫定判斷。

---

# 14. Integrity & Security

## 14.1 Immutable History

重要 Evidence、Decision、Learning Record 與 Audit Event 應採 Append-only 設計。

修正資料時，建立新版本或修正事件。

不得無痕修改歷史。

## 14.2 Integrity Verification

系統應定期檢查：

- Content Hash。
- Provenance Link。
- Evidence Reference。
- Claim Relationship。
- Knowledge Version。
- Learning Record Reference。

若資料完整性檢查失敗，應標記為 INTEGRITY_ERROR。

## 14.3 Access Control

不同 Agent 只能依照 Policy 存取必要資料。

Evidence 的存在不代表所有 Agent 都有權讀取其原始內容。

敏感資料應依照既有 AIA Policy 管理。

## 14.4 Prompt Injection & Untrusted Content

外部資料可能包含惡意指令或 Prompt Injection。

AIA 必須區分：

- 資料內容。
- Agent 指令。
- 系統政策。
- 使用者授權。

外部來源內容只能作為資料，不得因為被引用或保存為 Evidence，就取得指令權限。

## 14.5 Audit Integrity

Evidence 與 Knowledge 的狀態變更應留下 Audit。

Agent 不得：

- 刪除不利證據。
- 修改原始 Evidence。
- 隱藏來源衝突。
- 偽造驗證結果。
- 修改過去 Audit。
- 將未驗證資料標記為 VERIFIED。

---

# 15. Go Implementation Architecture

## 15.1 設計原則

AIA-16 是資料與知識完整性規格。

它不要求重新建立既有 Memory、Task、Worker、Audit 或 Agent Registry。

應在既有架構上新增或整合 Evidence 與 Provenance 能力。

## 15.2 建議元件

```text
Autonomy Core
    |
    +-- Provenance Service
    |
    +-- Source Registry
    |
    +-- Evidence Registry
    |
    +-- Claim Store
    |
    +-- Validation Pipeline
    |
    +-- Knowledge Promotion
    |
    +-- Conflict Resolver
    |
    +-- Temporal Validator
    |
    +-- Integrity Monitor
```

## 15.3 Component Responsibilities

### Source Registry

管理資料來源。

職責：

- 建立 Source。
- 保存來源資訊。
- 記錄來源類型。
- 保存來源品質特性。
- 維護來源關聯。
- 記錄來源歷史變化。

### Evidence Registry

管理 Evidence。

職責：

- 建立 Evidence。
- 保存原始內容或內容引用。
- 保存 Hash。
- 保存擷取資訊。
- 管理 Evidence Lifecycle。
- 提供 Evidence Lookup。

### Claim Store

管理 Claim 與 Assertion。

職責：

- 建立 Claim。
- 建立 Assertion。
- 管理 Evidence Relationship。
- 保存信心與適用範圍。
- 管理 Claim Version。
- 管理衝突狀態。

### Provenance Service

管理來源追溯。

職責：

- 建立 Provenance Chain。
- 記錄 Data Transformation。
- 建立來源關聯。
- 查詢資料來源。
- 追蹤衍生資料。

### Validation Pipeline

負責資料驗證。

職責：

- 格式檢查。
- 來源檢查。
- 時間檢查。
- Hash 檢查。
- Claim Support 檢查。
- Evidence Conflict 檢查。

### Knowledge Promotion

管理資訊從暫存到長期 Knowledge 的流程。

職責：

- 檢查來源。
- 檢查 Evidence。
- 檢查 Claim。
- 評估適用範圍。
- 決定是否可以提升。
- 保留提升紀錄。

### Conflict Resolver

負責辨識與管理來源衝突。

職責：

- 偵測衝突。
- 分類衝突。
- 比較 Evidence。
- 提出解決建議。
- 保留未決狀態。

### Temporal Validator

負責時間有效性。

職責：

- 檢查資料時間。
- 檢查有效期間。
- 偵測過時資料。
- 防止未來資料污染歷史決策。
- 支援 Bitemporal Query。

### Integrity Monitor

負責完整性檢查。

職責：

- 驗證 Hash。
- 檢查引用鏈。
- 偵測孤立資料。
- 偵測錯誤關聯。
- 產生 Integrity Alert。

## 15.4 與現有元件整合

| 現有元件 | AIA-16 整合方式 |
|---|---|
| Memory | 保存 Knowledge 與來源關聯 |
| Task | 保存任務使用的 Evidence |
| Worker | 執行資料取得、驗證與整理 |
| Agent Registry | 管理來源與驗證責任 Agent |
| Project | 綁定專案 Knowledge |
| Experiment | 保存實驗使用的資料快照 |
| Decision | 保存決策時的 Evidence |
| Learning | 保存學習來源與驗證結果 |
| Evaluation | 評估證據品質與知識可靠性 |
| Audit | 保存來源與知識狀態變更 |
| Policy | 管理資料存取與驗證要求 |

Worker 是既有執行單位，可以承載 Evidence Collection、Validation、Integrity Check 等工作。

本文件不重新定義 Worker 的生命週期，也不假設目前程式碼已經實作上述全部服務。

## 15.5 建議資料結構

以下為概念性 Go 結構，實際欄位應依既有資料模型整合。

```go
type Evidence struct {
    ID              string
    SourceID        string
    Domain          string
    EvidenceType    string

    URI             string
    ContentHash     string
    ContentVersion  string

    PublishedAt     *time.Time
    RetrievedAt     time.Time
    EffectiveAt     *time.Time

    ValidationState string
    ParentIDs       []string

    CreatedBy       string
    CreatedAt       time.Time
}

type Claim struct {
    ID             string
    Text           string
    Domain         string
    ClaimType      string

    Status         string
    Confidence     float64

    EvidenceIDs    []string
    Scope          string

    ValidFrom      *time.Time
    ValidUntil     *time.Time

    CreatedBy      string
    CreatedAt      time.Time
}

type EvidenceRelation struct {
    EvidenceID string
    ClaimID    string
    Relation   string
    Scope      string
    Notes      string
}

type ProvenanceRecord struct {
    ID              string
    InputIDs        []string
    OutputID        string

    Transformation  string
    AgentID         string
    ToolVersion     string

    CreatedAt       time.Time
}
```

以上為邏輯模型，不代表必須直接採用相同 Go Struct。

實際實作時應考慮：

- ID 與資料庫主鍵一致性。
- 時區與時間精度。
- 大型原始內容的外部儲存。
- Hash 演算法版本。
- 多版本資料關聯。
- 資料庫交易與一致性。
- Append-only Event Storage。

## 15.6 API Responsibilities

建議提供下列服務介面：

```text
SourceService
    RegisterSource
    GetSource
    UpdateSourceMetadata
    AssessSource

EvidenceService
    RegisterEvidence
    GetEvidence
    ValidateEvidence
    FindEvidence
    DeprecateEvidence

ClaimService
    CreateClaim
    AddEvidenceRelation
    EvaluateClaim
    ReviseClaim
    FindConflicts

ProvenanceService
    RecordTransformation
    GetProvenanceChain
    TraceDependencies

KnowledgeService
    PromoteKnowledge
    ReassessKnowledge
    FindRelatedKnowledge

IntegrityService
    VerifyHash
    VerifyReferences
    ScanIntegrity
```

服務介面應與儲存層分離，避免未來更換資料庫或實作語言時影響整體架構。

---

# 16. Metrics & Evaluation

AIA 必須能量測資料與知識完整性。

不能只看系統保存了多少筆 Memory 或 Evidence。

## 16.1 Provenance Coverage

重要 Claim 中具有完整來源追溯資訊的比例。

```text
Provenance Coverage =
Claims with Valid Provenance
/
Total Important Claims
```

## 16.2 Evidence Traceability

能夠從 Knowledge 追溯到原始 Evidence 的比例。

```text
Evidence Traceability =
Traceable Knowledge Records
/
Total Knowledge Records Requiring Evidence
```

## 16.3 Claim Support Rate

具有足夠 Evidence 支持的 Claim 比例。

```text
Claim Support Rate =
Supported Claims
/
Claims Requiring Evidence
```

必須清楚定義 Supported 的判斷標準，不得只計算是否存在引用。

## 16.4 Stale Knowledge Rate

已過時或可能過時的 Knowledge 比例。

```text
Stale Knowledge Rate =
Stale Knowledge Records
/
Total Time-sensitive Knowledge Records
```

## 16.5 Conflict Detection Rate

系統能正確辨識已知來源衝突的比例。

應使用已知衝突案例建立 Benchmark。

## 16.6 Correction Propagation Accuracy

當原始 Evidence 發生修正時，系統能正確找到受影響 Claim、Memory、Decision 與 Learning Record 的程度。

## 16.7 Evidence Independence Accuracy

系統辨識共同來源與轉載關係的能力。

## 16.8 Temporal Integrity

系統在歷史決策與回測中，是否錯誤使用決策時間之後才可取得的資料。

Investment Domain 必須特別評估此項指標。

## 16.9 Evaluation Principles

評估時必須注意：

- 引用數量不等於證據品質。
- 高信心不等於高正確率。
- 來源數量不等於來源獨立性。
- 知識量增加不等於知識品質提升。
- 沒有發現衝突不代表不存在衝突。
- 系統承認不知道，有時比產生錯誤答案更正確。

---

# 17. Implementation Phases

AIA-16 建議分階段實作。

## Phase E0：基本來源追溯

目標：

- Source Registry。
- Evidence Registry。
- 基本 Provenance Metadata。
- Content Hash。
- Evidence Reference。

驗收：

- 重要資料能追溯來源。
- 原始資料與摘要可以建立關聯。
- 能辨識來源缺失。

## Phase E1：Claim & Evidence Management

目標：

- Claim Store。
- Evidence Relationship。
- Claim Type。
- Assertion Status。
- 基本衝突管理。

驗收：

- Claim 可以連結多份 Evidence。
- Evidence 可以支持或反駁 Claim。
- 能區分 FACT、INFERENCE、HYPOTHESIS。

## Phase E2：Memory & Learning Integrity

目標：

- Memory Provenance。
- Knowledge Promotion。
- Stale Memory。
- Learning Record Evidence Link。
- Correction Propagation。

驗收：

- Memory 能保留來源與可信狀態。
- Learning Record 能追溯原始 Decision。
- Evidence 修正能觸發相關知識重新評估。

## Phase E3：Temporal Integrity

目標：

- Bitemporal Data。
- Temporal Validator。
- Historical Evidence Snapshot。
- Look-ahead Bias Prevention。

驗收：

- 能還原指定時間點可取得的資料。
- 能識別決策時尚未公開的資訊。
- 能將發布時間與有效時間分開保存。

## Phase E4：Cross-Agent Knowledge Integrity

目標：

- Knowledge Package。
- Provenance Exchange。
- Source Independence。
- Domain Applicability。
- Untrusted Agent Output。

驗收：

- 跨 Agent 傳遞知識時不遺失來源。
- 能辨識共同引用來源。
- 不會因為多個 Agent 重複引用就自動提高證據可信度。

## Phase E5：Integrity Monitoring & Evaluation

目標：

- Provenance Graph。
- Integrity Monitor。
- Conflict Detection。
- Knowledge Reassessment。
- Metrics & Benchmark。

驗收：

- 能從錯誤 Claim 追溯受影響資料。
- 能檢查引用鏈完整性。
- 能量測資料與知識完整性。
- 能定期提出知識重新評估工作。

以上階段是建議實作順序，不代表目前系統已完成任何尚未確認的功能。

---

# 18. Acceptance Criteria

AIA-16 的實作至少必須符合以下條件。

## 18.1 Data Provenance

- [ ] 重要資料具有來源識別碼。
- [ ] 保存資料取得時間。
- [ ] 保存原始內容或可追溯引用。
- [ ] 支援內容 Hash。
- [ ] 支援資料版本。
- [ ] 支援資料轉換歷程。

## 18.2 Evidence Management

- [ ] Evidence 具有生命週期狀態。
- [ ] 支援 Evidence 驗證。
- [ ] 支援 Evidence 棄用。
- [ ] 支援多份 Evidence。
- [ ] 支援反向證據。
- [ ] 支援來源衝突。

## 18.3 Claim Integrity

- [ ] Claim 可連結 Evidence。
- [ ] Claim 能區分事實、推論與假設。
- [ ] Claim 支援版本化。
- [ ] Claim 支援重新評估。
- [ ] 不會將模型生成內容自動視為事實。

## 18.4 Memory & Learning

- [ ] Memory 保留來源。
- [ ] Memory 保留不確定性。
- [ ] Memory 支援過時狀態。
- [ ] Learning Record 可以回溯 Evidence。
- [ ] 修正 Evidence 能觸發相關知識重新評估。
- [ ] 不會無痕覆寫歷史紀錄。

## 18.5 Temporal Integrity

- [ ] 區分發布時間與取得時間。
- [ ] 支援資料有效期間。
- [ ] 能還原歷史時間點資料。
- [ ] 投資回測能避免未來資料污染。
- [ ] 程式碼測試結果能對應特定版本。

## 18.6 Cross-Agent Integrity

- [ ] 跨 Agent 知識交換保留 Provenance。
- [ ] 能辨識共同來源。
- [ ] 不會將 Agent 輸出自動視為已驗證知識。
- [ ] 跨 Domain 使用知識時會檢查適用性。

## 18.7 Security

- [ ] 重要資料變更具有 Audit。
- [ ] 原始 Evidence 不可無痕修改。
- [ ] 外部資料不能覆蓋系統指令。
- [ ] Agent 不得修改 Audit。
- [ ] Agent 不得自行取消治理限制。

---

# 19. Risks & Design Considerations

## 19.1 Evidence Accumulation Cost

長期運作會累積大量資料。

應區分：

- 必須保存的原始資料。
- 可以保存引用與 Hash 的資料。
- 可以壓縮保存的衍生資料。
- 可以依保留政策封存的資料。

不得為了節省空間而破壞重要來源追溯鏈。

## 19.2 Over-Validation

不是每一項資訊都需要完整人工驗證。

AIA 應依據：

- 重要性。
- 風險。
- 使用頻率。
- 決策影響。
- 資料不確定性。

決定驗證成本。

例如，投資決策使用的財務資料需要較高驗證要求；一般工作筆記則不必採取同等程度的驗證流程。

## 19.3 False Confidence

來源數量、模型信心與重複引用可能產生虛假的確定感。

系統必須保留：

- 未知。
- 不確定。
- 相互矛盾。
- 證據不足。
- 適用範圍限制。

## 19.4 Knowledge Drift

即使原始資料正確，長期摘要、轉述與推論也可能逐步偏離原意。

因此重要 Knowledge 必須能回溯原始 Evidence，而不是只依賴層層摘要。

## 19.5 System Complexity

Provenance、Evidence、Claim、Knowledge 都是不同概念。

實作時不能為了簡化而將它們全部合併成單一 Memory Record。

但也不應為了理論完整性，建立大量彼此重複的服務。

應以現有資料模型與 Runtime 能力為基礎，逐步增加必要的資料關係與驗證流程。

---

# 20. Final Design Principles

AIA-16 最終確立以下原則：

1. **資料不是知識。** 取得資料不代表已經理解或驗證。
2. **來源不是證據。** 來源必須與具體主張建立支持關係。
3. **證據不是絕對真理。** 證據有範圍、時間與限制。
4. **推論不是事實。** 推論必須保留前提與不確定性。
5. **記憶不是驗證。** Memory 不能因為長期保存就變成真理。
6. **重複不是獨立。** 多個 Agent 或網站引用同一來源，不等於多份獨立證據。
7. **學習必須可追溯。** Learning Record 必須能回到當時的 Evidence 與 Decision。
8. **時間不可回填。** Agent 不能使用未來才取得的資料，假裝過去已經知道。
9. **修正不能抹除歷史。** 新知識可以取代舊判斷，但必須保留修正過程。
10. **不確定性必須存在。** AIA 必須能承認資料不足、證據衝突與無法判斷。

AIA 的長期自主能力，不只是能記住更多資料，而是能持續判斷自己為什麼相信某件事、根據什麼做出決策，以及在新證據出現時如何修正原本的認知。

**AIA 必須能追溯自己的知識，而不只是累積自己的知識。**

這是長期自主運作、持續學習與自我修正的必要基礎。
