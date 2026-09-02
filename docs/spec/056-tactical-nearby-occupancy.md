# Spec 056：戰術鄰近格位查詢與敵對側篩選

狀態：CONFORMED（overlay-25 entry 32 的完整資料流、篩選與輸出語意）；
DRAFT（`sub_13BE` 如何產生鄰近格位表、每筆前兩 byte 的語意、`6674h` 表的容量上限）。
日期：2026-09-02。

## 為什麼需要這一段

spec 053 的 `ResolveMovementProbe` 以 `attackTargetID` 決定目的格是進入還是攻擊，
但**誰提供那個 target** 一直未閉合；CONTEXT 也把 occupancy 列為未解。本規格閉合
讀取端：原版如何由一個 mover 取得附近格位上的敵對 combatant。

## 證據

- `docs/audit/ida-overlay25-nearby-opponents.json`，IDA Pro 9.4、overlay-local file
  offset、base 0：函式 `2465h..2591h`（十進位 9325..9617），110 條指令。
- 同目錄的 `ida-overlay25-opposing-side-selector.json` 保存被呼叫的
  `sub_23F5`：由 mover 取得對立陣營值。

## 資料流（exact）

1. 進入後以三個 far helper 由參數取得 mover 與距離，呼叫 `sub_13BE`，
   參數是 `ds:6674h` 與 `ds:6676h`。該呼叫產生**鄰近格位表**。
2. 表的筆數取自 `ds:6678h`（byte）。迴圈索引自 **1** 起算，每筆 **3 bytes**，
   位址為 `idx*3 + 6676h`；讀出的 byte 是 **combatant index**。
3. combatant record 由 `index*4` 查 `ds:6517h` 的 **far pointer 表**取得；
   record `+10Eh` 是陣營欄位。
4. 篩選條件是 `record[+10Eh] == sub_23F5(mover)`，即只留下對立陣營。
5. 命中者以 far memcpy（`5BBh:1692h`，長度 3）**原地往前壓縮**到
   `newCount*3 + 6676h`，迴圈結束後把 `ds:6678h` 更新為壓縮後的筆數。
6. 第二段迴圈把壓縮後每筆的 combatant index 寫進 **`ds:6CD7h`**，
   每筆 1 byte、同樣 1-based。函式回傳篩選後的筆數。

因此本函式同時是 in-place filter（`6674h` 表）與 index 匯出（`6CD7h` 陣列）。
spec 053 對它的描述「只留下 `+10Eh` 等於 mover 反值者」在此得到完整證實。

## 尚未閉合

- 鄰近格位表的產生規則：格位如何列舉、距離參數如何影響、是否含 mover 自身。
  這是完整 occupancy 的另一半，也是把 `attackTargetID` 接進 probe 的前提。

### ⚠ `sub_13BE` 這個名字不是 overlay-25 的位址

`ida-overlay25-nearby-opponents.json` 在 `2468h`（十進位 9384）的那條指令，
原始 bytes 是 **`9A 3E 00 38 01`**。`9Ah` 是 **far call**，其後依序是 offset 與
segment，因此目標是 **`0138h:003Eh`**。IDA 顯示的 `sub_13BE` 是把它線性化
（`0138h × 16 + 3Eh = 13BEh`）之後的自動命名，**與本檔其餘位址所用的
「overlay-local file offset, base 0」是兩個不同的位址基準**。

實測可證：`overlay-25.bin` 的 file offset `13BEh` 落在 `cmp ax, 13h`
（`3D 13 00`，起於 `13BDh`）的中間，不是指令邊界。以
`POOL_IDA_SEEDS=5054` 對該檔重跑匯出，得到的是首條 `adc ax, [bx+si]`、
尾為 `retf 4` 的錯位解碼。檔案本身無誤（SHA-256 `9fede24b…50c0e`，與該份
匯出記錄的完全相同）。

因此下一步不是在 overlay-25 內找 `13BEh`。掃過既有的 IDA 匯出可知，
**`0138h` 是被多個 overlay 共用的 stub 段**：

| 呼叫端 | bytes | 目標 |
|---|---|---|
| `ida-overlay13-move-probe.json` | `9A 34 00 38 01` | `0138h:0034h` |
| `ida-overlay13-move-budget-step.json` | `9A 34 00 38 01` | 同上 |
| `ida-overlay24-effect-apply-one.json` | `9A 34 00 38 01` | 同上 |
| `ida-overlay22-spell-dispatch.json` | `9A 3E 00 38 01` | `0138h:003Eh` |
| `ida-overlay25-nearby-opponents.json` | `9A 3E 00 38 01` | 同上 |

五個不同 overlay 只呼叫同一段的兩個 offset，且 segment 是硬編常數——這是 Borland
overlay 的 **entry stub 表**特徵，不是一般函式位址。真正的被呼叫者由 stub 內的
`CD 3F` 中斷加 overlay 編號與段內 offset 決定，必須先解 stub 才知道目標落在哪個
overlay 的哪個 offset。方法見
`~/.claude/knowledge-base/retro/borland-tpov-overlay-re.md`（`CD 3F` entry stub、
far call 目標查不到函式、stub offset 撞號要比 segment）。

`docs/audit/dos-ovr-manifest.json` 有 774 個 entry 的 `code_offset`／
`control_file_offset`／`executable_file_offset`，但沒有段載入位址，因此 stub 的
內容要回到 `START.EXE` 的 resident 部分取得。

這也是全域反組譯規則的實例：同時引用 IDA 命名與檔案偏移時必須逐項標明基準，
不可把兩種基準的數值並列成同一個位址。

### IDA 操作上已踩過的兩個坑

- 對已存在的 `.i64` 直接跑 `idat -A -B` 會以
  `Failed to initialize IDA as library (error code 1)` 失敗，要對 raw bin 重跑。
- raw binary 沒有 entry point，IDA 不會自動建立任何函式，函式清單會是空的；
  必須由 `POOL_IDA_SEEDS` 明確種入。
- 每筆 3 bytes 的前兩 byte 語意（合理推測是格座標，但未證）。
- 迴圈從 1 起算，因此 `6678h`（＝`6676h+2`，即筆 0 的第三欄）與筆 0 的前兩欄
  是否為保留槽或另有用途，屬 `strong inference`，未證實。
- `6674h` 表與 `6CD7h` 陣列的容量上限，以及超量時的行為。

## READY 契約

1. 篩選只依 combatant record `+10Eh` 與 mover 的對立值比較，不得改用距離、
   HP 或任何未證欄位。
2. 輸出必須保留原始順序：壓縮是往前搬移，不是重新排序。
3. 索引 1-based 的原始語意要保留在資料層；轉成 Go 的 0-based 只能發生在
   邊界轉換處，且要有測試釘住。
4. 在 `sub_13BE` 閉合前，不得自行推定鄰近格位的列舉順序或範圍，也不得用
   「八方向相鄰」之類的現代假設替代。
