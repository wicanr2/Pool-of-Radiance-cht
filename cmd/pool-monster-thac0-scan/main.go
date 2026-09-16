// pool-monster-thac0-scan 把八個 MONnCHA.DAX 的樣板記錄裡跟命中有關的幾格列出來
// （#36／#31，spec 063）：`+110h`（執行期 THAC0 欄）、`+2Dh`（基礎 THAC0）、`+111h`
// （AC）、`+2Fh`（職業碼）、`+96h..+9Dh`（八個職業等級）、`+0CCh`（備妥武器的遠指標，
// 樣板裡通常是 0）、`+0A0h..+0A7h`（攻擊次數與傷害骰的來源欄）。
//
// 用途只有一個：分辨 `+110h` 在樣板裡是「值」還是「殘值」。overlay-25 entry 7（`0BBEh`）
// 重算時 `+110h = +2Dh`（`0E65h`），所以真正決定命中的是 `+2Dh`；把兩欄並排，
// 不一致的那幾筆就是殘值的形狀。輸出寫進 docs/audit/monster-thac0-template-scan.json。
//
//	tools/go.sh run ./cmd/pool-monster-thac0-scan -zip "Pool of Radiance (1988).zip" -out docs/audit/monster-thac0-template-scan.json
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

type row struct {
	Archive     int    `json:"archive"`
	Block       int    `json:"block"`
	Name        string `json:"name"`
	THAC0Field  int    `json:"thac0_110h"`
	BaseTHAC0   int    `json:"base_thac0_2Dh"`
	ArmorClass  int    `json:"ac_111h"`
	ClassCode   int    `json:"class_2Fh"`
	Levels      [8]int `json:"levels_96h"`
	WeaponPtr   string `json:"weapon_0CCh"`
	AttackRates [2]int `json:"attack_rates_0A1h"`
	Mismatch    bool   `json:"thac0_differs_from_base"`
	Bit7        bool   `json:"thac0_bit7"`
}

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "原版 ZIP")
	out := flag.String("out", "", "輸出 JSON（空的就只印表）")
	flag.Parse()

	zr, err := zip.OpenReader(*zipPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer zr.Close()
	hashes := map[string]string{}
	rows := []row{}
	for archive := 1; archive <= 8; archive++ {
		name := fmt.Sprintf("MON%dCHA.DAX", archive)
		var member *zip.File
		for _, file := range zr.File {
			if strings.EqualFold(filepath.Base(file.Name), name) {
				member = file
			}
		}
		if member == nil {
			fmt.Fprintln(os.Stderr, "沒有", name)
			os.Exit(1)
		}
		reader, err := member.Open()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		sum := sha256.New()
		io.Copy(sum, reader)
		reader.Close()
		hashes[name] = fmt.Sprintf("%x", sum.Sum(nil))
		for block := 0; block < 256; block++ {
			record, err := gamepack.ReadDOSMonsterRecord(*zipPath, uint8(archive), uint8(block))
			if err != nil {
				continue
			}
			entry := row{Archive: archive, Block: block, Name: record.Name,
				THAC0Field: int(record.Raw[0x110]), BaseTHAC0: int(record.Raw[0x2D]),
				ArmorClass: int(record.Raw[0x111]), ClassCode: int(record.Raw[0x2F]),
				WeaponPtr:   fmt.Sprintf("%02x%02x%02x%02x", record.Raw[0xCC], record.Raw[0xCD], record.Raw[0xCE], record.Raw[0xCF]),
				AttackRates: [2]int{int(record.Raw[0xA1]), int(record.Raw[0xA2])}}
			for class := 0; class < 8; class++ {
				entry.Levels[class] = int(record.Raw[0x96+class])
			}
			entry.Mismatch = entry.THAC0Field != entry.BaseTHAC0
			entry.Bit7 = entry.THAC0Field&0x80 != 0
			rows = append(rows, entry)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Archive != rows[j].Archive {
			return rows[i].Archive < rows[j].Archive
		}
		return rows[i].Block < rows[j].Block
	})
	mismatch, bit7 := 0, 0
	for _, entry := range rows {
		if entry.Mismatch {
			mismatch++
		}
		if entry.Bit7 {
			bit7++
		}
		if entry.Mismatch || entry.Bit7 {
			fmt.Printf("mon%d/%-3d %-16s +110h=%3d +2Dh=%3d ac=%3d class=%d levels=%v weapon=%s\n",
				entry.Archive, entry.Block, entry.Name, entry.THAC0Field, entry.BaseTHAC0, entry.ArmorClass,
				entry.ClassCode, entry.Levels, entry.WeaponPtr)
		}
	}
	fmt.Printf("%d 筆；+110h ≠ +2Dh 的 %d 筆，bit 7 亮的 %d 筆\n", len(rows), mismatch, bit7)
	if *out == "" {
		return
	}
	payload := map[string]any{
		"schema": "pool-monster-thac0-template-scan/1", "input_sha256": hashes,
		"fields": map[string]string{"thac0_110h": "執行期 THAC0 欄（entry 7 重算時被 +2Dh 蓋掉）", "base_thac0_2Dh": "基礎 THAC0 internal（60 − 值 = 表面 THAC0）",
			"ac_111h": "AC internal", "class_2Fh": "職業碼", "levels_96h": "八個職業等級", "weapon_0CCh": "備妥武器遠指標（樣板）", "attack_rates_0A1h": "+0A1h／+0A2h"},
		"records": rows, "mismatch_count": mismatch, "bit7_count": bit7,
	}
	raw, _ := json.MarshalIndent(payload, "", " ")
	if err := os.WriteFile(*out, append(raw, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
