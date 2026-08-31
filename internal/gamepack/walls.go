package gamepack

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

const maxWallArchiveBytes = 1 << 20

// ReadDOSPieceSet decodes one Pool LOAD PIECES selection from the matching
// WALLDEF and 8X8D archives. Archive is the original DS:52D4 file-set number;
// setID and selector retain the original LoadWallSet arguments.
func ReadDOSPieceSet(zipPath string, archive, setID, selector uint8) (graphics.PieceSet, error) {
	if archive < 1 || archive > 8 {
		return graphics.PieceSet{}, fmt.Errorf("wall archive %d is outside 1..8", archive)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return graphics.PieceSet{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer zr.Close()
	wantWall := fmt.Sprintf("WALLDEF%d.DAX", archive)
	wantSymbols := fmt.Sprintf("8X8D%d.DAX", archive)
	wallMember, err := uniqueMember(zr.File, wantWall)
	if err != nil {
		return graphics.PieceSet{}, err
	}
	symbolMember, err := uniqueMember(zr.File, wantSymbols)
	if err != nil {
		return graphics.PieceSet{}, err
	}
	wallBlocks, err := readDAXBlocks(wallMember)
	if err != nil {
		return graphics.PieceSet{}, fmt.Errorf("%s: %w", wantWall, err)
	}
	symbolBlocks, err := readDAXBlocks(symbolMember)
	if err != nil {
		return graphics.PieceSet{}, fmt.Errorf("%s: %w", wantSymbols, err)
	}
	return graphics.ParsePieceSet(setID, selector, wallBlocks, symbolBlocks)
}

func uniqueMember(files []*zip.File, want string) (*zip.File, error) {
	var match *zip.File
	for _, member := range files {
		if !strings.EqualFold(filepath.Base(member.Name), want) {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("DOS ZIP has duplicate %s", want)
		}
		match = member
	}
	if match == nil {
		return nil, fmt.Errorf("DOS ZIP has no %s", want)
	}
	return match, nil
}

func readDAXBlocks(member *zip.File) (map[uint8][]byte, error) {
	data, err := readBoundedZIPMember(member, maxWallArchiveBytes)
	if err != nil {
		return nil, err
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return nil, err
	}
	result := make(map[uint8][]byte, len(blocks))
	for _, block := range blocks {
		if _, exists := result[block.Entry.ID]; exists {
			return nil, fmt.Errorf("duplicate block %d", block.Entry.ID)
		}
		result[block.Entry.ID] = block.Data
	}
	return result, nil
}
