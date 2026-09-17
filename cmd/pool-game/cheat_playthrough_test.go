package main

// #5 的主線收據（使用者 2026-09-17 決定的口徑，CLAUDE.md §3）：開產品的作弊選單（spec 141），
// 從標題以正常按鍵跑路線 (a) 到結局。每換一個 ECL 區塊記一筆 checkpoint，通過時寫成
// `docs/audit/remake-cheat-playthrough.json`，給原版那一側（dosgolem）逐段對照。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// afterTick 是每次 `Update()` 之後的測試鉤子；nil 就什麼都不做。探針的 checkpoint 記錄器與
// 一擊斃命的觀察器用它。
var afterTick func(*app)

// runAfterTick 在呼叫 `Update()` 的測試 helper 裡叫。
func runAfterTick(a *app) {
	if afterTick != nil {
		afterTick(a)
	}
}

// probeCheatWholeRun 為真時，探針建好隊伍就開兩個作弊開關，全程不關。
var probeCheatWholeRun = false

// cheatCheckpointFlags 是每一筆 checkpoint 帶的旗標（spec 137 路線 (a) 的段落旗標）。
var cheatCheckpointFlags = []uint16{0x4ABB, 0x4AC1, 0x4AB0, 0x4AA6, 0x4AA7, 0x4A24, 0x4AC4, 0x4A77, 0x4A78, 0x4ABA}

type cheatCheckpoint struct {
	Sequence   int               `json:"sequence"`
	ECLArchive int               `json:"ecl_archive"`
	ECLBlock   int               `json:"ecl_block"`
	Map        string            `json:"map"`
	X          int               `json:"x"`
	Y          int               `json:"y"`
	Clock      string            `json:"clock"`
	Flags      map[string]string `json:"flags"`
	Slots      map[string]string `json:"slots,omitempty"`
}

type cheatPlaythrough struct {
	checkpoints []cheatCheckpoint
	last        [2]int
}

// observe 在每一次 `Update()` 之後看目前的 ECL 區塊；換了就記一筆。
func (r *cheatPlaythrough) observe(a *app) {
	if a.eventSession == nil || a.eventMachine == nil {
		return
	}
	key := [2]int{int(a.eclArchive), int(a.eventSession.CurrentBlockID())}
	if len(r.checkpoints) != 0 && key == r.last {
		return
	}
	r.last = key
	memory := a.eventMachine.Memory
	flags := map[string]string{}
	for _, address := range cheatCheckpointFlags {
		flags[fmt.Sprintf("%04X", address)] = fmt.Sprintf("%02X", memory[address])
	}
	slots := map[string]string{}
	for address := uint16(0x4AA6); address <= 0x4ABF; address++ {
		if value := memory[address]; value == 0xFE || value == 0xFF {
			slots[fmt.Sprintf("%04X", address)] = fmt.Sprintf("%02X", value)
		}
	}
	r.checkpoints = append(r.checkpoints, cheatCheckpoint{
		Sequence: len(r.checkpoints), ECLArchive: key[0], ECLBlock: key[1],
		Map: fmt.Sprintf("GEO%d/%d", a.spawn.Map.Archive, a.spawn.Map.BlockID), X: int(a.spawn.X), Y: int(a.spawn.Y),
		Clock: fmt.Sprintf("%v", a.gameTime), Flags: flags, Slots: slots,
	})
}

// TestMainlineProbeCheatMenuToEnding 是 #5 的收據：house rule、seed 142、路線 (a)，建好隊伍就用
// F6 作弊選單打開鎖 HP 與一擊斃命，全程不關，跑到結局。
func TestMainlineProbeCheatMenuToEnding(t *testing.T) {
	recorder := &cheatPlaythrough{}
	previous := afterTick
	afterTick = recorder.observe
	defer func() { afterTick = previous }()
	probeRouteA, probeCheatWholeRun = true, true
	defer func() { probeRouteA, probeCheatWholeRun = false, false }()
	runMainlineProbe(t, true, 142)
	if t.Failed() {
		return
	}
	receipt := map[string]any{
		"schema":      "pool-remake-cheat-playthrough/1",
		"test":        "TestMainlineProbeCheatMenuToEnding",
		"route":       "spec 137 路線 (a)，house rule，dice seed 142",
		"cheats":      "spec 141：建好隊伍就開鎖 HP 與一擊斃命，全程不關",
		"checkpoints": recorder.checkpoints,
	}
	data, err := json.MarshalIndent(receipt, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "docs", "audit", "remake-cheat-playthrough.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d checkpoints to %s", len(recorder.checkpoints), path)
}
