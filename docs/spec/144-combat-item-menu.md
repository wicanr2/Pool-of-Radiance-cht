# Spec 144：戰鬥中的物品選單——Ready、Drop、Halve、Join、卷軸與 `+0Bh` 為 0 的物品法術

狀態：READY（四個選項在戰鬥中的規則與「不用掉行動」、卷軸挑一行與抹掉、參數表 `+0Bh`
為 0 的法術用物品放出時的結果，都從 overlay-19／22／25／16 讀出並實作，
`cmd/pool-game/combat_item_menu_test.go` 從 `Update()` 送鍵驗）；DRAFT（見〈未閉合〉）。
日期：2026-09-26。主台帳：GitHub issue #84（接 #75 的 U，spec 098〈用物品放法術〉）。

## 一句話

物品選單（overlay-19 entry 6）裡 Ready、Drop、Halve、Join **都不用掉行動**：它們不動
選單的結果，選單迴圈只在結果非 0 或身上沒東西時才離開，所以換完武器還能再按 U 或
回指令列攻擊，而新武器的 THAC0、傷害與射程當場生效。Ready **不會自動換手**：那一格
已經有東西就印 "already using"。卷軸讓玩家挑一行放，放完抹掉那一行。參數表 `+0Bh`
為 0 的法術（營地限定那幾條）用物品放出時照樣放，但不呼叫 entry 34——分數不歸零，
同一個人重選。

## 輸入

- `overlay-19.bin` SHA-256 `4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9`
- `overlay-22.bin` SHA-256 `967065cc3597…`（`docs/audit/dos-ovr-manifest.json` 那一筆）
- `overlay-25.bin` `9fede24be1e6…`、`overlay-16.bin` `a142d8a8f3b3…`、`overlay-08.bin` `932ce28178ff…`
- `START.EXE` SHA-256 `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`
- 工具：`coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel`，位址是
  overlay-local offset；far call `9A off seg` 以 `seg × 16 + 3B0h + off` 對 manifest 的
  stub `executable_file_offset` 反查（spec 109）。`+B0h` 的寫入端另以
  `cmd/pool-disp-scan -addresses B0,5AA` 掃過（`5AAh` 的四條 overlay-14 正對照出現）。

## 選單怎麼組、按鍵怎麼分（exact）

overlay-19 entry 6（`0EFBh`），`DS:4954h` == 5 是戰鬥中：

| 位址 | 位元組 | 內容 |
|---|---|---|
| `0F79h` | `BF 86 0E` | " Ready"（`0E86h`）一律接 |
| `0F8Ch..0FCCh` | `26 80 BD 0D 01 00`…`26 80 7D 02 00` | " Use"：`+10Dh` 非 0、`[4933h]+1CAh` 為 0，戰鬥中另要 runtime `+2` 非 0（spec 129）|
| `0FF6h..1017h` | `26 80 BD 84 00 80`、`80 3E 54 49 05 74 28` | " Trade"：戰鬥中不接 |
| `1046h` | `BF 98 0E` | " Drop" 一律接（**沒有** NPC 條件；那一條擋的是 Trade 與 Sell）|
| `106Eh` | `26 80 BD C7 00 10 73 28` | " Halve"：身上件數 `+C7h` < 10h |
| `10A1h` | `BF A5 0E` | " Join" 一律接 |
| `129Ch`／`12B0h`／`1351h`／`1412h`／`1422h` | `3C 52`／`3C 55`／`3C 44`／`3C 48`／`3C 4A` | R／U／D／H／J |
| `1469h` | `9A 43 00 0A 01` | 每一個選項做完都呼叫 overlay-25 entry 7（`0BBEh`）整份重算 |
| `1485h` | `E9 A7 FA` | 回到 `0F2Fh` 選單迴圈 |
| `0F42h` | `26 80 3D 00 74 03` | 結果 `[bp+6]` 非 0 才離開 |
| `0F51h` | `26 80 BD C7 00 00 77 03` | 身上沒東西就離開 |

R／D／H／J 四支都不寫 `[bp+6]`，所以**不用掉行動**（overlay-08 `036Ch` 看的就是這個
結果）。remake：`combatItemMenuInput`、`afterCombatItemChange`（重算走
`applyPartyGearStats`，與開打時同一支）、`combatItemFooter`（選項列照上表的條件組）。

## Ready：overlay-19 entry 7（`14CAh`，exact）

穿戴中的（`+34h` 非 0）：`14FAh` `+36h` 非 0 印 "It's Cursed"（`148Eh`），否則 `151Ah`
寫 `+34h = 0`。沒穿的依序判，後面成立的蓋掉前面的（同一格 `[bp-2]`）：

