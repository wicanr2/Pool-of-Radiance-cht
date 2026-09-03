# Spec 107：`NEWECL FFh` 是「不換區塊」

狀態：READY（`FFh` 不是合法目標、唯一走得到的那一格、與 `LOAD FILES FFh`
成對）；OPEN（原版 handler 的收尾路徑）。日期：2026-09-03。

## 一句話

`FFh` 在 `NEWECL` 的運算元裡是哨兵，跟 `LOAD FILES` 第一欄的 `FFh` 是同一個
意思：**這個方向沒有東西要換**。目前共用 engine 對它直接報錯，是世界巡迴
治具唯一還在冒出來的硬失敗（22 趟裡 7 次，全部來自 `ecl1/24`）。

## 固定輸入

DOS ZIP SHA-256
`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`；
`ECL1..8.DAX` 29 個區塊、`GEO1.DAX`；overlay-03 反組譯位址為 overlay-local。

## 一、`FFh` 不可能是區塊編號

八個 ECL 封存檔的區塊編號全部列得出來：

| 封存檔 | 區塊 |
|---|---|
| ECL1 | 18、24 |
| ECL2 | 9、15、20 |
| ECL3 | 0、8、11、14 |
| ECL4 | 2、10、21 |
| ECL5 | 3、4、5、6、7 |
| ECL6 | 1、19、25、28 |
| ECL7 | 17、22、23、26 |
| ECL8 | 13、16、27、29 |

最大是 29。**沒有 255**，所以 `NEWECL FFh` 不可能是在指某一個區塊。

## 二、它只出現在一張表裡，而且跟 `LOAD FILES` 成對

全遊戲只有五個區塊的 `NEWECL` 目標是變數（其餘都是字面常數）：
`ecl1/24`、`ecl2/9`、`ecl2/15`、`ecl5/3`、`ecl5/6`。逐張把表的原始位元組讀出來，
**只有 `ecl1/24` 那一張有 `FFh`**：

```
索引        0    1    2    3    4    5    6    7
GEO  @99A8  FF   1F   FF   FF   0E   1A   FF   18
ECL  @99B0  FF   FF   FF   FF   0E   1A   FF   FF
```

（`ecl2/15` 的兩張是 `1D 00 00 02` 與 `08 00 00 04`；`ecl5/3` 的是
`04 04 06 06`。位址對 `dax` payload 的基準是 `98FEh`——payload 前兩 bytes 是
`1388h` prefix，loader 會跳過，見 spec 002。）

入口 0（`9914h`）的骨幹：

```
992E  COMPARE @6DD5, 0 ; IF = ; EXIT      ; 不是要離開這張圖就整段跳過
9936  SAVE @C04D → @6E82                  ; 朝向 0 北 1 東 2 南 3 西
993D  COMPARE 1Fh, @49C5 ; IF = ; ADD 4 → @6E82
994D  CALL @C01E                           ; 走掉那一步、座標繞回
9951  SAVE 255 → @6DC9                     ; 不要再做一次一般的 GEO 移動
9957  GETTABLE 99A8h, @6E82 → @6E79        ; 要載哪一張 GEO
997B  LOAD FILES @6E79, 2, FF
9983  COMPARE @6E82, 1 ; IF = ; GOTO 99A7  ; ← 跳過 NEWECL
998E  COMPARE @6E82, 7 ; IF = ; GOTO 99A7  ; ← 跳過 NEWECL
9999  GETTABLE 99B0h, @6E82 → @6E7D
99A3  NEWECL @6E7D
99A7  EXIT
```

`49C5` 是這個腳本目前在哪一張 GEO 上：同一個區塊的入口 4 用
`B6CA COMPARE @49C5, 18h` 決定要 `LOAD FILES 24` 還是 `LOAD FILES 31`。
所以索引 0..3 是 GEO24 的四個朝向，4..7 是 GEO31 的。

## 三、走得到的只有四格半，其中一格會踩到 `FFh`

