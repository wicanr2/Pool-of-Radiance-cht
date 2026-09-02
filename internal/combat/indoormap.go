package combat

import "fmt"

// 牆面查詢的三種值。原版只產生 0、1、3；1 與 3 的玩家語意（實牆與門之類）
// 尚未閉合，所以這裡不給它們具象名字。
const (
	WallOpen      uint8 = 0
	WallBlocking  uint8 = 1
	WallAlternate uint8 = 3
)

// DungeonCellWalls 是 overlay-10 `0820h` 對每個地城格做的三次牆面查詢，
// 方向碼分別是 0（北）、6（西）、2（東）。
type DungeonCellWalls struct {
	North uint8
	West  uint8
	East  uint8
}

// IndoorPaint 是建構器對一個戰術格的一次落筆。Class 是建構器用的 0-based
// 類別，寫進地圖時要再經 StoredCellClass 加一。
type IndoorPaint struct {
	SubA  int
	SubB  int
	Class uint8
}

func checkWall(name string, value uint8) error {
	switch value {
	case WallOpen, WallBlocking, WallAlternate:
		return nil
	}
	return fmt.Errorf("Pool %s wall value %d is outside the original set (0, 1, 3)", name, value)
}

// PaintWestBand 重現第一支建構器 overlay-10 `02BEh`：先把 subA 2..4、
// subB 0..5 鋪成地板，再依西牆值畫上去。西牆是一條斜線，因為戰術格本身
// 是斜投影（spec 060）。
func PaintWestBand(west uint8) ([]IndoorPaint, error) {
	if err := checkWall("west", west); err != nil {
		return nil, err
	}
	paints := make([]IndoorPaint, 0, 18+9)
	for subA := 2; subA <= 4; subA++ {
		for subB := 0; subB <= 5; subB++ {
			paints = append(paints, IndoorPaint{SubA: subA, SubB: subB, Class: IndoorFloorBuilderClass})
		}
	}
	switch west {
	case WallBlocking:
		for subA := 2; subA <= 4; subA++ {
			paints = append(paints,
				IndoorPaint{SubA: subA, SubB: subA - 1, Class: 0x04},
				IndoorPaint{SubA: subA, SubB: subA, Class: 0x03},
				IndoorPaint{SubA: subA, SubB: subA + 1, Class: 0x0D},
			)
		}
	case WallAlternate:
		paints = append(paints,
			IndoorPaint{SubA: 2, SubB: 1, Class: 0x08},
			IndoorPaint{SubA: 4, SubB: 5, Class: 0x00},
		)
	}
	return paints, nil
}

// PaintNorthBand 重現第二支建構器 overlay-10 `037Bh`：subA 0..1、subB 3..4
// 的四格，北牆為 WallBlocking 時畫牆，其餘一律鋪地板。
// 注意它只分「等於 1」與「其他」，WallAlternate 走的是地板那一支。
func PaintNorthBand(north uint8) ([]IndoorPaint, error) {
	if err := checkWall("north", north); err != nil {
		return nil, err
	}
	if north == WallBlocking {
		return []IndoorPaint{
			{SubA: 0, SubB: 3, Class: 0x05},
			{SubA: 0, SubB: 4, Class: 0x05},
			{SubA: 1, SubB: 3, Class: 0x0A},
			{SubA: 1, SubB: 4, Class: 0x0A},
		}, nil
	}
	return []IndoorPaint{
		{SubA: 0, SubB: 3, Class: IndoorFloorBuilderClass},
		{SubA: 0, SubB: 4, Class: IndoorFloorBuilderClass},
		{SubA: 1, SubB: 3, Class: IndoorFloorBuilderClass},
		{SubA: 1, SubB: 4, Class: IndoorFloorBuilderClass},
	}, nil
}

func pick(open bool, whenOpen, whenClosed uint8) uint8 {
	if open {
		return whenOpen
	}
	return whenClosed
}

