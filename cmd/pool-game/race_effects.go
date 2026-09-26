package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 種族效果（spec 145，issue #88）：原版建角在 overlay-16 `078Ah..0905h` 依記錄 `+2Eh`
// 掛上 `gamepack.RaceEffectCodes` 的節點，之後它們就一直留在 `+7Fh` 的串列上。

// raceEffectNodes 是 raceID 建角時要掛的節點（存檔那一側的型別）。
func raceEffectNodes(raceID string) []poolsave.EffectNode {
	race, ok := creation.RaceDOSCode(raceID)
	if !ok {
		return nil
	}
	return storedEffects(gamepack.RaceEffects(race))
}

// withRaceEffects 把 member 缺的種族節點補到串列最前面，回傳是否改過。
//
// 原版在建角那一刻就掛，所以它們一定是串列上最早的節點；補在前面才與原版的順序相同
// （效果系統對每個碼只問最早的那一個，spec 112）。已經有的碼不重複掛。
func withRaceEffects(member *poolsave.Character) bool {
	if member.NPC {
		return false
	}
	want := raceEffectNodes(member.RaceID)
	missing := make([]poolsave.EffectNode, 0, len(want))
	for _, node := range want {
		present := false
		for _, have := range member.Effects {
			if have.Code == node.Code {
				present = true
				break
			}
		}
		if !present {
			missing = append(missing, node)
		}
	}
	if len(missing) == 0 {
		return false
	}
	member.Effects = append(missing, member.Effects...)
	return true
}

// migrateRaceEffects 補上 #88 之前的 remake 建角沒掛的種族效果。隊伍與人物名單都補：
// 名單裡的人之後加入隊伍時帶的是名單那一份。NPC 的記錄來自怪物檔，不動。
func (a *app) migrateRaceEffects() {
	for index := range a.state.Party {
		withRaceEffects(&a.state.Party[index])
	}
	for index := range a.state.CharacterLibrary {
		withRaceEffects(&a.state.CharacterLibrary[index])
	}
}
