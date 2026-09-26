package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 營地（戰鬥外）施法的掛效果（spec 098〈營地施法：同一支 `08BCh`，表換成 `0A88h`〉，
// issue #100）。
//
// 原版營地的 C)AST（overlay-15 `0512h`）與物品頁的 Use 都呼叫 overlay-22 entry 5，與戰鬥中
// 同一支：處理常式照樣把法術編號與四個覆寫參數推給 `08BCh`，`08BCh` 照樣對表上每一格擲豁免、
// 算 `07C7h` 的持續、問群組 9、叫 overlay-24 entry 20 掛參數表 `+0Ah`。不同的只有：
//
//   - 表：戰鬥外 `DS:6A78h` 指 overlay-22 entry 4（`0A88h`），依參數表 `+7` 收施法者自己、
//     挑一個隊員或整隊（gamepack.CampTarget）。
//   - `07C7h` 的 `082Ch`：編號 `3Fh` 在 `DS:4954h != 5` 時是 (Roll(1, 10) + 10) × 10。
//   - 祝福的 `0F88h`：不在戰鬥中就不問「貼身有沒有敵人」。
//
// 掛上去的節點就長在角色身上（`+7Fh`），開打時跟著人進盤面（tactical.go 抄 `Effects`）、
// 收場寫回（storeCombatEffects），戰鬥外走路與休息由 advancePartyEffects 倒數到期。
// 所以這裡只要把節點掛到 `a.state.Party[i].Effects`。

// campAttachesEffect 說這一條在營地走「只掛效果」那一路（戰鬥中是 castSpell 的 default、
// 模式 0Ah 與力量那一組）。治療、解除、復原、死靈、解除魔法等仍走 applyFieldEffect 或沒接。
func campAttachesEffect(effect gamepack.CastEffect, params gamepack.SpellParameters) bool {
	switch params.CampTarget() {
	case gamepack.CampTargetSelf, gamepack.CampTargetPick, gamepack.CampTargetParty:
	default:
		return false
	}
	if params.SaveRule() != gamepack.SaveRuleNone || params.RequiresAttackRoll() {
		// 營地放得出去的法術裡只有縮小術（`+8` = 1）要擲豁免；它的收表與豁免沒接（spec 098）。
		return false
	}
	if effect.Damage > 0 || effect.Heal > 0 || len(effect.RemoveEffects) > 0 ||
		effect.Restore || effect.AnimateDead || effect.Dispel || effect.Cloud ||
		effect.Ray != nil || effect.SleepBudget > 0 || effect.PersonOnly ||
		effect.HitPointBudgetFromCaster || effect.SaveModifierByTargetCount {
		return false
	}
	return effect.EffectCode != 0 || params.EffectCode() != 0
}

// campSpellTable 是 `0A88h` 收好的表（隊伍索引）。picked 是 "Cast Spell on whom" 挑到的人。
func (a *app) campSpellTable(kind gamepack.CampTargetKind, caster, picked int) []int {
	switch kind {
	case gamepack.CampTargetSelf:
		return []int{caster}
	case gamepack.CampTargetPick:
		if picked >= 0 && picked < len(a.state.Party) {
			return []int{picked}
		}
	case gamepack.CampTargetParty:
		// `0B2Ah..0B91h`：從 `DS:5CF4h` 沿 `+104h` 走，就是隊伍順序。
		table := make([]int, len(a.state.Party))
		for index := range table {
			table[index] = index
		}
		return table
	}
	return nil
}

