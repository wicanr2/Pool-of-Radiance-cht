package main

// "Attack Ally:"（overlay-13 `2977h`，spec 162，#118）。盤面是瓦海登墳場 ECL4 block 10：五名戰士、
// 吸血鬼、三隻 106、跟著隊伍的 EFREETI（`+84h` B2h、`+10Eh` 0）。兩條路都從 Update() 送鍵：
// 走進自己人那一格（overlay-08 `0D76h`）與 A）IM 的 Target（overlay-13 `2C3Fh`）。

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// allyFixture 挑出第一個隊員（輪到他、由玩家走）與 EFREETI，其餘的怪物離場。
func allyFixture(t *testing.T) (*app, *tacticalState, uint8, uint8, uint8) {
	t.Helper()
	application := valhingenFixture(t)
	application.roller = fixedRoller{value: 20}
	state := application.tactical
	hero, ally, efreeti, vampire := uint8(0), uint8(0), uint8(0), uint8(0)
	for index := 1; index < len(state.Roster); index++ {
		switch {
		case state.PartySlot[index] >= 0 && hero == 0:
			hero = uint8(index)
		case state.PartySlot[index] >= 0 && ally == 0:
			ally = uint8(index)
		case state.PartySlot[index] < 0:
			monster, ok := application.stagedMonsterFor(index, state.PartySlot, state.Friendly)
			if ok && monster.Record.Name == "EFREETI" {
				efreeti = uint8(index)
			} else if ok && monster.Record.Name == "VAMPIRE" {
				vampire = uint8(index)
			}
		}
	}
	if hero == 0 || ally == 0 || efreeti == 0 || vampire == 0 {
		t.Fatalf("board is missing someone: hero %d ally %d efreeti %d vampire %d", hero, ally, efreeti, vampire)
	}
	keepOnly(state, hero, ally, efreeti, vampire)
	placeAt(t, state, efreeti, hero, 1)
	placeAt(t, state, ally, hero, 1)
	state.Prompt, state.Mover, state.Moving, state.Notices = false, hero, false, nil
	state.Budgets[hero] = state.BaseMovement[hero] * 2
	state.FoeTargets[efreeti] = vampire
	state.HitPoints[efreeti], state.HitPoints[ally] = 100, 100
	return application, state, hero, ally, efreeti
}

// stepInto 找出從 hero 往哪一個方向走會撞到 target。
func stepInto(t *testing.T, state *tacticalState, hero, target uint8) int {
	t.Helper()
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	for direction := 0; direction < 8; direction++ {
		outcome, err := combat.ResolveDestination(snapshot, hero, uint8(direction), state.Budget())
		if err == nil && outcome.Action == combat.MovementAttack && outcome.Target == target {
			return direction
		}
	}
	t.Fatalf("no direction from %d runs into %d", hero, target)
	return 0
}

func efreetiRecordSide(application *app) uint8 {
	for _, monster := range application.combatMonsters {
		if monster.Record.Name == "EFREETI" {
			return monster.Record.Raw[gamepack.RecordSideOffset]
		}
	}
	return 0xFF
}

// 走進 EFREETI 那一格：先問；不是 Y 什麼也不做，Y 才翻成敵方並打下去。
func TestBumpingAnAllyAsksAttackAlly(t *testing.T) {
	application, state, hero, _, efreeti := allyFixture(t)
	key := tacticalStepKeypad[stepInto(t, state, hero, efreeti)]

	if err := press(application, key); err != nil {
		t.Fatal(err)
	}
	if state.AllyPrompt == nil || state.AllyPrompt.target != efreeti || state.AllyPrompt.aimed {
		t.Fatalf("bumping the EFREETI did not ask: prompt %+v status %q", state.AllyPrompt, state.Status)
	}
	// 不是 Y：`29C1h` 回 0，`0D7Dh` 跳到結尾——沒打、沒倒戈、還是他的回合。
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	if state.AllyPrompt != nil || !state.Friendly[efreeti] || state.HitPoints[efreeti] != 100 ||
		state.Mover != hero || application.eventMachine.Memory[attackedAllyAddress] != 0 {
		t.Fatalf("N should do nothing: prompt %+v friendly %v HP %d mover %d @6E33 %d", state.AllyPrompt,
			state.Friendly[efreeti], state.HitPoints[efreeti], state.Mover, application.eventMachine.Memory[attackedAllyAddress])
	}

	if err := press(application, key); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyY); err != nil {
		t.Fatal(err)
	}
	// `29D4h` @6E33 = 1；`2A0Ch` +10Eh = 1、追的目標清掉；接著照樣打。
	if state.Friendly[efreeti] || state.FoeTargets[efreeti] != 0 || efreetiRecordSide(application) != 1 ||
		application.eventMachine.Memory[attackedAllyAddress] != 1 {
		t.Fatalf("Y should turn the EFREETI: friendly %v target %d record side %d @6E33 %d", state.Friendly[efreeti],
			state.FoeTargets[efreeti], efreetiRecordSide(application), application.eventMachine.Memory[attackedAllyAddress])
	}
	if state.HitPoints[efreeti] >= 100 {
		t.Fatalf("the EFREETI was not hit after Y (HP %d)", state.HitPoints[efreeti])
	}
	// 隊員（`+84h` 0）一個都不倒戈。
	for index := 1; index < len(state.Roster); index++ {
		if state.PartySlot[index] >= 0 && !state.Friendly[index] {
			t.Fatalf("party member %d turned", index)
		}
	}
	// 翻過去的 EFREETI 在戰後算經驗值（overlay-05 entry 2 只跳過 `+10Eh != 1`）。
	want := uint32(0)
	for _, monster := range application.combatMonsters {
		want += monster.Record.ExperienceValue(int(monster.Record.MaxHitPoints())) * uint32(monster.Spawn.Count)
	}
	if share := application.awardCombatExperienceWithLoot(nil, 0, []bool{true, true, true, true, true}, 0); share != want/5 {
		t.Fatalf("share %d, want %d with the EFREETI counted", share, want/5)
	}
}

