package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 檔頭的一句話常常跨行，中英夾雜。接錯了索引表會出現「共用engine」這種字。
func TestFirstDocLineJoinsWrappedSentences(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		source string
		tool   string
		want   string
	}{
		{
			name: "中文換行接英文要補空格",
			source: "// pool-game 是 remake 的遊戲本體：Ebiten 視窗、玩家輸入，以及與共用\n" +
				"// engine 和 game pack 的接線。玩法規則寫在別處。\npackage main\n",
			tool: "pool-game",
			want: "remake 的遊戲本體：Ebiten 視窗、玩家輸入，以及與共用 engine 和 game pack 的接線",
		},
		{
			name:   "中文接中文不補空格",
			source: "// pool-worklist 把未完成項當資料管，\n// 不當散文管。\npackage main\n",
			tool:   "pool-worklist",
			want:   "把未完成項當資料管，不當散文管",
		},
		{
			name: "英文只留第一句",
			source: "// Command pool-ecl-audit measures the reusable ECL decoder against every\n" +
				"// Pool ECL block. Static reachability is not parity.\npackage main\n",
			tool: "pool-ecl-audit",
			want: "measures the reusable ECL decoder against every Pool ECL block",
		},
		{
			name:   "沒有檔頭就回空字串",
			source: "package main\n\nimport \"fmt\"\n",
			tool:   "export-title",
			want:   "",
		},
	} {
		if got := firstDocLine(testCase.source, testCase.tool); got != testCase.want {
			t.Errorf("%s：\n 得到 %q\n 該是 %q", testCase.name, got, testCase.want)
		}
	}
}

// 一份規格常常一半 CONFORMED 一半 DRAFT，而且狀態段會跨行寫。
// 只讀第一行會把後半的狀態漏掉。
func TestReadSpecCollectsEveryStatusAcrossWrappedLines(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "122-locked-doors.md")
	content := "# Spec 122：鎖住的門（`Bash`／`Pick`／`Knock`）\n\n" +
		"狀態：CONFORMED（門的狀態查詢、成功之後怎麼寫回\n" +
		"GEO）；DRAFT（那三個 byte 是誰設回 1 的）。\n" +
		"日期：2026-09-05。\n\n" +
		"## 為什麼要讀這一支\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := readSpec(path)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.number != "122" {
		t.Errorf("編號讀成 %q", parsed.number)
	}
	if want := "鎖住的門（`Bash`／`Pick`／`Knock`）"; parsed.title != want {
		t.Errorf("標題讀成 %q，該是 %q", parsed.title, want)
	}
	if got := strings.Join(parsed.statuses, "＋"); got != "CONFORMED＋DRAFT" {
		t.Errorf("狀態讀成 %q，該是 CONFORMED＋DRAFT", got)
	}
}

// 「日期：」之後的內文不算狀態——那裡出現 DRAFT 這個字是在講別的事。
func TestReadSpecStopsAtTheDateLine(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "100-exit-gate.md")
	content := "# Spec 100：出口閘門\n\n狀態：CONFORMED。\n日期：2026-09-06。\n\n" +
		"內文提到這一份取代了先前的 DRAFT 與 OPEN 版本。\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	parsed, err := readSpec(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(parsed.statuses, "＋"); got != "CONFORMED" {
		t.Errorf("狀態讀成 %q，該只有 CONFORMED", got)
	}
}

func TestNeedsSpaceOnlySkipsBetweenTwoHanCharacters(t *testing.T) {
	for _, testCase := range []struct {
		left, right string
		want        bool
	}{
		{"以及與共用", "engine 和 game pack", true},
		{"每一個 ECL", "區塊會 NEWECL", true},
		{"把未完成項當資料管，", "不當散文管", false},
		{"captured at", "an integer scale", true},
		{"", "開頭", false},
	} {
		if got := needsSpace(testCase.left, testCase.right); got != testCase.want {
			t.Errorf("needsSpace(%q, %q) = %v，該是 %v",
				testCase.left, testCase.right, got, testCase.want)
		}
	}
}

// 有幾份規格的實作在共用 engine，這個 repo 掃不到。標不出來的話那些會顯示成
// 「—」，和「真的還沒接」混在一起。
func TestReadSpecMarksSharedEngineImplementations(t *testing.T) {
	directory := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	shared := write("027-save-table-opcode.md",
		"# Spec 027：`35h SAVE TABLE`\n\n狀態：CONFORMED。\n日期：2026-09-01。\n\n"+
			"remake 這一側由共用 engine 實作（`eclvm/machine.go` 的 `case 0x35`）。\n")
	local := write("122-locked-doors.md",
		"# Spec 122：鎖住的門\n\n狀態：CONFORMED。\n日期：2026-09-05。\n\n"+
			"remake 這一側在 internal/gamepack/door.go。\n")

	parsed, err := readSpec(shared)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.sharedEngine {
		t.Error("提到 eclvm 的規格該標成共用 engine")
	}

	parsed, err = readSpec(local)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.sharedEngine {
		t.Error("沒提到 eclvm 的規格不該標成共用 engine")
	}
}

// 交叉判準要抓的是「同一件事兩個來源說法不同」，而不是「有沒有測試」本身。
// 兩個方向都要抓——只抓一邊等於只有一個判準。
func TestCrossCheckToolsCatchesBothDirections(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		tools   []tool
		passed  bool
		mention string
	}{
		{
			name: "兩邊都說有、兩邊都說沒有，都算一致",
			tools: []tool{
				{name: "alpha", hasTests: true, hasTestFuncs: true},
				{name: "beta", hasTests: false, hasTestFuncs: false},
			},
			passed: true,
		},
		{
			name:    "檔名對但裡面沒有 func Test",
			tools:   []tool{{name: "gamma", hasTests: true, hasTestFuncs: false}},
			passed:  false,
			mention: "沒有 func Test",
		},
		{
			name:    "有 func Test 但檔名不對",
			tools:   []tool{{name: "delta", hasTests: false, hasTestFuncs: true}},
			passed:  false,
			mention: "go test 不會跑它",
		},
	} {
		check := crossCheckTools(testCase.tools)
		if check.Passed != testCase.passed {
			t.Errorf("%s：passed=%v，該是 %v（%v）", testCase.name, check.Passed, testCase.passed, check.Mismatches)
			continue
		}
		if testCase.mention == "" {
			continue
		}
		if len(check.Mismatches) != 1 || !strings.Contains(check.Mismatches[0], testCase.mention) {
			t.Errorf("%s：理由是 %v，該提到 %q", testCase.name, check.Mismatches, testCase.mention)
		}
	}
}

// 判準 B 認的是 go test 真的會跑的那些函式。認太寬就跟判準 A 一樣寬，
// 交叉判準也就失去意義了。
func TestTestFunctionMatchesWhatGoTestActuallyRuns(t *testing.T) {
	for _, testCase := range []struct {
		source string
		want   bool
	}{
		{"func TestThing(t *testing.T) {}", true},
		{"func FuzzThing(f *testing.F) {}", true},
		{"func TestX(t *testing.T) {}\n", true},
		{"func BenchmarkThing(b *testing.B) {}", false},
		{"func Testing() {}", false},
		{"// func TestCommentedOut(t *testing.T) {}", false},
		{"\tfunc TestIndented(t *testing.T) {}", false},
		{"func testLower(t *testing.T) {}", false},
	} {
		if got := testFunction.MatchString(testCase.source); got != testCase.want {
			t.Errorf("%q → %v，該是 %v", testCase.source, got, testCase.want)
		}
	}
}
