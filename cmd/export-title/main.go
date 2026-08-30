package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	out := flag.String("out", "workplace/title-export", "output directory")
	flag.Parse()
	pictures, err := assets.ReadTitlePictures(*zipPath)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	atlas := image.NewRGBA(image.Rect(0, 0, 640, 200))
	for index, id := range []uint8{1, 2} {
		rendered, err := pictures[id].RGBA(0, graphics.EGA16)
		if err != nil {
			fail(err)
		}
		path := filepath.Join(*out, fmt.Sprintf("title-block-%02x.png", id))
		if err := writePNG(path, rendered); err != nil {
			fail(err)
		}
		draw.Draw(atlas, image.Rect(index*320, 0, (index+1)*320, 200), rendered, image.Point{}, draw.Src)
	}
	if err := writePNG(filepath.Join(*out, "title-atlas.png"), atlas); err != nil {
		fail(err)
	}
}

func writePNG(path string, source image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, source); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
