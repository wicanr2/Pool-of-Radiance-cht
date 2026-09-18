# Spec 102：城區的地點是 terrain 索引分派的，而且有些要面對它

狀態：READY

## 一句話

城區每一格「這裡有什麼」不是座標硬編碼，是 **`terrain & 0x7F` 當索引**去查
一張 `ON GOTO` 表；其中好幾個地點還要求隊伍**從特定方向走進去**。

## 分派怎麼走

`ECL3.DAX` block 0，入口 1（`99EBh`，移動之後跑的那一個）：

```
99F7  AND 127, @C04F → @6E82      ; C04F = 這一格的原始 terrain byte（spec 015）
9A00  COMPARE @4AC5, 1 ; IF < ; EXIT   ; 這個沒開，整支分派都不跑
9A37  SAVE 0 → @9800
9A3D  COMPARE @9800, @6E82
9A44  IF = → GOTO 9B4C
9A49  COMPARE @9800, 27 ; IF > → EXIT
9A51  ADD 1, @9800 → @9800
9A5A  GOTO 9A3D
9B4C  ON GOTO                     ; 28 個目標，索引就是 @9800
```

`9A3D..9A5A` 是一個從 0 數到 27 的線性搜尋，找到等於 `@6E82` 的那一個就用它當
`ON GOTO` 的索引。**索引就是 terrain byte 的低七位**。

`9B4Ch` 有 **28 個目標**，索引 0 起算，順序就是 terrain 索引 0..27
（`cmd/pool-ecl-trace` 的 `branch_targets`）。地點的名字取自那一支自己印出來
的第一句話，不是攻略：

| 索引 | 目標 | 地點 | 格數（geo3/0）|
|---:|---|---|---:|
| 0 | AE6A | 沒有地點（一般街道）| 185 |
| 1 | 9BA6 | 碼頭的船 | 1 |
| 2 | 9C43 | 港務長 | 1 |
| 3 | 9DD5 | 指示牌「TO THE NORTH IS THE PASSENGER DOCK.」| 1 |
| 4 | AE6A | 沒有地點 | 2 |
| 5 | 9E07 | 泰爾神殿的主教（BISHOP BRACCIO）| 1 |
| 6 | A064 | 泰爾神殿「THE PRIESTS OF TYR WELCOME YOU」| 3 |
| 7 | A0BC | 蘇妮神殿（PRIESTESS JOY OF SUNE）| 3 |
| 8 | A0F7 | 河邊（STOJANOW RIVER）| 1 |
| 9 | A140 | 旅店，住一晚一枚白金 | 7 |
| 10 | A22D | 雜貨店 | 2 |
| 11–16 | AE6A | 沒有地點 | 1、1、1、4、1、1 |
| 17 | A230 | 雜貨店（同一段的另一個入口）| 1 |
| 18 | AE6A | 沒有地點 | 1 |
| 19 | A233 | 雜貨店（同一段的第三個入口）| 5 |
| 20 | A2C6 | 酒館兼賭場 | 7 |
| 21 | A834 | 珠寶店 | 1 |
| 22 | A8BB | 武具店（spec 067 的 `A919h`）| 5 |
| 23 | A942 | 銀器店 | 2 |
| 24 | A9D3 | 坦帕斯神殿 | 2 |
| 25 | AAD5 | 城門（往未開拓區）；開場的起點 (0,4) | 1 |
| 26 | AB97 | 市政廳外的布告 | 3 |
| 27 | AD7F | 城衛兵遭遇 | 1 |

`AE6Ah` 是「這一格沒有地點」——索引 0、4、11–16、18 都指到它。
索引 10、17、19 都落在同一段雜貨店腳本（`A22Dh`→`A230h`→`A233h` 是連續的
三個進入點，都會走到 `A24Bh` 的「THE SHOPKEEPER GREETS YOU…」）。

四家店的位置因此可以直接從地形碼讀出來（`cmd/pool-world-graph` 的
`cell_terrain`）：

| 店 | 索引 | geo3/0 的格子 |
|---|---:|---|
| 雜貨店 | 10、17、19 | (6,1) (6,2) (9,0) (15,8) (9,10) (12,10) (9,11) (11,11) |
| 珠寶店 | 21 | (8,10) |
| 武具店 | 22 | (13,8) (8,11) (11,12) (8,13) (9,13) |
| 銀器店 | 23 | (11,10) (10,13) |

