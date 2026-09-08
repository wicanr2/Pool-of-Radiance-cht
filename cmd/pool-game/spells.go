package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 法術一覽兼記憶畫面。施展只在戰鬥中接得起來（`cast.go`）：`SpellIsImplemented`
// 認得 42 個編號，`*` 標的就是那些——選得到的都施得出來，不讓玩家選了才失敗。
// 探索畫面那一列的 `CAST` 還沒有分派（`command_bar.go`），spec 073 的派發表
// 也還有 67 格沒讀完。
//
// 表本身是原版 START.EXE 裡那 56 筆（spec 068），順序即原版的順序。
const (
	spellTextLeft   = 48
	// 每一段之間至少 18，字高 15 加下伸 3 剛好不相碰。原本 92／106／118
	// 三段只差 12–14，槽位那一行與清單第一列直接疊在一起。
	spellFirstLine  = 134
	spellLineHeight = 16
	spellLineCount  = 8
	spellColumns    = 64
)

// spellGroups 是六個分頁，順序照原版表的分組。
var spellGroups = []struct {
	Class gamepack.SpellClass
	Level int
}{
	{gamepack.SpellClassCleric, 1}, {gamepack.SpellClassCleric, 2}, {gamepack.SpellClassCleric, 3},
	{gamepack.SpellClassMagicUser, 1}, {gamepack.SpellClassMagicUser, 2}, {gamepack.SpellClassMagicUser, 3},
}

type spellState struct {
	catalogue *gamepack.SpellCatalogue
	group     int
	cursor    int
}

func (s *spellState) current() []gamepack.Spell {
	item := spellGroups[s.group]
	return s.catalogue.ByClassAndLevel(item.Class, item.Level)
}

func (a *app) openSpells() error {
	if a.spells == nil {
		catalogue, err := gamepack.TraditionalChineseSpells()
		if err != nil {
			return err
		}
		a.spells = &spellState{catalogue: catalogue}
	}
	a.spellsOpen = true
	return nil
}

func (a *app) spellsInput() {
	state := a.spells
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyK):
		a.spellsOpen = false
	case a.justPressed(ebiten.KeyTab), a.justPressed(ebiten.KeyRight):
		state.group = (state.group + 1) % len(spellGroups)
		state.cursor = 0
	case a.justPressed(ebiten.KeyLeft):
		state.group = (state.group - 1 + len(spellGroups)) % len(spellGroups)
		state.cursor = 0
	case a.justPressed(ebiten.KeyDown):
		if group := state.current(); len(group) > 0 {
			state.cursor = (state.cursor + 1) % len(group)
		}
	case a.justPressed(ebiten.KeyUp):
		if group := state.current(); len(group) > 0 {
			state.cursor = (state.cursor - 1 + len(group)) % len(group)
		}
	case a.justPressed(ebiten.KeyM):
		a.memoriseHighlightedSpell()
	case a.justPressed(ebiten.KeyF):
		a.forgetHighlightedSpell()
	case a.justPressed(ebiten.KeyL):
		a.learnHighlightedSpell()
	default:
		for index, key := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2,
			ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6} {
			if index < len(a.state.Party) && a.justPressed(key) {
				a.spellMember = index
				a.statusLine = strings.TrimSpace(a.state.Party[index].Name)
				return
			}
		}
	}
}

