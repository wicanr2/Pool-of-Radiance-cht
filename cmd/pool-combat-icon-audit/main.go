// Command pool-combat-icon-audit inventories Pool's complete combat-icon
// archives through the reusable engine decoder. It never exports source art.
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
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

type blockRow struct {
	Source string `json:"source"`
	ID     uint8  `json:"block_id"`
	Bytes  int    `json:"bytes"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Items  int    `json:"item_count"`
	Error  string `json:"error,omitempty"`
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
		if name != "CHEAD.DAX" && name != "CBODY.DAX" {
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
			row := blockRow{Source: name, ID: block.Entry.ID, Bytes: len(block.Data)}
			result.Blocks++
			picture, err := graphics.ParsePicture(block.Data, true, 0)
			if err != nil {
				row.Error = err.Error()
				result.Failed++
			} else {
				row.Width, row.Height, row.Items = picture.Width(), picture.Height(), int(picture.ItemCount)
				result.Decoded++
			}
			result.Rows = append(result.Rows, row)
		}
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		if result.Rows[i].Source != result.Rows[j].Source {
			return result.Rows[i].Source < result.Rows[j].Source
		}
		return result.Rows[i].ID < result.Rows[j].ID
	})
	if result.Archives != 2 {
		return report{}, fmt.Errorf("combat icon archive count %d, want 2", result.Archives)
	}
	return result, nil
}
