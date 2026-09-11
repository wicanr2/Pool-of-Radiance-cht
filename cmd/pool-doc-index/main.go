// pool-doc-index 產生 docs/spec/000-index.md：每份規格的狀態、實作它的檔案、
// 釘住它的測試，以及 cmd/ 底下每一支工具在做什麼。
//
// **這份索引是算出來的，不是寫出來的。** 對應關係的主鍵是 spec 編號——程式碼
// 註解裡的 `spec NNN` 就是那條線，本工具只是把它反過來收攏。所以索引不會過期：
// 改了註解重跑一次就對了，不需要有人記得同步。手改 000-index.md 會在下一次
// 執行時被蓋掉。
//
// 同時寫一份 docs/audit/doc-index.json 給機器讀：缺口在那裡是清單不是數字，
// 因為要看得出是哪幾份。worklist 的 json_gap 綁它的長度。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	specReference = regexp.MustCompile(`(?i)spec\s+(\d{3})`)
	specFileName  = regexp.MustCompile(`^(\d{3})-(.+)\.md$`)
	// 第二個判準：檔案裡真的有 go test 會跑的函式。
	// 和「檔名是 _test.go」問的是同一件事，但資訊來源不同——
	// 一個看檔名，一個看內容，所以同一個錯誤不容易讓兩邊一起錯。
	testFunction = regexp.MustCompile(`(?m)^func (Test|Fuzz)[A-Z_]`)
	statusWords  = []string{"CONFORMED", "READY", "DRAFT", "OPEN"}
)

type spec struct {
	number   string
	title    string
	file     string
	statuses []string
	code     []string
	tests    []string
	// sharedEngine 標記「實作在共用 engine，不在這個 repo」。少了它，
	// 那幾份的實作欄會是「—」，看起來像沒實作——而索引掃不到別的 repo。
	sharedEngine bool
	// outsideGo 是規格自己寫的那一行 `實作：…`，用在**實作根本不是 Go** 的
	// 那幾份：發行包是 shell 腳本、送鍵規則給的是 dosgolem。這一類永遠掃不到
	// `spec NNN` 的反向引用，混在「還沒接」裡就變成永遠清不掉的雜訊。
	//
	// 這一行是**規格作者手寫的斷言**，不是猜的：關鍵字命中不算數（那正是
	// spec 124 的教訓），所以只認這個明確的形狀。
	outsideGo string
}

// gapReport 是給機器讀的那一份：缺口不是數字而是清單，因為要看得出是哪幾份。
// worklist 的 verify 綁這份 JSON 的長度——註解會被順手改掉，項數不會。
// crossCheck 是這份報告自己的體檢結果。
//
// 缺口清單的問題在於「算錯」和「真的沒缺口」在數字上長得一模一樣——下游只看
// 得到 0，看不到那個 0 是怎麼來的。所以同一個問題用兩個資訊來源各算一次，
// 不一致就記在這裡，並且 passed 轉 false。讀這份 JSON 的人要先看它。
type crossCheck struct {
	Passed     bool     `json:"passed"`
	Criteria   string   `json:"criteria"`
	Mismatches []string `json:"mismatches"`
}

type gapReport struct {
	Schema                     string     `json:"schema"`
	CrossCheck                 crossCheck `json:"cross_check"`
	SpecCount                  int        `json:"spec_count"`
	ToolCount                  int        `json:"tool_count"`
	SpecsWithoutImplementation []string   `json:"specs_without_implementation"`
	SpecsWithoutTests          []string   `json:"specs_without_tests"`
	SpecsInSharedEngine        []string   `json:"specs_in_shared_engine"`
	// SpecsOutsideGo 是實作不是 Go 的那幾份（發行腳本、送鍵規則）。
	// 它們永遠掃不到 `spec NNN`，混在 without_implementation 裡就是永遠
	// 清不掉的雜訊，會讓那個數字失去「還剩多少沒接」的意思。
	SpecsOutsideGo             []string   `json:"specs_implemented_outside_go"`
	ToolsWithoutDoc            []string   `json:"tools_without_doc"`
	ToolsWithoutTests          []string   `json:"tools_without_tests"`
}

type tool struct {
	name    string
	summary string
	// 兩個判準各記各的，不要在這裡就合併——合併掉就看不出它們何時不一致了。
	hasTests     bool // 判準 A：檔名是 *_test.go
	hasTestFuncs bool // 判準 B：內容有 func TestXxx
	specs        []string
}