// campSpellEffect 是戰鬥外的 `08BCh`。handled 為假時這一條不走掛效果那一路，呼叫端照舊。
// 施法者在隊伍的 caster，casterLevel 是 `26F8h` 的等級（物品放的已經換成 6／12）。
func (a *app) campSpellEffect(caster int, option castOption, effect gamepack.CastEffect,
	casterLevel, picked int) (string, bool) {
	if int(option.ID) >= len(a.spellParameters) || caster < 0 || caster >= len(a.state.Party) {
		return "", false
	}
	params := a.spellParameters[option.ID]
	if !campAttachesEffect(effect, params) {
		return "", false
	}
	kind := params.CampTarget()
	table := a.campSpellTable(kind, caster, picked)
	casterName := strings.TrimSpace(a.state.Party[caster].Name)
	took := fmt.Sprintf(a.text(msgCastTookEffect), casterName, option.Label)
	if len(table) == 0 {
		// `0B05h`：沒挑到人，表空，`08E8h` 整段跳過。
		return took, true
	}
	first := &a.state.Party[table[0]]
	// 有前提的那幾支問的是表上第一格（`DS:6B89h`），與 castSpell 前段同一件事。
	if code := effect.BlockedByEffect; code != 0 {
		if index, ok := combatEffects(first.Effects).IndexOf(code); ok {
			first.Effects = storedEffects(combatEffects(first.Effects).RemoveAt(index))
			syncTrainedLibraryCharacter(&a.state, *first)
			return took, true
		}
	}
	if code := effect.RequiresEffect; code != 0 && !combatEffects(first.Effects).Has(code) {
		return took, true
	}
	if effect.MinimumHitPoints > 0 && first.CurrentHP < effect.MinimumHitPoints {
		first.CurrentHP = effect.MinimumHitPoints
	}
	if effect.StrengthValue > 0 || effect.StrengthFromTarget {
		return a.campStrengthSpell(casterName, option, effect, casterLevel, first), true
	}
	code := effect.EffectCode
	if code == 0 {
		code = params.EffectCode()
	}
	if code == 0 || effect.MessageOnly {
		return took, true
	}
	// 模式 0Ah 的祝福與急速（`0F35h`／`2724h`）再從表裡分邊。隊員的 `+10Eh` 都是 0，
	// 戰鬥外 `0F88h` 不問貼身的敵人；急速的額度與「先解緩速」照走。
	if filter, ok := gamepack.SpellSideFilterFor(option.ID); ok {
		list := make([]uint8, 0, len(table))
		for _, index := range table {
			list = append(list, uint8(index))
		}
		kept, err := gamepack.FilterSpellSide(filter, list, 0, casterLevel, gamepack.SpellSideQuery{
			Side: func(uint8) (uint8, bool) { return 0, true },
			CancelEffect: func(index uint8, cancel uint8) bool {
				member := &a.state.Party[index]
				at, found := combatEffects(member.Effects).IndexOf(cancel)
				if !found {
					return false
				}
				member.Effects = storedEffects(combatEffects(member.Effects).RemoveAt(at))
				syncTrainedLibraryCharacter(&a.state, *member)
				return true
			},
		})
		if err != nil {
			return took, true
		}
		table = table[:0]
		for _, index := range kept {
			table = append(table, int(index))
		}
	}
	// 等級覆寫：祈禱把施法者的邊（隊伍是 0）左移進去；友誼術推的是施法前的魅力。
	override := effect.CasterLevelOverride
	if effect.AbilityBonus.Amount > 0 && len(table) > 0 {
		override = raiseMemberAbility(&a.state.Party[table[0]], effect.AbilityBonus)
		syncTrainedLibraryCharacter(&a.state, a.state.Party[table[0]])
	}
	level := uint8(casterLevel)
	if uint8(override) != 0 {
		level = uint8(override)
	}
	affected := 0
	for _, index := range table {
		member := &a.state.Party[index]
		list := combatEffects(member.Effects)
		// `0A35h` 每一格各算一次持續，時點在群組 9 之前。
		duration := gamepack.SpellEffectDuration(option.ID, params, casterLevel, false, a.rollDice)
		if gamepack.SpellEffectImmunity(list, code, 0, casterLevel, a.rollDice) {
			continue
		}
		if duration < 0 {
			duration = 0
		}
		list = gamepack.ApplySpellEffectNode(list,
			gamepack.NewEffectNode(code, uint16(duration), level, effect.EffectParameter != 0))
		member.Effects = storedEffects(list)
		syncTrainedLibraryCharacter(&a.state, *member)
		affected++
	}
	if filter, ok := gamepack.SpellSideFilterFor(option.ID); ok &&
		filter.Routine == gamepack.SpellSideQuota {
		// 急速 `27F5h..2840h`：`08BCh` 之後對表上每一格派發群組 18，這一段沒有戰鬥判斷——
		// 營地施的急速一樣當場老一歲（`0C67h`）。
		for _, index := range table {
			member := &a.state.Party[index]
			list, aged := gamepack.MarkHasteAged(combatEffects(member.Effects))
			member.Effects = storedEffects(list)
			if aged {
				member.Age++
			}
			syncTrainedLibraryCharacter(&a.state, *member)
		}
	}
	if code == gamepack.SpiritualHammerEffectCode && len(table) > 0 {
		// 靈魂鎚 `19D1h`：`08BCh` 之後對表上第一格以模式 0 叫 `17h` 的常式。
		if a.giveSpiritualHammer(&a.state.Party[table[0]]) {
			return fmt.Sprintf(a.text(msgCastGainsItem),
				strings.TrimSpace(a.state.Party[table[0]].Name)), true
		}
	}
	switch {
	case kind == gamepack.CampTargetParty:
		return fmt.Sprintf(a.text(msgCastWholeSide), casterName, option.Label, affected), true
	case affected > 0:
		return fmt.Sprintf(a.text(msgFieldCastDone), casterName, option.Label,
			strings.TrimSpace(a.state.Party[table[0]].Name)), true
	}
	return took, true
}

// campStrengthSpell 是變大術（`128Dh`）、力量術（`1F16h`）在戰鬥外：對表上第一格用
// overlay-24 entry 18 調力量、`07C7h` 算持續、entry 10 掛節點。兩支都沒有 `DS:4954h`
// 的判斷，與 castSpell 力量那一支同形，只是串列是角色身上那一條。
func (a *app) campStrengthSpell(casterName string, option castOption, effect gamepack.CastEffect,
	casterLevel int, subject *poolsave.Character) string {
	current := uint8(subject.Abilities[gamepack.AbilityStrength])
	currentPercentile := uint8(subject.ExceptionalStrength)
	value, percentile := effect.StrengthValue, effect.StrengthPercentile
	if effect.StrengthFromTarget {
		value, percentile = gamepack.StrengthSpellResult(
			memberClassLevels(*subject), current, currentPercentile, a.roller)
	}
	duration := gamepack.SpellEffectDuration(option.ID, a.spellParameters[option.ID],
		casterLevel, false, a.rollDice)
	list, value, percentile, raised := gamepack.ApplyStrengthEffect(
		combatEffects(subject.Effects), effect.EffectCode, duration,
		current, currentPercentile, value, percentile)
	subject.Effects = storedEffects(list)
	if raised {
		subject.Abilities[gamepack.AbilityStrength] = int(value)
		subject.ExceptionalStrength = int(percentile)
	}
	syncTrainedLibraryCharacter(&a.state, *subject)
	if !raised {
		return fmt.Sprintf(a.text(msgCastTookEffect), casterName, option.Label)
	}
	return fmt.Sprintf(a.text(msgCastStronger), strings.TrimSpace(subject.Name), value, percentile)
}
