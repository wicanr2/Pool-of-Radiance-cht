package treasure

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 物品選單的 I）d（spec 067〈鑑定〉）。overlay-19 entry 17（`1F52h..2117h`），
// 只在商店裡出現（`1119h`：`DS:4954h == 1`，沒有角色條件）。

// IdentifyPrice 是 `1FEBh` 與 `2021h` 比的那個 `C8h`：200 金幣。
const IdentifyPrice = 200

// 物品記錄裡鑑定用到的欄位。
const (
	itemNameMax       = 0x28 // RTL `064Eh` 的長度上限：每次寫回名稱都截到 40
	itemTypeOffset    = 0x2e
	itemNamePartFirst = 0x2f // +2Fh／+30h／+31h 三段字詞編號，查 DS:10BBh
	itemHiddenOffset  = 0x35 // 三個位元各藏一段字詞；鑑定把整個 byte 清成 0
	itemCountOffset   = 0x39
)

// ItemName 重現 overlay-25 entry 1（`0441h..0753h`）組名稱的那一段，
// 不含兩個呈現用的前綴：
//
//   - 參數 `[bp+8]` 非 0 時的 ` Yes  `／` No   `（穿戴欄）；鑑定呼叫時傳 0。
//   - 隊伍有人帶著效果 5（`21DCh`）而物品 `+32h`／`+33h` 大於 0 或 `+36h` 非 0
//     時的 `* `。remake 還沒有接這個效果，這裡一律當成沒有。
//
// 其餘照位元組：數量大於 0 先放數字與空白；三段字詞由 `+31h` 往 `+2Fh` 排，
// `+35h` 的位元 0／1／2 分別藏住 `+31h`／`+30h`／`+2Fh`；每段後面接空白，
// 數量 ≥ 2 時有一段會改接 `s `（`066Bh..06D4h` 的判準）。
func ItemName(raw []byte, table *gamepack.ItemNameTable) (string, error) {
	if len(raw) <= itemCountOffset || table == nil {
		return "", fmt.Errorf("Pool item record has %d bytes, want at least %d", len(raw), itemCountOffset+1)
	}
	name := ""
	store := func(value string) {
		if len(value) > itemNameMax {
			value = value[:itemNameMax]
		}
		name = value
	}
	if count := raw[itemCountOffset]; count > 0 {
		store(fmt.Sprintf("%s%d ", name, count))
	}
	itemType := raw[itemTypeOffset]
	hidden := raw[itemHiddenOffset]
	// `0570h..05C6h`：看得見的段落組成遮罩，位元 i−1 對應 `+2Eh+i`。
	mask := 0
	for part := 1; part <= 3; part++ {
		if raw[itemTypeOffset+part] != 0 && (hidden>>(3-part))&1 == 0 {
			mask += 1 << (part - 1)
		}
	}
	plural := false
	for part := 3; part >= 1; part-- {
		if (mask>>(part-1))&1 == 0 {
			continue
		}
		store(name + table[raw[itemTypeOffset+part]])
		if raw[itemCountOffset] < 2 || plural {
			store(name + " ")
			continue
		}
		if itemNameTakesPlural(part, mask, itemType, raw[itemNamePartFirst+2]) {
			store(name + "s ")
			plural = true
			continue
		}
		store(name + " ")
	}
	return name, nil
}

// itemNameTakesPlural 是 `066Bh..06D4h` 的分支：哪一段接 `s `。
// 56h、49h、1Ch 是物品型別；B1h 是字詞表的 `Silver`。
func itemNameTakesPlural(part, mask int, itemType, lastPart byte) bool {
	switch {
	case 1<<(part-1) == mask:
		return true
	case part == 1 && mask > 4 && itemType != 0x56:
		return true
	case part == 2 && mask&1 == 0:
		return true
	case part == 3 && itemType == 0x56:
		return true
	case (itemType == 0x49 || itemType == 0x1c) && lastPart != 0xb1:
		return true
	}
	return false
}

