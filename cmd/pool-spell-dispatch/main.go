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
	HookAddress   string  `json:"hook_address"`
	SlotCount     int     `json:"slot_count"`
	DistinctCount int     `json:"distinct_handler_count"`
	Slots         []slot  `json:"slots"`
	Shared        []group `json:"shared_handlers"`
}

type slot struct {
	SpellID    int    `json:"spell_id"`
	Name       string `json:"name,omitempty"`
	StubOffset string `json:"stub_offset"`
	EntryIndex int    `json:"entry_index"`
	CodeOffset string `json:"code_offset"`
	InitOffset string `json:"init_offset"`
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
		HookAddress:  fmt.Sprintf("%#04x", gamepack.SpellDispatchHookAddress),
		SlotCount:    len(table),
	}
	distinct := make(map[uint16]struct{}, len(table))
	for _, entry := range table {
		distinct[entry.CodeOffset] = struct{}{}
		r.Slots = append(r.Slots, slot{
			SpellID:    entry.SpellID,
			Name:       nameOf(entry.SpellID),
			StubOffset: fmt.Sprintf("%#04x", entry.StubOffset),
			EntryIndex: entry.EntryIndex,
			CodeOffset: fmt.Sprintf("%#04x", entry.CodeOffset),
			InitOffset: fmt.Sprintf("%#04x", entry.InitOffset),
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

func sortGroups(groups []group) {
	for i := 1; i < len(groups); i++ {
		for j := i; j > 0 && groups[j].SpellIDs[0] < groups[j-1].SpellIDs[0]; j-- {
			groups[j], groups[j-1] = groups[j-1], groups[j]
		}
	}
}
