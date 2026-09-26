package gamepack

// 物品選單（overlay-19 entry 6，`0EFBh`）裡 Ready、Halve、Join 三支的規則，以及
// 卷軸那兩支（overlay-22 entry 12 列法術、entry 7 抹掉）。spec 144。
//
// 這一層只動物品記錄的位元組；誰按了什麼鍵、訊息怎麼印在 cmd/pool-game。
// 物品記錄沿用存檔的 63-byte 版面（spec 033），`+2Ah`／`+2Ch` 的串列指標在
// remake 裡換成切片順序。

// 物品記錄裡這幾支用到、別處還沒命名的欄位。
const (
	// ItemNameWordOffset 是名稱的三個字詞編號（`+2Fh`／`+30h`／`+31h`，spec 067）。
	ItemNameWordOffset = 0x2f
	// ItemCursedOffset 是 `+36h`：非 0 卸不下來（overlay-19 `14FAh`）。
	ItemCursedOffset = 0x36
	// ItemHiddenNameOffset 是 `+35h`：名稱藏字的位元（spec 067）；卷軸要它為 0 才讀得出內容。
	ItemHiddenNameOffset = 0x35
	// ItemEffectOffset 是 `+3Eh`：大於 7Fh 時是穿戴效果的代碼（overlay-19 `14D4h`）。
	ItemEffectOffset = 0x3e

	// ItemTypeHandsOffset 是型別表 `+1`（`DS:54E1h`）：拿這件要幾隻手。
	ItemTypeHandsOffset = 0x01
	// ItemTypeClassMaskOffset 是型別表 `+0Dh`（`DS:54EDh`）：哪幾類職業能用。
	ItemTypeClassMaskOffset = 0x0d

	// ItemTypeArrow 與 ItemTypeQuarrel 是另外占 `+F8h`／`+FCh` 的兩種型別索引
	// （overlay-25 `0D05h`／`0D21h`，spec 065）。
	ItemTypeArrow   = 0x49
	ItemTypeQuarrel = 0x1c

	// readyHandLimit 是 overlay-19 `156Eh` 的 `cmp ax, 2`：手數加起來大於 2 就拿不動。
	readyHandLimit = 2
	// equipmentRingCategory 是兩個戒指槽（`+F0h`／`+F4h`）的類別。
	equipmentRingCategory = 9
	// equipmentSlotCategories 是 `+CCh + 類別 × 4` 那九格（類別 0..8）。
	equipmentSlotCategories = 9

	// HalveCountLimit 是物品選單接 " Halve" 的條件：角色身上不到 10h 件（`1071h`）。
	HalveCountLimit = 0x10
	// joinCountLimit 是 Join 疊到一件的上限（`19F4h` 的 `cmp ax, 0FFh`）。
	joinCountLimit = 0xff

	// ScrollCleric 是牧師卷軸的類別（overlay-22 `047Ch` 的 `cmp …, 0Ch`）。
	ScrollCleric = 0x0c
	// ReadMagicEffectCode 是閱讀魔法（法術 18）掛上的效果碼；overlay-22 `0457h`
	// 以 overlay-25 entry 27（`21DCh`）問身上有沒有 10h。
	ReadMagicEffectCode = 0x10
	// scrollSpellFirst..scrollSpellLast 是卷軸的三行（`+3Ch..+3Eh`，`04ABh` 的 `+3Bh + i`）。
	scrollSpellFirst = 0x3c
	scrollSpellLast  = 0x3e
	// scrollOneSpellWord 是名稱字詞 D2h "With 1 Spell"；`+30h` 減到它以下就把卷軸拿掉
	// （overlay-22 `32A0h` 的 `cmp …, 0D2h`）。
	scrollOneSpellWord = 0xd2
)

// ClassUseMasks 是 `DS:05EAh` 的八格：職業索引（記錄 `+96h` 的順序）對到型別表
// `+0Dh` 的位元。overlay-16 `0D3Fh` 對每個等級大於 0 的職業把這一格加進記錄 `+0B0h`。
// START.EXE 的位元組是 `02 10 08 40 80 01 04 20`（測試對過）。
var ClassUseMasks = [ClassThac0ClassCount]uint8{0x02, 0x10, 0x08, 0x40, 0x80, 0x01, 0x04, 0x20}

