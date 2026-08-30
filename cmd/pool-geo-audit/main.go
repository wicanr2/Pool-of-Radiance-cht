// Command pool-geo-audit decodes every Pool GEO block through the shared
// engine and records only structural map evidence. It does not assign story
// names or guess the new-game entry map.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

type blockRow struct {
	Source           string        `json:"source"`
	BlockID          uint8         `json:"block_id"`
	Bytes            int           `json:"bytes"`
	Prefix           [2]uint8      `json:"prefix"`
	TerrainCounts    map[uint8]int `json:"terrain_counts"`
	WallSides        int           `json:"wall_sides"`
	DoorDetailCounts map[uint8]int `json:"wall_detail_counts"`
	BoundedMoveEdges int           `json:"bounded_move_edges"`
	WrappedMoveEdges int           `json:"wrapped_move_edges"`
	DungeonMoveEdges int           `json:"dungeon_move_edges"`
	Error            string        `json:"error,omitempty"`
}

type report struct {
	ZIP       string     `json:"zip"`
	ZIPSHA256 string     `json:"zip_sha256"`
	Archives  int        `json:"archives"`
	Blocks    int        `json:"blocks"`
	Decoded   int        `json:"decoded"`
	Failed    int        `json:"failed"`
	Rows      []blockRow `json:"block_results"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	flag.Parse()
	result, err := audit(*zipPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.Failed != 0 {
		os.Exit(2)
	}
}

func audit(zipPath string) (report, error) {
	input, err := os.Open(zipPath)
	if err != nil {
		return report{}, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		input.Close()
		return report{}, err
	}
	if err := input.Close(); err != nil {
		return report{}, err
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()
	result := report{ZIP: filepath.Base(zipPath), ZIPSHA256: fmt.Sprintf("%x", hash.Sum(nil))}
	for _, member := range archive.File {
		name := strings.ToUpper(filepath.Base(member.Name))
		if !isGEOArchive(name) {
			continue
		}
		result.Archives++
		stream, err := member.Open()
		if err != nil {
			return report{}, err
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, 1<<20))
		closeErr := stream.Close()
		if readErr != nil {
			return report{}, readErr
		}
		if closeErr != nil {
			return report{}, closeErr
		}
		if uint64(len(data)) != member.UncompressedSize64 {
			return report{}, fmt.Errorf("%s exceeds 1 MiB bound", name)
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return report{}, fmt.Errorf("parse %s: %w", name, err)
		}
		for _, block := range blocks {
			row := measure(name, block.Entry.ID, block.Data)
			result.Blocks++
			if row.Error != "" {
				result.Failed++
			} else {
				result.Decoded++
			}
			result.Rows = append(result.Rows, row)
		}
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		if result.Rows[i].Source != result.Rows[j].Source {
			return result.Rows[i].Source < result.Rows[j].Source
		}
		return result.Rows[i].BlockID < result.Rows[j].BlockID
	})
	if result.Archives != 8 {
		return report{}, fmt.Errorf("GEO archive count %d, want 8", result.Archives)
	}
	if result.Blocks != 29 {
		return report{}, fmt.Errorf("GEO block count %d, want 29", result.Blocks)
	}
	return result, nil
}

func isGEOArchive(name string) bool {
	return len(name) == 8 && strings.HasPrefix(name, "GEO") && name[3] >= '1' && name[3] <= '8' && name[4:] == ".DAX"
}

func measure(source string, id uint8, data []byte) blockRow {
	row := blockRow{Source: source, BlockID: id, Bytes: len(data), TerrainCounts: make(map[uint8]int), DoorDetailCounts: make(map[uint8]int)}
	if len(data) >= 2 {
		row.Prefix = [2]uint8{data[0], data[1]}
	}
	grid, err := geometry.Parse(id, data)
	if err != nil {
		row.Error = err.Error()
		return row
	}
	directions := []int{0, 2, 4, 6}
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			cell, _ := grid.Cell(x, y)
			row.TerrainCounts[cell.Terrain]++
			for index, direction := range directions {
				if cell.WallDirections[index] != 0 {
					row.WallSides++
					row.DoorDetailCounts[cell.DetailDirections[index]]++
				}
				if grid.CanMove(x, y, direction) {
					row.BoundedMoveEdges++
				}
				if grid.CanMoveWrapped(x, y, direction) {
					row.WrappedMoveEdges++
				}
				if grid.CanMoveDungeonWrapped(x, y, direction) {
					row.DungeonMoveEdges++
				}
			}
		}
	}
	return row
}
