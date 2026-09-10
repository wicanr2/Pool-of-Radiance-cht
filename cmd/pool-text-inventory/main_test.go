package main

import "testing"

func inventoryOf(sources ...string) *report {
	result := &report{Unique: len(sources)}
	for _, source := range sources {
		result.Entries = append(result.Entries, entry{Source: source, Length: len([]rune(source))})
		result.Characters += len([]rune(source))
	}
	return result
}

// 覆蓋率用兩個尺度：句數與字元數。只看句數會被大量短選項灌得好看，
// 只看字元數又會忽略短句其實最常出現——所以兩個都要算對。
func TestCoverageCountsStringsAndCharacters(t *testing.T) {
	result := inventoryOf("AB", "CDEF", "GH", "IJ")
	translated := map[string]bool{"AB": true, "CDEF": true}
	summary, err := coverageOf("zh-TW", translated, result)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TranslatedStrings != 2 || summary.TranslatedCharacters != 6 {
		t.Fatalf("翻了 %d 句 %d 字", summary.TranslatedStrings, summary.TranslatedCharacters)
	}
	if summary.StringPercent != 50 {
		t.Fatalf("句數比例是 %v，四句翻兩句該是 50", summary.StringPercent)
	}
	if summary.CharacterPercent != 60 {
		t.Fatalf("字元比例是 %v，十字翻六字該是 60", summary.CharacterPercent)
	}
	if summary.Locale != "zh-TW" {
		t.Fatalf("locale 是 %q", summary.Locale)
	}
}

// 譯文表裡有盤點檔沒有的原文 → **失敗即關閉**。那代表兩邊其中一個是舊的，
// 而繼續算下去會得到一個看起來合理的錯數字：分母是這一份盤點，分子混著
// 上一份的原文。
func TestCoverageRefusesSourcesTheInventoryDoesNotHave(t *testing.T) {
	result := inventoryOf("AB", "CD")
	translated := map[string]bool{"AB": true, "已經不在盤點裡的舊句子": true}
	if _, err := coverageOf("zh-TW", translated, result); err == nil {
		t.Fatal("譯文表有盤點檔沒有的原文，卻算出了覆蓋率")
	}
}

// 空盤點不能除以零，也不能報出一個假的 100%。
func TestCoverageOnAnEmptyInventoryIsZero(t *testing.T) {
	summary, err := coverageOf("zh-TW", map[string]bool{}, &report{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.StringPercent != 0 || summary.CharacterPercent != 0 {
		t.Fatalf("空盤點算出 %v／%v", summary.StringPercent, summary.CharacterPercent)
	}
}
