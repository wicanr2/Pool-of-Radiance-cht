# Spec 025：`NEWECL` 後的 command-set lifecycle

狀態：CONFORMED；日期：2026-09-01。

## 勘誤

Spec 019 把 Pool `NEWECL` 成功交接簡化為「目的 block entry 0 執行後即完成」。
這只足以讓初始地圖 sweep 不再報 unknown opcode，並不等於原版完整 lifecycle。
City Hall 正常玩家路徑已提供反例：`ECL3/block0 AD7Fh → block 8` 後，現行 session
停在目的 entry 0 的 `EXIT`，沒有執行目的 entry 4 `9A06h LOAD FILES 0,0,0`。

## 固定證據

- DOS ZIP、`overlay-03.bin`、IDA Pro 9.4 image、16-bit overlay-local 位址空間與雜湊
  同 Spec 019／023。
- `docs/audit/ida-overlay03-lifecycle-controller.json` 保留原始 bytes 與位址。
- `ECL3/block8` SHA-256：
  `fd446439973994d9f887b8628b512054a822369eb4360769b21eb2224982f012`；五個 entry 為
  `9914h, 9998h, 99D3h, 99FCh, 9A06h`。

## 原版順序（exact）

1. `NEWECL` handler `0CDDh..0D73h` 切換 block，於 `0D66h/0D6Bh` 同時令
   `DS:4390h=1`、`DS:4391h=1`。
2. interpreter controller `35DBh..361Ch` 因 `4390h` 返回。
3. lifecycle controller `3620h..377Fh` 在 `3741h..3746h` 呼叫 `DS:4944h`
   （command-set entry 0）。
4. 因 `4391h!=0`，`3749h..3778h` 不呼叫 entry 1，而是跳回 `3626h`。
5. `3658h..365Dh` 呼叫 `DS:494Ch`（command-set entry 4）。

所以 Pool 的 block transition sequence 是 entry `0 → 4`，兩者間保留同一 VM memory、
strings、RNG 與新 block identity。entry 0 若產生玩家 boundary，續行到其 `EXIT` 後仍須
接 entry 4，不能因暫停而遺失 pending lifecycle。

## City Hall 資源 identity

block 8 entry 4 從 `9A06h` 執行 `LOAD FILES 0,0,0`，再於 `9A0Dh` 執行
`LOAD PIECES 127,127,127`。依 Spec 009 已閉合的 Pool handler，第一個 LOAD FILES
operand 0 在 archive `DS:52D4=3` 下是 `GEO3.DAX block 0`。因此：

- `NEWECL 8` 是 ECL script identity，不是 GEO block identity。
- City Hall 仍使用 `GEO3/block0`；不得把前端地圖誤切為 GEO block 8。
- 正常門口動作從 `(3,4,facing 2)` 進到 `(4,4,facing 2)`，但 script block 變為 8。

## Typed 行為與驗收

- 共用 `BlockSession` 預設 transition entries 保持 `[0]`，避免改變 CoAB 或其他作品。
- 提供作品 adapter 可設定的 entry sequence；Pool 設為 `[0,4]`。設定須驗證索引範圍，
  Clone 必須保留設定及 pending queue。
- `NEWECL` 先安全切 block並啟動第一 entry；每個 entry 的 `EXIT` 若仍有 pending entry，
  自動切到下一 entry繼續，直到玩家 boundary、external event、最後 `EXIT` 或 step limit。
- engine synthetic 測試涵蓋 `0→4`、entry 0 中途 menu 暫停後續行、非法設定與 Clone。
- Pool 真檔正常按鍵在公告後向前：script block 8、GEO map仍為 `(3,0)`、位置 `(4,4,2)`，
  並停在原始 `LOAD FILES` external boundary；未有 consumer 前不得安靜略過。
- CoAB 以預設 `[0]` 做玩家／ECL 抽樣回歸，行為不得改變。

實作已由共用 `BlockSession.SetTransitionEntries` 提供可設定序列及可複製的 pending
queue；external event 同時保留所有 numeric arguments 與逐項 valid bit。Pool 設為
`[0,4]`，只自動消費已由本規格閉合的 `LOAD FILES` 與 sentinel `LOAD PIECES`；其他
wall selector 仍失敗即關閉。engine 全測試、Pool 全測試、CoAB
`cmd/azure-bonds-game`／`internal/game`／`internal/ecl` 抽樣均通過。
Pool 正式依賴已鎖定為
`v0.0.0-20260831165626-7e9305036c43`；這是包含本規格 transition lifecycle 與後續
Spec 027 `SAVE TABLE` 的相容超集，並在斷網 Docker／Xvfb 以該版本重跑全綠。
重生 sweep 仍為 856 `EXIT`／168 玩家事件／0 error，另明列 16 筆 `LOAD FILES`
與 16 筆 `LOAD PIECES` 為已觀察的資源 lifecycle，不再灌入玩家事件數。
