# Spec 135：原版紮營畫面的版面

狀態：READY（幾何、營火圖與動畫時序、整棵選單樹、每一層的字串位址、
`Game Speed` 的公式）。
日期：2026-09-09。

## 輸入

原版畫面：dosgolem 走到「導覽走完 → `E`（紮營）」與再按 `R`（REST）的兩幀
（`workplace/dosgolem-ref-camp/58-I.idx`、`59-R.idx`，320×200 的 EGA 色號陣列）。

鍵序（`tools/dosgolem-reference.sh` 的 `POOL_DOSGOLEM_KEYS`）：

```
rep:9:Space,Return,Return,c,rep:5:End,Return,Return,Return,Return,Return,
Return,y,H,E,R,O,Return,k,e,y,a,a,e,b,rep:18:Return,e,H,I,I,R
```

## 紮營不是彈出選單，是換一組指令列

這是與 remake 現況最大的差別。原版按 `E` 之後：

- **第一人稱視野換成一張營火圖**（框不動，換的是框內的 88×88）。
- **時鐘那一行後面接上 `CAMPING`**：`0,4 W 00:00 CAMPING`。
- **指令列整列換掉**，變成 `CAMP: SAVE VIEW MAGIC REST ALTER EXIT`，
  首字母亮白、其餘亮綠，前綴 `CAMP:` 是亮洋紅——與冒險畫面同一種畫法
  （spec 123／129）。
- 對話框區印一行 `THE PARTY MAKES CAMP...`。

再按 `R`（REST）：

- 指令列換成 `REST  DAYS HOURS MINS  INC DEC  EXIT`。
- 對話框區第一行換成 `REST TIME:  00:00:00`。

**沒有任何直排選單**，隊伍表也沒有被蓋住。

## 幾何（exact，量自色號陣列）

單位是 native 像素。

| y | x | 內容 |
|---|---|---|
| 16..22 | 137..300 | 隊伍表標題 `NAME` / `AC` / `HP` |
| 32..38 | 137..278 | 第一位隊員那一列 |
| 120..127 | 138..287 | 時鐘行 `0,4 W 00:00 CAMPING` |
| 136..142 | 18..158 | 對話框第一行（`REST TIME:  00:00:00` 在這裡）|
| 144..150 | 18..188 | 對話框第二行（`THE PARTY MAKES CAMP...` 在這裡）|
| 192..198 | 9..295 | 指令列 |

指令列的三種顏色：前綴 `CAMP:` 色號 13（亮洋紅，x 9..36）、各命令首字母
色號 15（亮白）、其餘字母色號 10（亮綠）。

營火那張圖畫在框內，與第一人稱視野同一塊：`(24,24)` 起 88×88（spec 047）。

## 營火那張圖：`PIC<區號>.DAX` 區塊 29，兩張的動畫

拿 `58-I.idx` 的 `(24,24)` 88×88 與八個 `pic*.dax` 的每一張逐格比對
（`workplace/campscan`，PIC 容器的版面見 spec 117）：

| 檔 | 區塊 | 張 | 相同 |
|---|---|---|---|
| `PIC1..8.DAX` | 29 | 0 | 91.27% |
| `PIC1..8.DAX` | 29 | **1** | **100.00%** |

區塊 29 只有兩張，差的 8.73% 就是火焰跳動的那幾格。八個檔的區塊 29 內容
相同——營火不分區，但原版仍是照現行區號（`DS:52D4h`）組檔名載入的
（spec 117 的同一條路）。基準幀拍到的是第 1 張。

## 選單樹（overlay-15 entry 1，`1E45h`）

紮營的主迴圈是 **overlay-15 entry 1**。它進場時把 `ds:4954h` 設成 2、離場還原
——時鐘行後面接 `CAMPING` 就是看這個。每一層都只換最下面那一列，畫面其餘
部分不動；離開一層就回上一層。

```
CAMP: SAVE VIEW MAGIC REST ALTER EXIT          前綴 overlay-15 1E31h，那一列 DS:051Bh
  1..6  換「目前角色」（ds:5CF0h/5CF2h）        overlay-19 entry 10
  S     存檔（overlay-17 entry 11）
        存完問 `Quit TO DOS `（overlay-15 1E38h），Y → far 26Bh:0（回 DOS）
  V     人物資料頁（overlay-19 entry 5，spec 130）
  M     MAGIC 層（overlay-15 13ECh ＝ entry 16），那一列 DS:0544h
        `Cast Memorize Scribe Display Rest Exit`
  R     排時間層（overlay-15 76Ch ＝ entry 10），那一列 overlay-20 0698h
  A     ALTER 層（overlay-15 1C29h）
  E     離開
```

