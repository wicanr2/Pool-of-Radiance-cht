package main

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 探索中的物品頁（overlay-19 entry 6／7／8、overlay-24 entry 1 的穿戴效果，spec 149，
// issue #91）。全部從 Update() 送鍵：先在冒險畫面按 I 開頁，再按選項鍵。

// premadeFile 從原版 ZIP 讀一個預設人物檔（`chrdatd1.itm` 之類）。
func premadeFile(t *testing.T, name string) []byte {
	t.Helper()
	archive, err := zip.OpenReader(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if !strings.EqualFold(filepath.Base(file.Name), name) {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		raw, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	t.Fatalf("the ZIP has no %s", name)
	return nil
}

// premadeItems 把一個 `.itm` 拆成物品（63 bytes 一件，名稱是 Pascal 字串）。
func premadeItems(t *testing.T, name string) []poolsave.Item {
	t.Helper()
	raw := premadeFile(t, name)
	var items []poolsave.Item
	for offset := 0; offset+63 <= len(raw); offset += 63 {
		record := append([]byte(nil), raw[offset:offset+63]...)
		items = append(items, poolsave.Item{
			Name: strings.TrimSpace(string(record[1 : 1+int(record[0])])), Raw: record})
	}
	return items
}

// premadeEffects 是 `.spc` 的節點（存檔那一側的型別）。
func premadeEffects(t *testing.T, name string) []poolsave.EffectNode {
	t.Helper()
	nodes, err := gamepack.ParseEffectList(premadeFile(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return storedEffects(nodes)
}

func findItem(t *testing.T, items []poolsave.Item, name string) int {
	t.Helper()
	for index, item := range items {
		if strings.Contains(item.Name, name) {
			return index
		}
	}
	t.Fatalf("no %q among %d items", name, len(items))
	return -1
}

// newItemPageApp 是冒險畫面上的一個人，按 I 就開物品頁。
func newItemPageApp(t *testing.T, member poolsave.Character) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	types, err := gamepack.ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	parameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	caster, err := gamepack.ReadDOSSpellCaster(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	application := &app{mode: modeAdventure, introDone: true, language: languageEnglish,
		roller: fixedRoller{7}, itemTypes: types, spellParameters: parameters, spellCaster: caster}
	application.state.Party = []poolsave.Character{member}
	pressAll(t, application, ebiten.KeyI)
	if !application.equipmentOpen {
		t.Fatal("I did not open the item page")
	}
	return application
}

// cursorTo 從 Update() 按 ↓／↑ 把游標移到 index。
func cursorTo(t *testing.T, application *app, index int) {
	t.Helper()
	for guard := 0; guard < 64 && application.equipment.item != index; guard++ {
		key := ebiten.KeyDown
		if application.equipment.item > index {
			key = ebiten.KeyUp
		}
		pressAll(t, application, key)
	}
	if application.equipment.item != index {
		t.Fatalf("cursor stuck at %d, want %d", application.equipment.item, index)
	}
}

func effectCodes(member poolsave.Character) []uint8 {
	var codes []uint8
	for _, node := range member.Effects {
		codes = append(codes, node.Code)
	}
	return codes
}

// 選項列照 `0F79h..1148h` 組：冒險中有 Use，隊伍選單（`DS:4954h` 為 0）開的沒有；
// Halve 要身上不到 10h 件。
func TestItemPageFooterFollowsTheMenuContext(t *testing.T) {
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassID: "fighter",
		Abilities: [6]int{12, 12, 12, 12, 12, 12},
		Inventory: []poolsave.Item{itemOf("SWORD", testTypeSword, false, 0)}})
	if footer := application.itemPageFooter(); footer != "READY USE DROP HALVE JOIN EXIT" {
		t.Fatalf("adventure footer %q", footer)
	}
	application.equipment.page.creationMenu = true
	if footer := application.itemPageFooter(); footer != "READY DROP HALVE JOIN EXIT" {
		t.Fatalf("party-menu footer %q", footer)
	}
	pressAll(t, application, ebiten.KeyR, ebiten.KeyU)
	if application.equipment.message != "" || application.equipment.page.stage != itemPagePicking {
		t.Fatalf("U without Use on the menu did something: %q", application.equipment.message)
	}
}

// Ready 那一格有東西就擋下（overlay-19 entry 7 `159Ch`），不換手；先卸下再裝才換。
func TestItemPageReadyRefusesAnOccupiedSlot(t *testing.T) {
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassID: "fighter",
		Abilities: [6]int{12, 12, 12, 12, 12, 12},
		Inventory: []poolsave.Item{itemOf("SWORD", testTypeSword, true, 0),
			itemOf("TWO-HANDED SWORD", testTypeTwoHander, false, 0)}})
	cursorTo(t, application, 1)
	pressAll(t, application, ebiten.KeyR)
	inventory := application.state.Party[0].Inventory
	if !strings.Contains(application.equipment.message, "ALREADY USING SWORD") ||
		inventory[0].Raw[gamepack.ItemReadiedOffset] != 1 || inventory[1].Raw[gamepack.ItemReadiedOffset] != 0 {
		t.Fatalf("second weapon: %q, flags %d %d", application.equipment.message,
			inventory[0].Raw[gamepack.ItemReadiedOffset], inventory[1].Raw[gamepack.ItemReadiedOffset])
	}
	// ENTER 是 R 的別名（remake 的舊習慣，按鍵層只多一個入口）。
	cursorTo(t, application, 0)
	pressAll(t, application, ebiten.KeyEnter)
	cursorTo(t, application, 1)
	pressAll(t, application, ebiten.KeyR)
	if inventory[0].Raw[gamepack.ItemReadiedOffset] != 0 || inventory[1].Raw[gamepack.ItemReadiedOffset] != 1 {
		t.Fatalf("after unreadying first: flags %d %d", inventory[0].Raw[gamepack.ItemReadiedOffset],
			inventory[1].Raw[gamepack.ItemReadiedOffset])
	}
}

