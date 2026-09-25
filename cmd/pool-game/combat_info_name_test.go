package main

import (
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 資訊欄第一行是行動者自己記錄裡的名字，不分玩家或怪物（#62）。原版
// overlay-25 entry 5 拿傳進來的記錄逐行畫，overlay-13 `0C30h` 只看 `+10Dh`
// 就把目前的行動者傳進去。貧民窟第一場是 ORC，輪到牠時要印牠的名字，不是
// 「敵方」；輪到隊員時照舊印隊員名。
func TestCombatInfoNamesTheMonsterOnItsTurn(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := application.eclCatalog.Archive(2)
	if !ok {
		t.Fatal("ECL2 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 20, 0x9E5D)
	if err != nil {
		t.Fatal(err)
	}
	application.mode, application.introDone = modeAdventure, true
	application.eventSession, application.eventMachine = session, session.Machine()
	application.eclArchive = 2
	application.state.Party = []poolsave.Character{{
		Name: "HERO", RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
		AlignmentID: "lawful-good", MaxHP: 12, CurrentHP: 12,
	}}
	application.spawn = gamepack.Spawn{Map: gamepack.MapKey{Archive: 2, BlockID: 20}, X: 3, Y: 4, Facing: 2}
	result, err := session.RunUntilEvent(16, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := application.consumeInitialSearch(result); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	state := application.tactical
	if state == nil {
		t.Fatal("ENTER did not enter tactical combat")
	}
	firstLine := func(mover int) string {
		state.Mover = uint8(mover)
		var got string
		drawnText = func(value string, x, y int) {
			if y == combatInfoLine1 && got == "" {
				got = value
			}
		}
		defer func() { drawnText = nil }()
		drawCombatInfo(ebiten.NewImage(logicalWidth, logicalHeight), application, color.White, color.White)
		return got
	}
	foe := -1
	for index := 1; index < len(state.Friendly); index++ {
		if !state.Friendly[index] {
			foe = index
			break
		}
	}
	if foe < 0 {
		t.Fatal("no foe on the board")
	}
	monster, ok := application.stagedMonsterFor(foe, state.Friendly)
	if !ok {
		t.Fatalf("foe %d has no staged monster", foe)
	}
	want := strings.TrimSpace(application.monsterText.Translate(monster.Record.Name))
	if want == "" || want == application.text(msgCombatFoe) {
		t.Fatalf("staged monster name %q is not usable", want)
	}
	if got := firstLine(foe); got != want {
		t.Fatalf("foe %d info line 1 = %q, want the monster's name %q", foe, got, want)
	}
	if got := firstLine(1); got != "HERO" {
		t.Fatalf("party info line 1 = %q, want HERO", got)
	}
}
