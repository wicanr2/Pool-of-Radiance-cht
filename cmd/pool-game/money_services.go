package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 商店與神殿的公款四個鍵（spec 067〈公款〉，issue #79）。兩處的選單迴圈同一個形狀
// （overlay-06 `05A8h..077Ah`、overlay-04 `0D1Bh..0F06h`）：每一輪先問 overlay-21 entry 14
// 公款有沒有錢，有錢是 `... Take Pool Share Appraise Exit`，沒錢是 `... Pool Appraise Exit`；
// P 走 entry 5、S 走 entry 7、T 走 entry 8——與戰利品同三支（spec 040）。
// 兩處進門都先 `FillChar(DS:6752h, 1Ch, 0)`（overlay-06 `0548h`、overlay-04 `0D01h`）。

// moneyTakeState 是 overlay-21 entry 8（`0C9Eh..0F2Ah`）：先挑幣別（"Select type of coin"，
// `0C6Dh`），再輸入數量（"How much <幣> will you take?"，`0C85h`／`0C8Fh`），錢給**目前的角色**
// ——`0EC2h` 推的是 `DS:5CF0h`，這一支不問給誰。公款還有錢就回到挑幣別（`0F1Dh`）。
type moneyTakeState struct {
	amountStage bool
	cursor      int
	currency    int
	amount      string
	message     string
}

// pooledCurrencies 是公款裡非零的幣別，順序與戰利品的 TAKE 頁相同
// （`docs/audit/dos-treasure-screens.json`：GOLD、PLATINUM、JEWELRY）。
func (a *app) pooledCurrencies() []int {
	currencies := make([]int, 0, pooltreasure.CurrencyCount)
	for currency, amount := range a.state.PooledMoney {
		if amount != 0 {
			currencies = append(currencies, currency)
		}
	}
	return currencies
}

// takeLimit 是數量輸入的上限：`0EADh` 推給輸入常式 `0292h` 的是公款那一欄的**低位字**。
func (a *app) takeLimit(currency int) uint32 {
	return uint32(uint16(a.state.PooledMoney[currency]))
}

// moneyTakeRows 是挑幣別那一頁的每一行。
func (a *app) moneyTakeRows(take *moneyTakeState) []string {
	rows := []string{}
	for index, currency := range a.pooledCurrencies() {
		mark := "  "
		if index == take.cursor {
			mark = "> "
		}
		rows = append(rows, fmt.Sprintf("%s%s %d", mark, pooltreasure.Names[currency],
			a.state.PooledMoney[currency]))
	}
	// 最後一項是 Exit（原版那一列 `SELECT TYPE OF COIN EXIT`）。
	mark := "  "
	if take.cursor == len(rows) {
		mark = "> "
	}
	return append(rows, mark+"Exit")
}

// moneyTakePrompt 是底下那一列：挑幣別或輸入數量，前面有訊息就先印訊息。
func (a *app) moneyTakePrompt(take *moneyTakeState) string {
	prompt := a.text(msgShopTakeCurrency)
	if take.amountStage {
		prompt = fmt.Sprintf(a.text(msgShopTakeAmount), pooltreasure.Names[take.currency],
			a.takeLimit(take.currency), take.amount)
	}
	if take.message != "" {
		return take.message + "  " + prompt
	}
	return prompt
}

