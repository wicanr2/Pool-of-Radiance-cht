package journal_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/journal"
)

// 每一頁的標題長這樣：`## p.12 · 標題（Pic0011-left）`。副標題可有可無，
// 括號裡那一段是它對應的掃描檔與左右半頁。
var manualPageHeading = regexp.MustCompile(`(?m)^## p\.(\d+)[^（\n]*（(Pic\d{4})-(left|right)）`)

// scanRoot 是掃描原檔的位置。它在 `workplace/` 底下，不進版控，所以這一組
// 測試在沒有掃描的環境會整批 Skip。
const scanRoot = "珍009-光芒之池"

func manualScanRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "workplace", "manual-scan", scanRoot)
	if _, err := os.Stat(root); err != nil {
		t.Skipf("手冊掃描不在版控裡：%v", err)
	}
	return root
}

type scanHalf struct{ file, side string }

// manualBooks 是兩冊轉錄與它們的第一頁頁碼。
var manualBooks = []struct {
	file  string
	first int
}{
	{"journal-vol1.md", 1},
	{"manual-vol2.md", 1},
}

func manualBookPath(file string) string {
	return filepath.Join("..", "..", "docs", "reference", "manual", file)
}

// claimScanHalves 掃過兩冊轉錄，回傳「哪一個掃描半頁被哪一頁認領」。
func claimScanHalves(t *testing.T, root string) map[scanHalf]string {
	t.Helper()
	claimed := map[scanHalf]string{}
	for _, book := range manualBooks {
		raw, err := os.ReadFile(manualBookPath(book.file))
		if err != nil {
			t.Fatal(err)
		}
		matches := manualPageHeading.FindAllStringSubmatch(string(raw), -1)
		if len(matches) == 0 {
			t.Fatalf("%s 一個頁標題都沒有；格式是不是改了？", book.file)
		}
		pages := make([]int, 0, len(matches))
		for _, match := range matches {
			number, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			pages = append(pages, number)
			if _, err := os.Stat(filepath.Join(root, match[2]+".jpg")); err != nil {
				t.Errorf("%s 的 p.%d 指到 %s，但掃描不存在", book.file, number, match[2])
			}
			key := scanHalf{file: match[2], side: match[3]}
			if owner, taken := claimed[key]; taken {
				t.Errorf("%s 的 p.%d 與 %s 都認領了 %s-%s",
					book.file, number, owner, key.file, key.side)
				continue
			}
			claimed[key] = fmt.Sprintf("%s p.%d", book.file, number)
		}
		sort.Ints(pages)
		if pages[0] != book.first {
			t.Errorf("%s 從 p.%d 起算，應該是 p.%d", book.file, pages[0], book.first)
		}
		for index := 1; index < len(pages); index++ {
			if pages[index] == pages[index-1] {
				t.Errorf("%s 的 p.%d 出現兩次", book.file, pages[index])
				continue
			}
			if pages[index] != pages[index-1]+1 {
				t.Errorf("%s 的頁碼從 p.%d 跳到 p.%d", book.file, pages[index-1], pages[index])
			}
		}
		t.Logf("%s：p.%d..p.%d 共 %d 頁，連續", book.file, pages[0], pages[len(pages)-1], len(pages))
	}
	return claimed
}

// 兩冊的轉錄不能漏頁。
//
// 逐字回對整整兩冊要人來做，但「整頁漏掉」這一類**機械檢查得出來**：
// 每一頁的標題都帶著它是哪一張掃描的哪一半，所以頁碼要連續、掃描檔要存在、
// 而且同一張掃描的同一半不能被兩頁認領。漏一頁的症狀是頁碼跳號，
// 抄錯來源的症狀是同一半被認領兩次——兩種在正文裡都看不出來。
func TestManualTranscriptionCoversEveryScannedPage(t *testing.T) {
	claimScanHalves(t, manualScanRoot(t))
}

