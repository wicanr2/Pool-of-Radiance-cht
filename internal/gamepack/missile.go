package gamepack

// 射擊、彈藥與 AI 換武器（spec 151，issue #98）。
//
// 這一層只讀寫物品記錄的位元組與型別表；誰在打誰、距離與盤面在 cmd/pool-game。
// 位址都是 overlay-local offset，overlay 與 SHA-256 見 spec 151。

const (
	// ItemTypeRateOfFireOffset 是型別表 `+05h`：射擊武器一回合的次數編碼
	// （overlay-13 `0D89h` 讀 `DS:54E5h`，小於 2 墊成 2）。
	ItemTypeRateOfFireOffset = 0x05
	// ItemTypeReachOffset 是型別表 `+0Ch`（射程加一，spec 065）。
	ItemTypeReachOffset = 0x0c

	// 型別表 `+0Eh` 在 overlay-25 entry 45（`2EB1h`）用到的另外兩個位元。
	itemTypeFlagMissile   = 0x08 // `2F14h`：射出去的武器，才去看箭與弩矢
	itemTypeFlagThrowSelf = 0x10 // `2EFCh`：射出去的就是武器自己
	// itemTypeThrowMeleeMask 是 entry 44（`2E98h`）的 `and al, 14h / cmp al, 14h`：
	// 丟得出去（bit 4）又能近戰（bit 2）。
	itemTypeThrowMeleeMask = 0x14
	// itemTypeNoAmmunitionFlags 是 `2F60h` 的 `cmp [bp-6], 0Ah`：旗標恰好是 0Ah 的武器
	// 不需要彈藥也射得出去（型別 2Fh、7Fh）。
	itemTypeNoAmmunitionFlags = 0x0a

	// thrownItemVanishesEffect 是 overlay-13 `1A13h` 的 `cmp es:[di+3Eh], 89h`：
	// 丟完的武器這個值不進戰利品堆，直接放掉（overlay-25 entry 17）。
	thrownItemVanishesEffect = 0x89
	// minimumRateOfFire 是 `0D90h` 的 `cmp [bp-4], 2 / jae`。
	minimumRateOfFire = 2

	// ClassUseMaskOffset 是記錄 `+0B0h`（ClassUseMask 寫的那一格；overlay-09 `1502h` 讀它）。
	ClassUseMaskOffset = 0xb0

	// ProtectionFromNormalMissilesEffectCode 是群組 5 的 `29h`（overlay-12 entry 40 `108Dh`）。
	ProtectionFromNormalMissilesEffectCode uint8 = 0x29
	// normalMissileAvoidChance 是 `10BDh` 的 `mov al, 64h`：`0FEAh` Roll(1, 100) <= 100，一定成立。
	normalMissileAvoidChance = 100
)

// MissileGear 是一個人身上跟射擊有關的四件事，全部照 overlay-25 讀武器槽 `+0CCh`
// 與 `+0F8h`／`+0FCh`（穿戴中的物品依 entry 7 的規則認回，readiedSlots）：
//
//	Ranged      entry 43 `2E29h`：有武器而型別表 `+0Ch` 大於 1
//	ThrownMelee entry 44 `2E6Ch`：Ranged，而且旗標 `& 14h == 14h`
//	Ammunition  entry 45 `2EB1h` 寫回的那一件（-1 是 NULL）
//	CanFire     entry 45 的回傳：Ammunition 不是 NULL，或旗標恰好是 0Ah
//
// entry 45 的順序：旗標 bit 4 → 武器自己；bit 3 成立時 bit 0 → `+0F8h`（箭，**即使是
// NULL 也照寫**，蓋掉 bit 4 那一步）、bit 7 → `+0FCh`（弩矢）。
type MissileGear struct {
	Weapon      int
	Ammunition  int
	Ranged      bool
	ThrownMelee bool
	CanFire     bool
	// RateOfFire 是型別表 `+05h`（至少 2）；只在 Ranged 且 CanFire 時有意義（overlay-13 `0D5Bh..0D9Dh`）。
	RateOfFire uint8
}

