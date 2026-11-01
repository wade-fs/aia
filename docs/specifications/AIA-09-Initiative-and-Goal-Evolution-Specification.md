# AIA-09
# Initiative & Goal Evolution Specification
## 主動性與目標演化規格書

**文件版本：** v2.0  
**文件代號：** AIA-09  
**文件性質：** Autonomy Core 子系統規格  
**適用範圍：** AIA Runtime、Autonomy Core、Continuity、所有 Domain Agent  
**主要依據：** AIA-01、AIA-02、AIA-08、AIA-10

---

# 1. 文件目的

本文件定義 AIA 如何具備主動發現工作、評估行動機會、選擇執行方式，以及持續演化目標的能力。

AIA 不應只在使用者提出要求後才開始工作，也不應只是依照固定排程執行任務。

AIA 應能持續觀察自身所管理的 Project、Goal、Task、Memory、Decision 與外部環境，判斷是否存在值得處理的問題或機會。

當發現值得處理的事項時，AIA 應：

1. 確認目前狀態。
2. 找出問題、機會或目標差距。
3. 產生候選行動。
4. 評估行動的價值、成本、風險與優先順序。
5. 決定執行、延後、觀察、詢問人類，或不採取行動。
6. 執行後驗證結果。
7. 根據結果重新評估原有目標與策略。

本文件也定義 Goal Evolution，使 AIA 能夠在長期運作過程中持續檢視目標是否仍然合理，以及是否需要調整。

本文件不負責定義完整的 Self-Correction 機制。自我修正與學習將由 AIA-11 定義。

## 1.1 核心定位

AIA 的主動性不是固定排程，而是自主判斷。

AIA 的目標演化不是任意改變使用者要求，而是在既定治理規則下，持續改善達成目標的方法與目標結構。

因此：

- Initiative 負責發現值得做的事。
- Goal Evolution 負責檢視與調整目標。
- Deliberation 負責評估候選行動。
- Decision 負責決定是否行動。
- Continuity 負責保存目標、決策與工作之間的時間關係。
- Execution 負責實際執行。
- Self-Correction 負責從執行結果中修正判斷與方法。

上述能力共同構成 AIA 的自主行為。

# 2. 核心概念與設計原則

## 2.1 Initiative 不等於 Schedule

Schedule 是時間觸發機制。

Initiative 是自主發現與提出行動的能力。

例如：

每天早上 09:00 啟動 Investment Agent，是 Schedule。

Investment Agent 發現台積電近期出現重大變化，主動檢查持股風險並提出調整方案，是 Initiative。

兩者不能混為一談。

排程可以讓 Agent 有機會進行判斷，但不能保證 Agent 具有主動性。

## 2.2 Goal Evolution 不等於 Task Generation

自動產生 Task 不代表目標演化。

例如：

- 將每天的投資觀察轉成 Task，不是 Goal Evolution。
- 發現目前投資策略長期無法達成目標，重新評估策略，是 Goal Optimization。
- 發現原本的工作目標已經失去意義，提出目標修訂，是 Goal Revision。

Goal Evolution 必須建立在原有目標、歷史決策、實際成果與環境變化之上。

## 2.3 目標與策略必須分離

AIA 必須區分：

| 層級 | 定義 | 變更方式 |
|---|---|---|
| Mission | AIA 長期存在目的 | 人類設定 |
| Strategic Goal | 使用者設定的主要目標 | 重大變更須核准 |
| Operational Goal | 為達成主要目標而設定的階段目標 | 可依政策調整 |
| Strategy | 達成目標的方法 | 可自主評估及修正 |
| Task | 具體執行工作 | 依授權執行 |
| Action | 單次操作 | 依 Policy 控制 |

Agent 可以自主改變 Strategy，不代表可以擅自改變 Strategic Goal。

## 2.4 目標演進必須保留歷史

AIA 不得直接覆寫既有 Goal。

每次變更都必須建立新的 Goal Version，保留：

- 原始目標。
- 變更前目標。
- 變更後目標。
- 變更原因。
- 使用的證據。
- 相關歷史決策。
- 預期效果。
- 風險評估。
- 核准者。
- 生效時間。
- 後續評估結果。

AIA 必須能回答：

「為什麼當時設定這個目標？後來為什麼改變？改變後是否真的比較好？」

# 3. Initiative Model

Initiative Engine 是 Autonomy Core 中負責主動發現與提出工作的核心元件。

它不直接執行所有工作，而是負責從現有狀態中找出值得評估的事項。

## 3.1 Initiative 的輸入來源

Initiative Engine 應整合以下資訊：

| 資訊來源 | 可發現的事項 |
|---|---|
| Project State | 專案停滯、進度異常、缺少必要工作 |
| Goal State | 目標差距、目標失效、階段目標未完成 |
| Task State | 長期未完成、重複失敗、依賴阻塞 |
| Decision History | 過去判斷可能需要重新評估 |
| Memory | 新舊資訊矛盾、重要知識尚未使用 |
| Environment | 市場、程式、文件或其他外部環境變化 |
| Execution Result | 執行失敗、品質下降、成果超出預期 |
| Schedule | 定期檢查、週期性評估 |
| User Interaction | 新需求、修正意見、方向變化 |

