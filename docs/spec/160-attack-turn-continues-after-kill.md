# Spec 160：殺了目標、還有剩的攻擊次數時回合繼續（overlay-13 `1404h`／entry 8、overlay-08／09 的迴圈）

狀態：READY（攻擊核心的扣次數與完成旗標、玩家指令迴圈與移動迴圈、電腦 entry 1／entry 5 的再進、
entry 8 的條件寫回都已逐段讀出並實作，送鍵與 foeTurn 測試加變異檢查）。沒有原版執行期收據逐下
對過「殺了之後續打」，證據是位元組（exact）。日期：2026-09-27。主台帳：GitHub issue #109。

## 輸入

- `START.EXE` SHA-256 `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`；overlay 由
  `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` | `0051h` |
| 09 | `6b47e49d1a2e73427b9863560350965d6d5b43d3dd89df89d9b7f43a68409258` | `0058h` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` | `0096h` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` | `010Ah` |

- 反組譯：`coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel`，overlay-local offset、
  base 0。far call 的 segment 以 (manifest `executable_file_offset` − 3B0h) ÷ 16 反查，stub offset
  `20h + i × 5` 是 entry i：`010Ah:00CAh` = overlay-25 entry 34（結束行動）、`0096h:006Bh` = overlay-13
  entry 15（攻擊包裝 `1883h`）、`0096h:0048h` = overlay-13 entry 8（`0D29h`）、`0096h:00A7h` = overlay-13
  entry 27（`37B8h` 挑目標）。

## 每回合初始化（overlay-13 entry 1 `0000h`，exact）

```
002E  26 C6 45 08 00        runtime +8 = 0
0036  26 C6 45 04 02        runtime +4 = 2（攻擊區段從第二形態起）
0042  E8 E4 0C              entry 8：+113h = 第一形態這一回合的次數（射擊另算，spec 151）
0045..006E                  +114h = e58h(群組 18 之後的 +0A2h)
```

## 攻擊核心（overlay-13 `1404h`，exact）

```
1413  完成旗標 = 0
1440  26 C6 45 08 01        runtime +8 = 1（這一回合出過手）
1666  形態 = runtime +4，由它倒數到 1：
1685    +112h[形態] == 0 → 下一形態
1690    目標已倒下（[bp-12h]）→ 下一形態
16A3    26 FE 8D 12 01      +112h[形態] −= 1，runtime +4 = 形態
16D2..1732                  命中、傷害（spec 050／051／153）
1755    26 80 BD 0D 01 00   目標 +10Dh == 0 → [bp-12h] = 1（停手）
1799    完成旗標 = 1
17A0..17C2                  形態 1、2 任一格 +112h[形態] > 0 → 完成旗標 = 0
17D5..17EC                  完成旗標非 0 才呼叫 entry 34（`9A CA 00 0A 01`），回傳值寫回完成旗標
```

所以一次出手打完每一格剩下的次數，除非目標先倒下；目標倒下而還有剩的，完成旗標是 0、entry 34
不叫，行動者的 runtime `+3`（先攻）與 `+6`（腳程）都還在。攻擊包裝 `1883h` 在進核心之前把攻擊者的
runtime `+0Ah` 寫成這一次的目標（`1929h`：`26 89 45 0A`）。

## 玩家（overlay-08，exact）

- 指令迴圈 `036Ch..05C8h`：`05A6h` 看結果 `[bp-2]`，為 0 就重畫指令列回到 `036Ch`。瞄準的 `T`
  （overlay-13 `36CDh` → `2C17h` → `1883h`）把完成旗標寫進 `[bp-2]`，所以殺了目標、還有剩的，
  指令列回來，可以再按 A 瞄下一個。
- 移動 `09C3h`：迴圈 `0A13h` 只看 runtime `+6 > 1` 與按鍵，不看完成旗標。撞上去（`0BF1h` →
  `0D30h` → `0DDDh` 攻擊包裝）之後回到 `0A13h`；entry 34 沒被叫到、腳程沒被清掉，就接著走、接著撞。
- 選單回來之後 `03B1h`／`03FDh` 叫 entry 8（見下）。

remake：`resolveWeaponAttack`（missile.go）打完把 `state.attackGoesOn` 設成「還有剩」；
`resolveAimedAttack`（cast.go）與撞上去的那一支（tactical.go）看它，成立就不呼叫
`endTurnAfterAction`。

## 電腦（overlay-09，exact）

- entry 5 `0F1Bh` 攻擊包裝回來：完成旗標非 0 → `[bp-4] = 0`；否則目標 `+10Dh == 0` → `[bp-2] = 1`。
  兩者都離開接近迴圈 `0B7Eh`；`0F3Eh` 回傳 `[bp-4] == 0`。所以「殺了還有剩」回傳 0。
- `0DFCh`（射擊武器、不能近戰、身邊有敵人）叫 entry 9 換武器後 `0DFFh` 只立 `[bp-2]`，`[bp-4]`
  還是 1，同樣回傳 0。
