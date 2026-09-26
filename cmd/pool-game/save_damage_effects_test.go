package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 群組 12／6／4／5 在 #89 之後接上的碼（spec 112〈群組 12／6／4／5〉，issue #96／#99）。全部從 Update() 送鍵：
// C 開施法清單、Enter 施、N 換目標、Enter 確定；A 出手。

// spellMenuIndex 是施法者清單裡那一條法術的位置。
func spellMenuIndex(t *testing.T, application *app, party int, spell uint8) int {
	t.Helper()
	for index, option := range application.spellOptionsFor(application.state.Party[party]) {
		if option.ID == spell {
			return index
		}
	}
	t.Fatalf("spell %d is not in party member %d's list", spell, party)
	return -1
}

// seatAlly 把一位完整的角色放進盤面第 2 格（隊伍第 1 位），照開打時那一段帶著自己的效果串列、
// 體質與陣營（tactical.go 的 rememberPartySaveRecord）。
func seatAlly(state *tacticalState, application *app, member poolsave.Character) {
	member.MaxHP, member.CurrentHP = 30, 30
	application.state.Party[1] = member
	state.Effects[2] = combatEffects(member.Effects)
	state.rememberPartySaveRecord(2, member)
}

// #96：`5Ah`／`61h` 看體質加豁免（`24B9h..251Eh`、`2687h..26ECh`）。同一骰 12 的定身術（單一
// 目標 −2）對目標值 11：人類 10 定住，矮人加上體質那一段就過。
func TestDwarfOutsavesAHumanOnTheSameRoll(t *testing.T) {
	for _, race := range []string{"human", "dwarf"} {
		t.Run(race, func(t *testing.T) {
			_, cursor, ok := findRaceIndex(race)
			if !ok {
				t.Fatalf("race %q is not in the catalog", race)
			}
			member := createWithKeys(t, cursor, 'D')
			application, state := effectOnlyBoard(t, 12, gamepack.SpellIDHoldPerson)
			seatAlly(state, application, member)
			for category := range state.SaveTargets[2] {
				state.SaveTargets[2][category] = 11
			}
			castFromMenuAt(t, application, state, 1, 0, 2)
			held := state.hasEffect(2, gamepack.HoldPersonEffectCode)
			bonus, known := gamepack.ConstitutionSaveBonus(uint8(member.Abilities[gamepack.AbilityConstitution]))
			if race == "dwarf" {
				if !known || bonus == 0 {
					t.Fatalf("a created dwarf has constitution %d", member.Abilities[gamepack.AbilityConstitution])
				}
				if held {
					t.Fatalf("dwarf (CON %d, +%d) was held on 12 − 2: %q",
						member.Abilities[gamepack.AbilityConstitution], bonus, state.Status)
				}
				return
			}
			if !held {
				t.Fatalf("human saved on 12 − 2 against 11: %q", state.Status)
			}
		})
	}
}

// 防護邪惡／善良（`0377h`／`03AEh`）：`DS:5CF0h`（施法者）的陣營 `+0A0h` 落在 2／5／8 或 0／3／6
// 才 +2。施法者先把防護掛在隊友身上，再對他放定身術（12 − 2 對目標值 11）。
func TestProtectionRaisesTheSaveOnlyAgainstTheRightAlignment(t *testing.T) {
	for _, tc := range []struct {
		name      string
		spell     uint8
		alignment string
		wantHeld  bool
	}{
		{"none", 0, "chaotic-evil", true},
		{"evil caster vs PfE", 6, "chaotic-evil", false},
		{"good caster vs PfE", 6, "lawful-good", true},
		{"good caster vs PfG", 7, "neutral-good", false},
		{"neutral caster vs PfG", 7, "true-neutral", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spells := []uint8{gamepack.SpellIDHoldPerson}
			if tc.spell != 0 {
				spells = append(spells, tc.spell)
			}
			application, state := effectOnlyBoard(t, 12, spells...)
			application.state.Party[0].AlignmentID = tc.alignment
			state.rememberPartySaveRecord(1, application.state.Party[0])
			for category := range state.SaveTargets[2] {
				state.SaveTargets[2][category] = 11
			}
			if tc.spell != 0 {
				castFromMenuAt(t, application, state, 1, spellMenuIndex(t, application, 0, tc.spell), 2)
				if !state.hasEffect(2, application.spellParameters[tc.spell].EffectCode()) {
					t.Fatalf("protection did not attach: %q", state.Status)
				}
			}
			castFromMenuAt(t, application, state, 1,
				spellMenuIndex(t, application, 0, gamepack.SpellIDHoldPerson), 2)
			if held := state.hasEffect(2, gamepack.HoldPersonEffectCode); held != tc.wantHeld {
				t.Fatalf("held %v, want %v (%q)", held, tc.wantHeld, state.Status)
			}
		})
	}
}

