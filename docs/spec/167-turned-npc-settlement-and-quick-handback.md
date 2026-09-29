# Spec 167：倒戈隊員的戰後結算、AI 代打中途交還

狀態：READY（兩節的碼逐條讀過位元組並實作，從 `Update()` 送鍵測試；沒有原版執行期收據）。
證據等級 exact，另註者除外。日期：2026-09-29。主台帳：GitHub issue #122、#123。
接手 spec 162〈未閉合〉、spec 150〈還沒接〉的 `1164h` 那一列、spec 096 與 spec 139 的 `+3 == 14h`。

## 輸入

- DOS ZIP `Pool of Radiance (1988).zip`，SHA-256 `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`；
  `poolrad/start.exe` `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`。
- overlay 取自 `workplace/ovr/overlay-NN.bin`（對 `docs/audit/dos-ovr-manifest.json` 的 `code_sha256`），
  位址是 overlay 檔內位移；反組譯 `coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel`
  （overlay-09 `07E8h` 起要以 `--start-address` 重新對齊）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 05 | `900ea1b8e57b03e0f6dd1c16024a686c8474e2b0ae9674c730d5619679809c16` | — |
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` | — |
| 09 | `6b47e49d1a2e73427b9863560350965d6d5b43d3dd89df89d9b7f43a68409258` | `0058h`（(`930h` − `3B0h`) ÷ 16）|

- far call 對照：`00B6h:002Fh` = overlay-16 entry 3（`3266h`）；`010Ah:00CAh` = overlay-25 entry 34（`266Dh`）；
  `0198h:002Fh` 是逐行印字（spec 108）；`013Dh:xxxx` 是 overlay-32（重畫）。

## 一、倒戈的隊員在戰後（#122）

戰鬥中的 `DS:5CF4h` 串列就是隊伍串列接上怪物：隊員在前（runtime `+13h == 0`），跟著的非隊員與
怪物在後。Attack Ally 答 Y 倒戈的 NPC（spec 162 `2A0Ch` 寫記錄 `+10Eh = 1`）、ADD NPC 編號 18h
帶進來的那一位（spec 091）都還在隊伍那一段。戰後 overlay-05 `14CAh` 的三段對它們一律只看 `+10Eh`：

### entry 2（`0000h`）：當敵方算

```
0068  26 80 BD 0E 01 01 / 74 03   +10Eh == 1 才往下
0079  26 80 BD 0C 01 03 / 75 03   +10Ch == 3（逃掉）→ 整段跳過
0089..00BE  七種錢 +88h + 2i 加進 DS:6752h + 4i（公款）
00C0..00F4  經驗值 += +B8h（word）+ +BAh × +B1h
00F7  [4937h]+5C6h == 1 → 物品跳過；否則 0106h 沿 +C8h 逐件：
0114  +3Ah > 0 收；否則 [bp-12h] >= 8 不收、Roll(1,10) <= 3 才收（[bp-12h] 在 0049h 只清一次，隊員與怪物共用）
0136..01FB  GetMem(3Fh)、Move、+34h = 0、插在 DS:676Eh 串列頭
```

錢與物品是**複製**：記錄上的不動，但下一段就把記錄摘掉。

### entry 3（`033Ah`）與除數

entry 3 發給 `+10Dh != 0` 而且 `+10Ch != 1` 的每一位（`035Ah`、`0368h`），不看陣營；除數是
`[4937h]+67Ch − DS:829Bh`，`829Bh` 數隊伍那一段 `+10Dh == 0` 或狀態 1 的（`05B3h..05C9h`）。
倒下的倒戈者 `+10Dh` 是 0：不分、不佔除數，與倒下的隊員同一條（spec 150）。

### `1164h`：摘掉

```
119E  26 80 BD 0E 01 01 / 75 5B   +10Eh == 1（或 runtime +13h == 1）的記錄
11A9  +10Dh != 1 → [4937h]+590h（@6DC8）加一
11EF  9A 2F 00 B6 00              overlay-16 entry 3(runtime +13h, 1)：+13h 是 0 → 從隊伍鏈摘掉、隊伍人數減一
1244  DS:5CF0h = DS:5CF4h
```

`14CAh` 的順序是 `04ADh`（entry 2／3）→ `1164h` → `14EDh` 全滅判定 → `1295h` NPC 分錢。所以
**倒戈的 NPC 打倒之後像一隻怪物那樣結算，然後離隊；分錢那一頁沒有他。** spec 162 原本寫的
「下一場照 Side 擺到對面」不成立。

### 隊員全倒或逃光、倒戈者還站著（`04ADh`）

`052Eh..05AEh` 對隊伍那一段狀態 0／1 的任何一位立 `82A0h`、清 `439Dh`，不看陣營；`4960h`（全滅）
只有 `+10Eh == 0`、`+84h < 80h`、狀態 0／1／3 的才清（`054Eh..058Eh`）。所以隊員有人逃掉、其餘
倒下、倒戈的 NPC 還站著時：`439Dh == 0` 走狀態換算（逃掉的 3 → 0，`0636h`）而不是留人
（`0688h`），entry 2／3 照跑，`14EDh` 不是全滅，`08E0h` 印打贏。

### remake

`cmd/pool-game/opposing_members.go`：

- `opposingMembers`：隊伍裡 `Side != 0` 的人（競技場複製品除外：那一場 entry 2 在 `0006h` 整段跳過，
  spec 150），狀態讀盤面。
- `opposingMembersExperience`：`+B8h + +BAh × +B1h`，NPC 讀自己的記錄，玩家建的角色讀 DOS 匯出那一份；
  逃掉的不算。`finishCombat` 加進 entry 2 的總額。
- `collectOpposingMembers`：錢包（`Character.Money`，DOS 匯出寫進 `+88h` 的就是它）加進公款、物品走與
  怪物同一支 `takeLootItem`，排在怪物之前（串列順序）。
- `removeOpposingMembers`：`1164h`。`finishCombat` 在發完經驗值之後、`openMonsterLoot`（`1295h`）
  之前呼叫；僵局、逃走與全滅三條也呼叫。
- `opposingMemberHoldsTheField`：上一小節的分岔，成立時 `finishCombat` 把結果當成打贏。
- `stagedMonsterCopy` 不再把隊員那一格對到怪物：倒戈者 `friendly` 是 false，以前會被數成前一隻
  怪物的複本，多收一份錢與物品。

測試 `cmd/pool-game/opposing_members_test.go`（貧民窟 ECL2 block 20，四名戰士加 SWORDSMAN
MON3CHA block 36）：

- `TestTurnedNPCIsSettledLikeAFoeAndLeaves`：走進 SWORDSMAN 那一格、答 Y、一擊打倒；收場後他不在
  隊伍裡、沒有 NPC 分錢頁、公款多了他的 77 金、他那件價值非 0 的物品在戰利品裡、每份經驗值含他的
  89（35 + 3 × 18）。
- `TestTurnedNPCStandingTurnsAFleeIntoTheWinningBranch`：一人逃掉、三人倒下、倒戈者站著：不是全滅、
  倒下的留在隊伍裡、逃掉的換回 0、@6DC7 不是 81h。

## 二、AI 代打中途交還（#123）

### 問鍵的三個點

overlay-09 entry 7（`0FC8h`）是 AI 回合裡唯一的問鍵：

```
0FD2  9A FA 02 12 05     有鍵？沒有就 10B2h
0FDE  9A 48 00 6B 02     讀一個鍵（0 再讀一次）
0FF7  3C 32              '2'：DS:6D23h 反相、印 Magic On／Off（spec 139）
103E  3C 20              SPACE：
1056..1085                 串列上 +84h < 80h 而且 +10Ch != 1 的記錄 +10Fh = 0
1087  26 80 BD 0F 01 00    這一位的 +10Fh 還是 0：
109A  26 C6 45 03 14         runtime +3 = 14h
109F  C6 46 FF 01            回 1
10A5  3C 2D              '-'：0096h:00B6h（沒讀）
10B2  9A F6 00 6B 02     清鍵盤緩衝區
```

呼叫端只有 overlay-09 自己的三處：

| 位置 | 時機 | 回 1 之後 |
|---|---|---|
| entry 1 `001Ch` | 回合開頭 | 戰術模式照擲（`0045h..00B1h`，寫 runtime `+15h`），`00B5h` 跳過士氣，`0106h` 跳到 `01FDh` 返回 |
| entry 1 `01ACh` | 換完武器（`13D5h`）之後、挑目標與 entry 5 之前 | `01B6h` 不進迴圈，返回 |
| `07E8h` `0843h` | 每一次走一步之前（接近與逃跑都經過；在 `084Dh` 腳程那一道之前） | `0848h` 跳 `0B36h` 返回；entry 5 `0BD8h` 看到 `+3 == 14h` 收工、回 1，entry 1 迴圈結束 |

三條都不叫 entry 34，所以 runtime `+3` 留在 14h。

### 為什麼交還的是同一位

- 先攻選取 overlay-08 `0124h..01E1h`：沿串列取 `+3` 最大的，同分以 `Roll(1,100)` 決定。分數上限是 14h
  （spec 062 `00E4h`），所以下一個被選的就是剛交還的那一位（除非另一位也是 14h）。
- overlay-08 entry 3 `0231h..0248h`：`+3 == 14h` 就改成 13h；`01F2h..020Ch` 照常清 runtime `+0Fh`／`+12h`／`+07h`；
  `02A0h` 看 `+10Fh`，已經是 0，走玩家指令迴圈 `0307h`。
- 腳程（runtime `+6`）這中間沒人動：走過的扣掉了，剩下的歸玩家。

overlay-08 `0525h` 的另一個 `+3 = 14h` 是指令迴圈的鍵 `10h`：目前這一位 `+3 = 14h`、全串列 `120Eh`
（Q）、停 200、結束——整隊交給電腦，同一位立刻由 AI 重走一次。remake 沒有這個鍵（停止線外，不影響本節）。

### remake

`cmd/pool-game/quick_handback.go`：

- `handableMover`：SPACE 收得回來的行動者（隊員、不是 NPC、`+84h < 80h`）而且現在由 AI 走。
- `runFoeTurn`：取代 AI 分派與 Q 那兩處的 `foeTurn`。收得回來的行動者、遊戲速度不是 0 時，foeTurn 在
  goroutine 裡跑，每到 `foeCheckpoint`（`01ACh`、接近迴圈與 `foeFleeStep` 的 `0843h`）停一個影格；
  主迴圈下一個影格以 `foeKeyCheck`（entry 7）帶那一影格的鍵回去。兩個不帶緩衝的 channel 交棒，同一時間
  只有一邊碰狀態。其餘行動者照舊同一個影格跑完——SPACE 收不回它們，entry 7 對它們一定回 0。
- 回合開頭（`001Ch`）就是原本 AI 分派前的 SPACE 檢查：收完這一位不再由 AI 走就 `handBackAtTurnStart`
  （擲戰術模式、寫回）。
- `handBackFoeTurn`：戰術模式寫回、步數記帳、分數 14h、`selectActor` 重選；`selectActor` 挑到 14h 的
  改 13h（entry 3 `0248h`）。
- 停拍時（`holdCombatNotice`）按下的 SPACE 記一格（`pendingQuickRelease`），下一次問鍵讀到——原版停拍不讀鍵，
  但鍵留在 BIOS 緩衝區。
- spec 096 表格那一列（「`+3 == 14h` 時 entry 5 直接收工」）因此照原版：交還的那一步直接收工。

速度 0 時不跨影格：原版那時的 Delay 是 0（`speedDelayTicks`），整個回合在兩次按鍵之間就跑完，只有回合
開頭那一次問得到新鍵（strong inference：緩衝區裡的鍵原版也會在後面的點讀到，remake 在開頭就處理掉，
效果相同）。

測試 `cmd/pool-game/quick_handback_test.go`：

- `TestSpaceTakesTheQuickCharacterBackMidWalk`：Q 之後停在 `01ACh`；放行兩個影格走了一步；SPACE：
  Quick 與 AI 都清掉、行動者還是他、分數 13h、位置與腳程不動、M 進得了移動。
- `TestSpaceAtTheStartOfAQuickTurnHandsItBack`：輪到代打的隊員時按 SPACE：同一位由玩家走、分數 13h、
  戰術模式照擲寫回、沒有移動。

## 未閉合

| 項目 | 等級 | 為什麼沒做 |
|---|---|---|
| 每一步之間停多久 | unknown | overlay-13 entry 5 的重畫（`27Fh:0000h`）沒讀；remake 停一個影格。停拍長度，見停止線 |
| entry 7 的 `-` 鍵（`0096h:00B6h`） | unknown | 沒讀；指令迴圈那一條（`056Eh`）remake 也沒有 |
| `1164h` 的 `@6DC8` 計數 | exact（碼） | 本份只接倒戈者；`@6DC8` 每一場都寫（怪物也算），貧民窟 `B13Eh` 讀它（spec 136），不屬 #122 |
