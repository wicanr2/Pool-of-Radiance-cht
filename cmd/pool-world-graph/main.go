// Command pool-world-graph 把「世界怎麼接起來」從原始資料量出來：每一個 ECL
// 區塊會 NEWECL 到哪些區塊、載入哪些檔案，以及每一張 GEO 地圖的邊界上哪些
// 格子往外沒有牆。它只報資料，不指派劇情語意。
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
	"strconv"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// codeBase 是解碼後緩衝區第 0 個位元組的位址（spec 002）。
const codeBase = 0x9900

// 運算元有兩種：立即值（就是區塊編號）與記憶體參照（編號存在那個變數裡，
// 寫成 @6E79）。兩者混在一起看會把「跳到區塊 28281」當成真的有那個區塊。
type eclBlock struct {
	Archive   int         `json:"archive"`
	BlockID   int         `json:"block_id"`
	NewECL    []string    `json:"newecl_targets"`
	LoadFiles [][3]string `json:"load_files"`
}

type geoExit struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Facing    string `json:"facing"`
	Component int    `json:"component"`
}

type geoBlock struct {
	Archive    int `json:"archive"`
	BlockID    int `json:"block_id"`
	Walkable   int `json:"walkable_cells"`
	Components int `json:"components"`
	// CellComponents 是 16 列 × 16 行的元件編號，[y][x]。要問「這兩格走
	// 不走得到彼此」的時候查它。
	CellComponents [][]int   `json:"cell_components"`
	Exits          []geoExit `json:"boundary_exits"`
}

