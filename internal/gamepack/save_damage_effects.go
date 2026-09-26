package gamepack

// 群組 12（豁免）、群組 6（法術傷害）、群組 4／5（近戰傷害）裡，#89 之後接上的碼
// （spec 112〈群組 12／6／4／5〉，issue #96／#99）。
//
// 輸入：overlay-12（SHA-256 `d1b05743…`）、overlay-13（`4d53df20…`）、overlay-22（`967065cc…`）、
// overlay-24（`e878166e…`）；`objdump -D -b binary -m i8086`（`coab-go-test:20260729`）。
// 位址是 overlay 檔內位移。除註明者外皆 exact（位元組逐條讀）。
//
// 處理常式的簽章是 f(記錄 `[bp+0Ch]`, 節點 `[bp+08h]`, 模式 `[bp+06h]`)，`retf 0Ah`。
// 累加格全部是 byte：`DS:6774h` 豁免骰、`DS:6776h` 傷害、`DS:6777h` 傷害種類、
// `DS:6788h` 這一次的豁免類別、`DS:6779h` 正在處理的法術、`DS:677Eh` 這一份目標是以一點收的。

// 這一批的碼。
const (
	// ProtectionFromEvilEffectCode 是 `08h`（防護邪惡，法術 6／16）；`2Dh` 是 10 呎半徑版（52）。
	// 兩個碼共用 overlay-12 entry 11 `0377h`。
	ProtectionFromEvilEffectCode     uint8 = 0x08
	ProtectionFromEvilAreaEffectCode uint8 = 0x2d
	// ProtectionFromGoodEffectCode 是 `09h`（防護善良，7／17）；`2Eh` 是半徑版（53）。
	// 共用 entry 12 `03AEh`。
	ProtectionFromGoodEffectCode     uint8 = 0x09
	ProtectionFromGoodAreaEffectCode uint8 = 0x2e
	// ResistColdEffectCode 是 `0Ah`（抗寒，8）：entry 13 `03E5h`。
	ResistColdEffectCode uint8 = 0x0a
	// ResistFireEffectCode 是 `14h`（抗火，24）：entry 21 `06F4h`。
	ResistFireEffectCode uint8 = 0x14
	// MirrorImageEffectCode 是 `1Ch`（鏡影，32）：entry 27 `09CDh`。節點 `+3` 是影像數。
	MirrorImageEffectCode uint8 = 0x1c
	// EnfeeblementEffectCode 是 `1Dh`（衰弱射線，33）：entry 28 `0A4Ah`。
	EnfeeblementEffectCode uint8 = 0x1d
	// RacePoisonSaveEffectCode 是 `5Ah`（矮人、半身人，spec 145）：entry 84 `24ACh`。
	RacePoisonSaveEffectCode uint8 = 0x5a
	// RaceMagicSaveEffectCode 是 `61h`（矮人、侏儒、半身人）：entry 90 `2673h`。
	RaceMagicSaveEffectCode uint8 = 0x61
)

// `DS:6777h` 的位元（`08BCh` 在 `08E2h` 從處理常式推的 `[bp+0Ah]` 寫入，傷害為 0 時 `08DBh` 寫 0；
// 射線的 `287Ch` 在 `28DCh` 寫 0Ch）。
const (
	// DamageFlagFire 是位元 0：`06F7h` `A0 77 67 / 24 01` 抗火讀它。
	DamageFlagFire uint8 = 0x01
	// DamageFlagCold 是位元 1：`03E8h` `A0 77 67 / 24 02` 抗寒讀它。全部 overlay 掃
	// `C6 06 77 67`／`A2 77 67` 的寫入與 `08BCh` 的推入值（8、9、0Ch），**沒有一處立這一位**，
	// 所以抗寒在 DOS 版的法術與已知特殊攻擊上都不會作用（exact：寫入點的窮舉）。
	DamageFlagCold uint8 = 0x02
	// DamageFlagMagic 是位元 3：群組 9 的 `2910h` 讀它決定要不要擲魔法抗性。
	DamageFlagMagic uint8 = 0x08
)

// SpellDamageKind 是處理常式推給 `08BCh` 的第五個參數（`[bp+0Ah]`，寫進 `DS:6777h`）。
// 只有傷害不為 0 時才寫；其餘法術在 `08DBh` 寫 0。
//
//	4  致輕傷    `10ABh` `B0 08 50`
//	9  燃燒之手  `1192h` `B0 09 50`
//	15 魔法飛彈  `146Bh` `B0 08 50`
//	20 電擊之握  `14ECh` `B0 0C 50`
//	47／64 火球  `270Ah` `B0 09 50`
//	65           `302Fh` `B0 08 50`
//	51／60 射線  `287Ch` 自己在 `28DCh` 寫 `C6 06 77 67 0C`
func SpellDamageKind(spell uint8) uint8 {
	switch spell {
	case SpellIDCauseLightWound, SpellIDMagicMissile, SpellIDMagicMissileAlt:
		return DamageFlagMagic
	case SpellIDBurningHands, SpellIDFireball, SpellIDFireballAlt:
		return DamageFlagMagic | DamageFlagFire
	case SpellIDShockingGrasp, SpellIDLightningBolt, spellIDRay3C:
		return 0x0c
	}
	return 0
}

