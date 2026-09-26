package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 戰後結算的後兩段（spec 148，issue #94）。
//
// overlay-05 主流程 `14CAh`：`04ADh` 在 `05E0h` 呼叫 entry 2（`0000h`）算經驗總額、
// entry 3（`033Ah`）發下去；接著 `1295h` 讓 NPC 拿走份額，`08E0h` 印標題，`0E85h`
// 開戰利品選單。entry 2 的總額是三項加起來再除一次人數：
//
//	00C0h..00F4h  每隻敵方 `+0B8h + +0BAh × 生命值`（spec 097）
//	0224h..02B1h  公款七欄（這時已經加上怪物身上的錢）依幣值換算
//	02B4h..0306h  戰利品串列每件加值 > 0 的 `加值 × 400`
//
// 沒有怪物的 `TREASURE → COMBAT`（spec 036）走的是同一條，所以撿到寶物、委任的
// 獎金也折成經驗值——`docs/audit/dosgolem-loot-experience.json` 是原版的收據。

// lootExperience 是 `0224h..0306h`。pool 是 entry 2 那一刻的公款；items 是那一刻的
// 戰利品串列（頭在前）。`DS:5CF8h` 的停點 remake 不模擬（spec 148〈停點〉）。
func lootExperience(pool [pooltreasure.CurrencyCount]uint32, items []gamepack.TreasureItemRecord) uint32 {
	plus := make([]int8, 0, len(items))
	for _, item := range items {
		plus = append(plus, int8(item.Raw[gamepack.LootItemPlusOffset]))
	}
	return gamepack.LootExperience(pool, plus)
}

// shareExperience 是 `0308h` 除人數與 entry 3（`033Ah`）依職業調整（spec 097），回每份
// （DS:829Ch，戰後結算頁印的那個數字）。
//
// eligible 是有資格分的人（`+10Dh` 非 0 而且狀態不是 1，spec 150〈有資格的人數〉）；
// entry 3 只發給他們。除數是隊伍人數減掉沒資格的人數（`0308h..0318h` 的
// `[4937h]+67Ch − DS:829Bh`）；divisor 大於 0 時直接用它——決鬥那一條不數 `829Bh`
// （`04ADh` 在 `0524h` 跳過那一段），除的是整隊人數。
func (a *app) shareExperience(total uint32, eligible []bool, divisor int) uint32 {
	if divisor <= 0 {
		for index := range a.state.Party {
			if index < len(eligible) && eligible[index] {
				divisor++
			}
		}
	}
	share := gamepack.DivideExperience(total, divisor)
	if share == 0 {
		return 0
	}
	for index := range a.state.Party {
		if index >= len(eligible) || !eligible[index] {
			continue
		}
		member := &a.state.Party[index]
		code, ok := creation.ClassDOSCode(member.ClassID)
		if !ok {
			// NPC 帶的是自己的 285-byte 記錄，職業碼在 `+2Fh`。
			if len(member.Record) > gamepack.ClassCodeOffset {
				code = member.Record[gamepack.ClassCodeOffset]
			} else {
				continue
			}
		}
		member.Experience += gamepack.ExperienceShare(share, code, member.Abilities)
	}
	return share
}

// awardTreasureExperience 是沒有怪物的那一場（`TREASURE → COMBAT`）：怪物那一項是 0，
// 總額只有公款與戰利品。
func (a *app) awardTreasureExperience(items []gamepack.TreasureItemRecord) uint32 {
	return a.shareExperience(lootExperience(a.state.PooledMoney, items), a.treasureExperienceEligible(), 0)
}

// monsterLootExperience 是有怪物的那一場：entry 2 換算的公款是原本的公款加上怪物身上的錢
// （`0089h..00BEh` 在同一個迴圈裡先加進去）。
func (a *app) monsterLootExperience(loot monsterLoot) uint32 {
	pool := a.state.PooledMoney
	for currency, amount := range loot.money {
		total := uint64(pool[currency]) + uint64(amount)
		if total > uint64(^uint32(0)) {
			total = uint64(^uint32(0))
		}
		pool[currency] = uint32(total)
	}
	return lootExperience(pool, loot.items)
}

// hideNPCShares 是 `1295h`：公款扣掉 NPC 藏起來的份額，回拿走的人的名字——有名字就有
// 那一頁（`1387h..146Ah`，postcombat.go）。
func (a *app) hideNPCShares() []string {
	members := make([]gamepack.NPCShareMember, len(a.state.Party))
	for index, member := range a.state.Party {
		if !member.NPC || len(member.Record) <= 0x85 {
			// 玩家建的角色 `+84h` 是 0（shopCanSell 同一個判準）。
			members[index] = gamepack.NPCShareMember{Status: member.Status}
			continue
		}
		members[index] = gamepack.NPCShareMember{
			NPC:    member.Record[gamepack.MoraleOffset] >= gamepack.MoraleCheckedBit,
			Status: member.Status,
			Share:  member.Record[gamepack.MoraleOffset+1],
		}
	}
	pool, hiders := gamepack.HideNPCShares(a.state.PooledMoney, members)
	a.state.PooledMoney = pool
	return a.postCombatHiderNames(hiders)
}
