package gamepack

import (
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/ecl"
)

// City Hall 的完成通知由 ECL3/block 8 驅動：clerk 逐槽掃描 4AA6h..4ABFh，值為 FEh
// 的槽才經 9D63h 的 ON GOSUB 演出一次通知，之後把該槽改成 FFh。二十六條分支中只有
// 一部分會把 4AC1h 加一。行為與位址見 spec 041，producer 端見 spec 055。
const (
	cityHallBlockID         = 8
	cityHallCodeBase        = 0x9900
	cityHallDispatchAddress = 0x9D63
	cityHallStateBase       = 0x4AA6
	cityHallProgressAddress = 0x4AC1
	cityHallSlotCount       = 26

	// CityHallSlotPending 是已完成但尚未在 City Hall 結算的值；
	// CityHallSlotAcknowledged 是 clerk 演出通知後寫回的值。
	CityHallSlotPending      byte = 0xFE
	CityHallSlotAcknowledged byte = 0xFF
)

// CityHallSlot 是一條完成通知分支。Text 取該分支的第一段玩家文字，
// IncrementsProgress 表示這條分支會把 4AC1h 加一。
type CityHallSlot struct {
	Index              int
	StateAddress       uint16
	Target             uint16
	Text               string
	IncrementsProgress bool
}

// ReadCityHallNotifications 由原版 ECL3/block 8 解出二十六條完成通知分支。
// 槽數、順序與文字一律取自原始 ON GOSUB edge，不接受任何硬編碼替代。
func ReadCityHallNotifications(archive ECLArchive) ([]CityHallSlot, error) {
	if archive.Number != 3 {
		return nil, fmt.Errorf("City Hall notifications live in ECL archive 3, got %d", archive.Number)
	}
	block, ok := archive.Blocks[cityHallBlockID]
	if !ok {
		return nil, fmt.Errorf("ECL archive 3 is missing block %d", cityHallBlockID)
	}
	points, _, err := ecl.EntryPoints(block, 5)
	if err != nil {
		return nil, err
	}
	starts := make([]int, len(points))
	for index, point := range points {
		starts[index] = int(point) - cityHallCodeBase
	}
	graph, err := ecl.TraceGraphAtBase(block, starts, cityHallCodeBase, len(block)*8)
	if err != nil {
		return nil, err
	}

	dispatchOffset := cityHallDispatchAddress - cityHallCodeBase
	targets := make([]int, 0, cityHallSlotCount)
	for _, edge := range graph.Edges {
		if edge.From == dispatchOffset && edge.Kind == "ON GOSUB" {
			targets = append(targets, edge.To)
		}
	}
	if len(targets) != cityHallSlotCount {
		return nil, fmt.Errorf("City Hall dispatch at 0x%04X has %d branches, want %d",
			cityHallDispatchAddress, len(targets), cityHallSlotCount)
	}

	byOffset := make(map[int]int, len(graph.Instructions))
	for index, instruction := range graph.Instructions {
		byOffset[instruction.Offset] = index
	}

	slots := make([]CityHallSlot, cityHallSlotCount)
	for slot, target := range targets {
		start, found := byOffset[target]
		if !found {
			return nil, fmt.Errorf("City Hall branch %d targets offset %d, which is absent from the graph", slot, target)
		}
		end := len(graph.Instructions)
		if slot+1 < len(targets) {
			if next, ok := byOffset[targets[slot+1]]; ok && next > start {
				end = next
			}
		}
		row := CityHallSlot{
			Index:        slot,
			StateAddress: uint16(cityHallStateBase + slot),
			Target:       uint16(cityHallCodeBase + target),
		}
		for index := start; index < end; index++ {
			instruction := graph.Instructions[index]
			name := instruction.Command.Name
			if row.Text == "" && (name == "PRINT" || name == "PRINTCLEAR") && len(instruction.Operands) != 0 {
				row.Text, _ = ecl.TextValue(instruction.Operands[0], nil)
			}
			if isProgressIncrement(instruction) {
				row.IncrementsProgress = true
			}
		}
		slots[slot] = row
	}
	return slots, nil
}

// PendingCityHallSlots 回傳狀態表中值為 FEh 的槽索引，順序即 clerk 的掃描順序。
// state 必須正好是 4AA6h..4ABFh 這二十六個位元組，長度不符即失敗即關閉。
func PendingCityHallSlots(state []byte) ([]int, error) {
	if len(state) != cityHallSlotCount {
		return nil, fmt.Errorf("City Hall state table needs %d bytes, got %d", cityHallSlotCount, len(state))
	}
	pending := make([]int, 0, cityHallSlotCount)
	for slot, value := range state {
		if value == CityHallSlotPending {
			pending = append(pending, slot)
		}
	}
	return pending, nil
}

// AcknowledgeCityHallSlot 把一個待結算的槽改成 FFh，並回報是否應該把 4AC1h 加一。
// 只有值為 FEh 的槽能被結算；已是 FFh 或其他值一律拒絕，避免重複演出。
func AcknowledgeCityHallSlot(state []byte, slots []CityHallSlot, slot int) (bool, error) {
	if len(state) != cityHallSlotCount {
		return false, fmt.Errorf("City Hall state table needs %d bytes, got %d", cityHallSlotCount, len(state))
	}
	if slot < 0 || slot >= cityHallSlotCount {
		return false, fmt.Errorf("City Hall slot %d is outside 0..%d", slot, cityHallSlotCount-1)
	}
	if len(slots) != cityHallSlotCount {
		return false, fmt.Errorf("City Hall notification table needs %d rows, got %d", cityHallSlotCount, len(slots))
	}
	if state[slot] != CityHallSlotPending {
		return false, fmt.Errorf("City Hall slot %d holds 0x%02X, only 0x%02X can be acknowledged", slot, state[slot], CityHallSlotPending)
	}
	state[slot] = CityHallSlotAcknowledged
	return slots[slot].IncrementsProgress, nil
}

// CityHallProgressAddress 是公告進度計數 4AC1h 的原始位址，供存檔與 runtime 對帳。
func CityHallProgressAddress() uint16 { return cityHallProgressAddress }

func isProgressIncrement(instruction ecl.Instruction) bool {
	if instruction.Command.Opcode != 0x04 || len(instruction.Operands) != 3 {
		return false
	}
	amount, source, destination := instruction.Operands[0], instruction.Operands[1], instruction.Operands[2]
	return amount.Code == 0 && amount.Low == 1 &&
		source.WordSet && source.Word == cityHallProgressAddress &&
		destination.WordSet && destination.Word == cityHallProgressAddress
}
