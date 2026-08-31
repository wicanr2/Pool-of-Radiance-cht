package save

import (
	"os"
	"path/filepath"
	"testing"
)

func validCharacter(name string) Character {
	return Character{Name: name, RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", MaxHP: 8, CurrentHP: 8, PortraitHead: 1, PortraitBody: 1, IconSize: 1}
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
	state := NewState()
	state.PooledGold = 123
	state.CharacterLibrary = []Character{character}
	state.Party = []Character{character}
	if err := WriteAtomic(path, state); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || got.PooledGold != 123 || len(got.CharacterLibrary) != 1 || len(got.Party) != 1 || got.Party[0].Name != "HERO" {
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
}
