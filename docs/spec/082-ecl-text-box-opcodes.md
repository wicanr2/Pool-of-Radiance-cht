# Spec 082：文字框的四條 opcode（`11h`、`12h`、`33h`、`3Dh`）

狀態：READY（四條都接上；分頁的分隔字元仍未讀，見「還沒讀」）。
日期：2026-09-09（原 2026-09-03；`11h`／`12h` 的區分於 09-04 加入，本輪讀出停頓點由腳本的單選項選單決定）。

## `33h PRINT RETURN`（overlay-03 `2E44h`，0 個運算元）

```
2E47h  [494Eh]++                       ; PC 走過這個位元組
2E4Bh  [82A5h] 非 0 時：
         [5D82h] = 1
         [5D83h]++
         [82A5h] = 0
       否則：
         [5D82h] = 1
         [5D83h]++
```

兩支的差別只有清不清 `82A5h`，共同的動作是 `[5D82h] = 1` 加 `[5D83h]++`——
一個旗標加一個計數器，計數器每呼叫一次加一。這是**文字游標換到下一行**。

## `3Dh CLEAR BOX`（overlay-03 `2E6Fh`，0 個運算元）

```
2E72h  [494Eh]++
2E76h  呼叫 0198h:0066h，參數 (1, 1, 0Fh, 0Fh)
2E87h  [82A4h] = 1
```

同一支 `0198h:0066h` 也被 `39h WHO` 用，參數換成 `(1, 11h, 26h, 16h)`——
兩組都是「左、上、右、下」的框，所以那支是**把一塊矩形清掉**。
`82A4h` 是「框已經清過」的旗標。

## `11h PRINT` 與 `12h PRINTCLEAR` 是兩件事

共用 VM 對這兩條走同一支 handler（`eclvm/machine.go` 的 `case 0x11, 0x12`），
只把文字包成事件送出去；**分頁的語意在消費端**，而且兩者不同：

- `12h PRINTCLEAR`：新的一頁。
- `11h PRINT`：**接在目前這一頁後面**，原版靠它把一句話拼起來。

證據是兩處指令序列，中間都沒有任何等待玩家的指令：

```
ecl7/23  A4A7 PRINTCLEAR "DO YOU REALLY MEAN"
         A4B9 PRINT      6E79h            ; 玩家剛打進去的字
         A4BD PRINT      "?"
         A4C1 GOSUB      9982h            ; [YES NO]

ecl3/0   AC22 PRINTCLEAR "PROCLAMATIONS ARE POSTED ON THE WALLS, IN YOUR JOURNAL YOU NOTE"
         AC55 COMPARE    4AC1h #0
         AC5C GOTO       AC9Bh
         AC9B PRINT      "PROCLAMATIONS LXIV, LXXVIII, CIX, AND LIX."
```

兩段都要拼起來才成句。把 `11h` 也當成取代的話，密碼確認框只會剩下一個
`?`，市政廳的布告則只剩後半句。

**兩段之間原版沒有空白**：四段文字的原始 bytes 解出來，`…YOU NOTE` 以 `E`
結尾、`PROCLAMATIONS LXIV…` 以 `P` 開頭，`DO YOU REALLY MEAN` 以 `N` 結尾、
`?` 就是問號本身，前後都不帶空格。直接串接會得到 `YOU NOTEPROCLAMATIONS`，
所以分隔由呈現層補：`joinPrintedText` 補一個空格，新片段以標點開頭時不補。

## remake 怎麼接

兩條都走 passthrough 讓前端拿到事件。共用 VM 的 `RunUntilEvent`
**一遇到事件就返回**，所以每個 result 通常只帶一則——文字框的狀態因此要
跨 result 留著，不能每次從 result 重建：

- `3Dh CLEAR BOX` 清掉整個框。
- `33h PRINT RETURN` 在框尾加一個換行。
- `11h PRINT` 接在目前這一頁後面（`joinPrintedText`）。
- `12h PRINTCLEAR` 是新的一頁；上一則以換行收尾時才續行。

