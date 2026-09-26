package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 祝福、詛咒、急速、緩速掛上去之後實際改的數值（spec 098〈模式 0Ah：效果怎麼掛上去〉、
// spec 112，issue #81）。全部從 Update() 送鍵：C 施法、瞄準、A 出手、ENTER 結束回合。

// sideEffectBoard：施法者 1 在 (5,5)；隊友 2 在 (8,5)（瞄準點，也是隊員，年齡 20）；
// 對面 3 在 (8,7)。三個人都由玩家操作（不交給 AI），ENTER 才推得動回合。
// THAC0 內部值 40、AC 內部值 50：命中骰要 10 才中。
func sideEffectBoard(t *testing.T, class int, level uint8, spell uint8, roll int) (*app, *tacticalState) {
	t.Helper()
	application, state := newSpellBoard(t,
		spellCasterWith(class, level, spell),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 7, FootprintClass: 1})
	state.Friendly[2], state.PartySlot[2] = true, 1
	application.state.Party = append(application.state.Party,
		poolsave.Character{Name: "ALLY", Age: 20, MaxHP: 30, CurrentHP: 30})
	for index := range state.AIDriven {
		state.AIDriven[index] = false
	}
	state.PartyAged = application.agePartyMember(state)
	application.roller = fixedRoller{roll}
	return application, state
}

// attackWithKeys 讓 attacker 輪到，按 A、把瞄準移到 target、ENTER 出手。
func attackWithKeys(t *testing.T, application *app, state *tacticalState, attacker, target uint8) {
	t.Helper()
	state.Mover, state.Scores[attacker] = attacker, 5
	pressAll(t, application, ebiten.KeyA)
	if !application.castTargeting || !application.castTargetingAttack {
		t.Fatalf("A did not open attack aiming: %q", state.Status)
	}
	for guard := 0; guard <= len(application.castTargets); guard++ {
		if application.castTargets[application.castTargetCursor] == target {
			break
		}
		pressAll(t, application, ebiten.KeyN)
	}
	if application.castTargets[application.castTargetCursor] != target {
		t.Fatalf("attack aiming never reached %d: %v", target, application.castTargets)
	}
	pressAll(t, application, ebiten.KeyEnter)
}

// endRoundWithKeys 一直按 ENTER 結束輪到的人，直到換到下一個回合。
func endRoundWithKeys(t *testing.T, application *app, state *tacticalState) {
	t.Helper()
	round := state.Round
	for guard := 0; guard < 64 && state.Round == round; guard++ {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if state.Round == round {
		t.Fatalf("ENTER never reached round %d: %q", round+1, state.Status)
	}
}

// 祝福術：參數表 `+0Ah` 的 01h 掛在留下的人身上，持續 `07C7h` 的 6 回合；那個人出手時
// 命中骰 +1（overlay-12 `0117h` `FE 06 80 67`）。命中骰 9 原本差一點，祝福之後打得中；
// 六個回合邊界之後節點摘掉，同一骰又打不中。
func TestBlessRaisesTheHitRollUntilItExpires(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, 9)
	castAndAimAtIndex(t, application, state, 2)
	at, ok := state.Effects[2].IndexOf(gamepack.BlessEffectCode)
	if !ok {
		t.Fatalf("ally 2 carries no bless node: %+v", state.Effects[2])
	}
	if got := state.Effects[2][at].Duration(); got != 6 {
		t.Errorf("bless lasts %d rounds, want 6 (+4 = 6, +5 = 0)", got)
	}
	if got := state.Effects[2][at].CasterLevel(); got != 1 {
		t.Errorf("the node records caster level %d, want 1", got)
	}
	if state.hasEffect(3, gamepack.BlessEffectCode) || state.hasEffect(1, gamepack.BlessEffectCode) {
		t.Fatal("bless reached someone outside the kept list")
	}

	state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 26 {
		t.Fatalf("a blessed roll of 9 should hit for 4: foe has %d hp, status %q", state.HitPoints[3], state.Status)
	}

	cast := state.Round
	for state.hasEffect(2, gamepack.BlessEffectCode) {
		if state.Round > cast+6 {
			t.Fatalf("bless still on after round %d", state.Round)
		}
		endRoundWithKeys(t, application, state)
	}
	if state.Round != cast+6 {
		t.Errorf("bless expired at round %d, want %d", state.Round, cast+6)
	}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 26 {
		t.Fatalf("after bless expires a roll of 9 should miss: foe has %d hp", state.HitPoints[3])
	}
}

// 詛咒術留的是對面：02h 讓牠的命中骰 −1（overlay-12 `0137h` `FE 0E 80 67`）。命中骰 10
// 原本剛好打中，被詛咒的一方同一骰打不中；施法者這一邊不受影響。
func TestCurseLowersTheOtherSidesHitRoll(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDCurse, 10)
	castAndAimAtIndex(t, application, state, 2)
	if !state.hasEffect(3, gamepack.CurseEffectCode) {
		t.Fatalf("foe 3 carries no curse node: %+v", state.Effects[3])
	}
	if state.hasEffect(2, gamepack.CurseEffectCode) {
		t.Fatal("curse reached the caster's own side")
	}
	state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
	attackWithKeys(t, application, state, 3, 2)
	if state.HitPoints[2] != 30 {
		t.Fatalf("a cursed roll of 10 should miss: ally has %d hp, status %q", state.HitPoints[2], state.Status)
	}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 26 {
		t.Fatalf("the uncursed side hits on 10: foe has %d hp", state.HitPoints[3])
	}
}

