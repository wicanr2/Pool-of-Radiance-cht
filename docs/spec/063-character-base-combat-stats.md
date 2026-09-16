# Spec 063：角色的基礎 AC、THAC0、移動與武器攻擊數值

狀態：READY（建角寫下的基礎值、`DS:3C16h` 職業／等級 THAC0 表、選取規則、
武器把 `+110h`／`+115h`／`+117h`／`+119h` 算出來的流程、物品型別表的四個欄位，
以及 entry 7 整份重算的九個輸出欄位）；CONFORMED（怪物開打時的 THAC0：`+110h = +2Dh`
加沒有武器時的力量修正，dosgolem 獸人家二十隻逐隻相同；`+0AAh` 在沒有武器那條路上
由 `12AEh` 自己查，exact）；DRAFT（物品型別表其餘欄位）。日期：2026-09-05；
2026-09-16 補怪物那一側（#31／#36）。

## 為什麼需要這一段

spec 049 閉合了 285-byte record 裡 AC（`+111h`）、THAC0（`+110h`）與傷害骰
（`+115h`／`+117h`／`+119h`）**怎麼讀**，但沒有回答**誰寫**。怪物那側也要回答：
`MON*CHA` block 裡的 `+110h` 是殘值，開打時整份重算會把它蓋掉（見〈怪物開打時
也跑同一支〉）。玩家這側要靠規則算出這些欄位；在它閉合之前，remake 的隊伍只能用
placeholder，勝負數字不作數。

## 證據

| 檔案 | 內容 | 輸入 SHA-256 |
|---|---|---|
| `docs/audit/ida-overlay16-character-record-init.json` | overlay-16 `0570h..0620h`（建角配置 285 bytes 後寫下的基礎值） | `d1b05743…b7da`（overlay-16.bin） |
| `docs/audit/ida-overlay16-class-thac0-selection.json` | overlay-16 `0C80h..0D60h`（依職業等級選 THAC0） | 同上 |
| `docs/audit/ida-overlay16-base-movement.json` | overlay-16 `1BE0h..1C40h`（`+72h` 基礎移動） | 同上 |
| `docs/audit/ida-ds-class-thac0-table.json` | `DS:3C16h` 起 88 bytes | `12811cbc…10d9f`（START.EXE） |
| `docs/audit/ida-overlay25-weapon-attack-stats.json` | overlay-25 `0000h..0210h`（武器決定 THAC0 與傷害） | `9fede24b…50c0e` |
| `docs/audit/ida-overlay25-combat-stat-recalc.json` | overlay-25 `0E30h..1010h`（整份戰鬥數值重算） | 同上 |

位址一律是 overlay-local file offset（base 0）；`DS:` 是共用資料段偏移。

## 建角寫下的基礎值（exact）

overlay-16 `0570h` 以 `GetMem(11Dh)` 配置 285 bytes——與 spec 049 的 unified
record 同一個大小——接著寫下：

| record offset | 值 | 語意 |
|---|---|---|
| `+0A9h` | `32h` | 基礎 AC internal，typed `60-50 = 10` |
| `+2Dh` | `28h` | 基礎 THAC0 internal，typed `60-40 = 20` |
| `+10Ch` | 0 | 狀態 |
| `+10Dh` | 1 | 參戰旗標。spec 052 的先攻與 spec 059 的反應攻擊都以它非 0 為前提，這裡是它的產生端 |
| `+0BBh`／`+0BCh`／`+6Ch` | 1 | 尚未命名 |
| `+0ABh` | `random(100h)` | 尚未命名 |

`+72h`（基礎移動）在同一支建角流程的 `1C0Bh` 寫 `0Ch`，即 12。overlay-12 的
`0B96h` 對召喚生物寫同一個常數，兩處獨立一致。

## 職業／等級 THAC0 表（exact）

overlay-16 `0CC6h..0D4Fh`：

```
record[+2Dh] = 0
for class = 0..7:                          ; 迴圈變數與 +96h 的 8 byte 陣列同索引
    level = record[+96h + class]
    if level > 0:
        value = DS:3C16h[class*0Bh + level]
        if value > record[+2Dh]:
            record[+2Dh] = value
    record[+0B0h] += <overlay-local table>[class]
```

表折疊成 1-based：每列 11 bytes，等級 1..10 用到 index 1..10，index 0 空著。
internal 值越大 typed THAC0 越小，所以「取最大」就是**取多職業裡最好的那個**。

`DS:3C16h` 起 88 bytes 逐位元組如下（括號內為 typed 值，等級 1..10）：

