package gamepack

// 群組 5 其餘的碼（spec 155，issue #113）：近戰（含射擊）傷害骰算完之後、扣血之前，overlay-13
// `022Ch` 對**目標**派發。順序是 `1Ch 29h 68h 78h 65h 73h 74h 77h 7Bh 60h 5Eh 3Ch 7Ah 75h`；`1Ch`
// 在 MirrorImageAbsorbs、`29h` 在 NormalMissileAvoided，這一支接其後的十二個。輸入：overlay-12
// （`d1b05743…`），`objdump -D -b binary -m i8086 -M intel`（`coab-go-test:20260729`），皆 exact。
//
// 處理常式看的「行動者」一律是 `DS:5CF0h`（輪到行動的人），不是傳進來的記錄：
//
//	1028h  f(記錄)：overlay-25 entry 45（彈藥查詢，spec 151）回非 0 而且指標非 NULL → 那一件，
//	       否則記錄 `+0CCh`（手上的武器）。這是「打中的那一件」。
//	0FB7h  f(記錄, 百分比)：5CF0h 的 `+0CCh` 是 NULL → 不管；overlay-25 entry 33（距離）<= 1 → 不管；
//	       Roll(1, 100) > 百分比 → 不管；否則印 "Avoids it"、`6776h = 0`、`6780h = FFh`、`6781h` 減一。

// CreatureBaneDamageFlags 是群組 4 的 `06h`（entry 9 `01C9h`）在 `0227h` `C6 06 77 67 09` 寫進
// `DS:6777h` 的值：之後的群組 5（`7Ah`）與倒下時的群組 13（`64h`）讀它。
const CreatureBaneDamageFlags = DamageFlagFire | DamageFlagMagic

const (
	// ThriKreenDodgeEffectCode 是 `68h`（THRI-KREEN）：entry 97 `28FAh`，`0FB7h(記錄, 3Ch)`。
	ThriKreenDodgeEffectCode uint8 = 0x68
	// GiantCatchEffectCode 是 `78h`（HILL／FIRE GIANT）：entry 114 `2BE1h`。
	GiantCatchEffectCode uint8 = 0x78
	// EdgedHalfEffectCode 是 `73h`（SKELETON）：entry 109 `2A95h`。型別表 `+7` 是 0（長劍 24h、雙手劍 26h）
	// 或位元 0 立著（01h 那一組）就減半、80h 那一組（0Ch、17h……巨人的石頭 57h／58h）不減——名字照 AD&D
	// 「骷髏怕鈍器」取（strong inference），規則本身照位元組。
	EdgedHalfEffectCode uint8 = 0x73
	// MagicWeaponHalfEffectCode 是 `74h`（MUMMY）：entry 110 `2AF8h`。
	MagicWeaponHalfEffectCode uint8 = 0x74
	// MagicWeaponOnlyEffectCode 是 `77h`（SPECTRE、VAMPIRE、MUMMY、JUJU ZOMBIE…）：entry 113 `2B96h`。
	MagicWeaponOnlyEffectCode uint8 = 0x77
	// SilverHalfEffectCode 是 `7Bh`（WRAITH）：entry 117 `2D99h`。
	SilverHalfEffectCode uint8 = 0x7b
	// SilverOrMagicEffectCode 是 `60h`（WIGHT）：entry 89 `2634h`。
	SilverOrMagicEffectCode uint8 = 0x60
	// BluntAndPiercingHalfEffectCode 是 `5Eh`（JUJU ZOMBIE）：entry 87 `25A1h`，型別表 `+7` 與 81h 有交集
	// （01h、80h 兩組）就減半；名字是 strong inference，規則照位元組。
	BluntAndPiercingHalfEffectCode uint8 = 0x5e
	// HolyWaterEffectCode 是 `75h`（不死生物）：entry 111 `2B36h`。
	HolyWaterEffectCode uint8 = 0x75

	// thriKreenDodgePercent 是 `2903h` `B0 3C`。
	thriKreenDodgePercent = 0x3c
	// giantCatchPercent 是 `2C1Ah` `B0 32`。
	giantCatchPercent = 0x32
	// silverNamePart 是物品記錄 `+31h` 的 `B1h`（字詞表的 `Silver`，treasure.itemNameTakesPlural）。
	silverNamePart = 0xb1
	// itemSilverOffset 是 `2661h` `26 80 7D 31 B1` 比的那一格。
	itemSilverOffset = 0x31
	// itemTypeDamageClassOffset 是型別表 `+7`（`2AC4h` `80 BD E7 54 00`：`DS:54E0h + 7 + 型別 × 10h`）。
	itemTypeDamageClassOffset = 7
	// holyWaterItemType 是 `2B58h` `26 80 7D 2E 55`；`2B5Fh..2B6Dh` 傷害 = Roll(1, 6) + 1（entry 9）。
	holyWaterItemType = 0x55
	// giantCatchItemTypes 是 `2C03h` `2E 57` 與 `2C0Dh` `2E 58`。
	giantCatchItemTypeA, giantCatchItemTypeB = 0x57, 0x58
	// magicWeaponOnlyHitDice 是 `2BCFh` `26 80 7D 73 04 / 7D 05`：行動者 `+2Eh`（種族，`2BC4h`
	// `26 80 7D 2E 00 / 7F 0B`）不大於 0 時，`+73h` 有號不小於 4 才打得動。
	magicWeaponOnlyHitDice = 4
)

