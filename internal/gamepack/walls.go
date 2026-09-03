package gamepack

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

const maxWallArchiveBytes = 1 << 20

// ReadDOSPieceSet decodes one Pool LOAD PIECES selection from the matching
// WALLDEF and 8X8D archives. Archive is the original DS:52D4 file-set number;
// setID and selector retain the original LoadWallSet arguments.
func ReadDOSPieceSet(zipPath string, archive, setID, selector uint8) (graphics.PieceSet, error) {
	if archive < 1 || archive > 8 {
		return graphics.PieceSet{}, fmt.Errorf("wall archive %d is outside 1..8", archive)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return graphics.PieceSet{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer zr.Close()
	wantWall := fmt.Sprintf("WALLDEF%d.DAX", archive)
	wantSymbols := fmt.Sprintf("8X8D%d.DAX", archive)
	wallMember, err := uniqueMember(zr.File, wantWall)
	if err != nil {
		return graphics.PieceSet{}, err
	}
	symbolMember, err := uniqueMember(zr.File, wantSymbols)
	if err != nil {
		return graphics.PieceSet{}, err
	}
	wallBlocks, err := readDAXBlocks(wallMember)
	if err != nil {
		return graphics.PieceSet{}, fmt.Errorf("%s: %w", wantWall, err)
	}
	symbolBlocks, err := readDAXBlocks(symbolMember)
	if err != nil {
		return graphics.PieceSet{}, fmt.Errorf("%s: %w", wantSymbols, err)
	}
	return graphics.ParsePieceSet(setID, selector, wallBlocks, symbolBlocks)
}

// ReadDOSPieceSlots loads all three explicit LOAD PIECES selectors and builds
// the flat three-band view consumed by the first-person renderer.
// ReadDOSPieceSlots 載入 `37h LOAD PIECES` 的三個 slot。
//
// selector 是 `FFh` 的那一個 slot **不載**，沿用 previous 的同一格：原版的
// handler 逐 slot 掃三欄，只對非 `FFh` 的呼叫 `LoadWallSet`（spec 043）。
// previous 沒有那一格時才報錯——那代表遊戲要求沿用一個從來沒載過的 slot。
func ReadDOSPieceSlots(zipPath string, archive uint8, selectors [3]uint8,
	previous graphics.PieceSet) (graphics.PieceSet, error) {
	present, err := readWallBlockIDs(zipPath, archive)
	if err != nil {
		return graphics.PieceSet{}, err
	}
	result := graphics.PieceSet{SetID: 1, Symbols: map[uint8]graphics.Picture{}}
	for index, selector := range selectors {
		if selector == 0xFF {
			// 沿用只認得「一格一筆」的舊資料；連續記錄的 slot 還沒有
			// 需要沿用的案例出現。
			if index >= len(previous.WallDefs) || index >= len(previous.SymbolSetIDs) ||
				index >= len(previous.SymbolBlockIDs) {
				return graphics.PieceSet{}, fmt.Errorf(
					"LOAD PIECES slot %d is FFh but no earlier set has that slot", index+1)
			}
			result.WallDefs = append(result.WallDefs, previous.WallDefs[index])
			result.SymbolSetIDs = append(result.SymbolSetIDs, previous.SymbolSetIDs[index])
			result.SymbolBlockIDs = append(result.SymbolBlockIDs, previous.SymbolBlockIDs[index])
			continue
		}
		base, record, err := resolveWallSelector(present, selector)
		if err != nil {
			return graphics.PieceSet{}, err
		}
		piece, err := ReadDOSPieceSet(zipPath, archive, uint8(index+1), base)
		if err != nil {
			return graphics.PieceSet{}, err
		}
		if record >= len(piece.WallDefs) || record >= len(piece.SymbolSetIDs) ||
			record >= len(piece.SymbolBlockIDs) {
			return graphics.PieceSet{}, fmt.Errorf(
				"LOAD PIECES slot %d selector %d wants record %d of block %d, which has %d",
				index+1, selector, record, base, len(piece.WallDefs))
		}
		result.WallDefs = append(result.WallDefs, piece.WallDefs[record])
		result.SymbolSetIDs = append(result.SymbolSetIDs, piece.SymbolSetIDs[record])
		result.SymbolBlockIDs = append(result.SymbolBlockIDs, piece.SymbolBlockIDs[record])
		for id, picture := range piece.Symbols {
			// 同一個編號出現兩次：內容一樣就留第一份（兩個 slot 指到同一塊
			// 圖形），不一樣才是真的撞號。
			if existing, exists := result.Symbols[id]; exists {
				if !reflect.DeepEqual(existing, picture) {
					return graphics.PieceSet{}, fmt.Errorf(
						"LOAD PIECES repeats symbol block %d with different pixels", id)
				}
				continue
			}
			result.Symbols[id] = picture
		}
	}
	// FFh 沿用的那幾格的圖形也要帶過來，否則畫面上會少一塊。
	for id, picture := range previous.Symbols {
		if _, exists := result.Symbols[id]; !exists {
			result.Symbols[id] = picture
		}
	}
	return result, nil
}

// resolveWallSelector 把 selector 換成「哪一塊 WALLDEF 的第幾筆記錄」。
//
// selector 本身就是編號時是第 0 筆。**編號不存在時退到比它小的最近一塊，
// 取第 (selector − 編號) 筆**：`WALLDEF4.DAX` 的 21 是兩筆連著的記錄，
// 而 ecl4/10 要的是 22；`WALLDEF5.DAX` 的 24 也是兩筆，ecl5/3、5/4、5/6
// 要的是 25。全遊戲 33 處 `LOAD PIECES` 只有這五處落在編號之外，
// 每一處都正好落在前一塊的記錄數之內。
func resolveWallSelector(present map[uint8]int, selector uint8) (uint8, int, error) {
	if count, ok := present[selector]; ok {
		if count <= 0 {
			return 0, 0, fmt.Errorf("WALLDEF selector %d has no records", selector)
		}
		return selector, 0, nil
	}
	best, found := uint8(0), false
	for id := range present {
		if id <= selector && (!found || id > best) {
			best, found = id, true
		}
	}
	if !found {
		return 0, 0, fmt.Errorf("WALLDEF selector %d is not present", selector)
	}
	record := int(selector) - int(best)
	if record >= present[best] {
		return 0, 0, fmt.Errorf(
			"WALLDEF selector %d falls past block %d, which has %d records",
			selector, best, present[best])
	}
	return best, record, nil
}

// readWallBlockIDs 回報一個 WALLDEF 檔裡有哪些編號、各有幾筆記錄。
func readWallBlockIDs(zipPath string, archive uint8) (map[uint8]int, error) {
	if archive < 1 || archive > 8 {
		return nil, fmt.Errorf("wall archive %d is outside 1..8", archive)
	}
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer reader.Close()
	member, err := uniqueMember(reader.File, fmt.Sprintf("WALLDEF%d.DAX", archive))
	if err != nil {
		return nil, err
	}
	blocks, err := readDAXBlocks(member)
	if err != nil {
		return nil, err
	}
	result := make(map[uint8]int, len(blocks))
	for id, data := range blocks {
		walls, err := graphics.ParseWallDefs(data)
		if err != nil {
			return nil, fmt.Errorf("WALLDEF%d block %d: %w", archive, id, err)
		}
		result[id] = len(walls)
	}
	return result, nil
}

func uniqueMember(files []*zip.File, want string) (*zip.File, error) {
	var match *zip.File
	for _, member := range files {
		if !strings.EqualFold(filepath.Base(member.Name), want) {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("DOS ZIP has duplicate %s", want)
		}
		match = member
	}
	if match == nil {
		return nil, fmt.Errorf("DOS ZIP has no %s", want)
	}
	return match, nil
}

func readDAXBlocks(member *zip.File) (map[uint8][]byte, error) {
	data, err := readBoundedZIPMember(member, maxWallArchiveBytes)
	if err != nil {
		return nil, err
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return nil, err
	}
	result := make(map[uint8][]byte, len(blocks))
	for _, block := range blocks {
		if _, exists := result[block.Entry.ID]; exists {
			return nil, fmt.Errorf("duplicate block %d", block.Entry.ID)
		}
		result[block.Entry.ID] = block.Data
	}
	return result, nil
}
