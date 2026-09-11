# Spec 136：貧民窟走一步會遇到什麼

狀態：READY（入口 1 的整條路徑、擲骰與門檻、四個旗標、編組、突襲四路、
四個出口、五張參數表與逃跑落點的位元組）；DRAFT（`@9805`／`@9806`／`@9827`
各是什麼、`@4A16` 那組遞減值、`@C04F` 的最高位）。
日期：2026-09-11。

## 走路遭遇跑的是入口 1

overlay-03 的主迴圈（`3626h`，spec 135）每一圈先跑入口 4，再依玩家指令分派；
走一步之後 `3741h` 跑入口 0，`4391h == 0` 時接著 `3750h` 跑**入口 1**。

貧民窟這一份在 ECL2 block 20 的 `9975h`，是**本格事件碼的分派器**：

```
9975  SAVE 255 → @6DE1 ; PICTURE 255 ; SAVE 0 → @4A1F ; PRINTCLEAR ""
9987  AND @C04F 7Fh → @6E82      ; 本格的值遮掉最高位 = 事件碼
9990  SAVE 0 → @6E79 / @6E7A / @6E7B / @9800
99A8  COMPARE @9800 @6E82        ; 迴圈：@9800 從 0 數上去
99AF  IF = GOTO @99C9
99B4  COMPARE @9800 20 ; IF > EXIT   ; 事件碼 > 20 就什麼都不做
99BC  ADD 1 + @9800 → @9800 ; GOTO @99A8
99C9  ON GOTO @9800 → 21 個目標
```

`@C04F` 是玩家所在格的值，`AND 7Fh` 之後就是**事件碼**；最高位另有用途（走過
的標記之類，還沒讀）。21 個目標的第一個是 `9AA5h`，那是**事件碼 0**——也就是
「這一格沒有特別的事」，走路遭遇就擲在這裡。

## 事件碼 0：三道否決，然後擲一次

```
9AA5  PARTYSTRENGTH → @9808          ; 隊伍強度
9AA9  COMPARE @4ABB 254 ; IF >= EXIT  ; 否決一
9AB1  COMPARE @4A00 255 ; IF <> PRINTCLEAR 'YOU HAVE ENTERED THE MONSTER-
                                        CRAWLING SLUMS OF PHLAN. ...'
9B24  SAVE 255 → @4A00               ; 這段導言一張圖只講一次
9B2A  COMPARE 0 @6E82 ; IF <> EXIT   ; 否決二：事件碼必須真的是 0
9B32  COMPARE @4A80 15 ; IF >= EXIT   ; 否決三
9B3A  RANDOM 13 → @980F              ; 0..13，十四個等機率值
9B40  COMPARE @6DCA 1  ; IF = ADD 4 + @980F → @980F
9B50  COMPARE @4A0B 255; IF = ADD 5 + @980F → @980F
9B60  COMPARE @980F 12 ; IF <= EXIT  ; 只有 > 12 才遭遇
```

`RANDOM N` 產生 **0..N**（engine `machine.go` 的 `bound = max + 1`）。同一段裡
`RANDOM 2` 正好餵三個 `ON GOTO` 目標、`RANDOM 1` 正好分兩路，這是腳本自己給的
一致性佐證。於是遭遇機率是

| `@6DCA == 1` | `@4A0B == 255` | 加值 | 遭遇的值 | 機率 |
|---|---|---|---|---|
| 否 | 否 | 0 | 13 | 1/14 ≈ 7.1% |
| 是 | 否 | +4 | ≥ 9 | 5/14 ≈ 35.7% |
| 否 | 是 | +5 | ≥ 8 | 6/14 ≈ 42.9% |
| 是 | 是 | +9 | ≥ 4 | 10/14 ≈ 71.4% |

### 四個旗標

三個讀出來了，都在 block 20 自己裡面：

**`@4A80`／`@4ABB` 是清區進度。** 打完一場從 `B118h` 回來：

