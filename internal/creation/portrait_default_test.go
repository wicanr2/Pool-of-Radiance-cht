package creation

import (
	"archive/zip"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"testing"
)

// 預設肖像依性別與職業取原版表的第一格（#130）。
func TestDefaultPortraitByGenderAndClass(t *testing.T) {
	for _, tc := range []struct {
		gender     uint8
		class      string
		head, body uint8
	}{
		{0, "fighter", 1, 1}, {1, "fighter", 6, 7},
		{0, "cleric", 1, 2}, {1, "cleric", 6, 2},
		{0, "magic-user", 1, 2}, {1, "magic-user", 6, 8},
		{0, "thief", 1, 6}, {1, "thief", 6, 8},
		// 多職業取列索引最小的組成職業：牧師 0 < 戰士 2 < 法師 5 < 盜賊 6。
		{1, "fighter-magic-user", 6, 7}, {1, "cleric-fighter", 6, 2},
		{1, "magic-user-thief", 6, 8},
	} {
		if _, ok := ClassComponents(tc.class); !ok {
			t.Fatalf("class %q is not in the catalogue", tc.class)
		}
		head, body := DefaultPortrait(tc.gender, tc.class)
		if head != tc.head || body != tc.body {
			t.Errorf("gender %d %s: %d/%d, want %d/%d", tc.gender, tc.class, head, body, tc.head, tc.body)
		}
	}
	if head, body := DefaultPortrait(0, "no-such-class"); head != 1 || body != 1 {
		t.Errorf("unknown class: %d/%d, want the original 1/1", head, body)
	}
}

// 從資料頁一路走到肖像頁，女性戰士一進肖像頁就是 6／7；
// 已經有值時 SetName 不覆蓋（退回姓名再回來不洗掉玩家挑的頭像）。
func TestSetNameUsesGenderDefaultAndKeepsChoice(t *testing.T) {
	flow := Flow{Stage: StageName, RaceIndex: raceIndex(t, "human"), GenderIndex: 1}
	flow.ClassIndex = classIndex(t, "human", "fighter")
	if err := flow.SetName("ANNA"); err != nil {
		t.Fatal(err)
	}
	if flow.PortraitHead != 6 || flow.PortraitBody != 7 {
		t.Fatalf("female fighter default %d/%d, want 6/7", flow.PortraitHead, flow.PortraitBody)
	}
	flow.PortraitHead, flow.Stage = 9, StageName
	if err := flow.SetName("ANNA"); err != nil {
		t.Fatal(err)
	}
	if flow.PortraitHead != 9 {
		t.Fatalf("SetName overwrote the chosen head: %d", flow.PortraitHead)
	}
}

func raceIndex(t *testing.T, id string) int {
	t.Helper()
	for index, race := range Races {
		if race.ID == id {
			return index
		}
	}
	t.Fatalf("race %q missing", id)
	return 0
}

func classIndex(t *testing.T, race, id string) int {
	t.Helper()
	for index, choice := range ClassesForRace(race) {
		if choice.ID == id {
			return index
		}
	}
	t.Fatalf("class %q missing for %s", id, race)
	return 0
}

// 表的位元組對原版 START.EXE 核對。資料段位址換算與 spec 006 同一套：
// image 基址 10000h、資料段 17400h，所以 DS:x 在檔案裡是 header + 7400h + x。
// 正對照是 spec 006 已經驗過的 DS 2884h HEAD3 block 表，它對得上，
// 這一套換算才可信。
func TestDefaultPortraitTablesMatchStartEXE(t *testing.T) {
	const path = "../../Pool of Radiance (1988).zip"
	if _, err := os.Stat(path); err != nil {
		t.Skip("原版磁碟映像不在版控裡，跳過")
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	var exe []byte
	for _, file := range archive.File {
		if strings.EqualFold(file.Name[strings.LastIndex(file.Name, "/")+1:], "START.EXE") {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			exe, err = io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(exe) < 0x40 {
		t.Fatal("START.EXE not found in the ZIP")
	}
	header := int(binary.LittleEndian.Uint16(exe[8:])) * 16
	ds := func(offset, length int) []byte {
		start := header + 0x7400 + offset
		return exe[start : start+length]
	}
	control := []byte{0x00, 0x08, 0x09, 0x0D, 0x10, 0x12, 0x16, 0x22, 0x2D, 0x33, 0x35, 0x39, 0x43, 0x44}
	if got := ds(0x2884, len(control)); string(got) != string(control) {
		t.Fatalf("positive control DS:2884h = % x, want % x", got, control)
	}
	for gender := 0; gender < 2; gender++ {
		want := defaultPortraitHeads[gender]
		if got := ds(0x8DC+gender*8, 8); string(got) != string(want[:]) {
			t.Errorf("HEAD gender %d: DS bytes % x, table % x", gender, got, want)
		}
	}
	for class, base := range map[uint8]int{0: 0x8EB, 0xFF: 0x8F5, 5: 0x8FF, 6: 0x909} {
		for gender := 0; gender < 2; gender++ {
			want := defaultPortraitBodies[class][gender]
			if got := ds(base+1+gender*5, 5); string(got) != string(want[:]) {
				t.Errorf("BODY class %d gender %d: DS bytes % x, table % x", class, gender, got, want)
			}
		}
	}
}
