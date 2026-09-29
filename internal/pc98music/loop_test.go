package pc98music

import (
	"encoding/binary"
	"testing"

	"github.com/wicanr2/golden-box-remake-engine/audio/pc98mscdrv"
)

// syntheticDriver 組一顆最小的驅動：曲目表之後接每個聲道的串流與區塊。
// streams 的每一項是一個聲道的串流位元組，裡面的指標用 fix 回填成絕對偏移。
type builder struct{ data []byte }

func (b *builder) at() int { return len(b.data) }
func (b *builder) emit(bytes ...byte) int {
	start := len(b.data)
	b.data = append(b.data, bytes...)
	return start
}
func (b *builder) word(at, value int) { binary.LittleEndian.PutUint16(b.data[at:], uint16(value)) }

func openSynthetic(t *testing.T, data []byte, channels int) *pc98mscdrv.Driver {
	t.Helper()
	driver, err := pc98mscdrv.Open(data, pc98mscdrv.Layout{
		DataSegment: 0, TrackTable: 0, Tracks: 1, Channels: channels, FMChannels: channels,
	})
	if err != nil {
		t.Fatal(err)
	}
	return driver
}

// 串流：A、(B 兩次)、然後 C 一直繞（`82` 跳回自己）。
// 前奏是 A B B（72 tick），循環段是 C（48 tick）。
func loopingChannel(b *builder, table int) {
	b.word(table, b.at())
	a := b.emit(0x80, 0, 0)
	b.emit(0x85, 2, 0)
	bb := b.emit(0x80, 0, 0)
	b.emit(0x86)
	c := b.emit(0x80, 0, 0)
	jump := b.emit(0x82, 0, 0)
	b.word(jump+1, c)
	b.word(a+1, b.emit(2, 0x3C, 24))
	b.word(bb+1, b.emit(4, 0x3E, 12, 0x80, 12))
	b.word(c+1, b.emit(2, 0x40, 48))
}

func TestAnalyzeChannelFindsIntroAndPeriod(t *testing.T) {
	b := &builder{}
	b.emit(0, 0)
	loopingChannel(b, 0)
	driver := openSynthetic(t, b.data, 1)
	result, err := driver.ExpandChannel(0, 0, pc98mscdrv.ExpandOptions{MaxBlocks: 64})
	if err != nil {
		t.Fatal(err)
	}
	loop, err := AnalyzeChannel(result)
	if err != nil {
		t.Fatal(err)
	}
	if !loop.Loops || loop.IntroBlock != 3 || loop.PeriodBlocks != 1 || loop.IntroTick != 72 || loop.PeriodTick != 48 {
		t.Fatalf("得到 %+v，要前奏 3 區塊／72 tick、週期 1 區塊／48 tick", loop)
	}
}

// `81` 收尾的聲道不循環，結束點是最後一個音的結尾。
func TestAnalyzeChannelReportsTheEndOfAFiniteChannel(t *testing.T) {
	b := &builder{}
	b.emit(0, 0)
	b.word(0, b.at())
	a := b.emit(0x80, 0, 0)
	b.emit(0x81)
	b.word(a+1, b.emit(4, 0x3C, 24, 0x80, 36))
	driver := openSynthetic(t, b.data, 1)
	song, err := AnalyzeSong(driver, 0)
	if err != nil {
		t.Fatal(err)
	}
	if song.Loops || song.EndTick != 60 {
		t.Fatalf("得到 %+v，要不循環、結束在 60 tick", song)
	}
}

// 兩個聲道週期 48 與 96：整首的週期是最小公倍數 96，前奏取最長的那一個。
func TestAnalyzeSongCombinesChannels(t *testing.T) {
	b := &builder{}
	b.emit(0, 0, 0, 0)
	loopingChannel(b, 0)
	b.word(2, b.at())
	c := b.emit(0x80, 0, 0)
	jump := b.emit(0x82, 0, 0)
	b.word(jump+1, c)
	b.word(c+1, b.emit(2, 0x30, 96))
	driver := openSynthetic(t, b.data, 2)
	song, err := AnalyzeSong(driver, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !song.Loops || song.StartTick != 72 || song.PeriodTick != 96 {
		t.Fatalf("得到 %+v，要從 72 tick 起、每 96 tick 一輪", song)
	}
}

// 樣本位置照排程逐段累加，速度命令在它出現的那一格之後生效。
func TestTimelineFollowsTempoChanges(t *testing.T) {
	b := &builder{}
	b.emit(0, 0)
	b.word(0, b.at())
	a := b.emit(0x80, 0, 0)
	b.emit(0x81)
	// 先 24 tick（預設 120 BPM），換成 60 BPM 再 24 tick。
	b.word(a+1, b.emit(6, 0x3C, 24, 0x84, 60, 0x3C, 24))
	driver := openSynthetic(t, b.data, 1)
	// 120 BPM × 24 tick／拍：一個 tick 1/48 秒；取樣率 4800 → 100 個樣本。
	timeline, err := NewTimeline(driver, 0, pc98mscdrv.RenderOptions{}, 4800)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		tick uint64
		want int
	}{{0, 0}, {12, 1200}, {24, 2400}, {36, 2400 + 2400}, {48, 2400 + 4800}} {
		if got := timeline.SampleAt(check.tick); got != check.want {
			t.Errorf("tick %d 在樣本 %d，要 %d", check.tick, got, check.want)
		}
	}
}

func TestParseManifestRejectsBrokenLoops(t *testing.T) {
	good := `{"sample_rate":44100,"tracks":[{"song":1,"file":"pc98-01.ogg","samples":100,"loops":true,"loop_start_sample":10,"loop_end_sample":100}]}`
	if _, err := ParseManifest([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string]string{
		"曲號超出":     `{"sample_rate":44100,"tracks":[{"song":16,"file":"pc98-16.ogg","samples":100}]}`,
		"檔名不對":     `{"sample_rate":44100,"tracks":[{"song":1,"file":"x.ogg","samples":100}]}`,
		"循環終點不在檔尾": `{"sample_rate":44100,"tracks":[{"song":1,"file":"pc98-01.ogg","samples":100,"loops":true,"loop_start_sample":10,"loop_end_sample":90}]}`,
		"循環段是空的":   `{"sample_rate":44100,"tracks":[{"song":1,"file":"pc98-01.ogg","samples":100,"loops":true,"loop_start_sample":100,"loop_end_sample":100}]}`,
		"重複":       `{"sample_rate":44100,"tracks":[{"song":1,"file":"pc98-01.ogg","samples":1},{"song":1,"file":"pc98-01.ogg","samples":1}]}`,
		"取樣率":      `{"sample_rate":0,"tracks":[]}`,
	} {
		if _, err := ParseManifest([]byte(raw)); err == nil {
			t.Errorf("%s：沒有報錯", name)
		}
	}
}
