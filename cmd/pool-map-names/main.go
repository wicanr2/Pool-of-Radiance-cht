// Command pool-map-names 把「換圖的目的地區塊」與「同一段腳本剛印出來的字」
// 配成對，用來替每一張地圖找出**原版自己給的名字**。
//
// 為什麼要這樣配：關鍵字命中不算數。市議會派任務時會把所有地名唸一遍
// （spec 055），所以「某個區塊的文字裡出現 SLUMS」證明不了那個區塊就是貧民窟。
// 唯一站得住的形狀是**同一條控制流上**：先印一句話或給一個選項，
// 緊接著 `NEWECL <區塊>`——文字與目的地在同一個分支裡成對出現。
//
// 這一支只產生證據，不下結論：它列出每一個 `NEWECL` 前面最近的幾段文字，
// 命名要由人看過再寫進規格（spec 102 對城區地點就是這樣做的）。
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
	"github.com/wicanr2/golden-box-remake-engine/ecl"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// poolCodeAddressBase 是 Pool 的 ECL 位址基準。
const poolCodeAddressBase = 0x9900

// newECLOpcode 是 `20h NEWECL`；menuOpcodes 是兩種選單。
const newECLOpcode = 0x20

var menuOpcodes = map[byte]bool{0x15: true, 0x2B: true}

// transition 是一次換圖，以及它前面最近的文字。
type transition struct {
	Archive     string   `json:"archive"`
	BlockID     uint8    `json:"block_id"`
	Address     string   `json:"address"`
	Destination int      `json:"destination"`
	// Before 是**沿控制流往回走**收到的文字，由近到遠。Distance 是隔了幾道
	// 指令：1 代表就在這一次 NEWECL 前面。距離愈近愈可能是這一段的提示語。
	//
	// **它仍是候選不是答案**：回走會經過所有前驅，其中有些分支實際上到不了
	// 這一次 NEWECL。命名要人看過再寫進規格。
	Before []textAt `json:"text_before"`
}

// textAt 是一段文字與它離 NEWECL 幾道指令。
type textAt struct {
	Distance int    `json:"distance"`
	Text     string `json:"text"`
}

// destination 是「某個目的地區塊」被指到的所有位置與文字。
type destination struct {
	Block   int      `json:"block"`
	Sources []string `json:"sources"`
	Texts   []string `json:"texts"`
}