// spellIDRay3C 是編號 60（`2F02h`，與閃電束同形的射線）。
const spellIDRay3C = 60

// ConstitutionSaveBonus 是 `5Ah`／`61h` 共用的那一段（`24B9h..2510h`／`2687h..26DEh`）：
// 記錄 `+14h`（體質）4..6 → 1、7..10 → 2、11..13 → 3、14..17 → 4、18..20 → 5。
// 範圍外的值不寫 `[bp-1]`，加上去的是堆疊上殘留的東西——ok 為假，remake 不加（unknown）。
// 建角的體質下限是 3，只有 3 會落在範圍外。
func ConstitutionSaveBonus(constitution uint8) (bonus uint8, ok bool) {
	switch {
	case constitution >= 4 && constitution <= 6:
		return 1, true
	case constitution >= 7 && constitution <= 10:
		return 2, true
	case constitution >= 11 && constitution <= 13:
		return 3, true
	case constitution >= 14 && constitution <= 17:
		return 4, true
	case constitution >= 18 && constitution <= 20:
		return 5, true
	}
	return 0, false
}

// 群組 12 的豁免類別（記錄 `+6Dh` 起的索引，spec 075）。
const (
	saveCategoryPoison = 0 // 癱瘓／毒／死亡魔法
	saveCategoryWand   = 2 // 法杖／魔杖／權杖
	saveCategorySpell  = 4 // 法術
)

// 陣營（記錄 `+0A0h`）的三個「善」與三個「惡」。值的順序與 creation.Alignments 相同
// （嚴守善良 0 … 混亂邪惡 8）；怪物樣板 GOBLIN／ORC／HOBGOBLIN 是 2、OGRE／VAMPIRE 是 8、
// GIANT SKELETON 是 4，與 AD&D 怪物圖鑑一致（正對照，MON1CHA.DAX）。
var (
	evilAlignments = [...]uint8{2, 5, 8} // `037Eh` `3C 02`、`038Ah` `05`、`0396h` `08`
	goodAlignments = [...]uint8{0, 3, 6} // `03B5h` `00`、`03C1h` `03`、`03CDh` `06`
)

// SaveRollEffects 是 overlay-24 entry 7 在 `0DB2h` 派發群組 12 時，處理常式會讀到的東西。
type SaveRollEffects struct {
	// Effects 是擲豁免那一個身上的串列。
	Effects EffectList
	// Category 是 `DS:6788h`（`0DACh` 寫入的豁免類別）。
	Category uint8
	// Side 是擲豁免那一個的 `+10Eh`（`31h` 比它）。
	Side uint8
	// Constitution 是擲豁免那一個的記錄 `+14h`，0 代表 remake 不知道。
	Constitution uint8
	// ActorAlignment 是 `DS:5CF0h`（輪到行動的那一個，施法時就是施法者）的 `+0A0h`；
	// ActorAlignmentKnown 為假時 `08h 09h 2Dh 2Eh` 不作用。
	ActorAlignment      uint8
	ActorAlignmentKnown bool
	// DamageFlags 是 `DS:6777h`。
	DamageFlags uint8
	// AreaNode 是 `014Dh` 的第二條路（自己沒帶、站在別人的作用範圍裡）。只對 `2Dh 2Eh 31h`
	// 成立（`CS:012Dh` 的集合）；半徑 `2Dh`／`2Eh` 是 1、`31h` 是 6，由呼叫端依碼決定。可以是 nil。
	AreaNode func(code uint8) (EffectNode, bool)
}

// saveRollGroup 是群組 12 的呼叫順序（spec 112 的二十組表）。
var saveRollGroup = [...]uint8{0x08, 0x09, 0x0a, 0x11, 0x14, 0x21, 0x24, 0x2d, 0x2e, 0x31,
	0x3d, 0x6f, 0x7d, 0x5a, 0x61}

// Apply 讓豁免骰 `DS:6774h` 依序過群組 12。全部是 byte 運算，回傳值是 byte；entry 7 的
// `0DC8h` `26 8A 45 6D / 3A 06 74 67 / 77 06` 拿它與目標值做**無號**比較（目標值大於它就失敗）。
//
// `3Dh 6Fh 7Dh` 不是法術或種族掛的，沒有接（spec 112〈OPEN〉）。
func (save SaveRollEffects) Apply(value uint8) uint8 {
	for _, code := range saveRollGroup {
		node, ok := save.node(code)
		if !ok {
			continue
		}
		value = save.applyCode(code, node, value)
	}
	return value
}

