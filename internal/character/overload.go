package character

import "fmt"

// ReceiveOverloaded 是 overlay-19 entry 9（`274Fh`，retf 8）本身：Trade（`178Fh`）在把物品交給
// 對方之前問它，spec 149〈Trade〉；拾取（overlay-06 entry 2 的 `0244h`，spec 035）也是它。
//
//	275B  overlay-25 entry 7 重算接收者（+C7h 物品數、+102h 總負重）
//	2767  +C7h > 0Fh                                  → 超重
//	2776  重量 = 物品 +37h；+39h > 0 就乘上它（16 位元 mul，取低字）
//	279C  上限 = overlay-25 entry 15（力量表）+ 5DCh
//	27AC  +102h + 重量（16 位元 add）> 上限（32 位元有號比較）→ 超重
//
// `+102h` 是物品重量加七欄錢的枚數（overlay-25 `0C17h`，spec 079），所以 money 要一起給。
// 全部 overlay 裡叫 `C9:004D` 的只有 overlay-06 `0244h`；另一個呼叫端是 overlay-19 自己的
// `178Fh`（near call）。CanReceiveItem 不算錢，拾取已經改用這一支。
func ReceiveOverloaded(strength, exceptional int, inventory [][]byte, money [7]uint16, incoming []byte) (bool, error) {
	if len(inventory) > 15 {
		return true, nil
	}
	capacity, err := CarryCapacity(strength, exceptional)
	if err != nil {
		return false, err
	}
	load := 0
	for _, coins := range money {
		load += int(coins)
	}
	for index, raw := range inventory {
		value, err := ItemLoad(raw)
		if err != nil {
			return false, fmt.Errorf("inventory item %d: %w", index, err)
		}
		load += value
	}
	incomingLoad, err := ItemLoad(incoming)
	if err != nil {
		return false, err
	}
	// `+102h` 是 word，重量是 `mul` 的低字，兩者相加也是 16 位元。
	total := (uint16(load) + uint16(incomingLoad))
	return int(total) > capacity, nil
}
