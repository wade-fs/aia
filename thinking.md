我的建議是：Phase 5 不應該成為第一次開始做 Writing Agent 和 Coding Agent 的階段。

但也不必在 Phase 5 前就把兩個 Agent 做到完整成熟。應該先建立它們的可運作版本，讓 Phase 5 負責長期深化與跨領域實驗。

## **一、重新看待三個 Agent 的定位**

這三個 Agent 其實提供三種不同的自主性驗證環境。

[The Best Trading Websites That Helped Me on with my ITPM Journey](https://images.openai.com/static-rsc-4/Vf_2uJF9NsfiaYZoydLQMMDhFXQ8HatV5z1PlBKtErjFsxgNEVKQuLxPEqLqipVkBwwbsIDpUEraqWRTtJzdtbczEg9egxFhDD6btm0lQjWlirQiLj_1VPdO82nV6AiPh4qNx4au5XjMWORJ32-LCJUAa1b1zhXgosKTYdl89lc?purpose=inline)

Investment Agent — 決策與結果

自主觀察市場、配置資產、判斷風險、修正投資策略。

驗證：決策品質、預測校準、長期成果

[Best Linux distro for developers of 2025 \| TechRadar](https://images.openai.com/static-rsc-4/dYsBYLYEWcbaMB8nPnuk3yUWddTNw7HOdNupafPVCW-spOQ8tvOqLjHs8UiCnl5FBGbSn5SnYMwXk8r36xUsU_ror3Sovk1W6hj8TcevLpOHm7L6cJRURT_qo2Dym3i0QsWG454L5FWHSmBDvAVob7ot-ZrH-hk0wrXaPIvrdPQ?purpose=inline)

Coding Agent — 問題與修正

自主理解程式架構、發現問題、除錯、修正、測試與追蹤技術債。

驗證：問題發現率、修正正確率、回歸率、技術改善

[人物專訪》陳耀昌用小說診斷歷史—浪漫書寫理性編織島嶼史 \- 自由評論網](https://images.openai.com/static-rsc-4/dPmod6cOcT3sLHH_7bHVymD0yJOVNHa28SEltoJu0sXmHdRTD6hGZmtBb9JSTRU7ZlGb88bAY3dMQ0QgJEJzQ9r1Ndn5pPyOcDL9HvHBRLcuz9aTY_5jFjP1_JHU316g0dECVGr3msmr3DzUxxVaieMWCWEqH7ieSVuUli4-B50?purpose=inline)

Writing Agent — 知識與一致性

自主管理長篇小說、人物年表、歷史資料、章節進度、跨卷關係與史實校勘。

驗證：矛盾偵測率、史實校勘、跨卷一致性、修正品質

這三者的價值在於它們的問題性質不同。

* 投資：未來不可完全預測，必須從結果修正判斷。  
* Coding：程式行為可以測試，能比較修正前後的結果。  
* Writing：長期知識與複雜關係必須維持一致，能使用既有素材建立測試基準。

如果只用投資測試 AIA，很可能只得到一個擅長投資決策的 Agent，而不是具備跨領域自主能力的 AIA。

## **二、我建議調整 Phase 3～5 的工作安排**

不是推翻七大 Phase，而是調整 Domain Agent 的進場時間。

| 階段 | Investment | Coding | Writing |
| ----- | ----- | ----- | ----- |
| Phase 0 | 盤點既有資產 | 盤點既有資產 | 盤點既有素材 |
| Phase 1 | 決策資料與市場證據 | 程式修改歷史與來源 | 人物、年表、史料來源 |
| Phase 2 | 投資風險治理 | 程式執行權限 | 文件修改權限 |
| Phase 3 | 自主投資 Pilot | 基礎 Coding Pilot | 基礎 Writing Pilot |
| Phase 4 | 投資決策 RCA | 除錯與修正驗證 | 歷史矛盾偵測與修正驗證 |
| Phase 5 | 長期虛擬投資實驗 | 長期自主 Coding | 長篇小說長期管理 |
| Phase 6 | 安全策略自我演進 | 安全程式碼演進 | 安全工具與工作流程演進 |

這樣 Phase 5 開始時，三個 Agent 都已經有實際運作經驗，而不是只有 Investment Agent 已經成熟。

## **三、Phase 5 開始前，兩個 Agent 應該完成到什麼程度？**

我會設定最低可用門檻，而不是要求功能全部完成。

### **Coding Agent**

Phase 5 前應具備

1. 能讀取並理解指定 Repository。  
2. 能建立專案結構、依賴與模組關係。  
3. 能發現程式錯誤、技術債與測試盲區。  
4. 能提出修改方案及預期效果。  
5. 能在隔離工作區修改程式。  
6. 能分析 Diff、執行驗證並追蹤結果。  
7. 能保存修改歷史與失敗經驗。  
8. 能在跨 Session 後恢復未完成工作。

不要求它此時就能修改 AIA 核心 Runtime。

尤其要區分：Coding Agent 可以協助開發 AIA，但修改 AIA 核心仍然必須遵守 Phase 6 的授權與安全演進機制。

### **Writing Agent**

Phase 5 前應具備

1. 能讀取長期專案大綱與章節資料。  
2. 能管理人物年表、歷史大事紀及人物關係。  
3. 能區分【史實骨架】【合理推演】【小說虛構】。  
4. 能辨識已知的時間、人物、事件矛盾。  
5. 能區分確定錯誤與尚待查證事項。  
6. 能提出修正方案，而不是直接覆蓋原稿。  
7. 能追蹤跨章、跨卷設定一致性。  
8. 能在跨 Session 後恢復小說進度及未完成校勘工作。

這裡有一個很好的優勢：你已經有唐末、宋、元系列的小說規劃與歷史素材，不需要從零建立 Writing Agent 的驗收資料。

## **四、三個 Agent 應該共用同一個 Autonomy Core**

這點不能因為提早建立三個 Agent 而改變。

```
                    AIA Runtime
                         |
                    Autonomy Core
                         |
       +-----------------+-----------------+
       |                 |                 |
 Investment Agent   Coding Agent     Writing Agent
       |                 |                 |
   台灣股市          Git Repository     歷史小說素材
   虛擬帳本          程式碼與測試       人物與年表
   投資決策          除錯與修正         史實校勘
       |                 |                 |
       +-----------------+-----------------+
                         |
                 Evidence / Memory
                 Learning / Evaluation
                 Experiment Management
```

三個 Agent 可以有不同的專業知識、工具與判斷方法，但：

* Continuity 共用。  
* Initiative 共用。  
* Deliberation 共用。  
* Self-Correction 共用。  
* Goal Evolution 共用。  
* Evidence 與 Learning 共用基礎治理。

否則最後可能變成三套獨立的 Agent 系統，而不是一個具有自主能力的 AIA。

## **五、我建議增加一個 Phase 5 Entry Gate**

在進入 Phase 5 前，應該驗收：

| 項目 | 必須達成 |
| ----- | ----- |
| Autonomy Core | 三個 Agent 都使用同一套核心自主循環 |
| Continuity | 三個 Agent 都能跨 Session 恢復 |
| Initiative | 至少能自行發現一項值得處理的工作 |
| Deliberation | 能比較行動與不行動 |
| Self-Correction | 至少完成一次可追溯的修正與驗證 |
| Evaluation | 三個 Domain 都有自己的 Baseline |
| Governance | 三個 Agent 都受到相同的治理架構約束 |

不要求三個 Agent 的成熟度相同，但不能有任何一個只是空殼。

## **六、我的結論：投資應該是第一個實驗，不應該是唯一的實驗**

我會保留目前 Investment Agent 在 Phase 3 提早 Pilot 的安排，因為它有很清楚的實驗條件：

* NT\$100,000 虛擬資金。  
* 規則版與 LLM 版世代比較。  
* 每個證券營業日觀察。  
* 每月 1 日檢驗資產。  
* 長期策略修正。

但同時：

* Coding Agent 應該成為 AIA 自我修正能力的主要驗證場景。  
* Writing Agent 應該成為 AIA 長期記憶、一致性與知識管理的主要驗證場景。

我甚至認為，Writing Agent 可能是最適合測試「記憶是否真正一致」的場景，而 Coding Agent 是最適合測試「修正是否真正有效」的場景。

這三個場景剛好互補。

因此，我建議在 Master Implementation Plan v2.0 中修正 Phase 3、Phase 4 與 Phase 5 的 Domain Integration Roadmap，增加三個 Agent 的最低可用門檻及 Phase 5 Entry Gate。

這不是增加三倍工作量，而是避免 AIA 花了很長時間，最後只證明了它能做自主投資，卻沒有證明它具備跨領域的自主學習能力。

