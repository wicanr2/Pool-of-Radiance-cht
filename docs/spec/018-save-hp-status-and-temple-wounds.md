# Spec 018：存檔生命值／狀態與神殿傷勢治療

狀態：CONFORMED（schema 2、schema 1 遷移、三種傷勢治療、付款來源順序與正常神殿路徑）；DRAFT（其他狀態治療）
日期：2026-08-31

## 範圍

本規格先消除 remake 單一 `HP` 欄位無法表達受傷的結構缺口，並只授權 Sune 神殿中
已有原始 bytes 支持的 Cure Light／Serious／Critical Wounds。Cure Blindness、Disease、
Neutralize Poison、Raise Dead、Remove Curse、Stone to Flesh 仍失敗即關閉；尚未閉合的
pooled／個人金錢選擇也不得猜測。

## 證據與位址空間

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `overlay-04.bin` SHA-256：`d948ce6bc533470ac1fa44a7787c2ce5006462cc33ada1cf61ffd128da8a91af`；
  IDA Pro 9.4 raw-binary base 0。Heal entry 11 為 `09B6h..0C19h`。
- `overlay-24.bin` SHA-256：`e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714`；
  dice entry 8 為 `0DE5h..0E2Dh`，HP apply entry 21 為 `175Dh..1845h`。
- `overlay-19.bin` SHA-256：`4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9`；
  entry 11 為 `28A2h..297Bh`。
- `overlay-21.bin` SHA-256：`8324f5875ca3c4a35083fc2bd599bd50b8d7f5551882b163275fcbce51b3fe26`；
  entry 15／16／17 的 code offsets 分別為 `012Eh`／`0183h`／`00ACh`。

## 原版契約

overlay-04 Heal 分派的十項依序為 Cure Blindness、Cure Disease、Cure Light Wounds、
Cure Serious Wounds、Cure Critical Wounds、Neutralize Poison、Raise Dead、Remove Curse、
Stone to Flesh、Exit；exact 價格依序是 1000、1000、100、350、600、1000、5500、3500、
2000 GP。三種傷勢治療呼叫共同常式並傳入：

- Light：`1d8`，100 GP。
- Serious：`2d8+1`，350 GP。
- Critical：`3d8+3`，600 GP。

overlay-24 entry 8 對每顆骰執行 `Random(sides)+1` 並加總。entry 21 讀角色 `+10Ch`
狀態 byte、`+11Bh` 目前 HP 與 `+32h` 最大 HP；一般受傷角色把治療量加到目前 HP，並
封頂於最大 HP。這證明三欄必須分開保存；狀態代碼的完整名稱表尚未閉合，remake 因此
先原樣保存 `uint8`，不以猜測 enum 覆蓋原值。

## 付款來源與順序（exact）

overlay-04 `00BFh..0229h` 在玩家回答 `Y` 後：

1. `016Ah..0177h` 把目前角色 far pointer `ds:5CF0h/5CF2h` 傳給
   `00C9:0057`。MZ header size `0x3B0` 換算 executable file offset 是
   `0x1097`，TPOV manifest 精確反查 overlay-19 entry 11。
2. overlay-19 `28B9h..28E7h` 讀該角色 `+88h` 起五個 16-bit 貨幣欄，乘上
   `0D38h` 的五份換算值後加總；`28F0h..2909h` 再做 Gold 等值換算並回傳。
3. 若角色金額足夠，overlay-04 `0184h..0194h` 扣完整費用後呼叫
   `00D9:006B`。正確 executable offset 是 `0x11AB`（早先手算成 `0x116B`
   是勘誤），反查 overlay-21 entry 15；`012Eh..0180h` 清掉五種角色貨幣並把
   餘額寫回目前角色。
4. 只有角色金額不足才走 `0196h..01D1h`：把 `ds:6752h` 傳給
   `00D9:0075`（file offset `0x11B5`、overlay-21 entry 17）計算 pooled money；
   足夠時扣完整費用，再呼叫 `00D9:0070`（file offset `0x11B0`、entry 16）
   清除並重寫 `6752h` 起的 pooled money。
5. 兩邊都不足才顯示 `Not enough money.`。控制流沒有「角色出一部分、pool 補差額」；
   因此 remake 也不可混合付款。

remake 目前以 `Gold`／`pooled_gold` 保存已換算 Gold 等值，不假稱已完整保存原版五幣別。
這足以重現本付款順序與價格；Pool／Share／Take 的五幣別 UI 仍是後續獨立切片。

## schema 2 與遷移

本規格實作當時的正式 schema 為 `pool-remake-state/2`；現行格式已由 Spec 035／037
依序升為 schema 3／4，但下列 HP 欄位與遷移契約仍有效。每名角色保存
`max_hp`、`current_hp`、`status`，
隊伍狀態另存非負的 `pooled_gold`；
合法條件為 `max_hp >= 1`、`0 <= current_hp <= max_hp`。新建角色三者初始化為
`rolled HP`、`rolled HP`、`0`。

讀取 `pool-remake-state/1` 時，舊 `hp` 確定性遷移為：

```text
max_hp = hp
current_hp = hp
status = 0
```

遷移只發生在記憶體；當時下一次原子寫檔輸出 schema 2，現行則輸出 schema 4。
未知 schema、未知 JSON 欄位、
不合法 HP 仍失敗即關閉。CoAB 已有 `HitPoints`、`MaxHitPoints` 與健康狀態，不套用這份
Pool schema；它只需升級到本輪共用 engine commit 並重編譯回歸。

## 實作閘門與驗收

1. schema 2 round-trip 保留最大／目前 HP 與 raw status。
2. schema 1 fixture 遷移後三欄精確符合上式，之後可寫成 schema 2。
3. 建角資料頁顯示 `HP current/max`，新角色兩值相同。
4. 三種傷勢治療使用可注入亂數、只增加目前 HP 並封頂；先嘗試由目前角色支付完整
   價格，個人不足才嘗試由 `pooled_gold` 支付完整價格，兩者不得合併。
5. Pool 全測試通過；CoAB 鎖定 engine `0819c64` 後重編譯並跑現行回歸，確認既有存檔、
   HP 與健康狀態沒有格式變更。

## 實作與驗證收據

- `internal/temple.CureWounds` 保存 exact 三組價格／骰式，付款只採「目前角色完整支付」
  或「個人不足時 pooled money 完整支付」，並同步 party／character library；不足不變更。
- `cmd/pool-game` 保留目前角色概念，主神殿／Heal 選單可按 1–6 切換；Heal 顯示原版十項
  順序，只有三種 Wounds 開放。付款確認、`Not enough money.` 與 `<name> is cured.` 均走
  玩家 UI；存檔失敗會把 HP 與兩種金錢來源整體回滾。
- 真實 `ECL3/block0` 的 Xvfb 按鍵測試已從 Rolf 導覽走到 Sune：YES → Heal →
  Cure Light Wounds → 100 GP 確認 → 個人扣款 → 1d8 治療 → Heal Exit → Temple Exit，
  並驗證同 VM 後續 `6DE1h=FFh`。Docker 內 `go test ./...` 全數通過。