// readSpec 讀檔頭：第一行是標題，「狀態：」到「日期：」之間是狀態段。
// 135 份規格全部照這個格式寫，所以不必猜。
func readSpec(path string) (spec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return spec{}, err
	}
	name := filepath.Base(path)
	match := specFileName.FindStringSubmatch(name)
	if match == nil {
		return spec{}, fmt.Errorf("檔名不是 NNN-描述.md：%s", name)
	}

	lines := strings.Split(string(raw), "\n")
	result := spec{number: match[1], file: name}
	// eclvm 是共用 engine 的套件名，比「共用 engine」這個詞更明確——
	// 後者在很多規格裡只是敘述的一部分。
	result.sharedEngine = strings.Contains(string(raw), "eclvm")
	for _, line := range lines {
		note, ok := strings.CutPrefix(strings.TrimSpace(line), "實作：")
		if !ok {
			continue
		}
		// 只取第一句：這一行會整個塞進索引表的一格，寫長了表格就散了。
		// 完整的理由留在規格自己那裡，索引只指路。
		if head, _, found := strings.Cut(note, "。"); found {
			note = head + "。"
		}
		result.outsideGo = strings.TrimSpace(note)
		break
	}
	if len(lines) > 0 {
		result.title = strings.TrimSpace(strings.TrimPrefix(lines[0], "#"))
		result.title = strings.TrimPrefix(result.title, "Spec "+result.number+"：")
		result.title = strings.TrimPrefix(result.title, "Spec "+result.number+":")
		result.title = strings.TrimSpace(result.title)
	}

	// 狀態段可能跨好幾行（一份規格常常一半 CONFORMED 一半 DRAFT）。
	var block strings.Builder
	collecting := false
	for _, line := range lines[:min(len(lines), 16)] {
		if strings.HasPrefix(line, "狀態：") {
			collecting = true
		}
		if collecting {
			if strings.HasPrefix(line, "日期：") {
				break
			}
			block.WriteString(line)
		}
	}
	for _, word := range statusWords {
		if strings.Contains(block.String(), word) {
			result.statuses = append(result.statuses, word)
		}
	}
	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// collectReferences 走過原始碼，把每個檔案提到的 spec 編號收起來。
func collectReferences(roots []string) (map[string][]string, map[string][]string, error) {
	code := map[string][]string{}
	tests := map[string][]string{}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			seen := map[string]bool{}
			for _, match := range specReference.FindAllStringSubmatch(string(raw), -1) {
				number := match[1]
				if seen[number] {
					continue
				}
				seen[number] = true
				if strings.HasSuffix(path, "_test.go") {
					tests[number] = append(tests[number], path)
				} else {
					code[number] = append(code[number], path)
				}
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	return code, tests, nil
}

// readTools 讀 cmd/ 每一支的 package doc 第一行，當作「這支在做什麼」。
func readTools(root string) ([]tool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var tools []tool
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		current := tool{name: entry.Name()}
		files, err := os.ReadDir(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".go") {
				continue
			}
			if strings.HasSuffix(file.Name(), "_test.go") {
				current.hasTests = true
			}
			raw, err := os.ReadFile(filepath.Join(root, entry.Name(), file.Name()))
			if err != nil {
				return nil, err
			}
			if testFunction.Match(raw) {
				current.hasTestFuncs = true
			}
			for _, match := range specReference.FindAllStringSubmatch(string(raw), -1) {
				if !seen[match[1]] {
					seen[match[1]] = true
					current.specs = append(current.specs, match[1])
				}
			}
			if current.summary == "" && file.Name() == "main.go" {
				current.summary = firstDocLine(string(raw), entry.Name())
			}
		}
		sort.Strings(current.specs)
		tools = append(tools, current)
	}
	return tools, nil
}

