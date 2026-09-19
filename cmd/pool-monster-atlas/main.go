// pool-monster-atlas 匯出怪物圖鑑（issue #50 前半）要用的資料與戰場 sprite 圖片。
//
// 四份輸出：
//
//  1. docs/audit/monster-atlas.json —— MON1CHA.DAX..MON8CHA.DAX 每一筆 285-byte
//     記錄的戰鬥數值（HD／HP／AC／THAC0／傷害骰／移動／經驗值)，配對應
//     MONnSPC.DAX 的效果節點；每筆都帶 archive／block，可以逐條回查 spec 048。
//  2. docs/archaeology/img/monsters/cbody-*.png —— 怪物在戰場上實際的造形：
//     `CBODY.DAX` 的三十二種身體各配同一顆頭（`CHEAD.DAX` block 1），做法
//     跟 cmd/pool-game/sprite_overview.go 的 drawMonsterOverview 一樣，直接
//     呼叫同一套 production code（internal/assets 的 ReadCombatIcon）。
//  3. docs/archaeology/img/monsters/comspr-*.png／icon-*.png —— `COMSPR.DAX`／
//     `ICON.DAX` 實際掃到的每一個 block；這兩個檔案裝的是戰鬥特效與狀態圖示
//     （箭、飛斧、擲石、閃光、爆炸等），不是怪物造形，收在這裡只是把它們
//     的內容如實記錄下來。
//  4. 本檔自己：可重跑，不手改圖。
//
// 用法（容器內）：
//
//	tools/go.sh run ./cmd/pool-monster-atlas \
//	  -zip "Pool of Radiance (1988).zip" \
//	  -img-out docs/archaeology/img/monsters \
//	  -receipt docs/audit/monster-atlas.json
package main

import (
	"archive/zip"
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

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/assets"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/graphics"
)

const monsterArchiveCount = 8

type attackSlotRow struct {
	Slot        uint8 `json:"slot"`
	AttackRate  uint8 `json:"attack_rate_raw"`
	DiceCount   uint8 `json:"dice_count"`
	DiceSides   uint8 `json:"dice_sides"`
	DiceBonus   int8  `json:"dice_bonus"`
	HasNoDamage bool  `json:"has_no_damage"`
}

type effectRow struct {
	Code       uint8  `json:"code"`
	CodeHex    string `json:"code_hex"`
	PayloadHex string `json:"payload_hex"`
}

type monsterRow struct {
	Archive               int             `json:"archive"`
	Block                 int             `json:"block"`
	Name                  string          `json:"name"`
	CreatureType          uint8           `json:"creature_type_raw"`
	BodySize              uint8           `json:"body_size_raw"`
	Dexterity             uint8           `json:"dexterity"`
	MaxHitPoints          uint8           `json:"max_hit_points"`
	ArmorClass            int             `json:"armor_class"`
	Thac0Surface          int             `json:"thac0_surface,omitempty"`
	Thac0Error            string          `json:"thac0_error,omitempty"`
	Attacks               []attackSlotRow `json:"attacks"`
	Movement              uint8           `json:"movement"`
	ExperienceBase        uint16          `json:"experience_base"`
	ExperiencePerHitPoint uint8           `json:"experience_per_hit_point"`
	Effects               []effectRow     `json:"effects,omitempty"`
	EffectsSource         string          `json:"effects_source"`
}

