// Command pool-city-hall-audit derives a reproducible structural inventory of
// the City Hall reward and commission loops from the original block-8 trace.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	wantSchema    = "pool-ecl-static-trace-v1"
	wantBlockHash = "fd446439973994d9f887b8628b512054a822369eb4360769b21eb2224982f012"
	codeBase      = 0x9900
)

type operand struct {
	Code    uint8  `json:"code"`
	Low     uint8  `json:"low"`
	Word    uint16 `json:"word,omitempty"`
	WordSet bool   `json:"word_set,omitempty"`
	Text    string `json:"text,omitempty"`
}
type instruction struct {
	Offset   int       `json:"offset"`
	Address  string    `json:"address"`
	Opcode   uint8     `json:"opcode"`
	Name     string    `json:"name"`
	Text     string    `json:"text,omitempty"`
	Operands []operand `json:"operands,omitempty"`
}
type edge struct {
	From int    `json:"From"`
	To   int    `json:"To"`
	Kind string `json:"Kind"`
}
type trace struct {
	Schema          string        `json:"schema"`
	ZIPSHA256       string        `json:"zip_sha256"`
	Member          string        `json:"member"`
	MemberSHA256    string        `json:"member_sha256"`
	BlockID         uint8         `json:"block_id"`
	BlockSHA256     string        `json:"block_sha256"`
	CodeAddressBase string        `json:"code_address_base"`
	Instructions    []instruction `json:"instructions"`
	Edges           []edge        `json:"edges"`
}
type dispatch struct {
	Address       string   `json:"address"`
	SlotCount     int      `json:"slot_count"`
	Targets       []string `json:"targets"`
	UniqueTargets []string `json:"unique_targets"`
}
type externalCall struct {
	Address string `json:"address"`
	Opcode  uint8  `json:"opcode"`
	Name    string `json:"name"`
}
type completionNotification struct {
	Index              int    `json:"index"`
	StateAddress       string `json:"state_address"`
	Target             string `json:"target"`
	Text               string `json:"text"`
	IncrementsProgress bool   `json:"increments_4ac1"`
	IncrementAddress   string `json:"increment_address,omitempty"`
}
type output struct {
	Schema             string                   `json:"schema"`
	SourceSchema       string                   `json:"source_schema"`
	SourceBlockSHA256  string                   `json:"source_block_sha256"`
	SourceMemberSHA256 string                   `json:"source_member_sha256"`
	CodeAddressBase    string                   `json:"code_address_base"`
	RewardDispatch     dispatch                 `json:"reward_dispatch"`
	CommissionDispatch dispatch                 `json:"commission_dispatch"`
	ProgressProducers  []string                 `json:"progress_4ac1_increment_producers"`
	CompletionTable    []completionNotification `json:"completion_notification_table"`
	ExternalCalls      []externalCall           `json:"external_calls"`
}

