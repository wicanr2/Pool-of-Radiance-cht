package gamepack

import (
	"path/filepath"
	"testing"
)

func missileTypes(t *testing.T) *ItemTypeTable {
	t.Helper()
	types, err := ReadDOSItemTypeTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	return types
}

func missileItem(itemType uint8, readied bool, count uint8) []byte {
	item := make([]byte, MonsterItemRecordSize)
	item[ItemTypeOffset] = itemType
	if readied {
		item[ItemReadiedOffset] = 1
	}
	item[ItemCountOffset] = count
	return item
}

// overlay-25 entry 45（`2EB1h`）：弓看 `+0F8h`（型別 49h 的箭），弩看 `+0FCh`（1Ch 的矢），
// 匕首之類射出去的是自己；旗標恰好 0Ah 的不必彈藥。
func TestMissileGearFollowsEntry45(t *testing.T) {
	types := missileTypes(t)
	for _, testCase := range []struct {
		name        string
		items       [][]byte
		ranged      bool
		thrownMelee bool
		canFire     bool
		ammunition  int
		rate        uint8
	}{
		{"short bow with readied arrows", [][]byte{missileItem(0x2c, true, 0), missileItem(ItemTypeArrow, true, 12)}, true, false, true, 1, 4},
		{"short bow, arrows carried but not readied", [][]byte{missileItem(0x2c, true, 0), missileItem(ItemTypeArrow, false, 12)}, true, false, false, -1, 4},
		{"short bow, no arrows", [][]byte{missileItem(0x2c, true, 0)}, true, false, false, -1, 4},
		// 2Eh 的旗標是 8Ah：看的是弩矢那一格，箭不算。
		{"crossbow with arrows only", [][]byte{missileItem(0x2e, true, 0), missileItem(ItemTypeArrow, true, 12)}, true, false, false, -1, 2},
		{"crossbow with quarrels", [][]byte{missileItem(ItemTypeQuarrel, true, 9), missileItem(0x2e, true, 0)}, true, false, true, 0, 2},
		// 匕首（型別 02h，旗標 14h）：丟出去的是自己，貼身也能砍。
		{"dagger", [][]byte{missileItem(0x02, true, 0)}, true, true, true, 0, 2},
		// 型別 2Fh 旗標 0Ah：不需要彈藥。
		{"no-ammunition missile", [][]byte{missileItem(0x2f, true, 0)}, true, false, true, -1, 2},
		{"long sword", [][]byte{missileItem(0x24, true, 0)}, false, false, false, -1, 2},
		{"unarmed", [][]byte{missileItem(ItemTypeArrow, true, 12)}, false, false, false, -1, 0},
	} {
		gear, err := MissileGearOf(testCase.items, types)
		if err != nil {
			t.Fatal(err)
		}
		if gear.Ranged != testCase.ranged || gear.ThrownMelee != testCase.thrownMelee ||
			gear.CanFire != testCase.canFire || gear.Ammunition != testCase.ammunition ||
			(testCase.rate != 0 && gear.RateOfFire != testCase.rate) {
			t.Errorf("%s: %+v", testCase.name, gear)
		}
	}
}

// overlay-13 `0DD1h..0E06h`：一次射幾發以彈藥數量封頂，數量 0 當 1。
func TestVolleyLimitIsTheAmmunitionCount(t *testing.T) {
	types := missileTypes(t)
	items := [][]byte{missileItem(0x2c, true, 0), missileItem(ItemTypeArrow, true, 1)}
	gear, err := MissileGearOf(items, types)
	if err != nil {
		t.Fatal(err)
	}
	if limit := gear.VolleyLimit(items); limit != 1 {
		t.Fatalf("one arrow allows %d shots", limit)
	}
	items[1][ItemCountOffset] = 0
	if limit := gear.VolleyLimit(items); limit != 1 {
		t.Fatalf("an arrow record with count 0 allows %d shots, want 1", limit)
	}
}

// overlay-13 `19D5h..1A96h`：扣實際射出的發數；到 0 就摘掉，丟得出去又能近戰的留一件在地上
// （`+34h` 清 0），`+3Eh == 89h` 的不留。
func TestSpendAmmunitionRemovesAndDrops(t *testing.T) {
	arrows := missileItem(ItemTypeArrow, true, 7)
	if spend := SpendAmmunition(arrows, 2, false); spend.Remove || arrows[ItemCountOffset] != 5 {
		t.Fatalf("7 arrows minus 2 shots: %+v count %d", spend, arrows[ItemCountOffset])
	}
	arrows[ItemCountOffset] = 1
	if spend := SpendAmmunition(arrows, 1, false); !spend.Remove || spend.Dropped != nil {
		t.Fatalf("last arrow: %+v", spend)
	}
	dagger := missileItem(0x02, true, 0)
	spend := SpendAmmunition(dagger, 1, true)
	if !spend.Remove || spend.Dropped == nil || spend.Dropped[ItemReadiedOffset] != 0 || dagger[ItemReadiedOffset] != 1 {
		t.Fatalf("thrown dagger: %+v", spend)
	}
	dagger[ItemEffectOffset] = 0x89
	if spend := SpendAmmunition(dagger, 1, true); !spend.Remove || spend.Dropped != nil {
		t.Fatalf("thrown dagger with +3Eh 89h: %+v", spend)
	}
}

