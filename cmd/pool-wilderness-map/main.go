// Command pool-wilderness-map 解出野外地圖上「哪一格有東西」的表。
//
// 表怎麼解在 `internal/gamepack.ReadDOSWildernessSheets`——探索器也用同一份，
// 這支只負責印出來。表的來歷與版面在 spec 105。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

type report struct {
	Schema    string                    `json:"schema"`
	ZIP       string                    `json:"zip"`
	ZIPSHA256 string                    `json:"zip_sha256"`
	Sheets    []gamepack.WildernessSheet `json:"sheets"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "original DOS ZIP")
	text := flag.Bool("text", false, "print a table instead of JSON")
	flag.Parse()
	result, err := build(*zipPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *text {
		for _, row := range result.Sheets {
			labels := make([]string, 0, len(row.Places))
			for _, item := range row.Places {
				labels = append(labels, fmt.Sprintf("(%d,%d)→%d", item.X, item.Y, item.LocationID))
			}
			fmt.Printf("ecl%d/%-2d %d 列 %2d 個地點  %s\n",
				row.Archive, row.BlockID, row.Rows, len(row.Places), strings.Join(labels, " "))
		}
		return
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(zipPath string) (report, error) {
	raw, err := os.ReadFile(zipPath)
	if err != nil {
		return report{}, err
	}
	sum := sha256.Sum256(raw)
	result := report{Schema: "pool-wilderness-map-v1", ZIP: filepath.Base(zipPath),
		ZIPSHA256: hex.EncodeToString(sum[:])}
	result.Sheets, err = gamepack.ReadDOSWildernessSheets(zipPath)
	if err != nil {
		return report{}, err
	}
	sort.Slice(result.Sheets, func(i, j int) bool {
		return result.Sheets[i].BlockID < result.Sheets[j].BlockID
	})
	return result, nil
}
