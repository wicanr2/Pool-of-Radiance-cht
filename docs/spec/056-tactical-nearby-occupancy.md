# Spec 056：戰術鄰近格位查詢與敵對側篩選

狀態：READY。日期：2026-09-02。

證據等級：overlay 25／31／32 三個函式與 `DS:2860h` 佔格表為 `exact`；
`arg_0`／`arg_2` 遠指標的內容、combatant record `+2` 欄位、以及兩個座標分量
何者為 X 何者為 Y 仍為 `待證`。

## 為什麼需要這一段

spec 053 的 `ResolveMovementProbe` 以 `attackTargetID` 決定目的格是進入還是攻擊，
但誰提供那個 target 一直未閉合。本規格把整條鏈打通：由一個 mover 的座標與體型，
產生候選格、比對場上每個 combatant 的佔格、算出成本與可及距離，最後篩出敵對陣營。

## 證據

| 檔案 | 內容 | 輸入 SHA-256 |
|---|---|---|
| `docs/audit/ida-overlay25-nearby-opponents.json` | overlay-25 `246Dh..2591h`（110 條） | `9fede24b…50c0e` |
| `docs/audit/ida-overlay25-opposing-side-selector.json` | `sub_23F5`，由 mover 取得對立陣營值 | 同上 |
| `docs/audit/ida-overlay31-nearby-cell-builder.json` | overlay-31 `0912h..0BDFh`（294 條）與 `0579h..0912h` | `64f1f7b8…151df` |
| `docs/audit/ida-overlay32-footprint-offset-provider.json` | overlay-32 `0000h..0068h` 與 `0068h..00E3h` | `efc22ba8…2dfc8` |
| `docs/audit/ida-ds-footprint-offset-table.json` | `DS:2858h` 起 256 bytes 初始化資料 | `12811cbc…10d9f` |

位址基準：overlay 檔案內一律為 overlay-local file offset（base 0）；`DS:` 前綴為
遊戲共用資料段的偏移；`segment:offset` 形式為 `START.EXE` 的實模式位址。
三者不可混用。

## overlay stub 段是可計算的，不必掃描

`START.EXE` 的 MZ header 為 59 paragraphs，故段 `S` 的 file base 為 `3B0h + S×16`。
每個 stub 段開頭是 32 bytes 的 overlay 描述子，其後才是 entry stub：

| 位移 | 大小 | 內容 |
|---|---|---|
| +0 | 2 | `CD 3F`（INT 3Fh，載入本單元） |
| +2 | 2 | 0 |
| +4 | 4 | 本 overlay 在 `GAME.OVR` 內的 file offset |
| +8 | 2 | code size |
| +A | 2 | relocation size |
| +C | 2 | entry 數 |
| +E | 2 | 前一個 overlay 的 stub 段（第一個為 0，構成回溯鏈） |
| +10 | 16 | 0（載入時的執行期欄位） |
| +20 起 | 5×N | entry stub：`CD 3F` ＋ word code offset ＋ byte flags |

這四個欄位對 38 顆 overlay 全部與 `workplace/re-pool/ovr-manifest.json` 逐項相符，
因此「某個 far call 的段落屬於哪一顆 overlay」是查表得到的，不需要靠掃描
`55 89 E5` 之類的序言特徵去猜。本規格用到的三段：

| 段 | overlay | code size | entry 數 |
|---|---|---|---|
| `010Ah` | overlay-25 | `30C1h` | 53 |
| `0138h` | overlay-31 | `0BE6h` | 8 |
| `013Dh` | overlay-32 | `12ECh` | 24 |

於是 `0138h:003Eh` ＝ overlay-31 entry 6 → `0912h`；
`0138h:0034h` ＝ entry 4 → `0579h`；`013Dh:002Ah` ＝ overlay-32 entry 2 → `0000h`。

## 三個 DS 結構

三者都是 Turbo Pascal 的 1-based 陣列，編譯器把「基底 − 元素大小」折進定址常數，
所以程式碼裡看到的常數比陣列真正的起點小一格。這是三處都出現的同一個模式，
讀的時候不要把折過的常數當成陣列起點。

