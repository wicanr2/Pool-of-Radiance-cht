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
	Schema         = "pool-remake-state/7"
	FacingSchema   = "pool-remake-state/6"
	PreviousSchema = "pool-remake-state/5"
	EarlierSchema  = "pool-remake-state/4"
	OlderSchema    = "pool-remake-state/3"
	OldestSchema   = "pool-remake-state/2"
	LegacySchema   = "pool-remake-state/1"

	// PartyMaximum 是戰場上的隊伍上限，含 NPC。原版在 overlay-17 entry 9
	// 擋「人數大於 7」，所以是八。
	PartyMaximum = 8
	// PlayerCharacterMaximum 是玩家自己建的角色能佔幾格。
	PlayerCharacterMaximum = 6
	// NPCRecordSize 是 MON*CHA 一筆記錄的大小。
	NPCRecordSize = 285
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
	// NPC 為真代表這一位是 `36h ADD NPC` 加進來的，不是玩家建的
	// （spec 091）。原版隊伍上限八人，玩家角色只佔得了六格。
	NPC bool `json:"npc,omitempty"`
	// Side 是原版記錄的 `+10Eh`：0 與隊伍同一邊，非 0 是另一邊。
	// 有一個 NPC 編號（18h）加進來就是敵方。
	Side uint8 `json:"side,omitempty"`
	// Record 是 NPC 的 285-byte MON*CHA 記錄。戰鬥數值直接讀它，
	// 不硬把 NPC 塞進建角那一套欄位。
	Record []byte `json:"record,omitempty"`
	// ThiefSkills 是記錄 `+77h` 起的八個賊技能百分比（spec 095）。
	// `1Eh CHECKPARTY` 的 `6BA7h` 模式統計的是其中的「找／解陷阱」。
	// remake 還沒有賊技能的產生端，非賊本來就是 0。
	ThiefSkills []uint8 `json:"thief_skills,omitempty"`
	// Memorised 是記憶法術陣列（spec 070，記錄 `+1Fh` 起 13 格）。
	// `3Bh SPELL` 問的就是這個；法術還沒接上來所以目前是空的。
	Memorised []uint8 `json:"memorised,omitempty"`
	// Spellbook 是會的法術編號（spec 110，記錄 `+32h + 編號`）。
	// 記憶畫面只列書上有的；空的代表還沒算過，載入時會補。
	Spellbook []uint8 `json:"spellbook,omitempty"`
	// SpellsToLearn 是法師還沒挑的新法術數。原版在法師等級上升時
	// 讓玩家學一條（overlay-16 `2F79h`），挑的那一支還沒讀出來，
	// 所以 remake 把它記成一次額度，由玩家在法術一覽上按 L 用掉。
	SpellsToLearn int `json:"spells_to_learn,omitempty"`
	// ClassLevels 是八個單一職業的等級（spec 097，記錄 `+96h` 起）。
	// 空的代表「每個組成職業都是第 1 級」，所以舊存檔與剛建好的角色照讀。
	ClassLevels []uint8 `json:"class_levels,omitempty"`
	// Experience 是累積經驗值（spec 097，記錄 `+0ACh`／`+0AEh` 的 32 bit）。
	// 舊存檔沒有這個欄位，讀回來是 0，與「還沒打過任何一場」同義。
	Experience uint32 `json:"experience,omitempty"`
	// Effects 是掛在身上的效果碼（spec 069 的串列，記錄 `+7Fh` 起）。
	// `1Eh CHECKPARTY` 的效果模式與神殿的失明／疾病／中毒／詛咒
	//（spec 115）問的都是這一串。法術還沒接上來，所以目前一律是空的
	// ——空的是正確答案，不是佔位。
	Effects []uint8 `json:"effects,omitempty"`
}

type Campaign struct {
	MapArchive uint8                      `json:"map_archive"`
	MapBlock   uint8                      `json:"map_block"`
	ECLArchive uint8                      `json:"ecl_archive"`
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
	// 原版的隊伍上限是八（overlay-17 entry 9 擋人數大於 7），但玩家自己
	// 建的角色只佔得了六格，多出來的兩格留給 `36h ADD NPC` 的 NPC。
	if len(state.Party) > PartyMaximum {
		return fmt.Errorf("Pool party has %d characters, maximum is %d", len(state.Party), PartyMaximum)
	}
	players := 0
	for _, character := range state.Party {
		if !character.NPC {
			players++
		}
	}
	if players > PlayerCharacterMaximum {
		return fmt.Errorf("Pool party has %d player characters, maximum is %d",
			players, PlayerCharacterMaximum)
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
		if character.NPC {
			// NPC 不進角色庫：它是劇情加進來的，不是玩家建的，
			// 也不該出現在「加入隊伍」的清單裡。
			continue
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
	if campaign.ECLArchive < 1 || campaign.ECLArchive > 8 {
		return fmt.Errorf("Pool campaign ECL archive %d is outside 1..8", campaign.ECLArchive)
	}
	if campaign.X > 15 || campaign.Y > 15 {
		return fmt.Errorf("Pool campaign position (%d,%d) is outside 16x16 map", campaign.X, campaign.Y)
	}
	// 0 北、1 東、2 南、3 西（spec 076）。原版 35 個位置只出現這四個值。
	if campaign.Facing > 3 {
		return fmt.Errorf("Pool campaign facing %d is outside 0..3", campaign.Facing)
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
	if character.NPC {
		// NPC 沒有走過建角，肖像與戰鬥造形不在那些範圍裡；它帶的是原版的
		// 整筆記錄，戰鬥數值從那裡讀。
		if len(character.Record) != NPCRecordSize {
			return fmt.Errorf("Pool NPC %q has a %d-byte record, want %d",
				character.Name, len(character.Record), NPCRecordSize)
		}
		return validateInventory(character)
	}
	if len(character.Record) != 0 {
		return fmt.Errorf("Pool character %q is not an NPC but carries a %d-byte record",
			character.Name, len(character.Record))
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
	return validateInventory(character)
}

func validateInventory(character Character) error {
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
	if header.Schema == FacingSchema {
		return readFacingSchemaState(raw)
	}
	if header.Schema == PreviousSchema || header.Schema == EarlierSchema || header.Schema == OlderSchema || header.Schema == OldestSchema {
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

// readFacingSchemaState 讀 schema 6。那個版本把隊伍朝向存成共用 engine 的
// 0/2/4/6，除以 2 就回到原版的 0..3（spec 076）。
func readFacingSchemaState(raw []byte) (State, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode schema 6 Pool save: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return State{}, fmt.Errorf("schema 6 Pool save has trailing JSON")
	}
	if state.Campaign != nil {
		if state.Campaign.Facing > 6 || state.Campaign.Facing%2 != 0 {
			return State{}, fmt.Errorf("schema 6 Pool campaign facing %d is not one of 0, 2, 4, 6", state.Campaign.Facing)
		}
		state.Campaign.Facing /= 2
	}
	state.Schema = Schema
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
	if state.Campaign != nil && state.Campaign.ECLArchive == 0 {
		state.Campaign.ECLArchive = state.Campaign.MapArchive
	}
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
	if state.Campaign != nil {
		if state.Campaign.Facing > 6 || state.Campaign.Facing%2 != 0 {
			return State{}, fmt.Errorf("legacy Pool campaign facing %d is not one of 0, 2, 4, 6", state.Campaign.Facing)
		}
		state.Campaign.Facing /= 2
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
