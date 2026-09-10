package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

func rowFor(t *testing.T, opcodes []entry, code string) entry {
	t.Helper()
	for _, row := range opcodes {
		if row.Opcode == code {
			return row
		}
	}
	t.Fatalf("報告裡沒有 %s", code)
	return entry{}
}

// 兩邊的運算元個數不同才叫分歧。這一條要正反都驗：一致的不能報、
// 不一致的一定要報——只驗一半的話「永遠不報」與「報對了」長得一樣。
func TestReportFlagsOnlyRealArityDisagreements(t *testing.T) {
	// 挑一條 engine 表裡有、而且不是可變長度的當樣本。
	var sample byte
	var arity int
	for code, command := range ecl.KnownCommands {
		if _, variable := ecl.VariableLengthCommands[code]; variable {
			continue
		}
		sample, arity = code, command.Arity
		break
	}
	if sample == 0 && arity == 0 {
		t.Skip("engine 的 KnownCommands 是空的")
	}
	agree := buildReport([]gamepack.ECLOpcode{{Opcode: sample, Handler: 0x1234, Operands: arity}})
	if row := agree.Opcodes[0]; row.Disagrees {
		t.Fatalf("兩邊都是 %d 個運算元卻報成分歧：%+v", arity, row)
	}
	disagree := buildReport([]gamepack.ECLOpcode{{Opcode: sample, Handler: 0x1234, Operands: arity + 1}})
	if row := disagree.Opcodes[0]; !row.Disagrees {
		t.Fatalf("掃出 %d、engine 說 %d，卻沒報分歧：%+v", arity+1, arity, row)
	}
	if row := disagree.Opcodes[0]; row.EngineArity == nil || *row.EngineArity != arity {
		t.Fatalf("engine 那一側的值沒有一起帶出來：%+v", row)
	}
}

// 可變長度那四條的 arity 對不起來是**正常的**——尾巴長度由資料決定。
// 把它們報成分歧會把真正的分歧淹掉。
func TestVariableLengthCommandsNeverCountAsDisagreements(t *testing.T) {
	if len(ecl.VariableLengthCommands) == 0 {
		t.Skip("engine 沒有列可變長度指令")
	}
	for code := range ecl.VariableLengthCommands {
		command, ok := ecl.KnownCommands[code]
		if !ok {
			t.Fatalf("可變長度的 %#02x 不在 KnownCommands 裡", code)
		}
		report := buildReport([]gamepack.ECLOpcode{
			{Opcode: code, Handler: 0x2000, Operands: command.Arity + 3},
		})
		row := report.Opcodes[0]
		if !row.Variable {
			t.Fatalf("%#02x 沒被標成可變長度：%+v", code, row)
		}
		if row.Disagrees {
			t.Fatalf("%#02x 是可變長度卻報成分歧：%+v", code, row)
		}
	}
}

// engine 表裡沒有的 opcode 是「沒得比」，不是「一致」。兩者在 JSON 裡靠
// `engine_arity` 在不在分辨，所以那一欄必須留空。
func TestUnknownOpcodesCarryNoEngineArity(t *testing.T) {
	var unknown byte = 0xFF
	for code := byte(0xFF); code > 0; code-- {
		if _, ok := ecl.KnownCommands[code]; !ok {
			unknown = code
			break
		}
	}
	report := buildReport([]gamepack.ECLOpcode{{Opcode: unknown, Handler: 0x30, Operands: 2}})
	row := report.Opcodes[0]
	if row.EngineArity != nil || row.Name != "" || row.Disagrees {
		t.Fatalf("engine 沒有 %#02x，報告卻填了東西：%+v", unknown, row)
	}
	if report.OpcodeCount != 1 {
		t.Fatalf("opcode_count 是 %d", report.OpcodeCount)
	}
}
