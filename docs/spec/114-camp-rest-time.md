# Spec 114：遊戲時鐘與紮營的休息時間（overlay-20）

狀態：READY（時間的進位規則、紮營挑天／時／分的界面、增減的級距與夾限、
打斷之後交棒給命令集入口 3、
休息回血與記完法術的條件、休息主迴圈與打斷的判定、**打斷那兩個欄位的來源**
都已讀出並實作）。
日期：2026-09-05（2026-09-10 補上打斷欄位的來源）。

## 時間是一個逐位進位的數

`DS:6CB6h` 起七個 word 就是一個時間，每一位的進位上限各不相同，整張表在
`DS:35D4h`：

| 索引 | 位址 | 意義 | 上限 |
|---:|---|---|---:|
| 0 | `6CB6h` | 比分更細的一格 | 10 |
| 1 | `6CB8h` | 分的個位 | 10 |
| 2 | `6CBAh` | 分的十位 | 6 |
| 3 | `6CBCh` | 小時 | 24 |
| 4 | `6CBEh` | 日 | 30 |
| 5 | `6CC0h` | 月 | 12 |
| 6 | `6CC2h` | 年 | 256 |

分成兩位存，所以畫面上的分鐘是 `[2] × 10 + [1]`（`0667h` 就是這樣算的）。

正規化在 **overlay-20 entry 5（`02B1h`）**：由低位往高位掃一遍，某一位到達
上限就 `[i+1]++`、`[i] -= 上限`。每一位只進位一次——原版每加一次就正規化一次，
一次最多只會超過上限一格。最高位（年）溢位時**沿隊伍串列把每個人的 `+30h`
加一**，那是年齡。

## 紮營挑的是一段長度，不是日期

**overlay-20 entry 6（`035Eh`）**在正規化之後多做兩件事，因為玩家挑的是一段
**長度**，畫面上沒有「月」這一欄：

```
0367  entry 5(DS:6CB6h)                 ; 正規化
036a  DS:6CC0h（月）> 0 →
0374    DS:6CBEh（日）+= DS:35DCh × 月   ; 35DCh ＝ 30，就是日那一位的上限
037e    月 = 0
0381  日 > 63h → 日 = 63h                ; 上限 99 天
```

減法在 **entry 7（`0437h`）**：日、時、分的兩位全是 0 就整支返回（減不下去），
否則從那一位借位減。

## 選單

**entry 10（`06E0h`）**是紮營時間的迴圈，選單列的字串在 `069Fh`：
`Rest   daYs Hours Mins   Inc Dec   Exit`。

```
06ea  欄位 = 2                            ; [bp-4]，2 分、3 時、4 日
06f3  entry 9(欄位) 重畫
0728  讀一個鍵
0736  H（上）→ I；P（下）→ D
074d  K（左）→ 欄位++，超過 4 回 2；M（右）→ 欄位−−，小於 2 回 4
077f  Enter → R
078c  R → 回傳 1（開始休息）
0796  Y → 欄位 4；H → 欄位 3；M → 欄位 2
07b8  I → 欄位是分就 DS:6CB8h += 5，否則 DS:[6CB6h + 欄位 × 2]++，再 entry 6
07db  D → 欄位是分就 entry 7(索引 1, 5)，否則 entry 7(欄位, 1)，再 entry 6
0800  鍵在 CS:6C0h 的集合裡就離開迴圈
```

**分鐘一次五分**，天與時一次一。加減都作用在那一欄自己的位上——分鐘加的是
**個位**（索引 1），進位由 entry 6 處理，所以按四次會看到 `05 10 15 20`。

畫面那一列由 **entry 9（`05ADh`）**印：`Rest Time:`（`05A0h`）之後三個兩位數，
選中的那一欄顏色由 `0Ah` 換成 `0Fh`。

## 休息的效果

| 位置 | 週期 | 做什麼 |
|---|---|---|
| entry 11（`0830h`）| `DS:6CCCh` 每滿 `120h`（288）刻 | 沿隊伍串列每人呼叫 overlay-24 entry 21 回一點生命力，印 `The Whole Party Is Healed`，計數歸零 |
| entry 15（`0B63h`）| 每滿 `0Ch`（12）刻 | 每個人的記錄 `+2Ch` 減一，歸零時跑 entry 13／12 把法術記完／抄完，結果 × 3 寫進 `DS:[6CC3h + 序號]` |

12 刻是一小時，所以**一刻五分鐘**——與選單的分鐘級距一致。288 刻就是二十四
小時。

**兩份獨立的來源對上**：中文說明書 p.29 寫「每休息二十四小時各隊員可恢復一點
HP」，而程式裡是 `288 刻 × 5 分 = 1440 分 = 24 小時`。說明書同一段也把按鍵寫得
一模一樣：「你可用 Y、H 和 M 來決定要改變天數、小時數及分鐘數，再按下
INCREASE 增加，DECREASE 減少，當時間調整好了之後，按下 R 鍵即可開始休息」。

## 實作

