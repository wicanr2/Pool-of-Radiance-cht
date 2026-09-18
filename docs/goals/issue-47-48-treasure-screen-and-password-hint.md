# Goal：戰利品畫面補齊（#47）＋密語提示改成作弊選單的開關（#48）

主台帳：[issue #47](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/47)（戰利品畫面與原版差很多）、
[issue #48](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/48)（密語提示的開關）。
上一個 goal 的結案位置：[`issue-5-cheat-playthrough-both-sides.md`](issue-5-cheat-playthrough-both-sides.md)〈2026-09-17 收在哪〉
（#5 關閉：兩邊作弊通關、逐段對照）。

## 使用者的決定（2026-09-18）

原話：「remake 需要實作 share 喔，另外 remake 密語 or 答案直接放在對話後面」。問過之後定案：

- **Share**：規則已經有（`internal/treasure.ShareMoney`，對過 overlay-21 `062Eh..09E8h`），使用者要的是
  **戰利品畫面不好用**——看不到金額、分完之後看不出誰拿到什麼。
- **密語提示**：`cmd/pool-game/ecl_input.go` 已經把答案接在問句後面，而且是一律顯示。改成作弊選單的開關，
  **預設開、F6 可以關**；關掉不算作弊（不寫 `CheatsUsed`）。
- **範圍**：只有**打字的密語**（`10h INPUT STRING`）。選單類的正解（TELL THE TRUTH、喬裝、投票 ATTACK）這一輪不做。

## 現況

**戰利品（#47）**

| | remake（`cmd/pool-game/main.go` `enterTreasureMain`／`selectTreasureOption`）| 原版（dosgolem，狀態檔 `workplace/dosgolem-cheat/handin-slums-stuck.state`）|
|---|---|---|
| 頂層 | `View / Take / Pool / Share / Exit`，文字「The party has found treasure!」 | 底列 `VIEW TAKE POOL SHARE EXIT` |
| 看錢 | `View` 把幣種與物品名稱串成一行，底下只有 `Return` | `TAKE` 列出每一種幣與數量（`GOLD 250`／`PLATINUM 50`／`JEWELRY 1`），底列 `SELECT TYPE OF COIN EXIT` |
| `VIEW` | 同上（名稱清單）| **人物頁**：屬性、`GOLD 110`、`LEVEL`／`EXP`、`AC`／`THAC0`／`ENCUMBRANCE`、`STATUS OKAY`，底列 `VIEW:TRADE DROP EXIT` |
| 分錢 | `Share` 只在狀態列印「Pooled money shared.」 | 分完之後頂層少掉 `TAKE`／`SHARE`，剩 `VIEW POOL EXIT` |
| 畫面識別字 | **沒有**（`screenName()` 不認得戰利品），對拍與截圖腳本等不到 | —— |

規則那一側已經有的：`internal/treasure/money.go` 的 `PoolMoney`（overlay-21 `0506h..05D7h`）與 `ShareMoney`
（`062Eh..09E8h`：每種幣各自除以人數、裝不下的留在 pool）。相關規格：spec 032（`27h TREASURE` 八欄）、
spec 034（戰後戰利品選單邊界）。

**密語提示（#48）**

- `eclInputAnswer`（`cmd/pool-game/ecl_input.go`）靜態解答案：`03h COMPARE` 直接比字面（`ecl7/23 A4CAh` 的 `NOKNOK`），
  或先 `09h SAVE` 進字串變數再比（`ecl4/21 9E8Dh`／`9E9Ah` 的 `SAMOSUD`／`SHESTNI`）。同一個位址被寫過兩個字面就都列。
- 已知的打字密語：費蘭 `LUX`（`ecl4/21 AA7Eh`）、索寇巡邏 `SHESTNI`／`SAMOSUD`（`ecl4/21 9E80h`）、
  城堡外圈城門 `RHODIA`（`ecl5/3 AA56h`）、野外密碼門 `NOKNOK`（`ecl7/23 A4A1h`）。全集要自己掃，不要憑這張表。
- 作弊選單現況（spec 141）：`F6` 開，`L` 鎖 HP、`O` 一擊斃命、`W` 穿牆，三個都是預設關、打開寫 `CheatsUsed`。

## 提示詞（可直接貼給 `/goal`）

> 目標：戰利品畫面補齊到原版看得懂的程度（#47），密語提示改成作弊選單的開關、預設開（#48）。
> 兩項都要有從 `Update()` 送鍵的測試與原版對照，做完關 #47、#48。
>
> **A. 原版基準（先做，後面才有裁判）**
>
> 1. 用 dosgolem 從 `workplace/dosgolem-cheat/handin-slums-stuck.state`（市政廳交件的獎金，戰利品頂層選單）抓三張：
>    頂層、`TAKE` 的幣種清單、`VIEW` 的人物頁；再按 `SHARE` 抓分完之後的頂層。
>    每張存 `.idx` 與 PNG，記下鍵序與狀態檔 SHA-256，收據寫 `docs/audit/dos-treasure-screens.json`。
>    駕駛沿用 `tools/dosgolem_drive.py`（`shots -serve`）。
> 2. 量出分錢前後每個隊員的錢包：隊伍鏈從 `DS:5CF4h`、`+104h` 下一個（spec 040／052／084／088），
>    七種幣的位移查 spec 032／040；pool 的位址也要量。**五個人 250 金怎麼分**是這一輪的斷言，不要用推的。
>
> **B. 戰利品畫面（#47）**
>
> 3. `screenName()` 加識別字：`treasure`、`treasure-take`、`treasure-take-money`、`treasure-view`、`treasure-items`、
>    `treasure-confirm-exit`，讓 `-screen-state` 等得到（截圖腳本與對拍都靠它）。
> 4. 畫面補齊：
>    - 頂層與 `TAKE` 要看得到**每一種幣與數量**（照原版的列法）；
>    - `VIEW` 改成人物頁（沿用既有的人物資料頁繪製，不要另畫一套）；
>    - `Pool`／`Share` 之後把結果寫出來（誰拿到多少、pool 剩多少），不是只有一句狀態列；
>    - 錢分完之後頂層的選項跟著變（原版少掉 `TAKE`／`SHARE`）。
> 5. 測試（全部從 `Update()` 送鍵）：
>    - 交件拿到獎金 → `Share` → 每個隊員的錢包與 pool 餘數，對上 A.2 量到的數字；
>    - `Take` → `Money` → 拿某一種幣 → 錢包與 pool 的變化；
>    - `View` 出人物頁、`Exit` 有剩下的東西時會問。
> 6. 截圖：`tools/capture-*.sh` 那一類新增一支或沿用既有的，用**發行包**拍 remake 的四張，與 A.1 的原版並列放
>    `docs/screenshots/treasure/`，差在哪寫進 spec。
>
> **C. 密語提示的開關（#48）**
>
> 7. 作弊選單加「密語提示」：預設**開**、`F6` 進去可以關；**不寫 `CheatsUsed`**（預設就是開的，關掉更接近原版）。
>    存檔欄位與其他開關同一組、`omitempty`，舊存檔讀回來是開。
> 8. `enterECLInput` 依開關決定要不要接「（答案）」。關掉時問句與原版逐字相同。
> 9. 測試：預設帶答案、關掉沒有、存讀檔記得、`CheatsUsed` 不變；`eclInputAnswer` 對 `LUX`／`SHESTNI`／`RHODIA`／`NOKNOK`
>    四種寫法各一條（兩種比法都要有）。
> 10. 掃一次全部 ECL 的 `10h INPUT STRING`，把「問句 → 答案」列成表放 spec 087（或它指到的 audit JSON），
>     確認 `eclInputAnswer` 每一處都解得出來；解不出來的列出來並說明原因。
>
> **D. 文件與台帳**
>
> 11. spec 141 補「密語提示」一節（含為什麼不寫 `CheatsUsed`）；spec 034／032 補戰利品畫面的原版版面；
>     spec 087 補 INPUT STRING 的答案表；README（作弊選單那一段、戰利品）、CONTEXT、playtest 補十六。
> 12. 規格索引重生（`tools/go.sh run ./cmd/pool-doc-index`）。
> 13. 台帳：#47、#48 以證據關閉；`docs/worklist.json` 的兩條移除，重生 `WORKLIST.md`，再跑
>     `-mode verify` 與 `gh issue list --state open` 雙向核對。本 repo 可以 commit 與 push；dosgolem 若有改，推送前先問。
> 14. 收尾照 CLAUDE.md §7：動到玩家看得到的畫面就要**打包後對拍**（`tools/package-release.sh` → `tools/appimage-dos-parity.sh`），
>     數字（含不變的那幾項）寫進 commit message。目前對拍仍停在 field-cast 之前（#42），照上一輪的做法用同一支比對程式量拍到的那幾張。
>
> **E. 停止線**
>
> - 原版的戰利品版面有一項量不出來（例如 `VIEW` 的人物頁與既有人物頁差在哪），換三種做法仍沒有結論；
> - 要改 `internal/combat` 或共用 engine 才做得到；
> - 需要作弊選單以外的注入；
> - 單一步驟超過 30 分鐘。
>
>   停下時寫明卡在哪、用過哪些做法、下一步的假設。

## 已知風險與待決

- **`VIEW` 的人物頁**：原版在戰利品裡的人物頁與冒險畫面的人物資料頁可能是同一支（`VIEW:TRADE DROP EXIT` 那一列是
  戰利品專屬）。先量原版兩張再決定 remake 要不要共用同一份繪製，不要先寫死。
- **分錢的餘數**：`ShareMoney` 已經照 overlay-21 把裝不下的留在 pool；畫面要把「留在 pool 的」寫出來，否則玩家
  以為錢不見了。A.2 的量測要涵蓋「裝不下」那一種情形（負重上限，spec 032／040）。
- **密語全集**：`eclInputAnswer` 只往後掃 64 條指令。掃全部 `INPUT STRING` 時若有超過這個距離的寫法，要嘛放寬，
  要嘛列成例外，不能默默漏掉（沉默不等於沒有）。
- **#45**（結局頁數）與 **#46**（毒荊棘即死不經過 `2266h`）還開著，與這一輪無關，別順手做。

## 2026-09-18 收在哪

- **A（原版基準）**：`tools/dosgolem-treasure-screens.py` → `docs/audit/dos-treasure-screens.json`（六張畫面的色號陣列與
  SHA-256、鍵序、狀態檔雜湊，分錢前後逐人錢包與 pool）。量到的關鍵事實兩條：`VIEW` 是人物頁（spec 040 原本寫錯）、
  五個人分 250 金是每人 50 金 10 白金、珠寶 1 給最前面那一位、**除不盡的不留在 pool**。
- **B（#47 戰利品畫面）**：頂層選項依錢與物品的有無組成、`View` 走既有人物頁、`Take: Money` 列出每一種幣、
  `Pool`／`Share` 之後寫出誰拿到多少與 pool 餘額；`screenName()` 補八個識別字。
  測試 `cmd/pool-game/treasure_screen_test.go` 五條，全部從 `Update()` 送鍵，分錢那一條拿 A 量到的數字當裁判。
- **C（#48 密語提示）**：作弊選單 `P`，預設開，關掉問句與原版逐字相同，**不寫 `CheatsUsed`**；
  存檔欄位存「關掉」那一側，舊存檔讀回來是開。測試 `TestPasswordHintToggleDefaultsOnAndDoesNotCountAsCheating`。
  第 10 步的全集掃描做成 `cmd/pool-password-audit`（→ `docs/audit/dos-password-prompts.json`）：
  **20 處、19 處解得出答案**，剩下那一處原版就沒有答案。掃描過程修掉三個「自洽但錯」的坑（spec 087 有表），
  並順帶修掉 `eclInstruction` 用 engine 底稿指令表解碼（`34h` 差一個運算元）。
- **D（文件）**：spec 034、040、087、141；README、CONTEXT、playtest 補十六；規格索引重生。
- **對拍**：`v.1.1.12-20260918 patch zh` 連跑兩次，數字逐項相同；腳本仍停在 #42 那一點，`field-cast` 兩張未量，
  報表由手動跑 `tools/dos-parity-compare.py` 產生。與 `docs/audit/dos-parity-sample.md` 那張基準表有十項對不上
  （兩項「視野」100.00%→91.27%），**不是這一輪造成的、但以前沒有人比對過表**，開 #51 追。
