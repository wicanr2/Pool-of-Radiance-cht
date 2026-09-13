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
	Schema            string                  `json:"schema"`
	Generator         string                  `json:"generator"`
	GeneratorRevision string                  `json:"generator_revision"`
	OriginalEXESHA256 string                  `json:"original_exe_sha256"`
	Actions           []dosCombatParityAction `json:"actions"`
	End               dosCombatParityEnd      `json:"end"`
}

type dosCombatParityAction struct {
	Round                   int    `json:"round"`
	Action                  int    `json:"action"`
	AttackerTHAC0Internal   int    `json:"attacker_thac0_internal"`
	AttackerTHAC0Display    int    `json:"attacker_thac0_display"`
	EffectiveACInternal     int    `json:"effective_ac_internal"`
	EffectiveACDisplay      int    `json:"effective_ac_display"`
	D20                     int    `json:"d20"`
	Hit                     bool   `json:"hit"`
	DamageRoll              int    `json:"damage_roll"`
	Damage                  int    `json:"damage"`
	AttackerHPBefore        int    `json:"attacker_hp_before"`
	AttackerHPAfter         int    `json:"attacker_hp_after"`
	VictimHPBefore          int    `json:"victim_hp_before"`
	VictimHPAfterDamage     int    `json:"victim_hp_after_damage"`
	VictimHPAfterResolution int    `json:"victim_hp_after_resolution"`
	VictimStateAfter        int    `json:"victim_state_after"`
	VictimPresentAfter      bool   `json:"victim_present_after"`
	Resolution              string `json:"resolution"`
}

type dosCombatParityEnd struct {
	ResolvedActions    int    `json:"resolved_actions"`
	ContinuePrompt     bool   `json:"continue_prompt"`
	ContinueAnswer     string `json:"continue_answer"`
	Outcome            string `json:"outcome"`
	XPPerCharacter     int    `json:"xp_per_character"`
	PromptFrameSHA256  string `json:"prompt_frame_sha256"`
	VictoryFrameSHA256 string `json:"victory_frame_sha256"`
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
	if receipt.Schema != "pool-dos-combat-parity/2" || receipt.Generator != "dosgolem" ||
		receipt.GeneratorRevision != "d351681ba86d97aab571d00b979c36e2336486f3" ||
		receipt.OriginalEXESHA256 != "12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f" {
		t.Fatalf("untrusted DOS combat receipt: %+v", receipt)
	}
	if len(receipt.Actions) != 1 || receipt.End.ResolvedActions != 1 {
		t.Fatalf("DOS battle has %d actions / end count %d, want one complete action",
			len(receipt.Actions), receipt.End.ResolvedActions)
	}
	action := receipt.Actions[0]
	if action.Round != 1 || action.Action != 1 || action.D20 != 20 || !action.Hit ||
		action.DamageRoll != 2 || action.Damage != 2 {
		t.Fatalf("unexpected controlled DOS action: %+v", action)
	}
	if action.AttackerTHAC0Display != 20 || action.EffectiveACDisplay != 6 {
		t.Fatalf("unexpected displayed DOS combat values: %+v", action)
	}
	if action.VictimHPAfterDamage != 2 || action.VictimHPAfterResolution != 0 ||
		action.VictimPresentAfter || action.VictimStateAfter != 4 {
		t.Fatalf("unexpected DOS foe removal: %+v", action)
	}
	if !receipt.End.ContinuePrompt || receipt.End.ContinueAnswer != "N" ||
		receipt.End.Outcome != "party_victory" || receipt.End.XPPerCharacter != 15 ||
		receipt.End.PromptFrameSHA256 != "0a76276072a68815d2549728b293824d29623912c24066905d2e5b4a1ca24aab" ||
		receipt.End.VictoryFrameSHA256 != "00efa7759fa40d6cc24bb1cc3573545941efacf87e1e77ccde829bdfef889149" {
		t.Fatalf("unexpected DOS battle ending: %+v", receipt.End)
	}

	state := newAttackState()
	state.HitPoints[1] = action.AttackerHPBefore
	state.HitPoints[2] = action.VictimHPBefore
	state.THAC0[1] = uint8(action.AttackerTHAC0Internal)
	state.ArmorClass[2] = action.EffectiveACInternal
	state.setSingleAttackForm(1, combat.DamageDice{Count: 1, Sides: 2})
	roller := &dosCombatParityRoller{rolls: []int{action.D20, action.DamageRoll}}
	application := &app{roller: roller, tactical: state}
	if err := application.resolveTacticalAttack(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.HitPoints[1] != action.AttackerHPAfter || state.HitPoints[2] != action.VictimHPAfterDamage {
		t.Fatalf("remake HP after controlled attack = attacker %d, victim %d; DOS wants %d, %d",
			state.HitPoints[1], state.HitPoints[2], action.AttackerHPAfter, action.VictimHPAfterDamage)
	}
	if roller.index != len(roller.rolls) {
		t.Fatalf("remake consumed %d controlled rolls, want %d", roller.index, len(roller.rolls))
	}

	// DOS 的 QUICK 在數值攻擊後把唯一敵人移出戰場；由這個同狀態邊界驗證
	// remake 的回合收尾也先詢問，答 N 後得到勝利。敵人為何被 QUICK 移除
	// 不屬於這份「命中／傷害數值」對拍，不把原版狀態碼 4 假裝成 remake 狀態碼。
	state.HitPoints[2] = action.VictimHPAfterResolution
	state.Roster[2].FootprintClass = 0
	state.States[2] = combat.DyingState
	state.Mover = 0
	state.endRound(application.rollDice)
	if !state.Prompt || state.Finished {
		t.Fatalf("remake cleared battle prompt=%v finished=%v, DOS wants a prompt", state.Prompt, state.Finished)
	}
	state.Prompt = false
	state.Finished, state.Outcome = true, combat.ResolveCombatOutcome(state.sideCounts())
	if state.Outcome != combat.CombatVictory {
		t.Fatalf("remake outcome %v after N, DOS wants party victory", state.Outcome)
	}
}
