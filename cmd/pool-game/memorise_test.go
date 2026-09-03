package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 只用按鍵在法術畫面上把法術記給一個牧師，記滿了就記不進去，忘掉一格又記得進去。
func TestMemoriseFromTheSpellScreen(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 第 1 級的牧師：建角寫下第一級一格，沒有睿智加成（spec 072）。
	cleric := poolsave.Character{Name: "A", RaceID: "human", GenderID: "male",
		ClassID: "cleric", AlignmentID: "lawful-good",
		Abilities: [6]int{10, 10, 18, 10, 10, 10}, MaxHP: 8, CurrentHP: 8,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{cleric}, Party: []poolsave.Character{cleric}}
	if err := application.openSpells(); err != nil {
		t.Fatal(err)
	}
	application.mode = modeMenu
	maxima, _, ok := application.spellMemberSlots(0)
	if !ok {
		t.Fatal("算不出可記憶數")
	}
	// 第 1 級牧師是 1/0/0：睿智 18 也拿不到加成，加成在「等級大於 1」的分支裡。
	if maxima[gamepack.SpellSlotGroupCleric] != [gamepack.SpellSlotLevels]int{1, 0, 0} {
		t.Fatalf("第 1 級牧師的可記憶數應該是 1/0/0，算出 %v",
			maxima[gamepack.SpellSlotGroupCleric])
	}
	// 游標停在牧師第 1 級的第一條，按 M 記下去。
	press(application, ebiten.KeyDigit1)
	press(application, ebiten.KeyM)
	member := application.state.Party[0]
	memorised := 0
	for _, value := range member.Memorised {
		if value != 0 {
			memorised++
		}
	}
	if memorised != 1 {
		t.Fatalf("應該記下一個，記了 %d 個（狀態列 %q）", memorised, application.statusLine)
	}
	// 只有一格，第二個記不進去。
	press(application, ebiten.KeyM)
	memorised = 0
	for _, value := range application.state.Party[0].Memorised {
		if value != 0 {
			memorised++
		}
	}
	if memorised != 1 {
		t.Fatalf("只有一格，卻記了 %d 個", memorised)
	}
	// 忘掉之後又記得進去。
	press(application, ebiten.KeyF)
	for _, value := range application.state.Party[0].Memorised {
		if value != 0 {
			t.Fatalf("按 F 之後不該還記著 %d", value)
		}
	}
	press(application, ebiten.KeyM)
	memorised = 0
	for _, value := range application.state.Party[0].Memorised {
		if value != 0 {
			memorised++
		}
	}
	if memorised != 1 {
		t.Fatalf("忘掉之後應該記得回去，記了 %d 個", memorised)
	}
	// 角色庫要跟著同步。
	if len(application.state.CharacterLibrary[0].Memorised) == 0 {
		t.Error("角色庫沒有跟著更新")
	}
}

