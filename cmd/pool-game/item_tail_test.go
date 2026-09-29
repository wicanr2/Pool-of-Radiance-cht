package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 物品頁與負重的尾巴（issue #114，spec 149／035／144）。全部從 Update() 送鍵。

// alignedSwordF0 是 ITEM4.DAX block 29 第 6 件：`+3Dh` F0h（限守序善良，錯了受 15 點）。
func alignedSwordF0(t *testing.T) poolsave.Item {
	t.Helper()
	good, err := gamepack.ReadDOSTreasureItemBlock(filepath.Join("..", "..", "Pool of Radiance (1988).zip"), 4, 29)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	record := good[6]
	if record.Raw[gamepack.ItemEffectOffset] != gamepack.AlignedWearEffectCode {
		t.Fatalf("%s has +3Eh %#02x", record.Name, record.Raw[gamepack.ItemEffectOffset])
	}
	return poolsave.Item{Name: record.Name, Raw: append([]byte(nil), record.Raw[:]...)}
}

// 戰鬥外 84h（overlay-24 entry 19）：受傷那一句印完停一拍（entry 26 `21C1h`），倒下那一句
// 再停一拍（`1602h`），停完 entry 21 清掉；停拍裡不收鍵。
func TestAlignedSwordStopsABeatPerSentence(t *testing.T) {
	wearer := tradeMember("CHAOS", alignedSwordF0(t))
	wearer.AlignmentID, wearer.CurrentHP = "chaotic-good", 10
	application := newItemPageApp(t, wearer)
	application.gameSpeed = 2
	beat := application.speedDelayTicks()
	pressAll(t, application, ebiten.KeyR)
	if application.equipment.page.stage != itemPageWearBeat ||
		application.equipment.message != "CHAOS TAKES 15 POINTS OF DAMAGE FROM MAGIC" {
		t.Fatalf("after R: stage %d, %q", application.equipment.page.stage, application.equipment.message)
	}
	frames := 1
	pressAll(t, application, ebiten.KeyEscape)
	for guard := 0; guard < 600 && application.equipment.message == "CHAOS TAKES 15 POINTS OF DAMAGE FROM MAGIC"; guard++ {
		idleFrame(t, application)
		frames++
	}
	if frames != beat || application.equipment.message != "CHAOS GOES DOWN, AND IS DYING" {
		t.Fatalf("first beat lasted %d frames (want %d), then %q", frames, beat, application.equipment.message)
	}
	frames = 0
	for guard := 0; guard < 600 && application.equipment.page.stage == itemPageWearBeat; guard++ {
		idleFrame(t, application)
		frames++
	}
	if frames != beat || application.equipment.message != "" || !application.equipmentOpen {
		t.Fatalf("second beat lasted %d frames (want %d), message %q, open %v", frames, beat,
			application.equipment.message, application.equipmentOpen)
	}
	if member := application.state.Party[0]; member.Status != gamepack.DyingState ||
		member.Inventory[0].Raw[gamepack.ItemReadiedOffset] != 0 {
		t.Fatalf("after the beats: %+v", member)
	}

	// 沒倒下只有一拍。
	wearer.CurrentHP = 20
	application = newItemPageApp(t, wearer)
	application.gameSpeed = 2
	pressAll(t, application, ebiten.KeyR)
	frames = 0
	for guard := 0; guard < 600 && application.equipment.page.stage == itemPageWearBeat; guard++ {
		idleFrame(t, application)
		frames++
	}
	if frames != beat || application.equipment.message != "" || application.state.Party[0].CurrentHP != 5 {
		t.Fatalf("one beat: %d frames (want %d), %q, hp %d", frames, beat, application.equipment.message,
			application.state.Party[0].CurrentHP)
	}
}

