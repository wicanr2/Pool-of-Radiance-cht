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

func init() {
	for id, key := range map[messageID]string{
		msgFoeSpellLine: "ui.foeSpellLine",
		msgFoeItemLine:  "ui.foeItemLine",
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
)

// combatNotice 是一次 entry 20 或 entry 19 的輸出，停拍倒數完才換下一則。
type combatNotice struct {
	// Name 與 Text 畫在右欄：Name 在 Row 那一列，Text 從下一列起折行。
	Name string
	Text string
	Row  int
	// Bottom 是第 23 列（17h）欄 0 的那一行（`32D3h` 的 "Spell:" 加法名）。
	Bottom string
	// Footer 是 entry 19 印在第 24 列欄 0 的固定字串，停拍時取代指令列。
	Footer string
	// Ticks 是還要停幾個影格。0 代表這一則不停拍。
	Ticks int
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

// panelNotice 是 overlay-25 entry 20：名字加一句，beat 為真時停一拍。回傳記錄用的
// 一行（名字、空白、那一句）。
func (a *app) panelNotice(state *tacticalState, index uint8, text string, row int, beat bool) string {
	name := a.combatantName(state, index)
	notice := combatNotice{Name: name, Text: text, Row: row}
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

// itemUseLine 是 overlay-19 entry 8 的 `1B2Dh..1B8Dh`：entry 20(記錄, "uses an item",
// 0Ah, 0)——**不停拍**——接著第 23 列欄 0 印 "Item:"、物品名由 overlay-25 entry 1 從
// 欄 5 起印。entry 1 停不停拍沒有讀，所以這一句只進記錄，不排進停拍佇列。
func (a *app) itemUseLine(state *tacticalState, mover uint8, item string) string {
	return a.combatantName(state, mover) + " " + state.say(msgFoeUsesItem) + " " +
		state.say(msgFoeItemLine, item)
}

// noticeSpellName 是 `32D3h` 接在 "Spell:" 後面的法名：`DS:2883h + 編號 × 29h`
// 的名稱表（spec 068）。英文照原版的拼法、以大寫字模顯示；中文用說明書譯名。
func (a *app) noticeSpellName(spell uint8) string {
	if a.language == languageEnglish {
		if a.spells == nil {
			a.spellLabel(spell)
		}
		if a.spells != nil {
			if entry, err := a.spells.catalogue.SpellByID(spell); err == nil {
				return strings.ToUpper(entry.Name)
			}
		}
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
	if notice.Name != "" || notice.Text != "" {
		panel = true
		drawText(screen, notice.Name, combatInfoLeft, noticeBaseline(notice.Row), accent)
		row := notice.Row + 1
		for _, line := range wrapDisplay(notice.Text, combatNoteColumns) {
			if row > noticeRowLast {
				break
			}
			drawText(screen, line, combatInfoLeft, noticeBaseline(row), foreground)
			row++
		}
	}
	if notice.Bottom != "" {
		drawText(screen, notice.Bottom, 0, noticeBaseline(noticeRowSpell), foreground)
	}
	if notice.Footer != "" {
		footer = true
		drawText(screen, notice.Footer, 0, footerBaseline, foreground)
	}
	return panel, footer
}
