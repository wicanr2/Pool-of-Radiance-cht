package main

import (
	"path/filepath"
	"strconv"
	"testing"
)

// 這一條把 spec 101 的圖釘住。世界的接法變了要嘛是解碼退步，要嘛是原始
// 資料換了——兩種都該讓測試紅。
func TestWorldGraphShape(t *testing.T) {
	result, err := build(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	if len(result.ECLBlocks) != 29 || len(result.GEOBlocks) != 29 {
		t.Fatalf("區塊數 ECL %d GEO %d，預期各 29",
			len(result.ECLBlocks), len(result.GEOBlocks))
	}
	targets := map[string][]string{}
	for _, block := range result.ECLBlocks {
		key := blockKey(block.Archive, block.BlockID)
		targets[key] = block.NewECL
	}
	// 26 是接得最廣的那一個：一個區塊接十一個。
	if got := len(targets["7/26"]); got != 11 {
		t.Errorf("ecl7/26 接到 %d 個區塊，預期 11：%v", got, targets["7/26"])
	}
	// 城區只從碼頭那一段接得到 21/26/27，其餘是室內場景與貧民窟。
	want := []string{"8", "11", "20", "21", "26", "27"}
	if got := targets["3/0"]; !sameStrings(got, want) {
		t.Errorf("ecl3/0 接到 %v，預期 %v", got, want)
	}
	exits := map[string][]geoExit{}
	for _, block := range result.GEOBlocks {
		exits[blockKey(block.Archive, block.BlockID)] = block.Exits
	}
	// 城區起點 (0,4) 往西那一個出口，是這一張圖上唯一站得到的邊界出口。
	city := exits["3/0"]
	if len(city) != 28 {
		t.Fatalf("geo3/0 邊界出口 %d 個，預期 28", len(city))
	}
	var start geoExit
	for _, exit := range city {
		if exit.X == 0 && exit.Y == 4 && exit.Facing == "W" {
			start = exit
		}
	}
	if start.Facing == "" {
		t.Fatal("geo3/0 沒有 (0,4)W 這個出口")
	}
	same := 0
	for _, exit := range city {
		if exit.Component == start.Component {
			same++
		}
	}
	if same != 1 {
		t.Errorf("geo3/0 與起點同一個連通元件的出口有 %d 個，預期 1", same)
	}
}

func blockKey(archive, block int) string {
	return strconv.Itoa(archive) + "/" + strconv.Itoa(block)
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

// 六個區塊的 NEWECL 目標存在變數裡。這一條釘住「餵值來源量得出來」：位址、
// 索引變數與表頭位元組任何一項變了都該紅——那代表解碼退步或原始資料換了。
// 表頭是原始位元組，不是解讀；解讀（哪幾格算數、對到哪個封存檔）在 spec 101。
func TestIndirectNewECLSourcesAreMeasured(t *testing.T) {
	result, err := build(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	type want struct {
		variable string
		table    string
		index    string
		head     string
	}
	expected := map[string]want{
		"1/24": {"@6E7D", "@99B0", "@6E82", "FFFFFFFF0E1AFFFF"},
		"2/15": {"@6E7F", "@9B09", "@C04D", "1D00000208000004"},
		"5/3":  {"@6E79", "@9AA6", "@C04D", "0404060602015F9A"},
		"8/29": {"@6E82", "@AFCA", "@C04D", "01140F1201020300"},
	}
	// ecl2/9 與 ecl5/6 沒有表，編號是 SAVE 立即值進去的。
	immediateOnly := map[string]bool{"2/9": true, "5/6": true}
	seen := map[string]bool{}
	for _, block := range result.ECLBlocks {
		key := blockKey(block.Archive, block.BlockID)
		for _, indirect := range block.Indirect {
			seen[key] = true
			if immediateOnly[key] {
				if len(indirect.Immediates) == 0 {
					t.Errorf("ecl%s 的 %s 一個立即值都沒收到", key, indirect.Address)
				}
				continue
			}
			row, ok := expected[key]
			if !ok {
				t.Errorf("ecl%s 多出一個間接 NEWECL %s（變數 %s）",
					key, indirect.Address, indirect.Variable)
				continue
			}
			if indirect.Variable != row.variable {
				t.Errorf("ecl%s 的變數是 %s，預期 %s", key, indirect.Variable, row.variable)
			}
			found := false
			for _, table := range indirect.Tables {
				if table.Address != row.table {
					continue
				}
				found = true
				if table.Index != row.index {
					t.Errorf("ecl%s 表 %s 的索引是 %s，預期 %s",
						key, row.table, table.Index, row.index)
				}
				if table.Head != row.head {
					t.Errorf("ecl%s 表 %s 的表頭是 %s，預期 %s",
						key, row.table, table.Head, row.head)
				}
			}
			if !found {
				t.Errorf("ecl%s 沒有表 %s：%v", key, row.table, indirect.Tables)
			}
		}
	}
	for key := range expected {
		if !seen[key] {
			t.Errorf("ecl%s 沒有間接 NEWECL 了", key)
		}
	}
	for key := range immediateOnly {
		if !seen[key] {
			t.Errorf("ecl%s 沒有間接 NEWECL 了", key)
		}
	}
}

// 把工具量到的立即值出邊，加上 spec 101 從上面那些表頭解出來的六條間接出邊，
// 從區塊 0 做廣度優先——29 個區塊全部到得了。缺的那 12 個因此不是被機制擋住
// 的，是探索器還沒走到。立即值那一半來自工具，所以解碼退步這一條也會紅。
func TestEveryECLBlockIsReachableFromTheCity(t *testing.T) {
	result, err := build(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	// 解讀過的間接出邊（spec 101 的表）：表頭位元組由上一條測試釘住。
	resolved := map[int][]int{
		24: {14, 26},
		9:  {6, 18},
		15: {29, 2},
		3:  {4, 6},
		6:  {3, 9},
		29: {20, 15, 18},
	}
	edges := map[int][]int{}
	blocks := map[int]bool{}
	for _, block := range result.ECLBlocks {
		blocks[block.BlockID] = true
		for _, target := range block.NewECL {
			id, err := strconv.Atoi(target)
			if err != nil {
				continue // `@6E79` 這種，改由 resolved 提供
			}
			edges[block.BlockID] = append(edges[block.BlockID], id)
		}
		edges[block.BlockID] = append(edges[block.BlockID], resolved[block.BlockID]...)
	}
	reached := map[int]bool{0: true}
	queue := []int{0}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range edges[current] {
			if !blocks[next] {
				t.Errorf("區塊 %d 指到不存在的區塊 %d", current, next)
				continue
			}
			if !reached[next] {
				reached[next] = true
				queue = append(queue, next)
			}
		}
	}
	var missing []int
	for id := range blocks {
		if !reached[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) != 0 {
		t.Errorf("從區塊 0 走不到 %v", missing)
	}
	if len(reached) != len(blocks) {
		t.Errorf("走到 %d 個區塊，總共 %d 個", len(reached), len(blocks))
	}
}
