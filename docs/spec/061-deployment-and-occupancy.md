# Spec 061：戰鬥部署與佔用格重建

狀態：CONFORMED（部署驅動 `1A99h`、樣板填寫、兩邊的偏移與象限、逐人放置
`1609h` 的掃描順序、換陣型的鄰格、放不下就摘掉、五張 DS 表——dosgolem 三筆同狀態
收據，貧民窟三場固定事件全對）；
READY（佔用格重建、畫面相對座標、四個失敗出口、`DS:45BAh` 的寫入者、屍體格、
死亡後體型類別歸 0 的效果）；DRAFT（runtime `+13h` 的語意、奇數象限多跳一腿
的用意、overlay-16 entry 3 對放不下的怪物做了什麼、戰鬥中把體型類別寫成 0 的
是哪一支）。
日期：2026-09-02；2026-09-16 補部署驅動與逐人放置、兩筆 dosgolem 收據（#32）。

## 證據

| 檔案 | 內容 | 輸入 SHA-256 |
|---|---|---|
| `docs/audit/ida-overlay32-occupancy-rebuild.json` | overlay-32 `03A2h..04C0h`（122 條） | `efc22ba8…2dfc8` |
| `docs/audit/ida-overlay10-deployment.json` | overlay-10 `14CFh..1609h`（132 條） | `b929c704…399b` |
| `docs/audit/ida-overlay10-deployment-driver.json` | overlay-10 `1A99h..1EC3h`（398 條，部署驅動） | 同上 |
| `docs/audit/ida-overlay10-deployment-place-one.json` | overlay-10 `1609h..1A96h`（473 條，逐人放置） | 同上 |
| `docs/audit/ida-overlay10-deployment-bounds.json` | overlay-10 `149Eh..14CCh`（界限檢查） | 同上 |
| `docs/audit/ida-overlay10-45BA-references.json` | overlay-10 內 `45BAh` 與 `14CFh` 的全部運算元 | 同上 |
| `docs/audit/ida-start-deployment-tables.json` | `START.EXE` DS `2D0h..33Fh`（112 bytes，五張表） | `12811cbc…10d9f` |
| `docs/audit/dosgolem-deployment-peek.json` | dosgolem 跑到第一場遭遇按 COMBAT 那一幀讀的 `6A0Bh`／`45B2h`／`6772h`／`5E85h`／`6039h`，正對照 `2D0h` | 同上（`start.exe`） |
| `docs/audit/dosgolem-deployment-peek-goblins.json` | 五人隊在 (15,5) 突襲四隻哥布林：開打那一幀與之後每一幀位置表變動的快照 | 同上 |
| `docs/audit/dosgolem-deployment-peek-orc-home.json` | 五人隊撬門走到獸人的家 (3,3)：開打那一幀二十五筆、二十四隻放上二十隻，之後兩批走位 | 同上 |
| `docs/audit/dosgolem-deployment-peek-guards.json` | 同一隊走到衛兵攔截 (0,7)：三十七筆、三十四隻放上三十二隻，之後兩批走位 | 同上 |
| `docs/audit/dosgolem-deployment-peek-alarm.json` | 同一隊走到驚動衛兵 (3,11)：二十八筆、三十三隻放上二十三隻，之後兩批走位 | 同上 |

跨 overlay 的呼叫用 spec 109 的對照表反查：`013Dh:0048h` 是 overlay-32 entry 8
（`03A2h`，佔用格重建）、`010Ah:00BBh` 是 overlay-25 entry 31（`2419h`，兩邊
存活數，spec 062）、`00B6h:002Fh` 是 overlay-16 entry 3（`3266h`，未讀）。

## 佔用格重建：overlay-32 `03A2h`

1. `FillChar(DS:6039h, 4E2h, 0)`——整個佔用格陣列清零。清空的長度與戰術格
   數相同，這是 `6039h` 佔 50×25 的第二個獨立佐證（第一個是它的結尾正好接上
   `6517h` 的第一筆，見 spec 058）。
