package main

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 戰鬥動畫：彈道、閃光與倒下（spec 166，issue #121）。
//
// 原版的動畫都畫在 `COMSPR.DAX` 的圖格上。開場時 overlay-11 `04FEh..0546h` 把區塊 0..0Bh
// 載進圖示槽 0Dh..18h、區塊 19h 載進槽 19h（overlay-33 entry 5，`COMSPR` 不換色）；之後各處
// 只說槽位。動畫一律用 overlay-25 的三支：
//
//	entry 23（`18E1h`）(槽, 動作?, 第幾格, 翻轉?)  把一張圖抄進 `DS:6D29h` 那張四格合成圖
//	entry 24（`1A30h`）(槽)                        四格：站立、站立翻轉、動作翻轉、動作
//	entry 25（`1A7Dh`）(起 X,Y, 終 X,Y, 格數, 毫秒) 沿直線以 8 像素一步畫過去，每步 Delay(毫秒)
//
// remake 的戰鬥邏輯在一個影格裡算完，畫面靠 tacticalState.Notices 一則一則放（combat_notice.go）。
// 動畫也是一則：Ticks 是它的長度，停拍期間 tacticalInput 不做別的事，所以**不改任何擲骰的
// 先後**——動畫只佔時間，不佔亂數。

// comsprSlotBase 是 `COMSPR.DAX` 區塊 0 所在的圖示槽（overlay-11 `051Ch..0525h`：槽 = 區塊 + 0Dh）。
const comsprSlotBase = 0x0D

// 用到的槽位（區塊 = 槽 − 0Dh）。
const (
	slotSpellBolt   = 0x12 // COMSPR 5：施法的那一道（overlay-22 `0D84h`）、凝視（overlay-12 `1D03h`）
	slotRay         = 0x13 // COMSPR 6：射線與吐息（overlay-22 `2906h`／`2922h`／`316Ah`）
	slotSparkle     = 0x16 // COMSPR 9：entry 26 旗標 1 的閃光
	slotHurtSparkle = 0x17 // COMSPR 0Ah：entry 26 旗標 0（受傷）的閃光；噴酸也用它
	slotSkull       = 0x18 // COMSPR 0Bh：倒下（overlay-32 entry 20 讀 `DS:5E78h` = 槽 18h 站立圖）
)

// 動畫的毫秒數（每一個都是 `0512h:029Eh` Delay 的參數）。
const (
	sparkleMilliseconds    = turnedSparkleMilliseconds // entry 26 `2173h`：46h
	spellBoltMilliseconds  = 0x1E                      // overlay-22 `0E21h`
	rayMilliseconds        = 0x32                      // overlay-22 `2AA5h`、`3182h`
	gazeMilliseconds       = 0x2D                      // overlay-12 `1D3Fh`、`1E0Eh`、`1F76h`
	acidSpitMilliseconds   = 0x1E                      // overlay-12 `2CE6h`
	sparkleFramesPerRound  = 4                         // entry 26 `2145h..21A5h` 畫 0..3
	missileFramesComposite = 4
)

// animationSprite 是一張 `COMSPR.DAX` 圖格（Block 是區塊編號，不是槽位）。
type animationSprite struct {
	Block  uint8
	Action bool
}

// animationFrame 是 `DS:6D29h` 合成圖的一格：哪一張、要不要左右翻（entry 23 的 `1953h` 走 `18E:0052`）。
type animationFrame struct {
	Block        uint8
	Action, Flip bool
}

type animationKind uint8

const (
	animationMissile animationKind = iota + 1
	animationSparkle
	animationSkull
)

// missileDraw 是 entry 25 畫的一次：位置以 8 像素為單位（格 × 3，原版 `[bp-0C1h]`／`[bp-0C2h]`
// 換成絕對座標），Frame 是 `[bp-0B3h]`。
type missileDraw struct {
	X, Y  int
	Frame int
}

