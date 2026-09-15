package main

// 主線第 6 到第 11 段（spec 137）：交完索寇件之後，從城區搭東航線進野外、
// 跨到圖 26 選 NORTH 進波多廣場、出北緣過斯托亞諾夫城門、進城堡、上樓、
// 站上覲見廳那一格打到結局。每一段是一個有界步驟，撞到 guard 就 Fatalf
// 印出當下的區塊、座標與主線旗標——不靠隨機探索。
//
// 所有移動都由 Update() 的正常按鍵完成：方向鍵轉向、上鍵前進、ENTER 翻
// 文字、選單用方向鍵移游標。不寫座標、不寫旗標、不改隊伍。

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// mainlineDriver 把探針測試的 step（送一個鍵進 Update）包起來，
// 加上每一段共用的「把等待中的事件按完」與「照規劃走到某格」。
type mainlineDriver struct {
	t     *testing.T
	a     *app
	step  func(key ebiten.Key)
	pilot *tacticalPilot
	log   []string
	trace bool
}

// flags 是每一個 Fatalf 都要帶的現場：主線旗標一次印齊，查失敗不用重跑。
func (d *mainlineDriver) flags() string {
	a := d.a
	block := uint16(0xFFFF)
	if a.eventSession != nil {
		block = a.eventSession.CurrentBlockID()
	}
	memory := map[uint16]uint16{}
	if a.eventMachine != nil {
		memory = a.eventMachine.Memory
	}
	return fmt.Sprintf("ECL%d/%d GEO%d/%d (%d,%d) facing=%d 4ABA=%02X 4AC1=%d 4AA7=%02X 4A01=%02X 4A00=%02X 4A6D=%02X 4A77=%02X 4A78=%d 49C9=%d pending=%t menu=%v door=%t tactical=%t encounter=%t text=%q status=%q",
		a.eclArchive, block, a.spawn.Map.Archive, a.spawn.Map.BlockID, a.spawn.X, a.spawn.Y,
		a.spawn.Facing, memory[0x4ABA], memory[0x4AC1], memory[0x4AA7], memory[0x4A01],
		memory[0x4A00], memory[0x4A6D], memory[0x4A77], memory[0x4A78], memory[0x49C9],
		a.cellEventPending, a.cellMenuOptions, a.door != nil, a.tactical != nil,
		a.encounter != nil, strings.TrimSpace(a.eventText), a.statusLine)
}

func (d *mainlineDriver) fatalf(format string, args ...any) {
	d.t.Helper()
	d.t.Fatalf(format+"\n  at %s\n  log: %s", append(args, d.flags(), strings.Join(d.log, " | "))...)
}

func (d *mainlineDriver) note(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	d.log = append(d.log, line)
	d.t.Logf("mainline: %s", line)
}

// busy 說目前有沒有東西在等玩家——有的話走路的鍵會被它吃掉。
func (d *mainlineDriver) busy() bool {
	a := d.a
	return a.door != nil || a.cellWaitingMenu || a.cellEventPending || a.encounter != nil ||
		a.treasureActive || a.tactical != nil || a.combatActive || a.shopActive ||
		a.endingActive || a.gameOver
}

