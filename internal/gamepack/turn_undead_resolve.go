package gamepack

// 轉變不死生物的兩段迴圈：挑目標（overlay-13 entry 13 `1352h`）與執行
//（entry 12 `116Ah`）。敵方 AI 與玩家的 `T)urn` 走的是同一組（spec 111）。

const (
	// TurnUndeadQuota 是 `116Ah` 的 `[bp-3]` 初值。它是一個下限：每處理一隻就
	// 減一，額度歸零時若配額還有剩、而且剛處理的那一隻門檻是負數，就補回一隻。
	TurnUndeadQuota = 6
	// TurnUndeadColumnLimit 是 entry 13 挑目標時的欄位上界（`[bp-4]` 初值 0Dh）。
	// 取最小值等於挑最容易轉變的那一隻。
	TurnUndeadColumnLimit = 0x0D
)

// TurnUndeadCandidate 是轉變迴圈看得到的一隻不死生物。
//
// 名單本身由 overlay-25 entry 32（`010Ah:00C0h`）以射程不限重建（spec 096），
// 這裡只留 entry 13 真正讀的三件事。
type TurnUndeadCandidate struct {
	// Column 是記錄 `+76h`。不是不死生物就是 0，entry 13 會跳過。
	Column int
	// Turned 是 runtime `+10h`：這一場已經被轉變過的不再被挑中。
	Turned bool
	// Removed 是被摧毀之後的狀態（記錄 `+10Ch = 8`、`+10Dh = 0`）。
	//
	// entry 13 自己沒有看這個欄位——`116Ah` 在清那兩個 byte 之前先呼叫
	// overlay-32 entry 20（`013Dh:0084h`）把目標從戰術地圖上收掉，所以下一輪
	// 重建名單時 `0912h` 的鄰近查詢已經看不到它。強推論。
	Removed bool
}

// SelectTurnUndeadTarget 重現 overlay-13 entry 13（`1352h`）：在名單裡挑
// `+76h` 最小的一隻，跳過已經被轉變過的與不是不死生物的。回傳名單索引。
//
// 上界初值是 TurnUndeadColumnLimit，比任何合法欄位都大，所以第一個合格的
// 候選一定進得來；之後只有更小的才換人（`13D1h` 是 `jge` 跳過，同分不換）。
func SelectTurnUndeadTarget(candidates []TurnUndeadCandidate) (int, bool) {
	best := TurnUndeadColumnLimit
	index := -1
	for position, candidate := range candidates {
		if candidate.Turned || candidate.Removed {
			continue
		}
		if candidate.Column <= 0 || candidate.Column >= best {
			continue
		}
		best = candidate.Column
		index = position
	}
	return index, index >= 0
}

// TurnUndeadEvent 是迴圈裡處理掉的一隻。
type TurnUndeadEvent struct {
	// Index 是這一隻在 candidates 裡的位置。
	Index int
	// Column 是牠的 `+76h`。
	Column int
	// Threshold 是查表得到的 signed byte，正負號就是轉變與摧毀的分界。
	Threshold int
	// Outcome 只會是 TurnTurns 或 TurnDestroys——判定失敗那一隻不進事件。
	Outcome TurnOutcome
}

// TurnUndeadResult 是一次 `T)urn` 的完整結果。
type TurnUndeadResult struct {
	// Roll 是整次判定只擲一次的 1d20（`11C1h`）。
	Roll int
	// Row 是牧師等級壓成的列號。
	Row int
	// Allowance 是起始的 1d12（`11B3h`），還沒被配額補過。
	Allowance int
	// Events 是依序被轉變或摧毀的目標。
	Events []TurnUndeadEvent
	// Stopped 為真表示是因為某一隻沒擲過門檻而收工（`[bp-6]`）；
	// 為假表示是挑不到目標或額度用完。
	Stopped bool
}

// Turned 回報這一次有沒有轉到任何一隻，對應 `[bp-5]`。
// 為假時原版會多印一句（`131Ah`）。
func (r TurnUndeadResult) Turned() bool { return len(r.Events) > 0 }

// ResolveTurnUndead 重現 overlay-13 entry 12（`116Ah`）的整個迴圈。
//
// candidates 會被就地改寫：轉變成功的立起 Turned，被摧毀的立起 Removed——
// 原版寫的是目標 runtime `+10h` 與記錄 `+10Ch`／`+10Dh`。
//
// 擲骰順序照原版：先 1d12 的額度，再 1d20 的點數。整次判定共用同一個點數，
// 所以一群不死生物是「同一擲對每一隻各查各的門檻」，不是逐隻重擲。
func ResolveTurnUndead(table TurnUndeadTable, clericLevel int,
	candidates []TurnUndeadCandidate, roller Roller) TurnUndeadResult {
	result := TurnUndeadResult{
		Allowance: roller.Roll(1, TurnUndeadCountDie),
		Roll:      roller.Roll(1, TurnUndeadDie),
		Row:       TurnUndeadRow(clericLevel),
	}

	quota := TurnUndeadQuota
	allowance := result.Allowance
	for {
		index, ok := SelectTurnUndeadTarget(candidates)
		// `11FEh` 每一輪都重挑一次，然後才看額度與收工旗標——所以判定失敗的
		// 那一輪會多挑一次目標再離開。順序照抄，結果一樣。
		if !ok || allowance <= 0 || result.Stopped {
			break
		}

		column := candidates[index].Column
		threshold := table.Threshold(clericLevel, column)
		need := threshold
		if need < 0 {
			need = -need
		}
		if result.Roll < need {
			// `1313h`：立起收工旗標再回到迴圈頭，下一輪在 `1223h` 離開。
			result.Stopped = true
			continue
		}

		event := TurnUndeadEvent{Index: index, Column: column, Threshold: threshold}
		if threshold > 0 {
			candidates[index].Turned = true
			event.Outcome = TurnTurns
		} else {
			candidates[index].Removed = true
			event.Outcome = TurnDestroys
		}
		result.Events = append(result.Events, event)

		if quota > 0 {
			quota--
		}
		allowance--
		// `12F7h`：額度剛好歸零、配額還有剩、而且這一隻的門檻是**負數**時補回
		// 一隻。門檻剛好是 0 的那一格雖然也算摧毀（`1280h` 走 `jle`），
		// 卻補不到（`1303h` 走 `jge`）——兩處的分界差一格，照碼接。
		if allowance == 0 && quota > 0 && threshold < 0 {
			allowance++
		}
	}
	return result
}
