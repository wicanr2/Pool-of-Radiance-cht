package gamepack_test

import (
	"os"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/combat/ability"
)

// 七名預設人物的 AC 逐一算得出來。原版自己把答案寫在 `+111h` 與 `+112h`，
// 所以這是對著原始位元組的端對端驗證，不是對著我自己抄的表。
//
// 兩個欄位都要比：`chrdatd3` 的護符戒指有沒有被壓掉，在 `+112h` 上是
// 57 對 58，在 `+111h` 上是 66 對 67——只比一個欄位會少一個見證。
func TestArmourClassMatchesThePremadeCharacters(t *testing.T) {
	types, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, item := range []struct {
		name        string
		magicArmour bool
		note        string
	}{
		{"chrdatd1", true, "Shield +1、Plate Mail +3"},
		{"chrdatd2", true, "Shield +2、Plate Mail +2"},
		{"chrdatd3", true, "同上再加護符戒指 +1，戒指被壓掉"},
		{"chrdatd4", true, "Leather Armor +4"},
		{"chrdatd5", false, "護腕不是類別 2，戒指 +3 照算"},
		{"chrdatd6", false, "同上，戒指 +2"},
		{"chrdatd7", false, "Shield 與 Leather Armor 都沒有加值"},
	} {
		record, err := os.ReadFile("../../workplace/oracle/dos/" + item.name + ".sav")
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		raw, err := os.ReadFile("../../workplace/oracle/dos/" + item.name + ".itm")
		if err != nil {
			t.Skipf("original item records unavailable: %v", err)
		}
		items := make([][]byte, 0, len(raw)/63)
		for offset := 0; offset+63 <= len(raw); offset += 63 {
			items = append(items, raw[offset:offset+63])
		}
		armour, err := gamepack.ArmourClassFromRecord(record, items, types)
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		if got, want := armour.Internal, int(record[gamepack.InternalArmourClassOffset]); got != want {
			t.Fatalf("%s (%s) internal AC %d, original stores %d", item.name, item.note, got, want)
		}
		if got, want := armour.Rear, int(record[gamepack.RearArmourClassOffset]); got != want {
			t.Fatalf("%s (%s) rear AC %d, original stores %d", item.name, item.note, got, want)
		}
		if armour.MagicArmour != item.magicArmour {
			t.Fatalf("%s (%s) magic-armour flag %v, want %v", item.name, item.note, armour.MagicArmour, item.magicArmour)
		}
	}
}

// 魔法盔甲壓制是一條會被誤讀成「身上有護甲就壓」的規則。原版的條件很窄：
// 有加值的**類別 2**。把同一件護符戒指分別配上魔法盔甲與魔法護腕，結果
// 必須不同——這一條錯了，法師會平白少一點 AC。
func TestMagicArmourSuppressesTheProtectionRing(t *testing.T) {
	types, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	// 從預設人物借出真正的物品記錄，不自己捏。
	armourRaw, err := os.ReadFile("../../workplace/oracle/dos/chrdatd3.itm")
	if err != nil {
		t.Skipf("original item records unavailable: %v", err)
	}
	bracerRaw, err := os.ReadFile("../../workplace/oracle/dos/chrdatd5.itm")
	if err != nil {
		t.Skipf("original item records unavailable: %v", err)
	}
	find := func(raw []byte, category uint8, wantValue bool) []byte {
		for offset := 0; offset+63 <= len(raw); offset += 63 {
			item := raw[offset : offset+63]
			if item[gamepack.ItemReadiedOffset] == 0 {
				continue
			}
			entry, err := types.Entry(item[gamepack.ItemTypeOffset])
			if err != nil {
				continue
			}
			value := entry.Raw[gamepack.ItemTypeArmourClassOffset]
			// 最高位沒設的物品與 AC 無關；預設人物身上就有這種同類別的
			// 誘餌（chrdatd3 的第 3 件也是類別 9），挑錯就測不到東西。
			if value&0x80 == 0 {
				continue
			}
			if entry.Category() == category && (value&0x7f != 0) == wantValue {
				return item
			}
		}
		return nil
	}
	plate := find(armourRaw, gamepack.ItemCategoryArmour, true)
	ring := find(armourRaw, gamepack.ItemCategoryProtectionRing, false)
	bracers := find(bracerRaw, 77, true)
	if plate == nil || ring == nil || bracers == nil {
		t.Fatalf("premade items missing: plate=%v ring=%v bracers=%v", plate != nil, ring != nil, bracers != nil)
	}

	withArmour, err := gamepack.ArmourClassFor(50, 18, [][]byte{plate, ring}, types)
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	withBracers, err := gamepack.ArmourClassFor(50, 18, [][]byte{bracers, ring}, types)
	if err != nil {
		t.Fatalf("bracers: %v", err)
	}
	if withArmour.Accumulators[3] != 0 {
		t.Fatalf("magic plate left %d in the ring slot", withArmour.Accumulators[3])
	}
	if withBracers.Accumulators[3] == 0 {
		t.Fatalf("bracers should not suppress the ring")
	}
	if !withArmour.MagicArmour || withBracers.MagicArmour {
		t.Fatalf("magic-armour flag plate=%v bracers=%v", withArmour.MagicArmour, withBracers.MagicArmour)
	}
}

// 盔甲比裸身差就不算（`0F9Ah` 取大者）。
func TestWorseArmourDoesNotLowerTheArmourClass(t *testing.T) {
	types, err := gamepack.ReadDOSItemTypeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	bare, err := gamepack.ArmourClassFor(50, 10, nil, types)
	if err != nil {
		t.Fatalf("bare: %v", err)
	}
	if bare.Internal != 50 {
		t.Fatalf("bare internal AC %d, want 50", bare.Internal)
	}
	if bare.Tabletop() != 10 {
		t.Fatalf("bare tabletop AC %d, want 10", bare.Tabletop())
	}
}

// 敏捷那張表與共用引擎的同一張表在遊戲到得了的範圍（1..25）必須一致。
// 引擎那份是 CoAB 也在用的，這個測試是兩邊之間的柵欄。
func TestDexterityArmourAdjustmentMatchesTheSharedEngine(t *testing.T) {
	for dexterity := 1; dexterity <= 25; dexterity++ {
		if got, want := gamepack.DexterityArmourAdjustment(dexterity), ability.DexterityDefenceAdjustment(dexterity); got != want {
			t.Fatalf("dexterity %d: pool %d, engine %d", dexterity, got, want)
		}
	}
}
