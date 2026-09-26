package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// AI 用身上的東西：overlay-09 entry 3（`03E3h`）與它交出去的 overlay-19 entry 8
// （`1A86h`）。spec 096〈entry 3〉，issue #71。
//
//	03F9  次數 = Roll(1, 7)                      ; 一定擲，閘門都在後面
//	0407  runtime +2 == 0                → 不用 ; 這一回合的行動權（沉默、咳嗽清掉）
//	0427  DS:[6772h + 23F5h(記錄)] == 0  → 不用 ; 對面沒人站著
//	0431  [4933h]+1CAh != 0              → 不用 ; 這一版恆為 0
//	0440  gamepack.ChooseAIItem（四道過濾、02EAh、門檻逐輪減一）
//	0519  挑到 → overlay-19 entry 8(物品, &結果)，回 1
//
// overlay-19 entry 8 在非卷軸那一支（卷軸 entry 3 根本不挑）：
//
//	1AE2  法術 = +3Dh（> 38h 減 17h）
//	1B3F  印 "<名字> uses an item"，戰鬥中再印 "Item:" 與物品名
//	1BD8  overlay-22 entry 5（0C14h）(法術, 記錄 +10Fh, 0, &結果)
//	      ; 與施法同一支；+10Fh 非零時目標由 AI 挑（DS:6A78h → overlay-13 20AEh）。
//	      ; 第三個引數 0：不印 "casts"（`0D23h`）
//	1BF4  戰鬥中而且參數表 +0Bh 非零 → overlay-25 entry 34，結果 = 1   ; 行動用掉
//	1C0E  結果非零 → gamepack.SpendAIItemUse
//
// entry 34（`266Dh`）永遠回 1，所以戰鬥中用掉就記帳，**找不找得到目標都一樣**。
// 效果不另寫：交給 cast.go 的 castSpell，與施法同一支。
//
// 隊員（Q）UICK 過的、被魅惑的、NPC）用自己身上的；怪物用 MONnITM.DAX 載進來的那一份
// （tacticalState.FoeItems，spec 142）。施法者等級照 `DS:6CB3h` 那一條：物品放的牧師／
// 法師法術一律 6 級、物品效果 12 級（overlay-25 `26F8h`，spec 098），與誰拿著無關。
//
// **不看 Magic On／Off**：`DS:6D23h` 只在 entry 4 的 `05C0h` 讀（spec 139），entry 3
// 沒有這道閘，所以交給電腦的隊員預設就會用物品。

// 訊息 msgFoeUsesItem 與轉變不死生物那幾句一起登記在 foe_turn_undead.go。

// foeUseItemPhase 是 entry 3。回傳 true 代表用了一件，這一隻的行動結束。
func (a *app) foeUseItemPhase(state *tacticalState, mover uint8, mode int) (bool, error) {
	rounds := a.rollDice(1, 7)

	index := int(mover)
	// `0407h`：runtime +2 在回合開頭設 1，只有沉默（15h，overlay-12 `0762h`）與咳嗽
	// （1Eh，`0AB3h`）的處理常式清成 0（全部 overlay 掃過，只有這兩處）。
	if state.hasEffect(index, silenceEffectCode) || state.hasEffect(index, gamepack.StinkingCloudEffectCode) {
		return false, nil
	}
	side, ok := state.sideOf(mover)
	if !ok {
		return false, nil
	}
	counts := state.sideCounts()
	opposing := counts.Foes
	if side == 1 {
		opposing = counts.Party
	}
	if opposing == 0 {
		return false, nil
	}
	items, slot := a.foeItemBearer(state, mover)
	if len(items) == 0 {
		return false, nil
	}
	caster, ok := a.foeSpellcasterFor(state, mover)
	if !ok {
		return false, nil
	}
	candidates := a.foeItemCandidates(items)
	var failure error
	chosen, found := gamepack.ChooseAIItem(candidates, rounds, func(spell, threshold uint8) bool {
		if failure != nil {
			return false
		}
		accepted, err := a.foeAcceptSpell(state, mover, caster, spell, threshold)
		if err != nil {
			failure = err
		}
		return accepted
	})
	if failure != nil || !found {
		return false, failure
	}
	state.setTacticMode(mover, mode)
	return true, a.foeUseItem(state, mover, slot, chosen, caster)
}

