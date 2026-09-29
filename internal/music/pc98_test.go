package music

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// `GAME.EXE $63CC` 逐筆：`$6403..$64D8` 的比較鏈，每一個 `cmp ax,N` 對到哪一個
// `mov byte [bp-1],曲號`。這張表是照反組譯逐行抄的，**不是**從 PC98AreaSongs 反推。
func TestPC98AreaTableMatchesGameExe63CC(t *testing.T) {
	want := map[int]int{
		0: 2, 8: 2, 11: 2, // $6408..$6417
		2: 3, 15: 3, 18: 3, 20: 3, 29: 3, // $641E..$6437
		14: 4, 21: 4, 24: 4, // $643E..$644D
		19: 5,                             // $6454
		1:  6, 13: 6, 16: 6, 17: 6, 28: 6, // $645F..$6478
		22: 7, 23: 7, // $647E..$6488
		10: 8,                         // $648E
		3:  9, 4: 9, 5: 9, 6: 9, 9: 9, // $6499..$64B2
		7:  10,                 // $64B8
		25: 13, 26: 13, 27: 13, // $64C3..$64D2
	}
	got := PC98AreaSongs()
	if len(got) != 29 || len(want) != 29 {
		t.Fatalf("表有 %d 筆（抄錄 %d 筆），原版是 29 個 ECL 區塊", len(got), len(want))
	}
	for block, song := range want {
		if got[block] != song {
			t.Errorf("區塊 %d 對到第 %d 首，`$63CC` 是第 %d 首", block, got[block], song)
		}
	}
	// 區塊編號正好是 0..11 與 13..29（spec 101：沒有 12）。
	for block := 0; block <= 29; block++ {
		_, ok := PC98AreaSong(block)
		if ok == (block == 12) {
			t.Errorf("區塊 %d 在表裡＝%v", block, ok)
		}
	}
	// 開機時 `[9D3Fh]` 先是 FFh（`$1C0B`）：表外的值不派曲。
	if _, ok := PC98AreaSong(0xFF); ok {
		t.Error("FFh 不該對到任何一首")
	}
	// 用到的曲號是 2..10 與 13；1、11、12、14、15 留給非區域的情境。
	used := map[int]bool{}
	for _, song := range got {
		used[song] = true
	}
	for _, song := range []int{PC98SongTitle, PC98SongCombat, PC98SongBossCombat, PC98SongShop, PC98SongEnding} {
		if used[song] {
			t.Errorf("第 %d 首是非區域曲，卻出現在區域表裡", song)
		}
	}
}

// fakeSink 記下每一次出聲與停止。
type fakeSink struct{ calls []string }

func (s *fakeSink) Play(key int) { s.calls = append(s.calls, fmt.Sprintf("play %d", key)) }
func (s *fakeSink) Stop(key int) { s.calls = append(s.calls, fmt.Sprintf("stop %d", key)) }
func (s *fakeSink) Close() error { return nil }

// fakeClock 是可以撥的時鐘。
type fakeClock struct{ at time.Time }

func (c *fakeClock) now() time.Time                 { return c.at }
func (c *fakeClock) advance(duration time.Duration) { c.at = c.at.Add(duration) }

func newTestPC98(t *testing.T) (*Player, *fakeSink, *fakeClock) {
	t.Helper()
	sink, clock := &fakeSink{}, &fakeClock{at: time.Unix(0, 0)}
	return NewPlayerWithSink(sink, SourcePC98, clock.now), sink, clock
}

// settle 走過 800 毫秒的靜音，讓等著的那一首開播。
func settle(player *Player, clock *fakeClock, scene Scene) {
	clock.advance(PC98SwitchSilence)
	player.Update(scene)
}

func area(block int) Scene { return Scene{Adventure: true, Block: block} }

