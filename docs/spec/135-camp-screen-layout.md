# Spec 135：原版紮營畫面的版面

狀態：READY（幾何、營火圖來源、四條指令列字串都定位了）；
OPEN（`SAVE`／`VIEW`／`ALTER` 三項按下去做什麼）。
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
   畫在第一人稱視野的內框 `(24,24)`（logical `(48,70)`）。
2. 紮營時時鐘行接上 `CAMPING`（繁中另譯）。
3. 指令列換成 `CAMP:` ＋ `SAVE VIEW MAGIC REST ALTER EXIT`，按 `R` 之後換成
   `REST  DAYS HOURS MINS  INC DEC  EXIT`，畫法與冒險畫面同一支
   （前綴一色、首字母一色、其餘一色）。
4. 對話框印 `THE PARTY MAKES CAMP...`；`REST` 之後第一行是
   `REST TIME:  00:00:00`。
5. **拿掉直排選單**與壓在視野框上的那三行。

## OPEN

**營火兩張怎麼交替、多久換一次還沒讀出來。** remake 目前只畫第 0 張——
猜一個閃動速度會讓畫面每一格都不一樣，而那是編的。

`SAVE`／`VIEW`／`ALTER` 三項按下去做什麼還沒讀。`VIEW` 大概是人物資料頁
（remake 的 `V` 已經有），`SAVE` 是存檔，`ALTER` 未知。在讀出來之前這三項
**照原版列在指令列上**，按下去給一句「還沒接」——列都不列的話，玩家看到的
指令列就與原版不同，而那正是這一份要修的東西。
