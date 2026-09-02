// pool-spell-dispatch 把 overlay-22 的法術效果派發表 dump 成 JSON，
// 供 spec 073 引用，也當作後續逐支解讀處理常式的工作清單。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

type report struct {
	Schema        string  `json:"schema"`
	Overlay       int     `json:"overlay"`
	InitEntry     int     `json:"init_entry"`
	CallEntry     int     `json:"call_entry"`
	TableAddress  string  `json:"table_address"`
	ParameterAddress string `json:"parameter_table_address"`
	HookAddress   string  `json:"hook_address"`
	SlotCount     int     `json:"slot_count"`
	DistinctCount int     `json:"distinct_handler_count"`
	Slots         []slot  `json:"slots"`
	Shared        []group `json:"shared_handlers"`
}

type slot struct {
	SpellID          int    `json:"spell_id"`
	Name             string `json:"name,omitempty"`
	Message          string `json:"message,omitempty"`
	StubOffset       string `json:"stub_offset"`
	EntryIndex       int    `json:"entry_index"`
	CodeOffset       string `json:"code_offset"`
	InitOffset       string `json:"init_offset"`
	Parameters       string `json:"parameters"`
	RequiresAttack   bool   `json:"requires_attack_roll"`
	SaveRule         uint8  `json:"save_rule"`
	SaveCategory     uint8  `json:"save_category"`
	EffectCode       string `json:"effect_code"`
}

type group struct {
	CodeOffset string `json:"code_offset"`
	SpellIDs   []int  `json:"spell_ids"`
	Names      []string `json:"names,omitempty"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "original DOS ZIP")
	outPath := flag.String("out", "", "JSON output; stdout when empty")
	flag.Parse()
	if err := run(*zipPath, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(zipPath, outPath string) error {
	table, err := gamepack.ReadDOSSpellDispatchTable(zipPath)
	if err != nil {
		return err
	}
	names, err := gamepack.ReadDOSSpellNames(zipPath)
	if err != nil {
		return err
	}
	parameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		return err
	}
	nameOf := func(id int) string {
		if id < 1 || id > len(names) {
			return ""
		}
		return names[id-1]
	}
	r := report{
		Schema:       "pool-spell-dispatch-v1",
		Overlay:      gamepack.SpellDispatchOverlay,
		InitEntry:    gamepack.SpellDispatchInitEntry,
		CallEntry:    gamepack.SpellDispatchCallEntry,
		TableAddress: fmt.Sprintf("%#04x", gamepack.SpellDispatchTableAddress),
		ParameterAddress: fmt.Sprintf("%#04x", gamepack.SpellParameterTableAddress),
		HookAddress:  fmt.Sprintf("%#04x", gamepack.SpellDispatchHookAddress),
		SlotCount:    len(table),
	}
	distinct := make(map[uint16]struct{}, len(table))
	for _, entry := range table {
		distinct[entry.CodeOffset] = struct{}{}
		record := parameters[entry.SpellID]
		r.Slots = append(r.Slots, slot{
			SpellID:        entry.SpellID,
			Name:           nameOf(entry.SpellID),
			Message:        entry.Message,
			StubOffset:     fmt.Sprintf("%#04x", entry.StubOffset),
			EntryIndex:     entry.EntryIndex,
			CodeOffset:     fmt.Sprintf("%#04x", entry.CodeOffset),
			InitOffset:     fmt.Sprintf("%#04x", entry.InitOffset),
			Parameters:     hexBytes(record.Raw[:]),
			RequiresAttack: record.RequiresAttackRoll(),
			SaveRule:       record.SaveRule(),
			SaveCategory:   uint8(record.SaveCategory()),
			EffectCode:     fmt.Sprintf("%#02x", record.EffectCode()),
		})
	}
	r.DistinctCount = len(distinct)
	for handler, ids := range gamepack.SpellDispatchGroups(table) {
		g := group{CodeOffset: fmt.Sprintf("%#04x", handler), SpellIDs: ids}
		for _, id := range ids {
			if name := nameOf(id); name != "" {
				g.Names = append(g.Names, name)
			}
		}
		r.Shared = append(r.Shared, g)
	}
	sortGroups(r.Shared)
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if outPath == "" {
		_, err = os.Stdout.Write(encoded)
		return err
	}
	return os.WriteFile(outPath, encoded, 0o644)
}

// hexBytes 把整筆記錄印成十六進位，讓還沒解讀的欄位也留在證據裡。
func hexBytes(raw []byte) string {
	out := make([]byte, 0, len(raw)*3)
	const digits = "0123456789abcdef"
	for index, value := range raw {
		if index > 0 {
			out = append(out, ' ')
		}
		out = append(out, digits[value>>4], digits[value&0xf])
	}
	return string(out)
}

func sortGroups(groups []group) {
	for i := 1; i < len(groups); i++ {
		for j := i; j > 0 && groups[j].SpellIDs[0] < groups[j-1].SpellIDs[0]; j-- {
			groups[j], groups[j-1] = groups[j-1], groups[j]
		}
	}
}
