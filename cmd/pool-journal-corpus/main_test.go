package main

import (
	"strconv"
	"strings"
	"testing"
)

// 造一份數量剛好的條目集，當作 checkCounts 的正對照。
func fullSet() []entry {
	entries := []entry{}
	for number := 1; number <= wantClues; number++ {
		entries = append(entries, entry{Kind: "clue", ID: strconv.Itoa(number), Text: "x"})
	}
	for number := 1; number <= wantRumours; number++ {
		entries = append(entries, entry{Kind: "rumour", ID: strconv.Itoa(number), Text: "x"})
	}
	for number := 1; number <= wantProclamations; number++ {
		entries = append(entries, entry{Kind: "proclamation", ID: romanOf(number), Text: "x"})
	}
	for number := 1; number <= wantAppendices; number++ {
		entries = append(entries, entry{Kind: "appendix", ID: strconv.Itoa(number), Text: "x"})
	}
	return entries
}

func romanOf(value int) string {
	table := []struct {
		value  int
		digits string
	}{{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"}, {100, "C"}, {90, "XC"},
		{50, "L"}, {40, "XL"}, {10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"}}
	out := ""
	for _, item := range table {
		for value >= item.value {
			out += item.digits
			value -= item.value
		}
	}
	return out
}

// 公告的編號是羅馬數字，**減法那幾個必須算對**：IV 是 4 不是 6，
// 算錯的症狀是公告的順序錯，而每一條的內容都對——沒有人會發現。
func TestRomanHandlesSubtractivePairs(t *testing.T) {
	for _, testCase := range []struct {
		text string
		want int
	}{
		{"I", 1}, {"IV", 4}, {"V", 5}, {"IX", 9}, {"X", 10},
		{"XIV", 14}, {"XVIII", 18}, {"XL", 40}, {"LIX", 59}, {"MCMXC", 1990},
	} {
		if got := roman(testCase.text); got != testCase.want {
			t.Errorf("%s 算成 %d，該是 %d", testCase.text, got, testCase.want)
		}
	}
}

// 線索與傳言照**數字**排，公告照羅馬數字的**值**排。兩者都不是字串序——
// 字串序會把 10 排在 2 前面、把 IX 排在 X 前面（那個剛好對，所以更危險）。
func TestLessIDSortsByValueNotByString(t *testing.T) {
	if !lessID(entry{Kind: "clue", ID: "2"}, entry{Kind: "clue", ID: "10"}) {
		t.Error("線索 2 該排在 10 前面")
	}
	if lessID(entry{Kind: "clue", ID: "10"}, entry{Kind: "clue", ID: "2"}) {
		t.Error("線索 10 排到 2 前面了")
	}
	if !lessID(entry{Kind: "proclamation", ID: "IX"}, entry{Kind: "proclamation", ID: "X"}) {
		t.Error("公告 IX 該排在 X 前面")
	}
	// 字串序在這一組會給出相反的答案：XVIII < XX 但 "XVIII" > "XX"。
	if !lessID(entry{Kind: "proclamation", ID: "XVIII"}, entry{Kind: "proclamation", ID: "XX"}) {
		t.Error("公告 XVIII 該排在 XX 前面——這一組正是字串序會排反的")
	}
}

// 數量、重複、缺號三種都要擋。少了任何一種，「某一條的標題沒被認出來」
// 這個症狀就會安靜地變成「手冊少一條」。
func TestCheckCountsCatchesEachKindOfHole(t *testing.T) {
	if err := checkCounts(fullSet()); err != nil {
		t.Fatalf("數量剛好卻報錯：%v", err)
	}
	short := fullSet()[1:]
	if err := checkCounts(short); err == nil {
		t.Error("少一條卻沒報")
	}
	duplicated := append(fullSet(), entry{Kind: "clue", ID: "1", Text: "x"})
	if err := checkCounts(duplicated); err == nil {
		t.Error("同一個編號出現兩次卻沒報")
	}
	// 缺號：把線索 7 換成 99，總數不變、編號不連續。
	gapped := fullSet()
	for index := range gapped {
		if gapped[index].Kind == "clue" && gapped[index].ID == "7" {
			gapped[index].ID = "99"
			break
		}
	}
	if err := checkCounts(gapped); err == nil {
		t.Error("總數對但缺編號 7，卻沒報——這正是標題沒被認出來的形狀")
	}
}

// extract 解不到說明書那個數量就要失敗，不能吐一份少了幾條的語料庫出來。
func TestExtractRefusesAnIncompleteManual(t *testing.T) {
	text := strings.Join([]string{
		"## p.1（Pic1）",
		"",
		"### 線索報導 1",
		"",
		"這是第一條線索。",
		"",
	}, "\n")
	if _, err := extract(text); err == nil {
		t.Fatal("只有一條線索卻通過了")
	}
}
