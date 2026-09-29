package main

// 倒戈的隊伍 NPC 在戰後怎麼結算（#122，spec 167）：overlay-05 entry 2 把 `+10Eh == 1` 的隊員
// 當敵方——經驗值、錢、物品——`1164h` 再把他從隊伍摘掉，`1295h` 分錢時人已經不在。
// 盤面是貧民窟 ECL2 block 20：四名戰士加一名 SWORDSMAN（MON3CHA block 36，`+85h` 份數 3），
// 對兩隻狗頭人。

import (
	"encoding/binary"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// swordsmanGold 是測試給 SWORDSMAN 錢包裡的金幣：戰後要整筆出現在公款裡。
const swordsmanGold = 77

// turnedNPCFixture 與 stageSlumsFight 同一個盤面，隊伍多一名 SWORDSMAN。
func turnedNPCFixture(t *testing.T) (*app, poolsave.Character) {
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
	party := make([]poolsave.Character, 0, 5)
	for index := 0; index < 4; index++ {
		party = append(party, poolsave.Character{Name: string(rune('B' + index)), RaceID: "dwarf",
			GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
			Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: 30, CurrentHP: 30,
			PortraitHead: 1, PortraitBody: 1, IconSize: 1})
	}
	record, err := application.loadMonster(3, 36)
	if err != nil {
		t.Fatal(err)
	}
	items, err := application.npcItems(3, 36)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("SWORDSMAN carries nothing in MON3ITM block 36")
	}
	// 第一件的價值改成非 0：`0111h` 價值大於 0 的一定收，不看 d10。
	items[0].Raw[monsterLootValueOffset] = 1
	npc := poolsave.Character{Name: strings.TrimSpace(record.Name), NPC: true,
		Record: append([]byte(nil), record.Raw[:]...), MaxHP: 30, CurrentHP: 30, Inventory: items}
	npc.Record[gamepack.MoraleOffset] = gamepack.NPCMoraleByte(50)
	npc.Money[pooltreasure.Gold] = swordsmanGold
	party = append(party, npc)
	application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party[:4], Party: party}
	if err := application.enterCombatStaging([]eclvm.MonsterSpawn{{MonsterID: 1, Count: 2, IconBlock: 4}}); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatal("no tactical state")
	}
	return application, npc
}

// boardIndexOf 是隊伍第 slot 位在盤面上的格。
func boardIndexOf(t *testing.T, state *tacticalState, slot int) uint8 {
	t.Helper()
	for index := 1; index < len(state.PartySlot); index++ {
		if state.PartySlot[index] == slot {
			return uint8(index)
		}
	}
	t.Fatalf("party slot %d is not on the board", slot)
	return 0
}

