package gamepack

import "fmt"

// 轉變不死生物（overlay-13 entry 12 `116Ah`）。
//
// 敵方 AI 的 overlay-09 entry 2（`0203h`）走的就是這一支——**它不是施法**。
// 三條互相獨立的證據：
//
//  1. `116Ah` 用的字串常數是 `turns undead...`／`is turned`／`Is destroyed`
//     （overlay-13 code offset `1130h` 起，Turbo Pascal 的字串躺在 code 段）。
//  2. 牧師等級被壓成 1..8／9／10 三段當表列，正是 AD&D 一版轉變表的列結構
//     （1..8 各一列、9–13 一列、14 以上一列）。
//  3. 表的內容逐格就是那張矩陣：10、7、4、1、0、−1、99。
const (
	// TurnUndeadTableOffset 是轉變表在 START.EXE 裡的 DS 位移。取值是
	// `欄 × 10 + 列`，所以欄 0 那十個 byte 是前一段資料的尾巴，不屬於表。
	TurnUndeadTableOffset = 0x45B
	// TurnUndeadColumns 是不死生物的欄位數（骷髏 1 到吸血鬼 10）。
	TurnUndeadColumns = 10
	// TurnUndeadRows 是牧師等級壓成的列數。
	TurnUndeadRows = 10
	// TurnUndeadImpossible 是「這一級的牧師動不了這種不死生物」的哨兵值：
	// 1d20 永遠到不了 99。
	TurnUndeadImpossible = 99

	// UndeadTurnColumnOffset 是角色／怪物記錄裡的轉變欄位（`+76h`）。
	// 不是不死生物就是 0；巨骸骨 8、吸血鬼 10。
	UndeadTurnColumnOffset = 0x76
	// TurnUndeadDie 是判定用的骰子面數（overlay-13 `11C1h` 擲 1d20）。
	TurnUndeadDie = 20
	// TurnUndeadCountDie 是「這一次最多影響幾隻」的骰子（`11B3h` 擲 1d12）。
	TurnUndeadCountDie = 12
)

// TurnOutcome 是一次判定的結果。
type TurnOutcome int

const (
	// TurnFails 是沒轉成。
	TurnFails TurnOutcome = iota
	// TurnTurns 是轉變成功：目標的 runtime `+10h` 立起來，之後不再被挑中。
	TurnTurns
	// TurnDestroys 是直接摧毀：記錄 `+10Ch = 8`、`+10Dh = 0`，從場上移除。
	TurnDestroys
)

// TurnUndeadTable 是整張矩陣，`[欄][列]`，兩邊都是 1-based（索引 0 不用）。
type TurnUndeadTable [TurnUndeadColumns + 1][TurnUndeadRows + 1]int8

// TurnUndeadRow 把牧師等級壓成表的列號（overlay-13 `11CFh`）。
// 1..8 各自一列，9..13 併成第 9 列，其餘（含 14 以上）第 10 列。
//
// 等級 0 也會落到第 10 列，但走不到這裡——entry 2 先要求 `+96h > 0`。
func TurnUndeadRow(clericLevel int) int {
	switch {
	case clericLevel >= 1 && clericLevel <= 8:
		return clericLevel
	case clericLevel >= 9 && clericLevel <= 13:
		return 9
	}
	return 10
}

// ParseTurnUndeadTable 解出 `欄 × 10 + 列` 那 100 個 signed byte。
func ParseTurnUndeadTable(raw []byte) (TurnUndeadTable, error) {
	var table TurnUndeadTable
	// 取值是 `欄 × 10 + 列`，兩邊都到 10，所以最大索引是 110，要 111 個 byte。
	need := TurnUndeadColumns*TurnUndeadRows + TurnUndeadRows + 1
	if len(raw) < need {
		return table, fmt.Errorf("Pool turn table has %d bytes, want at least %d", len(raw), need)
	}
	for column := 1; column <= TurnUndeadColumns; column++ {
		for row := 1; row <= TurnUndeadRows; row++ {
			table[column][row] = int8(raw[column*TurnUndeadRows+row])
		}
	}
	return table, nil
}

// ReadDOSTurnUndeadTable 從 DOS ZIP 的 START.EXE 讀出這張表。
func ReadDOSTurnUndeadTable(zipPath string) (TurnUndeadTable, error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return TurnUndeadTable{}, err
	}
	start := TurnUndeadTableOffset + startDataSegmentFileDelta
	end := start + TurnUndeadColumns*TurnUndeadRows + TurnUndeadRows + 1
	if len(raw) < end {
		return TurnUndeadTable{}, fmt.Errorf("START.EXE is %d bytes, the turn table needs %d", len(raw), end)
	}
	return ParseTurnUndeadTable(raw[start:end])
}

// Threshold 取一格的門檻。欄或列出界就回 TurnUndeadImpossible。
func (t TurnUndeadTable) Threshold(clericLevel, undeadColumn int) int {
	if undeadColumn < 1 || undeadColumn > TurnUndeadColumns {
		return TurnUndeadImpossible
	}
	return int(t[undeadColumn][TurnUndeadRow(clericLevel)])
}

// Outcome 是一次判定（overlay-13 `1244h`）：擲到的點數要 **大於等於門檻的
// 絕對值**才算成功；門檻為正是「轉變」，0 或負數是「摧毀」。
//
// 所以 0 與 −1 這種格子等於自動成功再自動摧毀——AD&D 表上的 `D`。
func (t TurnUndeadTable) Outcome(clericLevel, undeadColumn, roll int) TurnOutcome {
	threshold := t.Threshold(clericLevel, undeadColumn)
	need := threshold
	if need < 0 {
		need = -need
	}
	if roll < need {
		return TurnFails
	}
	if threshold > 0 {
		return TurnTurns
	}
	return TurnDestroys
}
