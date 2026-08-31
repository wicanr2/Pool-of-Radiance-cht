// Command pool-ecl-trace exports one original Pool ECL block's complete
// statically reachable graph without executing or assigning story semantics.
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
	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

const codeBase = 0x9900

type operandRow struct {
	Code      uint8  `json:"code"`
	Low       uint8  `json:"low"`
	Word      uint16 `json:"word,omitempty"`
	WordSet   bool   `json:"word_set,omitempty"`
	PackedHex string `json:"packed_hex,omitempty"`
	Text      string `json:"text,omitempty"`
}
type instructionRow struct {
	Offset          int          `json:"offset"`
	Address         string       `json:"address"`
	Opcode          uint8        `json:"opcode"`
	Name            string       `json:"name"`
	RecordEnd       int          `json:"record_end"`
	Operands        []operandRow `json:"operands,omitempty"`
	MenuDestination string       `json:"menu_destination,omitempty"`
	MenuOptions     []string     `json:"menu_options,omitempty"`
}
type report struct {
	Schema          string           `json:"schema"`
	ZIP             string           `json:"zip"`
	ZIPSHA256       string           `json:"zip_sha256"`
	Member          string           `json:"member"`
	MemberSHA256    string           `json:"member_sha256"`
	BlockID         uint8            `json:"block_id"`
	BlockSHA256     string           `json:"block_sha256"`
	CodeAddressBase string           `json:"code_address_base"`
	EntryAddresses  []string         `json:"entry_addresses"`
	Instructions    []instructionRow `json:"instructions"`
	Edges           []ecl.Edge       `json:"edges"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "original DOS ZIP")
	archive := flag.Int("archive", 3, "ECL archive 1..8")
	block := flag.Int("block", 0, "original DAX block ID")
	entry := flag.Int("entry", -1, "optional command-set entry index 0..4; default traces all")
	out := flag.String("out", "", "JSON output; stdout when empty")
	flag.Parse()
	r, err := trace(*zipPath, *archive, *block, *entry)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(b)
	} else {
		err = os.WriteFile(*out, b, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func trace(zipPath string, archiveNumber, blockID, entryIndex int) (report, error) {
	if archiveNumber < 1 || archiveNumber > 8 || blockID < 0 || blockID > 255 {
		return report{}, fmt.Errorf("archive/block outside byte range")
	}
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		return report{}, err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer zr.Close()
	want := fmt.Sprintf("ECL%d.DAX", archiveNumber)
	var member *zip.File
	for _, f := range zr.File {
		if strings.EqualFold(filepath.Base(f.Name), want) {
			if member != nil {
				return report{}, fmt.Errorf("duplicate %s", want)
			}
			member = f
		}
	}
	if member == nil {
		return report{}, fmt.Errorf("missing %s", want)
	}
	stream, err := member.Open()
	if err != nil {
		return report{}, err
	}
	memberBytes, err := io.ReadAll(io.LimitReader(stream, 16<<20))
	closeErr := stream.Close()
	if err != nil {
		return report{}, err
	}
	if closeErr != nil {
		return report{}, closeErr
	}
	blocks, err := dax.Parse(memberBytes)
	if err != nil {
		return report{}, err
	}
	var selected []byte
	for _, block := range blocks {
		if int(block.Entry.ID) == blockID {
			selected = block.Data
			break
		}
	}
	if selected == nil {
		return report{}, fmt.Errorf("%s has no block %d", want, blockID)
	}
	points, _, err := ecl.EntryPoints(selected, 5)
	if err != nil {
		return report{}, err
	}
	starts := make([]int, len(points))
	entryAddresses := make([]string, len(points))
	for i, p := range points {
		starts[i] = int(p) - codeBase
		entryAddresses[i] = fmt.Sprintf("0x%04X", p)
	}
	if entryIndex >= 0 {
		if entryIndex >= len(starts) {
			return report{}, fmt.Errorf("entry index %d is outside 0..%d", entryIndex, len(starts)-1)
		}
		starts = []int{starts[entryIndex]}
		entryAddresses = []string{entryAddresses[entryIndex]}
	}
	graph, err := ecl.TraceGraphAtBase(selected, starts, codeBase, len(selected)*8)
	if err != nil {
		return report{}, err
	}
	rows := make([]instructionRow, 0, len(graph.Instructions))
	for _, ins := range graph.Instructions {
		end, err := ecl.RecordEnd(selected, ins.Offset)
		if err != nil {
			return report{}, err
		}
		row := instructionRow{Offset: ins.Offset, Address: fmt.Sprintf("0x%04X", codeBase+ins.Offset), Opcode: ins.Command.Opcode, Name: ins.Command.Name, RecordEnd: end}
		if ins.Command.Opcode == 0x15 || ins.Command.Opcode == 0x2B {
			menu, err := ecl.DecodeMenuRecord(selected, ins.Offset)
			if err != nil {
				return report{}, err
			}
			destination, err := ecl.WordAddress(menu.Header[0])
			if err != nil {
				return report{}, fmt.Errorf("menu destination at 0x%04X: %w", codeBase+ins.Offset, err)
			}
			row.MenuDestination = fmt.Sprintf("0x%04X", destination)
			row.MenuOptions = menu.OptionTexts
		}
		for _, op := range ins.Operands {
			operand := operandRow{Code: op.Code, Low: op.Low, Word: op.Word, WordSet: op.WordSet, PackedHex: hex.EncodeToString(op.Packed)}
			if op.Code == 0x80 {
				operand.Text, _ = ecl.TextValue(op, nil)
			}
			row.Operands = append(row.Operands, operand)
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Offset < rows[j].Offset })
	sort.Slice(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From != graph.Edges[j].From {
			return graph.Edges[i].From < graph.Edges[j].From
		}
		if graph.Edges[i].To != graph.Edges[j].To {
			return graph.Edges[i].To < graph.Edges[j].To
		}
		return graph.Edges[i].Kind < graph.Edges[j].Kind
	})
	return report{Schema: "pool-ecl-static-trace-v1", ZIP: filepath.Base(zipPath), ZIPSHA256: digest(zipBytes), Member: member.Name, MemberSHA256: digest(memberBytes), BlockID: uint8(blockID), BlockSHA256: digest(selected), CodeAddressBase: "0x9900", EntryAddresses: entryAddresses, Instructions: rows, Edges: graph.Edges}, nil
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
