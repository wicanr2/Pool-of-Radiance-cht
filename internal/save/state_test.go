package save

import (
	"os"
	"path/filepath"
	"testing"
)

func validCharacter(name string) Character {
	return Character{Name: name, RaceID: "dwarf", GenderID: "male", ClassID: "fighter", AlignmentID: "lawful-good", PortraitHead: 1, PortraitBody: 1, IconSize: 1}
}

func TestStateAtomicRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "pool.json")
	character := validCharacter("HERO")
	state := NewState()
	state.CharacterLibrary = []Character{character}
	state.Party = []Character{character}
	if err := WriteAtomic(path, state); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != Schema || len(got.CharacterLibrary) != 1 || len(got.Party) != 1 || got.Party[0].Name != "HERO" {
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
