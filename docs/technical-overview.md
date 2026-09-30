# 技術總覽

這份文件整理 remake 目前的技術現況：完成度怎麼量、和原版怎麼對拍、各個子系統照原版的哪一段做。
玩家導向的介紹與操作說明在 [README](../README.md)。逐項的行為契約在
[`docs/spec/`](spec/000-index.md)，唯一的目前狀態與定案在 [CONTEXT.md](../CONTEXT.md)，
未完成工作的主台帳是 [GitHub issues](https://github.com/wicanr2/Pool-of-Radiance-cht/issues)。

## 完成度

以「完整繁中、正常主線可破關、三平台可發行」為分母，目前保守估計 **89～93%**。

| 分母 | 狀態 |
|---|---|
| 完整繁中 | 遊戲內文字 1,731 句全部翻完；介面逐張實機截圖看過，字型缺字 0 |
| 正常主線可破關 | 腳本層沒有走不過去的指令；從標題到結局已經只用鍵盤連續跑通 |
| 三平台可發行 | 發行包可重生；Linux 與 Wine 實測啟動；Windows／macOS 真機驗收未做 |

壓住上限的是 Windows 與 macOS 的真機啟動還沒回填（[#6](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/6)）。
估算依玩家垂直鏈與交付門檻，不以規格文件數量換算。

主線通關收據的口徑：用作弊選單（鎖 HP、一擊斃命）從標題以正常按鍵跑到結局，原版那一側
由 dosgolem 同樣作弊跑一次，兩邊逐段對照必經的腳本區塊與旗標。收據要數得出敵方挑人、
走位、出手、命中，隊伍照樣擲命中（`TestMainlineProbeCheatMenuToEnding` 的機制閘門）。
不開作弊、以原版規則強度通關是可選的量測。

影響玩法的缺口目前是空的。原版呈現細節、證據停在 unknown 的位元組與只缺收據的驗證，
依 [AGENTS.md](../AGENTS.md) §11〈收斂標準〉記在 [停止線清單](audit/stop-line.md)。

## 對原版的抽樣對拍

比的是**發行的那個 AppImage** 在真視窗裡的截圖，不是原始碼建置，也不是單元測試裡的
合成圖。原版那一側由 [dosgolem](https://github.com/wicanr2/dosgolem) 跑真的 `START.EXE`
產生，直接吐 320×200 的色號陣列；流程是 `tools/package-release.sh <版本>` 之後跑
`tools/appimage-dos-parity.sh <版本>`，對拍腳本會拒絕比原始碼舊的發行包。

目前抽樣 36 張，全部量到，與基準表 0 欄不符。代表性的兩張：

- **標題**：整張 63,866／64,000（99.79%）。差的只有 remake 自己加的按鍵提示那一行
  （一個 47×7 的方框），其餘逐格相同：同一份 `TITLE.DAX`、同樣的位置與縮放。
- **第一人稱框**：88×88 逐格 100%（7,744／7,744），由目前座標與原版 GEO 牆面資料
  現場合成（[spec 126](spec/126-first-person-inset-pixel-parity.md)）。

戰鬥畫面的外框 9,152／9,152 相同。逐項比例、狀態標籤（same-state／layout-only）與方法
在[對拍報告](audit/dos-parity-sample.md)。

## 文字與中文化

- 原版 ECL 文字盤點共 **1,731 句、110,473 個字元**
  （[`dos-ecl-text-inventory.json`](audit/dos-ecl-text-inventory.json)），覆蓋率 100%：
  八個 ECL 封存檔的敘述、對話、選單選項與怪物名。譯文以原文整句為鍵放在
  `internal/gametext`，原版檔案一個位元組都不改。
- 覆蓋率由 `cmd/pool-text-inventory -coverage` 量出來；譯文表出現盤點檔沒有的原文時
  失敗即關閉，另有測試逐條核對每個原文都真的存在。
- 用詞以軟體世界代理的官方說明書與探險者手冊為準，說明書沒收的專名逐條記在
  [詞彙表](reference/manual/glossary.md)。
- 字型是倚天 16×15 點陣字，屬第三方資產、不進 repo。`-lang zh` 沒有字型會失敗即關閉。
  點陣字不靠膨脹字身加粗，厚度用同色系暗一階的外圈（AGENTS.md §8）。
- `F7` 在執行中切換繁中與 English：交換字型、介面字串、遊戲文字表、怪物譯名表與攻略表，
  規則狀態不動。

## 探險者手冊

說明書上冊的線索報導 58 條、酒店傳言 23 條與議會公告 18 則，共 99 條，由
`cmd/pool-journal-corpus` 從轉錄稿切成 `internal/journal`。遊戲文字報了編號
（「成為線索報導 46」），按繼續就直接翻到那一則；一段話報好幾則時一則一則翻
（[spec 132](spec/132-journal-citations-open-in-place.md)）。下冊附錄的七節規則表也收在裡面：
金錢換算、法術表、裝備一覽、昇級所需經驗、牧師對抗不死怪物、各等級的裝備與武器、武器一覽。

## 腳本層（ECL）

- 原版 ECL 的 61 條 opcode 在八個封存檔的 29 個 block 裡**沒有一條走不過去**。盤點由
  `cmd/pool-ecl-frontier` 重生，「已處理」那一半直接讀共用 VM 的 switch 與 Pool 的
  passthrough 清單。
- payload 映射基準由原版 loader／resolver 閉合為 `9900h`；`9914h` 是五個 command-set
  headers 之後的第一條指令位址。
- 正式玩家路徑用同一個 VM session 跨 ECL archive 執行需要的 blocks；羅夫導覽之後沿用
  同一份 memory、旗標與 continuation 狀態。
- 原版 20 處要玩家打字的密語（守衛的口令、亡魂的暗語、矮人符文），remake 預設把答案接在
  問句後面；作弊選單的 `P` 關掉之後問句與原版逐字相同，關掉不算作弊
  （[spec 141](spec/141-cheat-menu.md)、[spec 087](spec/087-ecl-input-opcodes.md)）。

## 建角、隊伍與存檔

- 建角照原版順序：種族 → 性別 → 職業 → 陣營 → 資料頁 → 姓名 → HEAD／BODY 肖像 →
  READY／ACTION 戰鬥圖示（[spec 003](spec/003-dos-character-creation-flow.md)）。
  六種族的原版職業清單在 typed catalog。
- 肖像讀 `HEAD3.DAX`／`BODY3.DAX`，selector 範圍 1..14／1..12
  （[spec 006](spec/006-dos-portrait-archives.md)）。原版玩家建角一律從 1／1 開始；
  remake 的預設依性別與職業取原版自己的分組表第一格，這一點是刻意的偏離，位元組與理由
  記在 spec 006〈預設肖像〉。
- 人物管理選單在訓練所以外沒有 `T)RAIN`：原版 `T` 的啟用旗標 `DS:06D4h` 只在訓練所那一區
  設起來（[spec 008](spec/008-character-library-and-party-menu.md)）。
- DOS 285 位元組的 CHA／SPC 記錄由 `dos_export.go` 匯出，豁免欄照 overlay-23 重算，
  七名預設人物逐格相同。對話、服務與戰鬥中途不能存檔，與原版一致。
- 自訂規則「委任經驗值加倍」預設關，關著時所有對拍與收據都是原版規則
  （[spec 140](spec/140-house-rule-commission-experience.md)）。

## 戰鬥與規則

- **回合**：擺陣、先攻、移動預算、命中與傷害照原版規則
  （spec 057／059／061／062）；部署照原版的樣板填寫與逐人掃描，敵方站在「朝向前方走得到
  的格數」那一格（spec 078）。
- **裝備**：物品型別表的來源是 ZIP 的 `poolrad/items`（`DS:54E0h` 那段是未初始化的），
  索引由墓園那把 `Two-Handed Sword +1` 正對照釘住。武器決定 THAC0 與傷害骰
  （[spec 065](spec/065-weapon-driven-combat-stats.md)），盔甲與負重決定 AC 與腳程（spec 079／080）。
- **法術**：`START.EXE` 有一張 56 筆的法術名稱表（位移 41052，步長 41），分組界線與說明書
  下冊第六章的 LEVEL 標題逐條相符（[spec 068](spec/068-spell-name-table.md)）。記憶陣列在
  角色記錄 `+1Fh`（spec 070／072），67 格效果全部施得出來（spec 073／074／098），戰鬥與
  營地走同一支掛效果的常式（spec 112）。會不會一條法術由記錄 `+32h + 編號` 的法術書決定
  （[spec 110](spec/110-spellbook.md)）。
- **敵方 AI**：先問武器搆得到誰，搆得到就打，搆不到才依戰術模式那一列的五個相對方向依序
  試走；追擊目標跨回合黏著，否決查詢的四個代碼照原版（[spec 096](spec/096-monster-ai-structure.md)、
  spec 059／112）。脫離接敵會引來反應攻擊，照原版扣反應者這一回合的次數（spec 160）。
- **戰鬥畫面**（[spec 129](spec/129-combat-screen-layout-and-command-bar.md)）：左邊 7×7 格
  的戰場、右邊三行資訊、框外一列指令。那一列逐段判斷要不要接上去：身上沒東西沒有「使用」、
  沒記法術沒有「施法」、不是牧師沒有「轉變」。盤面牆呈斜線是投影本身的形狀
  （`X = 21 + 6dx + 5dy + subB`），視窗一進畫面就捲到行動者（overlay-32 `07D4h`）。
- **自動戰鬥**：`Q` 交給 AI，SPACE 收回（[spec 139](spec/139-quick-auto-combat.md)）。
- **戰場造形**：隊員用 `CBODY.DAX` 的三十二種身體配自己的六組配色；怪物用 ECL
  `LOAD MONSTER` 第三個運算元指到的 `CPIC<區號>.DAX` 區塊，八個檔共 218 張
  （[spec 166](spec/166-monster-icons-and-combat-animation.md)）。`COMSPR.DAX` 是十三組戰鬥特效
  （箭、飛斧、石頭、閃光、爆炸），不是怪物。戰場地形圖塊來自三個 `*COM.DAX`，地圖存的是
  格位類別碼，經類別表的 `PresentationCode` 對到圖塊序號（[spec 131](spec/131-combat-terrain-tiles.md)）。

## 城鎮與冒險畫面

- **商店**：四家店與墓園戰利品走同一條 ECL 服務邊界，差別在三個旗標；存貨是 `ITEM3.DAX`
  的四個 block，價格在記錄的 `+3Ah`（[spec 067](spec/067-shop-service-and-stock.md)），貨品清單照
  原版版面（[spec 168](spec/168-shop-buy-list-layout.md)）。
- **指令列**：冒險畫面最下面六個指令來自 `DS:04CAh`
  （[spec 119](spec/119-adventure-command-bar.md)）。`C` 施法走 overlay-15 entry 2，
  與戰鬥中的施法是同一支派發（overlay-22 entry 5），清單共用同一個閘。
- **人物資料頁**：`V` 走 overlay-19 entry 5，十三行與每一欄的位置逐格對原版量過
  （[spec 130](spec/130-character-sheet-layout.md)）。
- **遊戲內攻略（F3）**：座標一律出自原始 GEO 的地形碼，名字取自原版腳本印出來的第一句話，
  不抄第三方攻略；預設只顯示走過的格子。做法見 [`docs/guide/README.md`](guide/README.md)。
- **素材總覽（F4）**：四頁都是遊戲畫面，圖形只從玩家自己的原版 ZIP 讀出來，repo 不含也
  不產生任何素材檔。

## 配樂

原版 DOS 版只有 PC 喇叭音效。remake 可以播 PC-98 版的 15 首 YM2203 配樂（照 `GAME.EXE`
的區塊對照表切換，[spec 169](spec/169-pc98-music-playback.md)），也可以改用 Amiga 版的 6 首。
配樂屬第三方著作權，不隨公開發行包散布；本機完整版的做法見
[`packaging/README-配樂.md`](../packaging/README-配樂.md)。`Ctrl+O` 開關音樂，
是 PC-98 原版的按鍵。

## 發行

`tools/package-release.sh <版本>` 產出 Linux AppImage、Windows ZIP 與 macOS 雙架構 ZIP
（osxcross 交叉編譯，不需要 Mac），並寫 `manifest.json` 固定雜湊
（[spec 066](spec/066-three-platform-release.md)）。公開的 patch 包不含原版資料、字型與配樂；
含這三者的 `full-local` 只留本機，不散布。

## 原版輸入與本機盤點

原版檔案不進 Git。DOS ZIP 共 168 個檔案（`START.EXE`、`GAME.OVR`、八組
ECL／GEO／WALLDEF／PIC／SPRIT DAX 與角色存檔樣本）；共用 engine 的 `dax` 解得開
113／113 個 DAX、合計 1,245 blocks。輸入雜湊與盤點收據見
[input-inventory.md](audit/input-inventory.md)。把 ZIP 放在 repository 根目錄後：

```sh
tools/go.sh run ./cmd/pool-inventory -zip "Pool of Radiance (1988).zip"
```

所有分析、建置與測試都在 Docker 裡跑，`tools/go.sh test ./...` 會在測試容器內自行建立
有界的 Xvfb。

## 文件導覽

| 要找什麼 | 看哪裡 |
|---|---|
| 某個功能的規格、實作與測試在哪 | [`docs/spec/000-index.md`](spec/000-index.md)（由 `cmd/pool-doc-index` 產生） |
| 目前狀態、定案與被推翻的斷言 | [CONTEXT.md](../CONTEXT.md) |
| 未完成工作 | [GitHub issues](https://github.com/wicanr2/Pool-of-Radiance-cht/issues)、[WORKLIST.md](../WORKLIST.md) |
| 不追的項目與理由 | [`docs/audit/stop-line.md`](audit/stop-line.md) |
| 截圖的來源提交、日期與雜湊 | [繁中截圖 manifest](audit/remake-chinese-screenshot-manifest.json) |
| 標題與主選單的原版收據 | [spec 001](spec/001-dos-title-picture.md)、[DOS 標題／主選單 oracle](playtest/dos-title-main-menu.md) |
