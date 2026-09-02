package gamepack

import (
	"encoding/binary"
	"fmt"
)

// 角色的效果串列。原版的預設人物檔把它另存成 `.spc`，與 `.sav`（285-byte 角色
// 記錄）、`.itm`（63-byte 物品記錄）並列。
//
// 節點的形狀由 spec 059 決定，不是從檔案猜的：overlay-25 entry 27 走訪串列時，
// 節點的 `+0` 是效果代碼、`+5` 是下一個節點的遠指標。9 bytes 正好是
// 5 bytes 內容加 4 bytes 遠指標，而所有 `.spc` 的大小都是 9 的倍數。
//
// 存在檔案裡的遠指標是上次執行時的位址，重新載入時沒有意義；解析時只用它
// 判斷「後面還有沒有節點」——但**檔案本身就是順序排列的**，所以走訪照順序，
// 不追指標。
const (
	// EffectNodeSize 是一個節點的 byte 數。
	EffectNodeSize = 9
	// effectNodeNextOffset 是遠指標的位置，也就是內容的結尾。
	effectNodeNextOffset = 5
)

// EffectNode 是一個效果節點。除了代碼之外的四個 byte 語意尚未閉合，
// 原樣保留——它們在不同效果上代表不同東西，猜錯會讓數值默默算錯。
type EffectNode struct {
	// Code 是效果代碼（節點 `+0`）。
	Code uint8
	// Payload 是 `+1`..`+4`。
	Payload [4]byte
	// SavedNext 是存檔時的遠指標，只作為原始資料保留，不拿來走訪。
	SavedNext uint32
}

// ParseEffectList 解出一份 `.spc` 的內容。
func ParseEffectList(raw []byte) ([]EffectNode, error) {
	if len(raw)%EffectNodeSize != 0 {
		return nil, fmt.Errorf("Pool effect list has %d bytes, not a multiple of %d",
			len(raw), EffectNodeSize)
	}
	nodes := make([]EffectNode, 0, len(raw)/EffectNodeSize)
	for offset := 0; offset < len(raw); offset += EffectNodeSize {
		entry := raw[offset : offset+EffectNodeSize]
		node := EffectNode{
			Code:      entry[0],
			SavedNext: binary.LittleEndian.Uint32(entry[effectNodeNextOffset:]),
		}
		copy(node.Payload[:], entry[1:effectNodeNextOffset])
		nodes = append(nodes, node)
	}
	return nodes, nil
}
