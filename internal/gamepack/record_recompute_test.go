package gamepack_test

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 七名預設人物的記錄裡本來就存著重算的結果。把那些欄位清成 0 再重算一次，
// 每一格都必須回到原本的值——這是整條 `0E36h` 管線的端對端驗證：
// 職業等級查 THAC0 表、武器加值、力量與敏捷修正、負重、護甲五個槽、
// 盔甲與負重壓移動力，任一處錯了就對不上。
func TestRecomputeCombatFieldsMatchesThePremadeCharacters(t *testing.T) {
	types, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	derived := []struct {
		offset int
		name   string
	}{
		{gamepack.BaseThac0Offset, "+2Dh 基礎 THAC0"},
		{gamepack.CurrentThac0Offset, "+110h THAC0"},
		{gamepack.InternalArmourClassOffset, "+111h AC"},
		{gamepack.RearArmourClassOffset, "+112h 背面 AC"},
		{gamepack.DamageBonusOffset, "+119h 傷害加值"},
		{gamepack.CurrentMovementOffset, "+11Ch 移動力"},
	}
	// 傷害骰只有備妥武器時才由這一支寫；沒有武器時原版直接返回。
	dice := []struct {
		offset int
		name   string
	}{
		{gamepack.DamageDiceCountOffset, "+115h 傷害骰數"},
		{gamepack.DamageDieSidesOffset, "+117h 傷害骰面"},
	}
	for _, item := range []struct {
		name   string
		weapon bool
	}{
		{"chrdatd1", true}, {"chrdatd2", true}, {"chrdatd3", true}, {"chrdatd4", true},
		// TARRY 身上的 Quarter Staff +1 沒有備妥，所以走的是「沒有武器」那條路。
		// 原版記錄裡的 `+115h`／`+117h` 是 1／2，那不是這一支寫的——它從哪來
		// 還沒讀出來，所以這裡只驗證重算沒有去碰它們。
		{"chrdatd5", false},
		{"chrdatd6", true}, {"chrdatd7", true},
	} {
		name := item.name
		record, err := os.ReadFile("../../workplace/oracle/dos/" + name + ".sav")
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		items, err := readItemChain("../../workplace/oracle/dos/" + name + ".itm")
		if err != nil {
			t.Skipf("original item records unavailable: %v", err)
		}
		cleared := append([]byte(nil), record...)
		for _, field := range derived {
			cleared[field.offset] = 0
		}
		for _, field := range dice {
			cleared[field.offset] = 0
		}
		binary.LittleEndian.PutUint16(cleared[gamepack.CarriedWeightOffset:], 0)

		result, err := gamepack.RecomputeCombatFields(cleared, items, types)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, field := range derived {
			if result[field.offset] != record[field.offset] {
				t.Fatalf("%s 的 %s 重算成 %d，原本是 %d",
					name, field.name, result[field.offset], record[field.offset])
			}
		}
		for _, field := range dice {
			want := record[field.offset]
			if !item.weapon {
				want = 0 // 清成 0 之後不該被寫回去
			}
			if result[field.offset] != want {
				t.Fatalf("%s 的 %s 重算成 %d，預期 %d",
					name, field.name, result[field.offset], want)
			}
		}
		want := binary.LittleEndian.Uint16(record[gamepack.CarriedWeightOffset:])
		if got := binary.LittleEndian.Uint16(result[gamepack.CarriedWeightOffset:]); got != want {
			t.Fatalf("%s 的 +102h 負重重算成 %d，原本是 %d", name, got, want)
		}
		// 只有衍生欄位會變：其餘 285 bytes 一個都不能動。
		for offset := range record {
			if result[offset] != cleared[offset] && !isDerivedOffset(offset) {
				t.Fatalf("%s 的 +%02Xh 不該被重算改到", name, offset)
			}
		}
	}
}

// 沒有備妥武器時傷害三欄維持原值——原版那支直接返回，不補徒手傷害
//（spec 063 契約第 5 條）。
func TestRecomputeCombatFieldsLeavesDamageAloneWithoutAWeapon(t *testing.T) {
	types, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	record, err := os.ReadFile("../../workplace/oracle/dos/chrdatd1.sav")
	if err != nil {
		t.Skipf("original character records unavailable: %v", err)
	}
	result, err := gamepack.RecomputeCombatFields(record, nil, types)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []struct {
		offset int
		name   string
	}{
		{gamepack.DamageDiceCountOffset, "+115h"},
		{gamepack.DamageDieSidesOffset, "+117h"},
	} {
		if result[field.offset] != record[field.offset] {
			t.Fatalf("沒有武器時 %s 不該改變", field.name)
		}
	}
	// 命中與傷害則只剩下能力值那兩個加值。HAPLO 是 18/00：命中 +3、傷害 +6。
	index, err := gamepack.StrengthTableIndex(int(record[0x10]), int(record[0x16]))
	if err != nil {
		t.Fatal(err)
	}
	wantThac0 := int(result[gamepack.BaseThac0Offset]) + gamepack.StrengthHitAdjustment(index)
	if int(result[gamepack.CurrentThac0Offset]) != wantThac0 {
		t.Fatalf("沒有武器時 +110h 是 %d，預期 %d", result[gamepack.CurrentThac0Offset], wantThac0)
	}
	wantDamage := int(int8(record[gamepack.DamageBonusOffset])) + gamepack.StrengthDamageAdjustment(index)
	if int(int8(result[gamepack.DamageBonusOffset])) != wantDamage {
		t.Fatalf("沒有武器時 +119h 是 %d，預期 %d",
			int8(result[gamepack.DamageBonusOffset]), wantDamage)
	}
}

func isDerivedOffset(offset int) bool {
	switch offset {
	case gamepack.BaseThac0Offset, gamepack.CurrentThac0Offset,
		gamepack.InternalArmourClassOffset, gamepack.RearArmourClassOffset,
		gamepack.DamageDiceCountOffset, gamepack.DamageDieSidesOffset,
		gamepack.DamageBonusOffset, gamepack.CurrentMovementOffset,
		gamepack.CarriedWeightOffset, gamepack.CarriedWeightOffset + 1:
		return true
	}
	return false
}

func readItemChain(path string) ([][]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	items := make([][]byte, 0, len(raw)/63)
	for offset := 0; offset+63 <= len(raw); offset += 63 {
		items = append(items, raw[offset:offset+63])
	}
	return items, nil
}
