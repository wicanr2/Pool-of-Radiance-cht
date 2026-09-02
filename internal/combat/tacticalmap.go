package combat

// 戰術地圖的形狀（spec 060）。原版在戰鬥開始時以 GetMem 配置
// TacticalMapSize bytes，前 TacticalMapHeaderSize 個 byte 是標頭，
// 其後是 50×25 的格位類別。
const (
	TacticalMapHeaderSize = 7
	TacticalMapWidth      = TacticalRowStride
	TacticalMapHeight     = TacticalMaxY + 1
	TacticalMapCellCount  = TacticalMapWidth * TacticalMapHeight
	TacticalMapSize       = TacticalMapHeaderSize + TacticalMapCellCount
)

// OpenGroundCellClass 是室外戰場整面填入的格位類別。它同時是目的格探測的
// 初值 DefaultDestinationClass——兩處都是 17h，不是巧合。
const OpenGroundCellClass = 0x17

// NewOutdoorTacticalGrid 重現 overlay-10 `1255h` 的第一步：整面 1250 格填成
// OpenGroundCellClass，標頭的 `+4`、`+5`、`+6` 依 `12E5h` 分別設為 0、1、0。
// `+6` 為 0 表示地形判定生效。
//
// 原版接著會依隊伍在大地圖的位置挑背景並跑四支細節建構器，那一段尚未閉合
// （spec 060），所以這裡只產生平坦戰場。
func NewOutdoorTacticalGrid() TacticalGrid {
	terrain := make([]uint8, TacticalMapCellCount)
	for index := range terrain {
		terrain[index] = OpenGroundCellClass
	}
	return TacticalGrid{IgnoreTerrain: false, Terrain: terrain}
}

// TacticalMapHeader 是配置之後立刻寫下的三個標頭 byte。
// 其餘四個 byte 在這一步沒有被寫，語意未定。
type TacticalMapHeader struct {
	Field4 uint8
	Field5 uint8
	Field6 uint8
}

// NewTacticalMapHeader 回傳 `12E5h` 寫下的固定值。
func NewTacticalMapHeader() TacticalMapHeader {
	return TacticalMapHeader{Field4: 0, Field5: 1, Field6: 0}
}

// IndoorWindow 是 overlay-10 `0820h` 由地城幾何生成戰場時掃過的視窗：
// 相對隊伍位置 X 由 -6 到 +6、Y 由 -2 到 +2。
const (
	IndoorWindowMinX = -6
	IndoorWindowMaxX = 6
	IndoorWindowMinY = -2
	IndoorWindowMaxY = 2
)

// IndoorWindowCells 依原版的迴圈順序列出視窗內的相對座標：
// 外層是 Y、內層是 X，兩層都由負值遞增。
func IndoorWindowCells() [][2]int {
	cells := make([][2]int, 0,
		(IndoorWindowMaxY-IndoorWindowMinY+1)*(IndoorWindowMaxX-IndoorWindowMinX+1))
	for dy := IndoorWindowMinY; dy <= IndoorWindowMaxY; dy++ {
		for dx := IndoorWindowMinX; dx <= IndoorWindowMaxX; dx++ {
			cells = append(cells, [2]int{dx, dy})
		}
	}
	return cells
}
