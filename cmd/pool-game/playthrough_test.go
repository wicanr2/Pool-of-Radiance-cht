package main

import (
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 只用正常按鍵，從標題一路走到「隊伍在地圖上動了一步」。這條路徑上任何一段
// 需要測試自己塞狀態才走得通，就表示玩家也走不通——所以這裡不碰
// application.state、eventSession 或 spawn，全部靠 press。
func TestNormalKeysReachTheFirstDungeonStep(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	text := &scriptedTextKeys{scriptedKeys: scriptedKeys{}}
	application.keys = text
	step := func(what string, key ebiten.Key, chars ...rune) {
		t.Helper()
		text.scriptedKeys[key] = true
		text.chars = chars
		if err := application.Update(); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	idle := func() {
		t.Helper()
		text.chars = nil
		if err := application.Update(); err != nil {
			t.Fatalf("idle: %v", err)
		}
	}

	step("標題", ebiten.KeyEnter)
	if application.mode != modeMenu {
		t.Fatalf("title did not reach the menu, mode=%d", application.mode)
	}
	step("開始建角", ebiten.KeyC)
	for _, what := range []string{"種族", "性別", "職業", "陣營"} {
		step(what, ebiten.KeyEnter)
	}
	idle() // 骰值在第一次更新時產生，與真正的畫格路徑相同
	step("接受骰值", ebiten.KeyEnter)
	step("輸入姓名", ebiten.KeyEnter, 'H', 'E', 'R', 'O')
	step("保留肖像", ebiten.KeyK)
	step("確認造形", ebiten.KeyEnter)
	step("造形 OK", ebiten.KeyY)
	if len(application.state.CharacterLibrary) != 1 {
		t.Fatalf("character library holds %d after creation", len(application.state.CharacterLibrary))
	}
	step("加入隊伍", ebiten.KeyA)
	if len(application.state.Party) != 1 {
		t.Fatalf("party holds %d after A", len(application.state.Party))
	}
	step("開始冒險", ebiten.KeyB)
	if application.mode != modeAdventure {
		t.Fatalf("B did not enter the adventure, mode=%d status=%q", application.mode, application.statusLine)
	}

	// 開場與 34 步導覽是原版的自動流程，玩家只能按 ENTER 推進。
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			step("推進開場", ebiten.KeyEnter)
			continue
		}
		idle()
	}
	if !application.introDone {
		t.Fatalf("the opening never finished: waiting=%v tour=%v step=%d status=%q",
			application.introWaiting, application.tourActive, application.tourStep, application.statusLine)
	}

	before := application.spawn
	moved := false
	// 轉向是 45 度一格，但前進只認四個正方向，所以每次要按兩下才換到下一個
	// 可走的朝向。四個方向都試過還沒動，才算真的走不了。
	for attempt := 0; attempt < 4 && !moved; attempt++ {
		facing := application.spawn.Facing
		step("前進", ebiten.KeyArrowUp)
		t.Logf("朝向 %d 前進後 (%d,%d) 事件=%v 戰鬥=%v 狀態=%q",
			facing, application.spawn.X, application.spawn.Y,
			application.cellEventPending, application.combatActive, application.statusLine)
		if application.spawn.X != before.X || application.spawn.Y != before.Y {
			moved = true
			break
		}
		step("右轉", ebiten.KeyArrowRight)
		step("右轉", ebiten.KeyArrowRight)
	}
	if !moved {
		t.Fatalf("the party never moved from (%d,%d) facing %d: 事件=%v 狀態=%q",
			before.X, before.Y, application.spawn.Facing, application.cellEventPending, application.statusLine)
	}
	t.Log(fmt.Sprintf("走到 %+v，狀態列：%s", application.spawn, application.statusLine))

	// 存檔與讀檔也只用按鍵：F10 存、重開一份再按 L 讀回來。
	var saved poolsave.State
	application.saveState = func(state poolsave.State) error { saved = cloneSaveState(state); return nil }
	if err := press(application, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10 save=%v", err)
	}
	if saved.Campaign == nil {
		t.Fatal("F10 saved no campaign")
	}
	restored, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	restored.mode = modeMenu
	restored.loadState = func() (poolsave.State, error) { return cloneSaveState(saved), nil }
	if err := press(restored, ebiten.KeyL); err != nil {
		t.Fatal(err)
	}
	if restored.mode != modeAdventure || restored.spawn != application.spawn || len(restored.state.Party) != 1 {
		t.Fatalf("load restored mode=%d spawn=%+v party=%d", restored.mode, restored.spawn, len(restored.state.Party))
	}
	// 讀回來之後也要能繼續走，不是只把畫面切過去。
	resumed := restored.spawn
	if err := press(restored, ebiten.KeyArrowUp); err != nil {
		t.Fatal(err)
	}
	if restored.spawn.X == resumed.X && restored.spawn.Y == resumed.Y {
		t.Fatalf("the restored party could not move: %q", restored.statusLine)
	}
}

