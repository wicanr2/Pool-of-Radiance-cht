package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 效果串列在原版只有一份：掛在角色記錄的 `+7Fh`，戰場上與地圖上讀的是同一
// 條（spec 069）。remake 分成兩個型別存——存檔用 `poolsave.EffectNode`、
// 戰鬥用 `gamepack.EffectList`——所以進出戰鬥時要把同一條串列搬過去再搬回來。
//
// 不搬的症狀不是「戰鬥中沒效果」，而是**進戰鬥的那一刻身上的祝福、詛咒與
// 毒全部消失，打完又原封不動地回來**：戰鬥裡下的減益也一樣，走出戰場就沒了。

// combatEffects 把存檔的節點換成戰鬥用的串列。順序就是串列順序，不動內容。
func combatEffects(nodes []poolsave.EffectNode) gamepack.EffectList {
	if len(nodes) == 0 {
		return nil
	}
	list := make(gamepack.EffectList, 0, len(nodes))
	for _, node := range nodes {
		list = append(list, gamepack.EffectNode{Code: node.Code, Payload: node.Payload})
	}
	return list
}

// storedEffects 是反向。`SavedNext` 不寫回去——存進檔案的遠指標是上次執行時
// 的位址，重新載入沒有意義（spec 069）。
func storedEffects(list gamepack.EffectList) []poolsave.EffectNode {
	if len(list) == 0 {
		return nil
	}
	nodes := make([]poolsave.EffectNode, 0, len(list))
	for _, node := range list {
		nodes = append(nodes, poolsave.EffectNode{Code: node.Code, Payload: node.Payload})
	}
	return nodes
}

// storeCombatEffects 在戰鬥收場時把每個隊員身上的串列寫回隊伍。
//
// 勝敗都要寫回：原版的串列本來就長在角色記錄上，沒有「戰鬥結束就清掉」這件
// 事——效果只有自己到期或被解除才會消失。
func (a *app) storeCombatEffects(state *tacticalState) {
	if state == nil {
		return
	}
	for index, party := range state.PartySlot {
		if party < 0 || party >= len(a.state.Party) {
			continue
		}
		if index >= len(state.Effects) {
			continue
		}
		a.state.Party[party].Effects = storedEffects(state.Effects[index])
		// 角色庫存的是同一個人。原版沒有這個問題——`.CHA` 與 `.SPC` 就是那
		// 一份檔案，離隊再入隊讀的還是它；remake 存兩份，不同步的話效果會
		// 在離隊那一刻消失。
		syncTrainedLibraryCharacter(&a.state, a.state.Party[party])
	}
}

// advancePartyEffects 把整隊的效果往前推 minutes 分鐘（overlay-20 offset `0`）。
// 走一步一分、休息一刻五分，兩個呼叫點推的是同一支。
//
// **到期的節點目前只是摘掉**：原版在 `01B8h` 會呼叫 `0100:002A` 跑收尾
//（spec 069），那一支還沒讀，所以這裡不猜它做什麼——摘掉是收尾一定包含的
// 部分，多做的部分寧可缺著也不要發明。
func (a *app) advancePartyEffects(minutes int) {
	if minutes <= 0 {
		return
	}
	for index := range a.state.Party {
		member := &a.state.Party[index]
		if len(member.Effects) == 0 {
			continue
		}
		list, _ := combatEffects(member.Effects).AdvanceEffects(minutes)
		member.Effects = storedEffects(list)
		syncTrainedLibraryCharacter(&a.state, *member)
	}
}
