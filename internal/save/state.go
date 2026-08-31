package save

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wicanr2/golden-box-remake-engine/eclvm"
	"github.com/wicanr2/golden-box-remake-engine/randomstream"
)

const (
	Schema         = "pool-remake-state/5"
	PreviousSchema = "pool-remake-state/4"
	EarlierSchema  = "pool-remake-state/3"
	OlderSchema    = "pool-remake-state/2"
	LegacySchema   = "pool-remake-state/1"
)

type Item struct {
	Name string `json:"name"`
	Raw  []byte `json:"raw"`
}

type Character struct {
	Name                string      `json:"name"`
	RaceID              string      `json:"race_id"`
	GenderID            string      `json:"gender_id"`
	ClassID             string      `json:"class_id"`
	AlignmentID         string      `json:"alignment_id"`
	Age                 int         `json:"age"`
	Abilities           [6]int      `json:"abilities"`
	ExceptionalStrength int         `json:"exceptional_strength"`
	Money               [7]uint16   `json:"money"`
	Gold                int         `json:"gold,omitempty"` // schema 1..4 read-only migration field
	MaxHP               int         `json:"max_hp"`
	CurrentHP           int         `json:"current_hp"`
	Status              uint8       `json:"status"`
	RawHP               int         `json:"raw_hp"`
	PortraitHead        uint8       `json:"portrait_head"`
	PortraitBody        uint8       `json:"portrait_body"`
	IconHead            uint8       `json:"icon_head"`
	IconWeapon          uint8       `json:"icon_weapon"`
	IconSize            uint8       `json:"icon_size"`
	IconColors          [6][2]uint8 `json:"icon_colors"`
	Inventory           []Item      `json:"inventory,omitempty"`
}

type Campaign struct {
	MapArchive uint8                      `json:"map_archive"`
	MapBlock   uint8                      `json:"map_block"`
	X          uint8                      `json:"x"`
	Y          uint8                      `json:"y"`
	Facing     uint8                      `json:"facing"`
	Session    eclvm.BlockSessionSnapshot `json:"session"`
}

type State struct {
	Schema           string      `json:"schema"`
	PooledMoney      [7]uint32   `json:"pooled_money"`
	PooledGold       int         `json:"pooled_gold,omitempty"` // schema 2..4 read-only migration field
	CharacterLibrary []Character `json:"character_library"`
	Party            []Character `json:"party"`
	Campaign         *Campaign   `json:"campaign,omitempty"`
}

func NewState() State { return State{Schema: Schema} }

