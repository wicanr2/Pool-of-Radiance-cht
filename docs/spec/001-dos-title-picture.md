# Spec 001：DOS TITLE DAX 圖像

狀態：CONFORMED；日期：2026-08-31。

## 範圍與固定輸入

只涵蓋 DOS `TITLE.DAX` block 1／2 的 container、picture header、indexed pixels、
標準 EGA palette 與 2× 最近鄰呈現。不涵蓋播放時間、淡入、音訊、其他 PIC block
或主選單文字 renderer。

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`
- `TITLE.DAX` ZIP CRC32：`C6C5B1B9`；uncompressed bytes：33,898。
- Docker oracle：DOSBox 0.74-3，未修改 `START.EXE`，640×400 crop。
- 平台前提：IBM Enhanced Graphics Adapter technical reference 的 6-bit palette
  register contract；200-line 相容 palette 的數位 RGB levels 映射為
  `00／55／AA／FF`。來源：
  <https://bitsavers.org/pdf/ibm/pc/cards/Technical_Reference_Options_and_Adapters_Volume_2_Apr84.pdf>

## Exact container／picture facts

engine `dax.Parse` 得到兩個 block：

| block | decoded bytes | header | metadata |
|---:|---:|---|---|
| `01h` | 32,017 | 320×200、1 item、x=1、y=0 | `00 11 22 13 02 21 23 33` |
| `02h` | 32,017 | 320×200、1 item、x=1、y=0 | `00 11 22 13 22 11 23 33` |

17-byte header 後為 32,000 bytes packed nibbles；每 byte 高 nibble先行，正好
解出 64,000 個 4-bit indexed pixels。長度、尺寸、item count 任一不符即失敗，
不可截斷或猜 fallback。

## 同狀態視覺證據

block `01h` 經 engine `ParsePicture` 解碼並以最近鄰放大 2×，與 DOSBox
[`title-intro.png`](../reference/original-dos/title-intro.png) 比較：

- 十種 palette index 的 pixel count 逐色完全相同；
- 幾何／index 差異為 0；
- 舊 engine palette 使用 `173／82`，因此 127,240 pixels 的 RGB 不同；
- DOSBox／標準 EGA 是 `170／85`，白與黑等未受影響的 13,028／115,732 pixels
  也與上述差異數精確對帳。

因此 block layout、nibble order、2× scaling 為 `exact`；EGA levels 修正由平台規格
授權，並以同狀態 runtime 對帳。這不是憑單張觀感調色。

## Typed behavior

1. game adapter 從 `TITLE.DAX` 依 block ID 取得 decoded bytes。
2. engine `graphics.ParsePicture(data, false, 0)` 必須回 320×200×1。
3. engine `graphics.EGA16` 使用 `00／55／AA／FF` levels，color 6 為 `AA5500`。
4. offline exporter 輸出未縮放 320×200 PNG；frontend 以整數最近鄰 2× 顯示。
5. 原始 DAX 不進 Git／公開封包；repository 只保存研究對拍的 oracle PNG、規格與
   可重生工具。

## 驗收

- engine palette unit test 逐項固定 16 色 RGBA。
- Pool 真檔 anchor 固定兩 block 的 ID、decoded length、尺寸、item count、metadata。
- block 1 PNG 放大後與 DOSBox oracle 的 AE 必須為 0。
- malformed header／長度仍失敗即關閉。

2026-08-31 收據：`cmd/export-title` 匯出的 block 1 為 320×200；最近鄰放大
2× 後與固定 DOSBox oracle 比較，AE=`0`。`title-atlas.png` 為 640×200，依序
並列 block 1／2。`go test ./...` 的真檔錨點亦通過。