| 原版 | remake |
|---|---|
| `DS:35D4h` 的進位上限 | `gamepack.ReadDOSTimeRadix`（七個值逐一釘住）|
| entry 5 的正規化 | `gamepack.GameTime.Normalise` |
| entry 6 的折月與夾限 | `gamepack.RestDuration.Increase`／`Decrease` 之後的 `settle` |
| entry 7 的借位 | `GameTime.borrow` |
| entry 10 的欄位與按鍵 | `gamepack.RestField` ＋ `(*app).campInput` |
| entry 9 的那一列 | `(*app).campRestTimeLine`、`drawCamp` |
| entry 11 的回血 | `gamepack.RestHealing`、`(*app).restParty` |
| 休息迴圈 `0D5Eh` 的 entry 2 | `restParty` 裡逐刻 `advanceGameTime(RestMinutesPerTick)`——世界時鐘往前走，身上的效果跟著遞減。**遞減本身是 entry 4**（`0000h`），由 entry 2 在 `042Eh` 用 near call 叫（spec 069）|
| entry 15 的記憶完成 | `restParty` 裡「休息時數 ≥ 各法術等級的總和」才記完 |

旅店不再自動回滿。說明書 p.31 對旅店的保證是「絕對安全而且不會有人中途打擾」
與「你高興休息到什麼時候就待到什麼時候」——**安全與不限時長，不是免費回滿**。
回血一樣照二十四小時一點算。

## 休息的主迴圈與打斷

**entry 3（`0C45h`）**是整段休息：清掉每人的 `DS:6CC3h` 計數、開視窗、畫面、
呼叫 entry 10 讓玩家挑時間，回傳 1（R）est）才進迴圈。回傳值是「有沒有被打斷」。

```
0cd3  停止旗標立著 → 收工
0cdc  剩餘時間（日／時／分兩位）全是 0 → 收工
0cfb  0512h:02FAh 有按鍵 →
0d10    印 "Stop Resting? "（CS:0C15h），問 Y/N
0d28    Y → 停止
0d40  entry 7(索引 1, 5)               ; 剩餘時間減五分鐘
0d4a  每五刻重畫一次
0d5e  entry 2(索引 1, 5)               ; 世界時鐘加五分鐘
0d68  entry 11                          ; 回血檢查（288 刻一次）
0d6d  entry 14
0d76  entry 15                          ; 記憶／抄寫進度（12 刻一次）
0d79  [DS:4937h]+5A4h == 0 → 這一區不會被打擾，繼續
0d85  DS:495Ch++；還沒到 +5A4h → 繼續
0d97  DS:495Ch = 0；擲 骰(1,100)
0dad  擲出 > +5A6h → 沒遇上，繼續
0dd3  印 "The Party is rudely interrupted!"（CS:0C24h），停止並回傳 1
```

所以打斷是**逐區設定的**：`+5A4h` 是每幾刻檢查一次（0 ＝ 不檢查），
`+5A6h` 是 1d100 的門檻。這與說明書 p.29「最安全的休息地點是旅店和已被清除
的地區……而隨便在野外或未清除的地區睡覺是很危險的」相符。

規則實作成 `gamepack.SimulateRest`，兩個參數包在 `RestInterruption` 裡。

## 實跑：貧民區一定會被打斷（2026-09-09）

拿 dosgolem 走「建人類牧師 → 導覽走完 → `E` 紮營 → 排兩小時 → `R`」，
**休息在第五分鐘就被打斷**，畫面是：

```
YOU ARE ROUSTED BY THE CITY WATCH AND TOLD TO MOVE ALONG. WHAT DO YOU DO?
GO   STAY
```

**休息把世界時鐘推進了五分鐘**：紮營前狀態列是 `0, 4 W 00:00`，按下 `R` 之後
是 `0, 4 W 00:05`，也就是被打斷的那一刻正好睡了一刻（一刻五分鐘）。基準畫面在
[`docs/reference/original-dos/camp/`](../reference/original-dos/camp/)
（`00-rest-started-clock-00-05.png` 與 `01-city-watch-interruption.png`；
dosgolem `d351681ba86d`、`start.exe` `12811cbc8166`、2026-09-11，
鍵序尾端是 `…,e,r,h,i,r`——`E)NCAMP`、`R)EST`、選小時欄、加一小時、開始休息）。

那一段是 **ECL3／block 0 的 entry 3**（`0x9A93` 起：`SAVE 0 → @6DD3`、
`GOSUB 0xAF40`、`PRINTCLEAR` 那句話、`HORIZONTAL MENU GO／STAY`、`ON GOTO`），
不是 overlay-20 自己印的 `The Party is rudely interrupted!`。休息迴圈
（`0C45h`）裡沒有任何 ECL 呼叫——`0D69`／`0D6D`／`0D76` 是 entry 11／14／15，
`0DA2`／`0DB4` 是 overlay-24 entry 8 與 overlay-25 entry 21。所以順序是
**打斷 → 回傳 1 → 呼叫端跑 ECL 的紮營入口 → 城衛隊**。

## 那兩個欄位是 ECL 變數（2026-09-10）

