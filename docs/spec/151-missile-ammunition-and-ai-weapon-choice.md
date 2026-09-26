# Spec 151：射擊、彈藥與 AI 換武器（overlay-25 entry 43／44／45、overlay-13 射擊路徑、overlay-09 entry 9）

狀態：READY（彈藥查詢、射擊次數、瞄準與撞上去的閘門、扣彈藥與落地、`29h` 防護普通飛彈、
AI 換武器的位元組都已逐段讀出並實作，送鍵與 foeTurn 測試加變異檢查）；DRAFT（下面〈還沒接〉
那幾條）。沒有原版執行期收據逐發對過彈藥數，證據是位元組（exact）加上獸人家骰流的旁證。
日期：2026-09-26。主台帳：GitHub issue #98、#106（`29h` 那一條）。

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
- 變異：拿掉扣彈藥、瞄準閘門、`29h`、entry 9、封頂、撞上去的閘門、射擊次數，七個各自讓至少
  一條測試變紅。

## 還沒接（DRAFT）

| 項目 | 等級 | 為什麼 |
|---|---|---|
| `0E09h` 的條件寫回（`+108h` 的 `+8` 立著、而且新次數沒有變少時才寫） | exact（碼）| remake 每次出手才算次數，沒有 `+113h` 那一格；影響的是同一回合中途換武器的次數 |
| AI 換武器時 Ready 失敗的訊息（"already using"、"Your hands are full!"） | exact（碼）| `16F3h` 對電腦接手的有另一條路，沒讀；remake 不印 |
| 怪物換武器時的穿戴效果（`+3Eh > 7Fh`） | exact（碼）| 怪物的效果串列走 overlay-24 entry 1，remake 只接了隊員那一側 |
| NPC 換武器、扣彈藥之後的重算 | exact（`1380h` 對每一筆）| NPC 的戰鬥數值仍讀它帶的記錄（#97 的範圍）；物品鏈照原版改 |
| 反應攻擊（overlay-13 entry 6）帶不帶彈藥 | unknown | 沒讀；remake 的反應攻擊不扣彈藥 |
| `1883h` 的彈道動畫（`268Eh`）| exact（碼）| 畫面，不影響規則 |
| 距離走不到（`TraceMovement` 不完整）時 entry 33 回什麼 | unknown | remake 當 0（`29h` 不擋）|
