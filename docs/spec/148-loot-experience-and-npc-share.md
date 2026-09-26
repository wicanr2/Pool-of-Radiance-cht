# Spec 148：戰利品折算經驗值與 NPC 分錢

狀態：CONFORMED（公款折算經驗值：碼 exact、原版執行期收據逐人相同）；READY（物品加值折算、
NPC 分錢：碼 exact，remake 有測試，沒有原版執行期收據）；DRAFT（`DS:5CF8h` 停點、NPC
列名的畫面與停頓）。日期：2026-09-26。主台帳：GitHub issue #94（#76 留下，spec 142）。

## 輸入與位址空間

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- START.EXE（`workplace/START.EXE`）SHA-256
  `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`；RTL 段 `05BBh`
  以 MZ header `3B0h` 換算檔內位移。
- overlay（`workplace/ovr/`，對 `docs/audit/dos-ovr-manifest.json`），位址一律是
  overlay-local file offset、base 0；反組譯是 `coab-go-test:20260729` 的
  `objdump -D -b binary -m i8086 -M intel`：

| overlay | SHA-256 |
|---|---|
| 03 | `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f` |
| 05 | `900ea1b8e57b03e0f6dd1c16024a686c8474e2b0ae9674c730d5619679809c16` |
| 10 | `b929c7040aaa399ba803911c8630e06a8e21b9e7d64a67def69c700d7fe14439` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` |

## 戰後主流程的順序（exact）

overlay-05 entry 1 `14CAh`：

```
14E6  call 04ADh   ; 狀態換算；05E0h 看 DS:82A0h（有隊員站著）才呼叫
                   ;   entry 2（0000h，算經驗總額）與 entry 3（033Ah，發下去）
