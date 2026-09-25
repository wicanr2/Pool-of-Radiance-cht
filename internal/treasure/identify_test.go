package treasure

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/dax"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

const identifyDOSZIP = "../../Pool of Radiance (1988).zip"

func loadItemNameTable(t *testing.T) gamepack.ItemNameTable {
	t.Helper()
	table, err := gamepack.ReadDOSItemNameTable(identifyDOSZIP)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	return table
}

// authoredItemNames 是數量為 0、而內嵌名稱與重組結果不同的三筆：名稱是作者手寫的，
// 不是這支組出來的（字詞表寫 `+3 vs Undead`、記錄寫 `vs.`；卷軸寫的是法術名；
// 項鍊多一個前導空白）。
var authoredItemNames = map[string]string{
	"Two-Handed Sword +1 +3 vs. Undead":     "Two-Handed Sword +1 +3 vs Undead",
	"Clerical Scroll of Restoring Level(s)": "Clerical Scroll With 2 Spells",
	" Necklace":                             "Necklace",
}

// 正對照：ITEM*.DAX 每一筆記錄的 `+0` 帶著一個名稱。數量為 0 的記錄，名稱就是
// 看得見的字詞依 `+31h`、`+30h`、`+2Fh` 排起來——拿同一筆記錄的 `+2Eh..+35h`
// 重組一次，除了三筆作者手寫的之外必須逐字元相同（重組結果尾端多一個空白，
// 原版每段後面都接空白）。這一條驗了字詞表的位址與 stride、三段的順序與
// `+35h` 的位元對應：87 筆有藏字的記錄在裡面。
//
// 數量大於 0 的記錄不比：內嵌名稱是手寫的 `10 Arrow(s)`、`3 Potion`，而這支在
// 執行時組出的是 `10 Arrows `、`3 Potions `——那是原版顯示時的名稱，不是資料錯。
func TestItemNameRebuildsTheEmbeddedNames(t *testing.T) {
	table := loadItemNameTable(t)
	archive, err := zip.OpenReader(identifyDOSZIP)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	defer archive.Close()
	checked, hidden, authored := 0, 0, 0
	for _, member := range archive.File {
		base := strings.ToUpper(filepath.Base(member.Name))
		if !strings.HasPrefix(base, "ITEM") || !strings.HasSuffix(base, ".DAX") {
			continue
		}
		reader, err := member.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		for _, block := range blocks {
			for offset := 0; offset+0x3f <= len(block.Data); offset += 0x3f {
				raw := block.Data[offset : offset+0x3f]
				if raw[0x39] != 0 {
					continue
				}
				embedded := string(raw[1 : 1+int(raw[0])])
				got, err := ItemName(raw, &table)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasSuffix(got, " ") {
					t.Fatalf("rebuilt name %q lacks the trailing blank every part gets", got)
				}
				got = strings.TrimSuffix(got, " ")
				want := embedded
				if rebuilt, ok := authoredItemNames[embedded]; ok {
					want = rebuilt
					authored++
				}
				if got != want {
					t.Errorf("%s/%02Xh#%d: rebuilt %q, embedded %q (type %02X parts %02X %02X %02X hidden %02X)",
						base, block.Entry.ID, offset/0x3f, got, embedded,
						raw[0x2e], raw[0x2f], raw[0x30], raw[0x31], raw[0x35])
				}
				checked++
				if raw[0x35] != 0 {
					hidden++
				}
			}
		}
	}
	if checked < 300 || hidden < 80 || authored != 4 {
		t.Fatalf("positive control drifted: %d records, %d with hidden parts, %d authored", checked, hidden, authored)
	}
	t.Logf("%d records with count 0 rebuilt, %d with hidden parts, %d authored", checked, hidden, authored)
}

// 數量 ≥ 2 的 `s `：只有一段看得見就接在那一段（`10 Arrows `）；三段都看得見
// 而且型別不是 56h，接在 `+2Fh` 那一段——`Boots` 也照接，原版就是 `Bootss`。
func TestItemNamePluralFollowsTheOverlayBranches(t *testing.T) {
	table := loadItemNameTable(t)
	for _, tc := range []struct {
		itemType, first, second, third, count byte
		want                                  string
	}{
		{0x49, 0, 0, 0x3d, 10, "10 Arrows "},
		{0x44, 0x4d, 0xe7, 0x60, 25, "25 Dust Covered Bootss "},
		{0x48, 0, 0x82, 0xe6, 6, "6 Rotting Rugs "},
		{0x49, 0, 0x3d, 0xb1, 20, "20 Silver Arrows "},
		{0x56, 0xac, 0xa7, 0x63, 1, "1 Flask of Oil "},
	} {
		item := identifyItem("x", tc.itemType, tc.first, tc.second, tc.third, 0, tc.count)
		got, err := ItemName(item.Raw, &table)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("ItemName(%02X %02X %02X %02X ×%d) = %q, want %q",
				tc.itemType, tc.first, tc.second, tc.third, tc.count, got, tc.want)
		}
	}
}

