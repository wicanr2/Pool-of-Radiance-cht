package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 人物管理選擇項（Party Creation Menu）。十一個指令與它們的可見規則來自
// 說明書 p.8..p.10；四個指令的畫面順序另有原版截圖佐證（spec 008）。
//
// **可見規則**：「若你未將任何人物加入隊伍，則大部份的人物處理選擇項都將
// 不會出現」——隊伍是空的時候只剩 C、A、L、E 四項，正好是那張截圖上的四項。
// 需要對象的七項（D、M、T、V、R）與需要隊伍的兩項（S、B）都藏起來。

// partyMenuEntry 是選單上的一列。
type partyMenuEntry struct {
	key ebiten.Key
	// message 是那一列的字串。
	message messageID
	// needsParty 為真時，隊伍空的就不顯示也不接受按鍵。
	needsParty bool
	// emptyOnly 為真時反過來：**隊伍有人就不顯示**。
	// `L)OAD SAVED GAME` 是唯一一個——原版隊伍非空的截圖上沒有它。
	emptyOnly bool
}

// partyMenuEntries 依說明書 p.8..p.10 的敘述順序列出十一項。
func partyMenuEntries() []partyMenuEntry {
	return []partyMenuEntry{
		{ebiten.KeyC, msgMenuCreate, false, false},
		{ebiten.KeyD, msgMenuDrop, true, false},
		{ebiten.KeyM, msgMenuModify, true, false},
		// T)RAIN **原版在這個畫面上不顯示**（兩張原版截圖都沒有它）。
		// 它由 `DS:06D4h` 那個旗標控制，而那個旗標從哪裡設還沒讀出來，
		// 所以 remake 先一律顯示——藏起來玩家就昇不了級。這是**已知的
		// 偏差**，不是照原版接的。
		{ebiten.KeyT, msgMenuTrain, true, false},
		{ebiten.KeyV, msgMenuView, true, false},
		{ebiten.KeyA, msgMenuAdd, false, false},
		{ebiten.KeyR, msgMenuRemove, true, false},
		// L)OAD 只在隊伍是空的時候出現：原版隊伍非空的截圖上沒有它，
		// 空隊伍那一張有。
		{ebiten.KeyL, msgMenuLoad, false, true},
		{ebiten.KeyS, msgMenuSave, true, false},
		{ebiten.KeyB, msgMenuBegin, true, false},
		{ebiten.KeyE, msgMenuExit, false, false},
	}
}

// visiblePartyMenuEntries 過掉隊伍空的時候不該出現的那幾項。
func (a *app) visiblePartyMenuEntries() []partyMenuEntry {
	hasParty := len(a.state.Party) > 0
	visible := make([]partyMenuEntry, 0, len(partyMenuEntries()))
	for _, entry := range partyMenuEntries() {
		if entry.needsParty && !hasParty {
			continue
		}
		if entry.emptyOnly && hasParty {
			continue
		}
		visible = append(visible, entry)
	}
	return visible
}

// menuMemberOrZero 把選中的成員夾回合法範圍。
func (a *app) menuMemberOrZero() int {
	if a.menuMember < 0 || a.menuMember >= len(a.state.Party) {
		return 0
	}
	return a.menuMember
}

// dropMember 是 D）ROP CHARACTER：「將一名現在在隊伍中的人物從人物名單中
// 永遠除掉，他的一切資料都將消失」（說明書 p.8）。所以隊伍與人物名單兩邊
// 都要刪；只從隊伍拿掉是 R）EMOVE，不是這一項。
func (a *app) dropMember(index int) error {
	if index < 0 || index >= len(a.state.Party) {
		return fmt.Errorf("Pool party has no member %d", index+1)
	}
	name := a.state.Party[index].Name
	a.state.Party = append(a.state.Party[:index], a.state.Party[index+1:]...)
	for library := range a.state.CharacterLibrary {
		if a.state.CharacterLibrary[library].Name == name {
			a.state.CharacterLibrary = append(a.state.CharacterLibrary[:library],
				a.state.CharacterLibrary[library+1:]...)
			break
		}
	}
	a.menuMember = 0
	return a.persistState(fmt.Sprintf(a.text(msgMenuDropped), name))
}

