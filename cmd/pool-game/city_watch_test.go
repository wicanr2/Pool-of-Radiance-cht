package main

// 城區的城衛隊與晚上鎖門（#22、#26；spec 102〈晚上鎖門與城衛隊〉）。
//
// 城衛隊那一場 38 隻（`ecl3/0 ADDFh..AE08h`：LEVEL 3 MU×2、6TH LVL FIGHTER×12、
// AIDES×12、NOMAD×12）**不是到點出兵**。進到那一場的五條路都要玩家選：
//
//	99D8h  晚上的鎖門  "DO YOU WANT TO BREAK IN?"   YES → AD82h（NO 安全）
//	9AE6h  休息被驅趕  "ROUSTED BY THE CITY WATCH"  STAY → ADDFh（GO 安全）
//	9E9Ah  神殿衛兵    LEAVE／FORCE YOUR WAY PAST   FORCE → AD82h
//	A828h  酒館鬥毆後  STAY／RUN                    STAY → AD82h
//	ADC3h  衛兵趕到    STAY／RUN                    STAY → ADDFh（RUN → AEF4h 隨機搬走）
//
// 鎖門（YES）與兩個 STAY／RUN（STAY）的第 0 項就是開打的那個，所以「不認得就
// 按 ENTER」的駕駛會一路打進去——house rule 探針 seed 143 在市政廳門口 (3,4)
// 全滅就是這樣（22 點）。驅趕與神殿的第 0 項是安全的，但輪流試選項的探索器
// 照樣會試到。
//
// 晚上是 `ecl3/0 9920h` 的 `49C9 >= 14`（`6E7D = 8`，否則 11）。比較的方向對過原版：
// 開局時鐘 0、在城區走一步 `6E7D = 11`（`docs/audit/dosgolem-city-night-flag.json`）。
// 鎖的是面向特定牆型的門：地形 26 牆型 11（市政廳，(4,3) 朝南、(3,4) 朝東、
// (5,4) 朝西）、地形 0 牆型 9、地形 4 牆型 7。
//
// 都從 `Update()` 送鍵；產品碼沒有為這一條改過。

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// cityWatchAnswer 是城衛隊那幾個問句的安全答案；認不得的問句回 false，
// 由呼叫端照原本的規則答。
func cityWatchAnswer(text string, options []string) (string, bool) {
	has := func(label string) bool {
		for _, option := range options {
			if strings.EqualFold(option, label) {
				return true
			}
		}
		return false
	}
	upper := strings.ToUpper(text)
	switch {
	case strings.Contains(upper, "DO YOU WANT TO BREAK IN") && has("NO"):
		return "NO", true
	case strings.Contains(upper, "ROUSTED BY THE CITY WATCH") && has("GO"):
		return "GO", true
	case strings.Contains(upper, "CITY WATCH") && has("RUN"):
		return "RUN", true
	case has("FORCE YOUR WAY PAST") && has("LEAVE"):
		return "LEAVE", true
	}
	return "", false
}

// answerCityWatch 在等選單而且是城衛隊的問句時替玩家答，回 true 表示答了。
func (d *mainlineDriver) answerCityWatch() bool {
	a := d.a
	if !a.cellWaitingMenu || len(a.cellMenuOptions) == 0 {
		return false
	}
	answer, ok := cityWatchAnswer(a.eventText, a.cellMenuOptions)
	if !ok {
		return false
	}
	for index, option := range a.cellMenuOptions {
		if strings.EqualFold(option, answer) {
			answer = a.cellMenuOptions[index]
		}
	}
	d.note("city watch: %q → %s at (%d,%d) hour %d", firstLine(a.eventText), answer,
		a.spawn.X, a.spawn.Y, a.gameTime[gamepack.TimeDigitHour])
	if err := selectMenuOption(d.t, a, answer); err != nil {
		d.fatalf("city watch menu: %v", err)
	}
	return true
}

