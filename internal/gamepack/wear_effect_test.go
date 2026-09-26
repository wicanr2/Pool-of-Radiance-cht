package gamepack

import "testing"

func wearItem(code, granted uint8) []byte {
	raw := make([]byte, 63)
	raw[ItemEffectOffset] = code
	raw[ItemGrantedEffectOffset] = granted
	return raw
}

// `+3Eh` 不大於 7Fh 的（卷軸的第三行、一般物品）不派發（overlay-19 `14D4h` 的 `ja`）。
func TestWearEffectNeedsTheHighBit(t *testing.T) {
	for _, code := range []uint8{0x00, 0x2a, 0x7f} {
		result := ApplyWearEffect(nil, wearItem(code, 0x3d), WearOn, 12, 0)
		if result.Known || len(result.List) != 0 {
			t.Fatalf("+3Eh %#02x dispatched: %+v", code, result)
		}
	}
}

// 83h 卸下而現在不是 18/00（有更強的力量效果在作用）：摘的是快照解回 18/00 的
// 那一個 26h（`2FE3h..2FEDh`），不是第一個。
func TestGauntletsOffPicksTheirOwnNode(t *testing.T) {
	other := NewEffectNode(GiantStrengthEffectCode, 0, 0, true)
	other.Payload[effectNodeLevelOffset] = encodeStrengthSnapshot(19, 0) | strengthEffectInactiveBit
	own := NewEffectNode(GiantStrengthEffectCode, 0, 0, true)
	own.Payload[effectNodeLevelOffset] = encodeStrengthSnapshot(18, 100) | strengthEffectInactiveBit
	result := ApplyWearEffect(EffectList{other, own}, wearItem(0x83, 0x26), WearOff, 20, 0)
	if len(result.Removed) != 1 || result.Removed[0] != own || len(result.List) != 1 || result.List[0] != other {
		t.Fatalf("removed %+v, kept %+v", result.Removed, result.List)
	}
}

// 89h 卸下摘第一個 17h（`3195h`），戴上什麼也不做。
func TestCode89RemovesEffect17(t *testing.T) {
	list := EffectList{NewEffectNode(0x17, 0, 0, false), NewEffectNode(0x17, 5, 0, false)}
	if result := ApplyWearEffect(list, wearItem(0x89, 0), WearOn, 12, 0); len(result.List) != 2 {
		t.Fatalf("89h on changed the list: %+v", result.List)
	}
	result := ApplyWearEffect(list, wearItem(0x89, 0), WearOff, 12, 0)
	if len(result.List) != 1 || result.List[0].Duration() != 5 {
		t.Fatalf("89h off: %+v", result.List)
	}
}

// 84h（`30B1h`）：卸下（模式 1）什麼也不做；陣營相同不做；不同就卸下、受 `+3Dh ÷ 16`。
func TestAlignedWearDamage(t *testing.T) {
	sword := wearItem(AlignedWearEffectCode, 0x52)
	if refused, damage := AlignedWearDamage(sword, WearOff, 0); refused || damage != 0 {
		t.Fatalf("taking it off: %v %d", refused, damage)
	}
	if refused, _ := AlignedWearDamage(sword, WearOn, 2); refused {
		t.Fatal("the matching alignment was refused")
	}
	if refused, damage := AlignedWearDamage(sword, WearOn, 0); !refused || damage != 5 {
		t.Fatalf("the wrong alignment: %v %d, want true 5", refused, damage)
	}
	if result := ApplyWearEffect(nil, sword, WearOn, 12, 0); !result.Known || len(result.List) != 0 {
		t.Fatalf("84h touched the effect list: %+v", result)
	}
}
