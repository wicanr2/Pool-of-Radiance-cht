package main

import (
	"strings"
	"testing"
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
