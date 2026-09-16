package treasure

import (
	"fmt"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 原版付錢的三種服務（spec 116〈付款〉）。錢在記錄裡是五個 16-bit 幣欄
// （`+88h` 銅、`+8Ah` 銀、`+8Ch` 琥珀金、`+8Eh` 金、`+90h` 白金；寶石與珠寶不算錢），
// 隊伍共用的 pool 是 `DS:6752h` 起五個 32-bit。換算表 `DS:0D38h` 以銅為單位：

// CoinValueInCopper 是 `DS:0D38h` 的五個換算值：銅 1、銀 10、琥珀金 100、金 200、白金 1000。
var CoinValueInCopper = [Platinum + 1]int64{1, 10, 100, 200, 1000}

// copperPerGold 是 `DS:0D44h`（表的第四格），entry 11 用它把總數換回金幣。
const copperPerGold = 200

// GoldEquivalent 重現 overlay-19 entry 11（`28A2h`）：五種硬幣乘換算值加總，
// 加 100 再除以 200——也就是四捨五入到金幣。寶石與珠寶不算。
func GoldEquivalent(money [CurrencyCount]uint16) int64 {
	total := int64(0)
	for coin := Copper; coin <= Platinum; coin++ {
		total += int64(money[coin]) * CoinValueInCopper[coin]
	}
	return (total + 100) / copperPerGold
}

// PoolGoldEquivalent 重現 overlay-21 entry 17（`00ACh`）：同一條算式套在 pool 上。
func PoolGoldEquivalent(pool [CurrencyCount]uint32) int64 {
	total := int64(0)
	for coin := Copper; coin <= Platinum; coin++ {
		total += int64(pool[coin]) * CoinValueInCopper[coin]
	}
	return (total + 100) / copperPerGold
}

// RemintCharacterMoney 重現 overlay-21 entry 15（`012Eh`）：清掉五種硬幣，
// 把餘額重鑄成 `白金 = gp ÷ 5`、`金 = gp mod 5`。神殿與武具店付完錢都是這樣寫回，
// 所以買一次東西銀幣與銅幣就不見了——那是原版。
func RemintCharacterMoney(money *[CurrencyCount]uint16, gold int64) error {
	if gold < 0 || gold/5 > 0xFFFF {
		return fmt.Errorf("Pool remint of %d gold does not fit the platinum field", gold)
	}
	for coin := Copper; coin <= Platinum; coin++ {
		money[coin] = 0
	}
	money[Platinum] = uint16(gold / 5)
	money[Gold] = uint16(gold % 5)
	return nil
}

// RemintPooledMoney 重現 overlay-21 entry 16（`0183h`）：pool 版的同一件事。
func RemintPooledMoney(pool *[CurrencyCount]uint32, gold int64) error {
	if gold < 0 || gold/5 > 0xFFFFFFFF {
		return fmt.Errorf("Pool remint of %d pooled gold does not fit", gold)
	}
	for coin := Copper; coin <= Platinum; coin++ {
		pool[coin] = 0
	}
	pool[Platinum] = uint32(gold / 5)
	pool[Gold] = uint32(gold % 5)
	return nil
}

// PayInCoins 重現 overlay-16 `42C1h`（訓練所收 1000 金用的那一支）：要付的金額乘 200
// 換成銅，從銅開始一種一種付——每一種付 `剩餘 ÷ 面值 + 1` 枚（有幾枚付幾枚）——
// 付到剩餘不是正數為止；付多了就從白金往下找零。
//
// 五種硬幣付完還有剩的情形原版沒有守：迴圈會往 `0D38h` 表外讀（寶石、珠寶那兩格是
// 別的資料，再過去是 0，除以 0 就當掉）。呼叫端用 GoldEquivalent 擋在前面，但那一支
// 四捨五入——999 金 199 銅算 1000 金，過得了門卻付不滿。這裡**remake-owned** 的處置是
// 把五種硬幣全收走（差的不到半個金幣），不往表外走。
func PayInCoins(money *[CurrencyCount]uint16, gold int64) error {
	if gold < 0 {
		return fmt.Errorf("Pool payment of %d gold is negative", gold)
	}
	remaining := gold * copperPerGold
	for coin := Copper; remaining > 0; coin++ {
		if coin > Platinum {
			return nil
		}
		units := remaining/CoinValueInCopper[coin] + 1
		if int64(money[coin]) < units {
			units = int64(money[coin])
		}
		remaining -= units * CoinValueInCopper[coin]
		money[coin] -= uint16(units)
	}
	if remaining < 0 {
		change := -remaining
		for coin := Platinum; change > 0 && coin >= Copper; coin-- {
			units := change / CoinValueInCopper[coin]
			change -= units * CoinValueInCopper[coin]
			money[coin] += uint16(units)
		}
	}
	return nil
}

// PaySource 說錢是誰出的：原版先看角色自己的硬幣總值，不夠才看 pool，兩邊不混付。
type PaySource string

const (
	PaidByCharacter PaySource = "character"
	PaidByPool      PaySource = "pool"
)

// PayGold 是神殿與武具店的付款順序（overlay-06 `034Fh`、overlay-04 `016Ah..01D1h`，
// spec 018／116）：角色的 GoldEquivalent 夠就從角色扣、餘額重鑄（entry 15）；
// 不夠才看 pool（entry 17／16）；都不夠回 ok=false、什麼都不動。
func PayGold(state *poolsave.State, partyIndex int, price int64) (PaySource, bool, error) {
	if state == nil || partyIndex < 0 || partyIndex >= len(state.Party) {
		return "", false, fmt.Errorf("Pool payment has no party member %d", partyIndex)
	}
	character := &state.Party[partyIndex]
	if have := GoldEquivalent(character.Money); have >= price {
		if err := RemintCharacterMoney(&character.Money, have-price); err != nil {
			return "", false, err
		}
		syncLibraryCharacter(state, *character)
		return PaidByCharacter, true, nil
	}
	if have := PoolGoldEquivalent(state.PooledMoney); have >= price {
		if err := RemintPooledMoney(&state.PooledMoney, have-price); err != nil {
			return "", false, err
		}
		return PaidByPool, true, nil
	}
	return "", false, nil
}
