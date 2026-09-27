# Spec 151：射擊、彈藥與 AI 換武器（overlay-25 entry 43／44／45、overlay-13 射擊路徑、overlay-09 entry 9）

狀態：READY（彈藥查詢、射擊次數、瞄準與撞上去的閘門、扣彈藥與落地、`29h` 防護普通飛彈、
AI 換武器的位元組都已逐段讀出並實作，送鍵與 foeTurn 測試加變異檢查；entry 9 結尾每次都重算、
NPC 與怪物同一支重算，#109）；`0E09h` 的條件寫回卡在「殺了目標還有剩的攻擊次數要續打」，
見〈`0E09h` 的條件寫回〉。沒有原版執行期收據逐發對過彈藥數，證據是位元組（exact）加上獸人家
骰流的旁證。日期：2026-09-27。主台帳：GitHub issue #98、#106（`29h` 那一條）、#109。

## 輸入

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`（`poolrad/items`、`MON2ITM.DAX`）。
- `START.EXE` SHA-256 `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`；overlay
  由 `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` | `0051h` |
| 09 | `6b47e49d1a2e73427b9863560350965d6d5b43d3dd89df89d9b7f43a68409258` | `0058h` |
| 12 | `d1b057432515b7daa2b83199d2588c131a120037d30117e664b01e4c6f2bcb7e` | `006Ch` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` | `0096h` |
| 19 | `4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9` | `00C9h` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` | `010Ah` |

- 反組譯：`coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel`，overlay-local
  offset、base 0。far call 的 segment 以 (manifest `executable_file_offset` − 3B0h) ÷ 16 反查：
  `010Ah` = overlay-25（`0x1450`）、`0096h` = overlay-13（`0xD10`）、`00C9h` = overlay-19（`0x1040`）；
  stub offset `20h + i × 5` 是 entry i。

## 三支查詢（overlay-25，exact）

| entry | 位址 | stub | 回什麼 |
|---|---|---|---|
| 43 | `2E29h` | `010Ah:00F7h` | 記錄 `+0CCh` 有武器而型別表 `+0Ch` 大於 1（`2E52h`：`80 BD EC 54 01`）|
| 44 | `2E6Ch` | `010Ah:00FCh` | entry 43 而且旗標 `& 14h == 14h`（`2E98h`：`24 14 / 3C 14`）|
| 45 | `2EB1h` | `010Ah:0101h` | `f(記錄, &彈藥)`，`retf 8`，見下 |

entry 45 逐段：

```
2EBE  武器 = 記錄 +0CCh；*彈藥 = NULL；旗標 = 0
2EE3  有武器：旗標 = 型別表[武器 +2Eh] 的 +0Eh（DS:54EEh）
2EF9  旗標 & 10h → *彈藥 = 武器                 ; 丟出去的就是自己
2F11  旗標 & 08h：
2F1A    & 01h → *彈藥 = 記錄 +0F8h              ; 箭（型別 49h），NULL 也照寫
2F37    & 80h → *彈藥 = 記錄 +0FCh              ; 弩矢（型別 1Ch）
2F54  回 (*彈藥 != NULL) || 旗標 == 0Ah
```

所以 spec 065 旗標表的 bit 0「需要發射器」實際上是「要箭」：`+0F8h` 裝的是型別 49h（箭），
不是弓。型別表對得上：短弓族 `29h..2Ch` 旗標 `0Bh`、弩 `2Eh` 旗標 `8Ah`、匕首 `02h` 與手斧
`07h`、矛 `1Fh` 旗標 `14h`、`15h`／`55h..58h` 旗標 `1Ah`（丟自己）、`2Fh`／`7Fh` 旗標 `0Ah`
（不必彈藥）。

## 射擊次數與封頂（overlay-13 entry 8 `0D29h`，exact）

每回合初始化（overlay-13 `0042h`）、玩家指令迴圈（overlay-08 `03B1h`／`03FDh`）與 AI 換完武器
（overlay-09 `1813h`）都叫它：

```
0D3E  +113h = +0A1h（原本的攻擊次數編碼）
0D54  entry 43 且 entry 45：DS:6778h = 型別表 +05h（DS:54E5h），小於 2 墊成 2
      否則 DS:6778h = +113h
