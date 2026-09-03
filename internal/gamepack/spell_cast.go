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
	// AreaBudget 是範圍法術收人的預算，直接傳給 `0419h`（TraceMovement）
	// 當上限：走得到的才在範圍內。火球術的那一支寫死 2
	// （overlay-22 `2675h` 推給 `0138h:003Eh` 的第三個引數）。
	// 為零代表**還沒讀出那一支的預算**，呼叫端只好收下整邊。
	AreaBudget int
	// SleepBudget 是催眠術能放倒的生命骰總量（`DS:47A6h`）。大於零時
	// 呼叫端要依 SleepHitDiceCost 逐個目標扣，扣得動的就睡著。
	SleepBudget int
	// CasterLevelOverride 是 `08BCh` 的第一個覆寫參數（`[bp+10h]`）：
	// 非零就取代真正的施法者等級（`08F2h` 的 `cmpb $0` 之後分岔）。
	// 鏡影術借這一格傳「幾個影像」。
	CasterLevelOverride int
	// EffectParameter 是第二個覆寫參數（`[bp+0Eh]`），掛效果時一起傳給
	// 效果常式（`0A3Dh`）。致病術傳 1，其餘多半是 0；完整語意未閉合。
	EffectParameter int
	// RemoveEffects 是要從目標身上拿掉的效果碼。解病術走的是這條路，
	// 不掛新效果（overlay-22 `225Bh`）。
	RemoveEffects []uint8
	// BlockedByEffect 非零時代表：目標身上已經有這個效果就什麼都不做
	// （處理常式先問 `0100h:006Bh`，回非零就直接返回）。
	BlockedByEffect uint8
	// AbilityBonus 是直接加在能力值上的法術（例如友誼術加魅力）。
	AbilityBonus AbilityBonus
	// MinimumHitPoints 大於零時：目標的目前生命值低於它就墊上去。
	// 緩毒術把 0 墊成 1（`188Dh` 的 `cmpb $0, es:[di+11Bh]`）。
	MinimumHitPoints int
	// PersonOnly 為真代表這一支只對「人」有效：目標記錄的 `+9Fh` 大於 1
	// 或 `+6Ch` 大於 1 就完全不受影響（`11DAh`／`174Bh` 那兩道比較）。
	PersonOnly bool
	// CreatureTypeFiltered 為真時，只有 `+9Fh` 等於 CreatureType 的目標
	// 會被收進來。迷蛇術是 `0Eh`。用旗標而不是「非零」是因為 `0`（人類）
	// 本身也是合法的種類。
	CreatureTypeFiltered bool
	CreatureType         uint8
	// HitPointBudgetFromCaster 為真代表額度是施法者的目前生命值
	// （迷蛇術的 `DS:47A6h` 由 `+11Bh` 填），逐個目標扣掉它的目前生命值，
	// 扣得動的才被迷住。
	HitPointBudgetFromCaster bool
	// SaveModifierByTargetCount 為真代表豁免修正看這一次選了幾個目標，
	// 算法見 HoldPersonSaveModifier。
	SaveModifierByTargetCount bool
	// StrengthValue 非零時：把目標的力量設成它，百分位設成
	// StrengthPercentile。overlay-24 entry 18 只往上調，本來就更高就不動。
	StrengthValue      uint8
	StrengthPercentile uint8
	// RequiresEffect 非零時：目標身上**沒有**這個效果就什麼都不做。
	// 縮小術要求 `0Ch`（被變大過）。與 BlockedByEffect 方向相反。
	RequiresEffect uint8
	// MessageOnly 為真代表這一支過了前面的關卡之後**只印一句話**，
	// 不掛效果、不解效果、不算傷害。縮小術就是這樣。
	MessageOnly bool
}

// AbilityBonus 是「把某個能力值加上去，加到上限為止」。
type AbilityBonus struct {
	// Ability 是能力值的索引（AbilityStrength 那一組）。Amount 為 0 時無效。
	Ability int
	// Amount 是加多少。
	Amount int
	// Cap 是上限；原版是先加再夾。
	Cap int
}

