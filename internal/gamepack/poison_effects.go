package gamepack

// 中毒（spec 153，issue #106）。輸入：overlay-12（SHA-256 `d1b05743…`）、overlay-22（`967065cc…`）、
// overlay-24（`e878166e…`）、overlay-32（`…`，spec 153 的表）、overlay-04（神殿）；`objdump -D -b binary
// -m i8086`（`coab-go-test:20260729`）。除註明者外皆 exact（位元組逐條讀）。
//
// 中毒在 DOS 版是**當場死亡**，不是逐時扣血：
//
//	1553h  f(豁免修正: byte; 記錄: far)                 ; 40h／41h／42h／46h 共用
//	155Ch  目標 = 記錄 +108h 的 +0Ah（攻擊者當下的目標）
//	157Dh  entry 7(目標, 類別 0, 修正)                   ; 豁免成功就返回
//	15A3h  印 "is Poisoned"、停一下
//	15C1h  entry 10(目標, 37h, 持續 0, 等級 FFh, 不收尾)  ; 永久，直到被解
//	15E1h  005Ah(目標, 6, "is killed")                  ; 狀態 6、生命 0、離場
//
// 緩毒術（overlay-22 `1846h`）把人暫時救回來，`0Fh`／`4Eh`／`16h` 三支處理常式負責之後的事：
//
//	0Fh  entry 17 `05F7h`：重掛自己（持續 0Ah）；生命 +11Bh 大於 1 → entry 19 打 1 點
//	4Eh  entry 71 `19F5h`：overlay-24 entry 22 `1869h` 讓倒下的人站起來；站不起來就重掛（持續 1）
//	16h  entry 23 `078Bh`：身上還有 37h → 005Ah(記錄, 6, "dies from poison")；然後摘掉 0Fh

const (
	// SlowPoisonEffectCode 是緩毒術（法術 26）的參數表 `+0Ah`：到期時 `078Bh` 看還有沒有中毒。
	SlowPoisonEffectCode uint8 = 0x16
	// PoisonDrainEffectCode 是緩毒術 `18D2h` 另外掛的 `0Fh`：每 0Ah 扣 1 點。
	PoisonDrainEffectCode uint8 = 0x0f
	// PoisonRecoveryEffectCode 是 `4Eh`：緩毒術 `18BBh` 以模式 1 叫它的常式，把中毒倒下的人扶起來。
	PoisonRecoveryEffectCode uint8 = 0x4e

	// poisonDrainDuration 是 `18DBh` `B8 0A 00` 與 `060Bh` `B8 0A 00`。
	poisonDrainDuration = 0x0a
	// poisonRecoveryRetryDuration 是 `1A24h` `B8 01 00`。
	poisonRecoveryRetryDuration = 1
	// poisonDrainHitPointFloor 是 `061Ah` `26 80 BD 1B 01 01 / 76 2D`：大於 1 才扣。
	poisonDrainHitPointFloor = 1
)

// poisonAttacks 是群組 3 裡會叫 `1553h` 的四個碼，照 `02E2h` 的呼叫順序
// （`40h 41h 42h 43h 44h 45h 46h …`），連同推給 `1553h` 的豁免修正：
//
//	40h  entry 59 `1667h`  `B0 00 50`
//	41h  entry 60 `167Dh`  `B0 04 50`
//	42h  entry 61 `1693h`  `B0 02 50`
//	46h  entry 65 `1721h`  `B0 FE 50`（entry 7 `0D92h` 以 `cbw` 讀，是 −2）
//
// 四個碼只在群組 3（第二攻擊形態命中之後，overlay-13 `1740h..174Dh`），不在群組 2。
var poisonAttacks = [...]struct {
	code     uint8
	modifier int
}{{0x40, 0}, {0x41, 4}, {0x42, 2}, {0x46, -2}}

// PoisonAttackSaves 是攻擊者身上帶著的毒碼，照群組 3 的順序各給一個豁免修正。每個碼只問一次
// （`014Dh` 找最早掛上的那一個）；這四個碼不在 `CS:012Dh` 的作用範圍集合裡，只看自己身上。
func PoisonAttackSaves(attacker EffectList) []int {
	var modifiers []int
	for _, attack := range poisonAttacks {
		if attacker.Has(attack.code) {
			modifiers = append(modifiers, attack.modifier)
		}
	}
	return modifiers
}

// NewPoisonNode 是 `15B5h..15C1h` 掛的節點：`37h`、持續 0（永久，spec 069）、`+3` FFh、不收尾。
func NewPoisonNode() EffectNode {
	return NewEffectNode(PoisonEffectCode, 0, EffectUndispellable, false)
}

// NewPoisonDrainNode 是緩毒術 `18D2h..18E5h` 掛的 `0Fh`：持續 0Ah、`+3` FFh、有收尾。
func NewPoisonDrainNode() EffectNode {
	return NewEffectNode(PoisonDrainEffectCode, poisonDrainDuration, EffectUndispellable, true)
}

