package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 一級時五格要與 spec 031 的三份原版 `.CHA` 相同，升級之後要跟著變（#61）。
func TestLivePartyStrengthFollowsTheCurrentParty(t *testing.T) {
	a, err := newApp(dosZIPForTests, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	member := func(class string, str, dex, hp int) poolsave.Character {
		var abilities [6]int
		abilities[0], abilities[3] = str, dex
		return poolsave.Character{Name: class, ClassID: class, Abilities: abilities, MaxHP: hp, CurrentHP: hp}
	}
	anchors := []struct {
		member poolsave.Character
		want   character.PartyStrengthRecord
	}{
		{member("fighter", 14, 13, 7), character.PartyStrengthRecord{Field110: 40, Field111: 50, Field11B: 7}},
		{member("magic-user", 17, 14, 2), character.PartyStrengthRecord{Field9B: 1, Field110: 41, Field111: 50, Field11B: 2}},
		{member("thief", 13, 16, 4), character.PartyStrengthRecord{Field110: 40, Field111: 52, Field11B: 4}},
	}
	for _, anchor := range anchors {
		got, err := a.partyStrengthRecord(anchor.member)
		if err != nil {
			t.Fatalf("%s: %v", anchor.member.ClassID, err)
		}
		if got != anchor.want {
			t.Fatalf("%s: record = %+v, want the original .CHA %+v", anchor.member.ClassID, got, anchor.want)
		}
	}

	fighter := member("fighter", 14, 13, 7)
	a.state.Party = []poolsave.Character{fighter}
	levelOne, err := a.livePartyStrength()
	if err != nil {
		t.Fatal(err)
	}
	// 訓練到 7 級、HP 60：THAC0 與 HP 都要算進去，而不是停在建 session 那一刻。
	slot, ok := creation.ComponentClassIndex("fighter")
	if !ok {
		t.Fatal("fighter has no class index")
	}
	levels := make([]uint8, 8)
	levels[slot] = 7
	a.state.Party[0].ClassLevels = levels
	a.state.Party[0].MaxHP, a.state.Party[0].CurrentHP = 60, 60
	trained, err := a.livePartyStrength()
	if err != nil {
		t.Fatal(err)
	}
	record, err := a.partyStrengthRecord(a.state.Party[0])
	if err != nil {
		t.Fatal(err)
	}
	if record.Field110 <= 40 || record.Field11B != 60 {
		t.Fatalf("level 7 fighter record = %+v, want stored attack above 40 and HP 60", record)
	}
	if want := character.PartyStrength([]character.PartyStrengthRecord{record}); trained != want || trained <= levelOne {
		t.Fatalf("party strength level 1 = %d, level 7 = %d (want %d, and larger)", levelOne, trained, want)
	}
}
