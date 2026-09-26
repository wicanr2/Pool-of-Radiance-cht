package gamepack

// 命中擲骰時效果系統對命中骰 `DS:6780h` 的調整（spec 112〈群組 10／16：命中擲骰〉，issue #86）。
//
// overlay-24 entry 6（`0CB5h`，`retf 0Ah`，參數是攻擊者、目標、目標的 AC）：
//
//	0CBFh  0FCCh(攻擊者)：攻擊者身上的 19h 全部摘掉（`0FD8h` 推 19h 查、`0FF9h` 摘，迴圈到沒有）
//	0CC9h  6780h = 0DE5h(1, 14h)            ; 1d20
//	0CD6h  6780h <= 1 → 落空（不問效果系統）
//	0CDDh  6780h == 14h → 6780h = 64h
//	0CE9h  02E2h(0Ah, 攻擊者)               ; 群組 10
//	0CF6h  02E2h(10h, 目標)                 ; 群組 16
//	0D28h  6780h < 0（有號 byte）→ 落空
//	0D2Fh  cbw(6780h) + 攻擊者 +110h + 邊的加成 >= AC → 命中
//
// 處理常式的簽章是 f(記錄, 節點, 模式)，記錄 `[bp+0Ch]`、節點 `[bp+08h]`。群組 10 交給它們的
// 記錄是攻擊者，群組 16 是目標。來源一律是 overlay-12（SHA-256 `d1b05743…`）與
// overlay-24（`e878166e…`），位元組逐條讀過；各碼的位址寫在下面的 case 上。

// 群組 10 與群組 16 的代碼，照 overlay-24 `02E2h` 的呼叫順序（spec 112 的二十組表）。
var (
	attackerHitGroup = [...]uint8{0x01, 0x02, 0x21, 0x24, 0x31, 0x03, 0x06, 0x12, 0x1a}
	targetHitGroup   = [...]uint8{0x19, 0x47, 0x25, 0x2f, 0x30, 0x59}
)

// 代碼。
const (
	// InvisibilityEffectCode 是 `19h`（隱形術、參數表 `+0Ah`）：`0FCCh` 在每一次命中擲骰
	// 之前從攻擊者身上摘掉，群組 16 的處理常式（overlay-12 entry 25 `0927h`）讓對它出手 −4。
	InvisibilityEffectCode uint8 = 0x19
	// PrayerAreaEffectCode 是 `31h`（祈禱術的參數表 `+0Ah`）。它是 `014Dh` 那四個有作用範圍
	// 的碼之一，半徑 6。
	PrayerAreaEffectCode uint8 = 0x31
	// PhaseEffectCode 是 `25h`（相位蜘蛛身上帶的、閃現術的參數表 `+0Ah`）。
	PhaseEffectCode uint8 = 0x25
	// DisplacementEffectCode 是 `59h`（幻影移位，提拉尼薩克斯身上帶的）。
	DisplacementEffectCode uint8 = 0x59

	// prayerSideBit 是祈禱節點 `+3` 的位元 4：`249Dh` 推 `(施法者 +10Eh << 4) + 等級`
	// 在等級覆寫那一格（spec 098），`12CAh` 讀回來當成施法者那一邊。
	prayerSideBit = 0x10
	// displacementSpentBit 是幻影移位節點 `+3` 的位元 4：這一回已經讓一擊落空過了。
	displacementSpentBit = 0x10
)

// 生物種類（記錄 `+9Fh`）與這幾支常式比的值。
const (
	creatureTypeHumanoid = 1
	creatureTypeUndead   = 4
	creatureTypeNine     = 9
	creatureTypeTroll    = 0x0a
	creatureTypeTwelve   = 0x0c
)

// 名字表：`12h`／`1Ah` 比的是 START.EXE 資料段裡的 string[15] 陣列（每格 16 bytes），
// `30h` 比的是 overlay-12 自己 CS 裡的兩個字串。比對用 RTL `05BBh:0724h`（Pascal 字串相等）。
var (
	// GnomeFoeNames 是 `DS:0356h..0395h`（`06D7h` 的 `add di, 346h`，i = 1..4）。
	GnomeFoeNames = [...]string{"KOBOLD", "KOBOLD LEADER", "GOBLIN", "GOBLIN LEADER"}
	// DwarfFoeNames 是 `DS:0396h..0415h`（`09A2h` 的 `add di, 386h`，i = 1..8）。第 8 格是
	// "MACE"——原版的迴圈就是到 8，照搬。
	DwarfFoeNames = [...]string{"ORC", "ORC LEADER", "GOBLIN", "GOBLIN LEADER",
		"HOBGOBLIN", "HOBGOBLIN CHIEF", "GAGOOL", "MACE"}
	// GnomeLargeFoeNames 是 overlay-12 CS 的 `1275h`／`127Dh`（`07 "BUGBEAR"`、`05 "GNOLL"`）。
	GnomeLargeFoeNames = [...]string{"BUGBEAR", "GNOLL"}
)