// spellFacts 把參數表裡讀得出證據的三個欄位排成一行：持續、豁免、命中。
func (a *app) spellFacts(spell gamepack.Spell) string {
	id := spell.Index + 1
	if id < 1 || id >= len(a.spellParameters) {
		return ""
	}
	record := a.spellParameters[id]
	// 射程與持續都寫成「基礎值加每級增量」。原版在戰術地圖外把施法者等級
	// 當成 6，但一覽畫面不知道誰要施法，所以列的是公式而不是某個人的數字。
	baseRange, rangePerLevel := record.Range(0), record.Range(1)-record.Range(0)
	var reach string
	if rangePerLevel > 0 {
		reach = fmt.Sprintf(a.text(msgSpellsRangePerLevel), baseRange, rangePerLevel)
	} else {
		reach = fmt.Sprintf(a.text(msgSpellsRangeFixed), baseRange)
	}
	fixed, perLevel := record.Duration(0), record.Duration(1)-record.Duration(0)
	var duration string
	switch {
	case fixed > 0 && perLevel > 0:
		duration = fmt.Sprintf(a.text(msgSpellsDurationBoth), fixed, perLevel)
	case fixed > 0:
		duration = fmt.Sprintf(a.text(msgSpellsDurationFixed), fixed)
	case perLevel > 0:
		duration = fmt.Sprintf(a.text(msgSpellsDurationPerLevel), perLevel)
	default:
		duration = a.text(msgSpellsDurationUntilBroken)
	}
	save := a.text(msgSpellsSaveNone)
	if record.AllowsSavingThrow() {
		save = a.text(msgSpellsSaveSpell)
		if record.SaveCategory() == gamepack.SaveParalyzation {
			save = a.text(msgSpellsSavePoison)
		}
	}
	// 中文用全形空白分隔，英文用兩個半形——這是排版，不是可翻譯的字串。
	separator := "  "
	if a.language == languageTraditionalChinese {
		separator = "　"
	}
	line := reach + separator + duration + separator + save
	if record.RequiresAttackRoll() {
		line += separator + a.text(msgSpellsTouch)
	}
	return line
}

func drawSpells(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	state := a.spells
	for y := 40; y < 372; y++ {
		for x := 32; x < 608; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, a.text(msgSpellsTitle), 268, 62, accent)

	current := spellGroups[state.group]
	className := a.text(msgSpellsCleric)
	if current.Class == gamepack.SpellClassMagicUser {
		className = a.text(msgSpellsMagicUser)
	}
	drawText(screen, fmt.Sprintf(a.text(msgSpellsGroup), className, current.Level),
		spellTextLeft, 90, accent)

	// 被選中的人在這一級還能記幾個。上限與已記都由記錄與參數表算出來
	// （spec 072／074），不是寫死的。
	if maxima, used, ok := a.spellMemberSlots(a.spellMember); ok {
		member := a.state.Party[a.spellMember]
		spellGroup := gamepack.SpellSlotGroupCleric
		if current.Class == gamepack.SpellClassMagicUser {
			spellGroup = gamepack.SpellSlotGroupMagicUser
		}
		free := gamepack.FreeSpellSlots(maxima, used)[spellGroup][current.Level-1]
		drawText(screen, fmt.Sprintf(a.text(msgSpellsSlotLine),
			strings.TrimSpace(member.Name), a.spellMember+1,
			used[spellGroup][current.Level-1], maxima[spellGroup][current.Level-1], free),
			spellTextLeft, 112, foreground)
	}

	group := state.current()
	// 一頁放不下十三條，捲動時讓游標留在畫面內。
	first := state.cursor - spellLineCount/2
	if first < 0 {
		first = 0
	}
	if first+spellLineCount > len(group) {
		first = len(group) - spellLineCount
	}
	if first < 0 {
		first = 0
	}
	for offset := 0; offset < spellLineCount && first+offset < len(group); offset++ {
		spell := group[first+offset]
		cursor, ink := " ", foreground
		if first+offset == state.cursor {
			cursor, ink = ">", accent
		}
		// 標出哪幾條施得出來：逐支讀過的，或版型認得出來的（spec 098）。
		// 沒標的記得起來但施不出來，清單上不列——與其讓玩家選了才失敗，
		// 不如一開始就看得出差別。
		mark := " "
		if a.spellCaster.Implemented(uint8(spell.Index + 1)) {
			mark = "*"
		}
		drawText(screen, fmt.Sprintf("%s%s%-33s %s", cursor, mark, spell.Name, spell.Text),
			spellTextLeft, spellFirstLine+offset*spellLineHeight, ink)
	}

	// 說明來自說明書下冊第六章，是「說明書寫的行為」，不是反組譯出來的規則。
	if state.cursor < len(group) {
		effect := group[state.cursor].Effect
		if effect == "" {
			effect = a.text(msgSpellsNoEffect)
		}
		for index, line := range wrapDisplay(effect, spellColumns) {
			if index >= 4 {
				break
			}
			drawText(screen, line, spellTextLeft, 264+index*spellLineHeight, foreground)
		}
	}
	// 這一行的三件事全部來自原版的參數表（spec 074），不是說明書。
	if state.cursor < len(group) {
		drawText(screen, a.spellFacts(group[state.cursor]), spellTextLeft, 330, foreground)
	}
	drawText(screen, fmt.Sprintf(a.text(msgSpellsCount), len(group)), spellTextLeft, 348, foreground)
	// 鍵盤提示只有一列。原本是兩列（356 與 366），而基線只差 10、字高 15
	// ——兩列直接疊在一起，看起來像字型壞掉。底下那一列走 `footerBaseline`
	// （框外那一條），所以往下挪不了，只能併。
	drawText(screen, a.text(msgSpellsFooter), spellTextLeft, footerBaseline, accent)
}

