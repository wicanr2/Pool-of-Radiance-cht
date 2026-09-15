package main

import (
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// manualPartyBuild 是說明書 p.13 建議的隊伍：兩個牧師（一個專職）、兩個法師、
// 一個賊，其餘兼戰士。race／class 是建角畫面上要按幾下 ↓（照 `creation.Races`
// 與 `ClassesForRace` 的順序），good 是 MODIFY 重擲的收手條件。
// murderTheOldWoman 是「隨機遭遇打滿 15 場之後去殺算命的老婦人」那條路：
// `ecl2/20 A749h` 把隨機計數 `4A80` 歸零、`4A0B = FFh`，之後隨機遭遇的人數 +5、
// 機率 +5，而且屋內休息也會被打斷（入口 2 `9A0Eh`）。玩家不會這樣做——
// 差額由 14 個固定事件補。留著這個開關只是為了對照。
const murderTheOldWoman = false

var manualPartyBuild = []struct {
	name        rune
	race, class int
	classID     string
	good        func(abilities [6]int, hp int) bool
}{
	{'A', 5, 0, "cleric", func(ab [6]int, hp int) bool { return ab[gamepack.AbilityWisdom] >= 15 && hp >= 7 }},
	{'B', 3, 4, "cleric-fighter", func(ab [6]int, hp int) bool {
		return ab[gamepack.AbilityStrength] >= 16 && ab[gamepack.AbilityWisdom] >= 13 && hp >= 7
	}},
	{'C', 1, 3, "fighter-magic-user", func(ab [6]int, hp int) bool {
		return ab[gamepack.AbilityStrength] >= 16 && ab[gamepack.AbilityIntelligence] >= 13 && hp >= 6
	}},
	{'D', 5, 2, "magic-user", func(ab [6]int, hp int) bool { return ab[gamepack.AbilityIntelligence] >= 15 && hp >= 4 }},
	{'E', 3, 8, "fighter-thief", func(ab [6]int, hp int) bool { return ab[gamepack.AbilityStrength] >= 16 && hp >= 7 }},
	{'F', 5, 1, "fighter", func(ab [6]int, hp int) bool { return ab[gamepack.AbilityStrength] >= 17 && hp >= 9 }},
}

func TestMainlineProbeNaturalPartyFirstBattle(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	statePath := filepath.Join(t.TempDir(), "state.json")
	application, err := newApp(zipPath, statePath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.roller = diceRoller{random: rand.New(rand.NewSource(136))}
	application.eclSeed = 1
	text := &scriptedTextKeys{scriptedKeys: scriptedKeys{}}
	application.keys = text
	step := func(key ebiten.Key, chars ...rune) {
		t.Helper()
		application.keys = text
		text.scriptedKeys = scriptedKeys{key: true}
		text.chars = chars
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	idle := func() {
		t.Helper()
		application.keys = text
		text.scriptedKeys, text.chars = scriptedKeys{}, nil
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}

	step(ebiten.KeyEnter)
	// 隊伍照說明書 p.13 的建議組：「二個牧師，兩個魔法師和一個賊，其中至少要有
	// 一個專職牧師……其他的最好全部都兼戰士」；「全是戰士的隊伍當然無法長期生存」。
	// 種族與職業的游標位置照 `creation.Races` 與 `ClassesForRace` 的順序。
	for index, member := range manualPartyBuild {
		step(ebiten.KeyC)
		for down := 0; down < member.race; down++ {
			step(ebiten.KeyArrowDown)
		}
		step(ebiten.KeyEnter)
		step(ebiten.KeyEnter)
		for down := 0; down < member.class; down++ {
			step(ebiten.KeyArrowDown)
		}
		step(ebiten.KeyEnter)
		step(ebiten.KeyEnter)
		idle()
		step(ebiten.KeyEnter)
		step(ebiten.KeyEnter, member.name)
		step(ebiten.KeyK)
		step(ebiten.KeyE)
		step(ebiten.KeyY)
		step(ebiten.KeyA)
		if len(application.state.Party) != index+1 {
			t.Fatalf("after creating %c party size=%d, want %d", member.name,
				len(application.state.Party), index+1)
		}
		if got := application.state.Party[index].ClassID; got != member.classID {
			t.Fatalf("created %c as %q, want %q (cursor race %d class %d)", member.name, got,
				member.classID, member.race, member.class)
		}
	}
	// 玩家策略層第三條：M）ODIFY CHARACTER 重擲（說明書 p.8：經驗 0、身上只有錢
	// 的新人物可以「重新調整屬性與生命力」）。原版玩家開場就是這樣把戰士擲到
	// 高力量高生命；這裡對每個人按 1..6 選人再按 M，擲到各職業要的主屬性與
	// 生命為止，上限 400 次——全部是隊伍選單上的正常按鍵。
	memberKeys := []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
		ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6}
	for index := range application.state.Party {
		good := func() bool {
			member := application.state.Party[index]
			return manualPartyBuild[index].good(member.Abilities, member.MaxHP)
		}
		step(memberKeys[index])
		tries := 0
		for ; tries < 400 && !good(); tries++ {
			step(ebiten.KeyM)
		}
		t.Logf("modify %s: %d rerolls → %v hp=%d", application.state.Party[index].Name, tries,
			application.state.Party[index].Abilities, application.state.Party[index].MaxHP)
	}
	for index, member := range application.state.Party {
		t.Logf("party %d %s %s abilities=%v hp=%d money=%v inventory=%d", index,
			member.Name, member.ClassID, member.Abilities, member.CurrentHP, member.Money, len(member.Inventory))
	}
	step(ebiten.KeyB)
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			step(ebiten.KeyEnter)
			continue
		}
		idle()
	}
	if !application.introDone {
		t.Fatal("opening did not finish")
	}
	// 玩家策略層第二條：先去武具店把金幣換成裝備（mainline_outfit_test.go）。
	outfitter := &mainlineDriver{t: t, a: application, pilot: &tacticalPilot{},
		step: func(key ebiten.Key) { step(key) }}
	outfitter.outfitParty()
	// 第四條：牧師記輕傷治療、法師記催眠術（mainline_spells_test.go）；休息之後生效。
	t.Logf("memorised %d spells before leaving the city", outfitter.memoriseSpells())
	application.keys = text
	for _, member := range application.state.Party {
		items := []string{}
		for _, item := range member.Inventory {
			items = append(items, fmt.Sprintf("%s(ready=%d)", item.Name, item.Raw[itemReadyOffset]))
		}
		armour, movement, err := application.memberDefenceStats(member, creationArmorClassInternal, creationBaseMovement)
		t.Logf("equipped %s: ac=%d move=%d err=%v items=%v", strings.TrimSpace(member.Name), armour, movement, err, items)
	}
	for count := 0; count < 64 && application.encounter == nil; count++ {
		key := ebiten.KeyArrowUp
		if application.cellEventPending || application.cellWaitingMenu {
			key = ebiten.KeyEnter
		}
		step(key)
	}
	if application.spawn.Map != (gamepack.MapKey{Archive: 2, BlockID: 20}) {
		t.Fatalf("walking west from the gate did not enter the slums: %+v", application.spawn)
	}
	// 走進貧民窟的頭幾步不一定會遇敵（遭遇擲骰在 spec 136）；遇到了就在這裡打，
	// 沒遇到就交給下面的巡邏。
	if application.encounter != nil {
		if err := selectMenuOption(t, application, "COMBAT"); err != nil {
			t.Fatal(err)
		}
		for guard := 0; guard < 8 && !application.combatActive; guard++ {
			step(ebiten.KeyEnter)
		}
		if !application.combatActive {
			t.Fatalf("combat not staged: %q", application.eventText)
		}
		step(ebiten.KeyEnter)
		pilot := &tacticalPilot{}
		for tick := 0; tick < 20000 && application.tactical != nil; tick++ {
			step(pilot.key(application))
		}
		if application.tactical != nil {
			t.Fatalf("battle stalled at round %d mover %d", application.tactical.Round,
				application.tactical.Mover)
		}
		for index, member := range application.state.Party {
			t.Logf("after %d %s hp=%d status=%d", index, member.Name, member.CurrentHP, member.Status)
		}
		t.Logf("battle ended: combat=%v text=%q", application.combatActive, application.eventText)
	}

	visited := map[[3]int]bool{}
	maps := map[string]bool{}
	blocks := map[int]bool{}
	flags := map[uint16]uint16{}
	var failures []string
	slums := application.spawn.Map
	var planInside func(wanted func(x, y int) bool, rotate int) []exploreStep
	// planStreet 是 planInside，但路上只走地形碼 0 的街道格：屋內的事件格
	// 踩到就跑事件——潛在委託人那一間（地形 3）答 LEAVE 會把隊伍搬回門口
	// (14,10)，規劃器再把它排進路徑，就永遠走不到邊界。
	var planStreet func(wanted func(x, y int) bool, rotate int) []exploreStep
	walkThroughBoundary := func(from gamepack.MapKey) {
		t.Helper()
		application.keys = text
		pilot := &tacticalPilot{}
		doorAttempts := map[[5]int]map[string]bool{}
		targetX, outward := 0, uint8(3)
		if from == (gamepack.MapKey{Archive: 2, BlockID: 20}) {
			targetX, outward = 15, 1
		}
		isExit := func(x, y int) bool {
			return x == targetX &&
				application.initialMap.Grid.CanMoveDungeonWrapped(x, y, int(outward)*2)
		}
		for guard := 0; guard < 20000 && application.spawn.Map == from; guard++ {
			if application.gameOver {
				t.Fatalf("the party was destroyed while leaving %+v: %q (4ABB=%02X)", from,
					application.eventText, application.eventMachine.Memory[0x4ABB])
			}
			switch {
			case application.door != nil && !application.cellEventPending:
				key := explorerDoorKey(application)
				tried := doorAttempts[key]
				if tried == nil {
					tried = map[string]bool{}
					doorAttempts[key] = tried
				}
				want := explorerDoorChoice(application.door.Options, tried)
				if application.door.Options[application.door.Cursor] != want {
					step(ebiten.KeyArrowRight)
					continue
				}
				tried[want] = true
				step(ebiten.KeyEnter)
				continue
			case application.tactical != nil:
				step(pilot.key(application))
				continue
			case application.encounter != nil:
				if err := selectMenuOption(t, application, "COMBAT"); err != nil {
					t.Fatal(err)
				}
				continue
			case application.combatActive:
				step(ebiten.KeyEnter)
				continue
			}
			if key, busy := escapeKeyForWalk(application); busy {
				step(key)
				continue
			}
			if isExit(int(application.spawn.X), int(application.spawn.Y)) {
				if application.spawn.Facing != outward {
					step(ebiten.KeyArrowRight)
					continue
				}
				before := application.spawn
				step(ebiten.KeyArrowUp)
				if application.spawn.Map == from {
					block := uint16(0)
					if application.eventSession != nil {
						block = application.eventSession.CurrentBlockID()
					}
					t.Fatalf("outward step from %+v stayed at %+v: 6DD5=%d ecl=%d/%d status=%q event=%q",
						before, application.spawn, application.eventMachine.Memory[0x6DD5],
						application.eclArchive, block, application.statusLine, application.eventText)
				}
				continue
			}
			plan := planStreet(isExit, 0)
			if len(plan) == 0 {
				// 站在屋子裡出不到街上：先照屋內的路走出去。
				plan = planInside(isExit, 0)
			}
			if len(plan) == 0 {
				t.Fatalf("cannot reach boundary from %+v", application.spawn)
			}
			want := plan[0]
			if application.spawn.Facing != want.facing {
				key := ebiten.KeyArrowRight
				if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
					key = ebiten.KeyArrowLeft
				}
				step(key)
				continue
			}
			step(ebiten.KeyArrowUp)
		}
		if application.spawn.Map == from {
			cell, _ := application.initialMap.Grid.Cell(int(application.spawn.X), int(application.spawn.Y))
			t.Fatalf("boundary did not leave %+v: position=%+v terrain=%d status=%q event=%q pending=%v menu=%v/%v@%d door=%v C04B=%d,%d,%d",
				from, application.spawn, cell.Terrain&0x7F, application.statusLine, application.eventText,
				application.cellEventPending, application.cellWaitingMenu, application.cellMenuOptions,
				application.cellMenuCursor, application.door != nil,
				application.eventMachine.Memory[0xC04B], application.eventMachine.Memory[0xC04C],
				application.eventMachine.Memory[0xC04D])
		}
	}
	planStreetOrInside := func(wanted func(x, y int) bool, rotate int, streetOnly bool) []exploreStep {
		type node struct{ x, y int }
		start := node{int(application.spawn.X), int(application.spawn.Y)}
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
			for offset := 0; offset < 4; offset++ {
				facing := (offset + rotate) % 4
				if !application.initialMap.Grid.CanMoveDungeonWrapped(current.x, current.y, facing*2) {
					flags, door := application.initialMap.Grid.WallDoorFlagsWrapped(
						current.x, current.y, facing*2)
					if !door || flags != gamepack.DoorLocked && flags != gamepack.DoorBarred {
						continue
					}
				}
				next := node{current.x + exploreDeltas[facing][0], current.y + exploreDeltas[facing][1]}
				if next.x < 0 || next.x >= 16 || next.y < 0 || next.y >= 16 {
					continue
				}
				if _, seen := from[next]; seen {
					continue
				}
				if streetOnly && !wanted(next.x, next.y) {
					if cell, ok := application.initialMap.Grid.Cell(next.x, next.y); ok && cell.Terrain&0x7F != 0 {
						continue
					}
				}
				from[next], via[next] = current, uint8(facing)
				queue = append(queue, next)
			}
		}
		return nil
	}
	planInside = func(wanted func(x, y int) bool, rotate int) []exploreStep {
		return planStreetOrInside(wanted, rotate, false)
	}
	planStreet = func(wanted func(x, y int) bool, rotate int) []exploreStep {
		return planStreetOrInside(wanted, rotate, true)
	}
	completeOldWoman := func() {
		t.Helper()
		application.keys = text
		pilot := &tacticalPilot{}
		for guard := 0; guard < 40000 && application.eventMachine.Memory[0x4A0B] != 0xFF; guard++ {
			switch {
			case application.tactical != nil:
				step(pilot.key(application))
				continue
			case application.encounter != nil:
				if err := selectMenuOption(t, application, "COMBAT"); err != nil {
					t.Fatal(err)
				}
				continue
			case application.combatActive:
				step(ebiten.KeyEnter)
				continue
			}
			if application.door != nil && !application.cellEventPending {
				want := "BASH"
				for _, option := range application.door.Options {
					if option == "PICK" {
						want = option
						break
					}
				}
				if application.door.Options[application.door.Cursor] != want {
					step(ebiten.KeyArrowRight)
				} else {
					step(ebiten.KeyEnter)
				}
				continue
			}
			attack := false
			for _, option := range application.cellMenuOptions {
				if option == "ATTACK" {
					attack = true
					break
				}
			}
			if attack {
				if err := selectMenuOption(t, application, "ATTACK"); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if key, busy := escapeKeyForWalk(application); busy {
				step(key)
				continue
			}
			wanted := func(x, y int) bool {
				cell, ok := application.initialMap.Grid.Cell(x, y)
				return ok && cell.Terrain&0x7F == 8
			}
			plan := planInside(wanted, guard%4)
			if len(plan) == 0 {
				t.Fatalf("cannot reach the slums' terrain-8 old-woman event from %+v", application.spawn)
			}
			want := plan[0]
			if application.spawn.Facing != want.facing {
				key := ebiten.KeyArrowRight
				if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
					key = ebiten.KeyArrowLeft
				}
				step(key)
				continue
			}
			step(ebiten.KeyArrowUp)
		}
		if application.eventMachine.Memory[0x4A0B] != 0xFF {
			door := "none"
			if application.door != nil {
				door = application.door.Options[application.door.Cursor]
			}
			t.Fatalf("old-woman event did not finish: 4A0B=%02X at %+v status=%q event=%q door=%s options=%v",
				application.eventMachine.Memory[0x4A0B], application.spawn,
				application.statusLine, application.eventText, door, func() []string {
					if application.door == nil {
						return nil
					}
					return application.door.Options
				}())
		}
	}
	// 玩家策略層第一條：打完看 HP。有人掉到一半以下、或昏迷（狀態 4）就停下
	// 探索，找地方紮營休息到滿——每二十四小時回一點（spec 114）。
	hurt := func(a *app) bool {
		// 打到一半不算：戰鬥的生命值要等 `finishCombat` 才寫回隊伍，而續戰
		// 提示（`CONTINUE BATTLE`）還開著時按 E 開不了營。
		if a.tactical != nil {
			return false
		}
		for _, member := range a.state.Party {
			if member.Status == 4 || (member.Status == 0 && member.CurrentHP*2 < member.MaxHP) {
				return true
			}
		}
		// 催眠術用完也算：沒有催眠的一場架（13 名哥布林）就是全滅的那一場。
		// 原版玩家每打完一場就回去休息重記，這裡照做。
		return !sleepReady(a)
	}
	// restUntilHealed 就地紮營到全隊回滿：按鍵照原版紮營畫面（spec 135）：
	// E 紮營、R 排時間、Y 選天、I 加一天、R 休息、ESC 收掉。天數是全隊缺最多
	// 的那一位（每 24 小時回 1 點，spec 114）。
	//
	// 不再先走去屋內：打完一場常常只剩兩個人站著，走去找床的路上再撞一場
	// 就是全滅（實測第 14 場之後）。原版在街上休息會被打斷（`ecl2/20` 入口 2
	// 把打斷參數設成 24／24，屋內 0／0），remake 還沒跑那個入口（#24）；接上
	// 之後被打斷的那一場由同一個駕駛打，打完再睡，這裡的迴圈就是為那一天留的。
	restUntilHealed := func() {
		t.Helper()
		restPilot := &tacticalPilot{}
		settle := func() {
			for guard := 0; guard < 400 && (application.cellEventPending || application.cellWaitingMenu ||
				application.tactical != nil); guard++ {
				if application.tactical != nil {
					step(restPilot.key(application))
					continue
				}
				step(ebiten.KeyEnter)
			}
			if application.gameOver {
				t.Fatalf("the party was destroyed while resting: %q", application.eventText)
			}
		}
		for attempt := 0; attempt < 8 && (hurt(application) || pendingMemorisation(application)); attempt++ {
			settle()
			// 打過架用掉的法術格先補記，這一次休息順便記完。
			outfitter.memoriseSpells()
			days := 0
			for _, member := range application.state.Party {
				if member.Status == 0 || member.Status == 4 {
					if missing := member.MaxHP - member.CurrentHP; missing > days {
						days = missing
					}
				}
			}
			if days == 0 && pendingMemorisation(application) {
				days = 1
			}
			before := application.gameTime
			step(ebiten.KeyE)
			if !application.campOpen {
				t.Fatalf("E did not open the camp at %+v: %q (mode=%d help=%t tactical=%t event=%t menu=%t door=%t shop=%t program=%t text=%q)",
					application.spawn, application.statusLine, application.mode, application.help,
					application.tactical != nil, application.cellEventPending, application.cellWaitingMenu,
					application.door != nil, application.shopActive, application.campFromProgram, application.eventText)
			}
			step(ebiten.KeyR)
			step(ebiten.KeyY)
			for guard := 0; guard < 64 && !application.restDuration.IsZero(); guard++ {
				step(ebiten.KeyD)
			}
			for day := 0; day < days; day++ {
				step(ebiten.KeyI)
			}
			step(ebiten.KeyR)
			for guard := 0; guard < 8 && (application.campOpen || application.campFromProgram); guard++ {
				step(ebiten.KeyEscape)
			}
			settle()
			hp := []string{}
			for _, member := range application.state.Party {
				hp = append(hp, fmt.Sprintf("%s %d/%d st%d", strings.TrimSpace(member.Name), member.CurrentHP, member.MaxHP, member.Status))
			}
			t.Logf("rested %d days at %+v (clock %v → %v): %v", days, application.spawn, before, application.gameTime, hp)
		}
		if hurt(application) || pendingMemorisation(application) {
			t.Fatalf("still hurt or unmemorised after resting at %+v (status %q)", application.spawn, application.statusLine)
		}
	}
	reachable := true
	var lastBattle *tacticalState
	lastStatus := ""
	// 貧民窟一級隊伍打得起的只有 20 場（實跑紀錄「補四」）：連續三趟 `4ABB`
	// 沒動就收手，先去做索寇要塞，回頭再補。
	stalled, lastCount := 0, uint16(0)
	for patrol := 0; patrol < 40 && application.eventMachine.Memory[0x4ABB] != 0xFE && stalled < 3; patrol++ {
		if count := application.eventMachine.Memory[0x4ABB]; count == lastCount {
			stalled++
		} else {
			stalled, lastCount = 0, count
		}
		if application.gameOver {
			t.Fatalf("the party was destroyed in the slums: %q (4ABB=%02X, patrol %d)",
				application.eventText, application.eventMachine.Memory[0x4ABB], patrol)
		}
		restUntilHealed()
		avoid := map[[3]int]bool{}
		for _, key := range boundaryExitKeys(application) {
			avoid[key] = true
		}
		// 一級隊伍打不起的固定事件不踩（`ecl2/20` 入口 1 依地形碼分派）：
		// 9 獸人的家（20+4）、13 衛兵攔截（30+4）、15 驚動衛兵（12+21）。
		// 兩發催眠只放得倒 8 個生命骰、五回合就醒，這三場實測（15）全滅。
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				cell, ok := application.initialMap.Grid.Cell(x, y)
				if !ok {
					continue
				}
				switch cell.Terrain & 0x7F {
				case 9, 13, 15:
					avoid[[3]int{int(slums.Archive), int(slums.BlockID), y*100 + x}] = true
				}
			}
		}
		_, reachable = exploreWorldWithFlags(t,
			filepath.Join("..", "..", "Pool of Radiance (1988).zip"), 136, 0, 8, 300000,
			avoid, map[[3]int]bool{}, map[[3]int]int{}, map[string]int{},
			map[[4]int]int{}, visited, maps, blocks, flags, noBoatOverride, &failures, nil,
			application, &slums, func(a *app) bool {
				if a.tactical != nil && a.tactical != lastBattle {
					lastBattle = a.tactical
					names := []string{}
					for _, monster := range a.combatMonsters {
						names = append(names, fmt.Sprintf("%s×%d hp=%d ac=%d", monster.Record.Name,
							monster.Spawn.Count, monster.Record.MaxHitPoints(), monster.Record.ArmorClass()))
					}
					roster := []string{}
					for index := 1; index < len(a.tactical.Roster); index++ {
						roster = append(roster, fmt.Sprintf("%d:%s hp=%d ac=%d thac0=%d", index,
							map[bool]string{true: "P", false: "M"}[a.tactical.Friendly[index]],
							a.tactical.HitPoints[index], a.tactical.ArmorClass[index], a.tactical.THAC0[index]))
					}
					spells := []string{}
					for _, member := range a.state.Party {
						ready := []uint8{}
						for _, option := range a.spellOptionsFor(member) {
							ready = append(ready, option.ID)
						}
						spells = append(spells, fmt.Sprintf("%s:%v/%v", strings.TrimSpace(member.Name), ready, member.Memorised))
					}
					t.Logf("battle: %v roster=%v spells=%v", names, roster, spells)
				}
				if a.tactical != nil && a.tactical.Status != lastStatus {
					lastStatus = a.tactical.Status
					t.Logf("  r%d m%d %s / %s", a.tactical.Round, a.tactical.Mover, a.tactical.Status, a.tactical.FoeLog)
				}
				return a.eventMachine != nil && (a.eventMachine.Memory[0x4ABB] == 0xFE || hurt(a))
			}, false)
		if !reachable {
			t.Fatal("existing application was rejected while clearing the slums")
		}
		hp := []string{}
		for _, member := range application.state.Party {
			hp = append(hp, fmt.Sprintf("%s %d/%d st%d xp%d", strings.TrimSpace(member.Name), member.CurrentHP, member.MaxHP, member.Status, member.Experience))
		}
		t.Logf("slums patrol %d: 4ABB=%02X 4A80=%02X 4A0B=%02X at %+v party=%v",
			patrol+1, application.eventMachine.Memory[0x4ABB],
			application.eventMachine.Memory[0x4A80], application.eventMachine.Memory[0x4A0B],
			application.spawn, hp)
		if application.spawn.Map == slums {
			unvisited := []string{}
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					if visited[[3]int{int(slums.Archive), int(slums.BlockID), y*100 + x}] {
						continue
					}
					cell, _ := application.initialMap.Grid.Cell(x, y)
					unvisited = append(unvisited, fmt.Sprintf("(%d,%d)t%d", x, y, cell.Terrain&0x7F))
				}
			}
			t.Logf("  unvisited: %v", unvisited)
		}
		if application.gameOver {
			t.Fatalf("the party was destroyed in the slums: %q (4ABB=%02X)",
				application.eventText, application.eventMachine.Memory[0x4ABB])
		}
		if application.eventMachine.Memory[0x4ABB] != 0xFE {
			if hurt(application) {
				continue
			}
			if application.eventMachine.Memory[0x4A80] >= 15 &&
				application.eventMachine.Memory[0x4A0B] == 0 && murderTheOldWoman {
				completeOldWoman()
			} else {
				walkThroughBoundary(slums)
				city := application.spawn.Map
				walkThroughBoundary(city)
				if application.spawn.Map != slums {
					t.Fatalf("patrol returned to %+v, want %+v", application.spawn.Map, slums)
				}
			}
		}
	}
	slumsCleared := application.eventMachine.Memory[0x4ABB] == 0xFE
	t.Logf("slums phase ended at 4ABB=%02X (cleared=%t) at %+v; continuing to the city",
		application.eventMachine.Memory[0x4ABB], slumsCleared, application.spawn)
	if application.spawn.Map == slums {
		walkThroughBoundary(slums)
	}
	city := gamepack.MapKey{Archive: 3, BlockID: 0}
	if application.spawn.Map != city {
		t.Fatalf("east slums exit reached %+v, want %+v", application.spawn.Map, city)
	}
	application.keys = text
	for guard := 0; guard < 5000 && slumsCleared; guard++ {
		settled := application.eventMachine.Memory[0x4ABB] == 0xFF &&
			application.eventMachine.Memory[0x4AC1] == 1
		if settled && !application.treasureActive && !application.cellEventPending &&
			!application.cellWaitingMenu {
			break
		}
		if application.treasureActive {
			want := "Exit"
			hasMoney := false
			for _, amount := range application.state.PooledMoney {
				if amount != 0 {
					hasMoney = true
					break
				}
			}
			for _, option := range application.cellMenuOptions {
				if option == "Share" && hasMoney {
					want = option
					break
				}
				if option == "Yes" {
					want = option
				}
			}
			if err := selectMenuOption(t, application, want); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if application.cellEventPending || application.cellWaitingMenu {
			step(ebiten.KeyEnter)
			continue
		}
		plan := cityHallPlan(application, 0)
		if len(plan) == 0 {
			t.Fatalf("no natural City Hall plan at %+v: 4ABB=%02X 4AC1=%d 4A01=%d",
				application.spawn, application.eventMachine.Memory[0x4ABB],
				application.eventMachine.Memory[0x4AC1], application.eventMachine.Memory[0x4A01])
		}
		want := plan[0]
		if application.spawn.Facing != want.facing {
			key := ebiten.KeyArrowRight
			if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
				key = ebiten.KeyArrowLeft
			}
			step(key)
			continue
		}
		step(ebiten.KeyArrowUp)
	}
	if got := application.eventMachine.Memory[0x4ABB]; slumsCleared && got != 0xFF {
		t.Fatalf("City Hall did not acknowledge the natural slums commission: 4ABB=%02X", got)
	}
	if got := application.eventMachine.Memory[0x4AC1]; slumsCleared && got != 1 {
		t.Fatalf("City Hall progress=%d, want 1 after the natural slums commission", got)
	}
	savedSpawn := application.spawn
	savedSlums, savedProgress := application.eventMachine.Memory[0x4ABB], application.eventMachine.Memory[0x4AC1]
	application.keys = text
	text.scriptedKeys = scriptedKeys{ebiten.KeyF10: true}
	text.chars = nil
	if err := application.Update(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10 campaign save returned %v", err)
	}
	restored, err := newApp(zipPath, statePath)
	if err != nil {
		t.Fatal(err)
	}
	application = restored
	// 讀檔之後是另一個 app；照著它走的駕駛要跟著換，不然 K／E 按在舊的那一個上。
	outfitter.a = application
	text = &scriptedTextKeys{scriptedKeys: scriptedKeys{}}
	application.keys = text
	step(ebiten.KeyEnter)
	step(ebiten.KeyL)
	if application.mode != modeAdventure || application.spawn != savedSpawn ||
		application.eventMachine.Memory[0x4ABB] != savedSlums ||
		application.eventMachine.Memory[0x4AC1] != savedProgress {
		t.Fatalf("normal load restored mode=%d spawn=%+v 4ABB=%02X 4AC1=%d, want %+v %02X %d",
			application.mode, application.spawn, application.eventMachine.Memory[0x4ABB],
			application.eventMachine.Memory[0x4AC1], savedSpawn, savedSlums, savedProgress)
	}
	t.Logf("normal F10/L resumed the same campaign at %+v", application.spawn)
	// newApp 會建立新的正式亂數來源；這條可重播測試在讀檔之後重新固定
	// 測試 seed。只固定亂數，不更改正式遊戲狀態、座標、旗標或資源。
	application.roller = diceRoller{random: rand.New(rand.NewSource(136))}
	application.eclSeed = 1
	// 真實的 F10 寫入與標題 L 回讀已在上面完成。後續長程探索仍更新同一份
	// 記憶體狀態，但不必讓每一筆寶物操作都對測試暫存檔做 fsync。
	application.saveState = func(poolsave.State) error { return nil }
	readyToHandIn := func(a *app) bool {
		// LOAD FILES 已把玩家放回城區時，eventSession 的 block 可能還會保留
		// 前一張圖一個 tick；玩家所在的 GEO、原版待交旗標與共用工作格
		// 才是可交件條件。市政廳在 4A01 > 0 時會跳過 reward scan（spec 102）。
		return pendingCommission(a) && a.eclArchive == 3 &&
			a.spawn.Map == (gamepack.MapKey{Archive: cityArchive, BlockID: cityBlock}) &&
			a.eventMachine.Memory[0x4A01] == 0
	}
	clearReturnTicket := func() {
		t.Helper()
		if application.eventMachine.Memory[0x4A01] != 255 {
			return
		}
		ticketMoves := map[string]int{}
		// 港務長：站在 (11,2) 往北踏入地點，正常選 EAST，再由 WHO 選
		// 第一位隊員付款。貧民窟獎賞已經提供足夠白金，不補資源。
		for guard := 0; guard < 2000 && application.eventMachine.Memory[0x4A01] != 1; guard++ {
			switch {
			case application.shopActive:
				step(ebiten.KeyEscape)
			case application.cellWaitingMenu:
				want := application.cellMenuOptions[0]
				for _, option := range application.cellMenuOptions {
					if option == "EAST" {
						want = option
						break
					}
					if option == "Exit" || option == "EXIT" {
						want = option
					}
				}
				if err := selectMenuOption(t, application, want); err != nil {
					t.Fatal(err)
				}
			case application.cellEventPending:
				step(ebiten.KeyEnter)
			default:
				plan := planInside(func(x, y int) bool { return x == 11 && y == 2 }, 0)
				if int(application.spawn.X) == 11 && int(application.spawn.Y) == 2 {
					plan = []exploreStep{{facing: 0}}
				}
				if len(plan) == 0 {
					t.Fatalf("cannot reach the harbour master from %+v", application.spawn)
				}
				want := plan[0]
				if application.spawn.Facing != want.facing {
					ticketMoves[fmt.Sprintf("turn (%d,%d) %d→%d",
						application.spawn.X, application.spawn.Y, application.spawn.Facing, want.facing)]++
					key := ebiten.KeyArrowRight
					if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
						key = ebiten.KeyArrowLeft
					}
					step(key)
				} else {
					before := application.spawn
					step(ebiten.KeyArrowUp)
					ticketMoves[fmt.Sprintf("(%d,%d)→(%d,%d) dir=%d",
						before.X, before.Y, application.spawn.X, application.spawn.Y, before.Facing)]++
				}
			}
		}
		if application.eventMachine.Memory[0x4A01] != 1 {
			t.Fatalf("harbour master did not issue the normal EAST ticket: archive=%d block=%d spawn=%+v mode=%d waiting=%t pending=%t temple=%t shop=%t camp=%t panel=%t door=%t options=%v text=%q status=%q ticket=%d moves=%v",
				application.eclArchive, application.eventSession.CurrentBlockID(), application.spawn,
				application.mode, application.cellWaitingMenu, application.cellEventPending,
				application.templeActive, application.shopActive, application.campOpen,
				application.panelOpen(), application.door != nil, application.cellMenuOptions,
				application.eventText, application.statusLine, application.eventMachine.Memory[0x4A01], ticketMoves)
		}
		// 踩上 (15,1) 的碼頭；東航線進 block 27 時，原版 staging 將
		// 4A01 清為 0。
		for guard := 0; guard < 2000 && application.eclArchive == 3; guard++ {
			if application.cellEventPending || application.cellWaitingMenu {
				step(ebiten.KeyEnter)
				continue
			}
			plan := planToCellsAvoiding(application, 0,
				func(x, y int) bool { return x == 15 && y == 1 },
				func(x, y int) bool {
					if x == 15 && y == 1 {
						return false
					}
					cell, ok := application.initialMap.Grid.Cell(x, y)
					return ok && cell.Terrain&0x7F != 0
				})
			if len(plan) == 0 {
				t.Fatalf("cannot reach the pier from %+v", application.spawn)
			}
			want := plan[0]
			if application.spawn.Facing != want.facing {
				key := ebiten.KeyArrowRight
				if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
					key = ebiten.KeyArrowLeft
				}
				step(key)
			} else {
				step(ebiten.KeyArrowUp)
			}
		}
		for guard := 0; guard < 200; guard++ {
			if application.inWildernessOverland() && application.eventMachine.Memory[0x4A01] == 0 {
				break
			}
			if application.cellEventPending || application.cellWaitingMenu {
				step(ebiten.KeyEnter)
			} else {
				idle()
			}
		}
		if !application.inWildernessOverland() {
			t.Fatalf("EAST sailing did not reach wilderness: ECL%d/%d 4A01=%d",
				application.eclArchive, application.eventSession.CurrentBlockID(),
				application.eventMachine.Memory[0x4A01])
		}
		// (9,29) 是同一張原版地點表的回程船。下船時已站在該格，先
		// 沿可通行邊離開一格，再踏回並選 TAKE BOAT。
		here := [2]int{int(application.eventMachine.Memory[wildernessX]),
			int(application.eventMachine.Memory[wildernessY])}
		boat := [2]int{9, 29}
		if here == boat {
			moved := false
			for facing := uint8(0); facing < 4; facing++ {
				next := wildernessAdvance(here, facing)
				if route := wildernessRoute(application, here, next, nil); len(route) != 0 {
					faceTowards(application, route[0])
					step(ebiten.KeyArrowUp)
					moved = true
					break
				}
			}
			if !moved {
				t.Fatal("the EAST landing cannot leave the return-boat cell")
			}
		}
		t.Logf("EAST wilderness entry left scratch/ticket 4A01=%d",
			application.eventMachine.Memory[0x4A01])
		if application.eventMachine.Memory[0x4A01] != 0 {
			localVisited := map[[3]int]bool{}
			localMaps := map[string]bool{}
			localBlocks := map[int]bool{}
			localFlags := map[uint16]uint16{}
			localMenus := map[string]int{
				"5/6/15,1|TAKE BOAT|STAY": 1,
			}
			var localFailures []string
			_, ok := exploreWorldWithFlags(t, zipPath, 313, 0, 8, 100000,
				map[[3]int]bool{}, map[[3]int]bool{}, map[[3]int]int{}, localMenus,
				map[[4]int]int{}, localVisited, localMaps, localBlocks, localFlags,
				noBoatOverride, &localFailures, nil, application, nil,
				func(a *app) bool {
					return a.eclArchive == 3 &&
						a.spawn.Map == (gamepack.MapKey{Archive: cityArchive, BlockID: cityBlock})
				}, false)
			application.keys = text
			if !ok || application.eventMachine.Memory[0x4A01] != 0 {
				t.Fatalf("normal EAST exploration did not clear 4A01: ECL%d/%d at %+v value=%d failures=%v",
					application.eclArchive, application.eventSession.CurrentBlockID(), application.spawn,
					application.eventMachine.Memory[0x4A01], localFailures)
			}
		}
		for guard := 0; guard < 500 && application.eclArchive != 3; guard++ {
			switch {
			case application.cellWaitingMenu:
				want := application.cellMenuOptions[0]
				for _, option := range application.cellMenuOptions {
					if option == "TAKE BOAT" {
						want = option
					}
				}
				if err := selectMenuOption(t, application, want); err != nil {
					t.Fatal(err)
				}
			case application.cellEventPending:
				step(ebiten.KeyEnter)
			default:
				here = [2]int{int(application.eventMachine.Memory[wildernessX]),
					int(application.eventMachine.Memory[wildernessY])}
				route := wildernessRoute(application, here, boat, nil)
				if len(route) == 0 {
					t.Fatalf("cannot walk back to the return boat from %v", here)
				}
				faceTowards(application, route[0])
				step(ebiten.KeyArrowUp)
			}
		}
		if application.eclArchive != 3 || application.eventSession.CurrentBlockID() != 0 {
			t.Fatalf("return boat ended at ECL%d/%d with 4A01=%d",
				application.eclArchive, application.eventSession.CurrentBlockID(),
				application.eventMachine.Memory[0x4A01])
		}
		t.Logf("natural return-ticket lifecycle reached City Hall with 4A01=%d",
			application.eventMachine.Memory[0x4A01])
	}
	// 舊探針曾在交件前另買 EAST 船票，企圖把 4A01 清成 0；實跑證明
	// 市政廳不以船票旗標為閘門，保留探針供診斷但不插入正式主線收據。
	_ = clearReturnTicket
	handInPending := func() {
		t.Helper()
		sharedReward := false
		before := 0
		beforeSlots := []uint16{}
		for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
			if application.eventMachine.Memory[address] == uint16(gamepack.CityHallSlotPending) {
				before++
				beforeSlots = append(beforeSlots, address)
			}
		}
		for guard := 0; guard < 20000; guard++ {
			active := pendingCommission(application) || application.shopActive ||
				application.treasureActive || application.cellWaitingMenu ||
				application.cellEventPending
			if !active {
				break
			}
			switch {
			case application.shopActive:
				step(ebiten.KeyEscape)
			case application.treasureActive:
				want := "Exit"
				if !sharedReward && application.state.PooledMoney != ([7]uint32{}) {
					want = "Share"
				}
				for _, option := range application.cellMenuOptions {
					if option == "Yes" {
						want = option
					}
				}
				if err := selectMenuOption(t, application, want); err != nil {
					t.Fatal(err)
				}
				if want == "Share" {
					sharedReward = true
				}
			case application.cellWaitingMenu:
				want := application.cellMenuOptions[0]
				for _, option := range application.cellMenuOptions {
					if option == "Exit" || option == "EXIT" {
						want = option
					}
				}
				if err := selectMenuOption(t, application, want); err != nil {
					t.Fatal(err)
				}
			case application.cellEventPending:
				step(ebiten.KeyEnter)
			default:
				plan := cityHallPlan(application, 0)
				if len(plan) == 0 {
					t.Fatalf("City Hall pending slot has no normal key route at %+v ECL%d/%d: 4A01=%d 4A06=%d",
						application.spawn, application.eclArchive,
						application.eventSession.CurrentBlockID(),
						application.eventMachine.Memory[0x4A01],
						application.eventMachine.Memory[0x4A06])
				}
				want := plan[0]
				if application.spawn.Facing != want.facing {
					key := ebiten.KeyArrowRight
					if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
						key = ebiten.KeyArrowLeft
					}
					step(key)
				} else {
					step(ebiten.KeyArrowUp)
				}
			}
		}
		after := 0
		afterSlots := []uint16{}
		for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
			if application.eventMachine.Memory[address] == uint16(gamepack.CityHallSlotPending) {
				after++
				afterSlots = append(afterSlots, address)
			}
		}
		if after >= before {
			block := uint16(0)
			if application.eventSession != nil {
				block = application.eventSession.CurrentBlockID()
			}
			t.Fatalf("City Hall did not acknowledge a pending commission: %d %04X → %d %04X at %+v ECL%d/%d app[4AA7]=%d session[4AA7]=%d app[4AC1]=%d session[4AC1]=%d 4A01=%d 4A06=%d pending=%t waiting=%t options=%v text=%q status=%q",
				before, beforeSlots, after, afterSlots, application.spawn, application.eclArchive, block,
				application.eventMachine.Memory[0x4AA7], application.eventSession.Machine().Memory[0x4AA7],
				application.eventMachine.Memory[0x4AC1], application.eventSession.Machine().Memory[0x4AC1],
				application.eventMachine.Memory[0x4A01], application.eventMachine.Memory[0x4A06],
				application.cellEventPending, application.cellWaitingMenu,
				application.cellMenuOptions, application.eventText, application.statusLine)
		}
		t.Logf("City Hall naturally acknowledged pending commissions: %d → %d; progress=%d",
			before, after, application.eventMachine.Memory[0x4AC1])
		// 職員完成結算後還在 ECL3/block8 的市政廳內。先用正常方向鍵
		// 經 (3,4) 的 terrain 26 出門，再往西離開門口；否則通用探索器會把
		// 已探過的市政廳當成新 transition，在 block8 與城區間無限往返。
		for guard := 0; guard < 1000 && application.eclArchive == 3 &&
			application.eventSession.CurrentBlockID() == 8; guard++ {
			if application.cellEventPending || application.cellWaitingMenu {
				step(ebiten.KeyEnter)
				continue
			}
			plan := planToCells(application, 0, func(x, y int) bool { return x == 3 && y == 4 })
			if len(plan) == 0 {
				t.Fatalf("cannot leave City Hall normally from %+v", application.spawn)
			}
			want := plan[0]
			if application.spawn.Facing != want.facing {
				step(ebiten.KeyArrowRight)
			} else {
				step(ebiten.KeyArrowUp)
			}
		}
		if application.eclArchive != 3 || application.eventSession.CurrentBlockID() != 0 {
			t.Fatalf("City Hall exit did not return to ECL3/block0: ECL%d/%d at %+v",
				application.eclArchive, application.eventSession.CurrentBlockID(), application.spawn)
		}
		for application.cellEventPending || application.cellWaitingMenu {
			step(ebiten.KeyEnter)
		}
		plan := planToCells(application, 0, func(x, y int) bool { return x == 2 && y == 4 })
		for len(plan) != 0 {
			want := plan[0]
			if application.spawn.Facing != want.facing {
				step(ebiten.KeyArrowRight)
			} else {
				step(ebiten.KeyArrowUp)
			}
			plan = planToCells(application, 0, func(x, y int) bool { return x == 2 && y == 4 })
		}
	}
	// 第 4／5 段（spec 137）：索寇要塞與交件。要塞裡的走法（密語、費蘭、
	// 亡魂、回程船）由探索器的索寇專用計畫負責（coverage_test.go 的
	// `mustSettleSokalGhost`／`mainlineSokalExitPlan`），這裡只把它框在
	// 「槽 1 結案」這一個出口條件上，趟數有上限。每趟結束檢查城門的前置
	// 狀態沒有被探索器順路弄壞（`4A78 >= 3` 或 `4A77` 的馬車位一旦被寫，
	// 斯托亞諾夫城門就過不去了）。
	transitionUses := map[[3]int]int{}
	menuTurn := map[string]int{}
	exitUses := map[[4]int]int{}
	sokalPasses := 0
	for pass := 0; pass < 24 && application.eventMachine.Memory[0x4AA7] != 0xFF; pass++ {
		sokalPasses++
		if application.gameOver {
			t.Fatalf("the party was destroyed on the way to Sokal Keep: %q at %+v", application.eventText, application.spawn)
		}
		// 受傷或催眠用完就地紮營（要塞裡也一樣），再繼續探索。
		restUntilHealed()
		_, reachable = exploreWorldWithFlags(t, zipPath, int64(136+pass), pass%4, 8, 300000,
			map[[3]int]bool{}, map[[3]int]bool{}, transitionUses, menuTurn, exitUses,
			visited, maps, blocks, flags, noBoatOverride, &failures, nil,
			application, nil, func(a *app) bool {
				if a.tactical != nil && a.tactical != lastBattle {
					lastBattle = a.tactical
					names := []string{}
					for _, monster := range a.combatMonsters {
						names = append(names, fmt.Sprintf("%s×%d", monster.Record.Name, monster.Spawn.Count))
					}
					t.Logf("battle at %+v: %v", a.spawn, names)
				}
				return a.eventMachine != nil &&
					(a.eventMachine.Memory[0x4AA7] == 0xFF || readyToHandIn(a) || hurt(a))
			}, false)
		if !reachable {
			t.Fatal("the loaded natural campaign could not continue")
		}
		if readyToHandIn(application) {
			handInPending()
		}
		memory := application.eventMachine.Memory
		t.Logf("Sokal pass %d: 4AA7=%02X 4A01=%02X 4A00=%02X 4A77=%02X 4A78=%d at %+v ECL%d/%d",
			pass+1, memory[0x4AA7], memory[0x4A01], memory[0x4A00], memory[0x4A77], memory[0x4A78],
			application.spawn, application.eclArchive, application.eventSession.CurrentBlockID())
		if memory[0x4A78] >= 3 || memory[0x4A77]&64 != 0 {
			t.Fatalf("the Sokal exploration spoiled the Stojanow gate state: 4A77=%02X 4A78=%d",
				memory[0x4A77], memory[0x4A78])
		}
	}
	if got := application.eventMachine.Memory[0x4AA7]; got != 0xFF {
		t.Fatalf("Sokal Keep was not handed in after %d passes: 4AA7=%02X 4A01=%02X 4A26=%02X at %+v ECL%d/%d maps=%v failures=%v",
			sokalPasses, got, application.eventMachine.Memory[0x4A01],
			application.eventMachine.Memory[0x4A26], application.spawn, application.eclArchive,
			application.eventSession.CurrentBlockID(), sortedMapNames(maps), failures)
	}
	if len(failures) != 0 {
		t.Fatalf("continuous natural campaign recorded hard failures: %v", failures)
	}
	t.Logf("Sokal Keep handed in after %d explorer passes; maps so far %v", sokalPasses, sortedMapNames(maps))

	// 第 6～11 段（spec 137）：東航線 → 野外 → 波多廣場北緣 → 斯托亞諾夫城門 →
	// 城堡內部繞四張圖上樓 → 覲見廳 → 結局。每一段有界，撞 guard 就印旗標。
	application.keys = text
	driver := &mainlineDriver{t: t, a: application, pilot: &tacticalPilot{},
		step: func(key ebiten.Key) { step(key) }}
	driver.sailEast()
	driver.crossToPodol()
	driver.podolNorthEdge()
	driver.stojanowGate()
	driver.castleUpstairs()
	sawTyranthraxus, sawProgram8Ending, endingPages := driver.audienceHall()
	for _, name := range []string{"GEO1/18", "GEO2/9", "GEO5/3", "GEO5/6", "GEO5/5", "GEO5/4", "GEO5/7"} {
		maps[name] = true
	}
	for _, block := range []int{18, 9, 3, 5, 7} {
		blocks[block] = true
	}

	blockList := make([]int, 0, len(blocks))
	for block := range blocks {
		blockList = append(blockList, block)
	}
	sort.Ints(blockList)
	completed := []int{}
	pending := []int{}
	for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
		switch application.eventMachine.Memory[address] {
		case uint16(gamepack.CityHallSlotPending):
			pending = append(pending, int(address-0x4AA6))
		case 0xFF:
			completed = append(completed, int(address-0x4AA6))
		}
	}
	if got := application.eventMachine.Memory[0x4ABA]; got < 0xFE {
		t.Fatalf("natural campaign did not reach the ending: %s maps=%v blocks=%v completed=%v pending=%v failures=%v",
			driver.flags(), sortedMapNames(maps), blockList, completed, pending, failures)
	}
	// 主線必經的委任只有貧民窟（槽 21）與索寇要塞（槽 1）：結局旗標不讀
	// `4AC1`，其餘委任是隊伍強度的來源不是閘門（spec 137）。
	for _, slot := range []uint16{1, 21} {
		if got := application.eventMachine.Memory[0x4AA6+slot]; got != 0xFF {
			t.Fatalf("required City Hall commission slot %d ended at %02X, want FF", slot, got)
		}
	}
	if !sawTyranthraxus {
		t.Fatal("continuous natural campaign never entered the TYRANITHRAXUS battle")
	}
	if !sawProgram8Ending || endingPages != 3 {
		t.Fatalf("PROGRAM 8 ending receipt incomplete: saw=%t pages=%d", sawProgram8Ending, endingPages)
	}
	// 結局腳本最後 `NEWECL 0` 把隊伍送回文明區（spec 108）。
	if application.eclArchive != 3 || application.eventSession.CurrentBlockID() != 0 {
		t.Fatalf("after the ending the party is at ECL%d/%d, want ECL3/0",
			application.eclArchive, application.eventSession.CurrentBlockID())
	}
	t.Logf("continuous natural campaign maps=%v blocks=%v completed=%v pending=%v failures=%v",
		sortedMapNames(maps), blockList, completed, pending, failures)
	t.Logf("final 4ABA=%02X slot1=%02X slot21=%02X progress=%d location=%+v ecl=%d/%d party=%d pooled=%v",
		application.eventMachine.Memory[0x4ABA], application.eventMachine.Memory[0x4AA7],
		application.eventMachine.Memory[0x4ABB], application.eventMachine.Memory[0x4AC1],
		application.spawn, application.eclArchive, application.eventSession.CurrentBlockID(),
		len(application.state.Party), application.state.PooledMoney)
	t.Logf("route log: %s", strings.Join(driver.log, " | "))
}