// combatAnimation 是一則停拍期間盤面上播的東西。
type combatAnimation struct {
	Kind   animationKind
	Frames []animationFrame
	// Draws 與 StepMilliseconds 是彈道：每一次畫停 StepMilliseconds。
	Draws            []missileDraw
	StepMilliseconds int
	// Cell 與 Rounds 是閃光：在那一格畫四格、每格 StepMilliseconds，輪 Rounds 次。
	Cell   image.Point
	Rounds int
	// Index 與 Cells 是倒下：那一個人佔的每一格都蓋一張骷髏。
	Index int
	Cells []image.Point
	// Ticks 是這一則開始時的停拍長度，動畫用它算已經過了幾個影格。
	Ticks int
}

// comsprBlock 把槽位換回 `COMSPR.DAX` 的區塊。
func comsprBlock(slot uint8) uint8 {
	if slot == 0x19 {
		return 0x19
	}
	return slot - comsprSlotBase
}

// fourFrames 是 entry 24（`1A30h`）：`1A33h..1A74h` 依序以 (動作 0, 格 0, 不翻)、(0, 1, 翻)、
// (1, 2, 翻)、(1, 3, 不翻) 叫 entry 23。
func fourFrames(slot uint8) []animationFrame {
	block := comsprBlock(slot)
	return []animationFrame{{Block: block}, {Block: block, Flip: true},
		{Block: block, Action: true, Flip: true}, {Block: block, Action: true}}
}

// millisecondsToTicks 把 Delay 的毫秒換成 60 Hz 的影格，進位——最後一格至少畫得出來。
func millisecondsToTicks(milliseconds int) int {
	if milliseconds <= 0 {
		return 0
	}
	return (milliseconds*60 + 999) / 1000
}

// missileDraws 是 entry 25（`1A7Dh`）畫的每一步，delay 大於 0 的那一條路（所有呼叫端都是）。
//
//	1AFAh..1B2Ch  overlay-31 走訪器（`0196h`／`02C4h`）從起點 ×3 走到終點 ×3，每叫一次把
//	              `+19h` 的方向存進 `[bp-94h]` 表（整表先填 08h），走不動的那一次也存
//	1B3Bh         存了不到兩筆就整支返回
//	1CDDh..1E73h  畫在目前位置、Delay、擦掉；格數加一、到 frames 歸零；筆數加一，照
//	              **這一筆**的方向（`DS:274Ah`／`2753h`）移一步——所以第一筆的方向從來沒用上
//	1F4Dh..2027h  走完之後在終點再畫一次、Delay、擦掉
//
// 位置出了 7×7 的視窗原版會捲動畫面（`13D:0061`）；remake 的視窗不跟著捲，出框的那幾步
// 照樣佔時間、只是畫不出來。
func missileDraws(fromX, fromY, toX, toY, frames int) []missileDraw {
	walker := combat.NewStepWalker(fromX*3, fromY*3, toX*3, toY*3)
	directions := []uint8{}
	for guard := 0; guard < 0x94; guard++ {
		moved := walker.Step()
		directions = append(directions, walker.Direction)
		if !moved {
			break
		}
	}
	if len(directions) < 2 || frames <= 0 {
		return nil
	}
	x, y, frame := fromX*3, fromY*3, 0
	draws := []missileDraw{}
	for step := 0; step < len(directions); {
		draws = append(draws, missileDraw{X: x, Y: y, Frame: frame})
		frame++
		if frame >= frames {
			frame = 0
		}
		step++
		direction := uint8(8)
		if step < len(directions) {
			direction = directions[step]
		}
		x += stepDeltaX[direction]
		y += stepDeltaY[direction]
	}
	return append(draws, missileDraw{X: toX * 3, Y: toY * 3, Frame: frame})
}

// stepDeltaX／stepDeltaY 是 `DS:274Ah`／`DS:2753h`（方向 0..7 從北順時針，8 不動）。
var (
	stepDeltaX = [9]int{0, 1, 1, 1, 0, -1, -1, -1, 0}
	stepDeltaY = [9]int{-1, -1, 0, 1, 1, 1, 0, -1, 0}
)

