package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	poolchar "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 建角完成時寫出原版格式的角色檔。原版就在這個時機做同一件事：icon 確認
// `YES` 之後寫出 `<NAME>.CHA` 與 `<NAME>.SPC`，再回到 Party Creation Menu
//（spec 003 第 11 步）。
//
// remake 的真相仍然是自己的 JSON 存檔；這三個檔是**匯出**，壞掉不會弄丟角色。
const (
	// dosExportDir 是三個檔放的位置，與 JSON 存檔同一層。
	dosExportDir = "dos"
	// 副檔名：`.CHA` 與 `.SPC` 有 spec 003 的原版證據；`.ITM` 是從預設人物
	// 三個一組的形狀推的（`chrdatd4.sav`／`.itm`／`.spc`），屬**強推論**。
	dosRecordExtension  = ".CHA"
	dosItemExtension    = ".ITM"
	dosEffectsExtension = ".SPC"
)

// dosCharacterFiles 是一名角色的三個原版檔。
type dosCharacterFiles struct {
	Record  []byte
	Items   []byte
	Effects []byte
}

// buildDOSCharacterFiles 把一名 remake 角色轉成原版的三個檔。
//
// base 的來源分兩種：這個角色本來就是從原版記錄來的（NPC 的 `Record`），
// 就疊在那份上面，沒解出來的 285 bytes 原封不動；remake 自己建的角色沒有
// 那份，用 spec 063 的建角基礎值。
//
// 疊完之後一定要重算——remake 的角色模型只存「輸入」，THAC0、AC、負重與
// 移動力都是現算現用的，不重算就會寫出一份戰鬥數值全是 0 的記錄。
func (a *app) buildDOSCharacterFiles(member poolsave.Character) (dosCharacterFiles, error) {
	base := poolchar.NewDOSRecordBase()
	if len(member.Record) == poolchar.DOSRecordSize {
		base = member.Record
	}
	record, err := poolchar.ExportDOSRecord(base, member)
	if err != nil {
		return dosCharacterFiles{}, err
	}
	items, err := poolchar.ExportDOSItems(member.Inventory)
	if err != nil {
		return dosCharacterFiles{}, err
	}
	chain := make([][]byte, 0, len(member.Inventory))
	for offset := 0; offset+poolchar.ItemRecordSize <= len(items); offset += poolchar.ItemRecordSize {
		chain = append(chain, items[offset:offset+poolchar.ItemRecordSize])
	}
	record, err = gamepack.RecomputeCombatFields(record, chain, a.itemTypes)
	if err != nil {
		return dosCharacterFiles{}, err
	}
	return dosCharacterFiles{
		Record:  record,
		Items:   items,
		Effects: poolchar.ExportDOSEffects(member.Effects),
	}, nil
}

// dosExportName 把角色姓名變成檔名主幹。原版的姓名可以有空白
//（`PRINCESS FATIMA`），DOS 檔名不行，所以換成底線並截到 8 個字元。
func dosExportName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		}
		return '_'
	}, strings.TrimSpace(name))
	if cleaned == "" {
		return "NONAME"
	}
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}
	return cleaned
}

// writeDOSCharacterFiles 把三個檔寫進 dir。三個都成功才算成功；
// 中途失敗直接回錯，不刪已經寫出去的——半份匯出看得見，靜靜刪掉看不見。
func writeDOSCharacterFiles(dir, name string, files dosCharacterFiles) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	stem := filepath.Join(dir, dosExportName(name))
	for _, item := range []struct {
		path string
		data []byte
	}{
		{stem + dosRecordExtension, files.Record},
		{stem + dosItemExtension, files.Items},
		{stem + dosEffectsExtension, files.Effects},
	} {
		if err := os.WriteFile(item.path, item.data, 0o644); err != nil {
			return fmt.Errorf("Pool DOS export %s: %w", filepath.Base(item.path), err)
		}
	}
	return nil
}