// ClassUseMaskAddress 是那八格在 START.EXE 的 DS 位移。
const ClassUseMaskAddress = 0x05ea

// ClassUseMask 是記錄 `+0B0h`：overlay-16 `0CAEh` 先清 0，`0CD1h..0D4Fh` 對每個等級
// 大於 0 的職業加上它那一位（`0CE3h` 的 `jle` 同時跳過 THAC0 與這一格）。
func ClassUseMask(levels [ClassThac0ClassCount]uint8) uint8 {
	var mask uint8
	for class, level := range levels {
		if int8(level) > 0 {
			mask += ClassUseMasks[class]
		}
	}
	return mask
}

// ReadyOutcome 是 overlay-19 entry 7（`14CAh`）的結果。
type ReadyOutcome uint8

const (
	// ReadyDone 是裝上了（`1642h` 寫 `+34h = 1`）。
	ReadyDone ReadyOutcome = iota
	// UnreadyDone 是卸下了（`151Ah` 寫 `+34h = 0`）。
	UnreadyDone
	// ReadyCursed 是 `+36h` 非 0 卸不下來，印 "It's Cursed"（`148Eh`）。
	ReadyCursed
	// ReadyWrongClass 是型別表 `+0Dh` 與角色 `+0B0h` 沒有交集，印 "Wrong Class"（`149Ah`）。
	ReadyWrongClass
	// ReadyAlreadyUsing 是那一格已經有東西，印 "already using " 加那一件的名字（`14A6h`）。
	ReadyAlreadyUsing
	// ReadyHandsFull 是手數超過 2，印 "Your hands are full!"（`14B5h`）。
	ReadyHandsFull
)

// ReadyResult 帶著結果與「擋住的那一件」在切片裡的位置（只有 ReadyAlreadyUsing 用）。
type ReadyResult struct {
	Outcome ReadyOutcome
	Blocker int
}

// equipmentSlots 是 overlay-25 `0C21h..0D6Eh` 沿物品串列認回來的槽與手數：
// 類別 0..8 覆寫 `+CCh + 類別 × 4`（最後一件贏），類別 9 先填 `+F0h`、再填 `+F4h`
// （兩格都有了就不動），型別 49h 覆寫 `+F8h`、1Ch 覆寫 `+FCh`；`+100h` 累加手數。
type equipmentSlots struct {
	slot    [equipmentSlotCategories]int
	ring    [2]int
	arrow   int
	quarrel int
	hands   int
}

func readiedSlots(inventory [][]byte, types *ItemTypeTable) (equipmentSlots, error) {
	slots := equipmentSlots{arrow: -1, quarrel: -1, ring: [2]int{-1, -1}}
	for index := range slots.slot {
		slots.slot[index] = -1
	}
	for index, raw := range inventory {
		if len(raw) <= ItemReadiedOffset || raw[ItemReadiedOffset] == 0 {
			continue
		}
		entry, err := types.Entry(raw[ItemTypeOffset])
		if err != nil {
			return slots, err
		}
		switch category := entry.Category(); {
		case category < equipmentSlotCategories:
			slots.slot[category] = index
		case category == equipmentRingCategory:
			if slots.ring[0] < 0 {
				slots.ring[0] = index
			} else if slots.ring[1] < 0 {
				slots.ring[1] = index
			}
		}
		switch raw[ItemTypeOffset] {
		case ItemTypeArrow:
			slots.arrow = index
		case ItemTypeQuarrel:
			slots.quarrel = index
		}
		slots.hands = int(uint8(slots.hands + int(entry.Raw[ItemTypeHandsOffset])))
	}
	return slots, nil
}

