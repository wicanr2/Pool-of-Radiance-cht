// Command pool-font-coverage 報出遊戲要顯示、但倚天字型畫不出來的字。
//
// 畫不出來的字在畫面上是一個空白方塊，看起來像繪圖壞掉，而不是「這套字型
// 沒有這個字」。兩者的處置完全不同，所以要先分得出來：本工具拿實際的字型檔
// 逐字問「取得到字模嗎」，答案是實測，不是查編碼表。
//
// 字型是第三方資產、不進 repo，因此這是手動跑的稽核工具，不是測試。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"image"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"golang.org/x/image/font/basicfont"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/etenfont"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gametext"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

type miss struct {
	Rune  string `json:"rune"`
	Code  string `json:"code"`
	Count int    `json:"count"`
	Where string `json:"where"`
}

func main() {
	standard := flag.String("eten-font", "", "stdfont.15 的路徑")
	symbols := flag.String("eten-symbol-font", "", "usrfont.15m 的路徑")
	ascii := flag.String("eten-ascii-font", "", "ascfont.15 的路徑")
	flag.Parse()
	if *standard == "" {
		fmt.Fprintln(os.Stderr, "需要 -eten-font")
		os.Exit(2)
	}
	face, err := etenfont.LoadWithASCII(*standard, *symbols, *ascii, basicfont.Face7x13, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	counts := map[rune]int{}
	where := map[rune]string{}
	note := func(text, label string) {
		// 與畫面走同一條替換：稽核的是「實際會畫出來的字」。
		for _, r := range etenfont.ReplaceUnavailable(text) {
			// 空白字元本來就該是空的。
			if r == ' ' || r == '\t' || r == '\u3000' {
				continue
			}
			if r == '\n' {
				continue
			}
			// 判準是「畫出來有東西」，不是「取得到字模格」。
			// ASCII 那條路對任何 <= 0xFF 的碼位都取得到格子，但格子可能整片空白
			// ——那在畫面上與缺字沒有分別，而只看 ok 會把它漏掉。
			if mask, drawable := face.Bitmap(r); drawable && !blank(mask) {
				continue
			}
			counts[r]++
			if where[r] == "" {
				where[r] = label
			}
		}
	}

	corpus, err := journal.TraditionalChinese()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, kind := range journal.Kinds {
		for _, entry := range corpus.Entries(kind) {
			note(entry.Text, fmt.Sprintf("journal %s %s", entry.Kind, entry.ID))
		}
	}
	// 畫面上的字不只來自那兩份資料檔：UI 的字串直接寫在程式碼裡。
	// 只掃字串常值（不掃註解），因為只有它們會被畫出來。
	if err := noteSourceStrings(note); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	catalogue, err := gametext.TraditionalChinese()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, source := range catalogue.Sources() {
		note(catalogue.Translate(source), "gametext")
	}

	report := make([]miss, 0, len(counts))
	for r, count := range counts {
		report = append(report, miss{
			Rune: string(r), Code: fmt.Sprintf("U+%04X", r), Count: count, Where: where[r],
		})
	}
	sort.Slice(report, func(i, j int) bool { return report[i].Code < report[j].Code })
	out, err := json.MarshalIndent(struct {
		Missing []miss `json:"missing"`
		Total   int    `json:"total_occurrences"`
	}{report, sum(counts)}, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(append(out, '\n'))
	if len(report) > 0 {
		os.Exit(1)
	}
}

// noteSourceStrings 走過 repo 裡每個 .go 檔的字串常值。
func noteSourceStrings(note func(text, label string)) error {
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "workplace", "docs", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			note(value, path)
			return true
		})
		return nil
	})
}

// blank 判斷字模是不是整片空白。空白字元本來就該是空的，不算缺字。
func blank(mask *image.Alpha) bool {
	if mask == nil {
		return true
	}
	for _, value := range mask.Pix {
		if value != 0 {
			return false
		}
	}
	return true
}

func sum(counts map[rune]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}
