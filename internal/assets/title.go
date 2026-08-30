// Package assets contains Pool of Radiance title-owned archive selection and
// typed adapters. Reusable DAX and picture decoding remain in the engine.
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

// ReadTitlePictures returns TITLE.DAX blocks 1 and 2 keyed by legacy block ID.
func ReadTitlePictures(zipPath string) (map[uint8]graphics.Picture, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()
	var member *zip.File
	for _, candidate := range archive.File {
		if strings.EqualFold(filepath.Base(candidate.Name), "TITLE.DAX") {
			member = candidate
			break
		}
	}
	if member == nil {
		return nil, fmt.Errorf("DOS ZIP has no TITLE.DAX")
	}
	stream, err := member.Open()
	if err != nil {
		return nil, fmt.Errorf("open TITLE.DAX: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, 1<<20))
	closeErr := stream.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read TITLE.DAX: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close TITLE.DAX: %w", closeErr)
	}
	if uint64(len(data)) != member.UncompressedSize64 {
		return nil, fmt.Errorf("TITLE.DAX exceeds 1 MiB bound")
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse TITLE.DAX: %w", err)
	}
	if len(blocks) != 2 {
		return nil, fmt.Errorf("TITLE.DAX has %d blocks, want 2", len(blocks))
	}
	result := make(map[uint8]graphics.Picture, 2)
	for _, block := range blocks {
		if block.Entry.ID != 1 && block.Entry.ID != 2 {
			return nil, fmt.Errorf("TITLE.DAX has unexpected block 0x%02X", block.Entry.ID)
		}
		picture, err := graphics.ParsePicture(block.Data, false, 0)
		if err != nil {
			return nil, fmt.Errorf("TITLE.DAX block 0x%02X: %w", block.Entry.ID, err)
		}
		if picture.Width() != 320 || picture.Height() != 200 || picture.ItemCount != 1 {
			return nil, fmt.Errorf("TITLE.DAX block 0x%02X shape is %dx%dx%d, want 320x200x1", block.Entry.ID, picture.Width(), picture.Height(), picture.ItemCount)
		}
		result[block.Entry.ID] = picture
	}
	return result, nil
}
