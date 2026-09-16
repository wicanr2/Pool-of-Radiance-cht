package combat

import "fmt"

// 原版的部署（spec 061）：overlay-10 `1A99h` 先算兩邊的地城格偏移、填好陣型
// 樣板，再沿 combatant 串列逐筆呼叫 `1609h` 找一格放。這一份照那兩支函式與
// `START.EXE` DS `2D0h..33Fh` 的五張表重建；表的位元組在
// docs/audit/ida-start-deployment-tables.json，測試逐位元組對它。

// DeploymentSide 是 `DS:45B2h..45B9h` 裡屬於一邊的那四個 byte：地城格偏移、
// 第一腿的候選上限（`45B6h`／`45B7h`）與象限（`45B8h`／`45B9h`）。
type DeploymentSide struct {
	DX, DY        int8
	FirstLegLimit uint8
	Quadrant      uint8
}

// DeploymentSides 重現 `1A99h` 的 `1AD0h..1B91h`：我方偏移 (0, 0)、敵方偏移
// 「遭遇距離 × 朝向的單位向量」、象限是朝向 ÷ 2（敵方取反向），
// 第一腿上限是 `(人數 + 1) ÷ 2`。facing 用原版 `DS:6A0Dh` 的 0..7，
// distance 是 `[4937h]+582h`（ECL `@6DC1`）。
//
// 原版的乘法走 RTL 的長整數乘法再只取低八位；這裡的輸入範圍不會溢位，
// 但仍以 int8 截斷，保留同樣的環繞。
func DeploymentSides(facing uint8, distance int, counts [2]int) ([2]DeploymentSide, error) {
	if facing >= DirectionCount {
		return [2]DeploymentSide{}, fmt.Errorf("Pool deployment facing %d is outside 0..7", facing)
	}
	step := directionSteps[facing]
	var sides [2]DeploymentSide
	sides[1].DX = int8(int(step.X) * distance)
	sides[1].DY = int8(int(step.Y) * distance)
	sides[0].Quadrant = facing / 2
	sides[1].Quadrant = ((facing + 4) % DirectionCount) / 2
	for side := range sides {
		sides[side].FirstLegLimit = uint8((counts[side] + 1) / 2)
	}
	return sides, nil
}

// DeploymentTemplates 是 `DS:43A2h` 的執行期影像：兩邊 × 四個陣型 × 66 格。
type DeploymentTemplates [2][DeploymentsPerSet][DeploymentTemplateSize]uint8

// deploymentRowSpans 是 `DS:304h`：五組列範圍，每列 `lo, hi`，格在 `lo..hi`
// 內就是 1。前四組依象限，第五組給陣型 1。`lo=1, hi=0` 是整列為 0。
var deploymentRowSpans = [5][DeploymentTemplateRows][2]int8{
	{{1, 0}, {1, 0}, {1, 0}, {2, 9}, {3, 10}, {4, 10}},
	{{0, 2}, {0, 3}, {1, 4}, {2, 5}, {3, 6}, {4, 7}},
	{{0, 6}, {0, 7}, {1, 8}, {1, 0}, {1, 0}, {1, 0}},
	{{3, 6}, {4, 7}, {5, 8}, {6, 9}, {7, 10}, {8, 10}},
	{{0, 6}, {0, 7}, {1, 8}, {2, 9}, {3, 10}, {4, 10}},
}

// FillDeploymentTemplates 重現 `1A99h` 的 `1B93h..1C9Ah`：陣型 1 用第五組，
// 其餘用該邊象限那一組。
func FillDeploymentTemplates(sides [2]DeploymentSide) (DeploymentTemplates, error) {
	var templates DeploymentTemplates
	for side := range sides {
		if sides[side].Quadrant > 3 {
			return templates, fmt.Errorf("Pool deployment quadrant %d is outside 0..3", sides[side].Quadrant)
		}
		for formation := 0; formation < DeploymentsPerSet; formation++ {
			spans := deploymentRowSpans[sides[side].Quadrant]
			if formation == 1 {
				spans = deploymentRowSpans[4]
			}
			for row := 0; row < DeploymentTemplateRows; row++ {
				for col := 0; col < DeploymentTemplateCols; col++ {
					if int(spans[row][0]) <= col && col <= int(spans[row][1]) {
						templates[side][formation][row*DeploymentTemplateCols+col] = 1
					}
				}
			}
		}
	}
	return templates, nil
}

