package main

import (
	"image/color"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// 兩套外觀。F2 在它們之間切換。
//
// 原版只有 EGA 十六色，所以「現代」這一套不是另一份美術素材，是**同一份
// 索引像素換一張色盤**——`graphics.Picture.RGBA` 本來就收色盤參數，這是
// engine 既有的接縫，不需要為了換皮複製一份圖。
//
// 色盤的排法：十六格的色相與明度順序照 EGA 不動（換順序會讓原版美術的
// 明暗關係垮掉），只把過飽和的純色收斂、把最暗與最亮兩格對齊介面的
// 背景與前景。第 0 格與第 15 格因此與介面同色，畫面才不會出現「文字是
// 米白、圖還是純白」那種兩套配色打架的樣子。
type theme struct {
	background, foreground, accent color.RGBA
	// tacticalFloor 與 tacticalWall 是戰術地圖的地板與牆。
	tacticalFloor, tacticalWall color.RGBA
	// palette 是原版索引像素要用的十六色。
	palette [16]color.RGBA
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: 255} }

// classicTheme 是原版的 EGA 十六色與 CGA 青／黃介面。
func classicTheme() theme {
	return theme{
		background:    rgb(0, 0, 0),
		foreground:    rgb(170, 255, 255),
		accent:        rgb(255, 255, 85),
		tacticalFloor: rgb(40, 72, 72),
		tacticalWall:  rgb(170, 255, 255),
		palette:       graphics.EGA16,
	}
}

// modernTheme 是收斂過的那一套。
func modernTheme() theme {
	background, foreground := rgb(16, 20, 30), rgb(238, 232, 207)
	return theme{
		background:    background,
		foreground:    foreground,
		accent:        rgb(255, 202, 72),
		tacticalFloor: rgb(46, 54, 66),
		tacticalWall:  foreground,
		palette: [16]color.RGBA{
			background, rgb(43, 58, 140), rgb(47, 125, 70), rgb(47, 143, 146),
			rgb(156, 52, 52), rgb(139, 58, 130), rgb(154, 98, 36), rgb(185, 178, 162),
			rgb(74, 81, 98), rgb(91, 120, 230), rgb(111, 203, 122), rgb(116, 214, 218),
			rgb(224, 107, 96), rgb(217, 122, 203), rgb(240, 210, 100), foreground,
		},
	}
}

// currentTheme 回傳目前那一套。
func (a *app) currentTheme() theme {
	if a.modern {
		return modernTheme()
	}
	return classicTheme()
}

// artPalette 是原版素材要用的色盤。sprite（肖像、戰鬥圖示）、tileset
//（牆面圖章、第一人稱背景）與標題圖都走這一支，所以換主題時三者同步。
func (a *app) artPalette() [16]color.RGBA { return a.currentTheme().palette }

// switchTheme 換一套外觀，並把已經算成圖的素材重畫一次。
//
// 每格重算的東西（牆面圖章、背景色塊）下一影格就跟上；算過一次就存起來的
// 那幾張（標題、肖像、戰鬥圖示、NPC 半身像）不重畫的話會留在舊色盤上——
// 那正是「換了主題只有文字變色」的樣子。
func (a *app) switchTheme() error {
	a.modern = !a.modern
	if a.reloadTitle != nil {
		if err := a.reloadTitle(); err != nil {
			return err
		}
	}
	if a.portrait != nil && a.loadPortrait != nil {
		if err := a.reloadPortrait(); err != nil {
			return err
		}
	}
	if (a.iconReady != nil || a.iconAction != nil) && a.loadIcon != nil {
		if err := a.reloadIcons(); err != nil {
			return err
		}
	}
	a.clearNPCPortrait()
	return nil
}
