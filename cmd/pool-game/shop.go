package main

import (
	"fmt"
	"image/color"
	"strings"

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
	// 賣出（spec 067〈賣出〉）：V 打開目前這個人的物品，S 出價，Y 成交。
	selling   bool
	sellItem  int
	sellStage sellStage
	sellPrice uint16
	// leaving：公款還有錢時按 ESC，店主問要不要回去拿（overlay-06 `0684h..0722h`）。
	leaving bool
}

// sellStage 是賣出那一頁等的是哪一個問題。
type sellStage int

const (
	sellPicking sellStage = iota
	// sellScribe：卷軸上有人正要抄法術，先問丟不丟得（overlay-19 `0DD1h`）。
	sellScribe
	// sellOffer：`I'll give you N gold pieces for your X` / `Is It a Deal?`（`1D55h`）。
	sellOffer
	// sellIdentify：`For 200 gold pieces I'll identify your X` / `Is It a Deal?`
	//（overlay-19 entry 17 `1F7Ch..1FCAh`）。
	sellIdentify
)

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
	// 進店先把公款七欄清成 0（overlay-06 `0548h..0554h`：`FillChar(DS:6752h, 1Ch, 0)`，
	// spec 067〈公款〉）。上一家店離開時說「不拿了」留下的錢就在這裡消失。
	a.state.PooledMoney = [pooltreasure.CurrencyCount]uint32{}
	a.shop = &shopState{items: stock}
	a.shopActive = true
	a.cellEventPending, a.cellWaitingMenu = true, true
	a.cellMenuOptions, a.cellMenuCursor = nil, 0
	a.eventLabel = a.text(msgShopTitle)
	a.statusLine = fmt.Sprintf(a.text(msgShopStatus), len(stock))
	return nil
}

// buy 把選中的物品賣給目前的買家（overlay-06 entry 4 `034Fh`，#30，spec 116〈付款〉）：
// 先算買家五種硬幣的金幣等值（overlay-19 entry 11，四捨五入），夠就從他身上扣、餘額
// 重鑄成白金＋金（overlay-21 entry 15）；不夠才看隊伍 pool（entry 17／16）；兩邊都不夠
// 印 "Not enough money."。角色與 pool 不混付。收下物品那一步在前（entry 2，spec 035），
// 超重就不扣錢。
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
	price := int64(record.Price())
	member := a.state.Party[state.buyer]
	have := pooltreasure.GoldEquivalent(member.Money)
	pooled := pooltreasure.PoolGoldEquivalent(a.state.PooledMoney)
	if have < price && pooled < price {
		state.message = fmt.Sprintf(a.text(msgShopNoGold), member.Name, have, price)
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
	source, paid, err := pooltreasure.PayGold(&a.state, state.buyer, price)
	if err != nil {
		state.message = err.Error()
		return
	}
	if !paid {
		state.message = fmt.Sprintf(a.text(msgShopNoGold), member.Name, have, price)
		return
	}
	item := poolsave.Item{Name: record.Name, Raw: append([]byte(nil), record.Raw[:]...)}
	a.state.Party[state.buyer].Inventory = append(a.state.Party[state.buyer].Inventory, item)
	for index := range a.state.CharacterLibrary {
		if a.state.CharacterLibrary[index].Name == member.Name {
			a.state.CharacterLibrary[index].Inventory =
				append(a.state.CharacterLibrary[index].Inventory, item)
		}
	}
	if source == pooltreasure.PaidByPool {
		state.message = fmt.Sprintf(a.text(msgShopBoughtFromPool), member.Name, record.Name, price)
		return
	}
	state.message = fmt.Sprintf(a.text(msgShopBought), member.Name, record.Name, price)
}

