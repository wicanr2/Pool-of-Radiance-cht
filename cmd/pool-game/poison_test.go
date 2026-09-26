package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 中毒（spec 153，issue #106）。戰場上從 Update() 送鍵（A 出手、ENTER 結束回合），營地從 C 施法。

// stingingFoe 讓對面那一格（3）只剩第二攻擊形態 1d1、身上帶著毒碼 code。
func stingingFoe(state *tacticalState, code uint8, form int) {
	state.AttackForms[3] = [gamepack.MonsterAttackSlots]combat.DamageDice{}
	state.AttackRates[3] = [gamepack.MonsterAttackSlots]uint8{}
	state.AttackForms[3][form-1] = combat.DamageDice{Count: 1, Sides: 1}
	state.AttackRates[3][form-1] = 2
	state.Effects[3] = gamepack.EffectList{gamepack.NewEffectNode(code, 0, 0xff, false)}
}

// `1553h`：第二形態咬中、類別 0 的豁免失敗 → 掛 `37h`、狀態 6。擲 19：命中（要 10），豁免目標值 20
// 過不了；把目標值降到 19 就過（對照組），第一形態帶同一個碼也不毒（群組 2 沒有 40h）。
func TestAPoisonousStingKillsOnAFailedSave(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    int
		target  uint8
		poisons bool
	}{
		{"fails", 2, 20, true},
		{"saves", 2, 19, false},
		{"first form", 1, 20, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			application, state := effectOnlyBoard(t, 19)
			stingingFoe(state, 0x40, tc.form)
			state.SaveTargets[2][0] = tc.target
			attackWithKeys(t, application, state, 3, 2)
			ally := application.state.Party[1]
			poisoned := state.hasEffect(2, gamepack.PoisonEffectCode)
			if poisoned != tc.poisons || (state.States[2] == combat.DeadState) != tc.poisons ||
				(ally.Status == combat.DeadState) != tc.poisons {
				t.Fatalf("poisoned %v state %d status %d hp %d (%q)", poisoned, state.States[2],
					ally.Status, state.HitPoints[2], state.Status)
			}
			if !tc.poisons {
				return
			}
			if state.HitPoints[2] != 0 || state.Roster[2].FootprintClass != 0 ||
				!strings.Contains(state.Status, "Poisoned") {
				t.Fatalf("poison death: hp %d footprint %d status %q", state.HitPoints[2],
					state.Roster[2].FootprintClass, state.Status)
			}
			node := state.Effects[2][len(state.Effects[2])-1]
			if node.Code != gamepack.PoisonEffectCode || node.Duration() != 0 || !node.Undispellable() ||
				node.NeedsTeardown() {
				t.Fatalf("37h node %+v", node)
			}
		})
	}
}

// 修正照碼：41h 是 +4（`167Dh` `B0 04 50`）、46h 是 −2（`1721h` `B0 FE 50`，`cbw`）。擲 19、
// 目標值 22：41h 的 23 過得了；46h 的 17 與 40h 的 19 過不了。
func TestPoisonCodesCarryTheirOwnSaveModifiers(t *testing.T) {
	for code, dies := range map[uint8]bool{0x40: true, 0x41: false, 0x46: true, 0x42: true} {
		application, state := effectOnlyBoard(t, 19)
		stingingFoe(state, code, 2)
		state.SaveTargets[2][0] = 22
		attackWithKeys(t, application, state, 3, 2)
		if got := state.States[2] == combat.DeadState; got != dies {
			t.Errorf("%02Xh: died %v, want %v (%q)", code, got, dies, state.Status)
		}
	}
}

// 群組 12 的 `7Dh`（不死生物）：類別 0 的豁免骰寫成 100，一定過（`2E3Eh`）。同一骰、同一碼。
func TestUndeadShrugOffPoison(t *testing.T) {
	for _, undead := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 19)
		stingingFoe(state, 0x40, 2)
		if undead {
			state.Effects[2] = state.Effects[2].Append(
				gamepack.NewEffectNode(gamepack.UndeadImmunityEffectCode, 0, 0xff, false))
		}
		attackWithKeys(t, application, state, 3, 2)
		if poisoned := state.hasEffect(2, gamepack.PoisonEffectCode); poisoned == undead {
			t.Fatalf("undead %v: poisoned %v", undead, poisoned)
		}
	}
}

