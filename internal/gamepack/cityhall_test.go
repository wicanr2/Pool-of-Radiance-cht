package gamepack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cityHallSlots(t *testing.T) []CityHallSlot {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("original DOS ZIP is intentionally not tracked")
	}
	archive, err := ReadDOSECLArchive(zipPath, 3)
	if err != nil {
		t.Fatal(err)
	}
	slots, err := ReadCityHallNotifications(archive)
	if err != nil {
		t.Fatal(err)
	}
	return slots
}

func TestReadCityHallNotificationsMatchesOriginalDispatch(t *testing.T) {
	slots := cityHallSlots(t)
	if len(slots) != cityHallSlotCount {
		t.Fatalf("dispatch produced %d slots, want %d", len(slots), cityHallSlotCount)
	}
	for index, slot := range slots {
		if slot.Index != index {
			t.Fatalf("slot %d reports index %d", index, slot.Index)
		}
		want := uint16(cityHallStateBase + index)
		if slot.StateAddress != want {
			t.Fatalf("slot %d maps to 0x%04X, want 0x%04X", index, slot.StateAddress, want)
		}
	}
}

// 這些槽的位址與文字由 spec 041 的 producer 表固定；producer 端見 spec 055。
func TestCityHallSlotsCarryOriginalNotificationText(t *testing.T) {
	slots := cityHallSlots(t)
	for _, expected := range []struct {
		index    int
		address  uint16
		fragment string
	}{
		{0, 0x4AA6, "NORRIS THE GRAY"},
		{1, 0x4AA7, "SOKAL KEEP"},
		{10, 0x4AB0, "PODAL PLAZA"},
		{11, 0x4AB1, "GRAVEYARD MENACE"},
		{18, 0x4AB8, "CADORNA"},
		{21, 0x4ABB, "SLUM AREAS"},
	} {
		slot := slots[expected.index]
		if slot.StateAddress != expected.address {
			t.Fatalf("slot %d maps to 0x%04X, want 0x%04X", expected.index, slot.StateAddress, expected.address)
		}
		if !strings.Contains(slot.Text, expected.fragment) {
			t.Fatalf("slot %d text %q does not contain %q", expected.index, slot.Text, expected.fragment)
		}
	}
}

// spec 041：二十六條分支中只有十條會把 4AC1h 加一。
func TestCityHallProgressIncrementsAreTenSlots(t *testing.T) {
	slots := cityHallSlots(t)
	count := 0
	for _, slot := range slots {
		if slot.IncrementsProgress {
			count++
		}
	}
	if count != 10 {
		t.Fatalf("%d slots increment 0x%04X, want 10", count, CityHallProgressAddress())
	}
}

func TestPendingCityHallSlotsFollowsScanOrder(t *testing.T) {
	state := make([]byte, cityHallSlotCount)
	state[21] = CityHallSlotPending
	state[1] = CityHallSlotPending
	state[10] = CityHallSlotAcknowledged
	pending, err := PendingCityHallSlots(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || pending[0] != 1 || pending[1] != 21 {
		t.Fatalf("pending slots %v, want [1 21]", pending)
	}
}

func TestPendingCityHallSlotsRejectsWrongLength(t *testing.T) {
	if _, err := PendingCityHallSlots(make([]byte, 4)); err == nil {
		t.Fatal("short state table accepted")
	}
}

func TestAcknowledgeCityHallSlotWritesFFOnce(t *testing.T) {
	slots := cityHallSlots(t)
	state := make([]byte, cityHallSlotCount)
	state[21] = CityHallSlotPending

	increments, err := AcknowledgeCityHallSlot(state, slots, 21)
	if err != nil {
		t.Fatal(err)
	}
	if !increments {
		t.Fatal("clearing the Slums does not increment the public progress counter")
	}
	if state[21] != CityHallSlotAcknowledged {
		t.Fatalf("slot 21 holds 0x%02X after acknowledgement, want 0x%02X", state[21], CityHallSlotAcknowledged)
	}
	if _, err := AcknowledgeCityHallSlot(state, slots, 21); err == nil {
		t.Fatal("acknowledged slot was accepted a second time")
	}
}

func TestAcknowledgeCityHallSlotRejectsUnfinishedSlot(t *testing.T) {
	slots := cityHallSlots(t)
	state := make([]byte, cityHallSlotCount)
	if _, err := AcknowledgeCityHallSlot(state, slots, 0); err == nil {
		t.Fatal("slot holding zero was acknowledged")
	}
}