// queueAnimation 把一則動畫排進停拍佇列。ticks 是這一則的長度（動畫本身加上之後的停拍）。
//
// 前一則是**不停拍**的 entry 20（Kept）時，原版那幾行字還留在右欄——它不停拍就不清
// （combat_notice.go），動畫是畫在它旁邊的。所以把那幾行抄過來，動畫期間照樣看得到。
func (a *app) queueAnimation(state *tacticalState, anim *combatAnimation, ticks int) {
	if anim == nil || ticks <= 0 {
		return
	}
	anim.Ticks = ticks
	notice := combatNotice{Anim: anim, Ticks: ticks}
	if last := len(state.Notices) - 1; last >= 0 && state.Notices[last].Kept &&
		state.Notices[last].Ticks == 0 && state.Notices[last].Anim == nil {
		previous := state.Notices[last]
		notice.Name, notice.Text, notice.Row, notice.NameInk = previous.Name, previous.Text, previous.Row, previous.NameInk
		notice.Target, notice.TargetInk, notice.Detail = previous.Target, previous.TargetInk, previous.Detail
		notice.Bottom = previous.Bottom
	}
	state.Notices = append(state.Notices, notice)
}

// missileAnimation 排一段 entry 25：從 (fromX,fromY) 到 (toX,toY)，count 是合成圖用幾格、
// milliseconds 是每一步的 Delay。走不到兩步（同一格）就不畫——`1B3Bh` 直接返回。
func (a *app) missileAnimation(state *tacticalState, fromX, fromY, toX, toY int,
	frames []animationFrame, count, milliseconds int) {
	if state == nil || len(frames) == 0 {
		return
	}
	if count > len(frames) {
		count = len(frames)
	}
	draws := missileDraws(fromX, fromY, toX, toY, count)
	if len(draws) == 0 {
		return
	}
	anim := &combatAnimation{Kind: animationMissile, Frames: frames, Draws: draws, StepMilliseconds: milliseconds}
	a.queueAnimation(state, anim, millisecondsToTicks(len(draws)*milliseconds))
}

// combatantMissile 是兩個戰鬥員之間的 entry 25（起終點是 `13D:006B`／`0070`，記錄的 X、Y）。
func (a *app) combatantMissile(state *tacticalState, from, to uint8, frames []animationFrame,
	count, milliseconds int) {
	if state == nil || int(from) >= len(state.Roster) || int(to) >= len(state.Roster) {
		return
	}
	source, target := state.Roster[from], state.Roster[to]
	a.missileAnimation(state, int(source.X), int(source.Y), int(target.X), int(target.Y),
		frames, count, milliseconds)
}

// weaponMissileFrames 是 overlay-13 entry 24（`268Eh`）挑圖的那一段：依彈藥的物品型別（`+2Eh`）
// 與攻擊者看目標的方向（`261Bh`）。回傳合成圖、用幾格、每步幾毫秒。
//
//	26DBh  型別 09h／15h／1Ch／1Fh／49h：一格、10 毫秒（`26ABh` 的預設）
//	         方向是奇數（斜）：槽 0Eh；3、5 用動作圖，5、7 左右翻
//	         方向是偶數：槽 0Dh + 方向 mod 4（0Dh 或 0Fh），方向 ÷ 4 決定動作圖
//	277Ch  型別 02h／07h／14h：entry 24(槽 10h)，四格、50 毫秒
//	27A3h  型別 55h／56h：entry 24(槽 11h)，四格、50 毫秒
//	27C5h  其餘：槽 14h（型別 2Fh 是 15h）的站立與動作兩格、20 毫秒
func weaponMissileFrames(itemType, direction uint8) ([]animationFrame, int, int) {
	switch itemType {
	case 0x09, 0x15, 0x1C, 0x1F, 0x49:
		if direction&1 == 1 {
			return []animationFrame{{Block: comsprBlock(0x0E), Action: direction == 3 || direction == 5,
				Flip: direction == 5 || direction == 7}}, 1, 0x0A
		}
		return []animationFrame{{Block: comsprBlock(0x0D + direction%4), Action: direction/4 == 1}}, 1, 0x0A
	case 0x02, 0x07, 0x14:
		return fourFrames(0x10), missileFramesComposite, 0x32
	case 0x55, 0x56:
		return fourFrames(0x11), missileFramesComposite, 0x32
	}
	slot := uint8(0x14)
	if itemType == 0x2F {
		slot = 0x15
	}
	block := comsprBlock(slot)
	return []animationFrame{{Block: block}, {Block: block, Action: true}}, 2, 0x14
}

