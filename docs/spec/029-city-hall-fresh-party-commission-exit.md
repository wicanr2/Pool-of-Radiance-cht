# Spec 029：全新隊伍 City Hall 委託列舉與離場

狀態：READY；日期：2026-09-01。
實作：ECL 腳本本身，由共用 engine 的 `eclvm` 直接跑——**remake 這一側沒有市政廳專屬的程式碼**，前端只提供文字框與選單（`cmd/pool-game` 的事件消費）。玩家路徑的驗證在 `cmd/pool-game/coverage_test.go`。

## 範圍

只閉合正常按鍵路徑中「全新隊伍、七個 reward 槽皆無正差、委託旗標維持初始值」的
clerk 第一頁之後流程。它不授權 reward 副作用、接受 graveyard 特別委託、戰鬥、寶物、
隊伍強度或其他非預設旗標分支；那些仍由 DRAFT Spec 028 管理。

## 原版證據（exact）

- 固定輸入為 `docs/audit/dos-ecl3-block8-trace.json`：block SHA-256
  `fd446439973994d9f887b8628b512054a822369eb4360769b21eb2224982f012`，位址基準 `9900h`。
- `9C34h..9CB8h` 掃七個 reward 槽；全新狀態無正差，因此不進 reward external service。
- `A7A4h` 顯示委託引言，`A83Fh` 從索引零開始，`A85Bh` 依序分派十六個槽。
- 初始旗標路徑只產生三段玩家文字：
  1. `A8C8h`：清除西方 slums 的怪物。
  2. `A909h`：清除 Thorn Island 的 Sokal Keep。
  3. `A939h`：回收 old Phlan 的 books／maps／tomes 等資料。
- 其餘入口依初始條件返回；`AF4Bh` 顯示已列完所有現有委託，`AF78h GOSUB` 後由
  `AF7Ch EXIT` 離開本次 ECL cell event。
- 有序分派與位址由 `docs/audit/dos-city-hall-structure.json` 重生；少一條 edge、少一個
  `4AC1h` producer 或來源 hash 改變時，audit 測試必須失敗。

## 實作契約

1. 必須從標題、原版建角／組隊、Rolf、Sune 與 City Hall doorway 的正常按鍵路徑抵達，
   不得 direct-entry 到 block8 或 clerk 座標。
2. clerk 引言及後續每頁文字都採原版兩階段輸入：第一次確認顯示同文字的單項 Return
   menu，第二次確認選取後才推進原 ECL VM 的下一個玩家 boundary。三項委託必須依上述
   順序各出現一次，最後出現 `THESE ARE ALL...`。
3. `THESE ARE ALL...` 結尾頁不再建立另一個 Return menu；再按一次確認後必須由
   `AF7Ch EXIT` 結束 cell event，恢復地城移動。不得把未支援 external
   opcode 當 no-op，也不得以人工文字清單替代 VM。
4. 驗收同時斷言 block8 session、clerk 座標與最終 `cellEventPending=false`；只驗文字聯集
   不算完成。

## 負對照

- 改變 trace hash、刪除任一 commission edge 或改掉任一 `4AC1h` 增量形狀，機器 audit
  必須失敗。
- 測試若在預設路徑遇到 `PARTYSTRENGTH`／`TREASURE`／`COMBAT`，視為旗標 fixture 或
  控制流錯誤，不能 passthrough 強行通過。
