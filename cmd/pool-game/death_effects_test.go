package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 倒下時的群組 13、再生、群組 5 其餘的碼、麻痺、編號 58 的先後與瞄準屍體（spec 155，issue #113）。
// 全部從 Update() 送鍵：A 出手、ENTER 結束回合、C 施法、M 進 Manual 瞄準。

// deathBoard 是 effectOnlyBoard 多一隻站在遠處的敵人 4：3 號倒下之後戰鬥不會因為敵方清光而
// 停在「要不要繼續」那一問，ENTER 推得動回合。倒下的人靠 Footprint 放回原地（開打時抄的體型）。
func deathBoard(t *testing.T, roll int, spells ...uint8) (*app, *tacticalState) {
	t.Helper()
	caster := spellCasterWith(int(gamepack.ClassSlotCleric), 3, spells...)
	caster.ClassLevels[gamepack.ClassSlotMagicUser] = 3
	caster.Abilities = [6]int{12, 12, 12, 12, 12, 12}
	application, state := newSpellBoard(t, caster,
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 5, Y: 6, FootprintClass: 1},
		combat.CombatantCell{X: 20, Y: 12, FootprintClass: 1})
	state.Friendly[2], state.PartySlot[2] = true, 1
	application.state.Party = append(application.state.Party,
		poolsave.Character{Name: "ALLY", RaceID: "human", Age: 20, MaxHP: 30, CurrentHP: 30,
			Abilities: [6]int{12, 12, 12, 12, 12, 12}})
	for index := range state.AIDriven {
		state.AIDriven[index] = false
	}
	state.Footprint = []uint8{0, 1, 1, 1, 1}
	state.PartyAged = application.agePartyMember(state)
	state.PartyEffectTeardown = func(index int, node gamepack.EffectNode) {
		application.partyEffectTeardown(state, index, node)
	}
	application.roller = fixedRoller{roll}
	return application, state
}

func monsterNode(code uint8) gamepack.EffectNode {
	return gamepack.NewEffectNode(code, 0, gamepack.EffectUndispellable, false)
}

// `63h`（WILD BOAR，`2727h`）：倒下那一下打穿 0..5 點 → entry 22 以 6 − 打穿點數站起來、掛 `5Fh`
// （Roll(1, 4) ＋ 1 回合），`63h` 摘掉；`5Fh` 到期時還站著 → "Falls dead"、狀態 6。打穿 7 點不站起來。
func TestWildBoarFightsOnAndThenFallsDead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hp    int
		sides uint8
		rally int
	}{{"three through", 3, 6, 3}, {"seven through", 1, 8, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			application, state := deathBoard(t, 19)
			state.Effects[3] = gamepack.EffectList{monsterNode(gamepack.BoarRallyEffectCode)}
			state.HitPoints[3] = tc.hp
			state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: tc.sides})
			attackWithKeys(t, application, state, 2, 3)
			if tc.rally == 0 {
				if state.States[3] != combat.DyingState || state.Roster[3].FootprintClass != 0 {
					t.Fatalf("seven through: state %d footprint %d", state.States[3], state.Roster[3].FootprintClass)
				}
				return
			}
			at, falls := state.Effects[3].IndexOf(gamepack.BoarFallsEffectCode)
			if state.States[3] != 0 || state.HitPoints[3] != tc.rally || state.Roster[3].FootprintClass == 0 ||
				!falls || state.hasEffect(3, gamepack.BoarRallyEffectCode) {
				t.Fatalf("boar after going down: state %d hp %d footprint %d effects %+v (%q)", state.States[3],
					state.HitPoints[3], state.Roster[3].FootprintClass, state.Effects[3], state.Status)
			}
			rounds := int(state.Effects[3][at].Duration())
			if rounds != 5 {
				t.Fatalf("5Fh lasts %d rounds, want Roll(1, 4) + 1 = 5", rounds)
			}
			for round := 0; round < rounds; round++ {
				endRoundWithKeys(t, application, state)
			}
			if state.States[3] != combat.DeadState || state.Roster[3].FootprintClass != 0 {
				t.Fatalf("5Fh ran out: state %d footprint %d (%q)", state.States[3],
					state.Roster[3].FootprintClass, state.Status)
			}
		})
	}
}

