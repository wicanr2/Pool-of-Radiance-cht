package main

import (
	"os"
	"path/filepath"
	"strings"
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
	if _, ok := a.readiedWeapon(a.state.Party[0]); ok {
		t.Fatal("a freshly taken item was already readied")
	}
}

// ENTER 裝備、再按一次卸下。
func TestEnterTogglesReady(t *testing.T) {
	a := newEquipmentApp(t)
	a.equipmentOpen = true
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.equipmentInput()
	weapon, ok := a.readiedWeapon(a.state.Party[0])
	if !ok {
		t.Fatal("ENTER did not ready the sword")
	}
	if weapon.Name != "Two-Handed Sword +1 +3 vs. Undead" {
		t.Fatalf("readied %q", weapon.Name)
	}
	a.keys = scriptedKeys{ebiten.KeyEnter: true}
	a.equipmentInput()
	if _, ok := a.readiedWeapon(a.state.Party[0]); ok {
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
	weapon, _ := a.readiedWeapon(a.state.Party[0])
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
	weapon, _ := a.readiedWeapon(member)
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

// 空手的角色也要吃到敏捷的 AC 調整。原版的重算不看物品鏈就先把敏捷加進去，
// 提早返回會讓「還沒買裝備」的隊伍平白差 4 點 AC。
func TestDexterityReachesTheArmourClassWithoutItems(t *testing.T) {
	a := newEquipmentApp(t)
	member := poolsave.Character{Name: "HERO", ClassID: "fighter",
		Abilities: [6]int{18, 10, 10, 18, 10, 10}, ExceptionalStrength: 100}
	armor, movement, err := a.memberDefenceStats(member, creationArmorClassInternal, creationBaseMovement)
	if err != nil {
		t.Fatal(err)
	}
	if armor != 54 {
		t.Fatalf("internal AC %d, want 54（敏捷 18 的 +4）", armor)
	}
	if movement != creationBaseMovement {
		t.Fatalf("movement %d, want %d", movement, creationBaseMovement)
	}
}

// 穿上原版的板甲之後 AC 與腳程都要動。兩件事共用同一條物品鏈，只接一半
// 會出現「AC 算了裝備、腳程沒算」這種只在特定隊伍才看得出來的偏差。
func TestReadiedArmourDrivesTheDefenceStats(t *testing.T) {
	a := newEquipmentApp(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "workplace", "oracle", "dos", "chrdatd2.itm"))
	if err != nil {
		t.Skipf("original item records unavailable: %v", err)
	}
	var plate []byte
	for offset := 0; offset+63 <= len(raw); offset += 63 {
		item := raw[offset : offset+63]
		entry, err := a.itemTypes.Entry(item[gamepack.ItemTypeOffset])
		if err != nil || entry.Category() != gamepack.ItemCategoryArmour {
			continue
		}
		if entry.Raw[gamepack.ItemTypeArmourClassOffset]&0x80 != 0 {
			plate = append([]byte(nil), item...)
			break
		}
	}
	if plate == nil {
		t.Fatal("chrdatd2 has no armour item")
	}
	plate[gamepack.ItemReadiedOffset] = 1
	member := poolsave.Character{Name: "HERO", ClassID: "fighter",
		Abilities: [6]int{18, 10, 10, 18, 10, 10}, ExceptionalStrength: 100,
		Inventory: []poolsave.Item{{Name: "PLATE MAIL +2", Raw: plate}}}

	armor, movement, err := a.memberDefenceStats(member, creationArmorClassInternal, creationBaseMovement)
	if err != nil {
		t.Fatal(err)
	}
	// 板甲 +2 是 39h＋2 ＝ 59，加敏捷 4 ＝ 63，檯面上是 AC -3。
	if armor != 63 {
		t.Fatalf("internal AC %d, want 63", armor)
	}
	// 重 450 又有加值：spec 079 的 6 加 3。
	if movement != 9 {
		t.Fatalf("movement %d, want 9", movement)
	}
}

// 武器是類別 0 那一件，不是物品鏈上第一件裝備。`chrdatd2` 身上第一件裝備
// 是火焰抗性戒指（0d0），最後一件才是長劍 +4——挑錯的話整隊打不出傷害，
// 而戰鬥還是會照跑，報表上只看得到「打不贏」。
func TestReadiedWeaponSkipsTheRingsAndArmour(t *testing.T) {
	a := newEquipmentApp(t)
	raw, err := os.ReadFile(filepath.Join("..", "..", "workplace", "oracle", "dos", "chrdatd2.itm"))
	if err != nil {
		t.Skipf("original item records unavailable: %v", err)
	}
	var inventory []poolsave.Item
	for offset := 0; offset+63 <= len(raw); offset += 63 {
		record := append([]byte(nil), raw[offset:offset+63]...)
		length := int(record[0])
		inventory = append(inventory, poolsave.Item{
			Name: string(record[1 : 1+length]), Raw: record})
	}
	member := poolsave.Character{Name: "HERO", ClassID: "fighter",
		Abilities: [6]int{18, 10, 10, 18, 10, 10}, ExceptionalStrength: 100,
		Inventory: inventory}

	weapon, ok := a.readiedWeapon(member)
	if !ok {
		t.Fatal("chrdatd2 has a readied long sword")
	}
	if !strings.Contains(weapon.Name, "Long Sword") {
		t.Fatalf("readied weapon is %q", weapon.Name)
	}
	stats, err := a.weaponCombatStats(weapon, member, 0x28)
	if err != nil {
		t.Fatal(err)
	}
	if stats.DamageCount == 0 || stats.DamageSides == 0 {
		t.Fatalf("weapon damage %dd%d", stats.DamageCount, stats.DamageSides)
	}
}
