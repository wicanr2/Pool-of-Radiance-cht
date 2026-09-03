// Command pool-wilderness-map 解出野外地圖上「哪一格有東西」的表。
//
// 三張野外圖（ECL block 25、26、27）的入口 1 用同一組四張表把隊伍的野外座標
// （`DS:49C3h` 是 X、`DS:49C4h` 是 Y）換成一個地點編號，再用 `ON GOTO` 分派到
// 那個地點的腳本。四張表在區塊裡連著放：Y、X、每一列有幾個 X、地點編號。
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
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

// codeBase 是解碼後緩衝區第 0 個位元組的位址（spec 002）。
const codeBase = 0x9900

// sheet 是一張野外圖與它那四張表的位址。位址是從靜態追蹤讀出來的
// （`GETTABLE` 的第一個運算元），逐筆記在 spec 105。
type sheet struct {
	Archive uint8
	BlockID uint8
	// YTable 是每一列的 Y；XTable 緊接在後面，是攤平的 X；
	// CountTable 是每一列有幾個 X；IDTable 與 XTable 平行，是地點編號。
	YTable, XTable, CountTable, IDTable uint16
}

var sheets = []sheet{
	{Archive: 6, BlockID: 25, YTable: 0xADD6, XTable: 0xADDE, CountTable: 0xADF5, IDTable: 0xADFD},
	{Archive: 7, BlockID: 26, YTable: 0xB04A, XTable: 0xB053, CountTable: 0xB061, IDTable: 0xB06A},
	{Archive: 8, BlockID: 27, YTable: 0xABA6, XTable: 0xABAB, CountTable: 0xABB4, IDTable: 0xABB9},
}

type place struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	LocationID int `json:"location_id"`
}

type sheetReport struct {
	Archive int     `json:"archive"`
	BlockID int     `json:"block_id"`
	Rows    int     `json:"rows"`
	Places  []place `json:"places"`
}

type report struct {
	Schema    string        `json:"schema"`
	ZIP       string        `json:"zip"`
	ZIPSHA256 string        `json:"zip_sha256"`
	Sheets    []sheetReport `json:"sheets"`
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
	for _, item := range sheets {
		row, err := readSheet(zipPath, item)
		if err != nil {
			return report{}, err
		}
		result.Sheets = append(result.Sheets, row)
	}
	sort.Slice(result.Sheets, func(i, j int) bool {
		return result.Sheets[i].BlockID < result.Sheets[j].BlockID
	})
	return result, nil
}

func readSheet(zipPath string, item sheet) (sheetReport, error) {
	row := sheetReport{Archive: int(item.Archive), BlockID: int(item.BlockID)}
	payload, err := readBlockPayload(zipPath, item.Archive, item.BlockID)
	if err != nil {
		return row, err
	}
	// 四張表連著放，長度因此從位址差算得出來：列數 = X 表位址 − Y 表位址，
	// 項目數 = 每列數量表位址 − X 表位址。
	rows := int(item.XTable) - int(item.YTable)
	entries := int(item.CountTable) - int(item.XTable)
	if rows <= 0 || entries <= 0 {
		return row, fmt.Errorf("ecl%d/%d 的表位址不成立", item.Archive, item.BlockID)
	}
	if int(item.IDTable)-int(item.CountTable) != rows {
		return row, fmt.Errorf("ecl%d/%d 的每列數量表長度 %d，預期 %d",
			item.Archive, item.BlockID, int(item.IDTable)-int(item.CountTable), rows)
	}
	row.Rows = rows
	get := func(address uint16, index int) (int, error) {
		offset := int(address) - codeBase + index
		if offset < 0 || offset >= len(payload) {
			return 0, fmt.Errorf("位址 %04X+%d 超出區塊", address, index)
		}
		return int(payload[offset]), nil
	}
	cursor := 0
	for line := 0; line < rows; line++ {
		y, err := get(item.YTable, line)
		if err != nil {
			return row, err
		}
		count, err := get(item.CountTable, line)
		if err != nil {
			return row, err
		}
		for step := 0; step < count; step++ {
			if cursor >= entries {
				return row, fmt.Errorf("ecl%d/%d 的 X 表只有 %d 項，第 %d 列要不到",
					item.Archive, item.BlockID, entries, line)
			}
			x, err := get(item.XTable, cursor)
			if err != nil {
				return row, err
			}
			id, err := get(item.IDTable, cursor)
			if err != nil {
				return row, err
			}
			row.Places = append(row.Places, place{X: x, Y: y, LocationID: id})
			cursor++
		}
	}
	if cursor != entries {
		return row, fmt.Errorf("ecl%d/%d 的每列數量加起來是 %d，X 表有 %d 項",
			item.Archive, item.BlockID, cursor, entries)
	}
	return row, nil
}

func readBlockPayload(zipPath string, archiveNumber, blockID uint8) ([]byte, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	want := fmt.Sprintf("ECL%d.DAX", archiveNumber)
	for _, member := range reader.File {
		if !strings.EqualFold(filepath.Base(member.Name), want) {
			continue
		}
		stream, err := member.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, 16<<20))
		closeErr := stream.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return nil, err
		}
		for _, block := range blocks {
			if block.Entry.ID != blockID {
				continue
			}
			if len(block.Data) < 3 {
				return nil, fmt.Errorf("%s block %d is too short", want, blockID)
			}
			return block.Data[2:], nil
		}
		return nil, fmt.Errorf("%s has no block %d", want, blockID)
	}
	return nil, fmt.Errorf("missing %s", want)
}
