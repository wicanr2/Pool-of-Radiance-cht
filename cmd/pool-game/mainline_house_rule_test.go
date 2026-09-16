package main

// 自訂規則「委任折算經驗值」（spec 140，#28）開著時，主線探針多走的幾段：
// 一級隊伍打得起的第一件委任是曼多爾圖書館的六本書（spec 137 建議順序第 2 條，
// 最大一場 5），交件之後每人 2501 XP，全隊過二級門檻，再走去訓練所。
//
// 路線（全部是正常按鍵）：城區 (0,4) 西出 → 貧民窟 (15,4) 走到 (0,4) 西出 →
// 古托井 (15,4) 走到 (11,15) 南出 → 圖書館 (11,0)。圖書館兩道門從外面都是閂住的
//（`WallDoorFlags` 3），撞門選 PICK 由賊開；一次不成就 EXIT 再撞——remake 現在
// EXIT 之後 `a.door` 清掉，下一次撞是新的一份額度（spec 122 還沒讀到原版誰把
// `6CD2h`／`6CD3h` 設回 1）。書在書架格上按 L）OOK 搜出來（`ecl2/15 9DF5h` 只看
// `6DCA != 0`），每次 RANDOM 11／17 擲書。離開要避開幽靈：入口 0 `99F8h` 在
// 「站在門格、朝向等於表 `9B07h` 的出口方向、身上有書」時召出 SPECTRE（7 HD，
// 一級隊伍打不起）——所以從 (7,2) 朝東踏上北門格 (8,2)，再轉北撞門出去。
//
// 每一段的死區：圖書館地形 5 是石化蜥蜴、地形 11 是瘋戰士、地形 1 花園有隨機
// 遭遇、地形 3 書架會把 `4A01`（船票旗標）寫成 1；古托井地形 12 是諾里斯那一場
//（15 隻）；貧民窟 9／13／15 同主線探針。

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// planAllowing 是 planToCells 的「只准踩 allowed 的格子」版本：doors 為真時
// 鎖住／閂住的門也算一步（撞門選單由走的那一邊處理）。不繞邊界。
func planAllowing(a *app, wanted, allowed func(x, y int) bool, doors bool) []exploreStep {
	type node struct{ x, y int }
	start := node{int(a.spawn.X), int(a.spawn.Y)}
	from := map[node]node{start: start}
	via := map[node]uint8{}
	queue := []node{start}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current != start && wanted(current.x, current.y) {
			steps := []exploreStep{}
			for cursor := current; cursor != start; cursor = from[cursor] {
				steps = append([]exploreStep{{facing: via[cursor]}}, steps...)
			}
			return steps
		}
		for facing := 0; facing < 4; facing++ {
			if !a.initialMap.Grid.CanMoveDungeonWrapped(current.x, current.y, facing*2) {
				flags, door := a.initialMap.Grid.WallDoorFlagsWrapped(current.x, current.y, facing*2)
				if !doors || !door || (flags != gamepack.DoorLocked && flags != gamepack.DoorBarred) {
					continue
				}
			}
			next := node{current.x + exploreDeltas[facing][0], current.y + exploreDeltas[facing][1]}
			if next.x < 0 || next.x >= geometry.Width || next.y < 0 || next.y >= geometry.Height {
				continue
			}
			if _, seen := from[next]; seen {
				continue
			}
			if !wanted(next.x, next.y) && !allowed(next.x, next.y) {
				continue
			}
			from[next], via[next] = current, uint8(facing)
			queue = append(queue, next)
		}
	}
	return nil
}

// terrainCode 是格子的地形碼低五位（ECL 的 `AND @C04F 31`）。
func (d *mainlineDriver) terrainCode(x, y int) int {
	cell, ok := d.a.initialMap.Grid.Cell(x, y)
	if !ok {
		return -1
	}
	return int(cell.Terrain) & 0x1F
}

// restIfHurt 把探針的策略層（受傷就紮營）接進駕駛：沒接就不休息。
func (d *mainlineDriver) restIfHurt() {
	if d.rest != nil && d.hurt != nil && d.hurt(d.a) {
		d.rest()
	}
}

