# Spec 026：City Hall clerk office 正常入口

狀態：CONFORMED；日期：2026-09-01。

## 範圍與證據

本切片把正常玩家從 City Hall 門口 `(4,4)` 接到 clerk office 的外部提示與第一頁
clerk 對話；reward／commission 迴圈仍另行閉合。

固定 DOS ZIP、`ECL3.DAX`、GEO3/block0、IDA 工具與位址空間沿用 Spec 009、023、025。
`ECL3/block8` SHA-256 為
`fd446439973994d9f887b8628b512054a822369eb4360769b21eb2224982f012`。

## 原版 dispatcher 與座標（exact）

block 8 entry 1 `9998h` 將目前 terrain 與 `7Fh` AND 後，以 26 為基底；
`9A20h` 減 26，`9A29h ON GOSUB` 的前四個目標依序為：

- terrain 26 → `9A4Fh`
- terrain 27 → `9A52h`
- terrain 28 → `9B38h`（clerk office 外）
- terrain 29 → `9B91h`（進入 clerk office）

GEO3/block0 的實際格：

- `(4,5)` raw terrain `9Ch`，遮罩後 28；
- `(5,5)` raw terrain `9Dh`，遮罩後 29。

`9B38h` 顯示：

`YOU ARE OUTSIDE THE CLERK'S OFFICE. GUARDS POSTED AROUND A DOOR IN THE SOUTH WALL WATCH YOU CLOSELY.`

接著令 `4A06h=0` 並 RETURN。`9B91h` 在 `4A06h=0` 時通過入口 gate，於
`9BACh/9BB2h` 令 `4A01h=1`、`4A06h=1`，再由 `9BB8h` 顯示：

`AT YOUR ENTRY, THE COUNCIL CLERK BEGINS LOOKING THROUGH A STACK OF PAPERS.  'BEFORE I CAN OFFER ANY COMMISSIONS, I MUST SEE IF YOU ARE DUE A CURRENT REWARD.`

## 驗收

- 從標題／建隊後的同一正常 session，走完 Rolf、Sune、City Hall 公告與
  Spec 025 進門，不使用 direct-entry 或座標注入。
- `(4,4,2)` 轉向南、前進到 `(4,5,4)`，必須看到 clerk office 外部文字。
- Return 後轉向東、前進到 `(5,5,2)`，必須看到 clerk 第一頁，且 VM memory
  `4A01h=1`、`4A06h=1`。
- script block 保持 8、GEO identity 保持 `(3,0)`；文字後續 reward／commission
  尚未 READY 時可停在玩家 boundary，不得用手寫服務畫面跳過原始 ECL。

## Conformance

- `TestRealInitialAdventureUsesSharedVMToRolfExit` 從標題、原版建角、Rolf、Sune、
  City Hall 公告與 `NEWECL 8` 正常走到 `(4,5)`／`(5,5)`，沒有 direct-entry 或座標注入。
- 測試逐項斷言外部提示、clerk 第一頁、`4A01h=1`、`4A06h=1`、script block 8 與
  GEO `(3,0)`；2026-09-01 Docker/Xvfb 目標測試通過。
- 第一頁後穿過 `SAVE TABLE` 到下一個 commission 文字 boundary 由 Spec 027 接手；
  完整 reward／commission 迴圈仍須另立規格，本規格沒有擴張其完成聲明。
