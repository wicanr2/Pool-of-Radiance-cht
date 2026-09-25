package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 瞄準層的模式 8（射線）與 0Ah（以一點收再分邊），spec 098，issue #78。
// 玩家那一側從 Update() 送鍵；AI 那一側走 foeReleaseSpell，與玩家同一支 castSpell。

// castAndAimAtIndex 按 C、ENTER 選第一格記憶；施法時間不為 0 就再按一次 ENTER
// 讓它在輪到時放出去（overlay-08 `031Bh`）。瞄準開了之後 N 換到 index 那一個，
// ENTER 選定。
func castAndAimAtIndex(t *testing.T, application *app, state *tacticalState, index uint8) {
	t.Helper()
	pressAll(t, application, ebiten.KeyC, ebiten.KeyEnter)
	if state.Casting.Pending[1] != 0 {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if !application.castTargeting || application.castAim == nil || !application.castAim.plan.PointAim {
		t.Fatalf("the spell did not open point aiming: aiming %v status %q",
			application.castTargeting, state.Status)
	}
	for guard := 0; guard <= len(application.castTargets); guard++ {
		if application.castTargets[application.castTargetCursor] == index {
			break
		}
		pressAll(t, application, ebiten.KeyN)
	}
	if application.castTargets[application.castTargetCursor] != index {
		t.Fatalf("aiming never reached combatant %d: %v", index, application.castTargets)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if application.castTargeting || application.castAim != nil {
		t.Fatalf("aiming did not release the spell: %q", state.Status)
	}
}

// 閃電束（`2B75h`）：先打瞄準的那一格，再從那一格沿「施法者 → 瞄準」的方向拉射線，
// 停在每一個新的人身上（`2919h`）。長度 8 用完就停，所以遠處那一個打不到；
// 不在線上的人不受影響。fixedRoller{1}：Roll(5, 6) 是 1、豁免擲 1 必敗。
func TestLightningBoltStrikesAlongTheRay(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDLightningBolt),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 11, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 20, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 7, FootprintClass: 1})
	castAndAimAtIndex(t, application, state, 2)
	for index, want := range map[int]int{2: 29, 3: 29, 4: 30, 5: 30} {
		if state.HitPoints[index] != want {
			t.Errorf("combatant %d has %d hp after the bolt, want %d", index, state.HitPoints[index], want)
		}
	}
	if application.state.Party[0].Memorised[0] != 0 {
		t.Fatal("the bolt is still memorised")
	}
	if !strings.Contains(strings.ToUpper(state.Status), "HITS 2") {
		t.Fatalf("status %q", state.Status)
	}
}

// 牆（格位類別 FFh）把射線擋回來（`28B1h`、`2ACCh`）：回程從牆那一格往施法者那邊拉，
// 瞄準的那一個再挨一次；牆後的人打不到。
func TestLightningBoltBouncesOffAWall(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 5, gamepack.SpellIDLightningBolt),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 13, Y: 5, FootprintClass: 1})
	const wall = 6
	state.Classes[wall] = gamepack.CombatCellClass{EntryThreshold: combat.SpellRayWallClass}
	state.Grid.Terrain[5*combat.TacticalRowStride+11] = wall
	castAndAimAtIndex(t, application, state, 2)
	if state.HitPoints[2] != 28 {
		t.Errorf("the target has %d hp; the rebound should hit it a second time (28)", state.HitPoints[2])
	}
	if state.HitPoints[3] != 30 {
		t.Errorf("the foe behind the wall has %d hp", state.HitPoints[3])
	}
	if state.HitPoints[1] != 30 {
		t.Errorf("the caster has %d hp; the rebound runs out before reaching it", state.HitPoints[1])
	}
}

// blessBoard：施法者 1 在 (5,5)；隊友 2 在 (8,5)（瞄準點）、隊友 3 在 (10,5)
// （範圍內，但對面 4 在 (11,4) 貼著他）、隊友 5 在 (8,9)（範圍外）；對面 6 在 (8,7)
// （範圍內）。範圍是 `220Fh` 以 (8,5) 為中心、預算 2。
func blessBoard(t *testing.T, spell uint8) (*app, *tacticalState) {
	t.Helper()
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotCleric), 1, spell),
		combat.CombatantCell{X: 5, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 10, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 11, Y: 4, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 9, FootprintClass: 1},
		combat.CombatantCell{X: 8, Y: 7, FootprintClass: 1})
	for _, ally := range []int{2, 3, 5} {
		state.Friendly[ally] = true
	}
	return application, state
}