// walkAllowing 照 planAllowing 走到 wanted；每步之前先把事件按完、受傷就休息。
// 鎖住的門照探索器的順序試（BASH → PICK → KNOCK → EXIT），每一道門記已試過的。
// 換了圖或區塊就回 false。
func (d *mainlineDriver) walkAllowing(what string, wanted, allowed func(x, y int) bool, doors bool) bool {
	d.t.Helper()
	a := d.a
	startMap, startBlock := a.spawn.Map, a.eventSession.CurrentBlockID()
	tried := map[[5]int]map[string]bool{}
	for guard := 0; guard < 2000; guard++ {
		if a.door != nil && !a.cellEventPending {
			key := explorerDoorKey(a)
			if tried[key] == nil {
				tried[key] = map[string]bool{}
			}
			want := explorerDoorChoice(a.door.Options, tried[key])
			if a.door.Options[a.door.Cursor] != want {
				d.step(ebiten.KeyArrowRight)
				continue
			}
			tried[key][want] = true
			d.step(ebiten.KeyEnter)
			continue
		}
		d.settle()
		if a.gameOver {
			return false
		}
		d.restIfHurt()
		if a.spawn.Map != startMap || a.eventSession.CurrentBlockID() != startBlock {
			d.note("walkAllowing %s: left the map at (%d,%d) block %d → %d", what,
				a.spawn.X, a.spawn.Y, startBlock, a.eventSession.CurrentBlockID())
			return false
		}
		if wanted(int(a.spawn.X), int(a.spawn.Y)) {
			return true
		}
		plan := planAllowing(a, wanted, allowed, doors)
		if len(plan) == 0 {
			around := []string{}
			for facing := 0; facing < 4; facing++ {
				x, y := int(a.spawn.X)+exploreDeltas[facing][0], int(a.spawn.Y)+exploreDeltas[facing][1]
				around = append(around, fmt.Sprintf("%d:(%d,%d)t%d open=%t", facing, x, y, d.terrainCode(x, y),
					a.initialMap.Grid.CanMoveDungeonWrapped(int(a.spawn.X), int(a.spawn.Y), facing*2)))
			}
			d.note("walkAllowing %s: no plan from (%d,%d) on %+v ECL%d/%d: %v", what, a.spawn.X, a.spawn.Y,
				a.initialMap.Key, a.eclArchive, a.eventSession.CurrentBlockID(), around)
			return false
		}
		d.face(plan[0].facing)
		d.step(ebiten.KeyArrowUp)
	}
	d.fatalf("walkAllowing %s: guard exhausted", what)
	return false
}

// leaveMap 站在 wanted 的邊界格朝 outward 踏出去，期待換到別張圖。
func (d *mainlineDriver) leaveMap(what string, wanted, allowed func(x, y int) bool, doors bool, outward uint8) {
	d.t.Helper()
	a := d.a
	from := a.spawn.Map
	if !d.walkAllowing(what, wanted, allowed, doors) {
		if a.spawn.Map != from {
			return
		}
		d.fatalf("%s: cannot reach the boundary", what)
	}
	d.settle()
	d.face(outward)
	before := a.spawn
	d.step(ebiten.KeyArrowUp)
	d.settle()
	if a.spawn.Map == from {
		d.fatalf("%s: outward step from %+v stayed on the same map at %+v", what, before, a.spawn)
	}
	d.note("%s: %+v → %+v ECL%d/%d", what, before, a.spawn, a.eclArchive, a.eventSession.CurrentBlockID())
}

// slumsSafe 是貧民窟一級隊伍不踩的固定事件（主線探針同一份：9 獸人的家、
// 13 衛兵攔截、15 驚動衛兵）。
func (d *mainlineDriver) slumsSafe(x, y int) bool {
	switch d.terrainCode(x, y) {
	case 9, 13, 15:
		return false
	}
	return true
}

// kutoSafe 是古托井（GEO8/29）與井底藏身處（GEO8/32，同一支 `ecl8/29`）裡不跑
// 事件、或事件無害的地形（`9A4Bh` 依 `AFCEh` 的表分派）：0 街道（十分之一隨機
// 遭遇）、2／5／16／17 井邊描述、3／14／19 只關精靈、4 平常無事、9 地下墓穴描述、
// 13 諾里斯死後才有的藏寶、18 「你在廣場閒晃」、20 二十分之一狗頭人。1 是井本身
// （第一次踩：兩場各 9 隻狗頭人，之後問要不要爬下去）、12 是諾里斯（15 隻）、
// 6／7 有人物、10／11／21／22 各有事件。
func (d *mainlineDriver) kutoSafe(x, y int) bool {
	switch d.terrainCode(x, y) {
	case 0, 2, 3, 4, 5, 9, 13, 14, 16, 17, 18, 19, 20:
		return true
	}
	return false
}

