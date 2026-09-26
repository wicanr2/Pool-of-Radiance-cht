# Spec 149：探索中的物品頁與穿戴效果——Ready 不換手、Use／Drop／Halve／Join、`+3Eh` 效果碼

狀態：READY（選單組成、Ready 擋下同類、`+3Eh` 大於 7Fh 的穿戴效果怎麼掛與摘、
"already using" 前面那一次呼叫、戰鬥外的 Use 與 `+07h` 為 0 的 "Use it?"，都從
overlay-19／22／24／25／12 讀出並實作；`cmd/pool-game/item_page_test.go` 從 `Update()`
送鍵驗，穿戴效果拿原版預設人物的三件真實物品驗）；DRAFT（見〈未閉合〉）。
日期：2026-09-26。主台帳：GitHub issue #91（接 spec 144〈探索中（非戰鬥）的差異〉）。

## 一句話

探索與營地中的物品頁就是戰鬥中那一支 overlay-19 entry 6，只差在 `DS:4954h`。Ready
那一格有東西就擋下、印 "already using" 加那一件的名字，**不換手**；`+3Eh` 大於 7Fh
的物品戴上與拿下各把 `+3Eh` 當效果碼交給 overlay-24 entry 1，由 overlay-12 的
80h..8Bh 常式掛或摘效果節點（火焰抗性戒指掛 3Dh、位移斗篷掛 59h、食人魔之力手套以
26h 把力量調到 18/00）。戰鬥外用物品不離開選單；參數表 `+07h` 為 0 的物品法術問
"Use it?"，答 Y 照樣記帳但不放出去。

## 輸入

- `overlay-19.bin` SHA-256 `4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9`
- `overlay-12.bin` `d1b057432515b7daa2b83199d2588c131a120037d30117e664b01e4c6f2bcb7e`
- `overlay-22.bin` `967065cc35975465a7250026c63b8a5ae06b812b228abcfbbbd83d636538dda8`
- `overlay-24.bin` `e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714`
- `overlay-25.bin` `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e`
- 原版預設人物 `chrdatd1`／`d3`／`d6`（`.sav`／`.itm`／`.spc`，`Pool of Radiance (1988).zip`）
- 工具：`coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel
  --start-address=…`；位址是 overlay-local offset。far call `9A off seg` 以
  `seg × 16 + 3B0h + off` 對 `docs/audit/dos-ovr-manifest.json` 的 stub
  `executable_file_offset` 反查（spec 109）：`0100:0025` → `13D5h` → overlay-24 entry 1；
  `010A:0025` → `1475h` → overlay-25 entry 1（`0441h`）；`010A:007F` → overlay-25 entry 19。

## 選單在戰鬥外的組成（exact）

表與按鍵分派見 spec 144〈選單怎麼組、按鍵怎麼分〉。戰鬥外的差異：

| 位址 | 位元組 | 內容 |
|---|---|---|
| `0FA3h..0FBDh` | `80 3E 54 49 02 74 24`／`…03 74 1D`／`…04 74 16`／`…05 75 37` | " Use"：`DS:4954h` 是 2、3、4 才接（5 另看 runtime `+2`），其餘不接 |
| `0FF6h..1017h` | `26 80 BD 84 00 80 72 16` … `80 3E 54 49 05 74 28` | " Trade"：不在戰鬥中，而且 `+84h` < 80h 或 `+10Dh` 為 0 或 `+10Ch` == 1 |
| `10EAh`／`1119h` | `80 3E 54 49 01` | " Sell"／" Id"：商店中（spec 067）|
| `130Ah..1314h` | `80 3E 54 49 05 74 07 … 26 C6 05 00` | 不在戰鬥中時 Use 之後結果清 0：用完物品留在選單 |

`DS:4954h` 的寫入端（`C6 06 54 49 imm8` 全 overlay 掃過）：overlay-15 `1E51h` = 2（紮營，
spec 135）、overlay-16 `0167h` = 0（隊伍選單）、overlay-03 `1989h`／`3660h`／`379Eh` = 4 與
`1990h`／`3680h`／`369Dh`／`36DBh`／`37D4h` = 3、overlay-25 `2B98h` = 3、overlay-08 `0077h`
= 5（戰鬥）、overlay-04 `0CEAh`／overlay-06 `0531h` = 1、overlay-05 `14E0h` = 6、overlay-18
`02C8h` = 7。冒險中的值是 3 或 4（strong inference：寫入端在 overlay-03，兩個值都在 Use 的
集合裡，所以不影響選單）。

