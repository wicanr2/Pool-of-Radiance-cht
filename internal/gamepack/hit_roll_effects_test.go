package gamepack

import (
	"bytes"
	"path/filepath"
	"testing"
)

// 名字表是 START.EXE 資料段裡的 string[15]（每格 16 bytes）。正對照：同一個基準下
// `DS:2880h` 要讀到 `33 34 35 1F`（spec 059 的四個反應攻擊否決碼）。
func TestHitRollNameTablesMatchStartExe(t *testing.T) {
	raw, err := readStartExecutable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	if got := raw[startDataSegmentFileDelta+0x2880 : startDataSegmentFileDelta+0x2884]; !bytes.Equal(got, []byte{0x33, 0x34, 0x35, 0x1f}) {
		t.Fatalf("DS base is off: DS:2880h reads % X", got)
	}
	for _, table := range []struct {
		address int
		names   []string
	}{
		{GnomeFoeNamesAddress, GnomeFoeNames[:]},
		{DwarfFoeNamesAddress, DwarfFoeNames[:]},
	} {
		for index, name := range table.names {
			at := startDataSegmentFileDelta + table.address + index*16
			length := int(raw[at])
			if got := string(raw[at+1 : at+1+length]); got != name {
				t.Errorf("DS:%04Xh slot %d is %q, want %q", table.address, index+1, got, name)
			}
		}
	}
}

func hitValue(effects HitRollEffects, roll uint8) int8 {
	value, _ := effects.Apply(HitRollBase(roll))
	return value
}

func carrying(codes ...uint8) EffectList {
	var list EffectList
	for _, code := range codes {
		list = list.Append(NewEffectNode(code, 5, 1, false))
	}
	return list
}

// 逐碼的正例與反例（處理常式位址見 hit_roll_effects.go）。
func TestHitRollEffectsFollowTheHandlers(t *testing.T) {
	undead := HitRollCombatant{CreatureType: 4, BodySize: 1}
	goblin := HitRollCombatant{CreatureType: 1, BodySize: 1, Name: "GOBLIN"}
	for _, tc := range []struct {
		name    string
		effects HitRollEffects
		want    int8
	}{
		{"nothing", HitRollEffects{}, 10},
		{"21h", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x21)}}, 6},
		{"24h", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x24)}}, 6},
		{"03h undead", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x03)}, Target: undead}, 12},
		{"03h living", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x03)}, Target: goblin}, 10},
		{"06h undead", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x06)}, Target: undead}, 13},
		{"06h type 9", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x06)},
			Target: HitRollCombatant{CreatureType: 9}}, 12},
		{"06h goblin", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x06)}, Target: goblin}, 10},
		{"12h goblin", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x12)}, Target: goblin}, 11},
		{"1Ah goblin", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x1a)}, Target: goblin}, 11},
		{"12h + 1Ah", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x12, 0x1a)}, Target: goblin}, 12},
		{"12h ogre", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x12)},
			Target: HitRollCombatant{CreatureType: 1, BodySize: 1, Name: "OGRE"}}, 10},
		{"31h same side", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x31), Side: 0}}, 11},
		{"31h other side", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x31), Side: 1}}, 9},
		{"31h from the area", HitRollEffects{Attacker: HitRollCombatant{Side: 1},
			AreaNode: func(uint8) (EffectNode, bool) { return NewEffectNode(0x31, 5, 0x13, false), true }}, 11},
		{"19h on target", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x19)}}, 6},
		{"19h on attacker", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x19)}}, 10},
		{"47h on target", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x47)}}, 6},
		{"25h not yet acted", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x25), Score: 3}}, -1},
		{"25h acted", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x25)}}, 10},
		{"30h bugbear actor", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x30)},
			Actor: HitRollCombatant{CreatureType: 1, Name: "BUGBEAR"}}, 6},
		{"30h gnoll not humanoid", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x30)},
			Actor: HitRollCombatant{CreatureType: 2, Name: "GNOLL"}}, 10},
		{"2Fh not wired", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x2f)}}, 10},
		{"59h first", HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x59)}, AttackPhase: 1}, -1},
		{"natural 20 blessed", HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x01)}}, 11},
	} {
		if got := hitValue(tc.effects, 10); got != tc.want {
			t.Errorf("%s: 6780h = %d, want %d", tc.name, got, tc.want)
		}
	}
	if got := hitValue(HitRollEffects{Attacker: HitRollCombatant{Effects: carrying(0x01)}}, 20); got != 101 {
		t.Errorf("a blessed natural 20 is %d, want 101", got)
	}
}

// 59h 改的是目標自己的節點：第一擊立起位元 4，之後照常；相位 0 而命中骰 0 時清高四位。
func TestDisplacementRewritesItsNode(t *testing.T) {
	effects := HitRollEffects{Target: HitRollCombatant{Effects: carrying(0x59)}, AttackPhase: 2}
	value, list := effects.Apply(12)
	if value != -1 || list[0].Payload[effectNodeLevelOffset]&0x10 == 0 {
		t.Fatalf("first: %d %+v", value, list[0])
	}
	if effects.Target.Effects[0].Payload[effectNodeLevelOffset]&0x10 != 0 {
		t.Fatal("Apply wrote through the caller's list")
	}
	effects.Target.Effects = list
	if value, _ := effects.Apply(12); value != 12 {
		t.Fatalf("second: %d", value)
	}
	effects.AttackPhase = 0
	value, list = effects.Apply(0)
	if value != 0 || list[0].Payload[effectNodeLevelOffset]&0xf0 != 0 {
		t.Fatalf("reset: %d %+v", value, list[0])
	}
}

// `0FCCh` 摘光所有 19h，別的碼不動。
func TestDropInvisibilityRemovesEveryNode(t *testing.T) {
	list := DropInvisibility(carrying(0x19, 0x01, 0x19))
	if list.Has(0x19) || !list.Has(0x01) || len(list) != 1 {
		t.Fatalf("got %+v", list)
	}
}
