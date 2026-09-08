package main

import (
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 原版的指令列是**算出來的**（spec 129）：`Use ` 要身上有東西、`Cast ` 要
// 記著法術、`Turn ` 要有牧師等級。照抄八項會讓玩家看到按不動的鍵。
func TestCombatCommandBarFollowsTheOriginalConditions(t *testing.T) {
	segments := []gamepack.CombatCommandSegment{
		{Key: gamepack.CombatCommandMove, Text: "Move "},
		{Key: gamepack.CombatCommandViewAim, Text: "View Aim "},
		{Key: gamepack.CombatCommandUse, Text: "Use "},
		{Key: gamepack.CombatCommandCast, Text: "Cast "},
		{Key: gamepack.CombatCommandTurn, Text: "Turn "},
		{Key: gamepack.CombatCommandQuickDone, Text: "Quick Done"},
	}
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	newApp := func(member poolsave.Character) *app {
		state := &tacticalState{
			Roster:    make([]combat.CombatantCell, 2),
			PartySlot: []int{-1, 0},
			Mover:     1,
		}
		return &app{tactical: state, combatCommands: segments,
			state: poolsave.State{Schema: poolsave.Schema,
				Party: []poolsave.Character{member}}}
	}
	// 戰士：身上沒東西、沒記法術、牧師等級 0——就是原版第 49 幀那個角色。
	fighter := poolsave.Character{Name: "HERO",
		ClassLevels: append([]uint8(nil), levels...),
		Memorised:   make([]uint8, gamepack.MemorisedSpellSlots)}
	if got := newApp(fighter).combatCommandBar(); got != "Move View Aim Quick Done" {
		t.Errorf("戰士看到 %q，原版那一幕是 `MOVE VIEW AIM QUICK DONE`", got)
	}
	// 牧師：有東西、記了法術、牧師等級 6——六段全上。
	clericLevels := append([]uint8(nil), levels...)
	clericLevels[gamepack.ClassSlotCleric] = 6
	cleric := poolsave.Character{Name: "PRIEST", ClassLevels: clericLevels,
		Memorised: make([]uint8, gamepack.MemorisedSpellSlots),
		Inventory: []poolsave.Item{{Name: "MACE"}}}
	cleric.Memorised[0] = gamepack.SpellIDBless
	if got := newApp(cleric).combatCommandBar(); got != "Move View Aim Use Cast Turn Quick Done" {
		t.Errorf("牧師看到 %q，六段應該全上", got)
	}
	// 只差記憶法術這一格：`Cast ` 要掉，其餘不動。
	forgot := cleric
	forgot.Memorised = make([]uint8, gamepack.MemorisedSpellSlots)
	got := newApp(forgot).combatCommandBar()
	if strings.Contains(got, "Cast") {
		t.Errorf("沒記法術還看得到 Cast：%q", got)
	}
	if !strings.Contains(got, "Turn") || !strings.Contains(got, "Use") {
		t.Errorf("掉了不該掉的段：%q", got)
	}
}
