# Spec 042：DOS ECL archive catalog 與 Slums 完成計數

狀態：READY（archive catalog、ECL2/block20 計數 helper）；DRAFT（正常 Slums 地圖全事件與 City Hall 往返）。
日期：2026-09-01。

## 原始證據

- `docs/audit/dos-ecl2-block20-trace.json` 固定 `ECL2.DAX` block 20 SHA-256
  `a1e1dd7a7ffc07f6e3c3ce7fad25d373270c10edd06ec3fa1f1d0edc9a7297ec`、五個 entry 與
  `9900h` 位址基準。
- `9A8Bh LOAD FILES 20,2,FFh` 與 `9AB8h` 的原始文字共同證明這個 block 是玩家進入
  Phlan Slums 的腳本；這項證據只命名 script，不自行命名 GEO block。
- `B69Ch..B6BBh` 是被多個戰鬥後路徑呼叫的共用 helper：若 `[4ABBh] >= FEh` 立即返回；
  否則加一，未達 25 返回，達 25 時寫 `FEh`。City Hall 的 Spec 041 槽 21 隨後顯示
  `YOUR CLEARING OF THE SLUM AREAS PERMITS US TO EXPAND.`、增加 `4AC1h`，再 acknowledge
  為 `FFh`。
- 同一份 trace 內對 `B69Ch` 的呼叫共 **14 處**，全為 `GOSUB`，位址各自唯一：
  `9E73`、`9F08`、`A0BE`、`A3B2`、`A514`、`AA02`、`AA87`、`AB91`、`ABFE`、`AC4B`、
  `AF7A`、`B10D`、`B134`、`B14F`。**14 個固定呼叫點湊不滿 25 次門檻**，因此完成
  Slums 必須包含可重複觸發的遭遇；這與《軟體世界》創刊號攻略「約 15 群隨機出現的
  怪物」一致（見
  [`docs/reference/walkthrough-notes-softworld-001.md`](../reference/walkthrough-notes-softworld-001.md)）。
  呼叫點數與門檻是 `exact`；「差額由隨機遭遇補足」目前是 `strong inference`，尚未把
  每個呼叫點對回具體事件。

## READY 契約

1. game pack 必須能依 DOS archive 1..8 載入完整 ECL DAX，保留原始 block ID，不把
   不同 archive 的相同 block ID 合併。
2. 缺 archive、重複 archive member、重複 block ID、非法 archive 編號或無法解析的
   DAX 必須失敗即關閉。
3. ECL2/block20 的真 bytes 測試必須從 0 重播 helper 25 次：第 24 次為 24，第 25 次
   精確變 `FEh`；初值 `FEh` 再執行不得改變。
4. direct-entry 只證明 helper 規則，不算正常玩家完成 Slums。完成聲明仍需由地圖入口、
   25 個實際完成呼叫、City Hall 通知、`4AC1h` 增量及存檔串成正常垂直鏈。
