package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// 原版的 combat icon editor 是巢狀選單（spec 003 第 7..11 步）：
//
//	PARTS / COLOR-1 / COLOR-2 / SIZE / EXIT
//	  PARTS   → HEAD / WEAPON / EXIT      → NEXT / PREV / KEEP / EXIT
//	  COLOR-1 → WEAPON / BODY / HAIR / SHIELD / ARM / LEG / EXIT
//	                                       → NEXT / PREV / KEEP / EXIT
//	  COLOR-2 → 同上，HAIR 換成 FACE
//	  SIZE    → LARGE / KEEP / EXIT       （只有預設小號的種族看得到）
//	  EXIT    → IS THIS ICON OK? YES NO
//
// `NEXT / PREV / KEEP / EXIT` 那一層裡，**KEEP 接受目前的候選、EXIT 退回
// 進來時的值**。兩個都回上一層，差別只在改不改得算數——原版同時給這兩個
// 出口，不然其中一個就是多餘的。這一條是**強推論**：spec 003 只記到
//「以 `NEXT / PREV / KEEP / EXIT` 循環並接受候選」。

// iconMenuLevel 是目前停在哪一層。
type iconMenuLevel uint8

const (
	iconMenuTop iconMenuLevel = iota
	iconMenuParts
	iconMenuPartCycle
	iconMenuColour
	iconMenuColourCycle
	iconMenuSize
)

// iconMenuState 是巢狀選單自己的狀態。角色資料仍然只放在 creation.Flow 裡，
// 這裡只記「停在哪一層」與「進來時的值」。
type iconMenuState struct {
	level  iconMenuLevel
	cursor int
	// component 是 COLOR-1（0）或 COLOR-2（1）。
	component int
	// editingWeapon 為真時 PARTS 那一層改的是武器，否則是頭。
	editingWeapon bool
	// restore 是進 NEXT／PREV 迴圈時的值，EXIT 要退回它。
	restore uint8
}

// iconMenuOption 是選單上的一列：顯示字與它的首字母鍵。
type iconMenuOption struct {
	label string
	key   ebiten.Key
}

// iconColourPartOrder 是 COLOR 子選單的顯示順序（spec 003 第 9 步）與它們在
// Flow.IconColors 的索引。索引順序是記錄 `+0C1h..+0C6h` 的順序（spec 007），
// 兩者不同——畫面上武器排第一，記錄裡它是最後一格。
var iconColourPartOrder = []struct {
	label string
	index int
}{
	{"WEAPON", 5}, {"BODY", 0}, {"HAIR", 3}, {"SHIELD", 4}, {"ARM", 1}, {"LEG", 2},
}

// iconMenuOptions 回傳目前這一層的選項。
func (a *app) iconMenuOptions() []iconMenuOption {
	switch a.iconMenu.level {
	case iconMenuTop:
		options := []iconMenuOption{
			{"PARTS", ebiten.KeyP}, {"COLOR-1", ebiten.KeyDigit1}, {"COLOR-2", ebiten.KeyDigit2},
		}
		if a.flow.UsesIconSizeMenu() {
			options = append(options, iconMenuOption{"SIZE", ebiten.KeyS})
		}
		return append(options, iconMenuOption{"EXIT", ebiten.KeyE})
	case iconMenuParts:
		return []iconMenuOption{{"HEAD", ebiten.KeyH}, {"WEAPON", ebiten.KeyW}, {"EXIT", ebiten.KeyE}}
	case iconMenuColour:
		options := make([]iconMenuOption, 0, len(iconColourPartOrder)+1)
		for _, part := range iconColourPartOrder {
			label := part.label
			// COLOR-2 把 HAIR 換成 FACE（spec 003 第 9 步）。同一格資料，
			// 兩個名字——記錄的 `+0C4h` 本來就是 hair/face 合一。
			if label == "HAIR" && a.iconMenu.component == 1 {
				label = "FACE"
			}
			options = append(options, iconMenuOption{label, iconPartKey(label)})
		}
		return append(options, iconMenuOption{"EXIT", ebiten.KeyE})
	case iconMenuPartCycle, iconMenuColourCycle:
		return []iconMenuOption{
			{"NEXT", ebiten.KeyN}, {"PREV", ebiten.KeyP},
			{"KEEP", ebiten.KeyK}, {"EXIT", ebiten.KeyE},
		}
	case iconMenuSize:
		return []iconMenuOption{{"LARGE", ebiten.KeyL}, {"KEEP", ebiten.KeyK}, {"EXIT", ebiten.KeyE}}
	}
	return nil
}

