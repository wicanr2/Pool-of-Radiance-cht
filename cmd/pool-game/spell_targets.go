package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 施法的時間、打斷與收目標（spec 098〈施法時間與打斷〉〈收目標〉，issue #72／#73）。
//
// 玩家按 C 與 AI 施法在原版走同一條：
//
//	overlay-08 entry 4 `0410h`  C → overlay-13 entry 19（`23F9h`）(0, 0, &結果)
//	overlay-09 `0654h`          AI → 同一支 (法術, 1, &結果)
//	23F9h  施法時間 = 參數表 +0Ch ÷ 3
//	       0 → overlay-22 entry 5（`0C14h`）當場放，接著 entry 34 結束行動
//	       否則 印 "Begins Casting"、runtime +0 = 法術、先攻 +3 扣掉時間（不夠扣留 1），
//	            **不呼叫 entry 34**：這一格還在先攻排序裡
//	overlay-08 `031Bh`          輪到玩家時 runtime +0 非零 → 清掉它、
//	                            overlay-22 entry 5 (法術, 0, 1, &結果) 放出去、entry 34
//	overlay-13 `04E8h..054Bh`   受傷 > 0：runtime +1 清 0；+0 非零就印 "lost a spell"、
//	                            overlay-25 entry 16（`14ECh`）把那一格從記憶清掉、+0 清 0
//	overlay-08 `072Fh`          runtime +1 為 0 的回合，指令列不接 "Cast "
//
// 所以**瞄準是在放出去的那一刻**（overlay-22 entry 5 裡的 `DS:6A78h` → `20AEh`），
// 不是開始施法的時候；記憶也是放出去時才清（`0E8Dh` 的 `14ECh`）。
//
// 瞄準挑不到目標時 overlay-22 `0EB3h..0F0Bh`：玩家被問 "Abort Spell? "，按 Y
// 印 "Spell Aborted" 並把法術從記憶清掉（`0F06h` 的 `14ECh`），其他鍵重挑；
// AI 不問，直接 "Spell Aborted" 並清掉。兩者之後都由 entry 34 結束行動。
//
// remake 這一側與 AI 共用 foe_cast.go 的 `Casting.Pending`（runtime +0）與
// `woundedThisRound`（runtime +1 的受傷那一半）：開始施法、輪到時放出或丟失都
// 讀寫同一份。

// 這一段訊息另開 `iota + 1720`（#72），在 init 登記進 messageKeys，重號直接 panic。
const (
	msgCastAbortPrompt messageID = iota + 1720
	msgCastAborted
	msgCastAlreadyTargeted
	msgSpellOutOfRange
	msgCastCannotNow
	msgCastPickNext
)