Initiative Engine 不應只依賴單一資訊來源。

## 3.2 Initiative 類型

### A. Issue Discovery

發現已存在的問題。

例如：

- 投資組合出現異常風險。
- Coding Project 出現重複錯誤。
- Writing Project 發現人物年齡與歷史年表矛盾。
- 長期 Task 持續無法完成。

### B. Opportunity Discovery

發現值得利用的機會。

例如：

- 新的市場資訊可能改變投資判斷。
- 發現可以簡化程式架構的技術。
- 發現新的史料可以補足小說人物的歷史背景。

### C. Goal Gap Discovery

發現目前成果與目標之間存在差距。

例如：

- 投資總資產低於原先預期。
- 程式品質無法滿足專案要求。
- 小說進度與整體時間軸不一致。

### D. Continuity Discovery

發現長期工作中存在尚未處理的事項。

例如：

- 某個 Project 長期沒有更新。
- 某項決策尚未驗證。
- 某個人物設定尚未完成史實校勘。

### E. Improvement Discovery

發現自身工作方式可能需要改善。

例如：

- 現有投資觀察流程缺少國際市場資訊。
- Coding Agent 重複使用低效率的除錯方式。
- Writing Agent 缺乏跨卷人物年齡一致性檢查。

這類事項應進一步交由 Self-Correction 或相關服務處理。

# 4. Opportunity / Issue / Goal Gap Discovery

## 4.1 Discovery Model

每次發現事項都應建立 Discovery Record。

必要欄位：

| 欄位 | 說明 |
|---|---|
| Discovery ID | 唯一識別碼 |
| Project ID | 所屬 Project |
| Discovery Type | Issue / Opportunity / Goal Gap / Continuity / Improvement |
| Description | 發現內容 |
| Evidence | 支持發現的證據 |
| Related Goal | 相關目標 |
| Related Decision | 相關歷史決策 |
| Severity | 影響程度 |
| Confidence | 判斷信心 |
| Discovered At | 發現時間 |
| Status | 狀態 |

Discovery 本身不是 Task，也不代表必須採取行動。

## 4.2 Discovery Confidence

AIA 應區分：

- Confirmed：有充分證據支持。
- Probable：有合理證據，但仍需確認。
- Potential：可能存在問題或機會。
- Unverified：資訊不足，不能形成可靠判斷。

信心不足時，Agent 可以選擇補充資訊、持續觀察或詢問人類。

不可將推測直接視為事實。

## 4.3 重複 Discovery

同一問題可能被多次發現。

AIA 應將相同或高度相關的 Discovery 合併或建立關聯，避免不斷產生重複 Task。

但不得因為曾經發現過，就永久忽略後續變化。

已關閉的問題如果重新出現，應建立新的 Discovery，並連結過去紀錄。

# 5. Candidate Action Generation

Discovery 之後，AIA 可以產生一個或多個 Candidate Action。

Candidate Action 是候選行動，不是已授權執行的 Task。

## 5.1 Candidate Action 欄位

每個候選行動至少包含：

| 欄位 | 說明 |
|---|---|
| Candidate ID | 唯一識別碼 |
| Discovery ID | 來源 Discovery |
| Project ID | 所屬 Project |
| Goal ID | 相關目標 |
| Action Description | 預定行動 |
| Expected Outcome | 預期成果 |
| Estimated Cost | 預估成本 |
| Estimated Duration | 預估時間 |
| Risk | 風險 |
| Dependencies | 前置依賴 |
| Reversibility | 是否可逆 |
| Required Permission | 所需權限 |
| Confidence | 預期成果信心 |
| Opportunity Cost | 放棄其他工作的代價 |

## 5.2 候選行動類型

- EXECUTE：執行具體工作。
- INVESTIGATE：進一步調查。
- VALIDATE：驗證既有判斷。
- REVISE：修正策略或工作內容。
- DECOMPOSE：將大型目標拆分。
- ESCALATE：提交人類處理。
- MONITOR：持續觀察。
- SLEEP：暫停，等待適當條件。

候選行動可以有多種替代方案。

AIA 不應只提出自己最先想到的方案，而應在合理成本內比較不同選擇。

# 6. Deliberation

Deliberation 是對候選行動進行審慎評估的過程。

AIA 必須判斷：

「這件事值得現在做嗎？」

## 6.1 評估維度

| 維度 | 評估內容 |
|---|---|
| Goal Alignment | 是否有助於達成目標 |
| Expected Value | 預期價值 |
| Urgency | 時效性 |
| Importance | 長期重要性 |
| Risk | 可能造成的損失 |
| Cost | 時間、運算、金錢等成本 |
| Dependency | 是否有前置工作 |
| Reversibility | 是否容易撤回 |
| Confidence | 判斷可靠程度 |
| Opportunity Cost | 執行此事會放棄什麼 |

不能只根據緊急程度決定優先順序。

長期重要但不緊急的工作，也應保有適當資源。

## 6.2 Expected Value

AIA 可以採用預期價值模型輔助評估：

\[
EV = P_s \times V - C - R
\]

其中：

