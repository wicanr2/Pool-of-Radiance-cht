package main

import (
	"fmt"
	"strings"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/combat"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// AI 施法（spec 096〈entry 4：AI 挑法術〉、spec 139 的 `2` Magic On／Off，issue #64）。
//
// 原版一隻怪（或交給電腦的隊員）的一個回合是 overlay-09 entry 1 那條「依序試，
// 誰先動就停」的鏈。和施法有關的是中間這三環，全在戰術模式擲完之後、追擊目標
// （overlay-13 `37B8h`）之前：
//
//	010Fh  entry 3（03E3h）用身上的東西：次數 = Roll(1, 7)，一定擲
//	012Eh  runtime +0（開始施法、還沒放出去的法術）非零 → overlay-22 entry 5 放出去
//	0169h  entry 2（0203h）轉變不死生物（spec 111）
//	0187h  entry 4（053Eh）挑法術：次數 = Roll(1, 7)，一定擲；挑到就交給
//	       overlay-13 entry 19（23F9h）
//
// 骰流裡每一隻怪每一回合那一對 `d7 d7`（spec 096〈骰流對回去之後〉）就是 entry 3
// 與 entry 4 的次數骰。
//
// 法術效果**不在這裡**：挑好法術與目標之後交給 cast.go 的 castSpell，跟玩家按 C
// 施法是同一支。這一檔只放原版 AI 那一層——挑哪一條、打誰、什麼時候放。

// 這一段訊息另開 `iota + 1000`（`+900` 是紮營那一段，text.go 那則註解說明過為什麼
// 不能接著排），在 init 登記進 messageKeys，重號就直接 panic。
const (
	msgStatusMagicOn messageID = iota + 1000
	msgStatusMagicOff
	msgFoeCasts
	msgFoeBeginsCasting
	msgFoeLostSpell
)

func init() {
	for id, key := range map[messageID]string{
		msgStatusMagicOn:    "ui.statusMagicOn",
		msgStatusMagicOff:   "ui.statusMagicOff",
		msgFoeCasts:         "ui.foeCasts",
		msgFoeBeginsCasting: "ui.foeBeginsCasting",
		msgFoeLostSpell:     "ui.foeLostSpell",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

const (
	// silenceEffectCode 是沉默術（效果碼 15h）：overlay-12 entry 22（`0725h`）在
	// `076Fh` 把 runtime `+1`「這一回合還能施法」清成 0（spec 112）。
	silenceEffectCode = 0x15
	// aiSpellTargetTries 是 overlay-13 `1ECCh` 的 14h：AI 挑施法目標最多試二十次。
	aiSpellTargetTries = 0x14
)

// foeCasting 是 AI 施法要跨行動記著的東西，都是原版 runtime 子結構或全域的欄位。
type foeCasting struct {
	// MagicOn 是 `DS:6D23h`：交給電腦的隊員（記錄 `+84h < 80h`）放不放法術。
	// 每場開打由 overlay-10 `1F3Bh` 清成 0，戰鬥中按 `2` 反相（overlay-08 `0490h`、
	// overlay-09 entry 7 `0FF4h`）。怪物不看它。全程式只有這四處碰它。
	MagicOn bool
	// Spells 是怪物那一格的法術陣列（記錄 `+17h..+2Bh`）。隊員不放這裡，
	// 直接讀角色的記憶陣列，施完才存得回去。
	Spells map[int][]uint8
	// Names 與 Levels 是怪物的名字與（牧師、法師）等級（記錄 `+96h`／`+9Bh`，
	// 施法者等級走 overlay-25 `26F8h`，spec 098）。
	Names  map[int]string
	Levels map[int][2]int
	// Pending 是 runtime `+0`：開始施法、還沒放出去的法術（overlay-13 `2519h`
	// 寫、overlay-09 `012Eh` 讀）。每回合開頭 overlay-13 entry 1 `0017h` 清成 0。
	Pending map[int]uint8
	// RoundHitPoints 是回合開頭的生命值，給 runtime `+1` 用：原版在受傷的那一刻
	// （overlay-13 `04E8h..054Bh`）把 `+1` 清成 0，還有開始施法的就「lost a spell」。
	// remake 的傷害分散在好幾支，這裡改成輪到牠時比一次生命值（見 castingDisrupted）。
	RoundHitPoints []int
}

// rememberSpellbook 在建 roster 時記下怪物的法術陣列、名字與施法等級。
func (state *tacticalState) rememberSpellbook(index int, record gamepack.MonsterRecord) {
	book := &state.Casting
	if book.Spells == nil {
		book.Spells, book.Names, book.Levels = map[int][]uint8{}, map[int]string{}, map[int][2]int{}
	}
	array := record.Raw[gamepack.AISpellArrayOffset : gamepack.AISpellArrayOffset+gamepack.AISpellArraySlots]
	book.Spells[index] = append([]uint8(nil), array...)
	book.Names[index] = strings.TrimSpace(record.Name)
	book.Levels[index] = [2]int{int(record.Raw[0x96]), int(record.Raw[0x9B])}
}

// startCastingRound 是回合開頭 overlay-13 entry 1 對施法那兩格做的事：runtime `+0`
// 清成 0（`0017h`）、`+1` 設成 1（`001Eh`）。`+1` 在 remake 由回合開頭的生命值代表。
func (state *tacticalState) startCastingRound() {
	state.Casting.Pending = nil
	state.Casting.RoundHitPoints = append(state.Casting.RoundHitPoints[:0], state.HitPoints...)
}

// castingDisrupted 說這一格這一回合是不是已經不能施法（runtime `+1` 為 0）。
// 三個來源：這一回合受過傷（overlay-13 `04F6h`）、身上有沉默（15h，overlay-12
// `076Fh`）或咳嗽（1Eh，臭雲，overlay-12 `0AC0h`）。
//
// 受傷用「比回合開頭的生命值低」判斷：同一回合先被治療再受傷、傷沒有低過開頭那一格
// 的，會被當成沒受傷（remake 的近似）。
func (state *tacticalState) castingDisrupted(index int) bool {
	if state.hasEffect(index, silenceEffectCode) || state.hasEffect(index, gamepack.StinkingCloudEffectCode) {
		return true
	}
	return state.woundedThisRound(index)
}

// woundedThisRound 說這一格這一回合受過傷。**開始施法的那一條只有受傷會打斷**：
// 沉默與咳嗽只清 `+1`，不動 `+0`，而 `012Eh` 放出去之前只看 `+0`。
func (state *tacticalState) woundedThisRound(index int) bool {
	if index < len(state.Casting.RoundHitPoints) && index < len(state.HitPoints) {
		return state.HitPoints[index] < state.Casting.RoundHitPoints[index]
	}
	return false
}

// toggleMagic 是戰鬥中的 `2`：`DS:6D23h` 反相，狀態列印 Magic On／Off（原版字串
// 在 overlay-08 `02E2h`／`02EBh`、overlay-09 `0FB5h`／`0FBEh`）。
func (a *app) toggleMagic(state *tacticalState) {
	state.Casting.MagicOn = !state.Casting.MagicOn
	if state.Casting.MagicOn {
		state.Status = state.say(msgStatusMagicOn)
	} else {
		state.Status = state.say(msgStatusMagicOff)
	}
	a.statusLine = state.Status
}

// foeSpellcaster 是 AI 施法者的來源：怪物讀 roster 建立時記下的陣列，隊員讀自己的
// 記憶陣列（spec 070：隊員的 `+1Fh` 那 13 格是 `+17h` 那 21 格的後段；前 8 格
// remake 的隊員一律是空的）。
type foeSpellcaster struct {
	array   []uint8
	name    string
	levels  [2]int
	party   int
	allowed bool
}

func (a *app) foeSpellcasterFor(state *tacticalState, mover uint8) (foeSpellcaster, bool) {
	index := int(mover)
	if index < len(state.PartySlot) && state.PartySlot[index] >= 0 {
		slot := state.PartySlot[index]
		if slot >= len(a.state.Party) {
			return foeSpellcaster{}, false
		}
		member := a.state.Party[slot]
		levels := memberClassLevels(member)
		return foeSpellcaster{
			array: member.Memorised,
			name:  member.Name,
			levels: [2]int{int(levels[gamepack.ClassSlotCleric]),
				int(levels[gamepack.ClassSlotMagicUser])},
			party: slot,
			// `05B5h`：記錄 `+84h <= 7Fh`（不是 NPC）的要 Magic On 才放。
			allowed: member.NPC || state.Casting.MagicOn,
		}, true
	}
	array, ok := state.Casting.Spells[index]
	if !ok {
		return foeSpellcaster{}, false
	}
	// 怪物記錄的 `+84h` 都是 80h 以上（八個怪物檔掃過：FFh 或 B2h），不看 Magic On。
	return foeSpellcaster{array: array, name: state.Casting.Names[index],
		levels: state.Casting.Levels[index], party: -1, allowed: true}, true
}

// foeCastPhase 是 overlay-09 entry 1 從 `010Fh` 到 `0187h` 那三環。回傳 true 代表
// 這一隻的行動已經用掉了（放了、開始施法，或挑到法術卻找不到目標）。
//
// 原版在它前面還有 entry 8 的士氣、後面是 entry 9 與接近迴圈；它們不歸這一支。
func (a *app) foeCastPhase(state *tacticalState, mover uint8, mode int) (bool, error) {
	// entry 3 的次數骰（`03FFh`），在任何閘門之前。怪物的物品鏈 remake 還沒有
	// （tacticalState.AttackRange 那則註解），隊員用物品也還沒接，所以擲完就走。
	a.rollDice(1, 7)

	index := int(mover)
	if spell := state.Casting.Pending[index]; spell != 0 {
		delete(state.Casting.Pending, index)
		if !state.woundedThisRound(index) {
			// `012Eh`：開始施法的那一條現在放出去，接著 entry 34 結束行動。
			state.setTacticMode(mover, mode)
			return true, a.foeReleaseSpell(state, mover, spell)
		}
		// 施法中受傷：原版在受傷那一刻印 "lost a spell"、把那一格從記憶裡清掉
		// （overlay-13 `0509h..0547h`），這一隻後面照常走 entry 2／4 與接近迴圈。
		if caster, ok := a.foeSpellcasterFor(state, mover); ok {
			a.foeForgetSpell(state, caster, spell)
		}
		state.FoeLog = state.say(msgFoeLostSpell, mover)
		state.Status = state.FoeLog
	}

	spell, err := a.foeChooseSpell(state, mover)
	if err != nil || spell == 0 {
		return false, err
	}
	params := a.spellParameters[spell]
	if params.CampOnly() {
		// overlay-13 `2450h`：印 "Camp Only Spell"、不施，也不算行動（`2496h` 直接
		// 返回 0）。AI 挑不到它：這幾格的優先度（`+0Dh`）都是 0。
		return false, nil
	}
	state.setTacticMode(mover, mode)
	cost := params.CastingCost()
	if cost == 0 {
		return true, a.foeReleaseSpell(state, mover, spell)
	}
	// `24E7h..2552h`：印 "Begins Casting"、runtime +0 記下這一條、先攻分數扣掉施法
	// 時間（不夠扣就留 1）。**不呼叫 entry 34**，所以這一隻還在排序裡，輪到下一次
	// 才由 `012Eh` 放出去。
	if state.Casting.Pending == nil {
		state.Casting.Pending = map[int]uint8{}
	}
	state.Casting.Pending[index] = spell
	if index < len(state.Scores) {
		score, err := combat.ApplyCastingTimeInitiative(state.Scores[index], cost)
		if err != nil {
			return true, err
		}
		state.Scores[index] = score
	}
	state.FoeLog = state.say(msgFoeBeginsCasting, mover, a.spellLabel(spell))
	state.Status = state.FoeLog
	state.Moving = false
	state.selectActor(a.rollDice)
	if state.Mover == 0 {
		state.endRound(a.rollDice)
	}
	return true, nil
}

// foeChooseSpell 是 overlay-09 entry 4 的前半：收清單、擲次數、逐輪降門檻挑。
func (a *app) foeChooseSpell(state *tacticalState, mover uint8) (uint8, error) {
	caster, ok := a.foeSpellcasterFor(state, mover)
	var list []uint8
	// `0548h`：runtime +1 為 0（這一回合不能施法）就不收清單——次數骰照樣擲。
	if ok && !state.castingDisrupted(int(mover)) {
		list = gamepack.AISpellList(caster.array)
	}
	side, sideOK := state.sideOf(mover)
	counts := state.sideCounts()
	opposing := counts.Foes
	if side == 1 {
		opposing = counts.Party
	}
	allowed := ok && sideOK && caster.allowed && opposing > 0
	var failure error
	spell := gamepack.ChooseAISpell(list, allowed, a.rollDice, func(id, threshold uint8) bool {
		if failure != nil {
			return false
		}
		accepted, err := a.foeAcceptSpell(state, mover, caster, id, threshold)
		if err != nil {
			failure = err
		}
		return accepted
	})
	return spell, failure
}

// foeAcceptSpell 是 overlay-09 `02EAh`：這一條現在值不值得放。
//
//	02F4  優先度（+0Dh）< 門檻               → 否
//	030B  +0Eh 為 0 而且不是治療輕傷          → 是（打自己，不必找人）
//	0323  治療輕傷：1BF0h 找得到受傷的自己人  → 是
//	0344  n = 射程內的敵人（010Ah:00C0h）    ; 射程是 overlay-22 entry 3
//	035C  n == 0                               → 否
//	036D  +0Fh 為 0（不是範圍法術）           → 是
//	037A  每一個敵人的範圍（+0Fh）裡有自己人   → 否（`0255h`）
//	03D6                                      → 是
//
// remake 另外擋掉兩種：處理常式還沒讀的（`SpellCaster.Implemented`，施不出來），與
// 第 7 位立著的格子（還沒記完）。後者原版照樣拿它查表——`31A1h + 編號 × 16` 會讀到
// 表外，那一格讀出什麼沒有追。
func (a *app) foeAcceptSpell(state *tacticalState, mover uint8, caster foeSpellcaster,
	id, threshold uint8) (bool, error) {
	if int(id) >= len(a.spellParameters) || id == 0 || a.spellCaster == nil ||
		!a.spellCaster.Implemented(id) {
		return false, nil
	}
	params := a.spellParameters[id]
	if params.AIPriority() < threshold {
		return false, nil
	}
	if id == gamepack.AISpellCureLightWounds {
		if _, ok, err := state.cureLightWoundsTarget(mover); err != nil || ok {
			return ok, err
		}
	} else if params.TargetsCaster() {
		return true, nil
	}
	targets, err := state.opposingWithin(mover, params.Range(foeCasterLevel(params, caster)), false)
	if err != nil || len(targets) == 0 {
		return false, err
	}
	if params.AreaBudget() == 0 {
		return true, nil
	}
	for _, target := range targets {
		crowded, err := state.casterAlliesNear(mover, target, params.AreaBudget())
		if err != nil || crowded {
			return false, err
		}
	}
	return true, nil
}

// foeCasterLevel 是 overlay-25 `26F8h`（spec 098）：神術讀牧師等級、巫術讀法師等級。
func foeCasterLevel(params gamepack.SpellParameters, caster foeSpellcaster) int {
	return gamepack.CasterLevelFor(params, caster.levels[0], caster.levels[1], false)
}

// foeReleaseSpell 是 overlay-22 entry 5（`0C14h`）的 AI 那一側：挑目標（`DS:6A78h`
// 在戰鬥中指到 overlay-13 entry 18 `20AEh`），挑不到就不施（記憶也不動），挑到了
// 就把那一格從記憶裡清掉（overlay-25 entry 16 `14ECh`）再派發效果。兩種結果都用掉
// 這個行動（overlay-13 `24DAh` 的 entry 34）。
func (a *app) foeReleaseSpell(state *tacticalState, mover uint8, spell uint8) error {
	caster, ok := a.foeSpellcasterFor(state, mover)
	if !ok {
		state.endTurn(a.rollDice, false)
		return nil
	}
	label := a.spellLabel(spell)
	target, chosen, found, err := a.foeSpellTargets(state, mover, spell, caster)
	if err != nil {
		return err
	}
	if !found {
		a.tacticalStatus(state, fmt.Sprintf(a.text(msgCastNoTarget), label))
		state.FoeLog = state.Status
		state.endTurnAfterAction(a.rollDice)
		return nil
	}
	state.FoeLog = state.say(msgFoeCasts, mover, label)
	casting := spellCasting{
		name:      caster.name,
		partySlot: caster.party,
		level:     foeCasterLevel(a.spellParameters[spell], caster),
		consume:   func() { a.foeForgetSpell(state, caster, spell) },
	}
	if caster.party >= 0 {
		casting.member = &a.state.Party[caster.party]
	}
	return a.castSpell(state, casting, castOption{Slot: -1, ID: spell, Label: label}, target, chosen)
}

// foeForgetSpell 是 overlay-25 entry 16（`14ECh`）：在陣列裡清掉**第一個**等於這個
// 編號的格子。比的是整個 byte。
func (a *app) foeForgetSpell(state *tacticalState, caster foeSpellcaster, spell uint8) {
	for index, value := range caster.array {
		if value != spell {
			continue
		}
		caster.array[index] = 0
		break
	}
	if caster.party >= 0 && caster.party < len(a.state.Party) {
		syncTrainedLibraryCharacter(&a.state, a.state.Party[caster.party])
	}
}

// foeSpellTargets 是 overlay-13 entry 18（`20AEh`）在 AI 那一側怎麼挑目標，依參數表
// `+6` 的低四位（spec 074）：
//
//	0         打施法者自己，不挑（`20FDh`）
//	0Fh       `1E09h` 挑一個（`211Ah`）
//	8..0Eh    `1E09h` 挑一個，再以它為中心收範圍（`220Fh`）
//	其餘      `1E09h` 挑 (模式 & 3) + 1 個，重複的不算（`22BEh..23BEh`）
//
// castSpell 一次只作用在一個目標上（玩家那一側也是），所以多挑的那幾個照樣擲骰
// 但只交第一個出去。chosen 為 false 代表「打自己、不必指定」。
func (a *app) foeSpellTargets(state *tacticalState, mover, spell uint8,
	caster foeSpellcaster) (target uint8, chosen, found bool, err error) {
	params := a.spellParameters[spell]
	mode := params.TargetMode()
	if mode == gamepack.SpellTargetSelf {
		return 0, false, true, nil
	}
	count := 1
	if mode < 8 {
		count = int(mode&3) + 1
	}
	var picked []uint8
	for ; count > 0; count-- {
		pick, ok, err := a.foeSpellTarget(state, mover, spell, caster)
		if err != nil {
			return 0, false, false, err
		}
		if !ok {
			continue
		}
		duplicate := false
		for _, earlier := range picked {
			duplicate = duplicate || earlier == pick
		}
		if !duplicate {
			picked = append(picked, pick)
		}
	}
	if len(picked) == 0 {
		return 0, false, false, nil
	}
	return picked[0], true, true, nil
}

// foeSpellTarget 是 overlay-13 `1E09h` 的 AI 那一支（旗標 `[bp+0Ch]` 非 0）：
//
//	1E67  +0Eh 為 0：目標是施法者自己；治療輕傷改走 1BF0h，找不到就失敗
//	1ECC  否則最多試二十次：
//	1EFF    37B8h(施法者, 射程, 0, 強制重挑)   ; 從射程內的敵人擲骰挑一個
//	1F29    +0Fh 非 0：以它為中心的範圍裡有自己人 → 不要
//	1FC5    它身上已經有 33h／34h／35h／1Fh 之一，而這條法術掛的也是其中之一 → 不要
//	2005    要了就收工
func (a *app) foeSpellTarget(state *tacticalState, mover, spell uint8,
	caster foeSpellcaster) (uint8, bool, error) {
	params := a.spellParameters[spell]
	if params.TargetsCaster() {
		if spell == gamepack.AISpellCureLightWounds {
			return state.cureLightWoundsTarget(mover)
		}
		return mover, true, nil
	}
	budget := params.Range(foeCasterLevel(params, caster))
	for tries := aiSpellTargetTries; tries > 0; tries-- {
		target, ok, err := a.foeRetarget(state, mover, budget)
		if err != nil {
			return 0, false, err
		}
		if !ok {
			continue
		}
		if params.AreaBudget() > 0 {
			crowded, err := state.casterAlliesNear(mover, target, params.AreaBudget())
			if err != nil {
				return 0, false, err
			}
			if crowded {
				continue
			}
		}
		if gamepack.IsAIDisablingEffect(params.EffectCode()) && state.aiDisabled(target) {
			continue
		}
		return target, true, nil
	}
	return 0, false, nil
}

// foeRetarget 是 overlay-13 `37B8h` 帶「強制重挑」：舊目標不看，從射程內的敵人
// （`010Ah:00C0h(記錄, 射程)`，依直線追蹤成本排）擲 Roll(1, n) 挑一個，寫進 runtime
// `+0Ah`——所以放完法術之後，這一隻追的就是法術打的那一個。劃掉、二十次與兩輪制
// 與追擊共用 `rollFoeTarget`（含 `1087h`，#65）。
func (a *app) foeRetarget(state *tacticalState, mover uint8, budget int) (uint8, bool, error) {
	target, err := a.rollFoeTarget(state, mover, func(relaxed bool) ([]uint8, error) {
		return state.opposingWithin(mover, budget, relaxed)
	})
	if err != nil || target == 0 {
		return 0, false, err
	}
	state.setFoeTarget(mover, target)
	return target, true, nil
}

// opposingWithin 是 overlay-25 entry 32（`246Dh`）：以施法者的位置與體型，在 budget
// 之內找對面的人（spec 096）。relaxed 是 `DS:6674h` 的 +6（直線追蹤跳過地形）。
func (state *tacticalState) opposingWithin(mover uint8, budget int, relaxed bool) ([]uint8, error) {
	side, ok := state.sideOf(mover)
	if !ok {
		return nil, nil
	}
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return nil, err
	}
	snapshot.Map.IgnoreTerrain = relaxed
	if budget < 0 {
		budget = 0
	}
	here := state.Roster[mover]
	return combat.OpposingNearbyAt(snapshot, mover, here.X, here.Y, uint16(budget), 1-side, state.sideOf)
}

// casterAlliesNear 是 overlay-09 `0255h`（與 overlay-13 `1F33h..1FC3h` 同一個形狀）：
// 以 target 的位置為中心、體型 1、預算是範圍法術的 `+0Fh`，跑一次 `0912h`，
// 收到的人裡有沒有跟施法者同一邊的（施法者自己也算）。
func (state *tacticalState) casterAlliesNear(mover, target uint8, budget int) (bool, error) {
	side, ok := state.sideOf(mover)
	if !ok || int(target) >= len(state.Roster) {
		return false, nil
	}
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return false, err
	}
	centre := state.Roster[target]
	cells, err := combat.NearbyCells(combat.NearbyRequest{
		Map: snapshot.Map, Classes: snapshot.Classes, Cells: snapshot.Cells,
		Class: 1, Facing: combat.DirectionUnset, Budget: uint16(budget),
		BaseX: centre.X, BaseY: centre.Y,
	})
	if err != nil {
		return false, err
	}
	for _, cell := range cells {
		if other, ok := state.sideOf(cell.CombatantIndex); ok && other == side {
			return true, nil
		}
	}
	return false, nil
}

// aiDisabled 是 overlay-25 entry 6（`0B79h`）：身上有 AIDisablingEffects 之一。
func (state *tacticalState) aiDisabled(index uint8) bool {
	for _, code := range gamepack.AIDisablingEffects {
		if state.hasEffect(int(index), code) {
			return true
		}
	}
	return false
}

// cureLightWoundsTarget 是 overlay-13 entry 17（`1BF0h`）：看身邊八格加腳下那一格
// （方向 0..8，方向表 `DS:274Ah`／`2753h`，第 9 項是原地），站著的是自己人而且沒
// 滿血，就比目前挑到的更少血才換（`1CBFh`）；輪到自己那一格時，只要低於上限的一半
// 也換（`1CC4h..1CF9h`）。
//
// 沒讀完的一支：腳下沒人、地形是 1Fh 的格子（`1D1Ah..1DA6h`）走的是 `DS:6634h` 那張
// 每筆 7 bytes 的表，看起來是倒在地上的人；remake 沒有那張表，這一支不做。
func (state *tacticalState) cureLightWoundsTarget(caster uint8) (uint8, bool, error) {
	snapshot, err := state.tacticalSnapshot()
	if err != nil {
		return 0, false, err
	}
	here := state.Roster[caster]
	best, bestHitPoints := uint8(0), 0xFF
	for direction := uint8(0); direction <= combat.DirectionAny; direction++ {
		step, err := combat.DirectionStep(direction)
		if err != nil {
			return 0, false, err
		}
		occupant, _, err := snapshot.CellAt(here.X+uint8(step.X), here.Y+uint8(step.Y))
		if err != nil {
			return 0, false, err
		}
		if occupant == 0 || int(occupant) >= len(state.HitPoints) ||
			int(occupant) >= len(state.MaxHitPoints) {
			continue
		}
		if same, err := state.sameSide(occupant, caster); err != nil || !same {
			continue
		}
		hitPoints, maximum := state.HitPoints[occupant], state.MaxHitPoints[occupant]
		if hitPoints >= maximum {
			continue
		}
		if hitPoints < bestHitPoints || (occupant == caster && hitPoints < maximum/2) {
			best, bestHitPoints = occupant, hitPoints
		}
	}
	return best, best != 0, nil
}

// spellLabel 是法術的名字（與施法清單同一個來源；那份目錄平常在打開法術頁時
// 才建，這裡沒有就先建一份，不打開頁面）。
func (a *app) spellLabel(id uint8) string {
	if a.spells == nil {
		if catalogue, err := gamepack.TraditionalChineseSpells(); err == nil {
			a.spells = &spellState{catalogue: catalogue}
		}
	}
	if a.spells != nil {
		if spell, err := a.spells.catalogue.SpellByID(id); err == nil {
			return spell.Text
		}
	}
	return fmt.Sprintf("%d", id)
}