func (save SaveRollEffects) node(code uint8) (EffectNode, bool) {
	if index, ok := save.Effects.IndexOf(code); ok {
		return save.Effects[index], true
	}
	switch code {
	case ProtectionFromEvilAreaEffectCode, ProtectionFromGoodAreaEffectCode, PrayerAreaEffectCode:
		if save.AreaNode != nil {
			return save.AreaNode(code)
		}
	}
	return EffectNode{}, false
}

func (save SaveRollEffects) applyCode(code uint8, node EffectNode, value uint8) uint8 {
	switch code {
	case ProtectionFromEvilEffectCode, ProtectionFromEvilAreaEffectCode:
		// entry 11 `0377h`：`5CF0h` 的 `+0A0h` 是 2／5／8 → `80 06 74 67 02`（`80 2E 80 67 02`
		// 的命中骰 −2 在這個時點沒有讀者）。
		if save.ActorAlignmentKnown && alignmentIn(save.ActorAlignment, evilAlignments[:]) {
			value += 2
		}
	case ProtectionFromGoodEffectCode, ProtectionFromGoodAreaEffectCode:
		// entry 12 `03AEh`：0／3／6 → +2。
		if save.ActorAlignmentKnown && alignmentIn(save.ActorAlignment, goodAlignments[:]) {
			value += 2
		}
	case ResistColdEffectCode:
		// entry 13 `03E5h`：`6777h & 2` → 傷害格減半（在這個時點會被 entry 19 的 `1344h` 蓋掉）、
		// `80 06 74 67 03`。
		if save.DamageFlags&DamageFlagCold != 0 {
			value += 3
		}
	case ResistFireEffectCode:
		// entry 21 `06F4h`：`6777h & 1` → 同上。
		if save.DamageFlags&DamageFlagFire != 0 {
			value += 3
		}
	case ShieldEffectCode:
		value++ // entry 19 `0675h` `FE 06 74 67`
	case BlindnessEffectCode, BestowCurseEffectCode:
		value -= 4 // entry 31／34 `80 2E 74 67 04`
	case PrayerAreaEffectCode:
		// entry 46 `12C1h` → `0C1Ch`：節點 `+3` 位元 4 等於自己的 `+10Eh` +1，否則 −1。
		if save.Side == (node.Payload[effectNodeLevelOffset]&prayerSideBit)/prayerSideBit {
			value++
		} else {
			value--
		}
	case RacePoisonSaveEffectCode:
		// entry 84 `24B2h` `80 3E 88 67 00 / 75 68`：只在類別 0（毒）。
		if save.Category == saveCategoryPoison {
			value = addConstitutionBonus(value, save.Constitution)
		}
	case RaceMagicSaveEffectCode:
		// entry 90 `2679h` `80 3E 88 67 04 / 74 07 / 80 3E 88 67 02 / 75 68`：類別 4（法術）
		// 與 2（魔杖）。
		if save.Category == saveCategorySpell || save.Category == saveCategoryWand {
			value = addConstitutionBonus(value, save.Constitution)
		}
	}
	return value
}

// addConstitutionBonus 是 `2510h..251Eh`（`26DEh..26ECh`）：`al = 6774h; xor ah; add ax, dx;
// mov 6774h, al`。
func addConstitutionBonus(value, constitution uint8) uint8 {
	if bonus, ok := ConstitutionSaveBonus(constitution); ok {
		value += bonus
	}
	return value
}

func alignmentIn(alignment uint8, set []uint8) bool {
	for _, value := range set {
		if alignment == value {
			return true
		}
	}
	return false
}

// SpellDamageEffects 是 overlay-24 entry 19（`133Ah`）在 `1351h` 派發目標的群組 6 時，處理常式
// 讀到的東西。entry 19 先 `1344h` 把傷害寫進 `DS:6776h`，派發群組 6，**然後**才套豁免規則
// （`1354h`）。
type SpellDamageEffects struct {
	// Effects 是受傷那一個身上的串列。
	Effects EffectList
	// Spell 是 `DS:6779h`（派發 `0E95h` 寫、`0EA7h` 清）。近戰與地圖上的傷害是 0。
	Spell uint8
	// DamageFlags 是 `DS:6777h`。
	DamageFlags uint8
	// Area 是 `DS:677Eh`：`20AEh` 以一點收表（模式 8..0Eh、0Fh 選了空格）時立 1，火球 `2634h`、
	// 射線 `2987h` 自己也立。
	Area bool
	// Roll 是 overlay-24 entry 8（`0100h:0048h`）。
	Roll func(count, sides int) int
}

