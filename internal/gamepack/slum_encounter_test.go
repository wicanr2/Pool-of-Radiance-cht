package gamepack_test

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

// 貧民窟紮營被打斷之後排出來的那一場架（spec 136）。
//
// 那一段自己算所有東西：隊伍強度 `÷3 ×2` 決定數量、`RANDOM 2` 挑三種怪其中
// 一種、怪物編號從**腳本自己位元組裡**的五張三格表查出來。所以排不出來的
// 症狀不是崩潰，是 `LOAD MONSTER 0, N, 0`——一場「怪物 0」的架。
//
// 這一條釘的是整條路走得通：先問 `29h ENCOUNTER MENU`（原版那張
// `COMBAT WAIT FLEE PARLAY`），排得出群、編號與數量都不是 0，最後停在
// `24h COMBAT`——也就是真的要開打。
func TestSlumCampEncounterStagesRealMonsters(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	archive, err := gamepack.ReadDOSECLArchive(zipPath, 2)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	party := make([]gamepack.InitialCharacter, 0, 6)
	for index := 0; index < 6; index++ {
		party = append(party, gamepack.InitialCharacter{
			Name: string(rune('A' + index)), ClassID: "fighter", CurrentHP: 60,
			Abilities: [6]int{18, 10, 10, 16, 10, 10},
		})
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9900, party...)
	if err != nil {
		t.Fatal(err)
	}
	result, err := gamepack.RunInitialSessionCampEntry(session, geometry.Grid{},
		gamepack.Spawn{X: 14, Y: 4})
	if err != nil {
		t.Fatalf("跑入口 3：%v", err)
	}
	// 那一段會先丟幾個資源事件出來（`22h`／`23h`），要消費掉才走得到遭遇。
	sawEncounterMenu := false
	for round := 0; round < 24 && len(result.MonsterSpawns) == 0 && !result.Exited; round++ {
		sawEncounterMenu = sawEncounterMenu || hasOpcode(result, gamepack.EncounterMenuOpcode)
		next, err := session.RunUntilEvent(4096, nil, true)
		if err != nil {
			t.Fatalf("第 %d 輪：%v", round, err)
		}
		result = next
	}
	if len(result.MonsterSpawns) == 0 {
		t.Fatalf("入口 3 沒有排出任何怪物（steps=%d exited=%v）", result.Steps, result.Exited)
	}
	for index, spawn := range result.MonsterSpawns {
		t.Logf("第 %d 群：怪物 %d、數量 %d、造形 %d",
			index, spawn.MonsterID, spawn.Count, spawn.IconBlock)
		if spawn.MonsterID == 0 && spawn.IconBlock == 0 {
			t.Fatalf("第 %d 群的編號與造形都是 0——那五張表沒查到（spec 136）", index)
		}
		if spawn.Count == 0 {
			t.Fatalf("第 %d 群的數量是 0——隊伍強度那一段沒算出來", index)
		}
	}
	// 玩家先看到遭遇選單，排完怪之後才是開打。少了任何一段，這條路就不是
	// 原版那一條。
	if !sawEncounterMenu {
		t.Fatal("整段沒有出現 29h ENCOUNTER MENU")
	}
	if !hasOpcode(result, slumCombatOpcode) {
		t.Fatalf("排完怪之後停在 %+v，該是 24h COMBAT", result.Events)
	}
}

// slumCombatOpcode 是 `24h COMBAT`：排完怪之後真的開打的那一條。
const slumCombatOpcode = 0x24

func hasOpcode(result eclvm.Result, opcode byte) bool {
	for _, event := range result.Events {
		if event.Opcode == opcode {
			return true
		}
	}
	return false
}
