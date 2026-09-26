package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 只掛效果的那一批法術（spec 098〈只掛效果的那一批〉，issue #89）：祈禱、隱形、閃現、致盲、
// 降咒、護盾、閱讀魔法、防護邪惡……處理常式本身沒有算法，推「法術編號 + 四個覆寫參數」給
// overlay-22 `08BCh`，由那一支對 `20AEh` 收好的表逐格掛參數表 `+0Ah` 的碼。規則在
// internal/gamepack/spell_side_effects.go，這裡只把盤面接上去。

// castEffectOnly 是 `08BCh` 在傷害為 0 時走的那一段（`090Ch..0A62h`）：
//
//	092Fh  表上這一格是 nil → 跳過
//	095Eh  參數表 +8 非 0 → 096Bh 擲豁免（entry 7，類別 +9、修正 0）
//	0997h  參數表 +2 是 FFh → 碰觸：重算、群組 11、entry 6 擲命中；沒中 → 豁免結果當成 1
//	0A13h  參數表 +0Ah 非 0 → 0A35h 持續 = 07C7h(法術)
//	0A5Ah  overlay-24 entry 20(目標, 碼, 持續, 等級, [bp+0Eh], 規則, 豁免結果, 訊息)
//
// 等級是 `[bp-2Ah]`：等級覆寫（`[bp+10h]`）非 0 就用它，否則 26F8h 的施法者等級（`08F2h`）。
// `[bp+0Eh]` 是節點 `+4`（摘掉時要不要叫處理常式）。
func (a *app) castEffectOnly(state *tacticalState, caster spellCasting, option castOption,
	effect gamepack.CastEffect, targets spellTargets, casterLevel int) {
	params := a.spellParameters[option.ID]
	code := effect.EffectCode
	if code == 0 || effect.MessageOnly {
		// 開鎖術（`1A34h`）與縮小術過了關卡之後：參數表 `+0Ah` 為 0，`0A18h` 直接跳過。
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastTookEffect),
			strings.TrimSpace(caster.name), option.Label))
		return
	}
	victims := targets.List
	if !targets.Chosen {
		// 模式 0 的表永遠是施法者自己（`20FDh`）；其餘沒瞄過就沒有表，`08E8h` 整段跳過。
		victims = nil
		if params.TargetPlan().Kind == gamepack.SpellTargetKindSelf {
			victims = []uint8{state.Mover}
		}
	}
	if len(victims) == 0 {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), option.Label))
		return
	}
	override := effect.CasterLevelOverride
	if effect.CasterSideShift != 0 {
		side, _ := state.sideOf(state.Mover)
		override += int(side) << effect.CasterSideShift
	}
	if effect.AbilityBonus.Amount > 0 {
		// 友誼術 `13CEh..1404h`：記下表上第一格（`DS:6B89h`）的魅力、加 Roll(2, 4)、夾在 19h，
		// 再把**原本的**魅力推在等級覆寫那一格——節點 `+3` 存它，收尾時還原（`05E0h`）。
		override = a.raiseAbilityForSpell(state, victims[0], effect.AbilityBonus)
	}
	level := uint8(casterLevel)
	if uint8(override) != 0 {
		level = uint8(override)
	}
	rule := params.SaveRule()
	affected := 0
	for _, index := range victims {
		if int(index) >= len(state.Roster) || state.Roster[index].FootprintClass == 0 {
			continue
		}
		saved := false
		if rule != gamepack.SaveRuleNone {
			saved = a.savedAgainstCategory(state, index, params.SaveCategory(), 0)
		}
		if params.RequiresAttackRoll() && !a.touchSpellHits(state, index) {
			// `09D7h` `C6 46 0C 00`／`09DBh` `C6 46 D0 01`：沒碰到就當成豁免成功。
			saved = true
		}
		duration := gamepack.SpellEffectDuration(option.ID, params, casterLevel, true, a.rollDice)
		// entry 20：先問群組 9（`166Fh`），再看「豁免成功而且規則是 1」（`1689h..1693h`）。
		if a.unaffectedBySpellEffect(state, index, code, casterLevel) {
			continue
		}
		if saved && rule == gamepack.SaveRuleNegates {
			a.tacticalStatus(state, state.say(msgCastUnaffected, index))
			continue
		}
		state.attachSpellEffect(int(index), code, duration, level, effect.EffectParameter != 0)
		affected++
	}
	a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastWholeSide),
		strings.TrimSpace(caster.name), option.Label, affected))
	if code == gamepack.SpiritualHammerEffectCode {
		// 靈魂鎚 `19D1h`：`08BCh` 之後以模式 0 對表上第一格叫 `17h` 的常式（spiritual_hammer.go）。
		a.grantSpiritualHammer(state, victims[0])
	}
}

