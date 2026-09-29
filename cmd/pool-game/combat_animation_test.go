package main

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 戰鬥動畫（spec 166）：彈道、閃光、倒下。這幾條釘住三件事——畫什麼（COMSPR 的哪一格）、
// 畫在哪（原版走訪器的每一步）、畫多久（Delay 的毫秒換成影格），以及它們**只佔停拍**。

// entry 25（`1A7Dh`）：從 (0,0) 往東兩格。走訪器以 8 像素一步走 6 步、再叫一次走不動（方向 08h），
// 存下 7 筆；第一筆的方向沒用上，所以畫在 0,1,2,3,4,5,5（×8 像素），最後在終點 6 再畫一次。
// 四格輪著用，畫一次換一格、到 4 歸零。
func TestMissileDrawsFollowTheOriginalWalker(t *testing.T) {
	draws := missileDraws(0, 0, 2, 0, 4)
	wantX := []int{0, 1, 2, 3, 4, 5, 5, 6}
	wantFrame := []int{0, 1, 2, 3, 0, 1, 2, 3}
	if len(draws) != len(wantX) {
		t.Fatalf("%d draws %+v, want %d", len(draws), draws, len(wantX))
	}
	for index, draw := range draws {
		if draw.X != wantX[index] || draw.Y != 0 || draw.Frame != wantFrame[index] {
			t.Fatalf("draw %d = %+v, want x %d frame %d", index, draw, wantX[index], wantFrame[index])
		}
	}
	// 斜走：(1,1) → (0,0)，每一步 x、y 一起減。
	diagonal := missileDraws(1, 1, 0, 0, 1)
	if last := diagonal[len(diagonal)-1]; last.X != 0 || last.Y != 0 || diagonal[1].X != 2 || diagonal[1].Y != 2 {
		t.Fatalf("diagonal draws %+v", diagonal)
	}
	// `1B3Bh`：存了不到兩筆（同一格）就整支返回。
	if draws := missileDraws(3, 3, 3, 3, 4); draws != nil {
		t.Fatalf("same cell drew %+v", draws)
	}
}

// overlay-13 entry 24（`268Eh`）依彈藥型別挑圖：箭／弩矢／標槍／飛鏢／矛是單張、依方向挑、10 毫秒；
// 手斧／棍棒／鎚轉四格、50 毫秒；聖水與油瓶四格、50 毫秒；其餘（匕首）兩格、20 毫秒，投石索換一張。
func TestWeaponMissileFramesFollowTheItemType(t *testing.T) {
	cases := []struct {
		itemType, direction uint8
		frames              []animationFrame
		count, milliseconds int
	}{
		{gamepack.ItemTypeArrow, 0, []animationFrame{{Block: 0}}, 1, 10},
		{gamepack.ItemTypeArrow, 2, []animationFrame{{Block: 2}}, 1, 10},
		{gamepack.ItemTypeArrow, 4, []animationFrame{{Block: 0, Action: true}}, 1, 10},
		{gamepack.ItemTypeQuarrel, 6, []animationFrame{{Block: 2, Action: true}}, 1, 10},
		{0x15, 1, []animationFrame{{Block: 1}}, 1, 10},
		{0x09, 3, []animationFrame{{Block: 1, Action: true}}, 1, 10},
		{0x1F, 5, []animationFrame{{Block: 1, Action: true, Flip: true}}, 1, 10},
		{0x49, 7, []animationFrame{{Block: 1, Flip: true}}, 1, 10},
		{0x02, 0, fourFrames(0x10), 4, 50},
		{0x56, 0, fourFrames(0x11), 4, 50},
		{0x08, 0, []animationFrame{{Block: 7}, {Block: 7, Action: true}}, 2, 20},
		{0x2F, 0, []animationFrame{{Block: 8}, {Block: 8, Action: true}}, 2, 20},
	}
	for _, test := range cases {
		frames, count, milliseconds := weaponMissileFrames(test.itemType, test.direction)
		if count != test.count || milliseconds != test.milliseconds || len(frames) != len(test.frames) {
			t.Fatalf("type %02X dir %d: %+v ×%d %d ms", test.itemType, test.direction, frames, count, milliseconds)
		}
		for index := range frames {
			if frames[index] != test.frames[index] {
				t.Fatalf("type %02X dir %d frame %d = %+v, want %+v", test.itemType, test.direction,
					index, frames[index], test.frames[index])
			}
		}
	}
	// entry 24 的四格：站立、站立翻、動作翻、動作。
	if got := fourFrames(0x12); got[0] != (animationFrame{Block: 5}) || got[1] != (animationFrame{Block: 5, Flip: true}) ||
		got[2] != (animationFrame{Block: 5, Action: true, Flip: true}) || got[3] != (animationFrame{Block: 5, Action: true}) {
		t.Fatalf("fourFrames(12h) = %+v", got)
	}
}

