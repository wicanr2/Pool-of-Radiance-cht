# Spec 032：`27h TREASURE` 八欄請求與墓園順序

狀態：CONFORMED（作品中立 raw request 與墓園 `33h` 正常順序）；DRAFT（ITEM3/33h
物品節點、戰利品分配 UI、金錢欄位名稱與跨存檔持久化）。
日期：2026-09-01。

## 原版證據

- 輸入 `overlay-03.bin` SHA-256
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`；
  IDA Pro 9.4，位址空間為 overlay-local file offset、base 0。非破壞性匯出是
  `docs/audit/ida-overlay03-op27.json`，保留原始 seed、bytes、operands 與位址。
- dispatcher `346Eh` 比對 opcode `27h`，`3474h` 呼叫 handler `1A81h`。
- handler `1A88h..1ACBh` 先要求八個 operand：前七個逐一呼叫同一 numeric resolver，
  寫入七個四位元組槽；第八個再解析成 byte 並分派固定 block、`FFh` 或隨機路徑。
  因此「八欄均先依 ECL numeric operand 規則求值」為 `exact`；七欄的貨幣名稱與
  後續物品節點語意尚未由 Pool consumer 閉合，本規格只稱 `Amounts[7]`。
- 真實 `ECL3.DAX` block 8 在 `A780h` 是
  `TREASURE 0,0,0,0,0,0,0,33h`，下一筆 `A791h` 是 `COMBAT`。這固定墓園分支的
  raw 請求與執行順序；不能從 item block 編號單獨宣稱玩家已拿到哪件武器。

## 共用 VM 契約

1. `TreasureRequest` 保存七個已求值的 `uint16` 數量與第八個 `ItemBlock`，不放入
   Pool 位址、章節或物品名稱。
2. `27h` 是 inline 指令：請求加入本次結果後繼續執行；墓園因此在同一次
   `RunUntilEvent` 停於後續 `24h COMBAT` 外部邊界。
3. 任一 operand 無法作 numeric value 時失敗即關閉，且不得提交半筆請求。
4. `Machine.RunUntilEvent` 與 `BlockSession.RunUntilEvent` 均聚合請求；clone 沒有
   額外可變 resolver state。

## 驗收與停止線

- synthetic 正對照混用 immediate 與 memory operand，固定七欄、`33h` 及
  `TREASURE → COMBAT` 順序；string-memory 負對照必須報出第八欄錯誤。
- 真實 block 8 從 `A780h` 執行，必須得到一筆全零／`33h` 請求，然後停在 `24h`。
- 本輪不把 CoAB 的 ITEM record、隨機物品表或貨幣換算直接當成 Pool 事實；下一輪
  必須先解 `ITEM3.DAX/33h` 的 Pool payload consumer，再建立戰利品 UI 與持久化。
