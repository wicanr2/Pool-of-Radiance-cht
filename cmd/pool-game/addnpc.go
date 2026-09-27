package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `36h ADD NPC`（spec 091）：依編號從 `MON<n>CHA.DAX` 讀一筆 285-byte 記錄，
// 加進隊伍。編號 18h 那一位站到對面。

// addNPCEvent 找出結果裡的 `36h`。
func addNPCEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.AddNPCOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// applyAddNPC 把 NPC 加進隊伍，然後讓 ECL 繼續。
func (a *app) applyAddNPC(event eclvm.Event) error {
	instruction, err := a.eclInstruction(event.PC)
	if err != nil {
		return fmt.Errorf("decode Pool ADD NPC at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.AddNPCOperands {
		return fmt.Errorf("Pool ADD NPC has %d operands, want %d",
			len(instruction.Operands), gamepack.AddNPCOperands)
	}
	memory := a.eventSession.Machine().Memory
	id, err := ecl.NumericValue(instruction.Operands[0], memory)
	if err != nil {
		return fmt.Errorf("Pool ADD NPC id operand: %w", err)
	}
	if id > 0xFF {
		return fmt.Errorf("Pool ADD NPC id %d exceeds one byte", id)
	}
	// 隊伍滿了就不加（原版在 overlay-17 entry 9 擋「人數大於 7」）。
	if len(a.state.Party) >= poolsave.PartyMaximum {
		return a.continueInitialSearch(nil)
	}
	if a.loadMonster == nil {
		return fmt.Errorf("Pool monster loader is not configured")
	}
	archive := a.eclArchive
	if archive == 0 {
		archive = a.spawn.Map.Archive
	}
	record, err := a.loadMonster(archive, uint8(id))
	if err != nil {
		return fmt.Errorf("load Pool NPC archive %d block %d: %w", archive, id, err)
	}
	// 物品與記錄同一支讀（overlay-17 entry 9 `1244h` → `0E90h` 也讀 MONnITM，順序照檔案，
	// spec 142）。開打時 entry 7 從這一條認武器與盔甲（#97，spec 147）。
	items, err := a.npcItems(archive, uint8(id))
	if err != nil {
		return err
	}
	member := poolsave.Character{
		Name:      a.monsterText.Translate(record.Name),
		NPC:       true,
		Side:      gamepack.AddNPCSide(uint8(id)),
		Record:    append([]byte(nil), record.Raw[:]...),
		MaxHP:     int(record.MaxHitPoints()),
		CurrentHP: int(record.CurrentHitPoints()),
	}
	member.Inventory = items
	// 掛進隊伍（overlay-17 `13EDh` 的 `150Fh`）時跑 overlay-23 entry 1：`+2Dh`、`+73h`、`+A1h`
	// 照職業等級重寫（spec 154）。樣板的 `+2Dh` 不一定是這個值：WARRIOR 40 → 41、HERO 與
	// PRINCESS FATIMA 42 → 43，dosgolem 雇 WARRIOR 那一刻讀到的是 41。
	if err := gamepack.ApplyJoinDerivedStats(member.Record); err != nil {
		return fmt.Errorf("Pool NPC %q: %w", member.Name, err)
	}
	// **職業要從記錄帶出來。** NPC 沒有經過建角流程，它的職業只存在記錄的
	// `+2Fh`（複合職業碼）與 `+96h` 起的八個等級裡。不帶出來的話後面任何
	// 要查職業的地方都會拿到空字串——症狀不是顯示錯，是
	// `partyClassCodes`（ECL 的隊伍查詢）與 `partyClassLevels`（THAC0 與
	// 豁免）直接報錯，整局停在那裡。
	raw := record.Raw[:]
	if len(raw) > gamepack.ClassCodeOffset {
		if classID, ok := creation.ClassIDForDOSCode(raw[gamepack.ClassCodeOffset]); ok {
			member.ClassID = classID
		}
	}
	// **等級全是 0 的記錄不要帶。** 98 筆裡有 12 筆是這樣（純怪物，不是真的
	// NPC）。帶了的話 `partyClassLevels` 會看到「有等級」而直接回一組全零，
	// 而不是退回「沒訓練過就照建角的第 1 級算」（spec 097）。
	if levels, err := gamepack.ClassLevels(raw); err == nil {
		for _, level := range levels {
			if level != 0 {
				member.ClassLevels = append([]uint8(nil), levels[:]...)
				break
			}
		}
	}
	// 士氣（overlay-03 `2EFDh..2F25h`）：第二個運算元取低位元組、除以 2、立位元 7，
	// 寫進記錄 `+84h`。士氣判定讀它（spec 096〈entry 8〉，foe_flee.go）；不寫的話
	// NPC 留著怪物檔的 FFh（士氣 0），在隊伍那一側每一回合都過不了士氣。
	morale, err := ecl.NumericValue(instruction.Operands[1], memory)
	if err != nil {
		return fmt.Errorf("Pool ADD NPC morale operand: %w", err)
	}
	member.Record[gamepack.MoraleOffset] = gamepack.NPCMoraleByte(uint8(morale))
	// 效果串列同一支讀（`0E90h` 的 `1051h`：MONnSPC 同一個 block，鏈在 `+7Fh`，spec 154）。
	if a.loadMonsterEffects != nil {
		effects, err := a.loadMonsterEffects(archive, uint8(id))
		if err != nil {
			return fmt.Errorf("load Pool NPC effects archive %d block %d: %w", archive, id, err)
		}
		member.Effects = storedEffects(effects)
	}
	// overlay-03 `2F2Ah..2F40h`：陣營寫進記錄 `+10Eh`（編號 18h 是 1）。
	member.Record[npcSideOffset] = member.Side
	// overlay-03 `2F46h`：加入時跑一次 entry 7，結果寫在這一位自己的記錄裡（spec 154）。
	if err := a.recomputeNPCRecord(&member); err != nil {
		return err
	}
	if member.CurrentHP > member.MaxHP {
		member.CurrentHP = member.MaxHP
	}
	a.state.Party = append(a.state.Party, member)
	a.eventText = fmt.Sprintf(a.text(msgNPCJoined), member.Name)
	return a.continueInitialSearch(nil)
}

// migrateNPCMorale 把舊存檔裡士氣還是怪物檔原值（`FFh`）的隊伍 NPC 補回原版 ADD NPC
// 會給的士氣（#74）。原版加入時一定覆寫 `+84h`；只有 #74 之前的 remake 沒寫，而那個值在
// 隊伍這一側每回合都過不了士氣（兩關都不過就逃或投降）。以記錄 `+0` 的原文名字比對
// 八個呼叫點的怪物記錄；對不上的不動，記一行狀態。
func (a *app) migrateNPCMorale() {
	if a.loadMonster == nil {
		return
	}
	for index := range a.state.Party {
		member := &a.state.Party[index]
		if !member.NPC || len(member.Record) <= gamepack.MoraleOffset || member.Record[gamepack.MoraleOffset] != 0xFF {
			continue
		}
		name := recordName(member.Record)
		for _, source := range gamepack.NPCMoraleSources() {
			record, err := a.loadMonster(source.Archive, source.Block)
			if err != nil || strings.TrimSpace(record.Name) != name {
				continue
			}
			member.Record[gamepack.MoraleOffset] = gamepack.NPCMoraleByte(source.Morale)
			break
		}
	}
	a.migrateNPCRecords()
}

// migrateNPCRecords 把 #107 之前加入的 NPC 記錄補成 ADD NPC 當下該有的樣子：overlay-23
// entry 1 的三格、`+10Eh` 與 entry 7 的結果（spec 154）。三者都只由記錄本身、陣營與物品
// 決定，新存檔再跑一次得到同一組值，所以每次讀檔都跑。
//
// 物品欄空的舊 NPC 不補 MONnITM：schema 8 從 #97 之前就在用，檔案裡分不出「那時還沒載」
// 與「後來交給別人了」，補了會在後一種情形複製物品（spec 154〈舊存檔〉）。
func (a *app) migrateNPCRecords() {
	for index := range a.state.Party {
		member := &a.state.Party[index]
		if !member.NPC || len(member.Record) != poolsave.NPCRecordSize {
			continue
		}
		if err := gamepack.ApplyJoinDerivedStats(member.Record); err != nil {
			continue
		}
		member.Record[npcSideOffset] = member.Side
		if err := a.recomputeNPCRecord(member); err != nil {
			continue
		}
	}
}

// npcSideOffset 是記錄 `+10Eh`：0 與隊伍同一邊，1 是對面（spec 057）。
const npcSideOffset = 0x10e

// recomputeNPCRecord 是 overlay-25 entry 7 對 NPC 自己的記錄跑一次、結果寫回記錄
// （ADD NPC 的 `2F46h`，spec 154）。記錄是 NPC 在原版唯一的一份資料，`1Dh PARTYSTRENGTH`
// 讀的 `+110h`／`+111h` 就在這裡。沒有型別表（只建了盤面的測試）就不動。
func (a *app) recomputeNPCRecord(member *poolsave.Character) error {
	if a.itemTypes == nil || len(member.Record) != poolsave.NPCRecordSize {
		return nil
	}
	record, err := a.npcRecomputed(*member)
	if err != nil {
		return err
	}
	copy(member.Record, record.Raw[:])
	return nil
}

// npcRecomputed 是同一支 entry 7 的結果，不寫回。
func (a *app) npcRecomputed(member poolsave.Character) (gamepack.MonsterRecord, error) {
	var record gamepack.MonsterRecord
	copy(record.Raw[:], member.Record)
	record.Name = member.Name
	items := make([][]byte, 0, len(member.Inventory))
	for _, item := range member.Inventory {
		items = append(items, item.Raw)
	}
	return gamepack.RecomputeMonsterCombatFields(record, items, a.itemTypes)
}

// npcItems 讀這一位 NPC 的物品串列（MONnITM.DAX 同一個 block），名字照 overlay-25 entry 1 組。
func (a *app) npcItems(archive, block uint8) ([]poolsave.Item, error) {
	if a.loadMonsterItems == nil {
		return nil, nil
	}
	raws, err := a.loadMonsterItems(archive, block)
	if err != nil {
		return nil, fmt.Errorf("load Pool NPC items archive %d block %d: %w", archive, block, err)
	}
	var items []poolsave.Item
	for _, raw := range raws {
		items = append(items, a.namedItem(raw))
	}
	return items, nil
}

// recordName 是 285-byte 記錄 `+0` 的 Pascal 字串。
func recordName(record []byte) string {
	if len(record) == 0 || int(record[0])+1 > len(record) {
		return ""
	}
	return strings.TrimSpace(string(record[1 : 1+int(record[0])]))
}