// 食人魔之力手套（83h）：ALFRED 原版存檔戴著它，力量 18/00、`.spc` 有一個 26h
// （等級欄 01、要收尾）。卸下（overlay-12 entry 122 模式 1）回到快照裡的 18/0，
// 節點摘掉；再戴上（模式 0，entry 18）印 "is stronger"、掛回一個與原版存檔逐位元組
// 相同的節點。
func TestGauntletsOfOgrePowerWearEffect(t *testing.T) {
	levels := premadeFile(t, "chrdatd3.sav")[0x96:0x9e]
	member := poolsave.Character{Name: "ALFRED", ClassLevels: append([]uint8(nil), levels...),
		Abilities: [6]int{18, 12, 12, 12, 12, 12}, ExceptionalStrength: 100,
		Inventory: premadeItems(t, "chrdatd3.itm"), Effects: premadeEffects(t, "chrdatd3.spc")}
	original := append([]poolsave.EffectNode(nil), member.Effects...)
	application := newItemPageApp(t, member)
	gauntlets := findItem(t, application.state.Party[0].Inventory, "Gauntlets of Ogre Power")
	cursorTo(t, application, gauntlets)

	pressAll(t, application, ebiten.KeyR)
	alfred := application.state.Party[0]
	if alfred.Inventory[gauntlets].Raw[gamepack.ItemReadiedOffset] != 0 {
		t.Fatal("R did not take the gauntlets off")
	}
	if alfred.Abilities[gamepack.AbilityStrength] != 18 || alfred.ExceptionalStrength != 0 {
		t.Fatalf("strength after taking them off %d/%d, want 18/0",
			alfred.Abilities[gamepack.AbilityStrength], alfred.ExceptionalStrength)
	}
	if codes := effectCodes(alfred); len(codes) != 1 || codes[0] != 0x3d {
		t.Fatalf("effects after taking them off %x, want only the ring's 3Dh", codes)
	}

	pressAll(t, application, ebiten.KeyR)
	alfred = application.state.Party[0]
	if alfred.Abilities[gamepack.AbilityStrength] != 18 || alfred.ExceptionalStrength != 100 {
		t.Fatalf("strength after putting them on %d/%d, want 18/100",
			alfred.Abilities[gamepack.AbilityStrength], alfred.ExceptionalStrength)
	}
	if !strings.Contains(application.equipment.message, "ALFRED IS STRONGER") {
		t.Fatalf("putting them on printed %q", application.equipment.message)
	}
	if len(alfred.Effects) != len(original) {
		t.Fatalf("effects %x, want %x", effectCodes(alfred), original)
	}
	for index := range original {
		if alfred.Effects[index] != original[index] {
			t.Fatalf("node %d is %+v, the original save has %+v", index, alfred.Effects[index], original[index])
		}
	}
}