| class | bytes | typed |
|---|---|---|
| 0 | `28 28 28 28 2A 2A 2A 2C 2C 2C 2E` | 20 20 20 18 18 18 16 16 16 14 |
| 1 | 同 class 0 | 同 |
| 2 | `28 28 29 2A 2B 2C 2D 2E 2F 30 31` | 20 19 18 17 16 15 14 13 12 11 |
| 3 | 同 class 2 | 同 |
| 4 | 同 class 2 | 同 |
| 5 | `28 28 28 28 28 28 29 29 29 29 29` | 20 20 20 20 20 19 19 19 19 19 |
| 6 | `28 28 28 28 28 29 29 29 29 2C 2C` | 20 20 20 20 19 19 19 19 16 16 |
| 7 | 同 class 0 | 同 |

三組相同的列與 `internal/creation/catalog.go` 既有的職業碼吻合：0 牧師、2 戰士、
5 法師、6 賊。1／3／4／7 三列與其相同的那一組共用進程，但**哪一個索引對應哪個
職業名稱尚未由原版證據命名**，不得在程式碼裡替它們取名。

## 武器決定 THAC0 與傷害（exact）

overlay-25 entry 1（`0000h`，`retf 4`），參數是角色 record：

1. `weapon = record[+0CCh]`（far ptr）。為 NULL 直接返回——**沒有武器時
   `+115h`／`+117h`／`+119h` 維持原值**，這支不會補上徒手傷害。
2. `type = weapon[+2Eh]`；物品型別表 entry 為 `type × 16`。
3. `record[+110h] = record[+2Dh]`（基礎 THAC0）。
4. 表的旗標欄 bit 1 成立 → `+110h += sub_1173(record)`。
5. 旗標 bit 2 成立 → `+110h += sub_12AE(record)`、`+119h += sub_1366(record)`。
6. `record[+119h] = 表的傷害加值欄`。
7. `modifier = weapon[+32h]`；旗標 bit 7 且 `record[+0FCh]` 非 NULL 時再加該物品的
   `+32h`；旗標 bit 0 且 `record[+0F8h]` 非 NULL 時同樣再加。
8. `record[+119h] += modifier`。
9. `record[+2Eh] == 2` 且武器 type 落在 `24h`、`25h`、`29h..2Ch` 時 `modifier += 1`。
10. `record[+110h] += modifier`。
11. `record[+115h] = 表的骰數欄`、`record[+117h] = 表的骰面欄`。

物品型別表在 `DS:54E0h`，每筆 16 bytes，本規格用到四個欄位：

| entry offset | DS 位址（type 0） | 語意 |
|---|---|---|
| `+0` | `54E0h` | 類別（overlay-25 `01F8h` 以它是否為 2 分支） |
| `+9` | `54E9h` | 傷害骰數 |
| `+0Ah` | `54EAh` | 傷害骰面 |
| `+0Bh` | `54EBh` | 傷害加值 |
| `+0Eh` | `54EEh` | 旗標（bit 0／1／2／7 已用到） |

## 整份重算：overlay-25 entry 7（`0BBEh`，`0E36h` 是它的後半）

overlay-25 entry 7（`0BBEh`，stub `010Ah:0043h`）是「重算這個角色的戰鬥數值」。
原版每次裝備變動、載入角色與升級之後都跑它，順序是：

1. 清十二個裝備槽指標（`+0CCh..`），沿 `+0C8h` 的物品串列把備妥的認回槽位，
   `+0A2h/+0A4h/+0A6h` 抄到 `+114h..`（spec 051）。**`+2Dh` 在這一支裡只讀不寫**
   （整顆 overlay-25 沒有 `3C16h` 職業表的引用）；它是建角／訓練所（overlay-16
   `0CC6h`）寫下、跟著記錄走的值。remake 的 `RecomputeCombatFields` 順手從職業等級
   重算它，七名預設人物的值相同，但那不是 entry 7 做的事。
2. 三個基礎值搬進來（`0E43h..0E6Ch`）：`+111h = +0A9h`、`+11Ch = +72h`、`+110h = +2Dh`。
3. 有備妥武器（`+0CCh` 非 NULL，`0E7Eh`）→ 呼叫上一節那支（`+110h`／`+115h`／
   `+117h`／`+119h`）。沒有 → `+110h += 12AEh`（力量的命中修正）、`+119h += 1366h`
   （傷害修正）。`12AEh` 開頭自己查 `+0AAh`（`12C8h`：0 就回 0），所以「沒有武器
   時也由 `+0AAh` 決定要不要補」是 exact，不再是強推論。
