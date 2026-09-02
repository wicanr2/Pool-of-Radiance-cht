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

- `sub_13BE` 的產生規則：格位如何列舉、距離參數如何影響、是否含 mover 自身。
  這是完整 occupancy 的另一半，也是把 `attackTargetID` 接進 probe 的前提。
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