14EA  call 1164h
14ED  DS:4960h != 0 而且 DS:829Ah == 0 → 1557h 全滅畫面
14FC  call 1295h   ; NPC 分錢
1508  call 08E0h   ; 標題
150C  call 0E85h   ; 戰利品選單
150F..1552          放掉 DS:676Eh 整條串列、串列頭清 0
```

所以**經驗值先發、NPC 再分錢、最後才開選單**。經驗總額算的是分錢之前的公款。

沒有怪物的 `CLEARMONSTERS → TREASURE → COMBAT` 走的是同一個 `14CAh`（spec 036，overlay-03
`18C0h` 呼叫 `003Bh:0025h`）；`04ADh` 在隊伍鏈上找得到站著的隊員就設 `82A0h`，一樣呼叫
entry 2。所以**撿到的寶物與委任的獎金也折成經驗值**。

## 經驗總額：entry 2（exact）

`[bp-9]`（dword）在 `0035h` 清 0，接著三項加進去，最後 `0308h` 除一次人數：

| 位址 | 加什麼 |
|---|---|
| `00C0h..00F4h` | 每隻沒逃掉的敵方 `+0B8h + +0BAh × 生命值`（spec 097） |
| `0224h..02B1h` | 公款 `DS:6752h + 4i` 七欄（這時已加上怪物身上的錢，`0089h..00BEh`）依幣值換算 |
| `02B4h..0306h` | 戰利品串列 `DS:676Eh`（下一件 `+2Ah`）上、`DS:5CF8h` 那一件之前，每件加值 `+32h > 0` 的 `+32h × 400` |

公款換算的 bytes（`call 05BBh:0294h` 是 LongDiv、`05BBh:0279h` 是 LongMul）：

```
0224  C4 06 52 67 / B9 C8 00 / 9A 94 02 BB 05   銅     ÷ 200
023A  C4 06 56 67 / B9 14 00 / 9A 94 02 BB 05   銀     ÷ 20
0250  C4 06 5A 67 / B9 02 00 / 9A 94 02 BB 05   琥珀金 ÷ 2
0266  C4 06 5E 67 / 01 46 F7                    金     照加
0272  C4 06 62 67 / B9 05 00 / 9A 79 02 BB 05   白金   × 5
0288  C4 06 66 67 / B9 FA 00 / 9A 79 02 BB 05   寶石   × 250
029E  C4 06 6A 67 / B9 98 08 / 9A 79 02 BB 05   珠寶   × 2200
```

兩支 RTL 的身分由 START.EXE 檔內 bytes 認（exact）：`61D9h`（`05BBh:0279h`）是
`8B F0 8B FA F7 E1 … F7 E3 … F7 E1 … CB`，三次 `mul` 組 32×32 取低 32 位元；`61F4h`
（`05BBh:0294h`）先把兩個運算元取絕對值、`BD 21 00` 跑 33 輪移位相減、再依符號補回，是
有號長整數除法取商。spec 097 的 `0325h` 除人數用的是同一支。

物品那一項：

```
02DC  26 80 7D 32 00      cmp byte es:[di+32h], 0 ; jle 跳過
02E6  26 8A 45 32 / 98    al = +32h ; cbw
02EB  B9 90 01 / F7 E9    imul 400
02F0  99                  cwd                      ; 只留乘積低 16 位元再帶號擴展
```

所以加值 82 以上乘出來超過 7FFFh，會變成負數被加進去（原版照扣；遊戲資料裡沒有這麼大的
加值）。`+32h` 是物品的加值：spec 063（武器命中修正）、spec 080（護甲 AC）讀的是同一格
（strong inference 的語意，碼 exact）。

### 停點 `DS:5CF8h`

全 GAME.OVR 只有兩個寫入點：overlay-10 `1F59h`（`A3 F8 5C`）與一段全域初始化一起清成 0；
overlay-13 `1A3Bh..1A79h` 在隊員把物品丟進戰利品堆（GetMem(3Fh)、Move、插在 `676Eh` 串列頭、
`+34h` 清 0）之後，把 `5CF8h` 設成那個新節點。串列在 `14CAh` 的 `150Fh..1552h` 整條放掉，
而 entry 2 在同一個 `14CAh` 裡比選單早跑，所以**丟進去的那個節點在下一次 entry 2 之前已經放掉
了**；`5CF8h` 要對得上，只能是放掉的位址被 GetMem 重新配給新的一件物品。語意等級
`strong inference`（「玩家自己丟進堆裡的不算經驗值」），實際會不會對上取決於堆積配置，
remake 不模擬堆積位址，**串列上每一件都算**。

## 公款從哪裡來（exact）

- `1Ch CLEARMONSTERS`（overlay-03 分派 `33F8h..3401h` → `133Dh`）：`1356h..1362h`
  把 `6752h` 起 1Ch bytes 填 0，再放掉 `676Eh` 整條串列。
- `27h TREASURE`（overlay-03 `1A82h`）：`1AB5h` 以 `mov` 把七個運算元**寫入**公款（不是加）。
- entry 2 的 `0089h..00BEh`：怪物身上的七種錢加進公款（spec 142）。

remake：`TREASURE` 在 `enterTreasure` 取代公款（原本就是這樣）；有怪物的遭遇若同一個結果裡
有 `CLEARMONSTERS`，排遭遇時把 `PooledMoney` 清 0（`consumeInitialSearch`）。上一個戰利品
選單留在堆裡沒拿的錢，因此不會被下一場算進經驗值——除非腳本沒有先清，那原版也會算。

## 原版收據（exact）

`docs/audit/dosgolem-loot-experience.json`（`tools/dosgolem-loot-experience.py`，dosgolem
`57454c4`）比兩個 cheat 通關的狀態檔：交件前 `slums.state`（`ebbb65e8…a80fb`）與職員發完
獎金、停在戰利品頂層選單的 `handin-slums-stuck.state`（`2d976dfb…3e1d68`，spec 040／#47 的
畫面基準用的同一份）。兩者之間只有走路、睡覺與交件（`run-handin-slums.out` 沒有戰鬥）。

| | 值 |
|---|---|
| 公款（選單開著時） | 金 250、白金 50、珠寶 1；串列空、`5CF8h` 為 0、`829Bh` 為 0 |
| 換算 | 250 ＋ 50×5 ＋ 1×2200 ＝ 2700，五個人分 540 |
| 五個戰士（力量 17、16、18、14、15）實際多了 | 594、594、594、540、540 |

力量超過 15 的多拿十分之一（spec 097），逐人相同。同一個公款若照 spec 140 自訂規則的
金幣等值（珠寶 100）算只有 600——原版用的是 2200。

## NPC 分錢：`1295h`（exact）

```
12BC  +84h > 7Fh 而且 +10Ch == 0   → NPC 份數、總份數各加 (+85h & 7)
12EE  否則                          → 總份數加 1
1303  NPC 份數 == 0                 → 結束
1312..137C  七欄各自：公款 > 0（有號 dword）
1347    9A 94 02 BB 05   LongDiv(公款, 總份數)
134C    88 46 F9         每份 = 商的低位元組
135B    F7 EA / 99       每份 × NPC 份數（16-bit imul、cwd）
136C    29 8D 52 67 / 19 9D 54 67   公款 −= 那一筆
1374    有扣到的旗標 = 1
137E  旗標 0 → 結束
1387..13A8  清文字區（1255h 空字串）
13C5..141D  隊伍鏈上 +84h > 7Fh、+10Ch == 0、+85h > 0 的每一個：
            名字 + " takes and hides his share."（1256h），一人一行（`[bp-8]` 每次加 2）