// 火焰抗性戒指（81h）與位移斗篷（85h）走 overlay-12 entry 121：戴上掛物品 `+3Dh`
// 那個碼（持續 0、等級 0Ch、不收尾），拿下摘掉最早的那一個。戒指的節點與 HAPLO
// 原版 `.spc` 逐位元組相同。
func TestRingAndCloakWearEffects(t *testing.T) {
	haploEffects := premadeEffects(t, "chrdatd1.spc")
	ring := premadeItems(t, "chrdatd1.itm")[findItem(t, premadeItems(t, "chrdatd1.itm"), "Ring of Fire Resistance")]
	cloak := premadeItems(t, "chrdatd6.itm")[findItem(t, premadeItems(t, "chrdatd6.itm"), "Cloak of Displacement")]
	ring.Raw[gamepack.ItemReadiedOffset], cloak.Raw[gamepack.ItemReadiedOffset] = 0, 0
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassID: "fighter",
		Abilities: [6]int{12, 12, 12, 12, 12, 12}, Inventory: []poolsave.Item{ring, cloak}})

	pressAll(t, application, ebiten.KeyR)
	member := application.state.Party[0]
	if len(member.Effects) != 1 || member.Effects[0] != haploEffects[0] {
		t.Fatalf("ring on: effects %+v, HAPLO's save has %+v (%q)", member.Effects, haploEffects,
			application.equipment.message)
	}
	cursorTo(t, application, 1)
	pressAll(t, application, ebiten.KeyR)
	member = application.state.Party[0]
	want := gamepack.NewEffectNode(0x59, 0, 0x0c, false)
	if len(member.Effects) != 2 || member.Effects[1].Code != want.Code || member.Effects[1].Payload != want.Payload {
		t.Fatalf("cloak on: effects %+v (%q)", member.Effects, application.equipment.message)
	}
	cursorTo(t, application, 0)
	pressAll(t, application, ebiten.KeyR)
	if codes := effectCodes(application.state.Party[0]); len(codes) != 1 || codes[0] != 0x59 {
		t.Fatalf("ring off: effects %x", codes)
	}
	cursorTo(t, application, 1)
	pressAll(t, application, ebiten.KeyR)
	if codes := effectCodes(application.state.Party[0]); len(codes) != 0 {
		t.Fatalf("cloak off: effects %x", codes)
	}
}

// 墓園的雙手劍 +1 +3 對不死生物（88h，`+3Dh` 是 03h）同樣經 entry 121；戰鬥中的 R
// 掛在這一格的戰鬥串列上，存檔那一份跟著寫。
func TestCombatReadyAttachesTheWearEffect(t *testing.T) {
	sword := graveyardSword(t)
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5, sword)
	pressAll(t, application, ebiten.KeyU, ebiten.KeyR)
	if !state.hasEffect(1, 0x03) || len(application.state.Party[0].Effects) != 1 {
		t.Fatalf("readying the sword in combat: board %v, member %+v", state.Effects[1],
			application.state.Party[0].Effects)
	}
	pressAll(t, application, ebiten.KeyR)
	if state.hasEffect(1, 0x03) || len(application.state.Party[0].Effects) != 0 {
		t.Fatalf("unreadying the sword in combat: board %v", state.Effects[1])
	}
}

