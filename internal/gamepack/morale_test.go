package gamepack

import (
	"path/filepath"
	"testing"
)

// `+84h` 的位元 7 決定做不做判定，低七位 × 2 是士氣值，超過 66h 就當成 0（`113Eh..1164h`）。
func TestMoraleValueReadsTheCheckedBitAndTheCeiling(t *testing.T) {
	for _, tc := range []struct {
		raw     uint8
		value   uint8
		checked bool
	}{
		{0x00, 0, false},
		{0x7F, 0, false},
		{0xB2, 100, true}, // 0x32 × 2
		{0xB3, 102, true}, // 0x33 × 2 = 66h，剛好不超過
		{0xB4, 0, true},   // 0x34 × 2 = 104 > 66h
		{0xFF, 0, true},
	} {
		value, checked := MoraleValue(tc.raw)
		if value != tc.value || checked != tc.checked {
			t.Errorf("MoraleValue(%02X) = %d,%v, want %d,%v", tc.raw, value, checked, tc.value, tc.checked)
		}
	}
}

// 群組 11h：01h 加 5（byte 繞回），02h 減 5 但不低於 0（overlay-12 entry 5／6）。
func TestAdjustMoraleFollowsTheTwoHandlers(t *testing.T) {
	if got := AdjustMorale(100, true, false); got != 105 {
		t.Errorf("boost 100 → %d, want 105", got)
	}
	if got := AdjustMorale(253, true, false); got != 2 {
		t.Errorf("boost wraps as a byte: 253 → %d, want 2", got)
	}
	if got := AdjustMorale(3, false, true); got != 0 {
		t.Errorf("drop floors at 0: 3 → %d", got)
	}
	if got := AdjustMorale(100, true, true); got != 100 {
		t.Errorf("boost then drop: %d, want 100", got)
	}
}

// overlay-13 entry 25：(目前 × 20 ÷ 上限) × 5，也就是往下取到 5 的倍數；上限 0 不寫。
func TestSideMoraleRoundsDownToFivePercent(t *testing.T) {
	for _, tc := range []struct {
		current, maximum uint16
		want             uint8
	}{
		{30, 30, 100}, {29, 30, 95}, {2, 30, 5}, {1, 30, 0}, {0, 30, 0},
	} {
		got, ok := SideMorale(tc.current, tc.maximum)
		if !ok || got != tc.want {
			t.Errorf("SideMorale(%d,%d) = %d,%v, want %d", tc.current, tc.maximum, got, ok, tc.want)
		}
	}
	if _, ok := SideMorale(0, 0); ok {
		t.Error("with no foe hit points the original leaves DS:6D22h alone")
	}
}

func TestResolveMoraleBranches(t *testing.T) {
	base := MoraleCheck{Raw: 0xFF, HitPoints: 10, MaxHitPoints: 10, SideMorale: 50,
		PartyMorale: 70, FoeSide: true, Intelligence: 10, OwnSpeed: 6, FastestFoe: 6}
	for _, tc := range []struct {
		name  string
		check func(*MoraleCheck)
		want  MoraleOutcome
	}{
		{"turned always flees", func(c *MoraleCheck) { c.Turned = true; c.Raw = 0 }, MoraleForcedFlee},
		{"bit 7 clear skips the check", func(c *MoraleCheck) { c.Raw = 0x7F; c.SideMorale = 0 }, MoraleHolds},
		// FF：第一關士氣 0 一定過不了，第二關 50 >= 100 − 70。
		{"second gate holds", func(*MoraleCheck) {}, MoraleHolds},
		{"second gate needs the foe side", func(c *MoraleCheck) { c.FoeSide = false }, MoraleFlees},
		{"second gate fails, not slower: flees", func(c *MoraleCheck) { c.SideMorale = 25 }, MoraleFlees},
		{"slower and smart: surrenders", func(c *MoraleCheck) { c.SideMorale = 25; c.FastestFoe = 7 }, MoraleSurrenders},
		{"slower and dim: cornered", func(c *MoraleCheck) {
			c.SideMorale = 25
			c.FastestFoe = 7
			c.Intelligence = SurrenderIntelligence
		}, MoraleCornered},
		// B2（士氣 100）：第一關只要掉的不超過一百成就過。
		{"B2 holds when wounded", func(c *MoraleCheck) { c.Raw = 0xB2; c.HitPoints = 1; c.SideMorale = 0 }, MoraleHolds},
		{"B2 with the drop effect fails at 96 percent", func(c *MoraleCheck) {
			c.Raw = 0xB2
			c.HitPoints = 1
			c.MaxHitPoints = 25
			c.Drop = true
			c.SideMorale = 0
		}, MoraleFlees},
		// 第二關的士氣是 DS:6D22h 本身，0 一律不過。
		{"zero side morale never holds", func(c *MoraleCheck) { c.SideMorale = 0; c.PartyMorale = 100 }, MoraleFlees},
	} {
		check := base
		tc.check(&check)
		if got := ResolveMorale(check); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

// `08DCh..091Fh`：隊伍朝向 0／1／2／3（`DS:6A0Dh` 0／2／4／6）的基準方向。
func TestFleeBaseDirection(t *testing.T) {
	for facing, want := range [4][2]uint8{{7, 3}, {2, 6}, {3, 7}, {6, 2}} {
		if got := FleeBaseDirection(uint8(facing), false); got != want[0] {
			t.Errorf("facing %d foe side: %d, want %d", facing, got, want[0])
		}
		if got := FleeBaseDirection(uint8(facing), true); got != want[1] {
			t.Errorf("facing %d party side: %d, want %d", facing, got, want[1])
		}
	}
}

// overlay-13 entry 7：只有一樣快才擲 d2，擲 1 逃掉。
func TestEscapeSucceeds(t *testing.T) {
	asked := 0
	roll := func(value int) func(int, int) int {
		return func(count, sides int) int {
			asked++
			if count != 1 || sides != 2 {
				t.Fatalf("asked %dd%d, want 1d2", count, sides)
			}
			return value
		}
	}
	if !EscapeSucceeds(0, 1, 9, roll(2)) || !EscapeSucceeds(2, 6, 5, roll(2)) || EscapeSucceeds(2, 6, 7, roll(1)) {
		t.Fatal("no opponents or slower opponents escape; faster opponents block")
	}
	if asked != 0 {
		t.Fatalf("rolled %d dice outside the tie", asked)
	}
	if !EscapeSucceeds(1, 6, 6, roll(1)) || EscapeSucceeds(1, 6, 6, roll(2)) || asked != 2 {
		t.Fatalf("a tie rolls one d2 and escapes on 1 (asked %d)", asked)
	}
}

// 原版資料的正對照：八個怪物檔每一筆的 `+84h` 位元 7 都立著，而且只有兩種值——
// FF（士氣 0：第一關一定不過，全靠第二關的敵方整體生命成數）與 B2（士氣 100）。
func TestMonsterMoraleBytesInTheOriginalData(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	counts := map[uint8]int{}
	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			record, err := ReadDOSMonsterRecord(zipPath, archive, uint8(id))
			if err != nil {
				continue
			}
			counts[record.Raw[MoraleOffset]]++
		}
	}
	if len(counts) == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	if len(counts) != 2 || counts[0xFF] == 0 || counts[0xB2] == 0 {
		t.Fatalf("monster +84h values %v, want only FF and B2", counts)
	}
	t.Logf("+84h: FF × %d, B2 × %d", counts[0xFF], counts[0xB2])
}
