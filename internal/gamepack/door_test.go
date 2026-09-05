package gamepack

import "testing"

// 力量開門的兩張表逐格對 spec 122（＝ AD&D 一版力量表的 Open Doors 欄，
// 狀態 3 用括號裡的數字）。這一條是硬證據對照，不是抽樣。
func TestForceDoorRollMatchesTheOpenDoorsTable(t *testing.T) {
	for _, testCase := range []struct {
		name                  string
		state, str, pct       uint8
		sides, target         int
		possible              bool
	}{
		// 狀態 2：一般鎖住。
		{"力量 3 撞鎖住的門", DoorLocked, 3, 0, 6, 1, true},
		{"力量 7", DoorLocked, 7, 0, 6, 1, true},
		{"力量 8", DoorLocked, 8, 0, 6, 2, true},
		{"力量 15", DoorLocked, 15, 0, 6, 2, true},
		{"力量 16", DoorLocked, 16, 0, 6, 3, true},
		{"力量 17", DoorLocked, 17, 0, 6, 3, true},
		{"力量 18 沒有百分位", DoorLocked, 18, 0, 6, 3, true},
		{"力量 18/50", DoorLocked, 18, 50, 6, 3, true},
		{"力量 18/51", DoorLocked, 18, 51, 6, 4, true},
		{"力量 18/99", DoorLocked, 18, 99, 6, 4, true},
		{"力量 18/00", DoorLocked, 18, 100, 6, 5, true},
		{"力量 2 太弱", DoorLocked, 2, 0, 0, 0, false},
		// 19 以上在原版的狀態 2 表裡沒有分支——照碼，不補。
		{"力量 19 碰鎖住的門原版沒有分支", DoorLocked, 19, 0, 0, 0, false},
		{"力量 25 碰鎖住的門也一樣", DoorLocked, 25, 0, 0, 0, false},
		// 狀態 3：閂住。
		{"力量 18/90 撞不開閂住的門", DoorBarred, 18, 90, 0, 0, false},
		{"力量 18/91", DoorBarred, 18, 91, 6, 1, true},
		{"力量 18/99", DoorBarred, 18, 99, 6, 1, true},
		{"力量 18/00", DoorBarred, 18, 100, 6, 2, true},
		{"力量 19", DoorBarred, 19, 0, 6, 3, true},
		{"力量 20", DoorBarred, 20, 0, 6, 3, true},
		{"力量 21", DoorBarred, 21, 0, 6, 4, true},
		{"力量 22", DoorBarred, 22, 0, 6, 4, true},
		{"力量 23", DoorBarred, 23, 0, 6, 5, true},
		{"力量 24 換成 1d8", DoorBarred, 24, 0, 8, 7, true},
		{"力量 25 一定成功", DoorBarred, 25, 0, 0, 0, true},
		{"力量 17 撞不開閂住的門", DoorBarred, 17, 0, 0, 0, false},
		// 沒鎖的門不會走到這裡。
		{"沒鎖的門", DoorUnlocked, 18, 100, 0, 0, false},
	} {
		sides, target, possible := ForceDoorRoll(testCase.state, testCase.str, testCase.pct)
		if sides != testCase.sides || target != testCase.target || possible != testCase.possible {
			t.Errorf("%s：得到 1d%d ≤ %d（可能 %v），預期 1d%d ≤ %d（可能 %v）",
				testCase.name, sides, target, possible,
				testCase.sides, testCase.target, testCase.possible)
		}
	}
}

// 撞門是**整隊各試一次**，任何一個成功就開；而只要有一個人力氣不在表內，
// `Bash` 這個選項就不再出現（原版 `DS:6CD2h = 0`）。
func TestBashDoorTriesEveryMemberAndDropsTheOption(t *testing.T) {
	always := func(count, sides int) int { return 1 }
	never := func(count, sides int) int { return 6 }

	// 一個力量 16 的人擲 1 就開了。
	opened, keep := BashDoor(DoorLocked, []DoorForcer{{Strength: 16}}, always)
	if !opened || !keep {
		t.Errorf("力量 16 擲 1 應該開得了門（開 %v、留選項 %v）", opened, keep)
	}
	// 擲不到就沒開，但選項留著。
	if opened, keep := BashDoor(DoorLocked, []DoorForcer{{Strength: 16}}, never); opened || !keep {
		t.Errorf("擲 6 不該開門而且選項要留著（開 %v、留選項 %v）", opened, keep)
	}
	// 隊上有一個力量 2 的：選項掉了，但後面力量夠的人照樣試得到。
	opened, keep = BashDoor(DoorLocked,
		[]DoorForcer{{Strength: 2}, {Strength: 16}}, always)
	if !opened {
		t.Error("前面有人力氣不夠，不該擋住後面的人")
	}
	if keep {
		t.Error("隊上有力量不在表內的人，Bash 這個選項應該掉了")
	}
	// 力量 25 撞閂住的門連骰都不擲。
	panicRoller := func(count, sides int) int { t.Fatal("力量 25 不該擲骰"); return 0 }
	if opened, _ := BashDoor(DoorBarred, []DoorForcer{{Strength: 25}}, panicRoller); !opened {
		t.Error("力量 25 撞閂住的門應該一定開")
	}
}

// 開鎖是「1d100 不大於技能百分比」。
func TestPickLockOpensOnOrUnderTheSkill(t *testing.T) {
	for _, testCase := range []struct {
		roll  int
		skill uint8
		want  bool
	}{
		{1, 25, true},
		{25, 25, true},
		{26, 25, false},
		{100, 99, false},
		{100, 100, true},
		{50, 0, false},
	} {
		if got := PickLockOpens(testCase.roll, testCase.skill); got != testCase.want {
			t.Errorf("擲 %d 對技能 %d 得到 %v，預期 %v",
				testCase.roll, testCase.skill, got, testCase.want)
		}
	}
}
