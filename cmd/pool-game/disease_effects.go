package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 致病術的 `22h` 到期之後那一串（spec 098〈#99：收尾〉，issue #99）：`2Bh` 每 3Ch 減一點力量、`2Ch` 每 0Ah
// 扣一點生命值，到期就自己重掛，直到被治好。規則在 internal/gamepack/disease_effects.go。

const (
	// msgEffectWeakened 是 overlay-12 `10E1h`："is weakened"。
	msgEffectWeakened messageID = iota + 3000
)

func init() {
	for id, key := range map[messageID]string{
		msgEffectWeakened: "ui.effectWeakened",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// diseaseTeardown 跑一個到期節點的收尾，回傳掛完新節點的串列。hitPoints 是這個人現在的生命值
// （地圖上是角色的 CurrentHP，戰場上是盤面那一格）；力量直接改在角色上。不是這一串的碼原樣回傳。
func (a *app) diseaseTeardown(member *poolsave.Character, node gamepack.EffectNode,
	list gamepack.EffectList, hitPoints *int) gamepack.EffectList {
	if member == nil || !node.NeedsTeardown() || !gamepack.IsDiseaseEffect(node.Code) {
		return list
	}
	strength := member.Abilities[gamepack.AbilityStrength]
	if strength < 0 || strength > 0xff {
		strength = 0
	}
	result := gamepack.DiseaseTeardownOf(node, list, uint8(strength), *hitPoints)
	member.Abilities[gamepack.AbilityStrength] = int(result.Strength)
	*hitPoints = result.HitPoints
	if result.Weakened {
		a.statusLine = fmt.Sprintf(a.text(msgEffectWeakened), strings.TrimSpace(member.Name))
	}
	syncTrainedLibraryCharacter(&a.state, *member)
	return result.Effects
}

// partyEffectTeardown 是戰場上隊員的節點到期時那一下（tickEffects → effectTeardown）：先跑與地圖
// 共用的 expiredEffectTeardown，再補戰場才有的兩件事——靈魂鎚收走之後重算戰鬥數值（`07F6h` 的
// `0916h`），與致病那一串的重掛、扣血（entry 19 打 1 點，受傷打斷照走）。
//
// 中毒那一串（`0Fh`／`16h`／`4Eh`）、致病那一串與靈魂鎚的常式都不看 `+10Eh`，怪物身上一樣跑
// （poisonTeardown、foeEffectTeardown，spec 153）。
func (a *app) partyEffectTeardown(state *tacticalState, index int, node gamepack.EffectNode) {
	a.poisonTeardown(state, index, node)
	if index < 0 || index >= len(state.PartySlot) {
		return
	}
	party := state.PartySlot[index]
	if party < 0 || party >= len(a.state.Party) {
		a.foeEffectTeardown(state, index, node)
		return
	}
	a.expiredEffectTeardown(party, node, state.Effects[index])
	member := &a.state.Party[party]
	if node.Code == gamepack.SpiritualHammerEffectCode && !member.NPC {
		_ = a.applyPartyGearStats(state, index, *member)
	}
	if index < len(state.Effects) && index < len(state.HitPoints) {
		before := state.HitPoints[index]
		state.Effects[index] = a.diseaseTeardown(member, node, state.Effects[index], &state.HitPoints[index])
		a.woundCombatant(state, uint8(index), before-state.HitPoints[index])
	}
}

// foeEffectTeardown 是怪物身上的節點到期：靈魂鎚（`07F6h` 模式 1 摘鎚子、`0916h` 重算）與致病那一串
// （`0BE3h`／`10EDh`／`1177h`）。三支都只讀寫傳進來的記錄，沒有 `+10Eh` 的判斷（spec 153，exact）。
func (a *app) foeEffectTeardown(state *tacticalState, index int, node gamepack.EffectNode) {
	if !node.NeedsTeardown() || index < 0 || index >= len(state.Effects) {
		return
	}
	if node.Code == gamepack.SpiritualHammerEffectCode {
		items, slot := a.foeItemBearer(state, uint8(index))
		kept := append([]poolsave.Item(nil), items...)
		if removeSpiritualHammerFrom(&kept) {
			_ = a.storeCombatItems(state, index, slot, kept)
		}
		return
	}
	if !gamepack.IsDiseaseEffect(node.Code) || index >= len(state.HitPoints) {
		return
	}
	strength := a.foeStrength(state, index)
	before := state.HitPoints[index]
	result := gamepack.DiseaseTeardownOf(node, state.Effects[index], strength, before)
	state.Effects[index] = result.Effects
	if state.FoeStrength == nil {
		state.FoeStrength = map[int]uint8{}
	}
	state.FoeStrength[index] = result.Strength
	state.HitPoints[index] = result.HitPoints
	a.woundCombatant(state, uint8(index), before-result.HitPoints)
	if result.Weakened {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgEffectWeakened), a.combatantName(state, uint8(index))))
	}
}

// foeStrength 是怪物記錄的 `+10h`：`2Bh` 減過就用減過的值，否則讀開打時的記錄。
func (a *app) foeStrength(state *tacticalState, index int) uint8 {
	if value, ok := state.FoeStrength[index]; ok {
		return value
	}
	if monster, ok := a.stagedMonsterFor(index, state.Friendly); ok {
		return monster.Record.Raw[gamepack.AbilityStrength+recordAbilityOffset]
	}
	return 0
}

// recordAbilityOffset 是記錄裡能力值的起點（`+10h` 力量……`+15h` 魅力，spec 112〈群組 12／6／4／5〉）。
const recordAbilityOffset = 0x10