```
B118  COMPARE @6DC7 128 ; IF > GOTO @B225      ; 被打退 → 走逃跑那條
B123  COMPARE @6DC7 1   ; IF < ADD 1 + @4A80 → @4A80 ; GOSUB @B69C
B13E  COMPARE @6DC8 0   ; IF > ADD 1 + @4A80 → @4A80 ; GOSUB @B69C
B69C  COMPARE @4ABB 254 ; IF >= RETURN         ; 鎖上了就不再累加
B6A4  ADD 1 + @4ABB → @4ABB
B6AD  COMPARE @4ABB 25  ; IF < RETURN
B6B5  SAVE 254 → @4ABB                          ; 滿 25 就鎖成 254
```

`@6DC7`／`@6DC8` 是上一場的結果碼（overlay-05 寫的）。所以**打贏才加**，
兩個計數各自對上一道否決：`@4A80` 滿 15、或 `@4ABB` 滿 25（鎖成 254），
走路遭遇就停了。貧民窟清得完。

**`@4A0B` 是老太婆那條任務的狀態機。** 事件碼 8（`A63Ah`）是酒館那一格：

```
A63A  COMPARE @4A0B 250 ; IF > EXIT            ; 已經推進過就不再給
A642  SAVE 251 → @4A0B                          ; 接下任務
A651  PRINTCLEAR 'SEATED AT A TABLE IS A RAGGED OLD WOMAN. ...'
...
A749  SAVE 0 → @4A80                            ; 清區進度歸零
A74F  SAVE 255 → @4A0B                          ; 任務完成
A755  GOSUB @B6BC ; CLEARMONSTERS ; TREASURE
```

**任務完成之後貧民窟變危險**：遭遇機率 +5、怪物數量 +5，而且清區進度從頭算。

**`@6DCA` 是命令列的兩個搜尋旗標。** 它在八個封存共 64 筆引用，而且**一筆
ECL 寫入都沒有**——`SAVE @6DCA → @xxxx` 那幾筆是拿它當來源。寫它的是引擎：
換算成 class 1 的位移 `594h` 之後（`cmd/pool-disp-scan -addresses 6DCA,6DD5
-ecl-class 1`，`6DD5h → 5AAh` 當正對照）掃出 16 筆，寫入點都在 overlay-14
的鍵盤分派區：

| 位址 | bytes | 意思 |
|---|---|---|
| overlay-14 `0A9Eh`／`0AAAh` | `8B 85 94 05` ＋ `35 01` ＋ `89 85 94 05` | `ax = [di+594]` ; `xor ax,1` ; 寫回——**切第 0 位** |
| overlay-14 `0AB9h`／`0AC5h` | `8B 85 94 05` ＋ `0D 02` ＋ `89 85 94 05` | `or ax,2`——**設第 1 位** |
| overlay-03 `387Bh` | `C7 85 94 05 01 00` | 直接寫 1 |
| overlay-03 `3839h` | `83 BD 94 05 01` | `cmp word [di+594], 1` |

第 0 位是「邊走邊搜」（toggle）、第 1 位是「這一次要搜」（設一次就用掉）——
remake 的 `cmd/pool-game/command_bar.go` 早就這樣寫了（`searchFlags ^=
SearchWhileWalkingBit`／`|= LookOnceBit`），註解也標著 `[4937h]+594h`。

**所以「邊走邊搜」會把怪引出來**：`@6DCA == 1` 時那一擲加四，遭遇率從 1/14
變 5/14。`internal/gamepack/slum_wandering_test.go` 的
`TestSearchingWhileWalkingRaisesTheEncounterRate` 走 300 步，不搜 12 次、
邊走邊搜 45 次。

遊戲那一側原本沒把 `searchFlags` 投影進 ECL memory（旗標只活在 Go 的欄位裡），
腳本讀到的永遠是 0；`beginInitialSearch` 現在跑入口 1 之前會寫進去。

## 編組：強度決定來幾隻、分幾群

擲中之後落到 `9B68h`；紮營被打斷那一條（入口 3，`9A49h`）是 `GOTO @9B68`
**直接跳進來**，跳過上面整段擲骰——兩個入口共用編組碼，但**只有入口 1 會擲**。