// nonContentHalves 是**故意**沒有轉錄的掃描半頁，每一條都要說得出是什麼。
//
// 這份清單存在的理由是反向：正向檢查只證明「轉錄到的頁連續」，它對
// 「整整一章從來沒被轉錄過」沒有意見——那一章的掃描只會安靜地留在這裡沒人
// 認領。把不是內文的半頁逐一列出來之後，剩下的差集必須是空的，於是日後
// 補進新掃描、或某一段轉錄被刪掉，都會在這裡失敗。
//
// 認定依據是把 25 張未認領的半頁併成一張對照圖逐格看過，不是憑檔名推的。
var nonContentHalves = map[scanHalf]string{
	{"Pic0001", "left"}:  "外盒與封面翻拍",
	{"Pic0001", "right"}: "外盒與封面翻拍",
	{"Pic0002", "left"}:  "外盒背面翻拍（軟體世界代理說明、系統需求）",
	{"Pic0002", "right"}: "外盒背面翻拍（軟體世界代理說明、系統需求）",
	{"Pic0003", "left"}:  "外盒與封面翻拍",
	{"Pic0003", "right"}: "外盒與封面翻拍",

	{"Pic0004", "left"}:  "上冊封面內頁，空白",
	{"Pic0004", "right"}: "原版英文廣告頁「FREE NEW PHLAN!」，含 The Civilized Area of New Phlan 地圖；中文版原樣保留、不編頁碼",
	{"Pic0005", "left"}:  "上冊目錄（由 TestManualTranscriptionMatchesThePrintedTableOfContents 當結構來源）",

	{"Pic0032", "right"}: "上冊 p.54 之後的空白頁",
	{"Pic0033", "left"}:  "上冊末的軟體世界其他遊戲廣告（威探闖通關）",
	{"Pic0033", "right"}: "上冊封面翻拍",
	{"Pic0034", "left"}:  "下冊封面翻拍",
	{"Pic0034", "right"}: "下冊封面翻拍",
	{"Pic0035", "left"}:  "下冊目錄（由 TestManualTranscriptionMatchesThePrintedTableOfContents 當結構來源）",

	{"Pic0064", "left"}:  "下冊末的《軟體世界》雜誌試刊號廣告與封底",
	{"Pic0064", "right"}: "下冊末的《軟體世界》雜誌試刊號廣告與封底",
	{"Pic0065", "left"}:  "Master Disk A 標籤連同外盒的翻拍",
	{"Pic0065", "right"}: "Master Disk A 標籤連同外盒的翻拍",
	{"Pic0066", "left"}:  "Master Disk B 標籤連同外盒的翻拍",
	{"Pic0066", "right"}: "Master Disk B 標籤連同外盒的翻拍",
	{"Pic0067", "left"}:  "Master Disk C 標籤連同外盒的翻拍",
	{"Pic0067", "right"}: "Master Disk C 標籤連同外盒的翻拍",
	{"Pic0068", "left"}:  "譯碼轉盤（Translation Wheel）翻拍",
	{"Pic0068", "right"}: "譯碼轉盤（Translation Wheel）翻拍",
}

