# Spec 150：戰後結算頁、NPC 分錢頁、選單一定開、TREASURE 的時機、有資格的人數與決鬥

狀態：CONFORMED（結算頁 `08E0h` 的標題與兩行字、之後才開選單、選單文字框是空的：碼 exact、
原版 dosgolem 收據逐字相同）；READY（NPC 分錢頁、沒有戰利品也開選單、`TREASURE` 等到戰鬥
打完、第二隻起的效果串列反序、有資格分經驗值的人數、`CALL 8001h` 決鬥：碼 exact，remake
有從 `Update()` 送鍵的測試，沒有原版執行期收據）；DRAFT（見〈還沒接〉）。
日期：2026-09-27。主台帳：GitHub issue #95（#76 留下）、#103（#94 留下）。接 spec 142 與 148
的 DRAFT 表。

## 輸入與位址空間

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- overlay（`workplace/ovr/`，對 `docs/audit/dos-ovr-manifest.json`），位址一律是
  overlay-local file offset、base 0；反組譯是 `coab-go-test:20260729` 的
  `objdump -D -b binary -m i8086 -M intel`：

| overlay | SHA-256 |
|---|---|
| 03 | `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f` |
| 05 | `900ea1b8e57b03e0f6dd1c16024a686c8474e2b0ae9674c730d5619679809c16` |
| 07 | `a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae` |

- far call 由 `segment = (executable_file_offset − stub_offset − 3B0h) ÷ 16` 對 manifest 反查：
  `0045h:00A2h` = overlay-07 entry 26（`executable_file_offset 8A2h`、`stub 0A2h`、code `1AB3h`）。
- 原版執行期收據：`docs/audit/dosgolem-postcombat-screens.json`（`tools/dosgolem-postcombat-screens.py`，
  dosgolem `57454c4`，沿用 cheat 通關駕駛，從 `slums.state` 與 `handin-slums.state` 出發）。

## 一句話

打完或 `TREASURE → COMBAT` 之後，原版先發經驗值，有 NPC 拿走份額就清畫面逐行列出、等一個鍵，
再清畫面印 `The party has won.`（沒打怪是 `The party has found treasure!`）、
`Each character receives N`、`experience points.`、等一個鍵，**然後不論有沒有東西都開戰利品選單**，
選單的文字框是空的。`TREASURE` 後面接著一場有怪物的戰鬥時，選單等打完才開。

## 順序（exact，spec 148）

```
14CA  [4937h]+58Eh（@6DC7）= 0；DS:439Ch = 0
14E6  call 04ADh      ; 狀態換算、經驗值（entry 2／3）
14ED  4960h != 0 而且 829Ah == 0 → 1557h 全滅
14FC  0E E8 96 FD     call 1295h   ; NPC 分錢頁
1508  0E E8 D5 F3     call 08E0h   ; 結算頁（參數 DS:829Ch）
150C  0E E8 76 F9     call 0E85h   ; 戰利品選單
1606  26 89 85 CC 05  @6DE6 = 0
```

## 結算頁：overlay-05 `08E0h`（exact）

```
08EC..090E  150h:0310h(1, 1, 26h, 16h, 0, 0Fh, "")      ; 清掉整個框內
0913  80 3E 9C 43 00   DS:439Ch（打過怪）
091A  80 3E 9A 82 00   DS:829Ah（決鬥）              → 兩者皆 0 跳 09A1h
0924..0958  決鬥而且 DS:82A0h == 0 → "You have lost the duel."（0807h），參數清 0
095A..097F  決鬥               → "You have won the duel."（081Fh）
0981..099F  否則               → "The party has won."（0836h）
09A1..09CE  DS:439Dh（逃走）   → "The party has fled."（0849h），參數清 0
09D0..09E9  否則               → "The party has found treasure!"（085Dh）
            以上都是 198h:002Fh(1, 3, 0Ah, 字串)：第 1 欄、第 3 列
09EE..0A5B  Str(參數)；決鬥 "The duelist receives "（087Bh），否則 "Each character receives "（0891h）
0A60..0A6F  198h:002Fh(1, 5, 0Ah, …)   ; 第 5 列
0A74..0A8D  198h:002Fh(1, 7, 0Ah, "experience points.")（08AAh）
0A92..0ABD  "press <enter>/<return> to continue"（08BDh），11Dh:002Fh 等一個鍵
```

