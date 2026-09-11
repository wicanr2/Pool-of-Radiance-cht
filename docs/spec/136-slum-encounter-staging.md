# Spec 136：貧民窟的遭遇是怎麼排出來的

狀態：READY（入口 3 的整段流程、五張表的位元組、remake 實跑排出來的三群）；
DRAFT（`@9805`／`@9806`／`@9827` 三張表的語意、**走路時**由誰擲遭遇）。
日期：2026-09-11。

## 入口 3：紮營被打斷之後

打斷把棒子交給那一區的命令集入口 3（spec 114）。貧民窟這一份在 ECL2 block 20
的 `9A49h`：

```
9A49  SAVE 200 → @4A1F
9A4F  SAVE 0 → @6DCB
9A55  PARTYSTRENGTH → @9808        ; 隊伍強度
9A59  GOTO @9B68
...
9B68  DIVIDE @9808 / 3 → @9808     ; 強度 ÷ 3
9B71  MULTIPLY @9808 × 2 → @9808   ; × 2
9B7A  COMPARE @4A0B 255
9B81  ADD 5 + @9808 → @9808        ; 相等時再加五
9B8A  RANDOM 2 → @6E7F             ; 挑三種怪其中一種
9B90  ON GOTO                      ; KOBOLDS.／GOBLINS.／ORCS. 三支，只換名字
9BCF  GETTABLE @B6E3 @6E7F → @6DC1
9BD9  GETTABLE @B6E6 @6E7F → @6E80   ; **怪物編號**
9BE3  GETTABLE @B6E9 @6E7F → @9805
9BED  GETTABLE @B6EC @6E7F → @9806
9BF7  GETTABLE @B6EF @6E7F → @9827
9C01  PARTY SURPRISE @9802 @9803
9C08  SURPRISE @9802 @9803 0 0
9C1E  SETUP MONSTER @9827 2 @6E80
```

**`@9808` 是數量、`@6E80` 是怪物編號**：數量由隊伍強度推出來（`÷3 ×2 (+5)`），
所以隊伍越強來的越多。

## 分幾群由強度決定

```
B157  COMPARE @6E80 4          ; 是第四種怪嗎
B162  COMPARE @9808 8          ; 強度 < 8  → 一群
B171  COMPARE @9808 14         ; （第四種）強度 < 14 → 一群
B17C  ADD 1 + @6E80 → @6E7B    ; 第二群的編號 = 第一群 + 1
B185  COMPARE @6E80 1
B18C  SAVE 3 → @6E7A           ; 編號 < 1 時第二群的數量是 3
B193  SAVE 4 → @6E7A           ; 否則 4
B199  COMPARE @9808 18         ; 強度 > 18 → 三群
B1A8  一群：  LOAD MONSTER @6E80 @9808 @6E80
B1BF  兩群：  LOAD MONSTER @6E7B @6E7A @6E7B ＋ 上面那一群
B1E9  三群：  再加 LOAD MONSTER 63 1 63
```

`@4A16` 在每一支開頭各加 0／5／15——那是這一場的難度或獎賞，還沒對。

## 那五張表在腳本自己的位元組裡

`GETTABLE` 的基底 `B6E3h`..`B6EFh` 落在 block 20 自己的 payload 裡（code 從
`9900h` 起、長 7679 bytes），三格對應三種怪：

| 基底 | 三格 | 存到 |
|---|---|---|
| `B6E3h` | `00 13 02` | `@6DC1` |
| `B6E6h` | `02 02 00` | `@6E80`（怪物編號）|
| `B6E9h` | `02 04 06` | `@9805` |
| `B6ECh` | `06 09 06` | `@9806` |
| `B6EFh` | `06 09 00` | `@9827` |

**這在 remake 這一側本來就讀得到**：`eclvm.NewWithPassthroughSeed` 建 Machine
時把整份 payload 逐位元組寫進 `Memory`（`machine.go` 的 `for index, value :=
range payload`），所以 `codeBase` 之後的位址是有值的。原版也是同一件事——
ECL 的位址空間只有一個，腳本的位元組與變數在裡面並排。

## remake 實跑（2026-09-11）

`internal/gamepack/slum_encounter_test.go` 開 ECL2 block 20 的 session、跑入口
3，排出來的是

```
第 0 群：怪物 5、數量 4、造形 5
第 1 群：怪物 63、數量 1、造形 63
第 2 群：怪物 4、數量 28、造形 4
```

三群，與強度 > 18 那一支相符；中間那一群的 `63 1 63` 是字面值，對上 `B204h`。
接著停在 `29h ENCOUNTER MENU`（spec 078），也就是原版那張
`COMBAT WAIT FLEE PARLAY`。

## 原版走路會遇到怪（dosgolem，2026-09-11）

拿 dosgolem 從導覽結束的 `(0,4)` 往西走進貧民窟再繼續前進，**第二步之後就撞上**：

```
14, 4 W 00:01
YOU HAVE SURPRISED A PARTY OF GOBLINS.
COMBAT  WAIT  FLEE  PARLAY
```

基準畫面在 [`docs/reference/original-dos/slums/`](../reference/original-dos/slums/)
（`00-slums-14-4-after-one-step.png` 是走完一步的正常畫面，
`01-wandering-goblins-encounter.png` 是下一步的遭遇；dosgolem `d351681ba86d`）。

`GOBLINS.` 正是 `9B90h` 那三支 `ON GOTO` 的第二支——**走路的遭遇跑的是同一段**。
所以入口 3 不是「紮營專用」，它是**這一區的遭遇入口**：紮營被打斷會叫它，
走路擲中也會叫它。

## 還沒讀

- **走路時誰擲那一次。** 還沒找到。已經排除的：

  | 查過的地方 | 結果 |
  |---|---|
  | ECL 入口 0（每格）| 整張圖 1024 個樣本沒有一格走到 `24h COMBAT`（spec 100）；實跑也只停在 `2Dh CALL` 就結束 |
  | overlay-03 `312Ah` | 它跑入口 2 → 問 `00ACh:0025h` → 非零才跑入口 3。但 `00ACh` 是 **overlay-15**（spec 109），entry 1（`1E45h`）開頭設 `DS:4954h = 2`、清 `DS:6CB6h` 那七格、再畫面等鍵——那是**紮營畫面**，不是走路擲骰 |
  | `6DD2h`／`6DD3h`（休息的週期與門檻）| 只有 overlay-07 寫、overlay-20 的休息迴圈讀（`cmd/pool-disp-scan -ecl-class 1`）|
  | `6DC7h`／`6DC8h` | 只有 overlay-05 碰，而 overlay-05 是戰後服務（spec 034／036）——那兩格是上一場的結果碼，`B118h` 讀它決定要不要再打一場 |

  下一步是掃**移動**那一側：走一步的引擎路徑（overlay-07 的座標更新之後）
  有沒有擲骰再呼叫入口 3。
- `@9805`／`@9806`／`@9827` 是什麼（造形、等級或別的）。
- `@4A16` 那 0／5／15 的意思。