remake：冒險畫面的 `I` 與紮營開的物品頁接 Use；隊伍選單的 `V` 開的不接
（`itemPageState.creationMenu`）。Trade 沒接（〈未閉合〉）。

## Ready：擋下同類、不換手（exact）

規則全文在 spec 144〈Ready：overlay-19 entry 7〉。那一格（`+CCh + 類別 × 4`、戒指
`+F4h`、箭 `+F8h`、弩矢 `+FCh`）已有東西時 `[bp-2] = 2`，`168Bh` 走 "already using"
那一支，`+34h` 不動——**不會把舊的那一件卸下**。remake 以前的探索頁會自動換掉同類，
現在與戰鬥中共用 `readyMemberItem`（`cmd/pool-game/item_page.go`）。

### "already using" 前面那一次呼叫（exact）

```
1690..16B8  push 角色, push +CCh + 類別×4 那一件, push 0, 0, 0, 0
16B9        9A 25 00 0A 01      overlay-25 entry 1（0441h），retf 10h
16BE..16C9  "already using "（14A6h）載入暫存字串
16CE..16E2  串上那一件的 +0（Pascal 字串，就是名稱）
16E7        9A 7F 00 0A 01      overlay-25 entry 19 印出
```

overlay-25 entry 1 是**物品名稱重組**：`044Bh` `26 C6 05 00` 先把物品 `+0` 清空，
`044Fh` 第三個旗標非 0 才在前面加 " Yes "／" No  "（`042Ch`／`0433h`），`0735h` 最後一個
旗標非 0 才以 `0198:002F` 印到畫面。四個旗標都是 0，所以它**什麼也不印**，只把擋住的
那一件的名字重組好給下一步串。原版的訊息就是 "already using " + 名稱，沒有另外的前綴。
remake 的 `Item.Name` 在拿到、鑑定、抹卷軸時已經照同一支重組過，直接串。

## 穿戴效果：overlay-19 → overlay-24 entry 1 → overlay-12（exact）

```
14D4  26 80 7D 3E 7F 77 04       +3Eh > 7Fh → [bp-1] = 1
151A  26 C6 45 34 00             卸下：+34h = 0
1522  80 7E FF 00 74 1C          [bp-1] 為 0 就不派發
1528..153E  push +3Eh, 角色（DS:5CF0h）, 物品, B0 01 50（模式 1）
153F  9A 25 00 00 01             overlay-24 entry 1
1642  26 C6 45 34 01             裝上：+34h = 1
1650..1666  push +3Eh, 角色, 物品, B0 00 50（模式 0）
1667  9A 25 00 00 01             overlay-24 entry 1
```

overlay-24 entry 1（`0000h`，`retf 0Ch`）把（角色, 節點, 模式）原樣推給
`call dword ptr [di+6786h]`（`FF 9D 86 67`，`di` = 代碼 × 4），spec 112 的派發表。這裡的
「節點」是**物品記錄**。80h..8Bh 對到的常式（`docs/audit/dos-effect-handlers.json`）：

| 碼 | overlay-12 | 模式 0（戴上）| 模式 1（拿下）|
|---|---|---|---|
| 80h 81h 82h 85h 86h 88h 8Ah 8Bh | entry 121 `2EECh` | `2F35h` overlay-24 entry 10 掛（碼 = 物品 `+3Dh`、持續 0、等級 `0Ch`、不收尾）；`2F51h` 再以 entry 1（碼 = `+3Dh`、節點 NULL、模式 0）派發一次 | `2F17h` overlay-24 entry 2（碼 = `+3Dh`、節點 NULL）|
| 83h | entry 122 `2F68h` | `3032h` overlay-24 entry 18（`1158h`）以 18/00（`B0 12`、`B0 64`）調力量；調上去了印 "is stronger"（`2F5Ch`，戰鬥中把 `4954h` 暫設 2 再印）；`308Eh` `677Fh = 1`；`30A6h` entry 10 掛 26h（持續 0、等級 = entry 18 回傳的快照、要收尾）| 見下 |
| 84h | entry 123 `30B1h` | 物品 `+3Dh & 0Fh` ≠ 角色 `+0A0h`（陣營）→ `+34h = 0`、`6777h = 8`、以 overlay-24 entry 19（`133Ah`）受 `+3Dh ÷ 16` 點傷害 | 什麼也不做 |
| 87h | entry 124 `3141h` | 角色 `+10h`（力量）< 13h → `+34h = 0`，印 "Must have Giant Strength"（`3128h`）| 什麼也不做 |
| 89h | entry 125 `3186h` | 什麼也不做 | `319Eh` overlay-24 entry 2（碼 17h、節點 NULL）|

