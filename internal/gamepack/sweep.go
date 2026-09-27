package gamepack

// 橫掃（sweep，spec 154）：戰士一次行動砍倒身邊好幾個不到一個生命骰的對手。
//
// 上限寫在記錄 `+6Bh`，由 overlay-25 entry 7 的尾段 `1000h..102Eh` 算：
//
//	1003  cmp byte es:[di+98h], 0 ; jle 1026h     ; 戰士等級
//	100E  cmp byte es:[di+2Eh], 0 ; jle 1026h     ; 種族
//	1018  +6Bh = +98h
//	1029  +6Bh = 1
//
// 兩個比較都是有號的（`jle`）。每回合初始化 overlay-13 entry 1 在 `0076h` 把它抄進
// runtime `+5`；唯一的讀取端是 overlay-13 entry 10（`0E8Ch`）。
const (
	// SweepLimitOffset 是記錄 `+6Bh`。
	SweepLimitOffset = 0x6b
	// HandsOffset 是記錄 `+100h`：穿戴中物品佔的手數和，entry 7 每次重算都重寫（spec 144）。
	HandsOffset = 0x100
)

// SweepLimit 是 `1000h..102Eh`：戰士等級大於 0、種族碼大於 0 時是戰士等級，否則 1。
// 只看戰士那一格（`+98h`），聖騎士與遊俠的等級不算。
func SweepLimit(fighterLevel, race uint8) uint8 {
	if int8(fighterLevel) > 0 && int8(race) > 0 {
		return fighterLevel
	}
	return 1
}

// SweepInput 是 overlay-13 entry 10（`0E8Ch`）要讀的東西，全部是出手那一刻的值。
type SweepInput struct {
	// Remaining 是攻擊者這一相位第一形態還剩幾下（runtime `+113h`）。
	Remaining uint8
	// Limit 是攻擊者的 runtime `+5`（＝記錄 `+6Bh`）。
	Limit uint8
	// TargetHitDice 是目標的 `+73h`。
	TargetHitDice uint8
	// Distance 是 overlay-25 entry 33（`2591h`）量的攻擊者到目標的距離。
	Distance int
	// Nearby 是 overlay-25 entry 32（`246Dh`）以預算 1 列出的對面的人（`DS:6CD7h` 的順序）。
	Nearby []uint8
	// Target 是玩家或電腦挑的那一個。
	Target uint8
	// HitDice 回某個 combatant 的 `+73h`。
	HitDice func(uint8) uint8
}

// SweepTargets 是 `0E8Ch..107Eh` 的判定：不成立回 nil（照一般攻擊打），成立回依序要砍的人，
// 每人一下。
//
//	0E99  +113h >= runtime +5         → 不掃
//	0EAF  目標 +73h != 0              → 不掃
//	0EC8  entry 33(攻擊者, 目標) != 1 → 不掃
//	0EDD  entry 32(攻擊者, 1)：數名單上 +73h == 0 的人數 n
//	0F4D  n <= +113h                  → 不掃
//	0F62  n = min(n, runtime +5)
//	0FB3  目標若不在名單第一格，與第一格對調
//	0FEF  名單逐格：還有額度、+73h == 0 就砍一下（+113h = 1 之後進 entry 15），額度減一
func SweepTargets(in SweepInput) []uint8 {
	if in.Remaining >= in.Limit || in.TargetHitDice != 0 || in.Distance != 1 {
		return nil
	}
	count := 0
	position := -1
	for index, who := range in.Nearby {
		if who == in.Target {
			position = index
		}
		if in.HitDice(who) == 0 {
			count++
		}
	}
	if count <= int(in.Remaining) {
		return nil
	}
	if count > int(in.Limit) {
		count = int(in.Limit)
	}
	order := append([]uint8(nil), in.Nearby...)
	// `0FB3h..0FD7h`：[bp-5] 只在找到目標時寫過；距離 1 的對手一定在名單上。
	if position > 0 {
		order[position] = order[0]
		order[0] = in.Target
	}
	var targets []uint8
	for _, who := range order {
		if count == 0 {
			break
		}
		if in.HitDice(who) != 0 {
			continue
		}
		targets = append(targets, who)
		count--
	}
	return targets
}
