package combat

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// turboRandom 重現 START.EXE `5BBh:0C94h`（System.Random）與 `0CE3h`（NextRand）：
// RandSeed（`DS:438Ah`，dword）= RandSeed × 08088405h + 1，Random(n) = 新種子的
// 高 16 位 mod n（`0C9Fh..0CA4h` 的 `xchg dx,ax; div bx; xchg dx,ax`）。
type turboRandom struct{ seed uint32 }

func (r *turboRandom) random(n int) int {
	r.seed = r.seed*0x08088405 + 1
	if n == 0 {
		return 0
	}
	return int(r.seed>>16) % n
}

// roll 是 overlay-24 `0DE5h`：每顆 Random(sides)+1，加總存成 byte。
func (r *turboRandom) roll(count, sides int) int {
	total := 0
	for i := 0; i < count; i++ {
		total += r.random(sides) + 1
	}
	return total & 0xFF
}

type dosOutdoorReceipt struct {
	Generator      string `json:"generator"`
	SeedBefore     string `json:"seed_before_last_key"`
	Mode           uint8  `json:"mode_495B"`
	Background     uint8  `json:"background_45BC"`
	Block          uint16 `json:"block_82A2"`
	WildX          int    `json:"wild_x"`
	WildY          int    `json:"wild_y"`
	WalkFlag       int    `json:"walk_flag_49E6"`
	River          int    `json:"river_4AB3"`
	Header         string `json:"header"`
	Cells          string `json:"cells"`
	MatchedAdvance *int   `json:"remake_matched_advance"`
	LaterWrites    []struct {
		X, Y  int
		Class string
	} `json:"later_writes"`
}

// 原版對照：dosgolem 在野外 27 走到隨機遭遇、選 COMBAT，開打後讀 `[6674h]`
// 的 1250 格與按下最後一鍵前的 RandSeed（docs/audit/dosgolem-outdoor-battlefield.json）。
// 用 Turbo Pascal 的亂數從那個種子往前推 k 步再跑 GenerateOutdoorTacticalGrid，
// 收據記下的 k 必須逐格重現原版的盤面——這一條同時驗背景選擇、四支建構器的
// 規則與擲骰順序。
func TestOutdoorGeneratorReproducesTheDOSBattlefield(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-outdoor-battlefield.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt dosOutdoorReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Generator != "dosgolem" || receipt.MatchedAdvance == nil {
		t.Fatalf("receipt generator %q, matched advance %v", receipt.Generator, receipt.MatchedAdvance)
	}
	if receipt.Mode != gamepack.CombatAreaMode(receipt.Block, uint16(receipt.WalkFlag)) {
		t.Fatalf("DOS 495Bh = %d, CombatAreaMode(block %d, @49E6 %d) = %d", receipt.Mode, receipt.Block,
			receipt.WalkFlag, gamepack.CombatAreaMode(receipt.Block, uint16(receipt.WalkFlag)))
	}
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if table, err := gamepack.ReadDOSWildernessTerrainTable(zipPath); err == nil {
		got, err := table.Background(receipt.Mode, receipt.WildX, receipt.WildY)
		if err != nil || got != receipt.Background {
			t.Fatalf("Background(%d, %d, %d) = %02Xh, %v; DOS 45BCh = %02Xh", receipt.Mode,
				receipt.WildX, receipt.WildY, got, err, receipt.Background)
		}
	}
	header, err := hex.DecodeString(receipt.Header)
	if err != nil || len(header) != TacticalMapHeaderSize || header[4] != 0 || header[5] != 1 || header[6] != 0 {
		t.Fatalf("DOS header %s, want +4..+6 = 00 01 00 (%v)", receipt.Header, err)
	}
	want, err := hex.DecodeString(receipt.Cells)
	if err != nil || len(want) != TacticalMapCellCount {
		t.Fatalf("DOS cells: %d bytes, %v", len(want), err)
	}
	seedBytes, err := hex.DecodeString(receipt.SeedBefore)
	if err != nil || len(seedBytes) != 4 {
		t.Fatalf("seed %q: %v", receipt.SeedBefore, err)
	}
	seed := uint32(seedBytes[0]) | uint32(seedBytes[1])<<8 | uint32(seedBytes[2])<<16 | uint32(seedBytes[3])<<24
	random := &turboRandom{seed: seed}
	for i := 0; i < *receipt.MatchedAdvance; i++ {
		random.random(1)
	}
	grid, err := GenerateOutdoorTacticalGrid(OutdoorBattlefield{Terrain: receipt.Background,
		RiverCleared: receipt.River == 0xFF, Block: receipt.Block}, gamepack.OriginalCombatCellClassTable(), random.roll)
	if err != nil {
		t.Fatal(err)
	}
	// 生成之後另有常式改寫的格子（收據 later_writes，原版類別 1Fh 不是 `1255h`
	// 寫得出來的）不在這一條的範圍；其餘每一格都要相同。
	later := map[int]bool{}
	for _, write := range receipt.LaterWrites {
		later[write.Y*TacticalRowStride+write.X] = true
	}
	if len(later) != 1 {
		t.Fatalf("receipt lists %d later writes; it was recorded with exactly one", len(later))
	}
	for index := range want {
		if later[index] {
			continue
		}
		if grid.Terrain[index] != want[index] {
			t.Fatalf("cell (%d,%d) = %02Xh, DOS %02Xh", index%TacticalRowStride, index/TacticalRowStride,
				grid.Terrain[index], want[index])
		}
	}
}