type report struct {
	Schema       string        `json:"schema"`
	ZIP          string        `json:"zip"`
	Transitions  []transition  `json:"transitions"`
	Destinations []destination `json:"destinations"`
	FailedBlocks []string      `json:"failed_blocks,omitempty"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	recent := flag.Int("depth", 12, "沿控制流往回走幾層")
	flag.Parse()
	result, err := collect(*zipPath, *recent)
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
}

func collect(zipPath string, recentCount int) (report, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()

	result := report{Schema: "pool-map-names-v1", ZIP: filepath.Base(zipPath)}
	names := make([]string, 0, len(archive.File))
	members := map[string]*zip.File{}
	for _, member := range archive.File {
		names = append(names, member.Name)
		members[member.Name] = member
	}
	sort.Strings(names)

	byDestination := map[int]*destination{}
	for _, name := range names {
		base := strings.ToLower(filepath.Base(name))
		if !strings.HasPrefix(base, "ecl") || !strings.EqualFold(filepath.Ext(name), ".dax") {
			continue
		}
		data, err := readMember(members[name])
		if err != nil {
			return report{}, err
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return report{}, fmt.Errorf("%s: %w", name, err)
		}
		for _, block := range blocks {
			found, failures := scanBlock(base, block, recentCount)
			result.FailedBlocks = append(result.FailedBlocks, failures...)
			for _, item := range found {
				result.Transitions = append(result.Transitions, item)
				entry := byDestination[item.Destination]
				if entry == nil {
					entry = &destination{Block: item.Destination}
					byDestination[item.Destination] = entry
				}
				entry.Sources = appendUnique(entry.Sources,
					fmt.Sprintf("%s block %d @%s", item.Archive, item.BlockID, item.Address))
				for _, text := range item.Before {
					entry.Texts = appendUnique(entry.Texts,
						fmt.Sprintf("[%d] %s", text.Distance, text.Text))
				}
			}
		}
	}
	keys := make([]int, 0, len(byDestination))
	for key := range byDestination {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	for _, key := range keys {
		result.Destinations = append(result.Destinations, *byDestination[key])
	}
	return result, nil
}

func readMember(member *zip.File) ([]byte, error) {
	stream, err := member.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return io.ReadAll(io.LimitReader(stream, 64<<20))
}

// scanBlock 走一個 block 的控制流圖，把 NEWECL 與它前面的文字配起來。
func scanBlock(base string, block dax.Block, recentCount int) ([]transition, []string) {
	var failures []string
	points, _, err := ecl.EntryPoints(block.Data, 5)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s block %d: %v", base, block.Entry.ID, err)}
	}
	starts := make([]int, 0, len(points))
	for _, point := range points {
		starts = append(starts, int(point)-poolCodeAddressBase)
	}
	graph, graphErr := ecl.TraceGraphAtBaseWithCommands(block.Data, starts,
		poolCodeAddressBase, len(block.Data)*8, gamepack.PoolCommandTable())
	if graphErr != nil {
		// 追蹤中斷仍保留已走到的指令：整個 block 掛零會被誤讀成「這裡沒有換圖」。
		failures = append(failures, fmt.Sprintf("%s block %d: %v", base, block.Entry.ID, graphErr))
	}
	// 建反向圖：從 NEWECL 往回走才拿得到「這一段的提示語」。
	// 只看位址順序會把隔壁分支的字也算進來（那正是「關鍵字命中」的老毛病）。
	byOffset := map[int]ecl.Instruction{}
	for _, instruction := range graph.Instructions {
		byOffset[instruction.Offset] = instruction
	}
	predecessors := map[int][]int{}
	for _, edge := range graph.Edges {
		predecessors[edge.To] = append(predecessors[edge.To], edge.From)
	}
	// **`Graph.Edges` 只有分支邊，沒有循序邊**——只用它回走一步都走不到
	// （實測 73 個 NEWECL 全部收不到文字）。循序邊要自己補：
	// 除了 EXIT／GOTO／RETURN 之外，每一道指令都會落到 `Next`。
	for _, instruction := range graph.Instructions {
		switch instruction.Command.Opcode {
		case 0x00, 0x01, 0x13: // EXIT、GOTO、RETURN
			continue
		}
		if _, ok := byOffset[instruction.Next]; !ok {
			continue
		}
		predecessors[instruction.Next] = append(predecessors[instruction.Next], instruction.Offset)
	}

	// textOf 把一道指令會印出來的字收齊（操作元裡的封包字串 ＋ 選單選項）。
	textOf := func(instruction ecl.Instruction) []string {
		var out []string
		for _, operand := range instruction.Operands {
			if operand.Code != 0x80 {
				continue
			}
			if text := strings.TrimSpace(ecl.DecodePackedText(operand.Packed)); text != "" {
				out = append(out, text)
			}
		}
		if menuOpcodes[instruction.Command.Opcode] {
			if menu, menuErr := ecl.DecodeMenuRecord(block.Data, instruction.Offset); menuErr == nil {
				for _, text := range menu.OptionTexts {
					if trimmed := strings.TrimSpace(text); trimmed != "" {
						out = append(out, trimmed)
					}
				}
			}
		}
		return out
	}

	var out []transition
	for _, instruction := range graph.Instructions {
		if instruction.Command.Opcode != newECLOpcode || len(instruction.Operands) == 0 {
			continue
		}
		operand := instruction.Operands[0]
		if operand.Code != 0 {
			// 目的地不是常數就跳過。**這是掃描面的洞，要記著**：
			// 變數形式的 NEWECL 這一支看不到。
			continue
		}
		out = append(out, transition{
			Archive:     base,
			BlockID:     block.Entry.ID,
			Address:     fmt.Sprintf("%04X", poolCodeAddressBase+instruction.Offset),
			Destination: int(operand.Low),
			Before:      walkBack(instruction.Offset, predecessors, byOffset, textOf, recentCount),
		})
	}
	return out, failures
}

// walkBack 由 NEWECL 沿反向邊做廣度優先，收沿路的文字。
// 走 depth 層就停——再遠的字與這一段的關係已經弱到不能當證據。
func walkBack(start int, predecessors map[int][]int, byOffset map[int]ecl.Instruction,
	textOf func(ecl.Instruction) []string, depth int) []textAt {
	seen := map[int]bool{start: true}
	frontier := []int{start}
	var out []textAt
	for distance := 1; distance <= depth && len(frontier) > 0; distance++ {
		var next []int
		for _, offset := range frontier {
			for _, previous := range predecessors[offset] {
				if seen[previous] {
					continue
				}
				seen[previous] = true
				next = append(next, previous)
				instruction, ok := byOffset[previous]
				if !ok {
					continue
				}
				for _, text := range textOf(instruction) {
					out = append(out, textAt{Distance: distance, Text: text})
				}
			}
		}
		frontier = next
	}
	return out
}

func appendUnique(list []string, value string) []string {
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}
