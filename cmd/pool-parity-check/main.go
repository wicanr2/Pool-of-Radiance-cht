// Command pool-parity-check 把一次對拍跑出來的 parity.json 拿去對**基準表**
// （`docs/audit/dos-parity-sample.json`），而不是對上一次跑的結果。
//
// 為什麼要有這一支：歷輪 commit 寫的「變好 0、變差 0、不變 33」都是與上一次跑的
// 結果比。那種比法在每一輪都成立，卻永遠不會發現「表上的數字與現在的程式差了十項」
// ——2026-09-18 查出 `dos-parity-sample.md` 與它自己的 JSON 漂開十項，就是這樣來的
// （remake 沒有退步：v.1.1.5 與 v.1.1.12 今天量到的 33 張逐欄相同）。
//
// 表由這一支從 JSON 生成（`-write`），所以 md 與 JSON 不會再各說各話。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// screen 是一張畫面的量測結果。`captured` 為假代表這一次沒拍到
// （擷圖停在半路，目前是 #42）——那不是變差，是沒有量。
type screen struct {
	Name       string  `json:"name"`
	Label      string  `json:"label,omitempty"`
	Kind       string  `json:"kind,omitempty"`
	Reference  string  `json:"reference,omitempty"`
	RefStep    int64   `json:"reference_step,omitempty"`
	Remake     string  `json:"remake,omitempty"`
	Captured   *bool   `json:"captured,omitempty"`
	Same       int     `json:"same,omitempty"`
	Total      int     `json:"total,omitempty"`
	Ratio      float64 `json:"ratio,omitempty"`
	Note       string  `json:"note,omitempty"`
	FrameSame  int     `json:"frame_same,omitempty"`
	FrameTotal int     `json:"frame_total,omitempty"`
	FrameRatio float64 `json:"frame_ratio,omitempty"`
	ViewSame   int     `json:"view_same,omitempty"`
	ViewTotal  int     `json:"view_total,omitempty"`
	ViewRatio  float64 `json:"view_ratio,omitempty"`
	// ViewAlternatives 是「視野」允許的另一個值。營火是兩張動畫，紮營那幾張的
	// 視野本來就在 100% 與 91.27% 之間跳（`internal/assets/camp_fire_test.go`
	// 釘住這兩個數字），兩個都對，不能報成變差。
	ViewAlternatives []int `json:"view_alternatives,omitempty"`
}

func (s screen) taken() bool { return s.Captured == nil || *s.Captured }

type report struct {
	Screens []screen `json:"screens"`
}

// difference 是一張畫面的一欄對不上。
type difference struct {
	Screen, Field string
	Sample, Run   int
	Total         int
}

func main() {
	runPath := flag.String("run", "workplace/dos-parity-zh/parity.json", "這一次對拍產生的 parity.json")
	samplePath := flag.String("sample", "docs/audit/dos-parity-sample.json", "基準表")
	markdown := flag.String("markdown", "docs/audit/dos-parity-sample.md", "基準表的 Markdown；-write 會重生它的表")
	write := flag.Bool("write", false, "把這一次的數字寫回基準表並重生 Markdown 的表")
	flag.Parse()

	run, err := load(*runPath)
	if err != nil {
		fail(err)
	}
	sample, err := load(*samplePath)
	if err != nil {
		fail(err)
	}
	differences, missing, unknown := compare(sample, run)
	for _, name := range missing {
		fmt.Printf("%-18s 未量（擷圖沒拍到，不算變差）\n", name)
	}
	for _, name := range unknown {
		fmt.Printf("%-18s 基準表沒有這一張\n", name)
	}
	for _, d := range differences {
		fmt.Printf("%-18s %-11s 表 %d／%d　這次 %d／%d\n", d.Screen, d.Field, d.Sample, d.Total, d.Run, d.Total)
	}
	fmt.Printf("對表：%d 張量到、%d 張未量、%d 欄對不上\n",
		len(run.Screens)-len(missing), len(missing), len(differences))

	if !*write {
		if len(differences) != 0 || len(unknown) != 0 {
			os.Exit(1)
		}
		return
	}
	merged := merge(sample, run)
	if err := save(*samplePath, merged); err != nil {
		fail(err)
	}
	if err := writeTable(*markdown, merged); err != nil {
		fail(err)
	}
	fmt.Printf("寫回 %s 與 %s 的表\n", *samplePath, *markdown)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

func load(path string) (report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return report{}, err
	}
	var result report
	if err := json.Unmarshal(data, &result); err != nil {
		return report{}, fmt.Errorf("%s: %w", path, err)
	}
	return result, nil
}