// 第一個隊員撞進 SWORDSMAN 那一格、答 Y：SWORDSMAN 倒戈、挨一下倒下。打完之後他的經驗值
// （`+B8h + +BAh × +B1h` = 35 + 3 × 18 = 89）、金幣與物品都算進來，人從隊伍摘掉，
// 也就沒有「takes and hides his share」那一頁。
func TestTurnedNPCIsSettledLikeAFoeAndLeaves(t *testing.T) {
	application, npc := turnedNPCFixture(t)
	application.roller = fixedRoller{value: 20}
	state := application.tactical
	hero, sword := boardIndexOf(t, state, 0), boardIndexOf(t, state, 4)
	placeAt(t, state, sword, hero, 1)
	state.Prompt, state.Mover, state.Moving, state.Notices = false, hero, false, nil
	state.Budgets[hero] = state.BaseMovement[hero] * 2
	state.HitPoints[sword] = 1

	if err := press(application, tacticalStepKeypad[stepInto(t, state, hero, sword)]); err != nil {
		t.Fatal(err)
	}
	if state.AllyPrompt == nil {
		t.Fatalf("bumping the SWORDSMAN did not ask: status %q", state.Status)
	}
	if err := press(application, ebiten.KeyY); err != nil {
		t.Fatal(err)
	}
	if state.Friendly[sword] || application.state.Party[4].Side != 1 {
		t.Fatalf("Y did not turn the SWORDSMAN: friendly %v side %d", state.Friendly[sword], application.state.Party[4].Side)
	}
	if state.Roster[sword].FootprintClass != 0 {
		t.Fatalf("the SWORDSMAN is still standing (HP %d)", state.HitPoints[sword])
	}

	record := npc.Record
	npcValue := uint32(binary.LittleEndian.Uint16(record[0xB8:])) + uint32(record[0xBA])*uint32(record[0xB1])
	if npcValue != 89 {
		t.Fatalf("SWORDSMAN is worth %d, want 35 + 3 × 18 = 89", npcValue)
	}
	monsterValue, monsterMoney := uint32(0), [pooltreasure.CurrencyCount]uint32{}
	for _, monster := range application.combatMonsters {
		monsterValue += monster.Record.ExperienceValue(int(monster.Record.MaxHitPoints())) * uint32(monster.Spawn.Count)
		for currency := range monsterMoney {
			offset := gamepack.MonsterMoneyOffset + currency*2
			monsterMoney[currency] += uint32(binary.LittleEndian.Uint16(monster.Record.Raw[offset:])) * uint32(monster.Spawn.Count)
		}
	}
	poolBefore := application.state.PooledMoney
	state.Notices = nil // 攻擊那一則的停拍不是這裡要測的
	winByKeys(t, application)

	// `1164h`：人摘掉了；`1295h`：沒有人分錢，直接是結算頁。
	if len(application.state.Party) != 4 {
		t.Fatalf("party has %d members after the fight, want the SWORDSMAN gone", len(application.state.Party))
	}
	for _, member := range application.state.Party {
		if member.NPC {
			t.Fatalf("%s is still in the party", member.Name)
		}
	}
	if application.postCombat == nil || len(application.postCombat.hiders) != 0 ||
		application.treasureStage != treasureResult {
		t.Fatalf("post-combat %+v stage %d; want the result page with no NPC share", application.postCombat,
			application.treasureStage)
	}
	// entry 2 `0089h`：他的錢加進公款，而 `1295h` 沒有從公款扣。
	for currency := range poolBefore {
		want := poolBefore[currency] + monsterMoney[currency]
		if currency == pooltreasure.Gold {
			want += swordsmanGold
		}
		if got := application.state.PooledMoney[currency]; got != want {
			t.Fatalf("pool currency %d is %d, want %d (the SWORDSMAN's purse counted, nothing hidden)",
				currency, got, want)
		}
	}
	// `0111h`：價值大於 0 的那一件複製進戰利品。
	// 複製時 `+34h`（穿戴中）清成 0。
	want := append([]byte(nil), npc.Inventory[0].Raw...)
	want[gamepack.ItemReadiedOffset] = 0
	found := false
	for _, item := range application.treasureItems {
		if string(item.Raw[0x2E:]) == string(want[0x2E:]) {
			found = true
		}
	}
	if !found {
		t.Fatalf("the SWORDSMAN's %q is not in the loot %v", npc.Inventory[0].Name, application.treasureItems)
	}
	// 經驗總額含他的 89，除站著的四個人。
	plus := make([]int8, 0, len(application.treasureItems))
	for _, item := range application.treasureItems {
		plus = append(plus, int8(item.Raw[gamepack.LootItemPlusOffset]))
	}
	total := monsterValue + npcValue + gamepack.LootExperience(application.state.PooledMoney, plus)
	if want := gamepack.DivideExperience(total, 4); application.postCombat.share != want {
		t.Fatalf("share %d, want %d with the SWORDSMAN's 89 counted", application.postCombat.share, want)
	}
}

// 隊員全部倒下或逃走、倒戈的 NPC 還站著：`04ADh` 的 82A0h 不看陣營，439Dh 被清掉，
// 結算走打贏那一條——逃掉的換回狀態 0，倒下的留在隊伍裡，不是「The party has fled.」。
func TestTurnedNPCStandingTurnsAFleeIntoTheWinningBranch(t *testing.T) {
	application, _ := turnedNPCFixture(t)
	application.roller = fixedRoller{value: 20}
	state := application.tactical
	hero, sword := boardIndexOf(t, state, 0), boardIndexOf(t, state, 4)
	placeAt(t, state, sword, hero, 1)
	state.Prompt, state.Mover, state.Moving, state.Notices = false, hero, false, nil
	state.Budgets[hero] = state.BaseMovement[hero] * 2
	state.HitPoints[sword] = 100
	if err := press(application, tacticalStepKeypad[stepInto(t, state, hero, sword)]); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyY); err != nil {
		t.Fatal(err)
	}
	if state.Friendly[sword] || state.Roster[sword].FootprintClass == 0 {
		t.Fatal("the SWORDSMAN should have turned and still be standing")
	}
	// 第一個隊員逃掉，其餘三個倒下；怪物離場。
	state.leaveBoard(hero, gamepack.FledState)
	for slot := 1; slot < 4; slot++ {
		index := boardIndexOf(t, state, slot)
		state.HitPoints[index] = 0
		state.Roster[index].FootprintClass = 0
		state.States[index] = combat.DyingState
	}
	for _, foe := range foeIndexes(state) {
		if foe != int(sword) {
			state.leaveBoard(uint8(foe), gamepack.FledState)
		}
	}
	if err := application.finishCombat(combat.CombatDefeat); err != nil {
		t.Fatal(err)
	}
	if application.gameOver {
		t.Fatal("the party was wiped although one member fled")
	}
	if len(application.state.Party) != 4 || application.state.Party[0].Status != 0 {
		t.Fatalf("party after the fight: %d members, first status %d; want the fallen kept and the runner back at 0",
			len(application.state.Party), application.state.Party[0].Status)
	}
	if application.postCombat == nil || application.postCombat.fled || application.eventMachine.Memory[0x6DC7] == partyFledResultCode {
		t.Fatalf("post-combat %+v @6DC7 %d; want the winning branch", application.postCombat,
			application.eventMachine.Memory[0x6DC7])
	}
}
