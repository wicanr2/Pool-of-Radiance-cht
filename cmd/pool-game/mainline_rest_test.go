package main

// 玩家策略層第一條（#22）：打完看 HP 決定要不要就地紮營，兩條主線探針與戰術的
// 最小重現共用。從 mainline_probe_test.go 搬出來，內容不變。

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 玩家策略層第一條：打完看 HP。有人掉到一半以下、或昏迷（狀態 4）就停下
// 探索，找地方紮營休息到滿——每二十四小時回一點（spec 114）。
func partyHurt(a *app) bool {
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
// **就地睡，不走去找床**：打完一場常常只剩兩個人站著，走去找床的路上再撞一場
// 就是全滅（實測第 14 場之後；#24 接上入口 2 之後再試一次，四場架就死了）。
//
// 入口 2 接上之後（#24）這一條在貧民窟街上會睡不滿：街上是 24／24（每兩小時
// 擲一次、24% 會被城衛隊趕起來），而回一點生命力要連續睡滿二十四小時
// （`RestTicksPerHeal` 288 刻），十二次檢定全過的機率只有四%。屋內才是 0／0。
// **那是原版的規則，不是缺口**——原版玩家清出一間屋子或去旅店睡（城區的
// `4A07 != 0` 才寫 0／0，hypothesis：那是旅店的房間）。駕駛要挑地方睡是
// 玩家策略層的事，歸 #22。
func (d *mainlineDriver) restUntilHealed() {
	t, application, step := d.t, d.a, d.step
	t.Helper()
	// 找床要走路，而走路每一步都會問「受傷了沒」——不擋就無限遞迴。
	if d.resting {
		return
	}
	d.resting = true
	defer func() { d.resting = false }()
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
			// 量牆與探索那幾條測試把全滅當成合法結果（`tolerateDefeat`）：
			// 睡到一半被打光就記一行交還，呼叫端自己收（CLAUDE.md §6）。
			if d.tolerateDefeat {
				d.note("休息中被打光：%q", application.eventText)
				return
			}
			d.fatalf("the party was destroyed while resting: %q", application.eventText)
		}
	}
	// 這一區會不會打擾？入口 2 只寫變數，跑它沒有副作用（#24）。會打擾就
	// **不要睡**：貧民窟街上被打斷不是選單而是一場隨機遭遇（`ecl2/20` 入口 3
	// `9A49h`：`SAVE 200 @4A1F`、`PARTYSTRENGTH`、`GOTO 9B68h` 排一場架），
	// 傷兵在那裡連睡八次就是全滅。原版要睡得進屋、或去旅店（城區 `4A07 != 0`
	// 才寫 0／0，hypothesis）。挑地方睡是玩家策略層的事，見 #22。
	origin := application.spawn.Map
	if err := application.runRestEntry(); err == nil && application.restInterruption().Period != 0 {
		// 睡不成就找地方：貧民窟的屋內（0／0，不用錢）或旅店（#38），挑近的。
		// 2026-09-17 量過 139 次：屋內平均 8.7 步、旅店單程 19.7 步，屋內近的佔 81%。
		// 旅店房間開好之後這一格是 0／0，下面那段排時間的流程照跑；睡完再走回原來
		// 那張圖，不然呼叫端的路線就斷了。
		if indoor, inn := d.measureRestOptions(); indoor >= 0 && (inn < 0 || indoor <= inn) && d.restIndoors() {
			// 屋內睡得成，就地往下排時間。
		} else if !d.restAtTheInn() {
			d.note("restUntilHealed: %+v interrupts rest (%d／%d) and the inn is out of reach",
				application.spawn, application.restInterruption().Period, application.restInterruption().Threshold)
			// 有人倒地又睡不成，就停在這裡報原因。以前記一行就往下走，貧民窟階段
			// 帶著倒地的隊伍收場，後面才報成「沒走進古托井」——報錯離成因很遠
			// （seed 137／142／144：白金在倒地的人身上，醒著的人付不出來）。只看倒地、不看「HP 不到一半」：
			// seed 143 在古托井地面睡不成時 B 是 3/7，照樣打贏諾里斯。
			if down := partyDown(application); down != "" && !d.tolerateDefeat {
				d.fatalf("restUntilHealed: cannot rest at %+v with %s down: platinum=%d 4ABB=%02X party=%s",
					application.spawn, down, partyPlatinum(application),
					application.eventMachine.Memory[0x4ABB], partyHP(application))
			}
			return
		} else {
			defer d.walkBackFrom(origin)
		}
	}
	for attempt := 0; attempt < 8 && (partyHurt(application) || pendingMemorisation(application)); attempt++ {
		settle()
		// 打過架用掉的法術格先補記，這一次休息順便記完。
		d.memoriseSpells()
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
			d.fatalf("E did not open the camp at %+v: %q (mode=%d help=%t tactical=%t event=%t menu=%t door=%t shop=%t program=%t text=%q)",
				application.spawn, application.statusLine, application.mode, application.help,
				application.tactical != nil, application.cellEventPending, application.cellWaitingMenu,
				application.door != nil, application.shopActive, application.campFromProgram, application.eventText)
		}
		step(ebiten.KeyR)
		d.clearRestDuration()
		step(ebiten.KeyY)
		for day := 0; day < days; day++ {
			step(ebiten.KeyI)
		}
		// 睡到隔天早上。時鐘接上 ECL 之後（#20）城區過了十四點市政廳與幾道門
		// 會鎖上（`ecl3/0 9920h`，spec 102），而天數不會改變醒來的時刻，所以要
		// 另外補小時。
		// `H` 是切到小時那一欄（`camp.go` 的紮營鍵表）。
		if hours := hoursUntilMorning(application.gameTime); hours != 0 {
			step(ebiten.KeyH)
			for hour := 0; hour < hours; hour++ {
				step(ebiten.KeyI)
			}
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
	if partyHurt(application) || pendingMemorisation(application) {
		// 量牆的那幾條（`tolerateDefeat`）要的是「這一場打得贏嗎」，睡不滿是
		// 原版在街上的常態（見檔頭），記一行就往下走；主線探針還是要當錯。
		if d.tolerateDefeat {
			d.note("rested but still hurt at %+v: %s", application.spawn, application.statusLine)
			return
		}
		d.fatalf("still hurt or unmemorised after resting at %+v (status %q)", application.spawn, application.statusLine)
	}
}

