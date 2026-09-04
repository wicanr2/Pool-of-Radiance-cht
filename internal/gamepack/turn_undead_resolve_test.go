package gamepack_test

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// scriptedRoller 依序吐出排好的點數，用完就一直吐最後一個。
// 轉變只擲兩次（1d12 的額度、1d20 的點數），所以順序本身就是被測的東西。
type scriptedRoller struct {
	values []int
	next   int
}

func (r *scriptedRoller) Roll(count, sides int) int {
	if r.next < len(r.values) {
		value := r.values[r.next]
		r.next++
		return value
	}
	return r.values[len(r.values)-1]
}

func turnTable(t *testing.T) gamepack.TurnUndeadTable {
	t.Helper()
	table, err := gamepack.ReadDOSTurnUndeadTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	return table
}

func TestSelectTurnUndeadTargetTakesTheSmallestColumn(t *testing.T) {
	candidates := []gamepack.TurnUndeadCandidate{
		{Column: 0},                // 不是不死生物
		{Column: 8},                // 木乃伊
		{Column: 1, Turned: true},  // 已經被轉變過
		{Column: 3},                // 餓鬼：最小的合格者
		{Column: 2, Removed: true}, // 已經摧毀離場
		{Column: 3},                // 同分，不換人
	}
	index, ok := gamepack.SelectTurnUndeadTarget(candidates)
	if !ok || index != 3 {
		t.Fatalf("挑到索引 %d（%v），應該是 3", index, ok)
	}

	// 負對照：全部不合格就挑不到，而且 116Ah 就是靠這個回傳值收工。
	none := []gamepack.TurnUndeadCandidate{{Column: 0}, {Column: 5, Turned: true}}
	if _, ok := gamepack.SelectTurnUndeadTarget(none); ok {
		t.Error("沒有合格的候選時仍然挑到了目標")
	}
}

// 欄位上界是 13，但表只有 10 欄。原版讀 `45Bh + 欄 × 10 + 列` 會落到表外，
// 這裡回「不可能」；能這樣接是因為原版的記錄沒有一隻超過 10。
func TestSelectTurnUndeadTargetKeepsTheOriginalUpperBound(t *testing.T) {
	if gamepack.TurnUndeadColumnLimit != 13 {
		t.Fatalf("欄位上界是 %d，原版 `[bp-4]` 的初值是 0Dh", gamepack.TurnUndeadColumnLimit)
	}
	if _, ok := gamepack.SelectTurnUndeadTarget([]gamepack.TurnUndeadCandidate{{Column: 13}}); ok {
		t.Error("欄位等於上界時不該被挑中（`13D1h` 是 jge）")
	}
	if _, ok := gamepack.SelectTurnUndeadTarget([]gamepack.TurnUndeadCandidate{{Column: 12}}); !ok {
		t.Error("欄位 12 小於上界，原版挑得到")
	}
}

