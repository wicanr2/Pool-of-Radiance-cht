# Spec 045：跨 archive NEWECL 與 New Phlan→Slums

狀態：READY（跨 archive session 與 Slums 資源交接）；DRAFT（Slums 全事件／戰鬥）。

## 原版證據

- DOS `START.EXE`：47,936 bytes，SHA-256
  `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`。
- IDA Pro 9.4、DOS MZ loader；位址空間與逐指令 file offset 見
  `docs/audit/ida-start-ds52d4-references.json` 與
  `docs/audit/ida-start-archive-switch.json`。
- resident file `2A2Fh..2A43h` 保存 `DS:52D4h` 到 `52D5h`，再把 byte 參數寫入
  `52D4h`；`2A44h..2A50h` 做反向恢復。這證明 archive selector 有正式 controller，
  不能只盤點直接寫 `52D4h` 的 overlay。
- `overlay-07` SHA-256
  `a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae`；
  `0C16h..0C25h` 的特殊 operand 分派把新值傳給 resident archive setter；逐指令
  證據為 `docs/audit/ida-overlay07-archive-switch-span.json`。
  `0312h` 是該分派器內部識別，不是 ECL 可直接掃描的 runtime address。
- `ECL3.DAX` block 0 的真實 trace 在 `9955h..9965h` 依序為：
  `LOAD FILES FF,FF,7F`、`SAVE 2,6E12h`、`NEWECL 20`。`SAVE` 的一般 VM 語意
  必須保留 `Memory[6E12h]=2`；Pool adapter 以此選擇 ECL archive 2。
- 切到 `ECL2.DAX` block 20 後，既有 Spec 043 的 entry 4 證據依序要求
  `LOAD FILES 20,2,FF` 與 `LOAD PIECES 2,4,1`，因此正常結果是
  `ECL2/block20 + GEO2/block20 + WALLDEF2 slots 2/4/1`。
- `overlay-17` 的 `docs/audit/ida-overlay17-area-archive-record.json` 只證明
  party-save `+0624h ↔ DS:52D4h` 的保存／恢復；它不是正常跨區域 producer。

推論等級：上述 bytes、hash、指令順序與 archive/resource identity 為 `exact`；
「New Phlan→Slums」名稱由原版文字、commission 與 block 20 helper 交叉支持，為
`strong inference`，不把尚未逐格驗證的 Slums 全事件升格為完成。

## 實作契約

1. 共用 engine 的 `BlockSession` 可接受作品中立的 catalog resolver。resolver 只在
   `NEWECL` target 解析前收到 `(from,to,唯讀 memory snapshot)`，回傳另一份 block
   catalog；engine 不得知道 `6E12h`、Pool、archive 檔名或地名。
2. resolver 未設定時維持現行同 catalog 行為；回傳 catalog 缺 target、空 catalog
   或 callback error 必須失敗即關閉，且 current block 不得先改變。
3. Pool adapter 的 resolver 只接受 `Memory[6E12h]` 為 `1..8` 且 catalog 存在；它
   選出相同 archive 的 ECL blocks。跨界完成後同步 `eclArchive` 與 map archive，
   才消費新 block 的 `LOAD FILES`／`LOAD PIECES`。
4. `LOAD FILES FF,FF,7F` 依本路徑作為不換 GEO 的控制邊界消費；不得誤讀成
   `GEO*/block255`。其他未證實的 partial sentinel 組合仍失敗即關閉。
5. 真實 bytes 測試必須由 ECL3/block0 的 `9955h` 前開始，走過 archive 2 transition，
   最終同時驗證 ECL、GEO 與三個 wall selectors；direct-entry 到 ECL2 不算此驗收。
