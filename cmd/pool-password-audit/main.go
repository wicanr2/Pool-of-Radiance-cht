// Command pool-password-audit 盤點原版每一處 `10h INPUT STRING`（spec 087）：
// 玩家在哪裡被要求打字、問句是什麼、比對的答案是什麼。
//
// 原版要玩家翻說明書或記住幾十格前 NPC 說過的字（索寇要塞的亡魂、野外的密碼門）。
// remake 的作弊選單預設把答案附在問句後面（spec 141〈密語提示〉），所以要先知道
// 「一共有幾處、每一處解不解得出來」——解不出來的那幾處，提示會安靜地不出現，
// 而那和「這一處本來就沒有答案」長得一樣。
//
// 答案怎麼解由 `gamepack.InputAnswer` 決定，與遊戲裡顯示提示走的是同一支。
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

// poolCodeAddressBase 是 payload 第 0 個位元組對應的位址（spec 002）。
const poolCodeAddressBase = 0x9900

// promptWindow 是往回收問句時最多看幾條指令。問句由前面的 `12h PRINT` 給，
// 中間可能夾著清框、旗標判斷之類不吐字的指令。
const promptWindow = 12

type site struct {
	File        string `json:"file"`
	BlockID     uint8  `json:"block_id"`
	Address     uint16 `json:"address"`
	Destination uint16 `json:"destination"`
	Prompt      string `json:"prompt"`
	Answer      string `json:"answer,omitempty"`
}

type report struct {
	ZIP        string `json:"zip"`
	Sites      int    `json:"input_string_sites"`
	WithAnswer int    `json:"sites_with_answer"`
	Entries    []site `json:"entries"`
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
}

func audit(zipPath string) (report, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()

	result := report{ZIP: filepath.Base(zipPath), Entries: []site{}}
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
		data, err := read(members[name])
		if err != nil {
			return report{}, err
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return report{}, fmt.Errorf("%s: %w", name, err)
		}
		for _, block := range blocks {
			found, err := sitesIn(base, block.Entry.ID, block.Data)
			if err != nil {
				return report{}, err
			}
			result.Entries = append(result.Entries, found...)
		}
	}
	result.Sites = len(result.Entries)
	for _, entry := range result.Entries {
		if entry.Answer != "" {
			result.WithAnswer++
		}
	}
	return result, nil
}

func read(member *zip.File) ([]byte, error) {
	stream, err := member.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return io.ReadAll(io.LimitReader(stream, 64<<20))
}

// sitesIn 走一個 block 的控制流，找出每一處 `10h INPUT STRING`。
//
// 走控制流而不是掃位元組：`10h` 這個位元組在別的指令的運算元裡也會出現，掃出來的
// 那些位置會解出一整條看起來合理的假指令（CLAUDE.md §5）。
func sitesIn(file string, blockID uint8, block []byte) ([]site, error) {
	if len(block) < 2 {
		return nil, nil
	}
	payload := block[2:]
	commands := gamepack.PoolCommandTable()
	points, _, err := ecl.EntryPoints(block, 5)
	if err != nil {
		// 進入點讀不出來的 block 沒有可信的起點；照樣往下解只會得到假指令。
		return nil, nil
	}
	starts := make([]int, 0, len(points))
	for _, point := range points {
		starts = append(starts, int(point)-poolCodeAddressBase)
	}
	graph, _ := ecl.TraceGraphAtBaseWithCommands(block, starts, poolCodeAddressBase, len(block)*8, commands)
	instructions := append([]ecl.Instruction(nil), graph.Instructions...)
	sort.Slice(instructions, func(i, j int) bool { return instructions[i].Offset < instructions[j].Offset })

	decode := func(offset int) (ecl.Instruction, error) {
		return ecl.DecodeInstructionWithCommands(payload, offset, commands)
	}
	var found []site
	for index, instruction := range instructions {
		if instruction.Command.Opcode != gamepack.InputStringOpcode ||
			len(instruction.Operands) != gamepack.InputOperands {
			continue
		}
		destination, err := ecl.WordAddress(instruction.Operands[gamepack.InputDestinationOperand-1])
		if err != nil {
			return nil, fmt.Errorf("%s block %d at %04X: %w",
				file, blockID, poolCodeAddressBase+instruction.Offset, err)
		}
		found = append(found, site{
			File: file, BlockID: blockID,
			Address:     uint16(poolCodeAddressBase + instruction.Offset),
			Destination: destination,
			Prompt:      promptBefore(instructions, index),
			Answer:      gamepack.InputAnswer(decode, instruction.Next, destination),
		})
	}
	return found, nil
}

// promptBefore 把輸入前面那幾條指令印出來的字接起來，就是玩家看到的問句。
func promptBefore(instructions []ecl.Instruction, index int) string {
	var parts []string
	for back := index - 1; back >= 0 && index-back <= promptWindow; back-- {
		var line []string
		for _, operand := range instructions[back].Operands {
			if operand.Code != 0x80 {
				continue
			}
			if text := strings.TrimSpace(ecl.DecodePackedText(operand.Packed)); text != "" {
				line = append(line, text)
			}
		}
		if len(line) == 0 {
			continue
		}
		parts = append([]string{strings.Join(line, " ")}, parts...)
	}
	return strings.Join(parts, " ")
}
