package gamepack

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"
)

// 怪物身上的物品串列（記錄 `+C8h`）從哪裡來（spec 142，issue #76）。
//
// overlay-17 `0E90h` 是 `0Bh LOAD MONSTER` 用的載入常式，三個檔依序讀同一個
// block 編號（`[bp+0Ah]`），檔名都是 `"MON" + Str(DS:52D4h) + 後綴 + ".dax"`：
//
//	0EB7  "MON" (`0E64h`) + Str(52D4h) + "CHA" (`0E68h`)  → 0F28h 複製 11Dh bytes 進記錄
//	0F3Bh 起清掉 +C8h、+7Fh、+104h、+108h 四個遠指標
//	1051  "MON" + Str(52D4h) + "SPC" (`0E88h`)            → 9 bytes 一節，鏈在 +7Fh、下一節在 +5
//	112E  "MON" + Str(52D4h) + "ITM" (`0E8Ch`)            → 3Fh bytes 一節，鏈在 +C8h、下一節在 +2Ah
//
// 物品那一段（`1178h..11F7h`）：offset 從 0 起，每一輪 GetMem(3Fh)、Move(block+offset,
// 節點, 3Fh)、清掉節點的 +2Ah／+2Ch（下一節指標），offset += 3Fh，offset < 長度就再接一節。
// block 不存在（長度 0，`116Fh`）就是沒有東西。所以串列的順序就是 block 裡的順序，
// 每一件的 63 bytes 原封不動（名字、型別、+34h 穿戴中、+3Ch 次數、+3Dh 法術都照檔）。

// MonsterItemRecordSize 是一件物品的長度（overlay-17 `1186h`、`11AEh`、`11C4h` 的 3Fh）。
const MonsterItemRecordSize = 0x3F

// MonsterMoneyOffset 是記錄裡七種錢的起點：`+88h + i × 2`（word，i = 0..6）。戰後
// overlay-05 entry 2 `0089h..00BEh` 從這裡逐欄加進公款 `DS:6752h + i × 4`；隊員的錢包
// 也是這七欄（spec 040）。
const MonsterMoneyOffset = 0x88

// ReadDOSMonsterItems 讀同一隻怪物在 MONnITM.DAX 裡的物品串列。沒有這個檔或沒有這個
// block 都是空串列（原版 `116Fh` 長度 0 就整段跳過）。block 存在時長度一定要是 3Fh 的
// 整數倍，否則失敗即關閉——原版不檢查，照讀會讀出界。
func ReadDOSMonsterItems(zipPath string, archiveNumber, blockID uint8) ([][]byte, error) {
	if archiveNumber < 1 || archiveNumber > 8 {
		return nil, fmt.Errorf("Pool monster archive %d is outside 1..8", archiveNumber)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer zr.Close()

	name := fmt.Sprintf("MON%dITM.DAX", archiveNumber)
	found := false
	for _, candidate := range zr.File {
		if strings.EqualFold(filepath.Base(candidate.Name), name) {
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}
	member, err := uniqueMember(zr.File, name)
	if err != nil {
		return nil, err
	}
	blocks, err := readDAXBlocks(member)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	payload, ok := blocks[blockID]
	if !ok {
		return nil, nil
	}
	items, err := ParseMonsterItems(payload)
	if err != nil {
		return nil, fmt.Errorf("%s block %d: %w", name, blockID, err)
	}
	return items, nil
}

// ParseMonsterItems 把一個 MONnITM block 切成 3Fh bytes 一件（`11C4h` 的步長）。
func ParseMonsterItems(payload []byte) ([][]byte, error) {
	if len(payload)%MonsterItemRecordSize != 0 {
		return nil, fmt.Errorf("Pool monster item block has %d bytes, want a multiple of %d",
			len(payload), MonsterItemRecordSize)
	}
	items := make([][]byte, 0, len(payload)/MonsterItemRecordSize)
	for offset := 0; offset < len(payload); offset += MonsterItemRecordSize {
		item := append([]byte(nil), payload[offset:offset+MonsterItemRecordSize]...)
		// `11BAh`：下一節的遠指標清成 0；檔裡那四格是存檔時的殘值。
		for index := 0x2A; index < 0x2E; index++ {
			item[index] = 0
		}
		items = append(items, item)
	}
	return items, nil
}
