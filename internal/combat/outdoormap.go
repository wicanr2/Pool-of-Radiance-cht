package combat

import "fmt"

// 室外戰場（overlay-10 `1255h`，spec 060）。
//
// `1255h` 先把整面填成 OpenGroundCellClass，再以大地圖地形碼（`DS:45BCh`）
// 挑一組地形旗標，依序跑四支細節建構器：
//
//	0C7Ah  斜向的帶狀地形（32h／33h，偶有 34h／35h 的兩列）
//	0DB0h  兩格高的直立物（上半 20h..25h、下半 24h..29h）
//	0F3Fh  依旗標權重撒單格雜物（2Ah..3Fh，偶有 40h／41h 的上下兩格）
//	0000h  1% 以下的隨機物件（1Ah..1Dh，圖塊在 RandCom）
//
// 四支都直接寫 `[6674h] + 7 + y×32h + x`，**不經過** `022Eh` 的加一；
// 寫下的就是存進地圖的類別。每一支都只在「這一格的圖塊序號是 16h」時落筆，
// 也就是還是平地的格子——所以後跑的建構器不會蓋掉先跑的。

// OpenGroundPresentation 是平地的圖塊序號。建構器以
// `cmp byte ptr [類別×4 + 275Bh], 16h` 判斷一格還空著；275Bh 是類別表
// `DS:2758h` 每筆的第四個 byte。類別 17h 與 36h 的序號都是 16h。
const OpenGroundPresentation = 0x16

// 地形旗標（overlay-10 `08B4h` 的回傳值）。位元的玩家語意沒有讀到文字證據，
// 所以只用位元值命名，註解寫的是各建構器怎麼用它。
const (
	OutdoorFlag01 uint8 = 0x01 // 四支建構器都不讀它；`@4AB3 == FFh` 時 80h 換成它
	OutdoorFlag02 uint8 = 0x02 // `0DB0h` 密度 1；`0F3Fh` 權重 c=2Dh、d=0Ah
	OutdoorFlag04 uint8 = 0x04 // `0DB0h` 密度 3；`0F3Fh` c=19h
	OutdoorFlag08 uint8 = 0x08 // `0DB0h` 密度 7
	OutdoorFlag10 uint8 = 0x10 // `0C7Ah` 門檻 4Bh（後寫，蓋過 20h）
	OutdoorFlag20 uint8 = 0x20 // `0C7Ah` 門檻 23h
	OutdoorFlag40 uint8 = 0x40 // `0DB0h` 密度 4 並重擲高度；`0F3Fh` d=5、e=1Eh，不重擲 1d4 的 4
	OutdoorFlag80 uint8 = 0x80 // `0DB0h` 密度 0；`0F3Fh` a=0Fh、d=19h、c=0Ah
)

// outdoorFlagRule 是 `08B4h` 的一條比對：code 落在任一區間就回 flags。
type outdoorFlagRule struct {
	ranges [][2]uint8
	flags  uint8
}