// GnomeFoeNamesAddress 與 DwarfFoeNamesAddress 是兩張表第 1 格的 DS 位址（i = 1 那一格；
// 常式寫的基底是它減 16）。
const (
	GnomeFoeNamesAddress = 0x0356
	DwarfFoeNamesAddress = 0x0396
)

// HitRollCombatant 是處理常式在命中擲骰時會讀到的一個戰鬥者。
type HitRollCombatant struct {
	Effects      EffectList
	CreatureType uint8  // 記錄 `+9Fh`
	BodySize     uint8  // 記錄 `+6Ch`
	Name         string // 記錄 `+0` 的 Pascal 字串
	Side         uint8  // 記錄 `+10Eh`
	Score        uint8  // runtime `+3`（先攻分數）
}

// HitRollEffects 是一次命中擲骰問效果系統時的全部輸入。
type HitRollEffects struct {
	// Attacker 是群組 10 的記錄；Target 是群組 16 的記錄，也是攻擊者 runtime `+0Ah`
	// 指到的那一個（群組 10 的 03h／06h／12h／1Ah 經 `DS:6784h` 讀它）。
	Attacker, Target HitRollCombatant
	// Actor 是 `DS:5CF0h`，輪到行動的那一個。一般攻擊就是攻擊者；反應攻擊時是正在
	// 離開的那一個（spec 059）。`30h` 讀的是它，不是攻擊者。
	Actor HitRollCombatant
	// AttackPhase 是 `DS:6CD7h`（spec 051 的相位計數，戰鬥開始是 0）。
	AttackPhase uint8
	// AreaNode 是 `014Dh` 的第二條路：攻擊者自己沒帶 `31h` 時，找一個站得夠近的帶著的人，
	// 回他的節點。沒有就回 false。可以是 nil。
	AreaNode func(code uint8) (EffectNode, bool)
}

// HitRollBase 是 `0CD3h..0CE4h`：擲出來的 d20，自然 20 改寫成 100。
func HitRollBase(roll uint8) int8 {
	if roll == 20 {
		return 100
	}
	return int8(roll)
}

// Apply 讓命中骰依序過群組 10（攻擊者）與群組 16（目標）。回傳調整後的 `DS:6780h`
// 與目標改過的效果串列（`59h` 會改自己節點的 `+3`）。呼叫端只在 d20 大於 1 時叫它——
// `0CDBh` 在那之前就跳走了，處理常式一支都不會跑。
func (effects HitRollEffects) Apply(value int8) (int8, EffectList) {
	for _, code := range attackerHitGroup {
		node, ok := effects.attackerNode(code)
		if !ok {
			continue
		}
		value = effects.applyAttackerCode(code, node, value)
	}
	target := effects.Target.Effects
	for _, code := range targetHitGroup {
		index, ok := target.IndexOf(code)
		if !ok {
			continue
		}
		value, target = effects.applyTargetCode(code, index, target, value)
	}
	return value, target
}

// attackerNode 是 `014Dh`：先看攻擊者自己（overlay-25 entry 27 的線性搜尋，最早掛上的那一個），
// 沒有而代碼是 `31h` 時才走作用範圍那一條。
func (effects HitRollEffects) attackerNode(code uint8) (EffectNode, bool) {
	if index, ok := effects.Attacker.Effects.IndexOf(code); ok {
		return effects.Attacker.Effects[index], true
	}
	if code != PrayerAreaEffectCode || effects.AreaNode == nil {
		return EffectNode{}, false
	}
	return effects.AreaNode(code)
}

