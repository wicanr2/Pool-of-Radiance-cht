package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 關著：只有原版戰後結算把公款折出來的那一份（250 ＋ 50×5 ＋ 1×2200 ＝ 2700，
// 一個人分，spec 148）；開著：同一份再發一次，共 5400。兩條都走真的職員腳本。
func TestCommissionExperienceHouseRuleAtTheCityHall(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		application, session, _ := slumsCommissionApp(t, 25)
		application.state.HouseRules.CommissionExperience = enabled
		before := application.state.Party[0].Experience
		handInAtCityHall(t, application, session)
		if !application.treasureActive {
			t.Fatalf("enabled=%t: the reward service did not open", enabled)
		}
		gained := application.state.Party[0].Experience - before
		want := uint32(2700)
		if enabled {
			want *= 2
		}
		if gained != want {
			t.Fatalf("enabled=%t: the hand-in granted %d XP, want %d", enabled, gained, want)
		}
	}
}

// 隊伍選單的 H 是開關，跟著存檔走。
func TestHouseRuleToggleOnThePartyMenu(t *testing.T) {
	application := &app{mode: modeMenu, language: languageEnglish}
	application.state = poolsave.NewState()
	if err := press(application, ebiten.KeyH); err != nil {
		t.Fatal(err)
	}
	if !application.state.HouseRules.CommissionExperience {
		t.Fatal("H did not switch the house rule on")
	}
	if err := press(application, ebiten.KeyH); err != nil {
		t.Fatal(err)
	}
	if application.state.HouseRules.CommissionExperience {
		t.Fatal("H did not switch the house rule back off")
	}
}