func (a *app) shopInput() error {
	state := a.shop
	if state.selling {
		a.shopSellInput()
		return nil
	}
	if state.leaving {
		// `~Yes ~No`：選單回 1（No）就離店，錢留在公款裡；Yes 回到商店選單。
		switch {
		case a.justPressed(ebiten.KeyN):
			state.leaving = false
			return a.leaveShop()
		case a.justPressed(ebiten.KeyY):
			state.leaving, state.message = false, ""
		}
		return nil
	}
	switch {
	case a.justPressed(ebiten.KeyEscape):
		// overlay-06 `066Ch..0681h`：公款七欄有一欄非 0（overlay-21 entry 14）就先問。
		if a.hasPooledMoney() {
			state.leaving, state.message = true, a.text(msgShopLeaveMoney)
			return nil
		}
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
	case a.hasPooledMoney() && a.justPressed(ebiten.KeyS):
		// S）hare 只在公款有錢時出現在選單上（`05B7h`：選單字串 `0460h`／`0487h`），
		// 按下去走 overlay-21 entry 7，與戰利品的 Share 同一支（spec 040）。
		a.shareShopPool()
	case a.justPressed(ebiten.KeyG):
		return a.offerShopAppraise(appraiseGem)
	case a.justPressed(ebiten.KeyJ):
		return a.offerShopAppraise(appraiseJewel)
	case !state.appraising && a.justPressed(ebiten.KeyV):
		a.openShopSell()
	}
	return nil
}

// openShopSell 是商店選單的 V）iew → I）tems。原版先開人物資料頁（overlay-19
// entry 5）再按 I 進物品選單（entry 6）；remake 在商店裡沒有那一頁，所以 V 直接
// 列出目前這個人的物品。物品選單的 S）ell 只在店裡出現（`DS:4954h == 1`）。
func (a *app) openShopSell() {
	state := a.shop
	if len(a.state.Party) == 0 {
		state.message = a.text(msgShopNoParty)
		return
	}
	if state.buyer >= len(a.state.Party) {
		state.buyer = 0
	}
	state.selling, state.sellItem, state.sellStage = true, 0, sellPicking
	state.message = ""
}

func (a *app) shopSellInput() {
	state := a.shop
	if state.buyer >= len(a.state.Party) {
		state.selling = false
		return
	}
	inventory := a.state.Party[state.buyer].Inventory
	if state.sellStage == sellIdentify {
		// `1FCAh`：只有 Y 付錢，其他鍵都是不要。
		switch {
		case a.justPressed(ebiten.KeyY):
			a.completeIdentify()
		case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape):
			state.sellStage, state.message = sellPicking, ""
		}
		return
	}
	switch state.sellStage {
	case sellScribe, sellOffer:
		switch {
		case a.justPressed(ebiten.KeyY):
			if state.sellStage == sellScribe {
				a.offerSale()
				return
			}
			a.completeSale()
		case a.justPressed(ebiten.KeyN), a.justPressed(ebiten.KeyEscape):
			state.sellStage, state.message = sellPicking, ""
		}
		return
	}
	switch {
	case a.justPressed(ebiten.KeyEscape):
		state.selling, state.message = false, ""
	case a.justPressed(ebiten.KeyTab):
		state.buyer = (state.buyer + 1) % len(a.state.Party)
		state.sellItem, state.message = 0, ""
	case a.justPressed(ebiten.KeyDown):
		if len(inventory) > 0 {
			state.sellItem = (state.sellItem + 1) % len(inventory)
		}
	case a.justPressed(ebiten.KeyUp):
		if len(inventory) > 0 {
			state.sellItem = (state.sellItem - 1 + len(inventory)) % len(inventory)
		}
	case a.justPressed(ebiten.KeyS):
		a.startSale()
	case a.justPressed(ebiten.KeyI):
		a.startIdentify()
	}
}

// shareShopPool 是商店選單的 S）hare：公款平分給隊員（overlay-21 entry 7）。
func (a *app) shareShopPool() {
	next := cloneSaveState(a.state)
	if err := pooltreasure.ShareMoney(&next); err != nil {
		a.shop.message = err.Error()
		return
	}
	a.state = next
	a.shop.message = a.text(msgShopPoolShared)
}

