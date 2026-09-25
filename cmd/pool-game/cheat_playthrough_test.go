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
	// 戰鬥機制與 AI 的累計（#57）：每一場的 `tacticalState.Activity` 在換場時加進來。
	battle   *tacticalState
	battles  int
	activity combatActivity
}

// finishBattle 把目前這一場的計數加進累計。
func (r *cheatPlaythrough) finishBattle() {
	if r.battle != nil {
		r.activity.add(r.battle.Activity)
		r.battle = nil
	}
}

// observe 在每一次 `Update()` 之後看目前的 ECL 區塊；換了就記一筆。
func (r *cheatPlaythrough) observe(a *app) {
	if a.tactical != r.battle {
		r.finishBattle()
		if a.tactical != nil {
			r.battle = a.tactical
			r.battles++
		}
	}
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
	recorder.finishBattle()
	// #57 的關閉標準（使用者 2026-09-26）：開作弊通關也算對拍，前提是遊戲機制與 AI 都有作用。
	// 鎖 HP 是每個 tick 結束才寫回、一擊斃命只改隊伍命中後的傷害，所以敵方照樣挑人、
	// 走位、出手、命中，隊伍照樣要擲命中。這幾項是這條路線一定會發生的，當閘門；
	// 施法與反應攻擊要看路上遇到誰，只記錄。
	got := recorder.activity
	gates := []struct {
		name  string
		value int
	}{
		{"battles", recorder.battles},
		{"foe attacks", got.FoeAttacks},
		{"foe hits", got.FoeHits},
		{"foe steps", got.FoeSteps},
		{"party attacks", got.PartyAttacks},
		{"party hits", got.PartyHits},
	}
	for _, gate := range gates {
		if gate.value == 0 {
			t.Errorf("the cheat run shows no %s: combat mechanics or AI did not act (%+v, %d battles)",
				gate.name, got, recorder.battles)
		}
	}
	if got.PartyHits >= got.PartyAttacks {
		t.Errorf("every party attack hit (%d/%d): hit rolls are being skipped, not just damage", got.PartyHits, got.PartyAttacks)
	}
	t.Logf("mechanics: %d battles %+v", recorder.battles, got)
	receipt := map[string]any{
		"schema":      "pool-remake-cheat-playthrough/2",
		"test":        "TestMainlineProbeCheatMenuToEnding",
		"route":       "spec 137 路線 (a)，house rule，dice seed 142",
		"cheats":      "spec 141：建好隊伍就開鎖 HP 與一擊斃命，全程不關",
		"mechanics": map[string]any{
			"note": "作弊之下戰鬥機制與 AI 照樣作用的證據（#57）。battles 起到 party_hits 是閘門，其餘只記錄。",
			"battles": recorder.battles, "foe_attacks": got.FoeAttacks, "foe_hits": got.FoeHits,
			"foe_steps": got.FoeSteps, "party_attacks": got.PartyAttacks, "party_hits": got.PartyHits,
			"foe_casts": got.FoeCasts, "foe_casts_begun": got.FoeCastsBegun,
			"party_casts": got.PartyCasts, "reaction_attacks": got.ReactionAttacks,
		},
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
