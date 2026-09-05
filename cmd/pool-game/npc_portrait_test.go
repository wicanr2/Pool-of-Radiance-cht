package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// 半身像只在 APPROACH 的那一頁蓋上去；導覽開始之後那一框要變回視野。
// 原版兩張基準圖一正一反：`03-rolf-approach.png`（`15, 1 W`）有半身像，
// `05-tyr-stop-11-2-south.png`（`11, 2 S`）沒有。
func TestApproachPortraitOnlyCoversTheGreetingPage(t *testing.T) {
	event := gamepack.InitialEvent{
		Position:     gamepack.Spawn{Map: gamepack.MapKey{Archive: 3, BlockID: 0}, X: 15, Y: 1, Facing: 3},
		MonsterID:    12,
		SpriteBlock:  12,
		PortraitBody: 9,
	}
	var asked [][3]uint8
	application := &app{
		spawn:        gamepack.DOSInitialSpawn(),
		initialEvent: &event,
		introWaiting: true,
		loadNPCPortrait: func(archive, head, body uint8) (*ebiten.Image, error) {
			asked = append(asked, [3]uint8{archive, head, body})
			return ebiten.NewImage(88, 88), nil
		},
	}
	if application.approachPortrait() == nil {
		t.Fatal("APPROACH 的第一頁應該蓋半身像")
	}
	want := [3]uint8{3, gamepack.RolfPortraitHeadBlock, 9}
	if len(asked) != 1 || asked[0] != want {
		t.Fatalf("要到的是 %v，預期一次 %v", asked, want)
	}
	// 第二次要用快取，不再讀檔。
	if application.approachPortrait() == nil || len(asked) != 1 {
		t.Fatalf("第二次又去讀了一次檔：%v", asked)
	}
	// 導覽開始之後不再蓋——導覽的每一頁也在等 Return（`introWaiting` 仍是
	// 真），所以只看那一個旗標會整趟都蓋著。
	application.tourActive = true
	if application.approachPortrait() != nil {
		t.Fatal("導覽那幾頁不該蓋半身像")
	}
	application.tourActive, application.introWaiting = false, false
	if application.approachPortrait() != nil {
		t.Fatal("沒在等 Return 的時候不該蓋半身像")
	}
	// 換主題要重疊一次，否則半身像會留在舊色盤上。
	application.introWaiting = true
	application.clearNPCPortrait()
	if application.approachPortrait() == nil || len(asked) != 2 {
		t.Fatalf("換主題之後沒有重疊：%v", asked)
	}
}

// 疊不出來就退回視野，事件本身不能被一張圖擋住。
func TestApproachPortraitFallsBackToTheView(t *testing.T) {
	event := gamepack.InitialEvent{PortraitBody: 9}
	calls := 0
	application := &app{
		spawn:        gamepack.DOSInitialSpawn(),
		initialEvent: &event,
		introWaiting: true,
		loadNPCPortrait: func(archive, head, body uint8) (*ebiten.Image, error) {
			calls++
			return nil, errPortrait
		},
	}
	if application.approachPortrait() != nil {
		t.Fatal("疊不出來時不該回一張圖")
	}
	if application.approachPortrait() != nil || calls != 1 {
		t.Fatalf("失敗之後又試了一次：calls=%d", calls)
	}
}

var errPortrait = portraitError{}

type portraitError struct{}

func (portraitError) Error() string { return "沒有這個區塊" }
