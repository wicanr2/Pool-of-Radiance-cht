package gamepack

import (
	"encoding/binary"
	"path/filepath"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// 工具鏈的指紋（`docs/re/dos-toolchain-baseline.md`）。
//
// 這一則把「Pool 是 Turbo Pascal 5.x 家族、帶 Overlay 單元」那幾條證據
// 從原版位元組重生一次。少了它，基線那份文件會在資料換版時默默過期。
func TestDOSToolchainFingerprint(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}

	// MZ header：頭段 59 段（`3B0h`），進入點 `0000:0006`。
	if executable[0] != 'M' || executable[1] != 'Z' {
		t.Fatalf("START.EXE 不是 MZ：% X", executable[:2])
	}
	headerParagraphs := binary.LittleEndian.Uint16(executable[8:])
	entryIP := binary.LittleEndian.Uint16(executable[20:])
	entryCS := binary.LittleEndian.Uint16(executable[22:])
	if headerParagraphs != 59 || entryIP != 6 || entryCS != 0 {
		t.Fatalf("header=%d 段 進入點 %04X:%04X", headerParagraphs, entryCS, entryIP)
	}
	codeBase := int(headerParagraphs) * 16

	// Turbo Pascal 的 runtime error 訊息是三個連著的 ASCIIZ。
	// 這一組（沒有「Divide by zero」那類個別訊息）是 TP 4 之後的樣子：
	// 錯誤碼用數字印，不是查表印字。
	wantMessages := []byte("Runtime error \x00 at \x00.\r\n\x00")
	messages := indexOf(executable, wantMessages)
	if messages < 0 {
		t.Fatal("找不到 Turbo Pascal 的 runtime error 訊息三連")
	}

	// 訊息前面那一支是把一個 nibble 印到主控台的 RTL 輔助函式：
	//	pop ax / and al,0Fh / add al,'0' / cmp al,':' / jb +2 / add al,7
	//	mov dl,al / mov ah,6 / int 21h / ret
	// DOS 功能 06h 直接輸出——TP 的錯誤處理不走檔案系統，因為那時可能已經壞了。
	nibble := []byte{0x58, 0x24, 0x0F, 0x04, 0x30, 0x3C, 0x3A, 0x72, 0x02, 0x04, 0x07,
		0x8A, 0xD0, 0xB4, 0x06, 0xCD, 0x21, 0xC3}
	if at := indexOf(executable, nibble); at < 0 || at > messages {
		t.Fatalf("印 nibble 的 RTL 輔助函式在 %d，訊息在 %d", at, messages)
	}

	// 進入點是 Turbo Pascal 產生的「逐一呼叫每個單元的初始化」鏈。
	// 常駐單元直接呼叫 `segment:0000h`／別的位移，overlay 化的單元一律
	// 呼叫自己的 stub `segment:0020h`——那正是 entry 0，也就是單元初始化。
	entry := codeBase + int(entryCS)*16 + int(entryIP)
	farCalls, stubInits := 0, 0
	for offset := entry; offset+5 <= len(executable); offset += 5 {
		if executable[offset] != 0x9A {
			break
		}
		farCalls++
		if binary.LittleEndian.Uint16(executable[offset+1:]) == 0x0020 {
			stubInits++
		}
	}
	if farCalls < 20 || stubInits < 15 {
		t.Fatalf("進入點只有 %d 個遠呼叫、其中 %d 個是 stub 的 entry 0", farCalls, stubInits)
	}
	t.Logf("進入點是 %d 個連續遠呼叫，其中 %d 個進 overlay 的 entry 0", farCalls, stubInits)

	// GAME.OVR 是 Borland 的 overlay 容器，控制記錄在 EXE 裡串成一條鏈。
	overlay, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		t.Fatal(err)
	}
	if string(overlay[:4]) != "TPOV" {
		t.Fatalf("GAME.OVR 開頭是 % X，不是 TPOV", overlay[:4])
	}
	decoded, err := tpov.Decode(executable, overlay)
	if err != nil {
		t.Fatalf("解 GAME.OVR：%v", err)
	}
	if len(decoded) != 38 {
		t.Fatalf("解出 %d 顆 overlay，原版是 38 顆", len(decoded))
	}
}

func indexOf(haystack, needle []byte) int {
	for index := 0; index+len(needle) <= len(haystack); index++ {
		match := true
		for offset := range needle {
			if haystack[index+offset] != needle[offset] {
				match = false
				break
			}
		}
		if match {
			return index
		}
	}
	return -1
}