// librarySafe 是圖書館裡可以走的格子：不進花園（1）、石化蜥蜴的書架（5）、
// 瘋戰士（11）與會寫 `4A01` 的那一間書架（3）。
func (d *mainlineDriver) librarySafe(x, y int) bool {
	switch d.terrainCode(x, y) {
	case 1, 3, 5, 11:
		return false
	}
	return true
}

// crossSlums 從貧民窟的一側走到另一側再踏出去。east 為真是 (0,4) → (15,4) 東出，
// 否則 (15,4) → (0,4) 西出。路上鎖住的門照探索器撞開；隨機遭遇在 `4A80` 滿 15 之後
// 已經停發，只剩固定事件。
func (d *mainlineDriver) crossSlums(east bool) {
	d.t.Helper()
	targetX, outward := 0, uint8(3)
	if east {
		targetX, outward = 15, 1
	}
	d.leaveMap(fmt.Sprintf("slums → x=%d", targetX), func(x, y int) bool {
		return x == targetX && d.a.initialMap.Grid.CanMoveDungeonWrapped(x, y, int(outward)*2)
	}, d.slumsSafe, true, outward)
}

// crossKutoSouth 從古托井東緣走到南緣踏出去（南邊是圖書館，spec 101 `AFCAh`）。
func (d *mainlineDriver) crossKutoSouth() {
	d.t.Helper()
	d.leaveMap("Kuto's Well → south", func(x, y int) bool {
		return y == 15 && d.a.initialMap.Grid.CanMoveDungeonWrapped(x, y, 4)
	}, d.kutoSafe, false, 2)
}

// crossKutoEast 從古托井南緣回到東緣 (15,4) 踏出去（東邊是貧民窟）。
func (d *mainlineDriver) crossKutoEast() {
	d.t.Helper()
	d.leaveMap("Kuto's Well → east", func(x, y int) bool {
		return x == 15 && y == 4
	}, d.kutoSafe, false, 1)
}

// answerOnce 把文字按過去，遇到第一個選單就選 label（找不到就選第一項），只答一次。
func (d *mainlineDriver) answerOnce(label string) {
	d.t.Helper()
	a := d.a
	for guard := 0; guard < 64; guard++ {
		if a.cellWaitingMenu && len(a.cellMenuOptions) != 0 {
			pick := 0
			for index, option := range a.cellMenuOptions {
				if strings.EqualFold(option, label) {
					pick = index
				}
			}
			for turn := 0; a.cellMenuCursor != pick && turn < 8; turn++ {
				d.step(ebiten.KeyArrowDown)
			}
			d.step(ebiten.KeyEnter)
			return
		}
		if !a.cellEventPending {
			return
		}
		d.step(ebiten.KeyEnter)
	}
}

// approach 走到 target 旁邊一格（allowed 的路），回傳踏進去要面對的方向。
func (d *mainlineDriver) approach(what string, target, allowed func(x, y int) bool) uint8 {
	d.t.Helper()
	a := d.a
	facingTo := func(x, y int) (uint8, bool) {
		for facing := 0; facing < 4; facing++ {
			nx, ny := x+exploreDeltas[facing][0], y+exploreDeltas[facing][1]
			if target(nx, ny) && a.initialMap.Grid.CanMoveDungeonWrapped(x, y, facing*2) {
				return uint8(facing), true
			}
		}
		return 0, false
	}
	beside := func(x, y int) bool { _, ok := facingTo(x, y); return ok && !target(x, y) }
	if !d.walkAllowing(what, beside, allowed, false) {
		d.fatalf("approach %s: cannot get beside the target", what)
	}
	facing, _ := facingTo(int(a.spawn.X), int(a.spawn.Y))
	return facing
}

