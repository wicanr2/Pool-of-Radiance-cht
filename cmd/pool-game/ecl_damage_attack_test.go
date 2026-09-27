package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// `2Eh DAMAGE` 旗標 bit 7 沒設的那一種是**攻擊**（overlay-03 `2C91h`，spec 084，#90）：
// 旗標是次數，每一次 Roll(1, 人數) 挑人、overlay-24 entry 5（`0C4Dh`）以運算元 5 對他的
// `+111h` 擲命中，中了才吃傷害，之後重擲傷害。指令圖走得到的 48 條裡 23 條是這一種。
//
// 用真的 ECL3 block 14：session 從那一條指令開始跑，第一個邊界就是它，五個運算元照
// damageRequest 從 block 的位元組解；套用那一段是 applyDamageEvent 除了「讓 ECL 繼續」
// 以外的部分（繼續下去會跑到同一個 block 後面別的 DAMAGE，量不到這一條）。
func runECLDamageAt(t *testing.T, address uint16, party []poolsave.Character,
	roller *sequenceRoller) *app {
	t.Helper()
	return runECLDamageIn(t, 3, 14, address, party, roller, false)
}

func runECLDamageIn(t *testing.T, archiveNumber uint8, block uint16, address uint16,
	party []poolsave.Character, roller *sequenceRoller, saves bool) *app {
	t.Helper()
	return runECLDamageInWith(t, archiveNumber, block, address, party, roller, func(application *app) {
		if !saves {
			return
		}
		table, err := gamepack.ReadDOSSavingThrowTable(filepath.Join("..", "..", "Pool of Radiance (1988).zip"))
		if err != nil {
			t.Fatal(err)
		}
		application.savingThrows = table
	})
}

