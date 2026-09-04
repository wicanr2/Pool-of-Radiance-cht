package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 每一則訊息都要有繁中，缺一則就會在畫面上默默退回英文。
// 字串現在住在 game pack 裡，所以這一則同時是「pack 有沒有這一則」的檢查。
func TestEveryMessageHasATraditionalChineseString(t *testing.T) {
	for id := range messageKeys {
		if strings.TrimSpace(packMessage(id, "en")) == "" {
			t.Fatalf("message %d (%s) has no English string", id, messageKeys[id])
		}
		if strings.TrimSpace(packMessage(id, "zh-TW")) == "" {
			t.Fatalf("message %d (%s) has no Traditional Chinese string", id, messageKeys[id])
		}
	}
}

// messageKeys 與 pack 的 locale 表要一一對應。少一邊的症狀不一樣但都難查：
// key 沒有字串會在畫面上留白，字串沒有 key 則是永遠印不出來的死資料。
func TestMessageKeysAndThePackAgree(t *testing.T) {
	if len(messageKeys) == 0 {
		t.Fatal("messageKeys 是空的")
	}
	byKey := map[string]messageID{}
	for id, key := range messageKeys {
		if previous, clash := byKey[key]; clash {
			t.Fatalf("key %q 同時對到 %d 與 %d", key, previous, id)
		}
		byKey[key] = id
	}
	for _, locale := range []string{"en", "zh-TW"} {
		table, err := gamepack.LocaleTable(locale)
		if err != nil {
			t.Fatalf("載入 %s 的字串表：%v", locale, err)
		}
		if len(table) != len(messageKeys) {
			t.Errorf("%s 有 %d 條字串，messageKeys 有 %d 個", locale, len(table), len(messageKeys))
		}
		for key := range table {
			if _, ok := byKey[key]; !ok {
				t.Errorf("%s 的 %q 沒有任何 messageID 用得到", locale, key)
			}
		}
	}
}

// pack 載不進來時要看得見：畫面上出現 key 一眼就知道是資料沒載到，
// 整片空字串則會被誤認成排版壞掉。
func TestUnknownMessageReturnsNothing(t *testing.T) {
	if got := packMessage(messageID(9999), "en"); got != "" {
		t.Fatalf("未知的訊息回了 %q", got)
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

// 狀態列的訊息在沒有接上語言時要退回英文，接上之後要是中文——
// 測試直接建構 tacticalState，所以這條路徑一定會被走到。
func TestTacticalStatusFallsBackToEnglishWithoutALanguage(t *testing.T) {
	var state tacticalState
	if got := state.say(msgStatusRound, 3); got != "ROUND 3" {
		t.Fatalf("without a language: %q", got)
	}
	chinese := &app{language: languageTraditionalChinese}
	state.Text = chinese.text
	if got := state.say(msgStatusRound, 3); got != "第 3 回合" {
		t.Fatalf("with Chinese: %q", got)
	}
	if got := state.say(msgStatusVictory); got != "獲勝" {
		t.Fatalf("victory: %q", got)
	}
}

// 半形詞之間的空白要留著。原本的條件只看 token 寬度是不是 1，
// 於是整句英文會被接成一個字：`PRESS RETURN` → `PRESSRETURN`。
// 遊戲的事件文字在英文模式與未翻的句子都走這條路徑。
func TestWrapDisplayKeepsSpacesBetweenWords(t *testing.T) {
	lines := wrapDisplay("PRESS <RETURN> OR BUTTON TO CONTINUE", 68)
	if len(lines) != 1 {
		t.Fatalf("wrapped into %d lines: %q", len(lines), lines)
	}
	if lines[0] != "PRESS <RETURN> OR BUTTON TO CONTINUE" {
		t.Fatalf("lost the spacing: %q", lines[0])
	}
}

// 中英混排時，全形字兩側不補空白，半形詞之間補。
func TestWrapDisplayMixesHanAndLatin(t *testing.T) {
	lines := wrapDisplay("Hills with cave 有山洞的山丘", 68)
	if len(lines) != 1 || lines[0] != "Hills with cave有山洞的山丘" {
		t.Fatalf("mixed line came out as %q", lines)
	}
}

// 換行寬度是硬界限：一行最多讓一個收尾標點溢位。
func TestWrapDisplayBoundsTheClosingPunctuationOverflow(t *testing.T) {
	value := strings.Repeat("字、", 60) + "。」』〉》"
	for _, line := range wrapDisplay(value, 40) {
		width := 0
		for _, r := range line {
			width += runeWidth(r)
		}
		if width > 40+closingPunctuationSlack {
			t.Fatalf("line is %d columns wide: %q", width, line)
		}
	}
}

// 字型沒有破折號與刪節號的字模，畫出來會是空白方塊。顯示前換成畫得出來的
// 形狀，而不是留著讓玩家以為是缺字。
func TestDisplayTextReplacesGlyphsTheFontLacks(t *testing.T) {
	got := displayText("等一下…那是什麼—一把劍～")
	if strings.ContainsAny(got, "…—～") {
		t.Fatalf("display text still carries glyphs the font lacks: %q", got)
	}
	if got != "等一下...那是什麼--一把劍~" {
		t.Fatalf("display text came out as %q", got)
	}
}

// 兩種語言的格式化字串要吃同一組參數。順序寫反時 go vet 抓不到
// （兩邊都是合法的字串），但畫面上會印出 %!d(string=…) 這種東西。
func TestTranslatedFormatStringsTakeTheSameArguments(t *testing.T) {
	verbs := regexp.MustCompile(`%[-+ #0]*[0-9*]*(?:\.[0-9*]+)?[a-zA-Z]`)
	for id := range messageKeys {
		first, second := packMessage(id, "en"), packMessage(id, "zh-TW")
		english, chinese := verbs.FindAllString(first, -1), verbs.FindAllString(second, -1)
		if len(english) != len(chinese) {
			t.Fatalf("message %d has %d verbs in English and %d in Chinese: %q / %q",
				id, len(english), len(chinese), first, second)
		}
		for index := range english {
			if english[index][len(english[index])-1] != chinese[index][len(chinese[index])-1] {
				t.Fatalf("message %d verb %d is %s in English and %s in Chinese: %q / %q",
					id, index, english[index], chinese[index], first, second)
			}
		}
	}
}
