那以寫書為例，怎麼指定書的目錄給它？或是它會怎麼保管書？會使用 git 管理嗎？

程式碼呢？會使用 git 管理嗎？  
我建議兩者都使用 Git 管理，而且不應該讓 Agent 自己隨意決定資料放在哪裡。

但有一個重要區別：

* Writing Agent 管理的是一本書的長期知識、內容與創作進度。  
* Coding Agent 管理的是一個軟體專案的程式碼、修改與技術演進。

兩者都可以使用 Git 保存版本，但 Git 不是 AIA 的記憶系統。Git 負責保存檔案變化；AIA 負責記住為什麼改、目前進度、哪些決策已確認，以及下一步要做什麼。

我會設計成「Project Registration + Git Repository + AIA Continuity」三者整合。

# 一、Writing Agent：如何指定一本書？

假設你要讓 AIA 管理《長安秋暮》。

你不需要把所有內容貼給它，只需要在 AIA 註冊一個 Writing Project，指定書籍所在目錄。

例如：

aia project create \

  --type writing \

  --name "長安秋暮" \

  --path "/home/wade/books/長安秋暮"

AIA 註冊後，會讀取該目錄的專案說明與目錄結構，建立自己的 Project Context。

## 1. 書籍目錄應該由你掌握

我建議採用你原本就習慣的 Markdown 文件結構：

長安秋暮/

├── README.md

├── book.yaml

│

├── outline.md

├── timeline.md

├── timeline_audit.md

├── geography.md

├── social_system.md

│

├── characters/

│   ├── README.md

│   ├── primary.md

│   ├── secondary.md

│   └── relationships.md

│

├── volumes/

│   ├── volume1.md

│   ├── volume2.md

│   ├── volume3.md

│   └── ...

│

├── chapters/

│   ├── volume1/

│   │   ├── chapter001.md

│   │   ├── chapter002.md

│   │   └── ...

│   └── volume2/

│

├── research/

│   ├── sources.md

│   ├── historical_facts.md

│   ├── unresolved_questions.md

│   └── source_notes/

│

├── consistency/

│   ├── known_issues.md

│   ├── character_timeline_audit.md

│   └── cross_volume_audit.md

│

└── changelog.md

這是建議的範例，不代表每本書都必須有完全相同的目錄。AIA 可以依照書籍規模與使用者的習慣調整。

其中最重要的是 book.yaml。

它是 Writing Agent 的專案入口，告訴 Agent 這本書是什麼、如何管理、哪些文件具有權威性。

例如：

```
project:
  id: changan-autumn
  name: 長安秋暮
  type: historical_fiction
  language: zh-TW

structure:
  outline: outline.md
  timeline: timeline.md
  timeline_audit: timeline_audit.md
  volumes: volumes/
  chapters: chapters/
  characters: characters/
  research: research/
  consistency: consistency/

writing_policy:
  historical_accuracy: high
  battle_tactics_focus: false

  classifications:
    - historical_fact
    - reasonable_inference
    - fictional_content

  preserve_historical_figures: true
  require_consistency_check: true

authority:
  confirmed_settings: user_approved
  historical_facts: evidence_required
  unresolved_questions: never_assume_verified
```

這樣你只需要指定書籍目錄，Writing Agent 就能知道從哪裡開始理解這本書。

## 2. Agent 如何知道哪份文件才是準確版本？

這是很重要的問題。

例如：

* outline.md 定義全書架構。  
* volume1.md 定義第一卷章節安排。  
* characters/primary.md 定義主要人物。  
* timeline.md 定義人物與歷史事件的時間關係。  
* timeline_audit.md 保存校勘結果。

如果第一卷章節大綱與人物年表衝突，Agent 不應自行選擇其中一份並覆蓋另一份。

應該由 book.yaml 或其他專案規則定義文件的權威性與用途，並在衝突時建立待處理事項。

文件是書籍內容的來源，AIA 的資料庫則保存它對這些內容的理解與工作狀態。

## 3. Writing Agent 如何保管一本書？

我建議把書籍資料分成三種：

| 資料 | 保存位置 | 用途 |
| ----- | ----- | ----- |
| 書籍正式內容 | 書籍目錄 | 大綱、章節、人物、年表 |
| 研究資料 | 書籍目錄及來源庫 | 史料、考證、待查事項 |
| Agent 工作記憶 | AIA SQLite | 進度、決策、任務、校勘歷史、記憶關聯 |

例如 Writing Agent 今天發現：

「第一卷第 12 章的人物活動時間與人物年表衝突。」

它應該：

1. 保存問題及來源位置。  
2. 建立一筆一致性檢查紀錄。  
3. 找出相關人物與事件的歷史依據。  
4. 提出修正方案。  
5. 等待必要的授權。  
6. 修改章節或設定文件。  
7. 更新校勘結果，但保留原始問題與處理歷史。

