package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
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
	// 有人之後換一組。原版隊伍非空的截圖上是九項：
	// CREATE／DROP／MODIFY／VIEW／ADD／REMOVE／SAVE／BEGIN／EXIT
	// ——**沒有 LOAD，也沒有 TRAIN**。兩張截圖都不在訓練所裡，所以 `T` 的
	// 那一道（`DS:06D4h`，見 training_gate.go）不成立。
	a = menuApp(rookie("HERO"))
	keys = nil
	for _, entry := range a.visiblePartyMenuEntries() {
		keys = append(keys, entry.key)
	}
	want = []ebiten.Key{ebiten.KeyC, ebiten.KeyD, ebiten.KeyM,
		ebiten.KeyV, ebiten.KeyA, ebiten.KeyR, ebiten.KeyS, ebiten.KeyB, ebiten.KeyE}
	if len(keys) != len(want) {
		t.Fatalf("有隊伍時列出 %d 項，預期 %d 項", len(keys), len(want))
	}
	for index := range want {
		if keys[index] != want[index] {
			t.Fatalf("有隊伍時第 %d 項是 %v，預期 %v", index, keys[index], want[index])
		}
	}
	// L）OAD 只在空隊伍出現。
	for _, entry := range a.visiblePartyMenuEntries() {
		if entry.key == ebiten.KeyL {
			t.Fatal("隊伍有人時不該列出 L）OAD SAVED GAME")
		}
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

// `T)RAIN` 只在訓練所裡出現（spec 008）。原版的閘門是
// `es:[4937h]+550h > 0`，也就是 ECL 位址 `6DA8h` 非零——全遊戲只有
// ECL3 區塊 11 的四道門寫它。
func TestTrainOnlyAppearsInsideATrainingHall(t *testing.T) {
	a := menuApp(rookie("HERO"))
	if a.trainingHallOpen() {
		t.Fatal("沒進訓練所就開著")
	}
	listed := func() bool {
		for _, entry := range a.visiblePartyMenuEntries() {
			if entry.key == ebiten.KeyT {
				return true
			}
		}
		return false
	}
	if listed() {
		t.Error("沒進訓練所卻列出 T）RAIN")
	}
	a.eventMachine = &eclvm.Machine{Memory: map[uint16]uint16{}}
	for _, mask := range []uint16{TrainingMaskMagicUser, TrainingMaskCleric,
		TrainingMaskThief, TrainingMaskFighter} {
		a.eventMachine.Memory[trainingMaskAddress] = mask
		if !a.trainingHallOpen() || !listed() {
			t.Errorf("遮罩 %#x 該讓 T）RAIN 出現", mask)
		}
	}
	a.eventMachine.Memory[trainingMaskAddress] = 0
	if listed() {
		t.Error("離開訓練所之後 T）RAIN 還在")
	}
}

// 除錯碼：`J` 再輸入 `STING`，原版回 "I Understand, master..." 並讓
// `T)RAIN` 無條件出現（overlay-16 `049Ah` → `ds:466Eh = 1`）。
func TestTheStingCodeUnlocksTraining(t *testing.T) {
	a := menuApp(rookie("HERO"))
	a.stingPrompt = true
	text := &scriptedTextKeys{scriptedKeys: scriptedKeys{}, chars: []rune("sting")}
	a.keys = text
	if !a.stingInput() {
		t.Fatal("輸入列沒吃掉按鍵")
	}
	if a.stingBuffer != stingPassword {
		t.Fatalf("緩衝區是 %q，預期 %q", a.stingBuffer, stingPassword)
	}
	text.scriptedKeys[ebiten.KeyEnter] = true
	a.stingInput()
	if !a.stingUnlocked {
		t.Fatal("對的碼沒有解開")
	}
	if a.statusLine != stingAcknowledge {
		t.Errorf("回話是 %q，預期 %q", a.statusLine, stingAcknowledge)
	}
	if !a.trainingHallOpen() {
		t.Error("解開之後訓練所還是關的")
	}

	// 錯的碼什麼都不做。原版比的是整串。
	b := menuApp(rookie("HERO"))
	b.stingPrompt = true
	wrong := &scriptedTextKeys{scriptedKeys: scriptedKeys{}, chars: []rune("stin")}
	b.keys = wrong
	b.stingInput()
	wrong.scriptedKeys[ebiten.KeyEnter] = true
	b.stingInput()
	if b.stingUnlocked {
		t.Error("錯的碼卻解開了")
	}
}