// 急速術：掛參數表的 27h，持續 3 + 等級；放出去當下 `2835h` 派發群組 18，那個人老一歲
// （`0CB0h`），之後不再老。下一回合初始化起攻擊次數編碼與移動都左移一位：一回合兩下、
// 移動加倍。當回合還是原本的一下——原版把次數在回合初始化時就算好了。
func TestHasteDoublesAttacksAndMovementFromTheNextRound(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDHaste, 19)
	castAndAimAtIndex(t, application, state, 2)
	at, ok := state.Effects[2].IndexOf(gamepack.HasteEffectCode)
	if !ok {
		t.Fatalf("ally 2 carries no haste node: %+v", state.Effects[2])
	}
	if got := state.Effects[2][at].Duration(); got != 8 {
		t.Errorf("haste lasts %d rounds, want 3 + 5", got)
	}
	if state.hasEffect(3, gamepack.HasteEffectCode) {
		t.Fatal("haste reached the other side")
	}
	if got := application.state.Party[1].Age; got != 21 {
		t.Fatalf("the hasted ally is %d years old, want 21", got)
	}

	state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 26 {
		t.Fatalf("haste must not add a swing in the round it was cast: foe has %d hp", state.HitPoints[3])
	}
	endRoundWithKeys(t, application, state)
	if state.Budgets[2] != 36 || state.Budgets[3] != 18 {
		t.Errorf("movement after haste: ally %d (want 36), foe %d (want 18)", state.Budgets[2], state.Budgets[3])
	}
	if got := application.state.Party[1].Age; got != 21 {
		t.Errorf("haste aged the ally again at the round start: %d", got)
	}
	attackWithKeys(t, application, state, 2, 3)
	if state.HitPoints[3] != 18 {
		t.Fatalf("a hasted attacker swings twice for 4 each: foe has %d hp", state.HitPoints[3])
	}
}

// 緩速術留對面、掛 2Ah：下一回合起攻擊次數編碼與移動除以 2。一回合一下（編碼 2）變成
// 編碼 1——單數相位一下、雙數相位零下；移動 18 變 9。
func TestSlowHalvesAttacksAndMovementFromTheNextRound(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDSlow, 19)
	castAndAimAtIndex(t, application, state, 2)
	if !state.hasEffect(3, gamepack.SlowEffectCode) {
		t.Fatalf("foe 3 carries no slow node: %+v", state.Effects[3])
	}
	if state.hasEffect(2, gamepack.SlowEffectCode) {
		t.Fatal("slow reached the caster's own side")
	}
	endRoundWithKeys(t, application, state)
	if state.Budgets[3] != 9 || state.Budgets[2] != 18 {
		t.Errorf("movement after slow: foe %d (want 9), ally %d (want 18)", state.Budgets[3], state.Budgets[2])
	}
	want := 30
	if state.AttackPhase&1 == 1 {
		want = 26
	}
	state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
	attackWithKeys(t, application, state, 3, 2)
	if state.HitPoints[2] != want {
		t.Fatalf("slowed foe in phase %d: ally has %d hp, want %d", state.AttackPhase&1, state.HitPoints[2], want)
	}
	endRoundWithKeys(t, application, state)
	want -= 4 * int(state.AttackPhase&1)
	attackWithKeys(t, application, state, 3, 2)
	if state.HitPoints[2] != want {
		t.Fatalf("slowed foe in phase %d: ally has %d hp, want %d", state.AttackPhase&1, state.HitPoints[2], want)
	}
}

// 急速與緩速互相抵銷：`2724h` 先拿對面那一支的碼問 `0100h:006Bh`，身上有就摘掉，這一次
// 不掛自己的碼（額度照扣）。
func TestHasteCancelsSlowInsteadOfHasting(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDHaste, 19)
	state.addEffect(2, gamepack.SlowEffectCode, 5, 5)
	castAndAimAtIndex(t, application, state, 2)
	if state.hasEffect(2, gamepack.SlowEffectCode) || state.hasEffect(2, gamepack.HasteEffectCode) {
		t.Fatalf("haste should cure the slow and stop there: %+v", state.Effects[2])
	}
	if got := application.state.Party[1].Age; got != 20 {
		t.Errorf("the cured ally aged to %d", got)
	}
}

// 已經祝福過的人再被祝福：entry 20 在舊節點剩得比新的少時先摘掉舊的（`16DAh`），
// 不會疊成兩層，命中修正仍是 +1。
func TestBlessTwiceRefreshesInsteadOfStacking(t *testing.T) {
	application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, 9)
	state.addEffect(2, gamepack.BlessEffectCode, 2, 1)
	castAndAimAtIndex(t, application, state, 2)
	count := 0
	for _, node := range state.Effects[2] {
		if node.Code == gamepack.BlessEffectCode {
			count++
			if node.Duration() != 6 {
				t.Errorf("the kept bless node has %d rounds, want the fresh 6", node.Duration())
			}
		}
	}
	if count != 1 {
		t.Fatalf("ally 2 carries %d bless nodes, want 1: %+v", count, state.Effects[2])
	}
	if got, missed := state.hitRollAfterEffects(2, 3, 10); got != 1 || missed {
		t.Errorf("bless modifier is %d (missed %v), want +1", got, missed)
	}
}
