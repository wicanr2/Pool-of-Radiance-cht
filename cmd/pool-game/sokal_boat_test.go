package main

// 索寇要塞的回程船（#44）：答 YES 之後落在**城區碼頭**，不是貧民窟。
//
// 原版 `ecl4/21` 入口 0（exact）：
//
//	9977 SAVE 15 @C04B ; SAVE 1 @C04C ; SAVE 3 @C04D
//	9989 SAVE 3 @6E12          ← 下一個封存檔是 3
//	998F NEWECL 0              ← 換到它的 block 0
//
// `6E12 = 3` 指定封存檔 3，所以落點是 ECL3/0 的碼頭 (15,1) 朝西。

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func TestSokalBoatLandsAtTheCityDock(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(4)
	if !ok {
		t.Fatal("ECL4 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 21, 0x9918)
	if err != nil {
		t.Fatal(err)
	}
	hero := poolsave.Character{
		Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", MaxHP: 20, CurrentHP: 20,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1,
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 4
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	application.state.Party = []poolsave.Character{hero}
	application.state.CharacterLibrary = []poolsave.Character{hero}
	application.spawn = gamepack.Spawn{
		Map: gamepack.MapKey{Archive: 4, BlockID: 21}, X: 8, Y: 14, Facing: 2,
	}
	// `9918` 的 `COMPARE @6DD5 0` 為真就跳過整段：要塞清完那一支才會問船。
	application.eventMachine.Memory[0x6DD5] = 1

	result, err := session.RunUntilEvent(4096, nil, true)
	if err != nil {
		t.Fatalf("跑到問船那一步：%v", err)
	}
	if err := application.consumeInitialSearch(result); err != nil {
		t.Fatalf("跑到問船那一步：%v", err)
	}
	if !hasMenuOption(application, "YES") {
		t.Fatalf("沒有出現問船的選單：pending=%v menu=%v 選項=%v 文字=%q",
			application.cellEventPending, application.cellWaitingMenu,
			application.cellMenuOptions, application.eventText)
	}
	if err := selectMenuOption(t, application, "YES"); err != nil {
		t.Fatal(err)
	}
	// 按到換區為止就停：這個治具是直接從 ECL4/21 起跑的合成 session，導覽旗標
	// 沒有立起來，多按幾下城區入口就會把羅夫導覽叫出來（真正玩到這裡時早就跑完了），
	// 隊伍跟著被導覽帶走，落點斷言就白做了。
	for guard := 0; guard < 12; guard++ {
		if application.eclArchive != 4 && application.eventSession.CurrentBlockID() != 21 {
			break
		}
		if err := press(application, ebiten.KeyEnter); err != nil {
			t.Fatal(err)
		}
	}
	if application.eclArchive != 3 || application.eventSession.CurrentBlockID() != 0 {
		t.Fatalf("搭船之後在 ECL%d/%d，原版是 ECL3/0",
			application.eclArchive, application.eventSession.CurrentBlockID())
	}
	want := gamepack.MapKey{Archive: 3, BlockID: 0}
	if application.spawn.Map != want || application.spawn.X != 15 || application.spawn.Y != 1 ||
		application.spawn.Facing != 3 {
		t.Fatalf("落點是 GEO%d/%d (%d,%d) 朝 %d，原版是 GEO3/0 (15,1) 朝 3",
			application.spawn.Map.Archive, application.spawn.Map.BlockID,
			application.spawn.X, application.spawn.Y, application.spawn.Facing)
	}
}
