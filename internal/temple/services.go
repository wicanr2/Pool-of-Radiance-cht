package temple

import (
	"fmt"

	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 神殿的服務（overlay-04）。原版的選單列是
// `Heal View Take Pool Share Appraise Exit`（`0C1Ah`），H）EAL 底下是這九項。
//
// 每一項的形狀都一樣：先問「這個人有沒有那個毛病」，沒有就印一句
// `is not …` 並讓玩家確認要不要照做；接著印服務名稱、報價、由 entry 3
//（`00BFh`）收錢；付了才真的處理。

// StatusValue 是角色記錄 `+10Ch` 的狀態值。
const (
	StatusNormal     uint8 = 0
	StatusUnconfirm  uint8 = 1 // 起死回生也收這一個狀態
	StatusDead       uint8 = 6
	StatusStone      uint8 = 7
)

// DiseaseEffectCodes 是「疾病」的六個效果代碼，取自 `DS:0112h..0117h`
//（overlay-04 `031Eh` 用 `[di+111h]` 讀，索引 1..6——那是 Turbo Pascal
// `array[1..6]` 的偏移基底寫法，元素從 `0112h` 起）。
var DiseaseEffectCodes = []uint8{0x1F, 0x22, 0x2B, 0x2C, 0x32, 0x39}

// Service 是一項神殿服務。
type Service struct {
	// ID 是穩定識別，Name 是原版字串。
	ID, Name string
	// Cost 是金幣價。
	Cost int
	// RequiresEffects 非空時，角色身上要有其中任一個代碼才「有這個毛病」。
	RequiresEffects []uint8
	// RequiresStatus 非空時，角色的 `+10Ch` 要等於其中之一。
	RequiresStatus []uint8
	// RemovesEffects 是付款之後要拿掉的代碼。
	RemovesEffects []uint8
	// Refusal 是「他沒有這個毛病」時原版印的那一句。
	Refusal string
}

// Services 逐項照 overlay-04 的進入點順序列出。價錢與代碼都是從那幾支讀出來的。
var Services = []Service{
	{
		ID: "cure-blindness", Name: "Cure Blindness", Cost: 1000,
		RequiresEffects: []uint8{0x21}, RemovesEffects: []uint8{0x21},
		Refusal: "is not blind.",
	},
	{
		ID: "cure-disease", Name: "Cure Disease", Cost: 1000,
		RequiresEffects: DiseaseEffectCodes, RemovesEffects: DiseaseEffectCodes,
		Refusal: "is not Diseased.",
	},
	{ID: "cure-light-wounds", Name: "Cure Light Wounds", Cost: 100},
	{ID: "cure-serious-wounds", Name: "Cure Serious Wounds", Cost: 350},
	{ID: "cure-critical-wounds", Name: "Cure Critical Wounds", Cost: 600},
	{
		ID: "raise-dead", Name: "Raise Dead", Cost: 5500,
		RequiresStatus: []uint8{StatusDead, StatusUnconfirm},
		RemovesEffects: []uint8{0x20, 0x37},
		Refusal:        "is not dead.",
	},
	{
		ID: "neutralize-poison", Name: "Neutralize Poison", Cost: 1000,
		RequiresEffects: []uint8{0x37},
		RemovesEffects:  []uint8{0x37, 0x16, 0x0F},
		Refusal:         "is not poisoned.",
	},
	{
		ID: "remove-curse", Name: "Remove Curse", Cost: 3500,
		RequiresEffects: []uint8{0x24},
		Refusal:         "is not cursed.",
	},
	{
		ID: "stone-to-flesh", Name: "Stone to Flesh", Cost: 2000,
		RequiresStatus: []uint8{StatusStone},
		Refusal:        "is not stoned.",
	},
}

// ServiceByID 找一項服務。
func ServiceByID(id string) (Service, bool) {
	for _, service := range Services {
		if service.ID == id {
			return service, true
		}
	}
	return Service{}, false
}

// Applies 回報這個人有沒有這項服務要治的毛病。兩種前提都沒宣告時一律成立
//（三種傷藥就是這樣——任何人都可以買）。
func (s Service) Applies(character poolsave.Character) bool {
	if len(s.RequiresStatus) > 0 {
		for _, status := range s.RequiresStatus {
			if character.Status == status {
				return true
			}
		}
		return false
	}
	if len(s.RequiresEffects) == 0 {
		return true
	}
	for _, code := range s.RequiresEffects {
		if hasEffect(character.Effects, code) {
			return true
		}
	}
	return false
}

func hasEffect(effects []poolsave.EffectNode, code uint8) bool {
	for _, node := range effects {
		if node.Code == code {
			return true
		}
	}
	return false
}

// hasCode 問一組效果碼裡有沒有某一個。神殿那份「這項服務拿掉哪些」是碼的清單，
// 不是節點。
func hasCode(codes []uint8, code uint8) bool {
	for _, value := range codes {
		if value == code {
			return true
		}
	}
	return false
}

// Serve 收錢並套用一項服務。傷藥走 CureWounds（那三支要擲骰）。
//
// 付款方式與 `00BFh` 相同：先看這個人自己的金幣，不夠才整隊公款出，
// 兩邊**不合併**。
func Serve(state *poolsave.State, partyIndex int, id string, roller Roller) (Result, error) {
	if state == nil || partyIndex < 0 || partyIndex >= len(state.Party) {
		return Result{}, fmt.Errorf("Pool temple party index %d is invalid", partyIndex)
	}
	service, ok := ServiceByID(id)
	if !ok {
		return Result{}, fmt.Errorf("Pool temple has no service %q", id)
	}
	for index, wound := range WoundServices {
		if wound.Name == service.Name {
			return CureWounds(state, partyIndex, index, roller)
		}
	}
	character := &state.Party[partyIndex]
	if !service.Applies(*character) {
		return Result{}, fmt.Errorf("%s %s", character.Name, service.Refusal)
	}
	result := Result{Cost: service.Cost}
	switch {
	case int(character.Money[3]) >= service.Cost:
		character.Money[3] -= uint16(service.Cost)
		result.PaidFrom = "character"
	case uint64(state.PooledMoney[3]) >= uint64(service.Cost):
		state.PooledMoney[3] -= uint32(service.Cost)
		result.PaidFrom = "pool"
	default:
		return Result{}, ErrNotEnoughMoney
	}

	character.Effects = withoutEffects(character.Effects, service.RemovesEffects)
	switch service.ID {
	case "raise-dead":
		// `05E9h`／`05F3h`／`05FDh`：**生命力回到 1**、狀態正常、在場。
		// `+11Bh` 是目前生命值——overlay-09 的士氣拿 `+11Bh × 100 ÷ +32h`
		// 當「還剩幾成血」，所以復活是「活過來但只剩一點」。
		// `0607h` 再把體質減一：起死回生要付一點體質。
		character.Status = StatusNormal
		character.CurrentHP = 1
		if character.Abilities[constitutionAbilityIndex] > 0 {
			character.Abilities[constitutionAbilityIndex]--
		}
		character.MaxHP -= raiseDeadHitPointLoss(*character)
		if character.MaxHP < 1 {
			character.MaxHP = 1
		}
	case "stone-to-flesh":
		// `0976h`／`0980h`／`098Ah`：狀態正常、在場，生命力同樣回到 1。
		character.Status = StatusNormal
		character.CurrentHP = 1
	}
	syncLibraryCharacter(state, *character)
	return result, nil
}

func withoutEffects(effects []poolsave.EffectNode, remove []uint8) []poolsave.EffectNode {
	if len(effects) == 0 || len(remove) == 0 {
		return effects
	}
	kept := effects[:0:0]
	for _, node := range effects {
		if !hasCode(remove, node.Code) {
			kept = append(kept, node)
		}
	}
	return kept
}

// 起死回生的體質欄位索引。原版是記錄 `+14h`。
const constitutionAbilityIndex = 4

// 戰士的職業槽索引。原版比的是 `+96h + 2`（`0659h` 的 `cmp [bp-109h], 2`）。
const fighterClassSlot = 2

// raiseDeadHitPointLoss 重現 `060Fh`..`070Eh`：體質掉一點之後，把每生命骰的
// 體質加成重算一次，扣掉一個骰的份。
//
//	差額   = MaxHP − RawHP                 ; 目前的體質加成總量
//	權重   = Σ 各職業槽（等級 × 每級加成）
//	           戰士槽：體質 − 14
//	           其他槽：體質 > 15 → 2，否則 1
//	每級   = 差額 ÷ 權重                    ; 整數除
//	體質 >= 17 而且有戰士等級 → 不扣
//
// 體質是**扣掉一點之後**的值：原版在 `0607h` 先減，`0677h` 才讀。
func raiseDeadHitPointLoss(character poolsave.Character) int {
	bonus := character.MaxHP - character.RawHP
	if bonus <= 0 {
		return 0
	}
	constitution := character.Abilities[constitutionAbilityIndex]
	weight := 0
	fighterLevels := 0
	for slot, level := range character.ClassLevels {
		if level == 0 {
			continue
		}
		switch {
		case slot == fighterClassSlot:
			fighterLevels = int(level)
			weight += int(level) * (constitution - 14)
		case constitution > 15:
			weight += int(level) * 2
		default:
			weight += int(level)
		}
	}
	if weight <= 0 {
		return 0
	}
	if constitution >= 17 && fighterLevels > 0 {
		return 0
	}
	return bonus / weight
}
