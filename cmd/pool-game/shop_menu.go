package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// 商店的主選單照原版的構圖（spec 164）：第一人稱那一框換成店主肖像、右邊是
// `NAME AC HP` 名單、文字框清空、底列是 overlay-06 的選單字串
// `Buy View Pool Appraise Exit`（`0487h`），公款有錢時換成
// `Buy View Take Pool Share Appraise Exit`（`0460h`）。貨品清單要按 B）uy 才出現。

// 這一段訊息另開 `iota + 5700`，在 init 登記進 messageKeys，重號直接 panic。
const (
	msgShopCommandBuy messageID = iota + 5700
	msgShopCommandView
	msgShopCommandTake
	msgShopCommandPool
	msgShopCommandShare
	msgShopCommandAppraise
	msgShopCommandExit
	msgShopAppraiseChoose
)

func init() {
	for id, key := range map[messageID]string{
		msgShopCommandBuy:      "ui.shopCommandBuy",
		msgShopCommandView:     "ui.shopCommandView",
		msgShopCommandTake:     "ui.shopCommandTake",
		msgShopCommandPool:     "ui.shopCommandPool",
		msgShopCommandShare:    "ui.shopCommandShare",
		msgShopCommandAppraise: "ui.shopCommandAppraise",
		msgShopCommandExit:     "ui.shopCommandExit",
		msgShopAppraiseChoose:  "ui.shopAppraiseChoose",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// eclPicture 是最近一次 `0Eh PICTURE` 當下的狀態（spec 117）：`head` 是
// `6DE1h`（`[4937h]+5C2h`），不是 `FFh` 就走 HEAD／BODY 那一條、`value` 是
// BODY 區塊；是 `FFh` 就走 PIC 那一條。`PICTURE 255` 是收圖。
type eclPicture struct {
	head, value uint8
	set         bool
}

// picturePortraitHeadAddress 是 head 選擇子的 ECL 位址（spec 117〈Rolf 的 head 區塊〉）。
const picturePortraitHeadAddress = 0x6DE1

// recordECLPicture 記下這一張 PICTURE。它本身仍是純顯示的邊界（spec 082），
// 這裡只記，畫不畫由用到它的畫面決定——目前只有商店的主選單。
func (a *app) recordECLPicture(value uint16) {
	a.lastPicture = eclPicture{value: uint8(value), set: true, head: 0xFF}
	if a.eventMachine != nil {
		a.lastPicture.head = uint8(a.eventMachine.Memory[picturePortraitHeadAddress])
	}
}

// shopkeeperPortrait 是主選單那一框的店主。四家店都在進店前寫 `6DE1h` 再
// `PICTURE`：雜貨店與武具店是 head 42／body 9，珠寶店與銀器店是 head 63／
// body 34（ECL3/block 0 `A242h`／`A8BFh`／`A838h`／`A950h`，spec 164）。
// 走 PIC 那一條（`6DE1h == FFh`）或載不到時回 nil，畫面退回第一人稱視野。
func (a *app) shopkeeperPortrait() *ebiten.Image {
	state := a.shop
	if state == nil || state.buying || state.selling || state.take != nil {
		return nil
	}
	// `CAMP → ALTER → PICS` 的 `Portraits off`（`ds:4956h`），與 APPROACH 同一個開關。
	if a.portraitsHidden {
		return nil
	}
	if state.portrait != nil || state.portraitTried {
		return state.portrait
	}
	state.portraitTried = true
	picture := a.lastPicture
	if !picture.set || picture.head == 0xFF || picture.value == 0xFF || a.loadNPCPortrait == nil {
		return nil
	}
	portrait, err := a.loadNPCPortrait(uint8(a.spawn.Map.Archive), picture.head, picture.value)
	if err != nil {
		return nil
	}
	state.portrait = portrait
	return portrait
}

// shopMenuCommands 是底列的指令，照公款有沒有錢選兩個字串之一（overlay-06
// `05B7h`，spec 067〈離店時公款有錢就問〉）。
func (a *app) shopMenuCommands() []messageID {
	if a.hasPooledMoney() {
		return []messageID{msgShopCommandBuy, msgShopCommandView, msgShopCommandTake,
			msgShopCommandPool, msgShopCommandShare, msgShopCommandAppraise, msgShopCommandExit}
	}
	return []messageID{msgShopCommandBuy, msgShopCommandView, msgShopCommandPool,
		msgShopCommandAppraise, msgShopCommandExit}
}

// drawShopMenu 畫主選單在框裡框外的部分；肖像與名單由 drawAdventure 畫
// （`drawShopFrame`）。文字框原版是清空的，有話要說（估價、公款、錢不夠）
// 才印在框裡的第一行起。
func drawShopMenu(screen *ebiten.Image, a *app, background, foreground, accent color.Color) {
	state := a.shop
	panel := ebiten.NewImage(logicalWidth-2*shopMenuTextInset, dialogueBottom-dialogueTop)
	panel.Fill(background)
	screen.DrawImage(panel, &ebiten.DrawImageOptions{GeoM: translated(shopMenuTextInset, dialogueTop)})
	if state.message != "" {
		for index, line := range wrapDisplay(state.message, 68) {
			if index >= dialogueLines {
				break
			}
			drawText(screen, line, dialogueLeft, dialogueFirstRow+index*dialoguePitch, foreground)
		}
	}
	if state.leaving {
		// `~Yes ~No`（overlay-06 `0684h`）：這時等的是 Y／N，提問本身帶著兩個鍵，
		// 底列不畫商店選單。
		return
	}
	commands := a.shopMenuCommands()
	labels := make([]string, len(commands))
	for index, id := range commands {
		labels[index] = a.text(id)
	}
	drawCommandLabels(screen, 0, labels, foreground, accent)
}

// shopMenuTextInset 是清文字框時左右留下的寬度：外框一圈 tile（16）加上
// 不碰到繩索的一點空隙。
const shopMenuTextInset = 20

// drawShopFrame 在冒險畫面上畫主選單的那一框與名單。沒有肖像就回 false，
// 呼叫端照常畫第一人稱視野。
func drawShopFrame(screen *ebiten.Image, a *app, viewLeft, viewTop int, foreground, accent color.Color) bool {
	portrait := a.shopkeeperPortrait()
	if portrait == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(viewLeft), float64(viewTop))
	screen.DrawImage(portrait, op)
	// 買家只在隊伍不只一人時標出來：一個人的時候原版那一列就是一般的顏色。
	highlight := -1
	if len(a.state.Party) > 1 {
		highlight = a.shop.buyer
	}
	drawPartyRoster(screen, a, highlight, foreground, accent)
	return true
}
