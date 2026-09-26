# Spec 149：探索中的物品頁與穿戴效果——Ready 不換手、Use／Drop／Halve／Join、`+3Eh` 效果碼

狀態：READY（選單組成、Ready 擋下同類、`+3Eh` 大於 7Fh 的穿戴效果怎麼掛與摘、
"already using" 前面那一次呼叫、戰鬥外的 Use 與 `+07h` 為 0 的 "Use it?"，都從
overlay-19／22／24／25／12 讀出並實作；`cmd/pool-game/item_page_test.go` 從 `Update()`
送鍵驗，穿戴效果拿原版預設人物的三件真實物品驗）；#105 補上 Trade 與負重檢查、84h
的陣營限制與受傷、戰鬥外 Use 的 "uses an item" 那一拍與 " Use" 的兩道門
（`item_page_trade.go`／`item_page_trade_test.go`）；DRAFT（見〈未閉合〉）。
日期：2026-09-26。主台帳：GitHub issue #91、#105（接 spec 144〈探索中（非戰鬥）的差異〉）。

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
| `0F8Ch..0FA1h` | `26 80 BD 0D 01 00 74 5F`、`C4 3E 33 49 26 83 BD CA 01 00 75 53` | " Use"：記錄 `+10Dh` 為 0 或 `[4933h]+1CAh` 非 0 就不接（見〈Use 的兩道門〉）|
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
（`itemPageState.creationMenu`）。Trade 不看 `DS:4954h` 的 2／3／4，只要不是 5（戰鬥中）
就接，所以隊伍選單開的也有（`itemPageTradable`）。選項列是
`READY USE TRADE DROP HALVE JOIN EXIT`，與商店裡量到的 `READY TRADE DROP HALVE JOIN SELL ID EXIT`
（spec 067）同一個順序。

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

### Use 的兩道門（位址 exact，`+10Dh` 的讀法 strong inference）

```
0F8C  C4 7E C1                   角色記錄
0F8F  26 80 BD 0D 01 00 / 74 5F  +10Dh 為 0 → 0FF6h（不接 Use）
0F97  C4 3E 33 49                [4933h]
0F9B  26 83 BD CA 01 00 / 75 53  +1CAh（word）非 0 → 0FF6h
```

`[4933h]+1CAh` 是 class 0 的 ECL 變數 `@49E5`（`6E00h + 49E5h × 2 ≡ 1CAh`）。
`cmd/pool-ecl-memory-audit -addresses 49E5` 在全部 ECL 裡只找到一個引用：ECL8 block 16
`9BEEh` 的 `SAVE`，`pool-text-inventory` 同一段 `9C22h` 是 "YOU SENSE AN ANTI-MAGIC SHELL
AROUND YOU."——它是**反魔法區**的旗標（語意 strong inference），區塊載入時 overlay-07
`025Ah` 清成 0（spec 106，`gamepack/block_load.go` 已照做）。remake 讀
`eventMachine.Memory[0x49E5]`。

`+10Dh` 在 remake 沒有欄位。原版的寫入端都與 `+10Ch`（狀態）一起寫：overlay-25 entry 28
（`2266h`）扣血後狀態不在 {0, 1} 就寫 0（spec 084）、逃離與轉化不死生物寫 0、建角與神殿的
石化解除寫 1 並把狀態寫 0（spec 063／115）。所以讀成「狀態不大於 1」（strong inference），
倒下、瀕死、死亡的人沒有 Use。

### 戰鬥外 Use 的 "uses an item"（exact）

```
1B2D..1B4A  push [5CF0h]、"uses an item"（1A73h `0C 75 73 65 73 20 61 6E 20 69 74 65 6D`）、0Ah、0
            9A 84 00 0A 01   overlay-25 entry 20：名字＋那一句，最後旗標 0 → 不停
1B4F        80 3E 54 49 05 75 3E   戰鬥外 → 1B94h（不印 "Item:"）
1B94..1BAE  push [5CF0h]、物品、1、16h、0、1
            9A 25 00 0A 01   overlay-25 entry 1：旗標 (欄 1, 列 16h, 不加 Yes/No, 印) → 第 22 列印物品名
1BB3        9A 61 00 98 01   overlay-37 entry 13：等一拍（遊戲速度 × 225 ms）
1BB8        9A 89 00 0A 01   overlay-25 entry 21（1830h）：戰鬥外清 (1, 12h)..(26h, 16h) 那一塊
1BD8        9A 39 00 E2 00   overlay-22 entry 5
```

entry 20 戰鬥外（`17A3h..181Bh`）開框、第 `12h`／`13h` 列印名字、下一列起印那一句；
entry 1 的最後一個旗標非 0 才以 `0198:002F`（欄 = 第一個旗標、列 = 第二個）印（`0735h..074Bh`）。
卷軸（`DS:6CB3h` 為 0）整段跳過，與戰鬥中相同（spec 144）。

