# Spec 019：Pool ECL dialect 與 `20h` 跨 block 交接

狀態：CONFORMED（`20h` 終止訊號、目的 block、code window 與初始入口）；DRAFT（`0Ah` active character 投影）  
日期：2026-08-31

## 問題與勘誤

初始地圖 16×16×四方向 sweep 的 28 個 fail-closed 樣本落在 opcode `20h` 與
`0Ah`。先前文件曾以 TPOV overlay entry index 推算 opcode handler，並把 CoAB 的
mnemonic 直接套進 Pool；Pool 主 dispatcher 的逐值 IDA 匯出已證明這個方法錯誤。

`overlay-03:32DFh..35DAh` 直接比較 `DS:6D45h` 的 opcode byte：

- `0Ah → 02E3h`
- `20h → 0CDDh`
- `24h → 186Ch`

因此 handler 只能由 dispatcher callsite 定位，不可由 entry index、CoAB 名稱或函式
排列推算。舊的「`20h → 2DF9h`」「`0Ah → 0ED5h`」「`24h → 2E90h`」全部廢止。

## 輸入、工具與位址空間

- DOS ZIP SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `overlay-03.bin` SHA-256：
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`。
- IDA Pro 9.4，image `ida-pro-9.4-idapython:locked-v1`，metapc 16-bit raw binary；
  位址是 overlay-local file offset、base 0。
- 非破壞性匯出：
  `docs/audit/ida-overlay03-ecl-dispatcher.json`（主 dispatcher）與
  `docs/audit/ida-overlay03-op0a-op20-op24.json`（三個真 handler）。兩份均保留
  input hash、IDA 版本、原始位址、bytes、operand 與函式邊界。
- Pool ECL decoded payload 的 mapping base 是 `9900h`；五個 entry operands 共
  20 bytes，第一個 lifecycle entry 才是 `9914h`。

## `20h` 原版控制流（exact）

dispatcher `341Ch..3425h` 比較 opcode `20h`，呼叫 `0CDDh`。handler 的動作順序：

1. `0CE4h..0CE7h` 呼叫 operand cursor helper，準備一個 operand。
2. `0CECh..0CF5h` 將目前 block ID `DS:82A2h` 保存到 active party structure
   `ES:[DI+1E4h]`。
3. `0CFAh..0D08h` 解析第一個 operand byte並寫回 `DS:82A2h`；這個值就是目的
   ECL block ID。
4. `0D0Bh..0D61h` 由目的 ID 組合資源名稱、載入並初始化新 block。
5. `0D66h` 與 `0D6Bh` 將 `DS:4390h`、`DS:4391h` 設為 1；`0D70h` 清
   `DS:4398h`。
6. interpreter controller `35DBh..361Ch` 的迴圈在 `35E9h` 先檢查 `4390h`；非零
   立即離開，不會把目的 operand 後方的 byte 當成同一 block fallthrough。

Pool corpus 的初始地圖實例為 `A22Dh/A230h → block 11` 與 `AD7Fh → block 8`。
相鄰的兩筆 `20h` 可以是不同控制流入口；實作不得因物理相鄰而順序執行兩次。

以上與 CoAB 的 `20h` 終止／載入／重啟 lifecycle 行為獨立吻合；Pool 的位址、
code base 與 block catalog 仍由 Pool adapter 提供，不能寫進共用 engine。

## Typed 行為（READY）

1. 共用 VM 執行 `20h` 時解析一個 numeric operand，回傳
   `Result.NewECLBlockID`，並在該 record 結尾終止目前 block；不可產生一般 external
   passthrough event，也不可執行下一個 byte。
2. 共用 block session 只擁有作品中立的 `block ID → decoded block bytes`、目前 block
   identity、code base、VM continuation 與 deterministic random stream。
3. 交接時先驗證目的 block 存在並可由 `ecl.EntryPoints(block, 5)` 解出五個入口；
   失敗即關閉，原 session 不得半切換。
4. 成功交接時以目的 block payload 取代舊 code window，保留 code window 之外的 shared
   memory、Strings 與 random stream，清空 GOSUB stack。先前只寫「從目的 block
   entry 0 重啟」並不完整；Spec 025 依 lifecycle controller 訂正 Pool 的完整序列為
   entry `0 → 4`。共用 session 的預設仍可是 entry 0，但 Pool adapter 必須宣告兩段序列。
5. session 可在同一次 `RunUntilEvent` 內跨過一或多個 `20h`，直到真文字、選單、
   external boundary 或 `EXIT`；結果保留 transition 記錄，adapter 可查目前 block。
6. Pool `ReadDOSInitialEvent` 必須保存 ECL3.DAX 的完整 block catalog，不再只保存 block 0。
   初始 session 從 block 0／Rolf handler 建立；後續 target 8／11 由原始 catalog 載入。
7. block 交接不自動猜地圖、座標、牆組或劇情；目的 block 若發出 `LOAD FILES` 等
   external boundary，Pool adapter 仍應停住，等該服務另有 READY 規格。

## `0Ah` 邊界（DRAFT）

dispatcher `334Ah..3353h` 將 `0Ah` 路由到 `02E3h..03B0h`。handler 解析一個 byte，
低七位沿 `DS:5CF4h` 的 `+104h` far-pointer 鏈選角色，高位 `80h` 控制附加處理，最後
將 active pointer 寫到 `DS:5CF0h/5CF2h`。這已證明它不是可忽略的 passthrough；但
角色鏈如何由 remake party/save 投影、越界與空鏈語意仍未閉合，因此本規格不授權
實作 `0Ah`。

## 驗收

- engine synthetic：`20h` 回傳目的 ID，下一個 byte不執行；缺 block、壞 header、
  越界 entry 失敗即關閉；交接保留一般 memory／RNG、替換 code window、清 stack。
- engine session：連續兩個 block 交接後抵達目的文字／menu／EXIT，transition identity
  與 aggregate result 正確；Clone 不共享 mutable state。
- Pool 真檔：ECL3 block 0 的 `A22Dh`／`AD7Fh` 分支分別切到 block 11／8，不再回報
  opcode `20h` missing handler；目的 block 的第一個玩家可見／external boundary 可重現。
- 重新產生 `docs/audit/pool-initial-cell-sweep.json`。原本 12 筆 `20h` error 必須歸零；
  新暴露的 opcode 另按實際 handler 分類。第一次接線結果為 12 筆 `0Ah` 與 4 筆
  `14h`；不得用 passthrough 抹除，`14h` 由 Spec 020 接續。
- engine／Pool 全測試與 CoAB 現行 ECL／game 回歸通過；不得改變 CoAB 玩家行為。

## 實作收據

- 共用引擎 `eclvm.BlockSession` 已保存 shared memory、字串與亂數流，並在 `20h`
  驗證目的 block 後更換 code window、清除 stack。歷史實作只由 entry 0 繼續；
  Spec 025 已證明 Pool 還須接 entry 4，此處保留勘誤而不再把舊行為稱作完整 lifecycle。
- Pool adapter 已載入完整 ECL3 block catalog；正常遊戲、cell sweep 與 continuation
  都改走同一 session，不以座標表模擬交接。
- 正式依賴已鎖定共用引擎
  `v0.0.0-20260831122741-b9eee757e060` 時完成本切片驗收；Spec 021 後的現行鎖版為
  `v0.0.0-20260831132230-0a028e0956c1`。Pool 全套測試在斷網 Docker／Xvfb 通過。
- CoAB 以同版引擎重跑後，`cmd/azure-bonds-game`、`internal/game`、`internal/ecl`、
  `gamepack` 等玩家與 ECL 路徑通過。全儲存庫另有既存的 save ledger 對帳與
  `dist-all/` build 產物稽核失敗，兩者與本次引擎變更無關，未冒稱全綠。

## 停止線與權利

本切片只處理 bytecode control flow，不散布原版 DAX／overlay。完成 `20h` 後立即回到
玩家路徑；`0Ah`、目的 block 的地圖／戰鬥／角色副作用另開窄規格，不用本規格猜測。
