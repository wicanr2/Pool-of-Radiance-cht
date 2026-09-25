package treasure

import (
	"encoding/binary"
	"testing"

	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func saleItem(name string, itemType, count uint8, weight, value uint16) poolsave.Item {
	raw := make([]byte, poolcharacter.ItemRecordSize)
	raw[0] = byte(len(name))
	copy(raw[1:], name)
	raw[0x2e] = itemType
	raw[0x39] = count
	binary.LittleEndian.PutUint16(raw[0x37:], weight)
	binary.LittleEndian.PutUint16(raw[0x3a:], value)
	return poolsave.Item{Name: name, Raw: raw}
}

// 出價是 `+3Ah` 的一半（`1D02h..1D0Dh`），整數除法；一疊以上再乘數量除以 20。
func TestSellOfferFollowsTheOverlayArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		typ   uint8
		count uint8
		value uint16
		want  uint16
	}{
		{"Long Sword", 0x24, 0, 15, 7},
		{"Partisan", 0x19, 0, 10, 5},
		{"Hand Axe", 0x02, 0, 1, 0},
		{"Plate Mail", 0x3a, 0, 400, 200},
		{"nothing", 0x02, 0, 0, 0},
		{"count one is not a stack", 0x02, 1, 40, 20},
		// 數量 10、值 40：20 × 10 ÷ 20 = 10。
		{"10 Arrow(s)", 0x49, 10, 40, 10},
		// `1D1Dh` 的不除分支要 `+2Eh` 同時等於 49h 與 1Ch，走不到：弩矢照樣除以 20。
		{"20 Quarrel(s)", 0x1c, 20, 40, 20},
		// `mul` 之後丟掉 DX：30000 × 3 = 90000 → 低 16 位 24464 → ÷ 20 = 1223。
		{"wrap", 0x02, 3, 60000, 1223},
	} {
		got, err := SellOffer(saleItem(tc.name, tc.typ, tc.count, 1, tc.value).Raw)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%s: offer %d, want %d", tc.name, got, tc.want)
		}
	}
	if _, err := SellOffer(make([]byte, 10)); err == nil {
		t.Fatal("a short item record produced an offer")
	}
}

func TestSellGatesReadiedItemsAndMarkedScrolls(t *testing.T) {
	item := saleItem("Long Sword", 0x24, 0, 60, 15)
	if SellNeedsUnready(item.Raw) {
		t.Fatal("an unreadied item needs unreadying")
	}
	item.Raw[0x34] = 1
	if !SellNeedsUnready(item.Raw) {
		t.Fatal("a readied item can be sold")
	}
	scroll := saleItem("Scroll", 0x50, 0, 5, 300)
	scroll.Raw[0x3c] = 0x05
	if SellNeedsScribeConfirm(scroll.Raw, 0x0c) {
		t.Fatal("an unmarked scroll asks about scribing")
	}
	scroll.Raw[0x3d] = 0x85
	for category, want := range map[uint8]bool{0x0a: false, 0x0b: true, 0x0c: true, 0x0d: true, 0x0e: false} {
		if got := SellNeedsScribeConfirm(scroll.Raw, category); got != want {
			t.Errorf("category %02Xh: scribe confirm %v, want %v", category, got, want)
		}
	}
}

func saleState(strength int, money [CurrencyCount]uint16, items ...poolsave.Item) poolsave.State {
	member := poolsave.Character{Name: "HERO", Abilities: [6]int{strength, 10, 10, 10, 10, 10},
		Money: money, Inventory: items}
	return poolsave.State{Party: []poolsave.Character{member}, CharacterLibrary: []poolsave.Character{member}}
}

