package gamepack_test

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 力量索引把 18 點的百分比折成六個段。原版把 18/00 記成百分比 100。
func TestStrengthTableIndex(t *testing.T) {
	for _, item := range []struct {
		strength    int
		exceptional int
		want        uint8
	}{
		{3, 0, 3}, {17, 0, 17},
		{18, 0, 0x12}, {18, 1, 0x13}, {18, 50, 0x13},
		{18, 51, 0x14}, {18, 75, 0x14}, {18, 76, 0x15}, {18, 90, 0x15},
		{18, 91, 0x16}, {18, 99, 0x16}, {18, 100, 0x17},
		{19, 0, 0x18}, {25, 0, 0x1e},
	} {
		got, err := gamepack.StrengthTableIndex(item.strength, item.exceptional)
		if err != nil {
			t.Fatalf("strength %d/%d: %v", item.strength, item.exceptional, err)
		}
		if got != item.want {
			t.Fatalf("strength %d/%d gave index %#02x, want %#02x",
				item.strength, item.exceptional, got, item.want)
		}
	}
	if _, err := gamepack.StrengthTableIndex(26, 0); err == nil {
		t.Fatal("strength 26 was accepted")
	}
	if _, err := gamepack.StrengthTableIndex(18, 101); err == nil {
		t.Fatal("an exceptional strength of 101 was accepted")
	}
}

// 三張修正表逐段照抄 overlay-25 的分支，包含原版與規則書不同的邊界：
// 索引 0 與 31 以上都回 0，敏捷 0..2 回 -4。
func TestAbilityAdjustmentTables(t *testing.T) {
	for index, want := range map[uint8]int{
		0: 0, 1: -3, 3: -3, 4: -2, 5: -2, 6: -1, 7: -1, 8: 0, 16: 0,
		0x11: 1, 0x13: 1, 0x14: 2, 0x16: 2, 0x17: 3, 0x19: 3,
		0x1a: 4, 0x1b: 4, 0x1c: 5, 0x1e: 7, 0x1f: 0,
	} {
		if got := gamepack.StrengthHitAdjustment(index); got != want {
			t.Fatalf("strength hit index %#02x gave %d, want %d", index, got, want)
		}
	}
	for index, want := range map[uint8]int{
		0: 0, 1: -2, 2: -2, 3: -1, 5: -1, 6: 0, 0x0f: 0,
		0x10: 1, 0x11: 1, 0x12: 2, 0x13: 3, 0x14: 3, 0x1d: 12, 0x1e: 14, 0x1f: 0,
	} {
		if got := gamepack.StrengthDamageAdjustment(index); got != want {
			t.Fatalf("strength damage index %#02x gave %d, want %d", index, got, want)
		}
	}
	for dexterity, want := range map[int]int{
		0: -4, 2: -4, 3: -3, 4: -2, 5: -1, 6: 0, 15: 0,
		16: 1, 17: 2, 18: 3, 19: 3, 20: 3, 21: 4, 23: 4, 24: 5, 25: 5, 26: 0,
	} {
		if got := gamepack.DexterityMissileAdjustment(dexterity); got != want {
			t.Fatalf("dexterity %d gave %d, want %d", dexterity, got, want)
		}
	}
}

