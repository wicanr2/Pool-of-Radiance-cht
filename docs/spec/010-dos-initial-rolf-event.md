# Spec 010：DOS 初始 Rolf 導覽事件

狀態：READY（首次觸發、第一頁文字來源、Return 閘門與事件設定、初次 APPROACH 圖像）；後續導覽已由 Spec 011 取代並升為 READY；DRAFT（自由移動政策）
日期：2026-08-31

## 輸入與可重生證據

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- ZIP member：`poolrad/ecl3.dax`，SHA-256：
  `db58f0c6326d400cc65902933392ce928812c7cd3baf7f68827aed0dff64d076`。
- ECL3 block 0 SHA-256：
  `b0fe79c56c36d6cf9a7af1bc8e4825f0d195988bc3578072d7486855c4cf0631`。
- 位址空間：ECL3 block 0 payload base `9900h`；檔內 block 的兩-byte prefix 不屬於
  ECL code payload。
- `cmd/pool-ecl-trace -archive 3 -block 0 -entry 4` 直接從固定 ZIP 重生
  `docs/audit/dos-ecl3-block0-trace.json`。報告保留 offset、ECL 位址、opcode、
  operands、控制流 edge，以及原始 packed text 的 hex；長篇原作文案不複製進規格。

## 正常 Begin 後的首次事件（READY）

Spec 009 已閉合新隊伍由第五 command-set entry `9AF2h` 進入
`GEO3/block 0, (15,1), facing 6`。同一 entry 的首次事件鏈如下：

1. raw payload offset `0240h`／ECL `9B40h` 執行
   `COMPARE [4AC5h], 1`；新 party record 已由初始化清為 0。
2. offset `0246h` 是 `IF <`，offset `0247h` 的 `GOTO` 目標為 `B06Eh`；因此首次
   Begin 立即進入 handler，而不是先取得自由移動控制。
3. handler `B06Eh` 將 `4AC5h` 設為 1。offsets `178Eh/1794h/179Ah` 的三個
   `SAVE` 將位置 special variables `C04Bh/C04Ch/C04Dh` 設成 `(15,1,3)`，故事件
   畫面朝向 3，而不是 Spec 009 的進圖初值 6。
4. offset `17AAh` 的 `SETUP MONSTER` 是 `12, 2, 9`；接著有兩次 `APPROACH`。
   三個 operand 的語意與那張半身像已由 **Spec 117** 閉合：`12` 是
   `SPRIT3.DAX` 的區塊、`2` 是接近距離、`9` 是 `BODY3.DAX` 的區塊，
   head 是 `HEAD3.DAX` 區塊 8，兩張疊起來蓋滿第一人稱內框 `(24,24)` 起的 88×88。
   不再是「猜一張 Rolf 圖」——那張圖與原版畫面逐格相同。
5. offset `17B6h` 的 `PRINTCLEAR` 是 Rolf 歡迎新隊伍並邀請遊覽 Phlan 的第一頁。
   production adapter 必須從使用者本機原版 ZIP 解 packed text，不把完整原作文案
   複製進 repository。驗收只固定 `ROLF`、`PHLAN` 語意錨點。
6. handler 經 `GOSUB AF1Ch` 進入 offset `161Ch` 的 menu；它只有一個選項，精確文字為
   `PRESS <RETURN> OR BUTTON TO CONTINUE`。因此第一頁只接受 Return／等價確認，不能
   擅自改成自由移動。

以上由固定 bytes 與 decoder 直接閉合，推論等級為 `exact`。本 READY 範圍授權
game pack 暴露 typed initial-event adapter，並授權正常 `B` 路徑顯示第一頁及等待
Return。因故事位址、monster ID 與文案均是 Pool 資料，不可移入共用 engine。

## 後續切片交接（已由 Spec 011 閉合）

Spec 011 已逐 byte 閉合 34-step scripted movement、七頁停靠與最後 ECL `EXIT`，並由
正常玩家路徑實作。greeting 的 Rolf APPROACH 圖像已由 Spec 117 閉合並實作；
仍未閉合的是 ECL `EXIT` 後 adventure controller 採用的 Pool 專屬移動政策，
因此事件可完成，但玩家自由移動仍停用。
