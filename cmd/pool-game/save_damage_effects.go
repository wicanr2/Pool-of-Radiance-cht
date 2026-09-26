package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 群組 12（豁免）、6（法術傷害）、4／5（近戰傷害）在戰場上的接點（spec 112〈群組 12／6／4／5〉，issue #96／#99）。
// 規則在 internal/gamepack/save_damage_effects.go，這裡只把盤面接上去。

const (
	// msgCastLostImage 是 overlay-12 entry 27 的 `09BFh`："lost an image"。
	msgCastLostImage messageID = iota + 2980
)

func init() {
	for id, key := range map[messageID]string{
		msgCastLostImage: "ui.castLostImage",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// protectionAreaRadius 是 `014Dh` 的 `01EEh..01FBh`：`2Dh`／`2Eh` 的半徑 1（`31h` 是 6）。
const protectionAreaRadius = 1

// unknownAlignment 是 remake 沒有那一格記錄 `+0A0h` 時的值。
const unknownAlignment = 0xff

// 285-byte 記錄裡群組 12 讀的兩格：`24BCh` `26 8A 45 14`（體質）、`037Eh` `26 80 BD A0 00`（陣營）。
const (
	recordConstitutionOffset = 0x14
	recordAlignmentOffset    = 0xa0
)

// spellDamageContext 是施法那一趟的三個全域：`DS:6779h`（正在處理的法術）、`DS:6777h`（傷害
// 種類）與 `DS:677Eh`（這一份表是以一點收的）。原版在 `08BCh` 結尾 `0A6Ah` 把 `6777h` 寫 0、
// 派發之後 `0EA7h`／`0EACh` 把 `6779h`／`677Eh` 清 0——castSpell 收尾時整份歸零。
type spellDamageContext struct {
	Spell uint8
	Flags uint8
	Area  bool
}

// rememberSaveRecord 記下那一格記錄 `+14h`（體質）與 `+0A0h`（陣營），群組 12 的
// `5Ah`／`61h` 與 `08h 09h 2Dh 2Eh` 讀它們。與 RecordNames 同一個作法：用到才長。
func (state *tacticalState) rememberSaveRecord(index int, constitution, alignment uint8) {
	if index < 0 || index >= len(state.Roster) {
		return
	}
	if len(state.Constitution) < len(state.Roster) {
		values := make([]uint8, len(state.Roster))
		copy(values, state.Constitution)
		state.Constitution = values
	}
	if len(state.Alignment) < len(state.Roster) {
		values := make([]uint8, len(state.Roster))
		for at := range values {
			values[at] = unknownAlignment
		}
		copy(values, state.Alignment)
		state.Alignment = values
	}
	state.Constitution[index], state.Alignment[index] = constitution, alignment
}

// rememberPartySaveRecord 是隊員那一份：體質取能力值、陣營取建角的編號
// （creation.Alignments 的順序就是 `+0A0h` 的值）。
func (state *tacticalState) rememberPartySaveRecord(index int, member poolsave.Character) {
	alignment := uint8(unknownAlignment)
	for value, choice := range creation.Alignments {
		if choice.ID == member.AlignmentID {
			alignment = uint8(value)
		}
	}
	constitution := member.Abilities[gamepack.AbilityConstitution]
	if constitution < 0 || constitution > 0xff {
		constitution = 0
	}
	state.rememberSaveRecord(index, uint8(constitution), alignment)
}

// saveRecordOf 回傳那一格的體質（0 代表不知道）與陣營。
func (state *tacticalState) saveRecordOf(index uint8) (constitution, alignment uint8, known bool) {
	if int(index) < len(state.Constitution) {
		constitution = state.Constitution[index]
	}
	alignment = unknownAlignment
	if int(index) < len(state.Alignment) {
		alignment = state.Alignment[index]
	}
	return constitution, alignment, alignment != unknownAlignment
}

// saveRollAfterEffects 是 entry 7 的 `0DB2h`：豁免骰算好之後派發擲豁免那一個的群組 12。
// 回傳的是 byte：`0DC8h` 拿它與目標值做無號比較。
func (state *tacticalState) saveRollAfterEffects(target uint8, category gamepack.SaveCategory,
	value int) uint8 {
	if int(target) >= len(state.Effects) {
		return uint8(value)
	}
	side, _ := state.sideOf(target)
	constitution, _, _ := state.saveRecordOf(target)
	_, actorAlignment, known := state.saveRecordOf(state.Mover)
	return gamepack.SaveRollEffects{
		Effects:             state.Effects[target],
		Category:            uint8(category),
		Side:                side,
		Constitution:        constitution,
		ActorAlignment:      actorAlignment,
		ActorAlignmentKnown: known,
		DamageFlags:         state.SpellDamage.Flags,
		AreaNode: func(code uint8) (gamepack.EffectNode, bool) {
			radius := protectionAreaRadius
			if code == gamepack.PrayerAreaEffectCode {
				radius = prayerAreaRadius
			}
			return state.areaEffectNode(target, code, radius)
		},
	}.Apply(uint8(value))
}

// spellDamageAfterEffects 是 entry 19 的 `1344h..1351h`：傷害進 `DS:6776h` 之後派發受傷那一個
// 的群組 6。鏡影吃掉這一下時照原版印一句（`0A0Fh`）。
func (a *app) spellDamageAfterEffects(state *tacticalState, target, spell uint8, damage int) int {
	if int(target) >= len(state.Effects) {
		return damage
	}
	outcome := gamepack.SpellDamageEffects{
		Effects:     state.Effects[target],
		Spell:       spell,
		DamageFlags: state.SpellDamage.Flags,
		Area:        state.SpellDamage.Area,
		Roll:        a.rollDice,
	}.Apply(damage)
	state.Effects[target] = outcome.Effects
	if outcome.LostImage {
		a.tacticalStatus(state, state.say(msgCastLostImage, target))
	}
	return outcome.Damage
}

// meleeDamageAfterEffects 是 overlay-13 `0215h..022Ch`：近戰傷害骰算完之後，攻擊者的群組 4、
// 目標的群組 5。群組 5 的 `1Ch` 在這裡一定擲一次骰，但 `DS:6779h` 是 0，擋不下來。
func (a *app) meleeDamageAfterEffects(state *tacticalState, attacker, target uint8, damage int) int {
	if int(attacker) < len(state.Effects) {
		damage = gamepack.MeleeDamageAfterAttackerEffects(state.Effects[attacker], damage)
	}
	if int(target) < len(state.Effects) {
		state.Effects[target], _ = gamepack.MirrorImageAbsorbs(state.Effects[target], 0, false, a.rollDice)
	}
	return damage
}
