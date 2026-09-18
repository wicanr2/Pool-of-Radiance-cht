# Spec 034：戰後戰利品選單與物品鏈移除邊界

狀態：READY（選單入口、物品鏈巡覽／移除與 63-byte 釋放）；DRAFT（角色接收檢查、
容量／重量、金錢分配、卷軸與裝備 mutation）。
日期：2026-09-01。

## 證據

- `overlay-05.bin` SHA-256：
  `900ea1b8e57b03e0f6dd1c16024a686c8474e2b0ae9674c730d5619679809c16`。
  `docs/audit/ida-overlay05-postcombat.json` 由 IDA Pro 9.4 對十二個 TPOV entry seed
  非破壞性匯出；位址空間是 overlay-local file offset、base 0，保留 bytes 與原名稱。
- `14CAh` 是戰後主流程：先呼叫 `04ADh` 準備狀態，再呼叫 `08E0h` 顯示
  `The party has found treasure!` 等結果，接著 `0E85h` 進入戰利品選單；結束後
  `1510h..1554h` 逐節點沿 `+2Ah` next 釋放，每節點固定 `3Fh` bytes。
- `0E85h` 依是否有 money／item 組成 `View Take Pool Share`、`View Take Pool` 等選項；
  `0FF6h..1008h` 的 `T` 分支呼叫 `0CF0h`。
- `0CF0h` 顯示 `Take: Money Items Exit`；`I` 分支呼叫 `0BCAh`。`0BCAh` 從
  `DS:676Eh` 鏈頭開始，選定節點後由前驅或鏈頭移除，以 `+2Ah/+2Ch` 更新 next，將
  被移除節點 next 清零，再以 `FreeMem(3Fh)` 釋放。

上述控制流與資料結構為 `exact`。`0C18h` 呼叫的角色接收 helper 尚未閉合；只有 helper
回報成功後才移除節點。因此 remake 在解出該 helper 前不得先刪 loot、也不得假定任何
角色都能無條件取得物品。

## 原版版面（dosgolem 實測，2026-09-18）

收據 [`docs/audit/dos-treasure-screens.json`](../audit/dos-treasure-screens.json)（六張畫面的
色號陣列與 SHA-256、鍵序、狀態檔雜湊；產生器 `tools/dosgolem-treasure-screens.py`）：

| 畫面 | 底列 | 內容 |
|---|---|---|
| 頂層 | `VIEW TAKE POOL SHARE EXIT` | 左邊圖框、右邊隊伍欄（`NAME AC HP`） |
| `TAKE` | `SELECT TYPE OF COIN EXIT` | 逐行列出每一種幣與數量：`GOLD 250`／`PLATINUM 50`／`JEWELRY 1` |
| `VIEW` | `VIEW:TRADE DROP EXIT` | **人物頁**（不是幣別清單，見 spec 040） |
| `SHARE` 之後的頂層 | `VIEW POOL EXIT` | 錢分完，`TAKE`／`SHARE` 消失 |

remake 照這個版面補齊（#47）：頂層與 `Take` 看得到每一種幣與數量、`View` 走既有的人物資料頁、
`Pool`／`Share` 之後把「誰拿到多少、pool 剩多少」寫出來、選項依有沒有錢與物品組成。
畫面識別字 `treasure`、`treasure-take`、`treasure-take-money`、`treasure-take-who`、
`treasure-take-amount`、`treasure-items`、`treasure-items-who`、`treasure-confirm-exit`。

## 後續實作閘門

1. 追 `0C18h` 的 far-call target，固定接收成功／失敗條件及玩家可見訊息。
2. 將 Spec 033 的 raw record 放入 pending treasure chain；UI 至少保留 View／Take／Exit
   與五筆原順序。
3. 角色 inventory schema、容量／重量及存檔 mutation 必須先各有 READY 欄位契約；
   失敗時 loot 留在 pending chain，成功時才 exactly-once 移除。
