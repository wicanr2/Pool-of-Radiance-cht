# Spec 150：戰後結算頁、NPC 分錢頁、選單一定開、TREASURE 的時機、有資格的人數與決鬥

狀態：CONFORMED（結算頁 `08E0h` 的標題與兩行字、之後才開選單、選單文字框是空的：碼 exact、
原版 dosgolem 收據逐字相同；NPC 分錢頁的欄與列：碼 exact、原版 dosgolem 診斷收據相同）；
READY（沒有戰利品也開選單、`TREASURE` 等到戰鬥打完、第二隻起的效果串列反序、有資格分經驗值
的人數、`CALL 8001h` 決鬥、`CALL 8000h` 競技場、隊員踏出盤面逃走與 "The party has fled."、
選單的 ` Detect`：碼 exact，remake 有從 `Update()` 送鍵的測試，沒有原版執行期收據）；
DRAFT（見〈還沒接〉）。
日期：2026-09-27。主台帳：GitHub issue #95（#76 留下）、#103（#94 留下）、#111。接 spec 142 與 148
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
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` |
| 22 | `967065cc35975465a7250026c63b8a5ae06b812b228abcfbbbd83d636538dda8` |

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
`The party has fled.` 見〈隊伍逃走〉。

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

所以打完一場什麼都沒掉，選單是 `View Pool Exit`。` Detect` 見〈Detect〉。remake：`openMonsterLoot` 不再因為
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

## NPC 分錢頁：overlay-05 `1387h..146Ah`（exact 碼；版面有 dosgolem 診斷收據）

```
1387..13A8  150h:0310h(1, 1, 26h, 16h, 0, 0Fh, "")      ; 與結算頁同一個清法
13B9  C6 46 F8 00      [bp-8] = 0
13C5..13DE  +84h > 7Fh、+10Ch == 0、+85h > 0 的每一個（spec 148）
13E6..1418  198h:0039h(5, 5 + [bp-8], 22h, 16h, 0Ah, 1, 名字 + " takes and hides his share.")
141D  80 46 F8 02      [bp-8] += 2
1433..146A  "press <enter>/<return> to continue"（1272h），11Dh:002Fh 等一個鍵
```

`198h:002Fh` 的前兩個參數是欄與列（結算頁那三行，dosgolem 量到第 1 欄第 3／5／7 列）；
`198h:0039h` 是同一族、多了右下角的界限，前兩個參數是第 5 欄、第 5、7、9…列。

原版收據：`docs/audit/dosgolem-npc-share-screen.json`（`tools/dosgolem-npc-share-screen.py`，
dosgolem `57454c4`，從 `slums.state`（SHA-256 `ebbb65e8…`）出發走 cheat 通關的市政廳交件）。
cheat 通關的隊伍沒有 NPC，所以這一份是**診斷樣本**：出發前把隊伍鏈第二與第四個人（C、E）的
`+84h`／`+85h` 從 `00 00` 改成 `B2 01`，只拿來量版面，不拿來證明有 NPC 的隊伍走得到這裡。
結果：清掉的框內兩行 `C TAKES AND HIDES HIS SHARE.`、`E TAKES AND HIDES HIS SHARE.`，
名字的第一個字在第 5 欄、第 5 與第 7 列，底列 `PRESS <ENTER>/<RETURN> TO CONTINUE`；按一下
才是 `THE PARTY HAS FOUND TREASURE!` 那一頁。remake：`hideNPCShares` 回名字，
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

## 競技場：`CALL 8000h`（exact 碼）

`ecl3/11` 的競技場：`WHO WILL DUEL?`（`9C72h`）→ `COMPARE @6BB8, 128 ; IF <`（`9C80h`）→
`9CA5h CLEARMONSTERS` → `9CA6h CALL 8000h` → `9CAAh COMBAT` → `9CABh EXIT`。士氣 128 以上的
（NPC）印 `I CANNOT ARRANGE SUCH A DUEL.`。

overlay-07 entry 26 以參數 1 進來，`1AB9h..1B15h` 與 `8001h` 相同，之後（`1B17h` 參數非 0）：

```
1B20..1B36  Pascal "cpic"（1AA9h：04 63 70 69 63）、0Bh、DS:6D49h → 147h:0039h（overlay-33 entry 5）
1B3B..1B66  沿 DS:5CF4h 的 +104h 走到鏈尾
1B6D  B8 1D 01 / 1B71 9A 29 03 BB 05          GetMem(11Dh)
1B80  B8 1D 01 / 1B84 9A 5D 02 BB 05          Move(目前角色 [5CF0h] → 新記錄, 11Dh)
1B8C  26 C6 85 0D 01 01                       +10Dh = 1
1B97  26 89 85 04 01 / 26 89 85 06 01         +104h = nil
1BA1  BF AE 1A … 1BAF 9A 4E 06 BB 05          名字 = "ROLF"（1AAEh：04 52 4F 4C 46），上限 0Fh
1BB7  26 C6 85 0F 01 01                       +10Fh = 1（AI 走，spec 139 的 Quick 同一個欄位）
1BC0  26 C6 85 0E 01 01                       +10Eh = 1（敵方）
1BC9  26 C6 85 84 00 B2                       +84h = B2h（士氣 100、要判，spec 091）
1BCF  A0 49 6D / 1BD5 26 88 85 BF 00          +BFh = DS:6D49h（造形槽位）
1BDF  26 89 45 7F / 26 89 85 81 00            +7Fh = nil（效果清空）
1BED  26 89 85 C8 00 / 26 89 85 CA 00         +C8h = nil
1BFF  26 89 85 04 01                          接在鏈尾
1C19..1CDF  沿目前角色的 +C8h：GetMem(3Fh)、Move，第一件是頭、之後的插在頭上（舊的頭寫進新節點 +2Ah）
```

所以複製品是**整筆照抄**（能力值、生命值、職業等級、記憶法術、造形 `+BDh..+C6h`），改的只有名字、
陣營、AI、士氣、造形槽位、效果與物品鏈；物品順序是反的（與 spec 142 怪物物品、本份〈第二隻起的
效果串列反序〉同一個形狀）。`147h:0039h` 載入哪一份造形沒讀：remake 畫複製品用上場那個人的造形
（strong inference：`+BDh..+C6h` 在 Move 裡照抄）。

打完：

```
overlay-05 entry 2  0006  80 3E 9A 82 00 / 26 83 BD CC 05 00   決鬥而且 @6DE6 != 0
                    0019..002E  [5CF0h]+73h × 100 寫進參數，0032 E9 FF 02 跳到 0334h
                                ；不除人數、不收錢與物品（0035h 起的迴圈整段跳過）
