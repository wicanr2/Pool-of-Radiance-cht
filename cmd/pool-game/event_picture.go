package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// 事件裡的 `0Eh PICTURE`（spec 165）。
//
// 原版 overlay-03 `0822h..091Ch` 是這個 opcode 的 handler：
//
//   - 運算元不是 `FFh`：立 `DS:828Fh`、`DS:82A6h`，看 head 選擇子
//     （`[4937h]+5C2h`，就是 ECL `6DE1h`）。是 `FFh` 就載入 `PIC<區號>.DAX`
//     的那一個區塊、把第 1 張畫在 `(3,3)`；不是就交給 overlay-07 `051Eh`
//     疊 `HEAD`／`BODY`（spec 117），同一個位置。`(3,3)` 是 8 像素為單位，
//     就是第一人稱內框的原點。
//   - 運算元是 `FFh`：重畫視野（overlay-27 entry 1），清掉 `82A6h`。
//
// 圖一直留在框裡，直到視野被重畫：`PICTURE 255`、`COMBAT` 回來之後
// （overlay-03 `19BBh` 清 `82A6h`，接著 overlay-25 entry 37 照 `DS:4954h`
// 重畫）、腳本 `EXIT` 回到主迴圈（`0090h` 清 `82A6h`，主迴圈 `391Ch` 重畫視野）。
//
// `PIC` 那一條是動畫：只有在 `HORIZONTAL MENU`（`2Bh`）與 `ENCOUNTER MENU`
// （`29h`）等鍵時才會動——那兩支把 `82A6h && 82A7h` 當成「要動」傳給選單元件
// （overlay-03 `1238h..124Ch`、`2200h..2213h`），而疊 HEAD／BODY 的
// overlay-07 `0521h` 會把 `82A7h` 清成 0。`PARLAY` 與神殿、商店的服務選單傳 0。

// eventPicture 是現在蓋在第一人稱框上的事件圖片。
type eventPicture struct {
	source eclPicture
	// frames 是照當下色盤畫好的每一張；HEAD／BODY 那一條只有一張。
	frames []*ebiten.Image
	// ticks 是每一張停幾個影格（60 fps）。
	ticks  []int
	frame  int
	tick   int
	tried  bool
	moving bool
}

// eventPictureFramesPerBIOSTick 是一個 BIOS tick（54.9 ms）在 60 fps 底下佔幾個影格，
// 與營火同一個取整（`campFireTicksPerFrame`）。
const eventPictureFramesPerBIOSTick = campFireTicksPerFrame

// picAnimationTicks 把 PIC 每一張的延遲換成影格數。選單元件的等鍵迴圈在
// `經過的 BIOS tick > 延遲 ÷ 7` 時才換張（overlay-26 `0277h..02B8h`，嚴格大於），
// 所以一張停 `延遲 ÷ 7 + 1` 個 tick。
func picAnimationTicks(delay uint32) int {
	return (int(delay/7) + 1) * eventPictureFramesPerBIOSTick
}

// showEventPicture 照 `PICTURE` 的運算元換圖或收圖。
func (a *app) showEventPicture(picture eclPicture) {
	if !picture.set || picture.value == 0xFF {
		a.clearEventPicture()
		return
	}
	a.eventPicture = eventPicture{source: picture}
}

// clearEventPicture 在視野重畫的時候收掉事件圖片。
func (a *app) clearEventPicture() {
	a.eventPicture = eventPicture{}
}

// eventPictureShown 說這一刻框裡是不是事件圖片。圖只在事件進行中才會在框裡：
// 腳本 `EXIT` 之後主迴圈就重畫視野了。
func (a *app) eventPictureShown() bool {
	return a.eventPicture.source.set && a.cellEventPending && !a.introWaiting && !a.tourActive
}

// eventPictureImage 回傳這一刻要畫的那一張；載不到就回 nil，框裡照畫視野。
func (a *app) eventPictureImage() *ebiten.Image {
	if !a.eventPictureShown() {
		return nil
	}
	state := &a.eventPicture
	if !state.tried {
		state.tried = true
		a.loadEventPicture(state)
	}
	if len(state.frames) == 0 {
		return nil
	}
	return state.frames[state.frame%len(state.frames)]
}

func (a *app) loadEventPicture(state *eventPicture) {
	archive := uint8(a.spawn.Map.Archive)
	source := state.source
	if source.head != 0xFF {
		if a.loadNPCPortrait == nil {
			return
		}
		portrait, err := a.loadNPCPortrait(archive, source.head, source.value)
		if err != nil {
			return
		}
		state.frames, state.ticks, state.moving = []*ebiten.Image{portrait}, []int{0}, false
		return
	}
	if a.loadPICAnimation == nil {
		return
	}
	frames, delays, err := a.loadPICAnimation(archive, source.value)
	if err != nil || len(frames) == 0 || len(delays) != len(frames) {
		return
	}
	ticks := make([]int, len(delays))
	for index, delay := range delays {
		ticks[index] = picAnimationTicks(delay)
	}
	state.frames, state.ticks, state.moving = frames, ticks, len(frames) > 1
}

// eventPictureAnimating 說選單元件這一刻會不會推動畫。
func (a *app) eventPictureAnimating() bool {
	return a.cellWaitingMenu && !a.templeActive && !a.shopActive && !a.whoPending &&
		!a.treasureActive
}

// tickEventPicture 推動畫一格；由 Update 每個影格呼叫。
func (a *app) tickEventPicture() {
	// 圖由繪製那一邊載入；Update 不自己建圖（沒有畫面迴圈的測試裡，建出來的
	// 圖不會被送出，只會一直堆著）。
	state := &a.eventPicture
	if !a.eventPictureShown() || len(state.frames) == 0 {
		return
	}
	if !state.moving || !a.eventPictureAnimating() {
		state.tick = 0
		return
	}
	state.tick++
	if state.tick >= state.ticks[state.frame] {
		state.tick = 0
		state.frame = (state.frame + 1) % len(state.frames)
	}
}

// drawEventPicture 把事件圖片蓋在第一人稱框上，與視野同樣放大兩倍。
func (a *app) drawEventPicture(screen *ebiten.Image, left, top int) {
	picture := a.eventPictureImage()
	if picture == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(left), float64(top))
	screen.DrawImage(picture, op)
}
