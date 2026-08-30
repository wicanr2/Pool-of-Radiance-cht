package save

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const Schema = "pool-remake-state/1"

type Character struct {
	Name                string      `json:"name"`
	RaceID              string      `json:"race_id"`
	GenderID            string      `json:"gender_id"`
	ClassID             string      `json:"class_id"`
	AlignmentID         string      `json:"alignment_id"`
	Age                 int         `json:"age"`
	Abilities           [6]int      `json:"abilities"`
	ExceptionalStrength int         `json:"exceptional_strength"`
	Gold                int         `json:"gold"`
	HP                  int         `json:"hp"`
	RawHP               int         `json:"raw_hp"`
	PortraitHead        uint8       `json:"portrait_head"`
	PortraitBody        uint8       `json:"portrait_body"`
	IconHead            uint8       `json:"icon_head"`
	IconWeapon          uint8       `json:"icon_weapon"`
	IconSize            uint8       `json:"icon_size"`
	IconColors          [6][2]uint8 `json:"icon_colors"`
}

type State struct {
	Schema           string      `json:"schema"`
	CharacterLibrary []Character `json:"character_library"`
	Party            []Character `json:"party"`
}

func NewState() State { return State{Schema: Schema} }

func (state State) Validate() error {
	if state.Schema != Schema {
		return fmt.Errorf("Pool save schema %q, want %q", state.Schema, Schema)
	}
	if len(state.Party) > 6 {
		return fmt.Errorf("Pool party has %d characters, maximum is 6", len(state.Party))
	}
	seen := make(map[string]bool)
	for _, character := range state.CharacterLibrary {
		if err := validateCharacter(character); err != nil {
			return err
		}
		if seen[character.Name] {
			return fmt.Errorf("duplicate Pool library character %q", character.Name)
		}
		seen[character.Name] = true
	}
	for _, character := range state.Party {
		if err := validateCharacter(character); err != nil {
			return err
		}
		if !seen[character.Name] {
			return fmt.Errorf("party character %q is absent from the library", character.Name)
		}
	}
	return nil
}

func validateCharacter(character Character) error {
	if len(character.Name) < 1 || len(character.Name) > 15 {
		return fmt.Errorf("Pool character name length %d, want 1..15", len(character.Name))
	}
	if character.PortraitHead < 1 || character.PortraitHead > 14 || character.PortraitBody < 1 || character.PortraitBody > 12 {
		return fmt.Errorf("Pool character %q has invalid portrait", character.Name)
	}
	if character.IconHead > 13 || character.IconWeapon > 31 || (character.IconSize != 1 && character.IconSize != 2) {
		return fmt.Errorf("Pool character %q has invalid combat icon", character.Name)
	}
	return nil
}

func WriteAtomic(path string, state State) error {
	if err := state.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create Pool save directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".pool-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create Pool temporary save: %w", err)
	}
	temporaryName := temporary.Name()
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(temporaryName)
		}
	}()
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		temporary.Close()
		return fmt.Errorf("encode Pool save: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync Pool save: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Pool save: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("commit Pool save: %w", err)
	}
	keep = true
	return nil
}

func Read(path string) (State, error) {
	handle, err := os.Open(path)
	if err != nil {
		return State{}, err
	}
	defer handle.Close()
	decoder := json.NewDecoder(io.LimitReader(handle, 1<<20))
	decoder.DisallowUnknownFields()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode Pool save: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return State{}, fmt.Errorf("Pool save has trailing JSON")
	}
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	return state, nil
}