// 抗火（`06F4h`）：燃燒之手推的傷害種類是 9（`1192h`），位元 0 立著，群組 6 把傷害減半。
// 施法者牧師、法師各 3 級：燃燒之手的傷害是等級 3。
func TestResistFireHalvesBurningHands(t *testing.T) {
	for _, resist := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 19, 24, gamepack.SpellIDBurningHands)
		if resist {
			castFromMenuAt(t, application, state, 1, spellMenuIndex(t, application, 0, 24), 2)
			if !state.hasEffect(2, gamepack.ResistFireEffectCode) {
				t.Fatalf("resist fire did not attach: %q", state.Status)
			}
		}
		before := state.HitPoints[2]
		castFromMenuAt(t, application, state, 1,
			spellMenuIndex(t, application, 0, gamepack.SpellIDBurningHands), 2)
		want := 3
		if resist {
			want = 1
		}
		if lost := before - state.HitPoints[2]; lost != want {
			t.Fatalf("resist %v: burning hands took %d, want %d (%q)", resist, lost, want, state.Status)
		}
	}
}

// 鏡影（`09CDh`）：Roll(1, 影像數 + 1) 大於 1、`DS:6779h` 非 0、`DS:677Eh` 為 0 才擋。魔法飛彈
// 被影像吃掉（影像 4 → 3）；近戰那一次 6779h 是 0，只多擲一次骰，照樣打進去。
func TestMirrorImageAbsorbsASpellButNotAMeleeHit(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, gamepack.SpellIDMirrorImage)
	castFromMenuAt(t, application, state, 1, 0, 1)
	images := func() int {
		at, ok := state.Effects[1].IndexOf(gamepack.MirrorImageEffectCode)
		if !ok {
			return 0
		}
		return int(state.Effects[1][at].Magnitude())
	}
	if images() != 4 {
		t.Fatalf("mirror image made %d images, want 4", images())
	}
	ally := &application.state.Party[1]
	ally.ClassLevels = make([]uint8, gamepack.ClassThac0ClassCount)
	ally.ClassLevels[gamepack.ClassSlotMagicUser] = 5
	ally.Memorised = make([]uint8, gamepack.MemorisedSpellSlots)
	ally.Memorised[0] = gamepack.SpellIDMagicMissile
	before := state.HitPoints[1]
	castFromMenuAt(t, application, state, 2, 0, 1)
	if state.HitPoints[1] != before || images() != 3 {
		t.Fatalf("magic missile: hp %d → %d, images %d (%q)", before, state.HitPoints[1], images(), state.Status)
	}
	attackWithKeys(t, application, state, 3, 1)
	if state.HitPoints[1] >= before || images() != 3 {
		t.Fatalf("melee: hp %d → %d, images %d (%q)", before, state.HitPoints[1], images(), state.Status)
	}
}

// 衰弱射線（`0A4Ah`）：牠自己的近戰傷害減掉四分之一（群組 4，overlay-13 `021Eh`）。
func TestEnfeeblementCutsTheFoesMeleeDamage(t *testing.T) {
	lost := map[bool]int{}
	for _, cast := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 19, 33)
		state.Damage[3].Count, state.Damage[3].Sides, state.Damage[3].Bonus = 1, 12, 0
		state.setSingleAttackForm(3, state.Damage[3])
		if cast {
			castFromMenuAt(t, application, state, 1, 0, 3)
			if !state.hasEffect(3, gamepack.EnfeeblementEffectCode) {
				t.Fatalf("enfeeblement did not attach: %q", state.Status)
			}
		}
		before := state.HitPoints[2]
		attackWithKeys(t, application, state, 3, 2)
		lost[cast] = before - state.HitPoints[2]
	}
	if lost[false] != 12 || lost[true] != 9 {
		t.Fatalf("1d12 on 12: plain %d, enfeebled %d, want 12 and 9", lost[false], lost[true])
	}
}

// 傷害型碰觸法術（電擊之握，參數表 `+2` 是 FFh）：`0997h..09DBh` 先擲 entry 6，沒中就沒有傷害。
func TestShockingGraspNeedsATouch(t *testing.T) {
	for _, armoured := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 19, gamepack.SpellIDShockingGrasp)
		if armoured {
			state.ArmorClass[3] = 70
		}
		before := state.HitPoints[3]
		castFromMenuAt(t, application, state, 1, 0, 3)
		if hurt := state.HitPoints[3] < before; hurt == armoured {
			t.Fatalf("armoured %v: shocking grasp hurt %v (%q)", armoured, hurt, state.Status)
		}
	}
}