// ReadyItem 是 overlay-19 entry 7（`14CAh`）：穿戴中的就卸下（被詛咒的卸不下），
// 沒穿的照下面的順序判，後面成立的蓋掉前面的（`[bp-2]` 是同一格）：
//
//	1571  手數 + 角色 +100h > 2                         → 手滿了（3）
//	159C  類別 0..8 而那一格已有東西；類別 9 而 +F4h 已有  → 已經在用（2）
//	15D6  型別 49h 而 +F8h 已有；型別 1Ch 而 +FCh 已有     → 已經在用（2），擋的是那一格
//	1624  型別表 +0Dh & 角色 +0B0h == 0                   → 職業不對（1）
//	1642  都沒有                                           → +34h = 1
//
// 成立時直接改 inventory[index] 的 `+34h`。`+3Eh` 大於 7Fh 的物品原版還會交給
// overlay-24 entry 1 掛上或摘掉穿戴效果；remake 沒有那一層（探索時的裝備頁也沒有）。
func ReadyItem(inventory [][]byte, index int, types *ItemTypeTable, classMask uint8) (ReadyResult, error) {
	raw := inventory[index]
	if raw[ItemReadiedOffset] != 0 {
		if raw[ItemCursedOffset] != 0 {
			return ReadyResult{Outcome: ReadyCursed, Blocker: -1}, nil
		}
		raw[ItemReadiedOffset] = 0
		return ReadyResult{Outcome: UnreadyDone, Blocker: -1}, nil
	}
	entry, err := types.Entry(raw[ItemTypeOffset])
	if err != nil {
		return ReadyResult{}, err
	}
	slots, err := readiedSlots(inventory, types)
	if err != nil {
		return ReadyResult{}, err
	}
	result := ReadyResult{Outcome: ReadyDone, Blocker: -1}
	if int(entry.Raw[ItemTypeHandsOffset])+slots.hands > readyHandLimit {
		result.Outcome = ReadyHandsFull
	}
	switch category := entry.Category(); {
	case category < equipmentSlotCategories:
		if slots.slot[category] >= 0 {
			result = ReadyResult{Outcome: ReadyAlreadyUsing, Blocker: slots.slot[category]}
		}
	case category == equipmentRingCategory:
		// `15C0h` 問的是 +F4h，印的卻是 `+CCh + 9 × 4` 也就是 +F0h 那一件。
		if slots.ring[1] >= 0 {
			result = ReadyResult{Outcome: ReadyAlreadyUsing, Blocker: slots.ring[0]}
		}
	}
	if raw[ItemTypeOffset] == ItemTypeArrow && slots.arrow >= 0 {
		result = ReadyResult{Outcome: ReadyAlreadyUsing, Blocker: slots.arrow}
	}
	if raw[ItemTypeOffset] == ItemTypeQuarrel && slots.quarrel >= 0 {
		result = ReadyResult{Outcome: ReadyAlreadyUsing, Blocker: slots.quarrel}
	}
	if entry.Raw[ItemTypeClassMaskOffset]&classMask == 0 {
		result = ReadyResult{Outcome: ReadyWrongClass, Blocker: -1}
	}
	if result.Outcome == ReadyDone {
		raw[ItemReadiedOffset] = 1
	}
	return result, nil
}

// HalveItem 是 overlay-19 entry 14（`17F2h`）：一半 = 數量 div 2；為 0 印
// "Can't halve that"（`17E1h`）。否則複製一件（`1841h` 整筆 3Fh bytes）、新的那件
// 數量是一半而且不穿戴，接在原件後面；原件留下 數量 − 一半。
func HalveItem(raw []byte) ([]byte, bool) {
	if len(raw) <= ItemCountOffset {
		return nil, false
	}
	half := raw[ItemCountOffset] / 2
	if half == 0 {
		return nil, false
	}
	split := append([]byte(nil), raw...)
	split[ItemCountOffset] = half
	split[ItemReadiedOffset] = 0
	raw[ItemCountOffset] -= half
	return split, true
}