- \(P_s\)：成功機率。
- \(V\)：預期價值。
- \(C\)：執行成本。
- \(R\)：風險折減值。

這只是決策輔助，不應被視為所有 Domain 都能共用的唯一評分公式。

Investment、Coding、Writing 可以有不同的價值評估方法。

## 6.3 資訊不足時的選擇

當資訊不足以支持行動時，AIA 可以：

1. 進一步調查。
2. 等待新資訊。
3. 進行低成本驗證。
4. 詢問人類。
5. 暫不處理。

不應為了維持活動量而強行產生工作。

# 7. Prioritization

AIA 可能同時管理多個 Project，因此必須處理跨 Project 的資源競爭。

## 7.1 優先順序原則

優先順序應考慮：

1. 使用者明確要求的工作。
2. 治理、安全及資料完整性相關事項。
3. 具有時效性的重大事件。
4. 對長期目標有重大影響的工作。
5. 已承諾但尚未完成的工作。
6. 一般改善與探索工作。

這不是不可變更的固定排序。

實際優先順序仍需依據具體風險、時間及資源判斷。

## 7.2 多 Project 資源配置

AIA 應能同時維護多個 Project 的狀態。

例如：

- Investment Project：每日市場觀察。
- Coding Project：程式開發與維護。
- Writing Project：歷史小說與史實校勘。

Agent 不應因為目前正在處理 Investment，就忘記 Coding 或 Writing 的長期工作。

資源不足時，應保存未執行的 Candidate Action，並記錄延後原因。

## 7.3 長期停滯

當 Project 長期沒有進展時，AIA 應主動檢查：

- 是否缺少必要資訊。
- 是否存在未解決依賴。
- 是否有更高優先順序的工作。
- 原定目標是否仍然有效。
- 是否需要人類協助。
- 是否應暫停或結束 Project。

不能單純以 Project 長期沒有活動，就判定它已經失敗。

# 8. Decision Model

完成評估後，AIA 必須做出明確決策。

## 8.1 決策類型

| Decision | 意義 |
|---|---|
| ACT | 現在執行 |
| DEFER | 延後執行 |
| MONITOR | 持續觀察 |
| ASK_HUMAN | 需要人類判斷或核准 |
| NO_ACTION | 判斷目前不需要行動 |
| SLEEP | 暫停自主評估，等待喚醒條件 |

## 8.2 ACT

符合以下條件時可以 ACT：

- 與既定目標一致。
- 資訊足以支持判斷。
- 執行風險在授權範圍內。
- 資源足夠。
- 所需權限已具備。

## 8.3 DEFER

當工作值得執行，但現在不是適當時機，可以延後。

必須記錄：

- 延後理由。
- 預計重新評估時間。
- 重新評估條件。
- 是否存在其他依賴。

## 8.4 MONITOR

當環境可能發生變化，但尚未達到採取行動的條件時，應持續觀察。

例如：

台積電股價出現波動，但現有證據尚不足以支持買賣決策。

Investment Agent 可以選擇觀察，而不是為了每天交易而強行下單。

## 8.5 ASK_HUMAN

以下情況應交由人類：

- 重大目標方向改變。
- 超出既有資源預算。
- 需要新增權限。
- 涉及不可逆重大操作。
- 證據相互矛盾且無法合理消解。
- 涉及治理政策的變更。

## 8.6 NO_ACTION

NO_ACTION 是正式決策，不是執行失敗。

Agent 可以根據證據判斷：

- 現有策略仍然合理。
- 目前沒有值得處理的問題。
- 預期收益不足以抵銷成本。
- 行動可能造成不必要風險。
- 等待更多資訊更有價值。

NO_ACTION 必須保留決策紀錄。

## 8.7 SLEEP

SLEEP 是自主運作中的休眠決策。

Agent 可以在沒有值得處理的事項時，降低評估頻率或暫停工作。

但必須保留喚醒條件，例如：

- 到達下一次預定檢查時間。
- 出現重大環境事件。
- 某項 Task 狀態改變。
- 使用者提出新要求。
- 某個外部條件達到指定門檻。

休眠不是放棄目標。

# 9. No-Action Decision

No-Action 是衡量自主性的重要指標。

如果 AIA 每次被喚醒都一定要產生 Task，代表它可能只是自動化排程器。

真正的自主決策必須允許：

「經過觀察與評估，目前沒有必要做任何事。」

## 9.1 No-Action Record

必要欄位：

- Decision ID。
- Project ID。
- Related Goal。
- Evaluation Scope。
- Evidence。
- Reason。
- Confidence。
- Conditions for Reconsideration。
- Next Review Time。
- Created At。

## 9.2 不可無限期忽略

NO_ACTION 不代表永久關閉問題。

當環境發生變化、證據更新或目標狀態改變時，AIA 應重新評估。

例如：

今天不調整投資組合，不代表明天也不需要調整。

## 9.3 No-Action 品質

評估 No-Action 時，不能只看最後是否獲利或是否成功。

還必須檢查：

- 當時是否掌握足夠資訊。
- 判斷理由是否合理。
- 是否忽略重要風險。
- 重審條件是否適當。
- 後續是否依條件重新評估。

# 10. Goal Model

