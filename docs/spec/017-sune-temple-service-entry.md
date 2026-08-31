# Spec 017：Sune 神殿服務入口與離開續行

狀態：CONFORMED（服務路由、初始選單、Exit 後 ECL continuation）；READY（Heal 的三種傷勢治療，見 Spec 018）；DRAFT（其餘 Heal／View／Pool／Appraise）
日期：2026-08-31

## 範圍與停止線

本規格只授權正常玩家在 `(1,3,0)` 回答 Sune 的 `YES` 後進入原版神殿選單，並可由
`Exit` 回到同一個 ECL session。它不授權猜測治療價格、HP／狀態修復、復活、捐獻、
金錢 pooled state 或 Appraise 行為；這些項目仍須另以 overlay-04 bytes／DOS runtime
閉合後實作。

## 輸入、工具與位址空間

- DOS ZIP SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `ECL3.DAX` member SHA-256：
  `db58f0c6326d400cc65902933392ce928812c7cd3baf7f68827aed0dff64d076`；
  block 0 SHA-256：
  `b0fe79c56c36d6cf9a7af1bc8e4825f0d195988bc3578072d7486855c4cf0631`。
  ECL 位址空間是 decoded payload `block.Data[2:]` 映射至 `9900h`；不可把兩-byte
  DAX prefix 算進程式位址。
