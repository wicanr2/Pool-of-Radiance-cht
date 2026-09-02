# Spec 053：Pool 戰術移動預算與八方向成本

狀態：CONFORMED（初始化、effect code 12 accumulator、顯示單位、
cardinal／diagonal 扣除 primitive、八方向座標 delta、目的格 probe 分派、
`2758h` raw typed table）；
DRAFT（effect IDs 的 spell 名、全域 bonus 語意、reaction 的個別 predicate
與戰術畫面）。日期：2026-09-01，2026-09-02 依 spec 057／058 修訂目的格
probe 的歸屬與 `2758h` 四欄語意。

## 證據

- `overlay-13.bin` SHA-256：
  `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390`。
- `overlay-08.bin` SHA-256：
  `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f`。
- IDA Pro 9.4、overlay-local file offset、base 0、16-bit metapc：
  `docs/audit/ida-overlay13-move-budget-init.json` 保存 `0123h..018Eh`；
  `docs/audit/ida-overlay13-move-budget-step.json` 保存 entry 5 `0719h..08B6h`；
  `docs/audit/ida-overlay13-opportunity-attack.json` 保存 entry 6 `08B9h..0C51h`；
  `docs/audit/ida-overlay08-move-command.json` 保存 `09C3h..0D18h`。
- `docs/audit/ida-overlay24-effect-dispatch.json` 與
  `docs/audit/ida-overlay24-effect-apply-one.json` 保存 effect dispatcher entry 3；
  `docs/audit/ida-overlay12-effect-table-init.json` 保存 handler table 初始化；
  `docs/audit/ida-overlay12-movement-effect-handlers.json` 保存 IDs `27h/2Ah/3Ah` handlers。
- `docs/audit/ida-overlay22-haste-slow-producers.json` 與
  `docs/audit/ida-overlay22-opposed-speed-effect.json` 保存 `is Hasted`／`is Slowed`
  的相反效果移除鏈；`docs/audit/ida-overlay12-held-effect-producers.json` 保存
  `3Ah` 新增與 `is held fast` 玩家訊息。
- `docs/audit/ida-overlay22-spell-table-base-references.json`、
  `docs/audit/ida-overlay22-spell-dispatch.json` 與
  `docs/audit/ida-overlay22-spell-handler-table.json` 保存另一層 `DS:6A78h` far-pointer
  分派表。這一層依 selector × 4 間接呼叫；Hasted／Slowed handlers 分別位於槽 48／55，
  不是 effect IDs `27h`／`2Ah`，兩種編號不可混用。
- `docs/audit/ida-overlay09-movement-cell-table.json` 與同名的 overlay 10／13／22／32
  報表保存 `DS:2758h` 第一 byte 的其他 consumers；它們支持這是每四 byte 一筆的
  戰術格位類別表。四欄初始化來源由下一組 START.EXE 證據閉合；兩個 path byte 的
  玩家語意仍刻意保留為 DRAFT。
- `docs/audit/ida-start-mz-segments.json` 由 START.EXE 的正常 MZ loader 證實 entry
  DS selector `5952` 對應 `dseg` linear `95232`、file offset `30640`；
  `docs/audit/ida-start-combat-cell-class-table.json` 再從 `DS:2758h`／file offset
  `40712` 精確匯出 `0x108` bytes，即 66 筆四 byte 記錄。
- `docs/audit/ida-overlay31-combat-cell-path.json` 保存地圖格 `+7` 類別碼對
  `2759h/275Ah` 的路徑函式 consumers；`docs/audit/ida-overlay10-combat-cell-presentation-init.json`
  與 overlay-32 `275Bh` consumers 保存第四欄傳入 tactical tile drawing service，以及
  battlefield initialization 對 presentation code `16h` 的類別替換。
- `docs/audit/ida-start-tactical-direction-deltas.json` 保存 `DS:274Ah` 的 X 與
  `DS:2753h` 的 Y signed-byte 表；`docs/audit/ida-overlay13-move-budget-step.json`
  保存 entry 5 取 delta、提交座標與扣除 movement 的完整順序。