參數是 `14FFh` 推的 DS:829Ch——entry 2 的回傳值，也就是**職業調整之前**的每份。

原版收據（`docs/audit/dosgolem-postcombat-screens.json`）：

| 拍攝 | 第 3 列 | 第 5 列 | 829Ch | 439Ch |
|---|---|---|---:|---:|
| `handin-slums-00`（市政廳交件） | `THE PARTY HAS FOUND TREASURE!` | `EACH CHARACTER RECEIVES 540` | 540 | 0 |
| `podol-00`（第一場哥布林） | `THE PARTY HAS WON.` | `EACH CHARACTER RECEIVES 14` | 14 | 1 |
| `podol-02`（第二場） | `THE PARTY HAS WON.` | `EACH CHARACTER RECEIVES 24` | 24 | 1 |
| `podol-04`（交件） | `THE PARTY HAS FOUND TREASURE!` | `EACH CHARACTER RECEIVES 250` | 250 | 0 |

四張都是一整圈外框、框內沒有圖也沒有隊伍欄、第 7 列 `EXPERIENCE POINTS.`、底列
`PRESS <ENTER>/<RETURN> TO CONTINUE`；緊接著的選單頁文字框是空的（`menu-after-result`，
與 `docs/audit/dos-treasure-screens.json` 的 `top` 同一張）。

remake：`postcombat.go` 的 `postCombatReport`／`drawPostCombatPage`；戰利品選單
（`enterTreasureMain`）的文字框改成空的。畫面識別字 `treasure-result`。
`The party has fled.` 走不到：remake 的隊員不會離開盤面（`msgStatusOffBoard`）。

## 沒有戰利品時也開選單：overlay-05 `0E85h`（exact）

`0E85h` 是一個「畫選單 → 等鍵 → 分派」的迴圈，只有 `[bp-105h]` 立起來才出來，而入口沒有
任何提早返回：

```
0EA9  9A 66 00 D9 00   取 [bp-1]（有錢）、[bp-2]（有物品）
0EAE..0EBD  選項 = "View Pool"（0DE6h）
0EC7..0ED6  後綴 = " Exit"（0DF5h）；目前角色記著法術 05h 或 0Bh 而且有物品 → " Detect Exit"（0DFBh）
0F54  80 7E FF 00      有錢     → "View Take Pool Share"（0E08h）＋後綴
0F86  80 7E FE 00      否則有物品 → "View Take Pool"（0E1Dh）＋後綴
1047..1117  E（或 0）：錢與物品都沒了就直接出去；否則問 "There is still treasure left…"
```

所以打完一場什麼都沒掉，選單是 `View Pool Exit`。remake：`openMonsterLoot` 不再因為
「怪物那一側沒交出東西」就跳過選單。原版駕駛（`tools/dosgolem-cheat-playthrough.py`）
對每一場都認 `POOL … EXIT` 按 `E`，與此一致。

## TREASURE 的時機（exact 碼；ECL 盤點）

`27h TREASURE`（overlay-03 `1A82h`）只把公款寫進去、物品放上 `DS:676Eh`，不開選單；選單是
`24h COMBAT` 之後的 `14CAh` 開的。有怪物的 `COMBAT` 先打，打完 entry 2 把怪物的錢加進公款、
怪物的物品插在串列頭——所以 `TREASURE` 的物品排在最後，一起進同一個選單。