// 玩家射一箭：彈道在攻擊那一則之前、只佔停拍；停完之後盤面上的東西與射出去那一刻一樣
// ——生命值、行動計數、輪到誰都沒有因為動畫而變。
func TestArrowFlightHoldsBeforeTheAttackAndChangesNothing(t *testing.T) {
	application, _ := orcHomeGearFixture(t)
	application.roller = fixedRoller{value: 1}
	state := application.tactical
	archer := uint8(0)
	for index := 1; index < len(state.Roster); index++ {
		if state.Friendly[index] && state.PartySlot[index] >= 0 {
			archer = uint8(index)
			break
		}
	}
	slot := state.PartySlot[archer]
	bow, arrows := leaderItem(t, state, 0x2b), leaderItem(t, state, gamepack.ItemTypeArrow)
	bow.Raw[gamepack.ItemReadiedOffset], arrows.Raw[gamepack.ItemReadiedOffset] = 1, 1
	application.state.Party[slot].Inventory = []poolsave.Item{bow, arrows}
	if err := application.applyPartyGearStats(state, int(archer), application.state.Party[slot]); err != nil {
		t.Fatal(err)
	}
	clearFoesExcept(state, missileLeader)
	placeForShot(t, state, missileLeader, archer, archer, state.attackRangeOf(archer))
	drainCombatNotices(t, application)
	state.Prompt, state.Mover = false, archer
	state.swingsLeft = nil
	for _, key := range []ebiten.Key{ebiten.KeyA, ebiten.KeyEnter} {
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	if len(state.Notices) < 2 || state.Notices[0].Anim == nil || state.Notices[0].Anim.Kind != animationMissile {
		t.Fatalf("the first notice after the shot is not the arrow's flight: %+v", state.Notices)
	}
	flight := state.Notices[0]
	from, to := state.Roster[archer], state.Roster[missileLeader]
	direction, _ := combatFacingTowards(from.X, from.Y, to.X, to.Y)
	frames, count, milliseconds := weaponMissileFrames(gamepack.ItemTypeArrow, direction)
	draws := missileDraws(int(from.X), int(from.Y), int(to.X), int(to.Y), count)
	if len(flight.Anim.Frames) != len(frames) || flight.Anim.Frames[0] != frames[0] ||
		flight.Anim.StepMilliseconds != milliseconds || len(flight.Anim.Draws) != len(draws) {
		t.Fatalf("flight %+v, want frames %+v at %d ms over %d draws", flight.Anim, frames, milliseconds, len(draws))
	}
	if want := millisecondsToTicks(len(draws) * milliseconds); flight.Ticks != want {
		t.Fatalf("flight holds %d frames, want %d (%d draws × %d ms)", flight.Ticks, want, len(draws), milliseconds)
	}
	if _, ok := attackNoticeIn(state); !ok {
		t.Fatalf("no attack notice after the flight: %+v", state.Notices)
	}
	// 擲骰都在按下去的那一影格擲完；動畫期間 tacticalInput 什麼都不做。
	hitPoints := append([]int(nil), state.HitPoints...)
	attacks, mover := state.Activity.PartyAttacks, state.Mover
	for frame := 0; frame < flight.Ticks; frame++ {
		if notice, ok := state.shownNotice(); !ok || notice.Anim == nil || notice.Anim.Kind != animationMissile {
			t.Fatalf("frame %d: the flight ended early: %+v", frame, state.Notices)
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	drainCombatNotices(t, application)
	for index := range hitPoints {
		if state.HitPoints[index] != hitPoints[index] {
			t.Fatalf("hit points changed during the animation: %v → %v", hitPoints, state.HitPoints)
		}
	}
	if state.Activity.PartyAttacks != attacks || (state.Mover != mover && state.Mover == archer) {
		t.Fatalf("attacks %d → %d, mover %d → %d", attacks, state.Activity.PartyAttacks, mover, state.Mover)
	}
}

// 倒下（overlay-32 entry 20）：還不在屍體表的人，在他佔的每一格畫骷髏（COMSPR 0Bh 站立圖）、
// 停一拍；輪到骷髏之前他照樣畫在盤面上。已經在屍體表上的整支不做。
func TestDownedCombatantShowsTheSkullForOneBeat(t *testing.T) {
	application, state := newFoeCastApp(t)
	application.gameSpeed = 4
	beat := application.speedDelayTicks()
	state.Notices = []combatNotice{{Name: "EARLIER", Ticks: 3}}
	state.rememberFootprint(2)
	state.Roster[2].FootprintClass = 0
	application.combatantDown(state, 2, 0)
	if len(state.Notices) != 2 {
		t.Fatalf("notices %+v, want the earlier one and the skull", state.Notices)
	}
	skull := state.Notices[1]
	cell := state.Roster[2]
	if skull.Anim == nil || skull.Anim.Kind != animationSkull || skull.Ticks != beat || skull.Anim.Index != 2 ||
		skull.Anim.Frames[0] != (animationFrame{Block: 0x0B}) ||
		len(skull.Anim.Cells) != 1 || skull.Anim.Cells[0] != image.Pt(int(cell.X), int(cell.Y)) {
		t.Fatalf("skull %+v anim %+v, want COMSPR 0Bh on (%d,%d) for %d frames", skull, skull.Anim, cell.X, cell.Y, beat)
	}
	if !state.pendingSkull(2) {
		t.Fatal("before the skull's turn the fallen combatant must still be drawn")
	}
	state.Notices = state.Notices[1:]
	if state.pendingSkull(2) {
		t.Fatal("while the skull shows, the icon is not drawn")
	}
	// 怪物不進屍體表（`0F4Bh`：runtime `+13h` 為 0 才記），再倒一次照樣有骷髏。
	state.Notices = nil
	application.combatantDown(state, 2, 0)
	if len(state.Notices) != 1 || state.Notices[0].Anim == nil || state.Notices[0].Anim.Kind != animationSkull {
		t.Fatalf("a monster fell again and got %+v", state.Notices)
	}
	// 隊員第一次倒下記進屍體表；已經在表上的再倒一次：entry 20 `0E59h` 直接返回，沒有骷髏也不等。
	state.rememberFootprint(1)
	state.Roster[1].FootprintClass = 0
	state.Notices = nil
	application.combatantDown(state, 1, 0)
	if len(state.Notices) != 1 {
		t.Fatalf("the party member's first fall: %+v", state.Notices)
	}
	state.Notices = nil
	application.combatantDown(state, 1, 0)
	if len(state.Notices) != 0 {
		t.Fatalf("a corpse fell again and got %+v", state.Notices)
	}
	// 遊戲速度 0：一拍是 0，骷髏一個影格都不佔（原版 Delay(0)）。
	application.gameSpeed = 0
	state.Corpses = nil
	application.combatantDown(state, 2, 0)
	if len(state.Notices) != 0 {
		t.Fatalf("speed 0 still held %+v", state.Notices)
	}
}

// entry 26 的閃光：旗標 1（"is turned" 那一類）是槽 16h（COMSPR 9）、輪（速度 + 1）次；
// 旗標 0（法術與效果的傷害，overlay-24 `14ECh`）是槽 17h（COMSPR 0Ah）、一輪再等一拍。
func TestSparklesUseTheOriginalSlotsAndLengths(t *testing.T) {
	application, state := newFoeCastApp(t)
	application.gameSpeed = 2
	application.turnedNotice(state, 2, "IS TURNED")
	turned := state.Notices[len(state.Notices)-1]
	if turned.Anim == nil || turned.Anim.Kind != animationSparkle || turned.Anim.Rounds != 3 ||
		turned.Anim.Frames[0].Block != 9 || turned.Anim.StepMilliseconds != 0x46 ||
		turned.Ticks != 3*4*0x46*60/1000 {
		t.Fatalf("turned sparkle %+v / %+v", turned, turned.Anim)
	}
	application.hurtNotice(state, 1, hurtText(state, 7, 0x09))
	hurt := state.Notices[len(state.Notices)-1]
	if hurt.Anim == nil || hurt.Anim.Rounds != 1 || hurt.Anim.Frames[0].Block != 0x0A ||
		hurt.Ticks != millisecondsToTicks(4*0x46)+application.speedDelayTicks() {
		t.Fatalf("hurt sparkle %+v / %+v", hurt, hurt.Anim)
	}
	if hurt.Text != "TAKES 7 POINTS OF DAMAGE FROM FIRE" {
		t.Fatalf("hurt text %q", hurt.Text)
	}
	// 動畫只在四格之內畫：過了 4 × 70 毫秒就是單純的停拍。
	if _, _, _, ok := hurt.Anim.frameAt(4*0x46 - 1); !ok {
		t.Fatal("the last sparkle frame is missing")
	}
	if _, _, _, ok := hurt.Anim.frameAt(4 * 0x46); ok {
		t.Fatal("the sparkle is still drawn during the beat after it")
	}
}

// overlay-24 `13DEh..14D8h`：`6777h & F7h` 挑火／寒／電／酸，**另外** `6777h & 8 == 6777h`（含 0）
// 接 "from Magic"。
func TestHurtTextFollowsTheDamageKind(t *testing.T) {
	state := &tacticalState{}
	for flags, want := range map[uint8]string{
		0x00: "TAKES 5 POINTS OF DAMAGE FROM MAGIC",
		0x08: "TAKES 5 POINTS OF DAMAGE FROM MAGIC",
		0x09: "TAKES 5 POINTS OF DAMAGE FROM FIRE",
		0x0A: "TAKES 5 POINTS OF DAMAGE FROM COLD",
		0x0C: "TAKES 5 POINTS OF DAMAGE FROM ELECTRICITY",
		0x10: "TAKES 5 POINTS OF DAMAGE FROM ACID",
	} {
		if got := hurtText(state, 5, flags); got != want {
			t.Fatalf("flags %02X: %q, want %q", flags, got, want)
		}
	}
	if got := hurtText(state, 1, 0x08); !strings.HasPrefix(got, "TAKES 1 POINT OF DAMAGE") {
		t.Fatalf("one point: %q", got)
	}
}

// 怪物的造形是 `LOAD MONSTER` 第三個引數在 CPICn 裡的那一張（overlay-03 `0507h..0529h`），
// 不是 CBODY 配預設顏色。
func TestFoeBoardIconIsTheCPICBlock(t *testing.T) {
	application := &app{}
	application.combatMonsters = []stagedMonster{{Archive: 2}, {Archive: 0}}
	application.combatMonsters[0].Spawn.Count, application.combatMonsters[0].Spawn.IconBlock = 1, 0x31
	application.combatMonsters[1].Spawn.Count, application.combatMonsters[1].Spawn.IconBlock = 1, 0x05
	application.eclArchive = 4
	foes := []boardIcon{}
	for _, monster := range application.combatMonsters {
		archive := monster.Archive
		if archive == 0 {
			archive = application.monsterArchive()
		}
		foes = append(foes, monsterBoardIcon(archive, monster.Spawn.IconBlock))
	}
	next := 0
	first := application.boardIconFor(-1, false, foes, &next)
	second := application.boardIconFor(-1, false, foes, &next)
	if !first.Monster || first.Archive != 2 || first.Body != 0x31 || !second.Monster || second.Archive != 4 || second.Body != 5 {
		t.Fatalf("foe icons %+v %+v", first, second)
	}
	if first.Colours != ([6][2]uint8{}) {
		t.Fatalf("a monster icon carries a colour table %+v; CPIC uses its own colours", first.Colours)
	}
}

// drainCombatAnimations 把「下一影格要停在一則動畫上」的影格空轉過去（spec 166）；停在一般的
// 停拍上就不動。以按鍵量東西的駕駛拿它把加了動畫之後多出來的等待扣掉，其餘影格與加動畫之前
// 一模一樣。上限只為了讓迴圈有終點。
func drainCombatAnimations(t *testing.T, application *app) {
	t.Helper()
	animating := func() bool {
		state := application.tactical
		if state == nil || state.Finished {
			return false
		}
		for _, notice := range state.Notices {
			if notice.Ticks > 0 {
				return notice.Anim != nil
			}
		}
		return false
	}
	for guard := 0; guard < 1<<20 && animating(); guard++ {
		if err := press(application, combatNoticeIdleKey); err != nil {
			t.Fatal(err)
		}
	}
	if animating() {
		t.Fatalf("a combat animation never finished: %+v", application.tactical.Notices)
	}
}