func init() {
	for id, key := range map[messageID]string{
		msgCastAbortPrompt:     "ui.castAbortPrompt",
		msgCastAborted:         "ui.castAborted",
		msgCastAlreadyTargeted: "ui.castAlreadyTargeted",
		msgSpellOutOfRange:     "ui.spellOutOfRange",
		msgCastCannotNow:       "ui.castCannotNow",
		msgCastPickNext:        "ui.castPickNext",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// spellTargets 是 overlay-13 `20AEh` 收好的那一份：`DS:6B85h` 那張表（筆數
// `DS:6B88h`）、`DS:6CADh`／`6CAEh` 的中心點與 `DS:677Eh` 的範圍旗標。
type spellTargets struct {
	// List 是收到的戰鬥員，順序就是原版表的順序。
	List []uint8
	// X、Y 是 `DS:6CADh`／`6CAEh`：最後一次挑到的那一格，範圍法術的中心。
	X, Y int
	// Area 是 `DS:677Eh`：這一份是以一點為中心收出來的。
	Area bool
	// Chosen 為假代表模式 0（`20FDh`，打施法者自己）或 remake 沒接瞄準的那幾種，
	// 效果那一側照舊自己挑。
	Chosen bool
}

// first 是表上的第一個（`DS:6B89h`）。解除魔法那一支只讀這一格（spec 098）。
func (targets spellTargets) first() (uint8, bool) {
	if !targets.Chosen || len(targets.List) == 0 {
		return 0, false
	}
	return targets.List[0], true
}

// singleSpellTarget 是「挑了一個人」那一份，finishCast 與舊的呼叫端用。
func (state *tacticalState) singleSpellTarget(target uint8) spellTargets {
	targets := spellTargets{List: []uint8{target}, Chosen: true}
	if int(target) < len(state.Roster) {
		targets.X, targets.Y = int(state.Roster[target].X), int(state.Roster[target].Y)
	}
	return targets
}

// spellAreaMembers 是 `20AEh` 的 `2254h`（與火球術 `2687h`）：以 (x, y) 為中心、
// 體型 1、方向 FFh（不限）、預算 budget 跑一次 overlay-31 `0912h`，把收到的每一個
// 戰鬥員依序收進表。**不分敵我**——`0912h` 收的是盤面上所有人，陣營篩選是
// overlay-25 entry 32 另外做的，這裡沒有那一層。
func (state *tacticalState) spellAreaMembers(x, y, budget int) ([]uint8, error) {
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return nil, err
	}
	if budget < 0 {
		budget = 0
	}
	cells, err := combat.NearbyCells(combat.NearbyRequest{
		Map: snapshot.Map, Classes: snapshot.Classes, Cells: snapshot.Cells,
		Class: 1, Facing: combat.DirectionUnset, Budget: uint16(budget),
		BaseX: uint8(x), BaseY: uint8(y),
	})
	if err != nil {
		return nil, err
	}
	members := make([]uint8, 0, len(cells))
	for _, cell := range cells {
		index := cell.CombatantIndex
		if index == 0 || int(index) >= len(state.Roster) || state.Roster[index].FootprintClass == 0 {
			continue
		}
		members = append(members, index)
	}
	return members, nil
}

// spellRangeToCell 是瞄準時那一格離施法者多遠：overlay-25 `2591h` 的路徑成本除以二
// （與 tacticalRange 同一把尺），施法者自己那一格是 0。走不到回 false。
func (state *tacticalState) spellRangeToCell(from uint8, x, y int) (int, bool) {
	if int(from) >= len(state.Roster) {
		return 0, false
	}
	source := state.Roster[from]
	if int(source.X) == x && int(source.Y) == y {
		return 0, true
	}
	trace, err := combat.TraceMovement(state.Grid, state.Classes,
		int(source.X), int(source.Y), x, y, 0xFF)
	if err != nil || !trace.Complete {
		return 0, false
	}
	return int(trace.Cost) / 2, true
}

// castAim 是玩家這一側 `20AEh` 的進度。
type castAim struct {
	option castOption
	plan   gamepack.SpellTargetPlan
	// reach 是 `352Ch` 的 `[bp+12h]`：overlay-22 entry 3 的射程，0 與 FFh 墊成 1
	// （`359Fh..35ADh`）。`2AF2h`：距離大於它的目標沒有 "Target" 可按。
	reach int
	// picks 是模式 1..7 已經收到的那幾個，remaining 是還要挑幾個（`22D2h` 的計數）。
	picks     []uint8
	remaining int
	// release 為真代表這是開始施法之後輪到時的放出（overlay-08 `031Bh`）。
	release bool
	// abort 為真代表正在等 "Abort Spell?" 的 Y／N（overlay-22 `0EC6h`）。
	abort bool
}

// castAborting 說現在是不是在等 Abort Spell 的回答。
func (a *app) castAborting() bool { return a.castAim != nil && a.castAim.abort }

// casterLevelOf 是目前行動的隊員施這一條的等級（overlay-25 `26F8h`）。
func (a *app) casterLevelOf(state *tacticalState, id uint8) int {
	index, ok := a.moverPartyIndex(state.Mover)
	if !ok || int(id) >= len(a.spellParameters) {
		return 0
	}
	levels := memberClassLevels(a.state.Party[index])
	return gamepack.CasterLevelFor(a.spellParameters[id],
		int(levels[gamepack.ClassSlotCleric]), int(levels[gamepack.ClassSlotMagicUser]), false)
}

// aimSpell 是 overlay-22 entry 5 呼叫 `20AEh` 那一步的玩家這一側。
//
// 模式 0 不挑（`20FDh`）。**remake 另外兩種不挑**：模式 0Ah（整邊，`0F35h` 那一支的
// 範圍沒讀）與模式 8（閃電束，`2B75h` 走 `2919h` 的射線，幾何沒逐格對過），效果那一側
// 照舊自己決定作用在誰身上。
func (a *app) aimSpell(option castOption, release bool) error {
	state := a.tactical
	if state == nil {
		return nil
	}
	if int(option.ID) >= len(a.spellParameters) {
		return a.releaseSpell(option, spellTargets{}, release)
	}
	params := a.spellParameters[option.ID]
	plan := params.TargetPlan()
	mode := params.TargetMode()
	if plan.Kind == gamepack.SpellTargetKindSelf || mode == gamepack.SpellTargetBolt ||
		mode == gamepack.SpellTargetWholeSide {
		return a.releaseSpell(option, spellTargets{}, release)
	}
	reach := params.Range(a.casterLevelOf(state, option.ID))
	if reach == 0 || reach == 0xFF {
		reach = 1
	}
	remaining := plan.Count
	if plan.Kind != gamepack.SpellTargetKindCount {
		remaining = 1
	}
	a.castAim = &castAim{option: option, plan: plan, reach: reach, remaining: remaining, release: release}
	if !a.beginCastTargeting(option) {
		// 盤面上一個站著的都沒有：沒得挑，照舊交給效果那一側。
		a.castAim = nil
		return a.releaseSpell(option, spellTargets{}, release)
	}
	return nil
}

// confirmSpellCell 是 `352Ch` 選定一格（Next／Prev 停著的人，或 Manual 游標那一格）。
// occupant 是站在那一格的戰鬥員，空格是 0。
func (a *app) confirmSpellCell(x, y int, occupant uint8) error {
	state, aim := a.tactical, a.castAim
	if state == nil || aim == nil {
		return nil
	}
	// `30DEh`：空格子只有範圍法術（`1E09h` 的第三個引數是 1）選得到。
	if occupant == 0 && !aim.plan.PointAim {
		return nil
	}
	// `2AF2h`／`302Ah`：超出射程沒有 "Target" 可按，停在瞄準裡。
	distance, reachable := state.spellRangeToCell(state.Mover, x, y)
	if !reachable || distance > aim.reach {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgSpellOutOfRange),
			aim.option.Label, distance, aim.reach))
		return nil
	}
	switch aim.plan.Kind {
	case gamepack.SpellTargetKindArea:
		members, err := state.spellAreaMembers(x, y, aim.plan.AreaBudget)
		if err != nil {
			return err
		}
		return a.releaseSpell(aim.option, spellTargets{List: members, X: x, Y: y,
			Area: true, Chosen: true}, aim.release)
	case gamepack.SpellTargetKindCount:
		for _, earlier := range aim.picks {
			if earlier == occupant {
				// `2363h`：玩家挑到重複的印 "Already been targeted"，不算數、再挑一次。
				a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastAlreadyTargeted), occupant))
				return nil
			}
		}
		aim.picks = append(aim.picks, occupant)
		aim.remaining--
		if aim.remaining <= 0 {
			return a.releaseSpell(aim.option, spellTargets{List: aim.picks, X: x, Y: y,
				Chosen: true}, aim.release)
		}
		// 下一個：`22F3h` 再進一次 `1E09h`，瞄準從頭開始。
		a.castManual = false
		a.beginCastTargeting(aim.option)
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastPickNext),
			aim.option.Label, len(aim.picks), len(aim.picks)+aim.remaining))
		return nil
	default:
		return a.releaseSpell(aim.option, spellTargets{List: []uint8{occupant}, X: x, Y: y,
			Chosen: true}, aim.release)
	}
}