2. 走訪 combatant `1..DS:5E88h`，體型類別為 0 的跳過；其餘以體型展開四個佔格，
   每一格寫上該 combatant 的索引。
3. 再走訪一次，把每個 combatant 的座標減去戰術地圖 record 的 `+2`、`+3`，
   分別寫進 `DS:5FA8h` 與 `DS:5FF0h` 兩個以索引定址的 byte 陣列。

因此戰術地圖 record 的 `+2`／`+3` 是**畫面原點**：`5FA8h`／`5FF0h` 存的是每個
combatant 相對畫面的位置。spec 060 把 `+0`..`+5` 列為未定，其中這兩個由此閉合。
overlay-32 `0068h` 做的是同一件事的增量版本。

## 部署：overlay-10 `14CFh`

`function(formation, dy, dx, row, col, combatant: byte): boolean`，`retf 0Ch`。

1. 界限：`0 ≤ col ≤ 10`、`0 ≤ row ≤ 5`。
2. 查陣型樣板：索引為 `DS:45BAh × 108h + formation × 42h + row × 0Bh + col`，
   基底 `DS:43A2h`。該格為 0 就直接失敗。
   由此可讀出樣板的形狀：**每個陣型 11×6 ＝ 66 格，每個樣板組 4 個陣型**。
3. 算座標並寫進位置表：

   ```
   X = 22 + 6×dx + 5×dy + col
   Y = 10 +        5×dy + row
   ```

   與地圖建構器（spec 060）是同一個斜投影，只有 X 原點多一——樣板的第 0 欄
   對到建構器的 `subB` 1。
4. 以方向 8（原地、不位移）呼叫目的格探測 overlay-32 `0CB9h`，取得該
   combatant 佔的格有沒有人、以及最難進的類別。
5. 成立的條件是：沒有人佔著、類別不是 0、且該類別的 `EntryThreshold` 小於
   `0FFh`。任一不成立就失敗。
6. 成功時把樣板那一格清零——同一格不會被用第二次——並回傳 1。

## 陣型樣板不在檔案裡：由 `1A99h` 在每場開打前填

`DS:43A2h` 換算成 `START.EXE` 的檔案位移落在檔尾之外（DS 的檔案基底是
30640，`43A2h` 需要 47954，而檔案只有 47936 bytes），所以那一段是未初始化的
資料段，樣板由執行期填。用 IDA 讀那個範圍會得到整片 `FFh`，那是「沒有內容」
不是內容——不可當成樣板資料搬進 game pack。

填寫者是同一顆 overlay 的部署驅動 `1A99h`（`1B93h..1C9Ah`）。它只用一張 DS 表
`304h` 就把 2 組 × 4 個陣型 × 6 列 × 11 欄全部填完：

```
for side in 0..1:                          ; DS:45BAh 就是這個迴圈變數
  for formation in 0..3:
    F = (formation == 1) ? 4 : DS:45B8h[side]    ; 象限，見下
    for row in 0..5, col in 0..10:
      lo = DS:[304h + F×0Ch + row×2], hi = DS:[305h + F×0Ch + row×2]
      template[side][formation][row][col] = (lo ≤ col ≤ hi) ? 1 : 0
```

`DS:304h` 五組列範圍（`ida-start-deployment-tables.json`，每列 `lo hi`）：

| F | row 0 | row 1 | row 2 | row 3 | row 4 | row 5 | 形狀 |
|---|---|---|---|---|---|---|---|
| 0（面向 N／NE） | — | — | — | 2..9 | 3..10 | 4..10 | 下三列 |
| 1（面向 E／SE） | 0..2 | 0..3 | 1..4 | 2..5 | 3..6 | 4..7 | 左側斜帶 |
| 2（面向 S／SW） | 0..6 | 0..7 | 1..8 | — | — | — | 上三列 |
| 3（面向 W／NW） | 3..6 | 4..7 | 5..8 | 6..9 | 7..10 | 8..10 | 右側斜帶 |
| 4（陣型 1 專用） | 0..6 | 0..7 | 1..8 | 2..9 | 3..10 | 4..10 | 整塊 |