// 只用按鍵走到真的一場戰鬥裡施出魔法飛彈，並且確認：記憶的那一格被用掉、
// 敵人真的掉血、傷害落在 spec 098 算出來的範圍。
//
// 不用戰場預覽——預覽的盤面上沒有敵人，測起來會 skip，而 skip 與通過在
// 報表上分不出來。
func TestCastMagicMissileInCombat(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 第 6 級法師：魔法飛彈 3 發，每發 1d4+1（spec 098）。
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotMagicUser] = 6
	party := make([]poolsave.Character, 0, 6)
	for index := 0; index < 6; index++ {
		member := poolsave.Character{Name: string(rune('A' + index)),
			RaceID: "human", GenderID: "male", ClassID: "magic-user",
			AlignmentID: "lawful-good", Abilities: [6]int{10, 18, 10, 10, 10, 10},
			MaxHP: 30, CurrentHP: 30, PortraitHead: 1, PortraitBody: 1, IconSize: 1,
			ClassLevels: append([]uint8(nil), levels...),
			Memorised:   make([]uint8, gamepack.MemorisedSpellSlots)}
		member.Memorised[0] = gamepack.SpellIDMagicMissile
		party = append(party, member)
	}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: party, Party: party}
	application.saveState = func(poolsave.State) error { return nil }
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
	state := application.tactical
	if state == nil {
		t.Fatalf("tactical state absent: %q", application.statusLine)
	}
	// 輪到隊員動的時候按 C 施法。
	cast := false
	for tick := 0; tick < 4000 && !cast; tick++ {
		state = application.tactical
		if state == nil || state.Finished {
			break
		}
		if state.Prompt {
			if err := press(application, ebiten.KeyY); err != nil {
				t.Fatal(err)
			}
			continue
		}
		mover := state.Mover
		if mover == 0 || int(mover) >= len(state.PartySlot) || state.PartySlot[mover] < 0 {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		target, found := state.nearestReachableOpposing(mover)
		if !found {
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			continue
		}
		partyIndex := state.PartySlot[mover]
		before := state.HitPoints[target]
		if err := press(application, ebiten.KeyC); err != nil {
			t.Fatal(err)
		}
		if !application.castOpen {
			t.Fatalf("按 C 沒有開出施法清單（狀態列 %q）", application.statusLine)
		}
		if len(application.castOptions) != 1 ||
			application.castOptions[0].ID != gamepack.SpellIDMagicMissile {
			t.Fatalf("清單應該只有魔法飛彈一條，拿到 %+v", application.castOptions)
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		// 魔法飛彈是「挑一個目標」那一組（參數表 +6 的低四位 ＝ 4），
		// 所以選完法術會先進選目標那一步，再按一次 Enter 才施出去。
		if !application.castTargeting {
			t.Fatalf("挑一個目標的法術應該先進選目標那一步（狀態列 %q）",
				application.statusLine)
		}
		if len(application.castTargets) == 0 {
			t.Fatal("選目標那一步沒有任何候選")
		}
		// 停在繞得過去的最近敵人身上。
		if application.castTargets[application.castTargetCursor] != target {
			t.Errorf("預設應該停在 %d，停在 %d", target,
				application.castTargets[application.castTargetCursor])
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
		if application.castTargeting {
			t.Fatal("確定之後應該離開選目標那一步")
		}
		if application.state.Party[partyIndex].Memorised[0] != 0 {
			t.Fatal("施完之後那一格記憶沒有被用掉")
		}
		after := state.HitPoints[target]
		damage := before - after
		if damage <= 0 {
			t.Fatalf("敵人的血從 %d 變成 %d，法術沒有造成傷害（狀態列 %q）",
				before, after, application.statusLine)
		}
		// 3 發，每發 1d4+1 → 6..15。目標被打倒時血會被夾到 0，所以只查上界。
		if damage > 15 {
			t.Fatalf("第 6 級的魔法飛彈最多 15 點，造成了 %d 點", damage)
		}
		t.Logf("施出魔法飛彈：對 %d 造成 %d 點", target, damage)
		cast = true
	}
	if !cast {
		t.Fatal("整場都沒有機會施法")
	}
}

// 選好的法術要休息過才施得出來：法術畫面記下、紮營休息、才進得了施法清單。
func TestMemorisedSpellsNeedRestBeforeCasting(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotMagicUser] = 6
	member := poolsave.Character{Name: "A", RaceID: "human", GenderID: "male",
		ClassID: "magic-user", AlignmentID: "lawful-good",
		Abilities: [6]int{10, 18, 10, 10, 10, 10}, MaxHP: 30, CurrentHP: 12,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1,
		ClassLevels: append([]uint8(nil), levels...)}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{member}, Party: []poolsave.Character{member}}
	if err := application.openSpells(); err != nil {
		t.Fatal(err)
	}
	// 切到法師第 1 級那一頁，游標停在魔法飛彈上。
	for application.spells.group != 3 {
		press(application, ebiten.KeyTab)
	}
	found := false
	for index, spell := range application.spells.current() {
		if uint8(spell.Index+1) == gamepack.SpellIDMagicMissile {
			application.spells.cursor, found = index, true
			break
		}
	}
	if !found {
		t.Fatal("法師第 1 級那一頁找不到魔法飛彈")
	}
	press(application, ebiten.KeyDigit1)
	press(application, ebiten.KeyM)
	stored := application.state.Party[0].Memorised
	if len(stored) == 0 || stored[0]&0x7f != gamepack.SpellIDMagicMissile {
		t.Fatalf("沒有記下魔法飛彈：%v（狀態列 %q）", stored, application.statusLine)
	}
	if gamepack.MemorisedSpellIsReady(stored[0]) {
		t.Fatal("剛選好就變成可施展了，應該要先休息")
	}
	// 紮營休息：記完，而且整隊回滿。
	application.spellsOpen = false
	application.mode = modeAdventure
	press(application, ebiten.KeyE)
	if !application.campOpen {
		t.Fatalf("按 E 沒有開出紮營選單（狀態列 %q）", application.statusLine)
	}
	press(application, ebiten.KeyEnter) // 游標在「休息」上
	if application.campOpen {
		t.Fatal("休息完應該關掉紮營選單")
	}
	rested := application.state.Party[0]
	if !gamepack.MemorisedSpellIsReady(rested.Memorised[0]) {
		t.Fatalf("休息完應該記好了，還是 %#02x（狀態列 %q）",
			rested.Memorised[0], application.statusLine)
	}
	if rested.CurrentHP != rested.MaxHP {
		t.Errorf("休息完整隊要回滿，還是 %d/%d", rested.CurrentHP, rested.MaxHP)
	}
	if application.state.CharacterLibrary[0].CurrentHP != rested.MaxHP {
		t.Error("角色庫沒有跟著更新")
	}
}

