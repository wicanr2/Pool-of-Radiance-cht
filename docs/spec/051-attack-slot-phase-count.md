# Spec 051：Pool 雙攻擊槽與半回合攻擊次數

狀態：CONFORMED（兩槽 base source、phase counter 初始化／遞增、phase rounding
primitive、傷害骰的來源欄位、remake 的攻擊區段）；
DRAFT（裝備覆寫、effect code 12、玩家角色的攻擊次數來源與完整 initiative）。
日期：2026-09-04（原 2026-09-01）。

## 證據

- `overlay-13.bin` SHA-256：
  `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390`。
- IDA Pro 9.4、overlay-local file offset、base 0、16-bit metapc 匯出：
  `docs/audit/ida-overlay13-attack-slot-init.json`，保存 entry 1 `0000h`、local
  `0D29h` 與 `0E58h` 的原始 bytes／operands。
- `docs/audit/ida-overlay10-phase-init.json`：overlay-10 SHA-256
  `b929c7040aaa399ba803911c8630e06a8e21b9e7d64a67def69c700d7fe14439`，保存 combat
  setup `1ED6h..203Eh`。
- `docs/audit/ida-overlay08-phase-round-increment.json`：overlay-08 SHA-256
  `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f`，保存
  `0868h..0998h` 的回合邊界函式。
- Slums 真實 `MON2CHA.DAX` block 13／4：兩筆 `+A1h=2`、`+A2h=0`；這是兩筆
  ORC 的 raw anchor，不外推為所有同名 monster 的固定值。

## 原版資料流（exact）

1. entry 1 把 runtime attack selector 的目前槽設為 2，呼叫 `0D29h` 初始化第一槽，
   最後把結果寫 record runtime `+113h`。
2. `0D29h` 先把 record `+A1h` 複製到 `+113h`，再處理武器／effect 覆寫，最後呼叫
   `0E58h`。因覆寫 consumer 尚未閉合，`+A1h` 只能命名為 slot 1 base rate source，
   不能宣稱它永遠就是本回合攻擊數。
3. entry 1 另以 record `+A2h` 為 slot 2 base rate source，經 effect code `12h` 與
   `0E58h` 後寫 runtime `+114h`。
4. `0E58h` 取 source byte；若 `DS:6CD7h` bit 0 為 1 就先以 byte 加一，接著做 signed
   division by 2。對合法小值等價於：

   ```text
   attacks_this_phase = (encoded_rate + phase_bit) / 2
   ```

5. attack span `1678h..176Ah` 由 slot 2 倒數到 slot 1；每槽只有 runtime
   `record[112h+slot] > 0` 才進攻，進入後先減一，再呼叫命中／傷害鏈。因此
   `+113h/+114h` 是本 phase 兩槽的剩餘 attack count。
6. damage slot 是交錯陣列：slot `n` 使用 count `+114h+n`、sides `+116h+n`、
   signed bonus `+118h+n`；slot 1 即 `+115h/+117h/+119h`，slot 2 即
   `+116h/+118h/+11Ah`。**那是執行期的位置**，見下一節。

## 傷害骰的來源欄位是 `+0A2h/+0A4h/+0A6h`（2026-09-04）

`+114h..+11Ah` 是**執行期**的副本。overlay-25 `0DF4h` 在排怪時抄過去：

```
for i = 1; ; i++ {
    es:[record + 114h + i] = es:[record + 0A2h + i]   ; 顆數
    es:[record + 116h + i] = es:[record + 0A4h + i]   ; 面數
    es:[record + 118h + i] = es:[record + 0A6h + i]   ; 加值
    if i == 2 break
}
```

所以 `MONnCHA.DAX` 的**樣板記錄**要讀 `+0A2h/+0A4h/+0A6h`，不是 `+114h..`。
樣板裡那一段沒有初始化，殘留的是別的東西：172 筆有 **53 筆**與來源欄位不符，
而且殘留值看得出是文字——

| 記錄 | 來源欄位 | 執行期那一段 |
|---|---|---|
| `mon6/10` QUICKLINGS | 1d4 | `41 00 44 00 43 00` ＝ `'A' 'D' 'C'` → 65d68+67 |
| `mon6/118` THRI-KREEN | 1d4／1d4+1 | 加值 `66h` ＝ `'f'` → 1d4+102 |
| `mon2/13` ORC | 1d8 | 2d4−1 |
| `mon5/92` TYRANITHRAXUS | 1d6／4d6 | 2d4／`26h` |