// startIdentify 是物品選單的 I）d（overlay-19 entry 17 `1F52h`）。選項只看
// `DS:4954h == 1`（在店裡），不像 Sell 還看角色（`1119h`）。名稱先重組一次再印
//（`1F77h` 呼叫 overlay-25 entry 1）。
func (a *app) startIdentify() {
	state := a.shop
	member := a.state.Party[state.buyer]
	if len(member.Inventory) == 0 {
		return
	}
	if state.sellItem >= len(member.Inventory) {
		state.sellItem = 0
	}
	name, err := a.identifyName(member.Inventory[state.sellItem])
	if err != nil {
		state.message = err.Error()
		return
	}
	state.sellStage = sellIdentify
	state.message = fmt.Sprintf(a.text(msgShopIdentifyOffer), pooltreasure.IdentifyPrice, name)
}

// identifyName 是 overlay-25 entry 1 重組的名稱，去掉尾端空白。
func (a *app) identifyName(item poolsave.Item) (string, error) {
	if a.itemNames == nil {
		return "", fmt.Errorf("Pool item name table is not loaded")
	}
	name, err := pooltreasure.ItemName(item.Raw, a.itemNames)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(name, " "), nil
}

// completeIdentify 是按 Y（`1FD1h..20F4h`）：付 200 金，`+35h` 有藏字就清掉、
// 名稱重組；沒有藏字照樣收錢。
func (a *app) completeIdentify() {
	state := a.shop
	state.sellStage = sellPicking
	if a.itemNames == nil {
		state.message = "Pool item name table is not loaded"
		return
	}
	before, err := a.identifyName(a.state.Party[state.buyer].Inventory[state.sellItem])
	if err != nil {
		state.message = err.Error()
		return
	}
	result, err := pooltreasure.IdentifyItem(&a.state, state.buyer, state.sellItem, a.itemNames)
	if err != nil {
		state.message = err.Error()
		return
	}
	switch result.Outcome {
	case pooltreasure.IdentifyNotEnoughMoney:
		state.message = a.text(msgShopIdentifyNoMoney)
	case pooltreasure.IdentifyNothingNew:
		state.message = fmt.Sprintf(a.text(msgShopIdentifyNothingNew), before)
	default:
		state.message = fmt.Sprintf(a.text(msgShopIdentifyRevealed), result.Name)
	}
}

// shopCanSell 是物品選單放不放 `Sell` 的條件（overlay-19 `10C9h..10EFh`）：
// 記錄 `+84h` 位元 7 沒立（不是帶士氣的 NPC）、或 `+10Dh == 0`、或 `+10Ch == 1`。
// 玩家建的角色 `+84h` 是 0，一律可以賣；只有 ADD NPC 帶進來的記錄才看得到那三格。
func shopCanSell(member poolsave.Character) bool {
	if !member.NPC || len(member.Record) <= 0x10d {
		return true
	}
	return member.Record[0x84] < 0x80 || member.Record[0x10d] == 0 || member.Status == 1
}

// startSale 是按 S：先過 entry 20（`0D72h`）的放手檢查，再出價。
func (a *app) startSale() {
	state := a.shop
	member := a.state.Party[state.buyer]
	if len(member.Inventory) == 0 {
		return
	}
	if !shopCanSell(member) {
		state.message = fmt.Sprintf(a.text(msgShopSellNotAllowed), member.Name)
		return
	}
	if state.sellItem >= len(member.Inventory) {
		state.sellItem = 0
	}
	item := member.Inventory[state.sellItem]
	if pooltreasure.SellNeedsUnready(item.Raw) {
		state.message = a.text(msgShopSellUnready)
		return
	}
	if category, ok := a.itemCategory(item); ok && pooltreasure.SellNeedsScribeConfirm(item.Raw, category) {
		state.sellStage = sellScribe
		state.message = fmt.Sprintf(a.text(msgShopSellScribe), member.Name)
		return
	}
	a.offerSale()
}

