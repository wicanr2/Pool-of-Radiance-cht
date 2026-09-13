package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCaptureCampFireFrameIgnoresAnimationTick(t *testing.T) {
	first := &ebiten.Image{}
	second := &ebiten.Image{}
	fixed := 1
	application := &app{
		campOpen:             true,
		campFire:             []*ebiten.Image{first, second},
		campFireTick:         300,
		captureCampFireFrame: &fixed,
	}

	if got := application.campFireImage(); got != second {
		t.Fatal("捕捉模式沒有固定在指定的營火圖格")
	}
	application.campFireTick++
	if got := application.campFireImage(); got != second {
		t.Fatal("捕捉模式仍受動畫 tick 影響")
	}
}

func TestNormalCampFireStillAnimates(t *testing.T) {
	first := &ebiten.Image{}
	second := &ebiten.Image{}
	application := &app{
		campOpen: true,
		campFire: []*ebiten.Image{first, second},
	}

	if got := application.campFireImage(); got != first {
		t.Fatal("正常模式的第一張營火圖格錯誤")
	}
	application.campFireTick = campFireTicksPerFrame
	if got := application.campFireImage(); got != second {
		t.Fatal("捕捉參數影響了正常營火動畫")
	}
}
