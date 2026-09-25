package treasure

import (
	"encoding/binary"
	"fmt"

	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 商店賣出物品（spec 067〈賣出〉）。原版的入口在商店選單的 V）iew →
// I）tems → S）ell：overlay-19 entry 6（`0EFBh`）的物品選單在 `DS:4954h == 1`
// （overlay-06 `0531h` 進店時寫的）才把 `Sell` 放進選項（`10EAh`），
// 按 S 先過 entry 20（`0D72h`）的放手檢查，再叫 entry 16（`1CE9h`）出價與付錢。

// 物品記錄（63 bytes，spec 033）裡賣出用到的欄位。
const (
	sellItemTypeOffset    = 0x2e
	sellItemReadiedOffset = 0x34
	sellItemCountOffset   = 0x39
	sellItemValueOffset   = 0x3a
	sellScrollSpellFirst  = 0x3c
	sellScrollSpellLast   = 0x3e
)

// SellOffer 重現 overlay-19 entry 16 的出價（`1CF0h..1D52h`）：
//
//	價 = 0；+3Ah > 0 → 價 = +3Ah div 2
//	+39h > 1 → 價 = (+39h × 價 取低 16 位) div 20
//
// `1D1Dh` 另有一條「`+2Eh == 49h` 而且 `+2Eh == 1Ch`」才走的不除分支——同一格不可能
// 同時等於兩個值，所以那一支永遠走不到，數量大於 1 一律除以 20。
func SellOffer(raw []byte) (uint16, error) {
	if len(raw) != poolcharacter.ItemRecordSize {
		return 0, fmt.Errorf("Pool item has %d bytes, want %d", len(raw), poolcharacter.ItemRecordSize)
	}
	price := uint16(0)
	if value := binary.LittleEndian.Uint16(raw[sellItemValueOffset:]); value > 0 {
		price = value / 2
	}
	if count := raw[sellItemCountOffset]; count > 1 {
		price = uint16(uint32(count)*uint32(price)) / 20
	}
	return price, nil
}

// SellNeedsUnready 是 entry 20 的第一道（`0D7Fh`）：穿戴中（`+34h` 非 0）就印
// `Must be unreadied`，不賣。被詛咒的東西卸不下來，所以它也就賣不掉——原版沒有
// 另外一條「詛咒」的檢查。
func SellNeedsUnready(raw []byte) bool {
	return len(raw) > sellItemReadiedOffset && raw[sellItemReadiedOffset] != 0
}

// ScrollCategoryFirst／ScrollCategoryLast 是 overlay-22 entry 6（`31F6h`）認卷軸的
// 範圍：物品型別表（`DS:54E0h + 型別 × 16`）第 0 格大於 0Ah 且小於 0Eh。
const (
	ScrollCategoryFirst uint8 = 0x0b
	ScrollCategoryLast  uint8 = 0x0d
)

// SellNeedsScribeConfirm 是 entry 20 的第二道（`0DA3h..0E42h`）：卷軸而且
// `+3Ch..+3Eh` 任一格的位元 7 立著（有人正要從它抄法術），先問
// `<名字> was going to scribe from that scroll` / `is it Okay to lose it?`，Y 才放手。
func SellNeedsScribeConfirm(raw []byte, category uint8) bool {
	if category < ScrollCategoryFirst || category > ScrollCategoryLast || len(raw) <= sellScrollSpellLast {
		return false
	}
	for offset := sellScrollSpellFirst; offset <= sellScrollSpellLast; offset++ {
		if raw[offset] > 0x7f {
			return true
		}
	}
	return false
}

// SaleResult 是一筆賣出實際發生的事。
type SaleResult struct {
	Price uint16
	// Platinum／Gold 是進角色錢包的枚數，PooledPlatinum 是塞不下、進隊伍公款的白金。
	Platinum, Gold, PooledPlatinum uint16
	// Overloaded 為真時原版在 `Sold!` 之後再印 `Overloaded.  Money will be put in pool.`。
	Overloaded bool
}

// SellItem 重現 entry 16 按 Y 之後的那一段（`1DE6h..1EACh`）：
//
//  1. 從角色的物品串列摘掉這一件（overlay-25 entry 17，`156Ah`）。
//  2. 白金 = 價 div 5、金 = 價 mod 5（`1E0Dh..1E25h`）。
//  3. overlay-21 entry 4（`0058h`）問 `+102h + (白金 + 金)` 有沒有超過負重上限
//     （entry 0：力量表 + 1500）。**`+102h` 這時還沒重算**——摘物品的 entry 17
//     不碰它，要等選單回圈的 overlay-25 entry 7（`146Fh`）——所以比的是含那件物品的舊負重。
//  4. 沒超過：白金加進 `+90h`、金加進 `+8Eh`。超過：可容納量 = 上限 − `+102h`；
//     它大於白金就全給角色，否則角色拿可容納量、其餘白金加進公款 `DS:6762h`；
//     金一律給角色。
//
// 錢欄是 16 位元的 `add`，溢位就繞回，與原版相同。
func SellItem(state *poolsave.State, partyIndex, itemIndex int) (SaleResult, error) {
	if state == nil || partyIndex < 0 || partyIndex >= len(state.Party) {
		return SaleResult{}, fmt.Errorf("Pool sale has no party member %d", partyIndex)
	}
	character := &state.Party[partyIndex]
	if itemIndex < 0 || itemIndex >= len(character.Inventory) {
		return SaleResult{}, fmt.Errorf("Pool sale has no item %d for %s", itemIndex, character.Name)
	}
	raw := character.Inventory[itemIndex].Raw
	price, err := SellOffer(raw)
	if err != nil {
		return SaleResult{}, err
	}
	if SellNeedsUnready(raw) {
		return SaleResult{}, fmt.Errorf("Pool sale of a readied item")
	}
	capacity, err := poolcharacter.CarryCapacity(character.Abilities[0], character.ExceptionalStrength)
	if err != nil {
		return SaleResult{}, err
	}
	load, err := carriedLoad(*character)
	if err != nil {
		return SaleResult{}, err
	}

	result := SaleResult{Price: price, Platinum: price / GoldPerPlatinum, Gold: price % GoldPerPlatinum}
	coins := result.Platinum + result.Gold
	if uint16(load)+coins > uint16(capacity) {
		result.Overloaded = true
		room := uint16(capacity) - uint16(load)
		if room <= result.Platinum {
			result.PooledPlatinum = result.Platinum - room
			result.Platinum = room
		}
	}

	character.Inventory = append(character.Inventory[:itemIndex:itemIndex], character.Inventory[itemIndex+1:]...)
	character.Money[Platinum] += result.Platinum
	character.Money[Gold] += result.Gold
	state.PooledMoney[Platinum] += uint32(result.PooledPlatinum)
	syncLibraryCharacter(state, *character)
	return result, nil
}

// carriedLoad 是記錄 `+102h` 的值：物品重量乘數量（數量 0 當 1）加上七欄錢的枚數
// （overlay-25 `0C17h`，spec 079；與 `gamepack.CarriedWeight` 同一條算式）。
func carriedLoad(character poolsave.Character) (int, error) {
	load := 0
	for _, amount := range character.Money {
		load += int(amount)
	}
	for index, item := range character.Inventory {
		weight, err := poolcharacter.ItemLoad(item.Raw)
		if err != nil {
			return 0, fmt.Errorf("inventory item %d: %w", index, err)
		}
		load += weight
	}
	return load, nil
}
