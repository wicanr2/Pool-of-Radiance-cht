// pool-doc-index 產生 docs/spec/000-index.md：每份規格的狀態、實作它的檔案、
// 釘住它的測試，以及 cmd/ 底下每一支工具在做什麼。
//
// **這份索引是算出來的，不是寫出來的。** 對應關係的主鍵是 spec 編號——程式碼
// 註解裡的 `spec NNN` 就是那條線，本工具只是把它反過來收攏。所以索引不會過期：
// 改了註解重跑一次就對了，不需要有人記得同步。手改 000-index.md 會在下一次
// 執行時被蓋掉。
package main

import (
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
	statusWords   = []string{"CONFORMED", "READY", "DRAFT", "OPEN"}
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
}

type tool struct {
	name     string
	summary  string
	hasTests bool
	specs    []string
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

func main() {
	specDirectory := "docs/spec"
	outputPath := filepath.Join(specDirectory, "000-index.md")

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

	var noCode, noTests, shared int
	for _, item := range specs {
		switch {
		case len(item.code) > 0:
		case item.sharedEngine:
			shared++
		default:
			noCode++
		}
		if len(item.tests) == 0 {
			noTests++
		}
	}
	fmt.Fprintf(&out, "%d 份規格，其中 %d 份還沒有任何檔案的註解指回它、%d 份沒有測試提到它；\n"+
		"另有 %d 份實作在共用 engine（`eclvm`），不在這個 repo。\n",
		len(specs), noCode, noTests, shared)
	out.WriteString("這些數字是**盤點用的**：沒有反向引用不代表沒實作，只代表那條線還沒接起來。\n\n")

	out.WriteString("## 規格\n\n")
	out.WriteString("| # | 標題 | 狀態 | 實作 | 測試 |\n|---|---|---|---|---|\n")
	for _, item := range specs {
		status := strings.Join(item.statuses, "＋")
		if status == "" {
			status = "—"
		}
		implementation := shorten(item.code)
		if len(item.code) == 0 && item.sharedEngine {
			implementation = "共用 engine"
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
	fmt.Printf("寫出 %s：%d 份規格、%d 支工具\n", outputPath, len(specs), len(tools))
}