// 選目標那一步可以換人：N 往後、P 往前，換完施出去打的是換到的那一個。
func TestCastTargetingCyclesTargets(t *testing.T) {
	state := &tacticalState{
		Roster:     make([]combat.CombatantCell, 5),
		Friendly:   []bool{false, true, false, false, false},
		HitPoints:  []int{0, 20, 20, 20, 20},
		PartySlot:  []int{-1, 0, -1, -1, -1},
		States:     make([]uint8, 5),
		Scores:     make([]uint8, 5),
		Budgets:    make([]uint8, 5),
		HitDice:    make([]uint8, 5),
		SleepFlag:  make([]uint8, 5),
		Asleep:     make([]bool, 5),
		ArmorClass: make([]int, 5),
		THAC0:      make([]uint8, 5),
		Damage:     make([]combat.DamageDice, 5),
		Mover:      1,
	}
	for index := 1; index < 5; index++ {
		state.Roster[index] = combat.CombatantCell{X: uint8(index), Y: 1, FootprintClass: 1}
	}
	application := &app{tactical: state, tacticalPreview: true, mode: modeAdventure}
	if !application.beginCastTargeting(castOption{ID: gamepack.SpellIDMagicMissile, Label: "魔法飛彈"}) {
		t.Fatal("開不出選目標那一步")
	}
	if len(application.castTargets) != 4 {
		t.Fatalf("四個站著的都該是候選，拿到 %d 個", len(application.castTargets))
	}
	start := application.castTargetCursor
	press(application, ebiten.KeyN)
	if application.castTargetCursor == start {
		t.Error("按 N 沒有換人")
	}
	press(application, ebiten.KeyP)
	if application.castTargetCursor != start {
		t.Error("按 P 沒有換回去")
	}
	// 繞一圈回到原點。
	for index := 0; index < len(application.castTargets); index++ {
		press(application, ebiten.KeyN)
	}
	if application.castTargetCursor != start {
		t.Errorf("繞一圈應該回到 %d，停在 %d", start, application.castTargetCursor)
	}
	// ESC 取消不該把記憶用掉，也不該結束回合。
	press(application, ebiten.KeyEscape)
	if application.castTargeting {
		t.Error("ESC 應該離開選目標那一步")
	}
}

