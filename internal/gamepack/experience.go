package gamepack

import (
	"encoding/binary"
	"fmt"
)

// 昇級所需的經驗值表。overlay-22 的昇級常式以
// `class × 38h + (level+1) × 4 + 4013h` 取值，所以是 8 個職業 × 14 個
// 4-byte 欄位；欄位索引 k 是「升到第 k 級所需的經驗值」，第 0、1 欄不用。
//
// `FFFFFFFF` 是上限哨兵：那一級到不了。四個職業的上限因此直接讀得出來——
// 牧師 6、戰士 8、法師 6、賊 9。
const (
	// ExperienceTableOffset 是表在 START.EXE 裡的 DS 位移。
	ExperienceTableOffset = 0x4013
	// ExperienceRowSize 是一個職業的 byte 數。
	ExperienceRowSize = 0x38
	// ExperienceLevelSlots 是一列的欄數。
	ExperienceLevelSlots = ExperienceRowSize / 4
	// ExperienceUnreachable 是上限哨兵。
	ExperienceUnreachable = ^uint32(0)
	// experienceFirstLevel 是第一個有意義的欄位索引，即升到第 2 級。
	experienceFirstLevel = 2
)

// ExperienceTable 是 8 個職業的門檻。索引與角色記錄 `+96h` 起的每職業等級
// 陣列同順序（spec 063）。
type ExperienceTable [ClassThac0ClassCount][ExperienceLevelSlots]uint32

// ParseExperienceTable 解出 8 × 56 bytes。
func ParseExperienceTable(raw []byte) (ExperienceTable, error) {
	var table ExperienceTable
	want := ClassThac0ClassCount * ExperienceRowSize
	if len(raw) != want {
		return table, fmt.Errorf("Pool experience table has %d bytes, want %d", len(raw), want)
	}
	for class := 0; class < ClassThac0ClassCount; class++ {
		row := raw[class*ExperienceRowSize:]
		for slot := 0; slot < ExperienceLevelSlots; slot++ {
			table[class][slot] = binary.LittleEndian.Uint32(row[slot*4:])
		}
	}
	return table, nil
}

// RequiredExperience 回傳某職業升到指定等級所需的經驗值。
// 到不了的等級回 false——原版用 `FFFFFFFF` 表示，不是一個很大的門檻。
func (t ExperienceTable) RequiredExperience(class int, level int) (uint32, bool) {
	if class < 0 || class >= ClassThac0ClassCount {
		return 0, false
	}
	if level < experienceFirstLevel || level >= ExperienceLevelSlots {
		return 0, false
	}
	value := t[class][level]
	if value == ExperienceUnreachable || value == 0 {
		return 0, false
	}
	return value, true
}

// MaxLevel 是這個職業能到的最高等級。第 1 級一律到得了，之後逐級看門檻。
func (t ExperienceTable) MaxLevel(class int) int {
	level := 1
	for next := experienceFirstLevel; next < ExperienceLevelSlots; next++ {
		if _, ok := t.RequiredExperience(class, next); !ok {
			break
		}
		level = next
	}
	return level
}

// ReadDOSExperienceTable 從 DOS ZIP 的 START.EXE 讀出這張表。
func ReadDOSExperienceTable(zipPath string) (ExperienceTable, error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return ExperienceTable{}, err
	}
	offset := ExperienceTableOffset + startDataSegmentFileDelta
	end := offset + ClassThac0ClassCount*ExperienceRowSize
	if len(raw) < end {
		return ExperienceTable{}, fmt.Errorf("START.EXE is %d bytes, the experience table needs %d", len(raw), end)
	}
	return ParseExperienceTable(raw[offset:end])
}