> **目標的順序只能從 `branch_targets` 讀。** 靜態圖的 `edges` 是**排序過的**
> （`cmd/pool-ecl-trace` 對 `graph.Edges` 做 `sort.Slice`），拿它推「第幾個
> 選項跳到哪」會得到自洽但錯的結論。

## 城區有 36 個索引，分派表只涵蓋 0..27

geo3/0 的地形碼低七位共出現 **36 個值（0..35）**，而 `9B4Ch` 只有 28 個目標。
差額不是漏讀：`9A49h COMPARE @9800, 27 ; IF > ; EXIT` 讓線性搜尋數到 28 就
離開，**索引 28..35 的格子在 ECL3/0 的入口 1 不會分派到任何地點**。

那八個索引共 12 格，連成城區西邊的一塊：

| 索引 | 格子 |
|---:|---|
| 28 | (4,5) |
| 29 | (5,5) |
| 30 | (6,5) |
| 31 | (5,6) |
| 32 | (5,7) (6,7) (6,8) |
| 33 | (5,9) (6,9) |
| 34 | (4,6) |
| 35 | (4,7) |

**全部 36 個索引的格子都與起點在同一個雙向連通元件**（spec 101 的元件表），
所以走得到不是問題。

## 有些地點要面對它

港務長那一支（`9C43h`）開頭就是兩道閘門：

```
9C43  COMPARE @C04D, 0 ; IF <> → EXIT     ; C04D = 朝向（spec 015）；0 是北
9C4B  COMPARE @4A01, 1 ; IF =  → EXIT
9C53  SAVE 16 → @6DE1
9C59  PICTURE 7
9C5C  COMPARE @4AA7, 254 ; IF < → GOTO 9D54
9C67  PRINTCLEAR 'THE HARBOR MASTER TELLS YOU BOATS LEAVE FOR THE WEST, …'
9CF0  HORIZONTAL MENU [SOKAL, EAST, WEST, BAY, NONE]
```

**朝向不是北就直接離開。** 入口 1 是移動之後跑的，所以「朝向」等於隊伍**走進
這一格的方向**——要跟港務長講話，得從南邊（(11,2)）往北走進 (11,1)。

## 對探索的意思

地點腳本同時被**方向**與**旗標**閘住，所以自動探索器的目標不是「格子」：

- 目標是**（格子, 走進去的方向）**。只記「踩過這一格」的話，那一次的方向是
  規劃器隨手挑的，四分之三的機會問不到港務長。
- 旗標一變，走過的地點要**重來一次**。同一格在不同旗標下講不同的話，
  港務長更是「有票就不開口」，票的狀態一改就等於換了一個地點。
- 主線那幾段中間**不能離開這一張圖**。碼頭 (15,1) 就在港務長 (11,1) 旁邊，
  先踩到碼頭就會被載走，回來時入口 4 又把 `4AC4` 清成 0。

## 港務長的第二道閘門：`@4A01` 是船票

同一支的第二道閘門是 `9C4Bh COMPARE @4A01, 1 ; IF = → EXIT`。整個 ECL3/0 對
`4A01` 只有四處：

| 位址 | 動作 | 位置 |
|---|---|---|
| `9BA6h` | `COMPARE 0`，等於 0 就 `EXIT` | 碼頭的船（沒票不讓上） |
| `9C4Bh` | `COMPARE 1`，等於 1 就 `EXIT` | 港務長（有票就不再說） |
| `9D29h` | `SAVE 1` | 選完航線、`GOSUB A1B4`（`WHO 'WHO WILL PAY?'`）付錢之後 |
| `9DCEh` | `SAVE 1` | 「唯一的船是去索寇要塞」那一支，順便 `SAVE 0 → @4AC4` |

**這個位址不是碼頭專用的。** 同一張 GEO（block 0）上的市政廳職員腳本
（ECL3 block 8）拿它當「介紹過了沒」：

```
9BA1  COMPARE @4A01, 0 ; IF > ; GOTO A7FD   ; 大於 0 就是已經介紹過
9BAC  SAVE 1 -> @4A01
9BB2  SAVE 1 -> @4A06
9BB8  PRINTCLEAR 'AT YOUR ENTRY, THE COUNCIL CLERK BEGINS LOOKING THROUGH…'
```