**正負對照**：用來源欄位算，172 筆沒有一筆的骰子不合理（顆數 ≤ 8、面數是
D&D 的骰面、加值在 −4..+12）；用執行期那一段算，**48 處不合理**。
`TestMonsterDamageDiceAreAllPlausible` 同時量兩邊，負對照歸零就會紅——
那表示這一則不再證明任何事。

來源欄位與 AD&D 一版逐項相符，這是第二個獨立的核對：

| 怪物 | 次數編碼 | 形態 1 | 形態 2 | AD&D 一版 |
|---|---|---|---|---|
| ORC | 2／0 | 1d8 | — | 1 擊 1d8 |
| TROLL | 4／2 | 1d4+4 | 2d6 | 爪／爪／咬 |
| QUICKLINGS | 6／0 | 1d4 | — | 3 擊 1d4 |
| TYRANITHRAXUS | 4／2 | 1d6 | 4d6 | 兩爪一口 |
| POISONOUS FROG | 0／2 | — | 1d1 | 特殊攻擊在形態 1 |

## Phase counter 生命週期（exact）

1. overlay-10 combat setup `1F3Bh..1F45h` 依序清除 `DS:6D23h`、`DS:6CD7h` 與
   `DS:6780h`，之後才建立隊伍 combat runtime、呼叫 overlay-13 初始化 attack slots。
   因此新戰鬥的 phase counter 初值是 0。
2. overlay-08 `0868h` 函式在 `0879h` 對 `DS:6CD7h` 做 byte `inc`，緊接著呼叫
   overlay-13 entry 25，再走訪 `DS:5CF4h` 全 combatant 鏈並執行 effect code `13h`。
   同函式引用原始提示 `Your Teammate is Dying`／`Continue Battle:`。這支持它是戰鬥
   回合邊界；不需要替其他內部副作用命名，便可確定 attack phase counter 每次該邊界
   增加一。
3. counter 是 byte，`255 → 0` wrap。`0E58h` 只讀它的 bit 0，所以奇偶相位在 wrap 後
   仍連續交替。現行 remake 不允許戰鬥中途存檔，因此本輪不新增 save 欄位；日後若開放，
   phase counter 必須成為 combat continuation state，不能從畫面回推。

## Typed primitive

`AttacksThisPhase(encodedRate, phaseBit)`：

- `phaseBit` 只接受 0 或 1；其他值失敗即關閉；
- 依原版 byte 加法保留 wrap，再做 `/2`；
- 這支 primitive 不讀裝備、不執行 effect，也不自行推進 phase counter。

`AdvanceAttackPhase(counter)`：依原版 byte `inc` 回傳下一值，包含 `255 → 0` wrap。
combat setup 的初值固定為 0；只有戰術 runtime 抵達上述回合邊界才可呼叫，UI frame、
按鍵或單一角色行動都不能自行增加。

本規格只授權 rate rounding 與資料形狀。前端仍不得因算出 ORC slot 1 為一次攻擊，
就跳過 deployment、initiative、玩家輸入、AI、status 或勝敗 continuation。

## remake 的攻擊區段（2026-09-04）

`resolveTacticalAttack` 一次行動把這一相位的兩種形態都打完，順序由形態 2
倒數到形態 1（原版的攻擊區段 `1678h..176Ah` 就是這樣走的）：沒有骰子的形態
不揮，每一形態揮幾下由 `AttacksThisPhase(編碼, 相位 & 1)` 給。相位存在
`tacticalState.AttackPhase`，戰鬥開始是 0，`startRound` 從第二回合起每回合加一。

**玩家角色的攻擊次數還沒讀**（原版由職業等級表給），先用編碼 2＝一回合一次，
所以玩家這一側的行為與先前相同。

## 驗收

- 固定 encoded 0、1、2、3 在 phase 0／1 的相鄰值；另固定 255＋phase 1 的 byte wrap。
- 固定 phase counter `0→1→2` 與 `255→0`；以測試明示一次回合邊界只增加一次。
- 真實兩筆 ORC 的 base sources 固定為 slot 1=`2`、slot 2=`0`；在任一 phase 均得到
  一次 primary、零次 secondary，僅作這兩筆模板抽樣。
- Pool 全套 `go test ./...`、`go vet ./...`。
