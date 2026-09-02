package gamepack_test

import (
	"bytes"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 選單的四個選項就是 overlay-03 `29h` 常式前面那兩段字面常數。字串在原版的
// 位元組裡，所以這一條同時鎖住「我們讀的是哪一段碼」。
func TestEncounterMenuStringsAreInTheOriginalOverlay(t *testing.T) {
	overlay, err := gamepack.ReadDOSOverlayCode(dosZIP, gamepack.ECLDispatchOverlay)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, want := range []string{
		"~COMBAT ~WAIT ~FLEE ~PARLAY",
		"~COMBAT ~WAIT ~FLEE ~ADVANCE",
		"Both sides wait.",
		"The monsters flee.",
	} {
		if !bytes.Contains(overlay, []byte(want)) {
			t.Fatalf("overlay-03 does not contain %q", want)
		}
	}
}

// `29h` 在派發鏈裡吃 14 個運算元，與常數宣告一致。
func TestEncounterMenuOperandCountMatchesTheDispatchChain(t *testing.T) {
	table, err := gamepack.ReadDOSECLOpcodeTable(dosZIP)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	for _, entry := range table {
		if entry.Opcode != gamepack.EncounterMenuOpcode {
			continue
		}
		if entry.Operands != gamepack.EncounterMenuOperands {
			t.Fatalf("opcode 0x29 takes %d operands, want %d", entry.Operands, gamepack.EncounterMenuOperands)
		}
		return
	}
	t.Fatal("opcode 0x29 is absent from the dispatch chain")
}

// 三個運算元陣列的基底彼此相差固定值：型別、低位、高位各一張。
func TestECLOperandArraysAreParallel(t *testing.T) {
	if gamepack.ECLOperandLowBase-gamepack.ECLOperandHighBase != 0x40 {
		t.Fatalf("low base %#04x and high base %#04x are not 0x40 apart",
			gamepack.ECLOperandLowBase, gamepack.ECLOperandHighBase)
	}
	if gamepack.ECLOperandHighBase-gamepack.ECLOperandTypeBase != 0x40 {
		t.Fatalf("high base %#04x and type base %#04x are not 0x40 apart",
			gamepack.ECLOperandHighBase, gamepack.ECLOperandTypeBase)
	}
}
