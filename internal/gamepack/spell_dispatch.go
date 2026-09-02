package gamepack

import (
	"archive/zip"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// 法術效果的派發表。overlay-22 的 entry 5 在確認法術之後，用法術編號取出一個
// 遠指標就直接呼叫：
//
//	mov  al, [bp+0Eh]        ; 法術編號（1 起算）
//	mov  ds:6779h, al
//	mov  al, ds:6779h
//	xor  ah, ah
//	mov  di, ax
//	shl  di, 1
//	shl  di, 1               ; di = id × 4
//	call dword ptr [di+6A78h]
//	mov  byte ptr ds:6779h, 0
//
// 表本身在 BSS（`6A78h + 30640 = 57880`，已超過 START.EXE 的 47936 bytes），
// 執行期才由 overlay-22 的 entry 10 填。填的碼是 67 段一模一樣的
// `mov ax,off / mov dx,seg / mov [slot],ax / mov [slot+2],dx`，所以
// 「法術編號 → 處理常式」這層對應**可以完全靜態解出來**，不必跑遊戲。
const (
	// SpellDispatchOverlay 是表與派發點所在的 overlay。
	SpellDispatchOverlay = 22
	// SpellDispatchInitEntry 是填表的那支常式（code 0x33CC）。
	SpellDispatchInitEntry = 10
	// SpellDispatchCallEntry 是取用表的那支常式（code 0x0C14，派發在 0x0EA3）。
	SpellDispatchCallEntry = 5

	// SpellDispatchIDAddress 是派發前後寫入法術編號的 DS 位址；
	// 呼叫回來之後會被清成 0，所以它是「正在處理哪一個法術」而不是狀態。
	SpellDispatchIDAddress = 0x6779

	// SpellDispatchHookAddress 是 `6A78h`。它**不是**表的第一格：
	// 0x0D65 與 0x30FA 兩處用 `call dword ptr ds:6A78h` 直接呼叫它，
	// 而且會先推四個引數；派發點那個間接呼叫一個引數都不推。
	// 兩種簽章不可能是同一個東西。overlay-08 還會把它改寫成
	// `0096h:007Ah`——程序變數會被改寫，法術處理常式不會。
	// 表從下一個 double word 開始，剛好讓 `di = id × 4` 對上 1 起算的編號。
	SpellDispatchHookAddress = 0x6a78
	// SpellDispatchTableAddress 是法術 1 的槽。
	SpellDispatchTableAddress = 0x6a7c
	// SpellDispatchSlotSize 是一格的大小（一個遠指標）。
	SpellDispatchSlotSize = 4
	// SpellDispatchCount 是表的格數，也就是最大的法術編號。
	// 名稱表只有 56 個（spec 068），57..67 這 11 格沒有名字，
	// 是物品與怪物特殊效果借用同一條派發路徑。
	SpellDispatchCount = 67

	// spellDispatchInitStride 是填一格所需的碼長度。
	spellDispatchInitStride = 13
	// overlayControlHeaderSize 是控制段前面那段標頭，換算段號時要扣掉。
	overlayControlHeaderSize = 0x3b0
)

// SpellDispatchEntry 是一格：法術編號對到 overlay-22 的哪一支常式。
type SpellDispatchEntry struct {
	// SpellID 是 1 起算的法術編號，與名稱表（spec 068）同一套。
	SpellID int
	// StubOffset 是遠指標指到的常駐 stub 位移。
	StubOffset uint16
	// EntryIndex 是該 stub 在 overlay 進入點表裡的序號。
	EntryIndex int
	// CodeOffset 是處理常式在 overlay 碼段裡的位移。
	CodeOffset uint16
	// InitOffset 是填這一格的那段碼的位置，留著當證據。
	InitOffset int
}

// ParseSpellDispatchTable 從 overlay-22 的碼段還原整張表。
//
// 只認「填到表範圍內、而且遠指標的段號就是這個 overlay 自己」的那些寫入：
// 段號對不上就不是這張表的格子，光比位移會把巧合收進來。
func ParseSpellDispatchTable(overlay tpov.Overlay) ([]SpellDispatchEntry, error) {
	if len(overlay.Entries) == 0 {
		return nil, fmt.Errorf("overlay has no entries")
	}
	first := overlay.Entries[0]
	base := first.ExecutableOffset - int(first.StubOffset) - overlayControlHeaderSize
	if base < 0 || base%16 != 0 {
		return nil, fmt.Errorf("overlay control base %d is not a segment boundary", base)
	}
	segment := uint16(base / 16)

	last := SpellDispatchTableAddress + (SpellDispatchCount-1)*SpellDispatchSlotSize
	found := make(map[int]SpellDispatchEntry, SpellDispatchCount)
	code := overlay.Code
	for offset := 0; offset+spellDispatchInitStride <= len(code); offset++ {
		if code[offset] != 0xb8 || code[offset+3] != 0xba || code[offset+6] != 0xa3 {
			continue
		}
		if code[offset+9] != 0x89 || code[offset+10] != 0x16 {
			continue
		}
		pointerOffset := binary.LittleEndian.Uint16(code[offset+1:])
		pointerSegment := binary.LittleEndian.Uint16(code[offset+4:])
		low := binary.LittleEndian.Uint16(code[offset+7:])
		high := binary.LittleEndian.Uint16(code[offset+11:])
		if pointerSegment != segment || high != low+2 {
			continue
		}
		slot := int(low)
		if slot < SpellDispatchTableAddress || slot > last {
			continue
		}
		if (slot-SpellDispatchTableAddress)%SpellDispatchSlotSize != 0 {
			continue
		}
		id := (slot-SpellDispatchTableAddress)/SpellDispatchSlotSize + 1
		entry, ok := overlay.ResolveStub(pointerOffset)
		if !ok {
			return nil, fmt.Errorf("spell %d points at stub %#04x, which is not an entry", id, pointerOffset)
		}
		if previous, clash := found[id]; clash {
			return nil, fmt.Errorf("spell %d is filled twice, at %#x and %#x", id, previous.InitOffset, offset)
		}
		found[id] = SpellDispatchEntry{SpellID: id, StubOffset: pointerOffset, EntryIndex: entry.Index, CodeOffset: entry.CodeOffset, InitOffset: offset}
	}

	table := make([]SpellDispatchEntry, 0, SpellDispatchCount)
	for id := 1; id <= SpellDispatchCount; id++ {
		entry, ok := found[id]
		if !ok {
			return nil, fmt.Errorf("spell %d has no dispatch slot", id)
		}
		table = append(table, entry)
	}
	if len(found) != SpellDispatchCount {
		return nil, fmt.Errorf("dispatch table has %d slots, want %d", len(found), SpellDispatchCount)
	}
	return table, nil
}

// ReadDOSSpellDispatchTable 從原版 ZIP 直接解出派發表。
func ReadDOSSpellDispatchTable(zipPath string) ([]SpellDispatchEntry, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	overlayFile, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		return nil, err
	}
	overlays, err := tpov.Decode(executable, overlayFile)
	if err != nil {
		return nil, fmt.Errorf("decode GAME.OVR: %w", err)
	}
	if len(overlays) <= SpellDispatchOverlay {
		return nil, fmt.Errorf("GAME.OVR has %d overlays, want more than %d", len(overlays), SpellDispatchOverlay)
	}
	return ParseSpellDispatchTable(overlays[SpellDispatchOverlay])
}