// 87h（`3141h`）：力量不到 19 的戴不上，常式自己把 `+34h` 寫回 0。
func TestGiantStrengthItemRefusesAWeakWearer(t *testing.T) {
	item := itemOf("GIRDLE", 0x3f, false, 0)
	item.Raw[gamepack.ItemEffectOffset] = 0x87
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassID: "fighter",
		Abilities: [6]int{18, 12, 12, 12, 12, 12}, Inventory: []poolsave.Item{item}})
	pressAll(t, application, ebiten.KeyR)
	if application.state.Party[0].Inventory[0].Raw[gamepack.ItemReadiedOffset] != 0 ||
		!strings.Contains(application.equipment.message, "MUST HAVE GIANT STRENGTH") {
		t.Fatalf("a strength-18 wearer: %q", application.equipment.message)
	}
}

// Use：沒穿戴的印 "Must be Readied"；治療藥水穿上之後 U → 挑對象 → Enter 補血、
// 用掉（充能 1 → 拿掉）；ESC 放棄不記帳。
func TestItemPageUsesAHealingPotion(t *testing.T) {
	items := premadeItems(t, "chrdatd1.itm")
	potion := items[findItem(t, items, "Potion of Healing")]
	potion.Raw[gamepack.ItemCountOffset] = 0
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassID: "fighter",
		Abilities: [6]int{12, 12, 12, 12, 12, 12}, CurrentHP: 1, MaxHP: 30,
		Inventory: []poolsave.Item{potion}})
	pressAll(t, application, ebiten.KeyU)
	if !strings.Contains(application.equipment.message, "MUST BE READIED") {
		t.Fatalf("an unreadied potion: %q", application.equipment.message)
	}
	pressAll(t, application, ebiten.KeyR, ebiten.KeyU)
	if application.equipment.page.stage != itemPageTarget {
		t.Fatalf("U on a readied potion: stage %d, %q", application.equipment.page.stage,
			application.equipment.message)
	}
	pressAll(t, application, ebiten.KeyEscape)
	if len(application.state.Party[0].Inventory) != 1 || application.equipment.page.stage != itemPagePicking {
		t.Fatal("ESC while choosing a target spent the potion")
	}
	pressAll(t, application, ebiten.KeyU, ebiten.KeyEnter)
	member := application.state.Party[0]
	if member.CurrentHP <= 1 || len(member.Inventory) != 0 {
		t.Fatalf("after drinking: hp %d, items %d (%q)", member.CurrentHP, len(member.Inventory),
			application.equipment.message)
	}
	if !application.equipmentOpen {
		t.Fatal("using an item outside combat closed the page (`130Ah` keeps the menu)")
	}
}

// 參數表 `+07h` 為 0 的物品法術在戰鬥外問 "Use it?"：N 什麼都不做，Y 照樣記帳但不放。
func TestItemPageCombatOnlyWandAsks(t *testing.T) {
	items := premadeItems(t, "chrdatd4.itm")
	wand := items[findItem(t, items, "Wand of Magic Missiles")]
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassID: "magic-user",
		Abilities: [6]int{12, 12, 12, 12, 12, 12}, Inventory: []poolsave.Item{wand}})
	spell, _ := gamepack.AIItemSpell(wand.Raw, false)
	if gamepack.ItemUsableOutsideCombat(application.spellParameters[spell]) {
		t.Fatalf("spell %d is usable outside combat; pick another wand for this test", spell)
	}
	charges := wand.Raw[gamepack.AIItemChargesOffset]
	pressAll(t, application, ebiten.KeyR, ebiten.KeyU)
	if !strings.Contains(application.equipment.message, "COMBAT-ONLY") {
		t.Fatalf("U on the wand: %q", application.equipment.message)
	}
	pressAll(t, application, ebiten.KeyN)
	if application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset] != charges {
		t.Fatal("N spent a charge")
	}
	pressAll(t, application, ebiten.KeyU, ebiten.KeyY)
	if got := application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset]; got != charges-1 {
		t.Fatalf("Y left %d charges, want %d", got, charges-1)
	}
}