直接量原始 GEO 資料，兩張圖各有哪些邊界格走得出去：

| GEO1 區塊 | 北 | 東 | 南 | 西 |
|---|---|---|---|---|
| 24 | 無 | (15,4) (15,11) | 無 | 無 |
| 31 | (4,0) (11,0) | (15,4) (15,11) | (4,15) (11,15) | (0,4) (0,11) |

對回表：

| 索引 | 方向 | GEO | ECL | 走得到？ | 腳本怎麼處理 |
|---|---|---|---|---|---|
| 0 | GEO24 北 | FF | FF | 否（有牆）| — |
| 1 | GEO24 東 | 1F | FF | 是 | `9983h` 明文跳過 `NEWECL` |
| 2 | GEO24 南 | FF | FF | 否 | — |
| 3 | GEO24 西 | FF | FF | 否 | — |
| 4 | GEO31 北 | 0E | 0E | 是 | `NEWECL 14` |
| 5 | GEO31 東 | 1A | 1A | 是 | `NEWECL 26` |
| 6 | GEO31 南 | FF | FF | **是** | **沒有守衛，直接 `NEWECL FFh`** |
| 7 | GEO31 西 | 18 | FF | 是 | `998Eh` 明文跳過 `NEWECL` |

兩道守衛擋的正好是「要載圖但不換區塊」那兩格——GEO24 與 GEO31 是同一個腳本
管的兩張圖，東西向來回不該重跑 lifecycle。索引 6 兩欄都是 `FFh`，什麼都不用
載，所以腳本沒有為它多寫一道守衛：`FFh` 自己就得是無害的。

## 四、原版 handler 沒有特判

`NEWECL` 的 handler 在 overlay-03 `0CDDh..0D73h`：把目前的區塊編號存進
`[DS:4933]+1E4h`、讀運算元寫回 `DS:82A2h`、用 `DS:52D4h` 組出封存檔名、
載入該區塊，最後令 `DS:4390h=1`、`DS:4391h=1`、`DS:4398h=0`。**整段沒有比對
`FFh`**——和 `LOAD FILES` 的 handler（`0D80h`，`0DC3h` 明文 `cmpb $0FFh` 就跳過
地圖載入）不同。所以 `FFh` 是被下游的區塊載入吃掉的，不是在 handler 入口擋掉。
那條收尾路徑還沒讀完，這是本規格的 OPEN。

## 五、remake 的語意

**`NEWECL FFh` 不換區塊，直接執行下一條指令。**

在唯一走得到的那一格（索引 6），這個語意跟「換到自己再重跑一次 lifecycle」
分不出來：`99A3h` 的下一條就是 `99A7h EXIT`，而 `994D CALL @C01E` 已經把
`DS:6DD5h` 清掉，重跑入口 0 也會立刻 `EXIT`。差別只在入口 4 會不會多重載一次
同一張 GEO。取「不換」是因為它跟 `LOAD FILES FFh` 的既有語意一致，且不會憑空
產生一個不存在的區塊編號。

## 六、驗收

- 世界巡迴治具 `TestWorldTourReachesTheAreasBehindTheHarbour` 的硬失敗
  `ECL session target block 0xFF is unavailable` 歸零（目前 22 趟 7 次）。
- `ecl1/24` 走到 GEO31 南緣 (4,15) 或 (11,15) 往南一步之後：區塊仍是 24、
  archive 仍是 1、座標由 `CALL @C01E` 繞回 y=0。

## 七、實作位置（尚未動工）

判斷在共用 engine 的 `eclvm.BlockSession.switchTo`：目前對查不到的區塊一律
回錯。要加的是「目標為 `FFh` 時不換、讓呼叫端繼續往下跑」。
**那是另一個 repo（`golden-box-remake-engine`），本專案的 push 授權不涵蓋它，
所以還沒有動。** 另外該 repo 的 `git config user.email` 目前是公司位址，
動它之前要先設 repo-local 的 `wicanr2@gmail.com`。
