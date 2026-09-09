package etenfont

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func TestLoadRejectsTruncatedFont(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdfont.15")
	if err := os.WriteFile(path, []byte{1}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, "", basicfont.Face7x13, true); err == nil {
		t.Fatal("Load accepted a truncated font")
	}
}

func TestBoldExpandsGlyphRight(t *testing.T) {
	data := make([]byte, 5402*glyphBytes)
	data[0] = 0x80 // Big5 A440, 「一」: test pixel at x=0.
	path := filepath.Join(t.TempDir(), "stdfont.15")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	face, err := Load(path, "", basicfont.Face7x13, true)
	if err != nil {
		t.Fatal(err)
	}
	mask, ok := face.bitmap('一')
	if !ok || mask.AlphaAt(0, 0).A == 0 || mask.AlphaAt(1, 0).A == 0 {
		t.Fatal("bold face did not expand source pixel one column right")
	}
}

func TestASCIICompanionRendersHalfWidthAndFullWidthPunctuation(t *testing.T) {
	directory := t.TempDir()
	standardPath := filepath.Join(directory, "stdfont.15")
	if err := os.WriteFile(standardPath, make([]byte, 5402*glyphBytes), 0o600); err != nil {
		t.Fatal(err)
	}
	ascii := make([]byte, 256*asciiGlyphBytes)
	ascii[int(':')*asciiGlyphBytes] = 0x80
	asciiPath := filepath.Join(directory, "ascfont.15")
	if err := os.WriteFile(asciiPath, ascii, 0o600); err != nil {
		t.Fatal(err)
	}
	face, err := LoadWithASCII(standardPath, "", asciiPath, basicfont.Face7x13, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []rune{':', '：'} {
		advance, ok := face.GlyphAdvance(r)
		if !ok || advance != fixed.I(asciiGlyphWidth) {
			t.Fatalf("glyph %q advance=%v ok=%v, want %v", r, advance, ok, fixed.I(asciiGlyphWidth))
		}
		mask, ok := face.bitmap(r)
		if !ok || mask.AlphaAt(0, 0).A == 0 {
			t.Fatalf("glyph %q did not use the companion ASCII raster", r)
		}
	}
}

// 轉錄稿裡有 Big5 沒有碼位的異體字（敍、裏、却…）。字型畫不出來會是空白方塊，
// 看起來像字型壞了，而不是「這個字這套字型沒有」。別名表要把它們對到 ETen
// 實際有的標準字形，因此異體字必須取到與標準字形同一格字模。
func TestVariantHanFormsResolveToTheirBig5Standard(t *testing.T) {
	// 每一格填入自己的索引，讓「取到哪一格」看得出來。
	data := make([]byte, 13500*glyphBytes)
	for index := 0; index*glyphBytes < len(data); index++ {
		data[index*glyphBytes] = byte(index)
		data[index*glyphBytes+1] = byte(index >> 8)
	}
	path := filepath.Join(t.TempDir(), "stdfont.15")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	face, err := Load(path, "", basicfont.Face7x13, false)
	if err != nil {
		t.Fatal(err)
	}
	for variant, standard := range big5Variant {
		variantGlyph, okVariant := face.standardGlyph(variant)
		standardGlyph, okStandard := face.standardGlyph(standard)
		if !okStandard {
			t.Fatalf("the standard form %c is outside the font", standard)
		}
		if !okVariant {
			t.Fatalf("the variant %c produced no glyph", variant)
		}
		if !bytes.Equal(variantGlyph, standardGlyph) {
			t.Fatalf("%c did not resolve to %c", variant, standard)
		}
	}
}

// 別名表只收字型畫不出來的字：字型有的字放進來會蓋掉正確的字形。
func TestVariantTableOnlyCoversRunesBig5Lacks(t *testing.T) {
	// 倚天字型排到次常用字結尾（F9D5）；F9D6 之後是重複區，字型沒有排。
	lastGlyph := rawBig5(0xf9, 0xd5)
	for variant, standard := range big5Variant {
		if raw, ok := big5Index(variant); ok && raw <= lastGlyph {
			t.Fatalf("%c is inside the font's range and must not be aliased", variant)
		}
		raw, ok := big5Index(standard)
		if !ok || raw > lastGlyph {
			t.Fatalf("the replacement %c is not inside the font's range either", standard)
		}
	}
}

// 有兩個碼位的字要取常用字區那一個，不是 F9xx–FExx 的重複區。
// x/text 的編碼器對這幾個字會回重複區，而倚天字型沒有排到那裡。
func TestDoublyEncodedRunesUseTheCommonBig5Code(t *testing.T) {
	for r, want := range map[rune][2]int{
		'包': {0xa5, 0x5d}, '港': {0xb4, 0xe4}, '偽': {0xb0, 0xb0},
		'撐': {0xbc, 0xb5}, '煮': {0xb5, 0x4e},
	} {
		raw, ok := big5Index(r)
		if !ok {
			t.Fatalf("%c has no Big5 code point", r)
		}
		if expected := rawBig5(want[0], want[1]); raw != expected {
			t.Fatalf("%c resolved to raw %d, want %d (%02X%02X)", r, raw, expected, want[0], want[1])
		}
	}
}

// Big5 在符號區重複收了幾個常用字。字型的符號檔多半沒有載入，所以這些字
// 必須取漢字區那個碼位，否則「十」這種字會整批畫不出來。
func TestRunesDuplicatedInTheSymbolAreaUseTheHanCode(t *testing.T) {
	for r, want := range map[rune][2]int{'十': {0xa4, 0x51}, '卅': {0xa4, 0xca}} {
		raw, ok := big5Index(r)
		if !ok {
			t.Fatalf("%c has no Big5 code point", r)
		}
		if expected := rawBig5(want[0], want[1]); raw != expected {
			t.Fatalf("%c resolved to raw %d, want the Han-area %d", r, raw, expected)
		}
	}
}

// 加粗不能吃掉字與字之間的留白：字模的最後一欄是那道留白，讓加粗擴進去的話
// 相鄰兩個字會黏成一團。
func TestBoldKeepsTheTrailingColumnClear(t *testing.T) {
	// 一列裡把字身欄（0..14）全部點亮，最後一欄（15）留空。
	raw := make([]byte, 30)
	for y := 0; y < 15; y++ {
		raw[y*2] = 0xFF   // x 0..7
		raw[y*2+1] = 0xFE // x 8..14，第 15 欄留空
	}
	for _, bold := range []bool{false, true} {
		mask := rasterGlyph(raw, 16, 2, bold, false)
		if got := mask.AlphaAt(15, 7).A; got != 0 {
			t.Errorf("bold=%v 時最後一欄被點亮了（%d），那是字與字之間的留白", bold, got)
		}
		if got := mask.AlphaAt(14, 7).A; got == 0 {
			t.Errorf("bold=%v 時字身最後一欄不見了", bold)
		}
	}
}

// **外圈與字身不重疊**：外圈是「膨脹一格減掉字身」，所以字身亮的地方外圈
// 一定是暗的。兩者疊在一起才會是「字身用主色、外面一圈用暗色」。
func TestShadowIsTheRingOutsideTheBody(t *testing.T) {
	// 第 7 列點亮 x=4..6，其餘全空。
	raw := make([]byte, 30)
	raw[7*2] = 0x0E // 0000 1110 → x 4,5,6
	body := rasterGlyph(raw, 16, 2, false, false)
	ring := rasterGlyph(raw, 16, 2, false, true)
	for x := 0; x < 16; x++ {
		bodyOn := body.AlphaAt(x, 7).A != 0
		ringOn := ring.AlphaAt(x, 7).A != 0
		if bodyOn && ringOn {
			t.Errorf("x=%d 同時屬於字身與外圈", x)
		}
	}
	// 外圈就在字身右邊那一格。
	if ring.AlphaAt(7, 7).A == 0 {
		t.Error("字身右邊那一格不在外圈裡")
	}
	if ring.AlphaAt(3, 7).A != 0 {
		t.Error("字身左邊那一格不該在外圈裡——加厚只往右")
	}
}
