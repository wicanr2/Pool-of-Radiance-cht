package main

import "testing"

// ECL 位址與引擎位移的換算（spec 106／008）。這五對每一對都有獨立來源，
// 不是互相推出來的——換算式改壞了這裡會整排紅。
func TestEclDisplacementMatchesEveryKnownPair(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		address int
		class   int
		want    int
		source  string
	}{
		{"地圖出口閘門", 0x6DD5, 1, 0x5AA, "spec 125 讀出來的 overlay-14 06AEh"},
		{"休息打斷週期", 0x6DD2, 1, 0x5A4, "spec 114"},
		{"走近的肖像", 0x6DE1, 1, 0x5C2, "spec 117"},
		{"蘇恩神殿服務票", 0x6DE2, 1, 0x5C4, "spec 017，remake 已在用"},
		{"隊伍狀態", 0x49E6, 0, 0x1CC, "spec 074"},
	} {
		if got := eclDisplacement(testCase.address, testCase.class); got != testCase.want {
			t.Errorf("%s：%04Xh（class %d）換出 %Xh，該是 %Xh（%s）",
				testCase.name, testCase.address, testCase.class, got, testCase.want, testCase.source)
		}
	}
}

// 前一個位元組不一定是 modrm。這三條是分類會出錯的地方，錯了就會漏掉
// writer 或生出假的引用。
func TestScanTellsTheAddressingFormsApart(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		bytes []byte
		value int
		want  string
	}{
		{
			name:  "mov word [di+5AA], 1 是 ECL 變數在引擎側的形狀",
			bytes: []byte{0x26, 0xC7, 0x85, 0xAA, 0x05, 0x01, 0x00},
			value: 0x5AA,
			want:  "[基底+disp16]",
		},
		{
			name:  "mov byte [6CD2h], 0 是 DS 全域",
			bytes: []byte{0x05, 0xC6, 0x06, 0xD2, 0x6C, 0x00},
			value: 0x6CD2,
			want:  "[disp16] 絕對",
		},
		{
			name:  "A1h 的 moffs 也是 DS 全域——它的值域和 mod=10 重疊",
			bytes: []byte{0x90, 0x90, 0xA1, 0xD2, 0x6C, 0x90},
			value: 0x6CD2,
			want:  "[disp16] 絕對",
		},
		{
			name:  "mov di, 6CD2h 是把位址當常數，不是 modrm",
			bytes: []byte{0x6A, 0x06, 0xBF, 0xD2, 0x6C, 0x1E},
			value: 0x6CD2,
			want:  "imm16 常數",
		},
	} {
		hits := scan(testCase.bytes, testCase.value, func(int) string { return "x" })
		if len(hits) != 1 {
			t.Errorf("%s：掃到 %d 筆，該是 1 筆", testCase.name, len(hits))
			continue
		}
		if hits[0].kind != testCase.want {
			t.Errorf("%s：分類成 %q，該是 %q", testCase.name, hits[0].kind, testCase.want)
		}
	}
}

// 值出現在位元組裡但前面不是任何一種記憶體存取形式時不能算數——
// 少了這一條，掃出來的筆數會被資料表和位移相同的無關指令灌水。
func TestScanIgnoresBytesThatAreNotAnAccess(t *testing.T) {
	// 90h 是 nop，後面接的兩個位元組只是資料。
	hits := scan([]byte{0x90, 0x90, 0x90, 0xD2, 0x6C, 0x90}, 0x6CD2, func(int) string { return "x" })
	if len(hits) != 0 {
		t.Fatalf("掃到 %d 筆，該是零筆：%+v", len(hits), hits)
	}
}

// overlay 定位靠 code_bytes ＋ relocation_bytes 依 index 累加，起點是 15h。
// 用 manifest 的 executable_file_offset 會落在完全不同的地方——那是
// START.EXE 裡的 stub 位置。
func TestLocateWalksTheOverlaysInOrder(t *testing.T) {
	list := manifest{}
	list.Overlays = append(list.Overlays,
		struct {
			Index           int `json:"index"`
			CodeBytes       int `json:"code_bytes"`
			RelocationBytes int `json:"relocation_bytes"`
		}{Index: 0, CodeBytes: 0x100, RelocationBytes: 0x10},
		struct {
			Index           int `json:"index"`
			CodeBytes       int `json:"code_bytes"`
			RelocationBytes int `json:"relocation_bytes"`
		}{Index: 1, CodeBytes: 0x200, RelocationBytes: 0x20},
	)

	if got, want := list.locate(0x15), "overlay-00 0000h"; got != want {
		t.Errorf("第一顆 overlay 的起點是 %s，該是 %s", got, want)
	}
	if got, want := list.locate(0x15+0x100+0x10+0x40), "overlay-01 0040h"; got != want {
		t.Errorf("第二顆 overlay 內的位址是 %s，該是 %s", got, want)
	}
	// 落在 relocation 表裡的位置不屬於任何 overlay 的程式碼。
	if got, want := list.locate(0x15+0x100+0x05), "GAME.OVR+0011Ah"; got != want {
		t.Errorf("relocation 區的位址報成 %s，該是 %s", got, want)
	}
}
