# Spec 141：作弊選單——鎖 HP、一擊斃命（預設關）

狀態：READY（開關、存檔、標示、傷害路徑、說明頁分頁、實作與測試；這是 remake 自己的功能，沒有原版行為可對）。
日期：2026-09-17。主台帳：GitHub issue #43。

## 一句話

F6 開一個作弊選單，裡面兩個開關：**鎖 HP**（隊員的 HP 不會少）、**一擊斃命**（隊員命中
就讓目標 HP 歸零）。預設關；打開過一次，存檔就永遠記著「這場戰役開過作弊」，
畫面上也標出來。原版沒有這個功能，開過作弊的路線不算原版驗收，也不算 #5 的收據。

## 為什麼可以做

使用者 2026-09-17 決定要這個選單，要在 F1 說明頁列出它的按鍵，並在探針打諾里斯時打開。
CLAUDE.md §3／§6 規定原版忠實是驗收基準、主線驗收不得依賴 forced-win，
所以照 spec 140（自訂規則）的前例，而且標示更重：

- 兩個開關預設關。關著時，所有既有測試、對拍畫面與原版一致性不受影響。
- 開著時，冒險畫面與戰鬥畫面都標示目前開的是哪幾個。
- 打開過任何一個，就寫一個清不掉的 `CheatsUsed`。隊伍選單與冒險畫面會一直標「開過作弊」，
  讀檔後照樣標。

## 按鍵與選單

| 項 | 內容 |
|---|---|
| 開選單 | 冒險模式（含戰鬥畫面）按 `F6`；說明、攻略、商店等覆蓋層開著時不開 |
| 選單內 | `L` 切鎖 HP，`O` 切一擊斃命，`ESC` 或 `F6` 關閉；選單開著時其他按鍵一律不作用 |
| 畫面識別字 | `cheat-menu`（`-screen-state`） |
| 切換提示 | 狀態列印一句，例如「作弊：鎖 HP 開（原版沒有）」 |

不放進隊伍選單（spec 140 的 `H` 在那裡）：戰鬥中也要開得到，隊伍選單在戰鬥中進不去。

## 存檔

`poolsave.State` 新增：

| 欄位 | JSON | 意思 |
|---|---|---|
| `Cheats.LockHP` | `cheats.lock_hp` | 鎖 HP |
| `Cheats.OneHitKill` | `cheats.one_hit_kill` | 一擊斃命 |
| `CheatsUsed` | `cheats_used` | 打開過任何一個就是真，不會回到假 |

都有 `omitempty`，舊存檔讀得回來（全部是假）。存檔 schema 版本不變：這幾個欄位缺席時就是
舊行為，與 `HouseRules` 相同。

## 鎖 HP

每一次 `Update()` 結束時（不論從哪一個分支返回）寫回：

- 戰術盤上每一位隊員：HP 補到上限；倒地、昏迷、瀕死、死亡（狀態 4／5／6）拉回 0；
  瀕死計數歸零；死亡時被清掉的佔格類別從 `Footprint` 拿回來。
- 戰鬥外：`state.Party` 每一位的 `CurrentHP` 補到 `MaxHP`，狀態 4／5／6 拉回 0。

寫回次數記在 app 上（戰鬥中、戰鬥外、從死亡拉回來各一個計數），測試與探針讀它。
形狀與 `cmd/pool-game/hp_lock_test.go` 的治具版相同。治具版改成呼叫同一支寫回函式，
但仍由治具掛在 `afterTick`：治具要跨 app 實例（讀檔之後換新的 app）統計、不寫
`CheatsUsed`，而且不能讓探針存檔多出作弊標記。

**限制**：寫回在 tick 結束時才做，一個 tick 內從滿血打到全滅，仍會看到全滅畫面。
治具版在補十三的整趟路線量到 0 次。

## 一擊斃命

定義：**隊員造成的傷害大於 0 時，傷害改成目標剩下的 HP**。擲骰次數與順序不變
（命中、傷害骰、豁免照常擲），只改結果，所以亂數消耗與關著時相同。

戰術盤上減少 HP 的路徑（`grep 'HitPoints\[' cmd/pool-game`，2026-09-17 盤點）：

| 路徑 | 位置 | 造成者 | 蓋不蓋 |
|---|---|---|---|
| 近戰、遠程、快速戰鬥 AI 的攻擊 | `resolveTacticalAttack`（`tactical.go`）每一下命中的傷害 | 攻擊者（隊員或敵人） | 攻擊者是隊員就蓋 |
| 單體傷害法術 | `applySpellDamage`（`cast.go`），傷害已經過 `damageAfterSave` | 施法者 | 施法者是隊員、豁免後傷害仍大於 0 就蓋 |
| 範圍傷害法術 | 同上，對範圍內每一個目標各呼叫一次 | 施法者 | 同上 |
| 能量吸取 | `applyEnergyDrainSpecialAttack`（`tactical.go`） | 怪物的特殊攻擊，目標是隊員 | 不蓋（不是隊員造成） |
| ECL `DAMAGE`（spec 084） | `damageOne`（`damage.go`），直接改 `state.Party` | 腳本（陷阱、箭雨），目標是隊員 | 不蓋 |

