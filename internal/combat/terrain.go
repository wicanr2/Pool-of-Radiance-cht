package combat

import "fmt"

// TerrainRuleCount 是 DS:2758h 地形表的筆數。表由 2758h 起、每筆 4 bytes，
// 到 27D7h 為止；緊接著 27D8h 是另一張同構的表，2860h 起是佔格表，
// 三張表把 2758h..287Fh 鋪滿（spec 057）。
const TerrainRuleCount = 32

// originalTerrainRuleBytes 逐位元組照抄 START.EXE 的 DS:2758h..27D7h，
// 見 docs/audit/ida-ds-terrain-rule-table.json。
//
// 索引 0 不是真正的地形：2758h 是編譯器折疊出來的基底，那四個 byte 同時是
// DS:2753h 方向表尾端的資料（spec 053 已記下這個重疊）。真正的地形碼是
// 1..31。索引 0 讀出來的 Block 是 0FFh，於是被當成不可通行，這是佈局的
// 副作用而不是設計，重建時照抄即可，但不要賦予它地形語意。
var originalTerrainRuleBytes = [TerrainRuleCount][4]byte{
	{0x01, 0x00, 0xFF, 0x00}, {0xFF, 0x00, 0x02, 0x00},
	{0xFF, 0x00, 0x02, 0x01}, {0xFF, 0x00, 0x02, 0x02},
	{0xFF, 0x00, 0x02, 0x03}, {0x01, 0x00, 0x00, 0x04},
	{0xFF, 0x00, 0x02, 0x05}, {0xFF, 0x00, 0x02, 0x06},
	{0xFF, 0x00, 0x02, 0x07}, {0x01, 0x00, 0x00, 0x08},
	{0xFF, 0x00, 0x02, 0x09}, {0x01, 0x00, 0x00, 0x0A},
	{0xFF, 0x00, 0x02, 0x0B}, {0x01, 0x00, 0x00, 0x0C},
	{0xFF, 0x00, 0x02, 0x0D}, {0x01, 0x00, 0x00, 0x0E},
	{0xFF, 0x00, 0x02, 0x0F}, {0x01, 0x00, 0x00, 0x10},
	{0xFF, 0x00, 0x02, 0x11}, {0xFF, 0x00, 0x02, 0x12},
	{0xFF, 0x00, 0x02, 0x13}, {0xFF, 0x00, 0x02, 0x14},
	{0xFF, 0x00, 0x02, 0x15}, {0x01, 0x00, 0x00, 0x16},
	{0x01, 0x00, 0x00, 0x17}, {0xFF, 0x00, 0x02, 0x18},
	{0x01, 0x00, 0x00, 0x22}, {0x01, 0x00, 0x00, 0x23},
	{0x01, 0x00, 0x00, 0x24}, {0x01, 0x00, 0x00, 0x25},
	{0x01, 0x00, 0x00, 0x26}, {0x01, 0x00, 0x00, 0x27},
}

// OriginalTerrainRules 回傳原版第一張地形表，每次呼叫都是新的切片，
// 呼叫端改動不會污染表本身。
func OriginalTerrainRules() []TerrainRule {
	rules := make([]TerrainRule, TerrainRuleCount)
	for index, row := range originalTerrainRuleBytes {
		rules[index] = TerrainRule{
			EntryThreshold: row[0],
			Level:          row[1],
			Block:          row[2],
			Field3:         row[3],
		}
	}
	return rules
}

// SortNearbyCells 重現 overlay-31 `002Eh`：對結果表做交換排序，
// 成本小的排前面；成本相同時朝向索引小的排前面，但斜向不得越過正向
// （原版以 `facing mod 2` 比較，偶數是四個正向、奇數是四個對角）。
//
// 這個比較關係不是全序，所以不能換成一般的排序函式；必須照原版的
// 兩層迴圈逐對交換，結果才會一致。
func SortNearbyCells(cells []NearbyCell) {
	if len(cells) <= 1 {
		return
	}
	for i := 0; i < len(cells)-1; i++ {
		for j := i + 1; j < len(cells); j++ {
			if nearbyCellPrecedes(cells[j], cells[i]) {
				cells[i], cells[j] = cells[j], cells[i]
			}
		}
	}
}

func nearbyCellPrecedes(candidate, incumbent NearbyCell) bool {
	if candidate.Cost < incumbent.Cost {
		return true
	}
	if candidate.Cost != incumbent.Cost {
		return false
	}
	if candidate.Facing >= incumbent.Facing {
		return false
	}
	return candidate.Facing%2 <= incumbent.Facing%2
}

// TerrainRuleAt 取出一筆地形規則，超出表範圍時失敗即關閉——原版會讀進
// 緊接在後面的第二張表，那是資料佈局的巧合，不是可以依賴的行為。
func TerrainRuleAt(rules []TerrainRule, code uint8) (TerrainRule, error) {
	if int(code) >= len(rules) {
		return TerrainRule{}, fmt.Errorf("Pool terrain code %d is outside the rule table (0..%d)",
			code, len(rules)-1)
	}
	return rules[code], nil
}
