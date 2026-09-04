package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolcharacter "github.com/wicanr2/Pool-of-Radiance-cht/internal/character"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 商店走的是與墓園戰利品同一條 ECL 服務邊界，差別在三個旗標（spec 067）。
// 沒有這個判別，走進商店會把整櫃存貨當成免費戰利品發下去。
const (
	shopTextLeft   = 48
	shopFirstLine  = 118
	shopLineHeight = 16
	shopLineCount  = 13
)

type shopState struct {
	items   []gamepack.TreasureItemRecord
	cursor  int
	buyer   int
	message string
	// 估價中的那一件（spec 116）。原版是先把計數減一才擲骰，所以這裡
	// 一旦有值就代表那一顆已經從身上拿出來了，只剩賣或留。
	appraising    bool
	appraiseKind  appraiseKind
	appraiseValue int
}

// isShopBoundary 判斷這個服務邊界是不是商店。
//
// 判準是 ECL 記憶體的三個旗標，不是 item block 編號：block 只說「存貨在哪一格」，
// 之後若有新的戰利品用到相鄰的 block，用編號判會把它當成商店。
func (a *app) isShopBoundary(result eclvm.Result) bool {
	if a.eventMachine == nil || len(result.TreasureRequests) == 0 {
		return false
	}
	if a.eventMachine.Memory[gamepack.ShopServiceKindAddress] != gamepack.ShopServiceKind {
		return false
	}
	if a.eventMachine.Memory[gamepack.ShopServiceEnabledAddress] != 1 ||
		a.eventMachine.Memory[gamepack.ShopServicePartyAddress] != 1 {
		return false
	}
	// 商店的七個貨幣數量都是零；有錢就是戰利品，不是店。
	for _, request := range result.TreasureRequests {
		for _, amount := range request.Amounts {
			if amount != 0 {
				return false
			}
		}
	}
	return true
}

func (a *app) enterShop(requests []eclvm.TreasureRequest) error {
	if a.loadTreasure == nil {
		return fmt.Errorf("Pool treasure loader is not configured")
	}
	stock := make([]gamepack.TreasureItemRecord, 0)
	for _, request := range requests {
		if request.ItemBlock == 0 {
			continue
		}
		if request.ItemBlock > 0xFF {
			return fmt.Errorf("Pool shop item block 0x%X exceeds byte range", request.ItemBlock)
		}
		items, err := a.loadTreasure(a.spawn.Map.Archive, uint8(request.ItemBlock))
		if err != nil {
			return err
		}
		stock = append(stock, items...)
	}
	if len(stock) == 0 {
		return fmt.Errorf("Pool shop service produced no stock")
	}
	a.shop = &shopState{items: stock}
	a.shopActive = true
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventLabel = a.text(msgShopTitle)
	a.statusLine = fmt.Sprintf(a.text(msgShopStatus), len(stock))
	return nil
}

// buy 把選中的物品賣給目前的買家。付款只用金幣：原版的價目也以金幣標示，
// 其他六種貨幣的換算還沒閉合，先不動它們——換算錯的症狀是玩家買得起
// 買不起的東西，而那在畫面上看不出來。
func (a *app) buy() {
	state := a.shop
	if state == nil || len(state.items) == 0 {
		return
	}
	if len(a.state.Party) == 0 {
		state.message = a.text(msgShopNoParty)
		return
	}
	if state.buyer >= len(a.state.Party) {
		state.buyer = 0
	}
	record := state.items[state.cursor]
	price := record.Price()
	member := a.state.Party[state.buyer]
	if uint32(member.Money[pooltreasure.Gold]) < uint32(price) {
		state.message = fmt.Sprintf(a.text(msgShopNoGold), member.Name, member.Money[pooltreasure.Gold], price)
		return
	}
	rawInventory := make([][]byte, len(member.Inventory))
	for index := range member.Inventory {
		rawInventory[index] = member.Inventory[index].Raw
	}
	ok, err := poolcharacter.CanReceiveItem(member.Abilities[0], member.ExceptionalStrength,
		rawInventory, record.Raw[:])
	if err != nil {
		state.message = err.Error()
		return
	}
	if !ok {
		state.message = fmt.Sprintf(a.text(msgShopOverloaded), member.Name)
		return
	}
	item := poolsave.Item{Name: record.Name, Raw: append([]byte(nil), record.Raw[:]...)}
	a.state.Party[state.buyer].Money[pooltreasure.Gold] -= price
	a.state.Party[state.buyer].Inventory = append(a.state.Party[state.buyer].Inventory, item)
	for index := range a.state.CharacterLibrary {
		if a.state.CharacterLibrary[index].Name == member.Name {
			a.state.CharacterLibrary[index].Money[pooltreasure.Gold] -= price
			a.state.CharacterLibrary[index].Inventory =
				append(a.state.CharacterLibrary[index].Inventory, item)
		}
	}
	state.message = fmt.Sprintf(a.text(msgShopBought), member.Name, record.Name, price)
}

