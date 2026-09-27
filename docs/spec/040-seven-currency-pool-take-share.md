# Spec 040：七種貨幣的 View／Take／Pool／Share

狀態：READY。日期：2026-09-01（2026-09-18 以原版實測修正 `VIEW` 那一段，並補分錢的量測）。

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

1. 份數 n 是 entry 6（`05D8h`）數的人：`+84h` 等於 00h 或 B3h。`n=0` 原版除以零（RTL `05BBh:0294h`
   → 執行期錯誤 200），remake 失敗即關閉。
2. 七欄各自（公款當有號 dword，大於 0 才算）：`share=floor(pool[i]/n)`、`remainder=pool[i]%n`，
   兩者都只留低位字（`06B0h`、`06E1h`）。
3. 第一輪沿隊伍鏈，只發 `+84h` 小於 80h 的人（`0721h`；B3h 算進份數卻不在這一輪，他那一份
   從公款消失）。每人依 `6→0` 逐欄問容量 helper `0058h`（現重＋數量，16 位元相加，大於上限回 1）：
   超過就只發「上限 − 現重」、差額加回餘數；沒超過就發一份，餘數大於 0 時**拿整筆餘數**再問一次
   （`07B6h` 推的是餘數，不是 1），過得了才多給一枚。
4. 第二輪（`087Ah`）依 `6→0`，餘數大於 0 的那一欄沿**整隊**（不看 `+84h`）發給「上限 − 現重」
   無號大於 0 的人，發到餘數用完。
5. 公款每一欄改寫成那一欄的餘數（`09A3h..09C2h`），錢包加法都是 16 位元、會繞回。

超重的人「上限 − 現重」繞回成很大的無號數，原版照繞回的值發；dosgolem 收據
`docs/audit/dosgolem-shop-share-wrap.json`（spec 067〈P 之後直接按 S〉）逐欄對得上這個模型。
remake：`treasure.ShareMoney`／`PoolMoney`（`pooledMember`、`memberControl`：玩家建的角色是 0，
NPC 讀記錄 `+84h`）。

## Take Money（overlay-21 entry 8，`0C9Eh..0F2Ah`）

1. 只為非零 pool 建立選項，依 `6→0` 顯示名稱與數量。
2. 玩家先選幣別，再輸入數量（上限是 `pool[i]` 的低位字，`0EADh`）；錢給目前的角色
   `DS:5CF0h`（`0EC2h`），這一支不問給誰。戰利品選單的 remake 在這一步多問一次給誰（停止線），
   商店與神殿照原版給目前的買家／角色（spec 067〈公款〉）。
3. `0A70h..0B0Ah` 先以 `0058h` 容量 helper 檢查；超重時顯示原版錯誤且不改任何值。
4. 成功時原子執行 `pool[i]-=amount`、`wallet[i]+=amount`、`currentLoad+=amount`。
5. 取消、零數量、無有效角色或 uint16 wallet 溢位均不得造成部分 mutation。

## View 與主選單

Spec 034 已證明主選單依 money／item presence 組成 `View Take Pool Share` 等原順序；
`0F2Bh..0F79h` 對七個 32-bit pool 與 item chain 分開判斷。Take 同時有 Money／Items 時
先顯示該二級選單。

**`VIEW` 不是幣別清單。** 這一份原本寫「View 必須依固定七欄順序顯示名稱與非零數量」，
2026-09-18 用 dosgolem 在原版量到的是**人物頁**（屬性、`GOLD 110`、`LEVEL`／`EXP`、
`AC`／`THAC0`／`ENCUMBRANCE`、`STATUS OKAY`，底列 `VIEW:TRADE DROP EXIT`）；
逐幣別的數量出現在 **`TAKE`** 那一層（`GOLD 250`／`PLATINUM 50`／`JEWELRY 1`，
底列 `SELECT TYPE OF COIN EXIT`）。收據：[`docs/audit/dos-treasure-screens.json`](../audit/dos-treasure-screens.json)。

## 原版實測：五個人分 250 金（2026-09-18）

狀態檔 `workplace/dosgolem-cheat/handin-slums-stuck.state`（市政廳交完貧民窟的件、職員給獎金那一刻），
按 `S` 分錢前後逐一讀隊伍鏈（`DS:5CF4h`，下一個在 `+104h`，**far pointer**：offset 在前、segment 在後；
錢包 `+88h` 起七個 uint16，pool 在 `DS:6752h` 起七個 uint32）：

| 項 | 分錢前 pool | 每人拿到 | 分錢後 pool |
|---|---|---|---|
| 金幣 | 250 | 50 | 0 |
| 白金 | 50 | 10 | 0 |
| 珠寶 | 1 | 第一位 1 | 0 |

**除不盡的那一份不是留在 pool，是給排在前面的人**（珠寶 1 除以 5）。分完之後頂層選單少掉
`TAKE` 與 `SHARE`，剩 `VIEW POOL EXIT`——選項依「還有沒有錢」組成，與 `0F2Bh..0F79h` 的判斷一致。

## 存檔與驗收

- schema 5 必須保存每角色七個 uint16 wallet 與七個 uint32 pooled amount；schema 1..4
  的 `gold`／`pooled_gold` 無歧義遷移到 index 3，其他欄為零。
- Spec 044 後現行格式為 schema 6；上述七貨幣欄位與遷移契約不變，schema 6 只另增
  campaign 的 ECL archive identity。
- 遷移、Pool、Share（含 remainder／超重）、Take（成功／取消／超重／溢位）、原子保存
  失敗回滾都要有 deterministic 測試。
- 真實 Spec 039 墓園 request 必須能進入 money service，離開後才續跑原 ECL ack。

## 證據等級與停止線

上述欄位、名稱、順序、公式與 mutation 均為本作 bytes 支持的 `exact`。文字排版可先採
現行 remake panel，但不得宣稱逐像素 parity；寶石／珠寶價值、鑑價及商店兌換不在本規格，
不能為了完成本服務自行猜值。
