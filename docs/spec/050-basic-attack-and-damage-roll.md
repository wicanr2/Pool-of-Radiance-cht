# Spec 050：Pool 基礎命中與傷害骰 primitive

狀態：CONFORMED（d20 的 1 miss／20→100 score、encoded THAC0／AC 比較、NdS＋signed bonus、零下限）；
DRAFT（狀態／法術／距離 modifier、initiative、武器選擇、攻擊次數來源、backstab 判定、
特殊攻擊、受傷／死亡狀態與完整戰術回合）。
日期：2026-09-01。

## 輸入、工具與位址空間

- DOS `GAME.OVR`：SHA-256
  `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638`。
- `overlay-13.bin`：SHA-256
  `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390`。
  `docs/audit/ida-overlay13-attack-span.json` 保存 IDA Pro 9.4 對
  `1404h..1883h` 的逐指令匯出。
- `overlay-24.bin`：SHA-256
  `e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714`。
  `docs/audit/ida-overlay24-hit-and-dice.json` 保存 entry 6／9；
  `docs/audit/ida-overlay24-dice-core.json` 保存 local `0DE5h`。
- 位址全部是 **overlay-local file offset、base 0、16-bit metapc**。TPOV manifest
  證明 runtime segment `0096h` 對應 overlay 13、`0100h` 對應 overlay 24；far-call
  stub offset 依 control record 反查 entry，不以 overlay index 猜 runtime segment。

## 原版呼叫鏈（exact）

1. overlay-13 attack span `1617h` 先取目標 record `+111h` 的 internal AC，交由
   `1B15h` 套用尚未閉合的情境 modifier。
2. `16C2h..16D7h` 把 attacker、target 與 effective internal AC 傳給
   runtime `0100h:003Eh`，即 overlay-24 entry 6 `0CB5h..0D5Eh`。
3. entry 6 呼叫 local `0DE5h` 擲 `1d20`：
   - roll `1` 直接 miss；
   - roll `20` 先改成 score `100`，再進入後續比較；正常資料下近似必 hit，但原始指令
     沒有直接 return，故契約不得寫成無條件命中；
   - 其他結果在套用尚未閉合的 attacker／target effects 與一個已讀出的 byte modifier
     後，比較 `roll + attacker.Raw[110h] + modifier >= effectiveACInternal`。
4. 命中才由 overlay-13 `1713h` 呼叫 local `0191h`。該函式以 attack slot index 讀
   `+114h/+116h/+118h`；slot 1 即 Spec 049 已閉合的 `+115h/+117h/+119h`，依序為
   dice count、die sides、signed bonus。
5. `0191h` 呼叫 overlay-24 entry 9 `0E30h`，再進 local dice core `0DE5h`。
   dice core 對每顆骰呼叫 resident RNG，取得 `0..sides-1` 後加一，以 byte accumulator
   累加 count 次。caller 再加 signed bonus，負值 clamp 為 0。
6. `0191h:01DEh..020Dh` 若 backstab predicate 成立，才把已 clamp 的傷害乘以
   `attacker level byte / 4 + 2`。**level byte 是 `+9Ch`，也就是賊等級**
   （`01F5h` 直接讀它；spec 095 的 TINA 賊等級 9 就是這一格）——所以倍數是
   AD&D 的背刺級距：1–4 級 ×2、5–8 級 ×3、9–12 級 ×4。predicate 仍未閉合
   （overlay-13 的近呼叫 `2558h`），本規格只允許 caller 明確傳入 multiplier；
   一般攻擊固定為 1，不在 primitive 裡自行猜 backstab。
7. `01CFh` 把骰出來的傷害寫進 `DS:6776h`，倍數乘完之後 `021Eh` 呼叫
   **效果群組 4**（`1Dh 03h 06h`）讓效果再調整一次
   （[spec 112](112-effect-code-dispatch.md)）。`0210h` 先把傷害訊息旗標
   `DS:6777h` 清成 0。

### 2026-09-01 勘誤

第一版把 `0CE4h` 的 `roll 20 → 100` 簡寫成「自然 20 無條件命中」。重新沿
`0CE9h..0D52h` 讀到底後確認它仍會經過 effect modifier 與最終 compare，沒有直接
return；因此現行契約是「20 改成 comparison score 100」。舊結論形成原因是只看
特殊值寫入而未把同一控制流追到唯一回傳點，現以極端 synthetic AC 負對照防止復發。

## Typed 契約

### 命中

輸入：

- `roll`：已完成原版 `1d20` 的結果，必須在 1..20；
- `attackerTHAC0Internal`：record raw `+110h`；
- `effectiveACInternal`：目標 raw `+111h` 經 caller 已證實 modifier 後的值；
- `modifier`：原版 effects／全域 byte modifier 的**已合計 signed 值**。本輪不解析來源。

輸出：

- roll 1 → miss；roll 20 先把 comparison score 改成 100；
- 其餘 roll 的 comparison score 就是 roll；最後一律比較
  `score + attackerTHAC0Internal + modifier >= effectiveACInternal`。

非法 roll 或超出 signed intermediate 安全範圍必須回錯，不能 clamp 或擲新骰。這個
primitive 接受 internal encoding，避免先轉成畫面 THAC0／AC 再失去原版比較形狀。

### 傷害

輸入：`count`、`sides`、signed `bonus`、每顆已擲出的 1..sides 結果，以及
`multiplier >= 1`。count／roll 數必須相等，count 與 sides 不得為零。

輸出：`max(0, sum(rolls)+bonus) * multiplier`。非法骰值、長度或 multiplier 失敗即
關閉。原版 byte accumulator 的 wrap 尚未授權 remake 接受不合理超大資料；真實 corpus
的 attack dice 形狀應另作全掃後再決定上限。

## 明確排除

- `1B15h` 的 effective AC modifier 細目；
- `DS:4937h +6E0h/+6E2h` 的來源。（`02E2h` 的 effect codes `0Ah` 與 `10h`
  已經解出來了：`0Ah` 是**攻擊者**身上的命中修正
  `01h 02h 21h 24h 31h 03h 06h 12h 1Ah`，`10h` 是**目標**身上的
  `19h 47h 25h 2Fh 30h 59h`；擲出來的 d20 放在 `DS:6780h`，自然 1 直接失手、
  自然 20 改寫成 100，然後才讓效果調整。見 [spec 112](112-effect-code-dispatch.md)。）
- attack slot `+112h` 次數如何生成、武器如何覆寫 `+115h..+119h`；
- initiative、移動、目標選擇、AI、特殊攻擊、status transition、勝敗與 ECL continuation。

因此本規格不能授權前端自動戰鬥或 forced-win；它只關閉可被後續戰術 runtime 使用的
兩個純規則 primitive。

## 驗收

- 命中測試固定 roll 1、roll 20→100 score、一般 roll 的差一 miss／相等 hit，以及
  正負 modifier；另以極端 synthetic AC 證明 20 並非函式層無條件 return。
- 傷害測試固定 `1d8`、`2d4-1`、負值歸零、倍率、錯誤骰值與長度。
- 真實 Spec 049 兩筆 ORC typed damage 可直接餵入 primitive，但不以 remake 自測冒充
  DOS 同 seed 戰鬥對拍。
- Pool 全套 `go test ./...`、`go vet ./...`；前端 staging 仍不得越過 `9E6Dh`。
