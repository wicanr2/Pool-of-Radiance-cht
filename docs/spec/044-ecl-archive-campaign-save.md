# Spec 044：ECL archive campaign 存檔

狀態：READY／CONFORMED；日期：2026-09-01。

## 缺口

schema 5 保存 GEO archive／block 與 `BlockSessionSnapshot.Current`，但 block ID 只在單一
ECL archive 內唯一。進入 Slums 後，`Current=20` 若仍用起始 ECL3 catalog restore，會
拿到錯誤 block 或直接失敗；GEO archive 不能代替明確的 ECL namespace 契約。

## schema 6 契約

- `Campaign.ecl_archive` 明確保存 1..8；`Session.Current` 仍是該 archive 內的原始 block ID。
- F10 在穩定玩家邊界同時保存 GEO identity、ECL archive、session snapshot 與位置。
- Load 先從版本化八 archive catalog 選定 ECL namespace，再建立 callbacks 並原子 restore；
  不再永遠用 ECL3 起始事件 catalog。
- schema 5 缺少欄位時，以當時保存的 `map_archive` 遷移。這對 schema 5 已可保存的現行
  玩家路徑是確定性的；schema 1..4 仍沿既有遷移鏈處理。
- schema 6 的 `ecl_archive=0` 或超出 1..8 必須失敗即關閉。

## 驗收

- ECL3 既有 F10／Load round-trip 保持通過。
- ECL2/block20、GEO2/block20 fixture 保存 `4ABBh=24` 後讀回，必須仍是 archive 2、
  block 20、相同 map 與 memory，不可誤接 ECL3。
- schema 5 fixture 必須遷移 `ecl_archive=map_archive`；全專案測試與 vet 必須通過。