// `1609h` 用的四張表。
var (
	// DS:2D0h：`[象限][陣型]` 陣型 1..3 相對原地城格的方向碼；欄 0 是 8（不用）。
	deploymentNeighbourDirections = [4][DeploymentsPerSet]uint8{
		{8, 4, 6, 2}, {8, 6, 4, 0}, {8, 0, 6, 2}, {8, 2, 0, 4},
	}
	// DS:2E0h：`[象限][陣型]` 掃描軸，用時 ÷ 2。
	deploymentSweepAxis = [4][DeploymentsPerSet]uint8{
		{0, 0, 2, 6}, {2, 2, 0, 4}, {4, 4, 2, 6}, {6, 6, 4, 0},
	}
	// DS:2F0h：掃描方向環，值是 `274Ah`／`2753h` 的方向碼。
	deploymentSweepRing = [4]uint8{7, 2, 3, 6}
	// DS:2F4h／2FCh：`[陣型>0][軸]` 掃描原點的 col／row。
	deploymentOriginCol = [2][4]int8{{5, 4, 5, 6}, {3, 8, 7, 2}}
	deploymentOriginRow = [2][4]int8{{3, 2, 2, 3}, {0, 2, 5, 3}}
)

// DeploymentBoard 是 `1609h` 要問盤面的兩件事。
//
// Wall 是 `01BAh`（WallBetween）——室外（`DS:495Bh > 1`）時為 nil，那時原版
// 不查牆。PartyX／PartyY 是 `DS:6A0Bh`／`6A0Ch`。Probe 是 `14CFh` 那一步的
// 目的格探測：把這一個 combatant 放在戰術格 (x, y) 之後，以方向 8 問它佔的格
// 有沒有人、最難進的類別是什麼（overlay-32 `0CB9h`）。
type DeploymentBoard struct {
	Wall           WallProbe
	PartyX, PartyY int
	Probe          func(x, y int) (occupant uint8, class uint8, err error)
}

// Placement 是 `1609h` 一次呼叫的結果。Placed 為 false 時 X／Y 沒有意義。
type Placement struct {
	X, Y      int
	Formation int
	Placed    bool
}

// placementGuard 是 remake 自己加的迴圈上限。原版的迴圈靠「腿走出界 → 換
// 陣型 → 四個陣型用完」收斂，沒有計數；這裡加一個寬鬆的上限只為了讓
// 表或探測回錯值時報錯而不是掛住。66 格 × 4 陣型 × 兩趟都不到 1000。
const placementGuard = 4096