「—」是 `lo=1, hi=0`，整列為 0。所以樣板組數是 **2（兩邊）**，`DS:45BAh` 不是
「選用條件」而是**當下在放哪一邊**：驅動每走到一筆記錄就把記錄 `+10Eh`
寫進 `45BAh`（`1CEEh`），`14CFh` 與 `1609h` 讀它決定用哪一組樣板與哪一組偏移。
`45BAh` 在整個 overlay-10 裡只有 `1A99h` 寫（`1B62h`／`1B69h`／`1B93h`／`1B9Ah`
是填樣板的迴圈計數，`1CEEh` 是逐筆的陣營）。

## 部署驅動：overlay-10 `1A99h`（exact）

`retf`，無參數。順序：

1. overlay-25 entry 31（`2419h`）重數兩邊存活數 `DS:6772h`／`6773h`（spec 062）
   ——這就是 `6772h` 在部署前的寫入者。
2. `DS:[5E88h + 4i] = 0`，`i = 1..0FEh`：整張 combatant 表的體型類別清零。
3. overlay-32 entry 8（`03A2h`）先重建一次佔用格（此時全空）。
4. 兩邊的地城格偏移與象限，全部從隊伍朝向 `DS:6A0Dh` 與遭遇距離
   `[4937h]+582h`（ECL `@6DC1`，spec 078／136）算：

   ```
   45B2h = 0, 45B4h = 0                               ; 我方 dx, dy
   45B3h = 0 + 距離 × DS:274Ah[朝向]                   ; 敵方 dx
   45B5h = 0 + 距離 × DS:2753h[朝向]                   ; 敵方 dy
   45B8h = 朝向 ÷ 2                                    ; 我方象限 0..3
   45B9h = ((朝向 + 4) mod 8) ÷ 2                      ; 敵方象限（反向）
   45B6h[side] = (6772h[side] + 1) ÷ 2                 ; 每邊第一腿的上限
   ```

   乘法走 `5BBh:279h`（RTL 的長整數乘法），結果只取低位元組。
5. 填樣板（上一節）。
6. `DS:5E88h = 1`，沿 `DS:5CF4h` 串列（`+104h` 是 next）逐筆：
   - `6517h[i]` = 記錄遠指標；`45BAh` = 記錄 `+10Eh`；`5E87h+4i` = i；
     `5E88h+4i` = 記錄 `+6Ch AND 7`（體型類別）。
   - 呼叫 `1609h(i)`。**放上了**：記錄 `+10Dh` 為 0（不在場——倒地或死亡的
     隊員一樣會被擺上去）時把體型類別改回 0，並在 `DS:829Ah == 0` 且
     runtime（`+108h`）`+13h == 0` 時登記成屍體：`6673h` 加一，
     `663Ah[7n]` 存該格原地形、該格地形寫 `1Fh`、`6634h[7n]` 存記錄遠指標、
     `6638h`／`6639h` 存 X／Y（這張表就是 spec 121 收雲時掃的「板上物件表」）。
     然後再重建一次佔用格、i 加一、`5E88h` 加一。
   - **放不下**：體型類別改 0。runtime `+13h == 1` 的把 `6517h[i]` 清成 nil、
     `DS:5CF0h` = 記錄、呼叫 overlay-16 entry 3 `(0, 1)`，並把走訪指標退回前
     一筆——也就是這一筆從戰鬥裡摘掉，索引不前進；`+13h ≠ 1` 的留在表上、
     索引照加，只是體型 0（不佔格、不畫）。
7. 收尾把最後一筆的 `+104h` 清成 nil。

因此原版的部署**先我方後敵方不是規則**，順序是 `5CF4h` 串列的順序；每一筆
在 `1609h` 之前都會先重建佔用格，所以後放的人看得到先放的人。

## 逐人放置：overlay-10 `1609h`（exact）

`function(combatant: byte): boolean`，`retf 2`。回傳 0 只代表「四個陣型都
放不下」；成功放上與「還沒放上就走完」都回 1，放上與否要看它有沒有呼叫到
`14CFh` 成功（驅動用 `[bp-2]` 判斷）。

