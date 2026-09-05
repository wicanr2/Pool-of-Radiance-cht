package gamepack

import "fmt"

// 每個法術編號一筆 16-byte 參數。overlay-22 的共用施法常式 `08BCh` 全程用
// `di = 編號 × 16` 取這張表，一支常式服務 67 個編號、行為差異全部從這裡來。
//
// 表在 START.EXE 的資料段（`3194h + 30640 = 43476`，檔案只有 47936 bytes，
// 所以它是靜態初始化過的），編號 0 那筆整筆是零，是走不到的哨兵。
//
// 讀它的碼一律用「基底加欄位位移」的形式（`[di+3196h]` 就是 `+2`），
// 所以欄位編號要換算回 `3194h`。
const (
	// SpellParameterTableAddress 是編號 0 那筆的 DS 位址；那一筆整筆是零，
	// 是走不到的哨兵。
	SpellParameterTableAddress = 0x3194
	// SpellParameterRecordSize 是一筆的大小。
	SpellParameterRecordSize = 16
	// SpellParameterCount 是表的筆數，含編號 0 的哨兵，
	// 上限與派發表（spec 073）的 67 個編號相符。
	SpellParameterCount = SpellDispatchCount + 1

	// spellParameterClass 是 `+0`：0 牧師、1 法師、2 物品效果。
	// overlay-25 的 `26F8h` 用它決定施法者等級要讀角色記錄的哪一格。
	spellParameterClass = 0
	// spellParameterLevel 是 `+1`：法術等級。
	spellParameterLevel = 1
	// spellParameterAttackRoll 是 `+2`：值為 FFh 時 `08BCh` 走命中判定
	// （`0997h` 的 `cmp byte ptr [di+3196h], 0FFh`），先擲一次攻擊再談效果。
	// 同一個 byte 也是基礎射程，見 spellParameterBaseRange。
	spellParameterAttackRoll = 2
	// spellParameterBaseRange 是 `+2`：基礎射程（格）。它同時是命中旗標：
	// FFh 代表碰觸，射程算出來也是 1。
	spellParameterBaseRange = 2
	// spellParameterLevelRange 是 `+3`：每施法者等級再加的射程
	// （`0742h` 的 `mov al, [di+3197h]`）。
	spellParameterLevelRange = 3
	// spellParameterTargeting 是 `+6`。它有兩個用途：
	// `08BCh` 的 `07A3h` 只看它是不是零（射程算成 0 而它非零時墊成 1），
	// 而 **overlay-13 的挑目標常式（`20ADh`）看它的低四位**——那是
	// 目標模式（`20F0h` 的 `mov al,[di+319Ah]` 之後 `and $0Fh`）。
	spellParameterTargeting = 6
	// spellParameterRangeFloor 是同一格的別名，給射程那條路用。
	spellParameterRangeFloor = spellParameterTargeting
	// spellParameterArea 是 `+7`：範圍法術的形狀參數。overlay-13 依模式
	// 取它的低三位（`2241h` 的 `and $7`）或低兩位（`22CDh` 的 `and $3`）。
	spellParameterArea = 7
	// SpellTargetModeMask 取出目標模式。
	SpellTargetModeMask = 0x0f
	// spellParameterFixedDuration 是 `+4`：效果的固定回合數
	// （`08A2h` 的 `mov al, [di+3198h]`）。
	spellParameterFixedDuration = 4
	// spellParameterLevelDuration 是 `+5`：每施法者等級再加的回合數
	// （`088Dh` 的 `mov al, [di+3199h]`）。
	spellParameterLevelDuration = 5
	// spellParameterSaveRule 是 `+8`：0 表示不用擲豁免
	// （`095Eh` 的 `cmp byte ptr [di+319Ch], 0`）。非零時它還會一路傳給
	// `0100h:007Fh` 與 `0100h:0084h`，所以它不只是布林；1..3 的差別未讀。
	spellParameterSaveRule = 8
	// spellParameterSaveCategory 是 `+9`：豁免類別，索引角色記錄 `+6Dh` 起
	// 那五個目標值（spec 075）。`097Ch` 把它推給 `0100h:0043h`。
	spellParameterSaveCategory = 9
	// spellParameterEffectCode 是 `+0Ah`：掛到角色效果串列（spec 069）的
	// 效果碼。`0A13h` 先檢查它大於零才進掛效果那一段，所以值為零就是
	// 「不留狀態」。
	spellParameterEffectCode = 10
	// spellParameterAreaBudget 是 `+0Fh`：**範圍法術收人的預算**
	//（spec 074）。overlay-09 `0272h` 與 overlay-13 `1F56h` 都用
	// `[編號 × 16 + 31A3h]` 取它，推給 `0138h:003Eh`（鄰近查詢）當第三個
	// 引數——也就是 `TraceMovement` 的上限，所以「在範圍內」＝**那個預算
	// 內走得到**，不是半徑比大小。
	//
	// 六十七格裡只有七格非零，而且正好都是範圍法術：睡眠、沉默 15 呎、
	// 臭雲、解除魔法兩個編號各 1，火球與 64 號各 3。
	spellParameterAreaBudget = 0x0F

	// spellParameterAttackRollFlag 是 `+2` 代表「要擲命中」的值。
	spellParameterAttackRollFlag = 0xff
)

