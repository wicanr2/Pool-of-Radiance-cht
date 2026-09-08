package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 戰鬥畫面照原版的版面（spec 129）：外框下緣在 tile row 22、第 22 欄豎一條
// 把畫面切成左邊的戰場與右邊的資訊欄；戰場內部是 7×7 格，一格三個字元格。
// 換算到 640×400（原版的兩倍）就是下面這幾個數字。
const (
	// combatBoardLeft／Top 是戰場內部的左上角（native 8,8）。
	combatBoardLeft = 16
	combatBoardTop  = 16
	// combatBoardCell 是一格的邊長（native 24）。7 × 48 ＝ 336，
	// 正好鋪滿 x 16..351。
	combatBoardCell = 48
	// combatInfoLeft 是右側資訊欄的左界（native 184）。
	combatInfoLeft = 368
	// 資訊欄三行的基線。原版三行的字佔 native y 8..14／24..30／40..46，
	// 兩倍之後是 16../48../80..；倚天字型的 ascent 是 14，所以基線加 14。
	combatInfoLine1 = 30
	combatInfoLine2 = 62
	combatInfoLine3 = 94
	// combatNoteLine 是 remake 自己加的那一行（原版沒有）：標明這一頁哪幾項
	// 還是暫定的。放在資訊欄最下面，不動原版那三行的位置。
	combatNoteLine = 242
	// 那一塊排得下幾行、一行幾個半形位。資訊欄寬 640−368−16 ＝ 256，
	// 一個半形位 8 像素。
	combatNoteColumns = 30
	combatNoteLines   = 5

)

// geoDetailForDirection 取出 GEO cell 在該方向的 detail 位元。
// engine 的 DetailDirections 依 0／2／4／6 的順序存放。
func geoDetailForDirection(cell geometry.Cell, direction uint8) uint8 {
	switch direction {
	case 0:
		return cell.DetailDirections[0]
	case 2:
		return cell.DetailDirections[1]
	case 4:
		return cell.DetailDirections[2]
	case 6:
		return cell.DetailDirections[3]
	}
	return 0
}

// geoWallProbe 重現 overlay-10 `0138h`：把 GEO 資料換成戰術層的 0／1／3。
// engine 的 Wall 用的就是 0／2／4／6 這組方向碼，與戰術層同一套，不必轉換。
//
// 原版不對座標取模：超出 16×16 一律當成牆，只有「與隊伍同一列、且方向是東或西」
// 例外，那一格回開放，好讓走廊可以延伸出盤面。界內則是「沒有牆 → 0；
// 有牆且該方向的 detail 位元為 0 → 1；有牆且 detail 非 0 → 3」。
func geoWallProbe(grid geometry.Grid, partyY int) combat.WallProbe {
	return func(direction uint8, x, y int) (uint8, error) {
		if x < 0 || x >= geometry.Width || y < 0 || y >= geometry.Height {
			if y == partyY && (direction == combat.WallDirectionEast || direction == combat.WallDirectionWest) {
				return combat.WallOpen, nil
			}
			return combat.WallBlocking, nil
		}
		wall, ok := grid.Wall(x, y, int(direction))
		if !ok || wall == 0 {
			return combat.WallOpen, nil
		}
		cell, _ := grid.Cell(x, y)
		if geoDetailForDirection(cell, direction) != 0 {
			return combat.WallAlternate, nil
		}
		return combat.WallBlocking, nil
	}
}

// drawTactical 畫戰鬥畫面。版面照原版（spec 129）：左邊 7×7 格的戰場、
// 右邊三行資訊、框外一列指令。
//
// **這一頁同時是 `F5` 的預覽與實際戰鬥的畫面**——`combatActive` 時按 ENTER
// 進的是同一支。
//
// 原版沒有標題列，也沒有「格／阻擋／隊伍／敵方」那幾個數字：那些是 remake
// 自己的診斷欄位，現在收進資訊欄底下那一行，跟「哪幾項還是暫定的」一起講。
func drawTactical(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	a.drawCombatFrame(screen, foreground, accent)
	if a.initialMap == nil {
		drawText(screen, a.text(msgTacticalNoMap), combatBoardLeft, combatBoardTop+16, foreground)
		return
	}
	if a.tactical == nil {
		drawText(screen, a.text(msgTacticalNoState), combatBoardLeft, combatBoardTop+16, foreground)
		return
	}
	drawCombatBoard(screen, a, foreground)
	drawCombatInfo(screen, a, foreground, accent)

	// remake 自己加的兩行：鍵位與「還是暫定的那幾項」。原版沒有這兩行，
	// 但拿掉的話玩家看得到指令名卻不知道按什麼——remake 還沒有原版那套
	// 「先按 M 進移動模式」的子模式輸入。
	// 資訊欄只有 32 個半形位寬，這兩段都比它長，所以照寬度折行；
	// 不折的話字會直接畫出右框（第一次拍出來就是「結束回合」被切一半）。
	notes := append(wrapDisplay(a.text(msgTacticalKeys), combatNoteColumns),
		wrapDisplay(a.text(msgTacticalProvisional), combatNoteColumns)...)
	for index, line := range notes {
		if index >= combatNoteLines {
			break
		}
		drawText(screen, line, combatInfoLeft, combatNoteLine+index*18, foreground)
	}

	// 最下面那一列。挑目標的時候換成瞄準列——同一條基線，兩者不會同時出現。
	if !a.castTargeting || len(a.castTargets) == 0 {
		line := a.combatCommandBar()
		if a.tactical.Prompt {
			line = a.text(msgTacticalPrompt)
		}
		drawText(screen, line, 0, footerBaseline, accent)
	}
	drawCastMenu(screen, a, foreground, accent)
	drawCastTargeting(screen, a, accent)
}

// drawCastTargeting 標出選目標那一步停在誰身上。原版的瞄準列是
// `Aim: Next Prev Manual Center Exit`（spec 127）；Manual 的格子游標
// 已經接上，畫的是它停在哪一格。
func drawCastTargeting(screen *ebiten.Image, a *app, accent color.Color) {
	if !a.castTargeting || len(a.castTargets) == 0 {
		return
	}
	view := a.tactical.Viewport
	if a.castManual {
		drawText(screen, fmt.Sprintf(a.text(msgCastAimManual),
			a.castPending.Label, a.castManualX, a.castManualY, view.X, view.Y),
			0, footerBaseline, accent)
		return
	}
	target := a.castTargets[a.castTargetCursor]
	drawText(screen, fmt.Sprintf(a.text(msgCastAiming), a.castPending.Label, target, view.X, view.Y),
		0, footerBaseline, accent)
}

// drawCastMenu 把施法清單畫在盤面右邊。只列得出已經讀過處理常式的法術，
// 所以看得到的就是做得到的。
func drawCastMenu(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	if !a.castOpen || len(a.castOptions) == 0 {
		return
	}
	for index, option := range a.castOptions {
		cursor, ink := " ", foreground
		if index == a.castCursor {
			cursor, ink = ">", accent
		}
		drawText(screen, fmt.Sprintf("%s%s", cursor, option.Label),
			combatInfoLeft, combatInfoLine3+32+index*18, ink)
	}
}

// 這一段的部署是暫定的。原版由 DS:43A2h 的陣型樣板決定誰站哪一格，而那張表
// 是執行期填的、不在檔案裡（spec 061），填寫者還沒解出來。在解出來之前，
// 這裡只從部署投影的格子裡挑得進去的來放，並在畫面上標明它是暫定版面。
const (
	provisionalPartyOffsetX = -1
	provisionalFoeOffsetX   = 2
)

// deploymentCandidates 依部署投影列出某個地城格偏移底下可以站人的格子，
// 順序與原版走樣板的順序相同：外層 row、內層 col。
func deploymentCandidates(grid combat.TacticalGrid, classes combat.CellClasses, offsetX int, taken map[[2]int]bool) [][2]int {
	candidates := make([][2]int, 0, combat.DeploymentTemplateSize)
	for row := 0; row < combat.DeploymentTemplateRows; row++ {
		for column := 0; column < combat.DeploymentTemplateCols; column++ {
			x, y := combat.DeploymentCell(offsetX, 0, row, column)
			if x < 0 || x > combat.TacticalMaxX || y < 0 || y > combat.TacticalMaxY {
				continue
			}
			if taken[[2]int{x, y}] {
				continue
			}
			code := grid.Terrain[y*combat.TacticalRowStride+x]
			if code == combat.UnpaintedCellClass {
				continue
			}
			decision, err := combat.TryPlaceCombatant(0xFF, 0, code, classes)
			if err != nil || decision != combat.PlacementAccepted {
				continue
			}
			candidates = append(candidates, [2]int{x, y})
		}
	}
	return candidates
}

// provisionalRoster 把隊伍與已 staged 的怪物擺上戰場，回傳 1-based 的位置表、
// 各筆屬於哪一方，以及每一格對回哪一個隊伍成員（不是隊伍成員的是 −1）。
// 放不下的就不放——不擠、不重疊、不自行擴大範圍。
//
// **陣營是逐人看的**：`36h ADD NPC` 加進來的 NPC 記錄 `+10Eh` 非零時站在
// 對面（spec 091），所以不能整批當成我方；隊伍索引也因此要另外記，
// 不能靠「友方槽依序對應隊伍」那個假設。
func provisionalRoster(a *app, grid combat.TacticalGrid, classes combat.CellClasses) ([]combat.CombatantCell, []bool, []int) {
	cells := []combat.CombatantCell{{}}
	friendly := []bool{false}
	partySlot := []int{-1}
	taken := map[[2]int]bool{}

	// reach 非 nil 時只收「與隊伍走得通」的格子，見 assignOpposing 的說明。
	assign := func(members []int, offsetX int, isParty bool, reach map[int]int) int {
		next := 0
		for _, spot := range deploymentCandidates(grid, classes, offsetX, taken) {
			if next >= len(members) {
				return next
			}
			if reach != nil {
				if _, ok := reach[tacticalCellKey(uint8(spot[0]), uint8(spot[1]))]; !ok {
					continue
				}
			}
			taken[[2]int{spot[0], spot[1]}] = true
			cells = append(cells, combat.CombatantCell{
				X: uint8(spot[0]), Y: uint8(spot[1]), FootprintClass: 1,
			})
			friendly = append(friendly, isParty)
			partySlot = append(partySlot, members[next])
			next++
		}
		return next
	}

	allies, traitors := make([]int, 0, len(a.state.Party)), make([]int, 0, 1)
	for index, member := range a.state.Party {
		if member.Side != 0 {
			traitors = append(traitors, index)
			continue
		}
		allies = append(allies, index)
	}
	assign(allies, provisionalPartyOffsetX, true, nil)
	foes := 0
	for _, monster := range a.combatMonsters {
		foes += int(monster.Spawn.Count)
	}
	opposing := append([]int(nil), traitors...)
	for index := 0; index < foes; index++ {
		opposing = append(opposing, -1)
	}
	assignOpposing(grid, classes, cells, assign, opposing)
	return cells, friendly, partySlot
}