// foeItemBearer 回傳身上的物品串列（記錄 `+C8h`）與隊伍索引；怪物的索引是 −1。
func (a *app) foeItemBearer(state *tacticalState, mover uint8) ([]poolsave.Item, int) {
	index := int(mover)
	if index < len(state.PartySlot) {
		if slot := state.PartySlot[index]; slot >= 0 {
			if slot >= len(a.state.Party) {
				return nil, -1
			}
			return a.state.Party[slot].Inventory, slot
		}
	}
	return state.FoeItems[index], -1
}

// foeItemCandidates 把串列逐件過 AIItemSpell。卷軸由物品型別表的類別判（overlay-22
// entry 6 `31F6h`）；型別表不在就一件都不收（失敗即關閉）。
func (a *app) foeItemCandidates(items []poolsave.Item) []gamepack.AIItemCandidate {
	if a.itemTypes == nil {
		return nil
	}
	var candidates []gamepack.AIItemCandidate
	for index, item := range items {
		if len(item.Raw) <= gamepack.ItemTypeOffset {
			continue
		}
		entry, err := a.itemTypes.Entry(item.Raw[gamepack.ItemTypeOffset])
		if err != nil {
			continue
		}
		category := entry.Category()
		scroll := category >= treasure.ScrollCategoryFirst && category <= treasure.ScrollCategoryLast
		if spell, ok := gamepack.AIItemSpell(item.Raw, scroll); ok {
			candidates = append(candidates, gamepack.AIItemCandidate{Index: index, Spell: spell})
		}
	}
	return candidates
}

// foeUseItem 是 overlay-19 entry 8 的 AI 那一側：印一句、交給施法那一支、記帳。
func (a *app) foeUseItem(state *tacticalState, mover uint8, slot int,
	chosen gamepack.AIItemCandidate, caster foeSpellcaster) error {
	items, _ := a.foeItemBearer(state, mover)
	name := strings.TrimSpace(items[chosen.Index].Name)
	state.FoeLog = a.itemUseLine(state, mover, name)
	var member *poolsave.Character
	var spend func()
	if slot >= 0 {
		member = &a.state.Party[slot]
		spend = a.itemSpender(slot, chosen.Index)
	} else {
		spend = state.foeItemSpender(int(mover), chosen.Index)
	}
	spell := chosen.Spell
	label := a.spellLabel(spell)
	targets, found, err := a.foeSpellTargets(state, mover, spell, caster)
	if err != nil {
		return err
	}
	if !found {
		// 找不到目標：原版 `1BF4h` 仍然把結果改成 entry 34 的 1，照樣記帳。
		spend()
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), label))
		state.endTurnAfterAction(a.rollDice)
		return nil
	}
	casting := spellCasting{
		name:      caster.name,
		member:    member,
		partySlot: slot,
		// `1BBDh` 立 `DS:6CB3h` 之後才進 overlay-22 entry 5：物品的等級（combat_commands.go）。
		level:   itemCasterLevel(a.spellParameters[spell], caster.levels),
		consume: spend,
	}
	return a.castSpell(state, casting, castOption{Slot: -1, ID: spell, Label: label}, targets)
}

// foeItemSpender 是怪物那一側的記帳（同 spendFoeItem），只記一次。
func (state *tacticalState) foeItemSpender(index, item int) func() {
	spent := false
	return func() {
		if spent {
			return
		}
		spent = true
		items := state.FoeItems[index]
		if item < 0 || item >= len(items) {
			return
		}
		if gamepack.SpendAIItemUse(items[item].Raw) {
			state.FoeItems[index] = append(items[:item:item], items[item+1:]...)
		}
	}
}

// spendFoeItem 是 gamepack.SpendAIItemUse 加上「用完就拿掉」（overlay-25 entry 17
// `156Ah`：從串列摘掉、釋放 3Fh bytes）。
func (a *app) spendFoeItem(slot, item int) {
	if slot < 0 || slot >= len(a.state.Party) {
		return
	}
	member := &a.state.Party[slot]
	if item < 0 || item >= len(member.Inventory) {
		return
	}
	if gamepack.SpendAIItemUse(member.Inventory[item].Raw) {
		member.Inventory = append(member.Inventory[:item], member.Inventory[item+1:]...)
	}
	syncTrainedLibraryCharacter(&a.state, *member)
}
