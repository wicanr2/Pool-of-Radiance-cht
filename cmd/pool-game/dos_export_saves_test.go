package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 正對照：七名原版預設人物的 `+6Dh..+71h` 與 `+73h` 都是 overlay-23 那一支算出來的，
// 拿記錄自己的八個職業等級（`+96h..+9Dh`）重算，必須逐格相同。
func TestPremadeSavesAreTheLevelTable(t *testing.T) {
	saves, err := gamepack.ReadDOSSavingThrowTable(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	a := &app{savingThrows: saves}
	for _, name := range []string{"chrdatd1", "chrdatd2", "chrdatd3", "chrdatd4",
		"chrdatd5", "chrdatd6", "chrdatd7"} {
		original, err := os.ReadFile(filepath.Join("..", "..", "workplace", "oracle", "dos", name+".sav"))
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		record := append([]byte(nil), original...)
		for offset := dosSavingThrowOffset; offset <= dosTopLevelOffset; offset++ {
			record[offset] = 0
		}
		member := poolsave.Character{Name: name, ClassLevels: append([]uint8(nil), original[0x96:0x9E]...)}
		if err := a.recomputeDOSLevelFields(record, member); err != nil {
			t.Fatal(err)
		}
		for offset := dosSavingThrowOffset; offset <= dosTopLevelOffset; offset++ {
			if offset == 0x72 {
				continue // 移動，不在這一支
			}
			if record[offset] != original[offset] {
				t.Fatalf("%s +%02Xh: recomputed %d, original %d", name, offset, record[offset], original[offset])
			}
		}
	}
}

// remake 自己建的角色：匯出的 `.CHA` 帶得出豁免表與 `+73h`，不再是 0。
func TestDOSExportWritesSavesForANewCharacter(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	saves, err := gamepack.ReadDOSSavingThrowTable(dosZIPForTests)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{itemTypes: table, savingThrows: saves}
	member := poolsave.Character{
		Name: "HERO", RaceID: "human", GenderID: "male", ClassID: "fighter",
		Age: 18, Abilities: [6]int{18, 10, 10, 12, 10, 10},
		MaxHP: 10, CurrentHP: 10, RawHP: 8,
		ClassLevels: []uint8{0, 0, 3, 0, 0, 0, 0, 0},
	}
	files, err := a.buildDOSCharacterFiles(member)
	if err != nil {
		t.Fatal(err)
	}
	var levels [gamepack.ClassThac0ClassCount]uint8
	copy(levels[:], member.ClassLevels)
	want, err := saves.TargetsForLevels(levels)
	if err != nil {
		t.Fatal(err)
	}
	for category, target := range want {
		if target == 0 || files.Record[dosSavingThrowOffset+category] != target {
			t.Fatalf("save %d: exported %d, table %d", category, files.Record[dosSavingThrowOffset+category], target)
		}
	}
	if files.Record[dosTopLevelOffset] != 3 {
		t.Fatalf("+73h is %d, want the fighter's 3", files.Record[dosTopLevelOffset])
	}
}
