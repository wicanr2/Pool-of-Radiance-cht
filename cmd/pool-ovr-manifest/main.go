package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

type report struct {
	Schema       string          `json:"schema"`
	InputZIP     fileEvidence    `json:"input_zip"`
	Executable   fileEvidence    `json:"executable"`
	OverlayFile  fileEvidence    `json:"overlay_file"`
	OverlayCount int             `json:"overlay_count"`
	EntryCount   int             `json:"entry_count"`
	Overlays     []overlayReport `json:"overlays"`
}

type fileEvidence struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type overlayReport struct {
	Index                int           `json:"index"`
	ControlFileOffset    string        `json:"control_file_offset"`
	ExecutableFileOffset string        `json:"executable_file_offset"`
	CodeBytes            int           `json:"code_bytes"`
	RelocationBytes      int           `json:"relocation_bytes"`
	CodeSHA256           string        `json:"code_sha256"`
	RelocationSHA256     string        `json:"relocation_sha256"`
	RelocationOffsets    []string      `json:"relocation_offsets"`
	Entries              []entryReport `json:"entries"`
}

type entryReport struct {
	Index                int    `json:"index"`
	ExecutableFileOffset string `json:"executable_file_offset"`
	StubOffset           string `json:"stub_offset"`
	CodeOffset           string `json:"code_offset"`
	Flags                string `json:"flags"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "original DOS ZIP")
	outPath := flag.String("out", "", "JSON output; stdout when empty")
	extractDir := flag.String("extract-dir", "", "optional directory for transient overlay code spans")
	flag.Parse()
	if err := run(*zipPath, *outPath, *extractDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(zipPath, outPath, extractDir string) error {
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	exeName, exe, err := readNamed(zr.File, "START.EXE")
	if err != nil {
		return err
	}
	ovrName, ovr, err := readNamed(zr.File, "GAME.OVR")
	if err != nil {
		return err
	}
	decoded, err := tpov.Decode(exe, ovr)
	if err != nil {
		return err
	}
	r := report{Schema: "pool-dos-tpov-manifest-v1", InputZIP: evidence(filepath.Base(zipPath), zipBytes), Executable: evidence(exeName, exe), OverlayFile: evidence(ovrName, ovr), OverlayCount: len(decoded)}
	for index, overlay := range decoded {
		if extractDir != "" {
			if err := os.MkdirAll(extractDir, 0o755); err != nil {
				return err
			}
			name := filepath.Join(extractDir, fmt.Sprintf("overlay-%02d.bin", index))
			if err := os.WriteFile(name, overlay.Code, 0o644); err != nil {
				return err
			}
		}
		or := overlayReport{Index: index, ControlFileOffset: hexOffset(uint64(overlay.FileOffset)), ExecutableFileOffset: hexOffset(uint64(overlay.ExecutableOffset)), CodeBytes: len(overlay.Code), RelocationBytes: len(overlay.Relocation), CodeSHA256: digest(overlay.Code), RelocationSHA256: digest(overlay.Relocation)}
		for _, offset := range overlay.RelocationOffsets {
			or.RelocationOffsets = append(or.RelocationOffsets, hexOffset(uint64(offset)))
		}
		for _, entry := range overlay.Entries {
			or.Entries = append(or.Entries, entryReport{entry.Index, hexOffset(uint64(entry.ExecutableOffset)), hexOffset(uint64(entry.StubOffset)), hexOffset(uint64(entry.CodeOffset)), hexOffset(uint64(entry.Flags))})
			r.EntryCount++
		}
		r.Overlays = append(r.Overlays, or)
	}
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if outPath == "" {
		_, err = os.Stdout.Write(encoded)
		return err
	}
	return os.WriteFile(outPath, encoded, 0o644)
}

func readNamed(files []*zip.File, want string) (string, []byte, error) {
	var matches []*zip.File
	for _, f := range files {
		if strings.EqualFold(filepath.Base(f.Name), want) {
			matches = append(matches, f)
		}
	}
	if len(matches) != 1 {
		return "", nil, fmt.Errorf("ZIP contains %d %s files, want exactly one", len(matches), want)
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Name < matches[j].Name })
	r, err := matches[0].Open()
	if err != nil {
		return "", nil, err
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	return matches[0].Name, b, err
}

func evidence(name string, data []byte) fileEvidence {
	return fileEvidence{Name: name, Bytes: len(data), SHA256: digest(data)}
}
func digest(data []byte) string     { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func hexOffset(value uint64) string { return fmt.Sprintf("0x%X", value) }
