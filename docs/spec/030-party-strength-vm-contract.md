# Spec 030：`1Dh PARTYSTRENGTH` VM 契約

狀態：CONFORMED（命令 framing、逐隊員公式、byte 寫回與 typed resolver）；
DRAFT（Pool 完整角色投影）。日期：2026-09-01。

## 輸入、工具與位址空間

- DOS `overlay-03.bin` SHA-256：
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`。
- IDA Pro 9.4、image `ida-pro-9.4-idapython:locked-v1`、metapc 16-bit raw binary；
  位址是 overlay-local file offset、base 0。
- `docs/audit/ida-overlay03-ecl-dispatcher.json` 的 `3404h..340Dh` 以 `cmp ax,1Dh` 將
  opcode 路由到 `13A0h`。
- `docs/audit/ida-overlay03-op1d.json` 是從同一原始 binary 以
  `tools/ida-export-overlay-functions.py`、seed `13A0h` 非破壞性重生的 handler；保留
  input hash、IDA 版本、原始 bytes、operand 與 `13A0h..14B1h` 函式邊界。
- City Hall 真實 consumer 是 ECL3/block8 `A5A8h`，由固定 block hash
  `fd446439973994d9f887b8628b512054a822369eb4360769b21eb2224982f012` 支持。

## 原版 handler（exact）

1. `13A6h..13ADh` 準備一個 ECL operand；它是結果 destination。
2. `13AEh` 將 byte accumulator 清零，由 `DS:5CF4h` party head 開始，沿每筆 record
   `+104h` far pointer 走到 nil。
3. 每名隊員依序讀 `+11Bh`、`+111h`、`+110h`、`+9Bh`、`+96h` 五個 byte。
4. `+111h > 60` 時取 `+111h-60`，否則取 0；`+110h > 39` 時取 `+110h-39`，
   否則取 0。
5. 每名隊員貢獻為：

   ```text
   (+11Bh + 5*adjusted(+111h) + 5*adjusted(+110h) + 8*(+9Bh) + 4*(+96h)) / 10
   ```

   `idiv 10` 是整數除法。`1473h..147Ah` 把 quotient 加回 byte accumulator，因此總和
   採 8-bit wrap，不是 clamp。
6. `1490h..14A9h` 解析 destination，將 zero-extended accumulator 寫回 ECL memory；
   `14B1h retf` 後直譯器繼續下一條命令。

同系列 CoAB Spec 264／265 的獨立 consumer 將五欄命名為 current HP、stored AC、stored
attack bonus、magic-user level、cleric level，公式逐項相同。對 Pool 而言，位址、bytes
與公式為 `exact`；五個名稱目前是第二作品交叉支持的 `strong inference`，不得用名稱
取代 Pool 自己尚未完成的角色欄位證據。

## 共用 engine typed 契約

1. `PARTYSTRENGTH` 必須接受一個 address-form destination；立即值或無法取址者失敗即關閉。
2. 共用 VM 提供 title-neutral `PartyStrengthResolver func() (uint8,error)`。命令執行時
   同步呼叫 resolver，把 byte zero-extend 後寫入 destination，留下正常 `Write` receipt，
   並在 `Result.PartyStrengthRequests` 保存 destination 與 value。
3. resolver 缺席或回錯時失敗即關閉，不前進至下一條 ECL，不以 passthrough event 代替。
4. `Clone` 可共享 immutable callback，但 memory 與 result receipt 必須隔離。
5. 共用 engine 不含 `5CF4h`、Pool save schema、職業名稱、AC／THAC0 換算或劇情旗標。

## Pool 尚未 READY 的投影

現行 Pool save 只有 HP、能力、class ID，沒有 class-level slots、stored AC、stored attack
bonus 或裝備導出的更新鏈。既有 `BASE.CHA` 只提供單一一級角色 anchor：`+110h=40`、
`+111h=50`、`+11Bh=6`；不能證明所有職業、升級與裝備狀態。故本規格不授權用常數
`40/50` 或單純 party size 代替角色投影，也不授權讓 `A5A8h` passthrough。

## 驗收

- engine synthetic：resolver 值同步寫回、write/request receipt、下一條 compare 可讀到；
  缺 resolver、resolver error、非 address destination 都失敗即關閉。
- Clone：同一 callback 可用，兩邊 memory／receipts 不共享。
- engine 全測試與 CoAB 核心 ECL／遊戲回歸通過。
- Pool 在角色投影 READY 前，真實 `A5A8h` 仍應回報缺 resolver；這是刻意的安全邊界，
  不是可玩完成證據。

## 實作收據

- shared engine `4b10d7c5a302` 已加入 `PartyStrengthResolver`、inline byte writeback、
  request／write receipt、Clone 與 session aggregate；缺 resolver、resolver error 與非位址
  operand 皆失敗即關閉。engine 全套測試通過。
- 正式 pseudo-version 為 `v0.0.0-20260831181703-4b10d7c5a302`；canonical module h1
  `Fj64wjxNea9EGDIPyYyQdrhVfrprpEVzOoC2UqdWOn4=` 已先以舊版已知 h1 交叉驗證算法。
- Pool 全套測試與 CoAB `internal/ecl`、`internal/game`、`cmd/azure-bonds-game` 回歸通過。