// iconPartKey 是部位的首字母。HAIR 與 FACE 共用一格但首字母不同，
// 所以不能只看索引。
func iconPartKey(label string) ebiten.Key {
	switch label {
	case "WEAPON":
		return ebiten.KeyW
	case "BODY":
		return ebiten.KeyB
	case "HAIR":
		return ebiten.KeyH
	case "FACE":
		return ebiten.KeyF
	case "SHIELD":
		return ebiten.KeyS
	case "ARM":
		return ebiten.KeyA
	case "LEG":
		return ebiten.KeyL
	}
	return ebiten.KeyMax
}

// iconMenuInput 走一格輸入。方向鍵加 ENTER 或直接按首字母都可以——原版是
// 「以第一個字選擇之」，方向鍵是 remake 補的。
func (a *app) iconMenuInput() error {
	options := a.iconMenuOptions()
	if len(options) == 0 {
		return nil
	}
	if a.justPressed(ebiten.KeyArrowUp) {
		a.iconMenu.cursor = (a.iconMenu.cursor + len(options) - 1) % len(options)
		return nil
	}
	if a.justPressed(ebiten.KeyArrowDown) {
		a.iconMenu.cursor = (a.iconMenu.cursor + 1) % len(options)
		return nil
	}
	if a.iconMenu.cursor >= len(options) {
		a.iconMenu.cursor = 0
	}
	if a.justPressed(ebiten.KeyEnter) {
		return a.chooseIconMenu(options[a.iconMenu.cursor].label)
	}
	for _, option := range options {
		if option.key != ebiten.KeyMax && a.justPressed(option.key) {
			return a.chooseIconMenu(option.label)
		}
	}
	return nil
}

// chooseIconMenu 執行這一層的某一項。
func (a *app) chooseIconMenu(label string) error {
	state := &a.iconMenu
	switch state.level {
	case iconMenuTop:
		switch label {
		case "PARTS":
			state.level, state.cursor = iconMenuParts, 0
		case "COLOR-1":
			state.level, state.cursor, state.component = iconMenuColour, 0, 0
		case "COLOR-2":
			state.level, state.cursor, state.component = iconMenuColour, 0, 1
		case "SIZE":
			state.level, state.cursor, state.restore = iconMenuSize, 0, a.flow.IconSize
		case "EXIT":
			return a.flow.RequestIconConfirmation()
		}
		return nil
	case iconMenuParts:
		switch label {
		case "HEAD":
			state.editingWeapon, state.restore = false, a.flow.IconHead
			state.level, state.cursor = iconMenuPartCycle, 0
		case "WEAPON":
			state.editingWeapon, state.restore = true, a.flow.IconWeapon
			state.level, state.cursor = iconMenuPartCycle, 0
		case "EXIT":
			state.level, state.cursor = iconMenuTop, 0
		}
		return nil
	case iconMenuColour:
		if label == "EXIT" {
			state.level, state.cursor = iconMenuTop, 0
			return nil
		}
		for _, part := range iconColourPartOrder {
			if part.label == label || (part.label == "HAIR" && label == "FACE") {
				if err := a.flow.SelectIconPart(part.index); err != nil {
					return err
				}
				state.restore = a.flow.IconColors[part.index][state.component]
				state.level, state.cursor = iconMenuColourCycle, 0
				return nil
			}
		}
		return nil
	case iconMenuPartCycle:
		return a.cycleIconPart(label)
	case iconMenuColourCycle:
		return a.cycleIconColour(label)
	case iconMenuSize:
		switch label {
		case "LARGE":
			if err := a.flow.SetIconSize(2); err != nil {
				return err
			}
			state.level, state.cursor = iconMenuTop, 0
			return a.reloadIcons()
		case "KEEP":
			state.level, state.cursor = iconMenuTop, 0
		case "EXIT":
			if err := a.flow.SetIconSize(state.restore); err != nil {
				return err
			}
			state.level, state.cursor = iconMenuTop, 0
			return a.reloadIcons()
		}
		return nil
	}
	return nil
}

