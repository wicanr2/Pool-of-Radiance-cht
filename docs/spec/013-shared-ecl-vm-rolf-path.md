# Spec 013：共用 ECL VM 與 Rolf 真實 bytecode 路徑

狀態：CONFORMED（VM 核心、外部效果失敗即關閉、Rolf handler 至 EXIT、前端逐 boundary 接管）；
DRAFT（完整 Pool opcode 副作用）
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
前端在真實 `ScriptBlock` 存在時，Begin 即建立 production machine：SAVE 寫入直接更新
畫面座標，Return menu 暫停等待按鍵，`3Ah DELAY` 形成 34 個約 150ms frame，文字事件
驅動 dialogue，`EXIT` 才結束 tour。Xvfb 測試從 Begin 逐次按 Return／推進 update，驗證
第 34 frame 後到 `(0,4,3)` 且 `introDone=true`。舊 typed `TourStep` 播放只保留給沒有
原始 script 的合成 UI fixture，不是正式遊戲路徑。

這證明共用 VM 已驅動 production Rolf bytecode；仍不可將本規格冒稱為完整遊戲 ECL
runtime，因為其他 block 與 passthrough opcode 的作品副作用尚未逐項接完。
