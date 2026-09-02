package gamepack

import "fmt"

// 施法（spec 098）。原版每個法術一支處理常式（spec 073 的 67 格派發表），
// 大多數只是算出一個數字再交給共用的 `08BCh`。這裡照那個形狀接：每一條
// 都標出反組譯的位址，**沒有讀過的法術硬失敗**，不用規則書補。
//
// 硬失敗是刻意的：一個沒實作的法術若安靜地不做事，在報表上與「正確地
// 不做事」分不出來。

// CastEffect 是一次施法算出來的東西。
type CastEffect struct {
	// Damage 是傷害；`08BCh` 收到 0 就整段跳過，不套用也不擲豁免。
	Damage int
	// Heal 是治療量。
	Heal int
	// EffectCode 是要掛進效果串列的代碼（參數表 `+0Ah`，spec 069）。
	EffectCode uint8
	// WholeSide 為真代表效果作用在整邊，不是單一目標（例如祝福術）。
	WholeSide bool
	// Area 為真代表這是範圍法術（處理常式先寫 `DS:677Eh = 1`）。
	Area bool
	// SleepBudget 是催眠術能放倒的生命骰總量（`DS:47A6h`）。大於零時
	// 呼叫端要依 SleepHitDiceCost 逐個目標扣，扣得動的就睡著。
	SleepBudget int
}

// SleepEffectCode 是催眠術掛上去的效果碼（`15DEh` 推的 35h）。
// overlay-15 的名稱鏈把它叫 "Funky--"（spec 069）。
const SleepEffectCode = 0x35

// SleepHitDiceCost 是放倒一個目標要花多少額度（overlay-22 `1553h..15AFh`）。
//
// 依目標的 `+73h`（最高職業等級，怪物就是生命骰）分段。第 5 段還要看
// `+2Eh`：為零花 10，否則花 20。六段以上一律 20，等於放不倒。
func SleepHitDiceCost(hitDice int, fifthBandFlag uint8) int {
	switch {
	case hitDice <= 1:
		return 1
	case hitDice == 2:
		return 2
	case hitDice == 3:
		return 4
	case hitDice == 4:
		return 6
	case hitDice == 5:
		if fifthBandFlag == 0 {
			return 10
		}
		return 20
	}
	return 20
}

// CasterLevelFor 是 overlay-25 `26F8h`：參數表 `+0` 決定讀哪一個職業等級，
// 物品效果一律 12 級；`DS:6CB3h` 非零時（人物檢視畫面，也就是戰術地圖之外）
// 等級一律當成 6。
func CasterLevelFor(parameters SpellParameters, clericLevel, magicUserLevel int,
	outsideCombat bool) int {
	level := 0
	switch parameters.Source() {
	case SpellSourceCleric:
		level = clericLevel
	case SpellSourceMagicUser:
		level = magicUserLevel
	case SpellSourceItem:
		return 12
	}
	if outsideCombat {
		return 6
	}
	return level
}

// 已經讀出處理常式的法術編號。每一條都標了 overlay-22 裡的位址。
const (
	SpellIDBless          = 1  // 0FF5h
	SpellIDCureLightWound = 3  // 1051h
	SpellIDBurningHands   = 9  // 1178h
	SpellIDMagicMissile   = 15 // 1429h
	SpellIDShockingGrasp  = 20 // 14BFh
	SpellIDSleep          = 21 // 1513h
	SpellIDFireball       = 47 // 262Eh
	SpellIDLightningBolt  = 51 // 2B75h
)

// SpellCaster 把讀出來的處理常式與那一批純泛型的收在一起。
//
// 純泛型的那批沒有自己的算法：`08BCh` 依參數表判射程、豁免，再把
// 參數表 `+0Ah` 的效果碼掛上去（spec 074／098）。所以認得出版型就等於
// 接完一整批，不必逐支讀。
type SpellCaster struct {
	generic map[uint8]string
}

// NewSpellCaster 用解出來的泛型清單建一個施法器。
func NewSpellCaster(handlers []GenericSpellHandler) *SpellCaster {
	generic := make(map[uint8]string, len(handlers))
	for _, handler := range handlers {
		if handler.SpellID > 0 && handler.SpellID <= SpellDispatchCount {
			generic[uint8(handler.SpellID)] = handler.Message
		}
	}
	return &SpellCaster{generic: generic}
}

// ReadDOSSpellCaster 直接從原版 ZIP 建一個。
func ReadDOSSpellCaster(zipPath string) (*SpellCaster, error) {
	handlers, err := ReadDOSGenericSpellHandlers(zipPath)
	if err != nil {
		return nil, err
	}
	return NewSpellCaster(handlers), nil
}

