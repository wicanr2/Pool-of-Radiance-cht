package main

import (
	"os"
	"path/filepath"
	"testing"

	poolchar "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// remake 自己建的角色也匯得出一份能用的原版記錄：建角基礎值來自 spec 063，
// 戰鬥數值由 `0E36h` 重算補上。少了重算，THAC0 與 AC 會是 0——那份檔案
// 載得進原版，但那個人打不到東西。
func TestDOSExportRecomputesCombatFieldsForANewCharacter(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	a := &app{itemTypes: table}
	member := poolsave.Character{
		Name: "HERO", RaceID: "human", GenderID: "male", ClassID: "fighter",
		Age: 18, Abilities: [6]int{18, 10, 10, 12, 10, 10}, ExceptionalStrength: 100,
		MaxHP: 10, CurrentHP: 10, RawHP: 8,
		ClassLevels: []uint8{0, 0, 1, 0, 0, 0, 0, 0},
		Inventory:   []poolsave.Item{graveyardSword(t)},
	}
	member.Inventory[0].Raw[gamepack.ItemReadiedOffset] = 1

	files, err := a.buildDOSCharacterFiles(member)
	if err != nil {
		t.Fatal(err)
	}
	if len(files.Record) != poolchar.DOSRecordSize {
		t.Fatalf("記錄有 %d bytes，預期 %d", len(files.Record), poolchar.DOSRecordSize)
	}
	if len(files.Items) != poolchar.ItemRecordSize {
		t.Fatalf("物品鏈有 %d bytes，預期 %d", len(files.Items), poolchar.ItemRecordSize)
	}
	if len(files.Effects) != 0 {
		t.Fatalf("這名角色身上沒有效果，`.SPC` 應該是空的，卻有 %d bytes", len(files.Effects))
	}
	// 基礎值：AC internal 50（檯面 10）、移動 12、參戰旗標 1。
	if files.Record[poolchar.BaseArmourClassOffset] != poolchar.BaseArmourClassValue {
		t.Fatalf("+A9h 是 %d，預期 %d",
			files.Record[poolchar.BaseArmourClassOffset], poolchar.BaseArmourClassValue)
	}
	if files.Record[poolchar.BaseMovementOffset] != poolchar.BaseMovementValue {
		t.Fatalf("+72h 是 %d", files.Record[poolchar.BaseMovementOffset])
	}
	if files.Record[poolchar.PresenceOffset] != 1 {
		t.Fatal("+10Dh 參戰旗標應該是 1")
	}
	// 重算：戰士 1 級的基礎 THAC0 internal 是 28h，裝上有加值的雙手劍再往上加。
	if files.Record[gamepack.BaseThac0Offset] != 0x28 {
		t.Fatalf("+2Dh 是 %d，預期 40", files.Record[gamepack.BaseThac0Offset])
	}
	if files.Record[gamepack.CurrentThac0Offset] <= files.Record[gamepack.BaseThac0Offset] {
		t.Fatalf("+110h 是 %d，裝上武器之後應該大於 +2Dh 的 %d",
			files.Record[gamepack.CurrentThac0Offset], files.Record[gamepack.BaseThac0Offset])
	}
	if files.Record[gamepack.DamageDieSidesOffset] == 0 {
		t.Fatal("+117h 傷害骰面沒有被重算填上")
	}
	if files.Record[gamepack.InternalArmourClassOffset] == 0 {
		t.Fatal("+111h AC 沒有被重算填上")
	}
	// 匯出的記錄讀得回來。
	parsed, err := poolchar.ParseDOS(files.Record)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name != "HERO" || parsed.RaceCode != 7 || parsed.ClassCode != 2 {
		t.Fatalf("讀回 %+v", parsed)
	}
}

// NPC 是從原版 285-byte 記錄來的，匯出時要疊在那一份上面，
// 沒解出來的欄位不能歸零。
func TestDOSExportKeepsTheOriginalRecordOfAnNPC(t *testing.T) {
	table, err := gamepack.ReadDOSItemTypeTable(dosZIPForTests)
	if err != nil {
		t.Skipf("DOS ZIP unavailable: %v", err)
	}
	record, err := os.ReadFile("../../workplace/oracle/dos/chrdatd4.sav")
	if err != nil {
		t.Skipf("original character records unavailable: %v", err)
	}
	a := &app{itemTypes: table}
	member := poolsave.Character{
		Name: "TINA", RaceID: "human", GenderID: "female", ClassID: "thief",
		Abilities: [6]int{18, 18, 18, 18, 17, 14}, ExceptionalStrength: 100,
		MaxHP: 56, CurrentHP: 56, RawHP: 38, Age: 21,
		ClassLevels: []uint8{0, 0, 0, 0, 0, 0, 9, 0},
		Record:      record,
	}
	files, err := a.buildDOSCharacterFiles(member)
	if err != nil {
		t.Fatal(err)
	}
	// 豁免表沒有產生端，必須原樣留著。
	for offset := 0x6D; offset <= 0x71; offset++ {
		if files.Record[offset] != record[offset] {
			t.Fatalf("+%02Xh 的豁免目標值被改掉了", offset)
		}
	}
	if files.Record[0x73] != record[0x73] {
		t.Fatal("+73h 生命骰被改掉了")
	}
	// 賊技能同樣留著。
	for offset := 0x77; offset < 0x7F; offset++ {
		if files.Record[offset] != record[offset] {
			t.Fatalf("+%02Xh 的賊技能被改掉了", offset)
		}
	}
}

// 檔名：原版姓名可以有空白，DOS 檔名不行。
func TestDOSExportNameFitsADOSFilename(t *testing.T) {
	for _, item := range [][2]string{
		{"HERO", "HERO"},
		{"PRINCESS FATIMA", "PRINCESS"},
		{"tina", "TINA"},
		{"  ", "NONAME"},
		{"A-B", "A_B"},
	} {
		if got := dosExportName(item[0]); got != item[1] {
			t.Fatalf("%q 變成 %q，預期 %q", item[0], got, item[1])
		}
	}
}

// 三個檔真的寫得出來，而且大小是各自的單位倍數。
func TestWriteDOSCharacterFilesLandsThreeFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), dosExportDir)
	files := dosCharacterFiles{
		Record:  make([]byte, poolchar.DOSRecordSize),
		Items:   make([]byte, 2*poolchar.ItemRecordSize),
		Effects: make([]byte, 3*poolchar.EffectNodeSize),
	}
	if err := writeDOSCharacterFiles(dir, "HERO", files); err != nil {
		t.Fatal(err)
	}
	for _, item := range [][2]any{
		{dosRecordExtension, poolchar.DOSRecordSize},
		{dosItemExtension, 2 * poolchar.ItemRecordSize},
		{dosEffectsExtension, 3 * poolchar.EffectNodeSize},
	} {
		path := filepath.Join(dir, "HERO"+item[0].(string))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if int(info.Size()) != item[1].(int) {
			t.Fatalf("%s 有 %d bytes，預期 %d", path, info.Size(), item[1].(int))
		}
	}
}
