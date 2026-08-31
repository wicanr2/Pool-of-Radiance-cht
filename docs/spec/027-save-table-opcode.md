# Spec 027：Pool ECL opcode `35h SAVE TABLE`

狀態：CONFORMED；日期：2026-09-01。

## 玩家阻塞點

正常玩家已從 City Hall 外走到 clerk 第一頁；第二次 Return 後，原始
`ECL3/block8 9C9Eh` 停在尚未實作的 opcode `35h`。這是 clerk reward table 的回寫，
不能以 passthrough 略過，否則後續 reward／commission 狀態會與原版分岔。

## 固定證據

- DOS ZIP 與 `overlay-03.bin` 的雜湊、IDA Pro 9.4、metapc 16-bit、overlay-local
  位址空間同 Spec 019／025。
- dispatcher `350Fh..3518h`：比較 opcode `35h` 後呼叫 `0FA2h`。
- 非破壞性 handler 匯出：`docs/audit/ida-overlay03-op35.json`；函式
  `0FA2h..0FEEh` 保留原始 bytes、operands、輸入 SHA-256 與 IDA 版本。

## 原版 handler（exact）

`0FA2h..0FEEh` 的資料流：

1. `0FA8h..0FABh` 準備三個 operands。
2. `0FB0h..0FB8h` 以 numeric resolver 解析 operand 0，保存為待寫 value。
3. `0FBBh..0FC8h` 解析 operand 1 的 word address，保存為 table base。
4. `0FCBh..0FD3h` 以 numeric resolver 解析 operand 2，保存為 index。
5. `0FD6h..0FDCh` 計算 16-bit `destination = base + index`。
6. `0FDFh..0FE5h` 以既有 store service 寫入 `value`。

作品中立 typed 契約因此是：

`memory[wordAddress(operand1) + numeric(operand2)] = numeric(operand0)`。

地址加法採原版 16-bit wrap；任一 numeric／address operand 不合法時失敗即關閉。

## City Hall consumer

`9C9Eh` 的三個 operands 是 `[6E7Ah], 4A8Fh, [6E79h]`。前方先由兩張 table 取值、
計算是否有 reward，再由 `35h` 把本輪值回寫到 `4A8Fh + [6E79h]`。所以它不是
無玩家影響的 runtime helper。

## 驗收

- engine synthetic：驗證 value／base／index 順序、write receipt、16-bit wrap，及
  不合法 operand 失敗即關閉。
- Pool 真實正常按鍵：clerk 第一頁 → Return menu → 過 `9C9Eh`，不得再報 unknown
  opcode；後續第一個新 boundary 另依原始內容精確斷言。
- 共用引擎全測試、Pool 全測試與 CoAB ECL／game 抽樣通過。

## Conformance

- engine `7e93050` 在 `eclvm.Machine` 實作作品中立 `SAVE TABLE`，沿用既有
  `NumericValue`、`WordAddress` 與 `store` receipt；沒有 Pool 專屬分支。
- `TestSaveTableUsesValueBaseIndexAndWrapsAddress` 驗證 operand 順序、唯一 write
  receipt 與 `FFFFh + 2 -> 0001h`；`TestSaveTableFailsClosedOnInvalidOperands`
  分別驗證 value／base／index 不合法時失敗即關閉。
- engine `go test ./...` 全通過。Pool 正常按鍵測試穿過 `9C9Eh`，下一個玩家 boundary
  精確為 `THE CLERK SHUFFLES THROUGH HER PAPERS... I CAN OFFER THE FOLLOWING`，不再出現
  unknown opcode；Pool 使用正式 pseudo-version
  `v0.0.0-20260831165626-7e9305036c43`，沒有本機 `replace`。
- Pool `go test ./...` 全通過；CoAB 升到同一 engine pseudo-version 後，
  `go test ./internal/ecl ./cmd/azure-bonds-game` 抽樣通過。因此本規格的 engine、
  Pool 玩家路徑與既有作品回歸閘門均已閉合。
