package main

import (
	"fmt"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 射擊與彈藥（spec 151，issue #98）：玩家與電腦兩側共用同一組規則，全部照原版的
// 武器槽 `+0CCh`、`+0F8h`、`+0FCh`（gamepack.MissileGearOf）。
//
//	攻擊次數  overlay-13 `0D29h`：射擊武器有彈藥時一回合的次數是型別表 `+05h`，再以彈藥數量封頂
//	瞄準      overlay-13 `2B24h..2B8Bh`：射擊武器沒彈藥、或身邊有敵人而武器不能近戰，就沒有 Target
//	撞上去    overlay-08 `0D36h..0D68h`：射擊武器（不能近戰的）印 "Not with that weapon"，不打
//	射出去    overlay-13 `2CFAh..2D64h`（玩家）、overlay-09 `0EB3h..0F1Bh`（電腦）：彈藥交給攻擊包裝，
//	          丟得出去又能近戰的武器貼身打時不算射（彈藥 NULL）
//	扣彈藥    overlay-13 `19D5h..1A96h`：扣實際射出的發數，用完就從物品鏈摘掉，匕首之類落在地上
//	電腦換槍  overlay-09 entry 9（`13D5h`）：每回合接近之前、以及射擊武器貼身時重挑武器與盾

const (
	// msgStatusAvoidsMissile 是 overlay-12 `0FADh` 的 "Avoids it"（`29h` 防護普通飛彈）。
	msgStatusAvoidsMissile messageID = iota + 3980
	// msgStatusNotWithWeapon 是 overlay-08 `0D1Bh` 的 "Not with that weapon"。
	msgStatusNotWithWeapon
	// msgAimNoTarget 是瞄準時這一格沒有 Target 那一項（overlay-13 `2B33h`／`2B7Bh` 跳過
	// `2B7Dh`）。原版只是選單上少一個字，remake 按 Enter 時用這一句說明為什麼不能打。
	msgAimNoTarget
)

func init() {
	for id, key := range map[messageID]string{
		msgStatusAvoidsMissile: "ui.statusAvoidsMissile",
		msgStatusNotWithWeapon: "ui.statusNotWithWeapon",
		msgAimNoTarget:         "ui.aimNoTarget",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// partyNaturalAIScore 是玩家建的角色的 `+0A3h × +0A5h`：建角寫下 1 與 2（overlay-16 `1C02h`
// 那一段，spec 065〈不做〉），`+0A7h` 沒寫，是 0。
const partyNaturalAIScore = 2

func rawItems(items []poolsave.Item) [][]byte {
	raws := make([][]byte, len(items))
	for index := range items {
		raws[index] = items[index].Raw
	}
	return raws
}

// missileGear 讀這一格的射擊狀態與它的物品鏈（隊員從隊伍、怪物從 FoeItems，同 foeItemBearer）。
func (a *app) missileGear(state *tacticalState, index uint8) (gamepack.MissileGear, []poolsave.Item, int, error) {
	items, slot := a.foeItemBearer(state, index)
	gear, err := gamepack.MissileGearOf(rawItems(items), a.itemTypes)
	return gear, items, slot, err
}

// storeCombatItems 把改過的物品鏈寫回去，接著照原版重算（overlay-25 entry 7）。
//
// NPC 的戰鬥數值直接讀它帶的記錄（applyNPCCombatStats），物品改了先不重算。
func (a *app) storeCombatItems(state *tacticalState, index int, slot int, items []poolsave.Item) error {
	if slot >= 0 {
		member := &a.state.Party[slot]
		member.Inventory = items
		syncTrainedLibraryCharacter(&a.state, *member)
		if member.NPC {
			return nil
		}
		return a.applyPartyGearStats(state, index, *member)
	}
	if state.FoeItems == nil {
		state.FoeItems = map[int][]poolsave.Item{}
	}
	state.FoeItems[index] = items
	if monster, ok := a.stagedMonsterFor(index, state.Friendly); ok {
		return a.applyMonsterGearStats(state, index, monster.Record)
	}
	return nil
}

// adjacentEnemies 是 overlay-25 entry 32（`246Dh`）以 1 為預算：身邊有沒有對面的人。
func (state *tacticalState) adjacentEnemies(mover uint8) (bool, error) {
	side, ok := state.sideOf(mover)
	if !ok {
		return false, nil
	}
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return false, err
	}
	here := state.Roster[mover]
	nearby, err := combat.OpposingNearbyAt(snapshot, mover, here.X, here.Y, 1, 1-side, state.sideOf)
	return len(nearby) != 0, err
}

// aimOffersTarget 是 overlay-13 `2B24h..2B8Bh`：這一格拿現在的武器能不能對別人按 Target。
// 射程與「是不是自己」在呼叫端判（`2AEFh`、`2B15h`）。
func (a *app) aimOffersTarget(state *tacticalState, mover uint8) (bool, error) {
	gear, _, _, err := a.missileGear(state, mover)
	if err != nil || !gear.Ranged {
		return true, err
	}
	if !gear.CanFire {
		return false, nil
	}
	adjacent, err := state.adjacentEnemies(mover)
	if err != nil || !adjacent {
		return true, err
	}
	return gear.ThrownMelee, nil
}

// bumpAttackRefused 是 overlay-08 `0D36h..0D68h`：走進敵人那一格時，手上是射擊武器而且
// 不能近戰就印 "Not with that weapon"、不打。
func (a *app) bumpAttackRefused(state *tacticalState, mover uint8) (bool, error) {
	gear, _, _, err := a.missileGear(state, mover)
	if err != nil {
		return false, err
	}
	return gear.Ranged && !gear.ThrownMelee, nil
}

// resolveWeaponAttack 是攻擊包裝 overlay-13 `1883h` 連同它前面挑彈藥的那一段。fire 為 true 是
// 瞄準（`2C17h`）與電腦（overlay-09 `0EB3h`）那兩條，會帶彈藥；走進敵人那一格（overlay-08
// `0DD2h`）傳的是 NULL，fire 為 false。
func (a *app) resolveWeaponAttack(state *tacticalState, target uint8, fire bool) error {
	mover := state.Mover
	gear, items, slot, err := a.missileGear(state, mover)
	if err != nil {
		return err
	}
	ammunition := -1
	if fire && gear.Ranged && gear.CanFire {
		ammunition = gear.Ammunition
		if distance, _ := state.tacticalRange(mover, target); gear.ThrownMelee && distance == 1 {
			ammunition = -1
		}
	}
	swings, form2, err := a.volleySwings(state, mover, gear, items)
	if err != nil {
		return err
	}
	if err := a.resolveAttackSwings(state, mover, target, swings); err != nil {
		return err
	}
	if ammunition < 0 || ammunition >= len(items) {
		return nil
	}
	// `DS:6D20h` 只數第一形態（`16BEh` 的 `[di+6D1Fh]`，di 是形態）；攻擊區段由第二形態倒數。
	shots := state.lastSwings - form2
	if shots < 0 {
		shots = 0
	}
	spend := gamepack.SpendAmmunition(items[ammunition].Raw, shots, gear.ThrownMelee)
	if !spend.Remove {
		return a.storeCombatItems(state, int(mover), slot, items)
	}
	if spend.Dropped != nil {
		dropped := poolsave.Item{Name: items[ammunition].Name, Raw: spend.Dropped}
		state.ThrownLoot = append([]poolsave.Item{dropped}, state.ThrownLoot...)
	}
	remaining := make([]poolsave.Item, 0, len(items)-1)
	remaining = append(remaining, items[:ammunition]...)
	remaining = append(remaining, items[ammunition+1:]...)
	return a.storeCombatItems(state, int(mover), slot, remaining)
}

// volleySwings 是 attackSwingsThisPhase 加上 overlay-13 `0D29h` 的射擊那一段：射擊武器而且
// entry 45 成立時，第一形態一回合的次數編碼改成型別表 `+05h`（至少 2），照樣過群組 18
// 與半回合相位，最後以彈藥數量封頂（`0DD1h..0E06h`）。回傳第二形態揮幾下，扣彈藥要扣掉。
func (a *app) volleySwings(state *tacticalState, mover uint8, gear gamepack.MissileGear,
	items []poolsave.Item) ([]combat.DamageDice, int, error) {
	if int(mover) >= len(state.AttackRates) {
		return nil, 0, fmt.Errorf("Pool attacker %d is outside the roster", mover)
	}
	var swings []combat.DamageDice
	form2 := 0
	for slot := gamepack.MonsterAttackSlots; slot >= 1; slot-- {
		dice := state.AttackForms[mover][slot-1]
		if dice.Count == 0 || dice.Sides == 0 {
			continue
		}
		rate := state.attackRateThisRound(mover, slot-1)
		volley := 0
		if slot == 1 && gear.Ranged && gear.CanFire {
			rate = gear.RateOfFire
			if int(mover) < len(state.RoundRates) {
				rate = gamepack.AttackRateAfterEffects(rate, state.RoundRates[mover])
			}
			volley = gear.VolleyLimit(rawItems(items))
		}
		count, err := combat.AttacksThisPhase(rate, state.AttackPhase&1)
		if err != nil {
			return nil, 0, err
		}
		if volley > 0 && int(count) > volley {
			count = uint8(volley)
		}
		for swing := uint8(0); swing < count; swing++ {
			swings = append(swings, dice)
		}
		if slot == 2 {
			form2 = int(count)
		}
	}
	return swings, form2, nil
}

// normalMissileAvoided 是群組 5 的 `29h`（gamepack.NormalMissileAvoided）落在盤面上。
func (a *app) normalMissileAvoided(state *tacticalState, attacker, target uint8) (bool, error) {
	if int(target) >= len(state.Effects) || !state.Effects[target].Has(gamepack.ProtectionFromNormalMissilesEffectCode) {
		return false, nil
	}
	gear, items, _, err := a.missileGear(state, attacker)
	if err != nil {
		return false, err
	}
	// 距離是 overlay-25 entry 33（`2591h`，spec 098）；走不到的算 0。
	distance, _ := state.tacticalRange(target, attacker)
	return gamepack.NormalMissileAvoided(state.Effects[target], gear, rawItems(items), distance, a.rollDice), nil
}

// foeChooseGear 是 overlay-09 entry 9（`13D5h`，gamepack.ChooseAIGear）。previousTarget 是
// 這一回合挑目標之前記著的那一個（`+108h` 的 `+0Ah`），型別 55h 的分數看它是不是不死生物。
func (a *app) foeChooseGear(state *tacticalState, mover, previousTarget uint8) error {
	if a.itemTypes == nil {
		return nil
	}
	items, slot := a.foeItemBearer(state, mover)
	if len(items) == 0 {
		return nil
	}
	adjacent, err := state.adjacentEnemies(mover)
	if err != nil {
		return err
	}
	_, alignment, _ := state.saveRecordOf(mover)
	input := gamepack.AIGearInput{
		Alignment:       alignment,
		AdjacentEnemies: adjacent,
		TargetUndead:    previousTarget != 0 && a.undeadColumn(state, int(previousTarget)) > 0,
	}
	if slot >= 0 {
		member := a.state.Party[slot]
		input.ClassMask = memberClassUseMask(member)
		input.NaturalScore = partyNaturalAIScore
		if member.NPC && len(member.Record) == poolsave.NPCRecordSize {
			input.NaturalScore = gamepack.AIGearNaturalScore(member.Record)
		}
	} else if monster, ok := a.stagedMonsterFor(int(mover), state.Friendly); ok {
		input.ClassMask = monster.Record.Raw[gamepack.ClassUseMaskOffset]
		input.NaturalScore = gamepack.AIGearNaturalScore(monster.Record.Raw[:])
	}
	raws := rawItems(items)
	// 原版 `18FEh` 不論換沒換都跑一次 entry 7；remake 只在真的叫過 Ready 時重算——戰鬥中直接改在
	// 盤面上的數值（臭雲的 AC，spec 121）還沒有逐項併進重算，每回合重算會把它們洗掉。
	steps, _, err := gamepack.ChooseAIGear(raws, a.itemTypes, input)
	if err != nil || len(steps) == 0 {
		return err
	}
	// 穿戴效果（`+3Eh` 大於 7Fh）跟著 Ready 掛上或摘掉；怪物那一側還沒接（spec 151〈還沒接〉）。
	if slot >= 0 {
		for _, step := range steps {
			switch step.Outcome {
			case gamepack.ReadyDone:
				a.wearItem(slot, raws[step.Index], gamepack.WearOn, state, int(mover))
			case gamepack.UnreadyDone:
				a.wearItem(slot, raws[step.Index], gamepack.WearOff, state, int(mover))
			}
		}
	}
	return a.storeCombatItems(state, int(mover), slot, items)
}
