package main

// 怪物的 THAC0 來源（#31／#36，spec 063）：開打時 overlay-10 `1380h` 對每一個 combatant 跑
// overlay-25 entry 7，`+110h = +2Dh`（沒武器再加力量修正）；MONnCHA 樣板的 `+110h` 是殘值。
// 兩筆證據釘在這裡：諾里斯的樣板 154 對回 `+2Dh` 的 45，獸人家開打那一幀 dosgolem 讀到的
// 每一隻 `+110h` 與 remake 擺上戰場的 THAC0 逐隻相同。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func TestNorrisTHAC0ReadsTheBaseFieldNotTheTemplate(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	norris, err := gamepack.ReadDOSMonsterRecord(zipPath, 8, 32)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if norris.Name != "NORRIS THE GRAY" || norris.Raw[0x110] != 154 || norris.Raw[0x2D] != 45 {
		t.Fatalf("MON8CHA/32 is %q with +110h=%d +2Dh=%d", norris.Name, norris.Raw[0x110], norris.Raw[0x2D])
	}
	got, err := norris.CombatThac0Internal()
	if err != nil {
		t.Fatal(err)
	}
	// 45 是 +2Dh；他的力量 +10h 決定有沒有修正——這裡把兩個都釘住，改了資料就會開口。
	index, err := gamepack.StrengthTableIndex(int(norris.Raw[0x10]), int(norris.Raw[0x16]))
	if err != nil {
		t.Fatal(err)
	}
	want := 45
	if norris.Raw[0xAA] != 0 {
		want += gamepack.StrengthHitAdjustment(index)
	}
	if int(got) != want {
		t.Fatalf("Norris THAC0 internal %d, want %d (+2Dh 45, strength %d)", got, want, norris.Raw[0x10])
	}
	if got == 154 || got < 20 || got > 60 {
		t.Fatalf("Norris THAC0 internal %d is not a sane value", got)
	}
	// 正對照：蜥蜴人樣板 +110h 與 +2Dh 都是 44，兩條路要給同一個答案。
	lizard, err := gamepack.ReadDOSMonsterRecord(zipPath, 8, 57)
	if err != nil {
		t.Fatal(err)
	}
	control, err := lizard.CombatThac0Internal()
	if err != nil {
		t.Fatal(err)
	}
	if lizard.Name != "LIZARDMAN" || control != 44 || lizard.Raw[0x110] != 44 {
		t.Fatalf("lizardman %q THAC0 internal %d (template %d), want 44", lizard.Name, control, lizard.Raw[0x110])
	}
	t.Logf("Norris: template +110h 154 → combat THAC0 internal %d (surface %d); lizardman 44", got, 60-int(got))
}

func TestMonsterTHAC0MatchesTheOrcHomeRuntimeReceipt(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-monster-thac0-runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		TemplateOrc int `json:"template_orc_110h"`
		Records     []struct {
			Index int    `json:"index"`
			Name  string `json:"name"`
			THAC0 int    `json:"thac0_110h"`
			Base  int    `json:"base_2Dh"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	walk, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-deployment-peek-orc-home.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fight struct {
		PartyCell struct{ X, Y, Facing uint8 } `json:"party_cell"`
		Spawns    [][3]uint8                   `json:"spawns"`
	}
	if err := json.Unmarshal(walk, &fight); err != nil {
		t.Fatal(err)
	}
	application := newDeploymentFixture(t, zipPath, fight.PartyCell.X, fight.PartyCell.Y, fight.PartyCell.Facing, fight.Spawns, false)
	state := application.tactical
	checked := 0
	for _, record := range receipt.Records {
		if record.Index <= 5 {
			continue // 隊員的力量與收據那一隊不同，只對敵方
		}
		if record.Index >= len(state.Roster) || state.Friendly[record.Index] {
			t.Fatalf("receipt combatant %d (%s) is not a foe on the remake board", record.Index, record.Name)
		}
		if int(state.THAC0[record.Index]) != record.THAC0 {
			t.Errorf("combatant %d %s: remake THAC0 internal %d, original runtime +110h %d (+2Dh %d)",
				record.Index, record.Name, state.THAC0[record.Index], record.THAC0, record.Base)
		}
		checked++
	}
	if checked != 20 {
		t.Fatalf("checked %d foes, want 20", checked)
	}
	t.Logf("20 foes: remake THAC0 == original runtime +110h (orc template +110h was %d, runtime 41)", receipt.TemplateOrc)
}
