# Pool of Radiance remake 現況

更新日期：2026-08-31。

## 已證實

- DOS 來源 ZIP SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- ZIP 有 169 筆，內含 47,936-byte `START.EXE`、232,379-byte `GAME.OVR`，
  以及 `ECL1..8`、`GEO1..8`、`WALLDEF1..8` 等 DAX。這只證明檔案 inventory，
  尚未證明 executable 版本、DAX consumer 語意或遊戲完成度。
- `珍009-光芒之池.rar` SHA-256：
  `209265086b6ad98d28bb51bb689737b48eb54ce0396ecb773e52ebdd5676409e`；
  magic 是 RAR4。它只作歷史中文化線索，不是 DOS 行為 oracle。
- `amiga/` 中八檔實際大小都是 174,848 bytes，符合 D64 容器形狀；檔名中的
  `amiga` 尚未由內容證明，不把它當平台事實。
- 共用 engine `86ac57e498c9` 已提供 `dax` 與 `ecl` codec；Pool 是第二作品
  consumer，作品位址、文字與劇情不得回填 engine。
- 113／113 個 DOS DAX 已由 engine `dax.Parse` 成功解析，合計 1,245 blocks；
  這只關閉 container shape 閘門，不代表 payload semantic parity。
- `START.EXE` 是 MZ，`GAME.OVR` 以 `TPOV!` 開頭；配合 overlay/runtime 字串，
  Borland／Turbo Pascal overlay family 目前是 `strong inference`，精確版本未知。

## 尚未知／不阻擋目前盤點

- DOS 發行版精確 revision、compiler／linker／overlay 精確版本（family 已有強推論）。
- Pool 與 CoAB 的 DAX／ECL 共用程度，以及個別 payload record 語意。
- 歷史中文 RAR 的字碼、修改範圍、可執行檔差異與授權狀態。
- D64 實際平台、檔案系統內容及其與 DOS 版的關係。

## 現行驗證策略

先做唯讀 inventory、雜湊與 fail-closed codec 驗證；再建立 READY spec，才實作
玩家行為。首條玩家垂直鏈固定為標題 → 建角／建隊 → Phlan 第一個正常可操作
地圖 → 事件／戰鬥 → 存檔／讀檔。