// poisonedMember 是一個中毒倒下的隊員：狀態 6、生命 0、身上一個 `37h`。
func poisonedMember(name string) poolsave.Character {
	member := campCaster(name, 1)
	member.Status, member.CurrentHP = combat.DeadState, 0
	member.Effects = storedEffects(gamepack.EffectList{gamepack.NewPoisonNode()})
	return member
}

// 緩毒術在營地（`1846h` → `08BCh` → `18BBh`／`18D2h`）：中毒倒下的人墊到 1 點、站起來，掛 `16h`
// （有收尾）與 `0Fh`（0Ah、有收尾）。`0Fh` 每十分鐘扣 1 點、自己重掛；`16h` 到期還中毒就毒發。
func TestSlowPoisonHoldsThePoisonOnlyUntilItRunsOut(t *testing.T) {
	application := campCastApp(t, campCaster("A", 3, gamepack.SpellIDSlowPoison), poisonedMember("B"))
	message := castInCamp(t, application, 0, gamepack.SpellIDSlowPoison, 1)
	b := application.state.Party[1]
	slow, ok := memberEffect(b, gamepack.SlowPoisonEffectCode)
	drain, drains := memberEffect(b, gamepack.PoisonDrainEffectCode)
	if b.Status != 0 || b.CurrentHP != 1 || !ok || !slow.NeedsTeardown() || !drains ||
		drain.Duration() != 10 || !drain.NeedsTeardown() || !strings.Contains(message, "gets back up") {
		t.Fatalf("after slow poison: status %d hp %d 16h %v %+v 0Fh %v %+v (%q)", b.Status, b.CurrentHP,
			ok, slow, drains, drain, message)
	}
	application.state.Party[1].CurrentHP = 8
	application.advancePartyEffects(10)
	b = application.state.Party[1]
	if _, again := memberEffect(b, gamepack.PoisonDrainEffectCode); b.CurrentHP != 7 || !again {
		t.Fatalf("0Fh after ten minutes: hp %d, renewed %v", b.CurrentHP, again)
	}
	application.advancePartyEffects(int(slow.Duration()))
	b = application.state.Party[1]
	if b.Status != combat.DeadState || b.CurrentHP != 0 ||
		!strings.Contains(application.statusLine, "dies from poison") {
		t.Fatalf("16h ran out while poisoned: status %d hp %d (%q)", b.Status, b.CurrentHP,
			application.statusLine)
	}
	if _, left := memberEffect(b, gamepack.PoisonDrainEffectCode); left {
		t.Fatal("078Bh should take 0Fh off with the slowed poison")
	}
}

// 神殿的 Neutralize Poison（overlay-04 `0736h`）摘掉 `37h`／`16h`／`0Fh`：緩毒之後解毒，時間過了
// 也不會再毒發，`0Fh` 不再扣血。
func TestNeutralizedPoisonDoesNotComeBack(t *testing.T) {
	application := campCastApp(t, campCaster("A", 3, gamepack.SpellIDSlowPoison), poisonedMember("B"))
	castInCamp(t, application, 0, gamepack.SpellIDSlowPoison, 1)
	application.state.Party[1].CurrentHP = 8
	application.state.Party[1].Money[3] = 3000
	application.state.CharacterLibrary = []poolsave.Character{application.state.Party[1]}
	application.templeActive = true
	application.saveState = func(poolsave.State) error { return nil }
	application.roller = fixedTempleRoller(1)
	application.templeParty = 1
	application.enterTempleHeal()
	application.cellMenuCursor = templeServiceIndex(t, "neutralize-poison")
	for step := 0; step < 2; step++ { // 挑那一項、確認 YES
		if err := application.selectSuneTempleOption(); err != nil {
			t.Fatal(err)
		}
	}
	if combatEffects(application.state.Party[1].Effects).Has(gamepack.PoisonEffectCode) {
		t.Fatalf("the temple did not neutralize: %q %q", application.eventText, application.statusLine)
	}
	application.advancePartyEffects(10000)
	b := application.state.Party[1]
	if b.Status != 0 || b.CurrentHP != 8 || len(b.Effects) != 0 {
		t.Fatalf("after neutralize and time: status %d hp %d effects %+v", b.Status, b.CurrentHP, b.Effects)
	}
}

// templeServiceIndex 是那一項在神殿選單上的位置。
func templeServiceIndex(t *testing.T, id string) int {
	t.Helper()
	for index, service := range templeHealServiceIDs {
		if service == id {
			return index
		}
	}
	t.Fatalf("temple has no %q", id)
	return -1
}

