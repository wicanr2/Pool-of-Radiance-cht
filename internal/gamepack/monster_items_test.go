package gamepack

import (
	"bytes"
	"path/filepath"
	"testing"
)

// 八個 MONnITM.DAX 每一個 block 都是 3Fh 的整數倍，對到 MONnCHA 的 172 筆總共 301 件
// （overlay-17 `1178h..11F7h` 的步長，spec 142）。
func TestRealMonsterItemChainsAreWholeRecords(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := ReadDOSMonsterRecord(zipPath, 2, 13); err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	records, items, carriers := 0, 0, 0
	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			if _, err := ReadDOSMonsterRecord(zipPath, archive, uint8(id)); err != nil {
				continue
			}
			records++
			chain, err := ReadDOSMonsterItems(zipPath, archive, uint8(id))
			if err != nil {
				t.Fatalf("MON%dITM block %d: %v", archive, id, err)
			}
			if len(chain) != 0 {
				carriers++
			}
			for index, item := range chain {
				if len(item) != MonsterItemRecordSize {
					t.Fatalf("MON%dITM block %d item %d has %d bytes", archive, id, index, len(item))
				}
				if !bytes.Equal(item[0x2A:0x2E], []byte{0, 0, 0, 0}) {
					t.Fatalf("MON%dITM block %d item %d keeps a next pointer % X", archive, id, index, item[0x2A:0x2E])
				}
			}
			items += len(chain)
		}
	}
	if records != 172 || items != 301 {
		t.Fatalf("%d records carry %d items, want 172 and 301", records, items)
	}
	if carriers == 0 || carriers == records {
		t.Fatalf("%d of %d records carry items; the anchor lost its negative control", carriers, records)
	}
}

// 真檔錨點：MON2 block 1（KOBOLD LEADER）五件，順序照檔案；block 94（LEVEL 3 MU）
// 那支杖值 35000、穿戴中、法術 58h（大於 38h，entry 3 換成 41h）。
func TestRealKoboldLeaderAndMagicUserItems(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	kobold, err := ReadDOSMonsterItems(zipPath, 2, 1)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	wantTypes := []byte{0x2C, 0x49, 0x25, 0x3B, 0x34}
	if len(kobold) != len(wantTypes) {
		t.Fatalf("KOBOLD LEADER carries %d items, want %d", len(kobold), len(wantTypes))
	}
	for index, want := range wantTypes {
		if got := kobold[index][ItemTypeOffset]; got != want {
			t.Fatalf("item %d type %02X, want %02X", index, got, want)
		}
	}
	if kobold[1][ItemCountOffset] != 20 {
		t.Fatalf("arrows count %d, want 20", kobold[1][ItemCountOffset])
	}
	wizard, err := ReadDOSMonsterItems(zipPath, 2, 94)
	if err != nil {
		t.Fatal(err)
	}
	if len(wizard) != 1 {
		t.Fatalf("LEVEL 3 MU carries %d items, want 1", len(wizard))
	}
	wand := wizard[0]
	value := int(wand[0x3A]) | int(wand[0x3B])<<8
	if value != 35000 || wand[ItemReadiedOffset] != 1 || wand[AIItemSpellOffset] != 0x58 {
		t.Fatalf("wand value %d readied %d spell %02X", value, wand[ItemReadiedOffset], wand[AIItemSpellOffset])
	}
	if spell, ok := AIItemSpell(wand, false); !ok || spell != 0x41 {
		t.Fatalf("entry 3 reads spell %02X (%v), want 41h", spell, ok)
	}
	// 沒有 ITM block 的怪物是空串列，不是錯誤。
	none, err := ReadDOSMonsterItems(zipPath, 2, 4)
	if err != nil || len(none) != 0 {
		t.Fatalf("ORC items %d (%v), want none", len(none), err)
	}
}

// 長度不是 3Fh 的整數倍就失敗即關閉；下一節指標清成 0；空 block 是零件。
func TestParseMonsterItemsBounds(t *testing.T) {
	if _, err := ParseMonsterItems(make([]byte, MonsterItemRecordSize-1)); err == nil {
		t.Fatal("a short item block was accepted")
	}
	if _, err := ParseMonsterItems(make([]byte, MonsterItemRecordSize*2+1)); err == nil {
		t.Fatal("a block with a trailing byte was accepted")
	}
	empty, err := ParseMonsterItems(nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty block → %d items, %v", len(empty), err)
	}
	payload := make([]byte, MonsterItemRecordSize*2)
	for index := range payload {
		payload[index] = byte(index)
	}
	items, err := ParseMonsterItems(payload)
	if err != nil || len(items) != 2 {
		t.Fatalf("two records → %d items, %v", len(items), err)
	}
	if items[1][0] != MonsterItemRecordSize || items[1][0x3E] != byte(MonsterItemRecordSize*2-1) {
		t.Fatalf("second record starts at %d, ends with %d", items[1][0], items[1][0x3E])
	}
	if !bytes.Equal(items[0][0x2A:0x2E], []byte{0, 0, 0, 0}) {
		t.Fatalf("next pointer kept: % X", items[0][0x2A:0x2E])
	}
	payload[0] = 0xEE
	if items[0][0] == 0xEE {
		t.Fatal("items alias the payload")
	}
}