// 戰鬥中 84h：entry 26 戰鬥那一路（entry 20 旗標 0、閃光、`21BAh` 等一拍）、倒下那一句之後
// `1609h` 的 `1004h` 摘掉十六個戰鬥效果（最後一個是 `DS:0C37h` 的 4Bh），倒下動畫在
// overlay-32 `1006h` 再等一拍。兩則都排進戰鬥訊息佇列，名字由 entry 20 印在上一列。
func TestAlignedSwordInCombatStopsTwoBeatsAndStripsEffects(t *testing.T) {
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5, alignedSwordF0(t))
	application.gameSpeed = 2
	application.state.Party[0].AlignmentID = "chaotic-good"
	state.HitPoints[1] = 10
	state.Effects[1] = gamepack.EffectList{
		gamepack.NewEffectNode(0x4B, 0, 1, false),
		gamepack.NewEffectNode(0x33, 3, 1, false),
		gamepack.NewEffectNode(0x3D, 0, 12, false),
	}
	pressAll(t, application, ebiten.KeyU, ebiten.KeyR)
	// 三則：受傷那一句（閃光一輪 4 × 70 毫秒再等一拍，spec 166）、倒下那一句（entry 20 旗標 0，
	// 字留在右欄）、倒下動畫（骷髏，`1006h` 的一拍，字照樣留著）。
	if len(state.Notices) != 3 {
		t.Fatalf("notices %+v", state.Notices)
	}
	beat := application.speedDelayTicks()
	hurt, down, skull := state.Notices[0], state.Notices[1], state.Notices[2]
	if hurt.Name != "A" || hurt.Text != "TAKES 15 POINTS OF DAMAGE FROM MAGIC" ||
		hurt.Ticks != millisecondsToTicks(4*0x46)+beat || hurt.Anim == nil || hurt.Anim.Kind != animationSparkle ||
		down.Name != "A" || down.Text != "GOES DOWN, AND IS DYING" || down.Ticks != 0 ||
		skull.Anim == nil || skull.Anim.Kind != animationSkull || skull.Ticks != beat || skull.Text != down.Text {
		t.Fatalf("notices %+v %+v %+v, beat %d", hurt, down, skull, beat)
	}
	if state.HitPoints[1] != 0 || state.States[1] != gamepack.DyingState || state.Roster[1].FootprintClass != 0 {
		t.Fatalf("board after the sword: hp %d state %d footprint %d", state.HitPoints[1], state.States[1],
			state.Roster[1].FootprintClass)
	}
	if state.hasEffect(1, 0x4B) || state.hasEffect(1, 0x33) || !state.hasEffect(1, 0x3D) {
		t.Fatalf("1004h left %v", state.Effects[1])
	}
	// 停拍裡 tacticalInput 不做別的事：一影格扣一格。
	idleFrame(t, application)
	if len(state.Notices) != 3 || state.Notices[0].Ticks != hurt.Ticks-1 {
		t.Fatalf("hold: %+v", state.Notices)
	}
}

// 戰鬥中的 'U' 開的也是 overlay-19 entry 6（overlay-08 `03F2h`），`0F97h` 的反魔法門
// （`@49E5`）在 `DS:4954h` 分派之前：反魔法區裡選項列沒有 USE、U 不做事。
func TestCombatItemUseClosedInsideAntiMagic(t *testing.T) {
	wand := itemOf("WAND", testTypeSword, true, 0)
	wand.Raw[gamepack.AIItemChargesOffset] = 2
	wand.Raw[gamepack.AIItemSpellOffset] = 18
	application, state := newItemMenuApp(t, int(gamepack.ClassSlotFighter), 5, wand)
	application.eventMachine = &eclvm.Machine{Memory: map[uint16]uint16{antiMagicAddress: 1}}
	pressAll(t, application, ebiten.KeyU)
	if footer := application.combatItemFooter(state, 0); footer != "READY DROP HALVE JOIN EXIT" {
		t.Fatalf("anti-magic footer %q", footer)
	}
	pressAll(t, application, ebiten.KeyU)
	if charges := application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset]; charges != 2 ||
		strings.Contains(state.Status, "casts") {
		t.Fatalf("U inside anti-magic: charges %d, %q", charges, state.Status)
	}
	application.eventMachine.Memory[antiMagicAddress] = 0
	if footer := application.combatItemFooter(state, 0); footer != "READY USE DROP HALVE JOIN EXIT" {
		t.Fatalf("footer after the shell is gone %q", footer)
	}
	pressAll(t, application, ebiten.KeyU)
	if application.castTargeting {
		pressAll(t, application, ebiten.KeyEnter)
	}
	if charges := application.state.Party[0].Inventory[0].Raw[gamepack.AIItemChargesOffset]; charges != 1 {
		t.Fatalf("U outside anti-magic: charges %d, %q", charges, state.Status)
	}
}

// 拾取（overlay-06 entry 2 `0244h` → overlay-19 entry 9 `274Fh`）：`+102h` 含錢。力量 10
// 上限 1500，身上 1400 枚錢時 101 重的收不下，100 重的收得下；錢丟到 1300 就收得下。
func TestTreasurePickupCountsCoinsAgainstTheLoad(t *testing.T) {
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", Abilities: [6]int{10, 10, 10, 10, 10, 10}, MaxHP: 8, CurrentHP: 8,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1, Money: [7]uint16{3: 1400}}
	application := &app{mode: modeAdventure, introDone: true, cellEventPending: true, cellWaitingMenu: true,
		state: poolsave.State{Schema: poolsave.Schema, CharacterLibrary: []poolsave.Character{character},
			Party: []poolsave.Character{character}},
		treasureActive: true, treasureStage: treasureCharacter, cellMenuOptions: []string{"HERO", "Cancel"},
		treasureItems: []gamepack.TreasureItemRecord{treasureRecord("Heavy", 101)}}
	application.saveState = func(poolsave.State) error { return nil }
	pressAll(t, application, ebiten.KeyEnter)
	if len(application.state.Party[0].Inventory) != 0 || application.eventText != "OverLoaded" {
		t.Fatalf("1400 coins + 101: inventory %d, %q", len(application.state.Party[0].Inventory), application.eventText)
	}

	application.state.Party[0].Money[3] = 1300
	application.treasureStage, application.cellMenuCursor = treasureCharacter, 0
	application.cellMenuOptions = []string{"HERO", "Cancel"}
	pressAll(t, application, ebiten.KeyEnter)
	if len(application.state.Party[0].Inventory) != 1 {
		t.Fatalf("1300 coins + 101: inventory %d, %q", len(application.state.Party[0].Inventory), application.eventText)
	}
}