0DBB  群組 18（急速／緩速）
0DC5  e58h：(DS:6778h + 相位位元) ÷ 2
0DCB  射擊而且彈藥不是 NULL：次數 ≤ max(1, 彈藥 +39h)
0E09  寫回 +113h（`+108h` 的 `+8` 立著時另有條件，見〈還沒接〉）
```

短弓族 `+05h` 是 4：一回合兩發。remake：`gamepack.MissileGear.RateOfFire`／`VolleyLimit`，
`cmd/pool-game/missile.go` 的 `volleySwings`（第一形態換成射擊的次數，照樣過群組 18 與相位）。

## 玩家：瞄準選單的 Target（overlay-13 `2A7Bh`，exact）

瞄準迴圈 `352Ch` 每換一個目標叫 `2A7Bh` 畫選單列；`T` 鍵（`36CDh`）才進 `2C17h` 出手。
`2A7Bh` 在攻擊模式下（`[bp+8]` 非 0）決定要不要列出 `Target `（`2A50h`）：

```
2AEF  距離（entry 33 `2591h`）> 射程 → 不列
2B15  目標就是自己 → 不列
2B2A  entry 43 不成立（不是射擊武器）→ 列
2B53  entry 45 不成立（沒彈藥）→ 不列
2B65  entry 32（`246Dh`）以 1 找到對面的人（身邊有敵人）：entry 44 成立才列
      沒有 → 列
```

remake 沒有逐格重畫選單列，在 Enter 時用同一組條件擋下（`aimOffersTarget`），印
`ui.aimNoTarget`（remake 自己的說明句），回合不算用掉。

## 玩家：撞上去（overlay-08 `0D30h`，exact）

移動那一步探到有人（`0BC7h` 的 `013Dh:007Fh`）就叫 `0D30h`：

```
0D3C  entry 43 且不是 entry 44 → 印 "Not with that weapon"（`0D1Bh`），不打
0D76  其餘：`0096h:0084h`、面向、攻擊包裝 `0096h:006Bh`，彈藥傳 NULL（`0DD2h..0DD7h`）
```

## 出手與扣彈藥

`2C17h`（瞄準按 T）與 overlay-09 `0EB3h`（AI）挑彈藥的方式相同（exact）：

```
2D00  entry 43 → entry 45 取彈藥
2D1D  entry 44 而且距離 == 1 → 彈藥 = NULL      ; 匕首貼身是砍，不算丟
2D64  攻擊包裝 1883h(攻擊者, 目標, 0, 彈藥, &結果)
```

攻擊包裝 `1883h`（overlay-13 entry 15）打完之後（exact）：

```
1404h 的攻擊迴圈：第一形態每揮一下 `16BEh` inc [6D1Fh + 形態] → DS:6D20h
19E0  彈藥 +39h > 0 → +39h −= DS:6D20h
19F4  +39h == 0：
1A07    entry 44（攻擊者當下）且 +3Eh != 89h → GetMem(3Fh)、整筆複製、+34h = 0、插在
        戰利品串列 DS:676Eh 頭、DS:5CF8h 指它；再 entry 17 摘掉原件
