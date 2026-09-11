package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gametext"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

func TestJournalCitationsAreRecognisedInBothWordings(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
		ok   bool
	}{
		{"抄進手冊", "你們把它抄進手冊，成為線索報導 46。", "46", true},
		{"括號引用", "一封來自首領的信。（線索報導 33）", "33", true},
		{"沒有引用", "地板底下什麼也沒有。", "", false},
		{"只有兩個字", "你們把線索寫下來。", "", false},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			cues := journalCitationsIn(item.text)
			if (len(cues) != 0) != item.ok {
				t.Fatalf("認出 %d 則，要 ok=%v", len(cues), item.ok)
			}
			if !item.ok {
				return
			}
			if cues[0].ID != item.want {
				t.Fatalf("認成 %s，要 %s", cues[0].ID, item.want)
			}
			if cues[0].Kind != journal.Clue {
				t.Fatalf("章別是 %s，要 %s", cues[0].Kind, journal.Clue)
			}
		})
	}
}

// 遊戲文字報的每一個編號，手冊裡都要查得到。查不到的話玩家按下去會翻到
// 一則不相干的內容，而畫面上分不出是引用寫錯還是手冊漏轉錄。
func TestEveryCitedJournalEntryExists(t *testing.T) {
	catalogue, err := gametext.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	cited := 0
	for _, source := range catalogue.Sources() {
		text := catalogue.Translate(source)
		for _, cue := range journalCitationsIn(text) {
			cited++
			if _, found := corpus.Lookup(cue.Kind, cue.ID); !found {
				t.Errorf("遊戲文字引用 %s %s，手冊裡沒有這一則：%q", cue.Kind, cue.ID, text)
			}
		}
	}
	// 引用數是資料的性質，不是這一支的行為；釘住是為了「掃描面有沒有整塊
	// 消失」——譯文換寫法之後這裡歸零，測試仍會綠，那才是危險的。
	// 三十六則：三十二則線索報導（其中兩處一段話報兩則）加市政廳外牆
	// 那四則公告。釘住是為了「掃描面有沒有整塊消失」——譯文換寫法之後
	// 這裡歸零，測試仍會綠，那才是危險的。
	if cited < 30 {
		t.Fatalf("只掃到 %d 處手冊引用，原本有三十六處；掃描面可能破了", cited)
	}
}

// 自動翻頁要能跨章別跳，而不是只在目前這一章裡找。
func TestJumpToSwitchesKind(t *testing.T) {
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	state := newJournalState(corpus)
	state.kind = 1 // 停在酒店傳言
	if !state.jumpTo(journal.Clue, "46") {
		t.Fatal("跨章別跳不過去")
	}
	if state.currentKind() != journal.Clue {
		t.Fatalf("跳完停在 %s，要 %s", state.currentKind(), journal.Clue)
	}
	entry, ok := state.current()
	if !ok || entry.ID != "46" {
		t.Fatalf("跳完停在 %v，要線索報導 46", entry.ID)
	}
	if state.jumpTo(journal.Clue, "999") {
		t.Fatal("手冊裡沒有 999，不該回報跳成功")
	}
}

// 文字框報了編號就設 cue，取走一次之後不再彈——玩家讀完關掉手冊回到同一個
// 文字框，再按一次 ENTER 應該是「繼續」。
func TestJournalCueFiresOnceForTheSameText(t *testing.T) {
	a := &app{language: languageTraditionalChinese}
	a.eventText = "你們把它抄進手冊，成為線索報導 46。"
	a.updateJournalCue()
	cue, ok := a.takeJournalCue()
	if !ok || cue.ID != "46" {
		t.Fatalf("第一次沒取到線索報導 46，取到 %v／%v", cue, ok)
	}
	if _, again := a.takeJournalCue(); again {
		t.Fatal("同一則彈了第二次")
	}
	// 同一段文字再算一次也不該回來。
	a.updateJournalCue()
	if _, again := a.takeJournalCue(); again {
		t.Fatal("重算之後同一則又彈出來")
	}
	// 換一則就要彈。
	a.eventText = "你們把這幾頁抄進手冊，成為線索報導 3。"
	a.updateJournalCue()
	next, ok := a.takeJournalCue()
	if !ok || next.ID != "3" {
		t.Fatalf("換了一則沒彈，取到 %v／%v", next, ok)
	}
}

