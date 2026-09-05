package main

import (
	"strings"
	"testing"

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