// measureRestOptions 量「屋內最近幾步、旅店幾步」（goal issue-22-26-levels-before-sokal 第 3 步，
// 先量分布再決定走法）。只記一行，不改行為。貧民窟入口 2 `ecl2/20 9A24h` 只比
// `@6E82 == 0`，所以地形碼不為 0 的格子都睡得成；排除 3（潛在委託人，答 LEAVE 會把
// 隊伍搬到門口）與 9／13／15（20 隻以上的固定戰鬥）。
func (d *mainlineDriver) measureRestOptions() (indoor, inn int) {
	a := d.a
	if a.spawn.Map != slumsMap {
		return -1, -1
	}
	edge := func(x, y int) bool {
		return x == 15 && a.initialMap.Grid.CanMoveDungeonWrapped(x, y, 2)
	}
	steps := func(wanted func(x, y int) bool) int {
		if wanted(int(a.spawn.X), int(a.spawn.Y)) {
			return 0
		}
		if plan := planAllowing(a, wanted, d.slumsSafe, true); len(plan) != 0 {
			return len(plan)
		}
		return -1
	}
	indoor = steps(d.slumsIndoor)
	if inn = steps(edge); inn >= 0 {
		inn += innStepsFromTheGate
	}
	d.note("restOptions: at (%d,%d) indoor=%d inn=%d down=%q platinum=%d",
		a.spawn.X, a.spawn.Y, indoor, inn, partyDown(a), partyPlatinum(a))
	return indoor, inn
}

// slumsMap 是貧民窟（ECL2/20、GEO2/20）。
var slumsMap = gamepack.MapKey{Archive: 2, BlockID: 20}

// innStepsFromTheGate 是城門 (0,4) 走街道到旅店的步數（spec 114）。
const innStepsFromTheGate = 12

// slumsIndoor 是貧民窟裡睡得成、走進去也不會開打的格子。
//
// 睡得成：入口 2 `ecl2/20 9A24h` 只比 `@6E82 == 0`（地形碼 0 是街上），地形碼不為 0
// 的都是 0／0。**但那些格子本身是入口 1 的事件**（`99C9h ON GOTO` 以地形碼分派），
// 傷兵走進一間還沒打過的房間就是一場架——2026-09-17 第一版只看步數、最近的是地形碼 1
// 的獸人房（(13,1) 那一帶），15 個 seed 裡 10 個死在去屋內休息的那一段。所以只收兩種
// （`99C9h` 21 支從入口沿控制流走到底，含 GOSUB 與 IF 的跳過）：
//
//   - 整支沒有 `COMBAT`：4（髒房間）、7、11（儲藏室）、17（只印一句）；
//   - 只有一場、開頭 `COMPARE flag, 255 ; IF = ; EXIT`，而那個旗標已經是 255。
//
// 5／10／16 的判斷不是單純的 255，0／3／8／18／19 接到共用的隨機遭遇，都不收；
// 9／13／15 本來就是 `slumsSafe` 不走的大場。
func (d *mainlineDriver) slumsIndoor(x, y int) bool {
	code := d.terrainCode(x, y)
	switch code {
	case 4, 7, 11, 17:
		return true
	}
	flag, ok := slumsFightRoomDone[code]
	return ok && d.a.eventMachine.Memory[flag] == 255
}