// MeleeTargetEffects 是群組 5 那十二個碼讀的東西。
type MeleeTargetEffects struct {
	// Effects 是目標身上的串列。
	Effects EffectList
	// DamageFlags 是 `DS:6777h`：`0210h` 寫 0，群組 4 的 `06h` 改成 9。
	DamageFlags uint8
	// Dice 是 `DS:677Ah`（最近一次 entry 9 擲的骰數，`7Ah` 讀）。
	Dice uint8
	// Roll 是 overlay-24 entry 8／9。
	Roll func(count, sides int) int
	// Hitting 是 `1028h(DS:5CF0h)`：打中的那一件的記錄，NULL 就是 nil。
	Hitting []byte
	// Types 是物品型別表（`DS:54E0h`）；`73h`／`5Eh` 讀 `+7`。
	Types *ItemTypeTable
	// Wielded 是 `DS:5CF0h` 的 `+0CCh`（手上的武器，不看彈藥）；`0FB7h`、`75h`、`78h`、`7Ah` 讀它。
	Wielded []byte
	// ActorRace 是 `DS:5CF0h` 的 `+2Eh`；ActorHitDice 是 `+73h`。
	ActorRace, ActorHitDice uint8
	// Distance 是 overlay-25 entry 33（目標, `DS:5CF0h`）。
	Distance int
}

// MeleeTargetOutcome 是群組 5 那十二個碼跑完的結果。
type MeleeTargetOutcome struct {
	Damage int
	// Effects 是目標改過的串列（`65h` 掛 `3Bh`）。
	Effects EffectList
	// Dice 是跑完之後的 `DS:677Ah`（`75h` 擲 1d6 改成 1、`7Ah` 擲 3d8 改成 3）。
	Dice uint8
	// Avoided 為真時 `0FB7h` 擋掉了這一下：印 "Avoids it"、這一下不算命中。
	Avoided bool
}

// meleeTargetGroup 是群組 5 在 `1Ch 29h` 之後的順序（spec 112 的二十組表）。`3Ch` 是空常式 entry 126。
var meleeTargetGroup = [...]uint8{0x68, 0x78, 0x65, 0x73, 0x74, 0x77, 0x7b, 0x60, 0x5e, 0x3c, 0x7a, 0x75}

