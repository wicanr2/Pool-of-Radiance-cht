package music

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/pc98music"
)

// SampleRate 是渲染時用的取樣率，OGG 也是這個。
const SampleRate = 44100

// PC98Dir 是 PC-98 那 15 首在音樂目錄底下的子目錄（full-local 包的 `music/pc98/`）。
const PC98Dir = "pc98"

// Sink 是實際出聲的那一層。遊戲用 ebiten 的串流；測試換成記錄呼叫的替身，
// 不必有音訊裝置。key 是 Amiga 的 subsong 或 PC-98 的曲號。
type Sink interface {
	Play(key int)
	Stop(key int)
	Close() error
}

// Player 是可選的配樂輸出。**音訊資產不隨可散布的發行包走**，所以沒有目錄、
// 沒有檔案時它一律靜靜地不放音樂，遊戲照常跑——這不是錯誤路徑，是預設情況。
type Player struct {
	out    Sink
	source Source
	// mode 決定 Amiga 來源要不要放原版沒有的那兩個情境（見 Mode 的說明）。
	// PC-98 來源不看它：那一版本來就全程有音樂。
	mode Mode
	// enabled 是原版的音樂開關（`[9D42h]`，Ctrl+O）。
	enabled bool
	now     func() time.Time

	// Amiga：目前的情境。
	active Cue

	// PC-98：current 是最後一次派的曲號（原版的 `[9D40h]`，1 起算，0 ＝ 沒有），
	// sounding 是正在出聲的那一首，pending 是等靜音結束才開播的那一首。
	current, sounding, pending int
	pendingAt                  time.Time
	lastKind                   sceneKind
	lastScene                  Scene
}

// SetMode 換模式。換掉之後目前這一首若不再該放，下一次 Set 會停掉它。
func (p *Player) SetMode(mode Mode) {
	if p == nil {
		return
	}
	p.mode = mode
}

// Source 是這個 player 實際用的曲子來源。
func (p *Player) Source() Source {
	if p == nil {
		return ""
	}
	return p.source
}

// NewPlayer 開 Amiga 來源的輸出。dir 是放 OGG 的目錄；空字串代表不要音樂。
//
// **找不到檔案不是錯誤**：回傳的 Player 為 nil，呼叫端照常操作
//（所有方法都對 nil 安全）。發行包本來就不帶那些檔案。
func NewPlayer(dir string) (*Player, error) {
	return openAmiga(dir)
}

// Open 照想要的來源開輸出。要 PC-98 而 `dir/pc98/` 沒有那 15 首時退回 Amiga
//（回傳的 Player.Source 說明實際用了哪一個）；兩邊都沒有就回傳 nil。
func Open(dir string, source Source) (*Player, error) {
	if dir == "" {
		return nil, nil
	}
	switch source {
	case SourceAmiga:
		return openAmiga(dir)
	case SourcePC98:
		player, err := openPC98(filepath.Join(dir, PC98Dir))
		if err != nil || player != nil {
			return player, err
		}
		return openAmiga(dir)
	default:
		return nil, fmt.Errorf("不認得的配樂來源 %q（只有 %s、%s）", source, SourcePC98, SourceAmiga)
	}
}

func newPlayer(out Sink, source Source) *Player {
	return &Player{out: out, source: source, mode: ModeOriginal, enabled: true, now: time.Now}
}

// NewPlayerWithSink 用給定的出聲層與時鐘開一個 player。測試用它在沒有音訊裝置的
// 環境裡驗派曲規則；now 為 nil 時用 time.Now。
func NewPlayerWithSink(out Sink, source Source, now func() time.Time) *Player {
	player := newPlayer(out, source)
	if now != nil {
		player.now = now
	}
	return player
}