// settle 把等待中的事件按完。選單挑 prefer 裡認得的第一項，都不認得就選
// 第一項；戰鬥交給戰術駕駛；鎖住的門一律 EXIT（主線上沒有要撬的門）。
// guard 給寬：一場戰鬥就是幾百個 tick。
func (d *mainlineDriver) settle(prefer ...string) {
	d.t.Helper()
	a := d.a
	for tick := 0; tick < 40000 && d.busy(); tick++ {
		if a.gameOver {
			d.fatalf("the party was destroyed")
		}
		switch {
		case a.endingActive:
			d.step(ebiten.KeyEnter)
		case a.tactical != nil:
			d.step(d.pilot.key(a))
		case a.encounter != nil:
			if err := selectMenuOption(d.t, a, "COMBAT"); err != nil {
				d.fatalf("encounter menu: %v", err)
			}
		case a.combatActive:
			d.step(ebiten.KeyEnter)
		case a.shopActive:
			d.step(ebiten.KeyEscape)
		case a.treasureActive:
			want := "Exit"
			for _, option := range a.cellMenuOptions {
				if option == "Yes" {
					want = option
				}
			}
			if err := selectMenuOption(d.t, a, want); err != nil {
				d.fatalf("treasure menu: %v", err)
			}
		case a.door != nil && !a.cellEventPending:
			dismissDoorMenu(a)
			a.keys = nil
		case a.cellWaitingMenu && len(a.cellMenuOptions) != 0:
			pick := 0
			for _, label := range prefer {
				for index, option := range a.cellMenuOptions {
					if strings.EqualFold(option, label) {
						pick = index
					}
				}
				if pick != 0 || (len(a.cellMenuOptions) != 0 &&
					strings.EqualFold(a.cellMenuOptions[0], label)) {
					break
				}
			}
			for guard := 0; a.cellMenuCursor != pick && guard < 64; guard++ {
				d.step(ebiten.KeyArrowDown)
			}
			d.step(ebiten.KeyEnter)
		default:
			d.step(ebiten.KeyEnter)
		}
	}
	if d.busy() {
		d.fatalf("settle did not finish")
	}
}

// face 轉到指定朝向；右轉最多三下。
func (d *mainlineDriver) face(facing uint8) {
	d.t.Helper()
	for guard := 0; d.a.spawn.Facing != facing && guard < 8; guard++ {
		key := ebiten.KeyArrowRight
		if (int(facing)-int(d.a.spawn.Facing)+4)%4 == 3 {
			key = ebiten.KeyArrowLeft
		}
		d.step(key)
	}
	if d.a.spawn.Facing != facing {
		d.fatalf("could not face %d", facing)
	}
}

// walkTo 照規劃器走到 wanted 的格子；每一步之後把事件按完再重新規劃，
// 所以事件把隊伍搬走也接得住。avoid 是規劃時要繞開的格子（可為 nil）。
// 回傳 false 代表規劃不到；到了回傳 true。
func (d *mainlineDriver) walkTo(what string, wanted, avoid func(x, y int) bool,
	prefer ...string) bool {
	d.t.Helper()
	a := d.a
	here := func() [2]int { return [2]int{int(a.spawn.X), int(a.spawn.Y)} }
	startMap, startBlock := a.spawn.Map, a.eventSession.CurrentBlockID()
	last := here()
	for guard := 0; guard < 400; guard++ {
		d.settle(prefer...)
		if a.spawn.Map != startMap || a.eventSession.CurrentBlockID() != startBlock {
			d.note("walkTo %s: left the map before arriving (from %v to %v, block %d → %d)",
				what, last, here(), startBlock, a.eventSession.CurrentBlockID())
			return false
		}
		last = here()
		if position := here(); wanted(position[0], position[1]) {
			return true
		}
		// 不繞邊界：踏出邊界是「離開這一區」，不是走到另一側（spec 100）。
		skip := avoid
		if skip == nil {
			skip = func(int, int) bool { return false }
		}
		plan := planToCellsWithoutWrapping(a, 0, wanted, skip)
		if len(plan) == 0 {
			d.note("walkTo %s: no plan", what)
			return false
		}
		d.face(plan[0].facing)
		d.settle(prefer...)
		d.step(ebiten.KeyArrowUp)
		if d.trace {
			d.t.Logf("  walkTo %s: %v facing %d → %v (plan %d) %s", what, last,
				plan[0].facing, here(), len(plan), strings.TrimSpace(a.eventText))
		}
	}
	d.fatalf("walkTo %s: guard exhausted", what)
	return false
}

// stepOut 站在目前這一格朝 facing 踏出去，期待換到別的 ECL 區塊。
func (d *mainlineDriver) stepOut(what string, facing uint8, prefer ...string) uint16 {
	d.t.Helper()
	before := d.a.eventSession.CurrentBlockID()
	d.settle(prefer...)
	d.face(facing)
	d.step(ebiten.KeyArrowUp)
	d.settle(prefer...)
	after := d.a.eventSession.CurrentBlockID()
	if after == before {
		d.fatalf("%s: stepping out facing %d stayed in block %d", what, facing, before)
	}
	d.note("%s: block %d → %d at (%d,%d) C04B/C=(%d,%d)", what, before, after, d.a.spawn.X, d.a.spawn.Y,
		d.a.eventMachine.Memory[0xC04B], d.a.eventMachine.Memory[0xC04C])
	return after
}

