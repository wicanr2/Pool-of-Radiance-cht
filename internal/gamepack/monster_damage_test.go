package gamepack

import (
	"path/filepath"
	"strings"
	"testing"
)

// plausibleDieSides 是 D&D 會用到的骰面，加上 0（那一形態沒有骰子）與 1
//（毒蛙的 1d1，原始資料就是這樣寫的）。
var plausibleDieSides = map[uint8]bool{0: true, 1: true, 2: true, 3: true, 4: true,
	6: true, 8: true, 10: true, 12: true, 20: true}

// 每一筆怪物記錄的兩種攻擊形態都要是合理的骰子。
//
// 這一則的價值在**負對照**：同一份資料改讀執行期那一段（`+114h..+11Ah`）
// 就會出現 48 處不合理的值——那一段在樣板檔裡沒有初始化，殘留的是文字
//（QUICKLINGS 的 `41 00 44 00 43 00` 是 `'A' 'D' 'C'`，讀成骰子是 65d68+67）。
// 兩邊一起量才分得出「來源欄位讀對了」與「剛好沒踩到」。
func TestMonsterDamageDiceAreAllPlausible(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	records := 0
	stale := 0
	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			record, err := ReadDOSMonsterRecord(zipPath, archive, uint8(id))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(record.Name)
			if name == "" {
				continue
			}
			records++
			for slot := uint8(1); slot <= MonsterAttackSlots; slot++ {
				damage, err := record.AttackDamage(slot)
				if err != nil {
					t.Fatal(err)
				}
				if damage.Count > 8 || !plausibleDieSides[damage.Sides] ||
					damage.Bonus < -4 || damage.Bonus > 12 {
					t.Errorf("mon%d/%d %s 形態 %d 是 %dd%d%+d，不像骰子",
						archive, id, name, slot, damage.Count, damage.Sides, damage.Bonus)
				}
				if damage.Count != 0 && damage.Sides == 0 {
					t.Errorf("mon%d/%d %s 形態 %d 有顆數沒有面數", archive, id, name, slot)
				}
			}
			// 負對照：執行期那一段與來源不符的筆數要真的存在，
			// 否則這一則等於沒有在證明任何事。
			for slot := 1; slot <= MonsterAttackSlots; slot++ {
				if record.Raw[MonsterDamageCountBase+slot] != record.Raw[0x114+slot] ||
					record.Raw[MonsterDamageSidesBase+slot] != record.Raw[0x116+slot] ||
					record.Raw[MonsterDamageBonusBase+slot] != record.Raw[0x118+slot] {
					stale++
					break
				}
			}
		}
	}
	if records == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	if stale == 0 {
		t.Error("沒有一筆記錄的執行期區段與來源欄位不符；那一段若已經是對的，這一則就沒有在證明什麼")
	}
	t.Logf("%d 筆記錄的骰子都合理，其中 %d 筆的執行期區段是過期的", records, stale)
}

// 幾隻攻擊形態明確的怪物，逐項對回 AD&D 一版。
func TestKnownMonstersCarryBothAttackForms(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	byName := map[string]MonsterRecord{}
	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			record, err := ReadDOSMonsterRecord(zipPath, archive, uint8(id))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(record.Name)
			if name != "" {
				if _, seen := byName[name]; !seen {
					byName[name] = record
				}
			}
		}
	}
	if len(byName) == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for _, want := range []struct {
		name    string
		rates   [2]uint8
		damage  [2]MonsterAttackDamage
		comment string
	}{
		{"TROLL", [2]uint8{4, 2},
			[2]MonsterAttackDamage{{Count: 1, Sides: 4, Bonus: 4}, {Count: 2, Sides: 6}},
			"AD&D 一版的爪／爪／咬"},
		{"ORC", [2]uint8{2, 0},
			[2]MonsterAttackDamage{{Count: 1, Sides: 8}, {}},
			"一回合一擊 1d8"},
		{"POISONOUS FROG", [2]uint8{0, 2},
			[2]MonsterAttackDamage{{}, {Count: 1, Sides: 1}},
			"第一形態是特殊攻擊，傷害在第二形態"},
		{"TYRANITHRAXUS", [2]uint8{4, 2},
			[2]MonsterAttackDamage{{Count: 1, Sides: 6}, {Count: 4, Sides: 6}},
			"最後一戰：兩爪加一口"},
		{"THRI-KREEN", [2]uint8{8, 2},
			[2]MonsterAttackDamage{{Count: 1, Sides: 4}, {Count: 1, Sides: 4, Bonus: 1}},
			"加值是 0；執行期那一段殘留的 66h 會讀成 +102"},
		{"QUICKLINGS", [2]uint8{6, 0},
			[2]MonsterAttackDamage{{Count: 1, Sides: 4}, {}},
			"一回合三擊 1d4；執行期那一段殘留的是文字 'A' 'D' 'C'，會讀成 65d68+67"},
	} {
		record, ok := byName[want.name]
		if !ok {
			t.Errorf("找不到 %s", want.name)
			continue
		}
		for slot := uint8(1); slot <= MonsterAttackSlots; slot++ {
			rate, err := record.BaseAttackRate(slot)
			if err != nil {
				t.Fatal(err)
			}
			if rate != want.rates[slot-1] {
				t.Errorf("%s 形態 %d 的攻擊次數編碼是 %d，要的是 %d（%s）",
					want.name, slot, rate, want.rates[slot-1], want.comment)
			}
			damage, err := record.AttackDamage(slot)
			if err != nil {
				t.Fatal(err)
			}
			if damage != want.damage[slot-1] {
				t.Errorf("%s 形態 %d 是 %dd%d%+d，要的是 %dd%d%+d（%s）",
					want.name, slot, damage.Count, damage.Sides, damage.Bonus,
					want.damage[slot-1].Count, want.damage[slot-1].Sides,
					want.damage[slot-1].Bonus, want.comment)
			}
		}
	}
}
