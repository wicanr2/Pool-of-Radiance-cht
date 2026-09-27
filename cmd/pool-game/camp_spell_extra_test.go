package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 營地施法的尾巴（spec 098〈營地施法〉，issue #108）。全部從 Update() 送鍵。

// pickFieldCastSpell 按 C、挑第 caster 個人、挑編號 spell 的那一條，停在按下去之後。
func pickFieldCastSpell(t *testing.T, application *app, caster int, spell uint8) {
	t.Helper()
	pressAll(t, application, ebiten.KeyC)
	for guard := 0; guard <= len(application.state.Party) && application.fieldCastCursor != caster; guard++ {
		pressAll(t, application, ebiten.KeyArrowDown)
	}
	pressAll(t, application, ebiten.KeyEnter)
	want := -1
	for index, option := range application.fieldCastOptions {
		if option.ID == spell {
			want = index
		}
	}
	if want < 0 {
		t.Fatalf("spell %d is not on the list %+v", spell, application.fieldCastOptions)
	}
	for guard := 0; guard <= len(application.fieldCastOptions) && application.fieldCastCursor != want; guard++ {
		pressAll(t, application, ebiten.KeyArrowDown)
	}
	pressAll(t, application, ebiten.KeyEnter)
}

func memorises(member poolsave.Character, spell uint8) bool {
	for _, value := range member.Memorised {
		if value&0x7f == spell {
			return true
		}
	}
	return false
}

// `+7` 為 0 的記憶法術（魔法飛彈）：`0C49h` 問 "Lose it?"，N 留著、Y 才從記憶清掉，
// 兩條路都不放出去（`0D1Bh`）。
func TestCampCombatOnlySpellAsksLoseIt(t *testing.T) {
	application := campCastApp(t, campCaster("A", 3, gamepack.SpellIDMagicMissile), campCaster("B", 1))
	pickFieldCastSpell(t, application, 0, gamepack.SpellIDMagicMissile)
	if application.fieldCastStage != fieldCastLoseIt ||
		!strings.Contains(application.fieldCastMessage, "combat-only") {
		t.Fatalf("stage %d message %q, want the Lose it? question", application.fieldCastStage,
			application.fieldCastMessage)
	}
	pressAll(t, application, ebiten.KeyN)
	if application.fieldCastStage != fieldCastPickCaster ||
		!memorises(application.state.Party[0], gamepack.SpellIDMagicMissile) {
		t.Fatalf("N: stage %d memorised %v", application.fieldCastStage, application.state.Party[0].Memorised)
	}
	pressAll(t, application, ebiten.KeyEscape)
	pickFieldCastSpell(t, application, 0, gamepack.SpellIDMagicMissile)
	pressAll(t, application, ebiten.KeyY)
	if memorises(application.state.Party[0], gamepack.SpellIDMagicMissile) {
		t.Fatal("Y did not lose the spell")
	}
	for index, member := range application.state.Party {
		if len(member.Effects) != 0 || member.CurrentHP != 20 {
			t.Fatalf("member %d changed after losing a combat-only spell: %+v", index, member)
		}
	}
}

func withSavingThrows(t *testing.T, application *app) {
	t.Helper()
	table, err := gamepack.ReadDOSSavingThrowTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Fatal(err)
	}
	application.savingThrows = table
}

// 縮小術（`135Eh`）：豁免沒過而且身上有變大術的 `0Ch`，經 entry 15 摘掉（收尾把力量還原）、
// 印 "has been reduced"；豁免過了（自然 20）就什麼都不做。
func TestCampReduceUndoesTheEnlargement(t *testing.T) {
	for _, saved := range []bool{false, true} {
		application := campCastApp(t, campCaster("A", 3, gamepack.SpellIDEnlarge, gamepack.SpellIDReduce),
			campCaster("B", 1))
		withSavingThrows(t, application)
		castInCamp(t, application, 0, gamepack.SpellIDEnlarge, 1)
		if _, ok := memberEffect(application.state.Party[1], gamepack.EnlargeEffectCode); !ok {
			t.Fatalf("enlarge attached nothing: %v", application.state.Party[1].Effects)
		}
		enlarged := application.state.Party[1].Abilities[gamepack.AbilityStrength]
		if saved {
			application.roller = fixedRoller{20}
		}
		message := castInCamp(t, application, 0, gamepack.SpellIDReduce, 1)
		_, still := memberEffect(application.state.Party[1], gamepack.EnlargeEffectCode)
		strength := application.state.Party[1].Abilities[gamepack.AbilityStrength]
		if saved {
			if !still || strength != enlarged {
				t.Fatalf("saved: enlargement gone %v strength %d (%q)", !still, strength, message)
			}
			continue
		}
		if still || strength != 12 || !strings.Contains(message, "reduced") {
			t.Fatalf("failed save: enlargement kept %v strength %d message %q", still, strength, message)
		}
	}
}

// 解除魔法（`2356h`）：第一格身上的效果逐個擲 1d100，過了就摘；沒挑到的人不動。
func TestCampDispelMagicStripsThePickedMember(t *testing.T) {
	application := campCastApp(t, campCaster("A", 5, gamepack.SpellIDBless, gamepack.SpellIDDispelMagic),
		campCaster("B", 1))
	castInCamp(t, application, 0, gamepack.SpellIDBless, 0)
	message := castInCamp(t, application, 0, gamepack.SpellIDDispelMagic, 1)
	if _, ok := memberEffect(application.state.Party[1], gamepack.BlessEffectCode); ok {
		t.Fatalf("B kept the blessing (%q)", message)
	}
	if _, ok := memberEffect(application.state.Party[0], gamepack.BlessEffectCode); !ok {
		t.Fatal("dispel on B also stripped A")
	}
}

// 恢復術（`2C01h`）：把能量吸取的欠帳還一級，與戰鬥中同一支 RestoreDrainedLevel。
func TestCampRestorationGivesALevelBack(t *testing.T) {
	drained := campCaster("B", 0)
	drained.ClassLevels[2] = 2
	drained.DrainedLevels, drained.DrainedHitPoints, drained.Experience = 1, 6, 5000
	application := campCastApp(t, campCaster("A", 13, gamepack.SpellIDRestoration), drained)
	application.experienceTable[2][2], application.experienceTable[2][3] = 2000, 4000
	message := castInCamp(t, application, 0, gamepack.SpellIDRestoration, 1)
	got := application.state.Party[1]
	if got.ClassLevels[2] != 3 || got.DrainedLevels != 0 || got.MaxHP != 26 {
		t.Fatalf("after restoration %+v (%q)", got, message)
	}
}

// 物品頁的 Use：`+7` 是 4（整隊）的不問對象，`0A88h` 直接收整隊（祝福的魔杖）。
func TestCampItemOfBlessCoversThePartyWithoutAPicker(t *testing.T) {
	wand := itemOf("WAND", testTypeSword, true, 0)
	wand.Raw[gamepack.AIItemChargesOffset] = 2
	wand.Raw[gamepack.AIItemSpellOffset] = gamepack.SpellIDBless
	member := campCaster("A", 3)
	member.Inventory = []poolsave.Item{wand}
	application := campCastApp(t, member, campCaster("B", 1))
	pressAll(t, application, ebiten.KeyI, ebiten.KeyU)
	if application.equipment.page.stage == itemPageTarget {
		t.Fatal("a whole-party item still asked Cast Spell on whom")
	}
	for index, member := range application.state.Party {
		if _, ok := memberEffect(member, gamepack.BlessEffectCode); !ok {
			t.Fatalf("member %d has no blessing (%q)", index, application.equipment.message)
		}
	}
}
