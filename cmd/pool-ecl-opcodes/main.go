// pool-ecl-opcodes 把 overlay-03 的 ECL 派發鏈 dump 成 JSON：每條 opcode 的
// 處理常式位移與運算元個數，並標出與共用 engine 那張二手 arity 表的差異。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/golden-box-remake-engine/ecl"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

type report struct {
	Schema        string  `json:"schema"`
	Overlay       int     `json:"overlay"`
	DispatchChain string  `json:"dispatch_chain"`
	OperandFetch  string  `json:"operand_fetch"`
	OpcodeCount   int     `json:"opcode_count"`
	Opcodes       []entry `json:"opcodes"`
}

type entry struct {
	Opcode      string `json:"opcode"`
	Name        string `json:"name,omitempty"`
	Handler     string `json:"handler"`
	Operands    int    `json:"operands"`
	EngineArity *int   `json:"engine_arity,omitempty"`
	Variable    bool   `json:"variable_length,omitempty"`
	Disagrees   bool   `json:"disagrees,omitempty"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "original DOS ZIP")
	outPath := flag.String("out", "", "JSON output; stdout when empty")
	flag.Parse()
	if err := run(*zipPath, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(zipPath, outPath string) error {
	table, err := gamepack.ReadDOSECLOpcodeTable(zipPath)
	if err != nil {
		return err
	}
	r := report{
		Schema:        "pool-ecl-opcode-operands-v1",
		Overlay:       gamepack.ECLDispatchOverlay,
		DispatchChain: "overlay-03 的 cmp ax,opcode / jne / push cs / call 鏈",
		OperandFetch:  "mov al,N / push ax / lcall 0045h:002Ah",
		OpcodeCount:   len(table),
	}
	for _, item := range table {
		row := entry{Opcode: fmt.Sprintf("%#02x", item.Opcode),
			Handler: fmt.Sprintf("%#04x", item.Handler), Operands: item.Operands}
		if command, ok := ecl.KnownCommands[item.Opcode]; ok {
			row.Name = command.Name
			arity := command.Arity
			row.EngineArity = &arity
			_, row.Variable = ecl.VariableLengthCommands[item.Opcode]
			row.Disagrees = !row.Variable && arity != item.Operands
		}
		r.Opcodes = append(r.Opcodes, row)
	}
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if outPath == "" {
		_, err = os.Stdout.Write(encoded)
		return err
	}
	return os.WriteFile(outPath, encoded, 0o644)
}
