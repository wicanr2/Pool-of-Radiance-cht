package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/etenfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// languageSwitchKey 是遊戲中切換繁中／英文的鍵（#127）。
//
// 選 F7 的理由：原版說明書與鍵表沒有任何功能鍵，玩法全用字母、方向鍵、
// ENTER 與 ESC；remake 已經用掉 F1 說明、F2 配色、F3 攻略、F4 素材總覽、
// F5 戰術圖、F6 作弊與 F10 離開。F7 兩邊都沒人用。
const languageSwitchKey = ebiten.KeyF7

// errNoChineseFace 是沒有倚天字型時按 F7 的答覆。內建的 7x13 沒有漢字，
// 硬切過去會整片留白，看起來像繪製壞掉。
var errNoChineseFace = errors.New("Chinese UI needs -eten-font; no ETen font was loaded")

// loadSwitchableChineseFace 在英文啟動時也先把倚天字型載好，F7 才切得過去。
// 載不進來只報一行：英文照樣能玩，只是 F7 會說明為什麼切不到中文。
func (a *app) loadSwitchableChineseFace(standardPath, symbolPath, asciiPath string) {
	if a.chineseFace != nil || standardPath == "" {
		return
	}
	_, face, err := resolveUILanguage("zh", standardPath, symbolPath, asciiPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "language switch:", err)
		return
	}
	a.rememberChineseFace(face)
}

func (a *app) rememberChineseFace(face font.Face) {
	a.chineseFace = face
	if eten, ok := face.(*etenfont.Face); ok {
		a.chineseShadow = eten.ShadowFace()
	}
}

// switchLanguage 換掉顯示語言用到的每一份東西：字型、介面字串（`a.text` 每次
// 依 `a.language` 查）、遊戲文字表、怪物譯名表與攻略表。規則狀態一個都不碰，
// 這是表現層（AGENTS.md §3）。已經排好版、正在畫面上的那一段敘事維持原語言，
// 下一段才換——那一段的斷行是照舊字型的字寬算的。
func (a *app) switchLanguage() error {
	next := languageTraditionalChinese
	if a.language == languageTraditionalChinese {
		next = languageEnglish
	}
	if next == languageTraditionalChinese && a.chineseFace == nil {
		return errNoChineseFace
	}
	catalogue, err := gameTextFor(next)
	if err != nil {
		return err
	}
	monsters, err := monsterTextFor(next)
	if err != nil {
		return err
	}
	guideCatalogue, err := guideFor(next)
	if err != nil {
		return err
	}
	if next == languageTraditionalChinese {
		uiFace, uiShadowFace = a.chineseFace, a.chineseShadow
	} else {
		uiFace, uiShadowFace = basicfont.Face7x13, nil
	}
	a.language, a.gameText, a.monsterText, a.guide = next, catalogue, monsters, guideCatalogue
	return nil
}
