// Command pool-pc98-music 把 PC-98 版 `MSCDRV.EXE` 的 15 首曲子渲染成可以無縫
// 循環的 WAV，並寫出循環點清單（spec 169）。
//
//	tools/go.sh run ./cmd/pool-pc98-music -driver workplace/pc98-music/MSCDRV.EXE \
//	    -out workplace/pc98-music/wav
//
// 一般用 `tools/build-pc98-music.sh`，它接著在容器裡轉成 OGG。
//
// **輸出的形狀**：會循環的曲子寫「前奏＋兩輪」，循環段取**第二輪**
// （`loop_start_sample`..`loop_end_sample`，檔尾就是循環終點）。
// 只寫一輪的話，接回循環起點那一刻聽到的是「前奏結尾的殘響」而不是
// 「上一輪結尾的殘響」，FM 的釋音被切掉會喀一聲；第二輪的開頭帶的正是
// 上一輪的殘響，所以接縫逐樣本連續。不循環的曲子照原長，後面補釋音。
//
// 原版驅動與輸出的音訊都是第三方著作權（Pony Canyon 1989），不進版控。
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/pc98music"
	"github.com/wicanr2/golden-box-remake-engine/audio/pc98mscdrv"
	"github.com/wicanr2/golden-box-remake-engine/audio/pcm"
	"github.com/wicanr2/golden-box-remake-engine/audio/ym2203/ymfm"
)

// 原版 `MSCDRV.EXE` 的版面（pc98golem 的 docs/spec/005-pool-pc98-music.md，本 repo 的 spec 169）：15 首、每首 6 個聲道
// （3 FM ＋ 3 SSG）。偵測出來的版面對不上就停，不渲染一份可能錯位的東西。
const (
	wantTracks   = 15
	wantChannels = 6
	wantFM       = 3
)

// releaseSeconds 是不循環的曲子在最後一個事件之後再產生的長度，
// 讓最後幾個音的釋音自然收掉（`RenderTrack` 在最後一個事件就停）。
const releaseSeconds = 1.5

// verifySeam 為真時多渲染一輪，量接縫前後一秒的最大樣本差（-verify-seam）。
var (
	verifySeam bool
	seamReport = map[int][4]float64{}
)

func main() {
	driverPath := flag.String("driver", "", "MSCDRV.EXE（PC-98 版開機碟上的那一份）")
	outDir := flag.String("out", ".", "WAV 與 loops.json 的輸出目錄")
	rate := flag.Int("rate", 44100, "輸出取樣率")
	only := flag.Int("track", 0, "只渲染這一首（1 起算），0 代表全部")
	flag.BoolVar(&verifySeam, "verify-seam", false, "多渲染一輪，量第二輪與第三輪開頭一秒的最大樣本差")
	flag.Parse()
	if *driverPath == "" {
		fmt.Fprintln(os.Stderr, "用法：pool-pc98-music -driver MSCDRV.EXE -out 目錄")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*driverPath)
	if err != nil {
		fail(err)
	}
	digest := sha256.Sum256(raw)
	image, err := loadImage(raw)
	if err != nil {
		fail(err)
	}
	layout, err := pc98mscdrv.Detect(image, pc98mscdrv.DetectOptions{})
	if err != nil {
		fail(err)
	}
	if layout.Tracks != wantTracks || layout.Channels != wantChannels || layout.FMChannels != wantFM {
		fail(fmt.Errorf("版面是 %d 首 × %d 聲道（FM %d），原版是 %d × %d（FM %d）：不是同一份驅動",
			layout.Tracks, layout.Channels, layout.FMChannels, wantTracks, wantChannels, wantFM))
	}
	driver, err := pc98mscdrv.Open(image, layout)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail(err)
	}
	manifest := pc98music.Manifest{
		Source:       "MSCDRV.EXE（PC-98 Pool of Radiance，Pony Canyon 1989）",
		DriverSHA256: hex.EncodeToString(digest[:]),
		SampleRate:   *rate,
	}
	for index := 0; index < layout.Tracks; index++ {
		if *only != 0 && index != *only-1 {
			continue
		}
		track, err := renderOne(driver, index, *outDir, *rate)
		if err != nil {
			fail(fmt.Errorf("第 %d 首：%w", index+1, err))
		}
		manifest.Tracks = append(manifest.Tracks, track)
		loop := "不循環"
		if track.Loops {
			loop = fmt.Sprintf("循環 %.3f–%.3f 秒", track.LoopStartSeconds(*rate), track.Seconds(*rate))
		}
		fmt.Printf("  第 %2d 首 → %s：%.3f 秒，%s，peak %.1f dBFS，RMS %.1f dBFS，削波 %d\n",
			track.Song, track.File, track.Seconds(*rate), loop, track.PeakDBFS, track.RMSDBFS, track.Clipped)
		if seam, ok := seamReport[index]; ok {
			fmt.Printf("        接縫：第二輪與第三輪開頭一秒，最像的位移 %+d 樣本，平均差／平均幅度 %.3f；"+
				"三秒包絡最像的位移 %+d 格（10 ms），相關 %.4f\n",
				int(seam[0]), seam[1], int(seam[2]), seam[3])
		}
		if track.Clipped > 0 {
			fail(fmt.Errorf("第 %d 首削波 %d 個樣本：增益太大", track.Song, track.Clipped))
		}
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, pc98music.ManifestFile), append(encoded, '\n'), 0o644); err != nil {
		fail(err)
	}
}

