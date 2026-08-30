# Spec 007：DOS 戰鬥圖示

狀態：READY（圖庫形狀、selector 範圍、READY／ACTION 與大小 family、六部位雙色）
日期：2026-08-31

## 輸入、工具與位址基準

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `overlay-16.bin` SHA-256：`a142d8a8f3b3c46a7755cf231228e7981c5b9f77721313ce79ab6c0aec105b94`。
- IDA Pro 9.4；下列位址全是 overlay-16 local，不是 START resident 位址或 ZIP offset。
- 可重生 archive 結果：`docs/audit/dos-combat-icon-shapes.json`。

## Archive namespace（exact）

`CHEAD.DAX` 有 56 blocks：`00..0D`、`40..4D`、`80..8D`、`C0..CD`；
`CBODY.DAX` 有 128 blocks：`00..1F`、`40..5F`、`80..9F`、`C0..DF`。
CBODY 全為 24×24×1、305 bytes；CHEAD 的 `00/80` family 為 24×10×1、137 bytes，
`40/C0` family 為 24×8×1、113 bytes。184／184 blocks 解碼，失敗 0。

四個 family 由相同 selector 對應：base `00` 是 Large READY、`40` 是 Small READY、
`80` 是 Large ACTION、`C0` 是 Small ACTION。這個對應同時受到完整規則化 block
namespace、原版 editor 同時顯示 READY／ACTION，以及 `+C0h` 的 Small=`1`、Large=`2`
支持；證據等級為 `strong inference`，足以實作 99% remake，但不冒稱逐像素 runtime
call trace 的 `exact`。合成使用 masked picture：BODY 為 destination、HEAD 左上對齊
疊上；opaque 重疊依 engine 的 bitwise-OR 契約。此合成方向目前同樣標為
`strong inference`，後續可用原版單張 icon 像素擷取升級。

## Selector 與編輯器（exact）

overlay-16 `37F2h..3F00h`：

- `3BA0h..3BE2h` 對 CHA `+BDh` 做 Prev／Next：`0 ↔ 0Dh`，合法 Head 是 `0..13`。
- `3C3Fh..3C81h` 對 CHA `+BEh` 做 Prev／Next：`0 ↔ 1Fh`，合法 Weapon／Body 是 `0..31`。
- `3B11h`／`3B28h` 分別把 `+C0h` 設為 Large=`2`／Small=`1`。
- `3CE2h..3DA0h` 對 `+C1h..+C6h` 的 low／high nibble 做 modulo-16 Next／Prev。
- `3E27h..3E5Ah` 只有在最終確認後才提交 Head、Weapon、Size 與六個顏色 bytes。

欄位語意與原版差分仍由 Spec 003 負責：Body、Arm、Leg、Hair／Face、Shield、Weapon；
low nibble 是 Color-1，high nibble 是 Color-2。`+BFh` 在 `3851h..3873h` 暫時改成
`0Ch` 後恢復，語意仍未知；remake codec 必須原值保存，不得命名或覆寫。

## 實作閘門

adapter 必須拒絕 selector 或 size 越界，四種極值 block 都要在真實 ZIP 解碼並合成。
UI 可提供清楚的現代引導與 ESC 返回，但不能省略原版的 Head、Weapon、六部位雙色、
Size，以及 READY／ACTION 同時預覽。原版素材不提交 Git，只在本機 DOS ZIP 執行期解碼。