// SpellDispatchGroups 把共用同一支處理常式的法術編號收在一起，只留下真的
// 有人共用的那幾組。它是這張表最直接的語意證據：原版有幾對「同一個效果、
// 牧師與法師各一份」的法術，共用就該落在那幾對上。
func SpellDispatchGroups(table []SpellDispatchEntry) map[uint16][]int {
	groups := make(map[uint16][]int)
	for _, entry := range table {
		groups[entry.CodeOffset] = append(groups[entry.CodeOffset], entry.SpellID)
	}
	for handler, ids := range groups {
		if len(ids) < 2 {
			delete(groups, handler)
			continue
		}
		sort.Ints(ids)
	}
	return groups
}

// readArchiveMember 取出 ZIP 裡唯一一個同名檔案。
func readArchiveMember(zipPath, want string) ([]byte, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()

	var member *zip.File
	for _, candidate := range archive.File {
		if strings.EqualFold(filepath.Base(candidate.Name), want) {
			if member != nil {
				return nil, fmt.Errorf("DOS ZIP has duplicate %s", want)
			}
			member = candidate
		}
	}
	if member == nil {
		return nil, fmt.Errorf("DOS ZIP has no %s", want)
	}
	reader, err := member.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", want, err)
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", want, err)
	}
	return raw, nil
}
