package gamepack

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRealDOSGraveyardTreasureItemBlock(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	records, err := ReadDOSTreasureItemBlock(zipPath, 3, 0x33)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	wantNames := []string{
		"Clerical Scroll With 2 Spells",
		"Clerical Scroll With 2 Spells",
		"Clerical Scroll With 2 Spells",
		"Clerical Scroll With 2 Spells",
		"Two-Handed Sword +1 +3 vs. Undead",
	}
	names := make([]string, len(records))
	for index := range records {
		names[index] = records[index].Name
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("names=%q, want %q", names, wantNames)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(records[4].Raw[:])); got != "78dab94cb54b90e348d64a1f8131ff6342a74afb2655fac1b90b2ee7e35d94a0" {
		t.Fatalf("last record sha256=%s", got)
	}
}

func TestTreasureItemRecordParserFailsClosed(t *testing.T) {
	if _, err := parseTreasureItemRecords(make([]byte, 62)); err == nil {
		t.Fatal("short ITEM record accepted")
	}
	badName := make([]byte, treasureItemRecordSize)
	badName[0] = 41
	if _, err := parseTreasureItemRecords(badName); err == nil {
		t.Fatal("oversized ITEM name accepted")
	}
	if _, err := ReadDOSTreasureItemBlock(filepath.Join("..", "..", "Pool of Radiance (1988).zip"), 0, 0x33); err == nil {
		t.Fatal("invalid ITEM archive accepted")
	}
}