// 靈魂鎚（`07F6h`）：施放後施法者多一把鎚子（`+2Eh` 14h、`+31h` F3h），再放一次不重給；
// 節點到期（`+4` 是 1）時收尾把它摘掉。
func TestSpiritualHammerComesAndGoesWithItsNode(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, gamepack.SpellIDSpiritHammer, gamepack.SpellIDSpiritHammer)
	hammers := func() int {
		count := 0
		for _, item := range application.state.Party[0].Inventory {
			if gamepack.IsSpiritualHammer(item.Raw) {
				count++
			}
		}
		return count
	}
	castFromMenuAt(t, application, state, 1, 0, 1)
	if hammers() != 1 {
		t.Fatalf("spiritual hammer gave %d hammers (%q)", hammers(), state.Status)
	}
	castFromMenuAt(t, application, state, 1, 0, 1)
	if hammers() != 1 {
		t.Fatalf("a second cast gave %d hammers", hammers())
	}
	cast := state.Round
	for state.hasEffect(1, gamepack.SpiritualHammerEffectCode) {
		if state.Round > cast+8 {
			t.Fatalf("hammer node still on after round %d", state.Round)
		}
		endRoundWithKeys(t, application, state)
	}
	if hammers() != 0 {
		t.Fatalf("the hammer outlived its node: %d left", hammers())
	}
}

// entry 7 的豁免骰是 byte、與目標值無號比較（`0DA5h` `A2 74 67`、`0DC8h` `3A 06 74 67 / 77 06`）。
// 降咒（−4）之後擲 2 的定身術（−2）算出 −4，繞成 0FCh，比目標值 20 大——豁免成功。
func TestNegativeSaveRollWrapsIntoASuccess(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, 44, gamepack.SpellIDHoldPerson)
	castFromMenuAt(t, application, state, 1, spellMenuIndex(t, application, 0, 44), 3)
	if !state.hasEffect(3, gamepack.BestowCurseEffectCode) {
		t.Fatalf("bestow curse did not attach: %q", state.Status)
	}
	application.roller = fixedRoller{2}
	castFromMenuAt(t, application, state, 1,
		spellMenuIndex(t, application, 0, gamepack.SpellIDHoldPerson), 3)
	if state.hasEffect(3, gamepack.HoldPersonEffectCode) {
		t.Fatalf("2 − 2 − 4 should wrap to 0FCh and save: %q", state.Status)
	}
}

// 致病術（`22h`，`0BE3h`）到期：叫 `2Bh`（力量 −1、重掛 3Ch）與 `2Ch`（生命 −1、重掛 0Ah）；
// 力量到 3 以下就改掛 `1Fh`。碰觸、豁免都照放（擲 19：碰得到、目標值 20 過不了）。
func TestCauseDiseaseKeepsWeakeningAfterItExpires(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, gamepack.SpellIDCauseDisease)
	castFromMenuAt(t, application, state, 1, 0, 2)
	shorten := func(code uint8) {
		t.Helper()
		at, ok := state.Effects[2].IndexOf(code)
		if !ok {
			t.Fatalf("ally carries no %02Xh: %+v (%q)", code, state.Effects[2], state.Status)
		}
		state.Effects[2][at].SetDuration(1)
	}
	shorten(gamepack.DiseaseEffectCode)
	hp := state.HitPoints[2]
	endRoundWithKeys(t, application, state)
	ally := application.state.Party[1]
	if state.hasEffect(2, gamepack.DiseaseEffectCode) || !state.hasEffect(2, gamepack.DiseaseWeakeningEffectCode) ||
		!state.hasEffect(2, gamepack.DiseaseWastingEffectCode) {
		t.Fatalf("22h did not hand over to 2Bh/2Ch: %+v", state.Effects[2])
	}
	if ally.Abilities[gamepack.AbilityStrength] != 11 || state.HitPoints[2] != hp-1 {
		t.Fatalf("after 22h: strength %d hp %d, want 11 and %d",
			ally.Abilities[gamepack.AbilityStrength], state.HitPoints[2], hp-1)
	}
	shorten(gamepack.DiseaseWastingEffectCode)
	endRoundWithKeys(t, application, state)
	if state.HitPoints[2] != hp-2 || !state.hasEffect(2, gamepack.DiseaseWastingEffectCode) {
		t.Fatalf("2Ch did not bite and renew: hp %d effects %+v", state.HitPoints[2], state.Effects[2])
	}
	application.state.Party[1].Abilities[gamepack.AbilityStrength] = 3
	shorten(gamepack.DiseaseWeakeningEffectCode)
	endRoundWithKeys(t, application, state)
	if got := application.state.Party[1].Abilities[gamepack.AbilityStrength]; got != 3 ||
		!state.hasEffect(2, gamepack.HelplessEffectCode) {
		t.Fatalf("strength 3: now %d, helpless %v", got, state.hasEffect(2, gamepack.HelplessEffectCode))
	}
}