```
9B68  DIVIDE @9808 / 3 → @9808 ; MULTIPLY @9808 × 2 → @9808
9B7A  COMPARE @4A0B 255 ; IF = ADD 5 + @9808 → @9808
9B8A  RANDOM 2 → @6E7F
9B90  ON GOTO @6E7F → 'KOBOLDS.' / 'GOBLINS.' / 'ORCS.' → @9890
9BCF  GETTABLE @B6E3 @6E7F → @6DC1
9BD9  GETTABLE @B6E6 @6E7F → @6E80
9BE3  GETTABLE @B6E9 @6E7F → @9805
9BED  GETTABLE @B6EC @6E7F → @9806
9BF7  GETTABLE @B6EF @6E7F → @9827
```

`@9808` 是數量、`@6E80` 是怪物編號，數量由隊伍強度推出來（`÷3 ×2 (+5)`），
所以隊伍越強來的越多。

分幾群在 `B157h`（`24h COMBAT` 也在這裡）：

```
B157  COMPARE @6E80 4     ; 是第四種怪嗎
B162  COMPARE @9808 8     ; 強度 < 8 → 一群
B171  COMPARE @9808 14    ; （第四種）強度 < 14 → 一群
B17C  ADD 1 + @6E80 → @6E7B         ; 第二群的編號 = 第一群 + 1
B185  COMPARE @6E80 1 ; SAVE 3 或 4 → @6E7A   ; 第二群的數量
B199  COMPARE @9808 18 ; IF > GOTO @B1E9      ; 強度 > 18 → 三群
B1A8  一群：LOAD MONSTER @6E80 @9808 @6E80 ; COMBAT
B1BF  兩群：再 LOAD MONSTER @6E7B @6E7A @6E7B
B1E9  三群：再加 LOAD MONSTER 63 1 63
```

`@4A16` 在每一支開頭各加 0／5——還沒對上語意。

## 突襲決定演哪一齣

```
9C01  PARTY SURPRISE @9802 @9803
9C08  SURPRISE @9802 @9803 0 0      ; 結果落在 @6DCB
```

`@6DCB` 之後分四路，每一路的差別只有開場白與 `ENCOUNTER MENU` 的參數：

| `@6DCB` | 走到 | 玩家看到 |
|---|---|---|
| 1 | `9C1E`（不開選單，直接打）| `SETUP MONSTER` → `APPROACH` → `YOU ARE SURPRISED BY` ＋ 怪物名 → `GOTO @B157` |
| 2 | `9DBA` | `YOU HAVE SURPRISED A PARTY OF` ＋ 怪物名 |
| 3 | `9D01` | `YOU BUMP INTO SOME` ＋ 怪物名 |
| 其他 | `9C8A` 再 `RANDOM 1`：0 → `9D58`、1 → `9C9B` | `YOU SPY A GROUP OF SEEDY-LOOKING` ／ `YOU ARE BEING APPROACHED BY SOME ANGRY` |

`@9808 > 18` 時還會在怪物名前面加一句 `A BUGBEAR LEADING A PARTY OF`。

**只有 `@6DCB == 1`（怪物突襲玩家）不給選單**，直接開打——玩家被偷襲就沒得選，
這與原版體感一致。

## 四個出口

三支開選單的都接同一個 `ON GOTO @9802`（`@9802` 是 `29h ENCOUNTER MENU` 的
第 4 個運算元，也就是 spec 078 說的「結果要寫進哪個 ECL 變數」）：

| `@9802` | 目標 | 做什麼 |
|---|---|---|
| 0 | `B250` | `PICTURE 255` ; `EXIT`——選單自己已經把事情辦完了 |
| 1 | `B157` | 編組 ＋ `24h COMBAT` |
| 2 | `B225`／`B21A` | 逃走（見下）；「我方突襲」那一支改成 `B21A`＝`YOU HIDE.` 後直接 `EXIT` |
| 3 | `B254` | `YOU ARE CONVERSING WITH` …＝交談 |

