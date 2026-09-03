# Spec 058：目的格探測與戰術層資料版面

狀態：READY（`04C0h`、`0C61h`、`0CB9h` 三支函式與 Move handler 的分派；
戰術層各塊 DS 的位置與元素大小）；
DRAFT（類別 `1Eh` 的玩家語意、combatant record `+108h` 以下的欄位、
`5E89h` 表的 `+2` 欄位）。日期：2026-09-02。

## 為什麼需要這一段

`ResolveMovementProbe`（spec 053）需要兩個輸入：撞到的目標索引與目的格類別。
本規格閉合產生這兩個 byte 的函式，以及它們讀的那幾塊資料，
移動命令因此可以整條接起來。

## 證據

| 檔案 | 內容 | 輸入 SHA-256 |
|---|---|---|
| `docs/audit/ida-overlay32-destination-probe.json` | overlay-32 `0CB9h`、`04C0h`、`0C61h` 等 | `efc22ba8…2dfc8` |
| `docs/audit/ida-overlay08-move-command.json` | overlay-08 `09C3h..0D1Bh` 的 Move handler | `932ce281…a036f` |
| `docs/audit/ida-overlay13-opportunity-attack.json` | overlay-13 entry 6 `08B9h..0C51h` | `4d53df20…2390` |

## 戰術層的 DS 版面

各表都是 Turbo Pascal 的 1-based 陣列，程式碼裡的常數是折疊過的基底。

| 折疊常數 | 元素 | 實際起點 | 內容 |
|---|---|---|---|
| `25D4h` | 1 | `25D4h` | 3×3 步進方向查表（spec 057） |
| `274Ah` | 1 | `274Ah` | 九個方向的 X 位移（spec 056） |
| `2753h` | 1 | `2753h` | 九個方向的 Y 位移 |
| `2758h` | 4 | `2758h` | 戰術格位類別表，66 筆（spec 053、057） |
| `2858h` | 8 | `2860h` | 佔格偏移表，4 列（spec 056） |
| `5E85h` | 4 | `5E89h` | 戰場位置表，筆數在 `5E88h` |
| `6039h` | 1 | `6039h` | 佔用格陣列，50 寬，值是 combatant 索引，0 為空 |
| `6517h` | 4 | `651Bh` | combatant record 的遠指標表 |
| `6674h` | 4 | `6674h` | 指向戰術地圖的遠指標 |
| `6676h` | 3 | `6679h` | 鄰近查詢結果表，筆數在 `6678h`（spec 056） |
| `6CD7h` | 1 | `6CD8h` | 篩選後的 combatant 索引陣列 |

佔用格陣列佔 50×25 ＝ 1250 bytes，`6039h..651Ah`，結尾正好接上遠指標表的
第一筆 `651Bh`。這是兩塊資料相鄰的直接佐證。格位類別表同樣把
`2758h..285Fh` 鋪滿，尾端接上佔格表的第 1 列 `2860h`。

戰術地圖本身由 `6674h` 的遠指標取得：`+6` 是「跳過地形判定」旗標，
地形格自 `+7` 起、列距 50。`sub_419`（spec 057）與本規格的 `04C0h`
讀的是同一份。

## 逐格查詢：overlay-32 `04C0h`

`procedure(var class, occupant: byte; y, x: byte)`，`retf 0Ch`。

座標超出 `0 ≤ x ≤ 49`、`0 ≤ y ≤ 24` 時兩個輸出都寫 0。否則
`class` 取自地圖的地形格，`occupant` 取自 `6039h` 的佔用格陣列。

「盤面外」與「地形碼 0」因此在輸出上無法分辨，兩者都是 class 0——
上層也確實把它們當同一件事處理。

## 指標轉索引：overlay-32 `0C61h`

`function(combatant: far ptr): byte`，`retf 4`。逐一比對 `6517h` 表的
1..`ds:5E88h` 筆，相等即回傳該索引，找不到回 0。

## 目的格探測：overlay-32 `0CB9h`（`13Dh:7Fh`）

`procedure(var class, target: byte; direction: byte; mover: far ptr)`，`retf 0Eh`。