// removeMember 是 R）EMOVE CHARACTER FROM PARTY：「將人物自隊伍中移回
// 『人物名單』中」（說明書 p.9）。人還在，只是不在隊伍裡。
func (a *app) removeMember(index int) error {
	if index < 0 || index >= len(a.state.Party) {
		return fmt.Errorf("Pool party has no member %d", index+1)
	}
	member := a.state.Party[index]
	a.state.Party = append(a.state.Party[:index], a.state.Party[index+1:]...)
	found := false
	for _, existing := range a.state.CharacterLibrary {
		if existing.Name == member.Name {
			found = true
			break
		}
	}
	if !found {
		a.state.CharacterLibrary = append(a.state.CharacterLibrary, member)
	}
	a.menuMember = 0
	return a.persistState(fmt.Sprintf(a.text(msgMenuRemoved), member.Name))
}

// modifyMember 是 M）ODIFY CHARACTER：「重新調整隊伍中新創造人物的各項屬性
// 與生命力」，兩項限制是「人物的經驗點數必須為 0，身上不能有任何金錢以外的
// 物品」（說明書 p.8）。
//
// 重擲用的是建角同一支產生器，所以種族調整、年齡調整、職業下限與金幣、
// 生命骰全部照原版的規則走，不是另一套「修改」邏輯。
func (a *app) modifyMember(index int) error {
	if index < 0 || index >= len(a.state.Party) {
		return fmt.Errorf("Pool party has no member %d", index+1)
	}
	member := a.state.Party[index]
	if member.Experience != 0 {
		a.statusLine = a.text(msgMenuModifyExperience)
		return nil
	}
	if len(member.Inventory) != 0 {
		a.statusLine = a.text(msgMenuModifyItems)
		return nil
	}
	race, ok := findRace(member.RaceID)
	if !ok {
		return fmt.Errorf("Pool race %q is not in the catalog", member.RaceID)
	}
	gender, ok := findGender(member.GenderID)
	if !ok {
		return fmt.Errorf("Pool gender %q is not in the catalog", member.GenderID)
	}
	class, ok := findClass(member.RaceID, member.ClassID)
	if !ok {
		return fmt.Errorf("Pool class %q is not open to %q", member.ClassID, member.RaceID)
	}
	rolled := creation.RollCharacter(a.roller, race, gender, class)
	if rolled.Gold < 0 || rolled.Gold > int(^uint16(0)) {
		return fmt.Errorf("Pool character gold %d is outside uint16", rolled.Gold)
	}
	member.Age = rolled.Age
	member.Abilities = rolled.Abilities
	member.ExceptionalStrength = rolled.ExceptionalStrength
	member.MaxHP, member.CurrentHP, member.RawHP = rolled.HP, rolled.HP, rolled.RawHP
	member.Money = [7]uint16{}
	member.Money[pooltreasure.Gold] = uint16(rolled.Gold)
	// 法術書跟著新的睿智重算：原版的建角規則就是照能力值填（spec 109）。
	member.Spellbook = gamepack.NewCharacterSpellbook(memberClassLevels(member),
		member.Abilities[gamepack.AbilityWisdom], a.spellSlotTables, a.spellParameters)
	a.state.Party[index] = member
	a.replaceLibraryCharacter(member)
	return a.persistState(fmt.Sprintf(a.text(msgMenuModified), member.Name))
}

// replaceLibraryCharacter 讓人物名單裡的同一個人整份跟著改。名單與隊伍是
// 同一個人的兩份拷貝，只改一邊會讓 R）EMOVE 之後看到舊資料。
//
// 與 rob.go 的 syncLibraryCharacter 不同：那一支只同步錢與物品（被搶走的
// 東西），這一支換掉整份記錄，因為 M）ODIFY 重擲的是能力值與生命力。
func (a *app) replaceLibraryCharacter(member poolsave.Character) {
	for index := range a.state.CharacterLibrary {
		if a.state.CharacterLibrary[index].Name == member.Name {
			a.state.CharacterLibrary[index] = member
			return
		}
	}
}

// persistState 存檔並設好狀態列。存不了就把錯誤顯示出來——這裡的每一項都
// 改動了人物名單，靜靜失敗會讓玩家以為改好了。
func (a *app) persistState(line string) error {
	if a.saveState != nil {
		if err := a.saveState(a.state); err != nil {
			a.statusLine = err.Error()
			return nil
		}
	}
	a.statusLine = line
	return nil
}

func findRace(id string) (creation.Race, bool) {
	for _, race := range creation.Races {
		if race.ID == id {
			return race, true
		}
	}
	return creation.Race{}, false
}

