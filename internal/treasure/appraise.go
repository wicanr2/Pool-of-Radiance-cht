package treasure

import "fmt"

// 寶石與珠寶的估價與販賣（overlay-21 entry 19，`19F7h`）。
//
// 商店與神殿共用同一個 A）ppraise（說明書 p.34：「進入神殿後，各種處理錢財的
// 指令都與在商店中的一樣」）。畫面上的字串是 `No gems or jewelry`、
// `You have a fine collection of:`、`Appraise : `、`The gem is valued at `、
// `The jewel is valued at `、` gp.`、`Sell Keep`。

// GemValueBands 是寶石的價值表（`1CB1h`..`1D2Bh`）：擲 1d100 查一格，
// 值是固定的金幣數。這就是 AD&D 一版的寶石表。
var GemValueBands = []struct {
	Low, High, Value int
}{
	{1, 25, 10},
	{26, 50, 50},
	{51, 70, 100},
	{71, 90, 500},
	{91, 99, 1000},
	{100, 100, 5000},
}

// JewelryValueBands 是珠寶的價值表（`1F40h`..`2084h`）：擲 1d100 選一段，
// 再在那一段裡 `Random(Range) + Base`，所以值域是 `Base .. Base+Range-1`。
var JewelryValueBands = []struct {
	Low, High, Range, Base int
}{
	{1, 10, 900, 100},
	{11, 20, 1000, 200},
	{21, 40, 1500, 300},
	{41, 50, 2500, 500},
	{51, 70, 5000, 1000},
	{71, 90, 6000, 2000},
	{91, 100, 10000, 2000},
}

// GoldPerPlatinum 是原版在 `1F0Dh` 除的那個 5。估價是**金幣**，而賣得的錢
// 進的是角色記錄 `+90h`——依 spec 040 的七欄版面（`+88h + 2 × 索引`）那是
// **白金**那一欄。AD&D 一版 1 白金 ＝ 5 金，所以那個除以 5 是**幣別換算**，
// 不是折價：賣掉拿的是估價的全額，只是換成白金付。
//
// 同一支常式減的 `+92h`／`+94h` 也落在那張版面上（索引 5 寶石、6 珠寶），
// 三個欄位一起自洽。
const GoldPerPlatinum = 5

// GemValue 依 1d100 的點數查出一顆寶石值多少金幣。
func GemValue(roll int) (int, error) {
	for _, band := range GemValueBands {
		if roll >= band.Low && roll <= band.High {
			return band.Value, nil
		}
	}
	return 0, fmt.Errorf("Pool gem roll %d is outside 1..100", roll)
}

// JewelryValue 依 1d100 的點數與一次 `Random(N)` 算出一件珠寶值多少金幣。
// random 要與 Turbo Pascal 的 `Random(N)` 同語意：回 0..N−1。
func JewelryValue(roll int, random func(limit int) int) (int, error) {
	for _, band := range JewelryValueBands {
		if roll >= band.Low && roll <= band.High {
			offset := 0
			if random != nil {
				offset = random(band.Range)
			}
			if offset < 0 {
				offset = 0
			}
			if offset >= band.Range {
				offset = band.Range - 1
			}
			return band.Base + offset, nil
		}
	}
	return 0, fmt.Errorf("Pool jewelry roll %d is outside 1..100", roll)
}

// SellPrice 是賣掉一件估好價的東西實際拿到的**白金**（`1F07h` 的 `div 5`）。
func SellPrice(goldValue int) int { return goldValue / GoldPerPlatinum }

// KeptItemType 與 KeptItemSubtype 是「留著」時原版建出來的物品記錄欄位
//（`1E80h` 的 `+2Eh = 46h`、`1E65h` 的 `+31h = 65h`），估好的價值寫在 `+3Ah`。
const (
	KeptItemType    uint8 = 0x46
	KeptItemSubtype uint8 = 0x65
)

// KeepNeedsRoom 是「留著」那一項出不出得來的門檻：原版在 `1DA7h` 比
// 記錄 `+C7h`（帶著幾件物品）是不是已經到 10h。滿了就只剩 `Sell`。
const KeepNeedsRoom = 0x10
