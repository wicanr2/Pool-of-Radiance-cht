// pool-disp-scan 找 overlay 與 START.EXE 裡對某個位址／位移的記憶體存取。
//
// 同一個數值在原版有兩種完全不同的身分，掃法也不一樣（見 AGENTS.md §5）：
//
//   - ECL 變數：引擎側是「基底暫存器 + 結構位移」，modrm 的 mod=10。
//     位元組裡不會出現 ECL 位址的字面值，要先把位址反解成位移才掃得到。
//   - DS 段全域：modrm 的 mod=00、rm=110，也就是 [disp16] 絕對定址，
//     位元組裡就是位址本身。
//
// 兩種一起掃才分得出來——數值範圍分不出來，`6CD2h` 落在 ECL class 1 的窗內
// 卻是 DS 全域（spec 122），`6DD5h` 落在同一個窗內卻是 ECL 變數（spec 100）。
//
// 用法：
//
//	pool-disp-scan -addresses 6CD2,6CD3,6DD5
//	pool-disp-scan -addresses 5AA -ecl-class 1   # 先把 ECL 位址反解成位移
//
// **一定要多帶一個已知答案當正對照。** `5AAh` 必須掃出 overlay-14 那四條
// `mov word [di+5AA], 1`（spec 125）；掃不出來就是工具或輸入壞了，這時候
// 任何「零筆」都證明不了事情。
package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// overlay 在 GAME.OVR 裡依 index 依序排列，但檔頭佔掉 15h 個位元組。
// 這個值是對出來的：spec 125 已知的四條 [4937h]+5AAh = 1 要落回
// overlay-14 0700h/0716h/072Ch/0742h，spec 122 的三道閘門要落在 0DB7h 之後。
const overlayHeaderBytes = 0x15

type manifest struct {
	Overlays []struct {
		Index           int `json:"index"`
		CodeBytes       int `json:"code_bytes"`
		RelocationBytes int `json:"relocation_bytes"`
	} `json:"overlays"`
}

// locate 把 GAME.OVR 的檔案偏移換成「第幾顆 overlay 的第幾個位元組」。
//
// 不要用 manifest 的 executable_file_offset——那是 START.EXE 裡的 stub 位置，
// 不是 GAME.OVR 的偏移。
func (m manifest) locate(offset int) string {
	position := overlayHeaderBytes
	for _, overlay := range m.Overlays {
		end := position + overlay.CodeBytes
		if offset >= position && offset < end {
			return fmt.Sprintf("overlay-%02d %04Xh", overlay.Index, offset-position)
		}
		position = end + overlay.RelocationBytes
	}
	return fmt.Sprintf("GAME.OVR+%05Xh", offset)
}

func member(reader *zip.ReadCloser, name string) ([]byte, error) {
	for _, file := range reader.File {
		if !strings.EqualFold(file.Name, name) {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer stream.Close()
		return io.ReadAll(stream)
	}
	return nil, fmt.Errorf("封存檔裡沒有 %s", name)
}

type hit struct {
	where string
	kind  string
	bytes string
}

func scan(data []byte, value int, where func(int) string) []hit {
	low, high := byte(value&0xFF), byte((value>>8)&0xFF)
	var hits []hit
	for i := 2; i+1 < len(data); i++ {
		if data[i] != low || data[i+1] != high {
			continue
		}
		// 前一個位元組不一定是 modrm——有三個 opcode 家族自己就吃兩位元組的
		// 立即值／位址，而且值域和 modrm 的 mod=10（80h..BFh）重疊。
		// 不先把它們挑掉，A1h（mov ax,[disp16]）會被讀成 modrm，
		// BFh（mov di,imm16）也會，兩種都會產生假的「[基底+disp16]」。
		lead := data[i-1]
		var kind string
		switch {
		case lead >= 0xA0 && lead <= 0xA3:
			kind = "[disp16] 絕對" // moffs：mov ax/al ←→ [disp16]
		case lead >= 0xB8 && lead <= 0xBF:
			kind = "imm16 常數" // mov reg,imm16——位址被當常數傳出去
		case lead>>6 == 2:
			kind = "[基底+disp16]"
		case lead&0xC7 == 0x06:
			kind = "[disp16] 絕對"
		default:
			continue
		}
		start := i - 4
		if start < 0 {
			start = 0
		}
		end := i + 4
		if end > len(data) {
			end = len(data)
		}
		hits = append(hits, hit{where(i - 1), kind, fmt.Sprintf("% X", data[start:end])})
	}
	return hits
}

// eclDisplacement 把 ECL 位址反解成引擎側的結構位移（spec 106／008）。
func eclDisplacement(address, class int) int {
	base := 0x2A00 // class 1，經 [4937h]
	if class == 0 {
		base = 0x6E00 // class 0，經 [4933h]
	}
	return (base + address*2) & 0xFFFF
}

func main() {
	addresses := flag.String("addresses", "", "逗號分隔的十六進位值（位址或位移）")
	eclClass := flag.Int("ecl-class", -1, "先把輸入當 ECL 位址反解成引擎位移（0 或 1）")
	archive := flag.String("zip", "Pool of Radiance (1988).zip", "原版封存檔")
	manifestPath := flag.String("manifest", "docs/audit/dos-ovr-manifest.json", "overlay 清單")
	flag.Parse()

	if strings.TrimSpace(*addresses) == "" {
		fmt.Fprintln(os.Stderr, "要 -addresses")
		os.Exit(2)
	}

	reader, err := zip.OpenReader(*archive)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer reader.Close()

	raw, err := os.ReadFile(*manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var list manifest
	if err := json.Unmarshal(raw, &list); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	overlay, err := member(reader, "poolrad/game.ovr")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	executable, err := member(reader, "poolrad/start.exe")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, field := range strings.Split(*addresses, ",") {
		field = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(field), "0x"), "h")
		if field == "" {
			continue
		}
		parsed, err := strconv.ParseInt(field, 16, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q 不是十六進位值\n", field)
			os.Exit(2)
		}
		value := int(parsed)
		label := fmt.Sprintf("%Xh", value)
		if *eclClass >= 0 {
			value = eclDisplacement(value, *eclClass)
			label = fmt.Sprintf("%s → 引擎位移 %Xh（class %d）", label, value, *eclClass)
		}

		fmt.Printf("=== %s ===\n", label)
		hits := scan(overlay, value, list.locate)
		hits = append(hits, scan(executable, value, func(i int) string {
			return fmt.Sprintf("START.EXE+%05Xh", i)
		})...)
		for _, h := range hits {
			fmt.Printf("  %-22s %-14s bytes %s\n", h.where, h.kind, h.bytes)
		}
		fmt.Printf("  合計 %d 筆\n\n", len(hits))
	}
}