// FireballAreaBudget 是火球術收人的預算（overlay-22 `2675h`）。
// 它會傳給 `0419h` 當上限，所以「在範圍內」＝ 那個預算內走得到。
const FireballAreaBudget = 2

// HasteEffectCode 是急速術掛上去的效果碼（`2858h` 推的 2Ah）。
//
// 它同時解釋了編號 57 那一支（`2DB7h`）在防什麼：那支的前提正是
// 「目標身上有沒有 2Ah」——已經加速過就不再加。兩邊各自讀出來卻對上同一個碼。
const HasteEffectCode = 0x2a

// SlowEffectCode 是緩速術掛上去的效果碼（`2BCDh` 推的 27h）。
// overlay-15 的名稱鏈沒有它，所以它沒有顯示名稱。
const SlowEffectCode = 0x27

// CureDiseaseEffectCodes 是解病術會拿掉的效果碼（overlay-22 `225Bh`）。
// 2Ch 是致病、32h 是木乃伊惡疾、1Fh 是無助，與 overlay-15 的名稱鏈相符。
var CureDiseaseEffectCodes = [6]uint8{0x22, 0x2b, 0x2c, 0x1f, 0x32, 0x39}

// HoldPersonEffectCode 是定身術掛上去的效果碼（參數表 `+0Ah`，spec 074）。
// 兩個編號（23 與 49）用同一個碼。
const HoldPersonEffectCode = 0x34

// SpellIDHoldPerson 與 SpellIDHoldPersonAlt 是定身術的兩個編號。
const (
	SpellIDHoldPerson    = 23
	SpellIDHoldPersonAlt = 49
)

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
	SpellIDCurse          = 2  // 1026h
	SpellIDCureLightWound = 3  // 1051h
	SpellIDCauseLightWound = 4  // 108Fh
	SpellIDBurningHands   = 9  // 1178h
	SpellIDMagicMissile   = 15 // 1429h
	SpellIDShockingGrasp  = 20 // 14BFh
	SpellIDSleep          = 21 // 1513h
	SpellIDMirrorImage    = 32 // 1A6Fh
	SpellIDCauseDisease   = 40 // 231Dh
	SpellIDCureDisease    = 39 // 2300h → 225Bh
	SpellIDPrayer         = 42 // 249Dh
	SpellIDSpiritHammer   = 28 // 19A8h
	SpellIDSlow           = 55 // 2BC7h → 2724h
	SpellIDFriends        = 14 // 13C8h
	SpellIDCureBlindness  = 37 // 21E8h
	SpellIDRemoveCurse    = 43 // 2508h
	SpellIDFireballAlt    = 64 // 262Eh，與火球術同一支
	SpellIDMagicMissileAlt = 65 // 300Eh
	SpellIDNoOperation    = 66 // 3049h，整支是空的
	SpellIDGuardedGeneric = 57 // 2DB7h
	SpellIDGreaterHeal    = 58 // 2E02h
	SpellIDLesserHeal     = 62 // 2F85h
	SpellIDHaste          = 48 // 2852h
	SpellIDSlowPoison     = 26 // 1846h
	SpellIDEnlarge        = 12 // 128Dh
	SpellIDReadMagic      = 67 // 305Bh
	SpellIDFireball       = 47 // 262Eh
	SpellIDLightningBolt  = 51 // 2B75h
	SpellIDCharmPerson    = 10 // 11C5h
	SpellIDSnakeCharm     = 27 // 18F9h
	SpellIDReduce         = 13 // 135Eh
	SpellIDGiantStrength  = 59 // 2E9Ah
)


// CreatureTypeSnake 是迷蛇術收的那一族（`1927h` 的 `cmpb $0Eh`）。
// 原版資料裡是巨蛇與兩種蠍子。
const CreatureTypeSnake = 0x0E