AIA 必須維持可追溯的目標結構。

## 10.1 Goal Hierarchy

```text
Mission
  └── Strategic Goal
        └── Operational Goal
              └── Subgoal
                    └── Task
                          └── Action
```

Goal 與 Task 不應混為一談。

Goal 描述希望達成的狀態。

Task 描述為達成狀態而執行的工作。

## 10.2 Goal Persistence

Goal Persistence 是維持目標連續性的能力。

AIA 必須能夠：

- 記得目標的原始設定。
- 追蹤目標目前狀態。
- 追蹤目標與 Project 的關係。
- 保留目標相關決策。
- 在 Session 結束後恢復目標脈絡。
- 在長時間沒有活動後重新評估目標。

## 10.3 Goal Optimization

Goal Optimization 是改善達成目標的方法。

例如：

Investment Agent 的目標是最大化總資產。

它可以自主評估：

- 投資標的。
- 持股比例。
- 現金配置。
- 進出場策略。
- 觀察頻率。
- 資訊來源。
- 風險評估方法。

這些都是策略層面的調整。

不能因為某個策略表現不好，就擅自改變使用者設定的主要投資目標。

## 10.4 Goal Revision

Goal Revision 是修改目標本身。

可能原因：

- 使用者提出新需求。
- 原始目標已不具可行性。
- 外部環境出現重大變化。
- 目標彼此衝突。
- 原始假設被新證據推翻。

Agent 可以提出修訂方案，但重大目標變更必須取得人類核准。

## 10.5 Goal Decomposition

大型目標可以拆解成多個子目標。

例如 Coding Project：

```text
完成某項系統功能
  ├── 確認需求
  ├── 檢查既有架構
  ├── 設計實作方式
  ├── 完成程式
  ├── 除錯與驗證
  └── 更新文件
```

Agent 可以根據執行結果重新調整子目標，但不能無故改變最上層目標。

## 10.6 Goal Retirement

目標可能因為完成、失效或被取代而結束。

Goal Retirement 必須保留：

- 結束原因。
- 最終狀態。
- 實際成果。
- 未完成事項。
- 後續承接目標。
- 歷史版本。

不可直接刪除已結束的目標。

# 11. Goal Evolution Policy

## 11.1 可自主調整的範圍

在既定 Policy 與授權內，Agent 可以自主：

- 調整執行順序。
- 調整工作方法。
- 拆分或合併子目標。
- 提出新候選目標。
- 改變策略。
- 重新評估工作優先順序。
- 暫停低價值工作。
- 提出目標修訂建議。

## 11.2 必須取得人類核准的範圍

以下變更必須經過人類核准：

- Mission 變更。
- 主要 Strategic Goal 變更。
- 超出預算或資源授權。
- 金融風險授權擴張。
- 新增外部系統權限。
- 修改 AIA 治理政策。
- 修改 Audit、Approval 或安全限制。
- 重大且不可逆的外部操作。

## 11.3 不得自主執行的事項

Agent 不得：

- 自行解除 Policy。
- 自行提高權限。
- 自行修改人類核准紀錄。
- 自行刪除歷史 Decision。
- 自行隱藏失敗紀錄。
- 以 Goal Evolution 為理由規避治理限制。

Goal Evolution 不得成為權限擴張的藉口。

# 12. Goal Conflict Resolution

多個目標可能彼此衝突。

例如：

- 提高投資報酬可能增加風險。
- Coding Project 的快速交付可能降低程式品質。
- Writing Project 的劇情需要可能與史實證據衝突。
- 多個 Project 可能同時需要有限的運算資源。

## 12.1 衝突分類

| 類型 | 說明 |
|---|---|
| Goal vs Goal | 不同目標彼此衝突 |
| Goal vs Policy | 目標要求超出治理範圍 |
| Goal vs Resource | 資源不足以同時完成 |
| Goal vs Evidence | 目標假設與新證據衝突 |
| Short-term vs Long-term | 短期成果與長期利益衝突 |

## 12.2 處理原則

1. Policy 優先於所有 Agent Goal。
2. 人類核准的 Strategic Goal 優先於 Agent 自行提出的目標。
3. 已承諾的重要工作應納入評估。
4. 不可只追求短期指標而犧牲長期目標。
5. 無法合理解決時，應 ASK_HUMAN。

## 12.3 證據與目標的關係

目標可以持續存在，但其假設必須接受新證據檢驗。

例如：

Writing Agent 原本依據某份史料設定人物活動時間，後來發現史料年代有誤。

Agent 不應為了維持既有劇情而忽略新證據，而應重新評估人物設定與相關章節。

但歷史人物的真實生平不能因為劇情需要而被任意修改。

# 13. Initiative Trigger

Initiative Engine 可以由多種事件觸發。

## 13.1 Event-driven Trigger

由事件觸發：

- Task 完成或失敗。
- Project State 改變。
- Goal 狀態改變。
- 新增重要 Memory。
- 新的外部資訊進入系統。
- Decision 到達重新評估條件。

## 13.2 Environment-driven Trigger

由外部環境變化觸發：

- 市場出現重大變化。
- 重要軟體版本更新。
- 程式依賴出現安全問題。
- 新史料或歷史證據出現。