// 走完整條規則：一級戰士拿墓園那把 Two-Handed Sword +1，力量 18/00。
// 基礎 internal THAC0 是 28h（typed 20），力量命中 +3、武器 +1 → 2Ch（typed 16）；
// 傷害是 1d10，加值 6（力量）＋1（武器）。
func TestWeaponCombatStatsForTheGraveyardSword(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	stats, err := gamepack.WeaponCombatStats(table, 0x26, 1, gamepack.WeaponBearer{
		BaseThac0Internal:     0x28,
		Strength:              18,
		ExceptionalStrength:   100,
		Dexterity:             12,
		AbilityBonusesEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Thac0Internal != 0x2c {
		t.Fatalf("internal THAC0 %#02x, want 0x2c", stats.Thac0Internal)
	}
	if stats.DamageCount != 1 || stats.DamageSides != 10 {
		t.Fatalf("damage %dd%d, want 1d10", stats.DamageCount, stats.DamageSides)
	}
	if stats.DamageBonus != 7 {
		t.Fatalf("damage bonus %d, want 7", stats.DamageBonus)
	}
}

// 沒有力量加值的角色不套用兩個力量修正，但敏捷的投射修正不受那個閘門管
// ——原版那支沒有這個閘門。
func TestAbilityBonusGateOnlyCoversStrength(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	gated, err := gamepack.WeaponCombatStats(table, 0x26, 0, gamepack.WeaponBearer{
		BaseThac0Internal: 0x28, Strength: 18, ExceptionalStrength: 100, Dexterity: 18,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gated.Thac0Internal != 0x28 || gated.DamageBonus != 0 {
		t.Fatalf("the gate did not suppress the strength bonuses: %+v", gated)
	}

	// 找一筆帶敏捷旗標的型別，確認閘門關著時它仍然生效。
	missile := -1
	for candidate := 0; candidate < gamepack.ItemTypeCount; candidate++ {
		entry, err := table.Entry(uint8(candidate))
		if err != nil {
			t.Fatal(err)
		}
		if entry.Flags()&gamepack.ItemTypeFlagDexterityToHit != 0 &&
			entry.Flags()&gamepack.ItemTypeFlagStrengthBonuses == 0 {
			missile = candidate
			break
		}
	}
	if missile < 0 {
		t.Fatal("the table has no dexterity-only weapon type; the flag reading is wrong")
	}
	stats, err := gamepack.WeaponCombatStats(table, uint8(missile), 0, gamepack.WeaponBearer{
		BaseThac0Internal: 0x28, Dexterity: 18,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Thac0Internal != 0x2b {
		t.Fatalf("dexterity 18 gave internal THAC0 %#02x, want 0x2b", stats.Thac0Internal)
	}
}

// 型別表缺席時失敗即關閉：沒有表就沒有骰數，回零骰會被當成「這把武器不痛」。
func TestWeaponCombatStatsNeedsATable(t *testing.T) {
	if _, err := gamepack.WeaponCombatStats(nil, 0, 0, gamepack.WeaponBearer{}); err == nil {
		t.Fatal("a missing item type table was accepted")
	}
}

// `record[+2Eh] == 2` 是精靈（spec 003 的種族碼表），加值的四個武器型別是
// 長劍 `24h`、短劍 `25h` 與 `29h..2Ch` 那一族弓（`2Ch` 是 Short Bow）。
// 這正是 AD&D 精靈的武器加值。
//
// 只影響命中不影響傷害：那個 +1 在 `modifier` 已經加進傷害之後才加。
func TestElfWeaponBonusHitsOnlyTheListedTypes(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, item := range []struct {
		itemType uint8
		what     string
		bonus    bool
	}{
		{0x24, "Long Sword", true},
		{0x25, "Short Sword", true},
		{0x29, "弓那一族的第一件", true},
		{0x2c, "Short Bow", true},
		{0x2d, "弓那一族之外的下一個型別", false},
		{0x17, "Mace", false},
		{0x26, "Two-Handed Sword", false},
	} {
		bearer := gamepack.WeaponBearer{BaseThac0Internal: 0x28, Strength: 12, Dexterity: 12}
		plain, err := gamepack.WeaponCombatStats(table, item.itemType, 0, bearer)
		if err != nil {
			t.Fatalf("%s: %v", item.what, err)
		}
		bearer.ClassBonusApplies = true // 記錄 +2Eh 是 2，也就是精靈
		elf, err := gamepack.WeaponCombatStats(table, item.itemType, 0, bearer)
		if err != nil {
			t.Fatalf("%s: %v", item.what, err)
		}
		want := 0
		if item.bonus {
			want = 1
		}
		if got := int(elf.Thac0Internal) - int(plain.Thac0Internal); got != want {
			t.Fatalf("%s（型別 %02Xh）的精靈命中加值是 %d，預期 %d",
				item.what, item.itemType, got, want)
		}
		if elf.DamageBonus != plain.DamageBonus {
			t.Fatalf("%s 的精靈加值不該影響傷害：%d 對 %d",
				item.what, elf.DamageBonus, plain.DamageBonus)
		}
	}
}
