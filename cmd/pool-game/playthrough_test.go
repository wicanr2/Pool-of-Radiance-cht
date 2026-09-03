package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gametext"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
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

	// 走出這一區的那一步會留一段文字（spec 100：起點 (0,4) 就在西邊界上，
	// 往西一步是穿過城門）。玩家會先按 RETURN 把它讀完，測試也照做——
	// 對話開著的時候存不了檔。
	for tick := 0; tick < 64 && (application.cellEventPending || application.cellWaitingMenu); tick++ {
		step("讀完文字", ebiten.KeyEnter)
	}

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

	// 逐格走到第一場架。**不再釘住是哪一場**：世界接上「走出這一區」之後
	// （spec 100），第一場遇到的架跟路線有關，原本釘的索寇要塞骷髏／殭屍是
	// 舊世界的路線走出來的。這一條要驗的是**遭遇選單先出現、然後才進戰鬥**，
	// 那條管線與是哪一場無關。
	sawEncounter, menuShape := false, []string(nil)
	reached := walkThisAreaUntil(t, application, 60000, func() bool {
		if application.encounter != nil && !sawEncounter {
			sawEncounter = true
			menuShape = append([]string(nil), application.cellMenuOptions...)
		}
		return application.combatActive
	})
	if !reached {
		t.Fatalf("走完整區都沒打到架；最後在 %+v，遭遇選單出現過=%v",
			application.spawn, sawEncounter)
	}
	if !sawEncounter {
		t.Fatal("combat started without the encounter menu")
	}
	if len(menuShape) != 4 || menuShape[0] != "COMBAT" {
		t.Fatalf("encounter menu is %v", menuShape)
	}
	t.Logf("第一場架在 GEO%d/%d (%d,%d)：%q",
		application.spawn.Map.Archive, application.spawn.Map.BlockID,
		application.spawn.X, application.spawn.Y, application.eventText)
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
	for step := 0; step < 30000 && !application.combatActive; step++ {
		var err error
		switch {
		case application.programManaging:
			// 地圖上的隊伍管理畫面吃掉方向鍵，原版按 B 回地圖。
			err = press(application, ebiten.KeyB)
		case application.treasureActive && len(application.cellMenuOptions) != 0:
			// 寶物選單停在 View 上，一直按 Enter 只會一直看。
			if want := treasureMenuChoice(application.cellMenuOptions); want != application.cellMenuCursor {
				err = press(application, ebiten.KeyArrowRight)
			} else {
				err = press(application, ebiten.KeyEnter)
			}
		case application.programManaging:
			err = press(application, ebiten.KeyB)
		case application.shopActive:
			err = press(application, ebiten.KeyEscape)
		case application.encounter != nil, application.cellWaitingMenu, application.cellEventPending:
			if key, ok := menuEscapeKey(application); ok {
				err = press(application, key)
			} else {
				err = press(application, ebiten.KeyEnter)
			}
		default:
			if random.Intn(3) == 0 || forwardWouldLeaveTheArea(application) {
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
		t.Fatalf("never reached combat: GEO%d/%d (%d,%d) mode=%d 管理=%v 寶物=%v 輸入=%v 選單=%v 狀態=%q",
			application.spawn.Map.Archive, application.spawn.Map.BlockID,
			application.spawn.X, application.spawn.Y, application.mode,
			application.programManaging, application.treasureActive,
			application.eclInput != nil, application.cellMenuOptions,
			application.statusLine)
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
// **不斷言誰贏**：這一隊完全沒有裝備，空手的一級戰士打不過六隻骷髏加六隻
// 殭屍是合理的結果。「裝備好的隊伍打得贏」由
// TestAnEquippedPartyWinsTheFirstFight 驗。
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
	// 骰子固定：不然「放倒幾隻」每次都不一樣，門檻只能訂得很鬆或很脆。
	application.roller = diceRoller{random: rand.New(rand.NewSource(3))}
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
	// 逐格走到第一場架。原本靠固定種子亂走碰運氣，世界一變大就走去別的地方。
	walkThisAreaUntil(t, application, 60000, func() bool { return application.combatActive })
	if !application.combatActive {
		t.Fatalf("never reached combat: GEO%d/%d (%d,%d) mode=%d 管理=%v 寶物=%v 輸入=%v 選單=%v 狀態=%q",
			application.spawn.Map.Archive, application.spawn.Map.BlockID,
			application.spawn.X, application.spawn.Y, application.mode,
			application.programManaging, application.treasureActive,
			application.eclInput != nil, application.cellMenuOptions,
			application.statusLine)
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
	for tick := 0; tick < 200000; tick++ {
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
	// 迴圈可能停在「這一場已經判定結束、但前端還沒收尾」那一刻，
	// 所以先讓它收完再判。
	for tick := 0; tick < 64 && application.tactical != nil; tick++ {
		key := ebiten.KeyEnter
		if application.tactical.Prompt {
			// 敵方清光之後那一次問的是「還要不要繼續打」（spec 062）。
			key = ebiten.KeyY
			if application.tactical.sideCounts().Foes == 0 {
				key = ebiten.KeyN
			}
		}
		if err := press(application, key); err != nil {
			t.Fatal(err)
		}
	}
	if application.tactical != nil {
		t.Fatalf("combat never ended: round %d, status %q, foe log %q",
			application.tactical.Round, application.tactical.Status, application.tactical.FoeLog)
	}
	// 固定操作、固定骰子。門檻只要證明「攻擊真的造成死亡」——空手的隊伍
	// 打不贏是預期結果，能不能贏由 TestAnEquippedPartyWinsTheFirstFight 驗。
	if foesAtStart-fewestFoes < 2 {
		t.Fatalf("only %d of %d foes went down before the fight ended", foesAtStart-fewestFoes, foesAtStart)
	}
}

// 裝備好的隊伍打得贏索寇要塞第一場，而且戰後腳本會續跑。
//
// 這是主線可破關的實測門檻：規則對不對，看的不是單元測試綠不綠，而是
// 一支拿得動刀、穿得起甲的隊伍能不能實際打完第一場遭遇並走下去。
//
// 三件事各自都會讓這條路斷掉，而且斷得很安靜：
//   - 武器挑錯（拿戒指當武器，傷害骰 0d0）——戰鬥照跑，只是永遠打不死人。
//   - AC 沒接裝備——隊伍挨打的機率差 25% 以上。
//   - 清光敵人之後不按 N——原版問「還要繼續嗎」，答 Y 會一直空轉。
func TestAnEquippedPartyWinsTheFirstFight(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 骰子固定，否則「贏了」只是這一次的運氣。
	application.roller = diceRoller{random: rand.New(rand.NewSource(11))}

	kit, err := premadeReadiedKit()
	if err != nil {
		t.Skipf("original item records unavailable: %v", err)
	}
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		own := make([]poolsave.Item, 0, len(kit))
		for _, item := range kit {
			own = append(own, poolsave.Item{Name: item.Name, Raw: append([]byte(nil), item.Raw...)})
		}
		party = append(party, poolsave.Character{Name: string(rune('A' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{18, 10, 10, 16, 10, 10}, ExceptionalStrength: 100,
			MaxHP: 60, CurrentHP: 60, PortraitHead: 1, PortraitBody: 1, IconSize: 1,
			Inventory: own})
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
	for step := 0; step < 30000 && !application.combatActive; step++ {
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
		t.Fatalf("never reached combat: GEO%d/%d (%d,%d) mode=%d 管理=%v 寶物=%v 輸入=%v 選單=%v 狀態=%q",
			application.spawn.Map.Archive, application.spawn.Map.BlockID,
			application.spawn.X, application.spawn.Y, application.mode,
			application.programManaging, application.treasureActive,
			application.eclInput != nil, application.cellMenuOptions,
			application.statusLine)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatalf("tactical state absent: %q", application.statusLine)
	}
	// 武器與盔甲要真的變成戰鬥數值：長劍 +4 的 1d8、敏捷 16 加板甲 +2 加盾 +2。
	if got := application.tactical.Damage[1]; got.Count == 0 || got.Sides == 0 {
		t.Fatalf("party damage dice %v: the readied weapon never reached combat", got)
	}
	if got := application.tactical.ArmorClass[1]; got != 64 {
		t.Fatalf("party internal AC %d, want 64", got)
	}
	foesAtStart := 0
	for index := 1; index < len(application.tactical.Roster); index++ {
		if !application.tactical.Friendly[index] {
			foesAtStart++
		}
	}
	if err := driveTacticalCombat(t, application, 40000); err != nil {
		t.Fatal(err)
	}
	if application.tactical != nil {
		t.Fatalf("combat never ended: round %d, status %q",
			application.tactical.Round, application.tactical.Status)
	}
	if strings.Contains(application.statusLine, "defeated") {
		t.Fatalf("an equipped party lost the first fight: %q", application.statusLine)
	}
	// 勝利之後 finishCombat 會續跑戰後腳本；跑錯會回 error，跑不到會留著遭遇旗標。
	if application.combatActive {
		t.Fatal("the encounter is still staged after the fight")
	}
	// 經驗值要真的發下去（spec 097）。零就代表 finishCombat 沒走到發放那一段——
	// 上面那些斷言全部通過也看不出來。
	for _, member := range application.state.Party {
		if member.Experience == 0 {
			t.Fatalf("%s 打贏了卻沒拿到經驗值", member.Name)
		}
	}
	t.Logf("索寇要塞第一場：%d 隻怪物，隊伍勝出，每人 %d 點經驗值",
		foesAtStart, application.state.Party[0].Experience)
}

// premadeReadiedKit 借原版預設人物 chrdatd2 身上穿戴中的東西當裝備：
// 長劍 +4、板甲 +2、盾 +2 與一枚戒指。用真記錄才測得到「型別索引查得到表」。
func premadeReadiedKit() ([]poolsave.Item, error) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "workplace", "oracle", "dos", "chrdatd2.itm"))
	if err != nil {
		return nil, err
	}
	var kit []poolsave.Item
	for offset := 0; offset+63 <= len(raw); offset += 63 {
		record := append([]byte(nil), raw[offset:offset+63]...)
		if record[itemReadyOffset] == 0 {
			continue
		}
		length := int(record[0])
		kit = append(kit, poolsave.Item{Name: string(record[1 : 1+length]), Raw: record})
	}
	if len(kit) == 0 {
		return nil, errors.New("chrdatd2 has no readied items")
	}
	return kit, nil
}

// driveTacticalCombat 是「一個會繞路的玩家」：每個我方回合把八個方向依
// 「走完之後離目標多近」排序，一個一個試到真的動了為止。只試最好的那一個
// 等於撞牆就放棄，那樣量到的是驅動程式的極限，不是遊戲的。
func driveTacticalCombat(t *testing.T, application *app, budget int) error {
	t.Helper()
	for tick := 0; tick < budget; tick++ {
		state := application.tactical
		if state == nil || state.Finished {
			return nil
		}
		if state.Prompt {
			// 敵方清光時原版會問「還要繼續嗎」，N 才是收尾。
			if err := press(application, ebiten.KeyN); err != nil {
				return err
			}
			continue
		}
		mover := state.Mover
		if mover == 0 || int(mover) >= len(state.Friendly) || !state.Friendly[mover] {
			if err := press(application, ebiten.KeyEnter); err != nil {
				return err
			}
			continue
		}
		target, ok := state.nearestOpposing(mover)
		if !ok {
			// 場上沒有敵人了：結束回合，讓 endRound 去問「還要繼續嗎」。
			if err := press(application, ebiten.KeyEnter); err != nil {
				return err
			}
			continue
		}
		from, to := state.Roster[mover], state.Roster[target]
		// 依「繞得過去的實際步數」排序，不是直線距離。盤面是斜的又多牆
		// （spec 060），直線距離會把人帶進死角然後在那裡來回。
		distance := tacticalStepDistances(state.Grid, state.Classes, to.X, to.Y)
		order := make([]int, 0, len(tacticalStepKeys))
		for direction := range tacticalStepKeys {
			if _, _, err := combat.AdvanceTacticalCoordinate(from.X, from.Y, uint8(direction)); err == nil {
				order = append(order, direction)
			}
		}
		sort.SliceStable(order, func(i, j int) bool {
			xi, yi, _ := combat.AdvanceTacticalCoordinate(from.X, from.Y, uint8(order[i]))
			xj, yj, _ := combat.AdvanceTacticalCoordinate(from.X, from.Y, uint8(order[j]))
			return distance[tacticalCellKey(xi, yi)] < distance[tacticalCellKey(xj, yj)]
		})
		moved := false
		for _, direction := range order {
			if err := press(application, tacticalStepKeys[direction]); err != nil {
				return err
			}
			after := application.tactical
			if after == nil || after.Finished {
				return nil
			}
			if after.Mover != mover || after.Roster[mover].X != from.X || after.Roster[mover].Y != from.Y {
				moved = true
				break
			}
		}
		if !moved {
			if err := press(application, ebiten.KeyEnter); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("combat did not finish within %d ticks", budget)
}

// 只用按鍵走到 `38h PROGRAM` 那一格，開起隊伍管理，再回到地圖繼續走。
//
// 這條擋的是兩件事：opcode 硬失敗讓探索整個停住，以及「回地圖」誤走成
// 「開始冒險」——後者會把開場整個重跑一次，隊伍被丟回起點，而畫面上看起來
// 只是「怎麼又在講故事」。
func TestNormalKeysReachThePartyManagementCell(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.roller = diceRoller{random: rand.New(rand.NewSource(5))}
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
	// 逐格走到 `PROGRAM` 那一格（ECL3／block 11）。原本靠種子 19 亂走碰運氣，
	// 世界一變大就走去別的地方；逐格走不依賴種子。
	//
	// **不繞出這一區**：那一格在起始區裡，而 `escapeKeyForWalk` 會在隊伍管理
	// 畫面按 B 回地圖——所以停止條件要在按鍵之前就檢查到。
	reached := walkThisAreaUntil(t, application, 60000, func() bool {
		return application.programManaging
	})
	if !reached {
		t.Fatal("never reached the PROGRAM cell")
	}
	if application.mode != modeMenu {
		t.Fatalf("PROGRAM left the app in mode %d, want the party management screen", application.mode)
	}
	where := application.spawn
	if err := press(application, ebiten.KeyB); err != nil {
		t.Fatal(err)
	}
	if application.programManaging {
		t.Fatal("B did not leave the party management screen")
	}
	if application.mode != modeAdventure {
		t.Fatalf("B left the app in mode %d, want the map", application.mode)
	}
	// 回地圖不是重開冒險：位置與朝向都要留在原地。
	if application.spawn.Map != where.Map || application.spawn.X != where.X || application.spawn.Y != where.Y {
		t.Fatalf("the party moved from %v to %v while managing", where, application.spawn)
	}
	if application.introWaiting || application.tourActive {
		t.Fatal("returning to the map restarted the opening")
	}
	// 回來之後還走得動。
	for step := 0; step < 200; step++ {
		if application.cellEventPending || application.cellWaitingMenu {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			t.Fatalf("could not walk after party management: %v", err)
		}
	}
}

// forwardWouldLeaveTheArea 說往前一步會不會走出這一區（spec 100）。
//
// 只在起始區裡找架打的測試要靠它擋住那一步：世界接上「走出這一區」之後，
// 起點 (0,4) 往西一步就出城了，固定種子的路線會走去別的地方。這些測試量的
// 是戰鬥、記憶法術、裝備，不是地理，所以留在原地比重新挑種子誠實。
func forwardWouldLeaveTheArea(application *app) bool {
	dx, dy := 0, 0
	switch application.spawn.Facing {
	case 0:
		dy = -1
	case 1:
		dx = 1
	case 2:
		dy = 1
	case 3:
		dx = -1
	}
	x, y := int(application.spawn.X)+dx, int(application.spawn.Y)+dy
	return x < 0 || x >= geometry.Width || y < 0 || y >= geometry.Height
}

// menuEscapeKey 給「亂走找架」的迴圈用：選單裡有 Exit 就走過去按下確定。
//
// 世界變大之後這些迴圈會逛到神廟、商店這些帶選單的地方。一律按 Enter 會停在
// 第一項（像神廟的「治療失明」），而那一項還沒接就會卡在原地。
func menuEscapeKey(application *app) (ebiten.Key, bool) {
	options := application.cellMenuOptions
	if len(options) < 2 {
		return 0, false
	}
	for index, option := range options {
		if !strings.EqualFold(option, "Exit") {
			continue
		}
		if application.cellMenuCursor != index {
			return ebiten.KeyArrowRight, true
		}
		return ebiten.KeyEnter, true
	}
	return 0, false
}

// walkThisAreaUntil 在**這一區之內**逐格走，直到 stop 成立或走完整區。
//
// 原本這些測試靠固定亂數種子亂走碰運氣。世界接上「走出這一區」之後，同一個
// 種子就走去別的地方；每修一次別處，路線又會變一次——挑種子挑不完。逐格走
// 不依賴種子：廣度優先找最近一格沒踩過的走過去，不繞出這一區。
func walkThisAreaUntil(t *testing.T, application *app, budget int, stop func() bool) bool {
	t.Helper()
	walked := map[[2]int]bool{}
	var plan []exploreStep
	for step := 0; step < budget; step++ {
		if stop() {
			return true
		}
		walked[[2]int{int(application.spawn.X), int(application.spawn.Y)}] = true
		if key, busy := escapeKeyForWalk(application); busy {
			plan = nil
			if err := press(application, key); err != nil {
				t.Fatalf("第 %d 步：%v", step, err)
			}
			continue
		}
		if len(plan) == 0 {
			if plan = planInsideThisArea(application, walked); len(plan) == 0 {
				return stop()
			}
		}
		want := plan[0]
		if application.spawn.Facing != want.facing {
			key := ebiten.KeyArrowRight
			if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
				key = ebiten.KeyArrowLeft
			}
			if err := press(application, key); err != nil {
				t.Fatalf("第 %d 步：%v", step, err)
			}
			continue
		}
		before := application.spawn
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			t.Fatalf("第 %d 步：%v", step, err)
		}
		if application.spawn.X == before.X && application.spawn.Y == before.Y {
			plan = nil
			continue
		}
		plan = plan[1:]
	}
	return stop()
}

// escapeKeyForWalk 回「現在有畫面擋著嗎、該按哪一鍵離開」。
func escapeKeyForWalk(application *app) (ebiten.Key, bool) {
	switch {
	case application.programManaging:
		return ebiten.KeyB, true
	case application.shopActive:
		return ebiten.KeyEscape, true
	case application.tactical != nil:
		if application.tactical.Prompt {
			if application.tactical.sideCounts().Foes == 0 {
				return ebiten.KeyN, true
			}
			return ebiten.KeyY, true
		}
		return ebiten.KeyEnter, true
	case application.treasureActive && len(application.cellMenuOptions) != 0:
		if want := treasureMenuChoice(application.cellMenuOptions); want != application.cellMenuCursor {
			return ebiten.KeyArrowRight, true
		}
		return ebiten.KeyEnter, true
	case application.encounter != nil, application.cellWaitingMenu,
		application.cellEventPending, application.templeActive,
		application.mode != modeAdventure:
		if key, ok := menuEscapeKey(application); ok {
			return key, true
		}
		return ebiten.KeyEnter, true
	}
	return 0, false
}

// planInsideThisArea 找最近一格沒踩過的，路徑不跨出這一區的邊界。
func planInsideThisArea(application *app, walked map[[2]int]bool) []exploreStep {
	type node struct{ x, y int }
	start := node{int(application.spawn.X), int(application.spawn.Y)}
	from := map[node]node{start: start}
	via := map[node]uint8{}
	queue := []node{start}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current != start && !walked[[2]int{current.x, current.y}] {
			steps := []exploreStep{}
			for cursor := current; cursor != start; cursor = from[cursor] {
				steps = append([]exploreStep{{facing: via[cursor]}}, steps...)
			}
			return steps
		}
		for facing := 0; facing < 4; facing++ {
			if !application.initialMap.Grid.CanMoveDungeonWrapped(current.x, current.y, facing*2) {
				continue
			}
			x := current.x + exploreDeltas[facing][0]
			y := current.y + exploreDeltas[facing][1]
			if x < 0 || x >= geometry.Width || y < 0 || y >= geometry.Height {
				continue
			}
			next := node{x: x, y: y}
			if _, seen := from[next]; seen {
				continue
			}
			from[next], via[next] = current, uint8(facing)
			queue = append(queue, next)
		}
	}
	return nil
}

// slumsCommissionApp 起一個站在貧民窟遭遇格上的遊戲，`fights` 是要真的打贏
// 幾場。回傳 app 與 session，供兩條測試共用（一條打滿 25 場，一條不打，
// 當負對照）。
func slumsCommissionApp(t *testing.T, fights int) (*app, *eclvm.BlockSession, []uint16) {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9E5D)
	if err != nil {
		t.Fatal(err)
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	// 跨封存檔的 `NEWECL` 要靠 catalog resolver，正式遊戲是在這裡裝的；
	// 少了它，走出貧民窟時會報「目標區塊不存在」。
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	// 肖像與戰鬥圖示要填成合法值、人也要在角色庫裡：交差時的 Share 會存檔，
	// 而存檔會驗這幾件事。
	hero := poolsave.Character{
		Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", MaxHP: 40, CurrentHP: 40,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1,
	}
	application.state.Party = []poolsave.Character{hero}
	application.state.CharacterLibrary = []poolsave.Character{hero}
	application.spawn = gamepack.Spawn{
		Map: gamepack.MapKey{Archive: 2, BlockID: 20}, X: 3, Y: 4, Facing: 2,
	}

	progress := make([]uint16, 0, fights)
	for fight := 1; fight <= fights; fight++ {
		// 每一場都從遭遇那一格的入口重新跑起——原版是隊伍再走進去一次，
		// 這裡直接回到同一個入口，因為要測的是旗標與戰後腳本，不是走位。
		if err := session.Machine().SetPC(0x9E5D - 0x9900); err != nil {
			t.Fatalf("第 %d 場：%v", fight, err)
		}
		result, err := session.RunUntilEvent(4096, nil, true)
		if err != nil {
			t.Fatalf("第 %d 場擺場失敗：%v", fight, err)
		}
		if err := application.consumeInitialSearch(result); err != nil {
			t.Fatalf("第 %d 場擺場失敗：%v", fight, err)
		}
		if !application.combatActive {
			t.Fatalf("第 %d 場沒有擺出遭遇，事件文字是 %q", fight, application.eventText)
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatalf("第 %d 場進不了戰術盤：%v", fight, err)
		}
		state := application.tactical
		if state == nil {
			t.Fatalf("第 %d 場 ENTER 沒有進戰術盤", fight)
		}
		for index := 1; index < len(state.Roster); index++ {
			if !state.Friendly[index] {
				state.Roster[index].FootprintClass = 0
				state.Scores[index] = 0
				state.States[index] = combat.DyingState
			}
		}
		state.Mover, state.Prompt = 0, false
		state.endRound(application.rollDice)
		if !state.Prompt {
			t.Fatalf("第 %d 場清光敵人沒有跳出續戰詢問", fight)
		}
		if err := press(application, ebiten.KeyN); err != nil {
			t.Fatalf("第 %d 場收尾失敗：%v", fight, err)
		}
		if application.combatActive {
			t.Fatalf("第 %d 場打完之後遭遇還掛著", fight)
		}
		progress = append(progress, session.Machine().Memory[0x4ABB])
	}
	return application, session, progress
}

// handInAtCityHall 把隊伍從貧民窟送回城區、進市政廳、走到職員面前，
// 回傳最後看到的文字。
//
// 走出貧民窟走的是原版的路：入口 0 在 `DS:6DD5h` 非零時依朝向挑鄰居，
// 面向東走 archive 3 回城區（spec 100）。門口那一步是 spec 025／026 已經
// 釘住的 `(3,4)` 往東進到 `(4,4)`，script block 換成 8。
func handInAtCityHall(t *testing.T, application *app, session *eclvm.BlockSession) string {
	t.Helper()
	machine := session.Machine()
	machine.Memory[0xC04D] = 1
	machine.Memory[0x6DD5] = 1
	if err := session.SetEntry(0); err != nil {
		t.Fatal(err)
	}
	result, err := session.RunUntilEvent(4096, nil, true)
	if err != nil {
		t.Fatalf("走出貧民窟：%v", err)
	}
	if _, err := application.consumeInitialTransitionResources(result); err != nil {
		t.Fatalf("走出貧民窟：%v", err)
	}
	if application.eclArchive != 3 || session.CurrentBlockID() != 0 {
		t.Fatalf("走出貧民窟之後停在 ecl%d/%d，應該是 ecl3/0",
			application.eclArchive, session.CurrentBlockID())
	}
	application.spawn = gamepack.Spawn{
		Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 3, Y: 4, Facing: 1,
	}
	if err := press(application, ebiten.KeyArrowUp); err != nil {
		t.Fatalf("進市政廳：%v", err)
	}
	if session.CurrentBlockID() != 8 {
		t.Fatalf("進門之後停在區塊 %d，應該是 8", session.CurrentBlockID())
	}
	for step := 0; step < 40; step++ {
		if application.treasureActive {
			// 獎賞服務開起來就停下來，交給呼叫端決定收不收——
			// 在這裡亂按 Enter 只會在 View 與 Return 之間來回。
			break
		}
		if application.cellEventPending || application.cellWaitingMenu {
			if err := press(application, ebiten.KeyEnter); err != nil {
				break
			}
			continue
		}
		if int(application.spawn.X) == 5 && int(application.spawn.Y) == 5 {
			break
		}
		plan := planToCells(application, 0, func(x, y int) bool { return x == 5 && y == 5 })
		if len(plan) == 0 {
			break
		}
		want := plan[0]
		if application.spawn.Facing != want.facing {
			key := ebiten.KeyArrowRight
			if (int(want.facing)-int(application.spawn.Facing)+4)%4 == 3 {
				key = ebiten.KeyArrowLeft
			}
			if err := press(application, key); err != nil {
				break
			}
			continue
		}
		if err := press(application, ebiten.KeyArrowUp); err != nil {
			break
		}
	}
	for step := 0; step < 20; step++ {
		if application.treasureActive || (!application.cellEventPending && !application.cellWaitingMenu) {
			break
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			break
		}
	}
	return strings.TrimSpace(application.eventText)
}

// selectMenuOption 把游標移到指定的選項再按 Enter。找不到就回 false。
func selectMenuOption(t *testing.T, application *app, want string) error {
	t.Helper()
	for guard := 0; guard < 12; guard++ {
		if len(application.cellMenuOptions) == 0 {
			return fmt.Errorf("選單是空的")
		}
		if application.cellMenuOptions[application.cellMenuCursor] == want {
			return press(application, ebiten.KeyEnter)
		}
		if err := press(application, ebiten.KeyArrowRight); err != nil {
			return err
		}
	}
	return fmt.Errorf("選單裡沒有 %q：%v", want, application.cellMenuOptions)
}

// collectCityHallReward 收下市政廳的獎賞：主選單挑 Share 把錢分給隊伍，
// 再挑 Exit，腳本才會往下跑到 `9F5Ah` 把槽清成 `FFh`。
func collectCityHallReward(t *testing.T, application *app) {
	t.Helper()
	if !application.treasureActive {
		t.Fatal("獎賞服務沒有開起來")
	}
	if err := selectMenuOption(t, application, "Share"); err != nil {
		t.Fatalf("按下 Share：%v（cursor=%d 選項 %v）",
			err, application.cellMenuCursor, application.cellMenuOptions)
	}
	if err := selectMenuOption(t, application, "Exit"); err != nil {
		t.Fatalf("按下 Exit：%v（選項 %v）", err, application.cellMenuOptions)
	}
	// 還有東西沒拿的話會問一次「真的要留在這裡嗎」。
	if application.treasureActive && len(application.cellMenuOptions) != 0 &&
		application.cellMenuOptions[0] == "Yes" {
		if err := selectMenuOption(t, application, "Yes"); err != nil {
			t.Fatalf("離開獎賞服務的確認選單：%v", err)
		}
	}
	for step := 0; step < 20; step++ {
		if !application.cellEventPending && !application.cellWaitingMenu {
			break
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			break
		}
	}
}

// 貧民窟那一條委任的完整迴圈：打 25 場、走回城區、進市政廳交差、拿到報酬。
//
// 先前只有兩種測試：進度 helper 的直接呼叫（第 24／25 次臨界）與**一場**
// 真實戰鬥續跑腳本。兩者都不證明「委任做得完」——中間任何一場卡住、旗標少加
// 一次、戰後腳本沒回到遭遇入口、跨封存檔換不過去、或交差不認帳，都會在這裡
// 變紅。負對照在 TestCityHallPaysNothingBeforeTheCommissionIsDone。
func TestTwentyFiveRealSlumsWinsEarnTheCityHallReward(t *testing.T) {
	application, session, progress := slumsCommissionApp(t, 25)
	for fight, got := range progress {
		want := uint16(fight + 1)
		if want == 25 {
			want = 0xFE
		}
		if got != want {
			t.Fatalf("第 %d 場之後 4ABB=%d，應該是 %d（整條進度是 %v）",
				fight+1, got, want, progress)
		}
	}
	machine := session.Machine()
	text := handInAtCityHall(t, application, session)
	// 進度旗標要跟著過來——它在 `4900h..4CFFh` 那一塊，不隨換區塊清掉
	// （spec 106）。跟丟的話交差就永遠不會觸發。
	if got := machine.Memory[0x4ABB]; got != 0xFE {
		t.Fatalf("交差時 4ABB=%d，應該還是 FEh", got)
	}
	if machine.Memory[0x4AC1] != 1 {
		t.Errorf("交差之後 4AC1=%d，應該是 1", machine.Memory[0x4AC1])
	}
	if !application.treasureActive {
		t.Fatalf("交差之後獎賞服務沒有開起來，最後看到的文字是 %q", text)
	}
	// 金額不是「有就好」：ECL3/8 的四張獎賞表（`B5EDh`／`B604h`／`B61Bh`／
	// `B632h`）在槽 21 的原始位元組是 `FA`、`32`、`00`、`01` ＝ 250、50、0、1，
	// 由 `9F28h TREASURE` 的第 4..7 欄（金、白金、寶石、首飾）送出。
	if want := ([7]uint32{3: 250, 4: 50, 6: 1}); application.state.PooledMoney != want {
		t.Errorf("待分的獎賞是 %v，原版四張表在槽 21 給的是 %v",
			application.state.PooledMoney, want)
	}
	t.Logf("交差拿到 %v（4ABB=%d 4AC1=%d）", application.state.PooledMoney,
		machine.Memory[0x4ABB], machine.Memory[0x4AC1])

	// 收下獎賞：腳本要走完獎賞選單與戰利品服務，`9F5Ah` 的
	// `SAVE TABLE FF` 才會把槽清掉，這一條委任才算真的結案。
	collectCityHallReward(t, application)
	if got := machine.Memory[0x4ABB]; got != 0xFF {
		t.Errorf("收下獎賞之後 4ABB=%d，應該是 FFh（`9F5Ah` 的 SAVE TABLE）", got)
	}
	member := application.state.Party[0]
	gold := member.Money[pooltreasure.Gold]
	platinum := member.Money[pooltreasure.Platinum]
	if gold != 250 || platinum != 50 {
		t.Errorf("錢沒有進到角色身上：金 %d 白金 %d，應該是 250／50（錢包 %v）",
			gold, platinum, member.Money)
	}
	t.Logf("結案：4ABB=%d 4AC1=%d 錢包 %v",
		machine.Memory[0x4ABB], machine.Memory[0x4AC1], member.Money)
}

// 負對照：一場都沒打就去交差，市政廳不該給錢。
//
// 沒有這一條的話，上面那個「拿到 Gold」證不了因果——職員本來就會講話，
// 而任何一段文字裡都可能有 Gold。
func TestCityHallPaysNothingBeforeTheCommissionIsDone(t *testing.T) {
	application, session, _ := slumsCommissionApp(t, 0)
	machine := session.Machine()
	if got := machine.Memory[0x4ABB]; got != 0 {
		t.Fatalf("還沒打就已經 4ABB=%d", got)
	}
	text := handInAtCityHall(t, application, session)
	if application.treasureActive || application.state.PooledMoney != ([7]uint32{}) {
		t.Errorf("委任還沒做完就給了報酬：%q（待分 %v）", text, application.state.PooledMoney)
	}
	if machine.Memory[0x4AC1] != 0 {
		t.Errorf("委任還沒做完 4AC1 就變成 %d", machine.Memory[0x4AC1])
	}
	t.Logf("沒做完的時候看到 %q（4AC1=%d）", text, machine.Memory[0x4AC1])
}

// 破關那一場。ECL5/7 的 `A7DCh` 起是最後一戰：
//
//	a7fa  CLEARMONSTERS
//	a7fb  LOAD MONSTER 42h, 1, 42h      ; mon5/66 TYRANITHRAXUS，HP 80 AC 0
//	a802  COMBAT
//	a803  COMPARE @6DC7, 81h ; IF = ; GOTO A5EA   ; 沒打贏那一支
//	a80e  COMPARE @4ABA, FFh ; IF <>
//	a815  SAVE FEh → @4ABA               ; ← 破關旗標（市政廳槽 20）
//	a82a  PROGRAM 08
//	a82d  PRINTCLEAR "KNOWING THAT TYRANTHRAXUS HAS FINALLY BEEN DEFEATED..."
//	a8e5  座標設回 (0,4) 朝向 1、`6E12 = 3`、`NEWECL 0`   ; 回文明區的菲蘭
//
// 這一條從 `A7DCh` 起跑，理由與貧民窟那一條相同：要測的是「打贏之後旗標與
// 結局腳本會不會跑」，不是怎麼走到瓦傑渥城堡的頂樓。ECL5/7 是三個靜態展開
// 解不開的區塊之一，所以位址是用線性掃描讀出來的。
func TestDefeatingTyranthraxusSetsTheVictoryFlag(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(5)
	if !ok {
		t.Fatal("ECL5 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 7, 0xA7DC)
	if err != nil {
		t.Fatal(err)
	}
	hero := poolsave.Character{
		Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", MaxHP: 90, CurrentHP: 90,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1,
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 5
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.state.Party = []poolsave.Character{hero}
	application.state.CharacterLibrary = []poolsave.Character{hero}
	application.spawn = gamepack.Spawn{
		Map: gamepack.MapKey{Archive: 5, BlockID: 7}, X: 3, Y: 4, Facing: 2,
	}
	result, err := session.RunUntilEvent(4096, nil, true)
	if err != nil {
		t.Fatalf("擺出最後一戰：%v", err)
	}
	if err := application.consumeInitialSearch(result); err != nil {
		t.Fatalf("擺出最後一戰：%v", err)
	}
	if !application.combatActive || len(application.combatMonsters) != 1 {
		t.Fatalf("沒有擺出最後一戰：active=%v 怪物 %+v 文字 %q",
			application.combatActive, application.combatMonsters, application.eventText)
	}
	if name := application.combatMonsters[0].Record.Name; name != "TYRANITHRAXUS" {
		t.Fatalf("最後一戰擺出來的是 %q，原版是 TYRANITHRAXUS", name)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatal("ENTER 沒有進戰術盤")
	}
	for index := 1; index < len(state.Roster); index++ {
		if !state.Friendly[index] {
			state.Roster[index].FootprintClass = 0
			state.Scores[index] = 0
			state.States[index] = combat.DyingState
		}
	}
	state.Mover, state.Prompt = 0, false
	state.endRound(application.rollDice)
	if !state.Prompt {
		t.Fatal("打倒之後沒有跳出續戰詢問")
	}
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	machine := session.Machine()
	if got := machine.Memory[0x4ABA]; got != 0xFE {
		t.Fatalf("打贏之後 4ABA=%d，應該是 FEh（`A815h`）", got)
	}

	// 結局過場（spec 108）：`A82Ah PROGRAM 08` 進來，三頁台詞，一頁一個 ENTER。
	if !application.endingActive {
		t.Fatalf("打贏之後沒有進結局過場（文字 %q）",
			strings.TrimSpace(application.eventText))
	}
	if got := len(application.endingPages); got != 3 {
		t.Fatalf("結局過場切成 %d 頁，原版是 3 頁", got)
	}
	if !strings.Contains(application.eventText, "dragon roars") {
		t.Errorf("結局第一頁是 %q", application.eventText)
	}
	for page := 0; page < 3; page++ {
		if application.endingActive {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatalf("結局第 %d 頁：%v", page+1, err)
			}
		}
	}
	if application.endingActive {
		t.Error("翻完三頁之後結局過場還開著")
	}
	// 結局腳本：`A82Ah PROGRAM 08` 之後印出結局文字，再把座標設回
	// `(0,4)` 朝向 1、`6E12 = 3`、`NEWECL 0`，也就是回到文明區的菲蘭。
	ending := ""
	for step := 0; step < 40; step++ {
		if text := strings.TrimSpace(application.eventText); text != "" {
			ending = text
		}
		if !application.cellEventPending && !application.cellWaitingMenu {
			break
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatalf("結局第 %d 步：%v", step, err)
		}
	}
	// 結局腳本最後把隊伍送回文明區的菲蘭：`6E12 = 3` ＋ `NEWECL 0`。
	if application.eclArchive != 3 || session.CurrentBlockID() != 0 {
		t.Errorf("結局之後停在 ecl%d/%d，應該回到 ecl3/0",
			application.eclArchive, session.CurrentBlockID())
	}
	t.Logf("破關：4ABA=%d，結局把隊伍送回 ecl%d/%d，PC %04X，文字 %q",
		machine.Memory[0x4ABA], application.eclArchive,
		session.CurrentBlockID(), 0x9900+session.Machine().PC, ending)
}

// 結局過場走中文。這一條擋的是「翻了但沒接上」：十三行在對照表裡，
// 但過場如果沒有走翻譯管線，畫面上還是英文。
func TestEndingCutscenePagesAreTranslated(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	catalogue, err := gametext.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	application.gameText = catalogue
	if err := application.enterEnding(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(application.eventText, "巨龍") {
		t.Errorf("結局第一頁沒有走中文：%q", application.eventText)
	}
	// 最後一頁翻完會叫 ECL 往下跑，這一條沒有 session，所以只翻到最後一頁。
	pages := len(application.endingPages)
	if pages != 3 {
		t.Fatalf("結局過場切成 %d 頁，原版是 3 頁", pages)
	}
	for page := 1; page < pages; page++ {
		if err := application.advanceEnding(); err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(application.eventText, "abcdefghijklmnopqrstuvwxyz") {
			t.Errorf("第 %d 頁還有英文小寫：%q", page+1, application.eventText)
		}
	}
}
