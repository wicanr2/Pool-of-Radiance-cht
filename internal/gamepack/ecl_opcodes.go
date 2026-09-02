package gamepack

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// ECL 直譯器的 opcode 派發鏈與運算元個數，直接從 overlay-03 量出來。
//
// 派發鏈是一長串 `cmp ax, opcode / jne 下一個 / push cs / call handler`；
// 兩個 opcode 共用一支常式時前面多一個 `je`。每支常式開頭都用同一個樣式取
// 運算元：`mov al, N / push ax / lcall 0045h:002Ah`，N 就是個數。
//
// 這是**第一手**的數字。engine 的 `ecl.KnownCommands` 來自公開的 ECL dump 表
// 與 CoAB 的重製，是二手的；兩邊對得起來才算數，對不起來要以位元組為準。
const (
	// ECLDispatchOverlay 是直譯器所在的 overlay。
	ECLDispatchOverlay = 3

	// eclOperandFetchSegment／Offset 是取運算元的遠呼叫目標。
	eclOperandFetchSegment = 0x0045
	eclOperandFetchOffset  = 0x002a
	// eclHandlerScanLimit 是往後掃多少 byte 找取運算元的序言。
	// 上界另外用「下一支常式的起點」夾住——少了那一道，掃描會越過 `retf`
	// 讀到下一支常式的序言，量出來的數字看起來很合理但是錯的。
	eclHandlerScanLimit = 256
)

// ECLOpcode 是一條 opcode 的派發資訊。
type ECLOpcode struct {
	// Opcode 是指令碼。
	Opcode byte
	// Handler 是它在 overlay-03 碼段裡的位移。
	Handler uint16
	// Operands 是常式開頭一次取幾個運算元；0 表示沒有那段序言
	// （不吃運算元，或是長度要讀過運算元才知道的那幾條）。
	Operands int
}

// ParseECLOpcodeTable 從 overlay-03 的碼段解出整條派發鏈。
func ParseECLOpcodeTable(overlay tpov.Overlay) ([]ECLOpcode, error) {
	code := overlay.Code
	handlers := make(map[byte]uint16)
	for offset := 0; offset+14 <= len(code); {
		if code[offset] != 0x3d || code[offset+2] != 0x00 {
			offset++
			continue
		}
		// cmp ax, op / jne / push cs / call rel16
		if code[offset+3] == 0x75 && code[offset+5] == 0x0e && code[offset+6] == 0xe8 {
			target := int(offset) + 9 + int(int16(binary.LittleEndian.Uint16(code[offset+7:])))
			if target < 0 || target >= len(code) {
				return nil, fmt.Errorf("opcode 0x%02X dispatches to %d, outside the overlay", code[offset+1], target)
			}
			if _, seen := handlers[code[offset+1]]; !seen {
				handlers[code[offset+1]] = uint16(target)
			}
			offset += 9
			continue
		}
		// cmp ax, op / je / cmp ax, other / jne / push cs / call rel16
		if code[offset+3] == 0x74 && code[offset+5] == 0x3d && code[offset+7] == 0x00 &&
			code[offset+8] == 0x75 && code[offset+10] == 0x0e && code[offset+11] == 0xe8 {
			target := int(offset) + 14 + int(int16(binary.LittleEndian.Uint16(code[offset+12:])))
			if target < 0 || target >= len(code) {
				return nil, fmt.Errorf("opcode 0x%02X dispatches to %d, outside the overlay", code[offset+1], target)
			}
			for _, opcode := range []byte{code[offset+1], code[offset+6]} {
				if _, seen := handlers[opcode]; !seen {
					handlers[opcode] = uint16(target)
				}
			}
			offset += 14
			continue
		}
		offset++
	}
	if len(handlers) == 0 {
		return nil, fmt.Errorf("overlay %d has no ECL dispatch chain", ECLDispatchOverlay)
	}

	starts := make([]int, 0, len(handlers))
	for _, handler := range handlers {
		starts = append(starts, int(handler))
	}
	sort.Ints(starts)
	next := func(from int) int {
		for _, start := range starts {
			if start > from {
				return start
			}
		}
		return len(code)
	}

	table := make([]ECLOpcode, 0, len(handlers))
	for opcode, handler := range handlers {
		table = append(table, ECLOpcode{Opcode: opcode, Handler: handler,
			Operands: eclHandlerOperands(code, int(handler), next(int(handler)))})
	}
	sort.Slice(table, func(i, j int) bool { return table[i].Opcode < table[j].Opcode })
	return table, nil
}

// eclHandlerOperands 找 `mov al, N / push ax / lcall 0045h:002Ah`。
func eclHandlerOperands(code []byte, start, end int) int {
	if end > start+eclHandlerScanLimit {
		end = start + eclHandlerScanLimit
	}
	if end > len(code) {
		end = len(code)
	}
	for offset := start; offset+8 <= end; offset++ {
		if code[offset] != 0xb0 || code[offset+2] != 0x50 || code[offset+3] != 0x9a {
			continue
		}
		if binary.LittleEndian.Uint16(code[offset+4:]) != eclOperandFetchOffset {
			continue
		}
		if binary.LittleEndian.Uint16(code[offset+6:]) != eclOperandFetchSegment {
			continue
		}
		return int(code[offset+1])
	}
	return 0
}

// ReadDOSOverlayCode 取出一個 overlay 的碼段位元組。
func ReadDOSOverlayCode(zipPath string, index int) ([]byte, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	overlayFile, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		return nil, err
	}
	overlays, err := tpov.Decode(executable, overlayFile)
	if err != nil {
		return nil, fmt.Errorf("decode GAME.OVR: %w", err)
	}
	if index < 0 || index >= len(overlays) {
		return nil, fmt.Errorf("GAME.OVR has %d overlays, index %d is outside", len(overlays), index)
	}
	return overlays[index].Code, nil
}

// ReadDOSECLOpcodeTable 從原版 ZIP 解出 opcode 表。
func ReadDOSECLOpcodeTable(zipPath string) ([]ECLOpcode, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	overlayFile, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		return nil, err
	}
	overlays, err := tpov.Decode(executable, overlayFile)
	if err != nil {
		return nil, fmt.Errorf("decode GAME.OVR: %w", err)
	}
	if len(overlays) <= ECLDispatchOverlay {
		return nil, fmt.Errorf("GAME.OVR has %d overlays, want more than %d", len(overlays), ECLDispatchOverlay)
	}
	return ParseECLOpcodeTable(overlays[ECLDispatchOverlay])
}
