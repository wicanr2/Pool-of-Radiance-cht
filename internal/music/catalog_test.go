package music

import "testing"

// 目錄與對應表要自洽：六首、編號在模組的 1..6 裡、每個情境都指到真的有的那一首。
func TestCatalogMatchesTheModule(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	if len(Catalog()) != SubsongCount {
		t.Fatalf("目錄有 %d 首，模組是 %d 首", len(Catalog()), SubsongCount)
	}
}

// 三個情境是 C64 版驅動的三支包裝（INIT／DUNGEON／COMBAT）。
// 少一個或多一個都代表有人在沒有證據的情況下加了情境。
func TestCuesMatchTheOriginalHookSites(t *testing.T) {
	want := []Cue{CueTitle, CueAdventure, CueCombat}
	got := Cues()
	if len(got) != len(want) {
		t.Fatalf("有 %d 個情境，C64 版的包裝是 %d 支", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("第 %d 個情境是 %q，預期 %q", index, got[index], want[index])
		}
	}
}

// 標題要用會循環的那一首——標題畫面停多久由玩家決定，放完就沒了不行。
func TestTitleUsesALoopingTrack(t *testing.T) {
	track, found := TrackFor(CueTitle)
	if !found {
		t.Fatal("標題沒有對到曲子")
	}
	if !track.Looping {
		t.Errorf("標題用的 subsong %d 不會循環", track.Subsong)
	}
	// 走地圖的時間比標題更長，同樣要循環得起來。
	adventure, _ := TrackFor(CueAdventure)
	if !adventure.Looping {
		t.Errorf("地圖用的 subsong %d 不會循環", adventure.Subsong)
	}
}

// 沒有目錄就沒有音樂，而且不是錯誤——可散布的發行包本來就不帶音訊。
func TestNoDirectoryMeansNoPlayer(t *testing.T) {
	player, err := NewPlayer("")
	if err != nil {
		t.Fatal(err)
	}
	if player != nil {
		t.Fatal("沒給目錄卻開出 player")
	}
	// 所有方法對 nil 要安全。
	player.Set(CueTitle)
	if player.Active() != CueNone {
		t.Error("nil player 卻回報有東西在放")
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}
}

// 目錄在但一首都沒有，等同於沒給目錄——不要半開著。
func TestEmptyDirectoryMeansNoPlayer(t *testing.T) {
	player, err := NewPlayer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if player != nil {
		t.Fatal("空目錄卻開出 player")
	}
}