1. `target = 0`、`class = 17h`、內部門檻 `best = 1`。
2. 由 mover 遠指標取得索引，再由位置表取得它的 X、Y 與體型類別。
3. 對體型的四個佔格各做一次（查不到的槽跳過）：
   - 目的格 ＝ 佔格座標 ＋ 該方向的單格位移；
   - 以 `04C0h` 取出該格的佔用者與類別；
   - 佔用者是 mover 自己就當成 0，否則非零即寫進 `target`；
   - 類別為 0（盤面外）時 `class = 0` 並跳過其餘判斷，此後不再改變；
   - 類別為 `1Eh` 時同樣定案為 `1Eh`；
   - 其餘情形查地形表的 EntryThreshold，比目前的 `best` 大或相等才更新
     `best` 與 `class`。

所以 `class` 回報的是四個目的格裡**最難進**的那一個，`target` 是撞到的
任一 combatant。

## Move handler 如何用這兩個 byte

overlay-08 的 Move handler（`09C3h..0D1Bh`）：

1. 由按鍵得到方向 0..7，超出範圍就結束。
2. 呼叫 overlay-32 entry 14（`0AF6h`）設定面向，再呼叫 entry 19 取得兩個 byte。
3. `target` 非零 → 經 `6517h` 取得該 combatant，進 attack wrapper。
   這條分支先於任何門檻檢查，所以是「碰到目標便攻擊」。
4. `target` 為零且 `class` 為零 → 顯示提示並讀 Y／N。答 Y 就呼叫
   overlay-13 entry 7（`0C6Ch`）離開戰鬥，答 N 則不動。**盤面外不是「擋住」。**
5. 其餘 → 查 `[class×4 + 2758h]` 的 EntryThreshold，與 combatant record
   `+108h` 所指結構的 `+6`（剩餘步數）比較。門檻較大就顯示訊息並結束本次移動。
6. 門檻通過 → 呼叫 overlay-13 entry 6（`08B9h`）處理反應攻擊，再提交這一步。

## overlay-13 entry 6：離開威脅區的反應攻擊

`procedure(direction: byte; mover: far ptr)`，`retf 6`。

1. 以 mover 與距離參數 1 呼叫 overlay-25 entry 32，把 `6CD7h` 的結果
   （移動前鄰接的敵方）抄進區域陣列；沒有任何一筆就直接結束。
2. 把 mover 的位置**暫時**往 direction 推一格，再查一次，然後復原。
3. 兩份名單相減：移動後仍鄰接的從區域陣列裡清成 0，剩下的就是
   「這一步會脫離的敵人」。
4. 對每個剩下的敵人，依序通過數個 predicate（mover record `+10Dh` 非零、
   目標的兩個狀態查詢、`+108h` 結構的欄位比較、以及一次
   `overlay-31 0579h` 的朝向弧判定），通過者才發動一次攻擊，
   並把它 record `+112h` 的一次性旗標消耗掉。

因此反應攻擊的觸發條件精確地是「因這一步而失去鄰接」，
不是「所有鄰接的敵人」。

## READY 契約

1. 目的格探測必須跑完 mover 的全部佔格；只看單一格會讓大型怪的判定與原版不同。
2. `target` 的優先權高於任何門檻判斷。
3. `class` 為 0 要走「離開戰鬥」的詢問流程，不可當成一般的擋住。
4. `class` 的挑選是取 EntryThreshold 最大者，且 0 與 `1Eh` 一旦出現就定案。
5. 反應攻擊只針對「移動前鄰接、移動後不鄰接」的差集，不是全部鄰接敵人。
6. **體型類別查不到時，出界與地形檢查整段不生效**：類別為 0 或超出佔格偏移表的
   四列時，步驟 3 的四個槽全部跳過，`class` 維持初值 `17h`，於是既不會出現
   盤面外的 0、也不會查到任何地形門檻——探測回報的是「可以進」。原版靠更前面
   一道擋住這件事（離場的 combatant 不會被選成行動者，spec 062 契約 7），
   所以呼叫端必須自己保證那件事成立。讓體型 0 的 combatant 移動，它會走出盤面。
