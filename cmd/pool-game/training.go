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
// **這一家收哪幾類還沒讀出來**：原版每個指令各有一個啟用旗標（`T` 的在
// `DS:06D4h`），旗標從哪裡設還沒讀，所以這裡不限制職業類別，等同於一家
// 每一類都收的訓練所。

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
