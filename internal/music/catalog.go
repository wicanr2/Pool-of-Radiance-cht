// Package music 是 remake 的配樂目錄與派曲規則。
//
// # 素材
//
// 原版 **DOS 版沒有音樂**：`Pool of Radiance (1988).zip` 裡沒有任何音樂檔，
// 那一版只有 PC 喇叭。有音樂的是別的平台版本：
//
//   - **C64（1988）**：磁碟上有 `MUSIC`（$4000）與 `SOUNDFX`／`MDRIVER`（$BA00）。
//   - **Amiga（1990，U.S. Gold／SSI）**：Wally Beben 作曲，一個自訂播放器模組
//     `wb.Pool_of_Radiance` 裡有 **6 首 subsong**。
//
// remake 用 Amiga 那一份（見 docs/spec/128-music-cues.md）。音訊是第三方著作權，
// **不進 repo、不隨可散布的發行包走**；只有本機的 full-local 包會帶。
//
// # 派曲的形狀從哪裡來
//
// DOS 版沒有音樂，所以「什麼時候放」不可能有 DOS 的對照。C64 版有，而且結構
// 讀得出來：驅動掛在 `LIBRARY $1FA2` 的 IRQ 上每格 tick，`$BA03` 是「放第 A 首」，
// 而全遊戲只有三支包裝——**INIT（開機／標題）、DUNGEON、COMBAT**。
//
// 所以 [Cue] 這三個情境是**原版結構的證據**；至於每個情境配 Amiga 的哪一首，
// 是 **remake 自己決定的**，不是原版對照——Amiga 版的遊戲程式我們沒有。
// 這條界線寫在型別註解裡，因為它最容易在轉述時變成「照原版接的」。
package music

import "fmt"

// Cue 是「現在該放哪一種曲子」。三個值對應 C64 版驅動的三支包裝。
type Cue string

const (
	// CueTitle 對應 C64 的 INIT（開機與標題）。
	CueTitle Cue = "title"
	// CueAdventure 對應 C64 的 DUNGEON（在地圖上走動）。
	CueAdventure Cue = "adventure"
	// CueCombat 對應 C64 的 COMBAT。
	CueCombat Cue = "combat"
	// CueNone 是「不放」。派曲常式收到它就停。
	CueNone Cue = ""
)

// Cues 是全部的情境，順序固定，測試與工具靠它列舉。
func Cues() []Cue { return []Cue{CueTitle, CueAdventure, CueCombat} }

// SubsongCount 是 `wb.Pool_of_Radiance` 裡的 subsong 數，由 UADE 讀出來
//（`uade123 -g` 回報 "There are 6 subsongs in range [1, 6]"）。
const SubsongCount = 6

// Track 是一首可以放的曲子。
type Track struct {
	// Subsong 是模組裡的第幾首（1-based，與 UADE 的編號相同）。
	Subsong int
	// File 是渲染出來的 OGG 檔名。
	File string
	// Seconds 是渲染長度。Looping 為真代表原曲會一直循環，這個長度是
	// **渲染時截斷的**，不是曲子的自然結尾。
	Seconds float64
	Looping bool
}

// Catalog 是六首的目錄。長度、循環與否都是實際量出來的：
// 逐檔解出 PCM 之後算 RMS 與非靜音比例，六首都是真的音樂，不是空檔。
func Catalog() []Track {
	return []Track{
		{Subsong: 1, File: "por-amiga-01.ogg", Seconds: 240.0, Looping: true},
		{Subsong: 2, File: "por-amiga-02.ogg", Seconds: 7.7},
		{Subsong: 3, File: "por-amiga-03.ogg", Seconds: 7.7},
		{Subsong: 4, File: "por-amiga-04.ogg", Seconds: 15.4},
		{Subsong: 5, File: "por-amiga-05.ogg", Seconds: 7.7},
		{Subsong: 6, File: "por-amiga-06.ogg", Seconds: 153.7, Looping: true},
	}
}

// Binding 把情境對到 subsong。
//
// **這一張表是 remake 自己決定的**，理由寫在每一列的註解裡；它不是原版對照。
// Amiga 版的遊戲程式不在手上，所以「Amiga 版在哪裡放哪一首」目前無法核對。
func Bindings() map[Cue]int {
	return map[Cue]int{
		// 第 1 首是六首裡唯一夠長又會循環的（渲染 240 秒仍未結束），
		// 音量也最飽（peak −0.2 dBFS）。標題要的就是這種。
		CueTitle: 1,
		// 第 6 首是另一首長曲，153.7 秒自然結束。走地圖的時間最長，
		// 放第二長的那一首。
		CueAdventure: 6,
		// 第 4 首 15.4 秒，是四首短曲裡最長、也是唯一峰值低於 −3 dBFS 的
		// （其餘三首都逼近滿刻度），適合當戰鬥的底。
		CueCombat: 4,
	}
}

// TrackFor 回傳這個情境要放的那一首。
func TrackFor(cue Cue) (Track, bool) {
	subsong, found := Bindings()[cue]
	if !found {
		return Track{}, false
	}
	for _, track := range Catalog() {
		if track.Subsong == subsong {
			return track, true
		}
	}
	return Track{}, false
}

// Validate 檢查目錄與對應表自洽：subsong 編號在模組的範圍內、沒有重複的檔名、
// 每個情境都指到目錄裡真的有的那一首。
func Validate() error {
	seenSubsong := map[int]bool{}
	seenFile := map[string]bool{}
	for _, track := range Catalog() {
		if track.Subsong < 1 || track.Subsong > SubsongCount {
			return fmt.Errorf("subsong %d 超出模組的 1..%d", track.Subsong, SubsongCount)
		}
		if seenSubsong[track.Subsong] {
			return fmt.Errorf("subsong %d 列了兩次", track.Subsong)
		}
		if seenFile[track.File] {
			return fmt.Errorf("檔名 %s 列了兩次", track.File)
		}
		seenSubsong[track.Subsong], seenFile[track.File] = true, true
	}
	if len(seenSubsong) != SubsongCount {
		return fmt.Errorf("目錄有 %d 首，模組是 %d 首", len(seenSubsong), SubsongCount)
	}
	for _, cue := range Cues() {
		if _, found := TrackFor(cue); !found {
			return fmt.Errorf("情境 %q 沒有對到任何一首", cue)
		}
	}
	return nil
}
