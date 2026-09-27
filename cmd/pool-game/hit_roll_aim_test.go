package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 群組 16 的 `2Fh`（overlay-12 entry 44 `1208h`，spec 112，#90）：矮人與侏儒被大型類人
// 打的 −4，條件是**被打的人自己正在打的那一個**（runtime `+0Ah`）叫 OGRE／TROLL／…。
// `+0Ah` 由攻擊包裝 overlay-13 `1883h` 在每一次出手時寫（`1929h`），所以隊員先 A 打過
// 那隻食人魔，食人魔回手時才減。全部從 Update() 按 A。
func TestGiantClassPenaltyFollowsTheDefendersOwnTarget(t *testing.T) {
	run := func(t *testing.T, name string, defenderAttacksFirst bool) bool {
		t.Helper()
		application, state := sideEffectBoard(t, int(gamepack.ClassSlotCleric), 1, gamepack.SpellIDBless, 13)
		state.Roster[3] = combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1}
		withEffect(2, 0x2f, 1)(state)
		targetKind(3, 1, 2, name)(state)
		if defenderAttacksFirst {
			attackWithKeys(t, application, state, 2, 3)
			if aim := state.targetAimName(2); aim != name {
				t.Fatalf("the attack wrapper did not record the defender's target: %q", aim)
			}
		}
		before := state.HitPoints[2]
		attackWithKeys(t, application, state, 3, 2)
		return state.HitPoints[2] < before
	}
	// 命中骰 13：AC 內部值 50、THAC0 40，調到 10 才中；−4 之後是 9。
	if !run(t, "OGRE", false) {
		t.Fatal("control: the ogre should hit a dwarf that is not fighting it")
	}
	if run(t, "OGRE", true) {
		t.Fatal("the dwarf fighting the ogre should get −4 on the ogre's roll")
	}
	// 名字不在 `DS:0416h` 那張表上：照常命中。
	if !run(t, "BUGBEAR", true) {
		t.Fatal("a large humanoid off the giant table should not be penalised")
	}
}