// outdoorFlagRules 照 `08B4h` 的比對順序逐條抄下。原版是一串 `cmp`／`jl`／`jle`，
// **先命中的算數**，後面幾條與前面重疊的區間永遠輪不到（例如 3..4 同時出現在
// 第 4 與第 5 條），這裡保留原樣，不去整理。
var outdoorFlagRules = []outdoorFlagRule{
	{[][2]uint8{{0x01, 0x02}, {0x28, 0x29}, {0x4F, 0x56}, {0x5B, 0x5C}, {0x2B, 0x2B}, {0xF1, 0xF1}}, 0x01}, // 08BAh
	{[][2]uint8{{0xA8, 0xB1}}, 0x21},                                                                       // 08F8h
	{[][2]uint8{{0xB9, 0xB9}, {0xBC, 0xC0}}, 0x11},                                                         // 0909h
	{[][2]uint8{{0x03, 0x04}, {0x06, 0x07}, {0x08, 0x09}, {0x13, 0x15}, {0x17, 0x19}}, 0x03},               // 091Fh
	{[][2]uint8{{0x03, 0x04}, {0x07, 0x07}, {0x0A, 0x12}, {0x16, 0x16}, {0x1A, 0x24}, {0xC9, 0xD0}}, 0x02}, // 0958h
	{[][2]uint8{{0xD1, 0xD5}}, 0x12},                                                                       // 0991h
	{[][2]uint8{{0x25, 0x25}, {0x39, 0x39}, {0x40, 0x40}, {0x9A, 0x9E}, {0xA4, 0xA5}, {0xB4, 0xB4},
		{0xF3, 0xF3}, {0xEF, 0xF0}, {0xFA, 0xFA}, {0xFC, 0xFC}}, 0x04}, // 09A2h
	{[][2]uint8{{0x27, 0x2C}, {0x99, 0x99}, {0x9F, 0x9F}}, 0x05},               // 09EAh
	{[][2]uint8{{0x2D, 0x2E}, {0x33, 0x38}, {0x43, 0x4D}, {0x5D, 0x61}}, 0x44}, // 0A05h
	{[][2]uint8{{0xB5, 0xB8}, {0xBA, 0xBA}, {0xC1, 0xC8}}, 0x14},               // 0A34h
	{[][2]uint8{{0x2F, 0x32}, {0x62, 0x65}, {0xF2, 0xF2}}, 0x40},               // 0A54h
	{[][2]uint8{{0x26, 0x26}}, 0x41},                                           // 0A74h
	{[][2]uint8{{0xB2, 0xB3}}, 0x60},                                           // 0A80h
	{[][2]uint8{{0x3A, 0x3C}, {0x41, 0x42}}, 0x48},                             // 0A91h
	{[][2]uint8{{0x3D, 0x3F}, {0x79, 0x7C}, {0x87, 0x89}, {0x8B, 0x8B}, {0x8D, 0x8D}, {0x90, 0x93},
		{0x95, 0x95}, {0x97, 0x98}, {0xA0, 0xA2}, {0xA6, 0xA7}, {0xD6, 0xD8}, {0xEE, 0xEE}}, 0x08}, // 0AACh
	{[][2]uint8{{0x57, 0x5A}, {0xA3, 0xA3}, {0xE0, 0xE0}}, 0x88}, // 0B17h
	{[][2]uint8{{0x7D, 0x7D}, {0x7C, 0x7C}, {0x8A, 0x8A}, {0x8C, 0x8C}, {0x8E, 0x8F}, {0x94, 0x94},
		{0x96, 0x96}}, 0x09}, // 0B32h
	{[][2]uint8{{0x7E, 0x85}, {0xE5, 0xE7}}, 0x28},               // 0B61h
	{[][2]uint8{{0xD9, 0xDF}}, 0x18},                             // 0B7Ch
	{[][2]uint8{{0x4E, 0x4E}, {0xF5, 0xF5}, {0xE8, 0xE9}}, 0x20}, // 0B8Ch
	{[][2]uint8{{0x6D, 0x6D}, {0xF4, 0xF6}}, 0x80},               // 0BA6h
	{[][2]uint8{{0x6E, 0x71}, {0x73, 0x78}}, 0x81},               // 0BBBh
	{[][2]uint8{{0x66, 0x6C}, {0x72, 0x72}}, 0x90},               // 0BD5h
	{[][2]uint8{{0xE1, 0xE4}, {0xEA, 0xED}}, 0xA0},               // 0BEAh
}

// OutdoorTerrainFlags 重現 overlay-10 `08B4h`。riverCleared 是 ECL `@4AB3`
// （`[4933h]+366h`）等於 FFh：那時帶 80h 的旗標改成「去掉 80h 再或上 1」
// （`0C02h..0C21h`）。
//
// 地形碼沒有命中任何一條時，原版回的是沒初始化的區域變數 `[bp-2]`，這裡回錯。
// 野外三張圖上真的有這種格子（86h、BBh、F9h、FBh、FDh、FEh、FFh，
// 多半是地點格），產生戰場時改用 OutdoorUnmatchedFlags，見那裡的說明。
func OutdoorTerrainFlags(code uint8, riverCleared bool) (uint8, error) {
	for _, rule := range outdoorFlagRules {
		for _, span := range rule.ranges {
			if code >= span[0] && code <= span[1] {
				flags := rule.flags
				if flags&OutdoorFlag80 != 0 && riverCleared {
					flags = (flags - OutdoorFlag80) | OutdoorFlag01
				}
				return flags, nil
			}
		}
	}
	return 0, fmt.Errorf("Pool wilderness terrain code %02Xh has no entry in overlay-10 08B4h", code)
}

// OutdoorUnmatchedFlags 是 `08B4h` 沒有命中時 remake 用的旗標。
//
// 原版那時回的是堆疊上殘留的 byte：它取決於前一個用到同一個堆疊位置的
// 常式（擲骰、FillChar……），同一場戰鬥裡四支建構器各自讀到的值都可能不同，
// 靜態讀不出來。remake 取 0（四支建構器全走預設值：不畫帶狀、直立物密度 1、
// 撒點權重 0／6／0Fh／28h／0），是明確標出來的安全退路（hypothesis），
// 不是原版行為。要閉合得在 dosgolem 裡站在這種格子上開打，讀 `[6674h]` 的盤面。
const OutdoorUnmatchedFlags uint8 = 0

