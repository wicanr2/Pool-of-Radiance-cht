package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	poolchar "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 種族效果（spec 145，issue #88）。

// overlay16RaceNodes 從原版 overlay-16 的位元組讀出每個種族建角掛的節點，當作對照的 oracle：
// `078Ah..0905h` 依 `cmp al, 種族`（`3C rr 75 xx`）分支，每個碼一段
// `B0 碼 50 31 C0 50 B0 +3 50 B0 +4 50 9A 52 00 00 01`（overlay-24 entry 10）。
// 節點是 `碼、持續 00 00、+3、+4`（entry 10 的 `0ED3h..0EF6h`）。
func overlay16RaceNodes(t *testing.T) map[uint8][]byte {
	t.Helper()
	code, err := gamepack.ReadDOSOverlayCode(dosZIPForTests, 16)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// 正對照：`0790h` 是 `26 88 45 2E`（mov es:[di+2Eh], al，寫種族），`0797h` 讀回來比。
	if !bytes.Equal(code[0x790:0x794], []byte{0x26, 0x88, 0x45, 0x2e}) ||
		!bytes.Equal(code[0x797:0x79b], []byte{0x26, 0x8a, 0x45, 0x2e}) {
		t.Fatalf("overlay-16 0790h is % X, not the race store", code[0x790:0x79b])
	}
	nodes := map[uint8][]byte{}
	race := -1
	for at := 0x79b; at < 0x905; {
		switch {
		case code[at] == 0x3c && code[at+2] == 0x75:
			race = int(code[at+1])
			nodes[uint8(race)] = []byte{}
			at += 4
		case code[at] == 0xb0 && bytes.Equal(code[at+2:at+6], []byte{0x50, 0x31, 0xc0, 0x50}) &&
			code[at+6] == 0xb0 && code[at+8] == 0x50 && code[at+9] == 0xb0 && code[at+11] == 0x50 &&
			bytes.Equal(code[at+12:at+17], []byte{0x9a, 0x52, 0x00, 0x00, 0x01}):
			if race < 0 {
				t.Fatalf("attach call at %04Xh before any race compare", at)
			}
			nodes[uint8(race)] = append(nodes[uint8(race)], code[at+1], 0, 0, code[at+7], code[at+10])
			at += 17
		default:
			at++
		}
	}
	if len(nodes) != 5 {
		t.Fatalf("overlay-16 race branches: %v", nodes)
	}
	return nodes
}