// PaintNorthWestCorner 重現第三支建構器 overlay-10 `0413h`：subA 0..1、
// subB 1..2 的四格，依北牆、西牆與「西北角是否通透」選類別。
//
// cornerOpen 由呼叫端以兩次牆面查詢求得：上一格的西牆與左一格的北牆
// 都沒有時為真。
func PaintNorthWestCorner(north, west uint8, cornerOpen bool) ([]IndoorPaint, error) {
	if err := checkWall("north", north); err != nil {
		return nil, err
	}
	if err := checkWall("west", west); err != nil {
		return nil, err
	}

	var topLeft uint8
	if north == WallOpen {
		switch west {
		case WallOpen:
			topLeft = IndoorFloorBuilderClass
		case WallAlternate:
			topLeft = 0x0D
		default:
			topLeft = pick(cornerOpen, 0x00, 0x0D)
		}
	} else if west == WallOpen {
		topLeft = pick(cornerOpen, 0x0F, 0x05)
	} else {
		topLeft = pick(cornerOpen, 0x12, 0x02)
	}

	var topRight uint8
	switch north {
	case WallOpen:
		topRight = IndoorFloorBuilderClass
	case WallAlternate:
		topRight = 0x11
	default:
		topRight = 0x05
	}

	var bottomLeft uint8
	switch west {
	case WallOpen:
		if north == WallOpen {
			bottomLeft = IndoorFloorBuilderClass
		} else {
			bottomLeft = pick(cornerOpen, 0x10, 0x0A)
		}
	case WallAlternate:
		bottomLeft = pick(cornerOpen, 0x14, 0x07)
	default:
		bottomLeft = pick(cornerOpen, 0x01, 0x03)
	}

	var bottomRight uint8
	if west == WallBlocking {
		switch north {
		case WallOpen:
			bottomRight = 0x0D
		case WallAlternate:
			bottomRight = 0x15
		default:
			bottomRight = 0x06
		}
	} else {
		switch north {
		case WallOpen:
			bottomRight = IndoorFloorBuilderClass
		case WallAlternate:
			bottomRight = 0x17
		default:
			bottomRight = 0x0A
		}
	}

	return []IndoorPaint{
		{SubA: 0, SubB: 1, Class: topLeft},
		{SubA: 0, SubB: 2, Class: topRight},
		{SubA: 1, SubB: 1, Class: bottomLeft},
		{SubA: 1, SubB: 2, Class: bottomRight},
	}, nil
}

// PaintNorthEastCorner 重現第四支建構器 overlay-10 `061Fh`：subA 0..1、
// subB 5..6 的四格。除了本格的北牆與東牆之外，它還讀上一格的東牆
// （aboveEast）與右一格的北牆（rightNorth）；兩者都沒有時 cornerOpen 為真。
func PaintNorthEastCorner(north, east, aboveEast, rightNorth uint8, cornerOpen bool) ([]IndoorPaint, error) {
	for name, value := range map[string]uint8{
		"north": north, "east": east, "above-east": aboveEast, "right-north": rightNorth,
	} {
		if err := checkWall(name, value); err != nil {
			return nil, err
		}
	}

	var topLeft uint8
	switch north {
	case WallOpen:
		if aboveEast == WallBlocking {
			topLeft = 0x04
		} else {
			topLeft = IndoorFloorBuilderClass
		}
	case WallAlternate:
		topLeft = 0x0F
	default:
		topLeft = 0x05
	}

	var topRight uint8
	if north == WallOpen {
		switch aboveEast {
		case WallOpen:
			topRight = IndoorFloorBuilderClass
		case WallAlternate:
			topRight = pickCorner(east == WallOpen && rightNorth != WallOpen, 0x18, 0x01)
		default:
			topRight = pickCorner(east == WallOpen && rightNorth != WallOpen, 0x0B, 0x03)
		}
	} else {
		switch {
		case east != WallOpen:
			topRight = 0x09
		case rightNorth != WallOpen:
			topRight = 0x05
		case cornerOpen:
			topRight = 0x11
		default:
			topRight = 0x13
		}
	}

	var bottomLeft uint8
	switch north {
	case WallOpen:
		bottomLeft = IndoorFloorBuilderClass
	case WallAlternate:
		bottomLeft = 0x10
	default:
		bottomLeft = 0x0A
	}

	var bottomRight uint8
	if north == WallOpen {
		switch {
		case aboveEast == WallOpen:
			bottomRight = IndoorFloorBuilderClass
		case east != WallOpen:
			bottomRight = 0x04
		case rightNorth == WallOpen:
			bottomRight = 0x08
		default:
			bottomRight = 0x0C
		}
	} else {
		switch {
		case east != WallOpen:
			bottomRight = 0x0E
		case rightNorth == WallOpen:
			bottomRight = 0x17
		default:
			bottomRight = 0x0A
		}
	}

	return []IndoorPaint{
		{SubA: 0, SubB: 5, Class: topLeft},
		{SubA: 0, SubB: 6, Class: topRight},
		{SubA: 1, SubB: 5, Class: bottomLeft},
		{SubA: 1, SubB: 6, Class: bottomRight},
	}, nil
}

func pickCorner(condition bool, whenTrue, whenFalse uint8) uint8 {
	if condition {
		return whenTrue
	}
	return whenFalse
}
