package gamepack

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"

	goldenbox "github.com/wicanr2/golden-box-remake-engine/engine"
)

// Pool 的 game pack。共用 engine 是作品中立的，所以**內容一律留在這一側**：
// engine 只認 `engine.Pack` 這個結構，不認識菲蘭、不認識任何一條 Pool 的字串。
//
// 分檔、合併順序與 adapter 的邊界在 spec 113。
// 反過來也成立——這個 pack 不得抄 CoAB 的地名、位址或劇情資料。
//
//go:embed pack/*.json
var packFiles embed.FS

// PackDir 是拆檔存放的目錄。載入時依檔名排序合併，所以數字前綴決定合併順序：
// `00-` 放 header 與 presentation，`20-` 放各語言的字串表。
const PackDir = "pack"

var (
	packOnce sync.Once
	packData *goldenbox.Pack
	packErr  error
)

// PackPartNames 列出已提交的 pack 分檔，順序就是合併順序。
func PackPartNames() ([]string, error) {
	entries, err := packFiles.ReadDir(PackDir)
	if err != nil {
		return nil, fmt.Errorf("list Pool game pack parts: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, PackDir+"/"+entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("Pool game pack has no parts under %s", PackDir)
	}
	return names, nil
}

// Pack 載入並合併整份 pack。只解析一次，之後回同一個指標。
func Pack() (*goldenbox.Pack, error) {
	packOnce.Do(func() {
		names, err := PackPartNames()
		if err != nil {
			packErr = err
			return
		}
		packData, packErr = goldenbox.LoadPackPartsFS(packFiles.ReadFile, names)
	})
	return packData, packErr
}

// LocaleTable 取出一個語言的字串表。找不到那個語言就回錯，不要靜靜回空表——
// 空表的症狀是整個介面變成空字串，而那看起來像排版壞掉，不像資料沒載到。
func LocaleTable(locale string) (map[string]string, error) {
	pack, err := Pack()
	if err != nil {
		return nil, err
	}
	table, ok := pack.Locales[locale]
	if !ok {
		return nil, fmt.Errorf("Pool game pack has no locale %q", locale)
	}
	return table, nil
}
