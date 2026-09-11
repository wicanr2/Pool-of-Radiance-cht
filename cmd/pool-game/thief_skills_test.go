package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 建角走完之後，賊的八格技能要跟原版逐格相同（spec 095）。
//
// 這一條走的是玩家真的會走的那條路（finishCharacter），不是直接叫
// gamepack.Build——`internal/gamepack` 的測試證明算式對，這一條證明那條算式
// 真的接在建角的出口上。少了它，算式綠、玩家建出來的賊技能仍然是空的。
func TestFinishingANewThiefWritesTheOriginalSkills(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	for _, want := range []struct {
		name    string
		raceID  string
		classID string
		skills  []uint8
	}{
		{"矮人賊", "dwarf", "thief", []uint8{35, 45, 40, 15, 10, 10, 75, 0}},
		{"人類賊", "human", "thief", []uint8{35, 35, 25, 15, 10, 10, 85, 0}},
	} {
		application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
		if err != nil {
			t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
		}
		application.saveState = func(poolsave.State) error { return nil }
		application.exportDOSCharacter = nil
		_, raceIndex, ok := findRaceIndex(want.raceID)
		if !ok {
			t.Fatalf("種族 %q 不在名錄裡", want.raceID)
		}
		classIndex := -1
		for index, choice := range creation.ClassesForRace(want.raceID) {
			if choice.ID == want.classID {
				classIndex = index
			}
		}
		if classIndex < 0 {
			t.Fatalf("%q 開不出 %q", want.raceID, want.classID)
		}
		application.flow = creation.Flow{Stage: creation.StageIconConfirm,
			RaceIndex: raceIndex, ClassIndex: classIndex, Name: want.name}
		rolled := creation.RollCharacter(application.roller,
			application.flow.SelectedRace(), application.flow.SelectedGender(),
			application.flow.SelectedClass())
		application.rolled = &rolled
		if err := application.finishCharacter(); err != nil {
			t.Fatalf("%s 建角：%v", want.name, err)
		}
		if len(application.state.CharacterLibrary) != 1 {
			t.Fatalf("%s 沒有進人物名單：%v", want.name, application.state.CharacterLibrary)
		}
		got := application.state.CharacterLibrary[0].ThiefSkills
		if len(got) != len(want.skills) {
			t.Fatalf("%s 的技能有 %d 格，原版是 %d 格", want.name, len(got), len(want.skills))
		}
		for index := range want.skills {
			if got[index] != want.skills[index] {
				t.Errorf("%s 第 %d 格是 %d，原版是 %d（整份 %v）",
					want.name, index, got[index], want.skills[index], got)
			}
		}
	}
}

// 非賊建出來是空的，不是八個 0——匯出 `.CHA` 時空的才會讓記錄裡原本的
// 位元組留著。這一條擋住「順手都填一填」把 NPC 的技能抹掉。
func TestFinishingANewFighterLeavesTheThiefSkillsEmpty(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.saveState = func(poolsave.State) error { return nil }
	application.exportDOSCharacter = nil
	_, raceIndex, ok := findRaceIndex("human")
	if !ok {
		t.Fatal("human 不在名錄裡")
	}
	classIndex := -1
	for index, choice := range creation.ClassesForRace("human") {
		if choice.ID == "fighter" {
			classIndex = index
		}
	}
	if classIndex < 0 {
		t.Fatal("人類開不出 fighter")
	}
	application.flow = creation.Flow{Stage: creation.StageIconConfirm,
		RaceIndex: raceIndex, ClassIndex: classIndex, Name: "GRUNT"}
	rolled := creation.RollCharacter(application.roller,
		application.flow.SelectedRace(), application.flow.SelectedGender(),
		application.flow.SelectedClass())
	application.rolled = &rolled
	if err := application.finishCharacter(); err != nil {
		t.Fatalf("建角：%v", err)
	}
	if got := application.state.CharacterLibrary[0].ThiefSkills; len(got) != 0 {
		t.Errorf("戰士的賊技能是 %v，應該是空的", got)
	}
}