// SpellAffectsPerson 是「魅惑人類／定身術這一目標算不算人」。
//
// 原版兩支的判斷逐位元組相同（`11DAh` 與 `174Bh`）：
//
//	cmp es:[di+9Fh], 1 ; ja  不受影響
//	cmp es:[di+6Ch], 1 ; jbe 受影響
//
// 所以種類要小於等於 1（人類或類人），而且 `+6Ch` 整個 byte 要小於等於 1。
// 熊地精的 `+6Ch` 是 `81h`，所以雖然種類是 1 仍然免疫。
func SpellAffectsPerson(creatureType, bodySize uint8) bool {
	return creatureType <= 1 && bodySize <= 1
}

// HoldPersonSaveModifier 是定身術依「這一次選了幾個目標」給的豁免修正
//（`1656h..168Ch`）。目標愈少愈難擋。
//
//	1 個：定身術（編號 17h）−2，定身怪物（31h）−3
//	2 個：−1
//	3 或 4 個：0
//
// 原版對 5 個以上沒有賦值，那一格是未初始化的區域變數；選單挑不到那麼多，
// 這裡回 0 而不是模擬垃圾值。
func HoldPersonSaveModifier(id uint8, targets int) int {
	switch targets {
	case 1:
		if id == SpellIDHoldPerson {
			return -2
		}
		return -3
	case 2:
		return -1
	default:
		return 0
	}
}

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
//	02h Curse          1026h  同上，訊息是 "is Cursed"，作用在另一邊
//	03h Cure Light     1051h  Roll(1, 8) 的治療，直接呼叫 0100h:0089h，不走 08BCh
//	09h Burning Hands  1178h  傷害＝施法者等級，沒有擲骰
//	0Fh Magic Missile  1429h  Roll(等級÷2, 4) ＋ 等級÷2
//	04h Cause Light W. 108Fh  傷害 Roll(1, 8)
//	14h Shocking Grasp 14BFh  Roll(1, 8) ＋ 等級
//	20h Mirror Image   1A6Fh  Roll(1, 4) 推在施法者等級那一格
//	28h Cause Disease  231Dh  四個覆寫參數 0／1／0／0，只掛效果
//	0Eh Friends        13C8h  Roll(2, 4) 加在魅力上，上限 25
//	25h Cure Blindness 21E8h  解掉效果碼 21h
//	2Bh Remove Curse   2508h  解掉效果碼 24h，並清掉物品的 +36h
//	27h Cure Disease   2300h  轉呼叫 225Bh：拿掉六個病痛類的效果碼
//	2Ah Prayer         249Dh  `(哪一邊 << 4) + 等級` 推在等級覆寫那一格
//	1Ch Spiritual H.   19A8h  四個覆寫參數 0／1／0／0（生出鎚子那段未讀）
//	0Ch Enlarge        128Dh  效果碼 12h，強度依施法者等級查表
//	1Ah Slow Poison    1846h  目前生命值是 0 就墊成 1，再走 08BCh
//	30h Haste          2852h  推效果碼 2Ah 走 2724h，整邊
//	37h Slow           2BC7h  推效果碼 27h 走 2724h，範圍法術
//	15h Sleep          1513h  額度 Roll(4, 4) 生命骰，逐個目標依 HD 扣
//	2Fh Fireball       262Eh  Roll(等級, 6)
//	33h Lightning Bolt 2B75h  Roll(等級, 6)
//	40h （無名）       262Eh  與火球術同一支
//	41h （無名）       300Eh  Roll(2, 4) ＋ 2
//	39h （無名）       2DB7h  中了效果 2Ah 就不做，沒中才走泛型
//	3Ah （無名）       2E02h  治療 Roll(1, 4) ＋ 8，並解掉 16h 與病痛那組
//	3Eh （無名）       2F85h  治療 Roll(2, 4) ＋ 2
//	42h （無名）       3049h  整支是空的：原版什麼都不做
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
	case SpellIDBless, SpellIDCurse:
		// 兩支都走 0F35h 那條整邊的路，差別只在訊息（"is Blessed" 與
		// "is Cursed"）與作用在哪一邊。
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
	case SpellIDCauseLightWound:
		// `10A5h` 的 Roll(1, 8)，第五個參數 8。與治療輕傷同一個骰子。
		effect.Damage = roller.Roll(1, 8)
	case SpellIDCauseDisease:
		// `2323h` 推的四個覆寫參數是 0／1／0／0：沒有傷害，
		// 只把參數表的效果碼掛上去，第二個覆寫參數是 1。
		effect.EffectParameter = 1
	case SpellIDPrayer:
		// `24A7h` 把 `(施法者的 +10Eh << 4) + 施法者等級` 推在等級覆寫那一格。
		// 隊伍這一邊的 `+10Eh` 是 0，所以對玩家而言就等於施法者等級本身；
		// 高四位只有怪物施展時才不是零。
		effect.CasterLevelOverride = casterLevel
	case SpellIDSpiritHammer:
		// `19AEh` 的四個覆寫參數是 0／1／0／0。08BCh 之後還有一段
		// （`19D1h` 起，推效果碼 17h）還沒讀，那是把鎚子生出來的部分。
		effect.EffectParameter = 1
	case SpellIDReadMagic:
		// `305Bh` 其實就是泛型版型，只是三個參數不是字面值：法術編號推的是
		// `DS:6779h`（目前這一支），等級覆寫推 FFh，效果參數推 1。
		// 版型比對是逐位元組的，所以它落在泛型之外——語意上沒有差別。
		effect.CasterLevelOverride = 0xff
		effect.EffectParameter = 1
	case SpellIDEnlarge:
		// `128Dh`：力量設成 18、百分位依施法者等級查 EnlargeMagnitudeByLevel，
		// 交給 overlay-24 entry 18；成功才印 `has been enlarged`，
		// 最後一律用 `0100h:0052h` 掛效果碼 `0Ch`。
		effect.EffectCode = EnlargeEffectCode
		effect.StrengthValue = EnlargeStrengthValue
		effect.StrengthPercentile = enlargeMagnitude(casterLevel)
		effect.EffectParameter = int(effect.StrengthPercentile)
	case SpellIDGiantStrength:
		// `2E9Ah`：力量設成 21、沒有百分位，訊息是 `is stronger`，
		// 效果碼 `26h`。與變大術同一支 overlay-24 entry 18。
		effect.EffectCode = GiantStrengthEffectCode
		effect.StrengthValue = GiantStrengthValue
	case SpellIDReduce:
		// `135Eh`：三道關卡都過才印 `has been reduced`——沒有目標就返回、
		// 豁免成功（`0100h:0043h(目標, 4, 0)` 回非零）就返回、目標身上
		// 沒有效果 `0Ch`（沒被變大過）也返回。**過了之後整支只印一句話**：
		// 不掛效果、不解掉 `0Ch`、不算傷害。照碼接，不補原版沒有的行為。
		effect.RequiresEffect = EnlargeEffectCode
		effect.MessageOnly = true
		effect.EffectCode = 0
	case SpellIDSlowPoison:
		// `1873h` 先問 `010Ah:00A7h(目標, 37h)`（中毒），接著若目前生命值
		// 是 0 就墊成 1（`1892h`），最後走 `08BCh`，等級覆寫推的是 FFh。
		effect.MinimumHitPoints = 1
		effect.CasterLevelOverride = 0xff
	case SpellIDHaste:
		// `2858h` 推效果碼 2Ah 與施法者的 `+10Eh`（哪一邊）給 `2724h`，
		// 與緩速術同一支。訊息是 "is Speedy"。
		effect.WholeSide, effect.EffectCode = true, HasteEffectCode
	case SpellIDSlow:
		// `2BCDh` 先推效果碼 27h 再走 `2724h`——那一支會設 `DS:677Eh = 1`，
		// 是範圍法術。
		effect.Area, effect.EffectCode = true, SlowEffectCode
	case SpellIDFriends:
		// `13D9h` 的 Roll(2, 4) 加在記錄 `+15h`（魅力）上，上限 19h ＝ 25
		// （`13F0h` 的 `cmpb $19h` 之後 `13FBh` 夾住）。
		effect.AbilityBonus = AbilityBonus{
			Ability: AbilityCharisma, Amount: roller.Roll(2, 4), Cap: 25,
		}
	case SpellIDCureBlindness:
		// `21F6h` 問 `0100h:006Bh(目標, 21h)`，中了就解掉。
		effect.RemoveEffects = []uint8{0x21}
	case SpellIDRemoveCurse:
		// `2516h` 問效果碼 24h；另外還會把物品的 `+36h`（詛咒旗標）清成 0，
		// 那一段 remake 還沒有對應的欄位。
		effect.RemoveEffects = []uint8{0x24}
	case SpellIDCureDisease:
		// `225Bh` 逐個問 `0100h:006Bh(目標, 碼)`，中了就用 `0100h:002Ah`
		// 拿掉。碼與 overlay-15 的名稱鏈對得上：2Ch 是致病、32h 是
		// 木乃伊惡疾、1Fh 是無助。
		effect.RemoveEffects = append([]uint8(nil), CureDiseaseEffectCodes[:]...)
	case SpellIDMirrorImage:
		// `1A79h` 把 Roll(1, 4) 推在**第一個**覆寫參數的位置——那一格是
		// 施法者等級的覆寫（`08BCh` 的 `08F2h`），所以鏡影的數量是借
		// 等級那一格傳的。
		effect.CasterLevelOverride = roller.Roll(1, 4)
	case SpellIDSleep:
		// 額度是 4d4 生命骰（`151Eh` 的 Roll(4, 4)），效果碼 35h。
		effect.Area, effect.SleepBudget = true, roller.Roll(4, 4)
		effect.EffectCode = SleepEffectCode
	case SpellIDMagicMissileAlt:
		// `3024h` 的 Roll(2, 4) 之後 `add $2`，第五個覆寫參數 8。
		effect.Damage = roller.Roll(2, 4) + 2
	case SpellIDGuardedGeneric:
		// `2DC5h` 先問 `0100h:006Bh(目標, 2Ah)`；已經中了就整支返回，
		// 沒中才走泛型那條（四個覆寫參數全是 0）。
		effect.BlockedByEffect = 0x2a
	case SpellIDGreaterHeal:
		// `2E4Eh` 的 Roll(1, 4) ＋ 8 走治療常式 `0100h:0089h`。
		// 前面還會解掉 16h，並走一次解病術那條鏈（`225Bh`）。
		effect.Heal = roller.Roll(1, 4) + 8
		effect.RemoveEffects = append([]uint8{0x16}, CureDiseaseEffectCodes[:]...)
	case SpellIDLesserHeal:
		// `2F93h` 的 Roll(2, 4) ＋ 2，同一支治療常式。
		effect.Heal = roller.Roll(2, 4) + 2
	case SpellIDNoOperation:
		// `3049h` 整支只有 push bp / mov bp,sp / mov sp,bp / pop bp / retf——
		// **原版就是什麼都不做**。接成 no-op 是照實接，不是還沒做。
	case SpellIDFireball, SpellIDFireballAlt:
		// `2675h` 推給 `0138h:003Eh` 的預算是 2，方向 FFh（不限方向）。
		effect.Damage, effect.Area = roller.Roll(casterLevel, 6), true
		effect.AreaBudget = FireballAreaBudget
	case SpellIDCharmPerson:
		// `11C5h` 先擋不是人的目標，過得了才走泛型那條——四個覆寫參數是
		// `(施法者 +10Eh << 7) + 施法者等級`、1、0、0。高位那一段是
		// 「誰迷的」，隊伍這一邊的 `+10Eh` 是 0。
		effect.PersonOnly = true
		effect.CasterLevelOverride = casterLevel
		effect.EffectParameter = 1
	case SpellIDHoldPerson, SpellIDHoldPersonAlt:
		// `1650h` 兩支共用：先依目標數算豁免修正，再逐個目標擲豁免；
		// 不是人的目標一律當作豁免成功（`175Dh` 直接把結果設成 1）。
		effect.PersonOnly = true
		effect.SaveModifierByTargetCount = true
	case SpellIDSnakeCharm:
		// `18F9h`：額度是施法者的目前生命值（`+11Bh` → `DS:47A6h`），
		// 沿著目標串列走，`+9Fh` 是 `0Eh` 而且目前生命值扣得動的就收進來。
		// 與 AD&D 一版的「總生命值不超過牧師目前生命值」逐字相同。
		effect.CreatureTypeFiltered, effect.CreatureType = true, CreatureTypeSnake
		effect.HitPointBudgetFromCaster = true
	case SpellIDLightningBolt:
		// 閃電束走的是 `287Ch` 那條（目標模式 8＝直線），預算還沒讀。
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
	case SpellIDBless, SpellIDCurse, SpellIDCureLightWound, SpellIDCauseLightWound,
		SpellIDBurningHands, SpellIDMagicMissile, SpellIDShockingGrasp,
		SpellIDSleep, SpellIDMirrorImage, SpellIDCauseDisease, SpellIDCureDisease,
		SpellIDPrayer, SpellIDSpiritHammer, SpellIDSlow, SpellIDFriends,
		SpellIDCureBlindness, SpellIDRemoveCurse, SpellIDFireballAlt,
		SpellIDMagicMissileAlt, SpellIDNoOperation, SpellIDGuardedGeneric,
		SpellIDGreaterHeal, SpellIDLesserHeal, SpellIDHaste, SpellIDSlowPoison,
		SpellIDEnlarge, SpellIDReadMagic,
		SpellIDFireball, SpellIDLightningBolt,
		SpellIDCharmPerson, SpellIDHoldPerson, SpellIDHoldPersonAlt, SpellIDSnakeCharm,
		SpellIDReduce, SpellIDGiantStrength:
		return true
	}
	return false
}

