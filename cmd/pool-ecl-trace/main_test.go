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
	foundFiles, foundPieces, foundCityHallContinue := false, false, false
	for _, ins := range r.Instructions {
		if ins.Opcode == 0x21 {
			foundFiles = true
		}
		if ins.Opcode == 0x37 {
			foundPieces = true
		}
		if ins.Address == "0xAF1C" && ins.MenuDestination == "0x9801" && len(ins.MenuOptions) == 1 && ins.MenuOptions[0] == "PRESS <RETURN> OR BUTTON TO CONTINUE" {
			foundCityHallContinue = true
		}
	}
	if !foundFiles || !foundPieces || !foundCityHallContinue {
		t.Fatalf("LOAD FILES/PIECES/City Hall menu=%t/%t/%t", foundFiles, foundPieces, foundCityHallContinue)
	}
	t.Logf("ECL3/block0 instructions=%d edges=%d block_sha256=%s", len(r.Instructions), len(r.Edges), r.BlockSHA256)
}