// fightNorris 走古托井的諾里斯線（`ecl8/29`，exact）：踩井 (7,7) 打兩場各 9 隻
// 狗頭人（`A2D1h`／`A343h`，打完 `4A23 |= 1`）→ 再踩井答 YES 爬下去（`AE4Ah`，
// 換 GEO8/32）→ 井底 (7,7) 的「密門」選單答 NO 留在地下 → 走到地形 12 選 FIGHT
// （諾里斯＋5 蜥蜴人＋9 狗頭人，`9DF0h`）→ 贏了槽 0 變 FEh、`4A0F = 250`（藏身處）
// → 回井底答 YES 爬上來。路上只走 kutoSafe；每場都交給戰術駕駛，受傷就紮營。
func (d *mainlineDriver) fightNorris() {
	d.t.Helper()
	a := d.a
	kuto := gamepack.MapKey{Archive: 8, BlockID: 29}
	if a.spawn.Map != kuto {
		d.fatalf("fightNorris: not in Kuto's Well")
	}
	if a.eventMachine.Memory[0x4A24] == 0xFF {
		d.note("fightNorris: already resolved (4A24=FF)")
		return
	}
	// 井上的「要不要爬下去」與井底的「密門」選單預設都答 NO，只在要爬的那一步答 YES。
	saved := d.prefer
	d.prefer = []string{"FIGHT", "NO"}
	defer func() { d.prefer = saved }()
	well := func(x, y int) bool { return d.terrainCode(x, y) == 1 }
	for guard := 0; guard < 6 && a.eventMachine.Memory[0x4A23]&1 == 0; guard++ {
		facing := d.approach("the well", well, d.kutoSafe)
		d.face(facing)
		d.step(ebiten.KeyArrowUp)
		d.settle()
		d.restIfHurt()
		d.note("fightNorris: well visit %d 4A23=%02X 4A22=%d %s", guard+1,
			a.eventMachine.Memory[0x4A23], a.eventMachine.Memory[0x4A22], d.partyLine())
	}
	if a.eventMachine.Memory[0x4A23]&1 == 0 {
		d.fatalf("fightNorris: the well never opened (4A23=%02X)", a.eventMachine.Memory[0x4A23])
	}
	// 爬下去：從旁邊踏上井，第一個選單答 YES。
	facing := d.approach("the well again", well, d.kutoSafe)
	d.face(facing)
	d.step(ebiten.KeyArrowUp)
	d.answerOnce("YES")
	d.settle()
	if a.spawn.Map == kuto {
		d.fatalf("fightNorris: climbing down did not change the map: 4A10=%d text=%q",
			a.eventMachine.Memory[0x4A10], a.eventText)
	}
	d.note("fightNorris: underground at %+v 4A10=%d", a.spawn, a.eventMachine.Memory[0x4A10])
	norris := func(x, y int) bool { return d.terrainCode(x, y) == 12 }
	trace := d.trace
	d.trace = true
	if !d.walkAllowing("Norris", norris, d.kutoSafe, false) {
		d.fatalf("fightNorris: cannot reach Norris's hall")
	}
	d.settle()
	d.trace = trace
	d.restIfHurt()
	d.note("fightNorris: 4A24=%02X slot0=%02X 4A0F=%d at (%d,%d) text=%q",
		a.eventMachine.Memory[0x4A24], a.eventMachine.Memory[0x4AA6], a.eventMachine.Memory[0x4A0F],
		a.spawn.X, a.spawn.Y, strings.TrimSpace(a.eventText))
	if a.eventMachine.Memory[0x4AA6] != uint16(gamepack.CityHallSlotPending) {
		d.fatalf("fightNorris: slot 0 is %02X, want FE", a.eventMachine.Memory[0x4AA6])
	}
	// 爬上來：從旁邊踏回井底 (7,7)（地形 16，`9A81h` → `A1CCh`），答 YES。
	bottom := func(x, y int) bool { return d.terrainCode(x, y) == 16 }
	facing = d.approach("well bottom", bottom, d.kutoSafe)
	d.face(facing)
	d.step(ebiten.KeyArrowUp)
	d.answerOnce("YES")
	d.settle()
	if a.spawn.Map != kuto {
		d.fatalf("fightNorris: climbing up did not return to Kuto's Well: %+v text=%q", a.spawn, a.eventText)
	}
	d.note("fightNorris: back at %+v 4A10=%d", a.spawn, a.eventMachine.Memory[0x4A10])
}

