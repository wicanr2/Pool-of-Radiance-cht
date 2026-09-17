# Goal：作弊選單（鎖 HP、一擊斃命）——探針在諾里斯那一場打開，贏了之後照正常強度跑到卡住（GitHub #43；#22、#5）

主台帳：[issue #43](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/43)。
相關：[#22](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/22)（正常強度的策略層）、
[#5](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/5)（正常強度通關，維持 open）、
[#42](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/42)（休息後重複文字，對拍會停在這裡）。
上一個 goal 的結案位置：[`issue-41-block-load-scratch-clear.md`](issue-41-block-load-scratch-clear.md)〈2026-09-17 收在哪〉
（#41、#40 關閉：HP 鎖定探針從標題跑到結局）。

## 使用者的決定（2026-09-17）

使用者的原話：「增加作弊選單 F1 Help 會顯示選單按鍵，可以強制鎖 HP、強制一擊斃命，跟諾里斯打仗時強制打開過關」，
驗收則是「贏諾里斯後繼續跑到卡住為止」。

這一條與 CLAUDE.md §3／§6 有衝突：原版忠實是驗收基準、主線驗收不得依賴 forced-win。處理方式照
spec 140（自訂規則）的前例，而且標示要比 spec 140 更明顯：

- 預設關。開著時冒險畫面與戰鬥畫面都有標示。
- 存檔裡留一個**開過就不會消失**的標記，讀檔後照樣顯示。
- 開過作弊的路線不算 #5 的收據。#5 維持 open。

「跟諾里斯打仗時強制打開」這裡解讀成：**探針在諾里斯那一場用按鍵打開，打完關掉**。產品本身不會
在遇到諾里斯時自動打開。這個解讀若和使用者的意思不同，停下來問。

## 現況（2026-09-17，`5554ca0` 已推送）

- **F1**：`cmd/pool-game/main.go` 約 776 行 `a.help = !a.help`，說明頁 `drawHelp`（約 4020 行）畫
  `msgHelpKeys`（`internal/gamepack/pack/20-locale.zh-TW.json`／`20-locale.en.json` 的 `ui.helpKeys`，
  一行一句）加 `adventureCommandHelp()`。框是 (72,54)–(568,340)，已經畫了 11 行鍵說明、指令說明與出處，
  **剩下的空間不多**，要先量。
- **已用的功能鍵**：F1 說明、F2 配色、F3 攻略、F4 素材總覽、F5 戰鬥預覽、F10 存檔離開。F6～F9、F11、F12 未用。
- **自訂規則前例**：`poolsave.State.HouseRules.CommissionExperience`（`internal/save/state.go` 約 194 行），
  由隊伍選單的 `H` 切換（`party_menu.go` 約 305 行），冒險畫面右上角標「自訂規則」（`main.go` 約 3482 行），
  規格在 spec 140。
- **隊伍對目標扣 HP 的直接寫入**：近戰 `resolveTacticalAttack`（`tactical.go` 約 1814 行）、法術
  `cast.go` 約 685 行。遠程、特殊攻擊、ECL `DAMAGE`（spec 084）、能量吸取（spec 112）等其他路徑**還沒盤點**。
  一擊斃命只做近戰，會變成「這條路打得贏、那條路打不贏」。
- **HP 鎖的治具版**：`cmd/pool-game/hp_lock_test.go`。每個 tick 把戰鬥中的隊員補滿、倒地拉回 0、
  瀕死計數歸零、佔格類別拿回來；戰鬥外補 `state.Party`。產品版可以照這個形狀做，但要放在 `Update()`
  裡，而且寫回次數要能從外面讀到。
- **諾里斯**：`ecl8/29`（地面）→ `LOAD FILES 32` 換井底 GEO8/32，井底三格地形 12 → `9D1Bh` → `9DD4h`
  FIGHT（`LOAD MONSTER 32×1、57×5、1×9`）。贏了寫 `4A24 = FF`、槽 0 `4AA6 = FE`（spec 137〈古托井與索寇要塞的門〉）。
  駕駛 `fightNorris` 在 `cmd/pool-game/mainline_house_rule_test.go` 約 307 行。
- **探針**：
  - `TestMainlineProbeHouseRuleCommissionExperience`（`runMainlineProbe(t, true, 142)`）在 #41 之後改停在
    古托井 (13,4)：輸給諾里斯，A／B／D 倒地又睡不成。
  - 2026-09-17 之前 house rule seed 136..475 的諾里斯勝率約 15%（playtest 補十一）；那是 #41 修正之前量的。
  - `TestMainlineProbeHPLockedRouteA`（全程鎖血的治具版）走得到結局。
- **全套紅燈五條**：`TestSokalKeepOpensTheOtherBoatRoutes`、`TestWorldTourReachesTheAreasBehindTheHarbour`、
  `TestPlayingTheWorldCompletesCommissionsOnItsOwn`、`TestMainlineProbeNaturalPartyFirstBattle`、
  `TestMainlineProbeHouseRuleCommissionExperience`。
- **對拍**：`tools/appimage-dos-parity.sh` 目前會停在 field-cast 之前（#42）。已拍到的 33 張與 v.1.1.5 逐項相同。

## 提示詞（可直接貼給 `/goal`）

> 目標：加一個預設關的作弊選單（鎖 HP、一擊斃命），F1 說明頁列出它的按鍵。house rule 探針在諾里斯那一場
> 用按鍵打開、打完關掉，多個 seed 都過得了諾里斯；之後照正常強度往下跑，記下每個 seed 卡在哪裡。
>
> 1. **先寫規格。** 新增 spec 141（DRAFT → READY），內容包括：
>    - 開關的按鍵：建議 F6 開作弊選單，選單裡兩個開關；
>    - 兩個開關各自的語意與作用範圍（只在戰術盤上，還是戰鬥外也算）；
>    - 存檔欄位：兩個開關，加一個開過就不清的 `CheatsUsed`；
>    - 畫面標示的位置、文字與安全矩形（CLAUDE.md §8）；
>    - F1 說明頁怎麼放下新的一行；
>    - 「關著時與原版完全一致」的保證範圍；
>    - 為什麼可以做：引用使用者決定與 spec 140 前例。
>
>    寫進 `docs/spec/000-index.md`（跑 `cmd/pool-doc-index`）。
> 2. **盤點隊伍造成傷害的全部路徑。** `grep` 所有對 `tacticalState.HitPoints` 減值或歸零的地方，
>    再加上會讓敵人死亡或離場的狀態路徑，逐條列在 spec 141：
>    - 近戰、遠程、法術（含範圍）、ECL `DAMAGE`、能量吸取、毒、特殊攻擊；
>    - 每一條要標明「是不是隊伍造成的」「一擊斃命要不要蓋」。
>
>    一擊斃命的定義：隊員的攻擊**命中**時，目標 HP 直接歸零。擲骰照常，亂數消耗不變；只改傷害結果。
>    法術在目標沒豁免成功時才算。有漏掉的路徑就寫明「不蓋」並說明原因。
> 3. **產品實作**（`cmd/pool-game`、`internal/save`；**不改 `internal/combat` 與共用 engine**）：
>    - `poolsave.State` 加 `Cheats{LockHP, OneHitKill bool}` 與 `CheatsUsed bool`（`omitempty`，舊存檔讀得回來），
>      有 round-trip 測試。
>    - 作弊選單開、切、關都從按鍵來；切換時狀態列印一句說明。只要打開過任何一個，`CheatsUsed = true`，之後不能清。
>    - 鎖 HP：開著時在 `Update()` 裡，照 `hp_lock_test.go` 的形狀每個 tick 寫回，並記次數（給測試讀）。
>    - 一擊斃命：照第 2 步的清單，在「命中且是隊員造成」的那一點把傷害改成目標剩餘 HP。
>    - 標示：開著時冒險畫面與戰鬥畫面都標「作弊：鎖 HP／一擊斃命」；`CheatsUsed` 為真時標題選單的存檔資訊也標出來。
>    - F1 說明頁加一行作弊鍵，中英兩份 locale 都要加，並做長文字的安全矩形截圖檢查。
> 4. **測試**（一律從 `Update()` 送鍵）：
>    - 選單的開、切、關，`CheatsUsed` 保持為真，存讀檔後照樣在。
>    - 關著時的負對照：同一個 seed 的貧民窟衛兵攔截（地形 13，seed 136）照樣全滅。
>    - 鎖 HP 開著：同一場 `6DC7 = 0`，戰鬥收場。
>    - 一擊斃命開著：第 2 步清單上每一條要蓋的路徑各一條測試，都是命中即歸零；關著時同一擲骰不歸零。
>    - F1 說明頁有作弊鍵那一行（中、英）。
>    - 把 `hp_lock_test.go` 的治具版改成呼叫產品版，或寫明兩者分開的理由。`TestMainlineProbeHPLockedRouteA`
>      要照樣走到結局。
> 5. **探針：諾里斯那一場開作弊。**
>    - `fightNorris` 在井底開打前用按鍵打開兩個開關，戰鬥收場（`4A24 = FF`）之後關掉；其他戰鬥一律不開。
>    - 在 house rule 探針上掃 seed 136..150：每個 seed 記下有沒有走到諾里斯、有沒有打贏、之後卡在哪一段
>      （段落、旗標、時刻、等級、倒地的人），做成一張表。
>    - 走不到諾里斯的 seed（例如古托井地面就倒地又睡不成）照實記下，不在那裡開作弊。
>    - 探針的收據 seed 維持 142；它贏了諾里斯之後停在哪，寫進測試註解與 playtest。
>    - 這一條測試照樣會紅（卡在正常強度的牆上），那是預期的，失敗訊息要指出卡在哪一段。
> 6. **對拍。** 作弊關著時畫面不應該變，但 F1 說明頁與存檔資訊的版面會變。
>    `tools/package-release.sh` 打包之後跑 `tools/appimage-dos-parity.sh`：
>    - 遇到 #42 停下時，照上一輪的做法，只對已經拍到的畫面跑同一支比對程式，不改量測腳本；
>    - 把變好、變差、不變的數字寫進 commit message；
>    - 動到的說明頁不在抽樣裡的話，補一張真實字形截圖放 `docs/screenshots/`。
> 7. **文件。**
>    - spec 141 READY；README 的「自訂規則」那一段補上作弊選單，寫明不算原版驗收；
>    - playtest 補十四（seed 表）、spec 137〈撐強度的段落〉引用這張表；
>    - CONTEXT.md；本 goal 補〈收在哪〉。
> 8. **台帳。**
>    - #43 留言附 seed 表與卡住的位置；產品與測試都做完才關。
>    - 卡住的地方如果是 remake 與原版不符，先開 issue 帶證據；如果是戰力或策略，就在 #22 留言。
>    - `docs/worklist.json` 加 #43 條目，完成後移除；依序跑 `pool-worklist -mode render -write WORKLIST.md` →
>      `-mode verify` → `gh issue list --state open`，雙向核對。
>    - commit 與 push 本 repo 可以做；其他 repo 不動。
> 9. **不可越線。**
>    - 作弊預設關；關著時既有測試與對拍畫面不得變。
>    - 不改 `internal/combat`、共用 engine、CoAB。
>    - 除了作弊開關本身，不注入旗標、座標、時鐘、金錢、經驗值。
>    - 暫時的掃描與追蹤碼收尾前刪掉；用腳本改檔前先驗錨點唯一。
>    - `tools/go.sh fmt` 只排新檔。
> 10. **停止線。** 碰到下列任一情況，停下來回報：
>     - 一擊斃命要改 `internal/combat` 才做得到；
>     - 作弊關著時有既有測試或對拍數字變了，而且找不到成因；
>     - 同一個非戰力卡點出現第三次；
>     - 單一 seed 的探針超過 10 分鐘；
>     - 使用者的「強制打開」原意需要產品自動開作弊。

## 2026-09-17 收在哪

第 1～8 步做完，#43 關閉（`879adc2`）。

- **規格與盤點**：spec 141 READY。隊伍造成傷害只有兩個入口：`resolveTacticalAttack` 與 `applySpellDamage`；
  能量吸取、ECL `DAMAGE` 只打隊員，不蓋。
- **產品**：`cmd/pool-game/cheats.go`。存檔新增 `Cheats`／`CheatsUsed`。F1 說明頁本來就超出框，改成兩頁。
  標示的分隔字交給 locale，因為英文字型的全形斜線會畫成 `@`。
- **測試**：選單按鍵與存讀檔；鎖 HP 正反對照；一擊斃命近戰正反對照（打到沒打死：關著 11 次、開著 0 次）；
  法術與敵方傷害；說明頁兩頁；存檔往返。治具 `hp_lock_test.go` 改呼叫產品的 `restorePartyHitPoints`，
  `TestMainlineProbeHPLockedRouteA` 照樣綠。
- **探針**：seed 136..150 裡，走到諾里斯的 6 個全部打贏；之後 3 個停在古托井地面、3 個死在索寇中庭的巡邏；
  9 個走不到諾里斯（playtest 補十四）。收據 `TestMainlineProbeHouseRuleCheatAtNorris`（seed 142）會紅，
  停在古托井地面 (13,4)。
- **對拍**：`v.1.1.10-20260917`，照樣停在 #42；拍到的 33 張逐項不變。截圖用 `tools/capture-help-cheats.sh`
  拍，放在 `docs/screenshots/cheat-menu/`。
- **全套紅燈**：六條，原本五條加上新的收據探針。
- **沒有觸發停止線**：沒有改 `internal/combat`，作弊關著時沒有測試或對拍數字變化，單一 seed 約 5 秒。

## 已知風險與待決

- **F1 說明頁的空間**：框已經畫了十幾行。加一行可能超出 (568,340)，也可能擠掉出處那幾行。先量，
  再決定要不要分頁或縮行距。
- **傷害路徑漏網**：一擊斃命若只蓋近戰，探針有可能在法術或遠程那幾回合打不死，看起來像「作弊沒生效」。
  第 2 步的清單就是為了擋這件事。
- **亂數消耗**：一擊斃命只改傷害結果，不改擲骰次數。但敵人死得早，後面回合的擲骰序列還是會變，
  這是預期內的，不算回歸。
- **HP 鎖的時機**：治具版在一個 tick 內死掉的情況量到 0 次（補十三）。產品版若把寫回放在
  `Update()` 前段，時機不一樣，要重量一次。
- **正常強度的牆**：過了諾里斯之後，預期卡在索寇要塞 (8,5) 那 50 隻或古托井地面。那是 #22 的工作，
  這一輪只負責記錄位置，不負責打過去。
- **#42**：對拍會停在 field-cast 之前，數字只有 33 張。
