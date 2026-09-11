// pool-worklist 管未完成項。
//
// **權威是 `docs/worklist.json`，不是 WORKLIST.md。** markdown 那一節由這支
// 的 `render` 產生——手改會在下一次 render 時被蓋掉。
//
// 這支存在的理由是「過期斷言」：東西做好了而清單沒有人回頭改，於是清單上
// 留著一條假的「還沒接」。2026-09-10 抓到的臭雲術就是那樣——派發那一格早就
// 接上，條目卻還寫著「派發表六十七格裡只剩這一支」。所以每一項都掛一個
// `verify`：**跑起來為真代表「這一條仍然未完成」**，為假就是該回頭改條目了。
//
// # verify 擋得住什麼、擋不住什麼
//
// `present`／`absent` 自己去掃產品程式碼，所以它們看的是地上的真相。
// `json_len`／`json_gap` 不是——它們讀的是**別的工具算出來的那份 JSON**，
// 於是那支工具算錯的時候，verify 會照樣印出好消息。
//
// 2026-09-10 用一個小場景驗過：盤點工具把「目錄裡多於一個 .py」當成「有測試」
// 的代理指標，兩支沒有測試的工具因此被判成有，缺口清單變空，verify 於是宣告
// 「可能已完成」。條目沒問題、verify 沒問題、JSON 格式也沒問題——錯的是上游。
//
// 所以綁 JSON 的那幾條有一個前提：**產生那份 JSON 的工具要有測試。**
// 這不是衛生問題，是這一整套機制的單點失效，`tool-test-coverage` 那一條問的
// 就是它。
//
// # 「可能已完成」這四個字是刻意的，不要改成「已完成」
//
// verify 分不出「真的做完了」與「上游算錯了」——兩種都會讓它不再成立。
// 它能做的是把兩種都推到人面前：印**可能**已完成、`os.Exit(1)`、明講回頭去
// 看。**這個輸出是傳票不是判決**，措辭與離開碼都是這個意思的一部分。
//
// 同一個小場景裡，拿到這張傳票的人正是因為它開了口才去查資料夾，然後才發現
// 盤點工具的判準是錯的。把措辭改成肯定句、或讓 stale 時回 0，等於把那個
// 「回去確認」的時刻拿掉。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type verify struct {
	Kind    string   `json:"kind"` // present | absent | json_len | json_gap | manual
	Paths   []string `json:"paths,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
	// json_len 用：讀 Path 那份 JSON 的 Field，長度 <= Max 代表這一條仍然
	// 未完成。用在「還沒擴充到第幾張／第幾項」這種進度型的條目上，比 grep
	// 註解準——註解會被順手改掉，數量不會。
	//
	// json_gap 是同一份資料的另一個方向：Field 的長度 >= Min 代表缺口還在。
	// 兩個方向不能共用一個 kind——進度型的欄位是「越多越好」（涵蓋幾張圖），
	// 缺口型是「越少越好」（幾份規格沒人指回去）。共用一個名字，遲早有人把
	// max 當成 min 填，而填反了的 verify 會一直說好消息。
	Path  string `json:"path,omitempty"`
	Field string `json:"field,omitempty"`
	Max   int    `json:"max,omitempty"`
	Min   int    `json:"min,omitempty"`
	Note  string `json:"note,omitempty"`
}

type item struct {
	ID         string `json:"id"`
	Layer      string `json:"layer"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	BlockedBy  string `json:"blocked_by,omitempty"`
	Acceptance string `json:"acceptance"`
	Verify     verify `json:"verify"`
	// GitHubIssue 是這一條在 GitHub 上對應的 issue 編號。0 代表還沒開。
	// 兩邊並存：issue 給人討論，這一份留著是因為 verify 要跑得起來。
	GitHubIssue int `json:"github_issue,omitempty"`
}

// issueBase 是 issue 連結的前綴。render 會把編號接在後面。
const issueBase = "https://github.com/wicanr2/Pool-of-Radiance-cht/issues/"

type file struct {
	Schema string            `json:"schema"`
	Note   string            `json:"note"`
	Layers map[string]string `json:"layers"`
	Items  []item            `json:"items"`
}

