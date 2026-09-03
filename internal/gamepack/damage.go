package gamepack

// `2Eh DAMAGE`（spec 084）：ECL 直接對隊伍造成傷害，全遊戲 47 個呼叫點。
//
// 傷害**只擲一次**（overlay-03 `2B47h` 在挑目標之前就算好），全隊模式下
// 每個人吃的是同一個數字。豁免成功就完全不受傷，沒有折半。
const (
	// DamageOpcode 是這條 opcode。
	DamageOpcode = 0x2E
	// DamageOperands 是它吃幾個運算元。
	DamageOperands = 5

	// DamageFlagApply（bit 7）沒設的話整條什麼都不做（`2B89h` 的分支）。
	DamageFlagApply = 0x80
	// DamageFlagWholeParty（bit 6）是全隊；沒設就隨機挑一個人。
	DamageFlagWholeParty = 0x40
	// DamageFlagNoSave（bit 5）是不給豁免。
	DamageFlagNoSave = 0x20
	// DamageFlagSaveModifierMask（bit 0..4）是**擲豁免時的修正**，不是類別。
	// 呼叫端把 `旗標 & 1Fh` 與 `運算元5 & 7` 依序推進堆疊
	//（overlay-03 `2BCEh..2BD6h`），而豁免常式（overlay-24 entry 7、
	// code `0D61h`）第一個參數 `cbtw` 之後加進 d20，第二個才拿去索引
	// `record[+6Dh + 類別]`（spec 075）。
	DamageFlagSaveModifierMask = 0x1f
	// DamageSaveCategoryMask 是運算元 5 真正用到的位元（`2B9Dh` 的 `and 7`），
	// 也就是豁免類別。
	DamageSaveCategoryMask = 0x07

	// UnconsciousState 是打到剛好 0 點的狀態（`+10Ch` = 4）。
	UnconsciousState = 4
	// AliveStateMax 是「還活著」的最大狀態碼。原版用一個集合
	//（overlay-25 `2246h`，第一個位元組是 03h）測狀態 0 與 1。
	AliveStateMax = 1
	// AnnihilationMargin 是「超過負幾點就直接死透」：赤字大於 9。
	AnnihilationMargin = 9

	// HitPointsOffset 是目前生命值在角色記錄裡的位置。
	HitPointsOffset = 0x11b
	// CharacterStateOffset 是狀態。
	CharacterStateOffset = 0x10c
)

// DamageOutcome 是一個人吃完傷害之後的樣子。
type DamageOutcome struct {
	// HitPoints 是新的 `+11Bh`。倒下的人一律歸零。
	HitPoints int
	// State 是新的 `+10Ch`。
	State uint8
	// Downed 為真代表這一下把人打倒了（狀態離開 0..1 那一組）。
	Downed bool
}

// ApplyDamage 重現 overlay-25 entry 28（`2266h`）：算出剩餘與赤字，依兩者
// 決定新狀態，再決定要寫回剩餘生命值還是歸零。
//
// 三條界線都照原版：赤字大於 9 直接死透；剛好歸零而原本狀態是 1 也死透；
// 其餘赤字為正是瀕死，剛好歸零是不省人事。
func ApplyDamage(hitPoints int, state uint8, damage int) DamageOutcome {
	remaining, deficit := 0, 0
	if hitPoints >= damage {
		remaining = hitPoints - damage
	} else {
		deficit = damage - hitPoints
	}
	next := state
	switch {
	case deficit > AnnihilationMargin:
		next = DeadState
	case remaining == 0 && state == 1:
		next = DeadState
	case deficit > 0:
		next = DyingState
	case remaining == 0:
		next = UnconsciousState
	}
	if next <= AliveStateMax {
		return DamageOutcome{HitPoints: remaining, State: next}
	}
	return DamageOutcome{HitPoints: 0, State: next, Downed: true}
}

// DyingState 與 DeadState 與 internal/combat 的同名常數一致；那一份是戰術
// 戰鬥用的，這一份是 ECL 用的，兩邊指的是同一個 `+10Ch`。
const (
	DyingState uint8 = 5
	DeadState  uint8 = 6
)

// DamageRequest 是把一條 `2Eh` 的五個運算元解讀完的結果。
type DamageRequest struct {
	// Flags 是運算元 1。
	Flags uint8
	// DiceCount／DiceSides／Bonus 是運算元 2、3、4。
	DiceCount int
	DiceSides int
	Bonus     int
	// SaveCategory 是運算元 5 的低三位——`record[+6Dh + 類別]` 的索引
	// （spec 075 的五格）。
	SaveCategory int
}

// Applies 回報這一條要不要做事。
func (r DamageRequest) Applies() bool { return r.Flags&DamageFlagApply != 0 }

// WholeParty 回報是全隊還是隨機一個人。
func (r DamageRequest) WholeParty() bool { return r.Flags&DamageFlagWholeParty != 0 }

// AllowsSave 回報要不要擲豁免。
func (r DamageRequest) AllowsSave() bool { return r.Flags&DamageFlagNoSave == 0 }

// SaveModifier 是擲豁免時加進 d20 的修正（旗標的低五位）。
func (r DamageRequest) SaveModifier() int { return int(r.Flags & DamageFlagSaveModifierMask) }

// NewDamageRequest 把五個運算元的值組成一次請求。
func NewDamageRequest(operands [DamageOperands]uint16) DamageRequest {
	return DamageRequest{
		Flags:        uint8(operands[0]),
		DiceCount:    int(uint8(operands[1])),
		DiceSides:    int(uint8(operands[2])),
		Bonus:        int(uint8(operands[3])),
		SaveCategory: int(uint8(operands[4]) & DamageSaveCategoryMask),
	}
}