// slumsFightRoomDone 是只有一場架的房間與它「打過了」的旗標（`ecl2/20` 入口 1 各支開頭）。
var slumsFightRoomDone = map[int]uint16{
	1: 0x4ACA, 2: 0x4ACB, 6: 0x4A85, 12: 0x4AD6, 14: 0x4AD0, 20: 0x4AD9,
}

// restIndoors 走進貧民窟最近的屋內格，事件按完之後再跑一次入口 2；回 true 表示
// 這一格是 0／0、可以就地排時間。走不到、被事件搬走或打輸都回 false，讓呼叫端
// 改走旅店。
func (d *mainlineDriver) restIndoors() bool {
	a := d.a
	if a.spawn.Map != slumsMap {
		return false
	}
	if !d.slumsIndoor(int(a.spawn.X), int(a.spawn.Y)) &&
		!d.walkAllowing("indoors to rest", d.slumsIndoor, d.slumsSafe, true) {
		return false
	}
	d.settle()
	if a.gameOver || a.spawn.Map != slumsMap || !d.slumsIndoor(int(a.spawn.X), int(a.spawn.Y)) {
		return false
	}
	if err := a.runRestEntry(); err != nil || a.restInterruption().Period != 0 {
		d.note("restIndoors: (%d,%d) terrain %d still interrupts (%d／%d)", a.spawn.X, a.spawn.Y,
			d.terrainCode(int(a.spawn.X), int(a.spawn.Y)), a.restInterruption().Period, a.restInterruption().Threshold)
		return false
	}
	d.note("restIndoors: (%d,%d) terrain %d", a.spawn.X, a.spawn.Y, d.terrainCode(int(a.spawn.X), int(a.spawn.Y)))
	return true
}

// partyDown 列出倒地的隊員（昏迷、瀕死，或狀態正常但 HP 歸零），沒有就回空字串。
// 死亡不算：那要神殿，不是休息。
func partyDown(a *app) string {
	names := []string{}
	for _, member := range a.state.Party {
		if member.Status == 4 || member.Status == 5 || (member.Status == 0 && member.CurrentHP <= 0) {
			names = append(names, strings.TrimSpace(member.Name))
		}
	}
	return strings.Join(names, ",")
}

// partyPlatinum 是全隊身上的白金總數（旅店一晚一枚）。
func partyPlatinum(a *app) int {
	total := 0
	for _, member := range a.state.Party {
		total += int(member.Money[pooltreasure.Platinum])
	}
	return total
}

// partyHP 是六人的 HP 與狀態，報錯用。
func partyHP(a *app) string {
	parts := []string{}
	for _, member := range a.state.Party {
		parts = append(parts, fmt.Sprintf("%s %d/%d st%d", strings.TrimSpace(member.Name),
			member.CurrentHP, member.MaxHP, member.Status))
	}
	return strings.Join(parts, " ")
}

// innTerrain 是城區地形索引 9——旅店那七格（spec 102；GEO3/0 的 `89h`：
// (4,12) (6,12) (4,13) (6,13) (0,14) (1,14) (2,14)）。
const innTerrain = 9

