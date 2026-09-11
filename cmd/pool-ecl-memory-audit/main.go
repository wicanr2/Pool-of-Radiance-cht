// Command pool-ecl-memory-audit inventories raw ECL operand references to
// selected runtime addresses across every Pool ECL archive. It deliberately
// reports operand positions and opcodes without assigning story semantics.
//
// 兩份用它產的清冊：spec 038（墓園委託旗標的 producer）與 spec 039（七種
// 戰利品的累積池）。位址的語意在那兩份裡定，不在這支工具裡。
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
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

const codeBase = 0x9900

type reference struct {
	Archive     string `json:"archive"`
	BlockID     uint8  `json:"block_id"`
	Address     string `json:"address"`
	Opcode      uint8  `json:"opcode"`
	Name        string `json:"name"`
	Operand     int    `json:"operand"`
	OperandCode uint8  `json:"operand_code"`
	Target      string `json:"target"`
}
type failure struct {
	Archive string `json:"archive"`
	BlockID uint8  `json:"block_id"`
	Error   string `json:"error"`
}
type report struct {
	Schema     string      `json:"schema"`
	ZIP        string      `json:"zip"`
	ZIPSHA256  string      `json:"zip_sha256"`
	Targets    []string    `json:"targets"`
	References []reference `json:"references"`
	Failures   []failure   `json:"failures,omitempty"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	addresses := flag.String("addresses", "4AC1,4AB1,4A96", "comma-separated hexadecimal runtime addresses")
	out := flag.String("out", "", "JSON output; stdout when empty")
	flag.Parse()
	targets, err := parseTargets(*addresses)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result, err := audit(*zipPath, targets)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	raw = append(raw, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(raw)
	} else {
		err = os.WriteFile(*out, raw, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseTargets(value string) (map[uint16]bool, error) {
	targets := map[uint16]bool{}
	for _, field := range strings.Split(value, ",") {
		field = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(field), "0x"))
		number, err := strconv.ParseUint(field, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid ECL address %q", field)
		}
		targets[uint16(number)] = true
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no ECL addresses selected")
	}
	return targets, nil
}

func audit(zipPath string, targets map[uint16]bool) (report, error) {
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		return report{}, err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer zr.Close()
	result := report{Schema: "pool-ecl-memory-audit-v1", ZIP: filepath.Base(zipPath), ZIPSHA256: digest(zipBytes)}
	for target := range targets {
		result.Targets = append(result.Targets, fmt.Sprintf("0x%04X", target))
	}
	sort.Strings(result.Targets)
	for _, member := range zr.File {
		name := strings.ToUpper(filepath.Base(member.Name))
		if !strings.HasPrefix(name, "ECL") || filepath.Ext(name) != ".DAX" {
			continue
		}
		stream, err := member.Open()
		if err != nil {
			return report{}, err
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, 16<<20))
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
		for _, block := range blocks {
			points, _, err := ecl.EntryPoints(block.Data, 5)
			if err != nil {
				result.Failures = append(result.Failures, failure{Archive: name, BlockID: block.Entry.ID, Error: err.Error()})
				continue
			}
			starts := make([]int, len(points))
			for index, point := range points {
				starts[index] = int(point) - codeBase
			}
			graph, graphErr := ecl.TraceGraphAtBaseWithCommands(block.Data, starts, codeBase, len(block.Data)*8, gamepack.PoolCommandTable())
			if graphErr != nil {
				result.Failures = append(result.Failures, failure{Archive: name, BlockID: block.Entry.ID, Error: graphErr.Error()})
			}
			for _, instruction := range graph.Instructions {
				for operandIndex, operand := range instruction.Operands {
					if !operand.WordSet || !targets[operand.Word] {
						continue
					}
					result.References = append(result.References, reference{Archive: name, BlockID: block.Entry.ID, Address: fmt.Sprintf("0x%04X", codeBase+instruction.Offset), Opcode: instruction.Command.Opcode, Name: instruction.Command.Name, Operand: operandIndex, OperandCode: operand.Code, Target: fmt.Sprintf("0x%04X", operand.Word)})
				}
			}
		}
	}
	sort.Slice(result.References, func(i, j int) bool {
		a, b := result.References[i], result.References[j]
		if a.Archive != b.Archive {
			return a.Archive < b.Archive
		}
		if a.BlockID != b.BlockID {
			return a.BlockID < b.BlockID
		}
		if a.Address != b.Address {
			return a.Address < b.Address
		}
		return a.Operand < b.Operand
	})
	sort.Slice(result.Failures, func(i, j int) bool {
		if result.Failures[i].Archive != result.Failures[j].Archive {
			return result.Failures[i].Archive < result.Failures[j].Archive
		}
		return result.Failures[i].BlockID < result.Failures[j].BlockID
	})
	return result, nil
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
