package main

import (
	"fmt"
	"strings"


	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 鎖住的門（spec 122）。撞上一道門時原版出一個 `Bash`／`Pick`／`Knock`／`Exit`
// 的選單（overlay-14 `0DB7h`）；三個選項各有自己的閘門，成功就把 GEO 的門
// 旗標兩邊一起改成「沒鎖」。
//
// **這個選單不是 ECL 的選單**，所以不走 `cellMenuOptions` 那一條——那一條的
// 選擇要餵回 VM。這裡自己一份狀態，免得兩種選單互相踩到。

// doorMenu 是目前擋在前面的那一道門。
type doorMenu struct {
	// X／Y／Direction 是站的格子與朝向，對應原版的 `DS:6A0Bh`..`6A0Dh`。
	X, Y, Direction int
	// State 是 `0131h:0039h` 的回傳：2 鎖住、3 閂住。
	State uint8
	// Options 是這一次算出來的選項，Cursor 是游標。
	Options []string
	Cursor  int
	// Bash／Pick 對應原版的 `DS:6CD2h`／`DS:6CD3h`：試過就不再出現。
	// **誰把它們設回 1 原版還沒讀到**（spec 122），所以這裡綁在這一道門上：
	// 換一道門就是新的一份。
	Bash bool
	Pick bool
}

const (
	doorOptionBash  = "BASH"
	doorOptionPick  = "PICK"
	doorOptionKnock = "KNOCK"
	doorOptionExit  = "EXIT"
)

// beginDoorMenu 在移動被擋住時問「擋住的是不是一道鎖住的門」。
// 是就開選單並回 true；是實牆就回 false，由呼叫端印原本的訊息。
func (a *app) beginDoorMenu(x, y, direction int) bool {
	if a.initialMap == nil {
		return false
	}
	if wall, _ := a.initialMap.Grid.WallWrapped(x, y, direction); wall == 0 {
		return false
	}
	flags, _ := a.initialMap.Grid.WallDoorFlagsWrapped(x, y, direction)
	if flags != gamepack.DoorLocked && flags != gamepack.DoorBarred {
		return false
	}
	// 同一道門再撞一次要接續上一次的狀態，不能把 Bash／Pick 補回來。
	if a.door != nil && a.door.X == x && a.door.Y == y && a.door.Direction == direction {
		a.door.State = flags
	} else {
		a.door = &doorMenu{X: x, Y: y, Direction: direction, State: flags, Bash: true, Pick: true}
	}
	a.refreshDoorOptions()
	a.statusLine = a.doorMenuLine("Locked. ")
	return true
}

// doorMenuLine 是門選單那一行，游標那一項標 `>`。
//
// **游標要看得見。** 原本五處都是 `strings.Join(Options, " ")`，移動游標之後
// 字串一模一樣——玩家按了方向鍵，畫面上什麼都沒變，等於選不到 BASH 以外的
// 東西。這件事一直沒被發現，是因為門選單的按鍵分派本來就走不到
//（`main.go` 把它埋在 `cellEventPending` 裡面），按了根本沒反應。
func (a *app) doorMenuLine(prefix string) string {
	if a.door == nil || len(a.door.Options) == 0 {
		return prefix
	}
	parts := make([]string, len(a.door.Options))
	for index, option := range a.door.Options {
		if index == a.door.Cursor {
			parts[index] = "> " + option
		} else {
			parts[index] = "  " + option
		}
	}
	// **操作提示要寫在訊息上。** 門是 modal 的（spec 122）：方向鍵移游標、
	// ENTER 確定，其他鍵什麼都不做。玩家沒有理由知道這件事，而「按了沒反應」
	// 與「這個鍵不是這樣用」在畫面上長得一樣。
	return prefix + strings.Join(parts, " ") + "　←→ 選　ENTER 確定"
}

// refreshDoorOptions 重算選單。原版 `0E06h`..`0EBFh` 的三道閘門：
// `Bash` 看 `DS:6CD2h`、`Pick` 另外要隊上有賊（`0247h(6)`）、
// `Knock` 另外要有人記著敲門術（`0614h(1Fh)`）。
func (a *app) refreshDoorOptions() {
	if a.door == nil {
		return
	}
	options := make([]string, 0, 4)
	if a.door.Bash {
		options = append(options, doorOptionBash)
	}
	if a.door.Pick && a.partyHasThief() {
		options = append(options, doorOptionPick)
	}
	if a.knockCaster() >= 0 {
		options = append(options, doorOptionKnock)
	}
	options = append(options, doorOptionExit)
	a.door.Options = options
	if a.door.Cursor >= len(options) {
		a.door.Cursor = 0
	}
}

// partyHasThief 是 `0247h(6)`：隊上有沒有人的職業等級陣列第 6 格非零。
func (a *app) partyHasThief() bool {
	for _, member := range a.state.Party {
		if memberClassLevels(member)[gamepack.ThiefClassSlotIndex] > 0 {
			return true
		}
	}
	return false
}

// knockCaster 是 `0614h(1Fh)`：第一個記著敲門術的隊員，沒有就 −1。
func (a *app) knockCaster() int {
	for index, member := range a.state.Party {
		for _, slot := range member.Memorised {
			if slot&0x7f == gamepack.KnockSpellID {
				return index
			}
		}
	}
	return -1
}

// doorForcers 把隊伍換成撞門要看的兩個欄位。
func (a *app) doorForcers() []gamepack.DoorForcer {
	forcers := make([]gamepack.DoorForcer, 0, len(a.state.Party))
	for _, member := range a.state.Party {
		forcers = append(forcers, gamepack.DoorForcer{
			Strength:   uint8(member.Abilities[gamepack.AbilityStrength]),
			Percentile: uint8(member.ExceptionalStrength),
		})
	}
	return forcers
}

// openLockSkill 取這個隊員的開鎖百分比（記錄 `+78h`，spec 095）。
func openLockSkill(member poolsave.Character) uint8 {
	if gamepack.ThiefSkillOpenLocks < len(member.ThiefSkills) {
		return member.ThiefSkills[gamepack.ThiefSkillOpenLocks]
	}
	if skills, err := gamepack.ThiefSkillsFromRecord(member.Record); err == nil {
		return skills[gamepack.ThiefSkillOpenLocks]
	}
	return 0
}

// doorMenuInput 走選單：左右／上下移游標，Enter 選。
func (a *app) doorMenuInput(left, right, confirm bool) error {
	if a.door == nil || len(a.door.Options) == 0 {
		return nil
	}
	switch {
	case left:
		a.door.Cursor = (a.door.Cursor + len(a.door.Options) - 1) % len(a.door.Options)
		a.statusLine = a.doorMenuLine("Locked. ")
		return nil
	case right:
		a.door.Cursor = (a.door.Cursor + 1) % len(a.door.Options)
		a.statusLine = a.doorMenuLine("Locked. ")
		return nil
	case !confirm:
		return nil
	}
	return a.resolveDoorMenu(a.door.Options[a.door.Cursor])
}

// resolveDoorMenu 執行選到的那一項。
func (a *app) resolveDoorMenu(option string) error {
	door := a.door
	switch option {
	case doorOptionBash:
		opened, keep := gamepack.BashDoor(door.State, a.doorForcers(), a.rollDice)
		door.Bash = keep
		if opened {
			return a.openDoor("The door bursts open.")
		}
		a.refreshDoorOptions()
		a.statusLine = a.doorMenuLine("The door holds. ")
	case doorOptionPick:
		// **不論成敗都掉**：原版 `056Bh` 在迴圈外無條件清掉 `DS:6CD3h`。
		door.Pick = false
		opened := false
		for _, member := range a.state.Party {
			if memberClassLevels(member)[gamepack.ThiefClassSlotIndex] == 0 {
				continue
			}
			if gamepack.PickLockOpens(a.rollDice(1, 100), openLockSkill(member)) {
				opened = true
				break
			}
		}
		if opened {
			return a.openDoor("The lock gives way.")
		}
		a.refreshDoorOptions()
		a.statusLine = a.doorMenuLine("The lock will not turn. ")
	case doorOptionKnock:
		caster := a.knockCaster()
		if caster < 0 {
			a.statusLine = "Nobody has Knock memorised."
			return nil
		}
		// 敲門術一定成功，但吃掉那一格記憶（`0698h` 把 `+17h + 槽號` 寫 0）。
		member := &a.state.Party[caster]
		for slot, value := range member.Memorised {
			if value&0x7f == gamepack.KnockSpellID {
				member.Memorised[slot] = 0
				break
			}
		}
		syncTrainedLibraryCharacter(&a.state, *member)
		return a.openDoor(fmt.Sprintf("%s casts Knock; the door swings open.",
			strings.TrimSpace(member.Name)))
	default:
		a.door = nil
		a.statusLine = "You leave the door alone."
	}
	return nil
}

// openDoor 把門旗標兩邊一起改掉，然後關掉選單。
//
// 共用 engine 的 `UnlockDoorWrapped` 做的就是原版 `04AEh` 那兩次 `0112h`
// （這一邊 ＋ 對面那一格的反向）。原版在 `0112h` 裡是把座標**夾**在 0..15、
// 超出就整支不做，engine 是**繞**回去——貼著邊界的門會不會有差異還沒實測
// （spec 122）。
func (a *app) openDoor(line string) error {
	door := a.door
	a.initialMap.Grid.UnlockDoorWrapped(door.X, door.Y, door.Direction)
	a.door = nil
	a.statusLine = line
	return nil
}
