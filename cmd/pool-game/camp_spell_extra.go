package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 營地施法的尾巴（spec 098〈營地施法〉，issue #108）：
//
//   - 記憶法術參數表 `+7` 為 0 時，overlay-22 entry 5 的 `0C49h..0CB7h` 印法名與
//     "is a combat-only spell..."（`0BA2h`）、"Lose it? "（`0BBCh`），overlay-26 entry 6
//     取一鍵，是 'Y' 才 `010Ah:0070h`（overlay-25 entry 16 `14ECh`）把它從記憶清掉；
//     之後 `0D1Bh` 把「放出去」與 "casts" 兩個旗標都清掉——不論 Y 或 N 都不放。
//   - 縮小術、解除魔法、恢復術的處理常式（`135Eh`、`2356h`、`2C01h`）沒有 `DS:4954h`
//     的判斷，只讀 `0A88h` 收好的表的第一格（`DS:6B89h`）。
//
// 死靈術（`2043h`，`+7` = 4）在營地同樣走得到，但它把死掉的隊員改成 AI 控制的不死生物
// （`+10Fh`、`+72h`、`+9Fh`、`+84h`），那幾格 remake 的隊員存檔沒有；沒接，寫在回報。

// 這一段訊息另開 `iota + 5200`（#108／#79），在 init 登記進 messageKeys，重號直接 panic。
const (
	msgFieldCastLoseIt messageID = iota + 5200
	msgCastReduced
	msgShopPooled
	msgShopTakeCurrency
	msgShopTakeAmount
	msgShopTakeOverloaded
	msgShopTakeTaken
	msgTempleLeaveMoney
	msgShopTakeTitle
)