用到的四張表（`ida-start-deployment-tables.json`）：

| 表 | 內容 | 值 |
|---|---|---|
| `DS:2D0h` | `[象限×4 + 陣型]` 陣型 1..3 相對原格的地城方向（欄 0 是 8＝不用） | q0 `8 4 6 2`、q1 `8 6 4 0`、q2 `8 0 6 2`、q3 `8 2 0 4` |
| `DS:2E0h` | `[象限×4 + 陣型]` 掃描軸 k（取值 ÷ 2） | q0 `0 0 2 6`、q1 `2 2 0 4`、q2 `4 4 2 6`、q3 `6 6 4 0` |
| `DS:2F0h` | `[k']` 掃描方向環，值是 `274Ah`／`2753h` 的方向碼 | `7 2 3 6`（NW、E、SE、W） |
| `DS:2F4h`／`2FCh` | `[(陣型>0)×4 + k]` 掃描原點的 col／row | col `5 4 5 6 / 3 8 7 2`；row `3 2 2 3 / 0 2 5 3` |

流程（區域變數用名字寫）：

```
side = 45BAh; q = 45B8h[side]; dx = 45B2h[side]; dy = 45B4h[side]
formation = 0; leg = 0; firstLeg = 1; giveUp = 0; placed = 0; state = 1
loop:
  k = 2E0h[q×4 + formation] ÷ 2
  state 1: d = 2F0h[(k+2) mod 4]                ; 腿與腿之間的位移方向
           col0 = 2F4h[(formation>0)×4 + k] + 274Ah[d] × leg
           row0 = 2FCh[(formation>0)×4 + k] + 2753h[d] × leg
           (col,row) = (col0,row0); len = 1; n = 1; state = 2
  state 2: d = 2F0h[(k+1) mod 4]; (col,row) = (col0,row0) + 單位向量(d) × len
           n += 1; state = 3
  state 3: d = 2F0h[(k+3) mod 4]; (col,row) = (col0,row0) + 單位向量(d) × len
           n += 1; len += 1; state = 2
  anyOut  = col ∉ 0..10 或 row ∉ 0..5            ; [bp-5]
  bothOut = col ∉ 0..10 且 row ∉ 0..5            ; 149Eh 回 1 的條件
  if state > 1:
    anyOut 且非 bothOut           → 換腿（掃到列尾）
    firstLeg 且 n ≥ 45B6h[side]  → 換腿
    非 firstLeg 且 n > 0Bh        → 換腿
    換腿 = leg += 1; state = 1; firstLeg = 0;
           另外 side == 0、q 為奇數、formation == 0、leg == 1 時：
           對原地城格 (6A0Bh+dx, 6A0Ch+dy) 用 2D0h[q×4 + 1..3] 三個方向各問一次
           01BAh（spec 060 的牆面查詢）；在室外（495Bh > 1）或任一方向沒有牆
           → leg 再加一
  if anyOut 且非 bothOut: 這一格不試，回到 loop
  if bothOut:
    placed = 0; state = 0
    while formation < 3 且 state ≠ 1:
      formation += 1; d = 2D0h[q×4 + formation]
      室內（495Bh ≤ 1）且 01BAh(原地城格, d) 有牆 → 這個陣型跳過
      否則 dx = 45B2h[side] + 274Ah[d]; dy = 45B4h[side] + 2753h[d]
           leg = 0; state = 1
    state ≠ 1 → giveUp = 1
  else placed = 14CFh(formation, dy, dx, row, col, combatant)
  placed == 0 且 giveUp == 0 → loop
return !giveUp
```

兩道界限檢查長得像、答案不一樣：`[bp-5]` 是任一座標出界，`149Eh`
（`14A4h..14C2h`）是 `col ∉ 0..10` **且** `row ∉ 0..5` 才回 1。一列掃到
`col = 11` 或 `−1` 時只有 col 出界，走的是換腿；腿的原點沿垂直方向一路移出
樣板（例如面向 N 的 leg 6 原點 (11, 9)）兩個都出界，才換陣型。

