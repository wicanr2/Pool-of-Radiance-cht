package gamepack

// 時間推進時的效果遞減，重現 **overlay-20 offset `0`**（spec 069／114）。
//
// 那一支同時是「世界時鐘往前走」與「效果往到期靠近」——原版把兩件事寫在
// 同一個 entry 裡（休息迴圈的 `0D5Eh` 就是叫它加五分鐘），所以 remake 這
// 一側也由同一個呼叫點驅動，不另外開一條計時。
//
// 參數是（欄位索引, 數量），函式開頭用 `DS:35D4h` 的進位表把它換算成**分**：
//
//	0x6a  while 欄位 > 1 { 數量 *= radix[欄位-1]; 欄位-- }
//
// 欄位 1 是分的個位，所以換算的終點就是分；`entry 2(索引 1, 5)` 進來的
// 數量是 5，出去還是 5。

const (
	// EffectTimeBatch 是一次推進最多吃掉幾分鐘（`0x97` 的 `cmp [bp-5], 0Ah`）。
	// 原版把整段時間切成十分鐘一批，每批走完整隊才進下一批。
	//
	// 對單一角色來說分不分批結果相同（`到期` 的判準是累積量），照抄是因為
	// 到期的收尾常式可以再掛新節點——那個節點在原版會被後面的批次繼續減，
	// 一次減完就不會。
	EffectTimeBatch = 10
)

// AdvanceEffects 把一條串列往前推 minutes 分鐘，回傳剩下的串列與**到期被摘
// 掉的節點**（依到期順序，呼叫端要對它們跑收尾）。
//
// 三條規則全部照 `0165h`：
//
//	0168  持續是 0 → 直接跳過：不遞減、不到期（**0 是永久**）
//	018D  經過量 >= 持續 → 到期，摘掉節點
//	0197  否則 持續 -= 經過量（是 sub 不是 dec）
func (list EffectList) AdvanceEffects(minutes int) (EffectList, []EffectNode) {
	current := append(EffectList(nil), list...)
	var expired []EffectNode
	for minutes > 0 {
		chunk := minutes
		if chunk > EffectTimeBatch {
			chunk = EffectTimeBatch
		}
		minutes -= chunk
		kept := make(EffectList, 0, len(current))
		for _, node := range current {
			duration := node.Duration()
			if duration == 0 {
				kept = append(kept, node)
				continue
			}
			if uint16(chunk) >= duration {
				expired = append(expired, node)
				continue
			}
			node.SetDuration(duration - uint16(chunk))
			kept = append(kept, node)
		}
		current = kept
	}
	return current, expired
}
