package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 效果掛上、解除、老化時逐人印的那一行（issue #87，spec 163）。
//
// 原版印這幾句只走三條路，全部在 overlay-24／overlay-12 裡、以 overlay-25 的 entry 20／26 印：
//
//	overlay-24 entry 20（`1656h`）`171Fh`：訊息非空 → entry 26(目標, 1, 訊息)、entry 21 清右欄。
//	    訊息是處理常式推給 `08BCh` 的那一段 Pascal 字串（祝福 "is Blessed"、急速 "is Hasted"……）。
//	overlay-24 entry 15（`107Bh`）`10A8h..10B8h`：身上有那個碼 → entry 20(目標, "is Cured", 0Ah, 1)。
//	overlay-12 entry 36（`0C67h`，群組 18 的 `27h`）`0C98h..0CA8h`：第一次 → entry 20(目標, "ages", 0Ah, 1)。
//
// entry 26 旗標 1 在戰鬥中是 entry 20(…, 0Ah, 0) 加閃光動畫（turnedNotice 同一支），戰鬥外
// （`21C1h`）是 entry 20(…, 0Ah, 1)：下方訊息框印名字與那一句、停一拍。

// 這一段訊息另開 `iota + 5600`，在 init 登記進 messageKeys，重號直接 panic。
const (
	// msgNoticeCured 是 overlay-24 `1072h` 的 "is Cured"（entry 15）。
	msgNoticeCured messageID = iota + 5600
	// msgNoticeAges 是 overlay-12 `0C62h` 的 "ages"（`27h` 的處理常式）。
	msgNoticeAges
	// 以下是處理常式自己的字串（overlay-22，位址是長度 byte）：解盲 `21E0h`、除咒 `24E6h`／`24F3h`、
	// 編號 58 `2DF8h`、62 `2F7Bh`（兩段字面相同）。都經 entry 26 印。
	msgNoticeCanSee
	msgNoticeUncursed
	msgNoticeItemUncursed
	msgNoticeHealed
	// 以下是推給 `08BCh`、由 entry 20 `171Fh` 印的那一批（effectAttachMessages）。
	msgNoticeBlessed
	msgNoticeCursed
	msgNoticeAffected
	msgNoticeProtected
	msgNoticeColdResistant
	msgNoticeCharmed
	msgNoticeFriendly
	msgNoticeShielded
	msgNoticeAsleep
	msgNoticeFireResistant
	msgNoticeSilenced
	msgNoticeInvisible
	msgNoticeDuplicated
	msgNoticeWeakened
	msgNoticeBlind
	msgNoticeDiseased
	msgNoticePraying
	msgNoticeBestowCursed
	msgNoticeBlinking
	msgNoticeHasted
	msgNoticeSlowed
	msgNoticeSpeedy
	msgNoticeParalyzed
	msgNoticeReading
)

