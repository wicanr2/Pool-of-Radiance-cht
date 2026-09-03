# Spec 099：搭船旅行與它的進度閘門

狀態：CONFORMED（選單、閘門、派發表、旗標寫入點都逐條讀自原始 ECL）；
日期：2026-09-03。

## 為什麼需要這一段

remake 目前只走得到兩張圖：訓練所那一區（GEO3 block 0）與索寇要塞
（GEO4 block 21）。兩張圖都已經逐格踩過（226 ＋ 222 格），上面沒有別的出口。
問題是：**其餘 25 個有文字的 ECL block 為什麼走不到。**

答案不是「城區之間的移動還沒接上」。整套機制原版就寫在 ECL 裡，remake 也
已經接得起來——**擋住的是主線進度**。

## 證據

輸入：`Pool of Radiance (1988).zip` 的 `poolrad/ecl3.dax` block 0 與
`poolrad/ecl4.dax` block 21，以 `cmd/pool-ecl-trace` 靜態解碼。
座標繞回的部分讀自 overlay-30 `0358h`（`objdump -b binary -m i8086`）。

## 碼頭問的是哪四個地方

ecl3 block 0 `9C67h` 印「THE HARBOR MASTER TELLS YOU BOATS LEAVE FOR THE
WEST, THE EAST, SOKAL KEEP, AND ...」，接著 `9CF0h` 是一條
`2Bh HORIZONTAL MENU`：

```
9CF0  HORIZONTAL MENU  ["SOKAL", "EAST", "WEST", "BAY", "NONE"] → DS:4AC4h
```

選完之後 `9BC2h` 印「YOU BOARD A BOAT.」，`9BE3h` 起依 `DS:4AC4h` 派發：

| `DS:4AC4h` | 目的地 |
|---:|---|
| 0 | `DS:6E12h ← 4`，`NEWECL 21`（索寇要塞）|
| 1 | `DS:6E12h ← 8`，`NEWECL 27` |
| 2 | `DS:6E12h ← 7`，`NEWECL 26`（`DS:49C3h ← 7`、`DS:49C4h ← 29`）|
| 3 以上 | `DS:6E12h ← 7`，`NEWECL 26`（`DS:49C3h ← 13`、`DS:49C4h ← 27`）|

`DS:6E12h` 就是 ECL 的 archive 選擇器，remake 的
`syncArchiveFromEventMachine` 已經在讀它。

## 閘門：`DS:4AA7h` 沒到 254 就只有一條船

那張選單前面有一道比較：

```
9C5C  COMPARE DS:4AA7h, 254
9C62  IF <
9C63  GOTO 9D54h
9C67  PRINTCLEAR "THE HARBOR MASTER TELLS YOU BOATS LEAVE FOR..."
9CF0  HORIZONTAL MENU  SOKAL / EAST / WEST / BAY / NONE
```

`DS:4AA7h` 小於 254 就跳過整段，走 `9D54h` 那一支——玩家看到的是
「'BY ORDER OF THE CITY COUNCIL,' THE HARBOR MASTER SAYS, 'THE ONLY BOAT
OUT IS GOING TO SOKAL KEEP. YOU CAN CATCH IT AT THE END OF THE PIER.'」。

**寫這個旗標的只有索寇要塞**：ecl4 block 21 `ADAAh` 與 `ADD0h` 各有一條
`SAVE 254, DS:4AA7h`。也就是說，**清掉索寇要塞才會開出其他航線**。

## 座標是繞回去的，不是走出地圖

起點 (0,4) 的文字是「YOU ARE BY THE GATEWAY TO THE UNSETTLED AREAS」，
容易被讀成「往西走就出城」。**不是**：overlay-30 `0358h`（牆查詢，
`0DB7h` 的移動服務呼叫它）在查表之前先把座標夾回去——

```
383  cmp byte [bp+0Ah], 0Fh ; jle → 38D
389  mov byte [bp+0Ah], 0        ; X > 15 → 0
38D  cmp byte [bp+0Ah], 0  ; jge → 397
393  mov byte [bp+0Ah], 0Fh      ; X < 0 → 15
397..3A5                          ; Y 同樣處理
```

所以**牆的查詢**兩邊一樣繞回，換圖一律由格子事件的 `NEWECL`／`LOAD FILES`
完成。但這只證明了「查牆時座標會夾回去」——引擎有沒有另外記下「這一步本來
會走出地圖」是另一件事，見 spec 100：貧民窟與回程船都掛在 `DS:6DD5h` 上，
而那個變數最可能就是這個訊號。

## 對 remake 的意思

1. 機制已經在了：`2Bh HORIZONTAL MENU` 由共用 VM 處理並回報選項，
   `NEWECL` 與 archive 切換也接好了。**不需要新功能。**
2. 走不到其餘 25 個 block 的原因是**主線進度**：探索用的六個第 1 級戰士
   沒有清掉索寇要塞，`DS:4AA7h` 就停在初始值。
3. 因此下一步不是「做城區移動」，而是**把索寇要塞打完**——那需要的是
   戰鬥、法術與解謎能真的走完，不是新的畫面。
   `TestSokalKeepOpensTheOtherBoatRoutes` 已經證明這一段推得動：探索器拿到
   裝備、打贏那一場、`DS:4A21h` 變成 255，鬼魂說出 SAMOSUD，
   `DS:4AA7h` 變成 254。
4. **還缺一步**：要回碼頭去用那些航線，得先離開要塞，而離開要塞與往貧民窟
   都掛在 `DS:6DD5h` 上，remake 從來沒寫過它。詳見 spec 100。
