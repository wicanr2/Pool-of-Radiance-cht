package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 營地施法掛效果（spec 098〈營地施法〉，issue #100）。全部從 Update() 送鍵：探索畫面按 C、
// 挑施法者、挑法術；參數表 +7 是 2 的才多一步挑對象。

// campCastApp 是探索畫面上的隊伍，帶著原版的參數表與處理常式。
func campCastApp(t *testing.T, members ...poolsave.Character) *app {
	t.Helper()
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	types, err := gamepack.ReadDOSItemTypeTable(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	parameters, err := gamepack.ReadDOSSpellParameters(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	caster, err := gamepack.ReadDOSSpellCaster(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	application := &app{mode: modeAdventure, introDone: true, language: languageEnglish,
		keys: scriptedKeys{}, roller: fixedRoller{7}, itemTypes: types,
		spellParameters: parameters, spellCaster: caster}
	application.state = poolsave.State{Schema: poolsave.Schema, Party: members}
	return application
}

// campCaster 是牧師、法師各 level 級的隊員，記著 spells。
func campCaster(name string, level uint8, spells ...uint8) poolsave.Character {
	levels := make([]uint8, gamepack.ClassThac0ClassCount)
	levels[gamepack.ClassSlotCleric] = level
	levels[gamepack.ClassSlotMagicUser] = level
	member := poolsave.Character{Name: name, Age: 20, CurrentHP: 20, MaxHP: 20,
		ClassLevels: levels, Abilities: [6]int{12, 12, 12, 12, 12, 12},
		Memorised: make([]uint8, gamepack.MemorisedSpellSlots)}
	copy(member.Memorised, spells)
	return member
}

// castInCamp 按 C、挑第 caster 個人、挑編號 spell 的那一條；要挑對象的挑第 target 個。
// 回傳放完之後那一句訊息。
func castInCamp(t *testing.T, application *app, caster int, spell uint8, target int) string {
	t.Helper()
	pressAll(t, application, ebiten.KeyC)
	if !application.fieldCastOpen || application.fieldCastStage != fieldCastPickCaster {
		t.Fatalf("C did not open the caster picker (open %v stage %d)",
			application.fieldCastOpen, application.fieldCastStage)
	}
	for guard := 0; guard <= len(application.state.Party) && application.fieldCastCursor != caster; guard++ {
		pressAll(t, application, ebiten.KeyArrowDown)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if application.fieldCastStage != fieldCastPickSpell {
		t.Fatalf("picking caster %d: stage %d (%q)", caster, application.fieldCastStage,
			application.fieldCastMessage)
	}
	want := -1
	for index, option := range application.fieldCastOptions {
		if option.ID == spell {
			want = index
			break
		}
	}
	if want < 0 {
		t.Fatalf("spell %d is not on the list %+v", spell, application.fieldCastOptions)
	}
	for guard := 0; guard <= len(application.fieldCastOptions) && application.fieldCastCursor != want; guard++ {
		pressAll(t, application, ebiten.KeyArrowDown)
	}
	pressAll(t, application, ebiten.KeyEnter)
	if application.fieldCastStage == fieldCastPickTarget {
		for guard := 0; guard <= len(application.state.Party) && application.fieldCastCursor != target; guard++ {
			pressAll(t, application, ebiten.KeyArrowDown)
		}
		pressAll(t, application, ebiten.KeyEnter)
	}
	if application.fieldCastStage != fieldCastPickCaster {
		t.Fatalf("the cast did not come back to the caster picker: stage %d", application.fieldCastStage)
	}
	message := application.fieldCastMessage
	pressAll(t, application, ebiten.KeyEscape)
	if application.fieldCastOpen {
		t.Fatal("ESC did not close the cast page")
	}
	return message
}

func memberEffect(member poolsave.Character, code uint8) (gamepack.EffectNode, bool) {
	list := combatEffects(member.Effects)
	index, ok := list.IndexOf(code)
	if !ok {
		return gamepack.EffectNode{}, false
	}
	return list[index], true
}

// 護盾（19，+7 = 1 施法者自己）：營地施的掛在身上，持續 +5 × 等級 = 15、節點 +3 是等級。
// 進戰場帶著它（tactical.go 抄 Effects），魔法飛彈打施法者歸零（群組 6）。沒施的對照組照打。
func TestCampShieldCarriesIntoCombatAndStopsMagicMissile(t *testing.T) {
	for _, cast := range []bool{false, true} {
		application := bootCityParty(t, dosZIPForTests)
		application.state.Party[0] = campCaster("A", 3, 19)
		if cast {
			castInCamp(t, application, 0, 19, 0)
			node, ok := memberEffect(application.state.Party[0], gamepack.ShieldEffectCode)
			if !ok || node.Duration() != 15 || node.Magnitude() != 3 {
				t.Fatalf("camp shield: %v (%+v)", ok, node)
			}
			if application.state.Party[0].Memorised[0] != 0 {
				t.Fatal("the memorised shield was not spent")
			}
		}
		if err := application.enterTacticalPreview(); err != nil {
			t.Fatalf("enter combat: %v", err)
		}
		state := application.tactical
		slot := -1
		for index, party := range state.PartySlot {
			if party == 0 {
				slot = index
				break
			}
		}
		if slot < 0 {
			t.Fatal("the caster is not on the board")
		}
		if state.hasEffect(slot, gamepack.ShieldEffectCode) != cast {
			t.Fatalf("shield %v: board effects %v", cast, state.Effects[slot])
		}
		damage := application.damageAfterSave(state, uint8(slot), gamepack.SpellIDMagicMissile, 6)
		if (damage == 0) != cast {
			t.Fatalf("shield %v: magic missile did %d", cast, damage)
		}
	}
}

// 閱讀魔法（18，+7 = 1）：營地施完，同一個人打開藏字的法師卷軸就揭開（spec 144）；沒施的揭不開。
func TestCampReadMagicRevealsTheScroll(t *testing.T) {
	for _, cast := range []bool{false, true} {
		member := campCaster("A", 3, 18)
		member.Inventory = []poolsave.Item{scrollItem(0x02, gamepack.SpellIDMagicMissile)}
		application := campCastApp(t, member)
		if cast {
			castInCamp(t, application, 0, 18, 0)
			if _, ok := memberEffect(application.state.Party[0], gamepack.ReadMagicEffectCode); !ok {
				t.Fatalf("camp read magic attached nothing: %v", application.state.Party[0].Effects)
			}
		}
		pressAll(t, application, ebiten.KeyI)
		if !application.equipmentOpen {
			t.Fatal("I did not open the item page")
		}
		pressAll(t, application, ebiten.KeyU)
		revealed := application.equipment.page.stage == itemPageScroll
		if revealed != cast {
			t.Fatalf("read magic %v: scroll revealed %v (%q)", cast, revealed, application.equipment.message)
		}
	}
}

// 祝福（1，+7 = 4 整隊）：不問對象，每個人掛 01h、持續 6；戰鬥外走時間照原版倒數到期。
// 隱形（30，+7 = 2）：挑一個人，持續 0（永久），時間走多久都在。
func TestCampBlessCoversThePartyAndExpiresWithTime(t *testing.T) {
	application := campCastApp(t, campCaster("A", 3, gamepack.SpellIDBless, 30),
		campCaster("B", 1), campCaster("C", 1))
	message := castInCamp(t, application, 0, gamepack.SpellIDBless, 0)
	if !strings.Contains(message, "3") {
		t.Fatalf("bless message %q does not count three", message)
	}
	for index, member := range application.state.Party {
		node, ok := memberEffect(member, gamepack.BlessEffectCode)
		if !ok || node.Duration() != 6 || node.Magnitude() != 3 {
			t.Fatalf("member %d after camp bless: %v %+v", index, ok, node)
		}
	}
	castInCamp(t, application, 0, 30, 2)
	if _, ok := memberEffect(application.state.Party[2], 0x19); !ok {
		t.Fatalf("invisibility on C: %v", application.state.Party[2].Effects)
	}
	if _, ok := memberEffect(application.state.Party[1], 0x19); ok {
		t.Fatal("invisibility landed on B, who was not picked")
	}
	application.advancePartyEffects(5)
	if _, ok := memberEffect(application.state.Party[1], gamepack.BlessEffectCode); !ok {
		t.Fatal("bless expired after 5 of its 6 minutes")
	}
	application.advancePartyEffects(1)
	for index, member := range application.state.Party {
		if _, ok := memberEffect(member, gamepack.BlessEffectCode); ok {
			t.Fatalf("bless on member %d outlived its 6 minutes", index)
		}
	}
	if _, ok := memberEffect(application.state.Party[2], 0x19); !ok {
		t.Fatal("permanent invisibility expired with time")
	}
}

// 急速（48，+7 = 4）：`2724h` 的額度是施法者等級、先解緩速，`08BCh` 之後派發群組 18——
// 營地一樣當場老一歲。被解掉緩速的那一位不急速、不老。
func TestCampHasteAgesThePartyAndCancelsSlow(t *testing.T) {
	slowed := campCaster("B", 1)
	slowed.Effects = storedEffects(gamepack.EffectList{
		gamepack.NewEffectNode(gamepack.SlowEffectCode, 10, 5, false)})
	application := campCastApp(t, campCaster("A", 2, gamepack.SpellIDHaste), slowed,
		campCaster("C", 1))
	castInCamp(t, application, 0, gamepack.SpellIDHaste, 0)
	party := application.state.Party
	if _, ok := memberEffect(party[0], gamepack.HasteEffectCode); !ok || party[0].Age != 21 {
		t.Fatalf("caster: hasted %v age %d", ok, party[0].Age)
	}
	if _, ok := memberEffect(party[1], gamepack.SlowEffectCode); ok {
		t.Fatal("haste did not cancel B's slow")
	}
	if _, ok := memberEffect(party[1], gamepack.HasteEffectCode); ok || party[1].Age != 20 {
		t.Fatalf("B was cured of slow but also hasted %v / aged to %d", ok, party[1].Age)
	}
	// 額度 2：A 用一格、B（被解掉的）也扣一格，C 輪不到。
	if _, ok := memberEffect(party[2], gamepack.HasteEffectCode); ok {
		t.Fatal("C was hasted past the quota of 2")
	}
}

// 存檔讀檔之後效果還在：串列長在角色記錄上（spec 069），存檔就是那一份。
func TestCampEffectsSurviveSaveAndLoad(t *testing.T) {
	// 存檔要過 Validate，所以用建好的隊伍（頭像、種族那些都齊），只換記憶與等級。
	application := bootCityParty(t, dosZIPForTests)
	caster := campCaster("A", 3, 19)
	application.state.Party[0].ClassLevels = caster.ClassLevels
	application.state.Party[0].Memorised = caster.Memorised
	castInCamp(t, application, 0, 19, 0)
	path := filepath.Join(t.TempDir(), "state.json")
	if err := poolsave.WriteAtomic(path, application.state); err != nil {
		t.Fatal(err)
	}
	loaded, err := poolsave.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	node, ok := memberEffect(loaded.Party[0], gamepack.ShieldEffectCode)
	if !ok || node.Duration() != 15 {
		t.Fatalf("after save and load: %v %+v", ok, node)
	}
}

// 參數表 +7 為 0 的（魔法飛彈）在營地不掛任何東西。
func TestCampCombatOnlySpellAttachesNothing(t *testing.T) {
	application := campCastApp(t, campCaster("A", 3, gamepack.SpellIDMagicMissile), campCaster("B", 1))
	castInCamp(t, application, 0, gamepack.SpellIDMagicMissile, 1)
	for index, member := range application.state.Party {
		if len(member.Effects) != 0 {
			t.Fatalf("member %d got %v from magic missile in camp", index, member.Effects)
		}
	}
}

// 物品頁的 Use 在戰鬥外也是 overlay-22 entry 5（spec 149）：閱讀魔法的魔杖掛上 10h，
// 同一個人接著打開藏字卷軸就揭開。沒用魔杖的對照組揭不開。
func TestCampWandOfReadMagicRevealsTheScroll(t *testing.T) {
	for _, use := range []bool{false, true} {
		wand := itemOf("WAND", testTypeSword, true, 0)
		wand.Raw[gamepack.AIItemChargesOffset] = 2
		wand.Raw[gamepack.AIItemSpellOffset] = 18
		member := campCaster("A", 3)
		member.Inventory = []poolsave.Item{wand, scrollItem(0x02, gamepack.SpellIDMagicMissile)}
		application := campCastApp(t, member)
		pressAll(t, application, ebiten.KeyI)
		if use {
			pressAll(t, application, ebiten.KeyU)
			if application.equipment.page.stage == itemPageTarget {
				pressAll(t, application, ebiten.KeyEnter)
			}
			if _, ok := memberEffect(application.state.Party[0], gamepack.ReadMagicEffectCode); !ok {
				t.Fatalf("wand of read magic attached nothing (%q)", application.equipment.message)
			}
		}
		pressAll(t, application, ebiten.KeyArrowDown, ebiten.KeyU)
		if revealed := application.equipment.page.stage == itemPageScroll; revealed != use {
			t.Fatalf("wand used %v: scroll revealed %v (%q)", use, revealed, application.equipment.message)
		}
	}
}