## 13.3 Periodic Trigger

由固定時間喚醒：

- 每日投資檢查。
- 每週 Project Review。
- 每月 Goal Review。

週期性喚醒只是提供判斷機會。

Agent 不必每次都執行工作。

## 13.4 State-driven Trigger

由內部狀態觸發：

- 長期未完成的 Task。
- 反覆發生的錯誤。
- 目標進度異常。
- 記憶衝突。
- 資源不足。
- Project 長期停滯。

## 13.5 Trigger Deduplication

AIA 必須避免大量重複 Trigger 造成無意義的重複工作。

應支援：

- 相同事件合併。
- 事件去重。
- 冷卻時間。
- 重試上限。
- 優先級調整。
- 事件過期處理。

# 14. Integration with Continuity

AIA-10 定義 Continuity 的持久化與恢復方式。

Initiative Engine 必須讀取 Continuity 提供的狀態，而不是自行建立另一套互不相通的 Project 或 Goal 管理資料。

## 14.1 讀取 Project State

Initiative Engine 必須能取得：

- Project 目前狀態。
- 目前工作進度。
- 未完成 Task。
- 重要依賴。
- 最近一次活動。
- 歷史成果。
- 長期停滯原因。

## 14.2 讀取 Goal State

必須取得：

- Goal 原始版本。
- Goal 目前版本。
- Goal 狀態。
- 子目標。
- 預期成果。
- 目前差距。
- 歷史修訂。

## 14.3 讀取 Decision History

必須能檢查：

- 過去採取過哪些行動。
- 曾經拒絕哪些候選方案。
- 哪些決策尚未驗證。
- 哪些決策需要重新評估。
- 過去的策略是否有效。

## 14.4 Context Reconstruction

當 Agent 在新的 Session 或 Process 中恢復工作時，必須重建：

1. 目前 Mission 與 Strategic Goal。
2. 相關 Project 狀態。
3. 未完成 Task。
4. 最近重要 Decision。
5. 尚未處理的 Discovery。
6. 尚待評估的 Candidate Action。
7. 已設定的重新檢查條件。

不能只依賴最近一次對話內容判斷下一步。

# 15. Integration with Domain Agents

Autonomy Core 提供共同的主動性與目標演化能力。

Domain Agent 負責提供各領域的知識、評估方法與執行能力。

## 15.1 Investment Agent

Investment Agent 的主要目標：

**最大化投資組合總資產。**

實驗條件：

- 初始資金 NT$100,000。
- 使用虛擬資金。
- 投資台灣證券市場。
- 特別關注台積電 2330。
- 每個證券營業日進行觀察與判斷。
- 自主決定買進、賣出、增持、減持或持有現金。
- 每月 1 日由使用者檢查總資產。

Initiative 應包括：

- 主動發現市場變化。
- 主動尋找投資機會。
- 檢查持股風險。
- 評估是否調整投資組合。
- 檢討過去投資判斷。
- 評估投資資訊來源是否足夠。

Goal Evolution 可以調整策略與操作方法，但不能擅自改變最大化總資產的主要目標。

虛擬投資實驗不得被視為真實金融交易授權。

## 15.2 Coding Agent

Coding Agent 的主要目標由使用者指定。

Initiative 應包括：

- 主動檢查程式錯誤。
- 發現架構缺陷。
- 發現重複程式碼。
- 檢查依賴與安全問題。
- 發現測試不足。
- 評估改善程式品質的機會。
- 主動提出技術債處理方案。

Goal Evolution 可以調整開發順序、拆分工作及提出架構改善。

程式碼修改必須遵守：

- Workspace 隔離。
- Version Control。
- Policy。
- 測試與驗證。
- Diff Review。
- Recovery 與 Rollback。
- 必要的人類核准。

修改 AIA 自身核心程式與修改一般 Domain Project 程式碼，必須採取不同授權層級。詳細規格由 AIA-14 定義。

## 15.3 Writing Agent

Writing Agent 的主要目標是完成長期創作 Project。

Initiative 應包括：

- 主動檢查各卷進度。
- 檢查人物年齡與活動時間。
- 檢查人物關係。
- 檢查歷史事件順序。
- 發現史實與小說設定衝突。
- 發現尚未完成的章節。
- 檢查跨卷敘事連續性。
- 提出劇情與人物設定改善方案。

Goal Evolution 可以提出：

- 章節順序調整。
- 人物設定修正。
- 各卷篇幅調整。
- 新增或刪除小說虛構角色。
- 補充歷史背景。
- 調整敘事節奏。

但必須保留史實、合理推演與小說虛構的界線。

未經查證的推論不可被標記為史實。

重大劇情方向與既有內容變更應保留版本及修訂紀錄。

# 16. Initiative Trace

所有重要 Initiative 都必須具有可追溯紀錄。

## 16.1 Initiative Trace 欄位

- Initiative ID。
- Trigger ID。
- Project ID。
- Related Goal ID。
- Discovery IDs。
- Candidate Action IDs。
- Deliberation Summary。
- Decision。
- Decision Reason。
- Confidence。
- Created At。
- Execution Reference。
- Verification Reference。