`ENCOUNTER MENU` 的第 5..9 個運算元是 spec 078 的**五格結果表**，四支各不同：

| 開場白 | 位址 | 五格 |
|---|---|---|
| `YOU ARE BEING APPROACHED BY SOME ANGRY` | `9CC5` | 0, 0, 0, 1, 0 |
| `YOU BUMP INTO SOME` | `9D1D` | 0, 2, 0, 1, 4 |
| `YOU SPY A GROUP OF SEEDY-LOOKING` | `9D7E` | 0, 2, 2, 1, 4 |
| `YOU HAVE SURPRISED A PARTY OF` | `9DDE` | 0, 2, 2, 1, 4 |

同一支還各自 `SAVE 60/50/40/30 → @4A16`（被突襲那一支是 70）——遞減的順序正好
是「對玩家越不利，數字越大」。

## 逃走會把隊伍丟到哪

```
B225  RANDOM 3 → @C04D                  ; 朝向，0..3
B22B  RANDOM 4 → @9809                  ; 五個落點挑一個
B231  GETTABLE @B6F2 @9809 → @C04B      ; x
B23B  GETTABLE @B6F7 @9809 → @C04C      ; y
B245  PRINTCLEAR "" ; CALL 2C90h        ; 重畫（spec 135 的六個選擇子之一）
B24C  GOTO @9975                        ; 回到入口 1 開頭，重跑新格子的事件
```

`@C04B`／`@C04C`／`@C04D`／`@C04F` 連號，正是**玩家的 x／y／朝向／本格值**——
入口 1 開頭讀的就是 `@C04F`。逃跑不是「原地脫離」，是把隊伍**傳送到五個安全點
之一**再重新解析那一格。

## 五張參數表與落點表的位元組

`GETTABLE` 的基底落在 block 20 自己的 payload 裡（code 從 `9900h` 起、長 7679
bytes）；ECL 的位址空間只有一個，腳本的位元組與變數並排在裡面。

```
B6E3  02 02 02 00 02 04 06 06 09 06 06 09 00 02 04 0F
B6F3  0E 0B 0B 09 04 06 05 02 02 00
```

| 基底 | 三格（KOBOLDS／GOBLINS／ORCS）| 存到 |
|---|---|---|
| `B6E3h` | `02 02 02` | `@6DC1` |
| `B6E6h` | `00 02 04` | `@6E80`（怪物編號）|
| `B6E9h` | `06 06 09` | `@9805` |
| `B6ECh` | `06 06 09` | `@9806` |
| `B6EFh` | `00 02 04` | `@9827` |

逃跑落點五格：

| 索引 | 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|---|
| `B6F2h` → `@C04B`（x）| 15 | 14 | 11 | 11 | 9 |
| `B6F7h` → `@C04C`（y）| 4 | 6 | 5 | 2 | 2 |

位址基準用 `workplace/eclbytes`（`9900h` 基準）驗過：`9BCF` 讀回
`2A 01 E3 B6 01 7F 6E 01 C1 6D`，逐位元組對上反組譯的
`GETTABLE @B6E3 @6E7F → @6DC1`。

## 原版走路實測（dosgolem，2026-09-11）

從導覽結束的 `(0,4)` 往西走進貧民窟再前進，第二步之後：

```
14, 4 W 00:01
YOU HAVE SURPRISED A PARTY OF GOBLINS.
COMBAT  WAIT  FLEE  PARLAY
```

基準畫面在 [`docs/reference/original-dos/slums/`](../reference/original-dos/slums/)
（dosgolem `d351681ba86d`）。這一幕對應上表的 `@6DCB == 2`、怪物索引 1。

## remake 實跑（2026-09-11）

兩條路都跑得通：

- `slum_encounter_test.go` 跑入口 3（紮營那條），排出三群並停在
  `29h ENCOUNTER MENU`。