func (a *app) shopInput() error {
	state := a.shop
	switch {
	case a.justPressed(ebiten.KeyEscape):
		return a.leaveShop()
	case a.justPressed(ebiten.KeyTab):
		if len(a.state.Party) > 0 {
			state.buyer = (state.buyer + 1) % len(a.state.Party)
		}
		state.message = ""
	case a.justPressed(ebiten.KeyDown):
		state.cursor = (state.cursor + 1) % len(state.items)
	case a.justPressed(ebiten.KeyUp):
		state.cursor = (state.cursor - 1 + len(state.items)) % len(state.items)
	case a.justPressed(ebiten.KeyEnter):
		a.buy()
	case state.appraising && a.justPressed(ebiten.KeyS):
		a.resolveShopAppraise(false)
	case state.appraising && a.justPressed(ebiten.KeyK):
		a.resolveShopAppraise(true)
	case a.justPressed(ebiten.KeyG):
		return a.offerShopAppraise(appraiseGem)
	case a.justPressed(ebiten.KeyJ):
		return a.offerShopAppraise(appraiseJewel)
	}
	return nil
}

// offerShopAppraise 是商店那一側的 A）ppraise 入口。規則與神殿同一份
//（spec 116）：估好價之後只剩 S）ell 與 K）eep，背包滿了就沒有 K。
func (a *app) offerShopAppraise(kind appraiseKind) error {
	state := a.shop
	if state == nil || state.buyer >= len(a.state.Party) {
		return nil
	}
	character := &a.state.Party[state.buyer]
	slot := pooltreasure.Gems
	noun := a.text(msgShopAppraiseGem)
	if kind == appraiseJewel {
		slot, noun = pooltreasure.Jewelry, a.text(msgShopAppraiseJewel)
	}
	if character.Money[slot] == 0 {
		state.appraising = false
		state.message = a.text(msgShopAppraiseNone)
		return nil
	}
	character.Money[slot]--

	roll := a.rollDice(1, 100)
	var value int
	var err error
	if kind == appraiseGem {
		value, err = pooltreasure.GemValue(roll)
	} else {
		value, err = pooltreasure.JewelryValue(roll, func(limit int) int {
			return a.rollDice(1, limit) - 1
		})
	}
	if err != nil {
		return err
	}
	state.appraising, state.appraiseKind, state.appraiseValue = true, kind, value
	format := msgShopAppraiseValue
	if len(character.Inventory) >= pooltreasure.KeepNeedsRoom {
		format = msgShopAppraiseFull
	}
	state.message = fmt.Sprintf(a.text(format), noun, value, pooltreasure.SellPrice(value))
	return nil
}

// resolveShopAppraise 收下 S）ell 或 K）eep。背包滿了時 K 不生效——
// 原版連那一項都不印。
func (a *app) resolveShopAppraise(keep bool) {
	state := a.shop
	character := &a.state.Party[state.buyer]
	if keep && len(character.Inventory) >= pooltreasure.KeepNeedsRoom {
		return
	}
	if keep {
		character.Inventory = append(character.Inventory,
			keptTreasureItem(state.appraiseKind, state.appraiseValue))
		state.message = a.text(msgShopAppraiseKept)
	} else {
		paid := pooltreasure.SellPrice(state.appraiseValue)
		character.Money[pooltreasure.Gold] += uint16(paid)
		state.message = fmt.Sprintf(a.text(msgShopAppraiseSold), paid)
	}
	state.appraising = false
	syncTrainedLibraryCharacter(&a.state, *character)
}

// leaveShop 讓 ECL 從 COMBAT 邊界之後續行，與神殿離開走同一條路。
func (a *app) leaveShop() error {
	a.shopActive, a.shop = false, nil
	a.eventLabel = ""
	if a.eventSession == nil {
		a.cellEventPending, a.cellWaitingMenu = false, false
		return nil
	}
	result, err := a.eventSession.RunUntilEvent(4096, nil, true)
	if err != nil {
		return fmt.Errorf("continue Pool shop service: %w", err)
	}
	return a.consumeInitialSearch(result)
}

// shopWindow 讓游標附近的項目留在畫面上：存貨可以有 57 筆，一頁放不下。
func (s *shopState) window(lines int) (first, last int) {
	first = s.cursor - lines/2
	if first < 0 {
		first = 0
	}
	if first+lines > len(s.items) {
		first = len(s.items) - lines
	}
	if first < 0 {
		first = 0
	}
	last = first + lines
	if last > len(s.items) {
		last = len(s.items)
	}
	return first, last
}

func drawShop(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	state := a.shop
	for y := 40; y < 372; y++ {
		for x := 32; x < 608; x++ {
			screen.Set(x, y, background)
		}
	}
	drawText(screen, a.text(msgShopTitle), 280, 62, accent)
	if len(a.state.Party) > 0 {
		if state.buyer >= len(a.state.Party) {
			state.buyer = 0
		}
		buyer := a.state.Party[state.buyer]
		drawText(screen, fmt.Sprintf(a.text(msgShopBuyer),
			state.buyer+1, len(a.state.Party), buyer.Name, buyer.Money[pooltreasure.Gold]),
			shopTextLeft, 92, accent)
	}
	first, last := state.window(shopLineCount)
	for index := first; index < last; index++ {
		record := state.items[index]
		cursor, ink := " ", foreground
		if index == state.cursor {
			cursor, ink = ">", accent
		}
		drawText(screen, fmt.Sprintf("%s%-34s %6d", cursor, record.Name, record.Price()),
			shopTextLeft, shopFirstLine+(index-first)*shopLineHeight, ink)
	}
	drawText(screen, fmt.Sprintf(a.text(msgShopCount), state.cursor+1, len(state.items)),
		shopTextLeft, 336, foreground)
	footer := a.text(msgShopFooter)
	if state.message != "" {
		footer = state.message
	}
	drawText(screen, footer, shopTextLeft, 356, accent)
}