// offerSale 印出價，等 Y／N（entry 16 `1D55h..1DE3h`）。
func (a *app) offerSale() {
	state := a.shop
	item := a.state.Party[state.buyer].Inventory[state.sellItem]
	price, err := pooltreasure.SellOffer(item.Raw)
	if err != nil {
		state.sellStage, state.message = sellPicking, err.Error()
		return
	}
	state.sellStage, state.sellPrice = sellOffer, price
	state.message = fmt.Sprintf(a.text(msgShopSellOffer), price, item.Name)
}

// completeSale 是按 Y：`Sold!`，物品摘掉，錢照 entry 16 的分法進錢包或公款。
func (a *app) completeSale() {
	state := a.shop
	result, err := pooltreasure.SellItem(&a.state, state.buyer, state.sellItem)
	state.sellStage = sellPicking
	if err != nil {
		state.message = err.Error()
		return
	}
	state.message = a.text(msgShopSellSold)
	if result.Overloaded {
		state.message += a.text(msgShopSellOverloaded)
	}
	if count := len(a.state.Party[state.buyer].Inventory); state.sellItem >= count && count > 0 {
		state.sellItem = count - 1
	}
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
		// 白金那一欄，同 spec 116。
		paid := pooltreasure.SellPrice(state.appraiseValue)
		character.Money[pooltreasure.Platinum] += uint16(paid)
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
	if state.selling {
		drawShopSell(screen, a, foreground, accent)
		return
	}
	if len(a.state.Party) > 0 {
		if state.buyer >= len(a.state.Party) {
			state.buyer = 0
		}
		buyer := a.state.Party[state.buyer]
		drawText(screen, fmt.Sprintf(a.text(msgShopBuyer),
			// 金幣等值（五種硬幣，entry 11）：付錢與賣出都把錢放在白金那一欄，
			// 只印金幣欄會看起來錢沒動。
			state.buyer+1, len(a.state.Party), buyer.Name, pooltreasure.GoldEquivalent(buyer.Money)),
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
	if a.hasPooledMoney() {
		footer = a.text(msgShopFooterPool)
	}
	if state.message != "" {
		footer = state.message
	}
	drawText(screen, footer, shopTextLeft, footerBaseline, accent)
}

// drawShopSell 畫賣出那一頁：目前這個人的物品，穿戴中的前面有標記。
// 原版是人物資料頁上的物品選單（overlay-19 entry 6），版面是 remake 的呈現。
func drawShopSell(screen *ebiten.Image, a *app, foreground, accent color.Color) {
	state := a.shop
	if state.buyer >= len(a.state.Party) {
		return
	}
	member := a.state.Party[state.buyer]
	drawText(screen, fmt.Sprintf(a.text(msgShopSellTitle), state.buyer+1, len(a.state.Party), member.Name),
		shopTextLeft, 92, accent)
	if len(member.Inventory) == 0 {
		drawText(screen, a.text(msgShopSellNoItems), shopTextLeft, shopFirstLine, foreground)
	}
	first := 0
	if state.sellItem >= shopLineCount {
		first = state.sellItem - shopLineCount + 1
	}
	for index := first; index < len(member.Inventory) && index < first+shopLineCount; index++ {
		item := member.Inventory[index]
		cursor, ink := " ", foreground
		if index == state.sellItem {
			cursor, ink = ">", accent
		}
		marker := "  "
		if pooltreasure.SellNeedsUnready(item.Raw) {
			marker = a.text(msgEquipmentReadyMark)
		}
		drawText(screen, cursor+marker+item.Name, shopTextLeft,
			shopFirstLine+(index-first)*shopLineHeight, ink)
	}
	footer := a.text(msgShopSellFooter)
	if state.message != "" {
		footer = state.message
	}
	drawText(screen, footer, shopTextLeft, footerBaseline, accent)
}
