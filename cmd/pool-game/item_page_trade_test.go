package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 物品頁的 Trade、84h 與戰鬥外 Use 的那一拍（spec 149，issue #105）。一樣從 Update() 送鍵。

// idleFrame 跑一格沒有按鍵的 Update()（等拍子用）。
func idleFrame(t *testing.T, application *app) {
	t.Helper()
	application.keys = scriptedKeys{}
	if err := application.Update(); err != nil {
		t.Fatal(err)
	}
	runAfterTick(application)
}

func tradeMember(name string, items ...poolsave.Item) poolsave.Character {
	return poolsave.Character{Name: name, ClassID: "fighter", AlignmentID: "lawful-good",
		Abilities: [6]int{12, 12, 12, 12, 12, 12}, CurrentHP: 20, MaxHP: 20, Inventory: items}
}

// Trade（overlay-19 entry 13）：選項列在 Use 後面；穿戴中的先要卸下（entry 20）；
// 挑到對方就接在對方串列最後、自己身上拿掉；ESC 不做事。
func TestItemPageTradeGivesTheItemAway(t *testing.T) {
	application := newItemPageApp(t, tradeMember("A",
		itemOf("SWORD", testTypeSword, true, 0), itemOf("ARROWS", gamepack.ItemTypeArrow, false, 5)))
	application.state.Party = append(application.state.Party,
		tradeMember("B", itemOf("DAGGER", testTypeSword, false, 0)))
	if footer := application.itemPageFooter(); footer != "READY USE TRADE DROP HALVE JOIN EXIT" {
		t.Fatalf("footer %q", footer)
	}
	pressAll(t, application, ebiten.KeyT)
	if !strings.Contains(application.equipment.message, "MUST BE UNREADIED") ||
		application.equipment.page.stage != itemPagePicking {
		t.Fatalf("trading a readied sword: %q", application.equipment.message)
	}
	cursorTo(t, application, 1)
	pressAll(t, application, ebiten.KeyT)
	if application.equipment.page.stage != itemPageTrade ||
		!strings.Contains(application.equipment.message, "TRADE WITH WHOM?") ||
		application.equipment.page.target != 0 {
		t.Fatalf("T: stage %d target %d %q", application.equipment.page.stage,
			application.equipment.page.target, application.equipment.message)
	}
	pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyEscape)
	if len(application.state.Party[0].Inventory) != 2 || application.equipment.page.stage != itemPagePicking {
		t.Fatal("ESC traded the arrows")
	}
	pressAll(t, application, ebiten.KeyT)
	if application.equipment.page.target != 0 {
		t.Fatalf("no trade yet, the picker starts at %d, want self", application.equipment.page.target)
	}
	pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyEnter)
	a, b := application.state.Party[0], application.state.Party[1]
	if len(a.Inventory) != 1 || a.Inventory[0].Name != "SWORD" ||
		len(b.Inventory) != 2 || b.Inventory[1].Name != "ARROWS" ||
		b.Inventory[1].Raw[gamepack.ItemCountOffset] != 5 {
		t.Fatalf("after trading: A %+v, B %+v", a.Inventory, b.Inventory)
	}
	if !application.equipmentOpen || application.equipment.message != "" {
		t.Fatalf("after trading: open %v, %q", application.equipmentOpen, application.equipment.message)
	}
}