- `overlay-03.bin` SHA-256：
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`。
- `overlay-04.bin` SHA-256：
  `d948ce6bc533470ac1fa44a7787c2ce5006462cc33ada1cf61ffd128da8a91af`。
- `overlay-25.bin` SHA-256：
  `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e`；
  `overlay-26.bin` SHA-256：
  `cbdeafcf4aaee1fee7a87c858adba4b345e1ddf91391adef5fd8c6358a98930f`。
- IDA Pro 9.4、image `ida-pro-9.4-idapython:locked-v1`，metapc 16-bit raw binary；
  overlay 位址均為 local file offset、base 0。TPOV entry／stub 來自
  `docs/audit/dos-ovr-manifest.json`。

## ECL 分支（exact）

Sune YES 是原始 HORIZONTAL MENU 的 0-based index 0。選擇後由 `AA63h` 執行：

```text
AA63  1C                         CLEARMONSTERS
AA64  09 00 01 01 E2 6D         SAVE 1 -> 6DE2h
AA6A  24                         COMBAT
AA6B  09 00 FF 01 E1 6D         SAVE FFh -> 6DE1h
AA71  0E 00 FF                   PICTURE 255
AA74  00                         EXIT
```

因此 VM 必須在同一 continuation 中保留 `CLEARMONSTERS` 訊號與 `6DE2h=1`，並在
`COMBAT` 邊界停下交給 Pool adapter；不得先執行 `AA6Bh` 之後才回報服務。

NO 分支仍由 Spec 016 的 `AA38h..AA62h` 負責，顯示 `THEN YOU MUST LEAVE.`、保存位置、
`SAVE FFh -> 6DE1h`、`CALL 2C90h`、`EXIT`。`CALL 2C90h` 在 Rolf 導覽每一步也出現，
只能列為 redraw／movement service 候選，不能命名成神殿服務。

## DOS executable 路由

TPOV manifest 與 raw far-call 掃描得到：

1. opcode `24h COMBAT` handler 位於 overlay-03 entry 50、`2E90h`；它呼叫 resident
   stub `010A:00F2`，由 MZ header `0x3B0` 換算 executable file offset `0x1542`，
   精確反查 overlay-25 entry 42、`2C81h`。
2. overlay-25 entry 42 再呼叫 overlay-26 entry 3；這條鏈是 combat／service
   orchestration，但 entry 42 本身是通用選擇／鏈結處理，不得命名成 temple dispatcher。
3. 對 38 份 overlay 的 `9A off16 seg16` 原始 bytes 全掃，只有 overlay-03 `18B9h`
   的 `9A 25 00 35 00` 指向 overlay-04 entry 1。該 callsite 位於 opcode `15h`
   VERTICAL MENU handler（overlay-03 entry 35，`186Ch..19C8h`）：
   `18A2h..18B4h` 比較 runtime state `es:[di+5C4h] == 1`、清成 0，接著呼叫
   overlay-04 entry 1。
4. ECL `6DE2h` 與 runtime state `+5C4h` 的作品內映射目前為 `strong inference`，
   依據是同一服務鏈、值 `1`、單次消費後清零，以及位址差固定為 `681Eh`；尚未取得
   operand resolver 或 runtime watchpoint 的直接映射證據。因此文件不得把這一個
   位址換算單獨寫成 `exact`，但 ECL pattern＋唯一 temple call＋原始選單字串已達
   實作服務邊界的最小充分證據。

## overlay-04 選單（字串 exact；初始狀態選擇 strong inference）

overlay-04 entry 1 的 TPOV code range 為 `0CE4h..0F0Ch`。原始 Pascal 字串：

- `0C1Ah`：`Heal View Take Pool Share Appraise Exit`
- `0C42h`：`Heal View Pool Appraise Exit`
- `0995h`：`, how can we help you?`
- `09ACh`：`Heal Exit`

entry 1 在 `0D2Ah` 依服務狀態選用 `0C1Ah` 或 `0C42h`，再進入選擇器。新隊伍尚未建立
pooled money 時使用不含 `Take／Share` 的 `0C42h`，目前是由選項語意、分支形狀與
初始狀態共同支持的 `strong inference`。remake 初始 Sune 選單固定為：

```text
Heal  View  Pool  Appraise  Exit
```

prompt 使用第一位隊員姓名加 `, how can we help you?`；沒有隊員時失敗即關閉，不製造
匿名替代角色。

## Typed 行為與失敗模式

1. 共用 VM 對 opcode `1Ch` 記錄 `MonstersCleared=true`，不把作品位址或 UI 寫入 engine。
2. opcode `24h` 維持 external boundary，由 Pool 明確白名單；只有同一個 boundary
   同時具有 `MonstersCleared` 且 `Memory[6DE2h]==1` 時，Pool 才進入 Sune temple。
   其他 COMBAT 一律繼續 pending，不得誤路由。
3. Temple menu 自己消費方向鍵與 Enter，不把其 index 當作下一個 ECL menu selection。
4. 選 `Exit` 後清除 temple UI，從 COMBAT 的下一條 `AA6Bh` 繼續同一 VM；必須觀察
   `6DE1h=00FFh`、`PICTURE 255`，最後 `EXIT`，且玩家仍位於 `(1,3,0)`。
5. 選 `Heal／View／Pool／Appraise` 在各自 READY 規格完成前維持 fail-closed，顯示尚未
   完成的服務狀態但不扣錢、不改 HP、不跳過 ECL。

## 驗收

- 共用 engine 測試證明 `CLEARMONSTERS` 可與後續 external boundary 聚合，Clone／其他
  opcode 行為不變。
- Pool gamepack 測試使用真實 ECL3/block0，證明 YES 到達 `COMBAT` 時
  `MonstersCleared=true`、`6DE2h=1`。
- Xvfb 正常按鍵測試由標題既有流程走完 Rolf、走到 Sune、選 YES，看見上述五項原始
  temple menu，再選 Exit，證明 `6DE1h=FFh`、事件 `EXIT` 且移動狀態恢復。
- engine 完整測試與 CoAB 現行 adapter 唯讀回歸都必須通過；不得修改 CoAB repository。

## 實作與驗證收據

- engine commit `0819c64`：opcode `1Ch` 設定 `Result.MonstersCleared`，
  `RunUntilEvent` 會把訊號聚合到第一個 external boundary；合成測試證明 VM 在
  opcode `24h` 後、下一條指令前停止。
- Pool 正式相依：
  `v0.0.0-20260831065636-0819c64e451d`；module `h1` 以本機標準 file proxy 產生，並以
  舊 commit `cf52edc` 重算得到既有 go.sum 完全相同值後才採用。
- Pool 真實 ECL／Xvfb 正常按鍵測試通過：Rolf → `(1,3,0)` → Sune YES → 五項神殿
  選單 → Exit；驗得 `6DE2h=1`、離開後 `6DE1h=FFh`、事件不再 pending、位置與朝向
  仍為 `(1,3,0)`。
- Docker 內 engine `go test ./...`、Pool 正式 go.mod 的 `xvfb-run go test ./...`，以及
  CoAB 唯讀掛載下以新 engine 執行 `internal/ecl`、`internal/game` 回歸均通過。
