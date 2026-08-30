// Command pool-portrait-audit measures Pool's HEAD/BODY archives through the
// reusable engine picture decoder. It does not export or rewrite source art.
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
	Source    string `json:"source"`
	BlockID   uint8  `json:"block_id"`
	Bytes     int    `json:"bytes"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	ItemCount int    `json:"item_count,omitempty"`
	Error     string `json:"error,omitempty"`
}
type report struct {
	ZIP          string     `json:"zip"`
	ZIPSHA256    string     `json:"zip_sha256"`
	Archives     int        `json:"archives"`
	Blocks       int        `json:"blocks"`
	Decoded      int        `json:"decoded"`
	Failed       int        `json:"failed"`
	TotalItems   int        `json:"total_items"`
	BlockResults []blockRow `json:"block_results"`
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
		if !isPortraitArchive(name) {
			continue
		}
		result.Archives++
		stream, err := member.Open()
		if err != nil {
			return report{}, fmt.Errorf("open %s: %w", name, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, 8<<20))
		closeErr := stream.Close()
		if readErr != nil {
			return report{}, fmt.Errorf("read %s: %w", name, readErr)
		}
		if closeErr != nil {
			return report{}, fmt.Errorf("close %s: %w", name, closeErr)
		}
		if uint64(len(data)) != member.UncompressedSize64 {
			return report{}, fmt.Errorf("%s exceeds 8 MiB bound", name)
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return report{}, fmt.Errorf("parse %s: %w", name, err)
		}
		for _, block := range blocks {
			row := blockRow{Source: name, BlockID: block.Entry.ID, Bytes: len(block.Data)}
			result.Blocks++
			picture, parseErr := graphics.ParsePicture(block.Data, false, 0)
			if parseErr != nil {
				row.Error = parseErr.Error()
				result.Failed++
			} else {
				row.Width, row.Height, row.ItemCount = picture.Width(), picture.Height(), int(picture.ItemCount)
				result.Decoded++
				result.TotalItems += row.ItemCount
			}
			result.BlockResults = append(result.BlockResults, row)
		}
	}
	sort.Slice(result.BlockResults, func(i, j int) bool {
		if result.BlockResults[i].Source != result.BlockResults[j].Source {
			return result.BlockResults[i].Source < result.BlockResults[j].Source
		}
		return result.BlockResults[i].BlockID < result.BlockResults[j].BlockID
	})
	if result.Archives != 16 {
		return report{}, fmt.Errorf("portrait archive count %d, want 16", result.Archives)
	}
	return result, nil
}

func isPortraitArchive(name string) bool {
	if len(name) != len("HEAD1.DAX") || !strings.HasSuffix(name, ".DAX") {
		return false
	}
	prefix, digit := name[:4], name[4]
	return (prefix == "HEAD" || prefix == "BODY") && digit >= '1' && digit <= '8'
}
