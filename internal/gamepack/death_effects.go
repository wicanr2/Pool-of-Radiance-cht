package gamepack

// 倒下時的群組 13、再生與怪物特殊攻擊的麻痺（spec 155，issue #113）。輸入：overlay-12（SHA-256
// `d1b05743…`）、overlay-13（`4d53df20…`）、overlay-24（`e878166e…`）、overlay-08、START.EXE 的 DS 常數；
// `objdump -D -b binary -m i8086 -M intel`（`coab-go-test:20260729`）。除註明者外皆 exact（位元組逐條讀）。
//
// 倒下的三個入口都是同一個形狀——先摘 `DS:0C28h` 那十六個碼（overlay-24 entry 13 `1004h`），
// 再派發群組 13（`63h 64h 67h 4Bh 4Ah`），`+10Dh` 仍是 0 就離場：
//
//	overlay-13 entry 4  `05F5h` 1004h、`0603h` 群組 13、`0608h` 還躺著 → ov32 entry 20   ; 近戰
//	overlay-24 entry 19 `1610h` 1004h、`161Dh` 群組 13、`1620h` 還躺著 → ov32 entry 20   ; 法術傷害
//	overlay-12 `005Ah`  `00C2h` 1004h、`00D0h` 群組 13、`00E0h` 還躺著 → ov32 entry 20   ; 中毒等當場死亡

const (
	// BoarRallyEffectCode 是 `63h`（WILD BOAR）：entry 92 `2727h`，倒下時再站起來拚。
	BoarRallyEffectCode uint8 = 0x63
	// TrollRevivalEffectCode 是 `64h`（TROLL）：entry 93 `27D0h`，不是火或 10h 殺的就掛 `66h`。
	TrollRevivalEffectCode uint8 = 0x64
	// TrollWoundEffectCode 是 `65h`（TROLL）：entry 94 `280Dh`，群組 5 與 6——受傷就掛 `3Bh`。
	TrollWoundEffectCode uint8 = 0x65
	// TrollRisesEffectCode 是 `66h`：entry 95 `285Fh`，到期時以滿血站起來。
	TrollRisesEffectCode uint8 = 0x66
	// VampireEscapeEffectCode 是 `67h`（VAMPIRE）：entry 96 `28B8h`。
	VampireEscapeEffectCode uint8 = 0x67
	// BoarFallsEffectCode 是 `5Fh`：entry 88 `25F9h`，`63h` 站起來之後掛的計時，到期 "Falls dead"。
	BoarFallsEffectCode uint8 = 0x5f
	// RegenerationPendingEffectCode 是 `3Bh`：entry 54 `147Ch`，到期時掛上 `62h`。
	RegenerationPendingEffectCode uint8 = 0x3b
	// RegenerationEffectCode 是 `62h`（TROLL 受傷三回合後、VAMPIRE 生來就有）：entry 91 `26F5h`，
	// 群組 19——每回合 +3，不超過 `+32h`。
	RegenerationEffectCode uint8 = 0x62

	// boarRallyCeiling 是 `275Bh` `B8 06 00`：站起來的生命 = 6 − 倒地計數（`+10Ch == 4` 時 `276Eh`
	// 直接給 6）。
	boarRallyCeiling = 6
	// trollRisesDice 是 `27EEh..27F4h`：Roll(3, 6)。
	trollRisesCount, trollRisesSides = 3, 6
	// boarFallsDice 是 `2794h..279Ah`：Roll(1, 4)，`27A1h` `40` 再加一。
	boarFallsCount, boarFallsSides = 1, 4
	// regenerationPendingDuration 是 `284Ah` `B8 03 00`。
	regenerationPendingDuration = 3
	// regenerationStep 是 `26FBh` `26 80 85 1B 01 03`。
	regenerationStep = 3
	// trollRisesRetryDuration 是 `288Dh` `B8 01 00`（`0021h(記錄, 66h, 節點 +3, 1)`）。
	trollRisesRetryDuration = 1
	// trollSparingFlags 是 `27D3h..27E3h`：`6777h` 的位元 0（火）或位元 4（10h）立著就不掛。
	trollSparingFlags = DamageFlagFire | 0x10
)

