package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/ecl"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// `2Eh DAMAGE`（spec 084）：ECL 直接對隊伍造成傷害，全遊戲 47 個呼叫點。
// 規則本身在 internal/gamepack；這裡負責解運算元、挑目標、擲骰與擲豁免，
// 再把結果寫回隊伍。

// damageEvent 找出結果裡的 `2Eh`。
func damageEvent(result eclvm.Result) (eclvm.Event, bool) {
	for _, event := range result.Events {
		if event.Opcode == gamepack.DamageOpcode {
			return event, true
		}
	}
	return eclvm.Event{}, false
}

// damageRequest 解出五個運算元。它們在原始資料裡是位元組字面值或變數，
// 所以照 NumericValue 取值。
func (a *app) damageRequest(event eclvm.Event) (gamepack.DamageRequest, error) {
	if a.eventSession == nil {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool DAMAGE has no ECL session")
	}
	archive, ok := a.eclCatalog.Archive(a.eclArchive)
	if !ok {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool ECL archive %d is absent", a.eclArchive)
	}
	block, ok := archive.Blocks[a.eventSession.CurrentBlockID()]
	if !ok {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool ECL block %d is absent from archive %d",
			a.eventSession.CurrentBlockID(), a.eclArchive)
	}
	if len(block) < 2 {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool ECL block %d is shorter than its two-byte prefix",
			a.eventSession.CurrentBlockID())
	}
	instruction, err := ecl.DecodeInstruction(block[2:], event.PC)
	if err != nil {
		return gamepack.DamageRequest{}, fmt.Errorf("decode Pool DAMAGE at %d: %w", event.PC, err)
	}
	if len(instruction.Operands) != gamepack.DamageOperands {
		return gamepack.DamageRequest{}, fmt.Errorf("Pool DAMAGE has %d operands, want %d",
			len(instruction.Operands), gamepack.DamageOperands)
	}
	var values [gamepack.DamageOperands]uint16
	for index, operand := range instruction.Operands {
		value, err := ecl.NumericValue(operand, a.eventSession.Machine().Memory)
		if err != nil {
			return gamepack.DamageRequest{}, fmt.Errorf("Pool DAMAGE operand %d: %w", index+1, err)
		}
		values[index] = value
	}
	return gamepack.NewDamageRequest(values), nil
}

// applyDamageEvent 套用一條 `2Eh`，然後讓 ECL 繼續。整支照 overlay-03 `2AE2h..2D31h`：
//
//	2B4F  傷害 = Roll(個數, 面數) + 加值                      ; 先擲，全隊共用
//	2B62  旗標 bit 6 → 全隊；否則先擲 Roll(1, 人數) 挑一個   ; bit 7 有沒有設都擲
//	2B89  旗標 bit 7 沒設 → 2C91h 攻擊那一種
//	2BA2  全隊：沿串列每一個，bit 5 → 直接吃；否則擲豁免(類別, 修正)，沒過才吃
//	2C01  運算元 5 bit 7 → 目前角色（5CF0h）：類別 0 不擲豁免，否則擲 (類別 − 1)
//	2C3D  否則 2B7E 擲到的那一個：一律擲豁免（這一支不看 bit 5）
//	2C91  攻擊：旗標是次數；每一次 Roll(1, 人數) 挑人、overlay-24 entry 5 以運算元 5
//	      對他的 AC 擲命中，中了吃傷害；每一次之後重擲傷害（2D0Bh）
func (a *app) applyDamageEvent(event eclvm.Event) error {
	request, err := a.damageRequest(event)
	if err != nil {
		return err
	}
	if err := a.applyDamageRequest(request); err != nil {
		return err
	}
	return a.continueInitialSearch(nil)
}

// applyDamageRequest 是上面那一段不含「讓 ECL 繼續」的部分。
func (a *app) applyDamageRequest(request gamepack.DamageRequest) error {
	if len(a.state.Party) == 0 {
		return fmt.Errorf("Pool DAMAGE has no party to damage")
	}
	// 傷害只擲一次：全隊模式下每個人吃的是同一個數字（`2B47h` 在挑目標之前）。
	damage := a.rollDice(request.DiceCount, request.DiceSides) + request.Bonus
	picked := -1
	if !request.WholeParty() {
		picked = a.rollDice(1, len(a.state.Party)) - 1
	}
	var lines []string
	record := func(line string, err error) error {
		if err != nil {
			return err
		}
		if line != "" {
			lines = append(lines, line)
		}
		return nil
	}
	save := gamepack.DamageRequest{}
	switch {
	case !request.Applies():
		for count := request.AttackCount(); count > 0; count-- {
			index := a.rollDice(1, len(a.state.Party)) - 1
			hit, err := a.damageAttackHits(index, request.Operand5)
			if err != nil {
				return err
			}
			if hit {
				if err := record(a.damageOne(index, nil, damage)); err != nil {
					return err
				}
			}
			damage = a.rollDice(request.DiceCount, request.DiceSides) + request.Bonus
		}
	case request.WholeParty():
		for index := range a.state.Party {
			check := &request
			if !request.AllowsSave() {
				check = nil
			}
			if err := record(a.damageOne(index, check, damage)); err != nil {
				return err
			}
		}
	case request.CurrentCharacter():
		index := a.currentCharacter
		if index < 0 || index >= len(a.state.Party) {
			index = 0
		}
		var check *gamepack.DamageRequest
		if request.SaveCategory != 0 {
			save = request
			save.SaveCategory--
			check = &save
		}
		if err := record(a.damageOne(index, check, damage)); err != nil {
			return err
		}
	default:
		if err := record(a.damageOne(picked, &request, damage)); err != nil {
			return err
		}
	}
	if len(lines) != 0 {
		a.eventText = strings.Join(lines, "\n")
	}
	return nil
}