// Implemented 說這個編號施得出來了沒有：逐支讀過的，或版型認得出來的。
func (c *SpellCaster) Implemented(id uint8) bool {
	if SpellIsImplemented(id) {
		return true
	}
	if c == nil {
		return false
	}
	_, ok := c.generic[id]
	return ok
}

// Cast 算出一次施法的結果，泛型的那批走參數表。
func (c *SpellCaster) Cast(id uint8, parameters []SpellParameters, casterLevel int,
	roller Roller) (CastEffect, error) {
	if effect, err := CastSpell(id, parameters, casterLevel, roller); err == nil {
		return effect, nil
	}
	if c == nil {
		return CastEffect{}, fmt.Errorf("Pool spell %d has no handler and no caster table", id)
	}
	if _, ok := c.generic[id]; !ok {
		return CastEffect{}, fmt.Errorf(
			"Pool spell %d has no read handler; overlay-22 dispatch slot %d is still unread (spec 073)",
			id, id)
	}
	if id == 0 || int(id) >= len(parameters) {
		return CastEffect{}, fmt.Errorf("Pool spell %d is outside the parameter table", id)
	}
	// 純泛型：沒有傷害也沒有治療，只把參數表的效果碼掛上去。
	return CastEffect{EffectCode: parameters[id].EffectCode()}, nil
}

// GenericMessage 是那批泛型法術推給 `08BCh` 的字面訊息。
func (c *SpellCaster) GenericMessage(id uint8) (string, bool) {
	if c == nil {
		return "", false
	}
	message, ok := c.generic[id]
	return message, ok
}

// CastSpell 算出一次施法的結果。
//
// 逐條的出處：
//
//	01h Bless          0FF5h  推施法者的 `+10Eh`（哪一邊）給 0F35h，整邊掛效果
//	03h Cure Light     1051h  Roll(1, 8) 的治療，直接呼叫 0100h:0089h，不走 08BCh
//	09h Burning Hands  1178h  傷害＝施法者等級，沒有擲骰
//	0Fh Magic Missile  1429h  Roll(等級÷2, 4) ＋ 等級÷2
//	14h Shocking Grasp 14BFh  Roll(1, 8) ＋ 等級
//	15h Sleep          1513h  額度 Roll(4, 4) 生命骰，逐個目標依 HD 扣
//	2Fh Fireball       262Eh  Roll(等級, 6)
//	33h Lightning Bolt 2B75h  Roll(等級, 6)
//
// **Magic Missile 的發數與說明書不一致**：碼是 `等級 ÷ 2`，說明書寫
// 「每昇兩級多一發，第 3 或 4 級 2 發」＝ `(等級+1) ÷ 2`。這裡照碼接——
// 原始執行檔是一手資料，說明書是二手的。差異與待驗的實驗記在 spec 098。
func CastSpell(id uint8, parameters []SpellParameters, casterLevel int,
	roller Roller) (CastEffect, error) {
	if id == 0 || int(id) >= len(parameters) {
		return CastEffect{}, fmt.Errorf("Pool spell %d is outside the parameter table", id)
	}
	effect := CastEffect{EffectCode: parameters[id].EffectCode()}
	switch id {
	case SpellIDBless:
		effect.WholeSide = true
	case SpellIDCureLightWound:
		effect.Heal = roller.Roll(1, 8)
	case SpellIDBurningHands:
		effect.Damage = casterLevel
	case SpellIDMagicMissile:
		missiles := casterLevel / 2
		effect.Damage = roller.Roll(missiles, 4) + missiles
	case SpellIDShockingGrasp:
		effect.Damage = roller.Roll(1, 8) + casterLevel
	case SpellIDSleep:
		// 額度是 4d4 生命骰（`151Eh` 的 Roll(4, 4)），效果碼 35h。
		effect.Area, effect.SleepBudget = true, roller.Roll(4, 4)
		effect.EffectCode = SleepEffectCode
	case SpellIDFireball, SpellIDLightningBolt:
		effect.Damage, effect.Area = roller.Roll(casterLevel, 6), true
	default:
		return CastEffect{}, fmt.Errorf(
			"Pool spell %d has no read handler; overlay-22 dispatch slot %d is still unread (spec 073)",
			id, id)
	}
	return effect, nil
}

// SpellIsImplemented 說這個編號的處理常式讀出來了沒有。前端用它決定
// 一條法術能不能選，而不是讓玩家選了才失敗。
func SpellIsImplemented(id uint8) bool {
	switch id {
	case SpellIDBless, SpellIDCureLightWound, SpellIDBurningHands,
		SpellIDMagicMissile, SpellIDShockingGrasp, SpellIDSleep,
		SpellIDFireball, SpellIDLightningBolt:
		return true
	}
	return false
}
