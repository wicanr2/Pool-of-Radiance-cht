package creation

// 建角肖像的預設值（#130，spec 006〈預設肖像〉）。
//
// 原版玩家建角一律從 HEAD 1／BODY 1 開始：overlay-16 `05F5h`／`05FEh` 無條件把
// CHA `+BBh`／`+BCh` 寫成 1，不看性別。remake **刻意偏離**這一點（使用者
// 2026-09-30 定案）：女性一開始就看到男性頭像，玩家會以為選不到。
//
// 偏離的依據仍是原版自己的資料：overlay-17 `1216h` 產生角色時依性別（CHA
// `+9Eh`）與職業挑肖像，表在 START.EXE（SHA-256 `12811cbc8166…`）的資料段：
//
//   - HEAD：DS `8DBh + gender×8 + (1..8)`
//   - BODY：DS `<表> + gender×5 + (1..5)`，表依職業列索引 `+96h` 選：
//     牧師 0 → `8EBh`、法師 5 → `8FFh`、盜賊 6 → `909h`、其他 → `8F5h`
//
// 原版在那一支裡用亂數挑格；這裡固定取每一格的第一個（使用者選的做法），
// 同一個人每次建角看到的預設都一樣。多職業取列索引最小的那個組成職業
// （原版是在組成職業裡隨機挑一個）。
//
// 表的位元組由 `TestDefaultPortraitTablesMatchStartEXE` 對原版檔核對。
var (
	defaultPortraitHeads = [2][8]uint8{
		{1, 2, 3, 4, 5, 8, 11, 12},  // DS 8DCh..8E3h，男性
		{6, 7, 9, 10, 13, 14, 6, 7}, // DS 8E4h..8EBh，女性
	}
	defaultPortraitBodies = map[uint8][2][5]uint8{
		0:    {{2, 4, 2, 4, 2}, {2, 4, 8, 2, 4}},    // 牧師，DS 8ECh..8F5h
		5:    {{2, 2, 2, 2, 2}, {8, 10, 12, 8, 10}}, // 法師，DS 900h..909h
		6:    {{6, 6, 6, 6, 6}, {8, 10, 8, 10, 8}},  // 盜賊，DS 90Ah..913h
		0xFF: {{1, 3, 4, 5, 6}, {7, 9, 11, 7, 9}},   // 其他，DS 8F6h..8FFh
	}
)

// DefaultPortrait 回傳某個性別與職業組合的預設 HEAD／BODY selector。
// 查不到職業時退回原版的 1／1。
func DefaultPortrait(genderCode uint8, classID string) (head, body uint8) {
	if genderCode > 1 {
		return 1, 1
	}
	components, ok := ClassComponents(classID)
	if !ok || len(components) == 0 {
		return 1, 1
	}
	lowest := uint8(0xFF)
	for _, component := range components {
		if index, ok := ComponentClassIndex(component); ok && index < lowest {
			lowest = index
		}
	}
	if lowest == 0xFF {
		return 1, 1
	}
	table, ok := defaultPortraitBodies[lowest]
	if !ok {
		table = defaultPortraitBodies[0xFF]
	}
	return defaultPortraitHeads[genderCode][0], table[genderCode][0]
}

// DefaultPortrait 是目前建角選擇對應的預設肖像。
func (flow Flow) DefaultPortrait() (head, body uint8) {
	return DefaultPortrait(flow.SelectedGender().DOSCode, flow.SelectedClass().ID)
}
