package combat

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// scriptedDice 依序吐出 queue；用完之後交給 fallback。每一次呼叫都記下來，
// 用來核對擲骰的次數與順序（overlay-24 entry 8 的 count／sides）。
type scriptedDice struct {
	queue    []int
	fallback func(count, sides int) int
	calls    [][2]int
}

func (s *scriptedDice) roll(count, sides int) int {
	s.calls = append(s.calls, [2]int{count, sides})
	if len(s.queue) > 0 {
		value := s.queue[0]
		s.queue = s.queue[1:]
		return value
	}
	if s.fallback != nil {
		return s.fallback(count, sides)
	}
	return count * sides
}

func freshOutdoorCanvas() *outdoorCanvas {
	return &outdoorCanvas{terrain: NewOutdoorTacticalGrid().Terrain, classes: gamepack.OriginalCombatCellClassTable()}
}

func cellAt(c *outdoorCanvas, x, y int) uint8 { return c.terrain[y*TacticalRowStride+x] }

// `08B4h` 是先命中先算：3 同時落在第 4 條（→03）與第 5 條（→02）的區間裡，
// 答案是 03。沒有任何一條的地形碼原版回的是未初始化的區域變數，這裡要報錯。
func TestOutdoorTerrainFlagsFirstMatchWins(t *testing.T) {
	cases := map[uint8]uint8{0x01: 0x01, 0x03: 0x03, 0x0A: 0x02, 0x4E: 0x20, 0x6D: 0x80, 0x57: 0x88, 0xEC: 0xA0, 0x7C: 0x08}
	for code, want := range cases {
		got, err := OutdoorTerrainFlags(code, false)
		if err != nil || got != want {
			t.Errorf("08B4h(%02Xh) = %02Xh, %v; want %02Xh", code, got, err, want)
		}
	}
	for _, code := range []uint8{0x00, 0x05, 0xFF} {
		if _, err := OutdoorTerrainFlags(code, false); err == nil {
			t.Errorf("08B4h(%02Xh) matched nothing in the original but returned no error", code)
		}
	}
}

// `0C02h..0C21h`：`@4AB3 == FFh` 時帶 80h 的旗標去掉 80h 再或上 1。
func TestOutdoorTerrainFlagsRiverSwap(t *testing.T) {
	for code, want := range map[uint8]uint8{0x6D: 0x01, 0x6E: 0x01, 0x57: 0x09, 0x66: 0x11, 0xE1: 0x21, 0x01: 0x01} {
		got, err := OutdoorTerrainFlags(code, true)
		if err != nil || got != want {
			t.Errorf("08B4h(%02Xh) with @4AB3=FFh = %02Xh, %v; want %02Xh", code, got, err, want)
		}
	}
}

// `0C7Ah`：起點 34 − 5d4 退到 (x+2) 是 7 的倍數，每列往右一格；整條沒交叉時
// 在 16 − 1d9 那一列補兩列交叉。擲骰順序：1d100、5d4、每列一次 1d20、1d9。
func TestOutdoorBandDiagonalAndFallbackCrossing(t *testing.T) {
	canvas := freshOutdoorCanvas()
	dice := &scriptedDice{queue: []int{10, 10}, fallback: func(count, sides int) int {
		if sides == 9 {
			return 5
		}
		return 5
	}}
	outdoorBand(canvas, OutdoorFlag20, dice.roll)
	if canvas.err != nil {
		t.Fatal(canvas.err)
	}
	// 34 − 10 = 24 → 退到 19。
	if cellAt(canvas, 19, 0) != 0x32 || cellAt(canvas, 20, 0) != 0x33 {
		t.Fatalf("row 0 band = %02X %02X at x 19..20, want 32 33", cellAt(canvas, 19, 0), cellAt(canvas, 20, 0))
	}
	if cellAt(canvas, 43, 24) != 0x32 || cellAt(canvas, 44, 24) != 0x33 {
		t.Fatalf("row 24 band = %02X %02X at x 43..44, want 32 33", cellAt(canvas, 43, 24), cellAt(canvas, 44, 24))
	}
	// 補的兩列：y = 12 − 5 + 4 = 11，x = 11 + 19 = 30。
	for _, want := range [][3]int{{30, 11, 0x34}, {31, 11, 0x35}, {31, 12, 0x34}, {32, 12, 0x35}} {
		if got := cellAt(canvas, want[0], want[1]); got != uint8(want[2]) {
			t.Errorf("(%d,%d) = %02Xh, want %02Xh", want[0], want[1], got, want[2])
		}
	}
	wantCalls := 2 + 25 + 1
	if len(dice.calls) != wantCalls || dice.calls[0] != [2]int{1, 100} || dice.calls[1] != [2]int{5, 4} ||
		dice.calls[2] != [2]int{1, 20} || dice.calls[wantCalls-1] != [2]int{1, 9} {
		t.Fatalf("dice calls = %v, want 1d100, 5d4, 25×1d20, 1d9", dice.calls)
	}
}

