package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 拿真檔的墓園寶物當素材：那把 Two-Handed Sword +1 是遊戲裡玩家第一件能撿到
// 的武器，用合成資料測不到「型別索引查得到表」這件事。
func graveyardSword(t *testing.T) poolsave.Item {
	t.Helper()
	records, err := gamepack.ReadDOSTreasureItemBlock(dosZIPForTests, 3, 0x33)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, record := range records {
		if record.Name == "Two-Handed Sword +1 +3 vs. Undead" {
			raw := append([]byte(nil), record.Raw[:]...)
			return poolsave.Item{Name: record.Name, Raw: raw}
		}
	}
	t.Fatal("the graveyard treasure has no two-handed sword")
	return poolsave.Item{}
}

const dosZIPForTests = "../../Pool of Radiance (1988).zip"

func newEquipmentApp(t *testing.T) *app {
	t.Helper()
	table, err := gamepack.ReadDOSItemTypeTable(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	member := poolsave.Character{
		Name: "HERO", ClassID: "fighter",
		Abilities:           [6]int{18, 10, 10, 12, 10, 10},
		ExceptionalStrength: 100,
		Inventory:           []poolsave.Item{graveyardSword(t)},
	}
	return &app{
		itemTypes: table,
		equipment: &equipmentState{},
		state:     poolsave.State{Party: []poolsave.Character{member}},
	}
}

// 撿到的物品預設沒有裝備上：原版記錄的 +34h 是 0。
func TestTreasureItemsArriveUnreadied(t *testing.T) {
	a := newEquipmentApp(t)
	if _, ok := readiedWeapon(a.state.Party[0]); ok {
		t.Fatal("a freshly taken item was already readied")
	}
}

// ENTER 裝備、再按一次卸下。
func TestEnterTogglesReady(t *testing.T) {
	a := newEquipmentApp(t)
	a.equipmentOpen = true
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.equipmentInput()
	weapon, ok := readiedWeapon(a.state.Party[0])
	if !ok {
		t.Fatal("ENTER did not ready the sword")
	}
	if weapon.Name != "Two-Handed Sword +1 +3 vs. Undead" {
		t.Fatalf("readied %q", weapon.Name)
	}
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.equipmentInput()
	if _, ok := readiedWeapon(a.state.Party[0]); ok {
		t.Fatal("a second ENTER did not unready the sword")
	}
}

// 一次只能裝備一件：原版的角色記錄只有一個武器槽，兩件同時掛著會讓
// readiedWeapon 依順序挑，畫面上看起來像隨機換武器。
func TestReadyingOneItemUnreadiesTheOther(t *testing.T) {
	a := newEquipmentApp(t)
	second := graveyardSword(t)
	second.Name = "SECOND SWORD"
	a.state.Party[0].Inventory = append(a.state.Party[0].Inventory, second)
	a.equipmentOpen = true
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.equipmentInput()
	a.equipment.item = 1
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.equipmentInput()

	readied := 0
	for _, item := range a.state.Party[0].Inventory {
		if item.Raw[itemReadyOffset] != 0 {
			readied++
		}
	}
	if readied != 1 {
		t.Fatalf("%d items are readied at once", readied)
	}
	weapon, _ := readiedWeapon(a.state.Party[0])
	if weapon.Name != "SECOND SWORD" {
		t.Fatalf("the readied weapon is %q", weapon.Name)
	}
}

// 裝備上的武器要真的改變戰鬥數值，而不是只在裝備頁顯示。
// 一級戰士基礎 THAC0 internal 是 28h（typed 20）；力量 18/00 加 3、武器 +1，
// 傷害是雙手劍的 1d10 加 6（力量）加 1（武器）。
func TestReadiedWeaponDrivesTheCombatStats(t *testing.T) {
	a := newEquipmentApp(t)
	a.equipmentOpen = true
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.equipmentInput()

	member := a.state.Party[0]
	base, _, _, err := partyCombatStats(member)
	if err != nil {
		t.Fatal(err)
	}
	if base != 0x28 {
		t.Fatalf("base internal THAC0 %#02x, want 0x28", base)
	}
	weapon, _ := readiedWeapon(member)
	stats, err := a.weaponCombatStats(weapon, member, base)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Thac0Internal != 0x2c {
		t.Fatalf("internal THAC0 %#02x, want 0x2c", stats.Thac0Internal)
	}
	if stats.DamageCount != 1 || stats.DamageSides != 10 || stats.DamageBonus != 7 {
		t.Fatalf("damage %dd%d%+d, want 1d10+7",
			stats.DamageCount, stats.DamageSides, stats.DamageBonus)
	}
}

// 隊伍是空的就不開這一頁，並說明原因——安靜地不動會讓玩家以為按鍵沒吃到。
func TestEquipmentStaysClosedWithAnEmptyParty(t *testing.T) {
	a := &app{}
	a.openEquipment()
	if a.equipmentOpen {
		t.Fatal("the equipment screen opened with an empty party")
	}
	if a.statusLine == "" {
		t.Fatal("nothing explained why it did not open")
	}
}