職員**只在它等於 0 時才寫**，所以 255（說謊清掉的那個值）不會被他蓋掉。

競技場那一支（ECL3 block 11，同一張 GEO）就不一樣了：

```
9BFA  COMPARE @4A01, 0 ; IF > ; GOTO 9CAC
9C05  PRINTCLEAR 'THE ARENA MASTER ASKS IF YOU WISH TO DUEL…'
9CAC  SAVE 1 -> @4A01                      ; 進到這一支就一定會寫
9CBA  PRINTCLEAR "'DO YOU SEEK A PARTNER FOR YOUR ADVENTURING?'"
```

`4A01` 大於 0 就跳到 `9CACh`，而那一行**沒有再檢查什麼就寫 1**——255 也大於 0，
所以說謊之後走進競技場，票就又沒了——實測探索器
就是這樣把 255 弄回 1 的（`4A01 255→1 於 GEO3/0 (7,2) ECL block 11`）。
**說謊回城之後要先問港務長，再逛別的地方。**

三支合起來看，`4A01` 比較像「手上有沒有一件在辦的事」而不是單指船票：
先去市政廳領委託也會讓它變成 1，那時港務長不開口，但碼頭那一格只要求
**不等於 0**，所以照樣上得了船去索寇要塞。

所以 `4A01` 在碼頭這一支的意思是：0 沒票、1 有票。第一次找港務長時 `DS:4AA7h` 還小於
254，走的是 `9D54h` 那一支——免費給一張去索寇要塞的票，並把 `4A01` 設成 1。

**ECL 裡沒有清票的指令，清票的是引擎。** 全部 29 個 ECL 區塊裡寫 `4A01` 的，
只有 ECL3/0 的兩處 `SAVE 1`、ECL4/21 `ABBDh` 的 `SAVE 255`，以及其他區塊各自的
用法；但每載入一個 ECL 區塊，overlay-07 entry 3 會把 `4A00..4A1F` 清成 0
（spec 106〈載入區塊時清掉的兩段〉）。所以 `4A01` 只在**同一個區塊裡**有效：
在城區拿到票、不離開城區走到碼頭，票還在；走出城區（進任何一棟有自己區塊的
建築、出城、上船）票就沒了，港務長下次會再開口。

## 船票是誰清的

原版清票的是引擎載入區塊那一支（見上方與 spec 106）。ECL 裡唯一「清票」的指令是
ecl4/21 `ABBDh` 的 `SAVE 255 @4A01`，落在索寇要塞亡魂問話的
`LIE?` 那一支（`'YOU LIE,' THE SPIRIT MOANS AS IT FADES AWAY.'`）。
走到它的路是：登陸那一格的遭遇選 **PARLAY**（`AA0Ah`）→ 對亡魂說出 `LUX`
（`AA7Eh` 的 `INPUT STRING`）→ 費蘭現身問話（`AB67h`）→ 選 `LIE?`。

除了 ECL，會動這個位址的只有引擎載入區塊那一支：

- **引擎的 ECL 變數存取不以字面值出現。** 38 個 overlay 加 `START.EXE` 裡，
  `4A01` 的兩位元組字面值（`01 4A`）出現 **0 次**——這不代表引擎不寫它：引擎
  寫的是「基底＋位移」，而載入區塊的清除是迴圈 `add ax, 49FFh`，連位移都不是
  常數（spec 106）。
- **換區會清它。** 共用 engine 的 `NEWECL` 只換掉 `0x9900` 以上的程式碼段，
  但原版在載入區塊時另外把 `4A00..4A1F` 清成 0；dosgolem 實跑量到走出市政廳
  那一步 `4A01 01→00`，寫入點 `1997:02FE`（overlay-07）；跨 archive（貧民窟 → 城區）
  也量過。remake 由 `gamepack.BlockLoadWrites` 宣告、engine 在換區時套用（spec 106）。
- **`LOAD FILES` 的第三欄不動它。** 那一支載的是背景圖庫 `BACPAC`
  （overlay-03 `0E0Fh`，見 spec 043），不碰 ECL 的變數區。

兩道閘門逐位元組核對過，不是解碼問題：

```
9C43  03 01 4D C0 00 00   COMPARE @C04D, 0
9C49  17                  IF <>
9C4A  00                  EXIT
9C4B  03 01 01 4A 00 01   COMPARE @4A01, 1
9C51  16                  IF =
9C52  00                  EXIT
```