// assignOpposing 把敵方擺上去，而且**只擺在與隊伍走得通的格子**。
//
// 先照原本的偏移（`provisionalFoeOffsetX`）試，一個都擺不下才往別的偏移找，
// 全部都不通才退回原本的偏移不設限地擺——寧可擺得下也不要整場沒有敵人。
//
// 為什麼要這一層：實測 GEO4 block 21 那一場，雙方各自被地形圍在兩塊不相連
// 的區域裡（隊伍站 x=17..22、敵方站 x=36..38，中間走不過去）。誰都走不到
// 誰、誰都打不到誰，回合數一路加到兩百多還在跑——**那一場永遠結束不了**，
// 從外面看就像遊戲卡住。
//
// **這不是原版的部署演算法**：原版的部署還沒讀出來，整個 provisionalRoster
// 都是暫時的。這一步只是讓暫時的版本不會生出打不完的架。
func assignOpposing(grid combat.TacticalGrid, classes combat.CellClasses,
	placed []combat.CombatantCell,
	assign func(members []int, offsetX int, isParty bool, reach map[int]int) int,
	opposing []int) {
	if len(opposing) == 0 {
		return
	}
	if len(placed) < 2 {
		assign(opposing, provisionalFoeOffsetX, false, nil)
		return
	}
	anchor := placed[1]
	reach := tacticalStepDistances(grid, classes, anchor.X, anchor.Y)
	for _, offsetX := range []int{provisionalFoeOffsetX, 1, 3, 0, -2, 4} {
		if assign(opposing, offsetX, false, reach) != 0 {
			return
		}
	}
	assign(opposing, provisionalFoeOffsetX, false, nil)
}

// tacticalState 是戰術預覽跨影格保留的狀態。Scores 對應原版 runtime 的 `+3`
// 先攻排序分數，Budgets 對應 `+6` 的剩餘步數，兩者都是 1-based。
type tacticalState struct {
	Grid          combat.TacticalGrid
	Classes       combat.CellClasses
	Roster        []combat.CombatantCell
	Friendly      []bool
	// Footprint 是每一格原本的佔格類別。倒下時 `Roster` 的那一欄會歸零
	// （原版是 `+10Dh`），死靈術把屍體叫起來時要拿回來——不記著就只能猜
	// 一個值。
	Footprint []uint8
	// MaxHitPoints 是每一格的生命上限（原版記錄 `+32h`）。死靈術把屍體
	// 補到滿，補到哪裡由它決定。
	MaxHitPoints []int
	// AIDriven 對應原版記錄的 `+10Fh`：非零就由敵方 AI 分派這一格的行動。
	// 隊員是 0，怪物是 1；**被魅惑的隊員也會變成 1**（spec 112 的
	// overlay-12 entry 14 `048Fh`），所以「誰由 AI 走」不能拿陣營來判——
	// 倒戈之後陣營變了，控制權沒有跟著回到玩家手上。
	AIDriven []bool
	// PartySlot 把戰場上的位置換回隊伍索引，−1 代表那一格不是隊員。
	// 施法要用它才找得到「這個位置是誰」的記憶陣列。
	PartySlot []int
	// HitDice 是每一格的 `+73h`（最高職業等級，怪物就是生命骰）。
	// 催眠術用它算要花多少額度（spec 098）。
	HitDice []uint8
	// SleepFlag 是每一格的 `+2Eh`，催眠術第 5 段要看它。
	SleepFlag []uint8
	// Effects 是每一格身上的效果串列，對應原版角色記錄 `+7Fh` 的那條
	// 單向串列（spec 059／112）。新的接在尾端，順序就是掛上去的順序。
	//
	// 催眠（`35h`）、定身（`34h`）與魅惑（`0Bh`）都存在這裡，不再各留一個
	// 旗標——**解除魔法要走的就是這條串列**，兩份真相會讓它解不到東西。
	// 定身的剩餘回合數是節點的持續（`+1..+2`）。
	//
	// 被迷住之後仍**只讓它不再行動**：原版會讓它倒戈（spec 112 的
	// overlay-12 entry 14），那要等敵方 AI 那一側接上來。
	Effects []gamepack.EffectList
	// FoeTargets 是每一格上一回合追的目標（原版戰鬥子結構 `+0Ah` 的目標
	// 遠指標）。目標還有效就沿用，不是每回合重挑（spec 096）。
	FoeTargets []uint8
	// TacticModes 是每一格身上的戰術模式（原版戰鬥子結構的 `+15h`，值 1..6）。
	// 它決定敵方回合要往哪五個相對方向試，跨回合留著（spec 096）。
	TacticModes []uint8
	// CreatureType 是每一格的 `+9Fh`（生物種類）、BodySize 是 `+6Ch`（體型）。
	// 魅惑人類與定身術用它們判斷目標算不算「人」，迷蛇術用種類收目標。
	CreatureType []uint8
	BodySize     []uint8
	// SaveTargets 是每一格的五個豁免目標值（記錄 `+6Dh` 起，spec 075），
	// SaveBonus 是記錄 `+101h` 的修正。隊員的目標值由職業等級查表算出來。
	SaveTargets [][gamepack.SavingThrowCategories]uint8
	SaveBonus   []int
	Dexterity     []uint8
	Scores        []uint8
	Budgets       []uint8
	States        []uint8
	DyingCounters []uint8
	BaseMovement  []uint8
	HitPoints     []int
	THAC0         []uint8
	ArmorClass    []int
	Damage        []combat.DamageDice
	// AttackForms 是兩種攻擊形態的骰子，AttackRates 是各自的攻擊次數編碼
	//（每回合次數 × 2，spec 051）。`Damage` 留著給法術與其他不分形態的
	// 呼叫端用，等於第一種有骰子的那一形態。
	AttackForms [][gamepack.MonsterAttackSlots]combat.DamageDice
	AttackRates [][gamepack.MonsterAttackSlots]uint8
	// AttackPhase 是半回合相位（spec 051）：戰鬥開始是 0，每個回合邊界加一，
	// `AttacksThisPhase` 只看它的最低位。編碼 3（每兩回合三次）就靠它交替。
	AttackPhase uint8
	Round       int
	Mover         uint8
	Finished      bool
	Prompt        bool
	Outcome       combat.CombatOutcome
	BudgetSource  string
	Status        string
	FoeLog        string
	// Text 由建立者接上 app.text，讓狀態列的訊息也能翻譯。測試直接建構
	// tacticalState 時不設它，say 會退回英文，所以測試不必知道語言這件事。
	Text func(messageID) string
	// Clouds 是盤面上的雲團物件（spec 121）。臭雲術不是對目標下效果，
	// 是在盤上生一個活的物件；地形寫在 Grid.Terrain 裡，這條串列記著
	// 每一團的雲心、蓋過哪幾格與那幾格原本的地形。
	Clouds gamepack.CloudList
	// Viewport 是戰術地圖 record 的 `+2`／`+3`：7×7 視窗左上角對到哪一格
	// （spec 127）。原版一次只畫得下 7×7，瞄準時的 Manual 游標與 `Center`
	// 都在捲它；remake 目前整張盤面都畫得出來，所以它只影響狀態列顯示，
	// 捲動的規則本身照原版接在 combat.RecentreViewport 裡。
	//
	// **初值原版從哪裡來還沒讀**，這裡用零值。
	Viewport combat.ViewportOrigin
	// stallSignature／stalledRounds 是**非原版**的僵局安全閥，見 endRound。
	stallSignature string
	stalledRounds  int
}

// tacticalStalemateRounds 是「盤面完全沒變」幾回合之後判僵局。
//
// **非原版**，與同一個檔案裡的繞路備案同一個性質：原版怎麼收這種場面還沒讀。
// 實測有這樣一場——野外 25 的野豬 ×5，雙方誰也走不到誰，回合數一路衝到
// 兩萬兩千還在跑，從外面看就是遊戲不動了（WORKLIST 的「走得到的內容量」）。
// 五十回合完全沒有位移、沒有狀態變化、沒有人倒下，正常戰鬥不會發生。
const tacticalStalemateRounds = 50

// stallFingerprint 是「盤面有沒有變」的指紋：每一格的位置、生命值、狀態與
// 倒地計時。
//
// **生命值不能漏**：不還手的隊伍站在原地被打，位置與狀態都不會變，漏了生命值
// 就會被誤判成僵局——`TestPassiveCombatTerminates` 要的正是「這種隊伍會被
// 打死」。
func (state *tacticalState) stallFingerprint() string {
	// 測試會手工組 tacticalState，那些平行陣列不一定都填滿——指紋只是安全閥
	// 的輸入，取不到就當 0，不能因此炸掉。
	at := func(values []int, index int) int {
		if index < len(values) {
			return values[index]
		}
		return 0
	}
	atByte := func(values []uint8, index int) uint8 {
		if index < len(values) {
			return values[index]
		}
		return 0
	}
	var builder strings.Builder
	for index := 1; index < len(state.Roster); index++ {
		cell := state.Roster[index]
		fmt.Fprintf(&builder, "%d:%d,%d,%d,%d,%d,%d;", index, cell.X, cell.Y,
			cell.FootprintClass, at(state.HitPoints, index),
			atByte(state.States, index), atByte(state.DyingCounters, index))
	}
	return builder.String()
}

