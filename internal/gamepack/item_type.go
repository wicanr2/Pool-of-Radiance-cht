package gamepack

import (
	"archive/zip"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// 物品型別表。overlay-25 的武器數值那支以 `type × 16` 去索引 `DS:54E0h`
// （spec 063），但那段 DS 在 START.EXE 裡是未初始化的——匯出 1,024 bytes
// 全是 FF、file_offset 為 -1。表是執行時載進去的，來源是 ZIP 裡的
// `poolrad/items`：2,050 bytes ＝ 2-byte 檔頭加 128 筆 × 16 bytes。
//
// 對應關係有正對照：`ITEM3.DAX/33h` 的 `Two-Handed Sword +1 +3 vs. Undead`
// 記錄 `+2Eh` 是 `26h`，而表的第 `26h` 筆正是 1d10／3d6，也就是 AD&D 的
// 雙手劍；該記錄 `+32h` 是 1，與名稱的 `+1` 相符。
const (
	// ItemTypeEntrySize 是每筆的 byte 數，由 overlay-25 的 `shl di, 4` 決定。
	ItemTypeEntrySize = 16
	// ItemTypeCount 是表的筆數。
	ItemTypeCount = 128
	// itemTypeHeaderSize 是檔頭；表身接在後面。
	itemTypeHeaderSize = 2
	itemTypeFileName   = "ITEMS"
)

// 旗標欄（entry `+0Eh`）目前用到的四個位元，語意由 overlay-25 的分支決定。
const (
	// ItemTypeFlagNeedsLauncher（bit 0）成立時再加 `+0F8h` 那件物品的 `+32h`。
	ItemTypeFlagNeedsLauncher = 1 << 0
	// ItemTypeFlagDexterityToHit（bit 1）成立時 THAC0 加上敏捷的投射修正。
	ItemTypeFlagDexterityToHit = 1 << 1
	// ItemTypeFlagStrengthBonuses（bit 2）成立時 THAC0 與傷害加上力量修正。
	ItemTypeFlagStrengthBonuses = 1 << 2
	// ItemTypeFlagUsesAmmunition（bit 7）成立時再加 `+0FCh` 那件物品的 `+32h`。
	ItemTypeFlagUsesAmmunition = 1 << 7
)

// ItemTypeEntry 是一筆物品型別。保留完整 raw：本規格只替 overlay-25 讀過的
// 欄位命名，其餘欄位語意未閉合，不猜。
type ItemTypeEntry struct {
	Raw [ItemTypeEntrySize]byte
}

// Category 是 entry `+0`；overlay-25 `01F8h` 以它是否為 2 分支。
func (e ItemTypeEntry) Category() uint8 { return e.Raw[0x00] }

// LargeDamage 是對大型目標的傷害骰（entry `+2`／`+3`）。
func (e ItemTypeEntry) LargeDamage() (count, sides uint8) { return e.Raw[0x02], e.Raw[0x03] }

// Damage 是 overlay-25 寫進 record `+115h`／`+117h` 的那一組骰
//（entry `+9`／`+0Ah`），即對中小型目標的傷害。
func (e ItemTypeEntry) Damage() (count, sides uint8) { return e.Raw[0x09], e.Raw[0x0a] }

// DamageBonus 是 entry `+0Bh`，overlay-25 直接寫進 record `+119h`。
func (e ItemTypeEntry) DamageBonus() uint8 { return e.Raw[0x0b] }

// Flags 是 entry `+0Eh`。
func (e ItemTypeEntry) Flags() uint8 { return e.Raw[0x0e] }

// AttackRange 是這件武器打得到幾格（spec 065）。
//
// 出處是挑目標的介面：overlay-13 `358Dh` 取 `DS:54ECh + 型別×10h`
// （也就是本表的 `+0Ch`）之後 `dec ax`，呼叫端再把 0 與 FFh 都當成 1。
// 所以**表裡存的是「射程加一」，近戰武器存 0**：
//
//	0 或 1 → 1 格（相鄰）
//	n > 1  → n − 1 格
func (e ItemTypeEntry) AttackRange() int {
	value := int(e.Raw[0x0c])
	if value <= 1 {
		return 1
	}
	return value - 1
}

// ItemTypeTable 是整張表。
type ItemTypeTable struct {
	// Header 是檔頭那兩個 byte，原樣保留：它的語意還沒閉合，
	// 但不保留就無法逐位元組重生這個檔。
	Header  uint16
	Entries [ItemTypeCount]ItemTypeEntry
}

// Entry 依型別索引取一筆。索引超出表就失敗即關閉——超出範圍時原版會讀到
// 表外的記憶體，remake 不模仿那個行為，但也不能安靜地回一筆空的。
func (t *ItemTypeTable) Entry(itemType uint8) (ItemTypeEntry, error) {
	if int(itemType) >= ItemTypeCount {
		return ItemTypeEntry{}, fmt.Errorf("Pool item type %#02x is outside 0..%d", itemType, ItemTypeCount-1)
	}
	return t.Entries[itemType], nil
}

// ParseItemTypeTable 解出 `poolrad/items` 的內容。
func ParseItemTypeTable(raw []byte) (*ItemTypeTable, error) {
	want := itemTypeHeaderSize + ItemTypeCount*ItemTypeEntrySize
	if len(raw) != want {
		return nil, fmt.Errorf("Pool item type table has %d bytes, want %d", len(raw), want)
	}
	table := &ItemTypeTable{Header: binary.LittleEndian.Uint16(raw[:itemTypeHeaderSize])}
	body := raw[itemTypeHeaderSize:]
	for index := range table.Entries {
		copy(table.Entries[index].Raw[:], body[index*ItemTypeEntrySize:(index+1)*ItemTypeEntrySize])
	}
	return table, nil
}

// ReadDOSItemTypeTable 從 DOS ZIP 讀出物品型別表。
func ReadDOSItemTypeTable(zipPath string) (*ItemTypeTable, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()

	var member *zip.File
	for _, candidate := range archive.File {
		if !strings.EqualFold(filepath.Base(candidate.Name), itemTypeFileName) {
			continue
		}
		if member != nil {
			return nil, fmt.Errorf("DOS ZIP has duplicate %s", itemTypeFileName)
		}
		member = candidate
	}
	if member == nil {
		return nil, fmt.Errorf("DOS ZIP has no %s", itemTypeFileName)
	}
	reader, err := member.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", itemTypeFileName, err)
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", itemTypeFileName, err)
	}
	return ParseItemTypeTable(raw)
}
