package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// AI 的轉變不死生物：overlay-09 entry 2（`0203h`），spec 096／111，issue #71。
//
//	0211  runtime(+108h) 的 +11h != 0 → 回 0     ; 這一場已經轉過
//	021B  記錄 +96h（牧師等級）<= 0  → 回 0
//	022E  overlay-13 entry 13（1352h）挑一隻     ; 挑不到就回 0，一顆骰都不擲
//	0241  overlay-13 entry 12（116Ah）執行，回 1 ; entry 1 接著交給 entry 34
//
// 挑與執行共用 gamepack.SelectTurnUndeadTarget／ResolveTurnUndead（玩家的 T 在原版
// 走同一支 116Ah，spec 111）。這一檔只放 runtime 那一層：名單怎麼來、旗標記在哪、
// 摧毀怎麼從盤面上收掉。
//
// runtime `+11h` 是**一場一次**：runtime 子結構在 overlay-10 `13BCh` 以 GetMem(16h)
// 配出、`13D2h` FillChar 清成 0，之後全部 overlay 只有 116Ah 的 `11AAh` 寫它（寫 1），
// 沒有別處清。

const (
	msgFoeUsesItem messageID = iota + 1700
	msgFoeTurnsUndead
	msgFoeUndeadTurned
	msgFoeUndeadDestroyed
	msgFoeTurnNothing
)

