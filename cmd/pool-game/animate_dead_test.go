package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 死靈術在營地（overlay-22 `2043h`，spec 098〈死靈術〉，#108）與縮小術在戰鬥中（`135Eh`）。
// 全部從 Update() 送鍵。

func deadMember(name string) poolsave.Character {
	member := campCaster(name, 1, gamepack.SpellIDBless)
	member.Status, member.CurrentHP, member.MaxHP = combat.DeadState, 0, 15
	return member
}

// 營地的死靈術：額度是施法者等級（1），沿隊伍順序叫起第一個死掉的人——狀態 1、生命補到上限、
// Quick（`+10Fh`）、`+72h` 6、`+9Fh` 4、`+84h` B3h（玩家建的角色原本是 0）、記憶陣列清空、掛 20h。
// 第二個死掉的人額度用完不動，活著的人不動。
func TestCampAnimateDeadRaisesTheFirstDeadMember(t *testing.T) {
	application := campCastApp(t, campCaster("A", 1, gamepack.SpellIDAnimateDead), deadMember("B"),
		deadMember("C"), campCaster("D", 1))
	message := castInCamp(t, application, 0, gamepack.SpellIDAnimateDead, 0)
	raised := application.state.Party[1]
	if raised.Status != gamepack.AnimatedState || raised.CurrentHP != 15 || !raised.Quick {
		t.Fatalf("B status %d hp %d quick %v (%q)", raised.Status, raised.CurrentHP, raised.Quick, message)
	}
	if raised.Animated == nil || *raised.Animated != (poolsave.AnimatedRecord{
		Movement: gamepack.AnimatedDeadMovementRate, CreatureType: gamepack.CreatureTypeUndead, Morale: 0xB3}) {
		t.Fatalf("B record fields %+v", raised.Animated)
	}
	if memorises(raised, gamepack.SpellIDBless) {
		t.Fatalf("B still remembers %v", raised.Memorised)
	}
	if _, ok := memberEffect(raised, gamepack.AnimateDeadEffectCode); !ok {
		t.Fatalf("no 20h on B: %v", raised.Effects)
	}
	if !strings.Contains(message, "is animated") {
		t.Fatalf("message %q", message)
	}
	if still := application.state.Party[2]; still.Status != combat.DeadState || still.Animated != nil {
		t.Fatalf("the budget of one also raised C: %+v", still)
	}
	if alive := application.state.Party[3]; alive.Animated != nil || alive.Quick {
		t.Fatalf("the living member changed: %+v", alive)
	}
	if memorises(application.state.Party[0], gamepack.SpellIDAnimateDead) {
		t.Fatal("the caster kept the spell")
	}
}

// 叫起來的人帶著三格進下一場：不死生物、腳程以 6 為基礎、`+84h` B3h、由 AI 走；SPACE 收回
// 自動戰鬥時照原版只收 `+84h < 80h` 的人（overlay-08 `04D8h`），它留在 AI 手上。
func TestAnimatedMemberCarriesItsRecordIntoTheNextFight(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	const raised, quick = 1, 2
	application.state.Party = append([]poolsave.Character(nil), application.state.Party...)
	application.state.Party[raised].Status = gamepack.AnimatedState
	application.state.Party[raised].Quick = true
	application.state.Party[raised].Animated = &poolsave.AnimatedRecord{
		Movement: gamepack.AnimatedDeadMovementRate, CreatureType: gamepack.CreatureTypeUndead, Morale: 0xB3}
	application.state.Party[quick].Quick = true
	application.tactical = nil
	if err := application.enterTacticalPreview(); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	index := npcBoardIndex(t, state, raised)
	other := npcBoardIndex(t, state, quick)
	if state.CreatureType[index] != gamepack.CreatureTypeUndead || state.Morale.Raw[index] != 0xB3 {
		t.Fatalf("raised member creature type %d morale %02X", state.CreatureType[index], state.Morale.Raw[index])
	}
	if state.BaseMovement[index] > gamepack.AnimatedDeadMovementRate || state.BaseMovement[other] <= gamepack.AnimatedDeadMovementRate {
		t.Fatalf("movement raised %d other %d; want at most 6 and the usual 12-based rate",
			state.BaseMovement[index], state.BaseMovement[other])
	}
	if !state.aiDrives(index) || !state.aiDrives(other) {
		t.Fatalf("AI drives raised %v quick %v", state.aiDrives(index), state.aiDrives(other))
	}
	if err := press(application, ebiten.KeySpace); err != nil {
		t.Fatal(err)
	}
	if !state.aiDrives(index) || state.aiDrives(other) {
		t.Fatalf("after SPACE: raised AI %v, the plain Quick member AI %v", state.aiDrives(index), state.aiDrives(other))
	}
}

// 戰鬥中的死靈術叫起一名死掉的隊員：盤面那一份照舊，存檔那一份也寫下三格、Quick 與清空記憶。
func TestAnimateDeadInCombatRecordsThePartyMember(t *testing.T) {
	application, state := effectOnlyBoard(t, 19, gamepack.SpellIDAnimateDead)
	application.state.Party[1].Memorised = []uint8{gamepack.SpellIDBless}
	state.States[2], state.HitPoints[2] = combat.DeadState, 0
	state.MaxHitPoints = []int{0, 30, 30, 30}
	state.CreatureType = make([]uint8, len(state.Roster))
	state.Footprint = []uint8{0, 1, 1, 1}
	state.Roster[2].FootprintClass = 0
	castFromMenuAt(t, application, state, 1, 0, 1)
	member := application.state.Party[1]
	if state.States[2] != gamepack.AnimatedState || member.Animated == nil || !member.Quick ||
		member.Animated.CreatureType != gamepack.CreatureTypeUndead || member.Animated.Morale != 0xB3 {
		t.Fatalf("board state %d record %+v quick %v (%q)", state.States[2], member.Animated, member.Quick, state.Status)
	}
	if memorises(member, gamepack.SpellIDBless) {
		t.Fatalf("the raised member still remembers %v", member.Memorised)
	}
}

// 戰鬥中的縮小術（`135Eh`）：豁免沒過而且身上有變大術的 `0Ch`，經 entry 15 摘掉最早的那一個、
// 收尾把力量還原；豁免過了（自然 20）就什麼都不做。
func TestReduceInCombatUndoesTheEnlargement(t *testing.T) {
	for _, saved := range []bool{false, true} {
		application, state := effectOnlyBoard(t, 19, gamepack.SpellIDEnlarge, gamepack.SpellIDReduce)
		castFromMenuAt(t, application, state, 1, 0, 2)
		if !state.hasEffect(2, gamepack.EnlargeEffectCode) {
			t.Fatalf("enlarge attached nothing: %+v (%q)", state.Effects[2], state.Status)
		}
		enlarged := application.state.Party[1].Abilities[gamepack.AbilityStrength]
		if saved {
			application.roller = fixedRoller{20}
		}
		castFromMenuAt(t, application, state, 1, 0, 2)
		still := state.hasEffect(2, gamepack.EnlargeEffectCode)
		strength := application.state.Party[1].Abilities[gamepack.AbilityStrength]
		if saved {
			if !still || strength != enlarged {
				t.Fatalf("saved: enlargement gone %v strength %d (%q)", !still, strength, state.Status)
			}
			continue
		}
		if still || strength != 12 {
			t.Fatalf("failed save: enlargement kept %v strength %d status %q", still, strength, state.Status)
		}
		if _, ok := memberEffect(application.state.Party[1], gamepack.EnlargeEffectCode); ok {
			t.Fatalf("the party record kept 0Ch: %v", application.state.Party[1].Effects)
		}
	}
}