// PlaceCombatant 重現 overlay-10 `1609h`：從樣板中央一腿一列向兩側交替掃，
// 第一腿最多試 `FirstLegLimit − 1` 格，之後每腿最多 11 格；整塊都放不下就
// 依 `DS:2D0h` 試三個鄰接地城格（室內有牆的跳過）。找到的第一格經 `14CFh`
// 的四道判定後寫回，並清掉那一格樣板。
//
// 回傳 Placed=false 對應原版回 0：四個陣型都放不下。
func PlaceCombatant(sides [2]DeploymentSide, templates *DeploymentTemplates, side uint8,
	board DeploymentBoard, classes CellClasses) (Placement, error) {
	if side > 1 {
		return Placement{}, fmt.Errorf("Pool deployment side %d is outside 0..1", side)
	}
	if templates == nil || board.Probe == nil {
		return Placement{}, fmt.Errorf("Pool deployment needs templates and a probe")
	}
	q := int(sides[side].Quadrant)
	if q > 3 {
		return Placement{}, fmt.Errorf("Pool deployment quadrant %d is outside 0..3", q)
	}
	// `01BAh` 只有回 1 才算牆；3（另一種牆值）與 0 都當作可以過。
	walled := func(direction uint8) (bool, error) {
		if board.Wall == nil {
			return false, nil
		}
		wall, err := WallBetween(board.Wall, direction, board.PartyX+int(sides[side].DX), board.PartyY+int(sides[side].DY))
		if err != nil {
			return false, err
		}
		return wall == WallBlocking, nil
	}

	dx, dy := int(sides[side].DX), int(sides[side].DY)
	formation, leg, firstLeg, giveUp := 0, 0, true, false
	state := 1
	var col0, row0, col, row, length, visited int
	for guard := 0; ; guard++ {
		if guard >= placementGuard {
			return Placement{}, fmt.Errorf("Pool deployment scan did not converge for side %d", side)
		}
		k := int(deploymentSweepAxis[q][formation] / 2)
		origin := 0
		if formation > 0 {
			origin = 1
		}
		switch state {
		case 1:
			step := directionSteps[deploymentSweepRing[(k+2)%4]]
			col0 = int(deploymentOriginCol[origin][k]) + int(step.X)*leg
			row0 = int(deploymentOriginRow[origin][k]) + int(step.Y)*leg
			col, row = col0, row0
			length, visited, state = 1, 1, 2
		case 2:
			step := directionSteps[deploymentSweepRing[(k+1)%4]]
			col, row = col0+int(step.X)*length, row0+int(step.Y)*length
			visited++
			state = 3
		case 3:
			step := directionSteps[deploymentSweepRing[(k+3)%4]]
			col, row = col0+int(step.X)*length, row0+int(step.Y)*length
			visited++
			length++
			state = 2
		}
		// 兩道界限檢查長得像、答案不一樣：`[bp-5]` 是「任一座標出界」，
		// `149Eh` 卻是「**兩個都**出界才回 1」。一個座標出界（掃到列尾）就換腿，
		// 兩個都出界（腿走到樣板外）才換陣型。
		colOut := col < 0 || col > DeploymentTemplateCols-1
		rowOut := row < 0 || row > DeploymentTemplateRows-1
		outside, bothOut := colOut || rowOut, colOut && rowOut
		if state > 1 {
			advance := false
			switch {
			case outside && !bothOut:
				advance = true
			case firstLeg:
				advance = visited >= int(sides[side].FirstLegLimit)
			default:
				advance = visited > DeploymentTemplateCols
			}
			if advance {
				leg++
				// `18A4h..194Fh`：我方、奇數象限、陣型 0 的第一腿走完，原地城格
				// 三個鄰接方向任一沒有牆（或在室外）就再多跳一腿。
				if side == 0 && q%2 == 1 && formation == 0 && leg == 1 {
					open := false
					for j := 1; j <= 3; j++ {
						blocked, err := walled(deploymentNeighbourDirections[q][j])
						if err != nil {
							return Placement{}, err
						}
						if !blocked {
							open = true
						}
					}
					if open {
						leg++
					}
				}
				state, firstLeg = 1, false
			}
		}
		if outside && !bothOut {
			// 這一格不試（`1A4Ch` 看的是任一出界），下一圈從新的腿開始。
			continue
		}
		if outside {
			state = 0
			for formation < DeploymentsPerSet-1 && state != 1 {
				formation++
				direction := deploymentNeighbourDirections[q][formation]
				blocked, err := walled(direction)
				if err != nil {
					return Placement{}, err
				}
				if blocked {
					continue
				}
				step := directionSteps[direction]
				dx, dy = int(sides[side].DX)+int(step.X), int(sides[side].DY)+int(step.Y)
				leg, state = 0, 1
			}
			if state != 1 {
				giveUp = true
			}
		} else {
			slot := &templates[side][formation][row*DeploymentTemplateCols+col]
			x, y := DeploymentCell(dx, dy, row, col)
			occupant, class, err := board.Probe(x, y)
			if err != nil {
				return Placement{}, err
			}
			decision, err := TryPlaceCombatant(*slot, occupant, class, classes)
			if err != nil {
				return Placement{}, err
			}
			if decision == PlacementAccepted {
				*slot = 0
				return Placement{X: x, Y: y, Formation: formation, Placed: true}, nil
			}
		}
		if giveUp {
			return Placement{Formation: formation, Placed: false}, nil
		}
	}
}
