package main

// overlay-09 entry 9 結尾的重算（spec 151〈每回合的重算〉〈NPC 的重算〉，issue #109）：
// `1808h` 與 `18FEh` 不論換沒換都跑 overlay-25 entry 7，NPC 與怪物同一支。盤面是獸人家那一場
// （orcHomeGearFixture），兩條都從 tacticalInput() 驅動，與遊戲裡輪到電腦時同一條路。

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// aiTurnByInput 讓 mover 從 tacticalInput() 走一個回合：先把上一個行動的停拍按掉，再交給分派。
func aiTurnByInput(t *testing.T, application *app, mover uint8) {
	t.Helper()
	state := application.tactical
	for guard := 0; guard < 1000 && application.holdCombatNotice(state); guard++ {
	}
	state.Prompt, state.Mover = false, mover
	state.Budgets[mover] = state.BaseMovement[mover] * 2
	if err := application.tacticalInput(); err != nil {
		t.Fatal(err)
	}
}

// 臭雲把 AC 改在記錄上（overlay-12 `0AD3h..0AEFh`），而電腦下一次輪到、entry 9 跑完時的
// entry 7 從物品重算 `+111h`（`0E43h` 從 `+0A9h` 起算、`0FA9h..0FFBh` 結算），咳嗽的那兩點就沒了。
// 咳嗽的那一回合收掉了行動權，entry 9 沒跑，所以那一回合 AC 是差的。
func TestAIGearRecomputeWashesTheStinkingCloudArmourClass(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	application.roller = fixedRoller{value: 1}
	state := application.tactical
	// 身上沒有物品的獸人（MON2 block 4）：entry 9 什麼都不換，只剩結尾那兩次 entry 7。
	var mover uint8
	for index := 1; index < len(state.Roster); index++ {
		if !state.Friendly[index] && len(state.FoeItems[index]) == 0 {
			mover = uint8(index)
			break
		}
	}
	if mover == 0 {
		t.Fatal("no foe without items at the orc home")
	}
	clearFoesExcept(state, mover)
	base := state.ArmorClass[mover]
	coughed := gamepack.StinkingCloudArmourClass(base)
	if coughed == base {
		t.Fatalf("internal AC %#x is already at the floor; pick a combatant the cloud can worsen", base)
	}

	here := state.Roster[mover]
	if !state.placeCloud(1, int(here.X), int(here.Y), 3) {
		t.Fatal("could not place the cloud on the orc")
	}
	state.addEffect(int(mover), gamepack.StinkingCloudEffectCode, 0, 1)
	aiTurnByInput(t, application, mover)
	if state.ArmorClass[mover] != coughed {
		t.Fatalf("after coughing the orc's internal AC is %#x, want %#x", state.ArmorClass[mover], coughed)
	}

	// 走出雲（雲收掉、效果摘掉）之後的下一回合：entry 9 重挑武器，結尾的 entry 7 把 AC 算回來。
	if !state.disperseCloud(1, 1) {
		t.Fatal("the cloud did not disperse")
	}
	if at, ok := state.Effects[mover].IndexOf(gamepack.StinkingCloudEffectCode); ok {
		state.Effects[mover] = state.Effects[mover].RemoveAt(at)
	}
	var victim uint8
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] && state.Roster[index].FootprintClass != 0 {
			victim = uint8(index)
			break
		}
	}
	state.FoeTargets[mover] = victim
	aiTurnByInput(t, application, mover)
	if state.ArmorClass[mover] != base {
		t.Fatalf("after the AI's gear choice the orc's internal AC is %#x, want %#x recomputed from its record (log %q)",
			state.ArmorClass[mover], base, state.FoeLog)
	}
}

// 交給電腦的 NPC 拿著弓卻沒有箭：entry 9 換回長劍，結尾的 entry 7 對 NPC 一樣重算（spec 147〈NPC〉
// 的呼叫鏈沒有 NPC 分支），射程與傷害骰回到長劍的。
func TestAIDrivenNPCIsRecomputedAfterTheGearChoice(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	application.roller = fixedRoller{value: 1}
	record, err := application.loadMonster(3, 0x6D)
	if err != nil {
		t.Fatal(err)
	}
	items, err := application.npcItems(3, 0x6D)
	if err != nil {
		t.Fatal(err)
	}
	hero := poolsave.Character{Name: "HERO", NPC: true, Record: append([]byte(nil), record.Raw[:]...),
		MaxHP: int(record.MaxHitPoints()), CurrentHP: int(record.CurrentHitPoints()), Inventory: items}
	application.state.Party = append([]poolsave.Character(nil), application.state.Party...)
	slot := len(application.state.Party) - 1
	application.state.Party[slot] = hero
	application.tactical = nil
	if err := application.enterTacticalPreview(); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	index := npcBoardIndex(t, state, slot)
	sword := state.AttackForms[index][0]
	if sword != (combat.DamageDice{Count: 1, Sides: 10, Bonus: 2}) {
		t.Fatalf("HERO starts with %+v, want the long sword's 1d10+2", sword)
	}

	// 戰鬥中改拿短弓（物品選單那一條也重算），箭已經射完（那一件摘掉，spec 151 `1A7Fh`）。
	member := &application.state.Party[slot]
	kept := member.Inventory[:0]
	for _, item := range member.Inventory {
		if item.Raw[itemTypeOffset] != gamepack.ItemTypeArrow {
			kept = append(kept, item)
		}
	}
	if len(kept) == len(member.Inventory) {
		t.Fatal("HERO carries no arrows to use up")
	}
	member.Inventory = kept
	for i := range member.Inventory {
		switch member.Inventory[i].Raw[itemTypeOffset] {
		case 0x26:
			member.Inventory[i].Raw[itemReadyOffset] = 0
		case 0x2B:
			member.Inventory[i].Raw[itemReadyOffset] = 1
		}
	}
	state.Mover = uint8(index)
	if err := application.afterCombatItemChange(state, slot); err != nil {
		t.Fatal(err)
	}
	if state.AttackRange[index] <= 1 {
		t.Fatalf("HERO reach %d with the short bow readied", state.AttackRange[index])
	}

	state.AIDriven[index] = true
	aiTurnByInput(t, application, uint8(index))
	readied := func(itemType uint8) bool {
		for _, item := range application.state.Party[slot].Inventory {
			if item.Raw[itemTypeOffset] == itemType {
				return item.Raw[itemReadyOffset] != 0
			}
		}
		return false
	}
	if readied(0x2B) || !readied(0x26) {
		t.Fatalf("entry 9 left bow readied %v, long sword readied %v; want the long sword back (log %q)",
			readied(0x2B), readied(0x26), state.FoeLog)
	}
	if state.AttackRange[index] != 1 || state.AttackForms[index][0] != sword {
		t.Errorf("after the AI readied the long sword HERO has reach %d and %+v; want 1 and %+v",
			state.AttackRange[index], state.AttackForms[index][0], sword)
	}
}
