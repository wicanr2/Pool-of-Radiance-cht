package temple_test

import (
	"testing"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/temple"
)

type fixedRoll int

func (r fixedRoll) Roll(count, sides int) int { return int(r) }

// 九項服務的名稱與價錢逐項釘住。這些數字是從 overlay-04 那六支各自的
// `mov ax, imm` 讀出來的，不是抄手冊。
func TestTempleServicesMatchTheOriginal(t *testing.T) {
	want := []struct {
		id   string
		name string
		cost int
	}{
		{"cure-blindness", "Cure Blindness", 1000},
		{"cure-disease", "Cure Disease", 1000},
		{"cure-light-wounds", "Cure Light Wounds", 100},
		{"cure-serious-wounds", "Cure Serious Wounds", 350},
		{"cure-critical-wounds", "Cure Critical Wounds", 600},
		{"raise-dead", "Raise Dead", 5500},
		{"neutralize-poison", "Neutralize Poison", 1000},
		{"remove-curse", "Remove Curse", 3500},
		{"stone-to-flesh", "Stone to Flesh", 2000},
	}
	if len(temple.Services) != len(want) {
		t.Fatalf("有 %d 項服務，原版是 %d 項", len(temple.Services), len(want))
	}
	for index, expected := range want {
		got := temple.Services[index]
		if got.ID != expected.id || got.Name != expected.name || got.Cost != expected.cost {
			t.Errorf("第 %d 項是 %s／%q／%d，應該是 %s／%q／%d",
				index, got.ID, got.Name, got.Cost, expected.id, expected.name, expected.cost)
		}
	}
	// 三種傷藥要與既有的 WoundServices 完全一致，否則兩份表會漂開。
	for index, wound := range temple.WoundServices {
		service := temple.Services[2+index]
		if service.Name != wound.Name || service.Cost != wound.Cost {
			t.Errorf("傷藥第 %d 項兩份表不一致：%q/%d 對 %q/%d",
				index, service.Name, service.Cost, wound.Name, wound.Cost)
		}
	}
}

// 六個疾病代碼取自 `DS:0112h..0117h`。
func TestDiseaseEffectCodesMatchTheTable(t *testing.T) {
	want := []uint8{0x1F, 0x22, 0x2B, 0x2C, 0x32, 0x39}
	if len(temple.DiseaseEffectCodes) != len(want) {
		t.Fatalf("有 %d 個疾病代碼", len(temple.DiseaseEffectCodes))
	}
	for index, code := range want {
		if temple.DiseaseEffectCodes[index] != code {
			t.Errorf("第 %d 個是 %02Xh，應該是 %02Xh", index, temple.DiseaseEffectCodes[index], code)
		}
	}
}

func partyState(character poolsave.Character) *poolsave.State {
	state := poolsave.NewState()
	state.Party = []poolsave.Character{character}
	return &state
}

// 沒有那個毛病就治不了——原版印 `is not blind.` 之類的一句。
func TestServiceRefusesWhenThereIsNothingToCure(t *testing.T) {
	state := partyState(poolsave.Character{Name: "A", MaxHP: 10, CurrentHP: 10,
		Money: [7]uint16{3: 9999}})
	if _, err := temple.Serve(state, 0, "cure-blindness", fixedRoll(1)); err == nil {
		t.Fatal("沒有失明卻治得了")
	}
	// 錢不能被扣掉。
	if got := state.Party[0].Money[3]; got != 9999 {
		t.Errorf("治不了卻扣了錢，剩 %d", got)
	}
	// 正對照：真的瞎了就治得了，而且代碼被拿掉。
	state.Party[0].Effects = poolsave.PermanentEffects(0x21, 0x24)
	result, err := temple.Serve(state, 0, "cure-blindness", fixedRoll(1))
	if err != nil {
		t.Fatalf("治療失明：%v", err)
	}
	if result.Cost != 1000 || state.Party[0].Money[3] != 8999 {
		t.Errorf("收了 %d，剩 %d", result.Cost, state.Party[0].Money[3])
	}
	if len(state.Party[0].Effects) != 1 || state.Party[0].Effects[0].Code != 0x24 {
		t.Errorf("效果剩 %v，應該只拿掉 21h", state.Party[0].Effects)
	}
}

// 中毒一次拿掉三個代碼（`07C9h`／`07DFh`／`07F5h`）。
func TestNeutralizePoisonRemovesThreeCodes(t *testing.T) {
	state := partyState(poolsave.Character{Name: "A", MaxHP: 10, CurrentHP: 10,
		Money: [7]uint16{3: 2000}, Effects: poolsave.PermanentEffects(0x37, 0x16, 0x0F, 0x21)})
	if _, err := temple.Serve(state, 0, "neutralize-poison", fixedRoll(1)); err != nil {
		t.Fatalf("解毒：%v", err)
	}
	if len(state.Party[0].Effects) != 1 || state.Party[0].Effects[0].Code != 0x21 {
		t.Errorf("效果剩 %v，應該只留 21h", state.Party[0].Effects)
	}
}

