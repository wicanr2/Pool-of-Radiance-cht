package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 自訂規則「委任折算經驗值」（spec 140，issue #28）。貧民窟的獎賞照 spec 137 的表是
// 250 金＋50 白金＋1 珠寶：250 ＋ 50×5 ＋ 100（珠寶基準值）＝ 600。
func TestCommissionExperienceValueUsesGoldEquivalents(t *testing.T) {
	if got := commissionExperienceValue([7]uint32{3: 250, 4: 50, 6: 1}); got != 600 {
		t.Fatalf("slums reward valued at %d XP, want 600", got)
	}
	// 200 銅＝20 銀＝2 琥珀金＝1 金；寶石基準 10。
	if got := commissionExperienceValue([7]uint32{200, 20, 2, 0, 0, 1, 0}); got != 13 {
		t.Fatalf("mixed coins valued at %d XP, want 13", got)
	}
}

// 開著：交件之後每一位隊員另外各得 600 XP；關著：只有原版戰後結算把公款折出來的那一份
// （250 ＋ 50×5 ＋ 1×2200 ＝ 2700，一個人分，spec 148）。兩條都走真的職員腳本。
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
			want += 600
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