```
Alter: ORDER DROP SPEED ICON PICS EXIT         前綴 overlay-15 1BE0h，那一列 DS:056Eh
  O   Party Order（overlay-15 174Eh ＝ entry 19）
      前綴 overlay-15 170Eh，兩步的字串是 DS:0598h `Select Exit` 與
      DS:05C1h `Place Exit`（每筆 41 bytes，索引就是第幾步）
  D   Drop（overlay-15 18B6h）——丟的是**目前角色**，不另外挑人
      隊伍只剩他一個時改問 `quit TO DOS: `（18 48h）
      否則印 `<名字> will be gone`（1856h）再問 ` Drop from party? `（1863h）
      丟完依角色記錄 `+10Dh` 換收尾：非 0 是 ` bids you farewell`（1875h），
      0 是 ` is dumped in a ditch`（1887h）
      **答否**（`19DAh`）印的是第三句 ` Breathes A sigh of relief`（189Ch）
  S   Game Speed（overlay-15 1A67h）
      上面一行是 `Game Speed = ` ＋ 值 ＋ ` (0=fastest 9=slowest)`（1A00h／1A0Eh），
      值在 ds:4943h；那一列是組出來的——**值是 0 就不列 ` Faster`、是 9 就不列
      ` Slower`**（1AD9h／1AF1h 的兩個比較），最後接 ` Exit`
      前綴 1A3Bh `Game Speed:`
  I   Icon（overlay-16 entry 4）——對目前角色開戰鬥造形編輯器
  P   PICS 層（overlay-15 1CD2h），沒有前綴
      兩個開關直接畫成那一列：`Monsters on `／`Monsters off `（1BE8h／1BF6h）
      與 `Portraits on `／`Portraits off `（1C05h／1C14h），再接 `Exit`（1C24h）
      按 M 翻 ds:4957h、按 P 翻 ds:4956h
  E   回上一層
```

**每一層都用同一個選單元件**（overlay-26 entry 3 ＝ `00E4h`，spec 133）。
`Exit` 不在呼叫端的分派裡：元件按到它時回 `#0`，呼叫端用
`Pos(#0, <結束集合>)` 判斷要不要離開這一層——結束集合就是一個 `#0`
（overlay-15 `1E11h`／`1BC0h`）。

## `Game Speed` 是遊戲裡「等一拍」的統一單位

`ds:4943h` 不只是紮營畫面上的一個數字——**它是全遊戲延遲的來源**。
overlay-37 entry 13（`0C83h`）就是那一支：

```
mov al, ds:4943h    ; 遊戲速度
xor ah, ah
mov cx, 0E1h        ; 225
imul cx
push ax
call far 512h:29Eh  ; resident 的 Delay（參數是毫秒）
```

呼叫它的有 overlay-03（**ECL 分派器**）、overlay-10、12、13（戰鬥）、18（結局）、
19、20（休息）、22（法術效果）、24、25、32——**導覽每走一步的等待也是它**。

初始值是 **4**（overlay-11 `03BEh` 的 `mov byte ptr ds:4943h, 4`），所以預設
一拍是 `4 × 225 = 900 ms`。玩家嫌慢就是用 `CAMP → ALTER → SPEED` 調——那正是
這個選項存在的理由。

另有一支 `× 10`（overlay-37 `059Ch`，條件是 `ds:4961h != 0`），那是更短的一種
等待，還沒查是哪裡在用。

## 營火的動畫時序

選單元件等鍵的那個迴圈同時在推動畫（overlay-26 `02A0h` 一帶）：

```
畫第 ds:6A1Dh 張 → 算門檻 = 那一張的延遲 ÷ 7 → 目前計時 − 上次換張 > 門檻？
→ 是就 inc ds:6A1Dh，超過 ds:6A1Ch（張數）就繞回 1 → 沒按鍵就回頭
```

**張號是 1-based，而且繞回 1，不是 0。** 計時來自 overlay-37 entry 10 的
32-bit 值（BIOS tick）。

「那一張的延遲」就是 PIC 每張前面那 **4 bytes 前綴**——spec 117 只說有這四個
位元組，沒說是什麼。營火（區塊 29）兩張都是 **2**，`2 ÷ 7 = 0`，所以是
**每過一個 BIOS tick 就換**（18.2 Hz）。

除法而不是乘法是從值反推的：同一個容器裡的船（`PIC3.DAX` 區塊 41）延遲是
20／15，除以 7 是兩個 tick（0.11 秒，船在晃），乘以 7 會變成七秒一格。

## 四條指令列字串在哪裡

長度 byte 的位址（Turbo Pascal 字串，字串本體在下一個位元組）：

