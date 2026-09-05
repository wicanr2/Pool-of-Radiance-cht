package gamepack

// 鎖住的門（spec 122）。入口是 overlay-14 `0DB7h`：撞上一道門時原版出一個
// `Bash` ／`Pick`／`Knock`／`Exit` 的選單，三個選項各有自己的閘門與判定。
//
// 這一支只放**規則**：擲什麼、擲到多少算成功、誰有資格出現在選單上。
// 真正把 GEO 的門旗標改掉是共用 engine 的 `UnlockDoorWrapped`
//（原版 overlay-14 `0112h`，兩邊各寫一次）。

// 門的三種狀態，就是 GEO 牆位元組的 detail 值（`0131h:0039h` 的回傳）。
const (
	// DoorUnlocked：沒鎖，直接走得過去。
	DoorUnlocked uint8 = 1
	// DoorLocked：鎖住。力量開門走 AD&D 的 Open Doors 欄。
	DoorLocked uint8 = 2
	// DoorBarred：閂住／魔法定住。力量開門走那張表**括號裡**的數字，
	// 一般人力氣根本不夠。
	DoorBarred uint8 = 3
)

// ThiefClassSlotIndex 是 `0247h(6)` 的那個 6：職業等級陣列（記錄 `+96h`）
// 的第 6 格就是 `+9Ch` ＝ 賊等級。`Pick` 只在隊上有賊時出現。
//
// 它與 ClassSlotThief 是同一個數字，這裡取別名而不是重寫一份常數。
const ThiefClassSlotIndex = ClassSlotThief

// KnockSpellID 是敲門術的法術編號（`0669h` 推的 `1Fh`）。
const KnockSpellID uint8 = 0x1F

// ForceDoorRoll 回傳「這個人撞這扇門要擲什麼、擲到多少（含）算成功」。
// possible 為 false 代表**這個人怎麼擲都開不了**——原版在那兩條路上還會
// 順手把 `DS:6CD2h` 清成 0，也就是 `Bash` 下次不再出現在選單上。
//
// 兩張表逐格對應 AD&D 一版力量表的 Open Doors 欄：狀態 2 用正常的數字，
// 狀態 3 用括號裡的。**19 以上碰到狀態 2 的門在原版沒有分支**（`03C9h` 的
// 最後一個比較是 `cmp ax, 12h`），照碼回 possible = false。
func ForceDoorRoll(state, strength, percentile uint8) (sides, target int, possible bool) {
	switch state {
	case DoorLocked:
		switch {
		case strength >= 3 && strength <= 7:
			return 6, 1, true
		case strength >= 8 && strength <= 15:
			return 6, 2, true
		case strength >= 16 && strength <= 17:
			return 6, 3, true
		case strength == 18 && percentile <= 50:
			return 6, 3, true
		case strength == 18 && percentile >= 51 && percentile <= 99:
			return 6, 4, true
		case strength == 18 && percentile == 100:
			return 6, 5, true
		}
	case DoorBarred:
		switch {
		case strength == 18 && percentile >= 91 && percentile <= 99:
			return 6, 1, true
		case strength == 18 && percentile == 100:
			return 6, 2, true
		case strength == 19 || strength == 20:
			return 6, 3, true
		case strength == 21 || strength == 22:
			return 6, 4, true
		case strength == 23:
			return 6, 5, true
		case strength == 24:
			// 這一格原版換成 1d8——1d6 表達不了 7。
			return 8, 7, true
		case strength == 25:
			// 一定成功，連骰都不擲（`03BBh` 直接把旗標立起來）。
			return 0, 0, true
		}
	}
	return 0, 0, false
}

// DoorForcer 是撞門要看的兩個欄位：力量（記錄 `+10h`）與特殊力量百分位
//（`+16h`）。
type DoorForcer struct {
	Strength   uint8
	Percentile uint8
}

// BashDoor 重現 `02A1h`：沿隊伍串列每個人各試一次，**任何一個成功就開**
//（原版一立起旗標迴圈就停）。
//
// keepOption 對應 `DS:6CD2h`：只要碰到一個力量不在表內的人，原版就把它清成 0，
// `Bash` 下次不再出現。誰把它設回 1 還沒讀到（spec 122 的「還沒讀」），
// 所以這裡照碼回報，由呼叫端決定保留多久。
func BashDoor(state uint8, party []DoorForcer, roll func(count, sides int) int) (opened, keepOption bool) {
	keepOption = true
	for _, member := range party {
		sides, target, possible := ForceDoorRoll(state, member.Strength, member.Percentile)
		if !possible {
			keepOption = false
			continue
		}
		if sides == 0 || roll(1, sides) <= target {
			return true, keepOption
		}
	}
	return false, keepOption
}

// PickLockOpens 重現 `0509h` 的判定：`Roll(1, 100)` 不大於開鎖百分比
//（記錄 `+78h`，spec 095）就開了。
//
// **一扇門只能試一次**：`056Bh` 在迴圈外無條件把 `DS:6CD3h` 清成 0，
// 所以不論成敗 `Pick` 都會從選單上消失。那一段由呼叫端做。
func PickLockOpens(roll int, skill uint8) bool { return roll <= int(skill) }