// IdentifyOutcome 是按 Y 之後三種結果之一。
type IdentifyOutcome int

const (
	// IdentifyNotEnoughMoney：角色與公款都不到 200 金，印 `Not Enough Money`，什麼都不動。
	IdentifyNotEnoughMoney IdentifyOutcome = iota
	// IdentifyNothingNew：付了錢，但 `+35h` 本來就是 0——
	// `I can't tell anything new about your <名稱>`。錢照收。
	IdentifyNothingNew
	// IdentifyRevealed：付了錢，`+35h` 清成 0、名稱重組——
	// `It looks like some sort of <名稱>`。
	IdentifyRevealed
)

// IdentifyResult 說鑑定的結果、錢從哪裡出、物品現在叫什麼。
type IdentifyResult struct {
	Outcome IdentifyOutcome
	PaidBy  PaySource
	Name    string
}

// IdentifyItem 是按 Y 之後（`1FD1h..20F4h`）：
//
//  1. 付 200 金：先看角色（entry 11 `28A2h`）、不夠才看公款（overlay-21 entry 17），
//     付完重鑄成白金＋金（entry 15／16）——與購買同一條（`PayGold`）。都不夠就停。
//  2. 付了錢，`+35h` 是 0 → 沒有新東西可說，錢不退。
//  3. 否則 `+35h = 0`，overlay-25 entry 1 把名稱重組寫回記錄 `+0`。
//
// 物品其餘欄位不動。名稱寫回的是 Pascal 字串（長度 byte＋內容），長度之後的舊
// bytes 原版也不清，這裡同樣不清。
func IdentifyItem(state *poolsave.State, partyIndex, itemIndex int, table *gamepack.ItemNameTable) (IdentifyResult, error) {
	if state == nil || partyIndex < 0 || partyIndex >= len(state.Party) {
		return IdentifyResult{}, fmt.Errorf("Pool identify has no party member %d", partyIndex)
	}
	character := &state.Party[partyIndex]
	if itemIndex < 0 || itemIndex >= len(character.Inventory) {
		return IdentifyResult{}, fmt.Errorf("Pool identify has no item %d", itemIndex)
	}
	item := &character.Inventory[itemIndex]
	if len(item.Raw) <= itemCountOffset {
		return IdentifyResult{}, fmt.Errorf("Pool identify item %q has %d raw bytes", item.Name, len(item.Raw))
	}
	source, paid, err := PayGold(state, partyIndex, IdentifyPrice)
	if err != nil {
		return IdentifyResult{}, err
	}
	if !paid {
		return IdentifyResult{Outcome: IdentifyNotEnoughMoney, Name: item.Name}, nil
	}
	character = &state.Party[partyIndex]
	item = &character.Inventory[itemIndex]
	if item.Raw[itemHiddenOffset] == 0 {
		syncLibraryCharacter(state, *character)
		return IdentifyResult{Outcome: IdentifyNothingNew, PaidBy: source, Name: item.Name}, nil
	}
	raw := append([]byte(nil), item.Raw...)
	raw[itemHiddenOffset] = 0
	name, err := ItemName(raw, table)
	if err != nil {
		return IdentifyResult{}, err
	}
	if name == "" {
		return IdentifyResult{}, fmt.Errorf("Pool identify rebuilt an empty name for %q", item.Name)
	}
	raw[0] = byte(len(name))
	copy(raw[1:], name)
	// 名稱照原版帶著尾端那個空白。存檔要求 `Item.Name` 與記錄裡的 Pascal 字串
	// 一致（`save.validateItem`），所以兩邊同一份；畫面上印的時候再去掉。
	item.Raw, item.Name = raw, name
	syncLibraryCharacter(state, *character)
	return IdentifyResult{Outcome: IdentifyRevealed, PaidBy: source, Name: strings.TrimRight(name, " ")}, nil
}
