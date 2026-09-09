# Spec 135：原版紮營畫面的版面

狀態：DRAFT（幾何量自原版畫面；營火那張圖的來源還沒定位，選單模型已確定）。
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

## remake 現況與差距

remake 的 `E` 開的是一個**直排選單**（`休息／記憶法術／離開`）疊在冒險畫面
右側，休息時間與按鍵提示那三行印在 `(150,300)` 一帶——**壓在第一人稱視野
框上面**。畫面見 `docs/screenshots/pool-remake-chinese-camp.png`。

要照原版做，缺的是：

1. **營火那張圖的來源。** `PIC*.DAX` 的區塊是一段動畫（1 byte 張數＋每張
   4 byte 前綴＋17 byte 圖片頭，第二張以後與第一張 XOR，見 spec 117），
   營火會閃動正好是這個形狀，但還沒有比對過是哪一個檔的哪一個區塊。
   **驗收**：拿 `58-I.idx` 的 `(24,24)` 88×88 與八個 `pic*.dax`／`cpic*.dax`
   的每一張逐格比對，找到 100% 的那一張。
2. **指令列的兩組字串**在 overlay-20 的哪個位移。`Rest daYs Hours Mins Inc
   Dec Exit`（`069Fh`）已經在 spec 114 記過，`CAMP: SAVE VIEW MAGIC REST
   ALTER EXIT` 還沒定位。
3. remake 的紮營要從「彈出選單」改成「換指令列」——`campInput` 的按鍵已經
   照原版接了（`Y`／`H`／`M`／`I`／`D`／`R`／`E`），改的是畫法與 `SAVE`／
   `VIEW`／`ALTER` 三項還沒有。
