// Package etenfont adapts ETen's original 16x15 Big5 bitmap font to font.Face.
//
// 這份實作與 Curse of the Azure Bonds remake 的同名套件同源。兩個 remake 共用
// 一顆 engine，這種與作品無關的字型轉接理應收進那顆 engine；在使用者決定要不要
// 動共用 repo 之前，先各自持有一份，不要為了省一份複製而擅自改共用套件。
//
// 字型檔本身是第三方資產，不進 repo：執行時由路徑載入，沒有就退回內建字型。
package etenfont

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/encoding/traditionalchinese"
)

const (
	glyphWidth      = 16
	glyphHeight     = 15
	glyphBytes      = 30
	asciiGlyphWidth = 8
	asciiGlyphBytes = 15
)

type cachedGlyph struct {
	mask    *image.Alpha
	advance int
}

// Face reads ETen's original STDFONT.15 Chinese glyphs, optional SPCFONT.15
// full-width symbols, and its matching ASCFONT.15 half-width glyphs.
// Unsupported runes still delegate to Fallback.
type Face struct {
	standard []byte
	symbols  []byte
	ascii    []byte
	Fallback font.Face
	Bold     bool
	// Shadow 為真時畫的是字身外面加厚的那一圈，不是字身本身。
	Shadow bool
	mu     sync.Mutex
	cache  map[rune]cachedGlyph
}

// ShadowFace 回傳同一份字模資料的「外圈」面。呼叫端先用同色系暗一階把它畫
// 出來、再用主色畫字身，字就厚了一格而字內的縫隙還在。
func (f *Face) ShadowFace() *Face {
	if f == nil {
		return nil
	}
	return &Face{standard: f.standard, symbols: f.symbols, ascii: f.ascii,
		Fallback: f.Fallback, Bold: f.Bold, Shadow: true,
		cache: make(map[rune]cachedGlyph)}
}

// Load opens an ETen STDFONT.15 and optional SPCFONT.15.
func Load(standardPath, symbolPath string, fallback font.Face, bold bool) (*Face, error) {
	return LoadWithASCII(standardPath, symbolPath, "", fallback, bold)
}

// LoadWithASCII opens an ETen STDFONT.15, optional SPCFONT.15, and optional
// ASCFONT.15. ASCFONT.15 is the companion 8x15 raster used by the original
// ETen family for ASCII and ordinary punctuation. Keeping it separate lets a
// deployment without that local file retain the previous fallback behaviour.
func LoadWithASCII(standardPath, symbolPath, asciiPath string, fallback font.Face, bold bool) (*Face, error) {
	standard, err := os.ReadFile(standardPath)
	if err != nil {
		return nil, fmt.Errorf("read ETen standard font: %w", err)
	}
	if len(standard)%glyphBytes != 0 {
		return nil, fmt.Errorf("ETen standard font size %d is not divisible by %d", len(standard), glyphBytes)
	}
	var symbols []byte
	if symbolPath != "" {
		symbols, err = os.ReadFile(symbolPath)
		if err != nil {
			return nil, fmt.Errorf("read ETen symbol font: %w", err)
		}
		if len(symbols)%glyphBytes != 0 {
			return nil, fmt.Errorf("ETen symbol font size %d is not divisible by %d", len(symbols), glyphBytes)
		}
	}
	var ascii []byte
	if asciiPath != "" {
		ascii, err = os.ReadFile(asciiPath)
		if err != nil {
			return nil, fmt.Errorf("read ETen ASCII font: %w", err)
		}
		if len(ascii) < 256*asciiGlyphBytes || len(ascii)%asciiGlyphBytes != 0 {
			return nil, fmt.Errorf("ETen ASCII font size %d does not contain 256 8x15 glyphs", len(ascii))
		}
	}
	return &Face{standard: standard, symbols: symbols, ascii: ascii, Fallback: fallback, Bold: bold, cache: make(map[rune]cachedGlyph)}, nil
}

func (f *Face) Close() error { return nil }

func (f *Face) Metrics() font.Metrics {
	return font.Metrics{Height: fixed.I(glyphHeight), Ascent: fixed.I(14), Descent: fixed.I(1)}
}

func (f *Face) Kern(r0, r1 rune) fixed.Int26_6 { return 0 }

func (f *Face) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	if glyph, ok := f.glyph(r); ok {
		return fixed.I(glyph.advance), true
	}
	return f.Fallback.GlyphAdvance(r)
}

func (f *Face) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	if glyph, ok := f.glyph(r); ok {
		return fixed.R(0, -14, glyph.advance, 1), fixed.I(glyph.advance), true
	}
	return f.Fallback.GlyphBounds(r)
}

func (f *Face) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	glyph, ok := f.glyph(r)
	if !ok {
		return f.Fallback.Glyph(dot, r)
	}
	x, y := dot.X.Floor(), dot.Y.Floor()-14
	return image.Rect(x, y, x+glyph.advance, y+glyphHeight), glyph.mask, image.Point{}, fixed.I(glyph.advance), true
}

