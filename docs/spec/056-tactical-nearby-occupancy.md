# Spec 056：戰術鄰近格位查詢與敵對側篩選

狀態：CONFORMED（overlay-25 entry 32 的完整資料流、篩選與輸出語意；鄰近格位產生者定位到 overlay-31 `0912h` 與其三候選結構）；
DRAFT（`13Dh:2Ah` 的偏移表、每筆前兩 byte 的語意、`6674h` 表的容量上限）。
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

## 鄰近格位表的產生者：overlay-31 `0912h`（2026-09-02 閉合定位）

`0138h` 是 Borland overlay 的 **entry stub 段**，不是函式所在段。stub 取自
`START.EXE`：MZ header 為 59 paragraphs，故段 `0138h` 的 file base 是
`3B0h + 138h×16 = 1730h`，每個 stub **5 bytes**（`0034h`、`0039h`、`003Eh` 間隔 5）：

| stub | bytes | 目標 offset |
|---|---|---|
| `0138h:0034h` | `CD 3F 79 05 00` | `0579h` |
| `0138h:003Eh` | `CD 3F 12 09 00` | `0912h` |

`CD 3F` 即 INT 3Fh，Borland overlay manager 的載入中斷。目標 overlay 由實測定出：
掃過全部 overlay，**只有 `overlay-31.bin` 在 `0579h` 與 `0912h` 兩處同時是
`55 89 E5`（`push bp; mov bp,sp`）函式序言**；IDA 對該檔解出的兩個函式邊界
`0579h..0912h` 與 `0912h..0BDFh` 相鄰不重疊，且 `0BDFh` 正好是同一 stub 表另一筆
（`1750h`）的目標，三項互相支持。證據：
`docs/audit/ida-overlay31-nearby-cell-builder.json`，overlay-31 SHA-256
`64f1f7b8…151df`。

因此 spec 053 所稱的「overlay-25 entry 32 呼叫 `sub_13BE`」，真正的被呼叫者是
**overlay-31 `0912h`**。

### `0912h` 的結構（exact）

1. 先以迴圈 `var_3 = 1..3` 產生**三個候選**：每次以 `arg_4` 與迴圈索引呼叫另一個
   stub `13Dh:2Ah`，取回兩個 byte 偏移；回傳 `al != 0` 時把偏移分別加上基準座標
   `arg_C`／`arg_A`，存進兩個堆疊陣列（`var_F` 一組、`var_13` 一組）；回傳 0 時
   該格寫 `0FFh` 作為無效標記。**列舉是三格，不是八方向掃描。**
2. 清空 `ds:6678h`（結果筆數，即 spec 前段那張表的計數），再以 `ds:5E88h` 取得
   combatant 總數，逐一走訪。
3. combatant 屬性表基底為 **`ds:5E85h`，每筆 4 bytes**；`5E85h` 與 `5E88h` 是同一筆
   內的兩個欄位（相距 3）。索引以 `index×4` 計算，第 0 筆的欄位同時被當成總數使用
   ——這與結果表 `6674h`／`6678h` 的「表首兼放 count」是同一種佈局慣例，
   使前段標為 `strong inference` 的那一點多了一個獨立例證。
4. 逐 combatant 再次呼叫 `13Dh:2Ah`，這次帶入該 combatant 的 `+3` 欄位與內層索引，
   偏移加上該 combatant 的座標欄位（`5E85h` 一組）後展開——對應大型怪佔多格的情形。

## 尚未閉合

- **`13Dh:2Ah`**：偏移表的實際內容。它同樣是 overlay stub，要照本節的方法再解一層
  （`013Dh` 的 file base 為 `3B0h + 13Dh×16 = 1780h`）。解出它才知道三個候選格的
  幾何關係，以及大型怪的 footprint 展開規則。
- `arg_4`／`arg_A`／`arg_C` 的來源與語意（距離參數與基準座標的對應）。
- `0138h:0034h` → overlay-31 `0579h` 是被 overlay-13 的 move-probe 與
  move-budget-step、overlay-24 effect-apply 共用的另一個服務，尚未解讀。

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
