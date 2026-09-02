package save

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/randomstream"
)

func validCharacter(name string) Character {
	return Character{Name: name, RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", MaxHP: 8, CurrentHP: 8, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
}

func validItem(name string) Item {
	raw := make([]byte, 63)
	raw[0] = byte(len(name))
	copy(raw[1:], name)
	return Item{Name: name, Raw: raw}
}

func TestReadMigratesSchemaOneHPWithoutAmbiguity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	legacy := `{"schema":"pool-remake-state/1","character_library":[{"name":"HERO","race_id":"dwarf","gender_id":"male","class_id":"fighter","alignment_id":"lawful-good","age":20,"abilities":[10,10,10,10,10,10],"exceptional_strength":0,"gold":100,"hp":7,"raw_hp":7,"portrait_head":1,"portrait_body":1,"icon_head":0,"icon_weapon":0,"icon_size":1,"icon_colors":[[0,0],[0,0],[0,0],[0,0],[0,0],[0,0]]}],"party":[{"name":"HERO","race_id":"dwarf","gender_id":"male","class_id":"fighter","alignment_id":"lawful-good","age":20,"abilities":[10,10,10,10,10,10],"exceptional_strength":0,"gold":100,"hp":7,"raw_hp":7,"portrait_head":1,"portrait_body":1,"icon_head":0,"icon_weapon":0,"icon_size":1,"icon_colors":[[0,0],[0,0],[0,0],[0,0],[0,0],[0,0]]}]}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || got.Party[0].MaxHP != 7 || got.Party[0].CurrentHP != 7 || got.Party[0].Status != 0 {
		t.Fatalf("migration=%+v", got.Party[0])
	}
}

func TestStateAtomicRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "pool.json")
	character := validCharacter("HERO")
	character.Inventory = []Item{validItem("Two-Handed Sword +1")}
	state := NewState()
	state.PooledMoney[3] = 123
	state.CharacterLibrary = []Character{character}
	state.Party = []Character{character}
	state.Campaign = &Campaign{MapArchive: 3, MapBlock: 0, ECLArchive: 3, X: 5, Y: 5, Facing: 2, Session: eclvm.BlockSessionSnapshot{
		Current: 8, TransitionEntries: []int{0, 4}, PendingEntries: []int{4},
		Machine: eclvm.MachineSnapshot{PC: 3218, Memory: []eclvm.MemoryWord{{Address: 0x4A96, Value: 0}, {Address: 0x4AB1, Value: 0}, {Address: 0x4AC1, Value: 4}}, Strings: []eclvm.StringWord{{Address: 0x6100, Value: "campaign"}}, Random: randomstream.Snapshot{Seed: 1, Draws: 3}},
	}}
	if err := WriteAtomic(path, state); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || got.PooledMoney[3] != 123 || len(got.CharacterLibrary) != 1 || len(got.Party) != 1 || got.Party[0].Name != "HERO" || len(got.Party[0].Inventory) != 1 || got.Party[0].Inventory[0].Name != "Two-Handed Sword +1" || got.Campaign == nil || got.Campaign.X != 5 || got.Campaign.ECLArchive != 3 || got.Campaign.Session.Current != 8 || got.Campaign.Session.Machine.Memory[2] != (eclvm.MemoryWord{Address: 0x4AC1, Value: 4}) || got.Campaign.Session.Machine.Random.Draws != 3 {
		t.Fatalf("round trip = %+v", got)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "pool.json" {
		t.Fatalf("save directory = %v", entries)
	}
}

func TestReadMigratesSchemaFourWithoutCampaign(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	character := validCharacter("HERO")
	character.Inventory = []Item{validItem("Sword")}
	state := State{Schema: PreviousSchema, CharacterLibrary: []Character{character}, Party: []Character{character}}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || got.Campaign != nil || len(got.Party[0].Inventory) != 1 {
		t.Fatalf("schema 3 migration=%+v", got)
	}
}

func TestReadMigratesSchemaThreeThroughFiveGoldIntoSevenPools(t *testing.T) {
	for _, schema := range []string{OlderSchema, EarlierSchema, PreviousSchema} {
		t.Run(schema, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pool.json")
			character := validCharacter("HERO")
			character.Gold = 321
			state := State{Schema: schema, PooledGold: 654, CharacterLibrary: []Character{character}, Party: []Character{character}}
			raw, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := Read(path)
			if err != nil {
				t.Fatal(err)
			}
			if got.Schema != Schema || got.PooledMoney[3] != 654 || got.PooledGold != 0 || got.Party[0].Money[3] != 321 || got.Party[0].Gold != 0 || got.CharacterLibrary[0].Money[3] != 321 {
				t.Fatalf("migration=%+v", got)
			}
		})
	}
}

func TestReadRejectsAmbiguousLegacyAndSevenPoolMoney(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	character := validCharacter("HERO")
	character.Gold = 1
	character.Money[3] = 1
	state := State{Schema: PreviousSchema, PooledGold: 1, PooledMoney: [7]uint32{3: 1}, CharacterLibrary: []Character{character}, Party: []Character{character}}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("ambiguous old/new money fields accepted")
	}
}

func TestReadMigratesSchemaFiveCampaignECLArchiveFromMapArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	state := State{Schema: PreviousSchema, Campaign: &Campaign{
		MapArchive: 2, MapBlock: 20, X: 1, Y: 2, Facing: 4,
		Session: eclvm.BlockSessionSnapshot{Current: 20, TransitionEntries: []int{0, 4}, Machine: eclvm.MachineSnapshot{PC: 10, Random: randomstream.Snapshot{Seed: 1}}},
	}}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || got.Campaign == nil || got.Campaign.ECLArchive != 2 {
		t.Fatalf("schema 5 campaign migration=%+v", got.Campaign)
	}
}

func TestReadMigratesSchemaTwoWithEmptyInventory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	previous := `{"schema":"pool-remake-state/2","pooled_gold":0,"character_library":[{"name":"HERO","race_id":"dwarf","gender_id":"male","class_id":"fighter","alignment_id":"lawful-good","age":20,"abilities":[10,10,10,10,10,10],"exceptional_strength":0,"gold":100,"max_hp":7,"current_hp":6,"status":0,"raw_hp":7,"portrait_head":1,"portrait_body":1,"icon_head":0,"icon_weapon":0,"icon_size":1,"icon_colors":[[0,0],[0,0],[0,0],[0,0],[0,0],[0,0]]}],"party":[{"name":"HERO","race_id":"dwarf","gender_id":"male","class_id":"fighter","alignment_id":"lawful-good","age":20,"abilities":[10,10,10,10,10,10],"exceptional_strength":0,"gold":100,"max_hp":7,"current_hp":6,"status":0,"raw_hp":7,"portrait_head":1,"portrait_body":1,"icon_head":0,"icon_weapon":0,"icon_size":1,"icon_colors":[[0,0],[0,0],[0,0],[0,0],[0,0],[0,0]]}]}`
	if err := os.WriteFile(path, []byte(previous), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || got.Party[0].CurrentHP != 6 || len(got.Party[0].Inventory) != 0 {
		t.Fatalf("schema 2 migration=%+v", got)
	}
}

func TestStateRejectsUnknownVersionAndInvalidParty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	if err := os.WriteFile(path, []byte(`{"schema":"future","character_library":[],"party":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("future schema accepted")
	}
	state := NewState()
	state.Party = []Character{validCharacter("MISSING")}
	if err := WriteAtomic(path, state); err == nil {
		t.Fatal("party character absent from library accepted")
	}
	bad := validCharacter("BAD")
	bad.Inventory = []Item{{Name: "bad", Raw: make([]byte, 62)}}
	state = NewState()
	state.CharacterLibrary = []Character{bad}
	if err := WriteAtomic(path, state); err == nil {
		t.Fatal("malformed inventory item accepted")
	}
}