// PoisonRecoveryRetry 是 `4Eh` 站不起來時 `1A13h..1A29h` 的 `0021h(記錄, 4Eh, 節點 +3, 1)`：
// 接一個持續 1、有收尾的 `4Eh` 在尾端，下一刻再試。
func PoisonRecoveryRetry(list EffectList, level uint8) EffectList {
	return list.Append(NewEffectNode(PoisonRecoveryEffectCode, poisonRecoveryRetryDuration, level, true))
}

// PoisonTeardown 是一個節點到期時 `0Fh`／`16h`／`4Eh` 那一支做完的結果。
type PoisonTeardown struct {
	// Effects 是跑完之後的串列（`0Fh` 重掛在尾端、`16h` 摘掉 `0Fh`）。
	Effects EffectList
	// HitPoints 是改過之後的目前生命值；Drained 為真時 entry 19 打了 1 點。
	HitPoints int
	Drained   bool
	// DiesFromPoison 為真時呼叫端要跑 `005Ah(記錄, 6, "dies from poison")`。
	DiesFromPoison bool
	// Recover 為真時呼叫端要跑 overlay-24 entry 22（`1869h`）；站不起來再 PoisonRecoveryRetry。
	Recover bool
}

// PoisonTeardownOf 是三支常式的模式 1。node 是到期的那一個，list 是摘掉它之後的串列。
// 不是這三個碼的原樣回傳。
//
// `0021h` 在 `DS:677Dh` 立著時不重掛：只有治療那幾條路摘節點時立它，remake 那幾條路摘節點
// 不跑收尾，等同於立著（同致病，disease_effects.go）。
func PoisonTeardownOf(node EffectNode, list EffectList, hitPoints int) PoisonTeardown {
	result := PoisonTeardown{Effects: list, HitPoints: hitPoints}
	switch node.Code {
	case PoisonDrainEffectCode:
		// `05FAh..0610h`：0021h(記錄, 0Fh, 節點 +3, 0Ah)；回 1 才往下。
		result.Effects = list.Append(NewEffectNode(PoisonDrainEffectCode, poisonDrainDuration,
			node.Payload[effectNodeLevelOffset], true))
		if hitPoints > poisonDrainHitPointFloor {
			result.HitPoints--
			result.Drained = true
		}
	case SlowPoisonEffectCode:
		// `0791h..07C1h`：`010Ah:00A7h(記錄, 37h)` 找得到 → 005Ah；`07C4h..07DDh` 摘 0Fh
		// （`677Dh` 立著，0Fh 的收尾不重掛也不扣血）。
		result.DiesFromPoison = list.Has(PoisonEffectCode)
		result.Effects = list.Remove(PoisonDrainEffectCode)
	case PoisonRecoveryEffectCode:
		result.Recover = true
	}
	return result
}

// IsPoisonTeardownEffect 回答這個碼是不是這一串的。
func IsPoisonTeardownEffect(code uint8) bool {
	return code == PoisonDrainEffectCode || code == SlowPoisonEffectCode || code == PoisonRecoveryEffectCode
}

// NeutralizePoison 是神殿的 Neutralize Poison（overlay-04 entry 8 `0736h`）與編號 58（overlay-22
// `2E02h`）的中毒那一支共同的結果：摘掉 `37h`、`16h`、`0Fh`，都在 `677Dh` 立著的時候摘，不跑重掛。
// 編號 58 只摘 `37h`（entry 15）與 `16h`，`0Fh` 是 `16h` 的收尾 `078Bh` 摘的——結果相同。
// **不把人救活**：兩支都不動 `+10Ch`。回傳串列與有沒有中毒。
func NeutralizePoison(list EffectList) (EffectList, bool) {
	if !list.Has(PoisonEffectCode) {
		return list, false
	}
	list = list.Remove(PoisonEffectCode)
	list = list.Remove(SlowPoisonEffectCode)
	list = list.Remove(PoisonDrainEffectCode)
	return list, true
}

// SpellDamageDice 是傷害走 overlay-24 entry 9（`0100h:004Dh`，會寫 `DS:677Ah`）的那幾支法術擲的骰數
// （全部 overlay 掃 `9A 4D 00 00 01` 的窮舉，spec 153〈677Ah〉）：
//
//	致輕傷 `10A5h`（1, 8）、魔法飛彈 `1454h`（等級 ÷ 2, 4）、電擊之握 `14E1h`（1, 8）、
//	火球 `2704h`（等級, 6）、編號 65 `3024h`（2, 4）
//
// 燃燒之手（不擲骰）、閃電束與編號 60 的射線（entry 8）不寫，回 false。
func SpellDamageDice(id uint8, casterLevel int) (uint8, bool) {
	switch id {
	case SpellIDCauseLightWound, SpellIDShockingGrasp:
		return 1, true
	case SpellIDMagicMissile:
		return uint8(casterLevel / 2), true
	case SpellIDFireball, SpellIDFireballAlt:
		return uint8(casterLevel), true
	case SpellIDMagicMissileAlt:
		return 2, true
	}
	return 0, false
}