// pickDoor 站在 (x,y) 朝 facing 撞門，選 PICK 直到門開，再踏過去。上限 attempts 次。
func (d *mainlineDriver) pickDoor(what string, x, y int, facing uint8, attempts int) bool {
	d.t.Helper()
	a := d.a
	for attempt := 1; attempt <= attempts; attempt++ {
		d.settle()
		if int(a.spawn.X) != x || int(a.spawn.Y) != y {
			d.fatalf("pickDoor %s: standing at (%d,%d), want (%d,%d)", what, a.spawn.X, a.spawn.Y, x, y)
		}
		d.face(facing)
		d.step(ebiten.KeyArrowUp)
		if a.door == nil {
			if int(a.spawn.X) != x || int(a.spawn.Y) != y {
				d.note("pickDoor %s: the door was already open (attempt %d)", what, attempt)
				return true
			}
			d.fatalf("pickDoor %s: no door menu and no movement facing %d from (%d,%d): %q", what, facing, x, y, a.statusLine)
		}
		picked := false
		for _, option := range a.door.Options {
			if option == doorOptionPick {
				picked = true
			}
		}
		if !picked {
			d.fatalf("pickDoor %s: PICK is not offered: %v", what, a.door.Options)
		}
		for guard := 0; a.door != nil && a.door.Options[a.door.Cursor] != doorOptionPick && guard < 8; guard++ {
			d.step(ebiten.KeyArrowRight)
		}
		d.step(ebiten.KeyEnter)
		if a.door == nil {
			d.note("pickDoor %s: the lock gave way on attempt %d", what, attempt)
			d.face(facing)
			d.step(ebiten.KeyArrowUp)
			return true
		}
		// 失敗：EXIT 收掉選單，再撞一次是新的額度。
		for guard := 0; a.door != nil && a.door.Options[a.door.Cursor] != doorOptionExit && guard < 8; guard++ {
			d.step(ebiten.KeyArrowRight)
		}
		d.step(ebiten.KeyEnter)
	}
	d.note("pickDoor %s: %d attempts, still locked", what, attempts)
	return false
}

// bookBits 是 `4A2Fh` 的六本書位元（spec 041 槽 4..9）。
func (d *mainlineDriver) bookBits() uint16 { return d.a.eventMachine.Memory[0x4A2F] }

// searchStacks 站在 room 的每一格書架上按 L）OOK 直到 want 的位元都立起來。
// 每次 L 是一次 `RANDOM`，找到書會問「DO YOU TAKE IT?」，答 YES。
func (d *mainlineDriver) searchStacks(what string, room func(x, y int) bool, want uint16, presses int) bool {
	d.t.Helper()
	a := d.a
	for guard := 0; guard < 8 && d.bookBits()&want != want; guard++ {
		if !d.walkAllowing(what, room, d.librarySafe, false) {
			d.fatalf("searchStacks %s: cannot reach the stacks", what)
		}
		for press := 0; press < presses && d.bookBits()&want != want; press++ {
			d.settle("YES")
			d.step(ebiten.KeyL)
			d.settle("YES")
		}
		d.note("searchStacks %s: 4A2F=%02X 4A30=%d at (%d,%d)", what, d.bookBits(),
			a.eventMachine.Memory[0x4A30], a.spawn.X, a.spawn.Y)
	}
	return d.bookBits()&want == want
}