// 換曲先停、靜音 800 毫秒再放（`$6505..$6525`）；靜音裡同一首再派一次也不重播。
func TestPC98SwitchStopsThenWaits800Milliseconds(t *testing.T) {
	player, sink, clock := newTestPC98(t)
	player.Update(Scene{Title: true, Block: -1})
	if player.Current() != PC98SongTitle || player.Sounding() != 0 {
		t.Fatalf("標題：派 %d、出聲 %d，要派 1、先靜音", player.Current(), player.Sounding())
	}
	clock.advance(PC98SwitchSilence - time.Millisecond)
	player.Update(Scene{Title: true, Block: -1})
	if player.Sounding() != 0 {
		t.Fatal("還沒滿 800 毫秒就出聲了")
	}
	clock.advance(time.Millisecond)
	player.Update(Scene{Title: true, Block: -1})
	if player.Sounding() != PC98SongTitle {
		t.Fatalf("滿 800 毫秒之後出聲 %d，要 1", player.Sounding())
	}
	// 開始選單：開機初始化把 `[9D3Fh]` 設成 0，區域表放第 2 首。
	player.Update(Scene{Block: PC98StartBlock})
	if player.Sounding() != 0 || player.Current() != 2 {
		t.Fatalf("換曲那一刻出聲 %d、派 %d，要先停、派 2", player.Sounding(), player.Current())
	}
	clock.advance(PC98SwitchSilence / 2)
	player.Update(area(0))
	settle(player, clock, area(8)) // 市政廳也是第 2 首：不重來
	want := []string{"play 1", "stop 1"}
	if player.Sounding() != 2 {
		t.Fatalf("出聲 %d，要 2", player.Sounding())
	}
	want = append(want, "play 2")
	if fmt.Sprint(sink.calls) != fmt.Sprint(want) {
		t.Fatalf("呼叫 %v，要 %v（同一首不重播）", sink.calls, want)
	}
}

// 冒險換區、進戰鬥、回冒險、商店、神殿、結局——照原版的曲號。
func TestPC98CueSequenceFollowsTheOriginal(t *testing.T) {
	player, _, clock := newTestPC98(t)
	steps := []struct {
		name  string
		scene Scene
		want  int
	}{
		{"標題", Scene{Title: true, Block: -1}, PC98SongTitle},
		{"城區（區塊 0）", area(0), 2},
		{"貧民窟（區塊 20）", area(20), 3},
		{"一般戰鬥", Scene{Adventure: true, Combat: true, Block: 20}, PC98SongCombat},
		{"戰後回到地圖", area(20), 3},
		{"商店", Scene{Adventure: true, Shop: true, Block: 20}, PC98SongShop},
		// 神殿與商店同屬模式 1，只有商店派曲：場上那一首照放。
		{"出商店", area(20), 3},
		{"神殿", Scene{Adventure: true, Temple: true, Block: 20}, 3},
		{"荒野（區塊 25）", area(25), 13},
		{"荒野（區塊 27）", area(27), 13},
		{"瓦爾耶沃城堡（區塊 6）", area(6), 9},
		{"提蘭斯拉克蘇斯", Scene{Adventure: true, Combat: true, BossCombat: true, Block: 6}, PC98SongBossCombat},
		{"結局", Scene{Adventure: true, Ending: true, Block: 6}, PC98SongEnding},
	}
	for _, step := range steps {
		player.Update(step.scene)
		settle(player, clock, step.scene)
		if player.Sounding() != step.want {
			t.Errorf("%s：出聲第 %d 首，原版是第 %d 首", step.name, player.Sounding(), step.want)
		}
	}
}

// 戰鬥、商店、結局是「進場」派一次（overlay 直接呼叫 `$64E6`），不是每一影格；
// 戰鬥中再怎麼更新都不會回頭放區域曲，直到戰鬥結束。
func TestPC98CombatHoldsItsSongUntilItEnds(t *testing.T) {
	player, _, clock := newTestPC98(t)
	settle(player, clock, area(0))
	combat := Scene{Adventure: true, Combat: true, Block: 0}
	player.Update(combat)
	for frame := 0; frame < 10; frame++ {
		settle(player, clock, combat)
	}
	if player.Sounding() != PC98SongCombat {
		t.Fatalf("戰鬥中出聲 %d", player.Sounding())
	}
}

