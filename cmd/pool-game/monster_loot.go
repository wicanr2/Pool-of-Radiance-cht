package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 怪物身上的物品與錢（spec 142，issue #76）。
//
// 載入：`0Bh LOAD MONSTER`（overlay-03 `044Dh`）→ overlay-07 entry 6（`0423h`）→
// overlay-17 entry 8（`0E90h`）讀 MONnCHA／MONnSPC／MONnITM 同一個 block。第一隻用
// 載入常式那一份（檔案順序）；第二隻起 overlay-03 `0629h..06E9h` 逐件 GetMem(3Fh)、
// Move，**每一件都插在串列頭**（`0684h`：舊的頭接到新節點的 +2Ah），所以順序是反的。
//
// 戰後：overlay-05 entry 2（`0000h`）沿 `DS:5CF4h` 的串列，對 `+10Eh == 1` 而且
// `+10Ch != 3`（沒有逃掉）的每一隻：
//
//	0089..00BE  七種錢 `+88h + i×2`（word）加進 `DS:6752h + i×4`（公款）
//	00F7        `[4937h]+5C6h`（ECL @6DE3）== 1 → 物品整段跳過
//	0111        物品 +3Ah（word，價值）> 0 → 收
//	011B        否則已收滿 8 件 → 不收；否則 Roll(1, 10) <= 3 → 收
//	0136..01FB  收的那件：件數加一、overlay-25 entry 1 把名稱重組寫回 +0、
//	            GetMem(3Fh) 複製、+34h（穿戴中）清 0、**插在 `DS:676Eh` 串列頭**
//
// 主流程 `14CAh` 最後在 `15FBh` 把 @6DE3 清回 0。

// monsterLootItemCap 是 `011Bh` 的 8：價值 0 的物品在收滿 8 件之後就不再擲骰。
const monsterLootItemCap = 8

// monsterLootValueOffset 是物品的價值（word，`0111h` 的 `cmp word es:[di+3Ah], 0`）。
const monsterLootValueOffset = 0x3A

// monsterLootNoItemsAddress 是 ECL @6DE3（`[4937h] + 2A00h + 6DE3h × 2 ≡ 5C6h`）。
// 整個 ECL 只有 ECL3 block 0 `A764h` 一條 `SAVE 1` 寫它，接在一場隨機編成的戰鬥前面。
const monsterLootNoItemsAddress = 0x6DE3

// monsterLoot 是一場打完之後怪物那一側交出來的東西。
type monsterLoot struct {
	money [pooltreasure.CurrencyCount]uint32
	// items 照 `DS:676Eh` 串列的順序（頭在前）。
	items []gamepack.TreasureItemRecord
}

func (loot monsterLoot) empty() bool {
	return loot.money == [pooltreasure.CurrencyCount]uint32{} && len(loot.items) == 0
}

// rememberFoeItems 把這一隻的物品串列放上盤面。第二隻起順序反過來（`0684h`）。
func (a *app) rememberFoeItems(state *tacticalState, index int, monster stagedMonster, copyIndex int) {
	if len(monster.Items) == 0 {
		return
	}
	items := make([]poolsave.Item, 0, len(monster.Items))
	for _, raw := range monster.Items {
		items = append(items, a.namedItem(raw))
	}
	if copyIndex > 0 {
		for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
			items[left], items[right] = items[right], items[left]
		}
	}
	if state.FoeItems == nil {
		state.FoeItems = map[int][]poolsave.Item{}
	}
	state.FoeItems[index] = items
}

// namedItem 是 overlay-25 entry 1：依記錄重組名稱寫回 +0（spec 067）。名稱表不在或
// 組不出名字時照原樣（名字是空的）。
func (a *app) namedItem(raw []byte) poolsave.Item {
	item := poolsave.Item{Raw: append([]byte(nil), raw...)}
	if a.itemNames == nil || len(item.Raw) != gamepack.MonsterItemRecordSize {
		return item
	}
	name, err := pooltreasure.ItemName(item.Raw, a.itemNames)
	if err != nil || name == "" {
		return item
	}
	item.Raw[0] = byte(len(name))
	copy(item.Raw[1:], name)
	item.Name = name
	return item
}