remake 的 VM 不在 `TREASURE` 停（它沒有事件），同一個結果裡會同時帶 `TreasureRequests` 與
`CombatRequested`。原本的分派先看 `TreasureRequests` 就開選單，離開選單再續跑時已經跨過
`COMBAT`——**那一場架整個沒打**。全 ECL 至少 16 處是 `LOAD MONSTER … TREASURE … COMBAT`
（一次性掃描，沒有進版控：與 `cmd/pool-ecl-memory-audit` 同一種指令圖——
`ecl.TraceGraphAtBaseWithCommands` 加 `gamepack.PoolCommandTable()`——依位址排序後看每條
`TREASURE` 前後各六條）：

```
ecl1/18 B15Dh、B18Fh      ecl1/24 B074h        ecl2/15 A591h        ecl2/20 B0FBh
ecl4/2  A705h             ecl4/10 A5E3h、A7E1h、AAFEh               ecl5/3  A73Bh
ecl5/5  ABD0h             ecl5/6  A2A6h        ecl6/25 AAA4h        ecl7/17 A193h、B355h、B69Ah
```

remake：`consumeInitialSearch` 在同一個結果裡有怪物的戰鬥時，先把 `TREASURE` 的公款寫入、
物品記在 `pendingTreasure`，照常排戰鬥；`finishCombat` 把它們接在戰利品串列的最後。

## 第二隻起的效果串列反序：overlay-03 `06FAh..07B4h`（exact）

與物品那一段（`0629h..06E9h`，spec 142）同一個形狀：

```
06FA  8B 86 B2 FE / 0B 86 B4 FE   第一隻的節點還沒走完
0707  新記錄 +7Fh 是空的 → 0722 9A 29 03 BB 05 GetMem(9)、073A 9A 5D 02 BB 05 Move、+5 清 0
0752  否則 GetMem(9)、Move，**舊的頭寫進新節點的 +5**（079A 26 89 45 05）
07A2  沿第一隻的 +5 走下一個
```

所以第二隻起效果節點的順序是反的。串列上的搜尋回最早掛上的那一個（spec 059），順序會影響
同一個代碼有兩個節點時取到哪一個。remake：`enterTacticalPreview` 對 `copyIndex > 0` 的那一隻
把效果反過來。

## NPC 分錢頁：overlay-05 `1387h..146Ah`（exact 碼，版面 strong inference）

```
1387..13A8  150h:0310h(1, 1, 26h, 16h, 0, 0Fh, "")      ; 與結算頁同一個清法
13B9  C6 46 F8 00      [bp-8] = 0
13C5..13DE  +84h > 7Fh、+10Ch == 0、+85h > 0 的每一個（spec 148）
13E6..1418  198h:0039h(5, 5 + [bp-8], 22h, 16h, 0Ah, 1, 名字 + " takes and hides his share.")
141D  80 46 F8 02      [bp-8] += 2
1433..146A  "press <enter>/<return> to continue"（1272h），11Dh:002Fh 等一個鍵
```

`198h:002Fh` 的前兩個參數是欄與列（結算頁那三行，dosgolem 量到第 1 欄第 3／5／7 列）；
`198h:0039h` 是同一族、多了右下角的界限，前兩個參數照同一個意思讀成第 5 欄、第 5、7、9…列
（strong inference：沒有帶 NPC 的原版收據）。remake：`hideNPCShares` 回名字，
`postCombatPageLines` 照這個版面排；英繁兩語（`ui.postCombatHidesShare`）。

## 有資格分經驗值的人數：`829Bh`（exact）

```
04B3  C6 06 9B 82 00             829Bh = 0
05B3  26 80 BD 0D 01 00 / 74 0B  +10Dh == 0 → 加一
05BE  26 80 BD 0C 01 01 / 75 04  +10Ch == 1 → 加一
05C9  FE 06 9B 82
0308  A0 9B 82 … 26 8B 85 7C 06 2B C2   除數 = 隊伍人數（[4937h]+67Ch）− 829Bh
035A  26 80 BD 0D 01 00          entry 3：+10Dh == 0 不發
0368  26 80 BD 0C 01 01          entry 3：狀態 1 不發
```