// SpellTargetMode 是 `+6` 低四位的目標模式（overlay-13 `20ADh`）。
type SpellTargetMode uint8

// 六十七格用到的模式。分組本身就是語意證據：模式 0Ah 正好是祝福、詛咒、
// 急速與緩速這四支整邊的法術，模式 0 全是自身增益，模式 0Bh 是火球。
const (
	// SpellTargetSelf 是模式 0：不挑目標，作用在施法者自己
	//（`20FDh` 直接把 `DS:5CF0h` 當成唯一的目標）。
	SpellTargetSelf SpellTargetMode = 0x00
	// SpellTargetSingle 是模式 4：挑一個目標。最大的一組（30 支），
	// 治療與傷害都在裡面——原版由玩家自己瞄。
	SpellTargetSingle SpellTargetMode = 0x04
	// SpellTargetHold 是模式 6 與 7：定身術那一族。
	SpellTargetHold SpellTargetMode = 0x06
	// SpellTargetHoldAlt 是模式 7。
	SpellTargetHoldAlt SpellTargetMode = 0x07
	// SpellTargetBolt 是模式 8：閃電束那種直線。
	SpellTargetBolt SpellTargetMode = 0x08
	// SpellTargetArea 是模式 9：以一點為中心的範圍（催眠、臭雲、解除魔法）。
	SpellTargetArea SpellTargetMode = 0x09
	// SpellTargetWholeSide 是模式 0Ah：整邊（祝福、詛咒、急速、緩速）。
	SpellTargetWholeSide SpellTargetMode = 0x0a
	// SpellTargetBurst 是模式 0Bh：火球那種大範圍。
	SpellTargetBurst SpellTargetMode = 0x0b
	// SpellTargetPick 是模式 0Fh：走 `1E09h` 的挑目標介面。
	SpellTargetPick SpellTargetMode = 0x0f
)

// TargetMode 是這條法術怎麼挑目標。
func (p SpellParameters) TargetMode() SpellTargetMode {
	return SpellTargetMode(p.Raw[spellParameterTargeting] & SpellTargetModeMask)
}

// AreaParameter 是 `+7`：範圍法術的形狀參數。
func (p SpellParameters) AreaParameter() uint8 { return p.Raw[spellParameterArea] }

// AffectsWholeSide 是模式 0Ah。
func (p SpellParameters) AffectsWholeSide() bool {
	return p.TargetMode() == SpellTargetWholeSide
}

// AffectsArea 是「不只打一個」的那幾種模式。
func (p SpellParameters) AffectsArea() bool {
	switch p.TargetMode() {
	case SpellTargetBolt, SpellTargetArea, SpellTargetBurst:
		return true
	}
	return false
}

// SpellSource 是參數表 `+0` 的三個值。
type SpellSource uint8

const (
	// SpellSourceCleric 是神術，施法者等級讀角色記錄的 `+96h`。
	SpellSourceCleric SpellSource = 0
	// SpellSourceMagicUser 是巫術，讀 `+9Bh`。
	SpellSourceMagicUser SpellSource = 1
	// SpellSourceItem 是物品或怪物的效果，施法者等級固定 12。
	SpellSourceItem SpellSource = 2
)

// SpellParameters 是一筆原始記錄。只有讀得出證據的欄位有具名取值；其餘留在
// Raw 裡，不替它們編語意。
type SpellParameters struct {
	SpellID int
	Raw     [SpellParameterRecordSize]byte
}

// Source 是這個編號屬於神術、巫術，還是物品效果。
func (p SpellParameters) Source() SpellSource { return SpellSource(p.Raw[spellParameterClass]) }

// Level 是法術等級。
func (p SpellParameters) Level() int { return int(p.Raw[spellParameterLevel]) }