// 1d20 擲出 1 的那一列交叉，下一列因計數是奇數再交叉一次；有交叉就不補。
func TestOutdoorBandCrossesInPairs(t *testing.T) {
	canvas := freshOutdoorCanvas()
	row := 0
	dice := &scriptedDice{queue: []int{1, 10}, fallback: func(count, sides int) int {
		if sides == 20 {
			row++
			if row == 4 {
				return 1
			}
			return 7
		}
		t.Fatalf("unexpected %dd%d after the band loop", count, sides)
		return 0
	}}
	outdoorBand(canvas, 0x00, dice.roll)
	// 旗標 0 門檻 0，但佇列給 1……1d100 永遠 ≥ 1，所以 1 > 0 整支不畫。
	if len(dice.calls) != 1 {
		t.Fatalf("threshold 0 should stop after the first 1d100, got %v", dice.calls)
	}
	canvas = freshOutdoorCanvas()
	row = 0
	dice = &scriptedDice{queue: []int{0x4B, 10}, fallback: dice.fallback}
	outdoorBand(canvas, OutdoorFlag10|OutdoorFlag20, dice.roll)
	if canvas.err != nil {
		t.Fatal(canvas.err)
	}
	for y := 0; y < 25; y++ {
		x := 19 + y
		if x > TacticalMaxX {
			continue
		}
		want := uint8(0x32)
		if y == 3 || y == 4 {
			want = 0x34
		}
		if got := cellAt(canvas, x, y); got != want {
			t.Errorf("row %d at x %d = %02Xh, want %02Xh", y, x, got, want)
		}
	}
}

// 起點 26 時第 23 列落在 x = 49，右半格照 `y×32h + x` 寫進第 24 列的開頭。
func TestOutdoorBandSpillsIntoTheNextRow(t *testing.T) {
	canvas := freshOutdoorCanvas()
	dice := &scriptedDice{queue: []int{1, 6}, fallback: func(count, sides int) int { return 7 }}
	outdoorBand(canvas, OutdoorFlag20, dice.roll)
	if canvas.err != nil {
		t.Fatal(canvas.err)
	}
	if cellAt(canvas, 49, 23) != 0x32 || cellAt(canvas, 0, 24) != 0x33 {
		t.Fatalf("(49,23) = %02Xh, (0,24) = %02Xh; want 32 then 33", cellAt(canvas, 49, 23), cellAt(canvas, 0, 24))
	}
}

// `0DB0h`：兩格都是平地才擲；1d10 大於 8 改擲 1d2+4，否則 (v+1)/2；
// 高度 5、6 只有下半格。
func TestOutdoorUprights(t *testing.T) {
	canvas := freshOutdoorCanvas()
	dice := &scriptedDice{queue: []int{7, 3, 3, 9, 2}, fallback: func(count, sides int) int { return 100 }}
	outdoorUprights(canvas, OutdoorFlag08, dice.roll)
	// (0,1)：1d100 = 7 ≤ 7，1d10 = 3 → 2：下半 25h、上半 21h。
	if cellAt(canvas, 0, 1) != 0x25 || cellAt(canvas, 0, 0) != 0x21 {
		t.Fatalf("(0,0..1) = %02X %02X, want 21 25", cellAt(canvas, 0, 0), cellAt(canvas, 0, 1))
	}
	// (0,2)：上一格已經不是平地，不擲。(0,3)：1d100 = 3，1d10 = 9 → 1d2+4 = 6：只有下半 29h。
	if cellAt(canvas, 0, 3) != 0x29 || cellAt(canvas, 0, 2) != OpenGroundCellClass {
		t.Fatalf("(0,2..3) = %02X %02X, want 17 29", cellAt(canvas, 0, 2), cellAt(canvas, 0, 3))
	}
	// 其餘 50×24 − 3 對各擲一次 1d100。
	opens := 50*24 - 3
	if len(dice.calls) != opens+2+2 {
		t.Fatalf("dice calls = %d, want %d", len(dice.calls), opens+4)
	}
	// 旗標 40h：密度 4，高度改擲 1d3+3。
	canvas = freshOutdoorCanvas()
	dice = &scriptedDice{queue: []int{4, 2, 1}, fallback: func(count, sides int) int { return 100 }}
	outdoorUprights(canvas, OutdoorFlag40, dice.roll)
	if cellAt(canvas, 0, 1) != 0x27 || cellAt(canvas, 0, 0) != 0x23 {
		t.Fatalf("40h (0,0..1) = %02X %02X, want 23 27", cellAt(canvas, 0, 0), cellAt(canvas, 0, 1))
	}
	if dice.calls[2] != [2]int{1, 3} {
		t.Fatalf("40h should reroll the height with 1d3, calls %v", dice.calls[:3])
	}
}

