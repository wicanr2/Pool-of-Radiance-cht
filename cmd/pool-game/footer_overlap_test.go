package main

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 畫面最底下那一列**只能有一段字**。兩段畫在同一條基線上會疊成一團，而那是
// 在截圖上才看得出來的缺陷——單元測試看不到像素，所以這裡數的是「有幾個
// drawText 落在那一條」。
func TestFooterRowHasOneSegment(t *testing.T) {
	initialMap := gamepack.GeometryMap{}
	walls := graphics.PieceSet{}
	cases := []struct {
		name  string
		build func() *app
	}{
		{"導覽的一頁", func() *app {
			return &app{
				mode: modeAdventure, spawn: gamepack.Spawn{X: 15, Y: 1, Facing: 3},
				introWaiting: true,
				initialEvent: &gamepack.InitialEvent{
					Message: "GREETINGS", ContinueLabel: "PRESS <RETURN> OR BUTTON TO CONTINUE"},
				initialMap: &initialMap, initialWalls: &walls,
			}
		}},
		{"格子事件的文字", func() *app {
			return &app{
				mode: modeAdventure, spawn: gamepack.Spawn{X: 3, Y: 4, Facing: 1},
				cellEventPending: true, introDone: true,
				eventText:  "PROCLAMATIONS ARE POSTED ON THE WALLS.",
				eventLabel: "PRESS <RETURN> OR BUTTON TO CONTINUE",
				initialMap: &initialMap, initialWalls: &walls,
			}
		}},
	}
	for _, item := range cases {
		application := item.build()
		var footer []string
		drawnText = func(value string, x, y int) {
			// **不能只比對相等**：兩段字基線差幾個像素一樣會疊成一團，而那正是
			// 這一列出過的問題。
			if y > footerBaseline-16 && value != "" {
				footer = append(footer, fmt.Sprintf("%q@%d,%d", value, x, y))
			}
		}
		screen := ebiten.NewImage(logicalWidth, logicalHeight)
		drawAdventure(screen, application, color.RGBA{255, 255, 255, 255}, color.RGBA{255, 255, 0, 255})
		drawnText = nil
		if len(footer) > 1 {
			t.Errorf("%s：最底下那一列畫了 %d 段 %q，只能有一段", item.name, len(footer), footer)
		}
	}
}