// cycleIconPart 是 HEAD／WEAPON 的 NEXT／PREV／KEEP／EXIT。
func (a *app) cycleIconPart(label string) error {
	state := &a.iconMenu
	var err error
	switch label {
	case "NEXT":
		if state.editingWeapon {
			err = a.flow.NextIconWeapon()
		} else {
			err = a.flow.NextIconHead()
		}
	case "PREV":
		if state.editingWeapon {
			err = a.flow.PreviousIconWeapon()
		} else {
			err = a.flow.PreviousIconHead()
		}
	case "KEEP":
		state.level, state.cursor = iconMenuParts, 0
		return nil
	case "EXIT":
		if state.editingWeapon {
			a.flow.IconWeapon = state.restore
		} else {
			a.flow.IconHead = state.restore
		}
		state.level, state.cursor = iconMenuParts, 0
		return a.reloadIcons()
	default:
		return nil
	}
	if err != nil {
		return err
	}
	return a.reloadIcons()
}

// cycleIconColour 是某一個部位的 NEXT／PREV／KEEP／EXIT。
func (a *app) cycleIconColour(label string) error {
	state := &a.iconMenu
	var err error
	switch label {
	case "NEXT":
		err = a.flow.NextIconColor(state.component)
	case "PREV":
		err = a.flow.PreviousIconColor(state.component)
	case "KEEP":
		state.level, state.cursor = iconMenuColour, 0
		return nil
	case "EXIT":
		a.flow.IconColors[a.flow.IconPart][state.component] = state.restore
		state.level, state.cursor = iconMenuColour, 0
		return a.reloadIcons()
	default:
		return nil
	}
	if err != nil {
		return err
	}
	return a.reloadIcons()
}

// iconMenuPath 是畫面上那一行「現在在哪一層」。
func (a *app) iconMenuPath() string {
	switch a.iconMenu.level {
	case iconMenuParts:
		return "PARTS"
	case iconMenuPartCycle:
		if a.iconMenu.editingWeapon {
			return fmt.Sprintf("PARTS / WEAPON %02d", a.flow.IconWeapon)
		}
		return fmt.Sprintf("PARTS / HEAD %02d", a.flow.IconHead)
	case iconMenuColour:
		return colourMenuName(a.iconMenu.component)
	case iconMenuColourCycle:
		return fmt.Sprintf("%s / %s %X", colourMenuName(a.iconMenu.component),
			iconPartLabel(int(a.flow.IconPart), a.iconMenu.component),
			a.flow.IconColors[a.flow.IconPart][a.iconMenu.component])
	case iconMenuSize:
		return "SIZE"
	}
	return "COMBAT ICON EDITOR"
}

func colourMenuName(component int) string {
	if component == 1 {
		return "COLOR-2"
	}
	return "COLOR-1"
}

// iconPartLabel 把 Flow.IconColors 的索引換回畫面上的名字。
func iconPartLabel(index, component int) string {
	for _, part := range iconColourPartOrder {
		if part.index != index {
			continue
		}
		if part.label == "HAIR" && component == 1 {
			return "FACE"
		}
		return part.label
	}
	return "?"
}

// resetIconMenu 在進入 icon editor 時把選單收回頂層。
func (a *app) resetIconMenu() { a.iconMenu = iconMenuState{} }
