package music

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// 對拍：發行包裡的 OGG 解回來要和 UADE 渲染出來的 WAV 是同一段音訊。
//
// Vorbis 是有損的，所以比的不是位元組，是**同一時間軸上的相關係數**——
// 兩段一樣的音訊逐樣本相關會接近 1，兩段不同的（例如接錯 subsong）會接近 0。
// 比 RMS 或長度都擋不住「檔名對、內容接錯」這種錯法，相關係數擋得住。
//
// 素材不在 repo 裡（第三方著作權，spec 128），所以沒有就跳過。
func TestOGGMatchesTheRenderedSource(t *testing.T) {
	root := filepath.Join("..", "..", "workplace", "amiga-music")
	if _, err := os.Stat(filepath.Join(root, "ogg")); err != nil {
		t.Skip("workplace/amiga-music 不在（音訊資產不進 repo）")
	}
	for _, track := range Catalog() {
		oggPath := filepath.Join(root, "ogg", track.File)
		wavPath := filepath.Join(root, "wav", "por-amiga-sub"+strconv.Itoa(track.Subsong)+".wav")
		ogg, err := decodeOGG(oggPath)
		if err != nil {
			t.Fatalf("%s：%v", track.File, err)
		}
		wav, err := readWAV(wavPath)
		if err != nil {
			t.Fatalf("%s：%v", wavPath, err)
		}
		// 兩邊都是 44.1 kHz 立體聲 s16。比前 30 秒就夠——接錯 subsong
		// 在頭幾秒就分得出來，而整段比會讓測試慢得沒必要。
		limit := 30 * SampleRate * 2
		if limit > len(ogg) {
			limit = len(ogg)
		}
		if limit > len(wav) {
			limit = len(wav)
		}
		if limit < SampleRate {
			t.Fatalf("%s 只解出 %d 個樣本，太短", track.File, limit)
		}
		got := correlation(ogg[:limit], wav[:limit])
		if got < 0.95 {
			t.Errorf("%s 與 %s 的相關係數只有 %.3f：不是同一段音訊",
				track.File, filepath.Base(wavPath), got)
		}
		// 長度也要對得上目錄裡記的秒數（Vorbis 會有幾毫秒的差）。
		seconds := float64(len(ogg)) / 2 / SampleRate
		if math.Abs(seconds-track.Seconds) > 1.0 {
			t.Errorf("%s 解出來 %.1f 秒，目錄記的是 %.1f 秒",
				track.File, seconds, track.Seconds)
		}
	}
}

// 交叉核對：每一首都要和**自己**最像。拿第 1 首去比第 6 首要明顯低於 0.95，
// 否則上面那個門檻其實什麼都沒擋到（負對照）。
func TestDifferentSubsongsDoNotCorrelate(t *testing.T) {
	root := filepath.Join("..", "..", "workplace", "amiga-music")
	if _, err := os.Stat(filepath.Join(root, "ogg")); err != nil {
		t.Skip("workplace/amiga-music 不在（音訊資產不進 repo）")
	}
	first, err := decodeOGG(filepath.Join(root, "ogg", "por-amiga-01.ogg"))
	if err != nil {
		t.Fatal(err)
	}
	sixth, err := decodeOGG(filepath.Join(root, "ogg", "por-amiga-06.ogg"))
	if err != nil {
		t.Fatal(err)
	}
	limit := 30 * SampleRate * 2
	if limit > len(sixth) {
		limit = len(sixth)
	}
	if got := correlation(first[:limit], sixth[:limit]); got >= 0.95 {
		t.Errorf("第 1 首與第 6 首的相關係數是 %.3f：門檻擋不住接錯 subsong", got)
	}
}

func decodeOGG(path string) ([]int16, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	stream, err := vorbis.DecodeWithSampleRate(SampleRate, newByteReader(raw))
	if err != nil {
		return nil, err
	}
	pcm := make([]byte, stream.Length())
	if _, err := readFull(stream, pcm); err != nil {
		return nil, err
	}
	return toSamples(pcm), nil
}

func readFull(stream interface{ Read([]byte) (int, error) }, out []byte) (int, error) {
	total := 0
	for total < len(out) {
		n, err := stream.Read(out[total:])
		total += n
		if err != nil || n == 0 {
			return total, nil
		}
	}
	return total, nil
}

// readWAV 直接找 data chunk：UADE 寫的是 WAVE_FORMAT_EXTENSIBLE，
// 標準函式庫的 wave 讀不動。
func readWAV(path string) ([]int16, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	offset := 12
	for offset+8 <= len(raw) {
		size := int(binary.LittleEndian.Uint32(raw[offset+4 : offset+8]))
		if string(raw[offset:offset+4]) == "data" {
			end := offset + 8 + size
			if end > len(raw) {
				end = len(raw)
			}
			return toSamples(raw[offset+8 : end]), nil
		}
		offset += 8 + size + size&1
	}
	return nil, os.ErrNotExist
}

func toSamples(raw []byte) []int16 {
	out := make([]int16, len(raw)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(raw[i*2 : i*2+2]))
	}
	return out
}

func correlation(a, b []int16) float64 {
	var sumAB, sumAA, sumBB float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		sumAB += x * y
		sumAA += x * x
		sumBB += y * y
	}
	if sumAA == 0 || sumBB == 0 {
		return 0
	}
	return sumAB / math.Sqrt(sumAA*sumBB)
}
