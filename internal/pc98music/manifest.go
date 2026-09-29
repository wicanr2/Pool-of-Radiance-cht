package pc98music

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ManifestFile 是與 OGG 放在同一個目錄的循環點清單。
const ManifestFile = "loops.json"

// SongCount 是 `MSCDRV.EXE` 曲目表的曲數：表長 `$B4` ＝ 15 × 12，
// 之後緊接第 1 首的資料（pc98golem 的 docs/spec/005-pool-pc98-music.md，本 repo 的 spec 169）。
const SongCount = 15

// TrackFile 是第 song 首（1 起算）的 OGG 檔名。
func TrackFile(song int) string { return fmt.Sprintf("pc98-%02d.ogg", song) }

// Manifest 是渲染工具寫、遊戲讀的循環點清單。
type Manifest struct {
	Source       string  `json:"source"`
	DriverSHA256 string  `json:"driver_sha256"`
	SampleRate   int     `json:"sample_rate"`
	Tracks       []Track `json:"tracks"`
}

// Track 是一首曲子。樣本位置都以 SampleRate 計、單聲道。
type Track struct {
	Song    int    `json:"song"`
	File    string `json:"file"`
	Samples int    `json:"samples"`
	// Loops 為真時，播完 LoopEndSample 之後接回 LoopStartSample。
	// LoopEndSample 等於 Samples（檔尾就是循環終點）。
	Loops           bool `json:"loops"`
	LoopStartSample int  `json:"loop_start_sample,omitempty"`
	LoopEndSample   int  `json:"loop_end_sample,omitempty"`
	// IntroTicks、PeriodTicks 是音序 tick 上的循環形狀，回頭對驅動用。
	IntroTicks  uint64  `json:"intro_ticks,omitempty"`
	PeriodTicks uint64  `json:"period_ticks,omitempty"`
	PeakDBFS    float64 `json:"peak_dbfs"`
	RMSDBFS     float64 `json:"rms_dbfs"`
	Clipped     int     `json:"clipped_samples"`
}

// Seconds 是整個檔案的長度。
func (t Track) Seconds(rate int) float64 { return float64(t.Samples) / float64(rate) }

// LoopStartSeconds 是循環起點的時間。
func (t Track) LoopStartSeconds(rate int) float64 { return float64(t.LoopStartSample) / float64(rate) }

// ReadManifest 讀 dir 裡的循環點清單並檢查它自洽。
func ReadManifest(dir string) (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if err != nil {
		return Manifest{}, err
	}
	return ParseManifest(raw)
}

// ParseManifest 解析並檢查：曲號在 1..15 且不重複、檔名照規則、
// 循環點落在檔案裡而且循環段不是空的。**對不上就拒絕**——循環點錯了的症狀是
// 接縫處跳一下或整首從頭重來，聽得出來但查不出原因。
func ParseManifest(raw []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("%s：%w", ManifestFile, err)
	}
	if manifest.SampleRate <= 0 {
		return Manifest{}, fmt.Errorf("%s：取樣率 %d 不合理", ManifestFile, manifest.SampleRate)
	}
	seen := map[int]bool{}
	for _, track := range manifest.Tracks {
		if track.Song < 1 || track.Song > SongCount {
			return Manifest{}, fmt.Errorf("%s：曲號 %d 超出 1..%d", ManifestFile, track.Song, SongCount)
		}
		if seen[track.Song] {
			return Manifest{}, fmt.Errorf("%s：第 %d 首列了兩次", ManifestFile, track.Song)
		}
		seen[track.Song] = true
		if track.File != TrackFile(track.Song) {
			return Manifest{}, fmt.Errorf("%s：第 %d 首的檔名是 %q，預期 %q",
				ManifestFile, track.Song, track.File, TrackFile(track.Song))
		}
		if track.Samples <= 0 {
			return Manifest{}, fmt.Errorf("%s：第 %d 首沒有樣本", ManifestFile, track.Song)
		}
		if !track.Loops {
			continue
		}
		if track.LoopStartSample < 0 || track.LoopEndSample != track.Samples ||
			track.LoopStartSample >= track.LoopEndSample {
			return Manifest{}, fmt.Errorf("%s：第 %d 首的循環段 %d..%d 不在檔案 0..%d 裡",
				ManifestFile, track.Song, track.LoopStartSample, track.LoopEndSample, track.Samples)
		}
	}
	return manifest, nil
}
