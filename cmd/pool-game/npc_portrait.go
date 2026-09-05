package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// approachPortrait 回傳現在要蓋在第一人稱框上的 NPC 半身像；不該蓋就回 nil。
//
// 原版的 `SETUP MONSTER 12,2,9` 之後接兩個 `APPROACH`，那一框就被 Rolf 的
// 半身像蓋掉（`docs/reference/original-dos/adventure/03-rolf-approach.png`，
// 狀態列 `15, 1 W`）。導覽開始之後那一框又變回視野（同一批基準圖的
// `05-tyr-stop-11-2-south.png`，狀態列 `11, 2 S`），所以只有第一頁要蓋。
func (a *app) approachPortrait() *ebiten.Image {
	// `introWaiting` 只表示「在等 Return」，導覽的每一頁也是。要蓋半身像的
	// 是導覽開始**之前**那一頁，所以還要 `!tourActive`——原版按下第一個
	// Return 之後那一框就變回視野了（`19-Return-raw`，位置已經走到 `14, 1`）。
	if !a.introWaiting || a.tourActive || a.initialEvent == nil {
		return nil
	}
	if a.npcPortrait != nil {
		return a.npcPortrait
	}
	if a.loadNPCPortrait == nil {
		return nil
	}
	portrait, err := a.loadNPCPortrait(uint8(a.spawn.Map.Archive),
		gamepack.RolfPortraitHeadBlock, a.initialEvent.PortraitBody)
	if err != nil {
		// 疊不出來就退回原本的視野；不要因為缺一張圖就讓事件走不下去。
		a.loadNPCPortrait = nil
		return nil
	}
	a.npcPortrait = portrait
	return portrait
}

// clearNPCPortrait 在半身像該收掉的時候丟掉快取（換主題也要重疊一次）。
func (a *app) clearNPCPortrait() { a.npcPortrait = nil }
