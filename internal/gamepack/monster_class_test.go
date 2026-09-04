package gamepack_test

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 怪物記錄的 `+2Fh` 與玩家角色是同一個欄位：複合職業碼。
//
// 這一則是 overlay-09 entry 5 那條 `記錄 +2Fh == 5` 分支的依據——5 是純法師，
// 所以那一支是「不逃跑的純法師交給 entry 6」，不是某個未知旗標。
func TestMonsterClassCodeIsTheCharacterClassCode(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")

	// 建角能選到的複合職業碼就是這個欄位的合法值域。
	legal := map[uint8]string{}
	for _, race := range creation.Races {
		for _, choice := range creation.ClassesForRace(race.ID) {
			legal[choice.DOSCode] = choice.Label
		}
	}
	if len(legal) == 0 {
		t.Fatal("建角目錄沒有職業，這一則的值域來源壞了")
	}

	// 正對照：四個純職業各挑一隻叫得出名字的。
	want := map[string]uint8{
		"7TH LVL CLERIC": 0, // Cleric
		"OGRE":           2, // Fighter
		"LEVEL 6 MU":     5, // Magic-User
		"6TH LVL THIEF":  6, // Thief
	}
	seen := map[string]bool{}
	counts := map[uint8]int{}
	scanned := 0

	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			record, err := gamepack.ReadDOSMonsterRecord(zipPath, archive, uint8(id))
			if err != nil {
				continue
			}
			scanned++
			code := record.Raw[gamepack.ClassCodeOffset]
			counts[code]++
			if _, ok := legal[code]; !ok {
				t.Errorf("mon%d/%d %s 的 +2Fh 是 %d，不是任何一個合法的複合職業碼",
					archive, id, strings.TrimSpace(record.Name), code)
			}
			name := strings.TrimSpace(record.Name)
			if expected, interesting := want[name]; interesting {
				if code != expected {
					t.Errorf("%s 的 +2Fh 是 %d，應該是 %d（%s）",
						name, code, expected, legal[expected])
				}
				seen[name] = true
			}
		}
	}
	if scanned == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("原版裡找不到 %s，這一則的正對照失效了", name)
		}
	}

	codes := make([]int, 0, len(counts))
	for code := range counts {
		codes = append(codes, int(code))
	}
	sort.Ints(codes)
	for _, code := range codes {
		t.Logf("+2Fh = %2d %-26s %3d 筆", code, legal[uint8(code)], counts[uint8(code)])
	}
	// 負對照：值域沒有被「所有值都合法」這種空話蓋掉——實際只用到七個。
	if len(codes) < 4 {
		t.Errorf("只出現 %d 種職業碼，掃描面可能有洞", len(codes))
	}
	if _, ok := counts[gamepack.PureFighterClassCode]; !ok {
		t.Error("一隻純戰士都沒有，這一則的掃描面有洞")
	}
}
