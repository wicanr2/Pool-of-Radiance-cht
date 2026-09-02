package gamepack

import "testing"

// 表要對得上 overlay-15 那串比較鏈解出來的十五個碼。
func TestEffectNamesMatchTheOriginalChain(t *testing.T) {
	want := map[uint8]string{
		4: "Studying Manual of Bodily Health", 7: "Training with Manual of Bodily Health",
		27: "Feather Fall", 31: "Helpless", 35: "Prayer", 44: "Cause Disease",
		50: "Dreaded Mummy Disease", 53: "Funky--", 54: "Repulsed", 55: "Poisoned",
		59: "Regenerating", 61: "Fire Resistance", 71: "Invisible", 72: "Camouflaged",
		89: "Displaced",
	}
	if len(EffectNames()) != len(want) {
		t.Fatalf("表有 %d 筆，原版那串鏈是 %d 筆", len(EffectNames()), len(want))
	}
	for code, name := range want {
		entry, ok := EffectNameFor(code)
		if !ok {
			t.Errorf("效果碼 %#02x 查不到", code)
			continue
		}
		if entry.Name != name {
			t.Errorf("效果碼 %#02x 是 %q，原版是 %q", code, entry.Name, name)
		}
		if entry.Text == "" {
			t.Errorf("效果碼 %#02x 沒有中文", code)
		}
	}
	// 沒有顯示名稱的碼要回 false，不要硬編一個出來。
	// 26h 是 Gauntlets of Ogre Power（spec 069），它不在那串鏈裡。
	if entry, ok := EffectNameFor(0x26); ok {
		t.Errorf("效果碼 26h 不該有名稱，卻拿到 %q", entry.Name)
	}
}

// 3Dh 是 spec 069 那條推論的直接證據：原版字面上就叫 Fire Resistance。
func TestFireResistanceConfirmsTheRingInference(t *testing.T) {
	entry, ok := EffectNameFor(0x3d)
	if !ok || entry.Name != "Fire Resistance" {
		t.Fatalf("效果碼 3Dh 應該是 Fire Resistance，拿到 %+v（找到 %t）", entry, ok)
	}
}
