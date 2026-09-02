package gamepack_test

import (
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/ecl"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 派發鏈解得出 55 條 opcode，而且已知的幾條個數要對。
func TestECLOpcodeTableMatchesKnownArities(t *testing.T) {
	table, err := gamepack.ReadDOSECLOpcodeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	if len(table) != 55 {
		t.Fatalf("the dispatch chain has %d opcodes, want 55", len(table))
	}
	operands := make(map[byte]int, len(table))
	for _, entry := range table {
		operands[entry.Opcode] = entry.Operands
	}
	for opcode, want := range map[byte]int{
		0x01: 1, 0x03: 2, 0x0C: 3, 0x14: 4, 0x1E: 6,
		0x27: 8, 0x28: 3, 0x29: 14, 0x2C: 6, 0x2E: 5,
		0x31: 0, 0x3D: 0,
	} {
		if operands[opcode] != want {
			t.Fatalf("opcode 0x%02X takes %d operands, want %d", opcode, operands[opcode], want)
		}
	}
}

// 與共用 engine 的 arity 表逐條比對。engine 那張表是二手的（公開 ECL dump 加
// CoAB 重製），這裡量的是 Pool 自己的位元組，所以不一致要看得見。
//
// 四條長度可變的指令（VERTICAL MENU、ON GOTO、ON GOSUB、HORIZONTAL MENU）
// engine 記 0 並改用 RecordEnd 算結尾，量到的是它們的固定前綴，不算衝突。
func TestECLOpcodeTableAgreesWithTheEngineExceptTheKnownGaps(t *testing.T) {
	table, err := gamepack.ReadDOSECLOpcodeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	// 0x34 ECL CLOCK 是唯一一條真的對不上的：Pool 的常式只取一個運算元，
	// engine 表寫 2。改 engine 會動到 CoAB，所以先鎖在這裡，別讓它悄悄變。
	known := map[byte]int{0x34: 1}
	for _, entry := range table {
		command, ok := ecl.KnownCommands[entry.Opcode]
		if !ok {
			continue
		}
		if _, variable := ecl.VariableLengthCommands[entry.Opcode]; variable {
			continue
		}
		want := command.Arity
		if override, ok := known[entry.Opcode]; ok {
			want = override
		}
		if entry.Operands != want {
			t.Fatalf("opcode 0x%02X (%s) takes %d operands in overlay-03 but %d in the engine table",
				entry.Opcode, command.Name, entry.Operands, command.Arity)
		}
	}
}