type report struct {
	Schema    string     `json:"schema"`
	ZIP       string     `json:"zip"`
	ZIPSHA256 string     `json:"zip_sha256"`
	ECLBlocks []eclBlock `json:"ecl_blocks"`
	GEOBlocks []geoBlock `json:"geo_blocks"`
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
		printText(result)
		return
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printText(result report) {
	fmt.Println("ECL：NEWECL 目標與 LOAD FILES")
	for _, block := range result.ECLBlocks {
		fmt.Printf("  ecl%d/%-2d  NEWECL→ %v  LOAD FILES %v\n",
			block.Archive, block.BlockID, block.NewECL, block.LoadFiles)
	}
	fmt.Println("GEO：邊界上往外沒有牆的格子（component 相同才走得到彼此）")
	for _, block := range result.GEOBlocks {
		labels := make([]string, 0, len(block.Exits))
		for _, exit := range block.Exits {
			labels = append(labels, fmt.Sprintf("(%d,%d)%s#%d",
				exit.X, exit.Y, exit.Facing, exit.Component))
		}
		fmt.Printf("  geo%d/%-2d  可走 %3d 格 %d 區  出口 %2d  %s\n",
			block.Archive, block.BlockID, block.Walkable, block.Components,
			len(block.Exits), strings.Join(labels, " "))
	}
}

func build(zipPath string) (report, error) {
	raw, err := os.ReadFile(zipPath)
	if err != nil {
		return report{}, err
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()
	sum := sha256.Sum256(raw)
	result := report{Schema: "pool-world-graph-v1", ZIP: filepath.Base(zipPath),
		ZIPSHA256: hex.EncodeToString(sum[:])}
	for _, member := range archive.File {
		name := strings.ToUpper(filepath.Base(member.Name))
		if len(name) != 8 || name[4:] != ".DAX" || name[3] < '1' || name[3] > '8' {
			continue
		}
		kind := name[:3]
		if kind != "ECL" && kind != "GEO" {
			continue
		}
		number := int(name[3] - '0')
		data, err := readMember(member)
		if err != nil {
			return report{}, err
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return report{}, fmt.Errorf("parse %s: %w", name, err)
		}
		for _, block := range blocks {
			if kind == "ECL" {
				row, err := readECL(number, block)
				if err != nil {
					return report{}, err
				}
				result.ECLBlocks = append(result.ECLBlocks, row)
				continue
			}
			row, err := readGEO(number, block)
			if err != nil {
				return report{}, err
			}
			result.GEOBlocks = append(result.GEOBlocks, row)
		}
	}
	sortBlocks(result.ECLBlocks, func(i int) (int, int) {
		return result.ECLBlocks[i].Archive, result.ECLBlocks[i].BlockID
	})
	sort.Slice(result.GEOBlocks, func(i, j int) bool {
		if result.GEOBlocks[i].Archive != result.GEOBlocks[j].Archive {
			return result.GEOBlocks[i].Archive < result.GEOBlocks[j].Archive
		}
		return result.GEOBlocks[i].BlockID < result.GEOBlocks[j].BlockID
	})
	return result, nil
}

func sortBlocks(rows []eclBlock, key func(int) (int, int)) {
	sort.Slice(rows, func(i, j int) bool {
		ai, bi := rows[i].Archive, rows[i].BlockID
		aj, bj := rows[j].Archive, rows[j].BlockID
		if ai != aj {
			return ai < aj
		}
		return bi < bj
	})
}

func readMember(member *zip.File) ([]byte, error) {
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
	return data, nil
}

func readECL(number int, block dax.Block) (eclBlock, error) {
	row := eclBlock{Archive: number, BlockID: int(block.Entry.ID)}
	points, _, err := ecl.EntryPoints(block.Data, 5)
	if err != nil {
		// 有幾個區塊沒有 5 個命令集；那不是這支工具要回答的問題。
		return row, nil
	}
	starts := make([]int, 0, len(points))
	for _, point := range points {
		starts = append(starts, int(point)-codeBase)
	}
	graph, err := ecl.TraceGraphAtBase(block.Data, starts, codeBase, len(block.Data)*8)
	if err != nil {
		return row, nil
	}
	targets := map[string]bool{}
	files := map[[3]string]bool{}
	for _, instruction := range graph.Instructions {
		switch instruction.Command.Name {
		case "NEWECL":
			if len(instruction.Operands) == 1 {
				targets[operandValue(instruction.Operands[0])] = true
			}
		case "LOAD FILES":
			if len(instruction.Operands) == 3 {
				files[[3]string{operandValue(instruction.Operands[0]),
					operandValue(instruction.Operands[1]),
					operandValue(instruction.Operands[2])}] = true
			}
		}
	}
	for target := range targets {
		row.NewECL = append(row.NewECL, target)
	}
	sort.Slice(row.NewECL, func(i, j int) bool {
		return lessOperand(row.NewECL[i], row.NewECL[j])
	})
	for file := range files {
		row.LoadFiles = append(row.LoadFiles, file)
	}
	sort.Slice(row.LoadFiles, func(i, j int) bool {
		for index := 0; index < 3; index++ {
			if row.LoadFiles[i][index] != row.LoadFiles[j][index] {
				return lessOperand(row.LoadFiles[i][index], row.LoadFiles[j][index])
			}
		}
		return false
	})
	return row, nil
}

// lessOperand 讓立即值照數字排在前面，記憶體參照排在後面。
func lessOperand(a, b string) bool {
	an, aok := strconv.Atoi(a)
	bn, bok := strconv.Atoi(b)
	if aok == nil && bok == nil {
		return an < bn
	}
	if aok == nil {
		return true
	}
	if bok == nil {
		return false
	}
	return a < b
}

func operandValue(operand ecl.Operand) string {
	if operand.WordSet {
		return fmt.Sprintf("@%04X", operand.Word)
	}
	return fmt.Sprintf("%d", operand.Low)
}

// exitDeltas 是 0 北、1 東、2 南、3 西（spec 076）。
var exitDeltas = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

const facingNames = "NESW"

func readGEO(number int, block dax.Block) (geoBlock, error) {
	row := geoBlock{Archive: number, BlockID: int(block.Entry.ID)}
	grid, err := geometry.Parse(block.Entry.ID, block.Data)
	if err != nil {
		return row, fmt.Errorf("geo%d block %d: %w", number, block.Entry.ID, err)
	}
	// 連通元件：只走「兩邊都通」的邊。單向的邊（走得過去、回不來）不併成
	// 同一組——併起來的話，一張圖的出口看起來全部互通，而實際上走不到。
	// 城區 28 個邊界出口裡，站在起點真正走得到的只有一個。
	component := map[[2]int]int{}
	next := 0
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			if _, seen := component[[2]int{x, y}]; seen {
				continue
			}
			next++
			size := 0
			queue := [][2]int{{x, y}}
			component[[2]int{x, y}] = next
			for len(queue) > 0 {
				cell := queue[0]
				queue = queue[1:]
				size++
				for facing := 0; facing < 4; facing++ {
					nextX := cell[0] + exitDeltas[facing][0]
					nextY := cell[1] + exitDeltas[facing][1]
					if nextX < 0 || nextX >= geometry.Width ||
						nextY < 0 || nextY >= geometry.Height {
						continue
					}
					if !grid.CanMoveDungeonWrapped(cell[0], cell[1], facing*2) {
						continue
					}
					if !grid.CanMoveDungeonWrapped(nextX, nextY, ((facing+2)%4)*2) {
						continue
					}
					if _, seen := component[[2]int{nextX, nextY}]; seen {
						continue
					}
					component[[2]int{nextX, nextY}] = next
					queue = append(queue, [2]int{nextX, nextY})
				}
			}
			if size > row.Walkable {
				row.Walkable = size
			}
		}
	}
	row.Components = next
	row.CellComponents = make([][]int, geometry.Height)
	for y := 0; y < geometry.Height; y++ {
		row.CellComponents[y] = make([]int, geometry.Width)
		for x := 0; x < geometry.Width; x++ {
			row.CellComponents[y][x] = component[[2]int{x, y}]
		}
	}
	for y := 0; y < geometry.Height; y++ {
		for x := 0; x < geometry.Width; x++ {
			for facing := 0; facing < 4; facing++ {
				nextX := x + exitDeltas[facing][0]
				nextY := y + exitDeltas[facing][1]
				if nextX >= 0 && nextX < geometry.Width &&
					nextY >= 0 && nextY < geometry.Height {
					continue
				}
				if !grid.CanMoveDungeonWrapped(x, y, facing*2) {
					continue
				}
				row.Exits = append(row.Exits, geoExit{X: x, Y: y,
					Facing:    string(facingNames[facing]),
					Component: component[[2]int{x, y}]})
			}
		}
	}
	return row, nil
}