// MissileGearOf 讀出 inventory（物品鏈，照順序）的 MissileGear。
func MissileGearOf(inventory [][]byte, types *ItemTypeTable) (MissileGear, error) {
	gear := MissileGear{Weapon: -1, Ammunition: -1}
	if types == nil {
		return gear, nil
	}
	slots, err := readiedSlots(inventory, types)
	if err != nil {
		return gear, err
	}
	gear.Weapon = slots.slot[ItemCategoryWeapon]
	if gear.Weapon < 0 {
		return gear, nil
	}
	entry, err := types.Entry(inventory[gear.Weapon][ItemTypeOffset])
	if err != nil {
		return gear, err
	}
	flags := entry.Flags()
	gear.Ranged = entry.Raw[ItemTypeReachOffset] > 1
	gear.ThrownMelee = gear.Ranged && flags&itemTypeThrowMeleeMask == itemTypeThrowMeleeMask
	if flags&itemTypeFlagThrowSelf != 0 {
		gear.Ammunition = gear.Weapon
	}
	if flags&itemTypeFlagMissile != 0 {
		if flags&ItemTypeFlagNeedsLauncher != 0 {
			gear.Ammunition = slots.arrow
		}
		if flags&ItemTypeFlagUsesAmmunition != 0 {
			gear.Ammunition = slots.quarrel
		}
	}
	gear.CanFire = gear.Ammunition >= 0 || flags == itemTypeNoAmmunitionFlags
	gear.RateOfFire = entry.Raw[ItemTypeRateOfFireOffset]
	if gear.RateOfFire < minimumRateOfFire {
		gear.RateOfFire = minimumRateOfFire
	}
	return gear, nil
}

// VolleyLimit 是 overlay-13 `0DD1h..0E06h`：射擊的次數不超過彈藥的數量，數量 0 或 1
// 都算一發（`0DD9h` 先放 1，`+39h` 大於它才換）。沒有彈藥那一件（旗標 0Ah）不設上限，回 0。
func (gear MissileGear) VolleyLimit(inventory [][]byte) int {
	if gear.Ammunition < 0 || gear.Ammunition >= len(inventory) {
		return 0
	}
	count := int(inventory[gear.Ammunition][ItemCountOffset])
	if count < 1 {
		count = 1
	}
	return count
}

// AmmunitionSpend 是 overlay-13 `19D5h..1A96h` 對那一件做了什麼。
type AmmunitionSpend struct {
	// Remove 是整件用完了：overlay-25 entry 17（`156Ah`）從物品鏈摘掉並放掉。
	Remove bool
	// Dropped 是丟出去落在地上的那一件（`1A1Ah..1A5Bh`：GetMem(3Fh)、整筆複製、
	// `+34h = 0`、插在戰利品串列 `DS:676Eh` 的頭）；nil 代表沒有。
	Dropped []byte
}

// SpendAmmunition 扣掉這一次實際射出的發數（`DS:6D20h`，第一形態每揮一下 `16BEh` 加一）：
//
//	19E3  +39h > 0 → +39h −= 發數（byte）
//	19F7  +39h == 0 → 用完：
//	1A07    丟得出去又能近戰（entry 44，攻擊者當下的武器）而且 +3Eh != 89h → 複製一件到地上
//	1A6C／1A8B  entry 17 把原件摘掉
//
// `+39h` 本來就是 0 的（單件的匕首）直接走「用完」。thrownMelee 是扣之前攻擊者的 entry 44。
func SpendAmmunition(item []byte, shots int, thrownMelee bool) AmmunitionSpend {
	if len(item) <= ItemEffectOffset {
		return AmmunitionSpend{}
	}
	if item[ItemCountOffset] > 0 {
		item[ItemCountOffset] -= uint8(shots)
	}
	if item[ItemCountOffset] != 0 {
		return AmmunitionSpend{}
	}
	spend := AmmunitionSpend{Remove: true}
	if thrownMelee && item[ItemEffectOffset] != thrownItemVanishesEffect {
		spend.Dropped = append([]byte(nil), item...)
		spend.Dropped[ItemReadiedOffset] = 0
	}
	return spend
}

// MissileOf 是 overlay-12 `1028h`：攻擊者射出去的那一件。沒有武器是 -1；entry 45 成立而且
// 寫回的不是 NULL 就是那一件，否則是武器本身。
func (gear MissileGear) MissileOf() int {
	if gear.Weapon < 0 {
		return -1
	}
	if gear.CanFire && gear.Ammunition >= 0 {
		return gear.Ammunition
	}
	return gear.Weapon
}

