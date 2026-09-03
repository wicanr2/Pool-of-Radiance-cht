package gamepack

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/graphics"
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

func TestDOSSlumsThreePieceSlotsMatchECL2Block20(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	piece, err := ReadDOSPieceSlots(zipPath, 2, [3]uint8{2, 4, 1}, graphics.PieceSet{})
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if len(piece.WallDefs) != 3 || piece.SymbolSetIDs != nil && len(piece.SymbolSetIDs) != 3 || len(piece.SymbolBlockIDs) != 3 {
		t.Fatalf("Slums piece slots=%+v", piece)
	}
	for index, want := range []uint8{1, 2, 3} {
		if piece.SymbolSetIDs[index] != want {
			t.Fatalf("slot %d symbol set=%d, want %d", index+1, piece.SymbolSetIDs[index], want)
		}
	}
}

// FFh 的 slot 不換，沿用上一份的同一格（spec 043 的 handler 只對非 FFh 的
// selector 呼叫 LoadWallSet）。沒有上一份可以沿用時才報錯。
func TestPieceSlotsCarryOverTheFFhSentinel(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	base, err := ReadDOSPieceSlots(zipPath, 2, [3]uint8{2, 4, 1}, graphics.PieceSet{})
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	partial, err := ReadDOSPieceSlots(zipPath, 2, [3]uint8{2, 0xFF, 1}, base)
	if err != nil {
		t.Fatalf("FFh 那一格沒沿用成功：%v", err)
	}
	if len(partial.WallDefs) != 3 {
		t.Fatalf("沿用之後只剩 %d 個 slot", len(partial.WallDefs))
	}
	if partial.SymbolBlockIDs[1] != base.SymbolBlockIDs[1] {
		t.Errorf("第 2 格是 %d，預期沿用 %d",
			partial.SymbolBlockIDs[1], base.SymbolBlockIDs[1])
	}
	if _, err := ReadDOSPieceSlots(zipPath, 2, [3]uint8{2, 0xFF, 1},
		graphics.PieceSet{}); err == nil {
		t.Error("沒有上一份可以沿用時應該報錯")
	}
}
