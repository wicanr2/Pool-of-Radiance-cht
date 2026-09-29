// Package pc98music 把 PC-98 版（Pony Canyon 1989）`MSCDRV.EXE` 裡的 15 首
// YM2203 曲子整理成 remake 可以無縫循環播放的形狀（spec 169）。
//
// 解析與合成都在共用 engine 的 `audio/pc98mscdrv`；這裡只做兩件 engine
// 沒有的事：
//
//  1. **找循環點**：驅動的串流用 `82`（跳）、`85`／`86`（迴圈）與旗標分歧
//     讓曲子一直繞下去。engine 的 `ExpandChannel` 會把它展開到上限為止，
//     這裡從展開結果的**區塊序列**找出「從第幾個區塊開始、每幾個區塊重複一次」。
//  2. **把 tick 換成樣本位置**：照 `RenderTrack` 的排程規則（每段間隔四捨五入
//     成原生樣本數）重算一次，循環點才能對到渲染出來的那一個樣本。
//
// 原版驅動與渲染出來的音訊都是第三方著作權，不進版控。
package pc98music

import (
	"fmt"
	"math"
	"sort"

	"github.com/wicanr2/golden-box-remake-engine/audio/pc98mscdrv"
)

// ChannelLoop 是一個聲道的循環形狀，單位是音序 tick。
type ChannelLoop struct {
	// Loops 為真代表這個聲道會一直繞下去（展開碰到上限才停）。
	Loops bool
	// IntroTick 是循環段開始的 tick；PeriodTick 是一輪的長度。
	IntroTick, PeriodTick uint64
	// IntroBlock、PeriodBlocks 是同一件事的區塊計數，方便回頭對位元組。
	IntroBlock, PeriodBlocks int
	// EndTick 是不循環的聲道最後一個音（或休止）結束的 tick。
	EndTick uint64
}

// minimumRepeats 是判定循環時至少要看到的完整輪數。
//
// 只看到一輪半就下結論，會把「A 段重複兩次之後接 B 段」誤判成以 A 為週期。
// 要求循環段在觀察範圍內完整出現三次，而且**尾端三分之二**全部符合同一個週期。
const minimumRepeats = 3

// AnalyzeChannel 從一個聲道的展開結果找循環點。
//
// 判準只用區塊偏移序列：同一串區塊交出去，音高、時值、速度、音量都一樣
// （那些全在區塊裡）。旗標分歧會反映在區塊序列上，所以不必另外追旗標。
func AnalyzeChannel(result pc98mscdrv.ExpandResult) (ChannelLoop, error) {
	offsets, starts, end := blockTimeline(result.Events)
	if !result.Truncated {
		return ChannelLoop{EndTick: end}, nil
	}
	count := len(offsets)
	if count < minimumRepeats*2 {
		return ChannelLoop{}, fmt.Errorf("展開只有 %d 個區塊就碰到上限，判定不了循環", count)
	}
	tailStart := count / 3
	period := 0
	for candidate := 1; candidate*minimumRepeats <= count; candidate++ {
		matched := true
		for index := tailStart; index+candidate < count; index++ {
			if offsets[index] != offsets[index+candidate] {
				matched = false
				break
			}
		}
		if matched {
			period = candidate
			break
		}
	}
	if period == 0 {
		return ChannelLoop{}, fmt.Errorf("%d 個區塊裡找不到重複的週期", count)
	}
	intro := tailStart
	for intro > 0 && offsets[intro-1] == offsets[intro-1+period] {
		intro--
	}
	if intro+minimumRepeats*period > count {
		return ChannelLoop{}, fmt.Errorf("週期 %d 個區塊在 %d 個區塊裡不到 %d 輪",
			period, count, minimumRepeats)
	}
	loop := ChannelLoop{
		Loops: true, IntroBlock: intro, PeriodBlocks: period,
		IntroTick: starts[intro], PeriodTick: starts[intro+period] - starts[intro],
	}
	if loop.PeriodTick == 0 {
		return ChannelLoop{}, fmt.Errorf("循環段 %d 個區塊但長度 0 tick", period)
	}
	return loop, nil
}

