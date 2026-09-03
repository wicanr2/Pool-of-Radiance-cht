package gamepack

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/tpov"
)

// 結局過場在 overlay-18 entry 1（`02A1h`），由 `38h PROGRAM` 運算元 8 進來
// （spec 081）。唯一的呼叫點是 `ECL5/7` 的 `A82Ah`——打贏泰倫斯拉克斯之後。
const (
	EndingOverlay      = 18
	EndingEntryOffset  = 0x02A1
	EndingPictureName  = "FINAL"
	EndingPictureIndex = 5
)

// EndingLine 是過場裡的一行字。Offset 是它在 overlay-18 裡的位置，
// Row 是原版畫在第幾列（`0198h:002Fh` 的第三個引數）。
type EndingLine struct {
	Offset int
	Row    int
	Text   string
}

// EndingScript 是整段過場：先一頁字、等一個鍵，再一串圖，最後兩頁字。
type EndingScript struct {
	// Lines 依原版畫出來的順序。
	Lines []EndingLine
	// PictureBlocks 是 `FINAL5.DAX` 裡依序畫出來的區塊編號
	//（`018Eh:0039h` 載入、`018Eh:004Dh` 畫）。
	PictureBlocks []int
}

// endingLineLayout 是逐條讀出來的畫面順序。位移取自 overlay-18 的
// `mov di, imm16`，列號取自同一次呼叫推給 `0198h:002Fh` 的引數。
var endingLineLayout = []struct{ offset, row int }{
	{0x0111, 0x11}, {0x0135, 0x12}, {0x015A, 0x13},
	{0x0178, 0x11}, {0x019C, 0x12}, {0x01C1, 0x13},
	{0x01E8, 0x14}, {0x0208, 0x15}, {0x022D, 0x16},
	{0x0236, 0x11}, {0x025D, 0x12}, {0x0285, 0x13},
	{0x0298, 0x14},
}

// endingPictureBlocks 是原版依序載入的 `FINAL5.DAX` 區塊。
var endingPictureBlocks = []int{1, 3, 4, 5, 6}

// ReadDOSEndingScript 從原版 ZIP 取出結局過場的文字與圖序。
func ReadDOSEndingScript(zipPath string) (EndingScript, error) {
	executable, err := readStartExecutable(zipPath)
	if err != nil {
		return EndingScript{}, err
	}
	overlayFile, err := readArchiveMember(zipPath, "GAME.OVR")
	if err != nil {
		return EndingScript{}, err
	}
	overlays, err := tpov.Decode(executable, overlayFile)
	if err != nil {
		return EndingScript{}, fmt.Errorf("decode GAME.OVR: %w", err)
	}
	if len(overlays) <= EndingOverlay {
		return EndingScript{}, fmt.Errorf("GAME.OVR has %d overlays, want more than %d",
			len(overlays), EndingOverlay)
	}
	code := overlays[EndingOverlay].Code
	script := EndingScript{
		Lines:         make([]EndingLine, 0, len(endingLineLayout)),
		PictureBlocks: append([]int(nil), endingPictureBlocks...),
	}
	for _, layout := range endingLineLayout {
		text, ok := pascalString(code, layout.offset)
		if !ok {
			return EndingScript{}, fmt.Errorf(
				"Pool ending line at %04Xh is not a Pascal short string", layout.offset)
		}
		script.Lines = append(script.Lines, EndingLine{
			Offset: layout.offset, Row: layout.row, Text: text,
		})
	}
	return script, nil
}

// Pages 把台詞切成原版的頁。原版每開一次文字框就從第 `11h` 列重新印起
//（`0198h:0066h` 開框、`0198h:002Fh` 逐行），所以**列號回到第一列就是新的
// 一頁**——分頁是資料算出來的，不是抄行數。
func (script EndingScript) Pages() [][]EndingLine {
	if len(script.Lines) == 0 {
		return nil
	}
	first := script.Lines[0].Row
	var pages [][]EndingLine
	var current []EndingLine
	for _, line := range script.Lines {
		if line.Row == first && len(current) != 0 {
			pages = append(pages, current)
			current = nil
		}
		current = append(current, line)
	}
	return append(pages, current)
}