// 祝福術（`0FF5h` → `0F35h`）：`20AEh` 收到的表裡只留施法者那一邊，而且剔掉旁邊一步內
// 有對面的人（`0F81h`）。所以只有隊友 2：隊友 3 貼著敵人、隊友 5 在範圍外、對面 6 是
// 另一邊。
func TestBlessKeepsOnlyUnengagedAlliesInTheArea(t *testing.T) {
	application, state := blessBoard(t, gamepack.SpellIDBless)
	area, err := state.spellAreaMembers(8, 5, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []uint8{2, 3, 6} {
		found := false
		for _, index := range area {
			found = found || index == want
		}
		if !found {
			t.Fatalf("the board is not what the test assumes: area %v lacks %d", area, want)
		}
	}
	castAndAimAtIndex(t, application, state, 2)
	if !strings.Contains(strings.ToUpper(state.Status), "OVER 1 ALLIES") {
		t.Fatalf("bless should keep only ally 2: status %q", state.Status)
	}
	// 反例：拿掉貼著隊友 3 的敵人，隊友 3 就收得到。
	application, state = blessBoard(t, gamepack.SpellIDBless)
	state.Roster[4] = combat.CombatantCell{X: 20, Y: 20, FootprintClass: 1}
	castAndAimAtIndex(t, application, state, 2)
	if !strings.Contains(strings.ToUpper(state.Status), "OVER 2 ALLIES") {
		t.Fatalf("with no foe next to ally 3 bless should keep 2 and 3: status %q", state.Status)
	}
}

// 詛咒術（`1026h` → `0F35h`）推的是 23F5h(施法者)：同一個點、同一張表，只留對面那一個。
func TestCurseKeepsOnlyTheOtherSide(t *testing.T) {
	application, state := blessBoard(t, gamepack.SpellIDCurse)
	castAndAimAtIndex(t, application, state, 2)
	if !strings.Contains(strings.ToUpper(state.Status), "OVER 1 ALLIES") {
		t.Fatalf("curse should keep only foe 6: status %q", state.Status)
	}
}

// AI 施閃電束走同一層：`20AEh` 的 AI 那一支擲到最近的隊員，射線由牠穿過那一格
// 往外拉，後面那一個隊員也挨到；不在線上的不受影響。
func TestFoeLightningBoltUsesTheSameRay(t *testing.T) {
	application, state := newSpellBoard(t,
		spellCasterWith(int(gamepack.ClassSlotMagicUser), 3),
		combat.CombatantCell{X: 9, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 12, Y: 5, FootprintClass: 1},
		combat.CombatantCell{X: 12, Y: 8, FootprintClass: 1},
		combat.CombatantCell{X: 15, Y: 5, FootprintClass: 1})
	state.Friendly[2], state.PartySlot[2] = true, 1
	application.state.Party = append(application.state.Party, spellCasterWith(int(gamepack.ClassSlotFighter), 3))
	state.AIDriven = aiDriven(state.PartySlot)
	var record gamepack.MonsterRecord
	record.Name = "MAGE"
	record.Raw[gamepack.AISpellArrayOffset] = gamepack.SpellIDLightningBolt
	record.Raw[0x9B] = 5
	state.rememberSpellbook(4, record)
	state.Mover = 4
	if err := application.foeReleaseSpell(state, 4, gamepack.SpellIDLightningBolt); err != nil {
		t.Fatal(err)
	}
	for index, want := range map[int]int{1: 29, 2: 29, 3: 30, 4: 30} {
		if state.HitPoints[index] != want {
			t.Errorf("combatant %d has %d hp after the foe's bolt, want %d", index, state.HitPoints[index], want)
		}
	}
	if state.Casting.Spells[4][0] != 0 {
		t.Fatal("the foe still has the bolt memorised")
	}
}