// 記憶法術（spec 070／072／074）。原版的入口在紮營選單，remake 的紮營畫面
// （`camp.go`）也開得到它，另外掛在法術一覽上一份：1-6 挑人、M 記憶游標上
// 那一條、F 忘掉一格。
//
// 規則本身照原版接：可記憶數依職業等級與睿智算（睿智加成只給牧師），
// 那是**上限**不是遞減的剩餘量，所以「還能記幾個」是上限減掉已經記了幾個。

// spellMemberSlots 算出被選中的成員的可記憶數上限與已經記了幾個。
func (a *app) spellMemberSlots(index int) (maxima, used gamepack.SpellSlotCounts, ok bool) {
	if index < 0 || index >= len(a.state.Party) {
		return maxima, used, false
	}
	member := a.state.Party[index]
	levels := memberClassLevels(member)
	maxima = a.spellSlotTables.SpellSlotMaxima(int(levels[gamepack.ClassSlotCleric]),
		int(levels[gamepack.ClassSlotMagicUser]),
		member.Abilities[gamepack.AbilityWisdom])
	used = gamepack.MemorisedCounts(member.Memorised, a.spellParameters)
	return maxima, used, true
}

// learnHighlightedSpell 用掉一次「可以學新法術」的額度，把游標上那一條
// 寫進法術書。
//
// 原版在法師等級上升時自己叫一支選單常式（`00C9:005Ch`）挑一條寫進
// `記錄 + 32h + 編號`；那一支還沒讀出來，所以這裡把「挑」交給玩家，
// 規則本身（升一級學一條、只學得起施得出來的等級）照原版。
func (a *app) learnHighlightedSpell() {
	state := a.spells
	group := state.current()
	if len(group) == 0 || state.cursor >= len(group) {
		return
	}
	if len(a.state.Party) == 0 {
		a.statusLine = a.text(msgSpellsNeedsMember)
		return
	}
	if a.spellMember >= len(a.state.Party) {
		a.spellMember = 0
	}
	a.ensureSpellbook(a.spellMember)
	member := &a.state.Party[a.spellMember]
	name := strings.TrimSpace(member.Name)
	if member.SpellsToLearn <= 0 {
		a.statusLine = fmt.Sprintf("%s%s", name, a.text(msgSpellsNoCredit))
		return
	}
	id := uint8(group[state.cursor].Index + 1)
	if int(id) >= len(a.spellParameters) {
		return
	}
	entry := a.spellParameters[id]
	// 挑得到哪幾條照原版的 "Learn" 清單（overlay-22 entry 2 的模式 4，
	// spec 110）：**那一級有格子而且法術書上還沒有**。原版不另外看職業組
	// ——組別已經藏在「有沒有格子」裡，牧師的格子在牧師那一欄。
	maxima, _, ok := a.spellMemberSlots(a.spellMember)
	if !ok {
		return
	}
	source, level := int(entry.Source()), entry.Level()
	if source < 0 || source >= len(maxima) ||
		level < 1 || level > gamepack.SpellSlotLevels ||
		maxima[source][level-1] <= 0 {
		a.statusLine = fmt.Sprintf("%s%s%s", name, a.text(msgSpellsCannotLearn), group[state.cursor].Text)
		return
	}
	if gamepack.SpellbookKnows(member.Spellbook, id) {
		a.statusLine = fmt.Sprintf("%s%s%s", name, a.text(msgSpellsAlreadyKnown), group[state.cursor].Text)
		return
	}
	member.Spellbook = gamepack.AddToSpellbook(member.Spellbook, id)
	member.SpellsToLearn--
	syncTrainedLibraryCharacter(&a.state, *member)
	a.statusLine = fmt.Sprintf("%s%s%s", name, a.text(msgSpellsLearned), group[state.cursor].Text)
}

