package main

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
)

type scriptedKeys map[ebiten.Key]bool

func (keys scriptedKeys) JustPressed(key ebiten.Key) bool {
	pressed := keys[key]
	delete(keys, key)
	return pressed
}

func press(application *app, key ebiten.Key) error {
	application.keys = scriptedKeys{key: true}
	return application.Update()
}

func TestKeysDriveTitleToOriginalCharacterSheet(t *testing.T) {
	application := &app{
		mode:   modeTitle,
		flow:   creation.NewFlow(),
		roller: diceRoller{random: rand.New(rand.NewSource(1))},
	}
	for _, key := range []ebiten.Key{
		ebiten.KeyEnter, // title -> menu
		ebiten.KeyC,     // menu -> race
		ebiten.KeyEnter, // Dwarf
		ebiten.KeyEnter, // Male
		ebiten.KeyEnter, // Fighter
		ebiten.KeyEnter, // Lawful Good
	} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	// The roll page generates on its first update, matching the real frame path.
	application.keys = scriptedKeys{}
	if err := application.Update(); err != nil {
		t.Fatal(err)
	}
	if application.mode != modeCreation || application.flow.Stage != creation.StageRoll || application.rolled == nil {
		t.Fatalf("normal key path stopped at mode=%d stage=%d result=%v", application.mode, application.flow.Stage, application.rolled)
	}
}

func TestGlobalHelpThemeAndQuitKeys(t *testing.T) {
	application := &app{mode: modeMenu}
	if err := press(application, ebiten.KeyF2); err != nil || !application.modern {
		t.Fatalf("F2 theme: modern=%t err=%v", application.modern, err)
	}
	if err := press(application, ebiten.KeyF1); err != nil || !application.help {
		t.Fatalf("F1 help: help=%t err=%v", application.help, err)
	}
	application.help = false
	if err := press(application, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10=%v", err)
	}
}