- `docs/audit/ida-overlay13-post-move-reactions.json` 保存 entry 5 提交座標後呼叫的
  `0630h`；`docs/audit/ida-overlay25-nearby-opponents.json` 與
  `docs/audit/ida-overlay25-opposing-side-selector.json` 保存 nearby list 依 mover
  `+10Eh` 的反值篩成敵對側，以及後續逐筆進 attack wrapper 的鏈。
- Spec 049 已由原版角色資料頁 consumer 證實 record `+11Ch` 是 base movement。

## 原版資料流（exact）

1. overlay-13 `0123h` 讀 record `+11Ch`。record `+10Eh == 0` 時，再加 combat global
   `[4937h]+6E4h` 的 word；結果會先寫回 byte，因此低八位 wrap 是契約的一部分。
   這個 global 的玩家語意尚未閉合，不能先命名為負重、加速或隊伍 bonus。
2. byte 結果不在 1..96 時改為 1，接著乘 2 放入 `DS:6778h` effect accumulator，呼叫
   effect code `12h`，最後回傳並由 entry 1 寫入 runtime `+6`。
3. overlay-08 Move command 顯示 runtime `+6 / 2`；原始 Pascal 字串是
   `Move/Attack, Move Left = `。因此 `+6` 採半步單位，不是畫面直接顯示的格數。
4. 八個輸入 `H/I/M/Q/P/O/K/G` 映射 direction 0..7。overlay-13 entry 5 對 direction
   右移一位並檢查 carry：偶數方向成本 2，奇數方向成本 3。若成本大於剩餘 budget，
   budget 直接寫 0；否則相減。
5. Move handler 只有 runtime `+6 > 1` 才繼續讀方向；函式結束時若 `+6 < 2`，把它寫 0。
   正常走一步沒有修改 initiative `+3`，而是回到同一角色的命令流程。因此 remake
   不得把每次移動錯接成回合結束。
6. 目的格的兩個 byte 由 **overlay-32 entry 19（`0CB9h`）** 產生，細節見 spec 058。
   Move handler 先看第一個 byte：非零即把它當索引，經 `DS:6517h` far-pointer table
   取得目標並進 attack wrapper；這條分支先於任何 `2758h` threshold 檢查。
   它是「碰到目標便攻擊」，不是自由穿越。第一個 byte 為零且第二個也為零時，
   目的格在盤面外，Move handler 詢問玩家是否離開戰鬥（overlay-13 entry 7），
   那不是「擋住」。overlay-13 entry 6（`08B9h`）做的是離開威脅區的反應攻擊，
   在門檻通過之後才呼叫。
7. 第一個 byte 為零、第二個非零時，第二個 byte 乘 4 後查 `[index+2758h]` 的第一 byte；
   該 threshold `<= runtime +6` 才呼叫 overlay-13 entry 5 提交方向步。threshold 太高會
   顯示訊息並結束本次 Move。`2758h` 的第一 byte 是 entry threshold；同一筆的第二、
   第三 byte 是直線追蹤用的 Level 與 Block（spec 057），第四 byte 語意仍未定。
   表共 32 筆，索引 0 與方向表尾端共用儲存空間。
8. entry threshold 只負責准入；真正提交時仍由 entry 5 扣 cardinal 2／diagonal 3。
   不得自行把 threshold 再扣一次，也不得以 2／3 成本取代 threshold gate。
9. entry 5 由 direction 查 signed-byte delta：
   X=`[0,+1,+1,+1,0,-1,-1,-1]`，Y=`[-1,-1,0,+1,+1,+1,0,-1]`；加法結果寫回 byte。
   destination probe 必須先確保位置合法，座標 primitive 本身保留原版 byte wrap。
10. Y delta 的最後三 byte 恰好是 `DS:2758h..275Ah`，與 combat cell class record 0
    的前三欄共享儲存空間。這是原始 data layout，不是兩張表擷取錯位；重建時可分開
    typed 表達，但驗證 fixture 必須保留兩個 view 對同一組 `01 00 FF` bytes。