// memoriseHighlightedSpell 把游標上那一條記給被選中的成員。
func (a *app) memoriseHighlightedSpell() {
	state := a.spells
	group := state.current()
	if len(group) == 0 || state.cursor >= len(group) {
		return
	}
	if len(a.state.Party) == 0 {
		a.statusLine = a.text(msgSpellsNeedsMember)
		return
	}
	if a.spellMember >= len(a.state.Party) {
		a.spellMember = 0
	}
	a.ensureSpellbook(a.spellMember)
	member := &a.state.Party[a.spellMember]
	if len(member.Memorised) < gamepack.MemorisedSpellSlots {
		grown := make([]uint8, gamepack.MemorisedSpellSlots)
		copy(grown, member.Memorised)
		member.Memorised = grown
	}
	maxima, _, ok := a.spellMemberSlots(a.spellMember)
	if !ok {
		return
	}
	id := uint8(group[state.cursor].Index + 1)
	// 書上沒有就記不起來（spec 110）。原版的記憶畫面在 overlay-15 `0C33h`
	// 逐條檢查 `記錄 + 32h + 編號`，沒有的那一條連列都不列。
	if known, _ := a.memberSpellbook(*member); len(known) != 0 && !gamepack.SpellbookKnows(known, id) {
		a.statusLine = fmt.Sprintf("%s%s%s", strings.TrimSpace(member.Name),
			a.text(msgSpellsNotInBook), group[state.cursor].Text)
		return
	}
	if err := gamepack.Memorise(member.Memorised, id, a.spellParameters, maxima); err != nil {
		a.statusLine = fmt.Sprintf("%s%s", strings.TrimSpace(member.Name), a.text(msgSpellsNoSlot))
		return
	}
	syncTrainedLibraryCharacter(&a.state, *member)
	a.statusLine = fmt.Sprintf("%s%s%s", strings.TrimSpace(member.Name),
		a.text(msgSpellsMemorised), group[state.cursor].Text)
}

// forgetHighlightedSpell 忘掉被選中的成員身上第一個符合游標的那一格。
func (a *app) forgetHighlightedSpell() {
	state := a.spells
	group := state.current()
	if len(group) == 0 || state.cursor >= len(group) || len(a.state.Party) == 0 {
		return
	}
	if a.spellMember >= len(a.state.Party) {
		a.spellMember = 0
	}
	member := &a.state.Party[a.spellMember]
	id := uint8(group[state.cursor].Index + 1)
	for slot, value := range member.Memorised {
		if value&0x7f == id {
			if err := gamepack.ForgetMemorised(member.Memorised, slot); err != nil {
				return
			}
			syncTrainedLibraryCharacter(&a.state, *member)
			a.statusLine = fmt.Sprintf("%s%s%s", strings.TrimSpace(member.Name),
				a.text(msgSpellsForgot), group[state.cursor].Text)
			return
		}
	}
	a.statusLine = fmt.Sprintf("%s%s%s", strings.TrimSpace(member.Name),
		a.text(msgSpellsNotMemorised), group[state.cursor].Text)
}
