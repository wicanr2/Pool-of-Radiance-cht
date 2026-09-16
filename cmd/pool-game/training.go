package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 訓練所（spec 097）。原版把它掛在隊伍管理畫面的 `T` 指令上
//（overlay-16 entry 1 的 `036Fh` 比 `T`，過了才呼叫 entry 8 的訓練常式），
// 選單字串是 `DS:06ABh` 的 "Train Character"。
//
// **`T` 什麼時候出現在那個畫面上已經讀出來了**（spec 008）：`DS:06D4h`
// 由 overlay-16 `01BFh` 的兩道閘門決定，第一道是這一區的訓練所遮罩
// `[4937h]+550h`（＝ ECL 位址 `6DA8h`）。見 training_gate.go。
//
// 訓練常式本體是 overlay-16 `2997h`（#29，spec 097〈收費與門〉），順序是：
//
//  1. `+10Ch != 0`（不是清醒的）→ "we only train conscious people"（`29A2h`）。
//  2. overlay-19 entry 11 算五種硬幣的金幣等值，不到 1000 → "Training costs 1000 gp."
//     （`29D3h..2A02h`）。
//  3. 職業分類 `and` 這一家的遮罩 `[4937h]+550h`（ECL `6DA8h`）為 0 →
//     "We don't train that class here"（`2C8Ah..2CB3h`）；夠經驗的那些再 `and`
//     一次為 0 → "Not Enough Experience"（`2CB8h..2CE1h`）。
//  4. "Do you wish to train?" 要按 Y（`2E5Ch..2E7Bh`）。
//  5. 升級、"Congratulations"，最後 `42C1h(記錄, 1000)` 一種一種硬幣扣、白金往下找零
//     （`2E9Bh..2EABh`；`treasure.PayInCoins`）。
//
// `466Eh`（STING 除錯碼）非零時 1、2、3 的門與 5 的扣款都略過。

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

// trainingFee 是說明書 p.9 與 `29DFh` 的 1000 金。
const trainingFee = 1000

// trainingHallMask 是這一家收哪幾類（`[4937h]+550h`，ECL `6DA8h`）；輸入過除錯碼
// 就是 0，Train 把 0 當成不限。
func (a *app) trainingHallMask() uint8 {
	if a.stingUnlocked || a.eventMachine == nil {
		return 0
	}
	return uint8(a.eventMachine.Memory[trainingMaskAddress])
}

// trainMember 是按 T 那一下：走 `2997h` 的前三道門，過了就問 "Do you wish to train?"，
// 真正升級與扣款在 confirmTraining。回傳給玩家看的一行字。
func (a *app) trainMember(index int) (string, error) {
	if index < 0 || index >= len(a.state.Party) {
		return "", fmt.Errorf("Pool training has no party member %d", index)
	}
	member := &a.state.Party[index]
	code, ok := memberClassCode(*member)
	if !ok {
		return "", fmt.Errorf("Pool training cannot resolve the class of %q", member.Name)
	}
	name := strings.TrimSpace(member.Name)
	if member.Status != 0 && !a.stingUnlocked {
		return a.text(msgTrainNotConscious), nil
	}
	if pooltreasure.GoldEquivalent(member.Money) < trainingFee && !a.stingUnlocked {
		return a.text(msgTrainCosts), nil
	}
	levels := memberClassLevels(*member)
	hall := a.trainingHallMask()
	if hall != 0 && a.levelUpTables.ClassCategoryMask(levels)&hall == 0 {
		return a.text(msgTrainWrongClass), nil
	}
	outcome := a.levelUpTables.Train(levels, member.Experience, a.experienceTable,
		member.Abilities[gamepack.AbilityConstitution], code, hall, a.roller)
	if !outcome.Trained {
		return fmt.Sprintf("%s%s", name, a.text(msgTrainNotYet)), nil
	}
	a.trainPending, a.trainMemberIndex = true, index
	return fmt.Sprintf("%s%s", name, a.text(msgTrainAsk)), nil
}

// confirmTraining 是 "Do you wish to train?" 答 Y 之後的那一段（`2E80h..`）：升級、
// 收 1000 金。
func (a *app) confirmTraining() (string, error) {
	index := a.trainMemberIndex
	a.trainPending = false
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
		member.Abilities[gamepack.AbilityConstitution], code, a.trainingHallMask(), a.roller)
	name := strings.TrimSpace(member.Name)
	if !outcome.Trained {
		return fmt.Sprintf("%s%s", name, a.text(msgTrainNotYet)), nil
	}
	if !a.stingUnlocked {
		if err := pooltreasure.PayInCoins(&member.Money, trainingFee); err != nil {
			return "", err
		}
	}
	trained := *member
	trained.ClassLevels = append([]uint8(nil), outcome.Levels[:]...)
	if err := a.fillThiefSkillsWithDexterity(&trained,
		trained.Abilities[gamepack.AbilityDexterity]); err != nil {
		return "", err
	}
	member.ClassLevels = outcome.Levels[:]
	member.ThiefSkills = trained.ThiefSkills
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
	return a.fillThiefSkillsWithDexterity(character, gamepack.ThiefSkillBuildDexterity)
}

// fillThiefSkillsWithDexterity 重算指定等級的八格技能。建角傳 0；訓練時
// 原版 overlay-23 entry 1 直接再呼叫 entry 4，所以傳記錄裡的真實 DEX。
func (a *app) fillThiefSkillsWithDexterity(character *poolsave.Character, dexterity int) error {
	thiefLevel := int(memberClassLevels(*character)[gamepack.ThiefLevelIndex])
	if thiefLevel <= 0 {
		return nil
	}
	race, ok := findRace(character.RaceID)
	if !ok {
		return fmt.Errorf("Pool race %q is not in the catalog", character.RaceID)
	}
	skills, err := a.thiefSkillTables.Build(thiefLevel, int(race.DOSCode), dexterity)
	if err != nil {
		return err
	}
	character.ThiefSkills = append([]uint8(nil), skills[:]...)
	return nil
}