// restAtTheInn 去城區的旅店開一晚房間（#38，spec 114）。原版一枚白金一晚：
// 踏上那七格之一 → `ecl3/0 A140h` 問 "IT WILL COST YOU 1 PLATINUM PIECE TO REST
// HERE. DO YOU WANT TO STAY?" → YES → `WHO 'WHO WILL PAY?'` 扣 `6BC3` 一枚 →
// `SAVE 1 @4A07` → `GOSUB 9A63h`（城區入口 2 的本體，這時寫 0／0）→ `PROGRAM 9`
// 開紮營。
//
// 回 true 表示房間開好了（紮營畫面也開著）。開不成一律回 false 讓呼叫端維持
// 「不睡」——**不硬走**：傷兵從貧民窟走回城區的路上會再撞遭遇，硬走會死
//（#24 那一輪實測，四場架就全滅）。
func (d *mainlineDriver) restAtTheInn() bool {
	a := d.a
	if a.eventMachine == nil || a.eventSession == nil || a.initialMap == nil {
		return false
	}
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		if !d.walkBackToPhlan() {
			return false
		}
	}
	if a.eventMachine.Memory[0x4A07] != 0 {
		return true // 已經有房間（踏回街上 `AE6Ah` 就會清掉）
	}
	payer := -1
	for index, member := range a.state.Party {
		if member.Status == 0 && member.Money[pooltreasure.Platinum] >= 1 {
			payer = index
			break
		}
	}
	if payer < 0 {
		// 只讓醒著的人付：原版 `WHO` 的挑人範圍（overlay-25 entry 42 `2C81h`）含不含
		// 倒下的人沒讀（spec 090 OPEN），腳本 `A1B4h` 自己只擋 NPC 與白金不夠。
		// 醒著的人付，兩種讀法都合法。
		d.note("restAtTheInn: nobody conscious has a platinum piece (party holds %d)", partyPlatinum(a))
		return false
	}
	inn := func(x, y int) bool { return d.terrain(x, y) == innTerrain }
	street := func(x, y int) bool { return d.terrain(x, y) == 0 }
	if inn(int(a.spawn.X), int(a.spawn.Y)) {
		// 已經站在旅店那一格上：退一步再踏進來才會重跑那一支。
		facing, ok := d.approachIfPossible("off the inn", street, street)
		if !ok {
			return false
		}
		d.face(facing)
		d.step(ebiten.KeyArrowUp)
	}
	facing, ok := d.approachIfPossible("the inn", inn, street)
	if !ok {
		d.note("restAtTheInn: cannot reach the inn from (%d,%d)", a.spawn.X, a.spawn.Y)
		return false
	}
	d.face(facing)
	d.step(ebiten.KeyArrowUp)
	// 問句 → YES → 誰付錢。WHO 的選單是隊員名字（spec 090）。
	for guard := 0; guard < 64 && !a.campOpen; guard++ {
		switch {
		case a.whoPending && len(a.cellMenuOptions) != 0:
			for turn := 0; a.cellMenuCursor != payer && turn < 8; turn++ {
				d.step(ebiten.KeyArrowDown)
			}
			d.step(ebiten.KeyEnter)
		case a.cellWaitingMenu && len(a.cellMenuOptions) != 0:
			pick := 0
			for index, option := range a.cellMenuOptions {
				if strings.EqualFold(option, "YES") {
					pick = index
				}
			}
			for turn := 0; a.cellMenuCursor != pick && turn < 8; turn++ {
				d.step(ebiten.KeyArrowDown)
			}
			d.step(ebiten.KeyEnter)
		case a.cellEventPending:
			d.step(ebiten.KeyEnter)
		default:
			d.step(ebiten.KeyEnter)
		}
	}
	if !a.campOpen {
		d.note("restAtTheInn: the inn did not open the camp: 4A07=%d text=%q",
			a.eventMachine.Memory[0x4A07], strings.TrimSpace(a.eventText))
		return false
	}
	d.note("restAtTheInn: room taken at (%d,%d); %s paid, 4A07=%d, interruption %d／%d",
		a.spawn.X, a.spawn.Y, strings.TrimSpace(a.state.Party[payer].Name),
		a.eventMachine.Memory[0x4A07], a.restInterruption().Period, a.restInterruption().Threshold)
	// `PROGRAM 9` 已經把紮營畫面打開了，但呼叫端接下來要先記法術（那要在地圖
	// 畫面）再自己按 E。收掉它——**房間不會因此退掉**：清 `4A07` 的是街道那一支
	// （`AE6Ah`），而我們還站在旅店那一格上。
	for guard := 0; guard < 16 && (a.campOpen || a.campFromProgram); guard++ {
		d.step(ebiten.KeyEscape)
	}
	d.settle()
	if a.campOpen {
		d.note("restAtTheInn: could not close the camp screen")
		return false
	}
	if got := a.eventMachine.Memory[0x4A07]; got != 1 {
		d.note("restAtTheInn: the room was given up while closing the camp (4A07=%d)", got)
		return false
	}
	return true
}

