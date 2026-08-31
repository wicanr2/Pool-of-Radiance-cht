# Spec 015：第一張地圖 cell lifecycle 接線

狀態：PARTIAL（第一個空事件格 CONFORMED；玩家事件 effects DRAFT）
日期：2026-08-31

## 入口與暫存器

`ECL3.DAX` block 0 的五個 command-set entries 為 `9914h/99EBh/9A5Eh/9A93h/9AF2h`。
第五入口是已完成的載入／Rolf 流程。依系列五入口 ABI，第一入口 `9914h` 是 per-turn，
第二入口 `99EBh` 是 SearchLocation；Pool block shape 與資料流吻合，但 Pool executable
呼叫順序仍標為 `strong inference`。

Pool 與 CoAB 的 executable／GEO consumer 共同支持以下 bridge；Pool 本輪以真實
`ECL3/block0` 再驗證 consumer：

- `C04Bh/C04Ch/C04Dh`：X／Y／facing；
- `C04Eh`：目前面向的 wall cache；
- `C04Fh`：目前 GEO cell 的原始 terrain byte。

第一入口在 `9965h` 執行 `AND C04Fh,7Fh → 6E82h`，但隔離掃描的 1,024 個樣本都在
`997Dh EXIT`，因此它不能單獨支持「terrain event dispatch 已完成」的舊說法。第二入口
才在 `99F7h` 再算 `C04Fh & 7Fh → 6E82h` 並進入 terrain handlers。事件不是 UI 依
座標硬編碼，而是由 GEO projection 餵進原始 ECL；完整結果見 Spec 016。

## 同 session 實作與驗證

共用 engine `eclvm.Machine.SetPC` 可在保留 memory 的情況下切到同 block 另一 lifecycle
entry，並清除 GOSUB stack。Pool `RunInitialCellEntry` 在每次成功移動後同步上述五格，
將 PC 設為 `9914h` 後執行。

真實正常路徑：Rolf handler `B06Eh` 跑到 `AE85h EXIT` → 左轉 facing 2 → 從 `(0,4)`
前進到 `(1,4)` → 同一 VM 執行 entry 0。結果精確為 15 instructions、無文字／選單／
external event，於 `997Dh EXIT` 返回，next PC `997Eh`。

entry 0 的空結果只是 per-turn 完成，不再冒充完整 cell transaction。engine
`RunUntilEvent` 逐 instruction 推進，
遇到第一個文字、選單、passthrough external event 或 EXIT 立即返回；測試以
`SAVE → DELAY → SAVE` 證明 DELAY 後面的 SAVE 尚未執行。若其他格回傳尚未接的事件，
前端會設 `cellEventPending` 並停止後續移動，避免安靜跨過 COMBAT。下一步是將文字／
選單／戰鬥逐類接到 frontend continuation。entry 1 的第一個正常文字切片見 Spec 016。