// entry 9（`274Fh`）：對方已有 16 件，或 `+102h`（含錢）加上這一件超過力量表 + 1500，
// 印 "Overloaded"，東西不動。
func TestItemPageTradeRefusesAnOverloadedRecipient(t *testing.T) {
	sword := itemOf("SWORD", testTypeSword, false, 0)
	sword.Raw[gamepack.ItemWeightOffset] = 60
	application := newItemPageApp(t, tradeMember("A", sword))
	full := tradeMember("FULL")
	for len(full.Inventory) < 16 {
		full.Inventory = append(full.Inventory, itemOf("DAGGER", testTypeSword, false, 0))
	}
	weak := tradeMember("WEAK")
	weak.Abilities[gamepack.AbilityStrength] = 3 // 力量表 −350 → 上限 1150
	weak.Money[0] = 1091                         // 1091 + 60 > 1150；1090 就剛好收得下
	application.state.Party = append(application.state.Party, full, weak)
	for _, target := range []int{1, 2} {
		pressAll(t, application, ebiten.KeyT)
		for guard := 0; guard < 8 && application.equipment.page.target != target; guard++ {
			pressAll(t, application, ebiten.KeyArrowDown)
		}
		pressAll(t, application, ebiten.KeyEnter)
		if !strings.Contains(application.equipment.message, "OVERLOADED") ||
			len(application.state.Party[0].Inventory) != 1 {
			t.Fatalf("trading to %s: %q", application.state.Party[target].Name, application.equipment.message)
		}
	}
	if len(application.state.Party[1].Inventory) != 16 || len(application.state.Party[2].Inventory) != 0 {
		t.Fatal("an overloaded recipient got the sword")
	}
	// 15 件還收得下第 16 件（檢查在加入之前，`2767h` 的 `> 0Fh`）。
	application.state.Party[1].Inventory = application.state.Party[1].Inventory[:15]
	pressAll(t, application, ebiten.KeyT)
	for guard := 0; guard < 8 && application.equipment.page.target != 1; guard++ {
		pressAll(t, application, ebiten.KeyArrowDown)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if len(application.state.Party[1].Inventory) != 16 || len(application.state.Party[0].Inventory) != 0 {
		t.Fatalf("15 items could not take a 16th: %q", application.equipment.message)
	}
}

// ADD NPC 帶進來、`+84h` 位元 7 立著而且 `+10Dh` 非 0 的 NPC 沒有 Trade（`0FF6h..1015h`）。
func TestItemPageTradeHiddenForAMoraleNPC(t *testing.T) {
	npc := tradeMember("NPC", itemOf("SWORD", testTypeSword, false, 0))
	npc.NPC = true
	npc.Record = make([]byte, 0x11d)
	npc.Record[0x84], npc.Record[0x10d] = 0xb3, 1
	application := newItemPageApp(t, npc)
	if footer := application.itemPageFooter(); strings.Contains(footer, "TRADE") {
		t.Fatalf("NPC footer %q", footer)
	}
	pressAll(t, application, ebiten.KeyT)
	if application.equipment.page.stage != itemPagePicking {
		t.Fatal("T opened a trade for a morale NPC")
	}
}

// " Use" 的兩道門（`0F8Fh`、`0F9Bh`）：記錄 `+10Dh` 為 0（倒下的）沒有 Use；
// `[4933h]+1CAh`（ECL `@49E5`，反魔法）非 0 也沒有。
func TestItemPageUseNeedsAStandingMemberOutsideAntiMagic(t *testing.T) {
	down := tradeMember("A", itemOf("SWORD", testTypeSword, false, 0))
	down.Status = gamepack.UnconsciousState
	application := newItemPageApp(t, down)
	if footer := application.itemPageFooter(); footer != "READY TRADE DROP HALVE JOIN EXIT" {
		t.Fatalf("unconscious member footer %q", footer)
	}
	application.state.Party[0].Status = 0
	application.eventMachine = &eclvm.Machine{Memory: map[uint16]uint16{antiMagicAddress: 1}}
	if footer := application.itemPageFooter(); footer != "READY TRADE DROP HALVE JOIN EXIT" {
		t.Fatalf("anti-magic footer %q", footer)
	}
	application.eventMachine.Memory[antiMagicAddress] = 0
	if footer := application.itemPageFooter(); footer != "READY USE TRADE DROP HALVE JOIN EXIT" {
		t.Fatalf("footer after the shell is gone %q", footer)
	}
}

// 戰鬥外 Use：entry 8 先印 "<名字> uses an item" 與物品名，等一拍（遊戲速度 × 225 ms）
// 才進挑對象；那一拍裡不收鍵。
func TestItemPageUseWaitsABeatAfterUsesAnItem(t *testing.T) {
	items := premadeItems(t, "chrdatd1.itm")
	potion := items[findItem(t, items, "Potion of Healing")]
	application := newItemPageApp(t, tradeMember("HAPLO", potion))
	application.gameSpeed = 2
	pressAll(t, application, ebiten.KeyR, ebiten.KeyU)
	page := application.equipment.page
	if page.stage != itemPageUsesNotice ||
		application.equipment.message != "HAPLO USES AN ITEM  "+strings.TrimSpace(potion.Name) {
		t.Fatalf("U: stage %d, %q", page.stage, application.equipment.message)
	}
	frames := 1
	pressAll(t, application, ebiten.KeyEscape)
	for guard := 0; guard < 600 && application.equipment.page.stage == itemPageUsesNotice; guard++ {
		idleFrame(t, application)
		frames++
	}
	if application.equipment.page.stage != itemPageTarget || !application.equipmentOpen {
		t.Fatalf("after the beat: stage %d, open %v", application.equipment.page.stage, application.equipmentOpen)
	}
	if want := application.speedDelayTicks(); frames != want {
		t.Fatalf("the beat lasted %d frames, want %d", frames, want)
	}
}

// 84h（overlay-12 entry 123）拿原版的兩把長劍：ITEM5.DAX block 35 的 `+3Dh` 52h
// （限陣營 2，守序邪惡，錯了受 5 點）與 ITEM4.DAX block 29 的 F0h（限守序善良，受 15 點）。
func TestAlignedSwordHurtsTheWrongWearer(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	evil, err := gamepack.ReadDOSTreasureItemBlock(zipPath, 5, 35)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	good, err := gamepack.ReadDOSTreasureItemBlock(zipPath, 4, 29)
	if err != nil {
		t.Fatal(err)
	}
	sword := func(record gamepack.TreasureItemRecord) poolsave.Item {
		if record.Raw[gamepack.ItemEffectOffset] != gamepack.AlignedWearEffectCode {
			t.Fatalf("%s has +3Eh %#02x", record.Name, record.Raw[gamepack.ItemEffectOffset])
		}
		return poolsave.Item{Name: record.Name, Raw: append([]byte(nil), record.Raw[:]...)}
	}

	// 守序善良拿 52h 那把：卸下、受 5 點、印 "from Magic"。
	application := newItemPageApp(t, tradeMember("HAPLO", sword(evil[0])))
	pressAll(t, application, ebiten.KeyR)
	member := application.state.Party[0]
	if member.Inventory[0].Raw[gamepack.ItemReadiedOffset] != 0 || member.CurrentHP != 15 ||
		application.equipment.message != "HAPLO TAKES 5 POINTS OF DAMAGE FROM MAGIC" {
		t.Fatalf("lawful good and the 52h sword: readied %d, hp %d, %q",
			member.Inventory[0].Raw[gamepack.ItemReadiedOffset], member.CurrentHP, application.equipment.message)
	}

	// 守序邪惡拿同一把：照常裝上。
	wearer := tradeMember("EVIL", sword(evil[0]))
	wearer.AlignmentID = "lawful-evil"
	application = newItemPageApp(t, wearer)
	pressAll(t, application, ebiten.KeyR)
	if member := application.state.Party[0]; member.Inventory[0].Raw[gamepack.ItemReadiedOffset] != 1 ||
		member.CurrentHP != 20 || application.equipment.message != "" {
		t.Fatalf("lawful evil and the 52h sword: %+v %q", member, application.equipment.message)
	}

	// 混亂善良、10 點生命拿 F0h 那把：受 15 點，赤字 5 → 瀕死（spec 084）。
	wearer = tradeMember("CHAOS", sword(good[6]))
	wearer.AlignmentID, wearer.CurrentHP = "chaotic-good", 10
	application = newItemPageApp(t, wearer)
	pressAll(t, application, ebiten.KeyR)
	member = application.state.Party[0]
	if member.CurrentHP != 0 || member.Status != gamepack.DyingState ||
		application.equipment.message != "CHAOS TAKES 15 POINTS OF DAMAGE FROM MAGIC  CHAOS GOES DOWN, AND IS DYING" {
		t.Fatalf("chaotic good and the F0h sword: hp %d status %d %q", member.CurrentHP, member.Status,
			application.equipment.message)
	}
}
