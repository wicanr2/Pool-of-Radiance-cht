# Spec 063：角色的基礎 AC、THAC0、移動與武器攻擊數值

狀態：READY（建角寫下的基礎值、`DS:3C16h` 職業／等級 THAC0 表、選取規則、
武器把 `+110h`／`+115h`／`+117h`／`+119h` 算出來的流程、物品型別表的四個欄位，
以及 `0E36h` 整份重算的九個輸出欄位）；DRAFT（物品型別表其餘欄位、
`+0AAh` 在「沒有武器」那條路上的角色）。日期：2026-09-05。

## 為什麼需要這一段

spec 049 閉合了 285-byte record 裡 AC（`+111h`）、THAC0（`+110h`）與傷害骰
（`+115h`／`+117h`／`+119h`）**怎麼讀**，但沒有回答**誰寫**。怪物那側不需要回答
——數值直接躺在 `MON*CHA` block 裡；玩家這側需要，因為建角產生的角色記錄要靠
規則算出這些欄位。在它閉合之前，remake 的隊伍只能用 placeholder，勝負數字不作數。

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

## 整份重算：`0E36h`

overlay-25 `0E36h` 是「重算這個角色的戰鬥數值」。原版每次裝備變動、載入角色與
升級之後都跑它，順序是：

1. `+2Dh` ← 職業等級表（上一節的 `0CC6h`）。
2. 三個基礎值搬進來：`+111h = +0A9h`、`+11Ch = +72h`、`+110h = +2Dh`。
3. 有備妥武器 → 呼叫上一節那支（`+110h`／`+115h`／`+117h`／`+119h`）。
   沒有 → `+110h` 加命中的力量修正、`+119h` 加傷害的力量修正。
4. 沿 `+0C8h` 的物品串列累積負重（`039Fh`，spec 079）與護甲五個槽
   （`0281h`，spec 080），寫回 `+102h`、`+111h`、`+112h`、`+11Ch`。

spec 079 與 spec 080 之後分別把第 4 步的兩支助手讀完，所以整條管線現在是閉合的。

**驗證**：把七名預設人物記錄裡的 `+2Dh`、`+110h`、`+111h`、`+112h`、`+115h`、
`+117h`、`+119h`、`+11Ch`、`+102h` 全部清成 0 再重算，九個欄位逐一回到原值
（`TestRecomputeCombatFieldsMatchesThePremadeCharacters`）。實作在
`internal/gamepack/record_recompute.go`。

兩個仍未閉合的點：

- **沒有武器那條路上的 `+0AAh`。** 有武器時它決定要不要套用兩個力量修正；
  這裡假設同一個開關也管沒有武器的情形，屬**強推論**。七名預設人物的
  `+0AAh` 都是 1，樣本分不出兩種讀法。
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