// bitmap is retained for the focused raster tests. Production callers should
// use font.Face methods, which also preserve each glyph's native advance.
// Bitmap 回傳一個字的字模；取不到代表這套字型沒有這個字。
func (f *Face) Bitmap(r rune) (*image.Alpha, bool) { return f.bitmap(r) }

func (f *Face) bitmap(r rune) (*image.Alpha, bool) {
	glyph, ok := f.glyph(r)
	if !ok {
		return nil, false
	}
	return glyph.mask, true
}

func (f *Face) glyph(r rune) (cachedGlyph, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if cached, ok := f.cache[r]; ok {
		return cached, true
	}
	if raw, ok := f.standardGlyph(r); ok {
		glyph := cachedGlyph{mask: rasterGlyph(raw, glyphWidth, glyphBytes/glyphHeight, f.Bold, f.Shadow), advance: glyphWidth}
		f.cache[r] = glyph
		return glyph, true
	}
	if raw, ok := f.asciiGlyph(r); ok {
		glyph := cachedGlyph{mask: rasterGlyph(raw, asciiGlyphWidth, 1, f.Bold, f.Shadow), advance: asciiGlyphWidth}
		f.cache[r] = glyph
		return glyph, true
	}
	return cachedGlyph{}, false
}

// rasterGlyph 把一個字模畫成 alpha 遮罩。加粗是「把每個亮點往右也點一格」。
//
// **最後一欄不加粗。** 字身佔 `width-1` 欄（全形 16 欄的字模，Big5 的字身在
// 0..14），最後那一欄是字與字之間的留白；讓加粗擴進去的話字模就填滿整個
// advance，相鄰兩個字會黏成一團——筆畫密的字（「鈕繼」那種）在畫面上看起來
// 像疊字，而同一個字型不加粗畫出來是清楚的。
// rasterGlyph 把一個字模畫成 alpha 遮罩。
//
// **字身本身不加粗。** 倚天的字模是照「筆畫一格、空隙一格」設計的，把亮點
// 往右膨脹會同時吃掉字與字之間的留白**和字內的空隙**——「鈕」的金字旁、
// 「繼」的絲字旁那些一格寬的縫全部糊成一塊。
//
// 要的厚度改從顏色拿：`shadow` 為真時畫的是「膨脹出來的那一圈」（膨脹減去
// 字身），呼叫端先用同色系暗一階畫它、再用主色畫字身。亮暗有別，所以字內的
// 縫隙還讀得出來，字與字之間也還分得開。
func rasterGlyph(raw []byte, width, bytesPerRow int, bold, shadow bool) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, width, glyphHeight))
	for y := 0; y < glyphHeight; y++ {
		for x := 0; x < width; x++ {
			at := func(column int) bool {
				if column < 0 || column >= width {
					return false
				}
				return raw[y*bytesPerRow+column/8]&(0x80>>uint(column&7)) != 0
			}
			on := at(x)
			if shadow {
				// 膨脹一格再減掉字身，剩下的就是外圈。
				on = (at(x) || at(x-1)) && !at(x)
			} else if bold && x > 0 && x < width-1 {
				on = on || at(x-1)
			}
			if on {
				mask.SetAlpha(x, y, color.Alpha{A: 0xff})
			}
		}
	}
	return mask
}

// big5Variant maps variant Han forms the ETen font has no glyph for onto the
// standard form it does ship. Most of them are outside Big5 altogether; 裏 has
// only a compatibility code point (F9D8), past where the font's glyph table ends. The Softworld manual was typeset
// in Big5, so these variants come from the transcription rather than the page;
// the transcript keeps them, and this table is what lets the font draw them.
// Without it each one renders as an empty box, which reads as a broken font
// rather than a character the font never had.
var big5Variant = map[rune]rune{
	'兎': '兔', '册': '冊', '冲': '沖', '刧': '劫', '却': '卻',
	'携': '攜', '敍': '敘', '着': '著', '羣': '群', '衞': '衛',
	'裏': '裡', '踪': '蹤', '靑': '青',
}

// big5Code 由「解碼」建出 Unicode → Big5 的對照，不用編碼器。
//
// Big5 有一批字帶兩個碼位：正規的常用字碼，以及 F9xx–FExx 的重複區。
// x/text 的編碼器對其中幾個字（包、港、偽、撐、煮、冲…）會回重複區那一個，
// 而倚天的 stdfont.15 只排到常用字與次常用字，於是那幾個常見字取不到字模，
// 畫面上是一個空白方塊——看起來像字型壞了，實際上是查錯碼位。
//
// 逐一解碼再反向建表，就會拿到每個字最低的那個碼位，也就是字型排的那一個。
var (
	big5Once sync.Once
	big5Code map[rune]int
)

