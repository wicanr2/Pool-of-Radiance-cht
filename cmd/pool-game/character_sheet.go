package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 人物資料頁的欄位（spec 130）。原版那一頁除了性別／種族／職業、年齡與六屬性，
// 還有 `LEVEL`／`EXP`、`AC`／`THAC0`／`ENCUMBRANCE`、`HP`／`DAMAGE`／`MOVEMENT`
// 與 `STATUS`。remake 先前只畫得出前半，那八個欄位在別的畫面算得出來，
// **是這一頁沒顯示**。
//
// 建角當下沒有裝備，原版顯示 `AC 10 / THAC0 20 / ENCUMBRANCE 150 /
// DAMAGE 1D2+1 / MOVEMENT 12`：AC 10 是無甲、THAC0 20 是一級、負重是身上的
// 硬幣重量、傷害是徒手 1d2 加力量的傷害調整（STR 16 給 +1）、移動力 12 是基礎值。

const (
	// 無甲的護甲等級與一級的基礎移動力，就是原版那一頁在建角當下顯示的值。
	// **護甲填的是內部值**（記錄裡存 `60 − AC`），50 就是 AC 10——
	// 傳 10 進去會算成 AC 50。
	sheetBaseArmourClass = 50
	sheetBaseMovement    = 12
	// 內部值的基準：記錄裡的 AC 與 THAC0 都存成 `60 − 值`。
	sheetInternalBase = 60
	// 徒手的傷害骰（原版建角那一頁的 `1D2`）。
	sheetUnarmedDamageCount = 1
	sheetUnarmedDamageSides = 2
)

// characterSheet 是那一頁要顯示的一組值。
type characterSheet struct {
	Level       int
	Experience  uint32
	ArmourClass int
	Thac0       int
	Encumbrance int
	Movement    int
	Damage      string
	Status      string
}

// characterSheetFor 算出一個角色在資料頁上的那幾格。
func (a *app) characterSheetFor(member poolsave.Character) (characterSheet, error) {
	sheet := characterSheet{Level: 1, Experience: member.Experience, Status: "OKAY"}
	if level := highestClassLevel(member); level > 0 {
		sheet.Level = level
	}
	var levels [gamepack.ClassThac0ClassCount]uint8
	copy(levels[:], member.ClassLevels)
	if isEmptyLevels(levels) {
		levels[gamepack.ClassSlotFighter] = uint8(sheet.Level)
	}
	internalThac0, err := gamepack.BaseThac0Internal(levels)
	if err != nil {
		return characterSheet{}, err
	}
	sheet.Thac0 = sheetInternalBase - int(internalThac0)

	internalArmour, movement, err := a.memberDefenceStats(member,
		sheetBaseArmourClass, sheetBaseMovement)
	if err != nil {
		return characterSheet{}, err
	}
	sheet.ArmourClass = sheetInternalBase - internalArmour
	sheet.Movement = int(movement)

	items := make([][]byte, 0, len(member.Inventory))
	for _, item := range member.Inventory {
		items = append(items, item.Raw)
	}
	carried, err := gamepack.CarriedWeight(items, member.Money)
	if err != nil {
		return characterSheet{}, err
	}
	sheet.Encumbrance = carried
	sheet.Damage, err = a.sheetDamage(member)
	if err != nil {
		return characterSheet{}, err
	}
	return sheet, nil
}