// DeathStrippedEffects 是 overlay-24 entry 13（`1004h`）逐一摘掉的十六個碼：`100Ah` 從 1 起，
// `102Fh` `80 7E FF 10` 比到 16 才停，每一輪 `mov al, [di+0C27h]` 讀 `DS:0C28h..0C37h`。
// START.EXE 檔案位移 `30640 + 0C28h` 的位元組是 `07 0B 1E 1F 20 33 34 35 36 3A 3B 5F 62 89 4A 4B`
// （同一塊讀 `DS:2880h` 得 `33 34 35 1F` 是正對照）。每個碼摘最早掛上的那一個，`+4` 立著就先以
// 模式 1 叫一次處理常式（entry 2 `0028h`）。
//
// 逃離盤面（`0F00h` 的 `0FC3h`）叫的也是這一支，所以與 EscapeStrippedEffects 是同一張表。
var DeathStrippedEffects = EscapeStrippedEffects

// BoarRallyHitPoints 是 `63h` 的 `2734h..2776h`：狀態 5（瀕死）而且 runtime `+0Eh` 小於 6 →
// 6 − `+0Eh`；狀態 4（昏迷）→ 6；其餘 0（不站起來）。
//
// `+0Eh` 在倒下的那一下由 overlay-25 entry 28（`2266h`）寫成「打穿了幾點」（`22E7h..22F2h`），
// 打穿 10 點以上直接是狀態 6、剛好 0 點是狀態 4（`22AFh..2301h`）。所以 remake 用倒下那一下的
// 打穿點數 overkill（生命值扣到的負數取正）換算：0..5 → 6 − overkill，其餘 0。
func BoarRallyHitPoints(overkill int) int {
	if overkill < 0 || overkill >= boarRallyCeiling {
		return 0
	}
	return boarRallyCeiling - overkill
}

// BoarRallied 是 `63h` 在 entry 22 回 1 之後那一段（`278Bh..27C5h`）：掛 `5Fh`（持續 Roll(1, 4) + 1、
// `+3` FFh、有收尾），`+4` 清 0 再用 entry 2 摘掉 `63h` 自己——只拚一次。
func BoarRallied(list EffectList, roll func(count, sides int) int) EffectList {
	duration := uint16(1)
	if roll != nil {
		duration = uint16(uint8(roll(boarFallsCount, boarFallsSides))) + 1
	}
	list = list.Append(NewEffectNode(BoarFallsEffectCode, duration, EffectUndispellable, true))
	return list.Remove(BoarRallyEffectCode)
}

// TrollRevival 是 `64h`（`27D0h`）：`6777h` 沒有位元 0、也沒有位元 4 → 掛 `66h`（持續 Roll(3, 6)、`+3` FFh、
// 有收尾）。`64h` 不摘自己，所以每倒一次都再掛一次。回傳改過的串列與有沒有掛。
func TrollRevival(list EffectList, damageFlags uint8, roll func(count, sides int) int) (EffectList, bool) {
	if !list.Has(TrollRevivalEffectCode) || damageFlags&trollSparingFlags != 0 || roll == nil {
		return list, false
	}
	duration := uint16(uint8(roll(trollRisesCount, trollRisesSides)))
	return list.Append(NewEffectNode(TrollRisesEffectCode, duration, EffectUndispellable, true)), true
}

// TrollRisesRetry 是 `66h` 站不起來時 `287Ch..2892h` 的 `0021h(記錄, 66h, 節點 +3, 1)`。
func TrollRisesRetry(list EffectList, level uint8) EffectList {
	return list.Append(NewEffectNode(TrollRisesEffectCode, trollRisesRetryDuration, level, true))
}

// TrollWounded 是 `65h`（`280Dh`，群組 5 與 6 都有）：身上沒有 `62h`、也沒有 `3Bh` → 掛 `3Bh`
// （持續 3、`+3` FFh、有收尾）。傷害不動。
func TrollWounded(list EffectList) EffectList {
	if !list.Has(TrollWoundEffectCode) || list.Has(RegenerationEffectCode) ||
		list.Has(RegenerationPendingEffectCode) {
		return list
	}
	return list.Append(NewEffectNode(RegenerationPendingEffectCode, regenerationPendingDuration,
		EffectUndispellable, true))
}

// RegenerationBegins 是 `3Bh` 的處理常式（`147Ch`，不看模式）：掛 `62h`（持續 0、`+3` FFh、不收尾）。
// `3Bh` 不在任何群組裡，所以只在收尾（到期或被摘）時跑。
func RegenerationBegins(list EffectList) EffectList {
	return list.Append(NewEffectNode(RegenerationEffectCode, 0, EffectUndispellable, false))
}

