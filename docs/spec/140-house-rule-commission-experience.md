# Spec 140：自訂規則——委任獎賞折算經驗值（預設關）

狀態：READY（開關、發放點、換算、測試都在；這是 remake 自己的規則，不是原版行為，
所以沒有 CONFORMED 可言）。日期：2026-09-15。主台帳：GitHub issue #28。

## 一句話

原版的委任獎金在戰後結算折成經驗值、全隊分（spec 148：貧民窟 2700，珠寶一件算 2200）。
使用者要求「委任也能升級」，這一條在原版那一份之外另加，所以加一條**可切換、預設關、跟著存檔走**的自訂規則：
市政廳職員發的獎賞，每 1 金幣等值折 1 點經驗，**每一位隊員各得**。出處是 AD&D 一版
DMG 的「寶物 1 gp ＝ 1 XP」（SSI 沒實作）；「各得不除以人數」是 remake 的決定——
除的話一場委任只剩一百多點，升不了級，這條規則就沒有意義。

## 為什麼可以做

CLAUDE.md §3：原版忠實是驗收基準，改變原版規則狀態的東西必須是可切換的表現層。
這一條關著時，所有既有測試、對拍畫面與 #5 的主線收據一個都不動——隊伍選單照原版
截圖只列原版的項目，`H` 是不列出來的開關（README 寫了）；開著時選單底下與冒險畫面
右上角都標「自訂規則」。

## 規則

| 項 | 內容 |
|---|---|
| 開關 | `poolsave.State.HouseRules.CommissionExperience`，隊伍選單按 `H` 切換（不列在選單上），存檔帶著 |
| 發放點 | 市政廳職員（`ECL3/8`）的 `TREASURE`（`9F28h`，`CLEARMONSTERS → TREASURE → COMBAT` 那一場空戰鬥）進 remake 的寶物服務時 |
| 不算的 | 其他區塊的空戰鬥寶物（貧民窟的袋子、樓板下的箱子、寶藏室）——那是撿到的 |
| 換算 | 200 銅＝20 銀＝2 琥珀金＝1 金＝1/5 白金；寶石 10、珠寶 100（估價表最低一格，估價要擲骰，取基準值讓數字可重現） |
| 分配 | 每一位有職業碼的隊員各得換算值，再套 spec 097 的主屬性加成（`ExperienceShare`） |
| 提示 | 狀態列印「自訂規則：市政廳獎賞折算，每位隊員各得 N 點經驗」 |

依 spec 137 的獎賞表換算（未計主屬性加成）：貧民窟 600、索寇要塞 1350、
圖書館六本書合計 2500、Norris 1250、波多廣場 1250、斯托揚諾河 2750、結局 12000。
戰士二級 2001：貧民窟＋索寇之後就到。

## 實作與測試

`cmd/pool-game/main.go`：`commissionExperienceValue`／`awardCommissionExperience`（掛在
`enterTreasure`）、`houseRuleStateMessage`、冒險畫面的標記；`party_menu.go` 的 `H`。
測試 `cmd/pool-game/house_rule_test.go`：換算值、開／關各走一次真的職員交件、選單開關。

探針收據 `TestMainlineProbeHouseRuleCommissionExperience`（`mainline_probe_test.go`，
開關開著、seed 137）：貧民窟 20 場 → 古托井打諾里斯 → 市政廳交件每人 1250／1375 XP
→ 訓練所（`mainline_house_rule_test.go` 的 `enterTrainingHall`）牧師升二級 → 索寇要塞
(8,5) 全滅。數字與每一段的差異在 `docs/playtest/mainline-end-to-end.md` 補六。
規則本身照設計運作；一級隊伍過不了的仍是戰術（#22）。