同一個位址在不同區的 ECL 裡是不同的旗標（`4A01` 在 ECL1/18、ECL2/9、
ECL6/25 等處各有各的用法），所以「誰寫 `4A01`」要連區塊一起看。

亡魂那一段能走通，前提是遭遇選單的第四項索引要對（PARLAY 是類型表的第 4 格，
不是第 3 格），見 spec 078。

## 船去哪裡：`@4AC4` 是目的地

`4AA7h >= 254` 之後港務長給的是完整選單，選了什麼直接寫進 `4AC4`：

```
9CF0  HORIZONTAL MENU ['SOKAL','EAST','WEST','BAY','NONE'] -> @4AC4
9D10  ON GOTO                 ; 五個目標，NONE 那一個到 AE6Ah（不付錢就走）
9D25  GOSUB A1B4              ; WHO 'WHO WILL PAY?' → 扣一枚白金
9D29  SAVE 1 -> @4A01         ; 買到票
9D2F  PRINTCLEAR "'CATCH THE BOAT AT THE END OF THE PIER.'"
```

選單存的是 **0 起算**的游標，`ON GOTO` 也是 0 起算（引擎 `machine.go` 的
`store(destination, menu.Selected, pc)` 與 `targets[index]`）。所以 SOKAL = 0、
EAST = 1、WEST = 2、BAY = 3、NONE = 4。`9D10h` 的目標表照這個順序讀就對得上：

| 索引 | 目標 | 意思 |
|---:|---|---|
| 0..3 | `9D25h` | 付錢拿票 |
| 4 | `AE6Ah` | NONE，什麼都不做 |

碼頭那一格（(15,1)，`9BA6h`）拿 `4AC4` 分派：

```
9BA6  COMPARE @4A01, 0 ; IF =  ; EXIT          ; 沒票不讓上
9BAE  COMPARE @49C9, 14 ; IF > ; GOTO 9BBC     ; 不大於 14 才 PICTURE 41
9BC2  PRINTCLEAR 'YOU BOARD A BOAT.'
9BDC  LOAD FILES 255, 255, 127
9BE3  COMPARE @4AC4, 0 ; IF = ; SAVE 4 -> @6E12 ; IF = ; NEWECL 21
9BF4  COMPARE @4AC4, 2 ; IF < ; GOTO 9C2E ; IF > ; GOTO 9C19
9C04  SAVE 7  -> @49C3 ; 29 -> @49C4 ; 7 -> @6E12 ; NEWECL 26
9C19  SAVE 13 -> @49C3 ; 27 -> @49C4 ; 7 -> @6E12 ; NEWECL 26
9C2E  SAVE 9  -> @49C3 ; 29 -> @49C4 ; 8 -> @6E12 ; NEWECL 27
```

| 選項 | `4AC4` | 下船的地方 |
|---|---:|---|
| SOKAL | 0 | ECL／GEO block 21（索寇要塞），`6E12 = 4` |
| EAST | 1 | 野外圖 27 的 (9,29)，`6E12 = 8` |
| WEST | 2 | 野外圖 26 的 (7,29)，`6E12 = 7` |
| BAY | 3 | 野外圖 26 的 (13,27)，`6E12 = 7` |
| NONE | 4 | `AE6Ah`，不付錢也不上船 |

**船資是一枚白金**，`A1B4h` 的 `WHO 'WHO WILL PAY?'` 挑人來付；挑到的人身上
白金不足就印 `YOU DON'T HAVE ENOUGH PLATINUM.` 並且不發票（spec 090）。

`49C3`／`49C4` 是野外的座標（spec 105）。免費那一趟把 `4AC4` 寫成 0
（`9DC8h`），跟選 SOKAL 同一個值，所以走同一條路；入口 4 每次進城區也把
`4AC4` 清成 0（`9AF2h`），沒先找港務長就上船的話，去的一定是索寇要塞。

## 票的三個狀態與亡魂那兩支

原版每載入一個區塊就把 `4A01` 清成 0（spec 106），所以從索寇要塞回到城區時
`4A01` 一定是 0，港務長會開口——**說謊不是開航線的前提**，下面這兩張表只說明
「同一個區塊裡」各支寫了什麼。