type spriteBlockRow struct {
	ID      uint8  `json:"id"`
	Hex     string `json:"hex"`
	Pose    string `json:"pose"`
	StandID uint8  `json:"stand_id"`
	File    string `json:"file"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Items   int    `json:"item_count"`
}

type spriteArchiveRow struct {
	Description string           `json:"description"`
	Member      string           `json:"member"`
	SHA256      string           `json:"sha256"`
	Blocks      []spriteBlockRow `json:"blocks"`
}

// bodyRow 是 CBODY.DAX 一種身體配固定頭之後的合成結果。BodyBlockHex／
// HeadBlockHex 是 assets.CombatIconBlockIDs 實際算出來的來源 block，供逐筆
// 回查（不是本檔自己編號）。
type bodyRow struct {
	Body         uint8  `json:"body"`
	BodyBlockHex string `json:"body_block_hex"`
	HeadBlockHex string `json:"head_block_hex"`
	File         string `json:"file"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

type bodyArchiveRow struct {
	Description string      `json:"description"`
	BodyMember  string      `json:"body_member"`
	BodySHA256  string      `json:"body_sha256"`
	HeadMember  string      `json:"head_member"`
	HeadSHA256  string      `json:"head_sha256"`
	HeadUsed    uint8       `json:"head_used"`
	Colours     [6][2]uint8 `json:"colours"`
	Bodies      []bodyRow   `json:"bodies"`
}

type receipt struct {
	Schema        string                      `json:"schema"`
	GeneratedBy   string                      `json:"generated_by"`
	ZIPPath       string                      `json:"zip_path"`
	ZIPSHA256     string                      `json:"zip_sha256"`
	MemberSHA256  map[string]string           `json:"member_sha256"`
	Monsters      []monsterRow                `json:"monsters"`
	Bodies        bodyArchiveRow              `json:"bodies"`
	Sprites       map[string]spriteArchiveRow `json:"sprites"`
	OverviewImage string                      `json:"overview_image"`
	OverviewGrid  []string                    `json:"overview_grid_order"`
	BodyOverview  string                      `json:"body_overview_image"`
	Counts        map[string]int              `json:"counts"`
	Notes         []string                    `json:"notes"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS 原版 ZIP")
	imgOut := flag.String("img-out", "docs/archaeology/img/monsters", "sprite PNG 輸出目錄")
	receiptOut := flag.String("receipt", "docs/audit/monster-atlas.json", "收據 JSON 輸出路徑")
	flag.Parse()

	if err := run(*zipPath, *imgOut, *receiptOut); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(zipPath, imgOut, receiptOut string) error {
	zipSHA256, err := hashFile(zipPath)
	if err != nil {
		return fmt.Errorf("hash %s: %w", zipPath, err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open DOS ZIP: %w", err)
	}
	defer zr.Close()

	if err := os.MkdirAll(imgOut, 0o755); err != nil {
		return err
	}

	memberSHA256 := map[string]string{}

	monsters, err := scanMonsters(zipPath, zr.File, memberSHA256)
	if err != nil {
		return err
	}

	bodies, bodyCells, err := scanBodies(zipPath, zr.File, imgOut, memberSHA256)
	if err != nil {
		return err
	}
	bodyOverviewImage := "cbody-overview.png"
	if err := writeOverview(filepath.Join(imgOut, bodyOverviewImage), bodyCells); err != nil {
		return err
	}

	sprites, overviewCells, overviewGrid, err := scanSprites(zr.File, imgOut, memberSHA256)
	if err != nil {
		return err
	}
	if row, ok := sprites["comspr"]; ok {
		row.Description = "戰鬥特效與狀態圖示（箭、飛斧、擲石、閃光、爆炸等），不是怪物造形——實際怪物造形見 bodies 區塊的 CBODY.DAX 匯出。"
		sprites["comspr"] = row
	}
	if row, ok := sprites["icon"]; ok {
		row.Description = "戰鬥特效與狀態圖示，性質同 COMSPR.DAX，不是怪物造形。"
		sprites["icon"] = row
	}

	overviewImage := "overview.png"
	if err := writeOverview(filepath.Join(imgOut, overviewImage), overviewCells); err != nil {
		return err
	}

	uniqueNames := map[string]bool{}
	for _, m := range monsters {
		uniqueNames[m.Name] = true
	}
	spriteBlockTotal := 0
	for _, s := range sprites {
		spriteBlockTotal += len(s.Blocks)
	}

	out := receipt{
		Schema:        "pool-monster-atlas/2",
		GeneratedBy:   "cmd/pool-monster-atlas",
		ZIPPath:       zipPath,
		ZIPSHA256:     zipSHA256,
		MemberSHA256:  memberSHA256,
		Monsters:      monsters,
		Bodies:        bodies,
		Sprites:       sprites,
		OverviewImage: overviewImage,
		OverviewGrid:  overviewGrid,
		BodyOverview:  bodyOverviewImage,
		Counts: map[string]int{
			"monster_records": len(monsters),
			"unique_names":    len(uniqueNames),
			"sprite_blocks":   spriteBlockTotal,
			"cbody_bodies":    len(bodies.Bodies),
		},
		Notes: []string{
			"怪物在戰場上的實際造形是 CBODY.DAX 的三十二種身體配固定的頭（cmd/pool-game/sprite_overview.go 的 drawMonsterOverview 已經這樣畫），見 bodies 區塊；COMSPR.DAX／ICON.DAX 是戰鬥特效與狀態圖示，不是怪物造形，見 sprites 區塊各自的 description。internal/assets/monster_sprite.go 檔頭的舊註解仍把 COMSPR.DAX 稱作怪物圖形，待那邊自行更新，這份收據不依賴那個說法。",
			"THAC0 的樣板欄 +110h 多半是殘值，這裡的 thac0_surface 一律用 CombatThac0Internal()（+2Dh 再加力量修正）換算，理由見 internal/gamepack/monster.go。",
			"MON1SPC.DAX／MON3SPC.DAX 在原版 ZIP 裡不存在；這兩個 archive 的怪物 effects_source 一律是「無 MONxSPC.DAX」，不是掃描遺漏。",
		},
	}

	raw, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(receiptOut, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("%d 筆怪物記錄（%d 個不同名稱），%d 個 CBODY 造形，%d 張特效 sprite block，寫入 %s 與 %s\n",
		len(monsters), len(uniqueNames), len(bodies.Bodies), spriteBlockTotal, imgOut, receiptOut)
	return nil
}

// scanBodies 依 drawMonsterOverview 的做法，把 CBODY.DAX 的三十二種身體各配
// 同一顆頭（CHEAD.DAX block 1）合成戰場造形，直接呼叫 internal/assets 既有
// production code（assets.ReadCombatIcon），不重新實作合成或配色邏輯。
func scanBodies(zipPath string, files []*zip.File, imgOut string, memberSHA256 map[string]string) (bodyArchiveRow, []*image.RGBA, error) {
	const bodyMemberName, headMemberName = "CBODY.DAX", "CHEAD.DAX"
	const headUsed = 1
	const bodyCount = 32

	bodyMember, err := findMember(files, bodyMemberName)
	if err != nil {
		return bodyArchiveRow{}, nil, err
	}
	_, bodyHash, err := readMember(bodyMember)
	if err != nil {
		return bodyArchiveRow{}, nil, fmt.Errorf("%s: %w", bodyMemberName, err)
	}
	memberSHA256[bodyMemberName] = bodyHash

	headMember, err := findMember(files, headMemberName)
	if err != nil {
		return bodyArchiveRow{}, nil, err
	}
	_, headHash, err := readMember(headMember)
	if err != nil {
		return bodyArchiveRow{}, nil, fmt.Errorf("%s: %w", headMemberName, err)
	}
	memberSHA256[headMemberName] = headHash

	colours := [6][2]uint8{{1, 9}, {2, 10}, {3, 11}, {4, 12}, {6, 14}, {7, 15}}
	result := bodyArchiveRow{
		Description: "怪物在戰場上的實際造形：CBODY.DAX 的三十二種身體各配同一顆頭（CHEAD.DAX block 1），配色沿用 drawMonsterOverview 的預設六組。",
		BodyMember:  bodyMemberName, BodySHA256: bodyHash,
		HeadMember: headMemberName, HeadSHA256: headHash,
		HeadUsed: headUsed, Colours: colours,
	}

	var cells []*image.RGBA
	for body := uint8(0); body < bodyCount; body++ {
		selection := assets.CombatIconSelection{Head: headUsed, Body: body, Size: 1}
		picture, err := assets.ReadCombatIcon(zipPath, selection, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告：CBODY body %d 讀取失敗：%v\n", body, err)
			continue
		}
		rgba, err := picture.RGBA(0, graphics.EGA16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告：CBODY body %d 上色失敗：%v\n", body, err)
			continue
		}
		headID, bodyID, err := assets.CombatIconBlockIDs(selection, false)
		if err != nil {
			return bodyArchiveRow{}, nil, fmt.Errorf("CBODY body %d block id: %w", body, err)
		}
		filename := fmt.Sprintf("cbody-%02d.png", body)
		scaled := scaleNearest(rgba, 4)
		if err := writePNG(filepath.Join(imgOut, filename), scaled); err != nil {
			return bodyArchiveRow{}, nil, err
		}
		result.Bodies = append(result.Bodies, bodyRow{
			Body: body, BodyBlockHex: fmt.Sprintf("%02Xh", bodyID), HeadBlockHex: fmt.Sprintf("%02Xh", headID),
			File: filename, Width: picture.Width(), Height: picture.Height(),
		})
		cells = append(cells, scaled)
	}
	return result, cells, nil
}

func hashFile(path string) (string, error) {
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

func findMember(files []*zip.File, name string) (*zip.File, error) {
	for _, candidate := range files {
		if strings.EqualFold(filepath.Base(candidate.Name), name) {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("DOS ZIP 沒有 %s", name)
}

func readMember(member *zip.File) ([]byte, string, error) {
	stream, err := member.Open()
	if err != nil {
		return nil, "", err
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, 1<<20))
	closeErr := stream.Close()
	if readErr != nil {
		return nil, "", readErr
	}
	if closeErr != nil {
		return nil, "", closeErr
	}
	if uint64(len(data)) != member.UncompressedSize64 {
		return nil, "", fmt.Errorf("%s exceeds 1 MiB bound", member.Name)
	}
	sum := sha256.Sum256(data)
	return data, fmt.Sprintf("%x", sum[:]), nil
}

// scanMonsters 對每個 archive 先掃 MONnCHA.DAX 實際有哪些 block（不是盲猜
// 0..255），再逐 block 用 typed accessor 取數值，配對應 MONnSPC.DAX 的效果。
func scanMonsters(zipPath string, files []*zip.File, memberSHA256 map[string]string) ([]monsterRow, error) {
	var rows []monsterRow
	for archive := 1; archive <= monsterArchiveCount; archive++ {
		chaName := fmt.Sprintf("MON%dCHA.DAX", archive)
		chaMember, err := findMember(files, chaName)
		if err != nil {
			return nil, err
		}
		chaData, chaHash, err := readMember(chaMember)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", chaName, err)
		}
		memberSHA256[chaName] = chaHash
		blocks, err := dax.Parse(chaData)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", chaName, err)
		}
		ids := make([]int, 0, len(blocks))
		for _, block := range blocks {
			ids = append(ids, int(block.Entry.ID))
		}
		sort.Ints(ids)

		spcName := fmt.Sprintf("MON%dSPC.DAX", archive)
		spcMember, spcErr := findMember(files, spcName)
		hasSPC := spcErr == nil
		if hasSPC {
			_, spcHash, err := readMember(spcMember)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", spcName, err)
			}
			memberSHA256[spcName] = spcHash
		}

		for _, id := range ids {
			record, err := gamepack.ReadDOSMonsterRecord(zipPath, uint8(archive), uint8(id))
			if err != nil {
				fmt.Fprintf(os.Stderr, "警告：%s block %d 讀取失敗：%v\n", chaName, id, err)
				continue
			}
			row := monsterRow{
				Archive:               archive,
				Block:                 id,
				Name:                  record.Name,
				CreatureType:          record.CreatureType(),
				BodySize:              record.BodySize(),
				Dexterity:             record.Dexterity(),
				MaxHitPoints:          record.MaxHitPoints(),
				ArmorClass:            record.ArmorClass(),
				Movement:              record.Movement(),
				ExperienceBase:        record.ExperienceBase(),
				ExperiencePerHitPoint: record.ExperiencePerHitPoint(),
			}
			if internal, err := record.CombatThac0Internal(); err != nil {
				row.Thac0Error = err.Error()
			} else {
				row.Thac0Surface = 60 - int(internal)
			}
			for slot := uint8(1); slot <= gamepack.MonsterAttackSlots; slot++ {
				damage, dmgErr := record.AttackDamage(slot)
				rate, rateErr := record.BaseAttackRate(slot)
				if dmgErr != nil || rateErr != nil {
					continue
				}
				row.Attacks = append(row.Attacks, attackSlotRow{
					Slot:        slot,
					AttackRate:  rate,
					DiceCount:   damage.Count,
					DiceSides:   damage.Sides,
					DiceBonus:   damage.Bonus,
					HasNoDamage: damage.Count == 0 && damage.Sides == 0,
				})
			}
			switch {
			case !hasSPC:
				row.EffectsSource = "無 MONxSPC.DAX"
			default:
				nodes, err := gamepack.ReadDOSMonsterEffects(zipPath, uint8(archive), uint8(id))
				switch {
				case err != nil:
					row.EffectsSource = fmt.Sprintf("讀取失敗：%v", err)
				case len(nodes) == 0:
					row.EffectsSource = spcName + "（此 block 無節點）"
				default:
					row.EffectsSource = spcName
					for _, node := range nodes {
						row.Effects = append(row.Effects, effectRow{
							Code:       node.Code,
							CodeHex:    fmt.Sprintf("%02Xh", node.Code),
							PayloadHex: fmt.Sprintf("%X", node.Payload),
						})
					}
				}
			}
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Archive != rows[j].Archive {
			return rows[i].Archive < rows[j].Archive
		}
		return rows[i].Block < rows[j].Block
	})
	return rows, nil
}

// scanSprites 對 COMSPR.DAX 與 ICON.DAX 掃出實際存在的每一個 block，各輸出
// 一張放大 4 倍的 PNG，同時把縮圖收進總覽格。block ID ≥ 80h 視為「動作態」，
// 對應站立態是 id − 80h（monster_sprite.go 的既有慣例），這裡只是照 ID
// 分類，不影響輸出張數。
func scanSprites(files []*zip.File, imgOut string, memberSHA256 map[string]string) (map[string]spriteArchiveRow, []*image.RGBA, []string, error) {
	daxFiles := []string{"COMSPR.DAX", "ICON.DAX"}
	sprites := map[string]spriteArchiveRow{}
	var overviewCells []*image.RGBA
	var overviewGrid []string

	for _, daxName := range daxFiles {
		member, err := findMember(files, daxName)
		if err != nil {
			return nil, nil, nil, err
		}
		data, hash, err := readMember(member)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", daxName, err)
		}
		memberSHA256[daxName] = hash
		blocks, err := dax.Parse(data)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", daxName, err)
		}
		sort.Slice(blocks, func(i, j int) bool { return blocks[i].Entry.ID < blocks[j].Entry.ID })

		key := strings.ToLower(strings.TrimSuffix(daxName, ".DAX"))
		var rows []spriteBlockRow
		for _, block := range blocks {
			picture, err := graphics.ParsePicture(block.Data, true, 0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "警告：%s block %d 解碼失敗：%v\n", daxName, block.Entry.ID, err)
				continue
			}
			rgba, err := picture.RGBA(0, graphics.EGA16)
			if err != nil {
				fmt.Fprintf(os.Stderr, "警告：%s block %d 上色失敗：%v\n", daxName, block.Entry.ID, err)
				continue
			}
			pose, standID := "stand", block.Entry.ID
			if block.Entry.ID >= 0x80 {
				pose, standID = "action", block.Entry.ID-0x80
			}
			filename := fmt.Sprintf("%s-%02x-%s.png", key, standID, pose)
			scaled := scaleNearest(rgba, 4)
			if err := writePNG(filepath.Join(imgOut, filename), scaled); err != nil {
				return nil, nil, nil, err
			}
			rows = append(rows, spriteBlockRow{
				ID: block.Entry.ID, Hex: fmt.Sprintf("%02Xh", block.Entry.ID), Pose: pose,
				StandID: standID, File: filename,
				Width: picture.Width(), Height: picture.Height(), Items: int(picture.ItemCount),
			})
			overviewCells = append(overviewCells, scaled)
			overviewGrid = append(overviewGrid, fmt.Sprintf("%s/%s", daxName, filename))
		}
		sprites[key] = spriteArchiveRow{Member: daxName, SHA256: hash, Blocks: rows}
	}
	return sprites, overviewCells, overviewGrid, nil
}

func scaleNearest(src *image.RGBA, factor int) *image.RGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width*factor, height*factor))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := src.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					dst.SetRGBA(x*factor+dx, y*factor+dy, pixel)
				}
			}
		}
	}
	return dst
}

func writeOverview(path string, cells []*image.RGBA) error {
	if len(cells) == 0 {
		return fmt.Errorf("沒有可排總覽的 sprite")
	}
	const columns = 8
	cellW, cellH := 0, 0
	for _, cell := range cells {
		if w := cell.Bounds().Dx(); w > cellW {
			cellW = w
		}
		if h := cell.Bounds().Dy(); h > cellH {
			cellH = h
		}
	}
	rows := (len(cells) + columns - 1) / columns
	const pad = 6
	sheet := image.NewRGBA(image.Rect(0, 0, columns*(cellW+pad)+pad, rows*(cellH+pad)+pad))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{C: color.RGBA{R: 32, G: 32, B: 32, A: 255}}, image.Point{}, draw.Src)
	for index, cell := range cells {
		column, row := index%columns, index/columns
		origin := image.Pt(pad+column*(cellW+pad), pad+row*(cellH+pad))
		rect := image.Rectangle{Min: origin, Max: origin.Add(cell.Bounds().Size())}
		draw.Draw(sheet, rect, cell, cell.Bounds().Min, draw.Src)
	}
	return writePNG(path, sheet)
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
