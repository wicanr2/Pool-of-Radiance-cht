// Command pool-name-audit 把說明書定案的專有名詞回對原版資料自己的字串。
//
// 說明書是第二來源：它證明「當年官方怎麼譯」，不證明「遊戲畫面上出現什麼」。
//
// 原版的文字有兩種存法，兩種都要掃，只掃一種會得到假的零：
// 怪物名等是明碼 ASCII，ECL 的敘述文字則是 6-bit packed、以 0x80 長度前綴標記。
// 找不到的專名照樣列出來，讓「原版真的沒有這個詞」與「掃描面有洞」分得開。
package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/golden-box-remake-engine/dax"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

// minimumRun 是被當成一段文字的最短可列印長度。太短會把座標與旗標的位元組
// 湊成假字串，太長會漏掉像 "Phlan" 這種短地名所在的短句。
const minimumRun = 4

// Term 是一個要回對的專有名詞。Settled 是說明書的定案譯名，只作報表對照，
// 不參與比對。
type Term struct {
	English string `json:"english"`
	Settled string `json:"settled"`
	// Variants 是同一個對象在原版文字裡可能的其他拼法。少列一個拼法就會得到
	// 一個假的零，所以連字號、拆寫與已知誤拼都要列進來。
	Variants []string `json:"variants,omitempty"`
}

// terms 逐條取自 docs/reference/manual/glossary.md 的「定案譯名」表與撞名段。
var terms = []Term{
	{English: "Phlan", Settled: "菲蘭", Variants: []string{"New Phlan"}},
	{English: "Sembia", Settled: "桑比亞"},
	{English: "Braccio", Settled: "巴西歐"},
	{English: "Valjevo", Settled: "瓦傑渥"},
	{English: "Urslingen", Settled: "烏斯林根", Variants: []string{"Werner"}},
	{English: "Thentia", Settled: "珊提亞"},
	{English: "Mulmaster", Settled: "馬爾瑪斯特"},
	{English: "Lis", Settled: "里斯河", Variants: []string{"Lis River"}},
	{English: "Tesh", Settled: "塔斯河", Variants: []string{"Tesh River"}},
	{English: "Stormy Bay", Settled: "暴風灣", Variants: []string{"Stormy"}},
	{English: "Sokal", Settled: "索卡爾城堡", Variants: []string{"Sokal Keep", "Kosal"}},
	{English: "Kobold", Settled: "小妖魔", Variants: []string{"Kobolds"}},
	{English: "Magic-User", Settled: "魔法師", Variants: []string{"Magic User", "Magic-Users", "Magic Users"}},
	{English: "Thief", Settled: "賊", Variants: []string{"Thieves"}},
	{English: "Twilight", Settled: "黃昏之界", Variants: []string{"Twilight Marsh", "Twilight Mash"}},
	{English: "Yarash", Settled: "亞拉斯"},
	{English: "Yulash", Settled: "尤拉斯"},
}

// spellings 把一個 term 的所有拼法攤平。
func (term Term) spellings() []string {
	all := make([]string, 0, 1+len(term.Variants))
	all = append(all, term.English)
	all = append(all, term.Variants...)
	return all
}

type occurrence struct {
	File     string `json:"file"`
	BlockID  uint8  `json:"block_id"`
	Offset   int    `json:"offset"`
	Encoding string `json:"encoding"`
	Readable bool   `json:"readable"`
	Text     string `json:"text"`
}

type termResult struct {
	English     string       `json:"english"`
	Settled     string       `json:"settled"`
	Occurrences int          `json:"occurrences"`
	Readable    int          `json:"readable_occurrences"`
	Samples     []occurrence `json:"samples,omitempty"`
}

// readable 判斷一段解出來的文字是不是真的文字。6-bit 解碼會把任何位元組都解成
// 某個字元，所以圖形與程式碼片段也會產生「有字母、有空白」的字串；沒有這道
// 篩子，專名的命中數會被垃圾灌大。
//
// 判準刻意寬鬆：字母與空白佔七成以上，而且至少有兩個長度 2 以上的詞。
// 篩掉的並不丟棄，只是不計入 readable，讓兩個數字的差距看得見。
func readable(text string) bool {
	letters := 0
	for _, r := range text {
		if r == ' ' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			letters++
		}
	}
	if letters*10 < len(text)*7 {
		return false
	}
	// 6-bit 解碼會把圖形與程式碼也解成有字母有空白的字串，光看比例擋不住。
	// 英文散文幾乎一定帶著虛詞，垃圾幾乎一定不帶，所以以虛詞當判準。
	common := map[string]bool{
		"THE": true, "YOU": true, "AND": true, "OF": true, "TO": true,
		"IS": true, "ARE": true, "A": true, "IN": true, "THIS": true,
		"THAT": true, "IT": true, "FOR": true, "WITH": true, "HAVE": true,
	}
	seen := 0
	for _, word := range strings.Fields(strings.ToUpper(text)) {
		word = strings.Trim(word, ".,!?'\"();:")
		if common[word] {
			seen++
		}
	}
	return seen >= 2
}

