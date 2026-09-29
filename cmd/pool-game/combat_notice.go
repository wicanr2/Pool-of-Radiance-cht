package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// 戰鬥中 AI 的那幾句訊息怎麼印、停多久（issue #104，spec 098〈AI 戰鬥訊息的字串與印法〉）。
//
// 原版有兩支印訊息的常式，都在 overlay-25：
//
//	entry 20（`1738h`）(記錄, 字串, 列, 停)：清右欄 17h..26h × 列..15h，
//	    `1865h` 在欄 17h、那一列印記錄 `+0` 的名字，下一列起以色 0Ah 折行印字串；
//	    「停」非 0 就經 overlay-37 entry 13 等一拍再清掉右欄。
//	entry 19（`16DFh`）(字串)：清第 24 列、欄 0 以色 0Ah 印字串、等一拍、再清。
//
// 「等一拍」是 `Delay(遊戲速度 × 225 ms)`，remake 以 `speedDelayTicks` 換成影格：
// 停拍期間 tacticalInput 不做別的事（原版的 Delay 本身就不讀鍵）。

// 這一段訊息另開 `iota + 3104`，在 init 登記進 messageKeys，重號直接 panic。
const (
	msgFoeSpellLine messageID = iota + 3104
	msgFoeItemLine
)

// 攻擊那一則（overlay-13 entry 4 `02FEh`，#110）另開 `iota + 4200`。
const (
	// msgAttackAttacks 是 `0252h` 的 "Attacks"（模式 2 "-Backstabs-" `0237h`、3 "slays helpless" `0243h`
	// remake 沒有背刺與斬殺無助者，只用這一句）。
	msgAttackAttacks messageID = iota + 4200
	// msgAttackHitPoint／msgAttackHitPoints 是 "Hitting for " 串接傷害、" point "／" points "
	// 再串 "of damage"（`027Dh`、`028Ah`、`0292h`、`029Bh`）。
	msgAttackHitPoint
	msgAttackHitPoints
	// msgAttackMisses 是 `02A5h` 的 "and Misses"。
	msgAttackMisses
)

