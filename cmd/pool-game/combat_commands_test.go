package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 玩家的 T）urn、U）se（overlay-08 `0427h`／`03E9h`，issue #75）與受傷打斷
// （overlay-13 `04F6h`、overlay-24 `150Fh`，issue #77）。全部從 Update() 送鍵。

var turnUseSegments = []gamepack.CombatCommandSegment{
	{Key: gamepack.CombatCommandMove, Text: "Move "},
	{Key: gamepack.CombatCommandUse, Text: "Use "},
	{Key: gamepack.CombatCommandCast, Text: "Cast "},
	{Key: gamepack.CombatCommandTurn, Text: "Turn "},
	{Key: gamepack.CombatCommandQuickDone, Text: "Quick Done"},
}

// 八級牧師對骷髏（欄 1）：指令列有 Turn，按 T 走 116Ah——骰序 1d12、1d20，摧毀、
// 立 runtime +11h，這個行動用掉；同一場之後 Turn 不再出現（`077Dh`）。
func TestPlayerClericTurnsASkeleton(t *testing.T) {
	roller := &sequenceRoller{values: []int{1, 1, 1, 1, 1, 1, 1, 1}}
	application, state := newQuickTurnApp(t, 8, 1, roller)
	application.state.Party[0].Quick = false
	state.AIDriven[1] = false
	application.combatCommands = turnUseSegments
	if bar := application.combatCommandBar(); !strings.Contains(bar, "Turn") {
		t.Fatalf("a cleric's command bar lacks Turn: %q", bar)
	}
	if err := press(application, ebiten.KeyT); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "TURNS UNDEAD") || !strings.Contains(state.FoeLog, "SKELETON IS DESTROYED") {
		t.Fatalf("T did not destroy the skeleton: %q", state.FoeLog)
	}
	if state.Roster[2].FootprintClass != 0 || state.States[2] != turnDestroyedState {
		t.Fatalf("the destroyed skeleton is still on the board: %+v state %d", state.Roster[2], state.States[2])
	}
	if !askedRun(roller.asked, 12, 20) {
		t.Fatalf("dice %v, want the 116Ah pair d12 d20", roller.asked)
	}
	if !state.Undead.Tried[1] {
		t.Fatal("runtime +11h was not set (116Ah `11AAh`)")
	}
	state.Mover = 1
	if bar := application.combatCommandBar(); strings.Contains(bar, "Turn") {
		t.Fatalf("Turn is still offered after turning this fight: %q", bar)
	}
}

// 戰士（牧師等級 0）：指令列沒有 Turn，按 T 什麼也不做，一顆骰都不擲。
func TestFighterHasNoTurn(t *testing.T) {
	roller := &sequenceRoller{}
	application, state := newQuickTurnApp(t, 0, 1, roller)
	application.state.Party[0].Quick = false
	state.AIDriven[1] = false
	application.combatCommands = turnUseSegments
	if bar := application.combatCommandBar(); strings.Contains(bar, "Turn") {
		t.Fatalf("a fighter is offered Turn: %q", bar)
	}
	before := state.Status
	if err := press(application, ebiten.KeyT); err != nil {
		t.Fatal(err)
	}
	if len(roller.asked) != 0 || state.Undead.Tried[1] || state.Status != before || state.Mover != 1 {
		t.Fatalf("T acted for a fighter: dice %v tried %v status %q mover %d",
			roller.asked, state.Undead.Tried, state.Status, state.Mover)
	}
}

// 牧師對面沒有不死生物：`0427h` 不先挑，116Ah 照樣擲兩顆骰、印 Nothing Happens、
// 用掉行動（AI 那一側才會先問 1352h 而不轉）。
func TestPlayerTurnWithoutUndeadStillSpendsTheAction(t *testing.T) {
	roller := &sequenceRoller{values: []int{1, 1, 1, 1}}
	application, state := newQuickTurnApp(t, 3, 0, roller)
	application.state.Party[0].Quick = false
	state.AIDriven[1] = false
	application.combatCommands = turnUseSegments
	if err := press(application, ebiten.KeyT); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "NOTHING HAPPENS") || !askedRun(roller.asked, 12, 20) || !state.Undead.Tried[1] {
		t.Fatalf("turn without undead: log %q dice %v tried %v", state.FoeLog, roller.asked, state.Undead.Tried)
	}
	if state.Mover == 1 && state.Scores[1] != 0 {
		t.Fatalf("the action was not spent: mover %d score %d", state.Mover, state.Scores[1])
	}
}