remake：`useNoticeOnPage` 在選項列那一行印「名字 USES AN ITEM 物品名」，停
`speedDelayTicks()` 格、這段時間不收鍵，停完才進 `beginItemPageSpell`（挑對象或
"Use it?"）。遊戲速度為 0 時不停。版面是 remake 的呈現，停拍與順序照原版。

## Trade：overlay-19 entry 13（`173Bh`，exact）

`1328h` 'T' 先過 entry 20（`0D72h`，spec 067〈放手檢查〉：穿戴中印 "Must be unreadied"、
抄寫中的卷軸問 "is it Okay to lose it?"），過了才 `1341h` 呼叫 entry 13：

```
1741  les ax, [467Ch]              上一次交給誰（entry 5 `0AD9h` 開頁時設成自己）
174D  9A D9 00 0A 01               overlay-25 entry 37
1757..1769  "Trade with Whom?"（171Fh `10 54 72 61 64 65 …`）、旗標 1
            9A F2 00 0A 01         overlay-25 entry 42（2C81h）：從 [467Ch] 起挑一個隊員
176E  結果為 NULL（Exit／ESC）→ 17DBh 什麼也不做
177B  [467Ch] = 挑到的那一個
178F  E8 BD 0F                     entry 9（274Fh）(對方, 物品)
1792  非 0 → 179Bh "Overloaded"（1730h `0A 4F 76 65 72 6C 6F 61 64 65 64`）
             9A 7F 00 0A 01        overlay-25 entry 19：第 24 列印、停一拍、清
17B8  9A 7A 00 0A 01               overlay-25 entry 18（164Bh）(對方, 物品)：GetMem(3Fh)、複製、接在對方串列最後
17CB  9A 75 00 0A 01               overlay-25 entry 17 (自己 = [5CF0h], 物品)：從自己身上拿掉
17D6  9A 43 00 0A 01               overlay-25 entry 7 (對方)：重算
```

overlay-25 entry 42 的選單是 prompt + " Select"（`2C7Ah`）+ " Exit"（`2C52h`，旗標非 0 才加），
沿 `DS:5CF4h` 整條隊伍前後換人（`2D5Dh..2DF9h`），**自己也挑得到**；Enter 或 ESC 離開
（`2C58h` 的集合：`0Dh`、`1Bh`），按鍵是 'E' 或 0 回 NULL（`2DFDh..2E0Eh`）。交給自己就是
把那一件搬到最後一格。

### 負重檢查：overlay-19 entry 9（`274Fh`，exact）

```
275B  9A 43 00 0A 01              overlay-25 entry 7 重算接收者（+C7h 物品數、+102h 總負重）
2767  26 80 BD C7 00 0F / 76 04   +C7h > 0Fh → 超重
2776  26 8B 45 37                 重量 = 物品 +37h（word）
2780  26 80 7D 39 00 / 76 0F      +39h > 0 → 2790 F7 66 FC 重量 × +39h（取低字）
279C  9A 6B 00 0A 01 / 05 DC 05   上限 = overlay-25 entry 15（力量表）+ 5DCh（1500）
27AC  26 8B 85 02 01 / 03 46 FC   +102h + 重量（16 位元）
27B6..27BE                        32 位元有號比較，大於上限 → 超重
27D0  CA 08 00                    retf 8，回傳超重旗標
```

這一支就是 spec 035 拾取時走的同一支（overlay-06 經 `C9:004D`）。`+102h` 是物品重量加
七欄錢的枚數（spec 079），所以錢多的人收不下；15 件還收得下第 16 件（`> 0Fh`）。

remake：`poolcharacter.ReceiveOverloaded`（`internal/character/overload.go`，含錢）；
`startItemPageTrade`／`askItemPageTrade`／`itemPageTradeInput`／`tradeMemberItem`
（`cmd/pool-game/item_page_trade.go`）。挑人的畫面沿用挑對象那一種（上下換人、Enter 選、
ESC 離開），起點是這一頁上一次交給的人，開頁時是自己。spec 035 的
`CanReceiveItem`（拾取那一側）的負重沒有算錢，與這一支不一致，留在〈未閉合〉。

## 84h：陣營限制（overlay-12 entry 123 `30B1h`，exact）

```
30C2  80 7E 06 00 / 74 02 / EB 58  模式非 0（卸下）→ 什麼也不做
30CA  26 8A 45 3D / 24 0F          物品 +3Dh & 0Fh
30D3  26 3A 85 A0 00 / 74 45       == 角色 +0A0h（陣營）→ 什麼也不做
30DD  26 C6 45 34 00               物品 +34h = 0（卸下來）
30E5..30F4  +3Dh ÷ 10h → 傷害
30F7  C6 06 77 67 08               DS:6777h = 8（魔法）
30FC  戰鬥中先叫 overlay-25 entry 41（2BC1h，只清右欄與第 24 列、放游標）
3108..3118  push 角色、傷害、0、0 / 9A 7F 00 00 01   overlay-24 entry 19（規則 0、沒豁免）
311D  C6 06 7F 67 01               DS:677Fh = 1
```