// A 鍵瞄準攻擊：超出射程不打也不消耗回合，射程內才打。
func TestAimedAttackRespectsWeaponRange(t *testing.T) {
	state := &tacticalState{
		Roster:     make([]combat.CombatantCell, 3),
		Friendly:   []bool{false, true, false},
		HitPoints:  []int{0, 20, 20},
		PartySlot:  []int{-1, 0, -1},
		States:     make([]uint8, 3),
		Scores:     []uint8{0, 5, 5},
		Budgets:    make([]uint8, 3),
		HitDice:    make([]uint8, 3),
		SleepFlag:  make([]uint8, 3),
		Asleep:     make([]bool, 3),
		ArmorClass: []int{0, 50, 50},
		THAC0:      []uint8{0, 40, 40},
		Damage:     []combat.DamageDice{{}, {Count: 1, Sides: 8}, {Count: 1, Sides: 8}},
		Mover:      1,
	}
	// 距離用 TraceMovement 算（spec 098），所以要有盤面。這裡不測地形，
	// 給一整片可通行的格子並讓它跳過地形判定。
	state.Grid = combat.TacticalGrid{
		IgnoreTerrain: true,
		Terrain:       make([]uint8, combat.TacticalRowStride*(combat.TacticalMaxY+1)),
	}
	state.Roster[1] = combat.CombatantCell{X: 1, Y: 1, FootprintClass: 1}
	state.Roster[2] = combat.CombatantCell{X: 17, Y: 1, FootprintClass: 1} // 遠處
	application := &app{tactical: state, tacticalPreview: true, mode: modeAdventure,
		roller: diceRoller{random: rand.New(rand.NewSource(1))}}
	application.state = poolsave.State{Party: []poolsave.Character{{Name: "A", ClassID: "fighter"}}}
	press(application, ebiten.KeyA)
	if !application.castTargeting || !application.castTargetingAttack {
		t.Fatalf("按 A 應該進瞄準（狀態列 %q）", application.statusLine)
	}
	// 唯一的候選是八格外那個，空手只打得到一格。
	// 判準用「回合有沒有結束」而不是「有沒有掉血」——打得到也可能沒打中，
	// 而超出射程那條路在 endTurn 之前就返回了。
	before := state.HitPoints[2]
	mover := state.Mover
	press(application, ebiten.KeyEnter)
	if state.HitPoints[2] != before {
		t.Errorf("超出射程不該打得到，血從 %d 變成 %d", before, state.HitPoints[2])
	}
	if state.Mover != mover {
		t.Error("超出射程不該消耗回合")
	}
	// 拉到相鄰就打得到，回合也用掉。
	state.Roster[2] = combat.CombatantCell{X: 2, Y: 1, FootprintClass: 1}
	state.Mover = mover
	press(application, ebiten.KeyA)
	if !application.castTargetingAttack {
		t.Fatalf("第二次按 A 沒有進瞄準（狀態列 %q）", application.statusLine)
	}
	press(application, ebiten.KeyEnter)
	if state.Mover == mover && !state.Finished {
		t.Errorf("相鄰打完應該換人，還停在 %d（%q）", state.Mover, state.Status)
	}
}

// 火球術的範圍照原版收人：預算 2 內走得到的才吃傷害。
func TestFireballAreaUsesTheOriginalBudget(t *testing.T) {
	state := &tacticalState{
		Roster:    make([]combat.CombatantCell, 4),
		Friendly:  []bool{false, true, false, false},
		HitPoints: []int{0, 20, 20, 20},
		Grid: combat.TacticalGrid{IgnoreTerrain: true,
			Terrain: make([]uint8, combat.TacticalRowStride*(combat.TacticalMaxY+1))},
		Mover: 1,
	}
	state.Roster[1] = combat.CombatantCell{X: 2, Y: 2, FootprintClass: 1}
	state.Roster[2] = combat.CombatantCell{X: 3, Y: 2, FootprintClass: 1}  // 旁邊
	state.Roster[3] = combat.CombatantCell{X: 30, Y: 2, FootprintClass: 1} // 老遠
	if !state.withinArea(1, 2, gamepack.FireballAreaBudget) {
		t.Error("旁邊那個應該在範圍內")
	}
	if state.withinArea(1, 3, gamepack.FireballAreaBudget) {
		t.Error("老遠那個不該在範圍內")
	}
	// 距離也用同一套：旁邊是 1、老遠的走得到但很遠。
	if distance, ok := state.tacticalRange(1, 2); !ok || distance > 1 {
		t.Errorf("旁邊那個的距離應該是 1 以內，算出 %d（走得到 %t）", distance, ok)
	}
	if distance, ok := state.tacticalRange(1, 3); !ok || distance < 10 {
		t.Errorf("老遠那個的距離應該很大，算出 %d（走得到 %t）", distance, ok)
	}
}