// `0F3Fh`：1d255 依 a、b、c、e、d 的累加權重分支。
func TestOutdoorScatterWeights(t *testing.T) {
	canvas := freshOutdoorCanvas()
	// 旗標 0：a=0、b=6、c=0Fh、e=0、d=28h。
	dice := &scriptedDice{queue: []int{6, 2, 21, 3, 61, 4, 1, 62, 22, 1}, fallback: func(count, sides int) int { return 0xFF }}
	outdoorScatter(canvas, 0x00, dice.roll)
	for y, want := range []uint8{0x38, 0x2C, 0x2E, OpenGroundCellClass, 0x2E} {
		if got := cellAt(canvas, 0, y); got != want {
			t.Errorf("(0,%d) = %02Xh, want %02Xh", y, got, want)
		}
	}
	// 旗標 80h：a=0Fh，1d4 = 4 且上一格是平地 → 上 41h、下 40h。
	canvas = freshOutdoorCanvas()
	dice = &scriptedDice{queue: []int{0xFF, 15, 4}, fallback: func(count, sides int) int { return 0xFF }}
	outdoorScatter(canvas, OutdoorFlag80, dice.roll)
	if cellAt(canvas, 0, 0) != 0x41 || cellAt(canvas, 0, 1) != 0x40 {
		t.Fatalf("80h pair = %02X %02X, want 41 40", cellAt(canvas, 0, 0), cellAt(canvas, 0, 1))
	}
}

// `0000h`：1d100 = 98／99 放 1Ah／1Bh；100 時區塊不是 0Ah 且再擲 1d100 = 1，
// 1d10 的 1..6 放 1Ch、7..10 放 1Dh。
func TestOutdoorRandomObjects(t *testing.T) {
	canvas := freshOutdoorCanvas()
	dice := &scriptedDice{queue: []int{98, 99, 100, 1, 7, 100, 2}, fallback: func(count, sides int) int { return 50 }}
	outdoorRandomObjects(canvas, 0x1B, dice.roll)
	for y, want := range []uint8{0x1A, 0x1B, 0x1D, OpenGroundCellClass} {
		if got := cellAt(canvas, 0, y); got != want {
			t.Errorf("(0,%d) = %02Xh, want %02Xh", y, got, want)
		}
	}
	canvas = freshOutdoorCanvas()
	dice = &scriptedDice{queue: []int{100}, fallback: func(count, sides int) int { return 50 }}
	outdoorRandomObjects(canvas, 0x0A, dice.roll)
	if len(dice.calls) != 1250 {
		t.Fatalf("block 0Ah must not roll the extra 1d100, got %d calls", len(dice.calls))
	}
}

func TestOutdoorGenerationFailsClosed(t *testing.T) {
	classes := gamepack.OriginalCombatCellClassTable()
	if _, err := GenerateOutdoorTacticalGrid(OutdoorBattlefield{Terrain: 0x01}, classes, nil); err == nil {
		t.Error("a nil dice roller was accepted")
	}
}