// blockTimeline 回傳每一個交出去的區塊的偏移與起始 tick，以及最後的結束 tick。
//
// 區塊裡沒有音符或休止時 tick 不前進，起始 tick 沿用上一個區塊的結尾。
func blockTimeline(events []pc98mscdrv.Event) (offsets []int, starts []uint64, end uint64) {
	current := -1
	var tick uint64
	for _, event := range events {
		for event.BlockIndex > current {
			current++
			offsets = append(offsets, event.BlockOffset)
			starts = append(starts, tick)
		}
		if event.Kind == pc98mscdrv.EventNote || event.Kind == pc98mscdrv.EventRest {
			tick = event.Tick + uint64(event.Duration)
		}
	}
	return offsets, starts, tick
}

// SongLoop 是一首曲子的循環形狀。
type SongLoop struct {
	// Loops 為假代表整首會自然結束（所有聲道都跑到 `81` 或串流尾）。
	Loops bool
	// StartTick、PeriodTick：循環段從 StartTick 開始，每 PeriodTick 重複一次。
	StartTick, PeriodTick uint64
	// EndTick 是不循環的曲子的結尾。
	EndTick  uint64
	Channels []ChannelLoop
}

// maxPeriodRatio 是合成週期（各聲道週期的最小公倍數）最多可以是最長聲道週期的幾倍。
// 超過就代表各聲道的週期彼此對不上，那不是一般曲子的寫法，停下來查。
const maxPeriodRatio = 4

// AnalyzeSong 對一首曲子的每一個聲道找循環，合成整首的循環點。
func AnalyzeSong(driver *pc98mscdrv.Driver, track int) (SongLoop, error) {
	layout := driver.Layout()
	song := SongLoop{}
	var period, longest, start, stopped uint64
	for channel := 0; channel < layout.Channels; channel++ {
		result, err := driver.ExpandChannel(track, channel, pc98mscdrv.ExpandOptions{})
		if err != nil {
			return song, err
		}
		loop, err := AnalyzeChannel(result)
		if err != nil {
			return song, fmt.Errorf("第 %d 首聲道 %d：%w", track+1, channel, err)
		}
		song.Channels = append(song.Channels, loop)
		if !loop.Loops {
			if loop.EndTick > stopped {
				stopped = loop.EndTick
			}
			continue
		}
		if period == 0 {
			period = loop.PeriodTick
		} else {
			period = lcm(period, loop.PeriodTick)
		}
		if loop.PeriodTick > longest {
			longest = loop.PeriodTick
		}
		if loop.IntroTick > start {
			start = loop.IntroTick
		}
	}
	if period == 0 {
		song.EndTick = stopped
		return song, nil
	}
	if period > longest*maxPeriodRatio {
		return song, fmt.Errorf("第 %d 首各聲道的週期合不起來（最小公倍數 %d tick，最長聲道 %d）",
			track+1, period, longest)
	}
	// 不循環的聲道要先結束，循環段才每一輪都一樣。
	if stopped > start {
		start = stopped
	}
	song.Loops, song.StartTick, song.PeriodTick = true, start, period
	return song, nil
}

func lcm(a, b uint64) uint64 {
	x, y := a, b
	for y != 0 {
		x, y = y, x%y
	}
	return a / x * b
}

// Timeline 把 tick 換成原生樣本位置，照 `RenderTrack` 的排程規則重算。
//
// **為什麼要重算而不是用秒數乘取樣率**：`RenderTrack` 每一段間隔各自四捨五入成
// 樣本數，幾千段累積下來會與「總秒數 × 取樣率」差上幾百個樣本——循環點差
// 那麼多就不再無縫。照同一套規則逐段累加，得到的是渲染結果裡真正的那一個樣本。
type Timeline struct {
	ticks   []uint64
	samples []int
	tempos  []byte
	options pc98mscdrv.RenderOptions
	rate    float64
}

