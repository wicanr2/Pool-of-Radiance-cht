package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/geometry"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// doorFixture 是一張只有一道門的空圖：(4,4) 的東邊有一道 detail 為 flags 的牆。
func doorFixture(flags uint8, party []poolsave.Character) *app {
	grid := geometry.Grid{}
	// 方向 2 是東，索引 1（WallDirections／DetailDirections 的順序是北東南西）。
	grid.Cells[4][4].WallDirections[1] = 1
	grid.Cells[4][4].DetailDirections[1] = flags
	grid.Cells[4][5].WallDirections[3] = 1
	grid.Cells[4][5].DetailDirections[3] = flags
	application := &app{initialMap: &gamepack.GeometryMap{Grid: grid}}
	application.state.Party = party
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 4, 4, 1
	return application
}

// 力量 18/00 的戰士撞得開鎖住的門（1d6 ≤ 5），撞開之後兩邊的旗標都變 1。
func TestBashingOpensALockedDoorOnBothSides(t *testing.T) {
	application := doorFixture(gamepack.DoorLocked, []poolsave.Character{
		{Name: "BRUTE", Abilities: [6]int{18, 0, 0, 0, 0, 0}, ExceptionalStrength: 100},
	})
	application.roller = fixedRoller{1}

	if !application.beginDoorMenu(4, 4, 2) {
		t.Fatal("鎖住的門沒有開選單")
	}
	if got := strings.Join(application.door.Options, " "); got != "BASH EXIT" {
		t.Errorf("選單是 %q，預期 BASH EXIT（隊上沒有賊、沒人記著敲門術）", got)
	}
	if err := application.resolveDoorMenu(doorOptionBash); err != nil {
		t.Fatal(err)
	}
	if application.door != nil {
		t.Error("門開了選單還在")
	}
	for _, side := range []struct{ x, y, dir int }{{4, 4, 2}, {5, 4, 6}} {
		flags, ok := application.initialMap.Grid.WallDoorFlagsWrapped(side.x, side.y, side.dir)
		if !ok || flags != gamepack.DoorUnlocked {
			t.Errorf("(%d,%d) 朝向 %d 的旗標是 %d（ok=%v），預期 1",
				side.x, side.y, side.dir, flags, ok)
		}
	}
	if !application.initialMap.Grid.CanMoveDungeonWrapped(4, 4, 2) {
		t.Error("門開了還走不過去")
	}
}

// 撞不開的時候門留著、選單也留著；力量不在表內的人會把 BASH 這個選項弄掉。
func TestBashingKeepsTheDoorAndCanLoseTheOption(t *testing.T) {
	application := doorFixture(gamepack.DoorLocked, []poolsave.Character{
		{Name: "WEAK", Abilities: [6]int{9, 0, 0, 0, 0, 0}},
	})
	application.roller = fixedRoller{6}
	application.beginDoorMenu(4, 4, 2)
	if err := application.resolveDoorMenu(doorOptionBash); err != nil {
		t.Fatal(err)
	}
	if application.door == nil {
		t.Fatal("撞不開卻把選單關了")
	}
	if !application.door.Bash {
		t.Error("力量 9 在表內，BASH 不該掉")
	}
	if application.initialMap.Grid.CanMoveDungeonWrapped(4, 4, 2) {
		t.Error("撞不開卻走得過去")
	}

	// 力量 2 不在表內：原版順手把 `DS:6CD2h` 清成 0。
	application = doorFixture(gamepack.DoorLocked, []poolsave.Character{
		{Name: "FRAIL", Abilities: [6]int{2, 0, 0, 0, 0, 0}},
	})
	application.roller = fixedRoller{1}
	application.beginDoorMenu(4, 4, 2)
	application.resolveDoorMenu(doorOptionBash)
	if application.door == nil || application.door.Bash {
		t.Error("力量 2 撞完之後 BASH 應該從選單上掉了")
	}
	if got := strings.Join(application.door.Options, " "); got != "EXIT" {
		t.Errorf("選單是 %q，預期只剩 EXIT", got)
	}
}

