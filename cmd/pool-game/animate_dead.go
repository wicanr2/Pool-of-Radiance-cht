package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 死靈術（overlay-22 `2043h`）落在隊員身上時寫進存檔的那一半（spec 098〈死靈術〉，issue #108）。
//
// 原版的隊員就是 285-byte 記錄，死靈術改的每一格都留在記錄上：跨戰鬥、跨存檔。remake 的隊員
// 平常不存 `+72h`／`+9Fh`／`+84h`，所以另外記在 `poolsave.Character.Animated`；`+10Eh`、`+10Fh`、
// 記憶陣列、生命值、狀態與效果串列本來就有欄位。營地（`DS:4954h != 5`）與戰鬥中走同一支。

// animatedMovement／animatedMorale 是 `2105h..2153h` 寫的值。
const (
	animatedMovement = gamepack.AnimatedDeadMovementRate
	// moraleHighBit 是 `2138h` 的 `80 BD 84 00 7F / 76 ..`：原本的 `+84h` 大於 7Fh 就寫 B2h，否則 B3h。
	moraleHighBit = 0x7f
)

// animatedMorale 是 `2138h..2153h`。
func animatedMorale(raw uint8) uint8 {
	if raw > moraleHighBit {
		return 0xB2
	}
	return 0xB3
}

// memberMoraleRaw 是隊員記錄的 `+84h`：死靈術寫過的照存值；NPC 讀帶著的記錄；玩家建的角色是 0
// （建角沒寫這一格，spec 096〈remake 的對應〉）。
func memberMoraleRaw(member poolsave.Character) uint8 {
	if member.Animated != nil {
		return member.Animated.Morale
	}
	if member.NPC && len(member.Record) > gamepack.MoraleOffset {
		return member.Record[gamepack.MoraleOffset]
	}
	return 0
}

// memberCreatureType 是隊員記錄的 `+9Fh`：死靈術寫過的是 4，其餘是「人」（0，tactical.go）。
func memberCreatureType(member poolsave.Character) uint8 {
	if member.Animated != nil {
		return member.Animated.CreatureType
	}
	return 0
}

// persistAnimated 把 `2043h` 對一名隊員改的欄位寫進存檔那一份：`+10Eh` 換到施法者那一邊
// （`20E8h`）、`+10Fh = 1`（`20FCh`，Quick）、`+72h`／`+9Fh`／`+84h`、清記憶陣列（`211Ah`）。
// morale 是寫下去的新值。生命值、狀態與效果串列由呼叫端處理（盤面上收場時寫回，營地直接寫）。
func persistAnimated(member *poolsave.Character, side uint8, morale uint8) {
	member.Side = side
	member.Quick = true
	member.Animated = &poolsave.AnimatedRecord{
		Movement: animatedMovement, CreatureType: gamepack.CreatureTypeUndead, Morale: morale,
	}
	for slot := range member.Memorised {
		member.Memorised[slot] = 0
	}
}

// persistAnimatedOnBoard 是戰鬥中叫起來的那幾格裡是隊員的：盤面那一份（animateDead）已經改好，
// 這裡補存檔那一份。
func (a *app) persistAnimatedOnBoard(state *tacticalState, raised []int) {
	for _, index := range raised {
		member := a.partyMemberAt(state, uint8(index))
		if member == nil {
			continue
		}
		side := uint8(1)
		if index < len(state.Friendly) && state.Friendly[index] {
			side = 0
		}
		persistAnimated(member, side, state.Morale.Raw[index])
		syncTrainedLibraryCharacter(&a.state, *member)
	}
}

// campAnimateDead 是 `2043h` 在戰鬥外：沿 `DS:5CF4h`（就是隊伍順序）走，額度是施法者等級，
// 狀態 6（死亡）而且 `+9Fh == 0` 的人叫起來。overlay-32 entry 21 `1091h` 在戰鬥外（`1097h`）直接
// 回 1，所以「那一格站不站得住」一定過。每叫起一個印「<名字> is animated」（`2037h`）。
func (a *app) campAnimateDead(caster int, option castOption, casterLevel int) string {
	casterName := strings.TrimSpace(a.state.Party[caster].Name)
	budget := casterLevel
	var lines []string
	for index := range a.state.Party {
		if budget <= 0 {
			break
		}
		member := &a.state.Party[index]
		if member.Status != gamepack.DeadState || memberCreatureType(*member) != 0 {
			continue
		}
		originalSide := member.Side & 1
		persistAnimated(member, a.state.Party[caster].Side, animatedMorale(memberMoraleRaw(*member)))
		// `217Ch`：生命值補到 `+32h`（最大值）。
		member.CurrentHP = member.MaxHP
		// `2199h`：掛效果碼 20h（等級是施法者等級，原本那一邊記在節點 `+3`，與盤面上同一個形狀）。
		level := casterLevel
		if level > gamepack.EffectLevelMask {
			level = gamepack.EffectLevelMask
		}
		node := gamepack.NewEffectNode(gamepack.AnimateDeadEffectCode, 0, uint8(level), false)
		node.MarkApplied(originalSide)
		member.Effects = storedEffects(combatEffects(member.Effects).Append(node))
		// `21C0h`：狀態 1。
		member.Status = gamepack.AnimatedState
		syncTrainedLibraryCharacter(&a.state, *member)
		lines = append(lines, fmt.Sprintf(a.text(msgEffectAnimated), strings.TrimSpace(member.Name)))
		budget--
	}
	if len(lines) == 0 {
		return fmt.Sprintf(a.text(msgCastTookEffect), casterName, option.Label)
	}
	return strings.Join(lines, " ")
}
