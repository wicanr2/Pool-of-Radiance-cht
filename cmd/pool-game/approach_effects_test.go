package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 石化的隊員（狀態 7，`005Ah` 清掉 `+10Dh`）下一場只擺屍體，不上場。
func TestStonedMemberIsDeployedAsACorpse(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	const stoned = 1
	application.state.Party = append([]poolsave.Character(nil), application.state.Party...)
	application.state.Party[stoned].Status, application.state.Party[stoned].CurrentHP = gamepack.StonedState, 0
	application.tactical = nil
	if err := application.enterTacticalPreview(); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	index := npcBoardIndex(t, state, stoned)
	if state.Roster[index].FootprintClass != 0 || state.States[index] != gamepack.StonedState {
		t.Fatalf("the stoned member stands with footprint %d state %d", state.Roster[index].FootprintClass,
			state.States[index])
	}
}

// 群組 0Eh 其餘三個碼（spec 161，#82）：輪到帶著它的怪物時，overlay-09 entry 5 開場先問。
// 全部從 Update() 送 ENTER，輪到敵方時 tacticalInput 自己分派 foeTurn。
//
// 盤面是 newFoeCastApp：隊員（1，隊伍第 0 人）在 (5,5)，敵人（2）在 (10,5)；fixedRoller{7}
// 的 d20 過不了豁免（目標值是最差的那一格）。

func newApproachApp(t *testing.T, codes ...uint8) (*app, *tacticalState) {
	t.Helper()
	application, state := newFoeCastApp(t)
	// 豁免目標值一律最差的那一格（自然 20 才過），與開打時沒有記錄可讀的預設相同。
	state.SaveTargets = make([][gamepack.SavingThrowCategories]uint8, len(state.Roster))
	state.SaveBonus = make([]int, len(state.Roster))
	for index := range state.SaveTargets {
		for category := range state.SaveTargets[index] {
			state.SaveTargets[index][category] = gamepack.SavingThrowWorstTarget
		}
	}
	for _, code := range codes {
		state.Effects[2] = state.Effects[2].Append(gamepack.NewEffectNode(code, 0, 0xff, false))
	}
	return application, state
}

// mirrorIn 讓第 holder 格手上拿著一面鏡子（名稱字詞 `76h`），readied 決定 `+34h`。
func mirrorIn(state *tacticalState, holder int, readied bool) {
	mirror := make([]byte, gamepack.MonsterItemRecordSize)
	mirror[gamepack.ItemNameWordOffset+1] = gamepack.MirrorNameWord
	if readied {
		mirror[gamepack.ItemReadiedOffset] = 1
	}
	state.ItemsOf = func(index int) [][]byte {
		if index == holder {
			return [][]byte{mirror}
		}
		return nil
	}
}

// 石化凝視（`53h`）：豁免沒過的目標狀態 7、下場、生命 0，隊員那一份也跟著改；目標沒了就重挑，
// 挑不到收工。
func TestBasiliskGazeTurnsTheTargetToStone(t *testing.T) {
	application, state := newApproachApp(t, gamepack.GazeReflectableEffectCode, gamepack.GazeStoneEffectCode)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.States[1] != gamepack.StonedState || state.Roster[1].FootprintClass != 0 || state.HitPoints[1] != 0 {
		t.Fatalf("target state %d footprint %d hp %d; want stoned and off the board (log %q)",
			state.States[1], state.Roster[1].FootprintClass, state.HitPoints[1], state.FoeLog)
	}
	if member := application.state.Party[0]; member.Status != gamepack.StonedState || member.CurrentHP != 0 {
		t.Fatalf("party record status %d hp %d; want 7 and 0", member.Status, member.CurrentHP)
	}
	if state.States[2] != 0 {
		t.Fatalf("the basilisk stoned itself: %d", state.States[2])
	}
}

// 目標手上裝備著鏡子（`+34h` 非 0、字詞 76h）而凝視者帶 `7Fh`：反射回去，凝視者自己擲豁免、
// 自己變石頭。鏡子沒裝備就照常石化目標。
func TestMirrorReflectsTheGazeOnlyWhenReadied(t *testing.T) {
	application, state := newApproachApp(t, gamepack.GazeReflectableEffectCode, gamepack.GazeStoneEffectCode)
	mirrorIn(state, 1, true)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.States[2] != gamepack.StonedState || state.States[1] != 0 {
		t.Fatalf("reflected gaze: basilisk %d target %d; want 7 and 0", state.States[2], state.States[1])
	}

	application, state = newApproachApp(t, gamepack.GazeReflectableEffectCode, gamepack.GazeStoneEffectCode)
	mirrorIn(state, 1, false)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.States[1] != gamepack.StonedState || state.States[2] != 0 {
		t.Fatalf("mirror in the pack: basilisk %d target %d; want 0 and 7", state.States[2], state.States[1])
	}
}