// 只用按鍵走到第一場戰鬥。路線是固定亂數種子的漫遊，所以每次都一樣：
// 隊伍會經過渡船抵達索寇要塞，觸發 `29h` 的遭遇選單，選 COMBAT 之後 ECL
// 依結果碼分支，把原版的骷髏與殭屍記錄擺上場。
func TestNormalKeysReachTheFirstCombat(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	character := poolsave.Character{Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", Abilities: [6]int{16, 10, 10, 13, 10, 10}, MaxHP: 8, CurrentHP: 8,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{character}, Party: []poolsave.Character{character}}
	application.saveState = func(poolsave.State) error { return nil }
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if !application.introDone {
		t.Fatal("the opening never finished")
	}

	random := rand.New(rand.NewSource(7))
	sawEncounter := false
	for step := 0; step < 4000; step++ {
		if application.combatActive {
			if !sawEncounter {
				t.Fatal("combat started without the encounter menu")
			}
			if !strings.Contains(application.eventText, "SKELETON") || !strings.Contains(application.eventText, "ZOMBIE") {
				t.Fatalf("staged monsters are %q", application.eventText)
			}
			if application.spawn.Map.Archive != 4 {
				t.Fatalf("combat happened on archive %d", application.spawn.Map.Archive)
			}
			return
		}
		var err error
		switch {
		case application.encounter != nil:
			sawEncounter = true
			if len(application.cellMenuOptions) != 4 || application.cellMenuOptions[0] != "COMBAT" {
				t.Fatalf("encounter menu is %v", application.cellMenuOptions)
			}
			err = press(application, ebiten.KeyEnter) // COMBAT
		case application.cellWaitingMenu, application.cellEventPending:
			err = press(application, ebiten.KeyEnter)
		default:
			if random.Intn(3) == 0 {
				err = press(application, ebiten.KeyArrowRight)
			} else {
				err = press(application, ebiten.KeyArrowUp)
			}
		}
		if err != nil {
			t.Fatalf("step %d at %+v: %v", step, application.spawn, err)
		}
	}
	t.Fatalf("no combat in 4000 steps; last position %+v, encounter seen=%v", application.spawn, sawEncounter)
}

// 戰鬥要打得完。隊伍全程按 ENTER 不還手，怪物必須自己走過來把它打倒——
// 在敵方回合加上「追最近的敵人」這條退路之前，站得遠的怪物會回報找不到
// 目標然後原地結束回合，雙方隔著二十幾格互相不動，戰鬥永遠不結束。
func TestPassiveCombatTerminates(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	party := make([]poolsave.Character, 0, 4)
	for index := 0; index < 4; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{18, 10, 10, 16, 10, 10}, MaxHP: 40, CurrentHP: 40,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	random := rand.New(rand.NewSource(7))
	for step := 0; step < 4000 && !application.combatActive; step++ {
		var err error
		switch {
		case application.encounter != nil, application.cellWaitingMenu, application.cellEventPending:
			err = press(application, ebiten.KeyEnter)
		default:
			if random.Intn(3) == 0 {
				err = press(application, ebiten.KeyArrowRight)
			} else {
				err = press(application, ebiten.KeyArrowUp)
			}
		}
		if err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	if !application.combatActive {
		t.Fatal("never reached combat")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatalf("tactical state absent: %q", application.statusLine)
	}
	for tick := 0; tick < 40000; tick++ {
		state := application.tactical
		if state == nil || state.Finished {
			break
		}
		key := ebiten.KeyEnter
		if state.Prompt {
			key = ebiten.KeyY
		}
		if err := press(application, key); err != nil {
			t.Fatalf("combat tick %d: %v", tick, err)
		}
	}
	if application.tactical != nil {
		t.Fatalf("combat never ended: round %d, status %q, foe log %q",
			application.tactical.Round, application.tactical.Status, application.tactical.FoeLog)
	}
	if !strings.Contains(application.statusLine, "defeated") {
		t.Fatalf("a party that never fought back ended with %q", application.statusLine)
	}
}

// 主動作戰的一場也要收得了尾，而且隊伍真的殺得死怪物。
//
// 這一條擋的是三個已經踩過的坑：走進同伴那一格會砍同伴、攻擊不消耗行動、
// 以及怪物只肯往方向表要的那一格走、撞到地形就原地不動。任何一個回來，
// 這場戰鬥就會變成永遠打不完。
//
// **不斷言誰贏**：這一隊是空手的一級戰士，勝負取決於怪物強度，不是規則
// 對不對。AC 那條鏈有沒有接上另外驗（見底下對 ArmorClass 的斷言）。
func TestActiveCombatTerminatesAndKillsFoes(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, poolsave.Character{Name: string(rune('A' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{18, 10, 10, 16, 10, 10}, MaxHP: 60, CurrentHP: 60,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 20000 && !application.introDone; tick++ {
		if application.introWaiting || application.tourPage >= 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	random := rand.New(rand.NewSource(7))
	for step := 0; step < 4000 && !application.combatActive; step++ {
		switch {
		case application.encounter != nil, application.cellWaitingMenu, application.cellEventPending:
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
		default:
			key := ebiten.KeyArrowUp
			if random.Intn(3) == 0 {
				key = ebiten.KeyArrowRight
			}
			if err := press(application, key); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !application.combatActive {
		t.Fatal("never reached combat")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatalf("tactical state absent: %q", application.statusLine)
	}
	// 敏捷 16 的 AC 調整是 +2，內部值 52。規則寫在 gamepack 卻沒接進戰鬥時，
	// 這裡會停在建角的 50——而戰鬥報表上看不出差別，只會讓隊伍挨打。
	for index := 1; index <= len(application.state.Party) && index < len(application.tactical.ArmorClass); index++ {
		if !application.tactical.Friendly[index] {
			continue
		}
		if got := application.tactical.ArmorClass[index]; got != 52 {
			t.Fatalf("party slot %d internal AC %d, want 52", index, got)
		}
	}
	foesAtStart := 0
	for index := 1; index < len(application.tactical.Roster); index++ {
		if !application.tactical.Friendly[index] {
			foesAtStart++
		}
	}
	fewestFoes := foesAtStart
	sameCell := 0
	for tick := 0; tick < 20000; tick++ {
		state := application.tactical
		if state == nil || state.Finished {
			break
		}
		standing := 0
		for index := 1; index < len(state.Roster); index++ {
			if !state.Friendly[index] && state.Roster[index].FootprintClass != 0 {
				standing++
			}
		}
		if standing < fewestFoes {
			fewestFoes = standing
		}
		if state.Prompt {
			if err := press(application, ebiten.KeyY); err != nil {
				t.Fatal(err)
			}
			continue
		}
		mover := state.Mover
		if mover == 0 || int(mover) >= len(state.Friendly) || !state.Friendly[mover] {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		target, ok := state.nearestOpposing(mover)
		if !ok {
			break
		}
		from, to := state.Roster[mover], state.Roster[target]
		chosen, chosenDistance := -1, chebyshev(from.X, from.Y, to.X, to.Y)+1
		for direction := 0; direction < 8; direction++ {
			x, y, err := combat.AdvanceTacticalCoordinate(from.X, from.Y, uint8(direction))
			if err != nil {
				continue
			}
			if d := chebyshev(x, y, to.X, to.Y); d < chosenDistance {
				chosen, chosenDistance = direction, d
			}
		}
		if chosen < 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := press(application, tacticalStepKeys[chosen]); err != nil {
			t.Fatalf("combat tick %d: %v", tick, err)
		}
		after := application.tactical
		if after == nil || after.Finished {
			break
		}
		if after.Mover == mover && after.Roster[mover].X == from.X && after.Roster[mover].Y == from.Y {
			// 動不了也打不到就結束這一回合。攻擊本身已經會結束回合。
			sameCell++
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
		} else {
			sameCell = 0
		}
	}
	if application.tactical != nil {
		t.Fatalf("combat never ended: round %d, status %q, foe log %q",
			application.tactical.Round, application.tactical.Status, application.tactical.FoeLog)
	}
	// 固定操作、固定種子，實測十二隻裡放倒三隻之後隊伍就被打垮。門檻放在
	// 三隻，只證明「攻擊真的造成死亡」；打不贏是因為裝備還沒接進戰鬥，
	// 隊伍身上等於沒有盔甲也沒有武器，那是另一條 worklist。
	if foesAtStart-fewestFoes < 3 {
		t.Fatalf("only %d of %d foes went down before the fight ended", foesAtStart-fewestFoes, foesAtStart)
	}
}
