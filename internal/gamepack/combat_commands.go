package gamepack

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// 戰鬥畫面最下面那一列指令（spec 129）。
//
// 原版把它拼出來，不是印一條固定字串：overlay-08 的 DS 區從 `05E1h` 起
// 有六段 Pascal 字串，`0686h..07D4h` 逐段判斷要不要接上去。所以同一場戰鬥裡
// 不同角色看到的那一列不一樣——戰士沒有 `Cast `，身上沒東西的人沒有 `Use `。
// 照抄八項會讓玩家看到按不動的鍵。

// CombatCommandOverlay 是那六段字串所在的 overlay。
const CombatCommandOverlay = 8

// CombatCommandKey 是六段各自的身分。判斷條件要用它，不能拿字串比。
type CombatCommandKey string

const (
	CombatCommandMove      CombatCommandKey = "move"
	CombatCommandViewAim   CombatCommandKey = "view-aim"
	CombatCommandUse       CombatCommandKey = "use"
	CombatCommandCast      CombatCommandKey = "cast"
	CombatCommandTurn      CombatCommandKey = "turn"
	CombatCommandQuickDone CombatCommandKey = "quick-done"
)

// CombatCommandSegment 是一段。Text 是原版的字串本身（含原本的尾隨空白）。
type CombatCommandSegment struct {
	Key    CombatCommandKey
	Offset int
	Text   string
}

// combatCommandLayout 是六段在 overlay-08 裡的位置，逐一量自原版位元組。
var combatCommandLayout = []struct {
	key    CombatCommandKey
	offset int
}{
	{CombatCommandMove, 0x5e1},
	{CombatCommandViewAim, 0x5e7},
	{CombatCommandUse, 0x5f1},
	{CombatCommandCast, 0x5f6},
	{CombatCommandTurn, 0x5fc},
	{CombatCommandQuickDone, 0x602},
}

// ReadDOSCombatCommands 從原版 ZIP 取那六段字串。
func ReadDOSCombatCommands(zipPath string) ([]CombatCommandSegment, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	overlayFile, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		return nil, err
	}
	overlays, err := tpov.Decode(executable, overlayFile)
	if err != nil {
		return nil, err
	}
	if len(overlays) <= CombatCommandOverlay {
		return nil, fmt.Errorf("GAME.OVR has %d overlays, want more than %d",
			len(overlays), CombatCommandOverlay)
	}
	code := overlays[CombatCommandOverlay].Code
	segments := make([]CombatCommandSegment, 0, len(combatCommandLayout))
	for _, item := range combatCommandLayout {
		text, ok := pascalString(code, item.offset)
		if !ok {
			return nil, fmt.Errorf("overlay-%d has no Pascal string at %#x",
				CombatCommandOverlay, item.offset)
		}
		segments = append(segments, CombatCommandSegment{
			Key: item.key, Offset: item.offset, Text: text,
		})
	}
	return segments, nil
}
