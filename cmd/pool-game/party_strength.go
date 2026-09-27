package main

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/combat/ability"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// livePartyStrength 是 `1Dh PARTYSTRENGTH` 的 resolver（spec 030），每次都從
// 目前的隊伍算（#61）。建 session 時那一份一級快照（`gamepack` 的
// `initialPartyStrengthResolver`）不看訓練、升級與裝備，腳本讀到的隊伍強度會
// 一直偏低。原版的 handler 沿 `DS:5CF4h` 的隊伍串列逐筆讀記錄，這裡同樣逐人算。
func (a *app) livePartyStrength() (uint8, error) {
	records := make([]character.PartyStrengthRecord, 0, len(a.state.Party))
	for index, member := range a.state.Party {
		record, err := a.partyStrengthRecord(member)
		if err != nil {
			return 0, fmt.Errorf("Pool party member %d %q: %w", index, member.Name, err)
		}
		records = append(records, record)
	}
	return character.PartyStrength(records), nil
}

// partyStrengthRecord 把一個隊員投影成那五個 byte：
//
//	+96h   牧師等級
//	+9Bh   魔法師等級
//	+110h  儲存的攻擊值：職業等級表的 THAC0 內部值（overlay-16 `0CC6h`）＋力量命中調整
//	+111h  儲存的 AC 內部值：建角的 50 ＋ 敏捷，再算身上的裝備（spec 079）
//	+11Bh  目前 HP
//
// 一級時與 spec 031 的三份原版 `.CHA` 相同（FEM 40/50、HMU 41/50、HTH 40/52）。
// NPC 帶著自己的 285-byte 記錄，直接讀那五格，HP 用目前值。
func (a *app) partyStrengthRecord(member poolsave.Character) (character.PartyStrengthRecord, error) {
	hp := member.CurrentHP
	if hp < 0 {
		hp = 0
	}
	if hp > 0xFF {
		return character.PartyStrengthRecord{}, fmt.Errorf("current HP %d is outside byte range", hp)
	}
	// NPC 讀自己的記錄，而且**先於**職業等級那一條：NPC 從記錄帶出了等級（addnpc.go），
	// 走下去會被當成能力值全 0 的玩家角色算。記錄裡的 `+110h`／`+111h` 是 entry 7 寫的
	// （ADD NPC `2F46h`、換裝、開打都會再跑一次），它只由記錄與物品決定，所以這裡照同一支
	// 現算，舊存檔裡還是樣板殘值的記錄也拿得到對的值（spec 154）。
	if member.NPC && len(member.Record) == poolsave.NPCRecordSize {
		raw := member.Record
		if a.itemTypes != nil {
			record, err := a.npcRecomputed(member)
			if err != nil {
				return character.PartyStrengthRecord{}, err
			}
			raw = record.Raw[:]
		}
		return character.PartyStrengthRecord{
			Field96: raw[0x96], Field9B: raw[0x9B], Field110: raw[0x110], Field111: raw[0x111],
			Field11B: uint8(hp),
		}, nil
	}
	levels, err := partyClassLevels(member)
	if err != nil {
		if len(member.Record) != poolsave.NPCRecordSize {
			return character.PartyStrengthRecord{}, err
		}
		raw := member.Record
		return character.PartyStrengthRecord{
			Field96: raw[0x96], Field9B: raw[0x9B], Field110: raw[0x110], Field111: raw[0x111],
			Field11B: uint8(hp),
		}, nil
	}
	base, err := gamepack.BaseThac0Internal(levels)
	if err != nil {
		return character.PartyStrengthRecord{}, err
	}
	strengthIndex, ok := ability.StrengthIndex(member.Abilities[0], member.ExceptionalStrength)
	if !ok {
		return character.PartyStrengthRecord{}, fmt.Errorf("strength %d/%d has no original table index",
			member.Abilities[0], member.ExceptionalStrength)
	}
	attack := int(base) + ability.StrengthHitAdjustment(strengthIndex)
	armour, _, err := a.memberDefenceStats(member, creationArmorClassInternal, creationBaseMovement)
	if err != nil {
		return character.PartyStrengthRecord{}, err
	}
	if attack < 0 || attack > 0xFF || armour < 0 || armour > 0xFF {
		return character.PartyStrengthRecord{}, fmt.Errorf("derived attack/armour %d/%d is outside byte range", attack, armour)
	}
	return character.PartyStrengthRecord{
		Field96: levels[gamepack.ClassSlotCleric], Field9B: levels[gamepack.ClassSlotMagicUser],
		Field110: uint8(attack), Field111: uint8(armour), Field11B: uint8(hp),
	}, nil
}
