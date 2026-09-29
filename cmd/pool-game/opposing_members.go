package main

import (
	"encoding/binary"

	poolchar "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 站在對面的隊員在戰後怎麼算（#122，spec 167）。
//
// 隊伍串列 `DS:5CF4h` 在戰鬥中就是戰鬥員串列：隊員在前、怪物在後。Attack Ally 答 Y 之後
// 倒戈的 NPC（spec 162 `2A0Ch`）、ADD NPC 編號 18h 帶進來的那一位（spec 091）都還掛在
// 隊伍這一段，只是記錄 `+10Eh` 是 1。戰後主流程 `14CAh` 對它們做的事全部照 `+10Eh`：
//
//	04ADh  → entry 2（`0068h`）：`+10Eh == 1` 而且 `+10Ch != 3` 就當敵方——經驗值
//	         `+B8h + +BAh × +B1h`、七種錢 `+88h` 起加進公款、物品照怪物的規則複製進戰利品
//	1164h  → `119Eh` `+10Eh == 1` 的記錄從串列摘掉（`11EFh` overlay-16 entry 3，runtime `+13h`
//	         是 0 所以隊伍人數減一，spec 150）
//	1295h  → NPC 分錢：那時人已經不在串列上，不分、也不列名
//
// 也就是**倒戈的 NPC 被打倒之後像一隻怪物那樣結算，然後離隊**。

// opposingMember 是隊伍裡站在對面的一位：slot 是隊伍索引，status 是戰後那一刻的 `+10Ch`
// （在盤面上的讀盤面，沒上場的讀存檔那一份）。
type opposingMember struct {
	slot   int
	status uint8
}

// opposingMembers 列出隊伍裡 `Side != 0` 的人，依隊伍順序（就是串列順序）。競技場的
// 複製品不算：那一場 entry 2 在 `0006h` 整段跳過，複製品另由 removeArenaCopy 處理。
func (a *app) opposingMembers(state *tacticalState) []opposingMember {
	var members []opposingMember
	for slot, member := range a.state.Party {
		if member.Side == 0 || a.isArenaCopy(slot) {
			continue
		}
		status := member.Status
		if state != nil {
			for index := 1; index < len(state.PartySlot) && index < len(state.States); index++ {
				if state.PartySlot[index] == slot {
					status = state.States[index]
					break
				}
			}
		}
		members = append(members, opposingMember{slot: slot, status: status})
	}
	return members
}

// memberDOSRecord 是這一位在原版的 285-byte 記錄：NPC 帶著自己那一份；玩家建的角色
// 用 DOS 匯出那一條疊出來（dos_export.go 同一個 base）。
func memberDOSRecord(member poolsave.Character) []byte {
	if len(member.Record) == poolchar.DOSRecordSize {
		return member.Record
	}
	record, err := poolchar.ExportDOSRecord(poolchar.NewDOSRecordBase(), member)
	if err != nil {
		return nil
	}
	return record
}

// opposingMembersExperience 是 entry 2 `00C0h..00F4h` 對站在對面的隊員：
// `+B8h`（word）＋ `+BAh` × `+B1h`。逃掉的（`0079h` 的 `+10Ch == 3`）不算。
func (a *app) opposingMembersExperience(members []opposingMember) uint32 {
	total := uint32(0)
	for _, opposing := range members {
		if opposing.status == gamepack.FledState {
			continue
		}
		record := memberDOSRecord(a.state.Party[opposing.slot])
		if len(record) <= 0xBA {
			continue
		}
		total += uint32(binary.LittleEndian.Uint16(record[0xB8:])) + uint32(record[0xBA])*uint32(record[0xB1])
	}
	return total
}

// collectOpposingMembers 是 entry 2 `0089h..01FBh` 對站在對面的隊員：錢加進 loot，物品照
// 怪物的規則收（價值 0 的受八件與 d10 限制，taken 與怪物共用一個計數——`[bp-12h]` 在
// `0049h` 只清一次）。隊員在串列上排在怪物前面，所以先收他們。remake 的錢包是
// `Character.Money`（DOS 匯出寫進 `+88h` 的就是它）。
func (a *app) collectOpposingMembers(loot *monsterLoot, members []opposingMember, skipItems bool, taken *int) {
	for _, opposing := range members {
		if opposing.status == gamepack.FledState {
			continue
		}
		member := a.state.Party[opposing.slot]
		for currency := 0; currency < pooltreasure.CurrencyCount && currency < len(member.Money); currency++ {
			loot.money[currency] += uint32(member.Money[currency])
		}
		if skipItems {
			continue
		}
		for _, item := range member.Inventory {
			a.takeLootItem(loot, item, taken)
		}
	}
}

// removeOpposingMembers 是 `1164h`：`+10Eh == 1` 的隊員從隊伍摘掉。回傳有沒有人被摘。
func (a *app) removeOpposingMembers() bool {
	kept := a.state.Party[:0]
	removed := false
	for slot, member := range a.state.Party {
		if member.Side != 0 && !a.isArenaCopy(slot) {
			removed = true
			continue
		}
		kept = append(kept, member)
	}
	a.state.Party = kept
	if a.currentCharacter >= len(a.state.Party) {
		a.currentCharacter = 0
	}
	return removed
}

// opposingMemberHoldsTheField 是 `04ADh` 在隊員全倒或逃光時的那一個分岔：`052Eh..05AEh`
// 對隊伍那一段（runtime `+13h == 0`）狀態 0／1 的任何一位立 82A0h、清 439Dh，**不看陣營**。
// 站到對面的隊員還站著，而隊伍這一邊沒有全滅（4960h：`+10Eh == 0`、`+84h < 80h`、
// 狀態 0／1／3 的至少一位，`054Eh..058Eh`），結算就走打贏那一條：狀態換算、entry 2／3、
// 不留人、`14EDh` 不是全滅。
func (a *app) opposingMemberHoldsTheField(state *tacticalState) bool {
	if state == nil || a.arenaCopy {
		return false
	}
	holding := false
	for _, opposing := range a.opposingMembers(state) {
		if opposing.status == 0 || opposing.status == gamepack.AnimatedState {
			holding = true
		}
	}
	if !holding {
		return false
	}
	states := make([]uint8, len(a.state.Party))
	for slot, member := range a.state.Party {
		states[slot] = member.Status
	}
	for index := 1; index < len(state.PartySlot) && index < len(state.States); index++ {
		if slot := state.PartySlot[index]; slot >= 0 && slot < len(states) {
			states[slot] = state.States[index]
		}
	}
	_, wiped := a.partyFledOutcome(states)
	return !wiped
}
