package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 一次行動要揮幾下、每一下用哪一組骰子（spec 051）。
//
// 巨魔是樣本：攻擊次數編碼 `04 02` 加上 `1d4+4`／`2d6`，也就是 AD&D 一版的
// 爪／爪／咬。只讀一種形態的話牠一回合只揮一次爪——那在畫面上看起來像
// 「巨魔很弱」，不像少讀了一個欄位。
func TestATrollSwingsTwiceWithClawsAndOnceWithItsBite(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	var troll gamepack.MonsterRecord
	found := false
	for archive := uint8(1); archive <= 8 && !found; archive++ {
		for id := 0; id < 256; id++ {
			record, err := gamepack.ReadDOSMonsterRecord(zipPath, archive, uint8(id))
			if err == nil && record.Name == "TROLL" {
				troll, found = record, true
				break
			}
		}
	}
	if !found {
		t.Skip("原版資料裡找不到 TROLL")
	}
	state := &tacticalState{
		AttackForms: make([][gamepack.MonsterAttackSlots]combat.DamageDice, 2),
		AttackRates: make([][gamepack.MonsterAttackSlots]uint8, 2),
	}
	if err := applyMonsterAttackForms(state, 1, troll); err != nil {
		t.Fatal(err)
	}
	swings, err := application.attackSwingsThisPhase(state, 1)
	if err != nil {
		t.Fatal(err)
	}
	// 咬（第二形態）先，接著兩下爪——順序照原版的攻擊區段由第二形態倒數。
	want := []combat.DamageDice{
		{Count: 2, Sides: 6},
		{Count: 1, Sides: 4, Bonus: 4},
		{Count: 1, Sides: 4, Bonus: 4},
	}
	if len(swings) != len(want) {
		t.Fatalf("巨魔這一相位揮 %d 下，要的是 %d 下：%+v", len(swings), len(want), swings)
	}
	for index := range want {
		if swings[index] != want[index] {
			t.Errorf("第 %d 下是 %+v，要的是 %+v", index, swings[index], want[index])
		}
	}
}

// MONnSPC 的 55h 必須從真正的命中路徑扣到角色，而不是只測一支孤立 helper。
func TestAHitFromAnEnergyDrainingMonsterLowersTheTargetLevel(t *testing.T) {
	var thresholds gamepack.ExperienceTable
	thresholds[2][2], thresholds[2][3] = 2000, 4000
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[2] = 3
	state := &tacticalState{
		Roster:   []combat.CombatantCell{{}, {FootprintClass: 1}, {FootprintClass: 1}},
		Friendly: []bool{false, true, false}, PartySlot: []int{-1, 0, -1},
		HitPoints: []int{0, 18, 8}, MaxHitPoints: []int{0, 18, 8},
		THAC0: []uint8{0, 0, 60}, ArmorClass: make([]int, 3),
		States: make([]uint8, 3), Scores: []uint8{0, 1, 1},
		AttackForms: make([][gamepack.MonsterAttackSlots]combat.DamageDice, 3),
		AttackRates: make([][gamepack.MonsterAttackSlots]uint8, 3),
		Effects:     make([]gamepack.EffectList, 3), Mover: 2,
	}
	state.AttackForms[2][0] = combat.DamageDice{Count: 1, Sides: 1}
	state.AttackRates[2][0] = 2
	state.Effects[2] = gamepack.EffectList{{Code: gamepack.EnergyDrainOneEffectCode}}
	member := poolsave.Character{Name: "HERO", ClassID: "fighter",
		ClassLevels: levels, Experience: 5000, MaxHP: 18, CurrentHP: 18, RawHP: 18}
	application := &app{roller: fixedRoller{20}, experienceTable: thresholds,
		state: poolsave.State{Party: []poolsave.Character{member}}}

	if err := application.resolveTacticalAttack(state, 1); err != nil {
		t.Fatal(err)
	}
	got := application.state.Party[0]
	if got.ClassLevels[2] != 2 || got.DrainedLevels != 1 || got.DrainedHitPoints != 6 {
		t.Fatalf("命中後角色沒有被吸取：%+v", got)
	}
	if state.MaxHitPoints[1] != 12 || state.HitPoints[1] != 11 {
		t.Fatalf("戰場 HP 是 %d/%d，要的是 11/12", state.HitPoints[1], state.MaxHitPoints[1])
	}
}

// 編碼 3 的「每兩回合三次」要靠相位交替，不是每回合都一樣。
func TestTheAttackPhaseAlternatesAThreeHalvesRate(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	state := &tacticalState{
		AttackForms: make([][gamepack.MonsterAttackSlots]combat.DamageDice, 2),
		AttackRates: make([][gamepack.MonsterAttackSlots]uint8, 2),
	}
	state.AttackForms[1][0] = combat.DamageDice{Count: 1, Sides: 8}
	state.AttackRates[1][0] = 3
	for phase, want := range map[uint8]int{0: 1, 1: 2} {
		state.AttackPhase = phase
		swings, err := application.attackSwingsThisPhase(state, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(swings) != want {
			t.Errorf("相位 %d 揮 %d 下，要的是 %d 下", phase, len(swings), want)
		}
	}
}

// 相位在第一回合是 0，之後每個回合邊界加一。
func TestTheAttackPhaseAdvancesOncePerRound(t *testing.T) {
	state := &tacticalState{}
	roll := func(count, sides int) int { return 1 }
	state.startRound(roll)
	if state.Round != 1 || state.AttackPhase != 0 {
		t.Fatalf("第一回合 Round=%d 相位=%d，要的是 1 與 0", state.Round, state.AttackPhase)
	}
	state.startRound(roll)
	if state.Round != 2 || state.AttackPhase != 1 {
		t.Fatalf("第二回合 Round=%d 相位=%d，要的是 2 與 1", state.Round, state.AttackPhase)
	}
	state.startRound(roll)
	if state.AttackPhase != 2 {
		t.Fatalf("第三回合相位=%d，要的是 2", state.AttackPhase)
	}
}
