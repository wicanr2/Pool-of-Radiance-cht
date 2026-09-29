package main

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// 商店主選單按 B）uy 之後的貨品清單（spec 168）。
//
// 版面照 dosgolem 在菲蘭武具店按 `b` 那一幀（`workplace/dosgolem-ref-shop/80-b`，
// 雜湊 `4d5cbdcf`）：整頁外框、上方置中的 `SHOP`、一頁 19 行的清單（名稱從第 1 欄、
// 價格靠右到第 30 欄）、反白那一行白色其餘綠色、框外底列 `ITEMS: BUY NEXT EXIT`。
// 清單順序是 ITEM 記錄的**倒序**（武具店第一項是 `BATTLE AXE`，檔案裡它是最後一筆），
// 一打開反白在第二項。

// 這一段訊息另開 `iota + 6100`，在 init 登記進 messageKeys，重號直接 panic。
const (
	msgShopListPrefix messageID = iota + 6100
	msgShopListNext
	msgShopListPrev
)

func init() {
	for id, key := range map[messageID]string{
		msgShopListPrefix: "ui.shopListPrefix",
		msgShopListNext:   "ui.shopListNext",
		msgShopListPrev:   "ui.shopListPrev",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// 版面常數是**邏輯座標**（640×400，原版 native 的兩倍）。列給的是基線：
// `drawText` 把字畫在 `Row-14 .. Row-1`，原版文字最後一列 native `n` 對應 `2n+2`
// （同 spec 134 那一頁的換算）。
const (
	// shopListLines 是一頁幾行：原版 native 32..182，行距 8，共 19 行。
	shopListLines = 19
	// shopListFirstRow 是第一行的基線：原版 native 32..38。
	shopListFirstRow = 78
	shopListPitch    = 16
	// shopListLeft 是名稱的左緣：原版第 1 欄（native 8），字身從 9 起。
	shopListLeft = spellPageLeft
	// shopListPriceRight 是價格的右緣：原版靠右到第 30 欄（native 240..247）。
	shopListPriceRight = 496
	// shopListTitleCentre 是標題 `SHOP` 的中線：原版第 17..20 欄（native 136..167）。
	shopListTitleCentre = 304
	// shopListBuyerRow 是買家那一行（只在隊伍不只一人時畫，見 drawShopBuyList）：
	// 原版標題與清單之間 native 24..30 那一列是空的。
	shopListBuyerRow = 62
)

// shopStock 把 TREASURE 請求讀到的記錄排成原版清單的順序：整串倒過來。
// 武具店只有一個 block（`35h`），原版第一頁是 `BATTLE AXE, HAND AXE, BARDICHE…`，
// 檔案裡是 `…Bardiche, Hand Axe, Battle Axe`；翻頁到最後一頁的最後一行是 `SHIELD`，
// 檔案的第一筆（dosgolem `84-n`／`85-n`）。不只一個 block 的店（目前沒有）照同一條
// 倒過來是推論。
func shopStock(records []gamepack.TreasureItemRecord) []gamepack.TreasureItemRecord {
	stock := slices.Clone(records)
	slices.Reverse(stock)
	return stock
}

// openShopBuyList 打開貨品清單。原版每次打開反白都在第二項：第一次開是
// 第 0 行起、反白第 1 行（`80-b`）；翻到最後一頁、離開清單再按 `b`，清單從
// `HAND AXE` 起、反白在第 0 行（`88-b`）——反白回到第二項，起點只移到看得見它。
func (a *app) openShopBuyList() {
	state := a.shop
	state.buying, state.message = true, ""
	state.cursor = 0
	if len(state.items) > 1 {
		state.cursor = 1
	}
	state.keepCursorVisible()
}

// keepCursorVisible 讓反白留在這一頁：在起點之前就把起點拉到它，
// 在一頁之後就讓它成為最後一行。
func (s *shopState) keepCursorVisible() {
	if s.cursor < s.top {
		s.top = s.cursor
	}
	if s.cursor >= s.top+shopListLines {
		s.top = s.cursor - shopListLines + 1
	}
	if s.top < 0 {
		s.top = 0
	}
}

// shopListHasNext／shopListHasPrev 決定底列有沒有 `NEXT`／`PREV`：
// 第一頁是 `BUY NEXT EXIT`，中間頁 `BUY NEXT PREV EXIT`，最後一頁 `BUY PREV EXIT`。
func (s *shopState) shopListHasNext() bool { return s.top+shopListLines < len(s.items) }
func (s *shopState) shopListHasPrev() bool { return s.top > 0 }

// shopListCommands 是清單那一頁的底列指令。
func (a *app) shopListCommands() []messageID {
	commands := []messageID{msgShopCommandBuy}
	if a.shop.shopListHasNext() {
		commands = append(commands, msgShopListNext)
	}
	if a.shop.shopListHasPrev() {
		commands = append(commands, msgShopListPrev)
	}
	return append(commands, msgShopCommandExit)
}

// shopBuyInput 是貨品清單那一頁。原版的鍵：`B` 買、`N` 下一頁、`P` 上一頁、
// `E` 回主選單；`End`／`Home` 移反白（spec 133 那一族，原版不認上下鍵）。
// NEXT 與 PREV 整頁跳，反白在頁內的那一行不變（`83-Home` 第 2 行 → `84-n` 第 2 行）。
// remake 另外接 `ENTER` 買、上下鍵移動、`ESC` 回主選單、`TAB` 換買家。
func (a *app) shopBuyInput() {
	state := a.shop
	count := len(state.items)
	switch {
	case a.justPressed(ebiten.KeyEscape), a.justPressed(ebiten.KeyE):
		state.buying, state.message = false, ""
	case a.justPressed(ebiten.KeyTab):
		if len(a.state.Party) > 0 {
			state.buyer = (state.buyer + 1) % len(a.state.Party)
		}
		state.message = ""
	case count == 0:
	case a.justPressed(ebiten.KeyDown), a.justPressed(ebiten.KeyEnd):
		state.cursor = (state.cursor + 1) % count
		state.keepCursorVisible()
	case a.justPressed(ebiten.KeyUp), a.justPressed(ebiten.KeyHome):
		state.cursor = (state.cursor - 1 + count) % count
		state.keepCursorVisible()
	case state.shopListHasNext() && (a.justPressed(ebiten.KeyN) || a.justPressed(ebiten.KeyPageDown)):
		state.top += shopListLines
		state.cursor = min(state.cursor+shopListLines, count-1)
		state.keepCursorVisible()
	case state.shopListHasPrev() && (a.justPressed(ebiten.KeyP) || a.justPressed(ebiten.KeyPageUp)):
		state.top = max(state.top-shopListLines, 0)
		state.cursor = max(state.cursor-shopListLines, 0)
		state.keepCursorVisible()
	case a.justPressed(ebiten.KeyEnter), a.justPressed(ebiten.KeyB):
		a.buy()
	}
}

// shopListName 是清單上那一件的名稱。原版用組名稱那一支（overlay-25 entry 1），
// 所以帶數量的記錄會變成複數：檔案的 `4 Dart` 在清單上是 `4 DARTS`、
// `20 Quarrel(s)` 是 `20 QUARRELS`。字詞表沒載進來時退回記錄裡的名稱。
func (a *app) shopListName(record gamepack.TreasureItemRecord) string {
	if name, err := a.identifyName(poolsave.Item{Name: record.Name, Raw: record.Raw[:]}); err == nil && name != "" {
		return name
	}
	return record.Name
}

// drawShopBuyList 畫貨品清單那一頁：清掉框內、標題、一頁 19 行、框外底列。
// 有話要說（買到了、錢不夠、拿不動）時底列換成那一句。
func drawShopBuyList(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	state := a.shop
	panel := ebiten.NewImage(logicalWidth-2*guidePanelInset, guidePanelBottom-spellPagePanelTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{
		GeoM: translated(guidePanelInset, spellPagePanelTop)})

	title := a.text(msgShopTitle)
	width := font.MeasureString(uiFace, displayText(title)).Ceil()
	drawText(screen, title, shopListTitleCentre-width/2, spellPageTitleRow, accent)

	// 原版的買家是主選單上選定的那個人，這一頁不寫出來。remake 用 TAB 換人，
	// 隊伍不只一人時要看得到現在是誰在付錢；一個人時與原版一樣留白。
	if len(a.state.Party) > 1 {
		if state.buyer >= len(a.state.Party) {
			state.buyer = 0
		}
		buyer := a.state.Party[state.buyer]
		drawText(screen, fmt.Sprintf(a.text(msgShopBuyer), state.buyer+1, len(a.state.Party),
			buyer.Name, pooltreasure.GoldEquivalent(buyer.Money)), shopListLeft, shopListBuyerRow, foreground)
	}

	for row := 0; row < shopListLines && state.top+row < len(state.items); row++ {
		index := state.top + row
		record := state.items[index]
		ink := foreground
		if index == state.cursor {
			ink = accent
		}
		y := shopListFirstRow + row*shopListPitch
		drawText(screen, a.shopListName(record), shopListLeft, y, ink)
		drawTextRight(screen, fmt.Sprintf("%d", record.Price()), shopListPriceRight, y, ink)
	}

	if state.message != "" {
		drawText(screen, state.message, 0, footerBaseline, accent)
		return
	}
	prefix := a.text(msgShopListPrefix)
	drawText(screen, prefix, 0, footerBaseline, a.commandPrefixInk(accent))
	x := font.MeasureString(uiFace, displayText(prefix)).Ceil()
	commands := a.shopListCommands()
	labels := make([]string, len(commands))
	for index, id := range commands {
		labels[index] = a.text(id)
	}
	drawCommandLabels(screen, x, labels, foreground, accent)
}
