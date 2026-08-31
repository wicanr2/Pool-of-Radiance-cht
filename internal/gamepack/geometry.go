// Package gamepack adapts Pool-owned data to reusable engine types.
package gamepack

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/geometry"
)

const maxGEOArchiveBytes = 1 << 20

// MapKey is one Pool GEO block's stable legacy identity.
type MapKey struct {
	Archive uint8
	BlockID uint8
}

// Spawn is a Pool-owned entry into one legacy geometry block. Direction uses
// the original 0..7 facing values; movement policy remains map/ECL context.
type Spawn struct {
	Map    MapKey
	X      uint8
	Y      uint8
	Facing uint8
}

// DOSInitialSpawn is the normal new-party entry established by Spec 009.
func DOSInitialSpawn() Spawn {
	return Spawn{Map: MapKey{Archive: 3, BlockID: 0}, X: 15, Y: 1, Facing: 6}
}

// GeometryMap preserves Pool's archive identity and two-byte GEO prefix.
type GeometryMap struct {
	Key    MapKey
	Prefix [2]uint8
	Grid   geometry.Grid
}

// GeometryCatalog is the complete DOS GEO corpus without guessed story names,
// entry points, or movement policy.
type GeometryCatalog struct {
	maps map[MapKey]GeometryMap
}

// ReadDOSGeometryCatalog reads exactly GEO1.DAX..GEO8.DAX and fails closed on
// missing archives, duplicate identities, malformed DAX, or malformed GEO.
func ReadDOSGeometryCatalog(zipPath string) (GeometryCatalog, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return GeometryCatalog{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()

	byArchive := make(map[uint8]*zip.File, 8)
	for _, member := range archive.File {
		number, ok := geoArchiveNumber(filepath.Base(member.Name))
		if !ok {
			continue
		}
		if _, exists := byArchive[number]; exists {
			return GeometryCatalog{}, fmt.Errorf("DOS ZIP has duplicate GEO%d.DAX", number)
		}
		byArchive[number] = member
	}
	if len(byArchive) != 8 {
		return GeometryCatalog{}, fmt.Errorf("DOS ZIP has %d GEO archives, want 8", len(byArchive))
	}

	result := GeometryCatalog{maps: make(map[MapKey]GeometryMap, 29)}
	for archiveNumber := uint8(1); archiveNumber <= 8; archiveNumber++ {
		member := byArchive[archiveNumber]
		if member == nil {
			return GeometryCatalog{}, fmt.Errorf("DOS ZIP has no GEO%d.DAX", archiveNumber)
		}
		data, err := readBoundedZIPMember(member, maxGEOArchiveBytes)
		if err != nil {
			return GeometryCatalog{}, fmt.Errorf("read GEO%d.DAX: %w", archiveNumber, err)
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return GeometryCatalog{}, fmt.Errorf("parse GEO%d.DAX: %w", archiveNumber, err)
		}
		for _, block := range blocks {
			key := MapKey{Archive: archiveNumber, BlockID: block.Entry.ID}
			if _, exists := result.maps[key]; exists {
				return GeometryCatalog{}, fmt.Errorf("GEO%d.DAX has duplicate block 0x%02X", archiveNumber, block.Entry.ID)
			}
			grid, err := geometry.Parse(block.Entry.ID, block.Data)
			if err != nil {
				return GeometryCatalog{}, fmt.Errorf("GEO%d.DAX block 0x%02X: %w", archiveNumber, block.Entry.ID, err)
			}
			result.maps[key] = GeometryMap{Key: key, Prefix: [2]uint8{block.Data[0], block.Data[1]}, Grid: grid}
		}
	}
	if len(result.maps) != 29 {
		return GeometryCatalog{}, fmt.Errorf("DOS GEO corpus has %d maps, want 29", len(result.maps))
	}
	return result, nil
}

func geoArchiveNumber(name string) (uint8, bool) {
	upper := strings.ToUpper(name)
	if len(upper) != len("GEO1.DAX") || !strings.HasPrefix(upper, "GEO") || !strings.HasSuffix(upper, ".DAX") {
		return 0, false
	}
	number, err := strconv.Atoi(upper[3:4])
	if err != nil || number < 1 || number > 8 {
		return 0, false
	}
	return uint8(number), true
}

func readBoundedZIPMember(member *zip.File, limit int64) ([]byte, error) {
	if member.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("member is larger than %d bytes", limit)
	}
	stream, err := member.Open()
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, limit+1))
	closeErr := stream.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(data) > int(limit) || uint64(len(data)) != member.UncompressedSize64 {
		return nil, fmt.Errorf("member exceeds declared or bounded size")
	}
	return data, nil
}

func (c GeometryCatalog) Len() int { return len(c.maps) }

// Map returns a value copy; door mutations cannot alter the source catalog.
func (c GeometryCatalog) Map(key MapKey) (GeometryMap, bool) {
	value, ok := c.maps[key]
	return value, ok
}

// Keys returns identities in deterministic archive/block order.
func (c GeometryCatalog) Keys() []MapKey {
	keys := make([]MapKey, 0, len(c.maps))
	for key := range c.maps {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Archive != keys[j].Archive {
			return keys[i].Archive < keys[j].Archive
		}
		return keys[i].BlockID < keys[j].BlockID
	})
	return keys
}