// 力量術（overlay-22 `1F16h`）加多少，看**目標**的職業：
//
//	1F2Fh  +9Bh（法師）> 0        → Roll(1, 4)
//	1F48h  +96h（牧師）> 0 或
//	1F53h  +9Ch（賊）  > 0        → Roll(1, 6)
//	1F6Ch  +98h（戰士）> 0        → Roll(1, 8)
//
// 與 AD&D 逐項相同。後面的判斷會蓋掉前面的，所以多職業取最後一個成立的。
//
// **還沒接進施法**：`CastSpell` 目前只拿得到施法者的等級，而這一條看的是
// 目標的職業。等施法的介面把目標傳進來再接。
func StrengthSpellDie(levels [ClassThac0ClassCount]uint8) (count, sides int) {
	count, sides = 0, 0
	if levels[ClassSlotMagicUser] > 0 {
		count, sides = 1, 4
	}
	if levels[ClassSlotCleric] > 0 || levels[ClassSlotThief] > 0 {
		count, sides = 1, 6
	}
	if levels[ClassSlotFighter] > 0 {
		count, sides = 1, 8
	}
	return count, sides
}

// EnlargeMagnitudeByLevel 是變大術（overlay-22 `128Dh`）依施法者等級寫進
// `DS:47A7h` 的值，也就是要設成的**特殊力量百分位**：0、1、51、76、91、100
// 正是 AD&D 的 18/00、18/01、18/51、18/76、18/91 五段。索引就是等級，
// 0 那格走不到。
var EnlargeMagnitudeByLevel = [7]uint8{0, 0, 1, 0x33, 0x4c, 0x5b, 0x64}

