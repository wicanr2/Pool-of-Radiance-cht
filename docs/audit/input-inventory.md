# 原始輸入盤點（2026-08-31）

## DOS ZIP

- 檔名：`Pool of Radiance (1988).zip`
- SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`
- ZIP entries：169
- DAX files：113
- engine `dax.Parse` 成功：113／113
- decoded blocks 合計：1,245
- 機器可讀逐檔結果：[`dos-dax-inventory.json`](dos-dax-inventory.json)

這證明目前 DOS corpus 可由 engine commit `86ac57e498c9` 的 DAX container
contract 完整切塊，並使 Pool 成為第二個真實 consumer。它不證明每個 payload
record、圖片尺寸、ECL opcode 行為或玩法語意；那些仍須依 consumer 與 DOS oracle
逐項建立 READY spec。

## 歷史中文化 RAR

- 檔名：`珍009-光芒之池.rar`
- SHA-256：`209265086b6ad98d28bb51bb689737b48eb54ce0396ecb773e52ebdd5676409e`
- magic：`52 61 72 21 1A 07 00`（RAR4）

尚未解包；在記錄字碼、修改範圍、可執行檔差異與授權前，只作線索。

## D64 集合

八檔皆為 174,848 bytes。這符合標準 D64 容器大小，但目錄名稱 `amiga/` 與檔名
標示本身不能證明平台；後續應用 D64 檔案系統工具列目錄與 executable magic。

| 檔案 | SHA-256 |
|---|---|
| Disk 1 Side A | `7f3e851f8fc5ae74126f6f8fd8526f1289b566d82fe065c252c1ad2b255b86d7` |
| Disk 1 Side B | `b49ad459490e50acbadd6ea937300937dc7b70c30410271dfaf78034c56b0e1b` |
| Disk 2 Side A | `00749278d80eb82a2577ba2fa6c33a11d7c3fc6e670b993771dc2c85daf47a6c` |
| Disk 2 Side B | `8d398ef6b6b5733eb07bfd810665b4c77c3ca09f74d7ba5b91843b82ca22a8a5` |
| Disk 3 Side A | `1f37c7a333080b7441aa84a556006e7271998474992e3765e55dcd6998cf32c8` |
| Disk 3 Side B | `e356704626f7c8454876e40958e0efcb34dcb083fffe41b529f415f9c65bd5b8` |
| Disk 4 Side A | `5ba45c20af16e284451e62232a907a35c738cc3ff2fd6594ff5e8ddb2c68e5fa` |
| Disk 4 Side B | `92149a63c92168a55d74f5797928fc52f23e3fb15f0b3418465e10a482c7dbe8` |
