package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/music"
)

// 配樂（spec 128、169）。
//
// 原版 DOS 版沒有音樂，所以這一層**沒有 DOS 對照可比**。曲子有兩個來源：
//
//   - **PC-98 版**（預設，spec 169）：15 首 YM2203，派曲規則照那一版的
//     `GAME.EXE` 逐條讀出（區塊對照表 `$63CC`、標題／戰鬥／商店／結局的直接呼叫）。
//   - **Amiga 版**（spec 128）：Wally Beben 的六首；情境的形狀來自 C64 版驅動的
//     三支包裝，「哪一個情境配哪一首」是 remake 自己決定的（internal/music 的 Bindings）。
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

// musicScene 把目前的畫面狀態交給派曲規則（spec 169）。
//
// 各欄位對到 PC-98 版 `GAME.EXE` 的哪一個模式寫在 music.Scene；這裡只負責
// 從 remake 的狀態讀出來。
func (a *app) musicScene() music.Scene {
	scene := music.Scene{
		Title:      a.mode == modeTitle,
		Adventure:  a.mode == modeAdventure,
		Combat:     a.tactical != nil && a.tacticalPreview,
		BossCombat: a.lastLoadedMonster == music.PC98BossMonsterID,
		Shop:       a.shopActive,
		Temple:     a.templeActive,
		Ending:     a.endingActive,
		Block:      music.PC98StartBlock,
	}
	// 還沒有 ECL 區塊時（開始選單、建角）是開機初始化留下的 0；
	// 有了就是目前的區塊（原版 `[9D3Fh]` 由區塊載入時寫入，overlay 7 `803Dh`）。
	if a.eventSession != nil {
		scene.Block = int(a.eventSession.CurrentBlockID())
	}
	return scene
}

// updateMusic 每一影格叫一次。「同一首不重播」由 player 自己擋，
// 所以這裡不必記上一次是什麼——那條規則的理由寫在 music.Player。
func (a *app) updateMusic() {
	a.musicPlayer.Update(a.musicScene())
}

// toggleMusicKey 是原版的音樂開關：PC-98 `GAME.EXE $5EA6` 讀到按鍵碼 `0Fh`
//（Ctrl+O）就把 `[9D42h]` 反相並重派區域配樂（spec 169）。
//
// 按到了就吃掉這一影格：O 在營地、作弊選單裡另有用途，同一次按鍵不該兩邊都作用。
func (a *app) toggleMusicKey() bool {
	if !a.keyHeld(ebiten.KeyControl) || !a.justPressed(ebiten.KeyO) {
		return false
	}
	// 原版切換時畫面上沒有任何訊息（`$5EAC..$5EB9` 只改旗標、重派曲），這裡也不加。
	a.musicPlayer.ToggleEnabled()
	return true
}