`4A01` 有三個狀態，兩道閘門合起來看就清楚了：

| `4A01` | 港務長（`9C4Bh` 等於 1 就離開） | 碼頭的船（`9BA6h` 等於 0 就離開） |
|---:|---|---|
| 0 | 開口 | 不讓上船 |
| 1 | 不開口 | 上船（去索寇要塞） |
| 255 | 開口 | 上船 |

255 是這一份程式碼通用的「沒有」哨兵（`LOAD FILES` 的 `FF` 是不載地圖，
見 spec 043），而它同時通得過兩道閘門——所以「清票」不是寫回 0，是寫 255。

費蘭那張選單的兩支各開一半的鎖：

| 選項 | 位址 | 做的事 |
|---|---|---|
| `LIE?` | `AB98h` → `ABBDh` | `4A01 = 255`（清票），**不寫 `4A26`** |
| `TELL THE TRUTH?` | `ABC7h` → `ACE6h` → `AD42h` | 說出 SAMOSUD，`ADAAh` 寫 `4AA7 = 254`（開航線）、`ADA4h` 寫 `4A26 = 255` |

`ACE6h` 的 `COMPARE @4A21, 255 ; IF = ; GOTO AD42` **兩支都走到 `AD42h`**：
沒拿到裝備（`4A21 != 255`）只是多印一句「穿過武器庫的幻牆去找幫手」
（`ACF1h`），SAMOSUD 與那兩個旗標照樣拿得到。裝備不是開航線的前提。

遭遇本身在 `AA02h` 檢查 `4A26 == 255` 就不再出現。說謊不寫 `4A26`，
所以亡魂會再來，說謊之後還能再選說實話。

說謊之後回城區，港務長因為 `4AA7` 還小於 254 走「唯一的船是去索寇要塞」那一支，
`9DCEh` 把 `4A01` 寫成 1；那張票在城區裡有效，走出城區就清掉，下一次港務長照常
開口。所以說謊、說實話、打贏的順序都不影響航線怎麼開（「從要塞坐船回城也經過
清除」是 strong inference：清除段位元組 exact，城區換區實跑過兩種，船那一條沒單獨量）。

## 打贏也開航線

亡魂那一場遭遇（`AA0Ah`，選單結果存 `@6E79`）的 `ON GOTO` 在 `AA3Eh`：

| `@6E79` | 目標 | |
|---:|---|---|
| 0 | `AE0A` | PICTURE 255 + EXIT |
| 1 | `ADB4` | 戰鬥 |
| 2 | `ADE3` | 逃跑 |
| 3 | `AA50` | 交涉（`PARLAY`，spec 078 的結果碼 3）|

戰鬥那一支不是直接開打：

```
ADB4  SAVE 2   → @6E79      ; 戰鬥參數表的索引
ADBA  SAVE 0   → @9808      ; 只有一組怪
ADC0  SAVE 0   → @6DCB
ADC6  GOTO AE5D
```

`AE5Dh` 是這個區塊共用的開戰常式：五個 `GETTABLE` 以 `@6E79` 為索引取出怪物
參數，`LOAD MONSTER`，`COMBAT`，然後 `GOSUB ADD7h`（`@6DC7 > 128` 就跳
`ADE3h`）再 `ON GOTO AED8h`。`@9808 = 0` 讓 `AEB6h` 的 `ADD 1, @6E79` 被跳過，
所以到 `AED8h` 時 `@6E79` 還是 2，而 `AED8h` 的索引 2 就是：

```
ADCA  SAVE 255 → @4A26      ; 亡魂不再出現
ADD0  SAVE 254 → @4AA7      ; 碼頭航線打開
ADD6  EXIT
```

所以**打贏亡魂跟說實話一樣會開航線**，而且不碰 `4A01`。說實話多拿到的是 SAMOSUD——
出要塞時通過守衛要用。


## 晚上鎖門與城衛隊

城區入口 0（每一步都跑）開頭先判斷是不是晚上，再決定面前那道門開不開。
城衛隊那一場 38 隻**不是到點出兵**，每一條進去的路都要玩家選。

### 入口 0 的晚上判斷（exact）

`ecl3.dax` block 0（SHA-256 `b0fe79c5…`，`cmd/pool-ecl-trace -archive 3 -block 0`）：

