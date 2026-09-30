package main

import (
	"math/rand"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"golang.org/x/image/font/basicfont"
)

func keepUIFaces(t *testing.T) {
	t.Helper()
	face, shadow := uiFace, uiShadowFace
	t.Cleanup(func() { uiFace, uiShadowFace = face, shadow })
}

// F7 從 Update() 進去：英文 → 繁中 → 英文，介面字串、遊戲文字表、怪物譯名
// 與攻略表一起換（#127）。測試用內建字型頂替倚天字型——切換只管換哪一份，
// 不管字模。
func TestF7SwitchesLanguageThroughUpdate(t *testing.T) {
	keepUIFaces(t)
	chinese := basicfont.Face7x13
	a := &app{
		mode:        modeTitle,
		flow:        creation.NewFlow(),
		roller:      diceRoller{random: rand.New(rand.NewSource(1))},
		chineseFace: chinese,
	}
	english := a.text(msgMenuTitle)

	if err := press(a, languageSwitchKey); err != nil {
		t.Fatal(err)
	}
	if a.language != languageTraditionalChinese {
		t.Fatalf("F7 did not switch to Traditional Chinese (status %q)", a.statusLine)
	}
	if a.gameText == nil || a.monsterText == nil || a.guide == nil {
		t.Fatalf("catalogues not swapped: gameText=%v monsterText=%v guide=%v",
			a.gameText != nil, a.monsterText != nil, a.guide != nil)
	}
	if uiFace != chinese {
		t.Fatal("uiFace is not the Chinese face after F7")
	}
	if got := a.text(msgMenuTitle); got == english {
		t.Fatalf("menu title still English after F7: %q", got)
	}

	if err := press(a, languageSwitchKey); err != nil {
		t.Fatal(err)
	}
	if a.language != languageEnglish || a.gameText != nil || a.monsterText != nil {
		t.Fatal("second F7 did not return to English")
	}
	if uiFace != basicfont.Face7x13 || uiShadowFace != nil {
		t.Fatal("English face not restored")
	}
	if got := a.text(msgMenuTitle); got != english {
		t.Fatalf("menu title %q, want %q", got, english)
	}
}

// 沒有倚天字型時 F7 不切，狀態列說明原因——內建字型沒有漢字，硬切會整片留白。
func TestF7WithoutChineseFaceExplains(t *testing.T) {
	keepUIFaces(t)
	a := &app{mode: modeTitle, flow: creation.NewFlow(), roller: diceRoller{random: rand.New(rand.NewSource(1))}}
	if err := press(a, languageSwitchKey); err != nil {
		t.Fatal(err)
	}
	if a.language != languageEnglish {
		t.Fatal("switched to Chinese without a Chinese face")
	}
	if a.statusLine != errNoChineseFace.Error() {
		t.Fatalf("status %q, want %q", a.statusLine, errNoChineseFace.Error())
	}
}
