package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPoolDOSOverlayManifest(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip(err)
	}
	out := filepath.Join(t.TempDir(), "manifest.json")
	extractDir := filepath.Join(t.TempDir(), "overlays")
	if err := run(zipPath, out, extractDir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var got report
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.InputZIP.SHA256 != "1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633" {
		t.Fatalf("ZIP SHA-256 = %s", got.InputZIP.SHA256)
	}
	if got.OverlayCount != 38 || got.EntryCount != 774 {
		t.Fatalf("overlays/entries = %d/%d, want 38/774", got.OverlayCount, got.EntryCount)
	}
	if len(got.Overlays) != got.OverlayCount {
		t.Fatalf("overlay rows = %d", len(got.Overlays))
	}
	if got.Overlays[0].Index != 0 || got.Overlays[37].Index != 37 {
		t.Fatalf("overlay IDs = %d..%d, want zero-based 0..37", got.Overlays[0].Index, got.Overlays[37].Index)
	}
	files, err := filepath.Glob(filepath.Join(extractDir, "overlay-*.bin"))
	if err != nil || len(files) != 38 {
		t.Fatalf("extracted overlays = %d, err=%v", len(files), err)
	}
}
