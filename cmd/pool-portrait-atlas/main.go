// Command pool-portrait-atlas 把人物肖像圖鑑要用的 PNG 與收據匯出到磁碟。
//
// 輸入是 spec 006／117 已經驗證過形狀的 16 個 HEAD／BODY archive（109 個
// block）。這支工具只做三件事：
//
//  1. 把 109 個原始 block 逐張存成 PNG（不裁切、不合成）。
//  2. 合成兩組已經有證據的組合——建角可選的 14×12 表（archive 3，見
//     internal/assets/portrait.go）與 Rolf 的 NPC 半身像（archive 3、
//     HEAD 區塊 8、BODY 區塊 9，見 spec 117）——各自再存一張 PNG。
//  3. 寫一份收據 JSON，記輸入檔雜湊、每張 PNG 對回哪個 DAX 的哪個 block／
//     selector，以及數量；沒有一筆是手工貼的。
//
// 除了這兩組已知組合，其餘 block 的「這是誰」一律不在這支工具的職責內——
// 那是 docs/archaeology/portraits.md 要交代的事，而且大多數還查不到，
// 要如實留白。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"archive/zip"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

// rolfBodyBlock 是 Rolf 半身像的 BODY 區塊（spec 117：與 ECL 的
// `SETUP MONSTER 12, 2, 9` 第三個 operand 相同，兩者互相印證）。head 區塊有
// 專用常數 gamepack.RolfPortraitHeadBlock，body 目前沒有——這裡照 spec 117
// 與 internal/assets/npc_portrait_test.go 的用法直接寫 9，不新增常數到
// internal/gamepack（那個檔不在這支工具的修改範圍內）。
const rolfBodyBlock = 9

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	imgDir := flag.String("img-dir", "docs/archaeology/img/portraits", "PNG output directory")
	receiptPath := flag.String("receipt", "docs/audit/portrait-atlas.json", "receipt JSON output path")
	flag.Parse()
	result, err := run(*zipPath, *imgDir, *receiptPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("archives=%d blocks=%d raw_png=%d creation_grid_cells=%d known_npc=%d\n",
		result.Archives, result.Blocks, len(result.RawPortraits),
		len(result.CreationGrid.HeadSelectors)*len(result.CreationGrid.BodySelectors),
		len(result.KnownNPC))
}

type pngFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type rawEntry struct {
	Source     string  `json:"source"`
	Kind       string  `json:"kind"`
	Archive    uint8   `json:"archive"`
	BlockID    uint8   `json:"block_id"`
	BlockIDHex string  `json:"block_id_hex"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	PNG        pngFile `json:"png"`
}

type selectorBlock struct {
	Selector   uint8  `json:"selector"`
	BlockID    uint8  `json:"block_id"`
	BlockIDHex string `json:"block_id_hex"`
}

type creationGridReport struct {
	PNG           pngFile         `json:"png"`
	Archive       uint8           `json:"archive"`
	HeadSelectors []selectorBlock `json:"head_selectors"`
	BodySelectors []selectorBlock `json:"body_selectors"`
	Evidence      string          `json:"evidence"`
	Source        string          `json:"source"`
}

type npcEntry struct {
	Name      string  `json:"name"`
	Archive   uint8   `json:"archive"`
	HeadBlock uint8   `json:"head_block"`
	BodyBlock uint8   `json:"body_block"`
	PNG       pngFile `json:"png"`
	Evidence  string  `json:"evidence"`
	Source    string  `json:"source"`
}

type report struct {
	Tool         string             `json:"tool"`
	ZIP          string             `json:"zip"`
	ZIPSHA256    string             `json:"zip_sha256"`
	Archives     int                `json:"archives"`
	Blocks       int                `json:"blocks"`
	RawPortraits []rawEntry         `json:"raw_portraits"`
	OverviewPNG  pngFile            `json:"overview_png"`
	CreationGrid creationGridReport `json:"creation_grid"`
	KnownNPC     []npcEntry         `json:"known_npc_portraits"`
	Notes        []string           `json:"notes"`
}

// decodedArchive 是一份 HEAD／BODY archive 解出來的全部 block，仍保留原始
// block ID（不是排序後的位置）。
type decodedArchive struct {
	source  string // 例如 "HEAD3.DAX"
	kind    string // "head" 或 "body"
	archive uint8
	blocks  map[uint8]graphics.Picture
}

func run(zipPath, imgDir, receiptPath string) (report, error) {
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		return report{}, err
	}
	zipSHA, err := sha256File(zipPath)
	if err != nil {
		return report{}, err
	}
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer reader.Close()

	archives := make(map[string]decodedArchive)
	for _, kind := range []string{"HEAD", "BODY"} {
		for n := uint8(1); n <= 8; n++ {
			name := fmt.Sprintf("%s%d.DAX", kind, n)
			decoded, err := decodeArchive(reader.File, name, kind, n)
			if err != nil {
				return report{}, err
			}
			archives[name] = decoded
		}
	}

	var rawEntries []rawEntry
	blockTotal := 0
	for _, name := range sortedKeys(archives) {
		archive := archives[name]
		for _, blockID := range sortedBlockIDs(archive.blocks) {
			picture := archive.blocks[blockID]
			pngPath := filepath.Join(imgDir, fmt.Sprintf("%s%d-%02x.png", archive.kind, archive.archive, blockID))
			rendered, err := renderPicture(picture)
			if err != nil {
				return report{}, fmt.Errorf("%s block 0x%02X: %w", name, blockID, err)
			}
			sum, err := writePNGFile(pngPath, rendered)
			if err != nil {
				return report{}, err
			}
			rawEntries = append(rawEntries, rawEntry{
				Source: name, Kind: archive.kind, Archive: archive.archive,
				BlockID: blockID, BlockIDHex: fmt.Sprintf("0x%02X", blockID),
				Width: picture.Width(), Height: picture.Height(),
				PNG: pngFile{Path: filepath.ToSlash(pngPath), SHA256: sum},
			})
			blockTotal++
		}
	}
	if len(archives) != 16 {
		return report{}, fmt.Errorf("portrait archive count %d, want 16", len(archives))
	}
	if blockTotal != 109 {
		return report{}, fmt.Errorf("portrait block count %d, want 109 (spec 006)", blockTotal)
	}

	overviewPath := filepath.Join(imgDir, "overview.png")
	overviewSum, err := writeOverview(overviewPath, archives, rawEntries)
	if err != nil {
		return report{}, err
	}

	creationGrid, err := buildCreationGrid(zipPath, imgDir, archives)
	if err != nil {
		return report{}, err
	}

	rolf, err := buildRolfPortrait(zipPath, imgDir)
	if err != nil {
		return report{}, err
	}

	result := report{
		Tool:         "pool-portrait-atlas",
		ZIP:          filepath.Base(zipPath),
		ZIPSHA256:    zipSHA,
		Archives:     len(archives),
		Blocks:       blockTotal,
		RawPortraits: rawEntries,
		OverviewPNG:  pngFile{Path: filepath.ToSlash(overviewPath), SHA256: overviewSum},
		CreationGrid: creationGrid,
		KnownNPC:     []npcEntry{rolf},
		Notes: []string{
			"raw_portraits 涵蓋 16 個 archive 的全部 109 個 block，逐張未裁切、未合成。",
			"creation_grid 是唯一已驗證的建角組合（spec 006），只用 archive 3 的 HEAD／BODY。",
			"known_npc_portraits 目前只有 Rolf 一筆——spec 117 記載其餘 NPC 的 head 選擇子" +
				"（+5C2h）producer 尚未找到，沒有第二筆能同等確認的組合。",
		},
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return report{}, err
	}
	if err := os.WriteFile(receiptPath, append(data, '\n'), 0o644); err != nil {
		return report{}, err
	}
	return result, nil
}

func decodeArchive(members []*zip.File, name, kind string, archiveNumber uint8) (decodedArchive, error) {
	var member *zip.File
	for _, candidate := range members {
		if strings.EqualFold(filepath.Base(candidate.Name), name) {
			member = candidate
			break
		}
	}
	if member == nil {
		return decodedArchive{}, fmt.Errorf("DOS ZIP has no %s", name)
	}
	stream, err := member.Open()
	if err != nil {
		return decodedArchive{}, fmt.Errorf("open %s: %w", name, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, 8<<20))
	closeErr := stream.Close()
	if readErr != nil {
		return decodedArchive{}, fmt.Errorf("read %s: %w", name, readErr)
	}
	if closeErr != nil {
		return decodedArchive{}, fmt.Errorf("close %s: %w", name, closeErr)
	}
	if uint64(len(data)) != member.UncompressedSize64 {
		return decodedArchive{}, fmt.Errorf("%s exceeds 8 MiB bound", name)
	}
	blocks, err := dax.Parse(data)
	if err != nil {
		return decodedArchive{}, fmt.Errorf("parse %s: %w", name, err)
	}
	decoded := decodedArchive{source: name, kind: strings.ToLower(kind), archive: archiveNumber, blocks: map[uint8]graphics.Picture{}}
	for _, block := range blocks {
		picture, err := graphics.ParsePicture(block.Data, false, 0)
		if err != nil {
			return decodedArchive{}, fmt.Errorf("%s block 0x%02X: %w", name, block.Entry.ID, err)
		}
		if picture.ItemCount != 1 {
			return decodedArchive{}, fmt.Errorf("%s block 0x%02X has %d items, want 1", name, block.Entry.ID, picture.ItemCount)
		}
		if kind == "HEAD" && (picture.Width() != 88 || picture.Height() != 40) {
			return decodedArchive{}, fmt.Errorf("%s block 0x%02X is %dx%d, want 88x40 (spec 006)", name, block.Entry.ID, picture.Width(), picture.Height())
		}
		if kind == "BODY" && (picture.Width() != 88 || picture.Height() != 48) {
			return decodedArchive{}, fmt.Errorf("%s block 0x%02X is %dx%d, want 88x48 (spec 006)", name, block.Entry.ID, picture.Width(), picture.Height())
		}
		if _, dup := decoded.blocks[block.Entry.ID]; dup {
			return decodedArchive{}, fmt.Errorf("%s has duplicate block 0x%02X", name, block.Entry.ID)
		}
		decoded.blocks[block.Entry.ID] = picture
	}
	return decoded, nil
}

func renderPicture(picture graphics.Picture) (*image.RGBA, error) {
	return picture.RGBA(0, graphics.EGA16)
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func writePNGFile(path string, source image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, source); err != nil {
		return "", fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return fmt.Sprintf("%x", sum), nil
}

func sortedKeys(archives map[string]decodedArchive) []string {
	keys := make([]string, 0, len(archives))
	for key := range archives {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedBlockIDs(blocks map[uint8]graphics.Picture) []uint8 {
	ids := make([]uint8, 0, len(blocks))
	for id := range blocks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// --- 總覽接觸表 ---

type sheetCell struct {
	image *image.RGBA
	label string
}

const (
	sheetPad         = 6
	sheetLabelHeight = 14
	sheetColumns     = 16
)

func writeOverview(path string, archives map[string]decodedArchive, entries []rawEntry) (string, error) {
	_ = archives
	cells := make([]sheetCell, 0, len(entries))
	for _, entry := range entries {
		picture := archives[entry.Source].blocks[entry.BlockID]
		rendered, err := renderPicture(picture)
		if err != nil {
			return "", err
		}
		prefix := "H"
		if entry.Kind == "body" {
			prefix = "B"
		}
		cells = append(cells, sheetCell{
			image: rendered,
			label: fmt.Sprintf("%s%d:%02X", prefix, entry.Archive, entry.BlockID),
		})
	}
	return writeContactSheet(path, cells, sheetColumns)
}

func writeContactSheet(path string, cells []sheetCell, columns int) (string, error) {
	if len(cells) == 0 {
		return "", fmt.Errorf("沒有東西可以排進 %s", path)
	}
	cellW, cellH := 0, 0
	for _, cell := range cells {
		if w := cell.image.Bounds().Dx(); w > cellW {
			cellW = w
		}
		if h := cell.image.Bounds().Dy(); h > cellH {
			cellH = h
		}
	}
	rows := (len(cells) + columns - 1) / columns
	frameW := columns*(cellW+sheetPad) + sheetPad
	frameH := rows*(cellH+sheetLabelHeight+sheetPad) + sheetPad
	sheet := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.RGBA{R: 32, G: 32, B: 32, A: 255}}, image.Point{}, draw.Src)
	for index, cell := range cells {
		column := index % columns
		row := index / columns
		origin := image.Pt(sheetPad+column*(cellW+sheetPad), sheetPad+row*(cellH+sheetLabelHeight+sheetPad))
		rect := image.Rectangle{Min: origin, Max: origin.Add(cell.image.Bounds().Size())}
		draw.Draw(sheet, rect, cell.image, cell.image.Bounds().Min, draw.Src)
		drawLabel(sheet, origin.X, origin.Y+cellH+sheetLabelHeight-3, cell.label)
	}
	return writePNGFile(path, sheet)
}

func drawLabel(dst *image.RGBA, x, y int, text string) {
	drawer := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	drawer.DrawString(text)
}

// --- 建角 14×12 組合表 ---

func buildCreationGrid(zipPath, imgDir string, archives map[string]decodedArchive) (creationGridReport, error) {
	const headCount, bodyCount = 14, 12
	headArchive, ok := archives["HEAD3.DAX"]
	if !ok {
		return creationGridReport{}, fmt.Errorf("缺 HEAD3.DAX，無法核對建角組合")
	}
	bodyArchive, ok := archives["BODY3.DAX"]
	if !ok {
		return creationGridReport{}, fmt.Errorf("缺 BODY3.DAX，無法核對建角組合")
	}

	headSelectors := make([]selectorBlock, 0, headCount)
	for selector := uint8(1); selector <= headCount; selector++ {
		parts, err := assets.ReadCreationPortraitParts(zipPath, selector, 1)
		if err != nil {
			return creationGridReport{}, fmt.Errorf("建角 HEAD selector %d: %w", selector, err)
		}
		blockID, ok := matchBlockID(headArchive.blocks, parts.Head)
		if !ok {
			return creationGridReport{}, fmt.Errorf("建角 HEAD selector %d 在 HEAD3.DAX 裡找不到逐像素相同的 block", selector)
		}
		headSelectors = append(headSelectors, selectorBlock{Selector: selector, BlockID: blockID, BlockIDHex: fmt.Sprintf("0x%02X", blockID)})
	}
	bodySelectors := make([]selectorBlock, 0, bodyCount)
	for selector := uint8(1); selector <= bodyCount; selector++ {
		parts, err := assets.ReadCreationPortraitParts(zipPath, 1, selector)
		if err != nil {
			return creationGridReport{}, fmt.Errorf("建角 BODY selector %d: %w", selector, err)
		}
		blockID, ok := matchBlockID(bodyArchive.blocks, parts.Body)
		if !ok {
			return creationGridReport{}, fmt.Errorf("建角 BODY selector %d 在 BODY3.DAX 裡找不到逐像素相同的 block", selector)
		}
		bodySelectors = append(bodySelectors, selectorBlock{Selector: selector, BlockID: blockID, BlockIDHex: fmt.Sprintf("0x%02X", blockID)})
	}

	cells := make([]sheetCell, 0, headCount*bodyCount)
	for _, head := range headSelectors {
		for _, body := range bodySelectors {
			parts, err := assets.ReadCreationPortraitParts(zipPath, head.Selector, body.Selector)
			if err != nil {
				return creationGridReport{}, err
			}
			composed, err := assets.ComposeCreationPortrait(parts)
			if err != nil {
				return creationGridReport{}, err
			}
			rendered, err := renderPicture(composed)
			if err != nil {
				return creationGridReport{}, err
			}
			cells = append(cells, sheetCell{
				image: rendered,
				label: fmt.Sprintf("H%dB%d", head.Selector, body.Selector),
			})
		}
	}
	sum, err := writeContactSheet(filepath.Join(imgDir, "creation-grid.png"), cells, bodyCount)
	if err != nil {
		return creationGridReport{}, err
	}
	return creationGridReport{
		PNG:           pngFile{Path: filepath.ToSlash(filepath.Join(imgDir, "creation-grid.png")), SHA256: sum},
		Archive:       3,
		HeadSelectors: headSelectors,
		BodySelectors: bodySelectors,
		Evidence:      "exact",
		Source:        "docs/spec/006-dos-portrait-archives.md；internal/assets/portrait.go",
	}, nil
}

func matchBlockID(blocks map[uint8]graphics.Picture, target graphics.Picture) (uint8, bool) {
	for id, candidate := range blocks {
		if candidate.Width() != target.Width() || candidate.Height() != target.Height() {
			continue
		}
		if candidate.ItemCount != target.ItemCount {
			continue
		}
		if bytes.Equal(candidate.Pixels, target.Pixels) {
			return id, true
		}
	}
	return 0, false
}

// --- Rolf 的 NPC 半身像 ---

func buildRolfPortrait(zipPath, imgDir string) (npcEntry, error) {
	portrait, err := assets.ReadNPCPortrait(zipPath, 3, gamepack.RolfPortraitHeadBlock, rolfBodyBlock)
	if err != nil {
		return npcEntry{}, fmt.Errorf("Rolf 半身像: %w", err)
	}
	rendered, err := renderPicture(portrait)
	if err != nil {
		return npcEntry{}, err
	}
	path := filepath.Join(imgDir, "npc-rolf-archive3-head08-body09.png")
	sum, err := writePNGFile(path, rendered)
	if err != nil {
		return npcEntry{}, err
	}
	return npcEntry{
		Name: "Rolf", Archive: 3, HeadBlock: gamepack.RolfPortraitHeadBlock, BodyBlock: rolfBodyBlock,
		PNG:      pngFile{Path: filepath.ToSlash(path), SHA256: sum},
		Evidence: "exact",
		Source:   "docs/spec/117-npc-approach-portrait.md；internal/assets/npc_portrait_test.go",
	}, nil
}