func (state State) Validate() error {
	if state.Schema != Schema {
		return fmt.Errorf("Pool save schema %q, want %q", state.Schema, Schema)
	}
	if len(state.Party) > 6 {
		return fmt.Errorf("Pool party has %d characters, maximum is 6", len(state.Party))
	}
	if state.PooledGold != 0 {
		return fmt.Errorf("Pool schema 5 retains legacy pooled_gold %d", state.PooledGold)
	}
	if state.Campaign != nil {
		if err := validateCampaign(*state.Campaign); err != nil {
			return err
		}
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

func validateCampaign(campaign Campaign) error {
	if campaign.MapArchive < 1 || campaign.MapArchive > 8 {
		return fmt.Errorf("Pool campaign map archive %d is outside 1..8", campaign.MapArchive)
	}
	if campaign.X > 15 || campaign.Y > 15 {
		return fmt.Errorf("Pool campaign position (%d,%d) is outside 16x16 map", campaign.X, campaign.Y)
	}
	if campaign.Facing > 6 || campaign.Facing%2 != 0 {
		return fmt.Errorf("Pool campaign facing %d is not cardinal", campaign.Facing)
	}
	snapshot := campaign.Session
	if snapshot.Machine.PC < 0 {
		return fmt.Errorf("Pool campaign ECL PC %d is negative", snapshot.Machine.PC)
	}
	if len(snapshot.TransitionEntries) == 0 {
		return fmt.Errorf("Pool campaign transition entry sequence is empty")
	}
	for _, entry := range append(append([]int(nil), snapshot.TransitionEntries...), snapshot.PendingEntries...) {
		if entry < 0 {
			return fmt.Errorf("Pool campaign ECL entry %d is negative", entry)
		}
	}
	for index, word := range snapshot.Machine.Memory {
		if index != 0 && word.Address <= snapshot.Machine.Memory[index-1].Address {
			return fmt.Errorf("Pool campaign ECL memory is not strictly ordered at 0x%04X", word.Address)
		}
	}
	for index, word := range snapshot.Machine.Strings {
		if index != 0 && word.Address <= snapshot.Machine.Strings[index-1].Address {
			return fmt.Errorf("Pool campaign ECL strings are not strictly ordered at 0x%04X", word.Address)
		}
	}
	if snapshot.Machine.Random.Draws > randomstream.MaxReplayDraws {
		return fmt.Errorf("Pool campaign random draw count %d exceeds limit", snapshot.Machine.Random.Draws)
	}
	return nil
}

func validateCharacter(character Character) error {
	if character.Gold != 0 {
		return fmt.Errorf("Pool character %q retains legacy gold %d", character.Name, character.Gold)
	}
	if len(character.Name) < 1 || len(character.Name) > 15 {
		return fmt.Errorf("Pool character name length %d, want 1..15", len(character.Name))
	}
	if character.PortraitHead < 1 || character.PortraitHead > 14 || character.PortraitBody < 1 || character.PortraitBody > 12 {
		return fmt.Errorf("Pool character %q has invalid portrait", character.Name)
	}
	if character.IconHead > 13 || character.IconWeapon > 31 || (character.IconSize != 1 && character.IconSize != 2) {
		return fmt.Errorf("Pool character %q has invalid combat icon", character.Name)
	}
	if character.MaxHP < 1 || character.CurrentHP < 0 || character.CurrentHP > character.MaxHP {
		return fmt.Errorf("Pool character %q has invalid HP %d/%d", character.Name, character.CurrentHP, character.MaxHP)
	}
	if len(character.Inventory) > 16 {
		return fmt.Errorf("Pool character %q has %d items, maximum is 16", character.Name, len(character.Inventory))
	}
	for index, item := range character.Inventory {
		if err := validateItem(item); err != nil {
			return fmt.Errorf("Pool character %q item %d: %w", character.Name, index, err)
		}
	}
	return nil
}

func validateItem(item Item) error {
	if len(item.Raw) != 63 {
		return fmt.Errorf("raw record has %d bytes, want 63", len(item.Raw))
	}
	nameLength := int(item.Raw[0])
	if nameLength < 1 || nameLength > 40 || 1+nameLength > len(item.Raw) {
		return fmt.Errorf("raw name length %d is invalid", nameLength)
	}
	if item.Name != string(item.Raw[1:1+nameLength]) {
		return fmt.Errorf("display name does not match raw Pascal string")
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
	raw, err := io.ReadAll(io.LimitReader(handle, 1<<20))
	if err != nil {
		return State{}, fmt.Errorf("read Pool save: %w", err)
	}
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return State{}, fmt.Errorf("decode Pool save header: %w", err)
	}
	if header.Schema == LegacySchema {
		return readLegacyState(raw)
	}
	if header.Schema == PreviousSchema || header.Schema == EarlierSchema || header.Schema == OlderSchema {
		return readPreviousState(raw)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
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

func readPreviousState(raw []byte) (State, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode schema 2 Pool save: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return State{}, fmt.Errorf("schema 2 Pool save has trailing JSON")
	}
	if state.PooledGold < 0 || uint64(state.PooledGold) > uint64(^uint32(0)) {
		return State{}, fmt.Errorf("legacy Pool pooled gold %d is outside uint32", state.PooledGold)
	}
	if state.PooledMoney != ([7]uint32{}) && state.PooledGold != 0 {
		return State{}, fmt.Errorf("legacy Pool save has both pooled_money and pooled_gold")
	}
	if state.PooledMoney == ([7]uint32{}) {
		state.PooledMoney[3] = uint32(state.PooledGold)
	}
	state.PooledGold = 0
	for index := range state.CharacterLibrary {
		if err := migrateCharacterMoney(&state.CharacterLibrary[index]); err != nil {
			return State{}, fmt.Errorf("legacy library character %d: %w", index, err)
		}
	}
	for index := range state.Party {
		if err := migrateCharacterMoney(&state.Party[index]); err != nil {
			return State{}, fmt.Errorf("legacy party character %d: %w", index, err)
		}
	}
	state.Schema = Schema
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	return state, nil
}

func migrateCharacterMoney(character *Character) error {
	if character.Gold < 0 || character.Gold > int(^uint16(0)) {
		return fmt.Errorf("gold %d is outside uint16", character.Gold)
	}
	if character.Money != ([7]uint16{}) && character.Gold != 0 {
		return fmt.Errorf("has both money and gold")
	}
	if character.Money == ([7]uint16{}) {
		character.Money[3] = uint16(character.Gold)
	}
	character.Gold = 0
	return nil
}

type legacyCharacter struct {
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
type legacyState struct {
	Schema           string            `json:"schema"`
	CharacterLibrary []legacyCharacter `json:"character_library"`
	Party            []legacyCharacter `json:"party"`
}

func readLegacyState(raw []byte) (State, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var legacy legacyState
	if err := decoder.Decode(&legacy); err != nil {
		return State{}, fmt.Errorf("decode legacy Pool save: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return State{}, fmt.Errorf("legacy Pool save has trailing JSON")
	}
	convert := func(values []legacyCharacter) ([]Character, error) {
		out := make([]Character, len(values))
		for i, c := range values {
			if c.Gold < 0 || c.Gold > int(^uint16(0)) {
				return nil, fmt.Errorf("legacy character %d gold %d is outside uint16", i, c.Gold)
			}
			money := [7]uint16{}
			money[3] = uint16(c.Gold)
			out[i] = Character{Name: c.Name, RaceID: c.RaceID, GenderID: c.GenderID, ClassID: c.ClassID, AlignmentID: c.AlignmentID, Age: c.Age, Abilities: c.Abilities, ExceptionalStrength: c.ExceptionalStrength, Money: money, MaxHP: c.HP, CurrentHP: c.HP, RawHP: c.RawHP, PortraitHead: c.PortraitHead, PortraitBody: c.PortraitBody, IconHead: c.IconHead, IconWeapon: c.IconWeapon, IconSize: c.IconSize, IconColors: c.IconColors}
		}
		return out, nil
	}
	library, err := convert(legacy.CharacterLibrary)
	if err != nil {
		return State{}, err
	}
	party, err := convert(legacy.Party)
	if err != nil {
		return State{}, err
	}
	state := State{Schema: Schema, CharacterLibrary: library, Party: party}
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	return state, nil
}
