package gamepack

import (
	"encoding/binary"
	"fmt"
)

// 遊戲時鐘與休息時間（overlay-20）。
//
// 時間是一個**逐位進位的數**，每一位一個 word，進位上限逐位不同，整張表在
// `DS:35D4h`。紮營要玩家挑的天／時／分就是這個數的其中三位。

const (
	// TimeRadixOffset 是進位上限表在 START.EXE 裡的 DS 位移（`DS:35D4h`）。
	TimeRadixOffset = 0x35D4
	// TimeDigits 是時間有幾位。overlay-20 `02B1h` 的迴圈跑 0..6。
	TimeDigits = 7

	// 各位的意義。索引就是 `DS:6CB6h + 索引 × 2` 那個陣列的索引。
	TimeDigitTick        = 0 // 比分更細的一格
	TimeDigitMinuteOnes  = 1
	TimeDigitMinuteTens  = 2
	TimeDigitHour        = 3
	TimeDigitDay         = 4
	TimeDigitMonth       = 5
	TimeDigitYear        = 6

	// RestMinuteStep 是紮營調整分鐘時的級距（overlay-20 `07BEh` 加 5、
	// `07E1h` 減 5，都作用在分的個位上）。
	RestMinuteStep = 5
	// RestMaxDays 是紮營時間的天數上限（`0388h` 夾到 63h）。
	RestMaxDays = 99

	// RestTicksPerHeal 是「休息多久回一點生命力」的刻度數（`083Ah` 比 120h）。
	// 一刻是五分鐘，所以 288 刻正好二十四小時——與說明書 p.29
	//「每休息二十四小時各隊員可恢復一點 HP」相同。
	RestTicksPerHeal = 0x120
	// RestTicksPerHour 是一小時的刻度數（`0B72h` 比 0Ch）。
	RestTicksPerHour = 0x0C
	// RestMinutesPerTick 由上面兩個推出來：一小時 12 刻，所以一刻五分鐘。
	RestMinutesPerTick = 60 / RestTicksPerHour
)

// TimeRadix 是逐位的進位上限。
type TimeRadix [TimeDigits]int

// ParseTimeRadix 解出七個 word。
func ParseTimeRadix(raw []byte) (TimeRadix, error) {
	var radix TimeRadix
	if len(raw) < TimeDigits*2 {
		return radix, fmt.Errorf("Pool time radix has %d bytes, want %d", len(raw), TimeDigits*2)
	}
	for index := range radix {
		radix[index] = int(binary.LittleEndian.Uint16(raw[index*2:]))
	}
	return radix, nil
}

// ReadDOSTimeRadix 從 DOS ZIP 的 START.EXE 讀出進位上限表。
func ReadDOSTimeRadix(zipPath string) (TimeRadix, error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return TimeRadix{}, err
	}
	start := TimeRadixOffset + startDataSegmentFileDelta
	end := start + TimeDigits*2
	if len(raw) < end {
		return TimeRadix{}, fmt.Errorf("START.EXE is %d bytes, the time radix needs %d", len(raw), end)
	}
	return ParseTimeRadix(raw[start:end])
}

// GameTime 是那個逐位的數。
type GameTime [TimeDigits]int

// Normalise 重現 overlay-20 entry 5（`02B1h`）：由低位往高位掃，某一位到達
// 它的上限就進位一次。最高位溢位時原版是讓隊伍每個人的 `+30h`（年齡）加一，
// 那不屬於這個數本身，所以由回傳值告訴呼叫端發生了幾次。
//
// 原版每一位只進位一次（不是 while 迴圈），因為它每加一次就正規化一次，
// 一次最多只會超過上限一格。照它接。
func (t GameTime) Normalise(radix TimeRadix) (GameTime, int) {
	years := 0
	for index := 0; index < TimeDigits; index++ {
		if t[index] < radix[index] {
			continue
		}
		if index == TimeDigits-1 {
			years++
			continue
		}
		t[index+1]++
		t[index] -= radix[index]
	}
	return t, years
}

// Minutes 把分的十位與個位合成一個數，就是畫面上那一欄（`0667h`）。
func (t GameTime) Minutes() int {
	return t[TimeDigitMinuteTens]*10 + t[TimeDigitMinuteOnes]
}

// RestDuration 是紮營要玩家挑的那段時間，也就是 `DS:6CB6h` 那個陣列。
type RestDuration struct {
	Time  GameTime
	Radix TimeRadix
}

// NewRestDuration 開一段全零的休息時間。
func NewRestDuration(radix TimeRadix) RestDuration {
	return RestDuration{Radix: radix}
}

