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
// 到期的節點先跑收尾再摘掉，與原版的順序相同（`01B8h` 呼叫的
// `0100h:002Ah` 就是 overlay-24 entry 2，spec 112）。
func (a *app) advancePartyEffects(minutes int) {
	if minutes <= 0 {
		return
	}
	for index := range a.state.Party {
		member := &a.state.Party[index]
		if len(member.Effects) == 0 {
			continue
		}
		list, expired := combatEffects(member.Effects).AdvanceEffects(minutes)
		for _, node := range expired {
			a.expiredEffectTeardown(index, node)
		}
		member.Effects = storedEffects(list)
		syncTrainedLibraryCharacter(&a.state, *member)
	}
}

// expiredEffectTeardown 是 **overlay-24 entry 2（`0028h`）** 摘節點之前那一
// 步：`+4`（有收尾）非 0 的節點，先用**模式 1** 叫一次自己代碼的處理常式，
// 再摘掉（spec 112）。模式 0 是套用、模式 1 是收尾，同一支常式兩個入口。
//
// **地圖上目前一個代碼都不需要動作。** remake 實作過收尾的只有 `28h`——
// 雲團是盤面上的物件，走出戰場就不存在了，所以在這裡摘掉節點就是全部。
// 這一支留著是為了讓兩條路（戰場的 `tickEffects`、地圖的 `AdvanceEffects`）
// 有同一個收尾入口：下一個代碼接上來時不會只接到一邊。
func (a *app) expiredEffectTeardown(party int, node gamepack.EffectNode) {
	if !node.NeedsTeardown() {
		return
	}
	if node.Code == gamepack.CloudObjectEffectCode {
		// 雲團在盤面上，地圖上沒有東西可以收——戰場那一側是
		// `tacticalState.effectTeardown` → `disperseCloud`。
		return
	}
}
