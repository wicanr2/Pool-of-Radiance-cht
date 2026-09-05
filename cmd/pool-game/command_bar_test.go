package main

import (
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 那一列在 3-D 與野外是**兩個不同的字串**：原版 `DS:04CAh` 有 `Area`、
// `DS:04F3h` 沒有。說明書 p.21 的「在月之海沿岸的陸地上行動時，此指令完全
// 沒有作用」在位元組上就是這樣落地的。
func TestAdventureCommandListDropsAreaInTheWilderness(t *testing.T) {
	application := &app{}
	if got := strings.Join(application.adventureCommandList(), " "); got != "AREA CAST VIEW ENCAMP SEARCH LOOK" {
		t.Fatalf("3-D 那一列是 %q", got)
	}
	if len(wildernessAdventureCommands) != len(adventureCommands)-1 ||
		wildernessAdventureCommands[0] != "CAST" {
		t.Fatalf("野外那一列是 %v", wildernessAdventureCommands)
	}
}

// `S` 翻 `+594h` 的第 0 位，狀態列跟著多一段 `SEARCH`
//（說明書 p.20 的狀態列範例是 `15,4 N 12:33 SEARCH`）。
func TestSearchToggleShowsInTheStatusLine(t *testing.T) {
	radix, err := gamepack.ReadDOSTimeRadix("../../Pool of Radiance (1988).zip")
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	application := &app{mode: modeAdventure, timeRadix: radix,
		spawn: gamepack.Spawn{X: 15, Y: 4, Facing: 0}}
	if got := application.adventureStatusLine(); got != "15, 4 N 00:00" {
		t.Fatalf("關著的時候是 %q", got)
	}
	application.searchFlags ^= SearchWhileWalkingBit
	if got := application.adventureStatusLine(); got != "15, 4 N 00:00 SEARCH" {
		t.Fatalf("開著的時候是 %q", got)
	}
	application.searchFlags ^= SearchWhileWalkingBit
	if got := application.adventureStatusLine(); got != "15, 4 N 00:00" {
		t.Fatalf("再按一次之後是 %q", got)
	}
}
