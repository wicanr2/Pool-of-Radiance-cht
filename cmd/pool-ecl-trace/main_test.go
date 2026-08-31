package main

import (
	"path/filepath"
	"testing"
)

func TestInitialECLTracePinsEntryAndLoadChain(t *testing.T) {
	r, err := trace(filepath.Join("..", "..", "Pool of Radiance (1988).zip"), 3, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	if r.CodeAddressBase != "0x9900" || len(r.EntryAddresses) != 5 || r.EntryAddresses[4] != "0x9AF2" {
		t.Fatalf("base/entries=%s/%v", r.CodeAddressBase, r.EntryAddresses)
	}
	foundFiles, foundPieces := false, false
	for _, ins := range r.Instructions {
		if ins.Opcode == 0x21 {
			foundFiles = true
		}
		if ins.Opcode == 0x37 {
			foundPieces = true
		}
	}
	if !foundFiles || !foundPieces {
		t.Fatalf("LOAD FILES/PIECES=%t/%t", foundFiles, foundPieces)
	}
	t.Logf("ECL3/block0 instructions=%d edges=%d block_sha256=%s", len(r.Instructions), len(r.Edges), r.BlockSHA256)
}