// say 取出一則狀態訊息的目前語言版本。
func (state *tacticalState) say(id messageID, args ...any) string {
	format := packMessage(id, "en")
	if state != nil && state.Text != nil {
		format = state.Text(id)
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// Budget 回傳目前行動者的剩餘步數。
func (state *tacticalState) Budget() uint8 {
	if state.Mover == 0 || int(state.Mover) >= len(state.Budgets) {
		return 0
	}
	return state.Budgets[state.Mover]
}

// startRound 重現 spec 062 的回合開頭：逐一初始化移動預算，再擲先攻。
// 原版每回合都重擲，不是整場排一次。
func (state *tacticalState) startRound(roll func(count, sides int) int) {
	state.Round++
	// 相位在戰鬥開始是 0（overlay-10 的 setup 清成 0），每個回合邊界加一
	// （overlay-08 `0879h`）。第一回合還沒過邊界，所以只有第二回合起才推進。
	if state.Round > 1 {
		state.AttackPhase = combat.AdvanceAttackPhase(state.AttackPhase)
	}
	// 有計時的效果每個回合邊界減一，歸零就摘掉。
	for index := 1; index < len(state.Effects); index++ {
		state.tickEffects(index)
	}
	for index := 1; index < len(state.Roster); index++ {
		state.Budgets[index] = combat.InitialMovementBudgetBeforeEffects(state.BaseMovement[index], false, 0)
		// 已經離場的（體型 0，對應原版記錄的 `+10Dh`）分數一律 0，而且**連骰
		// 都不擲**。原版是 overlay-13 entry 1 的 `0084h`：`+10Dh` 為 0 就直接
		// 跳到 `00FFh` 把 runtime `+3` 寫 0，中間那段擲骰整個跳過（spec 062）。
		//
		// 少了這一條會連鎖：先攻選取（overlay-08 `0124h`）只看 `+3`，沒有另一
		// 道存活檢查，死者於是又被選成行動者；體型 0 的 mover 在
		// `ProbeDestination` 裡取不到佔格偏移，整個略過出界與地形檢查，一路走
		// 出盤面，再由 `RequiredFacing` 對出界座標回錯誤——外觀就是戰鬥停住。
		if state.Roster[index].FootprintClass == 0 {
			state.Scores[index] = 0
			continue
		}
		modifier := combat.DexterityInitiativeModifier(state.Dexterity[index])
		score, err := combat.ResolveInitiativeScore(modifier, uint8(roll(1, 6)), false)
		if err != nil {
			score = 0
		}
		state.Scores[index] = score
	}
	state.selectActor(roll)
}

// selectActor 重現 spec 062 的「每次行動後重選」：分數為 0 的不參與，
// 全部為 0 時代表這一回合沒有人可以行動。
func (state *tacticalState) selectActor(roll func(count, sides int) int) {
	scores := make([]uint8, 0, len(state.Roster))
	ties := make([]uint8, 0, len(state.Roster))
	for index := 1; index < len(state.Roster); index++ {
		scores = append(scores, state.Scores[index])
		ties = append(ties, uint8(roll(1, 100)))
	}
	selected, ok, err := combat.SelectInitiativeActor(scores, ties)
	if err != nil || !ok {
		state.Mover = 0
		return
	}
	state.Mover = uint8(selected + 1)
}

// endTurn 把目前行動者的分數歸零（Delay 時改成 1，與原版的 D 命令一致），
// 再重選；選不到人就跑回合收尾並開始下一回合。
func (state *tacticalState) endTurn(roll func(count, sides int) int, delay bool) {
	if state.Mover == 0 {
		return
	}
	if delay {
		state.Scores[state.Mover] = combat.DelayInitiative()
		state.Status = state.say(msgStatusDelayed)
	} else {
		state.Scores[state.Mover] = 0
		state.Status = state.say(msgStatusTurnEnded)
	}
	state.selectActor(roll)
	if state.Mover == 0 {
		state.endRound(roll)
	}
}

// endRound 重現 overlay-08 `0868h` 的回合收尾（spec 062）：先推進倒地計時，
// 再判結束；「我方還在、敵方清光」那一支要多問一次要不要繼續，不是直接結束。
//
// 原版另有一個不問的條件（`DS:4955h` 非 0），但全遊戲只有 overlay-11 `03C1h`
// 的啟動設定寫它，寫的是 0——所以正常遊玩時一律會問，這裡照做（spec 062）。
func (state *tacticalState) endRound(roll func(count, sides int) int) {
	for index := 1; index < len(state.Roster); index++ {
		state.States[index], state.DyingCounters[index] =
			combat.AdvanceDyingCounter(state.States[index], state.DyingCounters[index])
	}
	counts := state.sideCounts()
	if counts.Party > 0 && counts.Foes == 0 {
		state.Prompt = true
		state.Status = state.say(msgStatusContinuePrompt)
		return
	}
	if combat.RoundEndsCombat(counts, 0, false) {
		state.Finished, state.Outcome = true, combat.ResolveCombatOutcome(counts)
		state.Status = state.say(msgStatusDefeat)
		if state.Outcome == combat.CombatVictory {
			state.Status = state.say(msgStatusVictory)
		}
		return
	}
	// **非原版的僵局安全閥**（使用者 2026-09-06 決定留著）。
	// 盤面連續 `tacticalStalemateRounds` 回合完全
	// 沒變（沒人移動、沒人受傷、沒人倒下）就收場——雙方都走不到對方的時候，
	// 兩邊都會正常結束回合，於是回合數無限增加而什麼都不會發生。
	if fingerprint := state.stallFingerprint(); fingerprint == state.stallSignature {
		state.stalledRounds++
		if state.stalledRounds >= tacticalStalemateRounds {
			state.Finished, state.Outcome = true, combat.CombatOngoing
			return
		}
	} else {
		state.stallSignature, state.stalledRounds = fingerprint, 0
	}
	state.startRound(roll)
	state.Status = state.say(msgStatusRound, state.Round)
}

// placeholderBaseMovement 是隊伍成員的暫定移動值。remake 的角色記錄目前沒有
// 這個欄位，原版是 285-byte record 的 +11Ch；在它接上來之前，這個數字只是
// 讓移動判定可以被實際走一次，畫面上會標明它的來源。
const placeholderBaseMovement = 12

// 隊伍成員的戰鬥數值暫定值。remake 的角色記錄目前只有 HP，沒有 AC、THAC0
// 與傷害骰；原版那三項在 285-byte record 的 +110h／+111h／+115h..+119h。
// THAC0 與 AC 這裡存的是原版的內部編碼（60 減去顯示值），與 ResolveHit 一致。
const (
	placeholderHitPoints          = 8
	placeholderInternalTHAC0      = 40
	placeholderInternalArmorClass = 50
)

// dexterityAbilityIndex 是能力值陣列裡的 DEX，順序為 STR／INT／WIS／DEX／CON／CHA。
// placeholderDexterity 給沒有能力值的一方用；怪物的 DEX 在 285-byte record 的
// `+13h`，還沒接上來。
const (
	dexterityAbilityIndex = 3
	placeholderDexterity  = 12
)

// enterTacticalPreview 生成戰場、擺人、決定移動預算。
func (a *app) enterTacticalPreview() error {
	if a.initialMap == nil {
		return fmt.Errorf("Pool dungeon map is not loaded")
	}
	grid, err := combat.GenerateIndoorTacticalGrid(int(a.spawn.X), int(a.spawn.Y),
		geoWallProbe(a.initialMap.Grid, int(a.spawn.Y)))
	if err != nil {
		return err
	}
	classes := gamepack.OriginalCombatCellClassTable()
	roster, friendly, partySlot := provisionalRoster(a, grid, classes)

	base, source := uint8(placeholderBaseMovement), a.text(msgBudgetPlaceholder)
	if len(a.combatMonsters) > 0 {
		base, source = a.combatMonsters[0].Record.Movement(), a.text(msgBudgetStagedMonster)
	}
	size := len(roster)
	state := &tacticalState{
		Grid:         grid,
		Classes:      classes,
		Roster:       roster,
		Friendly:     friendly,
		Dexterity:    make([]uint8, size),
		Scores:       make([]uint8, size),
		Budgets:      make([]uint8, size),
		BaseMovement: make([]uint8, size),
		BudgetSource: source,
		PartySlot:    partySlot,
		AIDriven:     aiDriven(partySlot),
		Text:         a.text,
	}
	state.States = make([]uint8, size)
	state.DyingCounters = make([]uint8, size)
	state.HitPoints = make([]int, size)
	state.THAC0 = make([]uint8, size)
	state.ArmorClass = make([]int, size)
	state.Damage = make([]combat.DamageDice, size)
	state.AttackForms = make([][gamepack.MonsterAttackSlots]combat.DamageDice, size)
	state.AttackRates = make([][gamepack.MonsterAttackSlots]uint8, size)
	state.HitDice = make([]uint8, size)
	state.SleepFlag = make([]uint8, size)
	state.Effects = make([]gamepack.EffectList, size)
	state.Footprint = make([]uint8, size)
	state.MaxHitPoints = make([]int, size)
	for index := range roster {
		state.Footprint[index] = roster[index].FootprintClass
	}
	state.TacticModes = make([]uint8, size)
	state.FoeTargets = make([]uint8, size)
	state.CreatureType = make([]uint8, size)
	state.BodySize = make([]uint8, size)
	state.SaveTargets = make([][gamepack.SavingThrowCategories]uint8, size)
	state.SaveBonus = make([]int, size)
	for index := range state.SaveTargets {
		for category := range state.SaveTargets[index] {
			state.SaveTargets[index][category] = gamepack.SavingThrowWorstTarget
		}
	}
	for index := 1; index < size; index++ {
		// 隊員沒有 285-byte 記錄可讀，用原版「人」那一組值：種類 0、體型 1。
		// 原版資料裡的人形記錄（MACE、4TH LVL FIGHTER…）全部是這一組。
		state.CreatureType[index], state.BodySize[index] = 0, 1
		state.BaseMovement[index] = base
		state.Dexterity[index] = placeholderDexterity
		state.HitPoints[index] = placeholderHitPoints
		state.MaxHitPoints[index] = placeholderHitPoints
		state.THAC0[index] = placeholderInternalTHAC0
		state.ArmorClass[index] = placeholderInternalArmorClass
		// 隊員的攻擊次數還沒讀出來（原版由職業等級表給），先用「一回合一次」。
		state.setSingleAttackForm(index, combat.DamageDice{Count: 1, Sides: 8})
		if party := partySlot[index]; party >= 0 && party < len(a.state.Party) {
			member := a.state.Party[party]
			// NPC 沒有走過建角，戰鬥數值直接讀它帶著的原版記錄。
			if member.NPC {
				if err := applyNPCCombatStats(state, index, member); err != nil {
					return err
				}
				continue
			}
			state.Dexterity[index] = uint8(member.Abilities[dexterityAbilityIndex])
			if member.CurrentHP > 0 {
				state.HitPoints[index] = member.CurrentHP
			}
			if member.MaxHP > 0 {
				state.MaxHitPoints[index] = member.MaxHP
			}
			thac0, armor, movement, err := partyCombatStats(member)
			if err != nil {
				return err
			}
			state.THAC0[index] = thac0
			// `+73h` 是最高職業等級（spec 072 的 overlay-23）。催眠術用它。
			levels := memberClassLevels(member)
			for _, level := range levels {
				if level > state.HitDice[index] {
					state.HitDice[index] = level
				}
			}
			if a.savingThrows != nil {
				targets, err := a.savingThrows.TargetsForLevels(levels)
				if err != nil {
					return err
				}
				state.SaveTargets[index] = targets
			}
			// 穿在身上的東西改 AC 與腳程（spec 079／080）：`partyCombatStats`
			// 回的是建角值，那是「脫光了」的角色。
			armor, movement, err = a.memberDefenceStats(member, armor, movement)
			if err != nil {
				return err
			}
			state.ArmorClass[index] = armor
			state.BaseMovement[index] = movement
			// 手上有裝備好的武器時，THAC0 與傷害改由武器決定（spec 065）。
			if weapon, ok := a.readiedWeapon(member); ok {
				stats, err := a.weaponCombatStats(weapon, member, thac0)
				if err != nil {
					return err
				}
				state.THAC0[index] = stats.Thac0Internal
				state.setSingleAttackForm(index, combat.DamageDice{
					Count: stats.DamageCount, Sides: stats.DamageSides, Bonus: stats.DamageBonus,
				})
			}
			continue
		}
		if record, ok := a.stagedRecordFor(index, friendly); ok {
			state.BaseMovement[index] = record.Movement()
			state.HitDice[index] = record.Raw[0x73]
			state.SleepFlag[index] = record.Raw[0x2e]
			state.CreatureType[index] = record.CreatureType()
			state.BodySize[index] = record.BodySize()
			// 怪物記錄與角色記錄同一份 285-byte 版面，豁免那五格在 `+6Dh`。
			targets, err := gamepack.SavingThrowTargets(record.Raw[:])
			if err != nil {
				return err
			}
			bonus, err := gamepack.SavingThrowBonus(record.Raw[:])
			if err != nil {
				return err
			}
			state.SaveTargets[index], state.SaveBonus[index] = targets, bonus
			state.HitPoints[index] = int(record.CurrentHitPoints())
			state.MaxHitPoints[index] = int(record.MaxHitPoints())
			state.THAC0[index] = uint8(60 - record.THAC0())
			state.ArmorClass[index] = 60 - record.ArmorClass()
			state.Damage[index] = combat.DamageDice{
				Count: record.DamageDiceCount(),
				Sides: record.DamageDieSides(),
				Bonus: record.DamageBonus(),
			}
			if err := applyMonsterAttackForms(state, index, record); err != nil {
				return err
			}
		}
	}
	state.startRound(a.rollDice)
	state.Status = state.say(msgStatusRound, state.Round)
	a.tactical = state
	// 視窗一開始就對到行動者身上。原版重畫視窗時中心一定跟著行動者
	// （overlay-32 `07D4h`，餘裕 0 一定捲到正中央）；不捲的話視窗停在
	// (0,0)，而部署好的隊伍在別的地方——盤面看起來是空的。
	a.recentreOn(int(state.Roster[state.Mover].X), int(state.Roster[state.Mover].Y),
		combat.ViewportCentreMargin)
	return nil
}

// tacticalSnapshot 把跨影格的狀態組成 combat 層要的那份 DS 快照。佔用格每次
// 重建，與原版每個回合開頭呼叫 overlay-32 entry 20 的做法一致（spec 061）。
func (state *tacticalState) tacticalSnapshot() (combat.TacticalState, error) {
	occupancy, err := combat.RebuildOccupancy(state.Roster)
	if err != nil {
		return combat.TacticalState{}, err
	}
	return combat.TacticalState{
		Map:       state.Grid,
		Occupancy: occupancy,
		Cells:     state.Roster,
		Classes:   state.Classes,
	}, nil
}

// sideOf 把 Friendly 對回原版 combatant record 的 +10Eh 陣營欄位：隊伍 0、敵方 1。
// 查不到的索引一律回 false，讓陣營篩選失敗即關閉。
func (state *tacticalState) sideOf(index uint8) (uint8, bool) {
	if index == 0 || int(index) >= len(state.Friendly) {
		return 0, false
	}
	if state.Friendly[index] {
		return 0, true
	}
	return 1, true
}

// foeSearchBudget 是敵方找目標時給直線追蹤的預算。原版怎麼挑目標還沒讀出來，
// 這個值只用來保證整個盤面都落在搜尋範圍內。
const foeSearchBudget = 128

// foeMaxStepsPerTurn 是一回合內允許的步數上限。預算本身每步遞減、迴圈一定會停，
// 這個上限只是防止未來改動把它變成不會停的迴圈。
const foeMaxStepsPerTurn = 32

// foeReachBudget 是「武器搆不搆得到」的成本預算。原版 overlay-09 entry 5
// （`0D4Bh`）先用武器射程問一次搆得到誰，搆得到就打、搆不到才走；射程取自
// 手上武器型別的 `+0Ch` 減一，近戰武器就是一格（spec 096）。這裡只做近戰，
// 長柄與投射武器的射程還沒接進來。
const foeReachBudget = 1

// foeTurn 讓敵方的行動者走完一回合，骨架照 overlay-09 entry 5（`0B3Ch`）與
// 它呼叫的移動子程式 `07E8h`（spec 096）：每一步先問武器搆不搆得到人，搆得到
// 就打，搆不到才走一步；走的方向由戰術模式的五個相對偏移依序試，第一個進得去
// 的就走。每一步過 ResolveDestination（spec 058），攻擊走 spec 050／051。
//
// 兩處仍是近似，畫面上還是標 PROVISIONAL AI：**挑哪個目標**（原版 `0D97h` 是
// 從搆得到的名單裡擲骰隨機挑，追擊的目標則由 overlay-13 `37B8h` 決定，那一支
// 還沒讀），以及五個方向全不通時的**繞路備案**（原版沒有，靠跨回合換模式脫困）。
func (a *app) foeTurn(state *tacticalState) error {
	mover := state.Mover
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return err
	}
	side, ok := state.sideOf(mover)
	if !ok {
		return fmt.Errorf("Pool mover %d has no side", mover)
	}
	// 原版 overlay-13 `37B8h`：**還有效的目標就沿用**，換人才重挑。有效的
	// 條件是「不是自己這一邊」、「還在場上（`+10Dh`）」，再過一次 `1087h`
	// 的可打判定（那一支還沒讀）。追不到人時才換一個。
	target, ok := state.foeTarget(mover)
	if !ok {
		targets, err := combat.OpposingNearbyAt(snapshot, mover,
			state.Roster[mover].X, state.Roster[mover].Y, foeSearchBudget, 1-side, state.sideOf)
		if err != nil {
			return err
		}
		if len(targets) != 0 {
			// 原版是從候選名單裡擲骰隨機挑（`38A6h` 的 `骰(1, n)`），挑到
			// 不能打的就把那一格劃掉重擲，最多二十次。
			target = targets[a.rollDice(1, len(targets))-1]
		} else if nearest, ok := state.nearestReachableOpposing(mover); ok {
			// 反應距離內沒人時，改追盤面上最近的敵人。
			//
			// **這不是原版的名單**：原版的候選由 overlay-25 `010Ah:00C0h`
			// 填進 `DS:6CD7h`，那一支還沒讀。`OpposingNearbyAt` 重現的是
			// overlay-25 entry 32 的「鄰接反應」搜尋（spec 059），拿它當
			// 名單本來就是借用。少了這個退路，站得遠的怪物會回報找不到目標
			// 然後原地結束回合——實測索寇要塞那一場，最後兩隻殭屍與隊伍隔著
			// 22 格互相不動，戰鬥永遠打不完。
			target = nearest
		}
		state.setFoeTarget(mover, target)
	}
	if target == 0 {
		state.FoeLog = state.say(msgFoeNoTarget, mover)
		state.endTurn(a.rollDice, false)
		return nil
	}

	// 目標在這一回合裡不會換，所以步數表只算一次；它只給備案用。
	goalCell := state.Roster[target]
	stepDistance := tacticalStepDistances(state.Grid, state.Classes, goalCell.X, goalCell.Y)

	// 原版在 entry 5 開場把兩個全域清掉：`439Eh` 是上一步走的方向（8 代表
	// 還沒走過），`439Fh` 是「卡住」的次數。兩者都只活在這一隻的這一次接近。
	lastDirection := uint8(combat.DirectionAny)
	stuck := 0
	mode := state.tacticMode(mover, a.rollDice)

	steps := 0
	for ; steps < foeMaxStepsPerTurn; steps++ {
		snapshot, err = state.tacticalSnapshot()
		if err != nil {
			return err
		}
		here := state.Roster[mover]
		goal := state.Roster[target]

		// 先問武器搆得到誰（原版 `0D4Bh`）。搆得到就打，這一隻的回合結束。
		reachable, err := combat.OpposingNearbyAt(snapshot, mover,
			here.X, here.Y, foeReachBudget, 1-side, state.sideOf)
		if err != nil {
			return err
		}
		if len(reachable) != 0 {
			if err := a.resolveTacticalAttack(state, reachable[0]); err != nil {
				return err
			}
			state.FoeLog = state.say(msgFoeAttacked, mover, steps, state.Status)
			state.endTurn(a.rollDice, false)
			return nil
		}

		// 基準方向是目標相對自己的方位（overlay-13 `261Bh`：自 0 起找第一個
		// 弧內成立的方向，spec 098）。原版追的是記錄裡存著的目標指標，這裡用
		// 這一回合選定的目標。
		base, err := combat.RequiredFacing(here.X, here.Y, goal.X, goal.Y, combat.DirectionAny)
		if err != nil {
			return err
		}

		// 五個相對偏移依序試，第一個進得去的就走（原版 `092Ah` 的迴圈）。
		//
		// 「而且要離目標更近」**不是原版的條件**：原版只問進不進得去，不比距離。
		// 少了它，貼著牆的怪物會挑到 ±2 那兩個往旁邊走的方向，下一步又走回來，
		// 一整回合原地打轉——實測過整場打不完。距離用的是繞得過去的實際步數
		// （`tacticalStepDistances`），所以繞牆本身仍然算前進。
		hereDistance := tacticalDistanceAt(stepDistance, here.X, here.Y, goal)
		direction, found := uint8(0), false
		for step := 1; step <= gamepack.TacticSteps && !found; step++ {
			candidate, err := gamepack.DefaultTacticOffsets.Direction(mode, step, base)
			if err != nil {
				return err
			}
			outcome, err := combat.ResolveDestination(snapshot, mover, candidate, state.Budget())
			if err != nil {
				return err
			}
			if outcome.Action != combat.MovementEnter {
				continue
			}
			x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, candidate)
			if err != nil {
				return err
			}
			if tacticalDistanceAt(stepDistance, x, y, goal) >= hereDistance {
				continue
			}
			direction, found = candidate, true
		}
		if !found {
			// 五個方向都不合用：原版換模式、記一次卡住，然後就結束這一隻的移動，
			// 靠下一回合換到的模式脫困。照抄會讓怪物在死路裡卡上好幾回合，
			// 所以這裡再多一個**非原版**的繞路備案：八個方向裡挑離目標最近的。
			mode = gamepack.NextTacticMode(mode)
			stuck++
			detour, ok, err := state.detourStep(snapshot, mover, stepDistance, here, goal)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			direction, found = detour, true
		}
		if reverse := (int(direction) + combat.DirectionCount/2) % combat.DirectionCount; int(lastDirection) == reverse {
			// 這一步剛好是上一步的反方向：原版一樣記一次卡住並換模式，但第一次
			// 仍然走（`0A30h` 的 `439Fh <= 1`），第二次起才放棄這一隻的移動。
			mode = gamepack.NextTacticMode(mode)
			stuck++
			if stuck > 1 {
				// 原版 `0A37h`：卡住第二次就把記錄裡存著的目標清掉，
				// 下一回合重挑一個。
				state.setFoeTarget(mover, 0)
				break
			}
		}

		x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, direction)
		if err != nil {
			return err
		}
		budget, err := combat.SpendMovementStep(state.Budget(), direction)
		if err != nil {
			return err
		}
		if budget == state.Budget() {
			break
		}
		state.Roster[mover].X, state.Roster[mover].Y = x, y
		state.Budgets[mover] = budget
		lastDirection = direction
	}
	state.setTacticMode(mover, mode)
	state.FoeLog = state.say(msgFoeClosed, mover, steps, target)
	state.endTurn(a.rollDice, false)
	return nil
}

