package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// recordPortraitLoads 把 HEAD／BODY 與 PIC 的載入包一層，記下每一次要的是哪一塊。
type pictureLoads struct {
	portraits  [][3]uint8
	animations [][2]uint8
}

func (loads *pictureLoads) wrap(a *app) {
	portrait, animation := a.loadNPCPortrait, a.loadPICAnimation
	a.loadNPCPortrait = func(archive, head, body uint8) (*ebiten.Image, error) {
		loads.portraits = append(loads.portraits, [3]uint8{archive, head, body})
		return portrait(archive, head, body)
	}
	a.loadPICAnimation = func(archive, block uint8) ([]*ebiten.Image, []uint32, error) {
		loads.animations = append(loads.animations, [2]uint8{archive, block})
		return animation(archive, block)
	}
}

// 蘇恩神殿（spec 165）：走進 (1,3) 之後，腳本 `A0BCh SAVE 22 → @6DE1`、
// `A0C2h PICTURE 24`，那一框從「DO YOU SEEK HEALING?」起就是祭司（HEAD3/22 疊
// BODY3/24，與 dosgolem 基準逐格相同，見 internal/assets 的
// `TestSuneTemplePriestessMatchesTheDOSTempleShot`），服務選單時還在；選 Exit 之後
// 腳本 `AA6Bh SAVE FFh → @6DE1`、`AA71h PICTURE 255`，框裡換回視野。
func TestSuneTempleShowsThePriestessUntilTheScriptLeaves(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application := bootCityParty(t, zipPath)
	if application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在城區，而是 GEO%d/%d",
			application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	var loads pictureLoads
	loads.wrap(application)
	if application.eventPictureImage() != nil {
		t.Fatal("還沒進神殿，框裡就有事件圖片")
	}
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 1, 4, 0
	pressKeys(t, application, ebiten.KeyArrowUp)
	for tick := 0; tick < 50 && !application.cellWaitingMenu; tick++ {
		pressKeys(t, application, ebiten.KeyEnter)
	}
	if !application.cellWaitingMenu || application.spawn.X != 1 || application.spawn.Y != 3 {
		t.Fatalf("沒有停在神殿門口的選單：(%d,%d) %q", application.spawn.X, application.spawn.Y, application.eventText)
	}
	if got := application.eventPicture.source; got != (eclPicture{head: 22, value: 24, set: true}) {
		t.Fatalf("神殿門口記下的是 %+v，要 HEAD 22／BODY 24", got)
	}
	if application.eventPictureImage() == nil {
		t.Fatal("DO YOU SEEK HEALING? 那一頁框裡沒有祭司")
	}
	if len(loads.portraits) != 1 || loads.portraits[0] != [3]uint8{3, 22, 24} {
		t.Fatalf("載入的是 %v，要一次 HEAD3/22＋BODY3/24", loads.portraits)
	}
	pressKeys(t, application, ebiten.KeyEnter) // YES
	if !application.templeActive {
		t.Fatalf("答 YES 之後沒有進神殿：%q", application.eventText)
	}
	if application.eventPictureImage() == nil {
		t.Fatal("神殿服務選單時框裡沒有祭司（原版 64-y 那一框是祭司）")
	}
	// HEAD／BODY 那一條不會動：原版 `051Eh` 把 `82A7h` 清成 0。
	for tick := 0; tick < 30; tick++ {
		application.keys = scriptedKeys{}
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if application.eventPicture.frame != 0 {
		t.Fatalf("HEAD／BODY 的圖跑到第 %d 張", application.eventPicture.frame)
	}
	pressKeys(t, application, ebiten.KeyArrowLeft, ebiten.KeyEnter) // Exit
	for tick := 0; tick < 50 && application.cellEventPending; tick++ {
		pressKeys(t, application, ebiten.KeyEnter)
	}
	if application.templeActive || application.eventPicture.source.set || application.eventPictureImage() != nil {
		t.Fatalf("離開神殿之後框裡還是事件圖片：temple %v picture %+v",
			application.templeActive, application.eventPicture.source)
	}
	if len(loads.portraits) != 1 {
		t.Fatalf("祭司載了 %d 次，應該快取", len(loads.portraits))
	}
}

// 碼頭的船（spec 165）：有船票走上 (15,1)，腳本 `9BB9h PICTURE 41` 時 `6DE1h`
// 還是入口 1 開頭 `99F1h` 寫的 `FFh`，所以走 PIC 那一條——`PIC3.DAX` 區塊 41，
// 四張的船，延遲 20／15／20／15。接著的 `GOSUB AF1Ch` 是 `HORIZONTAL MENU`，
// 等 RETURN 的時候船照延遲換張：一張停 `延遲 ÷ 7 + 1` 個 BIOS tick，也就是
// 3 個 tick、9 個影格（overlay-26 `0277h..02B8h`）。按下 RETURN 上船、換區塊，
// 框裡就不再是船。
func TestTheBoatAnimatesWhileTheDockWaitsForReturn(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application := bootCityParty(t, zipPath)
	if application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在城區，而是 GEO%d/%d",
			application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	// 與 `TestBuyingTheEastRouteSailsIntoTheWilderness` 同一條路：先向港務長買票。
	application.eventMachine.Memory[0x4AA7] = 254
	application.eventMachine.Memory[0x4A01] = 255
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 11, 2, 0
	pressKeys(t, application, ebiten.KeyArrowUp)
	for tick := 0; tick < 200 && !application.cellWaitingMenu; tick++ {
		pressKeys(t, application, ebiten.KeyEnter)
	}
	if !application.cellWaitingMenu {
		t.Fatal("港務長沒有把選單擺出來")
	}
	pressKeys(t, application, ebiten.KeyArrowRight) // EAST
	for tick := 0; tick < 400 && application.cellEventPending; tick++ {
		pressKeys(t, application, ebiten.KeyEnter)
	}
	if got := application.eventMachine.Memory[0x4A01]; got != 1 {
		t.Fatalf("買完票 4A01 = %d，要 1", got)
	}
	var loads pictureLoads
	loads.wrap(application)
	application.spawn.X, application.spawn.Y, application.spawn.Facing = 14, 1, 1
	pressKeys(t, application, ebiten.KeyArrowUp)
	if !application.cellWaitingMenu || application.spawn.Map.BlockID != 0 {
		t.Fatalf("走上碼頭之後沒有停在「上船」那一頁：GEO%d/%d 文字 %q",
			application.spawn.Map.Archive, application.spawn.Map.BlockID, application.eventText)
	}
	if got := application.eventPicture.source; got != (eclPicture{head: 0xFF, value: 41, set: true}) {
		t.Fatalf("碼頭記下的是 %+v，要 PIC 區塊 41", got)
	}
	first := application.eventPictureImage()
	if first == nil || len(loads.animations) != 1 || loads.animations[0] != [2]uint8{3, 41} {
		t.Fatalf("船沒有從 PIC3/41 載入：%v", loads.animations)
	}
	if len(application.eventPicture.frames) != 4 {
		t.Fatalf("船有 %d 張，要 4", len(application.eventPicture.frames))
	}
	idle := func(count int) {
		for tick := 0; tick < count; tick++ {
			application.keys = scriptedKeys{}
			if err := application.Update(); err != nil {
				t.Fatal(err)
			}
		}
	}
	// 每一張 (20÷7+1)=3 個 BIOS tick ＝ 9 個影格；15÷7+1 也是 3。
	for _, want := range []struct{ updates, frame int }{
		{8, 0}, {1, 1}, {8, 1}, {1, 2}, {9, 3}, {9, 0},
	} {
		idle(want.updates)
		if application.eventPicture.frame != want.frame {
			t.Fatalf("等了之後停在第 %d 張，要第 %d 張", application.eventPicture.frame, want.frame)
		}
	}
	if application.eventPictureImage() != first {
		t.Fatal("繞一圈回到第 0 張，畫的卻不是第一張")
	}
	for tick := 0; tick < 600 && application.spawn.Map.BlockID == 0; tick++ {
		pressKeys(t, application, ebiten.KeyEnter)
	}
	if application.eventPictureImage() != nil {
		t.Fatalf("上船換區之後框裡還是船：%+v", application.eventPicture.source)
	}
}

// 神殿、商店與 WHO 的選單不推動畫（spec 165：overlay-04／06 傳 0）；
// 事件選單才推。
func TestEventPictureOnlyAnimatesInEventMenus(t *testing.T) {
	frames := []*ebiten.Image{ebiten.NewImage(88, 88), ebiten.NewImage(88, 88)}
	a := &app{mode: modeAdventure, introDone: true, keys: scriptedKeys{}}
	a.loadPICAnimation = func(archive, block uint8) ([]*ebiten.Image, []uint32, error) {
		return frames, []uint32{2, 2}, nil
	}
	a.showEventPicture(eclPicture{head: 0xFF, value: 29, set: true})
	a.cellEventPending, a.cellWaitingMenu, a.templeActive = true, true, true
	// 圖由繪製那一邊載入（Update 不建圖）。
	if a.eventPictureImage() == nil {
		t.Fatal("PIC 那一條沒有載入")
	}
	for tick := 0; tick < 10; tick++ {
		a.tickEventPicture()
	}
	if a.eventPicture.frame != 0 {
		t.Fatalf("神殿的服務選單推了動畫（第 %d 張）", a.eventPicture.frame)
	}
	a.templeActive = false
	for tick := 0; tick < eventPictureFramesPerBIOSTick; tick++ {
		a.tickEventPicture()
	}
	if a.eventPicture.frame != 1 {
		t.Fatalf("事件選單等了一個 BIOS tick 還在第 %d 張", a.eventPicture.frame)
	}
	// 腳本 EXIT 之後框裡是視野。
	a.cellEventPending, a.cellWaitingMenu = false, false
	if a.eventPictureImage() != nil {
		t.Fatal("事件結束了還畫事件圖片")
	}
	a.cellEventPending = true
	a.showEventPicture(eclPicture{head: 0xFF, value: 0xFF, set: true})
	if a.eventPictureImage() != nil {
		t.Fatal("PICTURE 255 沒有收圖")
	}
}