// cancelSpellPick 是瞄準時按 Exit（`1E09h` 回 0）。模式 1..7 是「要的數量減一」
// （`23BBh`），減完還有收到的就照那幾個放；其餘模式與一個都沒收到的，`20AEh`
// 回報失敗，overlay-22 `0EC0h` 問 "Abort Spell?"。
func (a *app) cancelSpellPick() error {
	aim := a.castAim
	if aim == nil {
		return nil
	}
	a.castManual = false
	if aim.plan.Kind == gamepack.SpellTargetKindCount {
		aim.remaining--
		if aim.remaining > 0 {
			a.beginCastTargeting(aim.option)
			return nil
		}
		if len(aim.picks) > 0 {
			last := aim.picks[len(aim.picks)-1]
			targets := a.tactical.singleSpellTarget(last)
			targets.List = aim.picks
			return a.releaseSpell(aim.option, targets, aim.release)
		}
	}
	aim.abort = true
	a.castTargeting = false
	a.tacticalStatus(a.tactical, a.text(msgCastAbortPrompt))
	return nil
}

// castAbortInput 是 "Abort Spell? " 的回答（overlay-22 `0EDEh` 的 `011D:003Eh`）：
// 只認 Y；其他鍵 `0EE5h` 跳回 `0D4Dh` 再瞄一次，`20AEh` 從頭收（`20BBh` 清表）。
func (a *app) castAbortInput() error {
	aim := a.castAim
	switch {
	case a.justPressed(ebiten.KeyY):
		return a.abortSpell()
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyEnter):
		aim.abort = false
		aim.picks = nil
		aim.remaining = aim.plan.Count
		if aim.plan.Kind != gamepack.SpellTargetKindCount {
			aim.remaining = 1
		}
		if !a.beginCastTargeting(aim.option) {
			return a.abortSpell()
		}
	}
	return nil
}

// abortSpell 是 overlay-22 `0EE7h..0F0Bh`：印 "Spell Aborted"、以 `14ECh` 把那一格
// 從記憶清掉，然後由 entry 34 結束這個行動。
func (a *app) abortSpell() error {
	state, aim := a.tactical, a.castAim
	a.castAim, a.castTargeting, a.castManual = nil, false, false
	if state == nil || aim == nil {
		return nil
	}
	if caster, ok := a.foeSpellcasterFor(state, state.Mover); ok {
		a.foeForgetSpell(state, caster, aim.option.ID)
	}
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastAborted), state.Mover))
	state.endTurnAfterAction(a.rollDice)
	a.statusLine = state.Status
	if state.Finished {
		return a.finishCombat(state.Outcome)
	}
	return nil
}