func TestStateRejectsMalformedCampaign(t *testing.T) {
	base := Campaign{MapArchive: 3, ECLArchive: 3, X: 1, Y: 1, Facing: 2, Session: eclvm.BlockSessionSnapshot{Current: 8, TransitionEntries: []int{0}, Machine: eclvm.MachineSnapshot{PC: 1, Random: randomstream.Snapshot{Seed: 1}}}}
	tests := []struct {
		name string
		edit func(*Campaign)
	}{
		{name: "archive", edit: func(c *Campaign) { c.MapArchive = 0 }},
		{name: "position", edit: func(c *Campaign) { c.X = 16 }},
		{name: "facing", edit: func(c *Campaign) { c.Facing = 4 }},
		{name: "pc", edit: func(c *Campaign) { c.Session.Machine.PC = -1 }},
		{name: "entries", edit: func(c *Campaign) { c.Session.TransitionEntries = nil }},
		{name: "memory order", edit: func(c *Campaign) { c.Session.Machine.Memory = []eclvm.MemoryWord{{Address: 2}, {Address: 1}} }},
		{name: "random", edit: func(c *Campaign) { c.Session.Machine.Random.Draws = randomstream.MaxReplayDraws + 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			campaign := base
			test.edit(&campaign)
			state := NewState()
			state.Campaign = &campaign
			if err := state.Validate(); err == nil {
				t.Fatal("invalid campaign accepted")
			}
		})
	}
}

// schema 6 把朝向存成共用 engine 的 0/2/4/6，讀進來要除以 2 回到原版的 0..3
// （spec 076）。
func TestReadMigratesSchemaSixFacing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	state := State{Schema: FacingSchema, Campaign: &Campaign{
		MapArchive: 3, MapBlock: 0, ECLArchive: 3, X: 1, Y: 4, Facing: 6,
		Session: eclvm.BlockSessionSnapshot{Current: 8, TransitionEntries: []int{0}, Machine: eclvm.MachineSnapshot{PC: 1, Random: randomstream.Snapshot{Seed: 1}}},
	}}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || got.Campaign == nil || got.Campaign.Facing != 3 {
		t.Fatalf("schema 6 facing migration=%+v", got.Campaign)
	}
}

// schema 6 不可能存下奇數朝向；出現了就是檔案壞了，不要猜。
func TestReadRejectsSchemaSixOddFacing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.json")
	state := State{Schema: FacingSchema, Campaign: &Campaign{
		MapArchive: 3, MapBlock: 0, ECLArchive: 3, X: 1, Y: 4, Facing: 3,
		Session: eclvm.BlockSessionSnapshot{Current: 8, TransitionEntries: []int{0}, Machine: eclvm.MachineSnapshot{PC: 1, Random: randomstream.Snapshot{Seed: 1}}},
	}}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("schema 6 odd facing accepted")
	}
}
