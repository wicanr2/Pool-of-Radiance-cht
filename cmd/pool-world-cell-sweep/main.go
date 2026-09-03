// Command pool-world-cell-sweep 把每一張地圖的每一格都餵進它自己的 ECL 區塊，
// 跑入口 0 再跑入口 1，記下停在哪一種邊界。
//
// 這**不是**「玩家走得到」的證明：每個區塊都從乾淨的變數開始，沒有走過主線。
// 它回答的是另一個問題——**remake 的 VM 撐不撐得住整包遊戲的格子腳本**，
// 以及剩下的世界會要求哪些前端還沒接的事件。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

const codeBase = 0x9900

type blockReport struct {
	Archive    uint8          `json:"archive"`
	BlockID    int            `json:"block_id"`
	GEOArchive uint8          `json:"geo_archive"`
	Cells      int            `json:"cells"`
	Boundaries map[string]int `json:"boundaries"`
	Opcodes    map[string]int `json:"passthrough_opcodes"`
	Errors     map[string]int `json:"errors,omitempty"`
}

type report struct {
	Schema      string        `json:"schema"`
	ZIP         string        `json:"zip"`
	ZIPSHA256   string        `json:"zip_sha256"`
	Scope       string        `json:"scope"`
	Blocks      []blockReport `json:"blocks"`
	TotalCells  int           `json:"total_cells"`
	TotalErrors int           `json:"total_errors"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "original DOS ZIP")
	text := flag.Bool("text", false, "print a table instead of JSON")
	out := flag.String("out", "", "write JSON here instead of stdout")
	flag.Parse()
	result, err := sweep(*zipPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *text {
		for _, block := range result.Blocks {
			fmt.Printf("ecl%d/%-2d geo%d %3d 格  %v", block.Archive, block.BlockID,
				block.GEOArchive, block.Cells, sortedCounts(block.Boundaries))
			if len(block.Opcodes) != 0 {
				fmt.Printf("  事件 %v", sortedCounts(block.Opcodes))
			}
			if len(block.Errors) != 0 {
				fmt.Printf("  錯誤 %v", sortedCounts(block.Errors))
			}
			fmt.Println()
		}
		fmt.Printf("合計 %d 格，%d 個錯誤\n", result.TotalCells, result.TotalErrors)
		return
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*out, data, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func sortedCounts(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return counts[keys[i]] > counts[keys[j]] })
	rows := make([]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return rows
}

func sweep(zipPath string) (report, error) {
	raw, err := os.ReadFile(zipPath)
	if err != nil {
		return report{}, err
	}
	sum := sha256.Sum256(raw)
	catalog, err := gamepack.ReadDOSECLCatalog(zipPath)
	if err != nil {
		return report{}, err
	}
	geometryCatalog, err := gamepack.ReadDOSGeometryCatalog(zipPath)
	if err != nil {
		return report{}, err
	}
	result := report{
		Schema: "pool-world-cell-sweep-v1", ZIP: zipPath,
		ZIPSHA256: hex.EncodeToString(sum[:]),
		Scope: "每個區塊都從乾淨的變數開始跑入口 0 與入口 1；" +
			"這是入口掃描，不是玩家走得到的證明",
	}
	for archiveNumber := uint8(1); archiveNumber <= 8; archiveNumber++ {
		archive, ok := catalog.Archive(archiveNumber)
		if !ok {
			continue
		}
		ids := make([]int, 0, len(archive.Blocks))
		for id := range archive.Blocks {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, id := range ids {
			// 區塊編號在八個 GEO 檔裡全域唯一（spec 043）。沒有對應地圖的
			// 區塊是室內場景，沒有格子可以掃。
			geoMap, ok := geometryCatalog.MapByBlock(uint8(id))
			if !ok {
				continue
			}
			row, err := sweepBlock(archive, uint16(id), geoMap)
			if err != nil {
				return report{}, err
			}
			result.Blocks = append(result.Blocks, row)
			result.TotalCells += row.Cells
			for _, count := range row.Errors {
				result.TotalErrors += count
			}
		}
	}
	return result, nil
}

func sweepBlock(archive gamepack.ECLArchive, blockID uint16, geoMap gamepack.GeometryMap) (blockReport, error) {
	row := blockReport{Archive: archive.Number, BlockID: int(blockID),
		GEOArchive: geoMap.Key.Archive,
		Boundaries: map[string]int{}, Opcodes: map[string]int{}}
	base, err := gamepack.NewCellSweepSession(archive, blockID)
	if err != nil {
		return row, err
	}
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			for facing := uint8(0); facing < 4; facing++ {
				row.Cells++
				session := base.Clone()
				spawn := gamepack.Spawn{Map: geoMap.Key, X: uint8(x), Y: uint8(y), Facing: facing}
				run, runErr := gamepack.RunInitialSessionCellEntry(session, geoMap.Grid, spawn)
				record(&row, run, runErr)
				if runErr != nil || !run.Exited {
					continue
				}
				if err := session.SetEntry(1); err != nil {
					return row, err
				}
				search, searchErr := session.RunUntilEvent(4096, nil, true)
				record(&row, search, searchErr)
			}
		}
	}
	return row, nil
}

func record(row *blockReport, run eclvm.Result, runErr error) {
	switch {
	case runErr != nil:
		row.Boundaries["error"]++
		if row.Errors == nil {
			row.Errors = map[string]int{}
		}
		row.Errors[runErr.Error()]++
	case run.WaitingForMenu:
		row.Boundaries["menu"]++
	case len(run.Events) != 0:
		row.Boundaries["event"]++
	case run.Exited:
		row.Boundaries["exit"]++
	default:
		row.Boundaries["step_limit"]++
	}
	for _, event := range run.Events {
		row.Opcodes[fmt.Sprintf("%02X", event.Opcode)]++
	}
}
