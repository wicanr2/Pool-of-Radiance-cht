# Spec 028：City Hall reward／commission 迴圈

狀態：DRAFT；日期：2026-09-01。

## 目標與停止線

本規格要閉合 clerk 第一頁之後的 reward 掃描、獎勵演出、commission 列舉、條件旗標
更新與離開 City Hall。Spec 027 只保證 `35h SAVE TABLE` 及其後第一個文字 boundary；
正常新隊伍目前能看到三項預設委託，不足以證明其他劇情狀態、獎勵、接受／拒絕分支或
外部服務已完成。

## 固定證據

- DOS ZIP、`ECL3.DAX` 與 ECL3/block8 SHA-256、`9900h` raw payload 基準沿用
  Spec 025／026／027。
- `docs/audit/dos-ecl3-block8-trace.json` 由
  `cmd/pool-ecl-trace -archive 3 -block 8` 直接從唯讀 DOS ZIP 重生；它保留五個 entry、
  每條 instruction、operand、packed text、menu 與控制流邊。
- `docs/audit/dos-city-hall-structure.json` 由 `cmd/pool-city-hall-audit` 從上述 trace
  重生，並以 block hash 失敗即關閉。它分別保留 reward 的七個**有序槽位**與四個
  唯一目的位址、commission 的十六個有序入口、十個 `4AC1h` 增量點，以及本範圍的
  external service 呼叫；不得用去重後的目的位址數取代槽位數。
- 本輪只把 trace 可直接支持的位址、讀寫與文字列為 `exact`；任務名稱與旗標意義若
  尚未追到 producer／consumer，維持 `strong inference` 或 `unknown`。

## 已證實的 reward scan 骨架（exact）

clerk 第一頁 `9BB8h` 後：

1. `9C34h` 令索引 `6E79h=0`，`9C3Ah` 清本輪是否有 reward 的暫存 `4A00h`。
2. `9C58h GETTABLE 4A39h,[6E79h] -> 6E7Ah`；`9C62h GETTABLE 4A8Fh,[6E79h] -> 6E80h`。
3. `9C6Ch` 算兩表差值到 `6E81h`；正差時 `9C7Dh` 令 `4A00h=1`。
4. `9C83h ON GOSUB` 依索引進七個 reward 子程式，再由 `9C9Eh SAVE TABLE` 把
   `6E7Ah` 回寫到 `4A8Fh + [6E79h]`。
5. `9CA8h` 索引加一，`9CB1h..9CB8h` 在索引小於七時回圈。

這證明 `4A39h` 與 `4A8Fh` 是七槽比較／acknowledgement table；每槽對應的劇情來源、
reward 數值與是否所有子程式都會有玩家輸出仍待逐槽閉合。

## `4AC1h` producer 勘誤

先前 WORKLIST 把 `4AC1h` 寫成「clerk 實際授予 commission 的 producer 尚未定位」。
block8 trace 已推翻這個單一 producer 模型：下列 reward 子程式都以
`ADD 1,[4AC1h] -> [4AC1h]` 更新它：

`9FAEh, 9FF3h, A179h, A1BBh, A201h, A24Ch, A304h, A342h, A37Fh, A4D1h`。

相鄰文字分別涉及 Norris the Gray、Sokal Keep、Podal Plaza、graveyard、Kovel Mansion、
Stojanow River、lizardmen、kobolds、nomads 與 slums；文字與增量位址為 `exact`，
「這十項就是完整 proclamation 進度語意」目前僅為 `strong inference`，尚須把每個
入口條件與 Spec 024 的 `1..9` 分派邊界對齊。不得再找一個不存在的 clerk 單點 SAVE。

## 已證實的 commission 列舉骨架（exact）

- 初次進入由 `A7A4h` 顯示 `ON THE MATTER OF COMMISSION...`；`4A06h` 控制第一次與
  `A80Fh BACK SO SOON...` 的重入文字。
- `A83Fh` 將索引 `6E79h=0`；`A85Bh ON GOSUB` 分派 16 個 commission 子程式。
- 每輪 `A891h` 增加 `4A05h`、`A89Eh` 增加索引 `6E79h`，再回到 `A845h` 的上限與
  sentinel 判斷。各子程式依作品旗標決定顯示、更新狀態或返回。
- 預設新隊伍的正常按鍵路徑顯示 Slums、Sokal Keep、古 Phlan 書籍／地圖三項，最後
  由 `AF4Bh` 顯示 `THESE ARE ALL OF THE COMMISSIONS CURRENTLY AVAILABLE`，
  `AF78h GOSUB` 後 `AF7Ch EXIT`。

## 尚未 READY 的行為

- 七個 reward table slot 的來源旗標、ack 值、金錢／物品／經驗等副作用與重入規則。
- 16 個 commission 子程式的完整條件矩陣、狀態 producer／consumer 與重入結果。
- reward 區另在 `9F28h`／`9F3Eh` 呼叫 `TREASURE`／`COMBAT`；這是本輪機器清冊補出的
  舊規格缺項。graveyard 特別委託則會走 `A5A8h PARTYSTRENGTH`、接受選擇、
  `A780h TREASURE` 與 `A791h COMBAT` 服務；
  這些外部 opcode 不能靠 passthrough 略過，須各有 READY 證據與 typed adapter 契約。
- `AF78h` 呼叫的離場 helper 尚未分類；目前正常路徑能安全到 `EXIT`，不代表 helper
  的所有可見副作用都已證實。

## READY 前驗收設計

1. 產生機器可讀的七槽 reward 與 16 路 commission 表：條件讀取、寫入、文字、
   external opcode、return target 與 evidence grade，不手抄一份不可重生表格。
2. 為預設新隊伍建立精確控制組，逐 boundary 斷言三項委託與 `AF7Ch EXIT`。
3. 為每個條件分支用獨立 VM clone／save fixture 執行真實 ECL；至少各含正對照與
   對調旗標的負對照，不能只取所有路徑文字聯集。
4. 對會進 external service 的分支先另立 READY opcode／adapter spec，再接正常玩家
   路徑；不得用 direct-entry 結果宣稱整段可玩。
5. 驗證 `4AC1h` 增量與 Spec 024 proclamation 分派的範圍；若十個 producer 可使值超過
   九，必須找出原版 cap／reset／排他條件，不能自行截斷。