// `29h`（overlay-12 entry 40 `108Dh`／`0FB7h`）：非魔法、兩格外、而且擲一次 d100。
func TestNormalMissileAvoidedOnlyStopsNonMagicalShotsFromAfar(t *testing.T) {
	types := missileTypes(t)
	protected := EffectList{{Code: ProtectionFromNormalMissilesEffectCode}}
	items := [][]byte{missileItem(0x2c, true, 0), missileItem(ItemTypeArrow, true, 7)}
	gear, err := MissileGearOf(items, types)
	if err != nil {
		t.Fatal(err)
	}
	rolls := 0
	roll := func(count, sides int) int {
		rolls++
		if count != 1 || sides != 100 {
			t.Fatalf("rolled %dd%d, want 1d100", count, sides)
		}
		return 100
	}
	if !NormalMissileAvoided(protected, gear, items, 3, roll) || rolls != 1 {
		t.Fatalf("plain arrow from 3 away: rolls %d", rolls)
	}
	if NormalMissileAvoided(protected, gear, items, 1, roll) || rolls != 1 {
		t.Fatal("adjacent shot was stopped or rolled")
	}
	if NormalMissileAvoided(nil, gear, items, 3, roll) || rolls != 1 {
		t.Fatal("unprotected target stopped the arrow")
	}
	items[1][ItemPlusOffset] = 1
	if NormalMissileAvoided(protected, gear, items, 3, roll) || rolls != 1 {
		t.Fatal("+1 arrow was stopped")
	}
	// 沒有彈藥時看的是武器本身（`1066h`）。
	bare := [][]byte{missileItem(0x2c, true, 0)}
	gear, err = MissileGearOf(bare, types)
	if err != nil {
		t.Fatal(err)
	}
	if !NormalMissileAvoided(protected, gear, bare, 3, roll) || rolls != 2 {
		t.Fatal("a plain bow without arrows is still a normal missile")
	}
}

// overlay-09 entry 9（`13D5h`）用原版獸人頭目身上的東西：MON2 block 15 那一隻穿著釘頭錘（0Ch +1）
// 與短弓（2Bh），block 14 的三隻兩件武器都沒穿，箭都穿著。
func TestChooseAIGearFollowsEntry9(t *testing.T) {
	types := missileTypes(t)
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	record, err := ReadDOSMonsterRecord(zipPath, 2, 14)
	if err != nil {
		t.Fatal(err)
	}
	in := AIGearInput{ClassMask: record.Raw[ClassUseMaskOffset], NaturalScore: AIGearNaturalScore(record.Raw[:])}
	if in.ClassMask != 0x08 || in.NaturalScore != 8 {
		t.Fatalf("orc leader +0B0h %#02x, natural score %d", in.ClassMask, in.NaturalScore)
	}
	readied := func(items [][]byte) []uint8 {
		var types []uint8
		for _, item := range items {
			if item[ItemReadiedOffset] != 0 {
				types = append(types, item[ItemTypeOffset])
			}
		}
		return types
	}
	load := func(block uint8) [][]byte {
		items, err := ReadDOSMonsterItems(zipPath, 2, block)
		if err != nil {
			t.Fatal(err)
		}
		return items
	}
	for _, testCase := range []struct {
		name     string
		block    uint8
		adjacent bool
		noArrows bool
		want     []uint8
	}{
		// 身邊沒人：短弓分數 12（1d6 + (4 − 1) × 2）大於近戰 19 的一半，有箭 → 穿上弓。
		{"unworn bow, nobody adjacent", 14, false, false, []uint8{0x37, ItemTypeArrow, 0x2b}},
		// 身邊有人：挑近戰的 23h（2d4 + 8 + 3 = 19，比徒手 1d8 的 8 高）。
		{"unworn weapons, enemy adjacent", 14, true, false, []uint8{0x23, 0x37, ItemTypeArrow}},
		{"no arrows, nobody adjacent", 14, false, true, []uint8{0x23, 0x37}},
		// 穿著弓又穿著釘頭錘：貼身時挑釘頭錘，先卸弓，再對已經穿著的釘頭錘叫一次 Ready——那一下把它也卸掉。
		{"bow and mace readied, enemy adjacent", 15, true, false, []uint8{0x37, ItemTypeArrow}},
		// 身邊沒人時弓本來就是挑中的那一件，不換；但釘頭錘加弓佔了三隻手（+100h = 3 > 2），
		// 又沒有盾可卸，`187Bh` 對挑中的弓叫 Ready——弓被卸下，留下釘頭錘。
		{"bow and mace readied, nobody adjacent", 15, false, false, []uint8{0x0c, 0x37, ItemTypeArrow}},
		{"bow and mace readied, no arrows", 15, false, true, []uint8{0x37}},
	} {
		items := load(testCase.block)
		if testCase.noArrows {
			kept := items[:0]
			for _, item := range items {
				if item[ItemTypeOffset] != ItemTypeArrow {
					kept = append(kept, item)
				}
			}
			items = kept
		}
		in.AdjacentEnemies = testCase.adjacent
		if _, _, err := ChooseAIGear(items, types, in); err != nil {
			t.Fatal(err)
		}
		got := readied(items)
		if string(got) != string(testCase.want) {
			t.Errorf("%s: readied % x, want % x", testCase.name, got, testCase.want)
		}
	}
}