// U 開物品選單，再按 U 用穿戴中的魔法飛彈杖：瞄準、放出去、次數減一（overlay-19
// `12B0h` → entry 8 `1A86h` → `1C3Fh`）。記憶陣列一格也不動（`DS:6CB3h`）。
func TestPlayerUsesAWandAndSpendsACharge(t *testing.T) {
	application, state := newQuickWandApp(t, 2, 0, true)
	application.state.Party[0].Quick = false
	state.AIDriven[1] = false
	application.combatCommands = turnUseSegments
	if bar := application.combatCommandBar(); !strings.Contains(bar, "Use") {
		t.Fatalf("a member carrying a wand lacks Use: %q", bar)
	}
	hp := state.HitPoints[2]
	pressAll(t, application, ebiten.KeyU)
	if application.combatItems == nil {
		t.Fatal("U did not open the item menu")
	}
	pressAll(t, application, ebiten.KeyU)
	if !strings.Contains(state.Status, "USES AN ITEM ITEM:WAND") {
		t.Fatalf("using the wand printed %q", state.Status)
	}
	if application.castTargeting {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if state.HitPoints[2] >= hp {
		t.Fatalf("the wand did no damage: hp %d → %d status %q", hp, state.HitPoints[2], state.Status)
	}
	inventory := application.state.Party[0].Inventory
	if len(inventory) != 1 || inventory[0].Raw[gamepack.AIItemChargesOffset] != 1 {
		t.Fatalf("one charge should be left: %+v", inventory)
	}
	if application.combatItem != nil || application.combatItems != nil {
		t.Fatal("the item menu or the pending use is still open")
	}
	if state.Mover == 1 && state.Scores[1] != 0 {
		t.Fatalf("using the wand did not spend the action: mover %d", state.Mover)
	}
}

// 沒穿戴的那一件：印 "Must be Readied"（`0EC6h`），留在選單，次數不動。
func TestPlayerCannotUseAnUnreadiedWand(t *testing.T) {
	application, state := newQuickWandApp(t, 2, 0, false)
	application.state.Party[0].Quick = false
	state.AIDriven[1] = false
	application.combatCommands = turnUseSegments
	pressAll(t, application, ebiten.KeyU, ebiten.KeyU)
	if !strings.Contains(state.Status, "MUST BE READIED") || application.combatItems == nil {
		t.Fatalf("unreadied wand: status %q menu %v", state.Status, application.combatItems)
	}
	if application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset] != 2 || state.Mover != 1 {
		t.Fatal("an unreadied wand was spent or used the action")
	}
	pressAll(t, application, ebiten.KeyEscape)
	if application.combatItems != nil {
		t.Fatal("ESC did not close the item menu")
	}
}

// 補血之後再受傷、生命值沒低過回合開頭：原版照樣在傷害入口清 runtime +1、丟失開始施法的
// 那一條（`04F6h`）。只比回合開頭生命值的判法會漏掉這一種。
func TestHealedCasterIsStillInterruptedByTheNextWound(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDFireball),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 6, Y: 5, FootprintClass: 1})
	application.combatCommands = turnUseSegments
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	if state.Casting.Pending[1] != gamepack.SpellIDFireball {
		t.Fatalf("fireball did not begin casting: %v", state.Casting.Pending)
	}
	// 同一回合先補血，然後輪到旁邊那一隻出手。
	state.HitPoints[1] += 20
	healed := state.HitPoints[1]
	state.Mover = 2
	state.AIDriven[2] = true
	application.roller = fixedRoller{20}
	pressAll(t, application, ebiten.KeyEnter)
	if state.HitPoints[1] >= healed {
		t.Fatalf("the foe never hit: hp %d, log %q", state.HitPoints[1], state.FoeLog)
	}
	if state.HitPoints[1] < 30 {
		t.Fatalf("the wound went below the round-start hit points (%d); the test needs a smaller hit", state.HitPoints[1])
	}
	// 敵方回合的訊息收在 FoeLog（攻擊那一行後面接著 "lost a spell"）。
	if len(state.Casting.Pending) != 0 || !strings.Contains(state.FoeLog, "LOST A SPELL") {
		t.Fatalf("the healed caster kept the spell: pending %v log %q", state.Casting.Pending, state.FoeLog)
	}
	if application.state.Party[0].Memorised[0] != 0 {
		t.Fatal("the lost fireball is still memorised (14ECh)")
	}
	if !state.castingDisrupted(1) {
		t.Fatal("runtime +1 is still set after the wound")
	}
}
