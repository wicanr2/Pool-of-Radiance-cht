package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 真正的資料要能載入且每一條都通過 schema 檢查。這一條守的是「有人手改
// docs/worklist.json 之後還能不能用」。
func TestRealWorklistLoads(t *testing.T) {
	decoded, err := load(filepath.Join("..", "..", "docs", "worklist.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Items) == 0 {
		t.Fatal("一條都沒有")
	}
	for _, layer := range layerOrder {
		if _, ok := decoded.Layers[layer]; !ok {
			t.Fatalf("layers 缺 %q，render 會少一節", layer)
		}
	}
}

// **verify 抓得到過期斷言嗎。** 這一條才是這支程式存在的理由：光看它印
// 「仍未完成」證明不了什麼，沉默相容於「機制有效」與「機制根本沒在看」。
// 所以正反兩面都要有對照。
func TestVerifyDetectsFinishedWork(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "pkg", "thing.go")
	if err := os.WriteFile(source, []byte("package pkg\n\n// remake 還沒有那個東西\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	present := item{ID: "x", Layer: "feature", Title: "t", Acceptance: "a",
		Verify: verify{Kind: "present", Paths: []string{"pkg"}, Pattern: "remake 還沒有那個東西"}}

	open, why, err := stillOpen(root, present)
	if err != nil || !open {
		t.Fatalf("自承還在就該是未完成：open=%v why=%q err=%v", open, why, err)
	}

	// 東西做好了、自承被拿掉——條目就過期了，verify 必須開口。
	if err := os.WriteFile(source, []byte("package pkg\n\n// 那個東西接好了\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	open, why, err = stillOpen(root, present)
	if err != nil {
		t.Fatal(err)
	}
	if open {
		t.Fatalf("自承不見了卻還說未完成——過期斷言就是這樣活下來的：why=%q", why)
	}
}

// absent 是反過來的：東西出現了就代表這一條動過。
func TestVerifyAbsentFlipsWhenTheThingAppears(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "pkg", "thing.go")
	if err := os.WriteFile(source, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	absent := item{ID: "y", Layer: "feature", Title: "t", Acceptance: "a",
		Verify: verify{Kind: "absent", Paths: []string{"pkg"}, Pattern: "CloudNode"}}

	if open, _, err := stillOpen(root, absent); err != nil || !open {
		t.Fatalf("還沒出現就該是未完成：open=%v err=%v", open, err)
	}
	if err := os.WriteFile(source, []byte("package pkg\n\ntype CloudNode struct{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	open, why, err := stillOpen(root, absent)
	if err != nil {
		t.Fatal(err)
	}
	if open {
		t.Fatalf("東西出現了卻還說未完成：why=%q", why)
	}
}

// json_len 綁的是數量，不是註解——註解會被順手改掉，項數不會。同樣要正反對照。
func TestVerifyJSONLenFlipsWhenTheListGrows(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data.json")
	write := func(count int) {
		entries := make([]string, count)
		for i := range entries {
			entries[i] = `{"name":"x"}`
		}
		body := `{"screens":[` + strings.Join(entries, ",") + `]}`
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	one := item{ID: "n", Layer: "presentation", Title: "t", Acceptance: "a",
		Verify: verify{Kind: "json_len", Path: "data.json", Field: "screens", Max: 2}}

	write(2)
	if open, _, err := stillOpen(root, one); err != nil || !open {
		t.Fatalf("剛好在上限就該是未完成：open=%v err=%v", open, err)
	}
	write(3)
	open, why, err := stillOpen(root, one)
	if err != nil {
		t.Fatal(err)
	}
	if open {
		t.Fatalf("清單長出第三項了卻還說未完成：why=%q", why)
	}
}

// 物件也算得出來（攻略那一條數的是 maps 底下有幾張圖）。
func TestVerifyJSONLenCountsObjectKeys(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "guide.json")
	if err := os.WriteFile(path, []byte(`{"maps":{"3/00":{},"4/21":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	one := item{ID: "g", Layer: "presentation", Title: "t", Acceptance: "a",
		Verify: verify{Kind: "json_len", Path: "guide.json", Field: "maps", Max: 1}}
	open, why, err := stillOpen(root, one)
	if err != nil {
		t.Fatal(err)
	}
	if open {
		t.Fatalf("兩張圖已經超過上限：why=%q", why)
	}
}

// **測試檔裡出現不算數。** 測試本來就會提到還沒接上的東西（為了釘住將來的
// 行為，或為了測那個資料結構本身）；把 `_test.go` 算進來，absent 會因為測試
// 裡有一行呼叫就判成「已經做了」，於是真缺口被蓋掉。
func TestVerifyIgnoresTestFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	testOnly := "package pkg\n\nfunc TestX() { list.RemoveAt(0) }\n"
	if err := os.WriteFile(filepath.Join(root, "pkg", "thing_test.go"), []byte(testOnly), 0o644); err != nil {
		t.Fatal(err)
	}
	one := item{ID: "t", Layer: "feature", Title: "t", Acceptance: "a",
		Verify: verify{Kind: "absent", Paths: []string{"pkg"}, Pattern: `\.RemoveAt\(`}}
	open, why, err := stillOpen(root, one)
	if err != nil {
		t.Fatal(err)
	}
	if !open {
		t.Fatalf("只有測試檔提到就判成做完了：why=%q", why)
	}

	// 產品程式碼裡出現才算。
	product := "package pkg\n\nfunc run() { list.RemoveAt(0) }\n"
	if err := os.WriteFile(filepath.Join(root, "pkg", "thing.go"), []byte(product), 0o644); err != nil {
		t.Fatal(err)
	}
	if open, why, _ = stillOpen(root, one); open {
		t.Fatalf("產品程式碼裡有了卻還說未完成：why=%q", why)
	}
}

// manual 沒有機器可判的訊號，一律回「仍未完成」並標出來——**沉默不等於通過**。
func TestManualStaysOpenAndSaysSo(t *testing.T) {
	open, why, err := stillOpen(t.TempDir(),
		item{ID: "z", Layer: "verification", Title: "t", Acceptance: "a", Verify: verify{Kind: "manual"}})
	if err != nil || !open || why != "要人判" {
		t.Fatalf("open=%v why=%q err=%v", open, why, err)
	}
}

func TestLoadRejectsBrokenData(t *testing.T) {
	for name, raw := range map[string]string{
		"重複 id":    `{"schema":"pool-worklist/1","layers":{"feature":"f"},"items":[{"id":"a","layer":"feature","title":"t","acceptance":"a","verify":{"kind":"manual"}},{"id":"a","layer":"feature","title":"t","acceptance":"a","verify":{"kind":"manual"}}]}`,
		"未知 layer": `{"schema":"pool-worklist/1","layers":{"feature":"f"},"items":[{"id":"a","layer":"nope","title":"t","acceptance":"a","verify":{"kind":"manual"}}]}`,
		"缺驗收":      `{"schema":"pool-worklist/1","layers":{"feature":"f"},"items":[{"id":"a","layer":"feature","title":"t","verify":{"kind":"manual"}}]}`,
		"verify 缺料": `{"schema":"pool-worklist/1","layers":{"feature":"f"},"items":[{"id":"a","layer":"feature","title":"t","acceptance":"a","verify":{"kind":"present"}}]}`,
		"schema 不對": `{"schema":"other/1","layers":{},"items":[]}`,
	} {
		path := filepath.Join(t.TempDir(), "worklist.json")
		if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := load(path); err == nil {
			t.Fatalf("%s 被接受了", name)
		}
	}
}

// render 的層順序不能跟著 map 的走訪順序跑，否則每次產出的排列都不一樣，
// diff 看起來像內容變了。
func TestRenderKeepsLayerOrder(t *testing.T) {
	decoded := &file{
		Layers: map[string]string{"feature": "功能", "verification": "驗證", "presentation": "版面"},
		Items: []item{
			{ID: "b", Layer: "presentation", Title: "版面的", Acceptance: "a", Verify: verify{Kind: "manual"}},
			{ID: "a", Layer: "feature", Title: "功能的", Acceptance: "a", Verify: verify{Kind: "manual"}},
		},
	}
	first := render(decoded)
	if strings.Index(first, "功能的") > strings.Index(first, "版面的") {
		t.Fatalf("功能那一層要排在版面前面：\n%s", first)
	}
	for i := 0; i < 5; i++ {
		if render(decoded) != first {
			t.Fatal("同一份資料 render 出不同結果")
		}
	}
}

// 多行的欄位要在清單項底下續得下去。頂到最左邊的第二段會被 markdown 讀成
// 另一個段落，清單就在那裡斷掉——而 JSON 那一側看起來完全正常。
func TestRenderIndentsContinuationLines(t *testing.T) {
	decoded := &file{
		Schema: "pool-worklist/1",
		Layers: map[string]string{"feature": "功能"},
		Items: []item{{
			ID: "x", Layer: "feature", Title: "t",
			Body:       "第一段。\n第二段。",
			BlockedBy:  "卡住的第一段。\n卡住的第二段。",
			Acceptance: "a",
			Verify:     verify{Kind: "manual"},
		}},
	}
	out := render(decoded)
	if strings.Contains(out, "\n第二段。") {
		t.Error("body 的續行頂到最左邊了")
	}
	if strings.Contains(out, "\n卡住的第二段。") {
		t.Error("blocked_by 的續行頂到最左邊了")
	}
	if !strings.Contains(out, "      第二段。") {
		t.Error("body 的續行沒有縮排")
	}
}