// 反過來：每一張掃描的每一半，不是被某一頁認領，就是列在 nonContentHalves 裡。
func TestEveryScannedHalfIsEitherTranscribedOrExplained(t *testing.T) {
	root := manualScanRoot(t)
	claimed := claimScanHalves(t, root)
	scans, err := filepath.Glob(filepath.Join(root, "Pic*.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scans) == 0 {
		t.Fatal("掃描目錄裡一張 Pic*.jpg 都沒有")
	}
	sort.Strings(scans)
	var unexplained []string
	seen := map[scanHalf]bool{}
	for _, scan := range scans {
		name := strings.TrimSuffix(filepath.Base(scan), ".jpg")
		for _, side := range []string{"left", "right"} {
			key := scanHalf{file: name, side: side}
			seen[key] = true
			if _, taken := claimed[key]; taken {
				continue
			}
			if _, allowed := nonContentHalves[key]; allowed {
				continue
			}
			unexplained = append(unexplained, name+"-"+side)
		}
	}
	if len(unexplained) != 0 {
		t.Errorf("%d 個掃描半頁既沒有轉錄也沒有列在 nonContentHalves：%v",
			len(unexplained), unexplained)
	}
	// 清單也不能有殘留：掃描換掉或轉錄補上之後，過期的條目要一起清掉，
	// 否則它會繼續替一個不存在的半頁背書。
	var stale []string
	for key, why := range nonContentHalves {
		label := key.file + "-" + key.side
		if !seen[key] {
			stale = append(stale, label+"（掃描不存在）")
			continue
		}
		if owner, taken := claimed[key]; taken {
			stale = append(stale, fmt.Sprintf("%s（已由 %s 轉錄，卻仍標成「%s」）", label, owner, why))
		}
	}
	sort.Strings(stale)
	if len(stale) != 0 {
		t.Errorf("nonContentHalves 有 %d 條過期：%v", len(stale), stale)
	}
	t.Logf("掃描 %d 張共 %d 半頁：轉錄 %d、非內文 %d",
		len(scans), len(scans)*2, len(claimed), len(nonContentHalves))
}

// tableOfContentsEntry 是印在目錄頁上的一條：標題與它標的頁碼。
type tableOfContentsEntry struct {
	label string
	page  int
}

// printedTableOfContents 照抄兩冊目錄頁上的字（`Pic0005-left`、`Pic0035-left`）。
// 標題只取前綴——目錄用的破折號與正文不一定同一個字（目錄寫
// 「古老的城市—古菲蘭城」，正文寫「－」），比對到分得出章節就夠了。
var printedTableOfContents = map[string][]tableOfContentsEntry{
	"journal-vol1.md": {
		{"第一章", 1},
		{"第二章", 3},
		{"一、前言", 3},
		{"二、地理位置", 3},
		{"三、古老的城市", 4},
		{"四、菲蘭城的重建", 5},
		{"五、菲蘭城的陷落", 9},
		{"六、菲蘭城的再現", 10},
		{"七、今日的菲蘭城", 12},
		{"第三章", 13},
		{"第四章", 18},
		{"第五章", 22},
		{"第六章", 45},
		{"附錄", 48},
	},
	"manual-vol2.md": {
		{"第一章", 1},
		{"第二章", 8},
		{"第三章", 10},
		{"第四章", 19},
		{"第五章", 36},
		{"第六章", 46},
	},
}

// headingsByPage 把一冊轉錄拆成「這個標題在第幾頁」，順序照原檔。
// 章名有時候只寫在頁標題的副標上（`## p.13 · 第三章 …`），有時候另外再起一個
// `###`，所以兩種都算。
func headingsByPage(t *testing.T, file string) []tableOfContentsEntry {
	t.Helper()
	raw, err := os.ReadFile(manualBookPath(file))
	if err != nil {
		t.Fatal(err)
	}
	pageLine := regexp.MustCompile(`^## p\.(\d+)`)
	var headings []tableOfContentsEntry
	page := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		if match := pageLine.FindStringSubmatch(line); match != nil {
			number, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			page = number
		}
		if page == 0 {
			continue
		}
		headings = append(headings, tableOfContentsEntry{label: line, page: page})
	}
	return headings
}

// 轉錄的章節要落在印刷目錄標的那一頁。
//
// 頁碼連續只證明沒有整頁掉出去，它擋不住「頁碼對、內容卻抄到隔壁頁」或
// 「某一章從頭到尾錯位一頁」。兩冊自己印了目錄，那份目錄是原書給的檢查點：
// 章節起始頁對不上，就是轉錄的分頁錯了。
func TestManualTranscriptionMatchesThePrintedTableOfContents(t *testing.T) {
	for _, book := range manualBooks {
		entries := printedTableOfContents[book.file]
		if len(entries) == 0 {
			t.Fatalf("%s 沒有登記印刷目錄", book.file)
		}
		headings := headingsByPage(t, book.file)
		if len(headings) == 0 {
			t.Fatalf("%s 讀不到任何標題", book.file)
		}
		matched := 0
		for _, entry := range entries {
			found := false
			for _, heading := range headings {
				if !strings.Contains(heading.label, entry.label) {
					continue
				}
				found = true
				if heading.page != entry.page {
					t.Errorf("%s：目錄說「%s」在 p.%d，轉錄第一次出現在 p.%d（%s）",
						book.file, entry.label, entry.page, heading.page, strings.TrimSpace(heading.label))
				} else {
					matched++
				}
				break
			}
			if !found {
				t.Errorf("%s：目錄有「%s」，轉錄裡找不到這個標題", book.file, entry.label)
			}
		}
		t.Logf("%s：印刷目錄 %d 條，對上 %d 條", book.file, len(entries), matched)
	}
}

// numberedSeries 是上冊裡「原書自己編了號」的四組。數量與連續性都寫死：
// 抄漏一則的症狀是序號跳掉，抄重一則的症狀是同一個號碼出現兩次，而整組
// 少了尾巴（例如 OCR 只到 p.44）在正文裡完全看不出來——最後一則照樣讀得通。
var numberedSeries = []struct {
	name    string
	pattern *regexp.Regexp
	count   int
}{
	{"第五章 探險者線索提示", regexp.MustCompile(`(?m)^### 線索報導 (\d+)\s*$`), 58},
	{"第六章 酒店傳言", regexp.MustCompile(`(?m)^\*\*傳言 (\d+)\*\*`), 23},
	{"附錄", regexp.MustCompile(`(?m)^#### (\d+)\. `), 7},
}

