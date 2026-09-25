package gamepack

// NPCMoraleSource 是一個 `36h ADD NPC` 呼叫點給的士氣（spec 091〈+84h 是士氣〉）：
// 第一個運算元是 MONnCHA 的區塊編號，第二個就是士氣。原版加入時一定覆寫記錄
// `+84h`，所以隊伍裡的 NPC 不會留著怪物檔的原值——只有 remake 在 #74 之前寫出的
// 存檔會。讀檔時拿這張表把那些 NPC 的士氣補回原版會給的值（`MigrateNPCMorale`）。
type NPCMoraleSource struct {
	Archive, Block, Morale uint8
}

// NPCMercenaryTiers 是 ecl3/11 `A170h` 的八格：雇傭兵等級 → MONnCHA 區塊。那一處的
// 士氣不是常數：`9F0Ah..9F1Ch` 是 `等級 × 5 + 50`（`MULTIPLY @6E7A #5`、`ADD #50`）。
var NPCMercenaryTiers = [8]uint8{0x6C, 0x67, 0x7A, 0x24, 0x6F, 0x70, 0x6D, 0x6E}

// NPCMoraleSources 是全部八個 ADD NPC 呼叫點：七個常數（ecl2/15 `A9F4h`、ecl3/0
// `A046h`、ecl4/10 `AE81h`、ecl4/2 `A5B8h`、ecl5/7 `A321h`、ecl8/13 `A8D8h`／`B2D1h`），
// 加上 ecl3/11 `9F1Ch` 的雇傭兵表展開。
func NPCMoraleSources() []NPCMoraleSource {
	sources := []NPCMoraleSource{
		{Archive: 2, Block: 25, Morale: 0},
		{Archive: 3, Block: 107, Morale: 99},
		{Archive: 4, Block: 24, Morale: 99},
		{Archive: 4, Block: 27, Morale: 100},
		{Archive: 5, Block: 88, Morale: 100},
		{Archive: 8, Block: 104, Morale: 100},
	}
	for tier, block := range NPCMercenaryTiers {
		sources = append(sources, NPCMoraleSource{Archive: 3, Block: block, Morale: uint8(tier*5 + 50)})
	}
	return sources
}

// NPCMoraleByte 是 ADD NPC 寫進 `+84h` 的值（overlay-03 `2EFDh..2F25h`）。
func NPCMoraleByte(morale uint8) uint8 {
	return morale/2 | MoraleCheckedBit
}
