# Spec 100：`DS:6DD5h` — 擋住整個世界的那一個變數

狀態：CONFORMED（兩個使用點、它們各自通往哪裡、以及「沒有任何 ECL 寫它」）；
OPEN（誰寫它、寫的是什麼值）。日期：2026-09-03。

## 一句話

貧民窟與索寇要塞的回程船**都掛在同一個變數上**：`DS:6DD5h` 不是 0 才會發生。
remake 從來沒有寫過它，所以兩條路都是死的——這才是「其餘 25 個 ECL block
走不到」的直接原因。

## 兩個使用點

**城區（`ecl3` block 0，入口 0 也就是每回合那一支）**：

```
993a  COMPARE DS:6DD5h, 0
9940  IF =
9941  GOTO 9965h            ; 是 0 就整段跳過
9945  SAVE 1, DS:49EBh
994b  CALL C01Eh
994f  SAVE 255, DS:6DC9h
9955  LOAD FILES FFh,FFh,7Fh
995c  SAVE 2, DS:6E12h      ; archive 2
9962  NEWECL 20             ; ← 貧民窟（ECL2 block 20，spec 043）
```

**索寇要塞（`ecl4` block 21，入口 0）**：

```
9914  GOSUB AE42h
9918  COMPARE DS:6DD5h, 0
991e  IF =
991f  GOTO 9998h            ; 是 0 就整段跳過
9923  PRINTCLEAR "'DO YOU WANT TO TAKE A BOAT BACK TO PHLAN?'"
9945  GOSUB AE32h           ; YES / NO
9963  PRINTCLEAR "'YOU BOARD A BOAT.'"
9977  SAVE 15, C04Bh / 1, C04Ch / 3, C04Dh   ; 回到碼頭 (15,1) 面向 3
9989  SAVE 3, DS:6E12h
998f  NEWECL 0
```

兩邊的形狀一模一樣：**站在邊上往外走**（城區的西門、要塞的碼頭），
`DS:6DD5h` 非零，ECL 才把隊伍送去下一張圖。

## 誰寫它：還沒找到

- **沒有任何 ECL 區塊寫它。** 29 個有文字的 block 裡，26 個靜態解得開，
  只有上面兩條 `COMPARE`，沒有 `SAVE`。（三個解不開的是 ecl5/7、ecl7/17、
  ecl7/22，都不是每回合入口。）
- **執行檔裡沒有任何一處直接定址它。** `START.EXE` 與 38 份 overlay 全掃
  兩位元組字面 `D5 6D`：**零筆**。所以它是被索引或指標寫進去的
  （像 `es:[di+n]`），不是 `mov ds:[6DD5h], al` 這種形式。
- 附近那一段（`6DC1h`、`6DC7h`、`6DC9h`、`6DCBh`）也被 ECL 讀寫，
  像是引擎與 ECL 之間的一組共用變數。

## 最可能的語意（**假設待驗**）

**「隊伍試圖走出這張圖的範圍」。** 兩個使用點都在地圖邊上，而且都通往
「離開這一區」。牆的查詢本身會把座標夾回 0..15（overlay-30 `0358h`，
spec 099），所以 wrap 之後那一格看起來是合法的——引擎另外記下「這一步
本來會走出去」才說得通。

**這只是假設。** 要證實它得找到寫入點：從移動服務（overlay-14 `0DB7h`，
overlay-03 `3917h` 呼叫它）往下讀，看它在套用位移前後改了哪些 DS 變數。

## 這一條與 spec 099 的關係

099 說得沒錯：碼頭的其他航線被 `DS:4AA7h` 擋著，而那個旗標由索寇要塞寫。
`TestSokalKeepOpensTheOtherBoatRoutes` 已經證明**那一段 remake 推得動**
（`4A21=255`、`4AA7=254`）。但 099 漏掉了這一條：**要走回碼頭去用那些航線，
得先能離開要塞**，而離開要塞掛在 `DS:6DD5h` 上。099 裡「地圖沒有走出邊界
這回事」那句話要收窄成它真正證明的東西：**牆的查詢會把座標夾回去**；
引擎有沒有另外記下「本來要走出去」，就是這一條的 OPEN。
