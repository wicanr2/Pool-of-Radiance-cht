# Spec 106：ECL 的位址空間是虛擬的

狀態：READY（分類器、四個類別各自的儲存、class 4 已讀出來的特例、載入區塊時清掉的兩段）；
DRAFT（class 1 的 entry 13 分支、class 4 特例清單是否還有沒讀到的、`4959h` 的寫入時機）。
remake 還沒有載入區塊的清除（#41）。

## 一句話

ECL 指令裡的「位址」不是直接的 DS 偏移：直譯器先把它**分類**，再決定要去
哪一塊記憶體、用 byte 還是 word 取。有一批位址根本不是記憶體，是引擎的暫存器。

## 分類器（overlay-07 entry 12，code `070Bh`）

```
預設 class = 4
4900h..4CFFh → 0
6B00h..6EFFh → 1
9700h..98FFh → 2
9900h..B6FFh → 3
```

範圍是逐條 `cmp`，後面的判斷會蓋掉前面的，所以四個區間互不重疊時等價於
一張查表。

## 取值（overlay-07 entry 15，code `0E2Eh`）

| class | 位址範圍 | 取法 |
|---:|---|---|
| 0 | `4900h..4CFFh` | `[4933h] + 6E00h + addr*2`，word |
| 1 | `6B00h..6EFFh` | 先問 entry 13（`07C7h`）；沒答就 `[4937h] + 2A00h + addr*2`，word |
| 2 | `9700h..98FFh` | `[493Bh] − 2E00h + addr*2`，word |
| 3 | `9900h..B6FFh` | `[493Fh] + 6700h + addr`，**byte** |
| 4 | 其餘 | 一串特例，見下 |

class 3 就是**目前這個 ECL 區塊的 payload**——所以 `42h GETTABLE` 讀
`@B019` 讀到的是區塊自己的位元組（spec 105 的四張表就是這樣讀的），
而 `NEWECL` 換掉 payload 之後，同一個位址讀到的是新區塊的內容。
class 0（`4A01h`、`4AA7h`、`4AC4h` 這些主線旗標）與 class 1
（`6DC9h`、`6E12h`、`6E79h`）在另外兩塊，**不隨 `NEWECL` 換掉**，
這就是旗標跨區持續的機制——只有兩段例外：每載入一個 ECL 區塊，引擎會把
class 0 的 `4A00h..4A1Fh` 與 class 1 的 `6E79h..6E82h` 清成 0，見下方
〈載入區塊時清掉的兩段〉。

## class 4：那些不是記憶體的位址

`addr >= C04Bh` 走另一支：`C04Bh + n` 直接對到 `DS:6A0Bh + n`
（X、Y、朝向、牆、terrain，spec 015）；其中 `C04Dh` 取的是
`DS:6A0Dh / 2`，所以 ECL 看到的朝向是 0..3 而引擎存的是 0..7。

`addr < C04Bh` 是一串 `cmp`：

| ECL 位址 | 對到 | 寬度 |
|---|---|---|
| `00B1h` | `DS:6D34h` | word |
| `00FBh` | `DS:6D30h` | word |
| `00FCh` | `DS:6D32h` | word |
| `033Dh` | `DS:6A0Dh` | byte（朝向，0..7）|
| `035Fh` | **沒有 case body** | — |

`035Fh` 那一格的位元組是 `3d 5f 03 75 00`——`cmp ax, 035Fh` 之後
`jne` 跳到**下一個位元組**，也就是一個空的 case。函式回傳的是區域變數
`-2(bp)`，而那個變數在這一條路徑上**從頭到尾沒有被寫過**。

## 對 remake 的意思

- class 0..3 用一張平坦的 map 模擬是等價的：位址不重疊，寬度差別只影響
  取出來的值域。class 3 那一塊共用 engine 已經接對了——`eclvm.NewMachine`
  在建機器時就把 payload 逐位元組寫進 `Memory[codeBase+index]`，
  `SwitchBlock` 換區時先把舊 payload 的位址刪掉再寫新的，所以
  `42h GETTABLE` 讀 `@B019` 讀到的確實是目前這個區塊的位元組。
- class 4 的那幾格要當成**引擎暫存器**：`033Dh` 是朝向、`00FBh`／`00FCh`
  是引擎的兩個 word。remake 目前把它們放在同一張 map 裡，前端寫、ECL 讀，
  行為等價。
- **`035Fh` 在原版沒有儲存**。野外的可通行判定（spec 105）拿它跟兩張表比，
  比的是一個沒有初始化的堆疊區域變數。remake 的 map 讓它一直是 0，
  而 0 不在任何一張表裡——所以「remake 永遠不擋」與原版「看堆疊剩什麼」
  一樣沒有可靠的擋路行為。這一格**不是 remake 少做了什麼**。

## 還沒讀

- entry 13（`07C7h`）在 class 1 上先答了什麼。
- class 4 的特例是不是還有沒被 `cmp` 掃到的（目前只讀了取值那一支；
  寫值那一支還沒逐條讀，但全部 38 個 overlay 裡 `cmp ax, 035Fh` 的
  位元組樣式只出現這一次）。