// 敲門術一定成功，而且吃掉那一格記憶。
func TestKnockOpensTheDoorAndSpendsTheSlot(t *testing.T) {
	application := doorFixture(gamepack.DoorBarred, []poolsave.Character{
		{Name: "MAGE", Abilities: [6]int{9, 0, 0, 0, 0, 0},
			Memorised: []uint8{gamepack.SpellIDMagicMissile, gamepack.KnockSpellID}},
	})
	application.roller = fixedRoller{6}
	if !application.beginDoorMenu(4, 4, 2) {
		t.Fatal("閂住的門沒有開選單")
	}
	if got := strings.Join(application.door.Options, " "); got != "BASH KNOCK EXIT" {
		t.Errorf("選單是 %q，預期 BASH KNOCK EXIT", got)
	}
	if err := application.resolveDoorMenu(doorOptionKnock); err != nil {
		t.Fatal(err)
	}
	if !application.initialMap.Grid.CanMoveDungeonWrapped(4, 4, 2) {
		t.Error("敲門術之後門還是關的")
	}
	if got := application.state.Party[0].Memorised[1]; got != 0 {
		t.Errorf("敲門術那一格記憶是 %d，應該被用掉變 0", got)
	}
	if got := application.state.Party[0].Memorised[0]; got != gamepack.SpellIDMagicMissile {
		t.Errorf("動到了別的記憶格：%d", got)
	}
}

// 開鎖一扇門只能試一次——不論成敗，PICK 都從選單上消失。
func TestPickingALockIsAOneShotAttempt(t *testing.T) {
	// 直接給 ClassLevels（訓練過的角色就是這樣帶著八個等級，spec 097），
	// 免得測試綁在建角的職業字串上。第 6 格是賊。
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ThiefClassSlotIndex] = 9
	thief := poolsave.Character{
		Name:        "TINA",
		Abilities:   [6]int{9, 0, 0, 0, 0, 0},
		ClassLevels: levels,
		ThiefSkills: []uint8{80, 0, 65, 80, 66, 30, 98, 45}, // 開鎖 0：一定失敗
	}
	application := doorFixture(gamepack.DoorLocked, []poolsave.Character{thief})
	application.roller = fixedRoller{1}
	application.beginDoorMenu(4, 4, 2)
	if !strings.Contains(strings.Join(application.door.Options, " "), doorOptionPick) {
		t.Fatal("隊上有賊卻沒有 PICK")
	}
	application.resolveDoorMenu(doorOptionPick)
	if application.door == nil {
		t.Fatal("開鎖技能 0 不該開得了門")
	}
	if application.door.Pick {
		t.Error("撬過一次之後 PICK 應該消失")
	}
	if strings.Contains(strings.Join(application.door.Options, " "), doorOptionPick) {
		t.Errorf("選單裡還有 PICK：%q", strings.Join(application.door.Options, " "))
	}
}

// 撞不開的門把遭遇選單的方向鍵與 ENTER 全吃掉（spec 122，2026-09-11 的隨機
// 探索撞到）。
//
// `a.door` 是**持久狀態**：同一道門再撞一次要接續上一次的 BASH／PICK 額度，
// 所以撞不開的時候它留著，這是對的。錯的是輸入分派把「門的狀態還在」當成
// 「門選單正在等玩家回答」——`a.door != nil` 那一條排在格子選單前面而且直接
// return，於是方向鍵移不動遭遇選單的游標，ENTER 又跑回去撞門。
//
// 隨機探索撞到的樣子是「格子選單卡住：GEO5/4 (13,3) 游標 0／
// [COMBAT WAIT FLEE PARLAY]、狀態列 Locked. BASH EXIT、這一格答過 0 次」——
// 答過 0 次就是線索：治具連一次都沒能把選擇送出去。
func TestALingeringDoorDoesNotSwallowTheCellMenuKeys(t *testing.T) {
	application := doorFixture(gamepack.DoorLocked, []poolsave.Character{
		{Name: "WEAK", Abilities: [6]int{9, 0, 0, 0, 0, 0}},
	})
	application.roller = fixedRoller{6}
	application.beginDoorMenu(4, 4, 2)
	if err := application.resolveDoorMenu(doorOptionBash); err != nil {
		t.Fatal(err)
	}
	if application.door == nil {
		t.Fatal("撞不開的門應該留著——這一條要測的是它留著之後的事")
	}

	// 門還沒處理完就跳出遭遇：走路撞上門、腳本同一格排了遭遇，原版走得到。
	// 隨機探索撞到的就是 `The monsters are 0 squares away.` 那一刻。
	application.mode, application.introDone = modeAdventure, true
	state := &encounterState{distance: 0, slowest: 12, fastest: 12}
	// 結果碼 1 那一族：WAIT 只印字再問一次，不需要 ECL 機器（spec 078）。
	// 這一條要測的是按鍵走到哪裡，不是遭遇怎麼收場。
	for index := range state.kinds {
		state.kinds[index] = 1
	}
	application.encounter = state
	application.showEncounterMenu()
	if got := application.cellMenuOptions; len(got) != 4 {
		t.Fatalf("貼身的遭遇選單是 %v，預期四項", got)
	}

	if err := press(application, ebiten.KeyArrowRight); err != nil {
		t.Fatal(err)
	}
	if application.cellMenuCursor != 1 {
		t.Errorf("按了右鍵，遭遇選單的游標是 %d，應該是 1（門把方向鍵吃掉了）",
			application.cellMenuCursor)
	}

	// ENTER 也要落在遭遇選單上，不可以跑回去撞門。
	doorCursor, doorOptions := application.door.Cursor, strings.Join(application.door.Options, " ")
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.door == nil {
		t.Fatal("ENTER 把門關掉了——那一下是玩家在回答遭遇選單")
	}
	if application.door.Cursor != doorCursor ||
		strings.Join(application.door.Options, " ") != doorOptions {
		t.Errorf("ENTER 跑回去動門了：游標 %d→%d、選項 %q→%q",
			doorCursor, application.door.Cursor,
			doorOptions, strings.Join(application.door.Options, " "))
	}
}

