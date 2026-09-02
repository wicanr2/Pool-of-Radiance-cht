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

	// spellDispatchMessageMax 是效果訊息的最大長度，用來限制往回找的範圍。
	spellDispatchMessageMax = 63
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
	// Message 是處理常式印出來的效果訊息，例如 Bless 的 `is Blessed`。
	// 它是一段 Pascal 字串，就放在處理常式的第一個 byte 前面；沒有訊息的
	// 常式這裡是空字串。
	Message string
	// MessageOffset 是那段字串的長度 byte 位置；沒有訊息時是 -1。
	MessageOffset int
}

// spellDispatchMessage 取處理常式前面那段 Pascal 字串。
//
// 只認「本體真的用到它」的那些：字串必須以某個 16-bit 立即數的形式出現在
// 這支常式的碼裡（Turbo Pascal 是 `mov di, imm16` 再 `push cs; push di`）。
// 少了這道檢查，往回找長度 byte 這個手法會把碰巧成立的位元組收成訊息。
func spellDispatchMessage(code []byte, start, end int) (string, int) {
	for length := 1; length <= spellDispatchMessageMax; length++ {
		at := start - 1 - length
		if at < 0 {
			return "", -1
		}
		if int(code[at]) != length {
			continue
		}
		text := code[at+1 : at+1+length]
		printable := true
		for _, b := range text {
			if b < 0x20 || b > 0x7e {
				printable = false
				break
			}
		}
		if !printable {
			continue
		}
		if !referencesWord(code[start:end], uint16(at)) {
			return "", -1
		}
		return string(text), at
	}
	return "", -1
}

// referencesWord 說這段碼裡有沒有出現這個 16-bit 立即數。
func referencesWord(code []byte, value uint16) bool {
	for offset := 0; offset+1 < len(code); offset++ {
		if binary.LittleEndian.Uint16(code[offset:]) == value {
			return true
		}
	}
	return false
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
		found[id] = SpellDispatchEntry{SpellID: id, StubOffset: pointerOffset, EntryIndex: entry.Index, CodeOffset: entry.CodeOffset, InitOffset: offset, MessageOffset: -1}
	}

	bounds := make([]int, 0, len(found))
	for _, entry := range found {
		bounds = append(bounds, int(entry.CodeOffset))
	}
	sort.Ints(bounds)
	next := func(start int) int {
		for _, candidate := range bounds {
			if candidate > start {
				return candidate
			}
		}
		return len(code)
	}

	table := make([]SpellDispatchEntry, 0, SpellDispatchCount)
	for id := 1; id <= SpellDispatchCount; id++ {
		entry, ok := found[id]
		if !ok {
			return nil, fmt.Errorf("spell %d has no dispatch slot", id)
		}
		start := int(entry.CodeOffset)
		entry.Message, entry.MessageOffset = spellDispatchMessage(code, start, next(start))
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

// 純泛型的處理常式（spec 098）。
//
// 六十七支裡有一大批只做兩件事：把一段字面訊息複製到區域變數，然後推
// 「法術編號 ＋ 四個零」呼叫 `08BCh`。它們**沒有自己的算法**——射程、
// 持續、豁免與掛哪一個效果全部來自參數表（spec 074），所以認得出這個
// 版型就等於一次接完一整批。
//
// 版型（以 Protection From Evil `110Bh` 為例）：
//
//	55 89 E5 [83 EC nn]      push bp / mov bp,sp / sub sp,nn
//	A0 79 67 50              mov al, ds:6779h / push ax     ; 法術編號
//	(B0 00 50) × 4           push 0 四次                     ; 四個覆寫參數
//	8D 7E nn 16 57           lea di,[bp-nn] / push ss / push di
//	BF lo hi 0E 57           mov di, 訊息位移 / push cs / push di
//	9A 34 06 BB 05           lcall 05BBh:0634h              ; 字串指派
//	0E E8 lo hi              push cs / call 08BCh
//	89 EC 5D CB              mov sp,bp / pop bp / retf
//
// 比對整個版型而不是只看「有沒有呼叫 08BCh」：會算傷害的那幾支也呼叫
// 08BCh，只看呼叫會把它們一起收進來，然後傷害就消失了。
const sharedCastRoutineOffset = 0x08bc

// GenericSpellHandler 是一支純泛型的處理常式。
type GenericSpellHandler struct {
	// SpellID 是 1-based 的法術編號。
	SpellID int
	// Message 是它推給 `08BCh` 的那一段字面訊息。
	Message string
}

// ParseGenericSpellHandlers 找出所有符合版型的處理常式。
func ParseGenericSpellHandlers(overlay tpov.Overlay, table []SpellDispatchEntry) []GenericSpellHandler {
	code := overlay.Code
	out := make([]GenericSpellHandler, 0, len(table))
	for _, entry := range table {
		message, ok := matchGenericSpellHandler(code, int(entry.CodeOffset))
		if !ok {
			continue
		}
		out = append(out, GenericSpellHandler{SpellID: entry.SpellID, Message: message})
	}
	return out
}

// matchGenericSpellHandler 逐位元組比對版型，回傳訊息字串。
func matchGenericSpellHandler(code []byte, offset int) (string, bool) {
	at := func(index int) byte {
		if index < 0 || index >= len(code) {
			return 0xff
		}
		return code[index]
	}
	expect := func(index int, want ...byte) bool {
		for step, value := range want {
			if at(index+step) != value {
				return false
			}
		}
		return true
	}
	position := offset
	if !expect(position, 0x55, 0x89, 0xe5) {
		return "", false
	}
	position += 3
	if expect(position, 0x83, 0xec) {
		position += 3
	}
	if !expect(position, 0xa0, 0x79, 0x67, 0x50) {
		return "", false
	}
	position += 4
	zeros := 0
	for expect(position, 0xb0, 0x00, 0x50) {
		zeros++
		position += 3
	}
	if zeros != 4 {
		return "", false
	}
	if !expect(position, 0x8d, 0x7e) {
		return "", false
	}
	position += 3
	if !expect(position, 0x16, 0x57) {
		return "", false
	}
	position += 2
	if at(position) != 0xbf {
		return "", false
	}
	messageOffset := int(binary.LittleEndian.Uint16(code[position+1:]))
	position += 3
	if !expect(position, 0x0e, 0x57) {
		return "", false
	}
	position += 2
	if !expect(position, 0x9a, 0x34, 0x06, 0xbb, 0x05) {
		return "", false
	}
	position += 5
	if !expect(position, 0x0e, 0xe8) {
		return "", false
	}
	target := position + 4 + int(int16(binary.LittleEndian.Uint16(code[position+2:])))
	if target != sharedCastRoutineOffset {
		return "", false
	}
	return pascalString(code, messageOffset)
}

// pascalString 讀一條 Turbo Pascal 短字串（長度在前）。
func pascalString(code []byte, offset int) (string, bool) {
	if offset < 0 || offset >= len(code) {
		return "", false
	}
	length := int(code[offset])
	if length == 0 || offset+1+length > len(code) {
		return "", false
	}
	text := code[offset+1 : offset+1+length]
	for _, value := range text {
		if value < 0x20 || value > 0x7e {
			return "", false
		}
	}
	return string(text), true
}

// ReadDOSGenericSpellHandlers 從原版 ZIP 直接解出那一批。
func ReadDOSGenericSpellHandlers(zipPath string) ([]GenericSpellHandler, error) {
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
	table, err := ParseSpellDispatchTable(overlays[SpellDispatchOverlay])
	if err != nil {
		return nil, err
	}
	return ParseGenericSpellHandlers(overlays[SpellDispatchOverlay], table), nil
}
