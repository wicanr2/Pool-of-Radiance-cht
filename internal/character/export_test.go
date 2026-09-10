package character_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/creation"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

const oracleDir = "../../workplace/oracle/dos"

// 原版七名 DOS 預設人物：讀進 remake 的角色模型，再寫回同一份記錄，
// 285 bytes 必須一個位元組都不差。
//
// 這一條同時擋住兩種錯法：寫錯位置（輸出會與輸入不同），以及「順手」把
// 沒解出來的欄位歸零（豁免表、賊技能、護甲、負重都在 base 裡，一改就露出來）。
func TestExportDOSRecordReproducesThePremades(t *testing.T) {
	for _, name := range []string{"chrdatd1", "chrdatd2", "chrdatd3", "chrdatd4",
		"chrdatd5", "chrdatd6", "chrdatd7"} {
		record, err := os.ReadFile(filepath.Join(oracleDir, name+".sav"))
		if err != nil {
			t.Skipf("original character records unavailable: %v", err)
		}
		person, err := readPremade(record)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		written, err := character.ExportDOSRecord(record, person)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for offset := range record {
			if written[offset] != record[offset] {
				t.Fatalf("%s (%s) 的 +%02Xh 寫成 %02X，原本是 %02X",
					name, person.Name, offset, written[offset], record[offset])
			}
		}
		// 「輸出等於輸入」也可能是因為根本沒寫。改一個欄位，
		// 差異必須正好落在那一個位元組上。
		person.CurrentHP = int(record[0x11B]) / 2
		hurt, err := character.ExportDOSRecord(record, person)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var differ []int
		for offset := range record {
			if hurt[offset] != record[offset] {
				differ = append(differ, offset)
			}
		}
		if len(differ) != 1 || differ[0] != 0x11B {
			t.Fatalf("%s：只改現在生命，差異卻落在 %v", name, differ)
		}
	}
}

// 只寫有出處的欄位：其餘位元組原封不動。用一份填滿 AAh 的 base 檢查，
// 沒被列進「會寫」清單的位置必須還是 AAh。
func TestExportDOSRecordLeavesUndocumentedBytesAlone(t *testing.T) {
	base := make([]byte, character.DOSRecordSize)
	for index := range base {
		base[index] = 0xAA
	}
	written, err := character.ExportDOSRecord(base, poolsave.Character{
		Name: "TINA", RaceID: "human", GenderID: "female", ClassID: "thief",
	})
	if err != nil {
		t.Fatal(err)
	}
	touched := map[int]bool{}
	for _, span := range [][2]int{
		{0x00, 0x10}, // 姓名長度與內容
		{0x10, 0x17}, // 六個能力值加特殊力量百分位
		{0x2E, 0x2F}, // 種族
		{0x2F, 0x30}, // 職業
		{0x30, 0x31}, // 年齡
		{0x32, 0x33}, // 生命上限
		{0x88, 0x96}, // 七種錢
		{0x96, 0x9E}, // 每職業等級
		{0x9E, 0x9F}, // 性別
		{0xAC, 0xB0}, // 經驗值
		{0xB1, 0xB2}, // 不含體質的生命
		{0xBB, 0xBF}, // 肖像與圖示的頭、武器
		{0xC0, 0xC7}, // 圖示大小與六組顏色
		{0x10C, 0x10D}, // 狀態
		{0x11B, 0x11C}, // 現在生命
	} {
		for offset := span[0]; offset < span[1]; offset++ {
			touched[offset] = true
		}
	}
	for offset, value := range written {
		if touched[offset] {
			continue
		}
		if value != 0xAA {
			t.Fatalf("+%02Xh 不在會寫的清單裡，卻從 AA 變成 %02X", offset, value)
		}
	}
	// 反向對照：清單裡的位置真的有被寫過，否則這個測試會隨著漏寫一起變綠。
	if written[0x2E] != 7 || written[0x2F] != 6 || written[0x9E] != 1 {
		t.Fatalf("種族／職業／性別寫成 %02X %02X %02X，預期 07 06 01",
			written[0x2E], written[0x2F], written[0x9E])
	}
	if written[0xBF] != 0xAA {
		t.Fatal("圖示的不透明旗標 +BFh 應該保留 base 的值")
	}
}