// `64h`（TROLL，`27D0h`）：倒下時 `6777h` 沒有火（位元 0）→ 掛 `66h`（Roll(3, 6) 回合），到期以 `+32h`
// 站起來（`285Fh`）。攻擊者帶 `06h`（`0227h` 寫 `6777h = 9`）打倒的就不再起來。
func TestTrollGetsBackUpUnlessBurnt(t *testing.T) {
	for _, burnt := range []bool{false, true} {
		application, state := deathBoard(t, 19)
		state.Effects[3] = gamepack.EffectList{monsterNode(gamepack.TrollRevivalEffectCode)}
		state.HitPoints[3] = 2
		state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: 6})
		if burnt {
			state.Effects[2] = gamepack.EffectList{monsterNode(gamepack.CreatureBaneEffectCode)}
		}
		attackWithKeys(t, application, state, 2, 3)
		if state.Roster[3].FootprintClass != 0 {
			t.Fatalf("burnt %v: the troll did not go down (%q)", burnt, state.Status)
		}
		if got := state.hasEffect(3, gamepack.TrollRisesEffectCode); got == burnt {
			t.Fatalf("burnt %v: 66h attached %v", burnt, got)
		}
		for guard := 0; guard < 8 && state.Roster[3].FootprintClass == 0; guard++ {
			endRoundWithKeys(t, application, state)
		}
		stood := state.Roster[3].FootprintClass != 0
		if stood == burnt || (stood && (state.States[3] != 0 || state.HitPoints[3] != 30)) {
			t.Fatalf("burnt %v: stood %v state %d hp %d (%q)", burnt, stood, state.States[3],
				state.HitPoints[3], state.Status)
		}
	}
}

// `65h`（TROLL，群組 5）受傷就掛 `3Bh`（3 回合、有收尾）；`3Bh` 到期掛 `62h`（`147Ch`）；`62h` 在群組 19
// 每回合 +3、不超過 `+32h`（`26F5h`），在 entry 4 減計時之前。
func TestTrollRegeneratesThreeRoundsAfterBeingHurt(t *testing.T) {
	application, state := deathBoard(t, 19)
	state.Effects[3] = gamepack.EffectList{monsterNode(gamepack.TrollWoundEffectCode)}
	state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: 6})
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 24 || !state.hasEffect(3, gamepack.RegenerationPendingEffectCode) {
		t.Fatalf("after the hit: hp %d effects %+v", state.HitPoints[3], state.Effects[3])
	}
	for round := 0; round < 3; round++ {
		endRoundWithKeys(t, application, state)
	}
	if !state.hasEffect(3, gamepack.RegenerationEffectCode) || state.HitPoints[3] != 24 {
		t.Fatalf("3Bh ran out: hp %d effects %+v", state.HitPoints[3], state.Effects[3])
	}
	endRoundWithKeys(t, application, state)
	endRoundWithKeys(t, application, state)
	endRoundWithKeys(t, application, state)
	if state.HitPoints[3] != 30 {
		t.Fatalf("62h: hp %d after three rounds, want 24 + 3 + 3 capped at 30", state.HitPoints[3])
	}
}

// deathBoardWeapon 讓 2 號拿著一件 itemType、`+32h` 是 plus、`+31h` 是 part 的武器（穿著）。
func deathBoardWeapon(t *testing.T, application *app, itemType uint8, plus int8, part uint8) {
	t.Helper()
	types, err := gamepack.ReadDOSItemTypeTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.itemTypes = types
	raw := make([]byte, gamepack.MonsterItemRecordSize)
	raw[gamepack.ItemTypeOffset], raw[gamepack.ItemPlusOffset] = itemType, uint8(plus)
	raw[0x31], raw[gamepack.ItemReadiedOffset] = part, 1
	application.state.Party[1].Inventory = []poolsave.Item{{Name: "WEAPON", Raw: raw}}
}

