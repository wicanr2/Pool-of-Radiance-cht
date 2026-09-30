package main

import (
	"image/color"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
)

// 女性角色一到資料頁，右上角就是女性預設頭像；退回陣營那一頁，
// 預設要清掉重算（#130）。全程從 Update() 進去。
func TestRollPageShowsGenderDefaultPortrait(t *testing.T) {
	flow := creation.Flow{Stage: creation.StageRoll, GenderIndex: 1}
	for index, race := range creation.Races {
		if race.ID == "human" {
			flow.RaceIndex = index
		}
	}
	for index, choice := range creation.ClassesForRace("human") {
		if choice.ID == "fighter" {
			flow.ClassIndex = index
		}
	}
	a := &app{
		mode:   modeCreation,
		flow:   flow,
		roller: diceRoller{random: rand.New(rand.NewSource(1))},
		loadPortrait: func(head, body uint8) (*ebiten.Image, error) {
			image := ebiten.NewImage(88, 88)
			image.Fill(color.RGBA{head, body, 0, 255})
			return image, nil
		},
	}
	a.keys = scriptedKeys{}
	if err := a.Update(); err != nil {
		t.Fatal(err)
	}
	if a.flow.PortraitHead != 6 || a.flow.PortraitBody != 7 {
		t.Fatalf("roll page portrait %d/%d, want female fighter 6/7", a.flow.PortraitHead, a.flow.PortraitBody)
	}
	if err := press(a, ebiten.KeyEscape); err != nil {
		t.Fatal(err)
	}
	if a.flow.Stage != creation.StageAlignment || a.flow.PortraitHead != 0 {
		t.Fatalf("back to alignment: stage %d head %d, want alignment and a cleared default", a.flow.Stage, a.flow.PortraitHead)
	}
}