func (d *mainlineDriver) terrain(x, y int) int {
	cell, ok := d.a.initialMap.Grid.Cell(x, y)
	if !ok {
		return -1
	}
	return int(cell.Terrain) & 0x1F
}

// mainlineSailEast 是第 6 段：港務長選 EAST、上船、落在野外圖 27。
// 前置：站在城區 ECL3/0，`4AA7 >= FEh`（索寇要塞已拿下），`4A01 == 0`。
func (d *mainlineDriver) sailEast() {
	d.t.Helper()
	a := d.a
	memory := a.eventMachine.Memory
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		d.fatalf("sailEast: not in the city")
	}
	if memory[0x4AA7] < 0xFE || memory[0x4A01] != 0 {
		d.fatalf("sailEast: harbour master preconditions not met")
	}
	if memory[0x4A00] != 0 {
		d.fatalf("sailEast: 4A00 is not clear; the Stojanow wagon would never appear")
	}
	// 港務長要面對他（spec 102）：站 (11,2) 朝北踏進去。
	plain := func(x, y int) bool { return d.terrain(x, y) != 0 && !(x == 11 && y == 2) }
	if !d.walkTo("harbour master", func(x, y int) bool { return x == 11 && y == 2 }, plain, "EXIT") {
		d.fatalf("sailEast: cannot reach (11,2)")
	}
	d.settle("EXIT")
	d.face(0)
	d.step(ebiten.KeyArrowUp)
	d.settle("EAST")
	if memory[0x4A01] != 1 {
		d.fatalf("sailEast: harbour master did not sell the EAST ticket")
	}
	d.note("EAST ticket bought, 4AC4=%d", memory[0x4AC4])
	// 踩上碼頭 (15,1) 那一步就開船（`9BA6h`），walkTo 會以「離開地圖」回報。
	if !d.walkTo("pier", func(x, y int) bool { return x == 15 && y == 1 },
		func(x, y int) bool { return d.terrain(x, y) != 0 && !(x == 15 && y == 1) }) &&
		a.eclArchive == 3 {
		d.fatalf("sailEast: cannot reach the pier")
	}
	for guard := 0; guard < 200 && !a.inWildernessOverland(); guard++ {
		d.settle()
		if !a.inWildernessOverland() {
			d.step(ebiten.KeyEnter)
		}
	}
	if !a.inWildernessOverland() || a.eventSession.CurrentBlockID() != 27 {
		d.fatalf("sailEast: the EAST boat did not land in wilderness block 27")
	}
	d.note("landed in the wilderness at (%d,%d)", memory[wildernessX], memory[wildernessY])
}

// mainlineCrossToPodol 是第 6 段後半到第 7 段：從圖 27 往西跨到圖 26，
// 踩 (11,28) 選 NORTH 進波多廣場（ECL1/18）。走法與
// `enterStojanowGate` 相同（spec 105 實測的東航線位移）。
func (d *mainlineDriver) crossToPodol() {
	d.t.Helper()
	a := d.a
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	memory := a.eventMachine.Memory
	walk := newWildernessWalk(true, 0)
	for step := 0; step < 3000 && int(a.eventSession.CurrentBlockID()) != 26; step++ {
		if !a.inWildernessOverland() {
			d.fatalf("crossToPodol: left the wilderness before reaching block 26")
		}
		key := ebiten.KeyArrowRight
		if want, ok := wildernessNextFacing(a, zipPath, walk); ok {
			key = wildernessTurnKey(a.spawn.Facing, want)
		}
		d.step(key)
		drainWildernessEvents(a)
		a.keys = nil
	}
	if int(a.eventSession.CurrentBlockID()) != 26 {
		d.fatalf("crossToPodol: never crossed into wilderness block 26")
	}
	d.note("crossed into wilderness block 26 at (%d,%d)", memory[wildernessX], memory[wildernessY])
	for step := 0; step < 400 && a.inWildernessOverland(); step++ {
		here := [2]int{int(memory[wildernessX]), int(memory[wildernessY])}
		if here == [2]int{11, 28} {
			break
		}
		route := wildernessRoute(a, here, [2]int{11, 28}, nil)
		if len(route) == 0 {
			d.fatalf("crossToPodol: no route from (%d,%d) to (11,28)", here[0], here[1])
		}
		d.face(route[0])
		d.step(ebiten.KeyArrowUp)
		for tick := 0; tick < 200 && (a.cellEventPending || a.cellWaitingMenu); tick++ {
			d.settle("NORTH")
		}
	}
	if got := int(a.eventSession.CurrentBlockID()); got != 18 {
		d.fatalf("crossToPodol: after (11,28) NORTH the block is %d, want 18", got)
	}
	d.note("entered Podol Plaza (ECL1/18) at (%d,%d)", a.spawn.X, a.spawn.Y)
}