// StrengthSpellValue 是「把力量設成這個值」的法術要設成多少。
//
// `1293h` 寫進 `DS:47A6h` 的 `12h` **不是效果碼，是十進位的 18**：
// 它與 `DS:47A7h` 一起交給 overlay-24 entry 18（`1158h`），那一支拿
// `[bp+0Ch]` 跟記錄 `+10h`（力量）比、`[bp+0Ah]` 跟 `+16h`（特殊力量
// 百分位）比，**只往上調不往下調**，而且只有 `[bp+0Ch] == 12h` 時才比
// 百分位——18 是唯一有百分位的力量值。編號 3Bh 那一支推的是 `15h` ＝ 21，
// 訊息是 `is stronger`。
const (
	EnlargeStrengthValue    = 18
	GiantStrengthValue      = 21
	StrengthRecordOffset    = 0x10
	StrengthPercentileOffset = 0x16
)

// EnlargeEffectCode 是變大術掛的效果碼：參數表 `+0Ah` 與 `1331h` 的
// `mov al, 0Ch`（推給掛效果的 `0100h:0052h`）兩條路都指到 `0Ch`。
// 縮小術要求目標身上有它，overlay-24 entry 18 也拿 `0Ch` 與 `26h`
// 去找既有的力量效果節點。
const EnlargeEffectCode = 0x0c

