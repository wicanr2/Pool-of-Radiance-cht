# Spec 010：DOS 初始 Rolf 導覽事件

狀態：READY（首次觸發、第一頁文字來源、Return 閘門與事件設定）；DRAFT（後續七頁、導覽位移、肖像／sprite 與自由移動交接）
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
4. offset `17AAh` 的 `SETUP MONSTER` 第一個 operand 是 12；接著有兩次
   `APPROACH`。這證明事件建立 monster 12，但在 PIC／sprite consumer 尚未閉合前，
   不授權 remake 猜一張 Rolf 圖。
5. offset `17B6h` 的 `PRINTCLEAR` 是 Rolf 歡迎新隊伍並邀請遊覽 Phlan 的第一頁。
   production adapter 必須從使用者本機原版 ZIP 解 packed text，不把完整原作文案
   複製進 repository。驗收只固定 `ROLF`、`PHLAN` 語意錨點。
6. handler 經 `GOSUB AF1Ch` 進入 offset `161Ch` 的 menu；它只有一個選項，精確文字為
   `PRESS <RETURN> OR BUTTON TO CONTINUE`。因此第一頁只接受 Return／等價確認，不能
   擅自改成自由移動。

以上由固定 bytes 與 decoder 直接閉合，推論等級為 `exact`。本 READY 範圍授權
game pack 暴露 typed initial-event adapter，並授權正常 `B` 路徑顯示第一頁及等待
Return。因故事位址、monster ID 與文案均是 Pool 資料，不可移入共用 engine。

## 明確停止線（DRAFT）

第一頁 Return 之後，原版尚有七頁導覽文字，依序涉及 Tyr 神殿、碼頭、訓練學校、
市政廳、Sune 神殿／公園、舊城門與導覽結束。這些頁面之間的 scripted movement、
朝向、背景、Rolf 圖像以及最後交回自由移動的狀態尚未逐項閉合。

因此目前 remake 在第一頁確認後必須停在「後續導覽待實作」狀態；不得直接放行移動，
也不得宣稱第一張地圖事件完成。下一份 READY 規格須一次閉合七頁順序、每段位移與最後
交接，避免只接文案卻漏掉 ECL 狀態鏈。