```
9914  SAVE 0 → @6E7B ; SAVE 11 → @6E7D
9920  COMPARE @49C9, 14 ; IF >= ; SAVE 8 → @6E7D
992D  SAVE @6E7D → @49FD ; SAVE 10 → @49FE
993A  COMPARE @6DD5, 0 ; IF = ; GOTO 9965h     ; 6DD5 另一支（NEWECL 20），不在本節
9965  AND 127, @C04F → @6E82
996E  COMPARE @4ABA, 254 ; IF >= ; EXIT         ; 通關之後不鎖
9976  COMPARE @6E7D, 8 ; IF <> ; EXIT           ; 白天到此為止
997E  COMPARE AND @6E82, 0,  @C04E, 9  ; IF = ; GOTO 99AFh
998E  COMPARE AND @6E82, 4,  @C04E, 7  ; IF = ; GOTO 99AFh
999E  COMPARE AND @6E82, 26, @C04E, 11 ; IF = ; GOTO 99AFh
99AE  EXIT
99AF  PRINTCLEAR "THE DOOR IS LOCKED.  DO YOU WANT TO BREAK IN?"
99D4  GOSUB AE5Ah（YES／NO）→ 99D8 ON GOTO [AD82h ← YES, 99E4h ← NO]
99E4  SAVE 255 → @6DC9 ; EXIT
```

**比較的方向對過原版**：engine 把 `COMPARE a, b` 讀成「a 對 b」，所以 `9920h` 是
`49C9 >= 14`；反過來讀是 `14 >= 49C9`。開局時鐘 0，兩種讀法在城區走一步之後寫進
`6E7D` 的值不同（11 對 8）。dosgolem 在原版從貧民窟走回城區，進城那一步
`49FE` 從 9 變成 10（正對照：位址換算對），同一步 `6E7D = 11`、`49FD = 11`、
`49C9 = 0`——是 engine 的讀法。收據 `docs/audit/dosgolem-city-night-flag.json`，
重生用 `tools/dosgolem-city-night-flag.py`。

`C04E` 是面前那道牆、`C04F` 是這一格的地形（spec 015）。三組條件在 GEO3/0 上
（以格子自己的 `WallDirections` 掃）：

| 地形 | 牆型 | 格子（朝向）| 地點 |
|---:|---:|---|---|
| 26 | 11 | (4,3) 南、(3,4) 東、(5,4) 西 | 市政廳的門（索引 26 是門口的布告，本表上方）|
| 0 | 9 | (14,8) 東、(11,9) 南、(12,9) 南、(13,9) 北、(10,10) 西、(7,11) 東、(10,11) 東、(8,12) 南、(9,12) 北、(10,12) 東與南 | 未對名字 |
| 4 | 7 | 沒有 | —— |

地點名與「牆型 9 的門是哪幾棟」是 `unknown`；地形 4 牆型 7 掃不到，而 remake 投影
`C04E` 用的是 `WallWrapped`（spec 015），是否含相鄰格那一側的牆沒另外掃——
`hypothesis`：那一組在這張圖上不會成立。

remake 實跑（`cmd/pool-game/city_watch_test.go`）：20 點在 (3,4) 朝東踏一步出現
鎖門問句、答 NO 不開打也不進市政廳；6 點同一步直接進 ECL3/8；答 YES 出現
`AD82h` 的城衛隊問句。原版晚上站上那一格的畫面還沒拍，所以「晚上會鎖市政廳」
這一句在原版 runtime 上是 `strong inference`（靜態分支 exact ＋ 比較方向 exact）。

### 城衛隊那一場的五條進入邊（exact）

`ADDFh..AE08h`：`LOAD MONSTER` 94×2、84×12、53×12、40×12 → `COMBAT`
（remake 顯示為 LEVEL 3 MU×2／6TH LVL FIGHTER×12／AIDES×12／NOMAD×12）。
進去的路只有這五條，都是玩家選了才開打。鎖門（YES）與兩個 `STAY／RUN`（STAY）
**第 0 項是開打的那個**，按 ENTER 就開打；驅趕與神殿的第 0 項是安全的，但輪流試
選項的駕駛照樣會試到：

