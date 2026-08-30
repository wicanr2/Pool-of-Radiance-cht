# Spec 002：DOS ECL 位址基準與 parser 缺口

狀態：DRAFT；日期：2026-08-31。

## 已證實範圍

固定輸入為 SHA-256
`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`
的 DOS ZIP。`ECL1..8.DAX` 合計 29 blocks；每個 payload 開頭 word 均為
`1388h`，但它不是可執行位址基準。

入口表中反覆出現 `9914h`，而以 `9914h` 為基準時該入口精確對應 payload code
offset 0。這支持作品層 `poolCodeAddressBase = 0x9914`；CoAB 的 `8000h` 不可作為
共用 engine 常數。共用 API 因此改為由 game adapter 明確傳入 code-address base。

## 尚未閉合

以 `9914h` 解析後，現有 engine graph decoder 只完整走過 3／29 blocks，共 24
條 reachable instructions；26 blocks 仍在 record 邊界或 opcode 解碼失敗。這些
失敗只證明 CoAB decoder 不能直接套用，不能據此宣稱：

- 原版有 26 個未知 gameplay opcodes；
- 失敗 byte 一定是 opcode；
- Pool ECL VM 已達可實作狀態。

機器可讀收據是 `docs/audit/dos-ecl-coverage.json`。在 production VM 使用前，須從
原版 consumer caller、入口表讀法及至少一個 runtime trace 交叉驗證 variable／
instruction record 邊界，更新本規格為 READY，並讓 29 blocks 的失敗分類具有
可回查 bytes、位址空間與推論等級。

## 下一個驗收閘門

1. 用 IDA `.i64` 的 caller／xref 確認 `1388h` 與入口表的讀取方式。
2. 對一個目前成功與一個目前失敗的 block 匯出原始 bytes、entry、逐步邊界。
3. 建立負對照，確保錯誤 code-address base 或截斷 record 必定失敗。
4. 只有 record contract 達 READY 後，才擴充 engine decoder 與 Pool VM adapter。
