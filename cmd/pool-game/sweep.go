package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 橫掃（spec 154）：overlay-13 entry 10（`0E8Ch`）。三條出手的路都先問它——走進敵人那一格
// （overlay-08 `0DA0h`）、電腦（overlay-09 `0E80h`）、瞄準按 T（overlay-13 `2CD0h`）——成立
// 就不照一般攻擊打，這一次行動用掉（`0DA9h` 寫出參／`0E8Fh`、`2CDDh` 叫 overlay-25 entry 34）。
// 反應攻擊（spec 059）直接進攻擊包裝，不經過這一支。

const (
	// msgStatusSweeps 是 `0E85h` 的 "sweeps"：entry 20(攻擊者, "sweeps", 0Ah, 1)（`0F77h..0F92h`）。
	msgStatusSweeps messageID = iota + 4500
)

func init() {
	for id, key := range map[messageID]string{
		msgStatusSweeps: "ui.statusSweeps",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// sweepState 記著 runtime `+5` 被清掉的那幾格。攻擊核心（overlay-13 `1404h`）打完在
// `17D0h` 把攻擊者的 `+5` 寫 0，回合開頭（entry 1 `0076h`）才從記錄 `+6Bh` 抄回來——
// 所以同一回合先出過手（例如反應攻擊，spec 059）的人，輪到自己時掃不了。
type sweepState struct {
	// round[i] 是 i 最後一次出手的回合數，0 是還沒出過手。
	round map[uint8]int
}

func (sweep *sweepState) struck(round int, index uint8) {
	if sweep.round == nil {
		sweep.round = map[uint8]int{}
	}
	sweep.round[index] = round
}

func (sweep *sweepState) spent(round int, index uint8) bool {
	last, ok := sweep.round[index]
	return ok && last == round
}

// sweepLimit 是這一格的 runtime `+5`：每回合初始化 overlay-13 entry 1 在 `0076h` 從記錄 `+6Bh`
// 抄來，`+6Bh` 是 entry 7 的尾段寫的（開打時對每一筆都跑，spec 147）。remake 的隊員沒有記錄，
// 用同一條規則從戰士等級與種族碼算。這一回合出過手就是 0（`17D0h`）。
func (a *app) sweepLimit(state *tacticalState, index uint8) uint8 {
	if state.Sweep.spent(state.Round, index) {
		return 0
	}
	if int(index) < len(state.PartySlot) {
		if slot := state.PartySlot[index]; slot >= 0 && slot < len(a.state.Party) {
			member := a.state.Party[slot]
			if member.NPC && len(member.Record) == poolsave.NPCRecordSize {
				raw := member.Record
				return gamepack.SweepLimit(raw[gamepack.ClassLevelOffset+gamepack.ClassSlotFighter], raw[gamepack.RaceOffset])
			}
			race, _ := creation.RaceDOSCode(member.RaceID)
			levels := memberClassLevels(member)
			return gamepack.SweepLimit(levels[gamepack.ClassSlotFighter], race)
		}
	}
	if monster, ok := a.stagedMonsterFor(int(index), state.PartySlot, state.Friendly); ok {
		raw := monster.Record.Raw[:]
		return gamepack.SweepLimit(raw[gamepack.ClassLevelOffset+gamepack.ClassSlotFighter], raw[gamepack.RaceOffset])
	}
	return 1
}

// sweep 是 `0E8Ch`：成立就一個一個砍、回 true；不成立回 false，呼叫端照一般攻擊打。
func (a *app) sweep(state *tacticalState, target uint8) (bool, error) {
	mover := state.Mover
	if int(mover) >= len(state.HitDice) || int(target) >= len(state.HitDice) {
		return false, nil
	}
	gear, items, _, err := a.missileGear(state, mover)
	if err != nil {
		return false, err
	}
	// runtime `+113h`：這一相位第一形態的次數（`0D29h` 算的那一個，射擊武器照齊射）。
	swings, form2, err := a.volleySwings(state, mover, gear, items)
	if err != nil {
		return false, err
	}
	remaining := len(swings) - form2
	if remaining > 0xFF {
		remaining = 0xFF
	}
	distance, reachable := state.tacticalRange(mover, target)
	if !reachable {
		distance = -1
	}
	side, ok := state.sideOf(mover)
	if !ok {
		return false, nil
	}
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return false, err
	}
	here := state.Roster[mover]
	nearby, err := combat.OpposingNearbyAt(snapshot, mover, here.X, here.Y, 1, 1-side, state.sideOf)
	if err != nil {
		return false, err
	}
	targets := gamepack.SweepTargets(gamepack.SweepInput{
		Remaining: uint8(remaining), Limit: a.sweepLimit(state, mover),
		TargetHitDice: state.HitDice[target], Distance: distance,
		Nearby: nearby, Target: target,
		HitDice: func(who uint8) uint8 {
			if int(who) >= len(state.HitDice) {
				return 0xFF
			}
			return state.HitDice[who]
		},
	})
	if len(targets) == 0 {
		return false, nil
	}
	a.panelNotice(state, mover, state.say(msgStatusSweeps), noticeRowPanel, true)
	// 每一格：`+113h = 1`（`1048h`）再進攻擊包裝，彈藥 NULL（`105Ah..1062h`）。第二形態
	// `+114h` 不動，所以第一個人還吃得到這一相位剩下的第二形態，之後就是 0。
	first := state.AttackForms[mover][0]
	for index, who := range targets {
		var cut []combat.DamageDice
		extra := 0
		if index == 0 {
			cut = append(cut, swings[:form2]...)
			extra = form2
		}
		cut = append(cut, first)
		if err := a.resolveAttackSwings(state, mover, who, cut, extra); err != nil {
			return true, err
		}
	}
	return true, nil
}