| 來源 | 問句 | 開打 | 不開打 |
|---|---|---|---|
| `99D8h` | 晚上的鎖門 BREAK IN? | YES → `AD82h` | NO → `99E4h` |
| `9AE6h` | 休息被驅趕（入口 3，spec 114）| STAY → `ADDFh` | GO → `AE6Ah` |
| `9E9Ah` | 神殿衛兵擋主教 | FORCE YOUR WAY PAST → `AD82h` | LEAVE → `A059h` |
| `A828h` | 酒館鬥毆後 CITY WATCH CHARGES IN | STAY → `AD82h` | RUN → `AEF4h` |
| `AFDCh` | 瘋子發作 | 直接 `GOTO AD82h` | —— |

`AD82h` 再問一次 `STAY／RUN`（`ADC3h`）：STAY → `ADDFh` 開打，RUN → `AEF4h`。
`AEF4h` 是 `RANDOM 3`、兩張 `GETTABLE`（`B617h`／`B61Bh`）、`CALL 2C90h`——隨機
搬到三個地點之一。打完 `AE13h` 寫 `4AC0 = 1`；緊接著的 `AE1Ah` 是另一支的
"DUE TO YOUR VICIOUS ATTACK… WE REFUSE YOU ALL SERVICE."，誰讀 `4AC0`、誰跳到
`AE1Ah` 沒有讀（`unknown`）。

時刻相關的腳本讀同一個門檻：`ecl3/0 9BAEh`（碼頭的圖）、`ecl2/9 ADAAh`（斯托亞諾夫
城門的馬車商人 `49C9 >= 14` 就 `EXIT`）、`ecl4/21 AE48h`（索寇要塞）。

## 驗收

`TestNormalKeysBuyAndEquipFromTheWeaponShop`（`cmd/pool-game`）只用按鍵從
標題走到武具店：建角拿到金幣 → 走到 (8,11) → 買 → 按 I 裝上。它同時是這張
索引表的實測——路線是照索引 22 的五格規劃的，走到就代表索引讀對了。

晚上鎖門：`TestCityHallIsLockedAtNightAndNoKeepsThePeace`、負對照
`TestCityHallIsOpenInTheMorning`、正對照
`TestBreakingIntoCityHallCallsTheWatchAndRunAvoidsTheFight`（`city_watch_test.go`，
都從 `Update()` 送鍵）；比較方向的原版收據 `docs/audit/dosgolem-city-night-flag.json`。

## OPEN

- 索引 28..35 那 12 格由誰處理。同一張 GEO 上還有 ECL3/8（市政廳）與
  ECL3/11（競技場），兩者都是 `NEWECL` 換進來的，它們自己的入口 1 還沒讀過。
- 牆型 9 的那十一個門面是哪幾棟建築；原版晚上站上鎖門格的畫面（目前只有靜態分支
  與比較方向的 runtime 證據，見〈晚上鎖門與城衛隊〉）。
- 其他區（貧民窟、要塞…）是不是同一套分派。ecl2/20 與 ecl4/21 的入口 1
  還沒逐條讀過。

## 晚上站在市政廳門口會怎樣（2026-09-19 實測，#39）

收據 [`docs/audit/dos-city-night-lock.json`](../audit/dos-city-night-lock.json)
（工具 `tools/dosgolem-city-night-lock.py`，六張畫面與 `49C9`／`6E7D`）：

| 時刻 | 位置 | 按前進之後 | `6E7D` |
|---|---|---|---|
| `49C9 = 0`（白天）| GEO3/0 (3,4) 朝東 | **進得去**：ECL3/8 (4,4)，文字「YOU ENTER THE CITY HALL…」| 11 → 0 |
| `49C9 = 14`（晚上）| 同一格同一個朝向 | **人沒動**，也**沒有問句** | 11 → **8** |

也就是說市政廳那一扇門晚上只是「進不去」，不是 `9920h` 那一支的
「THE DOOR IS LOCKED. DO YOU WANT TO BREAK IN?」——那個問句要面對的是牆型 9 的那十一個門面，
還沒逐一量（#39 仍開著）。

推時鐘的方法也記在這裡，兩條走不通的先寫下來：城區街上紮營會被城衛隊在 00:05 攔下來
（十二次只推進一小時，spec 114）；貧民窟睡得完但睡完會跳遭遇，一人隊伍收不了場。
可行的是**從既有的通關狀態檔接**——`workplace/dosgolem-cheat/chain1-sokal.state`
本來就站在門口、時鐘 13 點、六人隊伍，走幾十步（一步一分鐘）就過 14 點。