// collectMonsterLoot 是 overlay-05 entry 2 收錢與物品的那一段（見檔頭）。只在打贏時呼叫：
// 原版 `04ADh` 在 `05E0h` 看 `DS:82A0h`（有隊員站著）才叫 entry 2。
func (a *app) collectMonsterLoot(state *tacticalState) monsterLoot {
	var loot monsterLoot
	if state == nil {
		return loot
	}
	skipItems := false
	if a.eventMachine != nil {
		skipItems = a.eventMachine.Memory[monsterLootNoItemsAddress] == 1
		// `15FBh`：戰後主流程把它清回 0。
		a.eventMachine.Memory[monsterLootNoItemsAddress] = 0
	}
	// 戰鬥中丟出去落地的武器（spec 151）已經在戰利品串列上；怪物的物品之後插在它們前面。
	for _, item := range state.ThrownLoot {
		named := a.namedItem(item.Raw)
		if named.Name == "" {
			named.Name = item.Name
		}
		if named.Name == "" || len(named.Raw) != gamepack.MonsterItemRecordSize {
			continue
		}
		var record gamepack.TreasureItemRecord
		record.Name = named.Name
		copy(record.Raw[:], named.Raw)
		record.Raw[gamepack.ItemReadiedOffset] = 0
		loot.items = append(loot.items, record)
	}
	friendly := state.Friendly
	taken := 0
	for index := 1; index < len(state.Roster) && index < len(friendly); index++ {
		if friendly[index] {
			continue
		}
		monster, ok := a.stagedMonsterFor(index, friendly)
		if !ok {
			continue
		}
		// `0079h`：逃掉的（`+10Ch == 3`）整段跳過，與經驗值同一道（fledFoeRecords）。
		if index < len(state.States) && state.States[index] == gamepack.FledState &&
			state.Roster[index].FootprintClass == 0 {
			continue
		}
		for currency := range loot.money {
			offset := gamepack.MonsterMoneyOffset + currency*2
			amount := uint32(monster.Record.Raw[offset]) | uint32(monster.Record.Raw[offset+1])<<8
			loot.money[currency] += amount
		}
		if skipItems {
			continue
		}
		for _, item := range state.FoeItems[index] {
			if len(item.Raw) != gamepack.MonsterItemRecordSize {
				continue
			}
			value := uint16(item.Raw[monsterLootValueOffset]) | uint16(item.Raw[monsterLootValueOffset+1])<<8
			if value == 0 {
				if taken >= monsterLootItemCap || a.rollDice(1, 10) > 3 {
					continue
				}
			}
			taken++
			named := a.namedItem(item.Raw)
			if named.Name == "" {
				// 組不出名字的存不進存檔（`save.validateItem`）；原版不會發生，這裡失敗即關閉。
				continue
			}
			var record gamepack.TreasureItemRecord
			record.Name = named.Name
			copy(record.Raw[:], named.Raw)
			record.Raw[gamepack.ItemReadiedOffset] = 0
			loot.items = append([]gamepack.TreasureItemRecord{record}, loot.items...)
		}
	}
	return loot
}

// openMonsterLoot 把錢加進公款、物品交給戰後戰利品選單（spec 034）。沒有東西就不開，
// 回 false，由呼叫端照舊續跑戰後腳本。
func (a *app) openMonsterLoot(loot monsterLoot) bool {
	if loot.empty() {
		return false
	}
	for currency, amount := range loot.money {
		total := uint64(a.state.PooledMoney[currency]) + uint64(amount)
		if total > uint64(^uint32(0)) {
			total = uint64(^uint32(0))
		}
		a.state.PooledMoney[currency] = uint32(total)
	}
	a.treasureActive, a.treasureStage = true, treasureMain
	a.treasureItems, a.treasureSelected, a.treasureCurrency, a.treasureAmount = loot.items, 0, 0, ""
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.enterTreasureMain()
	// `1295h`：開選單之前 NPC 先拿走份額（spec 148）。
	a.hideNPCShares()
	return true
}
