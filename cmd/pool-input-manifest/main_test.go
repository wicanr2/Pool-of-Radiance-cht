package main

import (
	"path/filepath"
	"testing"
)

func TestFixedDOSInputManifest(t *testing.T) {
	result, err := buildManifest(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Fatal(err)
	}
	if result.ZIPSHA256 != "1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633" {
		t.Fatalf("unexpected ZIP hash %s", result.ZIPSHA256)
	}
	if result.Files != 168 || result.DecodedBytes != 1582291 {
		t.Fatalf("unexpected corpus shape: files=%d bytes=%d", result.Files, result.DecodedBytes)
	}
	for _, entry := range result.Entries {
		if entry.Path == "poolrad/start.exe" {
			if entry.SHA256 != "12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f" {
				t.Fatalf("unexpected START.EXE hash %s", entry.SHA256)
			}
			return
		}
	}
	t.Fatal("manifest does not contain poolrad/start.exe")
}
