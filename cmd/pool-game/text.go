package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/etenfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// language 是 UI 的顯示語言。
type language int

const (
	languageEnglish language = iota
	languageTraditionalChinese
)

type messageID int

const (
	msgTitleHint messageID = iota
	msgMenuTitle
	msgMenuCreate
	msgMenuAdd
	msgMenuLoad
	msgMenuBegin
	msgMenuCounts
	msgFooter
)

// messages 的繁中一律取自軟體世界代理當年的官方中文說明書
// （`docs/reference/manual/manual-vol2.md` p.8–p.10 的「人物管理選擇項」一節），
// 不是重新翻譯。說明書沒有對應字串的（畫面提示、功能鍵列）才自行擬定，
// 用詞依 `docs/reference/manual/glossary.md` 的定案譯名。
var messages = map[messageID][2]string{
	// 標題畫面的按鍵提示，說明書沒有，鍵名保持原文。
	msgTitleHint: {"ENTER / SPACE", "ENTER／空白鍵"},
	// 說明書 p.8：「螢幕上便會出現人物管理選擇項」。
	msgMenuTitle: {"PARTY CREATION MENU", "人物管理選擇項"},
	// 說明書 p.8 C）REATE NEW CHARACTER：「創造一名人物…並將之存放到『人物名單』內」。
	msgMenuCreate: {"C  CREATE NEW CHARACTER", "C　創造新人物"},
	// 說明書 p.9 A）DD CHARACTER：「將人物加入隊伍」。
	msgMenuAdd: {"A  ADD CHARACTER TO PARTY", "A　將人物加入隊伍"},
	// 說明書 p.9 L）OAD SAVED GAME：「叫出以前存下的遊戲進度」。
	msgMenuLoad: {"L  LOAD SAVED GAME", "L　叫出存下的進度"},
	// 說明書 p.10 B）EGIN ADVENTURING：「離開人物管理選擇項，開始冒險」。
	msgMenuBegin: {"B  BEGIN ADVENTURING", "B　開始冒險"},
	// 「人物名單」是說明書對 character library 的固定用詞。
	msgMenuCounts: {"LIBRARY %d   PARTY %d/6", "人物名單 %d　隊伍 %d/6"},
	// 功能鍵列是 remake 自己的東西，說明書沒有；鍵名保持原文。
	msgFooter: {
		"F1 Help  F2 Theme  F5 Tactical  ESC Back  F10 Quit",
		"F1 說明　F2 配色　F5 戰術圖　ESC 返回　F10 離開",
	},
}

// text 取出目前語言的字串。缺譯時退回英文而不是留白——留白在畫面上看不出是
// 缺譯還是繪製壞了。
func (a *app) text(id messageID) string {
	entry, ok := messages[id]
	if !ok {
		return ""
	}
	if a.language == languageTraditionalChinese && entry[1] != "" {
		return entry[1]
	}
	return entry[0]
}

// uiFace 是 drawText 實際使用的字型。Ebiten 的 Draw 是單執行緒的，啟動時設定
// 一次之後就不再變動，所以這裡用套件層變數而不是把每個呼叫點都改成帶字型。
var uiFace font.Face = basicfont.Face7x13

// resolveUILanguage 依旗標決定語言並載入字型。
//
// 指定繁中卻沒有給字型時失敗即關閉：內建的 7x13 沒有漢字，硬跑會整片留白，
// 而留白看起來像繪製壞掉，不像缺字型。
func resolveUILanguage(langFlag, standardPath, symbolPath, asciiPath string) (language, font.Face, error) {
	wantChinese := false
	switch langFlag {
	case "en":
	case "zh":
		wantChinese = true
	case "auto":
		wantChinese = standardPath != ""
	default:
		return 0, nil, fmt.Errorf("Pool UI language %q is not one of en, zh, auto", langFlag)
	}
	if !wantChinese {
		return languageEnglish, basicfont.Face7x13, nil
	}
	if standardPath == "" {
		return 0, nil, fmt.Errorf("Pool Traditional Chinese UI needs -eten-font; the built-in face has no Han glyphs")
	}
	if asciiPath == "" {
		candidate := filepath.Join(filepath.Dir(standardPath), "ascfont.15")
		if _, err := os.Stat(candidate); err == nil {
			asciiPath = candidate
		}
	}
	face, err := etenfont.LoadWithASCII(standardPath, symbolPath, asciiPath, basicfont.Face7x13, true)
	if err != nil {
		return 0, nil, err
	}
	return languageTraditionalChinese, face, nil
}