1A7F    否則 entry 17（`156Ah`）摘掉並放掉
1A96  entry 7 重算
```

所以箭用完那一件就不見了，`+0F8h` 跟著變 NULL，下一次 entry 45 回 false。`+39h` 本來就是 0 的
單件（匕首）一丟就用完。remake：`gamepack.SpendAmmunition`、`resolveWeaponAttack`；落地的一件
記在 `tacticalState.ThrownLoot`，打贏時排在怪物物品的後面進戰利品選單（`collectMonsterLoot`）。

## `29h` 防護普通飛彈（overlay-12 entry 40 `108Dh`，exact）

群組 5 在近戰傷害骰算完之後對**目標**派發（overlay-13 `0223h..022Ch`，順序 `1Ch 29h …`）：

```
1028h(攻擊者 = DS:5CF0h)：沒有 +0CCh → NULL；entry 45 成立而且彈藥不是 NULL → 彈藥；否則武器
10B0  那一件 +32h != 0 → 不管                    ; 魔法飛彈不擋
0FC1  攻擊者 +0CCh 是 NULL → 不管
0FDB  距離（entry 33）<= 1 → 不管
0FEA  Roll(1, 100) > 64h → 不管                   ; 一定擲、一定成立
1004  印 "Avoids it"（`0FADh`）、DS:6776h = 0、DS:6780h = FFh、DS:6781h 減一
```

`DS:6781h` 是這一形態命中的次數（`16F1h` inc [6780h + 形態]），減一就是這一下不算命中。
remake：`gamepack.NormalMissileAvoided`，`resolveAttackSwings` 在群組 4／5 之後問它，擋掉的那一下
不扣血、不算命中，全部擋掉時印 `ui.statusAvoidsMissile`。

## AI 換武器（overlay-09 entry 9 `13D5h`，exact）

呼叫點：entry 1 `019Bh`（前面各支都沒動，接近之前）與 entry 5 `0DFCh`（挑到的目標搆得到，
但手上是射擊武器、不是 entry 44、身邊又有敵人 → 換完這一回合就結束）。

```
13DB..1458  +100h −= 型別表 +1（手數）of +0CCh 與 +0D0h（盾）
1471        射擊最佳分數 = 1；近戰最佳分數 = +0A3h × +0A5h（+0A7h > 0 再加它 × 2）；盾 = 0
14C6..1603  沿物品鏈：
              類別 0 而型別 +0Dh & 記錄 +0B0h：分數 = 1297h
                旗標 & 08h 或 & 10h：分數 > 射擊最佳 → 射擊候選
                旗標沒有 08h：分數 > 近戰最佳 → 近戰候選
              類別 1 而職業合得上：分數 = +32h ≥ 0 ? +32h + 1 : 0；> 盾最佳 → 盾候選
1606..16D2  射擊候選的 entry 44 與 entry 45（照它自己的旗標、記錄的 +0F8h／+0FCh）
16D6..171F  射擊分數 > 近戰分數 ÷ 2、有彈藥、而且（entry 44 或身邊沒有敵人）→ 挑射擊的；
            否則挑近戰的（可能是 NULL）
172A..17FE  目前的 +0CCh 就是它或被詛咒 → 不換；否則 Ready(目前的)、entry 7、
            +100h 再扣沒被詛咒的盾、Ready(挑中的)
1802..18F8  entry 7、overlay-13 entry 8；+100h > 2：盾沒被詛咒就 Ready(盾)，否則 Ready(挑中的)；
            +100h < 2 而盾要換：Ready(目前的盾)、entry 7、Ready(最好的盾)