func init() {
	for id, key := range map[messageID]string{
		msgFoeSpellLine:    "ui.foeSpellLine",
		msgFoeItemLine:     "ui.foeItemLine",
		msgAttackAttacks:   "ui.attackAttacks",
		msgAttackHitPoint:  "ui.attackHitPoint",
		msgAttackHitPoints: "ui.attackHitPoints",
		msgAttackMisses:    "ui.attackMisses",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

const (
	// noticeRowPanel 是 entry 20 大多數呼叫端傳的列（0Ah）；"lost a spell" 傳 0Ch
	// （overlay-13 `0523h`、overlay-24 `1538h`），名字那一行因此往下兩列。
	noticeRowPanel = 0x0A
	noticeRowLost  = 0x0C
	// noticeRowSpell 是 `32D3h` 印 "Spell:" 的那一列（17h）。
	noticeRowSpell = 0x17
	// noticeRowLast 是 entry 20 清的右欄最下一列（15h）。
	noticeRowLast = 0x15
	// noticeRowTarget 是攻擊那一則 entry 22（`1865h`）印目標名字的列（0Ch，overlay-13
	// `036Bh..037Bh`）；傷害那一句從下一列起折行（`0367h` 的 0Ch 加一，`04BCh..04DAh`）。
	noticeRowTarget = 0x0C
)

// `1865h` 印名字的三種顏色（`186Eh..188Dh`）：記錄 `+10Dh` 為 0（已經離場）0Ch，
// `+10Eh` 是 1（敵方）0Eh，其餘 0Bh。字串本身一律是 0Ah（`1790h`、overlay-37 entry 3
// 的呼叫端）。都是 EGA 色號，畫的時候查目前主題的色盤。
const (
	noticeInkParty = 0x0B
	noticeInkOut   = 0x0C
	noticeInkFoe   = 0x0E
	noticeInkText  = 0x0A
)

// combatNotice 是一次 entry 20 或 entry 19 的輸出，停拍倒數完才換下一則。
type combatNotice struct {
	// Name 與 Text 畫在右欄：Name 在 Row 那一列，Text 從下一列起折行。
	Name string
	Text string
	Row  int
	// NameInk 是名字的 EGA 色號（noticeInk*），排進佇列那一刻依記錄算好。
	NameInk uint8
	// Target 與 Detail 只有攻擊那一則用：Target 是 entry 22 印在第 0Ch 列的目標名字，
	// Detail 是傷害那一句，從第 0Dh 列起折行。
	Target    string
	TargetInk uint8
	Detail    string
	// Bottom 是第 23 列（17h）欄 0 的那一行（`32D3h` 的 "Spell:" 加法名）。
	Bottom string
	// Footer 是 entry 19 印在第 24 列欄 0 的固定字串，停拍時取代指令列。
	Footer string
	// Ticks 是還要停幾個影格。0 代表這一則不停拍。
	Ticks int
	// Anim 是這一則停拍期間盤面上播的動畫（彈道、閃光、倒下；combat_animation.go，spec 166）。
	Anim *combatAnimation
	// Kept 是 entry 20 的「停」為 0：印完不清右欄，接著播的動畫期間字還在（queueAnimation）。
	Kept bool
}

// combatantName 是記錄 `+0` 的名字，也就是 `1865h` 印在右欄的那一行：隊員讀角色，
// 怪物讀開打時記下的記錄名（依語言換成譯名，與資訊欄第一行同一套）。
func (a *app) combatantName(state *tacticalState, index uint8) string {
	position := int(index)
	if position < len(state.PartySlot) {
		if slot := state.PartySlot[position]; slot >= 0 && slot < len(a.state.Party) {
			return strings.TrimSpace(a.state.Party[slot].Name)
		}
	}
	name := ""
	if position < len(state.RecordNames) {
		name = state.RecordNames[position]
	}
	if strings.TrimSpace(name) == "" {
		name = state.Casting.Names[position]
	}
	if strings.TrimSpace(name) == "" {
		return a.text(msgCombatFoe)
	}
	return a.monsterText.Translate(name)
}

// nameInk 是 `1865h` 替這一格挑的顏色：離場（體型 0，原版 `+10Dh` 為 0）、敵方
// （`+10Eh` 是 1）、其餘。
func (state *tacticalState) nameInk(index uint8) uint8 {
	if int(index) >= len(state.Roster) || state.Roster[index].FootprintClass == 0 {
		return noticeInkOut
	}
	if side, ok := state.sideOf(index); ok && side == 1 {
		return noticeInkFoe
	}
	return noticeInkParty
}

// panelNotice 是 overlay-25 entry 20：名字加一句，beat 為真時停一拍。回傳記錄用的
// 一行（名字、空白、那一句）。
func (a *app) panelNotice(state *tacticalState, index uint8, text string, row int, beat bool) string {
	name := a.combatantName(state, index)
	notice := combatNotice{Name: name, Text: text, Row: row, NameInk: state.nameInk(index), Kept: !beat}
	if beat {
		notice.Ticks = a.speedDelayTicks()
	}
	state.Notices = append(state.Notices, notice)
	return name + " " + text
}

// footerNotice 是 overlay-25 entry 19：第 24 列印一句固定字串、停一拍。
func (a *app) footerNotice(state *tacticalState, text string) string {
	state.Notices = append(state.Notices, combatNotice{Footer: text, Ticks: a.speedDelayTicks()})
	return text
}

// attackNotice 是 overlay-13 entry 4（`02FEh`）的畫面：entry 20(攻擊者, "Attacks", 0Ah, 0)、
// entry 22 在第 0Ch 列印目標名字、下一列起折行印 "Hitting for N points of damage" 或
// "and Misses"，接著 `0509h`／`054Dh`／`0554h` 三條路都經 overlay-37 entry 13 等一拍。
// 攻擊包裝（`1678h..176Ah`）每一下命中呼叫一次，一下都沒中才以 hit = 0 呼叫一次。
//
// 目標的顏色取呼叫當下：原版印目標名字（`037Bh`）在扣生命值（`048Dh`）之前。
func (a *app) attackNotice(state *tacticalState, attacker, target uint8, damage int, hit bool) {
	detail := state.say(msgAttackMisses)
	if hit {
		detail = state.say(msgAttackHitPoints, damage)
		if damage == 1 {
			detail = state.say(msgAttackHitPoint, damage)
		}
	}
	state.Notices = append(state.Notices, combatNotice{
		Name: a.combatantName(state, attacker), NameInk: state.nameInk(attacker),
		Text: state.say(msgAttackAttacks), Row: noticeRowPanel,
		Target: a.combatantName(state, target), TargetInk: state.nameInk(target),
		Detail: detail, Ticks: a.speedDelayTicks(),
	})
}

// castNotice 是 overlay-22 entry 5 在挑目標之前（`0D23h`，旗標 `[bp+0Ah]` 非 0）呼叫的
// `32D3h`：entry 20(記錄, "Casts a Spell", 0Ah, 1)，接著在第 23 列印 "Spell:" 加法名。
//
// 原版先停拍、清右欄，再印第 23 列，那一列留到挑目標與效果動畫之後；remake 的挑目標與
// 效果在同一個影格算完、沒有動畫層，所以兩段放在同一拍一起顯示。
func (a *app) castNotice(state *tacticalState, mover, spell uint8) string {
	line := a.panelNotice(state, mover, state.say(msgFoeCasts), noticeRowPanel, true)
	bottom := state.say(msgFoeSpellLine, a.noticeSpellName(spell))
	state.Notices[len(state.Notices)-1].Bottom = bottom
	return line + " " + bottom
}

// itemUseLine 是 overlay-19 entry 8 的 `1B2Dh..1BB8h`：entry 20(記錄, "uses an item", 0Ah, 0)，
// 戰鬥中接著在第 23 列欄 0 印 "Item:"（`1B56h..1B6Eh`）、overlay-25 entry 1 從欄 5 印物品名
// （`1B8Dh`；entry 1 本身沒有 Delay），然後 `1BB3h` 等一拍、`1BB8h` entry 21 清右欄。
// 所以停拍在物品名印完之後，名字、那一句與 "Item:" 那一行同一拍顯示。
// `DS:6CB3h` 為 0（卷軸）整段跳過（`1B19h`），呼叫端不叫這一支。
func (a *app) itemUseLine(state *tacticalState, mover uint8, item string) string {
	line := a.panelNotice(state, mover, state.say(msgFoeUsesItem), noticeRowPanel, true)
	bottom := state.say(msgFoeItemLine, item)
	state.Notices[len(state.Notices)-1].Bottom = bottom
	return line + " " + bottom
}

// turnedSparkleMilliseconds 是 overlay-25 entry 26 動畫每一格的 `0512h:029Eh`（Delay，毫秒）
// 參數 46h（`2173h`）。
const turnedSparkleMilliseconds = 0x46

// turnedNotice 是 overlay-25 entry 26（`2041h`）旗標 1 的那一路（"is turned"，overlay-13
// `129Dh..12A7h`）：entry 20(記錄, 字串, 0Ah, **0**) 之後播閃光動畫，(遊戲速度 + 1) 輪、
// 每輪四格、每格 Delay(70 ms)（`2130h..21ADh`）；遊戲速度是 0 才另外等一拍（`21B4h`，
// 那時一拍是 0）。閃光畫的是槽 16h（COMSPR 9）的四格（combat_animation.go，spec 166）。
func (a *app) turnedNotice(state *tacticalState, index uint8, text string) string {
	line := a.panelNotice(state, index, text, noticeRowPanel, false)
	notice := &state.Notices[len(state.Notices)-1]
	notice.Ticks = (int(a.gameSpeed) + 1) * 4 * turnedSparkleMilliseconds * 60 / 1000
	notice.Anim = sparkleAnimation(state, index, slotSparkle, int(a.gameSpeed)+1)
	if notice.Anim != nil {
		notice.Anim.Ticks = notice.Ticks
	}
	return line
}

// noticeSpellName 是 `32D3h` 接在 "Spell:" 後面的法名：`DS:2883h + 編號 × 29h`
// 的名稱表（spec 068）。英文照原版的拼法、以大寫字模顯示；中文用說明書譯名。
func (a *app) noticeSpellName(spell uint8) string {
	if a.language == languageEnglish {
		return strings.ToUpper(a.spellLabel(spell))
	}
	return a.spellLabel(spell)
}

// holdCombatNotice 在 tacticalInput 最前面：還有停拍中的訊息就這一影格什麼都不做。
// 不停拍的那幾則直接丟掉（原版印完就接著下一步，畫面上來不及看）。
//
// 戰鬥已經結束時不停：收尾（finishCombat）與原版一樣在同一步做完。
func (a *app) holdCombatNotice(state *tacticalState) bool {
	for len(state.Notices) > 0 {
		if state.Finished {
			state.Notices = nil
			return false
		}
		if state.Notices[0].Ticks > 0 {
			state.Notices[0].Ticks--
			return true
		}
		state.Notices = state.Notices[1:]
	}
	return false
}

// shownNotice 是現在畫面上那一則（停拍中的第一則）。
func (state *tacticalState) shownNotice() (combatNotice, bool) {
	if state == nil || len(state.Notices) == 0 {
		return combatNotice{}, false
	}
	return state.Notices[0], true
}

// noticeBaseline 是第 row 列（原版 8 像素一列）的字基線：兩倍之後一列 16，
// 倚天字型的 ascent 是 14（與 combatInfoLine1 同一個算法）。
func noticeBaseline(row int) int {
	return row*16 + 14
}

// drawCombatNotice 畫停拍中的那一則。回傳 true 代表右欄那一塊被這一則佔用
// （entry 20 清的是 17h..26h × 列..15h），呼叫端就不畫 remake 自己放在那裡的說明。
func drawCombatNotice(screen *ebiten.Image, a *app, foreground, accent color.Color) (panel, footer bool) {
	notice, ok := a.tactical.shownNotice()
	if !ok {
		return false, false
	}
	palette := a.currentTheme().palette
	text := palette[noticeInkText]
	if notice.Name != "" || notice.Text != "" {
		panel = true
		drawText(screen, notice.Name, combatInfoLeft, noticeBaseline(notice.Row), palette[notice.NameInk&0x0F])
		drawNoticeLines(screen, notice.Text, notice.Row+1, text)
	}
	if notice.Target != "" {
		drawText(screen, notice.Target, combatInfoLeft, noticeBaseline(noticeRowTarget), palette[notice.TargetInk&0x0F])
		drawNoticeLines(screen, notice.Detail, noticeRowTarget+1, text)
	}
	if notice.Bottom != "" {
		drawText(screen, notice.Bottom, 0, noticeBaseline(noticeRowSpell), text)
	}
	if notice.Footer != "" {
		footer = true
		drawText(screen, notice.Footer, 0, footerBaseline, text)
	}
	return panel, footer
}

// drawNoticeLines 是 overlay-37 entry 5（`619h`）：從 row 起在右欄折行，畫到第 15h 列為止。
func drawNoticeLines(screen *ebiten.Image, value string, row int, ink color.Color) {
	for _, line := range wrapDisplay(value, combatNoteColumns) {
		if row > noticeRowLast {
			break
		}
		drawText(screen, line, combatInfoLeft, noticeBaseline(row), ink)
		row++
	}
}
