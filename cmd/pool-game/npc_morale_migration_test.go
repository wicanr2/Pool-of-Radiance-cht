package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// #74 之前的存檔：隊伍 NPC 的 `+84h` 還是怪物檔的 FFh，讀檔時補回原版 ADD NPC 的值。
// 已經補過的、不是 NPC 的都不動。
func TestMigrateNPCMoraleRestoresTheAddNPCValue(t *testing.T) {
	a, err := newApp(dosZIPForTests, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	npc := func(archive, block uint8, morale byte) poolsave.Character {
		record, err := a.loadMonster(archive, block)
		if err != nil {
			t.Fatal(err)
		}
		raw := append([]byte(nil), record.Raw[:]...)
		raw[gamepack.MoraleOffset] = morale
		return poolsave.Character{Name: a.monsterText.Translate(record.Name), NPC: true, Record: raw}
	}
	a.state.Party = []poolsave.Character{
		{Name: "HERO", ClassID: "fighter"},
		npc(3, 107, 0xFF),                           // ecl3/0 A046h：士氣 99
		npc(3, gamepack.NPCMercenaryTiers[2], 0xFF), // 雇傭兵等級 2：2 × 5 + 50 = 60
		npc(4, 27, gamepack.NPCMoraleByte(100)),     // 已經是新值：不動
	}
	a.migrateNPCMorale()
	for index, want := range map[int]byte{1: gamepack.NPCMoraleByte(99), 2: gamepack.NPCMoraleByte(60), 3: gamepack.NPCMoraleByte(100)} {
		if got := a.state.Party[index].Record[gamepack.MoraleOffset]; got != want {
			t.Errorf("party %d morale byte %02X, want %02X", index, got, want)
		}
	}
	if a.state.Party[0].Record != nil {
		t.Fatal("a created character gained a record")
	}
}
