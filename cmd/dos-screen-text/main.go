// Command dos-screen-text reads an unscaled 320x200 DOS text grid captured at
// an integer nearest-neighbour scale. The font signature table is external so
// the command remains game-neutral and does not embed original artwork.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strconv"
	"strings"
)

const (
	columns = 40
	rows    = 25
)

type rgb struct{ r, g, b uint32 }

func fail(format string, values ...any) {
	fmt.Fprintf(os.Stderr, "dos-screen-text: "+format+"\n", values...)
	os.Exit(1)
}

func loadFont(path string) map[uint64]string {
	encoded, err := os.ReadFile(path)
	if err != nil {
		fail("read font: %v", err)
	}
	var source map[string]string
	if err := json.Unmarshal(encoded, &source); err != nil {
		fail("decode font: %v", err)
	}
	result := make(map[uint64]string, len(source))
	for key, value := range source {
		signature, err := strconv.ParseUint(key, 16, 64)
		if err != nil || value == "" {
			fail("invalid font entry %q", key)
		}
		result[signature] = value
	}
	return result
}

func pixel(value image.Image, x, y int) rgb {
	r, g, b, _ := value.At(x, y).RGBA()
	return rgb{r: r, g: g, b: b}
}

func signature(value image.Image, column, row, scale int) uint64 {
	counts := map[rgb]int{}
	values := make([]rgb, 0, 64)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			current := pixel(value, (column*8+x)*scale, (row*8+y)*scale)
			values = append(values, current)
			counts[current]++
		}
	}
	var background rgb
	maximum := -1
	// Match Python Counter.most_common(): ties keep the first-seen color. A map
	// iteration would make glyph recognition nondeterministic for 32/32 cells.
	seen := map[rgb]bool{}
	for _, current := range values {
		if seen[current] {
			continue
		}
		seen[current] = true
		count := counts[current]
		if count > maximum {
			background, maximum = current, count
		}
	}
	var bits uint64
	for index, current := range values {
		if current != background {
			bits |= uint64(1) << index
		}
	}
	return bits
}

func read(path string, font map[uint64]string, scale int) {
	handle, err := os.Open(path)
	if err != nil {
		fail("open %s: %v", path, err)
	}
	defer handle.Close()
	value, _, err := image.Decode(handle)
	if err != nil {
		fail("decode %s: %v", path, err)
	}
	if value.Bounds().Dx() < columns*8*scale || value.Bounds().Dy() < rows*8*scale {
		fail("%s is %dx%d; need at least %dx%d", path, value.Bounds().Dx(), value.Bounds().Dy(), columns*8*scale, rows*8*scale)
	}
	for row := 0; row < rows; row++ {
		var line strings.Builder
		for column := 0; column < columns; column++ {
			bits := signature(value, column, row, scale)
			if bits == 0 {
				line.WriteByte(' ')
			} else if character, ok := font[bits]; ok {
				line.WriteString(character)
			} else {
				line.WriteByte('?')
			}
		}
		text := strings.TrimRight(line.String(), " ")
		if strings.TrimSpace(text) != "" {
			fmt.Printf("%2d| %s\n", row, text)
		}
	}
}

func main() {
	fontPath := flag.String("font", "", "JSON map of 64-bit glyph signatures to characters")
	scale := flag.Int("scale", 2, "integer screenshot scale")
	flag.Parse()
	if *fontPath == "" || *scale < 1 || flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	font := loadFont(*fontPath)
	for _, path := range flag.Args() {
		read(path, font, *scale)
	}
}
