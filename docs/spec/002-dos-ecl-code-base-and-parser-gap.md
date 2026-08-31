# Spec 002：DOS ECL 位址基準與 parser 缺口

狀態：DRAFT；日期：2026-08-31。

## 2026-08-31 勘誤：映射基準是 `9900h`，不是 `9914h`

固定輸入為 SHA-256
`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`
的 DOS ZIP。`ECL1..8.DAX` 合計 29 blocks；每個 payload 開頭 word 均為
`1388h`，但它不是可執行位址基準。

舊版把入口表反覆出現的 `9914h` 稱為 payload code-address base；這是錯誤斷言。
`9914h` 是**第一條指令**的位址，不是整份解碼後 payload 的映射基準。原始證據為：

- `overlay-07.bin` SHA-256
  `a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae`；
  IDA Pro 9.4，overlay-local address space。
- ECL loader entry 5 `0353h..0420h` 在 `03F0h..0406h` 略過 DAX block 的兩-byte
  prefix，將其餘 bytes 複製到 `DS:493F` 指向的 `0x1E00` 緩衝。
- operand address classifier `070Bh..0763h` 在 `074Bh..0759h` 把
  `9900h..B6FFh` 明確分類為同一 bank；resolver `0EADh..0EC5h` 對該 bank 以
  `DS:493F + address + 6700h`（16-bit wrap）取值。因此緩衝第 0 byte 的位址是
  `9900h`。
- VM 初始化 `01C8h..0333h` 在 `0207h` 將 cursor 設為 `9900h`，接著五次呼叫
  operand parser（`025Fh..02C7h`），每次各消耗四 bytes，將五個入口存入
  `DS:4944..494C`。所以五個 header 恰佔 `0x14` bytes，第一條指令才位於
  `9900h + 14h = 9914h`。

結論（`exact`）：作品層 `poolCodeAddressBase = 0x9900`；入口位址減去該基準後，
得到的是**含五個 headers 的 raw payload offset**。CoAB 的 `8000h` 仍不可作為
共用 engine 常數。先前 `9914h` 結論保留在本勘誤中供追溯，但不得再被實作引用。

## 尚未閉合

舊收據以錯誤的 `9914h` 解析，只完整走過 3／29 blocks，共 24 條 reachable
instructions；26 blocks 所謂 record／opcode 失敗目前已降級為**錯誤位址基準造成的
量測假象**，不能據此宣稱：

- 原版有 26 個未知 gameplay opcodes；
- 失敗 byte 一定是 opcode；
- Pool ECL VM 已達可實作狀態。

機器可讀收據是 `docs/audit/dos-ecl-coverage.json`。下一步先以 `9900h` 乾淨重生；
若仍有失敗，再逐筆分類 record／opcode，而不可沿用舊的 26-block 數字。

## 下一個驗收閘門

1. 以 `9900h` 重生 29-block graph 收據；成功列保留 instruction count，失敗列保留
   最後 64 條 decoded tail 的 offset、opcode、next 與 operands，避免把 14,000 多條
   正常指令灌進版控報表。
2. 建立 `9914h` 負對照，至少證明 ECL3/block 0 會錯落到 raw payload `215` 的
   `AEh`，而 `9900h` 不會。
3. 對仍失敗的 block 才匯出原始 bytes、entry、逐步邊界。
4. 只有 record contract 達 READY 後，才擴充 engine decoder 與 Pool VM adapter。