func renderOne(driver *pc98mscdrv.Driver, index int, outDir string, rate int) (pc98music.Track, error) {
	song, err := pc98music.AnalyzeSong(driver, index)
	if err != nil {
		return pc98music.Track{}, err
	}
	synth, err := ymfm.New(pc98mscdrv.PC98YM2203ClockHz)
	if err != nil {
		return pc98music.Track{}, err
	}
	defer synth.Close()
	native := float64(synth.NativeSampleRate())
	options := pc98mscdrv.RenderOptions{}
	timeline, err := pc98music.NewTimeline(driver, index, options, native)
	if err != nil {
		return pc98music.Track{}, err
	}

	// 原生取樣率下要渲染到哪一個樣本。
	var loopStart, stop int
	if song.Loops {
		loopStart = timeline.SampleAt(song.StartTick + song.PeriodTick)
		stop = timeline.SampleAt(song.StartTick + 2*song.PeriodTick)
	} else {
		stop = timeline.SampleAt(song.EndTick)
	}
	options.MaxSeconds = float64(stop)/native + 1
	if verifySeam {
		for channel, loop := range song.Channels {
			fmt.Printf("        聲道 %d：循環 %v，前奏 %d tick（%d 區塊），週期 %d tick（%d 區塊），結束 %d\n",
				channel, loop.Loops, loop.IntroTick, loop.IntroBlock, loop.PeriodTick, loop.PeriodBlocks, loop.EndTick)
		}
	}
	if song.Loops && verifySeam {
		// 多渲染一輪：第三輪的開頭必須與第二輪的開頭逐樣本相同，
		// 否則「接回第二輪開頭」就不是無縫的。
		third := timeline.SampleAt(song.StartTick + 3*song.PeriodTick)
		options.MaxSeconds = float64(third)/native + 1
	}
	samples, report, err := driver.RenderTrack(index, synth, options)
	if err == nil && song.Loops && verifySeam {
		window := int(native)
		if loopStart+window > stop {
			window = stop - loopStart
		}
		// 量兩件事：對齊誤差（在 ±200 個樣本內哪一個位移最像）與
		// 最佳位移下的平均差對訊號平均幅度的比例。
		bestShift, bestSum := 0, math.MaxFloat64
		var energy float64
		for shift := -200; shift <= 200; shift++ {
			var sum float64
			for offset := 0; offset < window; offset++ {
				at := stop + offset + shift
				if at < 0 || at >= len(samples) {
					continue
				}
				sum += math.Abs(float64(samples[loopStart+offset]) - float64(samples[at]))
				if shift == 0 {
					energy += math.Abs(float64(samples[loopStart+offset]))
				}
			}
			if sum < bestSum {
				bestShift, bestSum = shift, sum
			}
		}
		seamReport[index] = [4]float64{float64(bestShift), bestSum / math.Max(energy, 1)}
		// 包絡（每 10 毫秒的 RMS）對得上，代表時間軸對、差的只是振盪器相位；
		// 包絡也對不上才是循環點找錯。
		frame := int(native / 100)
		envelope := func(from, frames int) []float64 {
			out := make([]float64, frames)
			for f := 0; f < frames; f++ {
				var sum float64
				for i := 0; i < frame; i++ {
					at := from + f*frame + i
					if at >= 0 && at < len(samples) {
						sum += float64(samples[at]) * float64(samples[at])
					}
				}
				out[f] = math.Sqrt(sum / float64(frame))
			}
			return out
		}
		frames := 300
		reference := envelope(loopStart, frames)
		bestFrame, bestCorrelation := 0, -2.0
		for shift := -50; shift <= 50; shift++ {
			candidate := envelope(stop+shift*frame, frames)
			if c := correlation(reference, candidate); c > bestCorrelation {
				bestFrame, bestCorrelation = shift, c
			}
		}
		report := seamReport[index]
		report[2], report[3] = float64(bestFrame), bestCorrelation
		seamReport[index] = report
	}
	if err != nil {
		return pc98music.Track{}, err
	}
	if len(report.Unknown) > 0 {
		return pc98music.Track{}, fmt.Errorf("展開時遇到未解命令 %v", report.Unknown)
	}
	if err != nil {
		return pc98music.Track{}, err
	}
	if song.Loops {
		if len(samples) < stop {
			return pc98music.Track{}, fmt.Errorf("渲染只有 %d 個樣本，循環終點在 %d", len(samples), stop)
		}
		samples = samples[:stop]
	} else {
		if report.Truncated {
			return pc98music.Track{}, fmt.Errorf("不循環的曲子被上限截斷")
		}
		tail, err := synth.Generate(int(releaseSeconds * native))
		if err != nil {
			return pc98music.Track{}, err
		}
		samples = append(samples, tail...)
	}

	resampler, err := pcm.NewLinearResampler(uint64(native), uint64(rate))
	if err != nil {
		return pc98music.Track{}, err
	}
	output, err := resampler.Append(nil, samples)
	if err != nil {
		return pc98music.Track{}, err
	}
	pcm16, clipped := scale(output)
	track := pc98music.Track{
		Song:    index + 1,
		File:    pc98music.TrackFile(index + 1),
		Samples: len(pcm16),
		Loops:   song.Loops,
		Clipped: clipped,
	}
	if song.Loops {
		track.LoopStartSample = int(math.Round(float64(loopStart) * float64(rate) / native))
		track.LoopEndSample = len(pcm16)
		track.IntroTicks, track.PeriodTicks = song.StartTick, song.PeriodTick
	}
	track.PeakDBFS, track.RMSDBFS = levels(pcm16)
	path := filepath.Join(outDir, fmt.Sprintf("pc98-%02d.wav", index+1))
	if err := writeWAV(path, pcm16, rate); err != nil {
		return pc98music.Track{}, err
	}
	return track, nil
}