// ClassLevels 是空的代表「每個組成職業都是第 1 級」。
// 這是讀取端的語意（spec 097），寫回去也要照它展開，不能留 base 的舊值。
func TestExportDOSRecordExpandsEmptyClassLevels(t *testing.T) {
	base := make([]byte, character.DOSRecordSize)
	for index := 0x96; index < 0x9E; index++ {
		base[index] = 9 // 舊值：故意不是 0，漏寫就會被抓到
	}
	written, err := character.ExportDOSRecord(base, poolsave.Character{
		Name: "NEW", RaceID: "elf", GenderID: "male", ClassID: "fighter-magic-user-thief",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [8]uint8{0, 0, 1, 0, 0, 1, 1, 0} // 戰士 2、法師 5、賊 6
	var got [8]uint8
	copy(got[:], written[0x96:0x9E])
	if got != want {
		t.Fatalf("每職業等級寫成 %v，預期 %v", got, want)
	}
}

// 匯出的記錄要能被自己的讀取端讀回來，欄位對得上。
func TestExportDOSRecordRoundTripsThroughParseDOS(t *testing.T) {
	base := make([]byte, character.DOSRecordSize)
	person := poolsave.Character{
		Name: "ALFRED", RaceID: "half-elf", GenderID: "male", ClassID: "cleric-magic-user",
		Age: 19, Abilities: [6]int{18, 12, 17, 13, 16, 11}, ExceptionalStrength: 100,
		MaxHP: 36, CurrentHP: 30, RawHP: 21, Status: 0,
		Money:        [7]uint16{833, 1000, 333, 168, 33, 19, 6},
		ClassLevels:  []uint8{6, 0, 0, 0, 0, 4, 0, 0},
		Experience:   275627,
		PortraitHead: 3, PortraitBody: 5, IconHead: 2, IconWeapon: 4, IconSize: 1,
		IconColors: [6][2]uint8{{1, 2}, {3, 4}, {5, 6}, {7, 8}, {9, 10}, {11, 12}},
	}
	written, err := character.ExportDOSRecord(base, person)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := character.ParseDOS(written)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name != person.Name {
		t.Fatalf("姓名讀回 %q，預期 %q", parsed.Name, person.Name)
	}
	if parsed.RaceCode != 4 || parsed.ClassCode != 11 || parsed.GenderCode != 0 {
		t.Fatalf("種族／職業／性別讀回 %d %d %d，預期 4 11 0",
			parsed.RaceCode, parsed.ClassCode, parsed.GenderCode)
	}
	for index, value := range person.Abilities {
		if int(parsed.Abilities[index]) != value {
			t.Fatalf("能力值 %d 讀回 %d，預期 %d", index, parsed.Abilities[index], value)
		}
	}
	if parsed.Portrait.Head != 3 || parsed.Portrait.Body != 5 {
		t.Fatalf("肖像讀回 %+v", parsed.Portrait)
	}
	if parsed.Icon.Body.Color1 != 1 || parsed.Icon.Body.Color2 != 2 {
		t.Fatalf("圖示顏色讀回 %+v", parsed.Icon.Body)
	}
	// 這五個位元組是 ECL opcode 1Dh 的隊伍實力算式吃的（spec 030）。
	// +96h/+9Bh 是牧師與法師的等級，+11Bh 是現在生命。
	if parsed.PartyStrength.Field96 != 6 || parsed.PartyStrength.Field9B != 4 ||
		parsed.PartyStrength.Field11B != 30 {
		t.Fatalf("隊伍實力欄位讀回 %+v", parsed.PartyStrength)
	}
	if got := binary.LittleEndian.Uint32(written[0xAC:0xB0]); got != person.Experience {
		t.Fatalf("經驗值讀回 %d，預期 %d", got, person.Experience)
	}
}

func TestExportDOSRecordRejectsBadInput(t *testing.T) {
	base := make([]byte, character.DOSRecordSize)
	good := poolsave.Character{Name: "X", RaceID: "human", GenderID: "male", ClassID: "fighter"}
	for _, item := range []struct {
		why    string
		base   []byte
		person poolsave.Character
	}{
		{"base 長度不對", make([]byte, 100), good},
		{"姓名超過 15 bytes", base, withName(good, "PRINCESS FATIMAS")},
		{"沒有這個種族", base, withRace(good, "orc")},
		{"沒有這個職業", base, withClass(good, "paladin")},
		{"沒有這個性別", base, withGender(good, "other")},
		{"等級陣列太長", base, withLevels(good, make([]uint8, 9))},
	} {
		if _, err := character.ExportDOSRecord(item.base, item.person); err == nil {
			t.Fatalf("%s：應該失敗卻通過了", item.why)
		}
	}
	if _, err := character.ExportDOSRecord(base, withName(good, "PRINCESS FATIMA")); err != nil {
		t.Fatalf("剛好 15 bytes 的姓名應該可以：%v", err)
	}
}

// `.itm` 就是背包裡每一件的 63 bytes 接起來，順序照背包。
func TestExportDOSItemsReproducesTheOriginalChain(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(oracleDir, "chrdatd4.itm"))
	if err != nil {
		t.Skipf("original item chains unavailable: %v", err)
	}
	if len(raw)%character.ItemRecordSize != 0 {
		t.Fatalf("chrdatd4.itm 有 %d bytes，不是 %d 的倍數", len(raw), character.ItemRecordSize)
	}
	var items []poolsave.Item
	for offset := 0; offset < len(raw); offset += character.ItemRecordSize {
		items = append(items, poolsave.Item{
			Raw: append([]byte(nil), raw[offset:offset+character.ItemRecordSize]...)})
	}
	written, err := character.ExportDOSItems(items)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != string(raw) {
		t.Fatal("寫回去的 .itm 與原檔不同")
	}
	if _, err := character.ExportDOSItems([]poolsave.Item{{Name: "短", Raw: []byte{1, 2}}}); err == nil {
		t.Fatal("原始記錄長度不對應該失敗，不能補零")
	}
}

// `.spc` 一個節點 9 bytes，`+0` 是效果碼；`+5..+8` 是上次執行的遠指標，
// 重新載入沒有意義（spec 069），所以寫 0。
func TestExportDOSEffectsMatchesTheOriginalNodeCodes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(oracleDir, "chrdatd4.spc"))
	if err != nil {
		t.Skipf("original effect chains unavailable: %v", err)
	}
	if len(raw)%character.EffectNodeSize != 0 {
		t.Fatalf("chrdatd4.spc 有 %d bytes，不是 %d 的倍數", len(raw), character.EffectNodeSize)
	}
	var nodes []poolsave.EffectNode
	for offset := 0; offset < len(raw); offset += character.EffectNodeSize {
		node := poolsave.EffectNode{Code: raw[offset]}
		copy(node.Payload[:], raw[offset+1:offset+5])
		nodes = append(nodes, node)
	}
	written := character.ExportDOSEffects(nodes)
	if len(written) != len(raw) {
		t.Fatalf("寫出 %d bytes，原檔 %d bytes", len(written), len(raw))
	}
	// 前五個 byte 要與原檔一模一樣：`+0` 是碼，`+1..+4` 是持續、等級與收尾
	// 旗標。只比對碼的話，schema 8 帶回來的那四個 byte 掉了也看不出來。
	for index := range nodes {
		at := index * character.EffectNodeSize
		if got, want := written[at:at+5], raw[at:at+5]; !bytes.Equal(got, want) {
			t.Fatalf("節點 %d 寫成 % X，預期 % X", index, got, want)
		}
		if got := written[at+5 : at+character.EffectNodeSize]; !bytes.Equal(got, []byte{0, 0, 0, 0}) {
			t.Fatalf("節點 %d 的遠指標寫成 % X，預期全 0", index, got)
		}
	}
	if len(character.ExportDOSEffects(nil)) != 0 {
		t.Fatal("沒有效果就是空檔案")
	}
}