`+10Dh` 是「在場」：部署時為 0 的只擺屍體（spec 061 `1A99h`）；傷害把狀態打出 {0, 1} 時清成 0
（overlay-25 `2266h`，spec 084）；離開盤面（逃走、投降）清成 0（spec 096 `0F5Fh`）；戰後
`04ADh` 的 `0645h`／`0677h` 在狀態換回 0 時寫回 1（**在發經驗值之後**）；神殿 Stone to Flesh
寫回 1（spec 115）。計數迴圈在 `0541h` 碰到 runtime `+13h == 1` 的記錄就停——那是暫時的
參戰者（怪物；放不下時整筆摘掉，spec 061），所以只數隊伍那一段（strong inference）。

所以**打完那一刻倒著的、死掉的、開打前就倒著沒上場的隊員不分經驗值，除數也少一個**。
remake：`combatExperienceEligible`（盤面上體型非 0、狀態不是 1）；沒有盤面的
`TREASURE → COMBAT` 用 `treasureExperienceEligible`（狀態 0；`+10Dh` 在戰鬥外沒有別的寫入端，
strong inference）。

## 決鬥：ECL `CALL 8000h`／`8001h`（exact 碼）

overlay-03 `3026h`（`2Dh CALL` 的分派，remake 的 `applyScriptCall`）把運算元減 `7FFFh`：

```
3047  2D FF 7F                          ax = 運算元 − 7FFFh
30B4  3D 01 00 / 75 0A / B0 01 50 / 9A A2 00 45 00   1（8000h）→ overlay-07 entry 26(1)
30C3  3D 02 00 / 75 0A / B0 00 50 / 9A A2 00 45 00   2（8001h）→ overlay-07 entry 26(0)
```

overlay-07 entry 26（`1AB3h`）：

```
1AB9  C6 06 9A 82 01            DS:829Ah = 1
1ABE..1AC6  26 89 85 CC 05     @6DE6（[4937h]+5CCh）= 參數
1ACB..1B15  隊伍鏈上不是 DS:5CF0h（目前角色）的每一個：1AFF 26 C6 85 0D 01 00  +10Dh = 0
1B17  參數 0 → 返回
1B20..1CDF  參數 1：把目前角色複製一份（GetMem(11Dh)、Move），+10Dh = 1、名字 "ROLF"（1AAEh）、
            +10Fh = 1、+10Eh = 1（敵方）、+84h = B2h、+BFh = DS:6D49h、效果清空、物品逐件複製，
            接在鏈尾
```

三處呼叫（同一種一次性掃描，找運算元是 `8000h`／`8001h` 的 `CALL`）與前面的台詞（`cmd/pool-text-inventory`）：

| 位置 | 呼叫 | 台詞 |
|---|---|---|
| `ecl1/18 A9D3h` | `CALL 8001h` → `LOAD MONSTER` → `COMBAT` | `WHO WILL CHALLENGE THE BUCCANEER?` |
| `ecl8/16 A344h` | `CALL 8001h` → `LOAD MONSTER` → `COMBAT` | `WHO WILL BE MY CHAMPION?` |
| `ecl3/11 9CA6h` | `CALL 8000h` → `COMBAT`（沒有 `LOAD MONSTER`） | 競技場：`THE DUELS ARE EVENLY MATCHED … DO YOU DUEL?`、`WHO WILL DUEL?` |

戰後（overlay-05）：

```
0507  829Ah != 0 → 050E C6 06 60 49 00   DS:4960h = 0：輸了也不是全滅
0524  829Ah != 0 → 0748h（跳過 829Bh 的計數：決鬥的除數是整隊人數）
0748..07A7  +10Dh == 1、+10Ch == 0、+10Eh != 1 的每一個：82A0h = 1，呼叫 entry 2 與 entry 3
07A9..0800  狀態 0／1 → +10Dh = 1；狀態 5 → 4
0006..0032  entry 2：829Ah 而且 @6DE6 != 0 → 回 [5CF0h]+73h（最高職業等級）× 100
0612..0686  4960h == 0 → 狀態換算照跑（3→0、5→4、4 而且有 HP → 0）
```

