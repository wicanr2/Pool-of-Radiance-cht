# Spec 138：戰鬥裡的 B）ANDAGE——止血，不看距離

狀態：READY（DONE 子選單的六項、`Bandage` 出現的條件、包紮的對象、效果與
「用掉這個行動」都從 overlay-08 讀出並實作；`TestBandageStopsTheBleedingAndUsesTheTurn`
從 `Update()` 送鍵驗）；DRAFT（runtime `+13h` 那個閘門欄位的語意）。
日期：2026-09-15。主台帳：GitHub issue #25。

## 一句話

倒地（狀態 5）的隊員每回合計數加一、超過 9 就死（spec 062）。B）ANDAGE 把**第一個**
倒地的我方改成昏迷（狀態 4）並把計數歸零，代價是包紮者這一個行動。原版**不看誰站在
哪裡**：說明書 p.41 寫「靠近該瀕死夥伴的隊員」，但程式從隊伍串列頭走到尾，只比陣營
與狀態。

## 輸入

- `overlay-08.bin` SHA-256 `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f`
  （`docs/audit/dos-ovr-manifest.json`），IDA 9.4，overlay-local offset。
- 匯出：`docs/audit/ida-overlay08-done-menu.json`（`0E2Fh..0FE9h`）、
  `docs/audit/ida-overlay08-bandage.json`（`0FE9h..1092h`）。
- 說明書：`docs/reference/manual/manual-vol2.md` 的 DONE 段（p.41）與狀況表（p.30）。

## DONE 子選單：overlay-08 `0E2Fh`（exact）

簽章 `f(結果: far byte*, 記錄: far)`，`retf 8`。先組指令列再進選單迴圈：

| 段 | 字串位址 | 接上條件 |
|---|---|---|
| `Guard ` | `0DE8h` | overlay-25 entry 43（`10Ah:0F7h`）回 0，或 entry 44（`10Ah:0FCh`）回非 0 |
| `Delay ` | `0DEFh` | 無條件 |
| `Bandage ` | `0DFBh` | **`0FE9h(0)` 回非 0**——有可包紮的人才列出來 |
| `Speed Exit` | `0E04h` | 無條件 |

按鍵分派（`0F5Dh..0FD4h`）：

| 鍵 | 做什麼 |
|---|---|
| `G` | overlay-25 entry 35（`10Ah:0BFh`）→ 結果 |
| `D` | runtime（記錄 `+108h`）的 `+3` = 1，結果 = 1（spec 062 的 Delay）|
| `Q` | overlay-25 entry 34（`10Ah:0BAh`，「一個行動用掉了」，spec 111）→ 結果 |
| `B` | `0FE9h(1)` → 結果；**接著同 `Q`**：overlay-25 entry 34 → 結果 |
| `S` | 近呼叫 `10D9h`（GameSpeed 子選單，`Slower Faster Exit`）|

所以包紮完這個角色的行動就結束了，與 QUIT 走同一支。

## 包紮本體：overlay-08 `0FE9h`（exact）

簽章 `f(套用: byte)`，`retf 2`，回傳「有沒有找到」：

```
0FF3  游標 = DS:5CF4h                      ; 隊伍串列頭（spec 030／040）
0FFF  游標為 0 → 回傳 找到
100A  runtime = 記錄 +108h
100F  runtime +13h != 0 → 下一個            ; 閘門，語意未讀（DRAFT）
1019  記錄 +10Eh != 0 → 下一個              ; 陣營：只包我方（spec 056）
1024  記錄 +10Ch != 5 → 下一個              ; 只包倒地的
102C  找到 = 1
1030  套用 == 0 → 下一個                    ; 帶 0 只是問「有沒有」
1039  記錄 +10Ch = 4                        ; 倒地 → 昏迷
1047  runtime +0Eh = 0                      ; 倒地計時歸零（spec 062 第 4 步加的就是它）
105C  印 「<名字> is bandaged」（0FDDh）
106C  套用 = 0                              ; 一次只包一個
1070  游標 = 記錄 +104h                     ; 下一個
```

沒有任何座標比較、沒有 `0138h:003Eh` 鄰近查詢（spec 056）。生命值不動：說明書
p.30 的狀況表說包紮只是止血，「狀況改善成不省人事」。

## READY 契約

1. 指令列有沒有 `Bandage` 由「我方有沒有狀態 5 的人」決定，不由距離決定。
2. 包紮對象是串列裡**第一個**狀態 5 的我方，一次一個；玩家不挑人。
3. 效果：`+10Ch` 5 → 4、倒地計時歸零，生命值不變。
4. 包紮用掉包紮者這一個行動（與 QUIT 同一支收尾）。

## 實作對應

`cmd/pool-game/tactical.go`：`bandageTarget`／`bandage` 與 `B` 鍵；訊息
`ui.statusBandaged`。測試 `cmd/pool-game/bandage_test.go`。測試駕駛
（`tactical_pilot_test.go`）有隊友倒地就按 B——旁邊有敵人時撐到計時 6 才包。