// NewTimeline 建一首曲子的時間軸。options 要與渲染時傳給 `RenderTrack` 的相同，
// rate 是音源的原生取樣率。
func NewTimeline(driver *pc98mscdrv.Driver, track int, options pc98mscdrv.RenderOptions, rate float64) (*Timeline, error) {
	applyRenderDefaults(&options)
	type item struct {
		tick  uint64
		order int
		tempo bool
		value byte
	}
	var schedule []item
	order := 0
	for channel := 0; channel < driver.Layout().Channels; channel++ {
		result, err := driver.ExpandChannel(track, channel, pc98mscdrv.ExpandOptions{Flags: options.Flags})
		if err != nil {
			return nil, err
		}
		gate := options.DefaultGate
		for _, event := range result.Events {
			if event.Kind == pc98mscdrv.EventGate {
				gate = event.Value
			}
			schedule = append(schedule, item{
				tick: event.Tick, order: order,
				tempo: event.Kind == pc98mscdrv.EventTempo, value: event.Value,
			})
			order++
			if event.Kind == pc98mscdrv.EventNote {
				sounding := uint64(event.Duration) * uint64(gate) / 8
				if sounding == 0 {
					sounding = uint64(event.Duration)
				}
				schedule = append(schedule, item{tick: event.Tick + sounding, order: order})
				order++
			}
		}
	}
	sort.SliceStable(schedule, func(i, j int) bool {
		if schedule[i].tick != schedule[j].tick {
			return schedule[i].tick < schedule[j].tick
		}
		return schedule[i].order < schedule[j].order
	})
	timeline := &Timeline{options: options, rate: rate}
	tempo := options.DefaultTempo
	var last uint64
	total := 0
	timeline.ticks = append(timeline.ticks, 0)
	timeline.samples = append(timeline.samples, 0)
	timeline.tempos = append(timeline.tempos, tempo)
	for _, entry := range schedule {
		if entry.tick > last {
			total += int(math.Round(secondsPerTick(tempo, options) * float64(entry.tick-last) * rate))
			last = entry.tick
			timeline.ticks = append(timeline.ticks, last)
			timeline.samples = append(timeline.samples, total)
			timeline.tempos = append(timeline.tempos, tempo)
		}
		if entry.tempo {
			tempo = entry.value
			// 同一個 tick 上改了速度：之後那一段用新速度。
			timeline.tempos[len(timeline.tempos)-1] = tempo
		}
	}
	return timeline, nil
}

// SampleAt 回傳 tick 對應的原生樣本位置。落在兩個排程點之間時照那一段的速度內插
// （渲染也是在那一段裡連續產生樣本，誤差在一個樣本以內）。
func (t *Timeline) SampleAt(tick uint64) int {
	index := sort.Search(len(t.ticks), func(i int) bool { return t.ticks[i] > tick }) - 1
	if index < 0 {
		index = 0
	}
	base := t.samples[index]
	if t.ticks[index] == tick {
		return base
	}
	return base + int(math.Round(secondsPerTick(t.tempos[index], t.options)*float64(tick-t.ticks[index])*t.rate))
}

// LastTick 是排程的最後一個 tick。
func (t *Timeline) LastTick() uint64 { return t.ticks[len(t.ticks)-1] }

func secondsPerTick(tempo byte, options pc98mscdrv.RenderOptions) float64 {
	if tempo == 0 {
		tempo = options.DefaultTempo
	}
	return 60.0 / (float64(tempo) * float64(options.TicksPerQuarter))
}

// applyRenderDefaults 與 `RenderOptions.applyDefaults` 相同的預設（那一支沒有匯出）。
// 只補時間軸用得到的三個欄位。
func applyRenderDefaults(options *pc98mscdrv.RenderOptions) {
	if options.TicksPerQuarter == 0 {
		options.TicksPerQuarter = pc98mscdrv.DefaultTicksPerQuarter
	}
	if options.DefaultTempo == 0 {
		options.DefaultTempo = 120
	}
	if options.DefaultGate == 0 {
		options.DefaultGate = 7
	}
}
