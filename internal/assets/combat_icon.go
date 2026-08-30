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

// CombatIconSelection stores Pool's two persistent selectors and size flag.
// Action and small variants occupy regular archive-ID families; they are not
// additional CHA fields.
type CombatIconSelection struct {
	Head, Body uint8
	Size       uint8
}

func CombatIconBlockIDs(selection CombatIconSelection, action bool) (uint8, uint8, error) {
	if selection.Head > 13 || selection.Body > 31 {
		return 0, 0, fmt.Errorf("Pool combat icon selectors head=%d body=%d exceed 13/31", selection.Head, selection.Body)
	}
	if selection.Size != 1 && selection.Size != 2 {
		return 0, 0, fmt.Errorf("Pool combat icon size %d, want 1 or 2", selection.Size)
	}
	family := uint8(0)
	if selection.Size == 1 {
		family += 0x40
	}
	if action {
		family += 0x80
	}
	return family + selection.Head, family + selection.Body, nil
}

func ReadCombatIcon(zipPath string, selection CombatIconSelection, action bool) (graphics.Picture, error) {
	return ReadCustomizedCombatIcon(zipPath, selection, action, [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}})
}

// ReadCustomizedCombatIcon applies Pool's six dual-color palette slots to
// both masked parts before their OR merge. The slot order is Body, Arm, Leg,
// Hair/Face, Shield, Weapon, matching CHA C1h..C6h.
func ReadCustomizedCombatIcon(zipPath string, selection CombatIconSelection, action bool, colors [6][2]uint8) (graphics.Picture, error) {
	headID, bodyID, err := CombatIconBlockIDs(selection, action)
	if err != nil {
		return graphics.Picture{}, err
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return graphics.Picture{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()
	body, err := readCombatIconBlock(archive.File, "CBODY.DAX", bodyID)
	if err != nil {
		return graphics.Picture{}, err
	}
	head, err := readCombatIconBlock(archive.File, "CHEAD.DAX", headID)
	if err != nil {
		return graphics.Picture{}, err
	}
	body = recolorCombatIcon(body, colors)
	head = recolorCombatIcon(head, colors)
	return graphics.MergePictures(body, head)
}

func recolorCombatIcon(picture graphics.Picture, colors [6][2]uint8) graphics.Picture {
	result := picture
	result.Pixels = append([]uint8(nil), picture.Pixels...)
	templates := [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}}
	for index, pixel := range result.Pixels {
		for part, pair := range templates {
			for component, source := range pair {
				if pixel == source {
					result.Pixels[index] = colors[part][component] & 0x0F
				}
			}
		}
	}
	return result
}

func readCombatIconBlock(members []*zip.File, name string, blockID uint8) (graphics.Picture, error) {
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
		picture, err := graphics.ParsePicture(block.Data, true, 0)
		if err != nil {
			return graphics.Picture{}, fmt.Errorf("%s block 0x%02X: %w", name, blockID, err)
		}
		if picture.Width() != 24 || picture.ItemCount != 1 {
			return graphics.Picture{}, fmt.Errorf("%s block 0x%02X shape is %dx%dx%d", name, blockID, picture.Width(), picture.Height(), picture.ItemCount)
		}
		return picture, nil
	}
	return graphics.Picture{}, fmt.Errorf("%s has no block 0x%02X", name, blockID)
}
