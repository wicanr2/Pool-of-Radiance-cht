package gamepack

const strengthEffectInactiveBit = 0x80

// IsStrengthEffect 回答這是不是共用 overlay-12 entry 15 的力量效果。
func IsStrengthEffect(code uint8) bool {
	return code == EnlargeEffectCode || code == GiantStrengthEffectCode
}

// encodeStrengthSnapshot／decodeStrengthSnapshot 重現 overlay-24 entry 16／17
// （10DEh／1107h）。18 力量以百分位加一編碼，其餘力量加 100。
func encodeStrengthSnapshot(value, percentile uint8) uint8 {
	if value == EnlargeStrengthValue {
		return percentile + 1
	}
	return value + 100
}

func decodeStrengthSnapshot(snapshot uint8) (uint8, uint8) {
	snapshot &= ^uint8(strengthEffectInactiveBit)
	if snapshot <= 101 {
		return EnlargeStrengthValue, snapshot - 1
	}
	return snapshot - 100, 0
}

// ApplyStrengthEffect 重現 overlay-24 entry 18（1158h）：較強的新效果成為
// active，舊 active 節點改存自己的效果值並立起最高位；較弱的新效果只掛成
// inactive。active 節點保存的是「拿掉它後應回到的值」。
func ApplyStrengthEffect(list EffectList, code uint8, duration int,
	current, currentPercentile, value, percentile uint8) (EffectList, uint8, uint8, bool) {
	if duration < 0 {
		duration = 0
	}
	if duration > 0xffff {
		duration = 0xffff
	}
	copyList := append(EffectList(nil), list...)
	actual, actualPercentile, raised := RaiseStrength(
		current, currentPercentile, value, percentile)
	snapshot := encodeStrengthSnapshot(value, percentile) | strengthEffectInactiveBit
	if raised {
		base, basePercentile := current, currentPercentile
		for index := range copyList {
			if !IsStrengthEffect(copyList[index].Code) ||
				copyList[index].Payload[effectNodeLevelOffset]&strengthEffectInactiveBit != 0 {
				continue
			}
			base, basePercentile = decodeStrengthSnapshot(
				copyList[index].Payload[effectNodeLevelOffset])
			copyList[index].Payload[effectNodeLevelOffset] =
				encodeStrengthSnapshot(current, currentPercentile) | strengthEffectInactiveBit
			break
		}
		snapshot = encodeStrengthSnapshot(base, basePercentile)
	}
	node := NewEffectNode(code, uint16(duration), 0, true)
	node.Payload[effectNodeLevelOffset] = snapshot
	copyList = append(copyList, node)
	return copyList, actual, actualPercentile, raised
}

// ExpireStrengthEffect 重現 overlay-12 entry 15（04CBh）：inactive 節點到期不
// 動能力值；active 節點到期先回到基礎值，再把其餘力量節點中最強的一個設成
// active。remaining 已經不含這個到期節點。
func ExpireStrengthEffect(remaining EffectList, expired EffectNode,
	current, currentPercentile uint8) (uint8, uint8) {
	if !IsStrengthEffect(expired.Code) ||
		expired.Payload[effectNodeLevelOffset]&strengthEffectInactiveBit != 0 {
		return current, currentPercentile
	}
	base, basePercentile := decodeStrengthSnapshot(expired.Payload[effectNodeLevelOffset])
	best := -1
	bestValue, bestPercentile := uint8(0), uint8(0)
	for index := range remaining {
		if !IsStrengthEffect(remaining[index].Code) {
			continue
		}
		value, percentile := decodeStrengthSnapshot(
			remaining[index].Payload[effectNodeLevelOffset])
		if value > bestValue ||
			(value == EnlargeStrengthValue && percentile > bestPercentile) {
			best, bestValue, bestPercentile = index, value, percentile
		}
	}
	if best < 0 {
		return base, basePercentile
	}
	remaining[best].Payload[effectNodeLevelOffset] =
		encodeStrengthSnapshot(base, basePercentile)
	return bestValue, bestPercentile
}