`12h` 那條的「換行才續行」是刻意的。原版靠什麼在兩頁之間清框還沒讀出來
——市政廳那幾個 block（ECL3）根本沒有 `3Dh`——而逐頁取代已經對過原版
（spec 015／016）。有 `33h` 的地方接得起來，沒有 `33h` 的地方維持原樣。

同一個 result **只能套用一次**：`33h` 之後框的內容會隨套用次數改變。
`consumeInitialSearch` 套過的 result 交給 `pauseAppliedCellResult`，
那一份不再套。

### 哪一則會停下來等玩家

**往文字框寫字不是停頓點。** `11h PRINT` 與 `12h PRINTCLEAR` 都只是把字送
進框裡（前者接續、後者換頁），兩者之間沒有等待指令，所以 remake 拿到這種
result 直接往下跑。

**要玩家按一下的時候，腳本自己會放一個只有一個選項的 `HORIZONTAL MENU`。**
市政廳外那一格（ECL3/block0）的序列是：

```
ABD2 PRINTCLEAR  「YOU ARE OUTSIDE THE CITY HALL. …」
AC1E GOSUB AF1C  → HORIZONTAL MENU，唯一的選項是字串
                    「PRESS <RETURN> OR BUTTON TO CONTINUE」
AC22 PRINTCLEAR  「PROCLAMATIONS ARE POSTED ON THE WALLS, …」
AC9B PRINT       「PROCLAMATIONS LXIV, LXXVIII, CIX, AND LIX.」
ACBE EXIT
```

overlay-03 `11B1h` 在選項數是 1 的時候，把那個選項換成
`PRESS <ENTER>/<RETURN> TO CONTINUE` 再畫（字串在 overlay-03 `1165h`／
`118Ah`，只被 `120Ch`／`1218h` 引用）。**所以「這一頁要不要等」是腳本決定
的，不是繪製端按頁數自動加的。** 玩家因此在市政廳外只按一次 Return，而
文書官辦公室裡每一則委託都要按一次——那裡每一頁後面都跟著一個同樣的單選項
選單。

同一則文字也可以帶著自己的選單：蘇恩神殿的
`12h 問候` → `11h 「DO YOU SEEK HEALING?」` → `YES NO` 是三個相鄰邊界，
中間沒有等待指令，所以走進那一格就同時看到整段問話與選項。

**兩條路徑共用同一個判準。** 每走一步跑的入口 0（`RunInitialSessionCellEntry`）
與 `SearchLocation` 都會拿到這種 result，remake 用同一個 `presentationBoundary`
判斷：`11h`、`12h` 與同屬純顯示的 `0Dh APPROACH`、`0Eh PICTURE`（spec 117：
兩者都是重畫，overlay-03 `07E1h`／`0822h` 做完就返回）算純顯示；帶著選單、
開戰、寶物、換區塊或挑角色的那一則不算，那些要前端接手。分成兩份寫過一次，
結果只有 `SearchLocation` 那條改對，走路那條照樣每則停一次。

**腳本跑完（`00h EXIT`）時文字要留在框裡。** 原版那時底下換回指令列、方向鍵
走得動，而那幾行字還在（dosgolem 實測：市政廳外第一段時按方向鍵一格都不動，
按 Return 換到第二段之後就走得動）。remake 的 `finishCellBlockKeepingText`
就是這個狀態，畫面識別字是 `adventure-cell-done`。

## 還沒讀

- **`11h` 兩段之間原版怎麼分隔**。原始文字前後都不帶空白，所以分隔一定由
  繪製端補；remake 目前補一個空格（標點開頭不補），但原版也可能是換行。
  要定案需要那兩格的原版畫面，`layout-reconstructed`。
- 兩頁之間是誰清的框。`11h` 已經排除——它是接續，不是清框。剩下的候選是
  `1Ah` 那一族或選單常式自己清。
- `5D82h`／`5D83h`／`82A4h`／`82A5h` 這組換行計數器的讀取端。停頓已經確定
  由腳本的單選項選單決定，所以它們不是停頓判準；剩下的用途候選是捲動與
  「印滿一框自動換頁」。
- `0198h:0066h` 的實際繪製行為（remake 的文字框沒有畫素級對拍）。