## 座標提交後的反應攻擊（部分 exact，完整 gate DRAFT）

entry 5 寫入新 X／Y、更新戰術格位與重畫後，呼叫 `overlay-13:0630h`。該函式：

1. 以 mover 與距離參數 1 呼叫 overlay-25 entry 32；後者從附近格位集合中，只留下
   combatant record `+10Eh` 等於 mover `+10Eh` 反值者，因此候選集合 exact 為敵對側。
2. 逐候選要求其 runtime `+7 != 0`，再通過一個尚未命名的 entry 16 predicate 與一個
   range／geometry service；通過者會先把候選 runtime `+7` 清零，再以候選為 attacker、
   mover 為 target 呼叫同一 attack wrapper。
3. 這證實「提交移動後可由附近敵方消耗一次性 `+7` 狀態發動反應攻擊」；但 `+7`
   producer 與兩個 predicate 尚未閉合，所以目前不可把所有鄰接敵人一律設定為可攻擊，
   也不可先宣稱它精確等同某一版 AD&D 的 attack of opportunity 規則。

## `DS:2758h` 戰術格位類別表

START.EXE 初始化資料是固定 66×4 bytes；戰術地圖每格 record `+7` 保存類別索引。
四欄按原位址保留：

| 位移 | 現行 typed 名稱 | 已證實用途 | 等級 |
|---|---|---|---|
| `+0`／`2758h` | `EntryThreshold` | 玩家 Move 的剩餘 budget gate；其他路徑 consumer 以 `FFh` 排除 | exact |
| `+1`／`2759h` | `PathByte1` | overlay-31 路徑候選計算讀取；原始 66 筆初始化值全為 0 | exact raw，玩家語意 DRAFT |
| `+2`／`275Ah` | `PathByte2` | overlay-31 路徑候選 gate 讀取；原始值集合為 `00h/02h/FFh` | exact raw，玩家語意 DRAFT |
| `+3`／`275Bh` | `PresentationCode` | overlay-32 傳入 tactical tile drawing service；overlay-10 依 code `16h` 替換地圖格類別 | exact consumer |

不能把 66 筆中的 `FFh` 一律解釋成同一種牆：`+0` 與 `+2` 是不同欄，且同一筆可有
不同組合。也不能因 `PathByte1` 初值全零就刪掉它；原版路徑函式仍明確讀取該欄。

## Effect code 12（exact accumulator）

1. overlay-24 entry 3 對 code `12h` 依序分派 effect IDs `27h`、`2Ah`、`3Ah`。
   overlay-12 entry 1 初始化的 far-pointer table 將它們分別映射到 entry 36 `0C67h`、
   entry 41 `10CAh`、entry 53 `145Ah`。
2. ID `27h` 將 `DS:6778h` 以 byte 左移一位；handler 還會設定 effect record `+3h`
   的 bit 4 並增加 actor word `+30h`，所以完整 effect runtime 不能只呼叫純 accumulator。
3. ID `2Ah` 將 zero-extended accumulator 做 signed `idiv 2`；對 byte 值等價於向零截斷 `/2`。
4. ID `3Ah` 把 actor runtime movement `+6` 清零；當 `DS:677Bh != 0`（初始化 caller 先設 1）
   時，也把 accumulator 清零。因此 movement initialization 的結果是依序 double、halve、zero。
5. 三個 ID 的 spell／item 名稱尚未沿「新增 effect」producer 閉合，code 與 schema
   必須保留數字。overlay-22 另有 selector × 4 的法術處理表；其槽 48／55 分別進入
   Hasted／Slowed handler，但這只證實分派表的兩個 selector，不會把 selector 48／55
   或 effect `27h`／`2Ah` 自動升格成法術名稱。

### 玩家語意的證據等級