// Regenerate 是 `62h`（`26F5h`）：`+11Bh += 3`（byte），無號大於 `+32h` 就墊回 `+32h`。
// 派發它的是群組 19（overlay-08 `089Ah..08A3h`，回合收尾對每一個戰鬥者，在 entry 4 減計時之前）。
func Regenerate(list EffectList, hitPoints, maximum int) int {
	if !list.Has(RegenerationEffectCode) {
		return hitPoints
	}
	value := uint8(hitPoints) + regenerationStep
	if value > uint8(maximum) {
		value = uint8(maximum)
	}
	return int(value)
}

// IsDeathTeardownEffect 回答這個碼到期時要不要跑這一支的收尾（`3Bh`、`5Fh`、`66h`）。
func IsDeathTeardownEffect(code uint8) bool {
	return code == RegenerationPendingEffectCode || code == BoarFallsEffectCode || code == TrollRisesEffectCode
}

// paralysisAttack 是群組 2／3 裡叫 `15F7h` 的三個碼。`15F7h(記錄, 持續, byte)` 的第三個參數
// （`[bp+6]`）**整支沒有讀**：豁免推的是 `B0 00 50 / B0 00 50`（類別 0、修正 0），所以 `45h` 推的
// `B0 FE 50` 不作用。
//
//	43h  entry 62 `16A9h`  持續 Roll(2, 8)（`16B2h..16B8h`，先擲再豁免）
//	44h  entry 63 `16CDh`  目標 `+2Eh`（種族）是 2（精靈）就整支不做（`16DCh` `26 80 7D 2E 02 / 74 11`）；持續 3Fh
//	45h  entry 64 `16FAh`  持續 Roll(1, 9) + 0Ah（`1703h..1710h`）
type paralysisAttack struct {
	code        uint8
	count       int
	sides       int
	bonus       int
	fixed       uint16
	sparesElves bool
}

// 群組 3 的順序是 `40h 41h 42h 43h 44h 45h 46h …`，群組 2 是 `55h 56h 57h 44h …`。
var (
	paralysisSecondForm = [...]paralysisAttack{
		{code: 0x43, count: 2, sides: 8},
		{code: 0x44, fixed: 0x3f, sparesElves: true},
		{code: 0x45, count: 1, sides: 9, bonus: 0x0a},
	}
	paralysisFirstForm = [...]paralysisAttack{{code: 0x44, fixed: 0x3f, sparesElves: true}}
)

// elfRaceCode 是記錄 `+2Eh` 的精靈（creation 的 DOS 種族碼，spec 145）。
const elfRaceCode = 2

// paralysisLevel 是 `1656h` `B0 0C 50`：掛上去的 `34h` 的 `+3`。
const paralysisLevel = 0x0c

// ParalysisAttacks 是攻擊者身上帶著的麻痺碼，照群組順序各擲出持續（Roll 在豁免之前）。form 是攻擊
// 形態（1 → 群組 2、2 → 群組 3）；targetRace 是目標 `+2Eh`。回傳每一個碼的持續；0 個代表不必擲豁免。
func ParalysisAttacks(attacker EffectList, form int, targetRace uint8,
	roll func(count, sides int) int) []uint16 {
	attacks := paralysisFirstForm[:]
	if form == 2 {
		attacks = paralysisSecondForm[:]
	}
	var durations []uint16
	for _, attack := range attacks {
		if !attacker.Has(attack.code) {
			continue
		}
		if attack.sparesElves && targetRace == elfRaceCode {
			continue
		}
		duration := attack.fixed
		if attack.count > 0 {
			if roll == nil {
				continue
			}
			duration = uint16(uint8(roll(attack.count, attack.sides)) + uint8(attack.bonus))
		}
		durations = append(durations, duration)
	}
	return durations
}

// NewParalysisNode 是 `1648h..165Ch` 的 entry 10(目標, 34h, 持續, 0Ch, 0)：與定身術同一個碼，
// 沒有收尾，解除魔法照 `+3` 的 0Ch 擲。
func NewParalysisNode(duration uint16) EffectNode {
	return NewEffectNode(HoldPersonEffectCode, duration, paralysisLevel, false)
}
