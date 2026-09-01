# Spec 051：Pool 雙攻擊槽與半回合攻擊次數

狀態：CONFORMED（兩槽 base source、phase counter 初始化／遞增、phase rounding primitive）；
DRAFT（裝備覆寫、effect code 12、攻擊槽選擇與完整 initiative）。
日期：2026-09-01。

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
   `+116h/+118h/+11Ah`。

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

## 驗收

- 固定 encoded 0、1、2、3 在 phase 0／1 的相鄰值；另固定 255＋phase 1 的 byte wrap。
- 固定 phase counter `0→1→2` 與 `255→0`；以測試明示一次回合邊界只增加一次。
- 真實兩筆 ORC 的 base sources 固定為 slot 1=`2`、slot 2=`0`；在任一 phase 均得到
  一次 primary、零次 secondary，僅作這兩筆模板抽樣。
- Pool 全套 `go test ./...`、`go vet ./...`。