// gainDivisor 是 ymfm 輸出換成 16 位元的除數。
//
// engine 的 `cmd/pc98-render-music` 除以 4，理由是 FM 加三路 SSG 疊起來可能溢位；
// 實際量過這 15 首在 ÷4 下最響的一首峰值 −7.7 dBFS、多數 RMS 在 −30 dBFS 以下，
// 放進遊戲裡明顯比 Amiga 那六首（RMS 約 −14 dBFS）小聲。改成 ÷2 是**同一個增益
// 套在全部 15 首**（各首的相對音量照原版的混音），峰值仍低於 −1.7 dBFS；
// 削波一個樣本就讓工具失敗，所以這個數字不會悄悄變成失真。
const gainDivisor = 2

// scale 把 ymfm 的輸出換成 16 位元，回傳削波的樣本數。
func scale(samples []int32) ([]int16, int) {
	out := make([]int16, len(samples))
	clipped := 0
	for index, sample := range samples {
		value := int64(sample) / gainDivisor
		if value > 32767 {
			value, clipped = 32767, clipped+1
		} else if value < -32768 {
			value, clipped = -32768, clipped+1
		}
		out[index] = int16(value)
	}
	return out, clipped
}

func correlation(a, b []float64) float64 {
	var meanA, meanB float64
	for index := range a {
		meanA += a[index]
		meanB += b[index]
	}
	meanA /= float64(len(a))
	meanB /= float64(len(b))
	var cov, varA, varB float64
	for index := range a {
		x, y := a[index]-meanA, b[index]-meanB
		cov += x * y
		varA += x * x
		varB += y * y
	}
	if varA == 0 || varB == 0 {
		return 1
	}
	return cov / math.Sqrt(varA*varB)
}

func levels(samples []int16) (peak, rms float64) {
	var maximum float64
	var sum float64
	for _, sample := range samples {
		value := math.Abs(float64(sample)) / 32768
		if value > maximum {
			maximum = value
		}
		sum += value * value
	}
	toDB := func(value float64) float64 {
		if value <= 0 {
			return -120
		}
		return math.Round(20*math.Log10(value)*10) / 10
	}
	if len(samples) == 0 {
		return -120, -120
	}
	return toDB(maximum), toDB(math.Sqrt(sum / float64(len(samples))))
}

func writeWAV(path string, samples []int16, rate int) error {
	body := make([]byte, len(samples)*2)
	for index, sample := range samples {
		binary.LittleEndian.PutUint16(body[index*2:], uint16(sample))
	}
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+len(body)))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:], uint32(rate*2))
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(body)))
	return os.WriteFile(path, append(header, body...), 0o644)
}

// loadImage 去掉 MZ 標頭，取出載入映像。
func loadImage(raw []byte) ([]byte, error) {
	if len(raw) < 0x20 || (string(raw[:2]) != "MZ" && string(raw[:2]) != "ZM") {
		return nil, fmt.Errorf("不是 MZ 執行檔")
	}
	header := int(binary.LittleEndian.Uint16(raw[8:10])) * 16
	if header <= 0 || header >= len(raw) {
		return nil, fmt.Errorf("MZ 標頭大小 %d 不合理", header)
	}
	return raw[header:], nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
