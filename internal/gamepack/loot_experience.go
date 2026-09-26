package gamepack

// 戰後結算的兩段（spec 148，issue #94）：戰利品折算經驗值（overlay-05 entry 2
// `0224h..0306h`）與 NPC 分錢（overlay-05 `1295h`）。算術照原版的寬度走：
// 公款是 32-bit、除與乘走 RTL 的 LongDiv／LongMul（`05BBh:0294h`／`0279h`），
// 物品那一項與 NPC 扣的那一筆是 16-bit `imul` 之後 `cwd`，份額是一個位元組。

// LootItemPlusOffset 是物品記錄的加值（`+32h`，有號位元組）。
const LootItemPlusOffset = 0x32

// LootItemExperiencePerPlus 是每一點加值折的經驗值（`02EBh mov cx, 190h`）。
const LootItemExperiencePerPlus = 400

// lootMoneyRates 是公款七欄（銅、銀、琥珀金、金、白金、寶石、珠寶）的換算：
// 前三欄除（`0224h..0263h` 的 LongDiv 200、20、2），金幣照加（`0266h`），
// 後三欄乘（`0272h..02B1h` 的 LongMul 5、250、2200）。
var lootMoneyRates = [7]struct {
	divide   bool
	operand  int32
	identity bool
}{
	{divide: true, operand: 200},
	{divide: true, operand: 20},
	{divide: true, operand: 2},
	{identity: true},
	{operand: 5},
	{operand: 250},
	{operand: 2200},
}

// LootExperience 是 entry 2 在怪物那一段之後加進經驗總額的部分：公款七欄依幣值
// 換算，再加上戰利品串列（`DS:676Eh`，頭在前）每件加值大於 0 的 `加值 × 400`。
// itemPlus 是串列上每件的 `+32h`，照串列順序、停在 `DS:5CF8h` 那一件之前。
//
// 物品那一項是 `imul cx` 之後 `cwd`：乘積只留低 16 位元再帶號擴展，所以加值
// 82 以上會變成負數（原版照扣）。
func LootExperience(pool [7]uint32, itemPlus []int8) uint32 {
	total := int32(0)
	for currency, rate := range lootMoneyRates {
		amount := int32(pool[currency])
		switch {
		case rate.identity:
			total += amount
		case rate.divide:
			total += amount / rate.operand
		default:
			total += amount * rate.operand
		}
	}
	for _, plus := range itemPlus {
		if plus <= 0 {
			continue
		}
		total += int32(int16(int16(plus) * LootItemExperiencePerPlus))
	}
	return uint32(total)
}

// NPCShareMember 是 `1295h` 對隊伍鏈每一個人看的三格。
type NPCShareMember struct {
	// NPC 是記錄 `+84h > 7Fh`（ADD NPC 寫進來的士氣位元組立著位元 7）。
	NPC bool
	// Status 是 `+10Ch`；只有 0 的 NPC 才分錢。
	Status uint8
	// Share 是 `+85h` 原值；份額取低三位元，是否列名看整個位元組。
	Share uint8
}

// HideNPCShares 是 overlay-05 `1295h`：戰後、開戰利品選單之前，隊伍裡的 NPC
// 依份額從公款拿走自己那一份藏起來（錢不進任何人的錢包，就是少了）。
//
//	12BCh..12F1h  NPC 而且狀態 0：NPC 份數與總份數各加 `+85h & 7`；其餘每人總份數加 1
//	1303h         NPC 份數 0 → 什麼都不做
//	1312h..137Ch  七欄各自：公款 > 0 時 每份 = 位元組(公款 ÷ 總份數)，
//	              公款 −= 每份 × NPC 份數（16-bit imul、cwd）
//	13C5h..141Dh  有扣到錢的話，逐一列出 NPC、狀態 0、`+85h > 0` 的人：
//	              「<名字> takes and hides his share.」
//
// 回傳扣完的公款與被列出來的成員索引（沒有扣到錢就是 nil）。
func HideNPCShares(pool [7]uint32, members []NPCShareMember) ([7]uint32, []int) {
	var npcShares, totalShares uint8
	for _, member := range members {
		if member.NPC && member.Status == 0 {
			npcShares += member.Share & 7
			totalShares += member.Share & 7
		} else {
			totalShares++
		}
	}
	if npcShares == 0 {
		return pool, nil
	}
	taken := false
	for currency := range pool {
		amount := int32(pool[currency])
		if amount <= 0 {
			continue
		}
		each := uint8(amount / int32(totalShares))
		amount -= int32(int16(uint16(each) * uint16(npcShares)))
		pool[currency] = uint32(amount)
		taken = true
	}
	if !taken {
		return pool, nil
	}
	var hiders []int
	for index, member := range members {
		if member.NPC && member.Status == 0 && member.Share > 0 {
			hiders = append(hiders, index)
		}
	}
	return pool, hiders
}