// sheetDamage 是資料頁上那一格傷害。備妥武器就用武器的骰，沒有就用徒手的
// 1d2；兩者都加上力量的傷害調整（spec 063 的 `0E36h`）。
func (a *app) sheetDamage(member poolsave.Character) (string, error) {
	count, sides := uint8(sheetUnarmedDamageCount), uint8(sheetUnarmedDamageSides)
	bonus := 0
	if weapon, ok := a.readiedWeapon(member); ok && a.itemTypes != nil &&
		len(weapon.Raw) > itemTypeOffset {
		entry, err := a.itemTypes.Entry(weapon.Raw[itemTypeOffset])
		if err != nil {
			return "", err
		}
		count, sides = entry.Damage()
		bonus += int(int8(entry.DamageBonus()))
	}
	index, err := gamepack.StrengthTableIndex(member.Abilities[0], member.ExceptionalStrength)
	if err == nil {
		bonus += gamepack.StrengthDamageAdjustment(index)
	}
	text := fmt.Sprintf("%dD%d", count, sides)
	switch {
	case bonus > 0:
		text += fmt.Sprintf("+%d", bonus)
	case bonus < 0:
		text += fmt.Sprintf("%d", bonus)
	}
	return text, nil
}

func highestClassLevel(member poolsave.Character) int {
	best := 0
	for _, level := range member.ClassLevels {
		if int(level) > best {
			best = int(level)
		}
	}
	return best
}

func isEmptyLevels(levels [gamepack.ClassThac0ClassCount]uint8) bool {
	for _, level := range levels {
		if level != 0 {
			return false
		}
	}
	return true
}

// 人物資料頁的版面，逐格量自原版（spec 130；`workplace/dosgolem-ref` 的
// 第 16 幀）。native 座標乘二就是這裡的數字，基線再加倚天字型的 ascent 14。
const (
	sheetLeft  = 16
	sheetLine1 = 62  // 性別／種族／AGE
	sheetLine2 = 78  // 陣營
	sheetLine3 = 94  // 職業
	sheetAbilityTop  = 126 // 六屬性，行距 16
	sheetAbilityStep = 16
	sheetAbilityValue = 86
	sheetGoldLabel    = 258
	sheetGoldValue    = 342
	sheetLevelLine  = 254
	sheetLevelValue = 118
	sheetExpLabel   = 274
	sheetExpValue   = 340
	sheetCombatLine1 = 286 // AC／THAC0／ENCUMBRANCE
	sheetCombatLine2 = 302 // HP／DAMAGE／MOVEMENT
	sheetArmourValue = 70
	sheetThac0Label  = 146
	sheetThac0Value  = 244
	sheetEncLabel    = 354
	sheetEncValue    = 550
	sheetHitPointValue = 68
	sheetDamageLabel   = 130
	sheetDamageValue   = 246
	sheetMovementLabel = 400
	sheetMovementValue = 550
	sheetStatusLine  = 366
	sheetStatusValue = 130
	// 肖像的左上角（native 216,8）。
	sheetPortraitLeft = 432
	sheetPortraitTop  = 16
)