// 英文模式沒有手冊（那是軟體世界的中譯），所以不能設 cue——設了會讓 ENTER
// 被吃掉卻什麼都不發生。
func TestJournalCueStaysEmptyInEnglish(t *testing.T) {
	a := &app{language: languageEnglish}
	a.eventText = "你們把它抄進手冊，成為線索報導 46。"
	a.updateJournalCue()
	if len(a.journalCues) != 0 {
		t.Fatalf("英文模式設了 cue：%v", a.journalCues)
	}
	if prompt := a.journalCuePrompt(); prompt != "" {
		t.Fatalf("英文模式印了提示 %q", prompt)
	}
}

// 從文字框那一下 ENTER 到「手冊開著、停在被引用的那一則」，整條走一次。
func TestOpeningTheJournalFromACitation(t *testing.T) {
	a := &app{language: languageTraditionalChinese}
	a.eventText = "指揮官說了一個關於那池子的故事……成為線索報導 46。"
	a.updateJournalCue()
	if prompt := a.journalCuePrompt(); prompt == "" {
		t.Fatal("文字框外沒有提示，玩家不會知道按下去會翻手冊")
	}
	cue, ok := a.takeJournalCue()
	if !ok {
		t.Fatal("沒有待翻的條目")
	}
	if err := a.openJournalAt(cue); err != nil {
		t.Fatal(err)
	}
	if !a.journalOpen {
		t.Fatal("手冊沒有開")
	}
	entry, ok := a.journal.current()
	if !ok || entry.Kind != journal.Clue || entry.ID != "46" {
		t.Fatalf("手冊停在 %v，要線索報導 46", entry)
	}
}

// 這一格的腳本跑完就忘掉已翻過的：再走回同一格還要再翻一次。
func TestFinishingACellForgetsWhatWasRead(t *testing.T) {
	a := &app{language: languageTraditionalChinese}
	a.eventText = "……成為線索報導 46。"
	a.updateJournalCue()
	if _, ok := a.takeJournalCue(); !ok {
		t.Fatal("第一次沒取到")
	}
	a.finishCellBlock()
	a.eventText = "……成為線索報導 46。"
	a.updateJournalCue()
	if _, ok := a.takeJournalCue(); !ok {
		t.Fatal("走回同一格之後翻不出來了")
	}
}

// 一段話引用多則是常態，不是例外。這三句都出自原版譯文。
func TestJournalCitationsReadWholeLists(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []journalCue
	}{
		{
			"市政廳外牆的四則公告",
			"牆上貼著公告，你們把編號記進手冊： 公告字號 LXIV、LXXVIII、CIX 與 LIX。",
			[]journalCue{
				{journal.Proclamation, "LXIV"}, {journal.Proclamation, "LXXVIII"},
				{journal.Proclamation, "CIX"}, {journal.Proclamation, "LIX"},
			},
		},
		{
			"一次兩則線索報導",
			"櫃子裡有幾份卷宗。你們把文件塞進手冊，成為線索報導 23 與 14。",
			[]journalCue{{journal.Clue, "23"}, {journal.Clue, "14"}},
		},
		{
			"酒館傳言",
			"你們聽到酒館裡的傳言 12。",
			[]journalCue{{journal.Rumour, "12"}},
		},
		{
			"章名後面沒有編號就不是引用",
			"公告字號",
			nil,
		},
		{
			"譯文裡的英文專有名詞不會被讀成羅馬字號",
			"一則給海盜的訊息，懸賞一隻真正的 Sahuagin。（線索報導 27）",
			[]journalCue{{journal.Clue, "27"}},
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := journalCitationsIn(item.text)
			if len(got) != len(item.want) {
				t.Fatalf("認出 %v，要 %v", got, item.want)
			}
			for index := range got {
				if got[index] != item.want[index] {
					t.Fatalf("第 %d 則是 %v，要 %v", index, got[index], item.want[index])
				}
			}
		})
	}
}