// 上冊四組編號要完整、連續、不重複。
func TestJournalNumberedSeriesAreCompleteAndInOrder(t *testing.T) {
	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, series := range numberedSeries {
		matches := series.pattern.FindAllStringSubmatch(text, -1)
		numbers := make([]int, 0, len(matches))
		for _, match := range matches {
			number, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			numbers = append(numbers, number)
		}
		if len(numbers) != series.count {
			t.Errorf("%s 抓到 %d 則，原書是 %d 則", series.name, len(numbers), series.count)
		}
		for index, number := range numbers {
			if number != index+1 {
				t.Errorf("%s 的第 %d 則編號是 %d，應該是 %d（順序或編號跳了）",
					series.name, index+1, number, index+1)
				break
			}
		}
		t.Logf("%s：1..%d 共 %d 則，連續且照順序", series.name, len(numbers), len(numbers))
	}
}

var councilNotice = regexp.MustCompile(`(?m)^\*\*公告字號 ([IVXLCDM]+)\*\*`)

// romanValue 把公告字號的羅馬數字換成整數。原書只用到 CCXIV，減法規則
// （IV、IX、XL…）照標準處理就夠。
func romanValue(roman string) int {
	digits := map[byte]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000}
	total := 0
	for index := 0; index < len(roman); index++ {
		value := digits[roman[index]]
		if index+1 < len(roman) && digits[roman[index+1]] > value {
			total -= value
			continue
		}
		total += value
	}
	return total
}

// 議會公告不是連號的（原書就跳號：LIX 之後是 LXIV），所以連續性檢查不適用；
// 能查的是「則數對」與「字號遞增」。抄錯一個羅馬數字最容易出現的樣子就是
// 順序倒過來——CXC 打成 CXL 會讓它掉到前一則之前。
func TestCouncilNoticesAreCompleteAndAscending(t *testing.T) {
	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	matches := councilNotice.FindAllStringSubmatch(string(raw), -1)
	if len(matches) != 18 {
		t.Errorf("公告抓到 %d 則，原書是 18 則", len(matches))
	}
	previous := 0
	for index, match := range matches {
		value := romanValue(match[1])
		if value <= previous {
			t.Errorf("第 %d 則公告字號 %s（%d）沒有比前一則（%d）大", index+1, match[1], value, previous)
		}
		previous = value
	}
	if len(matches) != 0 {
		t.Logf("議會公告 %d 則，字號 %s..%s 遞增",
			len(matches), matches[0][1], matches[len(matches)-1][1])
	}
}

// entryMarkers 是三種條目在轉錄裡的起頭。線索用 `### 標題`，傳言與公告
// 用粗體行——原書就是這樣排的，轉錄照排。
var entryMarkers = map[journal.Kind]*regexp.Regexp{
	journal.Clue:         regexp.MustCompile(`(?m)^### 線索報導 (\d+)\s*$`),
	journal.Rumour:       regexp.MustCompile(`(?m)^\*\*傳言 (\d+)\*\*：?`),
	journal.Proclamation: regexp.MustCompile(`(?m)^\*\*公告字號 ([IVXLCDM]+)\*\*`),
}

// entryStop 是一條的結束：下一個 `###`／`####` 標題，或下一條的起頭。
// **換頁（`## p.N`）不算**——原書的條目本來就會跨頁，把換頁當結束會讓
// 每一條跨頁的條目都少掉後半段。
var entryStop = regexp.MustCompile(`(?m)^(?:#{3,4} |\*\*傳言 \d+\*\*|\*\*公告字號 [IVXLCDM]+\*\*)`)

var pageHeadingLine = regexp.MustCompile(`(?m)^## p\.\d+.*$`)
var translatorNote = regexp.MustCompile(`〔[^〕]*〕`)

// compareText 把兩邊都壓成「只剩字」再比。轉錄照原書的行寬硬斷行，語料是
// 給遊戲排版用的，所以換行位置本來就不同；譯註〔…〕只在轉錄那一邊。
func compareText(s string) string {
	s = pageHeadingLine.ReplaceAllString(s, "")
	s = translatorNote.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '*', '>':
			return -1
		}
		return r
	}, s)
}

type transcribedEntry struct {
	text string
	page int
	pic  string
}

