// Command pool-inventory performs a read-only shape audit of a DOS game ZIP.
// It never extracts or rewrites proprietary input files.
package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
)

type daxFile struct {
	Name   string `json:"name"`
	Bytes  uint64 `json:"bytes"`
	Blocks int    `json:"blocks,omitempty"`
	Error  string `json:"error,omitempty"`
}

type report struct {
	ZIP       string    `json:"zip"`
	Entries   int       `json:"entries"`
	DAXFiles  []daxFile `json:"dax_files"`
	DAXOK     int       `json:"dax_ok"`
	DAXFailed int       `json:"dax_failed"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	flag.Parse()
	result, err := inventory(*zipPath)
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
	if result.DAXFailed != 0 {
		os.Exit(2)
	}
}

func inventory(zipPath string) (report, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer reader.Close()

	result := report{ZIP: filepath.Base(zipPath), Entries: len(reader.File)}
	for _, file := range reader.File {
		if !strings.EqualFold(filepath.Ext(file.Name), ".dax") {
			continue
		}
		row := daxFile{Name: file.Name, Bytes: file.UncompressedSize64}
		stream, openErr := file.Open()
		if openErr != nil {
			row.Error = openErr.Error()
		} else {
			data, readErr := io.ReadAll(io.LimitReader(stream, 64<<20))
			closeErr := stream.Close()
			if readErr != nil {
				row.Error = readErr.Error()
			} else if closeErr != nil {
				row.Error = closeErr.Error()
			} else if uint64(len(data)) != file.UncompressedSize64 {
				row.Error = "DAX exceeds 64 MiB audit bound"
			} else if blocks, parseErr := dax.Parse(data); parseErr != nil {
				row.Error = parseErr.Error()
			} else {
				row.Blocks = len(blocks)
			}
		}
		if row.Error == "" {
			result.DAXOK++
		} else {
			result.DAXFailed++
		}
		result.DAXFiles = append(result.DAXFiles, row)
	}
	sort.Slice(result.DAXFiles, func(i, j int) bool {
		return result.DAXFiles[i].Name < result.DAXFiles[j].Name
	})
	return result, nil
}