## 16.2 完整追溯鏈

```text
Trigger
  ↓
Discovery
  ↓
Candidate Actions
  ↓
Deliberation
  ↓
Decision
  ↓
Task / No-Action / Defer / Monitor
  ↓
Execution Result
  ↓
Verification
  ↓
Learning / Goal Review
```

每個環節都應保留關聯識別碼。

AIA 必須能回答：

- 為什麼發現這件事？
- 為什麼認為它值得處理？
- 為什麼選擇這個方案？
- 為什麼沒有選擇其他方案？
- 為什麼最後沒有採取行動？
- 執行後是否達到預期？

# 17. Goal Evolution Record

每次 Goal 變更都必須建立獨立紀錄。

## 17.1 必要欄位

- Goal Evolution ID。
- Goal ID。
- Previous Version。
- Proposed Version。
- Change Type。
- Change Reason。
- Supporting Evidence。
- Expected Outcome。
- Risk Assessment。
- Approval Status。
- Approved By。
- Effective At。
- Review Date。

## 17.2 Change Type

- OPTIMIZATION：改善目標達成方法。
- DECOMPOSITION：拆分目標。
- REVISION：修訂目標。
- MERGE：合併目標。
- RETIREMENT：結束目標。
- REPLACEMENT：以新目標取代舊目標。

## 17.3 目標演進歷史

AIA 應提供可讀取的目標演進紀錄。

例如：

```text
Goal v1.0
  原始目標：自主投資並追求資產成長
      ↓
Goal v1.1
  調整實驗方式：改採虛擬投資
      ↓
Goal v1.2
  保留原始投資目標
  強化自主觀察、決策與自我評估
```

以上是目標演進的示意。

實際 Goal Version 必須根據使用者的正式決策建立，不能由 Agent 自行補寫歷史。

# 18. Human Oversight & Approval

人類是 AIA 的最終治理者。

Agent 可以主動提出建議，但不能自行取得超出授權的權力。

## 18.1 Approval Levels

| 等級 | 行為 | 權限 |
|---|---|---|
| L0 | 觀察、分析、評估 | 自主 |
| L1 | 建立候選行動、一般 Task | 自主 |
| L2 | 在既有授權內執行可逆工作 | 依 Policy 自主 |
| L3 | 重大策略或資源變更 | 人類核准 |
| L4 | 權限、治理、安全邊界變更 | 人類明確核准，部分事項禁止 Agent 自行修改 |

## 18.2 Approval 不等於執行

人類核准某項 Goal Evolution，只代表允許該變更。

實際執行仍需經過：

- Policy Check。
- Resource Check。
- Dependency Check。
- Execution Authorization。

## 18.3 Approval Expiration

核准可能具有有效期限。

如果核准後環境已發生重大變化，AIA 應重新評估，不應盲目沿用過期核准。

# 19. Resource & Budget Control

Initiative 必須考慮執行資源。

資源包括：

- CPU。
- Memory。
- Token。
- API 使用量。
- 網路流量。
- 磁碟空間。
- 執行時間。
- 外部服務費用。
- 人類審核時間。

## 19.1 Resource Budget

每個 Project 可以有獨立資源預算。

Autonomy Core 應知道：

- 可用資源。
- 已使用資源。
- 預留資源。
- 預估成本。
- 超出預算的風險。

## 19.2 Resource Exhaustion

資源不足時，AIA 應：

1. 暫停低優先級工作。
2. 保留重要 Project 狀態。
3. 優先處理必要的安全與資料完整性工作。
4. 記錄未執行原因。
5. 必要時通知人類。

不能為了維持自主性而無限制消耗資源。

# 20. Go Implementation 建議

本章定義邏輯元件，不代表目前程式碼已經完成這些能力。

實際實作狀態仍需依據最新 Source Code 確認。

## 20.1 Initiative Engine

負責統合 Discovery、Candidate、Deliberation 與 Decision。

建議介面：

```go
type InitiativeEngine interface {
    Discover(
        ctx context.Context,
        scope InitiativeScope,
    ) ([]Discovery, error)

    GenerateCandidates(
        ctx context.Context,
        discoveries []Discovery,
    ) ([]CandidateAction, error)

    Deliberate(
        ctx context.Context,
        candidates []CandidateAction,
    ) ([]DeliberationResult, error)

    Decide(
        ctx context.Context,
        results []DeliberationResult,
    ) ([]InitiativeDecision, error)
}
```

## 20.2 Discovery Service

```go
type DiscoveryService interface {
    DiscoverIssues(
        ctx context.Context,
        scope InitiativeScope,
    ) ([]Discovery, error)

    DiscoverOpportunities(
        ctx context.Context,
        scope InitiativeScope,
    ) ([]Discovery, error)

    DiscoverGoalGaps(
        ctx context.Context,
        scope InitiativeScope,
    ) ([]Discovery, error)
}
```

## 20.3 Candidate Service

```go
type CandidateService interface {
    Generate(
        ctx context.Context,
        discovery Discovery,
    ) ([]CandidateAction, error)

    Validate(
        ctx context.Context,
        candidate CandidateAction,
    ) error
}
```

