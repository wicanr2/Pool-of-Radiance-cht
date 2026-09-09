package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 自動截圖用的畫面狀態輸出。
//
// 截圖腳本原本是「按幾下、睡幾秒、再拍」，判斷有沒有走對只靠比對前後兩張圖
// 不一樣。那個訊號太弱：漏掉一次按鍵時前後兩張還是不一樣（畫面確實動了，
// 只是動到別的地方去），於是流程整條偏掉而每一關都「通過」，最後在很後面
// 才以「A 打不開平面圖」的形式炸出來——而真正的原因發生在十幾步之前。
//
// 這裡把遊戲自己的狀態寫成一行字，腳本就能等到指定畫面才往下走，
// 錯的時候也停在出錯的那一步（rulebook 60：先建可驗證的訊號）。
// 只有帶 `-screen-state` 時才寫檔，正常遊玩完全不受影響。

// screenName 是目前畫面的識別字。順序照「誰蓋在誰上面」排：先問覆蓋層
// （（說明頁、面板），再問主畫面。
func (a *app) screenName() string {
	if a.help {
		return "help"
	}
	switch a.mode {
	case modeTitle:
		return "title"
	case modeMenu:
		if a.menuDropPending {
			return "menu-drop-confirm"
		}
		return "menu"
	case modeCreation:
		switch a.flow.Stage {
		case creation.StageRace:
			return "creation-race"
		case creation.StageGender:
			return "creation-gender"
		case creation.StageClass:
			return "creation-class"
		case creation.StageAlignment:
			return "creation-alignment"
		case creation.StageRoll:
			return "creation-roll"
		case creation.StageName:
			return "creation-name"
		case creation.StagePortrait:
			return "creation-portrait"
		case creation.StageIcon:
			return fmt.Sprintf("creation-icon-%d", a.iconMenu.level)
		case creation.StageIconConfirm:
			return "creation-icon-confirm"
		}
		return "creation"
	}
	switch {
	case a.guideOpen:
		// 三個狀態都要分得出來。攤開前會先出一次劇透警告，那一步只換底部
		// 那一列字——不報出來的話，截圖腳本只能盲按兩次 `V` 再賭結果，
		// 而漏掉其中一次的症狀是「停在 guide 等不到 guide-full」，
		// 看起來像功能壞了，其實是按鍵掉了。
		switch {
		case a.guideFull:
			return "guide-full"
		case a.guideSpoilerWarned:
			return "guide-warned"
		}
		return "guide"
	case a.spriteOpen:
		switch a.spritePage {
		case spritePageMonsters:
			return "sprites-monsters"
		case spritePageEffects:
			return "sprites-effects"
		case spritePageTerrain:
			return "sprites-terrain"
		}
		return "sprites"
	case a.viewSheetOpen:
		if a.viewSheetShown {
			return "view-sheet"
		}
		return "view-pick"
	case a.fieldCastOpen:
		// 挑法術那一步是整頁（spec 134），與挑人那個小框是兩張畫面；
		// 不分開報的話截圖腳本只能盲按。
		if a.fieldCastStage == fieldCastPickSpell {
			return "field-cast-spell"
		}
		return "field-cast"
	case a.journalOpen:
		// 章別也報出來。手冊有四章，翻章只換內容不換畫面，不報的話
		// 截圖腳本只能盲按 TAB 再賭自己停在哪一章。
		if a.journal != nil {
			return "journal-" + string(a.journal.currentKind())
		}
		return "journal"
	case a.equipmentOpen:
		return "equipment"
	case a.spellsOpen:
		// 組別也報出來。法術一覽有六組（神術一到三、巫術一到三），翻組只換
		// 內容不換畫面——不報的話截圖腳本只能盲按 TAB 再賭自己停在哪一組，
		// 而掉一次鍵的症狀是「拍到的那一張標題寫著別的級別」，看起來像沒事。
		if a.spells != nil && a.spells.group < len(spellGroups) {
			item := spellGroups[a.spells.group]
			class := "cleric"
			if item.Class == gamepack.SpellClassMagicUser {
				class = "magic-user"
			}
			return fmt.Sprintf("spells-%s-%d", class, item.Level)
		}
		return "spells"
	case a.shopActive:
		return "shop"
	case a.campOpen:
		// 兩層要分得出來（spec 135）：`camp` 是
		// `CAMP: SAVE VIEW MAGIC REST ALTER EXIT`，`camp-rest` 是按下
		// `REST` 之後那一列。換層只換最下面那一列，不報的話截圖腳本
		// 只能盲按。
		switch a.campStage {
		case campStageRest:
			return "camp-rest"
		case campStageAlter:
			return "camp-alter"
		case campStageOrderSelect:
			return "camp-order-select"
		case campStageOrderPlace:
			return "camp-order-place"
		case campStageSpeed:
			return "camp-speed"
		case campStagePics:
			return "camp-pics"
		case campStageQuitConfirm:
			return "camp-quit"
		case campStageDropConfirm:
			return "camp-drop"
		case campStageIcon:
			return "camp-icon"
		}
		return "camp"
	case a.templeActive:
		return "temple"
	case a.tacticalPreview:
		return "tactical"
	case a.combatActive:
		// 遭遇已經排好、還沒按 ENTER 進戰術盤面的那一步。不報出來的話
		// 截圖腳本沒辦法等到「架打起來了」。
		return "combat-staged"
	case a.introWaiting:
		return "adventure-intro"
	case a.tourActive:
		return "adventure-tour"
	case a.cellWaitingMenu:
		return "adventure-cell-menu"
	case a.cellEventPending:
		return "adventure-cell-text"
	case a.cellTextSticky && a.eventText != "":
		// 腳本跑完了，字還留在框裡：底下已經換成指令列、方向鍵走得動，
		// 但畫面與「什麼都沒發生的自由移動」不一樣，所以分開報。
		return "adventure-cell-done"
	case a.areaMapOpen:
		return "adventure-map"
	case a.freeMovementActive():
		return "adventure-move"
	}
	return "adventure"
}

// publishScreenName 把畫面識別字寫進 `-screen-state` 指定的檔案，
// 只在變動時寫。先寫暫存檔再 rename：腳本隨時可能讀，半個檔名讀起來
// 跟「還沒到那個畫面」分不出來。
func (a *app) publishScreenName() {
	if a.screenStatePath == "" {
		return
	}
	name := a.screenName()
	if name == a.screenStateLast {
		return
	}
	temp := a.screenStatePath + ".tmp"
	if err := os.WriteFile(temp, []byte(name+"\n"), 0o644); err != nil {
		return
	}
	if err := os.Rename(temp, a.screenStatePath); err != nil {
		return
	}
	a.screenStateLast = name
}

// defaultScreenStateDir 只是把相對路徑釘在工作目錄，避免寫到別處去。
func defaultScreenStatePath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(path)
}
