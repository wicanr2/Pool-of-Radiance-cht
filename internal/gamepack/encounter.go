package gamepack

import "fmt"

// `29h ENCOUNTER MENU` 的規則（spec 078）。原版把玩家的選擇先經過一張五格
// 類型表（運算元 5..9，用選擇當索引），再依「類型 × 選擇」決定要往 ECL
// 變數寫哪個結果碼，或是把怪物拉近一格再問一次。
//
// remake 只負責把結果碼寫對；`COMPARE`／`IF` 的分支由 ECL 自己做。
const (
	// EncounterChoiceCombat 等四個是選單的順序。
	EncounterChoiceCombat = 0
	// EncounterChoiceWait 是等待。
	EncounterChoiceWait = 1
	// EncounterChoiceFlee 是逃跑。
	EncounterChoiceFlee = 2
	// EncounterChoiceAdvance 是第四項是 ADVANCE 時的編號——**選單上的第 3 項
	// 就是表的第 3 格**，不改寫。
	EncounterChoiceAdvance = 3
	// EncounterChoiceParley 是第四項是 PARLAY 時的編號。原版在 `23B2h` 把
	// 選單索引 3 改寫成 **4**，而改寫的條件（`+582h == 0` 或 `+1CCh == 0`）
	// 與「第四項顯示 PARLAY」的條件是同一個。所以 PARLAY 指到表的第 4 格。
	EncounterChoiceParley = 4

	// EncounterMessageWait 是雙方按兵不動。
	EncounterMessageWait = "Both sides wait."
	// EncounterMessageFlee 是怪物跑掉。
	EncounterMessageFlee = "The monsters flee."
)

// EncounterOutcome 是一次選擇的結果。
type EncounterOutcome struct {
	// Store 為真時要把 ResultCode 寫進運算元 4 指的 ECL 變數。
	Store bool
	// ResultCode 是要寫的值。
	ResultCode uint16
	// Message 是要先印出來的字串；空字串表示不印。
	Message string
	// Approach 為真時怪物往前一格（距離減一），然後回到選單再問一次。
	Approach bool
	// Repeat 為真時不結束這條 opcode，回到選單。
	Repeat bool
}

// EncounterInputs 是判定要用到的數值。
type EncounterInputs struct {
	// Kind 是五格類型表在這個選擇上的值。
	Kind uint8
	// Choice 是玩家選了第幾項（0..4，4 是 ADVANCE）。
	Choice int
	// Distance 是怪物與隊伍的距離（記錄 `+582h`）。
	Distance int
	// SlowestMovement／FastestMovement 是隊伍的最慢與最快移動力（spec 079）。
	SlowestMovement int
	FastestMovement int
	// FleeThreshold 是運算元 13，AdvanceThreshold 是運算元 14。
	FleeThreshold    int
	AdvanceThreshold int
}

// ResolveEncounterChoice 依 spec 078 的表算出結果。讀不到證據的組合回錯誤，
// 不猜——猜錯的症狀是「遭遇被靜靜跳過」，在測試報表上看不出來。
func ResolveEncounterChoice(in EncounterInputs) (EncounterOutcome, error) {
	approach := func() EncounterOutcome {
		if in.Distance > 0 {
			return EncounterOutcome{Approach: true, Repeat: true}
		}
		return EncounterOutcome{Message: EncounterMessageWait, Repeat: true}
	}
	// 距離大於零就拉近一格再問；距離為零時 PARLAY 這一支才真的成立。
	parley := func() EncounterOutcome {
		if in.Distance > 0 {
			return EncounterOutcome{Approach: true, Repeat: true}
		}
		return EncounterOutcome{Store: true, ResultCode: 3}
	}
	switch in.Kind {
	case 0:
		if in.Choice == EncounterChoiceFlee {
			// 隊伍最慢的人跑得夠快才逃得掉。
			if in.SlowestMovement >= in.FleeThreshold {
				return EncounterOutcome{Store: true, ResultCode: 2}, nil
			}
			return EncounterOutcome{Store: true, ResultCode: 1}, nil
		}
		return EncounterOutcome{Store: true, ResultCode: 1}, nil
	case 1:
		// `241Bh`：戰鬥直接開打、等待只印字、逃跑存 2、ADVANCE 拉近、
		// PARLAY 在距離為零時存 3。
		switch in.Choice {
		case EncounterChoiceCombat:
			return EncounterOutcome{Store: true, ResultCode: 1}, nil
		case EncounterChoiceWait:
			return EncounterOutcome{Message: EncounterMessageWait, Repeat: true}, nil
		case EncounterChoiceFlee:
			return EncounterOutcome{Store: true, ResultCode: 2}, nil
		case EncounterChoiceAdvance:
			return approach(), nil
		case EncounterChoiceParley:
			return parley(), nil
		}
	case 2:
		if in.Choice == EncounterChoiceCombat && in.AdvanceThreshold <= in.FastestMovement {
			return EncounterOutcome{Store: true, ResultCode: 1}, nil
		}
		return EncounterOutcome{Store: true, ResultCode: 0, Message: EncounterMessageFlee}, nil
	case 3:
		// `25EDh`：等待與 ADVANCE 走同一支（`260Ch` 同時比 1 與 3），
		// PARLAY 另有一支（`2686h`）。
		switch in.Choice {
		case EncounterChoiceCombat:
			return EncounterOutcome{Store: true, ResultCode: 1}, nil
		case EncounterChoiceWait, EncounterChoiceAdvance:
			return approach(), nil
		case EncounterChoiceFlee:
			return EncounterOutcome{Store: true, ResultCode: 2}, nil
		case EncounterChoiceParley:
			return parley(), nil
		}
	case 4:
		// `26DBh`：等待、ADVANCE 與 PARLAY 三個走同一支（`26F4h` 同時比
		// 1、3、4），距離為零就存 3。
		switch in.Choice {
		case EncounterChoiceCombat:
			return EncounterOutcome{Store: true, ResultCode: 1}, nil
		case EncounterChoiceFlee:
			return EncounterOutcome{Store: true, ResultCode: 2}, nil
		case EncounterChoiceWait, EncounterChoiceAdvance, EncounterChoiceParley:
			return parley(), nil
		}
	}
	return EncounterOutcome{}, fmt.Errorf("Pool encounter kind %d with choice %d is not reverse engineered", in.Kind, in.Choice)
}

// EncounterMenuOptions 是要顯示的四個選項。第四項在沒有地圖上的怪物群時是
// PARLAY（overlay-03 `2333h`：距離為零或地圖情境旗標為零就用 PARLAY）。
func EncounterMenuOptions(distance int, hasMonsterGroup bool) []string {
	if distance == 0 || !hasMonsterGroup {
		return append([]string(nil), EncounterMenuChoices...)
	}
	return append([]string(nil), EncounterMenuAdvanceChoices...)
}

// EncounterChoiceIndex 把畫面上的第幾項換成類型表的索引。ADVANCE 那一版的
// 第四項在原版被改寫成 4（`23B2h`），所以兩版指到表裡不同格。
func EncounterChoiceIndex(selected int, options []string) int {
	if selected != EncounterChoiceAdvance || len(options) <= EncounterChoiceAdvance {
		return selected
	}
	if options[EncounterChoiceAdvance] == EncounterMenuAdvanceChoices[EncounterChoiceAdvance] {
		return EncounterChoiceAdvance
	}
	return EncounterChoiceParley
}