// drawCharacterSheet 畫原版那一頁。欄名保持英文，欄位的 x 逐格對原版。
func drawCharacterSheet(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	value := a.rolled
	if value == nil {
		return
	}
	gender := a.optionText(a.flow.SelectedGender().ID, a.flow.SelectedGender().Label)
	race := a.optionText(a.flow.SelectedRace().ID, a.flow.SelectedRace().Label)
	class := a.optionText(a.flow.SelectedClass().ID, a.flow.SelectedClass().Label)
	alignment := a.optionText(a.flow.SelectedAlignment().ID, a.flow.SelectedAlignment().Label)
	drawText(screen, fmt.Sprintf("%s %s  %s", gender, race,
		fmt.Sprintf(a.text(msgAge), value.Age)), sheetLeft, sheetLine1, accent)
	drawText(screen, alignment, sheetLeft, sheetLine2, accent)
	drawText(screen, class, sheetLeft, sheetLine3, accent)

	for index := range value.Abilities {
		line := sheetAbilityTop + index*sheetAbilityStep
		drawText(screen, a.abilityName(index), sheetLeft, line, foreground)
		text := fmt.Sprintf("%d", value.Abilities[index])
		if index == 0 && value.ExceptionalStrength != 0 {
			text += fmt.Sprintf("/%02d", value.ExceptionalStrength)
		}
		drawText(screen, text, sheetAbilityValue, line, foreground)
	}
	drawText(screen, a.text(msgSheetGold), sheetGoldLabel, sheetAbilityTop, foreground)
	drawText(screen, fmt.Sprintf("%d", value.Gold), sheetGoldValue, sheetAbilityTop, foreground)

	member := a.rolledCharacter()
	sheet, err := a.characterSheetFor(member)
	if err != nil {
		drawText(screen, err.Error(), sheetLeft, sheetLevelLine, foreground)
		return
	}
	drawText(screen, a.text(msgSheetLevel), sheetLeft, sheetLevelLine, foreground)
	drawText(screen, fmt.Sprintf("%d", sheet.Level), sheetLevelValue, sheetLevelLine, foreground)
	drawText(screen, a.text(msgSheetExperience), sheetExpLabel, sheetLevelLine, foreground)
	drawText(screen, fmt.Sprintf("%d", sheet.Experience), sheetExpValue, sheetLevelLine, foreground)

	drawText(screen, a.text(msgSheetArmourClass), sheetLeft, sheetCombatLine1, foreground)
	drawText(screen, fmt.Sprintf("%d", sheet.ArmourClass), sheetArmourValue, sheetCombatLine1, foreground)
	drawText(screen, a.text(msgSheetThac0), sheetThac0Label, sheetCombatLine1, foreground)
	drawText(screen, fmt.Sprintf("%d", sheet.Thac0), sheetThac0Value, sheetCombatLine1, foreground)
	drawText(screen, a.text(msgSheetEncumbrance), sheetEncLabel, sheetCombatLine1, foreground)
	drawText(screen, fmt.Sprintf("%d", sheet.Encumbrance), sheetEncValue, sheetCombatLine1, foreground)

	drawText(screen, a.text(msgSheetHitPoints), sheetLeft, sheetCombatLine2, foreground)
	drawText(screen, fmt.Sprintf("%d", value.HP), sheetHitPointValue, sheetCombatLine2, foreground)
	drawText(screen, a.text(msgSheetDamage), sheetDamageLabel, sheetCombatLine2, foreground)
	drawText(screen, sheet.Damage, sheetDamageValue, sheetCombatLine2, foreground)
	drawText(screen, a.text(msgSheetMovement), sheetMovementLabel, sheetCombatLine2, foreground)
	drawText(screen, fmt.Sprintf("%d", sheet.Movement), sheetMovementValue, sheetCombatLine2, foreground)

	drawText(screen, a.text(msgSheetStatus), sheetLeft, sheetStatusLine, foreground)
	drawText(screen, sheet.Status, sheetStatusValue, sheetStatusLine, accent)

	// 右上角的肖像。原版那一張佔 native x 216..311、y 8..103，
	// 兩倍之後從 (432,16) 起。
	if a.portrait != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(sheetPortraitLeft, sheetPortraitTop)
		screen.DrawImage(a.portrait, op)
	}
}

// rolledCharacter 把剛擲出來的數值包成一個角色，資料頁的那幾格才算得出來。
func (a *app) rolledCharacter() poolsave.Character {
	value := a.rolled
	member := poolsave.Character{
		Abilities:           value.Abilities,
		ExceptionalStrength: value.ExceptionalStrength,
		MaxHP:               value.HP,
		CurrentHP:           value.HP,
		ClassLevels:         make([]uint8, gamepack.ClassThac0ClassCount),
	}
	member.Money[pooltreasure.Gold] = uint16(value.Gold)
	member.ClassID = a.flow.SelectedClass().ID
	// 剛擲完還沒存檔，等級表要自己填。`partyClassLevels` 在 `ClassLevels`
	// 是空的時候就照建角的第 1 級算，所以先清空再問它——這樣多重職業的
	// 每一個組成都會拿到一級，與存檔之後走的是同一條路。
	member.ClassLevels = nil
	levels := memberClassLevels(member)
	member.ClassLevels = append([]uint8(nil), levels[:]...)
	return member
}