// libraryBooks 從圖書館北緣 (x,0) 進門、搜兩間書架。回傳拿到的書位元。
// **拿了書就出不去**：兩道門的出口方向都會召出幽靈（`99F8h`，見檔頭），而且
// 撞閂住的門那一下也算一步（remake 照 spec 101 在被擋住時仍跑入口 0）——
// 所以這一段只能證明「一級隊伍拿得到書」，主線探針不走它；見
// `TestLibraryBooksSummonTheSpectreOnTheWayOut`。
func (d *mainlineDriver) libraryBooks() uint16 {
	d.t.Helper()
	a := d.a
	street := func(x, y int) bool { return d.terrainCode(x, y) == 0 }
	if !d.walkAllowing("library north door", func(x, y int) bool { return x == 8 && y == 1 }, street, false) {
		d.fatalf("libraryBooks: cannot reach (8,1)")
	}
	if !d.pickDoor("library north door (in)", 8, 1, 2, 40) {
		d.fatalf("libraryBooks: the north door stayed shut")
	}
	// 歷史那一間（地形 4，(10..12,2..4) 的環）：書 3／4／5 ＝ 位元 4／8／16。
	history := func(x, y int) bool { return d.terrainCode(x, y) == 4 && !(x == 11 && y == 3) }
	d.searchStacks("history stacks", history, 4|8|16, 60)
	// 哲學那一間（地形 2，(12..14,6..8) 的環）：書 1／2 ＝ 位元 1／2，小書位元 128。
	philosophy := func(x, y int) bool { return d.terrainCode(x, y) == 2 && !(x == 13 && y == 7) }
	d.searchStacks("philosophy stacks", philosophy, 1|2|128, 60)
	bits := d.bookBits()
	pending := 0
	for slot := uint16(4); slot <= 9; slot++ {
		if a.eventMachine.Memory[0x4AA6+slot] == uint16(gamepack.CityHallSlotPending) {
			pending++
		}
	}
	d.note("libraryBooks: 4A2F=%02X pending slots=%d 4A00=%d 4A01=%d 4A31=%02X", bits, pending,
		a.eventMachine.Memory[0x4A00], a.eventMachine.Memory[0x4A01], a.eventMachine.Memory[0x4A31])
	return bits
}

// bumpLibraryNorthDoorFromInside 從 (7,2) 朝東踏上北門格（這一步朝向 1，不召幽靈），
// 轉北撞門。回傳撞門那一下之後的事件文字。
func (d *mainlineDriver) bumpLibraryNorthDoorFromInside() string {
	d.t.Helper()
	a := d.a
	if !d.walkAllowing("library (7,2)", func(x, y int) bool { return x == 7 && y == 2 }, d.librarySafe, false) {
		d.fatalf("cannot reach (7,2)")
	}
	d.face(1)
	d.step(ebiten.KeyArrowUp)
	d.settle()
	if a.tactical != nil || int(a.spawn.X) != 8 || int(a.spawn.Y) != 2 {
		d.fatalf("stepping east onto the door cell ended at (%d,%d)", a.spawn.X, a.spawn.Y)
	}
	d.face(0)
	d.step(ebiten.KeyArrowUp)
	return a.eventText
}

// enterTrainingHall 從城區走到 (5,2) 朝東踏進訓練所（ECL3/11 共用 GEO3/0），
// 再走到牧師那道門 (5,0)，答 YES 進隊伍管理，對每個人按 T。remake 的訓練所
// 不分職業（training.go 檔頭）、也還沒收 1000 金（說明書 p.9；原版收費的碼
// 沒讀到），所以一趟就把六個人都升完。回程從 (6,2) 朝西踏回城區。
// `4A00` 會被公告牌寫成 1（`ecl3/11 9AE6h`），要靠市政廳職員清掉。
func (d *mainlineDriver) enterTrainingHall() []string {
	d.t.Helper()
	a := d.a
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		d.fatalf("enterTrainingHall: not in the city")
	}
	street := func(x, y int) bool { return d.terrainCode(x, y) == 0 }
	if !d.walkAllowing("training hall (5,2)", func(x, y int) bool { return x == 5 && y == 2 }, street, false) {
		d.fatalf("enterTrainingHall: cannot reach (5,2)")
	}
	if block := d.stepOut("training hall", 1); block != 11 {
		d.fatalf("enterTrainingHall: (5,2) east led to block %d, want 11", block)
	}
	inside := func(x, y int) bool { return (x == 6 && y <= 2) || (x == 5 && y == 0) }
	if !d.walkAllowing("clerics' door", func(x, y int) bool { return x == 5 && y == 0 }, inside, false) {
		d.fatalf("enterTrainingHall: cannot reach the clerics' door")
	}
	for guard := 0; guard < 64 && !a.programManaging; guard++ {
		d.settle("YES")
		if !a.programManaging && !d.busy() {
			d.step(ebiten.KeyEnter)
		}
	}
	if !a.programManaging {
		d.fatalf("enterTrainingHall: the clerics' door did not open party management: %q", a.eventText)
	}
	lines := []string{}
	keys := []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4,
		ebiten.KeyDigit5, ebiten.KeyDigit6}
	for index := range a.state.Party {
		d.step(keys[index])
		d.step(ebiten.KeyT)
		// 原版問 "Do you wish to train?"（#29）；沒問就是被清醒、1000 金或職業門擋下，
		// 狀態列會說是哪一道。
		if a.trainPending {
			d.step(ebiten.KeyY)
		}
		member := a.state.Party[index]
		lines = append(lines, fmt.Sprintf("%s: %s (levels=%v xp=%d hp=%d/%d)", a.statusLine,
			member.ClassID, member.ClassLevels, member.Experience, member.CurrentHP, member.MaxHP))
	}
	d.step(ebiten.KeyB)
	if a.programManaging {
		d.fatalf("enterTrainingHall: B did not leave party management")
	}
	d.note("trained: %s", strings.Join(lines, " | "))
	if !d.walkAllowing("training hall exit (6,2)", func(x, y int) bool { return x == 6 && y == 2 }, inside, false) {
		d.fatalf("enterTrainingHall: cannot walk back to (6,2)")
	}
	if block := d.stepOut("training hall exit", 3); block != 0 {
		d.fatalf("enterTrainingHall: (6,2) west led to block %d, want 0", block)
	}
	return lines
}