func init() {
	for id, key := range map[messageID]string{
		msgNoticeCured:         "ui.noticeCured",
		msgNoticeAges:          "ui.noticeAges",
		msgNoticeCanSee:        "ui.noticeCanSee",
		msgNoticeUncursed:      "ui.noticeUncursed",
		msgNoticeItemUncursed:  "ui.noticeItemUncursed",
		msgNoticeHealed:        "ui.noticeHealed",
		msgNoticeBlessed:       "ui.noticeBlessed",
		msgNoticeCursed:        "ui.noticeCursed",
		msgNoticeAffected:      "ui.noticeAffected",
		msgNoticeProtected:     "ui.noticeProtected",
		msgNoticeColdResistant: "ui.noticeColdResistant",
		msgNoticeCharmed:       "ui.noticeCharmed",
		msgNoticeFriendly:      "ui.noticeFriendly",
		msgNoticeShielded:      "ui.noticeShielded",
		msgNoticeAsleep:        "ui.noticeAsleep",
		msgNoticeFireResistant: "ui.noticeFireResistant",
		msgNoticeSilenced:      "ui.noticeSilenced",
		msgNoticeInvisible:     "ui.noticeInvisible",
		msgNoticeDuplicated:    "ui.noticeDuplicated",
		msgNoticeWeakened:      "ui.noticeWeakened",
		msgNoticeBlind:         "ui.noticeBlind",
		msgNoticeDiseased:      "ui.noticeDiseased",
		msgNoticePraying:       "ui.noticePraying",
		msgNoticeBestowCursed:  "ui.noticeBestowCursed",
		msgNoticeBlinking:      "ui.noticeBlinking",
		msgNoticeHasted:        "ui.noticeHasted",
		msgNoticeSlowed:        "ui.noticeSlowed",
		msgNoticeSpeedy:        "ui.noticeSpeedy",
		msgNoticeParalyzed:     "ui.noticeParalyzed",
		msgNoticeReading:       "ui.noticeReading",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// effectAttachMessages 是每個編號的處理常式推給 `08BCh`（或模式 0Ah 的 `0F35h`／`2724h`，
// 它們再推給 `08BCh`）的訊息。英文字串與 `gamepack.ReadDOSSpellDispatchTable` 讀出來的
// `Message` 逐格相同（TestEffectAttachMessagesMatchTheDispatchTable）。
//
// 只收推給 `08BCh` 的那幾支：定身術 `1650h`、變大術、力量術、恢復術、臭雲術、死靈術、
// 解盲、解除魔法、除咒、58／59／62 的字串走別的印法，不在這張表。開鎖術（31）推了
// "Knock-Knock"，但參數表 `+0Ah` 是 0，`0A13h` 不叫 entry 20，所以永遠印不出來。
var effectAttachMessages = map[uint8]messageID{
	1: msgNoticeBlessed, 2: msgNoticeCursed,
	5: msgNoticeAffected, 11: msgNoticeAffected, 18: msgNoticeAffected, 22: msgNoticeAffected,
	26: msgNoticeAffected, 29: msgNoticeAffected,
	6: msgNoticeProtected, 7: msgNoticeProtected, 16: msgNoticeProtected, 17: msgNoticeProtected,
	52: msgNoticeProtected, 53: msgNoticeProtected, 54: msgNoticeProtected,
	8: msgNoticeColdResistant, 10: msgNoticeCharmed, 27: msgNoticeCharmed, 14: msgNoticeFriendly,
	19: msgNoticeShielded, 21: msgNoticeAsleep, 24: msgNoticeFireResistant, 25: msgNoticeSilenced,
	30: msgNoticeInvisible, 50: msgNoticeInvisible, 63: msgNoticeInvisible,
	32: msgNoticeDuplicated, 33: msgNoticeWeakened, 38: msgNoticeBlind, 40: msgNoticeDiseased,
	42: msgNoticePraying, 44: msgNoticeBestowCursed, 45: msgNoticeBlinking,
	48: msgNoticeHasted, 55: msgNoticeSlowed, 57: msgNoticeSpeedy, 61: msgNoticeParalyzed,
	67: msgNoticeReading,
}

// entry15Probes 是處理常式依序拿去問 overlay-24 entry 15 的碼（身上有就印 "is Cured"、摘掉）：
// 解盲 `21F6h`、解病鏈 `225Bh`（22h、2Bh、32h）、除咒 `2516h`、編號 58 `2E08h`（37h，
// 沒有才走解病鏈）。
func entry15Probes(spell uint8) []uint8 {
	switch spell {
	case gamepack.SpellIDCureBlindness:
		return []uint8{gamepack.BlindnessEffectCode}
	case gamepack.SpellIDCureDisease:
		return cureDiseaseProbes()
	case gamepack.SpellIDRemoveCurse:
		return []uint8{removeCurseEffectCode}
	}
	return nil
}

// removeCurseEffectCode 是除咒 `2516h` 問的 24h（降咒術掛的碼）。
const removeCurseEffectCode uint8 = 0x24

// cureDiseaseProbes 是解病鏈 `225Bh` 問 entry 15 的三個碼（`2272h`、`228Ah`、`22CEh`）。
func cureDiseaseProbes() []uint8 {
	return []uint8{gamepack.DiseaseEffectCode, gamepack.DiseaseWeakeningEffectCode, gamepack.NoHealingEffectCode}
}

// curedCount 是 probes 裡身上有幾個（每一個 entry 15 各印一次 "is Cured"）。
func curedCount(list gamepack.EffectList, probes []uint8) int {
	count := 0
	for _, code := range probes {
		if list.Has(code) {
			count++
		}
	}
	return count
}

// ---- 戰鬥中 ----

// curedNotice 是 entry 15 的 `10A8h..10B8h`：entry 20(目標, "is Cured", 0Ah, 1)。
func (a *app) curedNotice(state *tacticalState, index uint8) {
	a.panelNotice(state, index, state.say(msgNoticeCured), noticeRowPanel, true)
}

// agesNotice 是 `0C98h..0CA8h`：entry 20(目標, "ages", 0Ah, 1)。
func (a *app) agesNotice(state *tacticalState, index uint8) {
	a.panelNotice(state, index, state.say(msgNoticeAges), noticeRowPanel, true)
}

// attachNotice 是 entry 20 `171Fh`：掛上之後，處理常式推的訊息非空就經 entry 26 旗標 1 印
// （戰鬥中名字、那一句與閃光動畫）。
func (a *app) attachNotice(state *tacticalState, index, spell uint8) {
	if id, ok := effectAttachMessages[spell]; ok {
		a.turnedNotice(state, index, state.say(id))
	}
}

// sparkleNotice 是處理常式自己經 entry 26 旗標 1 印的那一句（"can see"、"is un-cursed"……）。
func (a *app) sparkleNotice(state *tacticalState, index uint8, id messageID) {
	a.turnedNotice(state, index, state.say(id))
}

// ---- 戰鬥外 ----

// fieldNotice 是戰鬥外 entry 20 在下方訊息框印的那一行：名字、那一句。收進 a.fieldNotices，
// 呼叫端施完之後交給 showEffectBeats 一拍一拍放。
func (a *app) fieldNotice(name string, id messageID) {
	text := a.text(id)
	line := strings.TrimSpace(name) + " " + text
	if strings.HasPrefix(text, "'") {
		// "'s item is un-cursed" 原版印在名字的下一列；排成一行時不要在撇號前面空一格。
		line = strings.TrimSpace(name) + text
	}
	a.fieldNotices = append(a.fieldNotices, line)
}

// takeFieldNotices 取走這一次施法收好的那幾行。
func (a *app) takeFieldNotices() []string {
	lines := a.fieldNotices
	a.fieldNotices = nil
	return lines
}

// fieldCureNotices 是戰鬥外解盲、解病、除咒、編號 58／62 那幾支的逐人訊息，before 是施法前
// 目標身上的串列、applied 是 applyFieldEffect 的結果。規則仍由 applyFieldEffect 做，這裡只
// 依施法前後的差照原版的先後排訊息。
func (a *app) fieldCureNotices(spell uint8, name string, before gamepack.EffectList, applied bool) {
	switch spell {
	case gamepack.SpellIDGreaterHeal:
		// `2E08h`：中毒就只印 "is Cured"；否則解病鏈各印一次，解到就結束；都沒有才治療、印 "is Healed"。
		if before.Has(gamepack.PoisonEffectCode) {
			a.fieldNotice(name, msgNoticeCured)
			return
		}
		if cured := curedCount(before, cureDiseaseProbes()); cured > 0 {
			for ; cured > 0; cured-- {
				a.fieldNotice(name, msgNoticeCured)
			}
			return
		}
		if applied {
			a.fieldNotice(name, msgNoticeHealed)
		}
		return
	case gamepack.SpellIDLesserHeal:
		if applied {
			a.fieldNotice(name, msgNoticeHealed)
		}
		return
	}
	cured := curedCount(before, entry15Probes(spell))
	for count := cured; count > 0; count-- {
		a.fieldNotice(name, msgNoticeCured)
	}
	switch spell {
	case gamepack.SpellIDCureBlindness:
		if cured > 0 {
			a.fieldNotice(name, msgNoticeCanSee)
		}
	case gamepack.SpellIDRemoveCurse:
		if cured > 0 {
			a.fieldNotice(name, msgNoticeUncursed)
		} else if applied {
			a.fieldNotice(name, msgNoticeItemUncursed)
		}
	}
}

// effectBeatQueue 是戰鬥外那幾行的停拍：每一行停一拍（`21C1h` 的 entry 20(…, 1)），停完才
// 顯示施法那一步原本的結果。停拍中不讀鍵（原版的 Delay 本身就不讀鍵）。
type effectBeatQueue struct {
	lines  []string
	ticks  int
	final  string
	target *string
}

// showEffectBeats 把 lines 一行一行放進 target（探索施法的訊息列、物品頁的訊息列），每一行
// 停一拍，最後放 final。速度 0 時一拍是 0，原版也是一閃而過，所以直接放 final。
func (a *app) showEffectBeats(target *string, lines []string, final string) {
	ticks := a.speedDelayTicks()
	if target == nil {
		return
	}
	if ticks == 0 || len(lines) == 0 {
		*target = final
		return
	}
	*target = lines[0]
	a.effectBeats = effectBeatQueue{lines: lines[1:], ticks: ticks, final: final, target: target}
}

// effectBeatInput 在 Update 前段：還有停拍中的那一行就這一影格什麼都不做。
func (a *app) effectBeatInput() bool {
	queue := &a.effectBeats
	if queue.target == nil {
		return false
	}
	if queue.ticks > 1 {
		queue.ticks--
		return true
	}
	if len(queue.lines) > 0 {
		*queue.target = queue.lines[0]
		queue.lines = queue.lines[1:]
		queue.ticks = a.speedDelayTicks()
		return true
	}
	*queue.target = queue.final
	*queue = effectBeatQueue{}
	return true
}

// removalNotices 是解盲、解病、除咒在戰鬥中：身上有幾個就印幾次 "is Cured"（entry 15），
// 解盲接著 entry 26 印 "can see"（`2202h..221Ch`）、除咒接著 "is un-cursed"（`2522h..253Ch`）。
func (a *app) removalNotices(state *tacticalState, spell, cell uint8, cured int) {
	for count := cured; count > 0; count-- {
		a.curedNotice(state, cell)
	}
	if cured == 0 {
		return
	}
	switch spell {
	case gamepack.SpellIDCureBlindness:
		a.sparkleNotice(state, cell, msgNoticeCanSee)
	case gamepack.SpellIDRemoveCurse:
		a.sparkleNotice(state, cell, msgNoticeUncursed)
	}
}
