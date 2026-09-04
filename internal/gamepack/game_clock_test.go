package gamepack_test

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

func timeRadix(t *testing.T) gamepack.TimeRadix {
	t.Helper()
	radix, err := gamepack.ReadDOSTimeRadix(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	return radix
}

// 七個上限逐個釘住。它們就是時間的意義：分兩位、時 24、日 30、月 12。
func TestTimeRadixMatchesTheOriginal(t *testing.T) {
	radix := timeRadix(t)
	want := gamepack.TimeRadix{10, 10, 6, 24, 30, 12, 256}
	if radix != want {
		t.Fatalf("進位上限是 %v，原版是 %v", radix, want)
	}
	// 一小時 12 刻、二十四小時 288 刻——兩個常數要自洽，一刻才會是五分鐘。
	if gamepack.RestTicksPerHeal != gamepack.RestTicksPerHour*24 {
		t.Errorf("288 刻不等於 24 小時：每小時 %d 刻", gamepack.RestTicksPerHour)
	}
	if gamepack.RestMinutesPerTick != gamepack.RestMinuteStep {
		t.Errorf("一刻 %d 分，但分鐘的級距是 %d——原版兩者相同",
			gamepack.RestMinutesPerTick, gamepack.RestMinuteStep)
	}
}

// 分鐘一次加五，加到 60 就進位成一小時。
func TestRestDurationCarriesMinutesIntoHours(t *testing.T) {
	duration := gamepack.NewRestDuration(timeRadix(t))
	for step := 1; step <= 12; step++ {
		duration = duration.Increase(gamepack.RestFieldMinutes)
		want := step * gamepack.RestMinuteStep
		if want < 60 {
			if duration.Minutes() != want || duration.Hours() != 0 {
				t.Fatalf("加了 %d 次之後是 %d 時 %d 分，應該是 0 時 %d 分",
					step, duration.Hours(), duration.Minutes(), want)
			}
			continue
		}
		if duration.Hours() != 1 || duration.Minutes() != 0 {
			t.Fatalf("加滿六十分之後是 %d 時 %d 分，應該是 1 時 0 分",
				duration.Hours(), duration.Minutes())
		}
	}
}

// 小時加到 24 進位成一天。
func TestRestDurationCarriesHoursIntoDays(t *testing.T) {
	duration := gamepack.NewRestDuration(timeRadix(t))
	for step := 0; step < 24; step++ {
		duration = duration.Increase(gamepack.RestFieldHours)
	}
	if duration.Days() != 1 || duration.Hours() != 0 {
		t.Fatalf("加了二十四小時之後是 %d 天 %d 時", duration.Days(), duration.Hours())
	}
}

// 天數加到 30 在原版是進位成「一個月」，但這是一段長度不是日期，
// 所以 entry 6 立刻把月折回天——畫面上會看到 30 天，不是 0 天。
func TestRestDurationFoldsMonthsBackIntoDays(t *testing.T) {
	duration := gamepack.NewRestDuration(timeRadix(t))
	for step := 0; step < 30; step++ {
		duration = duration.Increase(gamepack.RestFieldDays)
	}
	if duration.Days() != 30 {
		t.Fatalf("加了三十天之後是 %d 天，月應該被折回天", duration.Days())
	}
	for step := 0; step < 200; step++ {
		duration = duration.Increase(gamepack.RestFieldDays)
	}
	if duration.Days() != gamepack.RestMaxDays {
		t.Fatalf("天數是 %d，上限是 %d", duration.Days(), gamepack.RestMaxDays)
	}
}

// 減到零就停住，不會繞回去變成很大的數。
func TestRestDurationWillNotGoBelowZero(t *testing.T) {
	duration := gamepack.NewRestDuration(timeRadix(t))
	for _, field := range []gamepack.RestField{
		gamepack.RestFieldMinutes, gamepack.RestFieldHours, gamepack.RestFieldDays,
	} {
		duration = duration.Decrease(field)
		if !duration.IsZero() {
			t.Fatalf("從零往下減 %v 之後變成 %d 天 %d 時 %d 分",
				field, duration.Days(), duration.Hours(), duration.Minutes())
		}
	}
	// 正對照：有值的時候減得動，而且會從高位借。
	duration = duration.Increase(gamepack.RestFieldHours)
	duration = duration.Decrease(gamepack.RestFieldMinutes)
	if duration.Hours() != 0 || duration.Minutes() != 55 {
		t.Fatalf("一小時減五分變成 %d 時 %d 分，應該是 0 時 55 分",
			duration.Hours(), duration.Minutes())
	}
}

// 欄位左右循環在 2..4 之間。
func TestRestFieldsCycle(t *testing.T) {
	if got := gamepack.RestFieldDays.NextField(); got != gamepack.RestFieldMinutes {
		t.Errorf("天的下一欄是 %v", got)
	}
	if got := gamepack.RestFieldMinutes.PreviousField(); got != gamepack.RestFieldDays {
		t.Errorf("分的上一欄是 %v", got)
	}
	if got := gamepack.RestFieldHours.NextField(); got != gamepack.RestFieldDays {
		t.Errorf("時的下一欄是 %v", got)
	}
}

// 說明書 p.29：「每休息二十四小時各隊員可恢復一點 HP」。
// 兩份來源互相獨立：一邊是 overlay-20 `083Ah` 的 120h 刻，一邊是中文說明書。
func TestRestHealingMatchesTheManual(t *testing.T) {
	radix := timeRadix(t)
	oneDay := gamepack.NewRestDuration(radix)
	for step := 0; step < 24; step++ {
		oneDay = oneDay.Increase(gamepack.RestFieldHours)
	}
	if got := gamepack.RestHealing(oneDay.TotalTicks()); got != 1 {
		t.Fatalf("休息一天回 %d 點生命力，說明書寫的是 1 點", got)
	}
	// 差一點就是零：二十三小時五十五分回不了。
	almost := oneDay.Decrease(gamepack.RestFieldMinutes)
	if got := gamepack.RestHealing(almost.TotalTicks()); got != 0 {
		t.Fatalf("休息 %d 時 %d 分回了 %d 點，不該回",
			almost.Hours(), almost.Minutes(), got)
	}
	threeDays := oneDay
	for step := 0; step < 2; step++ {
		threeDays = threeDays.Increase(gamepack.RestFieldDays)
	}
	if got := gamepack.RestHealing(threeDays.TotalTicks()); got != 3 {
		t.Fatalf("休息三天回 %d 點", got)
	}
}
