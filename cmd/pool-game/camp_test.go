package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/temple"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

func campApp(t *testing.T) *app {
	t.Helper()
	radix, err := gamepack.ReadDOSTimeRadix(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application := &app{language: languageEnglish}
	application.restDuration = gamepack.NewRestDuration(radix)
	application.restField = gamepack.RestFieldMinutes
	application.state.Party = []poolsave.Character{
		{Name: "A", MaxHP: 20, CurrentHP: 5},
		{Name: "B", MaxHP: 12, CurrentHP: 12},
	}
	return application
}

// 休息一天：每人回一點生命力，滿血的不會超過上限。
// 說明書 p.29 寫的是「每休息二十四小時各隊員可恢復一點 HP」。
func TestRestingADayHealsOnePointEach(t *testing.T) {
	application := campApp(t)
	for step := 0; step < 24; step++ {
		application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)
	}
	application.restParty()
	if got := application.state.Party[0].CurrentHP; got != 6 {
		t.Errorf("休息一天之後第一個人是 %d 點，應該是 6", got)
	}
	if got := application.state.Party[1].CurrentHP; got != 12 {
		t.Errorf("滿血的人變成 %d 點，不該超過上限", got)
	}
	if !strings.Contains(application.statusLine, "+1") {
		t.Errorf("狀態列是 %q，應該說回了 1 點", application.statusLine)
	}
}

// 負對照：時間不夠就什麼都不會回。少了它，「休息一天回一點」證不了因果。
func TestRestingTooBrieflyHealsNothing(t *testing.T) {
	application := campApp(t)
	for step := 0; step < 12; step++ {
		application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)
	}
	application.restParty()
	if got := application.state.Party[0].CurrentHP; got != 5 {
		t.Errorf("休息半天之後是 %d 點，不該回", got)
	}
}

// 選欄與增減照原版：分鐘一次五分，Y／H／M 換欄。
func TestCampKeysAdjustTheRestTime(t *testing.T) {
	application := campApp(t)
	application.restField = gamepack.RestFieldMinutes
	application.restDuration = application.restDuration.Increase(application.restField)
	if got := application.restDuration.Minutes(); got != 5 {
		t.Errorf("分鐘加一次是 %d 分，原版一次五分", got)
	}
	application.restField = gamepack.RestFieldDays
	application.restDuration = application.restDuration.Increase(application.restField)
	if got := application.restDuration.Days(); got != 1 {
		t.Errorf("天數加一次是 %d 天", got)
	}
	line := application.campRestTimeLine()
	if !strings.Contains(line, "01") || !strings.Contains(line, "05") {
		t.Errorf("時間那一列是 %q，應該看得到 1 天與 5 分", line)
	}
}

// 神殿的九項都接上了：選單上的名稱與 temple.Services 一致，而且沒有一項
// 還停在「fail-closed」。
func TestTempleMenuCoversEveryService(t *testing.T) {
	if len(templeHealOptions) != len(templeHealServiceIDs)+1 {
		t.Fatalf("選單有 %d 項，服務有 %d 項加一個離開",
			len(templeHealOptions), len(templeHealServiceIDs))
	}
	for index, id := range templeHealServiceIDs {
		service, ok := temple.ServiceByID(id)
		if !ok {
			t.Fatalf("temple.Services 裡沒有 %q", id)
		}
		if templeHealOptions[index] != service.Name {
			t.Errorf("第 %d 項寫的是 %q，服務叫 %q", index, templeHealOptions[index], service.Name)
		}
	}
	if templeHealOptions[len(templeHealOptions)-1] != "Exit" {
		t.Errorf("最後一項是 %q", templeHealOptions[len(templeHealOptions)-1])
	}
}

// 走完一次石化解除：從主選單進 Heal、挑那一項、確認，錢與狀態都要動。
func TestTempleStoneToFleshRunsThroughTheMenu(t *testing.T) {
	character := poolsave.Character{
		Name: "HERO", MaxHP: 12, CurrentHP: 12, Status: temple.StatusStone,
		Money: [7]uint16{3: 3000},
	}
	application := &app{
		roller: fixedTempleRoller(1),
		state: poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: []poolsave.Character{character},
			Party:            []poolsave.Character{character}},
		templeActive: true,
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.enterTempleHeal()
	application.cellMenuCursor = len(templeHealServiceIDs) - 1 // Stone to Flesh
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("挑石化解除：%v", err)
	}
	if application.templeStage != templeConfirm ||
		!strings.Contains(application.eventText, "2000 gold pieces") {
		t.Fatalf("確認畫面 stage=%d text=%q", application.templeStage, application.eventText)
	}
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("確認：%v", err)
	}
	if got := application.state.Party[0].Status; got != temple.StatusNormal {
		t.Errorf("狀態還是 %d", got)
	}
	if got := application.state.Party[0].Money[3]; got != 1000 {
		t.Errorf("剩 %d 金幣，應該收 2000", got)
	}
}

// 負對照：沒有那個毛病時不收錢，也不會有人被治好。
func TestTempleRefusesWhenThereIsNothingToCure(t *testing.T) {
	character := poolsave.Character{Name: "HERO", MaxHP: 12, CurrentHP: 12,
		Money: [7]uint16{3: 3000}}
	application := &app{
		roller: fixedTempleRoller(1),
		state: poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: []poolsave.Character{character},
			Party:            []poolsave.Character{character}},
		templeActive: true,
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.enterTempleHeal()
	application.cellMenuCursor = len(templeHealServiceIDs) - 1 // Stone to Flesh
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("挑石化解除：%v", err)
	}
	if !strings.Contains(application.statusLine, "is not stoned") {
		t.Errorf("狀態列是 %q，應該說他沒有石化", application.statusLine)
	}
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("確認：%v", err)
	}
	if got := application.state.Party[0].Money[3]; got != 3000 {
		t.Errorf("沒毛病卻收了錢，剩 %d", got)
	}
}