func init() {
	for id, key := range map[messageID]string{
		msgFieldCastLoseIt:    "ui.fieldCastLoseIt",
		msgCastReduced:        "ui.castReduced",
		msgShopPooled:         "ui.shopPooled",
		msgShopTakeCurrency:   "ui.shopTakeCurrency",
		msgShopTakeAmount:     "ui.shopTakeAmount",
		msgShopTakeOverloaded: "ui.shopTakeOverloaded",
		msgShopTakeTaken:      "ui.shopTakeTaken",
		msgTempleLeaveMoney:   "ui.templeLeaveMoney",
		msgShopTakeTitle:      "ui.shopTakeTitle",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// askFieldCastLoseIt 是 `0C49h`：選中的是戰鬥用的記憶法術，問要不要丟掉。
func (a *app) askFieldCastLoseIt() {
	a.fieldCastStage = fieldCastLoseIt
	a.fieldCastMessage = fmt.Sprintf(a.text(msgFieldCastLoseIt), a.fieldCastChosen().Label)
}

// fieldCastLoseItInput 是 `0CA5h..0CB7h`：只有 Y 清掉記憶；其他鍵都是不要。
// 兩條路都不放出去，回到挑人那一步（overlay-15 的迴圈頂）。
func (a *app) fieldCastLoseItInput() {
	keys := []ebiten.Key{ebiten.KeyN, ebiten.KeyEscape, ebiten.KeyEnter, ebiten.KeySpace}
	if a.justPressed(ebiten.KeyY) {
		option := a.fieldCastChosen()
		if a.fieldCastCaster < len(a.state.Party) {
			caster := &a.state.Party[a.fieldCastCaster]
			if option.Slot < len(caster.Memorised) && caster.Memorised[option.Slot]&0x7f == option.ID {
				caster.Memorised[option.Slot] = 0
				syncTrainedLibraryCharacter(&a.state, *caster)
			}
		}
	} else {
		pressed := false
		for _, key := range keys {
			pressed = pressed || a.justPressed(key)
		}
		if !pressed {
			return
		}
	}
	a.fieldCastMessage = ""
	a.fieldCastStage, a.fieldCastCursor = fieldCastPickCaster, 0
	a.fieldCastOptions = nil
}

// campSpecialSpell 是縮小術、解除魔法、恢復術在戰鬥外：三支處理常式都只讀表上第一格。
// handled 為假時不是這三支。
func (a *app) campSpecialSpell(caster int, option castOption, effect gamepack.CastEffect,
	casterLevel, picked int) (string, bool) {
	if !effect.Restore && !effect.Dispel && option.ID != gamepack.SpellIDReduce {
		return "", false
	}
	params := a.spellParameters[option.ID]
	table := a.campSpellTable(params.CampTarget(), caster, picked)
	casterName := strings.TrimSpace(a.state.Party[caster].Name)
	took := fmt.Sprintf(a.text(msgCastTookEffect), casterName, option.Label)
	if len(table) == 0 {
		return took, true
	}
	index := table[0]
	subject := &a.state.Party[index]
	name := strings.TrimSpace(subject.Name)
	switch {
	case option.ID == gamepack.SpellIDReduce:
		// `1364h..13B3h`：表空就返回；`0100h:0043h(目標, 4, 0)` 豁免成功就返回；
		// `0100h:006Bh(目標, 0Ch)`（overlay-24 entry 15 `107Bh`：身上有就印 "is Cured"、
		// 經 entry 2 摘掉最早的那一個——`0Ch` 的收尾把力量還原）回 0 也返回；
		// 都過了才印 "has been reduced"（`134Dh`）。
		if a.campSavedAgainst(*subject, gamepack.SaveSpell, 0) {
			return took, true
		}
		list := combatEffects(subject.Effects)
		at, found := list.IndexOf(gamepack.EnlargeEffectCode)
		if !found {
			return took, true
		}
		node := list[at]
		list = list.RemoveAt(at)
		subject.Effects = storedEffects(list)
		a.expiredEffectTeardown(index, node, list)
		syncTrainedLibraryCharacter(&a.state, a.state.Party[index])
		return fmt.Sprintf(a.text(msgCastReduced), name), true
	case effect.Restore:
		// `2C01h`：`+74h`（欠的等級）是 0 就整支返回；與戰鬥中同一支 RestoreDrainedLevel。
		levels := memberClassLevels(*subject)
		outcome, restoredLevels, restoredExperience := gamepack.RestoreDrainedLevel(
			levels, subject.Experience, subject.DrainedLevels,
			subject.DrainedHitPoints, a.experienceTable)
		if !outcome.Restored {
			return fmt.Sprintf(a.text(msgCastNothingToRestore), name), true
		}
		subject.DrainedLevels, subject.DrainedHitPoints = outcome.DrainedLevels, outcome.DrainedHitPoints
		subject.ClassLevels = append([]uint8(nil), restoredLevels[:]...)
		subject.Experience = restoredExperience
		subject.MaxHP += outcome.HitPoints
		subject.CurrentHP += outcome.HitPoints
		subject.RawHP += outcome.HitPoints
		syncTrainedLibraryCharacter(&a.state, *subject)
		return fmt.Sprintf(a.text(msgCastRestored), name, outcome.HitPoints), true
	default:
		// 解除魔法 `2356h`：沿第一格的效果串列逐個擲（`+3` 是 FFh 的跳過），
		// 骰(1,100) <= DispelChance 就經 entry 2（`2449h` `0100h:002Ah`）摘掉——有收尾的先收尾。
		list := combatEffects(subject.Effects)
		removed := 0
		for position := 0; position < len(list); {
			node := list[position]
			if node.Undispellable() ||
				a.rollDice(1, 100) > gamepack.DispelChance(casterLevel, int(node.CasterLevel())) {
				position++
				continue
			}
			list = list.RemoveAt(position)
			removed++
			a.state.Party[index].Effects = storedEffects(list)
			a.expiredEffectTeardown(index, node, list)
			member := &a.state.Party[index]
			list = a.diseaseTeardown(member, node, list, &member.CurrentHP)
			list = a.mapPoisonTeardown(member, node, list)
		}
		a.state.Party[index].Effects = storedEffects(list)
		syncTrainedLibraryCharacter(&a.state, a.state.Party[index])
		return fmt.Sprintf(a.text(msgCastDispelled), name, removed), true
	}
}

// campSavedAgainst 是 overlay-24 entry 7（`0D61h`）在戰鬥外：1 失敗、20 成功，其餘
// 骰 ＋ `+101h`（remake 的隊員一律 0）＋ 修正，過群組 12 之後與目標值做無號比較。
func (a *app) campSavedAgainst(member poolsave.Character, category gamepack.SaveCategory,
	modifier int) bool {
	if a.savingThrows == nil || int(category) >= gamepack.SavingThrowCategories {
		return false
	}
	levels, err := partyClassLevels(member)
	if err != nil {
		return false
	}
	targets, err := a.savingThrows.TargetsForLevels(levels)
	if err != nil {
		return false
	}
	roll := a.rollDice(1, gamepack.SavingThrowDie)
	switch roll {
	case 1:
		return false
	case gamepack.SavingThrowDie:
		return true
	}
	value := gamepack.SaveRollEffects{
		Effects:      combatEffects(member.Effects),
		Category:     uint8(category),
		Constitution: uint8(member.Abilities[4]),
	}.Apply(uint8(roll + modifier))
	return targets[category] <= value
}