// damageAttackHits 是攻擊那一種的一次命中（overlay-24 entry 5 `0C4Dh`）：擲 d20、問目標的
// 群組 16，再比「骰 + 命中值 > 目標 +111h」。目標的 `+111h` 照隊伍名單那一支現算。
func (a *app) damageAttackHits(index int, score uint8) (bool, error) {
	member := &a.state.Party[index]
	natural := uint8(a.rollDice(1, 20))
	if natural <= 1 {
		return false, nil
	}
	roll, effects := gamepack.HitRollEffects{
		Target: gamepack.HitRollCombatant{Effects: combatEffects(member.Effects)},
	}.Apply(gamepack.HitRollBase(natural))
	member.Effects = storedEffects(effects)
	armour, _, err := a.memberDefenceStats(*member, creationArmorClassInternal, creationBaseMovement)
	if err != nil {
		return false, err
	}
	return gamepack.DamageAttackHits(natural, roll, score, armour), nil
}

// damageOne 對一個人擲豁免（save 為 nil 就不擲）、套用傷害並回報一行訊息。
func (a *app) damageOne(index int, save *gamepack.DamageRequest, damage int) (string, error) {
	member := a.state.Party[index]
	if member.Status > gamepack.AliveStateMax {
		// 已經倒下的人不再吃傷害（`2958h` 對狀態 6 直接返回）。
		return "", nil
	}
	if save != nil {
		saved, err := a.savingThrowFor(member, save.SaveCategory, save.SaveModifier())
		if err != nil {
			return "", err
		}
		if saved {
			return fmt.Sprintf(a.text(msgDamageSaved), member.Name), nil
		}
	}
	outcome := gamepack.ApplyDamage(member.CurrentHP, member.Status, damage)
	a.state.Party[index].CurrentHP = outcome.HitPoints
	a.state.Party[index].Status = outcome.State
	for library := range a.state.CharacterLibrary {
		if a.state.CharacterLibrary[library].Name == member.Name {
			a.state.CharacterLibrary[library].CurrentHP = outcome.HitPoints
			a.state.CharacterLibrary[library].Status = outcome.State
		}
	}
	if outcome.Downed {
		return fmt.Sprintf(a.text(msgDamageDies), member.Name), nil
	}
	return fmt.Sprintf(a.text(msgDamageHit), member.Name, damage), nil
}

// savingThrowFor 依 spec 075 判定：目標值由職業等級查 `DS:41E6h` 的表算出來
// （spec 075 的 `0253h`），修正欄位 `+101h` remake 目前一律是 0。
func (a *app) savingThrowFor(member poolsave.Character, category, modifier int) (bool, error) {
	if a.savingThrows == nil {
		return false, fmt.Errorf("Pool saving throw table is not configured")
	}
	levels, err := partyClassLevels(member)
	if err != nil {
		return false, err
	}
	targets, err := a.savingThrows.TargetsForLevels(levels)
	if err != nil {
		return false, err
	}
	// 類別是旗標的低五位（`2B95h` 的 `and 1Fh`），所以值可以超過五個類別。
	// 原版不檢查，直接拿它去索引記錄的 `+6Dh + 類別`——全遊戲只有一處這樣：
	// 旗標 `CAh`（全隊、要豁免、類別 10），會讀到 `+77h`。那一格是什麼還沒
	// 讀，所以這裡取「最差的目標值」：只有自然 20 過得了。
	target := int(gamepack.SavingThrowWorstTarget)
	if category < gamepack.SavingThrowCategories {
		target = int(targets[category])
	}
	roll := a.rollDice(1, gamepack.SavingThrowDie)
	if roll == 1 {
		return false, nil
	}
	if roll == gamepack.SavingThrowDie {
		return true, nil
	}
	return target <= roll+modifier, nil
}