// mainlinePodolNorthEdge 是第 7 段：走到 GEO1/18 的 (4,0) 或 (11,0)，
// 朝北踏出去進斯托亞諾夫城門（ECL2/9）。
func (d *mainlineDriver) podolNorthEdge() {
	d.t.Helper()
	avoid := func(x, y int) bool { return d.terrain(x, y) != 0 }
	for _, cell := range [][2]int{{4, 0}, {11, 0}} {
		target := func(x, y int) bool { return x == cell[0] && y == cell[1] }
		if !d.walkTo(fmt.Sprintf("Podol north edge (%d,0)", cell[0]), target, avoid) {
			if !d.walkTo(fmt.Sprintf("Podol north edge (%d,0) through events", cell[0]), target, nil) {
				continue
			}
		}
		if got := d.stepOut("Podol north edge", 0); got != stojanowGateBlockID {
			d.fatalf("podolNorthEdge: stepping north went to block %d, want %d", got, stojanowGateBlockID)
		}
		return
	}
	d.fatalf("podolNorthEdge: neither (4,0) nor (11,0) is reachable")
}

// mainlineStojanowGate 是第 8 段：拿馬車、付過路費、被腳本搬到北半邊，
// 從北緣踏出去進城堡（spec 101）。地形 3 是衛兵，踩到之後索賄必翻臉。
func (d *mainlineDriver) stojanowGate() {
	d.t.Helper()
	a := d.a
	memory := a.eventMachine.Memory
	if memory[0x4A00] != 0 || memory[0x4A77]&64 != 0 || memory[0x4A78] >= 3 {
		d.fatalf("stojanowGate: entry state already rules the wagon out")
	}
	guards := func(x, y int) bool { return d.terrain(x, y) == 3 }
	if !d.walkTo("wagon trader (terrain 9)", func(x, y int) bool { return d.terrain(x, y) == 9 }, guards) {
		d.fatalf("stojanowGate: cannot reach the wagon trader")
	}
	d.settle("COMBAT")
	if memory[0x4A77]&64 == 0 {
		d.fatalf("stojanowGate: the wagon flag (4A77 bit 6) is not set")
	}
	d.note("got the wagon")
	// 索賄那一支先擲 `RANDOM 9`（0..9）再與 `4A78` 比，擲到 0 就是
	// 「THEY ARE IMPOSTERS!」（`AB8Bh`）——原版本來就有一成機率。那個選單選
	// FLEE 只把隊伍搬回 (4,14)／(11,14)，不寫旗標（`A7FDh`），再走回去就是
	// 重擲；ATTACK 才會把 `4A78` 寫成 3 讓城門永遠過不去。
	through := false
	for attempt := 0; attempt < 8 && !through; attempt++ {
		// 付成功的那一刻隊伍已經在 (8,5)；目標要把它算進去，否則 walkTo 會從
		// 北半邊再走去踩另一格地形 5，第二次就是「THROW DOWN YOUR WEAPONS」。
		if !d.walkTo("toll bugbear (terrain 5)",
			func(x, y int) bool { return d.terrain(x, y) == 5 || (x == 8 && y == 5) },
			guards, "YES", "FLEE") {
			d.fatalf("stojanowGate: cannot reach the toll cell")
		}
		d.settle("YES", "FLEE")
		through = [2]int{int(a.spawn.X), int(a.spawn.Y)} == [2]int{8, 5}
		if !through {
			d.note("toll attempt %d ended at (%d,%d) 4A78=%d", attempt+1, a.spawn.X, a.spawn.Y, memory[0x4A78])
		}
		if memory[0x4A78] >= 3 {
			d.fatalf("stojanowGate: 4A78 reached 3, the gate is closed for good")
		}
	}
	if !through {
		d.fatalf("stojanowGate: never got through the gate")
	}
	d.note("through the gate at (8,5)")
	for _, cell := range [][2]int{{4, 0}, {11, 0}} {
		if !d.walkTo(fmt.Sprintf("gate north edge (%d,0)", cell[0]),
			func(x, y int) bool { return x == cell[0] && y == cell[1] }, guards, "YES") {
			continue
		}
		block := d.stepOut("gate north edge", 0, "YES")
		if block == stojanowGateBlockID {
			d.fatalf("stojanowGate: still at the gate")
		}
		return
	}
	d.fatalf("stojanowGate: the north edge is unreachable from (8,5)")
}