// 戰場上的緩毒：`4Eh` 叫 entry 22 把人扶起來要原地站得下（overlay-32 entry 21）；站不下就掛一個
// 持續 1 的 `4Eh`，下一回合再試。緩毒施在一個中毒但被扶起的人身上（鎖 HP 會這樣），站得起來。
func TestPoisonRecoveryWaitsForItsSpotInCombat(t *testing.T) {
	application, state := effectOnlyBoard(t, 19)
	// 開打時每一格的體型（tactical.go 抄 FootprintClass），倒下之後靠它放回去。
	state.Footprint = []uint8{0, 1, 1, 1}
	state.Effects[2] = gamepack.EffectList{gamepack.NewPoisonNode()}
	application.poisonKill(state, 2)
	// 緩毒術先把生命墊到 1（`1892h`）；第三格站到那具屍體的位置上。
	state.HitPoints[2] = 1
	state.Roster[3].X, state.Roster[3].Y = state.Roster[2].X, state.Roster[2].Y
	application.slowPoisonAftermath(state, 2)
	if state.States[2] != combat.DeadState || !state.hasEffect(2, gamepack.PoisonRecoveryEffectCode) {
		t.Fatalf("blocked spot: state %d effects %+v", state.States[2], state.Effects[2])
	}
	state.Roster[3].X, state.Roster[3].Y = 9, 9
	endRoundWithKeys(t, application, state)
	if state.States[2] != 0 || state.Roster[2].FootprintClass == 0 ||
		application.state.Party[1].Status != 0 {
		t.Fatalf("free spot: state %d footprint %d status %d (%q)", state.States[2],
			state.Roster[2].FootprintClass, application.state.Party[1].Status, state.Status)
	}
}

// 群組 12 的 `3Dh`（抗火戒指）：火焰傷害的豁免 +4（`14DCh`）。燃燒之手沒有豁免，拿火球：
// 擲 12，目標值 15——戴戒指的 16 過得了、沒戴的 12 過不了。
func TestFireResistanceRingRaisesTheSaveAgainstFire(t *testing.T) {
	for _, ring := range []bool{false, true} {
		application, state := newSpellBoard(t,
			spellCasterWith(int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDFireball),
			combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
			combat.CombatantCell{X: 12, Y: 5, FootprintClass: 1})
		state.AIDriven[2] = false
		for category := range state.SaveTargets[2] {
			state.SaveTargets[2][category] = 15
		}
		if ring {
			state.Effects[2] = gamepack.EffectList{
				gamepack.NewEffectNode(gamepack.FireResistanceEffectCode, 0, 0x0c, false)}
		}
		state.SpellDamage = spellDamageContext{Spell: gamepack.SpellIDFireball,
			Flags: gamepack.SpellDamageKind(gamepack.SpellIDFireball)}
		application.roller = fixedRoller{12}
		saved := application.savedAgainstCategory(state, 2, 4, 0)
		if saved != ring {
			t.Fatalf("ring %v: saved %v", ring, saved)
		}
	}
}

// 群組 6 其餘的碼（`2A17h`、`2A75h`、`2B76h`、`2581h`、`2535h`、`149Ch`）。表驅動、直接走群組 6。
func TestSpellDamageGroupReadsTheDamageKind(t *testing.T) {
	roll := func(count, sides int) int { return 1 }
	for _, tc := range []struct {
		name  string
		code  uint8
		spell uint8
		flags uint8
		dice  uint8
		in    int
		want  int
	}{
		{"fire giant vs fireball", gamepack.FireImmunityEffectCode, gamepack.SpellIDFireball, 9, 5, 20, 0},
		{"fire giant vs magic missile", gamepack.FireImmunityEffectCode, gamepack.SpellIDMagicMissile, 8, 2, 7, 7},
		{"vampire vs lightning", gamepack.ElectricHalfEffectCode, gamepack.SpellIDLightningBolt, 0x0c, 0, 21, 10},
		{"juju vs fireball halves", gamepack.FireHalfEffectCode, gamepack.SpellIDFireball, 9, 5, 21, 10},
		{"juju vs magic missile", gamepack.JujuImmunityEffectCode, gamepack.SpellIDMagicMissile, 8, 2, 7, 0},
		{"juju vs shocking grasp", gamepack.JujuImmunityEffectCode, gamepack.SpellIDShockingGrasp, 0x0c, 1, 9, 0},
		{"efreeti per die", gamepack.FireToleranceEffectCode, gamepack.SpellIDFireball, 9, 5, 20, 15},
		{"ring per die floors at dice", gamepack.FireResistanceEffectCode, gamepack.SpellIDFireball, 9, 5, 12, 5},
		{"ring on a one-die one", gamepack.FireResistanceEffectCode, gamepack.SpellIDBurningHands, 9, 1, 1, 255},
		{"mummy vs fireball", gamepack.FireVulnerabilityEffectCode, gamepack.SpellIDFireball, 9, 5, 20, 25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outcome := gamepack.SpellDamageEffects{
				Effects:         gamepack.EffectList{gamepack.NewEffectNode(tc.code, 0, 0xff, false)},
				Spell:           tc.spell,
				DamageFlags:     tc.flags,
				Roll:            roll,
				Dice:            tc.dice,
				ActorWeaponType: -1,
			}.Apply(tc.in)
			if outcome.Damage != tc.want {
				t.Fatalf("%d → %d, want %d", tc.in, outcome.Damage, tc.want)
			}
		})
	}
}

