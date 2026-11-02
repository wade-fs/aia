# AIA-20: Book Manifest 與小說專案規格書 (Book Manifest & Novel Project Specification)

> **文件狀態**：正式規格 (Normative Specification)  
> **版本**：v1.0  
> **建立日期**：2026-11-02  
> **參考文獻**：`thinking.md` §五（Project Manifest）、`novel2movie` 全系列實作（唐、宋、元、明、清、修仙）

---

## 1. 摘要與背景 (Abstract & Background)

在 AIA (Artificial Intelligence Agent) 自主架構中，Writing Domain Agent 不僅負責單一章節的文字生成，更承擔長篇歷史群像小說、跨卷巨著與體系化世界觀的生命週期管理。

早期雛型中，`book.yaml` 僅具備初階的大綱（outline）與人物（characters）路徑設定，無法如實反映現有作品在長篇創作中所沉澱之複雜架構，例如：
1. **全系列與分冊規模**：十六卷 1440 章、九卷 810 章、八卷 720 章、六卷 540 章等嚴謹架構。
2. **多維度結構文件**：包含章節總帳 (`chapter_ledger.md`)、年代總表與史實校核 (`timeline_audit.md`)、敘事驗收與一致性管理 (`narrative_validation.md`)、史料研究與驗收流程 (`research_qa.md`)、寫作總指南 (`master_bible.md`)、人物生命週期 (`character_lifecycle.md`)、制度與日常生活 (`institutions_and_daily_life.md`)、軍政地理與防線 (`military_reference.md`) 等。
3. **雙世界交錯與外星觀測視角**：如唐末、元末作品中「人類歷史正文」與「外星文明觀測實驗室」獨立並行、不干預歷史之雙軌機制。
4. **體系化世界觀規劃**：如修真作品中「以失換得」的存在本質轉換、求理與應對分界、非數值化階梯與歷劫規則。
5. **嚴格的史學與寫作規範**：【史實骨架】、【合理推演】、【小說虛構】三級標記、非戰場中心原則、防止時代錯置 (Anti-Anachronism)、古代行程時間檢驗 (Anti-Teleportation)、主編裁決約束等。
6. **專案治理與連續性 (Governance & Continuity)**：符合 `thinking.md` §五定義之儲存位置、Git 版控、變更授權與跨 Session 記憶機制。

本規格書以現有小說實作（`tang`, `song`, `yuan`, `ming`, `ching`, `xiu-xian`）為依據，全面補齊並標準化 Manifest 規格。

---

## 2. Manifest Schema 規範 (Normative Schema)

`book.yaml` 置於專案工作區根目錄，採 YAML 格式。