// layerOrder 是三層的固定順序。用 map 的走訪順序會讓 render 每次吐不同的
// 排列，diff 於是看起來像內容變了。
//
// 層**之內**照 JSON 裡的先後，不另外排序：那個順序是人排的重要性，
// 按 id 字母序會把它洗掉。
var layerOrder = []string{"feature", "verification", "presentation"}

func load(path string) (*file, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var decoded file
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if decoded.Schema != "pool-worklist/1" {
		return nil, fmt.Errorf("%s 的 schema 是 %q，不是 pool-worklist/1", path, decoded.Schema)
	}
	seen := map[string]bool{}
	for _, one := range decoded.Items {
		if one.ID == "" || one.Title == "" || one.Acceptance == "" {
			return nil, fmt.Errorf("條目 %q 缺 id／title／acceptance", one.ID)
		}
		if seen[one.ID] {
			return nil, fmt.Errorf("條目 id %q 重複", one.ID)
		}
		seen[one.ID] = true
		if _, ok := decoded.Layers[one.Layer]; !ok {
			return nil, fmt.Errorf("條目 %q 的 layer %q 不在 layers 裡", one.ID, one.Layer)
		}
		switch one.Verify.Kind {
		case "present", "absent":
			if one.Verify.Pattern == "" || len(one.Verify.Paths) == 0 {
				return nil, fmt.Errorf("條目 %q 的 verify 缺 pattern／paths", one.ID)
			}
		case "json_len":
			if one.Verify.Path == "" || one.Verify.Field == "" || one.Verify.Max <= 0 {
				return nil, fmt.Errorf("條目 %q 的 verify 缺 path／field／max", one.ID)
			}
		case "json_gap":
			if one.Verify.Path == "" || one.Verify.Field == "" || one.Verify.Min <= 0 {
				return nil, fmt.Errorf("條目 %q 的 verify 缺 path／field／min", one.ID)
			}
		case "manual":
		default:
			return nil, fmt.Errorf("條目 %q 的 verify.kind %q 不認得", one.ID, one.Verify.Kind)
		}
	}
	return &decoded, nil
}

// matches 說 pattern 在 paths 底下的 .go／.py／.sh 裡找不找得到。
func matches(root string, one verify) (bool, string, error) {
	expression, err := regexp.Compile(one.Pattern)
	if err != nil {
		return false, "", fmt.Errorf("pattern %q: %w", one.Pattern, err)
	}
	for _, target := range one.Paths {
		full := filepath.Join(root, target)
		info, err := os.Stat(full)
		if err != nil {
			return false, "", err
		}
		var found string
		walk := func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || found != "" {
				return err
			}
			switch filepath.Ext(path) {
			case ".go", ".py", ".sh", ".json", ".md":
			default:
				return nil
			}
			// **測試檔不算。** 條目問的是「產品程式碼做了這件事沒有」，
			// 而測試本來就會提到還沒接上的東西——把 `_test.go` 算進來，
			// `absent` 會因為測試裡有一行呼叫就判成「已經做了」。
			if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_test.py") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if expression.Match(raw) {
				rel, _ := filepath.Rel(root, path)
				found = rel
			}
			return nil
		}
		if info.IsDir() {
			if err := filepath.WalkDir(full, walk); err != nil {
				return false, "", err
			}
		} else {
			raw, err := os.ReadFile(full)
			if err != nil {
				return false, "", err
			}
			if expression.Match(raw) {
				found = target
			}
		}
		if found != "" {
			return true, found, nil
		}
	}
	return false, "", nil
}