overlay-24 entry 2（`0028h`）節點為 NULL 時沿 `+7Fh` 找**第一個**同碼的節點
（`0050h..0072h`），`+4` 非 0 就先以模式 1 叫那個碼的常式（`0089h..009Dh`，力量的還原
在這裡），再摘掉、`FreeMem(9)`（`011Eh` `B8 09 00`）。

### 83h 拿下時摘哪一個（exact，含一處照位元組的怪行為）

```
2F8D..2FA4  [bp-4] = (+10h == 12h 而且 +16h == 64h)      現在正好 18/00？
2FA8..301C  沿 +7Fh 走，找到一個就停（[bp-5]）：
2FB9          節點 +0 == 26h 才看
2FCF          overlay-24 entry 17（1107h）解快照 → (力量, 百分位)
2FD4..2FE1    [bp-4] 非 0 而且節點 +0 < 80h → 摘
2FE3..2FF3    否則快照解回 18/00 而且 [bp-4] 為 0 → 摘
2FF5..3004    overlay-24 entry 2（角色, 26h, 這個節點）
```

`2FDDh` 比的是節點 `+0`（代碼，恆為 26h）而不是 `+3` 的 inactive 位元，所以條件恆成立：
**現在正好 18/00 就摘第一個 26h**，不論它是不是這雙手套掛的。remake 照位元組。
entry 17 的解碼：`1118h` `24 7F` 去掉 inactive 位元，`1123h` `26 80 3D 65`，≤ 65h 是 18/(值 − 1)，
否則值 − 100（與 `gamepack.decodeStrengthSnapshot` 相同）。

### 立即派發（strong inference）

entry 121 掛完節點後以節點 NULL、模式 0 再派發一次 `+3Dh`。原版預設人物身上的三個碼
3Dh（entry 55）、59h（entry 83）、03h（entry 7）在盤點裡只寫 `6774h`／`6776h`／`6780h`／
`6784h`／`6786h`——豁免骰、傷害、命中骰與目前目標，都是戰鬥中每一次判定前重設的暫存。
所以戰鬥外這一下沒有留下狀態，remake 不做。其餘 `+3Dh` 值還沒看到出現在物品上。

### 與原版存檔對得上

- HAPLO（`chrdatd1`）戴火焰抗性戒指（`+3Eh` 81h、`+3Dh` 3Dh），`.spc` 是
  `3D 00 00 0C 00 …`：碼 3Dh、持續 0、等級 0Ch、不收尾——與 entry 121 掛的節點逐位元組相同。
- ALFRED（`chrdatd3`）戴手套（83h）與戒指，力量 18/100，`.spc` 的 26h 節點是
  `26 00 00 01 01`：快照 01h 解回 18/0、要收尾。拿下手套回到 18/0、再戴上掛回同一個節點。
- 位移斗篷（85h、`+3Dh` 59h）：CARRY（`chrdatd6`）與 `chrdate6` 的 `.spc` 有 59h（spec 069）；TARRY
  （`chrdatd5`）戴著斗篷卻只有 26h——預設人物檔本身不一致，不影響規則。

remake：`gamepack.ApplyWearEffect`（`internal/gamepack/wear_effect.go`）；探索中與戰鬥中
的 R 共用 `(*app).wearItem`，戰鬥中寫這一格的戰鬥串列，存檔那一份跟著寫；摘掉的節點經
`expiredEffectTeardown`（力量還原）。

## 戰鬥外的 Use（exact，挑對象是 strong inference）

```
12B4  26 80 7D 34 00 75 1A        沒穿戴 → "Must be Readied"（0EC6h）
12DE  9A 3E 00 E2 00              overlay-22 entry 6：卷軸？
12EA  26 80 7D 3D 00 76 34        否則 +3Dh 為 0 → 什麼也不做
12F4  26 80 7D 3E 80 73 2A        +3Eh ≥ 80h（穿戴效果物品）→ 什麼也不做
1307  E8 7C 07                    entry 8（1A86h）
```

