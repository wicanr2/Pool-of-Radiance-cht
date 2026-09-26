package gamepack

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 獸人家開打那一幀，原版對每一筆 combatant 跑過 overlay-25 entry 7 之後的欄位
// （docs/audit/dosgolem-monster-recompute-runtime.json，spec 147）。兩種獸人頭目
// 各自從 MON2 的樣板與物品串列重算，九個欄位逐位元組要對上：
//
//   - block 15 手上是短弓（型別 2Bh，穿戴中的最後一件武器）、身上是 +1 的盔甲：
//     AC 55 → 56、傷害 1d8 → 1d6、腳程 6 → 9（盔甲重 300 落在 151..399，`01F8h` 寫 9）。
//   - block 14 的武器都沒穿戴：AC、傷害照樣板，腳程一樣被盔甲改成 9。
//   - block 4 的獸人沒有物品：九個欄位就是樣板搬過去。
func TestMonsterRecomputeMatchesTheOrcHomeRuntimeRecords(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	types, err := ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "audit", "dosgolem-monster-recompute-runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Generator string `json:"generator"`
		Records   []struct {
			Name     string `json:"name"`
			THAC0    uint8  `json:"thac0_110h"`
			AC       uint8  `json:"ac_111h"`
			RearAC   uint8  `json:"rear_ac_112h"`
			Dice     string `json:"runtime_dice_114h_11Ah"`
			Move     uint8  `json:"move_11Ch"`
			Carried  int    `json:"carried_102h"`
			Weapon   string `json:"weapon_0CCh"`
			ItemsC7h int    `json:"item_count_C7h"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Generator != "dosgolem" || len(receipt.Records) != 20 {
		t.Fatalf("receipt from %q with %d records", receipt.Generator, len(receipt.Records))
	}
	recompute := func(block uint8) MonsterRecord {
		t.Helper()
		record, err := ReadDOSMonsterRecord(zipPath, 2, block)
		if err != nil {
			t.Fatal(err)
		}
		items, err := ReadDOSMonsterItems(zipPath, 2, block)
		if err != nil {
			t.Fatal(err)
		}
		out, err := RecomputeMonsterCombatFields(record, items, types)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	armed, unarmed, orc := recompute(15), recompute(14), recompute(4)
	matched := map[uint8]int{}
	for _, want := range receipt.Records {
		var got MonsterRecord
		var block uint8
		switch {
		case want.Name == "ORC":
			got, block = orc, 4
		case want.Name == "ORC LEADER" && want.Weapon != "00000000":
			got, block = armed, 15
		case want.Name == "ORC LEADER":
			got, block = unarmed, 14
		default:
			t.Fatalf("unexpected receipt record %q", want.Name)
		}
		dice, err := hex.DecodeString(want.Dice)
		if err != nil {
			t.Fatal(err)
		}
		carried := int(got.Raw[CarriedWeightOffset]) | int(got.Raw[CarriedWeightOffset+1])<<8
		if got.Raw[CurrentThac0Offset] != want.THAC0 || got.Raw[InternalArmourClassOffset] != want.AC ||
			got.Raw[RearArmourClassOffset] != want.RearAC || got.Raw[CurrentMovementOffset] != want.Move ||
			string(got.Raw[runtimeAttackBase:runtimeAttackBase+7]) != string(dice) || carried != want.Carried {
			t.Errorf("MON2 block %d %s: remake 110h=%d 111h=%d 112h=%d 114h..=% X 11Ch=%d 102h=%d, runtime %d %d %d % X %d %d",
				block, want.Name, got.Raw[CurrentThac0Offset], got.Raw[InternalArmourClassOffset],
				got.Raw[RearArmourClassOffset], got.Raw[runtimeAttackBase:runtimeAttackBase+7],
				got.Raw[CurrentMovementOffset], carried, want.THAC0, want.AC, want.RearAC, dice, want.Move, want.Carried)
		}
		matched[block]++
	}
	if matched[15] != 1 || matched[14] != 3 || matched[4] != 16 {
		t.Fatalf("receipt split %v, want one armed leader, three unarmed leaders and sixteen orcs", matched)
	}
}

// 負對照：同一支重算不能只是把樣板搬過去。block 15 的三個欄位在樣板裡是 55／1d8／6，
// 重算後是 56／1d6／9——哪一個沒動，上面那條就只證明了「沒有物品的怪物照樣板」。
// 另外 `+2Dh` 必須原封不動：entry 7 不寫它（spec 063），獸人頭目 44 不是一級戰士表的 40。
func TestMonsterRecomputeChangesTheArmedLeaderAndKeepsBaseThac0(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	types, err := ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	record, err := ReadDOSMonsterRecord(zipPath, 2, 15)
	if err != nil {
		t.Fatal(err)
	}
	items, err := ReadDOSMonsterItems(zipPath, 2, 15)
	if err != nil {
		t.Fatal(err)
	}
	out, err := RecomputeMonsterCombatFields(record, items, types)
	if err != nil {
		t.Fatal(err)
	}
	template, err := record.AttackDamage(1)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := out.RuntimeAttackDamage(1)
	if err != nil {
		t.Fatal(err)
	}
	if record.Raw[InternalArmourClassOffset] != 55 || template != (MonsterAttackDamage{Count: 1, Sides: 8}) || record.Raw[CurrentMovementOffset] != 6 {
		t.Fatalf("template changed: AC %d dice %+v move %d", record.Raw[InternalArmourClassOffset], template, record.Raw[CurrentMovementOffset])
	}
	if out.Raw[InternalArmourClassOffset] != 56 || runtime != (MonsterAttackDamage{Count: 1, Sides: 6}) || out.Raw[CurrentMovementOffset] != 9 {
		t.Fatalf("recomputed AC %d dice %+v move %d, want 56 1d6 9", out.Raw[InternalArmourClassOffset], runtime, out.Raw[CurrentMovementOffset])
	}
	if out.Raw[BaseThac0Offset] != record.Raw[BaseThac0Offset] || out.Raw[BaseThac0Offset] != 44 {
		t.Fatalf("+2Dh %d, template %d: entry 7 must not write it", out.Raw[BaseThac0Offset], record.Raw[BaseThac0Offset])
	}
}

// 沒有武器、`+0AAh` 開著的怪物，第一種形態的加值照 `0EAEh..0EC9h` 補上力量的傷害修正
// （`1366h` 自己查 `+0AAh`）。吸血鬼（MON1 block 23）18/xx 力量：1d6+4 → 1d6+8；
// 同一檔 ORC（`+0AAh` 是 0）不補。
func TestMonsterRecomputeAddsStrengthDamageOnlyWhenTheFlagIsSet(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	types, err := ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, testCase := range []struct {
		block uint8
		name  string
		want  MonsterAttackDamage
	}{
		{23, "VAMPIRE", MonsterAttackDamage{Count: 1, Sides: 6, Bonus: 8}},
		{4, "ORC", MonsterAttackDamage{Count: 1, Sides: 8}},
	} {
		record, err := ReadDOSMonsterRecord(zipPath, 1, testCase.block)
		if err != nil {
			t.Fatal(err)
		}
		if record.Name != testCase.name {
			t.Fatalf("MON1 block %d is %q, want %q", testCase.block, record.Name, testCase.name)
		}
		items, err := ReadDOSMonsterItems(zipPath, 1, testCase.block)
		if err != nil {
			t.Fatal(err)
		}
		out, err := RecomputeMonsterCombatFields(record, items, types)
		if err != nil {
			t.Fatal(err)
		}
		got, err := out.RuntimeAttackDamage(1)
		if err != nil {
			t.Fatal(err)
		}
		if got != testCase.want {
			t.Errorf("%s: slot 1 %+v, want %+v", testCase.name, got, testCase.want)
		}
	}
}