// Apply 照群組 5 的順序跑一遍。傷害格 `DS:6776h` 是 byte。
func (input MeleeTargetEffects) Apply(value int) MeleeTargetOutcome {
	outcome := MeleeTargetOutcome{Damage: value, Effects: input.Effects, Dice: input.Dice}
	list := input.Effects
	plus := func(item []byte) int8 { return int8(item[ItemPlusOffset]) }
	for _, code := range meleeTargetGroup {
		if !list.Has(code) {
			continue
		}
		switch code {
		case ThriKreenDodgeEffectCode:
			// `28FAh`：`0FB7h(記錄, 3Ch)`。
			if input.missileAvoided(thriKreenDodgePercent) {
				outcome.Damage, outcome.Avoided = 0, true
			}
		case GiantCatchEffectCode:
			// `2BE1h`：`+0CCh` 那一件的型別是 57h 或 58h → `0FB7h(記錄, 32h)`。
			if item := input.Wielded; len(item) > ItemTypeOffset &&
				(item[ItemTypeOffset] == giantCatchItemTypeA || item[ItemTypeOffset] == giantCatchItemTypeB) &&
				input.missileAvoided(giantCatchPercent) {
				outcome.Damage, outcome.Avoided = 0, true
			}
		case TrollWoundEffectCode:
			outcome.Effects = TrollWounded(outcome.Effects)
		case EdgedHalfEffectCode:
			// `2A95h`：打中的那一件存在，而型別表 `+7` 是 0 或位元 0 立著 → 減半。
			if class, ok := input.hittingDamageClass(); ok && (class == 0 || class&0x01 != 0) {
				outcome.Damage = halveDamage(outcome.Damage)
			}
		case MagicWeaponHalfEffectCode:
			// `2AF8h`：打中的那一件 `+32h` 有號大於 0（`26 80 7D 32 00 / 7E 0E`）→ 減半。
			if item := input.Hitting; len(item) > ItemPlusOffset && plus(item) > 0 {
				outcome.Damage = halveDamage(outcome.Damage)
			}
		case MagicWeaponOnlyEffectCode:
			// `2B96h`：打中的那一件是 NULL 或 `+32h` 是 0 → 行動者 `+2Eh` 有號大於 0（有種族的人）
			// 歸零；否則 `+73h` 有號小於 4 歸零。
			if item := input.Hitting; len(item) <= ItemPlusOffset || item[ItemPlusOffset] == 0 {
				if int8(input.ActorRace) > 0 || int8(input.ActorHitDice) < magicWeaponOnlyHitDice {
					outcome.Damage = 0
				}
			}
		case SilverHalfEffectCode:
			// `2D99h`：NULL → 歸零；`+32h` 是 0 而 `+31h` 是 B1h → 減半；`+32h` 是 0 → 歸零。
			item := input.Hitting
			switch {
			case len(item) <= ItemPlusOffset:
				outcome.Damage = 0
			case item[ItemPlusOffset] == 0 && item[itemSilverOffset] == silverNamePart:
				outcome.Damage = halveDamage(outcome.Damage)
			case item[ItemPlusOffset] == 0:
				outcome.Damage = 0
			}
		case SilverOrMagicEffectCode:
			// `2634h`：NULL → 歸零；`+32h` 是 0 而且 `+31h` 不是 B1h → 歸零。
			item := input.Hitting
			if len(item) <= ItemPlusOffset ||
				item[ItemPlusOffset] == 0 && item[itemSilverOffset] != silverNamePart {
				outcome.Damage = 0
			}
		case BluntAndPiercingHalfEffectCode:
			// `25A1h`：打中的那一件存在，型別表 `+7` 與 81h 有交集 → 減半。
			if class, ok := input.hittingDamageClass(); ok && class&0x81 != 0 {
				outcome.Damage = halveDamage(outcome.Damage)
			}
		case FireVulnerabilityEffectCode:
			// entry 116 `2D4Ch`，與群組 6 同一支（SpellDamageEffects）：`+0CCh` 是 56h → 3d8、骰數 3；
			// `6777h & 9` → 加骰數。
			if item := input.Wielded; len(item) > ItemTypeOffset && item[ItemTypeOffset] == oilFlaskItemType &&
				input.Roll != nil {
				outcome.Dice = 3
				outcome.Damage = int(uint8(input.Roll(3, 8)))
			}
			if input.DamageFlags&(DamageFlagFire|DamageFlagMagic) != 0 {
				outcome.Damage = int(uint8(outcome.Damage) + outcome.Dice)
			}
		case HolyWaterEffectCode:
			// `2B36h`：`+0CCh` 是 55h → 傷害 = Roll(1, 6) + 1（entry 9，骰數寫成 1）。
			if item := input.Wielded; len(item) > ItemTypeOffset && item[ItemTypeOffset] == holyWaterItemType &&
				input.Roll != nil {
				outcome.Dice = 1
				outcome.Damage = int(uint8(input.Roll(1, 6)) + 1)
			}
		}
	}
	return outcome
}

// missileAvoided 是 `0FB7h`：行動者手上沒有武器、或距離不到兩格，連骰都不擲。
func (input MeleeTargetEffects) missileAvoided(percent int) bool {
	if input.Wielded == nil || input.Distance <= 1 || input.Roll == nil {
		return false
	}
	return int(uint8(input.Roll(1, 100))) <= percent
}

// hittingDamageClass 是打中的那一件在型別表的 `+7`；沒有那一件（或型別表不在）回 false。
func (input MeleeTargetEffects) hittingDamageClass() (uint8, bool) {
	item := input.Hitting
	if len(item) <= ItemTypeOffset || input.Types == nil {
		return 0, false
	}
	entry, err := input.Types.Entry(item[ItemTypeOffset])
	if err != nil {
		return 0, false
	}
	return entry.Raw[itemTypeDamageClassOffset], true
}