// nightInTheCity 是 `ecl3/0 9920h` 的判斷：城區腳本把 `49C9 >= 14` 當晚上。
func nightInTheCity(clock gamepack.GameTime) bool {
	return clock[gamepack.TimeDigitHour] >= 14
}

// sleepUntilHour 睡到指定的時刻：這一區會打擾就先去旅店開房間（#38），然後
// 紮營只排小時那一欄。回 false 表示睡不成（沒白金、走不到旅店），呼叫端自己決定。
func (d *mainlineDriver) sleepUntilHour(target int) bool {
	a := d.a
	d.t.Helper()
	if err := a.runRestEntry(); err == nil && a.restInterruption().Period != 0 {
		if !d.restAtTheInn() {
			return false
		}
	}
	hours := (target - a.gameTime[gamepack.TimeDigitHour] + 24) % 24
	if hours == 0 {
		return true
	}
	before := a.gameTime
	d.step(ebiten.KeyE)
	if !a.campOpen {
		d.note("sleepUntilHour: E did not open the camp at %+v: %q", a.spawn, a.statusLine)
		return false
	}
	d.step(ebiten.KeyR)
	d.clearRestDuration()
	d.step(ebiten.KeyH)
	for hour := 0; hour < hours; hour++ {
		d.step(ebiten.KeyI)
	}
	d.step(ebiten.KeyR)
	for guard := 0; guard < 8 && (a.campOpen || a.campFromProgram); guard++ {
		d.step(ebiten.KeyEscape)
	}
	d.settle()
	d.note("sleepUntilHour: %d → %d at (%d,%d)", before[gamepack.TimeDigitHour],
		a.gameTime[gamepack.TimeDigitHour], a.spawn.X, a.spawn.Y)
	return a.gameTime[gamepack.TimeDigitHour] == target
}

// clearRestDuration 把紮營排時間那一列三欄都歸零。紮營畫面會記得上一次排的
// 時間，而 `D` 只減目前那一欄——只在天數欄連按 `D`，上一次留下的小時數還在，
// 接著按 `I` 會進位成多一天（23 小時再加一小時就是一天零小時）。
func (d *mainlineDriver) clearRestDuration() {
	a := d.a
	for _, field := range []ebiten.Key{ebiten.KeyY, ebiten.KeyH, ebiten.KeyM} {
		d.step(field)
		for guard := 0; guard < 128 && !a.restDuration.IsZero(); guard++ {
			before := a.restDuration
			d.step(ebiten.KeyD)
			if a.restDuration == before {
				break
			}
		}
	}
	if !a.restDuration.IsZero() {
		d.fatalf("could not clear the rest duration: %v", a.restDuration)
	}
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return text[:index]
	}
	return text
}

// cityHallAt 讓城區的隊伍睡到 hour、走到市政廳門口 (3,4) 朝東踏一步，停在
// 踏完的那一刻（還沒答任何問句）。
func cityHallAt(t *testing.T, hour int) *mainlineDriver {
	t.Helper()
	d := innDriver(t, 3)
	a := d.a
	if !d.sleepUntilHour(hour) {
		t.Fatalf("睡不到 %d 點：現在 %d 點、位置 %+v", hour, a.gameTime[gamepack.TimeDigitHour], a.spawn)
	}
	street := func(x, y int) bool { return d.terrain(x, y) == 0 }
	door := func(x, y int) bool { return x == 3 && y == 4 }
	if !d.walkAllowing("City Hall door (3,4)", door, street, false) {
		t.Fatalf("走不到市政廳門口，停在 %+v", a.spawn)
	}
	d.settle()
	d.face(1)
	d.step(ebiten.KeyArrowUp)
	for guard := 0; guard < 16 && a.cellEventPending && !a.cellWaitingMenu; guard++ {
		d.step(ebiten.KeyEnter)
	}
	// 門口的布告（`AB97h` "YOU ARE OUTSIDE THE CITY HALL"）是單選項，按過去。
	for guard := 0; guard < 16 && a.cellWaitingMenu && len(a.cellMenuOptions) == 1; guard++ {
		d.step(ebiten.KeyEnter)
		for inner := 0; inner < 16 && a.cellEventPending && !a.cellWaitingMenu; inner++ {
			d.step(ebiten.KeyEnter)
		}
	}
	return d
}