// attachSpellEffect 是 entry 20 的 `16B8h..1716h`，節點 `+3` 原樣存等級覆寫推進來的整個
// byte（祈禱的邊、友誼的魅力、鏡影的影像數、`0FFh` 的解不掉），不像 applySpellEffect 那樣
// 夾在低四位。
func (state *tacticalState) attachSpellEffect(index int, code uint8, duration int, level uint8,
	teardown bool) {
	if index < 0 || index >= len(state.Effects) {
		return
	}
	if duration < 0 {
		duration = 0
	}
	state.Effects[index] = gamepack.ApplySpellEffectNode(state.Effects[index],
		gamepack.NewEffectNode(code, uint16(duration), level, teardown))
}

// raiseAbilityForSpell 把能力值加上去、夾在上限，回傳加之前的值。怪物的記錄 remake 沒有
// 能力值那幾格，回 0（`08F2h` 會退回施法者等級）。
func (a *app) raiseAbilityForSpell(state *tacticalState, cell uint8, bonus gamepack.AbilityBonus) int {
	subject := a.partyMemberAt(state, cell)
	if subject == nil {
		return 0
	}
	before := raiseMemberAbility(subject, bonus)
	syncTrainedLibraryCharacter(&a.state, *subject)
	return before
}

// raiseMemberAbility 是 `13CEh..1404h` 落在隊員記錄上：加、夾在上限，回傳加之前的值。
// 營地施的友誼術走同一支（field_cast_effects.go）。
func raiseMemberAbility(subject *poolsave.Character, bonus gamepack.AbilityBonus) int {
	if bonus.Ability < 0 || bonus.Ability >= len(subject.Abilities) {
		return 0
	}
	before := subject.Abilities[bonus.Ability]
	value := before + bonus.Amount
	if value > bonus.Cap {
		value = bonus.Cap
	}
	subject.Abilities[bonus.Ability] = value
	return before
}

// partyMemberAt 是站在那一格的隊員；不是隊員回 nil。
func (a *app) partyMemberAt(state *tacticalState, cell uint8) *poolsave.Character {
	if int(cell) >= len(state.PartySlot) {
		return nil
	}
	slot := state.PartySlot[cell]
	if slot < 0 || slot >= len(a.state.Party) {
		return nil
	}
	return &a.state.Party[slot]
}

// touchSpellHits 是 `08BCh` 的 `0997h..09D5h`：參數表 `+2` 是 FFh 的法術要先碰得到。
//
//	099Eh  010Ah:0043h(目標)       ; 重算戰鬥數值
//	09B2h  0100h:002Fh(0Bh, 目標)  ; 群組 11
//	09CEh  0100h:003Eh(5CF0h, 目標, 目標 +111h)  ; entry 6，與近戰同一支
//
// entry 6 就是近戰那一支命中擲骰，所以群組 10／16 與隱形出手就現形都照走。
func (a *app) touchSpellHits(state *tacticalState, target uint8) bool {
	mover := state.Mover
	if int(mover) >= len(state.THAC0) || int(target) >= len(state.ArmorClass) {
		return false
	}
	roll := uint8(a.rollDice(1, 20))
	modifier, missed := state.hitRollAfterEffects(mover, target, roll)
	hit, err := combat.ResolveHit(roll, state.THAC0[mover], state.hitCheckArmourClass(target), modifier)
	return err == nil && hit && !missed
}

// hitCheckArmourClass 是出手之前的群組 11：從重算過的 AC 內部值起算（gamepack.HitCheckArmourClass）。
func (state *tacticalState) hitCheckArmourClass(target uint8) int {
	armourClass := state.ArmorClass[target]
	if int(target) < len(state.Effects) {
		armourClass = gamepack.HitCheckArmourClass(state.Effects[target], armourClass)
	}
	return armourClass
}