// castleInteriorExits 是 `ecl5/5` 入口 0 的換圖表（`9AEBh` 起，bytes
// `03 04 06 03 / 04 04 05 03 / 04 05 05 06 / 03 05 06 06`）：站在哪一張 GEO、
// 朝哪個方向踏出邊界，就 `LOAD FILES` 哪一張。區塊不變。
var castleInteriorExits = map[uint8][4]uint8{3: {3, 4, 6, 3}, 4: {4, 4, 5, 3}, 5: {4, 5, 5, 6}, 6: {3, 5, 6, 6}}

// castleInteriorNextFacing 在四張城堡 GEO 拼成的 32×32 圖上做 BFS，回傳往
// 「goalTerrain」那一格的下一步朝向。只走地形 0 的格子：地形 30／31 在區塊 5
// 踩到會 `NEWECL 3`（回到外圈腳本），地形 7 是隨機傳送陣，其餘是事件。
// （驗證用的獨立版本在 `workplace/castleq`，從城門格 (4,8) 量到 134 步。）
func castleInteriorNextFacing(a *app, goalTerrain int) (uint8, bool) {
	type node struct {
		geo  uint8
		x, y int
	}
	maps := map[uint8]geometry.Grid{}
	for _, id := range []uint8{3, 4, 5, 6} {
		m, ok := a.geometryCatalog.MapByBlock(id)
		if !ok {
			return 0, false
		}
		maps[id] = m.Grid
	}
	terrain := func(n node) int { return int(maps[n.geo].Cells[n.y][n.x].Terrain & 0x1F) }
	start := node{a.spawn.Map.BlockID, int(a.spawn.X), int(a.spawn.Y)}
	if _, ok := maps[start.geo]; !ok {
		return 0, false
	}
	prev := map[node]node{start: start}
	via := map[node]uint8{}
	queue := []node{start}
	for len(queue) != 0 {
		cur := queue[0]
		queue = queue[1:]
		for f := 0; f < 4; f++ {
			if !maps[cur.geo].CanMoveDungeonWrapped(cur.x, cur.y, f*2) {
				continue
			}
			nx, ny, ngeo := cur.x+exploreDeltas[f][0], cur.y+exploreDeltas[f][1], cur.geo
			if nx < 0 || nx > 15 || ny < 0 || ny > 15 {
				ngeo = castleInteriorExits[cur.geo][f]
				nx, ny = (nx+16)%16, (ny+16)%16
			}
			next := node{ngeo, nx, ny}
			if _, seen := prev[next]; seen {
				continue
			}
			prev[next], via[next] = cur, uint8(f)
			if terrain(next) == goalTerrain {
				for prev[next] != start {
					next = prev[next]
				}
				return via[next], true
			}
			if terrain(next) != 0 {
				continue
			}
			queue = append(queue, next)
		}
	}
	return 0, false
}