// Ctrl+O（`$5EAC`）：關掉就停、忘掉曲號；在地圖上打開會重派區域曲；
// 在戰鬥裡打開，區域表被模式 5 擋掉，要等下一次派曲才有聲音。
func TestPC98ToggleFollowsTheOriginalGate(t *testing.T) {
	player, sink, clock := newTestPC98(t)
	settle(player, clock, area(19))
	settle(player, clock, area(19))
	if player.Sounding() != 5 {
		t.Fatalf("洞穴出聲 %d，要 5", player.Sounding())
	}
	player.ToggleEnabled()
	if player.Enabled() || player.Sounding() != 0 || player.Current() != 0 {
		t.Fatalf("關掉之後 enabled=%v 出聲 %d 派 %d", player.Enabled(), player.Sounding(), player.Current())
	}
	settle(player, clock, area(2))
	if player.Sounding() != 0 || player.Current() != 0 {
		t.Fatal("關著的時候換區還是派了曲")
	}
	player.ToggleEnabled()
	settle(player, clock, area(2))
	if player.Sounding() != 3 {
		t.Fatalf("在地圖上打開之後出聲 %d，要區塊 2 的第 3 首", player.Sounding())
	}

	// 戰鬥裡關掉再打開：沒有聲音，直到戰鬥結束回到地圖。
	combat := Scene{Adventure: true, Combat: true, Block: 2}
	settle(player, clock, combat)
	player.ToggleEnabled()
	player.ToggleEnabled()
	settle(player, clock, combat)
	if player.Sounding() != 0 {
		t.Fatalf("戰鬥裡重新打開就出聲 %d；原版要等下一次派曲", player.Sounding())
	}
	settle(player, clock, area(2))
	settle(player, clock, area(2))
	if player.Sounding() != 3 {
		t.Fatalf("戰後回地圖出聲 %d，要 3", player.Sounding())
	}
	if len(sink.calls) == 0 {
		t.Fatal("沒有任何出聲紀錄")
	}
}

// Amiga 來源維持 spec 128 的行為：original 只有標題，full 三個情境都放；不加 800 毫秒。
func TestAmigaSourceKeepsTheSpec128Cues(t *testing.T) {
	sink := &fakeSink{}
	player := NewPlayerWithSink(sink, SourceAmiga, nil)
	player.Update(Scene{Title: true})
	player.Update(area(0))
	player.Update(Scene{Adventure: true, Combat: true})
	if fmt.Sprint(sink.calls) != "[play 1 stop 1]" {
		t.Fatalf("original 模式呼叫 %v", sink.calls)
	}
	sink.calls = nil
	player = NewPlayerWithSink(sink, SourceAmiga, nil)
	player.SetMode(ModeFull)
	player.Update(Scene{Title: true})
	player.Update(area(0))
	player.Update(Scene{Adventure: true, Combat: true})
	if fmt.Sprint(sink.calls) != "[play 1 stop 1 play 6 stop 6 play 4]" {
		t.Fatalf("full 模式呼叫 %v", sink.calls)
	}
}

// 要 PC-98 而 `pc98/` 沒有清單：退回 Amiga（這裡兩邊都沒有，所以是 nil）。
// 清單在但壞了要報錯，不能安靜地退回。
func TestOpenFallsBackOnlyWhenThePC98ManifestIsAbsent(t *testing.T) {
	dir := t.TempDir()
	player, err := Open(dir, SourcePC98)
	if err != nil || player != nil {
		t.Fatalf("空目錄：player=%v err=%v", player, err)
	}
	if err := os.MkdirAll(filepath.Join(dir, PC98Dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, PC98Dir, "loops.json"), []byte(`{"sample_rate":44100,"tracks":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir, SourcePC98); err == nil {
		t.Fatal("清單只有 0 首卻沒有報錯")
	}
	if _, err := Open(dir, Source("sid")); err == nil {
		t.Fatal("不認得的來源卻沒有報錯")
	}
}

// 本機有渲染好的 15 首時，整套真的開得起來（不出聲：沒有音訊裝置也能解碼）。
func TestRenderedPC98MusicOpens(t *testing.T) {
	dir := filepath.Join("..", "..", "workplace", "pc98-music", "ogg")
	if _, err := os.Stat(filepath.Join(dir, "loops.json")); err != nil {
		t.Skip("workplace/pc98-music/ogg 不在（PC-98 配樂是 Pony Canyon 的著作權，不進 repo）")
	}
	player, err := openPC98(dir)
	if err != nil {
		t.Fatal(err)
	}
	if player == nil || player.Source() != SourcePC98 {
		t.Fatalf("開出 %v", player)
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}
}
