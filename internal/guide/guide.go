// Package guide 提供遊戲內的攻略提示：目前這張地圖上有哪些地點，
// 各在哪一格。
//
// **座標一律出自原始資料，不出自攻略。** 每一個點的 `(x,y)` 都是從原始 GEO
// 的地形碼算出來的（`cmd/pool-world-graph` 的 `cell_terrain` 配上
// `docs/spec/102-city-location-dispatch.md` 的索引表），名字取自原版腳本自己
// 印出來的第一句話。把攻略的座標抄進來的症狀是「測試綠、玩家走不到」——
// 兩者在報表上分不出來（CoAB `CLAUDE.md` 的通則第 4 條）。
//
// 第三方攻略只當**檢查清單**用：它說某一區該有幾個地點，用來抓自己的盤點
// 是不是漏了一整塊（`docs/reference/walkthrough-notes-softworld-001.md`）。
// 敘述文字一律自己寫。
package guide

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed zh-TW.json
var traditionalChineseJSON []byte

//go:embed en.json
var englishJSON []byte

// Schema 是資料檔的版本；對不上就失敗即關閉，不要靜靜地當成空攻略——
// 空攻略在畫面上與「這張圖還沒建點」長得一樣。
const Schema = "pool-guide-maps/1"

// Point 是地圖上的一個地點。Source 指向 repo 裡建立這個座標的那份文件。
type Point struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
	Source  string `json:"source"`
}

// Map 是一張 GEO 地圖的攻略。
type Map struct {
	Title  string  `json:"title"`
	Source string  `json:"source"`
	Points []Point `json:"points"`
}

// Catalogue 是一個語言的整份攻略。
type Catalogue struct {
	Schema string         `json:"schema"`
	Note   string         `json:"note"`
	Maps   map[string]Map `json:"maps"`
}

// Key 是地圖的鍵：封存檔編號加區塊編號，與 `GEO<archive>/<block>` 同一組數字。
func Key(archive, block uint8) string { return fmt.Sprintf("%d/%02d", archive, block) }

// Parse 讀一份攻略。
func Parse(raw []byte) (*Catalogue, error) {
	var catalogue Catalogue
	if err := json.Unmarshal(raw, &catalogue); err != nil {
		return nil, fmt.Errorf("parse guide catalogue: %w", err)
	}
	if catalogue.Schema != Schema {
		return nil, fmt.Errorf("guide catalogue schema is %q, want %q", catalogue.Schema, Schema)
	}
	return &catalogue, nil
}

// TraditionalChinese 與 English 是內建的兩份。
func TraditionalChinese() (*Catalogue, error) { return Parse(traditionalChineseJSON) }
func English() (*Catalogue, error)            { return Parse(englishJSON) }

// Map 取出一張地圖的攻略。沒有那一張就回 false——**呼叫端要把它畫成
// 「這張圖還沒建點」**，不是畫成一張空地圖。
func (c *Catalogue) Map(archive, block uint8) (Map, bool) {
	if c == nil {
		return Map{}, false
	}
	definition, ok := c.Maps[Key(archive, block)]
	return definition, ok
}

// Labels 把同一張地圖上的點依標籤收攏，每個標籤留下最靠近 (x,y) 的那一格。
//
// 一個地點常常佔好幾格（旅店七格、酒館七格），逐格列在文字清單上會把畫面
// 灌滿重複的行；地圖上的標記仍然逐格畫，那才是玩家要看的。
func (m Map) Labels(x, y int) []Point {
	best := map[string]Point{}
	for _, point := range m.Points {
		current, seen := best[point.Label]
		if !seen || distance(point, x, y) < distance(current, x, y) {
			best[point.Label] = point
		}
	}
	out := make([]Point, 0, len(best))
	for _, point := range best {
		out = append(out, point)
	}
	sort.Slice(out, func(i, j int) bool {
		left, right := distance(out[i], x, y), distance(out[j], x, y)
		if left != right {
			return left < right
		}
		return out[i].Label < out[j].Label
	})
	return out
}

func distance(point Point, x, y int) int {
	dx, dy := point.X-x, point.Y-y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}
