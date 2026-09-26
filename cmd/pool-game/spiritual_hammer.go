package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 靈魂鎚的 `17h`（overlay-12 entry 24 `07F6h`，spec 098〈#99：收尾〉，issue #99）：施放後給施法者一把鎚子，
// 節點到期時收回。規則與位元組在 internal/gamepack/spiritual_hammer.go。

const (
	// msgCastGainsItem 是 `07E8h`："Gains an item"。
	msgCastGainsItem messageID = iota + 2990
)

func init() {
	for id, key := range map[messageID]string{
		msgCastGainsItem: "ui.castGainsItem",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// grantSpiritualHammer 是 `07F6h` 的模式 0：身上沒有這把、不到 16 件就接一把在物品串列尾端
// （overlay-25 entry 18）。這一支只讀寫傳進來的記錄 `+0C8h`／`+0C7h`，沒有 `+10Eh` 的判斷
// （spec 153，exact），所以怪物施法者一樣拿到——怪物的物品串列是 FoeItems（spec 142），
// `0916h` 的重算走 storeCombatItems。鎚子沒有裝備上，重算不會改任何數值。
func (a *app) grantSpiritualHammer(state *tacticalState, cell uint8) {
	member := a.partyMemberAt(state, cell)
	if member == nil {
		items, slot := a.foeItemBearer(state, cell)
		if slot >= 0 || hasSpiritualHammer(items) || len(items) >= gamepack.SpiritualHammerItemLimit {
			return
		}
		items = append(append([]poolsave.Item(nil), items...), a.namedItem(gamepack.SpiritualHammerItem()))
		_ = a.storeCombatItems(state, int(cell), slot, items)
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastGainsItem), a.combatantName(state, cell)))
		return
	}
	if !a.giveSpiritualHammer(member) {
		return
	}
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastGainsItem), strings.TrimSpace(member.Name)))
}

// giveSpiritualHammer 是 `07F6h` 模式 0 落在隊員身上那一段，回傳有沒有給。營地施的靈魂鎚
// 走同一支（field_cast_effects.go）。
func (a *app) giveSpiritualHammer(member *poolsave.Character) bool {
	if hasSpiritualHammer(member.Inventory) ||
		len(member.Inventory) >= gamepack.SpiritualHammerItemLimit {
		return false
	}
	member.Inventory = append(member.Inventory, a.namedItem(gamepack.SpiritualHammerItem()))
	syncTrainedLibraryCharacter(&a.state, *member)
	return true
}

// removeSpiritualHammer 是 `07F6h` 的模式 1（節點到期時 entry 2 叫的收尾）：`0849h` 找到就用
// overlay-25 entry 17 摘掉。回傳有沒有摘到。
func removeSpiritualHammer(member *poolsave.Character) bool {
	return removeSpiritualHammerFrom(&member.Inventory)
}

// removeSpiritualHammerFrom 是同一件事落在任意一條物品串列上（怪物的 FoeItems 也走它）。
func removeSpiritualHammerFrom(items *[]poolsave.Item) bool {
	for index, item := range *items {
		if gamepack.IsSpiritualHammer(item.Raw) {
			*items = append((*items)[:index:index], (*items)[index+1:]...)
			return true
		}
	}
	return false
}

func hasSpiritualHammer(items []poolsave.Item) bool {
	for _, item := range items {
		if gamepack.IsSpiritualHammer(item.Raw) {
			return true
		}
	}
	return false
}
