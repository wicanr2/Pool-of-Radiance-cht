package gamepack

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// ECLArchive preserves one DOS ECL DAX namespace. Block IDs are only unique
// within an archive and must never be merged across ECL1..8.
type ECLArchive struct {
	Number uint8
	Blocks map[uint16][]byte
}

type ECLCatalog struct {
	archives map[uint8]ECLArchive
}

func ReadDOSECLCatalog(zipPath string) (ECLCatalog, error) {
	result := ECLCatalog{archives: make(map[uint8]ECLArchive, 8)}
	for archiveNumber := uint8(1); archiveNumber <= 8; archiveNumber++ {
		archive, err := ReadDOSECLArchive(zipPath, archiveNumber)
		if err != nil {
			return ECLCatalog{}, err
		}
		result.archives[archiveNumber] = archive
	}
	return result, nil
}

func (catalog ECLCatalog) Archive(number uint8) (ECLArchive, bool) {
	archive, ok := catalog.archives[number]
	if !ok {
		return ECLArchive{}, false
	}
	blocks := make(map[uint16][]byte, len(archive.Blocks))
	for id, block := range archive.Blocks {
		blocks[id] = append([]byte(nil), block...)
	}
	archive.Blocks = blocks
	return archive, true
}

func ReadDOSECLArchive(zipPath string, archiveNumber uint8) (ECLArchive, error) {
	if archiveNumber < 1 || archiveNumber > 8 {
		return ECLArchive{}, fmt.Errorf("Pool ECL archive %d is outside 1..8", archiveNumber)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return ECLArchive{}, err
	}
	defer zr.Close()
	want := fmt.Sprintf("ECL%d.DAX", archiveNumber)
	var member *zip.File
	for _, candidate := range zr.File {
		if !strings.EqualFold(filepath.Base(candidate.Name), want) {
			continue
		}
		if member != nil {
			return ECLArchive{}, fmt.Errorf("Pool DOS ZIP contains duplicate %s", want)
		}
		member = candidate
	}
	if member == nil {
		return ECLArchive{}, fmt.Errorf("Pool DOS ZIP is missing %s", want)
	}
	stream, err := member.Open()
	if err != nil {
		return ECLArchive{}, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(stream, 16<<20))
	closeErr := stream.Close()
	if readErr != nil {
		return ECLArchive{}, readErr
	}
	if closeErr != nil {
		return ECLArchive{}, closeErr
	}
	blocks, err := dax.Parse(raw)
	if err != nil {
		return ECLArchive{}, fmt.Errorf("parse %s: %w", want, err)
	}
	result := ECLArchive{Number: archiveNumber, Blocks: make(map[uint16][]byte, len(blocks))}
	for _, block := range blocks {
		id := uint16(block.Entry.ID)
		if _, exists := result.Blocks[id]; exists {
			return ECLArchive{}, fmt.Errorf("%s repeats block 0x%X", want, id)
		}
		result.Blocks[id] = append([]byte(nil), block.Data...)
	}
	return result, nil
}

func NewDOSECLArchiveSession(archive ECLArchive, blockID uint16, startAddress uint16, characters ...InitialCharacter) (*eclvm.BlockSession, error) {
	if archive.Number < 1 || archive.Number > 8 || len(archive.Blocks) == 0 {
		return nil, fmt.Errorf("Pool ECL archive catalog is invalid")
	}
	if startAddress < 0x9900 {
		return nil, fmt.Errorf("Pool ECL start 0x%04X precedes code base", startAddress)
	}
	session, err := eclvm.NewBlockSession(archive.Blocks, blockID, 0x9900, int(startAddress)-0x9900, 5, initialEventPassthrough(), 1)
	if err != nil {
		return nil, err
	}
	if err := session.SetTransitionEntries(0, 4); err != nil {
		return nil, err
	}
	// 解碼用 Pool 自己量出來的指令表，不是共用 engine 那張二手的（spec 093）。
	// **讀檔這條路也要設**：`Restore` 會換掉 machine，而 engine 那一側搬的是
	// 這一份，沒設就整個 session 都在用二手表。
	session.Machine().SetCommands(PoolCommandTable())
	session.Machine().SetCharacterProjector(initialCharacterProjector(characters))
	session.Machine().SetPartyStrengthResolver(initialPartyStrengthResolver(characters))
	return session, nil
}