// stillOpen 回答「這一條仍然未完成嗎」。manual 沒有機器可判的訊號，回 true
// 並在報告裡標出來——**沉默不等於通過**，那幾條要人自己去看。
func stillOpen(root string, one item) (bool, string, error) {
	switch one.Verify.Kind {
	case "manual":
		return true, "要人判", nil
	case "present":
		hit, where, err := matches(root, one.Verify)
		if err != nil {
			return false, "", err
		}
		if hit {
			return true, "自承還在 " + where, nil
		}
		return false, "找不到 " + one.Verify.Pattern, nil
	case "absent":
		hit, where, err := matches(root, one.Verify)
		if err != nil {
			return false, "", err
		}
		if hit {
			return false, "已經出現在 " + where, nil
		}
		return true, "還沒出現", nil
	case "json_len":
		distrust, err := upstreamDistrusted(filepath.Join(root, one.Verify.Path))
		if err != nil {
			return false, "", err
		}
		if distrust != "" {
			return true, distrust + "，這份數字不能拿來下結論", nil
		}
		count, err := jsonFieldLen(filepath.Join(root, one.Verify.Path), one.Verify.Field)
		if err != nil {
			return false, "", err
		}
		if count <= one.Verify.Max {
			return true, fmt.Sprintf("%s 的 %s 有 %d 項（<= %d）", one.Verify.Path, one.Verify.Field, count, one.Verify.Max), nil
		}
		return false, fmt.Sprintf("%s 的 %s 已經有 %d 項（> %d）", one.Verify.Path, one.Verify.Field, count, one.Verify.Max), nil
	case "json_gap":
		distrust, err := upstreamDistrusted(filepath.Join(root, one.Verify.Path))
		if err != nil {
			return false, "", err
		}
		if distrust != "" {
			// 不可信的時候一律回「仍未完成」——沉默不等於通過，
			// 而「不知道」比「可能已完成」更接近實情。
			return true, distrust + "，這份數字不能拿來下結論", nil
		}
		count, err := jsonFieldLen(filepath.Join(root, one.Verify.Path), one.Verify.Field)
		if err != nil {
			return false, "", err
		}
		if count >= one.Verify.Min {
			return true, fmt.Sprintf("%s 的 %s 還有 %d 項（>= %d）", one.Verify.Path, one.Verify.Field, count, one.Verify.Min), nil
		}
		return false, fmt.Sprintf("%s 的 %s 只剩 %d 項（< %d）", one.Verify.Path, one.Verify.Field, count, one.Verify.Min), nil
	}
	return false, "", fmt.Errorf("verify.kind %q 不認得", one.Verify.Kind)
}