下次 AIA 重啟時，不必重新閱讀整本小說才知道這件事尚未完成。

這就是 Writing Project 的 Continuity。

# 二、Writing Agent 要不要使用 Git？

要，而且我認為應該預設使用 Git。  
建議預設啟用

一本長篇小說可能持續寫好幾年，Git 非常適合保存：

* 大綱歷次變更。  
* 人物設定變更。  
* 歷史考證修正。  
* 章節重寫。  
* 跨卷設定調整。  
* Agent 自主修正前後的差異。

例如：

Git Repository: 長安秋暮

main

 │

 ├── commit: 初始大綱

 ├── commit: 人物年表建立

 ├── commit: 第一卷章節規劃

 │

 ├── branch: agent/timeline-correction

 │     ├── 修正人物活動年代

 │     └── 修正第一卷相關章節

 │

 └── merge: 人工審核後合併

我建議的規則是：

* main：正式版本。  
* agent/*：Agent 的修改分支。  
* research/*：大型史料整理或研究工作。  
* 重大修改必須提供 Diff 與修改理由。  
* 人物死亡、重要歷史年代、跨卷核心設定等重大變更，必須經過人工確認。  
* 不允許 Agent 直接刪除或改寫歷史提交紀錄。

不需要每修改一個字就建立 Commit。可以依照工作單位建立 Commit，例如完成一次人物年表校勘、完成一章修訂或完成一次全書一致性修正。

## 4. Git 與 AIA Memory 的分工

這個區分必須牢記：

| Git 保存 | AIA 保存 |
| ----- | ----- |
| 哪個檔案被修改 | 為什麼修改 |
| 修改前後的內容 | 當時的決策依據 |
| Commit 與 Branch | 任務與實驗歷史 |
| 檔案版本 | 對設定的理解 |
| Merge 結果 | 是否經過人工核准 |
| Diff | 修正是否有效 |

Git 不會自動知道一個人物設定為什麼改變，也不會知道這項修改是否與其他卷衝突。

這是 AIA 必須補上的能力。

# 三、Coding Agent：程式碼要不要使用 Git？

這個答案更明確：必須使用 Git，並且 Git 應該是 Coding Agent 的核心工作介面之一。

假設你指定 AIA 自己的程式碼 Repository：

aia project create \

  --type coding \

  --name "AIA Core" \

  --path "/home/wade/src/github/aia"

Coding Agent 註冊後，應該先理解：

1. Repository 目錄結構。  
2. Go Module 與套件依賴。  
3. 核心 Runtime、Worker、Autonomy Core 的責任。  
4. SQLite Schema 與 Migration。  
5. CLI、Web Console。  
6. 現有測試。  
7. Git Branch、Commit 與未提交變更。  
8. 專案文件及實作狀態。

然後建立自己的 Project Context。

## 1. Coding Agent 的工作流程

我建議採用這種模式：

flowchart TD

    A["Coding Agent 接收或發現問題"] --> B["讀取 Repository 與 Git 狀態"]

    B --> C["分析架構、程式碼與文件"]

    C --> D["建立問題與修正提案"]

    D --> E["建立隔離 Branch / Worktree"]

    E --> F["修改程式碼"]

    F --> G["檢查 Diff"]

    G --> H["執行適用的驗證"]

    H --> I{"驗證是否通過？"}

    I -->|否| J["RCA 與修正"]

    J --> F

    I -->|是| K["產生變更報告"]

    K --> L["依授權等待審核或合併"]

    L --> M["保存 Learning Record"]

Coding Agent 接收或發現問題讀取 Repository 與 Git 狀態分析架構、程式碼與文件建立問題與修正提案建立隔離 Branch / Worktree修改程式碼檢查 Diff執行適用的驗證驗證是否通過？RCA 與修正產生變更報告依授權等待審核或合併保存 Learning Record否是

例如：

main

 │

 ├── commit: Runtime v1

 ├── commit: Worker Recovery

 │

 ├── agent/fix-worker-recovery

 │     ├── 修正 Worker 恢復問題

 │     ├── 新增測試

 │     └── 驗證修正結果

 │

 └── merge: 人工核准

這裡使用 Git Worktree 會很有價值。

它可以讓 Agent 在獨立工作目錄中修改程式碼，不會直接干擾你正在使用的工作目錄。

## 2. Coding Agent 不應該直接管理所有 Git 權限

我會把 Git 操作分成不同權限：

| 操作 | 建議權限 |
| ----- | ----- |
| git status / log / diff | 自動允許 |
| 建立 Agent Branch | 自動允許 |
| 在隔離工作區修改 | 依 Scope 授權 |
| 建立 Commit | 依專案政策允許 |
| Merge 到 main | 依風險與授權決定 |
| Push 到遠端 | 預設需要授權 |
| Force Push / 改寫歷史 | 禁止 Agent 自行執行 |
| 修改 Git Hook 或治理限制 | 嚴格授權 |

尤其 AIA 自己的 Repository 必須更嚴格。

Coding Agent 可以發現 AIA 核心問題、提出修正、建立 Branch、修改程式碼與驗證，但不能因此取得修改自身治理機制的權限。

# 四、三個 Agent 共用一套 Project 管理機制

我認為這是 AIA 架構上應該採取的方向。

不要分別設計：

* Writing Project Manager  
* Coding Project Manager  
* Investment Project Manager

而是建立共用的 Project Registry，由不同 Domain Agent 定義自己的專業規則。  
AIA Project Registry  
統一註冊、生命週期、目錄、權限與狀態管理  
Writing  
Book Project  
Coding  
Repository Project  
Investment  
Portfolio Project  
Continuity  
長期進度與狀態  
Memory  
記憶與知識  
Git Integration  
版本控制  
Governance  
授權與稽核

三種 Project 的差異只在於專業領域：

| Project Type | 主要資產 | 版本管理 |
| ----- | ----- | ----- |
| Writing | 書籍、人物、年表、史料 | Git |
| Coding | 原始碼、測試、文件 | Git |
| Investment | 虛擬帳本、決策、行情 | AIA Event History |

Investment 不一定需要 Git，因為每日交易與資產變動應由資料庫的交易紀錄與事件歷史管理，而不是用 Git Commit 記錄。

# 五、我會再加一個重要設計：Project Manifest

我建議每個 Project 都有一份由 AIA 管理的 Manifest，記錄：

```
project:
  id: changan-autumn
  name: 長安秋暮
  type: writing
  domain_agent: writing

storage:
  type: filesystem
  root: /home/wade/books/長安秋暮

version_control:
  type: git
  enabled: true
  default_branch: main

continuity:
  enabled: true

memory:
  enabled: true

governance:
  direct_main_write: false
  require_diff_review: true
  require_human_approval_for_major_changes: true
```

這樣 AIA 重啟後，可以知道：

* 這個 Project 是什麼。  
* 檔案放在哪裡。  
* 由哪個 Agent 負責。  
* 是否使用 Git。  
* 上次工作進度。  
* 目前有哪些未完成事項。  
* 哪些修改尚未核准。

Manifest 本身也應有版本與變更紀錄，避免 Agent 自行改掉專案管理規則。

## 六、我認為還有一件事應該特別注意

Git 管理的是檔案版本，不是所有工作狀態。

例如你可能同時有：

* 《長安秋暮》第一卷正在校勘。  
* 《丹心千秋》第二卷正在整理人物年表。  
* AIA Core 正在修正 Worker Recovery。  
* 另一個 Coding Project 正在開發新功能。

AIA 不應該因為切換到另一個 Project，就忘記前面三個 Project 的進度。

這些狀態應該由 AIA 的 Project、Task、Event History、Memory 和 Continuity 管理。

另外，SQLite 資料庫、OAuth Token、密鑰、暫存檔及大型產出不應因為使用 Git 就全部加入 Repository。各 Project 必須有 .gitignore、檔案白名單及目錄存取範圍；Agent 也不應任意掃描使用者的其他目錄。

## 七、我的建議：這應該成為 AIA 的正式共用功能

我會把它定義成：

Project Lifecycle & Workspace Management

而不是 Writing Agent 或 Coding Agent 的附屬功能。

建議職責如下：

1. Project Registration：指定書籍或程式碼目錄。  
2. Project Discovery：分析既有資料結構。  
3. Manifest Management：保存專案設定。  
4. Workspace Management：管理工作目錄。  
5. Git Integration：管理 Branch、Worktree、Commit、Diff。  
6. Project Continuity：跨 Session 恢復進度。  
7. Project Memory：保存專案知識與決策。  
8. Project Governance：管理檔案範圍與修改授權。  
9. Project Archiving：長期保存與封存。

這會讓你未來只需要告訴 AIA：

「這是一本新書，目錄在這裡。」

或者：

「這是另一個 Go 專案，請接手維護。」

AIA 就能註冊、理解、管理並持續工作。

我認為這項共用能力應該在 Phase 3～4 就建立基本版本，Phase 5 再讓三個 Agent 大規模使用。 不必等到 Phase 5 才開始處理。

這也讓 AIA 從「我叫 Agent 做一件事」，逐漸變成「我把一個長期專案交給 AIA，它能持續管理」。