// 估價走完一次：挑寶石、看到價、賣掉換成白金，計數減一。
func TestTempleAppraiseSellsAGemForPlatinum(t *testing.T) {
	character := poolsave.Character{Name: "HERO", MaxHP: 12, CurrentHP: 12}
	character.Money[pooltreasure.Gems] = 2
	application := &app{
		roller: fixedTempleRoller(100), // 1d100 擲 100 → 寶石值 5000
		state: poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: []poolsave.Character{character},
			Party:            []poolsave.Character{character}},
		templeActive: true,
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.enterTempleAppraise()
	if application.templeStage != templeAppraise {
		t.Fatalf("沒有進估價選單，stage=%d 文字 %q", application.templeStage, application.eventText)
	}
	application.cellMenuCursor = 0 // Gems
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("估寶石：%v", err)
	}
	if !strings.Contains(application.eventText, "5000 gp") {
		t.Fatalf("估價文字是 %q", application.eventText)
	}
	if got := application.state.Party[0].Money[pooltreasure.Gems]; got != 1 {
		t.Errorf("估完之後還有 %d 顆，原版是先減再擲", got)
	}
	application.cellMenuCursor = 0 // Sell
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("賣掉：%v", err)
	}
	if got := application.state.Party[0].Money[pooltreasure.Platinum]; got != 1000 {
		t.Errorf("賣得 %d 白金，五千金幣換算是 1000 白金", got)
	}
}

// 留著就變成一件物品，欄位照原版那 63 bytes。
func TestTempleAppraiseKeepsAJewelAsAnItem(t *testing.T) {
	character := poolsave.Character{Name: "HERO", MaxHP: 12, CurrentHP: 12}
	character.Money[pooltreasure.Jewelry] = 1
	application := &app{
		roller: fixedTempleRoller(1), // 1d100 擲 1 → 珠寶 100 ＋ Random(900) 的 0
		state: poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: []poolsave.Character{character},
			Party:            []poolsave.Character{character}},
		templeActive: true,
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.enterTempleAppraise()
	application.cellMenuCursor = 1 // Jewelry
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("估珠寶：%v", err)
	}
	if len(application.cellMenuOptions) != 2 {
		t.Fatalf("背包沒滿卻沒有 Keep：%v", application.cellMenuOptions)
	}
	application.cellMenuCursor = 1 // Keep
	if err := application.selectSuneTempleOption(); err != nil {
		t.Fatalf("留著：%v", err)
	}
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 1 || len(inventory[0].Raw) != 0x3F {
		t.Fatalf("背包是 %+v", inventory)
	}
	raw := inventory[0].Raw
	if raw[0x2E] != pooltreasure.KeptItemType || raw[0x31] != pooltreasure.KeptItemSubtype ||
		raw[0x37] != 1 {
		t.Errorf("物品欄位是 %02X／%02X／%02X", raw[0x2E], raw[0x31], raw[0x37])
	}
	if value := int(raw[0x3A]) | int(raw[0x3B])<<8; value != 100 {
		t.Errorf("價值寫成 %d，應該是 100", value)
	}
	if got := application.state.Party[0].Money[pooltreasure.Platinum]; got != 0 {
		t.Errorf("留著卻拿到 %d 白金", got)
	}
}

// 商店那一側的估價：規則同一份（spec 116），賣掉一樣換成白金。
func TestShopAppraiseSharesTheTempleRules(t *testing.T) {
	character := poolsave.Character{Name: "HERO", MaxHP: 12, CurrentHP: 12}
	character.Money[pooltreasure.Gems] = 1
	application := &app{
		roller: fixedTempleRoller(100), // 寶石值 5000
		state: poolsave.State{Schema: poolsave.Schema,
			CharacterLibrary: []poolsave.Character{character},
			Party:            []poolsave.Character{character}},
		shop: &shopState{},
	}
	if err := application.offerShopAppraise(appraiseGem); err != nil {
		t.Fatalf("估價：%v", err)
	}
	if !application.shop.appraising || application.shop.appraiseValue != 5000 {
		t.Fatalf("估價狀態 %+v", application.shop)
	}
	if got := application.state.Party[0].Money[pooltreasure.Gems]; got != 0 {
		t.Errorf("估完之後還有 %d 顆", got)
	}
	application.resolveShopAppraise(false)
	if got := application.state.Party[0].Money[pooltreasure.Platinum]; got != 1000 {
		t.Errorf("賣得 %d 白金，五千金幣換算是 1000 白金", got)
	}
	// 負對照：沒有存貨時不擲骰也不改任何東西。
	if err := application.offerShopAppraise(appraiseGem); err != nil {
		t.Fatalf("空手估價：%v", err)
	}
	if application.shop.appraising {
		t.Error("沒有寶石卻進了估價狀態")
	}
	if got := application.state.Party[0].Money[pooltreasure.Platinum]; got != 1000 {
		t.Errorf("空手估價之後白金變成 %d", got)
	}
}