// weaponMissile 是攻擊包裝 `1883h` 的 `193Ah..1981h`：帶著彈藥就以那一件叫 entry 24（`268Eh`）；
// 攻擊者手上（`+0CCh`）是型別 2Fh 的武器，再以那一件叫一次。都在擲命中（`19D2h` 的 `1404h`）之前。
func (a *app) weaponMissile(state *tacticalState, attacker, target uint8, items [][]byte,
	ammunition, weapon int) {
	for _, index := range []int{ammunition, weapon} {
		if index < 0 || index >= len(items) || len(items[index]) <= gamepack.ItemTypeOffset {
			continue
		}
		itemType := items[index][gamepack.ItemTypeOffset]
		if index == weapon && itemType != 0x2F {
			continue
		}
		direction := uint8(0)
		if int(attacker) < len(state.Roster) && int(target) < len(state.Roster) {
			from, to := state.Roster[attacker], state.Roster[target]
			if facing, ok := combatFacingTowards(from.X, from.Y, to.X, to.Y); ok {
				direction = facing
			}
		}
		frames, count, milliseconds := weaponMissileFrames(itemType, direction)
		a.combatantMissile(state, attacker, target, frames, count, milliseconds)
	}
}

// sparkleAnimation 是 entry 26（`2041h`）戰鬥中的那一段：entry 24 以槽 16h（旗標 1）或 17h
// （旗標 0）組四格，在目標那一格（`13D:007A` → `5FA8h`／`5FF0h`）逐格畫、每格 Delay(46h)。
func sparkleAnimation(state *tacticalState, index uint8, slot uint8, rounds int) *combatAnimation {
	if state == nil || int(index) >= len(state.Roster) || rounds <= 0 {
		return nil
	}
	cell := state.Roster[index]
	return &combatAnimation{Kind: animationSparkle, Frames: fourFrames(slot),
		StepMilliseconds: sparkleMilliseconds, Cell: image.Pt(int(cell.X), int(cell.Y)), Rounds: rounds}
}

// hurtSparkleTicks 是旗標 0 的長度：一輪四格（`20EEh` 輪數寫 0 → 只跑一輪），再等一拍（`21BAh`）。
func (a *app) hurtSparkleTicks() int {
	return millisecondsToTicks(sparkleFramesPerRound*sparkleMilliseconds) + a.speedDelayTicks()
}

// 受傷那一句（overlay-24 entry 19 的字串，`127Dh..12DFh`）另開 `iota + 5900`。
const (
	// msgHurtTakes 是 "takes " + 數字 + " points of damage "（`127Dh`、`1284h`）。
	msgHurtTakes messageID = iota + 5900
	// msgHurtTakesOne 是 "takes 1 point of damage "（`1297h`）。
	msgHurtTakesOne
	// 以下是 `13DEh..14D8h` 接在後面的種類：`6777h & F7h` 是 1／2／4／10h，或 `6777h & 8 == 6777h`。
	msgHurtFromFire
	msgHurtFromCold
	msgHurtFromElectricity
	msgHurtFromAcid
	msgHurtFromMagic
)

