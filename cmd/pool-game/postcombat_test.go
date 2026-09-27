package main

// 戰後結算的兩頁、選單的開法、TREASURE 的時機、有資格分經驗值的人數、決鬥（spec 150，
// issue #95／#103）。戰鬥與頁面都從 `Update()` 送鍵；腳本從原版 ECL 真實位址開始跑。

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// scriptApp 從原版 ECL 的某個位址開始跑，四個戰士在 GEO2/20 上（盤面只要一張圖）。
func scriptApp(t *testing.T, archiveNumber uint8, block uint16, start uint16) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(archiveNumber)
	if !ok {
		t.Fatalf("ECL%d archive is absent", archiveNumber)
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, block, start)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 2, BlockID: 20})
	if !ok {
		t.Fatal("GEO2/20 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = archiveNumber
	application.initialMap = &geoMap
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: 4, Y: 4, Facing: 0}
	application.introDone, application.mode = true, modeAdventure
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.roller = diceRoller{random: rand.New(rand.NewSource(3))}
	party := make([]poolsave.Character, 0, 4)
	for index := 0; index < 4; index++ {
		party = append(party, poolsave.Character{Name: string(rune('B' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: 30, CurrentHP: 30,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
	return application
}

// winByKeys 讓盤面上的敵方全部投降，再從「CONTINUE BATTLE?」按 N 收場。
func winByKeys(t *testing.T, application *app) {
	t.Helper()
	state := application.tactical
	if state == nil {
		t.Fatal("no tactical state")
	}
	for _, foe := range foeIndexes(state) {
		state.leaveBoard(uint8(foe), gamepack.SurrenderedState)
	}
	state.Prompt = true
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
}

// drawnPostCombatTexts 畫一次畫面，收下 drawText 畫出來的每一段字。
func drawnPostCombatTexts(application *app) []string {
	var texts []string
	drawnText = func(value string, _, _ int) { texts = append(texts, value) }
	defer func() { drawnText = nil }()
	application.Draw(ebiten.NewImage(logicalWidth, logicalHeight))
	return texts
}

// 打贏一場：`08E0h` 印 "The party has won."、"Each character receives N"（N 是職業調整前的每份）、
// "experience points."，底列是 "PRESS <ENTER>/<RETURN> TO CONTINUE"；按一下才是戰利品選單，
// 選單的文字框是空的（原版 dosgolem 收據 `podol-00`／`podol-01`）。
func TestVictoryShowsThePartyHasWonBeforeTheMenu(t *testing.T) {
	application, _, _ := finishWithLoot(t, false)
	if !application.postCombatPageActive() || application.treasureStage != treasureResult {
		t.Fatalf("after the fight stage=%d page=%v, want the result page", application.treasureStage,
			application.postCombatPageActive())
	}
	if application.screenName() != "treasure-result" {
		t.Fatalf("screen %q, want treasure-result", application.screenName())
	}
	share := application.postCombat.share
	if share == 0 || application.state.Party[0].Experience != share {
		t.Fatalf("share %d, member 0 has %d XP (strength 15 gets no bonus)", share, application.state.Party[0].Experience)
	}
	texts := drawnPostCombatTexts(application)
	for _, want := range []string{"The party has won.", "Each character receives " + itoa(share),
		"experience points.", "PRESS <ENTER>/<RETURN> TO CONTINUE"} {
		if !slices.Contains(texts, want) {
			t.Fatalf("result page draws %q, missing %q", texts, want)
		}
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.postCombatPageActive() || application.treasureStage != treasureMain || application.eventText != "" {
		t.Fatalf("after Enter stage=%d text=%q, want the menu with an empty text box", application.treasureStage,
			application.eventText)
	}
	if !slices.Contains(application.cellMenuOptions, "Take") {
		t.Fatalf("menu %v", application.cellMenuOptions)
	}
}

func itoa(value uint32) string { return fmt.Sprint(value) }

// 怪物身上什麼都沒有也開選單：`0E85h` 沒有提早返回的路，選項是 `View Pool Exit`；
// Exit 之後腳本才續跑。
func TestFightWithoutLootStillOpensTheMenu(t *testing.T) {
	application := stageSlumsFight(t, []eclvm.MonsterSpawn{{MonsterID: 1, Count: 1, IconBlock: 4}})
	for index := range application.combatMonsters {
		record := &application.combatMonsters[index].Record
		for offset := gamepack.MonsterMoneyOffset; offset < gamepack.MonsterMoneyOffset+14; offset++ {
			record.Raw[offset] = 0
		}
		application.combatMonsters[index].Items = nil
	}
	application.tactical.FoeItems = nil
	winByKeys(t, application)
	if application.treasureStage != treasureResult {
		t.Fatalf("stage %d, want the result page", application.treasureStage)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !application.treasureActive || !slices.Equal(application.cellMenuOptions, []string{"View", "Pool", "Exit"}) {
		t.Fatalf("menu %v active=%v, want View Pool Exit", application.cellMenuOptions, application.treasureActive)
	}
	if err := selectMenuOption(t, application, "Exit"); err != nil {
		t.Fatal(err)
	}
	if application.treasureActive {
		t.Fatal("Exit left the menu open")
	}
}

// 倒下的隊員沒有資格分（`+10Dh` 清成 0，spec 084），除數也跟著少一個（`829Bh`）。
func TestFallenMemberGetsNoExperience(t *testing.T) {
	application := stageSlumsFight(t, []eclvm.MonsterSpawn{{MonsterID: 1, Count: 2, IconBlock: 4}})
	state := application.tactical
	fallen := -1
	for index := 1; index < len(state.PartySlot); index++ {
		if state.PartySlot[index] == 1 {
			fallen = index
		}
	}
	if fallen < 0 {
		t.Fatal("member 1 is not on the board")
	}
	state.HitPoints[fallen] = 0
	state.Roster[fallen].FootprintClass = 0
	state.States[fallen] = gamepack.DyingState
	winByKeys(t, application)
	share := application.postCombat.share
	if share == 0 {
		t.Fatal("no experience was awarded")
	}
	if got := application.state.Party[1].Experience; got != 0 {
		t.Fatalf("the fallen member gained %d XP", got)
	}
	for _, index := range []int{0, 2, 3} {
		if got := application.state.Party[index].Experience; got != share {
			t.Fatalf("member %d gained %d XP, want the share %d", index, got, share)
		}
	}
	kobold := application.combatMonstersRecordFor(t, 2, 1)
	value := kobold.ExperienceValue(int(kobold.MaxHitPoints()))
	pool := application.state.PooledMoney
	plus := make([]int8, 0, len(application.treasureItems))
	for _, item := range application.treasureItems {
		plus = append(plus, int8(item.Raw[gamepack.LootItemPlusOffset]))
	}
	if want := (2*value + gamepack.LootExperience(pool, plus)) / 3; share != want {
		t.Fatalf("share %d, want the total over the three standing members %d", share, want)
	}
}

// `CLEARMONSTERS → LOAD MONSTER → TREASURE → COMBAT`（`ecl4/10 A5DBh`）：選單不在戰鬥前開，
// 打完才一起交出來；公款是 TREASURE 寫的那一筆加上怪物身上的錢。
func TestTreasureBeforeAFightWaitsForTheFight(t *testing.T) {
	application := scriptApp(t, 4, 10, 0xA5DB)
	if err := application.continueInitialSearch(nil); err != nil {
		t.Fatal(err)
	}
	if application.treasureActive || !application.combatActive {
		t.Fatalf("treasure=%v combat=%v; the menu must wait for the fight", application.treasureActive,
			application.combatActive)
	}
	treasurePool := application.state.PooledMoney
	if treasurePool == ([7]uint32{}) {
		t.Fatal("TREASURE did not write the pool")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	var monsterMoneyTotal [7]uint32
	for _, monster := range application.combatMonsters {
		money := monsterMoney(monster.Record)
		for currency := range money {
			monsterMoneyTotal[currency] += money[currency] * uint32(monster.Spawn.Count)
		}
	}
	winByKeys(t, application)
	if application.treasureStage != treasureResult || application.postCombat.title() != msgPostCombatWon {
		t.Fatalf("stage %d, want the result page of a won fight", application.treasureStage)
	}
	for currency := range treasurePool {
		if want := treasurePool[currency] + monsterMoneyTotal[currency]; application.state.PooledMoney[currency] != want {
			t.Fatalf("pool %v, want TREASURE %v plus the monsters' money %v", application.state.PooledMoney,
				treasurePool, monsterMoneyTotal)
		}
	}
}

// 同一條 LOAD MONSTER 的第二隻，效果串列是反的（overlay-03 `06FAh..07B4h`）。
func TestSecondMonsterCopyHasReversedEffects(t *testing.T) {
	probe := stageSlumsFight(t, []eclvm.MonsterSpawn{{MonsterID: 1, Count: 1, IconBlock: 4}})
	monsterID := uint8(0)
	for id := uint8(1); id < 200 && monsterID == 0; id++ {
		effects, err := probe.loadMonsterEffects(2, id)
		if err == nil && len(effects) >= 2 && effects[0].Code != effects[len(effects)-1].Code {
			monsterID = id
		}
	}
	if monsterID == 0 {
		t.Fatal("no MON2 record carries two different effects")
	}
	application := stageSlumsFight(t, []eclvm.MonsterSpawn{{MonsterID: monsterID, Count: 2, IconBlock: 4}})
	foes := foeIndexes(application.tactical)
	first, second := application.tactical.Effects[foes[0]], application.tactical.Effects[foes[1]]
	if len(first) < 2 || len(first) != len(second) {
		t.Fatalf("effects %v / %v", first, second)
	}
	for index := range first {
		if first[index].Code != second[len(second)-1-index].Code {
			t.Fatalf("MON2/%d copy order: first %v second %v", monsterID, first, second)
		}
	}
}

// 決鬥（`ecl1/18 A9D2h`：`CLEARMONSTERS → CALL 8001h → LOAD MONSTER → COMBAT`）：只有目前角色
// 上場；贏了印 "You have won the duel."／"The duelist receives N"，N 是總額除以整隊人數，
// 只有上場的人拿到。
func TestChampionDuelDeploysOnlyTheChampion(t *testing.T) {
	application := scriptApp(t, 1, 18, 0xA9D2)
	application.currentCharacter = 2
	if err := application.continueInitialSearch(nil); err != nil {
		t.Fatal(err)
	}
	if !application.duel || !application.combatActive {
		t.Fatalf("duel=%v combat=%v after CALL 8001h", application.duel, application.combatActive)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	// 其他人也在串列上、用掉一格樣板，只是 `+10Dh` 為 0、體型 0（overlay-10 `1D50h`），不登記屍體
	// （`1D55h` 的 `DS:829Ah`）。
	deployed := []int{}
	for index := 1; index < len(application.tactical.PartySlot); index++ {
		if slot := application.tactical.PartySlot[index]; slot >= 0 && application.tactical.Roster[index].FootprintClass != 0 {
			deployed = append(deployed, slot)
		}
	}
	if len(application.tactical.Corpses) != 0 {
		t.Fatalf("the duel registered corpses %v", application.tactical.Corpses)
	}
	if !slices.Equal(deployed, []int{2}) {
		t.Fatalf("deployed party slots %v, want only the champion 2", deployed)
	}
	monsters := uint32(0)
	for _, monster := range application.combatMonsters {
		monsters += monster.Record.ExperienceValue(int(monster.Record.MaxHitPoints())) * uint32(monster.Spawn.Count)
	}
	winByKeys(t, application)
	report := application.postCombat
	if report == nil || report.title() != msgPostCombatDuelWon {
		t.Fatalf("report %+v, want the won duel", report)
	}
	pool := application.state.PooledMoney
	plus := make([]int8, 0, len(application.treasureItems))
	for _, item := range application.treasureItems {
		plus = append(plus, int8(item.Raw[gamepack.LootItemPlusOffset]))
	}
	if want := (monsters + gamepack.LootExperience(pool, plus)) / 4; report.share != want {
		t.Fatalf("duelist share %d, want the total over the whole party %d", report.share, want)
	}
	for index, member := range application.state.Party {
		want := uint32(0)
		if index == 2 {
			want = report.share
		}
		if member.Experience != want {
			t.Fatalf("member %d gained %d XP, want %d", index, member.Experience, want)
		}
	}
	texts := drawnPostCombatTexts(application)
	if !slices.Contains(texts, "You have won the duel.") || !slices.Contains(texts, "The duelist receives "+itoa(report.share)) {
		t.Fatalf("duel page draws %q", texts)
	}
	if application.duel || application.eventMachine.Memory[duelArenaAddress] != 0 {
		t.Fatal("the duel flags survived the fight")
	}
}

// 輸掉決鬥不是全滅（`0507h` 把 DS:4960h 清 0）：印 "You have lost the duel."、經驗值 0，
// 選單照開，腳本照跑；倒下的人換成昏迷（`0656h`）。
func TestLostDuelIsNotAPartyWipe(t *testing.T) {
	application := scriptApp(t, 1, 18, 0xA9D2)
	application.currentCharacter = 0
	if err := application.continueInitialSearch(nil); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	for index := 1; index < len(state.PartySlot); index++ {
		if state.PartySlot[index] == 0 {
			state.HitPoints[index] = 0
			state.Roster[index].FootprintClass = 0
			state.States[index] = gamepack.DyingState
		}
	}
	state.Prompt = true
	if err := press(application, ebiten.KeyN); err != nil {
		t.Fatal(err)
	}
	if application.gameOver {
		t.Fatal("losing the duel ended the game")
	}
	report := application.postCombat
	if report == nil || report.title() != msgPostCombatDuelLost || report.shownShare() != 0 {
		t.Fatalf("report %+v, want the lost duel with 0 experience", report)
	}
	for index, member := range application.state.Party {
		if member.Experience != 0 {
			t.Fatalf("member %d gained %d XP in a lost duel", index, member.Experience)
		}
	}
	if got := application.state.Party[0].Status; got != gamepack.UnconsciousState {
		t.Fatalf("champion status %d, want unconscious (4)", got)
	}
	if err := selectMenuOption(t, application, "Exit"); err != nil {
		t.Fatal(err)
	}
	if application.treasureActive {
		t.Fatal("the menu did not close")
	}
}

// leaveLootMenu 是玩家打完一場的收尾：按過結算頁、在戰利品選單選 Exit，還有東西就答 Yes。
func leaveLootMenu(t *testing.T, application *app) {
	t.Helper()
	if !application.treasureActive {
		t.Fatal("no post-combat menu to leave")
	}
	if err := selectMenuOption(t, application, "Exit"); err != nil {
		t.Fatal(err)
	}
	if application.treasureActive && len(application.cellMenuOptions) != 0 && application.cellMenuOptions[0] == "Yes" {
		if err := selectMenuOption(t, application, "Yes"); err != nil {
			t.Fatal(err)
		}
	}
}
