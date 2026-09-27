package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 隊員踏出盤面（spec 150〈隊伍逃走〉，issue #111）。
//
// overlay-08 的移動常式（`09C3h..0D1Bh`，spec 058）在目的格的類別是 0（盤面外）時不擋，
// 而是問一句（exact，overlay-08 SHA-256 `932ce281…`）：
//
//	0BFD  "Flee:"（09B5h）              ; 11Dh:003Eh(0Dh, 0Ah, 0Fh, 字串)，回 'Y' 或 'N'
//	0C1A  3C 59                          ; 'Y'
//	0C24  9A 43 00 96 00                 ; overlay-13 entry 7（0C6Ch）：脫離戰場
//	0C31  3C 4E / 0C38 26 C6 05 00       ; 'N'：什麼都不做，還在移動
//
// overlay-13 entry 7 與怪物逃跑是同一支（spec 096）：對面沒有人、或自己比對面最快的還快
// 就逃掉（一樣快擲 d2），逃掉的 `+10Ch = 3`、生命值保留、體型歸零；逃不掉印
// "Escape is blocked"。兩條都結束這一個行動（entry 34）。
//
// 戰後 overlay-05 `04ADh`（exact，見 finishCombat）：
//
//	04E8  隊伍那一段有人 `+10Ch == 3` → DS:439Dh = 1
//	054E  `+10Eh == 0`、`+84h < 80h`、狀態 0／1／3 → DS:4960h = 0（不是全滅）
//	05A9  有人狀態 0／1 → DS:439Dh = 0（有人站著就不算逃走）
//	0688  439Dh 立著：@6DC7 = 81h；狀態 3 → 0、`+10Dh = 1`；**其餘的人從隊伍鏈摘掉**
//	      （06C4 `9A 2F 00 B6 00` = overlay-16 entry 3，參數 (0, 1)：隊伍人數減一）
//
// 接著 `14CAh` 照常走 `1295h` → `08E0h`（"The party has fled."，數字清 0）→ `0E85h`。
// DS:82A0h 是 0，所以 entry 2／3 不跑：沒有經驗值，也不收怪物身上的東西。

const (
	msgTacticalFleePrompt messageID = iota + 4800
)

func init() {
	for id, key := range map[messageID]string{
		msgTacticalFleePrompt: "ui.tacticalFleePrompt",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// partyFledResultCode 是 `068Ch` 寫進 @6DC7 的值（腳本拿它判戰果，spec 136）。
const partyFledResultCode = 0x81

// partyLeaveCombat 是答 Y 之後的 overlay-13 entry 7（`0C6Ch`）。
func (a *app) partyLeaveCombat(state *tacticalState, mover uint8) {
	opponents, err := state.opposingWithin(mover, 0xFF, false)
	if err != nil {
		opponents = nil
	}
	escaped := gamepack.EscapeSucceeds(len(opponents), state.speedOf(int(mover)),
		state.fastestOpponent(mover), a.rollDice)
	var result string
	if escaped {
		result = a.panelNotice(state, mover, state.say(msgFoeGotAway), noticeRowPanel, true)
		state.leaveBoard(mover, gamepack.FledState)
	} else {
		result = a.footerNotice(state, state.say(msgFoeEscapeBlocked))
	}
	state.Moving = false
	state.Budgets[mover] = 0
	a.tacticalStatus(state, a.combatantName(state, mover)+" "+result)
	state.endTurnAfterAction(a.rollDice)
}

// partyFledOutcome 是 `04ADh` 的 DS:439Dh 與 DS:4960h：fled 是「有人逃掉、沒有人站著」，
// wiped 是「隊伍那一側沒有一個不看士氣的人狀態是 0／1／3」。states 是戰後寫回的狀態。
func (a *app) partyFledOutcome(states []uint8) (fled, wiped bool) {
	wiped = true
	for index, status := range states {
		if index >= len(a.state.Party) {
			break
		}
		member := a.state.Party[index]
		if member.Side != 0 {
			continue
		}
		if status == gamepack.FledState {
			fled = true
		}
		moraleChecked := member.NPC && len(member.Record) > gamepack.MoraleOffset &&
			member.Record[gamepack.MoraleOffset] >= gamepack.MoraleCheckedBit
		if (status == 0 || status == gamepack.AnimatedState || status == gamepack.FledState) && !moraleChecked {
			wiped = false
		}
	}
	for index, status := range states {
		if index < len(a.state.Party) && a.state.Party[index].Side == 0 &&
			(status == 0 || status == gamepack.AnimatedState) {
			fled = false
		}
	}
	return fled, wiped
}

// leaveBehindAfterFleeing 是 `0688h..06D9h`：逃掉的換回狀態 0，其餘的人從隊伍摘掉。
func (a *app) leaveBehindAfterFleeing() {
	kept := a.state.Party[:0]
	for _, member := range a.state.Party {
		if member.Status != gamepack.FledState {
			continue
		}
		member.Status = 0
		syncTrainedLibraryCharacter(&a.state, member)
		kept = append(kept, member)
	}
	a.state.Party = kept
	if a.currentCharacter >= len(a.state.Party) {
		a.currentCharacter = 0
	}
	if a.eventMachine != nil {
		a.eventMachine.Memory[0x6DC7] = partyFledResultCode
	}
}