func openAmiga(dir string) (*Player, error) {
	if dir == "" {
		return nil, nil
	}
	if err := Validate(); err != nil {
		return nil, err
	}
	context := audioContext()
	streams := streamSink{}
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
		streams[track.Subsong] = made
	}
	if len(streams) == 0 {
		// 目錄在但一首都沒有：與「沒給目錄」同一件事，不要半開著。
		return nil, nil
	}
	return newPlayer(streams, SourceAmiga), nil
}

// openPC98 讀 `loops.json` 與 15 首 OGG。清單不在就回傳 nil（呼叫端退回 Amiga）；
// 清單在但檔案缺、長度或循環點對不上就回傳錯誤——那代表渲染與打包之間壞了，
// 靜靜退回會把問題藏起來。
func openPC98(dir string) (*Player, error) {
	manifest, err := pc98music.ReadManifest(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if manifest.SampleRate != SampleRate {
		return nil, fmt.Errorf("PC-98 配樂是 %d Hz，遊戲用 %d Hz", manifest.SampleRate, SampleRate)
	}
	if len(manifest.Tracks) != pc98music.SongCount {
		return nil, fmt.Errorf("PC-98 配樂清單有 %d 首，原版是 %d 首", len(manifest.Tracks), pc98music.SongCount)
	}
	context := audioContext()
	streams := streamSink{}
	// 解碼之後是 16 位元立體聲：一個樣本四個位元組。
	const frameBytes = 4
	for _, track := range manifest.Tracks {
		path := filepath.Join(dir, track.File)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("讀 %s：%w", path, err)
		}
		stream, err := vorbis.DecodeWithSampleRate(SampleRate, newByteReader(raw))
		if err != nil {
			return nil, fmt.Errorf("解 %s：%w", path, err)
		}
		if stream.Length() != int64(track.Samples)*frameBytes {
			return nil, fmt.Errorf("%s 解出 %d 個樣本，清單寫 %d：循環點會對不上",
				path, stream.Length()/frameBytes, track.Samples)
		}
		var made *audio.Player
		if track.Loops {
			intro := int64(track.LoopStartSample) * frameBytes
			loop := int64(track.LoopEndSample-track.LoopStartSample) * frameBytes
			made, err = context.NewPlayer(audio.NewInfiniteLoopWithIntro(stream, intro, loop))
		} else {
			made, err = context.NewPlayer(stream)
		}
		if err != nil {
			return nil, fmt.Errorf("開 %s：%w", path, err)
		}
		streams[track.Song] = made
	}
	return newPlayer(streams, SourcePC98), nil
}

func audioContext() *audio.Context {
	if context := audio.CurrentContext(); context != nil {
		return context
	}
	return audio.NewContext(SampleRate)
}

// Update 每一影格叫一次，照畫面狀態派曲。
func (p *Player) Update(scene Scene) {
	if p == nil {
		return
	}
	if p.source == SourcePC98 {
		p.updatePC98(scene)
		return
	}
	p.Set(amigaCue(scene))
}

// updatePC98 照 PC-98 版的規則派曲。
//
// 原版是**事件式**的：標題、商店、戰鬥、結局各在進場那一刻呼叫一次
// `$64E6`，區域配樂則在每一次顯示選單時呼叫 `$63CC`（overlay 26 的選單常式
// `GAME.OVR 32226h`，全遊戲 38 個呼叫點）。這裡把「進場」換成「畫面種類改變
// 的那一影格」，區域配樂每一影格都查——`$64E6` 自己擋掉同一首，所以結果一樣。
func (p *Player) updatePC98(scene Scene) {
	kind := pc98Kind(scene)
	entered := kind != p.lastKind
	p.lastKind, p.lastScene = kind, scene
	switch kind {
	case kindTitle:
		if entered {
			p.request(PC98SongTitle)
		}
	case kindEnding:
		if entered {
			p.request(PC98SongEnding)
		}
	case kindCombat:
		if entered {
			if scene.BossCombat {
				p.request(PC98SongBossCombat)
			} else {
				p.request(PC98SongCombat)
			}
		}
	case kindShop:
		if entered {
			p.request(PC98SongShop)
		}
	case kindTemple:
		// 神殿與商店同屬模式 1，但只有商店派曲：進神殿時場上那一首照放。
	case kindArea:
		p.requestArea(scene.Block)
	}
	p.startPending()
}