entry 8 在戰鬥外（`1B94h..1BB3h`）印 "uses an item" 與物品名，再以 overlay-22 entry 5
放出去。entry 5 戰鬥外：

```
0C3F  80 BD 9B 31 00 74 03       參數表 +07h 非 0 → 0D23h 照常瞄準、放出
0C49  80 3E B3 6C 00 75 69       物品（DS:6CB3h 非 0）→ 0CB9h
0CC7／0CE4／0CF8                 "That Item"／"is a combat-only item..."／"Use it? "
0D0B  9A 3E 00 1D 01 / 3C 59     Y → 0D17h 26 C6 05 01（結果 = 1）
0D1B  C6 46 0A 00                不放出去
```

結果非 0 就在 entry 8 `1C0Eh..1C7Bh` 記帳：卷軸以 overlay-22 entry 7 抹掉那一行；其餘
數量 `+39h` 大於 1 減數量，否則減充能 `+3Ch`，減到 0 以 overlay-25 entry 17 拿掉
（`gamepack.SpendAIItemUse`）。

`+07h` 非 0 的那一支之後照常經 `DS:6A78h` 瞄準。remake 戰鬥外沒有戰術格，瞄準改成與
`C)AST` 戰鬥外同一種挑對象（spec 119 的 `resolveFieldCast` 那幾種效果），這是 remake 的
呈現；「兩者經同一支 entry 5」是 strong inference。放棄挑對象（ESC）時瞄準回 0，
戰鬥外沒有 entry 34（`1BE2h` 只在戰鬥中），不記帳。施法者等級照 spec 098〈施法者等級〉：
從物品放的一律 6 級、物品效果 12 級。

實作：`useItemOnPage`、`beginItemPageSpell`、`itemPageCombatOnlyInput`、`itemPageTargetInput`、
`spendItemPageUse`（`cmd/pool-game/item_page.go`）。

## Drop、Halve、Join（exact）

與戰鬥中同一套（spec 144 的三節），戰鬥外一樣在選單裡做完、回到選單。`halveMemberItem`／
`joinMemberItems` 兩邊共用。

## 測試

`cmd/pool-game/item_page_test.go`，全部從 `Update()` 送鍵（冒險畫面按 `I` 開頁）：

- `TestItemPageFooterFollowsTheMenuContext`：冒險中 `READY USE DROP HALVE JOIN EXIT`，
  隊伍選單開的沒有 USE、U 不做事。
- `TestItemPageReadyRefusesAnOccupiedSlot`：第二把武器印 "ALREADY USING SWORD"、旗標不動；
  先卸下才換得上。
- `TestGauntletsOfOgrePowerWearEffect`（ALFRED 真檔）、`TestRingAndCloakWearEffects`
  （HAPLO 的戒指、`chrdatd6` 的斗篷）、`TestCombatReadyAttachesTheWearEffect`（墓園雙手劍
  88h，戰鬥中）、`TestGiantStrengthItemRefusesAWeakWearer`（87h）。
- `TestItemPageUsesAHealingPotion`、`TestItemPageCombatOnlyWandAsks`、
  `TestItemPageReadsAClericScroll`、`TestItemPageDropHalveJoin`。
- `internal/gamepack/wear_effect_test.go`：7Fh 門檻、83h 拿下時摘自己那一個、89h。

## 未閉合

- **Trade**（`0FF6h` 的條件、`173Bh`）：印 "Trade with Whom?"（`171Fh`）以 overlay-25
  entry 42 挑人，`274Fh` 的負重檢查不過印 "Overloaded"（`1730h`），過了以 overlay-25
  entry 18 給對方、entry 17 從自己拿掉、entry 7 重算對方。`274Fh` 沒讀，remake 不接，
  選項列也不列。
- **84h**（陣營限制、受傷）：傷害那一段 overlay-24 entry 19（`133Ah`）的訊息與 HP 寫入
  沒讀完，`ApplyWearEffect` 對 84h 回 `Known = false`、什麼也不做。原版預設人物與寶物裡
  還沒看到 84h 的物品。
- 戰鬥外 entry 8 那一句 "uses an item" 加物品名（`1B94h..1BB8h`）remake 沒印，直接進挑對象。
- " Use" 另要角色 `+10Dh` 非 0（`0F8Fh`）；隊員的 `+10Dh` 在建角時是 1（spec 063），
  死亡、離隊時的值沒讀，remake 不看。
- 物品頁的版面（清單、選項列）是 remake 的呈現，原版的版面沒量。