讀出來的形狀：

- **掃描是「一腿一列，從中間向兩側交替」**：`2F0h` 的 (k+1) 與 (k+3) 是一對
  反方向，候選格依序在原點、+1、−1、+2、−2……；(k+2) 是與它垂直的方向，
  每換一腿原點就往那個方向移一格。面向 N（q0）時原點 (5,3)、沿 E／W 掃、
  換腿往 SE，三腿正好對上樣板 F0 的三列。
- **第一腿的候選數是 `⌈n÷2⌉`**：`n` 從原點算 1，數到上限就把下一圈改成新的
  腿，但**這一格照試**（換腿之後仍走到 `1A4Ch`）。六人隊伍上限 3，第一列放
  原點、右一、左一三個人，其餘在第二列從中間往外排；單人隊上限 1，原點那一
  格試完就換腿。dosgolem 收據（`docs/audit/dosgolem-deployment-peek.json`）：
  單人隊面向 W、距離 0、雙方各一人，原版 `5E85h` 給隊員 (28, 13)、敵人
  (26, 12)，`45B2h..45B9h = 00 00 00 00 01 01 03 01`，`6039h` 只有那兩格非零；
  `combat.PlaceCombatant` 同狀態算出同一組（`TestPlaceCombatantMatchesTheDosgolemReceipt`）。
- **五人隊收據**（`docs/audit/dosgolem-deployment-peek-goblins.json`）：五個預設擲骰的
  矮人戰士在 (15,5) 面向 E 突襲四隻哥布林，距離 0，`45B2h..45BAh =
  00 00 00 00 03 02 01 03 01`；`5E85h`：隊員 (26,12)(27,13)(25,11)(24,12)(25,13)、
  哥布林 (28,13)(27,12)(29,13)(28,12)。第四個隊員落在 (24,12) 就是「奇數象限第一腿
  走完再多跳一腿」那一條（(15,5) 西／南／北任一沒有牆）；哥布林上限 2 的換腿也
  對上。remake 從隊伍、GEO、staging 一路走到 `deployRoster`，九格逐格相同
  （`TestDeploymentMatchesTheGoblinReceipt`）。
- **獸人的家收據**（`docs/audit/dosgolem-deployment-peek-orc-home.json`）：同一隊在 (3,3)
  面向 N 對二十四隻獸人，`45B2h..45BAh = 00 00 00 00 03 0C 00 02 01`，原版放上二十隻、
  摘掉四隻；remake 同狀態二十五格逐格相同、摘掉的也是四隻
  ——這一筆把「放不下就摘掉」與換陣型的鄰格掃描一起對上了。衛兵攔截（三十七筆、
  摘掉兩隻）與驚動衛兵（二十八筆、摘掉十隻）同法拍到，同狀態逐格相同；驚動衛兵第 17
  隻在原版第一次等輸入前已先攻走一格（收據標 `moved_before_first_prompt`）。三場一起
  釘在 `TestDeploymentMatchesTheSlumsReceipts`。
- 同一筆收據往後三輪：死掉的哥布林與隊員在 `5E85h` 的體型類別變成 0、從 `6039h`
  消失，而活著的哥布林**會走進死掉隊員那一格**（(26,12)）——死者的格子在戰鬥中
  不擋路；寫 0 的是哪一支仍沒讀。
- **陣型 1..3 是原地城格的三個鄰格**（`2D0h`：面向 N 時依序 S、W、E），每一個
  都從 `45B2h`／`45B4h` 重算，不累積；室內有牆的鄰格跳過。三個鄰格都放不下才
  回 0，驅動就把這一筆摘掉或留成體型 0。
- 每次呼叫都從 formation 0、leg 0 重新掃，靠 `14CFh` 清掉用過的樣板格與佔用
  格重建讓後面的人往後排。

## 佔用格重建對狀態的處理

