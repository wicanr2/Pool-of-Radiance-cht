package main

import (
	"errors"
	"image/color"
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

type scriptedTextKeys struct {
	scriptedKeys
	chars []rune
}

func (keys *scriptedTextKeys) JustPressed(key ebiten.Key) bool {
	return keys.scriptedKeys.JustPressed(key)
}
func (keys *scriptedTextKeys) Chars() []rune { result := keys.chars; keys.chars = nil; return result }

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

func TestKeysContinueThroughNameAndOriginalPortraitEditor(t *testing.T) {
	application := &app{
		mode:   modeCreation,
		flow:   creation.Flow{Stage: creation.StageRoll},
		rolled: &creation.RolledCharacter{},
		loadPortrait: func(head, body uint8) (*ebiten.Image, error) {
			image := ebiten.NewImage(88, 88)
			image.Fill(color.RGBA{uint8(head), uint8(body), 0, 255})
			return image, nil
		},
		loadIcon: func(head, body, size uint8, action bool, colors [6][2]uint8) (*ebiten.Image, error) {
			image := ebiten.NewImage(24, 24)
			image.Fill(color.RGBA{head, body, size, 255})
			return image, nil
		},
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.flow.Stage != creation.StageName {
		t.Fatalf("roll accept: stage=%d err=%v", application.flow.Stage, err)
	}
	application.keys = &scriptedTextKeys{scriptedKeys: scriptedKeys{}, chars: []rune("hero")}
	if err := application.Update(); err != nil {
		t.Fatal(err)
	}
	if application.nameInput != "HERO" {
		t.Fatalf("name input = %q", application.nameInput)
	}
	if err := press(application, ebiten.KeyEnter); err != nil || application.flow.Stage != creation.StagePortrait || application.portrait == nil {
		t.Fatalf("name accept: stage=%d portrait=%v err=%v", application.flow.Stage, application.portrait, err)
	}
	if err := press(application, ebiten.KeyH); err != nil || application.flow.PortraitHead != 2 {
		t.Fatalf("HEAD: %d err=%v", application.flow.PortraitHead, err)
	}
	if err := press(application, ebiten.KeyB); err != nil || application.flow.PortraitBody != 2 {
		t.Fatalf("BODY: %d err=%v", application.flow.PortraitBody, err)
	}
	if err := press(application, ebiten.KeyK); err != nil || application.flow.Stage != creation.StageIcon {
		t.Fatalf("KEEP: stage=%d err=%v", application.flow.Stage, err)
	}
	if application.iconReady == nil || application.iconAction == nil {
		t.Fatal("combat icon previews were not loaded")
	}
	if err := press(application, ebiten.KeyH); err != nil || application.flow.IconHead != 1 {
		t.Fatalf("icon HEAD: %d err=%v", application.flow.IconHead, err)
	}
	if err := press(application, ebiten.KeyW); err != nil || application.flow.IconWeapon != 1 {
		t.Fatalf("icon WEAPON: %d err=%v", application.flow.IconWeapon, err)
	}
	if err := press(application, ebiten.KeyP); err != nil || application.flow.IconPart != 1 {
		t.Fatalf("icon PART: %d err=%v", application.flow.IconPart, err)
	}
	before := application.flow.IconColors[1][0]
	if err := press(application, ebiten.KeyDigit1); err != nil || application.flow.IconColors[1][0] != (before+1)&0x0F {
		t.Fatalf("icon COLOR-1: %d err=%v", application.flow.IconColors[1][0], err)
	}
}
