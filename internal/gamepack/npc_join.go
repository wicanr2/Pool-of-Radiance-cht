package gamepack

import "fmt"

// ADD NPC 把記錄掛進隊伍的那一支（overlay-17 `13EDh`，spec 154）在 `150Fh` 呼叫
// overlay-23 entry 1（`00FCh:0025h` → `0000h`）：與建角、訓練同一支「依職業等級重算
// 衍生數值」（spec 072）。remake 接上其中三格：
//
//	0009h..0062h  +2Dh = 八個職業查 `3C16h` 取最大（先清 0，等級 0 那一列也參與，值是 40）
//	0066h..0078h  +73h = max(+73h, 各職業等級)（只往上寫）
//	007Ch..009Dh  +A1h = 戰士等級 > 6 ? 3 : 2（每一筆都寫）
//
// 豁免（`0253h`）由呼叫端用 SavingThrowTable 寫；法術格數（`00B0h..017Dh`）與盜賊技能
// （`031Eh`）沒有接，見 spec 154〈還沒接〉。
const (
	// AttackRateOffset 是記錄 `+A1h`：第一形態一回合的攻擊次數編碼。
	AttackRateOffset = 0xa1
	// HitDiceOffset 是記錄 `+73h`：最高職業等級（怪物是生命骰）。
	HitDiceOffset = 0x73
)

// ApplyJoinDerivedStats 把上面三格寫回 285-byte 記錄。
func ApplyJoinDerivedStats(record []byte) error {
	if len(record) <= ClassLevelOffset+ClassThac0ClassCount {
		return fmt.Errorf("Pool NPC record has %d bytes", len(record))
	}
	var levels [ClassThac0ClassCount]uint8
	copy(levels[:], record[ClassLevelOffset:])
	base, err := BaseThac0Internal(levels)
	if err != nil {
		return err
	}
	// 表的第 0 列（等級 0）八個職業都是 40，所以「等級 0 也參與」與 BaseThac0Internal
	// 跳過等級 0 只差在全部是 0 的記錄：那時原版是 40。
	if row := OriginalClassThac0Table(); base < row[0][0] {
		base = row[0][0]
	}
	record[BaseThac0Offset] = base
	for _, level := range levels {
		if int8(record[HitDiceOffset]) < int8(level) {
			record[HitDiceOffset] = level
		}
	}
	record[AttackRateOffset] = PlayerAttackRate(levels)
	return nil
}
