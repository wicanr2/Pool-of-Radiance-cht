# Spec 147：怪物開打時的裝備重算（overlay-25 entry 7）

狀態：CONFORMED（開打時對怪物跑 entry 7、`+2Dh` 不寫、`0DB4h` 的傷害骰抄寫、`+0F8h`／`+0FCh`
兩格、兩隻獸人頭目與十六隻獸人的九個欄位逐位元組對上 dosgolem 執行期記錄）；DRAFT（彈藥
消耗、NPC 那一側）。日期：2026-09-26。主台帳：GitHub issue #93（#76 留下，spec 142 的
DRAFT 表那一列）。

## 輸入

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`（`MON*CHA.DAX`、
  `MON*ITM.DAX`、`poolrad/items`）。
- overlay-25（`workplace/ovr/overlay-25.bin`）SHA-256
  `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e`，overlay-local
  file offset、base 0，`coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel`。
- 執行期：`docs/audit/dosgolem-monster-recompute-runtime.json`（`tools/dosgolem-monster-recompute-receipt.py`
  從 `workplace/orc-drive/thac0-recs/shots.json` 抄出；dosgolem `9bd269a`、`start.exe`
  `12811cbc…10d9f`，獸人家開打那一幀，與 spec 063 的 THAC0 收據同一批讀取）。

## 誰叫、何時叫（exact）

spec 063〈怪物開打時也跑同一支〉：overlay-10 開打初始化 `1ED6h` 在 `1F9Ch` 叫 `1380h`，
沿 `DS:5CF4h` 的 combatant 串列對每一筆呼叫 overlay-25 entry 7（`13A7h`：`9A 43 00 0A 01`）。
隊員與怪物是**同一支**，所以 remake 共用 `recomputeFromBase`（`internal/gamepack/record_recompute.go`），
不另寫一份。LOAD MONSTER（overlay-03 `044Dh`）本身不叫它；物品串列在那裡載好（spec 142），
重算在開打時。

## entry 7 寫哪些欄位（exact，逐段）

| 位址 | bytes（節錄） | 作用 |
|---|---|---|
| `0BC4h..0BEAh` | `26 89 85 CC 00` | 清十三個槽指標 `+0CCh + i×4` |
| `0C21h..0D82h` | `26 80 7D 34 00`（`0C65h`）| 沿 `+0C8h`：每件的重量（`+37h`，`+39h` 非 0 再乘）加進 `+102h`；**穿戴中**的才認槽：型別表類別 0..8 → `+0CCh + 類別×4`（`0CADh`），9 → `+0F0h`／`+0F4h`，型別索引 `49h` → `+0F8h`（`0D05h`：`26 80 7D 2E 49`），`1Ch` → `+0FCh`（`0D21h`）|
| `0D85h..0DB2h` | `26 8B 85 88 00` | 七種錢加進 `+102h` |
| `0DB4h..0E1Bh` | `26 8A 95 A2 00`／`26 88 95 14 01` | n = 1、2：`+114h+n = +0A2h+n`、`+116h+n = +0A4h+n`、`+118h+n = +0A6h+n`——兩種形態的顆數、面數、加值抄進執行期那一段 |
| `0E43h..0E6Ch` | `26 8A 85 A9 00`… | `+111h = +0A9h`、`+11Ch = +72h`、`+110h = +2Dh`（`+2Dh` 只讀） |
| `0E7Eh..0EC9h` | `26 8B 85 CC 00` | `+0CCh` 空：`+110h += 12AEh`、`+119h += 1366h`（兩支都自己查 `+0AAh`，`12C8h`／`1380h`：`26 80 BD AA 00 00`） |
| `0ECEh..0ED5h` | `E8 28 F1` | 呼叫 entry 1（`0000h`）：有武器就蓋掉 `+110h`／`+115h`／`+117h`／`+119h`（spec 063 十一步） |
| `0EE8h..0F33h` | `E8 EE F2`／`E8 5D F3` | 穿戴中的每件：`01F8h`（盔甲改 `+11Ch`，spec 079）、`0281h`（AC 累加器，spec 080） |
| `0F80h..0F87h` | `E8 15 F4` | `039Fh`：負重壓腳程 |
| `0FA9h..0FFBh` | `26 88 85 12 01` | `+111h` 與 `+112h` 結算 |

所以一隻怪物開打時：命中照手上的武器、傷害第一種形態照武器（第二種不動）、AC 照盔甲與盾、
腳程照盔甲與負重。攻擊次數（`+0A1h`／`+0A2h`）不在這一支裡。

射程不是 entry 7 寫的欄位：overlay-13 `358Dh` 挑目標時讀 `+0CCh` 那件的型別表 `+0Ch`（spec 065）。
`+0CCh` 由這一支認回，所以拿弓的怪物射程是弓的。

`01F8h` 的盔甲腳程是**寫死的值不是上限**（`0240h`：`26 C6 85 1C 01 09`）：重 151..399 的盔甲把
`+11Ch` 設成 9，基礎 6 的獸人頭目穿上反而變 9。執行期收據證實（下節）。

## 與 remake 舊做法的差別

- 舊的 `RecomputeCombatFields` 沒有 `0DB4h` 那一段，也不認 `+0F8h`／`+0FCh`。補上之後，spec 063
  〈仍未閉合的點〉TARRY 沒武器時的 1d2 有了來源：建角寫下的 `+0A3h = 1`、`+0A5h = 2`，
  `TestRecomputeCombatFieldsMatchesThePremadeCharacters` 現在要求七名預設人物的三個傷害欄位
  全部重算回原值（包含 TARRY）。spec 063 契約第 7 條「不得順手填 `+115h`／`+117h`」由這一節
  取代：填的是抄寫，不是補徒手傷害。
- 戰場上怪物原本讀樣板的 `+111h`、`+11Ch` 與 `+0A2h..` 骰子，射程一律 1。現在開打時
  `applyMonsterGearStats`（`cmd/pool-game/monster_gear.go`）用 `FoeItems`（spec 142，第二隻起
  反序）重算，蓋掉 THAC0、AC、腳程、兩種形態的骰子、`Damage` 與 `AttackRange`。

## 驗證

- 執行期（exact）：獸人家二十隻怪物。MON2 block 15 的獸人頭目（短弓 2Bh 與 +1 盔甲都穿著）
  `+110h 44`、`+111h 56`、`+112h 54`、`+114h.. 00 01 00 06 00 00 00`（1d6）、`+11Ch 9`、
  `+102h 527`；block 14 三隻（武器沒穿）AC 55、1d8、腳程 9；block 4 十六隻獸人照樣板。
  remake 逐位元組相同：`TestMonsterRecomputeMatchesTheOrcHomeRuntimeRecords`（gamepack）、
  `TestMonsterGearMatchesTheOrcHomeRuntimeReceipt`（戰場上的 THAC0／AC／腳程／骰子／射程，
  依 `DS:6517h` 的 combatant 索引）。
- 負對照：`TestMonsterRecomputeChangesTheArmedLeaderAndKeepsBaseThac0`（樣板 55／1d8／6 → 56／1d6／9，
  `+2Dh` 不動）；`TestMonsterRecomputeAddsStrengthDamageOnlyWhenTheFlagIsSet`（吸血鬼 `+0AAh = 1`
  1d6+4 → +8，獸人 `+0AAh = 0` 不補）。
- 遠程：`TestArmedOrcLeaderShootsFromOutsideMeleeReach`：同一格、同一批隊員，拿弓的頭目在
  `foeTurn` 原地射人（一次攻擊、零步），沒拿武器的頭目要先走。
- 規模（`MON*CHA` 172 筆全部重算）：AC 變的 66 筆、腳程 39 筆；穿戴中的武器是遠程的只有
  四筆（兩份 ORC 1/44、7/44 帶短弓 2Ch、ORC LEADER 2/15 帶 2Bh、DRIDER 7/69 帶 29h）。

## 還沒接（DRAFT）

| 項目 | 等級 | 為什麼 |
|---|---|---|
| 彈藥消耗與「沒有箭不能射」 | unknown | 隊員那一側也沒有；overlay-13 的射擊路徑還沒讀 |
| NPC（`applyNPCCombatStats`）開打時也該跑 entry 7 | exact（`1380h` 對每一筆）| 不在 #93 範圍；NPC 仍讀自己帶的記錄 |
| 戰鬥中怪物換裝 | exact（`146Fh` 重算）| 怪物不走物品選單；AI 用物品（spec 096 entry 3）不改穿戴 |
