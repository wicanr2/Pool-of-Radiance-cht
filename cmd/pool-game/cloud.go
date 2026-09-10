package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 臭雲術在盤面上的那一半（spec 121）。
//
// 它與其他法術不同：不是「對挑好的目標下效果」，而是在戰術盤上生一個活的
// 物件——蓋住地形、收起來要還原、彼此重疊時要重蓋。掛在人身上的效果碼
// `1Eh` 由參數表 `+0Ah` 帶進來，每次輪到那個人行動時才結算。

// tacticalBoard 讓 tacticalState 當 gamepack.TerrainWriter 用。
//
// 原版寫的是 `DS:6674h` 那張 `y × 32h + x` 的表的 `+7`，這裡就是
// `Grid.Terrain` 同一個定址（combat.TacticalRowStride）。
type tacticalBoard struct{ state *tacticalState }

func (b tacticalBoard) SetTerrain(x, y int, terrain uint8) {
	index := y*combat.TacticalRowStride + x
	if x < 0 || y < 0 || index < 0 || index >= len(b.state.Grid.Terrain) {
		return
	}
	b.state.Grid.Terrain[index] = terrain
}

// Occupied 對應原版「那一格有 `DS:6634h` 表裡的另一個物件」。
// **那張表還沒讀**（spec 121 的仍未閉合），所以這裡一律回 false：
// 收雲時每一格都還原成原地形，不會寫 ObstacleTerrain。
func (b tacticalBoard) Occupied(x, y int) bool { return false }

// footprintTerrain 取一格戰鬥者佔的四格的地形碼。
func (state *tacticalState) footprintTerrain(index int) []uint8 {
	if index <= 0 || index >= len(state.Roster) {
		return nil
	}
	cell := state.Roster[index]
	if cell.FootprintClass == 0 {
		return nil
	}
	codes := make([]uint8, 0, combat.FootprintSlots)
	for _, slot := range combat.FootprintCells(cell.FootprintClass, cell.X, cell.Y) {
		if !slot.Valid() {
			continue
		}
		code, err := state.Grid.TerrainAt(int(slot.X), int(slot.Y))
		if err != nil {
			continue
		}
		codes = append(codes, code)
	}
	return codes
}

// standingInCloud 重現 `013Dh:007Fh`（overlay-32 entry 19）對雲的那一條：
// **佔格裡只要有一格是雲，這個人就算站在雲裡**。
func (state *tacticalState) standingInCloud(index int) bool {
	return gamepack.StandingInCloud(state.footprintTerrain(index))
}

// placeCloud 生一團雲並蓋上去。caster 是施法者那一格，(x, y) 是雲心。
//
// 團數由 CountFor 數出來（原版在 `1AF6h` 走一遍串列比對 `DS:5CF0h`），
// 四格的原地形先存起來，收雲時要還原。
//
// **`Covered` 這裡取「在盤面上」**：原版節點 `+0Ch`..`+0Fh` 是逐格決定的，
// 但決定的規則還沒讀出來（spec 121）。取得寬一點的後果是雲蓋得比原版多，
// 取窄了則是玩家看不到效果；這裡先取寬，等那一段讀出來再收。
func (state *tacticalState) placeCloud(caster, x, y, casterLevel int) bool {
	cloud := gamepack.Cloud{
		Caster:  caster,
		Index:   state.Clouds.CountFor(caster) + 1,
		CentreX: x,
		CentreY: y,
	}
	covered := 0
	for slot := 0; slot < gamepack.CloudCells; slot++ {
		cellX, cellY, err := cloud.CellAt(slot)
		if err != nil {
			continue
		}
		terrain, err := state.Grid.TerrainAt(cellX, cellY)
		if err != nil {
			continue
		}
		cloud.Covered[slot], cloud.SavedTerrain[slot] = true, terrain
		covered++
	}
	if covered == 0 {
		return false
	}
	if err := cloud.Stamp(tacticalBoard{state}); err != nil {
		return false
	}
	state.Clouds = state.Clouds.Append(cloud)
	state.addCloudObjectEffect(caster, cloud.Index, casterLevel)
	return true
}

// addCloudObjectEffect 在**施法者**身上掛雲物件節點——原版走 overlay-24
// entry 10（`010Ah:0052h`），實參逐位元組讀出來是
// `(1, 等級 + 雲數 × 16, 等級, 28h, 施法者)`，反序對回
// `f(記錄, 代碼, 持續, 等級, 有收尾)` 就是**持續＝施法者等級、`+3`＝等級加
// 雲序號乘 16、`+4`＝1**（spec 121）。
//
// 雲的壽命就記在這裡：這個節點在回合邊界減到 0 時被摘掉，收尾那一下才去收雲。
//
// `+3` 的低四位只放得下 15，等級照 `EffectLevelMask` 夾；**持續不夾**——
// 那一格原版是 word，高等級施法者的雲本來就該撐比較久。
func (state *tacticalState) addCloudObjectEffect(caster, cloudIndex, casterLevel int) {
	if caster < 0 || caster >= len(state.Effects) {
		return
	}
	if casterLevel < 0 {
		casterLevel = 0
	}
	level := casterLevel
	if level > gamepack.EffectLevelMask {
		level = gamepack.EffectLevelMask
	}
	packed := uint8(level) | uint8(cloudIndex&0x0F)<<4
	state.Effects[caster] = state.Effects[caster].Append(
		gamepack.NewEffectNode(gamepack.CloudObjectEffectCode,
			uint16(casterLevel), packed, true))
}

// stinkingCloudTurn 是原版群組 15 在「輪到這個人行動」時對代碼 `1Eh` 做的事
// （overlay-12 entry 29 `0A76h`）：印一句 `is coughing`、把這一回合收掉、
// AC 變差 2 點（下限顯示 AC 10）。回傳「這一回合沒了」。
//
// **兩個條件都要成立**：身上掛著 `1Eh`（施法當下站在雲裡的人才有，
// 由 `08BCh` 依參數表 `+0Ah` 掛上），而且**現在腳下還是雲**——後者是
// overlay-24 `0B0Ch` 那一處的形狀（先 `013D:007F` 判地形，再 `010A:00A7`
// 查身上有沒有效果）。走出雲之後還會不會咳，原版哪一段管的還沒讀，
// 所以這裡取「兩個都要」的保守讀法。
func (state *tacticalState) stinkingCloudTurn(index int) bool {
	if index <= 0 || index >= len(state.Roster) {
		return false
	}
	if !state.hasEffect(index, gamepack.StinkingCloudEffectCode) || !state.standingInCloud(index) {
		return false
	}
	if index < len(state.ArmorClass) {
		state.ArmorClass[index] = gamepack.StinkingCloudArmourClass(state.ArmorClass[index])
	}
	return true
}