- `slum_wandering_test.go` 一步一步跑入口 1，擲中之後排出

  ```
  第 0 群：怪物 1、數量 3、造形 1
  第 1 群：怪物 63、數量 1、造形 63
  第 2 群：怪物 0、數量 28、造形 0
  ```

  主群是編號 0 的 KOBOLDS ×28（`B6E6h` 的第一格就是 0），第二群是 `@6E80 + 1`
  的 3 隻，第三群是 `B204h` 的字面值 `63 1 63`。

**入口 1 在遊戲本體本來就每一步都跑**：`cmd/pool-game/main.go` 走完一步之後
呼叫 `beginInitialSearch`，而它跑的就是命令集入口 1
（`RunInitialSessionSearchEntry`，名字還停在早期把入口 1 當「搜尋」的猜測上）。
位置投影 `projectInitialPosition` 也已經把 `@C04B`／`@C04C`／`@C04D`／`@C04F`
寫好了。所以走路遭遇缺的不是接線，是**遊戲層還沒有端到端測試**走進貧民窟證明
玩家真的遇得到。

### 隨機流每一局都一樣

`RANDOM` 的 seed 在三個建立點都硬寫成 1，走路遭遇的測試連跑五次、第 0 步擲中、
三群的編號與數量一個位元都不差。對拍與測試要這個決定性（spec 050 的「DOS 同
seed 戰鬥對拍」），正常遊玩不該有——原版的 seed 來自 DOS 計時器 tick。
記在 worklist 的 `ecl-random-seed-fixed`。

## 還沒讀

- `@9805`／`@9806`／`@9827` 三張表的語意（`@9827` 是 `SETUP MONSTER` 的第一個
  參數，`@6E80` 是 `LOAD MONSTER` 的，兩張表的值一樣都是 `00 02 04`）。
- `@C04F` 最高位（`AND 7Fh` 遮掉的那一位）的用途。
- `@4A16` 的 70／60／50／40／30。**線索（hypothesis，還不夠定名）**：每一支
  `ENCOUNTER MENU` 之後把它 `SAVE → @6DC6`，逃跑那條還先 `ADD 5`。`@6DC6` 換算
  成引擎位移是 `58Ch`，只有四筆引用：戰鬥 setup（overlay-10，spec 051 那段的
  隔壁）在 `1F70h` 比 100、`1F7Ch` 夾成 100，怪物 AI（overlay-09，spec 096）在
  `11C2h` 用 `sub ax, [di+58C]` 把它減掉。所以它是一個**上限 100、進戰鬥後由
  怪物 AI 扣減的百分比**；玩家越不利數字越大（被突襲 70、我方突襲 30），
  方向像是怪物的士氣或追擊意願，但還沒追到用它的那一支判斷。

## 被推翻的舊斷言

- 「`GOBLINS.` 正是入口 3 那三支 `ON GOTO` 的第二支，所以入口 3 是這一區的遭遇
  入口」——**錯**。入口 3 直接 `GOTO @9B68` 跳進編組，擲骰整段在入口 1。
- 五張表舊記為 `B6E3h = 00 13 02`、`B6E6h = 02 02 00`⋯——**整張往後錯一格**，
  正確值見上表。
- 「走路時誰擲那一次還沒找到」——找到了，就是入口 1 的 `9B3Ah`。

第一條與第三條同一個原因：ECL 的文字是 6-bit packed 的 `80h` 運算元，拿 ASCII
掃 payload **結構上一定零命中**，而零命中被讀成「原版沒有這句話」，於是只能靠
「哪個入口看起來像」去推。找字串一律走 `cmd/pool-text-inventory`（它跑指令圖、
解 `80h` 運算元與選單記錄，一次吐 1800 句連位址），不要 grep 位元組。

第二條是另一回事——讀表的位址基準偏了一格。這種錯不會報錯，只會安靜地給出一張
看起來合理的表。**讀內嵌資料表之前先拿已知指令對一次**：`workplace/eclbytes`
讀 `9BCF` 要得到 `2A 01 E3 B6 01 7F 6E 01 C1 6D`，逐位元組對上反組譯的
`GETTABLE @B6E3 @6E7F → @6DC1`，基準才算驗過。