// GiantStrengthEffectCode 是編號 3Bh（`2E9Ah`）掛的效果碼，
// 同樣兩條路對得上：參數表 `+0Ah` 是 `26h`，`1158h` 也認這個碼。
const GiantStrengthEffectCode = 0x26

// enlargeMagnitude 取變大術那張表的一格。等級超出範圍就取最後一格——
// 原版的比較鏈只寫到 6，再上去不會改 `DS:47A7h`，而它上一輪留下的值
// 就是第 6 級那個。
func enlargeMagnitude(casterLevel int) uint8 {
	if casterLevel < 0 {
		return 0
	}
	if casterLevel >= len(EnlargeMagnitudeByLevel) {
		return EnlargeMagnitudeByLevel[len(EnlargeMagnitudeByLevel)-1]
	}
	return EnlargeMagnitudeByLevel[casterLevel]
}

// RestorationOutcome 是「恢復術」還給角色的東西（overlay-22 `2C01h`，spec 097）。
type RestorationOutcome struct {
	// Restored 為真代表真的還了一級。身上沒有被吸取的等級就什麼都不做
	//（`2C16h` 的 `cmpb $0, es:[di+74h]`）。
	Restored bool
	// HitPoints 是還回來的生命值：欠的 HP 除以欠的等級數。
	HitPoints int
	// DrainedLevels／DrainedHitPoints 是還完之後剩下的欠帳。
	DrainedLevels    int
	DrainedHitPoints int
}

// Restore 重現 overlay-22 `2C01h` 的還帳那一段：
//
//	2c35  gain = +75h ÷ +74h
//	2c40  +32h  += gain      （最大生命值）
//	2c4a  +11Bh += gain      （目前生命值）
//	2c55  +0B1h += gain
//	2c60  +75h  -= gain
//	2c67  +74h  -= 1
//
// **還沒接進遊戲**：remake 還沒有能量吸取（overlay-12 `21C4h`），
// 沒有欠帳就沒有東西可還。規則先寫下來並釘住，接上吸取時就能直接用。
func Restore(drainedLevels, drainedHitPoints int) RestorationOutcome {
	if drainedLevels <= 0 {
		return RestorationOutcome{
			DrainedLevels: drainedLevels, DrainedHitPoints: drainedHitPoints,
		}
	}
	gain := drainedHitPoints / drainedLevels
	return RestorationOutcome{
		Restored:         true,
		HitPoints:        gain,
		DrainedLevels:    drainedLevels - 1,
		DrainedHitPoints: drainedHitPoints - gain,
	}
}
