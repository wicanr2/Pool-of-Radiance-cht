package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 戰後結算的兩頁（spec 150，issue #95／#103）。
//
// overlay-05 entry 1 `14CAh` 的順序（exact，spec 148）：
//
//	14E6  04ADh   狀態換算；有人站著（DS:82A0h）才呼叫 entry 2／3 發經驗值
//	14FC  1295h   NPC 從公款拿走份額；有扣到就清畫面、逐行列名、等一個鍵
//	1508  08E0h   清畫面，印標題、「Each character receives N」「experience points.」，等一個鍵
//	150C  0E85h   戰利品選單——**不論有沒有東西都開**（`0E85h` 沒有提早返回的路，
//	              選項至少是 `View Pool Exit`）
//
// 兩頁都是 `150h:0310h` 以 (1,1)–(26h,16h) 清掉整個框內再畫的：原版實測（dosgolem，
// `docs/audit/dosgolem-postcombat-screens.json`）是一整圈外框、框內沒有圖也沒有隊伍欄，
// 字從第 1 欄起，最下面那一列是 `PRESS <ENTER>/<RETURN> TO CONTINUE`。

// 這一段訊息另開 `iota + 4100`（#95／#103），在 init 登記進 messageKeys，重號直接 panic。
const (
	msgPostCombatWon messageID = iota + 4100
	msgPostCombatFled
	msgPostCombatTreasure
	msgPostCombatDuelWon
	msgPostCombatDuelLost
	msgPostCombatEachReceives
	msgPostCombatDuelistReceives
	msgPostCombatExperiencePoints
	msgPostCombatHidesShare
	msgPostCombatContinue
)