// walkBackToPhlan 從貧民窟走回城區（(15,4) 往東出界）。旅店在城區，而傷兵
// 走回去的路上會再撞遭遇——**走不回去就回 false**，呼叫端維持「不睡」。
// 只接貧民窟：別的區要穿過更多張圖，代價還沒量過（#38 的停止線）。
func (d *mainlineDriver) walkBackToPhlan() bool {
	a := d.a
	slums := gamepack.MapKey{Archive: 2, BlockID: 20}
	if a.spawn.Map != slums {
		return false
	}
	edge := func(x, y int) bool {
		return x == 15 && a.initialMap.Grid.CanMoveDungeonWrapped(x, y, 2)
	}
	if !edge(int(a.spawn.X), int(a.spawn.Y)) && !d.walkAllowing("back to Phlan", edge, d.slumsSafe, true) {
		return false
	}
	if a.gameOver || a.spawn.Map != slums {
		return false
	}
	d.face(1)
	d.step(ebiten.KeyArrowUp)
	d.settle()
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		d.note("walkBackToPhlan: east edge led to ECL%d/%d, not the city",
			a.eclArchive, a.eventSession.CurrentBlockID())
		return false
	}
	d.note("walkBackToPhlan: back in Phlan at (%d,%d)", a.spawn.X, a.spawn.Y)
	return true
}

// walkBackFrom 睡完把隊伍送回 origin 那張圖——去旅店是繞路，繞完要回到原處，
// 否則呼叫端（探針的貧民窟那一段）會發現自己站在別張圖上。只接貧民窟：
// 城區 (0,4) 往西出界。走不回去就記一行，讓呼叫端自己的檢查去報。
func (d *mainlineDriver) walkBackFrom(origin gamepack.MapKey) {
	a := d.a
	if a.gameOver || a.spawn.Map == origin {
		return
	}
	if origin != (gamepack.MapKey{Archive: 2, BlockID: 20}) {
		d.note("walkBackFrom: no route back to %+v", origin)
		return
	}
	street := func(x, y int) bool { return d.terrain(x, y) == 0 }
	gate := func(x, y int) bool { return x == 0 && y == 4 }
	if !gate(int(a.spawn.X), int(a.spawn.Y)) && !d.walkAllowing("the west gate", gate, street, false) {
		d.note("walkBackFrom: cannot reach the west gate")
		return
	}
	d.face(3)
	d.step(ebiten.KeyArrowUp)
	d.settle()
	if a.spawn.Map != origin {
		d.note("walkBackFrom: west of the gate is %+v, not %+v", a.spawn.Map, origin)
		return
	}
	d.note("walkBackFrom: back in the slums at (%d,%d)", a.spawn.X, a.spawn.Y)
}

// restMorningHour 是「睡到早上」的目標時刻。腳本把 `49C9 >= 14` 當晚上：城區
// 入口 0 `ecl3/0 9920h` 晚上鎖市政廳與幾道門，斯托亞諾夫城門的馬車商人
// `ecl2/9 ADAAh` 晚上不出現（spec 102、137）。醒在六點，離十四點還有八小時。
const restMorningHour = 6

// sleepUntilMorning 就地紮營睡到早上 `restMorningHour`，按鍵與 `restUntilHealed` 同一套
// （E 開營地、R 休息、清掉時長、H 切到小時那一欄、I 加時、R 開始）。休息可能被遭遇
// 打斷，所以跑到時刻落在白天為止，guard 給寬。回傳醒來時是不是白天（`49C9 < 14`）。
func (d *mainlineDriver) sleepUntilMorning(why string) bool {
	d.t.Helper()
	a := d.a
	for attempt := 0; attempt < 8 && d.scriptNight(); attempt++ {
		d.settle()
		before := a.gameTime
		d.step(ebiten.KeyE)
		if !a.campOpen {
			d.fatalf("sleepUntilMorning (%s): E did not open the camp at %+v: %q", why, a.spawn, a.statusLine)
		}
		d.step(ebiten.KeyR)
		d.clearRestDuration()
		d.step(ebiten.KeyY)
		if hours := hoursUntilMorning(a.gameTime); hours != 0 {
			d.step(ebiten.KeyH)
			for hour := 0; hour < hours; hour++ {
				d.step(ebiten.KeyI)
			}
		}
		d.step(ebiten.KeyR)
		for guard := 0; guard < 8 && (a.campOpen || a.campFromProgram); guard++ {
			d.step(ebiten.KeyEscape)
		}
		d.settle()
		d.note("sleepUntilMorning (%s): clock %v → %v at %+v", why, before, a.gameTime, a.spawn)
	}
	return !d.scriptNight()
}

// scriptNight 是腳本眼中的晚上：`49C9 >= 14`（`ecl3/0 9920h`、`ecl2/9 ADAAh`）。
func (d *mainlineDriver) scriptNight() bool {
	return d.a.gameTime[gamepack.TimeDigitHour] >= 14
}

// hoursUntilMorning 是從現在睡到隔天早上要幾個小時。已經是早上就回 0。
func hoursUntilMorning(clock gamepack.GameTime) int {
	hour := clock[gamepack.TimeDigitHour]
	if hour == restMorningHour {
		return 0
	}
	return (restMorningHour - hour + 24) % 24
}
