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

// buildReport 把掃出來的表對上共用 engine 那張**二手** arity 表。
//
// 判準只有一條：`Disagrees = 不是可變長度 && 兩邊的運算元個數不同`。
// 可變長度那四條（`VERTICAL MENU`、`ON GOTO`、`ON GOSUB`、`HORIZONTAL MENU`）
// 的 arity 本來就對不起來——它們的尾巴長度由資料決定，拿固定值去比一定不同，
// 那個「不同」沒有意義，報出來只會把真正的分歧淹掉。
//
// engine 表裡沒有的 opcode 不比：沒有第二個來源可以對，`Disagrees` 留 false
// 不是「一致」，是「沒得比」——那兩件事在 JSON 裡由 `engine_arity` 在不在
// 分辨。
func buildReport(table []gamepack.ECLOpcode) report {
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
	return r
}

func run(zipPath, outPath string) error {
	table, err := gamepack.ReadDOSECLOpcodeTable(zipPath)
	if err != nil {
		return err
	}
	r := buildReport(table)
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