// NormalMissileAvoided 是 `29h` 防護普通飛彈（overlay-12 entry 40 `108Dh` 與 `0FB7h`），群組 5
// 在近戰傷害骰算完之後對**目標**派發（overlay-13 `0223h..022Ch`）：
//
//	109Ch  1028h(攻擊者) 是 NULL → 不管
//	10B0h  那一件 +32h != 0（有附魔）→ 不管
//	0FC1h  攻擊者 +0CCh 是 NULL → 不管
//	0FDBh  距離（overlay-25 entry 33 `2591h`）<= 1 → 不管
//	0FEAh  Roll(1, 100) > 64h → 不管（不會發生，但骰一定擲）
//	1004h  印 "Avoids it"；DS:6776h = 0、DS:6780h = FFh、DS:6781h 減一
//
// 回 true 代表這一下被擋掉：傷害歸零、不算命中。
func NormalMissileAvoided(targetEffects EffectList, gear MissileGear, inventory [][]byte,
	distance int, roll func(count, sides int) int) bool {
	if !targetEffects.Has(ProtectionFromNormalMissilesEffectCode) {
		return false
	}
	missile := gear.MissileOf()
	if missile < 0 || missile >= len(inventory) || len(inventory[missile]) <= ItemPlusOffset {
		return false
	}
	if inventory[missile][ItemPlusOffset] != 0 {
		return false
	}
	if gear.Weapon < 0 || distance <= 1 {
		return false
	}
	return roll != nil && roll(1, normalMissileAvoidChance) <= normalMissileAvoidChance
}

// AIGearInput 是 overlay-09 entry 9（`13D5h`）讀的東西。
type AIGearInput struct {
	// ClassMask 是記錄 `+0B0h`：型別表 `+0Dh` 與它沒有交集的不列入。
	ClassMask uint8
	// NaturalScore 是 `1475h..14AFh`：`+0A3h × +0A5h`（byte），`+0A7h` 大於 0 再加它的兩倍
	// ——近戰的分數要比這個高才換。
	NaturalScore uint8
	// Alignment 是記錄 `+0A0h`（`139Fh`）。
	Alignment uint8
	// TargetUndead 是目前目標（`+108h` 的 `+0Ah`）的 `+76h` 大於 0（`1334h`）。
	TargetUndead bool
	// AdjacentEnemies 是 overlay-25 entry 32（`246Dh`）以射程 1 找到的對面人數非 0（`1709h`）。
	AdjacentEnemies bool
}

// AIGearStep 是 entry 9 叫過的一次 Ready（overlay-19 entry 7 `14CAh`，`0C9h:0043h`）。
type AIGearStep struct {
	Index   int
	Outcome ReadyOutcome
}

// AIGearNaturalScore 算 AIGearInput.NaturalScore（`1475h..14AFh`）。
func AIGearNaturalScore(record []byte) uint8 {
	if len(record) <= 0xa7 {
		return 0
	}
	score := record[0xa3] * record[0xa5]
	if bonus := int8(record[0xa7]); bonus > 0 {
		score += uint8(int(bonus) * 2)
	}
	return score
}

// aiWeaponScore 是 overlay-09 `1297h`：一件武器在 AI 眼裡值幾分（byte）。型別表那一筆
// 整個搬到 `[bp-12h]`，所以 `[bp-9]`／`[bp-8]` 是 `+9`／`+0Ah`、`[bp-7]` 是 `+0Bh`、
// `[bp-0Dh]` 是 `+5`、`[bp-11h]` 是 `+1`、`[bp-4]` 是 `+0Eh`。
//
//	12C0  骰數 × 骰面
//	12D4  +32h 大於 0 → 加 +32h × 8
//	12F4  +0Bh 大於 0 → 加 +0Bh × 2
//	130F  型別 55h 而目標是不死生物 → 改成 8
//	133F  射擊武器（旗標 bit 3）→ 加 (+5 − 1) × 2
//	135C  手數 <= 1 → 加 3
//	136D  角色 +100h + 手數 > 3 → 0
//	1389  +3Eh == 84h 而 +3Dh & 0Fh 不是角色的陣營 → 0
//	13AA  +3Dh == 53h → 0
//	13B8  被詛咒（+36h）→ 0
func aiWeaponScore(item []byte, entry ItemTypeEntry, in AIGearInput, hands uint8) uint8 {
	score := entry.Raw[0x09] * entry.Raw[0x0a]
	if plus := int8(item[ItemPlusOffset]); plus > 0 {
		score += uint8(int(plus) << 3)
	}
	if bonus := int8(entry.Raw[0x0b]); bonus > 0 {
		score += uint8(int(bonus) * 2)
	}
	if item[ItemTypeOffset] == aiHolyWeaponType && in.TargetUndead {
		score = 8
	}
	if entry.Flags()&itemTypeFlagMissile != 0 {
		score += uint8((int(entry.Raw[ItemTypeRateOfFireOffset]) - 1) * 2)
	}
	itemHands := entry.Raw[ItemTypeHandsOffset]
	if itemHands <= 1 {
		score += 3
	}
	if int(hands)+int(itemHands) > aiHandBudget {
		score = 0
	}
	if item[ItemEffectOffset] == aiAlignedItemEffect && item[aiAlignmentItemOffset]&0x0f != in.Alignment {
		score = 0
	}
	if item[aiAlignmentItemOffset] == aiRefusedItemCode {
		score = 0
	}
	if item[ItemCursedOffset] != 0 {
		score = 0
	}
	return score
}

