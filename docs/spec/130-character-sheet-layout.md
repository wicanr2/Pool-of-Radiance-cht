# Spec 130：人物資料頁的版面

狀態：READY（每一行與每一欄的位置、九個欄位的內容、建角當下那幾個值的來源）；
DRAFT（`STATUS` 除了 `OKAY` 以外的字、`VIEW` 從冒險畫面進來時那一頁多了什麼）。
日期：2026-09-08。

## 輸入

原版畫面：`workplace/dosgolem-ref` 的第 16 幀（建角擲完屬性那一頁），
320×200 的 EGA 色號陣列。鍵序見 `tools/dosgolem-reference.sh`。

## 版面（exact，量自色號陣列）

一行字高 7、行距 8。左界 native x 8..9。

| 行 | native y | 內容 |
|---|---|---|
| 1 | 24..30 | `MALE DWARF AGE 52` |
| 2 | 32..38 | `LAWFUL GOOD` |
| 3 | 40..47 | `FIGHTER` |
| 4..9 | 56..102（每 8） | `STR`／`INT`／`WIS`／`DEX`／`CON`／`CHA` |
| 10 | 120..126 | `LEVEL 1` 與 `EXP 0` |
| 11 | 136..142 | `AC 10`、`THAC0 20`、`ENCUMBRANCE 150` |
| 12 | 144..151 | `HP 8`、`DAMAGE 1D2+1`、`MOVEMENT 12` |
| 13 | 176..182 | `STATUS OKAY` |

欄位的 native x（標籤／值）：

| 欄位 | 標籤 | 值 |
|---|---:|---:|
| 屬性 | 9 | 43 |
| `GOLD` | 129 | 171 |
| `LEVEL` | 9 | 59 |
| `EXP` | 137 | 170 |
| `AC` | 9 | 35 |
| `THAC0` | 73 | 122 |
| `ENCUMBRANCE` | 177 | 275 |
| `HP` | 9 | 34 |
| `DAMAGE` | 65 | 123 |
| `MOVEMENT` | 200 | 275 |
| `STATUS` | 9 | 65 |

肖像佔 native x 216..311、y 8..103。最下面那一列
`KEEP THIS CHARACTER? YES NO` 在框外（native 192..198），與 spec 123 的
指令列同一條。

## 建角當下那幾個值從哪來

第 16 幀那個角色是 `MALE DWARF FIGHTER`、`STR 16`、`GOLD 150`，
顯示 `AC 10 / THAC0 20 / ENCUMBRANCE 150 / DAMAGE 1D2+1 / MOVEMENT 12`：

- **`AC 10`**：無甲。記錄裡存的是**內部值** `60 − AC`，所以是 50。
- **`THAC0 20`**：一級（`BaseThac0Internal`，同樣是 `60 − THAC0`）。
- **`ENCUMBRANCE 150`**：身上的硬幣重量（`CarriedWeight`），150 枚金幣。
- **`DAMAGE 1D2+1`**：徒手 1d2 加力量的傷害調整——`STR 16` 給 +1
  （`StrengthDamageAdjustment`）。
- **`MOVEMENT 12`**：基礎移動力，未被負重壓下來。

## remake 的驗收

1. 十三行與各欄位的 x 逐格對原版（native 座標乘二，基線再加倚天字型的
   ascent 14）。
2. 九個欄位都算得出來，`AC` 與 `THAC0` 顯示的是值本身、不是內部值。
3. 右上角有肖像；建角當下用預設那一張（`PortraitHead`／`Body` 都是 1）。
4. 最下面那一列由這一頁自己畫，全域的 F-key 提示不畫——兩邊都畫會疊成一團。
5. `docs/audit/dos-parity-sample.json` 的 `sheet` 外框比例不低於先前。
