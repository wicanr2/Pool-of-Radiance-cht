package gamepack

import (
	"path/filepath"
	"testing"
)

func TestDOSInitialPieceSetMatchesLoadPiecesEvidence(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	piece, err := ReadDOSPieceSet(zipPath, 3, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if piece.SetID != 1 || piece.Selector != 0 {
		t.Fatalf("identity = set %d selector %d", piece.SetID, piece.Selector)
	}
	if len(piece.WallDefs) != 3 || len(piece.SymbolBlockIDs) != 3 || piece.SymbolBlockIDs[0] != 101 || piece.SymbolBlockIDs[1] != 102 || piece.SymbolBlockIDs[2] != 103 {
		t.Fatalf("wall records/symbol blocks = %d/%v", len(piece.WallDefs), piece.SymbolBlockIDs)
	}
	picture, ok := piece.Symbols[101]
	if !ok || picture.ItemCount == 0 {
		t.Fatalf("8X8D3 block 0 = present %t items %d", ok, picture.ItemCount)
	}
	t.Logf("WALLDEF3 block 0 records=%d; 8X8D3 block 0 items=%d dimensions=%dx%d", len(piece.WallDefs), picture.ItemCount, picture.Width(), picture.Height())
}
