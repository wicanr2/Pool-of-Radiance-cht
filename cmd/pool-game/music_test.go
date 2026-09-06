package main

import (
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/music"
)

// 情境跟著畫面狀態走：戰術盤面＝戰鬥，地圖＝冒險，其餘＝標題。
func TestMusicCueFollowsTheScreen(t *testing.T) {
	application := &app{mode: modeTitle}
	if got := application.musicCue(); got != music.CueTitle {
		t.Errorf("標題畫面是 %q，預期 %q", got, music.CueTitle)
	}
	application.mode = modeMenu
	if got := application.musicCue(); got != music.CueTitle {
		t.Errorf("隊伍管理是 %q，預期 %q（還沒進遊戲）", got, music.CueTitle)
	}
	application.mode = modeAdventure
	if got := application.musicCue(); got != music.CueAdventure {
		t.Errorf("冒險畫面是 %q，預期 %q", got, music.CueAdventure)
	}
	application.tactical = &tacticalState{Roster: []combat.CombatantCell{{}}}
	application.tacticalPreview = true
	if got := application.musicCue(); got != music.CueCombat {
		t.Errorf("戰術盤面是 %q，預期 %q", got, music.CueCombat)
	}
	// 關掉盤面之後 a.tactical 還留著（那是遊戲的既有行為，見 spec 127 的註解），
	// 所以情境要看 tacticalPreview，不能只看 a.tactical 是不是 nil。
	application.tacticalPreview = false
	if got := application.musicCue(); got != music.CueAdventure {
		t.Errorf("關掉盤面之後是 %q，預期回到 %q", got, music.CueAdventure)
	}
}

// 沒有音訊資產時整條路徑要安靜地走完，不能 panic。
func TestUpdateMusicIsSafeWithoutAssets(t *testing.T) {
	application := &app{mode: modeAdventure}
	application.updateMusic()
	if application.musicPlayer != nil {
		t.Fatal("沒有資產卻有 player")
	}
}
