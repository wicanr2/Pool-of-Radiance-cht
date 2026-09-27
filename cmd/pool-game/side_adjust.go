package main

import (
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 遭遇腳本在開打前寫的三格戰場修正（spec 053、spec 096〈士氣崩了之後〉，issue #83）。
// 它們是隊伍記錄 `[4937h]` 的 `+6E0h`／`+6E2h`／`+6E4h`，換成 ECL 位址是 class 1 的
// `@6E70`／`@6E71`／`@6E72`（`(2A00h + 位址 × 2) mod 10000h`）；ECL5..8 有 27 處 `SAVE`
// 寫它們（`cmd/pool-ecl-memory-audit -addresses 6E70,6E71,6E72`）：命中那兩格是 FCh..05h
// 的有號 byte（敵方 −4..+2、隊伍 −3..+5），腳程那一格六處都是 FCh（ECL8 block 13／16，−4）。
//
//	overlay-24 entry 6 `0D03h..0D25h`  攻擊者 +10Eh == 0 → 讀 +6E2h，否則 +6E0h（byte，cbw）
//	                                   加進命中骰：`6780h + +110h + 它 >= AC`
//	overlay-13 `0123h` `0139h`         記錄 +10Eh == 0 → +11Ch 加 +6E4h 的 word，寫回 byte
//	                                   （FCh 就是腳程 −4），再夾到 1..96、乘 2、派發群組 12h
//	overlay-05 `15D4h..15F0h`          戰後主流程（entry 1 `14CAh`）一律把三格清成 0
//
// 戰鬥中 ECL 不會跑，所以開打時讀一次等於每次現讀。

const (
	eclFoeHitAdjustAddress    = 0x6E70 // +6E0h
	eclPartyHitAdjustAddress  = 0x6E71 // +6E2h
	eclPartyMoveAdjustAddress = 0x6E72 // +6E4h
)

// sideAdjustments 是開打時從 ECL 記憶體讀到的三格。
type sideAdjustments struct {
	FoeHit, PartyHit int8
	PartyMove        uint8
}

func readSideAdjustments(machine *eclvm.Machine) sideAdjustments {
	if machine == nil {
		return sideAdjustments{}
	}
	return sideAdjustments{
		FoeHit:    int8(machine.Memory[eclFoeHitAdjustAddress]),
		PartyHit:  int8(machine.Memory[eclPartyHitAdjustAddress]),
		PartyMove: uint8(machine.Memory[eclPartyMoveAdjustAddress]),
	}
}

// clearSideAdjustments 是 overlay-05 `15D4h..15F0h`。
func clearSideAdjustments(machine *eclvm.Machine) {
	if machine == nil {
		return
	}
	for _, address := range []uint16{eclFoeHitAdjustAddress, eclPartyHitAdjustAddress,
		eclPartyMoveAdjustAddress} {
		machine.Memory[address] = 0
	}
}

// sideHitBonus 是 entry 6 `0D03h`：出手的人那一邊的命中修正。
func (state *tacticalState) sideHitBonus(attacker uint8) int {
	if state.isFriendly(attacker) {
		return int(state.SideAdjust.PartyHit)
	}
	return int(state.SideAdjust.FoeHit)
}

// initialMovement 是 `0123h` 在群組 12h 之前的那一段：隊伍那一邊加 `+6E4h`（寫回 byte）。
func (state *tacticalState) initialMovement(index int) uint8 {
	friendly := index < len(state.Friendly) && state.Friendly[index]
	return combat.InitialMovementBudgetBeforeEffects(state.BaseMovement[index], friendly,
		int16(state.SideAdjust.PartyMove))
}