// 群組 5（overlay-13 `022Ch`，目標身上）看 `DS:5CF0h` 打中的那一件。長劍 24h 的型別表 `+7` 是 0、
// 0Ch 是 80h；B1h 是字詞表的 Silver。1d8 擲 8。
func TestUndeadWeaponRulesReadTheHittingItem(t *testing.T) {
	const longSword, blunt, silver = 0x24, 0x0c, 0xb1
	for _, tc := range []struct {
		name     string
		code     uint8
		itemType uint8
		plus     int8
		part     uint8
		lost     int
	}{
		{"skeleton vs sword", gamepack.EdgedHalfEffectCode, longSword, 0, 0, 4},
		{"skeleton vs blunt", gamepack.EdgedHalfEffectCode, blunt, 0, 0, 8},
		{"juju vs sword", gamepack.BluntAndPiercingHalfEffectCode, longSword, 0, 0, 8},
		{"juju vs blunt", gamepack.BluntAndPiercingHalfEffectCode, blunt, 0, 0, 4},
		{"spectre vs plain", gamepack.MagicWeaponOnlyEffectCode, longSword, 0, 0, 0},
		{"spectre vs +1", gamepack.MagicWeaponOnlyEffectCode, longSword, 1, 0, 8},
		{"mummy vs +1", gamepack.MagicWeaponHalfEffectCode, longSword, 1, 0, 4},
		{"mummy vs plain", gamepack.MagicWeaponHalfEffectCode, longSword, 0, 0, 8},
		{"wight vs plain", gamepack.SilverOrMagicEffectCode, longSword, 0, 0, 0},
		{"wight vs silver", gamepack.SilverOrMagicEffectCode, longSword, 0, silver, 8},
		{"wraith vs silver", gamepack.SilverHalfEffectCode, longSword, 0, silver, 4},
		{"wraith vs plain", gamepack.SilverHalfEffectCode, longSword, 0, 0, 0},
		{"wraith vs +1", gamepack.SilverHalfEffectCode, longSword, 1, 0, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			application, state := deathBoard(t, 19)
			deathBoardWeapon(t, application, tc.itemType, tc.plus, tc.part)
			state.Effects[3] = gamepack.EffectList{monsterNode(tc.code)}
			state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: 8})
			attackWithKeys(t, application, state, 2, 3)
			if lost := 30 - state.HitPoints[3]; lost != tc.lost {
				t.Fatalf("lost %d, want %d (%q)", lost, tc.lost, state.Status)
			}
		})
	}
}

// 麻痺（`15F7h`）：GHOUL 的 `44h` 在群組 2（第一形態）也在群組 3；目標是精靈（`+2Eh` 2）整支不做
// （`16DCh`）。THRI-KREEN 的 `43h` 只在群組 3，持續 Roll(2, 8)。豁免是類別 0、修正 0，擲 19 過不了 20。
func TestMonsterParalysisHoldsTheTargetExceptElvesAgainstGhouls(t *testing.T) {
	for _, tc := range []struct {
		name     string
		code     uint8
		form     int
		race     string
		duration uint16
	}{
		{"ghoul first form", 0x44, 1, "human", 0x3f},
		{"ghoul second form", 0x44, 2, "human", 0x3f},
		{"ghoul vs elf", 0x44, 1, "elf", 0},
		{"thri-kreen second form", 0x43, 2, "human", 8},
		{"thri-kreen first form", 0x43, 1, "human", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			application, state := deathBoard(t, 19)
			stingingFoe(state, tc.code, tc.form)
			application.state.Party[1].RaceID = tc.race
			state.SaveTargets[2][0] = 20
			attackWithKeys(t, application, state, 3, 2)
			at, held := state.Effects[2].IndexOf(gamepack.HoldPersonEffectCode)
			if held != (tc.duration != 0) {
				t.Fatalf("held %v, want %v (%q)", held, tc.duration != 0, state.Status)
			}
			if !held {
				return
			}
			node := state.Effects[2][at]
			if node.Duration() != tc.duration || node.CasterLevel() != 0x0c || node.NeedsTeardown() ||
				!strings.Contains(application.statusLine, "Paralyzed") {
				t.Fatalf("34h node %+v (%q)", node, application.statusLine)
			}
		})
	}
}

