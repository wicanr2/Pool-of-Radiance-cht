package assets

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// ReadEndingPictures 取出 `FINAL5.DAX` 裡結局過場用得到的區塊（spec 108）。
// 回傳的 key 是原版的區塊編號。
//
// 那個檔裡還有別的區塊解不成圖片（`0`、`2`、`7`、`8`），這裡只取要用的，
// 解不開的不當錯誤——它們不是這條路徑的東西。
func ReadEndingPictures(zipPath string, blocks []uint8) (map[uint8]graphics.Picture, error) {
	data, err := readArchiveMember(zipPath, "FINAL5.DAX")
	if err != nil {
		return nil, err
	}
	parsed, err := dax.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse FINAL5.DAX: %w", err)
	}
	wanted := make(map[uint8]bool, len(blocks))
	for _, id := range blocks {
		wanted[id] = true
	}
	result := make(map[uint8]graphics.Picture, len(blocks))
	for _, block := range parsed {
		if !wanted[block.Entry.ID] {
			continue
		}
		// 遮罩色 0：三張小圖疊在場景上，背景那一格要透明才看得到底下。
		picture, err := graphics.ParsePicture(block.Data, true, 0)
		if err != nil {
			return nil, fmt.Errorf("FINAL5.DAX block %d: %w", block.Entry.ID, err)
		}
		result[block.Entry.ID] = picture
	}
	for _, id := range blocks {
		if _, ok := result[id]; !ok {
			return nil, fmt.Errorf("FINAL5.DAX has no block %d", id)
		}
	}
	return result, nil
}

// readArchiveMember 依檔名（不分大小寫）取出 ZIP 裡的一個成員。
func readArchiveMember(zipPath, name string) ([]byte, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer archive.Close()
	var member *zip.File
	for _, candidate := range archive.File {
		if strings.EqualFold(filepath.Base(candidate.Name), name) {
			member = candidate
			break
		}
	}
	if member == nil {
		return nil, fmt.Errorf("DOS ZIP has no %s", name)
	}
	stream, err := member.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, 1<<20))
	closeErr := stream.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read %s: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close %s: %w", name, closeErr)
	}
	if uint64(len(data)) != member.UncompressedSize64 {
		return nil, fmt.Errorf("%s exceeds 1 MiB bound", name)
	}
	return data, nil
}

// ComposeEndingScene 依 spec 108 的層序把結局畫面疊出來。
// pictures 要含得下 layers 用到的每一個區塊。
func ComposeEndingScene(pictures map[uint8]graphics.Picture, layers []gamepack.EndingLayer) (graphics.Picture, error) {
	if len(layers) == 0 {
		return graphics.Picture{}, fmt.Errorf("Pool ending scene has no layers")
	}
	base, ok := pictures[layers[0].Block]
	if !ok {
		return graphics.Picture{}, fmt.Errorf("Pool ending scene is missing block %d", layers[0].Block)
	}
	scene := base
	scene.Pixels = append([]uint8(nil), base.Pixels...)
	for _, layer := range layers[1:] {
		source, ok := pictures[layer.Block]
		if !ok {
			return graphics.Picture{}, fmt.Errorf("Pool ending scene is missing block %d", layer.Block)
		}
		if err := gamepack.CheckEndingLayerFits(layer, source.Width(), source.Height()); err != nil {
			return graphics.Picture{}, err
		}
		merged, err := graphics.MergePicturesAt(scene, source, layer.X, layer.Y)
		if err != nil {
			return graphics.Picture{}, fmt.Errorf("Pool ending block %d: %w", layer.Block, err)
		}
		scene = merged
	}
	return scene, nil
}