// 白金 = 價 div 5、金 = 價 mod 5，直接加進錢包；不重鑄、不動其他幣別。
func TestSellItemPaysPlatinumAndGold(t *testing.T) {
	state := saleState(14, [CurrencyCount]uint16{Copper: 3, Silver: 4, Platinum: 10},
		saleItem("Dagger", 0x08, 0, 10, 2), saleItem("Long Sword", 0x24, 0, 60, 15))
	result, err := SellItem(&state, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Price != 7 || result.Platinum != 1 || result.Gold != 2 || result.Overloaded || result.PooledPlatinum != 0 {
		t.Fatalf("result %+v, want 7 = 1 platinum + 2 gold", result)
	}
	want := [CurrencyCount]uint16{Copper: 3, Silver: 4, Gold: 2, Platinum: 11}
	member := state.Party[0]
	if member.Money != want {
		t.Fatalf("money %v, want %v", member.Money, want)
	}
	if len(member.Inventory) != 1 || member.Inventory[0].Name != "Dagger" {
		t.Fatalf("inventory %+v, want only the dagger", member.Inventory)
	}
	if library := state.CharacterLibrary[0]; library.Money != want || len(library.Inventory) != 1 {
		t.Fatalf("library copy %+v did not follow the sale", library)
	}
}

// 超重：比的是**含那件物品**的舊負重（entry 17 摘物品不重算 `+102h`）。
// 力量 14 上限 1700；長劍 60 + 銅 1638 = 1698，賣 15 金的東西換 3 白金 → 1701 > 1700。
// 可容納 1700 − 1698 = 2 → 角色拿 2 白金，1 白金進公款；金一律給角色（這裡 0）。
func TestSellItemOverloadSendsPlatinumToThePool(t *testing.T) {
	capacity, err := poolcharacter.CarryCapacity(14, 0)
	if err != nil || capacity != 1700 {
		t.Fatalf("capacity %d (%v), want 1700", capacity, err)
	}
	state := saleState(14, [CurrencyCount]uint16{Copper: 1638},
		saleItem("Two-Handed Sword", 0x26, 0, 60, 30))
	state.PooledMoney[Platinum] = 5
	result, err := SellItem(&state, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Overloaded || result.Platinum != 2 || result.PooledPlatinum != 1 || result.Gold != 0 {
		t.Fatalf("result %+v, want overloaded 2 + 1 pooled platinum", result)
	}
	if got := state.Party[0].Money[Platinum]; got != 2 {
		t.Fatalf("character platinum %d, want 2", got)
	}
	if got := state.PooledMoney[Platinum]; got != 6 {
		t.Fatalf("pooled platinum %d, want 5 + 1", got)
	}
}

// 沒超過上限就全給角色：長劍（值 13）60 + 銅 1637 = 1697，出價 6 → 1 白金 1 金，
// 合計 2 枚 → 1699，不超過 1700。
func TestSellItemBelowTheLimitIsNotOverloaded(t *testing.T) {
	state := saleState(14, [CurrencyCount]uint16{Copper: 1637}, saleItem("Long Sword", 0x24, 0, 60, 13))
	result, err := SellItem(&state, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Overloaded || state.Party[0].Money[Platinum] != 1 || state.Party[0].Money[Gold] != 1 {
		t.Fatalf("result %+v money %v, want 1 platinum 1 gold without overload", result, state.Party[0].Money)
	}
}

func TestSellItemRefusesReadiedItemsAndBadIndexes(t *testing.T) {
	item := saleItem("Long Sword", 0x24, 0, 60, 15)
	item.Raw[0x34] = 1
	state := saleState(14, [CurrencyCount]uint16{}, item)
	if _, err := SellItem(&state, 0, 0); err == nil {
		t.Fatal("a readied item was sold")
	}
	if len(state.Party[0].Inventory) != 1 || state.Party[0].Money != [CurrencyCount]uint16{} {
		t.Fatal("a refused sale changed the character")
	}
	if _, err := SellItem(&state, 0, 3); err == nil {
		t.Fatal("an absent item was sold")
	}
	if _, err := SellItem(&state, 2, 0); err == nil {
		t.Fatal("an absent party member sold something")
	}
}
