package main

import (
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
)

// 每一則訊息都要有繁中，缺一則就會在畫面上默默退回英文。
func TestEveryMessageHasATraditionalChineseString(t *testing.T) {
	for id, entry := range messages {
		if strings.TrimSpace(entry[0]) == "" {
			t.Fatalf("message %d has no English string", id)
		}
		if strings.TrimSpace(entry[1]) == "" {
			t.Fatalf("message %d has no Traditional Chinese string", id)
		}
	}
}

func TestTextPicksTheLanguage(t *testing.T) {
	english := &app{language: languageEnglish}
	chinese := &app{language: languageTraditionalChinese}
	if got := english.text(msgMenuTitle); got != "PARTY CREATION MENU" {
		t.Fatalf("english menu title %q", got)
	}
	if got := chinese.text(msgMenuTitle); got != "人物管理選擇項" {
		t.Fatalf("chinese menu title %q", got)
	}
	if got := chinese.text(messageID(9999)); got != "" {
		t.Fatalf("an unknown message returned %q", got)
	}
}

// 指定繁中卻沒有字型要失敗即關閉：內建字型沒有漢字，硬跑會整片留白，
// 而留白在畫面上看起來像繪製壞掉。
func TestResolveUILanguageRefusesChineseWithoutAFont(t *testing.T) {
	if _, _, err := resolveUILanguage("zh", "", "", ""); err == nil {
		t.Fatal("the Chinese UI was accepted without a font")
	}
}

func TestResolveUILanguageDefaultsToEnglish(t *testing.T) {
	got, face, err := resolveUILanguage("auto", "", "", "")
	if err != nil || got != languageEnglish || face == nil {
		t.Fatalf("language %v face %v err %v", got, face != nil, err)
	}
	if _, _, err := resolveUILanguage("klingon", "", "", ""); err == nil {
		t.Fatal("an unknown language was accepted")
	}
}

// 兼職依說明書的寫法由組成職業合成，magic-user 自己帶連字號也要拆對。
func TestOptionTextComposesMulticlassNames(t *testing.T) {
	chinese := &app{language: languageTraditionalChinese}
	for id, want := range map[string]string{
		"magic-user":                "魔法師",
		"fighter-thief":             "戰士／賊",
		"fighter-magic-user-thief":  "戰士／魔法師／賊",
		"cleric-fighter-magic-user": "牧師／戰士／魔法師",
	} {
		if got := chinese.optionText(id, "IGNORED"); got != want {
			t.Fatalf("%s -> %q, want %q", id, got, want)
		}
	}
	english := &app{language: languageEnglish}
	if got := english.optionText("fighter", "Fighter"); got != "Fighter" {
		t.Fatalf("english label %q", got)
	}
	if got := chinese.optionText("bard", "Bard"); got != "Bard" {
		t.Fatalf("an unknown option should fall back to its label, got %q", got)
	}
}

// 每一個實際會出現的選項都要有譯名，否則畫面會默默混著英文。
func TestEveryCreationOptionHasAChineseName(t *testing.T) {
	chinese := &app{language: languageTraditionalChinese}
	flow := creation.NewFlow()
	for raceIndex := range creation.Races {
		flow.RaceIndex = raceIndex
		for _, stage := range []creation.Stage{
			creation.StageRace, creation.StageGender, creation.StageClass, creation.StageAlignment,
		} {
			flow.Stage = stage
			labels, ids := flow.Options(), flow.OptionIDs()
			for index, id := range ids {
				if got := chinese.optionText(id, labels[index]); got == labels[index] {
					t.Fatalf("option %q (%s) has no Chinese name", id, labels[index])
				}
			}
		}
	}
}

func TestAbilityNamesCoverSixScores(t *testing.T) {
	chinese := &app{language: languageTraditionalChinese}
	want := []string{"力量", "智慧", "睿智", "敏捷", "體質", "魅力"}
	for index, expected := range want {
		if got := chinese.abilityName(index); got != expected {
			t.Fatalf("ability %d -> %q, want %q", index, got, expected)
		}
	}
	if chinese.abilityName(6) != "" {
		t.Fatal("an out-of-range ability returned a name")
	}
}

// 每個有英文提示的階段都要有繁中提示，否則畫面會一半中文一半英文。
func TestEveryCreationHintHasATranslation(t *testing.T) {
	chinese := &app{language: languageTraditionalChinese}
	for _, stage := range []string{"race", "class", "alignment", "portrait", "icon"} {
		if creation.HintFor(stage) == "" {
			t.Fatalf("stage %q has no English hint to translate", stage)
		}
		if got := chinese.hint(stage); got == creation.HintFor(stage) {
			t.Fatalf("stage %q hint is still English", stage)
		}
	}
	english := &app{language: languageEnglish}
	if english.hint("race") != creation.HintFor("race") {
		t.Fatal("the English hint changed")
	}
}
