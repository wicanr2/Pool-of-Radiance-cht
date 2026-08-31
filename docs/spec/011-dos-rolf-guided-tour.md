# Spec 011：DOS Rolf 34-step 導覽與結束交接

狀態：READY（34-step 位置／朝向、七頁停靠順序、Return 閘門與 ECL EXIT）；DRAFT（每步硬體 wall-clock、Pool 專屬自由移動政策、Rolf 初次 APPROACH 圖像）
日期：2026-08-31

## 固定輸入與位址空間

沿用 Spec 010 的固定來源：DOS ZIP SHA-256
`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`、
`ECL3.DAX` SHA-256
`db58f0c6326d400cc65902933392ce928812c7cd3baf7f68827aed0dff64d076`、
block 0 SHA-256
`b0fe79c56c36d6cf9a7af1bc8e4825f0d195988bc3578072d7486855c4cf0631`。
位址均為 ECL payload base `9900h`；DAX block 的兩-byte prefix 不計入 payload offset。
可重生指令與 packed bytes 見 `cmd/pool-ecl-trace` 及
`docs/audit/dos-ecl3-block0-trace.json`。

## 迴圈與四張 byte table（READY）

初次 Rolf 歡迎頁 Return 後，`B138h` 關閉 sprite、`B139h PICTURE 255`，並在
`B13Ch` 將索引 `6E79h` 清為 0。`B142h..B1A0h` 的迴圈每次執行：

1. 以索引從 `B5B1h`、`B5D3h`、`B5F5h`、`B58Fh` 分別取 X、Y、Facing、頁面 selector。
2. `B1A4h..B1C1h` 把 X／Y 寫入 `C04Bh/C04Ch`，呼叫重畫／delay 路徑。
3. `B171h ON GOSUB` 依 selector 呼叫頁面 routine；targets 精確為
   `B1C1h, B1C2h, B257h, B2F3h, B36Ah, B400h, B48Fh`。
4. selector 不等於 6 時索引加一並回到 `B142h`；等於 6 時跳到 `AE6Ah`。

四張表各 34 bytes：

```text
selector: 0 0 0 0 0 1 0 2 0 0 0 0 0 0 0 0 0 0 0 0 3 0 0 0 0 4 0 0 0 0 0 0 5 6
x:       14 13 12 11 11 11 11 11 11 10 10 10 10 9 8 7 6 5 5 5 5 5 5 5 4 4 4 3 2 2 2 2 1 0
y:        1  1  1  1  1  2  2  2  2  2  2 3 3 3 3 3 3 3 3 2 2 2 3 3 3 3 3 3 3 3 4 4 4 4
facing:   3  3  3  3  2  2  3  0  3  3  2 2 3 3 3 3 3 3 0 0 1 2 2 3 3 2 3 3 3 2 2 3 3 3
```

這些是 byte tables，不是 17 個 little-endian words；若用 word 讀會得到看似合理但錯誤的
`0x0605`、`0x0D0E` 等值。production adapter 必須保留 34-step 形狀並逐 byte 驗證。

## 七頁停靠與文字來源（READY）

selector 1..6 的停靠 step 與頁面如下；完整英文仍於執行期從原版 ZIP 的 packed operands
解碼，不複製進 repository：

| step | 位置／朝向 | selector | 頁面語意錨點 |
|---:|---|---:|---|
| 5 | `(11,2,2)` | 1 | Temple of Tyr |
| 7 | `(11,2,0)` | 2 | passenger docks |
| 20 | `(5,2,1)` | 3 | training schools |
| 25 | `(4,3,2)` | 4 | city hall |
| 32 | `(1,4,3)` | 5 | Sune temple／city park |
| 33 | `(0,4,3)` | 6 | old city gate；其後同一 routine 顯示 tour ended／on your own |

每頁後均 `GOSUB AF1Ch`，沿用 Spec 010 唯一的 Return／button 選項。selector 6 的
`B48Fh` routine 內有兩個 `PRINTCLEAR` 與兩個 Return 閘門，不能把兩頁合併。

## ECL 結束與停止線

最後一頁確認後，迴圈在 `B18Ch..B193h` 偵測 selector 6 並跳到 `AE6Ah`；
`AE6Ah..AE7Ch` 將 `4A07h/4A0Fh/4A11h/4A10h` 清為 0，`AE82h PICTURE 255`，
`AE85h EXIT`。因此導覽事件本身確實結束，最終位置是 `(0,4,facing 3)`。

本規格授權 remake 以 34 個位置 frame、六個停靠 selector／七頁文字與最終 EXIT
重建導覽。每步 `CALL 2C90h`、`CALL BA03h`、`DELAY` 的 DOS 硬體 wall-clock 尚未量測；
依專案硬體時序停止線，可採可重現、玩家可辨識的短固定 delay，標為
`hardware-spec approximation`，不得冒稱逐週期一致。

`EXIT` 只證明 ECL handler 交回 adventure controller；它不自動證明 GEO3/block 0
採 bounded、wrapped 或 dungeon-door 哪種移動政策。因此導覽完成後可以呈現最終位置，
但在移動 consumer READY 前仍須停用玩家移動。
