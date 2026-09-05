package main

import (
	"path/filepath"
	"testing"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 狀態列的格式與原版逐字相同：`14, 1 W 00:00`。朝向那個字母是 spec 076 的
// 0 北 1 東 2 南 3 西換過來的——換錯的話與原版兩張基準圖對不上
//（`02` 是 `14, 1 W`、`05` 是 `11, 2 S`）。
func TestAdventureStatusLineMatchesTheDOSFormat(t *testing.T) {
	cases := []struct {
		spawn gamepack.Spawn
		want  string
	}{
		{gamepack.Spawn{X: 14, Y: 1, Facing: 3}, "14, 1 W 00:00"},
		{gamepack.Spawn{X: 11, Y: 2, Facing: 2}, "11, 2 S 00:00"},
		{gamepack.Spawn{X: 0, Y: 4, Facing: 0}, "0, 4 N 00:00"},
		{gamepack.Spawn{X: 7, Y: 9, Facing: 1}, "7, 9 E 00:00"},
	}
	for _, item := range cases {
		application := &app{spawn: item.spawn}
		if got := application.adventureStatusLine(); got != item.want {
			t.Fatalf("狀態列是 %q，預期 %q", got, item.want)
		}
	}
}

// 面板的 AC 是檯面值（`60 − 內部值`）。建角基礎 AC 內部值是 50，所以空手
// 且敏捷 10 的角色是 10——與原版兩張基準圖上的 `HERO 10` 相同。
func TestPartyPanelRowsUseTabletopArmourClass(t *testing.T) {
	member := poolsave.Character{Name: "HERO", CurrentHP: 6}
	member.Abilities[dexterityAbilityIndex] = 10
	application := &app{state: poolsave.State{Party: []poolsave.Character{member}}}
	rows := application.partyPanelRows()
	if len(rows) != 1 {
		t.Fatalf("面板列數 %d，預期 1", len(rows))
	}
	if rows[0].Name != "HERO" || rows[0].HitPoints != 6 {
		t.Fatalf("第一列是 %+v", rows[0])
	}
	if rows[0].ArmourClass != gamepack.ArmourClassScale-creationArmorClassInternal {
		t.Fatalf("AC 是 %d，預期 %d", rows[0].ArmourClass,
			gamepack.ArmourClassScale-creationArmorClassInternal)
	}
}

// 出處那幾列移到 F1 之後不能就這樣消失。
func TestProvenanceLinesMovedIntoHelp(t *testing.T) {
	event := gamepack.InitialEvent{MonsterID: 12}
	application := &app{spawn: gamepack.DOSInitialSpawn(), initialEvent: &event}
	lines := application.adventureProvenanceLines()
	want := map[string]bool{
		"GEO / WALL SOURCE: EXACT":         false,
		"VIEW TRAVERSAL: STRONG INFERENCE": false,
		"MOVE POLICY: PENDING / DISABLED":  false,
		"FIRST EVENT: ROLF / MONSTER 12":   false,
	}
	for _, line := range lines {
		if _, ok := want[line]; ok {
			want[line] = true
		}
	}
	for line, seen := range want {
		if !seen {
			t.Fatalf("說明頁少了 %q（現有：%q）", line, lines)
		}
	}
}

// 走一步加一分，轉向不加。原版量出來的是 `00:00 → 00:01 → 00:02`
//（`workplace/oracle/screens` 那一輪從導覽結束的 `(0,4)` 起走，spec 118）。
func TestGameClockAdvancesOneMinutePerStep(t *testing.T) {
	radix, err := gamepack.ReadDOSTimeRadix(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	application := &app{timeRadix: radix, spawn: gamepack.Spawn{X: 14, Y: 4, Facing: 3}}
	if got := application.adventureStatusLine(); got != "14, 4 W 00:00" {
		t.Fatalf("起點是 %q", got)
	}
	application.advanceGameMinute()
	if got := application.adventureStatusLine(); got != "14, 4 W 00:01" {
		t.Fatalf("走一步之後是 %q", got)
	}
	application.advanceGameMinute()
	if got := application.adventureStatusLine(); got != "14, 4 W 00:02" {
		t.Fatalf("走兩步之後是 %q", got)
	}
	// 分進到時：逐位上限表說一分的個位滿十進到十位，十位滿六進到時。
	for step := 0; step < 58; step++ {
		application.advanceGameMinute()
	}
	if got := application.adventureStatusLine(); got != "14, 4 W 01:00" {
		t.Fatalf("走滿六十步之後是 %q，預期進位到 01:00", got)
	}
}