// moneyTakeInput 處理 T）ake 的鍵，member 是拿錢的人。回傳 true 代表離開了這一頁。
func (a *app) moneyTakeInput(take *moneyTakeState, member int) bool {
	currencies := a.pooledCurrencies()
	if len(currencies) == 0 {
		return true
	}
	rows := len(currencies) + 1 // 加上 Exit
	if take.cursor >= rows {
		take.cursor = rows - 1
	}
	if !take.amountStage {
		switch {
		case a.justPressed(ebiten.KeyEscape):
			return true
		case a.justPressed(ebiten.KeyArrowUp), a.justPressed(ebiten.KeyArrowLeft):
			take.cursor = (take.cursor + rows - 1) % rows
		case a.justPressed(ebiten.KeyArrowDown), a.justPressed(ebiten.KeyArrowRight):
			take.cursor = (take.cursor + 1) % rows
		case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeySpace):
			if take.cursor == len(currencies) {
				return true
			}
			take.amountStage, take.currency, take.amount, take.message =
				true, currencies[take.cursor], "", ""
		}
		return false
	}
	if a.justPressed(ebiten.KeyEscape) {
		take.amountStage, take.message = false, ""
		return false
	}
	if a.justPressed(ebiten.KeyBackspace) && take.amount != "" {
		take.amount = take.amount[:len(take.amount)-1]
	}
	for _, entered := range a.inputChars() {
		if entered >= '0' && entered <= '9' && len(take.amount) < 5 {
			take.amount += string(entered)
		}
	}
	if !a.justPressed(ebiten.KeyEnter) {
		return false
	}
	take.amountStage = false
	amount, err := strconv.ParseUint(take.amount, 10, 32)
	if err != nil || amount == 0 {
		take.message = ""
		return false
	}
	if uint32(amount) > a.takeLimit(take.currency) {
		// 輸入常式的上限（`0292h`）：超過就重打。
		take.amountStage, take.amount, take.message = true, "", ""
		return false
	}
	next := cloneSaveState(a.state)
	if err := pooltreasure.TakeMoney(&next, member, take.currency, uint32(amount)); err != nil {
		if errors.Is(err, pooltreasure.ErrOverloaded) {
			// `0A88h`：容量 helper 回 1 就印 "Overloaded"（`0A65h`），什麼都不動。
			take.message = a.text(msgShopTakeOverloaded)
		} else {
			take.message = err.Error()
		}
		return false
	}
	a.state = next
	take.message = fmt.Sprintf(a.text(msgShopTakeTaken),
		strings.TrimSpace(a.state.Party[member].Name), amount, pooltreasure.Names[take.currency])
	// `0EE0h..0F24h`：公款全空就離開，否則回到挑幣別。
	return !a.hasPooledMoney()
}

// poolPartyMoney 是 P）ool：overlay-21 entry 5。
func (a *app) poolPartyMoney() error {
	next := cloneSaveState(a.state)
	if err := pooltreasure.PoolMoney(&next); err != nil {
		return err
	}
	a.state = next
	return nil
}

// sharePartyMoney 是 S）hare：overlay-21 entry 7。
func (a *app) sharePartyMoney() error {
	next := cloneSaveState(a.state)
	if err := pooltreasure.ShareMoney(&next); err != nil {
		return err
	}
	a.state = next
	return nil
}

// templeMainOptions 是神殿的頂層選單：公款有錢是 `Heal View Take Pool Share Appraise Exit`
// （overlay-04 `0C1Ah`），沒錢是 `Heal View Pool Appraise Exit`（`0C42h`）。
func (a *app) templeMainOptions() []string {
	if a.hasPooledMoney() {
		return []string{"Heal", "View", "Take", "Pool", "Share", "Appraise", "Exit"}
	}
	return []string{"Heal", "View", "Pool", "Appraise", "Exit"}
}

// showTempleTake 把 T）ake 那一頁寫進神殿的文字框與底列。
// 選項也換成幣別（最後是 Exit），輸入數量時是單一個 "Amount"，與戰利品那一頁同一個形狀。
func (a *app) showTempleTake() {
	take := a.templeTake
	if take.amountStage {
		a.cellMenuOptions, a.cellMenuCursor = []string{"Amount"}, 0
	} else {
		options := []string{}
		for _, currency := range a.pooledCurrencies() {
			options = append(options, fmt.Sprintf("%s %d", pooltreasure.Names[currency],
				a.state.PooledMoney[currency]))
		}
		a.cellMenuOptions = append(options, "Exit")
		if take.cursor >= len(a.cellMenuOptions) {
			take.cursor = len(a.cellMenuOptions) - 1
		}
		a.cellMenuCursor = take.cursor
	}
	a.eventText = a.text(msgShopTakeTitle) + "\n" + strings.Join(a.moneyTakeRows(a.templeTake), "\n")
	a.eventLabel = a.moneyTakePrompt(a.templeTake)
}
