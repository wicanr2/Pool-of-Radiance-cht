# Spec 014：第一張地圖基本 GEO walk

狀態：PARTIAL（位置、轉向、cardinal forward 已實作；事件與門互動 DRAFT）
日期：2026-08-31

## 已證實輸入

- 正常 Rolf ECL `EXIT` 的最後位置為 GEO3/block 0 `(0,4)`、facing `3`（Spec 011／013）。
- overlay-07 entry 27 對 cardinal facing `0/2/4/6` 做 16×16 coordinate wrap；奇數 facing
  不改座標（Spec 012）。
- 共用 engine `geometry` 的 GEO wall／detail decoder 已由 Pool 29／29 真實 blocks
  驗證結構相容；`CanMoveDungeonWrapped` 的 door detail consumer 來自 CoAB 原版證據，
  在 Pool 尚屬跨作品 strong inference。

真實 GEO3/block 0 在 `(0,4)` 的抽樣：N、E、W 可通，S 有 wall 4／detail 0 而阻擋。
這是原始 Pool GEO data 的 exact 結果，不是手寫地圖。

## 現行玩家行為

- 左／右方向鍵以 `-1/+1 mod 8` 改 facing，允許八方向觀看。
- 上方向鍵只在 cardinal facing 前進；奇數 facing 顯示引導，不猜斜向位移。
- cardinal forward 由 `CanMoveDungeonWrapped` 判斷，通過後以 16×16 wrap 提交座標；
  wall／locked door 阻擋時不改座標。
- 每格 ECL dispatch、門的 Bash／Pick／Knock、搜尋、時間與 encounter 尚未接，UI 明示
  `GEO WALK: ENABLED / EVENTS PENDING`。

Docker／Xvfb 測試從正式 Begin 跑完 VM Rolf 34 frame，於 `(0,4,3)` 左轉到 facing 2，
再以真實 GEO 前進到 `(1,4,2)`。此項只證明第一個正常可操作 movement slice，不支持
「第一張地圖事件完整」或「Pool 自由移動 parity 完成」。
