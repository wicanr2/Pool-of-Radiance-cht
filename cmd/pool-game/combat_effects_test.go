package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 效果串列在原版只有一份（spec 069），所以走進戰場的人身上帶著什麼，戰場上
// 就該看得到；打完走出來，戰場上多的那一份也要留在身上。
//
// 這一條走的是玩家的路徑：真的開盤面、真的收場，不是直接呼叫搬運那兩支——
// 直接呼叫證明不了呼叫端有接上。
func TestEffectsTravelIntoAndOutOfCombat(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	blessed := gamepack.NewEffectNode(0x3B, 40, 5, false)
	application.state.Party[0].Effects = storedEffects(gamepack.EffectList{blessed})

	if err := application.enterTacticalPreview(); err != nil {
		t.Fatalf("開戰場：%v", err)
	}
	state := application.tactical
	slot := -1
	for index, party := range state.PartySlot {
		if party == 0 {
			slot = index
			break
		}
	}
	if slot < 0 {
		t.Fatal("第一個隊員沒有站上戰場")
	}
	if !state.Effects[slot].Has(0x3B) {
		t.Fatalf("帶進戰場的效果是 %v，該有 3Bh", state.Effects[slot])
	}
	// 開盤面會開第一回合，回合邊界對有計時的效果各減一（spec 062），所以這裡
	// 看到的是 39。重點是**持續跟著進來**，不是被清成 0 或整個節點不見。
	if got := state.Effects[slot][0].Duration(); got != 39 {
		t.Fatalf("持續帶成 %d，該是 40 減掉第一回合的 1", got)
	}
	// 負對照：沒掛效果的人進去就是空的——不是每個人都被塞了一份。
	for index, party := range state.PartySlot {
		if party > 0 && len(state.Effects[index]) != 0 {
			t.Fatalf("隊員 %d 身上憑空多了 %v", party, state.Effects[index])
		}
	}

	// 戰鬥裡中了一個減益，收場之後要跟著人走出去。
	poisoned := gamepack.NewEffectNode(0x37, 100, 3, false)
	state.Effects[slot] = state.Effects[slot].Append(poisoned)

	if err := application.finishCombat(combat.CombatOngoing); err != nil {
		t.Fatalf("收場：%v", err)
	}
	got := combatEffects(application.state.Party[0].Effects)
	if len(got) != 2 || !got.Has(0x3B) || !got.Has(0x37) {
		t.Fatalf("走出戰場之後身上是 %v，該有 3Bh 與 37h", got)
	}
	if index, _ := got.IndexOf(0x37); got[index].Duration() != 100 {
		t.Fatalf("戰鬥裡中的那一份持續寫回成 %d，該是 100", got[index].Duration())
	}
}

// 走一步一分鐘（spec 118），效果跟著減一分鐘——原版把兩件事寫在
// overlay-20 entry 2 的同一支裡。
func TestWalkingOneStepAgesTheEffectsByOneMinute(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	application.state.Party[0].Effects = storedEffects(
		gamepack.EffectList{gamepack.NewEffectNode(0x3B, 3, 5, false)})

	// 開場停在的那一格是換圖點，往哪走都不加時間（spec 118：換圖那一步不加）。
	// 站到一格走得動的地方再走——(11,2) 往北是港務長那條路（spec 102）。
	if application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在城區，而是 GEO%d/%d",
			application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 11, 2, 0
	before := application.gameTime
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	if application.gameTime == before {
		t.Fatalf("走了一步時鐘沒動，停在 %v", application.gameTime)
	}
	got := combatEffects(application.state.Party[0].Effects)
	if len(got) != 1 || got[0].Duration() != 2 {
		t.Fatalf("走一步之後效果是 %v，持續該從 3 變 2", got)
	}
}

// 紮營要把睡掉的時間走到世界時鐘上（一刻五分鐘），效果照同一個節奏遞減。
// 少了這一段，有時限的效果在營地裡永遠不會過期。
func TestRestingAdvancesTheClockAndAgesTheEffects(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	// 這一區不打擾（Period 0 是原版的初值），這條才量得到整段休息。
	application.eventMachine.Memory[gamepack.RestInterruptionPeriodAddress] = 0
	application.state.Party[0].Effects = storedEffects(gamepack.EffectList{
		gamepack.NewEffectNode(0x3B, 90, 5, false),
		{Code: 0x21},
	})
	application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)

	application.restParty()

	if got := application.gameTime[gamepack.TimeDigitHour]; got != 1 {
		t.Fatalf("睡了一小時，時鐘的小時是 %d", got)
	}
	if got := application.gameTime.Minutes(); got != 0 {
		t.Fatalf("睡了一小時，分鐘是 %d", got)
	}
	got := combatEffects(application.state.Party[0].Effects)
	if len(got) != 2 {
		t.Fatalf("休息之後身上是 %v，兩個都不該不見", got)
	}
	if index, _ := got.IndexOf(0x3B); got[index].Duration() != 30 {
		t.Fatalf("持續剩 %d，90 減 60 該是 30", got[index].Duration())
	}
	// 持續 0 是永久，睡多久都不動（spec 069）。
	if index, _ := got.IndexOf(0x21); got[index].Duration() != 0 {
		t.Fatalf("永久效果的持續被動成 %d", got[index].Duration())
	}
}

// 睡得夠久，有時限的效果就該不見——這是「時間會讓效果到期」的正對照，
// 上面那條只驗到持續變小。永久的那一個同時當負對照：它不該跟著消失。
func TestRestingLongEnoughExpiresTheEffect(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	application.eventMachine.Memory[gamepack.RestInterruptionPeriodAddress] = 0
	application.state.Party[0].Effects = storedEffects(gamepack.EffectList{
		gamepack.NewEffectNode(0x3B, 30, 5, false),
		{Code: 0x21},
	})
	application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)

	application.restParty()

	got := combatEffects(application.state.Party[0].Effects)
	if len(got) != 1 || got[0].Code != 0x21 {
		t.Fatalf("睡了一小時之後身上是 %v，30 分鐘那個該到期、永久那個該留著", got)
	}
	// 角色庫那一份要跟著走，不然離隊再入隊會把到期的效果帶回來。
	library := combatEffects(application.state.CharacterLibrary[0].Effects)
	if len(library) != 1 || library[0].Code != 0x21 {
		t.Fatalf("角色庫那一份是 %v", library)
	}
}
