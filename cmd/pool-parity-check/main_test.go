package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func yes() *bool { value := true; return &value }
func no() *bool  { value := false; return &value }

func baseline() report {
	return report{Screens: []screen{
		{Name: "title", Label: "標題", Same: 100, Total: 100, Ratio: 1},
		{Name: "camp", Label: "紮營", Same: 80, Total: 100, Ratio: 0.8,
			ViewSame: 7744, ViewTotal: 7744, ViewRatio: 1, ViewAlternatives: []int{7068}},
		{Name: "field-cast", Label: "地圖施法", Same: 70, Total: 100, Ratio: 0.7},
	}}
}

// 沒拍到的畫面是「未量」，不是變差——擷圖停在半路（#42）時整份報表不該變成紅的。
func TestMissingScreenshotCountsAsNotMeasured(t *testing.T) {
	run := report{Screens: []screen{
		{Name: "title", Same: 100, Total: 100},
		{Name: "camp", Same: 80, Total: 100, ViewSame: 7744, ViewTotal: 7744},
		{Name: "field-cast", Captured: no()},
	}}
	differences, missing, unknown := compare(baseline(), run)
	if len(differences) != 0 || len(unknown) != 0 {
		t.Fatalf("差 %v 未知 %v", differences, unknown)
	}
	if len(missing) != 1 || missing[0] != "field-cast" {
		t.Fatalf("未量的是 %v", missing)
	}
}

// 營火是兩張動畫，紮營的「視野」兩個值都對。報成變差的話，下一個人會去翻繪圖程式碼。
func TestCampFireViewportAcceptsBothFrames(t *testing.T) {
	run := report{Screens: []screen{
		{Name: "camp", Same: 80, Total: 100, ViewSame: 7068, ViewTotal: 7744, Captured: yes()},
	}}
	differences, _, _ := compare(baseline(), run)
	if len(differences) != 0 {
		t.Fatalf("營火的另一張被當成變差：%v", differences)
	}
	run.Screens[0].ViewSame = 6000
	differences, _, _ = compare(baseline(), run)
	if len(differences) != 1 || differences[0].Field != "視野" {
		t.Fatalf("真的掉下去卻沒開口：%v", differences)
	}
}

// 整張與外框對不上就要開口，而且要說出表上是多少、這次是多少。
func TestDifferenceNamesBothNumbers(t *testing.T) {
	run := report{Screens: []screen{{Name: "title", Same: 99, Total: 100}}}
	differences, _, _ := compare(baseline(), run)
	if len(differences) != 1 {
		t.Fatalf("差異 %v", differences)
	}
	if differences[0].Sample != 100 || differences[0].Run != 99 {
		t.Fatalf("沒有同時說出兩個數字：%+v", differences[0])
	}
}

// 這一次沒量到的那幾張，基準表要保留原值——不要用缺席蓋掉既有斷言。
func TestMergeKeepsUnmeasuredScreens(t *testing.T) {
	run := report{Screens: []screen{
		{Name: "title", Same: 95, Total: 100, Ratio: 0.95},
		{Name: "field-cast", Captured: no()},
	}}
	merged := merge(baseline(), run)
	byName := map[string]screen{}
	for _, item := range merged.Screens {
		byName[item.Name] = item
	}
	if byName["title"].Same != 95 {
		t.Fatalf("量到的沒有更新：%+v", byName["title"])
	}
	if byName["field-cast"].Same != 70 {
		t.Fatalf("未量的被蓋掉了：%+v", byName["field-cast"])
	}
	if byName["title"].Label != "標題" {
		t.Fatalf("人寫的標籤被量測蓋掉了：%+v", byName["title"])
	}
}

// 表由 JSON 生成：兩份各自維護就會漂開（2026-09-18 漂了十項）。
func TestWriteTableRendersFromTheSampleJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.md")
	body := "# 標題\n\n前言\n\n" + tableStart + "\n\n舊表\n\n" + tableEnd + "\n\n後面那幾節\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeTable(path, baseline()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"| 標題 | 100.00% | — | — |", "100.00% 或 91.27%（營火動畫）", "後面那幾節"} {
		if !strings.Contains(text, want) {
			t.Fatalf("表裡沒有 %q：\n%s", want, text)
		}
	}
	if strings.Contains(text, "舊表") {
		t.Fatal("舊的表沒有被換掉")
	}
}

// 真實的基準表：每一項都要有標籤，否則生成的表會出現英文代號。
func TestRealSampleHasLabelsForEveryScreen(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dos-parity-sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result report
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Screens) == 0 {
		t.Fatal("基準表是空的")
	}
	for _, item := range result.Screens {
		if item.Label == "" {
			t.Errorf("%s 沒有中文標籤", item.Name)
		}
	}
}
