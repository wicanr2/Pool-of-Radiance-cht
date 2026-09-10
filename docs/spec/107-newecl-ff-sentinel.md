# Spec 107：`NEWECL FFh` 是「不換區塊」

狀態：READY（`FFh` 不是合法目標、唯一走得到的那一格、與 `LOAD FILES FFh`
成對；已實作並通過驗收；**`FFh` 是被誰吃掉的**）。
日期：2026-09-03（2026-09-10 補上收尾路徑）。

## 一句話

`FFh` 在 `NEWECL` 的運算元裡是哨兵，跟 `LOAD FILES` 第一欄的 `FFh` 是同一個
意思：**這個方向沒有東西要換**。

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

### 下游怎麼吃掉它（2026-09-10）

**沒有人特判，`FFh` 只是一個不存在的區塊編號。** 四段證據：

1. **handler 逐條讀完**（`0CDDh..0D78h`）：`0D02` 把運算元存進 `[bp-1]`、
   `0D08` 寫進 `DS:82A2h`，接著 `0D0B..0D4E` 組檔名，`0D53` 呼叫
   `00BEh:0061h`（overlay-17 entry 13），`0D5C` 把 `[bp-1]` 推進去呼叫
   **`0045h:0039h`**——依 spec 109 是 **overlay-7 的 entry 5**。
2. **檔名與運算元無關**：轉成字串的是 `DS:52D4h`（區域編號），接的兩個字串
   常數就在 overlay-03 裡——`0CD4h` 是 Pascal 字串 `ECL`、`0CD8h` 是 `.dax`。
   `DS:52D4h` 是區域編號有兩個獨立佐證：spec 006 的肖像檔名，以及全域符號集
   那一段（`52D4h` 為 1 時檔案是 `8X8D1.DAX`）。所以組出來的一律是
   `ECL<區域>.dax`，`FFh` 不會跑進檔名。
3. **overlay-7 沒有任何比對 `FFh` 的指令**：九種形狀（`3C FF`、`3D FF 00`、
   `3D FF FF`、`80 /7 FF` 的三種 modrm、`83 /7 FF`、兩種 `es:` 前綴）掃過整顆
   7886 bytes，**零命中**。同一組形狀掃 overlay-03 會在 `0DC3h` 命中
   `LOAD FILES` 那個明文比對——正對照成立，掃描面沒有洞。
4. **`FFh` 不是合法的區塊編號**：八個 `ECL*.dax` 的目錄逐筆讀出來，
   block id 只有 **0..29**（ecl1 兩個、ecl2 三個、ecl3 四個……），255 不在其中。

所以 `NEWECL FFh` 要求載入一個**不存在的區塊**，下游的查找找不到就什麼都沒
換——**`FFh` 的無害來自資料，不是來自程式碼裡的特判**。這也解釋了為什麼
`LOAD FILES` 需要明文比對而 `NEWECL` 不用：前者的 `FFh` 要擋的是地圖載入，
那一側沒有「編號查不到」這道天然的閘。

**還沒讀的只剩一件，而且不影響 remake 的語意**：overlay-7 entry 5 找不到編號
時具體回什麼（錯誤碼還是靜默返回）。不論哪一種，結果都是「不換區塊」。

## 五、remake 的語意

**`NEWECL FFh` 不換區塊，直接執行下一條指令。**

在唯一走得到的那一格（索引 6），這個語意跟「換到自己再重跑一次 lifecycle」
分不出來：`99A3h` 的下一條就是 `99A7h EXIT`，而 `994D CALL @C01E` 已經把
`DS:6DD5h` 清掉，重跑入口 0 也會立刻 `EXIT`。差別只在入口 4 會不會多重載一次
同一張 GEO。取「不換」是因為它跟 `LOAD FILES FFh` 的既有語意一致，且不會憑空
產生一個不存在的區塊編號。

## 六、驗收（已通過）

- 世界巡迴治具 `TestWorldTourReachesTheAreasBehindTheHarbour` 的
  `ECL session target block 0xFF is unavailable` 歸零（修之前 22 趟裡 7 次，
  全部來自 `ecl1/24`）。修完之後那一趟走到 12 張圖、12 個 ECL block，
  其中 **GEO1/31 是修之前走不進去的**。
- Pool 整包測試與 engine 整包測試都綠。

## 七、實作位置

共用 engine 的 `eclvm.BlockSession.switchTo` 呼叫端：目標等於
`eclvm.NoBlockChange`（`0FFh`）時不換區塊、不回報 transition，直接讓機器
從 `NEWECL` 的下一條指令跑下去——PC 在 handler 裡已經跨過運算元了。
測試是 `TestBlockSessionTreatsTheNoBlockChangeSentinelAsNoSwitch`。

## 八、修完之後才看得到的東西

GEO1/31 一走得進去，那一區的戰鬥就進了巡覽範圍，於是有兩類硬失敗浮上來。
兩者原因不同，**都不是這次的哨兵修正引入的**——它先前只是走不到那一區：

- **戰術地圖卡住**（`GEO1/31 第 20／23 回合行動者 7／10`，都是敵方）：已修。
  根因是離場的 combatant 每回合又拿到先攻分數，然後被選成行動者；體型 0 的
  mover 在目的格探測裡不受出界檢查，於是走出盤面。契約與未閉合項見
  [spec 062](062-combat-round-loop.md) 契約 7 與
  [spec 058](058-destination-probe-and-tactical-layout.md) 契約 6。
  那段碼自 `e6fe452`（2026-09-02 的回合迴圈）就在，三個敵方 AI commit
  都沒有碰過它——回合數要到二十幾才會有人死，所以要走得進這一區才看得到。
- **格子選單卡住**（`GEO7/23 (1,1)`）：**還在**，重現是 seed 106、destination 2。
  形狀與下一步見 `WORKLIST.md`。

巡覽治具的硬失敗訊息現在帶 seed、狀態列、戰術名冊與 ECL block ——
先前這三類卡住的訊息裡都沒有真正的原因：戰術地圖那條的錯誤被
`Update` 收進狀態列，而治具印的是 `tactical.Status`。
