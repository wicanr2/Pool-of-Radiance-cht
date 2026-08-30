// Command pool-input-manifest inventories the fixed DOS source ZIP without
// extracting or modifying its contents.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type fileRecord struct {
	Path   string `json:"path"`
	Bytes  uint64 `json:"bytes"`
	CRC32  string `json:"crc32"`
	SHA256 string `json:"sha256"`
}

type manifest struct {
	Schema       string       `json:"schema"`
	ZIP          string       `json:"zip"`
	ZIPSHA256    string       `json:"zip_sha256"`
	Files        int          `json:"files"`
	DecodedBytes uint64       `json:"decoded_bytes"`
	Entries      []fileRecord `json:"entries"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	flag.Parse()
	result, err := buildManifest(*zipPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildManifest(zipPath string) (manifest, error) {
	zipDigest, err := hashFile(zipPath)
	if err != nil {
		return manifest{}, err
	}
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return manifest{}, err
	}
	defer archive.Close()

	result := manifest{
		Schema:    "pool-dos-input-manifest/1",
		ZIP:       filepath.Base(zipPath),
		ZIPSHA256: zipDigest,
	}
	seen := make(map[string]bool)
	for _, member := range archive.File {
		if member.FileInfo().IsDir() {
			continue
		}
		if member.Name == "" || seen[member.Name] {
			return manifest{}, fmt.Errorf("duplicate or empty ZIP member %q", member.Name)
		}
		seen[member.Name] = true
		stream, err := member.Open()
		if err != nil {
			return manifest{}, fmt.Errorf("open %s: %w", member.Name, err)
		}
		digest := sha256.New()
		crc := crc32.NewIEEE()
		read, copyErr := io.Copy(io.MultiWriter(digest, crc), io.LimitReader(stream, int64(member.UncompressedSize64)+1))
		closeErr := stream.Close()
		if copyErr != nil {
			return manifest{}, fmt.Errorf("read %s: %w", member.Name, copyErr)
		}
		if closeErr != nil {
			return manifest{}, fmt.Errorf("close %s: %w", member.Name, closeErr)
		}
		if read != int64(member.UncompressedSize64) {
			return manifest{}, fmt.Errorf("%s decoded %d bytes, expected %d", member.Name, read, member.UncompressedSize64)
		}
		if crc.Sum32() != member.CRC32 {
			return manifest{}, fmt.Errorf("%s CRC mismatch", member.Name)
		}
		result.Entries = append(result.Entries, fileRecord{
			Path:   member.Name,
			Bytes:  member.UncompressedSize64,
			CRC32:  fmt.Sprintf("%08x", member.CRC32),
			SHA256: hex.EncodeToString(digest.Sum(nil)),
		})
		result.DecodedBytes += member.UncompressedSize64
	}
	sort.Slice(result.Entries, func(i, j int) bool { return result.Entries[i].Path < result.Entries[j].Path })
	result.Files = len(result.Entries)
	return result, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
