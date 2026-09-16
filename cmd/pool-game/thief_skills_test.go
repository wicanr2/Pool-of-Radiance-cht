package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
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

// 訓練所升級會用角色記錄裡的真實 DEX 重算八格，而不是沿用建角時的 DEX 0
//（spec 095）。這組 Dwarf／DEX 18／thief 1→2 的結果逐位元組來自 dosgolem。
func TestTrainingAThiefRecalculatesTheOriginalSkillsWithRealDexterity(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	thief := poolsave.Character{Name: "HUMAN", RaceID: "dwarf", GenderID: "male",
		ClassID: "thief", AlignmentID: "chaotic-good",
		Abilities: [6]int{14, 17, 14, 18, 15, 16}, MaxHP: 10, CurrentHP: 10,
		Experience: 1251, ThiefSkills: []uint8{35, 45, 40, 15, 10, 10, 75, 0}}
	thief.Money[pooltreasure.Platinum] = 200 // 訓練要 1000 金（#29）
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{thief}, Party: []poolsave.Character{thief}}
	application.mode, application.programManaging = modeMenu, true
	if err := press(application, ebiten.KeyDigit1); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyT); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyY); err != nil {
		t.Fatal(err)
	}
	want := []uint8{45, 54, 45, 31, 25, 10, 76, 0}
	got := application.state.Party[0]
	if len(got.ClassLevels) <= gamepack.ThiefLevelIndex ||
		got.ClassLevels[gamepack.ThiefLevelIndex] != 2 {
		t.Fatalf("盜賊應升到第 2 級，職業等級是 %v", got.ClassLevels)
	}
	if len(got.ThiefSkills) != len(want) {
		t.Fatalf("訓練後有 %d 格技能，原版是 %d 格：%v",
			len(got.ThiefSkills), len(want), got.ThiefSkills)
	}
	for index := range want {
		if got.ThiefSkills[index] != want[index] {
			t.Errorf("第 %d 格是 %d，原版是 %d（整份 %v）",
				index, got.ThiefSkills[index], want[index], got.ThiefSkills)
		}
	}
	if library := application.state.CharacterLibrary[0].ThiefSkills; len(library) != len(want) {
		t.Errorf("角色庫沒有同步訓練後技能：%v", library)
	} else {
		for index := range want {
			if library[index] != want[index] {
				t.Errorf("角色庫第 %d 格是 %d，訓練後應為 %d（整份 %v）",
					index, library[index], want[index], library)
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
