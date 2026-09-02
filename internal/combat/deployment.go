package combat

import "fmt"

// 部署樣板的形狀（spec 061）。`DS:43A2h` 是執行期填的陣型遮罩：
// 每個陣型 11×6 格，每個樣板組 4 個陣型，`DS:45BAh` 選組。
const (
	DeploymentTemplateCols = 11
	DeploymentTemplateRows = 6
	DeploymentTemplateSize = DeploymentTemplateCols * DeploymentTemplateRows
	DeploymentsPerSet      = 4
)

// DeploymentOriginX 是部署投影的 X 原點。它比戰術地圖建構器的
// IndoorOriginX 大一，因為樣板的第 0 欄對到建構器的 subB 1。
const DeploymentOriginX = 22

// DeploymentTemplateIndex 重現 overlay-10 `14CFh` 的索引計算：
// `set × 108h + formation × 42h + row × 0Bh + col`。
func DeploymentTemplateIndex(set, formation, row, col int) (int, error) {
	if row < 0 || row >= DeploymentTemplateRows {
		return 0, fmt.Errorf("Pool deployment row %d is outside 0..%d", row, DeploymentTemplateRows-1)
	}
	if col < 0 || col >= DeploymentTemplateCols {
		return 0, fmt.Errorf("Pool deployment column %d is outside 0..%d", col, DeploymentTemplateCols-1)
	}
	if formation < 0 || formation >= DeploymentsPerSet {
		return 0, fmt.Errorf("Pool deployment formation %d is outside 0..%d", formation, DeploymentsPerSet-1)
	}
	if set < 0 {
		return 0, fmt.Errorf("Pool deployment template set %d is negative", set)
	}
	return set*DeploymentsPerSet*DeploymentTemplateSize +
		formation*DeploymentTemplateSize +
		row*DeploymentTemplateCols + col, nil
}

// DeploymentCell 重現 overlay-10 `14CFh` 的座標計算。它與地圖建構器用的是
// 同一個斜投影，只是 X 原點多一。
func DeploymentCell(dx, dy, row, col int) (x, y int) {
	x = DeploymentOriginX + IndoorStepXPerColumn*dx + IndoorStepXPerRow*dy + col
	y = IndoorOriginY + IndoorStepYPerRow*dy + row
	return x, y
}

// RebuildOccupancy 重現 overlay-32 `03A2h` 的前半：先把整個佔用格陣列清成 0，
// 再依每個 combatant 的體型展開四個佔格，寫上它的索引。
//
// cells 是 1-based 的位置表，第 0 筆保留。體型類別為 0 的不參與。
// 落在盤面外的佔格會被跳過——原版的 footprint 查詢對這種情形回 false。
func RebuildOccupancy(cells []CombatantCell) ([]uint8, error) {
	occupancy := make([]uint8, TacticalMapCellCount)
	for index := 1; index < len(cells); index++ {
		cell := cells[index]
		if cell.FootprintClass == 0 {
			continue
		}
		for _, slot := range FootprintCells(cell.FootprintClass, cell.X, cell.Y) {
			if !slot.Valid() {
				continue
			}
			if !withinTactical(slot.X, slot.Y) {
				continue
			}
			position := int(slot.Y)*TacticalRowStride + int(slot.X)
			if position >= len(occupancy) {
				return nil, fmt.Errorf("Pool combatant %d occupies (%d,%d), which is outside the grid",
					index, slot.X, slot.Y)
			}
			occupancy[position] = uint8(index)
		}
	}
	return occupancy, nil
}

// ViewportOrigin 是戰術地圖 record 的 `+2`／`+3`：畫面左上角對應的戰術格。
type ViewportOrigin struct {
	X uint8
	Y uint8
}

// ScreenRelativeCells 重現 overlay-32 `03A2h` 的後半與 `0068h`：把每個
// combatant 的座標減去畫面原點，寫進兩個以索引定址的 byte 陣列
// （原版是 `DS:5FA8h` 與 `DS:5FF0h`）。減法照原版以位元組進行。
func ScreenRelativeCells(cells []CombatantCell, origin ViewportOrigin) (offsetsX, offsetsY []uint8) {
	offsetsX = make([]uint8, len(cells))
	offsetsY = make([]uint8, len(cells))
	for index := 1; index < len(cells); index++ {
		offsetsX[index] = cells[index].X - origin.X
		offsetsY[index] = cells[index].Y - origin.Y
	}
	return offsetsX, offsetsY
}

// PlacementRejection 說明一次部署嘗試為什麼失敗，對應 overlay-10 `14CFh`
// 的四個失敗出口。
type PlacementRejection int

const (
	PlacementAccepted PlacementRejection = iota
	PlacementOutsideTemplate
	PlacementTemplateSlotUsed
	PlacementCellOccupied
	PlacementCellNotEnterable
)

// TryPlaceCombatant 重現 overlay-10 `14CFh` 的判定順序：先看樣板允不允許，
// 再用方向 8（原地）跑一次目的格探測，要求沒有人佔著、類別不是 0、
// 且該類別的 EntryThreshold 不是 0FFh。成功時回報要寫入的座標。
//
// 原版成功後會把樣板格清零，讓同一格不會被用第二次；那一步由呼叫端在
// 拿到 PlacementAccepted 之後執行，這裡不改動輸入。
func TryPlaceCombatant(templateSlot uint8, occupant uint8, class uint8, classes CellClasses) (PlacementRejection, error) {
	if templateSlot == 0 {
		return PlacementTemplateSlotUsed, nil
	}
	if occupant != 0 {
		return PlacementCellOccupied, nil
	}
	if class == OffBoardDestinationClass {
		return PlacementCellNotEnterable, nil
	}
	record, err := CellClassAt(classes, class)
	if err != nil {
		return PlacementCellNotEnterable, err
	}
	if record.EntryThreshold >= 0xFF {
		return PlacementCellNotEnterable, nil
	}
	return PlacementAccepted, nil
}
