package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 法術書（spec 110）：記憶畫面只列這個人書上有的。
//
// 沒有它的話，一級法師記得起火球術——而畫面上那看起來像「法術系統做好了」，
// 不像少了一道閘門。

// memberSpellbook 取一個人的法術書，必要時就地補算。
//
// 三種來源，依序：存檔裡已經算過的、NPC 自己帶的 285-byte 記錄、
// 依現在的職業等級重算。第三種是舊存檔的補算路徑——牧師照
// 「那一級有格子就全會」重建得回來（原版就是這條規則），法師則是起手四條
// 加上每個等級一條的額度。
func (a *app) memberSpellbook(member poolsave.Character) ([]uint8, int) {
	if len(member.Spellbook) != 0 {
		return member.Spellbook, member.SpellsToLearn
	}
	if len(member.Record) > gamepack.SpellbookRecordBase+gamepack.SpellbookLastSpell {
		if known, err := gamepack.RecordSpellbook(member.Record); err == nil && len(known) != 0 {
			return known, member.SpellsToLearn
		}
	}
	levels := memberClassLevels(member)
	known := gamepack.NewCharacterSpellbook(levels, member.Abilities[gamepack.AbilityWisdom],
		a.spellSlotTables, a.spellParameters)
	credits := member.SpellsToLearn
	if mage := int(levels[gamepack.ClassSlotMagicUser]); mage > 1 {
		credits += mage - 1
	}
	return known, credits
}

// ensureSpellbook 把補算的結果寫回隊伍與角色庫，只做一次。
func (a *app) ensureSpellbook(index int) {
	if index < 0 || index >= len(a.state.Party) {
		return
	}
	member := &a.state.Party[index]
	if len(member.Spellbook) != 0 {
		return
	}
	known, credits := a.memberSpellbook(*member)
	if len(known) == 0 {
		return
	}
	member.Spellbook, member.SpellsToLearn = known, credits
	syncTrainedLibraryCharacter(&a.state, *member)
}

// refreshSpellbookAfterTraining 是昇級之後要跑的兩件事。
//
//   - 牧師：overlay-23 `0132h` 重掃一次「那一級有格子就全會」。
//   - 法師：等級上升就多一條可以學的額度（overlay-16 `2F79h` 在
//     `+9Bh` 變大時挑一條寫進書裡）。挑的那一支是原版的選單常式
//     （`00C9:005Ch`），還沒讀出來，所以 remake 讓玩家自己挑。
func (a *app) refreshSpellbookAfterTraining(member *poolsave.Character, before [gamepack.ClassThac0ClassCount]uint8) {
	after := memberClassLevels(*member)
	if len(member.Spellbook) == 0 {
		member.Spellbook, member.SpellsToLearn = a.memberSpellbook(*member)
	}
	member.Spellbook = gamepack.RefreshClericSpellbook(member.Spellbook, after,
		member.Abilities[gamepack.AbilityWisdom], a.spellSlotTables, a.spellParameters)
	if gained := int(after[gamepack.ClassSlotMagicUser]) - int(before[gamepack.ClassSlotMagicUser]); gained > 0 {
		member.SpellsToLearn += gained
	}
}
