package main

import (
	"archive/zip"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// 一份最小的合法 DAX：一個區塊，內容用 RLE 的字面段（控制碼 = 長度 − 1）。
func tinyDAX(id uint8, payload []byte) []byte {
	packed := append([]byte{byte(len(payload) - 1)}, payload...)
	out := make([]byte, 2+9)
	binary.LittleEndian.PutUint16(out[:2], 9)
	out[2] = id
	binary.LittleEndian.PutUint32(out[3:7], 0)
	binary.LittleEndian.PutUint16(out[7:9], uint16(len(payload)))
	binary.LittleEndian.PutUint16(out[9:11], uint16(len(packed)))
	return append(out, packed...)
}

func writeZIP(t *testing.T, members map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pool.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, data := range members {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// 盤點只看副檔名，而且**不分大小寫**——原版的檔名在不同來源裡大小寫不一，
// 只認大寫會整批漏掉。`entries` 數的是整包，不是只有 DAX。
func TestInventoryPicksEveryDAXRegardlessOfCase(t *testing.T) {
	result, err := inventory(writeZIP(t, map[string][]byte{
		"A.DAX":     tinyDAX(1, []byte("abc")),
		"b.dax":     tinyDAX(2, []byte("de")),
		"notes.txt": []byte("不是 DAX"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if result.Entries != 3 {
		t.Fatalf("entries 是 %d，該數整包", result.Entries)
	}
	if len(result.DAXFiles) != 2 || result.DAXOK != 2 || result.DAXFailed != 0 {
		t.Fatalf("盤點結果是 %+v", result)
	}
	for _, row := range result.DAXFiles {
		if row.Blocks != 1 {
			t.Fatalf("%s 數出 %d 個區塊", row.Name, row.Blocks)
		}
	}
	// 排序照檔名，報表才逐次可比。
	if result.DAXFiles[0].Name != "A.DAX" || result.DAXFiles[1].Name != "b.dax" {
		t.Fatalf("排序是 %s、%s", result.DAXFiles[0].Name, result.DAXFiles[1].Name)
	}
}

// 解不開的要留下原因並計進 `dax_failed`——`main` 靠它決定離開碼 2。
// 漏記的話一份壞掉的容器看起來就像「這個檔沒有區塊」。
func TestInventoryRecordsABrokenDAX(t *testing.T) {
	result, err := inventory(writeZIP(t, map[string][]byte{
		"good.DAX":   tinyDAX(1, []byte("abc")),
		"broken.DAX": {0x05, 0x00, 0x01},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if result.DAXOK != 1 || result.DAXFailed != 1 {
		t.Fatalf("好 %d 壞 %d", result.DAXOK, result.DAXFailed)
	}
	for _, row := range result.DAXFiles {
		if row.Name == "broken.DAX" && row.Error == "" {
			t.Fatal("壞掉的容器沒有留下原因")
		}
		if row.Name == "good.DAX" && row.Error != "" {
			t.Fatalf("好的容器被記成壞的：%s", row.Error)
		}
	}
}

// 開不了的 ZIP 是整支工具失敗，不是「零個 DAX」。
func TestInventoryFailsOnAnUnreadableZIP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.zip")
	if err := os.WriteFile(path, []byte("不是 zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := inventory(path); err == nil {
		t.Fatal("壞掉的 ZIP 沒有讓盤點失敗")
	}
}