`03A2h` 只看 `5E88h + 4i` 的體型類別，**不讀 `+10Ch`**（本規格前段，exact）。
狀態的效果全在誰把類別寫成 0：部署時是 `1A99h`（`+10Dh == 0`，exact）；
戰鬥中死亡的人離開佔用格是量到的（五人隊收據：死掉的哥布林 8 與隊員 1、2
體型類別變 0、`6039h` 沒有它們），寫入者應是把 `+10Dh` 清掉的那條路徑
（spec 062／084），**那一支還沒讀**——overlay-10 之外沒有掃過 `5E88h` 的寫入。昏迷（4）與睡著的（`+10Ch` 不變、`+10Dh` 仍非 0）在部署與
重建裡沒有任何特殊處理，照常佔格（exact：兩支函式都不讀 `+10Ch`）。屍體在
部署階段以地形 `1Fh` 留在格上，佔用格陣列裡沒有它。

## READY 契約

1. 佔用格重建要先清空整個陣列再重填，不可增量更新——原版每次都是整份重建。
2. 佔格展開用 combatant 自己的體型類別，四個槽都要跑。
3. 畫面相對座標是位元組減法，會繞回；不得改成有號數或先做界限裁切。
4. 部署的判定順序是樣板、佔用、類別、`EntryThreshold`，任一不成立即失敗；
   成功後才消耗樣板格。
5. 部署與地圖建構器共用同一個斜投影，X 原點差一；兩處不可各自寫死不同的常數。
6. 陣型樣板是執行期狀態，不得從 `START.EXE` 讀出來當資料；要照 `1A99h` 用
   `DS:304h` 五組列範圍在開打前填。
7. 兩邊的地城格偏移由朝向與遭遇距離算（我方 0、敵方 距離 × 朝向單位向量），
   象限由朝向 ÷ 2 取；不得用固定偏移。
8. 逐人放置照 `1609h` 的掃描順序（原點、±1、±2……、換腿、三個鄰格），第一腿
   上限 `⌈n÷2⌉`（上限那一格照試），一列掃到出界就換腿，腿走到兩個座標都出界
   才換陣型；`5CF4h` 的串列順序就是放置順序。
9. 放不下的人體型 0 留在表上（或依 runtime `+13h` 摘掉），不得自行擴大範圍。

## remake 的實作

`combat.DeploymentSides`／`FillDeploymentTemplates`／`PlaceCombatant`
（`internal/combat/deployment_original.go`）照 `1A99h`／`1609h` 重建，五張表
逐位元組對 `ida-start-deployment-tables.json`
（`TestDeploymentTablesMatchTheDataSegment`）；`cmd/pool-game/tactical.go` 的
`deployRoster` 把隊伍、倒戈的 NPC、怪物依序丟給它，遭遇距離讀 ECL `@6DC1`
（遭遇選單每次寫距離都鏡射到那一格，spec 078）。

與原版仍有的差（都是 remake 的簡化，不是原版規則）：

- 昏迷／倒地／死亡的隊員原版一樣擺上去、再改成體型 0 並登記屍體（地形
  `1Fh`），remake 直接不擺。
- 放置順序原版是 `5CF4h` 串列的順序，remake 固定是隊伍 → 倒戈 NPC → 怪物。
- 遭遇距離：原版 `0489h` 會讓怪物在地圖上先走最多兩步、撞到就停，距離是
  走到的格數（dosgolem 那一場是 0）；remake 沒有地圖上的怪物群，固定給 2
  再夾上限（spec 078）。同一場的敵方偏移因此可能差兩格，那是 spec 078 的缺口。
- 放不下的怪物原版從串列摘掉（overlay-16 entry 3），remake 直接不擺，效果相同。

之前那一版固定偏移（隊伍 −1、敵方 ＋2）加「敵方只擺在與隊伍連通的格子」
的護欄已經整段拿掉；連通護欄當初擋的是 GEO4 block 21 那種雙方被地形隔開、
打不完的場面，換成原版演算法之後敵方本來就擺在隊伍朝向前方 `距離` 格的
那一個地城格，同一條走廊上。那一場（GEO4/21 (8,5) 朝西，五十隻）重跑過：主線探針
第 7 回合分出勝負，探索器十個種子僵局安全閥 0 次（playtest 補八 8e）。
