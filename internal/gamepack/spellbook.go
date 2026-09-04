package gamepack

import "fmt"

// 法術書：角色記錄裡「這個人會不會這一條」的陣列（spec 110）。
//
// 位置是 `+32h + 1-based 法術編號`，一格一個 byte，1 是會。`+32h` 本身是
// 最大生命值（與 `MonsterRecord.MaxHitPoints` 同一格），編號從 1 起算，
// 所以第一條落在 `+33h`、最後一條 `38h`（56，Restoration）落在 `+6Ah`。
//
// 原版一律寫成「記錄基底加編號再取 `+32h`」——`add di, ax` 之後
// `es:[di+32h]`——所以直接用 `+33h` 去搜位元組會落空，要搜 `+32h`。
const (
	// SpellbookRecordBase 是索引的基底：`base + 編號`。
	SpellbookRecordBase = 0x32
	// SpellbookFirstSpell 與 SpellbookLastSpell 是編號的範圍。上限 56 由
	// 三處迴圈的 `cmp ..., 38h` 釘住（建角、昇級、卷軸）。
	SpellbookFirstSpell = 1
	SpellbookLastSpell  = 0x38
	// SpellbookKnown 是「會」的值。20 個預設人物檔裡非零的格子全部是 1。
	SpellbookKnown = 1
)

// newMagicUserSpellbook 是新法師的起手四條，寫死在建角裡（overlay-16
// `1C7Fh`..`1C97h` 連著四個 `mov byte es:[di+3Dh/44h/45h/47h], 1`）：
// 11 偵測魔法、18 閱讀魔法、19 護盾術、21 催眠術。
var newMagicUserSpellbook = []uint8{11, 18, 19, 21}

// RecordSpellbook 讀出角色記錄裡會的法術編號，由小到大。
func RecordSpellbook(record []byte) ([]uint8, error) {
	last := SpellbookRecordBase + SpellbookLastSpell
	if len(record) <= last {
		return nil, fmt.Errorf("Pool character record is %d bytes, the spellbook needs %d",
			len(record), last+1)
	}
	var known []uint8
	for id := SpellbookFirstSpell; id <= SpellbookLastSpell; id++ {
		if record[SpellbookRecordBase+id] != 0 {
			known = append(known, uint8(id))
		}
	}
	return known, nil
}

// SpellbookKnows 說這一本書裡有沒有這一條。
func SpellbookKnows(known []uint8, id uint8) bool {
	for _, entry := range known {
		if entry == id {
			return true
		}
	}
	return false
}

// AddToSpellbook 把一條加進書裡，維持由小到大且不重複。
func AddToSpellbook(known []uint8, id uint8) []uint8 {
	if id < SpellbookFirstSpell || id > SpellbookLastSpell || SpellbookKnows(known, id) {
		return known
	}
	grown := append(known, id)
	for index := len(grown) - 1; index > 0 && grown[index-1] > grown[index]; index-- {
		grown[index-1], grown[index] = grown[index], grown[index-1]
	}
	return grown
}

// NewCharacterSpellbook 是建角當下的法術書。
//
// 兩條規則都在 overlay-16 的建角迴圈裡，依職業槽分岔（槽 0 牧師、槽 5 法師，
// 與 `ClassSlot*` 相同）：
//
//   - 牧師：那一級有格子就會那一級的**全部**神術。原版在建角只給第 1 級的
//     （`cmp es:[di+1], 1`），但昇級時 overlay-23 `0132h` 會用「有格子就學」
//     重掃一次，所以兩者合起來就是這一條——預設人物 ALFRED（牧師 6）
//     書裡正好是第 1..3 級的全部 24 條。
//   - 法師：四條寫死的（偵測魔法、閱讀魔法、護盾術、催眠術），
//     其餘要昇級或抄卷軸才學得到。
func NewCharacterSpellbook(levels [8]uint8, wisdom int, tables SpellSlotTables,
	parameters []SpellParameters) []uint8 {
	maxima := tables.SpellSlotMaxima(int(levels[ClassSlotCleric]),
		int(levels[ClassSlotMagicUser]), wisdom)
	var known []uint8
	if levels[ClassSlotCleric] > 0 {
		known = clericSpellbook(known, maxima, parameters)
	}
	if levels[ClassSlotMagicUser] > 0 {
		for _, id := range newMagicUserSpellbook {
			known = AddToSpellbook(known, id)
		}
	}
	return known
}

// RefreshClericSpellbook 是 overlay-23 `0132h`：重算牧師的可記憶數之後，
// 把「那一級有格子」的神術全部補進書裡。昇級之後要跑一次。
//
// 只補不刪——原版那一支也只寫 1。
func RefreshClericSpellbook(known []uint8, levels [8]uint8, wisdom int,
	tables SpellSlotTables, parameters []SpellParameters) []uint8 {
	if levels[ClassSlotCleric] == 0 {
		return known
	}
	maxima := tables.SpellSlotMaxima(int(levels[ClassSlotCleric]),
		int(levels[ClassSlotMagicUser]), wisdom)
	return clericSpellbook(known, maxima, parameters)
}

func clericSpellbook(known []uint8, maxima SpellSlotCounts, parameters []SpellParameters) []uint8 {
	for id := SpellbookFirstSpell; id <= SpellbookLastSpell; id++ {
		if id >= len(parameters) {
			break
		}
		entry := parameters[id]
		if int(entry.Source()) != SpellSlotGroupCleric {
			continue
		}
		level := entry.Level()
		if level < 1 || level > SpellSlotLevels {
			continue
		}
		if maxima[SpellSlotGroupCleric][level-1] <= 0 {
			continue
		}
		known = AddToSpellbook(known, uint8(id))
	}
	return known
}
