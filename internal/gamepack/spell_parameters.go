package gamepack

import "fmt"

// 每個法術編號一筆 16-byte 參數。overlay-22 的共用施法常式 `08BCh` 全程用
// `di = 編號 × 16` 取這張表，一支常式服務 67 個編號、行為差異全部從這裡來。
//
// 表在 START.EXE 的資料段（`3196h + 30640 = 43478`，檔案只有 47936 bytes，
// 所以它是靜態初始化過的），編號 0 那筆整筆是零，是走不到的哨兵。
const (
	// SpellParameterTableAddress 是編號 0 那筆的 DS 位址。
	SpellParameterTableAddress = 0x3196
	// SpellParameterRecordSize 是一筆的大小。
	SpellParameterRecordSize = 16
	// SpellParameterCount 是表的筆數，含編號 0 的哨兵，
	// 上限與派發表（spec 073）的 67 個編號相符。
	SpellParameterCount = SpellDispatchCount + 1

	// spellParameterAttackRoll 是 `+0`：值為 FFh 時 `08BCh` 走命中判定
	// （`0997h` 的 `cmp byte ptr [di+3196h], 0FFh`），先擲一次攻擊再談效果。
	spellParameterAttackRoll = 0
	// spellParameterSaveCategory 是 `+6`：0 表示不用擲豁免
	// （`095Eh` 的 `cmp byte ptr [di+319Ch], 0`）。
	spellParameterSaveCategory = 6
	// spellParameterSaveModifier 是 `+7`：豁免判定的第二個引數
	// （`097Ch` 推給 `0100h:0043h`）。
	spellParameterSaveModifier = 7
	// spellParameterEffectCode 是 `+8`：掛到角色效果串列（spec 069）的效果碼。
	// `0A13h` 先檢查它大於零才進掛效果那一段，所以值為零就是「不留狀態」。
	spellParameterEffectCode = 8

	// spellParameterAttackRollFlag 是 `+0` 代表「要擲命中」的值。
	spellParameterAttackRollFlag = 0xff
)

// SpellParameters 是一筆原始記錄。只有讀得出證據的欄位有具名取值；其餘留在
// Raw 裡，不替它們編語意。
type SpellParameters struct {
	SpellID int
	Raw     [SpellParameterRecordSize]byte
}

// RequiresAttackRoll 說這個法術是不是要先擲中才生效。
func (p SpellParameters) RequiresAttackRoll() bool {
	return p.Raw[spellParameterAttackRoll] == spellParameterAttackRollFlag
}

// SaveCategory 是豁免的類別；0 表示不擲。類別 1..3 各自是什麼還沒讀。
func (p SpellParameters) SaveCategory() uint8 { return p.Raw[spellParameterSaveCategory] }

// SaveModifier 是豁免判定的第二個引數。
func (p SpellParameters) SaveModifier() uint8 { return p.Raw[spellParameterSaveModifier] }

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
