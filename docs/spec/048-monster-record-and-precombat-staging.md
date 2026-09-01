# Spec 048：怪物角色記錄與戰鬥前 staging

狀態：CONFORMED（MON*CHA block identity、最小 typed record、戰鬥前 staging）；
DRAFT（怪物完整能力欄位、戰術畫面、戰鬥規則與勝敗 continuation）。

## 輸入與證據

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- 共用 `dax.Parse` 對 `MON1CHA.DAX..MON8CHA.DAX` 全掃得到 172 個 block；每個
  decoded payload 都恰為 285 bytes，沒有尺寸例外。每筆 byte 0 是 1..15 的
  Pascal name 長度，且 `1+length` 均未越界。
- `docs/audit/dos-ecl2-block20-trace.json` 的 Pool ECL 位址空間
  `9E5Dh..9E6Ch` 依序為：`CLEARMONSTERS`、`LOAD MONSTER 13,1,4`、
  `LOAD MONSTER 4,3,4`、`COMBAT`。Spec 046 已閉合 descriptor 與 PC continuation。
- `MON2CHA.DAX` block 13 與 block 4 都是 285-byte 記錄，Pascal name 均為
  `ORC`。因此 ECL 第一個運算元對應同 archive `MON2CHA.DAX` 的 DAX block ID；
  第三個運算元 icon block 仍是獨立欄位，不得拿來索引角色記錄。

證據等級：archive／block identity、285-byte shape、Pascal name 與本場兩筆 ORC
為 `exact`；其餘 284 bytes 的能力、裝備、AI 與法術語意維持 `unknown`。

## 實作契約

1. Pool game pack 依目前 ECL archive `N` 唯一尋找 `MONNCHA.DAX`，用共用 DAX
   decoder 解析，再依 `MonsterID` 唯一選取 block；archive 必須在 1..8。
2. payload 必須恰為 285 bytes；name length 必須在 1..15 且不越界。typed record
   只公開 `ID`、`Name` 與完整 `Raw [285]byte`，不替未知 offset 命名。
3. duplicate archive member、duplicate block、缺 archive／block、尺寸或 name 異常
   一律失敗即關閉。Loader 不修改原版 ZIP。
4. 前端收到 `CombatRequested` 且 monster list 非空時，先逐 descriptor 載入 record，
   保留 ECL 的順序、數量與 icon block，進入不可存檔的戰鬥前 staging。玩家畫面可顯示
   原始名稱與數量，但不得先執行 `9E6Dh` 之後的戰後腳本。
5. 空 monster list 的 `COMBAT` 仍交給既有作品服務分派（例如 Sune／treasure），
   不誤判為一般遭遇。真正戰術戰鬥、勝利／逃跑／全滅與戰後續跑要由後續 READY
   spec 授權；目前 staging 沒有「繼續／自動勝利」按鍵。

## 驗收

- 合成 DAX／ZIP 測試正常 block、duplicate、缺 block、錯尺寸與錯 name length。
- 真實 `MON2CHA.DAX` 抽驗 block 13、4 皆為 `ORC` 且 raw 長度 285。
- 真實 `9E5Dh` session 經 Pool 前端 consumer 後，staging 順序為
  `ORC ×1 (icon 4)`、`ORC ×3 (icon 4)`，ECL PC 保持在 `9E6Dh`，`4ABBh` 未改變。
- Pool 全套 `go test ./...`、`go vet ./...`；不以 direct increment 或 forced-win
  冒充戰鬥完成。
