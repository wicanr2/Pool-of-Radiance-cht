package main

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// AI 施法（spec 096 entry 4、spec 139 的 `2`，issue #64）。全部從 Update() 送鍵：
// AI 那一格輪到時 tacticalInput 自己分派，按什麼鍵都一樣，這裡按 ENTER。

// newFoeCastApp 是一場一對一：隊員在 (5,5)，敵方在 (10,5)，隔著五格搆不到近戰。
// 參數表與處理常式照原版 ZIP 讀。
func newFoeCastApp(t *testing.T) (*app, *tacticalState) {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	parameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	caster, err := gamepack.ReadDOSSpellCaster(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	state := newFoeTurnState(10, 5, 5, 5, 20)
	state.HitPoints = []int{0, 40, 30}
	state.MaxHitPoints = []int{0, 40, 30}
	state.PartySlot = []int{-1, 0, -1}
	state.AIDriven = aiDriven(state.PartySlot)
	state.FoeTargets = make([]uint8, 3)
	state.TacticModes = make([]uint8, 3)
	state.startCastingRound()
	application := &app{mode: modeAdventure, tacticalPreview: true, roller: fixedRoller{7},
		tactical: state, language: languageEnglish, spellParameters: parameters, spellCaster: caster}
	application.state.Party = []poolsave.Character{{Name: "A"}}
	return application, state
}

// giveFoeSpells 讓敵方那一格帶著一個法術陣列（怪物記錄 `+17h` 起），法師 6 級。
func giveFoeSpells(state *tacticalState, spells ...uint8) {
	var record gamepack.MonsterRecord
	record.Name = "LEVEL 6 MU"
	copy(record.Raw[gamepack.AISpellArrayOffset+1:], spells)
	record.Raw[0x9B] = 6
	state.rememberSpellbook(2, record)
}

func TestFoeMagicUserCastsMagicMissile(t *testing.T) {
	application, state := newFoeCastApp(t)
	giveFoeSpells(state, gamepack.SpellIDMagicMissile)
	before := state.Roster[2]
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "CASTS") {
		t.Fatalf("the magic-user did not cast: log %q status %q", state.FoeLog, state.Status)
	}
	if state.HitPoints[1] >= 40 {
		t.Fatalf("magic missile did no damage: hp %d, status %q", state.HitPoints[1], state.Status)
	}
	if state.Roster[2] != before {
		t.Fatalf("the caster also walked: %+v → %+v", before, state.Roster[2])
	}
	for _, value := range state.Casting.Spells[2] {
		if value == gamepack.SpellIDMagicMissile {
			t.Fatal("the cast spell is still memorised (overlay-25 entry 16 clears it)")
		}
	}
	if target, ok := state.foeTarget(2); !ok || target != 1 {
		t.Fatalf("the spell target did not become the chase target (37B8h writes +0Ah): %d %v", target, ok)
	}
}

// 沒有法術的怪物照舊走過來，次數骰照擲（d7 d7），不會多出施法。
func TestFoeWithoutSpellsDoesNotCast(t *testing.T) {
	application, state := newFoeCastApp(t)
	before := state.Roster[2]
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.FoeLog, "CASTS") || strings.Contains(state.FoeLog, "BEGINS") {
		t.Fatalf("a foe without spells cast: %q", state.FoeLog)
	}
	if state.Roster[2] == before {
		t.Fatalf("the foe did not close in: %q", state.FoeLog)
	}
}

// 優先度不夠的法術、次數骰又擲得少的時候不放（門檻 7，只試一輪）。
func TestFoeSkipsLowPrioritySpellOnAShortRoll(t *testing.T) {
	application, state := newFoeCastApp(t)
	application.roller = fixedRoller{1}
	giveFoeSpells(state, gamepack.SpellIDMagicMissile) // 優先度 6
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.FoeLog, "CASTS") {
		t.Fatalf("magic missile passed a threshold of 7: %q", state.FoeLog)
	}
}