## class 0 就是那塊 0x800 位元組的隊伍記錄

`[4933h] + 6E00h + addr*2` 在 16 位截斷之下，把 class 0 的兩端算出來：

| ECL 位址 | 偏移 |
|---|---|
| `4900h` | `0000h` |
| `4A00h` | `0200h` |
| `4A01h` | `0202h` |
| `4A77h` | `02EEh` |
| `4CFFh` | `07FEh` |

正好鋪滿 **0x800 個位元組**，而那就是 `[DS:4933h]` 指到的隊伍記錄——
全域初始化在 `overlay-11 026Dh..027Ah` 把它清成 0（spec 009）。
**開機清一次；之後除了 `4A00h..4A1Fh` 這 32 格，不隨載入區塊重設。**

所以 class 0 是**整份存檔共用**的，不是每張圖各一份。remake 用一張平坦的 map
是對的——但 map 的前 32 格是區塊暫存，要在載入區塊時清掉（下一節）。

這也解釋了一個看起來像 bug 的行為：**同一個 class 0 位址會被不同區塊當成
自己的變數**。`4A00h` 在 `ecl2/9`（斯托亞諾夫城門）是「馬車出現過沒有」
（`ADBD COMPARE @4A00 0 / IF <> / EXIT`），在 `ecl4/10`（瓦海登墳場）是
`9AD6 ADD 1 @4A00` 的計數，`ecl4/21`／`ecl2/20` 又拿它 `SAVE 255`。
這些位址全落在 `4A00h..4A1Fh`：原版在載入下一個區塊時就清掉，所以它們是
**同一個區塊內**的暫存，墳場的計數帶不到城門。

（推論等級：算式與 0x800 的邊界是 exact，逐位元組對上。`[4933h]` 記錄的
寫入點至少還有下一節那一個，其餘沒有逐條掃過整份執行檔。）

## 載入區塊時清掉的兩段

overlay-07 entry 3（`01C8h`，ECL 區塊載入的初始化；overlay-07 SHA-256
`a59f9d16…78ae`，IDA 9.4，overlay 檔內偏移）：

```
02D8  80 3E 59 49 00      cmp  byte [4959h], 0
02DD  75 4C               jnz  032B              ; 旗標非 0：跳過清除
02DF  C7 46 FD 01 00      mov  [bp-3], 1         ; i = 1..20h
02E9  8B 46 FD            mov  ax, [bp-3]
02EC  05 FF 49            add  ax, 49FFh         ; addr = 4A00h..4A1Fh
02EF  D1 E0               shl  ax, 1
02F1  C4 3E 33 49         les  di, [4933h]
02F5  03 F8               add  di, ax
02F7  31 C0               xor  ax, ax
02F9  26 89 85 00 6E      mov  es:[di+6E00h], ax ; class 0 = 0
02FE  83 7E FD 20         cmp  [bp-3], 20h
0302  75 E2               jnz  02E6
0304  31 C0 / 89 46 FD    i = 0..9
030E  8B 46 FD / 05 79 6E add  ax, 6E79h         ; addr = 6E79h..6E82h
0313  D1 E0 / C4 3E 37 49 les  di, [4937h]
031A  03 F8 / 31 C0
031E  26 89 85 00 2A      mov  es:[di+2A00h], ax ; class 1 = 0
0323  83 7E FD 09 / 75 E2
0329  EB 05               jmp  0330
032B  C6 06 59 49 00      mov  byte [4959h], 0   ; 旗標只擋一次
```

- **清的範圍**：class 0 `4A00h..4A1Fh`（32 格），class 1 `6E79h..6E82h`（10 格）。
  exact（位元組）。
- **`4959h` 擋一次**：非 0 時跳過清除並把它歸零。寫 1 的是 overlay-16
  `03EFh`（建隊選單那一支，推測是讀檔之後第一次載入區塊，讓存檔裡的暫存
  存活），寫 0 的另有 overlay-11 `038Ch`。旗標語意是 strong inference。
- **實跑收據**：`tools/dosgolem-4a01-block-load.py` 從貧民窟狀態走進市政廳找職員
  再走出來，對 `2EA2:0202`（class 0 `4A01h`）設寫入監看，量到兩筆寫入：
  職員 `00→01`（`1997:0CBF`，ECL `SAVE`）與走出市政廳 `01→00`
  （`1997:02FE`，即上面的 `02F9` 指令之後）；`1997` 段經核對是 overlay-07。
  收據在 `docs/audit/dosgolem-4a01-block-load-clear.json`（dosgolem `b4d5a3f`）。
  「每一次 `NEWECL` 都經過 entry 3」只量了這一個換區，是 strong inference。
- **remake 現況**：共用 engine 的區塊切換沒有給 title 的載入 hook，Pool adapter
  也沒有清這兩段，於是 `4A01h` 留在 1——職員走 BACK SO SOON、港務長不開口。
  修正追蹤在 issue #41。