// JoinItems 是 overlay-19 entry 15（`18A2h`）：沿整條物品串列，把跟選中那件
// 「同一種」的疊進來，回傳要拿掉的幾件（切片索引，由小到大）。
//
// 同一種＝數量 > 0，而且 `+2Fh`、`+30h`、`+31h`、`+2Eh`、`+32h`、`+33h`、`+36h` 與
// `+37h`（word）都相同，另外 `+3Ch` < 2。`19A2h`／`19C2h`／`19D5h` 三處把那一件的
// `+3Ch`／`+3Dh`／`+3Eh` 跟**自己**比（兩個運算元都是 `[bp-4]`），恆成立——所以
// 充能與法術欄不必相同，照位元組做。
//
// 累加超過 255 時（`1A18h`）：目前收件的那件設成 255，這一件減去收件那件還放得下
// 的量，改由這一件接著收。最後把累加值寫回收件的那一件（`1A63h`）。
func JoinItems(inventory [][]byte, index int) []int {
	target := index
	total := int(inventory[index][ItemCountOffset])
	// 比對的欄位都跟收件的那件比；收件換人時兩者相同（換的條件就是全部相同），
	// 所以固定跟選中那件比是同一件事。
	selected := inventory[index]
	var removed []int
	for other, raw := range inventory {
		// `18E8h` 跳過的是**目前收件的那一件**（`[bp-0Ch]`），不是一開始選中的那件。
		if other == target || len(raw) <= ItemEffectOffset || raw[ItemCountOffset] == 0 {
			continue
		}
		same := true
		for _, offset := range []int{ItemNameWordOffset, ItemNameWordOffset + 1, ItemNameWordOffset + 2,
			ItemTypeOffset, ItemPlusOffset, ItemSaveBonusOffset, ItemCursedOffset,
			ItemWeightOffset, ItemWeightOffset + 1} {
			if raw[offset] != selected[offset] {
				same = false
				break
			}
		}
		if !same || raw[AIItemChargesOffset] >= 2 {
			continue
		}
		if total+int(raw[ItemCountOffset]) <= joinCountLimit {
			total += int(raw[ItemCountOffset])
			removed = append(removed, other)
			continue
		}
		inventory[target][ItemCountOffset] = joinCountLimit
		raw[ItemCountOffset] -= uint8(joinCountLimit - total)
		total = int(raw[ItemCountOffset])
		target = other
	}
	inventory[target][ItemCountOffset] = uint8(total)
	return removed
}

// ScrollReadable 是 overlay-22 entry 12（`0441h`）開頭：身上有閱讀魔法的效果，或
// 牧師等級（記錄 `+96h`）大於 0 而且這是牧師卷軸，就把 `+35h` 清成 0（揭開藏字）。
// `+35h` 仍非 0 就一條也列不出來。回傳 revealed 代表這一次把 `+35h` 清掉了。
func ScrollReadable(raw []byte, category uint8, readMagic bool, clericLevel uint8) (readable, revealed bool) {
	if len(raw) <= scrollSpellLast {
		return false, false
	}
	if readMagic || (int8(clericLevel) > 0 && category == ScrollCleric) {
		revealed = raw[ItemHiddenNameOffset] != 0
		raw[ItemHiddenNameOffset] = 0
	}
	return raw[ItemHiddenNameOffset] == 0, revealed
}

// ScrollSpells 是 overlay-22 entry 12 用模式 0 列出來的那幾行（`04B8h`：值 > 0，
// 位元 7 立著的也列，前面加 " *"），回傳法術編號（& 7Fh）與是不是有人正要抄。
func ScrollSpells(raw []byte) (spells []uint8, scribing []bool) {
	if len(raw) <= scrollSpellLast {
		return nil, nil
	}
	for offset := scrollSpellFirst; offset <= scrollSpellLast; offset++ {
		if raw[offset] == 0 {
			continue
		}
		spells = append(spells, raw[offset]&0x7f)
		scribing = append(scribing, raw[offset] > 0x7f)
	}
	return spells, scribing
}

// EraseScrollSpell 是 overlay-22 entry 7（`3243h`）：三行裡**最後一個** & 7Fh 等於這條
// 法術的清成 0、名稱字詞 `+30h` 減一（"With 3 Spells" → "With 2 Spells"）；減到 D2h
// 以下就回 true，由呼叫端把卷軸拿掉（overlay-25 entry 17）。一行都對不上就不動。
func EraseScrollSpell(raw []byte, spell uint8) bool {
	if len(raw) <= scrollSpellLast {
		return false
	}
	found := 0
	for offset := scrollSpellFirst; offset <= scrollSpellLast; offset++ {
		if raw[offset]&0x7f == spell&0x7f {
			found = offset
		}
	}
	if found == 0 {
		return false
	}
	raw[found] = 0
	raw[ItemNameWordOffset+1]--
	return raw[ItemNameWordOffset+1] < scrollOneSpellWord
}
