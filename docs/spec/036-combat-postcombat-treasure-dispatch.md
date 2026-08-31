# Spec 036：`24h COMBAT` 的戰鬥／神殿／戰後服務分派

狀態：CONFORMED（墓園 pending treasure → overlay-05 戰後服務與 continuation）；
DRAFT（有怪戰鬥、決鬥、其餘服務旗標與完整勝敗結果）。
日期：2026-09-01。

## Pool 原版 dispatcher

輸入與位址見 `docs/audit/ida-overlay03-op0a-op20-op24.json`：`overlay-03.bin`
SHA-256 `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`，
IDA Pro 9.4，overlay-local offset、base 0。

`24h` handler 是 `186Ch..19C9h`：

1. `1876h/187Dh` 先檢查兩個 encounter 狀態 bytes `82ADh/829Ah`；任一非零走
   `18C8h` 的戰鬥路徑。本輪保留 raw address，不臆測兩者各自名稱。
2. 兩者皆零時，`1884h..18BEh` 依序檢查 game-state `+6D8h == 1` 與 `+5C4h == 1`，
   分派另外兩個服務並清旗標。
3. 三者皆不成立時，`18C0h` bytes `9A 25 00 3B 00` 呼叫 runtime `003Bh:0025h`。
   START.EXE MZ header `3B0h` 與 TPOV control table將 segment `3Bh` 精確映射到
   overlay-05；stub `25h` 是 entry 1、code `14CAh`，即 Spec 034 的 post-combat
   treasure 主流程。

因此 `CLEARMONSTERS → TREASURE → COMBAT` 在沒有 encounter／服務旗標的墓園路徑是
「開啟戰後戰利品服務」，不是零隻怪的假戰鬥。此結論為 `exact`。

## Remake continuation

- shared VM 的 `27h` inline request 與隨後 `24h` external event 必須同一 result 保序。
- Pool adapter 看到 pending `TreasureRequests` 時先建立 Spec 033/035 的 View／Take／Exit
  服務，不啟動 combat renderer。
- 玩家離開服務後，從 VM 已停在 `24h` 後的 PC 繼續；不得重播 TREASURE、重複加入
  inventory 或再次進服務。仍有 loot 時先依原版提示確認離開。
- 沒有 pending treasure 的 `24h` 不能套用本規格，保持失敗即關閉，直到有怪戰鬥或
  其他 raw flags 各自 READY。

## 驗收

- `TestRealGraveyardTreasureBytesEnterFiveItemService` 直接從原始 ZIP 讀取
  `ITEM3.DAX/33h`，以真實 block 8 `A780h` 起始位元組執行 `TREASURE → COMBAT`；
  production adapter 進入 Take 服務並依原始順序列出五件物品，第五件為
  `Two-Handed Sword +1 +3 vs. Undead`。
- 單元測試另覆蓋成功 Take、`OverLoaded` 不移除物品，以及存檔失敗時角色 inventory
  與 pending loot 兩側都回滾。
