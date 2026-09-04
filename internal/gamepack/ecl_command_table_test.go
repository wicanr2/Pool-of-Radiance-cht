package gamepack

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

// 常數表要與真檔量出來的一致。多一條或少一條不一致都會紅。
func TestPoolCommandTableMatchesTheMeasuredChain(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	opcodes, err := ReadDOSECLOpcodeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	measured := ECLCommandTable(opcodes)
	constant := PoolCommandTable()
	if len(measured) != len(constant) {
		t.Fatalf("量出來 %d 條，常數表 %d 條", len(measured), len(constant))
	}
	for opcode, want := range measured {
		if got := constant[opcode]; got.Arity != want.Arity {
			t.Errorf("0x%02X 常數表寫 %d 個運算元，量出來是 %d", opcode, got.Arity, want.Arity)
		}
	}
	// 不一致的清單本身也釘住：只該有 `34h`。多出來的要先讀懂再放行。
	var differ []int
	for opcode, command := range measured {
		if base, ok := ecl.KnownCommands[opcode]; ok && base.Arity != command.Arity {
			differ = append(differ, int(opcode))
		}
	}
	sort.Ints(differ)
	if len(differ) != 1 || differ[0] != 0x34 {
		t.Errorf("與共用 engine 的二手表不一致的是 %v，目前只該有 0x34", differ)
	}
}

// 二十九個 ECL 區塊全部靜態走得完。
//
// 先前只走得完 26 個。三筆卡住的原因是兩種：`ECL5/7` 與 `ECL7/22` 走到
// `20h NEWECL` 之後**繼續往下讀**，而那後面放的是資料——NEWECL 換掉整個
// 程式碼段，後面的位元組永遠不會執行；`ECL7/17` 是 `34h ECL CLOCK` 的
// 運算元個數，二手表寫兩個而 Pool 的 overlay-03 量出來是一個。
func TestEveryECLBlockTracesEndToEnd(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	catalog, err := ReadDOSECLCatalog(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	commands := PoolCommandTable()
	blocks, instructions := 0, 0
	for number := uint8(1); number <= 8; number++ {
		archive, ok := catalog.Archive(number)
		if !ok {
			continue
		}
		ids := make([]int, 0, len(archive.Blocks))
		for id := range archive.Blocks {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, id := range ids {
			block := archive.Blocks[uint16(id)]
			blocks++
			points, _, err := ecl.EntryPoints(block, 5)
			if err != nil {
				t.Errorf("ecl%d/%d 沒有入口：%v", number, id, err)
				continue
			}
			starts := make([]int, len(points))
			for index, point := range points {
				starts[index] = int(point) - 0x9900
			}
			graph, err := ecl.TraceGraphAtBaseWithCommands(block, starts, 0x9900, len(block)*8, commands)
			if err != nil {
				t.Errorf("ecl%d/%d 追不動：%v", number, id, err)
				continue
			}
			instructions += len(graph.Instructions)
		}
	}
	if blocks != 29 {
		t.Fatalf("掃到 %d 個區塊，原版是 29 個", blocks)
	}
	if instructions != 16034 {
		t.Errorf("只走到 %d 條可達指令，先前量到 16034 條", instructions)
	}
	t.Logf("%d 個區塊全部走得完，共 %d 條可達指令", blocks, instructions)
}
