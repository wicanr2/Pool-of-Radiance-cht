package main

// dosgolem 收據（docs/audit/dosgolem-deployment-peek-{orc-home,guards,alarm}.json）：五人隊
// 撬門走到貧民窟三場固定事件開打，原版開打那一幀的 `5E85h` 位置表。同狀態（同格、同朝向、
// 同人數、同 LOAD MONSTER、距離 0）直接擺出來——這是診斷，不是玩家路徑——部署要逐格相同，
// 放不下被摘掉的隻數也要相同；之後隊員一律結束回合印敵方走位，跟收據並排（敵方挑目標
// 是擲骰，那一段只印不斷言，見 playtest 補八）。

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

type dosgolemDeployReceipt struct {
	PartyCell struct {
		X, Y   uint8
		Facing uint8
	} `json:"party_cell"`
	Spawns [][3]uint8 `json:"spawns"`
	// MovedBeforeFirstPrompt 是收據那一幀讀到時已經先攻走過一步的敵人：原版的幀是
	// 畫面靜下來等玩家輸入時拍的，先攻比隊員快的敵人已經動了，只能要求「差一格內」。
	MovedBeforeFirstPrompt []int `json:"moved_before_first_prompt"`
	DeployFrame            struct {
		Peek map[string]string `json:"peek"`
	} `json:"deploy_frame"`
}

// receiptTable 解 `5E85h` 的 peek：每筆 (X, Y, 索引, 體型類別)，第 0 筆的類別位就是筆數。
func receiptTable(t *testing.T, peek string) [][2]uint8 {
	t.Helper()
	parts := splitPeek(peek)
	if parts[0] != "0850" {
		t.Fatalf("5E85h peek was sampled in segment %s, not DS", parts[0])
	}
	raw, err := hex.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	count := int(raw[3])
	cells := make([][2]uint8, 0, count)
	for index := 1; index < count; index++ {
		cells = append(cells, [2]uint8{raw[4*index], raw[4*index+1]})
	}
	return cells
}

func splitPeek(peek string) [2]string {
	for i := 0; i < len(peek); i++ {
		if peek[i] == '|' {
			return [2]string{peek[:i], peek[i+1:]}
		}
	}
	return [2]string{"", peek}
}

func TestDeploymentMatchesTheSlumsReceipts(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	for _, name := range []string{"orc-home", "guards", "alarm"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-deployment-peek-"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var receipt dosgolemDeployReceipt
			if err := json.Unmarshal(raw, &receipt); err != nil {
				t.Fatal(err)
			}
			want := receiptTable(t, receipt.DeployFrame.Peek["ds:5E85"])

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
			application.spawn = gamepack.Spawn{Map: geoMap.Key, X: receipt.PartyCell.X, Y: receipt.PartyCell.Y, Facing: receipt.PartyCell.Facing}
			application.introDone, application.mode = true, modeAdventure
			if err := application.configureEventSession(session); err != nil {
				t.Fatal(err)
			}
			application.saveState = func(poolsave.State) error { return nil }
			application.eventMachine.Memory[encounterWalkFlagAddress] = 1
			application.eventMachine.Memory[encounterDistanceAddress] = 0

			party := make([]poolsave.Character, 0, 5)
			for index, hp := range []int{9, 6, 12, 6, 9} {
				party = append(party, poolsave.Character{Name: string(rune('B' + index)), RaceID: "dwarf",
					GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good",
					Abilities: [6]int{15, 10, 10, 13, 10, 10}, MaxHP: hp, CurrentHP: hp,
					PortraitHead: 1, PortraitBody: 1, IconSize: 1})
			}
			application.state = poolsave.State{Schema: poolsave.Schema, CharacterLibrary: party, Party: party}
			spawns := make([]eclvm.MonsterSpawn, 0, len(receipt.Spawns))
			for _, spawn := range receipt.Spawns {
				spawns = append(spawns, eclvm.MonsterSpawn{MonsterID: spawn[0], Count: spawn[1], IconBlock: spawn[2]})
			}
			if err := application.enterCombatStaging(spawns); err != nil {
				t.Fatal(err)
			}
			if err := press(application, ebiten.KeyEnter); err != nil {
				t.Fatal(err)
			}
			state := application.tactical
			if state == nil {
				t.Fatalf("no tactical state: %q", application.eventText)
			}
			if len(state.Roster) != len(want)+1 {
				t.Errorf("roster has %d entries, original 5E85h has %d", len(state.Roster)-1, len(want))
			}
			for index, cell := range want {
				if index+1 >= len(state.Roster) {
					break
				}
				got := state.Roster[index+1]
				if got.X == cell[0] && got.Y == cell[1] {
					continue
				}
				moved := false
				for _, early := range receipt.MovedBeforeFirstPrompt {
					if early == index+1 {
						moved = true
					}
				}
				dx, dy := int(got.X)-int(cell[0]), int(got.Y)-int(cell[1])
				if moved && dx >= -1 && dx <= 1 && dy >= -1 && dy <= 1 {
					t.Logf("combatant %d deployed at (%d,%d); the receipt frame already has it one step on at (%d,%d)", index+1, got.X, got.Y, cell[0], cell[1])
					continue
				}
				t.Errorf("combatant %d at (%d,%d), original 5E85h says (%d,%d)", index+1, got.X, got.Y, cell[0], cell[1])
			}
			snapshot := func(round int) string {
				line := fmt.Sprintf("round %d:", round)
				for index := 1; index < len(state.Roster); index++ {
					cell := state.Roster[index]
					line += fmt.Sprintf(" %d:(%d,%d,%d)", index, cell.X, cell.Y, cell.FootprintClass)
				}
				return line
			}
			t.Log(snapshot(state.Round))
			// 隊員一律結束回合（原版那邊是 GUARD），看敵方怎麼走；原版兩批 d,g 內五人就全滅。
			seen := state.Round
			for tick := 0; tick < 6000 && application.tactical == state && !state.Finished && seen < 4; tick++ {
				key := ebiten.KeyEnter
				if state.Prompt {
					key = ebiten.KeyY
				}
				if err := press(application, key); err != nil {
					t.Fatal(err)
				}
				if state.Round != seen {
					seen = state.Round
					t.Log(snapshot(seen))
				}
			}
		})
	}
}
