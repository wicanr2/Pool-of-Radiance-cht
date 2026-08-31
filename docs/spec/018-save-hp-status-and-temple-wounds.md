# Spec 018：存檔生命值／狀態與神殿傷勢治療

狀態：READY（schema 2、schema 1 遷移、三種傷勢治療）；DRAFT（其他狀態治療、pooled money 精確來源）
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

## schema 2 與遷移

正式 schema 為 `pool-remake-state/2`。每名角色保存 `max_hp`、`current_hp`、`status`；
合法條件為 `max_hp >= 1`、`0 <= current_hp <= max_hp`。新建角色三者初始化為
`rolled HP`、`rolled HP`、`0`。

讀取 `pool-remake-state/1` 時，舊 `hp` 確定性遷移為：

```text
max_hp = hp
current_hp = hp
status = 0
```

遷移只發生在記憶體；下一次原子寫檔輸出 schema 2。未知 schema、未知 JSON 欄位、
不合法 HP 仍失敗即關閉。CoAB 已有 `HitPoints`、`MaxHitPoints` 與健康狀態，不套用這份
Pool schema；它只需升級到本輪共用 engine commit 並重編譯回歸。

## 實作閘門與驗收

1. schema 2 round-trip 保留最大／目前 HP 與 raw status。
2. schema 1 fixture 遷移後三欄精確符合上式，之後可寫成 schema 2。
3. 建角資料頁顯示 `HP current/max`，新角色兩值相同。
4. 三種傷勢治療使用可注入亂數、只增加目前 HP 並封頂；金錢來源未閉合前，不得假稱
   pooled money exact。若先接玩家 UI，只能由角色自己的 `Gold` 支付並明標為暫定，
   否則維持 fail-closed。
5. Pool 全測試通過；CoAB 鎖定 engine `0819c64` 後重編譯並跑現行回歸，確認既有存檔、
   HP 與健康狀態沒有格式變更。