- ID `3Ah`：**exact 為 held 狀態 effect，spell 名仍 DRAFT**。overlay-12 `18D3h` 與
  `1953h` 都把 `3Ah` 傳給 effect 新增鏈；後者同一函式顯示原始字串
  `is held fast`。這證明狀態語意，但尚未區分是哪一種 Hold spell／怪物能力。
- ID `27h`：**strong inference 為 Haste**。overlay-22 entry 61 在顯示
  `is Slowed` 前把 `27h` 交給共用 effect 移除鏈；ID `27h` 的 movement handler 又會
  double 並使角色 `ages`。兩份獨立證據一致，但尚缺「新增 27h 與 Haste spell 名」
  同一條 call chain，所以 code／schema 仍保留數字。
- ID `2Ah`：**strong inference 為 Slow**。overlay-22 entry 57 在顯示
  `is Hasted` 前移除 `2Ah`，其 movement handler 做 halve。理由同上，尚不升格為 exact 名稱。

共用速度函式在移除相反 effect 後會重新執行 effect code `12h`，證明 double／halve
不是只在初始建 combatant 時生效；狀態變更後必須立即重算 movement budget。

## Typed primitives

`InitialMovementBudgetBeforeEffects(baseMovement, applyGlobalBonus, globalBonus)`：

- 依原版先做 byte wrap，再套 1..96 clamp，最後乘 2；
- `applyGlobalBonus=false` 時必須完全忽略 bonus；
- 回傳 effect 前的值，caller 再依下列 primitive 套用 effect code 12。

`SpendMovementStep(budget, direction)`：

- direction 只接受 0..7，其他值失敗即關閉；
- 偶數扣 2、奇數扣 3；不足時回傳 0，不做 unsigned wrap；
- 不處理碰撞、位置提交、地形額外成本、機會攻擊或 initiative。

`ApplyMovementEffectIDs(budget, effect27, effect2A, effect3A)`：依原 dispatcher 順序做
byte double、整數 halve、zero。它只重現 accumulator；effect `27h` 的 record bit／actor
word side effects 必須由未來完整 effect runtime 負責。

`ResolveMovementProbe(budget, attackTargetID, entryThreshold)`：

- `attackTargetID != 0` 一律先回 `MovementAttack`，即使 budget 為零或 threshold 不可進；
- 無目標且 `entryThreshold > budget` 回 `MovementBlocked`，否則回 `MovementEnter`；
- 本 primitive 不解析 `2758h` 的其餘三欄、不取得 target pointer，也不扣 2／3 step。

`AdvanceTacticalCoordinate(x, y, direction)`：

- direction 0..7 依原始 X／Y signed-byte 表相加；非法方向失敗即關閉；
- 結果保持 byte wrap；戰場邊界與 collision 必須在呼叫前由目的格 probe 驗證；
- 本 primitive 不更新 occupancy、不重畫，也不執行 post-move reactions。

`gamepack.ParseCombatCellClassTable(raw)`：

- 只接受 Pool 的 `66×4 = 264` bytes，短一 byte、多一 byte都失敗即關閉；
- 保留四個 raw byte，不把尚未閉合的路徑欄轉成布林或現代 terrain enum；
- typed parser 是作品資料層，不應移到共用 engine；共用引擎日後只消費作品中立的
  tactical cell contract。

## 驗收

- 固定 Slums ORC base movement 9 得到 effect 前 budget 18。
- 固定 bonus 套用／忽略、負值導致 clamp、96 上界、97 越界與 byte wrap。
- 固定 cardinal／diagonal、剛好足夠、不足與非法 direction。
- 固定三個 effect 單獨值、`27h→2Ah` 順序、byte wrap 與 `3Ah` 最後歸零。
- 固定空格可進、threshold 相等可進、超額／`FFh` 阻擋，以及目標優先於 threshold gate。
- 固定八方向座標、`(0,0)` 朝 direction 7 的 byte wrap，以及非法 direction 8。
- 固定 START.EXE 全 `0x108` bytes、代表性首／中／末記錄，以及 263／265-byte 負對照。
- Pool 全套 `go test ./...`、`go vet ./...`。
