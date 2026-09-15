package main

// battleTally 是「先量再改」的尺（#22）：一場戰鬥裡每一個 tick 看一次戰術盤，
// 把狀態列與敵方紀錄拆成可以數的東西——全隊命中／未中、包紮、移動、被擋、
// 施法、誰在第幾回合倒下、睡著的敵人每回合還剩幾隻。只讀狀態，不按鍵。

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

type battleTally struct {
	state      *tacticalState
	lastStatus string
	lastFoeLog string
	lastRound  int
	// 我方（盤面索引 1..6）
	hits, misses, bandages, moves, blocked, casts int
	foeHits, foeMisses                            int
	// 倒下順序：索引 → 回合
	downs []string
	// 每回合開始時睡著的敵人數
	asleep   []int
	rounds   int
	result   string
	reported bool
}

// observe 每一個 tick 叫一次；換了一場就重新開始。
func (tally *battleTally) observe(a *app) {
	state := a.tactical
	if state == nil {
		return
	}
	if state != tally.state {
		*tally = battleTally{state: state}
	}
	if state.Round != tally.lastRound {
		tally.lastRound = state.Round
		tally.rounds = state.Round
		sleeping := 0
		for index := 1; index < len(state.Roster); index++ {
			if standing(state, index) && index < len(state.Friendly) && !state.Friendly[index] &&
				state.hasEffect(index, gamepack.SleepEffectCode) {
				sleeping++
			}
		}
		tally.asleep = append(tally.asleep, sleeping)
	}
	if state.Status != tally.lastStatus {
		tally.lastStatus = state.Status
		tally.count(state.Status, true)
	}
	if state.FoeLog != tally.lastFoeLog {
		tally.lastFoeLog = state.FoeLog
		// 「FOE n AFTER k STEPS: <狀態>」後面那一段是敵方的攻擊結果。
		if colon := strings.Index(state.FoeLog, ": "); colon >= 0 {
			tally.count(state.FoeLog[colon+2:], false)
		}
	}
	if state.Finished || state.Status == "VICTORY" || state.Status == "DEFEAT" {
		tally.result = state.Status
	}
}

func (tally *battleTally) count(line string, party bool) {
	switch {
	case strings.HasPrefix(line, "HIT "):
		if party {
			tally.hits++
		} else {
			tally.foeHits++
		}
	case strings.HasPrefix(line, "ATTACK ") && strings.Contains(line, "MISSED"):
		if party {
			tally.misses++
		} else {
			tally.foeMisses++
		}
	case strings.HasSuffix(line, " IS BANDAGED"):
		tally.bandages++
	case strings.HasPrefix(line, "MOVED "):
		if party {
			tally.moves++
		}
	case line == "BLOCKED":
		if party {
			tally.blocked++
		}
	case strings.Contains(line, "puts ") || strings.Contains(line, "casts "):
		tally.casts++
	case strings.HasSuffix(line, " IS DOWN"):
		// 打倒人的那一下只印「n IS DOWN」不印 HIT，命中要把它算回去。
		who := strings.TrimSuffix(line, " IS DOWN")
		if index, err := strconv.Atoi(who); err == nil {
			side := "foe"
			if index < len(tally.state.Friendly) && tally.state.Friendly[index] {
				side = "party"
			}
			if party && side == "foe" {
				tally.hits++
			}
			if !party && side == "party" {
				tally.foeHits++
			}
			tally.downs = append(tally.downs, fmt.Sprintf("r%d:%s%d", tally.lastRound, side, index))
		}
	}
}

// line 是一場的一行摘要。
func (tally *battleTally) line() string {
	attacks := tally.hits + tally.misses
	rate := 0
	if attacks > 0 {
		rate = tally.hits * 100 / attacks
	}
	foeAttacks := tally.foeHits + tally.foeMisses
	foeRate := 0
	if foeAttacks > 0 {
		foeRate = tally.foeHits * 100 / foeAttacks
	}
	asleep := make([]string, 0, len(tally.asleep))
	for _, n := range tally.asleep {
		asleep = append(asleep, strconv.Itoa(n))
	}
	return fmt.Sprintf("%s in %d rounds; party attacks %d/%d (%d%%), bandages %d, moves %d, blocked %d, casts %d; foe attacks %d/%d (%d%%); downs %v; asleep per round [%s]",
		tally.result, tally.rounds, tally.hits, attacks, rate, tally.bandages, tally.moves, tally.blocked,
		tally.casts, tally.foeHits, foeAttacks, foeRate, tally.downs, strings.Join(asleep, " "))
}
