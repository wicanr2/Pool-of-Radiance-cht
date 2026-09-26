package gamepack

import (
	"reflect"
	"testing"
)

// 原版收據（docs/audit/dosgolem-loot-experience.json）：市政廳交完貧民窟的件，公款
// 250 金、50 白金、1 珠寶 → 250 ＋ 50×5 ＋ 2200 ＝ 2700。
func TestLootExperienceMatchesTheOriginalReceipt(t *testing.T) {
	if got := LootExperience([7]uint32{3: 250, 4: 50, 6: 1}, nil); got != 2700 {
		t.Fatalf("slums reward folds into %d XP, want 2700", got)
	}
	// 銅 ÷200、銀 ÷20、琥珀金 ÷2 都是截去；寶石 ×250。
	if got := LootExperience([7]uint32{399, 39, 3, 0, 0, 2, 0}, nil); got != 1+1+1+500 {
		t.Fatalf("mixed coins fold into %d XP, want 503", got)
	}
}

// `02DCh`：加值 > 0 才算；`imul cx` 之後 `cwd` 只留低 16 位元，所以 +82 起變負數。
func TestLootExperienceItemPlusesWrapAtSixteenBits(t *testing.T) {
	if got := LootExperience([7]uint32{}, []int8{1, 3, 0, -2}); got != 1600 {
		t.Fatalf("items +1 +3 +0 -2 fold into %d XP, want 1600", got)
	}
	// 82 × 400 = 32800 → int16 −32736。
	if got := int32(LootExperience([7]uint32{3: 40000}, []int8{82})); got != 40000-32736 {
		t.Fatalf("a +82 item folds into %d, want %d", got-40000, -32736)
	}
}

// `1295h`：兩個玩家角色、一個份額 3 的傭兵 → 總份數 5；每份取一個位元組。
func TestHideNPCSharesTakesByteSizedShares(t *testing.T) {
	members := []NPCShareMember{{}, {}, {NPC: true, Share: 3}}
	pool, hiders := HideNPCShares([7]uint32{3: 250, 4: 50, 6: 1}, members)
	// 金 250/5 = 50 → 扣 150；白金 50/5 = 10 → 扣 30；珠寶 1/5 = 0 → 不扣。
	if want := ([7]uint32{3: 100, 4: 20, 6: 1}); pool != want {
		t.Fatalf("pool after the share %v, want %v", pool, want)
	}
	if !reflect.DeepEqual(hiders, []int{2}) {
		t.Fatalf("hiders %v, want [2]", hiders)
	}
	// 每份存成位元組：10000/5 = 2000 → 0xD0 = 208，扣 624。
	pool, _ = HideNPCShares([7]uint32{3: 10000}, members)
	if pool[3] != 10000-208*3 {
		t.Fatalf("gold after a byte-wrapped share %d, want %d", pool[3], 10000-208*3)
	}
}

// 倒下的 NPC（`+10Ch != 0`）不分，只佔一份；沒有人分的時候什麼都不動。`+85h` 低三位元
// 是 0、但整個位元組非 0 的 NPC 仍會被列名（`13DEh` 比的是整個位元組）。
func TestHideNPCSharesSkipsDownedNPCs(t *testing.T) {
	pool := [7]uint32{3: 100}
	after, hiders := HideNPCShares(pool, []NPCShareMember{{}, {NPC: true, Status: 4, Share: 7}})
	if after != pool || hiders != nil {
		t.Fatalf("a downed NPC took a share: %v %v", after, hiders)
	}
	after, hiders = HideNPCShares(pool, []NPCShareMember{{NPC: true, Share: 0xFF}, {NPC: true, Share: 8}})
	// 總份數 7 ＋ 0 ＝ 7、NPC 份數 7：100/7 = 14 → 扣 98。
	if after[3] != 2 || !reflect.DeepEqual(hiders, []int{0, 1}) {
		t.Fatalf("pool %v hiders %v, want 2 gold left and both listed", after, hiders)
	}
	if after, hiders := HideNPCShares([7]uint32{}, []NPCShareMember{{NPC: true, Share: 3}}); after != ([7]uint32{}) || hiders != nil {
		t.Fatalf("an empty pool reported hiders %v", hiders)
	}
}