## 20.4 Deliberation Service

```go
type DeliberationService interface {
    Evaluate(
        ctx context.Context,
        candidate CandidateAction,
    ) (DeliberationResult, error)

    Compare(
        ctx context.Context,
        candidates []CandidateAction,
    ) ([]RankedCandidate, error)
}
```

## 20.5 Decision Service

```go
type DecisionService interface {
    Decide(
        ctx context.Context,
        result DeliberationResult,
    ) (InitiativeDecision, error)

    Record(
        ctx context.Context,
        decision InitiativeDecision,
    ) error
}
```

## 20.6 Goal Evolution Service

```go
type GoalEvolutionService interface {
    Review(
        ctx context.Context,
        goal Goal,
    ) (GoalReview, error)

    Propose(
        ctx context.Context,
        review GoalReview,
    ) (GoalEvolutionProposal, error)

    Approve(
        ctx context.Context,
        proposalID string,
        approval Approval,
    ) error

    Apply(
        ctx context.Context,
        proposalID string,
    ) error
}
```

Apply 必須檢查核准狀態及 Policy，不得僅因 Agent 呼叫就直接修改 Goal。

## 20.7 核心資料模型

建議定義：

```go
type InitiativeScope struct {
    ProjectIDs []string
    GoalIDs    []string
    TriggerID  string
}

type Discovery struct {
    ID          string
    ProjectID   string
    Type        string
    Description string
    EvidenceIDs []string
    Confidence  float64
    Status      string
}

type CandidateAction struct {
    ID              string
    DiscoveryID     string
    ProjectID       string
    GoalID          string
    Description     string
    ExpectedOutcome string
    EstimatedCost   float64
    Risk            string
    Reversibility   string
}

type InitiativeDecision struct {
    ID          string
    CandidateID string
    Decision    string
    Reason      string
    Confidence  float64
    ReviewAt    *time.Time
}

type GoalEvolutionProposal struct {
    ID              string
    GoalID          string
    PreviousVersion int
    ProposedVersion int
    ChangeType      string
    Reason          string
    EvidenceIDs     []string
    ApprovalStatus  string
}
```

上述為邏輯示例。

實際型別應配合 AIA 現有 Domain Model、Repository、Worker 與 Runtime 設計，不應為了符合文件而重複建立相同功能。

## 20.8 Worker Integration

Initiative Engine 可以由現有 Worker 執行。

但 Worker 的職責是執行工作，不等於具備 Initiative 能力。

建議分工：

- Runtime：管理生命週期與喚醒。
- Worker：執行分派的工作。
- Initiative Engine：判斷值得做什麼。
- Goal Evolution Service：管理目標演進。
- Continuity：保存長期狀態。
- Policy：控制授權。
- Audit：保存不可任意竄改的紀錄。

Worker 可以承載 Initiative 工作，但不能以 Worker 存在就認定 AIA 已具備自主性。

# 21. Implementation Phases

## I0 — Discovery Foundation

- 建立 Discovery Model。
- 讀取 Project、Goal、Task 狀態。
- 支援事件與週期觸發。
- 保存 Discovery Record。

## I1 — Candidate Generation

- 產生 Candidate Action。
- 支援多方案。
- 建立候選行動與 Discovery 關聯。
- 避免重複候選工作。

## I2 — Deliberation & Decision

- 建立評估模型。
- 支援跨 Project 優先排序。
- 支援 ACT、DEFER、MONITOR、ASK_HUMAN、NO_ACTION、SLEEP。
- 保存 Decision Trace。

## I3 — Goal Persistence & Optimization

- 讀取 Goal Version。
- 支援 Goal Review。
- 支援 Goal Decomposition。
- 支援策略最佳化提案。

## I4 — Goal Revision & Approval

- 建立 Goal Evolution Record。
- 支援人類核准。
- 支援版本化變更。
- 支援 Goal Retirement。

## I5 — Autonomous Domain Experiments

- Investment Initiative。
- Coding Initiative。
- Writing Initiative。
- 跨 Project 資源分配。
- 長期自主評估。

# 22. Acceptance Criteria

AIA-09 的驗收應以可觀察行為為準，而不是只確認程式中存在某個類別或函式。

## 22.1 Initiative

- [ ] 能在沒有使用者新指令時發現值得處理的事項。
- [ ] 能從 Project、Goal、Task、Memory 與環境變化發現問題。
- [ ] 能產生多個候選行動。
- [ ] 能說明候選行動與目標的關係。
- [ ] 能辨識重複 Discovery。
- [ ] 能保留尚未處理的候選工作。

## 22.2 Deliberation

- [ ] 能評估價值、成本、風險與依賴。
- [ ] 能比較不同候選方案。
- [ ] 能處理跨 Project 資源競爭。
- [ ] 能辨識資訊不足的情況。
- [ ] 能合理選擇不行動。

## 22.3 Decision

- [ ] 支援六種決策類型。
- [ ] 每次重要決策都有理由。
- [ ] DEFER 有重新評估條件。
- [ ] MONITOR 有觀察條件。
- [ ] NO_ACTION 有證據與理由。
- [ ] SLEEP 有喚醒條件。