func init() {
	for id, key := range map[messageID]string{
		msgFoeUsesItem:        "ui.foeUsesItem",
		msgFoeTurnsUndead:     "ui.foeTurnsUndead",
		msgFoeUndeadTurned:    "ui.foeUndeadTurned",
		msgFoeUndeadDestroyed: "ui.foeUndeadDestroyed",
		msgFoeTurnNothing:     "ui.foeTurnNothing",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// turnDestroyedState 是被摧毀的那一隻的記錄 `+10Ch`（overlay-13 `12DCh` 寫 8）。
const turnDestroyedState = 8

// animatedUndeadColumn 是死靈術叫起來的屍體的 `+76h`（overlay-22 `2105h` 寫 2）。
const animatedUndeadColumn = 2

// foeUndead 是轉變不死生物要跨行動記著的兩個 runtime 旗標與每一格的欄位。
type foeUndead struct {
	// Columns 是怪物記錄的 `+76h`（建 roster 時記下）。不是不死生物就是 0。
	Columns map[int]int
	// Turned 是 runtime `+10h`：被轉變了。entry 13 不再挑牠；原版 entry 8 讀到它就
	// 讓牠逃（`10FFh` 寫 runtime +14h，entry 5 的 `0B9Fh` 逃跑迴圈與 `07E8h` 的逃跑
	// 分支，spec 096）——**remake 的士氣與逃跑還沒接**，被轉變的照常行動。
	Turned map[int]bool
	// Tried 是 runtime `+11h`：這一場轉過了。
	Tried map[int]bool
}

// rememberUndeadColumn 在建 roster 時記下怪物的 `+76h`。
func (state *tacticalState) rememberUndeadColumn(index int, record gamepack.MonsterRecord) {
	if state.Undead.Columns == nil {
		state.Undead.Columns = map[int]int{}
	}
	state.Undead.Columns[index] = int(record.Raw[gamepack.UndeadTurnColumnOffset])
}

// undeadColumn 是那一格的 `+76h`：死靈術叫起來的是 2，NPC 讀帶著的記錄，玩家建的
// 角色不是不死生物。
func (a *app) undeadColumn(state *tacticalState, index int) int {
	if index < len(state.States) && state.States[index] == gamepack.AnimatedState {
		return animatedUndeadColumn
	}
	if index < len(state.PartySlot) && state.PartySlot[index] >= 0 {
		slot := state.PartySlot[index]
		if slot < len(a.state.Party) && len(a.state.Party[slot].Record) > gamepack.UndeadTurnColumnOffset {
			return int(a.state.Party[slot].Record[gamepack.UndeadTurnColumnOffset])
		}
		return 0
	}
	return state.Undead.Columns[index]
}

// foeTurnUndeadPhase 是 entry 2。回傳 true 代表轉了（成不成功都算），這一隻的行動結束。
func (a *app) foeTurnUndeadPhase(state *tacticalState, mover uint8, mode int) (bool, error) {
	index := int(mover)
	if state.Undead.Tried[index] || a.turnUndeadTable == nil {
		return false, nil
	}
	caster, ok := a.foeSpellcasterFor(state, mover)
	if !ok || caster.levels[0] <= 0 {
		return false, nil
	}
	opposing, candidates, err := a.turnUndeadCandidates(state, mover)
	if err != nil {
		return false, err
	}
	if _, ok := gamepack.SelectTurnUndeadTarget(candidates); !ok {
		return false, nil
	}
	state.setTacticMode(mover, mode)
	a.turnUndead(state, mover, caster.levels[0], opposing, candidates)
	return true, nil
}

// turnUndeadCandidates 是 1352h 看的名單：`010Ah:00C0h(記錄, 0FFh)`，射程不限的對面，
// 依直線追蹤成本排。
func (a *app) turnUndeadCandidates(state *tacticalState, mover uint8) (
	[]uint8, []gamepack.TurnUndeadCandidate, error) {
	opposing, err := state.opposingWithin(mover, 0xFF, false)
	if err != nil {
		return nil, nil, err
	}
	candidates := make([]gamepack.TurnUndeadCandidate, len(opposing))
	for position, other := range opposing {
		candidates[position] = gamepack.TurnUndeadCandidate{
			Column: a.undeadColumn(state, int(other)),
			Turned: state.Undead.Turned[int(other)],
		}
	}
	return opposing, candidates, nil
}

// turnUndead 是 entry 12（`116Ah`）本身，AI（entry 2 `0241h`）與玩家的 T（overlay-08
// `0427h`）共用：`11AAh` 立 runtime +11h、擲額度與點數、逐隻轉變或摧毀，最後 entry 34
// 結束行動（`133Fh`）。挑不到任何一隻也照樣擲骰、印 "Nothing Happens"、用掉行動。
func (a *app) turnUndead(state *tacticalState, mover uint8, clericLevel int,
	opposing []uint8, candidates []gamepack.TurnUndeadCandidate) {
	index := int(mover)
	if state.Undead.Tried == nil {
		state.Undead.Tried = map[int]bool{}
	}
	state.Undead.Tried[index] = true
	result := gamepack.ResolveTurnUndead(*a.turnUndeadTable, clericLevel, candidates, a.roller)
	var lines []string
	for _, event := range result.Events {
		other := opposing[event.Index]
		if event.Outcome == gamepack.TurnTurns {
			if state.Undead.Turned == nil {
				state.Undead.Turned = map[int]bool{}
			}
			state.Undead.Turned[int(other)] = true
			lines = append(lines, state.say(msgFoeUndeadTurned, other))
			continue
		}
		state.destroyTurnedUndead(other)
		lines = append(lines, state.say(msgFoeUndeadDestroyed, other))
	}
	if !result.Turned() {
		lines = append(lines, state.say(msgFoeTurnNothing))
	}
	state.FoeLog = state.say(msgFoeTurnsUndead, mover) + " " + strings.Join(lines, " ")
	a.tacticalStatus(state, state.FoeLog)
	state.endTurnAfterAction(a.rollDice)
}

// destroyTurnedUndead 是 116Ah 的摧毀那一支（`12CEh..12E5h`）：overlay-32 entry 20
// 先把牠從戰術地圖上收掉，再寫 `+10Ch = 8`、`+10Dh = 0`。生命值原版不動。
func (state *tacticalState) destroyTurnedUndead(index uint8) {
	state.rememberFootprint(int(index))
	state.Roster[index].FootprintClass = 0
	if int(index) < len(state.Scores) {
		state.Scores[index] = 0
	}
	if int(index) < len(state.States) {
		state.States[index] = turnDestroyedState
	}
}
