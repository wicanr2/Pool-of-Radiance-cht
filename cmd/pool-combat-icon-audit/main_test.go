package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// 合成一份 DAX：本工具的判讀規則要在**自己造的輸入**上驗，不能只在原版檔上
// 跑——原版檔不在版控裡，而且拿真檔驗不出「壞掉的區塊會不會被算成好的」。
func daxBytes(blocks []struct {
	ID      uint8
	Payload []byte
}) []byte {
	header := make([]byte, 2+len(blocks)*9)
	binary.LittleEndian.PutUint16(header[:2], uint16(len(blocks)*9))
	body := []byte{}
	for index, block := range blocks {
		packed := rleLiteral(block.Payload)
		entry := header[2+index*9:]
		entry[0] = block.ID
		binary.LittleEndian.PutUint32(entry[1:5], uint32(len(body)))
		binary.LittleEndian.PutUint16(entry[5:7], uint16(len(block.Payload)))
		binary.LittleEndian.PutUint16(entry[7:9], uint16(len(packed)))
		body = append(body, packed...)
	}
	return append(header, body...)
}

// DAX 的 RLE：控制碼 >= 0 是「接下來 control+1 個位元組照抄」，一段最多 128。
func rleLiteral(data []byte) []byte {
	out := []byte{}
	for pos := 0; pos < len(data); {
		count := len(data) - pos
		if count > 128 {
			count = 128
		}
		out = append(out, byte(count-1))
		out = append(out, data[pos:pos+count]...)
		pos += count
	}
	return out
}

// 一張 widthUnits×heightUnits 的圖：17 bytes 標頭，之後每個位元組兩個像素。
func picturePayload(widthUnits, heightUnits uint16, items uint8, fill byte) []byte {
	data := make([]byte, 17)
	binary.LittleEndian.PutUint16(data[0:2], heightUnits)
	binary.LittleEndian.PutUint16(data[2:4], widthUnits)
	data[8] = items
	size := int(widthUnits) * 8 * int(heightUnits) / 2 * int(items)
	return append(data, bytes.Repeat([]byte{fill}, size)...)
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

func oneBlockDAX(id uint8, widthUnits, heightUnits uint16) []byte {
	return daxBytes([]struct {
		ID      uint8
		Payload []byte
	}{{ID: id, Payload: picturePayload(widthUnits, heightUnits, 1, 0x12)}})
}

// 戰鬥造形只有兩個容器：`CHEAD.DAX` 與 `CBODY.DAX`。同一個 ZIP 裡其他 `.DAX`
// 不該被算進去——算進去的話「造形有幾塊」這個數字就摻了別的東西。
func TestAuditCountsOnlyTheTwoCombatIconArchives(t *testing.T) {
	path := writeZIP(t, map[string][]byte{
		"CHEAD.DAX": oneBlockDAX(3, 2, 8),
		"CBODY.DAX": oneBlockDAX(1, 2, 16),
		"TITLE.DAX": oneBlockDAX(1, 40, 200),
		"NOTES.TXT": []byte("不是 DAX"),
	})
	result, err := audit(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Archives != 2 || result.Blocks != 2 || result.Decoded != 2 || result.Failed != 0 {
		t.Fatalf("盤點結果是 %+v", result)
	}
	// 排序：先檔名再區塊編號，所以 CBODY 在 CHEAD 前面。
	if result.Rows[0].Source != "CBODY.DAX" || result.Rows[1].Source != "CHEAD.DAX" {
		t.Fatalf("排序是 %s、%s", result.Rows[0].Source, result.Rows[1].Source)
	}
	if got := result.Rows[1].Width; got != 16 {
		t.Fatalf("CHEAD 的區塊寬 %d，該是 2 個單位 ×8", got)
	}
	if got := result.Rows[1].Height; got != 8 {
		t.Fatalf("CHEAD 的區塊高 %d", got)
	}
}

// 少一個容器就不是完整的盤點——這一支的結論是「全部造形」，缺一半的報告
// 看起來與完整的一模一樣，只是數字小一點。
func TestAuditRefusesWhenAnArchiveIsMissing(t *testing.T) {
	path := writeZIP(t, map[string][]byte{"CHEAD.DAX": oneBlockDAX(3, 2, 8)})
	if _, err := audit(path); err == nil {
		t.Fatal("只有一個容器卻沒報錯")
	}
}

// 解不開的區塊要記成失敗，不能被當成沒有這一塊。`main` 靠 `Failed` 決定
// 離開碼 2，所以這個計數是對外的訊號。
func TestAuditRecordsUndecodableBlocks(t *testing.T) {
	broken := daxBytes([]struct {
		ID      uint8
		Payload []byte
	}{{ID: 5, Payload: make([]byte, 17)}}) // 尺寸全 0 的標頭
	path := writeZIP(t, map[string][]byte{
		"CHEAD.DAX": broken,
		"CBODY.DAX": oneBlockDAX(1, 2, 16),
	})
	result, err := audit(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 1 || result.Decoded != 1 {
		t.Fatalf("解得開 %d、解不開 %d", result.Decoded, result.Failed)
	}
	for _, row := range result.Rows {
		if row.Source == "CHEAD.DAX" && row.Error == "" {
			t.Fatal("解不開的區塊沒有留下原因")
		}
	}
}
