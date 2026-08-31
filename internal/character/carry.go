package character

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/golden-box-remake-engine/combat/ability"
)

const ItemRecordSize = 63

// ItemLoad decodes only the two Pool fields proven by Spec 035.
func ItemLoad(raw []byte) (int, error) {
	if len(raw) != ItemRecordSize {
		return 0, fmt.Errorf("Pool item has %d bytes, want %d", len(raw), ItemRecordSize)
	}
	weight := int(binary.LittleEndian.Uint16(raw[0x37:0x39]))
	if quantity := int(raw[0x39]); quantity > 0 {
		weight *= quantity
	}
	return weight, nil
}

// CarryCapacity reproduces overlay-25's adjustment table plus 1500.
func CarryCapacity(strength, exceptional int) (int, error) {
	index, ok := ability.StrengthIndex(strength, exceptional)
	if !ok {
		return 0, fmt.Errorf("strength %d/%d has no original table index", strength, exceptional)
	}
	adjustment := 0
	switch {
	case index <= 3:
		adjustment = -350
	case index <= 5:
		adjustment = -250
	case index <= 7:
		adjustment = -150
	case index <= 11:
		adjustment = 0
	case index <= 13:
		adjustment = 100
	case index <= 15:
		adjustment = 200
	case index == 16:
		adjustment = 350
	case index <= 21:
		adjustment = 500 + (index-17)*250
	case index <= 26:
		adjustment = 2000 + (index-22)*1000
	case index == 27:
		adjustment = 7500
	case index <= 30:
		adjustment = 9000 + (index-28)*3000
	default:
		return 0, fmt.Errorf("strength table index %d is outside 1..30", index)
	}
	return 1500 + adjustment, nil
}

// CanReceiveItem checks the original pre-insertion count and weight gates.
func CanReceiveItem(strength, exceptional int, inventory [][]byte, incoming []byte) (bool, error) {
	if len(inventory) > 15 {
		return false, nil
	}
	capacity, err := CarryCapacity(strength, exceptional)
	if err != nil {
		return false, err
	}
	load := 0
	for index, raw := range inventory {
		value, err := ItemLoad(raw)
		if err != nil {
			return false, fmt.Errorf("inventory item %d: %w", index, err)
		}
		load += value
	}
	incomingLoad, err := ItemLoad(incoming)
	if err != nil {
		return false, err
	}
	return load+incomingLoad <= capacity, nil
}