// 群組 4 的 `03h`／`06h`（近戰傷害骰之後，攻擊者身上）：`03h` 對 `+9Fh == 4` +2，`06h` 依種類
// 加 1／2／3（`0141h`、`01C9h`）。從 A 出手：1d1 的第一形態打一隻不死生物。
func TestMeleeBaneCodesAddDamageToTheRightKind(t *testing.T) {
	for _, tc := range []struct {
		code  uint8
		kind  uint8
		extra int
	}{
		{gamepack.UndeadBaneEffectCode, 4, 2},
		{gamepack.UndeadBaneEffectCode, 0, 0},
		{gamepack.CreatureBaneEffectCode, 4, 3},
		{gamepack.CreatureBaneEffectCode, 9, 2},
		{gamepack.CreatureBaneEffectCode, 0x0a, 1},
	} {
		application, state := effectOnlyBoard(t, 19)
		state.setSingleAttackForm(2, combat.DamageDice{Count: 1, Sides: 1})
		state.Effects[2] = gamepack.EffectList{gamepack.NewEffectNode(tc.code, 0, 0xff, false)}
		state.CreatureType[3] = tc.kind
		before := state.HitPoints[3]
		attackWithKeys(t, application, state, 2, 3)
		if lost := before - state.HitPoints[3]; lost != 1+tc.extra {
			t.Errorf("%02Xh vs kind %d: lost %d, want %d (%q)", tc.code, tc.kind, lost, 1+tc.extra, state.Status)
		}
	}
}

// 靈魂鎚在怪物身上（`07F6h` 不看 `+10Eh`）：怪物施法者也拿到鎚子，節點到期收回。
func TestSpiritualHammerAlsoArmsAFoeCaster(t *testing.T) {
	application, state := effectOnlyBoard(t, 19)
	state.Effects[3] = gamepack.EffectList{
		gamepack.NewEffectNode(gamepack.SpiritualHammerEffectCode, 2, 3, true)}
	application.grantSpiritualHammer(state, 3)
	if !hasSpiritualHammer(state.FoeItems[3]) {
		t.Fatalf("the foe caster has no hammer: %+v (%q)", state.FoeItems[3], state.Status)
	}
	for guard := 0; guard < 4 && state.hasEffect(3, gamepack.SpiritualHammerEffectCode); guard++ {
		endRoundWithKeys(t, application, state)
	}
	if hasSpiritualHammer(state.FoeItems[3]) {
		t.Fatal("the foe's hammer outlived its node")
	}
}

// 致病在怪物身上也到期收尾（`0BE3h`／`1177h` 不看 `+10Eh`）：`2Ch` 扣 1 點、重掛。
func TestCauseDiseaseAlsoWastesAFoe(t *testing.T) {
	application, state := effectOnlyBoard(t, 19)
	state.Effects[3] = gamepack.EffectList{
		gamepack.NewEffectNode(gamepack.DiseaseWastingEffectCode, 1, 3, true)}
	before := state.HitPoints[3]
	endRoundWithKeys(t, application, state)
	if state.HitPoints[3] != before-1 || !state.hasEffect(3, gamepack.DiseaseWastingEffectCode) {
		t.Fatalf("foe wasting: hp %d → %d, effects %+v", before, state.HitPoints[3], state.Effects[3])
	}
}

var _ = ebiten.KeyEnter
