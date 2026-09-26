package gamepack

// 靈魂鎚（法術 28，overlay-22 `19A8h`）掛的 `17h` 與它的處理常式 overlay-12 entry 24
// `07F6h..0924h`（305 bytes，spec 098〈#99：收尾〉，issue #99）。輸入同 save_damage_effects.go。exact。
//
//	07FCh  沿記錄 `+0C8h` 的物品串列（`+2Ah` 接下一件）找 `+2Eh == 14h` 而且 `+31h == F3h` 的那一件
//	0849h  模式 != 0（摘節點時的收尾）而且找到 → 010Ah:0075h（overlay-25 entry 17 `156Ah`：
//	       從串列摘掉並釋放，spec 067）
//	0868h  模式 != 0 → 0916h
//	0871h  找到了 → 0916h（一個人只有一把）
//	087Ah  記錄 `+0C7h`（物品件數）>= 10h → 0916h
//	0888h  GetMem(3Fh)、清成 0，`+2Eh = 14h`、`+30h = 14h`、`+31h = F3h`、`+32h = 1`、`+3Dh = 17h`、
//	       `+3Eh = 89h`；010Ah:007Ah（entry 18 `164Bh`：複製一份接在串列尾端）、FreeMem；
//	       印 "Gains an item"（`07E8h`）、停一下
//	0916h  010Ah:0043h（overlay-25 entry 7：重算戰鬥數值）
//
// `19D1h..19E5h` 在 `08BCh` 之後以模式 0、節點 NULL 對 `DS:6B89h`（表上第一格，模式 0 的表就是
// 施法者）叫它一次；節點的 `+4` 是 1（`19B5h` 推的 `[bp+0Eh]`），到期時 entry 2 以模式 1 再叫。

const (
	// SpiritualHammerEffectCode 是靈魂鎚的參數表 `+0Ah`。
	SpiritualHammerEffectCode uint8 = 0x17
	// SpiritualHammerItemLimit 是 `087Dh` `26 80 BD C7 00 10 / 72 03`：身上不到 16 件才給。
	SpiritualHammerItemLimit = 0x10

	spiritualHammerType     = 0x14 // `+2Eh`
	spiritualHammerNameWord = 0xf3 // `+31h`
)

// spiritualHammerBytes 是 `08AAh..08D2h` 寫的六格（其餘 0）。
var spiritualHammerBytes = [...]struct{ offset, value uint8 }{
	{0x2e, spiritualHammerType}, {0x30, 0x14}, {0x31, spiritualHammerNameWord},
	{0x32, 0x01}, {0x3d, 0x17}, {0x3e, 0x89},
}

// SpiritualHammerItem 是 `0888h..08D2h` 造出來的那一件（63 bytes，名字那 2Ah bytes 是 0，
// 由呼叫端依名稱字詞重組）。
func SpiritualHammerItem() []byte {
	raw := make([]byte, MonsterItemRecordSize)
	for _, field := range spiritualHammerBytes {
		raw[field.offset] = field.value
	}
	return raw
}

// IsSpiritualHammer 是 `0821h..0830h` 的比對：`+2Eh == 14h` 而且 `+31h == F3h`。
func IsSpiritualHammer(raw []byte) bool {
	return len(raw) > spiritualHammerNameWordOffset &&
		raw[spiritualHammerTypeOffset] == spiritualHammerType &&
		raw[spiritualHammerNameWordOffset] == spiritualHammerNameWord
}

const (
	spiritualHammerTypeOffset     = 0x2e
	spiritualHammerNameWordOffset = 0x31
)
