# Spec 163：逐人效果訊息（掛上、解除、老化）與編號 57 的呼叫端

狀態：READY（字串與印法逐條讀過位元組並實作，從 `Update()` 送鍵測試）。證據是位元組
（exact），另註者除外；沒有原版執行期收據逐次對過。日期：2026-09-27。主台帳：GitHub issue #87。
接手 spec 098〈模式 0Ah：效果怎麼掛上去〉〈只掛效果的那一批〉兩張「與原版不同」表裡
「狀態列只留總結那一句」那兩列，以及 spec 074〈57..67 這十一個編號是物品效果〉的「呼叫端還沒找到」。

## 輸入

- DOS ZIP `Pool of Radiance (1988).zip`，SHA-256 `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`；
  `START.EXE` `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`、
  `GAME.OVR` `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638`。
- overlay 由 `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json`）：

| overlay | SHA-256 | far call 的段 |
|---|---|---|
| 12 | `d1b057432515b7daa2b83199d2588c131a120037d30117e664b01e4c6f2bcb7e` | 經群組派發（spec 112） |
| 22 | `967065cc35975465a7250026c63b8a5ae06b812b228abcfbbbd83d636538dda8` | `00E2h` |
| 24 | `e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714` | `0100h` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` | `010Ah` |

- 工具：`coab-go-test:20260729` 的 GNU objdump 2.40（`-D -b binary -m i8086 -M intel`），
  位址是 overlay 檔內位移；far call 以清冊反查（segment = (executable_file_offset − stub_offset − 3B0h) ÷ 16）。
  overlay-25 的 entry 20 = `010Ah:0084h`（`1738h`）、21 = `0089h`、26 = `00A2h`（`2041h`）；
  overlay-24 的 entry 15 = `0100h:006Bh`（`107Bh`）。
- 字串是 Pascal 常數（長度 byte ＋ 內容），呼叫端 `mov di, 字串` ＋ `0x5BB:0x634` 抄進暫存。
  派發表與每一支處理常式的字串由 `cmd/pool-spell-dispatch` 讀出（`gamepack.ReadDOSSpellDispatchTable`）。

## 三條印法

### overlay-24 entry 20（`1656h`）：掛上之後

```
1716  E8 3B F7                  0E54h 掛新的節點
1719  80 7E D7 00 / 74 18       訊息（處理常式推給 08BCh 的那一段，抄在 [bp-29h]）長度 0 → 不印
171F  FF 76 18 / FF 76 16       目標
1725  B0 01 50                  旗標 1
1728  8D 7E D7 16 57            訊息
172D  9A A2 00 0A 01            overlay-25 entry 26
1732  9A 89 00 0A 01            overlay-25 entry 21（清右欄）
```

群組 9 擋下與「豁免成功而且規則是 1」兩條路在 `1682h`／`1689h` 就返回（印 "is Unaffected"，
spec 098），不會走到這裡。

**overlay-25 entry 26**（`2041h`，(記錄, 旗標, 字串)）：

- 戰鬥中（`DS:4954h == 5`）：`20DDh` entry 20(記錄, 字串, 0Ah, 0)，接著閃光動畫：旗標非 0 時
  跑 `DS:4943h`（遊戲速度）+ 1 輪（`20E6h..20ECh`），每輪四格、每格 `0512h:029Eh`（Delay）46h ms
  （`2173h`）；`[bp-2Fh]` 為 0（速度 0）才另外 `21BAh` 等一拍。與 "is turned" 同一支（spec 098
  〈其餘戰鬥訊息〉）。
- 戰鬥外（`21C1h`）：entry 20(記錄, 字串, 0Ah, 1)。entry 20 在戰鬥外（`17A3h..181Bh`）清下方
  訊息框（列 11h 或 12h 起，`DS:6CCEh` 決定）、名字在下一列、字串再下一列，停一拍再清。

### overlay-24 entry 15（`107Bh`）：解除

```
1094  9A A7 00 0A 01            010Ah:00A7h(目標, 碼) 找最早的同碼節點
109B  74 38                     沒有 → 回 0
10A8  BF 72 10                  "is Cured"（1072h：08 69 73 20 43 75 72 65 64）
10B8  9A 84 00 0A 01            overlay-25 entry 20(目標, 字串, 0Ah, 1)
10CE  E8 57 EF                  0028h 摘掉那一個
10D1  C6 46 FF 01               回 1
```

先印才摘。全部的呼叫端都在 overlay-22（其餘 overlay 掃 `9A 6B 00 00 01` 零筆）：

| 位置 | 問的碼 | 法術 |
|---|---|---|
| `138Dh` | `0Ch` | 縮小術（`135Eh`）|
| `21F9h` | `21h` | 解盲（`21E8h`），回 1 再經 entry 26 印 "can see"（`21E0h`）|
| `2275h`、`228Dh`、`22D1h` | `22h`、`2Bh`、`32h` | 解病鏈 `225Bh`（解病術、編號 58）|
| `2519h` | `24h` | 除咒（`2508h`），回 1 再經 entry 26 印 "is un-cursed"（`24E6h`）|
| `279Fh` | 對面那一支 | 急速／緩速 `2724h` 的分邊（spec 098）|
| `2DC8h` | `2Ah` | 編號 57（`2DB7h`）|
| `2E13h` | `37h` | 編號 58（`2E02h`）|

### overlay-12 entry 36（`0C67h`，群組 18 的 `27h`）：老化

```
0C70  26 8A 45 03 / 24 10 / 75 3A   節點 +3 位元 4 立著 → 跳過
0C83  05 10 00 / 26 88 45 03        立起位元 4
0C98  BF 62 0C                      "ages"（0C62h：04 61 67 65 73）
0CA8  9A 84 00 0A 01                overlay-25 entry 20(目標, 字串, 0Ah, 1)
0CB0  26 FF 45 30                   記錄 +30h 加一
```

回合初始化與急速放出去當下（`2835h`）都派發群組 18（spec 098），只有第一次印。

## 訊息對照

處理常式推給 `08BCh`（模式 0Ah 的四支先推給 `0F35h`／`2724h`，它們在 `0FDBh`／`27ECh` 把
`[bp-29h]` 原樣轉給 `08BCh`）、由 entry 20 `171Fh` 印的那一批。位址是長度 byte；每一段都
驗過「本體用 `mov di, 位址` 取它、隨後 `call 08BCh`」（或 `0F35h`／`2724h`）。exact。

| 編號 | 字串位址 | 原版字串 |
|---|---|---|
| 1 | `0FEAh` | "is Blessed" |
| 2 | `101Ch` | "is Cursed" |
| 5、11、18、22、29 | `10C5h` | "is affected" |
| 6、7、16、17、52、53、54 | `10FEh` | "is protected" |
| 8 | `1138h` | "is cold-resistant" |
| 10 | `11BAh` | "is charmed" |
| 14 | `13BCh` | "is friendly" |
| 19 | `1485h` | "is shielded" |
| 21 | `1506h` | "falls asleep" |
| 24 | `17C2h` | "is fire resistant" |
| 25 | `1801h` | "is silenced" |
| 26 | `183Ah` | "is affected" |
| 27 | `18EEh` | "is charmed" |
| 30、50 | `19EEh` | "is invisible" |
| 32 | `1A61h` | "is duplicated" |
| 33 | `1AA5h` | "is weakened" |
| 38 | `2225h` | "is blind" |
| 40 | `2311h` | "is diseased" |
| 42 | `2492h` | "is praying" |
| 44 | `25B6h` | "has been cursed!" |
| 45 | `25F4h` | "is blinking" |
| 48 | `2848h` | "is Hasted" |
| 55 | `2BBDh` | "is Slowed" |
| 57 | `2DADh` | "is Speedy" |
| 61 | `2F41h` | "is paralyzed" |
| 63 | `2FD3h` | "is invisible" |
| 67 | `3050h` | "is Reading" |

開鎖術（31）推 "Knock-Knock"（`1A28h`），但參數表 `+0Ah` 是 0，`0A13h` 不叫 entry 20，永遠
印不出來。

處理常式自己印的幾句（不經 `08BCh`）：

| remake 鍵 | 原版字串（位址） | 呼叫端 | 印法 | 證據 |
|---|---|---|---|---|
| `ui.noticeCured` | overlay-24 `1072h` "is Cured" | entry 15 `10A8h..10B8h` | entry 20(…, 0Ah, 1)：名字＋一句，一拍 | exact |
| `ui.noticeAges` | overlay-12 `0C62h` "ages" | `0C98h..0CA8h` | entry 20(…, 0Ah, 1) | exact |
| `ui.noticeCanSee` | overlay-22 `21E0h` "can see" | `2202h..221Ch` | entry 26(…, 1) | exact |
| `ui.noticeUncursed` | overlay-22 `24E6h` "is un-cursed" | `2522h..253Ch` | entry 26(…, 1) | exact |
| `ui.noticeItemUncursed` | overlay-22 `24F3h` "'s item is un-cursed" | `258Dh..25ADh`（沒有 `24h`、清到物品才印）| entry 26(…, 1) | exact |
| `ui.noticeHealed` | overlay-22 `2DF8h`（58）、`2F7Bh`（62）"is Healed" | `2E67h..2E85h`、`2FA7h..2FCAh`：entry 21 回 1 才印 | entry 26(…, 1) | exact |
| `ui.noticeAffected`（解除魔法） | overlay-22 `234Ah` "is affected" | `2460h..247Eh`：這一趟摘到東西（`[bp-7]`）才印 | entry 26(…, 1) | exact |

英文照原版字串、以大寫字模顯示；繁中是對應譯句。

## 編號 57 的呼叫端

物品記錄 `+3Dh` 大於 38h 時減 17h 才是法術編號（overlay-09 `04C7h`、overlay-19 `1B23h`，spec 098
〈用物品放法術〉），所以編號 57（39h）來自 `+3Dh` = 50h。掃原版資料裡每一筆 3Fh bytes 的物品記錄：

| 來源（SHA-256 前 8 碼）| 位置 | 型別 `+2Eh` | `+3Ch` |
|---|---|---|---|
| `ITEM1.DAX`（`54216036`）block 09h | `+0FCh` | 46h | 1 |
| `ITEM5.DAX`（`4c033806`）block 22h | `+0` | 47h | 1 |
| `MON7ITM.DAX`（`86ef5f43`）block 51h | `+0` | 47h | 1 |
| `CHRDATB1.ITM`（`d04c68ae`）| `+1F8h` | 46h | 1 |
| `CHRDATE1.ITM`（`95a7f878`）| `+1F8h` | 46h | 1 |

五筆 `+3Eh` 都是 0（AI 與玩家的 Use 都不擋）。所以呼叫端就是用物品：玩家的 U）se（overlay-19
entry 8 → overlay-22 entry 5）與 AI 的 entry 3。位元組 exact；哪一個 ECL 事件給出 ITEM1／ITEM5
那兩塊沒有追（strong inference：寶物塊與怪物物品串列都有玩家拿得到的路）。

`2DB7h`：`B0 2A 50 9A 6B 00 00 01 08 C0 75 23`——表上第一格有緩速就經 entry 15 印 "is Cured"、
摘掉、返回；否則推四個 0 與 "is Speedy" 給 `08BCh`，掛參數表 `+0Ah` 的 27h。remake 在
`castSpell` 前段（`BlockedByEffect`）與營地的 `campSpellEffect` 都照這兩條走，戰場上用的是
盤面那一份串列（#116 之後）。

## remake 的對應

`cmd/pool-game/effect_notice.go`：

- 戰鬥中：`attachNotice` 是 `171Fh`（entry 26 旗標 1 → `turnedNotice`，停的長度是閃光動畫）；
  `curedNotice` 是 entry 15、`agesNotice` 是 `0C98h`（`panelNotice`，停一拍）；`sparkleNotice` 是
  處理常式自己的 entry 26。`effectAttachMessages` 是上面那張表；
  `TestEffectAttachMessagesMatchTheDispatchTable` 拿原版派發表逐格對英文。
- 呼叫點：`castEffectOnly`、`castSpell` 的模式 0Ah（`sideSpellTargets` 的 `cured` 回呼是 `279Fh`）、
  `BlockedByEffect`（57）、解盲／解病／除咒那一支（`removalNotices`）、治療那一支（62）、
  `neutralizeOnBoard`（58）、`reduceOnBoard`、解除魔法；老化在 `agePartyMember`（群組 18 的 `PartyAged`，
  怪物也印）。狀態列那一句總結照舊。
- 戰鬥外：`fieldNotice` 收「名字 那一句」，`showEffectBeats` 在探索施法的訊息列與物品頁的訊息列
  逐行放、每行停一拍（`speedDelayTicks`），停完才放原本那一句；停拍中 `effectBeatInput`
  吃掉按鍵（原版的 Delay 不讀鍵）。速度 0 時一拍是 0，直接放原本那一句（原版也一閃而過）。
  `campSpellEffect`（掛上、分邊的解除、57、老化）、`campSpecialSpell`（縮小術、解除魔法）、
  `fieldCureNotices`（解盲、解病、除咒、58、62）。

測試（`effect_notice_test.go`，從 `Update()` 送鍵）：`TestBlessNamesEachBlessedTarget`（英繁）、
`TestHasteNamesTheTargetAndItsAging`（英繁，下一回合不再印）、`TestHasteOnTheSlowedNamesTheCure`、
`TestCureDiseaseOnTheBoardNamesTheCure`（英繁）、`TestSpeedyItemNamesTheCureOrTheSpeed`、
`TestCampHasteShowsEachLineForABeat`（英繁，停拍中 ESC 不作用）。

與原版不同、寫明的幾處：

| 原版 | remake | 理由 |
|---|---|---|
| 戰鬥外 entry 20 名字與句子分兩列 | 排成一行「名字 句子」 | 探索施法與物品頁的訊息列只有一行 |
| 解除魔法的外層迴圈跑 `DS:6B88h` 趟、每趟摘到東西各印一次 | 印一次 | remake 只走一趟（spec 098〈解除魔法〉）|
| 催眠、魅惑、迷蛇、定身、變大、力量、恢復、死靈、臭雲各自的效果句 | 沒有逐人印 | 這幾支在 remake 走各自的分支；列在停止線〈各處理常式效果句〉（#112）|
