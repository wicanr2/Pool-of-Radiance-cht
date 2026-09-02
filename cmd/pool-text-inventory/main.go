// Command pool-text-inventory 盤點原版 ECL 裡所有玩家看得到的敘述文字。
//
// 文字是 6-bit packed 的 0x80 運算元。掃描 0x80 位元組會從錯的位置起解，
// 解出來的字串前面掛著一段亂碼——看起來像文字，其實不是原版的任何一句話。
// 所以這裡改成從每個 block 的進入點做控制流追蹤，只取真正是指令運算元的文字：
// 位置是指令位址，內容是原版真的會顯示的那一句。
//
// 同一段文字會在多個 block 重複，因此以文字本身去重並記下每一處位置。
// 輸出同時是翻譯用的樣板——有了它才知道「全部要翻多少」，而不是邊玩邊發現。
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
)

// poolCodeAddressBase 是 Pool 解出來的 payload 第 0 個位元組對應的位址
// （spec 002）。指令位址寫成這個基底加上位移，與其他 audit 一致。
const poolCodeAddressBase = 0x9900

type location struct {
	File    string `json:"file"`
	BlockID uint8  `json:"block_id"`
	Address uint16 `json:"address"`
	Opcode  uint8  `json:"opcode"`
}

type entry struct {
	Source    string     `json:"source"`
	Length    int        `json:"length"`
	Locations []location `json:"locations"`
}

type report struct {
	ZIP          string   `json:"zip"`
	Archives     int      `json:"archives"`
	Blocks       int      `json:"blocks"`
	TracedBlocks int      `json:"traced_blocks"`
	FailedBlocks []string `json:"failed_blocks,omitempty"`
	TextOperands int      `json:"text_operands"`
	Unique       int      `json:"unique_strings"`
	Characters   int      `json:"characters"`
	Entries      []entry  `json:"entries"`
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
}

func inventory(zipPath string) (report, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()

	result := report{ZIP: filepath.Base(zipPath)}
	seen := map[string][]location{}
	names := make([]string, 0, len(archive.File))
	members := map[string]*zip.File{}
	for _, member := range archive.File {
		names = append(names, member.Name)
		members[member.Name] = member
	}
	sort.Strings(names)

	for _, name := range names {
		base := strings.ToLower(filepath.Base(name))
		if !strings.HasPrefix(base, "ecl") || !strings.EqualFold(filepath.Ext(name), ".dax") {
			continue
		}
		stream, err := members[name].Open()
		if err != nil {
			return report{}, err
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, 64<<20))
		closeErr := stream.Close()
		if readErr != nil {
			return report{}, readErr
		}
		if closeErr != nil {
			return report{}, closeErr
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return report{}, fmt.Errorf("%s: %w", name, err)
		}
		result.Archives++
		for _, block := range blocks {
			result.Blocks++
			points, _, err := ecl.EntryPoints(block.Data, 5)
			if err != nil {
				result.FailedBlocks = append(result.FailedBlocks,
					fmt.Sprintf("%s block %d: %v", base, block.Entry.ID, err))
				continue
			}
			starts := make([]int, 0, len(points))
			for _, point := range points {
				starts = append(starts, int(point)-poolCodeAddressBase)
			}
			graph, graphErr := ecl.TraceGraphAtBase(block.Data, starts, poolCodeAddressBase, len(block.Data)*8)
			if graphErr != nil {
				// 追蹤中斷仍保留已走到的指令：少一段比整個 block 掛零好，
				// 而且掛零會被誤讀成「這個 block 沒有文字」。
				result.FailedBlocks = append(result.FailedBlocks,
					fmt.Sprintf("%s block %d: %v", base, block.Entry.ID, graphErr))
			} else {
				result.TracedBlocks++
			}
			for _, instruction := range graph.Instructions {
				where := location{
					File: base, BlockID: block.Entry.ID,
					Address: uint16(poolCodeAddressBase + instruction.Offset),
					Opcode:  instruction.Command.Opcode,
				}
				record := func(text string) {
					if text == "" {
						return
					}
					result.TextOperands++
					seen[text] = append(seen[text], where)
				}
				for _, operand := range instruction.Operands {
					if operand.Code != 0x80 {
						continue
					}
					record(ecl.DecodePackedText(operand.Packed))
				}
				// 選單的選項字串不在 Operands 裡，而在選單記錄裡。只掃 Operands
				// 會漏掉「PRESS <RETURN> OR BUTTON TO CONTINUE」這一類玩家一定
				// 看得到的字，而漏掉的形狀是「盤點檔裡沒有」，跟「原版沒有」
				// 分不出來。
				if instruction.Command.Opcode != 0x15 && instruction.Command.Opcode != 0x2B {
					continue
				}
				menu, menuErr := ecl.DecodeMenuRecord(block.Data, instruction.Offset)
				if menuErr != nil {
					result.FailedBlocks = append(result.FailedBlocks,
						fmt.Sprintf("%s block %d menu at %04X: %v", base, block.Entry.ID, where.Address, menuErr))
					continue
				}
				for _, text := range menu.OptionTexts {
					record(text)
				}
			}
		}
	}

	sources := make([]string, 0, len(seen))
	for source := range seen {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	for _, source := range sources {
		result.Entries = append(result.Entries, entry{
			Source: source, Length: len(source), Locations: seen[source],
		})
		result.Characters += len(source)
	}
	result.Unique = len(result.Entries)
	return result, nil
}