func big5Index(r rune) (int, bool) {
	big5Once.Do(func() {
		big5Code = make(map[rune]int, 14000)
		decoder := traditionalchinese.Big5.NewDecoder()
		scan := func(firstLead, lastLead int) {
			for high := firstLead; high <= lastLead; high++ {
				for low := 0x40; low <= 0xfe; low++ {
					if low > 0x7e && low < 0xa1 {
						continue
					}
					decoded, err := decoder.Bytes([]byte{byte(high), byte(low)})
					if err != nil {
						continue
					}
					runes := []rune(string(decoded))
					if len(runes) != 1 || runes[0] == '\uFFFD' {
						continue
					}
					if _, seen := big5Code[runes[0]]; seen {
						continue
					}
					big5Code[runes[0]] = rawBig5(high, low)
				}
			}
		}
		// 漢字區先掃，符號區後掃。Big5 在符號區重複收了幾個字（十 A2CC 與
		// A451、卅 A2CE 與 A4CA…），先掃符號區會讓「十」指到符號字型那一格，
		// 而符號字型多半沒有載入——於是最常見的字反而畫不出來。
		scan(0xa4, 0xf9)
		scan(0xa1, 0xa3)
	})
	raw, ok := big5Code[r]
	return raw, ok
}

// unavailable 是這套字型畫不出字模的排版符號。倚天的符號字型（SPCFONT.15）
// 不在字型目錄裡，破折號與刪節號因此沒有字模；留著會是一串空白方塊，
// 看起來像缺字而不是「這套字型沒有這個符號」。
// 箭頭與數學符號同樣不在倚天的字模裡（`→` 在瞄準列上、`≤` 與 `−` 在
// 開發工具的輸出裡），沒有替換就是一個空白方塊。
var unavailable = strings.NewReplacer("…", "...", "—", "--", "－", "-", "～", "~", "〜", "~",
	"→", "->", "←", "<-", "−", "-", "≤", "<=", "≥", ">=", "※", "*")

// ReplaceUnavailable 把畫不出來的排版符號換成畫得出來的形狀。
// 顯示端與字型覆蓋率稽核都走這一個，兩邊才不會對不上。
func ReplaceUnavailable(value string) string { return unavailable.Replace(value) }

func (f *Face) standardGlyph(r rune) ([]byte, bool) {
	if standard, ok := big5Variant[r]; ok {
		r = standard
	}
	raw, ok := big5Index(r)
	if !ok {
		return nil, false
	}
	lastSymbol := rawBig5(0xa3, 0xbf)
	if raw <= lastSymbol {
		return glyphAt(f.symbols, raw)
	}
	const commonCount = 5401
	var index int
	if raw <= rawBig5(0xc6, 0x7e) {
		index = raw - rawBig5(0xa4, 0x40)
	} else {
		index = commonCount + raw - rawBig5(0xc9, 0x40)
	}
	return glyphAt(f.standard, index)
}

func (f *Face) asciiGlyph(r rune) ([]byte, bool) {
	if len(f.ascii) == 0 {
		return nil, false
	}
	value, ok := etenASCIICode(r)
	if !ok {
		return nil, false
	}
	offset := int(value) * asciiGlyphBytes
	if offset+asciiGlyphBytes > len(f.ascii) {
		return nil, false
	}
	return f.ascii[offset : offset+asciiGlyphBytes], true
}

// ETen's ASCFONT has the half-width forms used by the game UI. Map only
// typographic aliases; translation text remains data-driven and no game term
// is encoded here.
func etenASCIICode(r rune) (byte, bool) {
	// 只有 ASCII 那半段對得上。ETen 的 ASCFONT 高半部不是 Latin-1——
	// 0x80..0xFF 在 Big5 是前導位元組，那些格子畫出來是方塊或雜訊，
	// 而「畫出方塊」在畫面上與缺字沒有分別。所以高半部一律走別名表。
	if r >= 0 && r <= 0x7f {
		return byte(r), true
	}
	aliases := map[rune]byte{
		'　': ' ', '，': ',', '。': '.', '、': ',', '：': ':', '；': ';',
		'！': '!', '？': '?', '（': '(', '）': ')', '［': '[', '］': ']',
		'｛': '{', '｝': '}', '／': '/', '－': '-', '＋': '+', '＝': '=',
		'％': '%', '＆': '&', '＊': '*', '＜': '<', '＞': '>', '｜': '|',
		'「': '"', '」': '"', '『': '"', '』': '"', '〈': '<', '〉': '>',
		'《': '<', '》': '>', '【': '[', '】': ']', '〔': '[', '〕': ']',
		'～': '~', '〜': '~',
		'\u2018': '\'', '\u2019': '\'', '\u201c': '"', '\u201d': '"',
		'×': 'x', '÷': '/', '·': '.', '°': 'o', '±': '+',
	}
	value, ok := aliases[r]
	return value, ok
}

func rawBig5(high, low int) int {
	trail := low - 0x40
	if low >= 0x7f {
		trail = low - 0x62
	}
	return (high-0xa1)*157 + trail
}

func glyphAt(data []byte, index int) ([]byte, bool) {
	offset := index * glyphBytes
	if index < 0 || offset+glyphBytes > len(data) {
		return nil, false
	}
	return data[offset : offset+glyphBytes], true
}