// canTrain 說隊上有沒有人的經驗值過了下一級的門檻（spec 071 的表）。
func (d *mainlineDriver) canTrain() bool {
	a := d.a
	for _, member := range a.state.Party {
		levels := memberClassLevels(member)
		if eligible, _ := a.levelUpTables.EligibleTrainingMask(levels, member.Experience, a.experienceTable); eligible != 0 {
			return true
		}
	}
	return false
}

// visitClerk 走進市政廳踩職員格再出來——職員 `9C3Ah` 每次都把 `4A00` 清 0
// （spec 137 的死區表），訓練所的公告牌寫了 1 之後要靠這一趟清掉，不然斯托亞諾夫
// 城門的馬車商人不出現。
func (d *mainlineDriver) visitClerk() {
	d.t.Helper()
	a := d.a
	street := func(x, y int) bool { return d.terrainCode(x, y) == 0 }
	if !d.walkAllowing("City Hall door (3,4)", func(x, y int) bool { return x == 3 && y == 4 }, street, false) {
		d.fatalf("visitClerk: cannot reach (3,4)")
	}
	if block := d.stepOut("City Hall", 1); block != 8 {
		d.fatalf("visitClerk: (3,4) east led to block %d, want 8", block)
	}
	hall := func(x, y int) bool { return x >= 4 && x <= 6 && y >= 4 && y <= 6 }
	if !d.walkAllowing("clerk's threshold (4,5)", func(x, y int) bool { return x == 4 && y == 5 }, hall, false) {
		d.fatalf("visitClerk: cannot reach (4,5)")
	}
	d.face(1)
	d.step(ebiten.KeyArrowUp)
	d.settle("Exit")
	d.note("visitClerk: at (%d,%d) 4A00=%d 4A01=%d", a.spawn.X, a.spawn.Y,
		a.eventMachine.Memory[0x4A00], a.eventMachine.Memory[0x4A01])
	if !d.walkAllowing("City Hall door (3,4)", func(x, y int) bool { return x == 3 && y == 4 }, hall, false) {
		if a.eventSession.CurrentBlockID() == 0 {
			return
		}
		d.fatalf("visitClerk: cannot walk back to (3,4)")
	}
	if a.eventSession.CurrentBlockID() != 0 {
		d.fatalf("visitClerk: still in block %d at (3,4)", a.eventSession.CurrentBlockID())
	}
}

// partyLine 是每一段要記的：等級、XP、金幣。
func (d *mainlineDriver) partyLine() string {
	parts := []string{}
	for _, member := range d.a.state.Party {
		levels := []string{}
		for _, level := range memberClassLevels(member) {
			if level != 0 {
				levels = append(levels, fmt.Sprint(level))
			}
		}
		parts = append(parts, fmt.Sprintf("%s L%s xp=%d hp=%d/%d gp=%d pp=%d",
			strings.TrimSpace(member.Name), strings.Join(levels, "/"), member.Experience,
			member.CurrentHP, member.MaxHP, member.Money[3], member.Money[4]))
	}
	return strings.Join(parts, "; ")
}