// RequiresAttackRoll 說這個法術是不是要先擲中才生效。
func (p SpellParameters) RequiresAttackRoll() bool {
	return p.Raw[spellParameterAttackRoll] == spellParameterAttackRollFlag
}

// Range 是射程，單位是格。`0723h` 那一段算的是 `+2 + +3 × 施法者等級`，
// 算出來是 0 而 `+6` 非零就墊成 1，FFh 也是 1（碰觸）。
//
// 原版在戰術地圖外把施法者等級當成 6（`ds:6CB3h` 那個分支），所以戰鬥外
// 呼叫時要傳 6，不是角色的真實等級。
func (p SpellParameters) Range(casterLevel int) int {
	value := (int(p.Raw[spellParameterBaseRange]) + int(p.Raw[spellParameterLevelRange])*casterLevel) & 0xff
	if value == 0 && p.Raw[spellParameterRangeFloor] != 0 {
		return 1
	}
	if value == spellParameterAttackRollFlag {
		return 1
	}
	return value
}

// Duration 是效果持續幾回合：固定值加上每施法者等級的增量。
// `0875h` 那一段就是 `+4 + +5 × 等級`，兩個都是零表示不自己結束。
func (p SpellParameters) Duration(casterLevel int) int {
	return int(p.Raw[spellParameterFixedDuration]) + int(p.Raw[spellParameterLevelDuration])*casterLevel
}

// SaveRule 是 `+8`：豁免成功之後怎麼處置傷害。0 就不擲。
//
// 處置在 overlay-24 entry 19（code `133Ah`）：傷害先進 `DS:6776h`，
// 豁免成功（`1354h` 的 `[bp+6]` 非 0）時依規則值分三支——
// `135Dh` 值 1 把傷害清成 0、`1368h` 值 2 除以 2、其餘值不動傷害。
func (p SpellParameters) SaveRule() uint8 { return p.Raw[spellParameterSaveRule] }

// 豁免成功之後的三種處置（overlay-24 `133Ah`）。
const (
	// SaveRuleNone 是不用擲。
	SaveRuleNone uint8 = 0
	// SaveRuleNegates 是豁免成功就完全無效。
	SaveRuleNegates uint8 = 1
	// SaveRuleHalves 是豁免成功傷害減半（整數除法）。
	SaveRuleHalves uint8 = 2
)

// DamageAfterSave 依規則值算豁免成功之後剩下多少傷害。
func DamageAfterSave(rule uint8, damage int) int {
	switch rule {
	case SaveRuleNegates:
		return 0
	case SaveRuleHalves:
		return damage / 2
	default:
		return damage
	}
}

// AllowsSavingThrow 說目標有沒有豁免機會。
func (p SpellParameters) AllowsSavingThrow() bool { return p.SaveRule() != 0 }

// SaveCategory 是豁免類別，對到角色記錄 `+6Dh` 起那五個目標值的其中一格。
func (p SpellParameters) SaveCategory() SaveCategory {
	return SaveCategory(p.Raw[spellParameterSaveCategory])
}

// EffectCode 是掛上去的效果碼；0 表示這個法術不留狀態。
func (p SpellParameters) EffectCode() uint8 { return p.Raw[spellParameterEffectCode] }

// AreaBudget 是 `+0Fh`：範圍法術收人的預算（spec 074）。零代表這一支不靠
// 這條路收人。
func (p SpellParameters) AreaBudget() int { return int(p.Raw[spellParameterAreaBudget]) }

// ParseSpellParameterTable 從 START.EXE 的位元組解出整張表。
func ParseSpellParameterTable(executable []byte) ([]SpellParameters, error) {
	start := SpellParameterTableAddress + startDataSegmentFileDelta
	end := start + SpellParameterCount*SpellParameterRecordSize
	if len(executable) < end {
		return nil, fmt.Errorf("START.EXE is %d bytes, the spell parameter table needs %d", len(executable), end)
	}
	table := make([]SpellParameters, SpellParameterCount)
	for id := 0; id < SpellParameterCount; id++ {
		table[id].SpellID = id
		copy(table[id].Raw[:], executable[start+id*SpellParameterRecordSize:])
	}
	return table, nil
}

// ReadDOSSpellParameters 從原版 ZIP 解出參數表。
func ReadDOSSpellParameters(zipPath string) ([]SpellParameters, error) {
	raw, err := readStartExecutable(zipPath)
	if err != nil {
		return nil, err
	}
	return ParseSpellParameterTable(raw)
}
