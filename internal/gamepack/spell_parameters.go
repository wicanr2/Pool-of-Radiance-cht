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
	// spellParameterRangeFloor 是 `+6`：射程算成 0 而它非零時，射程墊成 1
	// （`07A3h` 的 `cmp byte ptr [di+319Ah], 0`）。它本身不是布林，
	// 其餘語意要到 overlay-13 的四個呼叫端去讀。
	spellParameterRangeFloor = 6
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

	// spellParameterAttackRollFlag 是 `+2` 代表「要擲命中」的值。
	spellParameterAttackRollFlag = 0xff
)

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

// SaveRule 是 0 就不擲豁免。非零代表要擲，值本身還會傳給後面兩支常式；
// 1..3 分別是什麼處置（無效、減半、其他）還沒讀。
func (p SpellParameters) SaveRule() uint8 { return p.Raw[spellParameterSaveRule] }

// AllowsSavingThrow 說目標有沒有豁免機會。
func (p SpellParameters) AllowsSavingThrow() bool { return p.SaveRule() != 0 }

// SaveCategory 是豁免類別，對到角色記錄 `+6Dh` 起那五個目標值的其中一格。
func (p SpellParameters) SaveCategory() SaveCategory {
	return SaveCategory(p.Raw[spellParameterSaveCategory])
}

// EffectCode 是掛上去的效果碼；0 表示這個法術不留狀態。
func (p SpellParameters) EffectCode() uint8 { return p.Raw[spellParameterEffectCode] }

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