// releaseSpell 把收好的目標交給效果那一側。開始施法之後的放出，記憶那一格是
// 放出去時才找：`14ECh` 清的是陣列裡**第一個**等於這個編號的格子。
func (a *app) releaseSpell(option castOption, targets spellTargets, release bool) error {
	a.castAim, a.castTargeting, a.castManual = nil, false, false
	state := a.tactical
	if release && state != nil {
		index, ok := a.moverPartyIndex(state.Mover)
		if !ok {
			return nil
		}
		option.Slot = memorisedSlotOf(a.state.Party[index], option.ID)
		if option.Slot < 0 {
			// 開始施法之後記憶被動過（例如紮營重記），沒有那一格可放。
			a.tacticalStatus(state, a.text(msgCastNothingReady))
			state.endTurnAfterAction(a.rollDice)
			if state.Finished {
				return a.finishCombat(state.Outcome)
			}
			return nil
		}
	}
	return a.finishCastTargets(option, targets)
}

// memorisedSlotOf 是 overlay-25 entry 16（`14ECh`）找的那一格：第一個整個 byte 等於
// 編號的。找不到回 −1。
func memorisedSlotOf(member poolsave.Character, id uint8) int {
	for slot, value := range member.Memorised {
		if value == id {
			return slot
		}
	}
	return -1
}

// beginCasting 是 overlay-13 `24E7h..2552h`：印 "Begins Casting"、runtime +0 記下
// 這一條、先攻分數扣掉施法時間（不夠扣就留 1）。**不呼叫 entry 34**，所以這一格
// 還在排序裡，重選之後輪到牠時才放出去。與 foe_cast.go 的 foeCastPhase 讀寫同一份
// `Casting.Pending`。
func (state *tacticalState) beginCasting(mover, spell, cost uint8, label string,
	roll func(count, sides int) int) error {
	index := int(mover)
	if state.Casting.Pending == nil {
		state.Casting.Pending = map[int]uint8{}
	}
	state.Casting.Pending[index] = spell
	if index < len(state.Scores) {
		score, err := combat.ApplyCastingTimeInitiative(state.Scores[index], cost)
		if err != nil {
			return err
		}
		state.Scores[index] = score
	}
	state.Status = state.say(msgFoeBeginsCasting, mover, label)
	state.Moving = false
	state.selectActor(roll)
	if state.Mover == 0 {
		state.endRound(roll)
	}
	return nil
}

// beginPlayerCasting 是玩家那一側的「開始施法」：目標與記憶都留到放出去那一刻。
func (a *app) beginPlayerCasting(option castOption, cost uint8) error {
	state := a.tactical
	if err := state.beginCasting(state.Mover, option.ID, cost, option.Label, a.rollDice); err != nil {
		return err
	}
	a.statusLine = state.Status
	if state.Finished {
		return a.finishCombat(state.Outcome)
	}
	return nil
}

// pendingSpellTurn 是 overlay-08 entry 4 的 `031Bh`：輪到玩家、runtime +0 還記著
// 開始施法的法術，就先清掉它再放出去（瞄準在這一刻），放完結束行動。回傳 true 代表
// 這一影格的按鍵已經用在這裡。
//
// 放出去之前受過傷：原版在受傷那一刻（overlay-13 `0509h..0547h`）就印 "lost a spell"、
// 清掉記憶與 +0，所以輪到時 +0 已經是 0，這一格照常出指令列（而 "Cast " 因為 +1 是 0
// 不會出現）。remake 的受傷判斷與 AI 同一支（woundedThisRound），在輪到時才看。
func (a *app) pendingSpellTurn(state *tacticalState) (bool, error) {
	mover := state.Mover
	if mover == 0 || a.castAim != nil || a.castTargeting || a.castOpen {
		return false, nil
	}
	spell := state.Casting.Pending[int(mover)]
	if spell == 0 {
		return false, nil
	}
	if _, ok := a.moverPartyIndex(mover); !ok {
		return false, nil
	}
	delete(state.Casting.Pending, int(mover))
	if state.woundedThisRound(int(mover)) {
		if caster, ok := a.foeSpellcasterFor(state, mover); ok {
			a.foeForgetSpell(state, caster, spell)
		}
		a.tacticalStatus(state, state.say(msgFoeLostSpell, mover))
		return true, nil
	}
	return true, a.aimSpell(castOption{Slot: -1, ID: spell, Label: a.spellLabel(spell)}, true)
}