`+5A4h`／`+5A6h` 不是「某個結構的欄位」，是 **ECL 記憶體裡的兩個位址**。
`DS:4937h` 是 class 1 的基底，換算式 `[4937h] + 2A00h + addr × 2`（mod
10000h，spec 008／106）把它們換回 ECL 位址 **`6DD2h`／`6DD3h`**。同一條式子
在 class 0 上把 `+1CCh` 換回 `49E6h`，與 spec 074 獨立讀出來的值相同——
式子有兩個方向的對照。

值由 ECL 腳本寫：`cmd/pool-ecl-memory-audit -addresses 6DD2,6DD3` 在
ECL1／2／3 找到 **123 個引用，其中 117 個是 `SAVE`**。本規格先前記的實跑
（貧民區被城衛隊趕起來）那一段 ECL 開頭正是 `SAVE 0 → @6DD3`——打斷過一次
就把門檻清掉。

remake 這一側：`RestInterruptionPeriodAddress`／`…ThresholdAddress` 兩個常數，
`camp.go` 的 `restInterruption()` 從 `eventMachine.Memory` 讀。ECL VM 本來就
在跑那些 `SAVE`，所以值是自然到位的，不需要另外餵。

### 教訓：掃 overlay 的定址形狀，看不到 ECL 寫進去的東西

先前遍尋不著這兩個欄位的 writer，掃法是「全 36 顆 overlay 的所有 disp16
modrm 形狀」——那個掃描面**結構上就涵蓋不到 ECL 腳本**，於是得到「只有
overlay-07 清成 0」這個自洽但不完整的答案，還據此推論過「這一版沒有哪一區
會打斷」（被實跑推翻）。

**規則：一個位址在 overlay 裡找不到 writer 時，先問它是不是 ECL 變數。**
判準是它有沒有落在 `[4933h]`／`[4937h]` 那兩個基底的窗內；是的話換算回 ECL
位址再掃一次 ECL 檔。這條連同它的反方向（ECL 位址的字面值也掃不到引擎的
「基底＋位移」寫入，見 spec 100／125）已經收進 `AGENTS.md` §5。

## 打斷之後跑的是入口 3（2026-09-10）

打斷不是終點，只是把棒子交出去：原版回傳 1 之後，呼叫端跑**目前區塊的命令集
入口 3**。貧民區那一份在 ECL3／block 0 的 `9A93h`，逐條讀出來是

```
9A93  SAVE 0 → 6DD3h        ; 先把這一區的打斷門檻關掉
9A99  GOSUB
9A9D  PRINTCLEAR            ; YOU ARE ROUSTED BY THE CITY WATCH...
9AD7  HORIZONTAL MENU       ; GO / STAY
9AE6  ON GOTO
```

第一條就回答了一個原本會猜錯的問題：**門檻是腳本自己清的**，remake 這一側
不要另外去碰 `6DD2h`／`6DD3h`。清掉之後同一次紮營不會再被打斷第二次。

入口 0 是每格、入口 1 是搜尋，這是第三個。每一張地圖的入口 3 是它自己的紮營
處理，有些地圖那一格是空的、跑起來直接返回——**那也是原版行為，不是缺口**。

remake 這一側：`gamepack.RunInitialSessionCampEntry` 起跑，
`cmd/pool-game/main.go` 的 `beginCampInterruption` 接到搜尋那條路的同一套消費
流程上（入口 3 產生的事件和搜尋一模一樣，文字框與選單不必另寫一份）。
`camp_interruption_test.go` 走玩家真的會走的路釘住它：真的紮營、真的被打斷、
真的按鍵推進，畫面上要出現城衛隊那一句與 `GO`／`STAY`；負對照是這一區不打擾
時（`Period` 為 0，也是原版初值）不該有人來問話。

## entry 14（`0AC0h`）是 `DS:6CC3h` 的另一個 writer（2026-09-11）

休息迴圈每一刻叫一次，形狀是走整條隊伍串列（`DS:5CF4h` 起，`+104h` 接下一位）：

```
0ae8  dec BYTE PTR [i + 6CC3h]         ; 每人一 byte 的計數往下減
0af3  減到 0 而且角色記錄 +2Ch 是 0 →
0b14    呼叫 CS:09A2h（角色遠指標、一個 byte 的輸出參數）
0b1a    回傳 0 → 再呼叫 CS:08E0h（同樣的參數）
0b3a    [i + 6CC3h] = 回傳值 × 3
```

所以 `6CC3h` 有兩個 writer（entry 14 與 entry 15），兩邊都寫 `結果 × 3`；
`+2Ch` 非 0 的人整段跳過。**這一格是什麼仍然未定**——形狀讀出來了，語意沒有。

## 還沒讀
- `DS:6CC3h` 那個每人一 byte 的格子是什麼。writer 是 entry 14 與 entry 15，
  缺的是語意；順帶記一筆——`cmd/pool-ecl-memory-audit -addresses 6CC3` 是零
  引用，**ECL 完全不碰它**，所以它是引擎內部的狀態，不是腳本看得到的旗標。
- `CS:09A2h`／`CS:08E0h` 這兩支算出來的是什麼，以及角色記錄 `+2Ch` 是什麼。
- 索引 0 那一位（上限 10）代表多細的一格；紮營碰不到它。
