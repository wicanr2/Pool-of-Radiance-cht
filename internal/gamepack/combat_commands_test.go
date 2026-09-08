package gamepack

import (
	"path/filepath"
	"testing"
)

// 那六段字串接起來就是說明書 p.37 那一列。少讀一段的症狀是玩家看不到某個
// 指令，而畫面本身看起來完全正常。
func TestCombatCommandSegmentsMatchTheOriginalOverlay(t *testing.T) {
	segments, err := ReadDOSCombatCommands(
		filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	want := []struct {
		key  CombatCommandKey
		text string
	}{
		{CombatCommandMove, "Move "},
		{CombatCommandViewAim, "View Aim "},
		{CombatCommandUse, "Use "},
		{CombatCommandCast, "Cast "},
		{CombatCommandTurn, "Turn "},
		{CombatCommandQuickDone, "Quick Done"},
	}
	if len(segments) != len(want) {
		t.Fatalf("讀到 %d 段，原版是 %d 段", len(segments), len(want))
	}
	joined := ""
	for index, segment := range segments {
		if segment.Key != want[index].key || segment.Text != want[index].text {
			t.Errorf("第 %d 段是 %s %q，預期 %s %q",
				index, segment.Key, segment.Text, want[index].key, want[index].text)
		}
		joined += segment.Text
	}
	// 說明書下冊 p.37 印的那一列。
	if joined != "Move View Aim Use Cast Turn Quick Done" {
		t.Errorf("接起來是 %q", joined)
	}
}
