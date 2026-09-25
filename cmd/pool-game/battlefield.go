package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// riverClearedAddress 是 ECL `@4AB3`（`[4933h]+366h`）。overlay-10 `08B4h`
// 在它等於 FFh 時把帶 80h 的地形旗標換掉（spec 060）。市政廳公告表第 13 筆
// 掛在同一個位址上（清完斯托約諾河那一則，spec 041），但 FFh 的確切語意
// 沒有讀到文字證據，所以名字只說「河」不說「清完」以外的事。
const riverClearedAddress = 0x4AB3

// combatAreaMode 重現 overlay-03 主迴圈頂端 `3653h..36DBh` 寫的 `DS:495Bh`：
// 預設 1，目前的 ECL 區塊是 25／26／27 而且 `@49E6` 為 0（站在野外地形上）
// 時是 2／3／4。原版每一圈都重算，所以戰鬥開打時取當下的區塊與旗標就是
// 那一圈的值。
func (a *app) combatAreaMode() uint8 {
	if a.eventSession == nil || a.eventMachine == nil {
		return gamepack.CombatAreaIndoor
	}
	return gamepack.CombatAreaMode(a.eventSession.CurrentBlockID(),
		a.eventMachine.Memory[encounterWalkFlagAddress])
}

// generateTacticalGrid 重現 overlay-10 `12E5h` 的分岔：`DS:495Bh` 等於 1 走室內
// `0820h`（由地城牆面生成），否則走室外 `1255h`（平地＋四支細節建構器）。
func (a *app) generateTacticalGrid() (combat.TacticalGrid, error) {
	mode := a.combatAreaMode()
	if mode == gamepack.CombatAreaIndoor {
		return combat.GenerateIndoorTacticalGrid(int(a.spawn.X), int(a.spawn.Y),
			geoWallProbe(a.initialMap.Grid, int(a.spawn.Y)))
	}
	if a.wildernessTerrain == nil {
		return combat.TacticalGrid{}, fmt.Errorf("Pool wilderness terrain table is not loaded")
	}
	memory := a.eventMachine.Memory
	background, err := a.wildernessTerrain.Background(mode,
		int(memory[wildernessX]), int(memory[wildernessY]))
	if err != nil {
		return combat.TacticalGrid{}, err
	}
	return combat.GenerateOutdoorTacticalGrid(combat.OutdoorBattlefield{
		Terrain:      background,
		RiverCleared: memory[riverClearedAddress] == 0xFF,
		Block:        a.eventSession.CurrentBlockID(),
	}, gamepack.OriginalCombatCellClassTable(), a.rollDice)
}

// deploymentWallProbe 是部署 `1609h` 要問的牆。室外（`495Bh > 1`）原版不查牆，
// 這裡回 nil（combat.DeploymentBoard 的約定）。
func (a *app) deploymentWallProbe(grid combat.TacticalGrid) combat.WallProbe {
	if grid.Outdoor {
		return nil
	}
	return geoWallProbe(a.initialMap.Grid, int(a.spawn.Y))
}