| 字串 | 位址 |
|---|---|
| `Camp:` | overlay-15 `1E31h` |
| `The party makes camp...` | overlay-15 `1E03h` |
| `Save View Magic Rest Alter Exit` | `DS:051Bh` |
| `Cast Memorize Scribe Display Rest Exit` | `DS:0544h` |
| `Rest Time` | overlay-20 `05A0h` |
| `Rest daYs Hours Mins Inc Dec Exit` | overlay-20 `069Fh`（spec 114 已記）|
| ` camping` | overlay-25 `2931h` |
| ` search` | overlay-25 `2939h` |

`DS` 的基準：`Area Cast View Encamp Search Look` 在 `START.EXE` 檔案位移
31867 而 spec 123 量到它是 `DS:04CAh`，所以 `DS` 段從檔案 30641 起；
`Cast View Encamp Search Look` 在 31908 ＝ `DS:04F3h`，對得上。

時鐘行後面接的那個字來自 overlay-25——休息主迴圈的 `0DB4 call far 10Ah:89h`
就是 overlay-25 entry 21（spec 109 的 stub 表：`010Ah` ＝ overlay-25），
` camping` 與 ` search` 是同一組的兩個。

## remake 現況：照這一份改好了

`docs/screenshots/pool-remake-chinese-camp.png` 與 `-camp-rest.png` 是實拍。
`E` 之後視野換成營火、時鐘行接上「紮營中」、指令列換成
`紮營：S 存檔 V 檢視 M 法術 R 休息 A 調整 E 離開`，對話框印
「隊伍紮下營來……」；`R` 再換成 `R 休息 Y 天 H 時 M 分 I 增 D 減 E 離開`
與「休息時間」那一行。**直排選單與壓在視野框上的那三行都拿掉了。**

改動連帶修好了指令列的高亮規則：原版**高亮的是大寫的那個字母，不是第一個
字母**——原版字型只有大寫字模，所以 `daYs` 顯示成 `DAYS` 而白色的是 `Y`
（native 量到格 7、8 綠、格 9 白、格 10 綠）。冒險畫面那一列的大寫剛好都在
字首，所以先前的「首字母高亮」在那裡看不出錯。

`M`（MAGIC）現在是紮營的**子層**，不再是「關掉紮營再開法術一覽」——
關掉法術一覽會回到紮營，與原版同一個形狀。

## 契約

1. `assets.ReadCampFire(zip, 區號)` 讀 `PIC<區號>.DAX` 區塊 29 的兩張，
   畫在第一人稱視野的內框 `(24,24)`（logical `(48,70)`），**每三個影格換一張**
   （60 fps 下的 50 ms，最接近原版那一個 BIOS tick 的 54.9 ms）。
2. 紮營時時鐘行接上 `CAMPING`（繁中另譯）。
3. 指令列換成 `CAMP:` ＋ `SAVE VIEW MAGIC REST ALTER EXIT`，按 `R` 之後換成
   `REST  DAYS HOURS MINS  INC DEC  EXIT`，畫法與冒險畫面同一支
   （前綴一色、首字母一色、其餘一色）。
4. 對話框印 `THE PARTY MAKES CAMP...`；`REST` 之後第一行是
   `REST TIME:  00:00:00`。
5. **拿掉直排選單**與壓在視野框上的那三行。

## remake 現況

整棵樹都接好了。

**`ALTER → ICON`** 對目前角色開的就是建角那一頁——原版兩處也是同一支
（overlay-16 entry 4）。編輯器本身讀寫 `creation.Flow`，所以進去時把角色的
四個造形欄位**連同種族**載回 flow（`UsesIconSizeMenu` 看的是 flow 的種族，
載錯的話小種族的 `SIZE` 那一項會消失），離開時寫回角色並把 flow 還原成
借用之前的樣子。頂層的 `EXIT` 在紮營裡不進「這個造形可以嗎」那一頁——
那一頁是建角流程的最後一步。

**`ALTER → SPEED`** 現在真的有作用：`speedDelayTicks()` 就是
`速度 × 225 ms` 換算成 60 Hz 的影格數，導覽每走一步等的就是它。先前那個寫死
的 `tourStepDelayTicks = 9` 註解自承是 approximation——原版的 DELAY 讀出來
之後就不需要近似了。**預設 4 是 900 ms，導覽因此比先前慢六倍**，那是原版的
速度；截圖腳本等導覽走完的輪數要跟著放大。

## OPEN

角色記錄 `+10Dh`（換 ` bids you farewell` 還是 ` is dumped in a ditch`）
讀成「還活著」是**推論**，還沒逐位元組確認。

overlay-37 `059Ch` 那一支 `速度 × 10` 的短等待是誰在用還沒查。