func init() {
	for id, key := range map[messageID]string{
		msgHurtTakes:           "ui.hurtTakes",
		msgHurtTakesOne:        "ui.hurtTakesOne",
		msgHurtFromFire:        "ui.hurtFromFire",
		msgHurtFromCold:        "ui.hurtFromCold",
		msgHurtFromElectricity: "ui.hurtFromElectricity",
		msgHurtFromAcid:        "ui.hurtFromAcid",
		msgHurtFromMagic:       "ui.hurtFromMagic",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// hurtText 是 overlay-24 entry 19 `1384h..14D8h` 組的那一句。flags 是 `DS:6777h`：
// `& F7h` 等於 1／2／4／10h 各接一種（`13DEh`），**另外**`& 8` 等於它本身（包含 0）就接
// "from Magic"（`14A8h`）——兩個判斷各自獨立，照抄。
func hurtText(state *tacticalState, damage int, flags uint8) string {
	text := state.say(msgHurtTakes, damage)
	if damage == 1 {
		text = state.say(msgHurtTakesOne)
	}
	switch flags & 0xF7 {
	case 0x01:
		text += state.say(msgHurtFromFire)
	case 0x02:
		text += state.say(msgHurtFromCold)
	case 0x04:
		text += state.say(msgHurtFromElectricity)
	case 0x10:
		text += state.say(msgHurtFromAcid)
	}
	if flags&0x08 == flags {
		text += state.say(msgHurtFromMagic)
	}
	return text
}

// hurtNotice 是 overlay-24 entry 19 在戰鬥中經 entry 26 旗標 0 印傷害那一句（`14ECh`）：
// 名字與那一句不停拍地印、閃光一輪、等一拍。
func (a *app) hurtNotice(state *tacticalState, index uint8, text string) {
	a.panelNotice(state, index, text, noticeRowPanel, false)
	notice := &state.Notices[len(state.Notices)-1]
	notice.Anim = sparkleAnimation(state, index, slotHurtSparkle, 1)
	notice.Ticks = a.hurtSparkleTicks()
	if notice.Anim != nil {
		notice.Anim.Ticks = notice.Ticks
	}
}

// skullAnimation 是 overlay-32 entry 20（`0E11h`）戰鬥中的那一段：這個人還不在屍體表
// （`0E2Fh..0E5Fh`）才做；在他佔的每一格（`DS:5E88h` 的體型、`DS:2858h` 的偏移）畫 `DS:5E78h`
// （槽 18h 站立圖＝COMSPR 0Bh 的骷髏），`0F3Eh` 顯示、`1001h` 聲音、`1006h` 等一拍，
// 然後才把他從盤面上拿掉（`1016h..1055h`）。
func (a *app) skullAnimation(state *tacticalState, index int) {
	if state == nil || index <= 0 || index >= len(state.Roster) {
		return
	}
	class := uint8(1)
	if index < len(state.Footprint) && state.Footprint[index] != 0 {
		class = state.Footprint[index]
	}
	cell := state.Roster[index]
	cells := []image.Point{}
	for _, slot := range combat.FootprintCells(class, cell.X, cell.Y) {
		if slot.Valid() {
			cells = append(cells, image.Pt(int(slot.X), int(slot.Y)))
		}
	}
	a.queueAnimation(state, &combatAnimation{Kind: animationSkull,
		Frames: []animationFrame{{Block: comsprBlock(slotSkull)}}, Index: index, Cells: cells}, a.speedDelayTicks())
}

// downBeat 是倒下之後那一拍（overlay-13 `0608h..0620h`、overlay-24 `1623h..1638h`）：群組 13
// 之後還躺著（`+10Dh` 為 0）就叫 overlay-32 entry 20——畫骷髏、等一拍；被群組 13 救起來
// 就只等一拍。entry 20 對已經在屍體表上的人什麼都不做。
func (a *app) downBeat(state *tacticalState, index int, alreadyCorpse bool) {
	if state == nil || index <= 0 || index >= len(state.Roster) {
		return
	}
	if state.Roster[index].FootprintClass != 0 {
		if ticks := a.speedDelayTicks(); ticks > 0 {
			state.Notices = append(state.Notices, combatNotice{Ticks: ticks})
		}
		return
	}
	if !alreadyCorpse {
		a.skullAnimation(state, index)
	}
}

// animationImage 取一張 COMSPR 圖格，載一次就留著。
func (a *app) animationImage(frame animationFrame) *ebiten.Image {
	if a.loadMonsterSprite == nil {
		return nil
	}
	key := animationSprite{Block: frame.Block, Action: frame.Action}
	if picture, ok := a.animationSprites[key]; ok {
		return picture
	}
	if a.animationSprites == nil {
		a.animationSprites = map[animationSprite]*ebiten.Image{}
	}
	loaded, err := a.loadMonsterSprite(frame.Block, frame.Action)
	if err != nil {
		loaded = nil
	}
	a.animationSprites[key] = loaded
	return loaded
}

// drawBoardPicture 把一張圖以 8 像素單位的位置畫進戰場，出了 7×7 的視窗就裁掉。flip 是左右翻
// （`18E:0052`）。
func drawBoardPicture(screen *ebiten.Image, state *tacticalState, picture *ebiten.Image, unitX, unitY int, flip bool) {
	if picture == nil {
		return
	}
	left := combatBoardLeft + (unitX-int(state.Viewport.X)*3)*combatBoardCell/3
	top := combatBoardTop + (unitY-int(state.Viewport.Y)*3)*combatBoardCell/3
	op := &ebiten.DrawImageOptions{}
	if flip {
		op.GeoM.Scale(-2, 2)
		op.GeoM.Translate(float64(left+picture.Bounds().Dx()*2), float64(top))
	} else {
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(left), float64(top))
	}
	boardClip(screen).DrawImage(picture, op)
}

// boardClip 是 7×7 視窗那一塊（原版的圖都畫在這一塊裡）。
func boardClip(screen *ebiten.Image) *ebiten.Image {
	span := combat.ViewportTileSpan * combatBoardCell
	return screen.SubImage(image.Rect(combatBoardLeft, combatBoardTop,
		combatBoardLeft+span, combatBoardTop+span)).(*ebiten.Image)
}

// elapsedMilliseconds 是這一則已經播了多久。
func (notice combatNotice) elapsedMilliseconds() int {
	if notice.Anim == nil {
		return 0
	}
	return (notice.Anim.Ticks - notice.Ticks) * 1000 / 60
}

// animationFrameAt 是這一則在 elapsed 毫秒時畫哪一格、畫在哪裡（8 像素單位）；沒東西畫回 false。
// 倒下那一種不走這裡（每一格都畫同一張）。
func (anim *combatAnimation) frameAt(elapsed int) (animationFrame, int, int, bool) {
	if anim == nil || anim.StepMilliseconds <= 0 {
		return animationFrame{}, 0, 0, false
	}
	step := elapsed / anim.StepMilliseconds
	switch anim.Kind {
	case animationMissile:
		if step >= len(anim.Draws) || anim.Draws[step].Frame >= len(anim.Frames) {
			return animationFrame{}, 0, 0, false
		}
		draw := anim.Draws[step]
		return anim.Frames[draw.Frame], draw.X, draw.Y, true
	case animationSparkle:
		if step >= anim.Rounds*sparkleFramesPerRound {
			return animationFrame{}, 0, 0, false
		}
		return anim.Frames[step%len(anim.Frames)], anim.Cell.X * 3, anim.Cell.Y * 3, true
	}
	return animationFrame{}, 0, 0, false
}

// drawCombatAnimation 畫停拍中那一則的動畫（沒有就什麼都不畫）。
func drawCombatAnimation(screen *ebiten.Image, a *app) {
	state := a.tactical
	notice, ok := state.shownNotice()
	if !ok || notice.Anim == nil {
		return
	}
	anim := notice.Anim
	if anim.Kind == animationSkull {
		skull := a.animationImage(anim.Frames[0])
		for _, cell := range anim.Cells {
			drawBoardPicture(screen, state, skull, cell.X*3, cell.Y*3, false)
		}
		return
	}
	if frame, x, y, ok := anim.frameAt(notice.elapsedMilliseconds()); ok {
		drawBoardPicture(screen, state, a.animationImage(frame), x, y, frame.Flip)
	}
}

// pendingSkull 回報第 index 格是不是還在等倒下那一段：佇列裡有它的骷髏、而且還沒輪到。
// 原版在骷髏那一刻之前，這個人一直畫在盤面上（`1016h` 之後才擦掉）。
func (state *tacticalState) pendingSkull(index int) bool {
	for position, notice := range state.Notices {
		if notice.Anim != nil && notice.Anim.Kind == animationSkull && notice.Anim.Index == index {
			return position > 0
		}
	}
	return false
}