// 建角寫下的三個基礎值，逐一取自 overlay-16（spec 063）：AC internal 32h
// （typed 10）、THAC0 internal 28h（typed 20）、基礎移動 0Ch。
const (
	creationArmorClassInternal = 0x32
	creationThac0Internal      = 0x28
	creationBaseMovement       = 0x0C
)

// firstCharacterLevel 是還沒訓練過的角色的等級。建角每個組成職業各寫下第 1 級
// （spec 072），存檔裡沒有 ClassLevels 就是這個狀態。
const firstCharacterLevel = 1

// partyCombatStats 依 spec 063 由職業算出隊伍成員的基礎戰鬥數值：THAC0 逐個
// component 查 DS:3C16h 的表取最好的一個，AC 與移動用建角寫下的基礎值。
//
// 這裡回的是「脫光了」的角色。裝備由 `enterCombatStaging` 在這之後套上：
// AC 與腳程走 `memberDefenceStats`（spec 079／080），武器走 `readiedWeapon`
// 與 `weaponCombatStats`（spec 065）。
func partyCombatStats(member poolsave.Character) (thac0Internal uint8, armorInternal int, movement uint8, err error) {
	levels, err := partyClassLevels(member)
	if err != nil {
		return 0, 0, 0, err
	}
	thac0Internal, err = gamepack.BaseThac0Internal(levels)
	if err != nil {
		return 0, 0, 0, err
	}
	return thac0Internal, creationArmorClassInternal, creationBaseMovement, nil
}

