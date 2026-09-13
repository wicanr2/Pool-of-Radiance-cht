package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
)

type dosCombatParityReceipt struct {
	Schema                string `json:"schema"`
	Generator             string `json:"generator"`
	GeneratorRevision     string `json:"generator_revision"`
	OriginalEXESHA256     string `json:"original_exe_sha256"`
	AttackerTHAC0Internal int    `json:"attacker_thac0_internal"`
	AttackerTHAC0Display  int    `json:"attacker_thac0_display"`
	EffectiveACInternal   int    `json:"effective_ac_internal"`
	EffectiveACDisplay    int    `json:"effective_ac_display"`
	D20                   int    `json:"d20"`
	Hit                   bool   `json:"hit"`
	DamageRoll            int    `json:"damage_roll"`
	Damage                int    `json:"damage"`
	AttackerHPBefore      int    `json:"attacker_hp_before"`
	AttackerHPAfter       int    `json:"attacker_hp_after"`
	VictimHPBefore        int    `json:"victim_hp_before"`
	VictimHPAfter         int    `json:"victim_hp_after"`
}

type dosCombatParityRoller struct {
	rolls []int
	index int
}

func (roller *dosCombatParityRoller) Roll(_, _ int) int {
	if roller.index >= len(roller.rolls) {
		panic("DOS combat parity roll stream exhausted")
	}
	value := roller.rolls[roller.index]
	roller.index++
	return value
}

// 這條測試消費 tools/pool-dos-combat-parity.sh 從正常玩家路徑重生的 DOS
// 收據，固定同一組數值與骰子，驗證 remake 的命中、傷害與 HP 鏈（spec 050）。
func TestDOSFirstCombatControlledTrace(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the parity test source")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "docs", "audit", "dos-combat-parity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt dosCombatParityReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Schema != "pool-dos-combat-parity/1" || receipt.Generator != "dosgolem" ||
		receipt.GeneratorRevision != "d351681ba86d97aab571d00b979c36e2336486f3" ||
		receipt.OriginalEXESHA256 != "12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f" {
		t.Fatalf("untrusted DOS combat receipt: %+v", receipt)
	}
	if receipt.D20 != 20 || !receipt.Hit || receipt.DamageRoll != 2 || receipt.Damage != 2 {
		t.Fatalf("unexpected controlled DOS rolls: %+v", receipt)
	}
	if receipt.AttackerTHAC0Display != 20 || receipt.EffectiveACDisplay != 6 {
		t.Fatalf("unexpected displayed DOS combat values: %+v", receipt)
	}

	state := newAttackState()
	state.HitPoints[1] = receipt.AttackerHPBefore
	state.HitPoints[2] = receipt.VictimHPBefore
	state.THAC0[1] = uint8(receipt.AttackerTHAC0Internal)
	state.ArmorClass[2] = receipt.EffectiveACInternal
	state.setSingleAttackForm(1, combat.DamageDice{Count: 1, Sides: 2})
	roller := &dosCombatParityRoller{rolls: []int{receipt.D20, receipt.DamageRoll}}
	application := &app{roller: roller, tactical: state}
	if err := application.resolveTacticalAttack(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[1] != receipt.AttackerHPAfter || state.HitPoints[2] != receipt.VictimHPAfter {
		t.Fatalf("remake HP after controlled attack = attacker %d, victim %d; DOS wants %d, %d",
			state.HitPoints[1], state.HitPoints[2], receipt.AttackerHPAfter, receipt.VictimHPAfter)
	}
	if roller.index != len(roller.rolls) {
		t.Fatalf("remake consumed %d controlled rolls, want %d", roller.index, len(roller.rolls))
	}
}