1433..146A  "press <enter>/<return> to continue"（1272h），等一個鍵
```

- 錢**不進任何人的錢包**，就是從公款消失。
- 份數、每份都是位元組：公款 ÷ 總份數超過 255 時只留低 8 位元，所以大額的時候 NPC 拿得比
  「一份」少。
- 列名看的是整個 `+85h > 0`，份數看的是低三位元：`+85h = 08h` 的人會被列出來但沒拿錢。
- `+84h` 位元 7 是 ADD NPC 寫士氣時立的（overlay-03 `2EFDh..2F25h`，`addnpc.go`）；玩家建的角色
  是 0。`+85h` 在怪物檔裡：傭兵 ACOLYTE／WARRIOR 01h、SWORDSMAN 03h、HERO 84h、EVOKER／
  ROBBER／CURATE／THEURGIST 與 LEVEL 6 MU FFh；其他劇情 NPC 00h。overlay-16 `28CDh` 在
  建角那一側寫 `+85h = 1`，但那些記錄 `+84h` 是 0，不影響分錢。

remake：`gamepack.HideNPCShares`，在開戰利品選單之前（`openMonsterLoot` 與 `enterTreasure`）呼叫；
列名那一行放在狀態列。

## 還沒接（DRAFT）

| 項目 | 位址 | 等級 | 為什麼沒接 |
|---|---|---|---|
| NPC 列名的畫面與停頓 | overlay-05 `1387h..146Ah` | exact（碼） | remake 只在狀態列印同一句，沒有清畫面與等鍵那一頁；沒有對拍 |
| `DS:829Ah` 那一條路 | overlay-05 `0006h..0032h`、`0748h..07A7h` | exact（碼）；`829Ah` 由 overlay-07 `1AB9h` 設、`@6DE6`（`[4937h]+5CCh`）同時寫入，語意 unknown | entry 2 直接回 `[5CF0h]+73h × 100`，不走上面的三項；觸發它的 ECL 命令還沒讀 |
| 有資格分的人數 | overlay-05 `05B3h..05C9h`（`829Bh`） | exact（碼）；`+10Dh` 語意 unknown | 沿用 spec 097 的全隊都分；收據那一場 `829Bh` 為 0 |
| 同一個結果裡 `TREASURE` 與有怪物的 `COMBAT` | — | — | remake 先開戰利品選單（spec 036 的既有分派），原版是打完才一起結算；沒有找到走得到的實例 |

## 被推翻的斷言

| 原本寫的 | 現況 | 證據 |
|---|---|---|
| spec 140：「原版的委任只給錢」「空戰鬥的經驗總額是 0」 | 委任獎金經 entry 2 折成經驗值（貧民窟：2700，全隊分） | 上面的收據 |
| spec 097：「七種貨幣……那是戰利品，不是經驗值」 | 同一個迴圈加進公款的錢，在 `0224h` 起也折成經驗值 | `0224h..02B1h` |

## 測試

- `internal/gamepack/loot_experience_test.go`：收據的 2700、三種除法截去、寶石 ×250；物品
  加值 0／負數不算、+82 乘積繞成負數；NPC 份數的位元組截斷、倒下的 NPC 只佔一份、`+85h = 08h`
  列名不分錢。
- `cmd/pool-game/loot_experience_test.go`（從 `Update()` 送鍵）：兩隻狗頭人首領與法師打完，
  每人得（怪物 93 ＋ 公款與法師那支杖 +10 的 4000）÷ 4；收據那一隊五個戰士走真的職員交件，
  逐人得 594／594／594／540／540；帶 SWORDSMAN 交件，經驗值照 2700 ÷ 6 發，公款剩金 157、
  白金 32、珠寶 1，狀態列列出他。
- `cmd/pool-game/house_rule_test.go`：自訂規則關著時交件也有原版的 2700，開著另加 600。
