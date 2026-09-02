// Command pool-ecl-frontier lists the ECL opcodes that appear in Pool blocks
// but have neither a core VM handler nor an adapter passthrough, with every
// call site. It answers "what is left before the main line runs end to end"
// as a measurement instead of a guess.
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

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

// poolCodeAddressBase 是 payload 位元組 0 的位址（spec 002）。
const poolCodeAddressBase = 0x9900

type site struct {
	File   string `json:"file"`
	Block  uint8  `json:"block_id"`
	Offset int    `json:"offset"`
}

type opcodeRow struct {
	Opcode string `json:"opcode"`
	Name   string `json:"name"`
	Arity  int    `json:"arity"`
	Sites  []site `json:"sites"`
}

type report struct {
	Schema     string      `json:"schema"`
	ZIP        string      `json:"zip"`
	Handled    []string    `json:"handled"`
	Unhandled  []opcodeRow `json:"unhandled"`
	SiteTotal  int         `json:"unhandled_site_total"`
	BlockTotal int         `json:"blocks_scanned"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	flag.Parse()
	result, err := scan(*zipPath)
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

func scan(zipPath string) (report, error) {
	handled := handledOpcodes()
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()
	result := report{Schema: "pool-ecl-opcode-frontier/1", ZIP: filepath.Base(zipPath)}
	for code := range handled {
		result.Handled = append(result.Handled, fmt.Sprintf("0x%02X", code))
	}
	sort.Strings(result.Handled)
	sites := map[byte][]site{}
	for _, member := range archive.File {
		base := strings.ToLower(filepath.Base(member.Name))
		if !strings.HasPrefix(base, "ecl") || filepath.Ext(base) != ".dax" {
			continue
		}
		stream, err := member.Open()
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
			return report{}, fmt.Errorf("%s: %w", member.Name, err)
		}
		for _, block := range blocks {
			points, _, err := ecl.EntryPoints(block.Data, 5)
			if err != nil {
				continue
			}
			result.BlockTotal++
			starts := make([]int, 0, len(points))
			for _, point := range points {
				starts = append(starts, int(point)-poolCodeAddressBase)
			}
			graph, _ := ecl.TraceGraphAtBase(block.Data, starts, poolCodeAddressBase, len(block.Data)*8)
			for _, instruction := range graph.Instructions {
				code := instruction.Command.Opcode
				if handled[code] {
					continue
				}
				sites[code] = append(sites[code], site{base, block.Entry.ID, instruction.Offset})
			}
		}
	}
	codes := make([]int, 0, len(sites))
	for code := range sites {
		codes = append(codes, int(code))
	}
	sort.Ints(codes)
	for _, code := range codes {
		list := sites[byte(code)]
		sort.Slice(list, func(i, j int) bool {
			if list[i].File != list[j].File {
				return list[i].File < list[j].File
			}
			if list[i].Block != list[j].Block {
				return list[i].Block < list[j].Block
			}
			return list[i].Offset < list[j].Offset
		})
		command := ecl.KnownCommands[byte(code)]
		result.Unhandled = append(result.Unhandled, opcodeRow{
			Opcode: fmt.Sprintf("0x%02X", code), Name: command.Name, Arity: command.Arity, Sites: list})
		result.SiteTotal += len(list)
	}
	return result, nil
}

// handledOpcodes 是「跑得過去」的集合：共用 VM 自己實作的，加上 Pool 這一側
// 宣告成 passthrough 的。少列一個會把已經接好的東西算進待辦，多列一個會讓
// 這份清單漏掉真的擋路的 opcode，所以兩邊都要照著來源列。
func handledOpcodes() map[byte]bool {
	handled := map[byte]bool{}
	// 共用 engine 的 eclvm.Machine 直接處理的（machine.go 的 switch）。
	for _, code := range []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D,
		0x20, 0x24, 0x25, 0x26, 0x27, 0x2A, 0x2B, 0x2F, 0x30, 0x35,
	} {
		handled[code] = true
	}
	for code := range gamepack.InitialEventPassthrough() {
		handled[code] = true
	}
	return handled
}
