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
	SpellIDFireball       = 47 // 262Eh
	SpellIDLightningBolt  = 51 // 2B75h
)

// CastSpell 算出一次施法的結果。
//
// 逐條的出處：
//
//	01h Bless          0FF5h  推施法者的 `+10Eh`（哪一邊）給 0F35h，整邊掛效果
//	03h Cure Light     1051h  Roll(1, 8) 的治療，直接呼叫 0100h:0089h，不走 08BCh
//	09h Burning Hands  1178h  傷害＝施法者等級，沒有擲骰
//	0Fh Magic Missile  1429h  Roll(等級÷2, 4) ＋ 等級÷2
//	14h Shocking Grasp 14BFh  Roll(1, 8) ＋ 等級
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
		SpellIDMagicMissile, SpellIDShockingGrasp, SpellIDFireball, SpellIDLightningBolt:
		return true
	}
	return false
}