func identifyItem(name string, itemType, first, second, third, hidden, count byte) poolsave.Item {
	item := saleItem(name, itemType, count, 10, 100)
	item.Raw[0x2f], item.Raw[0x30], item.Raw[0x31] = first, second, third
	item.Raw[0x35] = hidden
	return item
}

// Long Sword +1：+2Fh = A2h「+1」被 `+35h` 位元 2 藏著，鑑定之後名稱多出它。
func TestIdentifyRevealsTheHiddenPartAndCharges200(t *testing.T) {
	table := loadItemNameTable(t)
	state := &poolsave.State{Party: []poolsave.Character{{Name: "ARIA"}}}
	state.Party[0].Money[Platinum] = 50 // 250 金
	state.Party[0].Inventory = []poolsave.Item{identifyItem("Long Sword ", 0x24, 0xa2, 0, 0x24, 0x04, 0)}
	state.CharacterLibrary = []poolsave.Character{state.Party[0]}
	result, err := IdentifyItem(state, 0, 0, &table)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != IdentifyRevealed || result.PaidBy != PaidByCharacter {
		t.Fatalf("result = %+v", result)
	}
	item := state.Party[0].Inventory[0]
	if item.Name != "Long Sword +1 " || result.Name != "Long Sword +1" || item.Raw[0x35] != 0 ||
		string(item.Raw[1:1+item.Raw[0]]) != item.Name {
		t.Fatalf("item after identify = %q raw name %q hidden %02X", item.Name, item.Raw[1:1+item.Raw[0]], item.Raw[0x35])
	}
	// 250 − 200 = 50 金，重鑄成 10 白金。
	if state.Party[0].Money[Platinum] != 10 || state.Party[0].Money[Gold] != 0 {
		t.Fatalf("money = %v", state.Party[0].Money)
	}
	if state.CharacterLibrary[0].Inventory[0].Name != item.Name || state.CharacterLibrary[0].Money != state.Party[0].Money {
		t.Fatal("character library did not follow")
	}
}

func TestIdentifyWithoutHiddenPartsStillCharges(t *testing.T) {
	table := loadItemNameTable(t)
	state := &poolsave.State{Party: []poolsave.Character{{Name: "ARIA"}}}
	state.Party[0].Money[Gold] = 200
	state.Party[0].Inventory = []poolsave.Item{identifyItem("Long Sword ", 0x24, 0, 0, 0x24, 0, 0)}
	result, err := IdentifyItem(state, 0, 0, &table)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != IdentifyNothingNew || state.Party[0].Money != ([CurrencyCount]uint16{}) {
		t.Fatalf("result = %+v money %v", result, state.Party[0].Money)
	}
}

func TestIdentifyFallsBackToThePoolAndRefusesWhenBothAreShort(t *testing.T) {
	table := loadItemNameTable(t)
	state := &poolsave.State{Party: []poolsave.Character{{Name: "ARIA"}}}
	state.Party[0].Money[Gold] = 199
	state.PooledMoney[Platinum] = 40
	state.Party[0].Inventory = []poolsave.Item{identifyItem("Long Sword ", 0x24, 0xa2, 0, 0x24, 0x04, 0)}
	result, err := IdentifyItem(state, 0, 0, &table)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != IdentifyRevealed || result.PaidBy != PaidByPool || state.PooledMoney[Platinum] != 0 ||
		state.Party[0].Money[Gold] != 199 {
		t.Fatalf("pool payment: result %+v pool %v money %v", result, state.PooledMoney, state.Party[0].Money)
	}
	state.Party[0].Inventory[0].Raw[0x35] = 0x04
	before := state.Party[0].Inventory[0].Name
	result, err = IdentifyItem(state, 0, 0, &table)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != IdentifyNotEnoughMoney || state.Party[0].Inventory[0].Raw[0x35] != 0x04 ||
		state.Party[0].Inventory[0].Name != before || state.Party[0].Money[Gold] != 199 {
		t.Fatalf("refusal changed state: %+v", result)
	}
}