- entry 1 `01B2h..01FBh`：結果為 0 就再叫 `37B8h`（`0096h:00A7h`，挑目標），挑得到而且 runtime
  `+3 > 0` 就再進 entry 5（`01E6h`）；挑不到走 `0F56h` 結束。entry 9 與士氣、施法那幾段在
  `019Bh` 之前，不重跑。
- entry 5 開場 `0B42h..0B4Ch` 把 `439Eh`（上一步的方向）寫 8、`439Fh`（卡住）寫 0；戰術模式在
  runtime `+15h`，沿用。

remake：foeTurn 的接近段落前面放 `foeEntry` 標籤，殺了還有剩（或 `0DFCh` 換了武器）就
`goto foeEntry`：目標照 `37B8h` 重挑（攻擊包裝把 `+0Ah` 寫成已倒下的那一個，所以一定重挑），
清掉方向與卡住，逃跑的照樣先跑逃跑迴圈。原版這個迴圈沒有上限；`0DFCh` 在手上的射擊武器被
詛咒時會一直轉，remake 以 `foeMaxEntriesPerTurn`（8）收住。

## entry 8 的條件寫回（overlay-13 `0D29h..0E52h`，exact）

```
0D36  [bp-2] = +113h（舊）
0D49  +113h = +0A1h
0DC8  [bp-1] = 新次數（e58h；射擊再以彈藥封頂，spec 151）
0E09  runtime +8 == 0 → 寫回 +113h = 新
0E18  新 < 舊 → 寫回新
0E34  新 ≥ 舊 × 2 → 不寫（+113h 留著 0D49h 的 +0A1h）
0E41  射擊（[bp-5]，entry 43 且 entry 45）→ 不寫；否則寫回新
```

`+114h` 不在這一支裡。runtime `+8` 只有 `1440h` 寫 1、`002Eh` 寫 0，所以條件寫回只在「這一回合
出過手之後又叫 entry 8」時有作用：玩家的物品選單回來之後、電腦 `0DFCh` 換武器之後（entry 9 的
`1813h`）。「不寫」留下的是次數編碼 `+0A1h` 而不是舊的剩餘次數——例如一回合 3/2 次（編碼 3）的
戰士在兩下的相位一下殺了目標（剩 1），開物品選單再關掉：新 2 ≥ 1 × 2，`+113h` 變成 3，這一回合
還能再打三下。位元組就是這樣寫的，remake 照做。

remake：`recountSwings`（attack_continue.go）。玩家在物品選單收起來時叫（按 ESC、身上沒東西了、
用了不結束行動的物品法術之後），電腦在 `foeChooseGear` 最後叫。

## remake 的次數怎麼存

`tacticalState.swingsLeft` 是這一回合出過手的人的 `+113h`／`+114h`（`startRound` 清空，等於
`002Eh` 清 `+8`）。原版在回合初始化就數好；remake 在這一回合第一次出手時照 `volleySwings` 現數
（中間沒有東西會改它：群組 18 在回合初始化算好，換武器走 `recountSwings`），之後的出手照剩下的
打，打完扣掉實際揮出去的（`lastSwings`，第二形態先扣）。沒有骰子的形態不揮也不數，與
`attackSwingsThisPhase` 相同。

## 驗證

- 送鍵（`Update()`）：`TestPlayerFighterKeepsTheSwingLeftAfterAKill`——編碼 4 的戰士撞上第一隻、一下
  殺掉，回合沒結束、還在移動裡、剩 `[1 0]`；撞第二隻只打一下（92），回合結束。
  `TestPlayerArcherShootsTheSecondArrowAtTheNextTarget`——短弓一回合兩發，第一發殺了近的，A、Enter
  預設瞄剩下那一隻，射第二發；箭 5 → 4 → 3，打完回合結束。
- foeTurn：`TestFoeKeepsTheSwingLeftAfterAKill`——編碼 4 的獸人殺了貼身的隊員，重挑目標、往前一步、
  對兩格外的隊員打剩下的那一下（94），回合結束。
- `TestRecountSwingsFollowsTheConditionalWriteBack`：`0E18h`、`0E34h`、`0E41h` 近戰與射擊四種，外加
  還沒出過手的不動。`TestItemMenuAfterAKillLeavesTheRawRate`（送鍵）：瞄準一下殺了目標剩 `[1 0]`，
  U、ESC 之後變成 `[4 0]`。
- 變異：`spendSwings` 一律回 false（回合一律結束）讓三條送鍵／foeTurn 測試變紅；`recountSwings` 拿掉
  `0E34h` 或射擊那一支各自讓條件寫回那一條變紅；ESC 不叫 `recountSwings` 讓物品選單那一條變紅。
- `0DFCh` 換完武器再進 entry 5 沒有專屬測試：entry 9 在回合開頭就看身邊有沒有敵人，走得到
  `0DFCh` 的只有走近之後或殺了一個之後的再進；`TestFoeWalkReproducesEveryOriginalAction` 40/40 照舊。