// 門選單自己的方向鍵要動得了游標（spec 122 的〈兩種選單的輸入優先權〉）。
//
// 這一條是上一條的正對照：如果門選單本來就收不到按鍵，「門把遭遇選單的按鍵
// 吃掉」這個說法就站不住——那會是另一個缺陷。
func TestTheDoorMenuTakesTheArrowKeys(t *testing.T) {
	application := doorFixture(gamepack.DoorLocked, []poolsave.Character{
		{Name: "WEAK", Abilities: [6]int{9, 0, 0, 0, 0, 0}},
	})
	application.roller = fixedRoller{6}
	application.mode, application.introDone = modeAdventure, true
	if !application.beginDoorMenu(4, 4, 2) {
		t.Fatal("鎖住的門沒有開選單")
	}
	if got := strings.Join(application.door.Options, " "); got != "BASH EXIT" {
		t.Fatalf("選單是 %q，預期 BASH EXIT", got)
	}
	if err := press(application, ebiten.KeyArrowRight); err != nil {
		t.Fatal(err)
	}
	if application.door.Cursor != 1 {
		t.Fatalf("按了右鍵，門選單的游標是 %d，應該是 1", application.door.Cursor)
	}
	// 游標要看得見。移到 EXIT 之後那一行必須跟著變——不然玩家按了方向鍵
	// 畫面什麼都沒動，等於選不到 BASH 以外的東西。
	if got := application.statusLine; got != "Locked.   BASH > EXIT" {
		t.Errorf("狀態列是 %q，游標沒有標在 EXIT 上", got)
	}
}

// 最小重現：腳本跑完、字留在框裡（cellTextSticky），門選單同時開著。
// 城堡那條測試卡住時的狀態就是這個：eventText "HE SCRAMBLES OUT THE DOOR."、
// cellEventPending false、門 [BASH EXIT] 游標停在 EXIT，而 ENTER 關不掉它。
func TestDoorMenuEnterWorksWhileTextIsStickyInTheBox(t *testing.T) {
	for _, probe := range []struct {
		name      string
		eventText string
		sticky    bool
	}{
		{"框裡沒字", "", false},
		{"有字、不 sticky", "HE SCRAMBLES OUT THE DOOR.", false},
		{"有字、sticky", "HE SCRAMBLES OUT THE DOOR.", true},
	} {
		t.Run(probe.name, func(t *testing.T) {
			doorMenuEnterProbe(t, probe.eventText, probe.sticky)
		})
	}
}

func doorMenuEnterProbe(t *testing.T, eventText string, sticky bool) {
	t.Helper()
	application := doorFixture(gamepack.DoorLocked, []poolsave.Character{
		{Name: "WEAK", Abilities: [6]int{9, 0, 0, 0, 0, 0}},
	})
	application.roller = fixedRoller{6}
	application.mode, application.introDone = modeAdventure, true
	if !application.beginDoorMenu(4, 4, 2) {
		t.Fatal("鎖住的門沒有開選單")
	}
	// 腳本跑完之後留在框裡的字。
	application.eventText, application.cellTextSticky = eventText, sticky

	if err := press(application, ebiten.KeyArrowLeft); err != nil {
		t.Fatal(err)
	}
	if got, want := application.door.Cursor, len(application.door.Options)-1; got != want {
		t.Fatalf("左鍵之後游標在 %d，EXIT 在 %d", got, want)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.door != nil {
		t.Errorf("游標在 EXIT 上按了 ENTER，門選單還開著：%v 游標 %d，狀態列 %q",
			application.door.Options, application.door.Cursor, application.statusLine)
	}
}
