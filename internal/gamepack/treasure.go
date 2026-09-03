package gamepack

import (
	"errors"
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
)

const treasureItemRecordSize = 63

// TreasureItemRecord preserves one Pool ITEM record without assigning names
// to fields whose title consumer has not yet been closed.
type TreasureItemRecord struct {
	Name string
	Raw  [treasureItemRecordSize]byte
}

// ReadDOSTreasureItemBlock reads one exact ITEM archive/block identity. It
// deliberately returns raw records; equipment and value semantics stay
// fail-closed until their Pool consumers are evidenced.
func ReadDOSTreasureItemBlock(zipPath string, archiveNumber, blockID uint8) ([]TreasureItemRecord, error) {
	if archiveNumber < 1 || archiveNumber > 8 {
		return nil, fmt.Errorf("Pool ITEM archive %d is outside 1..8", archiveNumber)
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()

	name := fmt.Sprintf("ITEM%d.DAX", archiveNumber)
	var member *zip.File
	for _, candidate := range archive.File {
		if !strings.EqualFold(filepath.Base(candidate.Name), name) {
			continue
		}
		if member != nil {
			return nil, fmt.Errorf("DOS ZIP has duplicate %s", name)
		}
		member = candidate
	}
	if member == nil {
		return nil, fmt.Errorf("DOS ZIP has no %s", name)
	}
	data, err := readBoundedZIPMember(member, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	var payload []byte
	for _, block := range blocks {
		if block.Entry.ID != blockID {
			continue
		}
		if payload != nil {
			return nil, fmt.Errorf("%s has duplicate block 0x%02X", name, blockID)
		}
		payload = block.Data
	}
	if payload == nil {
		return nil, fmt.Errorf("%s has no block 0x%02X: %w", name, blockID, ErrTreasureBlockAbsent)
	}
	return parseTreasureItemRecords(payload)
}

// ErrTreasureBlockAbsent 說那個編號在 ITEM 檔裡沒有這一塊。呼叫端可以據此
// 分辨「這一格沒有東西」與「檔案讀壞了」——前者不該讓玩家路徑中斷。
var ErrTreasureBlockAbsent = errors.New("Pool ITEM block is absent")

func parseTreasureItemRecords(payload []byte) ([]TreasureItemRecord, error) {
	if len(payload) == 0 || len(payload)%treasureItemRecordSize != 0 {
		return nil, fmt.Errorf("Pool ITEM payload has %d bytes, want a positive multiple of %d", len(payload), treasureItemRecordSize)
	}
	records := make([]TreasureItemRecord, 0, len(payload)/treasureItemRecordSize)
	for offset := 0; offset < len(payload); offset += treasureItemRecordSize {
		raw := payload[offset : offset+treasureItemRecordSize]
		nameLength := int(raw[0])
		if nameLength < 1 || nameLength > 40 || 1+nameLength > len(raw) {
			return nil, fmt.Errorf("Pool ITEM record %d has invalid name length %d", offset/treasureItemRecordSize, nameLength)
		}
		record := TreasureItemRecord{Name: string(raw[1 : 1+nameLength])}
		copy(record.Raw[:], raw)
		records = append(records, record)
	}
	return records, nil
}