// memberDefenceStats 把角色身上的東西算進 AC 與移動力（spec 079／080）。
// 兩者共用同一條物品鏈，原版也是在同一支 overlay-25 `0C17h` 裡一起算的，
// 分開走會出現「AC 算了裝備、腳程沒算」這種只在特定隊伍才看得出來的偏差。
func (a *app) memberDefenceStats(member poolsave.Character, baseArmor int, baseMovement uint8) (int, uint8, error) {
	// 沒有任何物品也要走完：敏捷的 AC 調整與硬幣的重量都不看物品鏈，
	// 提早返回會讓空手的角色少掉敏捷那一項。
	items := make([][]byte, 0, len(member.Inventory))
	for _, item := range member.Inventory {
		items = append(items, item.Raw)
	}
	armour, err := gamepack.ArmourClassFor(baseArmor,
		member.Abilities[dexterityAbilityIndex], items, a.itemTypes)
	if err != nil {
		return 0, 0, fmt.Errorf("Pool character %q armour class: %w", member.Name, err)
	}
	carried, err := gamepack.CarriedWeight(items, member.Money)
	if err != nil {
		return 0, 0, fmt.Errorf("Pool character %q carried weight: %w", member.Name, err)
	}
	movement, err := gamepack.MovementRateFor(int(baseMovement), member.Abilities[0],
		member.ExceptionalStrength, carried, items, a.itemTypes)
	if err != nil {
		return 0, 0, fmt.Errorf("Pool character %q movement: %w", member.Name, err)
	}
	return armour.Internal, uint8(movement), nil
}

// applyNPCCombatStats 用 NPC 自己帶的 285-byte 記錄填戰鬥數值，做法與已
// staged 的怪物同一套（spec 091）。NPC 不走建角那一組欄位，硬套會得到
// 一個「一級戰士」，而那與原版差很多。
func applyNPCCombatStats(state *tacticalState, index int, member poolsave.Character) error {
	if len(member.Record) != poolsave.NPCRecordSize {
		return fmt.Errorf("Pool NPC %q has a %d-byte record", member.Name, len(member.Record))
	}
	var record gamepack.MonsterRecord
	copy(record.Raw[:], member.Record)
	record.Name = member.Name
	state.BaseMovement[index] = record.Movement()
	state.HitPoints[index] = int(record.CurrentHitPoints())
	if index < len(state.MaxHitPoints) {
		state.MaxHitPoints[index] = int(record.MaxHitPoints())
	}
	state.THAC0[index] = uint8(60 - record.THAC0())
	state.ArmorClass[index] = 60 - record.ArmorClass()
	state.Damage[index] = combat.DamageDice{
		Count: record.DamageDiceCount(),
		Sides: record.DamageDieSides(),
		Bonus: record.DamageBonus(),
	}
	return applyMonsterAttackForms(state, index, record)
}

// setSingleAttackForm 設定「一回合一次、只有一種形態」的攻擊，並把
// `Damage` 一起同步。玩家角色與武器覆寫走這一條——兩個欄位分開設會分岔，
// 而分岔的症狀是「畫面寫著長劍、打出來卻是拳頭」。
func (state *tacticalState) setSingleAttackForm(index int, dice combat.DamageDice) {
	state.Damage[index] = dice
	state.AttackForms[index] = [gamepack.MonsterAttackSlots]combat.DamageDice{dice}
	state.AttackRates[index] = [gamepack.MonsterAttackSlots]uint8{2, 0}
}

// applyMonsterAttackForms 把兩種攻擊形態與各自的攻擊次數搬進戰術狀態。
//
// 巨魔是這一條的樣本：`04 02` 加 `1d4+4`／`2d6`，也就是 AD&D 一版的
// 爪／爪／咬。只讀一種形態的話牠一回合只揮一次爪。
func applyMonsterAttackForms(state *tacticalState, index int, record gamepack.MonsterRecord) error {
	for slot := uint8(1); slot <= gamepack.MonsterAttackSlots; slot++ {
		rate, err := record.BaseAttackRate(slot)
		if err != nil {
			return err
		}
		damage, err := record.AttackDamage(slot)
		if err != nil {
			return err
		}
		state.AttackRates[index][slot-1] = rate
		state.AttackForms[index][slot-1] = combat.DamageDice{
			Count: damage.Count, Sides: damage.Sides, Bonus: damage.Bonus,
		}
	}
	return nil
}

// tacticalDistanceAt 查步數表；查不到（那一格與目標之間沒有通路）就退回
// 直線距離，讓行為不會比先前差。
func tacticalDistanceAt(distance map[int]int, x, y uint8, goal combat.CombatantCell) int {
	if step, ok := distance[tacticalCellKey(x, y)]; ok {
		return step
	}
	return chebyshev(x, y, goal.X, goal.Y) + len(distance)
}

// detourStep 是五個戰術方向全不通時的**非原版**備案：八個方向都問一次，挑一個
// 走得進去而且離目標更近的。原版沒有這一步，它靠的是跨回合換戰術模式脫困；照抄
// 會讓怪物在死路裡卡上好幾回合——實測過最後兩隻殭屍與隊伍隔著 22 格互不相動，
// 戰鬥永遠打不完。
func (state *tacticalState) detourStep(snapshot combat.TacticalState, mover uint8,
	stepDistance map[int]int, here, goal combat.CombatantCell) (uint8, bool, error) {
	best, bestDistance := uint8(0), tacticalDistanceAt(stepDistance, here.X, here.Y, goal)
	found := false
	for direction := uint8(0); direction < combat.DirectionCount; direction++ {
		outcome, err := combat.ResolveDestination(snapshot, mover, direction, state.Budget())
		if err != nil {
			return 0, false, err
		}
		if outcome.Action != combat.MovementEnter {
			continue
		}
		x, y, err := combat.AdvanceTacticalCoordinate(here.X, here.Y, direction)
		if err != nil {
			return 0, false, err
		}
		distance := tacticalDistanceAt(stepDistance, x, y, goal)
		if distance >= bestDistance {
			continue
		}
		best, bestDistance, found = direction, distance, true
	}
	return best, found, nil
}

// foeTarget 取這一格上一回合追的目標，順便驗它還有沒有效（原版 `37B8h`
// 開頭那三道：不是自己這一邊、還在場上、過得了可打判定）。
func (state *tacticalState) foeTarget(mover uint8) (uint8, bool) {
	if int(mover) >= len(state.FoeTargets) {
		return 0, false
	}
	target := state.FoeTargets[mover]
	if target == 0 || int(target) >= len(state.Roster) {
		return 0, false
	}
	if state.Roster[target].FootprintClass == 0 {
		return 0, false
	}
	moverSide, ok := state.sideOf(mover)
	if !ok {
		return 0, false
	}
	targetSide, ok := state.sideOf(target)
	if !ok || targetSide == moverSide {
		return 0, false
	}
	return target, true
}

// setFoeTarget 記下這一隻在追誰；0 代表忘掉。
func (state *tacticalState) setFoeTarget(mover, target uint8) {
	if int(mover) >= len(state.FoeTargets) {
		return
	}
	state.FoeTargets[mover] = target
}

// tacticMode 取這一格身上的戰術模式（原版存在戰鬥子結構的 `+15h`）。還沒擲過
// 的是 0，`RollTacticMode` 會替它擲一個出來。
func (state *tacticalState) tacticMode(index uint8, roll func(count, sides int) int) int {
	mode := 0
	if int(index) < len(state.TacticModes) {
		mode = int(state.TacticModes[index])
	}
	return gamepack.RollTacticMode(mode, roll)
}

// setTacticMode 把換過的模式寫回去，下一回合接著用。
func (state *tacticalState) setTacticMode(index uint8, mode int) {
	if int(index) >= len(state.TacticModes) {
		return
	}
	state.TacticModes[index] = uint8(mode)
}

// tacticalCellKey 把一格壓成一個查表用的鍵。
func tacticalCellKey(x, y uint8) int { return int(y)*256 + int(x) }

// tacticalStepDistances 從目標往外做一次寬度優先，回傳每一格到目標的步數。
//
// 挑方向要用**繞得過去的實際步數**，不是直線距離：戰場是原版的斜投影又多牆
// （spec 060），直線距離會把人帶進死角然後在那裡來回，盤面一擠就再也靠不近。
//
// 只看地形擋不擋路，不看誰站在那裡——佔用格每一步都在變，把它算進去會讓
// 同一條路每走一步就得到不同的答案。
//
// **這不是原版的選路**：原版（overlay-31 entry 6，spec 096）是先列出目標
// 周圍可站的格子再挑。這裡只是讓「繞得過去」這件事成立。
func tacticalStepDistances(grid combat.TacticalGrid, classes combat.CellClasses, targetX, targetY uint8) map[int]int {
	distance := map[int]int{tacticalCellKey(targetX, targetY): 0}
	queue := [][2]uint8{{targetX, targetY}}
	for len(queue) != 0 {
		cell := queue[0]
		queue = queue[1:]
		step := distance[tacticalCellKey(cell[0], cell[1])] + 1
		for direction := uint8(0); direction < 8; direction++ {
			x, y, err := combat.AdvanceTacticalCoordinate(cell[0], cell[1], direction)
			if err != nil {
				continue
			}
			key := tacticalCellKey(x, y)
			if _, seen := distance[key]; seen {
				continue
			}
			terrain, err := grid.TerrainAt(int(x), int(y))
			if err != nil {
				continue
			}
			record, err := combat.CellClassAt(classes, terrain)
			if err != nil || record.EntryThreshold >= 0xFF {
				continue
			}
			distance[key] = step
			queue = append(queue, [2]uint8{x, y})
		}
	}
	return distance
}