// 火球的施法時間是 1（`+0Ch` 3 ÷ 3）：先「開始施法」、扣先攻，輪到下一次才放。
func TestFoeFireballTakesACastingTurn(t *testing.T) {
	application, state := newFoeCastApp(t)
	giveFoeSpells(state, gamepack.SpellIDFireball)
	state.Scores[1], state.Scores[2] = 3, 5
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.Casting.Pending[2] != gamepack.SpellIDFireball || !strings.Contains(state.FoeLog, "BEGINS") {
		t.Fatalf("fireball did not start casting: pending %v log %q", state.Casting.Pending, state.FoeLog)
	}
	if state.Scores[2] != 4 || state.HitPoints[1] != 40 {
		t.Fatalf("after beginning: score %d (want 5−1), party hp %d", state.Scores[2], state.HitPoints[1])
	}
	// 分數 4 比隊員的 3 高，所以輪到的還是施法者，這一次放出去。
	if state.Mover != 2 {
		t.Fatalf("mover %d, the caster should act again", state.Mover)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[1] >= 40 || !strings.Contains(state.FoeLog, "CASTS") {
		t.Fatalf("the pending fireball never went off: hp %d log %q status %q",
			state.HitPoints[1], state.FoeLog, state.Status)
	}
	if len(state.Casting.Pending) != 0 {
		t.Fatalf("pending spell left behind: %v", state.Casting.Pending)
	}
}

// 開始施法之後挨了打：法術沒了（從記憶清掉），這一隻照常行動。
func TestFoeLosesAPendingSpellWhenHit(t *testing.T) {
	application, state := newFoeCastApp(t)
	giveFoeSpells(state, gamepack.SpellIDFireball)
	state.Scores[1], state.Scores[2] = 3, 5
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.Casting.Pending[2] == 0 {
		t.Fatal("fireball did not start casting")
	}
	// 經傷害入口扣血：runtime +1 清掉、開始施法的那一條當場丟失（damage_interrupt.go）。
	state.HitPoints[2] -= 3
	application.woundCombatant(state, 2, 3)
	if state.Casting.Pending[2] != 0 {
		t.Fatal("the pending fireball survived the wound")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[1] != 40 && strings.Contains(state.FoeLog, "CASTS") {
		t.Fatalf("a wounded caster still released the fireball: %q", state.FoeLog)
	}
	for _, value := range state.Casting.Spells[2] {
		if value == gamepack.SpellIDFireball {
			t.Fatal("the lost fireball is still memorised")
		}
	}
}

// QUICK 的隊員：Magic Off（每場開打的預設）不施法；按 `2` 之後才放。
func TestQuickMagicOffKeepsSpellsAndTwoTurnsItOn(t *testing.T) {
	application, state := newFoeCastApp(t)
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotMagicUser] = 6
	application.state.Party[0] = poolsave.Character{Name: "A", ClassLevels: levels,
		Memorised: []uint8{gamepack.SpellIDMagicMissile}, Quick: true}
	state.AIDriven[1] = true
	state.Mover = 1
	if state.Casting.MagicOn {
		t.Fatal("magic starts on; overlay-10 `1F3Bh` clears DS:6D23h every fight")
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.FoeLog, "CASTS") || application.state.Party[0].Memorised[0] == 0 {
		t.Fatalf("a quick member cast with magic off: %q", state.FoeLog)
	}

	// `2` 打開，下一次輪到他就放。
	state.Mover = 1
	if err := press(application, ebiten.KeyDigit2); err != nil {
		t.Fatal(err)
	}
	if !state.Casting.MagicOn || state.Status != "MAGIC ON" {
		t.Fatalf("2 did not turn magic on: %v %q", state.Casting.MagicOn, state.Status)
	}
	hp := state.HitPoints[2]
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(state.FoeLog, "CASTS") || state.HitPoints[2] >= hp {
		t.Fatalf("magic on but the quick member did not cast: %q, foe hp %d → %d",
			state.FoeLog, hp, state.HitPoints[2])
	}
	if application.state.Party[0].Memorised[0] != 0 {
		t.Fatal("the member's memorised slot was not used up")
	}
	// 再按一次關掉。
	if err := press(application, ebiten.KeyDigit2); err != nil {
		t.Fatal(err)
	}
	if state.Casting.MagicOn || state.Status != "MAGIC OFF" {
		t.Fatalf("2 did not turn magic off: %v %q", state.Casting.MagicOn, state.Status)
	}
}

// 接真的怪物記錄：貧民窟（ECL2）的 LEVEL 3 MU（mon2/94，法術 0F 15 22）從建 roster
// 那一步就記下法術陣列，開打之後只按 ENTER，牠會自己施法。
func TestStagedMagicUserCastsInANaturalFight(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: 2, BlockID: 20})
	if !ok {
		t.Fatal("GEO2/20 is absent")
	}
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
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
	if err := application.enterCombatStaging([]eclvm.MonsterSpawn{{MonsterID: 94, Count: 1, IconBlock: 4}}); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatal("no tactical state")
	}
	caster := 0
	for index, spells := range state.Casting.Spells {
		if len(gamepack.AISpellList(spells)) == 3 {
			caster = index
		}
	}
	if caster == 0 {
		t.Fatalf("the roster did not remember the magic-user's spells: %v", state.Casting.Spells)
	}
	cast := ""
	for tick := 0; tick < 400 && cast == "" && application.tactical != nil; tick++ {
		key := ebiten.KeyEnter
		if state.Prompt {
			key = ebiten.KeyN
		}
		moverBefore := state.Mover
		if err := press(application, key); err != nil {
			t.Fatalf("tick %d: %v", tick, err)
		}
		if int(moverBefore) == caster && strings.Contains(state.FoeLog, "CASTS") {
			cast = state.FoeLog
		}
	}
	if cast == "" {
		t.Fatalf("the level 3 magic-user never cast; spells left %x", state.Casting.Spells[caster])
	}
	t.Logf("%s; spells left %x", cast, gamepack.AISpellList(state.Casting.Spells[caster]))
}