## 22.4 Goal Evolution

- [ ] 能維持 Goal Persistence。
- [ ] 能提出 Goal Optimization。
- [ ] 能拆解與合併子目標。
- [ ] 能提出 Goal Revision。
- [ ] 重大變更必須取得人類核准。
- [ ] 每次變更都保留歷史版本。
- [ ] 不允許 Agent 自行變更治理限制。

## 22.5 Continuity

- [ ] 新 Session 能恢復既有 Goal。
- [ ] 能讀取歷史 Decision。
- [ ] 能辨識尚未驗證的決策。
- [ ] 能根據歷史狀態產生新的 Initiative。
- [ ] 不會因為 Session 結束而失去長期目標。

# 23. 風險與控制

| 風險 | 說明 | 控制方式 |
|---|---|---|
| Initiative Flood | 不斷產生候選工作 | 去重、成本限制、優先排序 |
| False Discovery | 將推測當成問題 | Evidence、Confidence |
| Goal Drift | 逐漸偏離原始目標 | Goal Version、Approval |
| Over-Optimization | 過度追求短期指標 | 長期目標評估 |
| Endless Deliberation | 不斷分析卻不決策 | 評估期限與停止條件 |
| Unnecessary Action | 為了活動而活動 | NO_ACTION、SLEEP |
| Resource Exhaustion | 過度使用資源 | Budget Control |
| Unauthorized Change | 擅自改變目標或權限 | Policy、Approval |
| Context Loss | 無法延續過去工作 | Continuity、Context Reconstruction |

# 24. 最終設計原則

AIA-09 的核心不是讓 Agent 不斷產生工作，而是讓 Agent 能夠自主判斷什麼值得做。

一個成熟的 AIA 應能：

1. 沒有人提醒時，仍能注意到重要變化。
2. 發現問題後，主動尋找可行方案。
3. 在多個工作之間合理分配資源。
4. 知道什麼時候應該執行。
5. 知道什麼時候應該等待。
6. 知道什麼時候應該詢問人類。
7. 知道什麼時候不應該做任何事。
8. 根據長期成果檢視原有目標與策略。
9. 保留每次判斷與目標變更的歷史。
10. 在不破壞治理邊界的前提下持續改善。

**AIA 不應只是等待命令的工具，也不應成為不受控制的自動化系統。**

它應是一個能夠維持長期目標、主動發現問題、審慎選擇行動、持續檢討成果，並在適當邊界內演化自身工作方式的自主系統。

主動性讓 AIA 知道下一步可能做什麼。

目標演化讓 AIA 知道為什麼要做，以及原本的方法是否仍然值得繼續。

兩者結合，才使時間連續性真正轉化為持續的自主行為。

---

# 25. 目標演化三態分流與「防移動球門」硬性治理約束 (v2.1 Normative Addendum)

當長期運作結果與目標指標產生落差時，AIA 必須明確區分以下三種演化模式，嚴禁混淆「方法失敗」與「目標錯誤」：

| 演化模式 | 核心問題 | 適用情境 | 系統允許動作 | 授權要求 |
|---|---|---|---|---|
| **Goal Persistence** | 原本目標是否仍值得繼續？ | 策略短期遭遇逆風但基本前提仍成立；或目標合理但算力/預算/資料資源暫時不足 | 維持 `ACTIVE`，或因資源瓶頸轉為 `PAUSED` 並記錄約束，**嚴禁修改目標指標** | `C0~C1`（暫停須留存證據與通知） |
| **Goal Optimization** | 有沒有更好的達成方式？ | 頂層目標仍合理，但現行執行策略、閾值或子任務拆解效率不佳 | **鎖定頂層 `STRATEGIC` Goal 的名稱與 `TargetMetricJSON` 不變**，僅調整下層 `Strategy` 或重組 `OPERATIONAL` 子目標 | `C1~C2` |
| **Goal Revision** | 原本的目標本身是否已失效？ | 支撐原目標的外部前提主張（Premise Claims）已被新證據推翻 | 提出含前後指標 Diff 的 `GoalEvolutionProposal`，經核准後寫入 `goal_revisions` | **`STRATEGIC` 強制人類 `Approval` (`C3`)** |

## 25.1 防移動球門硬規則 (Anti-Goalpost-Moving Guard)

1. **禁止因虧損或未達標自行降級目標**：例如 Investment Agent 目標為「最大化總資產」，絕對不能因為短期投資虧損，就自行將目標改寫為「降低波動」以美化帳面績效。
2. **單純績效落後退件規則**：若 `ActorType == AGENT` 對 `STRATEGIC` Goal 提出 `REVISION`，但僅附帶指標落後（Metric Shortfall）而未附上至少一筆經知識庫確認已失效/衝突之前提主張（`invalidated_claim_ids`），Runtime 必須直接退回（`REJECTED_GOALPOST_MOVING`），強制導向 `PERSISTENCE` 或 `OPTIMIZATION`。
3. **零直接寫入權**：Agent 對 `STRATEGIC` Goal 僅具備提案權（`ProposeEvolution`），唯有通過人類顯式 `Approval` 後，系統方可推進 `goal_revisions` 版本號並更新目標定義。