// Days、Hours、Minutes 是畫面上的三欄。
func (d RestDuration) Days() int    { return d.Time[TimeDigitDay] }
func (d RestDuration) Hours() int   { return d.Time[TimeDigitHour] }
func (d RestDuration) Minutes() int { return d.Time.Minutes() }

// settle 重現 overlay-20 entry 6（`035Eh`）：先正規化，再把月數折回天數
//（一個月 30 天，就是天那一位的上限），最後把天數夾到 99。
//
// 折回去是因為這是一段**長度**不是日期——沒有「月」這個欄位可以顯示。
func (d RestDuration) settle() RestDuration {
	settled, _ := d.Time.Normalise(d.Radix)
	if settled[TimeDigitMonth] > 0 {
		settled[TimeDigitDay] += d.Radix[TimeDigitDay] * settled[TimeDigitMonth]
		settled[TimeDigitMonth] = 0
	}
	if settled[TimeDigitDay] > RestMaxDays {
		settled[TimeDigitDay] = RestMaxDays
	}
	d.Time = settled
	return d
}

// RestField 是紮營時間的三個可調欄位，編號照原版（`[bp-4]` 的 2..4）。
type RestField int

const (
	RestFieldMinutes RestField = 2
	RestFieldHours   RestField = 3
	RestFieldDays    RestField = 4
)

// NextField、PreviousField 是左右鍵的循環（`074Dh`／`0764h`）。
func (f RestField) NextField() RestField {
	if f+1 > RestFieldDays {
		return RestFieldMinutes
	}
	return f + 1
}

func (f RestField) PreviousField() RestField {
	if f-1 < RestFieldMinutes {
		return RestFieldDays
	}
	return f - 1
}

// Increase 重現 `07B8h`：分鐘加在**個位**上而且一次五分，天與時各加一。
func (d RestDuration) Increase(field RestField) RestDuration {
	if field == RestFieldMinutes {
		d.Time[TimeDigitMinuteOnes] += RestMinuteStep
	} else {
		d.Time[int(field)]++
	}
	return d.settle()
}

// Decrease 重現 `07DBh` 與 entry 7（`0437h`）：整個數已經是零就不動，
// 否則從那一位借位減。分鐘一樣是減在個位上、一次五分。
func (d RestDuration) Decrease(field RestField) RestDuration {
	if d.IsZero() {
		return d
	}
	digit, amount := int(field), 1
	if field == RestFieldMinutes {
		digit, amount = TimeDigitMinuteOnes, RestMinuteStep
	}
	d.Time = d.Time.borrow(digit, amount, d.Radix)
	return d.settle()
}

// IsZero 對應 `043Dh` 那四個比較：天、時、分的十位與個位全是 0。
func (d RestDuration) IsZero() bool {
	return d.Time[TimeDigitDay] == 0 && d.Time[TimeDigitHour] == 0 &&
		d.Time[TimeDigitMinuteTens] == 0 && d.Time[TimeDigitMinuteOnes] == 0
}

// borrow 從指定的位減掉一個量，不夠就往高位借。
func (t GameTime) borrow(digit, amount int, radix TimeRadix) GameTime {
	if t[digit] >= amount {
		t[digit] -= amount
		return t
	}
	for higher := digit + 1; higher < TimeDigits; higher++ {
		if t[higher] == 0 {
			continue
		}
		t[higher]--
		// 借位要逐位往下傳：從第 j+1 位借到的一個，換成第 j 位的 radix[j] 個；
		// 若還要再往下借，就從第 j 位再拿一個出來。少了那個 `-1`，每經過一位
		// 就多出一整個進位單位——症狀是「一小時減五分變成一小時又五分」。
		for between := higher - 1; between >= digit; between-- {
			t[between] += radix[between]
			if between > digit {
				t[between]--
			}
		}
		break
	}
	if t[digit] >= amount {
		t[digit] -= amount
	} else {
		t[digit] = 0
	}
	return t
}

// TotalTicks 把一段休息時間換算成刻度數（一刻五分鐘）。
// 原版是逐刻推進的，這裡先算總量，讓「休息這麼久會發生什麼」算得出來。
func (d RestDuration) TotalTicks() int {
	minutes := d.Days()*24*60 + d.Hours()*60 + d.Minutes()
	return minutes / RestMinutesPerTick
}

// RestHealing 回報這段時間每個人回多少生命力：每 288 刻（二十四小時）一點
//（`0830h`，說明書 p.29）。
func RestHealing(ticks int) int { return ticks / RestTicksPerHeal }
