package main

// 作弊選單（spec 141）：開關從按鍵來、開過就記在存檔、鎖 HP 與一擊斃命各自有正反對照、
// F1 說明頁列出作弊鍵。

import (
	"errors"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// probeCheatAtNorris 為真時，house rule 探針只在諾里斯那一場開作弊（#43）。
var probeCheatAtNorris = false

// setCheats 用按鍵把兩個作弊開關切到指定的狀態（開選單、按 L／O、關選單）。
func (d *mainlineDriver) setCheats(lockHP, oneHitKill bool) {
	d.t.Helper()
	a := d.a
	d.step(ebiten.KeyF6)
	if !a.cheatOpen {
		d.fatalf("setCheats: F6 did not open the cheat menu at %+v (screen %s)", a.spawn, a.screenName())
	}
	if a.state.Cheats.LockHP != lockHP {
		d.step(ebiten.KeyL)
	}
	if a.state.Cheats.OneHitKill != oneHitKill {
		d.step(ebiten.KeyO)
	}
	d.step(ebiten.KeyEscape)
	if a.cheatOpen || a.state.Cheats.LockHP != lockHP || a.state.Cheats.OneHitKill != oneHitKill {
		d.fatalf("setCheats: menu open=%t cheats=%+v, want lock=%t kill=%t", a.cheatOpen, a.state.Cheats, lockHP, oneHitKill)
	}
	d.note("setCheats: lock HP %t, one-hit kill %t at %+v", lockHP, oneHitKill, a.spawn)
}

// TestMainlineProbeHouseRuleCheatAtNorris 是 #43 的收據：house rule 探針（seed 142），只在諾里斯
// 那一場開作弊，打完關掉，之後照正常強度往下跑，停在哪就是下一道牆（會紅，失敗訊息指出段落）。
//
// 2026-09-17：142 開作弊打贏諾里斯（`4A24 = FF`、槽 0 `FE`），關掉之後回程在古托井地面 (13,4)
// D 倒地、睡不成（12／12）停下。seed 136..150 的表在 playtest 補十四：走到諾里斯的 6 個全部打贏，
// 之後三個停在古托井地面、三個死在索寇要塞中庭的巡邏；另外 9 個走不到諾里斯。
func TestMainlineProbeHouseRuleCheatAtNorris(t *testing.T) {
	probeCheatAtNorris = true
	defer func() { probeCheatAtNorris = false }()
	runMainlineProbe(t, true, 142)
}

// toggleCheats 用按鍵打開作弊選單、切換指定的開關、關掉選單。
func toggleCheats(t *testing.T, step func(ebiten.Key), a *app, keys ...ebiten.Key) {
	t.Helper()
	step(ebiten.KeyF6)
	if !a.cheatOpen || a.screenName() != "cheat-menu" {
		t.Fatalf("F6 沒有開作弊選單：open=%t screen=%s", a.cheatOpen, a.screenName())
	}
	for _, key := range keys {
		step(key)
	}
	step(ebiten.KeyEscape)
	if a.cheatOpen {
		t.Fatal("ESC 沒有關掉作弊選單")
	}
}

func TestCheatMenuTogglesFromKeysAndRemembersItWasUsed(t *testing.T) {
	a := bootCityParty(t, dosZIPForTests)
	step := func(key ebiten.Key) {
		t.Helper()
		if err := press(a, key); err != nil {
			t.Fatal(err)
		}
	}
	if a.state.Cheats.LockHP || a.state.Cheats.OneHitKill || a.state.CheatsUsed || a.cheatMark() != "" {
		t.Fatalf("新遊戲的作弊就開著：%+v used=%t", a.state.Cheats, a.state.CheatsUsed)
	}
	step(ebiten.KeyF6)
	spawn := a.spawn
	step(ebiten.KeyArrowRight)
	step(ebiten.KeyArrowUp)
	if a.spawn != spawn {
		t.Fatalf("作弊選單開著時方向鍵還在走路：%+v → %+v", spawn, a.spawn)
	}
	step(ebiten.KeyL)
	if !a.state.Cheats.LockHP || !a.state.CheatsUsed || !strings.Contains(a.statusLine, a.text(msgCheatLockHPName)) {
		t.Fatalf("L 沒有打開鎖 HP：%+v used=%t status=%q", a.state.Cheats, a.state.CheatsUsed, a.statusLine)
	}
	step(ebiten.KeyO)
	step(ebiten.KeyL)
	step(ebiten.KeyO)
	if a.state.Cheats.LockHP || a.state.Cheats.OneHitKill {
		t.Fatalf("再按一次沒有關掉：%+v", a.state.Cheats)
	}
	if !a.state.CheatsUsed || a.cheatMark() != a.text(msgCheatUsedMark) {
		t.Fatalf("關掉之後 CheatsUsed=%t 標示 %q，開過就要一直標", a.state.CheatsUsed, a.cheatMark())
	}
	step(ebiten.KeyF6)
	if a.cheatOpen {
		t.Fatal("F6 沒有關掉作弊選單")
	}
	step(ebiten.KeyF6)
	step(ebiten.KeyO)
	step(ebiten.KeyEscape)

	// 存檔、讀檔之後，開關與「開過」都還在。
	statePath := filepath.Join(t.TempDir(), "state.json")
	a.saveState = func(state poolsave.State) error { return poolsave.WriteAtomic(statePath, state) }
	if err := press(a, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10 存檔回傳 %v", err)
	}
	restored, err := newApp(dosZIPForTests, statePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyL} {
		if err := press(restored, key); err != nil {
			t.Fatal(err)
		}
	}
	if restored.mode != modeAdventure || !restored.state.CheatsUsed || !restored.state.Cheats.OneHitKill || restored.state.Cheats.LockHP {
		t.Fatalf("讀檔之後 mode=%d cheats=%+v used=%t", restored.mode, restored.state.Cheats, restored.state.CheatsUsed)
	}
}

// 鎖 HP：同一場（貧民窟衛兵攔截，seed 136）不開全滅、開了打贏。負對照先跑。
func TestCheatLockHPTurnsTheSlumsGuardsWipeIntoAWin(t *testing.T) {
	const seed, guards = 136, 13
	plain := slumsWallFightWith(t, seed, guards, nil)
	if !plain.a.gameOver {
		t.Skipf("seed %d 沒開作弊也打贏了（%s），當不了負對照", seed, plain.tally.line())
	}
	locked := slumsWallFightPrepared(t, seed, guards, nil, func(d *mainlineDriver) {
		toggleCheats(t, d.step, d.a, ebiten.KeyL)
	})
	a := locked.a
	if a.gameOver || a.tactical != nil || a.combatActive {
		t.Fatalf("開了鎖 HP 還是沒打贏：over=%t tactical=%t %s", a.gameOver, a.tactical != nil, locked.tally.line())
	}
	if got := a.eventMachine.Memory[0x6DC7]; got != 0 {
		t.Fatalf("開了鎖 HP 結果碼 6DC7=%02X，打贏是 0", got)
	}
	if a.cheatRestores.inCombat == 0 || !a.state.CheatsUsed {
		t.Fatalf("鎖 HP 一次都沒寫回（%+v）或沒記 CheatsUsed", a.cheatRestores)
	}
	t.Logf("cheat lock HP: in combat %d, out of combat %d, revived %d", a.cheatRestores.inCombat,
		a.cheatRestores.outOfCombat, a.cheatRestores.revived)
}

// foeHitObserver 每個 tick 看敵人的 HP：少了多少、少完剩幾點。
type foeHitObserver struct {
	last     []int
	partial  int // 少了但沒歸零
	killed   int // 少了而且歸零
	partyCut int // 隊員被打掉 HP、沒歸零的次數（一擊斃命不該蓋敵人造成的傷害）
	partyHP  []int
}

func (o *foeHitObserver) observe(a *app) {
	state := a.tactical
	if state == nil {
		o.last, o.partyHP = nil, nil
		return
	}
	if len(o.last) != len(state.HitPoints) {
		o.last = append([]int(nil), state.HitPoints...)
		o.partyHP = append([]int(nil), state.HitPoints...)
		return
	}
	for index := 1; index < len(state.HitPoints) && index < len(state.PartySlot); index++ {
		before, now := o.last[index], state.HitPoints[index]
		if now >= before {
			continue
		}
		if state.PartySlot[index] >= 0 {
			if now > 0 {
				o.partyCut++
			}
			continue
		}
		if now == 0 {
			o.killed++
		} else {
			o.partial++
		}
	}
	o.last = append(o.last[:0], state.HitPoints...)
}

// 一擊斃命（近戰）：開著時敵人每一次掉 HP 都是歸零；關著時同一個 seed 有打到但沒打死的。
// 用獸人的家（地形 9）：獸人 1 HD，隊伍打得到也打得死，差別看得出來。
func TestCheatOneHitKillMakesEveryPartyHitLethal(t *testing.T) {
	const seed, orcs = 136, 9
	run := func(on bool) *foeHitObserver {
		observer := &foeHitObserver{}
		previous := afterTick
		afterTick = observer.observe
		defer func() { afterTick = previous }()
		prepare := func(d *mainlineDriver) {}
		if on {
			prepare = func(d *mainlineDriver) { toggleCheats(t, d.step, d.a, ebiten.KeyO) }
		}
		slumsWallFightPrepared(t, seed, orcs, nil, prepare)
		return observer
	}
	plain := run(false)
	if plain.partial == 0 {
		t.Skipf("關著時沒有一下是打到沒打死（歸零 %d），當不了負對照", plain.killed)
	}
	cheated := run(true)
	t.Logf("off: partial %d killed %d; on: partial %d killed %d party cut %d", plain.partial, plain.killed,
		cheated.partial, cheated.killed, cheated.partyCut)
	if cheated.killed == 0 || cheated.partial != 0 {
		t.Fatalf("一擊斃命開著：打到沒打死 %d 次、打死 %d 次，要全部歸零", cheated.partial, cheated.killed)
	}
}

// 一擊斃命（法術、敵方造成的傷害）：`applySpellDamage` 與 `cheatDamage` 直接驗。手動建的隊伍
// 沒有記傷害法術（只有催眠術），按鍵走不到施法傷害那一支，所以這一條直接呼叫。
func TestCheatOneHitKillCoversPartySpellsButNotFoes(t *testing.T) {
	a := &app{}
	a.state.Cheats.OneHitKill = true
	state := &tacticalState{
		PartySlot: []int{-1, 0, -1, -1},
		HitPoints: []int{0, 10, 30, 30},
		Roster:    make([]combat.CombatantCell, 4),
		Scores:    make([]uint8, 4),
		States:    make([]uint8, 4),
		Footprint: make([]uint8, 4),
		Mover:     1,
	}
	if got := a.cheatDamage(state, 1, 2, 4); got != 30 {
		t.Fatalf("隊員造成 4 點，一擊斃命該改成 30，得到 %d", got)
	}
	if got := a.cheatDamage(state, 1, 2, 0); got != 0 {
		t.Fatalf("豁免成功（傷害 0）不該歸零，得到 %d", got)
	}
	if got := a.cheatDamage(state, 2, 1, 4); got != 4 {
		t.Fatalf("敵人造成的傷害不該改，得到 %d", got)
	}
	a.applySpellDamage(state, 3, 5)
	if state.HitPoints[3] != 0 {
		t.Fatalf("隊員的法術 5 點，一擊斃命後目標 HP=%d，該是 0", state.HitPoints[3])
	}
	a.state.Cheats.OneHitKill = false
	state.HitPoints[2] = 30
	a.applySpellDamage(state, 2, 5)
	if state.HitPoints[2] != 25 {
		t.Fatalf("關掉之後法術 5 點，目標 HP=%d，該是 25", state.HitPoints[2])
	}
}

// F1 說明頁：第 1 頁有作弊鍵那一行，中英各一；兩頁都不超出框。
func TestHelpPagesListTheCheatKeyAndFitTheBox(t *testing.T) {
	for _, language := range []language{languageTraditionalChinese, languageEnglish} {
		a := &app{language: language}
		for page := 0; page < helpPages; page++ {
			a.helpPage = page
			var lines []string
			bottom := 0
			drawnText = func(value string, x, y int) {
				lines = append(lines, value)
				if y > bottom {
					bottom = y
				}
			}
			screen := ebiten.NewImage(logicalWidth, logicalHeight)
			drawHelp(screen, a, color.Black, color.White, color.White, []string{"A", "B", "C", "D", "E", "F", "G"})
			drawnText = nil
			if bottom > helpBottom-4 {
				t.Fatalf("language %d page %d 最後一行基線 %d，超出框 %d", language, page+1, bottom, helpBottom)
			}
			if page == 0 && !strings.Contains(strings.Join(lines, "\n"), "F6") {
				t.Fatalf("language %d 第 1 頁沒有 F6：%q", language, lines)
			}
		}
	}
}