// 編號 58（`2E02h`）沒中毒時：解病鏈 `225Bh` 解到東西就結束、不治療；什麼都沒解到才治療 Roll(1, 4) ＋ 8
// （entry 21，封頂 `+32h`）。營地（C 施法）與戰場（C 施法、瞄準）兩邊同一個先後。
func TestGreaterHealCuresDiseaseInsteadOfHealing(t *testing.T) {
	for _, diseased := range []bool{false, true} {
		sick := campCaster("B", 1)
		sick.CurrentHP = 5
		if diseased {
			sick.Effects = storedEffects(gamepack.EffectList{
				gamepack.NewEffectNode(gamepack.DiseaseEffectCode, 0, gamepack.EffectUndispellable, false)})
		}
		application := campCastApp(t, campCaster("A", 9, gamepack.SpellIDGreaterHeal), sick)
		castInCamp(t, application, 0, gamepack.SpellIDGreaterHeal, 1)
		b := application.state.Party[1]
		_, still := memberEffect(b, gamepack.DiseaseEffectCode)
		if diseased && (still || b.CurrentHP != 5) || !diseased && b.CurrentHP == 5 {
			t.Errorf("camp, diseased %v: disease left %v hp %d", diseased, still, b.CurrentHP)
		}

		boardApp, state := deathBoard(t, 4, gamepack.SpellIDGreaterHeal)
		state.HitPoints[2] = 5
		if diseased {
			state.Effects[2] = gamepack.EffectList{
				gamepack.NewEffectNode(gamepack.DiseaseEffectCode, 0, gamepack.EffectUndispellable, false)}
		}
		castFromMenuAt(t, boardApp, state, 1, 0, 2)
		if diseased && (state.hasEffect(2, gamepack.DiseaseEffectCode) || state.HitPoints[2] != 5) ||
			!diseased && state.HitPoints[2] != 5+4+8 {
			t.Errorf("board, diseased %v: effects %+v hp %d (%q)", diseased, state.Effects[2],
				state.HitPoints[2], state.Status)
		}
	}
}

// manualAimAt 在瞄準中按 M 進 Manual，一格一格走到 (x, y)。
func manualAimAt(t *testing.T, application *app, x, y int) {
	t.Helper()
	if !application.castManual {
		pressAll(t, application, ebiten.KeyM)
	}
	for guard := 0; guard < 64; guard++ {
		dx, dy := x-application.castManualX, y-application.castManualY
		if dx == 0 && dy == 0 {
			return
		}
		sign := func(v int) int {
			switch {
			case v > 0:
				return 1
			case v < 0:
				return -1
			}
			return 0
		}
		moved := false
		for direction, key := range tacticalStepKeys {
			step, err := combat.DirectionStep(uint8(direction))
			if err != nil {
				t.Fatal(err)
			}
			if int(int8(step.X)) == sign(dx) && int(int8(step.Y)) == sign(dy) {
				pressAll(t, application, key)
				moved = true
				break
			}
		}
		if !moved {
			t.Fatalf("no key steps toward %d,%d", dx, dy)
		}
	}
	t.Fatalf("manual cursor stuck at %d,%d", application.castManualX, application.castManualY)
}

// Manual 瞄準（`2DD3h`）空格上沒有人時查屍體表（`2F9Fh`）；施法那一條推的第四個引數是 0，`306Dh` 不擋
// 屍體——中毒倒下的隊員在戰場上施得到緩毒術，站起來（`18BBh`）。
func TestSlowPoisonReachesAPoisonedCorpseByManualAim(t *testing.T) {
	application, state := deathBoard(t, 19, gamepack.SpellIDSlowPoison)
	state.Effects[2] = gamepack.EffectList{gamepack.NewPoisonNode()}
	application.poisonKill(state, 2)
	if state.Roster[2].FootprintClass != 0 || state.States[2] != combat.DeadState {
		t.Fatalf("the ally did not die of poison: state %d", state.States[2])
	}
	state.Mover, state.Scores[1] = 1, 6
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	for guard := 0; guard < 8 && state.Casting.Pending[1] != 0 && !application.castTargeting; guard++ {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if !application.castTargeting {
		t.Fatalf("slow poison did not open aiming: %q", state.Status)
	}
	manualAimAt(t, application, 6, 5)
	pressAll(t, application, ebiten.KeyEnter)
	if application.castTargeting {
		t.Fatalf("the corpse was not accepted as a target: %q", state.Status)
	}
	if state.States[2] != 0 || state.Roster[2].FootprintClass == 0 || state.HitPoints[2] != 1 ||
		!state.hasEffect(2, gamepack.SlowPoisonEffectCode) {
		t.Fatalf("after slow poison: state %d footprint %d hp %d effects %+v (%q)", state.States[2],
			state.Roster[2].FootprintClass, state.HitPoints[2], state.Effects[2], state.Status)
	}
}
