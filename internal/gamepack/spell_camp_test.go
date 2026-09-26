package gamepack

import "testing"

// 參數表 `+7` 在戰鬥外是 overlay-22 `0A88h` 的收表方式（spec 098〈營地施法〉）。
// 六十七格只用到 0／1／2／4 四個值——`0A88h` 只認 1、2、4，其餘回 0，而 0 在 entry 5
// `0C3Fh` 先被擋成 "is a combat-only spell"。出現別的值就代表讀錯了那一格。
func TestCampTargetFollowsOverlay22Entry4(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, check := range []struct {
		id   uint8
		want CampTargetKind
	}{
		{SpellIDBless, CampTargetParty},
		{SpellIDHaste, CampTargetParty},
		{50, CampTargetParty}, // Invisibility, 10' Radius
		{19, CampTargetSelf},  // Shield
		{18, CampTargetSelf},  // Read Magic
		{SpellIDPrayer, CampTargetSelf},
		{30, CampTargetPick}, // Invisibility
		{SpellIDCureLightWound, CampTargetPick},
		{SpellIDMagicMissile, CampTargetCombatOnly},
		{SpellIDCurse, CampTargetCombatOnly},
	} {
		if got := parameters[check.id].CampTarget(); got != check.want {
			t.Errorf("法術 %d 的營地收表是 %d，原版是 %d", check.id, got, check.want)
		}
	}
	for id := 1; id < len(parameters); id++ {
		switch parameters[id].CampTarget() {
		case CampTargetCombatOnly, CampTargetSelf, CampTargetPick, CampTargetParty:
		default:
			t.Errorf("法術 %d 的 +7 是 %d，0A88h 不認得", id, parameters[id].CampTarget())
		}
	}
}