// mainlineCastleUpstairs 是第 9、10 段。城堡是四張 GEO（3 西北、4 東北、
// 6 西南、5 東南）拼成的 32×32；`ecl5/3`／`5/4`／`5/6` 是外圈腳本，
// `ecl5/5` 是**四張圖共用**的城堡內部腳本：不自載地圖，靠 `49C5`（目前
// GEO）挑換圖表（`castleInteriorExits`）。
//
//  1. 院子（區塊 3，GEO5/3）踩上城門格 (4,7)／(4,8)（地形 21）→ `NEWECL 5`。
//  2. 區塊 5 裡沿內部繞一圈：GEO3 南緣 → GEO6 → 東緣 → GEO5 → 北緣 → GEO4
//     → 西緣 → GEO3 (15,11) → (13,15)。那一格地形 4；朝東踏一次 →
//     `ecl5/5 9B28h`「地形 4 ＋ 朝向 1」→ `A467h`「YOU GO UPSTAIRS.」→ `NEWECL 7`。
func (d *mainlineDriver) castleUpstairs() {
	d.t.Helper()
	a := d.a
	for guard := 0; guard < 8 && a.eventSession.CurrentBlockID() != 3; guard++ {
		d.settle("YES")
	}
	if a.eventSession.CurrentBlockID() != 3 {
		d.fatalf("castleUpstairs: not in the castle courtyard (block 3)")
	}
	gate := func(x, y int) bool { return d.terrain(x, y) == 21 }
	arrived := d.walkTo("castle gate (terrain 21)", gate,
		func(x, y int) bool { return d.terrain(x, y) != 0 && !gate(x, y) }, "YES")
	if !arrived && a.eventSession.CurrentBlockID() == 3 {
		arrived = d.walkTo("castle gate (terrain 21) through events", gate, nil, "YES")
	}
	if arrived && a.eventSession.CurrentBlockID() == 3 {
		d.settle("YES")
	}
	if got := a.eventSession.CurrentBlockID(); got != 5 {
		d.fatalf("castleUpstairs: the gate cell led to block %d, want 5", got)
	}
	d.note("castle interior script (block 5) at GEO5/%d (%d,%d)", a.spawn.Map.BlockID, a.spawn.X, a.spawn.Y)
	lastGEO := a.spawn.Map.BlockID
	for step := 0; step < 400; step++ {
		d.settle("YES")
		if a.eventSession.CurrentBlockID() != 5 {
			d.fatalf("castleUpstairs: the interior walk left block 5 at step %d", step)
		}
		if a.spawn.Map.BlockID != lastGEO {
			d.note("interior walk: GEO5/%d → GEO5/%d at (%d,%d)", lastGEO, a.spawn.Map.BlockID, a.spawn.X, a.spawn.Y)
			lastGEO = a.spawn.Map.BlockID
		}
		if d.terrain(int(a.spawn.X), int(a.spawn.Y)) == 4 {
			break
		}
		facing, ok := castleInteriorNextFacing(a, 4)
		if !ok {
			d.fatalf("castleUpstairs: no plain-cell route to a terrain-4 cell from here")
		}
		d.face(facing)
		d.settle("YES")
		d.step(ebiten.KeyArrowUp)
	}
	if d.terrain(int(a.spawn.X), int(a.spawn.Y)) != 4 {
		d.fatalf("castleUpstairs: never reached a terrain-4 cell")
	}
	if got := d.stepOut("keep stairs", 1, "YES"); got != 7 {
		d.fatalf("castleUpstairs: terrain-4 east went to block %d, want 7", got)
	}
}

