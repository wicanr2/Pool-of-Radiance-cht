package main

import (
	"os"
	"path/filepath"
	"testing"
)

// 完整版：沒給 -zip／-eten-font、目前目錄也沒有 ZIP 時，改用執行檔旁 data/ 裡的檔。
func TestBundledZipAndFontBesideTheExecutable(t *testing.T) {
	base := t.TempDir()
	data := filepath.Join(base, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	const zipName = "Pool of Radiance (1988).zip"
	for _, name := range []string{zipName, "stdfont.15"} {
		if err := os.WriteFile(filepath.Join(data, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	saved := executableDir
	executableDir = func() string { return base }
	defer func() { executableDir = saved }()

	zipPath := filepath.Join(t.TempDir(), zipName) // 目前目錄沒有這個檔
	font := ""
	resolveBundledInputs(&zipPath, &font)
	if zipPath != filepath.Join(data, zipName) {
		t.Fatalf("zip = %q, want the bundled one", zipPath)
	}
	if font != filepath.Join(data, "stdfont.15") {
		t.Fatalf("font = %q, want the bundled one", font)
	}

	// 目前目錄有 ZIP、字型有指定：照玩家給的，不換。
	here := filepath.Join(t.TempDir(), zipName)
	if err := os.WriteFile(here, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath, font = here, "/my/stdfont.15"
	resolveBundledInputs(&zipPath, &font)
	if zipPath != here || font != "/my/stdfont.15" {
		t.Fatalf("explicit inputs were replaced: %q %q", zipPath, font)
	}

	// patch 包沒有 data/：什麼都不換，照舊由缺檔的錯誤說明。
	executableDir = func() string { return t.TempDir() }
	zipPath, font = filepath.Join(t.TempDir(), zipName), ""
	missing := zipPath
	resolveBundledInputs(&zipPath, &font)
	if zipPath != missing || font != "" {
		t.Fatalf("without data/ nothing should change: %q %q", zipPath, font)
	}
}
