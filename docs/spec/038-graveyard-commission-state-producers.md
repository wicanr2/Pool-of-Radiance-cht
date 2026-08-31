# Spec 038：墓園委託旗標 producer 清冊

狀態：DRAFT（producer 條件與正常主線）；CONFORMED（目前可解 ECL 的直接 operand 引用）。
日期：2026-09-01。

## 可重生證據

`cmd/pool-ecl-memory-audit` 從唯讀 DOS ZIP 掃描八份 ECL archive 的 reachable graph，
輸出 `docs/audit/dos-ecl-campaign-memory-refs.json`。每列保留 archive、block、runtime
address、opcode、operand index／code 與 raw target；不把欄位名稱寫進共用引擎。

清冊同時明列三個仍無法完整解碼的 block（ECL5/7、ECL7/17、ECL7/22），所以目前結果
是可重生下界，不可用來宣稱間接存取或未知尾段「不存在」。

## 已證實引用

- `4AC1h`：ECL3/block8 有十個 `ADD 1,[4AC1] → [4AC1]` producer（Spec 028 已列位址），
  墓園分支 `A592h` 要求值至少 4。ECL3/block0 `AC55h/AC60h` 讀取並減一後選公告。
- `4A96h`：墓園分支 `A5B7h` 排除 `FFh`；玩家接受並離開戰利品服務後，`A792h`
  精確執行 `SAVE FFh → 4A96h`。因此它至少是本次委託的重入閘門；更完整語意仍待
  其他 consumer 交叉驗證。
- `4AB1h`：墓園分支 `A59Dh` 排除 `FFh`。目前可達 ECL 唯一直寫在
  ECL4/block10 `B177h`，於 `THERE IS A VAMPIRE HERE.` 的 monster setup／COMBAT、
  結果 helper 與 `6E79h == 2` 分支後寫入 `FEh`。這證明欄位與吸血鬼事件相連，
  但 `FEh` 仍通過墓園條件，故不得把它草率命名為「吸血鬼已死」或墓園前置完成旗標。

## 下一個實作閘門

1. `4A39h..4A3Fh` 已另由 memory audit 與真實 block8 逐槽執行證明為墓園戰利品
   累積量，不是 `4AC1h` 的七個任務 producer；七種貨幣服務已由 Spec 040 閉合。
2. Spec 041 已證明 `9FAEh..A4D1h` 是 `4AA6h..4ABFh` 26 槽完成通知表中的十個增量
   分支。下一步改追各狀態槽真正的 `FEh` producer、external services、寫入後值與
   重入結果；墓園 gate 需要至少四條正常可達垂直鏈。
3. 找出 `4AB1h` 初值與所有可能的 `FFh` producer，包括 SAVE TABLE、間接位址及三個
   decoder 缺口；未完成前只保留 raw address。
4. 只有玩家可正常完成足以令 `4AC1h >= 4` 的任務鏈後，才可用正常 City Hall 按鍵
   驗收墓園委託；禁止測試直接注入 4 當成完整主線證據。