// OutdoorBattlefield 是 `1255h` 需要的外部狀態。
type OutdoorBattlefield struct {
	// Terrain 是 `DS:45BCh`：大地圖地形碼（gamepack.WildernessTerrainTable.Background）。
	Terrain uint8
	// RiverCleared 是 ECL `@4AB3 == FFh`（`08B4h` 的 `0C0Bh`）。
	RiverCleared bool
	// Block 是 `DS:82A2h`；`0000h` 在區塊 0Ah 不放 1Ch／1Dh。
	Block uint16
}

// DiceRoller 是 overlay-24 entry 8（`0100h:0048h`）：擲 count 顆 sides 面骰加總。
type DiceRoller func(count, sides int) int

// outdoorCanvas 是建構器落筆的盤面。原版以 `y×32h + x` 平面定址，X 超過 49
// 就寫進下一列的開頭；這裡照平面位移寫，只在超出整張配置時報錯。
type outdoorCanvas struct {
	terrain []uint8
	classes CellClasses
	err     error
}

func (c *outdoorCanvas) index(x, y int) int { return y*TacticalRowStride + x }

func (c *outdoorCanvas) set(x, y int, class uint8) {
	index := c.index(x, y)
	if index < 0 || index >= len(c.terrain) {
		if c.err == nil {
			c.err = fmt.Errorf("Pool outdoor builder wrote (%d,%d) outside the tactical map", x, y)
		}
		return
	}
	c.terrain[index] = class
}

// open 說 (x, y) 的圖塊序號是不是還是平地。
func (c *outdoorCanvas) open(x, y int) bool {
	index := c.index(x, y)
	if index < 0 || index >= len(c.terrain) {
		if c.err == nil {
			c.err = fmt.Errorf("Pool outdoor builder read (%d,%d) outside the tactical map", x, y)
		}
		return false
	}
	class := int(c.terrain[index])
	if class >= len(c.classes) {
		if c.err == nil {
			c.err = fmt.Errorf("Pool outdoor cell class %02Xh is outside the class table", class)
		}
		return false
	}
	return c.classes[class].PresentationCode == OpenGroundPresentation
}

// GenerateOutdoorTacticalGrid 重現 overlay-10 `1255h` 整支：FillChar 之後依序
// 跑 `0C7Ah`、`0DB0h`、`0F3Fh`、`0000h`。擲骰的次數與順序照原版，roll 由呼叫端
// 給（remake 的骰子流不是 DOS 的那一條，同狀態下的盤面不會逐格相同）。
func GenerateOutdoorTacticalGrid(field OutdoorBattlefield, classes CellClasses, roll DiceRoller) (TacticalGrid, error) {
	if roll == nil {
		return TacticalGrid{}, fmt.Errorf("Pool outdoor battlefield needs a dice roller")
	}
	flags, err := OutdoorTerrainFlags(field.Terrain, field.RiverCleared)
	if err != nil {
		flags = OutdoorUnmatchedFlags
	}
	grid := NewOutdoorTacticalGrid()
	canvas := &outdoorCanvas{terrain: grid.Terrain, classes: classes}
	outdoorBand(canvas, flags, roll)
	outdoorUprights(canvas, flags, roll)
	outdoorScatter(canvas, flags, roll)
	outdoorRandomObjects(canvas, field.Block, roll)
	if canvas.err != nil {
		return TacticalGrid{}, canvas.err
	}
	return grid, nil
}

// outdoorBand 是 `0C7Ah`。
func outdoorBand(c *outdoorCanvas, flags uint8, roll DiceRoller) {
	threshold := 0
	if flags&OutdoorFlag20 != 0 {
		threshold = 0x23
	}
	if flags&OutdoorFlag10 != 0 {
		threshold = 0x4B
	}
	// `0CAFh`：擲出來的 1d100 大於門檻就整支不畫。
	if roll(1, 100) > threshold {
		return
	}
	// `0CB7h..0CE3h`：起點 34 − 5d4，往左退到 (x+2) 是 7 的倍數。
	x := 0x22 - roll(5, 4)
	for (x+2)%7 > 0 {
		x--
	}
	start := x
	crossings := 0
	// 交叉兩列（34h／35h），並把呼叫端的計數加一（巢狀程序 `0C31h`）。
	cross := func(y, x int) {
		c.set(x, y, 0x34)
		c.set(x+1, y, 0x35)
		crossings++
	}
	for y := 0; y <= TacticalMaxY; y++ {
		if x > TacticalMaxX {
			continue
		}
		c.set(x, y, 0x32)
		c.set(x+1, y, 0x33)
		// `0D39h`：1d20 擲 1，或上一列剛交叉過（計數是奇數），這一列交叉。
		if roll(1, 20) == 1 || crossings&1 != 0 {
			cross(y, x)
		}
		x++
	}
	if crossings == 0 {
		// `0D65h`：整條都沒交叉就在 7..15 列補一段兩列。
		y := 0x0C - roll(1, 9) + 4
		x := y + start
		cross(y, x)
		cross(y+1, x+1)
	}
}