overlay-03 `197Dh`（`C6 06 9A 82 00`）在 `COMBAT` 收場時清 829Ah；主流程 `1606h` 清 @6DE6。
`COMBAT` 的分派（overlay-03 `187Dh`）在 829Ah 立著時不看神殿／商店旗標、直接開戰——
競技場那一場沒有 `LOAD MONSTER` 也打得起來，對手就是那個複製品。

remake：`CALL 8001h` 接上（`startChampionDuel`）：只有目前角色上場（`duelDeploys`）；輸了
不是全滅、倒下的換成昏迷、結算頁印 `You have lost the duel.` 與 0；贏了印
`You have won the duel.`、`The duelist receives N`，N 是總額除以**整隊**人數，只有上場的人拿到。

## 還沒接（DRAFT）

| 項目 | 位址 | 等級 | 為什麼沒接 |
|---|---|---|---|
| 競技場決鬥（`CALL 8000h`） | overlay-07 `1B20h..1CDFh`、overlay-05 `0006h..0032h` | exact（碼） | 對手是目前角色的複製品（`ROLF`），remake 沒有「角色記錄變成敵方」這一條；只接旗標不接對手會變成不打就贏。`ecl3/11 9CA6h` 照舊什麼都不做 |
| `The party has fled.` | overlay-05 `09A1h`、`04ADh` 的 `0688h` | exact（碼） | remake 的隊員不會離開盤面，走不到；字串與判斷已接（`postCombatReport.fled`） |
| 選單的 ` Detect Exit` | overlay-05 `0EE0h..0F4Fh` | exact（碼） | 目前角色記著偵測魔法（05h／0Bh）時的第六個選項；探索施法的偵測魔法另案 |
| NPC 分錢頁的列位 | overlay-05 `13E6h` | strong inference | 沒有帶 NPC 的原版收據 |

## 被推翻的斷言

| 原本寫的 | 現況 | 證據 |
|---|---|---|
| `tactical.go`：「倒下的成員在原版一樣分得到——它擋的不是死亡」 | 倒下的 `+10Dh` 已清成 0，entry 3 不發、除數少一個 | 〈有資格分經驗值的人數〉 |
| spec 142：「怪物那一側什麼都沒交出來時不開選單」 | `0E85h` 一定開 | 〈沒有戰利品時也開選單〉 |
| spec 034：「`08E0h` 顯示 `The party has found treasure!` 等結果，接著 `0E85h`」 | `08E0h` 是獨立的一頁、等一個鍵；打過怪印 `The party has won.`；選單頁文字框是空的 | 〈結算頁〉 |

## 測試

- `cmd/pool-game/postcombat_test.go`（全部從 `Update()` 送鍵）：打完先是結算頁（`The party has
  won.`、`Each character receives N`、底列提示，N 是職業調整前的每份），Enter 之後是空文字框的
  選單；怪物身上什麼都沒有也開 `View Pool Exit`；倒下的隊員不分、其餘三人分總額的三分之一；
  `ecl4/10 A5DBh` 的 `TREASURE` 等戰鬥打完才開選單，公款是兩筆相加；同一條 `LOAD MONSTER`
  第二隻的效果串列反序；`ecl1/18 A9D2h` 的決鬥只擺目前角色、贏了只有他拿、除的是整隊人數；
  輸掉決鬥不是全滅、印 0、選單照開。
- `cmd/pool-game/loot_experience_test.go`：帶 SWORDSMAN 交件先是 NPC 分錢頁（第 5 欄第 5 列一行），
  Enter 之後才是結算頁。
