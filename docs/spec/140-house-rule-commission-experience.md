# Spec 140：自訂規則——委任經驗值加倍（預設關）

狀態：READY（開關、發放點、測試都在；這是 remake 自己的規則，不是原版行為，
所以沒有 CONFORMED 可言）。日期：2026-09-26。主台帳：GitHub issue #28。

## 一句話

原版的市政廳獎賞在戰後結算就折成經驗值、全隊分（spec 148：貧民窟公款 2700，珠寶一件算 2200）。
開著這一條時，**同一份再發一次**，所以交件拿到的經驗值是原版的兩倍。可切換、預設關、跟著存檔走。

## 為什麼可以做

CLAUDE.md §3：原版忠實是驗收基準，改變原版規則狀態的東西必須是可切換的表現層。
這一條關著時，所有既有測試、對拍畫面與 #5 的主線收據一個都不動——隊伍選單照原版
截圖只列原版的項目，`H` 是不列出來的開關（README 寫了）；開著時選單底下與冒險畫面
右上角都標「自訂規則」。

## 規則

| 項 | 內容 |
|---|---|
| 開關 | `poolsave.State.HouseRules.CommissionExperience`，隊伍選單按 `H` 切換（不列在選單上），存檔帶著 |
| 發放點 | 市政廳職員（`ECL3/8`）的 `TREASURE`（`9F28h`，`CLEARMONSTERS → TREASURE → COMBAT` 那一場空戰鬥）進 remake 的寶物服務時，接在原版那一份（`awardTreasureExperience`）之後 |
| 不算的 | 其他區塊的空戰鬥寶物（貧民窟的袋子、樓板下的箱子、寶藏室）——那是撿到的 |
| 額外那一份 | 與原版同一套：公款七欄與物品依 spec 148 折算，除人數，再套 spec 097 的主屬性加成（`shareExperience`） |
| 提示 | 狀態列印「自訂規則：市政廳獎賞經驗值加倍，每位隊員另得 N 點經驗」，N 是除人數後、主屬性加成前的份額 |

貧民窟那一件：公款 250 金＋50 白金＋1 珠寶折成 2700，關著時一個人的隊伍得 2700，開著得 5400。

## 規則的前身

2026-09-15 的版本以「原版委任只給錢」為前提，另用 AD&D 一版 DMG 的 1 gp ＝ 1 XP 換算、
每位隊員各得。#94 讀出原版的空戰鬥同樣走 overlay-05 entry 2，前提不成立；使用者於
2026-09-26 決定保留開關、語意改成加倍。

## 實作與測試

`cmd/pool-game/main.go`：`awardCommissionExperience`（掛在 `enterTreasure`）、
`houseRuleStateMessage`、冒險畫面的標記；`party_menu.go` 的 `H`。
測試 `cmd/pool-game/house_rule_test.go`：開／關各走一次真的職員交件（2700／5400）、選單開關。

探針收據 `TestMainlineProbeHouseRuleCommissionExperience`（`mainline_probe_test.go`，
開關開著）走貧民窟 → 古托井打諾里斯 → 市政廳交件 → 訓練所升級 → 索寇要塞；
各段數字在 `docs/playtest/mainline-end-to-end.md`。一級隊伍過不了的仍是戰術（#22）。