// upstreamDistrusted 問那份 JSON「你自己信不信你自己」。
//
// json_len／json_gap 讀的是別的工具算出來的數字，而「算錯」和「真的沒缺口」
// 在數字上長得一模一樣。所以約定：產生資料的工具可以在頂層放一個
// `cross_check` 物件，用第二個資訊來源驗自己一次，`passed` 為 false 就表示
// 這份數字不能拿來下結論。
//
// 沒有這個欄位時回空字串——不是每一份輸入都有自檢，強制要求會讓既有的條目
// 一起壞掉。這是漸進的：有戳記的就檢查，沒有的照舊。
func upstreamDistrusted(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var decoded struct {
		CrossCheck *struct {
			Passed     bool     `json:"passed"`
			Mismatches []string `json:"mismatches"`
		} `json:"cross_check"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if decoded.CrossCheck == nil || decoded.CrossCheck.Passed {
		return "", nil
	}
	// 把 mismatches 的內容帶出來，不要只報數量——那幾行本身常常就指名了是誰
	// 出問題，短路掉等於把手上最有用的線索丟掉，只留一句「不知道」。
	detail := decoded.CrossCheck.Mismatches
	if len(detail) > 2 {
		detail = append(append([]string{}, detail[:2]...),
			fmt.Sprintf("…另外 %d 處", len(decoded.CrossCheck.Mismatches)-2))
	}
	return fmt.Sprintf("上游的交叉判準不一致 %d 處（%s）",
		len(decoded.CrossCheck.Mismatches), strings.Join(detail, "；")), nil
}

// jsonFieldLen 讀一份 JSON 的某個頂層欄位有幾項。物件數鍵、陣列數元素。
func jsonFieldLen(path, field string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, fmt.Errorf("%s: %w", path, err)
	}
	value, ok := decoded[field]
	if !ok {
		return 0, fmt.Errorf("%s 沒有欄位 %q", path, field)
	}
	var asObject map[string]json.RawMessage
	if err := json.Unmarshal(value, &asObject); err == nil {
		return len(asObject), nil
	}
	var asArray []json.RawMessage
	if err := json.Unmarshal(value, &asArray); err == nil {
		return len(asArray), nil
	}
	return 0, fmt.Errorf("%s 的 %q 既不是物件也不是陣列", path, field)
}

// indented 讓多行的欄位在 markdown 清單裡續得下去。條目的 `body` 或
// `blocked_by` 分成兩段時，第二段若頂到最左邊，markdown 會把它讀成另一個
// 段落——清單項就在那裡斷掉，而 JSON 那邊看起來完全正常。
func indented(text string) string {
	return strings.ReplaceAll(text, "\n", "\n      ")
}

func render(decoded *file) string {
	var out strings.Builder
	byLayer := map[string][]item{}
	for _, one := range decoded.Items {
		byLayer[one.Layer] = append(byLayer[one.Layer], one)
	}
	for index, layer := range layerOrder {
		list := byLayer[layer]
		if len(list) == 0 {
			continue
		}
		fmt.Fprintf(&out, "### %s、%s\n\n", []string{"一", "二", "三"}[index], decoded.Layers[layer])
		for _, one := range list {
			fmt.Fprintf(&out, "- [ ] **%s。** %s\n", one.Title, indented(one.Body))
			if one.BlockedBy != "" {
				fmt.Fprintf(&out, "      **卡在**：%s\n", indented(one.BlockedBy))
			}
			fmt.Fprintf(&out, "      **驗收**：%s\n", indented(one.Acceptance))
			if one.GitHubIssue != 0 {
				fmt.Fprintf(&out, "      **討論**：[#%d](%s%d)\n",
					one.GitHubIssue, issueBase, one.GitHubIssue)
			}
		}
		out.WriteString("\n")
	}
	return out.String()
}

// WORKLIST.md 那一節的邊界。用註解標記而不是靠節標題文字定位——標題會被改，
// 而定位錯了不會報錯，只會把手寫的段落一起蓋掉。
const (
	beginMarker = "<!-- worklist:begin"
	endMarker   = "<!-- worklist:end -->"
)

// writeInto 把 render 的結果寫回兩個標記之間。
//
// 沒有這一步，render 的輸出要靠人貼回 markdown，而那一步會出錯：
// 2026-09-10 發現 WORKLIST.md 裡「平面圖的畫法」那一條貼成了六份，
// 而 worklist.json 從頭到尾只有一條。
func writeInto(path, body string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(raw)

	begin := strings.Index(text, beginMarker)
	if begin < 0 {
		return fmt.Errorf("%s 裡找不到 %s 標記", path, beginMarker)
	}
	offset := strings.Index(text[begin:], "\n")
	if offset < 0 {
		return fmt.Errorf("%s 的 %s 標記後面沒有換行", path, beginMarker)
	}
	end := strings.Index(text, endMarker)
	if end < 0 {
		return fmt.Errorf("%s 裡找不到 %s 標記", path, endMarker)
	}
	if end < begin {
		return fmt.Errorf("%s 的兩個標記順序反了", path)
	}

	return os.WriteFile(path, []byte(text[:begin+offset+1]+"\n"+body+text[end:]), 0o644)
}

func main() {
	root := flag.String("root", ".", "repository root")
	source := flag.String("json", "docs/worklist.json", "未完成項的權威資料")
	mode := flag.String("mode", "verify", "verify｜render｜list")
	write := flag.String("write", "", "render 時寫回這份 markdown 的 worklist 標記之間；留空就印到 stdout")
	flag.Parse()

	decoded, err := load(filepath.Join(*root, *source))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	switch *mode {
	case "render":
		body := render(decoded)
		if *write == "" {
			fmt.Print(body)
			break
		}
		target := filepath.Join(*root, *write)
		if err := writeInto(target, body); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Printf("寫回 %s：%d 條\n", target, len(decoded.Items))
	case "list":
		for _, one := range decoded.Items {
			fmt.Printf("%-28s %-12s %s\n", one.ID, one.Layer, one.Title)
		}
	case "verify":
		stale := 0
		for _, one := range decoded.Items {
			open, why, err := stillOpen(*root, one)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", one.ID, err)
				os.Exit(2)
			}
			mark := "仍未完成"
			if !open {
				mark, stale = "**可能已完成**", stale+1
			}
			fmt.Printf("%-28s %-14s %s\n", one.ID, mark, why)
		}
		// 離開碼問的是「有沒有過期斷言」，不是「還剩多少沒做」。
		// 全部仍未完成時它是 0——那代表清單誠實，不代表東西做完了。
		if stale != 0 {
			fmt.Fprintf(os.Stderr, "\n%d 條的 verify 不再成立——回頭看那幾條是不是已經做完了。\n", stale)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "mode %q 不認得\n", *mode)
		os.Exit(2)
	}
}