// transcribedEntries 從上冊轉錄切出三章的條目，連同它落在哪一頁。
func transcribedEntries(t *testing.T) map[journal.Kind]map[string]transcribedEntry {
	t.Helper()
	raw, err := os.ReadFile(manualBookPath("journal-vol1.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)

	type pageAt struct {
		start  int
		number int
		pic    string
	}
	var pages []pageAt
	for _, span := range manualPageHeading.FindAllStringSubmatchIndex(text, -1) {
		number, err := strconv.Atoi(text[span[2]:span[3]])
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, pageAt{
			start:  span[0],
			number: number,
			pic:    text[span[4]:span[5]] + "-" + text[span[6]:span[7]],
		})
	}
	pageOf := func(offset int) pageAt {
		found := pageAt{}
		for _, page := range pages {
			if page.start > offset {
				break
			}
			found = page
		}
		return found
	}

	out := map[journal.Kind]map[string]transcribedEntry{}
	for kind, marker := range entryMarkers {
		out[kind] = map[string]transcribedEntry{}
		for _, span := range marker.FindAllStringSubmatchIndex(text, -1) {
			id := text[span[2]:span[3]]
			end := len(text)
			if stop := entryStop.FindStringIndex(text[span[1]:]); stop != nil {
				end = span[1] + stop[0]
			}
			page := pageOf(span[0])
			out[kind][id] = transcribedEntry{
				text: compareText(text[span[1]:end]),
				page: page.number,
				pic:  page.pic,
			}
		}
	}
	return out
}

// 轉錄與遊戲內建語料必須是同一段字。
//
// 兩份東西各自都通過了「則數對、編號連續」，卻仍然可能互相抄錯一個字——
// 而且改對其中一份之後很容易忘記另一份。這一則就是為了那個情況：
// `docs/reference/manual/journal-vol1.md` 給人讀，`internal/journal/zh-TW.json`
// 給遊戲查，差一個字只有玩家在遊戲裡翻到那一條時才看得到。
func TestTranscriptionAndGameCorpusAgree(t *testing.T) {
	corpus, err := journal.TraditionalChinese()
	if err != nil {
		t.Fatal(err)
	}
	transcribed := transcribedEntries(t)
	checked := 0
	for _, kind := range journal.Kinds {
		for _, entry := range corpus.Entries(kind) {
			source, ok := transcribed[kind][entry.ID]
			if !ok {
				t.Errorf("語料有 %s %s，轉錄裡找不到", kind, entry.ID)
				continue
			}
			agrees := true
			if want := compareText(entry.Text); source.text != want {
				t.Errorf("%s %s 兩份不一樣：%s", kind, entry.ID, firstDifference(source.text, want))
				agrees = false
			}
			if strconv.Itoa(source.page) != entry.Page || source.pic != entry.Pic {
				t.Errorf("%s %s 語料標 p.%s（%s），轉錄放在 p.%d（%s）",
					kind, entry.ID, entry.Page, entry.Pic, source.page, source.pic)
				agrees = false
			}
			if agrees {
				checked++
			}
		}
		// 反過來：轉錄有、語料沒有的條目，遊戲永遠翻不到。
		for id := range transcribed[kind] {
			if _, ok := corpus.Lookup(kind, id); !ok {
				t.Errorf("轉錄有 %s %s，語料裡沒有；遊戲翻不到這一條", kind, id)
			}
		}
	}
	t.Logf("轉錄與語料逐字相同：%d／%d 條", checked, corpus.Size())
}

// firstDifference 指出兩份抄本第一個岔開的位置，附前後文——差一個字的時候，
// 光說「不一樣」找不到是哪個字。
func firstDifference(got, want string) string {
	gotRunes, wantRunes := []rune(got), []rune(want)
	for index := 0; index < len(gotRunes) && index < len(wantRunes); index++ {
		if gotRunes[index] == wantRunes[index] {
			continue
		}
		from := index - 12
		if from < 0 {
			from = 0
		}
		return fmt.Sprintf("第 %d 字起，轉錄作「%s」，語料作「%s」", index,
			string(gotRunes[from:min(index+12, len(gotRunes))]),
			string(wantRunes[from:min(index+12, len(wantRunes))]))
	}
	if len(gotRunes) == len(wantRunes) {
		return "長度相同卻比不出差異"
	}
	if len(gotRunes) > len(wantRunes) {
		return fmt.Sprintf("轉錄多出 %d 字：「%s」", len(gotRunes)-len(wantRunes),
			string(gotRunes[len(wantRunes):min(len(wantRunes)+30, len(gotRunes))]))
	}
	return fmt.Sprintf("語料多出 %d 字：「%s」", len(wantRunes)-len(gotRunes),
		string(wantRunes[len(gotRunes):min(len(gotRunes)+30, len(wantRunes))]))
}
