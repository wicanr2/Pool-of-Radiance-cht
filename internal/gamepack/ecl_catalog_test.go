package gamepack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestReadDOSECLArchiveRejectsInvalidArchive(t *testing.T) {
	if _, err := ReadDOSECLArchive("unused.zip", 0); err == nil {
		t.Fatal("archive zero accepted")
	}
}

func TestReadDOSECLArchiveRejectsDuplicateMember(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplicate.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, name := range []string{"ECL2.DAX", "nested/ecl2.dax"} {
		entry, createErr := writer.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write([]byte("bad")); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDOSECLArchive(path, 2); err == nil {
		t.Fatal("duplicate ECL2 member accepted")
	}
}

func TestRealECL2Block20SlumsCompletionCounter(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	archive, err := ReadDOSECLArchive(zipPath, 2)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	session, err := NewDOSECLArchiveSession(archive, 20, 0xB69C)
	if err != nil {
		t.Fatal(err)
	}
	machine := session.Machine()
	for call := 1; call <= 25; call++ {
		if err := machine.SetPC(0xB69C - 0x9900); err != nil {
			t.Fatal(err)
		}
		if _, err := machine.Run(32, nil, true); err != nil {
			t.Fatalf("call %d: %v", call, err)
		}
		want := uint16(call)
		if call == 25 {
			want = 0xFE
		}
		if got := machine.Memory[0x4ABB]; got != want {
			t.Fatalf("call %d: 4ABB=%d, want %d", call, got, want)
		}
	}
	if err := machine.SetPC(0xB69C - 0x9900); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Run(32, nil, true); err != nil {
		t.Fatal(err)
	}
	if got := machine.Memory[0x4ABB]; got != 0xFE {
		t.Fatalf("acknowledged counter changed to %d", got)
	}
}

func TestRealDOSECLCatalogPreservesEightArchiveNamespaces(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	catalog, err := ReadDOSECLCatalog(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	total := 0
	for number := uint8(1); number <= 8; number++ {
		archive, ok := catalog.Archive(number)
		if !ok || archive.Number != number || len(archive.Blocks) == 0 {
			t.Fatalf("archive %d=%+v ok=%v", number, archive, ok)
		}
		total += len(archive.Blocks)
	}
	if total != 29 {
		t.Fatalf("ECL block total=%d, want 29", total)
	}
	first, _ := catalog.Archive(2)
	first.Blocks[20][0] ^= 0xFF
	second, _ := catalog.Archive(2)
	if first.Blocks[20][0] == second.Blocks[20][0] {
		t.Fatal("catalog leaked mutable block storage")
	}
}
