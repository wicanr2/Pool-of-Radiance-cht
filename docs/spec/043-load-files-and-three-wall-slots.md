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
- 第三欄不是 `FFh` 且另一隊伍狀態成立時走額外 loader：**那是背景圖庫**。
  `0E0Fh` 起把 overlay-03 `0D79h` 的 Pascal 字串 `BACPAC` 抄進堆疊上的暫存，
  再 `0E24h` 以 `(28h, 0)` 呼叫 overlay-33（段 `0147h`）的 `002Ah`。
  **它不重載 ECL 的變數區**——這一點在追「港務長的船票旗標 `4A01` 是誰清的」
  （spec 102）時要緊：`LOAD FILES` 不是答案。

opcode `37h LOAD PIECES` 時，正常分支依 slot 1→3 掃三欄；非 `FFh` 的 selector 以
`LoadWallSet(slot,selector)` 載入，**`FFh` 的那一格不換**——remake 這邊沿用上一份
的同一格（`ReadDOSPieceSlots` 的 `previous`）。先前是失敗即關閉，走到野外那幾張
圖之後會擋住整條路徑（`LOAD PIECES [1 3 255]`）。`7Fh` 第一欄是另一條 sentinel：固定載入 slot 1、
selector 0。這保留 Spec 009 的初始多-record wall set。

## Slums identity

ECL2/block20 entry 4：

1. `9A8Bh LOAD FILES 20,2,FFh`，在目前 archive 2 載入 `GEO2/block20`；第二欄 2
   不能再誤解為 GEO archive，因 handler 不讀它。
2. `9A92h LOAD PIECES 2,4,1`，依序載入 `WALLDEF2` selectors 2／4／1 到 slots 1／2／3，
   symbol 來源為 `8X8D2` 的相應 blocks。

## 區塊編號決定 archive

`21h` 只帶區塊編號（第二欄整支沒有 consumer），archive 由 `DS:52D4h` 補。
remake 這邊追那個值會落後：從碼頭搭船到 `ecl7` block 26 之後，那一區要
`LOAD FILES 5`，而當下 archive 是 7 —— GEO7 只有 17／22／23／26，
block 5 在 GEO5，於是硬失敗。

不必追：**區塊編號在八個 GEO 檔裡是全域唯一的**——29 個編號對 29 張圖，
一個不重複（`TestGeometryBlockIDsAreGloballyUnique` 釘住這條性質）。
所以編號本身就決定了它在哪一個檔案。`GeometryCatalog.MapByBlock` 拿編號查，
查不到就失敗即關閉。

| GEO | 區塊 |
|---:|---|
| 1 | 18、24、31 |
| 2 | 9、15、20 |
| 3 | 0、14 |
| 4 | 2、10、21 |
| 5 | 3、4、5、6、7 |
| 6 | 1、25、28 |
| 7 | 17、22、23、26 |
| 8 | 13、16、27、29、30、32 |

## 實作契約

- game pack 提供三槽 loader；三次載入後建立 renderer view 時，WallDefs 與 symbol band
  仍依全域 slot 1／2／3 排列，不得把每個獨立 PieceSet 都當 slot 1。
- 本輪前端只接受已閉合的初始 sentinel `127,127,127` 與完整三槽 `2,4,1`；含 `FFh`
  的局部替換在保存既有 slot state 前仍失敗即關閉。
- 跨 archive 的 `DS:52D4` producer 尚未閉合；本規格不授權在 LOAD FILES handler 裡
  由第二欄偷偷切 archive。
