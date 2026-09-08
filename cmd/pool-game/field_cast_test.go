package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func fieldCastApp(members ...poolsave.Character) *app {
	return &app{mode: modeAdventure, introDone: true, keys: scriptedKeys{},
		state: poolsave.State{Schema: poolsave.Schema, Party: members}}
}

func healer(name string, hp, max int, spell uint8) poolsave.Character {
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotCleric] = 6
	member := poolsave.Character{Name: name, CurrentHP: hp, MaxHP: max,
		ClassLevels: levels, Memorised: make([]uint8, gamepack.MemorisedSpellSlots)}
	if spell != 0 {
		member.Memorised[0] = spell
	}
	return member
}

// 探索時挑到沒記法術的人，原版會說 `<名字> has no spells memorized`（spec 119
// 的 `047Fh`）。那是正常回應，不是錯誤——安靜地什麼都不做會讓玩家以為按鍵沒吃到。
func TestFieldCastSaysWhenTheChosenCharacterHasNoSpells(t *testing.T) {
	application := fieldCastApp(healer("A", 5, 10, 0))
	application.openFieldCast()
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(application.fieldCastMessage, "A") {
		t.Fatalf("訊息是 %q，應該說出是誰沒有法術", application.fieldCastMessage)
	}
	if application.fieldCastStage != fieldCastPickCaster {
		t.Errorf("沒有法術卻走到第 %d 步", application.fieldCastStage)
	}
}

// 治療類在探索時就算得出來（原版走的是與戰鬥同一支派發，spec 119）。
func TestFieldCastHealsTheChosenTarget(t *testing.T) {
	effect := gamepack.CastEffect{Heal: 6}
	hurt := healer("B", 3, 20, 0)
	if !applyFieldEffect(&hurt, effect) {
		t.Fatal("治療沒有作用")
	}
	if hurt.CurrentHP != 9 {
		t.Errorf("治療後 %d 點，預期 9", hurt.CurrentHP)
	}
	// 不能超過上限。
	full := healer("C", 19, 20, 0)
	applyFieldEffect(&full, gamepack.CastEffect{Heal: 6})
	if full.CurrentHP != 20 {
		t.Errorf("治療超過上限：%d", full.CurrentHP)
	}
	// 沒有可結算的效果就要回 false，讓畫面說「這條要在戰鬥中才有目標」，
	// 而不是假裝施出去了。
	if applyFieldEffect(&full, gamepack.CastEffect{Damage: 8}) {
		t.Error("傷害法術在戰鬥外不該算成功")
	}
}

// C 開得起來，而且開著的時候底下的指令列不畫（panelOpen）。
func TestFieldCastOpensWithCAndCountsAsAPanel(t *testing.T) {
	application := fieldCastApp(healer("A", 5, 10, gamepack.SpellIDCureLightWound))
	application.openFieldCast()
	if !application.fieldCastOpen {
		t.Fatal("開不起來")
	}
	if !application.panelOpen() {
		t.Fatal("開著卻不算面板，底下的指令列會露出來")
	}
	if err := press(application, ebiten.KeyEscape); err != nil {
		t.Fatal(err)
	}
	if application.fieldCastOpen {
		t.Fatal("ESC 沒有關掉")
	}
}

// `V` 開得起來，先挑人再看那個人的資料頁（spec 119 → spec 130）。
func TestViewSheetPicksAMemberThenShowsTheSheet(t *testing.T) {
	application := fieldCastApp(healer("A", 5, 10, 0), healer("B", 8, 12, 0))
	application.openViewSheet()
	if !application.viewSheetOpen || application.viewSheetShown {
		t.Fatal("開起來應該停在挑人那一步")
	}
	if !application.panelOpen() {
		t.Fatal("開著卻不算面板，底下的指令列會露出來")
	}
	if got := application.screenName(); got != "view-pick" {
		t.Fatalf("挑人那一步回報 %q", got)
	}
	if err := press(application, ebiten.KeyArrowDown); err != nil {
		t.Fatal(err)
	}
	if application.viewSheetCursor != 1 {
		t.Fatalf("下鍵沒有換人，游標停在 %d", application.viewSheetCursor)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !application.viewSheetShown || application.screenName() != "view-sheet" {
		t.Fatalf("Enter 之後回報 %q", application.screenName())
	}
	// ESC 先回挑人那一步，再按一次才關掉——兩步都要回得去。
	if err := press(application, ebiten.KeyEscape); err != nil {
		t.Fatal(err)
	}
	if application.viewSheetShown || !application.viewSheetOpen {
		t.Fatal("第一次 ESC 應該回到挑人那一步")
	}
	if err := press(application, ebiten.KeyEscape); err != nil {
		t.Fatal(err)
	}
	if application.viewSheetOpen {
		t.Fatal("第二次 ESC 沒有關掉")
	}
}
