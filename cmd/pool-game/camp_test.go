package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

func campApp(t *testing.T) *app {
	t.Helper()
	radix, err := gamepack.ReadDOSTimeRadix(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	application := &app{language: languageEnglish}
	application.restDuration = gamepack.NewRestDuration(radix)
	application.restField = gamepack.RestFieldMinutes
	application.state.Party = []poolsave.Character{
		{Name: "A", MaxHP: 20, CurrentHP: 5},
		{Name: "B", MaxHP: 12, CurrentHP: 12},
	}
	return application
}

// 休息一天：每人回一點生命力，滿血的不會超過上限。
// 說明書 p.29 寫的是「每休息二十四小時各隊員可恢復一點 HP」。
func TestRestingADayHealsOnePointEach(t *testing.T) {
	application := campApp(t)
	for step := 0; step < 24; step++ {
		application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)
	}
	application.restParty()
	if got := application.state.Party[0].CurrentHP; got != 6 {
		t.Errorf("休息一天之後第一個人是 %d 點，應該是 6", got)
	}
	if got := application.state.Party[1].CurrentHP; got != 12 {
		t.Errorf("滿血的人變成 %d 點，不該超過上限", got)
	}
	if !strings.Contains(application.statusLine, "+1") {
		t.Errorf("狀態列是 %q，應該說回了 1 點", application.statusLine)
	}
}

// 負對照：時間不夠就什麼都不會回。少了它，「休息一天回一點」證不了因果。
func TestRestingTooBrieflyHealsNothing(t *testing.T) {
	application := campApp(t)
	for step := 0; step < 12; step++ {
		application.restDuration = application.restDuration.Increase(gamepack.RestFieldHours)
	}
	application.restParty()
	if got := application.state.Party[0].CurrentHP; got != 5 {
		t.Errorf("休息半天之後是 %d 點，不該回", got)
	}
}

// 選欄與增減照原版：分鐘一次五分，Y／H／M 換欄。
func TestCampKeysAdjustTheRestTime(t *testing.T) {
	application := campApp(t)
	application.restField = gamepack.RestFieldMinutes
	application.restDuration = application.restDuration.Increase(application.restField)
	if got := application.restDuration.Minutes(); got != 5 {
		t.Errorf("分鐘加一次是 %d 分，原版一次五分", got)
	}
	application.restField = gamepack.RestFieldDays
	application.restDuration = application.restDuration.Increase(application.restField)
	if got := application.restDuration.Days(); got != 1 {
		t.Errorf("天數加一次是 %d 天", got)
	}
	line := application.campRestTimeLine()
	if !strings.Contains(line, "01") || !strings.Contains(line, "05") {
		t.Errorf("時間那一列是 %q，應該看得到 1 天與 5 分", line)
	}
}
