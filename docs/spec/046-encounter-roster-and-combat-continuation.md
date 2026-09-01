# Spec 046：遭遇名冊與 COMBAT 續跑契約

狀態：READY（ECL 遭遇資料與續跑邊界）；DRAFT（戰術戰鬥畫面與完整規則）。

## 原版與同系列證據

- Pool DOS `ECL2.DAX` block 20 的 trace（`docs/audit/dos-ecl2-block20-trace.json`）
  有 20 條 `COMBAT`，其中 14 條在戰鬥後直接 `GOSUB B69Ch`。例如
  `9E5Dh` 起依序為 `CLEARMONSTERS`、兩條 `LOAD MONSTER`、`COMBAT 9E6Ch`、
  `SAVE`、`GOSUB B69Ch`、`EXIT`。因此 `B69Ch` 是成功返回後的腳本續段，不能在
  發出戰鬥請求時預先執行，也不能用自動勝利代替戰鬥結果。
- CoAB DOS `overlay-02:0179Ah` 與 PC-98 `overlay-02:01820h` 已逐指令交叉驗證：
  `COMBAT` handler 先把 ECL PC 推到下一條，再同步呼叫戰鬥，返回同一 ECL 主迴圈。
  詳見 CoAB spec 1095；兩作共用 ECL 執行模型，這裡只沿用作品中立契約。
- CoAB DOS `overlay-02:120Eh` 證明 `CLEARMONSTERS` 清除待戰怪物鏈；
  `SETUP MONSTER` 的 sprite／picture／距離資料不在該鏈中，必須保留。
- ECL opcode `0Bh LOAD MONSTER` 有三個可解析數值運算元：怪物 record ID、數量、
  icon block。戰鬥能力值必須另由當前 archive 的 `MON*CHA.DAX` 取得，不可從 icon
  或名稱猜測。

推論等級：opcode 順序、位址、PC 前進與清除範圍為 `exact`；Pool 的 14 個
`B69Ch` 呼叫皆代表何種 clearance 計分，尚未逐支閉合，維持 `strong inference`。

## 實作契約

1. 共用 `eclvm.Machine` 原生累積 `LOAD MONSTER` descriptor；`CLEARMONSTERS`
   清空累積名冊但保留最近的 `SETUP MONSTER`。兩種狀態都屬 continuation，必須
   進入 clone／snapshot／restore，否則戰鬥前存讀檔會換成另一場遭遇。
2. `COMBAT` 產生型別化 `CombatRequested` 邊界，攜帶當下 setup 與名冊；PC 在
   邊界交給前端前已指向下一條。前端完成真正戰鬥後才從該 PC 續跑。
3. engine 只保存 ECL descriptor，不得知道 Pool archive、`MON2CHA.DAX`、Slums、
   `B69Ch` 或勝利計數器。Pool adapter 負責依現行 archive 載入怪物 records。
4. 空名冊的 `COMBAT` 仍可代表作品服務分派，保留既有 event；Pool 只有在名冊
   非空時才把它判為遭遇。無法解析運算元或找不到怪物 record 時失敗即關閉。
5. 戰敗／全滅不得續跑戰後 ECL。測試真的全滅時停止該玩家路徑並記錄結果，
   不修改亂數、HP 或敵人資料強求通過。

## 本輪驗收

- 合成 bytes 驗證 `SETUP → CLEAR → LOAD×N → COMBAT` 的有序狀態、PC 與快照 round-trip。
- 真實 Pool `ECL2/block20` 至少抽一條 `9E5Dh..9E6Ch`，確認兩個怪物 descriptor
  原樣抵達型別化戰鬥邊界；不得 direct increment `4ABBh`。
- 共用 engine 與 Pool 全套 `go test ./...`、`go vet ./...` 必須通過。