**佔格偏移表**（唯讀，初始化資料）：折疊常數 `2858h`，實際起點 `DS:2860h`，
`array[1..4] of array[0..3] of packed record dx, dy: shortint end`，每列 8 bytes。
`dx` 為負（`0FFh`）代表該列到此為止。

| 體型類別 | 佔格 | 形狀 |
|---|---|---|
| 1 | `(0,0)` | 1 格 |
| 2 | `(0,0) (0,1)` | 沿分量 B 佔兩格 |
| 3 | `(0,0) (1,0)` | 沿分量 A 佔兩格 |
| 4 | `(0,0) (1,0) (0,1) (1,1)` | 2×2 |

`2860h..287Fh` 之後緊接著法術名稱字串，表只有這四列。

**戰場 combatant 表**：折疊常數 `5E85h`，筆數在 `DS:5E88h`，元素自 `DS:5E89h` 起，
每筆 4 bytes：`+0` 座標分量 A、`+1` 座標分量 B、`+2` 未定、`+3` 體型類別
（0 代表該筆不參與）。

**結果緩衝**：`DS:6674h` 是一個遠指標（offset 在 `6674h`、segment 在 `6676h`），
筆數在 `DS:6678h`，元素自 `DS:6679h` 起，每筆 3 bytes：
`+0` combatant 索引、`+1` 成本、`+2` 可及級數。程式碼中的折疊常數是 `6676h`。

## 佔格查詢：overlay-32 `0000h`（`13Dh:2Ah`）

`function(var b, a: shortint; col, row: byte): boolean`，`retf 0Ch`。

`row` 為 0 直接回 false。否則以 `row×8 + col×2` 查佔格偏移表，把偶數位元組寫進
`a`、奇數位元組寫進 `b`；`a` 為負時回 false，否則回 true。呼叫端把 `a` 加到座標
分量 A、`b` 加到分量 B。

同一顆 overlay 的 `0068h` 是相對座標展開：走訪 combatant 1..`ds:5E88h`，
把每筆的分量 A、B 減去 `ds:6674h` 所指結構的 `+2`、`+3`，分別寫入
`DS:5FA8h` 與 `DS:5FF0h` 兩個以 combatant 索引定址的 byte 陣列。

## 產生端：overlay-31 `0912h`

`retf 0Eh`，14 bytes 參數：

| 參數 | 位置 | 語意 |
|---|---|---|
| `arg_0`／`arg_2` | `bp+6`／`bp+8` | 遠指標，原封轉給 `sub_419`（內容待證） |
| `arg_4` | `bp+0Ah` | mover 的體型類別 |
| `arg_6` | `bp+0Ch` | 可及級數；小於 8 直接採用，否則逐級搜尋最小值 |
| `arg_8` | `bp+0Eh` | word，成本初值 |
| `arg_A` | `bp+10h` | 基準座標分量 B |
| `arg_C` | `bp+12h` | 基準座標分量 A |

流程：

1. `var_3 = 0..3` **四次**迭代，以 `arg_4` 為列、`var_3` 為欄查佔格表，
   偏移加上基準座標後存成四個候選格；查詢失敗的格子寫 `0FFh` 作無效標記。
   迭代次數與表的每列四組偏移一致。
2. `ds:6678h` 歸零。走訪 combatant `1..ds:5E88h`，`+3` 為 0 者跳過。
3. 對每個 combatant，同樣以它自己的體型類別展開四個佔格。
4. 兩層迴圈（自身四個候選格 × 對方四個佔格，各自跳過無效格）：
   - `sub_579(候選格分量A, 分量B, 對方格分量A, 分量B, arg_6)` 為 false 則跳過；
   - `sub_419(候選格分量A, 分量B, @對方格分量A, @分量B, @成本, arg_2:arg_0)`
     為 false 則跳過，為 true 時成本可能被就地更新；
   - 記錄成本最小的那組（候選格索引與對方佔格索引）。
5. 該 combatant 只要有任一組成立就追加一筆結果：`+0` 寫 combatant 索引、
   `+1` 寫最小成本的低位元組、`+2` 寫可及級數——`arg_6 < 8` 時直接沿用 `arg_6`，
   否則自 0 起遞增呼叫 `sub_579`，取第一個成立的級數。
