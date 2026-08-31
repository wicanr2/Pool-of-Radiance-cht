# Spec 043：LOAD FILES 與三個 WALLDEF slot

狀態：READY。跨 archive adventure controller 由 Spec 045 閉合。
日期：2026-09-01。

## IDA Pro 9.4 證據

輸入 `overlay-03.bin` SHA-256
`5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`，位址空間為
overlay-local、base 0、16-bit。非破壞性匯出為：

- `docs/audit/ida-overlay03-load-files-handler.json`，函式 `0D80h..0ED4h`。
- `docs/audit/ida-overlay03-archive-field-consumers.json`，包含 `NEWECL` handler
  `0CDDh..0D78h`。

handler 先依序讀三個 byte operands 到 local。opcode `21h LOAD FILES` 時：

- 第一欄不是 `FFh／7Fh` 且隊伍狀態允許時，寫 party `+018Ah` 並呼叫 GEO loader；
- 第二欄在此 handler 沒有 consumer；
- 第三欄不是 `FFh` 且另一隊伍狀態成立時走額外 loader，該分支語意尚未閉合。

opcode `37h LOAD PIECES` 時，正常分支依 slot 1→3 掃三欄；非 `FFh` 的 selector 以
`LoadWallSet(slot,selector)` 載入。`7Fh` 第一欄是另一條 sentinel：固定載入 slot 1、
selector 0。這保留 Spec 009 的初始多-record wall set。

## Slums identity

ECL2/block20 entry 4：

1. `9A8Bh LOAD FILES 20,2,FFh`，在目前 archive 2 載入 `GEO2/block20`；第二欄 2
   不能再誤解為 GEO archive，因 handler 不讀它。
2. `9A92h LOAD PIECES 2,4,1`，依序載入 `WALLDEF2` selectors 2／4／1 到 slots 1／2／3，
   symbol 來源為 `8X8D2` 的相應 blocks。

## 實作契約

- game pack 提供三槽 loader；三次載入後建立 renderer view 時，WallDefs 與 symbol band
  仍依全域 slot 1／2／3 排列，不得把每個獨立 PieceSet 都當 slot 1。
- 本輪前端只接受已閉合的初始 sentinel `127,127,127` 與完整三槽 `2,4,1`；含 `FFh`
  的局部替換在保存既有 slot state 前仍失敗即關閉。
- 跨 archive 的 `DS:52D4` producer 尚未閉合；本規格不授權在 LOAD FILES handler 裡
  由第二欄偷偷切 archive。