// partyClassLevels 把角色攤成原版記錄 `+96h` 起那八個職業等級。THAC0
// （spec 063）與豁免目標值（spec 075）查的是同一組索引，所以只算一次。
func partyClassLevels(member poolsave.Character) ([gamepack.ClassThac0ClassCount]uint8, error) {
	var levels [gamepack.ClassThac0ClassCount]uint8
	// 訓練過的角色帶著自己的八個等級（spec 097）；沒有的照建角的第 1 級算。
	if len(member.ClassLevels) > 0 {
		copy(levels[:], member.ClassLevels)
		return levels, nil
	}
	components, ok := creation.ClassComponents(member.ClassID)
	if !ok {
		return levels, fmt.Errorf("Pool character %q has unknown class %q", member.Name, member.ClassID)
	}
	for _, component := range components {
		index, ok := creation.ComponentClassIndex(component)
		if !ok {
			return levels, fmt.Errorf("Pool class component %q has no index", component)
		}
		if int(index) >= len(levels) {
			return levels, fmt.Errorf("Pool class component %q index %d is outside the table", component, index)
		}
		levels[index] = firstCharacterLevel
	}
	return levels, nil
}

// 物品記錄裡本規格用到的三個欄位（spec 033／035／063）。
const (
	itemTypeOffset  = 0x2e // 物品型別索引，查 DS:54E0h 那張表用
	itemPlusOffset  = 0x32 // 附魔值，武器的 +1／+2
	itemReadyOffset = 0x34 // 非零代表這件已經裝備上

	// itemCategoryWeapon 是型別表 `+0` 的武器類別，對應角色記錄的 `+CCh` 槽。
	itemCategoryWeapon = 0
)

// readiedWeapon 取出角色手上的武器。
//
// 原版把每件穿戴中的物品依型別表的類別放進 `+CCh + 類別 × 4` 的槽
//（overlay-25 `0C76h`，類別 0..8；類別 9 另外走 `+F0h`／`+F4h` 兩個戒指槽），
// 而判斷「有沒有武器」讀的是類別 0 那一格（`0E81h` 檢查 `+CCh`／`+CEh` 是不是
// 空指標，空的就走徒手那一支）。所以武器是**類別 0** 的那一件，不是物品鏈上
// 第一件標成裝備的東西——預設人物身上第一件裝備多半是戒指，那東西的傷害骰
// 是 0d0，拿它當武器整隊會打不出傷害。
//
// 槽是覆寫不是累加，所以同類別有多件時**最後一件**贏，這裡照同樣的順序。
func (a *app) readiedWeapon(member poolsave.Character) (poolsave.Item, bool) {
	var weapon poolsave.Item
	found := false
	for _, item := range member.Inventory {
		if len(item.Raw) <= itemReadyOffset || item.Raw[itemReadyOffset] == 0 {
			continue
		}
		entry, err := a.itemTypes.Entry(item.Raw[itemTypeOffset])
		if err != nil || entry.Category() != itemCategoryWeapon {
			continue
		}
		weapon, found = item, true
	}
	return weapon, found
}

// weaponCombatStats 是畫面與戰鬥共用的那一條規則：兩邊分開算，會出現
// 裝備頁顯示一組數字、打起來卻是另一組。
func (a *app) weaponCombatStats(weapon poolsave.Item, member poolsave.Character, baseThac0Internal uint8) (gamepack.WeaponStats, error) {
	if len(weapon.Raw) <= itemPlusOffset {
		return gamepack.WeaponStats{}, fmt.Errorf("Pool item %q is too short to be a weapon", weapon.Name)
	}
	return gamepack.WeaponCombatStats(a.itemTypes,
		weapon.Raw[itemTypeOffset], int(int8(weapon.Raw[itemPlusOffset])),
		weaponBearerFor(member, baseThac0Internal))
}

// weaponBearerFor 把角色接成武器規則要的形狀。
//
// `AbilityBonusesEnabled` 對應角色記錄的 `+0AAh`：原版以它決定要不要套用兩個
// 力量修正，但那個 byte 由誰寫、代表什麼還沒閉合。remake 的角色都是正常
// 建角出來的，所以先照「開著」接；等 `+0AAh` 的 producer 讀出來再改。
func weaponBearerFor(member poolsave.Character, baseThac0Internal uint8) gamepack.WeaponBearer {
	return gamepack.WeaponBearer{
		BaseThac0Internal:     baseThac0Internal,
		Strength:              member.Abilities[0],
		ExceptionalStrength:   member.ExceptionalStrength,
		Dexterity:             member.Abilities[dexterityAbilityIndex],
		AbilityBonusesEnabled: true,
	}
}

// rollDice 把 app 的骰子接成回合流程要的形狀。
func (a *app) rollDice(count, sides int) int { return a.roller.Roll(count, sides) }

// tacticalStepKeys 是原版 Move 命令的八個方向鍵，依 spec 053 對到 direction 0..7。
var tacticalStepKeys = [8]ebiten.Key{
	ebiten.KeyH, ebiten.KeyI, ebiten.KeyM, ebiten.KeyQ,
	ebiten.KeyP, ebiten.KeyO, ebiten.KeyK, ebiten.KeyG,
}