// readPremade 把原版記錄讀成 remake 的角色模型，只讀 ExportDOSRecord 會寫的欄位。
func readPremade(record []byte) (poolsave.Character, error) {
	person := poolsave.Character{
		Name:                string(record[1 : 1+int(record[0])]),
		Age:                 int(record[0x30]),
		ExceptionalStrength: int(record[0x16]),
		MaxHP:               int(record[0x32]),
		CurrentHP:           int(record[0x11B]),
		RawHP:               int(record[0xB1]),
		Status:              record[0x10C],
		Experience:          binary.LittleEndian.Uint32(record[0xAC:0xB0]),
		PortraitHead:        record[0xBB],
		PortraitBody:        record[0xBC],
		IconHead:            record[0xBD],
		IconWeapon:          record[0xBE],
		IconSize:            record[0xC0],
		ClassLevels:         append([]uint8(nil), record[0x96:0x9E]...),
		ThiefSkills:         append([]uint8(nil), record[0x77:0x7F]...),
	}
	for index := range person.Abilities {
		person.Abilities[index] = int(record[0x10+index])
	}
	for slot := range person.Money {
		person.Money[slot] = binary.LittleEndian.Uint16(record[0x88+slot*2:])
	}
	for index, offset := range []int{0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6} {
		person.IconColors[index] = [2]uint8{record[offset] & 0x0F, record[offset] >> 4}
	}
	raceID, ok := raceForCode(record[0x2E])
	if !ok {
		return person, errUnknownCode("race", record[0x2E])
	}
	person.RaceID = raceID
	classID, ok := classForCode(raceID, record[0x2F])
	if !ok {
		return person, errUnknownCode("class", record[0x2F])
	}
	person.ClassID = classID
	genderID, ok := genderForCode(record[0x9E])
	if !ok {
		return person, errUnknownCode("gender", record[0x9E])
	}
	person.GenderID = genderID
	return person, nil
}

func raceForCode(code uint8) (string, bool) {
	for _, race := range creation.Races {
		if race.DOSCode == code {
			return race.ID, true
		}
	}
	return "", false
}

func classForCode(raceID string, code uint8) (string, bool) {
	for _, choice := range creation.ClassesForRace(raceID) {
		if choice.DOSCode == code {
			return choice.ID, true
		}
	}
	return "", false
}

func genderForCode(code uint8) (string, bool) {
	for _, gender := range creation.Genders {
		if gender.DOSCode == code {
			return gender.ID, true
		}
	}
	return "", false
}

func errUnknownCode(field string, code uint8) error {
	return fmt.Errorf("Pool %s code %d has no catalog entry", field, code)
}

func withName(person poolsave.Character, name string) poolsave.Character {
	person.Name = name
	return person
}
func withRace(person poolsave.Character, id string) poolsave.Character {
	person.RaceID = id
	return person
}
func withClass(person poolsave.Character, id string) poolsave.Character {
	person.ClassID = id
	return person
}
func withGender(person poolsave.Character, id string) poolsave.Character {
	person.GenderID = id
	return person
}
func withLevels(person poolsave.Character, levels []uint8) poolsave.Character {
	person.ClassLevels = levels
	return person
}
