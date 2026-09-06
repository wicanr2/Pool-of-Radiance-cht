package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/music"
)

// 配樂（spec 128）。
//
// 原版 DOS 版沒有音樂，所以這一層**沒有 DOS 對照可比**。曲子來自 Amiga 版
// （Wally Beben），情境的形狀來自 C64 版驅動的三支包裝——`INIT`（開機／標題）、
// `DUNGEON`、`COMBAT`。「哪一個情境配哪一首」是 remake 自己決定的，
// 理由寫在 internal/music 的 Bindings。
//
// **音訊不隨可散布的發行包走**：沒有 `-music-dir`、或那個目錄裡沒有檔案，
// 遊戲就安靜地跑。那是預設情況，不是錯誤路徑。

// audioDeviceLikelyAvailable 在開音樂之前先看有沒有音訊裝置。
//
// **為什麼要先看**：oto 是第一次播放時才真的開 ALSA／PulseAudio，而第一次
// 播放就在標題那一影格。開不了的時候錯誤從遊戲迴圈冒出來，`RunGame` 直接返回
// ——**整個遊戲打不開**，而症狀完全看不出跟音樂有關。實測容器裡就是這樣：
// `oto: ALSA error at snd_pcm_open: "default": No such file or directory`。
//
// 失敗之後再跑一次 `RunGame` 是不行的：Ebiten 的主迴圈不能重來，第二次不會
// 開出視窗，整支就掛在那裡。所以只能事前判斷。
//
// 判斷只在 Linux 做，而且是保守的：看得到裝置才開音樂，看不到就安靜地跑。
// 誤判成「沒有」的代價是沒有音樂；誤判成「有」的代價是遊戲打不開——
// 兩邊的代價差很多，所以往沒有那一邊倒。
func audioDeviceLikelyAvailable() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	// PulseAudio／PipeWire 的 socket 在的話一定放得出來。
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "pulse", "native")); err == nil {
			return true
		}
	}
	// 退一步看 ALSA：沒有音效卡時 /proc/asound/cards 的內容是
	// `--- no soundcards ---`。
	cards, err := os.ReadFile("/proc/asound/cards")
	if err != nil {
		return false
	}
	return !strings.Contains(string(cards), "no soundcards")
}

// defaultMusicDir 是發行包裡的相對位置。full-local 包會把 OGG 放在
// 執行檔旁邊的 `music/`；可散布的 patch 包沒有那個目錄。
func defaultMusicDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Join(filepath.Dir(exe), "music")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	return ""
}

// musicCue 由目前的畫面狀態決定要放哪一種曲子。
//
// 判斷順序照「玩家看到什麼」：戰術盤面上就是戰鬥，地圖上就是冒險，
// 其餘（標題、隊伍管理、建角）都算標題。
func (a *app) musicCue() music.Cue {
	switch {
	case a.tactical != nil && a.tacticalPreview:
		return music.CueCombat
	case a.mode == modeAdventure:
		return music.CueAdventure
	default:
		return music.CueTitle
	}
}

// updateMusic 每一影格叫一次。`Set` 自己會擋掉「同一個情境重放」，
// 所以這裡不必記上一次是什麼——那條規則的理由寫在 music.Player.Set。
func (a *app) updateMusic() {
	a.musicPlayer.Set(a.musicCue())
}
