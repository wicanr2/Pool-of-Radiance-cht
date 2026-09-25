package gamepack

import "fmt"

// 物品名稱的字詞表（spec 067〈鑑定〉）。overlay-25 entry 1（`0441h`）組物品名稱時，
// 以 `DS:10BBh + 字詞編號 × 15h` 取一段 Pascal 字串（`0610h..061Dh`：
// `mov dx,15h; mul dx; add di,10BBh`）。字詞編號是 byte，所以表有 256 格；
// 第 0 格不會被取用（編號 0 代表「這一段沒有字」），內容是別的資料，照樣保留。
//
// 這段 DS 在 START.EXE 裡是已初始化的常數（與 `DS:54E0h` 那張執行時才載入的
// 物品型別表不同），位移換算沿用 `startDataSegmentFileDelta`。
const (
	ItemNameTableAddress = 0x10BB
	ItemNameEntrySize    = 0x15
	ItemNameEntryCount   = 256
	// itemNameMaxLength 是每格字串的上限：21 bytes 扣掉長度 byte。
	itemNameMaxLength = ItemNameEntrySize - 1
)

// ItemNameTable 是整張字詞表，索引就是物品記錄 `+2Fh`／`+30h`／`+31h` 的值。
type ItemNameTable [ItemNameEntryCount]string

// ParseItemNameTable 解出 256 格 × 21 bytes 的表。第 0 格的長度 byte 會超出
// 20，那一格不是字詞，留空。其餘格長度超過 20 就失敗即關閉。
func ParseItemNameTable(raw []byte) (ItemNameTable, error) {
	var table ItemNameTable
	if len(raw) != ItemNameEntryCount*ItemNameEntrySize {
		return table, fmt.Errorf("Pool item name table has %d bytes, want %d", len(raw), ItemNameEntryCount*ItemNameEntrySize)
	}
	for index := 1; index < ItemNameEntryCount; index++ {
		entry := raw[index*ItemNameEntrySize : (index+1)*ItemNameEntrySize]
		length := int(entry[0])
		if length > itemNameMaxLength {
			return table, fmt.Errorf("Pool item name %#02x has length %d, want at most %d", index, length, itemNameMaxLength)
		}
		table[index] = string(entry[1 : 1+length])
	}
	return table, nil
}

// ReadDOSItemNameTable 從 DOS ZIP 的 START.EXE 讀出字詞表。
func ReadDOSItemNameTable(zipPath string) (ItemNameTable, error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return ItemNameTable{}, err
	}
	offset := ItemNameTableAddress + startDataSegmentFileDelta
	end := offset + ItemNameEntryCount*ItemNameEntrySize
	if len(raw) < end {
		return ItemNameTable{}, fmt.Errorf("START.EXE is %d bytes, the item name table needs %d", len(raw), end)
	}
	return ParseItemNameTable(raw[offset:end])
}