func main() {
	in := flag.String("in", "docs/audit/dos-ecl3-block8-trace.json", "block-8 static trace")
	out := flag.String("out", "", "JSON output; stdout when empty")
	flag.Parse()
	raw, err := os.ReadFile(*in)
	if err != nil {
		fatal(err)
	}
	var tr trace
	if err := json.Unmarshal(raw, &tr); err != nil {
		fatal(err)
	}
	r, err := audit(tr)
	if err != nil {
		fatal(err)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fatal(err)
	}
	b = append(b, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(b)
	} else {
		err = os.WriteFile(*out, b, 0o644)
	}
	if err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

func audit(tr trace) (output, error) {
	if tr.Schema != wantSchema || tr.BlockID != 8 || tr.BlockSHA256 != wantBlockHash || tr.CodeAddressBase != "0x9900" {
		return output{}, fmt.Errorf("unexpected trace identity: schema=%q block=%d hash=%q base=%q", tr.Schema, tr.BlockID, tr.BlockSHA256, tr.CodeAddressBase)
	}
	reward, commission := dispatchAt(tr, 0x9C83), dispatchAt(tr, 0xA85B)
	if reward.SlotCount != 7 {
		return output{}, fmt.Errorf("reward dispatch has %d slots, want 7", reward.SlotCount)
	}
	if commission.SlotCount != 16 {
		return output{}, fmt.Errorf("commission dispatch has %d slots, want 16", commission.SlotCount)
	}
	completionDispatch := dispatchAt(tr, 0x9D63)
	if completionDispatch.SlotCount != 26 {
		return output{}, fmt.Errorf("completion dispatch has %d slots, want 26", completionDispatch.SlotCount)
	}
	producers := make([]string, 0, 10)
	externals := make([]externalCall, 0)
	for _, ins := range tr.Instructions {
		if isIncrementOf(ins, 0x4AC1) {
			producers = append(producers, ins.Address)
		}
		if ins.Offset >= 0x2B8 && ins.Offset <= 0x167C && (ins.Opcode == 0x1D || ins.Opcode == 0x24 || ins.Opcode == 0x27) {
			externals = append(externals, externalCall{Address: ins.Address, Opcode: ins.Opcode, Name: ins.Name})
		}
	}
	if len(producers) != 10 {
		return output{}, fmt.Errorf("4AC1 increment producer count=%d, want 10", len(producers))
	}
	completionTable, err := buildCompletionTable(tr, completionDispatch)
	if err != nil {
		return output{}, err
	}
	if len(externals) == 0 {
		return output{}, errors.New("no external service calls found in City Hall scope")
	}
	return output{Schema: "pool-city-hall-structural-audit-v1", SourceSchema: tr.Schema, SourceBlockSHA256: tr.BlockSHA256,
		SourceMemberSHA256: tr.MemberSHA256, CodeAddressBase: tr.CodeAddressBase, RewardDispatch: reward,
		CommissionDispatch: commission, ProgressProducers: producers, CompletionTable: completionTable, ExternalCalls: externals}, nil
}

func buildCompletionTable(tr trace, dispatch dispatch) ([]completionNotification, error) {
	byAddress := make(map[string]int, len(tr.Instructions))
	for index, ins := range tr.Instructions {
		byAddress[ins.Address] = index
	}
	result := make([]completionNotification, len(dispatch.Targets))
	for slot, target := range dispatch.Targets {
		start, ok := byAddress[target]
		if !ok {
			return nil, fmt.Errorf("completion target %s is absent from trace", target)
		}
		row := completionNotification{Index: slot, StateAddress: fmt.Sprintf("0x%04X", 0x4AA6+slot), Target: target}
		end := len(tr.Instructions)
		if slot+1 < len(dispatch.Targets) {
			if next, found := byAddress[dispatch.Targets[slot+1]]; found && next > start {
				end = next
			}
		}
		for index := start; index < end; index++ {
			ins := tr.Instructions[index]
			if row.Text == "" && (ins.Name == "PRINT" || ins.Name == "PRINTCLEAR") && len(ins.Operands) != 0 {
				row.Text = ins.Operands[0].Text
			}
			if isIncrementOf(ins, 0x4AC1) {
				row.IncrementsProgress = true
				row.IncrementAddress = ins.Address
			}
			if slot+1 == len(dispatch.Targets) && ins.Name == "RETURN" && row.Text != "" {
				break
			}
		}
		result[slot] = row
	}
	return result, nil
}

func dispatchAt(tr trace, address int) dispatch {
	from := address - codeBase
	targets := make([]string, 0)
	seen := map[int]bool{}
	unique := make([]string, 0)
	for _, e := range tr.Edges {
		if e.From != from || e.Kind != "ON GOSUB" {
			continue
		}
		targets = append(targets, fmt.Sprintf("0x%04X", codeBase+e.To))
		if !seen[e.To] {
			seen[e.To] = true
			unique = append(unique, fmt.Sprintf("0x%04X", codeBase+e.To))
		}
	}
	return dispatch{Address: fmt.Sprintf("0x%04X", address), SlotCount: len(targets), Targets: targets, UniqueTargets: unique}
}
func isIncrementOf(ins instruction, address uint16) bool {
	if ins.Opcode != 0x04 || len(ins.Operands) != 3 {
		return false
	}
	a, b, c := ins.Operands[0], ins.Operands[1], ins.Operands[2]
	return a.Code == 0 && a.Low == 1 && b.WordSet && b.Word == address && c.WordSet && c.Word == address
}
