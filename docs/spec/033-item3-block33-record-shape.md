# Spec 033：`ITEM3.DAX/33h` 五筆 63-byte 物品紀錄

狀態：CONFORMED（archive/block identity、record 數、63-byte shape、內嵌名稱與 typed loader）；DRAFT（其餘
欄位語意、卷軸法術內容、裝備規則、價值與戰利品分配）。
日期：2026-09-01。

## 可重生輸入與證據

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- ZIP member `poolrad/item3.dax`：4,687 bytes，SHA-256
  `84d71e83b5d51b02c407e9969cae51ef5034af111a8885b34a6c4e04e4e03d89`。
- 共用 `dax.Parse` 對 block `33h` 取得 decoded size 315、packed size 284；
  `315 = 5 × 63`。
- Spec 032 的 Pool handler `1B56h..1BF1h` 以固定 `3Fh` bytes 逐筆複製，並每次把
  source offset 加 `3Fh`，直到 block decoded length 用完。故 63-byte record shape 與
  五筆數量為 `exact`，不是由 CoAB struct 倒推。
- 每筆開頭是長度不超過 40 的 Pascal string；五筆原始名稱依序為：
  四筆 `Clerical Scroll With 2 Spells`，以及一筆
  `Two-Handed Sword +1 +3 vs. Undead`。本輪只把這段內嵌字串當 display name，
  不替後續 bytes 命名。

## Typed adapter 契約

1. `TreasureItemRecord` 保存完整 `[63]byte` raw record 與解出的名稱；呼叫端取得值副本。
2. loader 以 `(archive=3, block=33h)` 精確查找，拒絕缺檔、重複 `ITEM3.DAX`、重複
   block、非 63 倍數 payload、空名、名稱長度超過 40 或越過 record。
3. adapter 不把四張同名卷軸合併，也不把最後一筆只簡化成「附魔武器」；原順序與
   五份 raw identity 必須保留。

## 驗收與下一步

- 真檔測試固定 member 雜湊、315 bytes、五筆名稱及最後一筆 raw record 的雜湊。
- synthetic 負對照固定 malformed record length 與非法 Pascal name。
- 下一步需從 Pool 的 item display／take consumer 閉合欄位與分配流程；在此之前可以
  顯示原始名稱清冊，但不能修改角色裝備或宣稱卷軸法術已可使用。