豁免規則「成功就無效」的法術，豁免成功時傷害是 0，不蓋，照原版無效。「隊員」的判準是
戰術盤的 `PartySlot[index] >= 0`。**目前 remake 沒有其他會讓敵人 HP 減少的路徑**；之後新增的傷害
路徑要回到這張表登記。

## 說明頁

F1 說明頁原本就放不下：鍵說明 11 行、英文指令 3 行、出處 5～7 行，每行 22 像素、從 y=100 起畫，
第 14 行已經壓在繩索框上，出處整段畫到螢幕外（2026-09-17 發行包 `v.1.1.9-20260917` 實拍）。
改成兩頁：

| 頁 | 內容 |
|---|---|
| 1／2 | `ui.helpKeys` 鍵說明（新增一行 `F6：作弊選單（原版沒有）`） |
| 2／2 | 英文指令三行、空一行、出處各行 |

行距 20，第一行 y=100，框鋪到 y=364；兩頁都放得下 13 行。標題列右邊寫
「←→ 翻頁　F1 關閉」。`←`／`→` 翻頁，`F1` 關閉。畫面識別字：第 1 頁 `help`，第 2 頁 `help-2`。

## 標示

| 位置 | 條件 | 內容 |
|---|---|---|
| 冒險畫面右上角（「自訂規則」那一行的下一行） | 任一開關開著 | 「作弊：鎖 HP／一擊斃命」（只列開著的） |
| 冒險畫面右上角（同一行） | 開關都關、`CheatsUsed` 為真 | 「開過作弊」 |
| 戰鬥畫面資訊欄最上面一行 | 任一開關開著 | 同冒險畫面 |
| 隊伍選單右下（`396,336`） | `CheatsUsed` 為真 | 「開過作弊」 |

中英兩份 locale 都要有，並在發行包上用真實字形截圖確認不超出安全矩形（`tools/capture-help-cheats.sh`）。
標示裡兩個開關之間的分隔字由 locale 給（`ui.cheatMarkSeparator`：中文「／」、英文「 / 」）——英文字型沒有全形斜線，
直接寫「／」會畫成 `@`。

截圖（`v.1.1.10-20260917` 發行包，2026-09-17）：`docs/screenshots/cheat-menu/` 的 `zh-*`／`en-*`，
含說明頁兩頁、作弊選單、冒險畫面與戰鬥畫面的標示。

## 實作與測試

- `cmd/pool-game/cheats.go`：`cheatInput`（F6、L、O、ESC）、`setCheat`（寫 `CheatsUsed`）、
  `applyCheatLockHP`（`Update()` 開頭 `defer`）、`restorePartyHitPoints`（治具共用）、`cheatDamage`、
  `cheatMark`、`drawCheatMenu`。
- 掛點：`resolveTacticalAttack` 的每一下傷害、`applySpellDamage` 的開頭；冒險畫面、戰鬥畫面、隊伍選單的標示；
  `drawHelp` 兩頁、`screenName` 的 `help-2` 與 `cheat-menu`。
- `internal/save/state.go`：`Cheats`、`CheatsUsed`。
- 測試：
  - `TestCheatMenuTogglesFromKeysAndRemembersItWasUsed`：按鍵開、切、關；選單開著時方向鍵不走路；存讀檔帶著。
  - `TestCheatLockHPTurnsTheSlumsGuardsWipeIntoAWin`：seed 136 衛兵攔截，不開全滅、開了 `6DC7 = 0`。
  - `TestCheatOneHitKillMakesEveryPartyHitLethal`：seed 136 獸人的家；關著時打到沒打死 11 次，開著 0 次。
  - `TestCheatOneHitKillCoversPartySpellsButNotFoes`：法術路徑與敵人造成的傷害，直接呼叫。手動建的隊伍只記催眠術，
    按鍵走不到施法傷害那一支。
  - `TestHelpPagesListTheCheatKeyAndFitTheBox`：中英兩頁不超出框，第 1 頁有 F6。
  - `TestCheatsRoundTripAndOldSavesReadAsOff`（`internal/save`）：舊存檔讀回來全關，新欄位往返。
  - `TestMainlineProbeHouseRuleCheatAtNorris`：探針只在諾里斯那一場開作弊（playtest 補十四）。
