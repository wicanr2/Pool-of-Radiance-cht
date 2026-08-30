// Command pool-ecl-audit measures the reusable ECL decoder against every
// Pool ECL block. Static reachability is not title behavior parity.
package main

import (
	"archive/zip"
	"encoding/binary"
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

// poolCodeAddressBase is the repeated payload-zero entry address in the fixed
// DOS corpus. It is title data and must not be moved into the shared engine.
const poolCodeAddressBase = 0x9914

type blockResult struct {
	File           string   `json:"file"`
	BlockID        uint8    `json:"block_id"`
	Bytes          int      `json:"bytes"`
	PrefixWord     uint16   `json:"prefix_word"`
	EntryPoints    int      `json:"entry_points,omitempty"`
	EntryAddresses []uint16 `json:"entry_addresses,omitempty"`
	Instructions   int      `json:"instructions,omitempty"`
	Error          string   `json:"error,omitempty"`
}

type report struct {
	ZIP              string        `json:"zip"`
	Blocks           int           `json:"blocks"`
	DecodedBlocks    int           `json:"decoded_blocks"`
	FailedBlocks     int           `json:"failed_blocks"`
	EntryPoints      int           `json:"entry_points"`
	Instructions     int           `json:"instructions"`
	ReachableOpcodes []int         `json:"reachable_opcodes"`
	BlockResults     []blockResult `json:"block_results"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	flag.Parse()
	result, err := audit(*zipPath)
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
	if result.FailedBlocks != 0 {
		os.Exit(2)
	}
}

func audit(zipPath string) (report, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()
	result := report{ZIP: filepath.Base(zipPath)}
	opcodes := map[uint8]bool{}
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
			row := blockResult{File: member.Name, BlockID: block.Entry.ID, Bytes: len(block.Data)}
			if len(block.Data) >= 2 {
				row.PrefixWord = binary.LittleEndian.Uint16(block.Data[:2])
			}
			result.Blocks++
			points, _, err := ecl.EntryPoints(block.Data, 5)
			if err != nil {
				row.Error = err.Error()
			} else {
				row.EntryPoints = len(points)
				row.EntryAddresses = points
				starts := make([]int, 0, len(points))
				for _, point := range points {
					starts = append(starts, int(point)-poolCodeAddressBase)
				}
				graph, graphErr := ecl.TraceGraphAtBase(block.Data, starts, poolCodeAddressBase, len(block.Data)*8)
				if graphErr != nil {
					row.Error = graphErr.Error()
				} else {
					row.Instructions = len(graph.Instructions)
					result.EntryPoints += len(points)
					result.Instructions += len(graph.Instructions)
					result.DecodedBlocks++
					for _, instruction := range graph.Instructions {
						opcodes[instruction.Command.Opcode] = true
					}
				}
			}
			if row.Error != "" {
				result.FailedBlocks++
			}
			result.BlockResults = append(result.BlockResults, row)
		}
	}
	for opcode := range opcodes {
		result.ReachableOpcodes = append(result.ReachableOpcodes, int(opcode))
	}
	sort.Slice(result.ReachableOpcodes, func(i, j int) bool { return result.ReachableOpcodes[i] < result.ReachableOpcodes[j] })
	sort.Slice(result.BlockResults, func(i, j int) bool {
		if result.BlockResults[i].File != result.BlockResults[j].File {
			return result.BlockResults[i].File < result.BlockResults[j].File
		}
		return result.BlockResults[i].BlockID < result.BlockResults[j].BlockID
	})
	return result, nil
}