```yaml
# ==========================================
# 專案基本元數據 (Project Metadata)
# ==========================================
project:
  id: string                   # [必填] 專案識別碼 (如: tang, song, yuan, ming, ching, xiu-xian)
  name: string                 # [必填] 專案名稱 / 書名 (如: 《長安秋暮》, 《丹心千秋》)
  subtitle: string             # [選填] 副標題 (如: 《故國潮聲》, 上冊｜舊國新生，下冊｜分途之後)
  series: string               # [選填] 所屬系列 (如: 千載風雲)
  type: string                 # [必填] 專案類型: historical_fiction | dual_world_historical | xianxia_worldbuilding
  domain_agent: string         # [選填] 負責領域代理人 (預設: writing)
  language: string             # [必填] 語言代碼 (如: zh-TW)
  era: string                  # [選填] 歷史時代區間 (如: 唐末五代 875–979 年)
  status: string               # [選填] 進度狀態: planning | in_progress | completed
  tags: [string]               # [選填] 標籤關鍵字

# ==========================================
# 規模與卷期劃分 (Scale & Volume Layout)
# ==========================================
scale:
  total_volumes: int           # 總卷數 (例如: 16, 9, 8, 6)
  total_chapters: int          # 總章數 (例如: 1440, 810, 720, 540)
  chapters_per_volume: int     # 每卷標準章數 (例如: 90)
  volume_structure: string     # 卷結構形式: files (volumeN.md) | directories (volumeN/)

# ==========================================
# 檔案與結構對應 (Structure Mapping)
# ==========================================
structure:
  outline: string              # [必填] 全書總綱 (如: outline.md 或 outline_master.md)
  volumes_plan: string         # 分卷總綱 / 卷次規劃 (如: volumes.md, volume_plan.md)
  volumes: string              # 分卷存放路徑或模式 (如: . 或 volume1..16)
  chapters: string             # 章節正文資料夾 (如: chapters/)
  
  # 年代與史實管理
  timeline: string             # 編年史實 / 年表 (如: timeline.md, chronology.md)
  timeline_audit: string       # 年代總表與史實校核 (如: timeline_audit.md)
  
  # 章節管理與驗收流程
  chapter_ledger: string       # 章節總帳 (如: chapter_ledger.md)
  narrative_validation: string # 情節驗收與敘事一致性規範 (如: narrative_validation.md)
  research_qa: string          # 史料研究與驗收流程 (如: research_qa.md, research_workflow.md)
  
  # 人物追蹤系統
  characters: string           # 人物清單 / 分級表 (如: characters.md)
  character_network: string    # 人物關係網絡 (如: character_network.md, relationships.md)
  character_lifecycle: string  # 人物生命週期與行蹤追蹤 (如: character_lifecycle.md)
  
  # 地理、軍事與制度環境
  geography: string            # 地理環境與考察 (如: historical_geography.md, geography.md)
  social_systems: string       # 社會環境、制度與日常民生 (如: social_systems_and_daily_life.md, social_life.md)
  military_reference: string   # 軍事戰略與防線考證 (如: military_reference.md)
  
  # 考證書目與總綱規範
  bibliography: string         # 史料出處書目 (如: source_bibliography.md, sources_and_fact_policy.md)
  evidence_review: string      # 史實考證與裁決修訂 (如: historical_evidence_review.md)
  master_bible: string         # 寫作總指南與防雷規範 (如: master_bible.md)
  rules: string                # 章節撰寫 Prompt 規則 (如: chapter_writer_rules.md)
  guidelines: string           # 寫作指南 (如: laboratory_writing_guidelines.md)
  scripts: [string]            # 輔助工具或生成腳本 (如: generate_novel.py, novel_to_video.py)

# ==========================================
# 雙世界 / 外星觀測設定 (Dual World / Laboratory)
# ==========================================
laboratory:
  enabled: bool                # 是否啟用雙世界交錯設定
  perspective: string          # 觀測視角說明 (如: 外星文明人類觀測實驗室)
  specification: string        # 實驗室設定檔 (如: laboratory.md)
  guidelines: string           # 實驗室寫作指南 (如: laboratory_writing_guidelines.md)
  volume_map: string           # 實驗室各卷對應圖譜 (如: laboratory_volume_map.md)
  integration: string          # 雙世界整合規範 (如: volumes_lab_integration.md)
  non_interference: bool       # 不干預原則 (預設: true)

# ==========================================
# 玄幻/修真世界觀體系 (Worldview System)
# ==========================================
worldview:
  system_type: string          # 體系類型 (如: loss_exchange_tribulation 以失換得)
  entry_stages: [string]       # 入門門檻 (如: [煉骨, 煉氣])
  formal_realms: [string]      # 正式境界 (如: [知身, 離欲, 無我, 斷緣, 超時, 入道])
  cultivation_principles:      # 核心哲學原則清單
    - "強大是缺了些什麼的結果（以存在換存在）"
    - "應對問怎麼活下去，求理問為什麼會這樣"
    - "劫會長成問題的形狀"
    - "只有實踐的才是真理，沒實踐的都叫空談"

# ==========================================
# 寫作政策與約束 (Writing Policy)
# ==========================================
writing_policy:
  historical_accuracy: string  # 史實精確度要求: high | strict | relaxed
  battle_tactics_focus: bool   # 是否以戰場戰術為中心 (歷史小說專案規範一律為 false)
  narrative_dimensions:        # 敘事維度
    - "朝廷與制度線"
    - "戰爭與邊疆線"
    - "地方與生活線"
    - "人物與記憶線"
  classifications:             # 內容分類層級
    - "historical_fact"        # 【史實骨架】：無可爭辯之正史記載
    - "reasonable_inference"   # 【合理推演】：符合制度、民俗與行程之推論
    - "fictional_content"      # 【小說虛構】：藝術創作、基層原創人物、方外穿線
  preserve_historical_figures: bool # 不得竄改歷史人物生死或重大抉擇 (true)
  require_consistency_check: bool   # 強制年代/年號/地理一致性檢驗 (true)
  anti_anachronism: bool       # 時代錯置檢驗 (true)
  anti_teleportation: bool     # 地點衝突與合理行程天數檢驗 (true)
  information_boundary: bool   # 人物決策不得逾越當時當地情報邊界 (true)

# ==========================================
# 權威裁決與考證來源 (Authority & Adjudications)
# ==========================================
authority:
  confirmed_settings: string   # 設定確認權限 (如: user_approved, author_adjudication)
  historical_facts: string     # 史實採納依據 (如: multi_source_verified 多源互證, evidence_required)
  unresolved_questions: string # 存疑議題處置 (如: never_assume_verified, flag_as_inference)
  adjudications: [string]      # 主編最高裁決清單 (如: 29項主編裁決)

# ==========================================
# 專案儲存、版控與治理 (Storage, VCS & Governance)
# 依據 thinking.md §五 核心架構
# ==========================================
storage:
  type: filesystem             # 儲存類型: filesystem
  root: string                 # 專案絕對或工作區相對路徑

version_control:
  type: git                    # 版控類型: git
  enabled: bool                # 是否啟用 Git
  default_branch: string       # 預設分支 (main)

governance:
  direct_main_write: bool      # 禁止直接寫入 main 分支 (預設: false)
  require_diff_review: bool    # 需透過 Diff 檢視審批 (預設: true)
  require_human_approval_for_major_changes: bool # 重大修改需人工審批 (預設: true)

continuity:
  enabled: bool                # 啟用跨 Session 進度銜接

memory:
  enabled: bool                # 啟用專案脈絡記憶保存
```

