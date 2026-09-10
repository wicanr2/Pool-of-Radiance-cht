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

// 執行期的效果串列。原版把它掛在角色記錄的 `+7Fh`：一條單向串列，
// **新的接在尾端**（overlay-24 entry 10 `0E98h` 沿 `+5` 走到 NULL 才接），
// 所以效果的順序就是掛上去的順序，線性搜尋找到的是最早掛上的那一個
//（spec 059／112）。
//
// remake 這一側用 slice 表示同一條串列：順序就是 `+5` 指出來的順序，
// 不必也不該保留那個遠指標。
const (
	// effectNodeDurationOffset 是持續（word）在 Payload 裡的位置（節點 `+1`）。
	effectNodeDurationOffset = 0
	// effectNodeLevelOffset 是打包了四件事的那個 byte（節點 `+3`）。
	effectNodeLevelOffset = 2
	// effectNodeTeardownOffset 是「摘掉時要叫一次處理常式」（節點 `+4`）。
	effectNodeTeardownOffset = 3

	// EffectLevelMask 是節點 `+3` 的低四位：下這個效果的人的等級。
	EffectLevelMask = 0x0F
	// EffectAppliedBit 是位元 5：這個效果已經套過了。
	EffectAppliedBit = 0x20
	// EffectOriginalSideBit 是位元 6：被影響之前原本的陣營。
	EffectOriginalSideBit = 0x40
	// EffectCasterSideBit 是位元 7：施法者的陣營。
	EffectCasterSideBit = 0x80
	// EffectUndispellable 是整個 `+3` 都是 `0FFh`：解除魔法解不掉
	//（spec 098 的 `23BEh`：`節點 +3 >= 0FFh → 跳過`）。
	EffectUndispellable = 0xFF
)

// NewEffectNode 造一個節點。參數順序與 overlay-24 entry 10 相同。
func NewEffectNode(code uint8, duration uint16, casterLevel uint8, teardown bool) EffectNode {
	node := EffectNode{Code: code}
	node.Payload[effectNodeDurationOffset] = byte(duration)
	node.Payload[effectNodeDurationOffset+1] = byte(duration >> 8)
	node.Payload[effectNodeLevelOffset] = casterLevel
	if teardown {
		node.Payload[effectNodeTeardownOffset] = 1
	}
	return node
}

// Duration 是節點 `+1..+2`。
func (node EffectNode) Duration() uint16 {
	return uint16(node.Payload[effectNodeDurationOffset]) |
		uint16(node.Payload[effectNodeDurationOffset+1])<<8
}

// SetDuration 改寫持續。
func (node *EffectNode) SetDuration(value uint16) {
	node.Payload[effectNodeDurationOffset] = byte(value)
	node.Payload[effectNodeDurationOffset+1] = byte(value >> 8)
}

// CasterLevel 是節點 `+3` 的低四位。解不掉的節點沒有等級可言，回 0。
func (node EffectNode) CasterLevel() uint8 {
	if node.Undispellable() {
		return 0
	}
	return node.Payload[effectNodeLevelOffset] & EffectLevelMask
}

// CloudIndex 是節點 `+3` 的高四位：這是這個施法者的第幾團雲。
// `0CDEh` 一進來做的就是 `es:[di+3] ÷ 10h`（spec 121）。
//
// **只有代碼 `28h` 該叫這一支。** 同一個 byte 在別的代碼上是別的意思——
// 魅惑拿位元 6／7 記兩邊的陣營（spec 112），照這裡讀會讀出一個假的雲序號。
func (node EffectNode) CloudIndex() int {
	return int(node.Payload[effectNodeLevelOffset] >> 4)
}

// Undispellable 回答解除魔法解不解得掉。
func (node EffectNode) Undispellable() bool {
	return node.Payload[effectNodeLevelOffset] == EffectUndispellable
}

// Applied 是位元 5。
func (node EffectNode) Applied() bool {
	return node.Payload[effectNodeLevelOffset]&EffectAppliedBit != 0
}

// OriginalSide 是位元 6：被影響之前原本的陣營。
func (node EffectNode) OriginalSide() uint8 {
	return (node.Payload[effectNodeLevelOffset] & EffectOriginalSideBit) >> 6
}

// CasterSide 是位元 7：施法者的陣營。
func (node EffectNode) CasterSide() uint8 {
	return (node.Payload[effectNodeLevelOffset] & EffectCasterSideBit) >> 7
}

// NeedsTeardown 是節點 `+4`：摘掉之前要不要叫一次處理常式。
func (node EffectNode) NeedsTeardown() bool {
	return node.Payload[effectNodeTeardownOffset] != 0
}

// MarkApplied 重現 overlay-12 entry 14 的 `0450h`：立起位元 5，
// 同時把目前的陣營記進位元 6。套過的不重複套。
func (node *EffectNode) MarkApplied(currentSide uint8) bool {
	if node.Applied() {
		return false
	}
	node.Payload[effectNodeLevelOffset] += EffectAppliedBit + (currentSide&1)<<6
	return true
}

// EffectList 是一個戰鬥員身上的效果串列。
type EffectList []EffectNode

// Append 接在尾端，與原版的 `0E98h` 相同。
func (list EffectList) Append(node EffectNode) EffectList {
	return append(append(EffectList(nil), list...), node)
}

// IndexOf 線性搜尋一個代碼，回**最早掛上**的那一個（spec 059 的 `21DCh`）。
func (list EffectList) IndexOf(code uint8) (int, bool) {
	for index, node := range list {
		if node.Code == code {
			return index, true
		}
	}
	return 0, false
}

// Has 回答身上有沒有這個代碼。
func (list EffectList) Has(code uint8) bool {
	_, ok := list.IndexOf(code)
	return ok
}

// RemoveAt 摘掉一個節點。
func (list EffectList) RemoveAt(index int) EffectList {
	if index < 0 || index >= len(list) {
		return list
	}
	result := make(EffectList, 0, len(list)-1)
	result = append(result, list[:index]...)
	return append(result, list[index+1:]...)
}

// Remove 摘掉最早掛上的那一個指定代碼；沒有就原樣回傳。
func (list EffectList) Remove(code uint8) EffectList {
	index, ok := list.IndexOf(code)
	if !ok {
		return list
	}
	return list.RemoveAt(index)
}

// DispelChance 是解除魔法對一個節點的成功率（overlay-22 `23D7h`..`2420h`）。
//
// 以 50 為基準，施法者高一級加 5、低一級只扣 2——**不對稱**，對高階施法者
// 有利。這裡不夾在 0..100：原版也沒夾，擲的是 `骰(1,100) <= 這個值`，
// 所以負數等於必敗、超過 100 等於必成。
func DispelChance(casterLevel, effectLevel int) int {
	switch {
	case casterLevel > effectLevel:
		return 50 + (casterLevel-effectLevel)*5
	case casterLevel < effectLevel:
		return 50 - (effectLevel-casterLevel)*2
	}
	return 50
}

// Dispel 對串列上每一個節點各擲一次（overlay-22 `2356h` 的內層迴圈）。
// `+3` 是 `0FFh` 的節點直接跳過。roll 收到成功率、回傳 1..100 的骰值。
// 回傳剩下的串列與拿掉了幾個。
func (list EffectList) Dispel(casterLevel int, roll func() int) (EffectList, int) {
	result := make(EffectList, 0, len(list))
	removed := 0
	for _, node := range list {
		if node.Undispellable() {
			result = append(result, node)
			continue
		}
		if roll() <= DispelChance(casterLevel, int(node.CasterLevel())) {
			removed++
			continue
		}
		result = append(result, node)
	}
	return result, removed
}