18FE        entry 7
```

`1297h` 的分數（byte；型別表那一筆搬到 `[bp-12h]`）：骰數 × 骰面；`+32h > 0` 加 `+32h × 8`；
`+0Bh > 0` 加 `+0Bh × 2`；型別 55h 而目前目標 `+76h > 0` 改成 8；旗標 bit 3 加 `(+05h − 1) × 2`；
手數 ≤ 1 加 3；`+100h` + 手數 > 3 → 0；`+3Eh == 84h` 而 `+3Dh & 0Fh` 不是記錄 `+0A0h` → 0；
`+3Dh == 53h` → 0；被詛咒 → 0。

Ready 是 overlay-19 entry 7（`14CAh`，`00C9h:0043h`），**是切換**：穿著的叫一次就卸下（`14F3h`）。
所以「挑中的已經穿著、但目前 `+0CCh` 是另一件」時，這一支先卸掉目前的、再把挑中的也卸掉。
原版獸人頭目（MON2 block 15）同時穿著釘頭錘 0Ch 與短弓 2Bh，`+100h` = 3：

- 身邊有敵人：挑釘頭錘 → 卸弓 → 對已穿著的釘頭錘叫 Ready → 卸掉，徒手。
- 身邊沒人：弓就是挑中的、不換；`+100h` 3 > 2 又沒有盾 → 對弓叫 Ready → 卸弓，留釘頭錘。

block 14 的三隻（23h 與短弓都沒穿、箭穿著）：身邊有人穿上 23h（2d4），沒人穿上弓；沒有箭時
一律穿 23h。**這就是「箭用完之後」的行為**：下一個回合 entry 9 找不到能射的，換近戰武器
（沒有近戰武器時卸下弓，徒手）。

旁證（strong inference）：獸人家的 dosgolem 骰流（`dosgolem-deployment-peek-orc-home.json`）裡
有 `d20=18 d4=3 d4=4`、`d20=13 d4=1 d4=1` 這類 2d4 的傷害骰——整場只有 block 14 的 23h 是 2d4，
而那件在資料裡沒穿，只能是 entry 9 替牠穿上的；另有一次行動 `d20=3 d20=8 d6=3`（兩發、1d6），
是短弓一回合兩發。

remake：`gamepack.ChooseAIGear`（純規則，直接改物品的 `+34h`）、`cmd/pool-game/missile.go` 的
`foeChooseGear`（接在 foeTurn 的 foeCastPhase 之後；隊員的穿戴效果照 `wearItem` 掛上摘下）。

## 驗證

- gamepack：`TestMissileGearFollowsEntry45`（九種組合）、`TestVolleyLimitIsTheAmmunitionCount`、
  `TestSpendAmmunitionRemovesAndDrops`、`TestNormalMissileAvoidedOnlyStopsNonMagicalShotsFromAfar`、
  `TestChooseAIGearFollowsEntry9`（原版 MON2 block 14／15 的物品，六種情形）。
- 送鍵（`Update()`）：`TestPlayerArrowsAreSpentAndRunOut`——A、Enter 射一輪 3 支剩 1、再射一輪只射
  一發、箭那一件拿掉、再按沒有 Target 且回合不算用掉、拿弓撞上去是 "Not with that weapon"。
- foeTurn：`TestOrcLeaderSwitchesToMeleeWhenArrowsRunOut`（穿上弓射一發、箭用完、下一回合換 23h
  往前走）、`TestProtectionFromNormalMissilesStopsTheLeadersArrows`（`29h` 全擋、負對照會受傷）、
  `TestArmedOrcLeaderShootsFromOutsideMeleeReach`（改寫：7 號穿上弓原地射、6 號卸弓往前走）。
- tacticalInput（`ai_gear_recompute_test.go`，#109）：`TestAIGearRecomputeWashesTheStinkingCloudArmourClass`
  （沒有物品的獸人咳一次 AC 差 2，下一回合 entry 9 之後回到原值）、
  `TestAIDrivenNPCIsRecomputedAfterTheGearChoice`（交給電腦的 HERO 拿弓沒箭，entry 9 換回長劍，
  射程 1、1d10+2）。
- 變異：拿掉扣彈藥、瞄準閘門、`29h`、entry 9、封頂、撞上去的閘門、射擊次數，七個各自讓至少
  一條測試變紅；#109 的兩處（entry 9 沒有步驟就不重算、NPC 不重算）各自讓上面那兩條之一變紅。

## entry 9 結尾的重算（exact）

`176Eh`（`E9 91 00`）是「不換」那一支，也跳到 `1802h`；`1802h..1813h` 與 `18F8h..18FEh` 前面都沒有
條件跳躍：

```
1808  9A 43 00 0A 01   overlay-25 entry 7（重算）
1813  9A 48 00 96 00   overlay-13 entry 8（射擊次數，`0D29h`）
18FE  9A 43 00 0A 01   overlay-25 entry 7
```

所以 entry 9 **每跑一次就重算兩次**，換不換、身上有沒有物品都一樣。entry 7 對 AC 是整份重寫
（spec 147）：

```
0E43  26 8A 85 A9 00 / 26 88 85 11 01   +111h = +0A9h
0FAC  26 C6 85 11 01 00                 +111h = 0，再把四格累加器加回去
0FFB  26 88 85 12 01                    +112h = 結算值
```

臭雲的咳嗽（overlay-12 `0AD3h..0AEFh`，spec 121）改的是同一對 `+112h`／`+111h`，所以兩者的共存
就是**下一次 entry 7 把咳嗽扣的 AC 洗掉**。咳嗽那一回合 `0AC0h` 把 `+108h` 的 `+1` 清成 0、行動權
沒了，entry 1 不會走到 `019Bh`，那一回合 AC 是差的；之後電腦走到 entry 9（沒有先用物品或施法）
的那一回合，AC 回到物品算出來的值。玩家操作的人沒有 entry 9，咳嗽扣的 AC 留到戰鬥中換裝
（overlay-19 `146Fh`）為止。

remake：`foeChooseGear` 不論 `ChooseAIGear` 有沒有步驟、身上有沒有物品，最後都走
`storeCombatItems`（隊員 `applyPartyGearStats`、NPC `applyNPCGearStats`、怪物
`applyMonsterGearStats`）。remake 的盤面欄位裡只有臭雲的 AC 是戰鬥中直接改寫、又屬於 entry 7
重算範圍的（`cloud.go` 的 `StinkingCloudArmourClass`）；其餘效果在命中與傷害時才套。

## NPC 的重算（exact）

entry 7 的呼叫鏈沒有 NPC 分支（spec 147〈NPC〉）。會在戰鬥中改物品鏈、之後叫 entry 7 的三處：
扣彈藥之後的 `1A96h`（overlay-13）、entry 9 的 `1808h`／`18FEh`，以及物品選單的 `146Fh`。
remake 的 `storeCombatItems` 以前對 NPC 不重算（射完箭、AI 換了武器，射程與傷害骰還是舊的），
現在對 NPC 走 `applyNPCGearStats`，與物品選單同一支。

## 怪物換武器的穿戴效果（不接）

Ready 對 `+3Eh > 7Fh` 的物品呼叫 overlay-24 entry 1（spec 149），怪物也一樣（exact）。但原版資料裡
沒有 entry 9 會碰到的觸發者：MON1..8ITM 全部 301 件（`ReadDOSMonsterItems` 逐 block 掃），帶穿戴
效果的武器只有 HILL GIANT（MON2／MON5 block 55）的兩件型別 57h，效果碼 87h；盾（類別 1）一件
都沒有，其餘四件是類別 3／7／9，entry 9 不碰。87h 是 overlay-12 entry 124（`315Dh`：
`cmp byte es:[di+10h], 13h; jae`），力量小於 19 才卸下；HILL GIANT 的 `+10h` 是 19，常式什麼都
不做。所以接上與不接，任何一場戰鬥的結果都相同，列入停止線（`docs/audit/stop-line.md`）。

## `0E09h` 的條件寫回（卡住）

overlay-13 `0D29h` 結尾（exact）：

```
0D36  [bp-2] = +113h（舊的剩餘次數）；0D49 +113h = +0A1h
0DC8  [bp-1] = 新次數（0E58h）；射擊再以彈藥數封頂
0E09  26 C4 BD 08 01 / 26 80 7D 08 00   runtime +8 == 0 → 寫回 +113h = [bp-1]
0E18  新 < 舊 → 寫回
0E34  新 ≥ 舊 × 2 → 不寫（+113h 留著 +0A1h）
0E41  射擊（[bp-5] != 0）→ 不寫；否則寫回
```

runtime `+8` 只有 overlay-13 `1440h`（`26 C6 45 08 01`，攻擊核心 `1404h` 一開頭）寫 1、`002Eh`
（每回合初始化）寫 0，所以它的意思是「這一回合已經出過手」。這一段只在**同一回合出過手之後又
叫 entry 8** 時才有作用，而那要攻擊核心先把回合交還：`1755h` 目標 `+10Dh` 變 0 就停手，
`1799h..17C2h` 兩個形態任一格剩餘次數大於 0 就把「完成」旗標改回 0——殺了目標還有剩的次數，
這一回合繼續（exact，spec 052 第 9 條）。

remake 每一次攻擊之後一律結束回合（`endTurnAfterAction`、foeTurn 的 `endTurn`），沒有「剩的次數
續打」，`0E09h` 在 remake 裡沒有可以落地的狀態。接上它要先接續打那一條（玩家與電腦兩邊，
射擊的彈藥扣法也跟著），這不在 #109 的範圍；多次攻擊的角色殺了目標就少打剩下的幾下，是影響玩法
的缺口。

## 還沒接（DRAFT）

| 項目 | 等級 | 為什麼 |
|---|---|---|
| `0E09h` 的條件寫回 | exact（碼）| 卡在「殺了目標、還有剩的攻擊次數時回合繼續」（overlay-13 `1799h..17C2h`），remake 沒有這一條，見上一節 |

已移入停止線（`docs/audit/stop-line.md`）：Ready 失敗的訊息、彈道動畫（呈現）；反應攻擊帶不帶彈藥、
entry 33 走不到時的回值（未知）；怪物換武器的穿戴效果（原版資料沒有觸發者）。