// `08B4h` 沒命中的地形碼改用 OutdoorUnmatchedFlags：擲骰順序要與旗標 0 完全相同。
func TestOutdoorUnmatchedTerrainUsesTheFallbackFlags(t *testing.T) {
	classes := gamepack.OriginalCombatCellClassTable()
	record := func(code uint8) [][2]int {
		dice := &scriptedDice{fallback: func(count, sides int) int { return sides }}
		if _, err := GenerateOutdoorTacticalGrid(OutdoorBattlefield{Terrain: code, Block: 0x1A}, classes, dice.roll); err != nil {
			t.Fatalf("terrain %02Xh: %v", code, err)
		}
		return dice.calls
	}
	unmatched := record(0x86)
	zero := &outdoorCanvas{terrain: NewOutdoorTacticalGrid().Terrain, classes: classes}
	dice := &scriptedDice{fallback: func(count, sides int) int { return sides }}
	outdoorBand(zero, 0, dice.roll)
	outdoorUprights(zero, 0, dice.roll)
	outdoorScatter(zero, 0, dice.roll)
	outdoorRandomObjects(zero, 0x1A, dice.roll)
	if len(unmatched) != len(dice.calls) {
		t.Fatalf("unmatched terrain rolled %d times, flags 0 roll %d", len(unmatched), len(dice.calls))
	}
	for index := range unmatched {
		if unmatched[index] != dice.calls[index] {
			t.Fatalf("roll %d: unmatched %v, flags 0 %v", index, unmatched[index], dice.calls[index])
		}
	}
}

// 整支跑下來：每一格都是類別表裡的類別，而且圖塊都落在室外那三段裡。
func TestOutdoorGenerationStaysInsideTheWildernessTiles(t *testing.T) {
	classes := gamepack.OriginalCombatCellClassTable()
	random := rand.New(rand.NewSource(59))
	roll := func(count, sides int) int {
		total := 0
		for i := 0; i < count; i++ {
			total += random.Intn(sides) + 1
		}
		return total
	}
	for _, code := range []uint8{0x01, 0x0A, 0x4E, 0x57, 0x6D, 0x9A, 0xA8, 0xEC} {
		grid, err := GenerateOutdoorTacticalGrid(OutdoorBattlefield{Terrain: code, Block: 0x1B}, classes, roll)
		if err != nil {
			t.Fatalf("terrain %02Xh: %v", code, err)
		}
		if !grid.Outdoor || grid.IgnoreTerrain || len(grid.Terrain) != TacticalMapCellCount {
			t.Fatalf("terrain %02Xh: outdoor=%t ignore=%t cells=%d", code, grid.Outdoor, grid.IgnoreTerrain, len(grid.Terrain))
		}
		detail := 0
		for index, class := range grid.Terrain {
			if int(class) >= len(classes) {
				t.Fatalf("terrain %02Xh cell %d class %02Xh is outside the table", code, index, class)
			}
			if class != OpenGroundCellClass {
				detail++
			}
			presentation := classes[class].PresentationCode
			random := class >= 0x1A && class <= 0x1D
			wild := class == OpenGroundCellClass || (class >= 0x20 && class <= 0x41)
			if !random && !wild {
				t.Fatalf("terrain %02Xh cell %d class %02Xh (tile %02Xh) is not an outdoor class", code, index, class, presentation)
			}
		}
		if detail == 0 {
			t.Errorf("terrain %02Xh produced a completely flat field", code)
		}
	}
}

// 野外三張圖可以走到的格子裡，`08B4h` 認不得的地形碼只有這七個（原版沒命中
// 就是讀未初始化的值，remake 用 OutdoorUnmatchedFlags）。多一個少一個都表示
// 比對表抄錯了。
func TestEveryReachableWildernessTerrainHasFlags(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	table, err := gamepack.ReadDOSWildernessTerrainTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	want := map[uint8]bool{0x86: true, 0xBB: true, 0xF9: true, 0xFB: true, 0xFD: true, 0xFE: true, 0xFF: true}
	got := map[uint8]bool{}
	for _, mode := range []uint8{gamepack.CombatAreaWildernessWest, gamepack.CombatAreaWildernessMid, gamepack.CombatAreaWildernessEast} {
		for y := 8; y <= 33; y++ {
			for x := 2; x <= 15; x++ {
				code, err := table.Background(mode, x, y)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := OutdoorTerrainFlags(code, false); err != nil {
					got[code] = true
				}
			}
		}
	}
	for code := range want {
		if !got[code] {
			t.Errorf("terrain %02Xh was expected to have no 08B4h entry", code)
		}
	}
	for code := range got {
		if !want[code] {
			t.Errorf("terrain %02Xh has no 08B4h entry", code)
		}
	}
}
