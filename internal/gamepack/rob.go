package gamepack

// `28h ROB`（spec 088）：小偷得手之後把隊伍的錢按比例拿走，並逐件試著偷走
// 物品。錢那一半在 overlay-07 entry 28（`1CE8h`），物品那一半在 entry 29
//（`1E17h`）。
const (
	// RobOpcode 是這條 opcode。
	RobOpcode = 0x28
	// RobOperands 是它吃幾個運算元。運算元 3 被取出來但從沒被讀。
	RobOperands = 3
	// RobScopeOperand 是「只偷目前這個角色還是整隊」那個運算元的序號；
	// 0 是目前角色，非 0 是整隊。
	RobScopeOperand = 1
	// RobPercentOperand 是百分比那個運算元的序號。原版讀了它兩次，
	// 一次給錢用、一次給物品用。
	RobPercentOperand = 2

	// RobPercentBase 是百分比的分母。
	RobPercentBase = 100
	// RobItemDie 是每件物品那一擲的面數。
	RobItemDie = 100

	// RobHeavyWeight 之上的物品把成功率扣掉 RobHeavyPenalty。
	RobHeavyWeight  = 0xff
	RobHeavyPenalty = 0x5a
	// RobMediumWeight 之上的物品扣 RobMediumPenalty。
	RobMediumWeight  = 0x18
	RobMediumPenalty = 0x32
)

// RobMoney 依比例減少七種貨幣。原版把 `(100 − 百分比) / 100` 算成六位元組
// 實數，再逐個貨幣相乘轉回整數。
//
// **待證**：實數轉整數那一步是截去還是四捨五入（`05BBh:0C72h` 沒有符號名，
// 兩者只差一枚硬幣）。這裡照截去。
func RobMoney(money [MoneyCurrencies]uint16, percent int) [MoneyCurrencies]uint16 {
	if percent < 0 {
		percent = 0
	}
	if percent > RobPercentBase {
		percent = RobPercentBase
	}
	remaining := RobPercentBase - percent
	var result [MoneyCurrencies]uint16
	for index, coins := range money {
		result[index] = uint16(int(coins) * remaining / RobPercentBase)
	}
	return result
}

// RobChanceAfterWeight 是一件物品在擲骰**之前**把成功率壓成多少
//（`1E38h`）。重的東西不好偷，而且**壓下去的值會留給後面的物品**——
// 那個參數是傳值進來之後被就地改掉的，同一次搜身裡不會回復。
func RobChanceAfterWeight(chance, weight int) int {
	penalty := 0
	switch {
	case weight > RobHeavyWeight:
		penalty = RobHeavyPenalty
	case weight > RobMediumWeight:
		penalty = RobMediumPenalty
	default:
		return chance
	}
	if chance > penalty {
		return chance - penalty
	}
	return 0
}

// RobTakesItem 是擲完之後這一件有沒有被偷走（`1E9Ah` 的 `roll <= chance`）。
func RobTakesItem(roll, chance int) bool { return roll <= chance }