func findGender(id string) (creation.Gender, bool) {
	for _, gender := range creation.Genders {
		if gender.ID == id {
			return gender, true
		}
	}
	return creation.Gender{}, false
}

func findClass(raceID, classID string) (creation.ClassChoice, bool) {
	for _, choice := range creation.ClassesForRace(raceID) {
		if choice.ID == classID {
			return choice, true
		}
	}
	return creation.ClassChoice{}, false
}

// partyMenuCommand 依可見的那幾項分派按鍵。看不見的指令連按鍵都不接受——
// 畫面上沒有那一列卻按得動，等於藏了一個原版沒有的入口。
func (a *app) partyMenuCommand() error {
	for _, entry := range a.visiblePartyMenuEntries() {
		if !a.justPressed(entry.key) {
			continue
		}
		return a.runPartyMenuEntry(entry)
	}
	// ENTER 等同 C：標題進來之後最常做的就是建角。
	if a.justPressed(ebiten.KeyEnter) {
		return a.runPartyMenuEntry(partyMenuEntry{key: ebiten.KeyC, message: msgMenuCreate})
	}
	return nil
}

func (a *app) runPartyMenuEntry(entry partyMenuEntry) error {
	member := a.menuMemberOrZero()
	switch entry.key {
	case ebiten.KeyC:
		a.flow, a.cursor, a.rolled = creation.NewFlow(), 0, nil
		a.mode = modeCreation
		return nil
	case ebiten.KeyD:
		a.menuDropPending = true
		a.statusLine = fmt.Sprintf(a.text(msgMenuDropPrompt), a.state.Party[member].Name)
		return nil
	case ebiten.KeyM:
		return a.modifyMember(member)
	case ebiten.KeyT:
		line, err := a.trainMember(member)
		if err != nil {
			return err
		}
		a.statusLine = line
		return nil
	case ebiten.KeyV:
		// V）IEW 是「檢視人物的各項資料，以及進行交換錢幣或裝備之類的工作」
		//（說明書 p.9），也就是 remake 的裝備畫面。
		a.openEquipment()
		if a.equipment != nil {
			a.equipment.member = member
			a.equipment.item = 0
			a.equipment.clamp(a.state.Party)
		}
		return nil
	case ebiten.KeyA:
		return a.addFirstLibraryCharacter()
	case ebiten.KeyR:
		return a.removeMember(member)
	case ebiten.KeyL:
		return a.loadSavedGame()
	case ebiten.KeyS:
		return a.saveCurrentGame()
	case ebiten.KeyB:
		return a.beginAdventuring()
	case ebiten.KeyE:
		return a.exitToDOS()
	}
	return nil
}

// partyMenuDropConfirm 是 D）ROP 的再確認：「為了慎重起見，會再讓你確定
// 一次（是否真要除掉？ Y)ES／N)O）」（說明書 p.8）。
func (a *app) partyMenuDropConfirm() error {
	switch {
	case a.justPressed(ebiten.KeyY):
		a.menuDropPending = false
		return a.dropMember(a.menuMemberOrZero())
	case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape):
		a.menuDropPending = false
		a.statusLine = ""
	}
	return nil
}

// saveCurrentGame 是 S）AVE CURRENT GAME：「存下目前的遊戲進度」（說明書 p.9）。
// 原版分成 A..J 十個存檔位；remake 只有一份 JSON，所以這裡就是把它寫出去。
func (a *app) saveCurrentGame() error {
	if a.saveState == nil {
		a.statusLine = a.text(msgMenuNeedsMember)
		return nil
	}
	state, err := a.stateForSave()
	if err != nil {
		a.statusLine = err.Error()
		return nil
	}
	if err := a.saveState(state); err != nil {
		a.statusLine = err.Error()
		return nil
	}
	a.state = state
	a.statusLine = a.text(msgMenuSaved)
	return nil
}

// exitToDOS 是 E）XIT TO DOS：「跳回 DOS，結束遊戲」（說明書 p.10）。
// 與 F10 不同：F10 先存再離開，這一項照原版的敘述直接結束。
func (a *app) exitToDOS() error { return ebiten.Termination }

// 選單版面：左欄指令、右欄隊伍，兩欄同一條基線。行距沿用手冊那一頁量過的
// 數字放寬到 20，十一項才不會撞到狀態列。
const (
	partyMenuFirstLine  = 90
	partyMenuLineHeight = 20
)