// createWithKeys 從標題按鍵建一個人（種族游標 race、職業游標 0），按 A 加進隊伍，回傳那一個人。
func createWithKeys(t *testing.T, race int, name rune) poolsave.Character {
	t.Helper()
	application, err := newApp(dosZIPForTests, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.exportDOSCharacter = nil
	text := &scriptedTextKeys{scriptedKeys: scriptedKeys{}}
	step := func(key ebiten.Key, chars ...rune) {
		t.Helper()
		application.keys = text
		text.scriptedKeys, text.chars = scriptedKeys{key: true}, chars
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	step(ebiten.KeyEnter)
	step(ebiten.KeyC)
	for down := 0; down < race; down++ {
		step(ebiten.KeyArrowDown)
	}
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyEnter, ebiten.KeyEnter, ebiten.KeyEnter} {
		step(key)
	}
	step(ebiten.Key(-1))
	step(ebiten.KeyEnter)
	step(ebiten.KeyEnter, name)
	step(ebiten.KeyK)
	step(ebiten.KeyE)
	step(ebiten.KeyY)
	step(ebiten.KeyA)
	if len(application.state.Party) != 1 {
		t.Fatalf("creating race cursor %d left party %d: %q", race, len(application.state.Party), application.statusLine)
	}
	return application.state.Party[0]
}

// 從 Update() 建六個種族各一，隊伍裡那一位的效果串列逐位元組等於 overlay-16 推給 entry 10 的節點；
// 人類一個都沒有（原版預設人物全是人類，二十份 `.spc` 裡也沒有任何一個種族碼）。
func TestCreatingEachRaceAttachesTheOverlay16Effects(t *testing.T) {
	oracle := overlay16RaceNodes(t)
	for index, race := range []struct {
		id  string
		dos uint8
	}{{"dwarf", 1}, {"elf", 2}, {"gnome", 3}, {"half-elf", 4}, {"halfling", 5}, {"human", 7}} {
		t.Run(race.id, func(t *testing.T) {
			_, cursor, ok := findRaceIndex(race.id)
			if !ok {
				t.Fatalf("race %q is not in the catalog", race.id)
			}
			member := createWithKeys(t, cursor, rune('A'+index))
			if member.RaceID != race.id {
				t.Fatalf("created %q, want %q", member.RaceID, race.id)
			}
			got := poolchar.ExportDOSEffects(member.Effects)
			var want []byte
			for _, node := range oracle[race.dos] {
				want = append(want, node)
			}
			// `.SPC` 每個節點 9 bytes，後 4 bytes 是遠指標（匯出寫 0）。
			var wantSPC []byte
			for at := 0; at < len(want); at += 5 {
				wantSPC = append(wantSPC, want[at:at+5]...)
				wantSPC = append(wantSPC, 0, 0, 0, 0)
			}
			if !bytes.Equal(got, wantSPC) {
				t.Fatalf(".SPC % X, overlay-16 gives % X", got, wantSPC)
			}
		})
	}
}

// #88 之前的存檔：玩家角色沒有種族效果，讀檔時補在串列最前面；已經有的不重複、
// 原本的節點照順序留著、NPC 與人類不動。
func TestLoadingAnOldSaveAddsTheRaceEffects(t *testing.T) {
	application, err := newApp(dosZIPForTests, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	curse := poolsave.EffectNode{Code: 0x24, Payload: [4]byte{5, 0, 1, 0}}
	// 舊存檔的人照樣從 Update() 建（存檔要通過驗證），再把效果換成 #88 之前的樣子。
	oldMember := func(raceID string, name rune, effects ...poolsave.EffectNode) poolsave.Character {
		_, cursor, _ := findRaceIndex(raceID)
		member := createWithKeys(t, cursor, name)
		member.Effects = effects
		return member
	}
	gnome := oldMember("gnome", 'G', curse)
	elf := oldMember("elf", 'E')
	human := oldMember("human", 'H')
	dwarf := oldMember("dwarf", 'D')
	old := poolsave.State{
		Schema:           poolsave.Schema,
		Party:            []poolsave.Character{gnome, elf, human},
		CharacterLibrary: []poolsave.Character{dwarf, gnome, elf, human},
	}
	path := filepath.Join(t.TempDir(), "old.json")
	if err := poolsave.WriteAtomic(path, old); err != nil {
		t.Fatal(err)
	}
	application.loadState = func() (poolsave.State, error) { return poolsave.Read(path) }
	for range 2 { // 讀兩次：第二次不能再掛一份
		if err := application.loadSavedGame(); err != nil {
			t.Fatal(err)
		}
	}
	node := func(code uint8) poolsave.EffectNode {
		return poolsave.EffectNode{Code: code, Payload: [4]byte{0, 0, 0xff, 0}}
	}
	want := map[string][]poolsave.EffectNode{
		"G": {node(0x61), node(0x12), node(0x2f), node(0x30), curse},
		"E": {node(0x6b)},
		"H": nil,
	}
	for _, member := range application.state.Party {
		if !equalNodes(member.Effects, want[member.Name]) {
			t.Errorf("%s effects %v, want %v", member.Name, member.Effects, want[member.Name])
		}
	}
	dwarfNodes := []poolsave.EffectNode{node(0x5a), node(0x61), node(0x1a), node(0x2f)}
	if got := application.state.CharacterLibrary[0].Effects; !equalNodes(got, dwarfNodes) {
		t.Errorf("library dwarf effects %v, want %v", got, dwarfNodes)
	}
	// NPC 的記錄來自怪物檔，不補。
	npc := poolsave.Character{Name: "NPC", RaceID: "dwarf", NPC: true}
	if withRaceEffects(&npc) || npc.Effects != nil {
		t.Errorf("an NPC gained race effects: %v", npc.Effects)
	}
}

func equalNodes(a, b []poolsave.EffectNode) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

// 建角掛上的 30h 在戰鬥裡作用：從 Update() 建的侏儒帶著它進戰場，被 BUGBEAR 打時命中骰 −4
// （overlay-12 entry 45 `1283h`）；同一骰不帶種族效果的對照組打中。
func TestCreatedGnomeIsHarderForABugbearToHit(t *testing.T) {
	_, cursor, _ := findRaceIndex("gnome")
	gnome := createWithKeys(t, cursor, 'G')
	checkHitCases(t, []hitCase{{
		name: "gnome vs BUGBEAR −4", attacker: 3, target: 2, roll: 13, wantHit: false,
		setup: both(func(state *tacticalState) {
			state.Effects[2] = combatEffects(gnome.Effects)
		}, targetKind(3, 1, 2, "BUGBEAR")),
	}})
}
