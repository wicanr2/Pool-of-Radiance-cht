package gamepack

import "fmt"

// 豁免目標值那五格（`+6Dh`，spec 075）是誰寫的：overlay-23 `0253h`。
//
// 它把角色的八個職業等級（`+96h` 起連續八個位元組）各查一次表，五個類別
// 分別取最好的（數字最小）那一個：
//
//	for i in 0..4:
//	    record[+6Dh + i] = 20
//	    for j in 0..7:
//	        level = record[+96h + j]
//	        if level <= 0: continue
//	        value = table[j*45 + level*5 + i]
//	        if record[+6Dh + i] > value: record[+6Dh + i] = value
//
// 表在 `DS:41E6h`。每個職業佔 45 bytes ＝ 9 列 × 5 格，而**列是用等級直接
// 索引的**：AD&D 每三級才換一列，原版就把同一列重複三次。等級大於 8 時
// 索引會落進下一個職業的第 0 列——那不是溢位，是刻意的接續：牧師第 9 級
// 讀到的正是牧師 7..9 那一列的延續，賊第 9 級讀到的是 AD&D 賊 9..12 的值。
const (
	// SavingThrowTableAddress 是表在 START.EXE 資料段的位址。
	SavingThrowTableAddress = 0x41e6
	// SavingThrowTableClasses 是職業槽數，與記錄 `+96h` 起的等級陣列同長。
	SavingThrowTableClasses = 8
	// SavingThrowTableClassStride 是一個職業佔的位元組數。
	SavingThrowTableClassStride = 45
	// SavingThrowTableLevelStride 是一列的長度，等於類別數。
	SavingThrowTableLevelStride = SavingThrowCategories
	// SavingThrowWorstTarget 是初始化用的最差目標值（`026Bh` 的 14h）。
	SavingThrowWorstTarget = 20
	// ClassLevelsOffset 是記錄裡八個職業等級的起點。
	ClassLevelsOffset = 0x96
	// savingThrowTableBytes 讀到最後一個職業的第 9 列為止。
	savingThrowTableBytes = SavingThrowTableClasses*SavingThrowTableClassStride + SavingThrowTableLevelStride
)

// 職業槽的編號。由兩件事釘住：記錄 `+96h` 是牧師等級、`+9Bh` 是法師等級
//（相差 5），而表的第 5 個職業正是 AD&D 法師的那一列（14/13/11/15/12）。
const (
	ClassSlotCleric     = 0
	ClassSlotFighter    = 2
	ClassSlotMagicUser  = 5
	ClassSlotThief      = 6
)

// SavingThrowTable 是那 365 個位元組。
type SavingThrowTable struct {
	Raw []byte
}

// ReadDOSSavingThrowTable 從 DOS ZIP 的 START.EXE 取出表。
func ReadDOSSavingThrowTable(zipPath string) (*SavingThrowTable, error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	start := SavingThrowTableAddress + startDataSegmentFileDelta
	if start+savingThrowTableBytes > len(raw) {
		return nil, fmt.Errorf("START.EXE has %d bytes, the saving throw table needs %d", len(raw), start+savingThrowTableBytes)
	}
	return &SavingThrowTable{Raw: append([]byte(nil), raw[start:start+savingThrowTableBytes]...)}, nil
}

// TargetsForLevels 重現 `0253h`：八個職業等級各查一次，每個類別取最小值。
func (t *SavingThrowTable) TargetsForLevels(levels [SavingThrowTableClasses]uint8) ([SavingThrowCategories]uint8, error) {
	var targets [SavingThrowCategories]uint8
	if t == nil || len(t.Raw) < savingThrowTableBytes {
		return targets, fmt.Errorf("saving throw table has %d bytes, want %d", len(t.Raw), savingThrowTableBytes)
	}
	for category := 0; category < SavingThrowCategories; category++ {
		targets[category] = SavingThrowWorstTarget
		for slot, level := range levels {
			if level == 0 {
				continue
			}
			index := slot*SavingThrowTableClassStride + int(level)*SavingThrowTableLevelStride + category
			if index >= len(t.Raw) {
				return targets, fmt.Errorf("class slot %d level %d is outside the saving throw table", slot, level)
			}
			if value := t.Raw[index]; value < targets[category] {
				targets[category] = value
			}
		}
	}
	return targets, nil
}
