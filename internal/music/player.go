package music

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// SampleRate 是渲染時用的取樣率，OGG 也是這個。
const SampleRate = 44100

// Player 是可選的配樂輸出。**音訊資產不隨可散布的發行包走**，所以沒有目錄、
// 沒有檔案時它一律靜靜地不放音樂，遊戲照常跑——這不是錯誤路徑，是預設情況。
type Player struct {
	context *audio.Context
	streams map[int]*audio.Player
	active  Cue
	// mode 決定要不要放原版沒有的那兩個情境（見 Mode 的說明）。
	mode Mode
}

// SetMode 換模式。換掉之後目前這一首若不再該放，下一次 Set 會停掉它。
func (p *Player) SetMode(mode Mode) {
	if p == nil {
		return
	}
	p.mode = mode
}

// NewPlayer 開一個輸出。dir 是放 OGG 的目錄；空字串代表不要音樂。
//
// **找不到檔案不是錯誤**：回傳的 Player 為 nil，呼叫端照常操作
//（所有方法都對 nil 安全）。發行包本來就不帶那些檔案。
func NewPlayer(dir string) (*Player, error) {
	if dir == "" {
		return nil, nil
	}
	if err := Validate(); err != nil {
		return nil, err
	}
	context := audio.CurrentContext()
	if context == nil {
		context = audio.NewContext(SampleRate)
	}
	player := &Player{context: context, streams: map[int]*audio.Player{}, mode: ModeOriginal}
	loaded := 0
	for _, track := range Catalog() {
		path := filepath.Join(dir, track.File)
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("讀 %s：%w", path, err)
		}
		stream, err := vorbis.DecodeWithSampleRate(SampleRate, newByteReader(raw))
		if err != nil {
			return nil, fmt.Errorf("解 %s：%w", path, err)
		}
		var made *audio.Player
		if track.Looping {
			// 會循環的那兩首渲染時是截斷的，接回開頭才不會播完就沒了。
			made, err = context.NewPlayer(audio.NewInfiniteLoop(stream, stream.Length()))
		} else {
			made, err = context.NewPlayer(stream)
		}
		if err != nil {
			return nil, fmt.Errorf("開 %s：%w", path, err)
		}
		player.streams[track.Subsong] = made
		loaded++
	}
	if loaded == 0 {
		// 目錄在但一首都沒有：與「沒給目錄」同一件事，不要半開著。
		return nil, nil
	}
	return player, nil
}

// Set 切到某個情境。
//
// **同一個情境不重放**——那不是最佳化，是原版的行為：C64 版的派曲常式先比
// 目前曲號，一樣就直接返回（`$BBF9` 之前那一層）。少了它，每次重新派曲都會
// 把曲子從頭播起，在地圖上走幾步就聽得出來。
func (p *Player) Set(cue Cue) {
	if p == nil || cue == p.active {
		return
	}
	p.stopActive()
	p.active = cue
	if cue == CueNone {
		return
	}
	track, found := TrackForMode(p.mode, cue)
	if !found {
		// 這個模式下這個情境不放音樂——原版在地圖與戰鬥就是靜的。
		return
	}
	stream, ok := p.streams[track.Subsong]
	if !ok {
		return
	}
	_ = stream.Rewind()
	stream.Play()
}

// Active 是現在放的情境，測試用得到。
func (p *Player) Active() Cue {
	if p == nil {
		return CueNone
	}
	return p.active
}

func (p *Player) stopActive() {
	if p.active == CueNone {
		return
	}
	if track, found := TrackForMode(p.mode, p.active); found {
		if stream, ok := p.streams[track.Subsong]; ok {
			stream.Pause()
		}
	}
}

// Close 收掉所有的串流。
func (p *Player) Close() error {
	if p == nil {
		return nil
	}
	for _, stream := range p.streams {
		if err := stream.Close(); err != nil {
			return err
		}
	}
	p.streams = nil
	return nil
}

// newByteReader 把一段位元組包成 vorbis 解碼要的 ReadSeeker。
func newByteReader(raw []byte) *byteReader { return &byteReader{raw: raw} }

type byteReader struct {
	raw []byte
	pos int64
}

func (r *byteReader) Read(out []byte) (int, error) {
	if r.pos >= int64(len(r.raw)) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(out, r.raw[r.pos:])
	r.pos += int64(n)
	return n, nil
}

func (r *byteReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		r.pos = offset
	case 1:
		r.pos += offset
	case 2:
		r.pos = int64(len(r.raw)) + offset
	default:
		return 0, fmt.Errorf("seek whence %d", whence)
	}
	if r.pos < 0 {
		return 0, fmt.Errorf("seek 到負的位置")
	}
	return r.pos, nil
}