// 牧師卷軸：牧師讀得出來（overlay-22 `047Ch`），挑一行放，放完抹掉那一行。
func TestItemPageReadsAClericScroll(t *testing.T) {
	items := premadeItems(t, "chrdatb3.itm")
	scroll := items[findItem(t, items, "Clerical Scroll With 2 Spells")]
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotCleric] = 3
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassLevels: levels,
		Abilities: [6]int{12, 12, 12, 12, 12, 12}, CurrentHP: 1, MaxHP: 30,
		Inventory: []poolsave.Item{scroll}})
	words := scroll.Raw[gamepack.ItemNameWordOffset+1]
	pressAll(t, application, ebiten.KeyR, ebiten.KeyU)
	page := application.equipment.page
	if page.stage != itemPageScroll || len(page.scroll) == 0 {
		t.Fatalf("U on a cleric scroll: stage %d rows %d (%q)", page.stage, len(page.scroll),
			application.equipment.message)
	}
	pressAll(t, application, ebiten.KeyEnter)
	switch application.equipment.page.stage {
	case itemPageTarget:
		pressAll(t, application, ebiten.KeyEnter)
	case itemPageCombatOnly:
		pressAll(t, application, ebiten.KeyY)
	default:
		t.Fatalf("after picking a line: stage %d (%q)", application.equipment.page.stage,
			application.equipment.message)
	}
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 1 || inventory[0].Raw[gamepack.ItemNameWordOffset+1] != words-1 {
		t.Fatalf("the line was not erased: %d items, name word %#x (was %#x)", len(inventory),
			inventory[0].Raw[gamepack.ItemNameWordOffset+1], words)
	}
}

// Drop、Halve、Join 與戰鬥中同一套規則（spec 144），只是在物品頁上。
func TestItemPageDropHalveJoin(t *testing.T) {
	application := newItemPageApp(t, poolsave.Character{Name: "A", ClassID: "fighter",
		Abilities: [6]int{12, 12, 12, 12, 12, 12},
		Inventory: []poolsave.Item{itemOf("ARROWS", gamepack.ItemTypeArrow, true, 11),
			itemOf("SWORD", testTypeSword, false, 0)}})
	pressAll(t, application, ebiten.KeyD)
	if !strings.Contains(application.equipment.message, "MUST BE UNREADIED") {
		t.Fatalf("dropping readied arrows: %q", application.equipment.message)
	}
	pressAll(t, application, ebiten.KeyH)
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 3 || inventory[0].Raw[gamepack.ItemCountOffset] != 6 ||
		inventory[1].Raw[gamepack.ItemCountOffset] != 5 || inventory[1].Raw[gamepack.ItemReadiedOffset] != 0 {
		t.Fatalf("after H: %d items", len(inventory))
	}
	pressAll(t, application, ebiten.KeyJ)
	inventory = application.state.Party[0].Inventory
	if len(inventory) != 2 || inventory[0].Raw[gamepack.ItemCountOffset] != 11 {
		t.Fatalf("after J: %d items, count %d", len(inventory), inventory[0].Raw[gamepack.ItemCountOffset])
	}
	cursorTo(t, application, 1)
	pressAll(t, application, ebiten.KeyD)
	if !strings.Contains(application.equipment.message, "WILL BE GONE FOREVER") {
		t.Fatalf("drop prompt %q", application.equipment.message)
	}
	pressAll(t, application, ebiten.KeyN)
	if len(application.state.Party[0].Inventory) != 2 {
		t.Fatal("N dropped the sword")
	}
	pressAll(t, application, ebiten.KeyD, ebiten.KeyY)
	inventory = application.state.Party[0].Inventory
	if len(inventory) != 1 || inventory[0].Name != "ARROWS" {
		t.Fatalf("after Y: %+v", inventory)
	}
}