overlay-24 entry 19（`133Ah`）這一條路：

```
1341  DS:6776h = 傷害；1351 E8 8E EF 派發受傷那一個的群組 6（`DS:6779h` 為 0）
137A  6776h 為 0 → 1642h 整段跳過（不印、不扣）
1384  6776h == 1 → "takes 1 point of damage "（1297h）
      否則 "takes " + 數字 + " points of damage "（127Dh、1284h）
13DE  6777h & F7h 是 1／2／4／10h → 接 "from Fire"／"from Cold"／"from Electricity"／"from Acid"
14A8  6777h & 8 == 6777h → 接 "from Magic"（12DFh）
14EC  9A A2 00 0A 01               overlay-25 entry 26（2041h）(角色, 0, 字串)：
                                   戰鬥外 21C1h 以 entry 20(角色, 字串, 0Ah, 1) 印名字＋那一句，停一拍
14FB  9A AC 00 0A 01               overlay-25 entry 28（2266h）扣血（spec 084 的狀態轉移）
1560  +10Dh 非 0 → 163Dh
156E  "Goes Down"（12F7h）；狀態 5 接 ", and is Dying"（1301h）；
      狀態在 1310h 的集合（`C0 01` → 6、7、8）改成 "is killed"（1330h）
15F6  entry 20(角色, 字串, …, 0)；戰鬥外 1602h 等一拍
163D  entry 21 清掉
```

原版物品裡帶 84h 的有三件（掃過 ITEM1..8.DAX、MON1..8ITM.DAX、預設人物 `.itm`）：

| 來源 | `+3Dh` | 限定陣營 | 受傷 |
|---|---|---|---|
| ITEM4.DAX block 29 第 6 件 "Long Sword" | `F0h` | 0 守序善良 | 15 |
| ITEM5.DAX block 35 第 0、1 件 "Long Sword" | `52h` | 2 守序邪惡 | 5 |

ITEM4 那一件的 63 bytes：`0A 4C 6F 6E 67 20 53 77 6F 72 64 00 … 00 02 00 EA 14 24 00 A3 24 02 00
00 06 00 3C 00 00 A0 0F 00 F0 84`（`+2Eh` 型別 24h 長劍、`+3Dh` F0h、`+3Eh` 84h）。

remake：`gamepack.AlignedWearDamage`（`wear_effect.go`）判卸不卸、傷害多少；
`(*app).alignedWear`（`item_page_trade.go`）接在 `wearItem` 裡：卸下、群組 6
（`SpellDamageEffects`，旗標 8）、戰鬥外以 `gamepack.ApplyDamage` 扣角色的生命值，
戰鬥中扣盤面那一格（受傷打斷施法與倒下離場照其他傷害）；訊息是
「名字 TAKES N POINTS OF DAMAGE FROM MAGIC」，倒下再接「GOES DOWN」「GOES DOWN, AND IS DYING」
或「IS KILLED」。陣營值：玩家角色取建角的編號（`creation.Alignments` 的順序），NPC 讀記錄
`+0A0h`。原版兩句各停一拍；remake 的物品頁把它們放在同一行、不停拍（呈現）。

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
- `internal/gamepack/wear_effect_test.go`：7Fh 門檻、83h 拿下時摘自己那一個、89h、84h。
- `cmd/pool-game/item_page_trade_test.go`（#105）：`TestItemPageTradeGivesTheItemAway`、
  `TestItemPageTradeRefusesAnOverloadedRecipient`（16 件、錢壓到上限、15 件收得下第 16 件）、
  `TestItemPageTradeHiddenForAMoraleNPC`、`TestItemPageUseNeedsAStandingMemberOutsideAntiMagic`、
  `TestItemPageUseWaitsABeatAfterUsesAnItem`、`TestAlignedSwordHurtsTheWrongWearer`
  （ITEM4／ITEM5 的兩把真實長劍）。

## 未閉合

- overlay-25 entry 37（`280Fh`，Trade 開頭那一次）沒讀；它不帶參數、不回傳值，看形狀是
  清畫面的一段，remake 不做。
- 84h 在戰鬥中另叫 overlay-25 entry 41（只清畫面）與 entry 19 戰鬥那一支的 `1609h`
  （`1004h`、群組 13、`13D:0084` 離場）；remake 戰鬥中的離場照 `applySpellDamage` 那一套，
  `1004h` 與群組 13 沒有逐條對過。
- spec 035 拾取那一側的 `CanReceiveItem` 算負重時沒有加錢，與 `274Fh`（`+102h` 含錢）不一致；
  那一條路不在物品頁，沒改。
- 物品頁的版面（清單、選項列、訊息那一行）是 remake 的呈現，原版的版面沒量。
