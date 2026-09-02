package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

const (
	tacticalCellSize = 10
	tacticalLeft     = 70
	tacticalTop      = 76
)

// geoWallProbe 把 GEO 的牆面查詢接成戰術地圖生成器要的 probe。
// engine 的 Wall 用的就是 0／2／4／6 這組方向碼，與戰術層同一套，不必轉換。
//
// 原版的牆面值有 0、1、3 三種（spec 060），但 GEO 的牆面值是牆的樣式編號，
// 兩者之間的對應還沒解出來。這裡把所有非零一律當成 WallBlocking——
// 於是門目前會畫成實牆。這是刻意讓缺口看得見，不是猜一個對應。
func geoWallProbe(grid geometry.Grid) combat.WallProbe {
	return func(direction uint8, x, y int) (uint8, error) {
		wall, ok := grid.WallWrapped(x, y, int(direction))
		if !ok || wall == 0 {
			return combat.WallOpen, nil
		}
		return combat.WallBlocking, nil
	}
}

func fillTacticalCell(screen *ebiten.Image, column, row int, ink color.Color) {
	left := tacticalLeft + column*tacticalCellSize
	top := tacticalTop + row*tacticalCellSize
	for y := top; y < top+tacticalCellSize-1; y++ {
		for x := left; x < left+tacticalCellSize-1; x++ {
			screen.Set(x, y, ink)
		}
	}
}

// drawTactical 畫出由目前地城位置生成的戰術戰場。這一版只呈現地形，
// 還沒有 combatant、輸入或回合流程。
func drawTactical(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	drawFrame(screen, foreground, accent)
	drawText(screen, "TACTICAL MAP PREVIEW", 232, 52, accent)
	if a.initialMap == nil {
		drawText(screen, "DUNGEON MAP IS NOT LOADED", 196, 190, foreground)
		return
	}

	grid, err := combat.GenerateIndoorTacticalGrid(int(a.spawn.X), int(a.spawn.Y), geoWallProbe(a.initialMap.Grid))
	if err != nil {
		drawText(screen, "TACTICAL MAP ERROR", 232, 190, accent)
		return
	}

	classes := gamepack.OriginalCombatCellClassTable()
	floor := color.RGBA{40, 72, 72, 255}
	wall := color.RGBA{170, 255, 255, 255}
	if a.modern {
		floor = color.RGBA{46, 54, 66, 255}
		wall = color.RGBA{238, 232, 207, 255}
	}

	painted := 0
	blocking := 0
	for row := 0; row < combat.TacticalMapHeight; row++ {
		for column := 0; column < combat.TacticalMapWidth; column++ {
			code := grid.Terrain[row*combat.TacticalRowStride+column]
			if code == combat.UnpaintedCellClass {
				continue
			}
			painted++
			ink := floor
			if int(code) < len(classes) && classes[code].EntryThreshold == 0xFF {
				ink = wall
				blocking++
			}
			fillTacticalCell(screen, column, row, ink)
		}
	}

	originX, originY := combat.DeploymentCell(0, 0, 0, 0)
	if originX >= 0 && originX < combat.TacticalMapWidth && originY >= 0 && originY < combat.TacticalMapHeight {
		fillTacticalCell(screen, originX, originY, accent)
	}

	drawText(screen, fmt.Sprintf("DUNGEON %d,%d  CELLS %d  BLOCKING %d",
		a.spawn.X, a.spawn.Y, painted, blocking), 70, 344, foreground)
	drawText(screen, "GEO WALL TYPE TO 0/1/3 MAPPING: UNRESOLVED", 70, 362, accent)
	drawText(screen, "F5: BACK TO FIRST-PERSON VIEW", 360, 344, foreground)
}
