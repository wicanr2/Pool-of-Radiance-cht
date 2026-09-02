package gamepack

import "testing"

// 目標模式的分組本身就是語意證據：模式 0Ah 正好是那四支整邊的法術，
// 模式 0 全是自身增益。任何一格讀錯，分組就會散掉。
func TestSpellTargetModeGrouping(t *testing.T) {
	parameters, err := ReadDOSSpellParameters(poolZipPath())
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	counts := map[SpellTargetMode]int{}
	for id := 1; id < len(parameters); id++ {
		counts[parameters[id].TargetMode()]++
	}
	for mode, want := range map[SpellTargetMode]int{
		SpellTargetSelf: 21, SpellTargetSingle: 30, SpellTargetHold: 1,
		SpellTargetHoldAlt: 2, SpellTargetBolt: 1, SpellTargetArea: 5,
		SpellTargetWholeSide: 4, SpellTargetBurst: 2, SpellTargetPick: 1,
	} {
		if counts[mode] != want {
			t.Errorf("模式 %#02x 有 %d 支，原版是 %d 支", mode, counts[mode], want)
		}
	}
	// 整邊的那四支：祝福、詛咒、急速、緩速。
	side := map[int]bool{1: true, 2: true, 48: true, 55: true}
	for id := 1; id < len(parameters); id++ {
		if got := parameters[id].AffectsWholeSide(); got != side[id] {
			t.Errorf("法術 %d 的整邊判定是 %t，預期 %t", id, got, side[id])
		}
	}
	// 這一條與逐支讀出來的處理常式互相印證：祝福與詛咒是從碼讀出來走
	// 0F35h 那條整邊的路，而參數表把它們和急速、緩速歸在同一個模式。
	for _, id := range []uint8{SpellIDBless, SpellIDCurse} {
		if !parameters[id].AffectsWholeSide() {
			t.Errorf("法術 %d 的處理常式是整邊的，參數表卻不是", id)
		}
	}
	if !parameters[47].AffectsArea() || !parameters[51].AffectsArea() {
		t.Error("火球與閃電束應該算範圍")
	}
	if parameters[3].AffectsArea() {
		t.Error("治療輕傷不該算範圍")
	}
	if parameters[19].TargetMode() != SpellTargetSelf {
		t.Errorf("法術護盾應該是自身模式，拿到 %#02x", parameters[19].TargetMode())
	}
}
