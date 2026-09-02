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

	// SpellSearchOpcode 是 `3Bh SPELL`（spec 094）：找隊上誰記了某個法術。
	SpellSearchOpcode = 0x3b
	// SpellSearchOperands 是它吃幾個運算元。
	SpellSearchOperands = 3
	// SpellSearchBase 是原版掃描的基底（`2FC9h` 的 `es:[di+17h]`）。
	// 槽位編號 i 對到記錄 `+17h + i`，所以記憶陣列的第 k 格是 i = k + 8。
	SpellSearchBase = 0x17
	// SpellSearchLastSlot 是掃到第幾格為止（`2FD8h` 比的 51h）。
	SpellSearchLastSlot = 0x51
	// SpellSearchNotFound 是沒找到時寫回去的槽位編號（`2FE3h` 的 FFh）。
	SpellSearchNotFound = 0xff
)

// SpellSearchSlot 是記憶陣列第 k 格對應的原版槽位編號。
func SpellSearchSlot(index int) int { return index + MemorisedSpellOffset - SpellSearchBase }

// SearchMemorisedSpell 在一份記憶陣列裡找某個法術編號，回傳原版的槽位編號；
// 沒找到回 SpellSearchNotFound。
//
// 比對前會濾掉第 7 位的旗標，與原版取名時的 `and al, 7Fh` 同一套。
func SearchMemorisedSpell(memorised []uint8, wanted uint8) int {
	for index, value := range memorised {
		if index >= MemorisedSpellSlots {
			break
		}
		if value&memorisedSpellIDMask == wanted {
			return SpellSearchSlot(index)
		}
	}
	return SpellSearchNotFound
}

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
