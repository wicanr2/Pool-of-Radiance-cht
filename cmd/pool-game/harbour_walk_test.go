package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// bootCityParty 把一支隊伍帶到城區的地城移動狀態，供地點腳本的單點測試用。
func bootCityParty(t *testing.T, zipPath string) *app {
	t.Helper()
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.roller = diceRoller{random: rand.New(rand.NewSource(7))}
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)),
			RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
			AlignmentID: "lawful-good", Abilities: [6]int{18, 10, 10, 16, 10, 10},
			MaxHP: 60, CurrentHP: 60, PortraitHead: 1, PortraitBody: 1, IconSize: 1,
			Money: [7]uint16{3: 500, 4: 20}})
	}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
	press(application, ebiten.KeyEnter)
	press(application, ebiten.KeyB)
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			press(application, ebiten.KeyEnter)
			continue
		}
		application.keys = scriptedKeys{}
		application.Update()
	}
	return application
}

// 從南邊往北走進 (11,1)，港務長就該把完整的航線選單擺出來（spec 102）。
// 這一條走的是玩家真的會走的路徑：按鍵、格子事件、選單，不是直接跑 ECL。
func TestWalkingIntoTheHarbourMasterOpensTheRouteMenu(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application := bootCityParty(t, zipPath)
	if application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在城區，而是 GEO%d/%d",
			application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	// 要塞打完、船票清成 255 的狀態（spec 102）。
	application.eventMachine.Memory[0x4AA7] = 254
	application.eventMachine.Memory[0x4A01] = 255
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 11, 2, 0
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 200 && !application.cellWaitingMenu; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if !application.cellWaitingMenu {
		t.Fatalf("走進 (11,1) 之後沒有停在選單上：位置 (%d,%d) 朝向 %d 文字 %q",
			application.spawn.X, application.spawn.Y, application.spawn.Facing,
			application.eventText)
	}
	want := []string{"SOKAL", "EAST", "WEST", "BAY", "NONE"}
	if len(application.cellMenuOptions) != len(want) {
		t.Fatalf("選單是 %v，要 %v", application.cellMenuOptions, want)
	}
	for index, option := range want {
		if application.cellMenuOptions[index] != option {
			t.Fatalf("選單第 %d 項是 %q，要 %q", index, application.cellMenuOptions[index], option)
		}
	}
}

// 買了往東的船票再走上碼頭，隊伍就該被載到野外圖 27 的 (9,29)（spec 102）。
func TestBuyingTheEastRouteSailsIntoTheWilderness(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application := bootCityParty(t, zipPath)
	if application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在城區，而是 GEO%d/%d",
			application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	application.eventMachine.Memory[0x4AA7] = 254
	application.eventMachine.Memory[0x4A01] = 255
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 11, 2, 0
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 200 && !application.cellWaitingMenu; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if !application.cellWaitingMenu {
		t.Fatal("港務長沒有把選單擺出來")
	}
	// 游標從 SOKAL 開始，往右一格是 EAST。
	if err := press(application, ebiten.KeyArrowRight); err != nil {
		t.Fatal(err)
	}
	if application.cellMenuOptions[application.cellMenuCursor] != "EAST" {
		t.Fatalf("游標停在 %q", application.cellMenuOptions[application.cellMenuCursor])
	}
	// 選完之後還有「誰付錢」與幾段文字，一路按 RETURN 到腳本結束。
	for tick := 0; tick < 400 && application.cellEventPending; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if got := application.eventMachine.Memory[0x4AC4]; got != 1 {
		t.Fatalf("買完票 4AC4 = %d，要 1（EAST）", got)
	}
	if got := application.eventMachine.Memory[0x4A01]; got != 1 {
		t.Fatalf("買完票 4A01 = %d，要 1", got)
	}
	// 錢要真的離開付錢那個人的口袋：原版的視窗就是那個人的記錄，
	// 腳本的 `SUBTRACT 1 → @6BC3` 改的是本尊（spec 021、090）。
	if got := application.state.Party[0].Money[4]; got != 19 {
		t.Fatalf("付錢的人身上剩 %d 枚白金，要 19", got)
	}
	// 走上碼頭 (15,1)。
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 14, 1, 1
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 600 && application.spawn.Map.BlockID == 0; tick++ {
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	// 到的是 **ECL** block 27（在 ECL8）；GEO 那一頭由那個區塊自己
	// `LOAD FILES` 決定，不會是 27。
	if got := application.eventSession.CurrentBlockID(); got != 27 {
		t.Fatalf("上船之後停在 ECL block %d，要 27（GEO 是 %d/%d）",
			got, application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	if got := application.eventMachine.Memory[0x6E12]; got != 8 {
		t.Errorf("6E12 = %d，要 8", got)
	}
	if got := application.eventMachine.Memory[0x49C3]; got != 9 {
		t.Errorf("野外 X = %d，要 9", got)
	}
	if got := application.eventMachine.Memory[0x49C4]; got != 29 {
		t.Errorf("野外 Y = %d，要 29", got)
	}
}