func init() {
	for id, key := range map[messageID]string{
		msgPostCombatWon:              "ui.postCombatWon",
		msgPostCombatFled:             "ui.postCombatFled",
		msgPostCombatTreasure:         "ui.postCombatTreasure",
		msgPostCombatDuelWon:          "ui.postCombatDuelWon",
		msgPostCombatDuelLost:         "ui.postCombatDuelLost",
		msgPostCombatEachReceives:     "ui.postCombatEachReceives",
		msgPostCombatDuelistReceives:  "ui.postCombatDuelistReceives",
		msgPostCombatExperiencePoints: "ui.postCombatExperiencePoints",
		msgPostCombatHidesShare:       "ui.postCombatHidesShare",
		msgPostCombatContinue:         "ui.postCombatContinue",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// postCombatReport 是 `08E0h` 與 `1295h` 兩頁要的東西，名字照原版的 DS 變數。
type postCombatReport struct {
	// fought 是 DS:439Ch：entry 2 在 `0084h` 看到一隻沒逃掉的敵方就立起來。
	fought bool
	// fled 是 DS:439Dh：`04ADh` 看到隊員逃掉（`+10Ch == 3`）而且沒有人站著。
	fled bool
	// duel 是 DS:829Ah（overlay-07 entry 26，ECL `CALL 8000h`／`8001h`）。
	duel bool
	// standing 是 DS:82A0h：有隊員站著（狀態 0 或 1），entry 2／3 才跑。
	standing bool
	// share 是 DS:829Ch：entry 2 回的每份經驗值（職業調整之前）。
	share uint32
	// hiders 是 `1295h` 列出來的 NPC 名字，依隊伍鏈的順序。
	hiders []string
}

// title 是 `08E0h` 的 `0913h..09E9h`。
func (r postCombatReport) title() messageID {
	switch {
	case r.fought || r.duel:
		switch {
		case r.duel && !r.standing:
			return msgPostCombatDuelLost // 0807h
		case r.duel:
			return msgPostCombatDuelWon // 081Fh
		}
		return msgPostCombatWon // 0836h
	case r.fled:
		return msgPostCombatFled // 0849h
	}
	return msgPostCombatTreasure // 085Dh
}

// shownShare 是第二行印的數字：輸掉決鬥（`0950h`）與逃走（`09C6h`）把參數清成 0。
func (r postCombatReport) shownShare() uint32 {
	if (r.duel && !r.standing) || (!r.fought && !r.duel && r.fled) {
		return 0
	}
	return r.share
}

// postCombatPageActive 說現在是不是停在那兩頁之一。
func (a *app) postCombatPageActive() bool {
	return a.treasureActive && a.postCombat != nil &&
		(a.treasureStage == treasureNPCShare || a.treasureStage == treasureResult)
}

// openPostCombat 從 `1295h`（有人分錢才有那一頁）開始走到 `0E85h`。
// 呼叫端已經把錢與物品交給戰利品串列（`treasureItems`、`PooledMoney`）。
func (a *app) openPostCombat(report postCombatReport) {
	a.postCombat = &report
	a.treasureActive = true
	a.treasureSelected, a.treasureCurrency, a.treasureAmount = 0, 0, ""
	a.cellEventPending, a.cellWaitingMenu = true, false
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventText, a.eventLabel = "", ""
	a.treasureStage = treasureResult
	if len(report.hiders) != 0 {
		a.treasureStage = treasureNPCShare
	}
}

// advancePostCombatPage 是那兩頁的「按一個鍵」：`146Ah` 與 `0ABDh` 都是單選項的選單
// （`11Dh:002Fh`，底列 `PRESS <ENTER>/<RETURN> TO CONTINUE`）。
func (a *app) advancePostCombatPage() {
	if a.treasureStage == treasureNPCShare {
		a.treasureStage = treasureResult
		return
	}
	a.postCombat = nil
	a.cellWaitingMenu = true
	a.enterTreasureMain()
}

// partyStanding 是 `04ADh` 的 `0593h..05AEh`：隊伍那一側有人狀態是 0 或 1。
func partyStanding(states []uint8) bool {
	for _, status := range states {
		if status == 0 || status == gamepack.AnimatedState {
			return true
		}
	}
	return false
}

// postCombatPageLines 是那一頁框內的字：每一行帶原版的欄與列（以 8×8 字格計）。
type postCombatLine struct {
	column, row int
	text        string
}

func (a *app) postCombatPageLines() []postCombatLine {
	if a.postCombat == nil {
		return nil
	}
	report := a.postCombat
	if a.treasureStage == treasureNPCShare {
		// `13E6h..141Dh`：198h:0039h(5, 5 + [bp-8], 22h, 16h, 0Ah, 1, 名字 + 1256h)，
		// `[bp-8]` 每列加 2。第一個參數是欄、第二個是列——同一族的 `198h:002Fh`
		// 在 `08E0h` 以 (1, 3) 印標題，dosgolem 量到的正是第 1 欄第 3 列。
		lines := make([]postCombatLine, 0, len(report.hiders))
		for index, name := range report.hiders {
			lines = append(lines, postCombatLine{column: 5, row: 5 + index*2,
				text: fmt.Sprintf(a.text(msgPostCombatHidesShare), name)})
		}
		return lines
	}
	receives := msgPostCombatEachReceives
	if report.duel {
		receives = msgPostCombatDuelistReceives
	}
	return []postCombatLine{
		{column: 1, row: 3, text: a.text(report.title())},
		{column: 1, row: 5, text: fmt.Sprintf(a.text(receives), report.shownShare())},
		{column: 1, row: 7, text: a.text(msgPostCombatExperiencePoints)},
	}
}

// drawPostCombatPage 畫一整圈外框、框內的字，與最下面那一列提示。
func drawPostCombatPage(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	a.drawFrame(screen, foreground, accent)
	scale := logicalWidth / 320
	cell := frameTileSize * scale
	for _, line := range a.postCombatPageLines() {
		// 基線放在字格底下兩個像素，與 footerBaseline（第 24 列，398）同一個算法。
		drawText(screen, line.text, line.column*cell, line.row*cell+cell-2, foreground)
	}
}

// postCombatHiderNames 把 `HideNPCShares` 回的索引換成名字。
func (a *app) postCombatHiderNames(indexes []int) []string {
	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		if index >= 0 && index < len(a.state.Party) {
			names = append(names, strings.TrimSpace(a.state.Party[index].Name))
		}
	}
	return names
}

// treasureExperienceEligible 是沒有戰鬥的那一場（`TREASURE → COMBAT`）有資格分經驗值的
// 隊員。原版數的是 `+10Dh`（在場）與狀態 1（`05B3h..05C9h`，exact）；戰鬥外沒有盤面，
// `+10Dh` 是上一次寫下的值：傷害把狀態打出 {0, 1} 時清成 0（spec 084 `2266h`），
// 戰後換算與神殿把人救回狀態 0 時寫回 1（`04ADh` 的 `0645h`／`0677h`、spec 115）。
// 所以「狀態 0」就是「`+10Dh` 為 1 而且狀態不是 1」（strong inference）。
func (a *app) treasureExperienceEligible() []bool {
	eligible := make([]bool, len(a.state.Party))
	for index, member := range a.state.Party {
		eligible[index] = member.Status == 0
	}
	return eligible
}

// combatExperienceEligible 是打完那一刻有資格分的隊員：還在盤面上（`+10Dh` 非 0——倒下、
// 死亡、離場都會把它清掉，spec 084／096）而且狀態不是 1。沒有上場的（開打前就倒著的，
// `enterTacticalPreview` 不擺上去）一樣沒有資格。
func combatExperienceEligible(state *tacticalState, partySize int) []bool {
	eligible := make([]bool, partySize)
	if state == nil {
		return eligible
	}
	for index := 1; index < len(state.Roster) && index < len(state.PartySlot); index++ {
		slot := state.PartySlot[index]
		if slot < 0 || slot >= partySize {
			continue
		}
		status := uint8(0)
		if index < len(state.States) {
			status = state.States[index]
		}
		eligible[slot] = state.Roster[index].FootprintClass != 0 && status != gamepack.AnimatedState
	}
	return eligible
}

// combatPartyStates 是隊伍那一側在盤面上的狀態，給 partyStanding 用。
func combatPartyStates(state *tacticalState) []uint8 {
	if state == nil {
		return nil
	}
	states := make([]uint8, 0, len(state.PartySlot))
	for index := 1; index < len(state.Roster) && index < len(state.PartySlot); index++ {
		if state.PartySlot[index] < 0 || index >= len(state.States) {
			continue
		}
		// 擺不上去的（體型 0）還是在鏈上、狀態照舊；原版 `0593h` 只看狀態。
		states = append(states, state.States[index])
	}
	return states
}

// foughtAnyFoe 是 DS:439Ch：entry 2 `0068h..0084h` 對 `+10Eh == 1` 而且 `+10Ch != 3` 的
// 任何一隻立旗——倒下、投降都算，只有逃掉的不算。
func foughtAnyFoe(state *tacticalState) bool {
	if state == nil {
		return false
	}
	for index := 1; index < len(state.Roster) && index < len(state.Friendly); index++ {
		if state.Friendly[index] {
			continue
		}
		if index < len(state.States) && state.States[index] == gamepack.FledState &&
			state.Roster[index].FootprintClass == 0 {
			continue
		}
		return true
	}
	return false
}

// duelCall 是 ECL `CALL 8000h`／`8001h`：overlay-03 `3026h` 減 `7FFFh` 得 1／2，以參數
// 1／0 呼叫 overlay-07 entry 26（`1AB3h`，`0045h:00A2h`）。那一支把 DS:829Ah 立起來、
// @6DE6 寫成參數，並把 `DS:5CF0h`（目前角色）以外的隊員 `+10Dh` 清成 0——只有那一個人上場。
//
// 參數 1（競技場，`ecl3/11 9CA6h`）另外把目前角色複製一份、取名 `ROLF` 放進敵方當對手
// （`1B20h..1CDFh`）；remake 還沒有「角色記錄變成敵方」這一條，所以那一支不接（spec 150）。
const (
	duelArenaCall    = 0x8000
	duelChampionCall = 0x8001
	// duelArenaAddress 是 @6DE6（`[4937h] + 5CCh`）。
	duelArenaAddress = 0x6DE6
)

// startChampionDuel 是 `CALL 8001h`（參數 0）：只有目前角色上場。
func (a *app) startChampionDuel() {
	a.duel = true
	if a.eventMachine != nil {
		a.eventMachine.Memory[duelArenaAddress] = 0
	}
}

// duelDeploys 說 index 那一個隊員這一場上不上場。
func (a *app) duelDeploys(index int) bool {
	return !a.duel || index == a.currentCharacter
}

// finishDuelState 是決鬥收場：overlay-03 `197Dh` 清 DS:829Ah、主流程 `1606h` 清 @6DE6。
// `04ADh` 的 `07A9h..0800h` 把狀態 0／1 的人 `+10Dh` 寫回 1、倒地（5）改成昏迷（4）；
// 後者與打贏時的換算同一條（storeCombatHitPoints），前者在 remake 沒有對應的欄位。
func (a *app) finishDuelState() {
	a.duel = false
	if a.eventMachine != nil {
		a.eventMachine.Memory[duelArenaAddress] = 0
	}
}

// duelChampionStanding 是決鬥那一條的 DS:82A0h（`0748h..077Dh`）：`+10Dh == 1`、
// `+10Ch == 0`、不是敵方——也就是上場的那一個人打完還站著。
func duelChampionStanding(state *tacticalState) bool {
	if state == nil {
		return false
	}
	for index := 1; index < len(state.Roster) && index < len(state.PartySlot); index++ {
		if state.PartySlot[index] < 0 || index >= len(state.States) {
			continue
		}
		if state.Roster[index].FootprintClass != 0 && state.States[index] == 0 {
			return true
		}
	}
	return false
}
