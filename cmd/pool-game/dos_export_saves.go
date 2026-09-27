package main

import (
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 匯出 `.CHA` 時照職業等級重算的兩組欄位（spec 008〈仍未閉合〉、spec 075，issue #66）。
// 原版建角與升級都跑 overlay-23 那一支重算：
//
//	0066..0078  i = 0..7：+73h 有號小於 +96h + i → +73h = 它     ; 只往上（最高職業等級）
//	0253        五個豁免類別各取八個職業等級查 DS:41E6h 的最小值 → +6Dh..+71h
//
// `0253h` 是全部 overlay 唯一寫 `+6Dh` 的地方（spec 075），就是 gamepack.TargetsForLevels。
// remake 的角色模型只存職業等級，這兩組是現算的；不寫的話 remake 自己建的角色匯出去
// 豁免目標值全是 0（任何豁免都過）、`+73h` 是 0（橫掃、轉化不死生物把他當 0 級）。
const (
	dosSavingThrowOffset = 0x6D
	dosTopLevelOffset    = 0x73
)

func (a *app) recomputeDOSLevelFields(record []byte, member poolsave.Character) error {
	levels, err := partyClassLevels(member)
	if err != nil {
		return err
	}
	for _, level := range levels {
		if int8(record[dosTopLevelOffset]) < int8(level) {
			record[dosTopLevelOffset] = level
		}
	}
	if a.savingThrows == nil {
		// 表不在（原版 ZIP 讀不到）就留 base 的值，與其他沒有產生端的欄位同一個處置。
		return nil
	}
	targets, err := a.savingThrows.TargetsForLevels(levels)
	if err != nil {
		return err
	}
	copy(record[dosSavingThrowOffset:], targets[:])
	return nil
}
