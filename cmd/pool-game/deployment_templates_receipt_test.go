package main

// 陣型樣板對原版執行期（#35，spec 061）：dosgolem 在四場開打那一幀讀 `DS:43A2h` 的
// 528 bytes（2 組 × 4 陣型 × 6 列 × 11 欄，放過的格已被 `14CFh` 清零）與來源表 `DS:304h`；
// remake 同狀態部署完之後的 `deploymentTemplates` 要逐 byte 相同。

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

type dosgolemTemplateReceipt struct {
	Fights map[string]struct {
		Spans     string `json:"ds_304"`
		Templates string `json:"ds_43A2"`
		PartyCell string `json:"party_cell"`
		Offsets   string `json:"offsets_45B2"`
	} `json:"fights"`
}

func peekBytes(t *testing.T, field string) []byte {
	parts := strings.SplitN(field, "|", 2)
	if len(parts) != 2 || parts[0] != "0850" {
		t.Fatalf("peek %q was not read from the game's DS", field)
	}
	raw, err := hex.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestDeploymentTemplatesMatchTheOriginalAtRuntime(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-deployment-templates.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt dosgolemTemplateReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	// 同一份收據的四場都要帶同一張來源表，而且要等於 START.EXE 裡的 `DS:304h`
	//（`ida-start-deployment-tables.json` 是 2D0h 起 112 bytes，304h 從第 52 個起）。
	staticRaw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "ida-start-deployment-tables.json"))
	if err != nil {
		t.Fatal(err)
	}
	var static struct {
		Offset int    `json:"ds_offset"`
		Bytes  string `json:"bytes"`
	}
	if err := json.Unmarshal(staticRaw, &static); err != nil {
		t.Fatal(err)
	}
	if static.Offset != 0x2D0 || len(static.Bytes) < 112*2 {
		t.Fatalf("static table export starts at %#x with %d hex chars", static.Offset, len(static.Bytes))
	}
	for name, fight := range receipt.Fights {
		spans := peekBytes(t, fight.Spans)
		if !strings.EqualFold(hex.EncodeToString(spans), static.Bytes[52*2:112*2]) {
			t.Fatalf("%s: DS:304h at runtime %x differs from START.EXE %s", name, spans, static.Bytes[52*2:112*2])
		}
	}
	for _, name := range []string{"orc-home", "guards", "alarm", "goblins"} {
		fight, ok := receipt.Fights[name]
		if !ok {
			t.Fatalf("receipt has no fight %q", name)
		}
		t.Run(name, func(t *testing.T) {
			want := peekBytes(t, fight.Templates)
			if len(want) != 2*combat.DeploymentsPerSet*combat.DeploymentTemplateRows*combat.DeploymentTemplateCols {
				t.Fatalf("template peek has %d bytes", len(want))
			}
			cell := peekBytes(t, fight.PartyCell)
			offsets := peekBytes(t, fight.Offsets)
			var spawns [][3]uint8
			if name == "goblins" {
				// 哥布林那一場的收據沒有 spawns：四隻 GOBLIN GUARD，造形 4（deployment_receipt_test）。
				spawns = [][3]uint8{{0, 4, 4}}
			} else {
				walk, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-deployment-peek-"+name+".json"))
				if err != nil {
					t.Fatal(err)
				}
				var spawnsReceipt struct {
					Spawns [][3]uint8 `json:"spawns"`
				}
				if err := json.Unmarshal(walk, &spawnsReceipt); err != nil {
					t.Fatal(err)
				}
				spawns = spawnsReceipt.Spawns
			}
			application := newDeploymentFixture(t, zipPath, cell[0], cell[1], cell[2]/2, spawns, name == "goblins")
			got := application.deploymentTemplates
			flat := make([]byte, 0, len(want))
			for side := range got {
				for formation := range got[side] {
					flat = append(flat, got[side][formation][:]...)
				}
			}
			// 45B8h／45B9h 是兩邊的象限，先對它們——象限錯了整組樣板都會錯。
			distance := 0
			if application.eventMachine != nil {
				distance = int(application.eventMachine.Memory[encounterDistanceAddress])
			}
			sides, err := combat.DeploymentSides(cell[2], distance, [2]int{0, 0})
			if err != nil {
				t.Fatal(err)
			}
			if sides[0].Quadrant != offsets[6] || sides[1].Quadrant != offsets[7] {
				t.Fatalf("quadrants %d/%d, original 45B8h/45B9h %d/%d", sides[0].Quadrant, sides[1].Quadrant, offsets[6], offsets[7])
			}
			diffs := []string{}
			for i := range want {
				if want[i] != flat[i] {
					side, rest := i/264, i%264
					formation, rest := rest/66, rest%66
					diffs = append(diffs, fmt.Sprintf("side %d formation %d row %d col %d: original %d remake %d",
						side, formation, rest/11, rest%11, want[i], flat[i]))
				}
			}
			if len(diffs) != 0 {
				t.Fatalf("%d of 528 template bytes differ:\n%s", len(diffs), strings.Join(diffs, "\n"))
			}
			t.Logf("%s: 528 template bytes match the original (quadrants %d/%d)", name, sides[0].Quadrant, sides[1].Quadrant)
		})
	}
}

// newDeploymentFixture 把 remake 擺到同一格、同朝向、同隊伍、同怪物群，走到開打
// （與 deployment_slums_receipts_test 同一套）。goblins 那一場是走路遭遇的突襲，
// 距離走法一樣由 enterCombatStaging 壓。
func newDeploymentFixture(t *testing.T, zipPath string, x, y, facing uint8, spawns [][3]uint8, goblins bool) *app {
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
	application.spawn = gamepack.Spawn{Map: geoMap.Key, X: x, Y: y, Facing: facing}
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
	monsters := make([]eclvm.MonsterSpawn, 0, len(spawns))
	for _, spawn := range spawns {
		id := spawn[0]
		if goblins {
			// 走路遭遇的哥布林：怪物編號用名稱找（deployment_receipt_test 同一招）。
			found := false
			for candidate := uint8(0); candidate < 60 && !found; candidate++ {
				record, err := application.loadMonster(2, candidate)
				if err == nil && strings.HasPrefix(record.Name, "GOBLIN") {
					id, found = candidate, true
				}
			}
			if !found {
				t.Fatal("ECL2 has no GOBLIN monster record")
			}
		}
		monsters = append(monsters, eclvm.MonsterSpawn{MonsterID: id, Count: spawn[1], IconBlock: spawn[2]})
	}
	if err := application.enterCombatStaging(monsters); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil {
		t.Fatal("no tactical state")
	}
	return application
}
