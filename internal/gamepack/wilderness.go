package gamepack

import "fmt"

// 野外地圖上「哪一格有東西」的表（spec 105）。
//
// 三張野外圖（ECL block 25、26、27）的入口 1 用同一組四張表把隊伍的野外座標
// （`DS:49C3h` 是 X、`DS:49C4h` 是 Y）換成一個地點編號，再用 `ON GOTO` 分派到
// 那個地點的腳本。四張表在區塊裡連著放，所以長度從位址差算得出來，不必猜。

// wildernessCodeBase 是解碼後緩衝區第 0 個位元組的位址（spec 002）。
// 區塊前兩個位元組是長度標頭，所以它對到 `Blocks[id]` 的 offset 2。
const wildernessCodeBase = 0x9900

// WildernessSheetTables 是一張野外圖與它那四張表的位址。位址是從靜態追蹤讀
// 出來的（`GETTABLE` 的第一個運算元），逐筆記在 spec 105。
type WildernessSheetTables struct {
	Archive uint8
	BlockID uint8
	// YTable 是每一列的 Y；XTable 緊接在後面，是攤平的 X；
	// CountTable 是每一列有幾個 X；IDTable 與 XTable 平行，是地點編號。
	YTable, XTable, CountTable, IDTable uint16
}

// WildernessSheetTableSet 是三張野外圖的表位址。
var WildernessSheetTableSet = []WildernessSheetTables{
	{Archive: 6, BlockID: 25, YTable: 0xADD6, XTable: 0xADDE, CountTable: 0xADF5, IDTable: 0xADFD},
	{Archive: 7, BlockID: 26, YTable: 0xB04A, XTable: 0xB053, CountTable: 0xB061, IDTable: 0xB06A},
	{Archive: 8, BlockID: 27, YTable: 0xABA6, XTable: 0xABAB, CountTable: 0xABB4, IDTable: 0xABB9},
}

// WildernessPlace 是野外圖上有腳本的一格。
type WildernessPlace struct {
	X          int `json:"x"`
	Y          int `json:"y"`
	LocationID int `json:"location_id"`
}

// WildernessSheet 是一張野外圖解出來的地點表。
type WildernessSheet struct {
	Archive int               `json:"archive"`
	BlockID int               `json:"block_id"`
	Rows    int               `json:"rows"`
	Places  []WildernessPlace `json:"places"`
}

// ReadDOSWildernessSheets 解出三張野外圖的地點表。
func ReadDOSWildernessSheets(zipPath string) ([]WildernessSheet, error) {
	result := make([]WildernessSheet, 0, len(WildernessSheetTableSet))
	for _, tables := range WildernessSheetTableSet {
		archive, err := ReadDOSECLArchive(zipPath, tables.Archive)
		if err != nil {
			return nil, err
		}
		block, ok := archive.Blocks[uint16(tables.BlockID)]
		if !ok {
			return nil, fmt.Errorf("ECL%d.DAX has no block %d", tables.Archive, tables.BlockID)
		}
		if len(block) < 3 {
			return nil, fmt.Errorf("ECL%d.DAX block %d is too short", tables.Archive, tables.BlockID)
		}
		sheet, err := decodeWildernessSheet(block[2:], tables)
		if err != nil {
			return nil, err
		}
		result = append(result, sheet)
	}
	return result, nil
}

func decodeWildernessSheet(payload []byte, tables WildernessSheetTables) (WildernessSheet, error) {
	sheet := WildernessSheet{Archive: int(tables.Archive), BlockID: int(tables.BlockID)}
	// 四張表連著放，長度因此從位址差算得出來：列數 = X 表位址 − Y 表位址，
	// 項目數 = 每列數量表位址 − X 表位址。
	rows := int(tables.XTable) - int(tables.YTable)
	entries := int(tables.CountTable) - int(tables.XTable)
	if rows <= 0 || entries <= 0 {
		return sheet, fmt.Errorf("ecl%d/%d 的表位址不成立", tables.Archive, tables.BlockID)
	}
	if int(tables.IDTable)-int(tables.CountTable) != rows {
		return sheet, fmt.Errorf("ecl%d/%d 的每列數量表長度 %d，預期 %d",
			tables.Archive, tables.BlockID, int(tables.IDTable)-int(tables.CountTable), rows)
	}
	sheet.Rows = rows
	get := func(address uint16, index int) (int, error) {
		offset := int(address) - wildernessCodeBase + index
		if offset < 0 || offset >= len(payload) {
			return 0, fmt.Errorf("位址 %04X+%d 超出區塊", address, index)
		}
		return int(payload[offset]), nil
	}
	cursor := 0
	for line := 0; line < rows; line++ {
		y, err := get(tables.YTable, line)
		if err != nil {
			return sheet, err
		}
		count, err := get(tables.CountTable, line)
		if err != nil {
			return sheet, err
		}
		for step := 0; step < count; step++ {
			if cursor >= entries {
				return sheet, fmt.Errorf("ecl%d/%d 的 X 表只有 %d 項，第 %d 列要不到",
					tables.Archive, tables.BlockID, entries, line)
			}
			x, err := get(tables.XTable, cursor)
			if err != nil {
				return sheet, err
			}
			id, err := get(tables.IDTable, cursor)
			if err != nil {
				return sheet, err
			}
			sheet.Places = append(sheet.Places, WildernessPlace{X: x, Y: y, LocationID: id})
			cursor++
		}
	}
	if cursor != entries {
		return sheet, fmt.Errorf("ecl%d/%d 的每列數量加起來是 %d，X 表有 %d 項",
			tables.Archive, tables.BlockID, cursor, entries)
	}
	return sheet, nil
}
