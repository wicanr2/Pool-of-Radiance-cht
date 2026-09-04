package main

import (
	"fmt"
	"strings"

	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// A）ppraise：商店與神殿共用的估價與販賣（spec 116，overlay-21 entry 19）。
//
// 寶石與珠寶不是物品，是兩個計數。逐顆估價，玩家選 S）ell 或 K）eep；
// 賣掉只拿得到估價的五分之一，留著則變成一件物品。

// appraiseKind 是這一次在估哪一種。
type appraiseKind uint8

const (
	appraiseGem appraiseKind = iota
	appraiseJewel
)

// appraiseOptions 是估價選單。原版的字串是 `  Gems`／`  Jewelry`／` Exit`。
var appraiseOptions = []string{"Gems", "Jewelry", "Exit"}

// enterTempleAppraise 開估價選單。兩種都沒有時原版只印一句就退回。
func (a *app) enterTempleAppraise() {
	character := a.state.Party[a.templeParty]
	if character.Money[pooltreasure.Gems] == 0 && character.Money[pooltreasure.Jewelry] == 0 {
		a.eventText = "No gems or jewelry"
		a.statusLine = strings.TrimSpace(character.Name) + " has nothing to appraise."
		return
	}
	a.templeStage = templeAppraise
	a.cellMenuOptions = append(a.cellMenuOptions[:0], appraiseOptions...)
	a.cellMenuCursor = 0
	a.eventText = a.appraiseCollectionLine()
	a.eventLabel = a.cellMenuLabel()
	a.statusLine = "Original appraise menu."
}

// appraiseCollectionLine 是 `You have a fine collection of:` 那一段。
// 原版的單複數是分開的字串（` Gem`／` Gems`、` piece of jewelry`／
// ` pieces of jewelry`）。
func (a *app) appraiseCollectionLine() string {
	character := a.state.Party[a.templeParty]
	gems, jewels := int(character.Money[pooltreasure.Gems]), int(character.Money[pooltreasure.Jewelry])
	gemWord, jewelWord := " Gems", " pieces of jewelry"
	if gems == 1 {
		gemWord = " Gem"
	}
	if jewels == 1 {
		jewelWord = " piece of jewelry"
	}
	return fmt.Sprintf("You have a fine collection of:\n%d%s\n%d%s", gems, gemWord, jewels, jewelWord)
}

// offerAppraise 估一件的價。原版先把計數減一才擲骰（`1CACh`／`1F3Bh`），
// 所以取消也不會把它變回去——照它接。
func (a *app) offerAppraise(kind appraiseKind) error {
	character := &a.state.Party[a.templeParty]
	slot := pooltreasure.Gems
	if kind == appraiseJewel {
		slot = pooltreasure.Jewelry
	}
	if character.Money[slot] == 0 {
		a.statusLine = "There is none of that left to appraise."
		return nil
	}
	character.Money[slot]--

	roll := a.rollDice(1, 100)
	var value int
	var err error
	if kind == appraiseGem {
		value, err = pooltreasure.GemValue(roll)
	} else {
		// Turbo Pascal 的 `Random(N)` 回 0..N−1，這裡用 1..N 的骰子減一。
		value, err = pooltreasure.JewelryValue(roll, func(limit int) int {
			return a.rollDice(1, limit) - 1
		})
	}
	if err != nil {
		return err
	}
	a.appraiseKind, a.appraiseValue = kind, value
	noun := "gem"
	if kind == appraiseJewel {
		noun = "jewel"
	}
	a.eventText = fmt.Sprintf("The %s is valued at %d gp.", noun, value)
	// 背包滿了就沒有「留著」可選（`1DA7h` 比記錄 `+C7h` 是不是到 10h）。
	if len(character.Inventory) >= pooltreasure.KeepNeedsRoom {
		a.cellMenuOptions = []string{"Sell"}
	} else {
		a.cellMenuOptions = []string{"Sell", "Keep"}
	}
	a.templeStage = templeAppraiseOffer
	a.cellMenuCursor = 0
	a.eventLabel = a.cellMenuLabel()
	a.statusLine = fmt.Sprintf("Selling returns a fifth: %d gp.", pooltreasure.SellPrice(value))
	return nil
}

// resolveAppraise 收下玩家的決定。
func (a *app) resolveAppraise(keep bool) {
	character := &a.state.Party[a.templeParty]
	if keep {
		character.Inventory = append(character.Inventory, keptTreasureItem(a.appraiseKind, a.appraiseValue))
		a.statusLine = "Kept."
	} else {
		// 賣得的錢進的是白金那一欄（記錄 `+90h`，spec 040 的版面）——
		// 估價是金幣，1 白金 ＝ 5 金，spec 116。
		paid := pooltreasure.SellPrice(a.appraiseValue)
		character.Money[pooltreasure.Platinum] += uint16(paid)
		a.statusLine = fmt.Sprintf("Sold for %d gp.", paid)
	}
	syncTrainedLibraryCharacter(&a.state, *character)
	a.enterTempleAppraise()
}

// keptTreasureItem 建出原版留著時那 3Fh bytes 的物品記錄：
// `+2Eh = 46h`、`+31h = 65h`、`+37h = 1`，估好的價值放 `+3Ah`。
func keptTreasureItem(kind appraiseKind, value int) poolsave.Item {
	raw := make([]byte, 0x3F)
	raw[0x2E] = pooltreasure.KeptItemType
	raw[0x31] = pooltreasure.KeptItemSubtype
	raw[0x37] = 1
	raw[0x3A] = byte(value & 0xFF)
	raw[0x3B] = byte(value >> 8)
	name := "Gem"
	if kind == appraiseJewel {
		name = "Jewelry"
	}
	return poolsave.Item{Name: name, Raw: raw}
}