func (p *Player) requestArea(block int) {
	if song, ok := PC98AreaSong(block); ok {
		p.request(song)
	}
}

// request 是 `GAME.EXE $64E6`：關掉就不動、同一首不重播、先停、靜音 800 毫秒、再放。
func (p *Player) request(song int) {
	if !p.enabled || song == p.current {
		return
	}
	p.current = song
	p.stopSounding()
	p.pending, p.pendingAt = song, p.now().Add(PC98SwitchSilence)
}

func (p *Player) startPending() {
	if p.pending == 0 || !p.enabled || p.now().Before(p.pendingAt) {
		return
	}
	p.sounding, p.pending = p.pending, 0
	p.out.Play(p.sounding)
}

func (p *Player) stopSounding() {
	if p.sounding != 0 {
		p.out.Stop(p.sounding)
	}
	p.sounding, p.pending = 0, 0
}

// ToggleEnabled 是原版的音樂開關（`GAME.EXE $5EAC`：`[9D42h] xor 1` 之後呼叫
// `$63CC`）。關掉時停止並忘掉目前曲號；打開時只重派**區域配樂**——
// 在戰鬥、商店、神殿、結局裡打開，原版那張表被模式擋掉，要等下一次派曲才有聲音。
func (p *Player) ToggleEnabled() {
	if p == nil {
		return
	}
	p.enabled = !p.enabled
	if !p.enabled {
		if p.source == SourcePC98 {
			p.current = 0
			p.stopSounding()
		} else {
			p.stopActive()
			p.active = CueNone
		}
		return
	}
	if p.source == SourcePC98 && p.lastKind == kindArea {
		p.requestArea(p.lastScene.Block)
	}
}

// Enabled 回報音樂開關；nil player 一律是關的。
func (p *Player) Enabled() bool { return p != nil && p.enabled }

// Current 是 PC-98 來源最後一次派的曲號（0 ＝ 沒有）。
func (p *Player) Current() int {
	if p == nil {
		return 0
	}
	return p.current
}

// Sounding 是 PC-98 來源正在出聲的曲號（靜音那 800 毫秒裡是 0）。
func (p *Player) Sounding() int {
	if p == nil {
		return 0
	}
	return p.sounding
}

// Set 切到某個情境（Amiga 來源）。
//
// **同一個情境不重放**——那不是最佳化，是原版的行為：C64 版的派曲常式先比
// 目前曲號，一樣就直接返回（`$BBF9` 之前那一層）。少了它，每次重新派曲都會
// 把曲子從頭播起，在地圖上走幾步就聽得出來。
func (p *Player) Set(cue Cue) {
	if p == nil || !p.enabled || cue == p.active {
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
	p.out.Play(track.Subsong)
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
		p.out.Stop(track.Subsong)
	}
}

// Close 收掉所有的串流。
func (p *Player) Close() error {
	if p == nil || p.out == nil {
		return nil
	}
	err := p.out.Close()
	p.out = nil
	return err
}

// streamSink 是 ebiten 的串流，key 是 Amiga 的 subsong 或 PC-98 的曲號。
type streamSink map[int]*audio.Player

func (s streamSink) Play(key int) {
	if stream, ok := s[key]; ok {
		_ = stream.Rewind()
		stream.Play()
	}
}

func (s streamSink) Stop(key int) {
	if stream, ok := s[key]; ok {
		stream.Pause()
	}
}

func (s streamSink) Close() error {
	for key, stream := range s {
		if err := stream.Close(); err != nil {
			return err
		}
		delete(s, key)
	}
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
		// 要是 io.EOF 本身：解碼器靠它分辨「讀完了」與「讀壞了」。
		return 0, io.EOF
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