func (effects HitRollEffects) applyAttackerCode(code uint8, node EffectNode, value int8) int8 {
	target := effects.Target
	switch code {
	case BlessEffectCode:
		// entry 5 `010Fh`：`80 06 83 67 05`（士氣格 +5）、`FE 06 80 67`。
		value++
	case CurseEffectCode:
		// entry 6 `0121h`：士氣格夾在 0 以上減 5，`FE 0E 80 67`。
		value--
	case 0x21:
		// entry 31 `0BBEh`：`80 2E 80 67 04`。同一支還把記錄 `+111h`／`+112h` 各減 4、
		// `DS:6774h` 減 4——那兩格 remake 沒接（spec 112〈OPEN〉）。
		value -= 4
	case 0x24:
		// entry 34 `0C2Dh`：`80 2E 80 67 04`、`80 2E 74 67 04`。
		value -= 4
	case PrayerAreaEffectCode:
		// entry 46 `12C1h`：邊 = (節點 +3 and 10h) ÷ 10h；記錄 `+10Eh` 等於它就叫 `0C1Ch`
		// （`FE 06 74 67`、`FE 06 80 67`），不等就 `FE 0E 80 67`、`FE 0E 74 67`。
		if effects.Attacker.Side == (node.Payload[effectNodeLevelOffset]&prayerSideBit)/prayerSideBit {
			value++
		} else {
			value--
		}
	case 0x03:
		// entry 7 `0141h`：`DS:6784h` = 記錄 +108h 的 +0Ah，`+9Fh == 4` →
		// `80 06 80 67 02`（傷害格 `6776h` 也 +2，但近戰傷害在這之後才擲，spec 050）。
		if target.CreatureType == creatureTypeUndead {
			value += 2
		}
	case 0x06:
		// entry 9 `01C9h`：依 `+9Fh` 0Ah → 1、9／0Ch → 2、4 → 3，其餘 0，`00 06 80 67`。
		switch target.CreatureType {
		case creatureTypeTroll:
			value++
		case creatureTypeNine, creatureTypeTwelve:
			value += 2
		case creatureTypeUndead:
			value += 3
		}
	case 0x12:
		// entry 20 `068Bh`：`+9Fh == 1` 且 `+6Ch == 1`，名字在表上就 `FE 06 80 67`。
		value += int8(namedFoeBonus(target, GnomeFoeNames[:]))
	case 0x1a:
		// entry 26 `0956h`：同一個形狀，八個名字。
		value += int8(namedFoeBonus(target, DwarfFoeNames[:]))
	}
	return value
}

func namedFoeBonus(target HitRollCombatant, names []string) int {
	if target.CreatureType != creatureTypeHumanoid || target.BodySize != 1 {
		return 0
	}
	bonus := 0
	for _, name := range names {
		if target.Name == name {
			bonus++
		}
	}
	return bonus
}

func (effects HitRollEffects) applyTargetCode(code uint8, index int, list EffectList,
	value int8) (int8, EffectList) {
	switch code {
	case InvisibilityEffectCode, 0x47:
		// entry 25 `0927h`：`80 2E 80 67 04`（否決旗標 `677Ch` 只給 AI 挑目標用）；
		// entry 66 `1737h`：同一個 −4。
		value -= 4
	case PhaseEffectCode:
		// entry 35 `0C40h`：目標 runtime `+3` 有號大於 0 → `C6 06 80 67 FF`。
		if int8(effects.Target.Score) > 0 {
			value = -1
		}
	case 0x2f:
		// entry 44 `1208h`：比對的是**目標自己**的 runtime `+0Ah`（它正在追的那一個）的
		// 名字，remake 沒有隊員那一格。未接（spec 112〈OPEN〉）。
	case 0x30:
		// entry 45 `1283h`：`DS:5CF0h` 的 `+9Fh == 1`，名字是 BUGBEAR 或 GNOLL →
		// `80 2E 80 67 04`。
		actor := effects.Actor
		if actor.CreatureType == creatureTypeHumanoid {
			for _, name := range GnomeLargeFoeNames {
				if actor.Name == name {
					value -= 4
					break
				}
			}
		}
	case DisplacementEffectCode:
		// entry 83 `2461h`：相位 `6CD7h` 為 0 而且命中骰剛好是 0 → 節點 `+3` and 0Fh；
		// 否則節點 `+3` 位元 4 還沒立 → 命中骰寫 FFh、立起位元 4。
		level := list[index].Payload[effectNodeLevelOffset]
		switch {
		case effects.AttackPhase == 0 && value == 0:
			level &= 0x0f
		case level&displacementSpentBit == 0:
			value = -1
			level |= displacementSpentBit
		default:
			return value, list
		}
		list = append(EffectList(nil), list...)
		list[index].Payload[effectNodeLevelOffset] = level
	}
	return value, list
}

// DropInvisibility 是 `0FCCh`：只要攻擊者身上還找得到 `19h`，就摘掉最早的那一個再找。
func DropInvisibility(list EffectList) EffectList {
	for list.Has(InvisibilityEffectCode) {
		list = list.Remove(InvisibilityEffectCode)
	}
	return list
}