```
1571  型別表 +1（DS:54E1h，手數）+ 角色 +100h > 2        → 3 "Your hands are full!"（14B5h）
159C  類別 0..8 且 +CCh + 類別×4 已有；類別 9 且 +F4h 已有 → 2 "already using " + 那一件（14A6h）
15D6  型別 49h 且 +F8h 已有；型別 1Ch 且 +FCh 已有         → 2（擋的是 +F8h／+FCh 那一件）
1624  型別表 +0Dh（DS:54EDh）& 角色 +0B0h == 0             → 1 "Wrong Class"（149Ah）
1642  0 → +34h = 1
16F3  3 在戰鬥中而且 +10Fh 非 0（電腦接手）時不印
```

槽與手數是 overlay-25 `0C21h..0D6Eh` 沿物品串列認的（`+100h` 在 `0C0Ch` 清 0、
`0D57h..0D6Eh` 累加型別表 `+1`），與 spec 065 的槽表同一段。類別 9 那一格問 `+F4h`、
印的是 `+CCh + 9 × 4` 也就是 `+F0h` 那一件，照位元組。

角色 `+0B0h` 是 overlay-16 `0CAEh` 清 0、`0D3Fh` 對每個等級大於 0 的職業加上
`DS:05EAh` 那一格。那八格是 `02 10 08 40 80 01 04 20`（START.EXE file offset
`32154`，DS 位移換算以 `DS:3C16h` → `28 28 28 28 2A` 正對照）。寫 `+0B0h` 的只有
這兩處（`pool-disp-scan`），讀的是 overlay-19 `161Ch` 與 overlay-09 的 AI。

`+3Eh` 大於 7Fh 的物品還會交給 overlay-24 entry 1（`153Fh`／`1667h`）掛上或摘掉穿戴
效果；remake 沒有那一層（見〈未閉合〉）。實作：`gamepack.ReadyItem`、`ClassUseMask`。

## Drop（exact）

`1358h` 先過 entry 20（`0D72h`，spec 067〈放手檢查〉：穿戴中印 "Must be unreadied"；
卷軸有人正要抄就先問）。過了印 "Your "（`0ED6h`）＋名稱＋"will be gone forever"
（`0EDCh`），再問 "Drop It? "（`0EF1h`，overlay-26 entry 6）；`13DEh` `3C 59` 只認 Y，
Y 才 overlay-25 entry 17（`156Ah`）摘掉。

## Halve：overlay-19 entry 14（`17F2h`，exact）

一半 = `+39h` div 2（`17FBh..1807h`）。為 0 印 "Can't halve that"（`17E1h`）。否則配一筆
3Fh bytes（`182Eh`）、整筆複製（`1841h`），新的 `+39h` = 一半、`+34h` = 0，接在原件後面
（`1858h..1882h`），原件 `+39h` = 數量 − 一半。實作：`gamepack.HalveItem`。

## Join：overlay-19 entry 15（`18A2h`，exact）

沿整條串列，跳過**目前收件的那一件**（`18E8h` 比的是 `[bp-0Ch]`），數量 > 0、而且
`+2Fh`／`+30h`／`+31h`／`+2Eh`／`+32h`／`+33h`／`+36h`／`+37h`(word) 都相同、`+3Ch` < 2
的疊進來。`199Fh`／`19BFh`／`19D2h` 三處把那一件的 `+3Ch`／`+3Dh`／`+3Eh` 跟**自己**比
（兩個運算元都是 `[bp-4]`），恆成立——充能與法術欄不必相同，照位元組。

累加 ≤ FFh（`19F4h`）就加上並以 overlay-25 entry 17 摘掉那一件；超過時（`1A18h`）收件
那一件設成 FFh、這一件減去收件那件還放得下的量、改由這一件接著收。最後把累加值寫回
收件的那一件（`1A63h`）。實作：`gamepack.JoinItems`。

## 卷軸（exact）

entry 8（`1A86h`）認出卷軸（overlay-22 entry 6，類別 0Bh..0Dh）之後：

```
1AA9  DS:6A74h = 卷軸
1AC6  overlay-19 entry 12（297Bh）(模式 2, 1, &索引 = FFFFh, &…) → 法術
2A47  overlay-22 entry 2（0588h）(2) → entry 12（0441h）(0) 列行
0441  身上有效果 10h（overlay-25 entry 27）或 牧師等級 +96h > 0 且類別 == 0Ch
        → 卷軸 +35h = 0（揭開藏字）
048C  +35h 仍非 0 → 一行也不列
04B8  +3Ch..+3Eh 值 > 0 的都列（位元 7 立著的前面加 " *"，03BDh）
2ADE  overlay-22 entry 1（008Ah）挑一行，回法術編號（& 7Fh）
1B19  DS:6CB3h 為 0（卷軸）→ 不印 "uses an item"，直接 1BBDh
1BBD  DS:6CB3h = 1 → overlay-22 entry 5 放出去（等級見 spec 098〈施法者等級〉）
1C1D  結果非 0 → overlay-22 entry 7（3243h）抹掉那一行
```