overlay-05 1164h    119E  26 80 BD 0E 01 01   +10Eh == 1 的記錄
                    11A9  +10Dh != 1 → [4937h]+590h（@6DC8）加一
                    11EF  9A 2F 00 B6 00      overlay-16 entry 3(runtime +13h, 1)：從鏈摘掉、放掉造形槽位；
                                              複製品排在隊伍人數之外（+13h = 1），所以 +67Ch 不減（3323h）
```

`5CF0h` 在 entry 2 那一刻指的是上場的人或它的複製品，兩者的 `+73h` 相同。`entry 3`（`0794h`）
照職業調整上場那個人的份額。

remake：`applyScriptCall` 的 `8000h` → `startArenaDuel`（`postcombat.go`）把複製品接在
`a.state.Party` 尾端、`Side = 1`、`Quick`、效果清空、物品反序；`consumeInitialSearch` 在
`COMBAT` 沒有 `LOAD MONSTER` 但有複製品時照常開戰；`deployRoster` 把它擺到對面（陣營非 0 的
那一條）、`boardIconFor` 用它照抄來的造形、`enterTacticalPreview` 記 `B2h` 的士氣；
`finishCombat` 摘掉複製品（`removeArenaCopy`），每份是最高職業等級 × 100（`arenaDuelExperience`）、
只有上場的人拿到、不收複製品身上的東西。隊伍裡其他 `+10Eh == 1` 的記錄（倒戈的 NPC、ADD NPC 18h）
由 `removeOpposingMembers` 摘掉，結算見 spec 167。`@6DC8` 那一個計數沒有接（腳本 `9CABh` 直接 `EXIT`，
沒有讀它）。

## 隊伍逃走（exact 碼）

隊員踏出盤面，overlay-08 的移動常式（spec 058）問一句：

```
0BF7  80 7E F7 00 / 75 42        目的格類別 0（盤面外）
0C02  BF B5 09                   "Flee:"（09B5h：05 46 6C 65 65 3A）
0C15  9A 3E 00 1D 01             11Dh:003Eh(0Dh, 0Ah, 0Fh, 字串)：Y／N
0C1A  3C 59 / 0C24 9A 43 00 96 00   'Y' → overlay-13 entry 7（0C6Ch）
0C31  3C 4E / 0C38 26 C6 05 00      'N' → 什麼都不做
```

overlay-13 entry 7 與怪物逃跑同一支（spec 096〈脫離戰場〉）：對面沒有人、或自己比對面最快的快就
逃掉，一樣快擲 d2；逃掉的走 overlay-24 entry 11(記錄, 3, "Got Away")——`+10Ch = 3`、生命保留、
`+10Dh = 0`；逃不掉印 "Escape is blocked"。兩條都結束這個行動。

戰後 overlay-05 `04ADh`：

```
04B8  C6 06 60 49 01             DS:4960h = 1（全滅）
04E8  26 80 BD 0C 01 03 / 04F0 C6 06 9D 43 01    有人狀態 3 → DS:439Dh = 1
054E..058E  狀態 3／1／0、+10Eh == 0、+84h < 80h → 4960h = 0
05A9  C6 06 A0 82 01 / 05AE C6 06 9D 43 00       有人狀態 0／1 → 82A0h = 1、439Dh = 0
062F  80 3E 9D 43 00             439Dh 立著（有人逃掉、沒有人站著）：
068C  26 C7 85 8E 05 81 00         @6DC7 = 81h
0696  狀態 3 → 0、+10Dh = 1
06B2..06C4  其餘的人 9A 2F 00 B6 00  overlay-16 entry 3(0, 1)：從隊伍鏈摘掉、隊伍人數減一
```

之後 `14CAh` 照常：4960h 是 0 所以不是全滅，`1295h` → `08E0h`（439Ch 是 0——entry 2 沒跑——所以
印 `The party has fled.`，數字清 0）→ `0E85h`。82A0h 是 0，entry 2／3 不跑：沒有經驗值，
怪物身上的東西不進來，`TREASURE` 先放上去的那一份還在。**逃掉的只有 NPC（`+84h` 位元 7）時
4960h 還是 1，照全滅處理。**

remake：`tacticalInput` 在 `ResolveDestination` 回 `Leaving` 時立 `FleePrompt`、底列換成
`ui.tacticalFleePrompt`；Y 走 `partyLeaveCombat`（`party_flee.go`，與 `foeLeaveCombat` 同一組
規則），N 什麼都不做。收場時 `finishCombat` 在「沒打贏也不是決鬥」那一支先問
`partyFledOutcome`（439Dh／4960h），成立就 `leaveBehindAfterFleeing`（留下的人從隊伍摘掉、
逃掉的換回 0、@6DC7 = 81h），開 `postCombatReport{fled: true}`。戰鬥中丟出去的武器
（`ThrownLoot`，spec 151）在這一支沒有收進選單，原版那些在串列上（差異，還沒接）。

## Detect（exact 碼；作用 DRAFT）

`0E85h` 每一圈（`115Dh` 跳回 `0E9Fh`）重組選項：

```
0EA9  9A 66 00 D9 00             [bp-1] 有錢、[bp-2] 有物品
0EE0  80 7E FE 00 / 74 53        沒有物品 → 後綴 " Exit"
0EE6..0F37  i = 0..14h：[5CF0h]+17h+i 是 05h（26 80 7D 17 05）或 0Bh（26 80 7D 17 0B）→ 記下、停
0F40  BF FB 0D                   後綴 " Detect Exit"（0DFBh）
102A  3C 44                      按 D：
103F  9A 39 00 E2 00             00E2h:0039h(編號, 0, 0, &[bp-106h]) = overlay-22 entry 5（0C14h）
```

比的是整個位元組，第 7 位立著（還沒記完，spec 070）的不算。overlay-22 entry 5 與探索施法同一支，
第三個參數 0 不印誰施了什麼（`0D23h`）；`0E8Dh` 以 overlay-25 entry 16（`9A 70 00 0A 01`）把法術
從記憶清掉，再派發 `DS:6A78h + 編號 × 4`——05h／0Bh 都是 `10D1h`，只以 `DS:6779h` 叫 `08BCh`
把效果碼 05h 掛上去（spec 073／074）。

remake：`treasureDetectOption`／`treasureDetect`（`treasure_detect.go`）；`enterTreasureMain` 在
`Exit` 前面放 `Detect`，選了就走營地施法那一支（`campSpellEffect`）、清掉那一格記憶、重組選單。

效果 05h 掛上去之後怎麼改變物品的顯示（View／Take 的清單是否標出魔法物品）還沒讀：
對 overlay-25 entry 27（`10Ah:00A7h`，效果串列搜尋）的 39 處呼叫，推的效果碼沒有一處是字面值
05h，讀取端可能是間接傳入的代碼或另一支。

## 還沒接（DRAFT）

| 項目 | 位址 | 等級 | 為什麼沒接 |
|---|---|---|---|
| 效果 05h 對物品清單的作用 | 讀取端未定位 | unknown | 〈Detect〉：掛上去已接，之後物品怎麼顯示沒讀 |
| 競技場複製品的造形載入 | overlay-07 `1B36h`（`147h:0039h`）、`+BFh` | strong inference | 畫的是照抄的造形；`cpic`／0Bh／槽位那一支沒讀 |
| 逃走那一支的丟出去的武器 | overlay-05 `0E85h` 的串列 | strong inference | `ThrownLoot` 只在有人站著時收 |
| 原版執行期收據：競技場、逃走 | — | — | dosgolem 沒有走到競技場或逃走的駕駛段 |

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
- `cmd/pool-game/npc_share_receipt_test.go`：同一份隊伍（B..F，第二與第四個人改成 NPC）走同一段
  交件，兩行的欄、列、字與 `docs/audit/dosgolem-npc-share-screen.json` 逐行相同。
- `cmd/pool-game/arena_duel_test.go`：從 `ecl3/11 9CA5h` 跑，`CALL 8000h` 之後隊伍尾端是 `ROLF`
  （陣營 1、AI、效果清空、物品反序）；盤面上只有上場的人與它，它用上場那個人的造形、士氣 B2h；
  打贏印 `The duelist receives 100`（1 級 × 100）、只有上場的人拿到、複製品摘掉；兩邊都交給 AI
  打到收場時複製品真的出手。
- `cmd/pool-game/party_flee_test.go`：`ecl4/10 A5DBh` 的戰鬥裡隊員踏出盤面先問 `Flee:`，N 留在原地、
  Y 逃掉；其餘的人倒著時收場印 `The party has fled.` 與 0、倒著的人從隊伍摘掉、@6DC7 = 81h、
  選單只有 `TREASURE` 的那一份。
- `cmd/pool-game/treasure_detect_test.go`：有物品而且目前角色記完了 Detect Magic 才有 `Detect`，
  選了掛上效果 05h、清掉最前面那一格，記憶用完選項就消失。
