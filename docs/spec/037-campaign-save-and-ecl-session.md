# Spec 037：戰役存檔、地圖位置與 ECL session 續點

狀態：CONFORMED（穩定玩家邊界）；DRAFT（任意動畫 frame、尚未完成的戰鬥與其他未實作
外部服務中途續點）。日期：2026-09-01。

## 原版需求證據

- `ECL3.DAX` block 8 的墓園委託於 `A592h..A5BEh` 讀取 `4AC1h`、`4AB1h`、
  `4A96h`，並以 `A5A8h PARTYSTRENGTH` 寫入 `6E79h`；條件成立後才顯示墓園內容。
- 接受委託後同一 VM 依序執行 `A77Fh CLEARMONSTERS`、`A780h TREASURE`、
  `A791h COMBAT`，離開戰利品服務後還要從 `A792h` 繼續寫狀態。
- 因此角色 roster 不能代表戰役進度；至少還需要地圖 identity／座標／朝向、current
  ECL block、PC、stack、不透明 numeric／string memory、比較結果、亂數續點與 pending
  lifecycle entries。位址與控制流為 `exact`；這份 JSON schema 是 remake 契約，
  不冒充 DOS 存檔逐 byte 格式。

## 現行缺陷

schema 3 只有 pooled gold、character library、party 與 inventory。F10 寫出的檔案沒有
`spawn` 或 ECL session；主選單 `L` 也只載入 roster，按 `B` 會從 Rolf 開場重建全新 VM。
所以任何跨事件旗標都會遺失，不能支持正常墓園委託或完整主線。

## Schema 4 契約

1. `campaign` 可為空（尚未開始冒險）；存在時保存 GEO archive／block、`x/y/facing`，
   以及共用引擎 `BlockSessionSnapshot`。
2. `x/y` 僅接受 0..15；朝向僅接受 `0/2/4/6`；archive／block 必須由載入時的 typed
   catalog 再驗證，不能只信 JSON。
3. schema 3 確定性遷移成 `campaign=nil`；schema 2／1 依既有鏈遷移後同樣沒有戰役。
4. F10 在穩定玩家邊界先從 live app 建立 campaign snapshot，再原子寫入；快照失敗不得
   離開遊戲。載入時先以版本化 game pack 建立 session 並 restore，全部成功後才替換
   live app，避免半套狀態。
5. 主選單 Load 必須恢復 adventure，而不是只載入 roster 後等待 Begin 重開。

## 驗收

- schema 4 JSON round trip 保存 `(archive,block,x,y,facing)`、current ECL block、墓園三個
  旗標及亂數續點；schema 1..3 遷移維持既有角色／HP／inventory。
- 在正常按鍵可達的穩定 cell boundary 以 F10 保存，再由新 app 的 Load 恢復同一位置與
  ECL memory；繼續輸入不得重播 Rolf 或重置 City Hall。
- malformed map、朝向、session snapshot、缺 block 或 code-window memory 注入均失敗
  即關閉，且不覆蓋載入前狀態。

現行驗收由 `internal/save` schema 1..4 migration／round-trip 測試、共用 engine
`eclvm` snapshot 正負對照，以及 `TestF10AndLoadRoundTripStableCampaignSession` 的兩個
獨立 app 實例完成。Pool 正式鎖版全測試與 CoAB 唯讀 `internal/ecl`、`internal/game`、
`cmd/azure-bonds-game` 相容回歸均通過。