// SpellDamageOutcome 是群組 6 之後的結果。
type SpellDamageOutcome struct {
	Damage int
	// Effects 是受傷那一個改過的串列（鏡影少一個影像，用完就摘）。
	Effects EffectList
	// LostImage 為真時原版印 "lost an image"（`09BFh`）並停一下。
	LostImage bool
}

// Apply 照群組 6 的順序（`71h 3Dh 7Ah 3Ch 5Bh 0Ah 14h 69h 6Ah 70h 72h 76h 11h 5Dh 65h 1Ch`）
// 跑 remake 接了的四個碼：`0Ah`、`14h`、`11h`、`1Ch`。
func (damage SpellDamageEffects) Apply(value int) SpellDamageOutcome {
	outcome := SpellDamageOutcome{Damage: value, Effects: damage.Effects}
	list := damage.Effects
	if list.Has(ResistColdEffectCode) && damage.DamageFlags&DamageFlagCold != 0 {
		outcome.Damage = halveDamage(outcome.Damage) // `03F1h..03FCh`
	}
	if list.Has(ResistFireEffectCode) && damage.DamageFlags&DamageFlagFire != 0 {
		outcome.Damage = halveDamage(outcome.Damage) // `0700h..070Bh`
	}
	if list.Has(ShieldEffectCode) && damage.Spell == SpellIDMagicMissile {
		outcome.Damage = 0 // `067Eh` `80 3E 79 67 0F / 75 05 / C6 06 76 67 00`
	}
	var lost bool
	outcome.Effects, lost = MirrorImageAbsorbs(list, damage.Spell, damage.Area, damage.Roll)
	if lost {
		outcome.Damage = 0
		outcome.LostImage = true
	}
	return outcome
}

// halveDamage 是 `A0 76 67 / 30 E4 / 99 / B9 02 00 / F7 F9 / A2 76 67`：byte 零延伸再除 2。
func halveDamage(value int) int {
	return int(uint8(value)) / 2
}

// MirrorImageAbsorbs 是 `1Ch` 的處理常式（overlay-12 entry 27 `09CDh`），群組 5（近戰，目標）與
// 群組 6（法術傷害，目標）都叫它：
//
//	09D3h  Roll(1, 節點 +3 + 1)                 ; 一定先擲
//	09E6h  <= 1 → 返回（打到本尊）
//	09EAh  DS:6779h == 0 → 返回                 ; 不是法術
//	09F1h  DS:677Eh != 0 → 返回                 ; 以一點收的範圍
//	09F8h  0000h(0)：DS:6776h = 0               ; 這一下打在影像上
//	0A0Fh  印 "lost an image"、停一下
//	0A22h  節點 +3 減一；0A29h 為 0 → entry 2 摘掉（`+4` 是 0，不叫收尾）
//
// `DS:6779h` 只有法術派發（overlay-22 `0E95h` 寫、`0EA7h` 清）與 overlay-12 `1F7Eh` 寫，所以
// 近戰那一次（群組 5）影像擋不下來，只會多擲一次骰（strong inference：寫入點是 byte 掃描
// `C6 06 79 67`／`A2 79 67` 的窮舉）。回傳改過的串列與這一下有沒有被影像吃掉。
func MirrorImageAbsorbs(list EffectList, spell uint8, area bool,
	roll func(count, sides int) int) (EffectList, bool) {
	index, ok := list.IndexOf(MirrorImageEffectCode)
	if !ok || roll == nil {
		return list, false
	}
	images := list[index].Payload[effectNodeLevelOffset]
	if uint8(roll(1, int(uint8(images+1)))) <= 1 || spell == 0 || area {
		return list, false
	}
	result := append(EffectList(nil), list...)
	result[index].Payload[effectNodeLevelOffset]--
	if result[index].Payload[effectNodeLevelOffset] == 0 {
		result = result.RemoveAt(index)
	}
	return result, true
}

// MeleeDamageAfterAttackerEffects 是近戰傷害算完之後的群組 4（overlay-13 `021Eh`，記錄是攻擊者）。
// 只接 `1Dh`：entry 28 `0A4Ah` `A0 76 67 / 30 E4 / 99 / B9 04 00 / F7 F9 / 8B D0 / A0 76 67 /
// 30 E4 / 2B C2 / A2 76 67`——傷害減掉自己的四分之一（byte，整數除法）。
// 同組的 `03h`／`06h` 沒有接（spec 112〈OPEN〉）。
func MeleeDamageAfterAttackerEffects(list EffectList, damage int) int {
	if list.Has(EnfeeblementEffectCode) {
		value := uint8(damage)
		value -= value / 4
		return int(value)
	}
	return damage
}