// firstDocLine 抓 package doc 的第一句。工具的檔頭第一行就是它的用途。
//
// 一句話常常跨兩三行，所以要接到句號為止；中文行直接接，英文行之間補空格。
// 開頭的工具名要去掉——第一欄已經是工具名了，再寫一次只是把表格撐寬。
func firstDocLine(source, name string) string {
	var sentence string
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "//") {
			break
		}
		text := strings.TrimSpace(strings.TrimPrefix(line, "//"))
		if text == "" {
			if sentence != "" {
				break
			}
			continue
		}
		if needsSpace(sentence, text) {
			sentence += " "
		}
		sentence += text
		if strings.Contains(sentence, "。") || strings.Contains(sentence, ". ") ||
			strings.HasSuffix(sentence, ".") {
			break
		}
	}
	sentence = strings.TrimPrefix(sentence, "Command ")
	sentence = strings.TrimSpace(strings.TrimPrefix(sentence, name))
	sentence = strings.TrimPrefix(sentence, "是")
	// 只留第一句——英文的 package doc 常常一段三句，整段塞進表格會撐破版面。
	if index := strings.Index(sentence, "。"); index >= 0 {
		sentence = sentence[:index]
	}
	if index := strings.Index(sentence, ". "); index >= 0 {
		sentence = sentence[:index]
	}
	sentence = strings.TrimSuffix(strings.TrimSpace(sentence), ".")
	return strings.TrimSpace(sentence)
}

// needsSpace 看接縫兩側的那一個字，不是看整行有沒有中文——
// 「engine 和 game pack 的接線」整行有中文，但它接在中文後面時開頭仍要空格。
func needsSpace(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	// 只有兩側都是中文才不補空格。中文換行接英文（「共用」＋「engine」）
	// 在原文裡是一個空格，接起來不補就變成「共用engine」。
	return !(isHan(leftRunes[len(leftRunes)-1]) && isHan(rightRunes[0]))
}

// isHan 把漢字與全形標點都算進來——行末常常是「：」或「，」。
func isHan(letter rune) bool {
	switch {
	case letter >= 0x3000 && letter <= 0x9FFF:
		return true
	case letter >= 0xFF00 && letter <= 0xFFEF:
		return true
	}
	return false
}

func shorten(paths []string) string {
	if len(paths) == 0 {
		return "—"
	}
	sort.Strings(paths)
	trimmed := make([]string, 0, len(paths))
	for _, path := range paths {
		trimmed = append(trimmed, "`"+path+"`")
	}
	if len(trimmed) > 3 {
		return strings.Join(trimmed[:3], "、") + fmt.Sprintf(" 等 %d 個", len(trimmed))
	}
	return strings.Join(trimmed, "、")
}

// crossCheckTools 讓兩個判準對每一支工具各答一次。
//
// 不一致就是這份報告不可信的證據，兩個方向都是真的問題：檔名對但裡面沒有
// func Test（空的測試檔），或者有 func Test 卻不在 _test.go 裡（go test 根本
// 不會跑它）。單一判準看不出這兩種，因為它們在數字上和「真的有測試」一樣。
func crossCheckTools(tools []tool) crossCheck {
	check := crossCheck{
		Passed:     true,
		Criteria:   "檔名是 *_test.go｜內容有 func TestXxx",
		Mismatches: []string{},
	}
	for _, item := range tools {
		if item.hasTests == item.hasTestFuncs {
			continue
		}
		reason := "有 func Test 但檔名不是 *_test.go，go test 不會跑它"
		if item.hasTests {
			reason = "有 *_test.go 但裡面沒有 func Test"
		}
		check.Passed = false
		check.Mismatches = append(check.Mismatches, item.name+"："+reason)
	}
	return check
}

