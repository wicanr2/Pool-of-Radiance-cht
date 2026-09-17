package main

// 量牆（#22）：建好隊伍、休息記完兩發催眠、滿血，直接踩貧民窟的三場固定事件，
// 每一場記一行回合帳（`battleTally`）。這幾條是尺不是門檻：輸了只記不紅——
// 它們紅的那一天是駕駛改到能贏的那一天，到時再把「贏」寫成斷言。結果表在
// `docs/playtest/mainline-end-to-end.md` 補七。

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// slumsWallFight 把一級隊伍送到貧民窟地形 terrain 的固定事件上打一場，
// 開打那一刻把盤面印進 log（部署對原版的收據就是這幾張，spec 061）。
func slumsWallFight(t *testing.T, seed int64, terrain int) *mainlineDriver {
	t.Helper()
	return slumsWallFightWith(t, seed, terrain, func(state *tacticalState) {
		t.Logf("terrain %d seed %d board at round 1:", terrain, seed)
		for _, line := range boardLines(state, 1) {
			t.Log(line)
		}
	})
}

// slumsWallFightWith 同上，多一個每場開打時的回呼（印盤面用）。
func slumsWallFightWith(t *testing.T, seed int64, terrain int, onBattleStart func(*tacticalState)) *mainlineDriver {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application.roller = diceRoller{random: rand.New(rand.NewSource(seed))}
	application.eclSeed = 1
	text := &scriptedTextKeys{scriptedKeys: scriptedKeys{}}
	step := func(key ebiten.Key, chars ...rune) {
		t.Helper()
		application.keys = text
		text.scriptedKeys = scriptedKeys{key: true}
		text.chars = chars
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
		runAfterTick(application)
	}
	idle := func() {
		t.Helper()
		application.keys = text
		text.scriptedKeys, text.chars = scriptedKeys{}, nil
		if err := application.Update(); err != nil {
			t.Fatal(err)
		}
		runAfterTick(application)
	}
	driver := buildManualParty(t, application, step, idle, false)
	driver.hurt, driver.rest = partyHurt, driver.restUntilHealed
	driver.tolerateDefeat = true
	driver.onBattleStart = onBattleStart
	driver.stepOut("into the slums", 3)
	driver.restIfHurt()
	target := func(x, y int) bool { return driver.terrainCode(x, y) == terrain }
	// 路上避開三場固定事件，鎖住的門照探索器撬；隨機遭遇照打，打完受傷就睡。
	safe := func(x, y int) bool { c := driver.terrainCode(x, y); return c != 9 && c != 13 && c != 15 }
	if !driver.walkAllowing("slums wall", target, safe, true) && !application.gameOver {
		t.Fatalf("cannot reach terrain %d", terrain)
	}
	driver.settle()
	t.Logf("slums terrain %d seed %d: %s; stalemate endings so far %d", terrain, seed, driver.tally.line(), tacticalStalemateEndings)
	return driver
}

// 獸人的家：20 獸人＋4 首領（`ecl2/20` 地形 9）。
func TestWallSlumsOrcHome(t *testing.T) {
	for _, seed := range []int64{136, 137} {
		slumsWallFight(t, seed, 9)
	}
}

// 衛兵攔截：30 獸人＋4 首領（地形 13）。
func TestWallSlumsGuards(t *testing.T) {
	for _, seed := range []int64{136, 137} {
		slumsWallFight(t, seed, 13)
	}
}

// 驚動衛兵：12 哥布林衛兵＋21 哥布林頭目（地形 15）。
func TestWallSlumsAlarm(t *testing.T) {
	for _, seed := range []int64{136, 137} {
		slumsWallFight(t, seed, 15)
	}
}
