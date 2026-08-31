# Spec 020：`14h COMPARE AND` 四 operand 比較

狀態：CONFORMED  
日期：2026-08-31

## 範圍

Spec 019 接通 `20h NEWECL` 後，初始地圖 sweep 有四個樣本進入目的 block 並停在
`14h`。本規格只閉合該作品中立比較運算與六個 IF flag；不解釋各 Pool memory
address 的劇情語意。

## 輸入與證據

- DOS ZIP SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `overlay-03.bin` SHA-256：
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`。
- IDA Pro 9.4、`ida-pro-9.4-idapython:locked-v1`、metapc 16-bit raw binary；
  位址為 overlay-local file offset、base 0。
- 主 dispatcher `339Eh..33ACh` 比較 `14h` 後呼叫 `0AB5h`。
- 非破壞性匯出：`docs/audit/ida-overlay03-op14.json`，保留原始 bytes、operands、
  函式邊界、工具版本與 input hash。

## 原版行為（exact）

`0AB5h..0B19h`：

1. `0ABBh..0AC7h` 將 `DS:6D3Eh` 起的六 bytes 清為 0；這六格是後續
   `16h..1Bh` 六種 IF 的比較結果。
2. `0ACCh..0AF8h` 推進 cursor 並依序解析四個 numeric operands。
3. `0AFAh..0B08h` 判斷 `(value0 == value1) AND (value2 == value3)`。
4. 成立時只設 `DS:6D3Eh=1`（IF `=`）；不成立時只設 `DS:6D3Fh=1`
   （IF `<>`）。其餘 `<／>／<=／>=` 四格保持 0。

CoAB 現行 runtime 的 `14h` 也採相同四 operand／兩組相等且關係；這是第二作品的
交叉驗證，但 Pool exact 結論以上述 dispatcher 與 handler bytes 為主。

## Typed 行為

共用 `eclvm.Machine` 解析四個 operands，使用既有 `ecl.NumericValue` 取得值，將
compare flags 設為：

```text
[bothEqual, !bothEqual, false, false, false, false]
```

任何 operand 不可解析時失敗即關閉。`16h..1Bh` 沿用既有「條件不成立就跳過下一個
完整 record」行為。

## 驗收

- synthetic 正對照：兩組都相等時 `IF =` 執行、`IF <>` 跳過。
- synthetic 負對照：任一組不相等時 `IF <>` 執行、`IF =` 跳過。
- `<／>／<=／>=` 在 `COMPARE AND` 後不會沿用上一個 `COMPARE` 的 stale flags。
- Pool sweep 的四筆 `14h` error 歸零；新暴露邊界另行分類，不以測試期望掩蓋。
- engine／Pool 全測試與 CoAB ECL／game 回歸通過。

## 實作收據

- 共用 `eclvm.Machine` 已依上述固定六旗標契約實作 opcode `14h`，並以 synthetic
  測試覆蓋正反對照與 stale flag 清除。
- Pool 初始地圖 sweep 已由 852 `EXIT`／156 event／16 error 收斂為
  856 `EXIT`／156 event／12 error；四筆 `14h` 錯誤歸零，剩餘 12 筆全為
  `0Ah LOAD CHARACTER`，沒有用 passthrough 或改期望值掩蓋。
- 正式鎖定引擎 `v0.0.0-20260831122741-b9eee757e060` 後 Pool 全套測試通過；CoAB
  玩家與 ECL 核心路徑亦通過。CoAB 全儲存庫仍有兩項與本切片無關的既存稽核失敗，
  已分開記錄，不把它們誤報成 opcode 回歸。
