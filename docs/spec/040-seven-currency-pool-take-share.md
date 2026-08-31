# Spec 040：七種貨幣的 View／Take／Pool／Share

狀態：READY。日期：2026-09-01。

## 範圍與證據

本規格只閉合戰利品服務的七欄錢包、全隊池及四個玩家操作；物品鏈仍由 Spec 034／035，
商店價格與寶石／珠寶鑑價另立規格。

- `overlay-05.bin` SHA-256：
  `900ea1b8e57b03e0f6dd1c16024a686c8474e2b0ae9674c730d5619679809c16`。
  `0E85h` 的 `P`／`S`／Take→`M` far call 原始 bytes 分別是
  `9A 39 00 D9 00`、`9A 43 00 D9 00`、`9A 48 00 D9 00`。
- MZ header `3B0h`；因此 `00D9:0039/0043/0048` 的 START file offsets 是
  `1179h/1183h/1188h`。`docs/audit/dos-ovr-manifest.json` 將它們精確反查到
  overlay-21 entry `5/7/8`、code `0506h/062Eh/0C9Eh`。不得再把 segment:offset
  數字誤當 overlay-05 local `0DC9h/0DD3h/0DF6h`。
- `overlay-21.bin` SHA-256：
  `8324f5875ca3c4a35083fc2bd599bd50b8d7f5551882b163275fcbce51b3fe26`。
  `docs/audit/ida-overlay21-money-services.json` 由 IDA Pro 9.4、位址空間
  overlay-local file offset/base 0 非破壞性重生；保留 entry seed、bytes、operand 與
  原始函式名。下列控制流亦以 raw bytes／TPOV manifest 交叉驗證。

## typed 資料（exact）

- 全隊池是七個連續 32-bit unsigned 數：`DS:6752h + 4*i`，`i=0..6`。
- 每位角色是七個連續 16-bit unsigned 數：角色 record `+88h + 2*i`。
- overlay-21 `0B46h..0C64h` 解析顯示文字第一個字母；`G` 再看下一字母：
  `Copper=0, Silver=1, Electrum=2, Gold=3, Platinum=4, Gems=5, Jewelry=6`。
- 角色 record `+102h` 是現行負重；`0000h` helper 回傳該角色的負重上限，`0020h`／
  `003Ch` 分別扣除／增加現行負重。每一枚錢幣、寶石或珠寶在這套服務都增加一單位負重。
- active party 由 `DS:5CF4h` linked list 巡覽；`+84h` 等於 `00h` 或 `B3h` 才納入
  Pool／Share 與隊員數，鏈結欄位是 `+104h`。

## Pool（overlay-21 entry 5，`0506h..05D7h`）

對每位 active character、每欄 `i=0..6`：

1. `pool[i] += wallet[i]`，32-bit 不截斷；
2. `currentLoad -= wallet[i]`；
3. 七欄完成後把角色 record `+88h` 起的 14 bytes 清零。

非 active record 不變。`DS:6CB4h` 依池是否非空更新，屬呈現／可用性旗標。

## Share（overlay-21 entry 7，`062Eh..09E8h`）

1. 先計 active character 數 `n`；`n=0` 必須失敗即關閉，不能除以零。
2. 每欄先算 `share=floor(pool[i]/n)`、`remainder=pool[i]%n`。
3. 依欄位 `6→0`、角色 linked-list 順序發放。每位先嘗試 `share`；若負重空間不足，
   只發可容納數量。仍有 remainder 時，再逐位各發一枚且同樣檢查負重。
4. 所有角色處理後，未能發出的數量保留在 32-bit `pool[i]`；不丟棄、不轉換幣別。

## Take Money（overlay-21 entry 8，`0C9Eh..0F2Ah`）

1. 只為非零 pool 建立選項，依 `6→0` 顯示名稱與數量。
2. 玩家先選幣別，再選 active character，再輸入 `0..pool[i]` 的十進位數量。
3. `0A70h..0B0Ah` 先以 `0058h` 容量 helper 檢查；超重時顯示原版錯誤且不改任何值。
4. 成功時原子執行 `pool[i]-=amount`、`wallet[i]+=amount`、`currentLoad+=amount`。
5. 取消、零數量、無有效角色或 uint16 wallet 溢位均不得造成部分 mutation。

## View 與主選單

Spec 034 已證明主選單依 money／item presence 組成 `View Take Pool Share` 等原順序；
`0F2Bh..0F79h` 對七個 32-bit pool 與 item chain 分開判斷。View 必須依固定七欄順序顯示
名稱與非零數量；它不改 state。Take 同時有 Money／Items 時先顯示該二級選單。

## 存檔與驗收

- schema 5 必須保存每角色七個 uint16 wallet 與七個 uint32 pooled amount；schema 1..4
  的 `gold`／`pooled_gold` 無歧義遷移到 index 3，其他欄為零。
- 遷移、Pool、Share（含 remainder／超重）、Take（成功／取消／超重／溢位）、原子保存
  失敗回滾都要有 deterministic 測試。
- 真實 Spec 039 墓園 request 必須能進入 money service，離開後才續跑原 ECL ack。

## 證據等級與停止線

上述欄位、名稱、順序、公式與 mutation 均為本作 bytes 支持的 `exact`。文字排版可先採
現行 remake panel，但不得宣稱逐像素 parity；寶石／珠寶價值、鑑價及商店兌換不在本規格，
不能為了完成本服務自行猜值。
