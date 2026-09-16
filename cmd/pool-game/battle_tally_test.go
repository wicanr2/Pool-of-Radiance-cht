package main

// battleTally 是「先量再改」的尺（#22）：一場戰鬥裡每一個 tick 看一次戰術盤，
// 把狀態列與敵方紀錄拆成可以數的東西——全隊命中／未中、包紮、移動、被擋、
// 施法、誰在第幾回合倒下、睡著的敵人每回合還剩幾隻。只讀狀態，不按鍵。

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
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

// line 是一場的一行摘要。收場那一個 tick 常常看不到（`a.tactical` 在同一個
// Update 裡就被清掉），所以結果空著時用倒下的人數推：我方全倒是 DEFEAT，
// 否則 VICTORY。
func (tally *battleTally) line() string {
	if tally.result == "" && tally.state != nil {
		party, down := 0, 0
		for index := 1; index < len(tally.state.Friendly); index++ {
			if tally.state.Friendly[index] {
				party++
			}
		}
		for _, entry := range tally.downs {
			if strings.Contains(entry, ":party") {
				down++
			}
		}
		if party > 0 && down >= party {
			tally.result = "DEFEAT"
		} else if tally.rounds > 0 {
			tally.result = "VICTORY"
		}
	}
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

// boardLines 把戰術盤畫成文字：`.` 走得上去、`#` 牆、`Pn` 我方、`n` 敵方、
// `z` 睡著、`x` 倒下。只畫有人的那一圈外擴 pad 格。
func boardLines(state *tacticalState, pad int) []string {
	marks := map[[2]int]string{}
	minX, maxX, minY, maxY := 999, -1, 999, -1
	for index := 1; index < len(state.Roster); index++ {
		cell := state.Roster[index]
		if cell.FootprintClass == 0 {
			continue
		}
		mark := strconv.Itoa(index)
		if index < len(state.Friendly) && state.Friendly[index] {
			mark = "P" + mark
		}
		if state.hasEffect(index, gamepack.SleepEffectCode) {
			mark += "z"
		}
		if index < len(state.States) && state.States[index] >= 4 {
			mark += "x"
		}
		marks[[2]int{int(cell.X), int(cell.Y)}] = mark
		if int(cell.X) < minX {
			minX = int(cell.X)
		}
		if int(cell.X) > maxX {
			maxX = int(cell.X)
		}
		if int(cell.Y) < minY {
			minY = int(cell.Y)
		}
		if int(cell.Y) > maxY {
			maxY = int(cell.Y)
		}
	}
	lines := []string{}
	for y := minY - pad; y <= maxY+pad; y++ {
		if y < 0 || y > combat.TacticalMaxY {
			continue
		}
		line := fmt.Sprintf("y=%2d ", y)
		for x := minX - pad; x <= maxX+pad; x++ {
			if x < 0 || x >= combat.TacticalRowStride {
				continue
			}
			mark := marks[[2]int{x, y}]
			if mark == "" {
				terrain, err := state.Grid.TerrainAt(x, y)
				record, classErr := combat.CellClassAt(state.Classes, terrain)
				if err == nil && classErr == nil && record.EntryThreshold < 0xFF {
					mark = "."
				} else {
					mark = "#"
				}
			}
			line += fmt.Sprintf("%4s", mark)
		}
		lines = append(lines, line)
	}
	return lines
}