4. 沿 `+0C8h` 的物品串列累積負重（`039Fh`，spec 079）與護甲五個槽
   （`0281h`，spec 080），寫回 `+102h`、`+111h`、`+112h`、`+11Ch`。

spec 079 與 spec 080 之後分別把第 4 步的兩支助手讀完，所以整條管線現在是閉合的。

**驗證**：把七名預設人物記錄裡的 `+2Dh`、`+110h`、`+111h`、`+112h`、`+115h`、
`+117h`、`+119h`、`+11Ch`、`+102h` 全部清成 0 再重算，九個欄位逐一回到原值
（`TestRecomputeCombatFieldsMatchesThePremadeCharacters`）。實作在
`internal/gamepack/record_recompute.go`。

## 怪物開打時也跑同一支：樣板的 `+110h` 是殘值（exact）

overlay-10 開打初始化（`1ED6h`，`docs/audit/ida-overlay10-phase-init.json`）在部署
`1A99h` 之前先叫 `1380h`（`1F9Ch`）：沿 `DS:5CF4h` 的 combatant 串列，對**每一筆**
呼叫 overlay-25 entry 7（`13A7h`：`9A 43 00 0A 01`），再配一個 16h bytes 的執行期
子結構掛到 `+108h`。所以怪物的 `+110h` 開打時一律被 `+2Dh`（＋沒武器時的力量修正）
蓋掉，`MONnCHA` 樣板裡的 `+110h` 從來不進命中判定。

三條證據：

- 靜態：`1380h` 的迴圈與 entry 7 的 `0E65h`（`+110h = +2Dh`）。
- 樣板：`docs/audit/monster-thac0-template-scan.json`（`cmd/pool-monster-thac0-scan`）172 筆裡
  `+110h ≠ +2Dh` 的 53 筆、bit 7 亮的 43 筆，其中 42 筆恰好是 `+2Dh + 109`——同一個常數
  差，是樣板產生時留下的殘值形狀，不是編碼（另一筆 LEVEL 6 MU 差 192；沒亮 bit 7 的
  十筆差 −5..+45）。諾里斯的 154 ＝ 45 ＋ 109。
- 執行期：dosgolem 在獸人家開打那一幀讀 `6517h` 指到的二十五份記錄
  （`docs/audit/dosgolem-monster-thac0-runtime.json`）：獸人樣板 `+110h = 40`，執行期
  41 ＝ `+2Dh`；獸人頭目 44 ＝ `+2Dh`；隊員 `+110h` ＝ 40 ＋ 力量修正（力量 18 的 41／42）。
  `+2Dh` 本身開打時沒被重算——獸人的 41 不是一級戰士表的 40。

remake：`MonsterRecord.CombatThac0Internal()` ＝ `+2Dh` ＋（`+0AAh` 開著時的力量修正），
兩處排怪都改用它；沒有怪物的物品鏈，一律走沒有武器那條。諾里斯 154 → 46（表面 14，
力量修正 +1）；`TestNorrisTHAC0ReadsTheBaseFieldNotTheTemplate`、
`TestMonsterTHAC0MatchesTheOrcHomeRuntimeReceipt`（二十隻逐隻相同）。

仍未閉合的點：

- **`+115h`／`+117h` 在沒有武器時從哪來。** TARRY（`chrdatd5`）身上沒有備妥
  武器，記錄裡卻存著 1／2（1d2）。`0E36h` 那條路不寫這兩格，所以那個值是
  別處寫的，還沒讀出來。

## READY 契約

1. 建角寫下的三個基礎值是 `+0A9h = 32h`、`+2Dh = 28h`、`+72h = 0Ch`，
   不得改用現代規則推導。
2. THAC0 表逐位元組照抄 `DS:3C16h` 的 88 bytes，8 列 × 11，等級 1-based。
3. 多職業取 internal 最大者，即 typed 最小者；等級為 0 的職業不參與。
4. 職業索引 1／3／4／7 在原版證據命名之前不得取名。
5. 沒有武器時不得自行補傷害骰——原版那支直接返回。
6. 穿裝備後的 AC 依 spec 080 的五個累加器算；負重與移動力依 spec 079。
   `0E36h` 的九個輸出欄位以七名預設人物逐格對得上為驗收條件。
7. 沒有備妥武器時只補兩個力量修正，不得順手填 `+115h`／`+117h`。