效果 10h 是閱讀魔法（法術 18，參數表 `+0Ah` = 10h）。類別 0Bh 是法師卷軸
（型別 3Dh，型別表 `+0Dh` = 01h）、0Ch 是牧師卷軸（型別 3Eh，`+0Dh` = 02h）；兩者
型別表 `+1` 都是 2——**卷軸要兩隻手**，Ready 的手數規則照樣適用。

entry 7（`3243h`）：法術 & 7Fh 對 `+3Ch..+3Eh` 各 & 7Fh，**最後一個**相同的清 0，
`+30h` 減一，`+30h` < D2h 就 overlay-25 entry 17 拿掉卷軸。`+30h` 是名稱字詞：
D2h "With 1 Spell"、D3h "With 2 Spells"、D4h "With 3 Spells"（`DS:10BBh` 字詞表），
所以名稱跟著少一條。一行都對不上就不動。

列不出行（藏字沒揭開或三行都空）時 entry 12 回 0 → entry 8 結果 0 → 戰鬥中留在物品
選單，什麼也不印。挑的時候 ESC 同樣回 0。

實作：`gamepack.ScrollReadable`／`ScrollSpells`／`EraseScrollSpell`；
`openScrollPick`、`scrollPickInput`、`scrollEraser`。

## 參數表 `+0Bh` 為 0 的物品法術（exact／strong inference）

overlay-22 entry 5（`0C14h`）在戰鬥中**不看** `+0Bh`：`0C2Ah` `80 3E 54 49 05 75 03`
戰鬥中直接跳到 `0D23h`，略過非戰鬥時 `+07h`（`DS:319Bh`）為 0 的確認；之後照常經
`DS:6A78h` 瞄準（`0D63h`，回傳寫進結果）、放出（`0EA3h`）。整支 entry 5 沒有
`DS:319Fh` 的引用（exact）。

回到 entry 8 `1BF4h` `80 BD 9F 31 00 74 13`：`+0Bh` 為 0 就**不呼叫 entry 34**，結果留著
瞄準的回傳。所以（exact）：

- 放出去（瞄準回非 0）：記帳（充能減一／卷軸抹一行），選單離開，overlay-08 `036Ch`
  指令迴圈以結果非 0 離開——**但分數沒歸零**。
- 放棄瞄準（結果 0）：不記帳，戰鬥中留在物品選單（`1318h`）。

指令迴圈離開而沒有 entry 34 之後的重選，與「開始施法」（overlay-13 `24E7h`，
spec 098〈施法時間與打斷〉）同一個形狀：這一格還在排序裡，分數最高就再輪到他
（strong inference：overlay-08 entry 4 的呼叫端沒另讀）。remake：`spellCasting.keepTurn`
→ `keepCombatTurn`（與 `beginCasting` 同一種重選）。

`+0Bh` 為 0 的是 14、18（閱讀魔法）、22、31、35、39、67，全部是模式 0（打自己），
所以實際上瞄準不會被放棄；放棄那一支照位元組實作但沒有資料走得到。

## 探索中（非戰鬥）的差異（exact，remake 未接）

- `130Ah..1314h`：不在戰鬥中時 entry 8 之後把結果強制清 0——用完物品不離開選單。
- entry 8 `1B01h`／`1BE2h`：只有戰鬥中才重畫、才看 `+0Bh` 呼叫 entry 34。
- entry 5 `0C34h..0D1Bh`：非戰鬥時參數表 `+07h` 為 0 的是「只能在戰鬥中用」。記憶的法術
  印 "is a combat-only spell..." / "Lose it? "（`0BA2h`／`0BBCh`），Y 就從記憶清掉；
  物品印 "That Item" / "is a combat-only item..." / "Use it? "（`0BC6h`／`0BD0h`／`0BE9h`），
  Y 就把結果設 1（`0D17h`，照樣記帳）。兩種都**不放出去**（`0D1Fh` `[bp-1] = 0`）。
- 選單多 Trade（`0FF6h` 的 NPC 條件）；商店中（`4954h` == 1）另有 Sell、Id（spec 067）。

remake 的探索物品頁（`equipment.go`）只有 Enter 切換穿戴，而且是**自動換掉同類那一件**，
與上面 entry 7 的「那一格有東西就拒絕」不同；探索中也沒有 Use、Drop、Halve、Join。

## 未閉合

- `+3Eh` 大於 7Fh 的穿戴效果（overlay-24 entry 1）沒有接，探索與戰鬥都一樣。
- 探索中的物品選單（上一節）沒有接；探索頁的 Ready 規則與原版不同。
- "already using" 那一句前面原版先以 overlay-25 entry 1（`16B9h`）印一段，引數是角色與
  那一件；確切組字沒讀，remake 只印 "ALREADY USING <物品>"。
- 選單版面（右側清單、選項列）是 remake 的呈現，原版物品選單的版面沒量。