// 晚上市政廳的門是鎖的；答 NO 不會開打，隊伍還在城區。
func TestCityHallIsLockedAtNightAndNoKeepsThePeace(t *testing.T) {
	d := cityHallAt(t, 20)
	a := d.a
	if !strings.Contains(a.eventText, "THE DOOR IS LOCKED") || len(a.cellMenuOptions) != 2 {
		t.Fatalf("20 點在 (3,4) 朝東沒有問鎖門：選單 %v 文字 %q 位置 %+v", a.cellMenuOptions, a.eventText, a.spawn)
	}
	if !d.answerCityWatch() {
		t.Fatalf("駕駛認不得鎖門的問句：%v %q", a.cellMenuOptions, a.eventText)
	}
	d.settle()
	if a.tactical != nil || a.combatActive || a.gameOver {
		t.Fatalf("答 NO 之後還是開打了：tactical=%t combat=%t over=%t 文字 %q",
			a.tactical != nil, a.combatActive, a.gameOver, a.eventText)
	}
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 0 {
		t.Fatalf("答 NO 之後進了 ECL%d/%d，門該是鎖著的", a.eclArchive, a.eventSession.CurrentBlockID())
	}
}

// 負對照：白天同一格朝東就直接進市政廳（ECL3/8），沒有鎖門的問句。
func TestCityHallIsOpenInTheMorning(t *testing.T) {
	d := cityHallAt(t, 6)
	a := d.a
	d.settle("Exit")
	if strings.Contains(a.eventText, "THE DOOR IS LOCKED") {
		t.Fatalf("6 點也問鎖門：%q", a.eventText)
	}
	if a.eventSession.CurrentBlockID() != 8 {
		t.Fatalf("6 點在 (3,4) 朝東到了 ECL%d/%d，該進市政廳（block 8）",
			a.eclArchive, a.eventSession.CurrentBlockID())
	}
}

// 正對照：答 YES 會叫來城衛隊（`AD82h`），而那個問句答 RUN 不開打。
// 少了這一條，「NO 不開打」可能只是因為這一格根本不會開打。
func TestBreakingIntoCityHallCallsTheWatchAndRunAvoidsTheFight(t *testing.T) {
	d := cityHallAt(t, 20)
	a := d.a
	if !strings.Contains(a.eventText, "THE DOOR IS LOCKED") {
		t.Fatalf("20 點沒有問鎖門：%v %q", a.cellMenuOptions, a.eventText)
	}
	if err := selectMenuOption(t, a, "YES"); err != nil {
		t.Fatal(err)
	}
	for guard := 0; guard < 16 && a.cellEventPending && !a.cellWaitingMenu; guard++ {
		d.step(ebiten.KeyEnter)
	}
	if !strings.Contains(a.eventText, "THE CITY WATCH RESPONDS TO THE NOISE") {
		t.Fatalf("破門之後沒有叫來城衛隊：%v %q", a.cellMenuOptions, a.eventText)
	}
	if first := a.cellMenuOptions[0]; !strings.EqualFold(first, "STAY") {
		t.Fatalf("城衛隊的選單第 0 項是 %q，`ADC3h` 是 STAY", first)
	}
	if !d.answerCityWatch() {
		t.Fatalf("駕駛認不得城衛隊的問句：%v %q", a.cellMenuOptions, a.eventText)
	}
	d.settle()
	if a.tactical != nil || a.combatActive || a.gameOver {
		t.Fatalf("答 RUN 之後還是開打了：文字 %q", a.eventText)
	}
}
