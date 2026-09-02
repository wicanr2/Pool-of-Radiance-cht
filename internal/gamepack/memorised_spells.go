package gamepack

import "fmt"

// 角色記錄裡的記憶法術陣列。285-byte 記錄的 `+1Fh` 起 13 bytes，每個 byte 一格：
// 0 是空格，其餘是**1-based** 的法術編號。
//
// 1-based 這件事由 overlay-22 的取名路徑證實：`id and 7Fh` 之後乘 41（`29h`），
// 再加基底 `2883h`；而名稱表的第一筆 `Bless` 在 `28ACh`，正好是 `2883h + 41`。
// 因此編號 1 是 Bless，也就是名稱表索引 0。
//
// `and al, 7Fh` 同時說明**第 7 位是旗標，不是編號的一部分**。旗標的語意還沒
// 閉合：預設人物檔（都處於休息完的狀態）裡沒有一個 byte 設了它。
const (
	// MemorisedSpellOffset 是陣列在角色記錄裡的起點。
	MemorisedSpellOffset = 0x1f
	// MemorisedSpellSlots 是陣列的格數。
	MemorisedSpellSlots = 13
	// memorisedSpellIDMask 取出編號，濾掉第 7 位的旗標。
	memorisedSpellIDMask = 0x7f
	// MemorisedSpellFlag 是第 7 位。原版取名前會把它濾掉。
	MemorisedSpellFlag = 0x80
)

// MemorisedSpell 是一格記憶法術。
type MemorisedSpell struct {
	// Slot 是它在陣列裡的位置。
	Slot int
	// ID 是 1-based 的法術編號，也就是名稱表索引加一。
	ID uint8
	// Flagged 是第 7 位；語意未閉合，照實傳出來。
	Flagged bool
}

// MemorisedSpells 取出角色記錄裡已記憶的法術，空格不列入。
func MemorisedSpells(record []byte) ([]MemorisedSpell, error) {
	if len(record) < MemorisedSpellOffset+MemorisedSpellSlots {
		return nil, fmt.Errorf("Pool character record is %d bytes, the spell array needs %d",
			len(record), MemorisedSpellOffset+MemorisedSpellSlots)
	}
	var out []MemorisedSpell
	for slot := 0; slot < MemorisedSpellSlots; slot++ {
		value := record[MemorisedSpellOffset+slot]
		id := value & memorisedSpellIDMask
		if id == 0 {
			continue
		}
		if int(id) > SpellNameCount {
			return nil, fmt.Errorf("Pool memorised spell %d in slot %d is outside 1..%d",
				id, slot, SpellNameCount)
		}
		out = append(out, MemorisedSpell{Slot: slot, ID: id, Flagged: value&MemorisedSpellFlag != 0})
	}
	return out, nil
}

// SpellByID 依 1-based 編號取一條法術，也就是遊戲自己用的編號。
func (c *SpellCatalogue) SpellByID(id uint8) (Spell, error) {
	if id == 0 {
		return Spell{}, fmt.Errorf("Pool spell id 0 is the empty slot, not a spell")
	}
	return c.Spell(int(id) - 1)
}
