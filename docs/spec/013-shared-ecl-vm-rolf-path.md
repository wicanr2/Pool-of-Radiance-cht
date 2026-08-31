# Spec 013：共用 ECL VM 與 Rolf 真實 bytecode 路徑

狀態：READY（VM 核心、外部效果失敗即關閉、Rolf handler 至 EXIT）；
DRAFT（前端逐 boundary 接管、完整 Pool opcode 副作用）
日期：2026-08-31

## 決策與邊界

CoAB 與 Pool 共用 ECL framing 與 opcode 執行形狀；作品位址、旗標、服務 selector
與玩家可見副作用不自動共用。使用者已授權參考 CoAB 已驗證程式，但不得修改或干擾
現行 CoAB remake。因此新 VM 只新增於獨立 `golden-box-remake-engine/eclvm`，Pool
先成為 consumer；CoAB 只執行唯讀回歸測試。

## 共用核心（READY）

`eclvm.Machine` 目前直接消費 engine `ecl` decoder，保存 PC、GOSUB stack、numeric／
string memory 與六種 compare 結果，並執行：

- `EXIT`、`GOTO`、`GOSUB`、`RETURN`、六種 `IF`；
- `COMPARE`、`ADD`、`SUBTRACT`、`DIVIDE`、`MULTIPLY`、`AND`、`OR`；
- `SAVE`、`GETTABLE`；
- `VERTICAL MENU`、`HORIZONTAL MENU` 與缺少 selection 時的 continuation。

作品中立核心不知道 Phlan、Rolf、Pool memory map 或 CoAB service flags。其餘 opcode
若未列入 adapter passthrough，立即報錯；不能安靜略過未知副作用。passthrough 只保留
原始 opcode／PC／第一運算元，作品 adapter 再按自己的 READY spec 消費。

## 真實 Pool 驗證（READY）

固定輸入沿用 Spec 011：DOS ZIP、`ECL3.DAX` 與 block 0 雜湊不變，payload mapping
base `9900h`，handler `B06Eh`。`gamepack.NewInitialEventMachine` 從
`ReadDOSInitialEvent` 保存的原始 block 建立 VM，只明確允許本路徑實際出現的：

- `0Ch SETUP MONSTER`
- `0Dh APPROACH`
- `0Eh PICTURE`
- `2Dh CALL`
- `31h SPRITE OFF`
- `3Ah DELAY`

真實 corpus 測試逐次提供唯一 Return selection，VM 由 `B06Eh` 執行至 `AE85h EXIT`。
驗收結果：八次選單 continuation、七頁導覽文字錨點、最後寫入
`C04Bh=0／C04Ch=4／C04Dh=3`，且六個 passthrough opcode 均在同一路徑被實際走到。
這證明共用 VM 能驅動 Rolf bytecode；尚未證明前端已逐 boundary 使用 VM，因此前端
手寫 tour 狀態機的替換仍是下一項實作，不可將本規格冒稱為完整遊戲 ECL runtime。