// 四則公告要一則一則翻，翻完才輪到「繼續」；提示上看得出還剩幾則。
func TestJournalCuesQueueUpInOrder(t *testing.T) {
	a := &app{language: languageTraditionalChinese}
	a.eventText = "牆上貼著公告，你們把編號記進手冊： 公告字號 LXIV、LXXVIII、CIX 與 LIX。"
	a.updateJournalCue()
	if prompt := a.journalCuePrompt(); !strings.Contains(prompt, "LXIV") || !strings.Contains(prompt, "3") {
		t.Fatalf("提示 %q 沒有指出第一則與剩下幾則", prompt)
	}
	for _, want := range []string{"LXIV", "LXXVIII", "CIX", "LIX"} {
		cue, ok := a.takeJournalCue()
		if !ok || cue.ID != want {
			t.Fatalf("取到 %v／%v，要 %s", cue, ok, want)
		}
		if cue.Kind != journal.Proclamation {
			t.Fatalf("%s 的章別是 %s，要議會公告", want, cue.Kind)
		}
	}
	if _, ok := a.takeJournalCue(); ok {
		t.Fatal("四則翻完之後還有東西")
	}
	if prompt := a.journalCuePrompt(); prompt != "" {
		t.Fatalf("翻完了還印提示 %q", prompt)
	}
}

// 按鍵接線：文字框有引用時那一下 ENTER 要翻手冊，翻完才輪到「繼續」。
// 直接走 `Update` 而不是只叫 `takeJournalCue`——會壞的是「哪一個 case 先攔截」，
// 不是取佇列本身。
// `J` 一則一則翻完引用，而 **ENTER 一下都不碰手冊**——它留給門與「繼續」。
//
// 原本是「按繼續那一下就翻過去」，於是同一個 ENTER 有三種意思（翻頁、回答門的
// BASH／EXIT、把事件按過去），玩家按下去會發生什麼取決於看不見的狀態。鎖住的門
// 就是這樣變成按了沒反應（spec 122／132）。
func TestJKeyOpensTheJournalAndEnterNeverDoes(t *testing.T) {
	a := &app{
		mode:      modeAdventure,
		introDone: true,
		language:  languageTraditionalChinese,
	}
	a.cellEventPending = true
	a.eventText = "牆上貼著公告，你們把編號記進手冊： 公告字號 LXIV、LXXVIII、CIX 與 LIX。"
	a.updateJournalCue()
	if len(a.journalCues) != 4 {
		t.Fatalf("待翻 %d 則，要四則", len(a.journalCues))
	}
	for index, want := range []string{"LXIV", "LXXVIII", "CIX", "LIX"} {
		if err := press(a, ebiten.KeyJ); err != nil {
			t.Fatalf("第 %d 則：%v", index+1, err)
		}
		if !a.journalOpen {
			t.Fatalf("第 %d 則沒有翻開手冊", index+1)
		}
		entry, ok := a.journal.current()
		if !ok || entry.ID != want {
			t.Fatalf("第 %d 則停在 %v，要 %s", index+1, entry.ID, want)
		}
		// ESC 收起手冊，回到同一個文字框。
		if err := press(a, ebiten.KeyEscape); err != nil {
			t.Fatal(err)
		}
		if a.journalOpen {
			t.Fatalf("第 %d 則之後手冊沒關", index+1)
		}
		if !a.cellEventPending {
			t.Fatalf("第 %d 則之後文字框不見了", index+1)
		}
	}
	// 四則翻完，再按 `J` 就是開在上次停的地方（沒有引用可跳了）。
	if err := press(a, ebiten.KeyJ); err != nil {
		t.Fatal(err)
	}
	if !a.journalOpen {
		t.Fatal("引用翻完之後 J 應該照樣開得了手冊")
	}
	if err := press(a, ebiten.KeyEscape); err != nil {
		t.Fatal(err)
	}

	// **ENTER 從頭到尾都不開手冊。** 重新灌一批引用，按 ENTER 要走「繼續」
	// 那條路（沒有 ECL session 時會走到錯誤路徑，重點是手冊沒被翻開）。
	// 翻過的不會再記一次（`journalCueDone`），所以先把那份紀錄清掉。
	a.journalCueDone = nil
	a.eventText = "牆上貼著公告，你們把編號記進手冊： 公告字號 LXIV。"
	a.updateJournalCue()
	if len(a.journalCues) == 0 {
		t.Fatal("第二批引用沒有記下來")
	}
	_ = press(a, ebiten.KeyEnter)
	if a.journalOpen {
		t.Fatal("ENTER 把手冊翻開了——那一下要留給門與繼續")
	}
}