func main() {
	specDirectory := "docs/spec"
	outputPath := filepath.Join(specDirectory, "000-index.md")
	reportPath := filepath.Join("docs", "audit", "doc-index.json")

	entries, err := os.ReadDir(specDirectory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	code, tests, err := collectReferences([]string{"cmd", "internal"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var specs []spec
	for _, entry := range entries {
		if entry.IsDir() || !specFileName.MatchString(entry.Name()) {
			continue
		}
		if entry.Name() == "000-index.md" {
			continue
		}
		parsed, err := readSpec(filepath.Join(specDirectory, entry.Name()))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		parsed.code = code[parsed.number]
		parsed.tests = tests[parsed.number]
		specs = append(specs, parsed)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].number < specs[j].number })

	tools, err := readTools("cmd")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var out strings.Builder
	out.WriteString("# 規格索引\n\n")
	out.WriteString("> 這份由 `cmd/pool-doc-index` 產生，**手改會在下一次執行時被蓋掉**。\n")
	out.WriteString("> 對應關係的主鍵是 spec 編號——程式碼註解裡的 `spec NNN` 就是那條線，\n")
	out.WriteString("> 這份只是把它反過來收攏，所以改了註解重跑一次就對了。\n\n")

	var noCode, noTests, shared, outside int
	for _, item := range specs {
		switch {
		case len(item.code) > 0:
		case item.sharedEngine:
			shared++
		case item.outsideGo != "":
			outside++
		default:
			noCode++
		}
		if len(item.tests) == 0 {
			noTests++
		}
	}
	fmt.Fprintf(&out, "%d 份規格，其中 %d 份還沒有任何檔案的註解指回它、%d 份沒有測試提到它；\n"+
		"另有 %d 份實作在共用 engine（`eclvm`）、%d 份的實作不是 Go（規格自己寫的那行 `實作：`）。\n",
		len(specs), noCode, noTests, shared, outside)
	out.WriteString("這些數字是**盤點用的**：沒有反向引用不代表沒實作，只代表那條線還沒接起來。\n\n")

	out.WriteString("## 規格\n\n")
	out.WriteString("| # | 標題 | 狀態 | 實作 | 測試 |\n|---|---|---|---|---|\n")
	for _, item := range specs {
		status := strings.Join(item.statuses, "＋")
		if status == "" {
			status = "—"
		}
		implementation := shorten(item.code)
		if len(item.code) == 0 {
			switch {
			case item.sharedEngine:
				implementation = "共用 engine"
			case item.outsideGo != "":
				implementation = item.outsideGo
			}
		}
		fmt.Fprintf(&out, "| [%s](%s) | %s | %s | %s | %s |\n",
			item.number, item.file, item.title, status, implementation, shorten(item.tests))
	}

	out.WriteString("\n## `cmd/` 底下的工具\n\n")
	out.WriteString("| 工具 | 做什麼 | 測試 | 相關規格 |\n|---|---|---|---|\n")
	for _, item := range tools {
		marker := "—"
		if item.hasTests {
			marker = "有"
		}
		related := "—"
		if len(item.specs) > 0 {
			shown := item.specs
			suffix := ""
			if len(shown) > 6 {
				shown, suffix = shown[:6], fmt.Sprintf(" 等 %d 份", len(item.specs))
			}
			related = strings.Join(shown, "、") + suffix
		}
		summary := item.summary
		if summary == "" {
			summary = "（檔頭沒寫）"
		}
		fmt.Fprintf(&out, "| `%s` | %s | %s | %s |\n", item.name, summary, marker, related)
	}

	if err := os.WriteFile(outputPath, []byte(out.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	check := crossCheckTools(tools)

	report := gapReport{
		Schema:                     "pool-doc-index/1",
		CrossCheck:                 check,
		SpecCount:                  len(specs),
		ToolCount:                  len(tools),
		SpecsWithoutImplementation: []string{},
		SpecsWithoutTests:          []string{},
		SpecsInSharedEngine:        []string{},
		SpecsOutsideGo:             []string{},
		ToolsWithoutDoc:            []string{},
		ToolsWithoutTests:          []string{},
	}
	for _, item := range specs {
		switch {
		case len(item.code) > 0:
		case item.sharedEngine:
			report.SpecsInSharedEngine = append(report.SpecsInSharedEngine, item.number)
		case item.outsideGo != "":
			report.SpecsOutsideGo = append(report.SpecsOutsideGo, item.number)
		default:
			report.SpecsWithoutImplementation = append(report.SpecsWithoutImplementation, item.number)
		}
		if len(item.tests) == 0 {
			report.SpecsWithoutTests = append(report.SpecsWithoutTests, item.number)
		}
	}
	for _, item := range tools {
		if item.summary == "" {
			report.ToolsWithoutDoc = append(report.ToolsWithoutDoc, item.name)
		}
		if !item.hasTests {
			report.ToolsWithoutTests = append(report.ToolsWithoutTests, item.name)
		}
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(reportPath, append(encoded, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("寫出 %s：%d 份規格、%d 支工具\n", outputPath, len(specs), len(tools))
	fmt.Printf("寫出 %s：%d 份沒有實作引用、%d 份沒有測試、%d 支工具沒有測試\n",
		reportPath, len(report.SpecsWithoutImplementation),
		len(report.SpecsWithoutTests), len(report.ToolsWithoutTests))

	// 交叉判準沒過就大聲講，並且用非零離開碼——這份報告的數字下游會拿去當
	// 「還剩多少」的依據，不可信的時候要擋在那之前。
	if !check.Passed {
		fmt.Fprintf(os.Stderr, "\n交叉判準不一致 %d 處，這份報告的數字先不要用：\n", len(check.Mismatches))
		for _, line := range check.Mismatches {
			fmt.Fprintln(os.Stderr, "  "+line)
		}
		os.Exit(1)
	}
}
