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
// （overlay-25 entry 18）。隊員才有 remake 的物品串列；怪物那一側見 spec 098〈#99：收尾〉。
// 鎚子沒有裝備上，`0916h` 的重算不會改任何數值。
func (a *app) grantSpiritualHammer(state *tacticalState, cell uint8) {
	member := a.partyMemberAt(state, cell)
	if member == nil || !a.giveSpiritualHammer(member) {
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
	for index, item := range member.Inventory {
		if gamepack.IsSpiritualHammer(item.Raw) {
			member.Inventory = append(member.Inventory[:index:index], member.Inventory[index+1:]...)
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