---

## 3. 現有小說專案映射對照表

| 小說專案 ID | 書名 / 副標題 | 時代 / 範疇 | 規模 | 核心特徵與結構亮點 |
| :--- | :--- | :--- | :--- | :--- |
| **`tang`** | 《唐末》<br>（上冊《長安秋暮》、下冊《中原風雲》） | 875–979 年 | 16 卷 | 雙世界外星實驗室觀測視角、三地理主軸（中原、河東、南方）、人物生命週期與行蹤追蹤。 |
| **`song`** | 《宋末》<br>（《丹心千秋》） | 1100–1283 年 | 8 卷 720 章 | 四大敘事維度、四川山城與襄樊長江防線考證、跨年改元防雷紀年 (`master_bible.md`)、多源互證。 |
| **`yuan`** | 《元末明初歷史群像小說》<br>（《志在山河》/《山河易旌》） | 1328–1368 年 | 6 卷 540 章 | 雙世界實驗室觀測、民俗信仰與方外宿命、非戰場中心（糧價、徵發、文書、難民流動）。 |
| **`ming`** | 《明末清初歷史小說》<br>（《海上悲歌》/《故國潮聲》） | 1624–1722 年 | 9 卷 810 章 | 落實 29 項主編裁決修訂與卷題年代對齊、海疆與移民生活、史料研究與驗收工作流程。 |
| **`ching`** | 《天下易主：故國潮聲》<br>（《復我中華》） | 1840–2020 年 | 16 卷 1440 章 | 上下兩冊（《舊國新生》與《分途之後》）、16個分卷目錄、全系列章節總帳、情節驗收與敘事一致性。 |
| **`xiu-xian`** | 《求理與歷劫》<br>（修真小說世界觀規劃書） | 體系世界觀 | 6 大境界 | 「以失換得」存在轉換原則、應對與求理分界、劫會長成問題形狀、實踐是真理必要條件。 |

---

## 4. AIA 執行階段行為保證 (Runtime Guarantees)

1. **零損壞向下相容 (Backward Compatibility)**：
   舊版僅含有 `outline` 與 `characters` 的最小化 Manifest 可繼續被正常載入與解析，缺失欄位安全套用預設值。
2. **自動路徑解析 (Path Resolution)**：
   Writing Observer 與 CLI 解析 Manifest 時，所有相對路徑皆以專案根目錄 (`Workspace`) 為基準錨定。
3. **一致性稽核整合 (Autonomy Consistency Audit)**：
   `ManuscriptConsistencyObserver` 自動讀取 `writing_policy` 中的分類與防錯開關，對照正史骨架 (`timeline_audit.md`) 執行卒後登場、時代錯置與行蹤瞬移驗證。