func TestUndeadTurnColumnStaysInsideTheTable(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	scanned, undead, highest := 0, 0, 0
	for archive := uint8(1); archive <= 8; archive++ {
		for id := 0; id < 256; id++ {
			record, err := gamepack.ReadDOSMonsterRecord(zipPath, archive, uint8(id))
			if err != nil {
				continue
			}
			scanned++
			column := int(record.Raw[gamepack.UndeadTurnColumnOffset])
			if column == 0 {
				continue
			}
			undead++
			if column > highest {
				highest = column
			}
			if column > gamepack.TurnUndeadColumns {
				t.Errorf("%s 的 +76h 是 %d，超出表的 %d 欄",
					record.Name, column, gamepack.TurnUndeadColumns)
			}
		}
	}
	if scanned == 0 {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	// 正對照：真的掃到不死生物，而且最大的欄位剛好用滿整張表。
	if undead == 0 {
		t.Fatalf("掃了 %d 筆記錄卻一隻不死生物都沒有，這一則的掃描面有洞", scanned)
	}
	if highest != gamepack.TurnUndeadColumns {
		t.Errorf("最大的 +76h 是 %d，表有 %d 欄——沒用滿或超界",
			highest, gamepack.TurnUndeadColumns)
	}
	t.Logf("掃了 %d 筆記錄，其中 %d 隻是不死生物，最大欄位 %d", scanned, undead, highest)
}

// 一群骷髏對上第一級牧師：門檻 10，同一個 1d20 對每一隻都適用。
func TestResolveTurnUndeadSharesOneRoll(t *testing.T) {
	table := turnTable(t)
	candidates := []gamepack.TurnUndeadCandidate{{Column: 1}, {Column: 1}, {Column: 1}}
	roller := &scriptedRoller{values: []int{3, 10}} // 額度 3、點數 10
	result := gamepack.ResolveTurnUndead(table, 1, candidates, roller)

	if result.Roll != 10 || result.Allowance != 3 || result.Row != 1 {
		t.Fatalf("點數 %d 額度 %d 列 %d，應該是 10／3／1",
			result.Roll, result.Allowance, result.Row)
	}
	if len(result.Events) != 3 {
		t.Fatalf("轉到 %d 隻，額度 3 而且每一隻都擲得過，應該是 3", len(result.Events))
	}
	for _, event := range result.Events {
		if event.Outcome != gamepack.TurnTurns || event.Threshold != 10 {
			t.Errorf("骷髏對第一級牧師是門檻 10 的轉變，得到門檻 %d 結果 %v",
				event.Threshold, event.Outcome)
		}
	}
	for index, candidate := range candidates {
		if !candidate.Turned || candidate.Removed {
			t.Errorf("第 %d 隻的旗標是 turned=%v removed=%v，轉變只立 +10h",
				index, candidate.Turned, candidate.Removed)
		}
	}
	if !result.Turned() {
		t.Error("轉到了三隻，Turned() 卻是假")
	}
}

// 差一點就是零：門檻 10 擲出 9 什麼都不會發生，而且整次就此收工。
func TestResolveTurnUndeadStopsAtTheFirstMiss(t *testing.T) {
	table := turnTable(t)
	candidates := []gamepack.TurnUndeadCandidate{{Column: 1}, {Column: 1}}
	roller := &scriptedRoller{values: []int{12, 9}}
	result := gamepack.ResolveTurnUndead(table, 1, candidates, roller)

	if len(result.Events) != 0 || result.Turned() {
		t.Fatalf("擲出 9 沒到門檻 10，卻轉到了 %d 隻", len(result.Events))
	}
	if !result.Stopped {
		t.Error("判定失敗要立起收工旗標（`1313h`）")
	}
	if candidates[0].Turned || candidates[1].Turned {
		t.Error("失敗不該動任何一隻的 +10h")
	}
}

// 門檻是 0 或負數就自動成功並且直接摧毀，不必擲得到。
func TestResolveTurnUndeadDestroysOnNonPositiveThresholds(t *testing.T) {
	table := turnTable(t)
	if got := table.Threshold(8, 1); got != -1 {
		t.Fatalf("第八級牧師對骷髏的門檻是 %d，原版是 −1", got)
	}
	candidates := []gamepack.TurnUndeadCandidate{{Column: 1}}
	roller := &scriptedRoller{values: []int{1, 1}} // 額度 1、點數 1：最差的一擲
	result := gamepack.ResolveTurnUndead(table, 8, candidates, roller)

	if len(result.Events) != 1 || result.Events[0].Outcome != gamepack.TurnDestroys {
		t.Fatalf("門檻 −1 擲出 1 也該摧毀，得到 %+v", result.Events)
	}
	if !candidates[0].Removed || candidates[0].Turned {
		t.Errorf("摧毀立的是離場旗標，得到 turned=%v removed=%v",
			candidates[0].Turned, candidates[0].Removed)
	}
}

// 配額是一個下限：額度只擲到 1，但門檻是負數的那些會一直把額度補回來，
// 最多補到配額用完——淨效果是自動摧毀時至少處理六隻。
func TestResolveTurnUndeadQuotaFloorsAutomaticDestruction(t *testing.T) {
	table := turnTable(t)
	candidates := make([]gamepack.TurnUndeadCandidate, 10)
	for index := range candidates {
		candidates[index].Column = 1 // 骷髏
	}
	roller := &scriptedRoller{values: []int{1, 1}} // 額度 1、點數 1
	result := gamepack.ResolveTurnUndead(table, 8, candidates, roller)

	if len(result.Events) != gamepack.TurnUndeadQuota {
		t.Fatalf("額度只有 1，配額 %d 應該把它撐到 %d 隻，實際 %d 隻",
			gamepack.TurnUndeadQuota, gamepack.TurnUndeadQuota, len(result.Events))
	}
	// 負對照：門檻剛好是 0 的那一格也算摧毀，卻補不回額度（`1303h` 是 jge），
	// 所以同樣的盤面只會處理一隻。
	if got := table.Threshold(6, 1); got != 0 {
		t.Fatalf("第六級牧師對骷髏的門檻是 %d，原版是 0", got)
	}
	zeroed := make([]gamepack.TurnUndeadCandidate, 10)
	for index := range zeroed {
		zeroed[index].Column = 1
	}
	zeroResult := gamepack.ResolveTurnUndead(table, 6, zeroed, &scriptedRoller{values: []int{1, 1}})
	if len(zeroResult.Events) != 1 {
		t.Errorf("門檻 0 補不回額度，額度 1 就只處理一隻，實際 %d 隻",
			len(zeroResult.Events))
	}
}

// 轉變過的不再被挑中，所以同一群不死生物不會被重複處理。
func TestResolveTurnUndeadNeverRepeatsATarget(t *testing.T) {
	table := turnTable(t)
	candidates := []gamepack.TurnUndeadCandidate{{Column: 1}, {Column: 1}}
	roller := &scriptedRoller{values: []int{12, 20}} // 額度 12：比目標還多
	result := gamepack.ResolveTurnUndead(table, 1, candidates, roller)

	if len(result.Events) != 2 {
		t.Fatalf("兩隻骷髏應該各處理一次，實際 %d 次", len(result.Events))
	}
	if result.Events[0].Index == result.Events[1].Index {
		t.Error("同一隻被處理了兩次")
	}
	if result.Stopped {
		t.Error("挑不到目標時是正常收工，不該立收工旗標")
	}
}