func runECLDamageInWith(t *testing.T, archiveNumber uint8, block uint16, address uint16,
	party []poolsave.Character, roller *sequenceRoller, setup func(*app)) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	catalog, err := gamepack.ReadDOSECLCatalog(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	archive, ok := catalog.Archive(archiveNumber)
	if !ok {
		t.Fatalf("ECL%d is missing", archiveNumber)
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, block, address)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.RunUntilEvent(64, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	event, ok := damageEvent(result)
	if !ok {
		t.Fatalf("the first boundary at %04Xh is not DAMAGE: %+v", address, result.Events)
	}
	application := &app{eventSession: session, eventMachine: session.Machine(),
		eclCatalog: catalog, eclArchive: archiveNumber, roller: roller,
		state: poolsave.State{Schema: poolsave.Schema, Party: party,
			CharacterLibrary: append([]poolsave.Character(nil), party...)}}
	request, err := application.damageRequest(event)
	if err != nil {
		t.Fatal(err)
	}
	setup(application)
	if err := application.applyDamageRequest(request); err != nil {
		t.Fatal(err)
	}
	return application
}

func damageTestMember(dexterity int) poolsave.Character {
	return poolsave.Character{Name: "A", ClassID: "fighter", RaceID: "human",
		Abilities: [6]int{12, 10, 10, dexterity, 10, 10}, MaxHP: 20, CurrentHP: 20,
		ClassLevels: []uint8{0, 0, 1, 0, 0, 0, 0, 0}}
}

// ECL3 block 14 `9A98h`：`2E 01 01 06 00 3C`——一次、1d6、命中值 3Ch。
func TestECLDamageAttackRollsToHit(t *testing.T) {
	for _, tc := range []struct {
		d20  int
		hurt bool
	}{{1, false}, {2, true}} {
		// 1d6 = 4、2B7Eh 的 Roll(1,1)、這一次的 Roll(1,1)、d20、重擲 1d6。
		roller := &sequenceRoller{values: []int{4, 1, 1, tc.d20, 5}}
		application := runECLDamageAt(t, 0x9A98, []poolsave.Character{damageTestMember(10)}, roller)
		if want := []int{6, 1, 1, 20, 6}; !reflect.DeepEqual(roller.asked, want) {
			t.Fatalf("d20 %d: dice asked %v, want %v", tc.d20, roller.asked, want)
		}
		hp := application.state.Party[0].CurrentHP
		if (hp == 16) != tc.hurt || (!tc.hurt && hp != 20) {
			t.Fatalf("d20 %d: hp %d, hurt want %v", tc.d20, hp, tc.hurt)
		}
	}
}

// ECL3 block 14 `B566h`：`2E 03 01 04 01 32`——三次、1d4+1、命中值 32h。比的是目標的 AC：
// 骰 + 32h 必須嚴格大於 `+111h`（`0CA6h` 的 `7E 04`）。
func TestECLDamageAttackComparesTheArmourClass(t *testing.T) {
	member := damageTestMember(18)
	probe := &app{}
	armour, _, err := probe.memberDefenceStats(member, creationArmorClassInternal, creationBaseMovement)
	if err != nil {
		t.Fatal(err)
	}
	tie := armour - 0x32 // 骰 + 32h 剛好等於 AC：落空
	if tie < 2 || tie >= 20 {
		t.Fatalf("armour %d leaves no testable roll", armour)
	}
	// 1d4 → 3（傷害 4）、2B7Eh、然後三次（挑人、d20、重擲 1d4）：落空、中、中。
	roller := &sequenceRoller{values: []int{3, 1, 1, tie, 2, 1, tie + 1, 1, 1, tie + 1, 3}}
	application := runECLDamageAt(t, 0xB566, []poolsave.Character{member}, roller)
	// 第一次落空，第二次吃重擲出來的 3（2 + 1），第三次吃 2（1 + 1）。
	if got := application.state.Party[0].CurrentHP; got != 20-3-2 {
		t.Fatalf("hp %d, want 15 (miss on a tie, then 3 and 2)", got)
	}
}

// 擲豁免那一種、不是全隊的兩支（`2C01h`）：
//   - 運算元 5 的 bit 7 → 目前角色（`DS:5CF0h`），類別 0 不擲豁免。ECL4 block 2 `ADB1h`
//     是 `A0 0F 06 00 80`：打的是目前角色，不是 `2B7Eh` 擲到的那一個。
//   - 否則打 `2B7Eh` 擲到的那一個，而且**一律擲豁免**（`2C3Dh` 這一支不看 bit 5）。
//     ECL2 block 9 `A94Eh` 是 `A0 01 03 00 00`：bit 5 立著，照樣擲 d20。
func TestECLDamageSingleTargetBranches(t *testing.T) {
	party := func() []poolsave.Character {
		first, second := damageTestMember(10), damageTestMember(10)
		second.Name = "B"
		return []poolsave.Character{first, second}
	}
	// 15d6 → 5（測試骰一次給總數）、2B7Eh 擲到第一個人；目前角色是第二個人。
	roller := &sequenceRoller{values: []int{5, 1}}
	current := runECLDamageInWith(t, 4, 2, 0xADB1, party(), roller, func(application *app) {
		application.currentCharacter = 1
	})
	if current.state.Party[0].CurrentHP != 20 || current.state.Party[1].CurrentHP != 15 {
		t.Fatalf("current-character branch hit %d/%d hp", current.state.Party[0].CurrentHP, current.state.Party[1].CurrentHP)
	}
	if want := []int{6, 2}; !reflect.DeepEqual(roller.asked, want) {
		t.Fatalf("current-character branch asked %v, want %v (no save roll)", roller.asked, want)
	}
	// 1d3 → 2、擲到第二個人、d20 擲 1（一定沒過）→ 吃 2。
	roller = &sequenceRoller{values: []int{2, 2, 1}}
	random := runECLDamageIn(t, 2, 9, 0xA94E, party(), roller, true)
	if random.state.Party[1].CurrentHP != 18 || random.state.Party[0].CurrentHP != 20 {
		t.Fatalf("random branch hit %d/%d hp", random.state.Party[0].CurrentHP, random.state.Party[1].CurrentHP)
	}
	if want := []int{3, 2, 20}; !reflect.DeepEqual(roller.asked, want) {
		t.Fatalf("random branch asked %v, want %v (the save is rolled despite bit 5)", roller.asked, want)
	}
}
