# Spec 147：怪物開打時的裝備重算（overlay-25 entry 7）

狀態：CONFORMED（開打時對怪物跑 entry 7、`+2Dh` 不寫、`0DB4h` 的傷害骰抄寫、`+0F8h`／`+0FCh`
兩格、兩隻獸人頭目與十六隻獸人的九個欄位逐位元組對上 dosgolem 執行期記錄）；READY（隊伍 NPC
開打與戰鬥中換裝走同一支、ADD NPC 帶 MONnITM 物品，#97；呼叫鏈 exact，NPC 本身沒有執行期收據）；
彈藥消耗已由 spec 151 接上（#98）。日期：2026-09-26。主台帳：GitHub issue #93（#76 留下，spec 142 的
DRAFT 表那一列）、#97（NPC）。

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
- 遠程：`TestArmedOrcLeaderShootsFromOutsideMeleeReach`：同一格、同一批隊員。敵方回合先跑
  overlay-09 entry 9 重挑武器（spec 151）：block 14 的頭目穿上弓、原地射人（一次攻擊、零步）；
  block 15 的頭目釘頭錘與弓都穿著、三隻手，entry 9 把弓卸下，拿釘頭錘往前走。
- 規模（`MON*CHA` 172 筆全部重算）：AC 變的 66 筆、腳程 39 筆；穿戴中的武器是遠程的只有
  四筆（兩份 ORC 1/44、7/44 帶短弓 2Ch、ORC LEADER 2/15 帶 2Bh、DRIDER 7/69 帶 29h）。

## NPC（#97）

隊伍裡的 NPC 與怪物走同一支，沒有另外的分支。

| 位址 | bytes | 作用 | 等級 |
|---|---|---|---|
| overlay-10 `1386h..1392h` | `C4 06 F4 5C`… | 從 `DS:5CF4h` 串列頭開始，計數 `[bp-5] = 0` | exact |
| overlay-10 `13A1h..13ACh` | `9A 43 00 0A 01`、`FE 46 FB` | 每一筆**先**呼叫 entry 7 再加計數；呼叫前沒有任何條件跳躍 | exact |
| overlay-10 `13D7h..13EFh` | `26 3B 85 7C 06`、`26 C6 45 13 01` | 計數大於隊伍人數（`[4937h] + 67Ch`）才把 runtime `+13h` 設 1——串列前段是隊伍，NPC 算在隊伍人數裡（spec 091 `121Ch` 同一格） | exact |
| overlay-10 `1487h..1497h` | `26 C4 85 04 01` | 沿 `+104h` 走下一筆 | exact |
| overlay-25 `0BBEh..1031h` 與它呼叫的 `0000h`、`01F8h`、`0281h`、`039Fh`、`10E3h`、`1173h`、`12AEh`、`1366h`、`13F8h` | — | 條件跳躍只看物品欄位、`+102h`、`DS:47A8h`、`+98h`、`+2Eh`、`+0AAh`；沒有一處讀 `+10Dh`／`+10Eh`／`+10Fh`／runtime `+13h` 或任何「是不是 NPC」的旗標（整顆 overlay-25 唯一讀 `+10Dh` 的 `0B12h` 不在這條呼叫鏈上） | exact |
| overlay-03 `2F46h..2F4Eh` | `FF 36 F2 5C`、`9A 43 00 0A 01` | ADD NPC 加入時也跑一次 entry 7（spec 091） | exact |
| overlay-17 `1244h` | `E8 49 FC` | ADD NPC 的載入呼叫 `0E90h`，與 LOAD MONSTER 同一支，物品串列是同一個 block 的 MONnITM（spec 142） | exact |

far call `010Ah:0043h` 是 overlay-25 entry 7（entry 1 的 stub 在 `010Ah:0025h`，spec 142；stub 5 bytes 一格）。
輸入：overlay-10 SHA-256 `b929c7040aaa399ba803911c8630e06a8e21b9e7d64a67def69c700d7fe14439`、
overlay-03 `5a3a18bd…6ea68f`、overlay-17 `f92fed1b…f7d1e4`、overlay-25 同上，均對 `docs/audit/dos-ovr-manifest.json`。

remake：

- ADD NPC（`applyAddNPC`）把 MONnITM 同一個 block 的物品放進 `member.Inventory`，順序照檔案
  （`npcItems`）。
- 開打時 `applyNPCCombatStats` 先照記錄填（HP、士氣、豁免那幾格仍讀記錄），再由
  `applyNPCGearStats` 從記錄與 `member.Inventory` 跑 `RecomputeMonsterCombatFields`，蓋掉
  THAC0、AC、腳程、兩種形態的傷害骰與射程——與怪物共用 `applyRecordGearStats`／`recomputeFromBase`。
- 戰鬥中換裝（overlay-19 `1469h..1485h`）對 NPC 一樣重算。

資料面（ADD NPC 的八個呼叫點，`RecomputeMonsterCombatFields` 逐筆）：十四筆裡十二筆帶物品（MAD MAN、SKULLCRUSHER 沒有）；
重算改變 THAC0／AC／腳程／骰子的有 DIRTEN、ACOLYTE、WARRIOR、SWORDSMAN、ROBBER、CURATE、HERO、
PRINCESS FATIMA 等。例：HERO（MON3CHA 6Dh）樣板 THAC0 43（`+2Dh` 42 加力量）、AC 53、1d8、腳程 12；
開打時長劍 +1、盾、帶甲 +1 → THAC0 44、AC 61、1d10+2、腳程 9。

驗證：`TestAddNPCLoadsTheMonsterItemChain`（ECL3/b0 `A046h` 的 `ADD NPC 6Bh` 實跑，DIRTEN 的三件
逐位元組對 MON3ITM）、`TestNPCGearIsRecomputedWhenTheBattleStarts`（獸人家同一場，HERO 上盤面的
THAC0／AC／腳程／骰子／射程，負對照是只讀記錄的舊做法；戰鬥中改拿短弓，射程變成弓的）。

仍未閉合：

| 項目 | 等級 | 為什麼 |
|---|---|---|
| NPC 的執行期收據 | unknown | `workplace/` 沒有 dosgolem 讀出的隊伍 NPC 記錄；要駕駛原版雇一名傭兵再開打才拿得到 |
| ADD NPC 時 entry 7／entry 2 寫回記錄 | exact（`2F46h`／`2F53h`） | remake 的 `member.Record` 仍是樣板；`partyStrengthRecord` 讀 NPC 的 `+110h`／`+111h`，拿到的是樣板殘值（HERO 的 `+110h` 是 151）。戰場不受影響（開打時重算），影響的是遭遇強度 |
| #97 之前加入的 NPC | — | 舊存檔裡的 NPC 物品欄是空的；讀檔不補（補的條件分不出「本來沒有」與「交出去了」，要 schema 版本才能判） |
| entry 7 尾段 `1000h..102Eh` | exact（bytes） | `+98h > 0` 而且 `+2Eh > 0` 時 `+6Bh = +98h`，否則 `+6Bh = 1`；`+6Bh` 的讀取端沒追，remake 不寫 |

## 還沒接（DRAFT）

| 項目 | 等級 | 為什麼 |
|---|---|---|
| 彈藥消耗與「沒有箭不能射」 | exact | 已由 spec 151 接上（overlay-13 `1883h`／`2A7Bh`、overlay-25 entry 45、overlay-09 entry 9）|
| 戰鬥中怪物換裝 | exact（`146Fh` 重算）| 怪物不走物品選單；AI 用物品（spec 096 entry 3）不改穿戴 |