func save(path string, result report) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// compare 回傳對不上的欄、這一次沒量到的畫面，以及基準表裡沒有的畫面。
func compare(sample, run report) ([]difference, []string, []string) {
	baseline := map[string]screen{}
	for _, item := range sample.Screens {
		baseline[item.Name] = item
	}
	var differences []difference
	var missing, unknown []string
	for _, item := range run.Screens {
		base, ok := baseline[item.Name]
		if !ok {
			unknown = append(unknown, item.Name)
			continue
		}
		if !item.taken() {
			missing = append(missing, item.Name)
			continue
		}
		if item.Same != base.Same {
			differences = append(differences, difference{item.Name, "整張", base.Same, item.Same, base.Total})
		}
		if base.FrameTotal != 0 && item.FrameSame != base.FrameSame {
			differences = append(differences, difference{item.Name, "外框", base.FrameSame, item.FrameSame, base.FrameTotal})
		}
		if base.ViewTotal != 0 && item.ViewSame != base.ViewSame && !allowed(base, item.ViewSame) {
			differences = append(differences, difference{item.Name, "視野", base.ViewSame, item.ViewSame, base.ViewTotal})
		}
	}
	// 基準表有、這一次連提都沒提到的（例如整段沒跑到）也算未量。
	seen := map[string]bool{}
	for _, item := range run.Screens {
		seen[item.Name] = true
	}
	for name := range baseline {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(unknown)
	sort.Slice(differences, func(i, j int) bool { return differences[i].Screen < differences[j].Screen })
	return differences, missing, unknown
}

// allowed 說這個「視野」是不是基準允許的另一個值（營火兩張動畫）。
func allowed(base screen, value int) bool {
	for _, alternative := range base.ViewAlternatives {
		if alternative == value {
			return true
		}
	}
	return false
}

// merge 把這一次量到的數字蓋回基準，沒量到的保留基準原值（不要用缺席蓋掉既有斷言）。
func merge(sample, run report) report {
	fresh := map[string]screen{}
	for _, item := range run.Screens {
		if item.taken() {
			fresh[item.Name] = item
		}
	}
	merged := report{Screens: make([]screen, 0, len(sample.Screens))}
	for _, base := range sample.Screens {
		item, ok := fresh[base.Name]
		if !ok {
			merged.Screens = append(merged.Screens, base)
			continue
		}
		// 標籤、備註與允許的替代值是人寫的，量測不會帶，留基準的。
		item.Label, item.Note, item.ViewAlternatives = base.Label, base.Note, base.ViewAlternatives
		merged.Screens = append(merged.Screens, item)
	}
	return merged
}

const (
	tableStart = "<!-- 表由 cmd/pool-parity-check 產生，別手改 -->"
	tableEnd   = "<!-- 表結束 -->"
)

// writeTable 重生 Markdown 裡的那張表。表與 JSON 從此同一個來源——先前兩份各自維護，
// md 的十項與 JSON 對不上而沒有人發現。
func writeTable(path string, result report) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	begin := strings.Index(text, tableStart)
	finish := strings.Index(text, tableEnd)
	if begin < 0 || finish < 0 || finish < begin {
		return fmt.Errorf("%s 裡沒有 %q／%q 這對標記", path, tableStart, tableEnd)
	}
	var builder strings.Builder
	builder.WriteString(tableStart + "\n\n")
	builder.WriteString("| 畫面 | 整張 | 外框 | 視野 |\n|---|---:|---:|---:|\n")
	for _, item := range result.Screens {
		label := item.Label
		if label == "" {
			label = item.Name
		}
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			label, percent(item.Same, item.Total), percent(item.FrameSame, item.FrameTotal),
			viewPercent(item)))
	}
	builder.WriteString("\n")
	return os.WriteFile(path, []byte(text[:begin]+builder.String()+text[finish:]), 0o644)
}

func percent(same, total int) string {
	if total == 0 {
		return "—"
	}
	return fmt.Sprintf("%.2f%%", float64(same)*100/float64(total))
}

// viewPercent 把營火那兩張動畫寫成「兩個值都對」，不然下一個人又會把其中一個當退步。
func viewPercent(item screen) string {
	if item.ViewTotal == 0 {
		return "—"
	}
	value := percent(item.ViewSame, item.ViewTotal)
	if len(item.ViewAlternatives) == 0 {
		return value
	}
	parts := []string{value}
	for _, alternative := range item.ViewAlternatives {
		parts = append(parts, percent(alternative, item.ViewTotal))
	}
	return strings.Join(parts, " 或 ") + "（營火動畫）"
}
