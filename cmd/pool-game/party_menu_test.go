package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func menuApp(party ...poolsave.Character) *app {
	a := &app{mode: modeMenu, keys: scriptedKeys{}}
	a.state = poolsave.State{Party: party, CharacterLibrary: append([]poolsave.Character(nil), party...)}
	a.saveState = func(poolsave.State) error { return nil }
	return a
}

func rookie(name string) poolsave.Character {
	return poolsave.Character{
		Name: name, RaceID: "human", GenderID: "male", ClassID: "fighter",
		Abilities: [6]int{12, 10, 10, 12, 12, 10}, MaxHP: 8, CurrentHP: 8, RawHP: 8,
	}
}

// 說明書 p.9：「若你未將任何人物加入隊伍，則大部份的人物處理選擇項都將不會
// 出現」。空隊伍剩下的四項正好是 spec 008 那張原版截圖上的四項。
func TestEmptyPartyShowsOnlyTheFourOriginalEntries(t *testing.T) {
	a := menuApp()
	var keys []ebiten.Key
	for _, entry := range a.visiblePartyMenuEntries() {
		keys = append(keys, entry.key)
	}
	want := []ebiten.Key{ebiten.KeyC, ebiten.KeyA, ebiten.KeyL, ebiten.KeyE}
	if len(keys) != len(want) {
		t.Fatalf("空隊伍列出 %d 項，預期 %d 項", len(keys), len(want))
	}
	for index := range want {
		if keys[index] != want[index] {
			t.Fatalf("第 %d 項是 %v，預期 %v", index, keys[index], want[index])
		}
	}
	// 有人之後十一項全出現。
	a = menuApp(rookie("HERO"))
	if got := len(a.visiblePartyMenuEntries()); got != 11 {
		t.Fatalf("有隊伍時列出 %d 項，預期 11 項", got)
	}
}

// 看不見的指令連按鍵都不接受。
func TestHiddenPartyMenuEntriesIgnoreTheirKey(t *testing.T) {
	a := menuApp()
	a.keys = scriptedKeys{ebiten.KeyR: true}
	if err := a.partyMenuCommand(); err != nil {
		t.Fatal(err)
	}
	if a.statusLine != "" {
		t.Fatalf("空隊伍按 R 不該有反應，狀態列是 %q", a.statusLine)
	}
}

// D）ROP 要先問一次，Y 才真的除掉；除掉是「他的一切資料都將消失」，
// 所以隊伍與人物名單兩邊都要不見。
func TestDropAsksFirstThenClearsBothLists(t *testing.T) {
	a := menuApp(rookie("HERO"), rookie("TINA"))
	a.keys = scriptedKeys{ebiten.KeyD: true}
	if err := a.partyMenuCommand(); err != nil {
		t.Fatal(err)
	}
	if !a.menuDropPending {
		t.Fatal("D 沒有進入再確認")
	}
	if len(a.state.Party) != 2 {
		t.Fatal("再確認之前不該動到隊伍")
	}
	if !strings.Contains(a.statusLine, "HERO") {
		t.Fatalf("再確認沒有指名對象：%q", a.statusLine)
	}
	// N 取消。
	a.keys = scriptedKeys{ebiten.KeyN: true}
	if err := a.partyMenuDropConfirm(); err != nil {
		t.Fatal(err)
	}
	if a.menuDropPending || len(a.state.Party) != 2 {
		t.Fatal("N 應該取消而且不動到隊伍")
	}
	// 選 2 再 D，然後 Y。
	a.keys = scriptedKeys{ebiten.KeyD: true}
	a.menuMember = 1
	if err := a.partyMenuCommand(); err != nil {
		t.Fatal(err)
	}
	a.keys = scriptedKeys{ebiten.KeyY: true}
	if err := a.partyMenuDropConfirm(); err != nil {
		t.Fatal(err)
	}
	if len(a.state.Party) != 1 || a.state.Party[0].Name != "HERO" {
		t.Fatalf("除掉之後隊伍是 %v", a.state.Party)
	}
	if len(a.state.CharacterLibrary) != 1 || a.state.CharacterLibrary[0].Name != "HERO" {
		t.Fatalf("除掉之後人物名單是 %v", a.state.CharacterLibrary)
	}
}

// R）EMOVE 只把人移回名單，人還在。
func TestRemoveKeepsTheCharacterInTheRoster(t *testing.T) {
	a := menuApp(rookie("HERO"))
	a.state.CharacterLibrary = nil // 名單裡沒有的話要補回去
	a.keys = scriptedKeys{ebiten.KeyR: true}
	if err := a.partyMenuCommand(); err != nil {
		t.Fatal(err)
	}
	if len(a.state.Party) != 0 {
		t.Fatal("R 沒有把人移出隊伍")
	}
	if len(a.state.CharacterLibrary) != 1 || a.state.CharacterLibrary[0].Name != "HERO" {
		t.Fatalf("R 之後人物名單是 %v", a.state.CharacterLibrary)
	}
}

// M）ODIFY 的兩項限制：經驗點數為 0、身上除了金錢沒有別的物品。
func TestModifyRefusesExperiencedOrLoadedCharacters(t *testing.T) {
	a := menuApp(rookie("HERO"))
	a.state.Party[0].Experience = 1
	if err := a.modifyMember(0); err != nil {
		t.Fatal(err)
	}
	if a.statusLine != a.text(msgMenuModifyExperience) {
		t.Fatalf("有經驗值卻沒擋下來：%q", a.statusLine)
	}
	a.state.Party[0].Experience = 0
	a.state.Party[0].Inventory = []poolsave.Item{{Name: "SWORD", Raw: make([]byte, 63)}}
	if err := a.modifyMember(0); err != nil {
		t.Fatal(err)
	}
	if a.statusLine != a.text(msgMenuModifyItems) {
		t.Fatalf("身上有物品卻沒擋下來：%q", a.statusLine)
	}
}

// 1..6 指定的對象，五個指令共用。
func TestDigitsPickTheMemberTheCommandsActOn(t *testing.T) {
	a := menuApp(rookie("HERO"), rookie("TINA"), rookie("CARRY"))
	a.keys = scriptedKeys{ebiten.KeyDigit3: true}
	if err := a.Update(); err != nil {
		t.Fatal(err)
	}
	if a.menuMember != 2 {
		t.Fatalf("按 3 之後選中的是第 %d 位", a.menuMember+1)
	}
	// 隊伍裡沒有第 6 位，按 6 不該改變選擇。
	a.keys = scriptedKeys{ebiten.KeyDigit6: true}
	if err := a.Update(); err != nil {
		t.Fatal(err)
	}
	if a.menuMember != 2 {
		t.Fatal("按不存在的號碼不該改變選擇")
	}
}

// S）AVE 走的是與 F10 同一支存檔；E）XIT 直接結束。
func TestSaveAndExitEntries(t *testing.T) {
	saved := 0
	a := menuApp(rookie("HERO"))
	a.saveState = func(poolsave.State) error { saved++; return nil }
	a.keys = scriptedKeys{ebiten.KeyS: true}
	if err := a.partyMenuCommand(); err != nil {
		t.Fatal(err)
	}
	if saved != 1 {
		t.Fatalf("S 觸發了 %d 次存檔", saved)
	}
	if a.statusLine != a.text(msgMenuSaved) {
		t.Fatalf("存檔之後狀態列是 %q", a.statusLine)
	}
	a.keys = scriptedKeys{ebiten.KeyE: true}
	if err := a.partyMenuCommand(); err != ebiten.Termination {
		t.Fatalf("E 應該結束遊戲，得到 %v", err)
	}
}