// tacticalInput 讓那八個鍵驅動 ResolveDestination，並在允許進入時提交這一步。
func (a *app) tacticalInput() error {
	state := a.tactical
	if state == nil {
		return nil
	}
	if state.Prompt {
		if a.justPressed(ebiten.KeyY) {
			state.Prompt = false
			state.startRound(a.rollDice)
			state.Status = state.say(msgStatusRound, state.Round)
		}
		if a.justPressed(ebiten.KeyN) {
			state.Prompt = false
			state.Finished, state.Outcome = true, combat.ResolveCombatOutcome(state.sideCounts())
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	// 被定身的一輪到也直接結束回合（定身術，效果碼 `34h`）。
	if state.Mover != 0 && state.hasEffect(int(state.Mover), gamepack.HoldPersonEffectCode) {
		state.Status = state.say(msgStatusHeld, state.Mover)
		state.endTurn(a.rollDice, false)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	// 睡著的一輪到就直接結束回合（spec 098 的催眠術）。
	//
	// **被迷住的不在這裡**：原版讓它倒戈之後照樣行動，只是換一邊打
	//（spec 112）。倒戈在 applyCharm 那一支做，這裡不必再擋。
	asleep := state.Mover != 0 && state.hasEffect(int(state.Mover), gamepack.SleepEffectCode)
	if asleep {
		state.Status = state.say(msgStatusAsleep, state.Mover)
		state.endTurn(a.rollDice, false)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	// 站在臭雲裡的一輪到就咳一場，這一回合也沒了（spec 121）。
	// 順序放在定身與睡著之後：那兩個原版是群組 15 裡更前面的代碼
	// （`4Ah`／`4Bh` 在 `1Eh` 之後，但 `15h`、`1Eh` 在最前面），
	// 三者都是「這一回合不能動」，先後不影響結果。
	if state.Mover != 0 && state.stinkingCloudTurn(int(state.Mover)) {
		state.Status = state.say(msgStatusCoughing, state.Mover)
		state.endTurn(a.rollDice, false)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if state.Mover != 0 && state.aiDrives(int(state.Mover)) {
		if err := a.foeTurn(state); err != nil {
			return err
		}
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if a.castTargeting {
		return a.castTargetingInput()
	}
	if a.castOpen {
		return a.castInput()
	}
	if a.justPressed(ebiten.KeyC) && state.Mover != 0 {
		a.openCastMenu()
		return nil
	}
	if a.justPressed(ebiten.KeyA) && state.Mover != 0 {
		a.beginAimedAttack()
		return nil
	}
	if a.justPressed(ebiten.KeyEnter) {
		state.endTurn(a.rollDice, false)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if a.justPressed(ebiten.KeyD) {
		state.endTurn(a.rollDice, true)
		if state.Finished {
			return a.finishCombat(state.Outcome)
		}
		return nil
	}
	if state.Mover == 0 {
		return nil
	}
	for direction, key := range tacticalStepKeys {
		if !a.justPressed(key) {
			continue
		}
		tactical, err := state.tacticalSnapshot()
		if err != nil {
			return err
		}
		outcome, err := combat.ResolveDestination(tactical, state.Mover, uint8(direction), state.Budget())
		if err != nil {
			return err
		}
		switch {
		case outcome.Leaving:
			state.Status = state.say(msgStatusOffBoard)
		case outcome.Action == combat.MovementAttack:
			// 撞到自己人不打自己人。`ProbeDestination` 忠實重現原版，它只回報
			// 「那一格站著誰」——原版的格位表（`DS:5E89h`）本來就沒有陣營，
			// 陣營在角色記錄的 `+10Eh`，所以這個判斷是呼叫端的責任。
			// 少了它，隊伍排成一列時最左邊那個往右走就會砍死自己的同伴，
			// 而戰鬥永遠打不完。
			//
			// 待證：原版撞到同伴是「擋住」還是「換位」。這裡先擋住。
			if same, err := state.sameSide(state.Mover, outcome.Target); err != nil {
				return err
			} else if same {
				state.Status = state.say(msgStatusBlocked)
				return nil
			}
			if err := a.resolveTacticalAttack(state, outcome.Target); err != nil {
				return err
			}
			// 攻擊就用掉這一次行動。原版的玩家指令迴圈（overlay-08 `0307h`）
			// 把「這一回合結束了嗎」的旗標位址交給攻擊常式
			// （`0096h:0089h`，`03D7h` 那個 `lea -2(bp)`），由它決定要不要
			// 回到 `036Ch` 再問下一個指令；敵方回合（`foeTurn`）打完也是直接
			// `endTurn`。少了這一步，同一個角色可以對同一個目標無限連打。
			//
			// 待證：戰士的多次攻擊（記錄 `+A1h`，spec 051）還沒接，接上之後
			// 這裡要改成「打完所有攻擊次數才結束」。
			state.endTurn(a.rollDice, false)
			if state.Finished {
				return a.finishCombat(state.Outcome)
			}
		case outcome.Action == combat.MovementBlocked:
			state.Status = state.say(msgStatusBlocked)
		default:
			mover := state.Roster[state.Mover]
			x, y, err := combat.AdvanceTacticalCoordinate(mover.X, mover.Y, uint8(direction))
			if err != nil {
				return err
			}
			budget, err := combat.SpendMovementStep(state.Budget(), uint8(direction))
			if err != nil {
				return err
			}
			state.Roster[state.Mover].X, state.Roster[state.Mover].Y = x, y
			state.Budgets[state.Mover] = budget
			state.Status = state.say(msgStatusMoved, direction)
		}
		return nil
	}
	return nil
}

// stagedRecordFor 找出敵方第 index 筆對應的原版怪物記錄。staged 的每一筆帶著
// 數量，所以要依序展開才對得回去。
func (a *app) stagedRecordFor(index int, friendly []bool) (gamepack.MonsterRecord, bool) {
	position := 0
	for slot := 1; slot < index; slot++ {
		if !friendly[slot] {
			position++
		}
	}
	for _, monster := range a.combatMonsters {
		if position < int(monster.Spawn.Count) {
			return monster.Record, true
		}
		position -= int(monster.Spawn.Count)
	}
	return gamepack.MonsterRecord{}, false
}

// resolveTacticalAttack 以既有的命中與傷害規則（spec 050／051）解一次攻擊。
// 目標歸零時把它的體型類別寫 0，與原版一樣讓它不再佔格、也不再參與。
func (a *app) resolveTacticalAttack(state *tacticalState, target uint8) error {
	if int(target) >= len(state.HitPoints) {
		return fmt.Errorf("Pool attack target %d is outside the roster", target)
	}
	// 原版一次行動把這一相位的兩種形態都打完：攻擊區段（overlay-13
	// `1678h..176Ah`）由第二形態倒數到第一，每一形態剩幾次由
	// `AttacksThisPhase` 給（spec 051）。巨魔的爪／爪／咬就是這樣來的。
	swings, err := a.attackSwingsThisPhase(state, state.Mover)
	if err != nil {
		return err
	}
	if len(swings) == 0 {
		// 這一相位揮不出任何一下（編碼 3 的「每兩回合三次」在單數相位）。
		state.Status = state.say(msgStatusMissed, target, uint8(a.rollDice(1, 20)))
		return nil
	}
	total, landed := 0, 0
	var lastRoll uint8
	for _, dice := range swings {
		lastRoll = uint8(a.rollDice(1, 20))
		hit, err := combat.ResolveHit(lastRoll, state.THAC0[state.Mover], state.ArmorClass[target], 0)
		if err != nil {
			return err
		}
		if !hit {
			continue
		}
		rolls := make([]uint8, dice.Count)
		for index := range rolls {
			rolls[index] = uint8(a.rollDice(1, int(dice.Sides)))
		}
		damage, err := combat.ResolveDamage(dice, rolls, 1)
		if err != nil {
			return err
		}
		landed++
		total += damage
		state.HitPoints[target] -= damage
		if state.HitPoints[target] <= 0 {
			break
		}
	}
	if landed == 0 {
		state.Status = state.say(msgStatusMissed, target, lastRoll)
		return nil
	}
	damage := total
	if state.HitPoints[target] > 0 {
		state.Status = state.say(msgStatusHit, target, damage, state.HitPoints[target])
		return nil
	}
	state.HitPoints[target] = 0
	state.rememberFootprint(int(target))
	state.Roster[target].FootprintClass = 0
	state.Scores[target] = 0
	state.States[target] = combat.DyingState
	state.Status = state.say(msgStatusDown, target)
	return nil
}

// attackSwingsThisPhase 列出這一次行動要揮幾下、每一下用哪一組骰子。
//
// 形態的順序照原版的攻擊區段：由第二形態倒數到第一。沒有骰子的形態不揮
//（第一形態空的那六隻怪物就是這樣，牠們的第一形態是特殊攻擊）。
func (a *app) attackSwingsThisPhase(state *tacticalState, mover uint8) ([]combat.DamageDice, error) {
	if int(mover) >= len(state.AttackRates) {
		return nil, fmt.Errorf("Pool attacker %d is outside the roster", mover)
	}
	var swings []combat.DamageDice
	for slot := gamepack.MonsterAttackSlots; slot >= 1; slot-- {
		dice := state.AttackForms[mover][slot-1]
		if dice.Count == 0 || dice.Sides == 0 {
			continue
		}
		count, err := combat.AttacksThisPhase(state.AttackRates[mover][slot-1], state.AttackPhase&1)
		if err != nil {
			return nil, err
		}
		for swing := uint8(0); swing < count; swing++ {
			swings = append(swings, dice)
		}
	}
	return swings, nil
}

// finishCombat 依 spec 046 契約 5 處理戰後：只有勝利才從 COMBAT 邊界停下的 PC
// 續跑戰後腳本；戰敗不得續跑，也不得用自動勝利代替戰鬥結果。
func (a *app) finishCombat(outcome combat.CombatOutcome) error {
	staged := a.combatActive
	a.tacticalPreview, a.tactical = false, nil
	a.castOpen, a.castOptions, a.castCursor = false, nil, 0
	a.castTargeting, a.castTargets, a.castTargetCursor = false, nil, 0
	a.castTargetingAttack = false
	if outcome == combat.CombatOngoing {
		// 僵局收場：雙方都還在，只是誰也碰不到誰。跟打輸一樣要把排好的遭遇
		// 清掉，否則同一場架會被重新排出來。
		a.combatActive, a.combatMonsters = false, nil
		a.statusLine = "Tactical combat ended in a stalemate; neither side could close."
		return nil
	}
	if outcome != combat.CombatVictory {
		// 輸掉之後**要把排好的遭遇清掉**，否則同一場架會被重新排出來，
		// 隊伍的生命值又回到滿的（戰鬥的生命值是另一份陣列，沒有寫回隊伍），
		// 於是打輸、重來、再打輸——實測那個迴圈永遠不會停，從外面看就是
		// 遊戲不動了。
		//
		// **原版輸掉之後做什麼還沒讀**（overlay-08 `0868h` 判完就返回給呼叫端，
		// 呼叫端那一段還沒解）。所以這裡只做「不再重來」，沒有補上結束流程。
		a.combatActive, a.combatMonsters = false, nil
		a.statusLine = "Party defeated; the post-combat script does not run."
		return nil
	}
	if !staged {
		// F5 開的是預覽盤面，不是 ECL 排出來的遭遇，所以沒有戰後腳本可以續跑。
		a.statusLine = "Tactical preview finished; no encounter was staged."
		return nil
	}
	a.awardCombatExperience()
	a.combatActive, a.combatMonsters = false, nil
	a.cellEventPending, a.cellWaitingMenu = false, false
	a.eventText, a.eventLabel = "", ""
	result, err := a.eventSession.RunUntilEvent(4096, nil, true)
	if err != nil {
		return fmt.Errorf("continue after Pool combat: %w", err)
	}
	// 戰後腳本走的是跟走進一格時同一條邊界分派。只套文字的話，戰鬥後面接的
	// `PROGRAM`、`TREASURE`、換區塊那些邊界都會停在原地——結局那一段就是
	// 這樣卡住的：打贏泰倫斯拉克斯之後 `A82Ah PROGRAM 08` 沒有人接，
	// 後面的結局文字與回到菲蘭的 `NEWECL 0` 一條都不會跑。
	return a.consumeInitialSearch(result)
}

// sideCounts 數出兩邊還站著的人，對應原版的 DS:6772h 與 DS:6773h。
// nearestReachableOpposing 先挑「直線走得到」的敵人，沒有才退回最近的那個。
//
// 走得到與否用 `combat.TraceMovement`——那支是 overlay-31 `0419h` 的重現
// （spec 057），原版本來就是用它判斷地形擋不擋路。少了這一層，怪物會盯著
// 一個隔著牆的目標，然後每回合往牆上撞、回報走了零步。
//
// **挑目標的規則本身還不是原版的**：原版的敵方回合在 overlay-09 entry 1，
// 還沒讀。這裡只是讓「盯著走不到的目標」不再發生。
func (state *tacticalState) nearestReachableOpposing(mover uint8) (uint8, bool) {
	best, bestDistance := uint8(0), 0
	from := state.Roster[mover]
	budget := uint16(state.Budgets[mover])
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] == state.Friendly[mover] || state.Roster[index].FootprintClass == 0 {
			continue
		}
		to := state.Roster[index]
		trace, err := combat.TraceMovement(state.Grid, state.Classes,
			int(from.X), int(from.Y), int(to.X), int(to.Y), budget)
		if err != nil || !trace.Complete {
			continue
		}
		// 距離用原版的算法：TraceMovement 的成本除以二（spec 098）。
		// 成本本來就以半格為單位（`limit := budget*2 + 1`），所以斜走與
		// 直走不同價；切比雪夫距離在斜投影的盤面上會低估。
		distance := int(trace.Cost) / 2
		if best == 0 || distance < bestDistance {
			best, bestDistance = uint8(index), distance
		}
	}
	if best != 0 {
		return best, true
	}
	return state.nearestOpposing(mover)
}

// nearestOpposing 回報盤面上離 mover 最近、還站著的敵對參戰者。
// 距離用原版走位的切比雪夫距離（八方向一步一格）。
func (state *tacticalState) nearestOpposing(mover uint8) (uint8, bool) {
	if int(mover) >= len(state.Friendly) || mover == 0 {
		return 0, false
	}
	best, bestDistance := uint8(0), 0
	from := state.Roster[mover]
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] == state.Friendly[mover] {
			continue
		}
		if state.Roster[index].FootprintClass == 0 {
			continue
		}
		to := state.Roster[index]
		distance := chebyshev(from.X, from.Y, to.X, to.Y)
		if best == 0 || distance < bestDistance {
			best, bestDistance = uint8(index), distance
		}
	}
	return best, best != 0
}

// chebyshev 是八方向走位下的步數距離。
func chebyshev(ax, ay, bx, by uint8) int {
	dx, dy := int(ax)-int(bx), int(ay)-int(by)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// sameSide 說兩個參戰者是不是同一邊。
func (state *tacticalState) sameSide(a, b uint8) (bool, error) {
	if int(a) >= len(state.Friendly) || int(b) >= len(state.Friendly) || a == 0 || b == 0 {
		return false, fmt.Errorf("Pool combatant index %d or %d is outside the roster", a, b)
	}
	return state.Friendly[a] == state.Friendly[b], nil
}

func (state *tacticalState) sideCounts() combat.SideCounts {
	counts := combat.SideCounts{}
	for index := 1; index < len(state.Roster); index++ {
		if state.Roster[index].FootprintClass == 0 {
			continue
		}
		if state.Friendly[index] {
			counts.Party++
		} else {
			counts.Foes++
		}
	}
	return counts
}

// awardCombatExperience 把這一場的經驗值發給隊伍（spec 097）。
//
// 原版是先把所有敵方的經驗值加總、除以「有資格分的人數」，再由每個人依自己的
// 複合職業碼調整：純職業的主屬性超過 15 多拿十分之一，複合職業除以職業數。
//
// **有資格的判準還沒讀完**：原版跳過 `+10Dh` 為 0 與狀態為 1 的成員，兩個欄位
// 的語意都還沒閉合，所以這裡讓全隊都分。倒下的成員在原版一樣分得到——
// 它擋的不是死亡。
func (a *app) awardCombatExperience() {
	if len(a.state.Party) == 0 || len(a.combatMonsters) == 0 {
		return
	}
	total := uint32(0)
	for _, monster := range a.combatMonsters {
		value := monster.Record.ExperienceValue(int(monster.Record.MaxHitPoints()))
		total += value * uint32(monster.Spawn.Count)
	}
	share := gamepack.DivideExperience(total, len(a.state.Party))
	if share == 0 {
		return
	}
	for index := range a.state.Party {
		member := &a.state.Party[index]
		code, ok := creation.ClassDOSCode(member.ClassID)
		if !ok {
			// NPC 帶的是自己的 285-byte 記錄，職業碼在 `+2Fh`。
			if len(member.Record) > gamepack.ClassCodeOffset {
				code = member.Record[gamepack.ClassCodeOffset]
			} else {
				continue
			}
		}
		member.Experience += gamepack.ExperienceShare(share, code, member.Abilities)
	}
}


// tacticalRange 是原版算兩個參戰者之間距離的方式（spec 098）：
// overlay-25 `2591h` 用 overlay-31 entry 6 建的清單找到目標那一格，
// 取**路徑成本除以二**。成本由 `0419h`（TraceMovement，spec 057）算，
// 本來就以半格為單位。
//
// 走不到（地形擋住或超出預算）回 false，呼叫端要當成「打不到」。
func (state *tacticalState) tacticalRange(from, to uint8) (int, bool) {
	if int(from) >= len(state.Roster) || int(to) >= len(state.Roster) {
		return 0, false
	}
	source, target := state.Roster[from], state.Roster[to]
	// 預算給滿：算距離不受這一回合剩多少步影響。
	trace, err := combat.TraceMovement(state.Grid, state.Classes,
		int(source.X), int(source.Y), int(target.X), int(target.Y), 0xFF)
	if err != nil || !trace.Complete {
		return 0, false
	}
	return int(trace.Cost) / 2, true
}


// withinArea 是原版判斷「在不在範圍內」的方式（spec 098）：`0912h` 把預算
// 交給 `0419h`（TraceMovement）當上限，走得到的才收進清單。
func (state *tacticalState) withinArea(from, to uint8, budget uint16) bool {
	if int(from) >= len(state.Roster) || int(to) >= len(state.Roster) {
		return false
	}
	source, target := state.Roster[from], state.Roster[to]
	trace, err := combat.TraceMovement(state.Grid, state.Classes,
		int(source.X), int(source.Y), int(target.X), int(target.Y), budget)
	return err == nil && trace.Complete
}

// 效果串列的操作。原版把串列掛在角色記錄的 `+7Fh`，新的接在尾端
//（overlay-24 entry 10），線性搜尋找到的是最早掛上的那一個（spec 059／112）。

// addEffect 掛一個效果。duration 是回合數；0 代表沒有計時，要靠別的東西摘掉。
func (state *tacticalState) addEffect(index int, code uint8, duration int, casterLevel int) {
	if index < 0 || index >= len(state.Effects) {
		return
	}
	if duration < 0 {
		duration = 0
	}
	if casterLevel < 0 {
		casterLevel = 0
	}
	if casterLevel > gamepack.EffectLevelMask {
		// `+3` 的低四位只放得下 15；原版也是這個寬度。
		casterLevel = gamepack.EffectLevelMask
	}
	state.Effects[index] = state.Effects[index].Append(
		gamepack.NewEffectNode(code, uint16(duration), uint8(casterLevel), false))
}

// hasEffect 回答那一格身上有沒有這個代碼。
func (state *tacticalState) hasEffect(index int, code uint8) bool {
	if index < 0 || index >= len(state.Effects) {
		return false
	}
	return state.Effects[index].Has(code)
}

// effectRounds 回傳某個代碼剩幾回合；沒有那個效果回 0。
func (state *tacticalState) effectRounds(index int, code uint8) int {
	if index < 0 || index >= len(state.Effects) {
		return 0
	}
	at, ok := state.Effects[index].IndexOf(code)
	if !ok {
		return 0
	}
	return int(state.Effects[index][at].Duration())
}

// removeEffect 摘掉最早掛上的那一個指定代碼。
func (state *tacticalState) removeEffect(index int, code uint8) {
	if index < 0 || index >= len(state.Effects) {
		return
	}
	state.Effects[index] = state.Effects[index].Remove(code)
}

// tickEffects 把有計時的效果各減一，歸零的摘掉。持續為 0 的不動——
// 那代表「沒有回合計時」，例如催眠與魅惑。
func (state *tacticalState) tickEffects(index int) {
	if index < 0 || index >= len(state.Effects) {
		return
	}
	list := state.Effects[index]
	kept := make(gamepack.EffectList, 0, len(list))
	for _, node := range list {
		duration := node.Duration()
		if duration == 0 {
			kept = append(kept, node)
			continue
		}
		duration--
		if duration == 0 {
			continue
		}
		node.SetDuration(duration)
		kept = append(kept, node)
	}
	state.Effects[index] = kept
}

// dispelEffects 對一格身上的效果串列逐個擲解除（overlay-22 `2356h`）。
// 回傳拿掉了幾個。
func (state *tacticalState) dispelEffects(index, casterLevel int, roll func() int) int {
	if index < 0 || index >= len(state.Effects) {
		return 0
	}
	before := state.hasEffect(index, gamepack.CharmPersonEffectCode)
	after, removed := state.Effects[index].Dispel(casterLevel, roll)
	// 摘掉魅惑要跑收尾（還原陣營與控制權）。先把節點放回去再走 releaseCharm，
	// 收尾讀的正是那個節點的位元 6——直接丟掉會把「原本是哪一邊」一起丟掉。
	if before && !after.Has(gamepack.CharmPersonEffectCode) {
		kept := state.Effects[index]
		state.Effects[index] = after
		state.releaseCharmFrom(index, kept)
	} else {
		state.Effects[index] = after
	}
	return removed
}

// releaseCharmFrom 用摘掉之前的串列跑魅惑的收尾。
func (state *tacticalState) releaseCharmFrom(index int, previous gamepack.EffectList) {
	at, ok := previous.IndexOf(gamepack.CharmPersonEffectCode)
	if !ok {
		return
	}
	if index < len(state.Friendly) {
		state.Friendly[index] = previous[at].OriginalSide() == 1
	}
	if index < len(state.AIDriven) && index < len(state.PartySlot) {
		state.AIDriven[index] = state.PartySlot[index] < 0
	}
}

// aiDriven 依隊伍索引造出「誰由 AI 走」：隊員是 false，其餘是 true。
// 對應原版記錄的 `+10Fh`。
func aiDriven(partySlot []int) []bool {
	driven := make([]bool, len(partySlot))
	for index, slot := range partySlot {
		driven[index] = slot < 0
	}
	return driven
}

// aiDrives 回答那一格是不是由 AI 走。治具會手工組 tacticalState，
// 沒填 AIDriven 時退回「不是我方就由 AI 走」的舊判準。
func (state *tacticalState) aiDrives(index int) bool {
	if index < 0 {
		return false
	}
	if index < len(state.AIDriven) {
		return state.AIDriven[index]
	}
	return index < len(state.Friendly) && !state.Friendly[index]
}

// applyCharm 重現 overlay-12 entry 14（`040Ah`，spec 112）：掛上效果碼 `0Bh`
// 之後把目標**倒戈**到施法者那一邊，並改由 AI 分派它的行動。
//
// 原本的陣營記在節點 `+3` 的位元 6，解除時還原——所以「原本是哪一邊」
// 不必另外存一份，串列自己帶著。
func (state *tacticalState) applyCharm(index, casterLevel int) {
	if index < 0 || index >= len(state.Effects) || index >= len(state.Friendly) {
		return
	}
	state.addEffect(index, gamepack.CharmPersonEffectCode, 0, casterLevel)
	at, ok := state.Effects[index].IndexOf(gamepack.CharmPersonEffectCode)
	if !ok {
		return
	}
	side := uint8(0)
	if state.Friendly[index] {
		side = 1
	}
	if !state.Effects[index][at].MarkApplied(side) {
		return
	}
	if int(state.Mover) < len(state.Friendly) {
		state.Friendly[index] = state.Friendly[state.Mover]
	}
	if index < len(state.AIDriven) {
		state.AIDriven[index] = true
	}
}

// releaseCharm 是摘掉 `0Bh` 時的收尾（`0416h`）：從節點 `+3` 的位元 6
// 還原原本的陣營，控制權也跟著回去。
func (state *tacticalState) releaseCharm(index int) {
	if index < 0 || index >= len(state.Effects) {
		return
	}
	at, ok := state.Effects[index].IndexOf(gamepack.CharmPersonEffectCode)
	if !ok {
		return
	}
	original := state.Effects[index][at].OriginalSide()
	state.Effects[index] = state.Effects[index].RemoveAt(at)
	if index < len(state.Friendly) {
		state.Friendly[index] = original == 1
	}
	if index < len(state.AIDriven) && index < len(state.PartySlot) {
		state.AIDriven[index] = state.PartySlot[index] < 0
	}
}

// rememberFootprint 在倒下之前把佔格類別存起來，死靈術要拿回來。
// 已經是 0 的不覆寫——重複倒下不該把記住的那一個抹掉。
func (state *tacticalState) rememberFootprint(index int) {
	if index < 0 || index >= len(state.Roster) || index >= len(state.Footprint) {
		return
	}
	if state.Roster[index].FootprintClass != 0 {
		state.Footprint[index] = state.Roster[index].FootprintClass
	}
}

// animateDead 重現 overlay-22 `2043h`（spec 098）。
//
// 它**不是**在盤面上生一個新的戰鬥員——是把已經死掉的人類屍體叫起來，
// 換到施法者那一邊、改成不死生物。額度是施法者等級，一次叫一個。
//
// 逐條照碼：
//
//	2082  只對狀態 6（死亡）——瀕死（5）不算
//	2090  只對 `+9Fh == 0`（人類）
//	20E8  `+10Eh` 改成施法者那一邊、`+10Fh` 設 1 交給 AI
//	2105  `+76h = 2`（驅散不死欄）、`+6Bh = 0`、`+72h = 6`（基礎移動）
//	211A  清空記憶法術陣列 `+17h + 0..14h`
//	2138  士氣：原本 > 7Fh 就設 0B2h，否則 0B3h
//	215A  `+9Fh = 4`（不死）
//	217C  生命值補到 `+32h`；掛效果碼 20h，參數是 (原陣營 << 4) + 施法者等級
//	21C0  狀態 `+10Ch = 1`
func (state *tacticalState) animateDead(casterLevel int) int {
	budget := casterLevel
	raised := 0
	for index := 1; index < len(state.Roster) && budget > 0; index++ {
		if index >= len(state.States) || state.States[index] != combat.DeadState {
			continue
		}
		if index >= len(state.CreatureType) || state.CreatureType[index] != 0 {
			continue
		}
		originalSide := uint8(0)
		if index < len(state.Friendly) && state.Friendly[index] {
			originalSide = 1
		}
		if int(state.Mover) < len(state.Friendly) && index < len(state.Friendly) {
			state.Friendly[index] = state.Friendly[state.Mover]
		}
		if index < len(state.AIDriven) {
			state.AIDriven[index] = true
		}
		if index < len(state.BaseMovement) {
			state.BaseMovement[index] = gamepack.AnimatedDeadMovementRate
		}
		if index < len(state.CreatureType) {
			state.CreatureType[index] = gamepack.CreatureTypeUndead
		}
		if index < len(state.Footprint) {
			state.Roster[index].FootprintClass = state.Footprint[index]
		}
		if index < len(state.MaxHitPoints) && state.MaxHitPoints[index] > 0 {
			state.HitPoints[index] = state.MaxHitPoints[index]
		}
		state.States[index] = gamepack.AnimatedState
		if index < len(state.DyingCounters) {
			state.DyingCounters[index] = 0
		}
		// 效果碼 20h 的參數把原陣營與施法者等級打包在一起（`20C8h`）。
		state.addEffect(index, gamepack.AnimateDeadEffectCode, 0, casterLevel)
		if at, ok := state.Effects[index].IndexOf(gamepack.AnimateDeadEffectCode); ok {
			state.Effects[index][at].MarkApplied(originalSide)
		}
		budget--
		raised++
	}
	return raised
}