const (
	// aiHolyWeaponType 是 `130Fh` 的 `cmp es:[di+2Eh], 55h`。
	aiHolyWeaponType = 0x55
	// aiHandBudget 是 `1380h` 的 `cmp ax, 3`。
	aiHandBudget = 3
	// aiAlignedItemEffect 與 aiAlignmentItemOffset 是 `138Ch`／`1396h`；aiRefusedItemCode 是 `13ADh`。
	aiAlignedItemEffect   = 0x84
	aiAlignmentItemOffset = 0x3d
	aiRefusedItemCode     = 0x53
	// aiShieldCategory 是 `159Dh` 的 `cmp [di+54E0h], 1`（盾，`+0D0h`）。
	aiShieldCategory = 1
)

// ChooseAIGear 是 overlay-09 entry 9（`13D5h`）：電腦接手的人每一回合在接近之前重挑武器與盾
// （entry 1 `019Bh`），射擊武器貼身時也叫（entry 5 `0DFCh`）。直接改 inventory 的 `+34h`，
// 回傳叫過的每一次 Ready 與「有沒有換」（`[bp-1Dh]`）。
//
//	13DB..1458  +100h 扣掉目前武器（+0CCh）與盾（+0D0h）的手數
//	1471        射擊的最佳分數從 1 起、近戰從 NaturalScore 起、盾從 0 起
//	14C6..1603  沿物品鏈：類別 0 而職業合得上 → 打分；旗標 bit 3 或 bit 4 → 射擊候選，
//	            沒有 bit 3 → 近戰候選（都是「嚴格大於」才換）；類別 1 而職業合得上 →
//	            盾的分數是 +32h + 1（負的算 0）
//	1606..16D2  射擊候選的 entry 44 與 entry 45（照它自己的旗標、角色的 +0F8h／+0FCh）
//	16D6..1727  射擊分數 > 近戰分數 ÷ 2、有彈藥、而且（丟得出去又能近戰，或身邊沒有敵人）
//	            → 射擊候選；否則近戰候選（可能是 NULL）
//	172A..17FE  目前武器就是它或被詛咒 → 不換；否則 Ready(目前武器)、重算、+100h 再扣盾、
//	            Ready(挑中的)——**挑中的若已經穿著，這一下會把它卸掉**（14F3h 是切換）
//	1802..18F8  重算；+100h > 2：盾沒被詛咒就卸盾，否則 Ready(挑中的)；+100h < 2 而盾要換：
//	            卸掉目前的盾、重算、Ready(最好的盾)
func ChooseAIGear(inventory [][]byte, types *ItemTypeTable, in AIGearInput) ([]AIGearStep, bool, error) {
	if types == nil {
		return nil, false, nil
	}
	slots, err := readiedSlots(inventory, types)
	if err != nil {
		return nil, false, err
	}
	handsOf := func(index int) (uint8, error) {
		entry, err := types.Entry(inventory[index][ItemTypeOffset])
		if err != nil {
			return 0, err
		}
		return entry.Raw[ItemTypeHandsOffset], nil
	}
	hands := uint8(slots.hands)
	weapon, shield := slots.slot[ItemCategoryWeapon], slots.slot[aiShieldCategory]
	for _, index := range []int{weapon, shield} {
		if index < 0 {
			continue
		}
		used, err := handsOf(index)
		if err != nil {
			return nil, false, err
		}
		hands -= used
	}
	ranged, melee, bestShield := -1, -1, -1
	rangedScore, meleeScore, shieldScore := uint8(1), in.NaturalScore, uint8(0)
	for index, item := range inventory {
		if len(item) <= ItemEffectOffset {
			continue
		}
		entry, err := types.Entry(item[ItemTypeOffset])
		if err != nil {
			return nil, false, err
		}
		usable := entry.Raw[ItemTypeClassMaskOffset]&in.ClassMask != 0
		switch entry.Category() {
		case ItemCategoryWeapon:
			if !usable {
				continue
			}
			score := aiWeaponScore(item, entry, in, hands)
			flags := entry.Flags()
			if flags&(itemTypeFlagMissile|itemTypeFlagThrowSelf) != 0 && score > rangedScore {
				ranged, rangedScore = index, score
			}
			if flags&itemTypeFlagMissile == 0 && score > meleeScore {
				melee, meleeScore = index, score
			}
		case aiShieldCategory:
			if !usable {
				continue
			}
			score := uint8(0)
			if plus := int8(item[ItemPlusOffset]); plus >= 0 {
				score = uint8(plus) + 1
			}
			if score > shieldScore {
				bestShield, shieldScore = index, score
			}
		}
	}
	chosen := melee
	if ranged >= 0 && int(rangedScore) > int(meleeScore)/2 {
		entry, err := types.Entry(inventory[ranged][ItemTypeOffset])
		if err != nil {
			return nil, false, err
		}
		flags := entry.Flags()
		thrownMelee := entry.Raw[ItemTypeReachOffset] > 1 && flags&itemTypeThrowMeleeMask == itemTypeThrowMeleeMask
		ammunition := -1
		if flags&itemTypeFlagThrowSelf != 0 {
			ammunition = ranged
		}
		if flags&itemTypeFlagMissile != 0 {
			if flags&ItemTypeFlagNeedsLauncher != 0 {
				ammunition = slots.arrow
			}
			if flags&ItemTypeFlagUsesAmmunition != 0 {
				ammunition = slots.quarrel
			}
		}
		canFire := ammunition >= 0 || flags == itemTypeNoAmmunitionFlags
		if canFire && (thrownMelee || !in.AdjacentEnemies) {
			chosen = ranged
		}
	}

	var steps []AIGearStep
	changed := false
	ready := func(index int, hands int) error {
		result, err := readyItemWithHands(inventory, index, types, in.ClassMask, hands)
		if err != nil {
			return err
		}
		steps = append(steps, AIGearStep{Index: index, Outcome: result.Outcome})
		return nil
	}
	cursed := func(index int) bool { return inventory[index][ItemCursedOffset] != 0 }

	if weapon < 0 || (weapon != chosen && !cursed(weapon)) {
		if weapon >= 0 {
			if err := ready(weapon, -1); err != nil {
				return nil, false, err
			}
		}
		if slots, err = readiedSlots(inventory, types); err != nil {
			return nil, false, err
		}
		hands = uint8(slots.hands)
		if shield := slots.slot[aiShieldCategory]; shield >= 0 && !cursed(shield) {
			used, err := handsOf(shield)
			if err != nil {
				return nil, false, err
			}
			hands -= used
		}
		if chosen >= 0 {
			if err := ready(chosen, int(hands)); err != nil {
				return nil, false, err
			}
		}
		changed = true
	}

	if slots, err = readiedSlots(inventory, types); err != nil {
		return nil, false, err
	}
	shield = slots.slot[aiShieldCategory]
	changeShield := shield < 0 || (shield != bestShield && !cursed(shield))
	switch {
	case slots.hands > readyHandLimit:
		if shield >= 0 && !cursed(shield) {
			if err := ready(shield, -1); err != nil {
				return nil, false, err
			}
		} else if chosen >= 0 {
			// `187Bh` 對挑中的那一件再叫一次；挑中的是 NULL 時原版傳 NULL 進去（不會走到）。
			if err := ready(chosen, -1); err != nil {
				return nil, false, err
			}
		}
		changed = true
	case slots.hands < readyHandLimit && changeShield:
		if shield >= 0 {
			if err := ready(shield, -1); err != nil {
				return nil, false, err
			}
		}
		if bestShield >= 0 {
			if err := ready(bestShield, -1); err != nil {
				return nil, false, err
			}
		}
		changed = true
	}
	return steps, changed, nil
}
