# Spec 039：墓園七種戰利品累積池

狀態：READY（ECL 累積量 → `TREASURE` 請求 → ack）；貨幣服務由 Spec 040 READY。
日期：2026-09-01。

## 證據與勘誤

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `docs/audit/dos-ecl-reward-source-refs.json` 由 `cmd/pool-ecl-memory-audit` 從唯讀 ZIP
  重生；`4A39h..4A3Fh` 的可達直接寫入集中在 `ECL4.DAX/block10` 墓園流程，另有
  `ECL3.DAX/block8 9C58h` 的結算讀取。三個無法完整解碼的 block 仍明列為下界缺口。
- 真實 `ECL3/block8` 從 `9C34h` 逐槽只注入一單位的 runtime 正對照，七槽都顯示
  `ELIMINATED SOME UNDEAD FROM THE GRAVEYARD` 與 `HERE IS YOUR REWARD`，然後送出
  單一 `TREASURE` 請求。這推翻 Spec 028 舊有「七項 reward／quest」解釋。

## READY 契約

對索引 `i=0..6`：

1. 讀 `delta = memory[4A39h+i] - memory[4A8Fh+i]`；非正值不產生戰利品。
2. 正值依 `9C83h ON GOSUB` 送出一筆 `TREASURE`。七槽的 amounts 目的索引固定為
   `3,4,5,6,4,5,6`；`ItemBlock=FFh`。
3. 服務返回後，`9C9Eh SAVE TABLE` 才把 `memory[4A39h+i]` 寫入 `4A8Fh+i`。
4. 這段不改寫 `4AC1h`；`4AC1h` 的十個 producer 位於後段 `9FAEh..A4D1h`。

以上地址、順序、請求內容與 ack 為 `exact`。七欄名稱、角色錢包、全隊池與
View／Take／Pool／Share 的本作 consumer 已由 Spec 040 的 overlay-21 原始 bytes 閉合。

## 驗收

- 真實 block8 七個獨立 VM fixture 各只設一槽為 1；逐一斷言共同墓園文字、唯一
  `TREASURE` request、精確 amounts 索引、對應 ack=1、`4AC1h=0`。
- 產品依 Spec 040 保留七種數量，不得先折成單一 `PooledGold`；服務結束前不得讓 ECL
  跨過 `9C9Eh` 清除 ack。