// mainlineAudienceHall 是第 11 段：GEO5/7 的 (3,8) 是覲見廳（地形 8）。
// 衛兵那一場選 COMBAT，投票每個人都選 ATTACK，最後一戰交給戰術駕駛。
// 回傳三個收據：看到過 TYRANITHRAXUS、進過結局過場、結局頁數。
func (d *mainlineDriver) audienceHall() (sawTyranthraxus, sawEnding bool, pages int) {
	d.t.Helper()
	a := d.a
	memory := a.eventMachine.Memory
	// 先走到 (3,8) 旁邊一格（(4,8) 或 (3,7)），最後一步自己踏：站上去之後
	// 所有事件連在一起——覲見廳文字 → 衛兵 → 龍的說詞 → 投票 → 最後一戰 →
	// 結局——收據要沿路記，不能交給 walkTo 的 settle 一口氣按完。
	hall := [2]int{3, 8}
	beside := func(x, y int) bool {
		return (x == 4 && y == 8) || (x == 3 && y == 7)
	}
	if !d.walkTo("beside the audience hall", beside,
		func(x, y int) bool { return d.terrain(x, y) != 0 && !beside(x, y) }, "ATTACK") {
		d.fatalf("audienceHall: cannot reach a cell beside (3,8)")
	}
	d.settle("ATTACK")
	if int(a.spawn.X) == 4 {
		d.face(3)
	} else {
		d.face(2)
	}
	d.step(ebiten.KeyArrowUp)
	if got := [2]int{int(a.spawn.X), int(a.spawn.Y)}; got != hall && !d.busy() {
		d.fatalf("audienceHall: the last step did not enter (3,8)")
	}
	battles := 0
	tacticalTicks := 0
	lastLine := ""
	var lastState *tacticalState
	for tick := 0; tick < 60000 && d.busy() && !a.gameOver; tick++ {
		if a.tactical != nil {
			tacticalTicks++
			if d.trace {
				line := a.tactical.Status + " / " + a.tactical.FoeLog
				if line != lastLine {
					lastLine = line
					d.t.Logf("  round %d mover %d scores=%v states=%v %s", a.tactical.Round,
						a.tactical.Mover, a.tactical.Scores, a.tactical.States, line)
				}
			}
			if a.tactical != lastState {
				lastState = a.tactical
				battles++
				names := []string{}
				for _, monster := range a.combatMonsters {
					names = append(names, monster.Record.Name)
				}
				d.note("battle %d: %d monsters %v", battles, len(a.combatMonsters), names)
			}
			for _, monster := range a.combatMonsters {
				if monster.Record.Name == "TYRANITHRAXUS" {
					sawTyranthraxus = true
				}
			}
		} else if lastState != nil {
			party := []string{}
			for _, member := range a.state.Party {
				party = append(party, fmt.Sprintf("%s hp=%d/%d st=%d", strings.TrimSpace(member.Name),
					member.CurrentHP, member.MaxHP, member.Status))
			}
			d.note("battle %d over after %d ticks (round %d): %v", battles, tacticalTicks, lastState.Round, party)
			lastState = nil
		}
		if a.endingActive {
			sawEnding = true
			if len(a.endingPages) > pages {
				pages = len(a.endingPages)
			}
		}
		switch {
		case a.endingActive:
			d.step(ebiten.KeyEnter)
		case a.tactical != nil:
			d.step(d.pilot.key(a))
		case a.encounter != nil:
			if err := selectMenuOption(d.t, a, "COMBAT"); err != nil {
				d.fatalf("audienceHall encounter: %v", err)
			}
		case a.combatActive:
			d.step(ebiten.KeyEnter)
		case a.cellWaitingMenu && len(a.cellMenuOptions) != 0:
			pick := 0
			for index, option := range a.cellMenuOptions {
				if strings.EqualFold(option, "ATTACK") || strings.EqualFold(option, "COMBAT") {
					pick = index
				}
			}
			for guard := 0; a.cellMenuCursor != pick && guard < 64; guard++ {
				d.step(ebiten.KeyArrowDown)
			}
			d.step(ebiten.KeyEnter)
		default:
			d.step(ebiten.KeyEnter)
		}
		if memory[0x4ABA] >= 0xFE && !a.endingActive && !a.cellEventPending &&
			!a.cellWaitingMenu && a.tactical == nil {
			break
		}
	}
	d.note("audience hall done: 4ABA=%02X tyranthraxus=%t ending=%t pages=%d gameOver=%t",
		memory[0x4ABA], sawTyranthraxus, sawEnding, pages, a.gameOver)
	return sawTyranthraxus, sawEnding, pages
}