6. 收尾呼叫 `sub_2E`，其回傳值即本函式的回傳值。

`sub_579`（＝ `0138h:0034h` 指向的 overlay-31 `0579h`）是「兩格之間是否在第 N 級
可及範圍內」的判定，同時被 overlay-13 的 move-probe 與 move-budget-step、
overlay-24 的 effect-apply 共用。

## 篩選端：overlay-25 entry 32

`246Dh..2591h`，`retf 6`：`arg_0` 是 word 成本預算，`arg_2`／`arg_4` 組成 mover 的
遠指標。它以三個 far helper 由 mover 取出基準座標分量 A、分量 B 與體型類別，
連同 `arg_0` 與常數 `0FFh`（可及級數＝搜尋模式）呼叫上一節的 `0912h`，
然後：

1. 由 `ds:6678h` 取筆數，索引自 1 起算。
2. 每筆的 `+0` 是 combatant 索引，乘 4 查 `ds:6517h` 的遠指標表取得該 combatant 的
   record，`+10Eh` 是陣營欄位。
3. 只留下 `record[+10Eh] == sub_23F5(mover)` 者，以 far memcpy（`5BBh:1692h`，
   長度 3）原地往前壓縮，結束後把 `ds:6678h` 更新為壓縮後的筆數。
4. 第二段迴圈把壓縮後每筆的 combatant 索引寫進 `ds:6CD7h`，每筆 1 byte、
   同樣 1-based。回傳篩選後的筆數。

因此本函式同時是 in-place filter（`6674h` 結構）與索引匯出（`6CD7h` 陣列）。

## 尚未閉合

- `sub_419`（overlay-31 `0419h`）的成本函式本體，以及 `arg_0`／`arg_2` 遠指標指向
  什麼；`0912h` 的成本語意要靠它才能定。
- `sub_579` 的可及級數定義（級數 0..7 各自對應什麼幾何範圍）。
- 兩個座標分量何者為 X 何者為 Y。目前只知道分量 A 對應 combatant record `+0`
  與偏移表的偶數位元組，分量 B 對應 `+1` 與奇數位元組。
- combatant record `+2` 欄位。
- `sub_2E` 的收尾動作與回傳值語意。
- `6674h` 結構與 `6CD7h` 陣列的容量上限，以及超量時的行為。
- `013Dh:0025h` 的 code offset 是 `FFFFh`，是未使用的 entry 槽，用途待證。

## READY 契約

1. 佔格表照 `DS:2860h` 的四列原樣搬進 game pack，不得改寫成「八方向相鄰」
   之類的現代假設，也不得自行補第五列。
2. 候選格與佔格的列舉都是四次，`dx` 為負即停止；無效格以 `0FFh` 標記而不是跳過，
   因為索引位置本身要保留。
3. 篩選只依 combatant record `+10Eh` 與 mover 的對立值比較，不得改用距離、
   HP 或任何未證欄位。
4. 輸出必須保留原始順序：壓縮是往前搬移，不是重新排序。
5. 索引 1-based 的原始語意要保留在資料層；轉成 Go 的 0-based 只能發生在
   邊界轉換處，且要有測試釘住。
6. 引用位址時必須標明基準。IDA 對 far call 目標產生的 `sub_XXXX` 名稱是把
   `segment×16 + offset` 線性化的結果，不是 overlay file offset。

## IDA 操作備忘

- 不要覆寫容器內的 `HOME`。`idapyswitch` 選定的 interpreter 記在 `$HOME/.idapro`，
  `-e HOME=<掛載點>` 會讓 IDAPython 載不起來，症狀是 rc=0、沒有任何輸出、
  也沒有輸出檔——與腳本寫錯完全同形。
- 對已存在的 `.i64` 直接跑 `idat -A -B` 會以
  `Failed to initialize IDA as library (error code 1)` 失敗，要對 raw bin 重跑。
- raw binary 沒有 entry point，IDA 不會自動建立任何函式，函式清單會是空的；
  必須由 `POOL_IDA_SEEDS` 明確種入。
