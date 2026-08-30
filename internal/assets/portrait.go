package assets

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

var poolCreationHeadBlocks = [...]uint8{0x00, 0x08, 0x09, 0x0D, 0x10, 0x12, 0x16, 0x22, 0x2D, 0x33, 0x35, 0x39, 0x43, 0x44}
var poolCreationBodyBlocks = [...]uint8{0x01, 0x02, 0x03, 0x04, 0x07, 0x08, 0x12, 0x18, 0x1A, 0x21, 0x23, 0x25}

type PortraitParts struct {
	Head graphics.Picture
	Body graphics.Picture
}

// ReadCreationPortraitParts resolves Pool's original 1-based selectors through
// the evidence-backed HEAD3.DAX and BODY3.DAX descriptor tables. It deliberately
// does not compose the pictures while the original overlap geometry is pending.
func ReadCreationPortraitParts(zipPath string, headSelector, bodySelector uint8) (PortraitParts, error) {
	if headSelector < 1 || int(headSelector) > len(poolCreationHeadBlocks) {
		return PortraitParts{}, fmt.Errorf("Pool portrait HEAD selector %d, want 1..14", headSelector)
	}
	if bodySelector < 1 || int(bodySelector) > len(poolCreationBodyBlocks) {
		return PortraitParts{}, fmt.Errorf("Pool portrait BODY selector %d, want 1..12", bodySelector)
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return PortraitParts{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()
	head, err := readPortraitBlock(archive.File, "HEAD3.DAX", poolCreationHeadBlocks[headSelector-1], 88, 40)
	if err != nil {
		return PortraitParts{}, err
	}
	body, err := readPortraitBlock(archive.File, "BODY3.DAX", poolCreationBodyBlocks[bodySelector-1], 88, 48)
	if err != nil {
		return PortraitParts{}, err
	}
	return PortraitParts{Head: head, Body: body}, nil
}

func readPortraitBlock(members []*zip.File, name string, blockID uint8, width, height int) (graphics.Picture, error) {
	var member *zip.File
	for _, candidate := range members {
		if strings.EqualFold(filepath.Base(candidate.Name), name) {
			member = candidate
			break
		}
	}
	if member == nil {
		return graphics.Picture{}, fmt.Errorf("DOS ZIP has no %s", name)
	}
	stream, err := member.Open()
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("open %s: %w", name, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, 1<<20))
	closeErr := stream.Close()
	if readErr != nil {
		return graphics.Picture{}, fmt.Errorf("read %s: %w", name, readErr)
	}
	if closeErr != nil {
		return graphics.Picture{}, fmt.Errorf("close %s: %w", name, closeErr)
	}
	if uint64(len(data)) != member.UncompressedSize64 {
		return graphics.Picture{}, fmt.Errorf("%s exceeds 1 MiB bound", name)
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("parse %s: %w", name, err)
	}
	for _, block := range blocks {
		if block.Entry.ID != blockID {
			continue
		}
		picture, err := graphics.ParsePicture(block.Data, false, 0)
		if err != nil {
			return graphics.Picture{}, fmt.Errorf("%s block 0x%02X: %w", name, blockID, err)
		}
		if picture.Width() != width || picture.Height() != height || picture.ItemCount != 1 {
			return graphics.Picture{}, fmt.Errorf("%s block 0x%02X shape is %dx%dx%d, want %dx%dx1", name, blockID, picture.Width(), picture.Height(), picture.ItemCount, width, height)
		}
		return picture, nil
	}
	return graphics.Picture{}, fmt.Errorf("%s has no block 0x%02X", name, blockID)
}
