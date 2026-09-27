package gamepack

// 吐息：效果碼 `58h`（spec 098〈吐息〉，issue #82）。
//
// 派發表 `DS:6786h` 的 `58h` 那一格不是 overlay-12 填的，是 overlay-22 自己的處理常式
// `3092h`（`retf 0Ah`：記錄 `[bp+0Ch]`、節點 `[bp+08h]`）。它在群組 0Eh（14），而群組 0Eh
// 只有 overlay-09 entry 5 開場（`0B66h`）一個呼叫端——怪物接近之前問一次。全遊戲只有
// MON5 block 66 的 TYRANITHRAXUS 帶著它（`58 00 00 FF 00`）。
//
//	3098  80 3E D7 6C 00 / 74 12      相位 6CD7h 不是 0：Roll(1, 100) <= 50 就不吐
//	30B1  自己的 X、Y（overlay-32 entry 17／18）
//	30CD  entry 20(記錄, "Breathes!", 0Ah, 1)                ; 3088h
//	30ED  DS:6A78h(33h, 1, &6E8Ch)                          ; 挑目標：AI 放閃電束那一支
//	30FC  6CADh = X + sign(6CADh − X)，等於 X + 1 時再加一    ; overlay-31 entry 1 是 sign
//	3119  6CAEh 同樣                                         ; 起點是 2×2 身體外面那一格
//	315C  0100h:005Ch(記錄)                                  ; overlay-24 entry 12 = 0FCCh，摘 19h
//	31A3  287Ch(6CADh, 6CAEh, 記錄 +32h, 3, &擋住)            ; 先打起點那一格
//	31B8  2919h(0Ah, 記錄 +32h, 3, 0)                         ; 再拉射線，長度 10、不加價
//	31BE  節點 +3 > FDh → 減一；否則 0100h:002Ah 摘掉這個節點  ; FFh 起算，吐三次
//	31E8  010Ah:00CAh（entry 34）結束行動
//
// 傷害是記錄 `+32h`（生命上限，MonsterRecord.MaxHitPoints），豁免類別 3（吐息），處置規則
// 寫死減半（`287Ch` 的 `28EBh`）。

const (
	// BreathEffectCode 是 `58h`。
	BreathEffectCode uint8 = 0x58
	// BreathRayLength 是 `31A6h` 的 `B0 0A`：2919h 的長度。
	BreathRayLength = 10
	// BreathSaveCategory 是 `319Ah`／`31B1h` 的 `B0 03`。
	BreathSaveCategory SaveCategory = 3
	// breathLastUse 是 `31BEh` 的 `80 7D 03 FD`：節點 +3 大於它才減一，否則摘掉。
	breathLastUse = 0xfd
	// breathSkipPercent 是 `30AAh` 的 `3C 32 / 77 03`：相位不是 0 時擲到 50 以下不吐。
	breathSkipPercent = 50
)

// BreathSkipped 是 `3098h..30AEh`：相位不是 0 時擲一顆 d100，50 以下這一回合不吐。
// roll 為 nil 代表相位是 0，不擲。
func BreathSkipped(phase uint8, roll func() int) bool {
	if phase == 0 {
		return false
	}
	return roll() <= breathSkipPercent
}

// BreathStart 是 `30FCh..3158h`：起點是自己往目標方向（每一軸取正負號）走一格，正方向
// 走兩格——吐息的是 2×2 的龍，身體佔 X..X+1。
func BreathStart(selfX, selfY, targetX, targetY int) (int, int) {
	step := func(self, target int) int {
		switch {
		case target > self:
			return self + 2
		case target < self:
			return self - 1
		}
		return self
	}
	return step(selfX, targetX), step(selfY, targetY)
}

// BreathRay 是 `31B8h` 推給 2919h 的四個值。
func BreathRay(maxHitPoints int) SpellRayEffect {
	return SpellRayEffect{Length: BreathRayLength, Damage: maxHitPoints, SaveCategory: BreathSaveCategory}
}

// BreathAfterUse 是 `31BEh..31DDh`：節點 +3 大於 FDh 就減一，否則摘掉。
func BreathAfterUse(list EffectList) EffectList {
	index, ok := list.IndexOf(BreathEffectCode)
	if !ok {
		return list
	}
	if list[index].Payload[effectNodeLevelOffset] > breathLastUse {
		list = append(EffectList(nil), list...)
		list[index].Payload[effectNodeLevelOffset]--
		return list
	}
	return list.RemoveAt(index)
}
