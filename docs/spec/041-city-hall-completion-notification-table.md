# Spec 041：City Hall 完成通知狀態表

狀態：READY（結構清冊與 `4AC1h` 增量）；DRAFT（26 槽完整劇情語意與正常完成路徑）。
日期：2026-09-01。

## 問題與勘誤

`4AC1h` 不是單一委託旗標。ECL3/block8 的 `9D0Eh..9DBEh` 會逐槽掃描
`4AA6h..4ABFh`：值為 `FEh` 的槽才由 `9D63h ON GOSUB` 顯示一次完成通知，之後由
`9F5Ah SAVE TABLE FFh,4AA6h,[6E79h]` 將該槽 acknowledge。26 個通知分支中只有十個
執行 `ADD 1,[4AC1h] → [4AC1h]`。因此「十個 producer」是十類會提高公告進度的完成
通知，不是十個獨立儲存欄位，也不存在 clerk 單點授予 `4AC1h` 的 producer。

## 可重生證據

- 原始控制流：`docs/audit/dos-ecl3-block8-trace.json`，ECL3/block8 SHA-256
  `fd446439973994d9f887b8628b512054a822369eb4360769b21eb2224982f012`，位址基準
  `9900h`。
- 全 ECL 直接引用：`docs/audit/dos-city-hall-reward-state-refs.json`，由
  `cmd/pool-ecl-memory-audit -addresses 4AA6,...,4ABF` 重生。三個尚未完整解碼的 block
  仍列為 failure，因此這是直接 operand 引用的下界，不能用零列宣稱沒有間接 producer。
- 結構矩陣：`cmd/pool-city-hall-audit` 必須從 `9D63h` 的 26 條有序 edge 產生每槽
  address、target、第一段玩家文字及是否增加 `4AC1h`；刪除任一 edge 或 producer
  都必須使測試失敗。

## READY 契約

1. 狀態槽 `i` 對應 `4AA6h+i`，合法索引為 0..25；順序只能取自原始 ON GOSUB edge。
2. 掃描時只有值 `FEh` 會進通知；通知完成後該槽改為 `FFh`，避免重複演出。
3. `4AC1h` 只在矩陣標示的十槽各增加一，不能按所有 26 槽增加，也不能自行 cap 在 9。
4. 通知文字與增量只描述 clerk 的結算行為；各槽在其他 ECL 的真正完成條件仍須逐條
   追到 producer，未閉合者維持 raw address，不把文字標題冒充欄位名稱。

## 後續玩家路徑閘門

優先追正常早期可完成的 Slums、Sokal Keep、Norris／Podal 等 producer，建立至少四條
「地圖事件 → `4AA6h+i=FEh` → City Hall 通知 → `4AC1h` 增量 → 存檔」垂直鏈；達成前
不得用測試直接注入 `4AC1h=4` 宣稱墓園委託已正常解鎖。