// 起死回生：狀態回到正常、體質掉一點，生命力上限跟著重算。
func TestRaiseDeadCostsAPointOfConstitution(t *testing.T) {
	character := poolsave.Character{
		Name: "A", MaxHP: 30, CurrentHP: 0, RawHP: 24,
		Status: temple.StatusDead, Money: [7]uint16{3: 6000},
	}
	character.Abilities[4] = 16
	character.ClassLevels = []uint8{0, 0, 6, 0, 0, 0, 0, 0} // 六級戰士
	state := partyState(character)
	if _, err := temple.Serve(state, 0, "raise-dead", fixedRoll(1)); err != nil {
		t.Fatalf("起死回生：%v", err)
	}
	revived := state.Party[0]
	if revived.Status != temple.StatusNormal {
		t.Errorf("狀態是 %d，應該回到正常", revived.Status)
	}
	if revived.Abilities[4] != 15 {
		t.Errorf("體質是 %d，起死回生要付一點", revived.Abilities[4])
	}
	// `05E9h` 把 `+11Bh`（目前生命值）寫成 1：活過來但只剩一點。
	if revived.CurrentHP != 1 {
		t.Errorf("復活之後是 %d 點生命力，原版寫的是 1", revived.CurrentHP)
	}
	// 體質 15 的戰士每級加成是 15 − 14 = 1，六級共 6；目前的加成總量是
	// 30 − 24 = 6，所以每級 1 點，扣掉一個骰的份。
	if revived.MaxHP != 29 {
		t.Errorf("生命力上限是 %d，應該是 29", revived.MaxHP)
	}
	// 負對照：體質 17 以上的戰士不扣（`06E3h` 的 `cmp +14h, 11h`）。
	tough := character
	tough.Abilities[4] = 18
	tough.Money = [7]uint16{3: 6000}
	state = partyState(tough)
	if _, err := temple.Serve(state, 0, "raise-dead", fixedRoll(1)); err != nil {
		t.Fatalf("起死回生（高體質）：%v", err)
	}
	if got := state.Party[0].MaxHP; got != 30 {
		t.Errorf("體質 17 的戰士生命力上限變成 %d，不該扣", got)
	}
}

// 石化只看狀態，不看效果串列（`091Ch` 比 `+10Ch` 是不是 7）。
func TestStoneToFleshLooksAtTheStatus(t *testing.T) {
	state := partyState(poolsave.Character{Name: "A", MaxHP: 10, CurrentHP: 10,
		Status: temple.StatusStone, Money: [7]uint16{3: 3000}})
	if _, err := temple.Serve(state, 0, "stone-to-flesh", fixedRoll(1)); err != nil {
		t.Fatalf("石化解除：%v", err)
	}
	if state.Party[0].Status != temple.StatusNormal {
		t.Errorf("狀態是 %d", state.Party[0].Status)
	}
	// `098Ah` 一樣把目前生命值寫成 1。
	if state.Party[0].CurrentHP != 1 {
		t.Errorf("解石化之後是 %d 點生命力，原版寫的是 1", state.Party[0].CurrentHP)
	}
	if state.Party[0].Money[3] != 1000 {
		t.Errorf("剩 %d 金幣，應該收 2000", state.Party[0].Money[3])
	}
}

// 錢不夠時個人與公款不合併（`00BFh` 的兩段判斷）。
func TestServiceNeverCombinesTheTwoPurses(t *testing.T) {
	state := partyState(poolsave.Character{Name: "A", MaxHP: 10, CurrentHP: 10,
		Status: temple.StatusStone, Money: [7]uint16{3: 1500}})
	state.PooledMoney[3] = 1500
	if _, err := temple.Serve(state, 0, "stone-to-flesh", fixedRoll(1)); err == nil {
		t.Fatal("兩邊各 1500 湊成 3000 付掉了，原版不合併")
	}
	// 正對照：公款自己夠就付得出來。
	state.PooledMoney[3] = 2000
	if _, err := temple.Serve(state, 0, "stone-to-flesh", fixedRoll(1)); err != nil {
		t.Fatalf("公款付款：%v", err)
	}
	if state.PooledMoney[3] != 0 || state.Party[0].Money[3] != 1500 {
		t.Errorf("公款剩 %d、個人剩 %d", state.PooledMoney[3], state.Party[0].Money[3])
	}
}
