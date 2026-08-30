# Spec 009：DOS GEO 地圖盤點與 Phlan 入口

狀態：READY（GEO archive／block shape）；DRAFT（正常新遊戲入口、第一事件與地名）
日期：2026-08-31

## 可重生結構盤點

- 輸入 DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `cmd/pool-geo-audit` 只讀 `GEO1.DAX..GEO8.DAX`，先由 engine `dax.Parse` 解 archive，
  再由 engine `geometry.Parse` fail-closed 解 16×16 四平面。
- 報告：`docs/audit/dos-geo-inventory.json`。

八份 archive 合計 29 blocks；29／29 payload 皆為 `0x402` bytes（兩-byte prefix＋
四個 `0x100` planes），全部成功解成 16×16 cells，失敗 0。報告逐 block 保存：
原始 block ID、prefix、terrain histogram、四向 wall／detail 統計，以及 bounded、wrapped、
dungeon-door 三種 directed move edge 數。這些數字只證明資料與 engine contract 相容；
wrapped 與 dungeon-door 哪一個適用，必須由該 ECL／area consumer 決定。

驗收命令在 `coab-go-test:20260729` 的無網路 Xvfb 容器執行；使用暫存
`go.work` 將本專案連到唯讀的同層 engine，未在 `go.mod` 提交本機 `replace`。
`go test ./...` 全數通過；重新執行 `cmd/pool-geo-audit` 的 JSON 與版控報表相同，
且機械斷言確認 `(archives, blocks, decoded, failed) = (8, 29, 29, 0)`、每列
`bytes = 1026` 且沒有 `error`。

## 尚未閉合

- `GEO1` 不因編號最小就自動命名為 New Phlan 或 Slums。
- 正常玩家在 Party Creation Menu 選 Begin Adventuring 後的 ECL block、GEO archive、
  block ID、`x/y/facing` 必須由 runtime save／trace 或 executable producer 閉合。
- 第一個畫面事件與城內／地城 wrap 規則必須沿同一正常路徑驗證；direct-entry 只能縮小
  問題，不能作完成證據。
- terrain ID、wall art selector 與地名是不同資料層；本 inventory 不替它們猜名稱。

因此本規格目前只授權 Pool game-pack 的 GEO typed adapter 與完整 map inventory，
尚不授權把任一 block 接成玩家出生點。
