package main

// #125（spec 167〈三〉、spec 136、spec 154）：
//   - overlay-05 `1164h` 把不在場的敵方數寫進 @6DC8，瓦海登墳場的腳本拿它扣骷髏數；
//   - ADD NPC 加入的那一位身上帶著記錄 `+88h` 的錢。

import (
	"encoding/hex"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	pooltreasure "github.com/wicanr2/Pool-of-Radiance-cht/internal/treasure"
)

// ECL4 block 10 `9CD7h`：骷髏那一場打完回來，`SUBTRACT @6DC8 @4A01 → @4A01`（剩下的骷髏）、
// `ADD @6DC8 @4A39 → @4A39`。三隻全部打倒：@6DC8 = 3，剩下的從 10 變 7。
func TestGraveyardSkeletonsDropByTheDefeatedCount(t *testing.T) {
	application := scriptApp(t, 4, 10, 0x9CD7)
	const remaining, tally = 0x4A01, 0x4A39
	application.eventMachine.Memory[remaining] = 10
	application.eventMachine.Memory[tally] = 3
	application.eventMachine.Memory[defeatedCountAddress] = 0
	if err := application.enterCombatStaging([]eclvm.MonsterSpawn{{MonsterID: 106, Count: 3, IconBlock: 2}}); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	if application.tactical == nil || len(foeIndexes(application.tactical)) != 3 {
		t.Fatal("the three foes are not on the board")
	}
	winByKeys(t, application)
	if got := application.eventMachine.Memory[defeatedCountAddress]; got != 3 {
		t.Fatalf("@6DC8 = %d after three foes surrendered, want 3", got)
	}
	if err := press(application, ebiten.KeyEnter); err != nil {
		t.Fatal(err)
	}
	leaveLootMenu(t, application)
	if got := application.eventMachine.Memory[remaining]; got != 7 {
		t.Fatalf("@4A01 = %d after the script ran on, want 10 − 3", got)
	}
	if got := application.eventMachine.Memory[tally]; got != 6 {
		t.Fatalf("@4A39 = %d, want 3 + 3", got)
	}
}

// 只數不在場的：一隻逃掉、一隻還站著（隊伍逃走收場），@6DC8 = 1。
func TestDefeatedCountSkipsTheFoesStillStanding(t *testing.T) {
	application := stageSlumsFight(t, []eclvm.MonsterSpawn{{MonsterID: 1, Count: 2, IconBlock: 4}})
	state := application.tactical
	foes := foeIndexes(state)
	state.leaveBoard(uint8(foes[0]), gamepack.FledState)
	for index := 1; index < len(state.Roster); index++ {
		if state.PartySlot[index] >= 0 {
			state.leaveBoard(uint8(index), gamepack.FledState)
		}
	}
	if err := application.finishCombat(combat.CombatDefeat); err != nil {
		t.Fatal(err)
	}
	if got := application.eventMachine.Memory[defeatedCountAddress]; got != 1 {
		t.Fatalf("@6DC8 = %d with one foe gone and one standing, want 1", got)
	}
}

// 訓練所競技場雇傭兵（ecl3/11 `9EABh`，開價 1 份）：從「IS THIS ACCEPTABLE?」選 YES 走到
// `9F1Ch ADD NPC`，加入的 WARRIOR 錢包是記錄 `+88h` 起的七個 word——原版 dosgolem 那一刻
// `+8Ah` = 1（銀幣）。
func TestHiredMercenaryCarriesThePurseInItsRecord(t *testing.T) {
	receipt := readNPCCombatReceipt(t)
	original, err := hex.DecodeString(receipt.Joined.Record)
	if err != nil {
		t.Fatal(err)
	}
	application := scriptApp(t, 3, 11, 0x9EAB)
	application.eventMachine.Memory[0x6E7A] = 1
	if err := application.continueInitialSearch(nil); err != nil {
		t.Fatal(err)
	}
	if err := selectMenuOption(t, application, "YES"); err != nil {
		t.Fatalf("menu %v: %v", application.cellMenuOptions, err)
	}
	if len(application.state.Party) != 5 || !application.state.Party[4].NPC {
		t.Fatalf("party after YES: %d members", len(application.state.Party))
	}
	member := application.state.Party[4]
	for currency := range member.Money {
		offset := 0x88 + currency*2
		want := uint16(original[offset]) | uint16(original[offset+1])<<8
		if member.Money[currency] != want {
			t.Fatalf("%s carries %v, the original record at the hire has %d in currency %d",
				member.Name, member.Money, want, currency)
		}
	}
	if member.Money[pooltreasure.Silver] != 1 {
		t.Fatalf("%s carries %v, want the one silver piece of the receipt", member.Name, member.Money)
	}
}