// 凝視不結束行動：目標擲過豁免（自然 20）之後，凝視者照樣進接近迴圈往前走。
func TestGazeDoesNotEndTheAction(t *testing.T) {
	application, state := newApproachApp(t, gamepack.GazeReflectableEffectCode, gamepack.GazeStoneEffectCode)
	application.roller = fixedRoller{20}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.States[1] != 0 {
		t.Fatalf("a natural 20 still stoned the target: %d", state.States[1])
	}
	if state.Roster[2].X >= 10 {
		t.Fatalf("the basilisk stopped after gazing: %+v (log %q)", state.Roster[2], state.FoeLog)
	}
}

// 魅惑凝視（`54h`）：過了 `1087h` 與視線，目標擲豁免（−2）沒過就掛 `0Bh`、倒戈到吸血鬼那一邊、
// 交給 AI。目標手上的鏡子讓帶 `7Eh` 的吸血鬼根本不凝視（`1087h`）。
func TestVampireGazeCharmsUnlessTheTargetHoldsAMirror(t *testing.T) {
	application, state := newApproachApp(t, gamepack.GazeCharmEffectCode, vetoEffectTargetsItems)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.Friendly[1] || !state.aiDrives(1) || !state.hasEffect(1, gamepack.CharmPersonEffectCode) {
		t.Fatalf("target friendly %v ai %v effects %+v; want charmed onto the vampire's side",
			state.Friendly[1], state.aiDrives(1), state.Effects[1])
	}

	application, state = newApproachApp(t, gamepack.GazeCharmEffectCode, vetoEffectTargetsItems)
	mirrorIn(state, 1, true)
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !state.Friendly[1] || state.hasEffect(1, gamepack.CharmPersonEffectCode) {
		t.Fatalf("a readied mirror still let the vampire charm: friendly %v effects %+v",
			state.Friendly[1], state.Effects[1])
	}
}

// 視線是 `0419h`：中間有一格擋路的地形就不凝視。豁免過了也不掛。
func TestVampireGazeNeedsAClearLineAndAFailedSave(t *testing.T) {
	application, state := newApproachApp(t, gamepack.GazeCharmEffectCode)
	const wall = 9
	state.Classes[wall] = gamepack.CombatCellClass{EntryThreshold: 1, PathByte2: 1}
	state.Grid.Terrain[5*0x32+7] = wall
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !state.Friendly[1] || state.hasEffect(1, gamepack.CharmPersonEffectCode) {
		t.Fatalf("the gaze went through the wall: friendly %v effects %+v", state.Friendly[1], state.Effects[1])
	}

	application, state = newApproachApp(t, gamepack.GazeCharmEffectCode)
	application.roller = fixedRoller{20}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !state.Friendly[1] || state.hasEffect(1, gamepack.CharmPersonEffectCode) {
		t.Fatalf("a natural 20 was still charmed: friendly %v effects %+v", state.Friendly[1], state.Effects[1])
	}
}

// 噴酸（`79h`）：d100 不大於 25、距離小於 4 才噴；8d4（fixedRoller 每顆 4）、類別 3 豁免沒過整份
// 吃下；噴完摘掉 `79h` 與 `50h`，行動結束。
func TestAnkhegSpitsAcidOnceWithinThreeSquares(t *testing.T) {
	application, state := newApproachApp(t, gamepack.AcidBiteEffectCode, gamepack.AcidSpitEffectCode)
	state.Roster[2].X = 8
	before := state.Roster[2]
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[1] != 40-32 {
		t.Fatalf("party hp %d, want 40 − 8×4 (log %q)", state.HitPoints[1], state.FoeLog)
	}
	if state.hasEffect(2, gamepack.AcidSpitEffectCode) || state.hasEffect(2, gamepack.AcidBiteEffectCode) {
		t.Fatalf("the spit left %+v", state.Effects[2])
	}
	if state.Roster[2] != before || state.Activity.FoeAttacks != 0 {
		t.Fatalf("the spit did not end the action: %+v → %+v, attacks %d", before, state.Roster[2],
			state.Activity.FoeAttacks)
	}
}

// 距離 4 就不噴（`2C72h` 小於 4 才噴）；d100 大於 25 也不噴。兩種都照常接近、節點留著。
func TestAnkhegHoldsTheAcidOutOfReachOrOnAHighRoll(t *testing.T) {
	application, state := newApproachApp(t, gamepack.AcidBiteEffectCode, gamepack.AcidSpitEffectCode)
	state.Roster[2].X = 9
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !state.hasEffect(2, gamepack.AcidSpitEffectCode) || state.HitPoints[1] < 40-8 {
		t.Fatalf("four squares away: effects %+v party hp %d", state.Effects[2], state.HitPoints[1])
	}

	application, state = newApproachApp(t, gamepack.AcidBiteEffectCode, gamepack.AcidSpitEffectCode)
	state.Roster[2].X = 7
	application.roller = fixedRoller{26}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !state.hasEffect(2, gamepack.AcidSpitEffectCode) {
		t.Fatalf("d100 26 still spat: %+v", state.Effects[2])
	}
}