// A）IM 瞄一個隊員按 Target：先問；不是 Y 就留在瞄準列，Y 才打。打的是隊員，倒戈的是 EFREETI。
func TestAimingAtAnAllyAsksAttackAlly(t *testing.T) {
	application, state, _, ally, efreeti := allyFixture(t)
	if err := press(application, ebiten.KeyA); err != nil {
		t.Fatal(err)
	}
	for guard := 0; guard < 64 && application.castTargets[application.castTargetCursor] != ally; guard++ {
		if err := press(application, ebiten.KeyN); err != nil {
			t.Fatal(err)
		}
	}
	if !application.castTargeting || application.castTargets[application.castTargetCursor] != ally {
		t.Fatalf("aim never reached the ally (targeting %v)", application.castTargeting)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.AllyPrompt == nil || !state.AllyPrompt.aimed {
		t.Fatalf("Target on an ally did not ask: prompt %+v status %q", state.AllyPrompt, state.Status)
	}
	// 其他鍵：`2C46h` 旗標清 0，`376Dh` 留在瞄準列。
	if err := press(application, ebiten.KeyEscape); err != nil {
		t.Fatal(err)
	}
	if state.AllyPrompt != nil || !application.castTargeting || !application.castTargetingAttack ||
		state.HitPoints[ally] != 100 || !state.Friendly[efreeti] {
		t.Fatalf("declining should leave the aim open: prompt %+v targeting %v HP %d", state.AllyPrompt,
			application.castTargeting, state.HitPoints[ally])
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyY); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[ally] >= 100 || !state.Friendly[ally] {
		t.Fatalf("Y should hit the ally and leave him on the party side: HP %d friendly %v", state.HitPoints[ally], state.Friendly[ally])
	}
	if state.Friendly[efreeti] || application.eventMachine.Memory[attackedAllyAddress] != 1 {
		t.Fatalf("attacking an ally should turn the EFREETI: friendly %v @6E33 %d", state.Friendly[efreeti],
			application.eventMachine.Memory[attackedAllyAddress])
	}
}

// 打對面的人不問（`2983h`）；開打時 @6E33 歸零（overlay-10 `1F76h`）。
func TestAttackingAFoeDoesNotAsk(t *testing.T) {
	application, state, hero, _, _ := allyFixture(t)
	if got := application.eventMachine.Memory[attackedAllyAddress]; got != 0 {
		t.Fatalf("@6E33 is %d at the start of the fight", got)
	}
	var vampire uint8
	for index := 1; index < len(state.Roster); index++ {
		if !state.Friendly[index] && state.Roster[index].FootprintClass != 0 {
			vampire = uint8(index)
		}
	}
	placeAt(t, state, vampire, hero, 1)
	if err := press(application, tacticalStepKeypad[stepInto(t, state, hero, vampire)]); err != nil {
		t.Fatal(err)
	}
	if state.AllyPrompt != nil || state.Activity.PartyAttacks == 0 {
		t.Fatalf("attacking the vampire asked or did nothing: prompt %+v attacks %d", state.AllyPrompt, state.Activity.PartyAttacks)
	}
}

// 開打時 @6E33 歸零（overlay-10 `1F76h`）：上一場打過自己人，這一場重新算。
func TestCombatStartClearsAttackedAlly(t *testing.T) {
	application := valhingenFixture(t)
	application.eventMachine.Memory[attackedAllyAddress] = 1
	application.tactical, application.combatActive, application.tacticalPreview = nil, false, false
	if err := application.enterCombatStaging([]eclvm.MonsterSpawn{{MonsterID: 23, Count: 1, IconBlock: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatal("no second fight")
	}
	if got := application.eventMachine.Memory[attackedAllyAddress]; got != 0 {
		t.Fatalf("@6E33 is %d after the fight was set up", got)
	}
}