// containsWord 以詞界比對，免得 Lis 命中 LISTEN、Thief 命中 THIEVES 之外的東西。
func containsWord(haystack, needle string) bool {
	boundary := func(r byte) bool {
		return !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}
	for index := 0; ; {
		found := strings.Index(haystack[index:], needle)
		if found < 0 {
			return false
		}
		start := index + found
		end := start + len(needle)
		before := start == 0 || boundary(haystack[start-1])
		after := end == len(haystack) || boundary(haystack[end])
		if before && after {
			return true
		}
		index = start + 1
	}
}

type report struct {
	ZIP           string       `json:"zip"`
	Archives      int          `json:"archives"`
	Blocks        int          `json:"blocks"`
	Strings       int          `json:"strings"`
	PlainStrings  int          `json:"plain_strings"`
	PackedStrings int          `json:"packed_strings"`
	MinimumRun    int          `json:"minimum_run"`
	SampleLimit   int          `json:"sample_limit"`
	Terms         []termResult `json:"terms"`
	MissingTerms  []string     `json:"missing_terms"`
	ScannedFormat string       `json:"scanned_format"`
}

const sampleLimit = 6

func main() {
	zipPath := flag.String("zip", "Pool of Radiance (1988).zip", "DOS source ZIP")
	flag.Parse()
	result, err := audit(*zipPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// printableRuns 取出一段位元組裡所有長度足夠的可列印片段，連同它們的位移。
// 原版把文字直接放在 block 裡，長度前綴與控制位元組會自然把段落切開。
func printableRuns(data []byte, minimum int) []occurrence {
	runs := make([]occurrence, 0, 16)
	start := -1
	for index := 0; index <= len(data); index++ {
		printable := index < len(data) && data[index] >= 0x20 && data[index] < 0x7F
		if printable {
			if start < 0 {
				start = index
			}
			continue
		}
		if start >= 0 && index-start >= minimum {
			runs = append(runs, occurrence{Offset: start, Encoding: "plain", Text: string(data[start:index])})
		}
		start = -1
	}
	return runs
}

func audit(zipPath string) (report, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report{}, err
	}
	defer archive.Close()

	result := report{
		ZIP: filepath.Base(zipPath), MinimumRun: minimumRun, SampleLimit: sampleLimit,
		ScannedFormat: "every DAX block in the ZIP: printable ASCII runs and 0x80-prefixed 6-bit packed text",
	}
	found := make(map[string][]occurrence, len(terms))

	names := make([]string, 0, len(archive.File))
	for _, member := range archive.File {
		names = append(names, member.Name)
	}
	sort.Strings(names)
	members := make(map[string]*zip.File, len(archive.File))
	for _, member := range archive.File {
		members[member.Name] = member
	}

	for _, name := range names {
		if !strings.EqualFold(filepath.Ext(name), ".dax") {
			continue
		}
		member := members[name]
		stream, err := member.Open()
		if err != nil {
			return report{}, err
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, 64<<20))
		closeErr := stream.Close()
		if readErr != nil {
			return report{}, readErr
		}
		if closeErr != nil {
			return report{}, closeErr
		}
		blocks, err := dax.Parse(data)
		if err != nil {
			return report{}, fmt.Errorf("%s: %w", name, err)
		}
		result.Archives++
		for _, block := range blocks {
			result.Blocks++
			runs := printableRuns(block.Data, minimumRun)
			result.PlainStrings += len(runs)
			for _, candidate := range ecl.FindPackedTextCandidatesAt(block.Data) {
				runs = append(runs, occurrence{
					Offset: candidate.Offset, Encoding: "packed6", Text: candidate.Text,
				})
				result.PackedStrings++
			}
			for _, run := range runs {
				result.Strings++
				lowered := strings.ToLower(run.Text)
				isReadable := run.Encoding == "plain" || readable(run.Text)
				for _, term := range terms {
					matched := false
					for _, spelling := range term.spellings() {
						if containsWord(lowered, strings.ToLower(spelling)) {
							matched = true
							break
						}
					}
					if !matched {
						continue
					}
					found[term.English] = append(found[term.English], occurrence{
						File: filepath.Base(name), BlockID: block.Entry.ID,
						Offset: run.Offset, Encoding: run.Encoding,
						Readable: isReadable, Text: run.Text,
					})
				}
			}
		}
	}

	for _, term := range terms {
		hits := found[term.English]
		row := termResult{English: term.English, Settled: term.Settled, Occurrences: len(hits)}
		samples := make([]occurrence, 0, sampleLimit)
		for _, hit := range hits {
			if !hit.Readable {
				continue
			}
			row.Readable++
			if len(samples) < sampleLimit {
				samples = append(samples, hit)
			}
		}
		row.Samples = samples
		result.Terms = append(result.Terms, row)
		if row.Readable == 0 {
			result.MissingTerms = append(result.MissingTerms, term.English)
		}
	}
	return result, nil
}
