package main

// 載入 ECL 區塊時的寫入（#41；spec 106〈載入區塊時清掉的兩段〉）：原版每換一個區塊就把
// `4A00..4A1F` 與 `6E79..6E82` 清成 0，再跑新區塊的入口。職員格寫的 `4A01 = 1` 走出市政廳就沒了；
// 讀檔讀回來的值不清。對照的原版收據是 `docs/audit/dosgolem-4a01-block-load-clear.json`（同 archive）
// 與 `docs/audit/dosgolem-block-load-clear-cases.json`（跨 archive、讀檔）。

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// clerkVisit 從城區白天走進市政廳、踩到職員格 (5,5) 並把職員的話按完。
func clerkVisit(t *testing.T) *mainlineDriver {
	t.Helper()
	d := cityHallAt(t, 6)
	a := d.a
	d.settle("Exit")
	if a.eclArchive != 3 || a.eventSession.CurrentBlockID() != 8 {
		t.Fatalf("白天從 (3,4) 朝東到了 ECL%d/%d，該進市政廳（block 8）", a.eclArchive, a.eventSession.CurrentBlockID())
	}
	anywhere := func(int, int) bool { return true }
	clerk := func(x, y int) bool { return x == 5 && y == 5 }
	if !d.walkAllowing("City Hall clerk (5,5)", clerk, anywhere, false) {
		t.Fatalf("走不到職員格，停在 %+v", a.spawn)
	}
	d.settle()
	// 正對照：職員入口 `ecl3/8 9BACh SAVE 1 @4A01`。沒有這個 1，後面的「清成 0」什麼也證明不了。
	if got := a.eventMachine.Memory[0x4A01]; got != 1 {
		t.Fatalf("職員格之後 4A01=%d，`9BACh` 該寫 1（位置 %+v）", got, a.spawn)
	}
	return d
}

// leaveCityHall 從市政廳 (4,4) 朝西踏回城區 (3,4)，dosgolem 收據走的就是這一步。
func leaveCityHall(t *testing.T, d *mainlineDriver) {
	t.Helper()
	a := d.a
	anywhere := func(int, int) bool { return true }
	hall := func(x, y int) bool { return x == 4 && y == 4 }
	if !d.walkAllowing("City Hall (4,4)", hall, anywhere, false) {
		t.Fatalf("走不回 (4,4)，停在 %+v", a.spawn)
	}
	d.settle()
	d.face(3)
	d.step(ebiten.KeyArrowUp)
	d.settle()
	if a.eventSession.CurrentBlockID() != 0 || a.spawn.X != 3 || a.spawn.Y != 4 {
		t.Fatalf("從 (4,4) 朝西沒有回到城區 (3,4)：ECL%d/%d %+v", a.eclArchive, a.eventSession.CurrentBlockID(), a.spawn)
	}
}

func TestLeavingCityHallClearsTheBlockScratch(t *testing.T) {
	d := clerkVisit(t)
	a := d.a
	a.eventMachine.Memory[0x6E7A] = 5 // class 1 那一段也要清；城區入口不寫 6E7A
	leaveCityHall(t, d)
	memory := a.eventMachine.Memory
	if memory[0x4A01] != 0 || memory[0x6E7A] != 0 {
		t.Fatalf("走出市政廳 4A01=%d 6E7A=%d，原版 overlay-07 `02F9h`／`031Eh` 清成 0", memory[0x4A01], memory[0x6E7A])
	}
	// entry 3 的其他寫入。
	if memory[0x49E6] != 1 || memory[0x6DE1] != 0xFF {
		t.Fatalf("走出市政廳 49E6=%d 6DE1=%02X，overlay-07 `02D1h`／`0237h` 寫 1／FF", memory[0x49E6], memory[0x6DE1])
	}
}

// 讀檔讀回來的值不清：在職員格存檔、讀檔，`4A01` 還是 1；之後第一次換區照常清。
func TestLoadingInsideCityHallKeepsTheScratchUntilTheNextBlock(t *testing.T) {
	d := clerkVisit(t)
	a := d.a
	statePath := filepath.Join(t.TempDir(), "state.json")
	a.saveState = func(state poolsave.State) error { return poolsave.WriteAtomic(statePath, state) }
	savedSpawn := a.spawn
	if err := press(a, ebiten.KeyF10); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("F10 存檔回傳 %v", err)
	}
	restored, err := newApp(filepath.Join("..", "..", "Pool of Radiance (1988).zip"), statePath)
	if err != nil {
		t.Fatal(err)
	}
	restored.roller = a.roller
	// 讀檔之後是另一個 app；駕駛的 step 綁的是舊的那一個，要一起換。
	d.a = restored
	d.step = func(key ebiten.Key) {
		if err := press(restored, key); err != nil {
			t.Fatal(err)
		}
	}
	d.step(ebiten.KeyEnter)
	d.step(ebiten.KeyL)
	if restored.mode != modeAdventure || restored.spawn != savedSpawn || restored.eventSession.CurrentBlockID() != 8 {
		t.Fatalf("讀檔之後 mode=%d spawn=%+v ECL%d/%d，要回到市政廳 %+v", restored.mode, restored.spawn,
			restored.eclArchive, restored.eventSession.CurrentBlockID(), savedSpawn)
	}
	if got := restored.eventMachine.Memory[0x4A01]; got != 1 {
		t.Fatalf("讀檔之後 4A01=%d，原版讀回來是 1", got)
	}
	leaveCityHall(t, d)
	if got := restored.eventMachine.Memory[0x4A01]; got != 0 {
		t.Fatalf("讀檔之後第一次換區 4A01=%d，原版清成 0", got)
	}
}