// outdoorUprights 是 `0DB0h`。
func outdoorUprights(c *outdoorCanvas, flags uint8, roll DiceRoller) {
	density := 1
	if flags&OutdoorFlag02 != 0 {
		density = 1
	}
	if flags&OutdoorFlag04 != 0 {
		density = 3
	}
	if flags&OutdoorFlag40 != 0 {
		density = 4
	}
	if flags&OutdoorFlag08 != 0 {
		density = 7
	}
	if flags&OutdoorFlag80 != 0 {
		density = 0
	}
	for x := 0; x <= TacticalMaxX; x++ {
		for y := 1; y <= TacticalMaxY; y++ {
			if !c.open(x, y) || !c.open(x, y-1) {
				continue
			}
			if roll(1, 100) > density {
				continue
			}
			kind := roll(1, 10)
			if kind > 8 {
				kind = roll(1, 2) + 4
			} else {
				kind = (kind + 1) / 2
			}
			if flags&OutdoorFlag40 != 0 {
				kind = roll(1, 3) + 3
			}
			c.set(x, y, uint8(kind+0x1F+4))
			if kind < 5 {
				c.set(x, y-1, uint8(kind+0x1F))
			}
		}
	}
}

// outdoorScatter 是 `0F3Fh`。
func outdoorScatter(c *outdoorCanvas, flags uint8, roll DiceRoller) {
	a, b, cw, d, e := 0, 6, 0x0F, 0x28, 0
	if flags&OutdoorFlag02 != 0 {
		cw, d = 0x2D, 0x0A
	}
	if flags&OutdoorFlag04 != 0 {
		cw = 0x19
	}
	if flags&OutdoorFlag40 != 0 {
		d, e = 5, 0x1E
	}
	if flags&OutdoorFlag80 != 0 {
		a, d, cw = 0x0F, 0x19, 0x0A
	}
	for x := 0; x <= TacticalMaxX; x++ {
		for y := 0; y <= TacticalMaxY; y++ {
			if !c.open(x, y) {
				continue
			}
			r := roll(1, 0xFF)
			switch {
			case r <= a:
				kind := roll(1, 4)
				if kind == 4 && y > 0 && c.open(x, y-1) {
					c.set(x, y-1, 0x41)
					c.set(x, y, 0x40)
				}
				if kind < 4 {
					c.set(x, y, uint8(kind+0x1F+0x1D))
				}
			case r <= a+b:
				c.set(x, y, uint8(roll(1, 3)+0x1F+0x17))
			case r <= a+b+cw:
				c.set(x, y, uint8(roll(1, 4)+0x1F+0x0A))
			case r <= a+b+cw+e:
				c.set(x, y, uint8((roll(1, 10)-1)/3+1+0x1F+0x19))
			case r <= a+b+cw+e+d:
				kind := roll(1, 4)
				if kind == 4 && flags&OutdoorFlag40 == 0 {
					kind = roll(1, 3)
				}
				c.set(x, y, uint8(kind+0x1F+0x0E))
			}
		}
	}
}

// outdoorRandomObjects 是 `0000h`。
func outdoorRandomObjects(c *outdoorCanvas, block uint16, roll DiceRoller) {
	for x := 0; x <= TacticalMaxX; x++ {
		for y := 0; y <= TacticalMaxY; y++ {
			if !c.open(x, y) {
				continue
			}
			switch roll(1, 100) {
			case 0x62:
				c.set(x, y, 0x1A)
			case 0x63:
				c.set(x, y, 0x1B)
			case 0x64:
				if block == 0x0A || roll(1, 100) != 1 {
					continue
				}
				switch kind := roll(1, 10); {
				case kind >= 1 && kind <= 6:
					c.set(x, y, 0x1C)
				case kind >= 7 && kind <= 10:
					c.set(x, y, 0x1D)
				}
			}
		}
	}
}
