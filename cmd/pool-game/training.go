package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 訓練所（spec 097）。原版把它掛在隊伍管理畫面的 `T` 指令上
//（overlay-16 entry 1 的 `036Fh` 比 `T`，過了才呼叫 entry 8 的訓練常式），
// 選單字串是 `DS:06ABh` 的 "Train Character"。
//
// **`T` 什麼時候出現在那個畫面上已經讀出來了**（spec 008）：`DS:06D4h`
// 由 overlay-16 `01BFh` 的兩道閘門決定，第一道是這一區的訓練所遮罩
// `[4937h]+550h`（＝ ECL 位址 `6DA8h`）。見 training_gate.go。
//
// **這一家收哪幾類仍然不限制**：原版拿那個遮罩逐人 `and` 職業分類
//（`2C95h`／`2CC3h`／`2CF6h`），remake 目前只看它非不非零，等同於四道門
// 都走得進去的訓練所。

// memberClassLevels 取一個成員的八個職業等級。
//
// 沒有存過的（舊存檔、剛建好的角色）照組成職業各給第 1 級——建角本來就是
// 這樣寫下去的（spec 072 的 overlay-16 建角常式）。
func memberClassLevels(member poolsave.Character) [gamepack.ClassThac0ClassCount]uint8 {
	levels, err := partyClassLevels(member)
	if err != nil {
		// 職業認不出來的（NPC 帶自己的記錄）就是全零：升不動，也不會亂升。
		return [gamepack.ClassThac0ClassCount]uint8{}
	}
	return levels
}

// memberClassCode 取複合職業碼 `+2Fh`。NPC 帶自己的記錄。
func memberClassCode(member poolsave.Character) (uint8, bool) {
	if code, ok := creation.ClassDOSCode(member.ClassID); ok {
		return code, true
	}
	if len(member.Record) > gamepack.ClassCodeOffset {
		return member.Record[gamepack.ClassCodeOffset], true
	}
	return 0, false
}

// trainMember 對隊伍裡的一個人做一次訓練，回傳給玩家看的一行字。
func (a *app) trainMember(index int) (string, error) {
	if index < 0 || index >= len(a.state.Party) {
		return "", fmt.Errorf("Pool training has no party member %d", index)
	}
	member := &a.state.Party[index]
	code, ok := memberClassCode(*member)
	if !ok {
		return "", fmt.Errorf("Pool training cannot resolve the class of %q", member.Name)
	}
	levels := memberClassLevels(*member)
	before := levels
	outcome := a.levelUpTables.Train(levels, member.Experience, a.experienceTable,
		member.Abilities[gamepack.AbilityConstitution], code, 0, a.roller)
	name := strings.TrimSpace(member.Name)
	if !outcome.Trained {
		return fmt.Sprintf("%s%s", name, a.text(msgTrainNotYet)), nil
	}
	member.ClassLevels = outcome.Levels[:]
	member.Experience = outcome.Experience
	member.MaxHP, member.CurrentHP = gamepack.ApplyLevelUpHitPoints(
		member.MaxHP, member.CurrentHP, outcome.HitPointGain)
	member.RawHP += outcome.PlainHitPointGain
	a.refreshSpellbookAfterTraining(member, before)
	syncTrainedLibraryCharacter(&a.state, *member)
	return fmt.Sprintf("%s%s%+d", name, a.text(msgTrainGained), outcome.HitPointGain), nil
}

// syncTrainedLibraryCharacter 把升完的那一份寫回角色庫，
// 不然回主選單再看還是舊的。
func syncTrainedLibraryCharacter(state *poolsave.State, character poolsave.Character) {
	for index := range state.CharacterLibrary {
		if state.CharacterLibrary[index].Name == character.Name {
			state.CharacterLibrary[index] = character
			return
		}
	}
}

// fillThiefSkills 照 overlay-23 entry 4 算出八格賊技能（spec 095）。
//
// 只有賊等級大於 0 才寫。非賊的八格本來就是 0，但這裡刻意不寫進去——
// `internal/character` 匯出 `.CHA` 時把「空的」與「全部 0」分開看：空的
// 代表 remake 沒有這份資料，記錄裡原本的位元組不動。NPC 帶著自己的記錄
// 進來，硬塞八個 0 會把他們原本的技能抹掉。
func (a *app) fillThiefSkills(character *poolsave.Character) error {
	thiefLevel := int(memberClassLevels(*character)[gamepack.ThiefLevelIndex])
	if thiefLevel <= 0 {
		return nil
	}
	race, ok := findRace(character.RaceID)
	if !ok {
		return fmt.Errorf("Pool race %q is not in the catalog", character.RaceID)
	}
	skills, err := a.thiefSkillTables.Build(thiefLevel, int(race.DOSCode), gamepack.ThiefSkillBuildDexterity)
	if err != nil {
		return err
	}
	character.ThiefSkills = append([]uint8(nil), skills[:]...)
	return nil
}
