package gamepack

import "fmt"

// 盤面上的雲團物件（spec 098、spec 121）。
//
// 臭雲術不是「對目標下效果」，是在戰術盤上生一個活的物件掛在 `DS:6CAFh`
// 起頭的單向串列上。它與掛在人身上的效果不同：留在盤面、蓋住地形、
// 收起來的時候要把地形還原，而且移除一團之後其餘的雲要重新蓋一次
// ——那就是原版處理重疊的方式。

const (
	// CloudTerrain 是雲蓋住那一格時寫進戰術地形的值（overlay-12 `0F8Ah`）。
	CloudTerrain = 0x1E
	// ObstacleTerrain 是「那一格有 `DS:6634h` 表裡的另一個物件」時寫的值
	//（overlay-12 `0E5Ch`）。收雲的時候那一格不還原成原地形，寫這個。
	ObstacleTerrain = 0x1F
	// CloudCells 是一團雲蓋幾格。
	CloudCells = 4
)

// cloudDirections 是 `DS:28A7h` 的 `[1..4]`：中心、東、東南、南。
// 原版的迴圈從 1 跑到 4（`0D67h` 設 1、`0E8Eh` 比 4），所以索引 0 的 `00`
// 與索引 5 的 `05` 都不在窗內——**一團雲是 2×2 的四格，不是五格**。
var cloudDirections = [CloudCells]int{8, 2, 3, 4}

// directionDeltaX／Y 是 `DS:274Ah`／`DS:2753h` 九個方向的位移。
// 方向 0 北、1 東北、2 東、3 東南、4 南、5 西南、6 西、7 西北、8 原地。
var (
	directionDeltaX = [9]int{0, 1, 1, 1, 0, -1, -1, -1, 0}
	directionDeltaY = [9]int{-1, -1, 0, 1, 1, 1, 0, -1, 0}
)

// Cloud 是一團雲。
type Cloud struct {
	// Caster 是施法者在隊伍／怪物陣列裡的序號。原版存的是記錄的遠指標。
	Caster int
	// Index 是「這個施法者身上的第幾團雲」，原版存在節點 `+12h`，
	// 掛效果時被塞進 `+3` 的高四位。
	Index int
	// CentreX／CentreY 是雲心（節點 `+10h`／`+11h`）。
	CentreX, CentreY int
	// Covered 是四格各自蓋不蓋得住（節點 `+0Ch`..`+0Fh`）。
	Covered [CloudCells]bool
	// SavedTerrain 是四格原本的地形（節點 `+8`..`+0Bh`），收雲時還原用。
	SavedTerrain [CloudCells]uint8
}

// CellAt 回傳第 index 格的座標。
func (c Cloud) CellAt(index int) (int, int, error) {
	if index < 0 || index >= CloudCells {
		return 0, 0, fmt.Errorf("cloud cell %d is outside 0..%d", index, CloudCells-1)
	}
	direction := cloudDirections[index]
	return c.CentreX + directionDeltaX[direction], c.CentreY + directionDeltaY[direction], nil
}

// CloudList 是那一條單向串列。
type CloudList []Cloud

// Append 把一團雲掛上去。
func (l CloudList) Append(cloud Cloud) CloudList { return append(l, cloud) }

// CountFor 回傳某個施法者身上已經有幾團雲——生新的一團時要先數
//（原版在 `1AF6h` 走一遍串列比對 `DS:5CF0h`／`5CF2h`）。
func (l CloudList) CountFor(caster int) int {
	count := 0
	for _, cloud := range l {
		if cloud.Caster == caster {
			count++
		}
	}
	return count
}

// IndexOf 找某個施法者的第 index 團雲，回傳它在串列裡的位置。
func (l CloudList) IndexOf(caster, index int) int {
	for position, cloud := range l {
		if cloud.Caster == caster && cloud.Index == index {
			return position
		}
	}
	return -1
}

// TerrainWriter 是「把地形寫進戰術盤」的那一面。原版寫的是
// `DS:6674h` 那張 `y × 32h + x` 的表的 `+7`。
type TerrainWriter interface {
	SetTerrain(x, y int, terrain uint8)
	// Occupied 回傳那一格有沒有 `DS:6634h` 表裡的另一個物件。
	Occupied(x, y int) bool
}

// Stamp 把一團雲的四格蓋成 `CloudTerrain`。
func (c Cloud) Stamp(board TerrainWriter) error {
	for index := 0; index < CloudCells; index++ {
		if !c.Covered[index] {
			continue
		}
		x, y, err := c.CellAt(index)
		if err != nil {
			return err
		}
		board.SetTerrain(x, y, CloudTerrain)
	}
	return nil
}

// RemoveAt 收掉第 position 團雲：先把它蓋過的格子還原（那一格若有別的物件
// 就寫 `ObstacleTerrain`），再把剩下的雲重新蓋一次。
//
// **重新蓋是原版自己做的**（overlay-12 `0F0Ah` 之後那一段）：兩團雲重疊時，
// 收掉其中一團會把共用的那幾格還原掉，不重蓋就會在另一團中間開一個洞。
func (l CloudList) RemoveAt(position int, board TerrainWriter) (CloudList, error) {
	if position < 0 || position >= len(l) {
		return l, fmt.Errorf("cloud %d is outside 0..%d", position, len(l)-1)
	}
	cloud := l[position]
	for index := 0; index < CloudCells; index++ {
		if !cloud.Covered[index] {
			continue
		}
		x, y, err := cloud.CellAt(index)
		if err != nil {
			return l, err
		}
		if board.Occupied(x, y) {
			board.SetTerrain(x, y, ObstacleTerrain)
			continue
		}
		board.SetTerrain(x, y, cloud.SavedTerrain[index])
	}
	result := append(CloudList(nil), l[:position]...)
	result = append(result, l[position+1:]...)
	for _, remaining := range result {
		if err := remaining.Stamp(board); err != nil {
			return l, err
		}
	}
	return result, nil
}
