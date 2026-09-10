// export-title 把原版的標題畫面從 DAX 解出來寫成 PNG，給對拍與說明文件用。
//
// 版面與來源區塊見 spec 001。
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
	atlas, rendered, err := composeTitleAtlas(pictures)
	if err != nil {
		fail(err)
	}
	for index, id := range titleBlocks {
		path := filepath.Join(*out, fmt.Sprintf("title-block-%02x.png", id))
		if err := writePNG(path, rendered[index]); err != nil {
			fail(err)
		}
	}
	if err := writePNG(filepath.Join(*out, "title-atlas.png"), atlas); err != nil {
		fail(err)
	}
}

// 標題畫面是 `TITLE.DAX` 的**兩個區塊左右並排**，各 320×200（spec 001）。
// 順序有意義：`1` 在左、`2` 在右，反過來畫面就是左右顛倒的。
var titleBlocks = []uint8{1, 2}

const (
	titleBlockWidth  = 320
	titleBlockHeight = 200
)

// composeTitleAtlas 把兩個區塊拼成一整張，同時回傳各自那一張。
func composeTitleAtlas(pictures map[uint8]graphics.Picture) (*image.RGBA, []*image.RGBA, error) {
	atlas := image.NewRGBA(image.Rect(0, 0, titleBlockWidth*len(titleBlocks), titleBlockHeight))
	rendered := make([]*image.RGBA, 0, len(titleBlocks))
	for index, id := range titleBlocks {
		picture, ok := pictures[id]
		if !ok {
			return nil, nil, fmt.Errorf("TITLE.DAX 沒有區塊 %d", id)
		}
		block, err := picture.RGBA(0, graphics.EGA16)
		if err != nil {
			return nil, nil, fmt.Errorf("區塊 %d：%w", id, err)
		}
		rendered = append(rendered, block)
		draw.Draw(atlas, image.Rect(index*titleBlockWidth, 0,
			(index+1)*titleBlockWidth, titleBlockHeight), block, image.Point{}, draw.Src)
	}
	return atlas, rendered, nil
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
