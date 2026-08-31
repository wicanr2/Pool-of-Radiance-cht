# Spec 016：初始地圖 SearchLocation 與 Sune 第一個事件

狀態：CONFORMED（正常玩家文字與 YES／NO menu；後續服務入口見 Spec 017）
日期：2026-08-31

## 證據與信心

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- ECL3/block0 SHA-256：`b0fe79c56c36d6cf9a7af1bc8e4825f0d195988bc3578072d7486855c4cf0631`。
- 位址空間：payload base `9900h`；entry 0=`9914h`、entry 1=`99EBh`。
- `entry 0=移動前 lifecycle`、`entry 1=移動後 SearchLocation` 的 ABI 與呼叫順序：
  Pool overlay-03 controller 的本作 executable 證據，`exact`；詳見 Spec 022。
- Pool ECL bytes、GEO3/block0 terrain、VM 執行結果與正常按鍵路徑：`exact`。

## 可重生掃描

`cmd/pool-initial-cell-sweep` 先跑完整 Rolf handler 到 `(0,4,3)`，再對每個
`x=0..15`、`y=0..15`、`facing=0/2/4/6` 複製 VM。每個樣本依序：

1. 對同一個目標格同步 `C04B..C04F`，隔離執行 entry 0；
2. entry 0 正常 `EXIT` 才以同一副本執行 entry 1；此為 cell corpus 掃描，不冒充
   原版「舊格 entry 0 → 移動 → 新格 entry 1」的時間序列；
3. 空 `PRINTCLEAR` 與 `PICTURE` 分列為 observed presentation，不計作玩家事件；
4. 另以 `CanMoveDungeonWrapped` 從 `(0,4)` 做純幾何 BFS。

報表固定為 `docs/audit/pool-initial-cell-sweep.json`：1,024 個 entry 0 樣本全在
`997Dh EXIT`；entry 1 在 seed 1 下得到 840 `EXIT`、156 event、28 error。錯誤為
opcode `0Ah` 12 筆、opcode `20h` 16 筆，未被 passthrough。幾何可達 226／256 格；
BFS 不執行途中 ECL，不能證明劇情狀態下的完整玩家可達性。

## RANDOM 與第一個正常事件

entry 1 在 `9A0Eh` 執行 `RANDOM 19 → 6E79h`。共用 engine 的 opcode `08h` 採原版
inclusive `[0,maximum]` 契約、同 VM 連續 RNG stream，Clone 保留並隔離 continuation。
它是核心語意，不是 Pool passthrough。

正常按鍵測試由 Rolf `EXIT` 後執行：

```text
(0,4,3) → 左轉至 2 → 前進 (1,4,2)
        → 左轉兩次至 0 → 前進 (1,3,0)
```

第一步的 RANDOM 為 1、正常 `EXIT`；第二步為 7。`(1,3)` 的原始 terrain 是 `87h`，
entry 1 經 `PICTURE 24` 後於 `A0D0h PRINTCLEAR` 顯示：

`YOU ARE WELCOMED BY PRIESTESS JOY OF SUNE.`

前端會消費空清畫面／圖片事件並續跑；非空文字會顯示 dialogue、停止移動並等待
Return。下一段原始文字是 `' DO YOU SEEK HEALING?'`，其後 `AE5Ah HORIZONTAL MENU`
提供 `YES／NO`；前端可用方向鍵選擇並以 Enter 提交原始 0-based index。本規格到選單
提交為止已完成；YES 後的服務路由、神殿選單與 Exit continuation 由 Spec 017 管理，
治療規則不屬於本規格。

## 驗收

- sweep JSON 可重生且有固定 1,024 筆、boundary counts 與 `(1,3,0)` 文字 anchor；
- Docker／Xvfb 正常按鍵測試從標題後既有流程走過 Rolf、兩次前進並停在上述文字；
- 不修改 CoAB game pack；共用 engine 完整測試與 CoAB 唯讀回歸必須通過。
